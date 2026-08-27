//go:build !skip_db_tests
// +build !skip_db_tests

// Phase 78 Plan 05: sync.go 全链 entry-driven 测试
//
// 设计要点(详见 78-05-PLAN.md):
//   - 零 LDAP 网络 / 零 mock 框架:全部用 []*ldap.Entry 字面量驱动
//   - sqlite + 手动 CREATE TABLE(参考 dept_ou_mapper_test.go:18 注释:
//     AutoMigrate 在 sqlite 下有外键/__temp+RENAME 风险)
//   - 状态字段全部引用 models.* 常量,禁裸 0/1
//   - D-78-07:拒绝为 sync.go 加生产 seam,syncDataInternal happy path 接受不覆盖
//
// 本文件落地的 helper 供后续 78-06 / 78-07 复用(D-78-06e 禁止重定义):
//   - setupSync78DB(t):7 表 sqlite fixture
//   - entry78(dn, kv):ldap.Entry 构造
//   - newSyncSvc78(t):(SyncService, *gorm.DB) 返回
//   - insertConfig78(t, db):启用状态 ADConfig 行

package addomain

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/go-ldap/ldap/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// setupSync78DB 构造 7 表 sqlite 内存 fixture。
//
// 表名用生产表名(sys_ad_*)以便 GORM 查询解析正确;
// 表结构精简到被测函数实际引用的列。
//
// 记忆教训(参见 memory/xingran-sqlite-missing-table-pattern.md):
// 被测函数触达的任何表都必须在此创建;跑测试若报 `no such table: X`,
// 先补建表再继续,禁改生产 SQL。
func setupSync78DB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 1. sys_ad_config (D-78-07 失败路径会查/插入此表)
	// 含 BaseModel 列(created_by / updated_by / version)+ ADConfig 业务列
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_ad_config (
			id TEXT PRIMARY KEY,
			config_name TEXT,
			server_address TEXT,
			server_port INTEGER,
			base_dn TEXT,
			domain_name TEXT,
			admin_username TEXT,
			admin_password TEXT,
			use_ssl INTEGER,
			use_tls INTEGER,
			sync_enabled INTEGER,
			sync_interval INTEGER,
			member_ou_dn TEXT,
			last_sync_at DATETIME,
			status INTEGER,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)
	`).Error)

	// 2. sys_ad_ou (syncOUs / batchCreateOUs / batchUpdateOUs / categorizeOUs / getExistingOUs)
	// 含 BaseModel 列(ADOU 嵌入 BaseModel,GORM INSERT 会写 created_by 等)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_ad_ou (
			id TEXT PRIMARY KEY,
			ad_config_id TEXT,
			ou_dn TEXT,
			ou_name TEXT,
			ou_path TEXT,
			parent_dn TEXT,
			description TEXT,
			user_count INTEGER DEFAULT 0,
			group_count INTEGER DEFAULT 0,
			last_sync_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			UNIQUE(ad_config_id, ou_dn)
		)
	`).Error)

	// 3. sys_ad_group (syncGroups / createGroupsInBatches / updateGroupsInBatches)
	// 含 BaseModel 列(ADGroup 嵌入 BaseModel)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_ad_group (
			id TEXT PRIMARY KEY,
			ad_config_id TEXT,
			group_dn TEXT,
			group_name TEXT,
			description TEXT,
			member_count INTEGER,
			ou_dn TEXT,
			group_scope TEXT,
			group_type INTEGER,
			last_sync_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			UNIQUE(ad_config_id, group_dn)
		)
	`).Error)

	// 4. sys_ad_user (syncUsers:逐字段对应 ADUser 列)
	// 含 BaseModel 列(ADUser 嵌入 BaseModel)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_ad_user (
			id TEXT PRIMARY KEY,
			ad_config_id TEXT,
			user_dn TEXT,
			username TEXT,
			display_name TEXT,
			email TEXT,
			phone TEXT,
			mobile TEXT,
			title TEXT,
			department TEXT,
			company TEXT,
			description TEXT,
			ou_dn TEXT,
			user_account_control INTEGER,
			is_enabled INTEGER,
			is_locked INTEGER,
			password_expired INTEGER,
			last_logon DATETIME,
			password_last_set DATETIME,
			account_expires DATETIME,
			member_of TEXT,
			last_sync_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			UNIQUE(ad_config_id, user_dn)
		)
	`).Error)

	// 5. sys_ad_group_member (syncGroupMembers)
	// 含 BaseModel 列(ADGroupMember 嵌入 BaseModel)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_ad_group_member (
			id TEXT PRIMARY KEY,
			ad_config_id TEXT,
			group_dn TEXT,
			user_dn TEXT,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)
	`).Error)

	// 6. sys_ad_sync_log (syncDataInternal 创建日志 + updateSyncLog 状态机)
	// ADSyncLog 不是 BaseModel 嵌入,而是独立 ID,所以 deleted_at 列虽不在 model 上但
	// updateSyncLog 不会用到,省略;created_at 必有(BeforeCreate 钩子)。
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_ad_sync_log (
			id TEXT PRIMARY KEY,
			ad_config_id TEXT,
			sync_type TEXT,
			sync_status TEXT,
			start_time DATETIME,
			end_time DATETIME,
			duration INTEGER,
			ou_count INTEGER DEFAULT 0,
			group_count INTEGER DEFAULT 0,
			user_count INTEGER DEFAULT 0,
			computer_count INTEGER DEFAULT 0,
			error_count INTEGER DEFAULT 0,
			error_msg TEXT,
			created_at DATETIME,
			updated_at DATETIME
		)
	`).Error)

	// 7. sys_ad_service_accounts (D-78-07 空账号池失败路径)
	// 照搬 account_pool_test.go:26-46
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_ad_service_accounts (
			id TEXT PRIMARY KEY,
			config_id TEXT NOT NULL,
			username TEXT NOT NULL,
			password_ciphertext TEXT NOT NULL,
			status INTEGER NOT NULL DEFAULT 0,
			failure_count INTEGER NOT NULL DEFAULT 0,
			circuit_breaker_until DATETIME,
			last_success_at DATETIME,
			last_failure_at DATETIME,
			last_failure_reason TEXT,
			manual_unlock_reason TEXT,
			manual_unlocked_by TEXT,
			manual_unlocked_at DATETIME,
			remark TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)
	`).Error)

	return db
}

// entry78 构造单个 ldap.Entry 字面量,key→[]string 形式给属性赋值。
func entry78(dn string, kv map[string][]string) *ldap.Entry {
	attrs := make([]*ldap.EntryAttribute, 0, len(kv))
	for k, v := range kv {
		attrs = append(attrs, &ldap.EntryAttribute{Name: k, Values: v})
	}
	return &ldap.Entry{DN: dn, Attributes: attrs}
}

// newSyncSvc78 返回 (SyncService, *gorm.DB) 供同包白盒测试直调私有方法。
// pool 字段注入 AccountPool(为 Task 5 syncDataInternal 失败路径准备)。
func newSyncSvc78(t *testing.T) (*SyncService, *gorm.DB) {
	t.Helper()
	db := setupSync78DB(t)
	svc := &SyncService{db: db, pool: NewAccountPool(db, nil)}
	return svc, db
}

// insertConfig78 插入一条启用状态的 ADConfig 并返回。
func insertConfig78(t *testing.T, db *gorm.DB, id string) *models.ADConfig {
	t.Helper()
	if id == "" {
		id = uuid.NewString()
	}
	cfg := &models.ADConfig{
		ConfigName:    "cfg-" + id[:8],
		ServerAddress: "127.0.0.1",
		ServerPort:    389,
		BaseDN:        "DC=example,DC=com",
		DomainName:    "example.com",
		Status:        models.ADConfigStatusEnabled,
	}
	cfg.ID = id
	require.NoError(t, db.Create(cfg).Error)
	return cfg
}

// closeDB 关闭 sqlite 连接,避免测试泄漏连接导致 race 报错。
func closeDB(t *testing.T, db *gorm.DB) {
	t.Helper()
	sqlDB, err := db.DB()
	if err != nil {
		return
	}
	_ = sqlDB.Close()
}

// ============================================================================
// Task 1: OU 管道全链(syncOUs / categorizeOUs / batchCreateOUs / batchUpdateOUs /
// getExistingOUs / extractDNs / safeAttr)
// ============================================================================

// TestSync78_ExtractDNs_And_SafeAttr 覆盖 extractDNs 与 safeAttr 两个纯函数。
//
// extractDNs:输入 []*ldap.Entry,逐条取 entry.DN 拼接。空切片/多条两形态。
// safeAttr:基于 utils.SanitizeAndTruncate,length 截断与非法字符清洗行为已由
//   sanitize_test.go 覆盖;此处只做 smoke 断言(签名存在 + 长度限制生效)。
func TestSync78_ExtractDNs_And_SafeAttr(t *testing.T) {
	// extractDNs 空切片
	assert.Empty(t, extractDNs(nil))
	assert.Empty(t, extractDNs([]*ldap.Entry{}))

	// extractDNs 多条
	entries := []*ldap.Entry{
		{DN: "OU=A,DC=example,DC=com"},
		{DN: "OU=B,DC=example,DC=com"},
		{DN: "OU=C,DC=example,DC=com"},
	}
	dns := extractDNs(entries)
	assert.Equal(t, []string{"OU=A,DC=example,DC=com", "OU=B,DC=example,DC=com", "OU=C,DC=example,DC=com"}, dns)

	// safeAttr 短字符串透传
	assert.Equal(t, "hello", safeAttr("hello", 255))
	// safeAttr 超长截断:TruncateForLog 行为是 maxLen-1 runes + "…"(utils/string_helper.go:103)
	long := strings.Repeat("x", 300)
	truncated := safeAttr(long, 50)
	runeCount := len([]rune(truncated))
	assert.LessOrEqual(t, runeCount, 50, "rune 数不超过 maxLen(截断尾标占 1 rune)")
	assert.Contains(t, truncated, "…", "超长应追加 ellipsis 标记")
}

// TestSync78_GetExistingOUs 三形态:全部命中 / 部分命中 / 空列表;
// 此外断言 deleted_at IS NULL 过滤(预置一条软删行不应被查到)。
func TestSync78_GetExistingOUs(t *testing.T) {
	svc, db := newSyncSvc78(t)
	defer closeDB(t, db)
	ctx := context.Background()
	cfg := insertConfig78(t, db, "")

	// 预置两条 OU:ou-a 正常 + ou-b 软删
	now := time.Now()
	ouA := &models.ADOU{ADConfigID: cfg.ID, OUN: "OU=A,DC=example,DC=com", OUName: "A", LastSyncAt: &now}
	ouA.ID = uuid.NewString()
	require.NoError(t, db.Create(ouA).Error)

	ouB := &models.ADOU{ADConfigID: cfg.ID, OUN: "OU=B,DC=example,DC=com", OUName: "B", LastSyncAt: &now}
	ouB.ID = uuid.NewString()
	require.NoError(t, db.Create(ouB).Error)
	require.NoError(t, db.Delete(ouB).Error) // 软删

	// 1) 全部命中(只命中未删的)
	got := svc.getExistingOUs(ctx, cfg.ID, []string{"OU=A,DC=example,DC=com", "OU=B,DC=example,DC=com"})
	assert.Equal(t, 1, len(got), "软删行不应被 getExistingOUs 命中(GORM 默认过滤)")
	assert.Equal(t, "OU=A,DC=example,DC=com", got[0].OUN)

	// 2) 部分命中
	got = svc.getExistingOUs(ctx, cfg.ID, []string{"OU=A,DC=example,DC=com", "OU=Z,DC=example,DC=com"})
	assert.Equal(t, 1, len(got))
	assert.Equal(t, "OU=A,DC=example,DC=com", got[0].OUN)

	// 3) 空列表
	got = svc.getExistingOUs(ctx, cfg.ID, []string{})
	assert.Equal(t, 0, len(got))
	assert.Empty(t, got)

	// 4) 不存在的 configID
	got = svc.getExistingOUs(ctx, "non-existent-cfg", []string{"OU=A,DC=example,DC=com"})
	assert.Equal(t, 0, len(got))
}

// TestSync78_CategorizeOUs_CreateVsUpdate 覆盖 categorizeOUs 主分支:
// 既有 OU → ousToUpdate map;新 OU → ousToCreate slice。
// 既验证字段提取(ou_name/description 取自 entry),也验证默认值兜底(属性缺失时)。
func TestSync78_CategorizeOUs_CreateVsUpdate(t *testing.T) {
	svc, db := newSyncSvc78(t)
	defer closeDB(t, db)
	ctx := context.Background()
	cfg := insertConfig78(t, db, "")

	// 预置 1 条 OU(ou_existing)
	now := time.Now()
	existing := &models.ADOU{
		ADConfigID: cfg.ID, OUN: "OU=Existing,DC=example,DC=com", OUName: "OldName",
		Description: "OldDesc", LastSyncAt: &now,
	}
	existing.ID = uuid.NewString()
	require.NoError(t, db.Create(existing).Error)

	entries := []*ldap.Entry{
		// 已存在 — 应进 update map
		entry78("OU=Existing,DC=example,DC=com", map[string][]string{
			"ou":         {"Existing"},
			"description": {"NewDesc"},
		}),
		// 新建 1 — 应进 create slice
		entry78("OU=NewOne,DC=example,DC=com", map[string][]string{
			"ou":         {"NewOne"},
			"description": {"NewOneDesc"},
		}),
		// 新建 2 — 属性缺失,默认值验证
		entry78("OU=NoDesc,DC=example,DC=com", map[string][]string{
			"ou": {"NoDesc"},
		}),
	}
	existingOUs := svc.getExistingOUs(ctx, cfg.ID, extractDNs(entries))
	nowLater := time.Now()
	toCreate, toUpdate := svc.categorizeOUs(entries, existingOUs, cfg, nowLater)

	// 1 update + 2 create
	assert.Equal(t, 2, len(toCreate), "新建 OU 应有 2 条")
	assert.Equal(t, 1, len(toUpdate), "既有 OU 应被分到 update map")

	// update 验证:ou_name/description 被新值覆盖 + LastSyncAt 更新
	upd, ok := toUpdate["OU=Existing,DC=example,DC=com"]
	require.True(t, ok)
	assert.Equal(t, "Existing", upd.OUName, "既有的 ou_name 被新 entry 覆盖")
	assert.Equal(t, "NewDesc", upd.Description)
	require.NotNil(t, upd.LastSyncAt)
	assert.Equal(t, nowLater.Unix(), upd.LastSyncAt.Unix())

	// create 验证:字段映射 + OUPath/ParentDN 由 extractParentDN/buildOUPath 推导
	var newOne, noDesc *models.ADOU
	for i := range toCreate {
		switch toCreate[i].OUN {
		case "OU=NewOne,DC=example,DC=com":
			newOne = &toCreate[i]
		case "OU=NoDesc,DC=example,DC=com":
			noDesc = &toCreate[i]
		}
	}
	require.NotNil(t, newOne)
	require.NotNil(t, noDesc)
	assert.Equal(t, cfg.ID, newOne.ADConfigID)
	assert.Equal(t, "NewOne", newOne.OUName)
	assert.Equal(t, "NewOneDesc", newOne.Description)
	assert.Equal(t, "DC=example,DC=com", newOne.ParentDN, "extractParentDN 应去掉 CN/OU 首段")
	assert.Equal(t, "/NewOne", newOne.OUPath, "buildOUPath 应只拼接 OU= 段")

	// 默认值兜底:description 缺失 → ""
	assert.Equal(t, "", noDesc.Description)
}

// TestSync78_SyncOUs_FullChain 覆盖 syncOUs 整合:空 entries 早退 +
// 1 条既有 + 2 条新 entry → 写库后断言总行数与字段更新。
func TestSync78_SyncOUs_FullChain(t *testing.T) {
	svc, db := newSyncSvc78(t)
	defer closeDB(t, db)
	ctx := context.Background()
	cfg := insertConfig78(t, db, "")

	// 预置 1 条既有 OU
	pre := &models.ADOU{ADConfigID: cfg.ID, OUN: "OU=Keep,DC=example,DC=com", OUName: "Old"}
	pre.ID = uuid.NewString()
	require.NoError(t, db.Create(pre).Error)

	// 1) 空 entries 早退
	require.NoError(t, svc.syncOUs(ctx, cfg, nil))
	require.NoError(t, svc.syncOUs(ctx, cfg, []*ldap.Entry{}))

	// 2) 3 条 entry(1 既有 + 2 新)
	entries := []*ldap.Entry{
		entry78("OU=Keep,DC=example,DC=com", map[string][]string{"ou": {"Keep"}, "description": {"Updated"}}),
		entry78("OU=New1,DC=example,DC=com", map[string][]string{"ou": {"New1"}}),
		entry78("OU=New2,DC=example,DC=com", map[string][]string{"ou": {"New2"}}),
	}
	require.NoError(t, svc.syncOUs(ctx, cfg, entries))

	// 总行数 3
	var count int64
	db.Model(&models.ADOU{}).Where("ad_config_id = ?", cfg.ID).Count(&count)
	assert.Equal(t, int64(3), count)

	// 既有行被更新(ou_name 保持 "Keep",description 变 "Updated")
	var updated models.ADOU
	require.NoError(t, db.Where("ou_dn = ? AND ad_config_id = ?", "OU=Keep,DC=example,DC=com", cfg.ID).First(&updated).Error)
	assert.Equal(t, "Updated", updated.Description)
	require.NotNil(t, updated.LastSyncAt, "LastSyncAt 应被刷新")

	// 新建 2 条
	var new1, new2 models.ADOU
	require.NoError(t, db.Where("ou_dn = ? AND ad_config_id = ?", "OU=New1,DC=example,DC=com", cfg.ID).First(&new1).Error)
	require.NoError(t, db.Where("ou_dn = ? AND ad_config_id = ?", "OU=New2,DC=example,DC=com", cfg.ID).First(&new2).Error)
	assert.Equal(t, "New1", new1.OUName)
	assert.Equal(t, "New2", new2.OUName)
}

// TestSync78_BatchCreateOUs_Empty_And_Batching 覆盖两条分支:
//   - 空切片早退
//   - 单批阈值(batchSize=500,sync.go:290)穿越:101 条应一次 Create 落库
func TestSync78_BatchCreateOUs_Empty_And_Batching(t *testing.T) {
	svc, db := newSyncSvc78(t)
	defer closeDB(t, db)
	ctx := context.Background()
	cfg := insertConfig78(t, db, "")

	// 空切片早退
	require.NoError(t, svc.batchCreateOUs(ctx, nil))
	require.NoError(t, svc.batchCreateOUs(ctx, []models.ADOU{}))

	// 超单批阈值(501 条 → 触发 batchSize=500 的两批循环,sync.go:290-296)
	ous := make([]models.ADOU, 0, 501)
	for i := 0; i < 501; i++ {
		ou := models.ADOU{
			ADConfigID: cfg.ID,
			OUN:        "OU=Batch" + itoaForTest(i) + ",DC=example,DC=com",
			OUName:     "Batch" + itoaForTest(i),
		}
		ou.ID = uuid.NewString()
		ous = append(ous, ou)
	}
	require.NoError(t, svc.batchCreateOUs(ctx, ous))

	var count int64
	db.Model(&models.ADOU{}).Where("ad_config_id = ?", cfg.ID).Count(&count)
	assert.Equal(t, int64(501), count, "501 条应全部落库(跨批次循环)")
}

// TestSync78_BatchUpdateOUs_Empty_And_Multi 覆盖 batchUpdateOUs:
//   - 空 map 早退
//   - 多条更新逐行断言字段
//   - clause.OnConflict upsert 语义:相同 (ad_config_id, ou_dn) 二次调用不产生重复
func TestSync78_BatchUpdateOUs_Empty_And_Multi(t *testing.T) {
	svc, db := newSyncSvc78(t)
	defer closeDB(t, db)
	ctx := context.Background()
	cfg := insertConfig78(t, db, "")

	// 空 map 早退
	require.NoError(t, svc.batchUpdateOUs(ctx, nil))
	require.NoError(t, svc.batchUpdateOUs(ctx, map[string]*models.ADOU{}))

	// 预置 3 行 OU
	dns := []string{
		"OU=U1,DC=example,DC=com",
		"OU=U2,DC=example,DC=com",
		"OU=U3,DC=example,DC=com",
	}
	for _, dn := range dns {
		ou := &models.ADOU{ADConfigID: cfg.ID, OUN: dn, OUName: "Old"}
		ou.ID = uuid.NewString()
		require.NoError(t, db.Create(ou).Error)
	}

	// 构造 update map(ou_name 改为 "New")
	now := time.Now()
	updates := make(map[string]*models.ADOU)
	var liveModels []models.ADOU
	require.NoError(t, db.Where("ad_config_id = ?", cfg.ID).Find(&liveModels).Error)
	for i := range liveModels {
		liveModels[i].OUName = "New"
		liveModels[i].LastSyncAt = &now
		updates[liveModels[i].OUN] = &liveModels[i]
	}
	require.NoError(t, svc.batchUpdateOUs(ctx, updates))

	// 逐行断言
	for _, dn := range dns {
		var row models.ADOU
		require.NoError(t, db.Where("ou_dn = ? AND ad_config_id = ?", dn, cfg.ID).First(&row).Error)
		assert.Equal(t, "New", row.OUName)
	}

	// 总行数 3(upsert 不应新增)
	var count int64
	db.Model(&models.ADOU{}).Where("ad_config_id = ?", cfg.ID).Count(&count)
	assert.Equal(t, int64(3), count)
}

// ============================================================================
// Task 2: Group 管道(syncGroups / createGroupsInBatches /
// updateGroupsInBatches / parseGroupTypeFromLDAP)
// ============================================================================

// TestSync78_ParseGroupTypeFromLDAP_Table 表驱动覆盖位掩码全枚举 +
// 未知 + 空 + 非数字字符串。零 DB,纯函数。
//
// AD groupType 值(sync.go:694-700 + parseGroupTypeFromLDAP 位运算):
//   - scope (低 28 位):2=Global, 4=Local, 8=Universal
//   - high bit (0x80000000):Security vs Distribution
//   - 0/未知值/非数字字符串 → 默认 Global + Security
func TestSync78_ParseGroupTypeFromLDAP_Table(t *testing.T) {
	cases := []struct {
		name      string
		input     string
		wantScope models.ADGroupScope
		wantType  models.ADGroupType
	}{
		// 6 标准组合:Security/Distribution × Global/Local/Universal
		{"global_security", "-2147483646", models.ADGroupScopeGlobal, models.ADGroupTypeSecurity},     // 0x80000002
		{"local_security", "-2147483644", models.ADGroupScopeLocal, models.ADGroupTypeSecurity},       // 0x80000004
		{"universal_security", "-2147483640", models.ADGroupScopeUniversal, models.ADGroupTypeSecurity}, // 0x80000008
		{"global_distribution", "2", models.ADGroupScopeGlobal, models.ADGroupTypeDistribution},       // 0x00000002
		{"local_distribution", "4", models.ADGroupScopeLocal, models.ADGroupTypeDistribution},         // 0x00000004
		{"universal_distribution", "8", models.ADGroupScopeUniversal, models.ADGroupTypeDistribution}, // 0x00000008
		// 边界/兜底
		{"empty_string", "", models.ADGroupScopeGlobal, models.ADGroupTypeSecurity}, // sync.go:702-704
		{"unknown_scope", "16", models.ADGroupScopeGlobal, models.ADGroupTypeDistribution}, // 16 & 0x0FFFFFFF=16 不在 case 中,default → Global;但 high bit=0 → Distribution
		{"non_numeric", "abc", models.ADGroupScopeGlobal, models.ADGroupTypeSecurity}, // parseIntOrDefault → -2147483646
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotScope, gotType := parseGroupTypeFromLDAP(tc.input)
			assert.Equal(t, tc.wantScope, gotScope, "scope mismatch")
			assert.Equal(t, tc.wantType, gotType, "type mismatch")
		})
	}
}

// TestSync78_SyncGroups_CreateAndUpdate 覆盖 syncGroups 主路径:
//   - 空 entries 早退
//   - 1 既有 + 2 新 → 创建 2 + 更新 1
//   - member_count 从 entry.GetAttributeValues("member") 长度推导
func TestSync78_SyncGroups_CreateAndUpdate(t *testing.T) {
	svc, db := newSyncSvc78(t)
	defer closeDB(t, db)
	ctx := context.Background()
	cfg := insertConfig78(t, db, "")

	// 空 entries 早退
	require.NoError(t, svc.syncGroups(ctx, cfg, nil))
	require.NoError(t, svc.syncGroups(ctx, cfg, []*ldap.Entry{}))

	// 预置 1 条 group(同名 group_dn)
	existing := &models.ADGroup{
		ADConfigID: cfg.ID,
		GroupDN:    "CN=ExistingGroup,OU=Groups,DC=example,DC=com",
		GroupName:  "OldName",
		GroupScope: models.ADGroupScopeGlobal,
		GroupType:  models.ADGroupTypeSecurity,
	}
	existing.ID = uuid.NewString()
	require.NoError(t, db.Create(existing).Error)

	// 3 条 entry
	entries := []*ldap.Entry{
		// 既有 — 应被更新
		entry78("CN=ExistingGroup,OU=Groups,DC=example,DC=com", map[string][]string{
			"cn":         {"ExistingGroup"},
			"description": {"UpdatedDesc"},
			"groupType":  {"-2147483644"}, // LocalSecurity
			"member":     {"CN=u1,DC=example,DC=com", "CN=u2,DC=example,DC=com", "CN=u3,DC=example,DC=com"},
		}),
		// 新建 1 — 带 3 个 member 验证 member_count=3
		entry78("CN=NewGroup1,OU=Groups,DC=example,DC=com", map[string][]string{
			"cn":         {"NewGroup1"},
			"description": {"NG1Desc"},
			"groupType":  {"-2147483646"}, // GlobalSecurity
			"member":     {"CN=a,DC=example,DC=com", "CN=b,DC=example,DC=com", "CN=c,DC=example,DC=com"},
		}),
		// 新建 2 — 缺 member 属性 → member_count=0;缺 description → ""
		entry78("CN=NewGroup2,OU=Groups,DC=example,DC=com", map[string][]string{
			"cn":        {"NewGroup2"},
			"groupType": {"-2147483640"}, // UniversalSecurity
		}),
	}
	require.NoError(t, svc.syncGroups(ctx, cfg, entries))

	// 总行数 3
	var count int64
	db.Model(&models.ADGroup{}).Where("ad_config_id = ?", cfg.ID).Count(&count)
	assert.Equal(t, int64(3), count)

	// 既有行被更新(description/GroupScope/GroupType/MemberCount)
	var updated models.ADGroup
	require.NoError(t, db.Where("group_dn = ? AND ad_config_id = ?", "CN=ExistingGroup,OU=Groups,DC=example,DC=com", cfg.ID).First(&updated).Error)
	assert.Equal(t, "ExistingGroup", updated.GroupName)
	assert.Equal(t, "UpdatedDesc", updated.Description)
	assert.Equal(t, models.ADGroupScopeLocal, updated.GroupScope)
	assert.Equal(t, models.ADGroupTypeSecurity, updated.GroupType)
	assert.Equal(t, 3, updated.MemberCount, "member_count 应为 member 属性值数")
	assert.Equal(t, "OU=Groups,DC=example,DC=com", updated.OUN)
	require.NotNil(t, updated.LastSyncAt)

	// 新建 1:member_count=3, scope=Global, type=Security
	var ng1 models.ADGroup
	require.NoError(t, db.Where("group_dn = ? AND ad_config_id = ?", "CN=NewGroup1,OU=Groups,DC=example,DC=com", cfg.ID).First(&ng1).Error)
	assert.Equal(t, 3, ng1.MemberCount)
	assert.Equal(t, models.ADGroupScopeGlobal, ng1.GroupScope)
	assert.Equal(t, models.ADGroupTypeSecurity, ng1.GroupType)

	// 新建 2:member_count=0(无 member 属性), description=""
	var ng2 models.ADGroup
	require.NoError(t, db.Where("group_dn = ? AND ad_config_id = ?", "CN=NewGroup2,OU=Groups,DC=example,DC=com", cfg.ID).First(&ng2).Error)
	assert.Equal(t, 0, ng2.MemberCount)
	assert.Equal(t, "", ng2.Description)
}

// TestSync78_SyncGroups_MemberSync 验证 syncGroups 内联调用 syncGroupMembers
// (sync.go:373-377):group entry 带 member 属性 → sys_ad_group_member 行落库。
func TestSync78_SyncGroups_MemberSync(t *testing.T) {
	svc, db := newSyncSvc78(t)
	defer closeDB(t, db)
	ctx := context.Background()
	cfg := insertConfig78(t, db, "")

	entries := []*ldap.Entry{
		entry78("CN=G1,OU=Groups,DC=example,DC=com", map[string][]string{
			"cn":        {"G1"},
			"groupType": {"-2147483646"},
			"member": {
				"CN=alice,DC=example,DC=com",
				"CN=bob,DC=example,DC=com",
			},
		}),
	}
	require.NoError(t, svc.syncGroups(ctx, cfg, entries))

	// sys_ad_group_member 应有 2 行(同 ad_config_id+group_dn,user_dn 各异)
	var members []models.ADGroupMember
	require.NoError(t, db.Where("ad_config_id = ? AND group_dn = ?", cfg.ID, "CN=G1,OU=Groups,DC=example,DC=com").Find(&members).Error)
	assert.Equal(t, 2, len(members))
	userDNs := []string{members[0].UserDN, members[1].UserDN}
	assert.ElementsMatch(t, []string{"CN=alice,DC=example,DC=com", "CN=bob,DC=example,DC=com"}, userDNs)
}

// TestSync78_CreateGroupsInBatches_Empty_And_Batching 覆盖:
//   - 空切片早退
//   - 501 条触发 batchSize=500 双批
//   - 冲突分支:同 (ad_config_id, group_dn) 预置后再 Create → 不报错,upsert 行为
//     (createGroupsInBatches 用 s.db.Create(),无 clause.OnConflict;冲突会让
//     GORM 返回 unique constraint error。文档化:这是已知失败分支。)
func TestSync78_CreateGroupsInBatches_Empty_And_Batching(t *testing.T) {
	svc, db := newSyncSvc78(t)
	defer closeDB(t, db)
	ctx := context.Background()
	cfg := insertConfig78(t, db, "")

	// 空切片早退
	require.NoError(t, svc.createGroupsInBatches(ctx, nil))
	require.NoError(t, svc.createGroupsInBatches(ctx, []models.ADGroup{}))

	// 501 条触发多批
	groups := make([]models.ADGroup, 0, 501)
	for i := 0; i < 501; i++ {
		g := models.ADGroup{
			ADConfigID: cfg.ID,
			GroupDN:    "CN=B" + itoaForTest(i) + ",DC=example,DC=com",
			GroupName:  "B" + itoaForTest(i),
			GroupScope: models.ADGroupScopeGlobal,
			GroupType:  models.ADGroupTypeSecurity,
		}
		g.ID = uuid.NewString()
		groups = append(groups, g)
	}
	require.NoError(t, svc.createGroupsInBatches(ctx, groups))

	var count int64
	db.Model(&models.ADGroup{}).Where("ad_config_id = ?", cfg.ID).Count(&count)
	assert.Equal(t, int64(501), count)

	// 冲突分支(无 OnConflict,直接 Create → unique constraint 失败)
	// 文档化:这是 sync.go 现状行为,createGroupsInBatches 不处理冲突。
	dup := groups[0] // 同 (ad_config_id, group_dn)
	err := svc.createGroupsInBatches(ctx, []models.ADGroup{dup})
	assert.Error(t, err, "createGroupsInBatches 无 OnConflict → 重复 DN 应触发 unique constraint 错误")
}

// TestSync78_UpdateGroupsInBatches_Empty_And_Multi 覆盖:
//   - 空 map 早退
//   - 多条更新逐行断言 + OnConflict upsert 不产生重复行
func TestSync78_UpdateGroupsInBatches_Empty_And_Multi(t *testing.T) {
	svc, db := newSyncSvc78(t)
	defer closeDB(t, db)
	ctx := context.Background()
	cfg := insertConfig78(t, db, "")

	// 空 map 早退
	require.NoError(t, svc.updateGroupsInBatches(ctx, nil))
	require.NoError(t, svc.updateGroupsInBatches(ctx, map[string]*models.ADGroup{}))

	// 预置 3 条 group
	dns := []string{
		"CN=GU1,OU=Groups,DC=example,DC=com",
		"CN=GU2,OU=Groups,DC=example,DC=com",
		"CN=GU3,OU=Groups,DC=example,DC=com",
	}
	for _, dn := range dns {
		g := &models.ADGroup{
			ADConfigID: cfg.ID, GroupDN: dn, GroupName: "Old",
			GroupScope: models.ADGroupScopeGlobal, GroupType: models.ADGroupTypeSecurity,
		}
		g.ID = uuid.NewString()
		require.NoError(t, db.Create(g).Error)
	}

	// 构造 update map(group_name 改为 "New", member_count=10)
	now := time.Now()
	updates := make(map[string]*models.ADGroup)
	var live []models.ADGroup
	require.NoError(t, db.Where("ad_config_id = ?", cfg.ID).Find(&live).Error)
	for i := range live {
		live[i].GroupName = "New"
		live[i].MemberCount = 10
		live[i].LastSyncAt = &now
		updates[live[i].GroupDN] = &live[i]
	}
	require.NoError(t, svc.updateGroupsInBatches(ctx, updates))

	// 逐行断言 + 总行数 3(upsert 不增行)
	var count int64
	db.Model(&models.ADGroup{}).Where("ad_config_id = ?", cfg.ID).Count(&count)
	assert.Equal(t, int64(3), count)
	for _, dn := range dns {
		var row models.ADGroup
		require.NoError(t, db.Where("group_dn = ? AND ad_config_id = ?", dn, cfg.ID).First(&row).Error)
		assert.Equal(t, "New", row.GroupName)
		assert.Equal(t, 10, row.MemberCount)
	}
}

// TestSync78_SyncGroups_DBError 覆盖 syncGroups DB 错误早退分支:
// DROP TABLE sys_ad_group → GetOrCreate 链应包装错误返回而非 panic。
func TestSync78_SyncGroups_DBError(t *testing.T) {
	svc, db := newSyncSvc78(t)
	defer closeDB(t, db)
	ctx := context.Background()
	cfg := insertConfig78(t, db, "")

	// DROP 之前需要关掉 sqlite db 连接,DROP TABLE 才会真正生效
	sqlDB, err := db.DB()
	require.NoError(t, err)
	_, err = sqlDB.Exec(`DROP TABLE sys_ad_group`)
	require.NoError(t, err)

	entries := []*ldap.Entry{
		entry78("CN=X,OU=Groups,DC=example,DC=com", map[string][]string{"cn": {"X"}}),
	}
	err = svc.syncGroups(ctx, cfg, entries)
	assert.Error(t, err, "DROP TABLE 后应返回包装错误而非 panic")
}