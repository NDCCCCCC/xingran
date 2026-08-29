package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// ==================== 知识库服务 ====================

// KnowledgeService 知识库服务
type KnowledgeService struct {
	db *gorm.DB
}

// NewKnowledgeService 创建知识库服务
func NewKnowledgeService(db *gorm.DB) *KnowledgeService {
	return &KnowledgeService{db: db}
}

// KnowledgeArticleStatistics 知识库文章统计结果。
// status: KnowledgeArticleStatusDraft=草稿 / KnowledgeArticleStatusPublished=已发布（E 簇反转：1=已发布）。
type KnowledgeArticleStatistics struct {
	Total      int64 `json:"total"`
	Draft      int64 `json:"draft"`      // KnowledgeArticleStatusDraft
	Published  int64 `json:"published"`  // KnowledgeArticleStatusPublished
	TotalViews int64 `json:"totalViews"` // SUM(view_count)
	TotalLikes int64 `json:"totalLikes"` // SUM(like_count)
}

// GetArticleStatistics 统计知识库文章总数/状态计数及累计浏览/点赞数。
// 用条件聚合(SUM CASE)避免「按当前页 list 计算统计」的错误——旧前端用当前页 list 算
// total/draft/published/totalViews/totalLikes,多页时严重偏小。SUM 用 COALESCE 防空集返回 NULL。
func (s *KnowledgeService) GetArticleStatistics(ctx context.Context) (*KnowledgeArticleStatistics, error) {
	var result KnowledgeArticleStatistics
	err := s.db.WithContext(ctx).
		Model(&models.KnowledgeArticle{}).
		Select(
			"COUNT(*) AS total",
			fmt.Sprintf("SUM(CASE WHEN status = %d THEN 1 ELSE 0 END) AS draft", int(models.KnowledgeArticleStatusDraft)),
			fmt.Sprintf("SUM(CASE WHEN status = %d THEN 1 ELSE 0 END) AS published", int(models.KnowledgeArticleStatusPublished)),
			"COALESCE(SUM(view_count), 0) AS total_views",
			"COALESCE(SUM(like_count), 0) AS total_likes",
		).
		Scan(&result).Error
	if err != nil {
		return nil, fmt.Errorf("统计知识库文章失败: %w", err)
	}
	return &result, nil
}

// ==================== 请求/响应结构 ====================

// KnowledgeArticleListRequest 知识库文章列表请求
type KnowledgeArticleListRequest struct {
	base.BaseListRequest
	Title      string `json:"title"`
	CategoryID string `json:"categoryId"`
	TagID      string `json:"tagId"`
	Status     *int   `json:"status"`
	CreatedBy  string `json:"createdBy"`
}

// knowledgeArticleAllowedSortFields 知识库文章可排序字段白名单。
var knowledgeArticleAllowedSortFields = map[string]string{
	"title":         "title",
	"categoryId":    "category_id",
	"viewCount":     "view_count",
	"likeCount":     "like_count",
	"status":        "status",
	"createdAt":     "created_at",
	"updatedAt":     "updated_at",
}

// KnowledgeArticleCreateRequest 创建知识库文章请求
type KnowledgeArticleCreateRequest struct {
	Title             string   `json:"title" binding:"required,max=200"`
	Content           string   `json:"content" binding:"required"`
	Summary           string   `json:"summary"`
	CategoryID        string   `json:"categoryId" binding:"required,uuid"`
	Status            int      `json:"status"`
	TagIDs            []string `json:"tagIds"`
	SourceWorkOrderID *string  `json:"sourceWorkOrderId"` // 来源工单ID（用于工单转知识库）
}

// KnowledgeArticleUpdateRequest 更新知识库文章请求
type KnowledgeArticleUpdateRequest struct {
	Title      *string  `json:"title"`
	Content    *string  `json:"content"`
	Summary    *string  `json:"summary"`
	CategoryID *string  `json:"categoryId"`
	Status     *int     `json:"status"`
	TagIDs     []string `json:"tagIds"`
}

// KnowledgeCategoryListRequest 知识库分类列表请求
type KnowledgeCategoryListRequest struct {
	ParentID *string `json:"parentId"`
	Status   *int    `json:"status"`
}

