package system

import (
	"context"
	"fmt"
	"strconv"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"gorm.io/gorm"
)

// MenuService 菜单服务接口
type MenuService interface {
	Create(ctx context.Context, req *MenuCreateRequest) error
	Update(ctx context.Context, req *MenuUpdateRequest) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*models.Menu, error)
	GetTree(ctx context.Context) ([]models.Menu, error)
	List(ctx context.Context, params MenuListParams) ([]models.Menu, error)
	BatchDelete(ctx context.Context, ids []string) error
	UpdateStatus(ctx context.Context, id string, status int) error
	GetUserMenus(ctx context.Context, userID string) ([]models.Menu, error)
	GetAllUserMenus(ctx context.Context, userID string) ([]models.Menu, error)
	GetUserPermissions(ctx context.Context, userID string) ([]string, error)
	GetRoleMenuIDs(ctx context.Context, roleID string, menuIDs *[]string) error
}

// menuService 菜单服务实现
type menuService struct {
	db *gorm.DB
}

// NewMenuService 创建菜单服务实例
func NewMenuService(db *gorm.DB) MenuService {
	return &menuService{db: db}
}

// ==================== 请求/响应类型 ====================

// MenuListParams 菜单列表查询参数
type MenuListParams struct {
	MenuName string
	Status   string
}

// MenuCreateRequest 创建菜单请求
type MenuCreateRequest struct {
	MenuName  string
	ParentID  *string
	OrderNum  int
	Path      *string
	Component *string
	MenuType  models.MenuType
	Visible   models.VisibleType
	Status    models.MenuStatus
	Perms     *string
	Icon      *string
	Remark    *string
}

// ToModel 转换为菜单模型
func (r *MenuCreateRequest) ToModel() models.Menu {
	return models.Menu{
		MenuName:  r.MenuName,
		ParentID:  normalizeParentID(r.ParentID),
		OrderNum:  r.OrderNum,
		Path:      r.Path,
		Component: r.Component,
		MenuType:  r.MenuType,
		Visible:   r.Visible,
		Status:    r.Status,
		Perms:     r.Perms,
		Icon:      r.Icon,
		Remark:    stringPtrValue(r.Remark),
	}
}

// MenuUpdateRequest 更新菜单请求
type MenuUpdateRequest struct {
	ID        string
	MenuName  string
	ParentID  *string
	OrderNum  int
	Path      *string
	Component *string
	MenuType  models.MenuType
	Visible   models.VisibleType
	Status    models.MenuStatus
	Perms     *string
	Icon      *string
	Remark    *string
}

// ==================== 服务方法实现 ====================

// Create 创建菜单
func (s *menuService) Create(ctx context.Context, req *MenuCreateRequest) error {
	// 处理空字符串的 ParentID
	req.ParentID = normalizeParentID(req.ParentID)

	// 检查菜单名称是否已存在（同级下）
	if exists, err := s.checkMenuNameExists(ctx, req.ParentID, req.MenuName, ""); err != nil {
		return fmt.Errorf("检查菜单名称失败: %w", err)
	} else if exists {
		return fmt.Errorf("同级菜单名称已存在")
	}

	menu := req.ToModel()

	if err := s.db.WithContext(ctx).Create(&menu).Error; err != nil {
		return fmt.Errorf("创建菜单失败: %w", err)
	}

	return nil
}

// Update 更新菜单
func (s *menuService) Update(ctx context.Context, req *MenuUpdateRequest) error {
	// 检查菜单是否存在
	var menu models.Menu
	if err := s.db.WithContext(ctx).First(&menu, "id = ?", req.ID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("菜单不存在")
		}
		return fmt.Errorf("查询菜单失败: %w", err)
	}

	// 处理空字符串的 ParentID
	req.ParentID = normalizeParentID(req.ParentID)

	// 检查菜单名称是否已存在（同级下，排除自己）
	if exists, err := s.checkMenuNameExists(ctx, req.ParentID, req.MenuName, req.ID); err != nil {
		return fmt.Errorf("检查菜单名称失败: %w", err)
	} else if exists {
		return fmt.Errorf("同级菜单名称已存在")
	}

	// 更新菜单信息
	menu.MenuName = req.MenuName
	menu.ParentID = req.ParentID
	menu.OrderNum = req.OrderNum
	menu.Path = req.Path
	menu.Component = req.Component
	menu.MenuType = req.MenuType
	menu.Visible = req.Visible
	menu.Status = req.Status
	menu.Perms = req.Perms
	menu.Icon = req.Icon
	menu.Remark = stringPtrValue(req.Remark)

	if err := s.db.WithContext(ctx).Save(&menu).Error; err != nil {
		return fmt.Errorf("更新菜单失败: %w", err)
	}

	return nil
}

