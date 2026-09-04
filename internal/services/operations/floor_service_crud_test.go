package operations

import (
	"context"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	sysmodels "github.com/xingran-next/xingran-go-backend/internal/models"
	operationsmodels "github.com/xingran-next/xingran-go-backend/internal/models/operations"
	"github.com/xingran-next/xingran-go-backend/internal/models/system"
)

// =====================================================================
// Phase 91-03 Task 1: floor_service 级行为基线测试（迁移前先绿，Wave 0 缺口补齐）。
// floor 的 GetByID/List 管道即将迁入 base.GORMRepository[OpsFloor]（Task 3），
// 本文件锁定迁移前后必须逐字节同结果的全部行为：CRUD / 软删恢复分支 /
// 异步工位同步 / 楼层计数 / List 过滤矩阵 / 无 clamp 分页（P5 floor 语义）。
// =====================================================================

// newFloor91TestDB floor 基线库：ops_buildings + ops_floors + sys_workstation + sys_files。
// sys_files 为 GetByID/List 的 plan_image_url LEFT JOIN 目标；sys_workstation 为
// Update 异步 syncWorkstationBuildingID 的目标表。
// 用 t.TempDir 文件库而非 :memory:——Update 的异步 goroutine 与测试主 goroutine 并发
// 访问连接池，glebarez :memory: 每条连接是独立空库（见 geocoding_photo_floor_test.go
// 的 DeletePhoto quirk 注释），异步同步会被路由到空库静默失败（best-effort 吞错）。
func newFloor91TestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "floor91.db")), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	// Windows 下 sqlite 文件句柄未释放会导致 t.TempDir RemoveAll 失败——测试结束显式关库。
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	require.NoError(t, db.AutoMigrate(
		&operationsmodels.OpsBuilding{},
		&operationsmodels.OpsFloor{},
		&sysmodels.Workstation{},
		&system.SysFile{},
	))
	return db
}

// seedFloor91Building 建一栋楼宇并返回实体（测试断言 total_floors 用）。
func seedFloor91Building(t *testing.T, db *gorm.DB, name, orgID string) *operationsmodels.OpsBuilding {
	t.Helper()
	b := &operationsmodels.OpsBuilding{Name: name, Address: "addr", Level: 2, OrgID: orgID}
	require.NoError(t, db.Create(b).Error)
	return b
}

// floor91BuildingTotalFloors 读取楼宇当前 total_floors（updateBuildingFloorCount 写入目标列）。
func floor91BuildingTotalFloors(t *testing.T, db *gorm.DB, buildingID string) int {
	t.Helper()
	var b operationsmodels.OpsBuilding
	require.NoError(t, db.Where("id = ?", buildingID).First(&b).Error)
	return b.TotalFloors
}

