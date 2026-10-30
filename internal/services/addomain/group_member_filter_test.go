//go:build !skip_db_tests
// +build !skip_db_tests

package addomain

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// ============================================================================
// HYGIENE-04:嵌套组成员 DN 过滤（62-02）
// ============================================================================

// TestFilterNestedGroupMemberDNs_Basic 基本分流:集合内 DN 进 filtered,其余进 kept。
func TestFilterNestedGroupMemberDNs_Basic(t *testing.T) {
	set := map[string]struct{}{
		strings.ToLower("CN=Inner,OU=Groups,DC=example,DC=com"): {},
	}
	kept, filtered := filterNestedGroupMemberDNs([]string{
		"CN=UserA,OU=Users,DC=example,DC=com",
		"CN=Inner,OU=Groups,DC=example,DC=com",
	}, set)
	assert.Equal(t, []string{"CN=UserA,OU=Users,DC=example,DC=com"}, kept)
	assert.Equal(t, []string{"CN=Inner,OU=Groups,DC=example,DC=com"}, filtered)
}

// TestFilterNestedGroupMemberDNs_CaseInsensitive DN 大小写变体必须被过滤（T-02-SC2）。
func TestFilterNestedGroupMemberDNs_CaseInsensitive(t *testing.T) {
	set := map[string]struct{}{
		"cn=cxhubinternet-vip,ou=groups,dc=company,dc=com": {},
	}
	kept, filtered := filterNestedGroupMemberDNs([]string{
		"CN=CXHUBinternet-VIP,OU=Groups,DC=Company,DC=COM",
		"CN=UserB,OU=Users,DC=Company,DC=COM",
	}, set)
	assert.Equal(t, []string{"CN=UserB,OU=Users,DC=Company,DC=COM"}, kept)
	assert.Equal(t, []string{"CN=CXHUBinternet-VIP,OU=Groups,DC=Company,DC=COM"}, filtered)
}

// TestFilterNestedGroupMemberDNs_NilSetKeepsAll nil 集合=防御语义,全保留零过滤。
func TestFilterNestedGroupMemberDNs_NilSetKeepsAll(t *testing.T) {
	in := []string{"CN=A,DC=x", "CN=B,DC=x"}
	kept, filtered := filterNestedGroupMemberDNs(in, nil)
	assert.Equal(t, in, kept)
	assert.Empty(t, filtered)
	// kept 与 filtered 不得有交集
	for _, k := range kept {
		for _, f := range filtered {
			assert.NotEqual(t, k, f)
		}
	}
}

// TestFilterNestedGroupMemberDNs_EmptyMember 空输入 → 空输出。
func TestFilterNestedGroupMemberDNs_EmptyMember(t *testing.T) {
	kept, filtered := filterNestedGroupMemberDNs([]string{}, map[string]struct{}{"a": {}})
	assert.Empty(t, kept)
	assert.Empty(t, filtered)
}

// TestSyncGroupMembers_NestedGroupMemberNotPersisted 组 G 传 [用户A, 嵌套组B] 且集合含 B
// → 落库仅用户A（SC2 本体）。嵌套组 DN 形态取生产实锤之一（CN=CXHUBinternet-VIP）。
func TestSyncGroupMembers_NestedGroupMemberNotPersisted(t *testing.T) {
	svc, db := newSyncSvc78(t)
	defer closeDB(t, db)
	ctx := context.Background()
	cfg := insertConfig78(t, db, "")
	groupG := "CN=G,OU=Groups,DC=example,DC=com"
	nestedB := "CN=CXHUBinternet-VIP,OU=Groups,DC=example,DC=com"

	// 预置嵌套组 B 行（口径与生产一致:嵌套 DN 命中 sys_ad_group.group_dn）
	g := models.ADGroup{ADConfigID: cfg.ID, GroupDN: nestedB, GroupName: "CXHUBinternet-VIP", MemberCount: 0}
	g.ID = uuid.NewString()
	require.NoError(t, db.Create(&g).Error)

	set, err := svc.loadKnownGroupDNSet(ctx, cfg.ID)
	require.NoError(t, err)
	userA := "CN=UserA,OU=Users,DC=example,DC=com"
	require.NoError(t, svc.syncGroupMembers(ctx, cfg, groupG, []string{userA, nestedB}, set))

	var rows []models.ADGroupMember
	require.NoError(t, db.Unscoped().Where("ad_config_id = ? AND group_dn = ?", cfg.ID, groupG).Find(&rows).Error)
	require.Len(t, rows, 1, "嵌套组 DN 不得写入用户成员关系")
	assert.Equal(t, userA, rows[0].UserDN)
}

// TestSyncGroupMembers_ReplayPurgesLegacyNestedRow 预置历史污染行（group G + user_dn=B）
// → 过滤重放 [A,B] → B 行消失（重放自愈证明）。
func TestSyncGroupMembers_ReplayPurgesLegacyNestedRow(t *testing.T) {
	svc, db := newSyncSvc78(t)
	defer closeDB(t, db)
	ctx := context.Background()
	cfg := insertConfig78(t, db, "")
	groupG := "CN=G,OU=Groups,DC=example,DC=com"
	nestedB := "CN=CXHUBinternet-VIP,OU=Groups,DC=example,DC=com"

	// 历史污染行 + 嵌套组 B 行
	require.NoError(t, db.Create(&models.ADGroupMember{
		ADConfigID: cfg.ID, GroupDN: groupG, UserDN: nestedB,
	}).Error)
	g := models.ADGroup{ADConfigID: cfg.ID, GroupDN: nestedB, GroupName: "CXHUBinternet-VIP", MemberCount: 0}
	g.ID = uuid.NewString()
	require.NoError(t, db.Create(&g).Error)

	set, err := svc.loadKnownGroupDNSet(ctx, cfg.ID)
	require.NoError(t, err)
	userA := "CN=UserA,OU=Users,DC=example,DC=com"
	require.NoError(t, svc.syncGroupMembers(ctx, cfg, groupG, []string{userA, nestedB}, set))

	var rows []models.ADGroupMember
	require.NoError(t, db.Unscoped().Where("ad_config_id = ? AND group_dn = ?", cfg.ID, groupG).Find(&rows).Error)
	require.Len(t, rows, 1, "存量污染行随重放清除")
	assert.Equal(t, userA, rows[0].UserDN)
}

