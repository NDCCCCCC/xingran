package knowledge

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services"
)

// SetupKnowledgeRouter 设置知识库路由
func SetupKnowledgeRouter(r *gin.RouterGroup, core *core.Core) {
	// 创建知识库服务
	knowledgeService := services.NewKnowledgeService(core.DB.GetDB())

	// 创建处理器
	articleHandler := NewArticleHandler(knowledgeService)
	categoryHandler := NewCategoryHandler(knowledgeService)
	tagHandler := NewTagHandler(knowledgeService)

	// 文章路由
	articles := r.Group("/articles")
	{
		articles.POST("/list", articleHandler.List)
		articles.POST("/search", articleHandler.Search)
		articles.POST("", articleHandler.Create)
		articles.POST("/:id", articleHandler.GetByID)
		articles.POST("/:id/update", articleHandler.Update)
		articles.POST("/:id/delete", articleHandler.Delete)
		articles.POST("/:id/like", articleHandler.Like)
	}

	// 分类路由
	categories := r.Group("/categories")
	{
		categories.POST("/list", categoryHandler.List)
		categories.POST("", categoryHandler.Create)
		categories.POST("/:id", categoryHandler.GetByID)
		categories.POST("/:id/update", categoryHandler.Update)
		categories.POST("/:id/delete", categoryHandler.Delete)
	}

	// 标签路由
	tags := r.Group("/tags")
	{
		tags.POST("/all", tagHandler.GetAll)
		tags.POST("", tagHandler.Create)
		tags.POST("/:id/update", tagHandler.Update)
		tags.POST("/:id/delete", tagHandler.Delete)
	}

	// 工单转换路由（复用文章处理器）
	workorders := r.Group("/workorders")
	{
		workorders.POST("/:id", articleHandler.ConvertFromWorkOrder)
	}
}

// SetupKnowledgeViewRouter 设置知识库查看路由（无需权限）
func SetupKnowledgeViewRouter(r *gin.RouterGroup, core *core.Core) {
	// 创建知识库服务
	knowledgeService := services.NewKnowledgeService(core.DB.GetDB())

	// 创建处理器
	articleHandler := NewArticleHandler(knowledgeService)
	categoryHandler := NewCategoryHandler(knowledgeService)

	// 文章查看路由
	articles := r.Group("/articles")
	{
		articles.POST("/search", articleHandler.Search)
		articles.POST("/:id", articleHandler.GetByID)
	}

	// 分类查看路由
	categories := r.Group("/categories")
	{
		categories.POST("/list", categoryHandler.List)
		categories.POST("/:id", categoryHandler.GetByID)
	}
}
