package core

// =====================================================================
// Phase 78-01 Task 1+2: CaptchaService 三件套测试 (GenerateCaptcha 真实链路 +
// slider 全分支 + VerifySliderCaptcha + abs 剩余)。
//
// 关键纪律:
//   - since QUIRK-01 the IncrementBy 真实链路 is reachable —
//     MemoryCache 不实现 RateLimitCache,走 :265 Increment+Expire 降级分支;
//     74-08 时代的 captcha workaround 分支已作废(Phase 75 QUIRK-01 fix)。
//   - miniredis + RedisCache 走 :257 原子分支(IncrementWithExpire via Lua)。
//   - 装配 helper newCap78Mem / newCap78Redis 同包复用 + 78-02 wave 2 继承。
//   - t.Cleanup 关停一切 miniredis/RedisCache/MemoryCache(R-7 防护)。
//   - NO t.Parallel() — captcha 装配与包级全局共享 sqlite fixture。
// =====================================================================

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	coredb "github.com/xingran-next/xingran-go-backend/internal/core/db"
	"github.com/xingran-next/xingran-go-backend/pkg/cache"
	"github.com/xingran-next/xingran-go-backend/pkg/captcha"
)

// sysCaptchaBackgroundDDL 表结构精简到 GetRandomEnabled / IncrementUseCount 实际引用的列。
const sysCaptchaBackgroundDDL = `CREATE TABLE IF NOT EXISTS sys_captcha_background (
	id TEXT PRIMARY KEY,
	file_name TEXT,
	file_path TEXT,
	file_size INTEGER,
	file_width INTEGER,
	file_height INTEGER,
	file_md5 TEXT,
	piece_shape TEXT,
	difficulty_level INTEGER,
	allowed_shapes TEXT,
	status INTEGER,
	use_count INTEGER DEFAULT 0,
	sort_order INTEGER DEFAULT 0,
	last_used_at DATETIME,
	created_at DATETIME,
	updated_at DATETIME,
	deleted_at DATETIME
)`

// newCap78Mem 装配 CaptchaService + sqlite + MemoryCache + sys_captcha_background 表。
// 复制 core_74_08_test.go 的 newCaptchaTestDB 并扩展支持自定义背景图测试。
func newCap78Mem(t *testing.T, configs map[string]string) (*CaptchaService, *gorm.DB, *cache.MemoryCache) {
	t.Helper()
	gormDB, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "captcha78.db")), &gorm.Config{})
	require.NoError(t, err)
	if sqlDB, err := gormDB.DB(); err == nil {
		t.Cleanup(func() { _ = sqlDB.Close() })
	}
	require.NoError(t, gormDB.Exec(`CREATE TABLE sys_config (id TEXT PRIMARY KEY, config_key TEXT, config_value TEXT, deleted_at DATETIME)`).Error)
	for k, v := range configs {
		require.NoError(t, gormDB.Exec(`INSERT INTO sys_config (id, config_key, config_value) VALUES (?, ?, ?)`, "id-"+k, k, v).Error)
	}
	require.NoError(t, gormDB.Exec(sysCaptchaBackgroundDDL).Error)

	dbWrapper := &coredb.Database{DB: gormDB, Type: "sqlite"}
	mem := cache.NewMemoryCache(1000, 5*time.Minute)
	t.Cleanup(func() { _ = mem.Close() })
	return NewCaptchaService(dbWrapper, mem), gormDB, mem
}

// newCap78Redis 装配 CaptchaService + sqlite + miniredis + RedisCache(走 :257 原子分支)。
func newCap78Redis(t *testing.T, configs map[string]string) (*CaptchaService, *gorm.DB, *miniredis.Miniredis) {
	t.Helper()
	gormDB, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "captcha78r.db")), &gorm.Config{})
	require.NoError(t, err)
	if sqlDB, err := gormDB.DB(); err == nil {
		t.Cleanup(func() { _ = sqlDB.Close() })
	}
	require.NoError(t, gormDB.Exec(`CREATE TABLE sys_config (id TEXT PRIMARY KEY, config_key TEXT, config_value TEXT, deleted_at DATETIME)`).Error)
	for k, v := range configs {
		require.NoError(t, gormDB.Exec(`INSERT INTO sys_config (id, config_key, config_value) VALUES (?, ?, ?)`, "id-"+k, k, v).Error)
	}
	require.NoError(t, gormDB.Exec(sysCaptchaBackgroundDDL).Error)

	mr := miniredis.RunT(t)
	host, portStr, err := net.SplitHostPort(mr.Addr())
	require.NoError(t, err)
	port, err := strconv.Atoi(portStr)
	require.NoError(t, err)

	// NewRedisCache 内的 rdb.Ping 走完 go-redis 完整握手(INFRA-01 R-3 冒烟点)。
	rc, err := cache.NewRedisCache(&cache.CacheConfig{Host: host, Port: port}, "xingran")
	require.NoError(t, err)
	t.Cleanup(func() { _ = rc.Close() })

	dbWrapper := &coredb.Database{DB: gormDB, Type: "sqlite"}
	return NewCaptchaService(dbWrapper, rc), gormDB, mr
}

