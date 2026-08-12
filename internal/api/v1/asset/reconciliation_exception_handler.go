package asset

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/services/asset"
	"github.com/xingran-next/xingran-go-backend/internal/services/operations"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// ReconciliationExceptionHandler 资产对账例外规则 handler
//
// R1 边界: 仅 List + GetByID skeleton,R3 实现 Create/Update/Delete/enable/disable。
// 例外规则 seed 数据已就位(42-01 migration_169),R1 仅提供只读查询入口,
// 不调 operlog.Record(读操作)。
//
// Phase 44 R3 / Plan 44-02 Task 3: 加 baselineSvc 字段(降噪基线 Snapshot/Compare)。
type ReconciliationExceptionHandler struct {
	service    asset.ReconciliationExceptionService
	core       *core.Core
	baselineSvc asset.ReconciliationBaselineService
}

func NewReconciliationExceptionHandler(svc asset.ReconciliationExceptionService) *ReconciliationExceptionHandler {
	return &ReconciliationExceptionHandler{service: svc}
}

func (h *ReconciliationExceptionHandler) WithCore(core *core.Core) *ReconciliationExceptionHandler {
	if h != nil {
		h.core = core
	}
	return h
}

// WithBaselineService 注入 baseline service(Phase 44 R3 / Plan 44-02 Task 3)
//
// 链式注入,与 WithCore 模式一致。router 内构造 baselineSvc 后调用本方法。
func (h *ReconciliationExceptionHandler) WithBaselineService(svc asset.ReconciliationBaselineService) *ReconciliationExceptionHandler {
	if h != nil {
		h.baselineSvc = svc
	}
	return h
}

// ListRules 查询例外规则列表
func (h *ReconciliationExceptionHandler) ListRules(c *gin.Context) {
	var params asset.ExceptionRuleListParams
	if err := c.ShouldBindJSON(&params); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	result, err := h.service.List(c.Request.Context(), &params)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, result)
}

// GetRuleByID 按 ID 查询单条规则
func (h *ReconciliationExceptionHandler) GetRuleByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "规则ID不能为空")
		return
	}

	rule, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	if rule == nil {
		response.Error(c, http.StatusNotFound, "规则不存在")
		return
	}
	response.Success(c, rule)
}

// ============================================================================
// R3 写操作 + 命中测试 handler (Phase 44 R3 / Plan 44-01 Task 5)
//
// 强制约定(CLAUDE.md "操作日志记录约定 — 强制"):
//   - CreateRule / UpdateRule / DeleteRule 在 success path 末尾、response.Success
//     之前调 operlog.Record(ModuleReconciliationExceptionRule, OperTypeXxx)
//   - 例外规则 CRUD 无敏感字段(reason 是普通文本),用 operlog.Record 非 RecordWithBody
//   - TestRule 是读操作,不调 operlog(参考 ListRules/GetRuleByID)
// ============================================================================

// CreateRule 创建例外规则
//
// 行为:
//  1. ShouldBindJSON 失败 → 400
//  2. service.Create 失败(含 ValidateCIDR/Actions/SeverityOverride/Reason) → 500
//  3. success path 末尾 operlog.Record(OperTypeCreate) → 写 sys_oper_log
//  4. response.Success(rule)
func (h *ReconciliationExceptionHandler) CreateRule(c *gin.Context) {
	var req asset.CreateExceptionRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	rule, err := h.service.Create(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	// operlog 写入(CLAUDE.md 强制约定,新增 → OperTypeCreate)
	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), ModuleReconciliationExceptionRule, operlog.OperTypeCreate)

	response.Success(c, rule)
}

// UpdateRule 更新例外规则
//
// 行为:
//  1. URL 参数 :id + body(UpdateExceptionRuleRequest)
//  2. service.Update 失败(含校验 / 不存在) → 500
//  3. success path 末尾 operlog.Record(OperTypeUpdate)
func (h *ReconciliationExceptionHandler) UpdateRule(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "规则ID不能为空")
		return
	}

	var req asset.UpdateExceptionRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	if err := h.service.Update(c.Request.Context(), id, &req); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	// operlog 写入(CLAUDE.md 强制约定,修改 → OperTypeUpdate)
	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), ModuleReconciliationExceptionRule, operlog.OperTypeUpdate)

	response.Success(c, gin.H{"id": id})
}

// DeleteRule 软删除例外规则
//
// 行为:
//  1. URL 参数 :id
//  2. service.Delete 失败(规则不存在) → 500
//  3. success path 末尾 operlog.Record(OperTypeDelete)
func (h *ReconciliationExceptionHandler) DeleteRule(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "规则ID不能为空")
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	// operlog 写入(CLAUDE.md 强制约定,删除 → OperTypeDelete)
	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), ModuleReconciliationExceptionRule, operlog.OperTypeDelete)

	response.Success(c, gin.H{"id": id})
}

