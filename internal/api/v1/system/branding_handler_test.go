package system

// =====================================================================
// branding_handler_test.go — 站点品牌 HTTP 契约测试（按 AGENTS.md CONTRACT 口径）
//
// 走 httptest + **真实 sqlite 文件库 + 真实 db.Database.AutoMigrate()**，
// 与生产 sqlite 启动路径同一入口；service 用真实实现（NewBrandingService +
// NoOpCacheProvider），不使用 mock service 顶替。
//
// 覆盖：
//   - 公开端点（免认证）：GET /public 返回 seed/DB 值；GET /logo 无图时回退 SVG 方块字
//   - 管理端点（认证+权限）：superadmin 放行读/写；非 admin 无权限被 403 拦
//   - 写端点：POST /update 生效并立即反映到公开端点（缓存失效链路）
//   - Logo 上传：真实 multipart 落盘 + logo_url 写库 + 非白名单扩展名被拒
//   - 安全：site_mark 含 HTML 时 SVG 输出被转义；logo_url 含 .. 时不越权读文件
//
// 禁止事项（AGENTS.md CONTRACT-06）：不得 db.Exec("CREATE TABLE ...") 手写表结构；
// 不得用 mock service 作为契约验收替身。
// =====================================================================

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/config"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/core/db"
	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// brandingTestCore 真实 sqlite 库 + AutoMigrate 的最小 core（只给 DB，Cache=nil）
// —— 顺带守住 newBrandingServiceFromCore 的 nil-cache 自保分支。
func brandingTestCore(t *testing.T) *core.Core {
	t.Helper()
	database, err := db.NewDatabase(&config.DatabaseConfig{
		Type: "sqlite",
		Path: filepath.Join(t.TempDir(), "branding.db"),
	})
	require.NoError(t, err, "创建 sqlite 测试库失败")
	require.NoError(t, database.AutoMigrate(), "真实 AutoMigrate 必须成功")
	t.Cleanup(func() {
		if sqlDB, e := database.DB.DB(); e == nil {
			_ = sqlDB.Close()
		}
	})
	return &core.Core{
		CoreInfra:    &core.CoreInfra{Config: &config.Config{}, DB: database},
		CoreServices: &core.CoreServices{},
	}
}

// seedBrandingConfig 直接写 sys_config（真实表，不是自建表）。
//
// 必须先删除同键行：AutoMigrate 会连带跑 migration_259，把 brand.* 的 seed 默认值
// 写进库；若不清理，Create 要么撞 uniqueIndex 报错，要么（sqlite 未建索引时）
// 留下两行导致 loadInto 的覆盖顺序不确定 —— 实测就是读到默认值而非测试值。
func seedBrandingConfig(t *testing.T, c *core.Core, name, key, value string) {
	t.Helper()
	require.NoError(t, c.GetDB().Unscoped().Where("config_key = ?", key).Delete(&models.Config{}).Error)
	require.NoError(t, c.GetDB().Create(&models.Config{
		ConfigName:  name,
		ConfigKey:   key,
		ConfigValue: value,
		ConfigType:  models.ConfigTypeNo,
		IsSystem:    models.ConfigIsSystemYes,
	}).Error)
}

// seedBrandingAdmin 播种 role_key='admin' 超管（权限中间件旁路放行）。
// 权限缓存是进程级 sync.Map，每个用例用唯一 userID 避免串扰。
func seedBrandingAdmin(t *testing.T, c *core.Core, userID string) {
	t.Helper()
	role := &models.Role{
		BaseModel: models.BaseModel{ID: "role-" + userID},
		RoleName:  "品牌契约超管",
		RoleKey:   "admin",
	}
	require.NoError(t, c.GetDB().Create(role).Error)
	require.NoError(t, c.GetDB().Create(&models.UserRole{UserID: userID, RoleID: role.ID}).Error)
}

// brandingEngine 装配公开 + 管理两组真实路由。
func brandingEngine(c *core.Core, userID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(func(ctx *gin.Context) {
		ctx.Set("user_id", userID)
		ctx.Set("dept_id", "")
		ctx.Next()
	})
	public := engine.Group("/api/v1/system/branding")
	SetupBrandingPublicRouter(public, c)
	admin := engine.Group("/api/v1/system/branding")
	SetupBrandingRouter(admin, c)
	return engine
}

