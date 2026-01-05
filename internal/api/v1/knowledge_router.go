package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
)

// SetupKnowledgeArticlesRouter 设置知识库文章路由
func SetupKnowledgeArticlesRouter(r *gin.RouterGroup, core *core.Core) {
	r.POST("/list", listKnowledgeArticles(core))
	r.POST("", createKnowledgeArticle(core))
	r.POST("/search", searchKnowledgeArticles(core))
	r.POST("/:id", getKnowledgeArticle(core))
	r.POST("/:id/update", updateKnowledgeArticle(core))
	r.POST("/:id/delete", deleteKnowledgeArticle(core))
	r.POST("/:id/like", likeKnowledgeArticle(core))
}

// SetupKnowledgeCategoriesRouter 设置知识库分类路由
func SetupKnowledgeCategoriesRouter(r *gin.RouterGroup, core *core.Core) {
	r.POST("/list", listKnowledgeCategories(core))
	r.POST("/:id", getKnowledgeCategory(core))
	r.POST("", createKnowledgeCategory(core))
	r.POST("/:id/update", updateKnowledgeCategory(core))
	r.POST("/:id/delete", deleteKnowledgeCategory(core))
}

// SetupKnowledgeTagsRouter 设置知识库标签路由
func SetupKnowledgeTagsRouter(r *gin.RouterGroup, core *core.Core) {
	r.POST("/all", getAllTags(core))
	r.POST("", createTag(core))
	r.POST("/:id/update", updateTag(core))
	r.POST("/:id/delete", deleteTag(core))
}
