package migrations

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// freshSQLiteDBForMigrate208 构建内存 SQLite 库 (字典域最小表集)。
func freshSQLiteDBForMigrate208(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared&_busy_timeout=5000"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&models.DictType{},
		&models.DictData{},
	); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	return db
}

// expected208GroupSizes 11 组 × 组内值数 (硬编码锁形: dictSeedGroups 的意外
// 增删组/改值数会与本表失配而 fail, 逼改动者显式更新此表)。
var expected208GroupSizes = map[string]int64{
	"network_device_type":                   5,
	"ops_dedicated_line_type":               6,
	"ops_isp":                               5,
	"ops_info_point_type":                   2,
	"asset_reconciliation_conflict_type":    6,
	"asset_reconciliation_exception_action": 5,
	"asset_reconciliation_severity":         4,
	"asset_reconciliation_status":           2,
	"ops_workstation_type":                  3,
	"sys_user_sex":                          3,
	"duty_holiday_type":                     3,
}

// expected208DataTotal 44 = 11 组值数之和。
const expected208DataTotal int64 = 44

// dictTypeCount / dictDataCount 统计存活行数。
func dictTypeCount(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var n int64
	if err := db.Model(&models.DictType{}).Count(&n).Error; err != nil {
		t.Fatalf("count sys_dict_type: %v", err)
	}
	return n
}

func dictDataCount(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var n int64
	if err := db.Model(&models.DictData{}).Count(&n).Error; err != nil {
		t.Fatalf("count sys_dict_data: %v", err)
	}
	return n
}

// TestMigrate208SeedsAndIdempotent 空库: seed 11 组 / 44 值, 每组行数与
// 期望表一致; 第二遍运行行数完全不变 (幂等)。
func TestMigrate208SeedsAndIdempotent(t *testing.T) {
	db := freshSQLiteDBForMigrate208(t)

	if len(dictSeedGroups) != len(expected208GroupSizes) {
		t.Fatalf("dictSeedGroups has %d groups, want %d (update expected208GroupSizes)",
			len(dictSeedGroups), len(expected208GroupSizes))
	}

	if err := Migrate208DictSeed(db); err != nil {
		t.Fatalf("Migrate208 first run: %v", err)
	}

	if got := dictTypeCount(t, db); got != int64(len(expected208GroupSizes)) {
		t.Fatalf("sys_dict_type rows = %d, want %d", got, len(expected208GroupSizes))
	}
	if got := dictDataCount(t, db); got != expected208DataTotal {
		t.Fatalf("sys_dict_data rows = %d, want %d", got, expected208DataTotal)
	}
	for key, want := range expected208GroupSizes {
		var got int64
		if err := db.Model(&models.DictData{}).Where("dict_type = ?", key).Count(&got).Error; err != nil {
			t.Fatalf("count dict_data for %s: %v", key, err)
		}
		if got != want {
			t.Fatalf("dict_data rows for %s = %d, want %d", key, got, want)
		}
	}

	// 幂等: 第二遍运行后两个行数断言完全不变
	if err := Migrate208DictSeed(db); err != nil {
		t.Fatalf("Migrate208 second run: %v", err)
	}
	if got := dictTypeCount(t, db); got != int64(len(expected208GroupSizes)) {
		t.Fatalf("after re-run sys_dict_type rows = %d, want %d", got, len(expected208GroupSizes))
	}
	if got := dictDataCount(t, db); got != expected208DataTotal {
		t.Fatalf("after re-run sys_dict_data rows = %d, want %d", got, expected208DataTotal)
	}
}

