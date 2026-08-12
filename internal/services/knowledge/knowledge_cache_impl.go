package knowledge

import (
	"context"
	"fmt"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
	"gorm.io/gorm"
)

// KnowledgeCacheService 知识库缓存服务接口
type KnowledgeCacheService interface {
	// 基础服务方法
	GetKnowledgeArticleList(ctx context.Context, req *services.KnowledgeArticleListRequest) ([]models.KnowledgeArticle, int64, error)
	GetArticleStatistics(ctx context.Context) (*services.KnowledgeArticleStatistics, error)
	GetKnowledgeArticle(ctx context.Context, id string) (*models.KnowledgeArticle, error)
	CreateKnowledgeArticle(ctx context.Context, req *services.KnowledgeArticleCreateRequest, creatorID string) (*models.KnowledgeArticle, error)
	UpdateKnowledgeArticle(ctx context.Context, id string, req *services.KnowledgeArticleUpdateRequest, operatorID string) error
	DeleteKnowledgeArticle(ctx context.Context, id string) error
	IncrementViewCount(ctx context.Context, id string) error
	IncrementLikeCount(ctx context.Context, id string) error
	SearchKnowledgeArticles(ctx context.Context, req *services.SearchKnowledgeRequest) ([]models.KnowledgeArticle, int64, error)
	ConvertWorkOrderToArticle(ctx context.Context, workOrderID string, req *services.ConvertWorkOrderToArticleRequest, creatorID string) (*models.KnowledgeArticle, error)

	GetKnowledgeCategoryList(ctx context.Context, req *services.KnowledgeCategoryListRequest) ([]models.KnowledgeCategory, error)
	GetKnowledgeCategory(ctx context.Context, id string) (*models.KnowledgeCategory, error)
	CreateKnowledgeCategory(ctx context.Context, req *services.KnowledgeCategoryCreateRequest, creatorID string) (*models.KnowledgeCategory, error)
	UpdateKnowledgeCategory(ctx context.Context, id string, req *services.KnowledgeCategoryUpdateRequest, operatorID string) error
	DeleteKnowledgeCategory(ctx context.Context, id string) error

	GetAllTags(ctx context.Context) ([]models.KnowledgeTag, error)
	GetTagByName(ctx context.Context, name string) (*models.KnowledgeTag, error)
	CreateTag(ctx context.Context, name string) (*models.KnowledgeTag, error)
	UpdateTag(ctx context.Context, id string, name string) error
	DeleteTag(ctx context.Context, id string) error

	// 缓存失效方法
	InvalidateCategoryCache(ctx context.Context) error
	InvalidateTagCache(ctx context.Context) error
	InvalidateArticleCache(ctx context.Context, articleID string) error
	InvalidateAllArticleCache(ctx context.Context) error
}

// knowledgeCacheServiceImpl 知识库缓存服务实现
type knowledgeCacheServiceImpl struct {
	base   *services.KnowledgeService
	cache  systemServices.CacheProvider
	config *services.CacheConfigService
}

// NewKnowledgeServiceWithCache 创建带缓存的知识库服务
func NewKnowledgeServiceWithCache(
	db *gorm.DB,
	cache systemServices.CacheProvider,
	config *services.CacheConfigService,
) KnowledgeCacheService {
	base := services.NewKnowledgeService(db)
	return &knowledgeCacheServiceImpl{
		base:   base,
		cache:  cache,
		config: config,
	}
}

// getExpiration 获取缓存过期时间
func (s *knowledgeCacheServiceImpl) getExpiration(configKey string, defaultVal time.Duration) time.Duration {
	if s.config != nil {
		return s.config.GetDurationWithDefault(configKey, defaultVal)
	}
	return defaultVal
}

// ==================== 文章相关方法（部分缓存） ====================

// GetKnowledgeArticleList 获取知识库文章列表（不缓存，参数多变）
func (s *knowledgeCacheServiceImpl) GetKnowledgeArticleList(ctx context.Context, req *services.KnowledgeArticleListRequest) ([]models.KnowledgeArticle, int64, error) {
	return s.base.GetKnowledgeArticleList(ctx, req)
}

// GetArticleStatistics 统计知识库文章(总数/状态/浏览/点赞),委托给基础服务。不缓存。
func (s *knowledgeCacheServiceImpl) GetArticleStatistics(ctx context.Context) (*services.KnowledgeArticleStatistics, error) {
	return s.base.GetArticleStatistics(ctx)
}

