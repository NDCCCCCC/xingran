//go:build archive_skip


package migrations

import (
	"log"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// Migrate169ReconciliationDictsConfigs Phase 42 R1: 资产对账 seed 数据
//
// 一次性 seed 4 类字典 + 8 条 sys_config + 6 条 sys_workorder_category +
// 4 条 sys_job + 6 条 sys_menu 按钮权限,作为 R1 观测底座的前置数据。
//
// 所有 seed 块严格按 count-then-insert 幂等模式(migration_165 风格),
// 重复运行不会报错或产生重复行。
//
// D-10 (cron 走 sys_job 表):本迁移 seed 4 条 sys_job 记录,
// InvokeTarget 字符串对应 R1/R2/R3 任务:
//   - reconciliation:refreshView               (R1 实装)
//   - reconciliation:detectLayer3              (R1 实装)
//   - reconciliation:detectExpiredSilence      (R2 占位)
//   - reconciliation:cleanupExpiredExceptions  (R3 占位)
//
// 不创建独立目录菜单(per CONTEXT.md 锁定:菜单归属 "资产管理",挂在工位/资产管理下)。
// 也不 INSERT sys_role_menu(D-04 类似原则:谁也不给,管理员手动授权)。
func Migrate169ReconciliationDictsConfigs(db *gorm.DB) error {
	log.Println("Running migration 169: Phase 42 R1 dicts + configs + workorder categories + sys_job + menu seeds")

	// ============= A. 4 个 dict_type + data =============
	if err := seedReconciliationDictTypes(db); err != nil {
		log.Printf("Migration 169: dict seeds failed: %v", err)
		return err
	}

	// ============= B. 8 条 sys_config =============
	if err := seedReconciliationConfigs(db); err != nil {
		log.Printf("Migration 169: config seeds failed: %v", err)
		return err
	}

	// ============= C. 6 条 sys_workorder_category =============
	if err := seedReconciliationWorkOrderCategories(db); err != nil {
		log.Printf("Migration 169: workorder category seeds failed: %v", err)
		return err
	}

	// ============= D. 4 条 sys_job (D-10) =============
	if err := seedReconciliationSysJobs(db); err != nil {
		log.Printf("Migration 169: sys_job seeds failed: %v", err)
		return err
	}

	// ============= E. 6 条 sys_menu 按钮权限 + 菜单 =============
	if err := seedReconciliationMenus(db); err != nil {
		log.Printf("Migration 169: menu seeds failed: %v", err)
		return err
	}

	log.Println("Migration 169 completed: 4 dicts + 8 configs + 6 workorder categories + 4 sys_job + 6 menu permissions seeded")
	return nil
}

// seedReconciliationDictTypes seed 4 个 dict_type + 对应 dict_data
func seedReconciliationDictTypes(db *gorm.DB) error {
	type dictDataSpec struct {
		label     string
		value     string
		listClass string // primary/success/warning/error/info/default
		isDefault bool
	}
	type dictSpec struct {
		dictType   string
		dictName   string
		dataValues []dictDataSpec
	}

	dictTypes := []dictSpec{
		{
			dictType: "asset_reconciliation_conflict_type",
			dictName: "资产对账冲突类型",
			dataValues: []dictDataSpec{
				{"A类-完全一致", "A", "success", false},
				{"B类-物理无责", "B", "warning", false},
				{"C类-物理有责无", "C", "error", false},
				{"D类-物理与责任人不一致", "D", "warning", false},
				{"E类-三方不一致", "E", "default", false},
				{"F类-缺数据", "F", "info", false},
			},
		},
		{
			dictType: "asset_reconciliation_severity",
			dictName: "资产对账严重度",
			dataValues: []dictDataSpec{
				{"低", "low", "default", true},
				{"中", "medium", "warning", false},
				{"高", "high", "error", false},
				{"紧急", "critical", "error", false},
			},
		},
		{
			dictType: "asset_reconciliation_exception_action",
			dictName: "资产对账例外动作",
			dataValues: []dictDataSpec{
				{"不告警", "no_alert", "default", true},
				{"不通知", "no_notice", "default", false},
				{"不转工单", "no_workorder", "default", false},
				{"跳过严重度", "skip_severity", "warning", false},
				{"静默期", "silence", "default", false},
			},
		},
		{
			dictType: "asset_reconciliation_status",
			dictName: "资产对账状态",
			dataValues: []dictDataSpec{
				{"未解决", "open", "warning", true},
				{"已解决", "resolved", "success", false},
			},
		},
	}

	for _, dt := range dictTypes {
		// 1. dict_type:count-then-insert
		var typeCount int64
		if err := db.Model(&models.DictType{}).Where("dict_type = ?", dt.dictType).Count(&typeCount).Error; err != nil {
			return err
		}
		if typeCount == 0 {
			dictType := &models.DictType{
				DictName: dt.dictName,
				DictType: dt.dictType,
				Status:   0,
				Remark:   "Phase 42 R1 资产对账 seed",
			}
			if err := db.Create(dictType).Error; err != nil {
				log.Printf("Migration 169: create dict_type %s failed: %v", dt.dictType, err)
				continue
			}
			log.Printf("Migration 169: created dict_type %s", dt.dictType)
		}

		// 2. dict_data:按 (dict_type, dict_value) 幂等
		for _, dv := range dt.dataValues {
			var dataCount int64
			if err := db.Model(&models.DictData{}).
				Where("dict_type = ? AND dict_value = ?", dt.dictType, dv.value).
				Count(&dataCount).Error; err != nil {
				return err
			}
			if dataCount > 0 {
				continue
			}
			listClass := dv.listClass
			dictData := &models.DictData{
				DictSort:  0,
				DictLabel: dv.label,
				DictValue: dv.value,
				DictType:  dt.dictType,
				ListClass: &listClass,
				IsDefault: dv.isDefault,
				Status:    0,
			}
			if err := db.Create(dictData).Error; err != nil {
				log.Printf("Migration 169: create dict_data %s/%s failed: %v", dt.dictType, dv.value, err)
			}
		}
	}
	return nil
}

// seedReconciliationConfigs seed 8 条 sys_config
func seedReconciliationConfigs(db *gorm.DB) error {
	configs := []struct {
		configName  string
		configKey   string
		configValue string
		remark      string
	}{
		{"对账物化视图刷新间隔", "asset.reconciliation.view.refresh_interval", "5m", "R1: 5min CONCURRENTLY 刷新 reconciliation_normalized (D-01)"},
		{"物理证据权重", "asset.reconciliation.score.physical", "0.5", "R1: 物理链路证据置信度权重"},
		{"声明证据权重", "asset.reconciliation.score.declared", "0.3", "R1: 资产声明(责任人)证据权重"},
		{"AD 证据权重", "asset.reconciliation.score.ad", "0.2", "R1: AD 字段证据权重"},
		{"例外规则默认有效期(天)", "asset.reconciliation.exception.default_expiry_days", "30", "R3: 例外规则默认 30 天到期"},
		{"critical 告警阈值", "asset.reconciliation.alert.critical_threshold", "5", "R2: critical 级异常数量阈值(超过触发 SysNotice)"},
		{"已解决静默期(h)", "asset.reconciliation.alert.silence_after_resolved_hours", "168", "R2: 异常已解决后 168h 内不重复告警"},
		{"健康度分项权重", "asset.reconciliation.health.score_weights", `{"normal":1.0,"drift":0.5,"conflict":0.0,"nodata":0.7}`, "R4: 健康度各档权重(JSON)"},
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
			log.Printf("Migration 169: create config %s failed: %v", c.configKey, err)
			continue
		}
		applogger.Infof("[迁移] sys_config seed: %s = %s", c.configKey, c.configValue)
	}
	return nil
}

