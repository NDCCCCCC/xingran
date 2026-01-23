package system

import (
	"github.com/gin-gonic/gin"
	systemServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/system"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
)

// SetupDepartmentRouter 设置部门路由
func SetupDepartmentRouter(r *gin.RouterGroup, core *core.Core) {
	// 创建部门服务（注入依赖）
	departmentService := systemServices.NewDepartmentService(
		core.DB.GetDB(),
	)

	// 创建Handler
	handler := NewDepartmentHandler(departmentService)

	// 注册路由
	r.POST("/tree", handler.GetTree)                       // 获取部门树
	r.POST("/list", handler.List)                          // 查询部门列表
	r.POST("", handler.Create)                             // 创建部门
	r.POST("/:id", handler.GetByID)                        // 获取部门详情
	r.POST("/:id/update", handler.Update)                  // 更新部门
	r.POST("/:id/delete", handler.Delete)                  // 删除部门
	r.POST("/:id/status", handler.UpdateStatus)            // 更新部门状态
	r.POST("/batch", handler.BatchDelete)                  // 批量删除部门
	r.POST("/tree-select", handler.GetTree)                // 获取部门树选择器
	r.POST("/role-dept-tree-select/:roleId", handler.RoleDeptTreeSelect) // 获取角色部门树选择器
}
