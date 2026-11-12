package migrations

import (
	"fmt"
	"log"

	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// ============================================================================
// Migration 245 — rpa 域权限码 F 型按钮播种（rpa RBAC 接通 · 新码补齐）
//
// 背景（2026-11-11 rpa RBAC 审计）：/rpa 组 52 端点此前仅 JWT+OperLog 零权限。
// 本批端点级 RBAC 接线见 internal/api/v1/rpa/rpa_router.go；权限码常量见
// pkg/permission/config.go RPA 域。task/execution/worker 三组 8 码已由
// menu_catalog_seed.sql 播种（父菜单「RPA 管理」），无需重复播种；
// 本 migration 只补 6 个新码：
//   ops:rpa:worker:edit（Worker 扩缩容/自动扩缩容配置写）
//   rpa:credential:{list,add,edit,remove}（凭证 CRUD + 会话失效）
//   rpa:ai:use（AI 辅助 11 端点统一写码）
//
// 父 C 型菜单「RPA 管理」perms 为 NULL（menu_catalog_seed.sql 行 193），
// 走 parentName + parentComponent 特例定位（对齐 ops:asset 先例）。
// 复用 migration_226 全套 helper（resolveActionPermissionParents /
// ensureActionPermissionMenu / grantActionPermissionToParentHolders），
// 双方言（PG / sqlite 通用）、幂等、非阻断降级。
// ============================================================================

// rpaPermissionFTypeSeedCodes F 型声明块（供 check-permission-codes.mjs
// 的 F_TYPE 解析器识别；必须与 rpaPermissionSeeds 逐条 1:1 一致）。
var rpaPermissionFTypeSeedCodes = []string{
	// 1. ops:rpa:worker:edit（Worker 扩缩容写，父=RPA 管理）
	"ops:rpa:worker:edit",

	// 2. rpa:credential (4 条，命名对齐 network:credential:* 惯例)
	"rpa:credential:list",
	"rpa:credential:add",
	"rpa:credential:edit",
	"rpa:credential:remove",

	// 3. rpa:ai:use（AI 辅助统一写码）
	"rpa:ai:use",
}

// rpaPermissionSeeds 种子表（与上方声明块逐条一致）。
var rpaPermissionSeeds = []actionPermissionSeed{
	{name: "Worker 扩缩容", perms: "ops:rpa:worker:edit", parentName: "RPA 管理", parentComponent: "pages/operations/rpa/index", orderNum: 9},
	{name: "凭证查询", perms: "rpa:credential:list", parentName: "RPA 管理", parentComponent: "pages/operations/rpa/index", orderNum: 10},
	{name: "凭证新增", perms: "rpa:credential:add", parentName: "RPA 管理", parentComponent: "pages/operations/rpa/index", orderNum: 11},
	{name: "凭证编辑", perms: "rpa:credential:edit", parentName: "RPA 管理", parentComponent: "pages/operations/rpa/index", orderNum: 12},
	{name: "凭证删除", perms: "rpa:credential:remove", parentName: "RPA 管理", parentComponent: "pages/operations/rpa/index", orderNum: 13},
	{name: "AI 辅助调用", perms: "rpa:ai:use", parentName: "RPA 管理", parentComponent: "pages/operations/rpa/index", orderNum: 14},
}

// Migrate245RpaPermissionFTypeSeed 播种 rpa 域新码 F 型按钮并精准授权给持父菜单的角色。
// 双方言（PG / sqlite 通用）、幂等、非阻断降级（复用 migration_226 全套 helper）。
func Migrate245RpaPermissionFTypeSeed(db *gorm.DB) error {
	log.Println("Running migration 245: rpa RBAC F type permission seed")

	// 0) 入口硬断言：声明块与种子表必须条数相同且逐条相等
	if len(rpaPermissionFTypeSeedCodes) != len(rpaPermissionSeeds) {
		return fmt.Errorf("migration 245 致命漂移: F_TYPE 块声明数(%d) != 种子表数(%d)",
			len(rpaPermissionFTypeSeedCodes), len(rpaPermissionSeeds))
	}
	for i, code := range rpaPermissionFTypeSeedCodes {
		if code != rpaPermissionSeeds[i].perms {
			return fmt.Errorf("migration 245 致命漂移: 第 %d 项码不匹配声明(%s) != 种子(%s)",
				i, code, rpaPermissionSeeds[i].perms)
		}
	}

	// 逐条处理种子（单条失败仅记录日志并继续，保证非阻断）
	for _, seed := range rpaPermissionSeeds {
		parents, err := resolveActionPermissionParents(db, seed)
		if err != nil {
			log.Printf("Migration 245: WARNING 查询父菜单失败(perms=%s): %v, 跳过该码", seed.perms, err)
			continue
		}
		if len(parents) == 0 {
			log.Printf("Migration 245: WARNING 未找到 live 父菜单(perms=%s), 跳过创建与授权", seed.perms)
			continue
		}

		menuID, err := ensureActionPermissionMenu(db, seed, parents[0].ID)
		if err != nil {
			log.Printf("Migration 245: WARNING 创建/获取按钮菜单失败(perms=%s): %v, 跳过该码", seed.perms, err)
			continue
		}

		parentIDs := make([]string, 0, len(parents))
		for _, p := range parents {
			parentIDs = append(parentIDs, p.ID)
		}
		if err := grantActionPermissionToParentHolders(db, menuID, parentIDs); err != nil {
			log.Printf("Migration 245: WARNING 授权按钮菜单失败(perms=%s, menuID=%s): %v, 跳过授权",
				seed.perms, menuID, err)
			continue
		}
	}

	log.Println("Migration 245 completed: rpa RBAC F type permissions seeded")
	return nil
}

// 编译期引用校验：确保 models 包引用不因重构失效（菜单类型常量由复用 helper 使用）。
var _ = models.MenuTypeButton
