package asset

// Phase 44 R3 / Plan 44-02 Task 2 — 降噪基线 Snapshot/Compare service 测试
//
// 测试覆盖:
//  1. Snapshot 写 sys_config (config_key=asset.reconciliation.baseline), 含 4 字段
//  2. Snapshot COUNT 反映当前 DB 数据(500/120/30)
//  3. Snapshot 幂等覆盖(key 存在则 Update)
//  4. Snapshot COUNT 含 silence 记录(WARN-8: WHERE 仅 deleted_at IS NULL, 不加 silence 过滤)
//  5. Compare 返回下降百分比
//  6. Compare 无 baseline 返回 error
//  7. Compare 用独立 COUNT, 不调 ListExceptions(静态源码扫描)

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupBaselineTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:baseline_"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE IF NOT EXISTS sys_data_reconciliation (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			asset_id TEXT,
			conflict_type TEXT,
			severity TEXT,
			applied_actions TEXT,
			detected_at DATETIME,
			resolved_at DATETIME,
			workorder_id TEXT
		)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE IF NOT EXISTS sys_config (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			config_name TEXT,
			config_key TEXT,
			config_value TEXT,
			config_type TEXT,
			is_system INTEGER DEFAULT 0,
			remark TEXT
		)
	`).Error)
	return db
}

// insertExceptions 插入 N 条异常(severity + workorder_id + applied_actions 可控)
//
// prefix 用于避免不同批次的 ID 冲突(同 severity 多次调用时 prefix 区分)。
func insertExceptions(t *testing.T, db *gorm.DB, prefix string, n int, severity string, withWorkorder bool, actions string) {
	t.Helper()
	for i := 0; i < n; i++ {
		var woID interface{} = nil
		if withWorkorder {
			woID = "wo-" + prefix + "-" + severity + "-" + string(rune('a'+i%26)) + "-" + string(rune('A'+i/26))
		}
		// ID = prefix + severity + index(双重去重防同 severity 多批次冲突)
		id := "rec-" + prefix + "-" + severity + "-" + string(rune('a'+i%26)) + "-" + string(rune('A'+i/26))
		// 直接 INSERT (applied_actions 在 SQLite 用 TEXT 模拟,生产是 TEXT[])
		require.NoError(t, db.Exec(`INSERT INTO sys_data_reconciliation (id, asset_id, conflict_type, severity, applied_actions, detected_at, workorder_id, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			id,
			"asset-"+id,
			"B",
			severity,
			actions,
			time.Now(),
			woID,
			time.Now(),
		).Error)
	}
}

// TestSnapshotCreatesConfig Snapshot 写 sys_config, JSON 含 4 字段
func TestSnapshotCreatesConfig(t *testing.T) {
	db := setupBaselineTestDB(t)
	svc := NewReconciliationBaselineService(db)
	ctx := context.Background()

	snap, err := svc.Snapshot(ctx)
	require.NoError(t, err)
	require.NotNil(t, snap)
	assert.False(t, snap.SnapshotAt.IsZero(), "SnapshotAt 必须被填充")
	assert.Equal(t, int64(0), snap.TotalExceptions, "空 DB 应为 0")

	// 验证 sys_config 写入
	var configValue string
	require.NoError(t, db.Raw(`SELECT config_value FROM sys_config WHERE config_key = 'asset.reconciliation.baseline'`).Scan(&configValue).Error)
	assert.NotEmpty(t, configValue, "sys_config 必须写入 baseline")

	// JSON 含 4 字段
	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(configValue), &parsed))
	assert.Contains(t, parsed, "snapshot_at")
	assert.Contains(t, parsed, "total_exceptions")
	assert.Contains(t, parsed, "total_workorders")
	assert.Contains(t, parsed, "critical_exceptions")
}

