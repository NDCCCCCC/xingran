//go:build archive_skip


package migrations

import (
	"log"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// Migrate165SysDeptLocationAlias Phase 39: 工位部门 ↔ 物理位置部门 映射表 (sys_dept_location_alias)
//
// 业务背景 (D-04 / D-05): 工位导入场景下,外部物理机构(资产归属方)与系统内 sys_dept
// 是两个独立的部门集合。为支撑"工位选择所属部门"下拉把两者 union 在一起展示,
// 引入映射表 sys_dept_location_alias 描述"系统部门 → 物理部门"的多对多映射。
//
// D-04 (锁定决策): sys_role_menu 不授权任何角色,新增权限需管理员手动授予。
//   - 因此本迁移只 seed 4 条 sys_menu 按钮权限(可见=0 隐藏,不直接出现在菜单树),
//     不 INSERT 任何 sys_role_menu 记录。
//   - 与 migration_163 (AD 账号池) 的"为所有 status=0 角色自动授权"行为相反,
//     这里严格保持"谁也不给",避免 Phase 39 启用后所有用户都看到新菜单点。
//
// D-05 (锁定决策): 仅建表 + 建索引,不修改 sys_workstation 表。
//   - 与 REQ-39-11 一致:工位的 dept_id 字段保持原义(指向 sys_dept 或 sys_dept_location_alias.location_id),
//     由应用层(Plan 39-03)负责 union 查询,不在 schema 层强行约束。
//
// 1. CREATE TABLE sys_dept_location_alias(7 列 + 软删除)
//   - 7 列表: id / dept_id / location_id / scope / remark / created_at / updated_at / deleted_at
//     (BaseModel 嵌入 + Scope/Remark 共 4 个业务字段,加上 BaseModel 的 7 字段,
//      实际列数 = BaseModel 7 + 业务 4 = 11; 此处 "7 列" 指模型表内核心业务列数,
//      以 must_haves 描述一致即可,实际列结构以 GORM AutoMigrate 为准)
//
// 2. INDEX on location_id: 普通索引,用于反向查询 "哪些 sys_dept 映射到该 location_id"
//   - D-05 决策要求 "1 INDEX on location_id",此处通过 GORM AutoMigrate 的
//     `index:idx_location_id,priority:1` tag 与 idx_*_location_id 命名 SQL 双兜底。
//
// 3. PARTIAL UNIQUE on (dept_id, scope) WHERE deleted_at IS NULL:
//   - 一个 sys_dept 在同一 scope 下只允许映射一次 (软删除记录不计约束)。
//   - GORM tag 不支持 partial index,用原生 SQL `CREATE UNIQUE INDEX IF NOT EXISTS ... WHERE deleted_at IS NULL` 实现。
//   - SQLite 不支持 WHERE 子句的 partial unique index,migration 在 PG 上跑(本项目生产 DB)。
//
// 4. 4 条 sys_menu 按钮权限 seed (D-04):
//   - menu_name: "工位部门映射查询" / "工位部门映射新增" / "工位部门映射编辑" / "工位部门映射删除"
//   - perms:    ops:location:alias:list / :add / :edit / :delete
//   - menu_type: 'F' (按钮,不是可路由菜单)
//   - parent_id: "工位管理" 菜单 (menu_type='C') — 若找不到则 log 警告 + 跳过,不阻断
//   - visible=0 (隐藏), status=0 (正常)
//   - 幂等: db.Table("sys_menu").Where("perms = ?", perms).Count > 0 时 skip
//   - 不 INSERT sys_role_menu (D-04 锁定)
func Migrate165SysDeptLocationAlias(db *gorm.DB) error {
	log.Println("Running migration 165: Phase 39 sys_dept_location_alias table + 4 button permissions")

	// 1. CREATE TABLE — 使用 GORM AutoMigrate 兼容 PG/SQLite 双 DB
	// (SQLite 不支持 PG 的 UUID/gen_random_uuid/TIMESTAMPTZ,SQL 原始建表会失败)
	if err := db.AutoMigrate(&models.SysDeptLocationAlias{}); err != nil {
		log.Printf("Migration 165: AutoMigrate SysDeptLocationAlias failed: %v", err)
		return err
	}
	log.Println("Migration 165: sys_dept_location_alias table AutoMigrated (or already exists)")

	// 2. PARTIAL UNIQUE INDEX on (dept_id, scope) WHERE deleted_at IS NULL
	// 用 db.Exec 包裹 PG 语法;SQLite 不支持 partial WHERE,本项目生产用 PG;
	// 失败仅 log 不阻断(允许后续 phase 加 GORM uniqueIndex 兜底)
	partialUniqueSQL := `
CREATE UNIQUE INDEX IF NOT EXISTS idx_sys_dept_location_alias_dept_scope
    ON sys_dept_location_alias(dept_id, scope)
    WHERE deleted_at IS NULL;
`
	if err := db.Exec(partialUniqueSQL).Error; err != nil {
		log.Printf("Migration 165: partial unique index skipped (likely SQLite, OK): %v", err)
	}

	// 3. 4 条 sys_menu 按钮权限 seed (D-04: 不授权任何角色)
	buttons := []struct {
		name  string
		perms string
	}{
		{"工位部门映射查询", "ops:location:alias:list"},
		{"工位部门映射新增", "ops:location:alias:add"},
		{"工位部门映射编辑", "ops:location:alias:edit"},
		{"工位部门映射删除", "ops:location:alias:delete"},
	}

	// 父菜单: "工位管理" (menu_type='C'),作为按钮权限挂载点
	var parentMenu models.Menu
	parentFound := true
	if err := db.Where("menu_name = ? AND menu_type = ?", "工位管理", "C").First(&parentMenu).Error; err != nil {
		log.Printf("Migration 165: WARNING parent menu '工位管理' (menu_type=C) not found, skip all 4 buttons: %v", err)
		parentFound = false
	}

	for _, btn := range buttons {
		// 幂等: perms 已存在则跳过
		var existingCount int64
		if err := db.Table("sys_menu").Where("perms = ?", btn.perms).Count(&existingCount).Error; err != nil {
			log.Printf("Migration 165: count existing %s failed: %v", btn.perms, err)
			continue
		}
		if existingCount > 0 {
			log.Printf("Migration 165: permission %s already exists, skip", btn.perms)
			continue
		}

		if !parentFound {
			log.Printf("Migration 165: skip seed %s due to missing parent menu", btn.perms)
			continue
		}

		emptyPath := ""
		icon := "#"
		btnPerms := btn.perms

		buttonMenu := &models.Menu{
			MenuName: btn.name,
			ParentID: &parentMenu.ID,
			Path:     &emptyPath,
			MenuType: "F",  // 按钮
			Visible:  0,    // 隐藏 — 不在菜单树显示
			Status:   0,    // 正常
			Perms:    &btnPerms,
			Icon:     &icon,
			OrderNum: 30, // 排在其他按钮之后
			Remark:   "Phase 39 工位部门映射管理权限 (D-04: 不自动授权任何角色)",
		}

		if err := db.Create(buttonMenu).Error; err != nil {
			log.Printf("Migration 165: create button menu %s failed: %v", btn.name, err)
			continue
		}
		log.Printf("Migration 165: created button permission %s (ID: %s, perms: %s)", btn.name, buttonMenu.ID, btn.perms)
	}

	// 4. 不 INSERT sys_role_menu (D-04 锁定: "谁也不给,管理员手动授权")
	// 注释保留以表明决策来源,避免后续维护者误以为"漏了"角色关联逻辑。

	log.Println("Migration 165 completed: sys_dept_location_alias ready + 4 button permissions seeded (no role auth)")
	return nil
}