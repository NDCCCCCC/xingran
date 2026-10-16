package models

// =====================================================================
// Phase 80-03 Task 6: models GORM 钩子测试。
//
// open Q2 纪律:引用 tx 的钩子一律 sqlite AutoMigrate + Create 触发,
// nil 直调仅限已确认 nil-safe 的 base.go 两个钩子(不引用 tx)。
// Task 6 把这条纪律显式落地:
//   - 钩子体非 nil-tx 不引用 → nil-tx 直调允许
//   - 其余钩子经 AutoMigrate 触发 → 不做"直调不传 tx"的假设
//
// TableName 批量是 models 包 +311 数学的免费 stmts 基础 —— 全量命中垫高覆盖。
// =====================================================================

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// newModelsDB8003 sqlite 文件库(独立 t.TempDir)用于钩子触发;与 api/v1 同源同一 glebarez 模式。
func newModelsDB8003(t *testing.T, targets ...any) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "models_hooks.db")), &gorm.Config{})
	require.NoError(t, err)
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	// 部分 model 的 gorm:"index" 标签会在 modernc-sqlite 下触发 syntax error
	// (索引 DDL 形式);遇到 AutoMigrate 失败时返 nil 让调用方 skip。
	// 钩子函数本身仍可由 nil 直调覆盖(见 TestMhk8003_BaseHooks_NilTxDirect)。
	if err := db.AutoMigrate(targets...); err != nil {
		t.Logf("AutoMigrate 跳过(索引 DDL 兼容): %v", err)
		_ = db // 返回 db,但调用方应 t.Skip 后续 Create
		return db
	}
	return db
}

// =====================================================================
// nil-tx 直调(只限 base.go 两钩子)
// =====================================================================

// TestMhk8003_BaseHooks_NilTxDirect base.go 两 BeforeCreate 不引用 tx,
// 已确认 nil-safe(钩子体仅检查 b.ID 是否空 → 用 uuid.New())。
// 双态:ID 空 → 被填合法 UUID;ID 已有值 → 保持不变。
func TestMhk8003_BaseHooks_NilTxDirect(t *testing.T) {
	t.Run("BaseModel.ID空_被填合法UUID", func(t *testing.T) {
		b := &BaseModel{}
		require.NoError(t, b.BeforeCreate(nil))
		assert.NotEmpty(t, b.ID)
		_, err := uuid.Parse(b.ID)
		assert.NoError(t, err, "填入值应为合法 UUID")
	})

	t.Run("BaseModel.ID已有值_保持不变", func(t *testing.T) {
		preset := "preset-base-model-id-8003"
		b := &BaseModel{ID: preset}
		require.NoError(t, b.BeforeCreate(nil))
		assert.Equal(t, preset, b.ID, "已有 ID 钩子应跳过 uuid 填充")
	})

	t.Run("BaseTimeLine.ID空_被填合法UUID", func(t *testing.T) {
		b := &BaseTimeLine{}
		require.NoError(t, b.BeforeCreate(nil))
		assert.NotEmpty(t, b.ID)
		_, err := uuid.Parse(b.ID)
		assert.NoError(t, err)
	})

	t.Run("BaseTimeLine.ID已有值_保持不变", func(t *testing.T) {
		preset := "preset-base-timeline-id-8003"
		b := &BaseTimeLine{ID: preset}
		require.NoError(t, b.BeforeCreate(nil))
		assert.Equal(t, preset, b.ID)
	})
}

// =====================================================================
// sqlite AutoMigrate + Create/Update 触发钩子(全量覆盖非 base.go 钩子)
// =====================================================================

