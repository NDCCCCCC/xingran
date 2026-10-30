//go:build archive_skip


package migrations

import (
	"log"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// Migrate200FixSuggestionConfigSeeds Phase 46 R5: sys_config + sys_menu + sys_job 种子数据
//
// 业务背景:
//   - D-A3: 置信度门槛可配置
//     sys_config: asset.reconciliation.fix.confidence_threshold (默认 0.9)
//   - D-A4: 触发条件包含 workorder_id IS NULL(由 DetectLayer3 控制,本迁移不动)
//   - D-C2: 回滚窗口 7d
//     sys_config: asset.reconciliation.fix.rollback_window_days (默认 7)
//   - D-C5: 误修复率监控阈值
//     sys_config: asset.reconciliation.fix.mis_fix_threshold (默认 0.01)
//   - 紧急熔断: sys_config: asset.reconciliation.fix.enabled (默认 1)
//
// 步骤清单:
//   A. 4 条 sys_config seed
//   B. 1 个菜单("修复建议" type='C',挂在"资产管理"目录下)
//      + 5 个按钮权限 type='F'(命名空间 asset:reconciliation:fix:*)
//   C. 1 条 sys_job seed(reconciliation:generateFixSuggestions @every 5m)
//
// 不插入 sys_role_menu(沿用 migration_169 "谁也不给"原则 — 管理员手动授权)
func Migrate200FixSuggestionConfigSeeds(db *gorm.DB) error {
	log.Println("Running migration 200: Phase 46 R5 sys_config + sys_menu + sys_job seeds for fix suggestion")

	// ============= A. 4 条 sys_config =============
	if err := seedFixSuggestionConfigs(db); err != nil {
		log.Printf("Migration 200: config seeds failed: %v", err)
		return err
	}

	// ============= B. 1 菜单 + 5 按钮 =============
	if err := seedFixSuggestionMenus(db); err != nil {
		log.Printf("Migration 200: menu seeds failed: %v", err)
		return err
	}

	// ============= C. 1 条 sys_job =============
	if err := seedFixSuggestionSysJobs(db); err != nil {
		log.Printf("Migration 200: sys_job seeds failed: %v", err)
		return err
	}

	log.Println("Migration 200 completed: 4 configs + 1 menu + 5 buttons + 1 sys_job seeded")
	return nil
}

// seedFixSuggestionConfigs seed 4 条 sys_config
func seedFixSuggestionConfigs(db *gorm.DB) error {
	configs := []struct {
		configName  string
		configKey   string
		configValue string
		remark      string
	}{
		{"修复建议-置信度门槛", "asset.reconciliation.fix.confidence_threshold", "0.9", "Phase 46 R5 D-A3: 仅 confidence_score >= 此值才生成修复建议"},
		{"修复建议-误修复率阈值", "asset.reconciliation.fix.mis_fix_threshold", "0.01", "Phase 46 R5 D-C5: misFixRate > 此值触发告警"},
		{"修复建议-回滚窗口(天)", "asset.reconciliation.fix.rollback_window_days", "7", "Phase 46 R5 D-C2: applied_at + 7d 内允许回滚"},
		{"修复建议-功能总开关", "asset.reconciliation.fix.enabled", "1", "Phase 46 R5: 紧急熔断(0=禁用,1=启用)"},
	}

	for _, c := range configs {
		var existingCount int64
		if err := db.Model(&models.Config{}).Where("config_key = ?", c.configKey).Count(&existingCount).Error; err != nil {
			return err
		}
		if existingCount > 0 {
			continue
		}
		cfg := &models.Config{
			ConfigName:  c.configName,
			ConfigKey:   c.configKey,
			ConfigValue: c.configValue,
			ConfigType:  models.ConfigTypeYes, // Y = 系统参数,前端不允许编辑
			IsSystem:    models.ConfigIsSystemYes,
			Remark:      c.remark,
		}
		if err := db.Create(cfg).Error; err != nil {
			log.Printf("Migration 200: create config %s failed: %v", c.configKey, err)
			continue
		}
		applogger.Infof("[迁移] sys_config seed: %s = %s", c.configKey, c.configValue)
	}
	return nil
}

// seedFixSuggestionMenus seed 1 菜单 + 5 按钮
//
// 父菜单 "资产管理" 若不存在则 log 警告 + 跳过挂载(参考 migration_165 模式)
func seedFixSuggestionMenus(db *gorm.DB) error {
	// 1. 父菜单查找
	var parentMenu models.Menu
	parentFound := true
	if err := db.Where("menu_name = ? AND menu_type = ?", "资产管理", "C").First(&parentMenu).Error; err != nil {
		if err := db.Where("menu_name = ? AND menu_type = ?", "资产管理", "M").First(&parentMenu).Error; err != nil {
			log.Printf("Migration 200: WARNING parent menu '资产管理' (menu_type=C or M) not found, will skip children: %v", err)
			parentFound = false
		}
	}

	// 2. 1 个路由菜单
	menus := []struct {
		name      string
		path      string
		component string
		perms     string
		icon      string
		orderNum  int
		remark    string
	}{
		{"修复建议", "fix-suggestion", "asset/reconciliation/fix-suggestion/index", "asset:reconciliation:fix:list", "#", 30, "Phase 46 R5: 5 KPI + 8 列 Table + 3 Modal + 详情 Drawer (D-D1/D-D2)"},
	}

	for _, m := range menus {
		var existingCount int64
		if err := db.Model(&models.Menu{}).Where("perms = ?", m.perms).Count(&existingCount).Error; err != nil {
			return err
		}
		if existingCount > 0 {
			continue
		}
		if !parentFound {
			log.Printf("Migration 200: skip menu %s due to missing parent '资产管理'", m.name)
			continue
		}
		perms := m.perms
		path := m.path
		component := m.component
		icon := m.icon
		menu := &models.Menu{
			MenuName:  m.name,
			ParentID:  &parentMenu.ID,
			Path:      &path,
			Component: &component,
			MenuType:  models.MenuTypeMenu, // 'C' 菜单
			Visible:   models.VisibleShow,  // 1 显示
			Status:    models.MenuStatusNormal,
			Perms:     &perms,
			Icon:      &icon,
			OrderNum:  m.orderNum,
			Remark:    m.remark,
		}
		if err := db.Create(menu).Error; err != nil {
			log.Printf("Migration 200: create menu %s failed: %v", m.name, err)
			continue
		}
		applogger.Infof("[迁移] sys_menu seed: %s (perms=%s)", m.name, m.perms)
	}

	// 3. 5 个按钮权限(menu_type='F', visible=0 隐藏)
	buttons := []struct {
		name     string
		perms    string
		orderNum int
		remark   string
	}{
		{"修复建议-接受", "asset:reconciliation:fix:accept", 100, "Phase 46 R5: pending → accepted"},
		{"修复建议-拒绝", "asset:reconciliation:fix:reject", 101, "Phase 46 R5: pending → rejected(reason≥10 字符)"},
		{"修复建议-应用", "asset:reconciliation:fix:apply", 102, "Phase 46 R5: accepted → applied(写 ops_asset.user_id)"},
		{"修复建议-回滚", "asset:reconciliation:fix:rollback", 103, "Phase 46 R5: applied → rolled_back(7d 窗口内)"},
		{"修复建议-统计", "asset:reconciliation:fix:stats", 104, "Phase 46 R5: 7d KPI 卡片 + 误修复率"},
	}

	for _, btn := range buttons {
		var existingCount int64
		if err := db.Model(&models.Menu{}).Where("perms = ?", btn.perms).Count(&existingCount).Error; err != nil {
			return err
		}
		if existingCount > 0 {
			continue
		}

		// 找到所属的菜单作为父(挂在 "修复建议" 菜单下)
		var btnParent models.Menu
		if err := db.Where("perms = ?", "asset:reconciliation:fix:list").First(&btnParent).Error; err != nil {
			log.Printf("Migration 200: WARNING parent for button %s (perms=asset:reconciliation:fix:list) not found, skip: %v", btn.perms, err)
			continue
		}

		emptyPath := ""
		icon := "#"
		perms := btn.perms
		buttonMenu := &models.Menu{
			MenuName: btn.name,
			ParentID: &btnParent.ID,
			Path:     &emptyPath,
			MenuType: models.MenuTypeButton, // 'F' 按钮
			Visible:  models.VisibleHidden,  // 0 隐藏
			Status:   models.MenuStatusNormal,
			Perms:    &perms,
			Icon:     &icon,
			OrderNum: btn.orderNum,
			Remark:   btn.remark,
		}
		if err := db.Create(buttonMenu).Error; err != nil {
			log.Printf("Migration 200: create button %s failed: %v", btn.name, err)
			continue
		}
		applogger.Infof("[迁移] sys_menu button seed: %s (perms=%s)", btn.name, btn.perms)
	}

	// 不 INSERT sys_role_menu (沿用 D-04 原则:谁也不给,管理员手动授权)
	return nil
}

// seedFixSuggestionSysJobs seed 1 条 sys_job
//
// Deprecated: 此函数自 260704-ne5 后不再启动期调用(database.go 已删除
// migrations.MigrateNNN(d.DB) 启动调用块)。sys_job 的 seed 与 cron 自愈已收口到
// internal/scheduler/reconciliation_tasks.go (reconJobs slice 引用
// reconciliation_crons.go 常量 + legacyCronOverrides 黑名单自愈)。
//
// 本函数仅作历史归档保留,手动调用不会破坏数据(existingCount > 0 跳过),
// 但 cron 值可能滞后于代码常量 —— 新部署不应依赖此函数。
func seedFixSuggestionSysJobs(db *gorm.DB) error {
	jobs := []struct {
		jobName        string
		invokeTarget   string
		cronExpression string
		remark         string
	}{
		{"对账-修复建议生成", "reconciliation:generateFixSuggestions", "@every 5m", "Phase 46 R5: 扫描 Type B 高置信度异常 → 写 sys_reconciliation_fix_suggestion (D-A4 触发器)"},
	}

	for _, j := range jobs {
		var existingCount int64
		if err := db.Model(&models.Job{}).Where("job_name = ?", j.jobName).Count(&existingCount).Error; err != nil {
			return err
		}
		if existingCount > 0 {
			continue
		}
		remark := j.remark
		job := &models.Job{
			JobName:        j.jobName,
			JobGroup:       "reconciliation",
			InvokeTarget:   j.invokeTarget,
			CronExpression: j.cronExpression,
			MisfirePolicy:  1, // 1 = 立即执行(错过周期后立即补跑一次)
			Concurrent:     false,
			Status:         models.JobStatusNormal,
			Remark:         &remark,
		}
		if err := db.Create(job).Error; err != nil {
			log.Printf("Migration 200: create sys_job %s failed: %v", j.jobName, err)
			continue
		}
		applogger.Infof("[迁移] sys_job seed: %s (cron=%s, target=%s)", j.jobName, j.cronExpression, j.invokeTarget)
	}
	return nil
}
