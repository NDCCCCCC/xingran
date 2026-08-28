//go:build !skip_db_tests
// +build !skip_db_tests

// Phase 78 Plan 06: config.go 7 函数 CRUD + TestConnection 两条失败分支 (Task 4 第二段)
//
// 复用 78-05 helper（setupSync78DB / insertConfig78 / closeDB）— D-78-06e 禁止重定义。
// sys_ad_config + sys_ad_service_accounts 已在 78-05 7 表 fixture 内；本文件无新表。
//
// ⚠️ TestConnection 的 bind 成功路径不在本 plan（D-78-06b），归 78-07 的 in-process LDAP 应答器。
// 本 plan 仅覆盖：(1) 空账号池 → ErrAllAccountsUnavailable；(2) 账号池非空但全部
// bind 失败（指向 127.0.0.1 本地已关闭端口，零真实网络）。
//
// 零生产 .go 改动。

package addomain

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
)

// insertCfg78 插入一条 sys_ad_config 行（raw SQL，规避 GORM default:true quirk）。
//
// 可选参数顺序（按需）：
//   [0] config_name   (string)
//   [1] server_address (string)
//   [2] server_port    (int)
//   [3] base_dn        (string)
//   [4] domain_name    (string)
//   [5] status         (models.ADConfigStatus)
func insertCfg78(t *testing.T, db *gorm.DB, opts ...any) *models.ADConfig {
	t.Helper()
	id := uuid.NewString()
	name := "cfg-" + id[:8]
	serverAddr := "127.0.0.1"
	serverPort := 389
	baseDN := "DC=example,DC=com"
	domain := "example.com"
	status := models.ADConfigStatusEnabled
	if len(opts) > 0 {
		if v, ok := opts[0].(string); ok {
			name = v
		}
	}
	if len(opts) > 1 {
		if v, ok := opts[1].(string); ok {
			serverAddr = v
		}
	}
	if len(opts) > 2 {
		if v, ok := opts[2].(int); ok {
			serverPort = v
		}
	}
	if len(opts) > 3 {
		if v, ok := opts[3].(string); ok {
			baseDN = v
		}
	}
	if len(opts) > 4 {
		if v, ok := opts[4].(string); ok {
			domain = v
		}
	}
	if len(opts) > 5 {
		if v, ok := opts[5].(models.ADConfigStatus); ok {
			status = v
		}
	}
	now := time.Now()
	require.NoError(t, db.Exec(`
		INSERT INTO sys_ad_config
		(id, config_name, server_address, server_port, base_dn, domain_name, use_ssl, use_tls,
		 sync_enabled, sync_interval, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, 0, 0, 1, 3600, ?, ?, ?)
	`, id, name, serverAddr, serverPort, baseDN, domain, int(status), now, now).Error)

	cfg := &models.ADConfig{}
	require.NoError(t, db.First(cfg, "id = ?", id).Error)
	return cfg
}

// reservedClosedPort78 在本地拿一个当前可用的端口，立即关闭，模拟"已关闭端口"。
// T-78-06-01 mitigation。
func reservedClosedPort78(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := l.Addr().(*net.TCPAddr).Port
	require.NoError(t, l.Close())
	return port
}

// insertAccount78 raw SQL 插入一个 service account（避开 account_pool_test.go 的同包重定义风险）。
func insertAccount78(t *testing.T, db *gorm.DB, configID, username string, status int) string {
	t.Helper()
	id := uuid.NewString()
	now := time.Now()
	require.NoError(t, db.Exec(`
		INSERT INTO sys_ad_service_accounts
		(id, config_id, username, password_ciphertext, status, failure_count, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 0, ?, ?)
	`, id, configID, username, "encrypted_pwd", status, now, now).Error)
	return id
}

// =============================================================================
// Task 4 第二段：ConfigService 7 函数
// =============================================================================

// TestCfg78_NewConfigService 构造器。
func TestCfg78_NewConfigService(t *testing.T) {
	db := setupSync78DB(t)
	defer closeDB(t, db)
	pool := NewAccountPool(db, nil)
	svc := NewConfigService(db, pool)
	assert.NotNil(t, svc)
}

