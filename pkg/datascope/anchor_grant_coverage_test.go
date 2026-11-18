package datascope

// anchor_grant_coverage_test.go —— 补齐 anchor_grant.go 的 FormatRejections /
// AnchorDescription / LogAnchorGrant / CheckPostGrants 查询失败分支。
//
// 与 anchor_grant_test.go 共用 newGrantTestDB 夹具。

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

func TestFormatRejectionsBranches(t *testing.T) {
	assert.Empty(t, FormatRejections(nil))

	one := []PostGrantRejection{{PostID: "P1", PostName: "岗位甲", Reason: "不存在"}}
	assert.Equal(t, "岗位分配被拒绝：「岗位甲」：不存在", FormatRejections(one))

	noName := []PostGrantRejection{{PostID: "P1", Reason: "岗位不存在或已删除"}}
	assert.Equal(t, "岗位分配被拒绝：岗位不存在或已删除", FormatRejections(noName))

	many := []PostGrantRejection{
		{PostName: "A", Reason: "r1"},
		{PostName: "B", Reason: "r2"},
		{PostName: "C", Reason: "r3"},
		{PostName: "D", Reason: "r4"},
		{PostName: "E", Reason: "r5"},
	}
	msg := FormatRejections(many)
	assert.Contains(t, msg, "「A」：r1")
	assert.Contains(t, msg, "「C」：r3")
	assert.NotContains(t, msg, "「D」")
	assert.Contains(t, msg, "等 5 项岗位无法授予")
}

func TestAnchorDescriptionBranches(t *testing.T) {
	cases := []struct {
		mode models.AnchorMode
		want string
	}{
		{models.AnchorModeDept, "指定部门（含全部下级部门）"},
		{models.AnchorModeExternalOrg, "最近的外部机构（含全部下级）"},
		{models.AnchorModeLevel, "机构层级 2（含全部下级）"},
		{models.AnchorModeNone, "机构锚点"},
		{models.AnchorMode(9), "机构锚点"},
	}
	for _, c := range cases {
		cand := PostGrantCandidate{AnchorMode: c.mode, AnchorParam: "2"}
		assert.Equal(t, c.want, cand.AnchorDescription())
	}
}

func TestCheckPostGrants_QueryFailureReturnsError(t *testing.T) {
	// 缺 sys_post 表 → 查询失败，返回 error 交由调用方 500（不放行）。
	db := newBareCoverDB(t, "no-post-table.db")

	d, err := CheckPostGrants(context.Background(), db, []string{"P1"}, true, true, DefaultAnchorGrantPolicy())
	assert.Error(t, err)
	assert.Nil(t, d)
}

func TestLogAnchorGrant_NoPanic(t *testing.T) {
	// 空列表直接返回；非空列表写 WARN 日志（不返回错误，断言不 panic 即可）。
	assert.NotPanics(t, func() {
		LogAnchorGrant(context.Background(), "op", "target", nil)
		LogAnchorGrant(context.Background(), "op", "target", []PostGrantCandidate{
			{PostName: "中支管理员", AnchorMode: models.AnchorModeLevel, AnchorParam: "2"},
			{PostName: "外部机构管理员", AnchorMode: models.AnchorModeExternalOrg},
		})
	})
}
