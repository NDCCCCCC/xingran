package knowledge

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	knowledgeServices "github.com/xingran-next/xingran-go-backend/internal/services/knowledge"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// ArticleHandler 知识库文章处理器
type ArticleHandler struct {
	service knowledgeServices.KnowledgeCacheService
	core    *core.Core
}

// NewArticleHandler 创建文章处理器实例
func NewArticleHandler(service knowledgeServices.KnowledgeCacheService) *ArticleHandler {
	return &ArticleHandler{service: service}
}

// WithCore 注入 core 依赖（操作日志记录所需）。Phase 34 操作日志全模块覆盖。
func (h *ArticleHandler) WithCore(core *core.Core) *ArticleHandler {
	if h != nil && core != nil {
		h.core = core
	}
	return h
}

// List 查询知识库文章列表
// @Summary 查询知识库文章列表
// @Description 分页查询知识库文章列表，支持多条件过滤
// @Tags 知识库文章
// @Accept json
// @Produce json
// @Param request body services.KnowledgeArticleListRequest true "查询条件"
// @Success 200 {object} response.Response{data=response.PageResponse}
// @Router /knowledge/articles/list [post]
func (h *ArticleHandler) List(c *gin.Context) {
	var rawReq map[string]interface{}
	if err := c.ShouldBindJSON(&rawReq); err != nil {
		rawReq = make(map[string]interface{})
	}

	req := services.KnowledgeArticleListRequest{
		BaseListRequest: base.BaseListRequest{
			Current: func() int {
				if v, ok := rawReq["current"].(float64); ok {
					return int(v)
				}
				return 1
			}(),
			PageSize: func() int {
				if v, ok := rawReq["pageSize"].(float64); ok {
					return int(v)
				}
				return 10
			}(),
			OrderByColumn: func() string {
				if v, ok := rawReq["orderByColumn"].(string); ok {
					return v
				}
				return ""
			}(),
			IsAsc: func() *bool {
				if v, ok := rawReq["isAsc"].(bool); ok {
					return &v
				}
				return nil
			}(),
		},
	}

	if val, ok := rawReq["title"].(string); ok && val != "" {
		req.Title = val
	}
	if val, ok := rawReq["categoryId"].(string); ok && val != "" {
		req.CategoryID = val
	}
	if val, ok := rawReq["tagId"].(string); ok && val != "" {
		req.TagID = val
	}
	if val, ok := rawReq["status"].(float64); ok {
		status := int(val)
		req.Status = &status
	}
	if val, ok := rawReq["createdBy"].(string); ok && val != "" {
		req.CreatedBy = val
	}

	list, total, err := h.service.GetKnowledgeArticleList(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Page(c, list, total, req.Current, req.PageSize)
}

// Statistics 知识库文章统计(总数/草稿/已发布/累计浏览/累计点赞)
// @Summary 知识库文章统计
// @Description 返回文章总数/状态计数及累计浏览点赞数,供列表页统计卡片使用;用 COUNT 聚合而非按当前页 list 计算
// @Tags 知识库文章
// @Produce json
// @Success 200 {object} response.Response
// @Router /knowledge/articles/statistics [post]
func (h *ArticleHandler) Statistics(c *gin.Context) {
	result, err := h.service.GetArticleStatistics(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

// GetByID 获取知识库文章详情
// @Summary 获取知识库文章详情
// @Description 根据文章ID获取知识库文章详细信息，并增加浏览次数
// @Tags 知识库文章
// @Accept json
// @Produce json
// @Param id path string true "文章ID"
// @Success 200 {object} response.Response
// @Router /knowledge/articles/:id [post]
func (h *ArticleHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("文章ID"))
		return
	}

	// 增加浏览次数（best-effort，不影响主流程）
	_ = h.service.IncrementViewCount(c.Request.Context(), id)

	article, err := h.service.GetKnowledgeArticle(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, article)
}

// Create 创建知识库文章
// @Summary 创建知识库文章
// @Description 创建新的知识库文章
// @Tags 知识库文章
// @Accept json
// @Produce json
// @Param request body services.KnowledgeArticleCreateRequest true "文章信息"
// @Success 200 {object} response.Response
// @Router /knowledge/articles [post]
func (h *ArticleHandler) Create(c *gin.Context) {
	var req services.KnowledgeArticleCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, apperrors.Unauthorized())
		return
	}

	article, err := h.service.CreateKnowledgeArticle(c.Request.Context(), &req, userID.(string))
	if err != nil {
		response.Error(c, err)
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "知识库文章", operlog.OperTypeCreate)
	response.Success(c, article)
}