// TestRule 命中测试工具(EXCEPTION-04 / SC 6)
//
// 读操作 — 不调 operlog.Record(参考 ListRules/GetRuleByID 无 operlog)。
//
// 入参 body: { "ip": "192.168.0.10", "userId": "uuid?", "deptId": "uuid?" }
// 返回: MatchTestResult { matchedRules, mergedActions, finalSeverity, isSilence, needsUserDept }
func (h *ReconciliationExceptionHandler) TestRule(c *gin.Context) {
	var req struct {
		IP     string `json:"ip" binding:"required"`
		UserID string `json:"userId"`
		DeptID string `json:"deptId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: ip 必填")
		return
	}

	result, err := h.service.MatchTest(c.Request.Context(), req.IP, req.UserID, req.DeptID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, result)
}

// ============================================================================
// Phase 44 R3 / Plan 44-02 Task 3 — 降噪基线 Snapshot / Compare handler (BLOCKER-2)
//
// 44-01 Task 5 仅产出 CreateRule/UpdateRule/DeleteRule/TestRule。本 plan Task 3 在
// handler 文件新增 SnapshotBaseline + CompareBaseline 两个 handler(后端实现)。
//
// operlog 强制约定(CLAUDE.md):
//   - SnapshotBaseline: 写操作(写 sys_config 覆盖现有 baseline)→ success path 末尾
//     调 operlog.Record(OperTypeUpdate),基线是"更新当前快照"语义
//   - CompareBaseline: 读操作(读 sys_config + COUNT)→ 不调 operlog.Record
// ============================================================================

// SnapshotBaseline 记录当前为基线(D-R3-A4-01)
//
// ⚠ 运维责任(BLOCKER-3):R3 部署前 + R2 数据保留期内必须调用本端点记录 R2 末期基线,
// 否则 dashboard "降噪效果"卡片无法显示(SC 8 ≥60% 降噪不可量化验证)。
// 无 baseline 时 CompareBaseline 返回 400,前端显示引导提示。
//
// 行为:
//  1. baselineSvc.Snapshot 失败 → 500
//  2. success path 末尾 operlog.Record(OperTypeUpdate) → 写 sys_oper_log
//  3. response.Success(snapshot)
func (h *ReconciliationExceptionHandler) SnapshotBaseline(c *gin.Context) {
	if h.baselineSvc == nil {
		response.Error(c, http.StatusInternalServerError, "baseline service 未注入")
		return
	}

	snapshot, err := h.baselineSvc.Snapshot(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	// operlog 写入(CLAUDE.md 强制约定,基线是更新语义 → OperTypeUpdate)
	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), ModuleReconciliationExceptionRule, operlog.OperTypeUpdate)

	response.Success(c, snapshot)
}

// CompareBaseline 对比当前与 baseline,返回下降百分比(D-R3-A4-01)
//
// 读操作 — 不调 operlog.Record(参考 TestRule/Compare 只读路径)。
//
// 行为:
//  1. baselineSvc.Compare 失败(含 "未找到基线") → 400(前端依赖此状态码显示引导提示)
//  2. 其他 service 失败 → 500
//  3. success → response.Success(result)
func (h *ReconciliationExceptionHandler) CompareBaseline(c *gin.Context) {
	if h.baselineSvc == nil {
		response.Error(c, http.StatusInternalServerError, "baseline service 未注入")
		return
	}

	result, err := h.baselineSvc.Compare(c.Request.Context())
	if err != nil {
		// 无 baseline 时返回 400,前端 Alert 引导运维先记录基线(BLOCKER-3 可观察条件)
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	response.Success(c, result)
}

// ============================================================================
// Phase 44 R3 / Plan 44-02 Task 4 — Excel 导入/导出/模板 (SC 9)
//
// 方案 B (WARN-7 锁定):专用 ImportRules handler + service.ImportFromExcel 后处理
// scope_name→scope_id + TEXT[] 转换。
//
// operlog 强制约定(CLAUDE.md):
//   - ImportRules: 写操作 → success path 末尾 operlog.Record(OperTypeImport)
//   - ExportRules: 导出 → success path 末尾 operlog.Record(OperTypeExport)
//   - DownloadTemplate: 下载 → success path 末尾 operlog.Record(OperTypeDownload)
// ============================================================================

// ImportRules Excel 导入例外规则(SC 9)
//
// 行为:
//  1. c.FormFile 拿 file
//  2. service.ImportFromExcel 调 excel_service.ImportData 写入 + 后处理 scope_id + TEXT[]
//  3. success path 末尾 operlog.Record(OperTypeImport) → 写 sys_oper_log
//  4. response.Success(result)
func (h *ReconciliationExceptionHandler) ImportRules(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "请上传 Excel 文件")
		return
	}

	result, err := h.service.ImportFromExcel(c.Request.Context(), file)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	// operlog 写入(CLAUDE.md 强制约定,导入 → OperTypeImport)
	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), ModuleReconciliationExceptionRule, operlog.OperTypeImport)

	response.Success(c, result)
}

// ExportRules Excel 导出例外规则
func (h *ReconciliationExceptionHandler) ExportRules(c *gin.Context) {
	excelSvc := operations.NewExcelService(h.core.GetDB(), nil, nil, nil)
	f, err := excelSvc.ExportData(c.Request.Context(), "reconciliationExceptionRule", nil)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	// operlog 写入(导出 → OperTypeExport)
	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), ModuleReconciliationExceptionRule, operlog.OperTypeExport)

	c.Header("Content-Disposition", `attachment; filename="reconciliation_exception_rules.xlsx"`)
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	if err := f.Write(c.Writer); err != nil {
		// 写流失败仅 log,response 已部分发出无法改 status
		_ = err
	}
}

// DownloadTemplate 下载例外规则 Excel 模板
func (h *ReconciliationExceptionHandler) DownloadTemplate(c *gin.Context) {
	excelSvc := operations.NewExcelService(h.core.GetDB(), nil, nil, nil)
	f, err := excelSvc.GenerateTemplate("reconciliationExceptionRule")
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	// operlog 写入(下载模板 → OperTypeDownload)
	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), ModuleReconciliationExceptionRule, operlog.OperTypeDownload)

	c.Header("Content-Disposition", `attachment; filename="reconciliation_exception_rule_template.xlsx"`)
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	if err := f.Write(c.Writer); err != nil {
		_ = err
	}
}
