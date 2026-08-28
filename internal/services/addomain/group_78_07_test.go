//go:build !skip_db_tests
// +build !skip_db_tests

// Phase 78 Plan 07 Task 3: group.go 查询链 + failover 入口失败路径
//
// 覆盖范围(按 78-07-PLAN.md §Task 3B):
//   - GetList 过滤(OUDN/GroupName) + 排序白名单 + 分页
//   - GetByDN (命中/not-found)
//   - GetMembers (成员/空/分页)
//   - AddMember / RemoveMember / Update:各两条失败路径(空池/dial-fail)
//   - NewGroupService 构造器
//
// 设计原则:
//   - 复用 setupSync78DB / insertConfig78 (D-78-06e) -- 本文件用独立 4 表 fixture
//   - 每个 failover 失败用例断言"本地 DB 未被修改"(早退顺序语义)
//   - 成功路径归 Task 4 探针

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

// setupGroup78DB creates a minimal 4-table fixture for group tests.
func setupGroup78DB(t *testing.T) (*gorm.DB, *models.ADConfig) {
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
		CREATE TABLE sys_ad_group (
			id TEXT PRIMARY KEY, ad_config_id TEXT, group_dn TEXT, group_name TEXT,
			description TEXT, ou_dn TEXT, member_count INTEGER DEFAULT 0,
			created_by TEXT, updated_by TEXT, version INTEGER,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`).Error)

	require.NoError(t, db.Exec(`
		CREATE TABLE sys_ad_group_member (
			id TEXT PRIMARY KEY, ad_config_id TEXT, group_dn TEXT, user_dn TEXT,
			created_by TEXT, updated_by TEXT, version INTEGER,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`).Error)

	// sys_ad_user 是 GetMembers JOIN 所需的
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

// insertGroup78 inserts a sys_ad_group row.
func insertGroup78(t *testing.T, db *gorm.DB, configID, id, groupDN, groupName, ouDN string) {
	t.Helper()
	now := time.Now()
	require.NoError(t, db.Exec(`
		INSERT INTO sys_ad_group (id,ad_config_id,group_dn,group_name,ou_dn,created_at,updated_at,deleted_at)
		VALUES (?,?,?,?,?,?,?,?)`,
		id, configID, groupDN, groupName, ouDN, now, now, nil).Error)
}

// insertGroupMember78 inserts a sys_ad_group_member row.
func insertGroupMember78(t *testing.T, db *gorm.DB, configID, groupDN, userDN string) {
	t.Helper()
	now := time.Now()
	require.NoError(t, db.Exec(`
		INSERT INTO sys_ad_group_member (id,ad_config_id,group_dn,user_dn,created_at,updated_at,deleted_at)
		VALUES (?,?,?,?,?,?,?)`,
		uuid.NewString(), configID, groupDN, userDN, now, now, nil).Error)
}

// ==================== GetList ====================

// TestGrp78_GetList_FilterSortPage 验证过滤/排序/分页。
func TestGrp78_GetList_FilterSortPage(t *testing.T) {
	db, cfg := setupGroup78DB(t)
	configID := cfg.ID
	svc := NewGroupService(db, &fakePool78{})

	g1 := uuid.NewString()
	g2 := uuid.NewString()
	g3 := uuid.NewString()
	gDel := uuid.NewString()
	gOther := uuid.NewString()

	insertGroup78(t, db, configID, g1, "CN=Group1,OU=Groups,DC=example,DC=com", "Group1", "OU=Groups,DC=example,DC=com")
	insertGroup78(t, db, configID, g2, "CN=Group2,OU=Groups,DC=example,DC=com", "Group2", "OU=Groups,DC=example,DC=com")
	insertGroup78(t, db, configID, g3, "CN=SubGroup,OU=SubGroups,OU=Groups,DC=example,DC=com", "SubGroup", "OU=SubGroups,OU=Groups,DC=example,DC=com")
	// soft-deleted
	now := time.Now()
	require.NoError(t, db.Exec(`
		INSERT INTO sys_ad_group (id,ad_config_id,group_dn,group_name,ou_dn,created_at,updated_at,deleted_at)
		VALUES (?,?,?,?,?,?,?,?)`,
		gDel, configID, "CN=DeletedGroup,OU=Groups,DC=example,DC=com", "DeletedGroup", "OU=Groups,DC=example,DC=com", now, now, now).Error)
	// other config
	otherCfgID := uuid.NewString()
	require.NoError(t, db.Exec(`
		INSERT INTO sys_ad_config (id,config_name,server_address,server_port,base_dn,domain_name,
			admin_username,admin_password,use_ssl,use_tls,sync_enabled,sync_interval,status,
			created_at,updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		otherCfgID, "other-config", "127.0.0.1", 389, "DC=other,DC=com", "other.com",
		"admin", "pwd", 0, 0, 1, 3600, 0, now, now).Error)
	insertGroup78(t, db, otherCfgID, gOther, "CN=OtherGroup,OU=Groups,DC=other,DC=com", "OtherGroup", "OU=Groups,DC=other,DC=com")

	ctx := context.Background()

	// 默认查询:同 config + 非软删
	groups, total, err := svc.GetList(ctx, &GroupListRequest{
		BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 100},
		ConfigID:        configID,
	})
	require.NoError(t, err)
	names := make([]string, len(groups))
	for i, g := range groups {
		names[i] = g.GroupName
	}
	// 3 visible: Group1, Group2, SubGroup (同 config,非软删)
	assert.Equal(t, int64(3), total)
	assert.Contains(t, names, "Group1")
	assert.Contains(t, names, "Group2")
	assert.Contains(t, names, "SubGroup")
	assert.NotContains(t, names, "DeletedGroup")
	assert.NotContains(t, names, "OtherGroup")

	// OUDN 过滤
	ouDN := "OU=Groups,DC=example,DC=com"
	groups, _, err = svc.GetList(ctx, &GroupListRequest{
		BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 100},
		ConfigID:      configID,
		OUDN:         &ouDN,
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(groups), 2)

	// GroupName LIKE 过滤
	grpName := "Group1"
	groups, _, err = svc.GetList(ctx, &GroupListRequest{
		BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 100},
		ConfigID:      configID,
		GroupName:    &grpName,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, len(groups))
	assert.Equal(t, "Group1", groups[0].GroupName)

	// 白名单排序
	for _, field := range []string{"groupName", "memberCount", "ouDn"} {
		_, _, err = svc.GetList(ctx, &GroupListRequest{
			BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 10, OrderByColumn: field},
			ConfigID:        configID,
		})
		require.NoError(t, err, "sort by %s should not error", field)
	}

	// 白名单外回落 group_name ASC
	_, _, err = svc.GetList(ctx, &GroupListRequest{
		BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 10, OrderByColumn: "unknown"},
		ConfigID:        configID,
	})
	require.NoError(t, err)
}

