package system

import (
	"github.com/gin-gonic/gin"
	systemServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/system"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
)

// SetupRoleRouter 设置角色路由
func SetupRoleRouter(r *gin.RouterGroup, core *core.Core) {
	// 创建角色服务（注入依赖）
	roleService := systemServices.NewRoleService(
		core.DB.GetDB(),
	)

	// 创建Handler
	handler := NewRoleHandler(roleService)

	// 注册路由
	r.POST("/list", handler.List)
	r.POST("/all", handler.GetAllEnabled)
	r.POST("", handler.Create)
	r.POST("/:id", handler.GetByID)
	r.POST("/:id/update", handler.Update)
	r.POST("/:id/delete", handler.Delete)
	r.POST("/:id/status", handler.UpdateStatus)
	r.POST("/batch", handler.BatchDelete)
}
