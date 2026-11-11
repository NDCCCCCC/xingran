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

	// 文章读操作（task-3 G1 整改：写端点已拆至下方 strict 写组）
	r.POST("/list", articleHandler.List)
	r.POST("/statistics", articleHandler.Statistics)
	r.POST("/search", articleHandler.Search)
	r.POST("/:id", articleHandler.GetByID)
	// 口径记录（task-3）：点赞为已登录用户的轻量交互写（只增计数、不改文章实体），
	// 保持挂读组不拆 strict——按 strict 拆分会让仅持 list 码的普通用户无法点赞。
	r.POST("/:id/like", articleHandler.Like)
}

// ArticleWriteRouteGroups 知识库文章写操作路由分组（task-3 G1：逐动作拆分）。
type ArticleWriteRouteGroups struct {
	Add    *gin.RouterGroup
	Edit   *gin.RouterGroup
	Delete *gin.RouterGroup
}

// SetupArticleWriteRouter 设置知识库文章写路由（严格写组接线：逐动作单一权限点）。
//
//	新增   POST /knowledge/articles            → ops:knowledge:article:add
//	编辑   POST /knowledge/articles/:id/update → ops:knowledge:article:edit
//	删除   POST /knowledge/articles/:id/delete → ops:knowledge:article:delete
func SetupArticleWriteRouter(groups ArticleWriteRouteGroups, core *core.Core) {
	knowledgeService := createKnowledgeService(core)
	articleHandler := NewArticleHandler(knowledgeService).WithCore(core)

	groups.Add.POST("", articleHandler.Create)
	groups.Edit.POST("/:id/update", articleHandler.Update)
	groups.Delete.POST("/:id/delete", articleHandler.Delete)
}

// SetupCategoryRouter 设置知识库分类路由
func SetupCategoryRouter(r *gin.RouterGroup, core *core.Core) {
	knowledgeService := createKnowledgeService(core)
	categoryHandler := NewCategoryHandler(knowledgeService).WithCore(core)

	// 分类读操作（task-3 G1 整改：写端点已拆至下方 strict 写组）
	r.POST("/list", categoryHandler.List)
	r.POST("/:id", categoryHandler.GetByID)
}

// CategoryWriteRouteGroups 知识库分类写操作路由分组（task-3 G1）。
type CategoryWriteRouteGroups struct {
	Add    *gin.RouterGroup
	Edit   *gin.RouterGroup
	Delete *gin.RouterGroup
}

// SetupCategoryWriteRouter 设置知识库分类写路由（严格写组）。
//
//	新增   POST /knowledge/categories            → ops:knowledge:category:add
//	编辑   POST /knowledge/categories/:id/update → ops:knowledge:category:edit
//	删除   POST /knowledge/categories/:id/delete → ops:knowledge:category:delete
func SetupCategoryWriteRouter(groups CategoryWriteRouteGroups, core *core.Core) {
	knowledgeService := createKnowledgeService(core)
	categoryHandler := NewCategoryHandler(knowledgeService).WithCore(core)

	groups.Add.POST("", categoryHandler.Create)
	groups.Edit.POST("/:id/update", categoryHandler.Update)
	groups.Delete.POST("/:id/delete", categoryHandler.Delete)
}

// SetupTagRouter 设置知识库标签路由
func SetupTagRouter(r *gin.RouterGroup, core *core.Core) {
	knowledgeService := createKnowledgeService(core)
	tagHandler := NewTagHandler(knowledgeService).WithCore(core)

	// 标签读操作（task-3 G1 整改：写端点已拆至下方 strict 写组）
	r.POST("/all", tagHandler.GetAll)
}

// TagWriteRouteGroups 知识库标签写操作路由分组（task-3 G1）。
type TagWriteRouteGroups struct {
	Add    *gin.RouterGroup
	Edit   *gin.RouterGroup
	Delete *gin.RouterGroup
}

// SetupTagWriteRouter 设置知识库标签写路由（严格写组）。
//
//	新增   POST /knowledge/tags            → ops:knowledge:tag:add
//	编辑   POST /knowledge/tags/:id/update → ops:knowledge:tag:edit
//	删除   POST /knowledge/tags/:id/delete → ops:knowledge:tag:delete
func SetupTagWriteRouter(groups TagWriteRouteGroups, core *core.Core) {
	knowledgeService := createKnowledgeService(core)
	tagHandler := NewTagHandler(knowledgeService).WithCore(core)

	groups.Add.POST("", tagHandler.Create)
	groups.Edit.POST("/:id/update", tagHandler.Update)
	groups.Delete.POST("/:id/delete", tagHandler.Delete)
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
