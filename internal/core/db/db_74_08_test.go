package db

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/xingran-next/xingran-go-backend/internal/config"
	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// =====================================================================
// 74-08 Batch B: internal/core/db — FilterLogger Info/Warn/Error/Trace
// 分支 + Database keepalive/Close/GetDB + InitData/Bootstrap sqlite
// 真库路径 + createDatabaseIfNotExists 入参校验。
// =====================================================================

// ---------------- FilterLogger ----------------

func TestFilterLogger_InfoWarnBranches(t *testing.T) {
	// Silent:静默(只验证不 panic)
	l := NewFilterLogger(DefaultLogFilterConfig)
	l.Info(context.Background(), "silent-msg")
	l.Warn(context.Background(), "silent-msg")

	// Info 级别(4):Info/Warn 均输出(LogLevel >= msgLevel 语义)
	li := l.LogMode(logger.Info)
	assert.True(t, li.(*FilterLogger).shouldEmitInfo())
	assert.True(t, li.(*FilterLogger).shouldEmitWarn())
	li.Info(context.Background(), "info-msg")
	li.Info(context.Background(), "info-msg", "extra", 42)
	li.Warn(context.Background(), "info-level-warn")

	// Error 级别(2):Info/Warn 均静默
	le := l.LogMode(logger.Error)
	assert.False(t, le.(*FilterLogger).shouldEmitInfo())
	assert.False(t, le.(*FilterLogger).shouldEmitWarn())

	// Warn 级别(>= logger.Warn):Warn 输出
	ld := l.LogMode(logger.Warn)
	assert.True(t, ld.(*FilterLogger).shouldEmitWarn())
	ld.Warn(context.Background(), "warn-on")
	ld.Warn(context.Background(), "warn-on", "data")
}

func TestFilterLogger_Error(t *testing.T) {
	// FilterTypes[Error]=true → 静默
	l := NewFilterLogger(LogFilterConfig{
		FilterTypes: map[LogType]bool{LogTypeError: true},
	})
	assert.NotPanics(t, func() {
		l.Error(context.Background(), "filtered")
	})

	// 真实错误 → 输出;nil data / 非 error data / ErrRecordNotFound → 跳过
	l2 := NewFilterLogger(DefaultLogFilterConfig)
	l2.Error(context.Background(), "real-err", assert.AnError)
	l2.Error(context.Background(), "nil-data", nil)
	l2.Error(context.Background(), "not-err", "string-data")
	l2.Error(context.Background(), "record-not-found", logger.ErrRecordNotFound)
	l2.Error(context.Background(), "no-data")
}

func TestFilterLogger_TraceBranches(t *testing.T) {
	l := NewFilterLogger(DefaultLogFilterConfig)
	fc := func() (string, int64) { return "SELECT 1", 1 }

	// 快查询成功 → 静默
	assert.NotPanics(t, func() {
		l.Trace(context.Background(), time.Now(), fc, nil)
	})

	// 慢查询成功 → WARN 路径
	slow := NewFilterLogger(LogFilterConfig{
		FilterTypes:  map[LogType]bool{LogTypeSQL: true, LogTypeError: false},
		SlowThreshold: 1,
	})
	emit, msg, rows := slow.slowQueryLog(time.Now().Add(-50*time.Millisecond), fc)
	assert.True(t, emit)
	assert.Equal(t, int64(1), rows)
	assert.Contains(t, msg, "GORM慢查询")
	assert.NotPanics(t, func() {
		slow.Trace(context.Background(), time.Now().Add(-50*time.Millisecond), fc, nil)
	})

	// 错误路径:FilterTypes[Error]=false → Errorf;=true → 静默
	assert.NotPanics(t, func() {
		l.Trace(context.Background(), time.Now(), fc, assert.AnError)
	})
	quiet := NewFilterLogger(LogFilterConfig{
		FilterTypes: map[LogType]bool{LogTypeError: true},
	})
	assert.NotPanics(t, func() {
		quiet.Trace(context.Background(), time.Now(), fc, assert.AnError)
	})

	// ErrRecordNotFound → 视为成功路径(慢查询判定)
	assert.NotPanics(t, func() {
		l.Trace(context.Background(), time.Now().Add(-50*time.Millisecond), fc, logger.ErrRecordNotFound)
	})

	// FilterTypes[SQL]=false + 快查询 → 走保留分支(尾部)
	keep := NewFilterLogger(LogFilterConfig{
		FilterTypes:  map[LogType]bool{LogTypeSQL: false},
		SlowThreshold: 100000,
	})
	assert.NotPanics(t, func() {
		keep.Trace(context.Background(), time.Now(), fc, nil)
	})
}

