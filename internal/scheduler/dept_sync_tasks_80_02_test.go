package scheduler

// Phase 80 Plan 02 — dept_sync_tasks.go 测试
// 覆盖:RegisterDeptSyncTasks 注册 + pool stub 同包直注 + executeDeptToADSyncTask 分支
//
// D-80-02 口径:零 cron 触发时序断言,零 sleep,禁 t.Parallel。
// D-80-07:helper 一律 8002 后缀。

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/services/addomain"
)

// newSchedDB8002_DST sqlite 文件库。
func newSchedDB8002_DST(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "dst8002.db")
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.Job{}, &models.JobLog{}))
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

// newScheduler8002_DST scheduler fixture。
func newScheduler8002_DST(t *testing.T) (*Scheduler, *gorm.DB) {
	t.Helper()
	db := newSchedDB8002_DST(t)
	s := NewScheduler(db)
	s.SetLogger(&schedStubLogger8001{})
	return s, db
}

// stubADPool8002 DST pool stub。
type stubADPool8002 struct{}

func (s *stubADPool8002) PickAvailable(ctx context.Context, configID string) (*models.ADServiceAccount, error) {
	return nil, nil
}
func (s *stubADPool8002) ListAvailable(ctx context.Context, configID string) ([]models.ADServiceAccount, error) {
	return nil, nil
}
func (s *stubADPool8002) ListAll(ctx context.Context, configID string, page, pageSize int, statusFilter *int) ([]models.ADServiceAccount, int64, error) {
	return nil, 0, nil
}
func (s *stubADPool8002) CountByStatus(ctx context.Context, configID string) (total, available, disabled, circuitBroken int64, err error) {
	return 0, 0, 0, 0, nil
}
func (s *stubADPool8002) PickFirstAvailable(ctx context.Context, configID string) (*models.ADServiceAccount, error) {
	return nil, nil
}
func (s *stubADPool8002) RecoverExpiredBreakers(ctx context.Context) (int, error) {
	return 0, nil
}
func (s *stubADPool8002) InvalidateCache(configID string) {}
func (s *stubADPool8002) StartHotReload(ctx context.Context) error { return nil }
func (s *stubADPool8002) Create(ctx context.Context, account *models.ADServiceAccount) error { return nil }
func (s *stubADPool8002) Update(ctx context.Context, account *models.ADServiceAccount) error { return nil }
func (s *stubADPool8002) Delete(ctx context.Context, accountID string) error { return nil }
func (s *stubADPool8002) MarkSuccess(ctx context.Context, accountID string) error { return nil }
func (s *stubADPool8002) MarkFailure(ctx context.Context, accountID, reason string) error { return nil }
func (s *stubADPool8002) ManualUnlock(ctx context.Context, accountID, operator, reason string) error { return nil }
func (s *stubADPool8002) SetEnabled(ctx context.Context, accountID string, enabled bool) error { return nil }

var _ addomain.AccountPool = (*stubADPool8002)(nil)

// TestDst8002_RegisterDeptSyncTasks 注册函数。
func TestDst8002_RegisterDeptSyncTasks(t *testing.T) {
	s, _ := newScheduler8002_DST(t)
	RegisterDeptSyncTasks(s)
	assert.True(t, s.IsTaskRegistered("dept_to_ad_sync"))
	assert.True(t, s.IsTaskRegistered("dept_member_to_ad_group_sync"))
}

// TestDst8002_PoolSeam getGlobalADAccountPool → pool stub 同包直注。
func TestDst8002_PoolSeam(t *testing.T) {
	orig := globalADSyncScheduler
	t.Cleanup(func() { globalADSyncScheduler = orig })

	db := newSchedDB8002_DST(t)
	sched := newADSyncScheduler8002(t, db)
	sched.pool = &stubADPool8002{}
	globalADSyncScheduler = sched

	pool := getGlobalADAccountPool()
	assert.NotNil(t, pool)
}