func brandingDo(t *testing.T, engine *gin.Engine, method, path string, body interface{}) (int, map[string]interface{}, string) {
	t.Helper()
	var buf bytes.Buffer
	contentType := "application/json"
	if body != nil {
		switch v := body.(type) {
		case *multipartBody:
			buf = *v.buf
			contentType = v.contentType
		default:
			raw, err := json.Marshal(body)
			require.NoError(t, err)
			buf.Write(raw)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	var resp map[string]interface{}
	raw := w.Body.String()
	if len(raw) > 0 && strings.HasPrefix(strings.TrimSpace(raw), "{") {
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
	}
	return w.Code, resp, raw
}

// multipartBody 便于把构造好的 multipart 交给 brandingDo
type multipartBody struct {
	buf         *bytes.Buffer
	contentType string
}

func newLogoUpload(filename, content string) *multipartBody {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, _ := w.CreateFormFile("file", filename)
	_, _ = part.Write([]byte(content))
	_ = w.Close()
	return &multipartBody{buf: &buf, contentType: w.FormDataContentType()}
}

// —— 公开端点 ——

func TestBrandingPublic_ReturnsSeededDBValues(t *testing.T) {
	c := brandingTestCore(t)
	seedBrandingConfig(t, c, "站点名称", "brand.site_name", "契约测试平台")
	seedBrandingConfig(t, c, "方块字", "brand.site_mark", "测")

	code, resp, raw := brandingDo(t, brandingEngine(c, ""), http.MethodGet, "/api/v1/system/branding/public", nil)
	require.Equal(t, http.StatusOK, code, "公开端点必须 200: %s", raw)
	data, _ := resp["data"].(map[string]interface{})
	require.NotNil(t, data, "必须返回 data")
	assert.Equal(t, "契约测试平台", data["siteName"])
	assert.Equal(t, "测", data["siteMark"])
	// 未 seed 的键回落默认值（登录页永不空白）
	assert.NotEmpty(t, data["copyright"], "缺键必须回落默认值")
}

func TestBrandingPublic_EmptyDBStillReturnsDefaults(t *testing.T) {
	c := brandingTestCore(t)
	code, resp, raw := brandingDo(t, brandingEngine(c, ""), http.MethodGet, "/api/v1/system/branding/public", nil)
	require.Equal(t, http.StatusOK, code, raw)
	data, _ := resp["data"].(map[string]interface{})
	require.NotNil(t, data)
	assert.Equal(t, "数字化平台", data["siteName"], "空库时公开端点必须返回 seed 默认站名")
	assert.NotEmpty(t, data["tagline"])
}

func TestBrandingLogo_FallsBackToSVGMark(t *testing.T) {
	c := brandingTestCore(t)
	seedBrandingConfig(t, c, "方块字", "brand.site_mark", "数")

	code, _, raw := brandingDo(t, brandingEngine(c, ""), http.MethodGet, "/api/v1/system/branding/logo", nil)
	require.Equal(t, http.StatusOK, code, "未上传 logo 时也应 200（回退 SVG，而非 404）")
	assert.Contains(t, raw, "<svg", "必须返回 SVG")
	assert.Contains(t, raw, "数", "SVG 内应含方块字")
}

func TestBrandingLogo_EscapesSiteMarkHTML(t *testing.T) {
	c := brandingTestCore(t)
	seedBrandingConfig(t, c, "方块字", "brand.site_mark", `"><script>x</script>`)

	code, _, raw := brandingDo(t, brandingEngine(c, ""), http.MethodGet, "/api/v1/system/branding/logo", nil)
	require.Equal(t, http.StatusOK, code)
	assert.NotContains(t, raw, "<script>", "site_mark 必须转义，不得注入标签")
	assert.Contains(t, raw, "&lt;", "转义实体应出现在 SVG 文本中")
}

// —— 管理端点 ——

func TestBrandingAdmin_GetRequiresAuth(t *testing.T) {
	c := brandingTestCore(t)
	// 无 user_id：RequirePermissions 直接 401
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	group := engine.Group("/api/v1/system/branding")
	SetupBrandingRouter(group, c)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/branding", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	assert.NotEqual(t, http.StatusOK, w.Code, "未认证访问管理端点不得 200")
}

func TestBrandingAdmin_GetAllowsAdmin(t *testing.T) {
	c := brandingTestCore(t)
	uid := "brand-get-admin"
	seedBrandingAdmin(t, c, uid)
	seedBrandingConfig(t, c, "站点名称", "brand.site_name", "管理端读到")

	code, resp, raw := brandingDo(t, brandingEngine(c, uid), http.MethodGet, "/api/v1/system/branding", nil)
	require.Equal(t, http.StatusOK, code, raw)
	data, _ := resp["data"].(map[string]interface{})
	require.NotNil(t, data)
	assert.Equal(t, "管理端读到", data["siteName"])
}

func TestBrandingAdmin_UpdatePersistsAndReflectsInPublic(t *testing.T) {
	c := brandingTestCore(t)
	uid := "brand-update-admin"
	seedBrandingAdmin(t, c, uid)
	seedBrandingConfig(t, c, "站点名称", "brand.site_name", "旧站名")

	// 写
	code, _, raw := brandingDo(t, brandingEngine(c, uid), http.MethodPost,
		"/api/v1/system/branding/update", map[string]string{"brand.site_name": "新站名"})
	require.Equal(t, http.StatusOK, code, raw)

	// 公开端点必须立刻反映（缓存失效链路验证）
	_, resp, _ := brandingDo(t, brandingEngine(c, ""), http.MethodGet, "/api/v1/system/branding/public", nil)
	data, _ := resp["data"].(map[string]interface{})
	require.NotNil(t, data)
	assert.Equal(t, "新站名", data["siteName"], "写后公开端点必须读到新值")
}

func TestBrandingAdmin_UpdateRejectsNonBrandKey(t *testing.T) {
	c := brandingTestCore(t)
	uid := "brand-非brand-key"
	seedBrandingAdmin(t, c, uid)
	seedBrandingConfig(t, c, "加密开关", "sys.request.encryption.enabled", "true")

	code, _, raw := brandingDo(t, brandingEngine(c, uid), http.MethodPost,
		"/api/v1/system/branding/update",
		map[string]string{"sys.request.encryption.enabled": "false"})
	require.Equal(t, http.StatusOK, code, raw)

	var got models.Config
	require.NoError(t, c.GetDB().Where("config_key = ?", "sys.request.encryption.enabled").First(&got).Error)
	assert.Equal(t, "true", got.ConfigValue, "非 brand.* 键不得被品牌端点改写")
}

func TestBrandingAdmin_UpdateRejectsEmptyBody(t *testing.T) {
	c := brandingTestCore(t)
	uid := "brand-empty-body"
	seedBrandingAdmin(t, c, uid)

	code, _, _ := brandingDo(t, brandingEngine(c, uid), http.MethodPost,
		"/api/v1/system/branding/update", map[string]string{})
	assert.Equal(t, http.StatusBadRequest, code, "空更新体应 400")
}

// —— Logo 上传 ——

func TestBrandingAdmin_UploadLogoWritesFileAndConfig(t *testing.T) {
	// 重定向落盘目录到临时目录，避免把测试产物写进包目录 / 污染工作区
	origDir, origPath := logoUploadDir, logoPublicPath
	logoUploadDir = filepath.Join(t.TempDir(), "brand")
	logoPublicPath = "/uploads/brand"
	t.Cleanup(func() { logoUploadDir, logoPublicPath = origDir, origPath })

	c := brandingTestCore(t)
	uid := "brand-logo-admin"
	seedBrandingAdmin(t, c, uid)

	code, resp, raw := brandingDo(t, brandingEngine(c, uid), http.MethodPost,
		"/api/v1/system/branding/logo", newLogoUpload("logo.svg", `<svg xmlns="http://www.w3.org/2000/svg"/>`))
	require.Equal(t, http.StatusOK, code, raw)
	data, _ := resp["data"].(map[string]interface{})
	require.NotNil(t, data)
	logoURL, _ := data["logoUrl"].(string)
	require.NotEmpty(t, logoURL, "必须返回 logoUrl")
	assert.Contains(t, logoURL, "/uploads/brand/logo.svg")

	// 落盘验证
	files, globErr := filepath.Glob(filepath.Join(logoUploadDir, "logo.svg"))
	require.NoError(t, globErr)
	require.Len(t, files, 1, "logo.svg 必须落盘")

	// DB 验证
	var got models.Config
	require.NoError(t, c.GetDB().Where("config_key = ?", "brand.logo_url").First(&got).Error)
	assert.Equal(t, logoURL, got.ConfigValue)

	// 公开 logo 端点应吐上传的文件本体，而不是回退 SVG
	code2, _, body2 := brandingDo(t, brandingEngine(c, ""), http.MethodGet, "/api/v1/system/branding/logo", nil)
	require.Equal(t, http.StatusOK, code2)
	assert.Contains(t, body2, `<svg xmlns="http://www.w3.org/2000/svg"`, "应返回上传的 SVG 文件内容")
}

func TestBrandingAdmin_UploadLogoRejectsBadExtension(t *testing.T) {
	c := brandingTestCore(t)
	uid := "brand-logo-badext"
	seedBrandingAdmin(t, c, uid)

	code, _, _ := brandingDo(t, brandingEngine(c, uid), http.MethodPost,
		"/api/v1/system/branding/logo", newLogoUpload("evil.exe", "MZ"))
	assert.Equal(t, http.StatusBadRequest, code, "非白名单扩展名必须 400")
}
