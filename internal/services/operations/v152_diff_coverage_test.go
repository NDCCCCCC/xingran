package operations

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/xingran-next/xingran-go-backend/internal/api/v1/operations/requests"
	"github.com/xingran-next/xingran-go-backend/internal/models/operations"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
)

// =====================================================================
// PR#75 diff-coverage 补厚测试（v152）。
//
// 目标：把 check-diff-coverage.sh 报出的 PR#75 未覆盖分支补齐，每个用例都带
// 明确语义断言。全部符号以 DiffCov 前缀命名，避免与同包其它 agent 的
// scope_* 测试撞名。夹具优先复用既有 helper（setupBackfillDeptDB /
// setupFloorOrgOverrideDB / setupResolver77DB）；签名不合适时才用自建
// newDiffCovDB(t)（glebarez/sqlite + t.TempDir()）。
// =====================================================================

const (
	diffCovDeptHQ    = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" // 已存在的部门（UUID）
	diffCovDeptGhost = "dddddddd-dddd-dddd-dddd-dddddddddddd" // 合法 UUID 但不存在的部门
	diffCovUserA     = "11111111-1111-1111-1111-111111111111"
	diffCovOrgWH     = "22222222-2222-2222-2222-222222222222" // 武汉中支（楼层覆盖用）
)