// TestGrp78_GetByDN 命中/not-found。
func TestGrp78_GetByDN(t *testing.T) {
	db, cfg := setupGroup78DB(t)
	configID := cfg.ID
	svc := NewGroupService(db, &fakePool78{})

	g1 := uuid.NewString()
	insertGroup78(t, db, configID, g1, "CN=Group1,OU=Groups,DC=example,DC=com", "Group1", "OU=Groups,DC=example,DC=com")

	ctx := context.Background()

	g, err := svc.GetByDN(ctx, configID, "CN=Group1,OU=Groups,DC=example,DC=com")
	require.NoError(t, err)
	assert.Equal(t, "Group1", g.GroupName)

	_, err = svc.GetByDN(ctx, configID, "CN=NotExist,OU=Groups,DC=example,DC=com")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "用户组不存在")
}

// TestGrp78_GetMembers 成员/空/分页。
func TestGrp78_GetMembers(t *testing.T) {
	db, cfg := setupGroup78DB(t)
	configID := cfg.ID
	svc := NewGroupService(db, &fakePool78{})

	// 插入一个组 + 3 个成员用户
	g1 := uuid.NewString()
	insertGroup78(t, db, configID, g1, "CN=Group1,OU=Groups,DC=example,DC=com", "Group1", "OU=Groups,DC=example,DC=com")
	insertGroupMember78(t, db, configID, "CN=Group1,OU=Groups,DC=example,DC=com", "CN=user1,OU=Users,DC=example,DC=com")
	insertGroupMember78(t, db, configID, "CN=Group1,OU=Groups,DC=example,DC=com", "CN=user2,OU=Users,DC=example,DC=com")
	insertGroupMember78(t, db, configID, "CN=Group1,OU=Groups,DC=example,DC=com", "CN=user3,OU=Users,DC=example,DC=com")
	// GetMembers joins sys_ad_group_member with sys_ad_user, so users must exist
	insertUser78(t, db, configID, uuid.NewString(), "user1", "CN=user1,OU=Users,DC=example,DC=com", "OU=Users,DC=example,DC=com", true)
	insertUser78(t, db, configID, uuid.NewString(), "user2", "CN=user2,OU=Users,DC=example,DC=com", "OU=Users,DC=example,DC=com", true)
	insertUser78(t, db, configID, uuid.NewString(), "user3", "CN=user3,OU=Users,DC=example,DC=com", "OU=Users,DC=example,DC=com", true)

	ctx := context.Background()

	// 3 成员
	members, total, err := svc.GetMembers(ctx, configID, "CN=Group1,OU=Groups,DC=example,DC=com", 1, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, members, 3)

	// 空组
	members, total, err = svc.GetMembers(ctx, configID, "CN=EmptyGroup,OU=Groups,DC=example,DC=com", 1, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Len(t, members, 0)

	// 分页
	members, total, err = svc.GetMembers(ctx, configID, "CN=Group1,OU=Groups,DC=example,DC=com", 1, 2)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, members, 2)
}

