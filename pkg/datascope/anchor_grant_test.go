package datascope

// anchor_grant_test.go —— 机构锚点岗位准入保护的回归（v1.54 安全补丁）。
//
// 背景：verifier 报的阻塞级漏洞 P-1。
//   user_service 直接写 sys_user_post，对 PostIds 零校验；而岗位维度锚点按
//   **任职人自己的部门链**解析。于是持 system:user:edit 的中支管理员可以把
//   自己挂到省管理员建的「分公司管理员」岗位(anchor_mode=5,param="1")，
//   一次表单提交就把可见集从"本部门"扩展到"全省"。
//
// 本测试锁死准入判定的四条规则（见 pkg/datascope/anchor_grant.go CheckPostGrants）：
//  1. 岗位不存在 → 拒绝
//  2. 岗位停用 → 拒绝
//  3. 携带锚点 + 策略要求 post:edit + 操作者无此权限 → 拒绝
//  4. 携带锚点 + 策略要求超管 + 操作者非超管 → 拒绝
// 以及：
//  5. 无锚点岗位 → 任何持 user:edit 的人都能分配（零回归）
//  6. 超管 → 不受限
//  7. 携带锚点的岗位通过 → AnchorPosts 记录完整（供 operlog 审计）
//  8. 空 postIDs → 直接放行（只改用户基本信息）

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
)

// newGrantTestDB 工种岗位夹具：普通岗位 + 三种携带锚点的岗位。
func newGrantTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "grant.db")), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		if sqlDB, e := db.DB(); e == nil {
			_ = sqlDB.Close()
		}
	})

	require.NoError(t, db.Exec(`CREATE TABLE sys_post (
		id TEXT PRIMARY KEY, post_code TEXT, post_name TEXT, post_sort INTEGER DEFAULT 0,
		anchor_mode INTEGER DEFAULT 0, anchor_param TEXT DEFAULT '',
		status INTEGER DEFAULT 0, remark TEXT, deleted_at DATETIME)`).Error)

	require.NoError(t, db.Exec(`INSERT INTO sys_post (id, post_code, post_name, anchor_mode, anchor_param, status, deleted_at) VALUES
		('P-普通',   'normal',  '普通岗位',             0, '',   0, NULL),
		('P-中支',   'mid',     '中支管理员',           5, '2',  0, NULL),
		('P-分公司', 'branch',  '分公司管理员',         5, '1',  0, NULL),
		('P-支公司', 'office',  '支公司管理员',         5, '3',  0, NULL),
		('P-外部',   'ext',     '外部机构管理员',       3, '',   0, NULL),
		('P-指定',   'fixed',   '指定部门管理员',       2, 'D-wh', 0, NULL),
		('P-停用',   'disabled','停用岗位',             0, '',   1, NULL),
		('P-已删',   'deleted', '已删岗位',             0, '',   0, '2026-01-01 00:00:00')`).Error)

	return db
}

// TestCheckPostGrants_AnchorPost_RequiresPostEdit 规则 3：无 post:edit 时拒绝携带锚点的岗位。
func TestCheckPostGrants_AnchorPost_RequiresPostEdit(t *testing.T) {
	db := newGrantTestDB(t)
	ctx := context.Background()
	policy := DefaultAnchorGrantPolicy() // RequirePostEditPerm = true

	// 中支管理员（有 user:edit、无 post:edit）想把「分公司管理员」挂给某人 → 必须拒绝。
	d, err := CheckPostGrants(ctx, db, []string{"P-分公司"}, false, false, policy)
	require.NoError(t, err)
	assert.False(t, d.Allowed, "无 post:edit 者不得授予携带锚点的岗位（P-1 漏洞的核心）")
	require.Len(t, d.Rejected, 1)
	assert.Contains(t, d.Rejected[0].Reason, "system:post:edit")
	assert.Contains(t, d.Rejected[0].Reason, "分公司管理员")

	// 普通岗位不受影响（零回归）。
	d2, err := CheckPostGrants(ctx, db, []string{"P-普通"}, false, false, policy)
	require.NoError(t, err)
	assert.True(t, d2.Allowed, "普通岗位分配不得因本补丁被误伤")
	assert.Empty(t, d2.AnchorPosts)
}

// TestCheckPostGrants_AnchorPost_AllowedForPostEditHolder 有 post:edit → 放行并审计记录。
func TestCheckPostGrants_AnchorPost_AllowedForPostEditHolder(t *testing.T) {
	db := newGrantTestDB(t)
	ctx := context.Background()

	d, err := CheckPostGrants(ctx, db, []string{"P-中支"}, true, false, DefaultAnchorGrantPolicy())
	require.NoError(t, err)
	assert.True(t, d.Allowed)
	require.Len(t, d.AnchorPosts, 1)
	assert.Equal(t, "中支管理员", d.AnchorPosts[0].PostName)
	assert.Equal(t, models.AnchorModeLevel, d.AnchorPosts[0].AnchorMode)
	assert.Equal(t, "2", d.AnchorPosts[0].AnchorParam)
	// AnchorPost 记录是 operlog/审计的依据，必须带得出可读描述。
	assert.Contains(t, d.AnchorPosts[0].AnchorDescription(), "机构层级")
}

// TestCheckPostGrants_SuperAdminBypass 超管在有 post:edit 语义时天然通过。
func TestCheckPostGrants_SuperAdminBypass(t *testing.T) {
	db := newGrantTestDB(t)
	ctx := context.Background()

	// operatorHasPostEdit 由 handler 判定；超管路径会先把它置 true。
	d, err := CheckPostGrants(ctx, db, []string{"P-分公司"}, true, true, DefaultAnchorGrantPolicy())
	require.NoError(t, err)
	assert.True(t, d.Allowed)
	assert.Len(t, d.AnchorPosts, 1)
}

