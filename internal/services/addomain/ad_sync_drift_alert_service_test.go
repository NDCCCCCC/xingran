package addomain

// Phase 134 / FIX-11：AD 同步漂移常态化告警服务 sqlite 用例。
//
// 覆盖：
//  1. detectADDrift 纯函数判定表（首次观测 / 持平 / 下降 / 上涨 / 容差 / 恢复）；
//  2. 首次观测只记基线、不告警（三通道零产出）；
//  3. 漂移触发三通道：applogger WARN + sys_oper_log 触发记录 + sys_notice 已发布通知；
//  4. 节流：同一持续漂移 1h 内只通知 1 次（不成告警风暴）；恢复后再漂移立即触发；
//  5. 可配置：enabled=false 关闭 / tolerance 容差 / notice_enabled=false 只写 operlog；
//  6. 软失败：表缺失 / db=nil 不 panic、不返回 error（cron 零回归）；
//  7. 自适应基线：健康轮次把基线更新为当前计数。
//
// 复用既有设施：models.ADGroupMember / models.Config / models.OperLog / models.Notice
// 均用 GORM AutoMigrate 建表（sqlite 内存库），与项目双方言约定一致。

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// ==================== 1. 纯函数判定表 ====================

func TestPhase134_DetectADDrift(t *testing.T) {
	base := adDriftSample{MemberCount: 100, DistinctGroups: 10}
	tests := []struct {
		name            string
		cur             adDriftSample
		exists          bool
		tolerance       int
		wantDrift       bool
		wantFirst       bool
		wantMemberDelta int64
	}{
		{"无基线首次观测不告警", base, false, 0, false, true, 0},
		{"完全持平不告警", base, true, 0, false, false, 0},
		{"成员数下降→漂移", adDriftSample{95, 10}, true, 0, true, false, -5},
		{"组数下降→漂移", adDriftSample{100, 9}, true, 0, true, false, 0},
		{"成员数上涨→漂移(抖动)", adDriftSample{110, 10}, true, 0, true, false, 10},
		{"容差内下降不告警", adDriftSample{99, 10}, true, 1, false, false, -1},
		{"容差边界等于容差不告警", adDriftSample{98, 10}, true, 2, false, false, -2},
		{"超过容差告警", adDriftSample{97, 10}, true, 2, true, false, -3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := detectADDrift(tt.cur, base, tt.exists, tt.tolerance)
			assert.Equal(t, tt.wantDrift, v.Drift)
			assert.Equal(t, tt.wantFirst, v.FirstSample)
			assert.Equal(t, tt.wantMemberDelta, v.MemberDelta)
		})
	}
}

// ==================== 测试基座 ====================

// setupDriftAlertDB 建 4 张表：sys_ad_group_member / sys_config / sys_oper_log / sys_notice。
func setupDriftAlertDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&models.ADGroupMember{}, &models.Config{}, &models.OperLog{}, &models.Notice{},
	))
	return db
}

// seedDriftMembers 铺 n 个成员行，分布在 groupCount 个不同 group_dn 下。
func seedDriftMembers(t *testing.T, db *gorm.DB, n, groupCount int) {
	t.Helper()
	rows := make([]models.ADGroupMember, 0, n)
	for i := 0; i < n; i++ {
		rows = append(rows, models.ADGroupMember{
			ADConfigID: "cfg-1",
			GroupDN:    "CN=group-" + strconv.Itoa(i%groupCount) + ",DC=corp,DC=com",
			UserDN:     "CN=user-" + strconv.Itoa(i) + ",DC=corp,DC=com",
		})
	}
	if len(rows) > 0 {
		require.NoError(t, db.CreateInBatches(&rows, 100).Error)
	}
}

func seedDriftConfig(t *testing.T, db *gorm.DB, key, value string) {
	t.Helper()
	require.NoError(t, db.Create(&models.Config{ConfigKey: key, ConfigValue: value, ConfigName: "FIX-11 告警旋钮"}).Error)
}

