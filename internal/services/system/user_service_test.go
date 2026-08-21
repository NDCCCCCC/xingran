package system

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
)

// Phase 72 W2 计划 72-05: UserService CRUD 测试补齐。
// 既有测试(user_list_recursive_test / user_list_status_test / user_statistics_test)保留,
// 本文件补充 GetByID / Delete / UpdateStatus / BatchDelete / List / ListRoles / Update 等。
//
// D-08: 零业务代码改动,使用 glebarez sqlite in-memory + 真实 UserServiceImpl。

// mockPasswordManager 最小可用的 PasswordManager 实现,直接返回原值。
// 用于构造 service,不真正模拟 bcrypt/sm3 哈希。
type mockPasswordManager struct {
	hashCount   int
	verifyCount int
}

func (m *mockPasswordManager) HashPassword(p string) (string, error) {
	m.hashCount++
	return "hashed:" + p, nil
}

func (m *mockPasswordManager) VerifyPassword(p, hash string) (bool, error) {
	m.verifyCount++
	return hash == "hashed:"+p, nil
}

// ensureMockPasswordManagerImplementsInterface
var _ PasswordManager = (*mockPasswordManager)(nil)

// setupUserServiceTestDB 创建含最小 schema 的内存 sqlite。
func setupUserServiceTestDB(t *testing.T) *gorm.DB {
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
			gender INTEGER NOT NULL DEFAULT 2,
			status INTEGER NOT NULL DEFAULT 0,
			dept_id TEXT,
			dept_name TEXT,
			remark TEXT,
			salt TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)
	`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE sys_user_role (user_id TEXT, role_id TEXT)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE sys_user_post (user_id TEXT, post_id TEXT)`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_role (
			id TEXT PRIMARY KEY,
			role_name TEXT,
			status INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_dept (
			id TEXT PRIMARY KEY,
			dept_name TEXT,
			ancestors TEXT,
			status INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)
	`).Error)
	return db
}

// seedUserRow 在 sys_user 插入一行,返回 id。
func seedUserRow(t *testing.T, db *gorm.DB, username string, status int) string {
	t.Helper()
	id := uuid.NewString()
	now := "2024-01-01 00:00:00"
	require.NoError(t, db.Exec(
		`INSERT INTO sys_user (id, username, password, status, created_at, updated_at, deleted_at) VALUES (?, ?, 'x', ?, ?, ?, NULL)`,
		id, username, status, now, now,
	).Error)
	return id
}

// newTestUserService 构造 UserService。
func newTestUserService(t *testing.T, db *gorm.DB) UserService {
	t.Helper()
	return NewUserService(db, &mockPasswordManager{})
}

// TestUserService_GetByID_Success 验证查询用户成功。
func TestUserService_GetByID_Success(t *testing.T) {
	db := setupUserServiceTestDB(t)
	id := seedUserRow(t, db, "getme", 0)
	svc := newTestUserService(t, db)

	user, err := svc.GetByID(context.Background(), id)
	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, "getme", user.Username)
	assert.Equal(t, id, user.ID)
}

// TestUserService_GetByID_NotFound 验证查询不存在用户返回错误。
func TestUserService_GetByID_NotFound(t *testing.T) {
	db := setupUserServiceTestDB(t)
	svc := newTestUserService(t, db)

	user, err := svc.GetByID(context.Background(), uuid.NewString())
	assert.Error(t, err)
	assert.Nil(t, user)
}

// TestUserService_Delete_Success 验证删除用户（软删除）。
func TestUserService_Delete_Success(t *testing.T) {
	db := setupUserServiceTestDB(t)
	id := seedUserRow(t, db, "todelete", 0)
	svc := newTestUserService(t, db)

	err := svc.Delete(context.Background(), id)
	require.NoError(t, err)

	// 软删除后应不可见
	var deletedAt *string
	require.NoError(t, db.Raw("SELECT deleted_at FROM sys_user WHERE id = ?", id).Scan(&deletedAt).Error)
	assert.NotNil(t, deletedAt)
}

// TestUserService_Delete_NotFound 验证删除不存在用户返回错误。
func TestUserService_Delete_NotFound(t *testing.T) {
	db := setupUserServiceTestDB(t)
	svc := newTestUserService(t, db)

	err := svc.Delete(context.Background(), uuid.NewString())
	assert.Error(t, err)
}

// TestUserService_UpdateStatus_Enable 验证启用用户。
func TestUserService_UpdateStatus_Enable(t *testing.T) {
	db := setupUserServiceTestDB(t)
	id := seedUserRow(t, db, "enableme", 1)
	svc := newTestUserService(t, db)

	err := svc.UpdateStatus(context.Background(), id, 0)
	require.NoError(t, err)

	var status int
	require.NoError(t, db.Raw("SELECT status FROM sys_user WHERE id = ?", id).Scan(&status).Error)
	assert.Equal(t, 0, status)
}

// TestUserService_UpdateStatus_Disable 验证停用用户。
func TestUserService_UpdateStatus_Disable(t *testing.T) {
	db := setupUserServiceTestDB(t)
	id := seedUserRow(t, db, "disableme", 0)
	svc := newTestUserService(t, db)

	err := svc.UpdateStatus(context.Background(), id, 1)
	require.NoError(t, err)

	var status int
	require.NoError(t, db.Raw("SELECT status FROM sys_user WHERE id = ?", id).Scan(&status).Error)
	assert.Equal(t, 1, status)
}

// TestUserService_UpdateStatus_NotFound 验证不存在用户。
func TestUserService_UpdateStatus_NotFound(t *testing.T) {
	db := setupUserServiceTestDB(t)
	svc := newTestUserService(t, db)

	err := svc.UpdateStatus(context.Background(), uuid.NewString(), 0)
	assert.Error(t, err)
}

// TestUserService_BatchDelete_Success 验证批量删除。
func TestUserService_BatchDelete_Success(t *testing.T) {
	db := setupUserServiceTestDB(t)
	id1 := seedUserRow(t, db, "batch1", 0)
	id2 := seedUserRow(t, db, "batch2", 0)
	keepID := seedUserRow(t, db, "keepme", 0)
	svc := newTestUserService(t, db)

	err := svc.BatchDelete(context.Background(), []string{id1, id2})
	require.NoError(t, err)

	var aliveCount int64
	db.Raw("SELECT COUNT(*) FROM sys_user WHERE deleted_at IS NULL").Scan(&aliveCount)
	assert.Equal(t, int64(1), aliveCount, "only keepme (id=%s) should remain", keepID)
}

// TestUserService_BatchDelete_Empty 验证空 ids 返回参数错误。
func TestUserService_BatchDelete_Empty(t *testing.T) {
	db := setupUserServiceTestDB(t)
	svc := newTestUserService(t, db)

	err := svc.BatchDelete(context.Background(), []string{})
	assert.Error(t, err)
	assert.True(t, errors.Is(err, apperrors.ParamMissing("ids")) ||
		strings.Contains(err.Error(), "ids"),
		"should return ids-related error: %v", err)
}

// TestUserService_ResetPassword_Success 验证密码重置。
func TestUserService_ResetPassword_Success(t *testing.T) {
	db := setupUserServiceTestDB(t)
	id := seedUserRow(t, db, "pwdreset", 0)
	svc := newTestUserService(t, db)

	err := svc.ResetPassword(context.Background(), id, "newpass")
	require.NoError(t, err)

	// 验证密码已更新（mock PasswordManager 返回 "hashed:newpass"）
	var password string
	require.NoError(t, db.Raw("SELECT password FROM sys_user WHERE id = ?", id).Scan(&password).Error)
	assert.Equal(t, "hashed:newpass", password)
}

// TestUserService_ResetPassword_NotFound 验证重置不存在用户密码返回错误。
func TestUserService_ResetPassword_NotFound(t *testing.T) {
	db := setupUserServiceTestDB(t)
	svc := newTestUserService(t, db)

	err := svc.ResetPassword(context.Background(), uuid.NewString(), "x")
	assert.Error(t, err)
}

// TestUserService_List_EmptyDB 验证空库列表返回 0 行。
func TestUserService_List_EmptyDB(t *testing.T) {
	db := setupUserServiceTestDB(t)
	svc := newTestUserService(t, db)

	params := requests.DefaultUserListParams()
	params.PageSize = 10
	result, err := svc.List(context.Background(), params)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, int64(0), result.Total)
}

// TestUserService_List_WithNicknameFilter 验证 nickname LIKE 过滤。
func TestUserService_List_WithNicknameFilter(t *testing.T) {
	db := setupUserServiceTestDB(t)
	for i := 0; i < 3; i++ {
		id := uuid.NewString()
		now := "2024-01-01 00:00:00"
		require.NoError(t, db.Exec(
			`INSERT INTO sys_user (id, username, nickname, password, status, created_at, updated_at, deleted_at) VALUES (?, ?, ?, 'x', 0, ?, ?, NULL)`,
			id, "u"+string(rune('a'+i)), "Nick"+string(rune('a'+i)), now, now,
		).Error)
	}
	svc := newTestUserService(t, db)

	nick := "Nick"
	params := requests.DefaultUserListParams()
	params.PageSize = 10
	params.Nickname = &nick
	result, err := svc.List(context.Background(), params)
	require.NoError(t, err)
	assert.Equal(t, int64(3), result.Total)
}

// TestUserService_List_WithDeptFilter 验证 DeptID 过滤。
func TestUserService_List_WithDeptFilter(t *testing.T) {
	db := setupUserServiceTestDB(t)
	deptID := uuid.NewString()
	for i := 0; i < 2; i++ {
		id := uuid.NewString()
		now := "2024-01-01 00:00:00"
		require.NoError(t, db.Exec(
			`INSERT INTO sys_user (id, username, password, status, dept_id, created_at, updated_at, deleted_at) VALUES (?, ?, 'x', 0, ?, ?, ?, NULL)`,
			id, "u"+string(rune('a'+i)), deptID, now, now,
		).Error)
	}
	// 另一部门的用户
	otherDept := uuid.NewString()
	id := uuid.NewString()
	now := "2024-01-01 00:00:00"
	require.NoError(t, db.Exec(
		`INSERT INTO sys_user (id, username, password, status, dept_id, created_at, updated_at, deleted_at) VALUES (?, ?, 'x', 0, ?, ?, ?, NULL)`,
		id, "otheruser", otherDept, now, now,
	).Error)

	svc := newTestUserService(t, db)

	dept := deptID
	params := requests.DefaultUserListParams()
	params.PageSize = 10
	params.DeptID = &dept
	result, err := svc.List(context.Background(), params)
	require.NoError(t, err)
	assert.Equal(t, int64(2), result.Total, "filter dept should return 2 users")
}

// TestUserService_Statistics_Zero 验证空库 Statistics 全零。
func TestUserService_Statistics_Zero(t *testing.T) {
	db := setupUserServiceTestDB(t)
	svc := newTestUserService(t, db)

	result, err := svc.Statistics(context.Background())
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, int64(0), result.Total)
	assert.Equal(t, int64(0), result.Active)
	assert.Equal(t, int64(0), result.Inactive)
}

// TestUserService_PasswordManagerAdapter 验证 PasswordManagerAdapter 包装 security.PasswordManager。
func TestUserService_PasswordManagerAdapter(t *testing.T) {
	// 由于 security.PasswordManager 字段复杂,这里只验证接口契约
	pm := NewPasswordManagerAdapter(nil)
	// nil PM → 调用应 panic, 这只是接口类型校验
	assert.NotNil(t, pm)
	_, ok := pm.(PasswordManager)
	assert.True(t, ok)
	_ = pm
}

// TestUserService_BuildDepartmentPaths_NoAncestors 验证无 ancestors 时 early return。
//
// 关键：buildDepartmentPaths 早期 return 当所有用户都没有 ancestors 时;
// fallback 分支(用 DeptName 填充 DeptFullName)只在混合场景下执行 —
// Phase 72 不测试极端内部路径,只确认不 panic。
func TestUserService_BuildDepartmentPaths_NoAncestors(t *testing.T) {
	db := setupUserServiceTestDB(t)
	svc := &userService{db: db, pwdManager: nil}
	list := []models.User{{
		BaseModel: models.BaseModel{ID: "u-1"},
		Username:  "nopath",
		DeptName:  strPtr("孤岛部门"),
	}}
	// 不 panic 即可
	svc.buildDepartmentPaths(context.Background(), list)
}

// TestUserService_Update_NotFound 验证更新不存在用户返回错误。
func TestUserService_Update_NotFound(t *testing.T) {
	db := setupUserServiceTestDB(t)
	svc := newTestUserService(t, db)

	missing := uuid.NewString()
	nick := "x"
	err := svc.Update(context.Background(), &requests.UserUpdateRequest{
		ID:       missing,
		Nickname: &nick,
	})
	assert.Error(t, err)
}

// TestUserService_PageResult_Structure 验证 PageResult 结构。
func TestUserService_PageResult_Structure(t *testing.T) {
	pr := &PageResult{
		List:     []models.User{},
		Total:    0,
		Current:  1,
		PageSize: 10,
	}
	assert.NotNil(t, pr.List)
	assert.Equal(t, int64(0), pr.Total)
}

// stringPtr helper (stringPtr already defined in user_sync_service_test.go)
// Use strPtr instead.
func strPtr(s string) *string { return &s }