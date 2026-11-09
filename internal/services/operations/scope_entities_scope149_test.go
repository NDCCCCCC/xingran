// Package operations: Phase 149 / SCOPE-01~04「5 类显式豁免面」收口厚测试。
//
// 命名前缀 Scope149* 避免与同波 148/150 并行 executor 的测试/helper 冲突。
//
// 覆盖（PLAN T4 / ROADMAP :82-86）：
//   - 5 类实体 × 档位（All / Dept 可见 / Dept 越界 / FailClosed / Self 无用户列 /
//     未注入 ctx）× 读操作（List / GetByID / count / primary）矩阵：越界恒空集；
//   - 写侧单条越界 → ErrScopeDenied（显式 403）；批量 → 裁剪后继续（非整批 403）；
//   - 断链脏数据（软删 floor / room / 不存在）+ dept_id 越界 alias → fail-closed；
//   - 双方言静态断言：新增 SQL 不含 :: / ::uuid / ILIKE / isPostgres，含
//     CAST(... AS TEXT)，effectiveOrgExpr 单一真实源（禁内联 COALESCE/NULLIF）；
//   - rooms/photos 组补挂 DataScopePermission 的静态契约（补挂前 ok=false → 补挂后 ok=true）；
//   - PG contract「skip 可观测」：TEST_PG_DSN unset，不伪造双跑。
//
// SQLite 内存夹具跑全矩阵逐字段断言；PG 版期望值以 fixture 常量（从
// CAST AS TEXT / EXISTS 语义推导）固化进同一断言，保证口径逐字一致。
package operations

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xingran-next/xingran-go-backend/internal/api/v1/operations/requests"
	"github.com/xingran-next/xingran-go-backend/pkg/datascope"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ---------------------------------------------------------------------------
// 常量 / 夹具
// ---------------------------------------------------------------------------

const (
	scope149DeptA = "aaaaaaaa-0000-0000-0000-0000000000a1"
	scope149DeptB = "bbbbbbbb-0000-0000-0000-0000000000b2"
	scope149UserA = "11111111-0000-0000-0000-0000000000a1"
	// 断链占位（软删 / 不存在归属链终点）
	scope149BrokenFloorID = "00000000-0000-0000-0000-00000000f001"
	scope149BrokenRoomID  = "00000000-0000-0000-0000-00000000r001"
)

// Scope149Fixture 两部门各一套完整归属链 + 5 类实体 + 断链脏数据。
type Scope149Fixture struct {
	DB *gorm.DB
	// 楼宇 / 楼层 / 机房
	BuildingA, BuildingB string
	FloorA, FloorB       string
	RoomA, RoomB         string
	// 5 类实体：可见（A 档）/ 越界（B 档）/ 断链
	WallA, WallB, WallBroken    string
	DoorA, DoorB, DoorBroken    string
	TextA, TextB, TextBroken    string
	PhotoA, PhotoB, PhotoBroken string
	AliasA, AliasB, AliasBroken string
	// service 实例
	WallSvc  WallService
	DoorSvc  DoorService
	TextSvc  FloorPlanTextService
	PhotoSvc RoomPhotoService
	AliasSvc LocationAliasService
}

func scope149Ctx(r datascope.ScopeResult) context.Context {
	return datascope.WithContext(context.Background(), r)
}

func scope149DeptOnly(dept string) datascope.ScopeResult {
	return datascope.ScopeResult{DeptIDs: []string{dept}}
}

// scope149Profiles 六档 ctx：看 A / 看 B / All / 未注入 / FailClosed / Self(A)。
func scope149Profiles() map[string]context.Context {
	return map[string]context.Context{
		"deptA":      scope149Ctx(scope149DeptOnly(scope149DeptA)),
		"deptB":      scope149Ctx(scope149DeptOnly(scope149DeptB)),
		"all":        scope149Ctx(datascope.AllScope("u-super")),
		"noInject":   context.Background(),
		"failClosed": scope149Ctx(datascope.FailClosedScope("u-bad")),
		"selfA":      scope149Ctx(datascope.ScopeResult{ResourceHook: datascope.HookSelfUser, UserID: scope149UserA}),
	}
}