func countOperLogs(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Model(&models.OperLog{}).Where("method = ?", adDriftAlertMethod).Count(&n).Error)
	return n
}

func countDriftNotices(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Model(&models.Notice{}).Where("notice_title = ?", adDriftAlertTitle).Count(&n).Error)
	return n
}

// ==================== 2. 首次观测 ====================

func TestPhase134_DriftAlert_FirstSampleOnlyRecordsBaseline(t *testing.T) {
	db := setupDriftAlertDB(t)
	seedDriftMembers(t, db, 20, 4)
	svc := NewADSyncDriftAlertService(db)

	require.NoError(t, svc.CheckAndAlert(context.Background()))

	assert.Equal(t, 0, svc.AlertCount(), "首次观测只记基线，不得告警")
	assert.Equal(t, int64(0), countOperLogs(t, db))
	assert.Equal(t, int64(0), countDriftNotices(t, db))

	// 基线确实落 sys_config
	var v string
	require.NoError(t, db.Raw("SELECT config_value FROM sys_config WHERE config_key = ?", adDriftBaselineConfigKey).Scan(&v).Error)
	assert.Equal(t, "20,4", v)

	// 基线必须能被 UPDATE 更新（FirstOrCreate 变体曾撞 UNIQUE 2067 → 基线永不更新、
	// 后续每轮重复误告警，被软失败 WARN 掩盖）。二次写入同 key 不得重复 INSERT。
	svc.writeBaseline(context.Background(), adDriftSample{MemberCount: 22, DistinctGroups: 5})
	require.NoError(t, db.Raw("SELECT config_value FROM sys_config WHERE config_key = ?", adDriftBaselineConfigKey).Scan(&v).Error)
	assert.Equal(t, "22,5", v, "基线必须能被 UPDATE 更新")
	var rowCount int64
	require.NoError(t, db.Model(&models.Config{}).Where("config_key = ?", adDriftBaselineConfigKey).Count(&rowCount).Error)
	assert.Equal(t, int64(1), rowCount, "基线行必须唯一，不得重复 INSERT")
}

// ==================== 3. 漂移触发三通道 ====================

func TestPhase134_DriftAlert_DriftEmitsThreeChannels(t *testing.T) {
	db := setupDriftAlertDB(t)
	seedDriftMembers(t, db, 20, 4)
	svc := NewADSyncDriftAlertService(db)
	ctx := context.Background()

	require.NoError(t, svc.CheckAndAlert(ctx)) // 首次观测，基线 20,4
	// 模拟同步漂移：成员关系丢失
	require.NoError(t, db.Where("1 = 1").Delete(&models.ADGroupMember{}).Error)
	seedDriftMembers(t, db, 15, 3)

	require.NoError(t, svc.CheckAndAlert(ctx)) // 15,3 → 漂移

	assert.Equal(t, 1, svc.AlertCount(), "必须触发一次告警")
	assert.Equal(t, int64(1), countOperLogs(t, db), "通道2：sys_oper_log 必须有触发记录")
	assert.Equal(t, int64(1), countDriftNotices(t, db), "通道3：sys_notice 必须发布一条告警通知")

	// operlog 内容核对（BusinessType=OperTypeSync=14 / JsonResult 含诊断字段）
	var log models.OperLog
	require.NoError(t, db.Where("method = ?", adDriftAlertMethod).First(&log).Error)
	assert.Equal(t, adDriftAlertOperBusinessType, log.BusinessType)
	require.NotNil(t, log.JsonResult)
	assert.Contains(t, *log.JsonResult, `"memberCount":15`)
	assert.Contains(t, *log.JsonResult, `"memberDelta":-5`)

	// notice 内容核对（已发布 / 警告 / 紧急 / 全员）
	var notice models.Notice
	require.NoError(t, db.Where("notice_title = ?", adDriftAlertTitle).First(&notice).Error)
	assert.Equal(t, models.PublishStatusPublished, notice.PublishStatus)
	assert.Equal(t, adDriftAlertNoticeType, notice.NoticeType)
	assert.Equal(t, models.PriorityUrgent, notice.Priority)
	assert.Equal(t, models.TargetAll, notice.TargetType)
	assert.Contains(t, notice.NoticeContent, adDriftAlertContentPrefix)
}

