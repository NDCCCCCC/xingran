package db

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lib/pq"
	"github.com/xingran-next/xingran-go-backend/internal/config"
)

// TestIsDuplicateDatabaseError 纯函数:isDuplicateDatabaseError 对 pq.Error{Code:"42P04"} 返回 true,
// 对普通 error 返回 false,对 nil 返回 false。
//
// CDX-H1 修复:createDatabaseIfNotExists 必须在 CREATE DATABASE 撞 42P04 (duplicate_database) 时
// 容忍并 WARN 而非返回错误(并发 bootstrap 场景)。
func TestIsDuplicateDatabaseError(t *testing.T) {
	// Case 1: pq.Error with Code 42P04 → true
	pqErr := &pq.Error{Code: "42P04", Message: "database \"foo\" already exists"}
	if !isDuplicateDatabaseError(pqErr) {
		t.Fatalf("isDuplicateDatabaseError(pq.Error{Code:42P04}) = false, want true")
	}

	// Case 2: wrapped pq.Error → true (errors.As 解包)
	wrapped := errWrap{pqErr}
	if !isDuplicateDatabaseError(wrapped) {
		t.Fatalf("isDuplicateDatabaseError(wrapped 42P04) = false, want true (errors.As unwrap)")
	}

	// Case 3: 普通 error → false
	plain := errors.New("some other db error")
	if isDuplicateDatabaseError(plain) {
		t.Fatalf("isDuplicateDatabaseError(plain error) = true, want false")
	}

	// Case 4: pq.Error with non-42P04 code → false
	other := &pq.Error{Code: "42P01", Message: "undefined_table"}
	if isDuplicateDatabaseError(other) {
		t.Fatalf("isDuplicateDatabaseError(pq.Error{Code:42P01}) = true, want false")
	}

	// Case 5: nil → false
	if isDuplicateDatabaseError(nil) {
		t.Fatalf("isDuplicateDatabaseError(nil) = true, want false")
	}
}

// TestCreatePostgresConnectionErrorPropagates 源码断言:createPostgresConnection 不再吞
// createDatabaseIfNotExists 的错误 —— 失败必须 return 错误而非仅 applogger.Errorf 后继续。
//
// CDX-H1 修复:真实错误(认证/网络/缺 postgres 维护库)必须上抛,启动 fail-fast 暴露根因。
func TestCreatePostgresConnectionErrorPropagates(t *testing.T) {
	src, err := os.ReadFile(filepath.Join(".", "database.go"))
	if err != nil {
		t.Fatalf("read database.go: %v", err)
	}
	s := string(src)

	// 1) createDatabaseIfNotExists 调用点附近必须 return 错误,而不是仅 Errorf 后继续
	//    用 grep -A3 模式:必须看到 return nil, fmt.Errorf 紧邻 createDatabaseIfNotExists(adminDSN
	needle := "createDatabaseIfNotExists(adminDSN"
	idx := strings.Index(s, needle)
	if idx < 0 {
		t.Fatalf("createDatabaseIfNotExists(adminDSN) call site not found in database.go")
	}
	// 取 needle 之后 300 字符的窗口
	window := s[idx : idx+minInt(300, len(s)-idx)]
	if !strings.Contains(window, "return nil, fmt.Errorf") {
		t.Fatalf("createPostgresConnection swallows createDatabaseIfNotExists error: "+
			"expected 'return nil, fmt.Errorf' within 300 chars after %q, got:\n%s",
			needle, window)
	}
	// 反向断言:原吞错的 applogger.Errorf("创建数据库失败: %v", err) 后不能直接 continue(无 return)
	// 即吞错模式下:Errorf 后无 return —— 修复后必须有 return
	if !strings.Contains(window, "return") {
		t.Fatalf("createPostgresConnection swallows createDatabaseIfNotExists error: "+
			"no 'return' found within 300 chars after %q (still swallow mode)", needle)
	}

	// 2) createDatabaseIfNotExists 函数体内必须含 42P04 判定代码
	if !strings.Contains(s, "42P04") {
		t.Fatalf("createDatabaseIfNotExists must reference 42P04 (duplicate_database) for race tolerance (CDX-H1)")
	}

	// 3) 包内必须有 isDuplicateDatabaseError 私有函数
	if !strings.Contains(s, "isDuplicateDatabaseError") {
		t.Fatalf("package must expose isDuplicateDatabaseError helper (CDX-H1)")
	}
}

