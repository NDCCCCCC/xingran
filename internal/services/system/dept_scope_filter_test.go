package system

// dept_scope_filter_test.go —— 部门树 Scoped 收口的回归（v1.54）。
//
// 背景：部门树接口（/system/departments/tree）此前从不消费 ScopeResult，
// 机构管理员在「部门管理/系统用户」页面能看到全库 1060 个部门。
// 修复：DepartmentListParams 加 Scoped bool（显式 opt-in），GetTreeWithFilter
// 末尾按登录人数据范围/机构锚点收口。
//
// 语义对齐前端 DeptTree/visibleDeptFilter.ts（Phase 133 / DS-06 预留基建）：
//   - 节点自身命中 → 整棵子树保留；
//   - 未命中但有可见后代 → 作为祖先路径保留；
//   - 全未命中 → 丢弃；
//   - 未注入 / All → 原样（零回归）；
//   - FailClosed / Self → 空树 fail-closed。
//
// 另锁死 tree-select 场景的零回归承诺：不传 scoped 时树保持全量。

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
	"github.com/xingran-next/xingran-go-backend/pkg/datascope"
)

// 组织树（模拟湖北分公司结构）：
//
//	D-hb  湖北分公司(1) → D-wh 武汉中心支公司(2) → D-zh 综合管理部 / D-it IT部
//	                              → D-qk 硚口支公司(3)
//	                  → D-cs 长沙中心支公司(2) → D-cs1 长沙支公司一部
func newDeptScopeTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "dept_scope.db")), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		if sqlDB, e := db.DB(); e == nil {
			_ = sqlDB.Close()
		}
	})

	require.NoError(t, db.Exec(`CREATE TABLE sys_dept (
		id TEXT PRIMARY KEY, dept_name TEXT, dept_code TEXT, parent_id TEXT,
		ancestors TEXT DEFAULT '', org_level INTEGER DEFAULT 0,
		is_external_org INTEGER DEFAULT 0, status INTEGER DEFAULT 0,
		order_num INTEGER DEFAULT 0, deleted_at DATETIME)`).Error)

	require.NoError(t, db.Exec(`INSERT INTO sys_dept
		(id, dept_name, dept_code, parent_id, ancestors, org_level, status) VALUES
		('D-hb',  '湖北分公司',     'HB',  NULL,    '',               1, 0),
		('D-wh',  '武汉中心支公司', 'WH',  'D-hb',  'D-hb',           2, 0),
		('D-zh',  '综合管理部',     'ZH',  'D-wh',  'D-hb,D-wh',      0, 0),
		('D-it',  'IT部',          'IT',  'D-wh',  'D-hb,D-wh',      0, 0),
		('D-qk',  '硚口支公司',     'QK',  'D-wh',  'D-hb,D-wh',      3, 0),
		('D-cs',  '长沙中心支公司', 'CS',  'D-hb',  'D-hb',           2, 0),
		('D-cs1', '长沙支公司一部', 'CS1', 'D-cs',  'D-hb,D-cs',      3, 0)`).Error)

	return db
}

// anchorCtx 构造"刘洋场景"的 ctx：锚点命中武汉中心支公司（可见集=武汉子树全部）。
func anchorCtx() context.Context {
	return datascope.WithContext(context.Background(), datascope.ScopeResult{
		DeptIDs: []string{"D-wh", "D-zh", "D-it", "D-qk"},
		UserID:  "U-liu",
	})
}

func buildTree(db *gorm.DB, scoped bool, ctx context.Context) []*models.Department {
	svc := NewDepartmentService(db)
	params := requests.DepartmentListParams{Scoped: scoped}
	tree, err := svc.(*departmentService).GetTreeWithFilter(ctx, true, params)
	if err != nil {
		panic(err)
	}
	return tree
}

// collectIDs 递归收集树中所有节点 ID（用于全集断言）。
func collectIDs(nodes []*models.Department) map[string]struct{} {
	m := make(map[string]struct{})
	var walk func(list []*models.Department)
	walk = func(list []*models.Department) {
		for _, n := range list {
			m[n.ID] = struct{}{}
			walk(n.Children)
		}
	}
	walk(nodes)
	return m
}

// 用例 1：Scoped + 锚点（武汉中支子树）→ 只保留武汉子树 + 祖先路径（湖北分公司），
// 长沙中支及其下级全部不可见。
func TestDeptTree_Scoped_AnchorSubBang(t *testing.T) {
	db := newDeptScopeTestDB(t)
	tree := buildTree(db, true, anchorCtx())

	ids := collectIDs(tree)
	assert.Contains(t, ids, "D-hb", "祖先路径必须保留（树不断裂）")
	assert.Contains(t, ids, "D-wh")
	assert.Contains(t, ids, "D-zh")
	assert.Contains(t, ids, "D-it")
	assert.Contains(t, ids, "D-qk", "中支子树必须含下级支公司")
	assert.NotContains(t, ids, "D-cs", "兄弟中支不可见")
	assert.NotContains(t, ids, "D-cs1")
}

// 用例 2：Scoped + All（超管）→ 原样全树（零回归）。
func TestDeptTree_Scoped_All_Unchanged(t *testing.T) {
	db := newDeptScopeTestDB(t)
	ctx := datascope.WithContext(context.Background(), datascope.AllScope("U-admin"))

	full := collectIDs(buildTree(db, false, ctx))
	scoped := collectIDs(buildTree(db, true, ctx))
	assert.Equal(t, full, scoped, "All 用户 Scoped 收口必须与全量一致（C1 零回归）")
	assert.Len(t, scoped, 7)
}