// seedReconciliationWorkOrderCategories seed 6 条 sys_workorder_category (对账-A/B/C/D/E/F 类)
func seedReconciliationWorkOrderCategories(db *gorm.DB) error {
	categories := []struct {
		name        string
		description string
		sortOrder   int
	}{
		{"对账-A类", "物理有/责任人有且一致 健康无需动作(R1 仅作 dashboard 统计,不进 sys_data_reconciliation)", 100},
		{"对账-B类", "物理无(未采集)/责任人有 — 资产未上线或采集缺失", 101},
		{"对账-C类", "物理有/责任人无 — 资产已采集但未分配责任人", 102},
		{"对账-D类", "物理有/责任人有但不一致 — 责任人变更未生效或工位调岗", 103},
		{"对账-E类", "三方(物理/责任人/AD)互不一致 — 重大异常需人工核查", 104},
		{"对账-F类", "缺数据 — 资产或工位任一端基础数据缺失", 105},
	}

	for _, c := range categories {
		var existingCount int64
		if err := db.Model(&models.WorkOrderCategory{}).Where("category_name = ?", c.name).Count(&existingCount).Error; err != nil {
			return err
		}
		if existingCount > 0 {
			continue
		}
		woc := &models.WorkOrderCategory{
			CategoryName: c.name,
			Description:  c.description,
			Status:       models.WorkOrderCategoryStatusEnabled,
			SortOrder:    c.sortOrder,
		}
		if err := db.Create(woc).Error; err != nil {
			log.Printf("Migration 169: create workorder category %s failed: %v", c.name, err)
			continue
		}
		applogger.Infof("[迁移] sys_workorder_category seed: %s", c.name)
	}
	return nil
}

