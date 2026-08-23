package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	coredb "github.com/xingran-next/xingran-go-backend/internal/core/db"
	"github.com/xingran-next/xingran-go-backend/pkg/cache"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/pkg/captcha"
)

// =====================================================================
// 74-08 Batch A: internal/core — CaptchaService 配置加载/登录锁定/
// 验证码生成验证(MemoryCache)+ loadConnectionPoolConfig + parseDuration
// + initSM4Cipher + MetricsCacheService。
// (core.New 完整 Init 链路依赖 Redis/调度器/子进程,不在单测范围)
// =====================================================================

// newCaptchaTestDB 构造带 sys_config 表的 sqlite Database + MemoryCache。
func newCaptchaTestDB(t *testing.T, configs map[string]string) (*CaptchaService, *gorm.DB, *cache.MemoryCache) {
	t.Helper()
	gormDB, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "captcha.db")), &gorm.Config{})
	require.NoError(t, err)
	if sqlDB, err := gormDB.DB(); err == nil {
		t.Cleanup(func() { _ = sqlDB.Close() })
	}
	require.NoError(t, gormDB.Exec(`CREATE TABLE sys_config (id TEXT PRIMARY KEY, config_key TEXT, config_value TEXT, deleted_at DATETIME)`).Error)
	for k, v := range configs {
		require.NoError(t, gormDB.Exec(`INSERT INTO sys_config (id, config_key, config_value) VALUES (?, ?, ?)`, "id-"+k, k, v).Error)
	}

	dbWrapper := &coredb.Database{DB: gormDB, Type: "sqlite"}
	mem := cache.NewMemoryCache(1000, 5*time.Minute)
	t.Cleanup(func() { _ = mem.Close() })
	return NewCaptchaService(dbWrapper, mem), gormDB, mem
}

// ---------------- LoadConfig ----------------

func TestCaptchaService_LoadConfig(t *testing.T) {
	ctx := context.Background()

	// 空配置 → 全部默认值
	svc, _, _ := newCaptchaTestDB(t, nil)
	require.NoError(t, svc.LoadConfig(ctx))
	cfg := svc.GetConfig()
	assert.Equal(t, captcha.CaptchaTypeDisabled, cfg.Enabled)
	assert.Equal(t, 4, cfg.Type)
	assert.Equal(t, 5, cfg.ExpireTime)
	assert.Equal(t, 3, cfg.MaxAttempts)
	assert.Equal(t, 10, cfg.IPRateLimit)
	assert.Equal(t, 5, cfg.LoginMaxRetry)
	assert.Equal(t, 30, cfg.LoginLockTime)
	assert.Equal(t, "mixed", cfg.BackgroundMode)
	assert.Equal(t, "circle", cfg.PieceShape)
	assert.Equal(t, 1, cfg.Difficulty)
	assert.False(t, svc.IsEnabled())
	assert.NotNil(t, svc.GetDB())

	// 显式配置覆盖 + 非法开关值回退 normal
	svc2, _, _ := newCaptchaTestDB(t, map[string]string{
		"sys.account.captchaEnabled":     "slider",
		"sys.account.captchaType":        "6",
		"sys.account.captchaExpireTime":  "10",
		"sys.account.captchaMaxAttempts": "1",
		"sys.account.ipRateLimit":        "3",
		"sys.account.loginMaxRetry":      "2",
		"sys.account.loginLockTime":      "15",
		"sys.account.captchaPieceShape":  "star",
		"sys.account.captchaDifficulty":  "3",
	})
	require.NoError(t, svc2.LoadConfig(ctx))
	cfg2 := svc2.GetConfig()
	assert.Equal(t, captcha.CaptchaTypeSlider, cfg2.Enabled)
	assert.Equal(t, 6, cfg2.Type)
	assert.Equal(t, 10, cfg2.ExpireTime)
	assert.Equal(t, 1, cfg2.MaxAttempts)
	assert.Equal(t, 3, cfg2.IPRateLimit)
	assert.Equal(t, 2, cfg2.LoginMaxRetry)
	assert.Equal(t, 15, cfg2.LoginLockTime)
	assert.Equal(t, "star", cfg2.PieceShape)
	assert.Equal(t, 3, cfg2.Difficulty)
	assert.True(t, svc2.IsEnabled())

	// 非法开关值 → 回退 normal(偏安全)
	svc3, _, _ := newCaptchaTestDB(t, map[string]string{
		"sys.account.captchaEnabled": "weird-mode",
	})
	require.NoError(t, svc3.LoadConfig(ctx))
	assert.Equal(t, captcha.CaptchaTypeNormal, svc3.GetConfig().Enabled)

	// 非法数字 → Sscanf 失败 → 保留默认(错误被吞)
	svc4, _, _ := newCaptchaTestDB(t, map[string]string{
		"sys.account.captchaType": "not-a-number",
	})
	require.NoError(t, svc4.LoadConfig(ctx))
	assert.Equal(t, 4, svc4.GetConfig().Type, "非法数字保留默认值")
}