// GetKnowledgeArticle 获取知识库文章详情（带缓存）
func (s *knowledgeCacheServiceImpl) GetKnowledgeArticle(ctx context.Context, id string) (*models.KnowledgeArticle, error) {
	cacheKey := fmt.Sprintf("kb:article:%s", id)
	var result models.KnowledgeArticle

	expiration := s.getExpiration("cache.kb.article", 10*time.Minute)

	err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
		return s.base.GetKnowledgeArticle(ctx, id)
	})

	if err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateKnowledgeArticle 创建知识库文章（带缓存失效）
func (s *knowledgeCacheServiceImpl) CreateKnowledgeArticle(ctx context.Context, req *services.KnowledgeArticleCreateRequest, creatorID string) (*models.KnowledgeArticle, error) {
	article, err := s.base.CreateKnowledgeArticle(ctx, req, creatorID)
	if err != nil {
		return nil, err
	}
	// 创建文章不影响缓存（因为是新文章）
	return article, nil
}

// UpdateKnowledgeArticle 更新知识库文章（带缓存失效）
func (s *knowledgeCacheServiceImpl) UpdateKnowledgeArticle(ctx context.Context, id string, req *services.KnowledgeArticleUpdateRequest, operatorID string) error {
	if err := s.base.UpdateKnowledgeArticle(ctx, id, req, operatorID); err != nil {
		return err
	}
	// 清除该文章的缓存
	return s.InvalidateArticleCache(ctx, id)
}

// DeleteKnowledgeArticle 删除知识库文章（带缓存失效）
func (s *knowledgeCacheServiceImpl) DeleteKnowledgeArticle(ctx context.Context, id string) error {
	if err := s.base.DeleteKnowledgeArticle(ctx, id); err != nil {
		return err
	}
	// 清除该文章的缓存
	return s.InvalidateArticleCache(ctx, id)
}

// ==================== 分类相关方法（全部缓存） ====================

