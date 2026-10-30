package migrations

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// canonicalMenuCatalogSQL 内嵌规范菜单目录种子 (239 条 INSERT ... ON CONFLICT DO NOTHING)。
//
//go:embed menu_catalog_seed.sql
var canonicalMenuCatalogSQL string

// 规范菜单目录的两个顶级根 id (生产库事实, quick-260814-ehg 去重保留集):
const (
	canonicalSystemRootID = "d67f4240-f887-481a-b345-94fb36782500" // 系统管理 (M)
	canonicalOpsRootID    = "c50b5b01-fdbc-4821-a3e8-2e2178339b20" // 运维管理 (M)
)

// Migrate207SeedCanonicalMenuCatalog 把规范菜单目录 (239 条) 幂等种子进 sys_menu。
//
// 背景 (debug: admin-role-incomplete-menus):
//   - 全新安装的库中, Go 侧 init_data.go 只种子「运维管理」一族 (36 条);
//     完整菜单目录只存在于 archive SQL (不执行) 与一次性手工导入产物
//     xingran_menus_dedup.sql。历史 Go 迁移 033~201 按设计不再启动期调用
//     (database.go:585-606)。结果: 新库根本没有其它菜单, admin 只能看到
//     运维管理 —— 授权机制 (assignAllMenusToAdmin, 每次启动增量补全全部现存
//     菜单) 本身无缺陷, 是目录缺失。
//   - 本迁移把权威目录固化进启动链路, 之后 assignAllMenusToAdmin 自动授予
//     admin 全部现存菜单 (含未来迁移新增的菜单), 达成"超管始终拥有全部权限"
//     的机制语义。
//
// 双方言: PostgreSQL + SQLite。SQL 无 PG 专有语法 (无 ::uuid cast;
// ON CONFLICT DO NOTHING 无 conflict target, SQLite >= 3.24 支持;
// 时间戳已剥离 +00 时区后缀)。因此不使用 isPostgreSQL 守卫。
//
// 幂等:
//   - 快速路径: 规范「系统管理」根已存在 → 整体跳过 (PG 存量库零影响;
//     用户后续在 UI 删除的菜单不会被复活 —— 尊重用户删除意图)。
//   - 慢速路径: 每行 ON CONFLICT DO NOTHING, 重复执行无重复插入。
//
// 遗留归并 (仅慢速路径触发, 复刻 quick-260814-ehg dev 导入的已验证流程):
//   - init_data.go 历史种子的「运维管理」子树 (随机 UUID, 与规范集 id 不同)
//     会被整支软删 (软删可恢复, 非硬删), 其 sys_role_menu 关联同步清理;
//     规范集含完全等价的 7C+28F 节点 (perms 已核实一致, 见 260814-ehg
//     dedupe-report), 归并后 admin 经 assignAllMenusToAdmin 重新获得规范 id 授权。
//
// 失败策略: 单事务包裹, 任一行失败整体回滚, 不留半成品; 调用方 WARN 不阻断启动,
// 下次启动重试 (幂等, 部分插入的行因 ON CONFLICT 跳过)。
func Migrate207SeedCanonicalMenuCatalog(db *gorm.DB) error {
	// 快速路径: 规范根已存在 → 目录已种子, 跳过
	var count int64
	if err := db.Model(&models.Menu{}).Where("id = ?", canonicalSystemRootID).Count(&count).Error; err != nil {
		return fmt.Errorf("migration 207: 查询规范系统管理根失败: %w", err)
	}
	if count > 0 {
		applogger.Infof("Migration 207: 规范菜单目录已存在 (系统管理根 %s), 跳过", canonicalSystemRootID)
		return nil
	}

	stmts := splitSeedStatements(canonicalMenuCatalogSQL)
	if len(stmts) == 0 {
		return fmt.Errorf("migration 207: 内嵌种子 SQL 为空 (go:embed 异常)")
	}

	return db.Transaction(func(tx *gorm.DB) error {
		// 1. 归并 init_data.go 历史种子的非规范「运维管理」子树 (软删 + 清理授权)
		merged, err := reconcileLegacyOpsTree(tx)
		if err != nil {
			return fmt.Errorf("migration 207: 遗留运维管理子树归并失败: %w", err)
		}
		if merged > 0 {
			applogger.Infof("Migration 207: 软删归并遗留运维管理子树 %d 条 (含授权清理)", merged)
		}

		// 2. 逐行执行规范目录种子 (ON CONFLICT DO NOTHING 保证行级幂等)
		for i, stmt := range stmts {
			if err := tx.Exec(stmt).Error; err != nil {
				return fmt.Errorf("migration 207: 种子第 %d/%d 行失败: %w", i+1, len(stmts), err)
			}
		}

		// 3. 断言: 规范根必须已就位 (239 条全量校验由调用方/测试覆盖)
		var rootCount int64
		if err := tx.Model(&models.Menu{}).Where("id = ?", canonicalSystemRootID).Count(&rootCount).Error; err != nil {
			return fmt.Errorf("migration 207: 种子后校验失败: %w", err)
		}
		if rootCount == 0 {
			return fmt.Errorf("migration 207: 种子执行完毕但规范系统管理根仍缺失")
		}

		applogger.Infof("Migration 207: 规范菜单目录种子完成 (%d 条 INSERT, 幂等)", len(stmts))
		return nil
	})
}