// ---------------- Database keepalive / Close / GetDB ----------------

func TestDatabase_KeepaliveLifecycle(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "ka.db")
	d, err := NewDatabase(&config.DatabaseConfig{Type: "sqlite", Path: dbPath})
	require.NoError(t, err)

	// sqlite 分支不启动 keepalive(已有测试断言 nil) — 手动启动验证生命周期
	assert.Nil(t, d.keepaliveStop)
	d.startPoolKeepalive(2)
	require.NotNil(t, d.keepaliveStop)

	// 幂等:二次启动 no-op
	d.startPoolKeepalive(2)

	// ping 一轮(不 panic)
	d.pingPoolOnce(2)

	// Close 停止 goroutine 并等待 done
	require.NoError(t, d.Close())
	select {
	case <-d.keepaliveDone:
	default:
		t.Fatal("keepaliveDone 未关闭")
	}
	assert.Nil(t, d.keepaliveStop)

	// 二次 Close(keepaliveStop 已 nil)只关 DB,不 panic
	assert.NoError(t, d.Close())
}

func TestDatabase_GetDB(t *testing.T) {
	d := &Database{}
	assert.Nil(t, d.GetDB())

	dbPath := filepath.Join(t.TempDir(), "getdb.db")
	d2, err := NewDatabase(&config.DatabaseConfig{Type: "sqlite", Path: dbPath})
	require.NoError(t, err)
	defer d2.Close()
	assert.NotNil(t, d2.GetDB())
}

// ---------------- NewDatabase sqlite 错误分支 ----------------

func TestNewDatabase_NilConfig(t *testing.T) {
	_, err := NewDatabase(nil)
	assert.ErrorContains(t, err, "配置缺失")
}

func TestCreateSQLiteConnection_DirError(t *testing.T) {
	// 路径指向一个已存在文件(非目录)下的子路径 → MkdirAll 失败
	blocker := filepath.Join(t.TempDir(), "blocker.file")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o644))
	_, err := createSQLiteConnection(&config.DatabaseConfig{
		Type: "sqlite", Path: filepath.Join(blocker, "sub", "t.db"),
	})
	assert.ErrorContains(t, err, "创建SQLite数据目录失败")
}

// ---------------- InitData / Bootstrap sqlite 真库 ----------------

