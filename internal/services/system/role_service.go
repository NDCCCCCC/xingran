package system

import (
	"context"
	"fmt"
	"strconv"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// RoleService 角色服务接口
type RoleService interface {
	Create(ctx context.Context, req *requests.RoleCreateRequest) error
	Update(ctx context.Context, req *requests.RoleUpdateRequest) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*models.Role, error)
	List(ctx context.Context, params requests.RoleListParams) (*PageResult, error)
	Statistics(ctx context.Context) (*RoleStatisticsResult, error)
	GetAllEnabled(ctx context.Context) ([]*models.Role, error)
	BatchDelete(ctx context.Context, ids []string) error
	UpdateStatus(ctx context.Context, id string, status int) error

	// 新增缓存方法
	GetAllEnabledWithCache(ctx context.Context) ([]*models.Role, error)
	GetMenusWithCache(ctx context.Context, roleID string) ([]models.Menu, error)
	GetDeptsWithCache(ctx context.Context, roleID string) ([]models.Department, error)
	InvalidateRoleCache(ctx context.Context, roleID string) error
}

// userService 用户服务实现
type roleService struct {
	db *gorm.DB
}

// NewRoleService 创建角色服务实例
func NewRoleService(db *gorm.DB) RoleService {
	return &roleService{db: db}
}

// Statistics 统计角色总数及启停状态计数(status: 0=正常, 1=停用)。
// 用条件聚合(SUM CASE)避免「加载全量行进内存再 filter」; base query 与 List 一致,
// GORM 自动排除软删除记录。system 模块 list 的 MaxPageSize=100,旧前端用 pageSize:1000
// 拉全量再 .length 计数,角色数 >100 时会错误卡在 100。
func (s *roleService) Statistics(ctx context.Context) (*RoleStatisticsResult, error) {
	var result RoleStatisticsResult
	err := s.db.WithContext(ctx).
		Model(&models.Role{}).
		Select(
			"COUNT(*) AS total",
			"SUM(CASE WHEN status = 0 THEN 1 ELSE 0 END) AS active",
			"SUM(CASE WHEN status = 1 THEN 1 ELSE 0 END) AS inactive",
		).
		Scan(&result).Error
	if err != nil {
		return nil, fmt.Errorf("统计角色状态失败: %w", err)
	}
	return &result, nil
}

// RoleStatisticsResult 角色统计结果。status: 0=正常, 1=停用。
type RoleStatisticsResult struct {
	Total    int64 `json:"total"`
	Active   int64 `json:"active"`   // status = 0
	Inactive int64 `json:"inactive"` // status = 1
}

// roleAllowedSortFields 角色可排序字段白名单。
// 注:角色默认多列排序(role_sort ASC, created_at DESC),用户单选排序时仅生效一列,
// 不传 OrderByColumn 时保留原多列默认。
var roleAllowedSortFields = map[string]string{
	"roleName":  "role_name",
	"roleKey":   "role_key",
	"roleSort":  "role_sort",
	"status":    "status",
	"createdAt": "created_at",
}

// ==================== 服务方法实现 ====================

// Create 创建角色
func (s *roleService) Create(ctx context.Context, req *requests.RoleCreateRequest) error {
	// 检查角色名称是否已存在
	if exists, err := s.checkRoleNameExists(ctx, req.RoleName, ""); err != nil {
		return fmt.Errorf("检查角色名称失败: %w", err)
	} else if exists {
		return apperrors.RoleExistsWithName(req.RoleName)
	}

	// 检查权限字符是否已存在
	if exists, err := s.checkRoleKeyExists(ctx, req.RoleKey, ""); err != nil {
		return fmt.Errorf("检查权限字符失败: %w", err)
	} else if exists {
		return apperrors.RoleKeyExists(req.RoleKey)
	}

	// 使用事务创建角色及其关联
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		newRole := models.Role{
			RoleName:          req.RoleName,
			RoleKey:           req.RoleKey,
			RoleSort:          req.RoleSort,
			DataScope:         req.DataScope,
			MenuCheckStrictly: true,
			DeptCheckStrictly: true,
			Status:            req.Status,
			Remark:            toStringPtr(req.Remark),
		}

		if err := tx.Create(&newRole).Error; err != nil {
			return fmt.Errorf("创建角色失败: %w", err)
		}

		if err := s.assignRoleMenusAndDepts(tx, newRole.ID, req.MenuIds, req.DeptIds, false); err != nil {
			return err
		}

		return nil
	})

	return err
}

