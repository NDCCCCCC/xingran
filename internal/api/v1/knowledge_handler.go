package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// ==================== 知识库文章 Handler ====================

// listKnowledgeArticles 查询知识库文章列表
func listKnowledgeArticles(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req services.KnowledgeArticleListRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			// 允许空请求体
		}

		knowledgeSvc := services.NewKnowledgeService(core.DB.GetDB())
		list, total, err := knowledgeSvc.GetKnowledgeArticleList(c.Request.Context(), &req)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		pageResp := response.PageResponse{
			List:     list,
			Total:    total,
			Current:  req.Current,
			PageSize: req.PageSize,
		}
		response.Success(c, pageResp)
	}
}

// getKnowledgeArticle 获取知识库文章详情
func getKnowledgeArticle(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "文章ID不能为空")
			return
		}

		knowledgeSvc := services.NewKnowledgeService(core.DB.GetDB())

		// 增加浏览次数
		knowledgeSvc.IncrementViewCount(c.Request.Context(), id)

		article, err := knowledgeSvc.GetKnowledgeArticle(c.Request.Context(), id)
		if err != nil {
			response.Error(c, response.ErrNotFound, err.Error())
			return
		}

		response.Success(c, article)
	}
}

// createKnowledgeArticle 创建知识库文章
func createKnowledgeArticle(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req services.KnowledgeArticleCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		userID, _ := c.Get("user_id")
		knowledgeSvc := services.NewKnowledgeService(core.DB.GetDB())
		article, err := knowledgeSvc.CreateKnowledgeArticle(c.Request.Context(), &req, userID.(string))
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		// 记录操作日志
		recordOperLog(c, core, "知识库", OperTypeCreate)

		response.Success(c, article)
	}
}

// updateKnowledgeArticle 更新知识库文章
func updateKnowledgeArticle(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "文章ID不能为空")
			return
		}

		var req services.KnowledgeArticleUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		userID, _ := c.Get("user_id")
		knowledgeSvc := services.NewKnowledgeService(core.DB.GetDB())
		if err := knowledgeSvc.UpdateKnowledgeArticle(c.Request.Context(), id, &req, userID.(string)); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		// 记录操作日志
		recordOperLog(c, core, "知识库", OperTypeUpdate)

		response.Success(c, gin.H{"message": "更新成功"})
	}
}

// deleteKnowledgeArticle 删除知识库文章
func deleteKnowledgeArticle(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "文章ID不能为空")
			return
		}

		knowledgeSvc := services.NewKnowledgeService(core.DB.GetDB())
		if err := knowledgeSvc.DeleteKnowledgeArticle(c.Request.Context(), id); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		// 记录操作日志
		recordOperLog(c, core, "知识库", OperTypeDelete)

		response.Success(c, gin.H{"message": "删除成功"})
	}
}

// convertWorkOrderToArticle 将工单转换为知识库文章
func convertWorkOrderToArticle(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "工单ID不能为空")
			return
		}

		var req services.ConvertWorkOrderToArticleRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		userID, _ := c.Get("user_id")
		knowledgeSvc := services.NewKnowledgeService(core.DB.GetDB())
		article, err := knowledgeSvc.ConvertWorkOrderToArticle(c.Request.Context(), id, &req, userID.(string))
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		// 记录操作日志
		recordOperLog(c, core, "知识库", OperTypeCreate)

		response.Success(c, article)
	}
}

// searchKnowledgeArticles 搜索知识库文章
func searchKnowledgeArticles(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req services.SearchKnowledgeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			// 允许空请求体
		}

		knowledgeSvc := services.NewKnowledgeService(core.DB.GetDB())
		articles, total, err := knowledgeSvc.SearchKnowledgeArticles(c.Request.Context(), &req)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{
			"list":  articles,
			"total": total,
		})
	}
}

// likeKnowledgeArticle 点赞知识库文章
func likeKnowledgeArticle(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "文章ID不能为空")
			return
		}

		knowledgeSvc := services.NewKnowledgeService(core.DB.GetDB())
		if err := knowledgeSvc.IncrementLikeCount(c.Request.Context(), id); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{"message": "点赞成功"})
	}
}

