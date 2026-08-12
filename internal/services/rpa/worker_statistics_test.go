package rpa

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestWorkerService_Statistics 验证 Worker 统计:
// 全量查询后按实时心跳(now - last_heartbeat <= 120s)派生 online/offline/busy/error,
// + 容量聚合(max_concurrency/current_tasks), + 排除软删除。
// 判定顺序与前端 getWorkerActualStatus 一致。
func TestWorkerService_Statistics(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "create sqlite test db")

	require.NoError(t, db.Exec(`
		CREATE TABLE sys_rpa_workers (
			id TEXT PRIMARY KEY,
			worker_name TEXT,
			worker_id TEXT,
			status TEXT,
			max_concurrency INTEGER DEFAULT 3,
			current_tasks INTEGER DEFAULT 0,
			last_heartbeat INTEGER,
			deleted_at DATETIME,
			created_at DATETIME
		)
	`).Error)

	now := time.Now().Unix()
	alive := now      // 心跳在 120s 内 → 在线
	dead := now - 200 // 心跳超 120s → 离线
	createdStr := time.Now().Format("2006-01-02 15:04:05")

	insert := func(id, status string, heartbeat *int64, maxCon, tasks int) {
		require.NoError(t, db.Exec(
			`INSERT INTO sys_rpa_workers (id, worker_name, worker_id, status, max_concurrency, current_tasks, last_heartbeat, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			id, id, id, status, maxCon, tasks, heartbeat, createdStr,
		).Error)
	}
	insert("w-online", "online", &alive, 5, 2)   // 在线, max5 用2
	insert("w-busy", "busy", &alive, 4, 4)       // 忙碌(busy+在线), max4 用4
	insert("w-offline", "online", &dead, 3, 0)   // 心跳超时 → 离线
	insert("w-error", "error", &alive, 2, 0)     // 错误
	// 1 条已软删除,不计入统计
	require.NoError(t, db.Exec(
		`INSERT INTO sys_rpa_workers (id, worker_name, worker_id, status, max_concurrency, current_tasks, last_heartbeat, created_at, deleted_at) VALUES (?, ?, ?, 'online', 3, 0, ?, ?, ?)`,
		"w-deleted", "w-deleted", "w-deleted", &alive, createdStr, createdStr,
	).Error)

	svc := &workerServiceImpl{db: db}
	result, err := svc.Statistics(context.Background())
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, int64(4), result.Total, "排除 1 条软删除")
	require.Equal(t, int64(1), result.Online, "w-online")
	require.Equal(t, int64(1), result.Busy, "w-busy")
	require.Equal(t, int64(1), result.Offline, "w-offline 心跳超时")
	require.Equal(t, int64(1), result.Error, "w-error")
	// totalCapacity = 5+4+3+2 = 14; usedCapacity = 2+4+0+0 = 6
	require.Equal(t, int64(14), result.TotalCapacity)
	require.Equal(t, int64(6), result.UsedCapacity)
}
