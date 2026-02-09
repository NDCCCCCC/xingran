package system

import (
	"context"
	"fmt"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models/system/requests"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/gormutil"
	"gorm.io/gorm"
)

// userServiceOptimized 优化后的用户服务实现，解决N+1查询问题
type userServiceOptimized struct {
	db              *gorm.DB
	pwdManager      PasswordManager
	cacheInvalidator CacheInvalidator
}

// NewUserServiceOptimized 创建优化后的用户服务实例
func NewUserServiceOptimized(db *gorm.DB, pwdManager PasswordManager, cacheInvalidator CacheInvalidator) UserService {
	return &userServiceOptimized{
		db:              db,
		pwdManager:      pwdManager,
		cacheInvalidator: cacheInvalidator,
	}
}

// List 优化后的用户列表查询，使用预加载和批量查询解决N+1问题
func (s *userServiceOptimized) List(ctx context.Context, params requests.UserListParams) (*PageResult, error) {
	var total int64
	var list []models.User

	query := s.db.WithContext(ctx).Model(&models.User{})

	// 添加筛选条件
	if params.Username != nil && *params.Username != "" {
		query = query.Where("username LIKE ?", "%"+*params.Username+"%")
	}
	if params.Nickname != nil && *params.Nickname != "" {
		query = query.Where("nickname LIKE ?", "%"+*params.Nickname+"%")
	}
	if params.Phone != nil && *params.Phone != "" {
		query = query.Where("phone LIKE ?", "%"+*params.Phone+"%")
	}
	if params.Status != nil {
		query = query.Where("status = ?", *params.Status)
	}
	if params.DeptID != nil && *params.DeptID != "" {
		query = query.Where("dept_id = ?", *params.DeptID)
	}
	if params.BeginTime != nil && *params.BeginTime != "" {
		if beginTime, err := time.Parse("2006-01-02 15:04:05", *params.BeginTime); err == nil {
			query = query.Where("created_at >= ?", beginTime)
		}
	}
	if params.EndTime != nil && *params.EndTime != "" {
		if endTime, err := time.Parse("2006-01-02 15:04:05", *params.EndTime); err == nil {
			query = query.Where("created_at <= ?", endTime)
		}
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("统计用户数量失败: %w", err)
	}

	if total == 0 {
		return &PageResult{
			List:     []models.User{},
			Total:    0,
			Current:  params.Current,
			PageSize: params.PageSize,
		}, nil
	}

	// 使用PreloadBuilder构建预加载
	preloader := gormutil.NewPreloadBuilder()
	preloader.Add("Dept")

	// 分页查询并应用预加载
	offset := (params.Current - 1) * params.PageSize
	if err := preloader.Apply(query).
		Order("created_at DESC").
		Offset(offset).Limit(params.PageSize).
		Find(&list).Error; err != nil {
		return nil, fmt.Errorf("查询用户列表失败: %w", err)
	}

	if len(list) == 0 {
		return &PageResult{
			List:     list,
			Total:    total,
			Current:  params.Current,
			PageSize: params.PageSize,
		}, nil
	}

	// 批量加载用户角色信息（优化后的方法，使用一次查询）
	if err := s.loadUserRolesOptimized(ctx, list); err != nil {
		return nil, fmt.Errorf("加载用户角色失败: %w", err)
	}

	return &PageResult{
		List:     list,
		Total:    total,
		Current:  params.Current,
		PageSize: params.PageSize,
	}, nil
}

// loadUserRolesOptimized 优化后的角色加载方法，使用JOIN减少查询次数
func (s *userServiceOptimized) loadUserRolesOptimized(ctx context.Context, users []models.User) error {
	if len(users) == 0 {
		return nil
	}

	// 提取所有用户ID
	userIDs := gormutil.ExtractIDs(users, func(u models.User) string {
		return u.ID
	})

	// 使用JOIN一次性查询用户角色和角色信息
	// 这样可以从原来的3次查询（用户->用户角色->角色）减少到1次查询
	type UserRoleResult struct {
		UserID   string
		RoleID   string
		RoleName string
		RoleKey  string
	}

	var results []UserRoleResult
	err := s.db.WithContext(ctx).
		Table("sys_user_role").
		Select("sys_user_role.user_id, sys_user_role.role_id, sys_role.role_name, sys_role.role_key").
		Joins("INNER JOIN sys_role ON sys_user_role.role_id = sys_role.id").
		Where("sys_user_role.user_id IN ?", userIDs).
		Find(&results).Error

	if err != nil {
		return err
	}

	// 构建用户ID到角色名称列表的映射
	userRoleMap := gormutil.GroupBy(results, func(r UserRoleResult) string {
		return r.UserID
	})

	// 为每个用户填充角色信息
	for i := range users {
		if roleResults, ok := userRoleMap[users[i].ID]; ok {
			roleNames := gormutil.MapSlice(roleResults, func(r UserRoleResult) string {
				return r.RoleName
			})
			users[i].Roles = roleNames
		} else {
			users[i].Roles = []string{}
		}
	}

	return nil
}

