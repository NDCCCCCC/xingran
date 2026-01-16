package v1

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// BackupCreateRequest 创建备份请求
type BackupCreateRequest struct {
	DeviceID      string            `json:"deviceId" binding:"required"`
	BackupType    models.BackupType `json:"backupType" binding:"required,oneof=auto manual"`
	ChangeReason  string            `json:"changeReason"`
	CompressLarge bool              `json:"compressLarge"`
}

// BackupListRequest 备份列表请求
type BackupListRequest struct {
	Current  int    `json:"current"`
	PageSize int    `json:"pageSize"`
	DeviceID string `json:"deviceId"`
}

// DiffBackupsRequest 对比备份请求
type DiffBackupsRequest struct {
	BackupID1 string `json:"backupId1" binding:"required"`
	BackupID2 string `json:"backupId2" binding:"required"`
}

// RestoreBackupRequest 恢复配置请求
type RestoreBackupRequest struct {
	BackupID string `json:"backupId" binding:"required"`
	DeviceID string `json:"deviceId" binding:"required"`
}

// listConfigBackups 查询备份列表
func listConfigBackups(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req BackupListRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		current := req.Current
		if current <= 0 {
			current = 1
		}
		pageSize := req.PageSize
		if pageSize <= 0 {
			pageSize = 10
		}

		// 使用核心设备管理器（新架构）
		// manager := device.NewManager(core.DB.GetDB(), core.SM4Cipher, device.DefaultManagerConfig())
		// defer manager.Close()

		svc := services.NewConfigBackupService(core.DB.GetDB(), core.DeviceExecutor)
		backups, total, err := svc.GetBackupList(c.Request.Context(), current, pageSize, req.DeviceID)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
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
}

// createConfigBackup 创建备份
func createConfigBackup(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req BackupCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		// 查询设备名称
		var netDevice models.NetworkDevice
		if err := core.DB.GetDB().Where("id = ?", req.DeviceID).First(&netDevice).Error; err != nil {
			response.Error(c, response.ErrNotFound, "设备不存在")
			return
		}

		// 使用核心设备管理器（新架构）
		// manager := device.NewManager(core.DB.GetDB(), core.SM4Cipher, device.DefaultManagerConfig())
		// defer manager.Close()

		svc := services.NewConfigBackupService(core.DB.GetDB(), core.DeviceExecutor)
		result, err := svc.CreateBackup(c.Request.Context(), &services.BackupRequest{
			DeviceID:      req.DeviceID,
			DeviceName:    netDevice.DeviceName,
			BackupType:    req.BackupType,
			ChangeReason:  req.ChangeReason,
			CreatedBy:     c.GetString("userId"),
			CompressLarge: req.CompressLarge,
		})
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, result)
	}
}

// getBackupContent 获取备份内容
func getBackupContent(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		// 使用核心设备管理器（新架构）
		// manager := device.NewManager(core.DB.GetDB(), core.SM4Cipher, device.DefaultManagerConfig())
		// defer manager.Close()

		svc := services.NewConfigBackupService(core.DB.GetDB(), core.DeviceExecutor)
		content, err := svc.GetBackupContent(c.Request.Context(), id)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{"content": content})
	}
}

// deleteConfigBackup 删除备份
func deleteConfigBackup(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		// 使用核心设备管理器（新架构）
		// manager := device.NewManager(core.DB.GetDB(), core.SM4Cipher, device.DefaultManagerConfig())
		// defer manager.Close()

		svc := services.NewConfigBackupService(core.DB.GetDB(), core.DeviceExecutor)
		if err := svc.DeleteBackup(c.Request.Context(), id); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, nil)
	}
}

