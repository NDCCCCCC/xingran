package system

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// ProfileHandler 个人设置处理器
type ProfileHandler struct {
	service systemServices.ProfileService
	core    *core.Core
}

// NewProfileHandler 创建个人设置处理器实例
func NewProfileHandler(service systemServices.ProfileService) *ProfileHandler {
	return &ProfileHandler{service: service}
}

// WithCore 注入 core 依赖（操作日志记录所需）。返回 receiver 自身以支持链式调用。
// Phase 34 操作日志全模块覆盖新增，用于 operlog.Record 访问 core.OperLogService 与
// core.GetDB()。不改写 NewProfileHandler 单参构造器签名，避免破坏既有调用点。
func (h *ProfileHandler) WithCore(core *core.Core) *ProfileHandler {
	if h != nil {
		h.core = core
	}
	return h
}

// GetInfo 获取当前用户个人信息
// @Summary 获取个人信息
// @Description 获取当前登录用户的个人信息
// @Tags 个人设置
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/profile/info [get]
func (h *ProfileHandler) GetInfo(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, apperrors.Unauthorized())
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
// @Summary 更新个人信息
// @Description 更新当前登录用户的个人信息
// @Tags 个人设置
// @Accept json
// @Produce json
// @Param request body object{nickname=string,email=string,phone=string,gender=int,remark=string} true "个人信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/profile/info/update [post]
func (h *ProfileHandler) UpdateInfo(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, apperrors.Unauthorized())
		return
	}

	var req ProfileInfoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
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

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "个人中心", operlog.OperTypeUpdate)
	response.Success(c, gin.H{"message": "更新成功"})
}

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=6,max=20"`
}

// ChangePassword 修改密码
// @Summary 修改密码
// @Description 修改当前登录用户的密码
// @Tags 个人设置
// @Accept json
// @Produce json
// @Param request body object{oldPassword=string,newPassword=string} true "密码信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/profile/change-password [post]
func (h *ProfileHandler) ChangePassword(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, apperrors.Unauthorized())
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	if err := h.service.ChangePassword(c.Request.Context(), userID.(string), req.OldPassword, req.NewPassword); err != nil {
		// P1 fix: 用 errors.Is 比较 sentinel error 替代字符串比较
		if errors.Is(err, systemServices.ErrOldPasswordIncorrect) {
			response.Error(c, apperrors.BadRequest("旧密码错误"))
		} else {
			response.Error(c, err)
		}
		return
	}

	// T-34-W2-02: 修改密码属于敏感路径，oldPassword/newPassword 必须在 oper_param 中
	// 被遮蔽。ShouldBindJSON 已消费 body 流，所以 RecordWithBody 的 GetRawData 会 EOF
	// 并回退到普通 Record —— 这里改为显式构造已遮蔽的 oper_param 以保证密码不出现在
	// sys_oper_log 中。
	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "个人中心", operlog.OperTypeReset,
		operlog.WithOperParam(operlog.FilterSensitiveParams(`{"oldPassword":"******","newPassword":"******"}`)))
	response.Success(c, gin.H{"message": "密码修改成功"})
}

// UploadAvatar 上传头像
// @Summary 上传头像
// @Description 上传用户头像
// @Tags 个人设置
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "头像文件"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/profile/avatar/upload [post]
func (h *ProfileHandler) UploadAvatar(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, apperrors.Unauthorized())
		return
	}

	// 解析表单，获取文件
	file, err := c.FormFile("file")
	if err != nil {
		response.Error(c, apperrors.BadRequest("文件上传失败"))
		return
	}

	// 验证文件大小（最大 2MB）
	if file.Size > 2*1024*1024 {
		response.Error(c, apperrors.BadRequest("文件大小不能超过 2MB"))
		return
	}

	avatarURL, err := h.service.UploadAvatar(c.Request.Context(), userID.(string), file)
	if err != nil {
		response.Error(c, err)
		return
	}

	// T-34-W2-04: 头像是 multipart 表单，FilterSensitiveParams 对其是 no-op。
	// 手工构造 oper_param 记录文件名与大小，绝不记录原始 multipart body。
	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "个人中心", operlog.OperTypeUpload,
		operlog.WithOperParam(`{"filename":"`+file.Filename+`","size":`+strconv.FormatInt(file.Size, 10)+`}`))
	response.Success(c, gin.H{
		"avatar":  avatarURL,
		"message": "头像上传成功",
	})
}