// TestCheckPostGrants_AllowAnchorOnlyForAdmin 最严格档：非超管一律拒绝携带锚点的岗位。
func TestCheckPostGrants_AllowAnchorOnlyForAdmin(t *testing.T) {
	db := newGrantTestDB(t)
	ctx := context.Background()
	policy := AnchorGrantPolicy{AllowAnchorOnlyForAdmin: true}

	// 即便有 post:edit，非超管也不能授予。
	d, err := CheckPostGrants(ctx, db, []string{"P-中支"}, true, false, policy)
	require.NoError(t, err)
	assert.False(t, d.Allowed)
	assert.Contains(t, d.Rejected[0].Reason, "仅超级管理员")

	// 超管可以。
	d2, err := CheckPostGrants(ctx, db, []string{"P-中支"}, true, true, policy)
	require.NoError(t, err)
	assert.True(t, d2.Allowed)
}

// TestCheckPostGrants_RejectsMissingAndDisabled 规则 1/2：不存在与停用一律拒绝。
func TestCheckPostGrants_RejectsMissingAndDisabled(t *testing.T) {
	db := newGrantTestDB(t)
	ctx := context.Background()
	policy := DefaultAnchorGrantPolicy()

	// 不存在
	d, err := CheckPostGrants(ctx, db, []string{"P-not-exist"}, true, true, policy)
	require.NoError(t, err)
	assert.False(t, d.Allowed)
	assert.Contains(t, d.Rejected[0].Reason, "不存在或已删除")

	// 已软删（查询带 deleted_at IS NULL，等同不存在）
	d2, err := CheckPostGrants(ctx, db, []string{"P-已删"}, true, true, policy)
	require.NoError(t, err)
	assert.False(t, d2.Allowed)

	// 停用（即便超管也不该挂：停用岗位不产生任何权限，挂上会误导）
	d3, err := CheckPostGrants(ctx, db, []string{"P-停用"}, true, true, policy)
	require.NoError(t, err)
	assert.False(t, d3.Allowed)
	assert.Contains(t, d3.Rejected[0].Reason, "已停用")
}

// TestCheckPostGrants_EmptyAndDuplicateIDs 空列表放行；重复 ID 归一化。
func TestCheckPostGrants_EmptyAndDuplicateIDs(t *testing.T) {
	db := newGrantTestDB(t)
	ctx := context.Background()

	d, err := CheckPostGrants(ctx, db, nil, false, false, DefaultAnchorGrantPolicy())
	require.NoError(t, err)
	assert.True(t, d.Allowed, "不分配岗位时不得阻断用户编辑")

	d2, err := CheckPostGrants(ctx, db, []string{"P-普通", "P-普通", ""}, false, false, DefaultAnchorGrantPolicy())
	require.NoError(t, err)
	assert.True(t, d2.Allowed)
	assert.Empty(t, d2.Rejected, "重复/空 ID 归一化后不应产生拒绝")
}

// TestCheckPostGrants_MultipleMixed 混合场景：一批里有合法与越权岗位 → 整批拒绝。
func TestCheckPostGrants_MultipleMixed(t *testing.T) {
	db := newGrantTestDB(t)
	ctx := context.Background()

	d, err := CheckPostGrants(ctx, db,
		[]string{"P-普通", "P-分公司", "P-中支"}, false, false, DefaultAnchorGrantPolicy())
	require.NoError(t, err)
	assert.False(t, d.Allowed, "一批里只要有一个越权岗位就必须整批拒绝（不能部分落库）")
	require.Len(t, d.Rejected, 2, "分公司 + 中支两个锚点岗位都应被拒")

	msg := FormatRejections(d.Rejected)
	assert.Contains(t, msg, "岗位分配被拒绝")
	assert.Contains(t, msg, "分公司管理员")
}

// TestCheckPostGrants_AllThreeAnchorModes 三种锚点模式都被识别为"携带锚点"。
func TestCheckPostGrants_AllThreeAnchorModes(t *testing.T) {
	db := newGrantTestDB(t)
	ctx := context.Background()

	for _, id := range []string{"P-中支", "P-分公司", "P-支公司", "P-外部", "P-指定"} {
		d, err := CheckPostGrants(ctx, db, []string{id}, false, false, DefaultAnchorGrantPolicy())
		require.NoError(t, err)
		assert.False(t, d.Allowed, "%s 携带锚点，无 post:edit 时必须被拒", id)
	}
}

// TestAnchorCarries_IgnoresInvalidMode 脏 anchor_mode（如弃用的 1/4）不视为"携带锚点"。
//
// 理由：ResolveAnchor 对非法 mode 返回 BadMode 且不解析（不越权）。
// 但需要在授予侧保持一致：脏数据不构成额外的授权门槛，也不构成"安全假象"。
// 真正防护由 ResolveAnchor 的 BadMode 分支兜底。
func TestAnchorCarries_IgnoresInvalidMode(t *testing.T) {
	for _, mode := range []models.AnchorMode{0, 1, 4, 6, 9} {
		c := PostGrantCandidate{Exist: true, AnchorMode: mode}
		assert.False(t, c.AnchorCarries(), "mode=%d 不应视为携带机构锚点", mode)
	}
	for _, mode := range []models.AnchorMode{2, 3, 5} {
		c := PostGrantCandidate{Exist: true, AnchorMode: mode}
		assert.True(t, c.AnchorCarries(), "mode=%d 必须视为携带机构锚点", mode)
	}
	// 不存在的岗位永远不携带
	c := PostGrantCandidate{Exist: false, AnchorMode: 5}
	assert.False(t, c.AnchorCarries())
}