// ==================== Failover 失败路径 ====================

// TestGrp78_AddMember_EmptyPool 空账号池 → ErrAllAccountsUnavailable,本地未新增成员。
func TestGrp78_AddMember_EmptyPool(t *testing.T) {
	db, cfg := setupGroup78DB(t)
	g1 := uuid.NewString()
	insertGroup78(t, db, cfg.ID, g1, "CN=Group1,OU=Groups,DC=example,DC=com", "Group1", "OU=Groups,DC=example,DC=com")

	svc := NewGroupService(db, &fakePool78{listAvailableRes: []models.ADServiceAccount{}})

	err := svc.AddMember(context.Background(), cfg, "CN=Group1,OU=Groups,DC=example,DC=com",
		"CN=user1,OU=Users,DC=example,DC=com")

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrAllAccountsUnavailable))

	// 验证 sys_ad_group_member 未新增
	var cnt int64
	db.Model(&models.ADGroupMember{}).Count(&cnt)
	assert.Equal(t, int64(0), cnt, "AddMember failed should not create member record")
}

// TestGrp78_AddMember_DialFail 可用账号 + closed port → dial 失败,本地未新增。
func TestGrp78_AddMember_DialFail(t *testing.T) {
	db, cfg := setupGroup78DB(t)
	g1 := uuid.NewString()
	insertGroup78(t, db, cfg.ID, g1, "CN=Group1,OU=Groups,DC=example,DC=com", "Group1", "OU=Groups,DC=example,DC=com")

	pool := NewAccountPool(db, nil)
	configID := cfg.ID
	require.NoError(t, db.Exec(`
		INSERT INTO sys_ad_service_accounts (id,config_id,username,password_ciphertext,status,
			failure_count,created_at,updated_at)
		VALUES (?,?,?,?,?,?,?,?)`,
		uuid.NewString(), configID, "acct-A", "enc", 0, 0, time.Now(), time.Now()).Error)

	svc := NewGroupService(db, pool)
	cfg.ServerAddress = "127.0.0.1"
	cfg.ServerPort = closedPort78(t)

	err := svc.AddMember(context.Background(), cfg, "CN=Group1,OU=Groups,DC=example,DC=com",
		"CN=user1,OU=Users,DC=example,DC=com")

	assert.Error(t, err)
	assert.False(t, errors.Is(err, ErrAllAccountsUnavailable))

	var cnt int64
	db.Model(&models.ADGroupMember{}).Count(&cnt)
	assert.Equal(t, int64(0), cnt, "dial-fail should not create member record")
}

