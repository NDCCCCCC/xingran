package addomain

// DEBT-01（v1.51）第六批：部门↔OU 对账查询服务（dept_ou_mismatch_service.go）。
//
// 覆盖目标：
//  1. validMismatchType 四值白名单
//  2. ListDeptOUMismatch 全路径：空结果序列化为 []（非 null）、mismatchType/deptId
//     过滤、keyword 大小写不敏感 + LIKE 元字符转义（ESCAPE '\\'）、负 OFFSET 防御、
//     确定性排序
//
// v_dept_ou_mismatch 是 PG 视图（D-11 UNION ALL 四分支），SQLite 测试库手写
// 等价简化视图（列契约对齐 D-11 十一列），验证的是查询服务层的过滤/转义/分页
// 语义，不重复验证视图本身的 SQL。

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupMismatchTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	// 简化视图：两分支（user_ou_mismatch / dept_soft_deleted），列契约对齐 D-11
	require.NoError(t, db.Exec(`CREATE VIEW v_dept_ou_mismatch AS
		SELECT 'd-1' AS dept_id, '财务部' AS dept_name, NULL AS dept_deleted_at,
		       'cfg-1' AS ad_config_id, '主配置' AS config_name,
		       'OU=wrong,DC=t' AS mapped_ou_dn,
		       'u-1' AS user_id, 'Alice' AS username,
		       'OU=other,DC=t' AS user_ad_ou_dn,
		       'user_ou_mismatch' AS mismatch_type, '用户 OU 与部门映射不一致' AS mismatch_detail
		UNION ALL
		SELECT 'd-2', '被删部', '2026-01-01', 'cfg-1', '主配置',
		       'OU=gone,DC=t', NULL, NULL, NULL,
		       'dept_soft_deleted', '部门已软删但映射仍启用'`).Error)
	return db
}

func TestValidMismatchType(t *testing.T) {
	for _, v := range MismatchTypes {
		if !validMismatchType(v) {
			t.Errorf("白名单值 %q 应合法", v)
		}
	}
	for _, bad := range []string{"", "bogus", "user_ou_mismatch "} {
		if validMismatchType(bad) {
			t.Errorf("非法值 %q 应被拒绝", bad)
		}
	}
}

func TestListDeptOUMismatch(t *testing.T) {
	db := setupMismatchTestDB(t)
	ctx := context.Background()
	svc := NewDeptOUMismatchService(db)

	// 空结果 → 空切片（非 nil），total=0
	rows, total, err := svc.ListDeptOUMismatch(ctx, &DeptOUMismatchQuery{Current: 1, PageSize: 10, DeptID: "d-none"})
	require.NoError(t, err)
	if rows == nil || len(rows) != 0 || total != 0 {
		t.Fatalf("空结果应返回空切片, got rows=%v total=%d", rows, total)
	}

	// 全量 + 字段列契约（显式 column 标签 → D-11 列名不丢值）
	rows, total, err = svc.ListDeptOUMismatch(ctx, &DeptOUMismatchQuery{Current: 1, PageSize: 10})
	require.NoError(t, err)
	if total != 2 || len(rows) != 2 {
		t.Fatalf("应返回 2 条, got total=%d rows=%d", total, len(rows))
	}
	// 确定性排序：mismatch_type 升序 → dept_soft_deleted 在前
	if rows[0].MismatchType != "dept_soft_deleted" || rows[1].MismatchType != "user_ou_mismatch" {
		t.Fatalf("排序不符: %s, %s", rows[0].MismatchType, rows[1].MismatchType)
	}
	if rows[1].Username == nil || *rows[1].Username != "Alice" {
		t.Fatalf("user 分支 username 应命中（列映射不丢值）, got %v", rows[1].Username)
	}
	if rows[0].Username != nil {
		t.Fatalf("dept 分支 username 应为 NULL, got %v", rows[0].Username)
	}

	// mismatchType 过滤
	rows, total, err = svc.ListDeptOUMismatch(ctx, &DeptOUMismatchQuery{Current: 1, PageSize: 10, MismatchType: "user_ou_mismatch"})
	require.NoError(t, err)
	if total != 1 || len(rows) != 1 || rows[0].MismatchType != "user_ou_mismatch" {
		t.Fatalf("类型过滤不符: total=%d", total)
	}

	// deptId 过滤
	rows, total, err = svc.ListDeptOUMismatch(ctx, &DeptOUMismatchQuery{Current: 1, PageSize: 10, DeptID: "d-2"})
	require.NoError(t, err)
	if total != 1 || rows[0].DeptName == nil || *rows[0].DeptName != "被删部" {
		t.Fatalf("deptId 过滤不符: total=%d", total)
	}

	// keyword：大小写不敏感 + 命中 username 列
	rows, total, err = svc.ListDeptOUMismatch(ctx, &DeptOUMismatchQuery{Current: 1, PageSize: 10, Keyword: "alic"})
	require.NoError(t, err)
	if total != 1 {
		t.Fatalf("keyword 应大小写不敏感命中 Alice, got %d", total)
	}
	// keyword 命中 mapped_ou_dn 列
	rows, total, err = svc.ListDeptOUMismatch(ctx, &DeptOUMismatchQuery{Current: 1, PageSize: 10, Keyword: "WRONG"})
	require.NoError(t, err)
	if total != 1 {
		t.Fatalf("keyword 应命中 mapped_ou_dn, got %d", total)
	}
	// keyword 为 LIKE 元字符 → 字面量匹配（不退化为通配）
	rows, total, err = svc.ListDeptOUMismatch(ctx, &DeptOUMismatchQuery{Current: 1, PageSize: 10, Keyword: "%"})
	require.NoError(t, err)
	if total != 0 {
		t.Fatalf("LIKE 元字符应被转义为字面量, got %d", total)
	}
	// keyword 空白 → 不过滤
	_, total, err = svc.ListDeptOUMismatch(ctx, &DeptOUMismatchQuery{Current: 1, PageSize: 10, Keyword: "  "})
	require.NoError(t, err)
	if total != 2 {
		t.Fatalf("空白 keyword 应不过滤, got %d", total)
	}

	// 分页
	rows, total, err = svc.ListDeptOUMismatch(ctx, &DeptOUMismatchQuery{Current: 2, PageSize: 1})
	require.NoError(t, err)
	if total != 2 || len(rows) != 1 {
		t.Fatalf("分页不符: total=%d rows=%d", total, len(rows))
	}
	// 负 OFFSET 防御（Current=0 → offset 归 0，不报错）
	rows, total, err = svc.ListDeptOUMismatch(ctx, &DeptOUMismatchQuery{Current: 0, PageSize: 1})
	require.NoError(t, err)
	if total != 2 || len(rows) != 1 {
		t.Fatalf("负 OFFSET 应被防御归 0, got total=%d rows=%d", total, len(rows))
	}
}
