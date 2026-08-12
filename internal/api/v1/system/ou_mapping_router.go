package system

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/services/addomain"
)

// SetupOUMappingRouter 设置 OU 映射路由
func SetupOUMappingRouter(r *gin.RouterGroup, core *core.Core) {
	// 创建 DeptOUmapper 和 OUMappingHandler
	mapper := addomain.NewDeptOUmapper(core.DB.GetDB())
	handler := NewOUMappingHandler(core.DB.GetDB(), mapper).WithCore(core)

	// OU 部门映射路由
	r.GET("/ou/:ouDn/dept-mapping", handler.GetOUDeptMapping)
	r.POST("/ou/:ouDn/dept-mapping", handler.UpdateOUDeptMapping)
}
