package system

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"github.com/xingran-next/xingran-go-backend/pkg/datascope"
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
	UpdateStatus(ctx context.Context, id string, status models.RoleStatus) error

	// 新增缓存方法
	GetAllEnabledWithCache(ctx context.Context) ([]*models.Role, error)
	GetMenusWithCache(ctx context.Context, roleID string) ([]models.Menu, error)
	GetDeptsWithCache(ctx context.Context, roleID string) ([]models.Department, error)
	InvalidateRoleCache(ctx context.Context, roleID string) error

	// PreviewAnchor 机构锚点推导预览（契约 §六，只读）。
	PreviewAnchor(ctx context.Context, roleID string) (*AnchorPreviewResponse, error)
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
			fmt.Sprintf("SUM(CASE WHEN status = %d THEN 1 ELSE 0 END) AS active", int(models.RoleStatusEnabled)),
			fmt.Sprintf("SUM(CASE WHEN status = %d THEN 1 ELSE 0 END) AS inactive", int(models.RoleStatusDisabled)),
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
	Active   int64 `json:"active"`   // RoleStatusEnabled
	Inactive int64 `json:"inactive"` // RoleStatusDisabled
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

	// 配置期校验（契约 §五 防静默第 1 道防线）：anchor_mode 只允许 0/2/3/5；
	// mode=2 的 dept 必须存在、mode=3 必须有外部机构、mode=5 的层级必须有节点。
	// 必须在写库**之前**拒绝 —— 锚点算不出来时运行期只会 fail-closed 空集。
	if err := validateAnchorForWrite(ctx, s.db, req.AnchorMode, req.AnchorParam); err != nil {
		return err
	}

	// 使用事务创建角色及其关联
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		newRole := models.Role{
			RoleName:          req.RoleName,
			RoleKey:           req.RoleKey,
			RoleSort:          req.RoleSort,
			DataScope:         req.DataScope,
			AnchorMode:        models.AnchorMode(req.AnchorMode),
			AnchorParam:       sanitizeAnchorParam(req.AnchorMode, req.AnchorParam),
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

	// 配置期校验（同 Create）：保存前拒绝非法锚点配置（契约 §五）。
	if err := validateAnchorForWrite(ctx, s.db, req.AnchorMode, req.AnchorParam); err != nil {
		return err
	}

	// 使用事务更新角色及其关联
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 更新角色基本信息
		role.RoleName = req.RoleName
		role.RoleKey = req.RoleKey
		role.RoleSort = req.RoleSort
		role.DataScope = req.DataScope
		role.AnchorMode = models.AnchorMode(req.AnchorMode)
		role.AnchorParam = sanitizeAnchorParam(req.AnchorMode, req.AnchorParam)
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
func (s *roleService) UpdateStatus(ctx context.Context, id string, status models.RoleStatus) error {
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

// ---------------------------------------------------------------------------
// 机构锚点配置期校验与归一化（角色/岗位共用，见契约 §五）
// ---------------------------------------------------------------------------

// validateAnchorForWrite 写库前的锚点配置校验（防静默第 1 道防线）。
//
// 内部委托 pkg/datascope.ValidateAnchor（契约要求的唯一校验实现），失败一律转
// 400 拒绝保存：mode=1/4（弃用编号）、mode=2 参数空/部门不存在、mode=3 参数非空/
// 库里没有外部机构、mode=5 层级非法/库里没有该层级节点。
//
// 说明：ValidateAnchor 在统计计数时若底层 SQL 失败也会返回 error，这里同样落
// 400。这是刻意选择 —— 锚点配置是**管理员在配置面**的操作，任何"配置存不下去"
// 的必须让用户在保存处立刻看到原文，而不是静默存进去、运行期再 fail-closed。
func validateAnchorForWrite(ctx context.Context, db *gorm.DB, mode int, param string) error {
	if err := datascope.ValidateAnchor(ctx, db, mode, param); err != nil {
		// 契约 §八：校验失败 handler 必须 400 拒绝保存。
		return apperrors.BadRequest(err.Error())
	}
	return nil
}

// sanitizeAnchorParam 归一化入库的锚点参数。
//
// mode=0（不参与）与 mode=3（外部机构，走 ancestors 动态定位）**恒为空串**：
// 否则用户从「指定部门/指定层级」切到「外部机构」时，上一次的 param 会残留在
// sys_role/sys_post 上，运维排查时看到「模式=外部机构但参数还有值」的脏数据。
func sanitizeAnchorParam(mode int, param string) string {
	switch models.AnchorMode(mode) {
	case models.AnchorModeNone, models.AnchorModeExternalOrg:
		return ""
	default:
		return strings.TrimSpace(param)
	}
}

// ---------------------------------------------------------------------------
// 机构锚点推导预览（契约 §六，防静默第 2 道防线）
// ---------------------------------------------------------------------------

// AnchorPreviewResponse 锚点推导预览的对外响应结构（契约 §六）。
//
// 字段口径说明：
//   - 契约原文举例用 `unresolved[]`，而 pkg/datascope.AnchorPreviewResult 用
//     `items[]`（每项带 resolved 布尔）。两者**都保留**：`items` 是全量逐人结果，
//     `unresolved` 只含未命中的项（等价于 items 中 resolved=false 的子集）。
//   - resolvedCount / unresolvedCount / totalCount / mode / param 恒在 JSON 中，
//     前端不依赖 items 自己数。
//   - warnings 可选（omitempty）：目前用于提示「同一个目标机构下有多个任职人」，
//     这类配置通常意味着岗位模板被重复授予，值得人工看一眼。
type AnchorPreviewResponse struct {
	Mode            int                           `json:"mode"`
	Param           string                        `json:"param"`
	ResolvedCount   int                           `json:"resolvedCount"`
	UnresolvedCount int                           `json:"unresolvedCount"`
	TotalCount      int                           `json:"totalCount"`
	Items           []datascope.AnchorPreviewItem `json:"items"`
	Unresolved      []datascope.AnchorPreviewItem `json:"unresolved"`
	Warnings        []string                      `json:"warnings,omitempty"`
}

// anchorReasonNotConfigured mode=0（不参与机构授权）时的逐人说明。
//
// datascope.ResolveAnchor 对 AnchorModeNone 返回空 reason（那是"正常未命中"），
// 预览页若原样展示会被误读成"解析失败"；这里改成明确文案。
const anchorReasonNotConfigured = "该角色/岗位未配置机构锚点（anchorMode=0），不产生机构范围授权"

// PreviewAnchor 计算「角色当前锚点配置对全部任职人是否命中」（角色详情页预览）。
func (s *roleService) PreviewAnchor(ctx context.Context, roleID string) (*AnchorPreviewResponse, error) {
	// 直接查库而非 GetByID：缓存版 GetByID 有 30min TTL，刚保存锚点后预览会
	// 显示旧配置，与"保存后立刻预览"的使用方式冲突。
	var role models.Role
	if err := s.db.WithContext(ctx).First(&role, "id = ?", roleID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.RoleNotFound()
		}
		return nil, fmt.Errorf("查询角色失败: %w", err)
	}

	ids, err := listRoleAssigneeIDs(ctx, s.db, roleID)
	if err != nil {
		return nil, err
	}

	return buildAnchorPreview(ctx, s.db, role.AnchorMode, role.AnchorParam, ids)
}

// listRoleAssigneeIDs 取角色**全部**任职人（契约 §六：不看 status）。
//
// 停用用户也可能被重新启用，且预览是"配置面体检"而非"当前生效面"，
// 故与 ResolveAnchorGrants（仅启用角色/岗位）口径不同，这里不筛 status。
func listRoleAssigneeIDs(ctx context.Context, db *gorm.DB, roleID string) ([]string, error) {
	var rows []struct{ UserID string }
	if err := db.WithContext(ctx).Raw(
		`SELECT CAST(user_id AS TEXT) AS user_id
		   FROM sys_user_role
		  WHERE CAST(role_id AS TEXT) = ?`,
		roleID,
	).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("查询角色任职人失败: %w", err)
	}
	return normalizeUserIDs(rows), nil
}

