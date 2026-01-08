package services

import (
	"context"
	"fmt"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// PostCacheService 岗位缓存服务
type PostCacheService struct {
	db        *gorm.DB
	dataCache *DataCacheService
}

// NewPostCacheService 创建岗位缓存服务
func NewPostCacheService(db *gorm.DB, dataCache *DataCacheService) *PostCacheService {
	return &PostCacheService{
		db:        db,
		dataCache: dataCache,
	}
}

// GetAllPostsWithCache 获取所有岗位（带缓存）
func (s *PostCacheService) GetAllPostsWithCache(ctx context.Context) ([]*models.Post, error) {
	cacheKey := "post:all"
	var posts []*models.Post

	// 使用动态配置的缓存时间
	expiration := s.dataCache.GetExpiration(CacheConfigPostAll, 30*time.Minute)

	// 尝试从缓存获取
	err := s.dataCache.GetOrSet(ctx, cacheKey, &posts, expiration, func() (interface{}, error) {
		return s.queryAllPosts()
	})

	if err != nil {
		return nil, fmt.Errorf("获取岗位列表失败: %w", err)
	}

	return posts, nil
}

// queryAllPosts 从数据库查询所有岗位
func (s *PostCacheService) queryAllPosts() ([]*models.Post, error) {
	var posts []models.Post

	if err := s.db.Where("status = ?", models.PostStatusEnabled).
		Order("post_sort ASC").
		Find(&posts).Error; err != nil {
		return nil, fmt.Errorf("查询岗位列表失败: %w", err)
	}

	// 转换为指针数组
	postPtrs := make([]*models.Post, len(posts))
	for i := range posts {
		postPtrs[i] = &posts[i]
	}

	return postPtrs, nil
}

// GetPostByIDWithCache 根据ID获取岗位（带缓存）
func (s *PostCacheService) GetPostByIDWithCache(ctx context.Context, postID string) (*models.Post, error) {
	if postID == "" {
		return nil, fmt.Errorf("岗位ID不能为空")
	}

	cacheKey := fmt.Sprintf("post:id:%s", postID)
	var post models.Post

	// 尝试从缓存获取
	err := s.dataCache.GetOrSet(ctx, cacheKey, &post, 30*time.Minute, func() (interface{}, error) {
		return s.queryPostByID(postID)
	})

	if err != nil {
		return nil, fmt.Errorf("获取岗位失败: %w", err)
	}

	return &post, nil
}

// queryPostByID 从数据库查询岗位
func (s *PostCacheService) queryPostByID(postID string) (*models.Post, error) {
	var post models.Post

	if err := s.db.Where("id = ?", postID).First(&post).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("岗位不存在")
		}
		return nil, fmt.Errorf("查询岗位失败: %w", err)
	}

	return &post, nil
}

// GetPostByCodeWithCache 根据岗位编码获取岗位（带缓存）
func (s *PostCacheService) GetPostByCodeWithCache(ctx context.Context, postCode string) (*models.Post, error) {
	if postCode == "" {
		return nil, fmt.Errorf("岗位编码不能为空")
	}

	cacheKey := fmt.Sprintf("post:code:%s", postCode)
	var post models.Post

	// 尝试从缓存获取
	err := s.dataCache.GetOrSet(ctx, cacheKey, &post, 30*time.Minute, func() (interface{}, error) {
		return s.queryPostByCode(postCode)
	})

	if err != nil {
		return nil, fmt.Errorf("获取岗位失败: %w", err)
	}

	return &post, nil
}

// queryPostByCode 从数据库查询岗位
func (s *PostCacheService) queryPostByCode(postCode string) (*models.Post, error) {
	var post models.Post

	if err := s.db.Where("post_code = ?", postCode).First(&post).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("岗位不存在")
		}
		return nil, fmt.Errorf("查询岗位失败: %w", err)
	}

	return &post, nil
}

// GetUserPostsWithCache 获取用户的岗位列表（带缓存）
func (s *PostCacheService) GetUserPostsWithCache(ctx context.Context, userID string) ([]*models.Post, error) {
	if userID == "" {
		return nil, fmt.Errorf("用户ID不能为空")
	}

	cacheKey := fmt.Sprintf("post:user:%s", userID)
	var posts []*models.Post

	// 尝试从缓存获取
	err := s.dataCache.GetOrSet(ctx, cacheKey, &posts, 30*time.Minute, func() (interface{}, error) {
		return s.queryUserPosts(userID)
	})

	if err != nil {
		return nil, fmt.Errorf("获取用户岗位失败: %w", err)
	}

	return posts, nil
}

// queryUserPosts 从数据库查询用户岗位
func (s *PostCacheService) queryUserPosts(userID string) ([]*models.Post, error) {
	var postIDs []string

	// 查询用户关联的岗位ID
	if err := s.db.Table("sys_user_post").
		Where("user_id = ?", userID).
		Pluck("post_id", &postIDs).Error; err != nil {
		return nil, fmt.Errorf("查询用户岗位关联失败: %w", err)
	}

	if len(postIDs) == 0 {
		return []*models.Post{}, nil
	}

	// 查询岗位详情
	var posts []models.Post
	if err := s.db.Where("id IN ? AND status = ?", postIDs, models.PostStatusEnabled).
		Order("post_sort ASC").
		Find(&posts).Error; err != nil {
		return nil, fmt.Errorf("查询岗位详情失败: %w", err)
	}

	// 转换为指针数组
	postPtrs := make([]*models.Post, len(posts))
	for i := range posts {
		postPtrs[i] = &posts[i]
	}

	return postPtrs, nil
}

