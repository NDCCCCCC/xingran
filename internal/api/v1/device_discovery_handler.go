package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// DiscoveryCreateRequest 创建发现任务请求
type DiscoveryCreateRequest struct {
	TaskName      string              `json:"taskName" binding:"required,min=2,max=200"`
	DiscoveryType models.DiscoveryType `json:"discoveryType" binding:"required,oneof=snmp scan"`
	IPRanges      []models.IPRange    `json:"ipRanges" binding:"required,min=1"`
	SNMPCommunity string              `json:"snmpCommunity"`
	SNMPPort      int                 `json:"snmpPort" binding:"omitempty,min=1,max=65535"`
	AutoImport    bool                `json:"autoImport"`
	GroupID       *string             `json:"groupId"`
}

// DiscoveryListRequest 发现任务列表请求
type DiscoveryListRequest struct {
	Current  int    `json:"current"`
	PageSize int    `json:"pageSize"`
}

// ExecuteDiscoveryRequest 执行发现任务请求
type ExecuteDiscoveryRequest struct {
	DiscoveryID string `json:"discoveryId" binding:"required"`
}

// ImportDevicesRequest 导入设备请求
type ImportDevicesRequest struct {
	DiscoveryID string   `json:"discoveryId" binding:"required"`
	DeviceIDs   []string `json:"deviceIds" binding:"required,min=1"`
}

// listDeviceDiscoveries 查询发现任务列表
func listDeviceDiscoveries(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req DiscoveryListRequest
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

		svc := services.NewDeviceDiscoveryService(core.DB.GetDB())
		tasks, total, err := svc.GetDiscoveryList(c.Request.Context(), current, pageSize)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		pageResp := response.PageResponse{
			List:     tasks,
			Total:    total,
			Current:  current,
			PageSize: pageSize,
		}

		response.Success(c, pageResp)
	}
}

// createDeviceDiscovery 创建发现任务
func createDeviceDiscovery(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req DiscoveryCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		// 设置默认SNMP端口
		if req.SNMPPort == 0 {
			req.SNMPPort = 161
		}

		svc := services.NewDeviceDiscoveryService(core.DB.GetDB())
		discoveryID, err := svc.CreateDiscoveryTask(c.Request.Context(), &services.DiscoveryRequest{
			TaskName:      req.TaskName,
			DiscoveryType: req.DiscoveryType,
			IPRanges:      req.IPRanges,
			SNMPCommunity: req.SNMPCommunity,
			SNMPPort:      req.SNMPPort,
			AutoImport:    req.AutoImport,
			GroupID:       req.GroupID,
			CreatedBy:     c.GetString("userId"),
		})
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{"discoveryId": discoveryID})
	}
}

// getDeviceDiscovery 获取发现任务详情
func getDeviceDiscovery(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		svc := services.NewDeviceDiscoveryService(core.DB.GetDB())
		task, err := svc.GetDiscoveryByID(c.Request.Context(), id)
		if err != nil {
			response.Error(c, response.ErrNotFound, err.Error())
			return
		}

		response.Success(c, task)
	}
}

// executeDeviceDiscovery 执行发现任务
func executeDeviceDiscovery(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		svc := services.NewDeviceDiscoveryService(core.DB.GetDB())
		result, err := svc.ExecuteDiscovery(c.Request.Context(), id)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, result)
	}
}

// cancelDeviceDiscovery 取消发现任务
func cancelDeviceDiscovery(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		svc := services.NewDeviceDiscoveryService(core.DB.GetDB())
		if err := svc.CancelDiscovery(c.Request.Context(), id); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, nil)
	}
}

// deleteDeviceDiscovery 删除发现任务
func deleteDeviceDiscovery(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		svc := services.NewDeviceDiscoveryService(core.DB.GetDB())
		if err := svc.DeleteDiscovery(c.Request.Context(), id); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, nil)
	}
}

// batchDeleteDeviceDiscoveries 批量删除发现任务
func batchDeleteDeviceDiscoveries(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			DiscoveryIDs []string `json:"discoveryIds" binding:"required,min=1"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		svc := services.NewDeviceDiscoveryService(core.DB.GetDB())
		if err := svc.BatchDeleteDiscoveries(c.Request.Context(), req.DiscoveryIDs); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, nil)
	}
}

// importDiscoveredDevices 导入发现的设备
func importDiscoveredDevices(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		svc := services.NewDeviceDiscoveryService(core.DB.GetDB())
		count, err := svc.ImportDiscoveredDevices(c.Request.Context(), id, nil, c.GetString("userId"))
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{"count": count})
	}
}

// getDiscoveryResults 获取发现结果
func getDiscoveryResults(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		svc := services.NewDeviceDiscoveryService(core.DB.GetDB())
		devices, err := svc.GetDiscoveryResults(c.Request.Context(), id)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{"devices": devices})
	}
}

// SetupDeviceDiscoveryRouter 设置设备发现路由
func SetupDeviceDiscoveryRouter(r *gin.RouterGroup, core *core.Core) {
	// 发现任务列表
	r.POST("/list", listDeviceDiscoveries(core))
	// 创建发现任务
	r.POST("/create", createDeviceDiscovery(core))
	// 获取发现任务详情
	r.POST("/:id", getDeviceDiscovery(core))
	// 获取发现结果
	r.POST("/:id/results", getDiscoveryResults(core))
	// 执行发现任务
	r.POST("/:id/execute", executeDeviceDiscovery(core))
	// 取消发现任务
	r.POST("/:id/cancel", cancelDeviceDiscovery(core))
	// 删除发现任务
	r.POST("/:id/delete", deleteDeviceDiscovery(core))
	// 批量删除发现任务
	r.POST("/batch-delete", batchDeleteDeviceDiscoveries(core))
	// 导入发现的设备
	r.POST("/:id/import", importDiscoveredDevices(core))
}
