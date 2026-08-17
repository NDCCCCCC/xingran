package middleware

import (
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/pkg/permission"
)

// --- D-21 sqlite in-memory 模式(沿用 Phase 59 D-03 + Phase 60 D-02) ---

// setupInheritPermsTestDB 沿用 setupUsageLoggerTestDB 模式(os.TempDir + 唯一
// 文件名 + busy_timeout=5000),自动迁移本测试要 INSERT 的所有表:
//
//   sys_api_keys        — APIKey 模型(D-10 关联 User 读取)
//   sys_user            — User 模型(D-10 username/nickname 来源)
//   sys_role            — Role 模型(关联 sys_menu)
//   sys_menu            — Menu 模型(perms 字段 = 权限代码)
//   sys_role_menu       — RoleMenu 模型(join sys_role ↔ sys_menu)
//   sys_user_role       — UserRole 模型(join sys_user ↔ sys_role)
//
// 不复用 setupUsageLoggerTestDB — 该函数只建 sys_api_key_usage_logs 单表,
// 本测试需要 6 张表的完整 RBAC schema。
func setupInheritPermsTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dbPath := filepath.Join(os.TempDir(), fmt.Sprintf("xingran_inherit_%d_%d.db", time.Now().UnixNano(), os.Getpid()))
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)", dbPath)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)

	// 使用 GORM AutoMigrate 完整 6 表(RBAC schema)。
	// DisableForeignKeyConstraintWhenMigrating: true 让 sqlite 不强约束 FK,
	// 测试可独立 INSERT 各表而无须严格顺序。
	err = db.AutoMigrate(
		&models.APIKey{},
		&models.User{},
		&models.Role{},
		&models.RoleMenu{},
		&models.Menu{},
		&models.UserRole{},
	)

	return db
}

// createTestUserWithPermissions INSERT 一个 User + 关联 Role/Menu/UserRole,
// Menu.perms 字段 = permCodes[i],User 通过 UserRole 关联到 Role,Role 通过
// RoleMenu 关联到 Menu 集合 — 与生产 permission.Service.GetUserPermissions
// 的 SQL JOIN 路径完全一致(service.go:274-283)。
//
// 返回: userID
func createTestUserWithPermissions(t *testing.T, db *gorm.DB, permCodes []string) string {
	t.Helper()

	userID := uuid.New().String()
	roleID := uuid.New().String()

	nickname := "测试用户"
	user := models.User{
		BaseModel: models.BaseModel{ID: userID},
		Username:  "testuser_" + userID[:8],
		Password:  "hashed_pwd",
		Salt:      "salt_value",
		Nickname:  &nickname,
		Status:    models.UserStatusEnabled,
	}
	require.NoError(t, db.Create(&user).Error)

	role := models.Role{
		BaseModel: models.BaseModel{ID: roleID},
		RoleName:  "test_role",
		RoleKey:   "test_role_" + roleID[:8],
		RoleSort:  1,
		Status:    models.RoleStatusEnabled,
	}
	require.NoError(t, db.Create(&role).Error)

	for _, code := range permCodes {
		menuID := uuid.New().String()
		menu := models.Menu{
			BaseModel: models.BaseModel{ID: menuID},
			MenuName:  "menu_" + code,
			MenuType:  models.MenuTypeButton,
			Visible:   models.VisibleShow,
			Status:    models.MenuStatusNormal,
			Perms:     &code,
		}
		require.NoError(t, db.Create(&menu).Error)

		roleMenu := models.RoleMenu{
			RoleID: roleID,
			MenuID: menuID,
		}
		require.NoError(t, db.Create(&roleMenu).Error)
	}

	userRole := models.UserRole{
		UserID: userID,
		RoleID: roleID,
	}
	require.NoError(t, db.Create(&userRole).Error)

	return userID
}

// createTestAPIKey INSERT 一个 APIKey(含 UserID 关联 + InheritPerms 标志),
// KeyHash/Salt/KeyPrefix 用合法值(D-08 验证用 — 不实际触发 hash 验证,
// 因为测试不走真实 API Key 头路径,而是 fakeAPIKeyService 直接返回构造对象)。
//
// 返回: apiKeyID
func createTestAPIKey(t *testing.T, db *gorm.DB, userID *string, scopes []string, inheritPerms bool) string {
	t.Helper()

	apiKeyID := uuid.New().String()
	apiKey := models.APIKey{
		BaseModel:    models.BaseModel{ID: apiKeyID},
		Name:         "test-key",
		KeyHash:      "fake-hash-" + apiKeyID[:8],
		Salt:         "fake-salt-" + apiKeyID[:8],
		KeyPrefix:    "rec_" + apiKeyID[:8],
		UserID:       userID,
		Scopes:       scopes,
		InheritPerms: inheritPerms,
		IsActive:     true,
	}
	require.NoError(t, db.Create(&apiKey).Error)
	return apiKeyID
}

