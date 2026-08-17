package operations

import (
	"context"
	"testing"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/stretchr/testify/require"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// TestWorkstationUpdate_PreservesCreatedAt
// 回归测试:验证 workstationService.Update() 在 handler 传入零值 CreatedAt 的场景下,
// 不会用 0001-01-01 覆盖库中原有的 created_at。
//
// 根因(详见 .planning/debug/workstation-update-createdat-zeroed.md):
//   - handler(internal/api/v1/operations/workstation_handler.go:117-123) 把请求体
//     反序列化到 `var workstation models.Workstation`,只设置 ID,CreatedAt 是零值
//     time.Time(非指针,见 internal/models/base.go:13)。
//   - 修复前 service.Update 直接 Save,GORM 全量写库把 created_at 覆盖为 0001-01-01。
//   - 修复后:Save 前 First 查 existing 并回填 CreatedAt/CreatedBy。
//
// 用 sqlite 内存库重现(参考 internal/services/system/user_list_status_test.go 的建表风格)。
func TestWorkstationUpdate_PreservesCreatedAt(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "create sqlite test db")

	// 建 sys_workstation 表,只列本测试关心 + Workstation struct 反序列化会用到的字段。
	// GORM Save 在 sqlite 上会用 struct field→column 映射,缺列会报错,所以列要齐全。
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_workstation (
			id TEXT PRIMARY KEY,
			workstation_name TEXT,
			workstation_type INTEGER DEFAULT 0,
			status INTEGER DEFAULT 0,
			description TEXT,
			dept_id TEXT,
			dept_name TEXT,
			building_id TEXT,
			building_name TEXT,
			location TEXT,
			floor TEXT,
			floor_id TEXT,
			floor_name TEXT,
			user_id TEXT,
			user_name TEXT,
			position_x INTEGER,
			position_y INTEGER,
			rotation INTEGER,
			width INTEGER,
			depth INTEGER,
			desk_type INTEGER,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER DEFAULT 0
		)
	`).Error, "create sys_workstation table")

	// 插入一条已存在记录,created_at 设为 2024-06-01(非零)。
	originalCreatedAt := "2024-06-01 10:30:00"
	originalCreatedBy := "creator-user-id"
	require.NoError(t, db.Exec(`
		INSERT INTO sys_workstation (
			id, workstation_name, workstation_type, status,
			created_at, updated_at, created_by, version
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, "ws-1", "工位A", 0, 0, originalCreatedAt, originalCreatedAt, originalCreatedBy, 1).Error,
		"seed existing workstation")

	// 构造入参:模拟 handler 反序列化的对象 —— ID 匹配,CreatedAt 是零值(0001-01-01)。
	// WorkstationName 改一下,确保 Save 确实写库(updated_at 应刷新)。
	input := &models.Workstation{
		BaseModel: models.BaseModel{
			ID:        "ws-1",
			CreatedAt: time.Time{}, // 零值,这是 bug 触发条件
		},
		WorkstationName: "工位A-改名",
	}

	svc := &workstationService{db: db}
	require.NoError(t, svc.Update(context.Background(), input), "Update should succeed")

	// 关键断言:库中 created_at 仍是原非零值,未被零值覆盖。
	var got models.Workstation
	require.NoError(t, db.First(&got, "id = ?", "ws-1").Error, "re-read updated row")

	require.NotEqual(t, time.Time{}, got.CreatedAt,
		"REGRESSION: created_at was zeroed by Update() — bug returned")
	require.Equal(t, "2024-06-01", got.CreatedAt.Format("2006-01-02"),
		"created_at should be preserved at original 2024-06-01, got %v", got.CreatedAt)
	require.Equal(t, originalCreatedBy, got.CreatedBy,
		"created_by should be preserved, got %q", got.CreatedBy)

	// 业务字段确实更新了
	require.Equal(t, "工位A-改名", got.WorkstationName,
		"workstation_name should reflect the update")

	// updated_at 应该被 GORM 钩子刷新为较新的时间(≥ 原 created_at)
	require.True(t, got.UpdatedAt.After(got.CreatedAt) || got.UpdatedAt.Equal(got.CreatedAt),
		"updated_at should be >= created_at; updated_at=%v created_at=%v",
		got.UpdatedAt, got.CreatedAt)

	t.Logf("OK: created_at preserved=%v, created_by=%v, updated_at=%v, name=%q",
		got.CreatedAt, got.CreatedBy, got.UpdatedAt, got.WorkstationName)
}

// TestWorkstationUpdate_RecordNotFound
// 修复后 First 查询若记录不存在应返回错误(而不是悄悄 Save 一条新记录)。
func TestWorkstationUpdate_RecordNotFound(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "create sqlite test db")
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_workstation (
			id TEXT PRIMARY KEY,
			workstation_name TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER DEFAULT 0
		)
	`).Error, "create sys_workstation table")

	svc := &workstationService{db: db}
	input := &models.Workstation{
		BaseModel:      models.BaseModel{ID: "does-not-exist"},
		WorkstationName: "ghost",
	}
	err = svc.Update(context.Background(), input)
	require.Error(t, err, "Update on non-existent record should return error")
	require.ErrorIs(t, err, gorm.ErrRecordNotFound,
		"expected gorm.ErrRecordNotFound, got %v", err)
}