// ---------------- GenerateCaptcha / VerifyCaptcha(数字模式) ----------------

func TestCaptchaService_GenerateAndVerify_Normal(t *testing.T) {
	ctx := context.Background()
	svc, _, mem := newCaptchaTestDB(t, map[string]string{
		"sys.account.captchaEnabled": "normal",
	})
	require.NoError(t, svc.LoadConfig(ctx))

	// 禁用后 VerifyCaptcha 直接通过
	svc.GetConfig().Enabled = captcha.CaptchaTypeDisabled
	resp, err := svc.GenerateCaptcha(ctx, "1.2.3.4")
	require.NoError(t, err)
	assert.Nil(t, resp, "禁用时返回 nil")
	assert.NoError(t, svc.VerifyCaptcha(ctx, "any", "any", "1.2.3.4"))
	assert.ErrorContains(t, svc.VerifySliderCaptcha(ctx, "any", 1, "t"), "当前未启用滑动验证码", "禁用模式滑块验证提示未启用")

	// 启用 normal 模式后,GenerateCaptcha 在 MemoryCache 下应真实生成验证码
	svc.GetConfig().Enabled = captcha.CaptchaTypeNormal
	resp, err = svc.GenerateCaptcha(ctx, "1.2.3.4")
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.CaptchaID)
	assert.Contains(t, resp.CaptchaImg, "data:image/png;base64,")

	storedCode, err := mem.Get(ctx, fmt.Sprintf("captcha:data:%s", resp.CaptchaID))
	require.NoError(t, err)

	// 错误输入 → 验证码错误
	assert.ErrorContains(t, svc.VerifyCaptcha(ctx, resp.CaptchaID, "WRONG", "1.2.3.4"), "验证码错误")

	// 正确输入 → 通过
	assert.NoError(t, svc.VerifyCaptcha(ctx, resp.CaptchaID, storedCode, "1.2.3.4"))

	// 二次验证(数据已删) → 不存在
	assert.ErrorContains(t, svc.VerifyCaptcha(ctx, resp.CaptchaID, storedCode, "1.2.3.4"), "验证码不存在或已过期")

	// 不存在的验证码 → 不存在
	assert.ErrorContains(t, svc.VerifyCaptcha(ctx, "no-such-id", "x", "1.2.3.4"), "验证码不存在或已过期")
}

func TestCaptchaService_VerifyCaptcha_MaxAttempts(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := newCaptchaTestDB(t, map[string]string{
		"sys.account.captchaEnabled":     "normal",
		"sys.account.captchaMaxAttempts": "2",
	})
	require.NoError(t, svc.LoadConfig(ctx))

	// 启用 normal 模式后真实生成验证码
	svc.GetConfig().Enabled = captcha.CaptchaTypeNormal
	resp, err := svc.GenerateCaptcha(ctx, "9.9.9.9")
	require.NoError(t, err)
	require.NotNil(t, resp)

	// 2 次错误后(attempts 1,2),第 3 次 attempts>=2 → "已失效"
	assert.Error(t, svc.VerifyCaptcha(ctx, resp.CaptchaID, "bad1", "9.9.9.9"))
	assert.Error(t, svc.VerifyCaptcha(ctx, resp.CaptchaID, "bad2", "9.9.9.9"))
	assert.ErrorContains(t, svc.VerifyCaptcha(ctx, resp.CaptchaID, "bad3", "9.9.9.9"), "验证码已失效")
}