// TestFloor91_CRUD 覆盖 Create（validateBuilding + UUID 回填）→ GetByID（join 字段）→
// Update 改名 → Delete 软删 → GetByID ErrRecordNotFound → List 不含软删行。
func TestFloor91_CRUD(t *testing.T) {
	db := newFloor91TestDB(t)
	svc := NewFloorService(db)
	ctx := context.Background()

	b := seedFloor91Building(t, db, "B91", "org-91")
	file := &system.SysFile{FileName: "plan.png", StoragePath: "plans/91.png", UploaderID: "u1"}
	require.NoError(t, db.Create(file).Error)

	// Create：validateBuilding 通过后入库，BaseModel UUID 回填
	f := &operationsmodels.OpsFloor{Name: "F91-1", FloorNo: "1", BuildingID: b.ID, PlanImageID: &file.ID}
	require.NoError(t, svc.Create(ctx, f))
	assert.NotEmpty(t, f.ID, "create 后 BaseModel UUID 应回填")

	// Create 后楼层计数同步为 1
	assert.Equal(t, 1, floor91BuildingTotalFloors(t, db, b.ID))

	// GetByID：返回 building_name 与 plan_image_url join 字段
	got, err := svc.GetByID(ctx, f.ID)
	require.NoError(t, err)
	require.NotNil(t, got.BuildingName, "GetByID JOIN 应带出 building_name")
	assert.Equal(t, "B91", *got.BuildingName)
	require.NotNil(t, got.PlanImageUrl, "GetByID JOIN 应带出 plan_image_url")
	assert.Equal(t, "/uploads/plans/91.png", *got.PlanImageUrl)

	// Update 改名生效
	got.Name = "F91-1改"
	require.NoError(t, svc.Update(ctx, got))
	renamed, err := svc.GetByID(ctx, f.ID)
	require.NoError(t, err)
	assert.Equal(t, "F91-1改", renamed.Name)

	// Delete 软删后 GetByID 返回 ErrRecordNotFound
	require.NoError(t, svc.Delete(ctx, f.ID))
	_, err = svc.GetByID(ctx, f.ID)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)

	// List 不含软删行
	page, err := svc.List(ctx, map[string]interface{}{"buildingId": b.ID})
	require.NoError(t, err)
	assert.Equal(t, int64(0), page.Total, "软删楼层不应出现在 List")
}

// TestFloor91_Create_SoftDeleteRestore 锁定同 (building_id, floor_no) 唯一键的
// Create-after-soft-delete 当前行为：唯一约束冲突 → isDuplicateKeyError → 检索到
// 软删行 → 进入恢复分支。sqlite 无 NOW()（P-77-2，TestImp77_FloorCreate 已锁），
// 恢复 UPDATE 必失败 → 错误语义为「恢复楼层失败」。迁移后该分支路径不得改变。
func TestFloor91_Create_SoftDeleteRestore(t *testing.T) {
	db := newFloor91TestDB(t)
	// P-77-7 形状：唯一索引对齐「同楼宇同楼层号唯一」约束（生产 PG 为 migration 098
	// 的 partial unique index；sqlite 测试侧沿用既有 TestImp77 脚手架的全量唯一索引）。
	require.NoError(t, db.Exec(`CREATE UNIQUE INDEX ux_floor91_bno ON ops_floors(building_id, floor_no)`).Error)
	svc := NewFloorService(db)
	ctx := context.Background()

	b := seedFloor91Building(t, db, "B91R", "org-91")

	f := &operationsmodels.OpsFloor{Name: "F-R", FloorNo: "1", BuildingID: b.ID}
	require.NoError(t, svc.Create(ctx, f))
	require.NoError(t, svc.Delete(ctx, f.ID))

	// 同键再 Create → 唯一约束命中 → 恢复分支（sqlite 下恢复 UPDATE 因 NOW() 失败）
	f2 := &operationsmodels.OpsFloor{Name: "F-R", FloorNo: "1", BuildingID: b.ID}
	err := svc.Create(ctx, f2)
	require.Error(t, err, "唯一键命中软删行 → 恢复分支（非静默插入第二行）")
	assert.Contains(t, err.Error(), "恢复楼层失败",
		"sqlite 无 NOW() → 恢复 UPDATE 失败，错误应归因恢复分支（P-77-2）")
}