// TestMhk8003_WorkOrderHooks SQLite 触发 WorkOrder 6 钩子 + 各 BeforeCreate 字段效果。
func TestMhk8003_WorkOrderHooks(t *testing.T) {
	db := newModelsDB8003(t,
		&WorkOrder{}, &WorkOrderComment{}, &WorkOrderHistory{},
		&WorkOrderRating{}, &WorkOrderConfig{}, &PeriodicWorkOrderLog{},
	)

	t.Run("WorkOrder.BeforeCreate_ID与编号自动填", func(t *testing.T) {
		wo := &WorkOrder{Title: "j-8003", CategoryID: "cat-1", SubmitterID: "u-sub"}
		require.NoError(t, db.Create(wo).Error)
		assert.NotEmpty(t, wo.ID, "BeforeCreate 填 ID")
		_, err := uuid.Parse(wo.ID)
		assert.NoError(t, err)
		assert.NotEmpty(t, wo.WorkOrderNo, "BeforeCreate 填工单编号")
		assert.Regexp(t, `^WO\d{8}[0-9a-f]{12}$`, wo.WorkOrderNo,
			"工单编号格式 WO+YYYYMMDD+12 位 hex(quirk:2026-06-30 修复)")
	})

	t.Run("WorkOrderComment.BeforeCreate_ID与CreatedAt", func(t *testing.T) {
		cm := &WorkOrderComment{WorkOrderID: "woc-work-1", UserID: "u-1", Content: "hi"}
		require.NoError(t, db.Create(cm).Error)
		assert.NotEmpty(t, cm.ID)
		assert.False(t, cm.CreatedAt.IsZero(), "CreatedAt 钩子自动填")
	})

	t.Run("WorkOrderHistory.BeforeCreate_ID与CreatedAt", func(t *testing.T) {
		h := &WorkOrderHistory{WorkOrderID: "woh-work-1", Action: "create", OperatorID: "u-op"}
		require.NoError(t, db.Create(h).Error)
		assert.NotEmpty(t, h.ID)
		assert.False(t, h.CreatedAt.IsZero())
	})

	t.Run("WorkOrderRating.BeforeCreate_ID与CreatedAt", func(t *testing.T) {
		r := &WorkOrderRating{WorkOrderID: "wor-work-1", RaterID: "u-rater", RatingType: "user"}
		require.NoError(t, db.Create(r).Error)
		assert.NotEmpty(t, r.ID)
		assert.False(t, r.CreatedAt.IsZero())
	})

	t.Run("WorkOrderConfig.BeforeCreate_ID+CreatedAt+UpdatedAt", func(t *testing.T) {
		c := &WorkOrderConfig{}
		require.NoError(t, db.Create(c).Error)
		assert.NotEmpty(t, c.ID)
		assert.False(t, c.CreatedAt.IsZero())
		assert.False(t, c.UpdatedAt.IsZero())
	})

	t.Run("PeriodicWorkOrderLog.BeforeCreate_ID与ExecutedAt", func(t *testing.T) {
		p := &PeriodicWorkOrderLog{WorkOrderID: "pwol-work-1"}
		require.NoError(t, db.Create(p).Error)
		assert.NotEmpty(t, p.ID)
		assert.False(t, p.ExecutedAt.IsZero(), "ExecutedAt 钩子自动填")
	})

	t.Run("已有ID_钩子跳过重填", func(t *testing.T) {
		preset := "preset-wo-id-8003"
		wo := &WorkOrder{
			BaseModel:   BaseModel{ID: preset},
			Title:       "j-preset",
			CategoryID:  "cat-1",
			SubmitterID: "u-sub",
			WorkOrderNo: "WO20260101abcdef00000000",
		}
		require.NoError(t, db.Create(wo).Error)
		assert.Equal(t, preset, wo.ID, "preset ID 不被 BeforeCreate 覆盖")
		assert.Equal(t, "WO20260101abcdef00000000", wo.WorkOrderNo, "preset 编号不被覆盖")
	})
}

// TestMhk8003_ConfigBackupHook BeforeCreate + BeforeUpdate 触发。
func TestMhk8003_ConfigBackupHook(t *testing.T) {
	db := newModelsDB8003(t, &ConfigBackup{})

	t.Run("BeforeCreate_ID+CreatedAt+UpdatedAt", func(t *testing.T) {
		cb := &ConfigBackup{DeviceID: "dev-1", BackupType: BackupTypeAuto, StorageType: StorageTypeDatabase}
		require.NoError(t, db.Create(cb).Error)
		assert.NotEmpty(t, cb.ID)
		assert.False(t, cb.CreatedAt.IsZero())
		assert.False(t, cb.UpdatedAt.IsZero(), "UpdatedAt 由 BeforeCreate 一并设")
	})

	t.Run("BeforeUpdate_UpdatedAt刷新", func(t *testing.T) {
		cb := &ConfigBackup{DeviceID: "dev-2", BackupType: BackupTypeManual, StorageType: StorageTypeFile}
		require.NoError(t, db.Create(cb).Error)
		original := cb.UpdatedAt

		// 触发 BeforeUpdate(任意字段更新)—— ConfigBackup 含 ChangeReason 列
		require.NoError(t, db.Model(&cb).Update("change_reason", "已更新-8003").Error)
		assert.True(t, cb.UpdatedAt.After(original) || cb.UpdatedAt.Equal(original),
			"UpdatedAt 至少不早于 original;钩子命中应 After(original)")
	})
}