// TestCfg78_GetList_FilterSortPage status 过滤 + 白名单排序（含防注入回落）+ 分页与 total。
func TestCfg78_GetList_FilterSortPage(t *testing.T) {
	db := setupSync78DB(t)
	defer closeDB(t, db)
	svc := NewConfigService(db, NewAccountPool(db, nil))
	ctx := context.Background()

	// 预置 5 条配置（status 混合 + name 多样）
	for i := 0; i < 3; i++ {
		insertCfg78(t, db, "alpha-"+uuid.NewString()[:6])
	}
	insertCfg78(t, db, "bravo-"+uuid.NewString()[:6], "", 0, "", "", models.ADConfigStatusDisabled)
	insertCfg78(t, db, "charlie-"+uuid.NewString()[:6])

	// 全部
	list, total, err := svc.GetList(ctx, &ListRequest{
		BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 10},
	})
	require.NoError(t, err)
	assert.EqualValues(t, 5, total)
	assert.Len(t, list, 5)

	// status 过滤：仅 enabled
	enabled := int(models.ADConfigStatusEnabled)
	list, total, err = svc.GetList(ctx, &ListRequest{
		BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 10},
		Status:          &enabled,
	})
	require.NoError(t, err)
	assert.EqualValues(t, 4, total)
	assert.Len(t, list, 4)

	// status 过滤：仅 disabled
	disabled := int(models.ADConfigStatusDisabled)
	list, total, err = svc.GetList(ctx, &ListRequest{
		BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 10},
		Status:          &disabled,
	})
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)

	// 白名单内排序：configName 升序
	asc := true
	list, _, err = svc.GetList(ctx, &ListRequest{
		BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 10, OrderByColumn: "configName", IsAsc: &asc},
	})
	require.NoError(t, err)
	require.Len(t, list, 5)
	assert.Contains(t, list[0].ConfigName, "alpha-", "configName 升序 → alpha 三个排前")

	// 白名单外排序（防注入）→ 回落 created_at DESC，不报 SQL 错误
	list, _, err = svc.GetList(ctx, &ListRequest{
		BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 10, OrderByColumn: "1; DROP TABLE sys_ad_config--"},
	})
	require.NoError(t, err, "白名单外字段不报 SQL 错误")
	assert.Len(t, list, 5)

	// 表未被删
	var n int64
	require.NoError(t, db.Model(&models.ADConfig{}).Count(&n).Error)
	assert.EqualValues(t, 5, n)

	// 分页：pageSize=2 → 第 1 页 2 行 + 第 3 页 1 行
	list, total, err = svc.GetList(ctx, &ListRequest{
		BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 2},
	})
	require.NoError(t, err)
	assert.EqualValues(t, 5, total)
	assert.Len(t, list, 2)

	list, total, err = svc.GetList(ctx, &ListRequest{
		BaseListRequest: base.BaseListRequest{Current: 3, PageSize: 2},
	})
	require.NoError(t, err)
	assert.EqualValues(t, 5, total)
	assert.Len(t, list, 1, "第 3 页 1 行（5 - 2*2 = 1）")
}

// TestCfg78_GetByID 命中 / 不命中。
func TestCfg78_GetByID(t *testing.T) {
	db := setupSync78DB(t)
	defer closeDB(t, db)
	svc := NewConfigService(db, NewAccountPool(db, nil))
	ctx := context.Background()
	cfg := insertCfg78(t, db, "hit-me")

	got, err := svc.GetByID(ctx, cfg.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "hit-me", got.ConfigName)

	// 不命中
	_, err = svc.GetByID(ctx, uuid.NewString())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "AD配置不存在")
}

