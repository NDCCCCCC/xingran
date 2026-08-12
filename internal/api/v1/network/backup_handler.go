package network

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
	responseHelpers "github.com/xingran-next/xingran-go-backend/pkg/response"
	"gorm.io/gorm"
)

// BackupHandler 配置备份处理器
type BackupHandler struct {
	backupService *services.ConfigBackupService
	db            *gorm.DB
	core          *core.Core
}

// NewBackupHandler 创建配置备份处理器实例
func NewBackupHandler(backupService *services.ConfigBackupService, db *gorm.DB) *BackupHandler {
	return &BackupHandler{
		backupService: backupService,
		db:            db,
	}
}

// WithCore 注入 core 依赖（操作日志记录所需）。Phase 34 操作日志全模块覆盖。
func (h *BackupHandler) WithCore(core *core.Core) *BackupHandler {
	if h != nil {
		h.core = core
	}
	return h
}

// List 查询备份列表
// @Summary 查询备份列表
// @Description 分页查询配置备份列表
// @Tags 配置备份
// @Accept json
// @Produce json
// @Param request body object{current=int,pageSize=int,deviceId=string} true "查询条件"
// @Success 200 {object} response.Response{data=response.PageResponse}
// @Router /network/backups/list [post]
func (h *BackupHandler) List(c *gin.Context) {
	var rawReq map[string]interface{}
	if err := c.ShouldBindJSON(&rawReq); err != nil {
		rawReq = make(map[string]interface{})
	}

	current := getIntField(rawReq, "current", 1)
	pageSize := getIntField(rawReq, "pageSize", 10)

	var deviceID string
	if val, ok := rawReq["deviceId"].(string); ok {
		deviceID = val
	}

	backups, total, err := h.backupService.GetBackupList(c.Request.Context(), current, pageSize, deviceID, getOrderByColumn(rawReq), getIsAscPtr(rawReq))
	if err != nil {
		response.Error(c, apperrors.InternalServerError(err))
		return
	}

	pageResp := response.PageResponse{
		List:     backups,
		Total:    total,
		Current:  current,
		PageSize: pageSize,
	}

	response.Success(c, pageResp)
}

// Create 创建备份
// @Summary 创建备份
// @Description 为指定设备创建配置备份
// @Tags 配置备份
// @Accept json
// @Produce json
// @Param request body object{deviceId=string,backupType=string,changeReason=string,compressLarge=bool} true "备份请求"
// @Success 200 {object} response.Response
// @Router /network/backups [post]
func (h *BackupHandler) Create(c *gin.Context) {
	var req struct {
		DeviceID      string            `json:"deviceId" binding:"required"`
		BackupType    models.BackupType `json:"backupType" binding:"required,oneof=auto manual"`
		ChangeReason  string            `json:"changeReason"`
		CompressLarge bool              `json:"compressLarge"`
	}
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	// 查询设备名称
	var netDevice models.NetworkDevice
	if err := h.db.Where("id = ?", req.DeviceID).First(&netDevice).Error; err != nil {
		response.Error(c, apperrors.NetworkDeviceNotFound())
		return
	}

	userID, _ := c.Get("user_id")
	result, err := h.backupService.CreateBackup(c.Request.Context(), &services.BackupRequest{
		DeviceID:      req.DeviceID,
		DeviceName:    netDevice.DeviceName,
		BackupType:    req.BackupType,
		ChangeReason:  req.ChangeReason,
		CreatedBy:     userID.(string),
		CompressLarge: req.CompressLarge,
	})
	if !responseHelpers.HandleServiceError(c, err, "创建备份") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "配置备份", operlog.OperTypeCreate)
	response.Success(c, result)
}

// GetContent 获取备份内容
// @Summary 获取备份内容
// @Description 获取指定备份的配置内容
// @Tags 配置备份
// @Accept json
// @Produce json
// @Param id path string true "备份ID"
// @Success 200 {object} response.Response{data=string}
// @Router /network/backups/:id [post]
func (h *BackupHandler) GetContent(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("备份ID"))
		return
	}

	content, err := h.backupService.GetBackupContent(c.Request.Context(), id)
	if err != nil {
		response.Error(c, apperrors.InternalServerError(err))
		return
	}

	response.Success(c, gin.H{"content": content})
}

// GetContentFromBody 获取备份内容（从请求body中获取ID）
// @Summary 获取备份内容
// @Description 获取指定备份的配置内容（从请求body中获取ID）
// @Tags 配置备份
// @Accept json
// @Produce json
// @Param request body object{id=string} true "备份ID"
// @Success 200 {object} response.Response{data=string}
// @Router /network/backups/content [post]
func (h *BackupHandler) GetContentFromBody(c *gin.Context) {
	var req struct {
		ID string `json:"id" binding:"required"`
	}
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	content, err := h.backupService.GetBackupContent(c.Request.Context(), req.ID)
	if !responseHelpers.HandleServiceError(c, err, "获取备份内容") {
		return
	}

	response.Success(c, gin.H{"content": content})
}

// Delete 删除备份
// @Summary 删除备份
// @Description 删除指定备份记录
// @Tags 配置备份
// @Accept json
// @Produce json
// @Param id path string true "备份ID"
// @Success 200 {object} response.Response
// @Router /network/backups/:id/delete [post]
func (h *BackupHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("备份ID"))
		return
	}

	err := h.backupService.DeleteBackup(c.Request.Context(), id)
	if !responseHelpers.HandleServiceError(c, err, "删除备份") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "配置备份", operlog.OperTypeDelete)
	response.Success(c, gin.H{"message": "删除成功"})
}

