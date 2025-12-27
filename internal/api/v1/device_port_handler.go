package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/device"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// PortStatusListRequest 端口状态列表请求
type PortStatusListRequest struct {
	Current       int    `json:"current" binding:"min=1"`
	PageSize      int    `json:"pageSize" binding:"min=1,max=100"`
	DeviceID      string `json:"deviceId,omitempty"`
	InterfaceName string `json:"interfaceName,omitempty"`
	Dot1xEnabled  *bool  `json:"dot1xEnabled,omitempty"`
}

// PortCollectionRequest 端口采集请求
type PortCollectionRequest struct {
	DeviceID string `json:"deviceId" binding:"required"`
}

// listPortStatus 查询端口状态列表
func listPortStatus(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req PortStatusListRequest
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

		manager := device.NewManager(core.DB.GetDB(), core.SM4Cipher, device.DefaultManagerConfig())
		defer manager.Close()

		svc := services.NewPortCollectionService(core.DB.GetDB(), manager)
		portStatuses, total, err := svc.GetPortStatusList(c.Request.Context(), current, pageSize, req.DeviceID, req.InterfaceName, req.Dot1xEnabled)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		pageResp := response.PageResponse{
			List:     portStatuses,
			Total:    total,
			Current:  current,
			PageSize: pageSize,
		}

		response.Success(c, pageResp)
	}
}

// collectPortStatus 采集单个设备端口状态
func collectPortStatus(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req PortCollectionRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		manager := device.NewManager(core.DB.GetDB(), core.SM4Cipher, device.DefaultManagerConfig())
		defer manager.Close()

		svc := services.NewPortCollectionService(core.DB.GetDB(), manager)
		result, err := svc.CollectDevice(c.Request.Context(), req.DeviceID)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, result)
	}
}

// collectAllPortStatus 采集所有设备端口状态
func collectAllPortStatus(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		manager := device.NewManager(core.DB.GetDB(), core.SM4Cipher, device.DefaultManagerConfig())
		defer manager.Close()

		svc := services.NewPortCollectionService(core.DB.GetDB(), manager)
		results, err := svc.CollectAllDevices(c.Request.Context())
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, results)
	}
}

// getPortStatusStats 获取端口状态统计信息
func getPortStatusStats(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		manager := device.NewManager(core.DB.GetDB(), core.SM4Cipher, device.DefaultManagerConfig())
		defer manager.Close()

		svc := services.NewPortCollectionService(core.DB.GetDB(), manager)
		stats, err := svc.GetPortStatusStats(c.Request.Context())
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, stats)
	}
}

// CleanOldPortRecordsRequest 清理旧记录请求
type CleanOldPortRecordsRequest struct {
	Days int `json:"days" binding:"required,min=1,max=365"`
}

// BatchDeletePortRecordsRequest 批量删除请求
type BatchDeletePortRecordsRequest struct {
	IDs []string `json:"ids" binding:"required,min=1"`
}

// batchDeletePortRecords 批量删除端口状态记录
func batchDeletePortRecords(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req BatchDeletePortRecordsRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误: "+err.Error())
			return
		}

		if len(req.IDs) == 0 {
			response.Error(c, response.ErrBadRequest, "请选择要删除的数据")
			return
		}

		// 使用 GORM 批量删除
		if err := core.DB.GetDB().Delete(&models.DevicePortStatus{}, req.IDs).Error; err != nil {
			response.Error(c, response.ErrServerError, "批量删除失败: "+err.Error())
			return
		}

		response.Success(c, map[string]interface{}{
			"deletedCount": len(req.IDs),
		})
	}
}

// cleanOldPortRecords 清理旧的端口状态记录
func cleanOldPortRecords(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CleanOldPortRecordsRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		manager := device.NewManager(core.DB.GetDB(), core.SM4Cipher, device.DefaultManagerConfig())
		defer manager.Close()

		svc := services.NewPortCollectionService(core.DB.GetDB(), manager)
		count, err := svc.CleanOldRecords(c.Request.Context(), req.Days)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, map[string]interface{}{
			"deletedCount": count,
		})
	}
}

// SetupPortStatusRouter 设置端口状态路由
func SetupPortStatusRouter(r *gin.RouterGroup, core *core.Core) {
	// 端口状态列表
	r.POST("/list", listPortStatus(core))
	// 采集单个设备端口状态
	r.POST("/collect", collectPortStatus(core))
	// 采集所有设备端口状态
	r.POST("/collect-all", collectAllPortStatus(core))
	// 获取端口状态统计
	r.GET("/statistics", getPortStatusStats(core))
	// 清理旧记录
	r.POST("/clean", cleanOldPortRecords(core))
	// 批量删除
	r.POST("/batch-delete", batchDeletePortRecords(core))
}
