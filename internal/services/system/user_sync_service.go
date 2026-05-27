package system

import (
	"context"
	"errors"
	"fmt"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/core/security"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	applogger "github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// ADUserInfoForSync AD用户信息（用于同步，由认证器提供）
type ADUserInfoForSync struct {
	UserDN      string // 用户DN
	Username    string // sAMAccountName
	DisplayName string // 显示名称
	Email       string // 邮箱
	Phone       string // 电话
	Mobile      string // 手机
	Title       string // 职位
	Department  string // 部门
}

// UserSyncService 用户同步服务
// 负责将AD用户信息同步到sys_user表
type UserSyncService struct {
	db         *gorm.DB
	pwdManager PasswordManager
}

// NewUserSyncService 创建用户同步服务
func NewUserSyncService(db *gorm.DB, pwdManager PasswordManager) *UserSyncService {
	return &UserSyncService{
		db:         db,
		pwdManager: pwdManager,
	}
}

// SyncUserFromAD 从AD同步用户到sys_user表
// 如果用户已存在（按ad_username查找），则更新信息；否则创建新用户
func (s *UserSyncService) SyncUserFromAD(ctx context.Context, adUser *ADUserInfoForSync, defaultRoleID string) (*models.User, error) {
	var user models.User

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. 检查用户是否已存在（按ad_username查找）
		err := tx.Where("ad_username = ? AND deleted_at IS NULL", adUser.Username).First(&user).Error

		if err == nil {
			// 用户已存在，更新信息
			return s.updateExistingUser(tx, &user, adUser)
		}

		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 用户不存在，创建新用户
			return s.createNewUser(tx, &user, adUser, defaultRoleID)
		}

		return fmt.Errorf("查询用户失败: %w", err)
	})

	if err != nil {
		return nil, fmt.Errorf("同步AD用户失败: %w", err)
	}

	return &user, nil
}

// createNewUser 创建新用户（AD用户首次登录）
func (s *UserSyncService) createNewUser(tx *gorm.DB, user *models.User, adUser *ADUserInfoForSync, defaultRoleID string) error {
	adUsername := adUser.Username
	adDN := adUser.UserDN
	displayName := adUser.DisplayName
	email := adUser.Email
	phone := adUser.Phone

	user.Username = adUser.Username
	user.Nickname = &displayName
	user.Email = &email
	user.Phone = &phone
	user.Status = models.UserStatusEnabled
	user.AuthSource = "ad"
	user.ADUsername = &adUsername
	user.AdDn = &adDN
	user.InitFlag = true    // 标记为初始密码，需要修改
	user.PwdExpireDays = 90

	// 设置默认密码（首次登录后要求修改）
	hashedPassword, err := s.pwdManager.HashPassword("123456")
	if err != nil {
		return fmt.Errorf("生成默认密码失败: %w", err)
	}
	user.Password = hashedPassword

	// 设置默认部门（如果配置了）
	if defaultDeptID := s.getDefaultDeptID(tx); defaultDeptID != "" {
		user.DeptID = &defaultDeptID
	}

	// 创建用户
	if err := tx.Create(user).Error; err != nil {
		return fmt.Errorf("创建用户失败: %w", err)
	}

	// 分配默认角色
	if defaultRoleID != "" {
		if err := s.assignRole(tx, user.ID, defaultRoleID); err != nil {
			return fmt.Errorf("分配角色失败: %w", err)
		}
	}

	return nil
}

// updateExistingUser 更新已存在的用户信息
func (s *UserSyncService) updateExistingUser(tx *gorm.DB, user *models.User, adUser *ADUserInfoForSync) error {
	// 使用 GORM 的 Updates 方法，传入数据库列名
	updates := map[string]interface{}{
		"email":  &adUser.Email,
		"phone":  &adUser.Phone,
		"ad_dn": &adUser.UserDN, // 使用数据库列名 ad_dn
	}

	// 只有当nickname为空时才更新（尊重本地修改）
	if user.Nickname == nil || *user.Nickname == "" {
		updates["nickname"] = &adUser.DisplayName
	}

	if err := tx.Model(user).Updates(updates).Error; err != nil {
		return fmt.Errorf("更新用户信息失败: %w", err)
	}

	return nil
}

// assignRole 分配角色给用户
func (s *UserSyncService) assignRole(tx *gorm.DB, userID string, roleID string) error {
	// 使用原生SQL插入角色关联（ON CONFLICT避免重复）
	sql := `INSERT INTO sys_user_role (user_id, role_id, created_at)
			VALUES (?, ?, NOW())
			ON CONFLICT (user_id, role_id) DO NOTHING`

	if err := tx.Exec(sql, userID, roleID).Error; err != nil {
		return fmt.Errorf("分配角色失败: %w", err)
	}

	return nil
}