// GetKnowledgeCategoryList 获取知识库分类列表（树形结构，带缓存）
func (s *knowledgeCacheServiceImpl) GetKnowledgeCategoryList(ctx context.Context, req *services.KnowledgeCategoryListRequest) ([]models.KnowledgeCategory, error) {
	// 构建缓存键，包含查询条件
	cacheKey := "kb:category:tree"
	if req.ParentID != nil && *req.ParentID != "" {
		cacheKey = fmt.Sprintf("kb:category:parent:%s", *req.ParentID)
	}
	if req.Status != nil {
		cacheKey = fmt.Sprintf("%s:status:%d", cacheKey, *req.Status)
	}

	var result []models.KnowledgeCategory

	expiration := s.getExpiration("cache.kb.category", 30*time.Minute)

	err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
		return s.base.GetKnowledgeCategoryList(ctx, req)
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetKnowledgeCategory 获取知识库分类详情（不缓存，查询频率低）
func (s *knowledgeCacheServiceImpl) GetKnowledgeCategory(ctx context.Context, id string) (*models.KnowledgeCategory, error) {
	return s.base.GetKnowledgeCategory(ctx, id)
}

// CreateKnowledgeCategory 创建知识库分类（带缓存失效）
func (s *knowledgeCacheServiceImpl) CreateKnowledgeCategory(ctx context.Context, req *services.KnowledgeCategoryCreateRequest, creatorID string) (*models.KnowledgeCategory, error) {
	category, err := s.base.CreateKnowledgeCategory(ctx, req, creatorID)
	if err != nil {
		return nil, err
	}
	// 清除分类缓存
	_ = s.InvalidateCategoryCache(ctx)
	return category, nil
}

// UpdateKnowledgeCategory 更新知识库分类（带缓存失效）
func (s *knowledgeCacheServiceImpl) UpdateKnowledgeCategory(ctx context.Context, id string, req *services.KnowledgeCategoryUpdateRequest, operatorID string) error {
	if err := s.base.UpdateKnowledgeCategory(ctx, id, req, operatorID); err != nil {
		return err
	}
	// 清除分类缓存
	return s.InvalidateCategoryCache(ctx)
}

// DeleteKnowledgeCategory 删除知识库分类（带缓存失效）
func (s *knowledgeCacheServiceImpl) DeleteKnowledgeCategory(ctx context.Context, id string) error {
	if err := s.base.DeleteKnowledgeCategory(ctx, id); err != nil {
		return err
	}
	// 清除分类缓存
	return s.InvalidateCategoryCache(ctx)
}

// ==================== 标签相关方法（全部缓存） ====================

// GetAllTags 获取所有标签（带缓存）
func (s *knowledgeCacheServiceImpl) GetAllTags(ctx context.Context) ([]models.KnowledgeTag, error) {
	cacheKey := "kb:tags:all"
	var result []models.KnowledgeTag

	expiration := s.getExpiration("cache.kb.tags", 30*time.Minute)

	err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
		return s.base.GetAllTags(ctx)
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetTagByName 根据名称获取标签（不缓存，内部使用）
func (s *knowledgeCacheServiceImpl) GetTagByName(ctx context.Context, name string) (*models.KnowledgeTag, error) {
	return s.base.GetTagByName(ctx, name)
}

// CreateTag 创建标签（带缓存失效）
func (s *knowledgeCacheServiceImpl) CreateTag(ctx context.Context, name string) (*models.KnowledgeTag, error) {
	tag, err := s.base.CreateTag(ctx, name)
	if err != nil {
		return nil, err
	}
	// 清除标签缓存
	_ = s.InvalidateTagCache(ctx)
	return tag, nil
}

// UpdateTag 更新标签（带缓存失效）
func (s *knowledgeCacheServiceImpl) UpdateTag(ctx context.Context, id string, name string) error {
	if err := s.base.UpdateTag(ctx, id, name); err != nil {
		return err
	}
	// 清除标签缓存
	return s.InvalidateTagCache(ctx)
}

// DeleteTag 删除标签（带缓存失效）
func (s *knowledgeCacheServiceImpl) DeleteTag(ctx context.Context, id string) error {
	if err := s.base.DeleteTag(ctx, id); err != nil {
		return err
	}
	// 清除标签缓存
	return s.InvalidateTagCache(ctx)
}

// ==================== 缓存失效方法 ====================

// InvalidateCategoryCache 失效分类缓存
func (s *knowledgeCacheServiceImpl) InvalidateCategoryCache(ctx context.Context) error {
	keys := []string{"kb:category:*"}
	systemServices.InvalidateCacheByPattern(ctx, s.cache, keys, "KNOWLEDGE")
	return nil
}

// InvalidateTagCache 失效标签缓存
func (s *knowledgeCacheServiceImpl) InvalidateTagCache(ctx context.Context) error {
	keys := []string{"kb:tags:all"}
	systemServices.InvalidateCacheByKey(ctx, s.cache, keys, "KNOWLEDGE")
	return nil
}

// InvalidateArticleCache 失效文章缓存
func (s *knowledgeCacheServiceImpl) InvalidateArticleCache(ctx context.Context, articleID string) error {
	keys := []string{fmt.Sprintf("kb:article:%s", articleID)}
	systemServices.InvalidateCacheByKey(ctx, s.cache, keys, "KNOWLEDGE")
	return nil
}

// InvalidateAllArticleCache 失效所有文章缓存
func (s *knowledgeCacheServiceImpl) InvalidateAllArticleCache(ctx context.Context) error {
	keys := []string{"kb:article:*"}
	systemServices.InvalidateCacheByPattern(ctx, s.cache, keys, "KNOWLEDGE")
	return nil
}

// ==================== 文章其他方法（无缓存） ====================

// IncrementViewCount 增加浏览次数
func (s *knowledgeCacheServiceImpl) IncrementViewCount(ctx context.Context, id string) error {
	return s.base.IncrementViewCount(ctx, id)
}

// IncrementLikeCount 增加点赞次数
func (s *knowledgeCacheServiceImpl) IncrementLikeCount(ctx context.Context, id string) error {
	return s.base.IncrementLikeCount(ctx, id)
}

// SearchKnowledgeArticles 搜索知识库文章
func (s *knowledgeCacheServiceImpl) SearchKnowledgeArticles(ctx context.Context, req *services.SearchKnowledgeRequest) ([]models.KnowledgeArticle, int64, error) {
	return s.base.SearchKnowledgeArticles(ctx, req)
}

// ConvertWorkOrderToArticle 将工单转换为知识库文章
func (s *knowledgeCacheServiceImpl) ConvertWorkOrderToArticle(ctx context.Context, workOrderID string, req *services.ConvertWorkOrderToArticleRequest, creatorID string) (*models.KnowledgeArticle, error) {
	return s.base.ConvertWorkOrderToArticle(ctx, workOrderID, req, creatorID)
}
