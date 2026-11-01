package addomain

// DEBT-01（v1.51）第三批：AD→sys_user 状态联动 + 登录历史写入/分区服务 行为测试。
//
// 覆盖目标：
//  1. matchSysUserIDsByADDN：DN 归一化匹配、ad_config_id 归属防护（跨配置不误伤）、
//     软删行排除、列探测降级路径
//  2. disableSysUsersForADDNs：只停用启用行（幂等）、软性状态变更（不删行）、
//     绕过入队的单向收敛语义
//  3. ADLoginHistoryWriter.Flush：空事件短路、非 PG 守卫短路、
//     flushLoginEvents 的 computerKey 解析与未解析跳过
//  4. AD 登录历史分区服务：SQLite 方言守卫整体短路 + GetRetentionDays
//     配置读取（默认值 / 合法值 / 低于下限回退 / 非法值回退）

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

func setupLinkageTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.User{}, &models.Config{}, &models.ADLoginHistory{}))
	return db
}

func TestMatchSysUserIDsByADDN(t *testing.T) {
	db := setupLinkageTestDB(t)
	ctx := context.Background()
	config := &models.ADConfig{BaseModel: models.BaseModel{ID: "cfg-1"}}

	users := []models.User{
		// 命中：ad_config_id 归属 cfg-1，DN 大小写/空格差异靠归一化吸收
		{BaseModel: models.BaseModel{ID: "u-hit"}, Username: "hit", AdDn: strPtr("CN=Alice,DC=Test"), AdConfigID: strPtr("cfg-1")},
		// 跨配置防护：归属别的配置 → 不命中
		{BaseModel: models.BaseModel{ID: "u-other"}, Username: "other", AdDn: strPtr("CN=bob,DC=test"), AdConfigID: strPtr("cfg-2")},
		// 兜底：ad_config_id 为空 → 仅按 DN 匹配
		{BaseModel: models.BaseModel{ID: "u-legacy"}, Username: "legacy", AdDn: strPtr("CN=carol,DC=test")},
		// 无 ad_dn → 不命中
		{BaseModel: models.BaseModel{ID: "u-nodn"}, Username: "nodn"},
		// 软删 → 不命中（活行过滤）
		{BaseModel: models.BaseModel{ID: "u-dead", DeletedAt: gorm.DeletedAt{Time: time.Now(), Valid: true}}, Username: "dead", AdDn: strPtr("CN=dave,DC=test"), AdConfigID: strPtr("cfg-1")},
	}
	require.NoError(t, db.Create(&users).Error)

	// nil / 空入参防御
	ids, err := matchSysUserIDsByADDN(ctx, db, config, nil)
	require.NoError(t, err)
	if ids != nil {
		t.Fatalf("空 DN 列表应返回 nil, got %v", ids)
	}
	ids, err = matchSysUserIDsByADDN(ctx, db, nil, []string{"x"})
	require.NoError(t, err)
	if ids != nil {
		t.Fatalf("nil config 应返回 nil, got %v", ids)
	}

	ids, err = matchSysUserIDsByADDN(ctx, db, config, []string{"cn=ALICE, dc=test", "cn=Carol,DC=test", "cn=nobody,dc=test"})
	require.NoError(t, err)
	if len(ids) != 2 {
		t.Fatalf("应命中 2 个用户（归一化匹配 + 兜底）, got %v", ids)
	}
	for _, want := range []string{"u-hit", "u-legacy"} {
		found := false
		for _, id := range ids {
			if id == want {
				found = true
			}
		}
		if !found {
			t.Fatalf("应包含 %s, got %v", want, ids)
		}
	}
	// 列探测：AutoMigrate 的 sys_user 有 ad_config_id 列
	if !sysUserHasADConfigColumn(db) {
		t.Fatal("AutoMigrate 后应探测到 ad_config_id 列")
	}
	if sysUserHasADConfigColumn(nil) {
		t.Fatal("nil db 应返回 false")
	}
}

