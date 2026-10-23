// 幂等 seed：写入 5 条经理同步规则（经用户确认的分层方案）。
// 规则命中按 priority 升序、第一条命中即返回，特例在前兜底在后。
// 用法: go run ./scripts/diag/seed_manager_rules
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/config"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
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
	postID := func(code string) string {
		var row struct{ ID string }
		if err := g.Table("sys_post").Select("id").Where("post_code = ?", code).Scan(&row).Error; err != nil || row.ID == "" {
			fmt.Printf("找不到岗位 %s（err=%v）\n", code, err)
			os.Exit(1)
		}
		return row.ID
	}
	gmID := postID("GM")          // 总经理
	vpID := postID("VP")          // 分管副总
	branchGmID := postID("BRANCH_GM") // 中心支公司总经理

	strPtr := func(s string) *string { return &s }

	// ---- 5 条规则（用户 2026-10-23 确认稿）----
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
			ManagerType: models.ManagerSourcePostHolder, ManagerPostID: strPtr(vpID),
			ManagerOrgID: nil, // 自身机构（分管副总属分公司本部）
			ScopeType:    models.ManagerScopeAll,
		},
		{
			RuleName: "各机构·部门经理→本机构一把手", Priority: 30, Enabled: true,
			SubjectType: models.ManagerSubjectDeptLeader,
			ManagerType: models.ManagerSourceOrgHead, ManagerPostID: strPtr(branchGmID),
			ManagerOrgID: nil, // 自身机构
			ScopeType:    models.ManagerScopeOrgTree, ScopeID: strPtr(hqOrg.ID),
		},
		{
			RuleName: "各机构·一把手→分公司总经理", Priority: 35, Enabled: true,
			SubjectType: models.ManagerSubjectOrgHead, SubjectPostID: strPtr(gmID),
			ManagerType: models.ManagerSourcePostHolder, ManagerPostID: strPtr(gmID),
			ManagerOrgID: strPtr(hqOrg.ID), // 指定机构：湖北分公司
			ScopeType:    models.ManagerScopeOrgTree, ScopeID: strPtr(hqOrg.ID),
		},
		{
			RuleName: "各机构·员工→本机构一把手", Priority: 40, Enabled: true,
			SubjectType: models.ManagerSubjectEmployee,
			ManagerType: models.ManagerSourceOrgHead, ManagerPostID: strPtr(branchGmID),
			ManagerOrgID: nil, // 自身机构
			ScopeType:    models.ManagerScopeAll,
		},
	}

	for i := range rules {
		r := &rules[i]
		var count int64
		if err := g.Model(&models.ManagerRule{}).Where("rule_name = ?", r.RuleName).Count(&count).Error; err != nil {
			fmt.Println("查重失败:", err)
			os.Exit(1)
		}
		if count > 0 {
			fmt.Printf("跳过（已存在）: %s\n", r.RuleName)
			continue
		}
		if err := g.Create(r).Error; err != nil {
			fmt.Printf("写入失败 %s: %v\n", r.RuleName, err)
			os.Exit(1)
		}
		fmt.Printf("已写入: [%d] %s\n", r.Priority, r.RuleName)
	}
	fmt.Println("完成。注意：规则生效前需先在「岗位管理」给用户分配岗位（sys_user_post 目前为空）。")
}
