package system

import (
	"context"
	"errors"
	"testing"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
	"github.com/stretchr/testify/assert"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// stubConfigService 测试桩 — 仅实现 GetByKey,其它方法 panic
type stubConfigService struct {
	value string
	err   error
}

func (s *stubConfigService) GetByKey(ctx context.Context, configKey string) (*models.Config, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &models.Config{ConfigKey: configKey, ConfigValue: s.value}, nil
}

// 其它 ConfigService 方法在测试桩中未使用,实现以满足接口
func (s *stubConfigService) GetByID(ctx context.Context, id string) (*models.Config, error) {
	panic("not implemented")
}
func (s *stubConfigService) List(ctx context.Context, params requests.ConfigListParams) (*PageResult, error) {
	panic("not implemented")
}
func (s *stubConfigService) Create(ctx context.Context, req *requests.ConfigCreateRequest) error {
	panic("not implemented")
}
func (s *stubConfigService) Update(ctx context.Context, req *requests.ConfigUpdateRequest) error {
	panic("not implemented")
}
func (s *stubConfigService) Delete(ctx context.Context, id string) error {
	panic("not implemented")
}
func (s *stubConfigService) BatchDelete(ctx context.Context, ids []string) error {
	panic("not implemented")
}
func (s *stubConfigService) RefreshCache(ctx context.Context) error {
	panic("not implemented")
}
func (s *stubConfigService) Statistics(ctx context.Context) (*ConfigStatisticsResult, error) {
	panic("not implemented")
}

func newSettingsServiceForTest(t *testing.T, cfg ConfigService) *settingsService {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	assert.NoError(t, err)
	return &settingsService{db: db, configService: cfg}
}

func TestBuildDefaultPreferences_NoAdminConfig(t *testing.T) {
	// 管理员未配置 sys.theme.default → 应当回退到硬编码 light/minimal
	stub := &stubConfigService{err: errors.New("record not found")}
	svc := newSettingsServiceForTest(t, stub)

	prefs := svc.buildDefaultPreferences(context.Background())

	assert.Equal(t, "light", prefs.Theme)
	assert.Equal(t, "minimal", prefs.ThemeStyle)
	assert.Empty(t, prefs.CustomPrimaryColor)
	assert.Empty(t, prefs.CustomSidebarColor)
}

func TestBuildDefaultPreferences_AdminConfigured(t *testing.T) {
	// 管理员配置了 dark + luxury-quiet + 自定义颜色 → 应当合并到 prefs
	adminJSON := `{"mode":"dark","style":"luxury-quiet","customColors":{"primary":"#FF5733","sidebar":"#222831"}}`
	stub := &stubConfigService{value: adminJSON}
	svc := newSettingsServiceForTest(t, stub)

	prefs := svc.buildDefaultPreferences(context.Background())

	assert.Equal(t, "dark", prefs.Theme)
	assert.Equal(t, "luxury-quiet", prefs.ThemeStyle)
	assert.Equal(t, "#FF5733", prefs.CustomPrimaryColor)
	assert.Equal(t, "#222831", prefs.CustomSidebarColor)
	// 其它非主题字段保持默认值
	assert.Equal(t, "classic", prefs.LayoutType)
	assert.Equal(t, 280, prefs.SidebarWidth)
}

func TestBuildDefaultPreferences_PartialConfig(t *testing.T) {
	// 管理员只配置 mode 但没 customColors → 只覆盖 Theme,颜色字段保留空
	adminJSON := `{"mode":"dark","style":"minimal"}`
	stub := &stubConfigService{value: adminJSON}
	svc := newSettingsServiceForTest(t, stub)

	prefs := svc.buildDefaultPreferences(context.Background())

	assert.Equal(t, "dark", prefs.Theme)
	assert.Equal(t, "minimal", prefs.ThemeStyle)
	assert.Empty(t, prefs.CustomPrimaryColor)
	assert.Empty(t, prefs.CustomSidebarColor)
}

func TestBuildDefaultPreferences_InvalidJSON(t *testing.T) {
	// JSON 解析失败 → 应当回退硬编码值,不 panic
	stub := &stubConfigService{value: "not-json{{{"}
	svc := newSettingsServiceForTest(t, stub)

	prefs := svc.buildDefaultPreferences(context.Background())

	assert.Equal(t, "light", prefs.Theme)
	assert.Equal(t, "minimal", prefs.ThemeStyle)
}

func TestBuildDefaultPreferences_NilConfigService(t *testing.T) {
	// configService 为 nil(老调用路径兼容) → 应当回退硬编码值
	svc := newSettingsServiceForTest(t, nil)

	prefs := svc.buildDefaultPreferences(context.Background())

	assert.Equal(t, "light", prefs.Theme)
	assert.Equal(t, "minimal", prefs.ThemeStyle)
}