// TestFloor91_Update_SyncsWorkstation 锁定 Update 修改 floor.building_id 后
// 异步 goroutine 同步挂靠工位 building_id 的行为（项目规约：异步用 Eventually）。
func TestFloor91_Update_SyncsWorkstation(t *testing.T) {
	db := newFloor91TestDB(t)
	svc := NewFloorService(db)
	ctx := context.Background()

	b1 := seedFloor91Building(t, db, "B91A", "org-91")
	b2 := seedFloor91Building(t, db, "B91B", "org-91")
	f := &operationsmodels.OpsFloor{Name: "F-S", FloorNo: "1", BuildingID: b1.ID}
	require.NoError(t, svc.Create(ctx, f))

	ws := &sysmodels.Workstation{WorkstationName: "sync-91", FloorID: &f.ID}
	require.NoError(t, db.Create(ws).Error)

	// 楼层换楼 → 异步同步工位 building_id
	f.BuildingID = b2.ID
	require.NoError(t, svc.Update(ctx, f))

	require.Eventually(t, func() bool {
		var got sysmodels.Workstation
		if err := db.Where("id = ?", ws.ID).First(&got).Error; err != nil {
			return false
		}
		return got.BuildingID != nil && *got.BuildingID == b2.ID
	}, 2*time.Second, 20*time.Millisecond, "异步 syncWorkstationBuildingID 应把工位 building_id 更新为新楼宇")
}

// TestFloor91_Delete_UpdatesBuildingCount 锁定 Delete/BatchDelete 的
// updateBuildingFloorCount 编排：total_floors 随楼层增删同步。
func TestFloor91_Delete_UpdatesBuildingCount(t *testing.T) {
	db := newFloor91TestDB(t)
	svc := NewFloorService(db)
	ctx := context.Background()

	b := seedFloor91Building(t, db, "B91C", "org-91")
	f1 := &operationsmodels.OpsFloor{Name: "F-C1", FloorNo: "1", BuildingID: b.ID}
	f2 := &operationsmodels.OpsFloor{Name: "F-C2", FloorNo: "2", BuildingID: b.ID}
	require.NoError(t, svc.Create(ctx, f1))
	require.NoError(t, svc.Create(ctx, f2))
	require.Equal(t, 2, floor91BuildingTotalFloors(t, db, b.ID), "两次 Create 后 total_floors 应为 2")

	// Delete → 计数递减
	require.NoError(t, svc.Delete(ctx, f1.ID))
	require.Equal(t, 1, floor91BuildingTotalFloors(t, db, b.ID))

	// BatchDelete(nil) → NoError（空选择 no-op，不触碰计数）
	require.NoError(t, svc.BatchDelete(ctx, nil))
	require.Equal(t, 1, floor91BuildingTotalFloors(t, db, b.ID), "空批量删除不应影响计数")

	// BatchDelete 非空 → 计数同步归零
	require.NoError(t, svc.BatchDelete(ctx, []string{f2.ID}))
	require.Equal(t, 0, floor91BuildingTotalFloors(t, db, b.ID))
}

