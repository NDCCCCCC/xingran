package system

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// 测试数据 ID 跟踪 — cleanupTestData 按 tracked-IDs 精确删除,避免
// "Where(1=1).Delete" 把 parent-scope user 也清掉,导致后续 subtest
// 拿到失效 user.ID 误报"用户不存在"。setupTestDB 用 cache=shared,
// user.username 用 nanos 唯一所以跨 TestXxx 累积无 UNIQUE 冲突。
//
// testKeySeq 单调递增计数器,保证 createTestAPIKey 在快速循环内
// (如分页 subtest 5 次连续 create) 生成的 key 唯一 — 仅靠 time.Now().UnixNano()
// 在 µs 级循环里可能产生相同值,导致 UNIQUE 约束失败。
var (
	testTrackedUserIDs   []string
	testTrackedAPIKeyIDs []string
	testTrackedMu        sync.Mutex
	testKeySeq           uint64
)

// setupTestDB 创建测试数据库连接
// _enable_boolean=true 启用 mattn/go-sqlite3 driver 的布尔整数序列化模式,
// 让 Go bool 序列化为 1/0 (而非 "true"/"false"),与 GORM bool 字段语义一致,
// 修复 is_active 等列在 Where("is_active = ?", true) 查询时不匹配的预存问题。
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared&_enable_boolean=true"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true, // 禁用外键约束以避免SQLite兼容性问题
	})
	require.NoError(t, err)

	// 手动创建表结构，避免PostgreSQL特定的函数
	// 创建用户表
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS sys_user (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER,
			username TEXT UNIQUE,
			password TEXT,
			salt TEXT,
			nickname TEXT,
			employee_no TEXT,
			email TEXT,
			phone TEXT,
			avatar TEXT,
			gender INTEGER,
			status INTEGER,
			dept_id TEXT,
			dept_name TEXT,
			login_ip TEXT,
			login_time DATETIME,
			pwd_update_time DATETIME,
			pwd_expire_days INTEGER,
			init_flag BOOLEAN,
			remark TEXT,
			auth_source TEXT,
			ad_username TEXT,
			ad_dn TEXT,
			ad_ou_dn TEXT,
			ad_synced_at DATETIME
		)
	`).Error
	require.NoError(t, err)

	// 创建API密钥表
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS sys_api_keys (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER,
			name TEXT NOT NULL,
			key TEXT NOT NULL UNIQUE,
			user_id TEXT,
			expires_at DATETIME,
			last_used_at DATETIME,
			is_active INTEGER DEFAULT 1,
			scopes TEXT,
			ip_whitelist TEXT,
			description TEXT,
			inherit_perms BOOLEAN DEFAULT 0
		)
	`).Error
	require.NoError(t, err)

	// 创建使用日志表
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS sys_api_key_usage_logs (
			id TEXT PRIMARY KEY,
			api_key_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			method TEXT,
			path TEXT,
			status_code INTEGER,
			client_ip TEXT,
			user_agent TEXT,
			duration INTEGER,
			success BOOLEAN,
			created_at DATETIME
		)
	`).Error
	require.NoError(t, err)

	// 创建角色表 (sys_role) 及其多对多关联表 — 用于 role_service_apperrors_test.go
	// role test 同包共享此 setupTestDB,缺这三张表会导致 SELECT sys_role / INSERT sys_role_menu 报 "no such table"。
	// 列定义对齐 models.Role/RoleMenu/RoleDept 的 GORM tags (size / default / type:uuid / uniqueIndex)。
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS sys_role (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER,
			role_name TEXT NOT NULL UNIQUE,
			role_key TEXT NOT NULL UNIQUE,
			role_sort INTEGER DEFAULT 0,
			data_scope INTEGER DEFAULT 1,
			menu_check_strictly BOOLEAN DEFAULT 1,
			dept_check_strictly BOOLEAN DEFAULT 1,
			status INTEGER DEFAULT 0,
			remark TEXT
		)
	`).Error
	require.NoError(t, err)

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS sys_role_menu (
			role_id TEXT NOT NULL,
			menu_id TEXT NOT NULL,
			PRIMARY KEY (role_id, menu_id)
		)
	`).Error
	require.NoError(t, err)

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS sys_role_dept (
			role_id TEXT NOT NULL,
			dept_id TEXT NOT NULL,
			PRIMARY KEY (role_id, dept_id)
		)
	`).Error
	require.NoError(t, err)

	// 创建用户-角色 / 用户-岗位 多对多关联表 — 角色/用户 service 测试共用此 helper。
	// 列定义对齐 models.UserRole / UserPost (无 BaseModel, 复合主键)。
	// 与 pkg/middleware/permission_inherit_test.go:90 风格保持一致。
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS sys_user_role (
			user_id TEXT NOT NULL,
			role_id TEXT NOT NULL,
			PRIMARY KEY (user_id, role_id)
		)
	`).Error
	require.NoError(t, err)

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS sys_user_post (
			user_id TEXT NOT NULL,
			post_id TEXT NOT NULL,
			PRIMARY KEY (user_id, post_id)
		)
	`).Error
	require.NoError(t, err)

	return db
}