// KnowledgeCategoryCreateRequest 创建知识库分类请求
type KnowledgeCategoryCreateRequest struct {
	CategoryName string  `json:"categoryName" binding:"required,max=100"`
	Description  string  `json:"description"`
	ParentID     *string `json:"parentId"`
	SortOrder    int     `json:"sortOrder"`
	Status       int     `json:"status"`
}

// KnowledgeCategoryUpdateRequest 更新知识库分类请求
type KnowledgeCategoryUpdateRequest struct {
	CategoryName *string `json:"categoryName"`
	Description  *string `json:"description"`
	ParentID     *string `json:"parentId"`
	SortOrder    *int    `json:"sortOrder"`
	Status       *int    `json:"status"`
}

// ConvertWorkOrderToArticleRequest 工单转知识库请求
type ConvertWorkOrderToArticleRequest struct {
	Title      string   `json:"title" binding:"required,max=200"`
	Content    string   `json:"content" binding:"required"`
	Summary    string   `json:"summary"`
	CategoryID string   `json:"categoryId" binding:"required,uuid"`
	TagIDs     []string `json:"tagIds"`
	Status     int      `json:"status"`
}

// SearchKnowledgeRequest 搜索知识库请求
type SearchKnowledgeRequest struct {
	Keyword    string `json:"keyword"`
	CategoryID string `json:"categoryId"`
	TagID      string `json:"tagId"`
	PageSize   int    `json:"pageSize"` // 每页大小，默认100
	PageNum    int    `json:"pageNum"`  // 页码，从0开始
}

// ==================== 知识库文章方法 ====================

// GetKnowledgeArticleList 获取知识库文章列表
func (s *KnowledgeService) GetKnowledgeArticleList(ctx context.Context, req *KnowledgeArticleListRequest) ([]models.KnowledgeArticle, int64, error) {
	var list []models.KnowledgeArticle
	var total int64

	query := s.db.WithContext(ctx).Model(&models.KnowledgeArticle{})

	if req.Title != "" {
		query = query.Where("title LIKE ?", "%"+req.Title+"%")
	}
	if req.CategoryID != "" {
		query = query.Where("category_id = ?", req.CategoryID)
	}
	if req.Status != nil {
		query = query.Where("status = ?", *req.Status)
	}
	if req.CreatedBy != "" {
		query = query.Where("created_by = ?", req.CreatedBy)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询知识库文章总数失败: %w", err)
	}

	current := req.Current
	if current <= 0 {
		current = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}
	offset := (current - 1) * pageSize

	// 用户排序(白名单)优先,无 OrderByColumn 时保留 created_at DESC 默认
	query = base.ApplySort(query, req.BaseListRequest, knowledgeArticleAllowedSortFields)
	if req.OrderByColumn == "" {
		query = query.Order("created_at DESC")
	}

	if err := query.
		Preload("Category").
		Preload("Tags").
		Preload("SourceWorkOrder").
		Limit(pageSize).
		Offset(offset).
		Find(&list).Error; err != nil {
		return nil, 0, fmt.Errorf("查询知识库文章列表失败: %w", err)
	}

	return list, total, nil
}

// GetKnowledgeArticle 获取知识库文章详情
func (s *KnowledgeService) GetKnowledgeArticle(ctx context.Context, id string) (*models.KnowledgeArticle, error) {
	var article models.KnowledgeArticle

	if err := s.db.WithContext(ctx).
		Preload("Category").
		Preload("Tags").
		Preload("SourceWorkOrder").
		Where("id = ?", id).
		First(&article).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("知识库文章不存在")
		}
		return nil, fmt.Errorf("查询知识库文章详情失败: %w", err)
	}

	return &article, nil
}

