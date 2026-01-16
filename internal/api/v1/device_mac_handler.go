package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// MACAddressListRequest MAC地址列表请求
type MACAddressListRequest struct {
	Current       int    `json:"current" binding:"min=1"`
	PageSize      int    `json:"pageSize" binding:"min=1,max=100"`
	DeviceID      string `json:"deviceId,omitempty"`
	MACAddress    string `json:"macAddress,omitempty"`
	InterfaceName string `json:"interfaceName,omitempty"`
}

// MACCollectionRequest MAC采集请求
type MACCollectionRequest struct {
	DeviceID string `json:"deviceId" binding:"required"`
}

// listMACAddresses 查询MAC地址列表
func listMACAddresses(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req MACAddressListRequest
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

		svc := services.NewMACCollectionService(core.DB.GetDB(), core.DeviceExecutor)
		macAddrs, total, err := svc.GetMACAddressList(c.Request.Context(), current, pageSize, req.DeviceID, req.MACAddress, req.InterfaceName)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		pageResp := response.PageResponse{
			List:     macAddrs,
			Total:    total,
			Current:  current,
			PageSize: pageSize,
		}

		response.Success(c, pageResp)
	}
}

// collectMACAddress 采集单个设备MAC地址
func collectMACAddress(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req MACCollectionRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		// 使用核心设备管理器（新架构）
		// manager := device.NewManager(core.DB.GetDB(), core.SM4Cipher, device.DefaultManagerConfig())
		// defer manager.Close()

		svc := services.NewMACCollectionService(core.DB.GetDB(), core.DeviceExecutor)
		result, err := svc.CollectDevice(c.Request.Context(), req.DeviceID)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, result)
	}
}

// collectAllMACAddresses 采集所有设备MAC地址
func collectAllMACAddresses(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 使用核心设备管理器（新架构）
		// manager := device.NewManager(core.DB.GetDB(), core.SM4Cipher, device.DefaultManagerConfig())
		// defer manager.Close()

		svc := services.NewMACCollectionService(core.DB.GetDB(), core.DeviceExecutor)
		results, err := svc.CollectAllDevices(c.Request.Context())
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, results)
	}
}

// getMACAddressStats 获取MAC地址统计信息
func getMACAddressStats(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 使用核心设备管理器（新架构）
		// manager := device.NewManager(core.DB.GetDB(), core.SM4Cipher, device.DefaultManagerConfig())
		// defer manager.Close()

		svc := services.NewMACCollectionService(core.DB.GetDB(), core.DeviceExecutor)
		stats, err := svc.GetMACAddressStats(c.Request.Context())
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, stats)
	}
}

// CleanOldMACRecordsRequest 清理旧记录请求
type CleanOldMACRecordsRequest struct {
	Days int `json:"days" binding:"required,min=1,max=365"`
}

// cleanOldMACRecords 清理旧的MAC地址记录
func cleanOldMACRecords(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CleanOldMACRecordsRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		// 使用核心设备管理器（新架构）
		// manager := device.NewManager(core.DB.GetDB(), core.SM4Cipher, device.DefaultManagerConfig())
		// defer manager.Close()

		svc := services.NewMACCollectionService(core.DB.GetDB(), core.DeviceExecutor)
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

// SetupMACAddressRouter 设置MAC地址路由
func SetupMACAddressRouter(r *gin.RouterGroup, core *core.Core) {
	// MAC地址列表
	r.POST("/list", listMACAddresses(core))
	// 采集单个设备MAC地址
	r.POST("/collect", collectMACAddress(core))
	// 采集所有设备MAC地址
	r.POST("/collect/all", collectAllMACAddresses(core))
	// 获取MAC地址统计
	r.GET("/statistics", getMACAddressStats(core))
	// 清理旧记录
	r.POST("/clean", cleanOldMACRecords(core))
}