// buildAnchorPreview 组装契约 §六 的响应（角色/岗位共用）。
func buildAnchorPreview(
	ctx context.Context,
	db *gorm.DB,
	mode models.AnchorMode,
	param string,
	userIDs []string,
) (*AnchorPreviewResponse, error) {

	usernames, nicknames, err := queryAssigneeNames(ctx, db, userIDs)
	if err != nil {
		return nil, err
	}

	preview, err := datascope.PreviewAnchor(ctx, db, mode, param, userIDs, usernames, nicknames)
	if err != nil {
		return nil, fmt.Errorf("解析机构锚点失败: %w", err)
	}

	// mode=0：把空 reason 换成明确文案，避免前端显示成"解析失败"。
	if mode == models.AnchorModeNone {
		for i := range preview.Items {
			if !preview.Items[i].Resolved {
				preview.Items[i].Reason = anchorReasonNotConfigured
			}
		}
	}

	resp := &AnchorPreviewResponse{
		Mode:            preview.Mode,
		Param:           preview.Param,
		ResolvedCount:   preview.ResolvedCount,
		UnresolvedCount: preview.UnresolvedCount,
		TotalCount:      preview.TotalCount,
		Items:           preview.Items,
		Unresolved:      make([]datascope.AnchorPreviewItem, 0, preview.UnresolvedCount),
	}

	for _, item := range preview.Items {
		if !item.Resolved {
			resp.Unresolved = append(resp.Unresolved, item)
		}
	}

	// warnings：动态锚点（3/5）下多个任职人落到**同一个**目标机构 —— 通常意味着
	// 岗位模板被重复授予（例如同一中支的两个人共持"中支管理员"）。模式 2 是显式
	// 同一部门，属预期，故不告警。
	if mode != models.AnchorModeDept {
		byAnchor := make(map[string]int, len(preview.Items))
		for _, item := range preview.Items {
			if item.Resolved && item.AnchorDeptID != "" {
				byAnchor[item.AnchorDeptID]++
			}
		}
		for _, item := range preview.Items {
			if item.Resolved && byAnchor[item.AnchorDeptID] > 1 {
				resp.Warnings = append(resp.Warnings, fmt.Sprintf(
					"锚点部门 %s 下有 %d 名任职人共用同一机构范围，请确认是否为有意配置",
					item.AnchorDeptID, byAnchor[item.AnchorDeptID]))
				delete(byAnchor, item.AnchorDeptID) // 每个锚点只告警一次
			}
		}
	}

	return resp, nil
}

