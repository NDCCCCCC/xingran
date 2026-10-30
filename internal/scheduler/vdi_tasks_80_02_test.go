package scheduler

// Phase 80 Plan 02 — vdi_sync + fix_suggestion 测试
// 覆盖:VDIVMService seam + executeVDIVMSyncTask 分发 + syncAll/singleVDIServer
// + SyncVDIVMsManually + RegisterFixSuggestionMisFixMonitor
//
// D-80-02 口径:零 cron 触发时序断言,零 sleep,禁 t.Parallel。
// D-80-07:helper 一律 8002 后缀。

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// newSchedDB8002_VDI sqlite 文件库 + sys_vdi_server DDL。
func newSchedDB8002_VDI(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "vdi8002.db")
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.Job{}, &models.JobLog{}))
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS sys_vdi_server (
		id TEXT PRIMARY KEY,
		name TEXT,
		status INTEGER DEFAULT 0,
		last_sync_time DATETIME,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	)`).Error)
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

// stubVDIVMService8002 VDIVMService stub。
type stubVDIVMService8002 struct {
	syncErr error
}

func (s *stubVDIVMService8002) SyncAllVMs(ctx context.Context, serverID string) error {
	return s.syncErr
}
func (s *stubVDIVMService8002) SyncVMsFromVDIByServer(ctx context.Context, server *models.VDIServer) error {
	return s.syncErr
}

var _ VDIVMService = (*stubVDIVMService8002)(nil)

// TestVdi8002_Seams SetVDIVMService save/restore。
func TestVdi8002_Seams(t *testing.T) {
	orig := GlobalVDIVMService
	t.Cleanup(func() { SetVDIVMService(orig) })

	stub := &stubVDIVMService8002{}
	SetVDIVMService(stub)
	assert.Same(t, stub, GlobalVDIVMService)

	SetVDIVMService(nil)
	assert.Nil(t, GlobalVDIVMService)
}

// TestVdi8002_ExecuteDispatch executeVDIVMSyncTask 分发到 stub。
func TestVdi8002_ExecuteDispatch(t *testing.T) {
	orig := GlobalVDIVMService
	t.Cleanup(func() { SetVDIVMService(orig) })
	SetVDIVMService(&stubVDIVMService8002{})

	origDB := GlobalDB
	t.Cleanup(func() { SetDB(origDB) })
	db := newSchedDB8002_VDI(t)
	SetDB(&stubDBGetter8001{db: db})

	err := executeVDIVMSyncTask(context.Background(), map[string]any{"param": "auto"})
	// stub 成功返回 nil
	require.NoError(t, err)
}

// TestVdi8002_SyncAllEnabled 同步所有启用服务器(空列表跳过)。
func TestVdi8002_SyncAllEnabled(t *testing.T) {
	db := newSchedDB8002_VDI(t)

	origDB := GlobalDB
	t.Cleanup(func() { SetDB(origDB) })
	SetDB(&stubDBGetter8001{db: db})

	// 无启用服务器 → 正常返回 nil
	err := syncAllEnabledVDIServers(context.Background(), db)
	require.NoError(t, err)
}

// TestVdi8002_SyncAllEnabled_Servers 多个服务器,分发到各 stub。
func TestVdi8002_SyncAllEnabled_Servers(t *testing.T) {
	db := newSchedDB8002_VDI(t)

	origDB := GlobalDB
	t.Cleanup(func() { SetDB(origDB) })
	SetDB(&stubDBGetter8001{db: db})

	origVdi := GlobalVDIVMService
	t.Cleanup(func() { SetVDIVMService(origVdi) })
	SetVDIVMService(&stubVDIVMService8002{})

	// 建启用服务器(DDL 由 fixture 建)
	require.NoError(t, db.Exec(`INSERT INTO sys_vdi_server (id, name, status) VALUES (?, ?, ?)`,
		"vdi-1", "VDI-1", 0).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_vdi_server (id, name, status) VALUES (?, ?, ?)`,
		"vdi-2", "VDI-2", 0).Error)

	err := syncAllEnabledVDIServers(context.Background(), db)
	require.NoError(t, err)
}