func TestCaptchaService_VerifySliderCaptcha(t *testing.T) {
	ctx := context.Background()
	svc, _, mem := newCaptchaTestDB(t, nil)
	svc.GetConfig().Enabled = captcha.CaptchaTypeSlider
	svc.GetConfig().MaxAttempts = 3

	// 手工写验证数据进缓存
	storageKey := "captcha:data:cap-1"
	require.NoError(t, mem.SetJSON(ctx, storageKey, SliderVerifyData{XPos: 100, YPos: 50, Token: "tok-1"}, time.Minute))
	_ = mem.SetInt(ctx, "captcha:attempts:cap-1", 0, time.Minute)

	// 位置错误(容差 8px)
	assert.ErrorContains(t, svc.VerifySliderCaptcha(ctx, "cap-1", 200, "tok-1"), "位置不正确")
	// 位置对但 token 错
	assert.ErrorContains(t, svc.VerifySliderCaptcha(ctx, "cap-1", 105, "wrong"), "token无效")
	// 全对 → 通过(MemoryCache 非 L2ExposingCache → 普通写 verified 标记)
	assert.NoError(t, svc.VerifySliderCaptcha(ctx, "cap-1", 100, "tok-1"))

	// verified 标记已写 → VerifyCaptcha "verified" 通过
	assert.NoError(t, svc.VerifyCaptcha(ctx, "cap-1", "verified", "1.1.1.1"))
	// 错误 input → 验证码无效
	require.NoError(t, mem.Set(ctx, "captcha:verified:cap-2", "1", time.Minute))
	assert.ErrorContains(t, svc.VerifyCaptcha(ctx, "cap-2", "bogus", "1.1.1.1"), "验证码无效")

	// 数据不存在
	assert.ErrorContains(t, svc.VerifySliderCaptcha(ctx, "ghost", 1, "t"), "验证码不存在或已过期")

	// 未启用 slider → 提示未启用
	svc.GetConfig().Enabled = captcha.CaptchaTypeNormal
	assert.ErrorContains(t, svc.VerifySliderCaptcha(ctx, "cap-1", 100, "tok-1"), "未启用滑动验证码")
}

// ---------------- 登录锁定 ----------------

func TestCaptchaService_LoginLock(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := newCaptchaTestDB(t, map[string]string{
		"sys.account.loginMaxRetry": "3",
		"sys.account.loginLockTime": "30",
	})
	require.NoError(t, svc.LoadConfig(ctx))

	// 未锁定
	assert.NoError(t, svc.CheckLoginLock(ctx, "alice"))

	// 连续调用 RecordLoginFailure,缺 key 时 Increment 从 1 起算自然累积
	// 失败 1 次(3 次的一半阈值以下不提示剩余)
	err := svc.RecordLoginFailure(ctx, "alice")
	assert.ErrorContains(t, err, "用户名或密码错误")

	// 失败 2 次 → 剩余 1 次(<= max/2=1)
	err = svc.RecordLoginFailure(ctx, "alice")
	assert.ErrorContains(t, err, "还可尝试 1 次")

	// 失败 3 次 → 锁定
	err = svc.RecordLoginFailure(ctx, "alice")
	assert.ErrorContains(t, err, "账号已被锁定 30 分钟")

	// CheckLoginLock 检出锁定
	err = svc.CheckLoginLock(ctx, "alice")
	assert.ErrorContains(t, err, "账号已锁定")

	// 清除后可再失败
	svc.ClearLoginFailure(ctx, "bob")
	assert.NoError(t, svc.CheckLoginLock(ctx, "bob"))
}

// ---------------- IP 限流 ----------------

func TestCaptchaService_IPRateLimit(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := newCaptchaTestDB(t, map[string]string{
		"sys.account.captchaEnabled": "normal",
		"sys.account.ipRateLimit":    "2",
	})
	require.NoError(t, svc.LoadConfig(ctx))
	assert.Equal(t, 2, svc.GetConfig().IPRateLimit)
	_ = ctx
}

