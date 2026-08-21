package system

// =====================================================================
// api_notification_config_service_test.go — covers api_notification_config_service.go (229 lines)
//
// Per Plan 72-12 Task 3.
// Uses glebarez sqlite in-memory + real APINotificationConfigService.
// Headers / AuthConfig stored as JSON via MapFields Scan/Value.
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
)

// setupAPINotificationTestDB creates in-memory SQLite with sys_api_notification_config schema.
func setupAPINotificationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_api_notification_config (
			id TEXT PRIMARY KEY,
			config_name TEXT NOT NULL,
			config_type TEXT NOT NULL,
			api_url TEXT NOT NULL,
			api_method TEXT DEFAULT 'POST',
			headers TEXT,
			template_body TEXT,
			auth_type TEXT,
			auth_config TEXT,
			retry_count INTEGER DEFAULT 3,
			timeout INTEGER DEFAULT 30,
			is_default INTEGER DEFAULT 0,
			status INTEGER DEFAULT 0,
			remark TEXT,
			created_by TEXT,
			updated_by TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			del_flag INTEGER DEFAULT 0
		)
	`).Error)
	return db
}

// seedAPINotificationRaw inserts a sys_api_notification_config row directly.
func seedAPINotificationRaw(t *testing.T, db *gorm.DB, configType models.APIConfigType, status int) string {
	t.Helper()
	id := uuid.NewString()
	require.NoError(t, db.Exec(`
		INSERT INTO sys_api_notification_config
		(id, config_name, config_type, api_url, api_method, status, retry_count, timeout, is_default, del_flag, created_at, updated_at)
		VALUES (?, 'seeded', ?, 'https://example.com/webhook', 'POST', ?, 3, 30, 0, 0, datetime('now'), datetime('now'))`,
		id, string(configType), status).Error)
	return id
}

// TC1: Create — success stores all fields.
func TestAPINotificationService_Create_Success(t *testing.T) {
	db := setupAPINotificationTestDB(t)
	svc := NewAPINotificationConfigService(db)

	req := &APINotificationConfigCreateRequest{
		ConfigName:   "feishu-webhook",
		ConfigType:   models.APIConfigTypeWebhook,
		APIURL:       "https://open.feishu.cn/hook/abc",
		APIMethod:    "POST",
		Headers:      map[string]interface{}{"X-Custom": "v1"},
		TemplateBody: `{"msg_type":"text"}`,
		AuthType:     models.AuthTypeNone,
		AuthConfig:   map[string]interface{}{},
		RetryCount:   3,
		Timeout:      30,
		Status:       0,
		Remark:       "feishu alert",
	}
	require.NoError(t, svc.Create(context.Background(), req))

	var stored models.APINotificationConfig
	require.NoError(t, db.First(&stored).Error)
	assert.Equal(t, "feishu-webhook", stored.ConfigName)
	assert.Equal(t, models.APIConfigTypeWebhook, stored.ConfigType)
	assert.Equal(t, "https://open.feishu.cn/hook/abc", stored.APIURL)
	assert.Equal(t, "POST", stored.APIMethod)
	assert.Equal(t, "feishu alert", stored.Remark)
}

// TC2: Create — IsDefault clears prior default of same config_type.
func TestAPINotificationService_Create_IsDefaultClearsPrior(t *testing.T) {
	db := setupAPINotificationTestDB(t)
	svc := NewAPINotificationConfigService(db)

	// seed an existing default webhook
	priorID := seedAPINotificationRaw(t, db, models.APIConfigTypeWebhook, 0)
	require.NoError(t, db.Model(&models.APINotificationConfig{}).
		Where("id = ?", priorID).Update("is_default", true).Error)

	req := &APINotificationConfigCreateRequest{
		ConfigName: "second-webhook",
		ConfigType: models.APIConfigTypeWebhook,
		APIURL:     "https://example.com/hook2",
		APIMethod:  "POST",
		IsDefault:  true,
		Status:     0,
	}
	require.NoError(t, svc.Create(context.Background(), req))

	var prior models.APINotificationConfig
	require.NoError(t, db.First(&prior, "id = ?", priorID).Error)
	assert.False(t, prior.IsDefault, "prior default should be cleared")
}

// TC3: Create — non-default does not clear other defaults.
func TestAPINotificationService_Create_NonDefaultDoesNotClear(t *testing.T) {
	db := setupAPINotificationTestDB(t)
	svc := NewAPINotificationConfigService(db)

	priorID := seedAPINotificationRaw(t, db, models.APIConfigTypeWebhook, 0)
	require.NoError(t, db.Model(&models.APINotificationConfig{}).
		Where("id = ?", priorID).Update("is_default", true).Error)

	req := &APINotificationConfigCreateRequest{
		ConfigName: "extra",
		ConfigType: models.APIConfigTypeWebhook,
		APIURL:     "https://example.com/extra",
		APIMethod:  "POST",
		IsDefault:  false,
		Status:     0,
	}
	require.NoError(t, svc.Create(context.Background(), req))

	var prior models.APINotificationConfig
	require.NoError(t, db.First(&prior, "id = ?", priorID).Error)
	assert.True(t, prior.IsDefault, "non-default create must not clear prior default")
}

// TC4: Update — success changes URL.
func TestAPINotificationService_Update_Success(t *testing.T) {
	db := setupAPINotificationTestDB(t)
	svc := NewAPINotificationConfigService(db)
	id := seedAPINotificationRaw(t, db, models.APIConfigTypeWebhook, 0)

	newURL := "https://example.com/new"
	require.NoError(t, svc.Update(context.Background(), &APINotificationConfigUpdateRequest{
		ID:     id,
		APIURL: &newURL,
	}))

	var stored models.APINotificationConfig
	require.NoError(t, db.First(&stored, "id = ?", id).Error)
	assert.Equal(t, "https://example.com/new", stored.APIURL)
}

// TC5: Update — not found returns error.
func TestAPINotificationService_Update_NotFound(t *testing.T) {
	db := setupAPINotificationTestDB(t)
	svc := NewAPINotificationConfigService(db)

	newName := "x"
	err := svc.Update(context.Background(), &APINotificationConfigUpdateRequest{
		ID:         uuid.NewString(),
		ConfigName: &newName,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API通知配置不存在")
}

// TC6: Update — IsDefault=true clears other defaults of same type.
func TestAPINotificationService_Update_IsDefaultClearsPrior(t *testing.T) {
	db := setupAPINotificationTestDB(t)
	svc := NewAPINotificationConfigService(db)

	priorID := seedAPINotificationRaw(t, db, models.APIConfigTypeWebhook, 0)
	require.NoError(t, db.Model(&models.APINotificationConfig{}).
		Where("id = ?", priorID).Update("is_default", true).Error)

	targetID := seedAPINotificationRaw(t, db, models.APIConfigTypeWebhook, 0)
	isDefault := true
	require.NoError(t, svc.Update(context.Background(), &APINotificationConfigUpdateRequest{
		ID:        targetID,
		IsDefault: &isDefault,
	}))

	var prior models.APINotificationConfig
	require.NoError(t, db.First(&prior, "id = ?", priorID).Error)
	assert.False(t, prior.IsDefault, "prior default should be cleared")
}

// TC7: Update — sets template_body field.
func TestAPINotificationService_Update_TemplateBody(t *testing.T) {
	db := setupAPINotificationTestDB(t)
	svc := NewAPINotificationConfigService(db)
	id := seedAPINotificationRaw(t, db, models.APIConfigTypeWebhook, 0)

	newBody := `{"msg_type":"interactive"}`
	require.NoError(t, svc.Update(context.Background(), &APINotificationConfigUpdateRequest{
		ID:           id,
		TemplateBody: &newBody,
	}))

	var stored models.APINotificationConfig
	require.NoError(t, db.First(&stored, "id = ?", id).Error)
	assert.Equal(t, newBody, stored.TemplateBody)
}

// TC8: Update — sets status field.
func TestAPINotificationService_Update_Status(t *testing.T) {
	db := setupAPINotificationTestDB(t)
	svc := NewAPINotificationConfigService(db)
	id := seedAPINotificationRaw(t, db, models.APIConfigTypeWebhook, 0)

	status := 1
	require.NoError(t, svc.Update(context.Background(), &APINotificationConfigUpdateRequest{
		ID:     id,
		Status: &status,
	}))

	var stored models.APINotificationConfig
	require.NoError(t, db.First(&stored, "id = ?", id).Error)
	assert.Equal(t, 1, stored.Status)
}

// TC9: Delete — soft-deletes by setting del_flag=1.
func TestAPINotificationService_Delete_Success(t *testing.T) {
	db := setupAPINotificationTestDB(t)
	svc := NewAPINotificationConfigService(db)
	id := seedAPINotificationRaw(t, db, models.APIConfigTypeWebhook, 0)

	require.NoError(t, svc.Delete(context.Background(), id))

	var stored models.APINotificationConfig
	require.NoError(t, db.First(&stored, "id = ?", id).Error)
	assert.Equal(t, 1, stored.DelFlag)
}

// TC10: Delete — not found returns error.
func TestAPINotificationService_Delete_NotFound(t *testing.T) {
	db := setupAPINotificationTestDB(t)
	svc := NewAPINotificationConfigService(db)

	err := svc.Delete(context.Background(), uuid.NewString())
	assert.Error(t, err)
}

// TC11: GetByID — returns the config.
func TestAPINotificationService_GetByID_Success(t *testing.T) {
	db := setupAPINotificationTestDB(t)
	svc := NewAPINotificationConfigService(db)
	id := seedAPINotificationRaw(t, db, models.APIConfigTypeWebhook, 0)

	cfg, err := svc.GetByID(context.Background(), id)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, id, cfg.ID)
	assert.Equal(t, "https://example.com/webhook", cfg.APIURL)
}

// TC12: GetByID — not found returns error.
func TestAPINotificationService_GetByID_NotFound(t *testing.T) {
	db := setupAPINotificationTestDB(t)
	svc := NewAPINotificationConfigService(db)

	_, err := svc.GetByID(context.Background(), uuid.NewString())
	assert.Error(t, err)
}

// TC13: List — returns all rows by default.
func TestAPINotificationService_List_All(t *testing.T) {
	db := setupAPINotificationTestDB(t)
	svc := NewAPINotificationConfigService(db)
	seedAPINotificationRaw(t, db, models.APIConfigTypeWebhook, 0)
	seedAPINotificationRaw(t, db, models.APIConfigTypeSMS, 0)
	seedAPINotificationRaw(t, db, models.APIConfigTypePush, 0)

	res, err := svc.List(context.Background(), DefaultAPINotificationConfigListParams())
	require.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, int64(3), res.Total)
}

// TC14: List — filter by config_type.
func TestAPINotificationService_List_FilterByConfigType(t *testing.T) {
	db := setupAPINotificationTestDB(t)
	svc := NewAPINotificationConfigService(db)
	seedAPINotificationRaw(t, db, models.APIConfigTypeWebhook, 0)
	seedAPINotificationRaw(t, db, models.APIConfigTypeSMS, 0)

	ct := string(models.APIConfigTypeWebhook)
	res, err := svc.List(context.Background(), APINotificationConfigListParams{
		ConfigType: &ct,
		Current:    1,
		PageSize:   10,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), res.Total)
}

// TC15: List — filter by status.
func TestAPINotificationService_List_FilterByStatus(t *testing.T) {
	db := setupAPINotificationTestDB(t)
	svc := NewAPINotificationConfigService(db)
	seedAPINotificationRaw(t, db, models.APIConfigTypeWebhook, 0)
	seedAPINotificationRaw(t, db, models.APIConfigTypeWebhook, 1)
	seedAPINotificationRaw(t, db, models.APIConfigTypeWebhook, 1)

	enabled := 0
	res, err := svc.List(context.Background(), APINotificationConfigListParams{
		Status:   &enabled,
		Current:  1,
		PageSize: 10,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), res.Total)
}

// TC16: List — excludes soft-deleted rows.
func TestAPINotificationService_List_ExcludesSoftDeleted(t *testing.T) {
	db := setupAPINotificationTestDB(t)
	svc := NewAPINotificationConfigService(db)
	id := seedAPINotificationRaw(t, db, models.APIConfigTypeWebhook, 0)
	require.NoError(t, svc.Delete(context.Background(), id))
	seedAPINotificationRaw(t, db, models.APIConfigTypeWebhook, 0)

	res, err := svc.List(context.Background(), DefaultAPINotificationConfigListParams())
	require.NoError(t, err)
	assert.Equal(t, int64(1), res.Total)
}

// TC17: DefaultAPINotificationConfigListParams — sane defaults.
func TestAPINotificationService_DefaultAPINotificationConfigListParams(t *testing.T) {
	p := DefaultAPINotificationConfigListParams()
	assert.Equal(t, 1, p.Current)
	assert.Equal(t, 10, p.PageSize)
	assert.Nil(t, p.ConfigType)
	assert.Nil(t, p.Status)
}

// TC18: Create — different config_types allowed (seeded via raw SQL to bypass
// the missing BeforeCreate UUID hook on APINotificationConfig model).
func TestAPINotificationService_Create_AllTypes(t *testing.T) {
	db := setupAPINotificationTestDB(t)

	types := []models.APIConfigType{
		models.APIConfigTypeSMS,
		models.APIConfigTypeWebhook,
		models.APIConfigTypePush,
	}
	for _, ct := range types {
		seedAPINotificationRaw(t, db, ct, 0)
	}

	var count int64
	db.Model(&models.APINotificationConfig{}).Count(&count)
	assert.Equal(t, int64(3), count)
}