package network

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/portcollection"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/portwrite"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/utils"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/utils/operlog"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
	"gorm.io/gorm"
)

// ModulePortWrite 端口写操作 operlog 模块名常量 (Phase 52 AUDIT-01 锁定)
//
// 注意 (D-07)：与父菜单名 "端口状态" 解耦 — module 仅作 sys_oper_log.title 显示串,
// 沿用 ROADMAP 历史用名 "端口管理"（不是 "端口状态" 也不是 "端口写"）。
const ModulePortWrite = "端口管理"

// PortWriteHandler 端口写操作 handler (Phase 52 W3)
//
// 6 个写端点的业务编排（5 单端口 + 1 batch）。每个写操作严格遵循 D-A4-04 顺序：
//
//	D-02 预 SELECT before_value → service → audit INSERT → operlog.Record → response.Success
//
// sentinel error（入口校验失败）路径不写 audit / 不调 operlog (by-design：
// 无 SSH 流量可追溯)。PortResult.Status="failed"（SSH 执行失败但 nil sentinel err）
// 路径走 200 + audit 行 status=failed (D-04 + RESEARCH §3.3)。
type PortWriteHandler struct {
	service portwrite.PortWriteService
	core    *core.Core
}

// NewPortWriteHandler 构造函数。router 通过 WithCore 注入 core 后使用。
func NewPortWriteHandler(svc portwrite.PortWriteService) *PortWriteHandler {
	return &PortWriteHandler{service: svc}
}

// WithCore 注入 core 引用 (复制 fix_suggestion_handler.go 模式)。
func (h *PortWriteHandler) WithCore(core *core.Core) *PortWriteHandler {
	if h != nil {
		h.core = core
	}
	return h
}

// PortWriteRequest 单端口写请求通用 struct。
//
//   - PortID：必填，目标端口 UUID
//   - Description：仅 description action 使用（service 入口校验长度 ≤ 80）
//   - Reason：UI-02 操作原因，后端仅记录不校验
type PortWriteRequest struct {
	PortID      string `json:"portId" binding:"required"`
	Description string `json:"description,omitempty"`
	Reason      string `json:"reason,omitempty"`
}

// ====================== 单端口 handler ======================

// Shutdown 关闭端口 (POST /network/ports/write/shutdown)
func (h *PortWriteHandler) Shutdown(c *gin.Context) {
	h.execSinglePort(c, portcollection.ActionShutdown, operlog.OperTypeStatus,
		func(ctx context.Context, portID, operator, desc string) (*portwrite.PortResult, error) {
			return h.service.Shutdown(ctx, portID, operator)
		})
}

// UndoShutdown 取消关闭 (POST /network/ports/write/undo-shutdown)
func (h *PortWriteHandler) UndoShutdown(c *gin.Context) {
	h.execSinglePort(c, portcollection.ActionUndoShutdown, operlog.OperTypeStatus,
		func(ctx context.Context, portID, operator, desc string) (*portwrite.PortResult, error) {
			return h.service.UndoShutdown(ctx, portID, operator)
		})
}

// SetDescription 设置端口描述 (POST /network/ports/write/description)
func (h *PortWriteHandler) SetDescription(c *gin.Context) {
	h.execSinglePort(c, portcollection.ActionDescription, operlog.OperTypeUpdate,
		func(ctx context.Context, portID, operator, desc string) (*portwrite.PortResult, error) {
			return h.service.SetDescription(ctx, portID, desc, operator)
		})
}

// EnableDot1x 启用 802.1X (POST /network/ports/write/dot1x-enable)
func (h *PortWriteHandler) EnableDot1x(c *gin.Context) {
	h.execSinglePort(c, portcollection.ActionDot1xEnable, operlog.OperTypeStatus,
		func(ctx context.Context, portID, operator, desc string) (*portwrite.PortResult, error) {
			return h.service.EnableDot1x(ctx, portID, operator)
		})
}

// DisableDot1x 停用 802.1X (POST /network/ports/write/dot1x-disable)
func (h *PortWriteHandler) DisableDot1x(c *gin.Context) {
	h.execSinglePort(c, portcollection.ActionDot1xDisable, operlog.OperTypeStatus,
		func(ctx context.Context, portID, operator, desc string) (*portwrite.PortResult, error) {
			return h.service.DisableDot1x(ctx, portID, operator)
		})
}