// Update 更新知识库文章
// @Summary 更新知识库文章
// @Description 更新知识库文章信息
// @Tags 知识库文章
// @Accept json
// @Produce json
// @Param id path string true "文章ID"
// @Param request body services.KnowledgeArticleUpdateRequest true "文章信息"
// @Success 200 {object} response.Response
// @Router /knowledge/articles/:id/update [post]
func (h *ArticleHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("文章ID"))
		return
	}

	var req services.KnowledgeArticleUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, apperrors.Unauthorized())
		return
	}

	if err := h.service.UpdateKnowledgeArticle(c.Request.Context(), id, &req, userID.(string)); err != nil {
		response.Error(c, err)
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "知识库文章", operlog.OperTypeUpdate)
	response.Success(c, gin.H{"message": "更新成功"})
}

// Delete 删除知识库文章
// @Summary 删除知识库文章
// @Description 删除指定的知识库文章
// @Tags 知识库文章
// @Accept json
// @Produce json
// @Param id path string true "文章ID"
// @Success 200 {object} response.Response
// @Router /knowledge/articles/:id/delete [post]
func (h *ArticleHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("文章ID"))
		return
	}

	if err := h.service.DeleteKnowledgeArticle(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "知识库文章", operlog.OperTypeDelete)
	response.Success(c, gin.H{"message": "删除成功"})
}

// ConvertFromWorkOrder 将工单转换为知识库文章
// @Summary 将工单转换为知识库文章
// @Description 将已完成的工单转换为知识库文章
// @Tags 知识库文章
// @Accept json
// @Produce json
// @Param id path string true "工单ID"
// @Param request body services.ConvertWorkOrderToArticleRequest true "文章信息"
// @Success 200 {object} response.Response
// @Router /knowledge/workorders/:id [post]
func (h *ArticleHandler) ConvertFromWorkOrder(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("工单ID"))
		return
	}

	var req services.ConvertWorkOrderToArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, apperrors.Unauthorized())
		return
	}

	article, err := h.service.ConvertWorkOrderToArticle(c.Request.Context(), id, &req, userID.(string))
	if err != nil {
		response.Error(c, err)
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "知识库文章", operlog.OperTypeCreate)
	response.Success(c, article)
}

// Search 搜索知识库文章
// @Summary 搜索知识库文章
// @Description 根据关键词搜索已发布的知识库文章
// @Tags 知识库文章
// @Accept json
// @Produce json
// @Param request body services.SearchKnowledgeRequest true "搜索条件"
// @Success 200 {object} response.Response{data=object}
// @Router /knowledge/articles/search [post]
func (h *ArticleHandler) Search(c *gin.Context) {
	var req services.SearchKnowledgeRequest
	// 允许空请求体
	if err := c.ShouldBindJSON(&req); err != nil {
		// 使用默认值
		if req.PageSize <= 0 {
			req.PageSize = 100
		}
		if req.PageNum < 0 {
			req.PageNum = 0
		}
	}

	articles, total, err := h.service.SearchKnowledgeArticles(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{
		"list":  articles,
		"total": total,
	})
}

// Like 点赞知识库文章
// @Summary 点赞知识库文章
// @Description 增加知识库文章的点赞次数
// @Tags 知识库文章
// @Accept json
// @Produce json
// @Param id path string true "文章ID"
// @Success 200 {object} response.Response
// @Router /knowledge/articles/:id/like [post]
func (h *ArticleHandler) Like(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("文章ID"))
		return
	}

	if err := h.service.IncrementLikeCount(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "知识库文章", operlog.OperTypeOther)
	response.Success(c, gin.H{"message": "点赞成功"})
}

// ==================== 分类处理器 ====================

// CategoryHandler 知识库分类处理器
type CategoryHandler struct {
	service knowledgeServices.KnowledgeCacheService
	core    *core.Core
}

// NewCategoryHandler 创建分类处理器实例
func NewCategoryHandler(service knowledgeServices.KnowledgeCacheService) *CategoryHandler {
	return &CategoryHandler{service: service}
}

// WithCore 注入 core 依赖（操作日志记录所需）。Phase 34 操作日志全模块覆盖。
func (h *CategoryHandler) WithCore(core *core.Core) *CategoryHandler {
	if h != nil && core != nil {
		h.core = core
	}
	return h
}

// List 查询知识库分类列表
// @Summary 查询知识库分类列表
// @Description 查询知识库分类列表（树形结构）
// @Tags 知识库分类
// @Accept json
// @Produce json
// @Param request body services.KnowledgeCategoryListRequest true "查询条件"
// @Success 200 {object} response.Response
// @Router /knowledge/categories/list [post]
func (h *CategoryHandler) List(c *gin.Context) {
	var req services.KnowledgeCategoryListRequest
	// 允许空请求体
	if err := c.ShouldBindJSON(&req); err != nil {
		// 使用默认值
		_ = err
	}

	categories, err := h.service.GetKnowledgeCategoryList(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, categories)
}

// GetByID 获取知识库分类详情
// @Summary 获取知识库分类详情
// @Description 根据分类ID获取知识库分类详细信息
// @Tags 知识库分类
// @Accept json
// @Produce json
// @Param id path string true "分类ID"
// @Success 200 {object} response.Response
// @Router /knowledge/categories/:id [post]
func (h *CategoryHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("分类ID"))
		return
	}

	category, err := h.service.GetKnowledgeCategory(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, category)
}

