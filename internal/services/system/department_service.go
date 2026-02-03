package system

import (
	"context"
	"fmt"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models/system/requests"
	"gorm.io/gorm"
)

// DepartmentService 部门服务接口
type DepartmentService interface {
	Create(ctx context.Context, req *requests.DepartmentCreateRequest) error
	Update(ctx context.Context, req *requests.DepartmentUpdateRequest) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*models.Department, error)
	GetTree(ctx context.Context, includeDisabled bool) ([]*models.Department, error)
	List(ctx context.Context, params requests.DepartmentListParams) ([]models.Department, error)
	BatchDelete(ctx context.Context, ids []string) error
	UpdateStatus(ctx context.Context, id string, status int) error
	GetRoleDeptIDs(ctx context.Context, roleID string, deptIDs *[]string) error
	GetDB() *gorm.DB

	// 新增缓存方法
	GetTreeWithCache(ctx context.Context, includeDisabled bool) ([]*models.Department, error)
	GetSelectDataWithCache(ctx context.Context) ([]*models.Department, error)
	InvalidateDeptCache(ctx context.Context) error
}

// departmentService 部门服务实现
type departmentService struct {
	db *gorm.DB
}

// NewDepartmentService 创建部门服务实例
func NewDepartmentService(db *gorm.DB) DepartmentService {
	return &departmentService{db: db}
}

// ==================== 服务方法实现 ====================

// Create 创建部门
func (s *departmentService) Create(ctx context.Context, req *requests.DepartmentCreateRequest) error {
	// 构建祖先链
	ancestors, err := s.buildAncestors(ctx, req.ParentID)
	if err != nil {
		return fmt.Errorf("构建祖先链失败: %w", err)
	}

	// 检查部门名称是否已存在（同级下）
	if exists, err := s.checkDeptNameExists(ctx, req.ParentID, req.DeptName, ""); err != nil {
		return fmt.Errorf("检查部门名称失败: %w", err)
	} else if exists {
		return fmt.Errorf("同级部门名称已存在")
	}

	dept := req.ToModel(ancestors)

	if err := s.db.WithContext(ctx).Create(&dept).Error; err != nil {
		return fmt.Errorf("创建部门失败: %w", err)
	}

	return nil
}

// Update 更新部门
func (s *departmentService) Update(ctx context.Context, req *requests.DepartmentUpdateRequest) error {
	// 检查部门是否存在
	var dept models.Department
	if err := s.db.WithContext(ctx).First(&dept, "id = ?", req.ID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("部门不存在")
		}
		return fmt.Errorf("查询部门失败: %w", err)
	}

	// 如果修改了父部门，需要重新构建祖先链
	if (req.ParentID == nil && dept.ParentID != nil) || (req.ParentID != nil && dept.ParentID == nil) ||
		(req.ParentID != nil && dept.ParentID != nil && *req.ParentID != *dept.ParentID) {
		ancestors, err := s.buildAncestors(ctx, req.ParentID)
		if err != nil {
			return fmt.Errorf("构建祖先链失败: %w", err)
		}
		dept.Ancestors = ancestors
	}

	// 检查部门名称是否已存在（同级下，排除自己）
	if exists, err := s.checkDeptNameExists(ctx, req.ParentID, req.DeptName, req.ID); err != nil {
		return fmt.Errorf("检查部门名称失败: %w", err)
	} else if exists {
		return fmt.Errorf("同级部门名称已存在")
	}

	// 更新部门信息
	dept.DeptName = req.DeptName
	dept.ParentID = req.ParentID
	dept.OrderNum = req.OrderNum
	dept.Leader = req.Leader
	dept.Phone = req.Phone
	dept.Email = req.Email
	dept.IsExternalOrg = req.IsExternalOrg
	dept.Status = req.Status
	dept.Remark = toStringPtr(req.Remark)

	if err := s.db.WithContext(ctx).Save(&dept).Error; err != nil {
		return fmt.Errorf("更新部门失败: %w", err)
	}

	return nil
}