// ==================== 4. 节流（不成告警风暴） ====================

func TestPhase134_DriftAlert_ThrottlesRepeatedDrift(t *testing.T) {
	db := setupDriftAlertDB(t)
	seedDriftMembers(t, db, 20, 4)
	svc := NewADSyncDriftAlertService(db)
	ctx := context.Background()
	require.NoError(t, svc.CheckAndAlert(ctx)) // 基线 20,4

	require.NoError(t, db.Where("1 = 1").Delete(&models.ADGroupMember{}).Error)
	seedDriftMembers(t, db, 15, 3)

	// 连续 10 轮检测（同一持续漂移）→ 只应通知 1 次（默认节流 60min）
	for i := 0; i < 10; i++ {
		require.NoError(t, svc.CheckAndAlert(ctx))
	}
	assert.Equal(t, 1, svc.AlertCount(), "同一持续漂移 1h 内最多 1 次告警（防告警风暴）")
	assert.Equal(t, int64(1), countOperLogs(t, db))
	assert.Equal(t, int64(1), countDriftNotices(t, db))
}

func TestPhase134_DriftAlert_RecoveryThenNewDriftAlertsImmediately(t *testing.T) {
	db := setupDriftAlertDB(t)
	seedDriftMembers(t, db, 20, 4)
	svc := NewADSyncDriftAlertService(db)
	ctx := context.Background()
	require.NoError(t, svc.CheckAndAlert(ctx)) // 基线 20,4

	// 漂移 1
	require.NoError(t, db.Where("1 = 1").Delete(&models.ADGroupMember{}).Error)
	seedDriftMembers(t, db, 15, 3)
	require.NoError(t, svc.CheckAndAlert(ctx))
	assert.Equal(t, 1, svc.AlertCount())

	// 恢复到基线（20,4）→ 健康轮次清节流标记 + 基线自适应
	require.NoError(t, db.Where("1 = 1").Delete(&models.ADGroupMember{}).Error)
	seedDriftMembers(t, db, 20, 4)
	require.NoError(t, svc.CheckAndAlert(ctx))
	assert.Equal(t, 1, svc.AlertCount(), "恢复轮次不得告警")

	// 再漂移 → 状态变化，立即触发新告警
	require.NoError(t, db.Where("1 = 1").Delete(&models.ADGroupMember{}).Error)
	seedDriftMembers(t, db, 12, 2)
	require.NoError(t, svc.CheckAndAlert(ctx))
	assert.Equal(t, 2, svc.AlertCount(), "恢复后的新漂移必须立即触发（状态变化）")
	assert.Equal(t, int64(2), countOperLogs(t, db))
}

// ==================== 5. 可配置 ====================

func TestPhase134_DriftAlert_DisabledSkips(t *testing.T) {
	db := setupDriftAlertDB(t)
	seedDriftConfig(t, db, adDriftKeyEnabled, "false")
	seedDriftMembers(t, db, 20, 4)
	svc := NewADSyncDriftAlertService(db)
	ctx := context.Background()

	require.NoError(t, svc.CheckAndAlert(ctx))
	require.NoError(t, db.Where("1 = 1").Delete(&models.ADGroupMember{}).Error)
	seedDriftMembers(t, db, 15, 3)
	require.NoError(t, svc.CheckAndAlert(ctx))

	assert.Equal(t, 0, svc.AlertCount(), "enabled=false 必须整通道关闭")
	assert.Equal(t, int64(0), countOperLogs(t, db))
}

