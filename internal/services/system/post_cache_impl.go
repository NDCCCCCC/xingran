package system

import (
	"context"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	"gorm.io/gorm"
)

// postCacheService 岗位缓存服务
type postCacheService struct {
	*postService
	cache CacheProvider
	CacheServiceBase
}

// NewPostServiceWithCache 创建带缓存的岗位服务
func NewPostServiceWithCache(
	db *gorm.DB,
	cache CacheProvider,
	config *services.CacheConfigService,
) PostService {
	base := &postService{db: db}
	return &postCacheService{
		postService:      base,
		cache:            cache,
		CacheServiceBase: CacheServiceBase{Config: config},
	}
}

// GetAllWithCache 获取所有岗位（覆盖基础方法，使用缓存）
func (s *postCacheService) GetAllWithCache(ctx context.Context) ([]*models.Post, error) {
	cacheKey := CacheKeyPostAll
	var result []*models.Post

	expiration := s.GetExpiration(services.CacheConfigPostAll, 30*time.Minute)

	err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
		return s.queryAll(ctx)
	})

	return result, err
}

// GetEnabledWithCache 获取启用的岗位（带缓存）
func (s *postCacheService) GetEnabledWithCache(ctx context.Context) ([]*models.Post, error) {
	cacheKey := CacheKeyPostEnabled
	var result []*models.Post

	expiration := s.GetExpiration(services.CacheConfigPostEnabled, 30*time.Minute)

	err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
		return s.queryEnabled(ctx)
	})

	return result, err
}

// queryAll 查询所有岗位
func (s *postCacheService) queryAll(ctx context.Context) ([]*models.Post, error) {
	var posts []models.Post
	if err := s.db.WithContext(ctx).
		Order("post_sort ASC").
		Find(&posts).Error; err != nil {
		return nil, err
	}

	result := make([]*models.Post, len(posts))
	for i := range posts {
		result[i] = &posts[i]
	}
	return result, nil
}

// queryEnabled 查询启用的岗位
func (s *postCacheService) queryEnabled(ctx context.Context) ([]*models.Post, error) {
	var posts []models.Post
	if err := s.db.WithContext(ctx).
		Where("status = ?", models.PostStatusEnabled).
		Order("post_sort ASC").
		Find(&posts).Error; err != nil {
		return nil, err
	}

	result := make([]*models.Post, len(posts))
	for i := range posts {
		result[i] = &posts[i]
	}
	return result, nil
}

// InvalidatePostCache 失效岗位缓存
func (s *postCacheService) InvalidatePostCache(ctx context.Context) error {
	InvalidateCacheByPattern(ctx, s.cache, []string{
		CacheKeyPostAll + "*",
		CacheKeyPostEnabled + "*",
	}, "POST")
	return nil
}

// Create 创建岗位（带缓存失效）
func (s *postCacheService) Create(ctx context.Context, req *requests.PostCreateRequest) error {
	if err := s.postService.Create(ctx, req); err != nil {
		return err
	}
	return s.InvalidatePostCache(ctx)
}

// Update 更新岗位（带缓存失效）
func (s *postCacheService) Update(ctx context.Context, req *requests.PostUpdateRequest) error {
	if err := s.postService.Update(ctx, req); err != nil {
		return err
	}
	return s.InvalidatePostCache(ctx)
}

// Delete 删除岗位（带缓存失效）
func (s *postCacheService) Delete(ctx context.Context, id string) error {
	if err := s.postService.Delete(ctx, id); err != nil {
		return err
	}
	return s.InvalidatePostCache(ctx)
}

// BatchDelete 批量删除岗位（带缓存失效）
func (s *postCacheService) BatchDelete(ctx context.Context, ids []string) error {
	if err := s.postService.BatchDelete(ctx, ids); err != nil {
		return err
	}
	return s.InvalidatePostCache(ctx)
}

// List 查询岗位列表（全量缓存+内存筛选版本）
func (s *postCacheService) List(ctx context.Context, params requests.PostListParams) (*PageResult, error) {
	allPosts, err := s.GetAllWithCache(ctx)
	if err != nil {
		return nil, err
	}

	filtered := s.filterPosts(allPosts, params)
	paged, total := paginate(filtered, params.Current, params.PageSize)

	return &PageResult{
		List:     paged,
		Total:    total,
		Current:  params.Current,
		PageSize: params.PageSize,
	}, nil
}

// filterPosts 在内存中筛选岗位
func (s *postCacheService) filterPosts(posts []*models.Post, params requests.PostListParams) []*models.Post {
	result := posts

	if params.PostCode != nil && *params.PostCode != "" {
		result = filterSlice(result, func(p *models.Post) bool {
			return contains(p.PostCode, *params.PostCode)
		})
	}

	if params.PostName != nil && *params.PostName != "" {
		result = filterSlice(result, func(p *models.Post) bool {
			return contains(p.PostName, *params.PostName)
		})
	}

	if params.Status != nil {
		result = filterSlice(result, func(p *models.Post) bool {
			return int(p.Status) == *params.Status
		})
	}

	return result
}
