package services

// =====================================================================
// Phase 79-02 Task 3: duty_pool_service.go 全 CRUD + statistics
//
// 覆盖目标: duty_pool_service.go 4.9% → ≥70%(基线 102 stmts / 97 unc,79-RESEARCH §2)。
// 既有 TestDutyPoolStatistics_NotDerivedFromCurrentPage(条件聚合口径)不动,
// 本文件只补未覆盖分支(重名/成员校验/更新/删除/软删排除)。
// 纪律同 79-02 文件头:7902 后缀 helper、t.TempDir 文件库、禁 t.Parallel、具名常量。
// =====================================================================

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// newDpl7902 装配 DutyPoolService + sqlite(t.TempDir 文件库)+ AutoMigrate duty 链 model。
// CreateDutyPool 事务后 Preload("Members.User").Preload("Dept") 重载 → sys_user/sys_dept 必须存在。
func newDpl7902(t *testing.T) (*DutyPoolService, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "dpl7902.db")), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err, "open sqlite temp db")
	if sqlDB, err := db.DB(); err == nil {
		t.Cleanup(func() { _ = sqlDB.Close() })
	}
	require.NoError(t, db.AutoMigrate(
		&models.DutyPool{},
		&models.DutyPoolMember{},
		&models.DutySchedule{},
		&models.DutyExchange{},
		&models.Holiday{},
		&models.DutyConfig{},
		&models.Department{},
		&models.User{},
		&models.Post{},
		&models.UserPost{},
	), "auto migrate duty chain models")
	return NewDutyPoolService(db), db
}

// dpl7902Create 构造创建请求(成员落 sys_user,通过 CreateDutyPool 的存在性校验)。
func dpl7902Create(t *testing.T, db *gorm.DB, svc *DutyPoolService, name string, dailyCount int, memberIDs ...string) *models.DutyPool {
	t.Helper()
	for _, uid := range memberIDs {
		dsc7902User(t, db, uid, "user-"+uid, nil, nil)
	}
	pool, err := svc.CreateDutyPool(context.Background(), &DutyPoolCreateRequest{
		PoolName:   name,
		DailyCount: dailyCount,
		MemberIDs:  memberIDs,
	}, dsc7902Creator)
	require.NoError(t, err, "create pool %s", name)
	return pool
}

// Members round-trip:创建请求里的成员列表写库后读回必须一致(成员数/ID 集/排序字段)。
func TestDpl7902_Create_Success(t *testing.T) {
	svc, db := newDpl7902(t)

	pool := dpl7902Create(t, db, svc, "create-pool", 2, "member-a", "member-b", "member-c")

	require.NotNil(t, pool)
	assert.NotEmpty(t, pool.ID)
	assert.Equal(t, "create-pool", pool.PoolName)
	assert.Equal(t, 2, pool.DailyCount)
	assert.Equal(t, dsc7902Creator, pool.CreatedBy, "CreatedBy=creatorID 透传")
	assert.Equal(t, models.DutyPoolStatusEnabled, pool.Status, "新建池默认启用")

	// round-trip:读回的 Members 与请求一致
	require.Len(t, pool.Members, 3)
	gotIDs := make([]string, 0, len(pool.Members))
	for _, m := range pool.Members {
		gotIDs = append(gotIDs, m.UserID)
	}
	assert.ElementsMatch(t, []string{"member-a", "member-b", "member-c"}, gotIDs)

	// 再经 GetDutyPoolByID 独立读回,防"返回值即内存态"假阳性
	reloaded, err := svc.GetDutyPoolByID(context.Background(), pool.ID)
	require.NoError(t, err)
	require.Len(t, reloaded.Members, 3)
	rtIDs := make([]string, 0, len(reloaded.Members))
	for _, m := range reloaded.Members {
		rtIDs = append(rtIDs, m.UserID)
		assert.Equal(t, pool.ID, m.PoolID, "成员行 PoolID 归属正确")
	}
	assert.ElementsMatch(t, []string{"member-a", "member-b", "member-c"}, rtIDs)
	// MemberOrder 按请求下标 0..n-1 落库(轮询顺序的物理载体)
	orders := make([]int, 0, len(reloaded.Members))
	for _, m := range reloaded.Members {
		orders = append(orders, m.MemberOrder)
	}
	assert.ElementsMatch(t, []int{0, 1, 2}, orders)
}

