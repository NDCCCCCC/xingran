package network

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/portcollection"
	"github.com/xingran-next/xingran-go-backend/internal/services/portwrite"
	"github.com/xingran-next/xingran-go-backend/internal/utils"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
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

// SetAccessVlanRequest 修改端口 access VLAN 请求 (v1.20.1)。
//
//   - PortID：必填，目标端口 UUID
//   - VLANID：必填，1-4094 (binding 双重防线；service 仍校验为真相源)
//   - Reason：UI-02 操作原因，后端仅记录不校验
type SetAccessVlanRequest struct {
	PortID string `json:"portId" binding:"required"`
	VLANID int    `json:"vlanId" binding:"required,min=1,max=4094"`
	Reason string `json:"reason,omitempty"`
}

// PortBindingRequest 端口绑定请求 (v1.20.1)。
//
//   - PortID：必填，目标端口 UUID
//   - Op：必填，"add" 或 "remove" (oneof tag 双重防线；service 仍校验)
//   - IPAddress：必填，IPv4 地址 (service 层正则校验为真相源)
//   - MACAddress：可选，MAC 地址 (service 层 NormalizeMACAddress 校验，仅 op=add 时有意义)
//   - Reason：UI-02 操作原因，后端仅记录不校验
type PortBindingRequest struct {
	PortID     string `json:"portId" binding:"required"`
	Op         string `json:"op" binding:"required,oneof=add remove"`
	IPAddress  string `json:"ipAddress" binding:"required"`
	MACAddress string `json:"macAddress,omitempty"`
	Reason     string `json:"reason,omitempty"`
}

// ====================== 单端口 handler ======================

// Shutdown 关闭端口 (POST /network/ports/write/shutdown)
func (h *PortWriteHandler) Shutdown(c *gin.Context) {
	var req PortWriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}
	h.execSinglePort(c, portcollection.ActionShutdown, operlog.OperTypeDisable, req.PortID, req.Description,
		func(ctx context.Context, portID, operator, desc string) (*portwrite.PortResult, error) {
			return h.service.Shutdown(ctx, portID, operator)
		})
}

// UndoShutdown 取消关闭 (POST /network/ports/write/undo-shutdown)
func (h *PortWriteHandler) UndoShutdown(c *gin.Context) {
	var req PortWriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}
	h.execSinglePort(c, portcollection.ActionUndoShutdown, operlog.OperTypeEnable, req.PortID, req.Description,
		func(ctx context.Context, portID, operator, desc string) (*portwrite.PortResult, error) {
			return h.service.UndoShutdown(ctx, portID, operator)
		})
}

// SetDescription 设置端口描述 (POST /network/ports/write/description)
func (h *PortWriteHandler) SetDescription(c *gin.Context) {
	var req PortWriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}
	h.execSinglePort(c, portcollection.ActionDescription, operlog.OperTypeUpdate, req.PortID, req.Description,
		func(ctx context.Context, portID, operator, desc string) (*portwrite.PortResult, error) {
			return h.service.SetDescription(ctx, portID, desc, operator)
		})
}

// EnableDot1x 启用 802.1X (POST /network/ports/write/dot1x-enable)
func (h *PortWriteHandler) EnableDot1x(c *gin.Context) {
	var req PortWriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}
	h.execSinglePort(c, portcollection.ActionDot1xEnable, operlog.OperTypeEnable, req.PortID, req.Description,
		func(ctx context.Context, portID, operator, desc string) (*portwrite.PortResult, error) {
			return h.service.EnableDot1x(ctx, portID, operator)
		})
}

// DisableDot1x 停用 802.1X (POST /network/ports/write/dot1x-disable)
func (h *PortWriteHandler) DisableDot1x(c *gin.Context) {
	var req PortWriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}
	h.execSinglePort(c, portcollection.ActionDot1xDisable, operlog.OperTypeDisable, req.PortID, req.Description,
		func(ctx context.Context, portID, operator, desc string) (*portwrite.PortResult, error) {
			return h.service.DisableDot1x(ctx, portID, operator)
		})
}

