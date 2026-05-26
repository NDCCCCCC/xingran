package system

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/addomain"
	applogger "github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
	"gorm.io/gorm"
)

type ADDeptSyncHandler struct {
	db         *gorm.DB
	syncService *addomain.DeptToADSyncService
}

func NewADDeptSyncHandler(db *gorm.DB, syncService *addomain.DeptToADSyncService) *ADDeptSyncHandler {
	return &ADDeptSyncHandler{
		db:          db,
		syncService: syncService,
	}
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

	response.Success(c, gin.H{
		"message": "部门同步已启动",
	})
}
