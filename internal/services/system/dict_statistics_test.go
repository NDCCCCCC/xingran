package system

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// TestDictTypeService_Statistics 验证字典类型统计:
// 按 status(0=正常 1=停用) 聚合 + 排除软删除,专用 COUNT 端点不受 MaxPageSize=100 钳制。
func TestDictTypeService_Statistics(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "create sqlite test db")

	require.NoError(t, db.Exec(`
		CREATE TABLE sys_dict_type (
			id TEXT PRIMARY KEY,
			dict_name TEXT,
			dict_type TEXT,
			status INTEGER DEFAULT 0,
			deleted_at DATETIME,
			created_at DATETIME
		)
	`).Error)

	now := time.Now().Format("2006-01-02 15:04:05")
	insert := func(prefix string, n, status int) {
		for i := 0; i < n; i++ {
			require.NoError(t, db.Exec(
				`INSERT INTO sys_dict_type (id, dict_name, dict_type, status, created_at) VALUES (?, ?, ?, ?, ?)`,
				fmt.Sprintf("%s-%d", prefix, i), prefix, fmt.Sprintf("%s%d", prefix, i), status, now,
			).Error)
		}
	}
	// 50 正常 + 30 停用 = 80
	insert("a", 50, 0)
	insert("i", 30, 1)
	// 10 条已软删除,不计入
	for i := 0; i < 10; i++ {
		require.NoError(t, db.Exec(
			`INSERT INTO sys_dict_type (id, dict_name, dict_type, status, created_at, deleted_at) VALUES (?, ?, ?, 0, ?, ?)`,
			fmt.Sprintf("d-%d", i), "d", fmt.Sprintf("d%d", i), now, now,
		).Error)
	}

	svc := &dictTypeService{db: db}
	result, err := svc.Statistics(context.Background())
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, int64(80), result.Total, "排除 10 条软删除")
	require.Equal(t, int64(50), result.Active)
	require.Equal(t, int64(30), result.Inactive)
}

// TestDictDataService_Statistics 验证字典数据统计:
// 按 status 聚合 + 可选 dictType 过滤 + 排除软删除。
func TestDictDataService_Statistics(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "create sqlite test db")

	require.NoError(t, db.Exec(`
		CREATE TABLE sys_dict_data (
			id TEXT PRIMARY KEY,
			dict_type TEXT,
			dict_label TEXT,
			status INTEGER DEFAULT 0,
			deleted_at DATETIME,
			created_at DATETIME
		)
	`).Error)

	now := time.Now().Format("2006-01-02 15:04:05")
	insert := func(prefix, dt string, n, status int) {
		for i := 0; i < n; i++ {
			require.NoError(t, db.Exec(
				`INSERT INTO sys_dict_data (id, dict_type, dict_label, status, created_at) VALUES (?, ?, ?, ?, ?)`,
				fmt.Sprintf("%s-%d", prefix, i), dt, fmt.Sprintf("%s%d", prefix, i), status, now,
			).Error)
		}
	}
	// typeA: 25 正常 + 15 停用 = 40
	insert("a0", "typeA", 25, 0)
	insert("a1", "typeA", 15, 1)
	// typeB: 30 正常 + 20 停用 = 50 (按 typeA 过滤时不应计入)
	insert("b0", "typeB", 30, 0)
	insert("b1", "typeB", 20, 1)
	// typeA 软删除 5 条 (不计)
	for i := 0; i < 5; i++ {
		require.NoError(t, db.Exec(
			`INSERT INTO sys_dict_data (id, dict_type, dict_label, status, created_at, deleted_at) VALUES (?, 'typeA', ?, 0, ?, ?)`,
			fmt.Sprintf("ad-%d", i), fmt.Sprintf("ad%d", i), now, now,
		).Error)
	}

	svc := &dictDataService{db: db}

	// 按 typeA 过滤
	result, err := svc.Statistics(context.Background(), "typeA")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, int64(40), result.Total, "只计 typeA,排除 typeB 与软删除")
	require.Equal(t, int64(25), result.Active)
	require.Equal(t, int64(15), result.Inactive)

	// 不传 dictType → 全局统计
	all, err := svc.Statistics(context.Background(), "")
	require.NoError(t, err)
	require.Equal(t, int64(90), all.Total, "全局: typeA 40 + typeB 50")
	require.Equal(t, int64(55), all.Active, "25+30")
	require.Equal(t, int64(35), all.Inactive, "15+20")
}
