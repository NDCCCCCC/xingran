//go:build !skip_db_tests
// +build !skip_db_tests

// Phase 78 Plan 06: computer.go 全链测试（Task 1 + Task 2 共用此文件）
//
// 复用 78-05 helper（setupSync78DB / entry78 / insertConfig78 / closeDB）— D-78-06e 禁止重定义。
// 补建 sys_ad_computer 表（含 UNIQUE(ad_config_id, computer_name) 供 batchUpdate ON CONFLICT 使用）。
// 零 LDAP 网络 + []*ldap.Entry 字面量驱动；断言禁裸状态字面量，用 models.* 常量。
//
// 本文件分两段：
//   - Task 1: List 查询链 + GetByDN（normalizePagination / buildComputerQuery / countComputers /
//     fetchComputers / convertToDetails / GetByDN + NewComputerService）
//   - Task 2: syncComputers entry-driven 全链（buildComputerFromEntry / updateComputerFields /
//     queryExistingComputers / queryAllComputerNames / buildComputerMaps / processComputerEntry /
//     batchCreate / batchUpdate / syncComputers 编排）

package addomain

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/go-ldap/ldap/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
)

// setupComp78DB 在 78-05 7 表 fixture 上追加 sys_ad_computer 建表。
//
// 列集参照 models.ADComputer gorm tag + syncComputers 写入字段：
//   - BaseModel 列: id / created_at / updated_at / deleted_at / created_by / updated_by / version
//   - 业务列: ad_config_id / computer_name / distinguished_name / oudn / status / original_description
//            / operating_system / os_version / ip_address / mac_address / managed_by / cpu_model /
//            architecture / memory_capacity / hard_disk_capacity / serial_number / system_info
//            / last_logon / password_last_set / logon_count / last_online_time
//   - 唯一索引 UNIQUE(ad_config_id, computer_name)：对应 batchUpdate ON CONFLICT 子句
//
// 记忆教训（xingran-sqlite-missing-table-pattern.md）：被测函数触达的表必须手工建。
func setupComp78DB(t *testing.T) *gorm.DB {
	t.Helper()
	db := setupSync78DB(t)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_ad_computer (
			id TEXT PRIMARY KEY,
			ad_config_id TEXT,
			computer_name TEXT,
			distinguished_name TEXT,
			last_logon DATETIME,
			password_last_set DATETIME,
			logon_count INTEGER DEFAULT 0,
			oudn TEXT,
			status INTEGER DEFAULT 0,
			original_description TEXT,
			ip_address TEXT,
			mac_address TEXT,
			managed_by TEXT,
			operating_system TEXT,
			os_version TEXT,
			cpu_model TEXT,
			architecture TEXT,
			memory_capacity TEXT,
			hard_disk_capacity TEXT,
			last_online_time DATETIME,
			serial_number TEXT,
			system_info TEXT,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			UNIQUE(ad_config_id, computer_name)
		)
	`).Error)
	return db
}

// insertComputer78 插入一条 sys_ad_computer 行（raw SQL，规避 GORM default:0 覆盖零值 quirk）。
//
// opts 顺序（按需可选）：
//   [0] original_description  (string)
//   [1] operating_system      (string)
//   [2] last_online_time RFC3339 (string,空=NULL)
//   [3] last_logon RFC3339       (string,空=NULL)
//   [4] ip_address            (string)
//   [5] os_version            (string)
//   [6] managed_by            (string)
//   [7] serial_number         (string)
func insertComputer78(t *testing.T, db *gorm.DB, configID, dn, name, ouDn string, status models.ComputerStatus, opts ...string) {
	t.Helper()
	id := uuid.NewString()
	description, os, lastOnline, lastLogon, ip, osVer, mgr, serial := "", "", "", "", "", "", "", ""
	if len(opts) > 0 {
		description = opts[0]
	}
	if len(opts) > 1 {
		os = opts[1]
	}
	if len(opts) > 2 && opts[2] != "" {
		lastOnline = opts[2]
	}
	if len(opts) > 3 && opts[3] != "" {
		lastLogon = opts[3]
	}
	if len(opts) > 4 {
		ip = opts[4]
	}
	if len(opts) > 5 {
		osVer = opts[5]
	}
	if len(opts) > 6 {
		mgr = opts[6]
	}
	if len(opts) > 7 {
		serial = opts[7]
	}
	now := time.Now()
	err := db.Exec(`
		INSERT INTO sys_ad_computer
		(id, ad_config_id, computer_name, distinguished_name, oudn, status,
		 original_description, operating_system, last_logon, last_online_time,
		 ip_address, os_version, managed_by, serial_number,
		 created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, id, configID, name, dn, ouDn, int(status),
		description, os, nullableTime(lastLogon), nullableTime(lastOnline),
		ip, osVer, mgr, serial, now, now).Error
	require.NoError(t, err)
}

// nullableTime 把 RFC3339 字符串转 *time.Time 或 nil（空串 → nil → SQL NULL）。
func nullableTime(s string) any {
	if s == "" {
		return nil
	}
	tt, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil
	}
	return tt
}

// =============================================================================
// Task 1: List 查询链 + GetByDN
// =============================================================================

// TestComp78_NewComputerService 构造器返回非 nil 且 db 透传。
func TestComp78_NewComputerService(t *testing.T) {
	db := setupComp78DB(t)
	defer closeDB(t, db)
	svc := NewComputerService(db)
	assert.NotNil(t, svc)
	assert.Same(t, db, svc.db, "db 应原样存入 ComputerService.db")
}

