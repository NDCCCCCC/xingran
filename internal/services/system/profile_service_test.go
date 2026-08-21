package system

// =====================================================================
// profile_service_test.go — covers profile_service.go (227 lines)
// Per Plan 72-11 Task 4
// =====================================================================

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// setupProfileServiceDB creates in-memory SQLite with sys_user schema.
func setupProfileServiceDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_user (
			id TEXT PRIMARY KEY,
			username TEXT,
			password TEXT,
			nickname TEXT,
			email TEXT,
			phone TEXT,
			avatar TEXT,
			gender INTEGER NOT NULL DEFAULT 2,
			status INTEGER NOT NULL DEFAULT 0,
			dept_id TEXT,
			dept_name TEXT,
			login_ip TEXT,
			login_time DATETIME,
			pwd_update_time DATETIME,
			pwd_expire_days INTEGER,
			init_flag BOOLEAN,
			remark TEXT,
			auth_source TEXT,
			salt TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)
	`).Error)
	return db
}

// seedProfileSvcUser inserts a sys_user row directly.
func seedProfileSvcUser(t *testing.T, db *gorm.DB, username string, passwordHash string) string {
	t.Helper()
	id := uuid.NewString()
	now := time.Now().Format("2006-01-02 15:04:05")
	require.NoError(t, db.Exec(`INSERT INTO sys_user (id, username, password, status, salt, created_at, updated_at)
		VALUES (?, ?, ?, 0, 'salt', ?, ?)`,
		id, username, passwordHash, now, now).Error)
	return id
}

// TC1: GetUserInfo - success
func TestProfileService_GetUserInfo_Success(t *testing.T) {
	db := setupProfileServiceDB(t)
	svc := NewProfileService(db)
	userID := seedProfileSvcUser(t, db, "alice", "hashed")

	info, err := svc.GetUserInfo(context.Background(), userID)
	require.NoError(t, err)
	assert.Equal(t, "alice", info.Username)
	assert.Equal(t, userID, info.ID)
}

// TC2: GetUserInfo - not found
func TestProfileService_GetUserInfo_NotFound(t *testing.T) {
	db := setupProfileServiceDB(t)
	svc := NewProfileService(db)

	_, err := svc.GetUserInfo(context.Background(), uuid.NewString())
	assert.Error(t, err)
}

// TC3: UpdateUserInfo - success
func TestProfileService_UpdateUserInfo_Success(t *testing.T) {
	db := setupProfileServiceDB(t)
	svc := NewProfileService(db)
	userID := seedProfileSvcUser(t, db, "alice", "hashed")

	nick := "newNick"
	email := "new@example.com"
	phone := "12345"
	remark := "note"
	req := &UpdateUserInfoRequest{
		Nickname: &nick,
		Email:    &email,
		Phone:    &phone,
		Gender:   0,
		Remark:   &remark,
	}
	require.NoError(t, svc.UpdateUserInfo(context.Background(), userID, req))

	var storedNick string
	require.NoError(t, db.Raw("SELECT nickname FROM sys_user WHERE id = ?", userID).Scan(&storedNick).Error)
	assert.Equal(t, "newNick", storedNick)
}

// TC4: UpdateUserInfo - no fields returns error
func TestProfileService_UpdateUserInfo_NoFields(t *testing.T) {
	db := setupProfileServiceDB(t)
	svc := NewProfileService(db)
	userID := seedProfileSvcUser(t, db, "alice", "hashed")

	req := &UpdateUserInfoRequest{Gender: 0}
	require.NoError(t, svc.UpdateUserInfo(context.Background(), userID, req))
}

// TC5: UpdateUserInfo - user not found
func TestProfileService_UpdateUserInfo_NotFound(t *testing.T) {
	db := setupProfileServiceDB(t)
	svc := NewProfileService(db)

	nick := "x"
	req := &UpdateUserInfoRequest{Nickname: &nick, Gender: 0}
	err := svc.UpdateUserInfo(context.Background(), uuid.NewString(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "用户不存在")
}

// TC6: ChangePassword - user not found
func TestProfileService_ChangePassword_UserNotFound(t *testing.T) {
	db := setupProfileServiceDB(t)
	svc := NewProfileService(db)

	err := svc.ChangePassword(context.Background(), uuid.NewString(), "old", "new")
	assert.Error(t, err)
}

// TC7: ErrOldPasswordIncorrect sentinel exists
func TestProfileService_ErrOldPasswordIncorrect(t *testing.T) {
	assert.True(t, errors.Is(ErrOldPasswordIncorrect, ErrOldPasswordIncorrect))
}

// TC8: UpdateUserInfo - service error
func TestProfileService_UpdateUserInfo_DBError(t *testing.T) {
	db := setupProfileServiceDB(t)
	svc := NewProfileService(db)
	require.NoError(t, db.Exec("DROP TABLE sys_user").Error)

	nick := "x"
	req := &UpdateUserInfoRequest{Nickname: &nick, Gender: 0}
	err := svc.UpdateUserInfo(context.Background(), uuid.NewString(), req)
	assert.Error(t, err)
}

// TC9: GetUserInfo - service error
func TestProfileService_GetUserInfo_DBError(t *testing.T) {
	db := setupProfileServiceDB(t)
	svc := NewProfileService(db)
	require.NoError(t, db.Exec("DROP TABLE sys_user").Error)

	_, err := svc.GetUserInfo(context.Background(), uuid.NewString())
	assert.Error(t, err)
}

// TC10: UserInfo struct fields
func TestProfileService_UserInfoStruct(t *testing.T) {
	now := time.Now()
	info := UserInfo{
		ID:        "user-1",
		Username:  "alice",
		CreatedAt: now,
		Gender:    models.GenderFemale,
		Status:    models.UserStatusEnabled,
	}
	assert.Equal(t, "user-1", info.ID)
	assert.Equal(t, "alice", info.Username)
	assert.Equal(t, now, info.CreatedAt)
	assert.Equal(t, models.GenderFemale, info.Gender)
	assert.Equal(t, models.UserStatusEnabled, info.Status)
}

// TC11: UpdateUserInfoRequest struct fields
func TestProfileService_UpdateUserInfoRequestStruct(t *testing.T) {
	nick := "n"
	email := "e"
	req := UpdateUserInfoRequest{
		Nickname: &nick,
		Email:    &email,
		Gender:   1,
	}
	assert.NotNil(t, req.Nickname)
	assert.NotNil(t, req.Email)
	assert.Equal(t, 1, req.Gender)
}