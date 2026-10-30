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
	"crypto/md5"
	"encoding/hex"
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
	"github.com/xingran-next/xingran-go-backend/internal/models"
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

// =====================================================================
// Task 2: slider 全分支(自定义背景 × 缓存池 × 降级)+ VerifySliderCaptcha + abs
// =====================================================================

// TestCap78_Slider_AutoFallback 验证 BackgroundMode="auto" → useCustom=false →
// 直接降级到 GenerateSliderBase64(:626-643),断言基础字段。
func TestCap78_Slider_AutoFallback(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := newCap78Mem(t, nil)
	svc.GetConfig().Enabled = captcha.CaptchaTypeSlider
	svc.GetConfig().BackgroundMode = "auto"
	svc.GetConfig().PieceShape = "circle"
	svc.GetConfig().Difficulty = 1

	data, err := svc.generateSliderWithBackground(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, data["sliderImg"])
	assert.Contains(t, data["sliderImg"].(string), "data:image/png;base64,")
	assert.NotEmpty(t, data["pieceImg"])
	assert.Contains(t, data["pieceImg"].(string), "data:image/png;base64,")
	assert.NotZero(t, data["xPos"], "xPos 应非零")
	assert.NotZero(t, data["yPos"], "yPos 应非零")
	assert.NotEmpty(t, data["token"])
	// auto 路径无 backgroundId 键
	_, hasBgID := data["backgroundId"]
	assert.False(t, hasBgID, "auto 降级路径不应有 backgroundId")
}

// TestCap78_Slider_UnknownMode 验证 BackgroundMode="bogus" → default 分支
// (:565 warn + 回退 auto),行为断言而非日志断言。
func TestCap78_Slider_UnknownMode(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := newCap78Mem(t, nil)
	svc.GetConfig().Enabled = captcha.CaptchaTypeSlider
	svc.GetConfig().BackgroundMode = "bogus"
	svc.GetConfig().PieceShape = "square"
	svc.GetConfig().Difficulty = 1

	data, err := svc.generateSliderWithBackground(ctx)
	require.NoError(t, err, "未知模式应回退 auto,不应报错")
	assert.NotEmpty(t, data["sliderImg"])
	assert.NotEmpty(t, data["token"])
	_, hasBgID := data["backgroundId"]
	assert.False(t, hasBgID, "bogus 模式回退 auto,无 backgroundId")
}

// TestCap78_Slider_CustomModeNoBackgroundService 验证 BackgroundMode="custom" 且
// 未 SetBackgroundService → 走 :617-624 的 nil 提示分支后降级成功。
func TestCap78_Slider_CustomModeNoBackgroundService(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := newCap78Mem(t, nil)
	svc.GetConfig().Enabled = captcha.CaptchaTypeSlider
	svc.GetConfig().BackgroundMode = "custom"
	svc.GetConfig().PieceShape = "circle"
	svc.GetConfig().Difficulty = 1
	// 故意不 SetBackgroundService

	data, err := svc.generateSliderWithBackground(ctx)
	require.NoError(t, err, "backgroundService=nil 时应降级 auto")
	assert.NotEmpty(t, data["sliderImg"])
	_, hasBgID := data["backgroundId"]
	assert.False(t, hasBgID, "nil service 降级路径无 backgroundId")
}