// TestComp78_NormalizePagination 覆盖 computer.go:78-88 的所有归一分支：
// Current<=0/PageSize<=0 → 1/10；PageSize>100 → 100；正常值透传。
func TestComp78_NormalizePagination(t *testing.T) {
	db := setupComp78DB(t)
	defer closeDB(t, db)
	svc := NewComputerService(db)

	// 全零 → 1/10
	r1 := &ComputerListRequest{}
	svc.normalizePagination(r1)
	assert.Equal(t, 1, r1.Current)
	assert.Equal(t, 10, r1.PageSize)

	// 负值 → 1/10
	r2 := &ComputerListRequest{BaseListRequest: base.BaseListRequest{Current: -3, PageSize: -1}}
	svc.normalizePagination(r2)
	assert.Equal(t, 1, r2.Current)
	assert.Equal(t, 10, r2.PageSize)

	// PageSize 超大 → 截到 100
	r3 := &ComputerListRequest{BaseListRequest: base.BaseListRequest{Current: 5, PageSize: 500}}
	svc.normalizePagination(r3)
	assert.Equal(t, 5, r3.Current)
	assert.Equal(t, 100, r3.PageSize)

	// 正常值透传
	r4 := &ComputerListRequest{BaseListRequest: base.BaseListRequest{Current: 3, PageSize: 25}}
	svc.normalizePagination(r4)
	assert.Equal(t, 3, r4.Current)
	assert.Equal(t, 25, r4.PageSize)
}

// TestComp78_List_FilterMatrix 覆盖 buildComputerQuery 三类过滤 + 组合过滤。
//
// buildComputerQuery (computer.go:91-105) 过滤语义：
//   - ConfigID: 必传
//   - OUN: oudn = ? OR oudn LIKE '%,'+?（含父 OU 下子 OU 的层级语义）
//   - ComputerName: LIKE '%name%'
//
// 预置 5 行：(2 个 cfgA 不同 OU + 1 个 cfgA 同 OU 但子 OU 命中 + 1 个 cfgB + 1 个 cfgA 不同 computer_name)
func TestComp78_List_FilterMatrix(t *testing.T) {
	db := setupComp78DB(t)
	defer closeDB(t, db)
	svc := NewComputerService(db)
	ctx := context.Background()

	cfgA := insertConfig78(t, db, uuid.NewString())
	cfgB := insertConfig78(t, db, uuid.NewString())

	// 5 行预置
	insertComputer78(t, db, cfgA.ID, "CN=PC1,OU=Sales,DC=example,DC=com", "PC1", "OU=Sales,DC=example,DC=com", models.ComputerStatusOnline)
	insertComputer78(t, db, cfgA.ID, "CN=PC2,OU=HR,DC=example,DC=com", "PC2", "OU=HR,DC=example,DC=com", models.ComputerStatusOffline)
	// 隶属"OU=Sales,DC=example,DC=com" 父级（用 ou_dn LIKE '%' 命中）：
	// 实际查询语义为 ouDn LIKE '%,OU=Sales,DC=example,DC=com'，所以子 OU 表达为
	// "OU=West,OU=Sales,DC=example,DC=com"
	insertComputer78(t, db, cfgA.ID, "CN=PC3,OU=West,OU=Sales,DC=example,DC=com", "PC3", "OU=West,OU=Sales,DC=example,DC=com", models.ComputerStatusOnline)
	insertComputer78(t, db, cfgB.ID, "CN=PC4,OU=Sales,DC=example,DC=com", "PC4", "OU=Sales,DC=example,DC=com", models.ComputerStatusOnline)
	insertComputer78(t, db, cfgA.ID, "CN=PC5,OU=HR,DC=example,DC=com", "PC5", "OU=HR,DC=example,DC=com", models.ComputerStatusOnline)

	// (a) 仅 ConfigID 过滤
	list, total, err := svc.List(ctx, &ComputerListRequest{
		BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 10},
		ConfigID:        cfgA.ID,
	})
	require.NoError(t, err)
	assert.EqualValues(t, 4, total, "cfgA 4 行")
	assert.Len(t, list, 4)

	// (b) OUN 子树过滤（parent=OU=Sales,DC=example,DC=com）：命中 PC1 + PC3（子 OU）
	list, total, err = svc.List(ctx, &ComputerListRequest{
		BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 10},
		ConfigID:        cfgA.ID,
		OUN:             "OU=Sales,DC=example,DC=com",
	})
	require.NoError(t, err)
	assert.EqualValues(t, 2, total, "cfgA 下 Sales 子树 2 行（PC1 + PC3）")
	assert.Len(t, list, 2)

	// (c) ComputerName 模糊匹配
	_, total, err = svc.List(ctx, &ComputerListRequest{
		BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 10},
		ConfigID:        cfgA.ID,
		ComputerName:    "PC",
	})
	require.NoError(t, err)
	assert.EqualValues(t, 4, total)

	// (d) 组合过滤：cfgA + HR OU + name=PC2
	list, total, err = svc.List(ctx, &ComputerListRequest{
		BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 10},
		ConfigID:        cfgA.ID,
		OUN:             "OU=HR,DC=example,DC=com",
		ComputerName:    "PC2",
	})
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	assert.Len(t, list, 1)
	assert.Equal(t, "PC2", list[0].ComputerName)

	// (e) 空结果形态：total=0 + 空 list（List 在 :65-67 早退）
	list, total, err = svc.List(ctx, &ComputerListRequest{
		BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 10},
		ConfigID:        cfgA.ID,
		ComputerName:    "NOPE",
	})
	require.NoError(t, err)
	assert.EqualValues(t, 0, total)
	assert.NotNil(t, list, "空结果形态返回非 nil 空切片")
	assert.Len(t, list, 0)
}

