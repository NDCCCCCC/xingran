package workorder

import (
	"context"
	"fmt"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// CategoryService 分类服务
type CategoryService struct {
	db *gorm.DB
}

// NewCategoryService 创建分类服务
func NewCategoryService(db *gorm.DB) *CategoryService {
	return &CategoryService{db: db}
}

// GetTree 获取分类列表（树形结构）
func (s *CategoryService) GetTree(ctx context.Context) ([]models.WorkOrderCategory, error) {
	var list []models.WorkOrderCategory

	if err := s.db.WithContext(ctx).
		Where("parent_id IS NULL").
		Order("sort_order ASC, created_at ASC").
		Find(&list).Error; err != nil {
		return nil, fmt.Errorf("查询分类列表失败: %w", err)
	}

	// 加载子分类
	for i := range list {
		if err := s.loadChildren(ctx, &list[i]); err != nil {
			applogger.Warnf("加载工单分类 %s 的子分类失败: %v", list[i].ID, err)
		}
	}

	return list, nil
}

// loadChildren 递归加载子分类
func (s *CategoryService) loadChildren(ctx context.Context, category *models.WorkOrderCategory) error {
	var children []models.WorkOrderCategory

	if err := s.db.WithContext(ctx).
		Where("parent_id = ?", category.ID).
		Order("sort_order ASC, created_at ASC").
		Find(&children).Error; err != nil {
		return err
	}

	category.Children = children

	// 递归加载子分类的子分类
	for i := range children {
		if err := s.loadChildren(ctx, &children[i]); err != nil {
			applogger.Warnf("加载工单子分类 %s 递归失败: %v", children[i].ID, err)
		}
	}

	return nil
}

// GetByID 获取分类详情
func (s *CategoryService) GetByID(ctx context.Context, id string) (*models.WorkOrderCategory, error) {
	var category models.WorkOrderCategory

	if err := s.db.WithContext(ctx).
		Preload("Parent").
		Where("id = ?", id).
		First(&category).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("分类不存在")
		}
		return nil, fmt.Errorf("查询分类详情失败: %w", err)
	}

	return &category, nil
}

// Create 创建分类
func (s *CategoryService) Create(ctx context.Context, category *models.WorkOrderCategory, creatorID string) error {
	category.CreatedBy = creatorID
	category.UpdatedBy = creatorID

	if err := s.db.WithContext(ctx).Create(category).Error; err != nil {
		return fmt.Errorf("创建分类失败: %w", err)
	}

	return nil
}

// Update 更新分类
func (s *CategoryService) Update(ctx context.Context, category *models.WorkOrderCategory, operatorID string) error {
	category.UpdatedBy = operatorID

	if err := s.db.WithContext(ctx).Save(category).Error; err != nil {
		return fmt.Errorf("更新分类失败: %w", err)
	}

	return nil
}

// Delete 删除分类
func (s *CategoryService) Delete(ctx context.Context, id string) error {
	// 检查是否有子分类
	var count int64
	if err := s.db.WithContext(ctx).Model(&models.WorkOrderCategory{}).Where("parent_id = ?", id).Count(&count).Error; err != nil {
		return fmt.Errorf("检查子分类失败: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("该分类下有子分类，无法删除")
	}

	// 检查是否有关联工单
	if err := s.db.WithContext(ctx).Model(&models.WorkOrder{}).Where("category_id = ?", id).Count(&count).Error; err != nil {
		return fmt.Errorf("检查关联工单失败: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("该分类下有关联工单，无法删除")
	}

	// 删除分类
	if err := s.db.WithContext(ctx).Delete(&models.WorkOrderCategory{}, id).Error; err != nil {
		return fmt.Errorf("删除分类失败: %w", err)
	}

	return nil
}

// GetEnabled 获取启用的分类（用于下拉选择）
func (s *CategoryService) GetEnabled(ctx context.Context) ([]models.WorkOrderCategory, error) {
	var list []models.WorkOrderCategory

	if err := s.db.WithContext(ctx).
		Where("status = ?", models.WorkOrderCategoryStatusEnabled).
		Order("sort_order ASC, created_at ASC").
		Find(&list).Error; err != nil {
		return nil, fmt.Errorf("查询分类列表失败: %w", err)
	}

	return list, nil
}
