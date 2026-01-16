package v1

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
	"gorm.io/gorm"
)

// updateUserStatus 更新用户状态
// @Summary 更新用户状态
// @Description 批量启用或禁用用户
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param request body UserStatusRequest true "状态信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /users/status [put]
func updateUserStatus(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req UserStatusRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		// 检查用户是否存在
		var user models.User
		if err := core.DB.GetDB().First(&user, "id = ?", req.ID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				response.Error(c, response.ErrRecordNotFound, "用户不存在")
			} else {
				response.Error(c, response.ErrServerError, "查询用户失败")
			}
			return
		}

		// 更新状态
		user.Status = models.UserStatus(req.Status)

		if err := core.DB.GetDB().Save(&user).Error; err != nil {
			response.Error(c, response.ErrServerError, "更新用户状态失败")
			return
		}

		statusText := "启用"
		if req.Status == 1 {
			statusText = "禁用"
		}

		// 记录操作日志
		recordOperLog(c, core, "用户管理", OperTypeUpdate)

		response.Success(c, gin.H{
			"message": fmt.Sprintf("%s成功", statusText),
		})
	}
}

// resetPassword 重置密码
// @Summary 重置用户密码
// @Description 管理员重置指定用户的密码
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param id path string true "用户ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /users/{id}/reset-password [put]
func resetPassword(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.Param("id")
		if userID == "" {
			response.Error(c, response.ErrBadRequest, "用户ID不能为空")
			return
		}

		// 检查用户是否存在
		var user models.User
		if err := core.DB.GetDB().First(&user, "id = ?", userID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				response.Error(c, response.ErrRecordNotFound, "用户不存在")
			} else {
				response.Error(c, response.ErrServerError, "查询用户失败")
			}
			return
		}

		// 生成新密码（默认密码：123456）
		pwdManager := core.PwdManager
		hashedPassword, err := pwdManager.HashPassword("123456")
		if err != nil {
			response.Error(c, response.ErrServerError, "密码加密失败")
			return
		}

		// 更新密码
		user.Password = hashedPassword

		if err := core.DB.GetDB().Save(&user).Error; err != nil {
			response.Error(c, response.ErrServerError, "重置密码失败")
			return
		}

		// 记录操作日志
		recordOperLog(c, core, "用户管理", OperTypeUpdate)

		response.Success(c, gin.H{
			"message": "重置成功，新密码为：123456",
		})
	}
}