// 用例 3：不传 scoped（tree-select 场景）→ 即使范围是 Self 也保持全量（零回归）。
//
// 这是配置场景的硬约束：角色「自定义数据范围」勾选树、岗位锚点部门选择
// 必须看到全树，否则管理员无法配置。
func TestDeptTree_NoScoped_StayFull(t *testing.T) {
	db := newDeptScopeTestDB(t)
	ctx := datascope.WithContext(context.Background(), datascope.ScopeResult{
		ResourceHook: datascope.HookSelfUser,
		UserID:       "U-liu",
	})

	ids := collectIDs(buildTree(db, false, ctx))
	assert.Len(t, ids, 7, "不传 scoped 时树保持全量（tree-select 场景零回归）")
}

// 用例 4：Scoped + Self（仅本人无部门集）→ 空树 fail-closed。
func TestDeptTree_Scoped_Self_FailClosed(t *testing.T) {
	db := newDeptScopeTestDB(t)
	ctx := datascope.WithContext(context.Background(), datascope.ScopeResult{
		ResourceHook: datascope.HookSelfUser,
		UserID:       "U-liu",
	})

	tree := buildTree(db, true, ctx)
	assert.Empty(t, tree, "Self 档部门树 fail-closed（部门无用户归属列，按裁定空集）")
}

// 用例 5：Scoped + FailClosed（异常档位）→ 空树。
func TestDeptTree_Scoped_FailClosed(t *testing.T) {
	db := newDeptScopeTestDB(t)
	ctx := datascope.WithContext(context.Background(), datascope.FailClosedScope("U-x"))

	tree := buildTree(db, true, ctx)
	assert.Empty(t, tree)
}

// 用例 6：Scoped + Custom（勾选清单不展开子树）→ **精确清单**，不放大到子部门。
//
// 审计 P0 修正：之前的裁决（"自身命中保留整棵子树，与前端 Phase 133 同语义"）
// 会让 DataScopeDept(3)/Custom(2) 档用户看到下级部门树——数据范围放大。
// DeptIDs 已是最终可见集：DeptChild/锚点档已展开完整子树（精确匹配天然覆盖），
// Dept/Custom 档本就只该看精确清单。故统一为"集合内节点 + 祖先路径"精确裁剪。
func TestDeptTree_Scoped_CustomKeepsOwnSubtree(t *testing.T) {
	db := newDeptScopeTestDB(t)
	// 只勾了武汉中心支公司本身（未展开子树——Custom 档不做展开）
	ctx := datascope.WithContext(context.Background(), datascope.ScopeResult{
		DeptIDs: []string{"D-wh"},
		UserID:  "U-x",
	})

	ids := collectIDs(buildTree(db, true, ctx))
	assert.Contains(t, ids, "D-hb", "祖先路径保留")
	assert.Contains(t, ids, "D-wh")
	// 审计 P0：精确清单不包含子部门 → 子部门节点也不应出现在树里
	assert.NotContains(t, ids, "D-zh", "Custom 精确匹配：树也不得放大到子部门")
	assert.NotContains(t, ids, "D-qk")
	assert.NotContains(t, ids, "D-cs", "清单外的兄弟分支不可见")
	assert.NotContains(t, ids, "D-cs1")
}

// 用例 6b（审计 P0 回归）：锚点档 DeptIDs 已展开完整子树 → 精确匹配仍含全部后代。
// 证明"精确裁剪"不会破坏机构管理员看全中支（用例 A 语义）。
func TestDeptTree_Scoped_AnchorExpandedSubtreeStillVisible(t *testing.T) {
	db := newDeptScopeTestDB(t)
	// 模拟锚点=武汉中支，Resolve 已展开为完整子树清单（含自身+所有后代）
	ctx := datascope.WithContext(context.Background(), datascope.ScopeResult{
		DeptIDs: []string{"D-wh", "D-zh", "D-qk"},
		UserID:  "U-liu",
	})

	ids := collectIDs(buildTree(db, true, ctx))
	assert.Contains(t, ids, "D-hb", "祖先路径保留")
	assert.Contains(t, ids, "D-wh")
	assert.Contains(t, ids, "D-zh", "子树清单内的节点保留")
	assert.Contains(t, ids, "D-qk", "子树清单内的节点保留")
	assert.NotContains(t, ids, "D-cs", "兄弟中支不可见")
}

// 用例 7：祖先路径节点是**拷贝**而非原节点（防御性：未来接真缓存不污染）。
func TestDeptTree_Scoped_PathNodeIsCopy(t *testing.T) {
	db := newDeptScopeTestDB(t)
	ctx := datascope.WithContext(context.Background(), datascope.ScopeResult{
		DeptIDs: []string{"D-zh"}, // 只可见综合管理部
		UserID:  "U-x",
	})

	tree := buildTree(db, true, ctx)
	require.Len(t, tree, 1)
	root := tree[0]
	require.Equal(t, "D-hb", root.ID)
	require.Len(t, root.Children, 1)
	mid := root.Children[0]
	require.Equal(t, "D-wh", mid.ID)

	// 拷贝断言：祖先路径节点的 Children 必须是过滤后的新切片，
	// 且与"全量树中同 ID 节点的 Children"不是同一引用。
	full := buildTree(db, false, context.Background())
	var findNode func(nodes []*models.Department, id string) *models.Department
	findNode = func(nodes []*models.Department, id string) *models.Department {
		for _, n := range nodes {
			if n.ID == id {
				return n
			}
			if c := findNode(n.Children, id); c != nil {
				return c
			}
		}
		return nil
	}
	fullMid := findNode(full, "D-wh")
	require.NotNil(t, fullMid)
	assert.NotSame(t, fullMid, mid, "祖先路径节点必须是值拷贝，不得复用原树节点")
}
