package system

import (

	"github.com/gin-gonic/gin"
	systemServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/system"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
	apperrors "github.com/ruoyi-next/ruoyi-go-backend/pkg/errors"
)

// ProfileHandler 个人设置处理器
type ProfileHandler struct {
	service systemServices.ProfileService
}

// NewProfileHandler 创建个人设置处理器实例
func NewProfileHandler(service systemServices.ProfileService) *ProfileHandler {
	return &ProfileHandler{service: service}
}

// GetInfo 获取当前用户个人信息
func (h *ProfileHandler) GetInfo(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "未授权")
		return
	}

	userInfo, err := h.service.GetUserInfo(c.Request.Context(), userID.(string))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, userInfo)
}

// ProfileInfoRequest 个人信息更新请求
type ProfileInfoRequest struct {
	Nickname *string `json:"nickname,omitempty" binding:"omitempty,max=64"`
	Email    *string `json:"email,omitempty" binding:"omitempty,email"`
	Phone    *string `json:"phone,omitempty" binding:"omitempty,max=32"`
	Gender   int     `json:"gender" binding:"min=0,max=2"`
	Remark   *string `json:"remark,omitempty" binding:"omitempty,max=500"`
}

// UpdateInfo 更新个人信息
func (h *ProfileHandler) UpdateInfo(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "未授权")
		return
	}

	var req ProfileInfoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	updateReq := &systemServices.UpdateUserInfoRequest{
		Nickname: req.Nickname,
		Email:    req.Email,
		Phone:    req.Phone,
		Gender:   req.Gender,
		Remark:   req.Remark,
	}

	if err := h.service.UpdateUserInfo(c.Request.Context(), userID.(string), updateReq); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{"message": "更新成功"})
}

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=6,max=20"`
}

// ChangePassword 修改密码
func (h *ProfileHandler) ChangePassword(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "未授权")
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	if err := h.service.ChangePassword(c.Request.Context(), userID.(string), req.OldPassword, req.NewPassword); err != nil {
		// 根据错误类型返回不同的错误码
		if err.Error() == "旧密码错误" {
			response.Error(c, http.StatusBadRequest, "旧密码错误")
		} else {
			response.Error(c, err)
		}
		return
	}

	response.Success(c, gin.H{"message": "密码修改成功"})
}

// UploadAvatar 上传头像
func (h *ProfileHandler) UploadAvatar(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "未授权")
		return
	}

	// 解析表单，获取文件
	file, err := c.FormFile("file")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "文件上传失败")
		return
	}

	// 验证文件大小（最大 2MB）
	if file.Size > 2*1024*1024 {
		response.Error(c, http.StatusBadRequest, "文件大小不能超过 2MB")
		return
	}

	avatarURL, err := h.service.UploadAvatar(c.Request.Context(), userID.(string), file)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{
		"avatar":  avatarURL,
		"message": "头像上传成功",
	})
}