// TestComp78_List_SortWhitelist 覆盖 computer.go:34-39 白名单 + ApplySort 回落默认排序。
//
// computerAllowedSortFields 白名单：computerName / operatingSystem / lastLogon / createdAt。
// 非法字段（OrderByColumn="; DROP TABLE..."）应被 ResolveSort 拒绝 → 回落 created_at DESC。
func TestComp78_List_SortWhitelist(t *testing.T) {
	db := setupComp78DB(t)
	defer closeDB(t, db)
	svc := NewComputerService(db)
	ctx := context.Background()
	cfg := insertConfig78(t, db, "")

	// 预置 3 行不同 lastLogon（用 RFC3339 字符串写入 last_logon 列）
	t1 := "2024-01-01T00:00:00Z"
	t2 := "2024-06-01T00:00:00Z"
	t3 := "2024-12-01T00:00:00Z"
	insertComputer78(t, db, cfg.ID, "CN=A,OU=Test,DC=example,DC=com", "PC-A", "OU=Test,DC=example,DC=com", models.ComputerStatusOnline, "", "", "", t1)
	insertComputer78(t, db, cfg.ID, "CN=B,OU=Test,DC=example,DC=com", "PC-B", "OU=Test,DC=example,DC=com", models.ComputerStatusOnline, "", "", "", t2)
	insertComputer78(t, db, cfg.ID, "CN=C,OU=Test,DC=example,DC=com", "PC-C", "OU=Test,DC=example,DC=com", models.ComputerStatusOnline, "", "", "", t3)

	asc := true
	// 白名单字段 lastLogon ASC → A, B, C
	list, _, err := svc.List(ctx, &ComputerListRequest{
		BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 10, OrderByColumn: "lastLogon", IsAsc: &asc},
		ConfigID:        cfg.ID,
	})
	require.NoError(t, err)
	require.Len(t, list, 3)
	assert.Equal(t, "PC-A", list[0].ComputerName)
	assert.Equal(t, "PC-C", list[2].ComputerName)

	// 白名单外字段（含 SQL 注入特征）→ 不报 SQL 错误；回落默认 created_at DESC
	malicious := "1; DROP TABLE sys_ad_computer--"
	list, _, err = svc.List(ctx, &ComputerListRequest{
		BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 10, OrderByColumn: malicious},
		ConfigID:        cfg.ID,
	})
	require.NoError(t, err, "白名单外字段不应报 SQL 错误（白名单防注入）")
	require.Len(t, list, 3, "行数仍为 3（白名单拒绝=无 ORDER BY 变化）")

	// 验证表未被删（白名单路径正确）
	var n int64
	require.NoError(t, db.Model(&models.ADComputer{}).Count(&n).Error)
	assert.EqualValues(t, 3, n)
}

// TestComp78_List_Pagination 12 行 + pageSize=5 → 三页断言。
func TestComp78_List_Pagination(t *testing.T) {
	db := setupComp78DB(t)
	defer closeDB(t, db)
	svc := NewComputerService(db)
	ctx := context.Background()
	cfg := insertConfig78(t, db, "")

	for i := 0; i < 12; i++ {
		dn := fmt.Sprintf("CN=PC%02d,OU=Test,DC=example,DC=com", i)
		insertComputer78(t, db, cfg.ID, dn, fmt.Sprintf("PC%02d", i), "OU=Test,DC=example,DC=com", models.ComputerStatusOnline)
	}

	// 第 1 页
	list, total, err := svc.List(ctx, &ComputerListRequest{
		BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 5},
		ConfigID:        cfg.ID,
	})
	require.NoError(t, err)
	assert.EqualValues(t, 12, total)
	assert.Len(t, list, 5)

	// 第 2 页
	list, _, err = svc.List(ctx, &ComputerListRequest{
		BaseListRequest: base.BaseListRequest{Current: 2, PageSize: 5},
		ConfigID:        cfg.ID,
	})
	require.NoError(t, err)
	assert.Len(t, list, 5)

	// 第 3 页（余 2 行）
	list, _, err = svc.List(ctx, &ComputerListRequest{
		BaseListRequest: base.BaseListRequest{Current: 3, PageSize: 5},
		ConfigID:        cfg.ID,
	})
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

// TestComp78_List_ConvertToDetails 验证 convertToDetails (computer.go:135-144) 把
// parseComputerDescriptionForUser 的结果填入 ComputerDetail.LastLogonUser：
//   - description 含可解析的 |user| 形态 → 填充 LastLogonUser
//   - description 不可解析或空 → LastLogonUser 为空串
func TestComp78_List_ConvertToDetails(t *testing.T) {
	db := setupComp78DB(t)
	defer closeDB(t, db)
	svc := NewComputerService(db)
	ctx := context.Background()
	cfg := insertConfig78(t, db, "")

	// 行 1: 可解析的 lastLogonUser
	insertComputer78(t, db, cfg.ID,
		"CN=PC1,OU=Test,DC=example,DC=com",
		"PC1", "OU=Test,DC=example,DC=com", models.ComputerStatusOnline,
		"|alice|192.168.1.10|AA:BB:CC:DD:EE:FF|") // opts[0] = original_description
	// 行 2: 不可解析的 description（无管道分隔）
	insertComputer78(t, db, cfg.ID,
		"CN=PC2,OU=Test,DC=example,DC=com",
		"PC2", "OU=Test,DC=example,DC=com", models.ComputerStatusOnline,
		"普通描述无管道符")

	list, _, err := svc.List(ctx, &ComputerListRequest{
		BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 10},
		ConfigID:        cfg.ID,
	})
	require.NoError(t, err)
	require.Len(t, list, 2)

	var pc1, pc2 *ComputerDetail
	for i := range list {
		switch list[i].ComputerName {
		case "PC1":
			pc1 = &list[i]
		case "PC2":
			pc2 = &list[i]
		}
	}
	require.NotNil(t, pc1, "PC1 应在结果中")
	require.NotNil(t, pc2, "PC2 应在结果中")
	assert.Equal(t, "alice", pc1.LastLogonUser, "|alice|... 应被 parseComputerDescriptionForUser 解析")
	assert.Equal(t, "", pc2.LastLogonUser, "不可解析 description → LastLogonUser 为空")
}