// SetAccessVlan 修改端口 access VLAN (POST /network/ports/write/set-access-vlan)。
//
// OperType=Update(=2) per design.md §6：修改现有端口配置（与 description 同类）。
//
// 绑定 SetAccessVlanRequest 在 execSinglePort 之外（PATTERNS.md Option A），
// 校验 vlanId 1-4094 (binding 双重防线，service 仍校验为真相源)。
// serviceCall 闭包捕获 req.VLANID 并传给 service.SetAccessVlan。
func (h *PortWriteHandler) SetAccessVlan(c *gin.Context) {
	var req SetAccessVlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}
	h.execSinglePort(c, portcollection.ActionSetAccessVLAN, operlog.OperTypeUpdate, req.PortID, "",
		func(ctx context.Context, portID, operator, desc string) (*portwrite.PortResult, error) {
			return h.service.SetAccessVlan(ctx, portID, req.VLANID, operator)
		})
}

// PortBinding 端口绑定 (POST /network/ports/write/port-binding)。
//
// OperType 分流 (design.md §6)：
//   - op=add    → OperTypeCreate(=1)  新增静态绑定记录
//   - op=remove → OperTypeDelete(=3)  删除已有绑定记录
//
// 绑定 PortBindingRequest 在 execSinglePort 之外（PATTERNS.md Option A），
// 校验 op ∈ {add, remove} (oneof tag 双重防线，service 仍校验)。
// serviceCall 闭包捕获 req.Op/IPAddress/MACAddress 并传给 service.PortBinding。
func (h *PortWriteHandler) PortBinding(c *gin.Context) {
	var req PortBindingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}
	operType := operlog.OperTypeCreate
	if req.Op == "remove" {
		operType = operlog.OperTypeDelete
	}
	h.execSinglePort(c, portcollection.ActionPortBinding, operType, req.PortID, "",
		func(ctx context.Context, portID, operator, desc string) (*portwrite.PortResult, error) {
			return h.service.PortBinding(ctx, portID, req.Op, req.IPAddress, req.MACAddress, operator)
		})
}

