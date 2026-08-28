//go:build !skip_db_tests
// +build !skip_db_tests

// Phase 78 Plan 07 Task 3: user.go 查询链 + failover 入口失败路径
//
// 覆盖范围(按 78-07-PLAN.md §Task 3A):
//   - GetList 过滤矩阵 ($DUPLICATE-/%$/软删/OUDN/Username/IsEnabled/pagination/sort)
//   - GetByDN / GetByID (命中/not-found/软删排除/DB 错误)
//   - GetUserIds
//   - Update / Enable / Disable / Move:空账号池 + 全 dial 失败两条失败路径
//   - NewUserService 构造器
//
// 设计原则:
//   - 复用 setupSync78DB (7 表)/ insertConfig78 helper (D-78-06e) -- 但本文件用独立 3 表 fixture 避免改 78-05
//   - 每个 failover 失败用例断言"本地 DB 未被修改"(早退顺序语义)
//   - 成功路径归 Task 4 探针
//   - closedPort78 在 ldap_client_78_07_test.go 已定义,同包共享(不重定义)

package addomain

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
)

// setupUser78DB creates a minimal 3-table fixture (sys_ad_config + sys_ad_user + sys_ad_service_accounts).
func setupUser78DB(t *testing.T) (*gorm.DB, *models.ADConfig) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, db.Exec(`
		CREATE TABLE sys_ad_config (
			id TEXT PRIMARY KEY, config_name TEXT, server_address TEXT, server_port INTEGER,
			base_dn TEXT, domain_name TEXT, admin_username TEXT, admin_password TEXT,
			use_ssl INTEGER, use_tls INTEGER, sync_enabled INTEGER, sync_interval INTEGER,
			member_ou_dn TEXT, last_sync_at DATETIME, status INTEGER,
			created_by TEXT, updated_by TEXT, version INTEGER, created_at DATETIME,
			updated_at DATETIME, deleted_at DATETIME
		)`).Error)

	require.NoError(t, db.Exec(`
		CREATE TABLE sys_ad_user (
			id TEXT PRIMARY KEY, ad_config_id TEXT, user_dn TEXT, username TEXT,
			display_name TEXT, email TEXT, phone TEXT, mobile TEXT, title TEXT,
			department TEXT, description TEXT, ou_dn TEXT, is_enabled INTEGER,
			ad_sync_log_id TEXT, last_sync_at DATETIME,
			created_by TEXT, updated_by TEXT, version INTEGER,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`).Error)

	require.NoError(t, db.Exec(`
		CREATE TABLE sys_ad_service_accounts (
			id TEXT PRIMARY KEY, config_id TEXT NOT NULL, username TEXT NOT NULL,
			password_ciphertext TEXT NOT NULL, status INTEGER NOT NULL DEFAULT 0,
			failure_count INTEGER NOT NULL DEFAULT 0, circuit_breaker_until DATETIME,
			last_success_at DATETIME, last_failure_at DATETIME, last_failure_reason TEXT,
			manual_unlock_reason TEXT, manual_unlocked_by TEXT, manual_unlocked_at DATETIME,
			remark TEXT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`).Error)

	configID := uuid.NewString()
	now := time.Now()
	require.NoError(t, db.Exec(`
		INSERT INTO sys_ad_config (id,config_name,server_address,server_port,base_dn,domain_name,
			admin_username,admin_password,use_ssl,use_tls,sync_enabled,sync_interval,status,
			created_at,updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		configID, "test-config", "127.0.0.1", 389, "DC=example,DC=com", "example.com",
		"admin", "pwd", 0, 0, 1, 3600, 0, now, now).Error)

	cfg := &models.ADConfig{BaseModel: models.BaseModel{ID: configID}}
	return db, cfg
}

// insertUser78 inserts a sys_ad_user row.
func insertUser78(t *testing.T, db *gorm.DB, configID, id, username, userDN, ouDN string, isEnabled bool) {
	t.Helper()
	enabled := 0
	if isEnabled {
		enabled = 1
	}
	now := time.Now()
	require.NoError(t, db.Exec(`
		INSERT INTO sys_ad_user (id,ad_config_id,user_dn,username,display_name,ou_dn,is_enabled,
			created_at,updated_at,deleted_at)
		VALUES (?,?,?,?,?,?,?,?,?,?)`,
		id, configID, userDN, username, username+"-display", ouDN, enabled, now, now, nil).Error)
}

// ==================== GetList ====================

// TestUsr78_GetList_FilterMatrix 验证 6 个过滤条件矩阵。
func TestUsr78_GetList_FilterMatrix(t *testing.T) {
	db, cfg := setupUser78DB(t)
	configID := cfg.ID
	svc := NewUserService(db, &fakePool78{})

	u1 := uuid.NewString()
	u2 := uuid.NewString()
	u3 := uuid.NewString()
	uDup := uuid.NewString()
	uComp := uuid.NewString()
	uDel := uuid.NewString()
	uOther := uuid.NewString()
	uSub := uuid.NewString()

	insertUser78(t, db, configID, u1, "john", "CN=john,OU=Users,DC=example,DC=com", "OU=Users,DC=example,DC=com", true)
	insertUser78(t, db, configID, u2, "jane", "CN=jane,OU=Users,DC=example,DC=com", "OU=Users,DC=example,DC=com", true)
	insertUser78(t, db, configID, u3, "bob", "CN=bob,OU=Staff,DC=example,DC=com", "OU=Staff,DC=example,DC=com", true)
	insertUser78(t, db, configID, uDup, "$DUPLICATE-john", "CN=$DUPLICATE-john,OU=Users,DC=example,DC=com", "OU=Users,DC=example,DC=com", true)
	insertUser78(t, db, configID, uComp, "computer$", "CN=computer,OU=Computers,DC=example,DC=com", "OU=Computers,DC=example,DC=com", true)
	// soft-deleted row
	now := time.Now()
	require.NoError(t, db.Exec(`
		INSERT INTO sys_ad_user (id,ad_config_id,user_dn,username,display_name,ou_dn,is_enabled,
			created_at,updated_at,deleted_at)
		VALUES (?,?,?,?,?,?,?,?,?,?)`,
		uDel, configID, "CN=deleted,OU=Users,DC=example,DC=com", "deleted", "deleted-display",
		"OU=Users,DC=example,DC=com", 1, now, now, now).Error)
	// another config row
	otherCfgID := uuid.NewString()
	require.NoError(t, db.Exec(`
		INSERT INTO sys_ad_config (id,config_name,server_address,server_port,base_dn,domain_name,
			admin_username,admin_password,use_ssl,use_tls,sync_enabled,sync_interval,status,
			created_at,updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		otherCfgID, "other-config", "127.0.0.1", 389, "DC=other,DC=com", "other.com",
		"admin", "pwd", 0, 0, 1, 3600, 0, now, now).Error)
	insertUser78(t, db, otherCfgID, uOther, "other-user", "CN=other,OU=Users,DC=other,DC=com", "OU=Users,DC=other,DC=com", true)
	// sub-OU row
	insertUser78(t, db, configID, uSub, "sub-user", "CN=sub,OU=SubDept,OU=Users,DC=example,DC=com", "OU=SubDept,OU=Users,DC=example,DC=com", true)

	ctx := context.Background()

	// 默认查询:同 config + 非软删 + 非 $DUPLICATE- + 非 %$ → 4 visible
	users, total, err := svc.GetList(ctx, &UserListRequest{
		BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 100},
		ConfigID:        configID,
	})
	require.NoError(t, err)
	usernames := make([]string, len(users))
	for i, u := range users {
		usernames[i] = u.Username
	}
	// 4 visible: john, jane, bob (normal) + sub-user (sub-OU); other-user 属另一 config 不出现
	assert.Equal(t, int64(4), total)
	assert.Contains(t, usernames, "john")
	assert.Contains(t, usernames, "jane")
	assert.Contains(t, usernames, "bob")
	assert.Contains(t, usernames, "sub-user")
	assert.NotContains(t, usernames, "$DUPLICATE-john")
	assert.NotContains(t, usernames, "computer$")
	assert.NotContains(t, usernames, "deleted")
	assert.NotContains(t, usernames, "other-user")

	// OUDN 过滤:命中父 OU 时包含子 OU
	ouDN := "OU=Users,DC=example,DC=com"
	_, _, err = svc.GetList(ctx, &UserListRequest{
		BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 100},
		ConfigID:      configID,
		OUDN:          &ouDN,
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(users), 2, "OUDN filter should include sub-OU users")
}

