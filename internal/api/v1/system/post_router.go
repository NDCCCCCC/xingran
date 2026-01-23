package system

import (
	"github.com/gin-gonic/gin"
	systemServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/system"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
)

// SetupPostRouter 设置岗位路由
func SetupPostRouter(r *gin.RouterGroup, core *core.Core) {
	// 创建岗位服务（注入依赖）
	postService := systemServices.NewPostService(
		core.DB.GetDB(),
	)

	// 创建Handler
	handler := NewPostHandler(postService)

	// 注册路由
	r.POST("/list", handler.List)
	r.POST("", handler.Create)
	r.POST("/:id", handler.GetByID)
	r.POST("/:id/update", handler.Update)
	r.POST("/:id/delete", handler.Delete)
	r.POST("/batch-delete", handler.BatchDelete)
}
