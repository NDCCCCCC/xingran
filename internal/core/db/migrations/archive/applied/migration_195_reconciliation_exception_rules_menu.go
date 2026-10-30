//go:build archive_skip


package migrations

import (
	"log"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// Migrate195ReconciliationExceptionRulesMenu 补种"例外规则管理"路由菜单 (menu_type='C')。
//
// 根因 (Phase 44 R3 遗漏):
//   - exception-rules 页面 UI 已存在: pages/asset/reconciliation/exception-rules/index.tsx
//     (含"记录当前为基线"按钮 → reconciliationApi.baseline.snapshot)。
//   - 但 sys_menu 从未 seed 该路由菜单。migration_169 行 301 注释 "例外规则 (R3) R1 仅 seed
//     按钮权限占位, UI 在 R3 引入" + 行 400 "例外规则按钮(虽然菜单未建)" —— R1 只埋了按钮权限
//     占位, R3 加了 UI 却忘了补 menu_type='C' 的菜单 seed。
//   - 前端路由由 sys_menu 动态生成 (routeGenerator.ts: 仅 menuType='C' + 有 component 才生成路由)。
//     菜单缺失 → /asset/reconciliation/exception-rules 无路由 → DynamicRoutes.tsx:207
//     `<Route path="*" element={<Navigate to="/dashboard" replace />} />` 把所有指向它的导航
//     (对账看板无 baseline 引导里的"例外规则管理"链接, 以及 ReconciliationDrawer/ExceptionMatchList/
//     workstations 几处 window.open('/asset/reconciliation/exception-rules/new')) 全部重定向到仪表盘。
//
// 修复:
//   - 仿 migration_169 的 dashboard 菜单 seed, 补种 exception-rules 菜单。
//   - 复用 dashboard 菜单 (已知可用, route /asset/reconciliation/dashboard) 的 parent_id 作为父,
//     path=exception-rules, component=asset/reconciliation/exception-rules/index
//     → routeGenerator.resolvePath 解析为 /asset/reconciliation/exception-rules, 与 dashboard 同前缀。
//
// 权限:
//   - list 端点无 RequirePermissions (reconciliation_exception_router.go:41), 菜单存在即可加载列表。
//   - "记录当前为基线" (baseline/snapshot) 需 asset:reconciliation:exception:create
//     (reconciliation_exception_router.go:65, migration_169:375 已 seed, admin 持)。
//   - 菜单 perms 用 asset:reconciliation:exception:list (模块命名空间一致, 供非 admin 角色授权用)。
//
// 幂等: 按 menu_name='例外规则管理' count-then-insert (migration_165 风格)。
// 不 INSERT sys_role_menu (与 migration_169 行 429 一致: 谁也不给, admin 走超管旁路立即可见,
// 非 admin 角色由管理员手动授权)。
func Migrate195ReconciliationExceptionRulesMenu(db *gorm.DB) error {
	log.Println("Running migration 195: 补种例外规则管理菜单 (Phase 44 R3 遗漏)")

	// 1. 幂等: 已存在则跳过
	var count int64
	if err := db.Table("sys_menu").Where("menu_name = ?", "例外规则管理").Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		log.Println("Migration 195: 菜单 '例外规则管理' 已存在, 跳过")
		return nil
	}

	// 2. 找 dashboard 菜单 (component=asset/reconciliation/dashboard/index, 已知可用) 作为同级参照,
	//    复用其 parent_id → exception-rules 挂在同一父下, route 前缀与 dashboard 一致。
	var dashboardMenu models.Menu
	if err := db.Where("component = ?", "asset/reconciliation/dashboard/index").First(&dashboardMenu).Error; err != nil {
		log.Printf("Migration 195: 找不到 dashboard 菜单 (component=asset/reconciliation/dashboard/index),"+
			"无法确定父菜单, 跳过 (请检查 migration_169 是否已执行): %v", err)
		return nil
	}
	if dashboardMenu.ParentID == nil || *dashboardMenu.ParentID == "" {
		log.Printf("Migration 195: dashboard 菜单无 parent_id, 无法确定例外规则菜单挂载点, 跳过")
		return nil
	}
	parentID := *dashboardMenu.ParentID

	// 3. 创建例外规则管理菜单 (与 dashboard 同父, route 解析为 /asset/reconciliation/exception-rules)
	path := "exception-rules"
	component := "asset/reconciliation/exception-rules/index"
	perms := "asset:reconciliation:exception:list"
	icon := "#"
	menu := &models.Menu{
		MenuName:  "例外规则管理",
		ParentID:  &parentID,
		Path:      &path,
		Component: &component,
		MenuType:  models.MenuTypeMenu, // 'C' 路由菜单
		Visible:   models.VisibleShow,  // 1 显示
		Status:    models.MenuStatusNormal,
		Perms:     &perms,
		Icon:      &icon,
		OrderNum:  30,
		Remark:    "Phase 44 R3: 例外规则 CRUD + 命中测试 + 记录基线按钮 (migration_169 遗漏补种)",
	}
	if err := db.Create(menu).Error; err != nil {
		log.Printf("Migration 195: 创建菜单 '例外规则管理' 失败: %v", err)
		return err
	}

	applogger.Infof("[迁移] sys_menu seed: 例外规则管理 (perms=%s, parent=%s)", perms, parentID)
	return nil
}
