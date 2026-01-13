package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
)

// SetupWorkstationRouter 设置工位路由
func SetupWorkstationRouter(r *gin.RouterGroup, core *core.Core) {
	// 查询工位列表
	r.POST("/list", listWorkstations(core))
	// 获取可用工位列表
	r.GET("/available", getAvailableWorkstations(core))
	// 创建工位
	r.POST("", createWorkstation(core))
	// 获取工位详情
	r.POST("/:id", getWorkstation(core))
	// 更新工位
	r.POST("/:id/update", updateWorkstation(core))
	// 删除工位
	r.POST("/:id/delete", deleteWorkstation(core))
	// 批量删除工位
	r.POST("/batch-delete", batchDeleteWorkstations(core))
	// 分配用户到工位
	r.POST("/:id/assign", assignUser(core))
	// 取消工位用户分配
	r.POST("/:id/unassign", unassignUser(core))
}