// CreateKnowledgeArticle 创建知识库文章
func (s *KnowledgeService) CreateKnowledgeArticle(ctx context.Context, req *KnowledgeArticleCreateRequest, creatorID string) (*models.KnowledgeArticle, error) {
	var article *models.KnowledgeArticle

	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		newArticle := &models.KnowledgeArticle{
			Title:             req.Title,
			Content:           req.Content,
			Summary:           req.Summary,
			CategoryID:        req.CategoryID,
			Status:            models.KnowledgeArticleStatus(req.Status),
			ViewCount:         0,
			LikeCount:         0,
			SourceWorkOrderID: req.SourceWorkOrderID,
			BaseModel:         models.BaseModel{CreatedBy: creatorID, UpdatedBy: creatorID},
		}

		if err := tx.Create(newArticle).Error; err != nil {
			return fmt.Errorf("创建知识库文章失败: %w", err)
		}

		// 关联标签（支持标签名称或UUID）
		if len(req.TagIDs) > 0 {
			for _, tagInput := range req.TagIDs {
				// 判断是UUID还是标签名称
				var tagID string

				if _, err := uuid.Parse(tagInput); err == nil {
					// 是有效的UUID，直接使用
					tagID = tagInput
				} else {
					// 不是UUID，当作标签名称处理，获取或创建标签
					tag, err := s.GetOrCreateTag(ctx, tagInput)
					if err != nil {
						return fmt.Errorf("获取或创建标签失败: %w", err)
					}
					tagID = tag.ID
				}

				// 检查是否已经关联
				var count int64
				if err := tx.Model(&models.KnowledgeArticleTag{}).
					Where("article_id = ? AND tag_id = ?", newArticle.ID, tagID).
					Count(&count).Error; err != nil {
					return fmt.Errorf("检查标签关联失败: %w", err)
				}

				// 如果未关联，则创建关联
				if count == 0 {
					articleTag := &models.KnowledgeArticleTag{
						ArticleID: newArticle.ID,
						TagID:     tagID,
						CreatedAt: newArticle.CreatedAt,
					}
					if err := tx.Create(articleTag).Error; err != nil {
						return fmt.Errorf("关联标签失败: %w", err)
					}

					// 更新标签使用次数
					tx.Model(&models.KnowledgeTag{}).Where("id = ?", tagID).UpdateColumn("use_count", gorm.Expr("use_count + 1"))
				}
			}
		}

		article = newArticle
		return nil
	}); err != nil {
		return nil, err
	}

	return s.GetKnowledgeArticle(ctx, article.ID)
}

// UpdateKnowledgeArticle 更新知识库文章
func (s *KnowledgeService) UpdateKnowledgeArticle(ctx context.Context, id string, req *KnowledgeArticleUpdateRequest, operatorID string) error {
	var article models.KnowledgeArticle
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&article).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("知识库文章不存在")
		}
		return fmt.Errorf("查询知识库文章失败: %w", err)
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		updates := map[string]interface{}{
			"updated_by": operatorID,
		}

		if req.Title != nil {
			updates["title"] = *req.Title
		}
		if req.Content != nil {
			updates["content"] = *req.Content
		}
		if req.Summary != nil {
			updates["summary"] = *req.Summary
		}
		if req.CategoryID != nil {
			updates["category_id"] = *req.CategoryID
		}
		if req.Status != nil {
			updates["status"] = *req.Status
		}

		if err := tx.Model(&article).Updates(updates).Error; err != nil {
			return fmt.Errorf("更新知识库文章失败: %w", err)
		}

		// 更新标签关联
		if req.TagIDs != nil {
			// 删除旧的关联
			var oldTags []models.KnowledgeArticleTag
			tx.Where("article_id = ?", id).Find(&oldTags)
			for _, oldTag := range oldTags {
				tx.Delete(&oldTag)
				// 减少标签使用次数
				tx.Model(&models.KnowledgeTag{}).Where("id = ?", oldTag.TagID).UpdateColumn("use_count", gorm.Expr("use_count - 1"))
			}

			// 添加新的关联（支持标签名称或UUID）
			for _, tagInput := range req.TagIDs {
				// 判断是UUID还是标签名称
				var tagID string

				if _, err := uuid.Parse(tagInput); err == nil {
					// 是有效的UUID，直接使用
					tagID = tagInput
				} else {
					// 不是UUID，当作标签名称处理，获取或创建标签
					tag, err := s.GetOrCreateTag(ctx, tagInput)
					if err != nil {
						return fmt.Errorf("获取或创建标签失败: %w", err)
					}
					tagID = tag.ID
				}

				// 检查是否已经关联
				var count int64
				if err := tx.Model(&models.KnowledgeArticleTag{}).
					Where("article_id = ? AND tag_id = ?", id, tagID).
					Count(&count).Error; err != nil {
					return fmt.Errorf("检查标签关联失败: %w", err)
				}

				// 如果未关联，则创建关联
				if count == 0 {
					articleTag := &models.KnowledgeArticleTag{
						ArticleID: id,
						TagID:     tagID,
						CreatedAt: article.UpdatedAt,
					}
					if err := tx.Create(articleTag).Error; err != nil {
						return fmt.Errorf("关联标签失败: %w", err)
					}

					// 更新标签使用次数
					tx.Model(&models.KnowledgeTag{}).Where("id = ?", tagID).UpdateColumn("use_count", gorm.Expr("use_count + 1"))
				}
			}
		}

		return nil
	})
}

