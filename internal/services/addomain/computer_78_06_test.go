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
func nullableTime(s string) interface{} {
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
	list, total, err = svc.List(ctx, &ComputerListRequest{
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

// _ 占位导入，确保 strings 被引用（其他 Task 测试将使用）。
var _ = strings.Contains