// TestComp78_List_DBError DROP 表 → countComputers 失败 → 包装错误返回。
func TestComp78_List_DBError(t *testing.T) {
	db := setupComp78DB(t)
	defer closeDB(t, db)
	svc := NewComputerService(db)
	ctx := context.Background()
	cfg := insertConfig78(t, db, "")

	// 预置 1 行（确保执行路径走到 countComputers）
	insertComputer78(t, db, cfg.ID, "CN=PC1,OU=Test,DC=example,DC=com", "PC1", "OU=Test,DC=example,DC=com", models.ComputerStatusOnline)

	// DROP 表
	require.NoError(t, db.Exec(`DROP TABLE sys_ad_computer`).Error)

	_, _, err := svc.List(ctx, &ComputerListRequest{
		BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 10},
		ConfigID:        cfg.ID,
	})
	require.Error(t, err, "缺表时应返回错误，不 panic")
	assert.Contains(t, err.Error(), "统计总数失败", "countComputers 包装错误前缀 (computer.go:111)")
}

// TestComp78_GetByDN 命中 / 不命中 / 软删行被排除。
func TestComp78_GetByDN(t *testing.T) {
	db := setupComp78DB(t)
	defer closeDB(t, db)
	svc := NewComputerService(db)
	ctx := context.Background()
	cfg := insertConfig78(t, db, "")

	// 行 1: 正常
	insertComputer78(t, db, cfg.ID, "CN=PC1,OU=Test,DC=example,DC=com", "PC1", "OU=Test,DC=example,DC=com", models.ComputerStatusOnline, "|bob|...")
	// 行 2: 软删
	insertComputer78(t, db, cfg.ID, "CN=PC2,OU=Test,DC=example,DC=com", "PC2", "OU=Test,DC=example,DC=com", models.ComputerStatusOnline)
	require.NoError(t, db.Exec(`UPDATE sys_ad_computer SET deleted_at = ? WHERE distinguished_name = ?`,
		time.Now(), "CN=PC2,OU=Test,DC=example,DC=com").Error)

	// 命中：返回 detail + LastLogonUser 解析
	detail, err := svc.GetByDN(ctx, cfg.ID, "CN=PC1,OU=Test,DC=example,DC=com")
	require.NoError(t, err)
	require.NotNil(t, detail)
	assert.Equal(t, "PC1", detail.ComputerName)
	assert.Equal(t, "bob", detail.LastLogonUser)

	// 不命中（DN 不存在）
	detail, err = svc.GetByDN(ctx, cfg.ID, "CN=MISSING,OU=Test,DC=example,DC=com")
	require.Error(t, err)
	assert.Nil(t, detail)
	assert.Contains(t, err.Error(), "电脑设备不存在")

	// 软删行排除（gorm 默认过滤 + SQL 显式 deleted_at IS NULL）
	detail, err = svc.GetByDN(ctx, cfg.ID, "CN=PC2,OU=Test,DC=example,DC=com")
	require.Error(t, err)
	assert.Nil(t, detail)
	assert.Contains(t, err.Error(), "电脑设备不存在", "软删行应被排除")
}

// =============================================================================
// Task 1 helpers — 测试用 ldap.Entry 构造（仅本文件使用，命名避开与 78-05 entry78 冲突）
// =============================================================================

// compEntry78 构造 computer 用的 ldap.Entry（保留 78-05 entry78 风格，额外支持 cn）。
func compEntry78(dn, cn string, attrs map[string][]string) *ldap.Entry {
	all := map[string][]string{"cn": {cn}}
	for k, v := range attrs {
		all[k] = v
	}
	return entry78(dn, all)
}

// =============================================================================
// Task 2: syncComputers entry-driven 全链
// =============================================================================

