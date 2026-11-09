// Package operations: Phase 150 ① 侧信道根治 A —— 差分攻击回归测试。
//
// 目标（REQUIREMENTS.md:70 / PLAN T4）：任何非 All 档用户，通过单请求或多次
// 请求组合，都无法从响应体（含状态码、任何数值 / 序号 / 数组长度）推导出
// 「可见集之外的资源规模」或「越界资源是否存在 / 有多少」。
//
// 手段：静态断言恒跑 + SQLite 真跑（F-6）；TEST_PG_DSN 仍为 DIALECT-04③ 前置
// 依赖，本期不建 services: postgres、不伪造 PG 双跑。全部 fixture 复用既有
// helper（write403Ctx / write403DeptOnly / GetExcelConfig），零新依赖。
package operations

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	operationsmodels "github.com/xingran-next/xingran-go-backend/internal/models/operations"
	"github.com/xingran-next/xingran-go-backend/pkg/datascope"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// 差分攻击 fixture 的部门 / 用户 ID（与 scope_write_403_test.go 常量同款 UUID 文本，
// 保证 CAST(... AS TEXT) IN ? 双方言语义一致）。
const (
	diffAtkDeptA    = "aaaaaaaa-0000-0000-0000-000000000001" // 可见部门
	diffAtkDeptB    = "bbbbbbbb-0000-0000-0000-000000000002" // 越界部门
	diffAtkDeptASub = "cccccccc-0000-0000-0000-000000000003" // 可见子部门（DeptChild 档用，无楼宇归属）
	diffAtkUserA    = "11111111-0000-0000-0000-0000000000a1" // 部门 A 有效用户
	diffAtkUserB    = "22222222-0000-0000-0000-0000000000b2" // 部门 B 有效用户
	diffAtkUserBad  = "33333333-0000-0000-0000-0000000000c3" // 脏部门用户（dept_id 悬空）
)

var diffAtkForbiddenKeys = []string{
	"scopeTotal", "scopeVisible", "scopeOutOfScope", "scopeDeptTotal",
	"skippedNoUser", "skippedBadUserDept",
}

// diffAtkScopeDept / diffAtkScopeDeptChild 是「两个相邻 scope」：都基于同一
// 可见集（楼宇归属 deptA），后者额外含一个子部门但数据里无该子部门楼宇，
// 故可见集规模 M 不变。用于 (b) 按档位分组对比。
func diffAtkScopeDept() datascope.ScopeResult {
	return datascope.ScopeResult{DeptIDs: []string{diffAtkDeptA}}
}

func diffAtkScopeDeptChild() datascope.ScopeResult {
	return datascope.ScopeResult{DeptIDs: []string{diffAtkDeptA, diffAtkDeptASub}}
}

