package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// SetupProfileRouter 设置个人信息路由
func SetupProfileRouter(r *gin.RouterGroup, core *core.Core) {
	// 获取当前用户个人信息
	r.GET("/info", getProfileInfo(core))
	// 更新个人信息
	r.PUT("/info", updateProfileInfo(core))
	// 修改密码
	r.POST("/change-password", changePassword(core))
	// 上传头像
	r.POST("/avatar", uploadAvatar(core))
}

// ProfileInfoRequest 个人信息更新请求
type ProfileInfoRequest struct {
	Nickname *string `json:"nickname,omitempty" binding:"omitempty,max=64"`
	Email    *string `json:"email,omitempty" binding:"omitempty,email"`
	Phone    *string `json:"phone,omitempty" binding:"omitempty,max=32"`
	Gender   int     `json:"gender" binding:"min=0,max=2"`
	Remark   *string `json:"remark,omitempty" binding:"omitempty,max=500"`
}

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=6,max=20"`
}

// getProfileInfo 获取当前用户个人信息
func getProfileInfo(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从 JWT 中获取用户 ID
		userID, exists := c.Get("user_id")
		if !exists {
			response.Error(c, response.ErrUnauthorized)
			return
		}

		// 调用 service 层
		profileService := services.NewProfileService(core.DB.GetDB())
		userInfo, err := profileService.GetUserInfo(c.Request.Context(), userID.(string))
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, userInfo)
	}
}

// updateProfileInfo 更新个人信息
func updateProfileInfo(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			response.Error(c, response.ErrUnauthorized)
			return
		}

		var req ProfileInfoRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		profileService := services.NewProfileService(core.DB.GetDB())
		updateReq := &services.UpdateUserInfoRequest{
			Nickname: req.Nickname,
			Email:    req.Email,
			Phone:    req.Phone,
			Gender:   req.Gender,
			Remark:   req.Remark,
		}

		if err := profileService.UpdateUserInfo(c.Request.Context(), userID.(string), updateReq); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{"message": "更新成功"})
	}
}

// changePassword 修改密码
func changePassword(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			response.Error(c, response.ErrUnauthorized)
			return
		}

		var req ChangePasswordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		profileService := services.NewProfileService(core.DB.GetDB())
		if err := profileService.ChangePassword(c.Request.Context(), userID.(string), req.OldPassword, req.NewPassword); err != nil {
			// 根据错误类型返回不同的错误码
			if err.Error() == "旧密码错误" {
				response.Error(c, response.ErrCredentialInvalid, "旧密码错误")
			} else {
				response.Error(c, response.ErrServerError, err.Error())
			}
			return
		}

		response.Success(c, gin.H{"message": "密码修改成功"})
	}
}

// uploadAvatar 上传头像
func uploadAvatar(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			response.Error(c, response.ErrUnauthorized)
			return
		}

		// 解析表单，获取文件
		file, err := c.FormFile("file")
		if err != nil {
			response.Error(c, response.ErrBadRequest, "文件上传失败")
			return
		}

		// 验证文件大小（最大 2MB）
		if file.Size > 2*1024*1024 {
			response.Error(c, response.ErrBadRequest, "文件大小不能超过 2MB")
			return
		}

		profileService := services.NewProfileService(core.DB.GetDB())
		avatarURL, err := profileService.UploadAvatar(c.Request.Context(), userID.(string), file)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{
			"avatar":  avatarURL,
			"message": "头像上传成功",
		})
	}
}
