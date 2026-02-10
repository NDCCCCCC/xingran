package system

import (
	"context"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models/system/requests"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services"
	apperrors "github.com/ruoyi-next/ruoyi-go-backend/pkg/errors"
	"gorm.io/gorm"
)

// userCacheService 用户缓存服务
type userCacheService struct {
	*userService
	cache CacheProvider
	CacheServiceBase
}

// NewUserServiceWithCache 创建带缓存的用户服务
func NewUserServiceWithCache(
	db *gorm.DB,
	cache CacheProvider,
	config *services.CacheConfigService,
	pwdManager PasswordManager,
) UserService {
	base := &userService{db: db, pwdManager: pwdManager}
	return &userCacheService{
		userService:     base,
		cache:           cache,
		CacheServiceBase: CacheServiceBase{Config: config},
	}
}

// GetByIDWithCache 获取用户详情（带缓存）
func (s *userCacheService) GetByIDWithCache(ctx context.Context, id string) (*models.User, error) {
	cacheKey := services.GetUserByIDKey(id)
	var result models.User

	expiration := s.GetExpiration(services.CacheConfigUserByID, 30*time.Minute)

	err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
		return s.userService.GetByID(ctx, id)
	})

	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetByUsernameWithCache 根据用户名获取用户（带缓存）
func (s *userCacheService) GetByUsernameWithCache(ctx context.Context, username string) (*models.User, error) {
	cacheKey := services.GetUserByUsernameKey(username)
	var result models.User

	expiration := s.GetExpiration(services.CacheConfigUserByUsername, 30*time.Minute)

	err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
		return s.getByUsername(ctx, username)
	})

	if err != nil {
		return nil, err
	}
	return &result, nil
}

// getByUsername 根据用户名查询用户（内部方法）
func (s *userCacheService) getByUsername(ctx context.Context, username string) (*models.User, error) {
	var user models.User
	err := s.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.UserNotFoundWithID(username)
		}
		return nil, apperrors.DatabaseError(err)
	}
	return &user, nil
}

// GetRolesWithCache 获取用户角色（带缓存）
func (s *userCacheService) GetRolesWithCache(ctx context.Context, userID string) ([]models.Role, error) {
	cacheKey := services.GetUserRolesKey(userID)
	var result []models.Role

	expiration := s.GetExpiration(services.CacheConfigUserRoles, 30*time.Minute)

	err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
		return s.queryRoles(ctx, userID)
	})

	return result, err
}

// queryRoles 查询用户的角色
func (s *userCacheService) queryRoles(ctx context.Context, userID string) ([]models.Role, error) {
	var roleIDs []string
	if err := s.db.WithContext(ctx).
		Table("sys_user_role").
		Where("user_id = ?", userID).
		Pluck("role_id", &roleIDs).Error; err != nil {
		return nil, err
	}

	if len(roleIDs) == 0 {
		return []models.Role{}, nil
	}

	var roles []models.Role
	if err := s.db.WithContext(ctx).
		Where("id IN ? AND status = ?", roleIDs, models.RoleStatusEnabled).
		Find(&roles).Error; err != nil {
		return nil, err
	}

	return roles, nil
}

// GetPermissionsWithCache 获取用户权限（带缓存）
func (s *userCacheService) GetPermissionsWithCache(ctx context.Context, userID string) ([]string, error) {
	cacheKey := services.GetUserPermissionsKey(userID)
	var result []string

	expiration := s.GetExpiration(services.CacheConfigUserByID, 30*time.Minute)

	err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
		return s.queryPermissions(ctx, userID)
	})

	return result, err
}

