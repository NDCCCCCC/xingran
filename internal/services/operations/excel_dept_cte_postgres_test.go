package operations

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// =========================================================================
// PostgreSQL 专属回归守护 —— 递归 CTE 类型一致性 (SQLSTATE 42804)
// =========================================================================
//
// 事故: 工位「下载部门映射表」端点 500。
//
//   ERROR: recursive query "dept_path_cte" column 3 has type character varying(100)
//          in non-recursive term but type character varying overall (SQLSTATE 42804)
//
// 根因: PostgreSQL 要求递归 CTE 的非递归项与递归项列类型**精确一致**(含长度限定符)。
// base case 的 `dept_name AS path_text` 是 varchar(100);递归项
// `path_text || ' → ' || c.dept_name` 的结果是长度未定的 varchar → 类型不一致 → 整条查询失败。
//
// 为什么能潜伏到生产: SQLite 是动态类型,同样的 SQL 在 SQLite 单测里**能跑通**。
// 单测全绿 ≠ PostgreSQL 可用 —— 这类缺陷必须由 PG 实例或类型断言来守护。
//
// 本文件提供两层守护:
//   1. 静态断言(默认跑,无需数据库): 校验 DeptPathCTE 的 base case 显式 CAST 为 TEXT,
//      且两侧类型表达式一致。
//   2. 真实 PG 集成测试(可选): 设置 TEST_PG_DSN 后自动跑,直连 PG 验证
//      DepartmentQueryBuilder 与部门映射表 SQL 都能成功执行。

// TestDeptPathCTE_PostgresTypeConsistency 静态守护: 递归 CTE 的 base case 必须显式 CAST。
func TestDeptPathCTE_PostgresTypeConsistency(t *testing.T) {
	cte := DeptPathCTE

	// base case 的 path_text 必须是 CAST(dept_name AS TEXT) —— 不能是裸 dept_name。
	// 裸 dept_name 在 PG 里是 varchar(100),与递归项的 varchar(长度未定) 不一致 → 42804。
	assert.Contains(t, cte, "CAST(dept_name AS TEXT) AS path_text",
		"base case 的 path_text 必须显式 CAST 成 TEXT 以满足 PG 递归 CTE 类型一致性约束")
	assert.NotContains(t, cte, "dept_name AS path_text",
		"禁止使用裸 `dept_name AS path_text`(PG 下 varchar(100) 与递归项类型不一致 → SQLSTATE 42804)")

	// depth 列同理: base case 的 0 与递归项的 depth+1 必须都是 integer
	assert.Contains(t, cte, "CAST(0 AS INTEGER) AS depth",
		"base case 的 depth 必须显式 CAST 成 INTEGER 以与递归项 depth+1 类型一致")

	// 递归项的拼接表达式保持不变(链路方向正确性由 TestDepartmentQueryBuilder_* 覆盖)
	assert.Contains(t, cte, "dpc.path_text || ' → ' || c.dept_name",
		"递归项的链路拼接表达式不应被改动")
}

// TestDeptPathCTE_PostgresIntegration 直连真实 PostgreSQL 验证（需设置 TEST_PG_DSN）。
//
// 未设置环境变量时自动 skip —— 本地/CI 无 PG 时不阻塞，但一旦有 PG 就能守住这条线。
// v1.52 P1-9：门控变量与 pg_contract_test.go 统一为 TEST_PG_DSN。
func TestDeptPathCTE_PostgresIntegration(t *testing.T) {
	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		t.Skip("未设置 TEST_PG_DSN，跳过 PostgreSQL 集成验证")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err, "连接 PostgreSQL")

	t.Run("DepartmentQueryBuilder（部门导出）", func(t *testing.T) {
		cfg, ok := GetExportConfig("department")
		require.True(t, ok)

		q := NewDepartmentQueryBuilder().BuildQuery(context.Background(), db, cfg, nil)
		require.NotNil(t, q)

		var rows []map[string]any
		err := q.Find(&rows).Error
		require.NoError(t, err, "部门导出在 PostgreSQL 上不应因递归 CTE 类型不匹配而失败")
	})

	t.Run("部门映射表 SQL", func(t *testing.T) {
		sql := DeptPathCTE + `
SELECT d.dept_name,
       d.dept_code,
       (SELECT path_text
        FROM dept_path_cte
        WHERE id = d.id
        ORDER BY depth DESC
        LIMIT 1) AS parent_path
FROM sys_dept d
WHERE d.deleted_at IS NULL AND d.status = ?
ORDER BY d.dept_code ASC`

		type deptRow struct {
			DeptName   string `gorm:"column:dept_name"`
			DeptCode   string `gorm:"column:dept_code"`
			ParentPath string `gorm:"column:parent_path"`
		}
		var rows []deptRow
		err := db.Raw(sql, 0).Scan(&rows).Error
		require.NoError(t, err, "部门映射表 SQL 在 PostgreSQL 上不应失败")
	})

	t.Run("递归 CTE 两侧类型表达式一致", func(t *testing.T) {
		// 直接用 PG 报错文案做反向断言：改回裸 dept_name 必然报 42804
		broken := strings.Replace(DeptPathCTE,
			"CAST(dept_name AS TEXT) AS path_text", "dept_name AS path_text", 1)
		require.NotEqual(t, DeptPathCTE, broken, "构造反例失败")

		sql := broken + `
SELECT d.dept_name
FROM sys_dept d
WHERE d.deleted_at IS NULL LIMIT 1`
		var rows []map[string]any
		err := db.Raw(sql).Scan(&rows).Error
		require.Error(t, err, "未 CAST 的递归 CTE 在 PostgreSQL 上应当报类型错误")
		assert.Contains(t, err.Error(), "42804",
			"期望 SQLSTATE 42804 (recursive query column type mismatch)，实际: %v", err)
	})
}