// createTestUser 创建测试用户
func createTestUser(t *testing.T, db *gorm.DB) *models.User {
	nickname := "Test User"
	// 生成唯一的用户名以避免UNIQUE约束冲突
	uniqueID := fmt.Sprintf("testuser_%d", time.Now().UnixNano())
	user := &models.User{
		Username: uniqueID,
		Nickname: &nickname,
		Password: "hashed_password",
		Status:   models.UserStatusEnabled,
	}
	err := db.Create(user).Error
	require.NoError(t, err)

	testTrackedMu.Lock()
	testTrackedUserIDs = append(testTrackedUserIDs, user.ID)
	testTrackedMu.Unlock()
	return user
}

// createTestAPIKey 创建测试API密钥
func createTestAPIKey(t *testing.T, db *gorm.DB, userID string, isActive bool) *models.APIKey {
	// 每次生成唯一 key,避免软删除残留 + 快速循环 nanos 重合 导致 UNIQUE 冲突。
	// 必须恰好 64 hex chars 以通过 isValidKeyFormat(KeyLength=64)。
	// 高 16 位用 nanos,低 48 位用 atomic counter (单调递增) — 单靠 nanos 在
	// µs 级循环内可能产生相同值(如分页 subtest 5 次连续 create)。
	testTrackedMu.Lock()
	seq := testKeySeq
	testKeySeq++
	testTrackedMu.Unlock()
	key := fmt.Sprintf("rec_%016x%048x", time.Now().UnixNano(), seq)

	apiKey := &models.APIKey{
		Name:        "Test API Key",
		Key:         key,
		UserID:      &userID,
		Scopes:      []string{"read", "write"},
		IPWhitelist: []string{},
		// 注意:不要在 struct 里设 IsActive — GORM SQLite driver 会把 Go bool
		// 序列化为 SQL 字面量 "true"/"false",SQLite 将 "false" 转 INTEGER 时存为 1
		// (SQLite 把 "true" 字面量也识别为 1,见 mattn/go-sqlite3 driver 行为)。
		// 改在 Create 后用 raw SQL 显式写 0/1 整数,确保与 Where("is_active = ?", bool) 语义一致。
	}
	err := db.Create(apiKey).Error
	require.NoError(t, err)

	// 用 raw SQL 显式设置 is_active (整数 0/1,绕过 GORM bool 序列化 bug)
	// 使用 db.Exec 而非 db.Model().Update() 避免 GORM 事务持有导致 SQLite 共享写锁竞争。
	activeVal := 0
	if isActive {
		activeVal = 1
	}
	err = db.Exec("UPDATE sys_api_keys SET is_active = ? WHERE id = ?", activeVal, apiKey.ID).Error
	require.NoError(t, err)

	testTrackedMu.Lock()
	testTrackedAPIKeyIDs = append(testTrackedAPIKeyIDs, apiKey.ID)
	testTrackedMu.Unlock()

	// 同步内存对象
	apiKey.IsActive = isActive
	return apiKey
}

// cleanupTestData 清理测试数据(按 tracked-APIKeyIDs 精确删除,user 不删)
//
// 设计要点:parent-scope user 必须在 subtest 之间保持,否则后续 subtest
// 拿到失效 user.ID 误报"用户不存在"(如 TestCreateAPIKey/无效作用域)。
// user.username 用 nanos 唯一生成,跨 subtest/TestXxx 累积无 UNIQUE 冲突,
// 所以 user 表不删 — 只删 apikey + usage_log (按 tracked-APIKeyIDs 关联)。
func cleanupTestData(t *testing.T, db *gorm.DB) {
	testTrackedMu.Lock()
	apiKeyIDs := testTrackedAPIKeyIDs
	testTrackedAPIKeyIDs = nil
	// 同样清空 tracked-user 引用以避免内存膨胀(实际 user 行保留在 db)
	testTrackedUserIDs = nil
	testTrackedMu.Unlock()

	if len(apiKeyIDs) > 0 {
		require.NoError(t, db.Unscoped().Where("api_key_id IN ?", apiKeyIDs).Delete(&models.APIKeyUsageLog{}).Error)
		require.NoError(t, db.Unscoped().Where("id IN ?", apiKeyIDs).Delete(&models.APIKey{}).Error)
	}
}

