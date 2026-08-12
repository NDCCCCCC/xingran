package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestDeviceDiscoveryService_GetStatistics 验证设备发现统计:
// int 状态枚举聚合(0待执行/1执行中/2成功/3失败) + SUM(discovered_count),
// 用条件聚合避免加载全量行。DeviceDiscovery 无软删除。
func TestDeviceDiscoveryService_GetStatistics(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "create sqlite test db")

	require.NoError(t, db.Exec(`
		CREATE TABLE sys_device_discovery (
			id TEXT PRIMARY KEY,
			task_name TEXT,
			status INTEGER DEFAULT 0,
			discovered_count INTEGER DEFAULT 0,
			created_at DATETIME
		)
	`).Error)

	now := "2024-01-01 00:00:00"
	insertN := func(prefix string, n, status, discovered int) {
		for i := 0; i < n; i++ {
			require.NoError(t, db.Exec(
				`INSERT INTO sys_device_discovery (id, task_name, status, discovered_count, created_at) VALUES (?, ?, ?, ?, ?)`,
				fmt.Sprintf("%s-%d", prefix, i), fmt.Sprintf("%s%d", prefix, i), status, discovered, now,
			).Error)
		}
	}
	// 30 待执行(0) + 20 执行中(1) + 40 成功(2,各发现5) + 10 失败(3,各发现2) = 100 任务
	insertN("p", 30, 0, 0)
	insertN("r", 20, 1, 0)
	insertN("c", 40, 2, 5)
	insertN("f", 10, 3, 2)

	svc := &DeviceDiscoveryService{db: db}
	result, err := svc.GetStatistics(context.Background())
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, int64(100), result.Total)
	require.Equal(t, int64(30), result.Pending, "pending = status=0")
	require.Equal(t, int64(20), result.Running, "running = status=1")
	require.Equal(t, int64(40), result.Completed, "completed = status=2")
	require.Equal(t, int64(10), result.Failed, "failed = status=3")
	require.Equal(t, int64(220), result.TotalDevices, "totalDevices = 40*5 + 10*2 = 220")
}
