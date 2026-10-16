package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// TestNoticeStatusStatistics_CountExceedsPageSizeCap
// 回归测试:通知数超过 list 端点 MaxPageSize(100) 时,GetNoticeStatusStatistics 必须返回
// 真实总数与各发布状态计数,而不是被分页上限截断。同时锁定正确的状态桶语义:
// publishStatus 0=草稿 1=已发布 2=定时发布中 3=已撤回(models.PublishStatus)。
func TestNoticeStatusStatistics_CountExceedsPageSizeCap(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "create sqlite test db")

	require.NoError(t, db.Exec(`
		CREATE TABLE sys_notice (
			id TEXT PRIMARY KEY,
			notice_title TEXT NOT NULL,
			notice_content TEXT NOT NULL,
			status INTEGER DEFAULT 0,
			publish_status INTEGER DEFAULT 0,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)
	`).Error)

	now := "2024-01-01 00:00:00"
	insert := func(prefix string, n int, publishStatus int) {
		for i := 0; i < n; i++ {
			require.NoError(t, db.Exec(
				`INSERT INTO sys_notice (id, notice_title, notice_content, status, publish_status, created_at, updated_at, deleted_at) VALUES (?, ?, 'c', 0, ?, ?, ?, NULL)`,
				fmt.Sprintf("%s-%d", prefix, i), fmt.Sprintf("%s%d", prefix, i), publishStatus, now, now,
			).Error)
		}
	}
	// 60 已发布(1) + 40 草稿(0) + 30 定时发布中(2) + 20 已撤回(3) = 150,远超 MaxPageSize=100
	insert("published", 60, 1)
	insert("draft", 40, 0)
	insert("scheduled", 30, 2)
	insert("withdrawn", 20, 3)
	// 1 条软删除,必须排除
	require.NoError(t, db.Exec(
		`INSERT INTO sys_notice (id, notice_title, notice_content, status, publish_status, created_at, updated_at, deleted_at) VALUES ('ghost','g','c',0,0,?,?,?)`,
		now, now, now,
	).Error)

	svc := &NoticeService{db: db}

	result, err := svc.GetNoticeStatusStatistics(context.Background())
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, int64(150), result.Total, "total 必须是真实计数 150,而非被 pageSize 上限截断")
	require.Equal(t, int64(60), result.Published, "published 必须为 publishStatus=1 的计数")
	require.Equal(t, int64(40), result.Draft, "draft 必须为 publishStatus=0 的计数(不含定时发布)")
	require.Equal(t, int64(30), result.Scheduled, "scheduled 必须为 publishStatus=2 的计数(旧前端错误地用 3=已撤回)")
}
