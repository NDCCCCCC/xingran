package migrations

import (
	"fmt"
	"log"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// Migrate202PortWriteAudit Phase 52 W3: sys_port_write_audit 防御索引 + "端口配置" 菜单 seed + 精准授权。
//
// Path A(52-CONTEXT D-14 + 52-RESEARCH §1.2):sys_port_write_audit 表本身由 GORM
// AutoMigrate 通过 models.PortWriteAudit model 建(database.go Task 3 注册);本 migration
// 不含任何 DDL 建表语句(仅索引 + 菜单 seed)。
//
// 步骤清单:
//  1. SQLite 跳过(isPostgreSQL 守卫,参考 migration_201:45-51):Wave 1 handler 测试
//     用 sqlite 手动 AutoMigrate PortWriteAudit;生产路径仅 PG。
//  2. 防御索引命名漂移:CREATE INDEX IF NOT EXISTS 同名索引(model tag 已声明,
//     此处 noop;但若 GORM tag 命名漂移,此语句补回 — memory xingran-gorm-sql-constraint-naming-conflict)。
//  3. 菜单 seed(count-then-insert + F 型按钮):parent 菜单名 = "端口状态"(D-07 锁定,
//     ROADMAP / REQUIREMENTS 上的别名是笔误,以实际 DB sys_menu.menu_name 为准)。
//  4. D-08 helper 调用:GrantNewMenuToRolesHavingParent(db, "端口状态", menu.ID)。
//
// 字段约束(D-06 + RESEARCH §3.2 A7 + memory xingran-menu-no-java-fields):
//   - **绝对禁止** Java sys_menu 风格的 frame / cache 字段(Go sys_menu 用 Meta JSONB 表达)
//   - F 型按钮:Visible=VisibleHidden(0), Path="", Icon="#"
//
// 错误处理:非阻断,与 Migrate175/176 同风格(失败 log.Printf + return nil,不阻塞启动)。
func Migrate202PortWriteAudit(db *gorm.DB) error {
	log.Println("Running migration 202: port_write_audit defensive indexes + 端口配置 menu seed")

	// 1. SQLite 跳过:菜单 seed / index 均在 PG 分支
	if !isPostgreSQL(db) {
		log.Println("Migration 202: non-PostgreSQL dialect, skip (table created by AutoMigrate)")
		return nil
	}

	// 2. 防御索引(命名漂移兜底,model tag 已声明同名索引)
	//    Path A:GORM AutoMigrate 通过 model tag 建 idx_port_write_audit_device_port_created
	//    (复合 device_id,port_id,created_at) + idx_port_write_audit_created(单列 created_at)。
	//    CREATE INDEX IF NOT EXISTS 是 noop,防 GORM tag 命名漂移。
	defensiveIndexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_port_write_audit_device_port_created ON sys_port_write_audit(device_id, port_id, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_port_write_audit_created ON sys_port_write_audit(created_at)`,
	}
	for _, sql := range defensiveIndexes {
		if err := db.Exec(sql).Error; err != nil {
			// 非阻断:索引可能因表刚建 / 命名冲突暂时无法创建,留待下次启动
			log.Printf("Migration 202: defensive index create failed (non-blocking): %v", err)
		}
	}

	// 3. 菜单 seed(count-then-insert,参考 migration_200:171-207)
	var existingCount int64
	if err := db.Model(&models.Menu{}).Where("perms = ?", "network:port:write").Count(&existingCount).Error; err != nil {
		log.Printf("Migration 202: count existing menu failed (non-blocking): %v", err)
		return nil
	}
	if existingCount > 0 {
		log.Println("Migration 202: 端口配置 menu 已存在,跳过 seed + grant")
		return nil
	}

	// 3a. 父菜单查找(D-07:menu_name='端口状态')
	//     ROADMAP / REQUIREMENTS PERM-03 上的别名是笔误,以实际 DB sys_menu.menu_name
	//     为准(VERIFIED archive/053_fix_menu_paths_unified.sql:185)。
	var parentMenu models.Menu
	if err := db.Where("menu_name = ?", "端口状态").First(&parentMenu).Error; err != nil {
		// 非阻断:父菜单不存在,留待下次启动尝试(数据库可能未 seed 父菜单)
		log.Printf("Migration 202: WARNING 父菜单 '端口状态' 未找到,跳过 seed: %v", err)
		return nil
	}

	// 3b. 构造 F 型按钮菜单(D-06 + RESEARCH §3.2 + migration_200:187-201 analog)
	//     Path="" 空串(F 型 path 不参与 routeGenerator,migration_200 惯例)
	//     **不加** Java sys_menu 的 frame / cache 列(Go sys_menu 用 Meta JSONB 表达)
	emptyPath := ""
	icon := "#"
	perms := "network:port:write"
	menu := &models.Menu{
		MenuName: "端口配置",
		ParentID: &parentMenu.ID,
		Path:     &emptyPath,
		MenuType: models.MenuTypeButton, // 'F'
		Visible:  models.VisibleHidden,  // 0
		Status:   models.MenuStatusNormal,
		Perms:    &perms,
		Icon:     &icon,
		OrderNum: 100,
		Remark:   "Phase 52: 端口写操作按钮权限(5 单端口 + 1 batch)",
	}
	if err := db.Create(menu).Error; err != nil {
		return fmt.Errorf("create 端口配置 menu failed: %w", err)
	}
	log.Printf("Migration 202: sys_menu seed 端口配置 (id=%s, perms=%s, parent=端口状态)", menu.ID, perms)

	// 4. D-08 精准授权:把新菜单授权给所有已持有 "端口状态" 父菜单的角色
	//    解决 antd 父子联动陷阱(memory migration-grant-new-menu-precision-helper)。
	//    非阻断:菜单已 seed,管理员可手动授权;helper 失败不影响应用启动。
	if err := GrantNewMenuToRolesHavingParent(db, "端口状态", menu.ID); err != nil {
		log.Printf("Migration 202: WARNING GrantNewMenuToRolesHavingParent failed (non-blocking): %v", err)
	}

	log.Println("Migration 202 completed: 端口配置 menu seeded + role-menu grant applied")
	return nil
}
