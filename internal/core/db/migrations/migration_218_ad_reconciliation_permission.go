package migrations

import (
	"errors"
	"fmt"
	"log"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// adReconciliationListPerms 部门↔OU 对账查询按钮权限（D-15），**单一事实源**。
//
// 与 internal/api/v1/system/ad_domain_router.go 的 RequirePermissions 参数、
// 前端 hasPerm 判断字符串三处必须逐字一致——任一处漂移都会让普通角色恒 403
// （菜单 seed perms 与路由命名不一致是本仓库踩过的历史坑）。
const adReconciliationListPerms = "ops:ad:config:reconciliation:list"

// adConfigListPerms 是 AD 域控配置页的菜单权限标识（C 型菜单，menu_name='域控配置'）。
// 本迁移的对账按钮挂在它下面，授权面与该页面的既有按钮（新增/编辑/删除/测试连接/同步数据）同源。
const adConfigListPerms = "ops:ad:config:list"

// Migrate218ADReconciliationPermission Phase 59 / GUARD-02：对账查询按钮权限载体。
//
// 设计（59-03 Decision Reconciliation 第 2/3 条）：
//   - 只新增 menu_type='F' 按钮权限行（先例 migration_163），**不**新增可导航菜单 /
//     菜单路由 / 前端页面 —— 前端是现有 AD 配置页内嵌 Tab（D-17），但后端
//     RequirePermissions 与前端 permissions 数组都来自 sys_menu.perms，
//     没有 F 行则普通角色无法被授予、Tab 恒 403。
//   - **精准授权**：只授予「已关联『AD域控配置』父菜单」的角色（防 antd Tree
//     父子联动视觉陷阱），不用 migration_163 的「所有 status=0 角色」宽授权面。
//   - 幂等：perms 行已存在则跳过创建（复用既有行 ID 补跑授权）；
//     授权 INSERT 用 NOT EXISTS 兜底，可重复执行。
//   - 父菜单缺失（生产库/测试库实测：C 型菜单 menu_name='域控配置'，perms=ops:ad:config:list）
//     → log warning 并返回 nil（非阻断，部署清单登记「管理员手建按钮权限并授权」操作项）。
//     ⚠ 2026-10-20 UAT-59 Test 2 实测修正：原实现按 menu_name='AD域控配置' 查找，
//     而该名字在 seed 与归档 SQL 中均不存在 → 迁移恒降级跳过 → 对账按钮权限
//     永远建不出来（普通角色对账 Tab 恒 403）。改为按 perms 定位。
func Migrate218ADReconciliationPermission(db *gorm.DB) error {
	log.Println("Running migration 218: AD reconciliation button permission (no menu)")

	// 1) 幂等：该 perms 行已存在则复用其 ID，跳过创建（授权仍会执行）
	var existing models.Menu
	permErr := db.Where("perms = ?", adReconciliationListPerms).First(&existing).Error
	hasPermRow := permErr == nil
	if permErr != nil && !errors.Is(permErr, gorm.ErrRecordNotFound) {
		return fmt.Errorf("查询既有对账权限行失败: %w", permErr)
	}

	// 2) 查找父菜单（AD 域控配置页，C 型）
	//
	// ⚠ 按 **perms** 定位，不按 menu_name：
	// 生产/测试库的该菜单实际 menu_name 是「域控配置」（perms=ops:ad:config:list），
	// 规划时假设的「AD域控配置」在 seed 与归档 SQL 里都不存在 → 按名字查必然
	// ErrRecordNotFound → 静默降级跳过 → 对账按钮权限永远建不出来（普通角色恒 403）。
	// perms 才是该页面的稳定标识（菜单改名不影响路由鉴权）。
	var parent models.Menu
	if err := db.Where("perms = ? AND menu_type = ?", adConfigListPerms, "C").First(&parent).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("Warning: 未找到 AD 域控配置父菜单(perms=%s), migration_218 跳过创建与授权 "+
				"(部署清单须管理员手建按钮权限并授权): %v", adConfigListPerms, err)
			return nil
		}
		return fmt.Errorf("查询 AD 域控配置父菜单失败: %w", err)
	}

	// 3) 创建 F 型按钮权限行（幂等：已存在则复用既有行）
	menuID := existing.ID
	if !hasPermRow {
		emptyPath := ""
		icon := "#"
		perms := adReconciliationListPerms
		button := &models.Menu{
			MenuName: "对账查询",
			ParentID: &parent.ID,
			Path:     &emptyPath,
			MenuType: models.MenuTypeButton,
			Visible:  models.VisibleHidden,
			Status:   models.MenuStatusNormal,
			Perms:    &perms,
			Icon:     &icon,
			OrderNum: 30, // 排在 migration_163 的账号池按钮（OrderNum=20）之后
			Remark:   "查看部门-OU 对账不一致条目",
		}
		// 注：Visible/Status 为零值 + 模型带 default tag 时 GORM 会省略列、落 DB 默认值，
		// 与 migration_163 同款行为；按钮（F 型）的可见性不参与前端渲染判定，无语义影响。
		if err := db.Create(button).Error; err != nil {
			return fmt.Errorf("创建对账查询按钮权限行失败: %w", err)
		}
		menuID = button.ID
		log.Printf("Created button permission: 对账查询 (ID: %s)", menuID)
	}

	// 4) 精准授权（PG/sqlite 通用、幂等）：只授予已关联父菜单的角色
	//    （memory: migration-grant-new-menu-precision-helper —— 防父联动视觉陷阱）
	if err := db.Exec(
		`INSERT INTO sys_role_menu (role_id, menu_id)
		 SELECT rm.role_id, ? FROM sys_role_menu rm
		 WHERE rm.menu_id = ?
		   AND NOT EXISTS (
		       SELECT 1 FROM sys_role_menu x
		       WHERE x.role_id = rm.role_id AND x.menu_id = ?)`,
		menuID, parent.ID, menuID).Error; err != nil {
		return fmt.Errorf("精准授权对账查询按钮权限失败: %w", err)
	}

	log.Println("Migration 218 completed: reconciliation button permission ready")
	return nil
}
