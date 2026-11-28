package system

// =====================================================================
// branding_service_test.go — covers branding_service.go
//
// 覆盖点：
//   - GetAll：DB 有值时读取、缺键时回退 defaultBrandingConfig、值含空串也正确
//   - Update：只写 brand.* 键（忽略其它键，防越权）、更新已有行、缺失键自动补插
//   - GetPublic：CacheProvider 缓存命中/未命中、缓存故障时降级直读 DB、DB 也失败时返回默认值
//   - SetLogoURL：写入 brand.logo_url 并失效缓存
//
// fixture：内存 SQLite + 自建 sys_config 表（与 config_service_test.go 同口径，
// 该测试不承担 handler 契约验收，故按 AGENTS.md「就近单测」允许手建表）。
// =====================================================================

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
)

// setupBrandingTestDB 内存 SQLite + sys_config 表
func setupBrandingTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_config (
			id TEXT PRIMARY KEY,
			config_name TEXT NOT NULL,
			config_key TEXT NOT NULL UNIQUE,
			config_value TEXT,
			config_type TEXT DEFAULT 'Y',
			is_system INTEGER DEFAULT 0,
			remark TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER DEFAULT 0
		)
	`).Error)
	return db
}

// brandCacheStub 可控的 CacheProvider 桩（验证缓存命中/故障降级）
type brandCacheStub struct {
	store   map[string][]byte
	failAll bool
	deletes []string
}

func newBrandCacheStub() *brandCacheStub {
	return &brandCacheStub{store: map[string][]byte{}}
}

func (c *brandCacheStub) GetOrSet(ctx context.Context, key string, dest interface{}, _ time.Duration, query func() (interface{}, error)) error {
	if c.failAll {
		return assert.AnError
	}
	if _, ok := c.store[key]; ok {
		// 命中：把桩内存值回填到 dest（模拟 CacheAdapter 的反序列化行为）
		if val, ok2 := dest.(*BrandingConfig); ok2 {
			*val = cachedBranding
			return nil
		}
	}
	res, err := query()
	if err != nil {
		return err
	}
	// GetPublic 的 query 返回 BrandingConfig **值**（见 service 内注释），
	// 这里同时兼容值/指针，避免桩与真实实现的类型约定漂移。
	switch v := res.(type) {
	case BrandingConfig:
		cachedBranding = v
	case *BrandingConfig:
		cachedBranding = *v
	default:
		return nil
	}
	c.store[key] = []byte("hit")
	return nil
}

func (c *brandCacheStub) Delete(ctx context.Context, key string) error {
	c.deletes = append(c.deletes, key)
	delete(c.store, key)
	return nil
}

func (c *brandCacheStub) DeleteByPattern(ctx context.Context, pattern string) error { return nil }
func (c *brandCacheStub) MGet(ctx context.Context, keys ...string) (map[string]string, error) {
	return nil, nil
}
func (c *brandCacheStub) MDelete(ctx context.Context, keys ...string) error { return nil }
func (c *brandCacheStub) Exists(ctx context.Context, key string) (bool, error) {
	_, ok := c.store[key]
	return ok, nil
}
func (c *brandCacheStub) SetTTL(ctx context.Context, key string, _ time.Duration) error { return nil }
func (c *brandCacheStub) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	return 0, nil
}
func (c *brandCacheStub) GetStats(ctx context.Context) (*base.CacheStats, error) {
	return &base.CacheStats{}, nil
}

// cachedBranding 桩内存值
var cachedBranding BrandingConfig

// 保证桩满足接口（编译期断言）
var _ base.CacheProvider = (*brandCacheStub)(nil)

// —— GetAll ——

func TestBrandingService_GetAll_ReadsDBValues(t *testing.T) {
	db := setupBrandingTestDB(t)
	require.NoError(t, db.Exec(`INSERT INTO sys_config (id, config_name, config_key, config_value, config_type, is_system, version)
		VALUES ('1','站点名称','brand.site_name','云运维平台','N',1,0)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_config (id, config_name, config_key, config_value, config_type, is_system, version)
		VALUES ('2','宣传语','brand.login_tagline','自研调度','N',1,0)`).Error)

	svc := NewBrandingService(db, newBrandCacheStub())
	cfg, err := svc.GetAll(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "云运维平台", cfg.SiteName)
	assert.Equal(t, "自研调度", cfg.Tagline)
	// 未 seed 的键回退默认值
	assert.Equal(t, defaultBrandingConfig().Copyright, cfg.Copyright)
	assert.Equal(t, defaultBrandingConfig().SiteMark, cfg.SiteMark)
}

func TestBrandingService_GetAll_EmptyDBFallsBackToDefaults(t *testing.T) {
	db := setupBrandingTestDB(t)
	svc := NewBrandingService(db, newBrandCacheStub())
	cfg, err := svc.GetAll(context.Background())
	require.NoError(t, err)
	assert.Equal(t, defaultBrandingConfig(), cfg)
}

// —— Update ——

func TestBrandingService_Update_UpdatesExistingRow(t *testing.T) {
	db := setupBrandingTestDB(t)
	require.NoError(t, db.Exec(`INSERT INTO sys_config (id, config_name, config_key, config_value, config_type, is_system, version)
		VALUES ('1','站点名称','brand.site_name','旧名','N',1,0)`).Error)

	cache := newBrandCacheStub()
	svc := NewBrandingService(db, cache)
	require.NoError(t, svc.Update(context.Background(), map[string]string{"brand.site_name": "新名"}))

	var got string
	require.NoError(t, db.Raw(`SELECT config_value FROM sys_config WHERE config_key = 'brand.site_name'`).Scan(&got).Error)
	assert.Equal(t, "新名", got)
	// 写后必须失效公开缓存
	assert.Contains(t, cache.deletes, brandingCacheKey)
}

func TestBrandingService_Update_InsertsMissingKey(t *testing.T) {
	db := setupBrandingTestDB(t)
	svc := NewBrandingService(db, newBrandCacheStub())
	require.NoError(t, svc.Update(context.Background(), map[string]string{"brand.copyright": "© 2027"}))

	var rows []models.Config
	require.NoError(t, db.Where("config_key = ?", "brand.copyright").Find(&rows).Error)
	require.Len(t, rows, 1)
	assert.Equal(t, "© 2027", rows[0].ConfigValue)
	assert.Equal(t, models.ConfigIsSystemYes, rows[0].IsSystem)
}

func TestBrandingService_Update_IgnoresNonBrandKeys(t *testing.T) {
	db := setupBrandingTestDB(t)
	require.NoError(t, db.Exec(`INSERT INTO sys_config (id, config_name, config_key, config_value, config_type, is_system, version)
		VALUES ('1','加密开关','sys.request.encryption.enabled','true','Y',1,0)`).Error)

	svc := NewBrandingService(db, newBrandCacheStub())
	// 混入非 brand.* 键：不得被写入
	require.NoError(t, svc.Update(context.Background(), map[string]string{
		"brand.site_name":                "平台",
		"sys.request.encryption.enabled": "false",
	}))

	var got string
	require.NoError(t, db.Raw(`SELECT config_value FROM sys_config WHERE config_key = 'sys.request.encryption.enabled'`).Scan(&got).Error)
	assert.Equal(t, "true", got, "非 brand.* 键不得被 Update 改写")
}

func TestBrandingService_Update_AllowsClearingLogo(t *testing.T) {
	db := setupBrandingTestDB(t)
	require.NoError(t, db.Exec(`INSERT INTO sys_config (id, config_name, config_key, config_value, config_type, is_system, version)
		VALUES ('1','Logo','brand.logo_url','/uploads/brand/logo.png?v=1','N',1,0)`).Error)

	svc := NewBrandingService(db, newBrandCacheStub())
	require.NoError(t, svc.Update(context.Background(), map[string]string{"brand.logo_url": ""}))

	var got string
	require.NoError(t, db.Raw(`SELECT config_value FROM sys_config WHERE config_key = 'brand.logo_url'`).Scan(&got).Error)
	assert.Equal(t, "", got, "空值应能写入（清空 logo 是合法操作）")
}

// —— GetPublic（缓存）——

func TestBrandingService_GetPublic_CacheHitSkipsDB(t *testing.T) {
	db := setupBrandingTestDB(t)
	cache := newBrandCacheStub()
	svc := NewBrandingService(db, cache)

	// 首次：未命中 → 走 DB 并写缓存
	first, err := svc.GetPublic(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, first)

	// 之后 DB 里改名，但缓存仍在 → 应返回缓存值（证明走了缓存）
	require.NoError(t, db.Exec(`UPDATE sys_config SET config_value='DB新值' WHERE config_key='brand.site_name'`).Error)
	cachedBranding.SiteName = "缓存值"
	cachedBranding.Copyright = "缓存版权"
	second, err := svc.GetPublic(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "缓存值", second.SiteName, "缓存命中时不应回读 DB")
}

func TestBrandingService_GetPublic_CacheFailureFallsBackToDB(t *testing.T) {
	db := setupBrandingTestDB(t)
	require.NoError(t, db.Exec(`INSERT INTO sys_config (id, config_name, config_key, config_value, config_type, is_system, version)
		VALUES ('1','站点名称','brand.site_name','降级直读','N',1,0)`).Error)

	cache := newBrandCacheStub()
	cache.failAll = true
	svc := NewBrandingService(db, cache)

	cfg, err := svc.GetPublic(context.Background())
	require.NoError(t, err, "缓存故障不得让登录页失败")
	assert.Equal(t, "降级直读", cfg.SiteName)
}

// —— SetLogoURL ——

func TestBrandingService_SetLogoURL_WritesAndInvalidates(t *testing.T) {
	db := setupBrandingTestDB(t)
	cache := newBrandCacheStub()
	svc := NewBrandingService(db, cache)

	require.NoError(t, svc.SetLogoURL(context.Background(), "/uploads/brand/logo.png?v=42"))
	assert.Contains(t, cache.deletes, brandingCacheKey)

	cfg, err := svc.GetAll(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "/uploads/brand/logo.png?v=42", cfg.LogoURL)
}