// TestCreateAPIKey 测试创建API密钥
func TestCreateAPIKey(t *testing.T) {
	db := setupTestDB(t)
	service := NewAPIKeyService(db)
	ctx := context.Background()
	user := createTestUser(t, db)

	t.Run("正常创建", func(t *testing.T) {
		expiresAt := time.Now().Add(30 * 24 * time.Hour).Format(time.RFC3339)
		description := "Test description"
		req := &requests.CreateAPIKeyRequest{
			Name:         "Test Key",
			Scopes:       []string{"read", "write"},
			IPWhitelist:  []string{},
			Description:  &description,
			InheritPerms: false,
			ExpiresAt:    &expiresAt,
		}

		key, err := service.CreateAPIKey(ctx, user.ID, req)

		assert.NoError(t, err)
		assert.NotNil(t, key)
		assert.Equal(t, 68, len(*key))      // rec_ + 64 hex chars
		assert.Equal(t, "rec_", (*key)[:4]) // 检查前缀
		assert.NotContains(t, *key, " ")     // 无空格

		// 验证数据库中的记录
		var apiKey models.APIKey
		err = db.Where("key = ?", *key).First(&apiKey).Error
		assert.NoError(t, err)
		assert.Equal(t, "Test Key", apiKey.Name)
		assert.Equal(t, user.ID, *apiKey.UserID)
		assert.True(t, apiKey.IsActive)

		cleanupTestData(t, db)
	})

	t.Run("用户不存在错误", func(t *testing.T) {
		req := &requests.CreateAPIKeyRequest{
			Name:   "Test Key",
			Scopes: []string{"read"},
		}

		_, err := service.CreateAPIKey(ctx, "non-existent-user", req)

		assert.Error(t, err)
		// 检查错误类型或消息，避免nil指针问题
		if err != nil {
			assert.Contains(t, fmt.Sprintf("%v", err), "用户不存在")
		}
	})

	t.Run("密钥数量限制", func(t *testing.T) {
		// 为此测试创建新用户，避免数据冲突
		limitUser := createTestUser(t, db)

		// 创建100个密钥达到限制
		for i := 0; i < MaxKeysPerUser; i++ {
			key := "rec_" + string(rune(i)) + "000000000000000000000000000000000000000000000000000000000000000"
			apiKey := &models.APIKey{
				Key:      key,
				Name:     "Limit Key",
				UserID:   &limitUser.ID,
				Scopes:   []string{"read"},
				IsActive: true,
			}
			err := db.Create(apiKey).Error
			require.NoError(t, err)
		}

		req := &requests.CreateAPIKeyRequest{
			Name:   "Overflow Key",
			Scopes: []string{"read"},
		}

		_, err := service.CreateAPIKey(ctx, limitUser.ID, req)

		assert.Error(t, err)
		assert.Contains(t, fmt.Sprintf("%v", err), "已达到最大密钥数量限制")

		// 这 100 个 key 是直接 db.Create (不走 createTestAPIKey helper),
		// 不在 tracked-IDs 范围,需要显式按 user_id 删,避免 shared cache 残留
		require.NoError(t, db.Unscoped().Where("user_id = ?", limitUser.ID).Delete(&models.APIKey{}).Error)
		cleanupTestData(t, db)
	})

	t.Run("无效作用域", func(t *testing.T) {
		req := &requests.CreateAPIKeyRequest{
			Name:   "Invalid Scope Key",
			Scopes: []string{"invalid_scope"},
		}

		_, err := service.CreateAPIKey(ctx, user.ID, req)

		assert.Error(t, err)
		assert.Contains(t, fmt.Sprintf("%v", err), "无效的作用域")
	})

	t.Run("密钥格式正确性", func(t *testing.T) {
		// 为此测试创建新用户，避免数据冲突
		formatUser := createTestUser(t, db)

		req := &requests.CreateAPIKeyRequest{
			Name:   "Format Test Key",
			Scopes: []string{"admin"},
		}

		key, err := service.CreateAPIKey(ctx, formatUser.ID, req)

		assert.NoError(t, err)
		assert.NotNil(t, key)

		// 验证格式: rec_ + 64位hex
		assert.Equal(t, 68, len(*key))
		assert.Equal(t, "rec_", (*key)[:4])

		// 验证后64位为十六进制
		hexPart := (*key)[4:]
		assert.Equal(t, 64, len(hexPart))
		for _, c := range hexPart {
			assert.True(t, (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F'))
		}

		cleanupTestData(t, db)
	})
}

// TestValidateAPIKey 测试验证API密钥
func TestValidateAPIKey(t *testing.T) {
	db := setupTestDB(t)
	service := NewAPIKeyService(db)
	ctx := context.Background()
	user := createTestUser(t, db)

	t.Run("有效密钥", func(t *testing.T) {
		apiKey := createTestAPIKey(t, db, user.ID, true)

		validated, err := service.ValidateAPIKey(ctx, apiKey.Key)

		assert.NoError(t, err)
		assert.NotNil(t, validated)
		assert.Equal(t, apiKey.ID, validated.ID)
		assert.Equal(t, apiKey.Name, validated.Name)
		assert.True(t, validated.IsActive)
	})

	t.Run("无效格式_无前缀", func(t *testing.T) {
		invalidKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

		_, err := service.ValidateAPIKey(ctx, invalidKey)

		assert.Error(t, err)
		assert.Contains(t, fmt.Sprintf("%v", err), "无效的密钥格式")
	})

	t.Run("无效格式_长度错误", func(t *testing.T) {
		shortKey := "rec_0123456789abcdef"

		_, err := service.ValidateAPIKey(ctx, shortKey)

		assert.Error(t, err)
		assert.Contains(t, fmt.Sprintf("%v", err), "无效的密钥格式")
	})

	t.Run("无效格式_非十六进制", func(t *testing.T) {
		invalidHexKey := "rec_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcxyz"

		_, err := service.ValidateAPIKey(ctx, invalidHexKey)

		assert.Error(t, err)
		assert.Contains(t, fmt.Sprintf("%v", err), "无效的密钥格式")
	})

	t.Run("密钥不存在", func(t *testing.T) {
		nonExistentKey := "rec_" + "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"

		_, err := service.ValidateAPIKey(ctx, nonExistentKey)

		assert.Error(t, err)
		assert.Contains(t, fmt.Sprintf("%v", err), "密钥不存在或已禁用")
	})

	t.Run("密钥已禁用", func(t *testing.T) {
		apiKey := createTestAPIKey(t, db, user.ID, false)

		_, err := service.ValidateAPIKey(ctx, apiKey.Key)

		assert.Error(t, err)
		assert.Contains(t, fmt.Sprintf("%v", err), "密钥不存在或已禁用")

		cleanupTestData(t, db)
	})

	t.Run("密钥已过期", func(t *testing.T) {
		pastTime := time.Now().Add(-24 * time.Hour)
		apiKey := createTestAPIKey(t, db, user.ID, true)
		apiKey.ExpiresAt = &pastTime
		db.Save(apiKey)

		_, err := service.ValidateAPIKey(ctx, apiKey.Key)

		assert.Error(t, err)
		assert.Contains(t, fmt.Sprintf("%v", err), "密钥已过期")

		cleanupTestData(t, db)
	})

	t.Run("最后使用时间更新", func(t *testing.T) {
		apiKey := createTestAPIKey(t, db, user.ID, true)

		// 验证前：last_used_at 为空
		var beforeUpdate models.APIKey
		db.First(&beforeUpdate, "id = ?", apiKey.ID)
		assert.Nil(t, beforeUpdate.LastUsedAt)

		// 验证密钥
		_, err := service.ValidateAPIKey(ctx, apiKey.Key)
		assert.NoError(t, err)

		// 等待异步更新完成
		time.Sleep(100 * time.Millisecond)

		// 验证后：last_used_at 已更新
		var afterUpdate models.APIKey
		db.First(&afterUpdate, "id = ?", apiKey.ID)
		assert.NotNil(t, afterUpdate.LastUsedAt)
		assert.WithinDuration(t, time.Now(), *afterUpdate.LastUsedAt, 5*time.Second)

		cleanupTestData(t, db)
	})
}

// TestListAPIKeys 测试查询API密钥列表
func TestListAPIKeys(t *testing.T) {
	db := setupTestDB(t)
	service := NewAPIKeyService(db)
	ctx := context.Background()
	user := createTestUser(t, db)

	// 创建多个测试密钥
	key1 := createTestAPIKey(t, db, user.ID, true)
	key1.Name = "Active Key"
	key1.Scopes = []string{"read"}
	db.Save(key1)

	key2 := createTestAPIKey(t, db, user.ID, false)
	key2.Name = "Other Key" // 不能含 "Active" 子串,否则关键词搜索 LIKE '%Active%' 会误匹配
	key2.Scopes = []string{"write"}
	db.Save(key2)

	t.Run("正常查询", func(t *testing.T) {
		params := requests.ListAPIKeysParams{
			Current:  1,
			PageSize: 10,
		}

		result, err := service.ListAPIKeys(ctx, user.ID, params)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int64(2), result.Total)

		// 转换 List 为 []models.APIKey
		list, ok := result.List.([]models.APIKey)
		assert.True(t, ok)
		assert.Equal(t, 2, len(list))
		assert.Equal(t, 1, result.Current)
		assert.Equal(t, 10, result.PageSize)
	})

	t.Run("关键词搜索", func(t *testing.T) {
		keyword := "Active"
		params := requests.ListAPIKeysParams{
			Current:  1,
			PageSize: 10,
			Keyword:  &keyword,
		}

		result, err := service.ListAPIKeys(ctx, user.ID, params)

		assert.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, int64(1), result.Total)
		list, _ := result.List.([]models.APIKey)
		assert.Equal(t, 1, len(list))
		assert.Contains(t, list[0].Name, "Active")
	})

	t.Run("状态筛选", func(t *testing.T) {
		status := true
		params := requests.ListAPIKeysParams{
			Current:  1,
			PageSize: 10,
			Status:   &status,
		}

		result, err := service.ListAPIKeys(ctx, user.ID, params)

		assert.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, int64(1), result.Total)
		list, _ := result.List.([]models.APIKey)
		assert.True(t, list[0].IsActive)
	})

	t.Run("作用域筛选", func(t *testing.T) {
		scope := "read"
		params := requests.ListAPIKeysParams{
			Current:  1,
			PageSize: 10,
			Scope:    &scope,
		}

		result, err := service.ListAPIKeys(ctx, user.ID, params)

		assert.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, int64(1), result.Total)
		list, _ := result.List.([]models.APIKey)
		assert.Contains(t, list[0].Scopes, "read")
	})

	t.Run("分页功能", func(t *testing.T) {
		// 添加更多密钥以测试分页
		for i := 0; i < 5; i++ {
			key := createTestAPIKey(t, db, user.ID, true)
			key.Name = "Pagination Key " + string(rune(i))
			db.Save(key)
		}

		params := requests.ListAPIKeysParams{
			Current:  1,
			PageSize: 3,
		}

		result, err := service.ListAPIKeys(ctx, user.ID, params)

		assert.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, int64(7), result.Total) // 2 existing + 5 new
		list, _ := result.List.([]models.APIKey)
		assert.Equal(t, 3, len(list))
		assert.Equal(t, 1, result.Current)
		assert.Equal(t, 3, result.PageSize)

		cleanupTestData(t, db)
	})
}

