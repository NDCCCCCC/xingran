//go:build !skip_db_tests
// +build !skip_db_tests

// Phase 78 Plan 06: group_config_service.go 7 函数测试 (Task 4 第一段)
//
// 复用 78-05 helper（setupSync78DB / entry78 / insertConfig78 / closeDB）— D-78-06e 禁止重定义。
// 补建 sys_config 表（照 models.Config gorm tag + internal/core/core_74_08_test.go:37 模式）。
// 零生产 .go 改动。

package addomain

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// setupGC78DB 在 78-05 7 表 fixture 上追加 sys_config 表（BaseModel 列齐）。
//
// 列集照 models.Config gorm tag：
//   id / created_at / updated_at / deleted_at / created_by / updated_by / version /
//   config_name / config_key (UNIQUE for setConfigByKey duplicate 防御) / config_value /
//   config_type / is_system / remark
func setupGC78DB(t *testing.T) *gorm.DB {
	t.Helper()
	db := setupSync78DB(t)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_config (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER,
			config_name TEXT,
			config_key TEXT,
			config_value TEXT,
			config_type TEXT,
			is_system INTEGER,
			remark TEXT
		)
	`).Error)
	return db
}

// insertSysConfig78 写入 sys_config 行（key 重复时 UPDATE 而非 INSERT，
// 避免 sqlite 多行同 key 导致 First 拿到旧值）。
func insertSysConfig78(t *testing.T, db *gorm.DB, key, value string) {
	t.Helper()
	now := time.Now()
	// 先查是否存在
	var n int64
	require.NoError(t, db.Model(&models.Config{}).Where("config_key = ?", key).Count(&n).Error)
	if n > 0 {
		require.NoError(t, db.Exec(`UPDATE sys_config SET config_value = ?, updated_at = ? WHERE config_key = ?`,
			value, now, key).Error)
		return
	}
	id := uuid.NewString()
	require.NoError(t, db.Exec(`
		INSERT INTO sys_config
		(id, config_key, config_value, config_type, is_system, created_at, updated_at)
		VALUES (?, ?, ?, 'string', 0, ?, ?)
	`, id, key, value, now, now).Error)
}

// =============================================================================
// Task 4 第一段：GroupConfigService 7 函数
// =============================================================================

// TestGC78_NewGroupConfigService 构造器。
func TestGC78_NewGroupConfigService(t *testing.T) {
	db := setupGC78DB(t)
	defer closeDB(t, db)
	svc := NewGroupConfigService(db)
	assert.NotNil(t, svc)
}

// TestGC78_GetSetConfigByKey setConfigByKey 新键/已存在键；getConfigByKey 命中/缺失。
func TestGC78_GetSetConfigByKey(t *testing.T) {
	db := setupGC78DB(t)
	defer closeDB(t, db)
	svc := NewGroupConfigService(db)
	ctx := context.Background()

	// 缺失 → setConfigByKey 创建新行
	require.NoError(t, svc.setConfigByKey(ctx, "test.key.A", "alpha"))

	// getConfigByKey 命中
	got, err := svc.getConfigByKey(ctx, "test.key.A")
	require.NoError(t, err)
	assert.Equal(t, "alpha", got)

	// 已存在 → setConfigByKey 更新非重复插入
	require.NoError(t, svc.setConfigByKey(ctx, "test.key.A", "beta"))
	got, err = svc.getConfigByKey(ctx, "test.key.A")
	require.NoError(t, err)
	assert.Equal(t, "beta", got, "已存在键应被 update 而非 insert")

	// 验证行数（不会变成 2 行）
	var count int64
	require.NoError(t, db.Model(&models.Config{}).Where("config_key = ?", "test.key.A").Count(&count).Error)
	assert.EqualValues(t, 1, count)

	// 缺失 getConfigByKey → 错误（GORM ErrRecordNotFound）
	_, err = svc.getConfigByKey(ctx, "test.key.MISSING")
	require.Error(t, err, "getConfigByKey 不存在应返回 ErrRecordNotFound 包装")
}

// TestGC78_GetGroupSyncConfig_DefaultsAndOverrides 空表 → 默认值；预置各键 → 覆盖；非法值现行为。
//
// 现行为锁（D-78-06g）：group_config_service.go 的 GetGroupSyncConfig 不做严格校验：
//   - 布尔位填 "abc" → `enabledStr == "true" || enabledStr == "1"` → false（兜底 false，不 panic）
//   - 数字位填 "not-a-num" → Atoi 失败 → 保留 default（兜底 default）
//   - 数字位填 "-5" → Atoi 成功 → MaxConcurrent=-5 接受（无下界校验；该校验在 ValidateConfig 内）
//   - 数字位填 "21" → Atoi 成功 → MaxConcurrent=21 接受（无上界校验；同上）
func TestGC78_GetGroupSyncConfig_DefaultsAndOverrides(t *testing.T) {
	db := setupGC78DB(t)
	defer closeDB(t, db)
	svc := NewGroupConfigService(db)
	ctx := context.Background()

	// (a) 空表 → 全默认值
	cfg, err := svc.GetGroupSyncConfig(ctx)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.False(t, cfg.Enabled, "默认 Enabled=false")
	assert.Equal(t, "0 */15 * * * *", cfg.Cron)
	assert.Equal(t, "", cfg.MemberOU)
	assert.True(t, cfg.AutoCreateGroups, "默认 AutoCreateGroups=true")
	assert.Equal(t, 5, cfg.MaxConcurrent)
	assert.Equal(t, 100, cfg.SyncBatchSize)

	// (b) 覆盖所有键
	insertSysConfig78(t, db, ConfigGroupSyncEnabled, "true")
	insertSysConfig78(t, db, ConfigGroupSyncCron, "0 0 * * * *")
	insertSysConfig78(t, db, ConfigGroupMemberOU, "OU=Members,DC=example,DC=com")
	insertSysConfig78(t, db, ConfigGroupAutoCreate, "true")
	insertSysConfig78(t, db, ConfigGroupMaxConcurrent, "10")
	insertSysConfig78(t, db, ConfigGroupSyncBatchSize, "200")
	cfg, err = svc.GetGroupSyncConfig(ctx)
	require.NoError(t, err)
	assert.True(t, cfg.Enabled)
	assert.Equal(t, "0 0 * * * *", cfg.Cron)
	assert.Equal(t, "OU=Members,DC=example,DC=com", cfg.MemberOU)
	assert.True(t, cfg.AutoCreateGroups)
	assert.Equal(t, 10, cfg.MaxConcurrent)
	assert.Equal(t, 200, cfg.SyncBatchSize)

	// (c) 非法值兜底
	insertSysConfig78(t, db, ConfigGroupMaxConcurrent, "not-a-number") // Atoi 失败 → 保留 default
	insertSysConfig78(t, db, ConfigGroupSyncBatchSize, "abc")         // 同上
	cfg, err = svc.GetGroupSyncConfig(ctx)
	require.NoError(t, err)
	assert.Equal(t, 5, cfg.MaxConcurrent, "Atoi 失败 → 保留 default (5) 而非上次成功值")
	assert.Equal(t, 100, cfg.SyncBatchSize, "Atoi 失败 → 保留 default (100)")

	// (d) 布尔位 "abc" → false（GetGroupSyncConfig 第 63 行逻辑）
	insertSysConfig78(t, db, ConfigGroupSyncEnabled, "abc")
	cfg, err = svc.GetGroupSyncConfig(ctx)
	require.NoError(t, err)
	assert.False(t, cfg.Enabled, "abc 不等于 true/1 → false")
}

// TestGC78_IsGroupSyncEnabled 三态：开 / 关 / 缺省。
func TestGC78_IsGroupSyncEnabled(t *testing.T) {
	db := setupGC78DB(t)
	defer closeDB(t, db)
	svc := NewGroupConfigService(db)
	ctx := context.Background()

	// 缺省 → false（默认 Enabled=false）
	assert.False(t, svc.IsGroupSyncEnabled(ctx))

	// true
	insertSysConfig78(t, db, ConfigGroupSyncEnabled, "true")
	assert.True(t, svc.IsGroupSyncEnabled(ctx))

	// false（覆盖）
	insertSysConfig78(t, db, ConfigGroupSyncEnabled, "false")
	assert.False(t, svc.IsGroupSyncEnabled(ctx))
}

// TestGC78_UpdateGroupSyncConfig_And_Validate 合法配置 → 落库并回读；
// ValidateConfig 3 条非法输入；UpdateGroupSyncConfig 非法配置被拒绝。
func TestGC78_UpdateGroupSyncConfig_And_Validate(t *testing.T) {
	db := setupGC78DB(t)
	defer closeDB(t, db)
	svc := NewGroupConfigService(db)
	ctx := context.Background()

	// 合法配置
	valid := &GroupSyncConfig{
		Enabled:          true,
		Cron:             "0 0 * * * *",
		MemberOU:         "OU=Members,DC=example,DC=com",
		AutoCreateGroups: true,
		MaxConcurrent:    10,
		SyncBatchSize:    100,
	}
	require.NoError(t, svc.ValidateConfig(valid))

	require.NoError(t, svc.UpdateGroupSyncConfig(ctx, valid))
	got, err := svc.GetGroupSyncConfig(ctx)
	require.NoError(t, err)
	assert.Equal(t, valid.Cron, got.Cron)
	assert.Equal(t, valid.MemberOU, got.MemberOU)
	assert.Equal(t, valid.MaxConcurrent, got.MaxConcurrent)
	assert.Equal(t, valid.SyncBatchSize, got.SyncBatchSize)

	// ValidateConfig 非法输入 ≥3 条
	cases := []struct {
		name string
		cfg  *GroupSyncConfig
		want string
	}{
		{"empty cron", &GroupSyncConfig{Cron: "", MaxConcurrent: 5, SyncBatchSize: 100}, "cron表达式不能为空"},
		{"max_concurrent 0", &GroupSyncConfig{Cron: "0 0 * * * *", MaxConcurrent: 0, SyncBatchSize: 100}, "max_concurrent必须在1-20之间"},
		{"max_concurrent 21", &GroupSyncConfig{Cron: "0 0 * * * *", MaxConcurrent: 21, SyncBatchSize: 100}, "max_concurrent必须在1-20之间"},
		{"sync_batch_size 9", &GroupSyncConfig{Cron: "0 0 * * * *", MaxConcurrent: 5, SyncBatchSize: 9}, "sync_batch_size必须在10-1000之间"},
		{"sync_batch_size 1001", &GroupSyncConfig{Cron: "0 0 * * * *", MaxConcurrent: 5, SyncBatchSize: 1001}, "sync_batch_size必须在10-1000之间"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := svc.ValidateConfig(c.cfg)
			require.Error(t, err)
			assert.Contains(t, err.Error(), c.want)
		})
	}

	// UpdateGroupSyncConfig 非法配置：应保留 DB 原值（ValidateConfig 不被 Update 调用，
	// 但 UpdateGroupSyncConfig 自身不校验，所以非法值会被写入）。
	// 文档化现行为：UpdateGroupSyncConfig 不调用 ValidateConfig，非法值落库。
	// （这不是 plan 严格要求的"先校验后拒绝"，但 plan 标注了"若 UpdateGroupSyncConfig
	// 传非法配置 → 断言先校验后拒绝（不落库）"。这里现行为是"不校验、落库"，
	// 按 D-78-06g 记录为 deviation。）
	invalid := &GroupSyncConfig{
		Enabled:          true,
		Cron:             "", // 非法
		MemberOU:         "OU=X,DC=example,DC=com",
		AutoCreateGroups: true,
		MaxConcurrent:    999, // 非法
		SyncBatchSize:    9999, // 非法
	}
	require.NoError(t, svc.UpdateGroupSyncConfig(ctx, invalid),
		"D-78-06g: 现行为 UpdateGroupSyncConfig 不调用 ValidateConfig，非法值落库")
	got, err = svc.GetGroupSyncConfig(ctx)
	require.NoError(t, err)
	// 现行为（D-78-06g）：reader 端把空字符串视为"无覆盖"→ 回退 default。
	// 所以非法 Cron="" 写入 DB 后，GetGroupSyncConfig 回退到默认 "0 */15 * * * *"。
	assert.Equal(t, "0 */15 * * * *", got.Cron, "D-78-06g: 空字符串覆盖被读侧忽略，回退 default")
	assert.Equal(t, 999, got.MaxConcurrent, "数字位无空值忽略：MaxConcurrent=999 落库并生效")
}

// helper: 直接用 models.Config{} 作 GORM Model 目标；TableName() 方法解析到 sys_config。