// Delete 删除部门
func (s *departmentService) Delete(ctx context.Context, id string) error {
	// 检查部门是否存在
	var dept models.Department
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&dept).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("部门不存在")
		}
		return fmt.Errorf("查询部门失败: %w", err)
	}

	// 检查是否有子部门
	var count int64
	if err := s.db.WithContext(ctx).Model(&models.Department{}).Where("parent_id = ?", id).Count(&count).Error; err != nil {
		return fmt.Errorf("检查子部门失败: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("存在子部门，无法删除")
	}

	// 检查是否有用户
	if err := s.db.WithContext(ctx).Model(&models.User{}).Where("dept_id = ?", id).Count(&count).Error; err != nil {
		return fmt.Errorf("检查用户失败: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("部门下存在用户，无法删除")
	}

	if err := s.db.WithContext(ctx).Delete(&dept).Error; err != nil {
		return fmt.Errorf("删除部门失败: %w", err)
	}

	return nil
}

// GetByID 根据ID获取部门
func (s *departmentService) GetByID(ctx context.Context, id string) (*models.Department, error) {
	var dept models.Department
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&dept).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("部门不存在")
		}
		return nil, fmt.Errorf("查询部门失败: %w", err)
	}
	return &dept, nil
}

// GetTree 获取部门树
func (s *departmentService) GetTree(ctx context.Context, includeDisabled bool) ([]*models.Department, error) {
	var depts []models.Department
	query := s.db.WithContext(ctx).Model(&models.Department{})

	if !includeDisabled {
		query = query.Where("status = ?", models.DeptStatusNormal)
	}

	if err := query.Order("order_num ASC").Find(&depts).Error; err != nil {
		return nil, fmt.Errorf("查询部门列表失败: %w", err)
	}

	// 填充负责人信息
	s.fillLeaderInfo(ctx, depts)

	// 构建树形结构
	deptTree := s.buildDeptTree(depts, nil)
	return deptTree, nil
}

// List 查询部门列表
func (s *departmentService) List(ctx context.Context, params requests.DepartmentListParams) ([]models.Department, error) {
	var depts []models.Department
	query := s.db.WithContext(ctx).Model(&models.Department{})

	if params.DeptName != "" {
		query = query.Where("dept_name LIKE ?", "%"+params.DeptName+"%")
	}

	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	}

	if err := query.Order("order_num ASC").Find(&depts).Error; err != nil {
		return nil, fmt.Errorf("查询部门列表失败: %w", err)
	}

	// 填充负责人信息
	s.fillLeaderInfo(ctx, depts)

	return depts, nil
}

// BatchDelete 批量删除部门
func (s *departmentService) BatchDelete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return fmt.Errorf("ids不能为空")
	}

	// 检查是否有子部门
	var count int64
	if err := s.db.WithContext(ctx).Model(&models.Department{}).Where("parent_id IN ?", ids).Count(&count).Error; err != nil {
		return fmt.Errorf("检查子部门失败: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("存在子部门，无法删除")
	}

	// 检查是否有用户
	if err := s.db.WithContext(ctx).Model(&models.User{}).Where("dept_id IN ?", ids).Count(&count).Error; err != nil {
		return fmt.Errorf("检查用户失败: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("部分部门下存在用户，无法删除")
	}

	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Delete(&models.Department{}).Error; err != nil {
		return fmt.Errorf("批量删除部门失败: %w", err)
	}

	return nil
}

// UpdateStatus 更新部门状态
func (s *departmentService) UpdateStatus(ctx context.Context, id string, status int) error {
	// 检查部门是否存在
	var dept models.Department
	if err := s.db.WithContext(ctx).First(&dept, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("部门不存在")
		}
		return fmt.Errorf("查询部门失败: %w", err)
	}

	// 更新状态
	if err := s.db.WithContext(ctx).Model(&dept).Update("status", status).Error; err != nil {
		return fmt.Errorf("更新部门状态失败: %w", err)
	}

	return nil
}

// ==================== 私有辅助方法 ====================