// Update 更新角色
func (s *roleService) Update(ctx context.Context, req *requests.RoleUpdateRequest) error {
	// 检查角色是否存在
	var role models.Role
	if err := s.db.WithContext(ctx).First(&role, "id = ?", req.ID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return apperrors.RoleNotFound()
		}
		return fmt.Errorf("查询角色失败: %w", err)
	}

	// 检查角色名称是否已存在（排除自己）
	if exists, err := s.checkRoleNameExists(ctx, req.RoleName, req.ID); err != nil {
		return fmt.Errorf("检查角色名称失败: %w", err)
	} else if exists {
		return apperrors.RoleExistsWithName(req.RoleName)
	}

	// 检查权限字符是否已存在（排除自己）
	if exists, err := s.checkRoleKeyExists(ctx, req.RoleKey, req.ID); err != nil {
		return fmt.Errorf("检查权限字符失败: %w", err)
	} else if exists {
		return apperrors.RoleKeyExists(req.RoleKey)
	}

	// 使用事务更新角色及其关联
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 更新角色基本信息
		role.RoleName = req.RoleName
		role.RoleKey = req.RoleKey
		role.RoleSort = req.RoleSort
		role.DataScope = req.DataScope
		role.Status = req.Status
		role.Remark = toStringPtr(req.Remark)

		if err := tx.Save(&role).Error; err != nil {
			return fmt.Errorf("更新角色失败: %w", err)
		}

		// 更新角色菜单和部门关联
		if err := s.assignRoleMenusAndDepts(tx, req.ID, req.MenuIds, req.DeptIds, true); err != nil {
			return err
		}

		return nil
	})

	return err
}

// Delete 删除角色
func (s *roleService) Delete(ctx context.Context, id string) error {
	// 检查角色是否存在
	var role models.Role
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&role).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return apperrors.RoleNotFound()
		}
		return fmt.Errorf("查询角色失败: %w", err)
	}

	// 检查角色是否已分配给用户
	var count int64
	if err := s.db.WithContext(ctx).Table("sys_user_role").Where("role_id = ?", id).Count(&count).Error; err != nil {
		return fmt.Errorf("检查角色分配失败: %w", err)
	}
	if count > 0 {
		return apperrors.RoleHasUsers()
	}

	// 使用事务删除角色及其关联数据
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除角色菜单关联
		if err := tx.Table("sys_role_menu").Where("role_id = ?", id).Delete(&models.RoleMenu{}).Error; err != nil {
			return fmt.Errorf("删除角色菜单关联失败: %w", err)
		}

		// 删除角色部门关联
		if err := tx.Table("sys_role_dept").Where("role_id = ?", id).Delete(&models.RoleDept{}).Error; err != nil {
			return fmt.Errorf("删除角色部门关联失败: %w", err)
		}

		// 软删除角色
		if err := tx.Delete(&role).Error; err != nil {
			return fmt.Errorf("删除角色失败: %w", err)
		}

		return nil
	})

	return err
}

// GetByID 获取角色详情
func (s *roleService) GetByID(ctx context.Context, id string) (*models.Role, error) {
	var role models.Role
	err := s.db.WithContext(ctx).First(&role, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("角色不存在")
		}
		return nil, fmt.Errorf("查询角色失败: %w", err)
	}
	return &role, nil
}

// List 查询角色列表
func (s *roleService) List(ctx context.Context, params requests.RoleListParams) (*PageResult, error) {
	var total int64
	var list []models.Role

	query := s.db.WithContext(ctx).Model(&models.Role{})

	// 角色名称模糊查询
	if params.RoleName != "" {
		query = query.Where("role_name LIKE ?", "%"+params.RoleName+"%")
	}

	// 权限字符模糊查询
	if params.RoleKey != "" {
		query = query.Where("role_key LIKE ?", "%"+params.RoleKey+"%")
	}

	// 状态查询
	if params.Status != "" {
		if status, err := strconv.Atoi(params.Status); err == nil {
			query = query.Where("status = ?", status)
		}
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("查询角色总数失败: %w", err)
	}

	// 查询分页数据 - 用户排序(白名单)优先,无 OrderByColumn 时保留多列默认
	offset := (params.Current - 1) * params.PageSize
	query = base.ApplySort(query, params.BaseListRequest, roleAllowedSortFields)
	if params.OrderByColumn == "" {
		query = query.Order("role_sort ASC, created_at DESC")
	}
	if err := query.Offset(offset).Limit(params.PageSize).Find(&list).Error; err != nil {
		return nil, fmt.Errorf("查询角色列表失败: %w", err)
	}

	return &PageResult{
		List:     list,
		Total:    total,
		Current:  params.Current,
		PageSize: params.PageSize,
	}, nil
}