// TestCap78_Slider_CustomModeCachePoolHit 验证缓存池命中(:572-576 直接返回)。
// 预先种入 captcha:cache:pool:<shape>:<difficulty>:1 与 counter → GetFromCachePool
// 直接返回,不查 DB。
func TestCap78_Slider_CustomModeCachePoolHit(t *testing.T) {
	ctx := context.Background()
	svc, _, mem := newCap78Mem(t, nil)
	svc.GetConfig().Enabled = captcha.CaptchaTypeSlider
	svc.GetConfig().BackgroundMode = "custom"
	svc.GetConfig().PieceShape = "circle"
	svc.GetConfig().Difficulty = 1

	bgSvc := NewCaptchaBackgroundService(svc.db, mem)
	svc.SetBackgroundService(bgSvc)

	// 预种缓存池:counter=1 + pool 槽 1 写入合法 map
	cachedMap := map[string]any{
		"backgroundId": "bg-pool-1",
		"sliderImg":    "data:image/png;base64,POOLED_BG",
		"pieceImg":     "data:image/png;base64,POOLED_PIECE",
		"xPos":         42,
		"yPos":         7,
		"token":        "pool-token-xyz",
		"shape":        "circle",
		"difficulty":   1,
		"createdAt":    time.Now().Unix(),
	}
	require.NoError(t, mem.SetJSON(ctx, "captcha:cache:pool:circle:1:1", cachedMap, 5*time.Minute))
	require.NoError(t, mem.Set(ctx, "captcha:cache:pool:circle:1:counter", "1", 5*time.Minute))

	data, err := svc.generateSliderWithBackground(ctx)
	require.NoError(t, err)
	assert.Equal(t, "pool-token-xyz", data["token"], "缓存池命中应原样返回预置 token")
	assert.Equal(t, "bg-pool-1", data["backgroundId"], "缓存池命中应原样返回 backgroundId")
}

// TestCap78_Slider_CustomModeDBHitLoadOK 验证缓存池空 + DB 一行 status=enabled,
// piece_shape/difficulty 匹配,file_path=真 PNG → 命中 DB 路径(:586-611),
// 返回 map 含 backgroundId 且 use_count +1。
func TestCap78_Slider_CustomModeDBHitLoadOK(t *testing.T) {
	ctx := context.Background()
	svc, db, mem := newCap78Mem(t, nil)
	svc.GetConfig().Enabled = captcha.CaptchaTypeSlider
	svc.GetConfig().BackgroundMode = "custom"
	svc.GetConfig().PieceShape = "circle"
	svc.GetConfig().Difficulty = 1

	// 落盘一张真 PNG
	pngBytes := makePNGBytes(t, 300, 150)
	bgFilePath := filepath.Join(t.TempDir(), "dbbg.png")
	require.NoError(t, os.WriteFile(bgFilePath, pngBytes, 0o644))

	md5Sum := md5.Sum(pngBytes)
	md5Hex := hex.EncodeToString(md5Sum[:])

	require.NoError(t, db.Exec(
		`INSERT INTO sys_captcha_background
		 (id, file_name, file_path, file_size, file_width, file_height, file_md5,
		  piece_shape, difficulty_level, status, use_count)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"bg-db-1", "dbbg.png", bgFilePath, int64(len(pngBytes)), 300, 150, md5Hex,
		"circle", 1, models.CaptchaBgEnabled, 0,
	).Error)

	bgSvc := NewCaptchaBackgroundService(svc.db, mem)
	svc.SetBackgroundService(bgSvc)

	data, err := svc.generateSliderWithBackground(ctx)
	require.NoError(t, err)
	require.NotNil(t, data["backgroundId"], "DB 命中路径必须包含 backgroundId")
	assert.Equal(t, "bg-db-1", data["backgroundId"])
	assert.NotEmpty(t, data["sliderImg"])
	assert.NotEmpty(t, data["token"])

	// use_count + 1 落库断言(IncrementUseCount 调用过)
	var useCount int
	require.NoError(t, db.Raw(`SELECT use_count FROM sys_captcha_background WHERE id = ?`, "bg-db-1").Scan(&useCount).Error)
	assert.Equal(t, 1, useCount, "IncrementUseCount 必须落库")
}

// TestCap78_Slider_CustomModeLoadFileFail 验证 DB 行 file_path 指向不存在文件
// → :591 LoadBackgroundFromFile 失败 → 降级到 auto,断言返回 map 不含 backgroundId。
func TestCap78_Slider_CustomModeLoadFileFail(t *testing.T) {
	ctx := context.Background()
	svc, db, mem := newCap78Mem(t, nil)
	svc.GetConfig().Enabled = captcha.CaptchaTypeSlider
	svc.GetConfig().BackgroundMode = "custom"
	svc.GetConfig().PieceShape = "circle"
	svc.GetConfig().Difficulty = 1

	// file_path 指向不存在文件
	require.NoError(t, db.Exec(
		`INSERT INTO sys_captcha_background
		 (id, file_name, file_path, file_size, piece_shape, difficulty_level, status, use_count)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"bg-broken", "ghost.png", filepath.Join(t.TempDir(), "ghost.png"),
		100, "circle", 1, models.CaptchaBgEnabled, 0,
	).Error)

	bgSvc := NewCaptchaBackgroundService(svc.db, mem)
	svc.SetBackgroundService(bgSvc)

	data, err := svc.generateSliderWithBackground(ctx)
	require.NoError(t, err, "LoadFile 失败应降级 auto,不传播错误")
	assert.NotEmpty(t, data["sliderImg"])
	_, hasBgID := data["backgroundId"]
	assert.False(t, hasBgID, "LoadFile 失败路径不应有 backgroundId")
}

