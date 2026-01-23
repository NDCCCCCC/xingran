package system

import (
	"context"
	"fmt"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/core/security"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"gorm.io/gorm"
)

// PasswordManager 密码管理器接口
type PasswordManager interface {
	HashPassword(password string) (string, error)
	VerifyPassword(password, hash string) (bool, error)
}

// CacheInvalidator 缓存失效接口
type CacheInvalidator interface {
	InvalidateUserCache(ctx context.Context, userID string) error
	InvalidateAllUserCache(ctx context.Context) error
	InvalidateUserPostCache(ctx context.Context, userID string) error
}

// UserService 用户服务接口
type UserService interface {
	Create(ctx context.Context, user *UserCreateRequest) error
	Update(ctx context.Context, user *UserUpdateRequest) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*models.User, error)
	List(ctx context.Context, params UserListParams) (*PageResult, error)
	BatchDelete(ctx context.Context, ids []string) error
	UpdateStatus(ctx context.Context, id string, status int) error
	ResetPassword(ctx context.Context, id string, newPassword string) error
}

// userService 用户服务实现
type userService struct {
	db              *gorm.DB
	pwdManager      PasswordManager
	cacheInvalidator CacheInvalidator
}

// NewUserService 创建用户服务实例
func NewUserService(db *gorm.DB, pwdManager PasswordManager, cacheInvalidator CacheInvalidator) UserService {
	return &userService{
		db:              db,
		pwdManager:      pwdManager,
		cacheInvalidator: cacheInvalidator,
	}
}

// ==================== 请求/响应类型 ====================

// UserListParams 用户列表查询参数
type UserListParams struct {
	Username  *string
	Nickname  *string
	Phone     *string
	Status    *int
	DeptID    *string
	BeginTime *string
	EndTime   *string
	Current   int
	PageSize  int
}

// DefaultUserListParams 默认列表参数
func DefaultUserListParams() UserListParams {
	return UserListParams{
		Current:  1,
		PageSize: 10,
	}
}

// UserCreateRequest 创建用户请求
type UserCreateRequest struct {
	Username   string
	Password   string
	Nickname   *string
	EmployeeNo *string
	Email      *string
	Phone      *string
	Gender     models.Gender
	Status     models.UserStatus
	DeptID     *string
	RoleIds    []string
	PostIds    []string
	Remark     *string
}

// ToModel 转换为用户模型
func (r *UserCreateRequest) ToModel(hashedPassword string) models.User {
	return models.User{
		Username:   r.Username,
		Password:   hashedPassword,
		Nickname:   r.Nickname,
		EmployeeNo: r.EmployeeNo,
		Email:      r.Email,
		Phone:      r.Phone,
		Gender:     r.Gender,
		Status:     r.Status,
		DeptID:     r.DeptID,
		Remark:     toStringPtr(r.Remark),
	}
}

// UserUpdateRequest 更新用户请求
type UserUpdateRequest struct {
	ID         string
	Nickname   *string
	EmployeeNo *string
	Email      *string
	Phone      *string
	Gender     models.Gender
	Status     models.UserStatus
	DeptID     *string
	RoleIds    []string
	PostIds    []string
	Remark     *string
}

// ToModel 转换为用户模型
func (r *UserUpdateRequest) ToModel() models.User {
	return models.User{
		BaseModel:  models.BaseModel{ID: r.ID},
		Nickname:   r.Nickname,
		EmployeeNo: r.EmployeeNo,
		Email:      r.Email,
		Phone:      r.Phone,
		Gender:     r.Gender,
		Status:     r.Status,
		DeptID:     r.DeptID,
		Remark:     toStringPtr(r.Remark),
	}
}

// PageResult 分页结果
type PageResult struct {
	List     interface{} `json:"list"`
	Total    int64       `json:"total"`
	Current  int         `json:"current"`
	PageSize int         `json:"pageSize"`
}

// ==================== 服务方法实现 ====================

// Create 创建用户
func (s *userService) Create(ctx context.Context, req *UserCreateRequest) error {
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

		// 分配角色
		if len(req.RoleIds) > 0 {
			for _, roleID := range req.RoleIds {
				if err := tx.Table("sys_user_role").Create(&models.UserRole{
					UserID: user.ID,
					RoleID: roleID,
				}).Error; err != nil {
					return fmt.Errorf("分配角色失败: %w", err)
				}
			}
		}

		// 分配岗位
		if len(req.PostIds) > 0 {
			for _, postID := range req.PostIds {
				if err := tx.Table("sys_user_post").Create(&models.UserPost{
					UserID: user.ID,
					PostID: postID,
				}).Error; err != nil {
					return fmt.Errorf("分配岗位失败: %w", err)
				}
			}
		}

		return nil
	})

	return err
}

