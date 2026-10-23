package system

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	addomainServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/addomain"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/middleware"
)

// SetupManagerRuleRouter 经理同步规则/机构维护路由。
// 权限：ops:manager:rule:* 与 ops:manager:org:*（migration_221 菜单 seed 对应按钮权限）。
func SetupManagerRuleRouter(r *gin.RouterGroup, core *core.Core) {
	db := core.GetDB()
	ruleSvc := addomainServices.NewManagerRuleService(db)
	orgSvc := addomainServices.NewManagerOrgService(db)
	ruleHandler := NewManagerRuleHandler(core, ruleSvc)
	orgHandler := NewManagerOrgHandler(core, orgSvc)

	// 规则维护
	rules := r.Group("/manager-rules")
	rules.Use(middleware.RequirePermissions([]string{
		"ops:manager:rule:list",
		"ops:manager:rule:edit",
		"ops:manager:rule:delete",
	}, core))
	{
		rules.POST("/list", ruleHandler.ListRules)
		rules.POST("", ruleHandler.CreateRule)
		rules.POST("/:id/update", ruleHandler.UpdateRule)
		rules.POST("/:id/delete", ruleHandler.DeleteRule)
	}

	// 经理预览（用户列表「经理」列展示用）：仅要求登录态（父组已挂 auth），
	// 不要求经理规则权限——能看用户列表的管理员都可见；数据为内存计算结果，不落库。
	preview := r.Group("/manager-rules")
	{
		preview.POST("/manager-preview", ruleHandler.ManagerPreview)
	}

	// 机构维护
	orgs := r.Group("/manager-orgs")
	orgs.Use(middleware.RequirePermissions([]string{
		"ops:manager:org:list",
		"ops:manager:org:edit",
		"ops:manager:org:delete",
	}, core))
	{
		orgs.POST("/list", orgHandler.ListOrgs)
		orgs.POST("", orgHandler.CreateOrg)
		orgs.POST("/:id/update", orgHandler.UpdateOrg)
		orgs.POST("/:id/delete", orgHandler.DeleteOrg)
		// 从部门架构自动生成机构映射（幂等，不覆盖手工修改）
		orgs.POST("/generate-from-dept", orgHandler.GenerateOrgsFromDept)
	}
}
