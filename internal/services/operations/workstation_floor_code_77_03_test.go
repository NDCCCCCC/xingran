package operations

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	operationsmodels "github.com/xingran-next/xingran-go-backend/internal/models/operations"
)

// =====================================================================
// Phase 77-03 Task 2/3 — workstation/floor/code_generator 卫星测试
//
// 覆盖矩阵:
//   - workstation: Create/GetByID/Delete/BatchDelete + List 6 表 JOIN
//     (name/floorId/floorCode/status/type/orgId 过滤分支)
//   - workstation BatchUpdatePositions CASE WHEN + CAST + optional nil
//   - workstation GetWorkstationDeptOptions / SearchWorkstationOptions
//   - workstation Statistics
//   - floor: Create 正常/唯一约束/NOW() 软删恢复分支 (D-03 锁定现行为)
//   - floor: Update syncWorkstationBuildingID + Delete + GetTree + BatchDelete
//   - code_generator: 空表 → -001 / 递增 / GenerateCodeWithCustomPrefix
//     / Sscanf 非数字 → 回 1 (Q-77-D ErrRecordNotFound 死分支按现行为锁定)
//
// 平台限制 (P-77-2): sqlite 无 NOW() 函数, 软删恢复 UPDATE 必报
// "no such function: NOW" → 仅断言错误信息含「恢复楼层失败」即止。
// =====================================================================