func TestDisableSysUsersForADDNs(t *testing.T) {
	db := setupLinkageTestDB(t)
	ctx := context.Background()
	config := &models.ADConfig{BaseModel: models.BaseModel{ID: "cfg-1"}}

	users := []models.User{
		{BaseModel: models.BaseModel{ID: "u-1"}, Username: "a", Status: models.UserStatusEnabled, AdDn: strPtr("CN=a,DC=test"), AdConfigID: strPtr("cfg-1")},
		{BaseModel: models.BaseModel{ID: "u-2"}, Username: "b", Status: models.UserStatusEnabled, AdDn: strPtr("CN=b,DC=test"), AdConfigID: strPtr("cfg-1")},
		// 已停用 → 幂等：不重复更新
		{BaseModel: models.BaseModel{ID: "u-3"}, Username: "c", Status: models.UserStatusDisabled, AdDn: strPtr("CN=c,DC=test"), AdConfigID: strPtr("cfg-1")},
	}
	require.NoError(t, db.Create(&users).Error)

	// 空入参 → 0 行
	n, err := disableSysUsersForADDNs(ctx, db, config, nil)
	require.NoError(t, err)
	if n != 0 {
		t.Fatalf("空入参应 0 行, got %d", n)
	}

	n, err = disableSysUsersForADDNs(ctx, db, config, []string{"cn=A,DC=test", "cn=C,DC=test"})
	require.NoError(t, err)
	if n != 1 {
		t.Fatalf("应恰好停用 1 行（u-3 已停用不动）, got %d", n)
	}
	// 软性停用：status 翻转但行保留、无入队副作用（u-1 停用，u-3 不变）
	var u1, u3 models.User
	require.NoError(t, db.Where("id = ?", "u-1").First(&u1).Error)
	require.NoError(t, db.Where("id = ?", "u-3").First(&u3).Error)
	if u1.Status != models.UserStatusDisabled || u1.DeletedAt.Valid {
		t.Fatalf("u-1 应软性停用（行保留）, got %+v", u1)
	}
	if u3.Status != models.UserStatusDisabled {
		t.Fatalf("u-3 应保持停用, got %+v", u3)
	}
	// 幂等：重复调用 0 行
	n, err = disableSysUsersForADDNs(ctx, db, config, []string{"cn=A,DC=test"})
	require.NoError(t, err)
	if n != 0 {
		t.Fatalf("重复调用应幂等 0 行, got %d", n)
	}
	// 反向入队循环防护：联动不得产生 ad_push_queue 行（需要表存在才可断言）
	require.NoError(t, db.AutoMigrate(&models.ADPushQueue{}))
	var qcnt int64
	require.NoError(t, db.Model(&models.ADPushQueue{}).Count(&qcnt).Error)
	if qcnt != 0 {
		t.Fatalf("AD→sys_user 联动禁止反向入队, got %d 行", qcnt)
	}
}

func TestADLoginHistoryWriter(t *testing.T) {
	db := setupLinkageTestDB(t)
	ctx := context.Background()
	w := NewADLoginHistoryWriter(db)

	// sqlite（非 PG）守卫：Flush 整体短路，不写库、不报错
	w.Flush(ctx, []pendingLoginEvent{{computerID: "c-1"}}, nil)
	var cnt int64
	require.NoError(t, db.Model(&models.ADLoginHistory{}).Count(&cnt).Error)
	if cnt != 0 {
		t.Fatalf("非 PG 应跳过写入, got %d 行", cnt)
	}
	// 空事件短路
	w.Flush(ctx, nil, nil)

	// 绕过守卫直测 flushLoginEvents：key 解析 + 未解析跳过
	now := time.Now().UTC()
	events := []pendingLoginEvent{
		{computerID: "c-1", configID: "cfg-1", computerName: "PC1", eventType: models.ADLoginEventLogin, loginTime: now, detectedAt: now, loginUser: "alice", ipAddress: "10.0.0.1"},
		{computerKey: "cfg-1/PC2", configID: "cfg-1", computerName: "PC2", eventType: models.ADLoginEventInitial, loginTime: now, detectedAt: now},
		{computerKey: "cfg-1/PC-GONE", configID: "cfg-1", computerName: "PC-GONE", eventType: models.ADLoginEventInitial, loginTime: now, detectedAt: now},
	}
	flushLoginEvents(ctx, db, events, map[string]string{"cfg-1/PC2": "c-2"})
	require.NoError(t, db.Model(&models.ADLoginHistory{}).Count(&cnt).Error)
	if cnt != 2 {
		t.Fatalf("应写入 2 条（未解析 key 跳过）, got %d", cnt)
	}
	var row models.ADLoginHistory
	require.NoError(t, db.Where("computer_id = ?", "c-2").First(&row).Error)
	if row.EventType != models.ADLoginEventInitial || row.ComputerNameSnapshot != "PC2" {
		t.Fatalf("initial 事件写入不符: %+v", row)
	}
	// 全部未解析 → 0 行写入不报错
	flushLoginEvents(ctx, db, []pendingLoginEvent{{computerKey: "x/y"}}, nil)
	require.NoError(t, db.Model(&models.ADLoginHistory{}).Count(&cnt).Error)
	if cnt != 2 {
		t.Fatalf("未解析事件不应写库, got %d", cnt)
	}
	// detectLoginEvent 四分支（回归守护）
	t1 := now
	t2 := now.Add(time.Minute)
	if _, ok := detectLoginEvent(nil, nil); ok {
		t.Fatal("双 nil 不应产生事件")
	}
	if et, ok := detectLoginEvent(nil, &t2); !ok || et != models.ADLoginEventInitial {
		t.Fatal("旧 nil 新非 nil 应为 initial")
	}
	if et, ok := detectLoginEvent(&t1, &t2); !ok || et != models.ADLoginEventLogin {
		t.Fatal("新 > 旧应为 login")
	}
	if _, ok := detectLoginEvent(&t2, &t1); ok {
		t.Fatal("新 < 旧（时钟回拨）不应产生事件")
	}
	if _, ok := detectLoginEvent(&t1, &t1); ok {
		t.Fatal("相等不应产生事件")
	}
}