// BatchDelete 批量删除备份
// @Summary 批量删除备份
// @Description 批量删除多个备份记录
// @Tags 配置备份
// @Accept json
// @Produce json
// @Param request body object{backupIds=[]string} true "备份ID列表"
// @Success 200 {object} response.Response
// @Router /network/backups/batch-delete [post]
func (h *BackupHandler) BatchDelete(c *gin.Context) {
	var req struct {
		BackupIDs []string `json:"backupIds" binding:"required,min=1"`
	}
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	err := h.backupService.BatchDeleteBackups(c.Request.Context(), req.BackupIDs)
	if !responseHelpers.HandleServiceError(c, err, "批量删除备份") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "配置备份", operlog.OperTypeBatch)
	response.Success(c, gin.H{
		"message": "批量删除成功",
		"count":   len(req.BackupIDs),
	})
}

// Diff 对比备份配置
// @Summary 对比备份配置
// @Description 对比两个备份的配置差异
// @Tags 配置备份
// @Accept json
// @Produce json
// @Param request body object{backupId1=string,backupId2=string} true "备份ID"
// @Success 200 {object} response.Response
// @Router /network/backups/diff [post]
func (h *BackupHandler) Diff(c *gin.Context) {
	var req struct {
		BackupID1 string `json:"backupId1" binding:"required"`
		BackupID2 string `json:"backupId2" binding:"required"`
	}
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	name1, name2, diff, err := h.backupService.DiffBackups(c.Request.Context(), req.BackupID1, req.BackupID2)
	if !responseHelpers.HandleServiceError(c, err, "对比备份") {
		return
	}

	response.Success(c, gin.H{
		"name1": name1,
		"name2": name2,
		"diff":  diff,
	})
}

// Restore 恢复配置
// @Summary 恢复配置
// @Description 从备份恢复设备配置
// @Tags 配置备份
// @Accept json
// @Produce json
// @Param request body object{backupId=string,deviceId=string} true "备份ID和设备ID"
// @Success 200 {object} response.Response
// @Router /network/backups/:id/restore [post]
func (h *BackupHandler) Restore(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("备份ID"))
		return
	}

	var req struct {
		DeviceID string `json:"deviceId" binding:"required"`
	}
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	err := h.backupService.RestoreBackup(c.Request.Context(), id, req.DeviceID)
	if !responseHelpers.HandleServiceError(c, err, "恢复配置") {
		return
	}

	// Restore 是把备份配置下发到设备 — 属高危写操作
	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "配置备份", operlog.OperTypeUpdate)
	response.Success(c, gin.H{"message": "恢复成功"})
}

// GetStatistics 获取备份统计信息
// @Summary 获取备份统计信息
// @Description 获取备份数据统计（用于仪表盘）
// @Tags 配置备份
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Router /network/backups/statistics [get]
func (h *BackupHandler) GetStatistics(c *gin.Context) {
	stats, err := h.backupService.GetBackupStatistics(c.Request.Context())
	if !responseHelpers.HandleServiceError(c, err, "获取备份统计") {
		return
	}

	response.Success(c, stats)
}

// BatchBackup 批量备份设备
// @Summary 批量备份设备
// @Description 批量为多个设备创建配置备份
// @Tags 配置备份
// @Accept json
// @Produce json
// @Param request body object{deviceIds=[]string,backupType=string} true "设备ID列表"
// @Success 200 {object} response.Response
// @Router /network/backups/batch [post]
func (h *BackupHandler) BatchBackup(c *gin.Context) {
	var req struct {
		DeviceIDs  []string          `json:"deviceIds" binding:"required,min=1"`
		BackupType models.BackupType `json:"backupType" binding:"required,oneof=auto manual"`
	}
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	userID, _ := c.Get("user_id")
	results, err := h.backupService.BatchBackupDevices(c.Request.Context(), req.DeviceIDs, req.BackupType, userID.(string))
	if !responseHelpers.HandleServiceError(c, err, "批量备份设备") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "配置备份", operlog.OperTypeBatch)
	response.Success(c, results)
}

// GetByVersion 获取指定版本的备份
// @Summary 获取指定版本的备份
// @Description 获取设备指定版本的配置备份
// @Tags 配置备份
// @Accept json
// @Produce json
// @Param deviceId query string true "设备ID"
// @Param version query int true "版本号"
// @Success 200 {object} response.Response
// @Router /network/backups/version [get]
func (h *BackupHandler) GetByVersion(c *gin.Context) {
	deviceID := c.Query("deviceId")
	versionStr := c.Query("version")

	if deviceID == "" || versionStr == "" {
		response.Error(c, apperrors.BadRequest("设备ID和版本号不能为空"))
		return
	}

	version, err := strconv.Atoi(versionStr)
	if err != nil {
		response.Error(c, apperrors.BadRequest("版本号格式错误"))
		return
	}

	var backup models.ConfigBackup
	if err := h.db.Where("device_id = ? AND version = ?", deviceID, version).First(&backup).Error; err != nil {
		response.Error(c, apperrors.BackupNotFound())
		return
	}

	response.Success(c, backup)
}

// GetHistory 获取设备备份历史
// @Summary 获取设备备份历史
// @Description 获取设备所有备份记录
// @Tags 配置备份
// @Accept json
// @Produce json
// @Param deviceId query string true "设备ID"
// @Success 200 {object} response.Response
// @Router /network/backups/history [get]
func (h *BackupHandler) GetHistory(c *gin.Context) {
	deviceID := c.Query("deviceId")
	if deviceID == "" {
		response.Error(c, apperrors.ParamMissing("设备ID"))
		return
	}

	var backups []models.ConfigBackup
	if err := h.db.Where("device_id = ?", deviceID).
		Order("version DESC").
		Find(&backups).Error; err != nil {
		responseHelpers.HandleServiceError(c, err, "获取备份历史")
		return
	}

	response.Success(c, backups)
}
