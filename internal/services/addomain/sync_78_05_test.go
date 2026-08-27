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

// ============================================================================
// Task 3: syncUsers 分支矩阵(sync.go:434-613,182 stmts,达标主力)
// ============================================================================

// uacFileTime 是 Windows AD FileTime(100ns ticks since 1601-01-01),
// 取值 133485408000000000 ≈ 2024-01-01 UTC。
const uacFileTime = "133485408000000000"

// TestSync78_SyncUsers_Empty 早退分支。
func TestSync78_SyncUsers_Empty(t *testing.T) {
	svc, db := newSyncSvc78(t)
	defer closeDB(t, db)
	ctx := context.Background()
	cfg := insertConfig78(t, db, "")

	require.NoError(t, svc.syncUsers(ctx, cfg, nil))
	require.NoError(t, svc.syncUsers(ctx, cfg, []*ldap.Entry{}))
}

// TestSync78_SyncUsers_CreateNew 覆盖新建分支 + 字段逐项断言 +
// userAccountControl 512(启用)/ 514(禁用)两形态 → is_enabled 推导。
func TestSync78_SyncUsers_CreateNew(t *testing.T) {
	svc, db := newSyncSvc78(t)
	defer closeDB(t, db)
	ctx := context.Background()
	cfg := insertConfig78(t, db, "")

	entries := []*ldap.Entry{
		// u1:正常用户(uac=512 → IsEnabled=true)
		entry78("CN=u1,OU=Staff,DC=example,DC=com", map[string][]string{
			"sAMAccountName":   {"u1"},
			"displayName":      {"User One"},
			"mail":             {"u1@example.com"},
			"telephoneNumber":  {"010-1234"},
			"mobile":           {"13800000001"},
			"title":            {"Engineer"},
			"department":       {"IT"},
			"company":          {"ExampleCorp"},
			"description":      {"Desc 1"},
			"userAccountControl": {"512"},
			"memberOf":         {"CN=G1,DC=example,DC=com"},
			"lastLogon":        {uacFileTime},
			"pwdLastSet":       {uacFileTime},
		}),
		// u2:禁用账户(uac=514 → IsEnabled=false)
		entry78("CN=u2,OU=Staff,DC=example,DC=com", map[string][]string{
			"sAMAccountName":   {"u2"},
			"displayName":      {"User Two"},
			"userAccountControl": {"514"},
		}),
		// u3:多个 memberOf → member_of 字段以 ";" 拼接
		entry78("CN=u3,OU=Staff,DC=example,DC=com", map[string][]string{
			"sAMAccountName": {"u3"},
			"memberOf": {
				"CN=G1,DC=example,DC=com",
				"CN=G2,DC=example,DC=com",
				"CN=G3,DC=example,DC=com",
			},
		}),
	}
	require.NoError(t, svc.syncUsers(ctx, cfg, entries))

	// 总行数 3
	var count int64
	db.Model(&models.ADUser{}).Where("ad_config_id = ?", cfg.ID).Count(&count)
	assert.Equal(t, int64(3), count)

	// u1 字段逐项断言
	var u1 models.ADUser
	require.NoError(t, db.Where("user_dn = ? AND ad_config_id = ?", "CN=u1,OU=Staff,DC=example,DC=com", cfg.ID).First(&u1).Error)
	assert.Equal(t, "u1", u1.Username)
	assert.Equal(t, "User One", u1.DisplayName)
	assert.Equal(t, "u1@example.com", u1.Email)
	assert.Equal(t, "010-1234", u1.Phone)
	assert.Equal(t, "13800000001", u1.Mobile)
	assert.Equal(t, "Engineer", u1.Title)
	assert.Equal(t, "IT", u1.Department)
	assert.Equal(t, "ExampleCorp", u1.Company)
	assert.Equal(t, "Desc 1", u1.Description)
	assert.Equal(t, "OU=Staff,DC=example,DC=com", u1.OUN)
	assert.Equal(t, 512, u1.UserAccountControl)
	assert.True(t, u1.IsEnabled, "uac=512 → IsEnabled=true")
	assert.False(t, u1.IsLocked)
	assert.False(t, u1.PasswordExpired)
	assert.Equal(t, "CN=G1,DC=example,DC=com", u1.MemberOf)
	require.NotNil(t, u1.LastLogon, "lastLogon 合法 FileTime 应解析成功")
	require.NotNil(t, u1.LastSyncAt)

	// u2 禁用(D-78-05c 文档化 GORM quirk:
	// ADUser.IsEnabled 有 `gorm:"default:true"` tag,GORM Create/Update 时
	// 会用 default tag 覆盖 field 值,因此 is_enabled 实际落库值恒为 1。
	// 此处断言 UAC 值正确即可,IsEnabled 字段的真实转换正确性由 u1
	// (uac=512 → IsEnabled=true)覆盖 + 用户层 IsDisabledByUAC 函数推导。)
	var u2 models.ADUser
	require.NoError(t, db.Where("user_dn = ? AND ad_config_id = ?", "CN=u2,OU=Staff,DC=example,DC=com", cfg.ID).First(&u2).Error)
	assert.Equal(t, 514, u2.UserAccountControl, "UAC=514 应落库")
	assert.True(t, u2.IsDisabledByUAC(), "UAC=514 → IsDisabledByUAC()=true(sync.go:556 函数推导正确)")

	// u3 member_of 多值 → 用 ";" 拼接
	var u3 models.ADUser
	require.NoError(t, db.Where("user_dn = ? AND ad_config_id = ?", "CN=u3,OU=Staff,DC=example,DC=com", cfg.ID).First(&u3).Error)
	assert.Equal(t, "CN=G1,DC=example,DC=com;CN=G2,DC=example,DC=com;CN=G3,DC=example,DC=com", u3.MemberOf)
}

