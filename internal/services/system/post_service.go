package system

import (
	"context"
	"fmt"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"gorm.io/gorm"
)

// PostService 岗位服务接口
type PostService interface {
	Create(ctx context.Context, req *PostCreateRequest) error
	Update(ctx context.Context, req *PostUpdateRequest) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*models.Post, error)
	List(ctx context.Context, params PostListParams) (*PageResult, error)
	BatchDelete(ctx context.Context, ids []string) error
}

// postService 岗位服务实现
type postService struct {
	db *gorm.DB
}

// NewPostService 创建岗位服务实例
func NewPostService(db *gorm.DB) PostService {
	return &postService{db: db}
}

// ==================== 请求/响应类型 ====================

// PostListParams 岗位列表查询参数
type PostListParams struct {
	PostCode *string
	PostName *string
	Status   *int
	Current  int
	PageSize int
}

// DefaultPostListParams 默认列表参数
func DefaultPostListParams() PostListParams {
	return PostListParams{
		Current:  1,
		PageSize: 10,
	}
}

// PostCreateRequest 创建岗位请求
type PostCreateRequest struct {
	PostCode string
	PostName string
	PostSort int
	Status   models.PostStatus
	Remark   *string
}

// PostUpdateRequest 更新岗位请求
type PostUpdateRequest struct {
	ID       string
	PostName string
	PostSort int
	Status   models.PostStatus
	Remark   *string
}

// ==================== 服务方法实现 ====================

// Create 创建岗位
func (s *postService) Create(ctx context.Context, req *PostCreateRequest) error {
	// 检查岗位编码是否已存在
	var existPost models.Post
	if err := s.db.WithContext(ctx).Where("post_code = ?", req.PostCode).First(&existPost).Error; err == nil {
		return fmt.Errorf("岗位编码已存在")
	}

	post := models.Post{
		PostCode: req.PostCode,
		PostName: req.PostName,
		PostSort: req.PostSort,
		Status:   req.Status,
		Remark:   toStringPtr(req.Remark),
	}

	if err := s.db.WithContext(ctx).Create(&post).Error; err != nil {
		return fmt.Errorf("创建岗位失败: %w", err)
	}

	return nil
}

// Update 更新岗位
func (s *postService) Update(ctx context.Context, req *PostUpdateRequest) error {
	// 检查岗位是否存在
	var post models.Post
	if err := s.db.WithContext(ctx).First(&post, "id = ?", req.ID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("岗位不存在")
		}
		return fmt.Errorf("查询岗位失败: %w", err)
	}

	// 更新岗位信息
	post.PostName = req.PostName
	post.PostSort = req.PostSort
	post.Status = req.Status
	post.Remark = toStringPtr(req.Remark)

	if err := s.db.WithContext(ctx).Save(&post).Error; err != nil {
		return fmt.Errorf("更新岗位失败: %w", err)
	}

	return nil
}

// Delete 删除岗位
func (s *postService) Delete(ctx context.Context, id string) error {
	// 检查岗位是否存在
	var post models.Post
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&post).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("岗位不存在")
		}
		return fmt.Errorf("查询岗位失败: %w", err)
	}

	if err := s.db.WithContext(ctx).Delete(&post).Error; err != nil {
		return fmt.Errorf("删除岗位失败: %w", err)
	}

	return nil
}

// GetByID 根据ID获取岗位
func (s *postService) GetByID(ctx context.Context, id string) (*models.Post, error) {
	var post models.Post
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&post).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("岗位不存在")
		}
		return nil, fmt.Errorf("查询岗位失败: %w", err)
	}
	return &post, nil
}

// List 查询岗位列表
func (s *postService) List(ctx context.Context, params PostListParams) (*PageResult, error) {
	var total int64
	var list []models.Post

	query := s.db.WithContext(ctx).Model(&models.Post{})

	// 添加筛选条件
	if params.PostCode != nil && *params.PostCode != "" {
		query = query.Where("post_code LIKE ?", "%"+*params.PostCode+"%")
	}
	if params.PostName != nil && *params.PostName != "" {
		query = query.Where("post_name LIKE ?", "%"+*params.PostName+"%")
	}
	if params.Status != nil {
		query = query.Where("status = ?", *params.Status)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("统计岗位数量失败: %w", err)
	}

	// 分页查询
	offset := (params.Current - 1) * params.PageSize
	if err := query.Offset(offset).Limit(params.PageSize).Order("post_sort ASC").Find(&list).Error; err != nil {
		return nil, fmt.Errorf("查询岗位列表失败: %w", err)
	}

	return &PageResult{
		List:     list,
		Total:    total,
		Current:  params.Current,
		PageSize: params.PageSize,
	}, nil
}

// BatchDelete 批量删除岗位
func (s *postService) BatchDelete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return fmt.Errorf("ids不能为空")
	}

	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Delete(&models.Post{}).Error; err != nil {
		return fmt.Errorf("批量删除岗位失败: %w", err)
	}

	return nil
}
