package rpa

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// TestExecutionService_Statistics 验证执行记录统计:
// 按字符串 status 聚合(pending/running/success/failed) + 排除软删除,
// 用专用 COUNT 端点而非分页列表 length,不受 pageSize 钳制。
func TestExecutionService_Statistics(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "create sqlite test db")

	require.NoError(t, db.Exec(`
		CREATE TABLE sys_rpa_executions (
			id TEXT PRIMARY KEY,
			task_id TEXT,
			status TEXT,
			deleted_at DATETIME,
			created_at DATETIME
		)
	`).Error)

	now := time.Now().Format("2006-01-02 15:04:05")
	insert := func(prefix string, n int, status string) {
		for i := 0; i < n; i++ {
			require.NoError(t, db.Exec(
				`INSERT INTO sys_rpa_executions (id, task_id, status, created_at) VALUES (?, ?, ?, ?)`,
				fmt.Sprintf("%s-%d", prefix, i), "task-1", status, now,
			).Error)
		}
	}
	// 30 待执行 + 20 执行中 + 50 成功 + 15 失败 = 115
	insert("p", 30, "pending")
	insert("r", 20, "running")
	insert("s", 50, "success")
	insert("f", 15, "failed")
	// 10 条已软删除的成功记录,不计入统计
	for i := 0; i < 10; i++ {
		require.NoError(t, db.Exec(
			`INSERT INTO sys_rpa_executions (id, task_id, status, created_at, deleted_at) VALUES (?, ?, 'success', ?, ?)`,
			fmt.Sprintf("d-%d", i), "task-1", now, now,
		).Error)
	}

	svc := &executionServiceImpl{db: db}
	result, err := svc.Statistics(context.Background())
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, int64(115), result.Total, "total 排除 10 条软删除")
	require.Equal(t, int64(30), result.Pending)
	require.Equal(t, int64(20), result.Running)
	require.Equal(t, int64(50), result.Success)
	require.Equal(t, int64(15), result.Failed)
}
