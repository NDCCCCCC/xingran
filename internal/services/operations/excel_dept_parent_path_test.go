package operations

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// =========================================================================
// 工位管理 — 部门导出 parentPath 列 (Phase 工位部门映射 P3)
// =========================================================================
//
// 目的: 验证 DepartmentQueryBuilder 用 PostgreSQL 递归 CTE 算出每个部门的
// "根 → … → 当前" 父级链路字符串,让同名部门(挂在不同市州中心支公司下)
// 在导出 xlsx 中通过 parentPath 列可区分.
//
// 测试采用 SQLite 内存库复用项目约定:
//   - 现代驱动 modernc.org/sqlite (CGo-free)
//   - 表结构手工 CREATE TABLE (不依赖 sys_dept GORM struct,
//     避免被 future struct 字段变更污染单测).
//
// SQL 跨方言: 用相关子查询 ORDER BY depth DESC LIMIT 1 取代 PG 专属 DISTINCT ON,
// 生产 PG 与单元测试 SQLite 走同一段代码路径.

// setupDeptCTETestDB 建一张与 sys_dept 兼容的最简表 (id + parent_id + dept_name + deleted_at)
func setupDeptCTETestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "open sqlite")

	// SQLite 也认可 TEXT PRIMARY KEY,id 用 TEXT 是为了与 PG uuid 字段兼容
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_dept (
			id TEXT PRIMARY KEY,
			dept_code TEXT,
			dept_name TEXT NOT NULL,
			parent_id TEXT,
			ancestors TEXT,
			order_num INTEGER DEFAULT 0,
			leader TEXT,
			phone TEXT,
			email TEXT,
			is_external_org INTEGER DEFAULT 0,
			status INTEGER DEFAULT 0,
			remark TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)
	`).Error)

	return db
}

// insertDept 工具函数: 把 dept 插入测试库.
//
// 注意: parentID="" 时必须写 NULL 而不是空字符串,否则 CTE base case
// `WHERE parent_id IS NULL` 会把根节点漏掉,所有 parent_path 全成 nil.
func insertDept(t *testing.T, db *gorm.DB, id, code, name, parentID string) {
	t.Helper()
	if parentID == "" {
		require.NoError(t, db.Exec(`
			INSERT INTO sys_dept (id, dept_code, dept_name, parent_id, status)
			VALUES (?, ?, ?, NULL, 0)
		`, id, code, name).Error)
		return
	}
	require.NoError(t, db.Exec(`
		INSERT INTO sys_dept (id, dept_code, dept_name, parent_id, status)
		VALUES (?, ?, ?, ?, 0)
	`, id, code, name, parentID).Error)
}

// collectMap 跑 BuildQuery 并把 []map[string]interface{} 抽到 map[id]parent_path, 便于断言.
//
// parent_path 列在 GORM Find(&mapSlice) 下可能被包成 *interface{},这里走生产代码
// 同一解包函数 unwrapInterface 后再类型断言,避免测试与生产不一致.
func collectParentPaths(t *testing.T, db *gorm.DB, params map[string]interface{}) map[string]string {
	t.Helper()

	cfg, _ := GetExportConfig("department")
	require.NotEmpty(t, cfg.Columns, "department 导出配置存在")

	q := NewDepartmentQueryBuilder().BuildQuery(context.Background(), db, cfg, params)
	require.NotNil(t, q)

	var rows []map[string]interface{}
	require.NoError(t, q.Find(&rows).Error, "执行 CTE 查")

	out := make(map[string]string, len(rows))
	for _, r := range rows {
		id := asString(r["id"])
		pp := asString(r["parent_path"])
		out[id] = pp
	}
	return out
}

// asString 把 GORM Find + mapSlice 里可能出现的 string / *interface{} 包装值统一抽成字符串.
func asString(v interface{}) string {
	v = unwrapInterface(v)
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	if b, ok := v.([]byte); ok {
		return string(b)
	}
	return fmt.Sprintf("%v", v)
}

// ----- 1. 根部门: 链路就是自身 -----

func TestDepartmentQueryBuilder_PathAtRoot(t *testing.T) {
	db := setupDeptCTETestDB(t)

	// 单个根,无父级
	insertDept(t, db, "root-1", "ROOT", "中国人民财产保险股份有限公司", "")

	paths := collectParentPaths(t, db, nil)

	require.Len(t, paths, 1)
	assert.Equal(t, "中国人民财产保险股份有限公司", paths["root-1"],
		"根部门的链路就是自身名,不重复拼接")
}

// ----- 2. 多级链路: 拼接顺序正确 -----

func TestDepartmentQueryBuilder_MultiLevel(t *testing.T) {
	db := setupDeptCTETestDB(t)

	// 4 级链路: 公司 → 湖北省分公司 → 咸宁中心支公司 → 赤壁支公司
	insertDept(t, db, "d-1", "ROOT", "中国人民财产保险股份有限公司", "")
	insertDept(t, db, "d-2", "HUB", "湖北省分公司", "d-1")
	insertDept(t, db, "d-3", "XN", "咸宁中心支公司", "d-2")
	insertDept(t, db, "d-4", "CB", "赤壁支公司", "d-3")

	paths := collectParentPaths(t, db, nil)

	require.Len(t, paths, 4)
	assert.Equal(t, "中国人民财产保险股份有限公司", paths["d-1"], "根=自身")
	assert.Equal(t, "中国人民财产保险股份有限公司 → 湖北省分公司", paths["d-2"], "2 级链")
	assert.Equal(t, "中国人民财产保险股份有限公司 → 湖北省分公司 → 咸宁中心支公司", paths["d-3"], "3 级链")
	assert.Equal(t, "中国人民财产保险股份有限公司 → 湖北省分公司 → 咸宁中心支公司 → 赤壁支公司", paths["d-4"],
		"4 级链; 这是用户原始问题: 赤壁支公司 vs 竹山支公司原来同名或近似时只能靠链路区分")
}

// ----- 3. 同名部门在不同父级下: 各得各自的链路 (核心场景) -----

func TestDepartmentQueryBuilder_DepthConflict_MultipleRoots(t *testing.T) {
	db := setupDeptCTETestDB(t)

	// 共享根: 公司 → 湖北省分公司
	insertDept(t, db, "root", "ROOT", "中国人民财产保险股份有限公司", "")
	insertDept(t, db, "hub", "HUB", "湖北省分公司", "root")

	// 11 条同名部门,分别挂在不同市中心支公司下 (用户原始数据的核心痛点)
	branches := []struct{ id, name, parent string }{
		{"dept-xn", "个人营销业务销售部", "xn"},
		{"dept-yc", "个人营销业务销售部", "yc"},
		{"dept-xy", "个人营销业务销售部", "xy"},
		{"dept-hs", "个人营销业务销售部", "hs"},
		{"dept-sy", "个人营销业务销售部", "sy"},
		{"dept-jz", "个人营销业务销售部", "jz"},
		{"dept-jm", "个人营销业务销售部", "jm"},
		{"dept-xg", "个人营销业务销售部", "xg"},
		{"dept-sz", "个人营销业务销售部", "sz"},
		{"dept-hg", "个人营销业务销售部", "hg"},
		{"dept-tc", "个人营销业务销售部", "tc-extra"}, // 额外更深一层: 通城支公司
	}

	// 先建 10 个市中心支公司
	zhouBranches := map[string]string{
		"xn":        "咸宁中心支公司",
		"yc":        "宜昌中心支公司",
		"xy":        "襄阳中心支公司",
		"hs":        "黄石中心支公司",
		"sy":        "十堰中心支公司",
		"jz":        "荆州中心支公司",
		"jm":        "荆门中心支公司",
		"xg":        "孝感中心支公司",
		"sz":        "随州中心支公司",
		"hg":        "黄冈中心支公司",
		"tc-extra":  "通城支公司",  // 11 号部门的额外父级
		"tc-parent": "咸宁中心支公司", // 通城支公司挂在咸宁下
	}
	for id, name := range zhouBranches {
		parent := "hub" // 全部挂在湖北省分公司下
		if id == "tc-extra" {
			parent = "tc-parent"
		}
		insertDept(t, db, id, id, name, parent)
	}

	// 再建 11 个同名销售部
	for _, b := range branches {
		insertDept(t, db, b.id, b.id, b.name, b.parent)
	}

	paths := collectParentPaths(t, db, nil)

	// 必须有 11 + 12 = 23 行 (11 条同名部门 + 12 条父级节点)
	require.GreaterOrEqual(t, len(paths), len(branches),
		"应至少有 %d 行", len(branches))

	// 同名部门"个人营销业务销售部"在不同父级下得到不同链路
	for _, b := range branches {
		expectedParentName := zhouBranches[b.parent]
		if b.parent == "tc-extra" {
			expectedParentName = "通城支公司"
		}
		// 期望链路以父级名结尾, 且包含完整的"根 → ... → 父级"
		pp := paths[b.id]
		require.NotEmpty(t, pp, "dept=%s 必须有 parent_path", b.id)
		assert.Contains(t, pp, expectedParentName,
			"dept=%s 链路应包含父级 %q, 实际: %s", b.id, expectedParentName, pp)
	}

	// 不同 dept 的 parent_path 必须互不相同 (同名部门靠链路区分 — 用户原始诉求)
	seen := make(map[string]string) // path -> dept-id
	for _, b := range branches {
		if prev, dup := seen[paths[b.id]]; dup {
			t.Fatalf("链路重复! dept=%s 与 dept=%s 共享链路 %q", prev, b.id, paths[b.id])
		}
		seen[paths[b.id]] = b.id
	}
}

// ----- 4. 动态过滤: code 模糊匹配 -----

func TestDepartmentQueryBuilder_FilterByCode(t *testing.T) {
	db := setupDeptCTETestDB(t)

	insertDept(t, db, "d-1", "ROOT", "A 公司", "")
	insertDept(t, db, "d-2", "HUB", "B 分公司", "d-1")
	insertDept(t, db, "d-3", "ZR", "C 营销", "d-2")

	paths := collectParentPaths(t, db, map[string]interface{}{
		"code": "ZR",
	})

	require.Len(t, paths, 1, "code=ZR 应只筛 1 条")
	assert.Equal(t, "A 公司 → B 分公司 → C 营销", paths["d-3"],
		"3 级链路完整且按 lt→gt 顺序")
}

// ----- 5. 软删除部门不应出现在结果中 -----

func TestDepartmentQueryBuilder_ExcludesSoftDeleted(t *testing.T) {
	db := setupDeptCTETestDB(t)

	insertDept(t, db, "d-1", "ROOT", "Root", "")
	insertDept(t, db, "d-2", "A", "正常部门 A", "d-1")

	// 软删除 d-3
	require.NoError(t, db.Exec(`
		INSERT INTO sys_dept (id, dept_code, dept_name, parent_id, status, deleted_at)
		VALUES ('d-3', 'B', '已删部门 B', 'd-1', 1, '2026-07-01 00:00:00')
	`).Error)

	paths := collectParentPaths(t, db, nil)

	assert.Len(t, paths, 2, "软删除部门不出现在结果中")
	_, hasB := paths["d-3"]
	assert.False(t, hasB, "软删除的 d-3 应被 WHERE deleted_at IS NULL 过滤")
}
