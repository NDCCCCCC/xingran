package migrations

import (
	"log"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// Migrate253DeptManagedOrgPerms 条线部门管理归属映射权限 seed（scope='permission' 域）。
//
// 背景：条线部门（如「潜江团客共享作业服务中心」挂在运营服务本部条线下，ancestors
// 链上无任何中支）需要映射到实际管理归属机构（org_level∈{1,2,3} 的 sys_dept 节点），
// 复用表 sys_dept_location_alias 的 scope='permission' 行（migration_165 已建表）。
//
// 本迁移**不建表**（表已由 migration_165 建），只 seed 3 条 sys_menu 按钮权限：
//   - menu_name: "条线部门归属映射查询" / "条线部门归属映射新增" / "条线部门归属映射删除"
//   - perms:    sys:dept:managed-org:list / :add / :delete
//   - menu_type: 'F'（按钮），visible=0 隐藏，status=0 正常
//   - parent_id: "部门管理" 菜单（menu_type='C'）— 若找不到则 log 警告 + 跳过，不阻断
//   - 幂等: db.Table("sys_menu").Where("perms = ?", perms).Count > 0 时 skip
//
// D-04（锁定决策，与 migration_165 同款）：sys_role_menu 不授权任何角色，
// 新增权限需管理员手动授予；不 INSERT 任何 sys_role_menu 记录。
func Migrate253DeptManagedOrgPerms(db *gorm.DB) error {
	log.Println("Running migration 253: dept managed-org button permissions (3 seeds, no schema change)")

	// 3 条 sys_menu 按钮权限 seed (D-04: 不授权任何角色)
	buttons := []struct {
		name  string
		perms string
	}{
		{"条线部门归属映射查询", "sys:dept:managed-org:list"},
		{"条线部门归属映射新增", "sys:dept:managed-org:add"},
		{"条线部门归属映射删除", "sys:dept:managed-org:delete"},
	}

	// 父菜单: "部门管理" (menu_type='C'),作为按钮权限挂载点
	var parentMenu models.Menu
	parentFound := true
	if err := db.Where("menu_name = ? AND menu_type = ?", "部门管理", "C").First(&parentMenu).Error; err != nil {
		log.Printf("Migration 253: WARNING parent menu '部门管理' (menu_type=C) not found, skip all 3 buttons: %v", err)
		parentFound = false
	}

	for _, btn := range buttons {
		// 幂等: perms 已存在则跳过
		var existingCount int64
		if err := db.Table("sys_menu").Where("perms = ?", btn.perms).Count(&existingCount).Error; err != nil {
			log.Printf("Migration 253: count existing %s failed: %v", btn.perms, err)
			continue
		}
		if existingCount > 0 {
			log.Printf("Migration 253: permission %s already exists, skip", btn.perms)
			continue
		}

		if !parentFound {
			log.Printf("Migration 253: skip seed %s due to missing parent menu", btn.perms)
			continue
		}

		emptyPath := ""
		icon := "#"
		btnPerms := btn.perms

		buttonMenu := &models.Menu{
			MenuName: btn.name,
			ParentID: &parentMenu.ID,
			Path:     &emptyPath,
			MenuType: "F", // 按钮
			Visible:  0,   // 隐藏 — 不在菜单树显示
			Status:   0,   // 正常
			Perms:    &btnPerms,
			Icon:     &icon,
			OrderNum: 30,
			Remark:   "条线部门管理归属映射权限 (D-04: 不自动授权任何角色)",
		}

		if err := db.Create(buttonMenu).Error; err != nil {
			log.Printf("Migration 253: create button menu %s failed: %v", btn.name, err)
			continue
		}
		log.Printf("Migration 253: created button permission %s (ID: %s, perms: %s)", btn.name, buttonMenu.ID, btn.perms)
	}

	// 不 INSERT sys_role_menu (D-04 锁定: "谁也不给,管理员手动授权")

	log.Println("Migration 253 completed: dept managed-org permissions seeded (no role auth)")
	return nil
}
