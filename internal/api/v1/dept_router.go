package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
)

// SetupDeptRouter 设置部门路由
func SetupDeptRouter(r *gin.RouterGroup, core *core.Core) {
	// 查询部门列表
	r.POST("/list", listDepts(core))
	// 获取部门树
	r.GET("/tree", getDeptTree(core))
	// 创建部门
	r.POST("", createDept(core))
	// 获取部门详情
	r.POST("/:id", getDept(core))
	// 更新部门
	r.POST("/:id/update", updateDept(core))
	// 删除部门
	r.POST("/:id/delete", deleteDept(core))
	// 获取部门树选择器
	r.POST("/tree-select", getDeptTreeSelect(core))
	// 获取角色部门树选择器
	r.POST("/role-dept-tree-select/:roleId", getRoleDeptTreeSelect(core))
	// 获取部门用户列表
	r.GET("/:deptId/users", getDeptUsers(core))
}