// TestCfg78_Create 合法创建。
//
// 现行为锁（D-78-06h）：Create 写入 Status=ADConfigStatusEnabled 硬编码；handler 层 binding
// tag 做格式校验（不在本测试范围）。
func TestCfg78_Create(t *testing.T) {
	db := setupSync78DB(t)
	defer closeDB(t, db)
	svc := NewConfigService(db, NewAccountPool(db, nil))
	ctx := context.Background()

	req := &CreateRequest{
		ConfigName:    "new-cfg",
		ServerAddress: "10.0.0.1",
		ServerPort:    636,
		DomainName:    "corp.example.com",
		BaseDN:        "DC=corp,DC=example,DC=com",
		UseSSL:        false,
		UseTLS:        false,
		SyncEnabled:   true,
		SyncInterval:  300,
		MemberOUDN:    "OU=Members,DC=corp,DC=example,DC=com",
	}
	got, err := svc.Create(ctx, req, "creator-uuid")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.NotEmpty(t, got.ID)
	assert.Equal(t, "new-cfg", got.ConfigName)
	assert.Equal(t, "10.0.0.1", got.ServerAddress)
	assert.Equal(t, 636, got.ServerPort)
	assert.Equal(t, "corp.example.com", got.DomainName)
	assert.Equal(t, "DC=corp,DC=example,DC=com", got.BaseDN)
	assert.Equal(t, models.ADConfigStatusEnabled, got.Status, "Create 硬编码 Status=Enabled (config.go:113)")
	assert.Equal(t, 0, got.Version, "新行 Version=0")
	assert.Equal(t, "creator-uuid", got.CreatedBy)
}

// TestCfg78_Update 字段更新 + version 自增 + 不存在 id。
//
// 现行为锁：Update 不修改 admin_username / admin_password（config.go:147-159 未列入 updates）；
// 不修改 Status 除非 req.Status 非 nil；不修改 CreatedBy。
// plan §4.B 提到的"Update 内密码字段特殊处理"未在当前 service 落地（service 不动该字段），
// 故不另设断言。
func TestCfg78_Update(t *testing.T) {
	db := setupSync78DB(t)
	defer closeDB(t, db)
	svc := NewConfigService(db, NewAccountPool(db, nil))
	ctx := context.Background()
	cfg := insertCfg78(t, db, "old-name")

	newStatus := int(models.ADConfigStatusDisabled)
	upd := &UpdateRequest{
		ID:            cfg.ID,
		ConfigName:    "new-name",
		ServerAddress: "10.0.0.99",
		ServerPort:    636,
		DomainName:    "new.example.com",
		BaseDN:        "DC=new,DC=example,DC=com",
		UseSSL:        true,
		UseTLS:        false,
		SyncEnabled:   true,
		SyncInterval:  7200,
		MemberOUDN:    "OU=NewMembers,DC=new,DC=example,DC=com",
		Status:        &newStatus,
	}
	require.NoError(t, svc.Update(ctx, upd, "updater-uuid"))

	got, err := svc.GetByID(ctx, cfg.ID)
	require.NoError(t, err)
	assert.Equal(t, "new-name", got.ConfigName)
	assert.Equal(t, "10.0.0.99", got.ServerAddress)
	assert.Equal(t, 636, got.ServerPort)
	assert.Equal(t, models.ADConfigStatusDisabled, got.Status, "Status 应被更新")
	assert.Equal(t, "updater-uuid", got.UpdatedBy, "UpdatedBy 应被设置")
	assert.Equal(t, 1, got.Version, "Version 应自增")

	// 不存在 id
	err = svc.Update(ctx, &UpdateRequest{
		ID: uuid.NewString(), ConfigName: "x", ServerAddress: "x", ServerPort: 1,
		DomainName: "x", BaseDN: "x", SyncInterval: 60,
	}, "updater-uuid")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "AD配置不存在")
}

// TestCfg78_Delete 删除成功（软删）+ 不存在 id。
func TestCfg78_Delete(t *testing.T) {
	db := setupSync78DB(t)
	defer closeDB(t, db)
	svc := NewConfigService(db, NewAccountPool(db, nil))
	ctx := context.Background()
	cfg := insertCfg78(t, db, "to-delete")

	require.NoError(t, svc.Delete(ctx, cfg.ID))

	// 软删：deleted_at 非空，Unscoped 可查到
	var d models.ADConfig
	require.NoError(t, db.Unscoped().First(&d, "id = ?", cfg.ID).Error)
	assert.NotNil(t, d.DeletedAt, "软删后 DeletedAt 应被设置")

	// 正常查询找不到
	_, err := svc.GetByID(ctx, cfg.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "AD配置不存在")

	// 不存在 id
	err = svc.Delete(ctx, uuid.NewString())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "AD配置不存在")
}

