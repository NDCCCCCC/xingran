package system

import (
	"context"
	"testing"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// setupTestDB 创建测试数据库
func setupTestDBForSync(t *testing.T) *gorm.DB {
	// TODO: 配置测试数据库
	t.Skip("测试数据库配置未实现")
	return nil
}

// TestUserSyncService_SyncUserFromAD_FirstTime 测试首次登录创建用户场景
func TestUserSyncService_SyncUserFromAD_FirstTime(t *testing.T) {
	db := setupTestDBForSync(t)
	if db == nil {
		t.Skip("测试数据库未配置")
	}

	service := NewUserSyncService(db, nil, nil)

	adUser := &ADUserInfoForSync{
		UserDN:      "cn=aduser,dc=test,dc=com",
		Username:    "aduser",
		DisplayName: "AD User",
		Email:       "aduser@example.com",
		Phone:       "1234567890",
	}

	_, _, err := service.SyncUserFromAD(context.Background(), adUser, "default-role-id")

	// 验证结果
	assert.NoError(t, err)

	// 验证用户已创建
	var user models.User
	err = db.Where("username = ?", "aduser").First(&user).Error
	assert.NoError(t, err)
	assert.Equal(t, "AD User", *user.Nickname)
	assert.Equal(t, "aduser@example.com", *user.Email)
	assert.Equal(t, "ad", user.AuthSource)
	assert.Equal(t, "aduser", *user.ADUsername)
}

// TestUserSyncService_SyncUserFromAD_UpdateExisting 测试已存在用户更新信息场景
func TestUserSyncService_SyncUserFromAD_UpdateExisting(t *testing.T) {
	db := setupTestDBForSync(t)
	if db == nil {
		t.Skip("测试数据库未配置")
	}

	service := NewUserSyncService(db, nil, nil)

	// 先创建一个用户
	existingUser := &models.User{
		Username:    "existinguser",
		AuthSource:  "ad",
		ADUsername:  stringPtr("existinguser"),
		Nickname:    stringPtr("Old Nickname"),
		Email:       stringPtr("old@example.com"),
		Status:      models.UserStatusEnabled,
	}
	db.Create(existingUser)

	// 同步新信息
	adUser := &ADUserInfoForSync{
		UserDN:      "cn=existinguser,dc=test,dc=com",
		Username:    "existinguser",
		DisplayName: "Updated User",
		Email:       "new@example.com",
		Phone:       "9876543210",
	}

	_, _, err := service.SyncUserFromAD(context.Background(), adUser, "default-role-id")

	// 验证结果
	assert.NoError(t, err)

	// 验证用户信息已更新
	var user models.User
	err = db.Where("username = ?", "existinguser").First(&user).Error
	assert.NoError(t, err)

	// Email应该更新
	assert.Equal(t, "new@example.com", *user.Email)

	// Nickname如果原来有值，可能保留（取决于业务逻辑）
	// 这里假设nickname会被更新为空时才会更新
}

// TestUserSyncService_SyncUserFromAD_TransactionRollback 测试事务回滚场景
func TestUserSyncService_SyncUserFromAD_TransactionRollback(t *testing.T) {
	db := setupTestDBForSync(t)
	if db == nil {
		t.Skip("测试数据库未配置")
	}

	service := NewUserSyncService(db, nil, nil)

	// 使用无效的角色ID（会导致事务失败）
	adUser := &ADUserInfoForSync{
		UserDN:      "cn=rollbackuser,dc=test,dc=com",
		Username:    "rollbackuser",
		DisplayName: "Rollback User",
		Email:       "rollback@example.com",
		Phone:       "1111111111",
	}

	_, _, err := service.SyncUserFromAD(context.Background(), adUser, "invalid-role-id")

	// 验证事务已回滚
	assert.Error(t, err)

	// 验证用户没有被创建
	var user models.User
	err = db.Where("username = ?", "rollbackuser").First(&user).Error
	assert.Error(t, err) // 应该找不到用户
	assert.True(t, gorm.ErrRecordNotFound == err)
}

// TestUserSyncService_SyncUserFromAD_RoleAssignment 测试角色分配逻辑
func TestUserSyncService_SyncUserFromAD_RoleAssignment(t *testing.T) {
	db := setupTestDBForSync(t)
	if db == nil {
		t.Skip("测试数据库未配置")
	}

	service := NewUserSyncService(db, nil, nil)

	// 创建测试角色
	role := &models.Role{
		RoleName: "Default Role",
		Status:   0,
	}
	db.Create(role)

	adUser := &ADUserInfoForSync{
		UserDN:      "cn=roleuser,dc=test,dc=com",
		Username:    "roleuser",
		DisplayName: "Role User",
		Email:       "roleuser@example.com",
		Phone:       "5555555555",
	}

	_, _, err := service.SyncUserFromAD(context.Background(), adUser, role.ID)

	// 验证结果
	assert.NoError(t, err)

	// 验证用户已创建
	var user models.User
	err = db.Where("username = ?", "roleuser").First(&user).Error
	assert.NoError(t, err)

	// 验证角色已分配
	var userRole models.UserRole
	err = db.Where("user_id = ? AND role_id = ?", user.ID, role.ID).First(&userRole).Error
	assert.NoError(t, err)
}

// TestUserSyncService_SyncUserFromAD_DepartmentAssignment 测试部门关联逻辑
func TestUserSyncService_SyncUserFromAD_DepartmentAssignment(t *testing.T) {
	db := setupTestDBForSync(t)
	if db == nil {
		t.Skip("测试数据库未配置")
	}

	service := NewUserSyncService(db, nil, nil)

	// 创建测试部门
	dept := &models.Department{
		DeptName: "Test Department",
		Status:   0,
	}
	db.Create(dept)

	adUser := &ADUserInfoForSync{
		UserDN:      "cn=deptuser,dc=test,dc=com",
		Username:    "deptuser",
		DisplayName: "Dept User",
		Email:       "deptuser@example.com",
		Phone:       "7777777777",
	}

	// TODO: 配置默认部门ID到sys_config表
	// 或者通过service方法设置

	_, _, err := service.SyncUserFromAD(context.Background(), adUser, "")

	// 验证结果
	assert.NoError(t, err)

	// 验证用户已创建并关联部门
	var user models.User
	err = db.Where("username = ?", "deptuser").First(&user).Error
	assert.NoError(t, err)

	// 如果配置了默认部门，验证部门ID
	if user.DeptID != nil {
		assert.Equal(t, dept.ID, *user.DeptID)
	}
}

// TestUserSyncService_SyncUserFromAD_TableDrivenTests 表格驱动测试
func TestUserSyncService_SyncUserFromAD_TableDrivenTests(t *testing.T) {
	db := setupTestDBForSync(t)
	if db == nil {
		t.Skip("测试数据库未配置")
	}

	service := NewUserSyncService(db, nil, nil)

	tests := []struct {
		name        string
		adUser      *ADUserInfoForSync
		roleID      string
		setupFunc   func(*gorm.DB)
		verifyFunc  func(*testing.T, *gorm.DB, error)
		description string
	}{
		{
			name: "首次登录创建用户",
			adUser: &ADUserInfoForSync{
				UserDN:      "cn=newuser,dc=test,dc=com",
				Username:    "newuser",
				DisplayName: "New User",
				Email:       "newuser@example.com",
				Phone:       "2222222222",
			},
			roleID: "",
			setupFunc: func(db *gorm.DB) {
				// 不需要前置设置
			},
			verifyFunc: func(t *testing.T, db *gorm.DB, err error) {
				assert.NoError(t, err)
				var user models.User
				err = db.Where("username = ?", "newuser").First(&user).Error
				assert.NoError(t, err)
				assert.Equal(t, "New User", *user.Nickname)
			},
			description: "AD用户首次登录应该创建新用户",
		},
		{
			name: "更新已存在用户",
			adUser: &ADUserInfoForSync{
				UserDN:      "cn=updateuser,dc=test,dc=com",
				Username:    "updateuser",
				DisplayName: "Updated Name",
				Email:       "updated@example.com",
				Phone:       "3333333333",
			},
			roleID: "",
			setupFunc: func(db *gorm.DB) {
				user := &models.User{
					Username:   "updateuser",
					AuthSource: "ad",
					ADUsername: stringPtr("updateuser"),
					Email:      stringPtr("old@example.com"),
					Status:     models.UserStatusEnabled,
				}
				db.Create(user)
			},
			verifyFunc: func(t *testing.T, db *gorm.DB, err error) {
				assert.NoError(t, err)
				var user models.User
				err = db.Where("username = ?", "updateuser").First(&user).Error
				assert.NoError(t, err)
				assert.Equal(t, "updated@example.com", *user.Email)
			},
			description: "AD用户再次登录应该更新信息",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 清理测试数据
			db.Exec("DELETE FROM sys_user WHERE username = ?", tt.adUser.Username)

			// 执行前置设置
			if tt.setupFunc != nil {
				tt.setupFunc(db)
			}

			// 执行同步
			_, _, err := service.SyncUserFromAD(context.Background(), tt.adUser, tt.roleID)

			// 验证结果
			if tt.verifyFunc != nil {
				tt.verifyFunc(t, db, err)
			}

			// 清理测试数据
			db.Exec("DELETE FROM sys_user WHERE username = ?", tt.adUser.Username)
		})
	}
}

// 辅助函数
func stringPtr(s string) *string {
	return &s
}
