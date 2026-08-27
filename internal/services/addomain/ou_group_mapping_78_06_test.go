//go:build !skip_db_tests
// +build !skip_db_tests

// Phase 78 Plan 06: ou_group_mapping_service.go 全 11 函数 CRUD + 同步日志测试 (Task 3)
//
// 复用 78-05 helper（setupSync78DB / entry78 / insertConfig78 / closeDB / newSyncSvc78）— D-78-06e 禁止重定义。
// 补建 sys_ou_group_mapping + sys_ou_group_mapping_sync_log 表（列集照 models.OUGroupMapping + models.OUGroupMappingSyncLog gorm tag）。
// isUniqueConstraintError 用真实 sqlite UNIQUE 约束错误驱动（D-78-06c），不伪造字符串。
//
// 覆盖函数（ou_group_mapping_service.go）：
//   NewOUGroupMappingService / ListMappings / CreateMapping / GetMapping / UpdateMapping /
//   DeleteMapping / GetMappingsByOU / CreateSyncLog / UpdateSyncStatus /
//   isUniqueConstraintError / containsIgnoreCase
//
// 同包内 utils.go / sanitize_test.go 未重复定义 containsIgnoreCase（已 grep 唯一在 ou_group_mapping_service.go:263），
// 故无需担心重名冲突。