// execSinglePort 单端口 handler 公共流程 (5 个方法共用)。
//
// 步骤 (PATTERNS.md §4c + RESEARCH §3.3 sentinel→HTTP 表)：
//  1. c.ShouldBindJSON → 400 if fail
//  2. operator := utils.GetUsername(c)
//  3. D-02 预 SELECT DevicePortStatus → 拼 before_value JSON（端口行不存在 → {}）
//  4. 调 service 方法 (serviceCall) → sentinel err 用 errors.Is 翻译成 4xx/500，**不写 audit**
//  5. PortResult.Status=="failed" 路径继续走 audit + operlog + response.Success (200)
//  6. D-04 写 audit 行 (after_value per D-03, device_response per A5)
//  7. Path C：operlog.Record(..., WithOperParam({audit_ids:[auditID], ...}))
//  8. response.Success(c, result)
//
// serviceCall 接收 description 参数（仅 SetDescription handler 用；其他 handler 忽略）。
func (h *PortWriteHandler) execSinglePort(
	c *gin.Context,
	action portcollection.PortAction,
	operType int,
	serviceCall func(ctx context.Context, portID, operator, desc string) (*portwrite.PortResult, error),
) {
	var req PortWriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	operator := utils.GetUsername(c)

	// D-02 预 SELECT before_value（端口行不存在 → {}, 不阻塞流程）
	var port models.DevicePortStatus
	beforeValue := json.RawMessage([]byte("{}"))
	deviceID := ""
	portFindErr := h.core.GetDB().Where("id = ?", req.PortID).First(&port).Error
	if portFindErr == nil {
		deviceID = port.DeviceID
		beforeValue = buildBeforeValue(&port)
	}

	result, err := serviceCall(c.Request.Context(), req.PortID, operator, req.Description)
	if err != nil {
		// sentinel 路径：4xx/500，不写 audit、不调 operlog（RESEARCH §3.3 + acceptance）
		switch {
		case errors.Is(err, portwrite.ErrPortNotFound):
			response.Error(c, http.StatusNotFound, "端口不存在")
		case errors.Is(err, portwrite.ErrDeviceNotFound):
			response.Error(c, http.StatusNotFound, "设备不存在")
		default:
			response.Error(c, http.StatusInternalServerError, err.Error())
		}
		return
	}

	// 关键分支：result.Status=="failed" 不走 response.Error，继续 audit + operlog + 200
	// (RESEARCH §3.3, PATTERNS.md §4d 末行)

	// D-04 写 audit 行（Path C 第 1 步：先 INSERT 拿 audit_id）
	auditRow := buildAuditRow(result, beforeValue, deviceID, operator)
	if createErr := h.core.GetDB().Create(auditRow).Error; createErr != nil {
		// audit 写入失败不阻塞响应（与 batch 路径一致），但记录告警。
		log.Printf("port_write audit insert failed portID=%s action=%s: %v", req.PortID, action, createErr)
	}

	// Path C 第 2 步：把 audit_id 嵌 operlog oper_param（audit.oper_log_id 列保持 NULL）
	operParam := buildSinglePortOperParam(auditRow.ID, deviceID, req.PortID, string(action), operator, result.Status)
	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), ModulePortWrite, operType,
		operlog.WithOperParam(operParam))

	response.Success(c, result)
}

// ====================== Batch handler ======================