// makePNGBytes 用 stdlib 现场造一张可被 image.DecodeConfig 解析的真 PNG。
// LoadBackgroundFromFile 要求 image.RGBA 可读;真 PNG 字节保证 happy path。
// 注:Task 1 不调用此 helper,留作 Task 2 占位 import 占位以保证同文件测试可重排。
// (Task 2 追加后此注释移除)
func makePNGBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			img.Set(x, y, color.NRGBA{R: uint8(x % 255), G: uint8(y % 255), B: 128, A: 255})
		}
	}
	var buf strings.Builder
	tmp := filepath.Join(t.TempDir(), "src.png")
	f, err := os.Create(tmp)
	require.NoError(t, err)
	require.NoError(t, png.Encode(f, img))
	require.NoError(t, f.Close())
	data, err := os.ReadFile(tmp)
	require.NoError(t, err)
	_ = buf.String()
	return data
}

// TestCap78_GenerateCaptcha_MemoryCacheDegradedPath 验证 MemoryCache 不实现
// RateLimitCache 时走 :265 Increment+Expire 降级分支(QUIRK-01 修复后可直测)。
//
// since QUIRK-01 the IncrementBy 真实链路 is reachable — MemoryCache 不实现
// RateLimitCache,走 :265 Increment+Expire 降级分支;74-08 时代的 captcha
// workaround 分支已作废(Phase 75 QUIRK-01 fix 解锁 GenerateCaptcha 直测)。
func TestCap78_GenerateCaptcha_MemoryCacheDegradedPath(t *testing.T) {
	ctx := context.Background()
	svc, _, mem := newCap78Mem(t, map[string]string{
		"sys.account.captchaEnabled": "normal",
	})
	require.NoError(t, svc.LoadConfig(ctx))

	// 第 1 次 → Increment 缺 key 返回 1,触发 Expire
	resp, err := svc.GenerateCaptcha(ctx, "1.2.3.4")
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Contains(t, resp.CaptchaImg, "data:image/png;base64,")
	assert.Equal(t, captcha.CaptchaTypeNormal, resp.CaptchaType)

	// 限流键存在 + TTL > 0(:271 Expire 被调用)
	ttl, err := mem.TTL(ctx, fmt.Sprintf("captcha:rate:%s", "1.2.3.4"))
	require.NoError(t, err)
	assert.Greater(t, ttl, time.Duration(0), "降级分支 count==1 时 Expire 必须被调用")
}

// TestCap78_GenerateCaptcha_RateLimitCacheAtomicPath 验证 miniredis + RedisCache
// 走 :257 IncrementWithExpire 原子分支(Lua INCR+EXPIRE 复合)。
//
// since QUIRK-01 the IncrementBy 真实链路 is reachable — RedisCache 实现了
// RateLimitCache 接口,走 :257 原子分支;TTL 由 miniredis 提供,R-1 用 mr.TTL 断言。
func TestCap78_GenerateCaptcha_RateLimitCacheAtomicPath(t *testing.T) {
	ctx := context.Background()
	svc, _, mr := newCap78Redis(t, map[string]string{
		"sys.account.captchaEnabled": "normal",
	})
	require.NoError(t, svc.LoadConfig(ctx))

	resp, err := svc.GenerateCaptcha(ctx, "10.0.0.1")
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Contains(t, resp.CaptchaImg, "data:image/png;base64,")

	// miniredis 下带前缀 xingran:,ttl > 0 证明 Lua EXPIRE 已生效
	ttl := mr.TTL("xingran:captcha:rate:10.0.0.1")
	assert.Greater(t, ttl, time.Duration(0), "RateLimitCache 原子分支必须设置 TTL")
}