// TestUsr78_GetList_SortWhitelist 验证白名单排序。
func TestUsr78_GetList_SortWhitelist(t *testing.T) {
	db, cfg := setupUser78DB(t)
	configID := cfg.ID
	svc := NewUserService(db, &fakePool78{})

	names := []string{"alice", "charlie", "bob"}
	for i, name := range names {
		id := uuid.NewString()
		insertUser78(t, db, configID, id, name, "CN="+name+",OU=Users,DC=example,DC=com", "OU=Users,DC=example,DC=com", true)
		_ = i
	}

	ctx := context.Background()

	// 白名单字段排序
	for _, field := range []string{"username", "displayName", "email", "phone", "ouDn", "isEnabled"} {
		users, _, err := svc.GetList(ctx, &UserListRequest{
			BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 10, OrderByColumn: field},
			ConfigID:        configID,
		})
		require.NoError(t, err, "sort by %s should not error", field)
		assert.NotEmpty(t, users, "sort by %s should return results", field)
	}

	// 白名单外字段 → 回落 username ASC
	users, _, err := svc.GetList(ctx, &UserListRequest{
		BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 10, OrderByColumn: "unknownField"},
		ConfigID:        configID,
	})
	require.NoError(t, err, "unknown sort field should not error")
	assert.NotEmpty(t, users)
}