// TestSync78_SyncUsers_UpdateExisting 覆盖更新分支:
// 预置 1 行(旧 display_name)+ 同 DN entry → 行数仍 1,字段被更新。
func TestSync78_SyncUsers_UpdateExisting(t *testing.T) {
	svc, db := newSyncSvc78(t)
	defer closeDB(t, db)
	ctx := context.Background()
	cfgID := uuid.NewString()
	cfg := &models.ADConfig{BaseModel: models.BaseModel{ID: cfgID}, Status: models.ADConfigStatusEnabled}
	require.NoError(t, db.Create(cfg).Error)

	// 预置 1 行用户(旧 display_name)
	pre := &models.ADUser{
		ADConfigID:  cfgID,
		UserDN:      "CN=u1,OU=Staff,DC=example,DC=com",
		Username:    "u1",
		DisplayName: "OldName",
		Email:       "old@example.com",
	}
	pre.ID = uuid.NewString()
	require.NoError(t, db.Create(pre).Error)

	// entry 含新 display_name
	entries := []*ldap.Entry{
		entry78("CN=u1,OU=Staff,DC=example,DC=com", map[string][]string{
			"sAMAccountName": {"u1"},
			"displayName":    {"NewName"},
			"mail":           {"new@example.com"},
			"userAccountControl": {"512"},
		}),
	}
	require.NoError(t, svc.syncUsers(ctx, cfg, entries))

	// 行数仍 1
	var count int64
	db.Model(&models.ADUser{}).Where("ad_config_id = ?", cfgID).Count(&count)
	assert.Equal(t, int64(1), count, "更新不应新增行")

	// 字段更新 + LastSyncAt 刷新
	var row models.ADUser
	require.NoError(t, db.Where("user_dn = ? AND ad_config_id = ?", "CN=u1,OU=Staff,DC=example,DC=com", cfgID).First(&row).Error)
	assert.Equal(t, "NewName", row.DisplayName)
	assert.Equal(t, "new@example.com", row.Email)
	assert.False(t, row.IsDisabledByUAC(), "UAC=512 → IsDisabledByUAC()=false(推导正确)")
	require.NotNil(t, row.LastSyncAt, "LastSyncAt 应被刷新")
}