// TestGetAPIKey 测试获取API密钥详情
func TestGetAPIKey(t *testing.T) {
	db := setupTestDB(t)
	service := NewAPIKeyService(db)
	ctx := context.Background()
	user := createTestUser(t, db)

	t.Run("正常获取", func(t *testing.T) {
		apiKey := createTestAPIKey(t, db, user.ID, true)

		result, err := service.GetAPIKey(ctx, apiKey.ID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, apiKey.ID, result.ID)
		assert.Equal(t, apiKey.Name, result.Name)
		assert.Equal(t, apiKey.Key, result.Key)

		cleanupTestData(t, db)
	})

	t.Run("密钥不存在", func(t *testing.T) {
		_, err := service.GetAPIKey(ctx, "non-existent-id")

		assert.Error(t, err)
		assert.Contains(t, fmt.Sprintf("%v", err), "密钥不存在")
	})

	t.Run("密钥脱敏", func(t *testing.T) {
		// 注意：当前实现返回完整密钥，实际生产环境应该脱敏
		// 这里测试验证当前行为
		apiKey := createTestAPIKey(t, db, user.ID, true)

		result, err := service.GetAPIKey(ctx, apiKey.ID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		// 完整密钥长度68，脱敏应该只显示前12位
		// 但当前实现返回完整密钥
		assert.Equal(t, 68, len(result.Key))

		cleanupTestData(t, db)
	})
}

// TestUpdateAPIKey 测试更新API密钥
func TestUpdateAPIKey(t *testing.T) {
	db := setupTestDB(t)
	service := NewAPIKeyService(db)
	ctx := context.Background()
	user := createTestUser(t, db)

	t.Run("正常更新", func(t *testing.T) {
		apiKey := createTestAPIKey(t, db, user.ID, true)
		newName := "Updated Name"
		newDesc := "Updated description"

		req := &requests.UpdateAPIKeyRequest{
			Name:        &newName,
			Description: &newDesc,
		}

		err := service.UpdateAPIKey(ctx, apiKey.ID, req)

		assert.NoError(t, err)

		// 验证更新结果
		var updated models.APIKey
		db.First(&updated, "id = ?", apiKey.ID)
		assert.Equal(t, newName, updated.Name)
		assert.Equal(t, newDesc, *updated.Description)

		cleanupTestData(t, db)
	})

	t.Run("密钥不存在", func(t *testing.T) {
		newName := "Updated Name"
		req := &requests.UpdateAPIKeyRequest{
			Name: &newName,
		}

		err := service.UpdateAPIKey(ctx, "non-existent-id", req)

		assert.Error(t, err)
		assert.Contains(t, fmt.Sprintf("%v", err), "密钥不存在")
	})

	t.Run("无效作用域", func(t *testing.T) {
		apiKey := createTestAPIKey(t, db, user.ID, true)
		invalidScopes := []string{"invalid_scope"}

		req := &requests.UpdateAPIKeyRequest{
			Scopes: invalidScopes,
		}

		err := service.UpdateAPIKey(ctx, apiKey.ID, req)

		assert.Error(t, err)
		assert.Contains(t, fmt.Sprintf("%v", err), "无效的作用域")

		cleanupTestData(t, db)
	})
}

// TestDeleteAPIKey 测试删除API密钥
func TestDeleteAPIKey(t *testing.T) {
	db := setupTestDB(t)
	service := NewAPIKeyService(db)
	ctx := context.Background()
	user := createTestUser(t, db)

	t.Run("正常删除（软删除）", func(t *testing.T) {
		apiKey := createTestAPIKey(t, db, user.ID, true)

		err := service.DeleteAPIKey(ctx, apiKey.ID)

		assert.NoError(t, err)

		// 验证软删除（deleted_at 不为空）
		var deleted models.APIKey
		err = db.Unscoped().First(&deleted, "id = ?", apiKey.ID).Error
		assert.NoError(t, err)
		assert.NotNil(t, deleted.DeletedAt)

		cleanupTestData(t, db)
	})

	t.Run("密钥不存在", func(t *testing.T) {
		err := service.DeleteAPIKey(ctx, "non-existent-id")

		assert.Error(t, err)
		assert.Contains(t, fmt.Sprintf("%v", err), "密钥不存在")
	})

	t.Run("删除后不可查询", func(t *testing.T) {
		apiKey := createTestAPIKey(t, db, user.ID, true)

		// 删除
		err := service.DeleteAPIKey(ctx, apiKey.ID)
		assert.NoError(t, err)

		// 尝试查询（应该找不到）
		_, err = service.GetAPIKey(ctx, apiKey.ID)
		assert.Error(t, err)
		assert.Contains(t, fmt.Sprintf("%v", err), "密钥不存在")

		cleanupTestData(t, db)
	})
}

// TestToggleAPIKeyStatus 测试切换API密钥状态
func TestToggleAPIKeyStatus(t *testing.T) {
	db := setupTestDB(t)
	service := NewAPIKeyService(db)
	ctx := context.Background()
	user := createTestUser(t, db)

	t.Run("启用切换", func(t *testing.T) {
		apiKey := createTestAPIKey(t, db, user.ID, true)
		initialStatus := apiKey.IsActive

		err := service.ToggleAPIKeyStatus(ctx, apiKey.ID)

		assert.NoError(t, err)

		// 验证状态已切换
		var toggled models.APIKey
		db.First(&toggled, "id = ?", apiKey.ID)
		assert.NotEqual(t, initialStatus, toggled.IsActive)
		assert.False(t, toggled.IsActive)

		cleanupTestData(t, db)
	})

	t.Run("禁用切换", func(t *testing.T) {
		apiKey := createTestAPIKey(t, db, user.ID, false)
		initialStatus := apiKey.IsActive

		err := service.ToggleAPIKeyStatus(ctx, apiKey.ID)

		assert.NoError(t, err)

		// 验证状态已切换
		var toggled models.APIKey
		db.First(&toggled, "id = ?", apiKey.ID)
		assert.NotEqual(t, initialStatus, toggled.IsActive)
		assert.True(t, toggled.IsActive)

		cleanupTestData(t, db)
	})

	t.Run("密钥不存在", func(t *testing.T) {
		err := service.ToggleAPIKeyStatus(ctx, "non-existent-id")

		assert.Error(t, err)
		assert.Contains(t, fmt.Sprintf("%v", err), "密钥不存在")
	})
}

// TestListUsageLogs 测试查询使用日志
func TestListUsageLogs(t *testing.T) {
	db := setupTestDB(t)
	service := NewAPIKeyService(db)
	ctx := context.Background()
	user := createTestUser(t, db)

	t.Run("正常查询", func(t *testing.T) {
		apiKey := createTestAPIKey(t, db, user.ID, true)

		// 创建测试日志
		now := time.Now()
		logs := []models.APIKeyUsageLog{
			{
				APIKeyID:   apiKey.ID,
				UserID:     user.ID,
				Method:     "GET",
				Path:       "/api/v1/test",
				StatusCode: 200,
				ClientIP:   "127.0.0.1",
				Duration:   100,
				Success:    true,
				CreatedAt:  now.Add(-2 * time.Hour),
			},
			{
				APIKeyID:   apiKey.ID,
				UserID:     user.ID,
				Method:     "POST",
				Path:       "/api/v1/test2",
				StatusCode: 201,
				ClientIP:   "127.0.0.2",
				Duration:   200,
				Success:    true,
				CreatedAt:  now.Add(-1 * time.Hour),
			},
		}
		for _, log := range logs {
			db.Create(&log)
		}

		params := ListUsageLogsParams{
			APIKeyID:  apiKey.ID,
			Current:   1,
			PageSize:  10,
		}

		result, err := service.ListUsageLogs(ctx, params)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int64(2), result.Total)
		assert.Equal(t, 2, len(result.List))
		assert.Equal(t, 1, result.Current)
		assert.Equal(t, 10, result.PageSize)

		cleanupTestData(t, db)
	})

	t.Run("时间范围筛选", func(t *testing.T) {
		apiKey := createTestAPIKey(t, db, user.ID, true)

		now := time.Now()
		oldTime := now.Add(-48 * time.Hour)
		recentTime := now.Add(-1 * time.Hour)

		// 创建不同时间的日志
		oldLog := models.APIKeyUsageLog{
			APIKeyID:   apiKey.ID,
			UserID:     user.ID,
			Method:     "GET",
			Path:       "/api/v1/old",
			StatusCode: 200,
			ClientIP:   "127.0.0.1",
			Duration:   100,
			Success:    true,
			CreatedAt:  oldTime,
		}
		recentLog := models.APIKeyUsageLog{
			APIKeyID:   apiKey.ID,
			UserID:     user.ID,
			Method:     "GET",
			Path:       "/api/v1/recent",
			StatusCode: 200,
			ClientIP:   "127.0.0.1",
			Duration:   100,
			Success:    true,
			CreatedAt:  recentTime,
		}
		db.Create(&oldLog)
		db.Create(&recentLog)

		startTime := now.Add(-24 * time.Hour).Format(time.RFC3339)
		params := ListUsageLogsParams{
			APIKeyID:  apiKey.ID,
			Current:   1,
			PageSize:  10,
			StartTime: &startTime,
		}

		result, err := service.ListUsageLogs(ctx, params)

		assert.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, int64(1), result.Total)
		assert.Equal(t, "/api/v1/recent", result.List[0].Path)

		cleanupTestData(t, db)
	})

	t.Run("成功筛选", func(t *testing.T) {
		apiKey := createTestAPIKey(t, db, user.ID, true)

		// 创建成功和失败的日志
		successLog := models.APIKeyUsageLog{
			APIKeyID:   apiKey.ID,
			UserID:     user.ID,
			Method:     "GET",
			Path:       "/api/v1/success",
			StatusCode: 200,
			ClientIP:   "127.0.0.1",
			Duration:   100,
			Success:    true,
		}
		failLog := models.APIKeyUsageLog{
			APIKeyID:   apiKey.ID,
			UserID:     user.ID,
			Method:     "GET",
			Path:       "/api/v1/fail",
			StatusCode: 500,
			ClientIP:   "127.0.0.1",
			Duration:   200,
			Success:    false,
		}
		db.Create(&successLog)
		db.Create(&failLog)

		success := true
		params := ListUsageLogsParams{
			APIKeyID:  apiKey.ID,
			Current:   1,
			PageSize:  10,
			Success:   &success,
		}

		result, err := service.ListUsageLogs(ctx, params)

		assert.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, int64(1), result.Total)
		assert.True(t, result.List[0].Success)
		assert.Equal(t, "/api/v1/success", result.List[0].Path)

		cleanupTestData(t, db)
	})

	t.Run("分页功能", func(t *testing.T) {
		apiKey := createTestAPIKey(t, db, user.ID, true)

		// 创建5条日志
		for i := 0; i < 5; i++ {
			log := models.APIKeyUsageLog{
				APIKeyID:   apiKey.ID,
				UserID:     user.ID,
				Method:     "GET",
				Path:       "/api/v1/test",
				StatusCode: 200,
				ClientIP:   "127.0.0.1",
				Duration:   100,
				Success:    true,
			}
			db.Create(&log)
		}

		params := ListUsageLogsParams{
			APIKeyID:  apiKey.ID,
			Current:   1,
			PageSize:  3,
		}

		result, err := service.ListUsageLogs(ctx, params)

		assert.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, int64(5), result.Total)
		assert.Equal(t, 3, len(result.List))
		assert.Equal(t, 1, result.Current)
		assert.Equal(t, 3, result.PageSize)

		cleanupTestData(t, db)
	})
}

