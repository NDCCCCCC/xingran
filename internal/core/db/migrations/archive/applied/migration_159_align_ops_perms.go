//go:build archive_skip


package migrations

import (
	"log"

	"gorm.io/gorm"
)

// Migrate159AlignOpsPerms 对齐 ops 模块菜单 perms 命名 (复数/连字符/remove → 单数/无连字符/delete)。
//
// 背景: database.go 早期 seed 用了「复数+连字符+remove」命名 perms (ops:buildings:*,
// ops:server-rooms:*, ops:dedicated-lines:* 等), 而路由 RequirePermissions 用的是
// 「单数+无连字符+delete」(ops:building:*, ops:serverroom:*, ops:dedicatedline:*),
// 与 system/network/asset 模块一致。两边不一致导致 checkUserPermission 精确匹配失败,
// 持有这些菜单权限的角色读写接口全部 403。
//
// 本迁移把现有 sys_menu.perms 值更新为与路由一致的命名。sys_role_menu 关联 menu_id
// (不改 menu_id), 故角色已勾选的菜单关联不变, perms 值更新后自动匹配路由 → 读写都通。
//
// 幂等: 仅匹配旧命名前缀 (WHERE perms LIKE 'ops:buildings:%' 等), 已是正确命名的
// 不会重复处理; 重复执行 RowsAffected=0, 无副作用。
func Migrate159AlignOpsPerms(db *gorm.DB) error {
	log.Println("Running migration 159: Align ops menu perms (plural/hyphen/remove → singular/delete)")

	// 替换映射: 旧前缀 → 新前缀。先处理带连字符的(长串), 再处理复数。
	// 顺序无关紧要: 每条用 LIKE '<旧前缀>%' 限定, 旧前缀彼此不重叠 (buildings/floors/workstations
	// 是复数; dedicated-lines/server-rooms/info-points/room-devices 含连字符; 互不冲突)。
	renames := []struct{ from, to string }{
		// 连字符模块 (resource 名合并去连字符)
		{"ops:dedicated-lines:", "ops:dedicatedline:"},
		{"ops:server-rooms:", "ops:serverroom:"},
		{"ops:info-points:", "ops:infopoint:"},
		{"ops:room-devices:", "ops:roomdevice:"},
		// 复数模块 (去掉复数 s)
		{"ops:buildings:", "ops:building:"},
		{"ops:floors:", "ops:floor:"},
		{"ops:workstations:", "ops:workstation:"},
	}

	for _, r := range renames {
		// LIKE '<旧前缀>%' 限定只更新旧值行; 不会误伤已正确的 ops:building:%
		// (旧前缀含连字符或复数 s, 新前缀不含; 如 buildings→building, 旧值有 s, 新值无)。
		// ops:building:spaces (楼宇空间, 正确值) 不匹配 'ops:buildings:%' (少一个 s), 安全。
		res := db.Exec(
			"UPDATE sys_menu SET perms = REPLACE(perms, ?, ?) WHERE perms LIKE ?",
			r.from, r.to, r.from+"%",
		)
		if res.Error != nil {
			log.Printf("Migration 159: failed to rename %s → %s: %v", r.from, r.to, res.Error)
			return res.Error
		}
		if res.RowsAffected > 0 {
			log.Printf("Migration 159: %s → %s (%d rows)", r.from, r.to, res.RowsAffected)
		}
	}

	// 动作词 remove → delete (仅 ops 范围; system 等模块不用 remove, 不受影响)
	res := db.Exec(
		"UPDATE sys_menu SET perms = REPLACE(perms, ':remove', ':delete') WHERE perms LIKE 'ops:%:remove'",
	)
	if res.Error != nil {
		log.Printf("Migration 159: failed to rename :remove → :delete: %v", res.Error)
		return res.Error
	}
	if res.RowsAffected > 0 {
		log.Printf("Migration 159: :remove → :delete (%d rows)", res.RowsAffected)
	}

	log.Println("Migration 159 completed: ops menu perms aligned with route requirements")
	return nil
}