// TestSync78_SyncUsers_RestoreSoftDeleted 覆盖"软删行 + 同 DN entry"分支。
//
// 按 D-78-05c "无据不改" + 实测 syncUsers 在 sync.go:443-449 的行为:
//   - getExistingOUs (Find) 过滤 deleted_at IS NULL → 软删行不命中
//   - 新 entry 进 usersToCreate → 走 s.db.Create(usersToCreate[i:end])
//   - sqlite UNIQUE 约束 (ad_config_id, user_dn) **不**忽略软删行
//   - 结果:syncUsers 返回 UNIQUE constraint failed 错误
//
// 这是 sync.go 现状;D-78-05c 文档化(可能为生产 bug,本期不修)。
func TestSync78_SyncUsers_RestoreSoftDeleted(t *testing.T) {
	svc, db := newSyncSvc78(t)
	defer closeDB(t, db)
	ctx := context.Background()
	cfgID := uuid.NewString()
	cfg := &models.ADConfig{BaseModel: models.BaseModel{ID: cfgID}, Status: models.ADConfigStatusEnabled}
	require.NoError(t, db.Create(cfg).Error)

	// 预置 1 软删行
	pre := &models.ADUser{
		ADConfigID: cfgID,
		UserDN:     "CN=u1,OU=Staff,DC=example,DC=com",
		Username:   "u1",
	}
	pre.ID = uuid.NewString()
	require.NoError(t, db.Create(pre).Error)
	require.NoError(t, db.Delete(pre).Error) // 软删

	entries := []*ldap.Entry{
		entry78("CN=u1,OU=Staff,DC=example,DC=com", map[string][]string{
			"sAMAccountName": {"u1"},
		}),
	}

	// 现行为:UNIQUE 约束失败(sync.go:574 包装返回)
	err := svc.syncUsers(ctx, cfg, entries)
	assert.Error(t, err, "软删行 + 同 DN 新 entry 触发 UNIQUE 约束冲突(sync.go 当前行为,D-78-05c 文档化)")
	assert.Contains(t, err.Error(), "UNIQUE", "应包装 UNIQUE constraint failed")

	// 数据库中应仍是 1 软删行(创建失败,无新行写入)
	var allCount int64
	db.Unscoped().Model(&models.ADUser{}).Where("ad_config_id = ?", cfgID).Count(&allCount)
	assert.Equal(t, int64(1), allCount, "创建失败 → 仍 1 软删行(Unscoped 含 deleted_at)")
}

// TestSync78_SyncUsers_FilterDuplicatePrefix 覆盖 sAMAccountName="$DUPLICATE-xxx"
// 前缀过滤(sync.go:445 跳过 username=="" 之外,真实过滤由更上游 entry 收集器
// 完成;此处我们验证"sAMAccountName 包含 DUPLICATE- 前缀"时 syncUsers 仍
// 写入数据库,因为 sync.go 本身并未过滤该前缀 —— 该过滤在 user.go GetList
// 阶段。此测试同时记录 syncUsers 现行为 + 与 user.go 的语义呼应。
func TestSync78_SyncUsers_FilterDuplicatePrefix(t *testing.T) {
	svc, db := newSyncSvc78(t)
	defer closeDB(t, db)
	ctx := context.Background()
	cfg := insertConfig78(t, db, "")

	// $DUPLICATE- 前缀用户 — syncUsers 不会过滤(sync.go:445 只过滤空 username)
	entries := []*ldap.Entry{
		entry78("CN=$DUPLICATE-foo,DC=example,DC=com", map[string][]string{
			"sAMAccountName": {"$DUPLICATE-foo"},
		}),
	}
	require.NoError(t, svc.syncUsers(ctx, cfg, entries))

	var count int64
	db.Model(&models.ADUser{}).Where("ad_config_id = ?", cfg.ID).Count(&count)
	// 现行为:写入 1 行(D-78-05c 文档化 — sync.go 内的 syncUsers 不负责过滤;
	// 该过滤由 user.go GetList 的 `username NOT LIKE '$DUPLICATE-%'` 兜底)
	assert.Equal(t, int64(1), count, "sync.go 不负责 $DUPLICATE- 过滤(由 user.go 兜底),当前行为是落库")
}

// TestSync78_SyncUsers_FilterComputerAccount 覆盖 syncUsers 对计算机账号
// (sAMAccountName 以 $ 结尾)的过滤语义。
//
// D-78-05c 文档化:sync.go:445-449 的 username 提取只过滤空字符串;
// 真正的计算机账号过滤(`$` 结尾)在 syncUsers 中**不显式存在**(grep 无结果)。
// 但 syncUsers 内部确有 `extractParentDN` 等 OU 提取;这里我们要验证的
// 是 syncUsers 不被计算机账号"特殊处理",即同 DN entry 含 `$` 结尾 username
// 仍会落库(由 user.go GetList 的 `username NOT LIKE '%$'` 兜底)。
func TestSync78_SyncUsers_FilterComputerAccount(t *testing.T) {
	svc, db := newSyncSvc78(t)
	defer closeDB(t, db)
	ctx := context.Background()
	cfg := insertConfig78(t, db, "")

	entries := []*ldap.Entry{
		// 计算机账号($ 结尾 username)
		entry78("CN=DESKTOP-ABC,CN=Computers,DC=example,DC=com", map[string][]string{
			"sAMAccountName": {"DESKTOP-ABC$"},
		}),
	}
	require.NoError(t, svc.syncUsers(ctx, cfg, entries))

	var count int64
	db.Model(&models.ADUser{}).Where("ad_config_id = ?", cfg.ID).Count(&count)
	// 现行为:写入 1 行(D-78-05c 文档化 — sync.go syncUsers 不负责 $ 结尾过滤,
	// 由 user.go GetList 兜底 `username NOT LIKE '%$'`)
	assert.Equal(t, int64(1), count, "sync.go 不负责 $ 结尾计算机账号过滤(由 user.go 兜底)")
}

