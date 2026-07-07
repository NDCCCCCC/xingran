package migrations

import (
	"fmt"

	"gorm.io/gorm"
)

// GrantNewMenuToRolesHavingParent 把 newMenuID 精准授权给所有已持有父菜单(parentMenuName)的角色。
//
// 解决 antd 父子联动陷阱(memory migration-grant-new-menu-precision-helper):
// 仅 INSERT sys_menu 不 INSERT sys_role_menu 会被父联动带飞视觉但实际 checkedKeys
// 不含 → 路由不生成 → 链接 fallback dashboard。
//
// 幂等:ON CONFLICT DO NOTHING;只波及父已关联角色,admin 走超管旁路自动可见。
//
// 参数:
//   - db: 已连接的 *gorm.DB
//   - parentMenuName: 父菜单 menu_name(如 "端口状态",NOT "端口管理" — Phase 52 D-07)
//   - newMenuID: 新菜单 menu_id(UUID 字符串)
//
// 实现说明:
//   - newMenuID / parentMenuName 均为 migration 内部受控值(非 HTTP 输入),无 SQL 注入面;
//     与 migration_201 line 80-106 的 fmt.Sprintf SQL 风格一致(项目惯例)。
//   - ON CONFLICT DO NOTHING 依赖 sys_role_menu 的 (role_id, menu_id) 复合 PK / 唯一约束
//     (migration_144 已依赖此约束做 dedup)。
//   - SQLite 路径下 helper isPostgreSQL 守卫 return nil(SQLite 不支持 ::uuid cast +
//     ON CONFLICT DO NOTHING);PG functional 行为由 Phase 54 UAT 覆盖。
func GrantNewMenuToRolesHavingParent(db *gorm.DB, parentMenuName string, newMenuID string) error {
	// SQLite 跳过:SQLite 不支持 PG 的 ::uuid cast + ON CONFLICT DO NOTHING;
	// helper 是 PG 启动期 migration 调用点,SQLite 测试路径不消费。
	if !isPostgreSQL(db) {
		return nil
	}

	// D-08 锁定 SQL(52-PATTERNS.md §7 + 52-CONTEXT D-08):
	// INSERT INTO sys_role_menu (role_id, menu_id)
	//   SELECT rm.role_id, '<newMenuID>'::uuid
	//   FROM sys_role_menu rm
	//   JOIN sys_menu m ON rm.menu_id = m.id
	//   WHERE m.menu_name = '<parentMenuName>'
	//   ON CONFLICT DO NOTHING
	sql := fmt.Sprintf(`
INSERT INTO sys_role_menu (role_id, menu_id)
SELECT rm.role_id, '%s'::uuid
FROM sys_role_menu rm
JOIN sys_menu m ON rm.menu_id = m.id
WHERE m.menu_name = '%s'
ON CONFLICT DO NOTHING
`, newMenuID, parentMenuName)

	return db.Exec(sql).Error
}
