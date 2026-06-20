package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestCommandDispatchService_GetStatistics 验证命令执行统计:
// 按 execution_type='command' 过滤(排除 template 类型) + int 状态聚合。
// ConfigExecution 无软删除。同一张表 sys_config_execution 同时存 command/template 两类执行。
func TestCommandDispatchService_GetStatistics(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "create sqlite test db")

	require.NoError(t, db.Exec(`
		CREATE TABLE sys_config_execution (
			id TEXT PRIMARY KEY,
			execution_type TEXT,
			status INTEGER DEFAULT 0,
			created_at DATETIME
		)
	`).Error)

	now := "2024-01-01 00:00:00"
	insertCmd := func(prefix string, n, status int) {
		for i := 0; i < n; i++ {
			require.NoError(t, db.Exec(
				`INSERT INTO sys_config_execution (id, execution_type, status, created_at) VALUES (?, 'command', ?, ?)`,
				fmt.Sprintf("%s-%d", prefix, i), status, now,
			).Error)
		}
	}
	// command 类: 25 待执行(0) + 15 执行中(1) + 50 成功(2) + 10 失败(3) = 100
	insertCmd("p", 25, 0)
	insertCmd("r", 15, 1)
	insertCmd("s", 50, 2)
	insertCmd("f", 10, 3)
	// template 类(30 条),不得计入 command 统计
	for i := 0; i < 30; i++ {
		require.NoError(t, db.Exec(
			`INSERT INTO sys_config_execution (id, execution_type, status, created_at) VALUES (?, 'template', 2, ?)`,
			fmt.Sprintf("t-%d", i), now,
		).Error)
	}

	svc := &CommandDispatchService{db: db}
	result, err := svc.GetStatistics(context.Background())
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, int64(100), result.Total, "total 只计 command 类,排除 template")
	require.Equal(t, int64(25), result.Pending)
	require.Equal(t, int64(15), result.Running)
	require.Equal(t, int64(50), result.Success)
	require.Equal(t, int64(10), result.Failed)
}