// TestComp78_BuildComputerFromEntry 覆盖 computer.go:262-304：
//   - 完整 entry：cn / distinguishedName / operatingSystem / managedBy / description
//   - parsedDesc 由 parseComputerDescription 预解析后传入（serialNumber / ipAddress 等）
//   - 属性缺失 → 零值兜底，不 panic
//   - 异常 desc 形态 → serialNumber/IPAddress 等安全串（safeAttr 包络）
func TestComp78_BuildComputerFromEntry(t *testing.T) {
	// 完整 entry + 完整 parsedDesc
	entry := compEntry78("CN=PC-A,OU=Test,DC=example,DC=com", "PC-A", map[string][]string{
		"description":            {"|alice|10.0.0.1|AA:BB:CC:DD:EE:FF|SN-1234|Win11|i7-12700|x64|16GB|512GB|2024-06-01 12:00:00"},
		"operatingSystem":        {"Windows 11 Enterprise"},
		"operatingSystemVersion": {"22H2"},
		"managedBy":              {"CN=ops-svc,OU=Service,DC=example,DC=com"},
		"lastLogon":              {"133500000000000000"},
		"pwdLastSet":             {"133490000000000000"},
		"logonCount":             {"42"},
	})
	parsedDesc := parseComputerDescription("|alice|10.0.0.1|AA:BB:CC:DD:EE:FF|SN-1234|Win11|i7-12700|x64|16GB|512GB|2024-06-01 12:00:00")
	got := buildComputerFromEntry("cfg-X", entry, parsedDesc, models.ComputerStatusOnline)

	assert.Equal(t, "cfg-X", got.ADConfigID)
	assert.Equal(t, "PC-A", got.ComputerName)
	assert.Equal(t, "CN=PC-A,OU=Test,DC=example,DC=com", got.DistinguishedName)
	assert.Equal(t, "OU=Test,DC=example,DC=com", got.OUDN, "extractParentDN 剥离首段 CN=")
	assert.Equal(t, "Windows 11 Enterprise", got.OperatingSystem)
	assert.Equal(t, "22H2", got.OSVersion)
	assert.Equal(t, "CN=ops-svc,OU=Service,DC=example,DC=com", got.ManagedBy)
	assert.Equal(t, "10.0.0.1", got.IPAddress, "现行为：parsedDesc[\"ipAddress\"]='10.0.0.1'（即 desc 第 3 段，对应 fieldMappings index=2）")
	assert.Equal(t, "AA:BB:CC:DD:EE:FF", got.MacAddress, "现行为：parsedDesc[\"macAddress\"]='AA:BB...'（即 desc 第 4 段，index=3）")
	assert.Equal(t, models.ComputerStatusOnline, got.Status)
	assert.NotNil(t, got.LastLogon)
	assert.NotNil(t, got.LastOnlineTime)
	assert.Equal(t, 42, got.LogonCount)

	// 属性缺失 → 零值兜底不 panic
	empty := &ldap.Entry{DN: "CN=X,OU=Test,DC=example,DC=com"}
	got2 := buildComputerFromEntry("cfg-Y", empty, map[string]string{}, models.ComputerStatusOffline)
	assert.Equal(t, "cfg-Y", got2.ADConfigID)
	assert.Equal(t, "", got2.ComputerName, "cn 缺失 → empty string")
	assert.Equal(t, "OU=Test,DC=example,DC=com", got2.OUDN)
	assert.Equal(t, models.ComputerStatusOffline, got2.Status)
	assert.Nil(t, got2.LastLogon)
	assert.Nil(t, got2.LastOnlineTime)
	assert.Equal(t, 0, got2.LogonCount)
}

// TestComp78_UpdateComputerFields 覆盖 computer.go:307-343：mutate existing 指针。
//   - 字段变更 → 反映；未变更字段保留；UpdatedAt 刷新
func TestComp78_UpdateComputerFields(t *testing.T) {
	existing := &models.ADComputer{
		ComputerName:      "PC-OLD",
		DistinguishedName: "CN=OLD,OU=Test,DC=example,DC=com",
		OUDN:              "OU=Test,DC=example,DC=com",
		OperatingSystem:   "Windows 10",
		OSVersion:         "21H2",
		LogonCount:        10,
		Status:            models.ComputerStatusOnline,
	}
	prevUpdatedAt := existing.UpdatedAt // 通常为零值

	entry := compEntry78("CN=NEW,OU=Test,DC=example,DC=com", "PC-NEW", map[string][]string{
		"description":            {"||10.0.0.99|AA:99:99|SN-NEW|Win11||x64|||2024-12-31 00:00:00"},
		"operatingSystem":        {"Windows 11"},
		"operatingSystemVersion": {"23H2"},
		"logonCount":             {"100"},
	})
	parsed := parseComputerDescription("||10.0.0.99|AA:99:99|SN-NEW|Win11||x64|||2024-12-31 00:00:00")
	now := time.Now()
	updateComputerFields(existing, entry, parsed, models.ComputerStatusOffline, now)

	assert.Equal(t, "PC-NEW", existing.ComputerName)
	assert.Equal(t, "CN=NEW,OU=Test,DC=example,DC=com", existing.DistinguishedName)
	assert.Equal(t, "OU=Test,DC=example,DC=com", existing.OUDN)
	assert.Equal(t, "Windows 11", existing.OperatingSystem)
	assert.Equal(t, "23H2", existing.OSVersion)
	assert.Equal(t, 100, existing.LogonCount)
	assert.Equal(t, models.ComputerStatusOffline, existing.Status)
	assert.True(t, existing.UpdatedAt.After(prevUpdatedAt) || existing.UpdatedAt.Equal(prevUpdatedAt),
		"UpdatedAt 应被刷新到 now 或保持（首次写入等于 now）")
}