// ==================== 知识库分类 Handler ====================

// listKnowledgeCategories 查询知识库分类列表
func listKnowledgeCategories(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req services.KnowledgeCategoryListRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			// 允许空请求体
		}

		knowledgeSvc := services.NewKnowledgeService(core.DB.GetDB())
		categories, err := knowledgeSvc.GetKnowledgeCategoryList(c.Request.Context(), &req)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, categories)
	}
}

// getKnowledgeCategory 获取知识库分类详情
func getKnowledgeCategory(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "分类ID不能为空")
			return
		}

		knowledgeSvc := services.NewKnowledgeService(core.DB.GetDB())
		category, err := knowledgeSvc.GetKnowledgeCategory(c.Request.Context(), id)
		if err != nil {
			response.Error(c, response.ErrNotFound, err.Error())
			return
		}

		response.Success(c, category)
	}
}

// createKnowledgeCategory 创建知识库分类
func createKnowledgeCategory(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req services.KnowledgeCategoryCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		userID, _ := c.Get("user_id")
		knowledgeSvc := services.NewKnowledgeService(core.DB.GetDB())
		category, err := knowledgeSvc.CreateKnowledgeCategory(c.Request.Context(), &req, userID.(string))
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		// 记录操作日志
		recordOperLog(c, core, "知识库分类", OperTypeCreate)

		response.Success(c, category)
	}
}

// updateKnowledgeCategory 更新知识库分类
func updateKnowledgeCategory(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "分类ID不能为空")
			return
		}

		var req services.KnowledgeCategoryUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		userID, _ := c.Get("user_id")
		knowledgeSvc := services.NewKnowledgeService(core.DB.GetDB())
		if err := knowledgeSvc.UpdateKnowledgeCategory(c.Request.Context(), id, &req, userID.(string)); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		// 记录操作日志
		recordOperLog(c, core, "知识库分类", OperTypeUpdate)

		response.Success(c, gin.H{"message": "更新成功"})
	}
}

// deleteKnowledgeCategory 删除知识库分类
func deleteKnowledgeCategory(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "分类ID不能为空")
			return
		}

		knowledgeSvc := services.NewKnowledgeService(core.DB.GetDB())
		if err := knowledgeSvc.DeleteKnowledgeCategory(c.Request.Context(), id); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		// 记录操作日志
		recordOperLog(c, core, "知识库分类", OperTypeDelete)

		response.Success(c, gin.H{"message": "删除成功"})
	}
}

// ==================== 知识库标签 Handler ====================

// getAllTags 获取所有标签
func getAllTags(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		knowledgeSvc := services.NewKnowledgeService(core.DB.GetDB())
		tags, err := knowledgeSvc.GetAllTags(c.Request.Context())
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, tags)
	}
}

// createTag 创建标签
func createTag(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			TagName string `json:"tagName" binding:"required,max=50"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		knowledgeSvc := services.NewKnowledgeService(core.DB.GetDB())
		tag, err := knowledgeSvc.CreateTag(c.Request.Context(), req.TagName)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		// 记录操作日志
		recordOperLog(c, core, "知识库标签", OperTypeCreate)

		response.Success(c, tag)
	}
}

// updateTag 更新标签
func updateTag(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "标签ID不能为空")
			return
		}

		var req struct {
			TagName string `json:"tagName" binding:"required,max=50"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		knowledgeSvc := services.NewKnowledgeService(core.DB.GetDB())
		if err := knowledgeSvc.UpdateTag(c.Request.Context(), id, req.TagName); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		// 记录操作日志
		recordOperLog(c, core, "知识库标签", OperTypeUpdate)

		response.Success(c, gin.H{"message": "更新成功"})
	}
}

// deleteTag 删除标签
func deleteTag(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "标签ID不能为空")
			return
		}

		knowledgeSvc := services.NewKnowledgeService(core.DB.GetDB())
		if err := knowledgeSvc.DeleteTag(c.Request.Context(), id); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		// 记录操作日志
		recordOperLog(c, core, "知识库标签", OperTypeDelete)

		response.Success(c, gin.H{"message": "删除成功"})
	}
}