// TestCfg78_TestConnection_EmptyPool 空 sys_ad_service_accounts → ErrAllAccountsUnavailable
// + 包装文案含"账号池无可用账号"。
func TestCfg78_TestConnection_EmptyPool(t *testing.T) {
	db := setupSync78DB(t)
	defer closeDB(t, db)
	pool := NewAccountPool(db, nil)
	svc := NewConfigService(db, pool)
	ctx := context.Background()

	cfg := insertCfg78(t, db, "empty-pool-cfg")
	err := svc.TestConnection(ctx, cfg)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrAllAccountsUnavailable),
		"errors.Is(err, ErrAllAccountsUnavailable) 应成立 (config.go:199)")
	assert.Contains(t, err.Error(), "账号池无可用账号",
		"包装文案应含'账号池无可用账号' (config.go:200)")
}

// TestCfg78_TestConnection_AllBindFail 2 个可用账号 + 本地已关闭端口 → 全部 bind 失败分支：
//   - 错误文案含"全部 bind 失败"
//   - 错误**不**是 ErrAllAccountsUnavailable
//   - 两账号的 failure_count 被 MarkFailure 递增（从 0 → 1）
//   - 10s 硬超时守卫（实测通常 <2s）
func TestCfg78_TestConnection_AllBindFail(t *testing.T) {
	db := setupSync78DB(t)
	defer closeDB(t, db)
	pool := NewAccountPool(db, nil)
	svc := NewConfigService(db, pool)
	ctx := context.Background()

	cfg := insertCfg78(t, db, "allbindfail-cfg", "127.0.0.1", reservedClosedPort78(t))

	// 2 个可用账号（status=Available, failure_count=0）
	acctID1 := insertAccount78(t, db, cfg.ID, "svc-1", AccountStatusAvailable)
	acctID2 := insertAccount78(t, db, cfg.ID, "svc-2", AccountStatusAvailable)

	// 10s 硬超时守卫（与 78-05 Task 5 同纪律，T-78-06-02 mitigation）
	done := make(chan error, 1)
	go func() {
		done <- svc.TestConnection(ctx, cfg)
	}()

	start := time.Now()
	var testErr error
	select {
	case testErr = <-done:
		elapsed := time.Since(start)
		t.Logf("TestConnection 全 bind 失败用例实测耗时 %v", elapsed)
		require.Less(t, elapsed, 10*time.Second, "10s 硬超时守卫（T-78-06-02）")
	case <-time.After(10 * time.Second):
		t.Fatal("TestConnection 超过 10s 未返回（dial 超时失控）")
	}

	require.Error(t, testErr, "AllBindFail 应返回错误")
	assert.False(t, errors.Is(testErr, ErrAllAccountsUnavailable),
		"AllBindFail 不应是 ErrAllAccountsUnavailable (config.go:199-201)")
	assert.Contains(t, testErr.Error(), "全部 bind 失败",
		"错误文案应含'全部 bind 失败' (config.go:202)")

	// 验证两账号 failure_count 递增到 1（MarkFailure 被调用一次）
	var cnt1, cnt2 int
	require.NoError(t, db.Raw(`SELECT failure_count FROM sys_ad_service_accounts WHERE id = ?`, acctID1).Scan(&cnt1).Error)
	require.NoError(t, db.Raw(`SELECT failure_count FROM sys_ad_service_accounts WHERE id = ?`, acctID2).Scan(&cnt2).Error)
	assert.Equal(t, 1, cnt1, "acct1 应被 MarkFailure 一次")
	assert.Equal(t, 1, cnt2, "acct2 应被 MarkFailure 一次")
}

// =============================================================================
// 覆盖边界块注释
// =============================================================================
//
// D-78-06b TestConnection bind 成功路径不在本 plan：
// PickFirstConnect (failover_client.go:99-131) 内部 NewLDAPClient.Connect() 需真实 LDAP wire，
// 78-RESEARCH R8 已明确"仅测 factory != nil 路径 / 不强求生产路径测试"。
// bind 成功路径若需覆盖，归 78-07 的 in-process LDAP 应答器方案。
//
// 本 plan 已覆盖：(1) 空账号池 → ErrAllAccountsUnavailable；(2) 账号池非空但全部
// bind 失败 → "全部 bind 失败" 错误（不达 ErrAllAccountsUnavailable）；两条失败路径
// 即可达成 config.go 加权 ≥70%。