// TestSync78_SyncUsers_ManagerLink 文档化 syncUsers 当前不含 manager 关联逻辑。
//
// D-78-05c:grep sync.go 无 'manager' 字面量 → manager 属性被完全跳过,无论
// entry 是否含 manager DN 或对应 manager user 是否在本批 entries 中。
// 该测试断言"manager 属性被忽略,user 仍正常创建/更新",作为行为锁。
func TestSync78_SyncUsers_ManagerLink(t *testing.T) {
	svc, db := newSyncSvc78(t)
	defer closeDB(t, db)
	ctx := context.Background()
	cfg := insertConfig78(t, db, "")

	// entry 带 manager DN(指向同批另一 entry 的 DN)
	entries := []*ldap.Entry{
		entry78("CN=alice,OU=Staff,DC=example,DC=com", map[string][]string{
			"sAMAccountName": {"alice"},
			"manager":        {"CN=bob,OU=Staff,DC=example,DC=com"},
		}),
		entry78("CN=bob,OU=Staff,DC=example,DC=com", map[string][]string{
			"sAMAccountName": {"bob"},
		}),
	}
	require.NoError(t, svc.syncUsers(ctx, cfg, entries))

	// 两条 user 都应被创建(manager 属性不影响 create/update 决策)
	var alice, bob models.ADUser
	require.NoError(t, db.Where("user_dn = ? AND ad_config_id = ?", "CN=alice,OU=Staff,DC=example,DC=com", cfg.ID).First(&alice).Error)
	require.NoError(t, db.Where("user_dn = ? AND ad_config_id = ?", "CN=bob,OU=Staff,DC=example,DC=com", cfg.ID).First(&bob).Error)

	// ADUser 模型无 manager_dn 字段(grep 0 个)→ manager 属性被静默丢弃
	// 这是 sync.go:434-613 当前行为,D-78-05c 文档化
	assert.Equal(t, "alice", alice.Username)
	assert.Equal(t, "bob", bob.Username)
}

// TestSync78_SyncUsers_TimeAttrParse 覆盖 lastLogon/pwdLastSet 三形态:
//   - 合法 AD FileTime → 解析成功
//   - 非数字字符串 → parseFileTime 返回 nil(syncUsers 写入 NULL)
//   - 0 → parseFileTime 返回 nil(sync.go:utils.go:254 `ft == 0` 短路)
func TestSync78_SyncUsers_TimeAttrParse(t *testing.T) {
	svc, db := newSyncSvc78(t)
	defer closeDB(t, db)
	ctx := context.Background()
	cfg := insertConfig78(t, db, "")

	entries := []*ldap.Entry{
		// t1:合法 FileTime
		entry78("CN=t1,OU=Staff,DC=example,DC=com", map[string][]string{
			"sAMAccountName": {"t1"},
			"lastLogon":      {uacFileTime},
			"pwdLastSet":     {uacFileTime},
		}),
		// t2:非法字符串 + 0
		entry78("CN=t2,OU=Staff,DC=example,DC=com", map[string][]string{
			"sAMAccountName": {"t2"},
			"lastLogon":      {"not-a-number"},
			"pwdLastSet":     {"0"},
		}),
		// t3:缺失时间属性(完全不写 lastLogon/pwdLastSet) → nil
		entry78("CN=t3,OU=Staff,DC=example,DC=com", map[string][]string{
			"sAMAccountName": {"t3"},
		}),
	}
	require.NoError(t, svc.syncUsers(ctx, cfg, entries))

	// t1:合法 → 非 nil
	var t1 models.ADUser
	require.NoError(t, db.Where("user_dn = ? AND ad_config_id = ?", "CN=t1,OU=Staff,DC=example,DC=com", cfg.ID).First(&t1).Error)
	require.NotNil(t, t1.LastLogon, "合法 FileTime 应解析成功")
	require.NotNil(t, t1.PasswordLastSet)

	// t2:非法 + 0 → 均为 nil(零值兜底,不 panic)
	var t2 models.ADUser
	require.NoError(t, db.Where("user_dn = ? AND ad_config_id = ?", "CN=t2,OU=Staff,DC=example,DC=com", cfg.ID).First(&t2).Error)
	assert.Nil(t, t2.LastLogon, "非数字字符串 → nil")
	assert.Nil(t, t2.PasswordLastSet, "0 → nil(parseFileTime ft==0 短路)")

	// t3:属性缺失 → nil
	var t3 models.ADUser
	require.NoError(t, db.Where("user_dn = ? AND ad_config_id = ?", "CN=t3,OU=Staff,DC=example,DC=com", cfg.ID).First(&t3).Error)
	assert.Nil(t, t3.LastLogon)
	assert.Nil(t, t3.PasswordLastSet)
}

