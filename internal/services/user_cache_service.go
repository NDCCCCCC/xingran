package services

import (
	"context"
	"fmt"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// UserCacheService 用户缓存服务
type UserCacheService struct {
	db        *gorm.DB
	dataCache *DataCacheService
}

// NewUserCacheService 创建用户缓存服务
func NewUserCacheService(db *gorm.DB, dataCache *DataCacheService) *UserCacheService {
	return &UserCacheService{
		db:        db,
		dataCache: dataCache,
	}
}

// GetUserByIDWithCache 根据ID获取用户（带缓存）
func (s *UserCacheService) GetUserByIDWithCache(ctx context.Context, userID string) (*models.User, error) {
	if userID == "" {
		return nil, fmt.Errorf("用户ID不能为空")
	}

	cacheKey := fmt.Sprintf("user:id:%s", userID)
	var user models.User

	// 使用动态配置的缓存时间
	expiration := s.dataCache.GetExpiration(CacheConfigUserByID, 30*time.Minute)

	// 尝试从缓存获取
	err := s.dataCache.GetOrSet(ctx, cacheKey, &user, expiration, func() (interface{}, error) {
		return s.queryUserByID(userID)
	})

	if err != nil {
		return nil, fmt.Errorf("获取用户失败: %w", err)
	}

	return &user, nil
}

// queryUserByID 从数据库查询用户
func (s *UserCacheService) queryUserByID(userID string) (*models.User, error) {
	var user models.User

	if err := s.db.Where("id = ?", userID).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("用户不存在")
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}

	// 加载用户的角色信息
	var roleIDs []string
	if err := s.db.Table("sys_user_role").
		Where("user_id = ?", userID).
		Pluck("role_id", &roleIDs).Error; err != nil {
		return nil, fmt.Errorf("查询用户角色失败: %w", err)
	}

	// 转换为字符串数组
	user.Roles = roleIDs

	return &user, nil
}

// GetUserByUsernameWithCache 根据用户名获取用户（带缓存）
func (s *UserCacheService) GetUserByUsernameWithCache(ctx context.Context, username string) (*models.User, error) {
	if username == "" {
		return nil, fmt.Errorf("用户名不能为空")
	}

	cacheKey := fmt.Sprintf("user:username:%s", username)
	var user models.User

	// 尝试从缓存获取
	err := s.dataCache.GetOrSet(ctx, cacheKey, &user, 30*time.Minute, func() (interface{}, error) {
		return s.queryUserByUsername(username)
	})

	if err != nil {
		return nil, fmt.Errorf("获取用户失败: %w", err)
	}

	return &user, nil
}

// queryUserByUsername 从数据库查询用户
func (s *UserCacheService) queryUserByUsername(username string) (*models.User, error) {
	var user models.User

	if err := s.db.Where("username = ?", username).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("用户不存在")
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}

	// 加载用户的角色信息
	var roleIDs []string
	if err := s.db.Table("sys_user_role").
		Where("user_id = ?", user.ID).
		Pluck("role_id", &roleIDs).Error; err != nil {
		return nil, fmt.Errorf("查询用户角色失败: %w", err)
	}

	// 转换为字符串数组
	user.Roles = roleIDs

	return &user, nil
}

// GetUserRolesWithCache 获取用户角色列表（带缓存）
func (s *UserCacheService) GetUserRolesWithCache(ctx context.Context, userID string) ([]string, error) {
	if userID == "" {
		return nil, fmt.Errorf("用户ID不能为空")
	}

	cacheKey := fmt.Sprintf("user:roles:%s", userID)
	var roles []string

	// 尝试从缓存获取
	err := s.dataCache.GetOrSet(ctx, cacheKey, &roles, 30*time.Minute, func() (interface{}, error) {
		return s.queryUserRoles(userID)
	})

	if err != nil {
		return nil, fmt.Errorf("获取用户角色失败: %w", err)
	}

	return roles, nil
}

// queryUserRoles 从数据库查询用户角色
func (s *UserCacheService) queryUserRoles(userID string) ([]string, error) {
	var roleIDs []string

	if err := s.db.Table("sys_user_role").
		Where("user_id = ?", userID).
		Pluck("role_id", &roleIDs).Error; err != nil {
		return nil, fmt.Errorf("查询用户角色失败: %w", err)
	}

	return roleIDs, nil
}