// Create 创建知识库分类
// @Summary 创建知识库分类
// @Description 创建新的知识库分类
// @Tags 知识库分类
// @Accept json
// @Produce json
// @Param request body services.KnowledgeCategoryCreateRequest true "分类信息"
// @Success 200 {object} response.Response
// @Router /knowledge/categories [post]
func (h *CategoryHandler) Create(c *gin.Context) {
	var req services.KnowledgeCategoryCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, apperrors.Unauthorized())
		return
	}

	category, err := h.service.CreateKnowledgeCategory(c.Request.Context(), &req, userID.(string))
	if err != nil {
		response.Error(c, err)
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "知识库分类", operlog.OperTypeCreate)
	response.Success(c, category)
}

// Update 更新知识库分类
// @Summary 更新知识库分类
// @Description 更新知识库分类信息
// @Tags 知识库分类
// @Accept json
// @Produce json
// @Param id path string true "分类ID"
// @Param request body services.KnowledgeCategoryUpdateRequest true "分类信息"
// @Success 200 {object} response.Response
// @Router /knowledge/categories/:id/update [post]
func (h *CategoryHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("分类ID"))
		return
	}

	var req services.KnowledgeCategoryUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, apperrors.Unauthorized())
		return
	}

	if err := h.service.UpdateKnowledgeCategory(c.Request.Context(), id, &req, userID.(string)); err != nil {
		response.Error(c, err)
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "知识库分类", operlog.OperTypeUpdate)
	response.Success(c, gin.H{"message": "更新成功"})
}

// Delete 删除知识库分类
// @Summary 删除知识库分类
// @Description 删除指定的知识库分类
// @Tags 知识库分类
// @Accept json
// @Produce json
// @Param id path string true "分类ID"
// @Success 200 {object} response.Response
// @Router /knowledge/categories/:id/delete [post]
func (h *CategoryHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("分类ID"))
		return
	}

	if err := h.service.DeleteKnowledgeCategory(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "知识库分类", operlog.OperTypeDelete)
	response.Success(c, gin.H{"message": "删除成功"})
}

// ==================== 标签处理器 ====================

// TagHandler 知识库标签处理器
type TagHandler struct {
	service knowledgeServices.KnowledgeCacheService
	core    *core.Core
}

// NewTagHandler 创建标签处理器实例
func NewTagHandler(service knowledgeServices.KnowledgeCacheService) *TagHandler {
	return &TagHandler{service: service}
}

// WithCore 注入 core 依赖（操作日志记录所需）。Phase 34 操作日志全模块覆盖。
func (h *TagHandler) WithCore(core *core.Core) *TagHandler {
	if h != nil && core != nil {
		h.core = core
	}
	return h
}

// GetAll 获取所有标签
// @Summary 获取所有标签
// @Description 获取所有知识库标签列表
// @Tags 知识库标签
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Router /knowledge/tags/all [post]
func (h *TagHandler) GetAll(c *gin.Context) {
	tags, err := h.service.GetAllTags(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, tags)
}

// Create 创建标签
// @Summary 创建标签
// @Description 创建新的知识库标签
// @Tags 知识库标签
// @Accept json
// @Produce json
// @Param request body object{tagName=string} true "标签名称"
// @Success 200 {object} response.Response
// @Router /knowledge/tags [post]
func (h *TagHandler) Create(c *gin.Context) {
	var req struct {
		TagName string `json:"tagName" binding:"required,max=50"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	tag, err := h.service.CreateTag(c.Request.Context(), req.TagName)
	if err != nil {
		response.Error(c, err)
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "知识库标签", operlog.OperTypeCreate)
	response.Success(c, tag)
}

// Update 更新标签
// @Summary 更新标签
// @Description 更新知识库标签名称
// @Tags 知识库标签
// @Accept json
// @Produce json
// @Param id path string true "标签ID"
// @Param request body object{tagName=string} true "标签名称"
// @Success 200 {object} response.Response
// @Router /knowledge/tags/:id/update [post]
func (h *TagHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("标签ID"))
		return
	}

	var req struct {
		TagName string `json:"tagName" binding:"required,max=50"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	if err := h.service.UpdateTag(c.Request.Context(), id, req.TagName); err != nil {
		response.Error(c, err)
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "知识库标签", operlog.OperTypeUpdate)
	response.Success(c, gin.H{"message": "更新成功"})
}

// Delete 删除标签
// @Summary 删除标签
// @Description 删除指定的知识库标签
// @Tags 知识库标签
// @Accept json
// @Produce json
// @Param id path string true "标签ID"
// @Success 200 {object} response.Response
// @Router /knowledge/tags/:id/delete [post]
func (h *TagHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("标签ID"))
		return
	}

	if err := h.service.DeleteTag(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "知识库标签", operlog.OperTypeDelete)
	response.Success(c, gin.H{"message": "删除成功"})
}