// batchDeleteConfigBackups 批量删除备份
func batchDeleteConfigBackups(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			BackupIDs []string `json:"backupIds" binding:"required,min=1"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		// 使用核心设备管理器（新架构）
		// manager := device.NewManager(core.DB.GetDB(), core.SM4Cipher, device.DefaultManagerConfig())
		// defer manager.Close()

		svc := services.NewConfigBackupService(core.DB.GetDB(), core.DeviceExecutor)
		if err := svc.BatchDeleteBackups(c.Request.Context(), req.BackupIDs); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, nil)
	}
}

// diffConfigBackups 对比备份配置
func diffConfigBackups(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req DiffBackupsRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		// 使用核心设备管理器（新架构）
		// manager := device.NewManager(core.DB.GetDB(), core.SM4Cipher, device.DefaultManagerConfig())
		// defer manager.Close()

		svc := services.NewConfigBackupService(core.DB.GetDB(), core.DeviceExecutor)
		name1, name2, diff, err := svc.DiffBackups(c.Request.Context(), req.BackupID1, req.BackupID2)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{
			"name1": name1,
			"name2": name2,
			"diff":  diff,
		})
	}
}

// restoreConfigBackup 恢复配置
func restoreConfigBackup(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req RestoreBackupRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		// 使用核心设备管理器（新架构）
		// manager := device.NewManager(core.DB.GetDB(), core.SM4Cipher, device.DefaultManagerConfig())
		// defer manager.Close()

		svc := services.NewConfigBackupService(core.DB.GetDB(), core.DeviceExecutor)
		if err := svc.RestoreBackup(c.Request.Context(), req.BackupID, req.DeviceID); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, nil)
	}
}

// getBackupStatistics 获取备份统计信息
func getBackupStatistics(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 使用核心设备管理器（新架构）
		// manager := device.NewManager(core.DB.GetDB(), core.SM4Cipher, device.DefaultManagerConfig())
		// defer manager.Close()

		svc := services.NewConfigBackupService(core.DB.GetDB(), core.DeviceExecutor)
		stats, err := svc.GetBackupStatistics(c.Request.Context())
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, stats)
	}
}

// batchBackupDevices 批量备份设备
func batchBackupDevices(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			DeviceIDs  []string          `json:"deviceIds" binding:"required,min=1"`
			BackupType models.BackupType `json:"backupType" binding:"required,oneof=auto manual"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		// 使用核心设备管理器（新架构）
		// manager := device.NewManager(core.DB.GetDB(), core.SM4Cipher, device.DefaultManagerConfig())
		// defer manager.Close()

		svc := services.NewConfigBackupService(core.DB.GetDB(), core.DeviceExecutor)
		results, err := svc.BatchBackupDevices(c.Request.Context(), req.DeviceIDs, req.BackupType, c.GetString("userId"))
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, results)
	}
}

// getBackupByVersion 获取指定版本的备份
func getBackupByVersion(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		deviceID := c.Query("deviceId")
		versionStr := c.Query("version")

		if deviceID == "" || versionStr == "" {
			response.Error(c, response.ErrBadRequest, "设备ID和版本号不能为空")
			return
		}

		version, err := strconv.Atoi(versionStr)
		if err != nil {
			response.Error(c, response.ErrBadRequest, "版本号格式错误")
			return
		}

		var backup models.ConfigBackup
		if err := core.DB.GetDB().Where("device_id = ? AND version = ?", deviceID, version).First(&backup).Error; err != nil {
			response.Error(c, response.ErrNotFound, "备份记录不存在")
			return
		}

		response.Success(c, backup)
	}
}

// getDeviceBackupHistory 获取设备备份历史
func getDeviceBackupHistory(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		deviceID := c.Query("deviceId")
		if deviceID == "" {
			response.Error(c, response.ErrBadRequest, "设备ID不能为空")
			return
		}

		var backups []models.ConfigBackup
		if err := core.DB.GetDB().Where("device_id = ?", deviceID).
			Order("version DESC").
			Find(&backups).Error; err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, backups)
	}
}

// SetupConfigBackupRouter 设置配置备份路由
func SetupConfigBackupRouter(r *gin.RouterGroup, core *core.Core) {
	// 备份列表
	r.POST("/list", listConfigBackups(core))
	// 创建备份
	r.POST("", createConfigBackup(core))
	// 获取备份详情
	r.POST("/:id", getBackupContent(core))
	// 删除备份
	r.POST("/:id/delete", deleteConfigBackup(core))
	// 批量删除备份
	r.POST("/batch-delete", batchDeleteConfigBackups(core))
	// 对比备份配置
	r.POST("/diff", diffConfigBackups(core))
	// 恢复配置
	r.POST("/:id/restore", restoreConfigBackup(core))
	// 获取备份统计
	r.GET("/statistics", getBackupStatistics(core))
	// 批量备份设备
	r.POST("/batch", batchBackupDevices(core))
	// 获取指定版本备份
	r.GET("/version", getBackupByVersion(core))
	// 获取设备备份历史
	r.GET("/history", getDeviceBackupHistory(core))
}