// buildDiffAttackFixture 构造「全库 N 行、可见 M 行」的 SQLite 内存夹具：
//   - 可见工位 visible 个：挂 deptA 楼宇 → 楼层，带有效用户(部门A)；
//   - 越界工位 outOfScope 个：挂 deptB 楼宇 → 楼层，其中一半带有效用户(部门B)、
//     一半带脏部门用户，覆盖 workstation_service.go:644 的候选条件分支；
//   - 越界行虽带有效用户，但被 scopeWriteWorkstationVisibleSQL 的 visibleCond
//     裁掉，故响应体数值仅随 M 变化、不随 N−M 变化。
//
// 两组 fixture 的 sys_dept / sys_user 表内容逐字一致，保证 scopeDeptNames
// 不干扰「同档位两组 N 不同但 M 相同响应体逐字节相等」的断言。
func buildDiffAttackFixture(t *testing.T, visible, outOfScope int) (*gorm.DB, WorkstationService) {
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
	require.NoError(t, db.AutoMigrate(
		&operationsmodels.OpsBuilding{},
		&operationsmodels.OpsFloor{},
		&models.Workstation{},
		&operationsmodels.OpsInfoPoint{},
	))
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_dept (id TEXT PRIMARY KEY, dept_name TEXT, dept_code TEXT,
			ancestors TEXT, parent_id TEXT, deleted_at DATETIME);
		CREATE TABLE sys_user (id TEXT PRIMARY KEY, dept_id TEXT, nickname TEXT, deleted_at DATETIME);
	`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_dept (id, dept_name) VALUES
		(?, '部门A'), (?, '部门B'), (?, '子部门A')`,
		diffAtkDeptA, diffAtkDeptB, diffAtkDeptASub).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_user (id, dept_id, nickname) VALUES
		(?, ?, '用户A'), (?, ?, '用户B'), (?, 'ghost-dept', '用户C')`,
		diffAtkUserA, diffAtkDeptA, diffAtkUserB, diffAtkDeptB, diffAtkUserBad).Error)

	bA := &operationsmodels.OpsBuilding{Name: "楼宇A", OrgID: diffAtkDeptA}
	bB := &operationsmodels.OpsBuilding{Name: "楼宇B", OrgID: diffAtkDeptB}
	require.NoError(t, db.Create(bA).Error)
	require.NoError(t, db.Create(bB).Error)
	flA := &operationsmodels.OpsFloor{Name: "楼层A", FloorNo: "1", BuildingID: bA.ID}
	flB := &operationsmodels.OpsFloor{Name: "楼层B", FloorNo: "1", BuildingID: bB.ID}
	require.NoError(t, db.Create(flA).Error)
	require.NoError(t, db.Create(flB).Error)

	uA, uB, uBad := diffAtkUserA, diffAtkUserB, diffAtkUserBad
	for i := 0; i < visible; i++ {
		w := &models.Workstation{
			WorkstationName: fmt.Sprintf("可见工位-%d", i),
			FloorID:         &flA.ID,
			UserID:          &uA,
		}
		require.NoError(t, db.Create(w).Error)
	}
	for i := 0; i < outOfScope; i++ {
		uid := &uB
		if i%2 == 1 {
			uid = &uBad // 脏部门用户：覆盖「有用户但部门无效」候选条件
		}
		w := &models.Workstation{
			WorkstationName: fmt.Sprintf("越界工位-%d", i),
			FloorID:         &flB.ID,
			UserID:          uid,
		}
		require.NoError(t, db.Create(w).Error)
	}
	return db, NewWorkstationService(db)
}

// diffAtkMarshal 把结果序列化为响应体字节（带错误断言）。
func diffAtkMarshal(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return string(b)
}

// (a) 响应体零全库 / 越界规模字段（SIDECHANNEL-01）。
func TestSideChannel_Backfill_ResponseHasNoOutOfScopeFields(t *testing.T) {
	f := write403BuildFixture(t)
	ctx := write403Ctx(write403DeptOnly(write403DeptA))
	res, err := f.WSSvc.BackfillDeptFromUser(ctx, true, false)
	require.NoError(t, err)

	body := diffAtkMarshal(t, res)
	for _, key := range diffAtkForbiddenKeys {
		assert.NotContains(t, body, `"`+key+`"`, "响应体不得回显 %s", key)
	}
	// 可见集内业务字段必须保留（业务数值零回归）。
	for _, key := range []string{"candidates", "toFill", "toCorrect", "skippedAlreadyValid", "samples"} {
		assert.Contains(t, body, `"`+key+`"`, "可见集内字段 %s 应保留", key)
	}
}

// (b) 核心差分攻击：两组 fixture 可见行数 M 固定、全库 N 不同（N−M 不同），
// 对每组用 Dept 与 DeptChild 两个相邻 scope 各请求一次（共 4 次），按档位
// 分组断言「同档位下两组响应体逐字节相等」——所有数值字段仅随 M 变化、
// 不随 N−M 变化。
func TestSideChannel_Backfill_DiffAttack_OnlyVariesWithM(t *testing.T) {
	const visibleM = 2
	// 组 1：N=5 / M=2 / 越界 3；组 2：N=9 / M=2 / 越界 7。
	scenarios := []struct {
		name       string
		outOfScope int
	}{
		{"N5_M2", 3},
		{"N9_M2", 7},
	}
	scopes := []struct {
		name string
		s    datascope.ScopeResult
	}{
		{"Dept", diffAtkScopeDept()},
		{"DeptChild", diffAtkScopeDeptChild()},
	}

	// responses[scopeName][scenarioName] = 响应体字节。
	responses := map[string]map[string]string{}
	for _, sc := range scopes {
		responses[sc.name] = map[string]string{}
		for _, s := range scenarios {
			_, svc := buildDiffAttackFixture(t, visibleM, s.outOfScope)
			res, err := svc.BackfillDeptFromUser(write403Ctx(sc.s), true, false)
			require.NoError(t, err, "%s/%s 请求失败", sc.name, s.name)
			body := diffAtkMarshal(t, res)
			responses[sc.name][s.name] = body

			// 非 All 档 Samples 恒空、Candidates 仅随 M（<=M）、不随 N−M。
			assert.Empty(t, res.Samples, "%s/%s 非 All 档 Samples 恒空", sc.name, s.name)
			assert.LessOrEqual(t, res.Candidates, visibleM,
				"%s/%s Candidates 不得超过可见集规模 M", sc.name, s.name)
			assert.Equal(t, "Custom/Dept/DeptChild", res.ScopeTier)
			for _, key := range diffAtkForbiddenKeys {
				assert.NotContains(t, body, `"`+key+`"`, "%s/%s 响应体不得回显 %s", sc.name, s.name, key)
			}
		}
		// 同档位：N=5 与 N=9（N−M 3 vs 7）响应体逐字节相等。
		assert.Equal(t, responses[sc.name]["N5_M2"], responses[sc.name]["N9_M2"],
			"%s 档：全库 N 不同但可见 M 相同，响应体必须逐字节相等（不可差分出越界规模）", sc.name)
	}
}

// (c) 导入差分攻击：同一份上传记录（固定 1 条用户自有校验错误），改变越界
// 行数 N−M，断言 failed / errors 长度不变、越界行不进 Errors、outOfScope 仍
// 正确递增（审计链保留）、ImportResult 序列化不含 scopeOutOfScope。
func TestSideChannel_Import_DiffAttack_SilentSkip(t *testing.T) {
	config, ok := GetExcelConfig("building")
	require.True(t, ok)
	ctx := write403Ctx(write403DeptOnly(diffAtkDeptA))

	// 固定「用户上传文件自身」的 1 条校验错误（与越界无关）。
	newUserOwnedResult := func() *ImportResult {
		return &ImportResult{
			Failed: 1,
			Errors: []ImportError{{Row: 2, Field: "name", Error: "必填项缺失"}},
		}
	}
	// 固定 2 条可见 building 行；越界行数按场景变化。
	makeRecords := func(outOfScope int) []map[string]any {
		records := []map[string]any{
			{"name": "可见楼宇1", "org_id": diffAtkDeptA, "__excel_row__": 3},
			{"name": "可见楼宇2", "org_id": diffAtkDeptA, "__excel_row__": 4},
		}
		for i := 0; i < outOfScope; i++ {
			records = append(records, map[string]any{
				"name":          fmt.Sprintf("越界楼宇%d", i),
				"org_id":        diffAtkDeptB,
				"__excel_row__": 5 + i,
			})
		}
		return records
	}

	scenarios := []struct {
		name       string
		outOfScope int
	}{
		{"N4_越界2", 2},
		{"N8_越界6", 6},
	}
	type outcome struct {
		failed     int
		errorsLen  int
		keptLen    int
		outOfScope int
		body       string
	}
	outs := map[string]outcome{}
	for _, s := range scenarios {
		// building entity 的 effOrg 直接取行内 org_id，无需查库；用内存 db 占位。
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
		require.NoError(t, err)
		result := newUserOwnedResult()
		kept, outOfScope, err := filterScopeWriteImportRows(ctx, db, "building", config, makeRecords(s.outOfScope), result)
		require.NoError(t, err, s.name)

		assert.Equal(t, s.outOfScope, outOfScope, "%s 越界行数仍返回供 operlog 审计链", s.name)
		assert.Len(t, kept, 2, "%s kept 只含可见行", s.name)
		for _, rec := range kept {
			_, leaked := rec["__excel_row__"]
			assert.False(t, leaked, "%s 内部行号标记不得泄漏进 Upsert 字段集", s.name)
		}
		// 越界行静默跳过：failed / errors 只含用户自有校验错误，不随越界行数变化。
		assert.Equal(t, 1, result.Failed, "%s failed 不含越界行数", s.name)
		assert.Len(t, result.Errors, 1, "%s errors 不含越界行", s.name)
		assert.Equal(t, "name", result.Errors[0].Field, "%s errors 仍为用户自有校验错误", s.name)

		// 模拟 excel_service.go:460 的 `result.ScopeOutOfScope = outOfScope` 赋值：
		// 真实导入路径里该内部字段非 0，故若 tag 不是 json:"-" 就一定会序列化进响应体。
		result.ScopeOutOfScope = outOfScope
		body := diffAtkMarshal(t, result)
		assert.NotContains(t, body, "scopeOutOfScope", "%s ImportResult 序列化必须不含 scopeOutOfScope（json:\"-\"）", s.name)
		outs[s.name] = outcome{
			failed: result.Failed, errorsLen: len(result.Errors), keptLen: len(kept),
			outOfScope: outOfScope, body: body,
		}
	}
	// 差分核心：不同 N−M 时 failed / errors 长度相等。
	assert.Equal(t, outs["N4_越界2"].failed, outs["N8_越界6"].failed, "failed 不随越界行数变化")
	assert.Equal(t, outs["N4_越界2"].errorsLen, outs["N8_越界6"].errorsLen, "errors 长度不随越界行数变化")
	assert.Equal(t, outs["N4_越界2"].keptLen, outs["N8_越界6"].keptLen, "kept 只随可见集变化")
	// 响应体的 failed / errors 部分逐字节相同（gin.H 形状不变，F-4 间接泄漏根治）。
	assert.Equal(t, outs["N4_越界2"].body, outs["N8_越界6"].body,
		"同一份上传文件、不同越界行数：ImportResult 响应体逐字节相同")
	// 越界计数仅经返回值进 operlog，不进响应体：两组 outOfScope 不同但 body 相同。
	assert.NotEqual(t, outs["N4_越界2"].outOfScope, outs["N8_越界6"].outOfScope,
		"越界计数确实不同（证明差分条件成立）")
}

// (d) 重复请求不得通过 Row / 数组长度 / total / 状态码枚举出 N。
func TestSideChannel_RepeatedRequests_CannotEnumerateN(t *testing.T) {
	const visibleM = 2
	ctx := write403Ctx(diffAtkScopeDept())

	// 1) 同一非 All 档 scope 重复 10 次：响应体逐字节相等、不含 total / row。
	_, svc := buildDiffAttackFixture(t, visibleM, 7)
	var first string
	for i := 0; i < 10; i++ {
		res, err := svc.BackfillDeptFromUser(ctx, true, false)
		require.NoError(t, err)
		body := diffAtkMarshal(t, res)
		assert.NotContains(t, body, `"total"`, "响应体不得含 total")
		assert.NotContains(t, body, `"row"`, "零校验错误构造下响应体不得含 row")
		assert.NotContains(t, body, `"failed"`, "backfill 响应体不得含 failed")
		if i == 0 {
			first = body
		}
		assert.Equal(t, first, body, "第 %d 次重复请求响应体不得变化（不可枚举 N）", i+1)
	}

	// 2) 多次请求组合：两组 N 不同（N−M 3 vs 7）各跑 3 次，全部响应体相等。
	combined := []string{}
	for _, out := range []int{3, 7} {
		_, s := buildDiffAttackFixture(t, visibleM, out)
		for i := 0; i < 3; i++ {
			res, err := s.BackfillDeptFromUser(ctx, true, false)
			require.NoError(t, err)
			combined = append(combined, diffAtkMarshal(t, res))
		}
	}
	for i := 1; i < len(combined); i++ {
		assert.Equal(t, combined[0], combined[i],
			"单请求 / 多次请求组合均不得随 N−M 变化（第 %d 组）", i)
	}
}

// All 档零回归（MH-5）：admin 的业务数值（candidates / toFill / toCorrect /
// skippedAlreadyValid）与 130 前口径一致 —— All 档不裁剪、全库行都进候选；
// Samples 长度 <= 20；ScopeTier=All。复用 scope_write_403_test.go:327-331 口径。
func TestSideChannel_Backfill_AllTier_ZeroRegression(t *testing.T) {
	const visibleM = 2
	const outOfScope = 3 // 全部越界工位均带有效用户时 All 档候选 = M + (N−M) = 5
	// 单独构造一个「越界行全部带有效用户」的夹具，便于精确断言 All 档数值。
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	t.Cleanup(func() {
		if sqlDB, e := db.DB(); e == nil {
			_ = sqlDB.Close()
		}
	})
	require.NoError(t, db.AutoMigrate(
		&operationsmodels.OpsBuilding{}, &operationsmodels.OpsFloor{},
		&models.Workstation{}, &operationsmodels.OpsInfoPoint{},
	))
	require.NoError(t, db.Exec(`CREATE TABLE sys_dept (id TEXT PRIMARY KEY, dept_name TEXT, deleted_at DATETIME);
		CREATE TABLE sys_user (id TEXT PRIMARY KEY, dept_id TEXT, nickname TEXT, deleted_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_dept (id, dept_name) VALUES (?, '部门A'), (?, '部门B')`,
		diffAtkDeptA, diffAtkDeptB).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_user (id, dept_id) VALUES (?, ?), (?, ?)`,
		diffAtkUserA, diffAtkDeptA, diffAtkUserB, diffAtkDeptB).Error)
	bA := &operationsmodels.OpsBuilding{Name: "楼宇A", OrgID: diffAtkDeptA}
	bB := &operationsmodels.OpsBuilding{Name: "楼宇B", OrgID: diffAtkDeptB}
	require.NoError(t, db.Create(bA).Error)
	require.NoError(t, db.Create(bB).Error)
	flA := &operationsmodels.OpsFloor{Name: "楼层A", FloorNo: "1", BuildingID: bA.ID}
	flB := &operationsmodels.OpsFloor{Name: "楼层B", FloorNo: "1", BuildingID: bB.ID}
	require.NoError(t, db.Create(flA).Error)
	require.NoError(t, db.Create(flB).Error)
	uA, uB := diffAtkUserA, diffAtkUserB
	for i := 0; i < visibleM; i++ {
		require.NoError(t, db.Create(&models.Workstation{WorkstationName: fmt.Sprintf("A-%d", i), FloorID: &flA.ID, UserID: &uA}).Error)
	}
	for i := 0; i < outOfScope; i++ {
		require.NoError(t, db.Create(&models.Workstation{WorkstationName: fmt.Sprintf("B-%d", i), FloorID: &flB.ID, UserID: &uB}).Error)
	}
	svc := NewWorkstationService(db)

	res, err := svc.BackfillDeptFromUser(write403Ctx(datascope.ScopeResult{All: true}), true, false)
	require.NoError(t, err)
	assert.Equal(t, "All", res.ScopeTier)
	// All 档零回归：全库 5 条有有效用户的工位全部进候选（130 前行为）。
	assert.Equal(t, visibleM+outOfScope, res.Candidates)
	assert.Equal(t, visibleM+outOfScope, res.ToFill, "全部 dept_id 为空 → 全部待填补")
	assert.Equal(t, 0, res.ToCorrect)
	assert.Equal(t, 0, res.SkippedAlreadyValid)
	assert.LessOrEqual(t, len(res.Samples), 20)
	// All 档同样不含 6 个规模字段（D-1：全档位统一删除，无 if isAll 分支回显）。
	body := diffAtkMarshal(t, res)
	for _, key := range diffAtkForbiddenKeys {
		assert.NotContains(t, body, `"`+key+`"`, "All 档响应体亦不得回显 %s", key)
	}
}