// CreateDutyPool 校验分支(:59-146):重名 / 成员用户不存在 / 成员为空的实装行为。
func TestDpl7902_Create_ValidationBranches(t *testing.T) {
	svc, db := newDpl7902(t)
	dpl7902Create(t, db, svc, "dup-pool", 1, "member-a")

	// 重名分支
	dup, err := svc.CreateDutyPool(context.Background(), &DutyPoolCreateRequest{
		PoolName:   "dup-pool",
		DailyCount: 1,
		MemberIDs:  []string{"member-a"},
	}, dsc7902Creator)
	require.Error(t, err, "重名必须报错")
	assert.Contains(t, err.Error(), "值班池名称已存在")
	assert.Nil(t, dup)
	var poolCount int64
	require.NoError(t, db.Model(&models.DutyPool{}).Count(&poolCount).Error)
	assert.Equal(t, int64(1), poolCount, "重名失败不落库(事务回滚)")

	// 成员用户不存在分支(逐个校验 :101-105)
	missing, err := svc.CreateDutyPool(context.Background(), &DutyPoolCreateRequest{
		PoolName:   "missing-user-pool",
		DailyCount: 1,
		MemberIDs:  []string{"member-a", "ghost-user"},
	}, dsc7902Creator)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "用户不存在: ghost-user")
	assert.Nil(t, missing)
	require.NoError(t, db.Model(&models.DutyPool{}).Count(&poolCount).Error)
	assert.Equal(t, int64(1), poolCount, "成员校验失败事务回滚,池不落库")
	var memberCount int64
	require.NoError(t, db.Model(&models.DutyPoolMember{}).Count(&memberCount).Error)
	assert.Equal(t, int64(1), memberCount, "仅剩首个成功池(dup-pool)的 1 条成员行,失败池成员已回滚")

	// QUIRK-79-02-F(只锁不修):MemberIDs 为空 → 实装跳过成员段(:87 len>0),
	// 池照常创建且无成员;required/min=1 校验在 handler binding 层,service 层不拦。
	empty, err := svc.CreateDutyPool(context.Background(), &DutyPoolCreateRequest{
		PoolName:   "empty-member-pool",
		DailyCount: 1,
	}, dsc7902Creator)
	require.NoError(t, err, "实装不校验空成员(binding 层职责),按现行为锁定")
	require.NotNil(t, empty)
	reloaded, err := svc.GetDutyPoolByID(context.Background(), empty.ID)
	require.NoError(t, err)
	assert.Empty(t, reloaded.Members, "空成员池 Members 为空切片")
}

