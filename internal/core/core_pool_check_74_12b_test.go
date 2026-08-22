package core

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	coredb "github.com/xingran-next/xingran-go-backend/internal/core/db"
	"github.com/xingran-next/xingran-go-backend/internal/services/addomain"
)

// =====================================================================
// 74-11 escalation 补充: checkEmptyAccountPoolOnStartup + GetAuthFactory
// + SetBackgroundService —— 拉开 core 包对 P2 ratcheted floor(38.33%)
// 的测量余量: GetFromCachePool 的异步补充分支在并行时序下覆盖不稳定
// (288/754 vs 289/754 波动),本组确定性测试把 buffer 提到 2+ 语句。
// =====================================================================

// mockCorePool 嵌入接口本体:仅实现被消费的 CountByStatus,其余方法
// 落到嵌入的 nil 接口(调用即 panic — 正是"不该被调到"的期望)。
type mockCorePool struct {
	addomain.AccountPool // nil 嵌入
	counts               map[string][4]int64
	err                  error
	called               []string
}

func (m *mockCorePool) CountByStatus(ctx context.Context, configID string) (int64, int64, int64, int64, error) {
	m.called = append(m.called, configID)
	if m.err != nil {
		return 0, 0, 0, 0, m.err
	}
	c := m.counts[configID]
	return c[0], c[1], c[2], c[3], nil
}

// 编译期接口断言(防接口演化后 mock 静默失配)
var _ addomain.AccountPool = (*mockCorePool)(nil)

func newPoolCheckCore(t *testing.T, withCfg bool) *Core {
	t.Helper()
	gormDB, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "poolchk.db")), &gorm.Config{})
	require.NoError(t, err)
	if sqlDB, err := gormDB.DB(); err == nil {
		t.Cleanup(func() { _ = sqlDB.Close() })
	}
	require.NoError(t, gormDB.Exec(`CREATE TABLE sys_ad_config (
		id TEXT PRIMARY KEY, config_name TEXT, status INTEGER DEFAULT 0,
		sync_enabled BOOLEAN DEFAULT 1, server_address TEXT DEFAULT '', server_port INTEGER DEFAULT 389,
		domain_name TEXT DEFAULT '', base_dn TEXT DEFAULT '',
		admin_username TEXT DEFAULT '', admin_password TEXT DEFAULT '',
		use_ssl BOOLEAN DEFAULT 0, use_tls BOOLEAN DEFAULT 0, deleted_at DATETIME
	)`).Error)
	if withCfg {
		require.NoError(t, gormDB.Exec(`INSERT INTO sys_ad_config (id, config_name, status, sync_enabled)
			VALUES ('cfg-a', '生产域', 0, 1), ('cfg-b', '备份域', 0, 1)`).Error)
	}
	return &Core{CoreInfra: &CoreInfra{DB: &coredb.Database{DB: gormDB, Type: "sqlite"}}}
}

func TestCheckEmptyAccountPoolOnStartup(t *testing.T) {
	// 路径 1: 无表(查询失败) → Warn 后返回,不 panic
	c := newPoolCheckCore(t, false)
	require.NoError(t, c.DB.DB.Exec("DROP TABLE sys_ad_config").Error)
	assert.NotPanics(t, func() {
		c.checkEmptyAccountPoolOnStartup(&mockCorePool{})
	})

	// 路径 2: 无启用配置 → 循环零次,pool 不被调
	c2 := newPoolCheckCore(t, false)
	pool := &mockCorePool{}
	c2.checkEmptyAccountPoolOnStartup(pool)
	assert.Empty(t, pool.called)

	// 路径 3a: pool 查询出错 → continue 到下一配置
	c3 := newPoolCheckCore(t, true)
	errPool := &mockCorePool{err: errors.New("pool down")}
	assert.NotPanics(t, func() { c3.checkEmptyAccountPoolOnStartup(errPool) })
	assert.Len(t, errPool.called, 2, "出错 continue 仍遍历全部配置")

	// 路径 3b: 正常统计(cfg-a 空池触发告警分支,cfg-b 健康)
	c4 := newPoolCheckCore(t, true)
	okPool := &mockCorePool{counts: map[string][4]int64{
		"cfg-a": {0, 0, 0, 0}, // total=0 → 告警
		"cfg-b": {5, 3, 1, 1}, // 健康 → 无告警
	}}
	assert.NotPanics(t, func() { c4.checkEmptyAccountPoolOnStartup(okPool) })
	assert.ElementsMatch(t, []string{"cfg-a", "cfg-b"}, okPool.called)
}

func TestCore_GetAuthFactory(t *testing.T) {
	// Core 嵌入 *CoreInfra 指针,必须显式构造(零值字面量会 nil deref)
	c := &Core{CoreInfra: &CoreInfra{}}
	assert.Nil(t, c.GetAuthFactory())
}

func TestCaptchaService_SetBackgroundService(t *testing.T) {
	// setter 通路(nil 注入不 panic;非 nil 后续由 generateSliderWithBackground 覆盖)
	dbw := newPoolCheckCore(t, false).DB
	svc := NewCaptchaService(nil, nil)
	assert.NotPanics(t, func() { svc.SetBackgroundService(nil) })
	_ = dbw
}