// TestAdvisoryLockConcurrentMigrationProtection 源码断言:AutoMigrate 的 PG 迁移块
// 被 pg_try_advisory_lock(hashtext('xingran-migrations')) 包裹;未获锁分支含 applogger.Warnf
// 并跳过迁移块;defer 中有 pg_advisory_unlock 释放。
//
// C3 修复:多副本/滚动重启下 REFRESH CONCURRENTLY 与 CREATE OR REPLACE VIEW 竞态失败,
// 必须用 advisory lock 排他。
func TestAdvisoryLockConcurrentMigrationProtection(t *testing.T) {
	src, err := os.ReadFile(filepath.Join(".", "database.go"))
	if err != nil {
		t.Fatalf("read database.go: %v", err)
	}
	s := string(src)

	want := []string{
		// 1) 锁获取语句(字面匹配,锁定锁键名)
		"pg_try_advisory_lock(hashtext('xingran-migrations'))",
		// 2) 锁释放语句(defer 中)
		"pg_advisory_unlock",
		// 3) 专用连接 pinning(sqlDB.Conn)
		"sqlDB.Conn(",
		// 4) 未获锁分支的 WARN(skip)
		"[advisory-lock]",
	}
	for _, w := range want {
		if !strings.Contains(s, w) {
			t.Fatalf("database.go missing required fragment %q (C3 advisory lock protection)", w)
		}
	}

	// 5) 锁键名 'xingran-migrations' 必须出现(grep -c >=1)
	lockKey := "'xingran-migrations'"
	count := strings.Count(s, lockKey)
	if count < 2 {
		t.Fatalf("lock key 'xingran-migrations' must appear >=2 times (acquire + release), got %d", count)
	}
}

// errWrap 包一层 fmt.Errorf("...: %w", inner) 的测试 helper,模拟 errors.As 解包路径。
type errWrap struct {
	inner error
}

func (e errWrap) Error() string { return "wrapped: " + e.inner.Error() }
func (e errWrap) Unwrap() error { return e.inner }

// TestNewDatabaseSQLite 真实建库:Type="sqlite" 时 NewDatabase 经 glebarez/sqlite(纯 Go
// 驱动)连接本地文件库,返回 d.Type=="sqlite",ping 通过,且不启动 PG pool keepalive
// (d.keepaliveStop 为 nil — 本地文件无 TLS/auth 握手开销,保活无意义)。
//
// AutoMigrate 端到端:sqlite 下全量模型迁移必须成功 —— 回归守护 PG-only schema 片段
// (default:gen_random_uuid() / type:text[] 等)在 sqlite DDL 下的兼容性净化。
//
// 2026-08-17:恢复 dev 环境 SQLite 支持(纯 Go 驱动,区别于已删除的旧 CGO 路径)。
func TestNewDatabaseSQLite(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "t.db")

	d, err := NewDatabase(&config.DatabaseConfig{Type: "sqlite", Path: dbPath})
	if err != nil {
		t.Fatalf("NewDatabase(sqlite) error = %v, want nil", err)
	}
	defer func() {
		if cerr := d.Close(); cerr != nil {
			t.Errorf("Close() error = %v", cerr)
		}
	}()

	if d.Type != "sqlite" {
		t.Errorf("d.Type = %q, want %q", d.Type, "sqlite")
	}

	// sqlite 分支不启动 pool keepalive(本地文件无握手开销)
	if d.keepaliveStop != nil {
		t.Errorf("d.keepaliveStop != nil, want nil (sqlite branch must not start pool keepalive)")
	}

	// ping 验证连接可用
	sqlDB, err := d.DB.DB()
	if err != nil {
		t.Fatalf("d.DB.DB() error = %v", err)
	}
	if err := sqlDB.Ping(); err != nil {
		t.Errorf("Ping() error = %v, want nil", err)
	}

	// AutoMigrate 端到端:全量模型(含 default:gen_random_uuid() / text[] 的模型)
	// 在 sqlite 下必须建表成功
	if err := d.AutoMigrate(); err != nil {
		t.Fatalf("AutoMigrate() on sqlite error = %v, want nil (PG-only schema fragments must be sanitized)", err)
	}
	for _, table := range []string{"sys_user", "ops_asset", "sys_api_key_usage_logs", "sys_reconciliation_exception"} {
		if !d.DB.Migrator().HasTable(table) {
			t.Errorf("HasTable(%q) = false after AutoMigrate on sqlite, want true", table)
		}
	}
}

// TestNewDatabaseEmptyTypeFallsBackToPostgres 回归守护:database.type 缺省/为空时行为与
// 恢复 sqlite 分支之前完全一致 — 按 postgres 处理,Host 缺失时报与现有一致的
// fail-fast error,绝不静默落 SQLite。
func TestNewDatabaseEmptyTypeFallsBackToPostgres(t *testing.T) {
	_, err := NewDatabase(&config.DatabaseConfig{Type: "", Host: "", Port: 0})
	if err == nil {
		t.Fatalf("NewDatabase(empty type, missing host) error = nil, want fail-fast error")
	}
	if !strings.Contains(err.Error(), "数据库配置缺失") {
		t.Fatalf("NewDatabase(empty type, missing host) error = %q, want to contain %q",
			err.Error(), "数据库配置缺失")
	}
}

