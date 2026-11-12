package network

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/services/topology"
	"github.com/xingran-next/xingran-go-backend/pkg/middleware"
	"github.com/xingran-next/xingran-go-backend/pkg/permission"
	"gorm.io/gorm"
)

// SetupTopologyRouter 设置拓扑管理路由
func SetupTopologyRouter(r *gin.RouterGroup, db *gorm.DB, core *core.Core) {
	// 创建过滤规则服务
	filterRuleService := topology.NewFilterRuleService(db)

	// 创建拓扑处理器
	topologyHandler := NewTopologyHandler(filterRuleService, db).WithCore(core)

	// 注册拓扑管理路由
	topology := r.Group("/topology")
	{
		// RBAC 接线（task-4）：MAC 过滤规则归 MAC 域——
		//   读（list/GET effective）→ ANY-of([network:mac:query])
		//   写（create/update/delete）→ strict(network:mac:rule:edit)
		macRuleRead := []string{string(permission.NetworkMacQuery)}
		macRuleEdit := []string{string(permission.NetworkMacRuleEdit)}

		// MAC过滤规则管理
		rules := topology.Group("/rules")
		{
			rules.POST("/list", middleware.RequirePermissions(macRuleRead, core), topologyHandler.ListRules)    // 查询规则列表
			rules.POST("", middleware.RequireStrictPermissions(macRuleEdit, core), topologyHandler.CreateRule)  // 创建规则
			rules.POST("/:id", middleware.RequirePermissions(macRuleRead, core), topologyHandler.GetRule)       // 获取规则详情
			rules.POST("/:id/update", middleware.RequireStrictPermissions(macRuleEdit, core), topologyHandler.UpdateRule) // 更新规则
			rules.POST("/:id/delete", middleware.RequireStrictPermissions(macRuleEdit, core), topologyHandler.DeleteRule) // 删除规则
		}

		// 调试端点
		topology.GET("/rules/effective", middleware.RequirePermissions(macRuleRead, core), topologyHandler.GetEffectiveRule) // 获取设备的有效规则
	}
}
