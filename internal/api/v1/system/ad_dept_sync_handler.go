package system

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/addomain"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
	"gorm.io/gorm"
)

type ADDeptSyncHandler struct {
	db          *gorm.DB
	syncService *addomain.DeptToADSyncService
	core        *core.Core
}

func NewADDeptSyncHandler(db *gorm.DB, syncService *addomain.DeptToADSyncService) *ADDeptSyncHandler {
	return &ADDeptSyncHandler{
		db:          db,
		syncService: syncService,
	}
}

// WithCore 注入 core 依赖（用于操作日志埋点），链式调用。Phase 34 Wave 7 新增。
// NOTE: 该 handler 当前未注册在任何 router（仅 test 引用），但保留 setter
// 以便未来 wire-up 时立即可用。
func (h *ADDeptSyncHandler) WithCore(core *core.Core) *ADDeptSyncHandler {
	if h != nil {
		h.core = core
	}
	return h
}

// SyncDeptToAD 同步部门结构到AD
func (h *ADDeptSyncHandler) SyncDeptToAD(c *gin.Context) {
	var req struct {
		ADConfigID string `json:"adConfigId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	result, err := h.syncService.SyncDeptStructureToAD(c.Request.Context(), req.ADConfigID)
	if err != nil {
		applogger.Errorf("部门同步失败: %v", err)
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	recordOperLog(c, h.core, "AD部门同步", OperTypeSync)

	response.Success(c, result)
}

// GetDeptSyncStatus 查询部门同步状态
func (h *ADDeptSyncHandler) GetDeptSyncStatus(c *gin.Context) {
	configID := c.Param("configId")

	// 从数据库查询最新的部门同步日志（使用OUCount作为部门数量）
	var syncLog models.ADSyncLog
	err := h.db.Where("ad_config_id = ?", configID).
		Order("created_at DESC").
		First(&syncLog).Error

	stats := &addomain.DeptSyncStats{
		LastSyncStatus: "pending",
		TotalDepts:     0,
	}

	if err == nil {
		stats.LastSyncTime = &syncLog.CreatedAt
		stats.LastSyncStatus = string(syncLog.SyncStatus)
		stats.TotalDepts = syncLog.OUCount
		stats.SyncedDepts = syncLog.OUCount - syncLog.ErrorCount
		stats.FailedDepts = syncLog.ErrorCount
	}

	response.Success(c, stats)
}

// TriggerDeptSync 手动触发部门同步
func (h *ADDeptSyncHandler) TriggerDeptSync(c *gin.Context) {
	var req struct {
		ADConfigID string `json:"adConfigId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	// 异步执行同步，避免长时间阻塞HTTP请求
	go func() {
		_, err := h.syncService.SyncDeptStructureToAD(c.Request.Context(), req.ADConfigID)
		if err != nil {
			applogger.Errorf("手动触发部门同步失败: %v", err)
		}
	}()

	// TriggerDeptSync starts an async sync goroutine. Record the trigger as
	// Sync — the actual completion is logged to sys_ad_sync_log separately.
	recordOperLog(c, h.core, "AD部门同步", OperTypeSync)

	response.Success(c, gin.H{
		"message": "部门同步已启动",
	})
}
