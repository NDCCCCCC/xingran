package addomain

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// ============ parentDNFromDN（域控审查 P0-2 扩展：RenameOU 依赖） ============
func TestParentDNFromDN(t *testing.T) {
	cases := []struct {
		name string
		dn   string
		want string
	}{
		{"普通OU", "OU=IT,DC=corp,DC=com", "DC=corp,DC=com"},
		{"多层OU", "OU=武汉中支,OU=湖北分公司,DC=corp,DC=com", "OU=湖北分公司,DC=corp,DC=com"},
		{"值含转义逗号", `CN=Zhang\,San,OU=IT,DC=corp,DC=com`, "OU=IT,DC=corp,DC=com"},
		{"值含字面反斜杠后逗号（反斜杠是值内容，逗号是真分隔符）", `CN=a\\,OU=IT,DC=com`, "OU=IT,DC=com"},
		{"只有一个RDN", "DC=com", ""},
		{"空串", "", ""},
		{"空白", "   ", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, parentDNFromDN(tc.dn))
		})
	}
}

// ============ escapeLike（域控审查 P1-5：LIKE 值转义） ============

func TestEscapeLike(t *testing.T) {
	assert.Equal(t, `OU=100\%组,DC=com`, escapeLike("OU=100%组,DC=com"))
	assert.Equal(t, `OU=a\_b,DC=com`, escapeLike("OU=a_b,DC=com"))
	assert.Equal(t, `OU=a\\b,DC=com`, escapeLike(`OU=a\b,DC=com`))
	assert.Equal(t, "OU=普通,DC=com", escapeLike("OU=普通,DC=com"))
}

// ============ DeptADCascadeService 空池 no-op（域控审查 P0-2） ============

func TestDeptADCascadeService_NilPoolNoop(t *testing.T) {
	db := testCascadeDB(t)
	var s *DeptADCascadeService // nil 接收者
	require.NoError(t, s.RenameDeptOU(context.Background(), "dept-1", "新名"))
	require.NoError(t, s.DeleteDeptOU(context.Background(), "dept-1"))
	require.NoError(t, s.PurgeMappingMembers(context.Background(), "m-1"))

	poolNil := NewDeptADCascadeService(db, nil)
	require.NoError(t, poolNil.RenameDeptOU(context.Background(), "dept-1", "新名"))
	require.NoError(t, poolNil.DeleteDeptOU(context.Background(), "dept-1"))
	require.NoError(t, poolNil.PurgeMappingMembers(context.Background(), "m-1"))
}

// TestDeptADCascadeService_PurgeMappingMembers_MappingMissing 幂等：映射不存在静默返回。
func TestDeptADCascadeService_PurgeMappingMembers_MappingMissing(t *testing.T) {
	db := testCascadeDB(t)
	s := NewDeptADCascadeService(db, nil)
	require.NoError(t, s.PurgeMappingMembers(context.Background(), "not-exists"))
}

// ============ writeMemberSyncLog（域控审查 P0-1：同步日志链路） ============

func TestWriteMemberSyncLog(t *testing.T) {
	db := testCascadeDB(t)
	require.NoError(t, db.AutoMigrate(&models.OUGroupMapping{}, &models.OUGroupMappingSyncLog{}))

	svc := NewOUGroupMappingService(db)
	mapping := models.OUGroupMapping{
		ID:         "map-1",
		ADConfigID: "cfg-1",
		OUDN:       "OU=IT,DC=corp,DC=com",
		OUName:     "IT",
		ADGroupID:  "grp-1",
	}
	require.NoError(t, db.Create(&mapping).Error)

	start := time.Now().Add(-50 * time.Millisecond)
	WriteMemberSyncLog(context.Background(), svc, &mapping, start, 3, 2, 10, "success", "")
	WriteMemberSyncLog(context.Background(), svc, &mapping, start, 0, 0, 10, "failed", "移除失败=1")

	var logs []models.OUGroupMappingSyncLog
	require.NoError(t, db.Where("mapping_id = ?", "map-1").Order("created_at").Find(&logs).Error)
	require.Len(t, logs, 2)

	assert.Equal(t, "member_sync", logs[0].SyncType)
	assert.Equal(t, 3, logs[0].MembersAdded)
	assert.Equal(t, 2, logs[0].MembersRemoved)
	assert.Equal(t, 10, logs[0].TotalMembers)
	assert.Equal(t, "success", logs[0].Status)
	assert.Nil(t, logs[0].ErrorMsg)
	require.NotNil(t, logs[0].CompletedAt)
	assert.GreaterOrEqual(t, logs[0].CompletedAt.Sub(logs[0].StartedAt), time.Duration(0))

	assert.Equal(t, "failed", logs[1].Status)
	require.NotNil(t, logs[1].ErrorMsg)
	assert.Equal(t, "移除失败=1", *logs[1].ErrorMsg)

	// last_sync_at 已通过 UpdateSyncStatus 更新
	var updated models.OUGroupMapping
	require.NoError(t, db.Where("id = ?", "map-1").First(&updated).Error)
	require.NotNil(t, updated.LastSyncAt)
}

// testCascadeDB 内存 sqlite + 级联/日志 helper 依赖表。
// 原生 DDL（不走 GORM AutoMigrate）：OUGroupMapping 模型带 PG-only 的
// default:gen_random_uuid() 片段，sqlite 直接 AutoMigrate 会语法错误
// （同 dept_ou_mismatch_service_test.go phase59 的口径）。
func testCascadeDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared&_busy_timeout=5000"
	gdb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	statements := []string{
		`CREATE TABLE sys_ou_group_mapping (
		    id TEXT PRIMARY KEY,
		    ad_config_id TEXT,
		    ou_dn TEXT,
		    ou_name TEXT,
		    ad_group_id TEXT,
		    last_sync_at TEXT,
		    updated_at TEXT
		)`,
		`CREATE TABLE sys_ou_group_mapping_sync_log (
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
		    started_at TEXT,
		    completed_at TEXT,
		    duration_ms INTEGER DEFAULT 0,
		    created_at TEXT
		)`,
	}
	for _, stmt := range statements {
		require.NoError(t, gdb.Exec(stmt).Error)
	}
	t.Cleanup(func() {
		if sqlDB, err := gdb.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return gdb
}