// DeleteKnowledgeArticle 删除知识库文章
func (s *KnowledgeService) DeleteKnowledgeArticle(ctx context.Context, id string) error {
	var article models.KnowledgeArticle
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&article).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("知识库文章不存在")
		}
		return fmt.Errorf("查询知识库文章失败: %w", err)
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除标签关联
		var articleTags []models.KnowledgeArticleTag
		if err := tx.Where("article_id = ?", id).Find(&articleTags).Error; err != nil {
			return fmt.Errorf("查询标签关联失败: %w", err)
		}

		for _, tag := range articleTags {
			tx.Delete(&tag)
			// 减少标签使用次数
			tx.Model(&models.KnowledgeTag{}).Where("id = ?", tag.TagID).UpdateColumn("use_count", gorm.Expr("use_count - 1"))
		}

		// 删除文章
		if err := tx.Delete(&article).Error; err != nil {
			return fmt.Errorf("删除知识库文章失败: %w", err)
		}

		return nil
	})
}

// ConvertWorkOrderToArticle 将工单转换为知识库文章
func (s *KnowledgeService) ConvertWorkOrderToArticle(ctx context.Context, workOrderID string, req *ConvertWorkOrderToArticleRequest, creatorID string) (*models.KnowledgeArticle, error) {
	// 验证工单是否存在
	var workOrder models.WorkOrder
	if err := s.db.WithContext(ctx).Where("id = ?", workOrderID).First(&workOrder).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("工单不存在")
		}
		return nil, fmt.Errorf("查询工单失败: %w", err)
	}

	// 检查工单状态是否允许转换
	if workOrder.Status != models.WorkOrderStatusCompleted && workOrder.Status != models.WorkOrderStatusClosed {
		return nil, fmt.Errorf("只有已完成或已关闭的工单才能转换为知识库文章")
	}

	// 检查是否已经转换过
	var count int64
	if err := s.db.WithContext(ctx).Model(&models.KnowledgeArticle{}).
		Where("source_work_order_id = ?", workOrderID).
		Count(&count).Error; err != nil {
		return nil, fmt.Errorf("检查知识库文章失败: %w", err)
	}
	if count > 0 {
		return nil, fmt.Errorf("该工单已经转换为知识库文章")
	}

	// 创建知识库文章
	createReq := &KnowledgeArticleCreateRequest{
		Title:             req.Title,
		Content:           req.Content,
		Summary:           req.Summary,
		CategoryID:        req.CategoryID,
		Status:            req.Status,
		TagIDs:            req.TagIDs,
		SourceWorkOrderID: &workOrderID,
	}

	article, err := s.CreateKnowledgeArticle(ctx, createReq, creatorID)
	if err != nil {
		return nil, fmt.Errorf("创建知识库文章失败: %w", err)
	}

	return article, nil
}