// --- 集成测试 ---
//
// 本文件复用同 package 内 apikey_integration_test.go 已定义的 fakeAPIKeyService
// (满足 system.APIKeyService 全 9 方法,ValidateAPIKey 返回构造的 *models.APIKey)。
// 测试只需把 validKey 设置为带 User 关联的预置对象即可 — 模拟生产
// ValidateAPIKey 的 Preload("User") 行为(Phase 60 已落地)。

// TestMultiAuthInheritPerms_MergeScopes D-06 验证: InheritPerms=true 时,
// User 关联的 Menu.perms 通过 permission.Service.GetUserPermissions 加载,
// 与 API Key 自带 scopes 取并集写入 c.Set("scopes")。
func TestMultiAuthInheritPerms_MergeScopes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupInheritPermsTestDB(t)

	userID := createTestUserWithPermissions(t, db, []string{
		"system:user:list",
		"system:user:view",
	})
	apiKeyID := createTestAPIKey(t, db, &userID, []string{"read"}, true)

	// fakeAPIKeyService.validKey 必须含 User 关联(D-10),模拟生产 Preload 行为
	fakeSvc := &fakeAPIKeyService{
		validKey: &models.APIKey{
			BaseModel:    models.BaseModel{ID: apiKeyID},
			Name:         "test-key",
			KeyHash:      "fake-hash",
			Salt:         "fake-salt",
			KeyPrefix:    "rec_test",
			UserID:       &userID,
			Scopes:       []string{"read"},
			InheritPerms: true,
			IsActive:     true,
			User: &models.User{
				BaseModel: models.BaseModel{ID: userID},
				Username:  "testuser",
				Nickname:  stringPtr("测试用户"),
				Status:    models.UserStatusEnabled,
			},
		},
	}
	fakeLogger := newFakeUsageLogger()
	realPermSvc := permission.NewService()

	router := gin.New()
	router.Use(MultiAuth(fakeSvc, fakeLogger, realPermSvc, db))
	router.GET("/probe", func(c *gin.Context) {
		// 断言: handler 读到的 scopes 同时含 read + User 权限代码
		scopes := c.MustGet("scopes").([]string)
		assert.ElementsMatch(t,
			[]string{"read", "system:user:list", "system:user:view"},
			scopes,
			"InheritPerms=true 加载 User 权限与 scopes 取并集")
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/probe", nil)
	req.Header.Set("X-API-Key", "rec_"+strings.Repeat("a", 64))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code, "InheritPerms=true 加载成功应 200")
	fakeLogger.waitForLog(t)
}

// TestMultiAuthInheritPerms_FailClosed D-09 验证: UserID 为 nil + InheritPerms=true
// 视为配置错误,fail-closed 401。
func TestMultiAuthInheritPerms_FailClosed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupInheritPermsTestDB(t)

	// 创建 APIKey 不带 user_id + inherit_perms=true
	apiKeyID := createTestAPIKey(t, db, nil, []string{"read"}, true)

	fakeSvc := &fakeAPIKeyService{
		validKey: &models.APIKey{
			BaseModel:    models.BaseModel{ID: apiKeyID},
			Name:         "broken-key",
			KeyHash:      "fake-hash",
			Salt:         "fake-salt",
			KeyPrefix:    "rec_test",
			UserID:       nil, // D-09: UserID nil → 401
			Scopes:       []string{"read"},
			InheritPerms: true,
			IsActive:     true,
		},
	}
	fakeLogger := newFakeUsageLogger()
	realPermSvc := permission.NewService()

	router := gin.New()
	router.Use(MultiAuth(fakeSvc, fakeLogger, realPermSvc, db))
	router.GET("/probe", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/probe", nil)
	req.Header.Set("X-API-Key", "rec_"+strings.Repeat("a", 64))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, 401, w.Code, "InheritPerms=true 但 UserID 为 nil 应 401 (D-09 fail-closed)")
}

// TestMultiAuthInheritPerms_DBError D-09 验证: DB 关闭后 GetUserPermissions
// 返回 error → 401 + message 含 "用户权限加载失败"。
func TestMultiAuthInheritPerms_DBError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupInheritPermsTestDB(t)

	userID := createTestUserWithPermissions(t, db, []string{"system:user:list"})
	apiKeyID := createTestAPIKey(t, db, &userID, []string{"read"}, true)

	fakeSvc := &fakeAPIKeyService{
		validKey: &models.APIKey{
			BaseModel:    models.BaseModel{ID: apiKeyID},
			Name:         "db-error-key",
			KeyHash:      "fake-hash",
			Salt:         "fake-salt",
			KeyPrefix:    "rec_test",
			UserID:       &userID,
			Scopes:       []string{"read"},
			InheritPerms: true,
			IsActive:     true,
			User: &models.User{
				BaseModel: models.BaseModel{ID: userID},
				Username:  "testuser",
				Nickname:  stringPtr("测试"),
				Status:    models.UserStatusEnabled,
			},
		},
	}

	// 关闭 DB 模拟查询失败
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	fakeLogger := newFakeUsageLogger()
	realPermSvc := permission.NewService()

	router := gin.New()
	router.Use(MultiAuth(fakeSvc, fakeLogger, realPermSvc, db))
	router.GET("/probe", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/probe", nil)
	req.Header.Set("X-API-Key", "rec_"+strings.Repeat("a", 64))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, 401, w.Code, "DB 关闭后 InheritPerms 加载应 401 (D-09 fail-closed)")
	assert.Contains(t, w.Body.String(), "用户权限加载失败")
}

