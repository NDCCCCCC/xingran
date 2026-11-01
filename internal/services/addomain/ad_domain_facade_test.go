package addomain

// DEBT-01（v1.51）第四批：ADDomainService 门面 DB 路径 + 组管理校验分支 +
// LDAP 搜索错误处理纯逻辑。
//
// 覆盖目标：
//  1. ADDomainService 门面：AD 配置 CRUD 列表 / AD 用户与组列表（DB 过滤）/
//     同步日志分页 / 组成员查询 / 计算机列表；TestADConnection、SyncADData
//     的配置缺失错误路径（不触网）
//  2. GroupManagementService：DeleteGroup / AddMembers / RemoveMembers 的
//     LDAP 前置校验分支（组不存在 / 有成员 / 有映射 / 无有效 AD 用户）
//  3. LDAPClient.handleSearchError：SizeLimit 递归降级上限分支与非 SizeLimit
//     包装分支；extractPagingControl 提取逻辑

import (
	"context"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	goldap "github.com/go-ldap/ldap/v3"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	base "github.com/xingran-next/xingran-go-backend/internal/services/base"
	"github.com/xingran-next/xingran-go-backend/internal/models"
)

func setupFacadeTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	// 逐模型迁移：部分模型带 PG 专属 DDL（gen_random_uuid），失败则手写等价建表
	for _, m := range []interface{}{
		&models.ADConfig{}, &models.ADUser{}, &models.ADGroup{},
		&models.ADSyncLog{}, &models.ADComputer{}, &models.Department{},
		&models.ADGroupMember{}, &models.ADOU{}, &models.User{},
	} {
		if err := db.AutoMigrate(m); err != nil {
			t.Fatalf("迁移 %T 失败: %v", m, err)
		}
	}
	// OUGroupMapping（gen_random_uuid 默认值 sqlite 不兼容）→ 手写等价表
	if !db.Migrator().HasTable("sys_ou_group_mapping") {
		if err := db.Exec(`CREATE TABLE sys_ou_group_mapping (
			id TEXT PRIMARY KEY,
			ad_config_id TEXT NOT NULL,
			ou_dn TEXT NOT NULL UNIQUE,
			ou_name TEXT NOT NULL,
			ad_group_id TEXT NOT NULL,
			mapping_status TEXT NOT NULL DEFAULT 'active',
			sync_enabled INTEGER DEFAULT 1,
			last_sync_at DATETIME,
			created_by TEXT DEFAULT '', updated_by TEXT DEFAULT '',
			deleted_at DATETIME,
			created_at DATETIME, updated_at DATETIME
		)`).Error; err != nil {
			t.Fatalf("建 sys_ou_group_mapping 失败: %v", err)
		}
	}
	return db
}

