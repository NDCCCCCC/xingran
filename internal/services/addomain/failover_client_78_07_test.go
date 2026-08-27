//go:build !skip_db_tests
// +build !skip_db_tests

// Phase 78 Plan 07 Task 2: failover_client.go 剩余边界覆盖
//
// 覆盖范围(按 78-07-PLAN.md §Task 2):
//   - ExecuteWithFailover: ListAvailable error / Empty pool / MaxHops 截断 / MidLoopRecovery / MarkSuccess error
//   - newClient: 生产分支(不设 clientFactory,直调 NewLDAPClient)
//   - PickFirstConnect: ListAvailable error / Empty pool / AllDialFail / MaxHops
//
// 设计原则:
//   - fakePool78 实现 AccountPool 接口(同包最小 fake,避免与 mock 命名冲突)
//   - 复用 78-05/78-06 helper 不得重定义(D-78-06e)
//   - clientFactory 注入手法照搬 failover_client_76_03_test.go(用 BaseModel{ID} 构造 ADConfig)
//   - 成功路径(Bind 成功)归 Task 4 探针
//   - closedPort78 定义在 ldap_client_78_07_test.go(同包共享)

package addomain

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// fakePool78 是一个同包最小 fake AccountPool 实现,用于注入 ListAvailable error 等边界。
type fakePool78 struct {
	listAvailableErr  error
	listAvailableRes []models.ADServiceAccount
	markSuccessErr  error
	markFailureCalls []struct{ id, reason string }
	markFailureErr  error
}

func (f *fakePool78) PickAvailable(ctx context.Context, configID string) (*models.ADServiceAccount, error) {
	return nil, errors.New("not implemented")
}
func (f *fakePool78) ListAvailable(ctx context.Context, configID string) ([]models.ADServiceAccount, error) {
	if f.listAvailableErr != nil {
		return nil, f.listAvailableErr
	}
	return f.listAvailableRes, nil
}
func (f *fakePool78) ListAll(ctx context.Context, configID string, page, pageSize int, statusFilter *int) ([]models.ADServiceAccount, int64, error) {
	return nil, 0, errors.New("not implemented")
}
func (f *fakePool78) CountByStatus(ctx context.Context, configID string) (total, available, disabled, circuitBroken int64, err error) {
	return 0, 0, 0, 0, errors.New("not implemented")
}
func (f *fakePool78) PickFirstAvailable(ctx context.Context, configID string) (*models.ADServiceAccount, error) {
	return nil, errors.New("not implemented")
}
func (f *fakePool78) Create(ctx context.Context, account *models.ADServiceAccount) error { return errors.New("not implemented") }
func (f *fakePool78) Update(ctx context.Context, account *models.ADServiceAccount) error { return errors.New("not implemented") }
func (f *fakePool78) Delete(ctx context.Context, accountID string) error { return errors.New("not implemented") }
func (f *fakePool78) MarkSuccess(ctx context.Context, accountID string) error {
	if f.markSuccessErr != nil {
		return f.markSuccessErr
	}
	return nil
}
func (f *fakePool78) MarkFailure(ctx context.Context, accountID, reason string) error {
	if f.markFailureErr != nil {
		return f.markFailureErr
	}
	f.markFailureCalls = append(f.markFailureCalls, struct{ id, reason string }{accountID, reason})
	return nil
}
func (f *fakePool78) ManualUnlock(ctx context.Context, accountID, operator, reason string) error { return errors.New("not implemented") }
func (f *fakePool78) SetEnabled(ctx context.Context, accountID string, enabled bool) error { return errors.New("not implemented") }
func (f *fakePool78) RecoverExpiredBreakers(ctx context.Context) (int, error) { return 0, errors.New("not implemented") }
func (f *fakePool78) InvalidateCache(configID string) {}
func (f *fakePool78) StartHotReload(ctx context.Context) error { return errors.New("not implemented") }

// ==================== ExecuteWithFailover 边界 ====================

