package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
)

// SetupOperLogRouter 设置操作日志路由
func SetupOperLogRouter(r *gin.RouterGroup, core *core.Core) {
	// 查询操作日志列表
	r.POST("/list", listOperLogs(core))
	// 获取操作日志详情
	r.POST("/:id", getOperLog(core))
	// 删除操作日志
	r.POST("/:id/delete", deleteOperLog(core))
	// 批量删除操作日志
	r.POST("/batch-delete", batchDeleteOperLogs(core))
	// 清空操作日志
	r.POST("/clean", cleanOperLogs(core))
}

// SetupLoginLogRouter 设置登录日志路由
func SetupLoginLogRouter(r *gin.RouterGroup, core *core.Core) {
	// 查询登录日志列表
	r.POST("/list", listLoginLogs(core))
	// 获取登录日志详情
	r.POST("/:id", getLoginLog(core))
	// 删除登录日志
	r.POST("/:id/delete", deleteLoginLog(core))
	// 批量删除登录日志
	r.POST("/batch-delete", batchDeleteLoginLogs(core))
	// 清空登录日志
	r.POST("/clean", cleanLoginLogs(core))
	// 解锁用户
	r.POST("/unlock/:username", unlockUser(core))
}