// execSinglePort 单端口 handler 公共流程 (5 个 v1.19 方法 + 2 个 v1.20.1 方法共用)。
//
// 步骤 (PATTERNS.md §4c + RESEARCH §3.3 sentinel→HTTP 表)：
//  1. （绑定已由 caller 在进入本函数前完成 — v1.19 5 方法绑定 PortWriteRequest；
//     v1.20.1 SetAccessVlan/PortBinding 绑定各自专用 struct。caller 传 portID + description 进来）
//  2. operator := utils.GetUsername(c)
//  3. D-02 预 SELECT DevicePortStatus → 拼 before_value JSON（端口行不存在 → {}）
//  4. 调 service 方法 (serviceCall) → sentinel err 用 errors.Is 翻译成 4xx/500，**不写 audit**
//  5. PortResult.Status=="failed" 路径继续走 audit + operlog + response.Success (200)
//  6. D-04 写 audit 行 (after_value per D-03, device_response per A5)
//  7. Path C：operlog.Record(..., WithOperParam({audit_ids:[auditID], ...}))
//  8. response.Success(c, result)
//
// serviceCall 接收 description 参数（仅 SetDescription handler 用；其他 handler 忽略）。
//
// v1.20.1 重构：原本 execSinglePort 内部调 c.ShouldBindJSON(&PortWriteRequest{})，
// 但 SetAccessVlan/PortBinding handler 已经绑定过各自 struct（PATTERNS.md Option A：
// bind outside DRY）。两次 ShouldBindJSON 会让第二次读到 EOF。改为由 caller 绑定并
// 传 portID + description 作为参数；v1.19 5 方法在各自 handler 顶部绑定 PortWriteRequest
// 后再调 execSinglePort。
func (h *PortWriteHandler) execSinglePort(
	c *gin.Context,
	action portcollection.PortAction,
	operType int,
	portID string,
	description string,
	serviceCall func(ctx context.Context, portID, operator, desc string) (*portwrite.PortResult, error),
) {
	operator := utils.GetUsername(c)

	// D-02 预 SELECT before_value（端口行不存在 → {}, 不阻塞流程）
	var port models.DevicePortStatus
	beforeValue := json.RawMessage([]byte("{}"))
	deviceID := ""
	portFindErr := h.core.GetDB().Where("id = ?", portID).First(&port).Error
	if portFindErr == nil {
		deviceID = port.DeviceID
		beforeValue = buildBeforeValue(&port)
	}

	result, err := serviceCall(c.Request.Context(), portID, operator, description)
	if err != nil {
		// sentinel 路径（端口/设备不存在 + v1.20.1 入参校验）：4xx，不写 audit、不调 operlog
		switch {
		case errors.Is(err, portwrite.ErrPortNotFound):
			response.Error(c, http.StatusNotFound, "端口不存在")
			return
		case errors.Is(err, portwrite.ErrDeviceNotFound):
			response.Error(c, http.StatusNotFound, "设备不存在")
			return
		case errors.Is(err, portwrite.ErrVlanIdOutOfRange):
			response.Error(c, http.StatusBadRequest, "VLAN ID 必须在 1-4094 之间")
			return
		case errors.Is(err, portwrite.ErrBindOpInvalid):
			response.Error(c, http.StatusBadRequest, "绑定操作必须是 add 或 remove")
			return
		case errors.Is(err, portwrite.ErrIPAddressInvalid):
			response.Error(c, http.StatusBadRequest, "IP 地址格式不合法")
			return
		case errors.Is(err, portwrite.ErrMACAddressInvalid):
			response.Error(c, http.StatusBadRequest, "MAC 地址格式不合法")
			return
		}
		// 非 sentinel error：设备执行失败（拒绝/transport）或系统错误（DB/RenderCommand）。
		// 有 failed result → fall through 走 audit + 200（handler 设计：Status==failed
		//   记审计后返回，前端能看到失败原因 + 审计可追溯）。
		//   同时修复 response.Error(c, int, ...) 的 int→400 陷阱：toAppError 的 case int
		//   强制 HTTPStatus=BadRequest，传 500 整数会变 400（2026-07-09 华为 failed 400 真因）。
		// 无 result → 系统错误，500（传 err 作 error 类型，走 toAppError error 分支→500）
		if result == nil || result.Status != "failed" {
			response.Error(c, err, err.Error())
			return
		}
		// result.Status == "failed"：fall through 到下方 audit + operlog + 200
	}

	// 关键分支：result.Status=="failed" 不走 response.Error，继续 audit + operlog + 200
	// (RESEARCH §3.3, PATTERNS.md §4d 末行)

	// D-04 写 audit 行（Path C 第 1 步：先 INSERT 拿 audit_id）
	auditRow := buildAuditRow(result, beforeValue, deviceID, operator)
	if createErr := h.core.GetDB().Create(auditRow).Error; createErr != nil {
		// audit 写入失败不阻塞响应（与 batch 路径一致），但记录告警。
		log.Printf("port_write audit insert failed portID=%s action=%s: %v", portID, action, createErr)
	}

	// 查 device_name + interface_name 用于 oper_param 可读性（UUID 不便阅读）；
	// 查询失败用 deviceID/PortID fallback，不阻塞响应。
	deviceName := deviceID
	if deviceID != "" {
		var dev models.NetworkDevice
		if err := h.core.GetDB().Select("device_name").Where("id = ?", deviceID).First(&dev).Error; err == nil && dev.DeviceName != "" {
			deviceName = dev.DeviceName
		}
	}
	interfaceName := port.InterfaceName
	if interfaceName == "" {
		interfaceName = portID // fallback：端口行不存在
	}

	// Path C 第 2 步：把 audit_id 嵌 operlog oper_param（audit.oper_log_id 列保持 NULL）
	operParam := buildSinglePortOperParam(auditRow.ID, deviceName, interfaceName, string(action), operator, result.Status)
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
		// CR-04 (2026-07-09): batch 路径也必须把 v1.20.1 4 个 validator sentinel 翻译为 400,
		// 否则 service 即使 emit sentinel,handler fall through default:500。
		case errors.Is(err, portwrite.ErrVlanIdOutOfRange):
			response.Error(c, http.StatusBadRequest, "VLAN ID 必须在 1-4094 之间")
		case errors.Is(err, portwrite.ErrBindOpInvalid):
			response.Error(c, http.StatusBadRequest, "绑定操作必须是 add 或 remove")
		case errors.Is(err, portwrite.ErrIPAddressInvalid):
			response.Error(c, http.StatusBadRequest, "IP 地址格式不合法")
		case errors.Is(err, portwrite.ErrMACAddressInvalid):
			response.Error(c, http.StatusBadRequest, "MAC 地址格式不合法")
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

	// 查 device_name + portID→interface_name 用于 oper_param 可读性（UUID 不便阅读）；
	// 查询失败用 deviceID/PortID fallback，不阻塞响应。
	deviceName := req.DeviceID
	var dev models.NetworkDevice
	if err := h.core.GetDB().Select("device_name").Where("id = ?", req.DeviceID).First(&dev).Error; err == nil && dev.DeviceName != "" {
		deviceName = dev.DeviceName
	}
	portNames := loadPortInterfaceNames(h.core.GetDB(), req.PortIDs)

	// Path C：汇总 operlog 行 oper_param 含 device_name + 各分类接口名 + audit_ids 完整列表
	batchSummary := buildBatchSummaryOperParam(
		string(req.Action),
		len(req.PortIDs),
		result,
		deviceName,
		portNames,
		auditIDs,
	)
	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), ModulePortWrite, operlog.OperTypeBatch,
		operlog.WithOperParam(batchSummary))

	response.Success(c, result)
}

