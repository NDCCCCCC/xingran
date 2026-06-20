package system

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestUserStatistics_CountExceedsPageSizeCap
// 回归测试:用户数超过 list 端点 MaxPageSize(100) 时,Statistics 必须返回真实总数,
// 而不是被分页上限截断的 100。
//
// 背景:统计卡片原先调用 /system/users/list(pageSize:1000)拉全量、用 list.length 当总数,
// 但后端 user list 的 pageSize 上限为 100(constants.MaxPageSize),导致 1196 用户时
// 卡片错误显示 100。改用专用 Statistics(COUNT 聚合)后应返回真实计数。
// 这里种入 150 用户,断言 total=150(>100)、active/inactive 分别正确,且软删除行被排除。
func TestUserStatistics_CountExceedsPageSizeCap(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "create sqlite test db")

	require.NoError(t, db.Exec(`
		CREATE TABLE sys_user (
			id TEXT PRIMARY KEY,
			username TEXT,
			status INTEGER NOT NULL,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)
	`).Error)

	now := "2024-01-01 00:00:00"
	// 100 个正常用户(status=0)+ 50 个停用用户(status=1)= 150,远超 MaxPageSize=100
	for i := 0; i < 100; i++ {
		require.NoError(t, db.Exec(
			`INSERT INTO sys_user (id, username, status, created_at, updated_at, deleted_at) VALUES (?, ?, 0, ?, ?, NULL)`,
			fmt.Sprintf("active-%d", i), fmt.Sprintf("active%d", i), now, now,
		).Error)
	}
	for i := 0; i < 50; i++ {
		require.NoError(t, db.Exec(
			`INSERT INTO sys_user (id, username, status, created_at, updated_at, deleted_at) VALUES (?, ?, 1, ?, ?, NULL)`,
			fmt.Sprintf("inactive-%d", i), fmt.Sprintf("inactive%d", i), now, now,
		).Error)
	}
	// 1 个软删除用户,必须被排除在统计之外(验证与 List 的软删除口径一致)
	require.NoError(t, db.Exec(
		`INSERT INTO sys_user (id, username, status, created_at, updated_at, deleted_at) VALUES ('ghost-1','ghost',0,?,?,?)`,
		now, now, now,
	).Error)

	svc := &userService{db: db, pwdManager: nil}

	result, err := svc.Statistics(context.Background())
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, int64(150), result.Total, "total 必须是真实计数 150,而非被 pageSize 上限截断的 100")
	require.Equal(t, int64(100), result.Active, "active 必须正确统计 status=0 的用户")
	require.Equal(t, int64(50), result.Inactive, "inactive 必须正确统计 status=1 的用户")
}