// TestGetUsageLogSummary 测试获取使用统计汇总
func TestGetUsageLogSummary(t *testing.T) {
	db := setupTestDB(t)
	service := NewAPIKeyService(db)
	ctx := context.Background()
	user := createTestUser(t, db)

	t.Run("统计数据正确性", func(t *testing.T) {
		apiKey := createTestAPIKey(t, db, user.ID, true)

		// 创建测试日志
		logs := []models.APIKeyUsageLog{
			{
				APIKeyID:   apiKey.ID,
				UserID:     user.ID,
				Method:     "GET",
				Path:       "/api/v1/test1",
				StatusCode: 200,
				ClientIP:   "127.0.0.1",
				Duration:   100,
				Success:    true,
			},
			{
				APIKeyID:   apiKey.ID,
				UserID:     user.ID,
				Method:     "POST",
				Path:       "/api/v1/test2",
				StatusCode: 201,
				ClientIP:   "127.0.0.1",
				Duration:   200,
				Success:    true,
			},
			{
				APIKeyID:   apiKey.ID,
				UserID:     user.ID,
				Method:     "GET",
				Path:       "/api/v1/test3",
				StatusCode: 500,
				ClientIP:   "127.0.0.1",
				Duration:   300,
				Success:    false,
			},
		}
		for _, log := range logs {
			db.Create(&log)
		}

		summary, err := service.GetUsageLogSummary(ctx, apiKey.ID)

		assert.NoError(t, err)
		assert.NotNil(t, summary)
		assert.Equal(t, int64(3), summary.TotalRequests)

		cleanupTestData(t, db)
	})

	t.Run("总请求数", func(t *testing.T) {
		apiKey := createTestAPIKey(t, db, user.ID, true)

		// 创建10条日志
		for i := 0; i < 10; i++ {
			log := models.APIKeyUsageLog{
				APIKeyID:   apiKey.ID,
				UserID:     user.ID,
				Method:     "GET",
				Path:       "/api/v1/test",
				StatusCode: 200,
				ClientIP:   "127.0.0.1",
				Duration:   100,
				Success:    true,
			}
			db.Create(&log)
		}

		summary, err := service.GetUsageLogSummary(ctx, apiKey.ID)

		assert.NoError(t, err)
		assert.Equal(t, int64(10), summary.TotalRequests)

		cleanupTestData(t, db)
	})

	t.Run("成功率计算", func(t *testing.T) {
		apiKey := createTestAPIKey(t, db, user.ID, true)

		// 创建7条成功、3条失败的日志
		for i := 0; i < 7; i++ {
			log := models.APIKeyUsageLog{
				APIKeyID:   apiKey.ID,
				UserID:     user.ID,
				Method:     "GET",
				Path:       "/api/v1/success",
				StatusCode: 200,
				ClientIP:   "127.0.0.1",
				Duration:   100,
				Success:    true,
			}
			db.Create(&log)
		}
		for i := 0; i < 3; i++ {
			log := models.APIKeyUsageLog{
				APIKeyID:   apiKey.ID,
				UserID:     user.ID,
				Method:     "GET",
				Path:       "/api/v1/fail",
				StatusCode: 500,
				ClientIP:   "127.0.0.1",
				Duration:   200,
				Success:    false,
			}
			db.Create(&log)
		}

		summary, err := service.GetUsageLogSummary(ctx, apiKey.ID)

		assert.NoError(t, err)
		assert.InDelta(t, 70.0, summary.SuccessRate, 0.1) // 7/10 = 70%

		cleanupTestData(t, db)
	})

	t.Run("平均耗时", func(t *testing.T) {
		apiKey := createTestAPIKey(t, db, user.ID, true)

		// 创建耗时不同的日志
		durations := []int{100, 200, 300, 400, 500}
		for _, duration := range durations {
			log := models.APIKeyUsageLog{
				APIKeyID:   apiKey.ID,
				UserID:     user.ID,
				Method:     "GET",
				Path:       "/api/v1/test",
				StatusCode: 200,
				ClientIP:   "127.0.0.1",
				Duration:   duration,
				Success:    true,
			}
			db.Create(&log)
		}

		summary, err := service.GetUsageLogSummary(ctx, apiKey.ID)

		assert.NoError(t, err)
		assert.InDelta(t, 300.0, summary.AvgDuration, 0.1) // (100+200+300+400+500)/5 = 300

		cleanupTestData(t, db)
	})

	t.Run("按方法分组", func(t *testing.T) {
		apiKey := createTestAPIKey(t, db, user.ID, true)

		// 创建不同方法的日志
		methods := []string{"GET", "GET", "POST", "POST", "POST", "DELETE"}
		for _, method := range methods {
			log := models.APIKeyUsageLog{
				APIKeyID:   apiKey.ID,
				UserID:     user.ID,
				Method:     method,
				Path:       "/api/v1/test",
				StatusCode: 200,
				ClientIP:   "127.0.0.1",
				Duration:   100,
				Success:    true,
			}
			db.Create(&log)
		}

		summary, err := service.GetUsageLogSummary(ctx, apiKey.ID)

		assert.NoError(t, err)
		assert.Equal(t, int64(2), summary.RequestsByMethod["GET"])
		assert.Equal(t, int64(3), summary.RequestsByMethod["POST"])
		assert.Equal(t, int64(1), summary.RequestsByMethod["DELETE"])

		cleanupTestData(t, db)
	})

	t.Run("按路径分组", func(t *testing.T) {
		apiKey := createTestAPIKey(t, db, user.ID, true)

		// 创建不同路径的日志
		paths := []string{"/api/v1/users", "/api/v1/users", "/api/v1/posts", "/api/v1/posts", "/api/v1/posts"}
		for i, path := range paths {
			log := models.APIKeyUsageLog{
				APIKeyID:   apiKey.ID,
				UserID:     user.ID,
				Method:     "GET",
				Path:       path,
				StatusCode: 200,
				ClientIP:   "127.0.0.1",
				Duration:   100,
				Success:    true,
			}
			db.Create(&log)
			_ = i
		}

		summary, err := service.GetUsageLogSummary(ctx, apiKey.ID)

		assert.NoError(t, err)
		assert.Equal(t, int64(3), summary.RequestsByPath["/api/v1/posts"])
		assert.Equal(t, int64(2), summary.RequestsByPath["/api/v1/users"])

		cleanupTestData(t, db)
	})

	t.Run("错误统计", func(t *testing.T) {
		apiKey := createTestAPIKey(t, db, user.ID, true)

		// 创建不同错误状态的日志
		errors := []int{400, 400, 401, 403, 500}
		for _, statusCode := range errors {
			log := models.APIKeyUsageLog{
				APIKeyID:   apiKey.ID,
				UserID:     user.ID,
				Method:     "GET",
				Path:       "/api/v1/test",
				StatusCode: statusCode,
				ClientIP:   "127.0.0.1",
				Duration:   100,
				Success:    false,
			}
			db.Create(&log)
		}

		summary, err := service.GetUsageLogSummary(ctx, apiKey.ID)

		assert.NoError(t, err)
		assert.Equal(t, 2, summary.ErrorsByStatus[400])
		assert.Equal(t, 1, summary.ErrorsByStatus[401])
		assert.Equal(t, 1, summary.ErrorsByStatus[403])
		assert.Equal(t, 1, summary.ErrorsByStatus[500])

		cleanupTestData(t, db)
	})
}

