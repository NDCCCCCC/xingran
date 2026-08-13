// Package core 验证码服务
// 负责验证码的生成、验证、配置加载等核心业务逻辑
package core

import (
	"context"
	"encoding/base64"
	"fmt"
	"math/rand"
	"time"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"github.com/xingran-next/xingran-go-backend/internal/constants"
	"github.com/xingran-next/xingran-go-backend/internal/core/db"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/pkg/cache"
	"github.com/xingran-next/xingran-go-backend/pkg/captcha"
	"gorm.io/gorm"
)

// CaptchaService 验证码服务
type CaptchaService struct {
	db                *db.Database
	cache             cache.Cache
	config            *CaptchaConfig
	backgroundService *CaptchaBackgroundService // 背景图服务
}

// CaptchaConfig 验证码配置
type CaptchaConfig struct {
	Enabled        captcha.CaptchaType // 验证码开关: disabled, normal, slider
	Type           int                 // 数字验证码长度 4-6
	ExpireTime     int                 // 有效期(分钟)
	MaxAttempts    int                 // 最大验证次数
	IPRateLimit    int                 // IP限流(次/分钟)
	LoginMaxRetry  int                 // 登录最大重试次数
	LoginLockTime  int                 // 账号锁定时间(分钟)
	BackgroundMode string              // 背景图模式: auto=自动生成 custom=仅自定义 mixed=混合模式
	PieceShape     string              // 默认拼图形状: circle=圆形 square=方形 star=星形 heart=心形
	Difficulty     int                 // 默认难度级别: 1=简单 2=中等 3=困难
}

// SliderVerifyData 滑动验证码验证数据
type SliderVerifyData struct {
	XPos  int    `json:"xPos"`  // X坐标（缺口位置）
	YPos  int    `json:"yPos"`  // Y坐标（缺口位置）
	Token string `json:"token"` // 验证token
}

// NewCaptchaService 创建验证码服务
func NewCaptchaService(db *db.Database, cache cache.Cache) *CaptchaService {
	return &CaptchaService{
		db:    db,
		cache: cache,
		config: &CaptchaConfig{
			Enabled:        captcha.CaptchaTypeDisabled,
			Type:           4,
			ExpireTime:     5,
			MaxAttempts:    3,
			IPRateLimit:    10,
			LoginMaxRetry:  5,
			LoginLockTime:  30,
			BackgroundMode: "mixed",
			PieceShape:     "circle",
			Difficulty:     1,
		},
	}
}

// SetBackgroundService 设置背景图服务
func (s *CaptchaService) SetBackgroundService(bgService *CaptchaBackgroundService) {
	s.backgroundService = bgService
}

// GetDB 获取数据库连接
func (s *CaptchaService) GetDB() *gorm.DB {
	return s.db.GetDB()
}

