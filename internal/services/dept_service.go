package services

import (
	"context"
	"fmt"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// DeptService 部门服务（带缓存）
type DeptService struct {
	db        *gorm.DB
	dataCache *DataCacheService
}

// NewDeptService 创建部门服务
func NewDeptService(db *gorm.DB, dataCache *DataCacheService) *DeptService {
	return &DeptService{
		db:        db,
		dataCache: dataCache,
	}
}

// GetDeptTreeWithCache 获取部门树（带缓存）
func (s *DeptService) GetDeptTreeWithCache(ctx context.Context, includeDisabled bool) ([]*models.Department, error) {
	cacheKey := CacheKeyDeptTree
	if includeDisabled {
		cacheKey += ":all"
	}

	var depts []*models.Department

	// 使用动态配置的缓存时间
	expiration := s.dataCache.GetExpiration(CacheConfigDeptTree, 30*time.Minute)

	// 尝试从缓存获取
	err := s.dataCache.GetOrSet(ctx, cacheKey, &depts, expiration, func() (interface{}, error) {
		return s.queryDeptTree(includeDisabled)
	})

	if err != nil {
		return nil, fmt.Errorf("获取部门树失败: %w", err)
	}

	return depts, nil
}

// queryDeptTree 从数据库查询部门树
func (s *DeptService) queryDeptTree(includeDisabled bool) ([]*models.Department, error) {
	var depts []models.Department
	query := s.db.Model(&models.Department{})

	if !includeDisabled {
		query = query.Where("status = ?", models.DeptStatusNormal)
	}

	if err := query.Order("order_num ASC").Find(&depts).Error; err != nil {
		return nil, fmt.Errorf("查询部门列表失败: %w", err)
	}

	// 构建树形结构
	deptTree := s.buildDeptTree(depts, nil)
	return deptTree, nil
}

// buildDeptTree 构建部门树
func (s *DeptService) buildDeptTree(depts []models.Department, parentID *string) []*models.Department {
	var tree []*models.Department
	for _, dept := range depts {
		if (parentID == nil && dept.ParentID == nil) || (parentID != nil && dept.ParentID != nil && *dept.ParentID == *parentID) {
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

// GetDeptListWithCache 获取部门列表（带缓存）
func (s *DeptService) GetDeptListWithCache(ctx context.Context, deptName, status string) ([]models.Department, error) {
	// 对于有过滤条件的查询，不使用缓存（缓存命中率低）
	// 只对无条件查询使用缓存
	if deptName != "" || status != "" {
		return s.queryDeptList(deptName, status)
	}

	cacheKey := CacheKeyDeptList

	var depts []models.Department

	err := s.dataCache.GetOrSet(ctx, cacheKey, &depts, 30*time.Minute, func() (interface{}, error) {
		return s.queryDeptList("", "")
	})

	if err != nil {
		return nil, fmt.Errorf("获取部门列表失败: %w", err)
	}

	return depts, nil
}

// queryDeptList 从数据库查询部门列表
func (s *DeptService) queryDeptList(deptName, status string) ([]models.Department, error) {
	var depts []models.Department
	query := s.db.Model(&models.Department{})

	if deptName != "" {
		query = query.Where("dept_name LIKE ?", "%"+deptName+"%")
	}

	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Order("order_num ASC").Find(&depts).Error; err != nil {
		return nil, fmt.Errorf("查询部门列表失败: %w", err)
	}

	return depts, nil
}

// InvalidateDeptCache 清除部门相关缓存
func (s *DeptService) InvalidateDeptCache(ctx context.Context) error {
	// 清除所有部门相关的缓存
	patterns := []string{
		CacheKeyDeptTree + "*",
		CacheKeyDeptList + "*",
		CacheKeyDeptTreeSelect + "*",
	}

	for _, pattern := range patterns {
		if err := s.dataCache.DeleteByPattern(ctx, pattern); err != nil {
			return fmt.Errorf("清除缓存失败 %s: %w", pattern, err)
		}
	}

	return nil
}

// GetDeptByID 根据ID获取部门
func (s *DeptService) GetDeptByID(ctx context.Context, id string) (*models.Department, error) {
	var dept models.Department
	if err := s.db.Where("id = ?", id).First(&dept).Error; err != nil {
		return nil, fmt.Errorf("查询部门失败: %w", err)
	}
	return &dept, nil
}

// CreateDept 创建部门
func (s *DeptService) CreateDept(ctx context.Context, dept *models.Department) error {
	if err := s.db.Create(dept).Error; err != nil {
		return fmt.Errorf("创建部门失败: %w", err)
	}

	// 清除缓存
	if err := s.InvalidateDeptCache(ctx); err != nil {
		// 记录错误但不影响主流程
		logger.Warnf("[DEPT] 清除缓存失败: %v", err)
	}

	return nil
}

// UpdateDept 更新部门
func (s *DeptService) UpdateDept(ctx context.Context, dept *models.Department) error {
	if err := s.db.Save(dept).Error; err != nil {
		return fmt.Errorf("更新部门失败: %w", err)
	}

	// 清除缓存
	if err := s.InvalidateDeptCache(ctx); err != nil {
		logger.Warnf("[DEPT] 清除缓存失败: %v", err)
	}

	return nil
}

// DeleteDept 删除部门
func (s *DeptService) DeleteDept(ctx context.Context, id string) error {
	if err := s.db.Delete(&models.Department{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("删除部门失败: %w", err)
	}

	// 清除缓存
	if err := s.InvalidateDeptCache(ctx); err != nil {
		logger.Warnf("[DEPT] 清除缓存失败: %v", err)
	}

	return nil
}

// GetChildDeptIDs 获取所有子部门ID（递归）
func (s *DeptService) GetChildDeptIDs(parentID string) []string {
	var childIds []string
	var depts []models.Department

	if err := s.db.Where("parent_id = ?", parentID).Find(&depts).Error; err != nil {
		return childIds
	}

	for _, dept := range depts {
		childIds = append(childIds, dept.ID)
		// 递归获取子部门的子部门
		grandChildIds := s.GetChildDeptIDs(dept.ID)
		childIds = append(childIds, grandChildIds...)
	}

	return childIds
}

// BuildAncestors 构建祖先链
func (s *DeptService) BuildAncestors(parentID *string) (string, error) {
	if parentID == nil || *parentID == "" {
		return "", nil
	}

	var parentDept models.Department
	if err := s.db.First(&parentDept, "id = ?", *parentID).Error; err != nil {
		return "", fmt.Errorf("查询父部门失败: %w", err)
	}

	if parentDept.Ancestors != "" {
		return parentDept.Ancestors + "," + *parentID, nil
	}

	return *parentID, nil
}

// CheckDeptNameExists 检查部门名称是否存在
func (s *DeptService) CheckDeptNameExists(parentID *string, deptName string, excludeID string) (bool, error) {
	var count int64
	query := s.db.Model(&models.Department{}).Where("dept_name = ?", deptName)

	if parentID != nil {
		query = query.Where("parent_id = ?", *parentID)
	} else {
		query = query.Where("parent_id IS NULL")
	}

	if excludeID != "" {
		query = query.Where("id != ?", excludeID)
	}

	if err := query.Count(&count).Error; err != nil {
		return false, fmt.Errorf("检查部门名称失败: %w", err)
	}

	return count > 0, nil
}
