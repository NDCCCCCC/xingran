package services

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/core/security"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"gorm.io/gorm"
)

// ProfileService 个人信息服务
type ProfileService struct {
	db            *gorm.DB
	passwordMgr   *security.PasswordManager
	uploadBaseDir string
}

// NewProfileService 创建个人信息 service
func NewProfileService(db *gorm.DB) *ProfileService {
	return &ProfileService{
		db:          db,
		passwordMgr: security.NewPasswordManager(nil),
		uploadBaseDir: "./uploads/avatar",
	}
}

// UserInfo 用户详细信息
type UserInfo struct {
	ID            string            `json:"id"`
	Username      string            `json:"username"`
	Nickname      *string           `json:"nickname"`
	Email         *string           `json:"email"`
	Phone         *string           `json:"phone"`
	Avatar        *string           `json:"avatar"`
	Gender        models.Gender     `json:"gender"`
	Status        models.UserStatus `json:"status"`
	DeptID        *string           `json:"deptId,omitempty"`
	DeptName      *string           `json:"deptName,omitempty"`
	Remark        *string           `json:"remark,omitempty"`
	LoginIP       *string           `json:"loginIp,omitempty"`
	LoginTime     *time.Time        `json:"loginTime,omitempty"`
	PwdUpdateTime *time.Time        `json:"pwdUpdateTime,omitempty"`
	CreatedAt     time.Time         `json:"createdAt"`
}

// GetUserInfo 获取用户信息
func (s *ProfileService) GetUserInfo(ctx context.Context, userID string) (*UserInfo, error) {
	var user models.User
	if err := s.db.WithContext(ctx).Where("id = ?", userID).First(&user).Error; err != nil {
		return nil, fmt.Errorf("查询用户信息失败: %w", err)
	}

	return &UserInfo{
		ID:            user.ID,
		Username:      user.Username,
		Nickname:      user.Nickname,
		Email:         user.Email,
		Phone:         user.Phone,
		Avatar:        user.Avatar,
		Gender:        user.Gender,
		Status:        user.Status,
		DeptID:        user.DeptID,
		DeptName:      user.DeptName,
		Remark:        &user.Remark,
		LoginIP:       user.LoginIP,
		LoginTime:     user.LoginTime,
		PwdUpdateTime: user.PwdUpdateTime,
		CreatedAt:     user.CreatedAt,
	}, nil
}

// UpdateUserInfoRequest 更新用户信息请求
type UpdateUserInfoRequest struct {
	Nickname *string
	Email    *string
	Phone    *string
	Gender   int
	Remark   *string
}

// UpdateUserInfo 更新用户信息
func (s *ProfileService) UpdateUserInfo(ctx context.Context, userID string, req *UpdateUserInfoRequest) error {
	// 使用 map 仅更新非空字段
	updates := make(map[string]interface{})

	if req.Nickname != nil {
		updates["nickname"] = *req.Nickname
	}
	if req.Email != nil {
		updates["email"] = *req.Email
	}
	if req.Phone != nil {
		updates["phone"] = *req.Phone
	}
	if req.Remark != nil {
		updates["remark"] = *req.Remark
	}
	updates["gender"] = req.Gender

	if len(updates) == 0 {
		return fmt.Errorf("没有需要更新的字段")
	}

	result := s.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", userID).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("更新用户信息失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("用户不存在")
	}

	return nil
}

// ChangePassword 修改密码
func (s *ProfileService) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	// 1. 查询用户当前密码
	var user models.User
	if err := s.db.WithContext(ctx).Select("id", "password", "salt").Where("id = ?", userID).First(&user).Error; err != nil {
		return fmt.Errorf("查询用户信息失败: %w", err)
	}

	// 2. 验证旧密码
	if ok, err := s.passwordMgr.VerifyPassword(oldPassword, user.Password); err != nil || !ok {
		if err != nil {
			return fmt.Errorf("密码验证失败: %w", err)
		}
		return fmt.Errorf("旧密码错误")
	}

	// 3. 加密新密码
	newHash, err := s.passwordMgr.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("密码加密失败: %w", err)
	}

	// 4. 更新密码
	now := time.Now()
	result := s.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"password":        newHash,
		"pwd_update_time": &now,
		"init_flag":       false,
	})

	if result.Error != nil {
		return fmt.Errorf("更新密码失败: %w", result.Error)
	}

	return nil
}

// UploadAvatar 上传头像
func (s *ProfileService) UploadAvatar(ctx context.Context, userID string, file *multipart.FileHeader) (string, error) {
	// 1. 打开上传的文件
	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("打开文件失败: %w", err)
	}
	defer src.Close()

	// 2. 验证文件类型（仅允许图片）
	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowedExts := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
		".webp": true,
	}
	if !allowedExts[ext] {
		return "", fmt.Errorf("不支持的文件类型，仅支持 jpg、jpeg、png、gif、webp 格式")
	}

	// 3. 生成唯一文件名
	filename := fmt.Sprintf("%s_%d%s", userID, time.Now().Unix(), ext)

	// 4. 确保上传目录存在
	if err := os.MkdirAll(s.uploadBaseDir, 0755); err != nil {
		return "", fmt.Errorf("创建上传目录失败: %w", err)
	}

	// 5. 保存文件
	dstPath := filepath.Join(s.uploadBaseDir, filename)
	dst, err := os.Create(dstPath)
	if err != nil {
		return "", fmt.Errorf("创建文件失败: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return "", fmt.Errorf("保存文件失败: %w", err)
	}

	// 6. 更新用户头像路径
	avatarURL := fmt.Sprintf("/uploads/avatar/%s", filename)
	if err := s.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", userID).Update("avatar", avatarURL).Error; err != nil {
		return "", fmt.Errorf("更新头像失败: %w", err)
	}

	return avatarURL, nil
}
