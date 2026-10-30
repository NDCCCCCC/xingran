package system

// =====================================================================
// email_config_service_test.go — covers email_config_service.go (282 lines)
//
// Per Plan 72-12 Task 3.
// Uses glebarez sqlite in-memory + real EmailConfigService + real SM4
// cipher (D-03 — no mock cipher). Password is encrypted on Create/Update
// and decrypted on read via services.EncryptPassword (real cipher).
//
// Per CLAUDE.md: status 0=enabled, 1=disabled. Email config follows
// the "only one email config" invariant — most tests seed via raw SQL
// to bypass that single-row check, or test the rejection path directly.
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

// setupEmailConfigTestDB creates in-memory SQLite with sys_email_config schema.
func setupEmailConfigTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_email_config (
			id TEXT PRIMARY KEY,
			config_name TEXT NOT NULL,
			host TEXT NOT NULL,
			port INTEGER NOT NULL,
			username TEXT NOT NULL,
			password TEXT NOT NULL,
			from_name TEXT,
			from_email TEXT,
			use_ssl INTEGER DEFAULT 1,
			use_start_tls INTEGER DEFAULT 1,
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

// seedEmailConfigRaw inserts a sys_email_config row directly (bypasses
// the single-row invariant check used by the service Create path).
func seedEmailConfigRaw(t *testing.T, db *gorm.DB, status int) string {
	t.Helper()
	id := uuid.NewString()
	require.NoError(t, db.Exec(`
		INSERT INTO sys_email_config
		(id, config_name, host, port, username, password, from_email, status, use_ssl, use_start_tls, is_default, del_flag, created_at, updated_at)
		VALUES (?, 'seeded', 'smtp.example.com', 587, 'user@example.com', 'plain-pwd', 'from@example.com', ?, 1, 1, 0, 0, datetime('now'), datetime('now'))`,
		id, status).Error)
	return id
}

// TC1: Create — success encrypts password via real SM4 cipher.
func TestEmailConfigService_Create_Success(t *testing.T) {
	db := setupEmailConfigTestDB(t)
	svc := NewEmailConfigService(db)

	plain := "RealSecretPwd_123"
	req := &EmailConfigCreateRequest{
		ConfigName: "smtp-primary",
		Host:       "smtp.example.com",
		Port:       587,
		Username:   "user@example.com",
		Password:   plain,
		FromName:   "Sender",
		FromEmail:  "from@example.com",
		UseSSL:     true,
		Status:     0,
	}

	require.NoError(t, svc.Create(context.Background(), req))

	var stored models.EmailConfig
	require.NoError(t, db.First(&stored).Error)
	assert.NotEqual(t, plain, stored.Password, "password should be encrypted at rest")
	assert.NotEmpty(t, stored.Password, "password column should be populated")
	assert.True(t, stored.IsDefault, "first/only config is auto-default")
}

// TC2: Create — rejects duplicate (single-row invariant).
func TestEmailConfigService_Create_RejectsDuplicate(t *testing.T) {
	db := setupEmailConfigTestDB(t)
	svc := NewEmailConfigService(db)
	seedEmailConfigRaw(t, db, 0)

	req := &EmailConfigCreateRequest{
		ConfigName: "second",
		Host:       "smtp.example.com",
		Port:       587,
		Username:   "user2@example.com",
		Password:   "pwd",
		Status:     0,
	}
	err := svc.Create(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "邮件配置已存在")
}

// TC3: GetByID — returns DTO with raw (still encrypted) password.
func TestEmailConfigService_GetByID_Success(t *testing.T) {
	db := setupEmailConfigTestDB(t)
	svc := NewEmailConfigService(db)
	id := seedEmailConfigRaw(t, db, 0)

	dto, err := svc.GetByID(context.Background(), id)
	require.NoError(t, err)
	require.NotNil(t, dto)
	assert.Equal(t, id, dto.ID)
	assert.Equal(t, "smtp.example.com", dto.Host)
	assert.Equal(t, 587, dto.Port)
	assert.NotEmpty(t, dto.Password)
}

// TC4: GetByID — not found returns error.
func TestEmailConfigService_GetByID_NotFound(t *testing.T) {
	db := setupEmailConfigTestDB(t)
	svc := NewEmailConfigService(db)

	_, err := svc.GetByID(context.Background(), uuid.NewString())
	assert.Error(t, err)
}

// TC5: Update — changes name + port, password unchanged when nil.
func TestEmailConfigService_Update_PartialNoPassword(t *testing.T) {
	db := setupEmailConfigTestDB(t)
	svc := NewEmailConfigService(db)
	id := seedEmailConfigRaw(t, db, 0)

	// Snapshot original password from the DB.
	var snap struct {
		Password string
	}
	require.NoError(t, db.Raw("SELECT password FROM sys_email_config WHERE id = ?", id).Scan(&snap).Error)
	originalPassword := snap.Password

	newName := "renamed"
	newPort := 465
	err := svc.Update(context.Background(), &EmailConfigUpdateRequest{
		ID:         id,
		ConfigName: &newName,
		Port:       &newPort,
	})
	require.NoError(t, err)

	var stored models.EmailConfig
	require.NoError(t, db.First(&stored, "id = ?", id).Error)
	assert.Equal(t, "renamed", stored.ConfigName)
	assert.Equal(t, 465, stored.Port)
	assert.Equal(t, originalPassword, stored.Password, "password must remain unchanged when not in update payload")
}

// TC6: Update — encrypts new password when provided.
func TestEmailConfigService_Update_NewPasswordEncrypted(t *testing.T) {
	db := setupEmailConfigTestDB(t)
	svc := NewEmailConfigService(db)
	id := seedEmailConfigRaw(t, db, 0)

	// Snapshot original password from the DB.
	var snap struct {
		Password string
	}
	require.NoError(t, db.Raw("SELECT password FROM sys_email_config WHERE id = ?", id).Scan(&snap).Error)
	originalPassword := snap.Password

	newPwd := "NewSecret_456"
	err := svc.Update(context.Background(), &EmailConfigUpdateRequest{
		ID:       id,
		Password: &newPwd,
	})
	require.NoError(t, err)

	var stored models.EmailConfig
	require.NoError(t, db.First(&stored, "id = ?", id).Error)
	assert.NotEqual(t, originalPassword, stored.Password, "password should be re-encrypted")
	assert.NotEqual(t, newPwd, stored.Password, "stored password should not be plaintext")
}

// TC7: Update — not found returns error.
func TestEmailConfigService_Update_NotFound(t *testing.T) {
	db := setupEmailConfigTestDB(t)
	svc := NewEmailConfigService(db)

	newName := "x"
	err := svc.Update(context.Background(), &EmailConfigUpdateRequest{
		ID:         uuid.NewString(),
		ConfigName: &newName,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "邮箱配置不存在")
}

// TC8: Delete — soft-deletes by setting del_flag=1.
func TestEmailConfigService_Delete_Success(t *testing.T) {
	db := setupEmailConfigTestDB(t)
	svc := NewEmailConfigService(db)
	id := seedEmailConfigRaw(t, db, 0)

	require.NoError(t, svc.Delete(context.Background(), id))

	var stored models.EmailConfig
	require.NoError(t, db.First(&stored, "id = ?", id).Error)
	assert.Equal(t, 1, stored.DelFlag)
}

// TC9: Delete — not found returns error.
func TestEmailConfigService_Delete_NotFound(t *testing.T) {
	db := setupEmailConfigTestDB(t)
	svc := NewEmailConfigService(db)

	err := svc.Delete(context.Background(), uuid.NewString())
	assert.Error(t, err)
}

// TC10: List — returns all rows by default.
func TestEmailConfigService_List_All(t *testing.T) {
	db := setupEmailConfigTestDB(t)
	svc := NewEmailConfigService(db)
	id1 := seedEmailConfigRaw(t, db, 0)
	id2 := seedEmailConfigRaw(t, db, 1)
	_ = id1
	_ = id2

	res, err := svc.List(context.Background(), DefaultEmailConfigListParams())
	require.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, int64(2), res.Total)
}

// TC11: List — filtered by status.
func TestEmailConfigService_List_FilterByStatus(t *testing.T) {
	db := setupEmailConfigTestDB(t)
	svc := NewEmailConfigService(db)
	seedEmailConfigRaw(t, db, 0)
	seedEmailConfigRaw(t, db, 1)
	seedEmailConfigRaw(t, db, 1)

	enabled := 0
	res, err := svc.List(context.Background(), EmailConfigListParams{
		Status:   &enabled,
		Current:  1,
		PageSize: 10,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), res.Total)
}

// TC12: List — excludes soft-deleted rows.
func TestEmailConfigService_List_ExcludesSoftDeleted(t *testing.T) {
	db := setupEmailConfigTestDB(t)
	svc := NewEmailConfigService(db)
	id := seedEmailConfigRaw(t, db, 0)
	require.NoError(t, svc.Delete(context.Background(), id))
	seedEmailConfigRaw(t, db, 0)

	res, err := svc.List(context.Background(), DefaultEmailConfigListParams())
	require.NoError(t, err)
	assert.Equal(t, int64(1), res.Total)
}

// TC13: List — pagination honored.
func TestEmailConfigService_List_Pagination(t *testing.T) {
	db := setupEmailConfigTestDB(t)
	svc := NewEmailConfigService(db)
	for i := 0; i < 5; i++ {
		seedEmailConfigRaw(t, db, 0)
	}

	res, err := svc.List(context.Background(), EmailConfigListParams{Current: 2, PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, int64(5), res.Total)
	assert.Equal(t, 2, res.Current)
	assert.Equal(t, 2, res.PageSize)
}

// TC14: toEmailConfigDTO — maps all fields correctly.
func TestEmailConfigService_ToEmailConfigDTO_AllFields(t *testing.T) {
	id := uuid.NewString()
	m := &models.EmailConfig{
		ID:          id,
		ConfigName:  "x",
		Host:        "smtp.x.com",
		Port:        587,
		Username:    "u",
		Password:    "p",
		FromName:    "fn",
		FromEmail:   "from@x.com",
		UseSSL:      true,
		UseSTARTTLS: true,
		IsDefault:   true,
		Status:      0,
		Remark:      "r",
	}

	dto := toEmailConfigDTO(m)
	require.NotNil(t, dto)
	assert.Equal(t, id, dto.ID)
	assert.Equal(t, "x", dto.ConfigName)
	assert.Equal(t, "smtp.x.com", dto.Host)
	assert.Equal(t, 587, dto.Port)
	assert.Equal(t, "u", dto.Username)
	assert.Equal(t, "p", dto.Password)
	assert.Equal(t, "fn", dto.FromName)
	assert.Equal(t, "from@x.com", dto.FromEmail)
	assert.True(t, dto.UseSSL)
	assert.True(t, dto.UseSTARTTLS)
	assert.True(t, dto.IsDefault)
	assert.Equal(t, 0, dto.Status)
	assert.Equal(t, "r", dto.Remark)
}

// TC15: DefaultEmailConfigListParams — sane defaults.
func TestEmailConfigService_DefaultEmailConfigListParams(t *testing.T) {
	p := DefaultEmailConfigListParams()
	assert.Equal(t, 1, p.Current)
	assert.Equal(t, 10, p.PageSize)
	assert.Nil(t, p.Status)
}

// TC16: Update — sets use_ssl field.
func TestEmailConfigService_Update_UseSSL(t *testing.T) {
	db := setupEmailConfigTestDB(t)
	svc := NewEmailConfigService(db)
	id := seedEmailConfigRaw(t, db, 0)

	useSSL := false
	require.NoError(t, svc.Update(context.Background(), &EmailConfigUpdateRequest{
		ID:     id,
		UseSSL: &useSSL,
	}))

	var stored models.EmailConfig
	require.NoError(t, db.First(&stored, "id = ?", id).Error)
	assert.False(t, stored.UseSSL)
}

// TC17: Update — sets status field.
func TestEmailConfigService_Update_Status(t *testing.T) {
	db := setupEmailConfigTestDB(t)
	svc := NewEmailConfigService(db)
	id := seedEmailConfigRaw(t, db, 0)

	status := 1
	require.NoError(t, svc.Update(context.Background(), &EmailConfigUpdateRequest{
		ID:     id,
		Status: &status,
	}))

	var stored models.EmailConfig
	require.NoError(t, db.First(&stored, "id = ?", id).Error)
	assert.Equal(t, 1, stored.Status)
}

// TC18: Update — sets remark field.
func TestEmailConfigService_Update_Remark(t *testing.T) {
	db := setupEmailConfigTestDB(t)
	svc := NewEmailConfigService(db)
	id := seedEmailConfigRaw(t, db, 0)

	r := "production smtp"
	require.NoError(t, svc.Update(context.Background(), &EmailConfigUpdateRequest{
		ID:     id,
		Remark: &r,
	}))

	var stored models.EmailConfig
	require.NoError(t, db.First(&stored, "id = ?", id).Error)
	assert.Equal(t, "production smtp", stored.Remark)
}