// --- Phase 59 Plan 02: SC#3 (OBSERV-02 successRate 连锁防回归) ---

// TestGetUsageLogSummaryMixed SC#3: 混合 Success=true/false 行后, GetUsageLogSummary 必须返回
// successRate ∈ (0,100) 开区间, 不恒 ≈ 0% (修复前 Success 永远 false → 聚合恒 0%)。
//
// 直接 seed 真实 DB 行 (不经 middleware 异步链), 纯聚合逻辑防回归锚。
// 2 Success=true + 2 Success=false → 2/4 = 50% 精确锚。
func TestGetUsageLogSummaryMixed(t *testing.T) {
	db := setupTestDB(t)
	service := NewAPIKeyService(db)
	ctx := context.Background()
	user := createTestUser(t, db)
	apiKey := createTestAPIKey(t, db, user.ID, true)

	// 混合 seed: 2 Success=true + 2 Success=false (下游 403/429, 非 pre-auth 401)
	logs := []models.APIKeyUsageLog{
		{
			APIKeyID:   apiKey.ID,
			UserID:     user.ID,
			Method:     "GET",
			Path:       "/api/v1/success-1",
			StatusCode: 200,
			ClientIP:   "127.0.0.1",
			Duration:   50,
			Success:    true, // 2xx → Success=true (OBSERV-01 修复后)
		},
		{
			APIKeyID:   apiKey.ID,
			UserID:     user.ID,
			Method:     "POST",
			Path:       "/api/v1/success-2",
			StatusCode: 204,
			ClientIP:   "127.0.0.1",
			Duration:   30,
			Success:    true, // 2xx → Success=true (OBSERV-01 修复后)
		},
		{
			APIKeyID:   apiKey.ID,
			UserID:     user.ID,
			Method:     "GET",
			Path:       "/api/v1/forbidden",
			StatusCode: 403, // 下游 RequireScope 失败
			ClientIP:   "127.0.0.1",
			Duration:   20,
			Success:    false, // D-01: 4xx → Success=false
		},
		{
			APIKeyID:   apiKey.ID,
			UserID:     user.ID,
			Method:     "GET",
			Path:       "/api/v1/ratelimit",
			StatusCode: 429, // 下游 RateLimitByScope 失败
			ClientIP:   "127.0.0.1",
			Duration:   10,
			Success:    false, // D-01: 4xx → Success=false
		},
	}
	for _, log := range logs {
		require.NoError(t, db.Create(&log).Error)
	}

	// 调聚合
	summary, err := service.GetUsageLogSummary(ctx, apiKey.ID)
	require.NoError(t, err)
	require.NotNil(t, summary)

	// SC#3 防回归三道断言: successRate ∈ (0,100) 开区间, 精确锚 50%
	assert.Greater(t, summary.SuccessRate, 0.0, "SuccessRate 必须 > 0 (不恒 ≈ 0%)")
	assert.Less(t, summary.SuccessRate, 100.0, "SuccessRate 必须 < 100 (不全成功)")
	assert.InDelta(t, 50.0, summary.SuccessRate, 0.1, "2/4 = 50% 精确锚")

	cleanupTestData(t, db)
}
