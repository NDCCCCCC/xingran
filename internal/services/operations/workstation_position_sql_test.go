package operations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =====================================================================
// BatchUpdatePositions 的 SQL 构造回归测试
//
// 背景（workstation-positions-500 会话）：
// `CASE id WHEN ? THEN ? END` 的 THEN 分支在 PostgreSQL Parse 阶段无法推断类型
// （CASE 结果类型 ≠ 赋值目标列类型），PG 把整个 CASE 退化为 text，赋值时报
//
//	ERROR: column "position_x" is of type integer but expression is of type text
//	(SQLSTATE 42804)
//
// SQLite 无严格类型推断，无法复现该缺陷，因此这里对 SQL 形状做纯函数断言，
// 把「THEN 分支必须带显式类型转换」锁成可回归的契约。
// 真实 PostgreSQL 端到端复现见 workstation_position_pg_test.go。
// =====================================================================

// TestBuildBatchPositionUpdateSQL_ThenBranchMustCast 锁定 THEN 分支的显式类型转换。
func TestBuildBatchPositionUpdateSQL_ThenBranchMustCast(t *testing.T) {
	items := []PositionUpdateItem{
		{ID: "id-1", PositionX: 100, PositionY: 200},
		{ID: "id-2", PositionX: 300, PositionY: 400},
	}

	sql, _ := buildBatchPositionUpdateSQL(items)

	// 必填两列：THEN 分支必须是 CAST(? AS int)
	assert.Contains(t, sql, "position_x = CASE id WHEN ? THEN CAST(? AS int)",
		"position_x 的 THEN 分支必须显式 CAST 成 int，否则 PostgreSQL 报 42804")
	assert.Contains(t, sql, "position_y = CASE id WHEN ? THEN CAST(? AS int)",
		"position_y 的 THEN 分支必须显式 CAST 成 int，否则 PostgreSQL 报 42804")

	// 禁止出现裸 THEN ?（未类型化占位符）
	assert.NotContains(t, sql, "THEN ? ",
		"SQL 中不得出现未类型化的 THEN ? 占位符（PostgreSQL 会把 CASE 退化为 text）")

	// 每个 CASE 分支的 THEN 都带 CAST
	whenCount := strings.Count(sql, "WHEN ? THEN CAST(? AS int)")
	assert.Equal(t, 4, whenCount, "2 个 item × (position_x + position_y) = 4 个 THEN 分支")
}

// TestBuildBatchPositionUpdateSQL_OptionalClauseMustCast 锁定可选列的 THEN 分支同样带 CAST。
func TestBuildBatchPositionUpdateSQL_OptionalClauseMustCast(t *testing.T) {
	rot, w, d, dt := 45, 1800, 800, 1
	items := []PositionUpdateItem{
		{ID: "id-1", PositionX: 1, PositionY: 2,
			Rotation: &rot, Width: &w, Depth: &d, DeskType: &dt},
	}

	sql, _ := buildBatchPositionUpdateSQL(items)

	for _, col := range []string{"rotation", "width", "depth", "desk_type"} {
		assert.Contains(t, sql, col+" = CASE id WHEN ? THEN CAST(? AS int)",
			"可选列 %s 的 THEN 分支必须显式 CAST 成 int", col)
	}
	assert.NotContains(t, sql, "THEN ? ", "SQL 中不得出现未类型化的 THEN ? 占位符")
}

// TestBuildBatchPositionUpdateSQL_ArgOrder 锁定参数顺序与 SQL 中 ? 的出现顺序一一对应。
func TestBuildBatchPositionUpdateSQL_ArgOrder(t *testing.T) {
	rot := 45
	items := []PositionUpdateItem{
		{ID: "id-1", PositionX: 100, PositionY: 200},
		{ID: "id-2", PositionX: 300, PositionY: 400, Rotation: &rot},
	}

	sql, args := buildBatchPositionUpdateSQL(items)

	// args 顺序：全部 (id, X) → 全部 (id, Y) → rotation 的 (id, v) → ids
	require.Len(t, args, 11, "2×2(position_x) + 2×2(position_y) + 1×2(rotation) + 1(ids) = 11")

	// position_x 段
	assert.Equal(t, "id-1", args[0])
	assert.Equal(t, 100, args[1])
	assert.Equal(t, "id-2", args[2])
	assert.Equal(t, 300, args[3])

	// position_y 段
	assert.Equal(t, "id-1", args[4])
	assert.Equal(t, 200, args[5])
	assert.Equal(t, "id-2", args[6])
	assert.Equal(t, 400, args[7])

	// rotation 段（仅第 2 个 item 带 rotation）
	assert.Equal(t, "id-2", args[8])
	assert.Equal(t, 45, args[9])

	// 末尾是 ids（[]string），供 GORM 展开成 IN (?,?,?)
	ids, ok := args[len(args)-1].([]string)
	require.True(t, ok, "最后一个参数应是 []string（ids），供 GORM 展开 IN (?)")
	assert.Equal(t, []string{"id-1", "id-2"}, ids)

	// SQL 中 ? 的个数（不含 IN ? 展开后的 2 个）应与固定参数个数一致
	assert.Equal(t, 1, strings.Count(sql, "WHERE id IN ?"), "WHERE 子句应保持单个 IN ? 占位符")
}

// TestBuildBatchPositionUpdateSQL_OptionalOmitted 全部 optional 为 nil 时不追加对应子句。
func TestBuildBatchPositionUpdateSQL_OptionalOmitted(t *testing.T) {
	items := []PositionUpdateItem{{ID: "id-1", PositionX: 1, PositionY: 2}}

	sql, args := buildBatchPositionUpdateSQL(items)

	for _, col := range []string{"rotation", "width", "depth", "desk_type"} {
		assert.NotContains(t, sql, col+" = CASE", "optional 全为 nil 时不应生成 %s 子句", col)
	}
	// 2×(id,X) + 2×(id,Y) + 1(ids)
	assert.Len(t, args, 5)
}
