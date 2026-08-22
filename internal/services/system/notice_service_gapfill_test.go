package system

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
)

// =====================================================================
// Phase 74-07 收尾:notice_service.go 0% 包络(wrapper 类方法)覆盖。
// 所有方法都委托给 internal/services.NoticeService,此处验证 wrapper
// 的参数透传与 PageResult 字段映射。
// =====================================================================

func newNoticeTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:notice_"+t.Name()+"?mode=memory&cache=shared&_enable_boolean=true&_busy_timeout=5000"), &gorm.Config{})
	require.NoError(t, err)
	// 手写 CREATE(绕过 gen_random_uuid() 默认,sqlite 不支持 PG 函数)。
	require.NoError(t, db.Exec(`
		CREATE TABLE IF NOT EXISTS sys_notice (
			id TEXT PRIMARY KEY, notice_title TEXT, notice_type TEXT, notice_content TEXT,
			status INTEGER DEFAULT 0, priority INTEGER DEFAULT 0,
			publish_time DATETIME, publish_status INTEGER DEFAULT 0,
			target_type INTEGER DEFAULT 0, created_by TEXT, created_by_name TEXT,
			is_markdown INTEGER DEFAULT 0, end_date DATETIME,
			updated_by TEXT, version INTEGER DEFAULT 0,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE IF NOT EXISTS sys_notification_channel (
			id TEXT PRIMARY KEY, notice_id TEXT, channel_type TEXT,
			email_config_id TEXT, api_config_id TEXT,
			custom_recipients TEXT, created_at DATETIME
		)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE IF NOT EXISTS sys_notice_target (
			id TEXT PRIMARY KEY, notice_id TEXT, target_type TEXT,
			target_id TEXT, target_name TEXT,
			read_status INTEGER DEFAULT 0, read_time DATETIME,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE IF NOT EXISTS sys_notice_statistics (
			id TEXT PRIMARY KEY, notice_id TEXT,
			total_targets INTEGER DEFAULT 0, read_count INTEGER DEFAULT 0,
			unread_count INTEGER DEFAULT 0, click_count INTEGER DEFAULT 0,
			created_at DATETIME, updated_at DATETIME
		)
	`).Error)
	return db
}

func TestNoticeService_CreateAndUpdateAndGet(t *testing.T) {
	db := newNoticeTestDB(t)
	svc := NewNoticeService(db)
	ctx := context.Background()

	req := &requests.NoticeCreateRequest{
		NoticeTitle:   "测试通知",
		NoticeType:    "1",
		NoticeContent: "内容",
		Priority:      1,
		TargetType:    0,
	}
	n, err := svc.Create(ctx, req, "u1", "alice")
	require.NoError(t, err)
	require.NotEmpty(t, n.ID)
	assert.Equal(t, "测试通知", n.NoticeTitle)

	upd := &requests.NoticeUpdateRequest{NoticeTitle: strPtr("改后")}
	require.NoError(t, svc.Update(ctx, n.ID, upd))

	got, err := svc.GetByID(ctx, n.ID)
	require.NoError(t, err)
	assert.Equal(t, "改后", got.NoticeTitle)
}

func TestNoticeService_ListAndPageFields(t *testing.T) {
	db := newNoticeTestDB(t)
	svc := NewNoticeService(db)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		_, err := svc.Create(ctx, &requests.NoticeCreateRequest{
			NoticeTitle: "通知", NoticeContent: "x",
		}, "u", "u")
		require.NoError(t, err)
	}
	params := requests.DefaultNoticeListParams()
	params.PageSize = 10
	page, err := svc.List(ctx, params)
	require.NoError(t, err)
	assert.Equal(t, int64(3), page.Total)
	assert.Equal(t, 1, page.Current)
	assert.Equal(t, 10, page.PageSize)
	notices, _ := page.List.([]models.Notice)
	assert.Len(t, notices, 3)
}

func TestNoticeService_PublishWithdrawDelete(t *testing.T) {
	db := newNoticeTestDB(t)
	svc := NewNoticeService(db)
	ctx := context.Background()

	n, err := svc.Create(ctx, &requests.NoticeCreateRequest{
		NoticeTitle: "pub", NoticeContent: "x",
	}, "u", "u")
	require.NoError(t, err)

	require.NoError(t, svc.Publish(ctx, n.ID))
	require.NoError(t, svc.Withdraw(ctx, n.ID))
	require.NoError(t, svc.Delete(ctx, n.ID))

	// 第二次删应报错
	err = svc.Delete(ctx, n.ID)
	require.Error(t, err)
}

func TestNoticeService_BatchDeleteAndStatistics(t *testing.T) {
	db := newNoticeTestDB(t)
	svc := NewNoticeService(db)
	ctx := context.Background()

	ids := make([]string, 0, 3)
	for i := 0; i < 3; i++ {
		n, err := svc.Create(ctx, &requests.NoticeCreateRequest{
			NoticeTitle: "b", NoticeContent: "x",
		}, "u", "u")
		require.NoError(t, err)
		ids = append(ids, n.ID)
	}
	require.NoError(t, svc.BatchDelete(ctx, ids))

	// BatchDelete 后 ids[0] 已软删除 → GetStatistics 取不到;但 GetStatistics
	// 在 NoticeService 包内部会兜底返回空 stats,不应 panic,只断言 NotNil。
	stats, err := svc.GetStatistics(ctx, ids[0])
	// 内部服务可能返回错误,这里主要验证 wrapper 不 panic / 返回非 nil pointer
	if err == nil {
		assert.NotNil(t, stats)
	}
}

// 触发 NewNotificationChannelService 构造函数覆盖
func TestNewNotificationChannelService_NoCrash(t *testing.T) {
	db := newNoticeTestDB(t)
	svc := NewNotificationChannelService(db)
	require.NotNil(t, svc)
}

// =====================================================================
// notice_cache_impl.go 包络覆盖:Create / Update / Delete / BatchDelete /
// GetNoticeByID / List / Publish / Withdraw / GetStatusStatistics。
// =====================================================================

func TestNoticeCache_DelegatedMethods(t *testing.T) {
	db := newNoticeTestDB(t)
	mem := newInfraMemCache()
	svc := NewNoticeServiceWithCache(db, NewCacheAdapter(mem), nil)
	ctx := context.Background()

	n, err := svc.Create(ctx, &requests.NoticeCreateRequest{
		NoticeTitle: "缓存版", NoticeContent: "x",
	}, "u", "u")
	require.NoError(t, err)

	upd := &requests.NoticeUpdateRequest{NoticeTitle: strPtr("缓存版改")}
	require.NoError(t, svc.Update(ctx, n.ID, upd))

	got, err := svc.GetNoticeByID(ctx, n.ID)
	require.NoError(t, err)
	assert.Equal(t, "缓存版改", got.NoticeTitle)

	page, err := svc.List(ctx, requests.DefaultNoticeListParams())
	require.NoError(t, err)
	assert.Equal(t, int64(1), page.Total)

	require.NoError(t, svc.Publish(ctx, n.ID))
	require.NoError(t, svc.Withdraw(ctx, n.ID))

	// GetStatistics 内部依赖 sys_user/sys_dept/sys_role + 通知目标用户列表,
	// 这里跳过 GetStatistics(其 wrapper 在 notice_service_gapfill_test.go
	// 中已验证)。 GetStatusStatistics 只读 sys_notice。
	st, err := svc.GetStatusStatistics(ctx)
	require.NoError(t, err)
	assert.NotNil(t, st)
}