func newSQLiteGorm(t *testing.T) *gorm.DB {
	t.Helper()
	dir := t.TempDir()
	db, err := gorm.Open(sqlite.Open(filepath.Join(dir, "init.db")), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	if sqlDB, err := db.DB(); err == nil {
		t.Cleanup(func() { _ = sqlDB.Close() })
	}
	require.NoError(t, db.Migrator().CreateTable(&models.Department{}, &models.User{}, &models.Role{},
		&models.UserRole{}, &models.Menu{}, &models.Config{}, &models.Job{}, &models.JobLog{}))
	return db
}

func TestDatabase_InitData_SQLite(t *testing.T) {
	db := newSQLiteGorm(t)
	d := &Database{DB: db, Type: "sqlite"}

	// InitData 全链路(部门/用户/角色/关联/参数/菜单等种子)
	require.NoError(t, d.InitData())

	// 幂等:二次执行不报错
	require.NoError(t, d.InitData())

	// 种子落地断言
	var deptCount, userCount, roleCount int64
	require.NoError(t, db.Model(&models.Department{}).Count(&deptCount).Error)
	require.NoError(t, db.Model(&models.User{}).Count(&userCount).Error)
	require.NoError(t, db.Model(&models.Role{}).Count(&roleCount).Error)
	assert.Greater(t, deptCount, int64(0))
	assert.Greater(t, userCount, int64(0))
	assert.Greater(t, roleCount, int64(0))

	var configs int64
	require.NoError(t, db.Model(&models.Config{}).Where("config_key LIKE 'sys.%'").Count(&configs).Error)
	assert.Greater(t, configs, int64(0), "系统参数种子落地")
}

func TestDatabase_BootstrapMissingTables_SQLite(t *testing.T) {
	db := newSQLiteGorm(t)
	d := &Database{DB: db, Type: "sqlite"}

	// 无 sys_api_keys 表 → 补建 + 索引;幂等重入
	require.NoError(t, d.BootstrapMissingTables())
	require.NoError(t, d.BootstrapMissingTables())

	assert.True(t, db.Migrator().HasTable(&models.APIKey{}))
	assert.True(t, db.Migrator().HasTable(&models.APIKeyUsageLog{}))
}

// ---------------- createDatabaseIfNotExists 入参校验 ----------------

func TestCreateDatabaseIfNotExists_InvalidName(t *testing.T) {
	for _, name := range []string{"bad-name", "1starts-digit", "has space", "drop;--"} {
		err := createDatabaseIfNotExists("postgres://localhost:1/maint", name)
		assert.ErrorContains(t, err, "非法数据库名", name)
	}

	// 合法名但 DSN 不可达 → ping 失败错误
	err := createDatabaseIfNotExists("postgres://127.0.0.1:1/maint?connect_timeout=1", "valid_name")
	assert.Error(t, err)
}

// ---------------- AutoMigrate sqlite 全量 + PG 守卫 ----------------

func TestDatabase_AutoMigrate_SQLiteFull(t *testing.T) {
	d, err := NewDatabase(&config.DatabaseConfig{
		Type: "sqlite", Path: filepath.Join(t.TempDir(), "automigrate.db"),
	})
	require.NoError(t, err)
	defer d.Close()

	require.NoError(t, d.AutoMigrate())

	// 抽查关键表落地
	for _, m := range []interface{}{
		&models.User{}, &models.Role{}, &models.Menu{}, &models.Config{},
		&models.APIKey{}, &models.OperLog{}, &models.LoginLog{},
		&models.ADServiceAccount{}, &models.DictType{}, &models.DictData{},
		&models.Workstation{}, &models.UserPreference{},
	} {
		assert.True(t, d.DB.Migrator().HasTable(m), "%T 应建表", m)
	}

	// 幂等重入
	require.NoError(t, d.AutoMigrate())
}

func TestDatabase_AutoMigrate_SkipEnvOnlyPostgres(t *testing.T) {
	// SKIP_AUTOMIGRATE=true 仅对 postgres 生效;sqlite 仍执行全量
	t.Setenv("SKIP_AUTOMIGRATE", "true")
	d, err := NewDatabase(&config.DatabaseConfig{
		Type: "sqlite", Path: filepath.Join(t.TempDir(), "skipenv.db"),
	})
	require.NoError(t, err)
	defer d.Close()
	require.NoError(t, d.AutoMigrate())
	assert.True(t, d.DB.Migrator().HasTable(&models.User{}))
}

func TestDatabase_AutoMigrate_PGGuards(t *testing.T) {
	// type=postgres(手工构造,gorm DB 为 sqlite) → PG 分支守卫函数 + advisory lock
	// 失败路径(sqlite 不支持 pg_try_advisory_lock)应 fail-safe 跳过迁移块。
	db := newSQLiteGorm(t)
	d := &Database{DB: db, Type: "postgres"}

	// advisory lock:sqlite 上 pg_try_advisory_lock 报语法错误 → false(fail-safe)
	assert.False(t, d.acquireMigrationAdvisoryLock())

	// release 在 conn==nil 时 noop
	assert.NotPanics(t, d.releaseMigrationAdvisoryLock)

	// auditConstraintNaming: PG-only 查询失败仅 Debug 日志,不 panic
	assert.NotPanics(t, d.auditConstraintNaming)

	// cleanupOldConstraints / dropDependentMaterializedViews 在 sqlite 上不执行
	// (guard d.Type=="postgres" 内);直接调用 noop 文档函数不 panic
	assert.NotPanics(t, d.dropDependentMaterializedViews)
}

func TestDatabase_InitData_MissingTableFails(t *testing.T) {
	// 空库(无 sys_dept)→ initData 第一步即失败,错误应带 "创建默认部门"
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "empty.db")), &gorm.Config{})
	require.NoError(t, err)
	if sqlDB, err := db.DB(); err == nil {
		t.Cleanup(func() { _ = sqlDB.Close() })
	}
	d := &Database{DB: db, Type: "sqlite"}
	err = d.InitData()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "创建默认部门")
}
