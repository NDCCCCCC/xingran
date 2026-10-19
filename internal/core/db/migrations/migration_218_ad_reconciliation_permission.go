package migrations

import (
	"errors"
	"fmt"
	"log"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"gorm.io/gorm"
)

// adReconciliationListPerms 部门↔OU 对账查询按钮权限（D-15），**单一事实源**。
//
// 与 internal/api/v1/system/ad_domain_router.go 的 RequirePermissions 参数、
// 前端 hasPerm 判断字符串三处必须逐字一致——任一处漂移都会让普通角色恒 403
// （菜单 seed perms 与路由命名不一致是本仓库踩过的历史坑）。
const adReconciliationListPerms = "ops:ad:config:reconciliation:list"

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
//   - 父菜单缺失（FLAGGED ASSUMPTION：生产库应有 menu_name='AD域控配置' 且
//     menu_type='C'，与 migration_163 同款假设）→ log warning 并返回 nil（非阻断，
//     部署清单登记「管理员手建按钮权限并授权」操作项）。
func Migrate218ADReconciliationPermission(db *gorm.DB) error {
	log.Println("Running migration 218: AD reconciliation button permission (no menu)")

	// 1) 幂等：该 perms 行已存在则复用其 ID，跳过创建（授权仍会执行）
	var existing models.Menu
	permErr := db.Where("perms = ?", adReconciliationListPerms).First(&existing).Error
	hasPermRow := permErr == nil
	if permErr != nil && !errors.Is(permErr, gorm.ErrRecordNotFound) {
		return fmt.Errorf("查询既有对账权限行失败: %w", permErr)
	}

	// 2) 查找父菜单（只认 C 型『AD域控配置』）
	var parent models.Menu
	if err := db.Where("menu_name = ? AND menu_type = ?", "AD域控配置", "C").First(&parent).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("Warning: 未找到『AD域控配置』父菜单, migration_218 跳过创建与授权 "+
				"(部署清单须管理员手建按钮权限并授权): %v", err)
			return nil
		}
		return fmt.Errorf("查询『AD域控配置』父菜单失败: %w", err)
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