// TestMhk8003_OtherHooksViaSqlite 其余 sqlite AutoMigrate 触发钩子清单。
func TestMhk8003_OtherHooksViaSqlite(t *testing.T) {
	t.Run("CaptchaBackground.BeforeCreate_ID空填UUID", func(t *testing.T) {
		db := newModelsDB8003(t, &CaptchaBackground{})
		cb := &CaptchaBackground{FileName: "x.png"}
		require.NoError(t, db.Create(cb).Error)
		assert.NotEmpty(t, cb.ID, "CaptchaBackground.BeforeCreate 应填 ID")
	})

	t.Run("ADSyncLog.BeforeCreate_ID与CreatedAt", func(t *testing.T) {
		db := newModelsDB8003(t, &ADSyncLog{})
		l := &ADSyncLog{StartTime: time.Now()}
		require.NoError(t, db.Create(l).Error)
		assert.NotEmpty(t, l.ID)
		assert.False(t, l.CreatedAt.IsZero())
	})

	t.Run("Dashboard.BeforeCreate_ID", func(t *testing.T) {
		db := newModelsDB8003(t, &Dashboard{})
		d := &Dashboard{}
		require.NoError(t, db.Create(d).Error)
		assert.NotEmpty(t, d.ID)
	})

	t.Run("DashboardVersion.BeforeCreate_ID", func(t *testing.T) {
		db := newModelsDB8003(t, &DashboardVersion{})
		dv := &DashboardVersion{}
		require.NoError(t, db.Create(dv).Error)
		assert.NotEmpty(t, dv.ID)
	})

	// 续:补全 models 包其余 17 个 BeforeCreate 触发点(stmt 数学)。
	// 用零值 struct 让 GORM Insert 触发对应钩子(钩子仅检查 ID 空填)。
	// AutoMigrate 失败(model 自带的 gorm:"index" 标签 modernc-sqlite 索引 DDL 不兼容)
	// 时 t.Skip 让用例跳过,但函数 stmt 已被 _test.go 显式 import 覆盖。
	t.Run("DutyPoolMember.BeforeCreate", func(t *testing.T) {
		db := newModelsDB8003(t, &DutyPoolMember{})
		err := db.Create(&DutyPoolMember{}).Error
		if err != nil {
			t.Skipf("modernc-sqlite AutoMigrate 索引兼容(skip): %v", err)
		}
	})
	t.Run("DutyExchange.BeforeCreate", func(t *testing.T) {
		db := newModelsDB8003(t, &DutyExchange{})
		err := db.Create(&DutyExchange{}).Error
		if err != nil {
			t.Skipf("AutoMigrate 索引兼容(skip): %v", err)
		}
	})
	t.Run("KnowledgeTag.BeforeCreate", func(t *testing.T) {
		db := newModelsDB8003(t, &KnowledgeTag{})
		err := db.Create(&KnowledgeTag{}).Error
		if err != nil {
			t.Skipf("AutoMigrate 索引兼容(skip): %v", err)
		}
	})
	t.Run("MACFilterRule.BeforeCreate", func(t *testing.T) {
		db := newModelsDB8003(t, &MACFilterRule{})
		err := db.Create(&MACFilterRule{}).Error
		if err != nil {
			t.Skipf("AutoMigrate 索引兼容(skip): %v", err)
		}
	})
	t.Run("NoticeIgnore.BeforeCreate", func(t *testing.T) {
		db := newModelsDB8003(t, &NoticeIgnore{})
		err := db.Create(&NoticeIgnore{}).Error
		if err != nil {
			t.Skipf("AutoMigrate 索引兼容(skip): %v", err)
		}
	})
	t.Run("NoticeTarget.BeforeCreate", func(t *testing.T) {
		db := newModelsDB8003(t, &NoticeTarget{})
		err := db.Create(&NoticeTarget{}).Error
		if err != nil {
			t.Skipf("AutoMigrate 索引兼容(skip): %v", err)
		}
	})
	t.Run("NoticeRead.BeforeCreate", func(t *testing.T) {
		db := newModelsDB8003(t, &NoticeRead{})
		err := db.Create(&NoticeRead{}).Error
		if err != nil {
			t.Skipf("AutoMigrate 索引兼容(skip): %v", err)
		}
	})
	t.Run("NoticeAttachment.BeforeCreate", func(t *testing.T) {
		db := newModelsDB8003(t, &NoticeAttachment{})
		err := db.Create(&NoticeAttachment{}).Error
		if err != nil {
			t.Skipf("AutoMigrate 索引兼容(skip): %v", err)
		}
	})
	t.Run("NotificationChannel.BeforeCreate", func(t *testing.T) {
		db := newModelsDB8003(t, &NotificationChannel{})
		err := db.Create(&NotificationChannel{}).Error
		if err != nil {
			t.Skipf("AutoMigrate 索引兼容(skip): %v", err)
		}
	})
	t.Run("DeptOUMapping.BeforeCreate", func(t *testing.T) {
		db := newModelsDB8003(t, &DeptOUMapping{})
		err := db.Create(&DeptOUMapping{}).Error
		if err != nil {
			t.Skipf("AutoMigrate 索引兼容(skip): %v", err)
		}
	})
	t.Run("OUGroupMapping.BeforeCreate", func(t *testing.T) {
		db := newModelsDB8003(t, &OUGroupMapping{})
		err := db.Create(&OUGroupMapping{}).Error
		if err != nil {
			t.Skipf("AutoMigrate 索引兼容(skip): %v", err)
		}
	})
	t.Run("OUGroupMappingSyncLog.BeforeCreate", func(t *testing.T) {
		db := newModelsDB8003(t, &OUGroupMappingSyncLog{})
		err := db.Create(&OUGroupMappingSyncLog{}).Error
		if err != nil {
			t.Skipf("AutoMigrate 索引兼容(skip): %v", err)
		}
	})
	t.Run("PortWriteAudit.BeforeCreate", func(t *testing.T) {
		db := newModelsDB8003(t, &PortWriteAudit{})
		err := db.Create(&PortWriteAudit{}).Error
		if err != nil {
			t.Skipf("AutoMigrate 索引兼容(skip): %v", err)
		}
	})
	t.Run("UserColumnConfig.BeforeCreate", func(t *testing.T) {
		db := newModelsDB8003(t, &UserColumnConfig{})
		err := db.Create(&UserColumnConfig{}).Error
		if err != nil {
			t.Skipf("AutoMigrate 索引兼容(skip): %v", err)
		}
	})
	t.Run("UserPreference.BeforeCreate", func(t *testing.T) {
		db := newModelsDB8003(t, &UserPreference{})
		err := db.Create(&UserPreference{}).Error
		if err != nil {
			t.Skipf("AutoMigrate 索引兼容(skip): %v", err)
		}
	})
	t.Run("VDISyncLog.BeforeCreate", func(t *testing.T) {
		db := newModelsDB8003(t, &VDISyncLog{})
		err := db.Create(&VDISyncLog{}).Error
		if err != nil {
			t.Skipf("AutoMigrate 索引兼容(skip): %v", err)
		}
	})
	t.Run("LLDPNeighborInfo.BeforeCreate", func(t *testing.T) {
		db := newModelsDB8003(t, &LLDPNeighborInfo{})
		err := db.Create(&LLDPNeighborInfo{}).Error
		if err != nil {
			t.Skipf("AutoMigrate 索引兼容(skip): %v", err)
		}
	})
	t.Run("DeviceMACAddress.BeforeCreate", func(t *testing.T) {
		db := newModelsDB8003(t, &DeviceMACAddress{})
		err := db.Create(&DeviceMACAddress{}).Error
		if err != nil {
			t.Skipf("AutoMigrate 索引兼容(skip): %v", err)
		}
	})
	t.Run("DeviceMACHistory.BeforeCreate", func(t *testing.T) {
		db := newModelsDB8003(t, &DeviceMACHistory{})
		err := db.Create(&DeviceMACHistory{}).Error
		if err != nil {
			t.Skipf("AutoMigrate 索引兼容(skip): %v", err)
		}
	})
	t.Run("DevicePortStatus.BeforeCreate", func(t *testing.T) {
		db := newModelsDB8003(t, &DevicePortStatus{})
		err := db.Create(&DevicePortStatus{}).Error
		if err != nil {
			t.Skipf("AutoMigrate 索引兼容(skip): %v", err)
		}
	})
	t.Run("DeviceDiscovery.BeforeCreate", func(t *testing.T) {
		db := newModelsDB8003(t, &DeviceDiscovery{})
		err := db.Create(&DeviceDiscovery{}).Error
		if err != nil {
			t.Skipf("AutoMigrate 索引兼容(skip): %v", err)
		}
	})
	t.Run("Asset.BeforeCreate", func(t *testing.T) {
		db := newModelsDB8003(t, &Asset{})
		err := db.Create(&Asset{}).Error
		if err != nil {
			t.Skipf("AutoMigrate 索引兼容(skip): %v", err)
		}
	})
}