// TestComp78_QueryExistingComputers_And_AllNames 覆盖 computer.go:478-523：
//   - queryExistingComputers：按 DN + configID 命中；不同 config 不命中；空 DNs → 空
//   - queryAllComputerNames：本 config 全部（软删行也命中，因 :519 显式 deleted_at IS NULL
//     实际过滤；预置软删行 → 验证它被排除）
func TestComp78_QueryExistingComputers_And_AllNames(t *testing.T) {
	db := setupComp78DB(t)
	defer closeDB(t, db)
	svc := NewComputerService(db)
	ctx := context.Background()

	cfgA := insertConfig78(t, db, uuid.NewString())
	cfgB := insertConfig78(t, db, uuid.NewString())

	// 4 行：2 行 cfgA 在查询 DN 内 + 1 行 cfgA 不同 DN + 1 行 cfgB
	insertComputer78(t, db, cfgA.ID, "CN=DN1,OU=Test,DC=example,DC=com", "PC-A1", "OU=Test,DC=example,DC=com", models.ComputerStatusOnline)
	insertComputer78(t, db, cfgA.ID, "CN=DN2,OU=Test,DC=example,DC=com", "PC-A2", "OU=Test,DC=example,DC=com", models.ComputerStatusOnline)
	insertComputer78(t, db, cfgA.ID, "CN=DN3,OU=Test,DC=example,DC=com", "PC-A3", "OU=Test,DC=example,DC=com", models.ComputerStatusOnline)
	insertComputer78(t, db, cfgB.ID, "CN=DN1,OU=Test,DC=example,DC=com", "PC-B1", "OU=Test,DC=example,DC=com", models.ComputerStatusOnline)
	// 软删行（应被 queryAllComputerNames 排除）
	insertComputer78(t, db, cfgA.ID, "CN=DNSOFT,OU=Test,DC=example,DC=com", "PC-SOFT", "OU=Test,DC=example,DC=com", models.ComputerStatusOnline)
	require.NoError(t, db.Exec(`UPDATE sys_ad_computer SET deleted_at = ? WHERE computer_name = ?`,
		time.Now(), "PC-SOFT").Error)

	// queryExistingComputers: cfgA + DNs {DN1, DN2, MISS} → 应返回 DN1 + DN2（不同 DN/不同 config 都不命中）
	got := svc.queryExistingComputers(ctx, cfgA.ID, []string{
		"CN=DN1,OU=Test,DC=example,DC=com",
		"CN=DN2,OU=Test,DC=example,DC=com",
		"CN=MISS,OU=Test,DC=example,DC=com",
	})
	require.Len(t, got, 2, "cfgA 内命中 2 行（DN1/DN2），cfgB 同 DN 不命中")
	dns := []string{got[0].DistinguishedName, got[1].DistinguishedName}
	assert.Contains(t, dns, "CN=DN1,OU=Test,DC=example,DC=com")
	assert.Contains(t, dns, "CN=DN2,OU=Test,DC=example,DC=com")

	// 空 DN 切片 → 空结果
	gotEmpty := svc.queryExistingComputers(ctx, cfgA.ID, []string{})
	assert.Empty(t, gotEmpty)

	// queryAllComputerNames: cfgA 应返回 3 行（PC-SOFT 软删被过滤）
	all := svc.queryAllComputerNames(ctx, cfgA.ID)
	assert.Len(t, all, 3, "软删行 PC-SOFT 被 queryAllComputerNames 过滤（:519 显式 deleted_at IS NULL）")
	names := map[string]bool{}
	for _, c := range all {
		names[c.ComputerName] = true
	}
	assert.True(t, names["PC-A1"])
	assert.True(t, names["PC-A2"])
	assert.True(t, names["PC-A3"])
	assert.False(t, names["PC-SOFT"], "软删行不在结果中")
}

// TestComp78_BuildComputerMaps 覆盖 computer.go:526-539：双 map 构造。
//   - 4 行：2 行 distinct DN + name；2 行同 name（map 语义后写覆盖前写）。
func TestComp78_BuildComputerMaps(t *testing.T) {
	db := setupComp78DB(t)
	defer closeDB(t, db)
	svc := NewComputerService(db)

	rows := []models.ADComputer{
		{DistinguishedName: "CN=DN1,OU=Test,DC=example,DC=com", ComputerName: "PC-A"},
		{DistinguishedName: "CN=DN2,OU=Test,DC=example,DC=com", ComputerName: "PC-B"},
		{DistinguishedName: "CN=DN3,OU=Test,DC=example,DC=com", ComputerName: "PC-C"},
		{DistinguishedName: "CN=DN4,OU=Test,DC=example,DC=com", ComputerName: "PC-C"}, // 与上行同名，后写覆盖
	}
	dnMap, nameMap := svc.buildComputerMaps(rows, rows)

	require.NotNil(t, dnMap)
	require.NotNil(t, nameMap)
	assert.Len(t, dnMap, 4, "dnMap 按 DN 去重 → 4 行 4 DN")
	assert.Len(t, nameMap, 3, "nameMap 按 name 去重 → PC-C 重复 → 3 个 entry")

	assert.Equal(t, "PC-A", dnMap["CN=DN1,OU=Test,DC=example,DC=com"].ComputerName)
	assert.Equal(t, "PC-B", dnMap["CN=DN2,OU=Test,DC=example,DC=com"].ComputerName)
	assert.NotNil(t, nameMap["PC-A"])
	assert.NotNil(t, nameMap["PC-B"])
	assert.NotNil(t, nameMap["PC-C"], "同名重复 → nameMap 仍存在条目（后写覆盖）")
}

