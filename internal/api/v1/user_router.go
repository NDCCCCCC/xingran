package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
)

// SetupUserRouter 设置用户路由
func SetupUserRouter(r *gin.RouterGroup, core *core.Core) {
	// 查询用户列表
	r.POST("/list", listUsers(core))
	// 创建用户
	r.POST("", createUser(core))
	// 获取用户详情
	r.POST("/:id", getUser(core))
	// 更新用户
	r.POST("/:id/update", updateUser(core))
	// 删除用户
	r.POST("/:id/delete", deleteUser(core))
	// 批量删除用户
	r.POST("/batch-delete", batchDeleteUsers(core))
	// 更改用户状态
	r.POST("/:id/status", updateUserStatus(core))
	// 重置密码
	r.POST("/:id/reset-password", resetPassword(core))
}