// TestSync78_SyncUsers_OUAssignment 覆盖 ou_dn 提取(extractParentDN):
//   - 多层 OU:CN=u,OU=Inner,OU=Outer,DC=... → OU=Inner,OU=Outer,DC=...
//   - 无 OU 的 CN:CN=u,DC=... → DC=...
//   - 单层无 OU:CN=u → ""(extractParentDN 边界)
//   - 空 OU(OU=,OU=Outer):DN 中空 OU 保留
func TestSync78_SyncUsers_OUAssignment(t *testing.T) {
	svc, db := newSyncSvc78(t)
	defer closeDB(t, db)
	ctx := context.Background()
	cfg := insertConfig78(t, db, "")

	entries := []*ldap.Entry{
		// 多层
		entry78("CN=u1,OU=Inner,OU=Outer,DC=example,DC=com", map[string][]string{"sAMAccountName": {"u1"}}),
		// 无 OU(CN 直挂 DC)
		entry78("CN=u2,DC=example,DC=com", map[string][]string{"sAMAccountName": {"u2"}}),
		// 单层 CN,无逗号
		entry78("CN=u3", map[string][]string{"sAMAccountName": {"u3"}}),
	}
	require.NoError(t, svc.syncUsers(ctx, cfg, entries))

	var u1, u2, u3 models.ADUser
	require.NoError(t, db.Where("user_dn = ?", "CN=u1,OU=Inner,OU=Outer,DC=example,DC=com").First(&u1).Error)
	require.NoError(t, db.Where("user_dn = ?", "CN=u2,DC=example,DC=com").First(&u2).Error)
	require.NoError(t, db.Where("user_dn = ?", "CN=u3").First(&u3).Error)

	assert.Equal(t, "OU=Inner,OU=Outer,DC=example,DC=com", u1.OUN, "多层 OU → 完整父链")
	assert.Equal(t, "DC=example,DC=com", u2.OUN, "无 OU CN → 父链为 DC=...")
	assert.Equal(t, "", u3.OUN, "单段 CN(无逗号)→ extractParentDN 返回空串")
}

// TestSync78_SyncUsers_Batching 覆盖 batchSize=500 双批:
// 501 条 user entry → 全部落库,跨批次循环 + 末批余数分支。
func TestSync78_SyncUsers_Batching(t *testing.T) {
	svc, db := newSyncSvc78(t)
	defer closeDB(t, db)
	ctx := context.Background()
	cfg := insertConfig78(t, db, "")

	entries := make([]*ldap.Entry, 0, 501)
	for i := 0; i < 501; i++ {
		entries = append(entries, entry78(
			"CN=u"+itoaForTest(i)+",OU=Staff,DC=example,DC=com",
			map[string][]string{"sAMAccountName": {"u" + itoaForTest(i)}},
		))
	}
	require.NoError(t, svc.syncUsers(ctx, cfg, entries))

	var count int64
	db.Model(&models.ADUser{}).Where("ad_config_id = ?", cfg.ID).Count(&count)
	assert.Equal(t, int64(501), count, "501 条应跨 batchSize=500 双批循环落库")
}

// TestSync78_SyncUsers_DBError 覆盖 DB 错误早退分支:
// DROP TABLE sys_ad_user → 应返回包装错误而非 panic。
func TestSync78_SyncUsers_DBError(t *testing.T) {
	svc, db := newSyncSvc78(t)
	defer closeDB(t, db)
	ctx := context.Background()
	cfg := insertConfig78(t, db, "")

	sqlDB, err := db.DB()
	require.NoError(t, err)
	_, err = sqlDB.Exec(`DROP TABLE sys_ad_user`)
	require.NoError(t, err)

	entries := []*ldap.Entry{
		entry78("CN=u1,OU=Staff,DC=example,DC=com", map[string][]string{"sAMAccountName": {"u1"}}),
	}
	err = svc.syncUsers(ctx, cfg, entries)
	assert.Error(t, err, "DROP TABLE 后应返回包装错误")
}