// LoadConfig 从数据库加载配置
func (s *CaptchaService) LoadConfig(ctx context.Context) error {
	configs := []struct {
		key   string
		ptr   interface{}
		def   interface{}
		parse func(string, interface{}) error
	}{
		{
			key: "sys.account.captchaEnabled",
			ptr: &s.config.Enabled,
			def: captcha.CaptchaTypeDisabled,
			parse: func(val string, target interface{}) error {
				if p, ok := target.(*captcha.CaptchaType); ok {
					switch captcha.CaptchaType(val) {
					case captcha.CaptchaTypeDisabled, captcha.CaptchaTypeNormal, captcha.CaptchaTypeSlider:
						*p = captcha.CaptchaType(val)
					default:
						// 非法值兜底为 normal（要求验证码，偏安全）并告警，提示运维修正配置
						applogger.Warnf("[Captcha] 非法的验证码开关值 %q（合法值: disabled/normal/slider），已回退为 normal", val)
						*p = captcha.CaptchaTypeNormal
					}
					return nil
				}
				return fmt.Errorf("invalid target type")
			},
		},
		{
			key: "sys.account.captchaType",
			ptr: &s.config.Type,
			def: 4,
			parse: func(val string, target interface{}) error {
				if p, ok := target.(*int); ok {
					_, err := fmt.Sscanf(val, "%d", p)
					return err
				}
				return fmt.Errorf("invalid target type")
			},
		},
		{
			key: "sys.account.captchaExpireTime",
			ptr: &s.config.ExpireTime,
			def: 5,
			parse: func(val string, target interface{}) error {
				if p, ok := target.(*int); ok {
					_, err := fmt.Sscanf(val, "%d", p)
					return err
				}
				return fmt.Errorf("invalid target type")
			},
		},
		{
			key: "sys.account.captchaMaxAttempts",
			ptr: &s.config.MaxAttempts,
			def: 3,
			parse: func(val string, target interface{}) error {
				if p, ok := target.(*int); ok {
					_, err := fmt.Sscanf(val, "%d", p)
					return err
				}
				return fmt.Errorf("invalid target type")
			},
		},
		{
			key: "sys.account.ipRateLimit",
			ptr: &s.config.IPRateLimit,
			def: 10,
			parse: func(val string, target interface{}) error {
				if p, ok := target.(*int); ok {
					_, err := fmt.Sscanf(val, "%d", p)
					return err
				}
				return fmt.Errorf("invalid target type")
			},
		},
		{
			key: "sys.account.loginMaxRetry",
			ptr: &s.config.LoginMaxRetry,
			def: 5,
			parse: func(val string, target interface{}) error {
				if p, ok := target.(*int); ok {
					_, err := fmt.Sscanf(val, "%d", p)
					return err
				}
				return fmt.Errorf("invalid target type")
			},
		},
		{
			key: "sys.account.loginLockTime",
			ptr: &s.config.LoginLockTime,
			def: 30,
			parse: func(val string, target interface{}) error {
				if p, ok := target.(*int); ok {
					_, err := fmt.Sscanf(val, "%d", p)
					return err
				}
				return fmt.Errorf("invalid target type")
			},
		},
		{
			key: "sys.account.captchaBackgroundMode",
			ptr: &s.config.BackgroundMode,
			def: "mixed",
			parse: func(val string, target interface{}) error {
				if p, ok := target.(*string); ok {
					*p = val
					return nil
				}
				return fmt.Errorf("invalid target type")
			},
		},
		{
			key: "sys.account.captchaPieceShape",
			ptr: &s.config.PieceShape,
			def: "circle",
			parse: func(val string, target interface{}) error {
				if p, ok := target.(*string); ok {
					*p = val
					return nil
				}
				return fmt.Errorf("invalid target type")
			},
		},
		{
			key: "sys.account.captchaDifficulty",
			ptr: &s.config.Difficulty,
			def: 1,
			parse: func(val string, target interface{}) error {
				if p, ok := target.(*int); ok {
					_, err := fmt.Sscanf(val, "%d", p)
					return err
				}
				return fmt.Errorf("invalid target type")
			},
		},
	}

	for _, cfg := range configs {
		var configValue string
		err := s.db.GetDB().
			Table("sys_config").
			Where("config_key = ?", cfg.key).
			Pluck("config_value", &configValue).Error

		if err != nil || configValue == "" {
			// 使用默认值
			_ = cfg.parse(fmt.Sprintf("%v", cfg.def), cfg.ptr)
		} else {
			_ = cfg.parse(configValue, cfg.ptr)
		}
	}

	return nil
}

// GetConfig 获取当前配置
func (s *CaptchaService) GetConfig() *CaptchaConfig {
	return s.config
}