// Create 创建用户
func (s *userServiceOptimized) Create(ctx context.Context, req *requests.UserCreateRequest) error {
	// 检查用户名是否已存在
	var count int64
	if err := s.db.WithContext(ctx).Model(&models.User{}).Where("username = ?", req.Username).Count(&count).Error; err != nil {
		return fmt.Errorf("检查用户名失败: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("用户名已存在")
	}

	// 加密密码
	hashedPassword, err := s.pwdManager.HashPassword(req.Password)
	if err != nil {
		return fmt.Errorf("密码加密失败: %w", err)
	}

	user := req.ToModel(hashedPassword)

	// 使用事务创建用户及其关联
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 创建用户
		if err := tx.Create(&user).Error; err != nil {
			return fmt.Errorf("创建用户失败: %w", err)
		}

		// 批量分配角色（优化：使用批量插入替代循环）
		if len(req.RoleIds) > 0 {
			userRoles := make([]models.UserRole, len(req.RoleIds))
			for i, roleID := range req.RoleIds {
				userRoles[i] = models.UserRole{
					UserID: user.ID,
					RoleID: roleID,
				}
			}
			if err := tx.Table("sys_user_role").Create(&userRoles).Error; err != nil {
				return fmt.Errorf("分配角色失败: %w", err)
			}
		}

		// 批量分配岗位（优化：使用批量插入替代循环）
		if len(req.PostIds) > 0 {
			userPosts := make([]models.UserPost, len(req.PostIds))
			for i, postID := range req.PostIds {
				userPosts[i] = models.UserPost{
					UserID: user.ID,
					PostID: postID,
				}
			}
			if err := tx.Table("sys_user_post").Create(&userPosts).Error; err != nil {
				return fmt.Errorf("分配岗位失败: %w", err)
			}
		}

		return nil
	})

	return err
}

// Update 更新用户
func (s *userServiceOptimized) Update(ctx context.Context, req *requests.UserUpdateRequest) error {
	// 检查用户是否存在
	var user models.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", req.ID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("用户不存在")
		}
		return fmt.Errorf("查询用户失败: %w", err)
	}

	// 使用事务更新用户及其关联
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 更新用户基本信息
		user.Nickname = req.Nickname
		user.EmployeeNo = req.EmployeeNo
		user.Email = req.Email
		user.Phone = req.Phone
		user.Gender = req.Gender
		user.Status = req.Status
		user.DeptID = req.DeptID
		user.Remark = toStringPtr(req.Remark)

		// 如果部门发生变化，同步更新部门名称
		if req.DeptID != nil {
			var dept models.Department
			if err := tx.Select("dept_name").First(&dept, "id = ?", *req.DeptID).Error; err == nil {
				user.DeptName = &dept.DeptName
			}
		} else {
			user.DeptName = nil
		}

		if err := tx.Save(&user).Error; err != nil {
			return fmt.Errorf("更新用户失败: %w", err)
		}

		// 更新角色关联：先删除旧关联，再批量添加新关联
		if err := tx.Table("sys_user_role").Where("user_id = ?", req.ID).Delete(&models.UserRole{}).Error; err != nil {
			return fmt.Errorf("删除用户角色关联失败: %w", err)
		}
		if len(req.RoleIds) > 0 {
			userRoles := make([]models.UserRole, len(req.RoleIds))
			for i, roleID := range req.RoleIds {
				userRoles[i] = models.UserRole{
					UserID: req.ID,
					RoleID: roleID,
				}
			}
			if err := tx.Table("sys_user_role").Create(&userRoles).Error; err != nil {
				return fmt.Errorf("分配角色失败: %w", err)
			}
		}

		// 更新岗位关联：先删除旧关联，再批量添加新关联
		if err := tx.Table("sys_user_post").Where("user_id = ?", req.ID).Delete(&models.UserPost{}).Error; err != nil {
			return fmt.Errorf("删除用户岗位关联失败: %w", err)
		}
		if len(req.PostIds) > 0 {
			userPosts := make([]models.UserPost, len(req.PostIds))
			for i, postID := range req.PostIds {
				userPosts[i] = models.UserPost{
					UserID: req.ID,
					PostID: postID,
				}
			}
			if err := tx.Table("sys_user_post").Create(&userPosts).Error; err != nil {
				return fmt.Errorf("分配岗位失败: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	// 清除用户相关缓存
	if s.cacheInvalidator != nil {
		_ = s.cacheInvalidator.InvalidateUserCache(ctx, req.ID)
		_ = s.cacheInvalidator.InvalidateUserPostCache(ctx, req.ID)
	}

	return nil
}

// Delete 删除用户
func (s *userServiceOptimized) Delete(ctx context.Context, id string) error {
	// 检查用户是否存在
	var user models.User
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("用户不存在")
		}
		return fmt.Errorf("查询用户失败: %w", err)
	}

	// 使用事务删除用户及其关联数据
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除用户角色关联
		if err := tx.Table("sys_user_role").Where("user_id = ?", id).Delete(&models.UserRole{}).Error; err != nil {
			return fmt.Errorf("删除用户角色关联失败: %w", err)
		}

		// 删除用户岗位关联
		if err := tx.Table("sys_user_post").Where("user_id = ?", id).Delete(&models.UserPost{}).Error; err != nil {
			return fmt.Errorf("删除用户岗位关联失败: %w", err)
		}

		// 软删除用户
		if err := tx.Delete(&user).Error; err != nil {
			return fmt.Errorf("删除用户失败: %w", err)
		}

		return nil
	})

	if err != nil {
		return err
	}

	// 清除用户缓存
	if s.cacheInvalidator != nil {
		_ = s.cacheInvalidator.InvalidateUserCache(ctx, id)
		_ = s.cacheInvalidator.InvalidateUserPostCache(ctx, id)
	}

	return nil
}