// Delete 删除菜单
func (s *menuService) Delete(ctx context.Context, id string) error {
	// 检查菜单是否存在
	var menu models.Menu
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&menu).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("菜单不存在")
		}
		return fmt.Errorf("查询菜单失败: %w", err)
	}

	// 检查是否有子菜单
	var count int64
	if err := s.db.WithContext(ctx).Model(&models.Menu{}).Where("parent_id = ?", id).Count(&count).Error; err != nil {
		return fmt.Errorf("检查子菜单失败: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("存在子菜单，无法删除")
	}

	if err := s.db.WithContext(ctx).Delete(&menu).Error; err != nil {
		return fmt.Errorf("删除菜单失败: %w", err)
	}

	return nil
}

// GetByID 根据ID获取菜单
func (s *menuService) GetByID(ctx context.Context, id string) (*models.Menu, error) {
	var menu models.Menu
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&menu).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("菜单不存在")
		}
		return nil, fmt.Errorf("查询菜单失败: %w", err)
	}
	return &menu, nil
}

// GetTree 获取菜单树
func (s *menuService) GetTree(ctx context.Context) ([]models.Menu, error) {
	var menus []models.Menu
	if err := s.db.WithContext(ctx).Order("order_num ASC").Find(&menus).Error; err != nil {
		return nil, fmt.Errorf("查询菜单列表失败: %w", err)
	}

	// 构建树形结构
	menuTree := s.buildMenuTree(menus, nil)
	return menuTree, nil
}

// List 查询菜单列表
func (s *menuService) List(ctx context.Context, params MenuListParams) ([]models.Menu, error) {
	var menus []models.Menu
	query := s.db.WithContext(ctx).Model(&models.Menu{})

	if params.MenuName != "" {
		query = query.Where("menu_name LIKE ?", "%"+params.MenuName+"%")
	}

	if params.Status != "" {
		if status, err := strconv.Atoi(params.Status); err == nil {
			query = query.Where("status = ?", status)
		}
	}

	if err := query.Order("order_num ASC").Find(&menus).Error; err != nil {
		return nil, fmt.Errorf("查询菜单列表失败: %w", err)
	}

	return menus, nil
}

// BatchDelete 批量删除菜单
func (s *menuService) BatchDelete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return fmt.Errorf("ids不能为空")
	}

	// 检查是否有子菜单
	var count int64
	if err := s.db.WithContext(ctx).Model(&models.Menu{}).Where("parent_id IN ?", ids).Count(&count).Error; err != nil {
		return fmt.Errorf("检查子菜单失败: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("存在子菜单，无法删除")
	}

	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Delete(&models.Menu{}).Error; err != nil {
		return fmt.Errorf("批量删除菜单失败: %w", err)
	}

	return nil
}

// UpdateStatus 更新菜单状态
func (s *menuService) UpdateStatus(ctx context.Context, id string, status int) error {
	// 检查菜单是否存在
	var menu models.Menu
	if err := s.db.WithContext(ctx).First(&menu, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("菜单不存在")
		}
		return fmt.Errorf("查询菜单失败: %w", err)
	}

	// 更新状态
	if err := s.db.WithContext(ctx).Model(&menu).Update("status", status).Error; err != nil {
		return fmt.Errorf("更新菜单状态失败: %w", err)
	}

	return nil
}

// ==================== 私有辅助方法 ====================

// buildMenuTree 构建菜单树
func (s *menuService) buildMenuTree(menus []models.Menu, parentID *string) []models.Menu {
	var tree []models.Menu
	for _, menu := range menus {
		if (parentID == nil && menu.ParentID == nil) || (parentID != nil && menu.ParentID != nil && *menu.ParentID == *parentID) {
			children := s.buildMenuTree(menus, &menu.ID)
			if len(children) > 0 {
				menu.Children = children
			}
			tree = append(tree, menu)
		}
	}
	return tree
}

// checkMenuNameExists 检查菜单名称是否存在
func (s *menuService) checkMenuNameExists(ctx context.Context, parentID *string, menuName string, excludeID string) (bool, error) {
	var count int64
	query := s.db.WithContext(ctx).Model(&models.Menu{}).Where("menu_name = ?", menuName)

	if parentID != nil {
		query = query.Where("parent_id = ?", *parentID)
	} else {
		query = query.Where("parent_id IS NULL")
	}

	if excludeID != "" {
		query = query.Where("id != ?", excludeID)
	}

	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// stringPtrValue 安全地从字符串指针获取值
func stringPtrValue(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}

// normalizeParentID 处理空字符串的 ParentID
func normalizeParentID(parentID *string) *string {
	if parentID != nil && *parentID == "" {
		return nil
	}
	return parentID
}

// GetUserMenus 获取用户的菜单列表
func (s *menuService) GetUserMenus(ctx context.Context, userID string) ([]models.Menu, error) {
	// 获取用户的角色
	var userRoles []models.UserRole
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).Find(&userRoles).Error; err != nil {
		return nil, fmt.Errorf("查询用户角色失败: %w", err)
	}

	if len(userRoles) == 0 {
		return []models.Menu{}, nil
	}

	// 提取角色ID
	roleIDs := make([]string, len(userRoles))
	for i, ur := range userRoles {
		roleIDs[i] = ur.RoleID
	}

	// 获取角色关联的菜单ID（去重）
	var roleMenus []models.RoleMenu
	if err := s.db.WithContext(ctx).Distinct("menu_id").Where("role_id IN ?", roleIDs).Find(&roleMenus).Error; err != nil {
		return nil, fmt.Errorf("查询角色菜单关联失败: %w", err)
	}

	if len(roleMenus) == 0 {
		return []models.Menu{}, nil
	}

	// 提取菜单ID
	menuIDs := make([]string, len(roleMenus))
	for i, rm := range roleMenus {
		menuIDs[i] = rm.MenuID
	}

	// 获取菜单详情（只获取启用且可见的菜单）
	var menus []models.Menu
	if err := s.db.WithContext(ctx).
		Where("id IN ? AND status = ? AND visible = ?", menuIDs, int(models.MenuStatusNormal), int(models.VisibleShow)).
		Order("order_num ASC").
		Find(&menus).Error; err != nil {
		return nil, fmt.Errorf("查询菜单列表失败: %w", err)
	}

	// 构建树形结构
	menuTree := s.buildMenuTree(menus, nil)
	return menuTree, nil
}

