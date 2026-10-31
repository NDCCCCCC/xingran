package migrations

import (
	"fmt"
	"log"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// Migrate229PortWriteAuditMenu —— v1.51 Phase 127 PWX-04：端口写审计明细页菜单注册。
//
// 页面 pages/network/port-audits/index.tsx（审计明细: 分页表 + 过滤 + before/after JSON
// 对比展开行）。查询契约 30a80c94: POST /network/ports/write/audit/list,
// 权限沿用 network:port:write（与写同组, 非只读码）。
//
// 幂等: 按 component count-then-insert; 已存在则仅做可见性兜底修正（复用 228 的
// ensureMenuVisible）。父节点按「已知可用子菜单」参照反查（migration_195 模式:
// network/devices/index 的 parent_id = 网络设备 M 根）。
//
// 不 INSERT sys_role_menu（与 169/195/228 一致: admin 超管旁路立即可见,
// 非 admin 角色由管理员手动授权）。失败非阻断（调用方 WARN）。
func Migrate229PortWriteAuditMenu(db *gorm.DB) error {
	log.Println("Running migration 229: PWX-04 端口写审计明细页菜单注册")

	component := "network/port-audits/index"

	var count int64
	if err := db.Model(&models.Menu{}).Where("component = ?", component).Count(&count).Error; err != nil {
		return fmt.Errorf("migration 229: 幂等检查失败: %w", err)
	}
	if count > 0 {
		log.Println("Migration 229: 端口写审计菜单已存在, 仅做可见性兜底")
		return ensureMenuVisible(db, component, "端口写审计")
	}

	// 参照反查父节点: 网络设备 M 根（经 devices 子菜单）
	var deviceMenu models.Menu
	if err := db.Where("component = ?", "network/devices/index").First(&deviceMenu).Error; err != nil {
		log.Printf("Migration 229: 找不到网络设备参照菜单, 端口写审计菜单补种跳过: %v", err)
		return nil
	}
	if deviceMenu.ParentID == nil || *deviceMenu.ParentID == "" {
		log.Println("Migration 229: 网络设备参照菜单无 parent_id, 端口写审计菜单补种跳过")
		return nil
	}
	parentID := *deviceMenu.ParentID

	path := "port-audits"
	perms := "network:port:write"
	icon := "FileSearchOutlined"
	menu := models.Menu{
		MenuName:  "端口写审计",
		ParentID:  &parentID,
		Path:      &path,
		Component: &component,
		MenuType:  models.MenuTypeMenu,
		Visible:   models.VisibleShow,
		Status:    models.MenuStatusNormal,
		Perms:     &perms,
		Icon:      &icon,
		OrderNum:  11,
		Remark:    "v1.51 Phase 127 PWX-04: 端口写审计明细页（migration_229）",
	}
	if err := db.Create(&menu).Error; err != nil {
		return fmt.Errorf("migration 229: 创建失败: %w", err)
	}
	applogger.Infof("[迁移] sys_menu seed: 端口写审计 (perms=%s, parent=%s)", perms, parentID)
	return nil
}