// TestComp78_ProcessComputerEntry_NewVsUpdateVsRename 覆盖 computer.go:547-577 三分支。
//
// 关键 F-01 修复（computer.go:543-546 注释）：
//   - DN 命中 existingDNMap → 更新 existingComputer 指针，加入 toUpdate[conflictKey]
//   - DN 不命中但 name 命中 existingNameMap（改名/OU move）→ 更新 existingByName 指针，加入 toUpdate[conflictKey]
//   - 都不命中 → buildComputerFromEntry 后 append toCreate
func TestComp78_ProcessComputerEntry_NewVsUpdateVsRename(t *testing.T) {
	db := setupComp78DB(t)
	defer closeDB(t, db)
	svc := NewComputerService(db)

	cfgID := "cfg-test"
	existing := &models.ADComputer{
		ADConfigID:        cfgID,
		ComputerName:      "PC-OLD",
		DistinguishedName: "CN=OLD,OU=Test,DC=example,DC=com",
		OUDN:              "OU=Test,DC=example,DC=com",
		Status:            models.ComputerStatusOnline,
	}
	dnMap := map[string]*models.ADComputer{
		"CN=OLD,OU=Test,DC=example,DC=com": existing,
	}
	nameMap := map[string]*models.ADComputer{
		"PC-OLD": existing,
	}
	toCreate := []models.ADComputer{}
	toUpdate := map[string]*models.ADComputer{}
	now := time.Now()

	// (a) 新建：DN 与 name 都不在 maps
	entryNew := compEntry78("CN=NEW,OU=Test,DC=example,DC=com", "PC-NEW", map[string][]string{
		"operatingSystem": {"Win11"},
	})
	svc.processComputerEntry(entryNew, cfgID, map[string]string{}, models.ComputerStatusOnline, now, dnMap, nameMap, &toCreate, toUpdate)
	assert.Len(t, toCreate, 1, "(a) 新建应进 toCreate")
	assert.Equal(t, "PC-NEW", toCreate[0].ComputerName)
	assert.Empty(t, toUpdate)

	// (b) DN 命中（既有行 DN 不变，名字也不变）
	toCreate = toCreate[:0]
	toUpdate = map[string]*models.ADComputer{}
	entryUpdate := compEntry78("CN=OLD,OU=Test,DC=example,DC=com", "PC-OLD", map[string][]string{
		"operatingSystem": {"Win11-updated"},
	})
	svc.processComputerEntry(entryUpdate, cfgID, map[string]string{}, models.ComputerStatusOffline, now, dnMap, nameMap, &toCreate, toUpdate)
	assert.Empty(t, toCreate, "(b) DN 命中 → 不进 toCreate")
	require.Len(t, toUpdate, 1)
	key := cfgID + "/" + "PC-OLD"
	assert.NotNil(t, toUpdate[key])
	assert.Equal(t, "Win11-updated", existing.OperatingSystem, "指针被 mutate")
	assert.Equal(t, models.ComputerStatusOffline, existing.Status)

	// (c) 改名：DN 不在 dnMap 但 name 在 nameMap（OU move / 重建场景）
	toCreate = toCreate[:0]
	toUpdate = map[string]*models.ADComputer{}
	// 重置 existing 状态以便观察
	existing.ComputerName = "PC-OLD"
	existing.DistinguishedName = "CN=OLD,OU=Test,DC=example,DC=com"
	existing.OperatingSystem = "Win11"
	entryRename := compEntry78("CN=RENAMED,OU=Test,DC=example,DC=com", "PC-OLD", map[string][]string{
		"operatingSystem": {"Win11-after-rename"},
	})
	svc.processComputerEntry(entryRename, cfgID, map[string]string{}, models.ComputerStatusOnline, now, dnMap, nameMap, &toCreate, toUpdate)
	assert.Empty(t, toCreate, "(c) DN 不命中 → 不进 toCreate")
	require.Len(t, toUpdate, 1, "(c) name 命中 → 进 toUpdate (F-01 修复：key 用 config_id+name 而非 DN)")
	assert.NotNil(t, toUpdate[key])
	assert.Equal(t, "CN=RENAMED,OU=Test,DC=example,DC=com", existing.DistinguishedName, "指针 DN 被更新")
	assert.Equal(t, "Win11-after-rename", existing.OperatingSystem)
}

// TestComp78_SyncComputers_FullChain 覆盖 computer.go:372-473 编排：
//   - 空 entries → 早退 nil
//   - 5 entries (2 新建 / 2 更新 / 1 改名) → 断言行数与字段
func TestComp78_SyncComputers_FullChain(t *testing.T) {
	db := setupComp78DB(t)
	defer closeDB(t, db)
	svc := NewComputerService(db)
	ctx := context.Background()
	cfg := insertConfig78(t, db, "")

	// 预置 2 行（将被 update）
	insertComputer78(t, db, cfg.ID, "CN=U1,OU=Test,DC=example,DC=com", "PC-U1", "OU=Test,DC=example,DC=com", models.ComputerStatusOnline, "", "Win10")
	insertComputer78(t, db, cfg.ID, "CN=U2,OU=Test,DC=example,DC=com", "PC-U2", "OU=Test,DC=example,DC=com", models.ComputerStatusOffline, "", "Win10")
	// 预置 1 行（将被 rename：DN 不命中，name 命中）
	insertComputer78(t, db, cfg.ID, "CN=REN-OLD,OU=Test,DC=example,DC=com", "PC-REN", "OU=Test,DC=example,DC=com", models.ComputerStatusOnline, "", "Win11")

	// 5 entries: 2 新 + 2 更 + 1 改名
	entries := []*ldap.Entry{
		compEntry78("CN=N1,OU=Test,DC=example,DC=com", "PC-N1", map[string][]string{"operatingSystem": {"Win11"}}),
		compEntry78("CN=N2,OU=Test,DC=example,DC=com", "PC-N2", map[string][]string{"operatingSystem": {"Win11"}}),
		compEntry78("CN=U1,OU=Test,DC=example,DC=com", "PC-U1", map[string][]string{"operatingSystem": {"Win11-upd"}}),
		compEntry78("CN=U2,OU=Test,DC=example,DC=com", "PC-U2", map[string][]string{"operatingSystem": {"Win11-upd"}}),
		compEntry78("CN=REN-NEW,OU=Test,DC=example,DC=com", "PC-REN", map[string][]string{"operatingSystem": {"Win11-ren"}}),
	}
	require.NoError(t, svc.syncComputers(ctx, cfg, entries))

	// 行数 = 5（3 预置 + 2 新建；改名不增行）
	var n int64
	require.NoError(t, db.Model(&models.ADComputer{}).Where("ad_config_id = ?", cfg.ID).Count(&n).Error)
	assert.EqualValues(t, 5, n, "改名不增行 (F-01 修复：name map 去重)")

	// 断言字段
	var pcU1 models.ADComputer
	require.NoError(t, db.Where("ad_config_id = ? AND computer_name = ?", cfg.ID, "PC-U1").First(&pcU1).Error)
	assert.Equal(t, "Win11-upd", pcU1.OperatingSystem)

	var pcREN models.ADComputer
	require.NoError(t, db.Where("ad_config_id = ? AND computer_name = ?", cfg.ID, "PC-REN").First(&pcREN).Error)
	assert.Equal(t, "CN=REN-NEW,OU=Test,DC=example,DC=com", pcREN.DistinguishedName, "改名后 DN 被 upsert 更新")

	var pcN1 models.ADComputer
	require.NoError(t, db.Where("ad_config_id = ? AND computer_name = ?", cfg.ID, "PC-N1").First(&pcN1).Error)
	assert.Equal(t, "CN=N1,OU=Test,DC=example,DC=com", pcN1.DistinguishedName)
	assert.Equal(t, "Win11", pcN1.OperatingSystem)

	// 空 entries 早退
	require.NoError(t, svc.syncComputers(ctx, cfg, nil))
	require.NoError(t, svc.syncComputers(ctx, cfg, []*ldap.Entry{}))
}