// minInt 取最小值(避免引入 Go 1.21+ 内置 min 与老版本冲突)。
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestNewDatabaseRequiresPostgresConfig 源码断言:NewDatabase 的 PG 分支在 Host==\"\" 或
// Port<=0 时必须返回错误(type=postgres 或缺省时 fail-fast,不静默落 SQLite)。
//
// 2026-08-15:cgo-gcc-missing-on-run — 删除旧 CGO sqlite 路径(gorm.io/driver/sqlite →
// mattn/go-sqlite3,Windows 无 gcc 时 go run 直接失败)。
// 2026-08-17:以纯 Go 驱动 glebarez/sqlite(底层 modernc.org/sqlite,无需 CGO)恢复
// sqlite 分支 — createSQLiteConnection 与 sqlite.Open( 重新合法;CGO 驱动禁令仍然有效,
// 由下方 banned 列表守护。
func TestNewDatabaseRequiresPostgresConfig(t *testing.T) {
	src, err := os.ReadFile(filepath.Join(".", "database.go"))
	if err != nil {
		t.Fatalf("read database.go: %v", err)
	}
	s := string(src)

	// 1) CGO sqlite 驱动必须消除(纯 Go glebarez/sqlite 是唯一合法 sqlite 路径)
	for _, banned := range []string{
		"func sqliteFallbackWarning(",
		"_ \"modernc.org/sqlite\"",
		"\"gorm.io/driver/sqlite\"",
	} {
		if strings.Contains(s, banned) {
			t.Fatalf("database.go must NOT contain %q (CGO sqlite path banned; cgo-gcc-missing-on-run fix)", banned)
		}
	}

	// 2) NewDatabase 必须 fail-fast on Host==\"\" 或 Port<=0(PG 分支)
	for _, required := range []string{
		"cfg.Host == \"\"",
		"cfg.Port <= 0",
		"数据库配置缺失",
	} {
		if !strings.Contains(s, required) {
			t.Fatalf("database.go missing required fragment %q (NewDatabase fail-fast)", required)
		}
	}
}

// TestBootstrapMissingTablesModelDerived 源码断言:BootstrapMissingTables 不再含硬编码
// CREATE TABLE DDL(APIKey schema 的第三份拷贝消除),表结构由 models.APIKey /
// models.APIKeyUsageLog 经 gorm.Migrator().CreateTable 派生;六条显式索引语句保留。
//
// C7 修复:bootstrap DDL 与 model 漂移 → 旁路路径建出错误表结构;CreateTable 与
// AutoMigrate 同源(MigrateModelList),单一事实源,天然防漂移。
func TestBootstrapMissingTablesModelDerived(t *testing.T) {
	src, err := os.ReadFile(filepath.Join(".", "database.go"))
	if err != nil {
		t.Fatalf("read database.go: %v", err)
	}
	s := string(src)

	// 1) 硬编码 CREATE TABLE DDL 必须消除(两个表都不允许)
	for _, banned := range []string{
		"CREATE TABLE IF NOT EXISTS public.sys_api_keys",
		"CREATE TABLE IF NOT EXISTS public.sys_api_key_usage_logs",
	} {
		if strings.Contains(s, banned) {
			t.Fatalf("database.go must NOT contain hardcoded DDL %q (C7: third-copy elimination)", banned)
		}
	}

	// 2) 必须经 gorm.Migrator().CreateTable 从 model 派生建表
	for _, required := range []string{
		"Migrator().CreateTable(&models.APIKey{})",
		"Migrator().CreateTable(&models.APIKeyUsageLog{})",
		// 先判定再补建(幂等)
		"Migrator().HasTable(&models.APIKey{})",
		"Migrator().HasTable(&models.APIKeyUsageLog{})",
	} {
		if !strings.Contains(s, required) {
			t.Fatalf("database.go missing required fragment %q (C7: model-derived bootstrap)", required)
		}
	}

	// 3) 六条 CREATE INDEX IF NOT EXISTS 显式索引语句全部保留(model tag 之外的索引面兜底)
	idxCount := strings.Count(s, "CREATE INDEX IF NOT EXISTS idx_api_keys_") +
		strings.Count(s, "CREATE INDEX IF NOT EXISTS idx_api_key_logs_")
	if idxCount != 6 {
		t.Fatalf("database.go must retain exactly 6 explicit CREATE INDEX IF NOT EXISTS statements "+
			"(idx_api_keys_* + idx_api_key_logs_*), got %d", idxCount)
	}
}

// TestNowFuncUtc 源码断言:database.go 不含 time.Now().Local();NowFunc 均为
// time.Now().UTC()(createPostgresConnection 一处;2026-08-15 删除 createSQLiteConnection
// 后仅保留 postgres NowFunc)。
//
// CDX-M-UTC 修复:NowFunc 全项目 UTC 一致性,与 SQL DEFAULT NOW() 语义对齐。
func TestNowFuncUtc(t *testing.T) {
	src, err := os.ReadFile(filepath.Join(".", "database.go"))
	if err != nil {
		t.Fatalf("read database.go: %v", err)
	}
	s := string(src)

	// 反向断言:不得保留 time.Now().Local()
	if strings.Contains(s, "time.Now().Local()") {
		t.Fatalf("database.go must NOT use time.Now().Local() (CDX-M-UTC): project-wide UTC")
	}

	// 正向断言:time.Now().UTC() 出现次数 >= 1(postgres NowFunc 保留)
	utcCount := strings.Count(s, "time.Now().UTC()")
	if utcCount < 1 {
		t.Fatalf("database.go must contain time.Now().UTC() >=1 times (postgres NowFunc), got %d", utcCount)
	}
}