// getDefaultDeptID 获取默认部门ID
func (s *UserSyncService) getDefaultDeptID(tx *gorm.DB) string {
	var config models.Config
	err := tx.Where("config_key = ?", "sys.auth.ad.default_dept_id").First(&config).Error
	if err != nil {
		return "" // 没有配置默认部门
	}
	return config.ConfigValue
}

// SyncADUser 实现 security.UserSyncer 接口
// 将 security.ADUserInfo 转换为内部类型并调用同步逻辑
func (s *UserSyncService) SyncADUser(ctx context.Context, adUserInfo *security.ADUserInfo, defaultRoleID string) (*security.SyncedUser, error) {
	// 添加调试日志
	applogger.Infof("[SyncADUser] 开始同步 AD 用户: username=%s, userDN=%s", adUserInfo.Username, adUserInfo.UserDN)

	// 转换 ADUserInfo -> ADUserInfoForSync
	adUser := &ADUserInfoForSync{
		UserDN:      adUserInfo.UserDN,
		Username:    adUserInfo.Username,
		DisplayName: adUserInfo.DisplayName,
		Email:       adUserInfo.Email,
		Phone:       adUserInfo.Phone,
		Mobile:      adUserInfo.Mobile,
		Title:       adUserInfo.Title,
		Department:  adUserInfo.Department,
	}

	user, err := s.SyncUserFromAD(ctx, adUser, defaultRoleID)
	if err != nil {
		applogger.Errorf("[SyncADUser] 同步失败: %v", err)
		return nil, err
	}

	applogger.Infof("[SyncADUser] 同步成功: userID=%s, username=%s", user.ID, user.Username)

	// 转换 models.User -> security.SyncedUser
	result := &security.SyncedUser{
		ID:       user.ID,
		Username: user.Username,
		Nickname: user.Nickname,
		Email:    user.Email,
		Phone:    user.Phone,
		Status:   int(user.Status),
		DeptID:   user.DeptID,
		Roles:    user.Roles,
	}

	return result, nil
}

// SyncError 同步错误信息
type SyncError struct {
	Username string `json:"username"`
	Error    string `json:"error"`
}

// BatchSyncResult 批量同步结果
type BatchSyncResult struct {
	Total   int         `json:"total"`
	Success int         `json:"success"`
	Failed  int         `json:"failed"`
	Skipped int         `json:"skipped"`
	Errors  []SyncError `json:"errors,omitempty"`
}

// BatchSyncADUsers 批量同步AD用户到sys_user表
// 支持批量操作，返回成功/失败统计和详细错误信息
func (s *UserSyncService) BatchSyncADUsers(
	ctx context.Context,
	users []*ADUserInfoForSync,
	defaultRoleID string,
) (*BatchSyncResult, error) {
	result := &BatchSyncResult{
		Total:   len(users),
		Success: 0,
		Failed:  0,
		Skipped: 0,
		Errors:  []SyncError{},
	}

	if len(users) == 0 {
		return result, nil
	}

	applogger.Infof("[BatchSyncADUsers] 开始批量同步 %d 个AD用户", len(users))

	for _, adUser := range users {
		// 转换为 security.ADUserInfo 以复用 SyncADUser 方法
		adUserInfo := &security.ADUserInfo{
			UserDN:      adUser.UserDN,
			Username:    adUser.Username,
			DisplayName: adUser.DisplayName,
			Email:       adUser.Email,
			Phone:       adUser.Phone,
			Mobile:      adUser.Mobile,
			Title:       adUser.Title,
			Department:  adUser.Department,
		}

		// 调用现有的同步逻辑
		syncedUser, err := s.SyncADUser(ctx, adUserInfo, defaultRoleID)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, SyncError{
				Username: adUser.Username,
				Error:    err.Error(),
			})
			applogger.Errorf("[BatchSyncADUsers] 同步用户 %s 失败: %v", adUser.Username, err)
		} else {
			result.Success++
			applogger.Infof("[BatchSyncADUsers] 同步用户 %s 成功, userID=%s", adUser.Username, syncedUser.ID)
		}
	}

	applogger.Infof("[BatchSyncADUsers] 批量同步完成: 成功%d, 失败%d, 跳过%d",
		result.Success, result.Failed, result.Skipped)

	return result, nil
}
