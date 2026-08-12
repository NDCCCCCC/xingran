package system

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/constants"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
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
		lockKey := fmt.Sprintf(constants.LoginLockKeyFormat, req.Username)
		_ = core.Cache.Delete(c.Request.Context(), lockKey)

		// 记录操作日志（合规敏感：账号解锁需审计 who-unlocked-whom）。
		// 使用专用 OperTypeUnlock(24) 而非 OperTypeOther(0)：解锁是独立的审计动词，
		// 不属于状态变更语义（不修改 sys_user.status），而是清除登录锁定缓存。
		// oper_param 记录被解锁的用户名，操作者由 operlog 自动提取。
		operlog.Record(c, core.OperLogService, core.GetDB(), "用户解锁", operlog.OperTypeUnlock,
			operlog.WithOperParam("username="+req.Username))
		response.Success(c, gin.H{
			"message": "用户账号已解锁",
		})
	}
}