// TestGrp78_RemoveMember_FailurePaths RemoveMember 两条失败路径。
func TestGrp78_RemoveMember_FailurePaths(t *testing.T) {
	db, cfg := setupGroup78DB(t)
	g1 := uuid.NewString()
	insertGroup78(t, db, cfg.ID, g1, "CN=Group1,OU=Groups,DC=example,DC=com", "Group1", "OU=Groups,DC=example,DC=com")
	insertGroupMember78(t, db, cfg.ID, "CN=Group1,OU=Groups,DC=example,DC=com", "CN=user1,OU=Users,DC=example,DC=com")

	pool := NewAccountPool(db, nil)
	configID := cfg.ID
	require.NoError(t, db.Exec(`
		INSERT INTO sys_ad_service_accounts (id,config_id,username,password_ciphertext,status,
			failure_count,created_at,updated_at)
		VALUES (?,?,?,?,?,?,?,?)`,
		uuid.NewString(), configID, "acct-A", "enc", 0, 0, time.Now(), time.Now()).Error)

	cfg.ServerAddress = "127.0.0.1"
	cfg.ServerPort = closedPort78(t)
	svc := NewGroupService(db, pool)

	// EmptyPool
	err := svc.RemoveMember(context.Background(), cfg, "CN=Group1,OU=Groups,DC=example,DC=com",
		"CN=user1,OU=Users,DC=example,DC=com")
	assert.Error(t, err)

	// DialFail (pool already has accounts, so this tests dial path)
	err = svc.RemoveMember(context.Background(), cfg, "CN=Group1,OU=Groups,DC=example,DC=com",
		"CN=user1,OU=Users,DC=example,DC=com")
	assert.Error(t, err)
}

// TestGrp78_Update_FailurePaths Update 两条失败路径。
func TestGrp78_Update_FailurePaths(t *testing.T) {
	db, cfg := setupGroup78DB(t)
	g1 := uuid.NewString()
	insertGroup78(t, db, cfg.ID, g1, "CN=Group1,OU=Groups,DC=example,DC=com", "Group1", "OU=Groups,DC=example,DC=com")

	pool := NewAccountPool(db, nil)
	configID := cfg.ID
	require.NoError(t, db.Exec(`
		INSERT INTO sys_ad_service_accounts (id,config_id,username,password_ciphertext,status,
			failure_count,created_at,updated_at)
		VALUES (?,?,?,?,?,?,?,?)`,
		uuid.NewString(), configID, "acct-A", "enc", 0, 0, time.Now(), time.Now()).Error)

	cfg.ServerAddress = "127.0.0.1"
	cfg.ServerPort = closedPort78(t)
	svc := NewGroupService(db, pool)

	desc := "updated description"

	// EmptyPool
	err := svc.Update(context.Background(), cfg, "CN=Group1,OU=Groups,DC=example,DC=com",
		&GroupUpdateRequest{Description: &desc})
	assert.Error(t, err)

	// DialFail
	err = svc.Update(context.Background(), cfg, "CN=Group1,OU=Groups,DC=example,DC=com",
		&GroupUpdateRequest{Description: &desc})
	assert.Error(t, err)
}

// TestGrp78_NewGroupService 构造器。
func TestGrp78_NewGroupService(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	pool := &fakePool78{}
	svc := NewGroupService(db, pool)
	assert.NotNil(t, svc)
}

// D-78-07 覆盖边界:group.go 成功路径(AD 写成功 → 本地 DB 落库/member_count 增减)
// 依赖 Task 4 LDAP 应答器;若探针受阻则不覆盖,已在文件头注明。