// GetAllUserMenus 获取用户的所有菜单（包含隐藏菜单）
func (s *menuService) GetAllUserMenus(ctx context.Context, userID string) ([]models.Menu, error) {
	// 获取用户的角色
	var userRoles []models.UserRole
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).Find(&userRoles).Error; err != nil {
		return nil, fmt.Errorf("查询用户角色失败: %w", err)
	}

	if len(userRoles) == 0 {
		return []models.Menu{}, nil
	}

	// 提取角色ID
	roleIDs := make([]string, len(userRoles))
	for i, ur := range userRoles {
		roleIDs[i] = ur.RoleID
	}

	// 获取角色关联的菜单ID（去重）
	var roleMenus []models.RoleMenu
	if err := s.db.WithContext(ctx).Distinct("menu_id").Where("role_id IN ?", roleIDs).Find(&roleMenus).Error; err != nil {
		return nil, fmt.Errorf("查询角色菜单关联失败: %w", err)
	}

	if len(roleMenus) == 0 {
		return []models.Menu{}, nil
	}

	// 提取菜单ID
	menuIDs := make([]string, len(roleMenus))
	for i, rm := range roleMenus {
		menuIDs[i] = rm.MenuID
	}

	// 获取菜单详情（只获取启用状态的菜单，包含隐藏菜单）
	var menus []models.Menu
	if err := s.db.WithContext(ctx).
		Where("id IN ? AND status = ?", menuIDs, int(models.MenuStatusNormal)).
		Order("order_num ASC").
		Find(&menus).Error; err != nil {
		return nil, fmt.Errorf("查询菜单列表失败: %w", err)
	}

	// 构建树形结构
	menuTree := s.buildMenuTree(menus, nil)
	return menuTree, nil
}

// GetUserPermissions 获取用户的权限列表
func (s *menuService) GetUserPermissions(ctx context.Context, userID string) ([]string, error) {
	// 获取用户的角色
	var userRoles []models.UserRole
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).Find(&userRoles).Error; err != nil {
		return nil, fmt.Errorf("查询用户角色失败: %w", err)
	}

	if len(userRoles) == 0 {
		return []string{}, nil
	}

	// 提取角色ID
	roleIDs := make([]string, len(userRoles))
	for i, ur := range userRoles {
		roleIDs[i] = ur.RoleID
	}

	// 获取角色关联的菜单ID（去重）
	var roleMenus []models.RoleMenu
	if err := s.db.WithContext(ctx).Distinct("menu_id").Where("role_id IN ?", roleIDs).Find(&roleMenus).Error; err != nil {
		return nil, fmt.Errorf("查询角色菜单关联失败: %w", err)
	}

	if len(roleMenus) == 0 {
		return []string{}, nil
	}

	// 提取菜单ID
	menuIDs := make([]string, len(roleMenus))
	for i, rm := range roleMenus {
		menuIDs[i] = rm.MenuID
	}

	// 获取菜单的权限标识（perms字段，不为空的去重）
	var perms []string
	if err := s.db.WithContext(ctx).
		Model(&models.Menu{}).
		Where("id IN ? AND perms IS NOT NULL AND perms != ''", menuIDs).
		Pluck("perms", &perms).Error; err != nil {
		return nil, fmt.Errorf("查询权限列表失败: %w", err)
	}

	// 去重
	permissionMap := make(map[string]bool)
	for _, perm := range perms {
		if perm != "" {
			permissionMap[perm] = true
		}
	}

	// 转换为切片
	result := make([]string, 0, len(permissionMap))
	for perm := range permissionMap {
		result = append(result, perm)
	}

	return result, nil
}

// GetRoleMenuIDs 获取角色的菜单ID列表
func (s *menuService) GetRoleMenuIDs(ctx context.Context, roleID string, menuIDs *[]string) error {
	return s.db.WithContext(ctx).
		Table("sys_role_menu").
		Where("role_id = ?", roleID).
		Pluck("menu_id", menuIDs).Error
}