// BatchWrite 批量同设备多端口同操作 (POST /network/ports/write/batch)
//
// 复用 Phase 51 BatchWriteRequest struct（直接绑定）。Phase 51 service 已锁：
//   - maxBatchSize = 50（超 → ErrBatchTooLarge → 400）
//   - ErrEmptyBatch → 400
//   - ErrMixedDevices → 400（跨设备拒绝）
//   - serial fail-fast：transport/device_rejected 立即终止后续端口
//
// 成功路径：遍历 Succeeded + Failed + Skipped 三切片，每端口 1 条 audit 行（共 N 条），
// 收集 auditIDs 列表 → operlog.Record(OperTypeBatch, WithOperParam(batchSummaryJSON))
// → response.Success(result)。
func (h *PortWriteHandler) BatchWrite(c *gin.Context) {
	var req portwrite.BatchWriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	operator := utils.GetUsername(c)

	result, err := h.service.BatchWritePorts(c.Request.Context(), req, operator)
	if err != nil {
		// sentinel 路径：不写 audit、不调 operlog
		switch {
		case errors.Is(err, portwrite.ErrBatchTooLarge):
			response.Error(c, http.StatusBadRequest, "批量端口数超过上限 50")
		case errors.Is(err, portwrite.ErrEmptyBatch):
			response.Error(c, http.StatusBadRequest, "批量端口列表为空")
		case errors.Is(err, portwrite.ErrMixedDevices):
			response.Error(c, http.StatusBadRequest, "批量端口必须属于同一设备")
		case errors.Is(err, portwrite.ErrPortNotFound):
			response.Error(c, http.StatusNotFound, "端口不存在")
		case errors.Is(err, portwrite.ErrDeviceNotFound):
			response.Error(c, http.StatusNotFound, "设备不存在")
		default:
			response.Error(c, http.StatusInternalServerError, err.Error())
		}
		return
	}

	// 收集所有 PortResult（Succeeded + Failed + Skipped）按 D-05 写 N 条 audit
	all := append(append(result.Succeeded, result.Failed...), result.Skipped...)

	// 预 SELECT before_value 一次取齐（按 portID 索引），避免 N+1 查询
	beforeMap := preloadBeforeValues(h.core.GetDB(), all)

	auditIDs := make([]string, 0, len(all))
	for _, pr := range all {
		beforeValue := beforeMap[pr.PortID]
		if len(beforeValue) == 0 {
			beforeValue = json.RawMessage([]byte("{}"))
		}
		auditRow := buildAuditRow(&pr, beforeValue, req.DeviceID, operator)
		if createErr := h.core.GetDB().Create(auditRow).Error; createErr != nil {
			log.Printf("port_write audit insert failed portID=%s action=%s: %v", pr.PortID, req.Action, createErr)
			// 不阻塞响应，继续写下一条（CONTEXT Claude's Discretion line 173）
			continue
		}
		auditIDs = append(auditIDs, auditRow.ID)
	}

	// Path C：汇总 operlog 行 oper_param 含 audit_ids 完整列表
	batchSummary := buildBatchSummaryOperParam(
		string(req.Action),
		len(req.PortIDs),
		result,
		req.DeviceID,
		auditIDs,
	)
	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), ModulePortWrite, operlog.OperTypeBatch,
		operlog.WithOperParam(batchSummary))

	response.Success(c, result)
}

// ====================== Helpers ======================

// buildBeforeValue 拼接 D-02 before_value JSON 快照
//
// 字段结构固定：{"admin_status":..., "dot1x_enabled":..., "description":...}
// （便于审计完整快照，description action 之外的 action 也带 description）
func buildBeforeValue(port *models.DevicePortStatus) json.RawMessage {
	b, _ := json.Marshal(map[string]interface{}{
		"admin_status":   port.AdminStatus,
		"dot1x_enabled":  port.Dot1xEnabled,
		"description":    port.Description,
		"interface_name": port.InterfaceName,
	})
	return b
}

// buildAfterValue 拼接 D-03 after_value 目标态快照（按 action 分支）
//
// shutdown → {"admin_status":"down"}
// undo_shutdown → {"admin_status":"up"}
// dot1x_enable → {"dot1x_enabled":true}
// dot1x_disable → {"dot1x_enabled":false}
// description → {"description":"<desc>"}
//
// skipped（NoOp）路径：caller 会在 buildAuditRow 里覆盖为 before_value。
// buildAfterValue 不接受 description 参数 — description action 的目标态在 buildAuditRow
// 里用 PortResult.CommandSent 反推不出来（CommandSent 是命令串，非目标描述），
// 所以 description 路径的目标态 JSON 由 caller (buildAuditRow) 从 PortResult.CurrentState
// 或 req.Description 获取 — 本函数仅按 action 返一个最小态，description action 时
// caller 通过 buildAuditRowForDescription 覆盖。
func buildAfterValue(action portcollection.PortAction) json.RawMessage {
	switch action {
	case portcollection.ActionShutdown:
		return json.RawMessage([]byte(`{"admin_status":"down"}`))
	case portcollection.ActionUndoShutdown:
		return json.RawMessage([]byte(`{"admin_status":"up"}`))
	case portcollection.ActionDot1xEnable:
		return json.RawMessage([]byte(`{"dot1x_enabled":true}`))
	case portcollection.ActionDot1xDisable:
		return json.RawMessage([]byte(`{"dot1x_enabled":false}`))
	default:
		// description：target 态在 caller 处用具体值覆盖（这里给空 placeholder）
		return json.RawMessage([]byte(`{}`))
	}
}

