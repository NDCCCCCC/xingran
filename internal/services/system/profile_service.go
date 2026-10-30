package system

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/core/security"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
	"github.com/xingran-next/xingran-go-backend/internal/services/addomain"
	"gorm.io/gorm"
)

// P1 fix: 定义 sentinel error 替代字符串比较。
// handler 用 errors.Is(err, ErrOldPasswordIncorrect) 而非
// err.Error() == "旧密码错误" — 后者在错误信息本地化/修改时会静默失效。
var ErrOldPasswordIncorrect = errors.New("旧密码错误")

// ProfileService 个人设置服务接口
type ProfileService interface {
	GetUserInfo(ctx context.Context, userID string) (*UserInfo, error)
	UpdateUserInfo(ctx context.Context, userID string, req *UpdateUserInfoRequest) error
	ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error
	UploadAvatar(ctx context.Context, userID string, file *multipart.FileHeader) (string, error)
}

// profileService 个人设置服务实现
type profileService struct {
	db            *gorm.DB
	passwordMgr   *security.PasswordManager
	uploadBaseDir string
}

// NewProfileService 创建个人设置服务实例
func NewProfileService(db *gorm.DB) ProfileService {
	return &profileService{
		db:            db,
		passwordMgr:   security.NewPasswordManager(nil),
		uploadBaseDir: "./uploads/avatar",
	}
}

// ==================== 数据类型 ====================

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

// UpdateUserInfoRequest 更新用户信息请求
type UpdateUserInfoRequest struct {
	Nickname *string
	Email    *string
	Phone    *string
	Gender   int
	Remark   *string
}

// ==================== 服务方法 ====================

// GetUserInfo 获取用户信息
func (s *profileService) GetUserInfo(ctx context.Context, userID string) (*UserInfo, error) {
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

// UpdateUserInfo 更新用户信息（Phase 57 GAP-02：自助资料修改与 AD 入队同事务）
//
// 信任边界决策（为什么自助路径也入队，而不是按"管理员专用"排除）：
//   - 该路径的 userID 来自会话（profile_handler.go `c.Get("user_id")`），不是请求体 ——
//     用户只能为"自己"入队，不存在跨用户写面；nickname/email/phone 本就是自助可改字段，
//     队列行只是把这次自编辑的既有效果延伸到自己的 AD 账号。
//   - 若不入队：AD 同步用户（auth_source='ad'）的 email/phone 在每次 AD 同步时被 AD 侧值
//     无条件覆盖（user_sync_service.go updateExistingUser；nickname 为空才回填），自助
//     改动会被下次同步**静默回滚**，且不产生任何队列行 —— Phase 59 GUARD-01 以
//     pending/failed 行判据，会把这次分叉判成"一致"。入队让 Phase 58 排水器把编辑推到
//     AD、下次同步读回同一值，分叉闭环。
//   - AD 侧写权限：排水器按 D-05 重读 DB 全量推送（属性 Modify 幂等），与管理员路径
//     同一执行面，本路径不新增任何 LDAP 语义；v1.21 的门控仍是全局 D-02
//     （sync_enabled + 配置启用），不按"自助/管理员"区分 —— 与 Phase 57 既有契约一致。
func (s *profileService) UpdateUserInfo(ctx context.Context, userID string, req *UpdateUserInfoRequest) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 旧值快照：diff 输入，必须与写入同一事务读取（同 snapshot，T-57-02 同款约束）。
		var oldUser models.User
		if err := tx.Where("id = ?", userID).First(&oldUser).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return fmt.Errorf("用户不存在")
			}
			return fmt.Errorf("查询用户信息失败: %w", err)
		}

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

		result := tx.Model(&models.User{}).Where("id = ?", userID).Updates(updates)
		if result.Error != nil {
			return fmt.Errorf("更新用户信息失败: %w", result.Error)
		}

		// Phase 57 GAP-02: nickname/email/phone 实际变更时同事务入队 update_user。
		// 复用 addomain.HasADFieldChanged / EnqueueUpdateIfChanged —— 与管理员 Update
		// 路径完全相同的 diff（nil = 未提供 = 无变更、TrimSpace 比较）与 D-02 两层降级。
		// gender / remark 不在 AD 四字段契约内，改了也不入队；DeptID 留 nil（自助路径
		// 无部门字段）→ 不参与 diff。入队失败 → 返回 error 使事务整体回滚（业务写与
		// 队列行同生共死，QUEUE-01）。
		if err := addomain.NewADPushEnqueuer().EnqueueUpdateIfChanged(tx, &oldUser, &requests.UserUpdateRequest{
			ID:       userID,
			Nickname: req.Nickname,
			Email:    req.Email,
			Phone:    req.Phone,
		}); err != nil {
			return err
		}

		return nil
	})
}

// ChangePassword 修改密码
func (s *profileService) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
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
		return ErrOldPasswordIncorrect
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
func (s *profileService) UploadAvatar(ctx context.Context, userID string, file *multipart.FileHeader) (string, error) {
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
	if mkdirErr := os.MkdirAll(s.uploadBaseDir, 0755); mkdirErr != nil {
		return "", fmt.Errorf("创建上传目录失败: %w", mkdirErr)
	}

	// 5. 保存文件
	dstPath := filepath.Join(s.uploadBaseDir, filename)
	dst, err := os.Create(dstPath)
	if err != nil {
		return "", fmt.Errorf("创建文件失败: %w", err)
	}
	defer dst.Close()

	if _, copyErr := io.Copy(dst, src); copyErr != nil {
		return "", fmt.Errorf("保存文件失败: %w", copyErr)
	}

	// 6. 更新用户头像路径
	avatarURL := fmt.Sprintf("/uploads/avatar/%s", filename)
	if updateErr := s.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", userID).Update("avatar", avatarURL).Error; updateErr != nil {
		return "", fmt.Errorf("更新头像失败: %w", updateErr)
	}

	return avatarURL, nil
}
