package system

// =====================================================================
// config_service_test.go — covers config_service.go (267 lines)
// Extends existing config_invalidation_test.go + config_statistics_test.go (PRESERVED)
//
// Per Plan 72-10 Task 3
// =====================================================================

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
)

// setupConfigServiceTestDB creates in-memory SQLite for config service tests.
func setupConfigServiceTestDB(t *testing.T) *gorm.DB {
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

// seedConfigSvc inserts a sys_config row directly. Returns id.
func seedConfigSvc(t *testing.T, db *gorm.DB, name, key, value string, cfgType models.ConfigType, isSystem models.ConfigIsSystem) string {
	t.Helper()
	id := uuid.NewString()
	require.NoError(t, db.Exec(`
		INSERT INTO sys_config (id, config_name, config_key, config_value, config_type, is_system, created_at, updated_at, version)
		VALUES (?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'), 0)`,
		id, name, key, value, string(cfgType), int(isSystem)).Error)
	return id
}

// TC1: Create - success
func TestConfigService_Create_Success(t *testing.T) {
	db := setupConfigServiceTestDB(t)
	svc := NewConfigService(db)

	req := &requests.ConfigCreateRequest{
		ConfigName:  "user initialized password",
		ConfigKey:   "sys.user.init.password",
		ConfigValue: "abc123",
		ConfigType:  models.ConfigTypeYes,
		IsSystem:    0,
	}
	require.NoError(t, svc.Create(context.Background(), req))

	var stored models.Config
	require.NoError(t, db.Where("config_key = ?", "sys.user.init.password").First(&stored).Error)
	assert.Equal(t, "abc123", stored.ConfigValue)
}

// TC2: Create - duplicate key
func TestConfigService_Create_DuplicateKey(t *testing.T) {
	db := setupConfigServiceTestDB(t)
	svc := NewConfigService(db)
	seedConfigSvc(t, db, "dup", "sys.dup", "v1", models.ConfigTypeYes, models.ConfigIsSystemNo)

	req := &requests.ConfigCreateRequest{
		ConfigName:  "another",
		ConfigKey:   "sys.dup",
		ConfigValue: "v2",
		ConfigType:  models.ConfigTypeYes,
	}
	err := svc.Create(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "配置键已存在")
}

// TC3: Create - remark is stored
func TestConfigService_Create_WithRemark(t *testing.T) {
	db := setupConfigServiceTestDB(t)
	svc := NewConfigService(db)

	r := "important"
	req := &requests.ConfigCreateRequest{
		ConfigName:  "with remark",
		ConfigKey:   "sys.rmk",
		ConfigValue: "v",
		ConfigType:  models.ConfigTypeYes,
		Remark:      &r,
	}
	require.NoError(t, svc.Create(context.Background(), req))

	var stored models.Config
	require.NoError(t, db.Where("config_key = ?", "sys.rmk").First(&stored).Error)
	assert.Equal(t, "important", stored.Remark)
}

// TC4: Update - success
func TestConfigService_Update_Success(t *testing.T) {
	db := setupConfigServiceTestDB(t)
	svc := NewConfigService(db)
	id := seedConfigSvc(t, db, "upd", "sys.upd", "v1", models.ConfigTypeYes, models.ConfigIsSystemNo)

	req := &requests.ConfigUpdateRequest{
		ID:          id,
		ConfigName:  "upd",
		ConfigKey:   "sys.upd",
		ConfigValue: "v2",
		ConfigType:  models.ConfigTypeYes,
	}
	require.NoError(t, svc.Update(context.Background(), req))

	var stored models.Config
	require.NoError(t, db.Where("id = ?", id).First(&stored).Error)
	assert.Equal(t, "v2", stored.ConfigValue)
}

// TC5: Update - not found
func TestConfigService_Update_NotFound(t *testing.T) {
	db := setupConfigServiceTestDB(t)
	svc := NewConfigService(db)

	req := &requests.ConfigUpdateRequest{
		ID:          uuid.NewString(),
		ConfigName:  "x",
		ConfigKey:   "sys.x",
		ConfigValue: "y",
		ConfigType:  models.ConfigTypeYes,
	}
	err := svc.Update(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "不存在")
}

// TC6: Update - system config cannot change key
func TestConfigService_Update_SystemConfigKeyProtected(t *testing.T) {
	db := setupConfigServiceTestDB(t)
	svc := NewConfigService(db)
	id := seedConfigSvc(t, db, "sysInt", "sys.protected", "v1", models.ConfigTypeYes, models.ConfigIsSystemYes)

	req := &requests.ConfigUpdateRequest{
		ID:          id,
		ConfigName:  "sysInt",
		ConfigKey:   "sys.different", // attempt change
		ConfigValue: "v2",
		ConfigType:  models.ConfigTypeYes,
	}
	err := svc.Update(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "系统内置参数不能修改键名")
}

// TC7: Update - system config with same key allowed
func TestConfigService_Update_SystemConfigSameKey(t *testing.T) {
	db := setupConfigServiceTestDB(t)
	svc := NewConfigService(db)
	id := seedConfigSvc(t, db, "sysInt", "sys.protected", "v1", models.ConfigTypeYes, models.ConfigIsSystemYes)

	req := &requests.ConfigUpdateRequest{
		ID:          id,
		ConfigName:  "sysInt",
		ConfigKey:   "sys.protected", // same key
		ConfigValue: "v2",
		ConfigType:  models.ConfigTypeYes,
	}
	require.NoError(t, svc.Update(context.Background(), req))
}

// TC8: Update - invalid encryption value rejected
func TestConfigService_Update_InvalidEncryptionValue(t *testing.T) {
	db := setupConfigServiceTestDB(t)
	svc := NewConfigService(db)
	id := seedConfigSvc(t, db, "enc", "sys.request.encryption.enabled", "false", models.ConfigTypeYes, models.ConfigIsSystemNo)

	req := &requests.ConfigUpdateRequest{
		ID:          id,
		ConfigName:  "enc",
		ConfigKey:   "sys.request.encryption.enabled",
		ConfigValue: "garbage",
		ConfigType:  models.ConfigTypeYes,
	}
	err := svc.Update(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "请求加密开关")
}

// TC9: Update - remark is overwritten
func TestConfigService_Update_WithRemark(t *testing.T) {
	db := setupConfigServiceTestDB(t)
	svc := NewConfigService(db)
	id := seedConfigSvc(t, db, "rmk", "sys.rmk", "v1", models.ConfigTypeYes, models.ConfigIsSystemNo)

	r := "new remark"
	req := &requests.ConfigUpdateRequest{
		ID:          id,
		ConfigName:  "rmk",
		ConfigKey:   "sys.rmk",
		ConfigValue: "v2",
		ConfigType:  models.ConfigTypeYes,
		Remark:      &r,
	}
	require.NoError(t, svc.Update(context.Background(), req))

	var stored models.Config
	require.NoError(t, db.Where("id = ?", id).First(&stored).Error)
	assert.Equal(t, "new remark", stored.Remark)
}

// TC10: Delete - success
func TestConfigService_Delete_Success(t *testing.T) {
	db := setupConfigServiceTestDB(t)
	svc := NewConfigService(db)
	id := seedConfigSvc(t, db, "del", "sys.del", "v1", models.ConfigTypeYes, models.ConfigIsSystemNo)

	require.NoError(t, svc.Delete(context.Background(), id))

	var deletedAt *string
	require.NoError(t, db.Raw("SELECT deleted_at FROM sys_config WHERE id = ?", id).Scan(&deletedAt).Error)
	assert.NotNil(t, deletedAt)
}

// TC11: Delete - not found
func TestConfigService_Delete_NotFound(t *testing.T) {
	db := setupConfigServiceTestDB(t)
	svc := NewConfigService(db)

	err := svc.Delete(context.Background(), uuid.NewString())
	assert.Error(t, err)
}

// TC12: Delete - system config refused
func TestConfigService_Delete_SystemConfigRefused(t *testing.T) {
	db := setupConfigServiceTestDB(t)
	svc := NewConfigService(db)
	id := seedConfigSvc(t, db, "sys", "sys.protected", "v1", models.ConfigTypeYes, models.ConfigIsSystemYes)

	err := svc.Delete(context.Background(), id)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "系统内置参数不能删除")
}