// TestVdi8002_SyncSingle 同步单个服务器不存在分支。
func TestVdi8002_SyncSingle_NotFound(t *testing.T) {
	db := newSchedDB8002_VDI(t)

	origDB := GlobalDB
	t.Cleanup(func() { SetDB(origDB) })
	SetDB(&stubDBGetter8001{db: db})

	err := syncSingleVDIServer(context.Background(), db, "non-existent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "不存在")
}

// TestVdi8002_SyncSingle_Disabled 同步单个服务器未启用分支。
func TestVdi8002_SyncSingle_Disabled(t *testing.T) {
	db := newSchedDB8002_VDI(t)

	origDB := GlobalDB
	t.Cleanup(func() { SetDB(origDB) })
	SetDB(&stubDBGetter8001{db: db})

	require.NoError(t, db.Exec(`INSERT INTO sys_vdi_server (id, name, status) VALUES (?, ?, ?)`,
		"vdi-disabled", "VDI-禁用", 1).Error) // status=1 停用

	err := syncSingleVDIServer(context.Background(), db, "vdi-disabled")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "未启用")
}

// TestVdi8002_SyncManually SyncVDIVMsManually 公开入口。
func TestVdi8002_SyncManually(t *testing.T) {
	db := newSchedDB8002_VDI(t)

	origDB := GlobalDB
	t.Cleanup(func() { SetDB(origDB) })
	SetDB(&stubDBGetter8001{db: db})

	origVdi := GlobalVDIVMService
	t.Cleanup(func() { SetVDIVMService(origVdi) })
	SetVDIVMService(&stubVDIVMService8002{})

	err := SyncVDIVMsManually(context.Background(), db, "auto")
	require.NoError(t, err)
}

// TestFsm8002_Register RegisterFixSuggestionMisFixMonitor → IsTaskRegistered。
func TestFsm8002_Register(t *testing.T) {
	db := newSchedDB8002_VDI(t)
	s := NewScheduler(db)
	s.SetLogger(&schedStubLogger8001{})

	RegisterFixSuggestionMisFixMonitor(s, db, nil, nil, nil)
	// RegisterFixSuggestionMisFixMonitor 只注册 reconciliation taskType
	// 我们验证 handler 被注册即可
	assert.True(t, s.IsTaskRegistered("reconciliation"))
}

// TestFsm8002_HandlerDispatch 误修复率监控 handler 分发:
// 非本监控 target → nil 跳过;monitor target → CheckAndNotify(软失败 nil)。
func TestFsm8002_HandlerDispatch(t *testing.T) {
	db := newSchedDB8002_VDI(t)
	s := NewScheduler(db)
	s.SetLogger(&schedStubLogger8001{})

	RegisterFixSuggestionMisFixMonitor(s, db, nil, nil, nil)
	handler := s.GetTaskHandler("reconciliation")
	require.NotNil(t, handler)

	// 非本监控任务(不同 target 字符串)→ 跳过分支返回 nil
	require.NoError(t, handler(context.Background(), map[string]any{"param": "someOtherTarget"}))

	// 本监控 target → CheckAndNotify(内部错误仅 Warnf 软失败,不 panic 即覆盖)
	require.NoError(t, handler(context.Background(), map[string]any{"param": "monitorFixSuggestionMisFix"}))
}

// ============================================================================
// 缺口补足 Round 1 — RegisterVDISyncTasks / 分发分支 / syncVDIServerVMs 三态
// ============================================================================

// TestVdi8002_RegisterTasks RegisterVDISyncTasks 注册 + handler 分发(GlobalDB nil → 错误)。
func TestVdi8002_RegisterTasks(t *testing.T) {
	s, _ := newScheduler8002_DST(t)
	RegisterVDISyncTasks(s)
	assert.True(t, s.IsTaskRegistered("vdi_vm_sync"))

	origDB := GlobalDB
	t.Cleanup(func() { SetDB(origDB) })
	SetDB(nil)
	require.Error(t, s.GetTaskHandler("vdi_vm_sync")(context.Background(), nil))
}

// TestVdi8002_ExecuteDispatch_SingleID executeVDIVMSyncTask 指定 serverID 分支。
func TestVdi8002_ExecuteDispatch_SingleID(t *testing.T) {
	db := newSchedDB8002_VDI(t)

	origDB := GlobalDB
	t.Cleanup(func() { SetDB(origDB) })
	SetDB(&stubDBGetter8001{db: db})

	origVdi := GlobalVDIVMService
	t.Cleanup(func() { SetVDIVMService(origVdi) })
	SetVDIVMService(&stubVDIVMService8002{})

	require.NoError(t, db.Exec(`INSERT INTO sys_vdi_server (id, name, status) VALUES ('vdi-target', 'VDI-目标', 0)`).Error)

	// 指定存在的 serverID → syncSingleVDIServer → 成功
	require.NoError(t, executeVDIVMSyncTask(context.Background(), map[string]any{"param": "vdi-target"}))
}

// TestVdi8002_SyncAll_QueryError syncAllEnabledVDIServers 查询错误分支(表缺失)。
func TestVdi8002_SyncAll_QueryError(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "vdierr8002.db")
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})

	err = syncAllEnabledVDIServers(context.Background(), db)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "查询 VDI 服务器失败")
}

