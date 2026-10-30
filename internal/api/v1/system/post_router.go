package system

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
)

// PostWriteRouteGroups 岗位**写操作**路由分组（85-02 RBAC-04，范式对齐 UserWriteRouteGroups）。
//
// 对应关系：Add → system:post:add；Edit → system:post:edit（编辑）；
// Remove → system:post:remove（删除 + 批量删除）。
type PostWriteRouteGroups struct {
	Add    *gin.RouterGroup
	Edit   *gin.RouterGroup
	Remove *gin.RouterGroup
}

// SetupPostRouter 设置岗位**读**路由
//
// 读操作保留组级 ANY-of 权限（system:post:list / system:post:view，
// 由 internal/api/router.go 施加）；写操作不在本函数注册——见 SetupPostWriteRouter。
//
// ⚠️ 读组与写组必须是**兄弟组**（不是父子）：gin 的 Group() 会继承父组中间件链，
// 若写组挂在读组之下，读组的 ANY-of 会先于严格权限放行写操作。
func SetupPostRouter(r *gin.RouterGroup, core *core.Core) {
	handler := newPostHandler(core)

	// 读路由
	r.POST("/list", handler.List)
	// 岗位统计(专用 COUNT 聚合,不依赖分页列表)
	r.POST("/statistics", handler.Statistics)
	r.POST("/all", handler.GetAll)
	r.POST("/enabled", handler.GetAllEnabled)
	r.POST("/:id", handler.GetByID)
}

// SetupPostWriteRouter 设置岗位**写**路由（85-02 RBAC-04 逐动作拆分）。
//
// 每个写操作落到单一权限点的严格权限子组上（分组由 router.go 创建）。
func SetupPostWriteRouter(groups PostWriteRouteGroups, core *core.Core) {
	handler := newPostHandler(core)

	// 新增
	groups.Add.POST("", handler.Create)
	// 编辑
	groups.Edit.POST("/:id/update", handler.Update)
	// 删除 + 批量删除
	groups.Remove.POST("/:id/delete", handler.Delete)
	groups.Remove.POST("/batch-delete", handler.BatchDelete)
}

// newPostHandler 构造岗位处理器（含缓存适配；读/写路由共用）。
func newPostHandler(core *core.Core) *PostHandler {
	// 创建缓存提供者适配器
	cacheProvider := systemServices.NewCacheProvider(core.DataCacheService)

	// 创建带缓存的岗位服务
	var postService systemServices.PostService
	if core.DataCacheService != nil {
		postService = systemServices.NewPostServiceWithCache(
			core.DB.GetDB(),
			cacheProvider,
			core.CacheConfigService,
		)
	} else {
		postService = systemServices.NewPostService(core.DB.GetDB())
	}

	// 创建Handler
	return NewPostHandler(postService).WithCore(core)
}