// GenerateCaptcha 生成验证码
func (s *CaptchaService) GenerateCaptcha(ctx context.Context, clientIP string) (*captcha.CaptchaResponse, error) {
	// 检查验证码是否启用
	if s.config.Enabled == captcha.CaptchaTypeDisabled {
		return nil, nil
	}

	// IP限流检查 - 从数据库读取限制次数
	ipRateLimit := s.getIPRateLimit(ctx)
	rateLimitKey := fmt.Sprintf("captcha:rate:%s", clientIP)

	var count int64
	var err error

	// 尝试使用原子操作（IncrementWithExpire）确保TTL正确设置
	if redisCache, ok := s.cache.(cache.RateLimitCache); ok {
		count, err = redisCache.IncrementWithExpire(ctx, rateLimitKey, 1*time.Minute)
		if err != nil {
			applogger.Errorf("[Captcha] Rate limit check failed for IP %s: %v", clientIP, err)
			// fail-close：限流基础设施不可用时拒绝，避免被绕过
			return nil, fmt.Errorf("服务繁忙，请稍后再试")
		}
	} else {
		// 降级到普通Increment + Expire（旧逻辑）
		count, err = s.cache.Increment(ctx, rateLimitKey)
		if err != nil {
			applogger.Errorf("[Captcha] Rate limit check failed for IP %s: %v", clientIP, err)
			// fail-close：限流基础设施不可用时拒绝，避免被绕过
			return nil, fmt.Errorf("服务繁忙，请稍后再试")
		}
		if count == 1 {
			_ = s.cache.Expire(ctx, rateLimitKey, 1*time.Minute)
		}
	}

	if count > int64(ipRateLimit) {
		return nil, fmt.Errorf("获取验证码过于频繁，请稍后再试")
	}

	// 生成验证码ID
	captchaID := fmt.Sprintf("%d", time.Now().UnixNano())

	// 计算过期时间
	expireDuration := time.Duration(s.config.ExpireTime) * time.Minute

	var response *captcha.CaptchaResponse

	switch s.config.Enabled {
	case captcha.CaptchaTypeNormal:
		// 数字字母验证码
		generator := captcha.NewTextCaptcha(s.config.Type)
		base64Img, code, err := generator.GenerateBase64()
		if err != nil {
			return nil, fmt.Errorf("生成验证码失败: %w", err)
		}

		// 存储验证码到缓存
		storageKey := fmt.Sprintf("captcha:data:%s", captchaID)
		err = s.cache.Set(ctx, storageKey, code, expireDuration)
		if err != nil {
			return nil, fmt.Errorf("存储验证码失败: %w", err)
		}

		// 存储验证次数
		attemptsKey := fmt.Sprintf("captcha:attempts:%s", captchaID)
		_ = s.cache.SetInt(ctx, attemptsKey, 0, expireDuration)

		response = &captcha.CaptchaResponse{
			CaptchaID:   captchaID,
			CaptchaType: captcha.CaptchaTypeNormal,
			CaptchaImg:  "data:image/png;base64," + base64Img,
		}

	case captcha.CaptchaTypeSlider:
		// 滑动拼图验证码 - 支持自定义背景图
		sliderData, err := s.generateSliderWithBackground(ctx)
		if err != nil {
			return nil, fmt.Errorf("生成滑动验证码失败: %w", err)
		}

		// 存储验证数据到缓存
		verifyData := SliderVerifyData{
			XPos:  sliderData["xPos"].(int),
			YPos:  sliderData["yPos"].(int),
			Token: sliderData["token"].(string),
		}
		storageKey := fmt.Sprintf("captcha:data:%s", captchaID)
		err = s.cache.SetJSON(ctx, storageKey, verifyData, expireDuration)
		if err != nil {
			return nil, fmt.Errorf("存储验证码失败: %w", err)
		}

		// 存储验证次数
		attemptsKey := fmt.Sprintf("captcha:attempts:%s", captchaID)
		_ = s.cache.SetInt(ctx, attemptsKey, 0, expireDuration)

		response = &captcha.CaptchaResponse{
			CaptchaID:   captchaID,
			CaptchaType: captcha.CaptchaTypeSlider,
			SliderImg:   sliderData["sliderImg"].(string),
			PieceImg:    sliderData["pieceImg"].(string),
			YPos:        sliderData["yPos"].(int),
			Token:       sliderData["token"].(string),
		}
	}

	return response, nil
}