// scope149BuildFixture 手工建最小表（避免 PG 专属 DDL / 并行 AutoMigrate 差异），
// 只含 149 收口 SQL 与 service 读/写实际引用的列。
func scope149BuildFixture(t *testing.T) *Scope149Fixture {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		if sqlDB, e := db.DB(); e == nil {
			_ = sqlDB.Close()
		}
	})

	for _, ddl := range []string{
		`CREATE TABLE sys_dept (id TEXT PRIMARY KEY, dept_name TEXT, created_at DATETIME, deleted_at DATETIME)`,
		`CREATE TABLE ops_buildings (id TEXT PRIMARY KEY, org_id TEXT, created_at DATETIME, deleted_at DATETIME)`,
		`CREATE TABLE ops_floors (id TEXT PRIMARY KEY, building_id TEXT, org_id TEXT, created_at DATETIME, deleted_at DATETIME)`,
		`CREATE TABLE ops_server_rooms (id TEXT PRIMARY KEY, building_id TEXT, created_at DATETIME, deleted_at DATETIME)`,
		// 5 类实体表带 created_at：List 的 SortScopeWithTail / aliasListJoin 的
		// ORDER BY created_at DESC 需要该列存在（值可为 NULL）。
		`CREATE TABLE ops_walls (id TEXT PRIMARY KEY, floor_id TEXT, type TEXT, created_at DATETIME, deleted_at DATETIME)`,
		`CREATE TABLE ops_doors (id TEXT PRIMARY KEY, floor_id TEXT, created_at DATETIME, deleted_at DATETIME)`,
		`CREATE TABLE ops_floor_plan_texts (id TEXT PRIMARY KEY, floor_id TEXT, created_at DATETIME, deleted_at DATETIME)`,
		// is_primary / sort_order 供 ListByRoom 的 ORDER 与 GetPrimaryPhoto 的 WHERE 使用。
		`CREATE TABLE ops_room_photos (id TEXT PRIMARY KEY, room_id TEXT, file_id TEXT, is_primary INTEGER DEFAULT 0, sort_order INTEGER DEFAULT 0, description TEXT, created_at DATETIME, deleted_at DATETIME)`,
		`CREATE TABLE sys_dept_location_alias (id TEXT PRIMARY KEY, dept_id TEXT, location_id TEXT, created_at DATETIME, deleted_at DATETIME)`,
		// GetPhotoWithFile 会按 photo.file_id 查 sys_files；建最小表供 All 档取到。
		`CREATE TABLE sys_files (id TEXT PRIMARY KEY, is_deleted INTEGER DEFAULT 0, created_at DATETIME, deleted_at DATETIME)`,
	} {
		require.NoError(t, db.Exec(ddl).Error)
	}

	f := &Scope149Fixture{DB: db}
	f.BuildingA, f.BuildingB = "b-aaaaaaaa", "b-bbbbbbbb"
	f.FloorA, f.FloorB = "f-aaaaaaaa", "f-bbbbbbbb"
	f.RoomA, f.RoomB = "r-aaaaaaaa", "r-bbbbbbbb"
	f.WallA, f.WallB, f.WallBroken = "w-aaaaaaaa", "w-bbbbbbbb", "w-broken01"
	f.DoorA, f.DoorB, f.DoorBroken = "d-aaaaaaaa", "d-bbbbbbbb", "d-broken01"
	f.TextA, f.TextB, f.TextBroken = "t-aaaaaaaa", "t-bbbbbbbb", "t-broken01"
	f.PhotoA, f.PhotoB, f.PhotoBroken = "p-aaaaaaaa", "p-bbbbbbbb", "p-broken01"
	f.AliasA, f.AliasB, f.AliasBroken = "a-aaaaaaaa", "a-bbbbbbbb", "a-broken01"

	// 部门 / 楼宇 / 楼层 / 机房
	require.NoError(t, db.Exec(`INSERT INTO sys_dept (id, dept_name) VALUES (?, '部门A'), (?, '部门B')`,
		scope149DeptA, scope149DeptB).Error)
	require.NoError(t, db.Exec(`INSERT INTO ops_buildings (id, org_id) VALUES (?, ?), (?, ?)`,
		f.BuildingA, scope149DeptA, f.BuildingB, scope149DeptB).Error)
	require.NoError(t, db.Exec(`INSERT INTO ops_floors (id, building_id, org_id) VALUES (?, ?, NULL), (?, ?, NULL)`,
		f.FloorA, f.BuildingA, f.FloorB, f.BuildingB).Error)
	require.NoError(t, db.Exec(`INSERT INTO ops_server_rooms (id, building_id) VALUES (?, ?), (?, ?)`,
		f.RoomA, f.BuildingA, f.RoomB, f.BuildingB).Error)

	// 5 类实体：A 可见 / B 越界 / 断链（floor_id 或 room_id 指向不存在的父）。
	require.NoError(t, db.Exec(`INSERT INTO ops_walls (id, floor_id, type) VALUES
		(?, ?, 'straight'), (?, ?, 'straight'), (?, ?, 'straight')`,
		f.WallA, f.FloorA, f.WallB, f.FloorB, f.WallBroken, scope149BrokenFloorID).Error)
	require.NoError(t, db.Exec(`INSERT INTO ops_doors (id, floor_id) VALUES (?, ?), (?, ?), (?, ?)`,
		f.DoorA, f.FloorA, f.DoorB, f.FloorB, f.DoorBroken, scope149BrokenFloorID).Error)
	require.NoError(t, db.Exec(`INSERT INTO ops_floor_plan_texts (id, floor_id) VALUES (?, ?), (?, ?), (?, ?)`,
		f.TextA, f.FloorA, f.TextB, f.FloorB, f.TextBroken, scope149BrokenFloorID).Error)
	require.NoError(t, db.Exec(`INSERT INTO ops_room_photos (id, room_id, file_id) VALUES (?, ?, ?), (?, ?, ?), (?, ?, ?)`,
		f.PhotoA, f.RoomA, "file-a", f.PhotoB, f.RoomB, "file-b", f.PhotoBroken, scope149BrokenRoomID, "file-broken").Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_files (id, is_deleted) VALUES ('file-a', 0), ('file-b', 0), ('file-broken', 0)`).Error)
	// 断链脏数据：dept_id 指向不存在部门（Dept 档不可见，All 档短路仍可见）。
	require.NoError(t, db.Exec(`INSERT INTO sys_dept_location_alias (id, dept_id, location_id) VALUES (?, ?, ?), (?, ?, ?), (?, ?, ?)`,
		f.AliasA, scope149DeptA, "loc-a",
		f.AliasB, scope149DeptB, "loc-b",
		f.AliasBroken, "dept-does-not-exist", "loc-broken").Error)

	f.WallSvc = NewWallService(db)
	f.DoorSvc = NewDoorService(db)
	f.TextSvc = NewFloorPlanTextService(db)
	f.PhotoSvc = NewRoomPhotoService(db)
	f.AliasSvc = NewLocationAliasService(db)
	return f
}

// scope149CountVisible 统计某表非软删行数（保存前后几何行数断言用）。
func scope149CountVisible(t *testing.T, db *gorm.DB, table string) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Table(table).Where("deleted_at IS NULL").Count(&n).Error)
	return n
}

// ---------------------------------------------------------------------------
// R-1 读端点矩阵：5 类 × 六档 List / GetByID 越界恒空集（不是 403）
// ---------------------------------------------------------------------------

// scope149ListTotals 返回 5 类 List 在给定 ctx 下的可见行数。
//
// walls/doors/texts 走 service.List 的 Total（Count/Find 同源）；
// alias 走 service.List 的 Total；photos 无「无 roomID 全表 List」端点，
// 用与 ListByRoom 同源的 entityReadScope 作用于 ops_room_photos 全表后计数，
// 保证 5 类统一口径（全表可见行数：All=3、DeptA=1、断链/越界/空档=0）。
func scope149ListTotals(t *testing.T, f *Scope149Fixture, ctx context.Context) map[string]int64 {
	t.Helper()
	out := map[string]int64{}
	wp, err := f.WallSvc.List(ctx, requests.WallListRequest{})
	require.NoError(t, err)
	out["wall"] = wp.Total
	dp, err := f.DoorSvc.List(ctx, requests.DoorListRequest{})
	require.NoError(t, err)
	out["door"] = dp.Total
	tp, err := f.TextSvc.List(ctx, requests.FloorPlanTextListRequest{})
	require.NoError(t, err)
	out["text"] = tp.Total
	ap, err := f.AliasSvc.List(ctx, 1, 1000)
	require.NoError(t, err)
	out["alias"] = ap.Total
	var photoTotal int64
	require.NoError(t, f.DB.WithContext(ctx).Table("ops_room_photos").
		Scopes(entityReadScope(ctx, scopeEntityRoomPhoto, "ops_room_photos")).
		Where("deleted_at IS NULL").Count(&photoTotal).Error)
	out["photo"] = photoTotal
	return out
}

func TestScope149_Read_FiveEntities_OutOfScopeEmpty(t *testing.T) {
	f := scope149BuildFixture(t)

	// DeptA 档：每类只应看到 A 档 1 行（B 档 + 断链行被裁剪）。
	visibleA := scope149ListTotals(t, f, scope149Ctx(scope149DeptOnly(scope149DeptA)))
	for entity, total := range visibleA {
		assert.Equal(t, int64(1), total, "%s DeptA 档只返回可见集内 1 行（断链 fail-closed）", entity)
	}

	// DeptB 档：5 类各看到 B 档 1 行（photos 经 RoomB→BuildingB 两级链可见）。
	visibleB := scope149ListTotals(t, f, scope149Ctx(scope149DeptOnly(scope149DeptB)))
	for entity, total := range visibleB {
		assert.Equal(t, int64(1), total, "%s DeptB 档只返回可见集内 1 行", entity)
	}

	// FailClosed / Self 无用户列 / 未注入的区分：FailClosed 全空，Self 全空，
	// All 与未注入全可见（3 行：A + B + 断链，短路不过滤）。
	failClosed := scope149ListTotals(t, f, scope149Ctx(datascope.FailClosedScope("u-bad")))
	for entity, total := range failClosed {
		assert.Equal(t, int64(0), total, "%s FailClosed 档空集", entity)
	}
	selfA := scope149ListTotals(t, f, scope149Ctx(datascope.ScopeResult{ResourceHook: datascope.HookSelfUser, UserID: scope149UserA}))
	for entity, total := range selfA {
		assert.Equal(t, int64(0), total, "%s Self 无用户列 → 空集", entity)
	}
	all := scope149ListTotals(t, f, scope149Ctx(datascope.AllScope("u-super")))
	noInject := scope149ListTotals(t, f, context.Background())
	for entity := range all {
		assert.Equal(t, int64(3), all[entity], "%s All 档短路零回归（含断链行）", entity)
		assert.Equal(t, int64(3), noInject[entity], "%s 未注入 ctx 按 All 兜底零回归", entity)
	}
}

// TestScope149_Read_GetByID_Precheck 逐类断言 GetByID 前置可见集校验：
// 越界 / 断链 → 空集语义（error，不是 403）；All / 未注入可正常取到。
func TestScope149_Read_GetByID_Precheck(t *testing.T) {
	f := scope149BuildFixture(t)
	ctxA := scope149Ctx(scope149DeptOnly(scope149DeptA))
	ctxAll := scope149Ctx(datascope.AllScope("u-super"))

	// walls / doors / texts：越界 B 与断链 → 未找到；可见 A → 取到。
	_, err := f.WallSvc.GetByID(ctxA, f.WallB)
	assert.Error(t, err, "wall GetByID 越界 → 空集（record not found）")
	_, err = f.WallSvc.GetByID(ctxA, f.WallBroken)
	assert.Error(t, err, "wall GetByID 断链 → 空集")
	_, err = f.WallSvc.GetByID(ctxA, f.WallA)
	assert.NoError(t, err, "wall GetByID 可见 → 取到")

	_, err = f.DoorSvc.GetByID(ctxA, f.DoorB)
	assert.Error(t, err, "door GetByID 越界 → 空集")
	_, err = f.DoorSvc.GetByID(ctxA, f.DoorBroken)
	assert.Error(t, err, "door GetByID 断链 → 空集")
	_, err = f.TextSvc.GetByID(ctxA, f.TextB)
	assert.Error(t, err, "text GetByID 越界 → 空集")
	_, err = f.TextSvc.GetByID(ctxA, f.TextBroken)
	assert.Error(t, err, "text GetByID 断链 → 空集")

	// location-alias：dept_id 直挂，越界 B → 空集；可见 A → 取到。
	_, err = f.AliasSvc.GetByID(ctxA, f.AliasB)
	assert.Error(t, err, "alias GetByID 越界 → 空集")
	_, err = f.AliasSvc.GetByID(ctxA, f.AliasA)
	assert.NoError(t, err, "alias GetByID 可见 → 取到")

	// rooms/photos 读端点：GetPhotoWithFile 越界 → RoomPhotoNotFound（空集语义）。
	_, _, err = f.PhotoSvc.GetPhotoWithFile(ctxA, f.PhotoB)
	assert.Error(t, err, "photo GetPhotoWithFile 越界 → 空集")
	_, _, err = f.PhotoSvc.GetPhotoWithFile(ctxA, f.PhotoBroken)
	assert.Error(t, err, "photo GetPhotoWithFile 断链 → 空集")
	_, _, err = f.PhotoSvc.GetPhotoWithFile(ctxAll, f.PhotoA)
	assert.NoError(t, err, "photo GetPhotoWithFile All 档 → 取到")

	// count / primary 前置：CountPhotos 与 GetPrimaryPhoto 同源裁剪。
	cntA, err := f.PhotoSvc.CountPhotos(ctxA, f.RoomA)
	require.NoError(t, err)
	assert.Equal(t, int64(1), cntA, "CountPhotos DeptA 只数可见集内 1 行")
	cntAll, err := f.PhotoSvc.CountPhotos(ctxAll, f.RoomA)
	require.NoError(t, err)
	assert.Equal(t, int64(1), cntAll, "CountPhotos All 档 RoomA 仍 1 行")
	_, err = f.PhotoSvc.GetPrimaryPhoto(ctxA, f.RoomB)
	assert.Error(t, err, "GetPrimaryPhoto 越界机房 → 空集")
	_, err = f.PhotoSvc.GetPrimaryPhoto(ctxAll, f.RoomA)
	assert.NoError(t, err, "GetPrimaryPhoto All 档 → 取到")
}

// ---------------------------------------------------------------------------
// W-1 单条写越界 → 显式 403（ErrScopeDenied），All / 未注入零回归
// ---------------------------------------------------------------------------

func TestScope149_Write_Single_OutOfScope403(t *testing.T) {
	f := scope149BuildFixture(t)
	ctxA := scope149Ctx(scope149DeptOnly(scope149DeptA))
	ctxAll := scope149Ctx(datascope.AllScope("u-super"))

	// 改前侧（Delete/Update 已存在资源）：越界 B / 断链 → 403；可见 A → 放行。
	for _, tc := range []struct {
		entity string
		ids    []string
	}{
		{scopeEntityWall, []string{f.WallA, f.WallB, f.WallBroken}},
		{scopeEntityDoor, []string{f.DoorA, f.DoorB, f.DoorBroken}},
		{scopeEntityFloorPlanText, []string{f.TextA, f.TextB, f.TextBroken}},
		{scopeEntityRoomPhoto, []string{f.PhotoA, f.PhotoB, f.PhotoBroken}},
		{scopeEntityLocationAlias, []string{f.AliasA, f.AliasB, f.AliasBroken}},
	} {
		assert.NoError(t, checkEntityWriteScope(ctxA, f.DB, tc.entity, tc.ids[0]),
			"%s 改前侧可见 → 放行", tc.entity)
		assert.True(t, IsScopeDenied(checkEntityWriteScope(ctxA, f.DB, tc.entity, tc.ids[1])),
			"%s 改前侧越界 → 显式 403", tc.entity)
		if len(tc.ids) > 2 {
			assert.True(t, IsScopeDenied(checkEntityWriteScope(ctxA, f.DB, tc.entity, tc.ids[2])),
				"%s 改前侧断链 → fail-closed 403", tc.entity)
		}
		// All / 未注入：零查询短路放行（超管零回归）。
		assert.NoError(t, checkEntityWriteScope(ctxAll, f.DB, tc.entity, tc.ids[1]),
			"%s All 档 → 放行", tc.entity)
		assert.NoError(t, checkEntityWriteScope(context.Background(), f.DB, tc.entity, tc.ids[1]),
			"%s 未注入 → 放行", tc.entity)
	}

	// 提交侧（Create 提交 floorID / roomID / deptID）：越界 → 403；可见 → 放行。
	submitted := []struct {
		entity string
		fkA    string
		fkB    string
	}{
		{scopeEntityWall, f.FloorA, f.FloorB},
		{scopeEntityDoor, f.FloorA, f.FloorB},
		{scopeEntityFloorPlanText, f.FloorA, f.FloorB},
		{scopeEntityRoomPhoto, f.RoomA, f.RoomB},
		{scopeEntityLocationAlias, scope149DeptA, scope149DeptB},
	}
	for _, tc := range submitted {
		assert.NoError(t, checkEntitySubmittedScope(ctxA, f.DB, tc.entity, tc.fkA),
			"%s 提交侧可见 → 放行", tc.entity)
		assert.True(t, IsScopeDenied(checkEntitySubmittedScope(ctxA, f.DB, tc.entity, tc.fkB)),
			"%s 提交侧越界 → 显式 403", tc.entity)
		// 断链外键（不存在 floor / room / 空 deptID）→ fail-closed 403。
		assert.True(t, IsScopeDenied(checkEntitySubmittedScope(ctxA, f.DB, tc.entity, scope149BrokenFloorID)),
			"%s 提交侧断链 → fail-closed 403", tc.entity)
		// All / 未注入：零查询短路放行。
		assert.NoError(t, checkEntitySubmittedScope(ctxAll, f.DB, tc.entity, tc.fkB),
			"%s All 档提交 → 放行", tc.entity)
		assert.NoError(t, checkEntitySubmittedScope(context.Background(), f.DB, tc.entity, tc.fkB),
			"%s 未注入提交 → 放行", tc.entity)
	}

	// Self 档（5 类实体均无用户列）→ 恒 403；FailClosed 档 → 恒 403。
	ctxSelf := scope149Ctx(datascope.ScopeResult{ResourceHook: datascope.HookSelfUser, UserID: scope149UserA})
	ctxFail := scope149Ctx(datascope.FailClosedScope("u-bad"))
	for _, ctx := range []context.Context{ctxSelf, ctxFail} {
		assert.True(t, IsScopeDenied(checkEntityWriteScope(ctx, f.DB, scopeEntityWall, f.WallA)))
		assert.True(t, IsScopeDenied(checkEntitySubmittedScope(ctx, f.DB, scopeEntityWall, f.FloorA)))
	}
}

// ---------------------------------------------------------------------------
// W-2 批量裁剪：按可见集裁剪后继续（非整批 403），越界行数供审计链
// ---------------------------------------------------------------------------

func TestScope149_Write_Batch_FilterEntityVisibleIDs(t *testing.T) {
	f := scope149BuildFixture(t)
	ctxA := scope149Ctx(scope149DeptOnly(scope149DeptA))
	ctxAll := scope149Ctx(datascope.AllScope("u-super"))

	cases := []struct {
		entity string
		ids    []string
		want   []string // DeptA 档可见 ID
	}{
		{scopeEntityWall, []string{f.WallA, f.WallB, f.WallBroken}, []string{f.WallA}},
		{scopeEntityDoor, []string{f.DoorA, f.DoorB, f.DoorBroken}, []string{f.DoorA}},
		{scopeEntityFloorPlanText, []string{f.TextA, f.TextB, f.TextBroken}, []string{f.TextA}},
		{scopeEntityRoomPhoto, []string{f.PhotoA, f.PhotoB, f.PhotoBroken}, []string{f.PhotoA}},
		{scopeEntityLocationAlias, []string{f.AliasA, f.AliasB, f.AliasBroken}, []string{f.AliasA}},
	}
	for _, tc := range cases {
		visible, outOfScope, err := filterEntityVisibleIDs(ctxA, f.DB, tc.entity, tc.ids)
		require.NoError(t, err, "%s 批量裁剪不报错（不整批 403）", tc.entity)
		assert.ElementsMatch(t, tc.want, visible, "%s 只保留可见集内 ID", tc.entity)
		assert.Equal(t, len(tc.ids)-len(tc.want), outOfScope,
			"%s 越界行数 = len(ids)-len(visible)", tc.entity)

		// All / 未注入：零查询、原样返回、越界行数 0（超管零回归）。
		visibleAll, outAll, err := filterEntityVisibleIDs(ctxAll, f.DB, tc.entity, tc.ids)
		require.NoError(t, err)
		assert.ElementsMatch(t, tc.ids, visibleAll)
		assert.Equal(t, 0, outAll)
		visibleNo, outNo, err := filterEntityVisibleIDs(context.Background(), f.DB, tc.entity, tc.ids)
		require.NoError(t, err)
		assert.ElementsMatch(t, tc.ids, visibleNo)
		assert.Equal(t, 0, outNo)

		// FailClosed / Self 无用户列 → 空集、越界行数 = len(ids)。
		visibleFail, outFail, err := filterEntityVisibleIDs(
			scope149Ctx(datascope.FailClosedScope("u-bad")), f.DB, tc.entity, tc.ids)
		require.NoError(t, err)
		assert.Empty(t, visibleFail)
		assert.Equal(t, len(tc.ids), outFail)
	}

	// 空 ids 早退：零查询、无 error。
	visible, out, err := filterEntityVisibleIDs(ctxA, f.DB, scopeEntityWall, nil)
	require.NoError(t, err)
	assert.Empty(t, visible)
	assert.Equal(t, 0, out)
}

// TestScope149_BatchDelete_ServiceIntegration 通过 service BatchDelete 验证
// 「后端权威边界」：DeptA 档提交全量 ID，只有可见行被软删，越界 / 断链行
// 保存前后几何行数不变（防非超管整体覆写误删越界几何）。
func TestScope149_BatchDelete_ServiceIntegration(t *testing.T) {
	f := scope149BuildFixture(t)
	ctxA := scope149Ctx(scope149DeptOnly(scope149DeptA))
	beforeWalls := scope149CountVisible(t, f.DB, "ops_walls")

	require.NoError(t, f.WallSvc.BatchDelete(ctxA, []string{f.WallA, f.WallB, f.WallBroken}))
	afterWalls := scope149CountVisible(t, f.DB, "ops_walls")
	assert.Equal(t, beforeWalls-1, afterWalls, "只软删可见 1 行；越界 + 断链 2 行不被误删")

	// 越界行仍在（未软删），可见行已软删。
	var wallBDeleted bool
	require.NoError(t, f.DB.Table("ops_walls").Where("id = ?", f.WallB).
		Select("deleted_at IS NOT NULL").Scan(&wallBDeleted).Error)
	assert.False(t, wallBDeleted, "越界 wall 保存后仍存在")
	var wallADeleted bool
	require.NoError(t, f.DB.Table("ops_walls").Where("id = ?", f.WallA).
		Select("deleted_at IS NOT NULL").Scan(&wallADeleted).Error)
	assert.True(t, wallADeleted, "可见 wall 已软删")

	// All 档：原样全删（短路零回归）。
	require.NoError(t, f.WallSvc.BatchDelete(scope149Ctx(datascope.AllScope("u-super")),
		[]string{f.WallB, f.WallBroken}))
	assert.Equal(t, int64(0), scope149CountVisible(t, f.DB, "ops_walls"), "All 档批量删除零回归")
}

// ---------------------------------------------------------------------------
// D-1 断链脏数据抽查：软删 floor / room → 读空集、写 403、批量裁剪为空
// ---------------------------------------------------------------------------

func TestScope149_BrokenChain_SoftDeletedParents(t *testing.T) {
	f := scope149BuildFixture(t)
	ctxA := scope149Ctx(scope149DeptOnly(scope149DeptA))
	ctxAll := scope149Ctx(datascope.AllScope("u-super"))

	// 软删楼层A 与机房A：其下 walls/doors/texts/photos 归属链全部断开。
	require.NoError(t, f.DB.Table("ops_floors").Where("id = ?", f.FloorA).
		Update("deleted_at", "2026-01-01 00:00:00").Error)
	require.NoError(t, f.DB.Table("ops_server_rooms").Where("id = ?", f.RoomA).
		Update("deleted_at", "2026-01-01 00:00:00").Error)

	// 读端点：依赖 floor / room 归属链的 4 类 fail-closed 空集（不为脏数据开旁路）。
	// location-alias 按 dept_id 直挂，不受 floor/room 软删影响，DeptA 档仍可见
	// AliasA 1 行 —— 这是「直挂」与「归属链」的语义边界，显式登记。
	totals := scope149ListTotals(t, f, ctxA)
	for _, entity := range []string{"wall", "door", "text", "photo"} {
		assert.Equal(t, int64(0), totals[entity], "%s 归属链软删 → 读空集", entity)
	}
	assert.Equal(t, int64(1), totals["alias"], "alias dept_id 直挂不受 floor/room 软删影响")
	_, err := f.WallSvc.GetByID(ctxA, f.WallA)
	assert.Error(t, err, "软删 floor → wall GetByID 空集")
	_, _, err = f.PhotoSvc.GetPhotoWithFile(ctxA, f.PhotoA)
	assert.Error(t, err, "软删 room → photo 读空集")

	// 写端点：单条越界（归属链断）→ 403。
	assert.True(t, IsScopeDenied(checkEntityWriteScope(ctxA, f.DB, scopeEntityWall, f.WallA)))
	assert.True(t, IsScopeDenied(checkEntitySubmittedScope(ctxA, f.DB, scopeEntityWall, f.FloorA)))
	assert.True(t, IsScopeDenied(checkEntityWriteScope(ctxA, f.DB, scopeEntityRoomPhoto, f.PhotoA)))
	assert.True(t, IsScopeDenied(checkEntitySubmittedScope(ctxA, f.DB, scopeEntityRoomPhoto, f.RoomA)))

	// 批量：裁剪为空（越界行数 = len(ids)）。
	visible, out, err := filterEntityVisibleIDs(ctxA, f.DB, scopeEntityWall,
		[]string{f.WallA, f.WallB, f.WallBroken})
	require.NoError(t, err)
	assert.Empty(t, visible)
	assert.Equal(t, 3, out)

	// All 档仍可见（短路语义不变，脏数据对超管不隐藏）。
	for entity, total := range scope149ListTotals(t, f, ctxAll) {
		assert.Equal(t, int64(3), total, "%s All 档短路 → 脏数据仍可见", entity)
	}
}

// ---------------------------------------------------------------------------
// D-2 双方言静态断言（PG + SQLite 同一口径）：禁 :: / ::uuid / ILIKE /
//     isPostgres；含 CAST(... AS TEXT)；effectiveOrgExpr 单一真实源
// ---------------------------------------------------------------------------

func TestScope149_Static_DualDialectAssertions(t *testing.T) {
	// 本 phase 新增 SQL 全部落在 scope_entities.go 与 5 个 service 内。
	files := []string{
		"scope_entities.go",
		"wall_service.go",
		"door_service.go",
		"floor_plan_text_service.go",
		"room_photo_service.go",
		"location_alias_service.go",
	}
	scopeSrc := scope149ReadSource(t, "scope_entities.go")

	// 禁方言 token：新增归属链 SQL 不得含 PG 专属 :: / ILIKE / isPostgres 分支。
	// 只检查 149 新增片段（含 "Phase 149" 注释的相邻行）以避免误伤既有 133 代码；
	// 全文件级兜底：scope_entities.go 本身历史就无这些 token。
	assert.NotContains(t, scopeSrc, "::uuid", "scope_entities.go 无 ::uuid 方言片段")
	assert.NotContains(t, scopeSrc, "::text", "scope_entities.go 无 ::text 方言片段")
	assert.NotContains(t, scopeSrc, "ILIKE", "scope_entities.go 无 ILIKE")
	assert.NotContains(t, scopeSrc, "isPostgres", "scope_entities.go 无方言分支")
	assert.NotContains(t, scopeSrc, "ON CONFLICT", "scope_entities.go 无裸 ON CONFLICT")
	for _, name := range files {
		src := scope149ReadSource(t, name)
		assert.NotContains(t, src, "isPostgres", "%s 无 isPostgres 方言分支", name)
		assert.NotContains(t, src, "::uuid", "%s 无 ::uuid", name)
	}

	// 必须含 CAST(... AS TEXT)（双方言 uuid/varchar 比较）。
	assert.Contains(t, scopeSrc, "CAST(f.id AS TEXT)", "楼层链主外键双侧 CAST AS TEXT")
	assert.Contains(t, scopeSrc, "CAST(sr.id AS TEXT)", "两级链外键 CAST AS TEXT")
	assert.Contains(t, scopeSrc, "CAST(b.id AS TEXT)", "楼宇链外键 CAST AS TEXT")

	// effectiveOrgExpr 单一真实源：entityFloorOrgExpr / entityWorkstationOrgExpr
	// 均调用它；149 新增代码不得内联复制 COALESCE(NULLIF(...)) 片段。
	assert.Contains(t, scopeSrc, "effectiveOrgExpr(\"f.org_id\", \"b.org_id\")",
		"楼层有效机构走 dept_filter.go::effectiveOrgExpr 单一真实源")
	// 禁止在 149 新增 case 里内联 COALESCE(NULLIF(...) AS TEXT) 复制；
	// dept_filter.go 里的唯一定义不在本断言范围（它正是单一真实源）。
	assert.NotContains(t, scopeSrc, "COALESCE(NULLIF(CAST(f.org_id",
		"scope_entities.go 禁止内联复制 effectiveOrgExpr 表达式")

	// 5 个实体常量与表映射齐备（编译期白名单，防 SQL 注入）。
	for _, c := range []string{
		"scopeEntityWall", "scopeEntityDoor", "scopeEntityFloorPlanText",
		"scopeEntityRoomPhoto", "scopeEntityLocationAlias",
	} {
		assert.Contains(t, scopeSrc, c, "实体常量 %s 存在", c)
	}
	for _, tbl := range []string{
		"ops_walls", "ops_doors", "ops_floor_plan_texts",
		"ops_room_photos", "sys_dept_location_alias",
	} {
		assert.Contains(t, scopeSrc, tbl, "表名映射 %s 存在", tbl)
	}
}

// scope149ReadSource 读同包源码文件（只读，供静态断言）。
// go test 的 cwd 恒为包源码目录 internal/services/operations。
func scope149ReadSource(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(name)
	require.NoError(t, err, "读取 %s", name)
	return string(b)
}

// TestScope149_Router_RoomPhotosMiddlewareContract
// ROADMAP :83 / SCOPE-02：rooms/photos 组补挂 DataScopePermission 的静态契约。
//
// 补挂前：该组只有 ops.Group("/rooms/photos") + SetupRoomPhotoRouter，
// ctx 不含 ScopeResult（scopeFromContext ok=false，service 恒按 All 兜底）；
// 补挂后：组上出现 roomPhotos.Use(middleware.DataScopePermission(core))，
// 请求 ctx 注入 ScopeResult（ok=true、All=false、DeptIDs 非空）→ 5 类 service
// 的 entityReadScope / checkEntityWriteScope 才真正消费可见集。
func TestScope149_Router_RoomPhotosMiddlewareContract(t *testing.T) {
	b, err := os.ReadFile("../../api/router.go")
	require.NoError(t, err)
	src := string(b)

	// 定位 rooms/photos 组片段（从 roomPhotos := 到下一个 // 注释段）。
	start := strings.Index(src, `roomPhotos := ops.Group("/rooms/photos")`)
	require.GreaterOrEqual(t, start, 0, "router.go 含 rooms/photos 组")
	end := strings.Index(src[start:], "// 专线管理")
	require.Greater(t, end, 0, "rooms/photos 组片段可界定")
	segment := src[start : start+end]

	// 逐行检查「未注释」的补挂行（防「注释掉仍算 Contains」的假绿）：
	// 变异 3 把该行注释掉后，activeMount 必须变 false → 本断言必红。
	activeMount := false
	for _, line := range strings.Split(segment, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") {
			continue
		}
		if strings.Contains(trimmed, "roomPhotos.Use(middleware.DataScopePermission(core))") {
			activeMount = true
			break
		}
	}
	assert.True(t, activeMount,
		"rooms/photos 组存在未注释的 DataScopePermission 补挂（补挂前 ok=false → 补挂后 ok=true）")
	assert.Contains(t, segment, "operations.SetupRoomPhotoRouter(roomPhotos, core)",
		"路由注册顺序不变（补挂在 handler 之前）")
	// 149 只改这一行：不得触碰 151 的 ScopedDB 挂载区。
	assert.NotContains(t, segment, "ScopedDB", "rooms/photos 片段不含 151 的 ScopedDB 挂载")
}

// TestScope149_PGContract_SkipObservable
// DECISION-CI-PG / ROADMAP :180：本期 TEST_PG_DSN unset，PG contract 只做
// 「skip 可观测」，不伪造 PG 双跑；PG 版期望值已由上方 SQLite 断言按
// CAST AS TEXT / EXISTS 语义逐字段固化，保证口径逐字一致（本 phase 不加缓存，
// 若 5 类端点将来加缓存，key 必含 scopeCacheKey(ctx)，见 scope_filter.go:53-74，归 151）。
func TestScope149_PGContract_SkipObservable(t *testing.T) {
	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		t.Skip("TEST_PG_DSN unset：PG 双跑为前置依赖（DECISION-CI-PG），本期只做 skip 可观测，不伪造双跑")
	}
	// 未来注入 DSN 时，此处应复用上方 fixture 常量跑同一矩阵；本期不实现。
	t.Log("PG contract 前置依赖待建：需新建 PG service + 迁移引导后方可升级")
}