// TestDst8002_GetDefaultADConfigIDForDept 空 config / 有 config 两分支。
func TestDst8002_GetDefaultADConfigIDForDept(t *testing.T) {
	db := newSchedDB8002_DST(t)
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS sys_ad_config (
		id TEXT PRIMARY KEY,
		config_name TEXT,
		status INTEGER DEFAULT 0,
		created_at DATETIME,
		deleted_at DATETIME
	)`).Error)

	// 分支 1:空表
	_, err := getDefaultADConfigIDForDept(context.Background(), db, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "未找到启用的AD配置")

	// 分支 2:有启用配置
	require.NoError(t, db.Exec(`INSERT INTO sys_ad_config (id, config_name, status) VALUES (?, ?, ?)`,
		"cfg-dst", "测试配置", models.ADConfigStatusEnabled).Error)
	id, err := getDefaultADConfigIDForDept(context.Background(), db, nil)
	require.NoError(t, err)
	assert.Equal(t, "cfg-dst", id)
}

// ============================================================================
// 缺口补足 Round 1 — pool nil / execute 双函数全分支
// ============================================================================

// newDeptSyncDB8002 sqlite 库 + dept_sync 全表(sys_ad_config / sys_dept / sys_ou_group_mapping)。
func newDeptSyncDB8002(t *testing.T) *gorm.DB {
	t.Helper()
	db := newSchedDB8002_DST(t)
	ddl := []string{
		`CREATE TABLE IF NOT EXISTS sys_ad_config (
			id TEXT PRIMARY KEY,
			config_name TEXT,
			sync_enabled INTEGER DEFAULT 1,
			status INTEGER DEFAULT 0,
			sync_interval INTEGER DEFAULT 3600,
			last_sync_at DATETIME,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)`,
		// models.Department 全列(BaseModel + NOT NULL 列),AutoMigrate 关联时不再加列
		`CREATE TABLE IF NOT EXISTS sys_dept (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER DEFAULT 0,
			dept_name TEXT NOT NULL DEFAULT '',
			dept_code TEXT NOT NULL DEFAULT '',
			parent_id TEXT,
			ancestors TEXT DEFAULT '',
			order_num INTEGER DEFAULT 0,
			is_external_org INTEGER DEFAULT 0,
			status INTEGER DEFAULT 0,
			remark TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS sys_ou_group_mapping (
			id TEXT PRIMARY KEY,
			ad_config_id TEXT,
			ou_dn TEXT,
			ou_name TEXT,
			ad_group_id TEXT,
			mapping_status TEXT DEFAULT 'active',
			sync_enabled INTEGER DEFAULT 1,
			last_sync_at DATETIME,
			created_at DATETIME,
			updated_at DATETIME
		)`,
	}
	for _, stmt := range ddl {
		require.NoError(t, db.Exec(stmt).Error)
	}
	return db
}

// TestDst8002_PoolNil globalADSyncScheduler=nil → getGlobalADAccountPool 返回 nil(记日志)。
func TestDst8002_PoolNil(t *testing.T) {
	orig := globalADSyncScheduler
	t.Cleanup(func() { globalADSyncScheduler = orig })
	globalADSyncScheduler = nil

	assert.Nil(t, getGlobalADAccountPool())
}

// TestDst8002_ExecuteDeptToAD_Errors executeDeptToADSyncTask 错误分支:
// GlobalDB nil / 无 config / config 停用。
func TestDst8002_ExecuteDeptToAD_Errors(t *testing.T) {
	db := newDeptSyncDB8002(t)

	origDB := GlobalDB
	t.Cleanup(func() { SetDB(origDB) })

	// GlobalDB nil → 错误
	SetDB(nil)
	require.Error(t, executeDeptToADSyncTask(context.Background(), nil))

	// 无启用 config → 错误
	SetDB(&stubDBGetter8001{db: db})
	require.Error(t, executeDeptToADSyncTask(context.Background(), nil))

	// config 停用 → 错误(经 params 显式传 configId:自动路径只查启用配置,够不到停用分支)
	require.NoError(t, db.Exec(`INSERT INTO sys_ad_config (id, config_name, status) VALUES (?, ?, ?)`,
		"cfg-dst-disabled", "停用配置", models.ADConfigStatusDisabled).Error)
	err := executeDeptToADSyncTask(context.Background(), map[string]interface{}{"configId": "cfg-dst-disabled"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "未启用")
}

// TestDst8002_ExecuteDeptToAD_Success executeDeptToADSyncTask 成功路径:
// 启用 config + 空 sys_dept → SyncDeptStructureToAD completed(无二级部门,跳过同步)。
func TestDst8002_ExecuteDeptToAD_Success(t *testing.T) {
	db := newDeptSyncDB8002(t)

	origDB := GlobalDB
	t.Cleanup(func() { SetDB(origDB) })
	SetDB(&stubDBGetter8001{db: db})

	orig := globalADSyncScheduler
	t.Cleanup(func() { globalADSyncScheduler = orig })
	sched := newADSyncScheduler8002(t, db)
	sched.pool = &stubADPool8002{}
	globalADSyncScheduler = sched

	require.NoError(t, db.Exec(`INSERT INTO sys_ad_config (id, config_name, status) VALUES (?, ?, ?)`,
		"cfg-dst-ok", "正常配置", models.ADConfigStatusEnabled).Error)

	require.NoError(t, executeDeptToADSyncTask(context.Background(), nil))
}

// TestDst8002_ExecuteMember_Errors executeDeptMemberToADGroupSyncTask 错误/跳过分支:
// GlobalDB nil / config 停用 / 空 mappings / ErrAllAccountsUnavailable。
func TestDst8002_ExecuteMember_Errors(t *testing.T) {
	db := newDeptSyncDB8002(t)

	origDB := GlobalDB
	t.Cleanup(func() { SetDB(origDB) })

	// GlobalDB nil → 错误
	SetDB(nil)
	require.Error(t, executeDeptMemberToADGroupSyncTask(context.Background(), nil))

	// 无 config → 错误
	SetDB(&stubDBGetter8001{db: db})
	require.Error(t, executeDeptMemberToADGroupSyncTask(context.Background(), nil))

	// config 停用 → 错误(params 显式传 configId)
	require.NoError(t, db.Exec(`INSERT INTO sys_ad_config (id, config_name, status) VALUES (?, ?, ?)`,
		"cfg-m-disabled", "停用配置", models.ADConfigStatusDisabled).Error)
	err := executeDeptMemberToADGroupSyncTask(context.Background(), map[string]interface{}{"configId": "cfg-m-disabled"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "未启用")

	// 启用 config + 无 mappings → nil(跳过)
	require.NoError(t, db.Exec(`INSERT INTO sys_ad_config (id, config_name, status) VALUES (?, ?, ?)`,
		"cfg-m-ok", "正常配置", models.ADConfigStatusEnabled).Error)
	require.NoError(t, executeDeptMemberToADGroupSyncTask(context.Background(), nil))

	// 有 mappings + 池无可用账号 → ErrAllAccountsUnavailable 包装错误
	require.NoError(t, db.Exec(`INSERT INTO sys_ou_group_mapping (id, ad_config_id, ou_dn, ou_name, ad_group_id) VALUES (?, ?, ?, ?, ?)`,
		"map-1", "cfg-m-ok", "OU=dept,DC=test", "研发部", "grp-1").Error)
	orig := globalADSyncScheduler
	t.Cleanup(func() { globalADSyncScheduler = orig })
	sched := newADSyncScheduler8002(t, db)
	sched.pool = &stubADPool8002{} // ListAvailable 返回空
	globalADSyncScheduler = sched

	err = executeDeptMemberToADGroupSyncTask(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "账号池无可用账号")
}

// TestDst8002_HandlerDispatch 经注册 handler 分发(GlobalDB nil → 错误,零 wire)。
func TestDst8002_HandlerDispatch(t *testing.T) {
	s, _ := newScheduler8002_DST(t)
	RegisterDeptSyncTasks(s)

	origDB := GlobalDB
	t.Cleanup(func() { SetDB(origDB) })
	SetDB(nil)

	require.Error(t, s.GetTaskHandler("dept_to_ad_sync")(context.Background(), nil))
	require.Error(t, s.GetTaskHandler("dept_member_to_ad_group_sync")(context.Background(), nil))
}

// ============================================================================
// 缺口补足 Round 2 — 参数键分支 / 查询错误分支 / 同步失败分支
// ============================================================================

// TestDst8002_GetDefaultADConfigIDForDept_ParamKeys adConfigId 参数键分支。
func TestDst8002_GetDefaultADConfigIDForDept_ParamKeys(t *testing.T) {
	db := newSchedDB8002_DST(t)
	ctx := context.Background()

	// adConfigId 命中
	id, err := getDefaultADConfigIDForDept(ctx, db, map[string]interface{}{"adConfigId": "via-adcfg"})
	require.NoError(t, err)
	assert.Equal(t, "via-adcfg", id)

	// configId 优先
	id, err = getDefaultADConfigIDForDept(ctx, db, map[string]interface{}{"configId": "a", "adConfigId": "b"})
	require.NoError(t, err)
	assert.Equal(t, "a", id)
}

// TestDst8002_GetDefaultADConfigIDForDept_QueryError 查询非 NotFound 错误分支(表缺失)。
func TestDst8002_GetDefaultADConfigIDForDept_QueryError(t *testing.T) {
	db := newSchedDB8002_DST(t) // 不建 sys_ad_config 表

	_, err := getDefaultADConfigIDForDept(context.Background(), db, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "查询AD配置失败")
}

// TestDst8002_ExecuteDeptToAD_ConfigQueryError configId 指向不存在行 → 查询配置失败分支。
func TestDst8002_ExecuteDeptToAD_ConfigQueryError(t *testing.T) {
	db := newDeptSyncDB8002(t)

	origDB := GlobalDB
	t.Cleanup(func() { SetDB(origDB) })
	SetDB(&stubDBGetter8001{db: db})

	orig := globalADSyncScheduler
	t.Cleanup(func() { globalADSyncScheduler = orig })
	sched := newADSyncScheduler8002(t, db)
	sched.pool = &stubADPool8002{}
	globalADSyncScheduler = sched

	err := executeDeptToADSyncTask(context.Background(), map[string]interface{}{"configId": "no-such-cfg"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "查询 AD 配置失败")
}

// TestDst8002_ExecuteDeptToAD_SyncError 启用 config + sys_dept 表缺失 →
// SyncDeptStructureToAD getRootDepartments 报错 → 同步失败上抛分支。
func TestDst8002_ExecuteDeptToAD_SyncError(t *testing.T) {
	db := newDeptSyncDB8002(t)
	require.NoError(t, db.Exec(`DROP TABLE sys_dept`).Error) // 触发 getRootDepartments 查询错误

	origDB := GlobalDB
	t.Cleanup(func() { SetDB(origDB) })
	SetDB(&stubDBGetter8001{db: db})

	orig := globalADSyncScheduler
	t.Cleanup(func() { globalADSyncScheduler = orig })
	sched := newADSyncScheduler8002(t, db)
	sched.pool = &stubADPool8002{}
	globalADSyncScheduler = sched

	require.NoError(t, db.Exec(`INSERT INTO sys_ad_config (id, config_name, status) VALUES (?, ?, ?)`,
		"cfg-dst-syncerr", "正常配置", models.ADConfigStatusEnabled).Error)

	err := executeDeptToADSyncTask(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "获取根部门失败")
}

// TestDst8002_ExecuteMember_ConfigQueryError configId 不存在 → 查询配置失败分支。
func TestDst8002_ExecuteMember_ConfigQueryError(t *testing.T) {
	db := newDeptSyncDB8002(t)

	origDB := GlobalDB
	t.Cleanup(func() { SetDB(origDB) })
	SetDB(&stubDBGetter8001{db: db})

	err := executeDeptMemberToADGroupSyncTask(context.Background(), map[string]interface{}{"configId": "no-such-cfg"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "查询AD配置失败")
}

// TestDst8002_ExecuteMember_MappingQueryError sys_ou_group_mapping 表缺失 → 映射查询错误分支。
func TestDst8002_ExecuteMember_MappingQueryError(t *testing.T) {
	db := newDeptSyncDB8002(t)
	require.NoError(t, db.Exec(`DROP TABLE sys_ou_group_mapping`).Error)

	origDB := GlobalDB
	t.Cleanup(func() { SetDB(origDB) })
	SetDB(&stubDBGetter8001{db: db})

	require.NoError(t, db.Exec(`INSERT INTO sys_ad_config (id, config_name, status) VALUES (?, ?, ?)`,
		"cfg-map-err", "正常配置", models.ADConfigStatusEnabled).Error)

	err := executeDeptMemberToADGroupSyncTask(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "查询OU-组映射失败")
}

// errListPool8002 ListAvailable 返回错误的 AccountPool stub →
// ExecuteWithFailover "查询账号池失败" 包装(非 ErrAllAccountsUnavailable)→ 通用错误分支。
type errListPool8002 struct{ stubADPool8002 }

func (e *errListPool8002) ListAvailable(ctx context.Context, configID string) ([]models.ADServiceAccount, error) {
	return nil, assert.AnError
}

var _ addomain.AccountPool = (*errListPool8002)(nil)

// TestDst8002_ExecuteMember_PoolQueryError 池查询错误 → "连接AD失败" 通用分支。
func TestDst8002_ExecuteMember_PoolQueryError(t *testing.T) {
	db := newDeptSyncDB8002(t)

	origDB := GlobalDB
	t.Cleanup(func() { SetDB(origDB) })
	SetDB(&stubDBGetter8001{db: db})

	require.NoError(t, db.Exec(`INSERT INTO sys_ad_config (id, config_name, status) VALUES (?, ?, ?)`,
		"cfg-pool-err", "正常配置", models.ADConfigStatusEnabled).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_ou_group_mapping (id, ad_config_id, ou_dn, ou_name, ad_group_id) VALUES (?, ?, ?, ?, ?)`,
		"map-pool-err", "cfg-pool-err", "OU=x,DC=test", "部门", "grp-x").Error)

	orig := globalADSyncScheduler
	t.Cleanup(func() { globalADSyncScheduler = orig })
	sched := newADSyncScheduler8002(t, db)
	sched.pool = &errListPool8002{}
	globalADSyncScheduler = sched

	err := executeDeptMemberToADGroupSyncTask(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "连接AD失败")
}