// ====================== Helpers ======================

// buildBeforeValue 拼接 D-02 before_value JSON 快照
//
// 字段结构固定：{"admin_status":..., "dot1x_enabled":..., "description":..., "interface_name":..., "vlan":...}
// （便于审计完整快照，description action 之外的 action 也带 description）
//
// CR-02 (2026-07-09 修复)：
// v1.20.1 set_access_vlan 的 skipped 路径 (NoOp short-circuit) 会让 after_value 被
// 覆盖为 before_value (buildAuditRow:493-495),而 set_access_vlan 审计的核心字段
// 就是 vlan — 原版 buildBeforeValue 不带 vlan 字段导致审计行 before/after 都无
// vlan 信息,审计员无法回答"这个端口的 vlan 当前值是什么"。
// port.VLAN 是 *int (nullable),nil 跳过（端口尚未被 cron 采集）。
func buildBeforeValue(port *models.DevicePortStatus) json.RawMessage {
	snapshot := map[string]interface{}{
		"admin_status":   port.AdminStatus,
		"dot1x_enabled":  port.Dot1xEnabled,
		"description":    port.Description,
		"interface_name": port.InterfaceName,
	}
	if port.VLAN != nil {
		snapshot["vlan"] = *port.VLAN
	}
	b, _ := json.Marshal(snapshot)
	return b
}