// GetDutyPoolList:分页 + 关键字/部门/状态过滤 + Preload 成员。
func TestDpl7902_GetList_Pagination(t *testing.T) {
	svc, db := newDpl7902(t)
	deptA := "dept-a-uuid"
	deptB := "dept-b-uuid"
	dpl7902Create(t, db, svc, "alpha-pool", 1, "member-a")
	poolB := dpl7902Create(t, db, svc, "beta-pool", 1, "member-b")
	poolC := dpl7902Create(t, db, svc, "gamma-pool", 1, "member-c")
	// 归属部门(直接改列,CreateDutyPool 不带 DeptID 场景)
	require.NoError(t, db.Model(&models.DutyPool{}).Where("id = ?", poolB.ID).
		Update("dept_id", deptA).Error)
	require.NoError(t, db.Model(&models.DutyPool{}).Where("id = ?", poolC.ID).
		Update("dept_id", deptB).Error)

	// 分页语义(默认 PageSize=10;Current=1 PageSize=2)
	pools, total, err := svc.GetDutyPoolList(context.Background(), &DutyPoolListRequest{
		BaseListRequest: base7902Req(1, 2),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, pools, 2, "PageSize=2 只取 2 行")

	// 关键字模糊过滤
	pools, total, err = svc.GetDutyPoolList(context.Background(), &DutyPoolListRequest{
		PoolName: &[]string{"eta"}[0], // beta 关键字
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, pools, 1)
	assert.Equal(t, "beta-pool", pools[0].PoolName)

	// 部门过滤
	pools, total, err = svc.GetDutyPoolList(context.Background(), &DutyPoolListRequest{
		DeptID: &deptA,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, pools, 1)
	assert.Equal(t, poolB.ID, pools[0].ID)

	// 状态过滤:先停用 gamma
	disabled := int(models.DutyPoolStatusDisabled)
	require.NoError(t, svc.UpdateDutyPool(context.Background(), &DutyPoolUpdateRequest{
		ID:         poolC.ID,
		PoolName:   "gamma-pool",
		DailyCount: 1,
		Status:     &disabled,
		MemberIDs:  []string{"member-c"},
	}, dsc7902Creator))
	pools, total, err = svc.GetDutyPoolList(context.Background(), &DutyPoolListRequest{
		Status: &disabled,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, pools, 1)
	assert.Equal(t, models.DutyPoolStatusDisabled, pools[0].Status)

	// Preload("Members.User") 生效
	pools, _, err = svc.GetDutyPoolList(context.Background(), &DutyPoolListRequest{
		PoolName: &[]string{"alpha"}[0],
	})
	require.NoError(t, err)
	require.Len(t, pools, 1)
	require.Len(t, pools[0].Members, 1)
	require.NotNil(t, pools[0].Members[0].User, "成员的 User 应被预载")
	assert.Equal(t, "user-member-a", pools[0].Members[0].User.Username)
}

// GetDutyPoolByID 命中/未命中两分支。
func TestDpl7902_GetByID_MissAndHit(t *testing.T) {
	svc, db := newDpl7902(t)
	pool := dpl7902Create(t, db, svc, "getbyid-pool", 1, "member-a")

	hit, err := svc.GetDutyPoolByID(context.Background(), pool.ID)
	require.NoError(t, err)
	assert.Equal(t, "getbyid-pool", hit.PoolName)
	assert.Equal(t, models.DutyPoolStatusEnabled, hit.Status)

	miss, err := svc.GetDutyPoolByID(context.Background(), "no-such-pool")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "值班池不存在")
	assert.Nil(t, miss)
}

// UpdateDutyPool:合法更新(名称+成员+DailyCount+Status)读回一致;不存在 ID;
// 重名;QUIRK:更新段不校验成员存在性(与创建不对称)。
func TestDpl7902_Update_FullAndMissing(t *testing.T) {
	svc, db := newDpl7902(t)
	pool := dpl7902Create(t, db, svc, "update-pool", 1, "member-a")
	dpl7902Create(t, db, svc, "other-pool", 1, "member-b")
	dsc7902User(t, db, "member-c", "user-member-c", nil, nil)

	enabled := int(models.DutyPoolStatusDisabled)
	require.NoError(t, svc.UpdateDutyPool(context.Background(), &DutyPoolUpdateRequest{
		ID:          pool.ID,
		PoolName:    "update-pool-v2",
		Description: "更新后的描述",
		DailyCount:  3,
		Status:      &enabled,
		MemberIDs:   []string{"member-a", "member-c"},
	}, "updater-7902"))

	reloaded, err := svc.GetDutyPoolByID(context.Background(), pool.ID)
	require.NoError(t, err)
	assert.Equal(t, "update-pool-v2", reloaded.PoolName)
	assert.Equal(t, "更新后的描述", reloaded.Description)
	assert.Equal(t, 3, reloaded.DailyCount)
	assert.Equal(t, models.DutyPoolStatusDisabled, reloaded.Status, "显式 Status 指针才更新状态")
	assert.Equal(t, "updater-7902", reloaded.UpdatedBy)
	// 成员被整段替换(先删后建)
	require.Len(t, reloaded.Members, 2)
	newIDs := make([]string, 0, 2)
	for _, m := range reloaded.Members {
		newIDs = append(newIDs, m.UserID)
	}
	assert.ElementsMatch(t, []string{"member-a", "member-c"}, newIDs)

	// 不存在 ID → "值班池不存在"
	err = svc.UpdateDutyPool(context.Background(), &DutyPoolUpdateRequest{
		ID:         "no-such-pool",
		PoolName:   "whatever",
		DailyCount: 1,
		MemberIDs:  []string{"member-a"},
	}, dsc7902Creator)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "值班池不存在")

	// 与其他池重名 → "值班池名称已存在"
	err = svc.UpdateDutyPool(context.Background(), &DutyPoolUpdateRequest{
		ID:         pool.ID,
		PoolName:   "other-pool",
		DailyCount: 1,
		MemberIDs:  []string{"member-a"},
	}, dsc7902Creator)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "值班池名称已存在")

	// QUIRK-79-02-G(只锁不修):更新段不校验成员存在性(:246-258 无存在性查询),
	// 与创建段(:87-105)不对称;幽灵成员照常落库,按现行为锁定。
	err = svc.UpdateDutyPool(context.Background(), &DutyPoolUpdateRequest{
		ID:         pool.ID,
		PoolName:   "update-pool-v3",
		DailyCount: 1,
		MemberIDs:  []string{"ghost-member"},
	}, dsc7902Creator)
	require.NoError(t, err, "实装更新不校验成员存在(quirk 锁定)")
	after, err := svc.GetDutyPoolByID(context.Background(), pool.ID)
	require.NoError(t, err)
	require.Len(t, after.Members, 1)
	assert.Equal(t, "ghost-member", after.Members[0].UserID)
}

// DeleteDutyPool:有排班记录拒删;正常删除连带清成员;再删不存在的 ID 不报错。
func TestDpl7902_Delete(t *testing.T) {
	svc, db := newDpl7902(t)
	pool := dpl7902Create(t, db, svc, "delete-pool", 1, "member-a")

	// 有排班记录 → 拒删(:269-274)
	require.NoError(t, db.Create(&models.DutySchedule{
		BaseModel:    models.BaseModel{CreatedBy: dsc7902Creator},
		ScheduleDate: dsc7902Mon,
		PoolID:       pool.ID,
		UserID:       "member-a",
		DutyType:     models.ScheduleModeWeekday,
		Status:       models.DutyStatusNormal,
	}).Error)
	err := svc.DeleteDutyPool(context.Background(), pool.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "该值班池存在排班记录，无法删除")
	var poolCount int64
	require.NoError(t, db.Model(&models.DutyPool{}).Count(&poolCount).Error)
	assert.Equal(t, int64(1), poolCount, "拒删后池仍在")

	// 无排班 → 删除成功且成员连带清理
	require.NoError(t, db.Where("pool_id = ?", pool.ID).Delete(&models.DutySchedule{}).Error)
	require.NoError(t, svc.DeleteDutyPool(context.Background(), pool.ID))

	_, err = svc.GetDutyPoolByID(context.Background(), pool.ID)
	require.Error(t, err, "删除后 GetByID miss")
	var memberCount int64
	require.NoError(t, db.Model(&models.DutyPoolMember{}).Where("pool_id = ?", pool.ID).Count(&memberCount).Error)
	assert.Zero(t, memberCount, "成员行连带删除")

	// 再删不存在的 ID → 实装不报错(排班计数 0 → 删除空集,两段 Delete 均 0 行)
	err = svc.DeleteDutyPool(context.Background(), pool.ID)
	require.NoError(t, err, "软删除空集不报错(按现行为锁定)")
}

// Statistics:启停聚合 + 成员总数;删除池后其成员被软删 scope 排除。
// 口径对照既有 TestDutyPoolStatistics_NotDerivedFromCurrentPage(条件聚合 + 子查询),
// 本用例走 service 全链(Create/Update/Delete)驱动,补足非 SQL 直插路径。
func TestDpl7902_Statistics(t *testing.T) {
	svc, db := newDpl7902(t)
	dpl7902Create(t, db, svc, "stat-pool-1", 1, "member-a", "member-b") // 启用,2 成员
	pool2 := dpl7902Create(t, db, svc, "stat-pool-2", 1, "member-c")    // 启用,1 成员
	pool3 := dpl7902Create(t, db, svc, "stat-pool-3", 1, "member-d")    // 启用,1 成员

	disabled := int(models.DutyPoolStatusDisabled)
	require.NoError(t, svc.UpdateDutyPool(context.Background(), &DutyPoolUpdateRequest{
		ID:         pool2.ID,
		PoolName:   "stat-pool-2",
		DailyCount: 1,
		Status:     &disabled,
		MemberIDs:  []string{"member-c"},
	}, dsc7902Creator))

	stats, err := svc.GetDutyPoolStatistics(context.Background())
	require.NoError(t, err)
	require.NotNil(t, stats)
	assert.Equal(t, int64(3), stats.Total)
	assert.Equal(t, int64(2), stats.Enabled, "两个启用池")
	assert.Equal(t, int64(1), stats.Disabled, "一个停用池")
	assert.Equal(t, int64(4), stats.TotalMembers, "2+1+1=4 个成员(跨池合计)")

	// 删除 stat-pool-3 → 池软删,其成员被子查询的软删 scope 排除
	require.NoError(t, svc.DeleteDutyPool(context.Background(), pool3.ID))
	stats, err = svc.GetDutyPoolStatistics(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(2), stats.Total, "删除后只剩 2 池")
	assert.Equal(t, int64(1), stats.Enabled)
	assert.Equal(t, int64(1), stats.Disabled)
	assert.Equal(t, int64(3), stats.TotalMembers, "被删池的成员一并排除(2+1=3)")
}