func TestPhase134_DriftAlert_ToleranceSuppressesNoise(t *testing.T) {
	db := setupDriftAlertDB(t)
	seedDriftConfig(t, db, adDriftKeyTolerance, "5")
	seedDriftMembers(t, db, 20, 4)
	svc := NewADSyncDriftAlertService(db)
	ctx := context.Background()

	require.NoError(t, svc.CheckAndAlert(ctx)) // 基线 20,4
	require.NoError(t, db.Where("1 = 1").Delete(&models.ADGroupMember{}).Error)
	seedDriftMembers(t, db, 18, 4) // Δ=-2，容差 5 内
	require.NoError(t, svc.CheckAndAlert(ctx))

	assert.Equal(t, 0, svc.AlertCount(), "容差内的抖动不得告警（降噪）")
}

func TestPhase134_DriftAlert_NoticeDisabledStillWritesOperLog(t *testing.T) {
	db := setupDriftAlertDB(t)
	seedDriftConfig(t, db, adDriftKeyNoticeEnabled, "false")
	seedDriftMembers(t, db, 20, 4)
	svc := NewADSyncDriftAlertService(db)
	ctx := context.Background()

	require.NoError(t, svc.CheckAndAlert(ctx))
	require.NoError(t, db.Where("1 = 1").Delete(&models.ADGroupMember{}).Error)
	seedDriftMembers(t, db, 15, 3)
	require.NoError(t, svc.CheckAndAlert(ctx))

	assert.Equal(t, 1, svc.AlertCount())
	assert.Equal(t, int64(1), countOperLogs(t, db), "operlog 触发记录始终要有（审计不变量）")
	assert.Equal(t, int64(0), countDriftNotices(t, db), "notice_enabled=false 时不写通知")
}

func TestPhase134_DriftAlert_CustomThrottle(t *testing.T) {
	db := setupDriftAlertDB(t)
	seedDriftConfig(t, db, adDriftKeyThrottle, "1") // 1 分钟
	seedDriftMembers(t, db, 20, 4)
	baseTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := baseTime
	svc := NewADSyncDriftAlertService(db)
	svc.now = func() time.Time { return clock }
	ctx := context.Background()

	require.NoError(t, svc.CheckAndAlert(ctx)) // 基线
	require.NoError(t, db.Where("1 = 1").Delete(&models.ADGroupMember{}).Error)
	seedDriftMembers(t, db, 15, 3)

	require.NoError(t, svc.CheckAndAlert(ctx)) // 第 1 次漂移 → 告警
	assert.Equal(t, 1, svc.AlertCount())
	clock = baseTime.Add(30 * time.Second)
	require.NoError(t, svc.CheckAndAlert(ctx)) // 节流内
	assert.Equal(t, 1, svc.AlertCount())
	clock = baseTime.Add(61 * time.Second)
	require.NoError(t, svc.CheckAndAlert(ctx)) // 超过 1min → 再次告警
	assert.Equal(t, 2, svc.AlertCount())
}

// ==================== 6. 软失败（cron 零回归） ====================

func TestPhase134_DriftAlert_SoftFailures(t *testing.T) {
	// db=nil → 不 panic、不报错
	require.NoError(t, NewADSyncDriftAlertService(nil).CheckAndAlert(context.Background()))

	// 表全缺失（sqlite 空库）→ 采样返回 error 但不抛错、不 panic
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, NewADSyncDriftAlertService(db).CheckAndAlert(context.Background()))
	assert.Equal(t, 0, NewADSyncDriftAlertService(db).AlertCount())
}

func TestPhase134_DriftAlert_ConfigFallback(t *testing.T) {
	// 表缺失 / 非法值 → 回落默认，不 panic
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	assert.Equal(t, defaultADDriftConfig(), NewADSyncDriftAlertService(db).loadConfig())

	db2 := setupDriftAlertDB(t)
	seedDriftConfig(t, db2, adDriftKeyTolerance, "not-a-number")
	seedDriftConfig(t, db2, adDriftKeyThrottle, "-1")
	seedDriftConfig(t, db2, adDriftKeyEnabled, "maybe")
	assert.Equal(t, defaultADDriftConfig(), NewADSyncDriftAlertService(db2).loadConfig())
}