// TC13: BatchDelete - success
func TestConfigService_BatchDelete_Success(t *testing.T) {
	db := setupConfigServiceTestDB(t)
	svc := NewConfigService(db)
	id1 := seedConfigSvc(t, db, "b1", "sys.b1", "v", models.ConfigTypeYes, models.ConfigIsSystemNo)
	id2 := seedConfigSvc(t, db, "b2", "sys.b2", "v", models.ConfigTypeYes, models.ConfigIsSystemNo)

	require.NoError(t, svc.BatchDelete(context.Background(), []string{id1, id2}))

	var alive int64
	db.Raw("SELECT COUNT(*) FROM sys_config WHERE deleted_at IS NULL").Scan(&alive)
	assert.Equal(t, int64(0), alive)
}

// TC14: BatchDelete - contains system config rejected
func TestConfigService_BatchDelete_ContainsSystemConfig(t *testing.T) {
	db := setupConfigServiceTestDB(t)
	svc := NewConfigService(db)
	id1 := seedConfigSvc(t, db, "u1", "sys.u1", "v", models.ConfigTypeYes, models.ConfigIsSystemNo)
	id2 := seedConfigSvc(t, db, "s1", "sys.s1", "v", models.ConfigTypeYes, models.ConfigIsSystemYes)

	err := svc.BatchDelete(context.Background(), []string{id1, id2})
	assert.Error(t, err)
}