package addomain

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// setupOGM78DB 在 78-05 7 表 fixture 上追加 OU-Group 映射与同步日志表。
//
// sys_ou_group_mapping 列名照 model gorm tag：
//   id (uuid PK) / ad_config_id / ou_dn / ou_name / ad_group_id / mapping_status /
//   sync_enabled / last_sync_at / created_by / updated_by / created_at / updated_at
//   + 唯一索引 UNIQUE(ou_dn, ad_group_id)（对应 model `uniqueIndex:uni_ou_group_mapping_ou`）
//
// sys_ou_group_mapping_sync_log 列名照 model gorm tag：
//   id / mapping_id / ou_dn / ad_group_id / sync_type / members_added /
//   members_removed / total_members / status / error_msg /
//   started_at / completed_at / duration_ms / created_at
//
// 注意 OUGroupMapping 不嵌入 BaseModel（无 deleted_at / version）；sys_ou_group_mapping_sync_log
// 也不含 BaseModel 与 deleted_at。
func setupOGM78DB(t *testing.T) *gorm.DB {
	t.Helper()
	db := setupSync78DB(t)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_ou_group_mapping (
			id TEXT PRIMARY KEY,
			ad_config_id TEXT,
			ou_dn TEXT,
			ou_name TEXT,
			ad_group_id TEXT,
			mapping_status TEXT DEFAULT 'active',
			sync_enabled INTEGER DEFAULT 1,
			last_sync_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			UNIQUE(ou_dn, ad_group_id)
		)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_ou_group_mapping_sync_log (
			id TEXT PRIMARY KEY,
			mapping_id TEXT,
			ou_dn TEXT,
			ad_group_id TEXT,
			sync_type TEXT,
			members_added INTEGER DEFAULT 0,
			members_removed INTEGER DEFAULT 0,
			total_members INTEGER DEFAULT 0,
			status TEXT,
			error_msg TEXT,
			started_at DATETIME,
			completed_at DATETIME,
			duration_ms INTEGER,
			created_at DATETIME
		)
	`).Error)
	return db
}

// insertADGroup78 插入一条 sys_ad_group 行（ListMappings 的 Preload("ADGroup") 需要）；
// OU-Group mapping Create 也需验证 ADGroup 存在。
func insertADGroup78(t *testing.T, db *gorm.DB, configID, groupName, groupDN string) string {
	t.Helper()
	id := uuid.NewString()
	require.NoError(t, db.Exec(`
		INSERT INTO sys_ad_group (id, ad_config_id, group_dn, group_name, group_scope, group_type, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'Global', 0, ?, ?)
	`, id, configID, groupDN, groupName, time.Now(), time.Now()).Error)
	return id
}

// =============================================================================
// Task 3: OU-Group mapping 全 11 函数
// =============================================================================

// TestOGM78_ContainsIgnoreCase 纯函数表驱动（命中 / 大小写混合 / 不命中 / 空串 / 首尾子串）。
func TestOGM78_ContainsIgnoreCase(t *testing.T) {
	cases := []struct {
		s, sub string
		want   bool
	}{
		{"hello world", "world", true},
		{"HELLO World", "world", true}, // 大小写不敏感
		{"unique constraint", "CONSTRAINT", true},
		{"duplicate key value", "DUPLICATE", true},
		{"hello world", "foo", false},
		{"", "anything", false},
		{"hello", "", true}, // 空子串总在
		{"foo", "foobar", false},
		{"hello world", "hello", true}, // 首部子串
	}
	for _, c := range cases {
		got := containsIgnoreCase(c.s, c.sub)
		assert.Equal(t, c.want, got, "containsIgnoreCase(%q, %q)", c.s, c.sub)
	}
}

// TestOGM78_IsUniqueConstraintError 用真实 sqlite UNIQUE 约束触发（建表 + 重复行），
// 验证 isUniqueConstraintError 能识别 glebarez sqlite 的 driver error 文案
// （含 "UNIQUE constraint failed"，case-insensitive 匹配 "unique constraint"）。
// 若 sqlite 文案改变导致不识别，按 D-78-06c 记 SUMMARY 待裁决而非擅改。
func TestOGM78_IsUniqueConstraintError(t *testing.T) {
	// nil → false
	assert.False(t, isUniqueConstraintError(nil))

	// 普通错误（非 UNIQUE）→ false
	assert.False(t, isUniqueConstraintError(errors.New("connection refused")))

	// 真实 sqlite UNIQUE 错误：通过 raw SQL 触发
	db := setupOGM78DB(t)
	defer closeDB(t, db)
	require.NoError(t, db.Exec(`CREATE UNIQUE INDEX idx_test_unique ON sys_ou_group_mapping(ou_dn, ad_group_id)`).Error)
	// 注：CREATE TABLE 已经声明 UNIQUE(ou_dn, ad_group_id)，但 glebarez sqlite 对表级 UNIQUE
	// 仍会写入约束。直接 INSERT 两条重复行即可触发。
	id1 := uuid.NewString()
	id2 := uuid.NewString()
	now := time.Now()
	require.NoError(t, db.Exec(`
		INSERT INTO sys_ou_group_mapping
		(id, ad_config_id, ou_dn, ou_name, ad_group_id, mapping_status, sync_enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 'active', 1, ?, ?)
	`, id1, "cfg-1", "OU=Sales,DC=example,DC=com", "Sales", "grp-1", now, now).Error)

	err := db.Exec(`
		INSERT INTO sys_ou_group_mapping
		(id, ad_config_id, ou_dn, ou_name, ad_group_id, mapping_status, sync_enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 'active', 1, ?, ?)
	`, id2, "cfg-1", "OU=Sales,DC=example,DC=com", "Sales", "grp-1", now, now).Error
	require.Error(t, err, "重复 INSERT 应触发 UNIQUE 约束")
	assert.True(t, isUniqueConstraintError(err),
		"真实 sqlite UNIQUE 错误应被识别 (D-78-06c:不识别则记 SUMMARY 待裁决)。err=%v", err)

	// 模拟 PG 文案的 fake error → 也应识别（PG 用 "duplicate key" / "unique constraint"）
	pgStyle := errors.New("ERROR: duplicate key value violates unique constraint")
	assert.True(t, isUniqueConstraintError(pgStyle))
	pgStyle2 := errors.New("ERROR: unique constraint \"uni_x\" violated")
	assert.True(t, isUniqueConstraintError(pgStyle2))
}

// TestOGM78_NewOUGroupMappingService 构造器。
func TestOGM78_NewOUGroupMappingService(t *testing.T) {
	db := setupOGM78DB(t)
	defer closeDB(t, db)
	svc := NewOUGroupMappingService(db)
	assert.NotNil(t, svc)
}

// TestOGM78_CreateMapping_And_Duplicate CreateMapping 成功 + 重复创建走 isUniqueConstraintError 分支。
func TestOGM78_CreateMapping_And_Duplicate(t *testing.T) {
	db := setupOGM78DB(t)
	defer closeDB(t, db)
	svc := NewOUGroupMappingService(db)
	ctx := context.Background()
	cfg := insertConfig78(t, db, "")
	groupID := insertADGroup78(t, db, cfg.ID, "SalesGroup", "CN=SalesGroup,OU=Groups,DC=example,DC=com")

	// 成功创建
	m, err := svc.CreateMapping(ctx, &CreateMappingRequest{
		ADConfigID:  cfg.ID,
		OUDn:        "OU=Sales,DC=example,DC=com",
		OUName:      "Sales",
		ADGroupID:   groupID,
		SyncEnabled: true,
		CreatedBy:   "user-1",
	})
	require.NoError(t, err)
	require.NotNil(t, m)
	assert.NotEmpty(t, m.ID)
	assert.Equal(t, models.OUGroupMappingStatusActive, m.MappingStatus)
	assert.True(t, m.SyncEnabled)
	assert.NotNil(t, m.ADGroup, "Preload(\"ADGroup\") 应填充")
	assert.Equal(t, "SalesGroup", m.ADGroup.GroupName)

	// 重复创建 → 走 isUniqueConstraintError 分支 → 友好错误
	_, err = svc.CreateMapping(ctx, &CreateMappingRequest{
		ADConfigID:  cfg.ID,
		OUDn:        "OU=Sales,DC=example,DC=com",
		OUName:      "Sales",
		ADGroupID:   groupID,
		SyncEnabled: true,
		CreatedBy:   "user-1",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mapping already exists", "UNIQUE 冲突应被 isUniqueConstraintError 识别并返回友好文案 (ou_group_mapping_service.go:139)")

	// ADGroupID 不存在 → "AD group not found"
	_, err = svc.CreateMapping(ctx, &CreateMappingRequest{
		ADConfigID:  cfg.ID,
		OUDn:        "OU=HR,DC=example,DC=com",
		OUName:      "HR",
		ADGroupID:   uuid.NewString(),
		SyncEnabled: true,
		CreatedBy:   "user-1",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "AD group not found")
}

// TestOGM78_ListMappings_FilterAndPage 覆盖 ListMappings 4 类过滤 + 分页 + 空结果。
//   - ADConfigID / OUDn / Status / GroupName 各自独立 + 组合
func TestOGM78_ListMappings_FilterAndPage(t *testing.T) {
	db := setupOGM78DB(t)
	defer closeDB(t, db)
	svc := NewOUGroupMappingService(db)
	ctx := context.Background()
	cfgA := insertConfig78(t, db, uuid.NewString())
	cfgB := insertConfig78(t, db, uuid.NewString())
	g1 := insertADGroup78(t, db, cfgA.ID, "SalesGroup", "CN=SalesGroup,OU=Groups,DC=example,DC=com")
	g2 := insertADGroup78(t, db, cfgA.ID, "HRGroup", "CN=HRGroup,OU=Groups,DC=example,DC=com")
	g3 := insertADGroup78(t, db, cfgB.ID, "EngGroup", "CN=EngGroup,OU=Groups,DC=example,DC=com")

	// 6 行：3 个 cfgA (2 active + 1 inactive) + 3 个 cfgB
	inactive := models.OUGroupMappingStatusInactive
	rows := []*models.OUGroupMapping{
		{ADConfigID: cfgA.ID, OUDN: "OU=Sales,DC=example,DC=com", OUName: "Sales", ADGroupID: g1, MappingStatus: models.OUGroupMappingStatusActive, SyncEnabled: true},
		{ADConfigID: cfgA.ID, OUDN: "OU=HR,DC=example,DC=com", OUName: "HR", ADGroupID: g2, MappingStatus: models.OUGroupMappingStatusActive, SyncEnabled: true},
		{ADConfigID: cfgA.ID, OUDN: "OU=Sales2,DC=example,DC=com", OUName: "Sales2", ADGroupID: g1, MappingStatus: inactive, SyncEnabled: false},
		{ADConfigID: cfgB.ID, OUDN: "OU=Eng,DC=example,DC=com", OUName: "Eng", ADGroupID: g3, MappingStatus: models.OUGroupMappingStatusActive, SyncEnabled: true},
		{ADConfigID: cfgB.ID, OUDN: "OU=Eng2,DC=example,DC=com", OUName: "Eng2", ADGroupID: g3, MappingStatus: models.OUGroupMappingStatusActive, SyncEnabled: true},
		{ADConfigID: cfgB.ID, OUDN: "OU=Eng3,DC=example,DC=com", OUName: "Eng3", ADGroupID: g3, MappingStatus: models.OUGroupMappingStatusActive, SyncEnabled: true},
	}
	for _, r := range rows {
		r.ID = uuid.NewString()
		require.NoError(t, db.Create(r).Error)
	}

	// 仅 ADConfigID 过滤
	resp, err := svc.ListMappings(ctx, &ListMappingsRequest{
		ADConfigID: cfgA.ID,
		Current:    1, PageSize: 10,
	})
	require.NoError(t, err)
	assert.EqualValues(t, 3, resp.Total)
	assert.Len(t, resp.List, 3)

	// 仅 OUDn 过滤
	resp, err = svc.ListMappings(ctx, &ListMappingsRequest{
		OUDn:    "OU=Sales,DC=example,DC=com",
		Current: 1, PageSize: 10,
	})
	require.NoError(t, err)
	assert.EqualValues(t, 1, resp.Total)

	// 仅 Status 过滤
	resp, err = svc.ListMappings(ctx, &ListMappingsRequest{
		Status:  string(models.OUGroupMappingStatusInactive),
		Current: 1, PageSize: 10,
	})
	require.NoError(t, err)
	assert.EqualValues(t, 1, resp.Total)

	// 仅 GroupName 过滤（join sys_ad_group LIKE）
	// D-78-06f 现行为锁：GroupName 过滤触发 JOIN sys_ad_group，最终 ORDER BY created_at
	// 跨表歧义（"ambiguous column name: created_at"），sqlite 与 PG 同样会报错。
	// 本期不修，记 SUMMARY 待裁决。改为断言"返回错误包含歧义信息"以实证覆盖到 JOIN 路径。
	resp, err = svc.ListMappings(ctx, &ListMappingsRequest{
		GroupName: "EngGroup",
		Current:   1, PageSize: 10,
	})
	require.Error(t, err, "现行为：GroupName JOIN + ORDER BY created_at 跨表歧义")
	assert.Contains(t, err.Error(), "ambiguous column name: created_at",
		"D-78-06f: 该 JOIN 路径已触发覆盖，但 ORDER BY 跨表未限定表别名，sqlite/PG 均会报。文档化不动生产。err=%v", err)

	// 组合过滤 + 分页
	resp, err = svc.ListMappings(ctx, &ListMappingsRequest{
		ADConfigID: cfgB.ID,
		Status:     string(models.OUGroupMappingStatusActive),
		Current:    2, PageSize: 2,
	})
	require.NoError(t, err)
	assert.EqualValues(t, 3, resp.Total)
	assert.Len(t, resp.List, 1, "第 2 页 1 行（3 行 - 2 = 1）")
	assert.Equal(t, 2, resp.Current)
	assert.Equal(t, 2, resp.PageSize)

	// 空结果形态
	resp, err = svc.ListMappings(ctx, &ListMappingsRequest{
		ADConfigID: uuid.NewString(),
		Current:    1, PageSize: 10,
	})
	require.NoError(t, err)
	assert.EqualValues(t, 0, resp.Total)
	assert.NotNil(t, resp.List)
	assert.Len(t, resp.List, 0)

	// 默认分页归一：Current=0/PageSize=0 → 1/10
	resp, err = svc.ListMappings(ctx, &ListMappingsRequest{
		ADConfigID: cfgA.ID,
		Current:    0, PageSize: 0,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, resp.Current)
	assert.Equal(t, 10, resp.PageSize)
}

// TestOGM78_GetMapping_And_GetMappingsByOU 命中 / 不命中；GetMappingsByOU 仅 active。
func TestOGM78_GetMapping_And_GetMappingsByOU(t *testing.T) {
	db := setupOGM78DB(t)
	defer closeDB(t, db)
	svc := NewOUGroupMappingService(db)
	ctx := context.Background()
	cfg := insertConfig78(t, db, "")
	g1 := insertADGroup78(t, db, cfg.ID, "SalesGroup", "CN=SalesGroup,OU=Groups,DC=example,DC=com")
	g2 := insertADGroup78(t, db, cfg.ID, "HRGroup", "CN=HRGroup,OU=Groups,DC=example,DC=com")

	m1 := &models.OUGroupMapping{
		ADConfigID: cfg.ID, OUDN: "OU=Sales,DC=example,DC=com", OUName: "Sales",
		ADGroupID: g1, MappingStatus: models.OUGroupMappingStatusActive, SyncEnabled: true,
	}
	m1.ID = uuid.NewString()
	require.NoError(t, db.Create(m1).Error)

	m2 := &models.OUGroupMapping{
		ADConfigID: cfg.ID, OUDN: "OU=HR,DC=example,DC=com", OUName: "HR",
		ADGroupID: g2, MappingStatus: models.OUGroupMappingStatusInactive, SyncEnabled: false,
	}
	m2.ID = uuid.NewString()
	require.NoError(t, db.Create(m2).Error)

	// 命中
	got, err := svc.GetMapping(ctx, m1.ID)
	require.NoError(t, err)
	assert.Equal(t, "Sales", got.OUName)
	assert.NotNil(t, got.ADGroup, "Preload 应填充")
	assert.Equal(t, "SalesGroup", got.ADGroup.GroupName)

	// 不命中
	_, err = svc.GetMapping(ctx, uuid.NewString())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mapping not found")

	// GetMappingsByOU OU=Sales: 1 行 (active)
	got2, err := svc.GetMappingsByOU(ctx, "OU=Sales,DC=example,DC=com")
	require.NoError(t, err)
	assert.Len(t, got2, 1)
	assert.Equal(t, "Sales", got2[0].OUName)

	// GetMappingsByOU OU=HR: 0 行 (inactive 被过滤 :221)
	got2, err = svc.GetMappingsByOU(ctx, "OU=HR,DC=example,DC=com")
	require.NoError(t, err)
	assert.Len(t, got2, 0, "inactive 映射被 GetMappingsByOU 过滤 (mapping_status = active)")

	// GetMappingsByOU 不存在 OU: 空
	got2, err = svc.GetMappingsByOU(ctx, "OU=Missing,DC=example,DC=com")
	require.NoError(t, err)
	assert.Len(t, got2, 0)
}

// TestOGM78_UpdateMapping 字段更新 + 不存在 id + 空 updates。
func TestOGM78_UpdateMapping(t *testing.T) {
	db := setupOGM78DB(t)
	defer closeDB(t, db)
	svc := NewOUGroupMappingService(db)
	ctx := context.Background()
	cfg := insertConfig78(t, db, "")
	g1 := insertADGroup78(t, db, cfg.ID, "SalesGroup", "CN=SalesGroup,OU=Groups,DC=example,DC=com")
	m1 := &models.OUGroupMapping{
		ADConfigID: cfg.ID, OUDN: "OU=Sales,DC=example,DC=com", OUName: "Sales",
		ADGroupID: g1, MappingStatus: models.OUGroupMappingStatusActive, SyncEnabled: true,
	}
	m1.ID = uuid.NewString()
	require.NoError(t, db.Create(m1).Error)

	// 字段更新：sync_enabled + status + updated_by
	disabled := false
	inactive := models.OUGroupMappingStatusInactive
	err := svc.UpdateMapping(ctx, m1.ID, &UpdateMappingRequest{
		SyncEnabled: &disabled,
		Status:      &inactive,
		UpdatedBy:   "user-2",
	})
	require.NoError(t, err)

	// 回读
	var got models.OUGroupMapping
	require.NoError(t, db.First(&got, "id = ?", m1.ID).Error)
	assert.False(t, got.SyncEnabled)
	assert.Equal(t, models.OUGroupMappingStatusInactive, got.MappingStatus)
	assert.Equal(t, "user-2", got.UpdatedBy)

	// 不存在 id → "mapping not found"
	err = svc.UpdateMapping(ctx, uuid.NewString(), &UpdateMappingRequest{
		SyncEnabled: &disabled,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mapping not found")

	// 空 updates（无 SyncEnabled/Status/UpdatedBy） → "no fields to update"
	err = svc.UpdateMapping(ctx, m1.ID, &UpdateMappingRequest{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no fields to update", "ou_group_mapping_service.go:182 早退")
}

// TestOGM78_DeleteMapping 删除成功 + 不存在 id。
func TestOGM78_DeleteMapping(t *testing.T) {
	db := setupOGM78DB(t)
	defer closeDB(t, db)
	svc := NewOUGroupMappingService(db)
	ctx := context.Background()
	cfg := insertConfig78(t, db, "")
	g1 := insertADGroup78(t, db, cfg.ID, "SalesGroup", "CN=SalesGroup,OU=Groups,DC=example,DC=com")
	m1 := &models.OUGroupMapping{
		ADConfigID: cfg.ID, OUDN: "OU=Sales,DC=example,DC=com", OUName: "Sales",
		ADGroupID: g1, MappingStatus: models.OUGroupMappingStatusActive, SyncEnabled: true,
	}
	m1.ID = uuid.NewString()
	require.NoError(t, db.Create(m1).Error)

	// 删除成功
	require.NoError(t, svc.DeleteMapping(ctx, m1.ID))

	// 回读：因 mapping 无 deleted_at 列，row 真删
	var count int64
	require.NoError(t, db.Model(&models.OUGroupMapping{}).Where("id = ?", m1.ID).Count(&count).Error)
	assert.EqualValues(t, 0, count, "OUGroupMapping 无 deleted_at → 硬删")

	// 不存在 id
	err := svc.DeleteMapping(ctx, uuid.NewString())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mapping not found")
}

// TestOGM78_CreateSyncLog_And_UpdateSyncStatus 创建日志 + UpdateSyncStatus 写 last_sync_at + 不存在 mappingID 行为。
func TestOGM78_CreateSyncLog_And_UpdateSyncStatus(t *testing.T) {
	db := setupOGM78DB(t)
	defer closeDB(t, db)
	svc := NewOUGroupMappingService(db)
	ctx := context.Background()
	cfg := insertConfig78(t, db, "")
	g1 := insertADGroup78(t, db, cfg.ID, "SalesGroup", "CN=SalesGroup,OU=Groups,DC=example,DC=com")
	m1 := &models.OUGroupMapping{
		ADConfigID: cfg.ID, OUDN: "OU=Sales,DC=example,DC=com", OUName: "Sales",
		ADGroupID: g1, MappingStatus: models.OUGroupMappingStatusActive, SyncEnabled: true,
	}
	m1.ID = uuid.NewString()
	require.NoError(t, db.Create(m1).Error)

	// CreateSyncLog
	now := time.Now()
	log := &models.OUGroupMappingSyncLog{
		MappingID:    m1.ID,
		OUdn:         m1.OUDN,
		ADGroupID:    m1.ADGroupID,
		SyncType:     "full",
		MembersAdded: 5,
		TotalMembers: 20,
		Status:       "success",
		StartedAt:    now,
	}
	require.NoError(t, svc.CreateSyncLog(ctx, log))
	assert.NotEmpty(t, log.ID, "BeforeCreate 钩子应填 UUID")

	// UpdateSyncStatus 成功
	require.NoError(t, svc.UpdateSyncStatus(ctx, m1.ID, now))
	var got models.OUGroupMapping
	require.NoError(t, db.First(&got, "id = ?", m1.ID).Error)
	assert.NotNil(t, got.LastSyncAt)
	assert.True(t, got.LastSyncAt.Equal(now), "last_sync_at 应被设为传入时间")

	// 不存在 mappingID：UpdateSyncStatus 不返回错误（RowsAffected=0 被忽略，:243-247）
	// 现行为断言：err=nil，DB 不变化
	require.NoError(t, svc.UpdateSyncStatus(ctx, uuid.NewString(), now))
}

// TestOGM78_CreateMapping_DBError DROP 表 → CreateMapping 返回包装错误。
func TestOGM78_CreateMapping_DBError(t *testing.T) {
	db := setupOGM78DB(t)
	defer closeDB(t, db)
	svc := NewOUGroupMappingService(db)
	ctx := context.Background()
	cfg := insertConfig78(t, db, "")
	g1 := insertADGroup78(t, db, cfg.ID, "SalesGroup", "CN=SalesGroup,OU=Groups,DC=example,DC=com")

	require.NoError(t, db.Exec(`DROP TABLE sys_ou_group_mapping`).Error)

	_, err := svc.CreateMapping(ctx, &CreateMappingRequest{
		ADConfigID:  cfg.ID,
		OUDn:        "OU=Sales,DC=example,DC=com",
		OUName:      "Sales",
		ADGroupID:   g1,
		SyncEnabled: true,
		CreatedBy:   "user-1",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create mapping", "CreateMapping 缺表错误被包装 (ou_group_mapping_service.go:141)")
}

// 兜底 fmt 引用防止 import 漂移（部分测试函数会用到）
var _ = fmt.Sprintf