func TestADLoginHistoryPartitionService_SQLiteGuard(t *testing.T) {
	db := setupLinkageTestDB(t)
	ctx := context.Background()
	svc := NewADLoginHistoryPartitionService(db)

	impl, ok := svc.(*adLoginHistoryPartitionServiceImpl)
	if !ok {
		t.Fatal("应返回私有实现")
	}
	if impl.isPostgres() {
		t.Fatal("sqlite 方言不应判定为 postgres")
	}

	// 分区 DDL 是 PG 专属：sqlite 下整体短路（nil 返回）
	require.NoError(t, svc.CreateMonthlyPartition(ctx, 2026, 10))
	require.NoError(t, svc.CreateMonthlyPartition(ctx, 1900, 13)) // 守卫短路连参数校验都不触达
	require.NoError(t, svc.EnsurePartitionsExist(ctx, 3))
	require.NoError(t, svc.EnsurePartitionsExist(ctx, 0))
	require.NoError(t, svc.DropExpiredPartitions(ctx))

	// GetRetentionDays：无配置 → 默认 120
	if got := svc.GetRetentionDays(ctx); got != defaultADLoginHistoryRetentionDays {
		t.Fatalf("无配置应返回默认 %d, got %d", defaultADLoginHistoryRetentionDays, got)
	}
	// 合法配置 → 生效
	require.NoError(t, db.Create(&models.Config{ConfigKey: ADLoginHistoryRetentionConfigKey, ConfigValue: "45"}).Error)
	if got := svc.GetRetentionDays(ctx); got != 45 {
		t.Fatalf("配置 45 应生效, got %d", got)
	}
	// 低于最小值 30 → 回退默认 + Warn
	require.NoError(t, db.Model(&models.Config{}).Where("config_key = ?", ADLoginHistoryRetentionConfigKey).Update("config_value", "10").Error)
	if got := svc.GetRetentionDays(ctx); got != defaultADLoginHistoryRetentionDays {
		t.Fatalf("低于最小值应回退默认, got %d", got)
	}
	// 非数字 → 回退默认
	require.NoError(t, db.Model(&models.Config{}).Where("config_key = ?", ADLoginHistoryRetentionConfigKey).Update("config_value", "abc").Error)
	if got := svc.GetRetentionDays(ctx); got != defaultADLoginHistoryRetentionDays {
		t.Fatalf("非数字应回退默认, got %d", got)
	}
	// 负数 → parse 成功但 days<=0 → 回退默认
	require.NoError(t, db.Model(&models.Config{}).Where("config_key = ?", ADLoginHistoryRetentionConfigKey).Update("config_value", "-5").Error)
	if got := svc.GetRetentionDays(ctx); got != defaultADLoginHistoryRetentionDays {
		t.Fatalf("非正数应回退默认, got %d", got)
	}
}
