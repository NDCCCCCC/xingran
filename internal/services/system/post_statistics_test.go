package system

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestPostService_Statistics 验证岗位统计:
// 按 status(0=正常 1=停用) 聚合 + 排除软删除,专用 COUNT 端点不受 MaxPageSize=100 钳制。
func TestPostService_Statistics(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "create sqlite test db")

	require.NoError(t, db.Exec(`
		CREATE TABLE sys_post (
			id TEXT PRIMARY KEY,
			post_code TEXT,
			post_name TEXT,
			status INTEGER DEFAULT 0,
			deleted_at DATETIME,
			created_at DATETIME
		)
	`).Error)

	now := time.Now().Format("2006-01-02 15:04:05")
	insert := func(prefix string, n, status int) {
		for i := 0; i < n; i++ {
			require.NoError(t, db.Exec(
				`INSERT INTO sys_post (id, post_code, post_name, status, created_at) VALUES (?, ?, ?, ?, ?)`,
				fmt.Sprintf("%s-%d", prefix, i), fmt.Sprintf("c%d", i), prefix, status, now,
			).Error)
		}
	}
	// 70 正常(0) + 30 停用(1) = 100
	insert("a", 70, 0)
	insert("i", 30, 1)
	// 10 条已软删除,不计入统计
	for i := 0; i < 10; i++ {
		require.NoError(t, db.Exec(
			`INSERT INTO sys_post (id, post_code, post_name, status, created_at, deleted_at) VALUES (?, ?, ?, 0, ?, ?)`,
			fmt.Sprintf("d-%d", i), fmt.Sprintf("dc%d", i), "d", now, now,
		).Error)
	}

	svc := &postService{db: db}
	result, err := svc.Statistics(context.Background())
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, int64(100), result.Total, "排除 10 条软删除")
	require.Equal(t, int64(70), result.Active, "status=0")
	require.Equal(t, int64(30), result.Inactive, "status=1")
}
