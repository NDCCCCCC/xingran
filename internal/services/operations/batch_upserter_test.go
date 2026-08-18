//go:build !skip_db_tests
// +build !skip_db_tests

package operations

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	// 使用纯 Go 实现的 SQLite 驱动
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
			"name":      "科技楼",
			"city_code": "wuhan",
			"org_id":    "1",
		},
		{
			"name":      "研发楼",
			"city_code": "wuhan",
			"org_id":    "2",
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
			"name":      "科技楼",
			"city_code": "wuhan",
			"org_id":    "1",
		},
	}
	_, _, err := upserter.Upsert(ctx, insertRecords)
	assert.NoError(t, err)

	// 更新数据
	updateRecords := []map[string]interface{}{
		{
			"name":      "科技楼",
			"city_code": "beijing", // 更新城市
			"org_id":    "1",
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
			"name":      "科技楼",
			"city_code": "beijing", // 更新
			"org_id":    "1",
		},
		{
			"name":      "研发楼", // 新增
			"city_code": "wuhan",
			"org_id":    "2",
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

// TestGORMNamingStrategy 测试 GORM 命名策略的字段名转换
func TestGORMNamingStrategy(t *testing.T) {
	namer := newGORMNamingStrategy()

	tests := []struct {
		name      string
		fieldName string
		want      string
	}{
		{
			name:      "驼峰转蛇形 - deviceCode",
			fieldName: "deviceCode",
			want:      "device_code",
		},
		{
			name:      "驼峰转蛇形 - roomId",
			fieldName: "roomId",
			want:      "room_id",
		},
		{
			name:      "驼峰转蛇形 - serialNumber",
			fieldName: "serialNumber",
			want:      "serial_number",
		},
		{
			name:      "驼峰转蛇形 - positionU",
			fieldName: "positionU",
			want:      "position_u",
		},
		{
			name:      "驼峰转蛇形 - assetNumber",
			fieldName: "assetNumber",
			want:      "asset_number",
		},
		{
			name:      "驼峰转蛇形 - purchaseDate",
			fieldName: "purchaseDate",
			want:      "purchase_date",
		},
		{
			name:      "驼峰转蛇形 - warrantyDate",
			fieldName: "warrantyDate",
			want:      "warranty_date",
		},
		{
			name:      "驼峰转蛇形 - powerConsumption",
			fieldName: "powerConsumption",
			want:      "power_consumption",
		},
		{
			name:      "驼峰转蛇形 - positionDesc",
			fieldName: "positionDesc",
			want:      "position_desc",
		},
		{
			name:      "已经是蛇形 - device_type",
			fieldName: "device_type",
			want:      "device_type",
		},
		{
			name:      "小写 - name",
			fieldName: "name",
			want:      "name",
		},
		{
			name:      "单驼峰 - vendor",
			fieldName: "vendor",
			want:      "vendor",
		},
		{
			name:      "多级驼峰 - responsiblePersonName",
			fieldName: "responsiblePersonName",
			want:      "responsible_person_name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := namer.FieldToDB(tt.fieldName)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestBatchUpsertResolveFieldName 测试字段名解析
func TestBatchUpsertResolveFieldName(t *testing.T) {
	config := ExcelConfig{
		Columns: []ExcelColumn{
			{Field: "deviceCode", DBField: "device_code"},
			{Field: "name", DBField: "name"},
			{Field: "roomId", DBField: "room_id"},
			{Field: "serialNumber"}, // 没有 DBField，应该使用 NamingStrategy
		},
	}

	upserter := NewBatchUpsert(nil, config)

	tests := []struct {
		name      string
		fieldName string
		want      string
	}{
		{
			name:      "使用配置的DBField - deviceCode",
			fieldName: "deviceCode",
			want:      "device_code",
		},
		{
			name:      "使用配置的DBField - name",
			fieldName: "name",
			want:      "name",
		},
		{
			name:      "使用配置的DBField - roomId",
			fieldName: "roomId",
			want:      "room_id",
		},
		{
			name:      "没有DBField时使用NamingStrategy - serialNumber",
			fieldName: "serialNumber",
			want:      "serial_number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := upserter.resolveFieldName(tt.fieldName)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestBatchUpsertConvertRecordFields 测试记录字段转换
func TestBatchUpsertConvertRecordFields(t *testing.T) {
	config := ExcelConfig{
		Columns: []ExcelColumn{
			{Field: "deviceCode", DBField: "device_code"},
			{Field: "name"},
			{Field: "roomId"},
			{Field: "serialNumber"},
		},
	}

	upserter := NewBatchUpsert(nil, config)

	records := []map[string]interface{}{
		{
			"deviceCode":   "DEV001",
			"name":         "服务器1",
			"roomId":       "room-123",
			"serialNumber": "SN12345",
		},
	}

	converted := upserter.convertRecordFields(records)

	assert.Len(t, converted, 1)
	assert.Equal(t, "DEV001", converted[0]["device_code"])
	assert.Equal(t, "服务器1", converted[0]["name"])
	assert.Equal(t, "room_id", upserter.resolveFieldName("roomId"))
	assert.Equal(t, "serial_number", upserter.resolveFieldName("serialNumber"))
}

// TestBatchUpsertWithCamelCaseFields 测试驼峰命名字段的 Upsert
func TestBatchUpsertWithCamelCaseFields(t *testing.T) {
	db := setupTestDBForUpsert(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	// 创建包含驼峰命名列的测试表
	// 注:必须含 deleted_at — standardUpsert 无条件写入该列以恢复软删除记录
	err := db.Exec(`
		CREATE TABLE test_devices (
			id TEXT PRIMARY KEY,
			device_code TEXT UNIQUE,
			device_type TEXT,
			serial_number TEXT,
			purchase_date TEXT,
			power_consumption REAL,
			updated_at DATETIME,
			created_at DATETIME,
			deleted_at DATETIME
		);
	`).Error
	assert.NoError(t, err)

	config := ExcelConfig{
		TableName: "test_devices",
		Columns: []ExcelColumn{
			{Field: "deviceCode", UpsertKey: true}, // 没有 DBField，使用 NamingStrategy
			{Field: "deviceType"},
			{Field: "serialNumber"},
			{Field: "purchaseDate"},
			{Field: "powerConsumption"},
		},
	}

	upserter := NewBatchUpsert(db, config)

	records := []map[string]interface{}{
		{
			"deviceCode":       "SRV-001",
			"deviceType":       "server",
			"serialNumber":     "SN12345",
			"purchaseDate":     "2024-01-15",
			"powerConsumption": 500.5,
		},
	}

	ctx := context.Background()
	inserted, updated, err := upserter.Upsert(ctx, records)

	assert.NoError(t, err)
	assert.Equal(t, 1, inserted)
	assert.Equal(t, 0, updated)

	// 验证数据已正确插入到驼峰命名转换后的列中
	var device struct {
		DeviceCode       string
		DeviceType       string
		SerialNumber     string
		PurchaseDate     string
		PowerConsumption float64
	}
	db.Table("test_devices").Where("device_code = ?", "SRV-001").First(&device)

	assert.Equal(t, "SRV-001", device.DeviceCode)
	assert.Equal(t, "server", device.DeviceType)
	assert.Equal(t, "SN12345", device.SerialNumber)
	assert.Equal(t, "2024-01-15", device.PurchaseDate)
	assert.Equal(t, 500.5, device.PowerConsumption)
}

// TestBatchUpsertFieldCache 测试字段名缓存
func TestBatchUpsertFieldCache(t *testing.T) {
	config := ExcelConfig{
		Columns: []ExcelColumn{
			{Field: "deviceCode"},
			{Field: "name"},
			{Field: "roomId"},
		},
	}

	upserter := NewBatchUpsert(nil, config)

	// 第一次调用 - 会进行转换
	result1 := upserter.resolveFieldName("deviceCode")
	assert.Equal(t, "device_code", result1)

	// 第二次调用 - 应该从缓存获取
	result2 := upserter.resolveFieldName("deviceCode")
	assert.Equal(t, "device_code", result2)

	// 验证缓存大小
	upserter.fieldNameMutex.RLock()
	cacheSize := len(upserter.fieldNameCache)
	upserter.fieldNameMutex.RUnlock()

	assert.Equal(t, 1, cacheSize, "缓存应该只包含一个条目")
}

// BenchmarkResolveFieldName 性能测试 - 验证缓存有效性
func BenchmarkResolveFieldName(b *testing.B) {
	config := ExcelConfig{
		Columns: []ExcelColumn{
			{Field: "deviceCode", DBField: "device_code"},
			{Field: "name"},
			{Field: "roomId"},
			{Field: "serialNumber"},
			{Field: "purchaseDate"},
		},
	}

	upserter := NewBatchUpsert(nil, config)

	fields := []string{"deviceCode", "name", "roomId", "serialNumber", "purchaseDate"}

	b.Run("WithCache", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, field := range fields {
				_ = upserter.resolveFieldName(field)
			}
		}
	})
}
