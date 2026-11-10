package addomain

// Phase 144 · 漂移告警 query 层单测（sqlite 内存库，零真机/生产依赖）。
//
// 覆盖：GetStatus 字段 / GetConfig 默认值与覆盖 / UpdateConfig 白名单与幂等
// / ListTriggerLogs 分页（list/total/current/pageSize，current 从 1）
// / TriggerOnce 软失败。

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

func setupDriftQueryDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.Config{}, &models.OperLog{}, &models.Notice{}, &models.ADGroupMember{}))
	return db
}

func TestPhase144_GetStatus_DefaultsAndConfig(t *testing.T) {
	db := setupDriftQueryDB(t)
	svc := NewADSyncDriftAlertService(db)
	ctx := context.Background()

	st, err := svc.GetStatus(ctx)
	require.NoError(t, err)
	assert.True(t, st.Enabled)
	assert.True(t, st.NoticeEnabled)
	assert.True(t, st.WSEnabled, "ws_enabled 默认 true")
	assert.False(t, st.WebhookEnabled, "webhook_enabled 默认 false")
	assert.Equal(t, 60, st.ThrottleMinutes)
	assert.Equal(t, 0, st.Tolerance)
	assert.Equal(t, 0, st.AlertCount)
	assert.False(t, st.LastAlertDrifted)
	assert.Equal(t, adDriftBaselineConfigKey, st.BaselineKey)

	// 覆盖配置（含 3 个新旋钮）
	seedDriftConfig(t, db, adDriftKeyWSEnabled, "false")
	seedDriftConfig(t, db, adDriftKeyWebhookEnabled, "true")
	seedDriftConfig(t, db, adDriftKeyWebhookConfigID, "cfg-abc")
	seedDriftConfig(t, db, adDriftKeyTolerance, "7")
	st2, err := svc.GetStatus(ctx)
	require.NoError(t, err)
	assert.False(t, st2.WSEnabled)
	assert.True(t, st2.WebhookEnabled)
	assert.Equal(t, "cfg-abc", st2.WebhookConfigID)
	assert.Equal(t, 7, st2.Tolerance)
}

func TestPhase144_GetConfig_DefaultsAndOverride(t *testing.T) {
	db := setupDriftQueryDB(t)
	svc := NewADSyncDriftAlertService(db)
	ctx := context.Background()

	view, err := svc.GetConfig(ctx)
	require.NoError(t, err)
	require.Len(t, view, 9, "必须暴露 9 个 sys_config 旋钮")
	assert.Equal(t, "true", view[adDriftKeyEnabled])
	assert.Equal(t, "false", view[adDriftKeyWebhookEnabled])
	assert.Equal(t, "true", view[adDriftKeyWSEnabled])

	// 库里有值 → 覆盖默认
	seedDriftConfig(t, db, adDriftKeyWebhookEnabled, "true")
	view2, err := svc.GetConfig(ctx)
	require.NoError(t, err)
	assert.Equal(t, "true", view2[adDriftKeyWebhookEnabled])
}

func TestPhase144_UpdateConfig_WhitelistAndIdempotent(t *testing.T) {
	db := setupDriftQueryDB(t)
	svc := NewADSyncDriftAlertService(db)
	ctx := context.Background()

	err := svc.UpdateConfig(ctx, map[string]string{
		adDriftKeyWSEnabled:       "false",
		adDriftKeyWebhookEnabled:  "true",
		adDriftKeyWebhookConfigID: "cfg-xyz",
		"arbitrary_key":           "must-be-ignored", // 白名单外
		"webhookUrl":              "https://hook?key=SECRET",
	})
	require.NoError(t, err)

	cfg := svc.loadConfig()
	assert.False(t, cfg.WSEnabled)
	assert.True(t, cfg.WebhookEnabled)
	assert.Equal(t, "cfg-xyz", cfg.WebhookConfigID)

	// 白名单外 key 绝不落 sys_config
	var n int64
	require.NoError(t, db.Model(&models.Config{}).Where("config_key = ?", "arbitrary_key").Count(&n).Error)
	assert.Equal(t, int64(0), n)

	// 再更新一次 → 幂等（不重复 INSERT，值被 UPDATE）
	require.NoError(t, svc.UpdateConfig(ctx, map[string]string{adDriftKeyWSEnabled: "true"}))
	cfg2 := svc.loadConfig()
	assert.True(t, cfg2.WSEnabled)
	var rows int64
	require.NoError(t, db.Model(&models.Config{}).Where("config_key = ?", adDriftKeyWSEnabled).Count(&rows).Error)
	assert.Equal(t, int64(1), rows, "同一 key 不得重复插入")
}