// GetUserPermissionsWithCache 获取用户权限列表（带缓存）
func (s *UserCacheService) GetUserPermissionsWithCache(ctx context.Context, userID string) ([]string, error) {
	if userID == "" {
		return nil, fmt.Errorf("用户ID不能为空")
	}

	cacheKey := fmt.Sprintf("user:permissions:%s", userID)
	var permissions []string

	// 尝试从缓存获取
	err := s.dataCache.GetOrSet(ctx, cacheKey, &permissions, 30*time.Minute, func() (interface{}, error) {
		return s.queryUserPermissions(userID)
	})

	if err != nil {
		return nil, fmt.Errorf("获取用户权限失败: %w", err)
	}

	return permissions, nil
}

// queryUserPermissions 从数据库查询用户权限
func (s *UserCacheService) queryUserPermissions(userID string) ([]string, error) {
	var permissions []string

	// 通过用户的角色获取菜单权限标识
	query := `
		SELECT DISTINCT m.perms
		FROM sys_menu m
		INNER JOIN sys_role_menu rm ON m.id = rm.menu_id
		INNER JOIN sys_user_role ur ON rm.role_id = ur.role_id
		WHERE ur.user_id = ?
		  AND m.status = 0
		  AND m.perms IS NOT NULL
		  AND m.perms != ''
		ORDER BY m.order_num ASC
	`

	if err := s.db.Raw(query, userID).Pluck("perms", &permissions).Error; err != nil {
		return nil, fmt.Errorf("查询用户权限失败: %w", err)
	}

	return permissions, nil
}

// InvalidateUserCache 清除用户相关缓存
func (s *UserCacheService) InvalidateUserCache(ctx context.Context, userID string) error {
	if userID == "" {
		return fmt.Errorf("用户ID不能为空")
	}

	patterns := []string{
		fmt.Sprintf("user:id:%s", userID),
		fmt.Sprintf("user:username:*"), // 用户名可能改变，所以清除所有用户名缓存
		fmt.Sprintf("user:roles:%s", userID),
		fmt.Sprintf("user:permissions:%s", userID),
	}

	// 先获取用户信息以便获取用户名
	user, err := s.queryUserByID(userID)
	if err == nil {
		patterns = append(patterns, fmt.Sprintf("user:username:%s", user.Username))
	}

	for _, pattern := range patterns {
		// 如果pattern包含*，使用DeleteByPattern
		if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
			if err := s.dataCache.DeleteByPattern(ctx, pattern[:len(pattern)-1]); err != nil {
				return fmt.Errorf("清除缓存失败 %s: %w", pattern, err)
			}
		} else {
			if err := s.dataCache.Delete(ctx, pattern); err != nil {
				return fmt.Errorf("清除缓存失败 %s: %w", pattern, err)
			}
		}
	}

	return nil
}

// InvalidateAllUserCache 清除所有用户缓存
func (s *UserCacheService) InvalidateAllUserCache(ctx context.Context) error {
	patterns := []string{
		"user:id:*",
		"user:username:*",
		"user:roles:*",
		"user:permissions:*",
	}

	for _, pattern := range patterns {
		if err := s.dataCache.DeleteByPattern(ctx, pattern); err != nil {
			return fmt.Errorf("清除缓存失败 %s: %w", pattern, err)
		}
	}

	return nil
}

// CreateUser 创建用户
func (s *UserCacheService) CreateUser(ctx context.Context, user *models.User) error {
	if err := s.db.Create(user).Error; err != nil {
		return fmt.Errorf("创建用户失败: %w", err)
	}

	// 不需要清除缓存，因为新用户还没有被缓存过
	return nil
}

// UpdateUser 更新用户
func (s *UserCacheService) UpdateUser(ctx context.Context, user *models.User) error {
	if err := s.db.Save(user).Error; err != nil {
		return fmt.Errorf("更新用户失败: %w", err)
	}

	// 清除缓存
	if err := s.InvalidateUserCache(ctx, user.ID); err != nil {
		logger.Warnf("[USER] 清除缓存失败: %v", err)
	}

	return nil
}