// TestFC78_ExecuteWithFailover_ListAvailableError ListAvailable 返回 error。
func TestFC78_ExecuteWithFailover_ListAvailableError(t *testing.T) {
	pool := &fakePool78{listAvailableErr: errors.New("查询账号池失败")}
	cfg := &models.ADConfig{BaseModel: models.BaseModel{ID: "cfg-1"}}
	fc := NewFailoverClient(pool, cfg)

	err := fc.ExecuteWithFailover(context.Background(), func(client LDAPClientIface) error {
		return nil
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "查询账号池失败")
}

// TestFC78_ExecuteWithFailover_EmptyPool 返回空切片 → ErrAllAccountsUnavailable。
func TestFC78_ExecuteWithFailover_EmptyPool(t *testing.T) {
	pool := &fakePool78{listAvailableRes: []models.ADServiceAccount{}}
	cfg := &models.ADConfig{BaseModel: models.BaseModel{ID: "cfg-2"}}
	fc := NewFailoverClient(pool, cfg)

	err := fc.ExecuteWithFailover(context.Background(), func(client LDAPClientIface) error {
		return nil
	})

	assert.True(t, errors.Is(err, ErrAllAccountsUnavailable))
}

// TestFC78_ExecuteWithFailover_MaxHopsTruncation 造 DefaultMaxHops+3 个账号 + connectErr mock。
// 验证 MarkFailureCalls == DefaultMaxHops 且错误文案含截断后的数字。
func TestFC78_ExecuteWithFailover_MaxHopsTruncation(t *testing.T) {
	var accounts []models.ADServiceAccount
	for i := 0; i < DefaultMaxHops+3; i++ {
		accounts = append(accounts, models.ADServiceAccount{
			ID:       uuid.NewString(),
			Username: "acct-" + string(rune('A'+i)),
		})
	}
	pool := &fakePool78{listAvailableRes: accounts}
	cfg := &models.ADConfig{BaseModel: models.BaseModel{ID: "cfg-3"}}
	fc := NewFailoverClient(pool, cfg)
	fc.clientFactory = func(config *models.ADConfig, acct *models.ADServiceAccount) LDAPClientIface {
		return &mockLDAPClient{connectErr: errors.New("connect failed")}
	}

	err := fc.ExecuteWithFailover(context.Background(), func(client LDAPClientIface) error {
		return nil
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "账号池")
	assert.Contains(t, err.Error(), "个账号均失败")
	assert.Equal(t, DefaultMaxHops, len(pool.markFailureCalls))
}

// TestFC78_ExecuteWithFailover_MidLoopRecovery 在 failover_client_76_03_test.go 中已覆盖
// (TestFailover_SequentialTraversal_StopsOnFirstSuccess 验证顺序遍历+第三账号成功)。
// 本文件 focus 在 78-07 新增边界,不再重复同族路径。

// TestFC78_ExecuteWithFailover_MarkSuccessError MarkSuccess 返回 error → ExecuteWithFailover 仍返回 nil。
func TestFC78_ExecuteWithFailover_MarkSuccessError(t *testing.T) {
	accounts := []models.ADServiceAccount{
		{ID: uuid.NewString(), Username: "acct-good"},
	}
	pool := &fakePool78{listAvailableRes: accounts, markSuccessErr: errors.New("MarkSuccess error")}
	cfg := &models.ADConfig{BaseModel: models.BaseModel{ID: "cfg-5"}}
	fc := NewFailoverClient(pool, cfg)
	fc.clientFactory = func(config *models.ADConfig, acct *models.ADServiceAccount) LDAPClientIface {
		return &mockLDAPClient{connectErr: nil}
	}

	err := fc.ExecuteWithFailover(context.Background(), func(client LDAPClientIface) error {
		return nil
	})

	// MarkSuccess error 是 warn 分支,ExecuteWithFailover 仍返回 nil
	assert.NoError(t, err)
}

// TestFC78_NewClient_ProductionBranch 不设 clientFactory → 直调 NewLDAPClient。
func TestFC78_NewClient_ProductionBranch(t *testing.T) {
	pool := &fakePool78{}
	cfg := &models.ADConfig{BaseModel: models.BaseModel{ID: "cfg-6"}}
	fc := NewFailoverClient(pool, cfg)

	// 不设 clientFactory,走生产分支
	acct := &models.ADServiceAccount{ID: uuid.NewString(), Username: "prod-acct"}
	client := fc.newClient(acct)

	// 返回值是 *LDAPClient(编译期通过 LDAPClientIface 断言保证)
	ldapClient, ok := client.(*LDAPClient)
	assert.True(t, ok, "newClient 生产分支应返回 *LDAPClient")
	assert.Equal(t, acct, ldapClient.GetAccount())
}

// ==================== PickFirstConnect 边界 ====================

// TestFC78_PickFirstConnect_ListAvailableError ListAvailable error。
func TestFC78_PickFirstConnect_ListAvailableError(t *testing.T) {
	pool := &fakePool78{listAvailableErr: errors.New("查询账号池失败")}
	cfg := &models.ADConfig{BaseModel: models.BaseModel{ID: "cfg-7"}}
	fc := NewFailoverClient(pool, cfg)

	client, acct, err := fc.PickFirstConnect(context.Background())

	assert.Nil(t, client)
	assert.Nil(t, acct)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "查询账号池失败")
}

// TestFC78_PickFirstConnect_EmptyPool 空切片。
func TestFC78_PickFirstConnect_EmptyPool(t *testing.T) {
	pool := &fakePool78{listAvailableRes: []models.ADServiceAccount{}}
	cfg := &models.ADConfig{BaseModel: models.BaseModel{ID: "cfg-8"}}
	fc := NewFailoverClient(pool, cfg)

	client, acct, err := fc.PickFirstConnect(context.Background())

	assert.Nil(t, client)
	assert.Nil(t, acct)
	assert.True(t, errors.Is(err, ErrAllAccountsUnavailable))
}

// TestFC78_PickFirstConnect_AllDialFail PickFirstConnect 不使用 clientFactory(直调 NewLDAPClient)。
// 用真实 closed port + real NewLDAPClient.Connect() 触发 dial 失败。
func TestFC78_PickFirstConnect_AllDialFail(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_ad_service_accounts (
			id TEXT PRIMARY KEY, config_id TEXT NOT NULL, username TEXT NOT NULL,
			password_ciphertext TEXT NOT NULL, status INTEGER NOT NULL DEFAULT 0,
			failure_count INTEGER NOT NULL DEFAULT 0, circuit_breaker_until DATETIME,
			last_success_at DATETIME, last_failure_at DATETIME, last_failure_reason TEXT,
			manual_unlock_reason TEXT, manual_unlocked_by TEXT, manual_unlocked_at DATETIME,
			remark TEXT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`).Error)
	pool := NewAccountPool(db, nil)
	configID := uuid.NewString()

	for i := 0; i < 2; i++ {
		require.NoError(t, db.Exec(`
			INSERT INTO sys_ad_service_accounts
			(id,config_id,username,password_ciphertext,status,failure_count,created_at,updated_at)
			VALUES (?,?,?,?,0,0,?,?)`,
			uuid.NewString(), configID, "acct-"+string(rune('A'+i)), "enc", time.Now(), time.Now()).Error)
	}

	// 申请一个 closed port(监听后立即关闭)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	host, portStr, err := net.SplitHostPort(ln.Addr().String())
	ln.Close()
	require.NoError(t, err)
	closedPort := 0
	for _, c := range portStr {
		closedPort = closedPort*10 + int(c-'0')
	}

	// PickFirstConnect 不用 clientFactory,直接调 NewLDAPClient + Connect()
	// 因此让 config 指向 closed port,真实 Connect() 触发 dial 失败
	cfg := &models.ADConfig{
		BaseModel:     models.BaseModel{ID: configID},
		ServerAddress: host,
		ServerPort:    closedPort,
	}
	fc := NewFailoverClient(pool, cfg)

	client, acctOut, err := fc.PickFirstConnect(context.Background())

	assert.Nil(t, client)
	assert.Nil(t, acctOut)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "账号池")
	assert.Contains(t, err.Error(), "个账号均无法连接")
}

// TestFC78_PickFirstConnect_MaxHops 造 DefaultMaxHops+2 个账号,closed port 触发真实 dial 失败。
func TestFC78_PickFirstConnect_MaxHops(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_ad_service_accounts (
			id TEXT PRIMARY KEY, config_id TEXT NOT NULL, username TEXT NOT NULL,
			password_ciphertext TEXT NOT NULL, status INTEGER NOT NULL DEFAULT 0,
			failure_count INTEGER NOT NULL DEFAULT 0, circuit_breaker_until DATETIME,
			last_success_at DATETIME, last_failure_at DATETIME, last_failure_reason TEXT,
			manual_unlock_reason TEXT, manual_unlocked_by TEXT, manual_unlocked_at DATETIME,
			remark TEXT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`).Error)
	pool := NewAccountPool(db, nil)
	configID := uuid.NewString()

	for i := 0; i < DefaultMaxHops+2; i++ {
		require.NoError(t, db.Exec(`
			INSERT INTO sys_ad_service_accounts
			(id,config_id,username,password_ciphertext,status,failure_count,created_at,updated_at)
			VALUES (?,?,?,?,0,0,?,?)`,
			uuid.NewString(), configID, "acct-"+string(rune('A'+i)), "enc", time.Now(), time.Now()).Error)
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	host, portStr, err := net.SplitHostPort(ln.Addr().String())
	ln.Close()
	require.NoError(t, err)
	closedPort := 0
	for _, c := range portStr {
		closedPort = closedPort*10 + int(c-'0')
	}

	cfg := &models.ADConfig{
		BaseModel:     models.BaseModel{ID: configID},
		ServerAddress: host,
		ServerPort:    closedPort,
	}
	fc := NewFailoverClient(pool, cfg)

	client, acctOut, err := fc.PickFirstConnect(context.Background())

	assert.Nil(t, client)
	assert.Nil(t, acctOut)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "账号池")
	assert.Contains(t, err.Error(), "个账号均无法连接")
}