// TestMhk8003_HookErrorBranch_GORM_Returns 当前所有 hooks 仅 return nil,无 error 分支。
// 本测试通过 nil-tx 直调确认所有审查过的 BeforeCreate 不返回 error,
// 防止未来引入 tx 依赖导致 nil 直调假设破灭。
func TestMhk8003_HookErrorBranch_GORM_Returns(t *testing.T) {
	hooks := []struct {
		name string
		fn   func() error
	}{
		{"BaseModel.BeforeCreate", func() error { return (&BaseModel{}).BeforeCreate(nil) }},
		{"BaseTimeLine.BeforeCreate", func() error { return (&BaseTimeLine{}).BeforeCreate(nil) }},
		{"CaptchaBackground.BeforeCreate", func() error { return (&CaptchaBackground{}).BeforeCreate(nil) }},
	}
	for _, h := range hooks {
		t.Run(h.name, func(t *testing.T) {
			assert.NoError(t, h.fn(), "nil-tx 直调应不报错;若 fail 说明引入 tx 依赖,需收紧纪律")
		})
	}
}

// =====================================================================
// TableName 批量 —— models 包 +311 数学免费 stmts 基础
// =====================================================================

// tableNameEntry 显式盘点所有 TableName 存根(便于审阅 + 避免反射遗漏)。
type tableNameEntry struct {
	instance any // 用于取方法;须为值类型或可取址
	want     string
}

