package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestKnowledgeService_GetArticleStatistics 验证知识库文章统计:
// 1) 条件聚合正确统计 total/draft(status=0)/published(status=1);
// 2) COALESCE(SUM(view_count/like_count), 0) 正确累计且空集不返回 NULL;
// 3) 软删除文章被排除。
func TestKnowledgeService_GetArticleStatistics(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "create sqlite test db")

	require.NoError(t, db.Exec(`
		CREATE TABLE sys_knowledge_article (
			id TEXT PRIMARY KEY,
			title TEXT,
			status INTEGER DEFAULT 0,
			view_count INTEGER DEFAULT 0,
			like_count INTEGER DEFAULT 0,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)
	`).Error)

	now := "2024-01-01 00:00:00"
	// 80 已发布(views 10, likes 2) + 50 草稿(views 3, likes 0) = 130 篇
	for i := 0; i < 80; i++ {
		require.NoError(t, db.Exec(
			`INSERT INTO sys_knowledge_article (id, title, status, view_count, like_count, created_at, updated_at, deleted_at) VALUES (?, ?, 1, 10, 2, ?, ?, NULL)`,
			fmt.Sprintf("p-%d", i), fmt.Sprintf("p%d", i), now, now,
		).Error)
	}
	for i := 0; i < 50; i++ {
		require.NoError(t, db.Exec(
			`INSERT INTO sys_knowledge_article (id, title, status, view_count, like_count, created_at, updated_at, deleted_at) VALUES (?, ?, 0, 3, 0, ?, ?, NULL)`,
			fmt.Sprintf("d-%d", i), fmt.Sprintf("d%d", i), now, now,
		).Error)
	}
	// 1 篇软删除,必须排除
	require.NoError(t, db.Exec(
		`INSERT INTO sys_knowledge_article (id, title, status, view_count, like_count, created_at, updated_at, deleted_at) VALUES ('ghost', 'g', 1, 9999, 9999, ?, ?, ?)`,
		now, now, now,
	).Error)

	svc := &KnowledgeService{db: db}
	result, err := svc.GetArticleStatistics(context.Background())
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, int64(130), result.Total, "total 必须排除软删除")
	require.Equal(t, int64(50), result.Draft, "draft = status=0 的文章数")
	require.Equal(t, int64(80), result.Published, "published = status=1 的文章数")
	// views: 80*10 + 50*3 = 950; likes: 80*2 + 50*0 = 160; ghost 的 9999 必须排除
	require.Equal(t, int64(950), result.TotalViews, "totalViews 必须排除软删除文章的浏览数")
	require.Equal(t, int64(160), result.TotalLikes, "totalLikes 必须排除软删除文章的点赞数")
}