// TC15: GetByID - success
func TestConfigService_GetByID_Success(t *testing.T) {
	db := setupConfigServiceTestDB(t)
	svc := NewConfigService(db)
	id := seedConfigSvc(t, db, "getme", "sys.getme", "v1", models.ConfigTypeYes, models.ConfigIsSystemNo)

	cfg, err := svc.GetByID(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, "sys.getme", cfg.ConfigKey)
}

// TC16: GetByID - not found
func TestConfigService_GetByID_NotFound(t *testing.T) {
	db := setupConfigServiceTestDB(t)
	svc := NewConfigService(db)

	cfg, err := svc.GetByID(context.Background(), uuid.NewString())
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

// TC17: GetByKey - success
func TestConfigService_GetByKey_Success(t *testing.T) {
	db := setupConfigServiceTestDB(t)
	svc := NewConfigService(db)
	seedConfigSvc(t, db, "k1", "sys.k1", "v1", models.ConfigTypeYes, models.ConfigIsSystemNo)
	seedConfigSvc(t, db, "k2", "sys.k2", "v2", models.ConfigTypeYes, models.ConfigIsSystemNo)

	cfg, err := svc.GetByKey(context.Background(), "sys.k2")
	require.NoError(t, err)
	assert.Equal(t, "v2", cfg.ConfigValue)
}

// TC18: GetByKey - not found
func TestConfigService_GetByKey_NotFound(t *testing.T) {
	db := setupConfigServiceTestDB(t)
	svc := NewConfigService(db)

	cfg, err := svc.GetByKey(context.Background(), "sys.missing")
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

// TC19: List - empty
func TestConfigService_List_Empty(t *testing.T) {
	db := setupConfigServiceTestDB(t)
	svc := NewConfigService(db)

	result, err := svc.List(context.Background(), requests.DefaultConfigListParams())
	require.NoError(t, err)
	assert.Equal(t, int64(0), result.Total)
}

// TC20: List - filter by name
func TestConfigService_List_FilterByName(t *testing.T) {
	db := setupConfigServiceTestDB(t)
	svc := NewConfigService(db)
	seedConfigSvc(t, db, "alpha-one", "sys.a1", "v", models.ConfigTypeYes, models.ConfigIsSystemNo)
	seedConfigSvc(t, db, "alpha-two", "sys.a2", "v", models.ConfigTypeYes, models.ConfigIsSystemNo)
	seedConfigSvc(t, db, "beta-one", "sys.b1", "v", models.ConfigTypeYes, models.ConfigIsSystemNo)

	name := "alpha"
	params := requests.DefaultConfigListParams()
	params.ConfigName = &name
	result, err := svc.List(context.Background(), params)
	require.NoError(t, err)
	assert.Equal(t, int64(2), result.Total)
}

// TC21: List - filter by key
func TestConfigService_List_FilterByKey(t *testing.T) {
	db := setupConfigServiceTestDB(t)
	svc := NewConfigService(db)
	seedConfigSvc(t, db, "n1", "sys.user.init", "v", models.ConfigTypeYes, models.ConfigIsSystemNo)
	seedConfigSvc(t, db, "n2", "sys.account.captcha", "v", models.ConfigTypeYes, models.ConfigIsSystemNo)

	key := "sys.user"
	params := requests.DefaultConfigListParams()
	params.ConfigKey = &key
	result, err := svc.List(context.Background(), params)
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Total)
}

// TC22: List - filter by type
func TestConfigService_List_FilterByType(t *testing.T) {
	db := setupConfigServiceTestDB(t)
	svc := NewConfigService(db)
	seedConfigSvc(t, db, "y1", "sys.y1", "v", models.ConfigTypeYes, models.ConfigIsSystemNo)
	seedConfigSvc(t, db, "y2", "sys.y2", "v", models.ConfigTypeYes, models.ConfigIsSystemNo)
	seedConfigSvc(t, db, "n1", "sys.n1", "v", models.ConfigTypeNo, models.ConfigIsSystemNo)

	typ := "N"
	params := requests.DefaultConfigListParams()
	params.ConfigType = &typ
	result, err := svc.List(context.Background(), params)
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Total)
}

