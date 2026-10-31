package migrations

import (
	"fmt"
	"log"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// Migrate228ServerSortMenus —— TIDY-01 (v1.51)：服务端排序 4 页面菜单注册/修复。
//
// 背景（docs/管理员使用手册.md §22.3）：
//   值班池 / 排班 / 知识库文章 / 网络模板 4 个页面的服务端排序代码已完成
//   （前端统一 orderByColumn/isAsc 参数），但菜单在 sys_menu 中不可达，
//   路由无法生成（routeGenerator 仅对 menu_type='C' 且有 component 的菜单生成路由）。
//
// 各页面现状与处置：
//   1. 值班池   component=duty/pools/index     —— 菜单存在但 visible=0/status=1
//      且 meta.hidden=true（三重不可达）→ 激活为 visible=1/status=0/hidden=false。
//   2. 排班     component=duty/schedules/index —— 幂等保底修正为可见
//      （catalog seed 中已是 visible=1/status=0，此处为存量库兜底，通常 no-op）。
//   3. 知识库文章 component=knowledge/articles/index —— 存量数据 component 带前导
//      斜杠（'/knowledge/articles/index'，componentLoader 虽可剥离，但与非斜杠形态
//      漂移）→ 统一标准化为无斜杠形态并确保可见；整行缺失则补种（parent=运维管理根）。
//   4. 网络模板 component=network/templates/index —— 页面已存在
//      (pages/network/templates/index.tsx) 但 sys_menu 从未注册 → 补种
//      （parent=网络设备 M 节点，参照 migration_195「已知可用父」定位法）。
//
// 权限：值班池/排班/知识文章沿用既有 perms；网络模板用 network:template:list
// （pkg/permission/config.go:180 NetworkTemplateList，前端 permissionCodes.ts 同源）。
// 不 INSERT sys_role_menu（与 migration_169/195 一致：admin 走超管旁路立即可见，
// 非 admin 角色由管理员手动授权）。
//
// 排序生效说明：前端 4 页均已通过 orderByColumn/isAsc 传参（duty/pools/index.tsx:129、
// network/templates/index.tsx:124-132 等），后端白名单已就绪，菜单可达后排序即生效，
// 无需额外前端开关。
//
// 幂等：count-then-act / first-then-update；重复执行 no-op。
// 失败策略：参照 195，单个页面失败不阻断其余页面，返回聚合错误供调用方 WARN（非阻断）。
func Migrate228ServerSortMenus(db *gorm.DB) error {
	log.Println("Running migration 228: TIDY-01 服务端排序 4 页面菜单注册/修复")

	var errs []error

	// ---------- 1 & 2. 值班池 / 排班：激活既存菜单 ----------
	for _, m := range []struct {
		component string
		name      string
		order     int
	}{
		{component: "duty/pools/index", name: "值班池管理", order: 1},
		{component: "duty/schedules/index", name: "排班管理", order: 2},
	} {
		if err := ensureMenuVisible(db, m.component, m.name); err != nil {
			errs = append(errs, fmt.Errorf("migration 228: %s: %w", m.name, err))
		}
	}

	// ---------- 3. 知识库文章：component 标准化 + 可见，缺失则补种 ----------
	if err := fixKnowledgeArticleMenu(db); err != nil {
		errs = append(errs, fmt.Errorf("migration 228: 知识库文章: %w", err))
	}

	// ---------- 4. 网络模板：补种路由菜单 ----------
	if err := seedNetworkTemplateMenu(db); err != nil {
		errs = append(errs, fmt.Errorf("migration 228: 网络模板: %w", err))
	}

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// ensureMenuVisible 将既存 C 型菜单修正为可见可用（visible=1/status=0/hidden=false）。
func ensureMenuVisible(db *gorm.DB, component, name string) error {
	var menu models.Menu
	err := db.Where("component = ?", component).First(&menu).Error
	if err == gorm.ErrRecordNotFound {
		log.Printf("Migration 228: 菜单 %s (component=%s) 不存在, 跳过", name, component)
		return nil
	}
	if err != nil {
		return fmt.Errorf("查询失败: %w", err)
	}

	needUpdate := menu.Visible != models.VisibleShow || menu.Status != models.MenuStatusNormal ||
		(menu.Meta != nil && menu.Meta.Hidden)
	if !needUpdate {
		log.Printf("Migration 228: 菜单 %s 已可达, 无需修复", name)
		return nil
	}

	updates := map[string]interface{}{
		"visible": models.VisibleShow,
		"status":  models.MenuStatusNormal,
	}
	if menu.Meta != nil {
		meta := *menu.Meta
		meta.Hidden = false
		updates["meta"] = meta
	}
	if err := db.Model(&models.Menu{}).Where("id = ?", menu.ID).Updates(updates).Error; err != nil {
		return fmt.Errorf("更新失败: %w", err)
	}
	applogger.Infof("[迁移] sys_menu 修复: %s → visible=1/status=0/hidden=false", name)
	return nil
}

// fixKnowledgeArticleMenu 标准化知识库文章菜单 component 并确保可见；缺失则补种。
func fixKnowledgeArticleMenu(db *gorm.DB) error {
	const (
		legacyComponent     = "/knowledge/articles/index" // 历史脏数据（前导斜杠）
		normalizedComponent = "knowledge/articles/index"
	)

	var menu models.Menu
	err := db.Where("component IN ?", []string{legacyComponent, normalizedComponent}).
		Order("updated_at DESC").First(&menu).Error
	if err == nil {
		updates := map[string]interface{}{}
		if menu.Component != nil && *menu.Component == legacyComponent {
			updates["component"] = normalizedComponent
		}
		if menu.Visible != models.VisibleShow || menu.Status != models.MenuStatusNormal {
			updates["visible"] = models.VisibleShow
			updates["status"] = models.MenuStatusNormal
		}
		if len(updates) == 0 {
			log.Println("Migration 228: 知识库文章菜单已标准且可达, 无需修复")
			return nil
		}
		if err := db.Model(&models.Menu{}).Where("id = ?", menu.ID).Updates(updates).Error; err != nil {
			return fmt.Errorf("更新失败: %w", err)
		}
		applogger.Infof("[迁移] sys_menu 修复: 知识库文章 component=%s visible=1/status=0", normalizedComponent)
		return nil
	}
	if err != gorm.ErrRecordNotFound {
		return fmt.Errorf("查询失败: %w", err)
	}

	// 整行缺失 → 补种（与 catalog seed 行 202 等价：parent=运维管理根）
	parentID := findOpsRootID(db)
	if parentID == "" {
		log.Println("Migration 228: 找不到运维管理根菜单, 知识库文章补种跳过")
		return nil
	}

	path := "knowledge/articles"
	component := normalizedComponent
	perms := "ops:knowledge:article:list"
	icon := "FileTextOutlined"
	menu = models.Menu{
		MenuName:  "知识库文章",
		ParentID:  &parentID,
		Path:      &path,
		Component: &component,
		MenuType:  models.MenuTypeMenu,
		Visible:   models.VisibleShow,
		Status:    models.MenuStatusNormal,
		Perms:     &perms,
		Icon:      &icon,
		OrderNum:  10,
		Remark:    "TIDY-01: 知识库文章页面菜单补种（服务端排序收尾, migration_228）",
	}
	if err := db.Create(&menu).Error; err != nil {
		return fmt.Errorf("创建失败: %w", err)
	}
	applogger.Infof("[迁移] sys_menu seed: 知识库文章 (component=%s, parent=%s)", component, parentID)
	return nil
}

// seedNetworkTemplateMenu 补种网络模板路由菜单（parent=网络设备 M 节点）。
func seedNetworkTemplateMenu(db *gorm.DB) error {
	component := "network/templates/index"
	var count int64
	if err := db.Model(&models.Menu{}).Where("component = ?", component).Count(&count).Error; err != nil {
		return fmt.Errorf("幂等检查失败: %w", err)
	}
	if count > 0 {
		log.Println("Migration 228: 网络模板菜单已存在, 跳过")
		return ensureMenuVisible(db, component, "网络模板")
	}

	// 参照 migration_195: 按「已知可用子菜单」反查父节点（网络设备 M 根）
	var deviceMenu models.Menu
	if err := db.Where("component = ?", "network/devices/index").First(&deviceMenu).Error; err != nil {
		log.Printf("Migration 228: 找不到网络设备参照菜单, 网络模板补种跳过: %v", err)
		return nil
	}
	if deviceMenu.ParentID == nil || *deviceMenu.ParentID == "" {
		log.Println("Migration 228: 网络设备参照菜单无 parent_id, 网络模板补种跳过")
		return nil
	}
	parentID := *deviceMenu.ParentID

	path := "templates"
	perms := "network:template:list"
	icon := "FileTextOutlined"
	menu := models.Menu{
		MenuName:  "网络模板",
		ParentID:  &parentID,
		Path:      &path,
		Component: &component,
		MenuType:  models.MenuTypeMenu,
		Visible:   models.VisibleShow,
		Status:    models.MenuStatusNormal,
		Perms:     &perms,
		Icon:      &icon,
		OrderNum:  10,
		Remark:    "TIDY-01: 网络模板页面菜单补种（服务端排序收尾, migration_228）",
	}
	if err := db.Create(&menu).Error; err != nil {
		return fmt.Errorf("创建失败: %w", err)
	}
	applogger.Infof("[迁移] sys_menu seed: 网络模板 (perms=%s, parent=%s)", perms, parentID)
	return nil
}

// findOpsRootID 定位运维管理 M 根菜单 id：优先规范 id（migration_207 生产库事实），
// 退化按 menu_name 匹配（早期安装库根 id 不同的场景）。
func findOpsRootID(db *gorm.DB) string {
	var menu models.Menu
	if err := db.Where("id = ? AND menu_type = ?", canonicalOpsRootID, models.MenuTypeDir).
		First(&menu).Error; err == nil {
		return menu.ID
	}
	if err := db.Where("menu_name = ? AND menu_type = ?", "运维管理", models.MenuTypeDir).
		First(&menu).Error; err == nil {
		return menu.ID
	}
	return ""
}