func TestADDomainServiceFacade_DBPaths(t *testing.T) {
	db := setupFacadeTestDB(t)
	ctx := context.Background()
	svc := NewADDomainService(db, nil)

	cfg := &models.ADConfig{
		ConfigName: "主配置", ServerAddress: "127.0.0.1", ServerPort: 389,
		DomainName: "test.com", BaseDN: "dc=test", SyncEnabled: true, Status: models.ADConfigStatusEnabled,
	}
	require.NoError(t, db.Create(cfg).Error)

	// 配置列表 + 分页
	list, total, err := svc.GetADConfigList(ctx, nil, 1, 10, "", nil)
	require.NoError(t, err)
	if total != 1 || len(list) != 1 || list[0].ID != cfg.ID {
		t.Fatalf("配置列表不符: total=%d", total)
	}

	// 按 ID 查
	got, err := svc.GetADConfigByID(ctx, cfg.ID)
	require.NoError(t, err)
	if got.ID != cfg.ID {
		t.Fatalf("GetADConfigByID 不符: %+v", got)
	}
	if _, err := svc.GetADConfigByID(ctx, "nope"); err == nil {
		t.Fatal("不存在的配置应报错")
	}
	// TestADConnection / SyncADData：配置不存在即报错，不触网
	if err := svc.TestADConnection(ctx, "nope"); err == nil {
		t.Fatal("TestADConnection 配置缺失应报错")
	}
	if _, err := svc.SyncADData(ctx, "nope", "full"); err == nil {
		t.Fatal("SyncADData 配置缺失应报错")
	}

	// AD 用户列表：DB 过滤（config + username）
	users := []models.ADUser{
		{ADConfigID: cfg.ID, UserDN: "CN=alice,DC=test", Username: "alice", IsEnabled: true, OUN: "OU=a,DC=test"},
		{ADConfigID: cfg.ID, UserDN: "CN=bob,DC=test", Username: "bob", IsEnabled: false, MemberOf: "CN=g1,DC=test"},
	}
	require.NoError(t, db.Create(&users).Error)
	ul, total, err := svc.GetADUserList(ctx, &ADUserListRequest{ConfigID: cfg.ID, Username: strPtr("ali"), BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 10}})
	require.NoError(t, err)
	if total != 1 || len(ul) != 1 || ul[0].Username != "alice" {
		t.Fatalf("用户过滤不符: total=%d ul=%+v", total, ul)
	}
	// GetByDN / GetByID
	u, err := svc.GetADUserByDN(ctx, cfg.ID, "CN=alice,DC=test")
	require.NoError(t, err)
	if u.Username != "alice" {
		t.Fatalf("GetADUserByDN 不符: %+v", u)
	}
	u, err = svc.GetADUserByID(ctx, users[1].ID)
	require.NoError(t, err)
	if u.Username != "bob" {
		t.Fatalf("GetADUserByID 不符: %+v", u)
	}

	// AD 组列表 + 组成员（member_of 包含匹配）
	groups := []models.ADGroup{
		{ADConfigID: cfg.ID, GroupDN: "CN=g1,DC=test", GroupName: "g1", MemberCount: 1},
	}
	require.NoError(t, db.Create(&groups).Error)
	gl, total, err := svc.GetADGroupList(ctx, &ADGroupListRequest{ConfigID: cfg.ID, GroupName: strPtr("g1"), BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 10}})
	require.NoError(t, err)
	if total != 1 || len(gl) != 1 {
		t.Fatalf("组列表不符: total=%d", total)
	}
	// 组成员查询：走 sys_ad_group_member 关联行
	require.NoError(t, db.Create(&models.ADGroupMember{ADConfigID: cfg.ID, GroupDN: "CN=g1,DC=test", UserDN: "CN=bob,DC=test"}).Error)
	members, total, err := svc.GetADGroupMembers(ctx, cfg.ID, "CN=g1,DC=test", 1, 10)
	require.NoError(t, err)
	if total != 1 || len(members) != 1 || members[0].Username != "bob" {
		t.Fatalf("组成员查询不符: total=%d members=%+v", total, members)
	}

	// 同步日志分页
	logs := []models.ADSyncLog{{ADConfigID: cfg.ID}}
	require.NoError(t, db.Create(&logs).Error)
	ll, total, err := svc.GetADSyncLogList(ctx, cfg.ID, 1, 10, "", nil)
	require.NoError(t, err)
	if total != 1 || len(ll) != 1 {
		t.Fatalf("同步日志分页不符: total=%d", total)
	}

	// 计算机列表 + GetByDN
	computers := []models.ADComputer{
		{ADConfigID: cfg.ID, ComputerName: "PC1", DistinguishedName: "CN=PC1,DC=test", IPAddress: "10.0.0.2"},
	}
	require.NoError(t, db.Create(&computers).Error)
	cl, total, err := svc.GetADComputerList(ctx, &ComputerListRequest{ConfigID: cfg.ID, BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 10}})
	require.NoError(t, err)
	if total != 1 || len(cl) != 1 {
		t.Fatalf("计算机列表不符: total=%d", total)
	}
	cd, err := svc.GetADComputerByDN(ctx, cfg.ID, "CN=PC1,DC=test")
	require.NoError(t, err)
	if cd == nil || cd.IPAddress != "10.0.0.2" {
		t.Fatalf("GetADComputerByDN 不符: %+v", cd)
	}

	// OU 树：无映射数据 → 空树（实现返回 nil 或空切片均视为空树）
	tree, err := svc.GetOUTree(ctx, cfg.ID)
	require.NoError(t, err)
	if len(tree) != 0 {
		t.Fatalf("空数据应返回空树, got %d 节点", len(tree))
	}

	// 配置删除
	require.NoError(t, svc.DeleteADConfig(ctx, cfg.ID))
	if _, err := svc.GetADConfigByID(ctx, cfg.ID); err == nil {
		t.Fatal("删除后查询应报错")
	}
}