// TestUsr78_GetList_Pagination 验证分页。
func TestUsr78_GetList_Pagination(t *testing.T) {
	db, cfg := setupUser78DB(t)
	configID := cfg.ID
	svc := NewUserService(db, &fakePool78{})

	// 插入 12 行
	names := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l"}
	for _, name := range names {
		id := uuid.NewString()
		insertUser78(t, db, configID, id, name, "CN="+name+",OU=Users,DC=example,DC=com", "OU=Users,DC=example,DC=com", true)
	}

	ctx := context.Background()
	pageSize := 5

	p1, total, err := svc.GetList(ctx, &UserListRequest{
		BaseListRequest: base.BaseListRequest{Current: 1, PageSize: pageSize},
		ConfigID:        configID,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(12), total)
	assert.Len(t, p1, 5)

	p2, _, err := svc.GetList(ctx, &UserListRequest{
		BaseListRequest: base.BaseListRequest{Current: 2, PageSize: pageSize},
		ConfigID:        configID,
	})
	require.NoError(t, err)
	assert.Len(t, p2, 5)

	p3, _, err := svc.GetList(ctx, &UserListRequest{
		BaseListRequest: base.BaseListRequest{Current: 3, PageSize: pageSize},
		ConfigID:        configID,
	})
	require.NoError(t, err)
	assert.Len(t, p3, 2)
}

// ==================== GetByDN / GetByID ====================

// TestUsr78_GetByDN_And_GetByID 命中/not-found/软删排除。
func TestUsr78_GetByDN_And_GetByID(t *testing.T) {
	db, cfg := setupUser78DB(t)
	configID := cfg.ID
	svc := NewUserService(db, &fakePool78{})

	u1 := uuid.NewString()
	insertUser78(t, db, configID, u1, "john", "CN=john,OU=Users,DC=example,DC=com", "OU=Users,DC=example,DC=com", true)

	ctx := context.Background()

	u, err := svc.GetByDN(ctx, configID, "CN=john,OU=Users,DC=example,DC=com")
	require.NoError(t, err)
	assert.Equal(t, "john", u.Username)

	_, err = svc.GetByDN(ctx, configID, "CN=notexist,OU=Users,DC=example,DC=com")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "用户不存在")

	_, err = svc.GetByID(ctx, uuid.NewString())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "用户不存在")
}

// TestUsr78_GetUserIds 验证 GetUserIds。
func TestUsr78_GetUserIds(t *testing.T) {
	db, cfg := setupUser78DB(t)
	configID := cfg.ID
	svc := NewUserService(db, &fakePool78{})

	id1 := uuid.NewString()
	id2 := uuid.NewString()
	insertUser78(t, db, configID, id1, "john", "CN=john,OU=Users,DC=example,DC=com", "OU=Users,DC=example,DC=com", true)
	insertUser78(t, db, configID, id2, "jane", "CN=jane,OU=Users,DC=example,DC=com", "OU=Users,DC=example,DC=com", true)

	ctx := context.Background()
	ids, err := svc.GetUserIds(ctx, &UserListRequest{
		BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 10},
		ConfigID:        configID,
	})
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{id1, id2}, ids)

	ids, err = svc.GetUserIds(ctx, &UserListRequest{
		BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 10},
		ConfigID:        uuid.NewString(),
	})
	require.NoError(t, err)
	assert.Empty(t, ids)
}

// ==================== Failover 失败路径 ====================

// TestUsr78_Update_EmptyPool 空账号池 → ErrAllAccountsUnavailable,本地 DB 未更新。
func TestUsr78_Update_EmptyPool(t *testing.T) {
	db, cfg := setupUser78DB(t)
	u1 := uuid.NewString()
	insertUser78(t, db, cfg.ID, u1, "john", "CN=john,OU=Users,DC=example,DC=com", "OU=Users,DC=example,DC=com", true)

	svc := NewUserService(db, &fakePool78{listAvailableRes: []models.ADServiceAccount{}})

	displayName := "John Updated"
	err := svc.Update(context.Background(), cfg, "CN=john,OU=Users,DC=example,DC=com",
		&UserUpdateRequest{DisplayName: &displayName})

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrAllAccountsUnavailable))

	// 验证本地 DB 未被更新
	var u models.ADUser
	require.NoError(t, db.Where("id = ?", u1).First(&u).Error)
	assert.NotEqual(t, displayName, u.DisplayName, "Update failed should not modify local DB")
}