// InvalidatePostCache 清除岗位相关缓存
func (s *PostCacheService) InvalidatePostCache(ctx context.Context) error {
	patterns := []string{
		"post:all",
		"post:id:*",
		"post:code:*",
	}

	for _, pattern := range patterns {
		if err := s.dataCache.DeleteByPattern(ctx, pattern); err != nil {
			return fmt.Errorf("清除缓存失败 %s: %w", pattern, err)
		}
	}

	return nil
}

// InvalidatePostCacheByID 清除指定岗位的缓存
func (s *PostCacheService) InvalidatePostCacheByID(ctx context.Context, postID string) error {
	patterns := []string{
		"post:all",
		fmt.Sprintf("post:id:%s", postID),
	}

	// 获取岗位信息以便获取岗位编码
	post, err := s.queryPostByID(postID)
	if err == nil {
		patterns = append(patterns, fmt.Sprintf("post:code:%s", post.PostCode))
	}

	for _, pattern := range patterns {
		if err := s.dataCache.Delete(ctx, pattern); err != nil {
			return fmt.Errorf("清除缓存失败 %s: %w", pattern, err)
		}
	}

	return nil
}

// InvalidateUserPostCache 清除用户岗位缓存
func (s *PostCacheService) InvalidateUserPostCache(ctx context.Context, userID string) error {
	cacheKey := fmt.Sprintf("post:user:%s", userID)
	if err := s.dataCache.Delete(ctx, cacheKey); err != nil {
		return fmt.Errorf("清除用户岗位缓存失败: %w", err)
	}
	return nil
}

// CreatePost 创建岗位
func (s *PostCacheService) CreatePost(ctx context.Context, post *models.Post) error {
	if err := s.db.Create(post).Error; err != nil {
		return fmt.Errorf("创建岗位失败: %w", err)
	}

	// 清除缓存
	if err := s.InvalidatePostCache(ctx); err != nil {
		logger.Warnf("[POST] 清除缓存失败: %v", err)
	}

	return nil
}

// UpdatePost 更新岗位
func (s *PostCacheService) UpdatePost(ctx context.Context, post *models.Post) error {
	if err := s.db.Save(post).Error; err != nil {
		return fmt.Errorf("更新岗位失败: %w", err)
	}

	// 清除缓存
	if err := s.InvalidatePostCacheByID(ctx, post.ID); err != nil {
		logger.Warnf("[POST] 清除缓存失败: %v", err)
	}

	return nil
}

// DeletePost 删除岗位
func (s *PostCacheService) DeletePost(ctx context.Context, postID string) error {
	if err := s.db.Delete(&models.Post{}, "id = ?", postID).Error; err != nil {
		return fmt.Errorf("删除岗位失败: %w", err)
	}

	// 清除缓存
	if err := s.InvalidatePostCacheByID(ctx, postID); err != nil {
		logger.Warnf("[POST] 清除缓存失败: %v", err)
	}

	return nil
}

// AssignUserPosts 分配用户岗位
func (s *PostCacheService) AssignUserPosts(ctx context.Context, userID string, postIDs []string) error {
	// 使用事务
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 删除旧的岗位关联
	if err := tx.Table("sys_user_post").Where("user_id = ?", userID).Delete(&models.UserPost{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除旧岗位关联失败: %w", err)
	}

	// 添加新的岗位关联
	if len(postIDs) > 0 {
		var userPosts []models.UserPost
		for _, postID := range postIDs {
			userPosts = append(userPosts, models.UserPost{
				UserID: userID,
				PostID: postID,
			})
		}
		if err := tx.Create(&userPosts).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("添加岗位关联失败: %w", err)
		}
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	// 清除用户岗位缓存
	if err := s.InvalidateUserPostCache(ctx, userID); err != nil {
		logger.Warnf("[POST] 清除缓存失败: %v", err)
	}

	return nil
}

// GetEnabledPostsWithCache 获取启用的岗位列表（带缓存）
func (s *PostCacheService) GetEnabledPostsWithCache(ctx context.Context) ([]*models.Post, error) {
	cacheKey := "post:enabled"
	var posts []*models.Post

	// 使用动态配置的缓存时间
	expiration := s.dataCache.GetExpiration(CacheConfigPostEnabled, 30*time.Minute)

	// 尝试从缓存获取
	err := s.dataCache.GetOrSet(ctx, cacheKey, &posts, expiration, func() (interface{}, error) {
		return s.queryEnabledPosts()
	})

	if err != nil {
		return nil, fmt.Errorf("获取启用岗位列表失败: %w", err)
	}

	return posts, nil
}

// queryEnabledPosts 从数据库查询启用的岗位
func (s *PostCacheService) queryEnabledPosts() ([]*models.Post, error) {
	var posts []models.Post

	if err := s.db.Where("status = ?", models.PostStatusEnabled).
		Order("post_sort ASC").
		Find(&posts).Error; err != nil {
		return nil, fmt.Errorf("查询启用岗位失败: %w", err)
	}

	// 转换为指针数组
	postPtrs := make([]*models.Post, len(posts))
	for i := range posts {
		postPtrs[i] = &posts[i]
	}

	return postPtrs, nil
}