func TestGroupManagementValidationBranches(t *testing.T) {
	db := setupFacadeTestDB(t)
	ctx := context.Background()
	svc := NewGroupManagementService(db, nil)

	cfg := &models.ADConfig{BaseModel: models.BaseModel{ID: "cfg-1"}, ConfigName: "c", ServerAddress: "127.0.0.1", ServerPort: 389, DomainName: "d", BaseDN: "dc=t"}
	require.NoError(t, db.Create(cfg).Error)
	groups := []models.ADGroup{
		{BaseModel: models.BaseModel{ID: "g-full"}, ADConfigID: "cfg-1", GroupDN: "CN=full,DC=t", GroupName: "full", MemberCount: 2},
		{BaseModel: models.BaseModel{ID: "g-mapped"}, ADConfigID: "cfg-1", GroupDN: "CN=mapped,DC=t", GroupName: "mapped"},
		{BaseModel: models.BaseModel{ID: "g-clean"}, ADConfigID: "cfg-1", GroupDN: "CN=clean,DC=t", GroupName: "clean"},
	}
	require.NoError(t, db.Create(&groups).Error)
	require.NoError(t, db.Create(&models.OUGroupMapping{ADConfigID: "cfg-1", ADGroupID: "g-mapped", OUDN: "OU=x,DC=t", OUName: "x"}).Error)

	// DeleteGroup：组不存在
	if err := svc.DeleteGroup(ctx, "nope"); err == nil || !strings.Contains(err.Error(), "不存在") {
		t.Fatalf("组不存在应报错, got %v", err)
	}
	// DeleteGroup：有成员
	if err := svc.DeleteGroup(ctx, "g-full"); err == nil || !strings.Contains(err.Error(), "成员") {
		t.Fatalf("有成员应拒绝删除, got %v", err)
	}
	// DeleteGroup：有 OU 映射
	if err := svc.DeleteGroup(ctx, "g-mapped"); err == nil || !strings.Contains(err.Error(), "映射") {
		t.Fatalf("有映射应拒绝删除, got %v", err)
	}
	// AddMembers：组不存在 / 无有效 AD 用户
	if _, err := svc.AddMembers(ctx, "nope", []string{"u-1"}); err == nil || !strings.Contains(err.Error(), "不存在") {
		t.Fatalf("AddMembers 组不存在应报错, got %v", err)
	}
	if _, err := svc.AddMembers(ctx, "g-clean", []string{"u-none"}); err == nil || !strings.Contains(err.Error(), "没有有效") {
		t.Fatalf("AddMembers 无有效 AD 用户应报错, got %v", err)
	}
	// RemoveMembers：组不存在 / 无有效 AD 用户
	if _, err := svc.RemoveMembers(ctx, "nope", []string{"u-1"}); err == nil || !strings.Contains(err.Error(), "不存在") {
		t.Fatalf("RemoveMembers 组不存在应报错, got %v", err)
	}
	if _, err := svc.RemoveMembers(ctx, "g-clean", []string{"u-none"}); err == nil || !strings.Contains(err.Error(), "没有有效") {
		t.Fatalf("RemoveMembers 无有效 AD 用户应报错, got %v", err)
	}
}

func TestLDAPClientSearchErrorHelpers(t *testing.T) {
	c := &LDAPClient{}

	// 非 SizeLimit 错误 → 原样包装
	_, err := c.handleSearchError(context.DeadlineExceeded, "dc=t", "(objectClass=*)", nil, 100, 0)
	if err == nil || !strings.Contains(err.Error(), "搜索失败") {
		t.Fatalf("非 SizeLimit 错误应包装, got %v", err)
	}
	// SizeLimit 且已到递归降级上限 → 终止错误
	sizeErr := &goldap.Error{ResultCode: goldap.LDAPResultSizeLimitExceeded, Err: context.DeadlineExceeded}
	_, err = c.handleSearchError(sizeErr, "dc=t", "(objectClass=*)", nil, 100, maxSearchRetryDepth)
	if err == nil || !strings.Contains(err.Error(), "递归降级") {
		t.Fatalf("到达降级上限应终止, got %v", err)
	}
	// SizeLimit 但 pageSize 已到底 → 终止错误
	_, err = c.handleSearchError(sizeErr, "dc=t", "(objectClass=*)", nil, minPageSize, 0)
	if err == nil || !strings.Contains(err.Error(), "分页") {
		t.Fatalf("pageSize 到底应终止, got %v", err)
	}
	// extractPagingControl：提取第一个分页控件；无控件 → nil
	pc := goldap.NewControlPaging(100)
	if got := c.extractPagingControl([]goldap.Control{pc}); got != pc {
		t.Fatal("应提取到分页控件")
	}
	if got := c.extractPagingControl([]goldap.Control{}); got != nil {
		t.Fatal("空控件列表应返回 nil")
	}
}
