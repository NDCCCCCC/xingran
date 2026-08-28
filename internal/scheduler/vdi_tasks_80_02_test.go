package scheduler

// Phase 80 Plan 02 — vdi_sync + fix_suggestion 测试
// 覆盖:VDIVMService seam + executeVDIVMSyncTask 分发 + syncAll/singleVDIServer
// + SyncVDIVMsManually + RegisterFixSuggestionMisFixMonitor
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

	err := executeVDIVMSyncTask(context.Background(), map[string]interface{}{"param": "auto"})
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