// GetAllEnabled 获取所有启用的角色
func (s *roleService) GetAllEnabled(ctx context.Context) ([]*models.Role, error) {
	var roles []models.Role
	err := s.db.WithContext(ctx).Where("status = ?", models.RoleStatusEnabled).
		Order("role_sort ASC, created_at DESC").
		Find(&roles).Error
	if err != nil {
		return nil, fmt.Errorf("查询角色列表失败: %w", err)
	}

	// 转换为指针数组
	result := make([]*models.Role, len(roles))
	for i := range roles {
		result[i] = &roles[i]
	}
	return result, nil
}

// BatchDelete 批量删除角色
func (s *roleService) BatchDelete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return fmt.Errorf("ids不能为空")
	}

	// 检查角色是否已分配给用户
	var count int64
	if err := s.db.WithContext(ctx).Table("sys_user_role").Where("role_id IN ?", ids).Count(&count).Error; err != nil {
		return fmt.Errorf("检查角色分配失败: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("部分角色已分配给用户，无法删除")
	}

	// 使用事务批量删除角色及其关联数据
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 批量删除角色菜单关联
		if err := tx.Table("sys_role_menu").Where("role_id IN ?", ids).Delete(&models.RoleMenu{}).Error; err != nil {
			return fmt.Errorf("删除角色菜单关联失败: %w", err)
		}

		// 批量删除角色部门关联
		if err := tx.Table("sys_role_dept").Where("role_id IN ?", ids).Delete(&models.RoleDept{}).Error; err != nil {
			return fmt.Errorf("删除角色部门关联失败: %w", err)
		}

		// 批量软删除角色
		if err := tx.Where("id IN ?", ids).Delete(&models.Role{}).Error; err != nil {
			return fmt.Errorf("批量删除角色失败: %w", err)
		}

		return nil
	})

	return err
}

// UpdateStatus 更新角色状态
func (s *roleService) UpdateStatus(ctx context.Context, id string, status int) error {
	// 检查角色是否存在
	var role models.Role
	if err := s.db.WithContext(ctx).First(&role, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return apperrors.RoleNotFound()
		}
		return fmt.Errorf("查询角色失败: %w", err)
	}

	// 更新状态
	if err := s.db.WithContext(ctx).Model(&role).Update("status", status).Error; err != nil {
		return fmt.Errorf("更新角色状态失败: %w", err)
	}

	return nil
}

// ==================== 私有辅助方法 ====================

