package knowledge

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	knowledgeServices "github.com/xingran-next/xingran-go-backend/internal/services/knowledge"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
)

// SetupArticleRouter 设置知识库文章路由
func SetupArticleRouter(r *gin.RouterGroup, core *core.Core) {
	knowledgeService := createKnowledgeService(core)
	articleHandler := NewArticleHandler(knowledgeService).WithCore(core)

	r.POST("/list", articleHandler.List)
	r.POST("/statistics", articleHandler.Statistics)
	r.POST("/search", articleHandler.Search)
	r.POST("", articleHandler.Create)
	r.POST("/:id", articleHandler.GetByID)
	r.POST("/:id/update", articleHandler.Update)
	r.POST("/:id/delete", articleHandler.Delete)
	r.POST("/:id/like", articleHandler.Like)
}

// SetupCategoryRouter 设置知识库分类路由
func SetupCategoryRouter(r *gin.RouterGroup, core *core.Core) {
	knowledgeService := createKnowledgeService(core)
	categoryHandler := NewCategoryHandler(knowledgeService).WithCore(core)

	r.POST("/list", categoryHandler.List)
	r.POST("", categoryHandler.Create)
	r.POST("/:id", categoryHandler.GetByID)
	r.POST("/:id/update", categoryHandler.Update)
	r.POST("/:id/delete", categoryHandler.Delete)
}

// SetupTagRouter 设置知识库标签路由
func SetupTagRouter(r *gin.RouterGroup, core *core.Core) {
	knowledgeService := createKnowledgeService(core)
	tagHandler := NewTagHandler(knowledgeService).WithCore(core)

	r.POST("/all", tagHandler.GetAll)
	r.POST("", tagHandler.Create)
	r.POST("/:id/update", tagHandler.Update)
	r.POST("/:id/delete", tagHandler.Delete)
}

// SetupWorkOrderRouter 设置工单转知识库路由
func SetupWorkOrderRouter(r *gin.RouterGroup, core *core.Core) {
	knowledgeService := createKnowledgeService(core)
	articleHandler := NewArticleHandler(knowledgeService).WithCore(core)

	r.POST("/:id", articleHandler.ConvertFromWorkOrder)
}

// SetupKnowledgeViewRouter 设置知识库查看路由（无需权限）
func SetupKnowledgeViewRouter(r *gin.RouterGroup, core *core.Core) {
	knowledgeService := createKnowledgeService(core)

	articleHandler := NewArticleHandler(knowledgeService).WithCore(core)
	categoryHandler := NewCategoryHandler(knowledgeService).WithCore(core)

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

// createKnowledgeService 创建知识库服务（带缓存）
func createKnowledgeService(core *core.Core) knowledgeServices.KnowledgeCacheService {
	var cacheProvider systemServices.CacheProvider
	if core.DataCacheService != nil {
		cacheProvider = systemServices.NewCacheProvider(core.DataCacheService)
	} else {
		cacheProvider = &systemServices.NoOpCacheProvider{}
	}

	return knowledgeServices.NewKnowledgeServiceWithCache(
		core.DB.GetDB(),
		cacheProvider,
		core.CacheConfigService,
	)
}
