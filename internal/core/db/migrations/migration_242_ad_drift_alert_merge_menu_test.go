package migrations

// migration_242 数据侧证据：同步漂移告警并入「AD 域控配置」页的菜单迁移。
//
// 覆盖：
//  1. 旧 C 菜单「同步漂移告警」被隐藏+停用（行保留）；
//  2. 4 个 F 权限码原值不变、parent_id 改挂「域控配置」C 菜单、精准授权给持该菜单的角色；
//  3. 连跑两次幂等（菜单/授权行数不变）；
//  4. 宿主「域控配置」菜单缺失 → 非阻断（仍隐藏旧菜单，不创建孤立 F 行）。
//
// 复用 migration_226 的 sqlite 夹具（sys_menu / sys_role / sys_role_menu）。

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

const test242ConfigMenuID = "cfg242aaaaaaaaaaaaaaaaaaaaaaaaaaaa"

// seed242ConfigMenu 插入「域控配置」宿主菜单 + 持有它的角色。
func seed242ConfigMenu(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.Exec(
		`INSERT INTO sys_menu (id, menu_name, menu_type, status, visible, component, path)
		 VALUES (?, 'AD域控配置', 'C', 0, 1, 'ad-domain/configs/index', 'configs')`,
		test242ConfigMenuID,
	).Error)
	require.NoError(t, db.Exec(
		`INSERT INTO sys_role (id, role_name, status) VALUES (?, ?, 0)`, "r242", "持域控配置角色",
	).Error)
	require.NoError(t, db.Exec(
		`INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)`, "r242", test242ConfigMenuID,
	).Error)
}

func TestMigrate242ADDriftAlertMergeMenu_MovesPermissions(t *testing.T) {
	db := freshSQLiteDBForMigrate226(t)
	seed238SystemRoot(t, db)
	// 先复现线上状态：238 播种独立菜单 + 4 个 F 权限
	require.NoError(t, Migrate238ADDriftAlert(db))
	seed242ConfigMenu(t, db)

	require.NoError(t, Migrate242ADDriftAlertMergeMenu(db))

	// ① 旧 C 菜单隐藏 + 停用（行仍在，旧链接可解析）
	var legacy struct {
		Visible int
		Status  int
	}
	require.NoError(t, db.Raw(
		`SELECT visible, status FROM sys_menu WHERE menu_name = ? AND menu_type = 'C'`, "同步漂移告警",
	).Scan(&legacy).Error)
	assert.Equal(t, 0, legacy.Visible, "旧菜单 visible=0 隐藏")
	assert.Equal(t, 1, legacy.Status, "旧菜单 status=1 停用")

	// ② 4 个 F 权限码原值不变、父菜单改为域控配置
	for _, seed := range adDriftAlertFTypeSeeds {
		var row struct {
			ParentID string
			MenuType string
			Status   int
			Visible  int
		}
		require.NoError(t, db.Raw(
			`SELECT parent_id, menu_type, status, visible FROM sys_menu WHERE perms = ?`, seed.perms,
		).Scan(&row).Error, "权限码 %s 应仍存在", seed.perms)
		assert.Equal(t, test242ConfigMenuID, row.ParentID, "%s 应改挂域控配置菜单", seed.perms)
		assert.Equal(t, "F", row.MenuType, "%s 仍为 F 型", seed.perms)
		assert.Equal(t, 0, row.Status, "%s status=0 启用", seed.perms)
		assert.Equal(t, 0, row.Visible, "%s visible=0 隐藏", seed.perms)

		var granted int64
		require.NoError(t, db.Raw(
			`SELECT COUNT(*) FROM sys_role_menu rm JOIN sys_menu m ON rm.menu_id = m.id
			 WHERE rm.role_id = ? AND m.perms = ?`, "r242", seed.perms,
		).Scan(&granted).Error)
		assert.Equal(t, int64(1), granted, "%s 应授权给持域控配置菜单的角色", seed.perms)
	}
}