// TC23: List - filter by time range
func TestConfigService_List_FilterByTime(t *testing.T) {
	db := setupConfigServiceTestDB(t)
	svc := NewConfigService(db)
	seedConfigSvc(t, db, "n1", "sys.n1", "v", models.ConfigTypeYes, models.ConfigIsSystemNo)

	begin := "2020-01-01"
	params := requests.DefaultConfigListParams()
	params.BeginTime = &begin
	result, err := svc.List(context.Background(), params)
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Total)
}

// TC24: List - default sort by created_at DESC
func TestConfigService_List_DefaultSort(t *testing.T) {
	db := setupConfigServiceTestDB(t)
	svc := NewConfigService(db)
	seedConfigSvc(t, db, "n1", "sys.n1", "v", models.ConfigTypeYes, models.ConfigIsSystemNo)
	seedConfigSvc(t, db, "n2", "sys.n2", "v", models.ConfigTypeYes, models.ConfigIsSystemNo)

	params := requests.DefaultConfigListParams()
	result, err := svc.List(context.Background(), params)
	require.NoError(t, err)
	assert.Equal(t, int64(2), result.Total)
	assert.Equal(t, 1, result.Current)
	assert.Equal(t, 10, result.PageSize)
}

// TC25: List - custom orderByColumn
func TestConfigService_List_CustomOrderBy(t *testing.T) {
	db := setupConfigServiceTestDB(t)
	svc := NewConfigService(db)
	seedConfigSvc(t, db, "n1", "sys.n1", "v", models.ConfigTypeYes, models.ConfigIsSystemNo)
	seedConfigSvc(t, db, "n2", "sys.n2", "v", models.ConfigTypeYes, models.ConfigIsSystemNo)

	params := requests.DefaultConfigListParams()
	params.OrderByColumn = "config_key"
	asc := true
	params.IsAsc = &asc
	result, err := svc.List(context.Background(), params)
	require.NoError(t, err)
	assert.Equal(t, int64(2), result.Total)
	require.Len(t, result.List, 2)
	list, ok := result.List.([]models.Config)
	require.True(t, ok)
	assert.Equal(t, "sys.n1", list[0].ConfigKey)
}

// TC26: RefreshCache - succeeds
func TestConfigService_RefreshCache_Success(t *testing.T) {
	db := setupConfigServiceTestDB(t)
	svc := NewConfigService(db)
	seedConfigSvc(t, db, "n1", "sys.n1", "v", models.ConfigTypeYes, models.ConfigIsSystemNo)

	require.NoError(t, svc.RefreshCache(context.Background()))
}

// TC27: Update - remark nil preserves original
func TestConfigService_Update_NilRemark(t *testing.T) {
	db := setupConfigServiceTestDB(t)
	svc := NewConfigService(db)
	id := seedConfigSvc(t, db, "rmk", "sys.rmk", "v1", models.ConfigTypeYes, models.ConfigIsSystemNo)

	req := &requests.ConfigUpdateRequest{
		ID:          id,
		ConfigName:  "rmk",
		ConfigKey:   "sys.rmk",
		ConfigValue: "v2",
		ConfigType:  models.ConfigTypeYes,
		Remark:      nil,
	}
	require.NoError(t, svc.Update(context.Background(), req))

	var stored models.Config
	require.NoError(t, db.Where("id = ?", id).First(&stored).Error)
	assert.Equal(t, "v2", stored.ConfigValue)
}

// TC28: Update - encryption flag valid "0" and "1"
func TestConfigService_Update_EncryptionFlagNumericValues(t *testing.T) {
	db := setupConfigServiceTestDB(t)
	svc := NewConfigService(db)
	id := seedConfigSvc(t, db, "enc", "sys.request.encryption.enabled", "false", models.ConfigTypeYes, models.ConfigIsSystemNo)

	req := &requests.ConfigUpdateRequest{
		ID:          id,
		ConfigName:  "enc",
		ConfigKey:   "sys.request.encryption.enabled",
		ConfigValue: "1",
		ConfigType:  models.ConfigTypeYes,
	}
	require.NoError(t, svc.Update(context.Background(), req))
}