// TestFloor91_List_Filters 锁定 map 过滤矩阵（name LIKE / buildingId / orgId EXISTS /
// status）命中与不命中分支 + 默认排序 ops_floors.order_num ASC + 分页回填。
func TestFloor91_List_Filters(t *testing.T) {
	db := newFloor91TestDB(t)
	svc := NewFloorService(db)
	ctx := context.Background()

	orgHit := "org-hit-91"
	b := seedFloor91Building(t, db, "B91L", orgHit)
	bOther := seedFloor91Building(t, db, "B91L-other", "org-miss")

	fA := &operationsmodels.OpsFloor{Name: "alpha-91", FloorNo: "1", BuildingID: b.ID, OrderNum: 5}
	fB := &operationsmodels.OpsFloor{Name: "beta-91", FloorNo: "2", BuildingID: b.ID, OrderNum: 1}
	fC := &operationsmodels.OpsFloor{Name: "gamma", FloorNo: "1", BuildingID: bOther.ID, OrderNum: 9, Status: operationsmodels.FloorStatusStopped}
	for _, f := range []*operationsmodels.OpsFloor{fA, fB, fC} {
		require.NoError(t, svc.Create(ctx, f))
	}

	// name LIKE 命中 / 不命中
	page, err := svc.List(ctx, map[string]interface{}{"name": "-91"})
	require.NoError(t, err)
	assert.Equal(t, int64(2), page.Total)
	page, err = svc.List(ctx, map[string]interface{}{"name": "ghost"})
	require.NoError(t, err)
	assert.Zero(t, page.Total)

	// buildingId 等值
	page, err = svc.List(ctx, map[string]interface{}{"buildingId": b.ID})
	require.NoError(t, err)
	assert.Equal(t, int64(2), page.Total)

	// orgId EXISTS 子查询 命中 / 不命中
	page, err = svc.List(ctx, map[string]interface{}{"orgId": orgHit})
	require.NoError(t, err)
	assert.Equal(t, int64(2), page.Total, "orgId 经楼宇 org_id EXISTS 命中子树楼层")
	page, err = svc.List(ctx, map[string]interface{}{"orgId": "no-such-org"})
	require.NoError(t, err)
	assert.Zero(t, page.Total)

	// status 等值 命中 / 不命中
	page, err = svc.List(ctx, map[string]interface{}{"status": 1})
	require.NoError(t, err)
	assert.Equal(t, int64(1), page.Total)
	page, err = svc.List(ctx, map[string]interface{}{"status": 0})
	require.NoError(t, err)
	assert.Equal(t, int64(2), page.Total)

	// 默认排序 ops_floors.order_num ASC（两行不同 order_num 断言顺序）
	page, err = svc.List(ctx, map[string]interface{}{"buildingId": b.ID})
	require.NoError(t, err)
	list := page.List.([]operationsmodels.OpsFloor)
	require.Len(t, list, 2)
	assert.Equal(t, "beta-91", list[0].Name, "默认排序应 ops_floors.order_num ASC（1 在 5 前）")
	assert.Equal(t, "alpha-91", list[1].Name)

	// 分页 total/current/pageSize 回填
	page, err = svc.List(ctx, map[string]interface{}{"buildingId": b.ID, "current": 2, "pageSize": 1})
	require.NoError(t, err)
	assert.Equal(t, int64(2), page.Total)
	assert.Equal(t, 2, page.Current)
	assert.Equal(t, 1, page.PageSize)
	list = page.List.([]operationsmodels.OpsFloor)
	require.Len(t, list, 1)
	assert.Equal(t, "alpha-91", list[0].Name, "第 2 页（按 order_num ASC）应为 alpha-91")
}

// TestFloor91_List_NoClamp 锁定 floor 分页提取的无 clamp 语义（P5 floor 行：
// 内联断言 current/pageSize 任意值直传，不经 clampPageSize）。
func TestFloor91_List_NoClamp(t *testing.T) {
	db := newFloor91TestDB(t)
	svc := NewFloorService(db)
	ctx := context.Background()

	b := seedFloor91Building(t, db, "B91N", "org-91")
	for i := 1; i <= 3; i++ {
		require.NoError(t, svc.Create(ctx, &operationsmodels.OpsFloor{
			Name: "F-N" + strconv.Itoa(i), FloorNo: strconv.Itoa(i), BuildingID: b.ID, OrderNum: i,
		}))
	}

	// pageSize=1 原值生效（无下限 clamp 到 10——typed 服务的 GetPagination 会 clamp 到 ≥10）
	page, err := svc.List(ctx, map[string]interface{}{"buildingId": b.ID, "current": 1, "pageSize": 1})
	require.NoError(t, err)
	assert.Equal(t, 1, page.PageSize, "pageSize=1 应原值生效（floor 无 clamp 契约）")
	assert.Len(t, page.List.([]operationsmodels.OpsFloor), 1)

	// pageSize=5000 原值生效（无上限 clamp——map 服务的 clampPageSize 会截到 10000/100）
	page, err = svc.List(ctx, map[string]interface{}{"buildingId": b.ID, "current": 1, "pageSize": 5000})
	require.NoError(t, err)
	assert.Equal(t, 5000, page.PageSize, "pageSize=5000 应原值生效（floor 无 clamp 契约）")
	assert.Len(t, page.List.([]operationsmodels.OpsFloor), 3)
}