// TestCap78_GenerateCaptcha_RateLimitExceeded 表驱动:MemoryCache 与 miniredis
// 两装配各跑一遍,断言第 2 次调用因 ipRateLimit=1 被拒(:275 文案"过于频繁")。
func TestCap78_GenerateCaptcha_RateLimitExceeded(t *testing.T) {
	tests := []struct {
		name    string
		factory func(t *testing.T) *CaptchaService
	}{
		{"MemoryCache", func(t *testing.T) *CaptchaService {
			svc, _, _ := newCap78Mem(t, nil)
			return svc
		}},
		{"RedisCache", func(t *testing.T) *CaptchaService {
			svc, _, _ := newCap78Redis(t, nil)
			return svc
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			svc := tt.factory(t)
			require.NoError(t, svc.LoadConfig(ctx))
			svc.GetConfig().Enabled = captcha.CaptchaTypeNormal

			// ipRateLimit=1 → 第 2 次必拒
			require.NoError(t, svc.GetDB().Exec(
				`INSERT INTO sys_config (id, config_key, config_value) VALUES (?, ?, ?)`,
				"ip-1", "sys.captcha.ip_rate_limit", "1",
			).Error)

			resp1, err1 := svc.GenerateCaptcha(ctx, "9.9.9.9")
			require.NoError(t, err1)
			require.NotNil(t, resp1)

			resp2, err2 := svc.GenerateCaptcha(ctx, "9.9.9.9")
			assert.Nil(t, resp2)
			require.Error(t, err2)
			assert.Contains(t, err2.Error(), "过于频繁",
				"第 2 次调用应触发 :275 限流文案")
		})
	}
}

// TestCap78_GenerateCaptcha_FailClose 验证限流基础设施不可用时 fail-close(:260)。
// miniredis 装配下主动 mr.Close() → RedisCache 命令失败 → 业务返"服务繁忙"。
//
// since QUIRK-01 the IncrementBy 真实链路 is reachable — 限流原子分支在底层
// 不可用时仍走 fail-close 路径,业务拒绝而非降级放行(防绕过)。
func TestCap78_GenerateCaptcha_FailClose(t *testing.T) {
	ctx := context.Background()
	svc, _, mr := newCap78Redis(t, map[string]string{
		"sys.account.captchaEnabled": "normal",
	})
	require.NoError(t, svc.LoadConfig(ctx))

	// 主动关停 miniredis → 后续 RedisCache 命令必失败
	mr.Close()

	resp, err := svc.GenerateCaptcha(ctx, "5.5.5.5")
	assert.Nil(t, resp)
	require.Error(t, err, "底层 Redis 不可用 → fail-close 必返错")
	assert.Contains(t, err.Error(), "服务繁忙", ":260 文案应命中")
}

// TestCap78_GenerateCaptcha_Disabled 验证 :245 disabled 时返回 (nil, nil)。
func TestCap78_GenerateCaptcha_Disabled(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := newCap78Mem(t, nil)
	require.NoError(t, svc.LoadConfig(ctx))
	// 默认 Enabled = CaptchaTypeDisabled
	assert.Equal(t, captcha.CaptchaTypeDisabled, svc.GetConfig().Enabled)

	resp, err := svc.GenerateCaptcha(ctx, "0.0.0.0")
	require.NoError(t, err)
	assert.Nil(t, resp, "disabled 时应直接返回 (nil, nil) 而非调用 Increment")
}

// TestCap78_GetIPRateLimit_Edge 验证 getIPRateLimit 的三档语义:
// 无配置 → 10 / 非法值 → 10 / 合法值 25 → 25。
func TestCap78_GetIPRateLimit_Edge(t *testing.T) {
	ctx := context.Background()

	// 无配置 → 10
	svc1, _, _ := newCap78Mem(t, nil)
	assert.Equal(t, 10, svc1.getIPRateLimit(ctx))

	// 非法值 → 10
	svc2, db2, _ := newCap78Mem(t, nil)
	require.NoError(t, db2.Exec(
		`INSERT INTO sys_config (id, config_key, config_value) VALUES (?, ?, ?)`,
		"ip-bad", "sys.captcha.ip_rate_limit", "garbage",
	).Error)
	assert.Equal(t, 10, svc2.getIPRateLimit(ctx))

	// 非正值 → 10
	svc3, db3, _ := newCap78Mem(t, nil)
	require.NoError(t, db3.Exec(
		`INSERT INTO sys_config (id, config_key, config_value) VALUES (?, ?, ?)`,
		"ip-neg", "sys.captcha.ip_rate_limit", "-5",
	).Error)
	assert.Equal(t, 10, svc3.getIPRateLimit(ctx))

	// 合法值 → 25
	svc4, db4, _ := newCap78Mem(t, nil)
	require.NoError(t, db4.Exec(
		`INSERT INTO sys_config (id, config_key, config_value) VALUES (?, ?, ?)`,
		"ip-ok", "sys.captcha.ip_rate_limit", "25",
	).Error)
	assert.Equal(t, 25, svc4.getIPRateLimit(ctx))
}
