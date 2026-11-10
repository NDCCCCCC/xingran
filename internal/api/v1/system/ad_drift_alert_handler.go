package system

// Phase 144 · 同步漂移告警通道收口 —— 查询/配置/触发端点（ALRT-01/02）。
//
// 5 个 POST 端点：
//   POST /system/ad-drift-alert/status         读组 RequirePermissions
//   POST /system/ad-drift-alert/logs           读组 RequirePermissions（分页）
//   POST /system/ad-drift-alert/config         读组 RequirePermissions
//   POST /system/ad-drift-alert/config/update  写组 RequireStrictPermissions
//   POST /system/ad-drift-alert/trigger        写组 RequireStrictPermissions
//
// 全部 response.Success()/Error() 包装（禁裸 c.JSON）、c.Request.Context() 透传；
// 分页 list/total/current/pageSize（current 从 1）。
// 写端点 success path 在 response.Success 之前记录 operlog：
//   config/update → RecordWithBody(module "同步漂移告警配置", OperTypeUpdate)
//   trigger       → Record(module "同步漂移告警", OperTypeSync)
// 敏感 webhook 配置值走 RecordWithBody 自动脱敏（operlog.go 已登记
// webhookUrl/webhookKey/access_token 等字面量）。

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	"github.com/xingran-next/xingran-go-backend/internal/services/addomain"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
	"gorm.io/gorm"
)

// AdDriftAlertHandler 漂移告警端点处理器（Handler-Service 分层）。
type AdDriftAlertHandler struct {
	core *core.Core
	db   *gorm.DB
	svc  *addomain.ADSyncDriftAlertService
}

// NewAdDriftAlertHandler 构造（依赖注入 core）。
func NewAdDriftAlertHandler(c *core.Core) *AdDriftAlertHandler {
	db := c.GetDB()
	return &AdDriftAlertHandler{
		core: c,
		db:   db,
		svc:  addomain.NewADSyncDriftAlertService(db),
	}
}

// GetStatus 通道运行状态。
func (h *AdDriftAlertHandler) GetStatus(c *gin.Context) {
	status, err := h.svc.GetStatus(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, status)
}

// ListLogs 触发记录分页（current 从 1）。
func (h *AdDriftAlertHandler) ListLogs(c *gin.Context) {
	var req struct {
		Current  int `json:"current"`
		PageSize int `json:"pageSize"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		// 空体 / 非法体：回落默认分页（不阻断读端点）
		req.Current, req.PageSize = 1, 10
	}
	// B-1 / quick 261110-161：传当前登录用户，供 ListTriggerLogs 按
	// sys_notice_read 补每行的 IsRead（按用户维度已读，复用既有设施）。
	userID, _ := getUserID(c)
	page, err := h.svc.ListTriggerLogs(c.Request.Context(), req.Current, req.PageSize, userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, page)
}

// MarkLogRead 把某条触发记录对应的站内通知标记为已读（B-1 / quick 261110-161）。
//
// 关联键 = 触发记录 JsonResult 里的 noticeId（emit 通道 2 写入）。复用
// services.NoticeService.MarkNoticeRead 的按用户已读设施，不新建表、不给
// append-only 的 sys_oper_log 加列。noticeId 为空 / 记录不存在 → 400，不静默成功。
func (h *AdDriftAlertHandler) MarkLogRead(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	// logs 记录 ID（sys_oper_log.id）；按它查出 noticeId，再按用户标记已读。
	var req struct {
		LogID string `json:"logId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.LogID == "" {
		response.Error(c, apperrors.ParamMissing("logId"))
		return
	}
	var jsonResult string
	if err := h.db.WithContext(c.Request.Context()).
		Model(&models.OperLog{}).
		Where("id = ? AND method = ?", req.LogID, addomain.DriftAlertTriggerLogMethod).
		Pluck("json_result", &jsonResult).Error; err != nil || jsonResult == "" {
		response.Error(c, apperrors.ParamMissing("触发记录不存在或无关联通知"))
		return
	}
	var payload struct {
		NoticeID string `json:"noticeId"`
	}
	if err := json.Unmarshal([]byte(jsonResult), &payload); err != nil || payload.NoticeID == "" {
		response.Error(c, apperrors.ParamMissing("该记录无关联通知，无法标记已读"))
		return
	}
	noticeSvc := services.NewNoticeService(h.db)
	if err := noticeSvc.MarkNoticeRead(c.Request.Context(), payload.NoticeID, userID, c.ClientIP()); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	operlog.Record(c, h.core.OperLogService, h.db, "同步漂移告警", operlog.OperTypeUpdate)
	response.Success(c, gin.H{"message": "标记成功"})
}

// GetConfig 读 9 个 sys_config 旋钮。
func (h *AdDriftAlertHandler) GetConfig(c *gin.Context) {
	view, err := h.svc.GetConfig(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, view)
}

// UpdateConfig 写 9 个 sys_config 旋钮（白名单过滤 + RecordWithBody 脱敏）。
func (h *AdDriftAlertHandler) UpdateConfig(c *gin.Context) {
	// 先读取并恢复一次 body：ShouldBindJSON 会消费 Request.Body，
	// 若不缓存 raw，后续 operlog.RecordWithBody 的 GetRawData() 只能拿到空串，
	// 敏感 webhook 值就无从脱敏（Phase 144 安全红线）。
	raw, _ := c.GetRawData()
	c.Request.Body = io.NopCloser(bytes.NewBuffer(raw))

	var updates map[string]string
	if err := c.ShouldBindJSON(&updates); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}
	if err := h.svc.UpdateConfig(c.Request.Context(), updates); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	// 再次恢复 body 供 RecordWithBody 读取（bind 已消费一次）。
	c.Request.Body = io.NopCloser(bytes.NewBuffer(raw))
	operlog.RecordWithBody(c, h.core.OperLogService, h.db, "同步漂移告警配置", operlog.OperTypeUpdate)
	response.Success(c, gin.H{"message": "配置已更新"})
}

// Trigger 手动跑一轮漂移检测（写操作）。
func (h *AdDriftAlertHandler) Trigger(c *gin.Context) {
	if err := h.svc.TriggerOnce(c.Request.Context()); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	operlog.Record(c, h.core.OperLogService, h.db, "同步漂移告警", operlog.OperTypeSync)
	response.Success(c, gin.H{"message": "已触发一轮漂移检测"})
}