// seedReconciliationSysJobs seed 4 条 sys_job (D-10)
//
// Deprecated: 此函数自 260704-ne5 后不再启动期调用(database.go 已删除
// migrations.MigrateNNN(d.DB) 启动调用块)。sys_job 的 seed 与 cron 自愈已收口到
// internal/scheduler/reconciliation_tasks.go (reconJobs slice 引用
// reconciliation_crons.go 常量 + legacyCronOverrides 黑名单自愈)。
//
// 本函数仅作历史归档保留,手动调用不会破坏数据(existingCount > 0 跳过),
// 但 cron 值可能滞后于代码常量 —— 新部署不应依赖此函数。
func seedReconciliationSysJobs(db *gorm.DB) error {
	jobs := []struct {
		jobName        string
		invokeTarget   string
		cronExpression string
		remark         string
	}{
		{"对账-物化视图刷新", "reconciliation:refreshView", "@every 5m", "R1: REFRESH MATERIALIZED VIEW CONCURRENTLY reconciliation_normalized (D-01 5min)"},
		{"对账-Layer3检测", "reconciliation:detectLayer3", "@every 6m", "R1: 扫描 reconciliation_normalized → 分类 Type A-F → 写 sys_data_reconciliation"},
		{"对账-静默期重检测", "reconciliation:detectExpiredSilence", "0 2 * * *", "R2 占位: 静默期到期重新检测"},
		{"对账-例外规则清理", "reconciliation:cleanupExpiredExceptions", "0 3 * * *", "R3 占位: 清理过期例外规则"},
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
			MisfirePolicy:  1, // 1 = 立即执行(错过周期后立即补跑一次,符合 R1 cron 错过后行为)
			Concurrent:     false,
			Status:         models.JobStatusNormal,
			Remark:         &remark,
		}
		if err := db.Create(job).Error; err != nil {
			log.Printf("Migration 169: create sys_job %s failed: %v", j.jobName, err)
			continue
		}
		applogger.Infof("[迁移] sys_job seed: %s (cron=%s, target=%s)", j.jobName, j.cronExpression, j.invokeTarget)
	}
	return nil
}

