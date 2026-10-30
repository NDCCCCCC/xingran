//go:build !skip_db_tests
// +build !skip_db_tests

package operations

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/stretchr/testify/require"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupLocationAliasTestDB 创建内存 SQLite + AutoMigrate 建表
// 包含 SysDeptLocationAlias + Department 两张表,validateAlias 需要查 sys_dept。
//
// 注:`file::memory:?cache=shared` 让同一进程内多个 gorm.Open 共享同一份内存数据库。
// 为避免测试间数据污染,每次 setup 先 TRUNCATE(alias 软删除 + department 硬删)。
func setupLocationAliasTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Dialector{
		DriverName: "sqlite",
		DSN:        "file::memory:?cache=shared",
	}, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err, "connect sqlite test db")

	require.NoError(t,
		db.AutoMigrate(&models.SysDeptLocationAlias{}, &models.Department{}),
		"auto-migrate test tables",
	)

	// 手动建 partial unique index(SQLite 支持 CREATE UNIQUE INDEX ... WHERE ...)
	// AutoMigrate 不支持 partial index 语法,需 Exec 手写。
	// 生产 migration_165 同样手写 partial unique idx_sys_dept_location_alias_dept_scope。
	require.NoError(t, db.Exec(
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_test_alias_dept_scope ON sys_dept_location_alias(dept_id, scope) WHERE deleted_at IS NULL`,
	).Error)

	// 清空数据(避免其它测试文件残留干扰)
	require.NoError(t, db.Where("1=1").Delete(&models.SysDeptLocationAlias{}).Error)
	require.NoError(t, db.Exec("DELETE FROM sys_dept").Error)

	return db
}

// seedDept 插入一条 sys_dept 记录,返回 id + ancestors + isExternalOrg
func seedDept(t *testing.T, db *gorm.DB, name string, isExternalOrg int, ancestors string) string {
	dept := &models.Department{
		DeptName:      name,
		DeptCode:      "code-" + name,
		Ancestors:     ancestors,
		IsExternalOrg: isExternalOrg,
	}
	require.NoError(t, db.Create(dept).Error)
	return dept.ID
}

// TestValidateAlias_SelfMappingRejected 验证三级校验第 1 级:
// dept_id == location_id 直接拒绝("自映射")。
func TestValidateAlias_SelfMappingRejected(t *testing.T) {
	db := setupLocationAliasTestDB(t)
	svc := NewLocationAliasService(db)
	ctx := context.Background()

	// 任意合法 UUID,只要 DeptID == LocationID 即触发
	same := uuid.NewString()
	alias := &models.SysDeptLocationAlias{
		DeptID:     same,
		LocationID: same,
		Scope:      aliasDefaultScope,
	}

	err := svc.(*locationAliasServiceImpl).validateAlias(ctx, alias)
	require.Error(t, err, "自映射必须被拒绝")
	require.Contains(t, err.Error(), "自映射", "错误信息须含 '自映射' 关键字,供前端展示")
}

// TestValidateAlias_LocationNotExternalOrgRejected 验证三级校验第 2 级:
// location 必须是外部机构(is_external_org = 1),否则拒绝。
func TestValidateAlias_LocationNotExternalOrgRejected(t *testing.T) {
	db := setupLocationAliasTestDB(t)
	svc := NewLocationAliasService(db)
	ctx := context.Background()

	// 建一个 location 部门(is_external_org = 0)和一个 dept 部门
	locationID := seedDept(t, db, "loc-非外部", 0, "")
	deptID := seedDept(t, db, "原组织", 0, ","+locationID+",")

	alias := &models.SysDeptLocationAlias{
		DeptID:     deptID,
		LocationID: locationID,
		Scope:      aliasDefaultScope,
	}

	err := svc.(*locationAliasServiceImpl).validateAlias(ctx, alias)
	require.Error(t, err, "is_external_org != 1 必须被拒绝")
	require.Contains(t, err.Error(), "外部机构", "错误信息须含 '外部机构' 关键字,供前端展示")
}

// TestValidateAlias_CrossBranchMappingPasses 验证 Phase 39 核心场景:
// dept 编制上属于另一分支(非 location 后代),物理上挂在 location 办公 → 校验通过。
//
// 历史偏差:原 SPEC REQ-39-02 规则③要求"dept 必须是 location 后代",与功能本质矛盾
// (alias 存在的意义正是 dept 不是 location 后代)。UAT 确认移除规则③后,跨分支映射
// 应当通过(只要 location 是外部机构 + 非自映射)。
func TestValidateAlias_CrossBranchMappingPasses(t *testing.T) {
	db := setupLocationAliasTestDB(t)
	svc := NewLocationAliasService(db)
	ctx := context.Background()

	// location = 外部机构(中心支公司B);dept = 另一分支的子部门(运营服务部/子部门A),
	// 其 ancestors 不含 location_id(编制上不在 location 子树下)。
	locationID := seedDept(t, db, "中心支公司B", 1, "")
	deptID := seedDept(t, db, "运营服务部/子部门A", 0, ",其他上级分支,") // ancestors 不含 location

	alias := &models.SysDeptLocationAlias{
		DeptID:     deptID,
		LocationID: locationID,
		Scope:      aliasDefaultScope,
	}

	err := svc.(*locationAliasServiceImpl).validateAlias(ctx, alias)
	require.NoError(t, err, "跨分支物理位置映射(非后代)应当通过 — 这是 Phase 39 的核心场景")
}

// TestValidateAlias_LocationNotExistRejected 验证 location 不存在时校验失败
// (原 SubstringFalsePositive 用例的前提——ancestor 子串匹配——已随规则③移除,
// 改为守护 location 必须真实存在这一约束)。
func TestValidateAlias_LocationNotExistRejected(t *testing.T) {
	db := setupLocationAliasTestDB(t)
	svc := NewLocationAliasService(db)
	ctx := context.Background()

	deptID := seedDept(t, db, "原组织", 0, "")
	notExistLocation := uuid.NewString() // 不存在的 location_id

	alias := &models.SysDeptLocationAlias{
		DeptID:     deptID,
		LocationID: notExistLocation,
		Scope:      aliasDefaultScope,
	}

	err := svc.(*locationAliasServiceImpl).validateAlias(ctx, alias)
	require.Error(t, err, "location 不存在必须被拒绝")
	require.Contains(t, err.Error(), "物理位置部门不存在")
}

// TestValidateAlias_Pass 完整链路:
// 建 location(外部) + 建 dept(其下子部门) → 校验通过。
func TestValidateAlias_Pass(t *testing.T) {
	db := setupLocationAliasTestDB(t)
	svc := NewLocationAliasService(db)
	ctx := context.Background()

	// 外部机构
	locationID := seedDept(t, db, "物理位置-外部", 1, "")
	// 原组织是 location 的后代(ancestors 含 locationID)
	deptID := seedDept(t, db, "原组织-子部门", 0, ","+locationID+",")

	alias := &models.SysDeptLocationAlias{
		DeptID:     deptID,
		LocationID: locationID,
		Scope:      aliasDefaultScope,
	}

	err := svc.(*locationAliasServiceImpl).validateAlias(ctx, alias)
	require.NoError(t, err, "完整链路: dept 是 location 后代 → 校验通过")
}

// TestLocationAliasService_Create_Pass 端到端 Create:
// scope 空值兜底为 "workstation" + validateAlias 通过 → 落库成功。
func TestLocationAliasService_Create_Pass(t *testing.T) {
	db := setupLocationAliasTestDB(t)
	svc := NewLocationAliasService(db)
	ctx := context.Background()

	locationID := seedDept(t, db, "外部-创建", 1, "")
	deptID := seedDept(t, db, "内部-创建", 0, ","+locationID+",")

	req := &LocationAliasCreateRequest{
		DeptID:     deptID,
		LocationID: locationID,
		// Scope 缺省,期望 service 兜底为 "workstation"
		Remark: "test create",
	}
	alias, err := svc.Create(ctx, req)
	require.NoError(t, err)
	require.NotEmpty(t, alias.ID)
	require.Equal(t, aliasDefaultScope, alias.Scope, "scope 空值兜底为 workstation")
	require.Equal(t, deptID, alias.DeptID)
	require.Equal(t, locationID, alias.LocationID)

	// 落库可查
	var got models.SysDeptLocationAlias
	require.NoError(t, db.Where("id = ?", alias.ID).First(&got).Error)
	require.Equal(t, aliasDefaultScope, got.Scope)
}

// TestLocationAliasService_Create_RejectSelfMapping Create 触发自映射校验。
func TestLocationAliasService_Create_RejectSelfMapping(t *testing.T) {
	db := setupLocationAliasTestDB(t)
	svc := NewLocationAliasService(db)
	ctx := context.Background()

	same := uuid.NewString()
	req := &LocationAliasCreateRequest{DeptID: same, LocationID: same}
	_, err := svc.Create(ctx, req)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "自映射"))
}

// TestLocationAliasService_Create_DuplicateRejectedFriendly 验证 partial unique idx
// (dept_id, scope) 冲突时,Create 返回友好中文(CR-03a:不再透传 PG 索引名
// idx_sys_dept_location_alias_dept_scope 等)。SQLite 走 GORM ErrDuplicatedKey
// 同样覆盖。
func TestLocationAliasService_Create_DuplicateRejectedFriendly(t *testing.T) {
	db := setupLocationAliasTestDB(t)
	svc := NewLocationAliasService(db)
	ctx := context.Background()

	locationID := seedDept(t, db, "外部-去重", 1, "")
	deptID := seedDept(t, db, "内部-去重", 0, ","+locationID+",")

	// 第 1 次创建应成功
	_, err := svc.Create(ctx, &LocationAliasCreateRequest{
		DeptID: deptID, LocationID: locationID, Scope: aliasDefaultScope,
	})
	require.NoError(t, err)

	// 第 2 次同 (dept_id, scope) 必须被拒绝,且错误信息是中文友好提示,
	// 不含 PG 索引名 idx_...
	_, err = svc.Create(ctx, &LocationAliasCreateRequest{
		DeptID: deptID, LocationID: locationID, Scope: aliasDefaultScope,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "该 dept_id + scope 组合已存在", "必须返回友好中文提示")
	require.Contains(t, err.Error(), deptID, "错误信息应包含 dept_id 便于定位")
	require.NotContains(t, err.Error(), "idx_", "不应透传 PG 索引名 idx_...")
}

// TestLocationAliasService_Update_DuplicateRejectedFriendly 验证 Update 触发
// partial unique 冲突时同样返回友好中文(CR-03a)。
func TestLocationAliasService_Update_DuplicateRejectedFriendly(t *testing.T) {
	db := setupLocationAliasTestDB(t)
	svc := NewLocationAliasService(db)
	ctx := context.Background()

	locationID := seedDept(t, db, "外部-更新去重", 1, "")
	deptA := seedDept(t, db, "内部A", 0, ","+locationID+",")
	deptB := seedDept(t, db, "内部B", 0, ","+locationID+",")

	// 创建两条 alias(deptA / deptB, 同 location + scope)
	first, err := svc.Create(ctx, &LocationAliasCreateRequest{
		DeptID: deptA, LocationID: locationID, Scope: aliasDefaultScope,
	})
	require.NoError(t, err)
	_, err = svc.Create(ctx, &LocationAliasCreateRequest{
		DeptID: deptB, LocationID: locationID, Scope: aliasDefaultScope,
	})
	require.NoError(t, err)

	// 把 first 的 dept_id 改成 deptB → 触发 (dept_id, scope) 唯一冲突
	err = svc.Update(ctx, first.ID, &LocationAliasUpdateRequest{DeptID: &deptB})
	require.Error(t, err)
	require.Contains(t, err.Error(), "该 dept_id + scope 组合已存在")
	require.NotContains(t, err.Error(), "idx_")
}

// TestLocationAliasService_Update_ScopeChangedRunsValidation 验证 scope 单字段变更
// 也会触发 validateAlias(CR-03b:之前只 dept/location 变更触发,scope 漏校验)。
//
// 设计:location 部门原本 is_external_org=1(创建时通过校验),创建 alias 后将其
// 降级为 is_external_org=0,然后 Update 仅改 scope → 应触发 validateAlias →
// location 不是外部机构 → 失败。这样不依赖软删除,稳定触发校验失败。
func TestLocationAliasService_Update_ScopeChangedRunsValidation(t *testing.T) {
	db := setupLocationAliasTestDB(t)
	svc := NewLocationAliasService(db)
	ctx := context.Background()

	locationID := seedDept(t, db, "外部-scope改", 1, "")
	deptID := seedDept(t, db, "内部-scope改", 0, ","+locationID+",")

	created, err := svc.Create(ctx, &LocationAliasCreateRequest{
		DeptID: deptID, LocationID: locationID, Scope: aliasDefaultScope,
	})
	require.NoError(t, err)

	// 把 location 降级为非外部机构(模拟数据/权限漂移)
	require.NoError(t, db.Model(&models.Department{}).
		Where("id = ?", locationID).
		Update("is_external_org", 0).Error)

	// 仅改 scope("workstation" → "floor") → 应触发 validateAlias → location
	// 不是外部机构 → 应失败。scope 必须**实际改变**才能触发 scopeChanged 分支
	//("workstation"=="workstation" → scopeChanged=false → validateAlias 不跑)。
	newScope := "floor"
	err = svc.Update(ctx, created.ID, &LocationAliasUpdateRequest{Scope: &newScope})
	require.Error(t, err, "scope 单独变更也必须跑 validateAlias")
	require.Contains(t, err.Error(), "物理位置必须是外部机构")
}

// TestLocationAliasService_List 列表查询 + 分页 + 含 JOIN 部门名。
func TestLocationAliasService_List(t *testing.T) {
	db := setupLocationAliasTestDB(t)
	svc := NewLocationAliasService(db)
	ctx := context.Background()

	locationID := seedDept(t, db, "外部-列表", 1, "")
	// 建 3 个不同的 dept(partial unique (dept_id, scope) 不能三条都用同 dept)
	deptIDs := []string{
		seedDept(t, db, "内部-列表-A", 0, ","+locationID+","),
		seedDept(t, db, "内部-列表-B", 0, ","+locationID+","),
		seedDept(t, db, "内部-列表-C", 0, ","+locationID+","),
	}

	for _, deptID := range deptIDs {
		req := &LocationAliasCreateRequest{
			DeptID:     deptID,
			LocationID: locationID,
			Scope:      aliasDefaultScope,
		}
		_, err := svc.Create(ctx, req)
		require.NoError(t, err)
	}

	result, err := svc.List(ctx, 1, 10)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.EqualValues(t, 3, result.Total)
	require.Equal(t, 1, result.Current)
	require.Equal(t, 10, result.PageSize)

	list, ok := result.List.([]LocationAliasListItem)
	require.True(t, ok)
	require.Len(t, list, 3)
}

// TestLocationAliasService_Update 完整链路更新:
// 只改 remark 不跑 validateAlias;改 location_id 触发 validateAlias。
func TestLocationAliasService_Update(t *testing.T) {
	db := setupLocationAliasTestDB(t)
	svc := NewLocationAliasService(db)
	ctx := context.Background()

	// 两条 location 都建外部机构
	locA := seedDept(t, db, "外部A", 1, "")
	locB := seedDept(t, db, "外部B", 1, "")
	deptID := seedDept(t, db, "原组织-更新", 0, ","+locA+",")

	created, err := svc.Create(ctx, &LocationAliasCreateRequest{
		DeptID:     deptID,
		LocationID: locA,
	})
	require.NoError(t, err)

	// 1. 仅改 remark → 不触发 validateAlias,应成功
	remark := "备注-已更新"
	err = svc.Update(ctx, created.ID, &LocationAliasUpdateRequest{Remark: &remark})
	require.NoError(t, err)

	got, err := svc.GetByID(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, "备注-已更新", got.Remark)

	// 2. 改 location_id 到 locB + 改 dept.ancestors 含 locB → 应成功
	require.NoError(t, db.Model(&models.Department{}).
		Where("id = ?", deptID).
		Update("ancestors", ","+locB+",locX").Error)

	locBCopy := locB
	err = svc.Update(ctx, created.ID, &LocationAliasUpdateRequest{LocationID: &locBCopy})
	require.NoError(t, err)

	got, err = svc.GetByID(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, locB, got.LocationID)

	// 3. 改 location_id 到一个不存在的 uuid → 校验失败
	notExist := uuid.NewString()
	err = svc.Update(ctx, created.ID, &LocationAliasUpdateRequest{LocationID: &notExist})
	require.Error(t, err)
	require.Contains(t, err.Error(), "物理位置部门不存在")
}

// TestLocationAliasService_Delete 软删除后 List/GetByID 不可见。
func TestLocationAliasService_Delete(t *testing.T) {
	db := setupLocationAliasTestDB(t)
	svc := NewLocationAliasService(db)
	ctx := context.Background()

	locationID := seedDept(t, db, "外部-删除", 1, "")
	deptID := seedDept(t, db, "内部-删除", 0, ","+locationID+",")

	created, err := svc.Create(ctx, &LocationAliasCreateRequest{
		DeptID:     deptID,
		LocationID: locationID,
	})
	require.NoError(t, err)

	err = svc.Delete(ctx, created.ID)
	require.NoError(t, err)

	// GetByID 不可见(软删除)
	_, err = svc.GetByID(ctx, created.ID)
	require.Error(t, err, "软删除后 GetByID 应失败")

	// List 不计总数
	result, err := svc.List(ctx, 1, 10)
	require.NoError(t, err)
	require.EqualValues(t, 0, result.Total)
}