// queryPermissions 查询用户的权限
func (s *userCacheService) queryPermissions(ctx context.Context, userID string) ([]string, error) {
	// 通过用户角色获取菜单权限
	var perms []string

	// 获取用户角色
	roles, err := s.queryRoles(ctx, userID)
	if err != nil {
		return nil, err
	}

	if len(roles) == 0 {
		return []string{}, nil
	}

	// 获取角色关联的菜单权限
	var menuIDs []string
	if err := s.db.WithContext(ctx).
		Table("sys_role_menu").
		Select("DISTINCT menu_id").
		Where("role_id IN ?", getRoleIDs(roles)).
		Pluck("menu_id", &menuIDs).Error; err != nil {
		return nil, err
	}

	if len(menuIDs) == 0 {
		return []string{}, nil
	}

	// 获取菜单的权限标识
	if err := s.db.WithContext(ctx).
		Model(&models.Menu{}).
		Where("id IN ? AND status = ?", menuIDs, models.MenuStatusNormal).
		Pluck("perms", &perms).Error; err != nil {
		return nil, err
	}

	return perms, nil
}

// getRoleIDs 提取角色ID列表
func getRoleIDs(roles []models.Role) []string {
	ids := make([]string, len(roles))
	for i, r := range roles {
		ids[i] = r.ID
	}
	return ids
}

// InvalidateUserCache 失效用户相关缓存
func (s *userCacheService) InvalidateUserCache(ctx context.Context, userID string) error {
	keys := []string{
		services.GetUserByIDKey(userID),
	}
	// 还需要失效用户名缓存，但需要先查询用户名
	if user, err := s.userService.GetByID(ctx, userID); err == nil && user.Username != "" {
		keys = append(keys, services.GetUserByUsernameKey(user.Username))
	}
	keys = append(keys,
		services.GetUserRolesKey(userID),
		services.GetUserPermissionsKey(userID),
	)
	InvalidateCacheByKey(ctx, s.cache, keys, "USER")
	return nil
}

// InvalidateAllUserCache 失效所有用户缓存
func (s *userCacheService) InvalidateAllUserCache(ctx context.Context) error {
	InvalidateCacheByPattern(ctx, s.cache, []string{services.CacheKeyUserByID + "*"}, "USER")
	return nil
}

// Create 创建用户（带缓存失效）
func (s *userCacheService) Create(ctx context.Context, req *requests.UserCreateRequest) error {
	if err := s.userService.Create(ctx, req); err != nil {
		return err
	}
	// 新建用户不需要清除缓存，但清除列表缓存
	InvalidateCacheByPattern(ctx, s.cache, []string{services.CacheKeyUserList + "*"}, "USER")
	return nil
}

// Update 更新用户（带缓存失效）
func (s *userCacheService) Update(ctx context.Context, req *requests.UserUpdateRequest) error {
	if err := s.userService.Update(ctx, req); err != nil {
		return err
	}
	return s.InvalidateUserCache(ctx, req.ID)
}

// Delete 删除用户（带缓存失效）
func (s *userCacheService) Delete(ctx context.Context, id string) error {
	if err := s.userService.Delete(ctx, id); err != nil {
		return err
	}
	return s.InvalidateUserCache(ctx, id)
}

// BatchDelete 批量删除用户（带缓存失效）
func (s *userCacheService) BatchDelete(ctx context.Context, ids []string) error {
	if err := s.userService.BatchDelete(ctx, ids); err != nil {
		return err
	}
	// 清除所有用户相关缓存
	InvalidateCacheByPattern(ctx, s.cache, []string{services.CacheKeyUserByID + "*"}, "USER")
	return nil
}

// UpdateStatus 更新用户状态（带缓存失效）
func (s *userCacheService) UpdateStatus(ctx context.Context, id string, status int) error {
	if err := s.userService.UpdateStatus(ctx, id, status); err != nil {
		return err
	}
	return s.InvalidateUserCache(ctx, id)
}

// ResetPassword 重置用户密码（带缓存失效）
func (s *userCacheService) ResetPassword(ctx context.Context, id string, newPassword string) error {
	if err := s.userService.ResetPassword(ctx, id, newPassword); err != nil {
		return err
	}
	return s.InvalidateUserCache(ctx, id)
}