// seedReconciliationMenus seed 6 条 sys_menu(2 路由菜单 + 6 按钮权限)
//
// 菜单策略:
//   - 不创建独立的 "资产对账" 目录菜单(避免与 "资产管理" 重复)
//   - 直接创建 2 个菜单(menu_type='C'): 对账看板 / 异常列表,挂在 "资产管理" 目录下
//   - 6 个按钮权限(visible=0)挂在这两个菜单下
//   - 例外规则 (R3) R1 仅 seed 按钮权限占位,UI 在 R3 引入
//
// 父菜单 "资产管理" 若不存在则 log 警告 + 跳过挂载(参考 migration_165 模式)
func seedReconciliationMenus(db *gorm.DB) error {
	// 1. 父菜单查找
	var parentMenu models.Menu
	parentFound := true
	if err := db.Where("menu_name = ? AND menu_type = ?", "资产管理", "C").First(&parentMenu).Error; err != nil {
		// "资产管理" 可能是 'M'(目录) — 也试一下
		if err := db.Where("menu_name = ? AND menu_type = ?", "资产管理", "M").First(&parentMenu).Error; err != nil {
			log.Printf("Migration 169: WARNING parent menu '资产管理' (menu_type=C or M) not found, will skip children: %v", err)
			parentFound = false
		}
	}

	// 2. 两个路由菜单
	menus := []struct {
		name     string
		path     string
		component string
		perms    string
		icon     string
		orderNum int
		remark   string
	}{
		{"对账看板", "dashboard", "asset/reconciliation/dashboard/index", "asset:reconciliation:dashboard", "#", 10, "Phase 42 R1: 5 KPI + 3 图表(D-04 父路由 302 → dashboard)"},
		{"异常列表", "exceptions", "asset/reconciliation/exceptions/index", "asset:reconciliation:list", "#", 20, "Phase 42 R1: Type A-F 异常只读列表(D-18 R1 无 markResolved)"},
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
			log.Printf("Migration 169: skip menu %s due to missing parent '资产管理'", m.name)
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
			log.Printf("Migration 169: create menu %s failed: %v", m.name, err)
			continue
		}
		applogger.Infof("[迁移] sys_menu seed: %s (perms=%s)", m.name, m.perms)
	}

	// 3. 6 个按钮权限(menu_type='F', visible=0 隐藏)
	//    注意 perms 用单数连字符,遵循 `ops 菜单 seed perms 与路由命名不一致` 教训
	buttons := []struct {
		name     string
		perms    string
		orderNum int
		remark   string
	}{
		{"对账-导出", "asset:reconciliation:export", 100, "Phase 42 R1: 异常列表导出"},
		{"对账-例外创建", "asset:reconciliation:exception:create", 200, "R3: 例外规则创建"},
		{"对账-例外更新", "asset:reconciliation:exception:update", 201, "R3: 例外规则更新"},
		{"对账-例外删除", "asset:reconciliation:exception:delete", 202, "R3: 例外规则删除"},
		{"对账-例外测试", "asset:reconciliation:exception:test", 203, "R3: 例外规则命中测试"},
		{"对账-标记已解决", "asset:reconciliation:markResolved", 300, "R2: 标记异常已解决(D-18 R1 不暴露 UI)"},
	}

	for _, btn := range buttons {
		var existingCount int64
		if err := db.Model(&models.Menu{}).Where("perms = ?", btn.perms).Count(&existingCount).Error; err != nil {
			return err
		}
		if existingCount > 0 {
			continue
		}

		// 找到所属的菜单作为父
		var btnParent models.Menu
		var parentPerms string
		switch {
		case btn.perms == "asset:reconciliation:export":
			parentPerms = "asset:reconciliation:list" // 挂在异常列表下
		case btn.perms == "asset:reconciliation:markResolved":
			parentPerms = "asset:reconciliation:list" // 挂在异常列表下
		default:
			parentPerms = "asset:reconciliation:list" // 例外规则按钮(虽然菜单未建)挂在异常列表下
		}
		if err := db.Where("perms = ?", parentPerms).First(&btnParent).Error; err != nil {
			log.Printf("Migration 169: WARNING parent for button %s (perms=%s) not found, skip: %v", btn.perms, parentPerms, err)
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
			log.Printf("Migration 169: create button %s failed: %v", btn.name, err)
			continue
		}
		applogger.Infof("[迁移] sys_menu button seed: %s (perms=%s)", btn.name, btn.perms)
	}

	// 不 INSERT sys_role_menu (D-04 类似原则:谁也不给,管理员手动授权)
	return nil
}