func TestCaptchaService_GetIPRateLimit(t *testing.T) {
	ctx := context.Background()

	// 无配置 → 默认 10
	svc, _, _ := newCaptchaTestDB(t, nil)
	assert.Equal(t, 10, svc.getIPRateLimit(ctx))

	// 非法/非正值 → 默认
	svc2, db2, _ := newCaptchaTestDB(t, nil)
	require.NoError(t, db2.Exec(`INSERT INTO sys_config (id, config_key, config_value) VALUES ('x1', 'sys.captcha.ip_rate_limit', 'garbage')`).Error)
	assert.Equal(t, 10, svc2.getIPRateLimit(ctx))
	require.NoError(t, db2.Exec(`UPDATE sys_config SET config_value = '-5' WHERE config_key = 'sys.captcha.ip_rate_limit'`).Error)
	assert.Equal(t, 10, svc2.getIPRateLimit(ctx))

	// 合法值
	svc3, db3, _ := newCaptchaTestDB(t, nil)
	require.NoError(t, db3.Exec(`INSERT INTO sys_config (id, config_key, config_value) VALUES ('x2', 'sys.captcha.ip_rate_limit', '25')`).Error)
	assert.Equal(t, 25, svc3.getIPRateLimit(ctx))
}

// ---------------- 滑块生成(auto 降级) ----------------

func TestCaptchaService_GenerateSlider_AutoFallback(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := newCaptchaTestDB(t, nil)
	svc.GetConfig().Enabled = captcha.CaptchaTypeSlider
	// backgroundService 为 nil → auto 降级路径
	data, err := svc.generateSliderWithBackground(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, data["sliderImg"])
	assert.NotEmpty(t, data["pieceImg"])
	assert.NotEmpty(t, data["token"])
}

// ---------------- loadConnectionPoolConfig / parseDuration / initSM4Cipher ----------------

func TestLoadConnectionPoolConfig(t *testing.T) {
	gormDB, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "pool.db")), &gorm.Config{})
	require.NoError(t, err)
	if sqlDB, err := gormDB.DB(); err == nil {
		t.Cleanup(func() { _ = sqlDB.Close() })
	}
	require.NoError(t, gormDB.Exec(`CREATE TABLE sys_config (id TEXT PRIMARY KEY, config_key TEXT, config_value TEXT, deleted_at DATETIME)`).Error)

	// 无配置 → 默认 50/300s
	cfg := loadConnectionPoolConfig(gormDB)
	assert.Equal(t, 50, cfg.MaxConnections)
	assert.Equal(t, 300*time.Second, cfg.MaxIdle)

	// 合法配置覆盖
	require.NoError(t, gormDB.Exec(`INSERT INTO sys_config (id, config_key, config_value) VALUES
		('k1', 'network.connection_pool.max_connections', '20'),
		('k2', 'network.connection_pool.max_idle_seconds', '60')`).Error)
	cfg = loadConnectionPoolConfig(gormDB)
	assert.Equal(t, 20, cfg.MaxConnections)
	assert.Equal(t, 60*time.Second, cfg.MaxIdle)

	// 非法/非正 → 回退默认
	require.NoError(t, gormDB.Exec(`UPDATE sys_config SET config_value = 'garbage' WHERE config_key = 'network.connection_pool.max_connections'`).Error)
	require.NoError(t, gormDB.Exec(`UPDATE sys_config SET config_value = '-1' WHERE config_key = 'network.connection_pool.max_idle_seconds'`).Error)
	cfg = loadConnectionPoolConfig(gormDB)
	assert.Equal(t, 50, cfg.MaxConnections)
	assert.Equal(t, 300*time.Second, cfg.MaxIdle)
}

func TestParseDuration(t *testing.T) {
	assert.Equal(t, 30*time.Second, parseDuration("", 30*time.Second))
	assert.Equal(t, 90*time.Second, parseDuration("90s", 30*time.Second))
	assert.Equal(t, 2*time.Minute, parseDuration("2m", 30*time.Second))
	assert.Equal(t, 30*time.Second, parseDuration("not-a-duration", 30*time.Second), "非法值回退默认")
}

func TestInitSM4Cipher(t *testing.T) {
	// 空 key → 错误(fail-fast)
	_, err := initSM4Cipher("")
	assert.ErrorContains(t, err, "SM4_KEY 未配置")

	// 仓库默认 key → 警告但可用
	cipher, err := initSM4Cipher("dGVzdC1zZWNyZXQxNiEhIQ==")
	require.NoError(t, err)
	require.NotNil(t, cipher)

	// 非法 base64 长度 → 错误
	_, err = initSM4Cipher("too-short")
	assert.ErrorContains(t, err, "创建 SM4 cipher 失败")

	// 合法 key → 成功
	cipher2, err := initSM4Cipher("MDEyMzQ1Njc4OWFiY2RlZg==") // base64(16 bytes)
	require.NoError(t, err)
	require.NotNil(t, cipher2)

	// 加解密 roundtrip
	enc, err := cipher2.Encrypt("secret")
	require.NoError(t, err)
	dec, err := cipher2.Decrypt(enc)
	require.NoError(t, err)
	assert.Equal(t, "secret", dec)
}