// GetByID 获取用户详情
func (s *userServiceOptimized) GetByID(ctx context.Context, id string) (*models.User, error) {
	var user models.User
	err := s.db.WithContext(ctx).Preload("Dept").First(&user, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("用户不存在")
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	return &user, nil
}

// BatchDelete 批量删除用户
func (s *userServiceOptimized) BatchDelete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return fmt.Errorf("ids不能为空")
	}

	// 使用事务批量删除用户及其关联数据
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 批量删除用户角色关联
		if err := tx.Table("sys_user_role").Where("user_id IN ?", ids).Delete(&models.UserRole{}).Error; err != nil {
			return fmt.Errorf("删除用户角色关联失败: %w", err)
		}

		// 批量删除用户岗位关联
		if err := tx.Table("sys_user_post").Where("user_id IN ?", ids).Delete(&models.UserPost{}).Error; err != nil {
			return fmt.Errorf("删除用户岗位关联失败: %w", err)
		}

		// 批量软删除用户
		if err := tx.Where("id IN ?", ids).Delete(&models.User{}).Error; err != nil {
			return fmt.Errorf("批量删除用户失败: %w", err)
		}

		return nil
	})

	if err != nil {
		return err
	}

	// 清除所有用户缓存
	if s.cacheInvalidator != nil {
		_ = s.cacheInvalidator.InvalidateAllUserCache(ctx)
	}

	return nil
}

// UpdateStatus 更新用户状态
func (s *userServiceOptimized) UpdateStatus(ctx context.Context, id string, status int) error {
	// 检查用户是否存在
	var user models.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("用户不存在")
		}
		return fmt.Errorf("查询用户失败: %w", err)
	}

	// 更新状态
	if err := s.db.WithContext(ctx).Model(&user).Update("status", status).Error; err != nil {
		return fmt.Errorf("更新用户状态失败: %w", err)
	}

	// 清除用户缓存
	if s.cacheInvalidator != nil {
		_ = s.cacheInvalidator.InvalidateUserCache(ctx, id)
	}

	return nil
}

// ResetPassword 重置用户密码
func (s *userServiceOptimized) ResetPassword(ctx context.Context, id string, newPassword string) error {
	// 检查用户是否存在
	var user models.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("用户不存在")
		}
		return fmt.Errorf("查询用户失败: %w", err)
	}

	// 加密新密码
	hashedPassword, err := s.pwdManager.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("密码加密失败: %w", err)
	}

	// 更新密码
	if err := s.db.WithContext(ctx).Model(&user).Update("password", hashedPassword).Error; err != nil {
		return fmt.Errorf("重置密码失败: %w", err)
	}

	// 清除用户缓存
	if s.cacheInvalidator != nil {
		_ = s.cacheInvalidator.InvalidateUserCache(ctx, id)
	}

	return nil
}
