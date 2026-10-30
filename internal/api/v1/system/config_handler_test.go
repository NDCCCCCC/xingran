package system

// =====================================================================
// Method Enumeration (Plan 72-10 Task 1)
//
// config_handler.go (401 lines) — method coverage:
//   - Statistics      GET  /system/configs/statistics
//   - Create          POST /system/configs
//   - List            POST /system/configs/list
//   - GetByID         POST /system/configs/:id
//   - GetByKey        GET  /system/configs/key/:configKey
//   - Update          POST /system/configs/:id/update
//   - Delete          POST /system/configs/:id/delete
//   - BatchDelete     POST /system/configs/batch-delete
//   - RefreshCache    POST /system/configs/refresh-cache
//
// Per CLAUDE.md: ConfigType ("Y"/"N") and ConfigIsSystem (0/1) conventions
// are enforced via models.ConfigTypeYes / Models.ConfigIsSystemYes constants.
// =====================================================================

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
)

// setupConfigTestDB creates in-memory SQLite with sys_config schema.
func setupConfigTestDB(t *testing.T) *gorm.DB {
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

// newConfigTestHandler wires a real ConfigService into the handler.
func newConfigTestHandler(t *testing.T, db *gorm.DB) (*ConfigHandler, *gorm.DB) {
	t.Helper()
	svc := systemServices.NewConfigService(db)
	h := NewConfigHandler(svc, nil)
	// Inject empty Core so operlog.Record short-circuits via nil checks.
	h.core = &core.Core{
		CoreInfra:    &core.CoreInfra{},
		CoreServices: &core.CoreServices{},
	}
	return h, db
}

// seedConfigRow inserts a sys_config row directly.
func seedConfigRow(t *testing.T, db *gorm.DB, name, key, value string, cfgType models.ConfigType, isSystem models.ConfigIsSystem) string {
	t.Helper()
	id := uuid.NewString()
	require.NoError(t, db.Exec(`
		INSERT INTO sys_config (id, config_name, config_key, config_value, config_type, is_system, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))`,
		id, name, key, value, string(cfgType), int(isSystem)).Error)
	return id
}

// TC1: Statistics - returns count summary
func TestConfigHandler_Statistics(t *testing.T) {
	h, db := newConfigTestHandler(t, setupConfigTestDB(t))
	seedConfigRow(t, db, "sys1", "k1", "v1", models.ConfigTypeYes, models.ConfigIsSystemNo)
	seedConfigRow(t, db, "sys2", "k2", "v2", models.ConfigTypeYes, models.ConfigIsSystemNo)
	seedConfigRow(t, db, "usr1", "k3", "v3", models.ConfigTypeNo, models.ConfigIsSystemNo)

	w := doJSON(t, h.Statistics, "POST", "/system/configs/statistics", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Total    int64 `json:"total"`
			Active   int64 `json:"active"`
			Inactive int64 `json:"inactive"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, int64(3), resp.Data.Total)
	assert.Equal(t, int64(2), resp.Data.Active)
	assert.Equal(t, int64(1), resp.Data.Inactive)
}

// TC2: Statistics - empty DB returns zero
func TestConfigHandler_Statistics_EmptyDB(t *testing.T) {
	h, _ := newConfigTestHandler(t, setupConfigTestDB(t))
	w := doJSON(t, h.Statistics, "POST", "/system/configs/statistics", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Total int64 `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, int64(0), resp.Data.Total)
}

// TC3: Create - success
func TestConfigHandler_Create_Success(t *testing.T) {
	h, db := newConfigTestHandler(t, setupConfigTestDB(t))

	body := requests.ConfigCreateRequest{
		ConfigName:  "用户初始密码",
		ConfigKey:   "sys.user.init.password",
		ConfigValue: "abc123",
		ConfigType:  models.ConfigTypeYes,
		IsSystem:    0,
	}
	w := doJSON(t, h.Create, "POST", "/system/configs", body, nil)
	assert.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

	var resp struct {
		Code int `json:"code"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)

	// Verify row was persisted
	var count int64
	db.Raw("SELECT COUNT(*) FROM sys_config WHERE config_key = ?", "sys.user.init.password").Scan(&count)
	assert.Equal(t, int64(1), count)
}

// TC4: Create - duplicate key returns error
func TestConfigHandler_Create_DuplicateKey(t *testing.T) {
	h, db := newConfigTestHandler(t, setupConfigTestDB(t))
	seedConfigRow(t, db, "dupName", "sys.dup.key", "v1", models.ConfigTypeYes, models.ConfigIsSystemNo)

	body := requests.ConfigCreateRequest{
		ConfigName:  "another name",
		ConfigKey:   "sys.dup.key",
		ConfigValue: "v2",
		ConfigType:  models.ConfigTypeYes,
		IsSystem:    0,
	}
	w := doJSON(t, h.Create, "POST", "/system/configs", body, nil)
	assert.NotEqual(t, http.StatusOK, w.Code, "duplicate key should error")
}

// TC5: Create - bad JSON returns 400
func TestConfigHandler_Create_BadJSON(t *testing.T) {
	h, _ := newConfigTestHandler(t, setupConfigTestDB(t))
	w := doJSON(t, h.Create, "POST", "/system/configs", "{not-valid", nil)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC6: Create - captcha enabled invalid value rejected
func TestConfigHandler_Create_CaptchaInvalidValue(t *testing.T) {
	h, _ := newConfigTestHandler(t, setupConfigTestDB(t))
	body := requests.ConfigCreateRequest{
		ConfigName:  "captcha启用",
		ConfigKey:   "sys.account.captchaEnabled",
		ConfigValue: "off", // invalid, must be disabled/normal/slider
		ConfigType:  models.ConfigTypeYes,
		IsSystem:    0,
	}
	w := doJSON(t, h.Create, "POST", "/system/configs", body, nil)
	assert.NotEqual(t, http.StatusOK, w.Code, "captcha value 'off' must be rejected")
}

// TC7: Create - captcha enabled valid value accepted
func TestConfigHandler_Create_CaptchaValidValue(t *testing.T) {
	h, _ := newConfigTestHandler(t, setupConfigTestDB(t))
	body := requests.ConfigCreateRequest{
		ConfigName:  "captcha启用",
		ConfigKey:   "sys.account.captchaEnabled",
		ConfigValue: "slider",
		ConfigType:  models.ConfigTypeYes,
		IsSystem:    0,
	}
	w := doJSON(t, h.Create, "POST", "/system/configs", body, nil)
	assert.Equal(t, http.StatusOK, w.Code, "captcha value 'slider' must be accepted")
}

// TC8: List - empty
func TestConfigHandler_List_Empty(t *testing.T) {
	h, _ := newConfigTestHandler(t, setupConfigTestDB(t))
	w := doJSON(t, h.List, "POST", "/system/configs/list", map[string]interface{}{}, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Total int64 `json:"total"`
			List  []interface{} `json:"list"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, int64(0), resp.Data.Total)
}

// TC9: List - with configKey filter
func TestConfigHandler_List_FilterByKey(t *testing.T) {
	h, db := newConfigTestHandler(t, setupConfigTestDB(t))
	seedConfigRow(t, db, "user_init", "sys.user.init.password", "v1", models.ConfigTypeYes, models.ConfigIsSystemNo)
	seedConfigRow(t, db, "account", "sys.account.captchaEnabled", "disabled", models.ConfigTypeYes, models.ConfigIsSystemNo)
	seedConfigRow(t, db, "registration", "sys.account.registerEnabled", "true", models.ConfigTypeYes, models.ConfigIsSystemNo)

	w := doJSON(t, h.List, "POST", "/system/configs/list",
		map[string]interface{}{"configKey": "sys.user", "current": 1, "pageSize": 10}, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Total int64 `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, int64(1), resp.Data.Total)
}

// TC10: List - with configType filter
func TestConfigHandler_List_FilterByType(t *testing.T) {
	h, db := newConfigTestHandler(t, setupConfigTestDB(t))
	seedConfigRow(t, db, "yes1", "k1", "v1", models.ConfigTypeYes, models.ConfigIsSystemNo)
	seedConfigRow(t, db, "yes2", "k2", "v2", models.ConfigTypeYes, models.ConfigIsSystemNo)
	seedConfigRow(t, db, "no1", "k3", "v3", models.ConfigTypeNo, models.ConfigIsSystemNo)

	w := doJSON(t, h.List, "POST", "/system/configs/list",
		map[string]interface{}{"configType": "N", "current": 1, "pageSize": 10}, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Total int64 `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int64(1), resp.Data.Total)
}

// TC11: GetByID - success
func TestConfigHandler_GetByID_Success(t *testing.T) {
	h, db := newConfigTestHandler(t, setupConfigTestDB(t))
	id := seedConfigRow(t, db, "getme", "sys.getme", "v1", models.ConfigTypeYes, models.ConfigIsSystemNo)

	w := doJSON(t, h.GetByID, "POST", "/system/configs/"+id, nil, map[string]string{"id": id})
	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Code int `json:"code"`
		Data models.Config `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, "sys.getme", resp.Data.ConfigKey)
}

// TC12: GetByID - missing returns error
func TestConfigHandler_GetByID_NotFound(t *testing.T) {
	h, _ := newConfigTestHandler(t, setupConfigTestDB(t))
	missing := uuid.NewString()
	w := doJSON(t, h.GetByID, "POST", "/system/configs/"+missing, nil, map[string]string{"id": missing})
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC13: GetByKey - success returns matching config
func TestConfigHandler_GetByKey_Success(t *testing.T) {
	h, db := newConfigTestHandler(t, setupConfigTestDB(t))
	seedConfigRow(t, db, "n1", "sys.first", "v1", models.ConfigTypeYes, models.ConfigIsSystemNo)
	seedConfigRow(t, db, "n2", "sys.second", "v2", models.ConfigTypeYes, models.ConfigIsSystemNo)

	w := doJSON(t, h.GetByKey, "GET", "/system/configs/key/sys.second", nil,
		map[string]string{"configKey": "sys.second"})
	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Code int `json:"code"`
		Data models.Config `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, "sys.second", resp.Data.ConfigKey)
	assert.Equal(t, "v2", resp.Data.ConfigValue)
}