// ---------------- MetricsCacheService ----------------

func TestMetricsCacheService(t *testing.T) {
	c := newTestCoreForSplitCompat(t)
	svc := NewMetricsCacheService(c)
	defer svc.Stop()

	// 无缓存(core.Cache nil)→ manager 内部直采
	ctx := context.Background()
	// CPU 采样稳定性:gopsutil 差值采样在空闲机器(CI Linux runner / 空闲
	// Windows)两次采样差可为 0 → 业务码报"CPU时间差值计算为0"。空转 burn
	// 制造非零差,让 getRealtimeMetrics 成功路径成为**确定性**覆盖(曾因
	// 环境施舍导致 core 包 288/754 跌破 P2 floor 38.33)。
	burnCPU := func(d time.Duration) {
		deadline := time.Now().Add(d)
		for time.Now().Before(deadline) {
		}
	}
	burnCPU(200 * time.Millisecond)
	_, _ = svc.GetCurrentMetrics(ctx) // 预热:建立采样基线+写 L1(覆盖 realtime 分支)
	burnCPU(200 * time.Millisecond)
	metrics, err := svc.GetCurrentMetrics(ctx)
	require.NoError(t, err)
	require.NotNil(t, metrics)

	info, err := svc.GetServerInfo(ctx)
	require.NoError(t, err)
	require.NotNil(t, info)

	// Stop 双重调用会 close 已关闭 channel(QUIRK D-12);defer 一次即可
}

// ---------------- CaptchaBackgroundService ----------------

func newBgService(t *testing.T) (*CaptchaBackgroundService, *gorm.DB, *cache.MemoryCache) {
	t.Helper()
	gormDB, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "bg.db")), &gorm.Config{})
	require.NoError(t, err)
	if sqlDB, err := gormDB.DB(); err == nil {
		t.Cleanup(func() { _ = sqlDB.Close() })
	}
	require.NoError(t, gormDB.Migrator().CreateTable(&models.CaptchaBackground{}))
	mem := cache.NewMemoryCache(1000, 5*time.Minute)
	t.Cleanup(func() { _ = mem.Close() })
	dbWrapper := &coredb.Database{DB: gormDB, Type: "sqlite"}
	return NewCaptchaBackgroundService(dbWrapper, mem), gormDB, mem
}

func TestCaptchaBackgroundService_ValidateFile(t *testing.T) {
	svc, _, _ := newBgService(t)

	// 合法
	assert.NoError(t, svc.validateFile("bg.png", 1024))
	assert.NoError(t, svc.validateFile("bg.jpg", 1024))

	// 超大小
	assert.ErrorContains(t, svc.validateFile("bg.png", 3*1024*1024), "超过限制")
	// 不支持格式
	assert.ErrorContains(t, svc.validateFile("bg.gif", 100), "不支持的文件格式")
	// 无扩展名(Q-5):不得 panic,应返回不支持的文件格式
	assert.ErrorContains(t, svc.validateFile("noext", 100), "不支持的文件格式")
	// 扩展名为单个点(Q-5):filepath.Ext 返回 "."
	assert.ErrorContains(t, svc.validateFile("bg.", 100), "不支持的文件格式")
}

func TestCaptchaBackgroundService_CalculateMD5(t *testing.T) {
	svc, _, _ := newBgService(t)
	md5 := svc.calculateMD5([]byte("hello"))
	assert.Len(t, md5, 32)
	assert.Equal(t, "5d41402abc4b2a76b9719d911017c592", md5)
}

func TestCaptchaBackgroundService_GetImageDimensions(t *testing.T) {
	svc, _, _ := newBgService(t)

	// 不存在文件 → 错误
	_, _, err := svc.getImageDimensions(filepath.Join(t.TempDir(), "no.png"))
	assert.Error(t, err)

	// 非图片内容 → DecodeConfig 错误
	p := filepath.Join(t.TempDir(), "fake.png")
	require.NoError(t, os.WriteFile(p, []byte("not an image"), 0o644))
	_, _, err = svc.getImageDimensions(p)
	assert.Error(t, err)
}

