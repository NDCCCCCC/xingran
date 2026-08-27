package operations

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// =====================================================================
// Phase 77-03 Task 2 — ReferenceResolver 尾部测试
//
// 覆盖矩阵:
//   - ResolveSingleWithCondition 条件命中/未命中/无软删表分支
//   - ResolveBatchWithDependencies (21 unc) 沿用 depends_test.go 形态
//   - ResolveBatch 空输入 / 无效引用 / 部分命中
//   - ResolveSingle gorm.ErrRecordNotFound → wrap
//   - makeKey / groupByReference / extractValues / parseReference / batchQueryIDs
//   - ResolveDept / ResolveUser / ResolveAssetDept / ResolveAssetUser
// =====================================================================

// setupResolver77DB 构造最小 resolver 测试 fixture (sqlite 在表).
func setupResolver77DB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, db.Exec(`
		CREATE TABLE sys_dept (
			id TEXT PRIMARY KEY, dept_name TEXT, dept_code TEXT, status INTEGER
		)`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_user (
			id TEXT PRIMARY KEY, username TEXT, nickname TEXT, status INTEGER,
			deleted_at DATETIME
		)`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE ops_floors (
			id TEXT PRIMARY KEY, name TEXT, building_id TEXT,
			deleted_at DATETIME
		)`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE ops_buildings (
			id TEXT PRIMARY KEY, name TEXT, deleted_at DATETIME
		)`).Error)
	return db
}

// TestImp77_ResolveSingleWithCondition 覆盖 ResolveSingleWithCondition
// 条件命中/未命中 + 无效引用三态。
func TestImp77_ResolveSingleWithCondition(t *testing.T) {
	db := setupResolver77DB(t)
	require.NoError(t, db.Exec(`INSERT INTO ops_buildings (id, name) VALUES ('b1', '楼宇A')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO ops_buildings (id, name) VALUES ('b2', '楼宇B')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO ops_floors (id, name, building_id) VALUES ('f1', 'A-1F', 'b1')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO ops_floors (id, name, building_id) VALUES ('f2', 'B-1F', 'b2')`).Error)

	resolver := NewReferenceResolver(db)
	ctx := context.Background()

	t.Run("条件命中", func(t *testing.T) {
		id, err := resolver.ResolveSingleWithCondition(ctx,
			ReferenceRequest{Reference: "ops_floors.name", Value: "A-1F"},
			map[string]string{"building_id": "b1"})
		require.NoError(t, err)
		assert.Equal(t, "f1", id)
	})

	t.Run("条件不命中 (b2 无 A-1F)", func(t *testing.T) {
		_, err := resolver.ResolveSingleWithCondition(ctx,
			ReferenceRequest{Reference: "ops_floors.name", Value: "A-1F"},
			map[string]string{"building_id": "b2"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "引用记录不存在")
		assert.Contains(t, err.Error(), "building_id=b2")
	})

	t.Run("无效引用格式", func(t *testing.T) {
		_, err := resolver.ResolveSingleWithCondition(ctx,
			ReferenceRequest{Reference: "invalid", Value: "x"},
			nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "无效的引用配置")
	})

	t.Run("无软删表 (sys_dept) 不加 deleted_at 过滤", func(t *testing.T) {
		require.NoError(t, db.Exec(`INSERT INTO sys_dept (id, dept_name, dept_code) VALUES ('d1', '技术', 'TECH')`).Error)
		id, err := resolver.ResolveSingleWithCondition(ctx,
			ReferenceRequest{Reference: "sys_dept.dept_code", Value: "TECH"},
			nil)
		require.NoError(t, err)
		assert.Equal(t, "d1", id)
	})

	t.Run("软删过滤", func(t *testing.T) {
		require.NoError(t, db.Exec(`INSERT INTO sys_user (id, username, deleted_at) VALUES ('u1', 'alice', '2026-01-01')`).Error)
		_, err := resolver.ResolveSingleWithCondition(ctx,
			ReferenceRequest{Reference: "sys_user.username", Value: "alice"},
			nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "引用记录不存在")
	})
}

// TestImp77_ResolveBatchWithDependencies_Tail 覆盖 ResolveBatchWithDependencies
// 沿用 depends_test.go 形态。
//
// 注意: ResolveBatchWithDependencies 内部把 resolvedIDs 作为 conditions
// 传给 ResolveSingleWithCondition (格式: column→value, 不是 reference:value→id)。
// 实测依赖引用场景 (resolveDependentReferencesBatch) 也是这种格式:
// `conditions := map[string]string{ s.getTargetFieldForReferenceByName(...): depID }`
func TestImp77_ResolveBatchWithDependencies_Tail(t *testing.T) {
	db := setupResolver77DB(t)
	require.NoError(t, db.Exec(`INSERT INTO ops_buildings (id, name) VALUES ('b1', '楼宇A')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO ops_floors (id, name, building_id) VALUES ('f1', 'A-1F', 'b1')`).Error)

	resolver := NewReferenceResolver(db)
	ctx := context.Background()

	// 已解析的 buildingName → building_id 映射作为 conditions
	resolvedIDs := map[string]string{
		"building_id": "b1",
	}

	refs := []ReferenceRequest{
		{Reference: "ops_floors.name", Value: "A-1F"},
	}

	results, err := resolver.ResolveBatchWithDependencies(ctx, refs, resolvedIDs)
	require.NoError(t, err)
	assert.Equal(t, "f1", results["ops_floors.name:A-1F"])

	// 空 refs
	empty, err := resolver.ResolveBatchWithDependencies(ctx, nil, nil)
	require.NoError(t, err)
	assert.Empty(t, empty)

	// 无效引用
	_, err = resolver.ResolveBatchWithDependencies(ctx,
		[]ReferenceRequest{{Reference: "invalid", Value: "x"}}, nil)
	require.Error(t, err)

	// 条件未命中 (依赖的 building_id 不存在) → ResolveSingleWithCondition 报错并向上传递
	_, err = resolver.ResolveBatchWithDependencies(ctx,
		[]ReferenceRequest{{Reference: "ops_floors.name", Value: "GHOST"}},
		map[string]string{"building_id": "ghost-b"})
	assert.Error(t, err, "ResolveBatchWithDependencies 把首个 ResolveSingleWithCondition 错误传递")
	assert.Contains(t, err.Error(), "引用记录不存在")
}

// TestImp77_ResolveBatch 覆盖 ResolveBatch 边界形态。
func TestImp77_ResolveBatch(t *testing.T) {
	db := setupResolver77DB(t)
	require.NoError(t, db.Exec(`INSERT INTO sys_dept (id, dept_name, dept_code) VALUES ('d1', '技术部', 'TECH')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_dept (id, dept_name, dept_code) VALUES ('d2', '研发', 'RD')`).Error)

	resolver := NewReferenceResolver(db)
	ctx := context.Background()

	t.Run("空 refs", func(t *testing.T) {
		r, err := resolver.ResolveBatch(ctx, nil)
		require.NoError(t, err)
		assert.Empty(t, r)
	})

	t.Run("正常解析", func(t *testing.T) {
		r, err := resolver.ResolveBatch(ctx, []ReferenceRequest{
			{Reference: "sys_dept.dept_code", Value: "TECH"},
			{Reference: "sys_dept.dept_code", Value: "RD"},
		})
		require.NoError(t, err)
		assert.Len(t, r, 2)
		assert.Equal(t, "d1", r["sys_dept.dept_code:TECH"])
		assert.Equal(t, "d2", r["sys_dept.dept_code:RD"])
	})

	t.Run("无效引用格式 → 报错并返回 nil", func(t *testing.T) {
		_, err := resolver.ResolveBatch(ctx, []ReferenceRequest{
			{Reference: "invalid", Value: "x"},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "无效的引用配置")
	})

	t.Run("部分命中 (1 个找不到)", func(t *testing.T) {
		r, err := resolver.ResolveBatch(ctx, []ReferenceRequest{
			{Reference: "sys_dept.dept_code", Value: "TECH"},
			{Reference: "sys_dept.dept_code", Value: "GHOST"},
		})
		require.NoError(t, err)
		assert.Len(t, r, 1, "只命中存在的项")
		assert.Equal(t, "d1", r["sys_dept.dept_code:TECH"])
	})
}

// TestImp77_ResolveSingle 覆盖 ResolveSingle 错误路径。
func TestImp77_ResolveSingle(t *testing.T) {
	db := setupResolver77DB(t)
	require.NoError(t, db.Exec(`INSERT INTO sys_dept (id, dept_name, dept_code) VALUES ('d1', '技术', 'TECH')`).Error)

	resolver := NewReferenceResolver(db)
	ctx := context.Background()

	t.Run("命中", func(t *testing.T) {
		id, err := resolver.ResolveSingle(ctx, ReferenceRequest{
			Reference: "sys_dept.dept_code", Value: "TECH",
		})
		require.NoError(t, err)
		assert.Equal(t, "d1", id)
	})

	t.Run("未命中 wrap gorm.ErrRecordNotFound", func(t *testing.T) {
		_, err := resolver.ResolveSingle(ctx, ReferenceRequest{
			Reference: "sys_dept.dept_code", Value: "GHOST",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "引用记录不存在")
	})

	t.Run("无效引用", func(t *testing.T) {
		_, err := resolver.ResolveSingle(ctx, ReferenceRequest{
			Reference: "invalid", Value: "x",
		})
		require.Error(t, err)
	})
}

// TestImp77_ResolveDept_And_ResolveUser 覆盖 dept/user 解析 + 空值降级。
func TestImp77_ResolveDept_And_ResolveUser(t *testing.T) {
	db := setupResolver77DB(t)
	require.NoError(t, db.Exec(`INSERT INTO sys_dept (id, dept_name, dept_code) VALUES ('d1', '技术', 'TECH')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_user (id, username, nickname) VALUES ('u1', 'alice', 'Alice')`).Error)

	resolver := &referenceResolverImpl{db: db}
	ctx := context.Background()

	// Dept by name
	id, err := resolver.ResolveDept(ctx, "技术")
	require.NoError(t, err)
	assert.Equal(t, "d1", id)

	// Dept by code
	id, err = resolver.ResolveDept(ctx, "TECH")
	require.NoError(t, err)
	assert.Equal(t, "d1", id)

	// Dept empty → no-op
	id, err = resolver.ResolveDept(ctx, "")
	require.NoError(t, err)
	assert.Equal(t, "", id)

	// Dept not found
	_, err = resolver.ResolveDept(ctx, "GHOST")
	require.Error(t, err)

	// User by username
	id, err = resolver.ResolveUser(ctx, "alice")
	require.NoError(t, err)
	assert.Equal(t, "u1", id)

	// User by nickname
	id, err = resolver.ResolveUser(ctx, "Alice")
	require.NoError(t, err)
	assert.Equal(t, "u1", id)

	// Asset 包装
	id, err = resolver.ResolveAssetDept(ctx, "技术")
	require.NoError(t, err)
	assert.Equal(t, "d1", id)

	id, err = resolver.ResolveAssetUser(ctx, "alice")
	require.NoError(t, err)
	assert.Equal(t, "u1", id)
}

// TestImp77_InternalHelpers 覆盖私有 helper 白盒。
func TestImp77_InternalHelpers(t *testing.T) {
	resolver := &referenceResolverImpl{}

	t.Run("makeKey", func(t *testing.T) {
		assert.Equal(t, "table.field:value", resolver.makeKey("table.field", "value"))
	})

	t.Run("parseReference 正确", func(t *testing.T) {
		table, field := resolver.parseReference("sys_dept.dept_code")
		assert.Equal(t, "sys_dept", table)
		assert.Equal(t, "dept_code", field)
	})

	t.Run("parseReference 错误格式", func(t *testing.T) {
		table, field := resolver.parseReference("nodot")
		assert.Equal(t, "", table)
		assert.Equal(t, "", field)
	})

	t.Run("groupByReference", func(t *testing.T) {
		refs := []ReferenceRequest{
			{Reference: "a.b", Value: "1"},
			{Reference: "a.b", Value: "2"},
			{Reference: "c.d", Value: "3"},
		}
		g := resolver.groupByReference(refs)
		assert.Len(t, g, 2)
		assert.Len(t, g["a.b"], 2)
		assert.Len(t, g["c.d"], 1)
	})

	t.Run("extractValues 去重保序", func(t *testing.T) {
		refs := []ReferenceRequest{
			{Reference: "a.b", Value: "1"},
			{Reference: "a.b", Value: "1"},
			{Reference: "a.b", Value: "2"},
		}
		v := resolver.extractValues(refs)
		assert.Equal(t, []string{"1", "2"}, v)
	})
}

// TestImp77_BatchQueryIDs_Helper 覆盖 batchQueryIDs 含 deleted_at 处理。
func TestImp77_BatchQueryIDs_Helper(t *testing.T) {
	db := setupResolver77DB(t)
	require.NoError(t, db.Exec(`INSERT INTO ops_buildings (id, name) VALUES ('b1', '楼A')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO ops_buildings (id, name, deleted_at) VALUES ('b2', '楼B', '2026-01-01')`).Error)

	resolver := &referenceResolverImpl{db: db}
	ctx := context.Background()

	t.Run("正常查询", func(t *testing.T) {
		r, err := resolver.batchQueryIDs(ctx, "ops_buildings", "name", []string{"楼A", "楼B"})
		require.NoError(t, err)
		assert.Equal(t, "b1", r["楼A"])
		_, exists := r["楼B"]
		assert.False(t, exists, "软删记录应被过滤")
	})

	t.Run("空 values", func(t *testing.T) {
		r, err := resolver.batchQueryIDs(ctx, "ops_buildings", "name", nil)
		require.NoError(t, err)
		assert.Empty(t, r)
	})

	t.Run("无软删表", func(t *testing.T) {
		require.NoError(t, db.Exec(`INSERT INTO sys_dept (id, dept_name, dept_code) VALUES ('d1', 'A', 'A')`).Error)
		r, err := resolver.batchQueryIDs(ctx, "sys_dept", "dept_code", []string{"A"})
		require.NoError(t, err)
		assert.Equal(t, "d1", r["A"])
	})
}

// TestImp77_ResolveBatchWithCondition_Helper 覆盖 ResolveBatchWithCondition。
func TestImp77_ResolveBatchWithCondition_Helper(t *testing.T) {
	db := setupResolver77DB(t)
	require.NoError(t, db.Exec(`INSERT INTO ops_floors (id, name, building_id) VALUES ('f1', 'A-1F', 'b1')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO ops_floors (id, name, building_id) VALUES ('f2', 'B-1F', 'b2')`).Error)

	resolver := &referenceResolverImpl{db: db}
	ctx := context.Background()

	t.Run("正常 + 条件", func(t *testing.T) {
		r, err := resolver.ResolveBatchWithCondition(ctx, "ops_floors.name",
			[]string{"A-1F", "B-1F"},
			map[string]string{"building_id": "b1"})
		require.NoError(t, err)
		assert.Equal(t, "f1", r["A-1F"])
		_, exists := r["B-1F"]
		assert.False(t, exists, "条件 scope 不命中")
	})

	t.Run("空 values", func(t *testing.T) {
		r, err := resolver.ResolveBatchWithCondition(ctx, "ops_floors.name", nil, nil)
		require.NoError(t, err)
		assert.Empty(t, r)
	})

	t.Run("无效引用", func(t *testing.T) {
		_, err := resolver.ResolveBatchWithCondition(ctx, "invalid", []string{"x"}, nil)
		require.Error(t, err)
	})
}
