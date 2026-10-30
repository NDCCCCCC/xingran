// 幂等 seed：写入 5 条经理同步规则（经用户确认的分层方案）。
// 规则命中按 priority 升序、第一条命中即返回，特例在前兜底在后。
// 用法: go run ./scripts/diag/seed_manager_rules
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/xingran-next/xingran-go-backend/internal/config"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	_ = godotenv.Load()
	cfg, err := config.Load(context.Background())
	if err != nil {
		fmt.Println("加载配置失败:", err)
		os.Exit(1)
	}
	g, err := gorm.Open(postgres.Open(fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.User, cfg.Database.Password, cfg.Database.DBName)), &gorm.Config{})
	if err != nil {
		fmt.Println("连接失败:", err)
		os.Exit(1)
	}

	// ---- 按名称/编码实时查询关键 ID（不硬编码 UUID）----
	var hqOrg models.ManagerOrg // 湖北分公司（根机构）
	if err := g.Where("org_name = ?", "中国太平洋财产保险股份有限公司湖北分公司").First(&hqOrg).Error; err != nil {
		fmt.Println("找不到机构「中国太平洋财产保险股份有限公司湖北分公司」:", err)
		os.Exit(1)
	}
	var hqBranch models.ManagerOrg // 分公司本部
	if err := g.Where("org_name = ?", "分公司本部").First(&hqBranch).Error; err != nil {
		fmt.Println("找不到机构「分公司本部」:", err)
		os.Exit(1)
	}
	strPtr := func(s string) *string { return &s }

	// ---- 5 条规则（用户 2026-10-23 确认稿；B 方案：一把手来源统一用机构负责人 org_leader，不依赖岗位）----
	// 配套数据要求：
	//   规则3/5：各中支/支公司的机构节点部门（如武汉中心支公司）负责人 = 该机构一把手
	//   规则4：湖北分公司根部门负责人 = 分公司总经理
	//   规则2：分公司本部部门负责人 = 分管副总
	rules := []models.ManagerRule{
		{
			RuleName: "分公司本部·员工→部门经理", Priority: 10, Enabled: true,
			SubjectType: models.ManagerSubjectEmployee, SubjectOrgID: strPtr(hqBranch.ID),
			ManagerType: models.ManagerSourceDeptLeader,
			ScopeType:   models.ManagerScopeAll,
		},
		{
			RuleName: "分公司本部·部门经理→分管副总", Priority: 20, Enabled: true,
			SubjectType: models.ManagerSubjectDeptLeader, SubjectOrgID: strPtr(hqBranch.ID),
			ManagerType: models.ManagerSourceOrgLeader, // 分公司本部机构节点部门的负责人 = 分管副总
			ManagerOrgID: nil, // 自身机构（分公司本部）
			ScopeType:    models.ManagerScopeAll,
		},
		{
			RuleName: "各机构·部门经理→本机构一把手", Priority: 30, Enabled: true,
			SubjectType: models.ManagerSubjectDeptLeader,
			ManagerType: models.ManagerSourceOrgLeader, // 本机构节点部门的负责人
			ManagerOrgID: nil, // 自身机构
			ScopeType:    models.ManagerScopeOrgTree, ScopeID: strPtr(hqOrg.ID),
		},
		{
			RuleName: "各机构·一把手→分公司总经理", Priority: 35, Enabled: true,
			SubjectType: models.ManagerSubjectDeptLeader, // 主体=部门负责人（机构节点部门的负责人即一把手，不依赖岗位）
			ManagerType: models.ManagerSourceOrgLeader, // 湖北分公司机构节点部门的负责人 = 分公司总经理
			ManagerOrgID: strPtr(hqOrg.ID), // 指定机构：湖北分公司
			ScopeType:    models.ManagerScopeOrgTree, ScopeID: strPtr(hqOrg.ID),
		},
		{
			RuleName: "各机构·员工→本机构一把手", Priority: 40, Enabled: true,
			SubjectType: models.ManagerSubjectEmployee,
			ManagerType: models.ManagerSourceOrgLeader, // 本机构节点部门的负责人
			ManagerOrgID: nil, // 自身机构
			ScopeType:    models.ManagerScopeAll,
		},
	}

	for i := range rules {
		r := &rules[i]
		var existing models.ManagerRule
		err := g.Where("rule_name = ?", r.RuleName).First(&existing).Error
		if err == nil {
			// 已存在 → 更新（B 方案调整 manager_type，清掉岗位依赖）
			if err := g.Model(&existing).Updates(map[string]interface{}{
				"subject_type":    r.SubjectType,
				"subject_post_id": r.SubjectPostID,
				"manager_type":    r.ManagerType,
				"manager_post_id": r.ManagerPostID,
				"manager_org_id":  r.ManagerOrgID,
				"subject_org_id":  r.SubjectOrgID,
				"scope_type":      r.ScopeType,
				"scope_id":        r.ScopeID,
				"priority":        r.Priority,
				"enabled":         r.Enabled,
			}).Error; err != nil {
				fmt.Printf("更新失败 %s: %v\n", r.RuleName, err)
				os.Exit(1)
			}
			fmt.Printf("已更新: [%d] %s → %s\n", r.Priority, r.RuleName, r.ManagerType)
			continue
		}
		if err := g.Create(r).Error; err != nil {
			fmt.Printf("写入失败 %s: %v\n", r.RuleName, err)
			os.Exit(1)
		}
		fmt.Printf("已写入: [%d] %s\n", r.Priority, r.RuleName)
	}
	fmt.Println("完成。注意：规则2/3/4/5 依赖机构节点部门的负责人字段（sys_dept.leader），请确认已配置。")
}