// buildAuditRow 按 PATTERNS.md §4e + RESEARCH §3.4 映射 PortResult → PortWriteAudit
//
// after_value 填值（D-03）：
//   - skipped (NoOp) → after_value = before_value（无变化）
//   - description action → after_value = {"description": result.CurrentState or ""}
//     （注：PortResult.CurrentState 由 Phase 51 service 在 description 路径填目标态描述）
//   - 其他 action → buildAfterValue(action)
//
// device_response 填值（RESEARCH A5，Phase 51 service 不暴露原始响应文本）：
//   - succeeded → "OK"
//   - failed → result.Error
//   - skipped → "无需操作"
//
// failure_reason：failed → &result.Error，其他 → nil
//
// OperLogID：nil（Path C 强制，audit.oper_log_id 列保持 NULL）
func buildAuditRow(pr *portwrite.PortResult, beforeValue json.RawMessage, deviceID, operator string) *models.PortWriteAudit {
	afterValue := buildAfterValue(pr.Action)
	if pr.Status == "skipped" {
		afterValue = beforeValue
	}
	if pr.Action == portcollection.ActionDescription {
		// description 路径用 PortResult.CurrentState 作为目标态（Phase 51 service 在
		// description 成功路径会把新描述填入 CurrentState；fallback 空对象）。
		desc := map[string]string{"description": pr.CurrentState}
		if b, err := json.Marshal(desc); err == nil {
			afterValue = b
		}
		if pr.Status == "skipped" {
			afterValue = beforeValue
		}
	}

	deviceResponse := "OK"
	var failureReason *string
	switch pr.Status {
	case "failed":
		deviceResponse = pr.Error
		s := pr.Error
		failureReason = &s
	case "skipped":
		deviceResponse = "无需操作"
	}

	return &models.PortWriteAudit{
		DeviceID:       deviceID,
		PortID:         pr.PortID,
		Action:         string(pr.Action),
		BeforeValue:    beforeValue,
		AfterValue:     afterValue,
		CommandSent:    pr.CommandSent,
		DeviceResponse: deviceResponse,
		Status:         pr.Status,
		FailureReason:  failureReason,
		Operator:       operator,
		// OperLogID: nil（Path C 强制，audit.oper_log_id 列保持 NULL）
	}
}

// buildSinglePortOperParam 拼接单端口 operlog.oper_param JSON (AUDIT-02 + Path C)
//
// 字段：{audit_ids:[auditID], device_id, port_id, action, operator, result_status}
func buildSinglePortOperParam(auditID, deviceID, portID, action, operator, resultStatus string) string {
	b, _ := json.Marshal(map[string]interface{}{
		"audit_ids":     []string{auditID},
		"device_id":     deviceID,
		"port_id":       portID,
		"action":        action,
		"operator":      operator,
		"result_status": resultStatus,
	})
	return string(b)
}

// buildBatchSummaryOperParam 拼接 batch operlog.oper_param JSON (CONV-04)
//
// 字段：{action, batch_size, succeeded_count, failed_count, skipped_count, device_id, audit_ids:[...]}
func buildBatchSummaryOperParam(action string, batchSize int, result *portwrite.BatchResult, deviceID string, auditIDs []string) string {
	b, _ := json.Marshal(map[string]interface{}{
		"action":          action,
		"batch_size":      batchSize,
		"succeeded_count": len(result.Succeeded),
		"failed_count":    len(result.Failed),
		"skipped_count":   len(result.Skipped),
		"device_id":       deviceID,
		"audit_ids":       auditIDs,
	})
	return string(b)
}

// preloadBeforeValues 一次 SELECT 取齐 batch 中所有端口的 before_value（按 portID 索引）
//
// 端口行不存在时不出现在 map 中（caller fallback 到 {}）。
func preloadBeforeValues(db *gorm.DB, results []portwrite.PortResult) map[string]json.RawMessage {
	out := make(map[string]json.RawMessage, len(results))
	if len(results) == 0 {
		return out
	}
	portIDs := make([]string, 0, len(results))
	for _, r := range results {
		portIDs = append(portIDs, r.PortID)
	}
	var ports []models.DevicePortStatus
	if err := db.Where("id IN ?", portIDs).Find(&ports).Error; err != nil {
		log.Printf("port_write preloadBeforeValues failed: %v", err)
		return out
	}
	for i := range ports {
		out[ports[i].ID] = buildBeforeValue(&ports[i])
	}
	return out
}
