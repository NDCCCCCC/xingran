package workorder

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestPeriodicService_GetStatistics 验证周期性工单模板统计:
// 1) 用条件聚合正确统计总数/启停数(is_enabled 布尔跨库——PG boolean / SQLite 0-1);
// 2) SUM(total_generated) 用 COALESCE 防空集 NULL;
// 3) 软删除模板被排除。
func TestPeriodicService_GetStatistics(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "create sqlite test db")

	require.NoError(t, db.Exec(`
		CREATE TABLE sys_periodic_workorder_template (
			id TEXT PRIMARY KEY,
			title TEXT,
			is_enabled INTEGER DEFAULT 1,
			total_generated INTEGER DEFAULT 0,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)
	`).Error)

	now := "2024-01-01 00:00:00"
	// 60 启用(各生成 3) + 40 停用(各生成 5) = 100 模板
	for i := 0; i < 60; i++ {
		require.NoError(t, db.Exec(
			`INSERT INTO sys_periodic_workorder_template (id, title, is_enabled, total_generated, created_at, updated_at, deleted_at) VALUES (?, ?, 1, 3, ?, ?, NULL)`,
			fmt.Sprintf("e-%d", i), fmt.Sprintf("e%d", i), now, now,
		).Error)
	}
	for i := 0; i < 40; i++ {
		require.NoError(t, db.Exec(
			`INSERT INTO sys_periodic_workorder_template (id, title, is_enabled, total_generated, created_at, updated_at, deleted_at) VALUES (?, ?, 0, 5, ?, ?, NULL)`,
			fmt.Sprintf("d-%d", i), fmt.Sprintf("d%d", i), now, now,
		).Error)
	}
	// 1 个软删除(生成 99),必须排除
	require.NoError(t, db.Exec(
		`INSERT INTO sys_periodic_workorder_template (id, title, is_enabled, total_generated, created_at, updated_at, deleted_at) VALUES ('ghost', 'g', 1, 99, ?, ?, ?)`,
		now, now, now,
	).Error)

	svc := &PeriodicService{db: db}
	result, err := svc.GetStatistics(context.Background())
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, int64(100), result.Total, "total 必须排除软删除")
	require.Equal(t, int64(60), result.Enabled, "enabled = is_enabled 为真的模板数")
	require.Equal(t, int64(40), result.Disabled, "disabled = is_enabled 为假的模板数")
	require.Equal(t, int64(380), result.TotalGenerated, "totalGenerated = 60*3 + 40*5 = 380,软删除的 99 必须排除")
}