func TestCaptchaBackgroundService_GetDB(t *testing.T) {
	svc, _, _ := newBgService(t)
	assert.NotNil(t, svc.GetDB())
}

func TestCaptchaBackgroundService_Upload_ValidationFail(t *testing.T) {
	svc, _, _ := newBgService(t)

	// 格式不合法 → 验证失败(不落盘)
	_, err := svc.Upload(context.Background(), &UploadRequest{
		FileName: "evil.gif", FileData: []byte("x"), FileSize: 1,
	})
	assert.ErrorContains(t, err, "文件验证失败")

	// 合法格式但非图片内容 → 尺寸解析失败
	svc.config.StoragePath = filepath.Join(t.TempDir(), "uploads")
	_, err = svc.Upload(context.Background(), &UploadRequest{
		FileName: "fake.png", FileData: []byte("not-image"), FileSize: 9,
	})
	assert.ErrorContains(t, err, "获取图片尺寸失败")
}

func TestCaptchaBackgroundService_GetRandomEnabled(t *testing.T) {
	ctx := context.Background()
	svc, db, mem := newBgService(t)

	// sqlite 空库时精确匹配 0 行,PG-only fallback 被方言 guard 跳过,
	// 直接走"没有找到可用的背景图"业务分支(Q-6)
	svcEmpty, _, _ := newBgService(t)
	_, err := svcEmpty.GetRandomEnabled(ctx, models.PieceShapeCircle, 1)
	assert.ErrorContains(t, err, "没有找到可用的背景图")
	assert.NotContains(t, err.Error(), "unrecognized token")
	assert.NotContains(t, err.Error(), "syntax error")

	// 精确命中正路径(确保 guard 不影响主查询)
	require.NoError(t, db.Exec(`INSERT INTO sys_captcha_background
		(id, file_name, file_path, piece_shape, difficulty_level, status, use_count, allowed_shapes)
		VALUES ('bg-1', 'a.png', 'uploads/a.png', 'circle', 1, 1, 0, '["circle"]')`).Error)

	bg, err := svc.GetRandomEnabled(ctx, models.PieceShapeCircle, 1)
	require.NoError(t, err)
	require.NotNil(t, bg)
	assert.Equal(t, "bg-1", bg.ID)

	// 缓存命中路径(第二次查询直接返回缓存列表)
	bg2, err := svc.GetRandomEnabled(ctx, models.PieceShapeCircle, 1)
	require.NoError(t, err)
	assert.Equal(t, "bg-1", bg2.ID)

	// IncrementUseCount
	require.NoError(t, svc.IncrementUseCount("bg-1"))
	var useCount int
	require.NoError(t, db.Raw(`SELECT use_count FROM sys_captcha_background WHERE id = 'bg-1'`).Scan(&useCount).Error)
	assert.Equal(t, 1, useCount)

	_ = mem
}

func TestCaptchaBackgroundService_CachePool(t *testing.T) {
	ctx := context.Background()
	svc, _, mem := newBgService(t)

	// 空池 → cache pool is empty
	_, err := svc.GetFromCachePool(ctx, "circle", 1)
	assert.ErrorContains(t, err, "cache pool is empty")

	// 计数器 0 → empty
	require.NoError(t, mem.Set(ctx, "captcha:cache:pool:circle:1:counter", "0", time.Minute))
	_, err = svc.GetFromCachePool(ctx, "circle", 1)
	assert.ErrorContains(t, err, "cache pool is empty")

	// 种入计数器与缓存项
	require.NoError(t, mem.Set(ctx, "captcha:cache:pool:circle:1:counter", "1", time.Minute))
	require.NoError(t, mem.Set(ctx, "captcha:cache:pool:circle:1:1", `{"token":"tk","xPos":10}`, time.Minute))
	item, err := svc.GetFromCachePool(ctx, "circle", 1)
	require.NoError(t, err)
	assert.Equal(t, "tk", item["token"])
	assert.Equal(t, float64(10), item["xPos"])

	// 取走最后一项后计数器归零 → 删除
	_, err = mem.Get(ctx, "captcha:cache:pool:circle:1:counter")
	assert.Error(t, err, "计数器被删除")
}

func TestCaptchaBackgroundService_PreGeneratePool_Empty(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := newBgService(t)
	// 空库 → 全部配置 0 背景 → 直接返回 nil(静默)
	require.NoError(t, svc.PreGeneratePool(ctx))
}
