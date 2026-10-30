//go:build archive_skip


package migrations

import (
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// Migrate189RemoveMACTrajectoryMenu 删除已废弃的"MAC 轨迹查询"独立菜单及其角色关联。
//
// 背景 (2026-06-30 quick task 20260630-merge-mac-trajectory-to-list-drawer):
//   - MAC 轨迹查询功能已从独立页面 (/network/mac/trajectory) 合并到
//     MAC 地址列表页 (/network/mac) 的右侧 Drawer。
//   - 前端页面、路由、API 调用与后端 /history/trajectory 接口均已移除,
//     但 sys_menu 表里旧的菜单记录是运行时数据(代码层无 seed),
//     必须用迁移显式 DELETE 才能在所有已部署环境清掉侧边栏残留入口,
//     否则用户点击会跳到已删除的组件 → 404/空白。
//
// 操作 (单事务, 幂等, 可重复执行):
//  1. 按 component / path 匹配定位旧菜单 IDs (component 最精确, path 兼容完整/相对两种形态)
//  2. 先删 sys_role_menu 关联 (避免孤儿外键引用)
//  3. 再删 sys_menu 主记录
//  4. 不存在则跳过, 日志输出删除行数
func Migrate189RemoveMACTrajectoryMenu(db *gorm.DB) error {
	const (
		componentExact = "pages/network/mac/trajectory"
		pathFull       = "network/mac/trajectory"
		pathRelative   = "trajectory" // Xingran 子菜单常以相对父级的短路径存储
	)

	var menuIDs []string
	if err := db.Raw(
		`SELECT id FROM sys_menu
		  WHERE component = ?
		     OR path = ?
		     OR path = ?`,
		componentExact, pathFull, pathRelative,
	).Scan(&menuIDs).Error; err != nil {
		applogger.Errorf("[迁移189] 查询 MAC 轨迹菜单失败: %v", err)
		return err
	}

	if len(menuIDs) == 0 {
		applogger.Infof("[迁移189] 未发现 MAC 轨迹菜单残留 (component=%s), 跳过", componentExact)
		return nil
	}

	return db.Transaction(func(tx *gorm.DB) error {
		// 先删角色-菜单关联
		if err := tx.Exec(`DELETE FROM sys_role_menu WHERE menu_id IN (?)`, menuIDs).Error; err != nil {
			applogger.Errorf("[迁移189] 删除 sys_role_menu 关联失败: %v", err)
			return err
		}

		// 再删菜单主记录
		res := tx.Exec(`DELETE FROM sys_menu WHERE id IN (?)`, menuIDs)
		if res.Error != nil {
			applogger.Errorf("[迁移189] 删除 sys_menu 记录失败: %v", res.Error)
			return res.Error
		}

		applogger.Infof("[迁移189] 已删除 MAC 轨迹菜单 %d 条 (含 sys_role_menu 关联)", len(menuIDs))
		return nil
	})
}
