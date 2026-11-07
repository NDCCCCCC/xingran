package operations

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// 「按人员反推部门」批量回填回归测试（quick-261106-wsb）。
//
// 与导入兜底 inferDeptFromUser 判定规则一致，但作用于已落库的存量工位：
//   - 默认仅填补 dept_id 空缺，不覆盖已填部门；
//   - fixMismatch=true 才纠正不一致（默认关闭）；
//   - 用户部门为 NULL / 部门名称 / 悬空 ID 一律跳过；
//   - 绝不改 status；
//   - dryRun 不写库；执行幂等。

const (
	bdDeptA = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" // 部门 A（有效）
	bdDeptB = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb" // 部门 B（有效）
	bdUserA = "11111111-1111-1111-1111-111111111111" // 用户张三 → 部门 A
	bdUserB = "22222222-2222-2222-2222-222222222222" // 用户李四 → 部门 B
	bdUserC = "33333333-3333-3333-3333-333333333333" // 用户王五 → dept_id NULL
	bdUserD = "44444444-4444-4444-4444-444444444444" // 用户赵六 → dept_id 是部门名称（脏）
	bdUserE = "55555555-5555-5555-5555-555555555555" // 用户钱七 → dept_id 悬空 ID
)

// setupBackfillDeptDB 建最小 schema + 覆盖各分支的工位数据。
func setupBackfillDeptDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	require.NoError(t, db.Exec(`
		CREATE TABLE sys_dept (id TEXT PRIMARY KEY, dept_name TEXT, deleted_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_user (id TEXT PRIMARY KEY, username TEXT, nickname TEXT, dept_id TEXT, deleted_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_workstation (
			id TEXT PRIMARY KEY, workstation_name TEXT, dept_id TEXT, user_id TEXT,
			status INTEGER DEFAULT 0, deleted_at DATETIME, created_at DATETIME, updated_at DATETIME
		)`).Error)

	require.NoError(t, db.Exec(`INSERT INTO sys_dept (id, dept_name) VALUES (?, '部门A'), (?, '部门B')`,
		bdDeptA, bdDeptB).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_user (id, username, nickname, dept_id) VALUES
		(?, 'zhangsan', '张三', ?),
		(?, 'lisi', '李四', ?),
		(?, 'wangwu', '王五', NULL),
		(?, 'zhaoliu', '赵六', '部门A'),
		(?, 'qianqi', '钱七', 'ghost-dept-id')`,
		bdUserA, bdDeptA, bdUserB, bdDeptB, bdUserC, bdUserD, bdUserE).Error)

	// 工位矩阵：
	// ws-null   dept_id NULL + 张三(部门A)      → fill
	// ws-empty  dept_id ''   + 张三(部门A)      → fill
	// ws-valid  dept_id=A    + 张三(部门A)      → 已一致，skip
	// ws-mism   dept_id=B    + 张三(部门A)      → 不一致（默认 skip / fixMismatch 时 correct）
	// ws-nouser dept_id NULL + 无用户           → skippedNoUser
	// ws-bad1   dept_id NULL + 王五(NULL 部门)  → skippedBadUserDept
	// ws-bad2   dept_id NULL + 赵六(部门名称)   → skippedBadUserDept
	// ws-bad3   dept_id NULL + 钱七(悬空 ID)    → skippedBadUserDept
	// ws-maint  dept_id NULL + 李四(部门B), status=2(维护) → fill 但 status 必须保持
	require.NoError(t, db.Exec(`INSERT INTO sys_workstation
		(id, workstation_name, dept_id, user_id, status) VALUES
		('ws-null',   'W-NULL',   NULL, ?, 0),
		('ws-empty',  'W-EMPTY',  '',   ?, 0),
		('ws-valid',  'W-VALID',  ?,    ?, 1),
		('ws-mism',   'W-MISM',   ?,    ?, 1),
		('ws-nouser', 'W-NOUSER', NULL, NULL, 0),
		('ws-bad1',   'W-BAD1',   NULL, ?, 0),
		('ws-bad2',   'W-BAD2',   NULL, ?, 0),
		('ws-bad3',   'W-BAD3',   NULL, ?, 0),
		('ws-maint',  'W-MAINT',  NULL, ?, 2)`,
		bdUserA, bdUserA, bdDeptA, bdUserA, bdDeptB, bdUserA,
		bdUserC, bdUserD, bdUserE, bdUserB).Error)
	return db
}

// readBackfillDept 读取单个工位的 dept_id / status。
func readBackfillDept(t *testing.T, db *gorm.DB, id string) (string, int) {
	t.Helper()
	var rec struct {
		DeptID *string
		Status int
	}
	require.NoError(t, db.Table("sys_workstation").Where("id = ?", id).First(&rec).Error, id)
	dept := ""
	if rec.DeptID != nil {
		dept = *rec.DeptID
	}
	return dept, rec.Status
}

// TestBackfillDept_DryRun_DefaultOnlyFill 默认模式 dry-run：只统计填补，不写库。
func TestBackfillDept_DryRun_DefaultOnlyFill(t *testing.T) {
	db := setupBackfillDeptDB(t)
	svc := NewWorkstationService(db)

	res, err := svc.BackfillDeptFromUser(context.Background(), true, false)
	require.NoError(t, err)
	assert.True(t, res.DryRun)
	assert.False(t, res.FixMismatch)
	assert.Equal(t, 5, res.Candidates, "有效候选 = null/empty/valid/mism/maint 共 5 条（3 个 bad 用户 + 1 个无用户不计）")
	assert.Equal(t, 3, res.ToFill, "null / empty / maint 三条待填补")
	assert.Equal(t, 0, res.ToCorrect, "默认不纠正")
	assert.Equal(t, 2, res.SkippedAlreadyValid, "ws-valid 一致 + ws-mism 不纠正")
	assert.Equal(t, 1, res.SkippedNoUser)
	assert.Equal(t, 3, res.SkippedBadUserDept)
	assert.LessOrEqual(t, len(res.Samples), 20)

	// dryRun 必须零写库。
	dept, status := readBackfillDept(t, db, "ws-null")
	assert.Empty(t, dept)
	assert.Equal(t, 0, status)
}

// 上一条断言里 SkippedAlreadyValid 应为 2（ws-valid 一致 + ws-mism 不纠正）。
// 拆成独立断言避免混淆，这里再单独核对。
func TestBackfillDept_DryRun_SkipCounts(t *testing.T) {
	db := setupBackfillDeptDB(t)
	svc := NewWorkstationService(db)
	res, err := svc.BackfillDeptFromUser(context.Background(), true, false)
	require.NoError(t, err)
	assert.Equal(t, 2, res.SkippedAlreadyValid, "ws-valid(一致) + ws-mism(不纠正) 都归入跳过")
	assert.Equal(t, 0, res.ToCorrect)
}

// TestBackfillDept_Execute_DefaultOnlyFill 默认执行：只填空缺，不覆盖已填部门，不动 status。
func TestBackfillDept_Execute_DefaultOnlyFill(t *testing.T) {
	db := setupBackfillDeptDB(t)
	svc := NewWorkstationService(db)

	res, err := svc.BackfillDeptFromUser(context.Background(), false, false)
	require.NoError(t, err)
	assert.Equal(t, 3, res.ToFill)
	assert.Equal(t, 0, res.ToCorrect)

	for _, id := range []string{"ws-null", "ws-empty"} {
		dept, _ := readBackfillDept(t, db, id)
		assert.Equal(t, bdDeptA, dept, "%s 应回填部门A", id)
	}
	deptMaint, statusMaint := readBackfillDept(t, db, "ws-maint")
	assert.Equal(t, bdDeptB, deptMaint)
	assert.Equal(t, 2, statusMaint, "维护状态必须保持不变（本次绝不碰 status）")

	// 已填部门一律不覆盖。
	deptValid, statusValid := readBackfillDept(t, db, "ws-valid")
	assert.Equal(t, bdDeptA, deptValid)
	assert.Equal(t, 1, statusValid)
	deptMism, _ := readBackfillDept(t, db, "ws-mism")
	assert.Equal(t, bdDeptB, deptMism, "默认模式不得纠正已有部门")

	// 脏数据工位保持空部门。
	for _, id := range []string{"ws-bad1", "ws-bad2", "ws-bad3", "ws-nouser"} {
		dept, _ := readBackfillDept(t, db, id)
		assert.Empty(t, dept, "%s 不应被回填", id)
	}
}

// TestBackfillDept_Execute_FixMismatch 开启纠正：覆盖不一致，但用户部门脏的仍跳过。
func TestBackfillDept_Execute_FixMismatch(t *testing.T) {
	db := setupBackfillDeptDB(t)
	svc := NewWorkstationService(db)

	res, err := svc.BackfillDeptFromUser(context.Background(), false, true)
	require.NoError(t, err)
	assert.True(t, res.FixMismatch)
	assert.Equal(t, 3, res.ToFill)
	assert.Equal(t, 1, res.ToCorrect, "仅 ws-mism 需纠正")

	deptMism, _ := readBackfillDept(t, db, "ws-mism")
	assert.Equal(t, bdDeptA, deptMism, "开启纠正后应改为用户部门A")
	deptValid, _ := readBackfillDept(t, db, "ws-valid")
	assert.Equal(t, bdDeptA, deptValid, "已一致的无需改写")
}

// TestBackfillDept_Idempotent 连跑两次，第二次零变更。
func TestBackfillDept_Idempotent(t *testing.T) {
	db := setupBackfillDeptDB(t)
	svc := NewWorkstationService(db)

	first, err := svc.BackfillDeptFromUser(context.Background(), false, false)
	require.NoError(t, err)
	assert.Equal(t, 3, first.ToFill)

	second, err := svc.BackfillDeptFromUser(context.Background(), false, false)
	require.NoError(t, err)
	assert.Equal(t, 0, second.ToFill, "第二次执行应零填补")
	assert.Equal(t, 0, second.ToCorrect)
}

// TestBackfillDept_StatusNeverTouched 三种状态在回填后全部保持。
func TestBackfillDept_StatusNeverTouched(t *testing.T) {
	db := setupBackfillDeptDB(t)
	// 追加一条占用状态工位。
	require.NoError(t, db.Exec(
		`INSERT INTO sys_workstation (id, workstation_name, dept_id, user_id, status)
		 VALUES ('ws-occ', 'W-OCC', NULL, ?, 1)`, bdUserA).Error)
	svc := NewWorkstationService(db)

	_, err := svc.BackfillDeptFromUser(context.Background(), false, false)
	require.NoError(t, err)
	for _, tc := range []struct {
		id   string
		want int
	}{{"ws-occ", 1}, {"ws-maint", 2}, {"ws-null", 0}} {
		_, status := readBackfillDept(t, db, tc.id)
		assert.Equal(t, tc.want, status, "%s 的 status 不应被改动", tc.id)
	}
}

// TestBackfillDept_CaseInsensitiveUUID 已有部门 UUID 大小写不同不算「不一致」，
// 避免 fixMismatch 模式下误覆盖（varchar 列里的 UUID 文本大小写可能不统一）。
func TestBackfillDept_CaseInsensitiveUUID(t *testing.T) {
	db := setupBackfillDeptDB(t)
	upper := "AAAAAAAA-AAAA-AAAA-AAAA-AAAAAAAAAAAA"
	require.NoError(t, db.Exec(
		`INSERT INTO sys_workstation (id, workstation_name, dept_id, user_id, status)
		 VALUES ('ws-upper', 'W-UPPER', ?, ?, 1)`, upper, bdUserA).Error)
	svc := NewWorkstationService(db)

	res, err := svc.BackfillDeptFromUser(context.Background(), true, true)
	require.NoError(t, err)
	// ws-upper 应归入「已一致」而非「待纠正」。
	assert.Equal(t, 1, res.ToCorrect, "仅 ws-mism 需纠正，大小写差异不计")
	dept, _ := readBackfillDept(t, db, "ws-upper")
	assert.Equal(t, upper, dept, "大小写差异不应被改写")
}

// TestBackfillDept_SoftDeletedExcluded 软删工位/用户不参与。
func TestBackfillDept_SoftDeletedExcluded(t *testing.T) {
	db := setupBackfillDeptDB(t)
	require.NoError(t, db.Exec(
		`INSERT INTO sys_workstation (id, workstation_name, dept_id, user_id, status, deleted_at)
		 VALUES ('ws-del', 'W-DEL', NULL, ?, 0, CURRENT_TIMESTAMP)`, bdUserA).Error)
	svc := NewWorkstationService(db)

	res, err := svc.BackfillDeptFromUser(context.Background(), true, false)
	require.NoError(t, err)
	assert.Equal(t, 3, res.ToFill, "软删工位不计入")
	assert.Equal(t, 1, res.SkippedNoUser, "软删工位不计入 total")
}
