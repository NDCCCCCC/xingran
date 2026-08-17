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

// TestConfigService_Statistics 验证参数配置统计:
// 按 config_type(Y/N) 聚合 + 排除软删除,专用 COUNT 端点不受 MaxPageSize=100 钳制。
func TestConfigService_Statistics(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "create sqlite test db")

	require.NoError(t, db.Exec(`
		CREATE TABLE sys_config (
			id TEXT PRIMARY KEY,
			config_name TEXT,
			config_key TEXT,
			config_type TEXT,
			deleted_at DATETIME,
			created_at DATETIME
		)
	`).Error)

	now := time.Now().Format("2006-01-02 15:04:05")
	insert := func(prefix string, n int, ct string) {
		for i := 0; i < n; i++ {
			require.NoError(t, db.Exec(
				`INSERT INTO sys_config (id, config_name, config_key, config_type, created_at) VALUES (?, ?, ?, ?, ?)`,
				fmt.Sprintf("%s-%d", prefix, i), prefix, fmt.Sprintf("%s.key.%d", prefix, i), ct, now,
			).Error)
		}
	}
	// 60 个 Y + 40 个 N = 100
	insert("y", 60, "Y")
	insert("n", 40, "N")
	// 10 条已软删除(Y),不计入统计
	for i := 0; i < 10; i++ {
		require.NoError(t, db.Exec(
			`INSERT INTO sys_config (id, config_name, config_key, config_type, created_at, deleted_at) VALUES (?, ?, ?, 'Y', ?, ?)`,
			fmt.Sprintf("d-%d", i), "d", fmt.Sprintf("d.key.%d", i), now, now,
		).Error)
	}

	svc := &configService{db: db}
	result, err := svc.Statistics(context.Background())
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, int64(100), result.Total, "排除 10 条软删除")
	require.Equal(t, int64(60), result.Active, "config_type=Y")
	require.Equal(t, int64(40), result.Inactive, "config_type=N")
}
