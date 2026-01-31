package system

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// UnlockUserRequest 解锁用户请求
type UnlockUserRequest struct {
	Username string `json:"username" binding:"required"`
}

// SetupUserUnlockRouter 设置用户解锁路由
func SetupUserUnlockRouter(r *gin.RouterGroup, core *core.Core) {
	r.POST("/unlock", unlockUser(core))
}

// unlockUser 解锁用户账号
// @Summary 解锁用户账号
// @Description 管理员手动解锁被锁定的用户账号
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body UnlockUserRequest true "解锁请求"
// @Success 200 {object} response.Response
// @Router /system/user/unlock [post]
func unlockUser(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req UnlockUserRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		// 清除登录失败记录
		core.CaptchaService.ClearLoginFailure(c.Request.Context(), req.Username)

		// 清除锁定状态
		// 锁定状态使用 key: login:lock:{username}
		lockKey := "login:lock:" + req.Username
		_ = core.Cache.Delete(c.Request.Context(), lockKey)

		response.Success(c, gin.H{
			"message": "用户账号已解锁",
		})
	}
}