// newDiffCovDB 自建最小夹具：sys_dept + sys_user + sys_workstation，文件库
// （t.TempDir）避免 :memory: 连接池多连接各自空库的坑。
func newDiffCovDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "diffcov.db")), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_dept (id TEXT PRIMARY KEY, dept_name TEXT, dept_code TEXT,
			parent_id TEXT, ancestors TEXT, deleted_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_user (id TEXT PRIMARY KEY, username TEXT, nickname TEXT,
			dept_id TEXT, deleted_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_workstation (
			id TEXT PRIMARY KEY, workstation_name TEXT, dept_id TEXT, user_id TEXT,
			floor_id TEXT, status INTEGER DEFAULT 0,
			deleted_at DATETIME, created_at DATETIME, updated_at DATETIME)`).Error)
	return db
}

// diffCovSeedDept 插入一个部门行（可被 drop/rename 制造 DB 错误）。
func diffCovSeedDept(t *testing.T, db *gorm.DB, id string) {
	t.Helper()
	require.NoError(t, db.Exec(
		`INSERT INTO sys_dept (id, dept_name, dept_code, ancestors) VALUES (?, '总部', 'HQ', '0')`, id).Error)
}

// diffCovFloorSvc 构造仅用于 validateOrgOptional 白盒调用的 floorService。
func diffCovFloorSvc(db *gorm.DB) *floorService {
	return &floorService{db: db}
}

// ---------------------------------------------------------------------
// 1) floor_service.validateOrgOptional（门禁行 571-583）
//    分支矩阵：nil / 空串 / 全空格(trim) / 非法 UUID / 合法但不存在的部门 /
//    合法且存在 / Count 查询报错。
// ---------------------------------------------------------------------

// TestDiffCov_ValidateOrgOptional_NilAndEmpty nil 与空串直接放行，不查库。
func TestDiffCov_ValidateOrgOptional_NilAndEmpty(t *testing.T) {
	db := newDiffCovDB(t)
	svc := diffCovFloorSvc(db)
	ctx := context.Background()

	require.NoError(t, svc.validateOrgOptional(ctx, nil), "nil 覆盖值应放行（继承楼宇机构）")
	empty := ""
	require.NoError(t, svc.validateOrgOptional(ctx, &empty), "空串覆盖值应放行")
	blank := "   \t  "
	require.NoError(t, svc.validateOrgOptional(ctx, &blank), "全空白 trim 后为空，应放行")
}

// TestDiffCov_ValidateOrgOptional_InvalidUUID 非法 UUID 被 1001 参数错误拒绝。
func TestDiffCov_ValidateOrgOptional_InvalidUUID(t *testing.T) {
	db := newDiffCovDB(t)
	svc := diffCovFloorSvc(db)
	bad := "部门名称不是UUID"
	err := svc.validateOrgOptional(context.Background(), &bad)
	require.Error(t, err, "非法 UUID 必须拒绝")
	var appErr *apperrors.AppError
	require.True(t, errors.As(err, &appErr), "应包装为 AppError, got %T: %v", err, err)
	assert.Equal(t, apperrors.CodeParamError, appErr.GetCode())
	assert.Contains(t, err.Error(), "合法 UUID")
}

// TestDiffCov_ValidateOrgOptional_DeptNotFound 合法 UUID 但 sys_dept 无此行 → 1001。
func TestDiffCov_ValidateOrgOptional_DeptNotFound(t *testing.T) {
	db := newDiffCovDB(t)
	svc := diffCovFloorSvc(db)
	ghost := diffCovDeptGhost
	err := svc.validateOrgOptional(context.Background(), &ghost)
	require.Error(t, err, "合法 UUID 但部门不存在应拒绝")
	var appErr *apperrors.AppError
	require.True(t, errors.As(err, &appErr), "应包装为 AppError, got %T: %v", err, err)
	assert.Equal(t, apperrors.CodeParamError, appErr.GetCode())
	assert.Contains(t, err.Error(), "不存在")
}

// TestDiffCov_ValidateOrgOptional_Valid 合法 UUID 且部门存在 → 放行（含前后空格 trim）。
func TestDiffCov_ValidateOrgOptional_Valid(t *testing.T) {
	db := newDiffCovDB(t)
	diffCovSeedDept(t, db, diffCovDeptHQ)
	svc := diffCovFloorSvc(db)
	padded := "  " + diffCovDeptHQ + "  "
	require.NoError(t, svc.validateOrgOptional(context.Background(), &padded),
		"trim 后合法且存在的部门应放行")
}

// TestDiffCov_ValidateOrgOptional_DBError Count 查询报错时原样返回 DB 错误。
func TestDiffCov_ValidateOrgOptional_DBError(t *testing.T) {
	db := newDiffCovDB(t)
	svc := diffCovFloorSvc(db)
	require.NoError(t, db.Exec(`DROP TABLE sys_dept`).Error)
	valid := diffCovDeptHQ
	err := svc.validateOrgOptional(context.Background(), &valid)
	require.Error(t, err, "sys_dept 缺失时 Count 应报错并原样上抛")
	assert.Contains(t, err.Error(), "sys_dept", "错误信息应含缺失表名")
}

// TestDiffCov_ValidateOrgOptional_ViaCreate Create 入口联调：非法 org 在创建前拦下，
// 合法 org 可落库（覆盖 Create 调用点 + validateOrgOptional 成功/失败两态）。
func TestDiffCov_ValidateOrgOptional_ViaCreate(t *testing.T) {
	db := newDiffCovDB(t)
	diffCovSeedDept(t, db, diffCovDeptHQ)
	// Create 的 validateBuilding 与 updateBuildingFloorCount 需要楼宇表。
	require.NoError(t, db.Exec(`
		CREATE TABLE ops_buildings (id TEXT PRIMARY KEY, name TEXT, org_id TEXT,
			total_floors INTEGER DEFAULT 0, deleted_at DATETIME,
			created_at DATETIME, updated_at DATETIME)`).Error)
	require.NoError(t, db.AutoMigrate(&operations.OpsFloor{}))
	require.NoError(t, db.Exec(
		`INSERT INTO ops_buildings (id, name, org_id) VALUES ('b-diffcov', '测试楼', ?)`, diffCovDeptHQ).Error)

	// 非法 org：Create 直接拒绝，不落库。
	bad := "not-a-uuid"
	fBad := &operations.OpsFloor{Name: "BAD", FloorNo: "99", BuildingID: "b-diffcov", OrgID: &bad}
	err := NewFloorService(db).Create(context.Background(), fBad)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "合法 UUID")
	var cnt int64
	require.NoError(t, db.Table("ops_floors").Where("floor_no = ?", "99").Count(&cnt).Error)
	assert.Equal(t, int64(0), cnt, "非法 org 不得落库")

	// 合法 org：Create 成功落库。
	good := diffCovDeptHQ
	fGood := &operations.OpsFloor{Name: "GOOD", FloorNo: "1", BuildingID: "b-diffcov", OrgID: &good}
	require.NoError(t, NewFloorService(db).Create(context.Background(), fGood))
	require.NoError(t, db.Table("ops_floors").Where("floor_no = ?", "1").Count(&cnt).Error)
	assert.Equal(t, int64(1), cnt, "合法 org 应成功落库")
}

// ---------------------------------------------------------------------
// 2) floor_service.SearchFloorOptions orgId 分支（门禁行 435-439 附近）
//    len(deptIDs)==0 提前返回 / Pluck 报错 / 正常命中楼层覆盖。
// ---------------------------------------------------------------------

// TestDiffCov_SearchFloorOptions_NoDept 子树展开为空（orgId 不存在）→ 返回空下拉，
// 命中门禁「len(deptIDs)==0 { return result, nil }」行。
func TestDiffCov_SearchFloorOptions_NoDept(t *testing.T) {
	db := setupFloorOrgOverrideDB(t)
	svc := NewFloorService(db)
	opts, err := svc.SearchFloorOptions(context.Background(), map[string]interface{}{
		"orgId": diffCovDeptGhost, // 部门表里不存在该 UUID
	})
	require.NoError(t, err)
	assert.Empty(t, opts, "部门子树为空时应返回空下拉而非全量")
}

// TestDiffCov_SearchFloorOptions_DBError 删除 sys_dept 制造 Pluck 错误 → 返回 error。
func TestDiffCov_SearchFloorOptions_DBError(t *testing.T) {
	db := setupFloorOrgOverrideDB(t)
	require.NoError(t, db.Exec(`DROP TABLE sys_dept`).Error)
	svc := NewFloorService(db)
	_, err := svc.SearchFloorOptions(context.Background(), map[string]interface{}{
		"orgId": wsoOrgWuhan,
	})
	require.Error(t, err, "Pluck 子查询失败应向上返回 error")
	assert.Contains(t, err.Error(), "sys_dept")
}

// TestDiffCov_SearchFloorOptions_OrgHit orgId 命中楼层覆盖值 → 下拉只出该机构的楼层。
func TestDiffCov_SearchFloorOptions_OrgHit(t *testing.T) {
	db := setupFloorOrgOverrideDB(t)
	svc := NewFloorService(db)
	opts, err := svc.SearchFloorOptions(context.Background(), map[string]interface{}{
		"orgId": wsoOrgWuhan,
	})
	require.NoError(t, err)
	require.Len(t, opts, 2, "武汉中支应只出 1/3 楼")
	labels := []string{opts[0].Label, opts[1].Label}
	assert.ElementsMatch(t, []string{"1楼", "3楼"}, labels)
}

// ---------------------------------------------------------------------
// 3) infopoint_service.filterScope OrgID 分支（门禁行 206）
//    有效机构 = 楼层覆盖优先 / 空串继承楼宇；含子部门递归；空 OrgID 不过滤。
// ---------------------------------------------------------------------

// diffCovInfoPointDB 建信息点 List 所需最小 schema 与跨机构数据。
// 复用 setupFloorOrgOverrideDB 的楼宇/楼层/工位/部门，再补 ops_info_points 与
// joinScope 用到的 sys_user / sys_files（AutoMigrate 不含它们）。
func diffCovInfoPointDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := setupFloorOrgOverrideDB(t)
	// AutoMigrate 直接返回 error；gorm.DB 的 .Error 是 func() string 方法值，不能当 error 用。
	require.NoError(t, db.AutoMigrate(&operations.OpsInfoPoint{}))
	// setupFloorOrgOverrideDB 已建 sys_user / sys_files 最小表。
	require.NoError(t, db.Exec(`
		CREATE TABLE IF NOT EXISTS sys_user (id TEXT PRIMARY KEY, nickname TEXT, deleted_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE IF NOT EXISTS sys_files (id TEXT PRIMARY KEY, storage_path TEXT, deleted_at DATETIME)`).Error)
	// 每层一个信息点：4F 继承本部、1F/3F 覆盖武汉中支。
	require.NoError(t, db.Exec(`
		INSERT INTO ops_info_points (id, name, info_point_type, workstation_id, status, created_at) VALUES
			('ip-4f', 'IP-4F', 'network', 'ws-4f', 0, CURRENT_TIMESTAMP),
			('ip-1f', 'IP-1F', 'network', 'ws-1f', 0, CURRENT_TIMESTAMP),
			('ip-3f', 'IP-3F', 'power',   'ws-3f', 0, CURRENT_TIMESTAMP)`).Error)
	return db
}

// TestDiffCov_InfoPointList_OrgFilter 机构筛选走 EXISTS 子查询 + 有效机构表达式。
func TestDiffCov_InfoPointList_OrgFilter(t *testing.T) {
	db := diffCovInfoPointDB(t)
	svc := NewInfoPointService(db)
	ctx := context.Background()

	// 武汉中支：楼层覆盖命中 1F/3F 两条信息点。
	resWH, err := svc.List(ctx, requests.InfoPointListRequest{OrgID: wsoOrgWuhan})
	require.NoError(t, err)
	assert.Equal(t, int64(2), resWH.Total, "楼层覆盖到武汉中支的信息点应被筛出")

	// 本部：只命中继承楼宇的 4F（覆盖楼层不再算本部）。
	resHQ, err := svc.List(ctx, requests.InfoPointListRequest{OrgID: wsoOrgHQ})
	require.NoError(t, err)
	assert.Equal(t, int64(1), resHQ.Total, "继承楼宇机构的信息点应被筛出")

	// 不存在的部门：空结果。
	resGhost, err := svc.List(ctx, requests.InfoPointListRequest{OrgID: diffCovDeptGhost})
	require.NoError(t, err)
	assert.Equal(t, int64(0), resGhost.Total, "无匹配部门时不应漏出信息点")

	// 空 OrgID：不追加 EXISTS 过滤，三条全出。
	resAll, err := svc.List(ctx, requests.InfoPointListRequest{})
	require.NoError(t, err)
	assert.Equal(t, int64(3), resAll.Total, "空 OrgID 不应过滤")
}

// TestDiffCov_InfoPointList_OrgInheritEmptyString 楼层 org_id 空串应继承楼宇机构。
func TestDiffCov_InfoPointList_OrgInheritEmptyString(t *testing.T) {
	db := diffCovInfoPointDB(t)
	require.NoError(t, db.Exec(`UPDATE ops_floors SET org_id = '' WHERE id = ?`, wsoFloorWuhan).Error)
	svc := NewInfoPointService(db)

	res, err := svc.List(context.Background(), requests.InfoPointListRequest{OrgID: wsoOrgHQ})
	require.NoError(t, err)
	assert.Equal(t, int64(2), res.Total, "空串覆盖值应继承楼宇(本部)：4F + 1F")

	resWH, err := svc.List(context.Background(), requests.InfoPointListRequest{OrgID: wsoOrgWuhan})
	require.NoError(t, err)
	assert.Equal(t, int64(1), resWH.Total, "空串楼层不应再算武汉中支")
}

// ---------------------------------------------------------------------
// 4) reference_resolver.queryUserDeptMap（门禁行 526-527、547-548）
//    空 slice / 400 分块边界 / 软删/空/NULL/悬空部门过滤 / 查询报错。
// ---------------------------------------------------------------------

// TestDiffCov_QueryUserDeptMap_Empty 空 slice 直接返回空 map，不查库。
func TestDiffCov_QueryUserDeptMap_Empty(t *testing.T) {
	db := setupResolver77DB(t) // 该夹具无 sys_user 表 → 若误查库必然报错
	got, err := queryUserDeptMap(context.Background(), db, nil)
	require.NoError(t, err)
	assert.Empty(t, got, "nil 输入应返回空 map")
	got2, err := queryUserDeptMap(context.Background(), db, []string{})
	require.NoError(t, err)
	assert.Empty(t, got2, "空 slice 应返回空 map")
}

// TestDiffCov_QueryUserDeptMap_Normal 正常映射 + 各类脏数据过滤。
func TestDiffCov_QueryUserDeptMap_Normal(t *testing.T) {
	db := newDiffCovDB(t)
	diffCovSeedDept(t, db, diffCovDeptHQ)
	require.NoError(t, db.Exec(`
		INSERT INTO sys_user (id, username, nickname, dept_id, deleted_at) VALUES
			(?, 'u-a', '张三', ?, NULL),
			('u-b', 'u-b', '空部门', '', NULL),
			('u-c', 'u-c', 'NULL部门', NULL, NULL),
			('u-d', 'u-d', '悬空部门', ?, NULL),
			('u-e', 'u-e', '软删用户', ?, '2026-01-01')`,
		diffCovUserA, diffCovDeptHQ, diffCovDeptGhost, diffCovDeptHQ).Error)

	got, err := queryUserDeptMap(context.Background(), db, []string{
		diffCovUserA, "u-b", "u-c", "u-d", "u-e", "u-not-in-ids",
	})
	require.NoError(t, err)
	require.Len(t, got, 1, "仅有效且未软删且部门存在的用户应入 map")
	assert.Equal(t, diffCovDeptHQ, got[diffCovUserA])
	for _, bad := range []string{"u-b", "u-c", "u-d", "u-e"} {
		assert.NotContains(t, got, bad, "%s 应被过滤", bad)
	}
}

// TestDiffCov_QueryUserDeptMap_ChunkBoundary >400 触发分块：401 个 ID 分两块，
// 两块里的用户都能被查到（验证 i+=chunkSize 与 end 截断）。
func TestDiffCov_QueryUserDeptMap_ChunkBoundary(t *testing.T) {
	db := newDiffCovDB(t)
	diffCovSeedDept(t, db, diffCovDeptHQ)
	ids := make([]string, 0, 401)
	for i := 0; i < 401; i++ {
		id := fmt.Sprintf("u-chunk-%03d", i)
		ids = append(ids, id)
		require.NoError(t, db.Exec(
			`INSERT INTO sys_user (id, username, dept_id) VALUES (?, ?, ?)`,
			id, id, diffCovDeptHQ).Error)
	}
	// 第 401 个 ID 落在第二块（index 400）；第一块 400 个。
	got, err := queryUserDeptMap(context.Background(), db, ids)
	require.NoError(t, err)
	assert.Len(t, got, 401, "跨 400 边界的两块用户都应解析到部门")
	assert.Equal(t, diffCovDeptHQ, got["u-chunk-000"], "第一块首用户")
	assert.Equal(t, diffCovDeptHQ, got["u-chunk-400"], "第二块（末块短）用户")
}

// TestDiffCov_QueryUserDeptMap_DBError sys_user 表缺失 → 返回 error（门禁行 547-548）。
func TestDiffCov_QueryUserDeptMap_DBError(t *testing.T) {
	db := newDiffCovDB(t)
	require.NoError(t, db.Exec(`DROP TABLE sys_user`).Error)
	got, err := queryUserDeptMap(context.Background(), db, []string{diffCovUserA})
	require.Error(t, err, "查询报错应上抛")
	assert.Nil(t, got, "错误时不应返回部分 map")
	assert.Contains(t, err.Error(), "sys_user")
}

// ---------------------------------------------------------------------
// 5) reference_resolver.chunksOf（门禁行 562-566）
//    size<=0 / 空 slice / 正常分块 + 末块短。
// ---------------------------------------------------------------------

// TestDiffCov_ChunksOf size<=0 单块、空 slice 返回 nil、正常末块短。
func TestDiffCov_ChunksOf(t *testing.T) {
	// 空 slice：无论 size 都返回 nil（门禁 564-566）。
	assert.Nil(t, chunksOf([]int{}, 3))
	assert.Nil(t, chunksOf[int](nil, 0))

	// size<=0：把 size 归一为 len(items) → 单块（门禁 562-563）。
	items := []int{1, 2, 3, 4, 5}
	assert.Equal(t, [][]int{{1, 2, 3, 4, 5}}, chunksOf(items, 0))
	assert.Equal(t, [][]int{{1, 2, 3, 4, 5}}, chunksOf(items, -7))

	// 正常分块 + 末块短（门禁 567-575）。
	assert.Equal(t, [][]int{{1, 2}, {3, 4}, {5}}, chunksOf(items, 2))
	assert.Equal(t, [][]int{{1, 2, 3, 4, 5}}, chunksOf(items, 10))
	// 字符串泛型也可用。
	assert.Equal(t, [][]string{{"a", "b"}, {"c"}}, chunksOf([]string{"a", "b", "c"}, 2))
}

// ---------------------------------------------------------------------
// 6) workstation_service 构造器 / setter / 缓存失效
//    （门禁行 270-288、644-645）
// ---------------------------------------------------------------------

// diffCovSpyCache 记录 DeleteByPattern 调用的 CacheProvider 探针。
type diffCovSpyCache struct {
	*base.NoOpCacheProvider
	patterns []string
	failOn   string
}

func (c *diffCovSpyCache) DeleteByPattern(_ context.Context, pattern string) error {
	c.patterns = append(c.patterns, pattern)
	if c.failOn == pattern {
		return errors.New("模拟缓存失效失败")
	}
	return nil
}

// TestDiffCov_WorkstationCacheWiring 验证两个构造器 + WithCacheProvider setter 的
// 返回值与字段注入语义（门禁 270-275、286-288）。
func TestDiffCov_WorkstationCacheWiring(t *testing.T) {
	db := newDiffCovDB(t)
	spy := &diffCovSpyCache{NoOpCacheProvider: &base.NoOpCacheProvider{}}

	// NewWorkstationServiceWithCache：构造后 cache 已注入。
	svcWith := NewWorkstationServiceWithCache(db, spy)
	implWith, ok := svcWith.(*workstationService)
	require.True(t, ok, "应返回 *workstationService, got %T", svcWith)
	assert.Same(t, spy, implWith.cache, "构造器应注入缓存提供者")
	require.NotNil(t, implWith.repo)
	assert.Same(t, db, implWith.db)

	// NewWorkstationService：单参构造不带 cache。
	plain := NewWorkstationService(db)
	implPlain, ok := plain.(*workstationService)
	require.True(t, ok)
	assert.Nil(t, implPlain.cache, "单参构造不应带缓存")

	// WithCacheProvider：返回自身并注入。
	ret := implPlain.WithCacheProvider(spy)
	assert.Same(t, implPlain, ret, "setter 应返回同一实例")
	assert.Same(t, spy, implPlain.cache, "setter 应注入缓存")
}

// TestDiffCov_Backfill_InvalidateCache 执行回填成功后，按两个 pattern 失效缓存；
// 即使 DeleteByPattern 报错也应吞掉并正常返回（门的行 644-645）。
func TestDiffCov_Backfill_InvalidateCache(t *testing.T) {
	db := setupBackfillDeptDB(t)
	spy := &diffCovSpyCache{NoOpCacheProvider: &base.NoOpCacheProvider{}}
	svc := NewWorkstationServiceWithCache(db, spy)

	res, err := svc.BackfillDeptFromUser(context.Background(), false, false)
	require.NoError(t, err)
	assert.Equal(t, 3, res.ToFill)
	assert.Equal(t, []string{"workstation:*", "workstations:*"}, spy.patterns,
		"事务提交后应失效两个 workstation 缓存 pattern")

	// dryRun 不写库也不清缓存。
	spy2 := &diffCovSpyCache{NoOpCacheProvider: &base.NoOpCacheProvider{}}
	db2 := setupBackfillDeptDB(t)
	svc2 := NewWorkstationServiceWithCache(db2, spy2)
	_, err = svc2.BackfillDeptFromUser(context.Background(), true, false)
	require.NoError(t, err)
	assert.Empty(t, spy2.patterns, "dryRun 不应触发缓存失效")

	// 缓存失效报错被 InvalidatePattern 吞掉，不影响回填结果。
	spy3 := &diffCovSpyCache{NoOpCacheProvider: &base.NoOpCacheProvider{}, failOn: "workstation:*"}
	db3 := setupBackfillDeptDB(t)
	svc3 := NewWorkstationServiceWithCache(db3, spy3)
	res3, err := svc3.BackfillDeptFromUser(context.Background(), false, false)
	require.NoError(t, err, "缓存失效失败不应让回填整体失败")
	assert.Equal(t, 3, res3.ToFill)
}

// ---------------------------------------------------------------------
// 7) BackfillDeptFromUser 错误路径 / 统计分支 / 样例补位 / 批量 UPDATE
//    （门禁行 527-528、540-541、543-544、580、633-634、638-639、644-645）
// ---------------------------------------------------------------------

// TestDiffCov_Backfill_QueryError 候选查询失败 → 「查询反推候选工位失败」。
func TestDiffCov_Backfill_QueryError(t *testing.T) {
	db := setupBackfillDeptDB(t)
	require.NoError(t, db.Exec(`DROP TABLE sys_user`).Error)
	svc := NewWorkstationService(db)
	res, err := svc.BackfillDeptFromUser(context.Background(), true, false)
	require.Error(t, err)
	assert.Nil(t, res)
	assert.Contains(t, err.Error(), "查询反推候选工位失败")
	assert.Contains(t, err.Error(), "sys_user")
}

// TestDiffCov_Backfill_StatsError 统计查询（Scan→gorm:row callback）失败 →
// 「统计工位总数失败」。用 callback 精确只打断统计阶段，候选 Find 不受影响。
func TestDiffCov_Backfill_StatsError(t *testing.T) {
	db := newDiffCovDB(t)
	// 候选 Find 走 gorm:query；统计 Scan 走 gorm:row。替换 row callback 让后者必失败。
	require.NoError(t, db.Callback().Row().Replace("gorm:row", func(tx *gorm.DB) {
		tx.AddError(errors.New("模拟统计查询失败"))
	}))
	svc := NewWorkstationService(db)
	res, err := svc.BackfillDeptFromUser(context.Background(), true, false)
	require.Error(t, err)
	assert.Nil(t, res)
	assert.Contains(t, err.Error(), "统计工位总数失败")
	assert.Contains(t, err.Error(), "模拟统计查询失败")
}

// TestDiffCov_Backfill_StatsLenZero 空工位表时条件聚合恒有一行(0,0)，
// len(stats)!=0：断言 SkippedNoUser/BadUserDept 归零且提前返回不写库。
func TestDiffCov_Backfill_StatsLenZero(t *testing.T) {
	db := newDiffCovDB(t)
	svc := NewWorkstationService(db)
	res, err := svc.BackfillDeptFromUser(context.Background(), false, false)
	require.NoError(t, err)
	assert.Equal(t, 0, res.Candidates)
	assert.Equal(t, 0, res.ToFill)
	assert.Equal(t, 0, res.SkippedNoUser)
	assert.Equal(t, 0, res.SkippedBadUserDept)
	assert.Empty(t, res.Samples, "无候选时样例为空")
}

// TestDiffCov_Backfill_UpdateError 批量 CASE UPDATE 报错 → 事务回滚且错误上抛
// （门禁 633-634、638-639）。通过触发器让 UPDATE 失败。
func TestDiffCov_Backfill_UpdateError(t *testing.T) {
	db := setupBackfillDeptDB(t)
	require.NoError(t, db.Exec(`
		CREATE TRIGGER diffcov_block_update BEFORE UPDATE ON sys_workstation
		BEGIN SELECT RAISE(ABORT, '模拟批量回填失败'); END`).Error)
	svc := NewWorkstationService(db)
	res, err := svc.BackfillDeptFromUser(context.Background(), false, false)
	require.Error(t, err, "UPDATE 失败应上抛")
	assert.Nil(t, res)
	assert.Contains(t, err.Error(), "批量回填工位部门失败")
	// 事务回滚：第一条待填补工位仍是空部门。
	dept, _ := readBackfillDept(t, db, "ws-null")
	assert.Empty(t, dept, "事务应回滚，不得留下半写入")
}

// TestDiffCov_Backfill_SkipSampleBudget 21 条「已一致」候选 + 1 条待填补：
// skip 样例不得挤占 fill 样例（门禁 579-580 的 continue 分支）；样例数 capped 20。
func TestDiffCov_Backfill_SkipSampleBudget(t *testing.T) {
	db := setupBackfillDeptDB(t)
	// 追加 21 条 dept_id 已等于用户部门的工位（skip 分支）。
	for i := 0; i < 21; i++ {
		require.NoError(t, db.Exec(
			`INSERT INTO sys_workstation (id, workstation_name, dept_id, user_id, status)
			 VALUES (?, ?, ?, ?, 1)`,
			fmt.Sprintf("ws-skip-%02d", i), fmt.Sprintf("W-SKIP-%02d", i),
			bdDeptA, bdUserA).Error)
	}
	svc := NewWorkstationService(db)
	res, err := svc.BackfillDeptFromUser(context.Background(), true, false)
	require.NoError(t, err)
	assert.Equal(t, 23, res.SkippedAlreadyValid, "原有 2 条 skip（ws-valid 一致 + ws-mism 不纠正）+ 21 新增 skip")
	assert.Equal(t, 3, res.ToFill)
	assert.Len(t, res.Samples, 20, "样例上限 20")
	fillSamples := 0
	for _, s := range res.Samples {
		if s.Action == "fill" {
			fillSamples++
		}
	}
	assert.Equal(t, 3, fillSamples, "3 条 fill 样例都应进入预览（skip 不挤占）")
	// 幂等二跑：ToFill=0 时预览不再被 20 行 skip 占满（skip 只补 maxSamples-1）。
	_, err = svc.BackfillDeptFromUser(context.Background(), false, false)
	require.NoError(t, err)
	res2, err := svc.BackfillDeptFromUser(context.Background(), true, false)
	require.NoError(t, err)
	assert.Equal(t, 0, res2.ToFill, "二跑幂等")
	assert.Len(t, res2.Samples, 19, "全 skip 时样例上限被压到 19（skip 补位策略）")
	for _, s := range res2.Samples {
		assert.Equal(t, "skip", s.Action)
	}
}

// TestDiffCov_Backfill_ExecuteBatchCase 正常执行：批量 CASE UPDATE 写库、
// fixMismatch 纠正、status 不变、缓存失效、幂等（门禁 610-645 全链路）。
func TestDiffCov_Backfill_ExecuteBatchCase(t *testing.T) {
	db := setupBackfillDeptDB(t)
	spy := &diffCovSpyCache{NoOpCacheProvider: &base.NoOpCacheProvider{}}
	svc := NewWorkstationServiceWithCache(db, spy)
	ctx := context.Background()

	res, err := svc.BackfillDeptFromUser(ctx, false, true)
	require.NoError(t, err)
	assert.Equal(t, 3, res.ToFill)
	assert.Equal(t, 1, res.ToCorrect, "fixMismatch 应纠正 ws-mism")

	deptNull, statusNull := readBackfillDept(t, db, "ws-null")
	assert.Equal(t, bdDeptA, deptNull, "CASE UPDATE 应回填部门A")
	assert.Equal(t, 0, statusNull, "status 绝不改动")
	deptMism, _ := readBackfillDept(t, db, "ws-mism")
	assert.Equal(t, bdDeptA, deptMism, "fixMismatch 应纠正为部门A")
	deptValid, _ := readBackfillDept(t, db, "ws-valid")
	assert.Equal(t, bdDeptA, deptValid)
	assert.Equal(t, []string{"workstation:*", "workstations:*"}, spy.patterns)

	// 幂等：第二次零变更。
	spy.patterns = nil
	res2, err := svc.BackfillDeptFromUser(ctx, false, true)
	require.NoError(t, err)
	assert.Equal(t, 0, res2.ToFill)
	assert.Equal(t, 0, res2.ToCorrect)
	assert.Empty(t, spy.patterns, "无待更新行时不应清缓存（dryRun||len(toUpdate)==0 提前返回）")
}

// TestDiffCov_Backfill_ChunkedCaseUpdate >400 条待填补工位 → chunksOf 分两块
// 批量 CASE UPDATE，全部回填成功（覆盖门禁 657-682 的分块循环）。
func TestDiffCov_Backfill_ChunkedCaseUpdate(t *testing.T) {
	db := setupBackfillDeptDB(t)
	// 追加 405 条空部门 + 用户A 的工位，加上原夹具 3 条 fill 共 408 条待更新。
	for i := 0; i < 405; i++ {
		require.NoError(t, db.Exec(
			`INSERT INTO sys_workstation (id, workstation_name, dept_id, user_id, status)
			 VALUES (?, ?, NULL, ?, 0)`,
			fmt.Sprintf("ws-bulk-%03d", i), fmt.Sprintf("W-BULK-%03d", i), bdUserA).Error)
	}
	svc := NewWorkstationService(db)
	res, err := svc.BackfillDeptFromUser(context.Background(), false, false)
	require.NoError(t, err)
	assert.Equal(t, 408, res.ToFill, "408 条待填补应全部计入")
	assert.LessOrEqual(t, len(res.Samples), 20)

	// 抽查第一块与跨 400 边界的第二块都写库成功。
	for _, id := range []string{"ws-bulk-000", "ws-bulk-399", "ws-bulk-400", "ws-bulk-404"} {
		dept, _ := readBackfillDept(t, db, id)
		assert.Equal(t, bdDeptA, dept, "%s 应被分块 CASE UPDATE 回填", id)
	}
}

// TestDiffCov_Backfill_DryRunFlags dryRun 与 fixMismatch 标志透传 + 零写库。
func TestDiffCov_Backfill_DryRunFlags(t *testing.T) {
	db := setupBackfillDeptDB(t)
	svc := NewWorkstationService(db)
	res, err := svc.BackfillDeptFromUser(context.Background(), true, true)
	require.NoError(t, err)
	assert.True(t, res.DryRun)
	assert.True(t, res.FixMismatch)
	assert.Equal(t, 3, res.ToFill)
	assert.Equal(t, 1, res.ToCorrect, "dryRun 也预览 fixMismatch 的纠正")
	dept, _ := readBackfillDept(t, db, "ws-null")
	assert.Empty(t, dept, "dryRun 必须零写库")
}

// TestDiffCov_Backfill_TrimAndCaseInsensitive 当前部门带空格/大小写差异时
// 不误判为不一致（覆盖 TrimSpace + EqualFold 分支）。
func TestDiffCov_Backfill_TrimAndCaseInsensitive(t *testing.T) {
	db := setupBackfillDeptDB(t)
	padded := "  " + strings.ToUpper(bdDeptA) + "  "
	require.NoError(t, db.Exec(
		`INSERT INTO sys_workstation (id, workstation_name, dept_id, user_id, status)
		 VALUES ('ws-pad', 'W-PAD', ?, ?, 1)`, padded, bdUserA).Error)
	svc := NewWorkstationService(db)
	res, err := svc.BackfillDeptFromUser(context.Background(), true, true)
	require.NoError(t, err)
	assert.Equal(t, 1, res.ToCorrect, "仅 ws-mism 需纠正，空格/大小写差异不算不一致")
	dept, _ := readBackfillDept(t, db, "ws-pad")
	assert.Equal(t, padded, dept, "trim+EqualFold 判定为一致，不应被改写")
}

// 编译期断言：确保本文件不依赖未使用的导入。
