package scheduler

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// TestJobLogService_Statistics 验证任务日志统计:
// 1) 按 jobName/jobGroup 过滤(只计指定任务的日志,排除其他任务);
// 2) 条件聚合正确统计 total/success(status=0)/fail(status=1);
// 3) 不再受 list pageSize:50 截断(>50 次执行时旧卡片卡在 50)。
func TestJobLogService_Statistics(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "create sqlite test db")

	require.NoError(t, db.Exec(`
		CREATE TABLE sys_job_log (
			id TEXT PRIMARY KEY,
			job_name TEXT,
			job_group TEXT,
			status INTEGER DEFAULT 0,
			created_at DATETIME,
			deleted_at DATETIME
		)
	`).Error)

	now := "2024-01-01 00:00:00"
	insertN := func(prefix string, n, status int, job, group string) {
		for i := 0; i < n; i++ {
			require.NoError(t, db.Exec(
				`INSERT INTO sys_job_log (id, job_name, job_group, status, created_at, deleted_at) VALUES (?, ?, ?, ?, ?, NULL)`,
				fmt.Sprintf("%s-%d", prefix, i), job, group, status, now,
			).Error)
		}
	}
	// 任务 "cleanup"(group "default"): 60 成功 + 30 失败 = 90 次(> pageSize 50)
	insertN("c-ok", 60, 0, "cleanup", "default")
	insertN("c-fail", 30, 1, "cleanup", "default")
	// 另一个任务 "other",不得计入 cleanup 的统计
	insertN("o-ok", 40, 0, "other", "default")
	// 注:JobLog 无软删除(CleanOldLogs 为硬删除),故不设软删除排除断言。

	svc := &jobLogServiceImpl{db: db}
	result, err := svc.Statistics(context.Background(), &JobLogListParams{JobName: "cleanup", JobGroup: "default"})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, int64(90), result.Total, "cleanup 总次数 = 60+30,排除 other 任务,且不受 pageSize:50 截断")
	require.Equal(t, int64(60), result.Success, "success = status=0 的次数")
	require.Equal(t, int64(30), result.Fail, "fail = status=1 的次数")
}
