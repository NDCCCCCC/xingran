package system

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/pkg/middleware"
)

// permBrandingRead 品牌配置读权限（复用系统配置权限族，不新增权限码）
var permBrandingRead = []string{"system:config:list", "system:config:view"}

// permBrandingEdit 品牌配置写权限（strict 单码）
var permBrandingEdit = []string{"system:config:edit"}

// SetupBrandingPublicRouter 公开品牌路由（免认证；登录页渲染品牌信息用）
//
// GET /system/branding/public  聚合品牌配置（CacheProvider 缓存 5 分钟）
// GET /system/branding/logo    当前 Logo 文件（无则 404，前端回退方块字）
func SetupBrandingPublicRouter(r *gin.RouterGroup, core *core.Core) {
	handler := NewBrandingHandler(newBrandingServiceFromCore(core)).WithCore(core)
	r.GET("/public", handler.GetPublicBranding)
	r.GET("/logo", handler.ServeLogo)
}

// SetupBrandingRouter 品牌管理路由（需认证 + 权限）
//
// GET  /system/branding         读全部品牌配置（ANY-of config:list/view）
// POST /system/branding/update  批量更新（strict config:edit）
// POST /system/branding/logo    上传 Logo（strict config:edit）
func SetupBrandingRouter(r *gin.RouterGroup, core *core.Core) {
	handler := NewBrandingHandler(newBrandingServiceFromCore(core)).WithCore(core)
	r.GET("", middleware.RequirePermissions(permBrandingRead, core), handler.GetBranding)
	r.POST("/update", middleware.RequireStrictPermissions(permBrandingEdit, core), handler.UpdateBranding)
	r.POST("/logo", middleware.RequireStrictPermissions(permBrandingEdit, core), handler.UploadLogo)
}
