package operations

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupAssetListFilterDB 建立内存 SQLite 数据库 + ops_asset 表(含 component_type 列)。
// 仅创建 List/Statistics 路径必需的列,避免维护 60+ 列 schema(参考 asset_statistics_test.go 模式)。
func setupAssetListFilterDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "create sqlite test db")
	require.NoError(t, db.Exec(`
		CREATE TABLE ops_asset (
			id TEXT PRIMARY KEY,
			devicesn TEXT,
			device_model_name TEXT,
			component_type TEXT,
			status INTEGER DEFAULT 0,
			nbf_status INTEGER DEFAULT 0,
			deleted_at DATETIME,
			created_at DATETIME
		)
	`).Error, "create ops_asset table")
	return db
}

// insertAssetRow 插入一行 ops_asset,componentType 为空字符串视为主设备(NULL),非空视为组件。
func insertAssetRow(t *testing.T, db *gorm.DB, id, deviceSN, componentType string) {
	t.Helper()
	now := time.Now().Format("2006-01-02 15:04:05")
	var ct interface{}
	if componentType != "" {
		ct = componentType
	}
	require.NoError(t, db.Exec(
		`INSERT INTO ops_asset (id, devicesn, component_type, status, nbf_status, created_at) VALUES (?, ?, ?, 0, 0, ?)`,
		id, deviceSN, ct, now,
	).Error, "insert ops_asset row")
}

// TestAssetListExcludesComponents 验证 List() 默认过滤 component_type IS NULL(D-07):
// 即便 params 不传任何筛选,组件行也不出现在常规列表。
func TestAssetListExcludesComponents(t *testing.T) {
	db := setupAssetListFilterDB(t)
	insertAssetRow(t, db, "asset-main-1", "SWITCH-SN-001", "")
	insertAssetRow(t, db, "asset-main-2", "SWITCH-SN-002", "")
	insertAssetRow(t, db, "asset-card-1", "CARD-SN-001", "card")
	insertAssetRow(t, db, "asset-fan-1", "FAN-SN-001", "fan")

	svc := &assetService{db: db}
	result, err := svc.List(context.Background(), map[string]interface{}{})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, int64(2), result.Total, "2 主设备行入选,2 组件行(component_type IS NOT NULL)被默认排除")

	// 同步独立校验 component_type IS NULL 子句对全表的行为(防 service 实现意外删 WHERE)
	var filteredCount int64
	require.NoError(t, db.Table("ops_asset").Where("component_type IS NULL").Count(&filteredCount).Error)
	require.Equal(t, int64(2), filteredCount, "component_type IS NULL 子句独立验证 2 行主设备")

	var allCount int64
	require.NoError(t, db.Table("ops_asset").Count(&allCount).Error)
	require.Equal(t, int64(4), allCount, "全表 4 行(2 主 + 2 组件)")
}

// TestAssetStatisticsExcludesComponents 验证 Statistics() 默认排除 component_type IS NOT NULL(D-07):
// total/normal/stopped/nbf 都不含组件行。
func TestAssetStatisticsExcludesComponents(t *testing.T) {
	db := setupAssetListFilterDB(t)
	// 3 主设备:main-1 正常 / main-2 停用 / main-3 正常且拟报废
	insertAssetRow(t, db, "asset-main-1", "SWITCH-SN-001", "")
	insertAssetRow(t, db, "asset-main-2", "SWITCH-SN-002", "")
	insertAssetRow(t, db, "asset-main-3", "SWITCH-SN-003", "")
	require.NoError(t, db.Exec(`UPDATE ops_asset SET status=1 WHERE id=?`, "asset-main-2").Error)
	require.NoError(t, db.Exec(`UPDATE ops_asset SET nbf_status=1 WHERE id=?`, "asset-main-3").Error)

	// 3 组件:card(也置 status=1,验证即便组件有非默认状态也不会混入统计)/fan/transceiver
	insertAssetRow(t, db, "asset-card-1", "CARD-SN-001", "card")
	insertAssetRow(t, db, "asset-fan-1", "FAN-SN-001", "fan")
	insertAssetRow(t, db, "asset-transceiver-1", "XFP-SN-001", "transceiver")
	require.NoError(t, db.Exec(`UPDATE ops_asset SET status=1 WHERE id=?`, "asset-card-1").Error)

	svc := &assetService{db: db}
	result, err := svc.Statistics(context.Background())
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, int64(3), result.Total, "Statistics 排除 3 组件行,仅 3 主设备")
	require.Equal(t, int64(2), result.Normal, "status=0 主设备 2 个(main-1 + main-3)")
	require.Equal(t, int64(1), result.Stopped, "status=1 主设备 1 个(main-2)")
	require.Equal(t, int64(1), result.NBF, "nbf_status=1 主设备 1 个(main-3;组件 card-1 即便 status=1 也不计入)")
}

// TestAssetListFilterDoesNotBreakExistingFilters 验证 component_type IS NULL 默认过滤
// 与现有筛选(devicesn LIKE)并存:既排除组件,又保留 LIKE 过滤。
func TestAssetListFilterDoesNotBreakExistingFilters(t *testing.T) {
	db := setupAssetListFilterDB(t)
	insertAssetRow(t, db, "asset-main-1", "SWITCH-SN-001", "")
	insertAssetRow(t, db, "asset-main-2", "ROUTER-SN-001", "")
	insertAssetRow(t, db, "asset-card-1", "SWITCH-CARD-001", "card")
	insertAssetRow(t, db, "asset-fan-1", "SWITCH-FAN-001", "fan")

	svc := &assetService{db: db}
	// 用 devicesn LIKE 'SWITCH' 筛选 → 应该只命中 1 行主设备(SWITCH-SN-001)
	// 2 行组件(SWITCH-CARD-001/SWITCH-FAN-001)即便 devicesn 匹配也应被 component_type IS NULL 排除
	result, err := svc.List(context.Background(), map[string]interface{}{
		"devicesn": "SWITCH",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, int64(1), result.Total, "LIKE SWITCH 仅命中 1 主设备,2 组件被 component_type IS NULL 排除")
}