// checkRoleNameExists 检查角色名称是否存在
func (s *roleService) checkRoleNameExists(ctx context.Context, roleName string, excludeID string) (bool, error) {
	var count int64
	query := s.db.WithContext(ctx).Model(&models.Role{}).Where("role_name = ?", roleName)
	if excludeID != "" {
		query = query.Where("id != ?", excludeID)
	}
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// checkRoleKeyExists 检查权限字符是否存在
func (s *roleService) checkRoleKeyExists(ctx context.Context, roleKey string, excludeID string) (bool, error) {
	var count int64
	query := s.db.WithContext(ctx).Model(&models.Role{}).Where("role_key = ?", roleKey)
	if excludeID != "" {
		query = query.Where("id != ?", excludeID)
	}
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// assignRoleMenusAndDepts 分配角色菜单和部门权限
func (s *roleService) assignRoleMenusAndDepts(tx *gorm.DB, roleID string, menuIds, deptIds []string, isUpdate bool) error {
	// 【重要】不自动添加父菜单ID，让前端通过 Tree 的父子关联自动勾选父菜单
	// 这样可以避免父菜单被勾选时所有子菜单都显示为勾选状态的问题

	// 如果是更新操作，先删除旧的菜单和部门权限（不在新列表中的）
	if isUpdate {
		// 删除不再需要的菜单权限
		if len(menuIds) > 0 {
			if err := tx.Table("sys_role_menu").Where("role_id = ? AND menu_id NOT IN ?", roleID, menuIds).Delete(&models.RoleMenu{}).Error; err != nil {
				return fmt.Errorf("删除旧菜单权限失败: %w", err)
			}
		} else {
			// 如果没有新菜单，删除所有旧菜单
			if err := tx.Table("sys_role_menu").Where("role_id = ?", roleID).Delete(&models.RoleMenu{}).Error; err != nil {
				return fmt.Errorf("删除旧菜单权限失败: %w", err)
			}
		}
		// 删除不再需要的部门权限
		if len(deptIds) > 0 {
			if err := tx.Table("sys_role_dept").Where("role_id = ? AND dept_id NOT IN ?", roleID, deptIds).Delete(&models.RoleDept{}).Error; err != nil {
				return fmt.Errorf("删除旧部门权限失败: %w", err)
			}
		} else {
			// 如果没有新部门，删除所有旧部门
			if err := tx.Table("sys_role_dept").Where("role_id = ?", roleID).Delete(&models.RoleDept{}).Error; err != nil {
				return fmt.Errorf("删除旧部门权限失败: %w", err)
			}
		}
	}

	// 使用 ON CONFLICT DO NOTHING 批量插入菜单权限（忽略已存在的记录）
	// 遵循 Go 最佳实践：使用批量插入避免 N+1 问题
	if len(menuIds) > 0 {
		roleMenus := make([]models.RoleMenu, len(menuIds))
		for i, menuID := range menuIds {
			roleMenus[i] = models.RoleMenu{
				RoleID: roleID,
				MenuID: menuID,
			}
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "role_id"}, {Name: "menu_id"}},
			DoNothing: true,
		}).Create(&roleMenus).Error; err != nil {
			return fmt.Errorf("批量分配菜单权限失败: %w", err)
		}
	}

	// 使用 ON CONFLICT DO NOTHING 批量插入部门权限（忽略已存在的记录）
	// 遵循 Go 最佳实践：使用批量插入避免 N+1 问题
	if len(deptIds) > 0 {
		roleDepts := make([]models.RoleDept, len(deptIds))
		for i, deptID := range deptIds {
			roleDepts[i] = models.RoleDept{
				RoleID: roleID,
				DeptID: deptID,
			}
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "role_id"}, {Name: "dept_id"}},
			DoNothing: true,
		}).Create(&roleDepts).Error; err != nil {
			return fmt.Errorf("批量分配部门权限失败: %w", err)
		}
	}

	return nil
}

// GetAllEnabledWithCache 获取所有启用的角色（无缓存版本，直接查询数据库）
func (s *roleService) GetAllEnabledWithCache(ctx context.Context) ([]*models.Role, error) {
	return s.GetAllEnabled(ctx)
}

// GetMenusWithCache 获取角色的菜单（无缓存版本，直接查询数据库）
func (s *roleService) GetMenusWithCache(ctx context.Context, roleID string) ([]models.Menu, error) {
	var menuIDs []string
	if err := s.db.WithContext(ctx).
		Table("sys_role_menu").
		Where("role_id = ?", roleID).
		Pluck("menu_id", &menuIDs).Error; err != nil {
		return nil, fmt.Errorf("查询角色菜单失败: %w", err)
	}

	if len(menuIDs) == 0 {
		return []models.Menu{}, nil
	}

	var menus []models.Menu
	if err := s.db.WithContext(ctx).
		Where("id IN ? AND status = ?", menuIDs, models.MenuStatusNormal).
		Order("order_num ASC").
		Find(&menus).Error; err != nil {
		return nil, fmt.Errorf("查询菜单失败: %w", err)
	}

	return menus, nil
}

// GetDeptsWithCache 获取角色的部门（无缓存版本，直接查询数据库）
func (s *roleService) GetDeptsWithCache(ctx context.Context, roleID string) ([]models.Department, error) {
	var deptIDs []string
	if err := s.db.WithContext(ctx).
		Table("sys_role_dept").
		Where("role_id = ?", roleID).
		Pluck("dept_id", &deptIDs).Error; err != nil {
		return nil, fmt.Errorf("查询角色部门失败: %w", err)
	}

	if len(deptIDs) == 0 {
		return []models.Department{}, nil
	}

	var depts []models.Department
	if err := s.db.WithContext(ctx).
		Where("id IN ? AND status = ?", deptIDs, models.DeptStatusNormal).
		Order("order_num ASC").
		Find(&depts).Error; err != nil {
		return nil, fmt.Errorf("查询部门失败: %w", err)
	}

	return depts, nil
}

// InvalidateRoleCache 失效角色缓存（无缓存版本，空操作）
func (s *roleService) InvalidateRoleCache(ctx context.Context, roleID string) error {
	return nil
}

// toStringPtr 函数已在 user_service.go 中定义