// DeleteUser 删除用户
func (s *UserCacheService) DeleteUser(ctx context.Context, userID string) error {
	if err := s.db.Delete(&models.User{}, "id = ?", userID).Error; err != nil {
		return fmt.Errorf("删除用户失败: %w", err)
	}

	// 清除缓存
	if err := s.InvalidateUserCache(ctx, userID); err != nil {
		logger.Warnf("[USER] 清除缓存失败: %v", err)
	}

	return nil
}

// AssignUserRoles 分配用户角色
func (s *UserCacheService) AssignUserRoles(ctx context.Context, userID string, roleIDs []string) error {
	// 使用事务
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 删除旧的角色关联
	if err := tx.Table("sys_user_role").Where("user_id = ?", userID).Delete(&models.UserRole{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除旧角色关联失败: %w", err)
	}

	// 添加新的角色关联
	if len(roleIDs) > 0 {
		var userRoles []models.UserRole
		for _, roleID := range roleIDs {
			userRoles = append(userRoles, models.UserRole{
				UserID: userID,
				RoleID: roleID,
			})
		}
		if err := tx.Create(&userRoles).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("添加角色关联失败: %w", err)
		}
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	// 清除缓存
	if err := s.InvalidateUserCache(ctx, userID); err != nil {
		logger.Warnf("[USER] 清除缓存失败: %v", err)
	}

	return nil
}

// UpdateUserStatus 更新用户状态
func (s *UserCacheService) UpdateUserStatus(ctx context.Context, userID string, status models.UserStatus) error {
	if err := s.db.Model(&models.User{}).
		Where("id = ?", userID).
		Update("status", status).Error; err != nil {
		return fmt.Errorf("更新用户状态失败: %w", err)
	}

	// 清除缓存
	if err := s.InvalidateUserCache(ctx, userID); err != nil {
		logger.Warnf("[USER] 清除缓存失败: %v", err)
	}

	return nil
}

// UpdateUserPassword 更新用户密码
func (s *UserCacheService) UpdateUserPassword(ctx context.Context, userID, password, salt string) error {
	if err := s.db.Model(&models.User{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"password":       password,
			"salt":           salt,
			"pwd_update_time": time.Now(),
		}).Error; err != nil {
		return fmt.Errorf("更新用户密码失败: %w", err)
	}

	// 清除缓存
	if err := s.InvalidateUserCache(ctx, userID); err != nil {
		logger.Warnf("[USER] 清除缓存失败: %v", err)
	}

	return nil
}

// GetUserListWithCache 获取用户列表（带缓存，仅用于不常变化的数据）
func (s *UserCacheService) GetUserListWithCache(ctx context.Context, deptID string, status models.UserStatus) ([]*models.User, error) {
	cacheKey := fmt.Sprintf("user:list:dept:%s:status:%d", deptID, status)
	var users []*models.User

	// 使用动态配置的缓存时间
	expiration := s.dataCache.GetExpiration(CacheConfigUserList, 5*time.Minute)

	// 尝试从缓存获取
	err := s.dataCache.GetOrSet(ctx, cacheKey, &users, expiration, func() (interface{}, error) {
		return s.queryUserList(deptID, status)
	})

	if err != nil {
		return nil, fmt.Errorf("获取用户列表失败: %w", err)
	}

	return users, nil
}

// queryUserList 从数据库查询用户列表
func (s *UserCacheService) queryUserList(deptID string, status models.UserStatus) ([]*models.User, error) {
	var users []models.User

	query := s.db.Model(&models.User{})

	if deptID != "" {
		query = query.Where("dept_id = ?", deptID)
	}

	if status >= 0 {
		query = query.Where("status = ?", status)
	}

	if err := query.Order("created_at DESC").Find(&users).Error; err != nil {
		return nil, fmt.Errorf("查询用户列表失败: %w", err)
	}

	// 转换为指针数组
	userPtrs := make([]*models.User, len(users))
	for i := range users {
		userPtrs[i] = &users[i]
	}

	return userPtrs, nil
}