func TestMigrate242ADDriftAlertMergeMenu_Idempotent(t *testing.T) {
	db := freshSQLiteDBForMigrate226(t)
	seed238SystemRoot(t, db)
	require.NoError(t, Migrate238ADDriftAlert(db))
	seed242ConfigMenu(t, db)

	require.NoError(t, Migrate242ADDriftAlertMergeMenu(db))
	var menuFirst, grantFirst int64
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM sys_menu`).Scan(&menuFirst).Error)
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM sys_role_menu`).Scan(&grantFirst).Error)

	require.NoError(t, Migrate242ADDriftAlertMergeMenu(db))
	var menuSecond, grantSecond int64
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM sys_menu`).Scan(&menuSecond).Error)
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM sys_role_menu`).Scan(&grantSecond).Error)

	assert.Equal(t, menuFirst, menuSecond, "连跑两次菜单行数不变")
	assert.Equal(t, grantFirst, grantSecond, "连跑两次授权行数不变")
}

func TestMigrate242ADDriftAlertMergeMenu_NoHostMenuNonBlocking(t *testing.T) {
	db := freshSQLiteDBForMigrate226(t)
	seed238SystemRoot(t, db)
	require.NoError(t, Migrate238ADDriftAlert(db))
	// 不建「域控配置」宿主菜单 → 只隐藏旧菜单，不创建孤立 F 行

	require.NoError(t, Migrate242ADDriftAlertMergeMenu(db))

	var legacy struct {
		Visible int
		Status  int
	}
	require.NoError(t, db.Raw(
		`SELECT visible, status FROM sys_menu WHERE menu_name = ? AND menu_type = 'C'`, "同步漂移告警",
	).Scan(&legacy).Error)
	assert.Equal(t, 0, legacy.Visible, "宿主缺失时旧菜单仍应隐藏")
	assert.Equal(t, 1, legacy.Status, "宿主缺失时旧菜单仍应停用")

	// F 行仍是 238 播种的 4 行，且父菜单未被改成空
	for _, seed := range adDriftAlertFTypeSeeds {
		var parent string
		require.NoError(t, db.Raw(
			`SELECT parent_id FROM sys_menu WHERE perms = ?`, seed.perms,
		).Scan(&parent).Error)
		assert.NotEmpty(t, parent, "%s 父菜单不应为空", seed.perms)
	}
}

// TestMigrate242ADDriftAlertMergeMenu_UpgradePathDedupesFPermissions 升级库升级路径：
// 先 238（随机 UUID F 行）→ 后 207 seed（固定 id F 行，ON CONFLICT 仅按 id 不拦 perms →
// 重复）→ 再 242。断言 242 去重后每个 perms 仅一行、挂 seed 的「域控配置」宿主，且幂等。
func TestMigrate242ADDriftAlertMergeMenu_UpgradePathDedupesFPermissions(t *testing.T) {
	db := freshSQLiteDBForMigrate207(t)
	require.NoError(t, db.AutoMigrate(&models.Config{}))

	// ① 模拟升级库：非规范 id 的「系统管理」根（避免触发 207 快速路径整体跳过）
	const legacyRootID = "legacy-root-242a-4242-8242-aaaaaaaaaaaa"
	require.NoError(t, db.Exec(
		`INSERT INTO sys_menu (id, menu_name, menu_type, status) VALUES (?, '系统管理', 'M', 0)`,
		legacyRootID,
	).Error)
	// 238 在升级库上已跑过：创建旧 C 菜单 + 4 个随机 UUID F 行
	require.NoError(t, Migrate238ADDriftAlert(db))

	// ② 207 seed 首次执行（慢速路径）：固定 id F 行插入成功，与 238 随机行 perms 相同 → 重复
	require.NoError(t, Migrate207SeedCanonicalMenuCatalog(db))
	for _, seed := range adDriftAlertFTypeSeeds {
		var dupCount int64
		require.NoError(t, db.Model(&models.Menu{}).
			Where("perms = ? AND menu_type = ?", seed.perms, models.MenuTypeButton).
			Count(&dupCount).Error)
		require.Equal(t, int64(2), dupCount,
			"%s 升级库上应存在 238 随机行与 seed 固定 id 行共两行", seed.perms)
	}

	// ③ 242：先去重、再改挂宿主 —— 最终每个 perms 仅一行且挂 seed 的域控配置菜单
	const seededConfigMenuID = "22ba4218-06e3-45e6-92de-a8f487c42aea"
	require.NoError(t, Migrate242ADDriftAlertMergeMenu(db))
	for _, seed := range adDriftAlertFTypeSeeds {
		var rows []struct {
			ParentID string
		}
		require.NoError(t, db.Raw(
			`SELECT parent_id FROM sys_menu WHERE perms = ? AND menu_type = 'F' AND deleted_at IS NULL`,
			seed.perms,
		).Scan(&rows).Error)
		require.Len(t, rows, 1, "%s 去重后应仅存活一行", seed.perms)
		assert.Equal(t, seededConfigMenuID, rows[0].ParentID,
			"%s 应挂在 seed 的域控配置菜单下", seed.perms)
	}

	// ④ 幂等：再跑一次 242，菜单行数不变（重复行已软删，不会被重复处理）
	var menuFirst int64
	require.NoError(t, db.Model(&models.Menu{}).Count(&menuFirst).Error)
	require.NoError(t, Migrate242ADDriftAlertMergeMenu(db))
	var menuSecond int64
	require.NoError(t, db.Model(&models.Menu{}).Count(&menuSecond).Error)
	assert.Equal(t, menuFirst, menuSecond, "重复执行 242 行数不变")
}