// queryAssigneeNames 批量取任职人的 username / nickname 映射（PreviewAnchor 入参）。
//
// CAST(... AS TEXT) 保证 PG(uuid) 与 SQLite 行为一致（契约 C7）。
func queryAssigneeNames(ctx context.Context, db *gorm.DB, userIDs []string) (map[string]string, map[string]string, error) {
	usernames := make(map[string]string, len(userIDs))
	nicknames := make(map[string]string, len(userIDs))
	if len(userIDs) == 0 {
		return usernames, nicknames, nil
	}

	var rows []struct {
		ID       string
		Username string
		Nickname string
	}
	if err := db.WithContext(ctx).Raw(
		`SELECT CAST(id AS TEXT) AS id, username, COALESCE(nickname, '') AS nickname
		   FROM sys_user
		  WHERE CAST(id AS TEXT) IN ?`,
		userIDs,
	).Scan(&rows).Error; err != nil {
		return nil, nil, fmt.Errorf("查询任职人身份信息失败: %w", err)
	}
	for _, r := range rows {
		usernames[r.ID] = r.Username
		nicknames[r.ID] = r.Nickname
	}
	return usernames, nicknames, nil
}

// normalizeUserIDs 把 Scan 出来的行转成去重排序的 ID 切片。
func normalizeUserIDs(rows []struct{ UserID string }) []string {
	if len(rows) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(rows))
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		id := strings.TrimSpace(r.UserID)
		if id == "" {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// toStringPtr 函数已在 user_service.go 中定义