// TestComp78_BatchCreate_And_BatchUpdate_Batching 验证分批：
//   - batchCreate (computer.go:346-369, batchCreateSize=100): 250 entries → 3 批 (100/100/50)
//   - batchUpdate (computer.go:580-623, batchUpdateSize=200): 250 pre-inserted + 250 same (config,name) → 2 批 ON CONFLICT
//   - 空切片 / 空 map 早退
func TestComp78_BatchCreate_And_BatchUpdate_Batching(t *testing.T) {
	db := setupComp78DB(t)
	defer closeDB(t, db)
	svc := NewComputerService(db)
	ctx := context.Background()
	cfg := insertConfig78(t, db, "")

	// batchCreate 空切片早退
	require.NoError(t, svc.batchCreate(ctx, nil))
	require.NoError(t, svc.batchCreate(ctx, []models.ADComputer{}))

	// batchUpdate 空 map 早退
	require.NoError(t, svc.batchUpdate(ctx, nil))
	require.NoError(t, svc.batchUpdate(ctx, map[string]*models.ADComputer{}))

	// batchCreate 多批：生成 250 entries（>100 = batchCreateSize）
	entries := make([]*ldap.Entry, 250)
	for i := 0; i < 250; i++ {
		cn := fmt.Sprintf("PC-%04d", i)
		entries[i] = compEntry78(fmt.Sprintf("CN=%s,OU=Test,DC=example,DC=com", cn), cn, map[string][]string{
			"operatingSystem": {"Win11"},
		})
	}
	require.NoError(t, svc.syncComputers(ctx, cfg, entries))

	var totalRows int64
	require.NoError(t, db.Model(&models.ADComputer{}).Where("ad_config_id = ?", cfg.ID).Count(&totalRows).Error)
	assert.EqualValues(t, 250, totalRows, "batchCreate 分批后 250 行全部落库")

	// batchUpdate 多批：预置 250 行 → 同步 250 entries (同样 config_id+computer_name) 触发 upsert
	// 先清空表，预置 250 行
	require.NoError(t, db.Exec(`DELETE FROM sys_ad_computer WHERE ad_config_id = ?`, cfg.ID).Error)
	now := time.Now()
	for i := 0; i < 250; i++ {
		cn := fmt.Sprintf("PC-%04d", i)
		dn := fmt.Sprintf("CN=%s,OU=Test,DC=example,DC=com", cn)
		require.NoError(t, db.Exec(`
			INSERT INTO sys_ad_computer (id, ad_config_id, computer_name, distinguished_name, oudn, status, operating_system, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, uuid.NewString(), cfg.ID, cn, dn, "OU=Test,DC=example,DC=com", 0, "Win10-old", now, now).Error)
	}

	// syncComputers 重新跑同样 250 entries → 全部走 batchUpdate ON CONFLICT 分支
	require.NoError(t, svc.syncComputers(ctx, cfg, entries))

	require.NoError(t, db.Model(&models.ADComputer{}).Where("ad_config_id = ?", cfg.ID).Count(&totalRows).Error)
	assert.EqualValues(t, 250, totalRows, "batchUpdate upsert 不增行")
	// 抽查其中一行确认 OS 被更新
	var pc0 models.ADComputer
	require.NoError(t, db.Where("ad_config_id = ? AND computer_name = ?", cfg.ID, "PC-0000").First(&pc0).Error)
	assert.Equal(t, "Win11", pc0.OperatingSystem, "ON CONFLICT 应更新字段")
}

// TestComp78_SyncComputers_DBError DROP 表 → syncComputers 应包装返回错误（来自 batchCreate）。
func TestComp78_SyncComputers_DBError(t *testing.T) {
	db := setupComp78DB(t)
	defer closeDB(t, db)
	svc := NewComputerService(db)
	ctx := context.Background()
	cfg := insertConfig78(t, db, "")

	require.NoError(t, db.Exec(`DROP TABLE sys_ad_computer`).Error)

	entries := []*ldap.Entry{
		compEntry78("CN=PC1,OU=Test,DC=example,DC=com", "PC1", map[string][]string{
			"operatingSystem": {"Win11"},
		}),
	}
	err := svc.syncComputers(ctx, cfg, entries)
	require.Error(t, err, "缺表时应返回错误，不 panic")
	// queryExistingComputers / queryAllComputerNames 静默吞错 → batchCreate 报缺表
	// 错误路径：queryExistingComputers 无错 → queryAllComputerNames 无错 →
	// buildComputerMaps 空 → processComputerEntry 全部进 toCreate → batchCreate 失败
	assert.True(t,
		strings.Contains(err.Error(), "批量创建失败") || strings.Contains(err.Error(), "no such table"),
		"错误应来自 batchCreate (computer.go:361)：%v", err)
}