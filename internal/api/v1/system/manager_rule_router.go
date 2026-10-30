package system

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	addomainServices "github.com/xingran-next/xingran-go-backend/internal/services/addomain"
	"github.com/xingran-next/xingran-go-backend/pkg/middleware"
	"github.com/xingran-next/xingran-go-backend/pkg/permission"
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
	// 85-03 RBAC-01：读组保留 ANY-of，写操作走 strict 兄弟组（域内无独立 add 码，
	// 创建规则归 ops:manager:rule:edit——与 executions 域同款零新码裁定）。
	rules := r.Group("/manager-rules")
	rules.Use(middleware.RequirePermissions([]string{
		"ops:manager:rule:list",
		"ops:manager:rule:edit",
		"ops:manager:rule:delete",
	}, core))
	{
		rules.POST("/list", ruleHandler.ListRules)
	}
	rulesWrites := r.Group("/manager-rules")
	{
		ruleEdit := rulesWrites.Group("")
		ruleEdit.Use(middleware.RequireStrictPermissions([]string{string(permission.OpsManagerRuleEdit)}, core))
		ruleEdit.POST("", ruleHandler.CreateRule)
		ruleEdit.POST("/:id/update", ruleHandler.UpdateRule)
		ruleDelete := rulesWrites.Group("")
		ruleDelete.Use(middleware.RequireStrictPermissions([]string{string(permission.OpsManagerRuleDelete)}, core))
		ruleDelete.POST("/:id/delete", ruleHandler.DeleteRule)
	}

	// 经理预览（用户列表「经理」列展示用）：仅要求登录态（父组已挂 auth），
	// 不要求经理规则权限——能看用户列表的管理员都可见；数据为内存计算结果，不落库。
	preview := r.Group("/manager-rules")
	{
		preview.POST("/manager-preview", ruleHandler.ManagerPreview)
	}

	// 机构维护
	// 85-03 RBAC-01：读组保留 ANY-of，写操作走 strict 兄弟组（create/generate 归
	// ops:manager:org:edit，域内无独立 add 码）。
	orgs := r.Group("/manager-orgs")
	orgs.Use(middleware.RequirePermissions([]string{
		"ops:manager:org:list",
		"ops:manager:org:edit",
		"ops:manager:org:delete",
	}, core))
	{
		orgs.POST("/list", orgHandler.ListOrgs)
	}
	orgsWrites := r.Group("/manager-orgs")
	{
		orgEdit := orgsWrites.Group("")
		orgEdit.Use(middleware.RequireStrictPermissions([]string{string(permission.OpsManagerOrgEdit)}, core))
		orgEdit.POST("", orgHandler.CreateOrg)
		orgEdit.POST("/:id/update", orgHandler.UpdateOrg)
		// 从部门架构自动生成机构映射（幂等，不覆盖手工修改）=批量创建映射，归编辑语义
		orgEdit.POST("/generate-from-dept", orgHandler.GenerateOrgsFromDept)
		orgDelete := orgsWrites.Group("")
		orgDelete.Use(middleware.RequireStrictPermissions([]string{string(permission.OpsManagerOrgDelete)}, core))
		orgDelete.POST("/:id/delete", orgHandler.DeleteOrg)
	}
}
