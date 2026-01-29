package system

import (
	"github.com/gin-gonic/gin"
	systemServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/system"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
)

// SetupFileRouter 设置文件管理路由
func SetupFileRouter(r *gin.RouterGroup, core *core.Core) {
	// 创建文件服务（注入依赖）
	fileService := systemServices.NewFileService(
		core.DB.GetDB(),
	)

	// 创建Handler
	handler := NewFileHandler(fileService)

	// 注册路由
	r.POST("/upload", handler.Upload)
	r.POST("/batch-delete", handler.BatchDelete)
	r.GET("", handler.List)
	r.GET("/:id", handler.GetByID)
	r.DELETE("/:id", handler.Delete)
}