// TestMigrate208RespectsExistingGroups 预插 dict_type 已存在的组 (管理员自建):
// seed 整组跳过不写入、不复活; 其余 10 组正常 seed。
func TestMigrate208RespectsExistingGroups(t *testing.T) {
	db := freshSQLiteDBForMigrate208(t)

	// 模拟管理员自建组: ops_isp 已有类型 + 1 条自定义值
	existingType := models.DictType{
		DictName: "运营商-自建",
		DictType: "ops_isp",
		Status:   int(models.DictStatusNormal),
	}
	if err := db.Create(&existingType).Error; err != nil {
		t.Fatalf("create existing dict_type: %v", err)
	}
	customData := models.DictData{
		DictSort: 1, DictLabel: "自建运营商", DictValue: "custom_isp",
		DictType: "ops_isp", IsDefault: true, Status: int(models.DictStatusNormal),
	}
	if err := db.Create(&customData).Error; err != nil {
		t.Fatalf("create custom dict_data: %v", err)
	}

	if err := Migrate208DictSeed(db); err != nil {
		t.Fatalf("Migrate208: %v", err)
	}

	// 该组 DictData 仍为 1 条 (seed 未写入、未复活)
	var ispRows int64
	if err := db.Model(&models.DictData{}).Where("dict_type = ?", "ops_isp").Count(&ispRows).Error; err != nil {
		t.Fatalf("count ops_isp rows: %v", err)
	}
	if ispRows != 1 {
		t.Fatalf("ops_isp dict_data rows = %d, want 1 (seed must not touch existing group)", ispRows)
	}
	var seedValueLeak int64
	if err := db.Model(&models.DictData{}).
		Where("dict_type = ? AND dict_value = ?", "ops_isp", "telecom").Count(&seedValueLeak).Error; err != nil {
		t.Fatalf("count leaked telecom row: %v", err)
	}
	if seedValueLeak != 0 {
		t.Fatalf("seed wrote telecom into admin-owned ops_isp group (leak = %d)", seedValueLeak)
	}

	// 其余 10 组正常 seed: 11 类型 / 44 - 5 (ops_isp seed 值) + 1 (自建) = 40 值
	if got := dictTypeCount(t, db); got != int64(len(expected208GroupSizes)) {
		t.Fatalf("sys_dict_type rows = %d, want %d", got, len(expected208GroupSizes))
	}
	if got := dictDataCount(t, db); got != expected208DataTotal-5+1 {
		t.Fatalf("sys_dict_data rows = %d, want %d", got, expected208DataTotal-5+1)
	}
}

// TestMigrate208IsDefaultSemantics 每组恰好一条 IsDefault=true 的 DictData
// (消费端 isDefault 默认值逻辑的数据前提, dedicated-lines 三件套依赖)。
func TestMigrate208IsDefaultSemantics(t *testing.T) {
	db := freshSQLiteDBForMigrate208(t)

	if err := Migrate208DictSeed(db); err != nil {
		t.Fatalf("Migrate208: %v", err)
	}

	for key := range expected208GroupSizes {
		var defaults int64
		if err := db.Model(&models.DictData{}).
			Where("dict_type = ? AND is_default = ?", key, true).
			Count(&defaults).Error; err != nil {
			t.Fatalf("count defaults for %s: %v", key, err)
		}
		if defaults != 1 {
			t.Fatalf("group %s has %d IsDefault=true rows, want exactly 1", key, defaults)
		}
	}
}

// TestMigrate208RespectsSoftDeletedGroups 管理员软删的组不被 seed 复活
// (组级查重走 Unscoped; 软删行占位 uniqueIndex, 复活插入也必然撞约束)。
func TestMigrate208RespectsSoftDeletedGroups(t *testing.T) {
	db := freshSQLiteDBForMigrate208(t)

	if err := Migrate208DictSeed(db); err != nil {
		t.Fatalf("Migrate208 first run: %v", err)
	}

	// 模拟管理员删除整组 (DictType 带 gorm.DeletedAt, Delete 即软删)
	var victim models.DictType
	if err := db.Where("dict_type = ?", "ops_info_point_type").First(&victim).Error; err != nil {
		t.Fatalf("find victim dict_type: %v", err)
	}
	if err := db.Delete(&victim).Error; err != nil {
		t.Fatalf("soft-delete victim: %v", err)
	}

	if err := Migrate208DictSeed(db); err != nil {
		t.Fatalf("Migrate208 re-run after soft-delete must not error: %v", err)
	}

	var alive int64
	if err := db.Model(&models.DictType{}).Where("dict_type = ?", "ops_info_point_type").Count(&alive).Error; err != nil {
		t.Fatalf("count alive victim: %v", err)
	}
	if alive != 0 {
		t.Fatalf("soft-deleted dict_type ops_info_point_type was resurrected by seed")
	}
	if got := dictTypeCount(t, db); got != int64(len(expected208GroupSizes))-1 {
		t.Fatalf("after re-run sys_dict_type rows = %d, want %d", got, len(expected208GroupSizes)-1)
	}
}