// TestSnapshotReflectsCurrentCounts COUNT 反映 DB 真实数据
func TestSnapshotReflectsCurrentCounts(t *testing.T) {
	db := setupBaselineTestDB(t)
	ctx := context.Background()

	// 500 总异常 + 120 工单(workorder_id 非 NULL)+ 30 critical
	insertExceptions(t, db, "high-no-wo", 350, "high", false, "")
	insertExceptions(t, db, "crit-wo", 120, "critical", true, "")
	insertExceptions(t, db, "crit-no-wo", 30, "critical", false, "")

	svc := NewReconciliationBaselineService(db)
	snap, err := svc.Snapshot(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(500), snap.TotalExceptions, "总异常 = 500")
	assert.Equal(t, int64(120), snap.TotalWorkorders, "工单数 = 120(workorder_id 非 NULL)")
	assert.Equal(t, int64(150), snap.CriticalExceptions, "critical 数 = 120+30=150")
}

// TestSnapshotIdempotentOverwrite 二次 Snapshot 覆盖,sys_config 不出现重复 key
func TestSnapshotIdempotentOverwrite(t *testing.T) {
	db := setupBaselineTestDB(t)
	ctx := context.Background()
	svc := NewReconciliationBaselineService(db)

	_, err := svc.Snapshot(ctx)
	require.NoError(t, err)

	insertExceptions(t, db, "after-snapshot", 10, "high", false, "")
	_, err = svc.Snapshot(ctx)
	require.NoError(t, err)

	// sys_config 仅有 1 条 baseline key
	var count int64
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM sys_config WHERE config_key = 'asset.reconciliation.baseline'`).Scan(&count).Error)
	assert.Equal(t, int64(1), count, "二次 Snapshot 应覆盖现有 baseline,sys_config 不重复")
}

// TestSnapshotCountIncludesSilenceRecords WARN-8: COUNT 含 silence 记录
//
// silence 是降噪手段之一,应计入"当前告警量"基准;
// ListExceptions UI 默认隐藏 silence 是运维视图偏好,与 baseline COUNT 解耦。
func TestSnapshotCountIncludesSilenceRecords(t *testing.T) {
	db := setupBaselineTestDB(t)
	ctx := context.Background()

	// 480 普通 + 20 silence(用 "[\"silence\"]" 模拟 PG TEXT[] 的 JSON 序列化)
	insertExceptions(t, db, "normal", 480, "medium", false, "")
	insertExceptions(t, db, "silenced", 20, "medium", false, `[\"silence\"]`)

	svc := NewReconciliationBaselineService(db)
	snap, err := svc.Snapshot(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(500), snap.TotalExceptions,
		"Snapshot COUNT 必须含 silence 记录 (WARN-8: WHERE 仅 deleted_at IS NULL, 不加 silence 过滤)")
}

// TestCompareReturnsReductionPct baseline 500/120/150-critical, 当前 200/50/10
//
// baseline:
//   - 总异常 = 350(high,no-wo) + 120(critical, wo) + 30(critical, no-wo) = 500
//   - 工单数 = 120(仅 critical+wo)
//   - critical 数 = 120 + 30 = 150(critical 与是否有工单无关)
//
// current:
//   - 总异常 = 140(medium) + 50(high, wo) + 10(critical, no-wo) = 200
//   - 工单数 = 50
//   - critical 数 = 10
//
// 异常下降% = (500-200)/500*100 = 60.0
// 工单下降% = (120-50)/120*100 ≈ 58.33
// critical 下降% = (150-10)/150*100 ≈ 93.33
func TestCompareReturnsReductionPct(t *testing.T) {
	db := setupBaselineTestDB(t)
	ctx := context.Background()

	// 先写 baseline (DB 含 500/120/150-critical)
	insertExceptions(t, db, "high-no-wo", 350, "high", false, "")
	insertExceptions(t, db, "crit-wo", 120, "critical", true, "")
	insertExceptions(t, db, "crit-no-wo", 30, "critical", false, "")

	svc := NewReconciliationBaselineService(db)
	_, err := svc.Snapshot(ctx)
	require.NoError(t, err)

	// 删除大部分数据模拟降噪后(剩余 200/50/10)
	require.NoError(t, db.Exec(`DELETE FROM sys_data_reconciliation`).Error)
	insertExceptions(t, db, "after-medium", 140, "medium", false, "")
	insertExceptions(t, db, "after-high-wo", 50, "high", true, "")
	insertExceptions(t, db, "after-crit", 10, "critical", false, "")

	result, err := svc.Compare(ctx)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.InDelta(t, 60.0, result.ExceptionsReductionPct, 0.1, "异常下降% ≈ 60.0")
	assert.InDelta(t, 58.33, result.WorkordersReductionPct, 0.1, "工单下降% ≈ 58.33")
	assert.InDelta(t, 93.33, result.CriticalReductionPct, 0.1, "critical 下降% ≈ 93.33 (baseline=150, current=10)")
}

// TestCompareNoBaselineReturnsError 无 baseline 时 Compare 返回 error 引导用户先 Snapshot
func TestCompareNoBaselineReturnsError(t *testing.T) {
	db := setupBaselineTestDB(t)
	ctx := context.Background()
	svc := NewReconciliationBaselineService(db)

	_, err := svc.Compare(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "基线", "无 baseline 错误信息引导用户先 Snapshot")
}

// TestCompareCountsUseIndependentCountNotListLength 静态源码扫描:
//  1. countCurrent 函数体不含 ListExceptions 调用
//  2. countCurrent 函数体含 3 个 Count( 独立 COUNT
//
// 注:Go 格式化后方法链常把 . 放前一行,Count( 在新行开头,
// 故扫描用 "Count(" (不带前导点) 而非 ".Count("。
func TestCompareCountsUseIndependentCountNotListLength(t *testing.T) {
	src := mustReadBaselineSrc(t)
	// countCurrent 是实际做 COUNT 的函数,提取其函数体扫描
	body := extractBaselineFuncBody(t, src, "countCurrent")
	require.NotEmpty(t, body, "countCurrent 函数必须存在")
	// 严禁调 ListExceptions (Pitfall 5: MaxPageSize=100 钳制)
	assert.NotContains(t, body, "ListExceptions",
		"countCurrent 严禁调 ListExceptions (Pitfall 5: MaxPageSize=100 钳制)")
	// 含 3 个独立 Count(&...) (total / totalWorkorders / critical)
	assert.GreaterOrEqual(t, strings.Count(body, "Count("), 3,
		"countCurrent 必须用独立 Count(&...) 3 次: total/totalWorkorders/critical")
}

// TestSnapshotDoesNotFilterSilenceStatic 静态断言:Snapshot 函数体无 silence 过滤(WARN-8)
func TestSnapshotDoesNotFilterSilenceStatic(t *testing.T) {
	src := mustReadBaselineSrc(t)
	snapshotBody := extractBaselineFuncBody(t, src, "Snapshot")
	assert.NotContains(t, snapshotBody, "silence",
		"Snapshot COUNT 严禁加 silence 过滤(WARN-8: 与 ListExceptions UI 隐藏 silence 的语义解耦)")
	assert.NotContains(t, snapshotBody, "applied_actions",
		"Snapshot 不读 applied_actions 列(WARN-8: WHERE 仅 deleted_at IS NULL)")
}

// TestBaselineServiceHasOperationalDoc 静态断言:文件头/Snapshot doc 含运维文档化注释
//
// BLOCKER-3: SC 8 ≥60% 降噪量化验证前置 — 运维必须在 R3 部署前 + R2 数据保留期内调用 Snapshot。
// doc comment 必须明确告知运维责任。
func TestBaselineServiceHasOperationalDoc(t *testing.T) {
	src := mustReadBaselineSrc(t)
	assert.True(t,
		strings.Contains(src, "R3 部署前") || strings.Contains(src, "R2 末期基线"),
		"baseline service 必须在文件头或 Snapshot doc 含运维文档化注释(BLOCKER-3)")
}

// helpers

func mustReadBaselineSrc(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("reconciliation_baseline.go")
	require.NoError(t, err, "must read reconciliation_baseline.go (Task 2 GREEN 应创建此文件)")
	return string(data)
}

func extractBaselineFuncBody(t *testing.T, src, funcName string) string {
	t.Helper()
	needle := "func (s *reconciliationBaselineServiceImpl) " + funcName + "("
	idx := strings.Index(src, needle)
	if idx < 0 {
		return ""
	}
	start := strings.Index(src[idx:], "{")
	if start < 0 {
		return ""
	}
	start += idx
	depth := 0
	for i := start; i < len(src); i++ {
		switch src[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return src[start : i+1]
			}
		}
	}
	return src[start:]
}

// 注:上述 helper 同时支持导出 (Snapshot/Compare) 和未导出 (countCurrent) 方法签名。