// buildAfterValue 拼接 D-03 after_value 目标态快照（按 action 分支）。
//
// v1.19 actions (固定态)：
//
//	shutdown → {"admin_status":"down"}
//	undo_shutdown → {"admin_status":"up"}
//	dot1x_enable → {"dot1x_enabled":true}
//	dot1x_disable → {"dot1x_enabled":false}
//	description → caller 在 buildAuditRow 里用 PortResult.CurrentState 覆盖
//
// v1.20.1 actions (动态态，从 PortResult.Extra 读取 service 注入的 vlanId / bindOp / ipAddress / macAddress)：
//
//	set_access_vlan → {"vlanId":100}
//	port_binding    → {"ipAddress":"...","macAddress":"...","bindOp":"add"} (macAddress 可选)
//
// skipped（NoOp）路径：caller 会在 buildAuditRow 里覆盖为 before_value。
// pr 参数为 nil 或 pr.Extra 为 nil 时，v1.20.1 两个 action 返空 `{}` 兜底（向后兼容 v1.19 callers）。
func buildAfterValue(action portcollection.PortAction, pr *portwrite.PortResult) json.RawMessage {
	switch action {
	case portcollection.ActionShutdown:
		return json.RawMessage([]byte(`{"admin_status":"down"}`))
	case portcollection.ActionUndoShutdown:
		return json.RawMessage([]byte(`{"admin_status":"up"}`))
	case portcollection.ActionDot1xEnable:
		// dot1x_user_limit 由 W2 service 写入 pr.Extra["dot1xUserLimit"] (Renderer 实际下发的 limit 值)。
		// 类型断言 int: 防御非 int 类型(避免 %v 输出 "<nil>" 等异常值),失败则降级不加字段。
		parts := []string{`"dot1x_enabled":true`}
		if pr != nil && pr.Extra != nil {
			if v, ok := pr.Extra["dot1xUserLimit"]; ok {
				if n, ok := v.(int); ok {
					parts = append(parts, fmt.Sprintf(`"dot1x_user_limit":%d`, n))
				}
			}
		}
		return json.RawMessage([]byte("{" + strings.Join(parts, ",") + "}"))
	case portcollection.ActionDot1xDisable:
		return json.RawMessage([]byte(`{"dot1x_enabled":false}`))
	case portcollection.ActionSetAccessVLAN:
		// v1.20.1: 从 PortResult.Extra 读 vlanId（W2 service 在成功路径写入）
		if pr != nil && pr.Extra != nil {
			if v, ok := pr.Extra["vlanId"]; ok {
				return json.RawMessage([]byte(fmt.Sprintf(`{"vlanId":%v}`, v)))
			}
		}
		return json.RawMessage([]byte(`{}`))
	case portcollection.ActionPortBinding:
		// v1.20.1: 从 PortResult.Extra 读 bindOp / ipAddress / macAddress（W2 service 在成功路径写入）
		if pr != nil && pr.Extra != nil {
			parts := make([]string, 0, 3)
			if ip, ok := pr.Extra["ipAddress"].(string); ok && ip != "" {
				parts = append(parts, fmt.Sprintf(`"ipAddress":%q`, ip))
			}
			if mac, ok := pr.Extra["macAddress"].(string); ok && mac != "" {
				parts = append(parts, fmt.Sprintf(`"macAddress":%q`, mac))
			}
			if op, ok := pr.Extra["bindOp"].(string); ok && op != "" {
				parts = append(parts, fmt.Sprintf(`"bindOp":%q`, op))
			}
			if len(parts) > 0 {
				return json.RawMessage([]byte("{" + strings.Join(parts, ",") + "}"))
			}
		}
		return json.RawMessage([]byte(`{}`))
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
//   - 其他 action → buildAfterValue(action, pr)
//     v1.20.1 set_access_vlan / port_binding 的 after_value 由 buildAfterValue 从 pr.Extra 读出
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
	afterValue := buildAfterValue(pr.Action, pr)
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
// 字段：{audit_ids:[auditID], device_name, interface_name, action, operator, result_status}
//
// 2026-07-09 可读性优化：device_id/port_id (UUID) 换成 device_name/interface_name，
// 便于审计日志直接阅读；audit_ids 保留用于关联 PortWriteAudit 详情。
func buildSinglePortOperParam(auditID, deviceName, interfaceName, action, operator, resultStatus string) string {
	b, _ := json.Marshal(map[string]interface{}{
		"audit_ids":      []string{auditID},
		"device_name":    deviceName,
		"interface_name": interfaceName,
		"action":         action,
		"operator":       operator,
		"result_status":  resultStatus,
	})
	return string(b)
}

// buildBatchSummaryOperParam 拼接 batch operlog.oper_param JSON (CONV-04)
//
// 字段：{device_name, action, batch_size, succeeded/failed/skipped_count,
//
//	succeeded/failed/skipped_interfaces:[...], audit_ids:[...]}
//
// 2026-07-09 可读性优化：device_id 换成 device_name，新增各分类接口名列表
// （succeeded_interfaces 等），便于审计日志直接阅读；audit_ids 保留关联详情。
func buildBatchSummaryOperParam(action string, batchSize int, result *portwrite.BatchResult, deviceName string, portNames map[string]string, auditIDs []string) string {
	b, _ := json.Marshal(map[string]interface{}{
		"device_name":          deviceName,
		"action":               action,
		"batch_size":           batchSize,
		"succeeded_count":      len(result.Succeeded),
		"failed_count":         len(result.Failed),
		"skipped_count":        len(result.Skipped),
		"succeeded_interfaces": collectInterfaceNames(result.Succeeded, portNames),
		"failed_interfaces":    collectInterfaceNames(result.Failed, portNames),
		"skipped_interfaces":   collectInterfaceNames(result.Skipped, portNames),
		"audit_ids":            auditIDs,
	})
	return string(b)
}

// collectInterfaceNames 从 PortResult 列表提取接口名（portNames 映射缺失时 fallback PortID）。
func collectInterfaceNames(results []portwrite.PortResult, portNames map[string]string) []string {
	names := make([]string, 0, len(results))
	for _, r := range results {
		if name, ok := portNames[r.PortID]; ok && name != "" {
			names = append(names, name)
		} else {
			names = append(names, r.PortID) // fallback：端口行不存在时用 UUID
		}
	}
	return names
}

// loadPortInterfaceNames 一次 SELECT 取齐 portID → interface_name 映射（oper_param 可读性用）。
//
// 端口行不存在时不出现在 map 中（caller fallback 到 PortID）。
func loadPortInterfaceNames(db *gorm.DB, portIDs []string) map[string]string {
	out := make(map[string]string, len(portIDs))
	if len(portIDs) == 0 {
		return out
	}
	var ports []models.DevicePortStatus
	if err := db.Where("id IN ?", portIDs).Find(&ports).Error; err != nil {
		return out
	}
	for _, p := range ports {
		out[p.ID] = p.InterfaceName
	}
	return out
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
