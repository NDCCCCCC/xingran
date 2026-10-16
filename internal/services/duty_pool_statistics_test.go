package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// TestDutyPoolStatistics_NotDerivedFromCurrentPage
// 回归测试:GetDutyPoolStatistics 必须返回真实的池总数/启停数/成员总数,
// 而不是像旧前端那样用「当前页 list(~10 条)」计算(多页时严重偏小)。
// 同时验证 totalMembers 只统计非软删除池的成员(子查询套用软删除 scope)。
func TestDutyPoolStatistics_NotDerivedFromCurrentPage(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "create sqlite test db")

	require.NoError(t, db.Exec(`
		CREATE TABLE sys_duty_pool (
			id TEXT PRIMARY KEY,
			pool_name TEXT,
			status INTEGER DEFAULT 0,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_duty_pool_member (
			id TEXT PRIMARY KEY,
			pool_id TEXT,
			user_id TEXT,
			member_order INTEGER DEFAULT 0,
			created_at DATETIME
		)
	`).Error)

	now := "2024-01-01 00:00:00"
	// 70 启用(status=0) + 50 停用(status=1) = 120 个活跃池,远超单页
	for i := 0; i < 70; i++ {
		require.NoError(t, db.Exec(
			`INSERT INTO sys_duty_pool (id, pool_name, status, created_at, updated_at, deleted_at) VALUES (?, ?, 0, ?, ?, NULL)`,
			fmt.Sprintf("enabled-%d", i), fmt.Sprintf("e%d", i), now, now,
		).Error)
	}
	for i := 0; i < 50; i++ {
		require.NoError(t, db.Exec(
			`INSERT INTO sys_duty_pool (id, pool_name, status, created_at, updated_at, deleted_at) VALUES (?, ?, 1, ?, ?, NULL)`,
			fmt.Sprintf("disabled-%d", i), fmt.Sprintf("d%d", i), now, now,
		).Error)
	}
	// 1 个软删除池,必须排除;其成员也应被排除
	require.NoError(t, db.Exec(
		`INSERT INTO sys_duty_pool (id, pool_name, status, created_at, updated_at, deleted_at) VALUES ('ghost-pool','ghost',0,?,?,?)`,
		now, now, now,
	).Error)

	// 成员:每个启用池 2 个(=140) + 每个停用池 1 个(=50) + 软删除池 5 个(应排除)
	memberSeq := 0
	addMembers := func(poolID string, n int) {
		for j := 0; j < n; j++ {
			require.NoError(t, db.Exec(
				`INSERT INTO sys_duty_pool_member (id, pool_id, user_id, member_order, created_at) VALUES (?, ?, ?, ?, ?)`,
				fmt.Sprintf("m-%d", memberSeq), poolID, fmt.Sprintf("u-%d", memberSeq), j, now,
			).Error)
			memberSeq++
		}
	}
	for i := 0; i < 70; i++ {
		addMembers(fmt.Sprintf("enabled-%d", i), 2)
	}
	for i := 0; i < 50; i++ {
		addMembers(fmt.Sprintf("disabled-%d", i), 1)
	}
	addMembers("ghost-pool", 5)

	svc := &DutyPoolService{db: db}

	result, err := svc.GetDutyPoolStatistics(context.Background())
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, int64(120), result.Total, "total 必须是真实池数 120,而非当前页条数")
	require.Equal(t, int64(70), result.Enabled, "enabled 必须为 status=0 的池数")
	require.Equal(t, int64(50), result.Disabled, "disabled 必须为 status=1 的池数")
	require.Equal(t, int64(190), result.TotalMembers, "totalMembers=140+50=190,软删除池的 5 个成员必须排除")
}