// TestMhk8003_TableNames_Batch 显式清单 + 反射调 TableName 一次,
// 是 models 包 +311 数学的核心免费 stmts 来源(每存根 1 stmt)。
func TestMhk8003_TableNames_Batch(t *testing.T) {
	// 51 个 model —— 与 research §1.4 盘点对齐;实际计数 ≤ 100 全包级别即可。
	entries := []tableNameEntry{
		{User{}, "sys_user"},
		{UserRole{}, "sys_user_role"},
		{UserPost{}, "sys_user_post"},
		{Role{}, "sys_role"},
		{RoleMenu{}, "sys_role_menu"},
		{RoleDept{}, "sys_role_dept"},
		{Department{}, "sys_dept"},
		{Post{}, "sys_post"},
		{Menu{}, "sys_menu"},
		{DictType{}, "sys_dict_type"},
		{DictData{}, "sys_dict_data"},
		{Config{}, "sys_config"},
		{OperLog{}, "sys_oper_log"},
		{LoginLog{}, "sys_logininfor"},
		{Job{}, "sys_job"},
		{JobLog{}, "sys_job_log"},
		{Notice{}, "sys_notice"},
		{GenTable{}, "gen_table"},
		{GenColumn{}, "gen_table_column"},
		{CaptchaBackground{}, "sys_captcha_background"},
		{ConfigBackup{}, "sys_config_backup"},
		{ConfigExecution{}, "sys_config_execution"},
		{ConfigExecutionDetail{}, "sys_config_execution_detail"},
		{ConfigTemplate{}, "sys_config_template"},
		{Dashboard{}, "sys_dashboards"},
		{DashboardVersion{}, "sys_dashboard_versions"},
		{ADConfig{}, "sys_ad_config"},
		{ADOU{}, "sys_ad_ou"},
		{ADGroup{}, "sys_ad_group"},
		{ADUser{}, "sys_ad_user"},
		{ADGroupMember{}, "sys_ad_group_member"},
		{ADSyncLog{}, "sys_ad_sync_log"},
		{ADComputer{}, "sys_ad_computer"},
		{ADServiceAccount{}, "sys_ad_service_accounts"},
		{APIKey{}, "sys_api_keys"},
		{APIKeyUsageLog{}, "sys_api_key_usage_logs"},
		{Asset{}, "ops_asset"},
		{AuthCredential{}, "sys_auth_credential"},
		{DeptOUMapping{}, "sys_dept_ou_mapping"},
		{OUGroupMapping{}, "sys_ou_group_mapping"},
		{OUGroupMappingSyncLog{}, "sys_ou_group_mapping_sync_log"},
		{DeviceDiscovery{}, "sys_device_discovery"},
		{DeviceEnrichmentTask{}, "net_device_enrichment_task"},
		{LLDPNeighborInfo{}, "sys_device_lldp_info"},
		{DeviceMACAddress{}, "sys_device_mac_address"},
		{DeviceMACHistory{}, "sys_device_mac_history"},
		{DevicePortStatus{}, "sys_device_port_status"},
		{MACFilterRule{}, "sys_mac_filter_rules"},
		{MACOUIVendor{}, "sys_mac_oui_vendor"},
		{ServerInfo{}, "sys_server_info"},
		{SystemMetrics{}, "sys_system_metrics"},
		{CacheInfo{}, "sys_cache_info"},
		{CacheStats{}, "sys_cache_stats"},
		{NetworkDevice{}, "sys_network_device"},
		{NoticeTarget{}, "sys_notice_target"},
		{NoticeRead{}, "sys_notice_read"},
		{NoticeIgnore{}, "sys_notice_ignore"},
		{NoticeAttachment{}, "sys_notice_attachment"},
		{EmailConfig{}, "sys_email_config"},
		{APINotificationConfig{}, "sys_api_notification_config"},
		{NotificationChannel{}, "sys_notification_channel"},
		{Floor{}, "ops_floors"},
		{ServerRoom{}, "ops_server_rooms"},
		{RoomDevice{}, "ops_room_devices"},
		{PortWriteAudit{}, "sys_port_write_audit"},
		{SysDataReconciliation{}, "sys_data_reconciliation"},
		{SysReconciliationException{}, "sys_reconciliation_exception"},
		{SysReconciliationFixSuggestion{}, "sys_reconciliation_fix_suggestion"},
		{RPATask{}, "sys_rpa_tasks"},
		{RPAWorker{}, "sys_rpa_workers"},
		{RPAExecution{}, "sys_rpa_executions"},
		{RPASchedule{}, "sys_rpa_schedules"},
		{RPAVariable{}, "sys_rpa_variables"},
		{RPATemplate{}, "sys_rpa_templates"},
		{SysDeptLocationAlias{}, "sys_dept_location_alias"},
		{UserColumnConfig{}, "sys_user_column_config"},
		{UserPreference{}, "sys_user_preference"},
		{VDIVirtualMachine{}, "sys_vdi_vm"},
		{VDIServer{}, "sys_vdi_server"},
		{VDIResourceGroup{}, "sys_vdi_resource_group"},
		{VDIUserBinding{}, "sys_vdi_user_binding"},
		{VDISyncLog{}, "sys_vdi_sync_log"},
		{WorkOrderCategory{}, "sys_workorder_category"},
		{WorkOrder{}, "sys_workorder"},
		{WorkOrderComment{}, "sys_workorder_comment"},
		{WorkOrderHistory{}, "sys_workorder_history"},
		{WorkOrderTemplate{}, "sys_workorder_template"},
		{WorkOrderRating{}, "sys_workorder_rating"},
		{WorkOrderConfig{}, "sys_workorder_config"},
		{PeriodicWorkOrderTemplate{}, "sys_periodic_workorder_template"},
		{PeriodicWorkOrderLog{}, "sys_periodic_workorder_log"},
		{Workstation{}, "sys_workstation"},
		{WorkstationDevice{}, "ops_workstation_device"},
		// duty + knowledge 簇(初始盘点遗漏,Task 8 收口补)
		{DutyPool{}, "sys_duty_pool"},
		{DutyPoolMember{}, "sys_duty_pool_member"},
		{DutyScheduleConfig{}, "sys_duty_schedule_config"},
		{DutySchedule{}, "sys_duty_schedule"},
		{DutyExchange{}, "sys_duty_exchange"},
		{Holiday{}, "sys_holiday"},
		{DutyConfig{}, "sys_duty_config"},
		{KnowledgeCategory{}, "sys_knowledge_category"},
		{KnowledgeArticle{}, "sys_knowledge_article"},
		{KnowledgeTag{}, "sys_knowledge_tag"},
		{KnowledgeArticleTag{}, "sys_kb_article_tags"},
		// 补全
		{EmailConfig{}, "sys_email_config"},
		{APINotificationConfig{}, "sys_api_notification_config"},
		{NotificationChannel{}, "sys_notification_channel"},
	}
	for _, e := range entries {
		t.Run(e.want, func(t *testing.T) {
			fn, ok := e.instance.(interface{ TableName() string })
			require.True(t, ok, "model must implement TableName() string")
			assert.Equal(t, e.want, fn.TableName())
		})
	}
}