// TestSyncGroupMembers_NilSetKeepsAllWarns nil 集合=防御路径:不过滤、旧行为原样。
func TestSyncGroupMembers_NilSetKeepsAllWarns(t *testing.T) {
	svc, db := newSyncSvc78(t)
	defer closeDB(t, db)
	ctx := context.Background()
	cfg := insertConfig78(t, db, "")
	groupG := "CN=G,OU=Groups,DC=example,DC=com"
	// 传入 DN 命中库里组 DN（若误过滤会丢行）——nil 集合必须全保留
	nestedB := "CN=CXHUBinternet-VIP,OU=Groups,DC=example,DC=com"
	g := models.ADGroup{ADConfigID: cfg.ID, GroupDN: nestedB, GroupName: "CXHUBinternet-VIP", MemberCount: 0}
	g.ID = uuid.NewString()
	require.NoError(t, db.Create(&g).Error)

	userA := "CN=UserA,OU=Users,DC=example,DC=com"
	require.NoError(t, svc.syncGroupMembers(ctx, cfg, groupG, []string{userA, nestedB}, nil))

	var rows []models.ADGroupMember
	require.NoError(t, db.Unscoped().Where("ad_config_id = ? AND group_dn = ?", cfg.ID, groupG).Find(&rows).Error)
	require.Len(t, rows, 2, "nil 集合=防御语义,全部保留")
}

// TestLoadKnownGroupDNSet_ExcludesSoftDeleted 软删组不得进入集合。
func TestLoadKnownGroupDNSet_ExcludesSoftDeleted(t *testing.T) {
	svc, db := newSyncSvc78(t)
	defer closeDB(t, db)
	ctx := context.Background()
	cfg := insertConfig78(t, db, "")
	live := "CN=Live,OU=Groups,DC=example,DC=com"
	soft := "CN=Soft,OU=Groups,DC=example,DC=com"
	glive := models.ADGroup{ADConfigID: cfg.ID, GroupDN: live}
	glive.ID = uuid.NewString()
	require.NoError(t, db.Create(&glive).Error)
	gsoft := models.ADGroup{ADConfigID: cfg.ID, GroupDN: soft}
	gsoft.ID = uuid.NewString()
	gsoft.DeletedAt = gorm.DeletedAt{Time: time.Now(), Valid: true}
	require.NoError(t, db.Create(&gsoft).Error)

	set, err := svc.loadKnownGroupDNSet(ctx, cfg.ID)
	require.NoError(t, err)
	_, hasLive := set[strings.ToLower(live)]
	_, hasSoft := set[strings.ToLower(soft)]
	assert.True(t, hasLive)
	assert.False(t, hasSoft, "软删组不得进入过滤集合")
}


// TestSyncGroupMembers_AllMembersNestedNoEmptySliceError 回归 62-02 生产缺陷:
// 生产存在成员"全部"是嵌套组的组(CN=CXHUB-人资条线,OU=外网权限,...),
// 过滤后 kept 为空,GORM Create(空切片) 返回 ErrEmptySlice("empty slice found"),
// 事务回滚并让整轮 syncGroups 中止(2026-10-22 14:05 轮 sync_status=failed)。
// 期望:不返回错误,且该组零行(Delete 已随事务提交)。
func TestSyncGroupMembers_AllMembersNestedNoEmptySliceError(t *testing.T) {
	svc, db := newSyncSvc78(t)
	defer closeDB(t, db)
	ctx := context.Background()
	cfg := insertConfig78(t, db, "")
	groupG := "CN=CXHUB-人资条线,OU=外网权限,OU=MgtGrp,DC=example,DC=com"
	nestedOnly := "CN=CXHUB-人力资源部,OU=Groups,DC=example,DC=com"

	// 预置:唯一成员本身是一个已入库的组 DN
	g := models.ADGroup{ADConfigID: cfg.ID, GroupDN: nestedOnly, GroupName: "CXHUB-人力资源部", MemberCount: 0}
	g.ID = uuid.NewString()
	require.NoError(t, db.Create(&g).Error)

	// 预置历史行(模拟重放前的存量关系)，验证 Delete 之后不回填
	legacy := models.ADGroupMember{ADConfigID: cfg.ID, GroupDN: groupG, UserDN: nestedOnly}
	legacy.ID = uuid.NewString()
	require.NoError(t, db.Create(&legacy).Error)

	set, err := svc.loadKnownGroupDNSet(ctx, cfg.ID)
	require.NoError(t, err)

	// 关键断言:不得返回 "empty slice found"
	require.NoError(t, svc.syncGroupMembers(ctx, cfg, groupG, []string{nestedOnly}, set),
		"全嵌套成员的组不得因 GORM ErrEmptySlice 中断整轮同步")

	var rows []models.ADGroupMember
	require.NoError(t, db.Unscoped().Where("ad_config_id = ? AND group_dn = ?", cfg.ID, groupG).Find(&rows).Error)
	assert.Len(t, rows, 0, "过滤后为空:该组应零行")
}
