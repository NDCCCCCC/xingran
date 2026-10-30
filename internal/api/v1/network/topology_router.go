package network

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/services/topology"
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
		// MAC过滤规则管理
		rules := topology.Group("/rules")
		{
			rules.POST("/list", topologyHandler.ListRules)         // 查询规则列表
			rules.POST("", topologyHandler.CreateRule)              // 创建规则
			rules.POST("/:id", topologyHandler.GetRule)             // 获取规则详情
			rules.POST("/:id/update", topologyHandler.UpdateRule)   // 更新规则
			rules.POST("/:id/delete", topologyHandler.DeleteRule)   // 删除规则
		}

		// 调试端点
		topology.GET("/rules/effective", topologyHandler.GetEffectiveRule) // 获取设备的有效规则
	}
}