// TestVdi8002_SyncAll_ServerFails 循环内单服务器失败 → failCount 分支(整体仍 nil)。
func TestVdi8002_SyncAll_ServerFails(t *testing.T) {
	db := newSchedDB8002_VDI(t)

	origDB := GlobalDB
	t.Cleanup(func() { SetDB(origDB) })
	SetDB(&stubDBGetter8001{db: db})

	origVdi := GlobalVDIVMService
	t.Cleanup(func() { SetVDIVMService(origVdi) })
	SetVDIVMService(&stubVDIVMService8002{syncErr: assert.AnError}) // 服务返回错误

	require.NoError(t, db.Exec(`INSERT INTO sys_vdi_server (id, name, status) VALUES ('vdi-bad', 'VDI-坏', 0)`).Error)

	require.NoError(t, syncAllEnabledVDIServers(context.Background(), db), "单服务器失败不阻断,整体返回 nil")
}

// TestVdi8002_SyncSingle_Success 单服务器成功路径(stub 返回 nil → last_sync_time 回写)。
func TestVdi8002_SyncSingle_Success(t *testing.T) {
	db := newSchedDB8002_VDI(t)

	origDB := GlobalDB
	t.Cleanup(func() { SetDB(origDB) })
	SetDB(&stubDBGetter8001{db: db})

	origVdi := GlobalVDIVMService
	t.Cleanup(func() { SetVDIVMService(origVdi) })
	SetVDIVMService(&stubVDIVMService8002{})

	require.NoError(t, db.Exec(`INSERT INTO sys_vdi_server (id, name, status) VALUES ('vdi-ok', 'VDI-正常', 0)`).Error)

	require.NoError(t, syncSingleVDIServer(context.Background(), db, "vdi-ok"))

	// last_sync_time 已回写
	var lastSync sql.NullTime
	require.NoError(t, db.Raw(`SELECT last_sync_time FROM sys_vdi_server WHERE id = ?`, "vdi-ok").Scan(&lastSync).Error)
	assert.True(t, lastSync.Valid, "成功同步后 last_sync_time 应回写")
}

// TestVdi8002_SyncSingle_ServiceNil 服务未注入 → 错误分支。
func TestVdi8002_SyncSingle_ServiceNil(t *testing.T) {
	db := newSchedDB8002_VDI(t)

	origDB := GlobalDB
	t.Cleanup(func() { SetDB(origDB) })
	SetDB(&stubDBGetter8001{db: db})

	origVdi := GlobalVDIVMService
	t.Cleanup(func() { SetVDIVMService(origVdi) })
	SetVDIVMService(nil)

	require.NoError(t, db.Exec(`INSERT INTO sys_vdi_server (id, name, status) VALUES ('vdi-nil', 'VDI-无服务', 0)`).Error)

	err := syncSingleVDIServer(context.Background(), db, "vdi-nil")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "未初始化")
}

// TestVdi8002_SyncSingle_SyncError 服务返回错误 → 错误包装分支。
func TestVdi8002_SyncSingle_SyncError(t *testing.T) {
	db := newSchedDB8002_VDI(t)

	origDB := GlobalDB
	t.Cleanup(func() { SetDB(origDB) })
	SetDB(&stubDBGetter8001{db: db})

	origVdi := GlobalVDIVMService
	t.Cleanup(func() { SetVDIVMService(origVdi) })
	SetVDIVMService(&stubVDIVMService8002{syncErr: assert.AnError})

	require.NoError(t, db.Exec(`INSERT INTO sys_vdi_server (id, name, status) VALUES ('vdi-err', 'VDI-错', 0)`).Error)

	err := syncSingleVDIServer(context.Background(), db, "vdi-err")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "同步失败")
}

// TestVdi8002_SyncManually_Single SyncVDIVMsManually 指定 serverID 入口。
func TestVdi8002_SyncManually_Single(t *testing.T) {
	db := newSchedDB8002_VDI(t)

	origDB := GlobalDB
	t.Cleanup(func() { SetDB(origDB) })
	SetDB(&stubDBGetter8001{db: db})

	origVdi := GlobalVDIVMService
	t.Cleanup(func() { SetVDIVMService(origVdi) })
	SetVDIVMService(&stubVDIVMService8002{})

	require.NoError(t, db.Exec(`INSERT INTO sys_vdi_server (id, name, status) VALUES ('vdi-manual', 'VDI-手动', 0)`).Error)

	require.NoError(t, SyncVDIVMsManually(context.Background(), db, "vdi-manual"))
}