// TestMultiAuthInheritPerms_UsernameFromUser D-10 验证: c.Set("username") 取
// apiKey.User.Username(不是 apiKey.Name),c.Set("nickname") 取 apiKey.User.Nickname。
func TestMultiAuthInheritPerms_UsernameFromUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupInheritPermsTestDB(t)

	userID := createTestUserWithPermissions(t, db, []string{"system:user:list"})
	apiKeyID := createTestAPIKey(t, db, &userID, []string{"read"}, true)

	fakeSvc := &fakeAPIKeyService{
		validKey: &models.APIKey{
			BaseModel:    models.BaseModel{ID: apiKeyID},
			Name:         "apikey-name-not-username", // 与 Username 不同,验证 D-10 修正
			KeyHash:      "fake-hash",
			Salt:         "fake-salt",
			KeyPrefix:    "rec_test",
			UserID:       &userID,
			Scopes:       []string{"read"},
			InheritPerms: true,
			IsActive:     true,
			User: &models.User{
				BaseModel: models.BaseModel{ID: userID},
				Username:  "zhangsan",
				Nickname:  stringPtr("张三"),
				Status:    models.UserStatusEnabled,
			},
		},
	}
	fakeLogger := newFakeUsageLogger()
	realPermSvc := permission.NewService()

	router := gin.New()
	router.Use(MultiAuth(fakeSvc, fakeLogger, realPermSvc, db))
	router.GET("/probe", func(c *gin.Context) {
		assert.Equal(t, "zhangsan", c.GetString("username"),
			"username 应取 apiKey.User.Username (D-10 修正), 不是 apiKey.Name=%q",
			"apikey-name-not-username")
		assert.Equal(t, "张三", c.GetString("nickname"),
			"nickname 应取 apiKey.User.Nickname, 不是空字符串")
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/probe", nil)
	req.Header.Set("X-API-Key", "rec_"+strings.Repeat("a", 64))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	fakeLogger.waitForLog(t)
}

// TestMultiAuthInheritPerms_False_NoUserLoad D-08 验证: InheritPerms=false 时
// 不加载 User 权限,scopes 仅含 API Key 自带。
func TestMultiAuthInheritPerms_False_NoUserLoad(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupInheritPermsTestDB(t)

	// 即使 DB 中有 User + Menu 关联,InheritPerms=false 也不加载
	userID := createTestUserWithPermissions(t, db, []string{"system:user:list"})
	apiKeyID := createTestAPIKey(t, db, &userID, []string{"read"}, false)

	fakeSvc := &fakeAPIKeyService{
		validKey: &models.APIKey{
			BaseModel:    models.BaseModel{ID: apiKeyID},
			Name:         "no-inherit-key",
			KeyHash:      "fake-hash",
			Salt:         "fake-salt",
			KeyPrefix:    "rec_test",
			UserID:       &userID,
			Scopes:       []string{"read"},
			InheritPerms: false,
			IsActive:     true,
			User: &models.User{
				BaseModel: models.BaseModel{ID: userID},
				Username:  "testuser",
				Status:    models.UserStatusEnabled,
			},
		},
	}
	fakeLogger := newFakeUsageLogger()
	realPermSvc := permission.NewService()

	router := gin.New()
	router.Use(MultiAuth(fakeSvc, fakeLogger, realPermSvc, db))
	router.GET("/probe", func(c *gin.Context) {
		scopes := c.MustGet("scopes").([]string)
		assert.ElementsMatch(t, []string{"read"}, scopes,
			"InheritPerms=false 时 scopes 仅含 API Key 自带, 不加载 User 权限 (D-08)")
		_, hasInheritPerms := c.Get("inherit_perms")
		assert.False(t, hasInheritPerms,
			"InheritPerms=false 时不应设置 inherit_perms=true")
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/probe", nil)
	req.Header.Set("X-API-Key", "rec_"+strings.Repeat("a", 64))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	fakeLogger.waitForLog(t)
}

// stringPtr 辅助函数 — 返回 *string 指针(Nickname 是 *string 类型)。
func stringPtr(s string) *string {
	return &s
}