// VerifyCaptcha 验证验证码
func (s *CaptchaService) VerifyCaptcha(ctx context.Context, captchaID, input string, clientIP string) error {
	if s.config.Enabled == captcha.CaptchaTypeDisabled {
		return nil
	}

	// 检查验证次数
	attemptsKey := fmt.Sprintf("captcha:attempts:%s", captchaID)
	attempts, _ := s.cache.GetInt(ctx, attemptsKey)

	if attempts >= s.config.MaxAttempts {
		// 超过最大验证次数，删除验证码
		storageKey := fmt.Sprintf("captcha:data:%s", captchaID)
		_ = s.cache.Delete(ctx, storageKey)
		_ = s.cache.Delete(ctx, attemptsKey)
		return fmt.Errorf("验证码已失效，请重新获取")
	}

	storageKey := fmt.Sprintf("captcha:data:%s", captchaID)

	switch s.config.Enabled {
	case captcha.CaptchaTypeNormal:
		// 验证数字字母验证码
		storedCode, err := s.cache.Get(ctx, storageKey)
		if err != nil {
			return fmt.Errorf("验证码不存在或已过期")
		}

		if storedCode != input {
			// 增加失败次数
			_, _ = s.cache.Increment(ctx, attemptsKey)
			return fmt.Errorf("验证码错误")
		}

	case captcha.CaptchaTypeSlider:
		// 验证滑动拼图验证码 - 检查"验证通过"标记
		verifiedKey := fmt.Sprintf(constants.CaptchaVerifiedKeyFormat, captchaID)
		verified, err := s.cache.Exists(ctx, verifiedKey)
		if err != nil || !verified {
			return fmt.Errorf("滑动验证码未通过验证或已过期，请重新验证")
		}

		// 验证输入是 "verified" 标记
		if input != "verified" {
			return fmt.Errorf("验证码无效")
		}

		// 验证成功，删除验证标记
		_ = s.cache.Delete(ctx, verifiedKey)
		return nil
	}

	// 验证成功，删除验证码（普通数字验证码）
	_ = s.cache.Delete(ctx, storageKey)
	_ = s.cache.Delete(ctx, attemptsKey)

	return nil
}