// TC14: GetByKey - not found
func TestConfigHandler_GetByKey_NotFound(t *testing.T) {
	h, _ := newConfigTestHandler(t, setupConfigTestDB(t))
	w := doJSON(t, h.GetByKey, "GET", "/system/configs/key/sys.missing", nil,
		map[string]string{"configKey": "sys.missing"})
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC15: Update - success
func TestConfigHandler_Update_Success(t *testing.T) {
	h, db := newConfigTestHandler(t, setupConfigTestDB(t))
	id := seedConfigRow(t, db, "updateme", "sys.update", "v1", models.ConfigTypeYes, models.ConfigIsSystemNo)

	body := requests.ConfigUpdateRequest{
		ID:          id,
		ConfigName:  "updateme",
		ConfigKey:   "sys.update",
		ConfigValue: "v2",
		ConfigType:  models.ConfigTypeYes,
		Remark:      stringPtr("updated"),
	}
	w := doJSON(t, h.Update, "POST", "/system/configs/"+id+"/update", body, map[string]string{"id": id})
	assert.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

	var stored models.Config
	require.NoError(t, db.Where("id = ?", id).First(&stored).Error)
	assert.Equal(t, "v2", stored.ConfigValue)
}

// TC16: Update - bad JSON returns 400
func TestConfigHandler_Update_BadJSON(t *testing.T) {
	h, db := newConfigTestHandler(t, setupConfigTestDB(t))
	id := seedConfigRow(t, db, "upd", "sys.upd", "v1", models.ConfigTypeYes, models.ConfigIsSystemNo)
	w := doJSON(t, h.Update, "POST", "/system/configs/"+id+"/update", "{not-json", map[string]string{"id": id})
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC17: Update - not found returns error
func TestConfigHandler_Update_NotFound(t *testing.T) {
	h, _ := newConfigTestHandler(t, setupConfigTestDB(t))
	missing := uuid.NewString()
	body := requests.ConfigUpdateRequest{
		ID:          missing,
		ConfigName:  "x",
		ConfigKey:   "sys.x",
		ConfigValue: "y",
		ConfigType:  models.ConfigTypeYes,
	}
	w := doJSON(t, h.Update, "POST", "/system/configs/"+missing+"/update", body, map[string]string{"id": missing})
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC18: Update - isCaptchaConfig path executes (no error)
func TestConfigHandler_Update_CaptchaConfig(t *testing.T) {
	h, db := newConfigTestHandler(t, setupConfigTestDB(t))
	id := seedConfigRow(t, db, "captcha", "sys.account.captchaEnabled", "disabled", models.ConfigTypeYes, models.ConfigIsSystemNo)

	body := requests.ConfigUpdateRequest{
		ID:          id,
		ConfigName:  "captcha",
		ConfigKey:   "sys.account.captchaEnabled",
		ConfigValue: "normal",
		ConfigType:  models.ConfigTypeYes,
	}
	w := doJSON(t, h.Update, "POST", "/system/configs/"+id+"/update", body, map[string]string{"id": id})
	assert.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
}

// TC19: Update - invalid encryption value rejected
func TestConfigHandler_Update_InvalidEncryptionValue(t *testing.T) {
	h, db := newConfigTestHandler(t, setupConfigTestDB(t))
	id := seedConfigRow(t, db, "enc", "sys.request.encryption.enabled", "false", models.ConfigTypeYes, models.ConfigIsSystemNo)

	body := requests.ConfigUpdateRequest{
		ID:          id,
		ConfigName:  "enc",
		ConfigKey:   "sys.request.encryption.enabled",
		ConfigValue: "maybe", // invalid
		ConfigType:  models.ConfigTypeYes,
	}
	w := doJSON(t, h.Update, "POST", "/system/configs/"+id+"/update", body, map[string]string{"id": id})
	assert.NotEqual(t, http.StatusOK, w.Code, "invalid encryption value must error")
}

// TC20: Delete - success
func TestConfigHandler_Delete_Success(t *testing.T) {
	h, db := newConfigTestHandler(t, setupConfigTestDB(t))
	id := seedConfigRow(t, db, "delme", "sys.delme", "v1", models.ConfigTypeYes, models.ConfigIsSystemNo)

	w := doJSON(t, h.Delete, "POST", "/system/configs/"+id+"/delete", nil, map[string]string{"id": id})
	assert.Equal(t, http.StatusOK, w.Code)

	var deletedAt *string
	require.NoError(t, db.Raw("SELECT deleted_at FROM sys_config WHERE id = ?", id).Scan(&deletedAt).Error)
	assert.NotNil(t, deletedAt, "row should be soft-deleted")
}

// TC21: Delete - system config cannot be deleted
func TestConfigHandler_Delete_SystemConfigRefused(t *testing.T) {
	h, db := newConfigTestHandler(t, setupConfigTestDB(t))
	id := seedConfigRow(t, db, "sysInt", "sys.protected", "v1", models.ConfigTypeYes, models.ConfigIsSystemYes)

	w := doJSON(t, h.Delete, "POST", "/system/configs/"+id+"/delete", nil, map[string]string{"id": id})
	assert.NotEqual(t, http.StatusOK, w.Code, "system config should not be deletable")
}

// TC22: Delete - not found returns error
func TestConfigHandler_Delete_NotFound(t *testing.T) {
	h, _ := newConfigTestHandler(t, setupConfigTestDB(t))
	missing := uuid.NewString()
	w := doJSON(t, h.Delete, "POST", "/system/configs/"+missing+"/delete", nil, map[string]string{"id": missing})
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC23: BatchDelete - success
func TestConfigHandler_BatchDelete_Success(t *testing.T) {
	h, db := newConfigTestHandler(t, setupConfigTestDB(t))
	id1 := seedConfigRow(t, db, "b1", "sys.batch1", "v1", models.ConfigTypeYes, models.ConfigIsSystemNo)
	id2 := seedConfigRow(t, db, "b2", "sys.batch2", "v2", models.ConfigTypeYes, models.ConfigIsSystemNo)

	w := doJSON(t, h.BatchDelete, "POST", "/system/configs/batch-delete",
		map[string]interface{}{"ids": []string{id1, id2}}, nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var alive int64
	db.Raw("SELECT COUNT(*) FROM sys_config WHERE deleted_at IS NULL").Scan(&alive)
	assert.Equal(t, int64(0), alive)
}

// TC24: BatchDelete - empty ids fails validation
func TestConfigHandler_BatchDelete_Empty(t *testing.T) {
	h, _ := newConfigTestHandler(t, setupConfigTestDB(t))
	w := doJSON(t, h.BatchDelete, "POST", "/system/configs/batch-delete",
		map[string]interface{}{"ids": []string{}}, nil)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC25: RefreshCache - success
func TestConfigHandler_RefreshCache_Success(t *testing.T) {
	h, db := newConfigTestHandler(t, setupConfigTestDB(t))
	seedConfigRow(t, db, "r1", "sys.r1", "v1", models.ConfigTypeYes, models.ConfigIsSystemNo)
	seedConfigRow(t, db, "r2", "sys.r2", "v2", models.ConfigTypeYes, models.ConfigIsSystemNo)

	w := doJSON(t, h.RefreshCache, "POST", "/system/configs/refresh-cache", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Code int `json:"code"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	// sanity: rows still exist after refresh
	var count int64
	db.Raw("SELECT COUNT(*) FROM sys_config").Scan(&count)
	assert.Equal(t, int64(2), count)
}

// TC26: CaptchaConfig helper - isCaptchaConfig true cases
func TestConfigHandler_IsCaptchaConfig_True(t *testing.T) {
	h := &ConfigHandler{}
	captchaKeys := []string{
		"sys.account.captchaEnabled",
		"sys.account.captchaType",
		"sys.account.captchaExpireTime",
		"sys.account.captchaMaxAttempts",
		"sys.account.ipRateLimit",
		"sys.account.loginMaxRetry",
		"sys.account.loginLockTime",
		"sys.account.captchaBackgroundMode",
		"sys.account.captchaPieceShape",
		"sys.account.captchaDifficulty",
	}
	for _, key := range captchaKeys {
		assert.True(t, h.isCaptchaConfig(key), "%s should be captcha key", key)
	}
}

// TC27: CaptchaConfig helper - isCaptchaConfig false cases
func TestConfigHandler_IsCaptchaConfig_False(t *testing.T) {
	h := &ConfigHandler{}
	assert.False(t, h.isCaptchaConfig("sys.user.init.password"))
	assert.False(t, h.isCaptchaConfig("sys.request.encryption.enabled"))
	assert.False(t, h.isCaptchaConfig(""))
}

// TC28: validateCaptchaConfigValue - non-captchaKey returns nil
func TestConfigHandler_ValidateCaptchaConfigValue_NonCaptchaKey(t *testing.T) {
	h := &ConfigHandler{}
	assert.NoError(t, h.validateCaptchaConfigValue("sys.user.init.password", "anything"))
}

// TC29: validateCaptchaConfigValue - valid values
func TestConfigHandler_ValidateCaptchaConfigValue_Valid(t *testing.T) {
	h := &ConfigHandler{}
	assert.NoError(t, h.validateCaptchaConfigValue("sys.account.captchaEnabled", "disabled"))
	assert.NoError(t, h.validateCaptchaConfigValue("sys.account.captchaEnabled", "normal"))
	assert.NoError(t, h.validateCaptchaConfigValue("sys.account.captchaEnabled", "slider"))
}

// TC30: validateCaptchaConfigValue - invalid value rejected
func TestConfigHandler_ValidateCaptchaConfigValue_Invalid(t *testing.T) {
	h := &ConfigHandler{}
	err := h.validateCaptchaConfigValue("sys.account.captchaEnabled", "off")
	assert.Error(t, err)
	err = h.validateCaptchaConfigValue("sys.account.captchaEnabled", "true")
	assert.Error(t, err)
}

// TC31: Update - encryption flag triggers middleware refresh path (no panic)
func TestConfigHandler_Update_EncryptionFlag(t *testing.T) {
	h, db := newConfigTestHandler(t, setupConfigTestDB(t))
	id := seedConfigRow(t, db, "enc", "sys.request.encryption.enabled", "false", models.ConfigTypeYes, models.ConfigIsSystemNo)

	body := requests.ConfigUpdateRequest{
		ID:          id,
		ConfigName:  "enc",
		ConfigKey:   "sys.request.encryption.enabled",
		ConfigValue: "true",
		ConfigType:  models.ConfigTypeYes,
	}
	w := doJSON(t, h.Update, "POST", "/system/configs/"+id+"/update", body, map[string]string{"id": id})
	assert.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
}

// TC32: CaptchaPieceShape - isCaptchaConfig matches
func TestConfigHandler_IsCaptchaConfig_PieceShape(t *testing.T) {
	h := &ConfigHandler{}
	assert.True(t, h.isCaptchaConfig("sys.account.captchaPieceShape"))
}

// TC33: List - bad JSON body falls back to empty params
func TestConfigHandler_List_BadJSON(t *testing.T) {
	h, _ := newConfigTestHandler(t, setupConfigTestDB(t))
	w := doJSON(t, h.List, "POST", "/system/configs/list", "{not-json", nil)
	assert.Equal(t, http.StatusOK, w.Code, "list should tolerate malformed JSON and return defaults")
}

// TC34: List - service error bubbles up
func TestConfigHandler_List_ServiceError(t *testing.T) {
	h, db := newConfigTestHandler(t, setupConfigTestDB(t))
	// drop the table to force service error
	require.NoError(t, db.Exec("DROP TABLE sys_config").Error)
	w := doJSON(t, h.List, "POST", "/system/configs/list", map[string]interface{}{}, nil)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC35: CaptchaConfig coverage - lower-case comparison across all keys
func TestConfigHandler_IsCaptchaConfig_AllKeys(t *testing.T) {
	h := &ConfigHandler{}
	keys := []string{
		"sys.account.captchaEnabled",
		"sys.account.captchaType",
		"sys.account.captchaExpireTime",
		"sys.account.captchaMaxAttempts",
		"sys.account.ipRateLimit",
		"sys.account.loginMaxRetry",
		"sys.account.loginLockTime",
		"sys.account.captchaBackgroundMode",
		"sys.account.captchaPieceShape",
		"sys.account.captchaDifficulty",
	}
	matched := 0
	for _, k := range keys {
		if h.isCaptchaConfig(k) {
			matched++
		}
	}
	assert.Equal(t, len(keys), matched)
}

// TC36: WithCore - nil receiver returns nil
func TestConfigHandler_WithCore_NilReceiver(t *testing.T) {
	var h *ConfigHandler
	result := h.WithCore(&core.Core{})
	assert.Nil(t, result)
}

// TC37: Update - remark field stored
func TestConfigHandler_Update_StoredRemark(t *testing.T) {
	h, db := newConfigTestHandler(t, setupConfigTestDB(t))
	id := seedConfigRow(t, db, "rmk", "sys.rmk", "v1", models.ConfigTypeYes, models.ConfigIsSystemNo)

	r := "important remark"
	body := requests.ConfigUpdateRequest{
		ID:          id,
		ConfigName:  "rmk",
		ConfigKey:   "sys.rmk",
		ConfigValue: "v2",
		ConfigType:  models.ConfigTypeYes,
		Remark:      &r,
	}
	w := doJSON(t, h.Update, "POST", "/system/configs/"+id+"/update", body, map[string]string{"id": id})
	require.Equal(t, http.StatusOK, w.Code)

	var stored models.Config
	require.NoError(t, db.Where("id = ?", id).First(&stored).Error)
	assert.Equal(t, "important remark", stored.Remark)
}

// TC38: GetByID - empty id returns error
func TestConfigHandler_GetByID_EmptyID(t *testing.T) {
	h, _ := newConfigTestHandler(t, setupConfigTestDB(t))
	w := doJSON(t, h.GetByID, "POST", "/system/configs/", nil, map[string]string{"id": ""})
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC39: Statistics - service error bubbles up
func TestConfigHandler_Statistics_ServiceError(t *testing.T) {
	h, db := newConfigTestHandler(t, setupConfigTestDB(t))
	require.NoError(t, db.Exec("DROP TABLE sys_config").Error)
	w := doJSON(t, h.Statistics, "POST", "/system/configs/statistics", nil, nil)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC40: RefreshCache - empty DB still succeeds
func TestConfigHandler_RefreshCache_EmptyDB(t *testing.T) {
	h, _ := newConfigTestHandler(t, setupConfigTestDB(t))
	w := doJSON(t, h.RefreshCache, "POST", "/system/configs/refresh-cache", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

// TC41: Update - system config cannot change key
func TestConfigHandler_Update_SystemConfigKeyProtected(t *testing.T) {
	h, db := newConfigTestHandler(t, setupConfigTestDB(t))
	id := seedConfigRow(t, db, "sysInt", "sys.protected", "v1", models.ConfigTypeYes, models.ConfigIsSystemYes)

	body := requests.ConfigUpdateRequest{
		ID:          id,
		ConfigName:  "sysInt",
		ConfigKey:   "sys.changed", // attempt to change key
		ConfigValue: "v2",
		ConfigType:  models.ConfigTypeYes,
	}
	w := doJSON(t, h.Update, "POST", "/system/configs/"+id+"/update", body, map[string]string{"id": id})
	assert.NotEqual(t, http.StatusOK, w.Code, "system config key change must be rejected")
}

// TC42: ConfigListParams default values
func TestConfigHandler_DefaultConfigListParams(t *testing.T) {
	p := requests.DefaultConfigListParams()
	current, pageSize := p.GetPagination()
	assert.GreaterOrEqual(t, current, 1)
	assert.GreaterOrEqual(t, pageSize, 1)
}

// TC43: List - pagination via raw map (current/pageSize as float)
func TestConfigHandler_List_RawPagination(t *testing.T) {
	h, db := newConfigTestHandler(t, setupConfigTestDB(t))
	for i := 0; i < 5; i++ {
		seedConfigRow(t, db, "n", "raw.k"+string(rune('a'+i)), "v",
			models.ConfigTypeYes, models.ConfigIsSystemNo)
	}
	w := doJSON(t, h.List, "POST", "/system/configs/list",
		map[string]interface{}{"current": float64(1), "pageSize": float64(3)}, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Total    int64 `json:"total"`
			Current  int    `json:"current"`
			PageSize int    `json:"pageSize"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int64(5), resp.Data.Total)
	assert.Equal(t, 1, resp.Data.Current)
	assert.Equal(t, 3, resp.Data.PageSize)
}

// TC44: List - all fields filter combination
func TestConfigHandler_List_AllFieldsFilter(t *testing.T) {
	h, db := newConfigTestHandler(t, setupConfigTestDB(t))
	seedConfigRow(t, db, "nameMatch", "k1", "v1", models.ConfigTypeYes, models.ConfigIsSystemNo)
	seedConfigRow(t, db, "other", "k2", "v2", models.ConfigTypeYes, models.ConfigIsSystemNo)

	w := doJSON(t, h.List, "POST", "/system/configs/list",
		map[string]interface{}{
			"configName": "nameMatch",
			"configKey":  "k1",
			"configType": "Y",
			"current":    1,
			"pageSize":   10,
		}, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Total int64 `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int64(1), resp.Data.Total)
}

// TC45: BatchDelete - mixed system + non-system rejected
func TestConfigHandler_BatchDelete_ContainsSystemConfig(t *testing.T) {
	h, db := newConfigTestHandler(t, setupConfigTestDB(t))
	id1 := seedConfigRow(t, db, "u1", "sys.us1", "v1", models.ConfigTypeYes, models.ConfigIsSystemNo)
	id2 := seedConfigRow(t, db, "s1", "sys.sys1", "v1", models.ConfigTypeYes, models.ConfigIsSystemYes)

	w := doJSON(t, h.BatchDelete, "POST", "/system/configs/batch-delete",
		map[string]interface{}{"ids": []string{id1, id2}}, nil)
	assert.NotEqual(t, http.StatusOK, w.Code, "batch with system config must error")
	_ = strings.Builder{} // keep strings import alive
}

// TC46: GetByKey - empty key returns error
func TestConfigHandler_GetByKey_EmptyKey(t *testing.T) {
	h, _ := newConfigTestHandler(t, setupConfigTestDB(t))
	w := doJSON(t, h.GetByKey, "GET", "/system/configs/key/", nil, map[string]string{"configKey": ""})
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC47: GetByKey - service error bubbles up
func TestConfigHandler_GetByKey_ServiceError(t *testing.T) {
	h, db := newConfigTestHandler(t, setupConfigTestDB(t))
	require.NoError(t, db.Exec("DROP TABLE sys_config").Error)
	w := doJSON(t, h.GetByKey, "GET", "/system/configs/key/sys.x", nil, map[string]string{"configKey": "sys.x"})
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC48: Update - service error bubbles up
func TestConfigHandler_Update_ServiceError(t *testing.T) {
	h, db := newConfigTestHandler(t, setupConfigTestDB(t))
	require.NoError(t, db.Exec("DROP TABLE sys_config").Error)
	missing := uuid.NewString()
	body := requests.ConfigUpdateRequest{
		ID:          missing,
		ConfigName:  "x",
		ConfigKey:   "sys.x",
		ConfigValue: "y",
		ConfigType:  models.ConfigTypeYes,
	}
	w := doJSON(t, h.Update, "POST", "/system/configs/"+missing+"/update", body, map[string]string{"id": missing})
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC49: Delete - service error bubbles up
func TestConfigHandler_Delete_ServiceError(t *testing.T) {
	h, db := newConfigTestHandler(t, setupConfigTestDB(t))
	require.NoError(t, db.Exec("DROP TABLE sys_config").Error)
	missing := uuid.NewString()
	w := doJSON(t, h.Delete, "POST", "/system/configs/"+missing+"/delete", nil, map[string]string{"id": missing})
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC50: BatchDelete - service error bubbles up
func TestConfigHandler_BatchDelete_ServiceError(t *testing.T) {
	h, db := newConfigTestHandler(t, setupConfigTestDB(t))
	require.NoError(t, db.Exec("DROP TABLE sys_config").Error)
	w := doJSON(t, h.BatchDelete, "POST", "/system/configs/batch-delete",
		map[string]interface{}{"ids": []string{uuid.NewString()}}, nil)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// stringPtr returns a pointer to the given string.
func stringPtr(s string) *string { return &s }
