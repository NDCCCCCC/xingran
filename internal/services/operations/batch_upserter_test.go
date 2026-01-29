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

// setupTestDBForUpsert 创建用于Upsert测试的数据库
func setupTestDBForUpsert(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default,
	})
	assert.NoError(t, err)

	// 创建测试表
	err = db.Exec(`
		CREATE TABLE ops_buildings (
			id TEXT PRIMARY KEY,
			name TEXT UNIQUE,
			city_code TEXT,
			org_id TEXT,
			updated_at DATETIME,
			created_at DATETIME,
			deleted_at DATETIME
		);
	`).Error
	assert.NoError(t, err)

	return db
}

// TestBatchUpsert_Insert 测试批量插入
func TestBatchUpsert_Insert(t *testing.T) {
	db := setupTestDBForUpsert(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	config := ExcelConfig{
		TableName: "ops_buildings",
		Columns: []ExcelColumn{
			{Field: "name", UpsertKey: true, DBField: "name"},
			{Field: "city_code", Header: "城市代码", DBField: "city_code"},
			{Field: "org_id", Header: "机构ID", DBField: "org_id"},
		},
	}

	upserter := NewBatchUpsert(db, config)

	records := []map[string]interface{}{
		{
			"name":       "科技楼",
			"city_code":  "wuhan",
			"org_id":     "1",
		},
		{
			"name":       "研发楼",
			"city_code":  "wuhan",
			"org_id":     "2",
		},
	}

	ctx := context.Background()
	inserted, updated, err := upserter.Upsert(ctx, records)

	assert.NoError(t, err)
	assert.Equal(t, 2, inserted)
	assert.Equal(t, 0, updated)

	// 验证数据已插入
	var count int64
	db.Table("ops_buildings").Where("deleted_at IS NULL").Count(&count)
	assert.Equal(t, int64(2), count)
}

// TestBatchUpsert_Update 测试批量更新
func TestBatchUpsert_Update(t *testing.T) {
	db := setupTestDBForUpsert(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	config := ExcelConfig{
		TableName: "ops_buildings",
		Columns: []ExcelColumn{
			{Field: "name", UpsertKey: true, DBField: "name"},
			{Field: "city_code", Header: "城市代码", DBField: "city_code"},
			{Field: "org_id", Header: "机构ID", DBField: "org_id"},
		},
	}

	upserter := NewBatchUpsert(db, config)

	ctx := context.Background()

	// 先插入数据
	insertRecords := []map[string]interface{}{
		{
			"name":       "科技楼",
			"city_code":  "wuhan",
			"org_id":     "1",
		},
	}
	_, _, err := upserter.Upsert(ctx, insertRecords)
	assert.NoError(t, err)

	// 更新数据
	updateRecords := []map[string]interface{}{
		{
			"name":       "科技楼",
			"city_code":  "beijing", // 更新城市
			"org_id":     "1",
		},
	}

	inserted, updated, err := upserter.Upsert(ctx, updateRecords)

	assert.NoError(t, err)
	assert.Equal(t, 0, inserted)
	assert.Equal(t, 1, updated)

	// 验证数据已更新
	var building struct {
		Name     string
		CityCode string
	}
	db.Table("ops_buildings").Where("name = ?", "科技楼").First(&building)
	assert.Equal(t, "beijing", building.CityCode)
}

// TestBatchUpsert_Mixed 测试混合插入和更新
func TestBatchUpsert_Mixed(t *testing.T) {
	db := setupTestDBForUpsert(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	config := ExcelConfig{
		TableName: "ops_buildings",
		Columns: []ExcelColumn{
			{Field: "name", UpsertKey: true, DBField: "name"},
			{Field: "city_code", Header: "城市代码", DBField: "city_code"},
			{Field: "org_id", Header: "机构ID", DBField: "org_id"},
		},
	}

	upserter := NewBatchUpsert(db, config)

	ctx := context.Background()

	// 先插入一条数据
	insertRecords := []map[string]interface{}{
		{
			"name":      "科技楼",
			"city_code": "wuhan",
			"org_id":    "1",
		},
	}
	_, _, err := upserter.Upsert(ctx, insertRecords)
	assert.NoError(t, err)

	// 混合操作：一条更新，一条插入
	mixedRecords := []map[string]interface{}{
		{
			"name":       "科技楼",
			"city_code":  "beijing", // 更新
			"org_id":     "1",
		},
		{
			"name":       "研发楼", // 新增
			"city_code":  "wuhan",
			"org_id":     "2",
		},
	}

	inserted, updated, err := upserter.Upsert(ctx, mixedRecords)

	assert.NoError(t, err)
	assert.Equal(t, 1, inserted)
	assert.Equal(t, 1, updated)

	// 验证最终结果
	var count int64
	db.Table("ops_buildings").Where("deleted_at IS NULL").Count(&count)
	assert.Equal(t, int64(2), count)
}

// TestBuildUpdateColumns 测试构建更新列
func TestBuildUpdateColumns(t *testing.T) {
	config := ExcelConfig{
		Columns: []ExcelColumn{
			{Field: "name", UpsertKey: true, DBField: "name"},
			{Field: "city_code", Header: "城市代码", DBField: "city_code"},
			{Field: "org_id", Header: "机构ID", DBField: "org_id"},
			{Field: "id", Header: "ID", DBField: "id"}, // 不应该更新
		},
	}

	upserter := NewBatchUpsert(nil, config)
	columns := upserter.buildUpdateColumns()

	// 验证只包含需要更新的列
	assert.Contains(t, columns, "city_code")
	assert.Contains(t, columns, "org_id")
	assert.NotContains(t, columns, "name") // UpsertKey不应该更新
	assert.NotContains(t, columns, "id")   // ID不应该更新
}