// splitSeedStatements 把种子文件拆成单条 INSERT 语句。
// 文件格式约定: 注释行以 -- 开头; 每条 INSERT 独占一行 (生成器保证)。
func splitSeedStatements(content string) []string {
	var stmts []string
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "--") {
			continue
		}
		if !strings.HasPrefix(line, "INSERT INTO") {
			continue
		}
		stmts = append(stmts, line)
	}
	return stmts
}

// reconcileLegacyOpsTree 软删所有非规范 id 的存活「运维管理」顶级目录及其整支子树,
// 并清理对应的 sys_role_menu 关联。返回软删的菜单条数。
//
// 识别规则: menu_name='运维管理' AND menu_type='M' AND parent_id IS NULL
// AND id != canonicalOpsRootID —— 只有 init_data.go 的 Go 种子会造出这种行
// (规范集的运维管理 id 固定为 c50b5b01)。已软删的历史行 GORM 默认过滤, 不触碰。
func reconcileLegacyOpsTree(tx *gorm.DB) (int, error) {
	var legacyRoots []models.Menu
	if err := tx.Where(
		"menu_name = ? AND menu_type = ? AND parent_id IS NULL AND id <> ?",
		"运维管理", string(models.MenuTypeDir), canonicalOpsRootID,
	).Find(&legacyRoots).Error; err != nil {
		return 0, fmt.Errorf("查询遗留运维管理根失败: %w", err)
	}
	if len(legacyRoots) == 0 {
		return 0, nil
	}

	// 收集整支子树 id (逐层 BFS, Go 种子树深 3 层: 根→页面→按钮; 写成通用循环防未来加深)
	legacyIDs := make([]string, 0, 64)
	frontier := make([]string, 0, len(legacyRoots))
	for _, r := range legacyRoots {
		legacyIDs = append(legacyIDs, r.ID)
		frontier = append(frontier, r.ID)
	}
	for len(frontier) > 0 {
		var children []models.Menu
		if err := tx.Select("id").Where("parent_id IN ?", frontier).Find(&children).Error; err != nil {
			return 0, fmt.Errorf("查询遗留子树后代失败: %w", err)
		}
		frontier = frontier[:0]
		for _, c := range children {
			legacyIDs = append(legacyIDs, c.ID)
			frontier = append(frontier, c.ID)
		}
	}

	// 先清理授权关联, 再软删菜单 (sys_role_menu 无 FK, 顺序仅为语义整洁;
	// 同事务内可回滚, 无中间态外泄)
	if err := tx.Exec("DELETE FROM sys_role_menu WHERE menu_id IN ?", legacyIDs).Error; err != nil {
		return 0, fmt.Errorf("清理遗留子树授权失败: %w", err)
	}
	if err := tx.Where("id IN ?", legacyIDs).Delete(&models.Menu{}).Error; err != nil {
		return 0, fmt.Errorf("软删遗留子树失败: %w", err)
	}

	return len(legacyIDs), nil
}