// buildDeptTree 构建部门树
func (s *departmentService) buildDeptTree(depts []models.Department, parentID *string) []*models.Department {
	var tree []*models.Department
	for _, dept := range depts {
		// 检查是否为父级部门
		isParent := false
		if parentID == nil {
			// 查找顶级部门：parentID 为 nil 或 ancestors 为空
			isParent = dept.ParentID == nil || dept.Ancestors == ""
		} else {
			// 查找子部门：parentID 匹配
			isParent = dept.ParentID != nil && *dept.ParentID == *parentID
		}

		if isParent {
			children := s.buildDeptTree(depts, &dept.ID)
			if len(children) > 0 {
				dept.Children = children
			}
			// 复制部门对象到指针
			deptCopy := dept
			tree = append(tree, &deptCopy)
		}
	}
	return tree
}

// buildAncestors 构建祖先链
func (s *departmentService) buildAncestors(ctx context.Context, parentID *string) (string, error) {
	if parentID == nil || *parentID == "" {
		return "", nil
	}

	var parentDept models.Department
	if err := s.db.WithContext(ctx).First(&parentDept, "id = ?", *parentID).Error; err != nil {
		return "", fmt.Errorf("查询父部门失败: %w", err)
	}

	if parentDept.Ancestors != "" {
		return parentDept.Ancestors + "," + *parentID, nil
	}

	return *parentID, nil
}

// checkDeptNameExists 检查部门名称是否存在
func (s *departmentService) checkDeptNameExists(ctx context.Context, parentID *string, deptName string, excludeID string) (bool, error) {
	var count int64
	query := s.db.WithContext(ctx).Model(&models.Department{}).Where("dept_name = ?", deptName)

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

// GetRoleDeptIDs 获取角色的部门ID列表
func (s *departmentService) GetRoleDeptIDs(ctx context.Context, roleID string, deptIDs *[]string) error {
	return s.db.WithContext(ctx).
		Table("sys_role_dept").
		Where("role_id = ?", roleID).
		Pluck("dept_id", deptIDs).Error
}

// GetTreeWithCache 获取部门树（无缓存版本，直接查询数据库）
func (s *departmentService) GetTreeWithCache(ctx context.Context, includeDisabled bool) ([]*models.Department, error) {
	return s.GetTree(ctx, includeDisabled)
}

// GetSelectDataWithCache 获取选择器数据（无缓存版本，直接查询数据库）
func (s *departmentService) GetSelectDataWithCache(ctx context.Context) ([]*models.Department, error) {
	return s.GetTree(ctx, false)
}

// InvalidateDeptCache 失效部门缓存（无缓存版本，空操作）
func (s *departmentService) InvalidateDeptCache(ctx context.Context) error {
	return nil
}

// GetDB 获取数据库连接
func (s *departmentService) GetDB() *gorm.DB {
	return s.db
}

// fillLeaderInfo 填充部门负责人信息
// 批量查询所有负责人信息，避免N+1查询问题
func (s *departmentService) fillLeaderInfo(ctx context.Context, depts []models.Department) {
	// 收集所有负责人ID
	var leaderIDs []string
	for _, dept := range depts {
		if dept.Leader != nil && *dept.Leader != "" {
			leaderIDs = append(leaderIDs, *dept.Leader)
		}
	}

	if len(leaderIDs) == 0 {
		return
	}

	// 批量查询负责人信息
	var leaders []models.User
	if err := s.db.WithContext(ctx).
		Select("id, username, nickname").
		Where("id IN ?", leaderIDs).
		Find(&leaders).Error; err != nil {
		return
	}

	// 构建ID到用户的映射
	leaderMap := make(map[string]*models.User)
	for i := range leaders {
		leaderMap[leaders[i].ID] = &leaders[i]
	}

	// 填充负责人信息
	for i := range depts {
		if depts[i].Leader != nil && *depts[i].Leader != "" {
			if leader, exists := leaderMap[*depts[i].Leader]; exists {
				depts[i].LeaderName = leader.Nickname
				depts[i].LeaderUsername = &leader.Username
			}
		}
	}
}