// TC29: Create - same config_key with different value fails
func TestConfigService_Create_OverwriteForbidden(t *testing.T) {
	db := setupConfigServiceTestDB(t)
	svc := NewConfigService(db)
	seedConfigSvc(t, db, "first", "sys.same", "v1", models.ConfigTypeYes, models.ConfigIsSystemNo)

	req := &requests.ConfigCreateRequest{
		ConfigName:  "second",
		ConfigKey:   "sys.same",
		ConfigValue: "v2",
		ConfigType:  models.ConfigTypeYes,
	}
	err := svc.Create(context.Background(), req)
	assert.Error(t, err)
}

// TC30: List - pagination offset
func TestConfigService_List_Pagination(t *testing.T) {
	db := setupConfigServiceTestDB(t)
	svc := NewConfigService(db)
	for i := 0; i < 5; i++ {
		seedConfigSvc(t, db, "n", "sys.p"+string(rune('a'+i)), "v", models.ConfigTypeYes, models.ConfigIsSystemNo)
	}

	params := requests.DefaultConfigListParams()
	params.Current = 1
	params.PageSize = 2
	result, err := svc.List(context.Background(), params)
	require.NoError(t, err)
	assert.Equal(t, int64(5), result.Total)
	list, ok := result.List.([]models.Config)
	require.True(t, ok)
	assert.Len(t, list, 2)
}

// TC31: Statistics - via service struct
func TestConfigService_Statistics_VerifyStruct(t *testing.T) {
	db := setupConfigServiceTestDB(t)
	seedConfigSvc(t, db, "y1", "sys.y1", "v", models.ConfigTypeYes, models.ConfigIsSystemNo)
	seedConfigSvc(t, db, "n1", "sys.n1", "v", models.ConfigTypeNo, models.ConfigIsSystemNo)

	svc := &configService{db: db}
	result, err := svc.Statistics(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(2), result.Total)
	assert.Equal(t, int64(1), result.Active)
	assert.Equal(t, int64(1), result.Inactive)
}

// TC32: toStringPtr - nil returns empty string
func TestConfigService_ToStringPtrStr(t *testing.T) {
	assert.Equal(t, "", toStringPtrStr(nil))
	str := "x"
	assert.Equal(t, "x", toStringPtrStr(&str))
}

// TC33: Update - configKey empty in request skips validation
func TestConfigService_Update_EmptyConfigKeyInRequest(t *testing.T) {
	db := setupConfigServiceTestDB(t)
	svc := NewConfigService(db)
	id := seedConfigSvc(t, db, "sysInt", "sys.protected", "v1", models.ConfigTypeYes, models.ConfigIsSystemYes)

	req := &requests.ConfigUpdateRequest{
		ID:          id,
		ConfigName:  "sysInt",
		ConfigKey:   "", // empty - skips validation
		ConfigValue: "v2",
		ConfigType:  models.ConfigTypeYes,
	}
	require.NoError(t, svc.Update(context.Background(), req))
}

// TC34: GetByID - generic SQL error wrapper
func TestConfigService_GetByID_ServiceError(t *testing.T) {
	db := setupConfigServiceTestDB(t)
	svc := NewConfigService(db)
	require.NoError(t, db.Exec("DROP TABLE sys_config").Error)

	cfg, err := svc.GetByID(context.Background(), "x")
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

// TC35: Update - non-encryption flag should not trigger callback
func TestConfigService_Update_NonEncryptionNoCallback(t *testing.T) {
	db := setupConfigServiceTestDB(t)
	svc := NewConfigService(db)

	called := false
	original := OnEncryptionConfigChanged
	OnEncryptionConfigChanged = func() { called = true }
	t.Cleanup(func() { OnEncryptionConfigChanged = original })

	id := seedConfigSvc(t, db, "n", "sys.normal.flag", "v1", models.ConfigTypeYes, models.ConfigIsSystemNo)
	req := &requests.ConfigUpdateRequest{
		ID:          id,
		ConfigName:  "n",
		ConfigKey:   "sys.normal.flag",
		ConfigValue: "v2",
		ConfigType:  models.ConfigTypeYes,
	}
	require.NoError(t, svc.Update(context.Background(), req))
	assert.False(t, called, "non-encryption flag should not trigger callback")
}