// Update 更新用户
func (s *userService) Update(ctx context.Context, req *UserUpdateRequest) error {
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

		if err := tx.Save(&user).Error; err != nil {
			return fmt.Errorf("更新用户失败: %w", err)
		}

		// 更新角色关联：先删除旧关联，再添加新关联
		if err := tx.Table("sys_user_role").Where("user_id = ?", req.ID).Delete(&models.UserRole{}).Error; err != nil {
			return fmt.Errorf("删除用户角色关联失败: %w", err)
		}
		if len(req.RoleIds) > 0 {
			for _, roleID := range req.RoleIds {
				if err := tx.Table("sys_user_role").Create(&models.UserRole{
					UserID: req.ID,
					RoleID: roleID,
				}).Error; err != nil {
					return fmt.Errorf("分配角色失败: %w", err)
				}
			}
		}

		// 更新岗位关联：先删除旧关联，再添加新关联
		if err := tx.Table("sys_user_post").Where("user_id = ?", req.ID).Delete(&models.UserPost{}).Error; err != nil {
			return fmt.Errorf("删除用户岗位关联失败: %w", err)
		}
		if len(req.PostIds) > 0 {
			for _, postID := range req.PostIds {
				if err := tx.Table("sys_user_post").Create(&models.UserPost{
					UserID: req.ID,
					PostID: postID,
				}).Error; err != nil {
					return fmt.Errorf("分配岗位失败: %w", err)
				}
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
func (s *userService) Delete(ctx context.Context, id string) error {
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
func (s *userService) GetByID(ctx context.Context, id string) (*models.User, error) {
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

// List 查询用户列表
func (s *userService) List(ctx context.Context, params UserListParams) (*PageResult, error) {
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

	// 分页查询
	offset := (params.Current - 1) * params.PageSize
	if err := query.Preload("Dept").
		Order("created_at DESC").
		Offset(offset).Limit(params.PageSize).
		Find(&list).Error; err != nil {
		return nil, fmt.Errorf("查询用户列表失败: %w", err)
	}

	// 填充用户角色信息
	userIDs := make([]string, len(list))
	for i, u := range list {
		userIDs[i] = u.ID
	}

	// 查询用户角色关系
	var userRoles []models.UserRole
	if len(userIDs) > 0 {
		s.db.WithContext(ctx).Where("user_id IN ?", userIDs).Find(&userRoles)
	}

	// 构建用户ID到角色ID的映射
	userRoleMap := make(map[string][]string)
	for _, ur := range userRoles {
		userRoleMap[ur.UserID] = append(userRoleMap[ur.UserID], ur.RoleID)
	}

	// 查询所有相关角色
	var allRoles []models.Role
	if len(userRoles) > 0 {
		roleIDs := make([]string, 0)
		for _, ur := range userRoles {
			roleIDs = append(roleIDs, ur.RoleID)
		}
		s.db.WithContext(ctx).Where("id IN ?", roleIDs).Find(&allRoles)
	}

	// 构建角色ID到角色名称的映射
	roleMap := make(map[string]string)
	for _, r := range allRoles {
		roleMap[r.ID] = r.RoleName
	}

	// 填充每个用户的角色名称数组
	for i := range list {
		if roleIDs, ok := userRoleMap[list[i].ID]; ok {
			roleNames := make([]string, 0, len(roleIDs))
			for _, rid := range roleIDs {
				if roleName, exists := roleMap[rid]; exists {
					roleNames = append(roleNames, roleName)
				}
			}
			list[i].Roles = roleNames
		}
	}

	return &PageResult{
		List:     list,
		Total:    total,
		Current:  params.Current,
		PageSize: params.PageSize,
	}, nil
}

// BatchDelete 批量删除用户
func (s *userService) BatchDelete(ctx context.Context, ids []string) error {
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
func (s *userService) UpdateStatus(ctx context.Context, id string, status int) error {
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
func (s *userService) ResetPassword(ctx context.Context, id string, newPassword string) error {
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

// ==================== 辅助函数 ====================

// toStringPtr 将字符串转换为指针
func toStringPtr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// passwordManagerAdapter 密码管理器适配器
type passwordManagerAdapter struct {
	*security.PasswordManager
}

// 确保passwordManagerAdapter实现了PasswordManager接口
var _ PasswordManager = (*passwordManagerAdapter)(nil)

// NewPasswordManagerAdapter 创建密码管理器适配器
func NewPasswordManagerAdapter(pm *security.PasswordManager) PasswordManager {
	return &passwordManagerAdapter{PasswordManager: pm}
}