func TestPhase144_ListTriggerLogs_Pagination(t *testing.T) {
	db := setupDriftQueryDB(t)
	svc := NewADSyncDriftAlertService(db)
	ctx := context.Background()

	// 铺 5 条触发记录（method=CheckAndAlert）+ 2 条其它 method 干扰
	for i := 0; i < 5; i++ {
		require.NoError(t, db.Create(&models.OperLog{
			Title: "AD 同步漂移告警", Method: adDriftAlertMethod,
			BusinessType: adDriftAlertOperBusinessType,
		}).Error)
	}
	for i := 0; i < 2; i++ {
		require.NoError(t, db.Create(&models.OperLog{Title: "其它", Method: "Other.Method"}).Error)
	}

	p1, err := svc.ListTriggerLogs(ctx, 1, 2, "")
	require.NoError(t, err)
	assert.Equal(t, int64(5), p1.Total, "只统计 CheckAndAlert 记录")
	assert.Equal(t, 1, p1.Current)
	assert.Equal(t, 2, p1.PageSize)
	assert.Len(t, p1.List, 2)

	p3, err := svc.ListTriggerLogs(ctx, 3, 2, "")
	require.NoError(t, err)
	assert.Equal(t, 3, p3.Current)
	assert.Len(t, p3.List, 1, "第 3 页只剩 1 条")

	// current<1 归一到 1；pageSize>200 截断
	p0, err := svc.ListTriggerLogs(ctx, 0, 999, "")
	require.NoError(t, err)
	assert.Equal(t, 1, p0.Current)
	assert.Equal(t, 200, p0.PageSize)
}

// B-1 / quick 261110-161：JsonResult 里的 noticeId 必须被解析到 NoticeID，
// 且按 userID 查 sys_notice_read 补 IsRead（按用户隔离，复用既有已读设施）。
func TestB1_ListTriggerLogs_NoticeIDAndIsRead(t *testing.T) {
	db := setupDriftQueryDB(t)
	// 用真实模型 AutoMigrate，列类型与生产一致（uuid 文本主键）
	require.NoError(t, db.AutoMigrate(&models.NoticeRead{}))
	svc := NewADSyncDriftAlertService(db)
	ctx := context.Background()

	// 两条触发记录：一条带 noticeId="n-1"，一条旧格式无 noticeId
	jr1 := `{"memberCount":10,"noticeId":"n-1"}`
	jr2 := `{"memberCount":11}`
	require.NoError(t, db.Create(&models.OperLog{
		Title: "AD 同步漂移告警", Method: adDriftAlertMethod, JsonResult: &jr1,
	}).Error)
	require.NoError(t, db.Create(&models.OperLog{
		Title: "AD 同步漂移告警", Method: adDriftAlertMethod, JsonResult: &jr2,
	}).Error)
	// 只给 user-u1 标记 n-1 已读
	require.NoError(t, db.Exec(
		`INSERT INTO sys_notice_read (notice_id, user_id) VALUES ('n-1', 'user-u1')`).Error)

	// 未传 userID：IsRead 全 false，但 NoticeID 仍解析
	page0, err := svc.ListTriggerLogs(ctx, 1, 10, "")
	require.NoError(t, err)
	require.Len(t, page0.List, 2)
	byNotice := map[string]bool{}
	for _, r := range page0.List {
		byNotice[r.NoticeID] = r.IsRead
	}
	assert.Contains(t, byNotice, "n-1")
	assert.False(t, byNotice["n-1"], "无 userID 不得查已读")

	// user-u1：n-1 已读；user-u2：未读（按用户隔离）
	p1, err := svc.ListTriggerLogs(ctx, 1, 10, "user-u1")
	require.NoError(t, err)
	for _, r := range p1.List {
		if r.NoticeID == "n-1" {
			assert.True(t, r.IsRead, "user-u1 已读 n-1")
		}
	}
	p2, err := svc.ListTriggerLogs(ctx, 1, 10, "user-u2")
	require.NoError(t, err)
	for _, r := range p2.List {
		if r.NoticeID == "n-1" {
			assert.False(t, r.IsRead, "user-u2 未读 n-1")
		}
	}
}

func TestPhase144_TriggerOnce_SoftFail(t *testing.T) {
	// 空表库：采样失败也应返回 nil（不阻断 cron / 端点）
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	svc := NewADSyncDriftAlertService(db)
	require.NoError(t, svc.TriggerOnce(context.Background()))
	assert.Equal(t, 0, svc.AlertCount())
}