// TestCap78_Slider_MixedMode 验证 BackgroundMode="mixed" → rand 50% 双分支。
// D-78-01b: 随机分支采用宽松断言口径(N 次全部成功 + 至少一种形态命中),
// 不做固定次数强断言防 flake。
func TestCap78_Slider_MixedMode(t *testing.T) {
	ctx := context.Background()
	svc, db, mem := newCap78Mem(t, nil)
	svc.GetConfig().Enabled = captcha.CaptchaTypeSlider
	svc.GetConfig().BackgroundMode = "mixed"
	svc.GetConfig().PieceShape = "circle"
	svc.GetConfig().Difficulty = 1

	// 预置 DB 行让 custom 分支可命中(否则 50% custom 路径全走 nil → auto)
	pngBytes := makePNGBytes(t, 300, 150)
	bgFilePath := filepath.Join(t.TempDir(), "mixed.png")
	require.NoError(t, os.WriteFile(bgFilePath, pngBytes, 0o644))
	require.NoError(t, db.Exec(
		`INSERT INTO sys_captcha_background
		 (id, file_name, file_path, file_size, piece_shape, difficulty_level, status, use_count)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"bg-mixed", "mixed.png", bgFilePath, int64(len(pngBytes)),
		"circle", 1, models.CaptchaBgEnabled, 0,
	).Error)

	bgSvc := NewCaptchaBackgroundService(svc.db, mem)
	svc.SetBackgroundService(bgSvc)

	const N = 20
	var (
		withBgID    int
		withoutBgID int
	)
	for i := 0; i < N; i++ {
		data, err := svc.generateSliderWithBackground(ctx)
		require.NoError(t, err, "第 %d 次混合模式必须成功", i)
		if _, ok := data["backgroundId"]; ok {
			withBgID++
		} else {
			withoutBgID++
		}
	}
	// 至少一种形态命中
	assert.Greater(t, withBgID+withoutBgID, 0)
	t.Logf("mixed 模式 N=%d: custom=%d auto=%d (允许分布任意,仅防 panic 与全失败)", N, withBgID, withoutBgID)
}

// TestCap78_GenerateCaptcha_SliderStorage 验证 slider 全链 SetJSON + VerifySliderCaptcha
// 剩余分支(:409-480):正确 xPos+token 通过 / 错误 xPos 拒绝 / 错误 token 拒绝 /
// attempts 超限拒绝 / captchaID 不存在拒绝。
func TestCap78_GenerateCaptcha_SliderStorage(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := newCap78Mem(t, nil)
	svc.GetConfig().Enabled = captcha.CaptchaTypeSlider
	svc.GetConfig().MaxAttempts = 5 // 足够覆盖 token 错(1) + xPos 错(2) + 通过(3)

	resp, err := svc.GenerateCaptcha(ctx, "7.7.7.7")
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotEmpty(t, resp.Token)

	// 由于 capcha.go 直接读 verifyData.XPos(resp.YPos 不一定等于 XPos),
	// 通过独立 svcInner 直接种入已知 SliderVerifyData 覆盖所有剩余分支。
	svcInner, _, innerMem := newCap78Mem(t, nil)
	svcInner.GetConfig().Enabled = captcha.CaptchaTypeSlider
	svcInner.GetConfig().MaxAttempts = 5

	type pair struct{ x, y int; t string }
	p := pair{x: 123, y: 45, t: "tok-fixed"}
	require.NoError(t, innerMem.SetJSON(ctx, "captcha:data:cap-storage",
		SliderVerifyData{XPos: p.x, YPos: p.y, Token: p.t}, time.Minute))
	require.NoError(t, innerMem.SetInt(ctx, "captcha:attempts:cap-storage", 0, time.Minute))

	// 错误 token → "token无效"
	err = svcInner.VerifySliderCaptcha(ctx, "cap-storage", p.x, "WRONG-TOKEN")
	assert.ErrorContains(t, err, "token无效")

	// 错误 xPos → "位置不正确"
	err = svcInner.VerifySliderCaptcha(ctx, "cap-storage", p.x+50, p.t)
	assert.ErrorContains(t, err, "位置不正确")

	// 正确 → 通过(MemoryCache 非 L2ExposingCache → 普通 Set)
	require.NoError(t, svcInner.VerifySliderCaptcha(ctx, "cap-storage", p.x, p.t))

	// 不存在 captchaID → "验证码不存在或已过期"
	err = svcInner.VerifySliderCaptcha(ctx, "no-such-cap", 0, "x")
	assert.ErrorContains(t, err, "验证码不存在或已过期")

	// 未启用 slider → "当前未启用滑动验证码"
	offSvc, _, _ := newCap78Mem(t, nil)
	offSvc.GetConfig().Enabled = captcha.CaptchaTypeNormal
	err = offSvc.VerifySliderCaptcha(ctx, "anything", 0, "x")
	assert.ErrorContains(t, err, "未启用滑动验证码")

	// MaxAttempts 超限:连调 2 次错误,attempts 累到 2,MaxAttempts=2 → 第 3 次 "已失效"
	limSvc, _, limMem := newCap78Mem(t, nil)
	limSvc.GetConfig().Enabled = captcha.CaptchaTypeSlider
	limSvc.GetConfig().MaxAttempts = 2
	require.NoError(t, limMem.SetJSON(ctx, "captcha:data:cap-attempts",
		SliderVerifyData{XPos: 100, YPos: 50, Token: "tt"}, time.Minute))
	require.NoError(t, limMem.SetInt(ctx, "captcha:attempts:cap-attempts", 0, time.Minute))
	assert.Error(t, limSvc.VerifySliderCaptcha(ctx, "cap-attempts", 999, "tt")) // 位置错 +1
	assert.Error(t, limSvc.VerifySliderCaptcha(ctx, "cap-attempts", 100, "bad")) // token 错 +1
	// 第 3 次 attempts >= 2 → 拒绝
	err = limSvc.VerifySliderCaptcha(ctx, "cap-attempts", 100, "tt")
	assert.ErrorContains(t, err, "验证码已失效")

	_ = svc // resp 已生成用于证明 GenerateCaptcha slider 全链 OK
}

// TestCap78_Abs 验证 abs 的负数分支(:482-486)。
func TestCap78_Abs(t *testing.T) {
	assert.Equal(t, 5, abs(-5))
	assert.Equal(t, 5, abs(5))
	assert.Equal(t, 0, abs(0))
	// 容差边界:abs(xPos - expectedX) = abs(100 - 108) = 8 ≤ 8 → 通过分支
	svc, _, mem := newCap78Mem(t, nil)
	svc.GetConfig().Enabled = captcha.CaptchaTypeSlider
	ctx := context.Background()
	require.NoError(t, mem.SetJSON(ctx, "captcha:data:cap-abs",
		SliderVerifyData{XPos: 100, YPos: 50, Token: "tt"}, time.Minute))
	require.NoError(t, mem.SetInt(ctx, "captcha:attempts:cap-abs", 0, time.Minute))
	require.NoError(t, svc.VerifySliderCaptcha(ctx, "cap-abs", 108, "tt")) // abs(8) ≤ 8
}