// TestMigrate242ADDriftAlertMergeMenu_FullChainFromSeed 端到端：新库启动链
// 207（规范菜单目录 seed，含 migration_242 的 4 个 F 行）→ 238（创建旧 C 菜单 +
// 复用既有 F 行）→ 242（隐藏旧 C、F 行父菜单收敛到 seed 的「域控配置」）。
// 这是「seed 与运行时迁移共存」的关键证据：F 行不重复、父菜单最终唯一。
func TestMigrate242ADDriftAlertMergeMenu_FullChainFromSeed(t *testing.T) {
	db := freshSQLiteDBForMigrate207(t)
	require.NoError(t, db.AutoMigrate(&models.Config{}))

	require.NoError(t, Migrate207SeedCanonicalMenuCatalog(db))
	// seed 里的「域控配置」C 菜单（ad-domain/configs/index）
	const seededConfigMenuID = "22ba4218-06e3-45e6-92de-a8f487c42aea"

	require.NoError(t, Migrate238ADDriftAlert(db))
	require.NoError(t, Migrate242ADDriftAlertMergeMenu(db))

	// ① 旧 C 菜单唯一且已隐藏停用
	var legacyCount int64
	require.NoError(t, db.Raw(
		`SELECT COUNT(*) FROM sys_menu WHERE menu_name = ? AND menu_type = 'C'`, "同步漂移告警",
	).Scan(&legacyCount).Error)
	assert.Equal(t, int64(1), legacyCount, "旧 C 菜单应唯一（seed 不含它，238 创建一次）")

	var legacy struct {
		Visible int
		Status  int
	}
	require.NoError(t, db.Raw(
		`SELECT visible, status FROM sys_menu WHERE menu_name = ? AND menu_type = 'C'`, "同步漂移告警",
	).Scan(&legacy).Error)
	assert.Equal(t, 0, legacy.Visible)
	assert.Equal(t, 1, legacy.Status)

	// ② 4 个 F 权限码各只有一行，且父菜单 = seed 的域控配置菜单
	for _, seed := range adDriftAlertFTypeSeeds {
		var rows []struct {
			ParentID string
		}
		require.NoError(t, db.Raw(
			`SELECT parent_id FROM sys_menu WHERE perms = ? AND menu_type = 'F'`, seed.perms,
		).Scan(&rows).Error)
		require.Len(t, rows, 1, "%s 应只有一行（seed 与 238 共用既有行）", seed.perms)
		assert.Equal(t, seededConfigMenuID, rows[0].ParentID, "%s 应挂在域控配置菜单下", seed.perms)
	}
}
