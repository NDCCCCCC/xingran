package addomain

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/addomain"
)

// SetupGroupSyncRouter 设置组同步API路由
func SetupGroupSyncRouter(r *gin.RouterGroup, adService *addomain.ADDomainService) {
	handler := NewGroupSyncHandler(adService)

	// 应用认证和权限中间件
	// r.Use(middleware.Auth()) // TODO: 根据项目需求启用认证
	// r.Use(middleware.Permissions()) // TODO: 根据项目需求启用权限检查

	group := r.Group("/groups")
	{
		// 同步操作
		group.POST("/sync", handler.SyncAllMembers)                    // 手动触发同步所有成员
		group.GET("/sync/status", handler.GetSyncStatus)               // 查询同步状态
		group.POST("/sync/dept/:deptId", handler.SyncDeptMembers)      // 同步指定部门
		group.POST("/exclusive", handler.EnsureExclusiveMembership)    // 排他性成员关系

		// 映射管理
		group.GET("/mappings", handler.ListMappings)        // 查询映射列表
		group.POST("/mappings", handler.CreateMapping)      // 创建映射
		group.GET("/mappings/:id", handler.GetMapping)      // 获取映射详情
		group.PUT("/mappings/:id", handler.UpdateMapping)   // 更新映射
		group.DELETE("/mappings/:id", handler.DeleteMapping) // 删除映射

		// 自动映射
		group.POST("/automap", handler.AutoMapDepartments) // 批量自动映射
	}
}