// SearchKnowledgeArticles 搜索知识库文章
func (s *KnowledgeService) SearchKnowledgeArticles(ctx context.Context, req *SearchKnowledgeRequest) ([]models.KnowledgeArticle, int64, error) {
	var list []models.KnowledgeArticle
	var total int64

	query := s.db.WithContext(ctx).Model(&models.KnowledgeArticle{}).
		Where("status = ?", models.KnowledgeArticleStatusPublished)

	if req.Keyword != "" {
		keyword := "%" + req.Keyword + "%"
		query = query.Where("title LIKE ? OR content LIKE ? OR summary LIKE ?", keyword, keyword, keyword)
	}
	if req.CategoryID != "" {
		query = query.Where("category_id = ?", req.CategoryID)
	}
	if req.TagID != "" {
		query = query.Joins("INNER JOIN sys_kb_article_tags ON sys_kb_article_tags.article_id = sys_knowledge_article.id").
			Where("sys_kb_article_tags.tag_id = ?", req.TagID)
	}

	// 先获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计知识库文章数量失败: %w", err)
	}

	// 设置分页参数，默认100条，最大500条
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 100
	} else if pageSize > 500 {
		pageSize = 500
	}

	pageNum := req.PageNum
	if pageNum < 0 {
		pageNum = 0
	}

	// 应用分页和预加载
	if err := query.
		Preload("Category").
		Preload("Tags").
		Order("sys_knowledge_article.created_at DESC").
		Limit(pageSize).
		Offset(pageNum * pageSize).
		Find(&list).Error; err != nil {
		return nil, 0, fmt.Errorf("搜索知识库文章失败: %w", err)
	}

	return list, total, nil
}

// IncrementViewCount 增加浏览次数
func (s *KnowledgeService) IncrementViewCount(ctx context.Context, id string) error {
	if err := s.db.WithContext(ctx).Model(&models.KnowledgeArticle{}).
		Where("id = ?", id).
		UpdateColumn("view_count", gorm.Expr("view_count + 1")).Error; err != nil {
		return fmt.Errorf("增加浏览次数失败: %w", err)
	}
	return nil
}

// IncrementLikeCount 增加点赞次数
func (s *KnowledgeService) IncrementLikeCount(ctx context.Context, id string) error {
	if err := s.db.WithContext(ctx).Model(&models.KnowledgeArticle{}).
		Where("id = ?", id).
		UpdateColumn("like_count", gorm.Expr("like_count + 1")).Error; err != nil {
		return fmt.Errorf("增加点赞次数失败: %w", err)
	}
	return nil
}

// ==================== 知识库分类方法 ====================

// GetKnowledgeCategoryList 获取知识库分类列表（树形结构）
func (s *KnowledgeService) GetKnowledgeCategoryList(ctx context.Context, req *KnowledgeCategoryListRequest) ([]models.KnowledgeCategory, error) {
	var list []models.KnowledgeCategory

	query := s.db.WithContext(ctx).Model(&models.KnowledgeCategory{})

	if req.ParentID == nil || *req.ParentID == "" {
		query = query.Where("parent_id IS NULL")
	} else {
		query = query.Where("parent_id = ?", *req.ParentID)
	}

	if req.Status != nil {
		query = query.Where("status = ?", *req.Status)
	}

	if err := query.
		Order("sort_order ASC, created_at ASC").
		Find(&list).Error; err != nil {
		return nil, fmt.Errorf("查询知识库分类列表失败: %w", err)
	}

	// 递归加载子分类
	for i := range list {
		if err := s.loadKnowledgeChildrenCategories(ctx, &list[i]); err != nil {
			applogger.Warnf("加载知识库分类 %s 的子分类失败: %v", list[i].ID, err)
		}
	}

	return list, nil
}

// loadKnowledgeChildrenCategories 递归加载子分类
func (s *KnowledgeService) loadKnowledgeChildrenCategories(ctx context.Context, category *models.KnowledgeCategory) error {
	var children []models.KnowledgeCategory

	if err := s.db.WithContext(ctx).
		Where("parent_id = ?", category.ID).
		Order("sort_order ASC, created_at ASC").
		Find(&children).Error; err != nil {
		return err
	}

	category.Children = children

	// 递归加载子分类的子分类
	for i := range children {
		if err := s.loadKnowledgeChildrenCategories(ctx, &children[i]); err != nil {
			applogger.Warnf("加载知识库子分类 %s 递归失败: %v", children[i].ID, err)
		}
	}

	return nil
}