// VerifySliderCaptcha 验证滑动拼图验证码
func (s *CaptchaService) VerifySliderCaptcha(ctx context.Context, captchaID string, xPos int, token string) error {
	if s.config.Enabled != captcha.CaptchaTypeSlider {
		return fmt.Errorf("当前未启用滑动验证码")
	}

	// 检查验证次数
	attemptsKey := fmt.Sprintf("captcha:attempts:%s", captchaID)
	attempts, _ := s.cache.GetInt(ctx, attemptsKey)

	if attempts >= s.config.MaxAttempts {
		storageKey := fmt.Sprintf("captcha:data:%s", captchaID)
		_ = s.cache.Delete(ctx, storageKey)
		_ = s.cache.Delete(ctx, attemptsKey)
		return fmt.Errorf("验证码已失效，请重新获取")
	}

	// 获取存储的验证数据
	storageKey := fmt.Sprintf("captcha:data:%s", captchaID)
	var verifyData SliderVerifyData
	err := s.cache.GetJSON(ctx, storageKey, &verifyData)
	if err != nil {
		return fmt.Errorf("验证码不存在或已过期")
	}

	// 验证X坐标（允许一定误差）
	tolerance := 8               // 允许8像素误差
	expectedX := verifyData.XPos // 使用存储的正确 X 位置

	// 验证滑动位置是否在容差范围内
	if abs(xPos-expectedX) > tolerance {
		_, _ = s.cache.Increment(ctx, attemptsKey)
		return fmt.Errorf("验证失败，位置不正确")
	}

	// 验证token
	if token == "" || token != verifyData.Token {
		_, _ = s.cache.Increment(ctx, attemptsKey)
		return fmt.Errorf("验证失败，token无效")
	}

	// 验证成功，设置"验证通过"标记（用于后续登录验证）
	verifiedKey := fmt.Sprintf(constants.CaptchaVerifiedKeyFormat, captchaID)

	// CRITICAL: Write verified marker directly to Redis (bypass L2Writer async worker pool)
	// to ensure immediate availability for subsequent login verification
	// L2Writer queues writes asynchronously, causing race condition where
	// Exists check happens before Redis write completes
	if mlCache, ok := s.cache.(cache.L2ExposingCache); ok {
		// Bypass L2Writer - write directly to L2 (Redis) for immediate availability
		l2Cache := mlCache.GetL2Cache()
		err = l2Cache.Set(ctx, verifiedKey, "1", 5*time.Minute)
		if err != nil {
			applogger.Errorf("[Captcha] Failed to set verified marker directly to Redis: %v", err)
			return fmt.Errorf("设置验证标记失败")
		}
		applogger.Infof("[Captcha] Set verified marker directly to Redis: %s", verifiedKey)
	} else {
		// Fallback to normal cache set (may use L2Writer)
		err = s.cache.Set(ctx, verifiedKey, "1", 5*time.Minute)
		if err != nil {
			applogger.Errorf("[Captcha] Failed to set verified marker: %v", err)
		} else {
			applogger.Infof("[Captcha] Set verified marker via normal cache: %s", verifiedKey)
		}
	}

	// 删除原始验证码数据（防止重复验证）
	_ = s.cache.Delete(ctx, storageKey)
	_ = s.cache.Delete(ctx, attemptsKey)

	return nil
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// CheckLoginLock 检查账号是否被锁定
func (s *CaptchaService) CheckLoginLock(ctx context.Context, username string) error {
	lockKey := fmt.Sprintf(constants.LoginLockKeyFormat, username)
	locked, _ := s.cache.Exists(ctx, lockKey)
	if locked {
		ttl, _ := s.cache.TTL(ctx, lockKey)
		remainingMinutes := int(ttl.Minutes())
		return fmt.Errorf("账号已锁定，请 %d 分钟后再试", remainingMinutes)
	}
	return nil
}

// RecordLoginFailure 记录登录失败
func (s *CaptchaService) RecordLoginFailure(ctx context.Context, username string) error {
	failKey := fmt.Sprintf("login:fail:%s", username)
	count, err := s.cache.Increment(ctx, failKey)
	if err == nil && count == 1 {
		// 首次失败，设置过期时间为1小时
		_ = s.cache.Expire(ctx, failKey, 1*time.Hour)
	}

	if count >= int64(s.config.LoginMaxRetry) {
		// 锁定账号
		lockKey := fmt.Sprintf(constants.LoginLockKeyFormat, username)
		lockDuration := time.Duration(s.config.LoginLockTime) * time.Minute
		_ = s.cache.Set(ctx, lockKey, "1", lockDuration)
		_ = s.cache.Delete(ctx, failKey)
		return fmt.Errorf("登录失败次数过多，账号已被锁定 %d 分钟", s.config.LoginLockTime)
	}

	remaining := int64(s.config.LoginMaxRetry) - count
	if remaining > 0 && remaining <= int64(s.config.LoginMaxRetry/2) {
		return fmt.Errorf("登录失败，还可尝试 %d 次", remaining)
	}

	return fmt.Errorf("用户名或密码错误")
}

// ClearLoginFailure 清除登录失败记录（登录成功时调用）
func (s *CaptchaService) ClearLoginFailure(ctx context.Context, username string) {
	failKey := fmt.Sprintf("login:fail:%s", username)
	_ = s.cache.Delete(ctx, failKey)
}

// IsEnabled 检查验证码是否启用
func (s *CaptchaService) IsEnabled() bool {
	return s.config.Enabled != captcha.CaptchaTypeDisabled
}

// generateSliderWithBackground 生成带背景图的滑动验证码
// 根据 BackgroundMode 配置决定使用自定义背景还是自动生成背景
func (s *CaptchaService) generateSliderWithBackground(ctx context.Context) (map[string]interface{}, error) {
	shape := s.config.PieceShape
	difficulty := s.config.Difficulty

	// 调试日志：打印配置信息
	applogger.Debugf("[Captcha] BackgroundMode=%s, PieceShape=%s, Difficulty=%d",
		s.config.BackgroundMode, shape, difficulty)
	applogger.Debugf("[Captcha] backgroundService is nil: %v", s.backgroundService == nil)

	// 检查是否使用自定义背景图
	useCustom := false
	switch s.config.BackgroundMode {
	case "custom":
		useCustom = true
		applogger.Debugf("[Captcha] Using custom background mode")
	case "mixed":
		// 混合模式：50%概率使用自定义背景
		useCustom = rand.Intn(2) == 0
		applogger.Debugf("[Captcha] Using mixed background mode, useCustom=%v", useCustom)
	case "auto":
		useCustom = false
		applogger.Debugf("[Captcha] Using auto background mode")
	default:
		// 未知模式，使用自动生成
		useCustom = false
		applogger.Warnf("[Captcha] Unknown background mode: %s, falling back to auto", s.config.BackgroundMode)
	}

	// 尝试使用自定义背景图
	if useCustom && s.backgroundService != nil {
		applogger.Debugf("[Captcha] Attempting to use custom background")
		// 先尝试从缓存池获取
		cached, err := s.backgroundService.GetFromCachePool(ctx, shape, difficulty)
		if err == nil && cached != nil {
			applogger.Debugf("[Captcha] Cache pool hit, using cached background")
			return cached, nil
		}
		applogger.Debugf("[Captcha] Cache pool miss: %v", err)

		// 缓存未命中，从数据库获取随机背景图并生成
		bg, err := s.backgroundService.GetRandomEnabled(ctx, models.PieceShape(shape), difficulty)
		if err != nil {
			applogger.Warnf("[Captcha] GetRandomEnabled failed: %v", err)
		} else if bg == nil {
			applogger.Warnf("[Captcha] GetRandomEnabled returned nil background")
		} else {
			applogger.Debugf("[Captcha] Got background from DB: %s", bg.FileName)
			generator, _ := captcha.NewSliderCaptchaWithShape(shape)
			bgImg, err := generator.LoadBackgroundFromFile(bg.FilePath)
			if err != nil {
				applogger.Warnf("[Captcha] LoadBackgroundFromFile failed: %v", err)
			} else {
				applogger.Debugf("[Captcha] Background loaded successfully")
				bgBytes, pieceBytes, xPos, yPos, token, err := generator.GenerateSliderWithCustomBackground(bgImg, shape, difficulty)
				if err != nil {
					applogger.Warnf("[Captcha] GenerateSliderWithCustomBackground failed: %v", err)
				} else {
					applogger.Infof("[Captcha] Successfully generated captcha with custom background: %s", bg.FileName)
					// 更新使用次数
					_ = s.backgroundService.IncrementUseCount(bg.ID)
					return map[string]interface{}{
						"backgroundId": bg.ID,
						"sliderImg":    "data:image/png;base64," + base64.StdEncoding.EncodeToString(bgBytes),
						"pieceImg":     "data:image/png;base64," + base64.StdEncoding.EncodeToString(pieceBytes),
						"xPos":         xPos,
						"yPos":         yPos,
						"token":        token,
						"shape":        shape,
						"difficulty":   difficulty,
						"createdAt":    time.Now().Unix(),
					}, nil
				}
			}
		}
	} else {
		if !useCustom {
			applogger.Debugf("[Captcha] useCustom=false, skipping custom background")
		}
		if s.backgroundService == nil {
			applogger.Warnf("[Captcha] backgroundService is nil, cannot use custom backgrounds")
		}
	}

	// 使用自动生成的背景图（降级方案）
	applogger.Infof("[Captcha] Falling back to auto-generated background")
	generator, _ := captcha.NewSliderCaptchaWithShape(shape)
	bgBase64, pieceBase64, xPos, yPos, token, err := generator.GenerateSliderBase64()
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"sliderImg":  "data:image/png;base64," + bgBase64,
		"pieceImg":   "data:image/png;base64," + pieceBase64,
		"xPos":       xPos,
		"yPos":       yPos,
		"token":      token,
		"shape":      shape,
		"difficulty": difficulty,
		"createdAt":  time.Now().Unix(),
	}, nil
}

// getIPRateLimit 从数据库读取 IP 限流配置（次/分钟）
// 如果数据库中没有配置，则使用默认值 10
func (s *CaptchaService) getIPRateLimit(ctx context.Context) int {
	var config models.Config
	err := s.db.GetDB().Where("config_key = ?", "sys.captcha.ip_rate_limit").First(&config).Error
	if err != nil {
		// 配置不存在，返回默认值 10
		return 10
	}

	// 解析配置值
	var limit int
	if _, err := fmt.Sscanf(config.ConfigValue, "%d", &limit); err != nil || limit <= 0 {
		// 解析失败或值无效，返回默认值 10
		return 10
	}

	return limit
}
