package system

import (
	"context"
	"fmt"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"gorm.io/gorm"
)

// PostService 岗位服务接口
type PostService interface {
	Create(ctx context.Context, req *requests.PostCreateRequest) error
	Update(ctx context.Context, req *requests.PostUpdateRequest) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*models.Post, error)
	List(ctx context.Context, params requests.PostListParams) (*PageResult, error)
	BatchDelete(ctx context.Context, ids []string) error

	// 新增缓存方法
	GetAllWithCache(ctx context.Context) ([]*models.Post, error)
	GetEnabledWithCache(ctx context.Context) ([]*models.Post, error)
	InvalidatePostCache(ctx context.Context) error
	// Statistics 岗位统计(专用 COUNT 聚合,不依赖分页列表,不受 MaxPageSize=100 钳制)。
	Statistics(ctx context.Context) (*PostStatisticsResult, error)
}

// PostStatisticsResult 岗位统计结果(status: 0=正常 1=停用)。
type PostStatisticsResult struct {
	Total    int64 `json:"total"`
	Active   int64 `json:"active"`   // PostStatusEnabled
	Inactive int64 `json:"inactive"` // PostStatusDisabled
}

// postService 岗位服务实现
type postService struct {
	db *gorm.DB
}

// NewPostService 创建岗位服务实例
func NewPostService(db *gorm.DB) PostService {
	return &postService{db: db}
}

// postAllowedSortFields 岗位列表可排序字段白名单。
// 值为 DB 列名(纯列名,无 JOIN),与 Post 模型字段一一对应。
var postAllowedSortFields = map[string]string{
	"postCode":  "post_code",
	"postName":  "post_name",
	"postSort":  "post_sort",
	"status":    "status",
	"remark":    "remark",
	"createdAt": "created_at",
}

// Statistics 统计岗位(按 status 聚合,排除软删除)。
// 不依赖分页列表,避免「用 pageSize:1000 拉全量再 .length」被 MaxPageSize=100 钳制。
func (s *postService) Statistics(ctx context.Context) (*PostStatisticsResult, error) {
	var result PostStatisticsResult
	err := s.db.WithContext(ctx).
		Model(&models.Post{}).
		Select(
			"COUNT(*) AS total",
			fmt.Sprintf("COALESCE(SUM(CASE WHEN status = %d THEN 1 ELSE 0 END), 0) AS active", int(models.PostStatusEnabled)),
			fmt.Sprintf("COALESCE(SUM(CASE WHEN status = %d THEN 1 ELSE 0 END), 0) AS inactive", int(models.PostStatusDisabled)),
		).
		Scan(&result).Error
	if err != nil {
		return nil, fmt.Errorf("统计岗位失败: %w", err)
	}
	return &result, nil
}

// ==================== 服务方法实现 ====================

// Create 创建岗位
func (s *postService) Create(ctx context.Context, req *requests.PostCreateRequest) error {
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
func (s *postService) Update(ctx context.Context, req *requests.PostUpdateRequest) error {
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
func (s *postService) List(ctx context.Context, params requests.PostListParams) (*PageResult, error) {
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

	// 分页查询 - 用户排序(白名单)优先,无 OrderByColumn 时保留 post_sort ASC 默认排序
	offset := (params.Current - 1) * params.PageSize
	query = base.ApplySort(query, params.BaseListRequest, postAllowedSortFields)
	if params.OrderByColumn == "" {
		query = query.Order("post_sort ASC")
	}
	if err := query.Offset(offset).Limit(params.PageSize).Find(&list).Error; err != nil {
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

// GetAllWithCache 获取所有岗位（无缓存版本，直接查询数据库）
func (s *postService) GetAllWithCache(ctx context.Context) ([]*models.Post, error) {
	var posts []models.Post
	if err := s.db.WithContext(ctx).
		Order("post_sort ASC").
		Find(&posts).Error; err != nil {
		return nil, fmt.Errorf("查询岗位失败: %w", err)
	}

	result := make([]*models.Post, len(posts))
	for i := range posts {
		result[i] = &posts[i]
	}
	return result, nil
}

// GetEnabledWithCache 获取启用的岗位（无缓存版本，直接查询数据库）
func (s *postService) GetEnabledWithCache(ctx context.Context) ([]*models.Post, error) {
	var posts []models.Post
	if err := s.db.WithContext(ctx).
		Where("status = ?", models.PostStatusEnabled).
		Order("post_sort ASC").
		Find(&posts).Error; err != nil {
		return nil, fmt.Errorf("查询岗位失败: %w", err)
	}

	result := make([]*models.Post, len(posts))
	for i := range posts {
		result[i] = &posts[i]
	}
	return result, nil
}

// InvalidatePostCache 失效岗位缓存（无缓存版本，空操作）
func (s *postService) InvalidatePostCache(ctx context.Context) error {
	return nil
}