// GetKnowledgeCategory 获取知识库分类详情
func (s *KnowledgeService) GetKnowledgeCategory(ctx context.Context, id string) (*models.KnowledgeCategory, error) {
	var category models.KnowledgeCategory

	if err := s.db.WithContext(ctx).
		Preload("Parent").
		Where("id = ?", id).
		First(&category).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("知识库分类不存在")
		}
		return nil, fmt.Errorf("查询知识库分类详情失败: %w", err)
	}

	return &category, nil
}

// CreateKnowledgeCategory 创建知识库分类
func (s *KnowledgeService) CreateKnowledgeCategory(ctx context.Context, req *KnowledgeCategoryCreateRequest, creatorID string) (*models.KnowledgeCategory, error) {
	category := &models.KnowledgeCategory{
		CategoryName: req.CategoryName,
		Description:  req.Description,
		ParentID:     req.ParentID,
		SortOrder:    req.SortOrder,
		Status:       models.KnowledgeArticleStatus(req.Status),
		BaseModel:    models.BaseModel{CreatedBy: creatorID, UpdatedBy: creatorID},
	}

	if err := s.db.WithContext(ctx).Create(category).Error; err != nil {
		return nil, fmt.Errorf("创建知识库分类失败: %w", err)
	}

	return category, nil
}

// UpdateKnowledgeCategory 更新知识库分类
func (s *KnowledgeService) UpdateKnowledgeCategory(ctx context.Context, id string, req *KnowledgeCategoryUpdateRequest, operatorID string) error {
	var category models.KnowledgeCategory
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&category).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("知识库分类不存在")
		}
		return fmt.Errorf("查询知识库分类失败: %w", err)
	}

	updates := map[string]interface{}{
		"updated_by": operatorID,
	}

	if req.CategoryName != nil {
		updates["category_name"] = *req.CategoryName
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.ParentID != nil {
		updates["parent_id"] = *req.ParentID
	}
	if req.SortOrder != nil {
		updates["sort_order"] = *req.SortOrder
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}

	if err := s.db.WithContext(ctx).Model(&category).Updates(updates).Error; err != nil {
		return fmt.Errorf("更新知识库分类失败: %w", err)
	}

	return nil
}

