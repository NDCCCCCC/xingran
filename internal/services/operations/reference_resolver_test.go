//go:build !skip_db_tests
// +build !skip_db_tests

package operations

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	// 使用纯 Go 实现的 SQLite 驱动
	_ "modernc.org/sqlite"
)

// setupTestDB 创建测试数据库
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default,
	})
	assert.NoError(t, err)

	// 创建测试表
	err = db.Exec(`
		CREATE TABLE sys_dept (
			id TEXT PRIMARY KEY,
			dept_name TEXT,
			dept_code TEXT UNIQUE
		);
	`).Error
	assert.NoError(t, err)

	err = db.Exec(`
		CREATE TABLE ops_buildings (
			id TEXT PRIMARY KEY,
			name TEXT,
			building_name TEXT
		);
	`).Error
	assert.NoError(t, err)

	err = db.Exec(`
		CREATE TABLE ops_floors (
			id TEXT PRIMARY KEY,
			name TEXT,
			building_id TEXT
		);
	`).Error
	assert.NoError(t, err)

	return db
}

// TestReferenceResolver_ResolveBatch 测试批量解析引用
func TestReferenceResolver_ResolveBatch(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	// 插入测试数据
	db.Exec("INSERT INTO sys_dept (id, dept_name, dept_code) VALUES ('1', '技术部', 'TECH')")
	db.Exec("INSERT INTO sys_dept (id, dept_name, dept_code) VALUES ('2', '研发中心', 'RD')")

	db.Exec("INSERT INTO ops_buildings (id, name) VALUES ('1', '科技楼')")
	db.Exec("INSERT INTO ops_buildings (id, name) VALUES ('2', '研发楼')")

	resolver := NewReferenceResolver(db)

	tests := []struct {
		name        string
		requests    []ReferenceRequest
		expectCount int
		expectError bool
	}{
		{
			name: "正常解析部门",
			requests: []ReferenceRequest{
				{Reference: "sys_dept.dept_code", Value: "TECH"},
				{Reference: "sys_dept.dept_code", Value: "RD"},
			},
			expectCount: 2,
			expectError: false,
		},
		{
			name: "正常解析楼宇",
			requests: []ReferenceRequest{
				{Reference: "ops_buildings.name", Value: "科技楼"},
				{Reference: "ops_buildings.name", Value: "研发楼"},
			},
			expectCount: 2,
			expectError: false,
		},
		{
			name: "混合解析",
			requests: []ReferenceRequest{
				{Reference: "sys_dept.dept_code", Value: "TECH"},
				{Reference: "ops_buildings.name", Value: "科技楼"},
			},
			expectCount: 2,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			results, err := resolver.ResolveBatch(ctx, tt.requests)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectCount, len(results))
			}
		})
	}
}

// TestReferenceResolver_ResolveSingle 测试单个引用解析
func TestReferenceResolver_ResolveSingle(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	// 插入测试数据
	db.Exec("INSERT INTO sys_dept (id, dept_name, dept_code) VALUES ('1', '技术部', 'TECH')")

	resolver := NewReferenceResolver(db)

	ctx := context.Background()
	result, err := resolver.ResolveSingle(ctx, ReferenceRequest{
		Reference: "sys_dept.dept_code",
		Value:     "TECH",
	})

	assert.NoError(t, err)
	assert.Equal(t, "1", result)
}

// TestReferenceResolver_InvalidReference 测试无效引用
func TestReferenceResolver_InvalidReference(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	resolver := NewReferenceResolver(db)

	ctx := context.Background()
	_, err := resolver.ResolveSingle(ctx, ReferenceRequest{
		Reference: "sys_dept.dept_code",
		Value:     "NOTEXIST",
	})

	assert.Error(t, err)
}