// setupWSFloorCode77DB 工位/楼层 6 表 fixture。
func setupWSFloorCode77DB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	require.NoError(t, db.AutoMigrate(
		&operationsmodels.OpsBuilding{},
		&operationsmodels.OpsFloor{},
		&models.Workstation{},
		&models.Department{},
	))
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_user (
			id TEXT PRIMARY KEY, username TEXT, nickname TEXT,
			deleted_at DATETIME, created_at DATETIME, updated_at DATETIME
		)`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_files (
			id TEXT PRIMARY KEY, file_name TEXT, storage_path TEXT,
			deleted_at DATETIME, created_at DATETIME
		)`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE ops_workstation_device (
			id TEXT PRIMARY KEY, workstation_id TEXT, is_primary INTEGER, priority INTEGER,
			device_serial TEXT, created_at DATETIME, deleted_at DATETIME
		)`).Error)
	// P-77-7: 唯一索引对齐生产 PG 模式 (building_id, floor_no) UNIQUE
	require.NoError(t, db.Exec(`CREATE UNIQUE INDEX ux_test_floors_bno ON ops_floors(building_id, floor_no)`).Error)
	return db
}

// TestImp77_WorkstationCRUD 覆盖 workstation CRUD + 6 表 JOIN List 过滤分支。
func TestImp77_WorkstationCRUD(t *testing.T) {
	db := setupWSFloorCode77DB(t)
	buildingID, floorID := seedBuildingFloor(t, db, "ws-b")
	ctx := context.Background()
	svc := NewWorkstationService(db)

	// Create
	ws := &models.Workstation{
		WorkstationName: "工位-101",
		WorkstationType: models.WorkstationTypeFixed,
		Status:          models.WorkstationStatusAvailable,
		FloorID:         &floorID,
	}
	require.NoError(t, svc.Create(ctx, ws))

	// GetByID: 6 表 JOIN 应成功
	got, err := svc.GetByID(ctx, ws.ID)
	require.NoError(t, err)
	assert.Equal(t, "工位-101", got.WorkstationName)
	assert.Equal(t, models.WorkstationStatusAvailable, got.Status)
	assert.NotNil(t, got.FloorID)

	// GetByID 不存在 → 报错
	_, err = svc.GetByID(ctx, "ghost-id")
	require.Error(t, err)

	// Create 第二条用于 List 过滤测试
	// 用 UserID 触发 applyWorkstationOccupancyLink → status 联动为 Occupied(1)
	userID := "user-001"
	ws2 := &models.Workstation{
		WorkstationName: "工位-102",
		WorkstationType: models.WorkstationTypeHotDesk,
		Status:          models.WorkstationStatusOccupied,
		FloorID:         &floorID,
		UserID:          &userID,
	}
	require.NoError(t, svc.Create(ctx, ws2))
	require.NoError(t, db.Exec(`INSERT INTO sys_user (id, username) VALUES ('user-001', 'alice')`).Error)

	t.Run("List name LIKE", func(t *testing.T) {
		page, err := svc.List(ctx, map[string]interface{}{"name": "工位-1"})
		require.NoError(t, err)
		assert.Equal(t, int64(2), page.Total)
	})

	t.Run("List name LIKE 不命中", func(t *testing.T) {
		page, err := svc.List(ctx, map[string]interface{}{"name": "ghost"})
		require.NoError(t, err)
		assert.Zero(t, page.Total)
	})

	t.Run("List floorId 等值", func(t *testing.T) {
		page, err := svc.List(ctx, map[string]interface{}{"floorId": floorID})
		require.NoError(t, err)
		assert.Equal(t, int64(2), page.Total)
	})

	t.Run("List status 等值", func(t *testing.T) {
		occupied := int(models.WorkstationStatusOccupied)
		page, err := svc.List(ctx, map[string]interface{}{"status": occupied})
		require.NoError(t, err)
		assert.Equal(t, int64(1), page.Total)
	})

	t.Run("List type 等值", func(t *testing.T) {
		fixed := int(models.WorkstationTypeFixed)
		page, err := svc.List(ctx, map[string]interface{}{"type": fixed})
		require.NoError(t, err)
		assert.Equal(t, int64(1), page.Total)
	})

	t.Run("List 状态 status < 0 → 跳过 status 过滤", func(t *testing.T) {
		// status = -1 (extractor fallback) → 不应用 status 过滤
		page, err := svc.List(ctx, map[string]interface{}{"status": -1})
		require.NoError(t, err)
		assert.Equal(t, int64(2), page.Total)
	})

	t.Run("List floorCode 通过 ops_floors 子查询", func(t *testing.T) {
		page, err := svc.List(ctx, map[string]interface{}{"floorCode": "1"})
		require.NoError(t, err)
		assert.Equal(t, int64(2), page.Total)
	})

	t.Run("List orgId EXISTS 子查询", func(t *testing.T) {
		// buildingID 关联的 building 的 org_id 缺省 — 验证 EXISTS 路径不报错
		page, err := svc.List(ctx, map[string]interface{}{"orgId": buildingID})
		require.NoError(t, err)
		// org_id 缺省, 命中 0
		_ = page
	})

	// Update (createdAt 守护)
	got.WorkstationName = "工位-101-改"
	require.NoError(t, svc.Update(ctx, got))

	// Delete
	require.NoError(t, svc.Delete(ctx, ws2.ID))

	// BatchDelete
	require.NoError(t, svc.BatchDelete(ctx, []string{ws.ID}))
	require.NoError(t, svc.BatchDelete(ctx, nil))
}

// TestImp77_WorkstationOptions 覆盖 GetWorkstationDeptOptions + SearchWorkstationOptions。
func TestImp77_WorkstationOptions(t *testing.T) {
	db := setupWSFloorCode77DB(t)
	buildingID, floorID := seedBuildingFloor(t, db, "ws-b")
	ctx := context.Background()
	svc := NewWorkstationService(db)

	// Create 3 个工位
	for i, name := range []string{"工位-A", "工位-B", "工位-C"} {
		wsType := models.WorkstationTypeFixed
		if i == 1 {
			wsType = models.WorkstationTypeHotDesk
		}
		require.NoError(t, svc.Create(ctx, &models.Workstation{
			WorkstationName: name,
			WorkstationType: wsType,
			FloorID:         &floorID,
			Status:          models.WorkstationStatusAvailable,
		}))
	}

	// GetWorkstationDeptOptions: 空 orgId → 空列表
	opts, err := svc.GetWorkstationDeptOptions(ctx, "")
	require.NoError(t, err)
	assert.Empty(t, opts)

	// GetWorkstationDeptOptions: 真实 orgId → UNION 子孙 + alias
	// 但因 sqlite 没 sys_dept_location_alias 表 → JOIN 该表会 no such table
	// 故用 orgId="" 路径断言无异常 + opts 为空 slice
	_ = opts
	_ = buildingID

	// SearchWorkstationOptions
	t.Run("name LIKE", func(t *testing.T) {
		opts, err := svc.SearchWorkstationOptions(ctx, map[string]interface{}{"name": "工位"})
		require.NoError(t, err)
		assert.Len(t, opts, 3)
	})

	t.Run("floorId 等值", func(t *testing.T) {
		opts, err := svc.SearchWorkstationOptions(ctx, map[string]interface{}{"floorId": floorID})
		require.NoError(t, err)
		assert.Len(t, opts, 3)
	})

	t.Run("status 等值", func(t *testing.T) {
		available := int(models.WorkstationStatusAvailable)
		opts, err := svc.SearchWorkstationOptions(ctx, map[string]interface{}{"status": available})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(opts), 1)
	})

	t.Run("type 等值", func(t *testing.T) {
		fixed := int(models.WorkstationTypeFixed)
		opts, err := svc.SearchWorkstationOptions(ctx, map[string]interface{}{"type": fixed})
		require.NoError(t, err)
		assert.Equal(t, 2, len(opts))
	})

	t.Run("floorCode 子查询", func(t *testing.T) {
		opts, err := svc.SearchWorkstationOptions(ctx, map[string]interface{}{"floorCode": "1"})
		require.NoError(t, err)
		assert.Len(t, opts, 3)
	})

	t.Run("status=-1 跳过", func(t *testing.T) {
		opts, err := svc.SearchWorkstationOptions(ctx, map[string]interface{}{"status": -1})
		require.NoError(t, err)
		assert.Len(t, opts, 3)
	})
}

// TestImp77_WorkstationStatistics 覆盖 Statistics 三态聚合。
func TestImp77_WorkstationStatistics(t *testing.T) {
	db := setupWSFloorCode77DB(t)
	_, floorID := seedBuildingFloor(t, db, "ws-b")
	ctx := context.Background()
	svc := NewWorkstationService(db)

	// Create 一空闲 + 一占用
	require.NoError(t, svc.Create(ctx, &models.Workstation{
		WorkstationName: "空闲位", FloorID: &floorID,
		Status: models.WorkstationStatusAvailable,
	}))
	userID := "u1"
	require.NoError(t, svc.Create(ctx, &models.Workstation{
		WorkstationName: "占用位", FloorID: &floorID,
		Status: models.WorkstationStatusOccupied, UserID: &userID,
	}))

	stats, err := svc.Statistics(ctx, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(2), stats.Total)
	assert.GreaterOrEqual(t, stats.Available, int64(1))
	assert.GreaterOrEqual(t, stats.Occupied, int64(1))
}

// TestImp77_BatchUpdatePositions 覆盖 BatchUpdatePositions CASE WHEN + 可选 nil。
func TestImp77_BatchUpdatePositions(t *testing.T) {
	db := setupWSFloorCode77DB(t)
	_, floorID := seedBuildingFloor(t, db, "ws-b")
	ctx := context.Background()
	svc := NewWorkstationService(db)

	// Create 3 个工位
	ids := make([]string, 3)
	for i, name := range []string{"位-A", "位-B", "位-C"} {
		ws := &models.Workstation{
			WorkstationName: name, FloorID: &floorID,
			Status: models.WorkstationStatusAvailable,
		}
		require.NoError(t, svc.Create(ctx, ws))
		ids[i] = ws.ID
	}

	// 必填字段 + 全部 optional 字段 nil (rotation/width/depth/deskType)
	items := []PositionUpdateItem{
		{ID: ids[0], PositionX: 100, PositionY: 200},
		{ID: ids[1], PositionX: 300, PositionY: 400},
		{ID: ids[2], PositionX: 500, PositionY: 600},
	}
	require.NoError(t, svc.BatchUpdatePositions(ctx, items))

	// 断言位置落库
	for i, item := range items {
		var ws models.Workstation
		require.NoError(t, db.First(&ws, "id = ?", item.ID).Error)
		assert.Equal(t, item.PositionX, *ws.PositionX, "idx %d PositionX", i)
		assert.Equal(t, item.PositionY, *ws.PositionY, "idx %d PositionY", i)
	}

	// optional 字段带值 → 应落库
	rotation := 45
	width := 1800
	depth := 800
	deskType := 1
	items2 := []PositionUpdateItem{
		{
			ID: ids[0], PositionX: 1000, PositionY: 2000,
			Rotation: &rotation, Width: &width, Depth: &depth, DeskType: &deskType,
		},
	}
	require.NoError(t, svc.BatchUpdatePositions(ctx, items2))

	var ws models.Workstation
	require.NoError(t, db.First(&ws, "id = ?", ids[0]).Error)
	assert.Equal(t, rotation, *ws.Rotation)
	assert.Equal(t, width, *ws.Width)
	assert.Equal(t, depth, *ws.Depth)
	assert.Equal(t, deskType, *ws.DeskType)

	// 空列表 no-op
	require.NoError(t, svc.BatchUpdatePositions(ctx, nil))
}

// TestImp77_WorkstationSearchOrgId 验证 orgId EXISTS 子查询 (org_id 为 NULL 时不命中)。
func TestImp77_WorkstationSearchOrgId(t *testing.T) {
	db := setupWSFloorCode77DB(t)
	_, floorID := seedBuildingFloor(t, db, "ws-b")
	ctx := context.Background()
	svc := NewWorkstationService(db)

	require.NoError(t, svc.Create(ctx, &models.Workstation{
		WorkstationName: "OrgTest", FloorID: &floorID,
		Status: models.WorkstationStatusAvailable,
	}))

	// orgId 不存在 → 0 条
	opts, err := svc.SearchWorkstationOptions(ctx, map[string]interface{}{"orgId": "ghost-org"})
	require.NoError(t, err)
	assert.Empty(t, opts)

	page, err := svc.List(ctx, map[string]interface{}{"orgId": "ghost-org"})
	require.NoError(t, err)
	assert.Zero(t, page.Total)
}

// =====================================================================
// Floor
// =====================================================================

// TestImp77_FloorCreate 覆盖 Create 正常 + 唯一约束 + sqlite 软删恢复 NOW() 报错。
func TestImp77_FloorCreate(t *testing.T) {
	db := setupWSFloorCode77DB(t)
	buildingID, _ := seedBuildingFloor(t, db, "ws-b")
	ctx := context.Background()
	svc := NewFloorService(db)

	// 正常 Create
	f := &operationsmodels.OpsFloor{Name: "新楼层", FloorNo: "2", BuildingID: buildingID}
	require.NoError(t, svc.Create(ctx, f))
	assert.NotEmpty(t, f.ID, "create 后 ID 应被填充")

	// 软删后再次 Create 同名 + 同 building + 同 floor_no
	require.NoError(t, svc.Delete(ctx, f.ID))

	// P-77-2: sqlite 无 NOW() 函数 → 恢复 UPDATE 必报「恢复楼层失败」
	f2 := &operationsmodels.OpsFloor{Name: "新楼层", FloorNo: "2", BuildingID: buildingID}
	err := svc.Create(ctx, f2)
	require.Error(t, err, "sqlite 软删恢复分支不可达(P-77-2)")
	assert.Contains(t, err.Error(), "恢复楼层失败",
		"错误信息应明确归因到恢复分支")

	// 唯一约束错误(building_id 缺省 → validateBuilding 报错)
	require.Error(t, svc.Create(ctx, &operationsmodels.OpsFloor{Name: "无楼宇", FloorNo: "3", BuildingID: ""}))

	// 楼宇不存在
	require.Error(t, svc.Create(ctx, &operationsmodels.OpsFloor{Name: "幽灵", FloorNo: "3", BuildingID: "ghost-b"}))
}

// TestImp77_FloorGetTree_GetByID_List 覆盖 floor List/GetTree/GetByID。
func TestImp77_FloorGetTree_GetByID_List(t *testing.T) {
	db := setupWSFloorCode77DB(t)
	buildingID, f1 := seedBuildingFloor(t, db, "ws-b")
	ctx := context.Background()
	svc := NewFloorService(db)

	// Add 第二楼层
	f2 := &operationsmodels.OpsFloor{Name: "ws-b-F2", FloorNo: "2", BuildingID: buildingID}
	require.NoError(t, svc.Create(ctx, f2))

	// GetByID
	got, err := svc.GetByID(ctx, f1)
	require.NoError(t, err)
	assert.Equal(t, f1, got.ID)

	// GetTree
	tree, err := svc.GetTree(ctx)
	require.NoError(t, err)
	require.Len(t, tree, 1)
	assert.Equal(t, buildingID, tree[0].ID)
	assert.Len(t, tree[0].Children, 2)

	// List name LIKE
	page, err := svc.List(ctx, map[string]interface{}{"name": "ws-b"})
	require.NoError(t, err)
	assert.Equal(t, int64(2), page.Total)

	// List buildingId 等值
	page, err = svc.List(ctx, map[string]interface{}{"buildingId": buildingID})
	require.NoError(t, err)
	assert.Equal(t, int64(2), page.Total)

	// List orgId EXISTS 子查询
	page, err = svc.List(ctx, map[string]interface{}{"orgId": buildingID})
	require.NoError(t, err)
	// org_id 为 NULL → 不命中
	_ = page

	// List status 等值
	normal := 0
	page, err = svc.List(ctx, map[string]interface{}{"status": normal})
	require.NoError(t, err)
	assert.Equal(t, int64(2), page.Total)

	// SearchFloorOptions
	opts, err := svc.SearchFloorOptions(ctx, map[string]interface{}{"name": "ws-b"})
	require.NoError(t, err)
	assert.Len(t, opts, 2)

	// Statistics
	stats, err := svc.Statistics(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), stats.Total)
	assert.Equal(t, int64(2), stats.Active)

	// Update
	f2.Name = "ws-b-F2-改"
	require.NoError(t, svc.Update(ctx, f2))

	// BatchDelete
	require.NoError(t, svc.BatchDelete(ctx, []string{f1, f2.ID}))
	require.NoError(t, svc.BatchDelete(ctx, nil))
}

// TestImp77_FloorSyncWorkstationBuildingID 验证 syncWorkstationBuildingID (白盒直调)。
func TestImp77_FloorSyncWorkstationBuildingID(t *testing.T) {
	db := setupWSFloorCode77DB(t)
	buildingID, f1 := seedBuildingFloor(t, db, "ws-b")
	ctx := context.Background()
	svc := NewFloorService(db).(*floorService)

	// Create 一个工位绑定到 f1,无 building_id
	require.NoError(t, db.Create(&models.Workstation{
		WorkstationName: "sync-test", FloorID: &f1,
	}).Error)

	// 调 syncWorkstationBuildingID → 应把 workstation.building_id 设为新值
	require.NoError(t, svc.syncWorkstationBuildingID(ctx, f1, buildingID))

	var ws models.Workstation
	require.NoError(t, db.Where("workstation_name = ?", "sync-test").First(&ws).Error)
	assert.Equal(t, buildingID, *ws.BuildingID)
}

// =====================================================================
// CodeGenerator (Q-77-D: ErrRecordNotFound 死分支按现行为锁定)
// =====================================================================

// TestImp77_GenerateCode 覆盖空表/递增/非数字/GenerateCodeWithCustomPrefix。
func TestImp77_GenerateCode(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE test_codes (id INTEGER PRIMARY KEY, code TEXT)`).Error)

	g := NewCodeGenerator(db)
	ctx := context.Background()

	t.Run("空表 → -001", func(t *testing.T) {
		code, err := g.GenerateCode(ctx, CodeTypeBuilding, "test_codes", "code")
		require.NoError(t, err)
		// yearMonth 动态, 形态断言: 包含 "-001"
		assert.Contains(t, code, "-001")
	})

	t.Run("预置当前月 007 → 应 -008", func(t *testing.T) {
		// 用独立表避免与其他测试相互污染
		// 使用当前年月保持跨跨跨跨跨跨跨跨跨月稳定
		currentYM := time.Now().Format("200601")
		require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS test_codes_inc (id INTEGER PRIMARY KEY, code TEXT)`).Error)
		require.NoError(t, db.Exec(fmt.Sprintf(`INSERT INTO test_codes_inc (code) VALUES ('BLD-%s-007')`, currentYM)).Error)
		code, err := g.GenerateCode(ctx, CodeTypeBuilding, "test_codes_inc", "code")
		require.NoError(t, err)
		assert.Contains(t, code, "-008")
	})

	t.Run("serial 非数字 → 回 1", func(t *testing.T) {
		currentYM := time.Now().Format("200601")
		require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS test_codes_str (id INTEGER PRIMARY KEY, code TEXT)`).Error)
		require.NoError(t, db.Exec(fmt.Sprintf(`INSERT INTO test_codes_str (code) VALUES ('BLD-%s-abc')`, currentYM)).Error)
		code, err := g.GenerateCode(ctx, CodeTypeBuilding, "test_codes_str", "code")
		require.NoError(t, err)
		assert.Contains(t, code, "-001", "Sscanf 失败回退 nextSerial=1")
	})

	t.Run("GenerateCodeWithCustomPrefix 递增", func(t *testing.T) {
		require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS test_codes_prefix (id INTEGER PRIMARY KEY, code TEXT)`).Error)
		require.NoError(t, db.Exec(`INSERT INTO test_codes_prefix (code) VALUES ('ROM-005')`).Error)
		code, err := g.GenerateCodeWithCustomPrefix(ctx, "ROM", "test_codes_prefix", "code")
		require.NoError(t, err)
		assert.Equal(t, "ROM-006", code)
	})

	t.Run("GenerateCodeWithCustomPrefix 空表 → -001", func(t *testing.T) {
		require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS test_codes_empty (id INTEGER PRIMARY KEY, code TEXT)`).Error)
		code, err := g.GenerateCodeWithCustomPrefix(ctx, "DEV", "test_codes_empty", "code")
		require.NoError(t, err)
		assert.Equal(t, "DEV-001", code)
	})

	t.Run("GenerateCodeWithCustomPrefix 非数字 → 回 1", func(t *testing.T) {
		require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS test_codes_prefix_str (id INTEGER PRIMARY KEY, code TEXT)`).Error)
		require.NoError(t, db.Exec(`INSERT INTO test_codes_prefix_str (code) VALUES ('ROM-xyz')`).Error)
		code, err := g.GenerateCodeWithCustomPrefix(ctx, "ROM", "test_codes_prefix_str", "code")
		require.NoError(t, err)
		assert.Equal(t, "ROM-001", code)
	})
}
