package system

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// brandingCacheKey 公开品牌信息缓存键（CacheProvider 自动加 xingran: 前缀）
const brandingCacheKey = "system:branding:public"

// brandingCacheTTL 公开品牌信息缓存 TTL
const brandingCacheTTL = 5 * time.Minute

// BrandingConfig 站点品牌配置（brand.* 命名空间的聚合视图）
//
// 前端消费方：登录页 / 侧边栏 / document.title；管理端：系统设置-站点品牌 tab。
// 默认值仅是 seed 值的镜像（migration_259），DB 读取失败时兜底用，不是品牌事实源。
type BrandingConfig struct {
	SiteName  string `json:"siteName"`
	SiteMark  string `json:"siteMark"`
	LogoURL   string `json:"logoUrl"`
	Tagline   string `json:"tagline"`
	Copyright string `json:"copyright"`
	Subtitle  string `json:"subtitle"`
}

// brandingKeys brand.* → 字段 setter 的映射（单一来源，GetAll/Update 共用）
var brandingKeys = map[string]func(value string, cfg *BrandingConfig){
	"brand.site_name":     func(v string, c *BrandingConfig) { c.SiteName = v },
	"brand.site_mark":     func(v string, c *BrandingConfig) { c.SiteMark = v },
	"brand.logo_url":      func(v string, c *BrandingConfig) { c.LogoURL = v },
	"brand.login_tagline": func(v string, c *BrandingConfig) { c.Tagline = v },
	"brand.copyright":     func(v string, c *BrandingConfig) { c.Copyright = v },
	"brand.site_subtitle": func(v string, c *BrandingConfig) { c.Subtitle = v },
}

// BrandingService 站点品牌配置服务
type BrandingService interface {
	// GetAll 读取全部品牌配置（管理端；DB 缺失键回退默认值）
	GetAll(ctx context.Context) (*BrandingConfig, error)
	// Update 批量更新品牌配置（只更新传入的非空键），成功后失效公开缓存
	Update(ctx context.Context, values map[string]string) error
	// GetPublic 读取公开品牌信息（登录页用；CacheProvider 缓存 5 分钟）
	GetPublic(ctx context.Context) (*BrandingConfig, error)
	// SetLogoURL 写入 logo 访问 URL 并失效公开缓存
	SetLogoURL(ctx context.Context, url string) error
}

// brandingServiceImpl BrandingService 实现
type brandingServiceImpl struct {
	db    *gorm.DB
	cache CacheProvider
}

// NewBrandingService 创建站点品牌配置服务实例
func NewBrandingService(db *gorm.DB, cache CacheProvider) BrandingService {
	return &brandingServiceImpl{db: db, cache: cache}
}

// defaultBrandingConfig DB 缺失键时的兜底值（与 migration_259 seed 保持一致）
func defaultBrandingConfig() *BrandingConfig {
	return &BrandingConfig{
		SiteName:  "数字化平台",
		SiteMark:  "数",
		LogoURL:   "",
		Tagline:   "以国密算法为基座,融合自动化与 AI 调度,让每一次资源调度都可观测、可审计、可信赖。",
		Copyright: "© 2026 数字化平台",
		Subtitle:  "请使用您的账号登录以访问运维控制台",
	}
}

// GetAll 读取全部品牌配置
func (s *brandingServiceImpl) GetAll(ctx context.Context) (*BrandingConfig, error) {
	cfg := defaultBrandingConfig()
	if err := s.loadInto(ctx, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// loadInto 从 sys_config 读取 brand.* 前缀行并按映射填入 cfg
func (s *brandingServiceImpl) loadInto(ctx context.Context, cfg *BrandingConfig) error {
	var rows []models.Config
	if err := s.db.WithContext(ctx).
		Where("config_key LIKE ?", "brand.%").
		Find(&rows).Error; err != nil {
		return fmt.Errorf("读取品牌配置失败: %w", err)
	}
	for _, row := range rows {
		if setter, ok := brandingKeys[row.ConfigKey]; ok {
			setter(row.ConfigValue, cfg)
		}
	}
	return nil
}

// Update 批量更新品牌配置（map: config_key → 新值；空值也写入——支持清空 logo）
func (s *brandingServiceImpl) Update(ctx context.Context, values map[string]string) error {
	for key, value := range values {
		if _, ok := brandingKeys[key]; !ok {
			continue // 忽略非 brand.* 键，防止越权写其它配置
		}
		var count int64
		if err := s.db.WithContext(ctx).Model(&models.Config{}).
			Where("config_key = ?", key).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			if err := s.db.WithContext(ctx).Create(&models.Config{
				ConfigName:  "品牌-" + strings.TrimPrefix(key, "brand."),
				ConfigKey:   key,
				ConfigValue: value,
				ConfigType:  models.ConfigTypeNo,
				IsSystem:    models.ConfigIsSystemYes,
			}).Error; err != nil {
				return err
			}
			continue
		}
		if err := s.db.WithContext(ctx).Model(&models.Config{}).
			Where("config_key = ?", key).
			Update("config_value", value).Error; err != nil {
			return err
		}
	}
	s.invalidateCache(ctx)
	return nil
}

// GetPublic 公开品牌信息（缓存 5 分钟；缓存查询失败降级直读 DB）
//
// dest 必须传 &cfg（*BrandingConfig）、query 必须返回 BrandingConfig **值**：
// CacheAdapter / NoOpCacheProvider 都靠 base.SetValue(反射) 回填，它要求
// reflect.ValueOf(result).Type() 能 AssignableTo 到 Elem(dest) 的类型。
// 若 query 返回 *BrandingConfig 而 dest 是 *BrandingConfig，类型不匹配 → 静默
// 不回填，调用方读到零值（表现为"永远返回默认站名"）。此处的值类型约定
// 有 TestBrandingService_GetPublic_* 与 handler 契约用例双向锁住。
func (s *brandingServiceImpl) GetPublic(ctx context.Context) (*BrandingConfig, error) {
	var cfg BrandingConfig
	err := s.cache.GetOrSet(ctx, brandingCacheKey, &cfg, brandingCacheTTL, func() (interface{}, error) {
		fresh := defaultBrandingConfig()
		if loadErr := s.loadInto(ctx, fresh); loadErr != nil {
			return nil, loadErr
		}
		return *fresh, nil // 值类型（不可改成 &fresh，见上方注释）
	})
	if err != nil {
		// 缓存故障不阻断登录页：降级直读 DB
		applogger.Warnf("[Branding] 缓存读取失败,降级直读 DB: %v", err)
		fallback := defaultBrandingConfig()
		if loadErr := s.loadInto(ctx, fallback); loadErr != nil {
			return defaultBrandingConfig(), nil // DB 也失败时返回 seed 默认值，登录页永不空白
		}
		return fallback, nil
	}
	// 缓存命中时 cfg 已被反序列化填充；未命中时 query 已回填。
	// 兜底：DB 里 site_name 被清空（空串）时补默认值，避免标题变空。
	if cfg.SiteName == "" {
		cfg.SiteName = defaultBrandingConfig().SiteName
	}
	return &cfg, nil
}

// SetLogoURL 写入 logo URL
func (s *brandingServiceImpl) SetLogoURL(ctx context.Context, url string) error {
	return s.Update(ctx, map[string]string{"brand.logo_url": url})
}

// invalidateCache 失效公开品牌缓存（写操作后调用）
func (s *brandingServiceImpl) invalidateCache(ctx context.Context) {
	if err := s.cache.Delete(ctx, brandingCacheKey); err != nil {
		applogger.Warnf("[Branding] 失效品牌缓存失败(不影响写入结果): %v", err)
	}
}