// DeleteKnowledgeCategory 删除知识库分类
func (s *KnowledgeService) DeleteKnowledgeCategory(ctx context.Context, id string) error {
	// 检查是否有子分类
	var count int64
	if err := s.db.WithContext(ctx).Model(&models.KnowledgeCategory{}).Where("parent_id = ?", id).Count(&count).Error; err != nil {
		return fmt.Errorf("检查子分类失败: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("该分类下有子分类，无法删除")
	}

	// 检查是否有关联文章
	if err := s.db.WithContext(ctx).Model(&models.KnowledgeArticle{}).Where("category_id = ?", id).Count(&count).Error; err != nil {
		return fmt.Errorf("检查关联文章失败: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("该分类下有关联文章，无法删除")
	}

	// 删除分类
	if err := s.db.WithContext(ctx).Delete(&models.KnowledgeCategory{}, id).Error; err != nil {
		return fmt.Errorf("删除知识库分类失败: %w", err)
	}

	return nil
}

// ==================== 知识库标签方法 ====================

// GetAllTags 获取所有标签
func (s *KnowledgeService) GetAllTags(ctx context.Context) ([]models.KnowledgeTag, error) {
	var tags []models.KnowledgeTag

	if err := s.db.WithContext(ctx).
		Order("use_count DESC, created_at ASC").
		Find(&tags).Error; err != nil {
		return nil, fmt.Errorf("查询标签列表失败: %w", err)
	}

	return tags, nil
}

// GetTagByName 根据名称获取标签
func (s *KnowledgeService) GetTagByName(ctx context.Context, name string) (*models.KnowledgeTag, error) {
	var tag models.KnowledgeTag

	if err := s.db.WithContext(ctx).
		Where("tag_name = ?", name).
		First(&tag).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("查询标签失败: %w", err)
	}

	return &tag, nil
}

// CreateTag 创建标签
func (s *KnowledgeService) CreateTag(ctx context.Context, name string) (*models.KnowledgeTag, error) {
	tag := &models.KnowledgeTag{
		ID:        uuid.New().String(),
		TagName:   name,
		UseCount:  0,
		CreatedAt: s.db.NowFunc(),
	}

	if err := s.db.WithContext(ctx).Create(tag).Error; err != nil {
		return nil, fmt.Errorf("创建标签失败: %w", err)
	}

	return tag, nil
}

// GetOrCreateTag 获取或创建标签
func (s *KnowledgeService) GetOrCreateTag(ctx context.Context, name string) (*models.KnowledgeTag, error) {
	// 先尝试获取
	tag, err := s.GetTagByName(ctx, name)
	if err != nil {
		return nil, err
	}

	// 如果不存在则创建
	if tag == nil {
		tag, err = s.CreateTag(ctx, name)
		if err != nil {
			return nil, err
		}
	}

	return tag, nil
}

// UpdateTag 更新标签
func (s *KnowledgeService) UpdateTag(ctx context.Context, id string, name string) error {
	if err := s.db.WithContext(ctx).Model(&models.KnowledgeTag{}).
		Where("id = ?", id).
		Update("tag_name", name).Error; err != nil {
		return fmt.Errorf("更新标签失败: %w", err)
	}
	return nil
}

// DeleteTag 删除标签
func (s *KnowledgeService) DeleteTag(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除文章标签关联
		if err := tx.Where("tag_id = ?", id).Delete(&models.KnowledgeArticleTag{}).Error; err != nil {
			return fmt.Errorf("删除文章标签关联失败: %w", err)
		}

		// 删除标签
		if err := tx.Delete(&models.KnowledgeTag{}, id).Error; err != nil {
			return fmt.Errorf("删除标签失败: %w", err)
		}

		return nil
	})
}

// ParseTagsFromContent 从内容中解析标签（#标签）
func (s *KnowledgeService) ParseTagsFromContent(content string) []string {
	// 简单实现：查找 #标签 格式
	tags := make(map[string]bool)
	words := strings.Fields(content)

	for _, word := range words {
		if strings.HasPrefix(word, "#") {
			tag := strings.Trim(word, "#")
			if tag != "" && len(tag) <= 20 {
				tags[tag] = true
			}
		}
	}

	result := make([]string, 0, len(tags))
	for tag := range tags {
		result = append(result, tag)
	}

	return result
}

// AutoCreateTagsFromContent 从内容中自动创建标签
func (s *KnowledgeService) AutoCreateTagsFromContent(ctx context.Context, articleID string, content string) error {
	// 解析标签
	tagNames := s.ParseTagsFromContent(content)
	if len(tagNames) == 0 {
		return nil
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, tagName := range tagNames {
			// 获取或创建标签
			tag, err := s.GetOrCreateTag(ctx, tagName)
			if err != nil {
				return fmt.Errorf("获取或创建标签失败: %w", err)
			}

			// 检查是否已经关联
			var count int64
			if err := tx.Model(&models.KnowledgeArticleTag{}).
				Where("article_id = ? AND tag_id = ?", articleID, tag.ID).
				Count(&count).Error; err != nil {
				return fmt.Errorf("检查标签关联失败: %w", err)
			}

			if count == 0 {
				// 创建关联
				articleTag := &models.KnowledgeArticleTag{
					ArticleID: articleID,
					TagID:     tag.ID,
					CreatedAt: tx.NowFunc(),
				}
				if err := tx.Create(articleTag).Error; err != nil {
					return fmt.Errorf("关联标签失败: %w", err)
				}

				// 更新标签使用次数
				if err := tx.Model(&models.KnowledgeTag{}).
					Where("id = ?", tag.ID).
					UpdateColumn("use_count", gorm.Expr("use_count + 1")).Error; err != nil {
					return fmt.Errorf("更新标签使用次数失败: %w", err)
				}
			}
		}

		return nil
	})
}