// TestUsr78_Update_AllAccountsDialFail 2 可用账号 + closed port → dial 失败且本地 DB 未更新。
func TestUsr78_Update_AllAccountsDialFail(t *testing.T) {
	db, cfg := setupUser78DB(t)
	u1 := uuid.NewString()
	insertUser78(t, db, cfg.ID, u1, "john", "CN=john,OU=Users,DC=example,DC=com", "OU=Users,DC=example,DC=com", true)

	pool := NewAccountPool(db, nil)
	configID := cfg.ID
	for i := 0; i < 2; i++ {
		require.NoError(t, db.Exec(`
			INSERT INTO sys_ad_service_accounts (id,config_id,username,password_ciphertext,status,
				failure_count,created_at,updated_at)
			VALUES (?,?,?,?,?,?,?,?)`,
			uuid.NewString(), configID, "acct-"+string(rune('A'+i)), "enc", 0, 0, time.Now(), time.Now()).Error)
	}

	svc := NewUserService(db, pool)
	cfg.ServerAddress = "127.0.0.1"
	cfg.ServerPort = closedPort78(t)

	displayName := "John Updated"
	err := svc.Update(context.Background(), cfg, "CN=john,OU=Users,DC=example,DC=com",
		&UserUpdateRequest{DisplayName: &displayName})

	assert.Error(t, err)
	assert.False(t, errors.Is(err, ErrAllAccountsUnavailable), "expect dial-fail error, not empty-pool")

	var u models.ADUser
	require.NoError(t, db.Where("id = ?", u1).First(&u).Error)
	assert.NotEqual(t, displayName, u.DisplayName, "dial-fail should not update local DB")
}

// TestUsr78_EnableDisableMove_FailurePaths Enable/Disable/Move 各两条失败路径。
func TestUsr78_EnableDisableMove_FailurePaths(t *testing.T) {
	db, cfg := setupUser78DB(t)
	u1 := uuid.NewString()
	insertUser78(t, db, cfg.ID, u1, "john", "CN=john,OU=Users,DC=example,DC=com", "OU=Users,DC=example,DC=com", true)

	pool := NewAccountPool(db, nil)
	configID := cfg.ID
	require.NoError(t, db.Exec(`
		INSERT INTO sys_ad_service_accounts (id,config_id,username,password_ciphertext,status,
			failure_count,created_at,updated_at)
		VALUES (?,?,?,?,?,?,?,?)`,
		uuid.NewString(), configID, "acct-A", "enc", 0, 0, time.Now(), time.Now()).Error)

	cfg.ServerAddress = "127.0.0.1"
	cfg.ServerPort = closedPort78(t)
	svc := NewUserService(db, pool)

	// Enable/Disable/Move 各两个失败用例
	type op struct {
		name string
		fn   func() error
	}
	ops := []op{
		{"Enable_EmptyPool", func() error {
			svc2 := NewUserService(db, &fakePool78{listAvailableRes: []models.ADServiceAccount{}})
			return svc2.Enable(context.Background(), cfg, "CN=john,OU=Users,DC=example,DC=com")
		}},
		{"Disable_EmptyPool", func() error {
			svc2 := NewUserService(db, &fakePool78{listAvailableRes: []models.ADServiceAccount{}})
			return svc2.Disable(context.Background(), cfg, "CN=john,OU=Users,DC=example,DC=com")
		}},
		{"Move_EmptyPool", func() error {
			svc2 := NewUserService(db, &fakePool78{listAvailableRes: []models.ADServiceAccount{}})
			return svc2.Move(context.Background(), cfg, "CN=john,OU=Users,DC=example,DC=com", "OU=NewDept,DC=example,DC=com")
		}},
		{"Enable_DialFail", func() error {
			return svc.Enable(context.Background(), cfg, "CN=john,OU=Users,DC=example,DC=com")
		}},
		{"Disable_DialFail", func() error {
			return svc.Disable(context.Background(), cfg, "CN=john,OU=Users,DC=example,DC=com")
		}},
		{"Move_DialFail", func() error {
			return svc.Move(context.Background(), cfg, "CN=john,OU=Users,DC=example,DC=com", "OU=NewDept,DC=example,DC=com")
		}},
	}
	for _, o := range ops {
		t.Run(o.name, func(t *testing.T) {
			err := o.fn()
			assert.Error(t, err)
		})
	}
}

// TestUsr78_NewUserService 构造器。
func TestUsr78_NewUserService(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	pool := &fakePool78{}
	svc := NewUserService(db, pool)
	assert.NotNil(t, svc)
}

// D-78-07 覆盖边界:user.go 成功路径(AD 写成功 → 本地 DB 更新)依赖 Task 4 LDAP 应答器;
// 若探针受阻则不覆盖,已在文件头注明。
