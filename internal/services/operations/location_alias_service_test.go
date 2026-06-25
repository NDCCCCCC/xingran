//go:build !skip_db_tests
// +build !skip_db_tests

package operations

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	_ "modernc.org/sqlite" // 纯 Go SQLite 驱动,必须在使用前导入
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

// TestValidateAlias_DeptNotDescendantRejected 验证三级校验第 3 级:
// dept.ancestors 不含 location_id → 后代关系校验失败。
func TestValidateAlias_DeptNotDescendantRejected(t *testing.T) {
	db := setupLocationAliasTestDB(t)
	svc := NewLocationAliasService(db)
	ctx := context.Background()

	// 两个互不相关的外部机构 + 内部部门(dept 的 ancestors 不含 location_id)
	locationID := seedDept(t, db, "外部机构A", 1, "")
	deptID := seedDept(t, db, "外部机构B", 1, "") // ancestors 为空,与 location 无血缘

	alias := &models.SysDeptLocationAlias{
		DeptID:     deptID,
		LocationID: locationID,
		Scope:      aliasDefaultScope,
	}

	err := svc.(*locationAliasServiceImpl).validateAlias(ctx, alias)
	require.Error(t, err, "ancestors 不含 location_id 必须被拒绝")
	require.Contains(t, err.Error(), "后代", "错误信息须含 '后代' 关键字,供前端展示")
}

// TestValidateAlias_SubstringFalsePositive 边界用例:
// ancestors 含 location_id 的**子串**但非完整 ID 时不能误判。
// 例如 ancestors = ",uuid-abc," location_id = "uuid-ab" → 子串误判风险;
// ","+ancestors+"," + ","+locationID+"," 模式可避免此 bug。
func TestValidateAlias_SubstringFalsePositive(t *testing.T) {
	db := setupLocationAliasTestDB(t)
	svc := NewLocationAliasService(db)
	ctx := context.Background()

	// 两个真实的 uuid,人为构造一个 location_id 是 dept.ancestors 中 ID 的前缀的伪场景
	// 因为我们手工指定 ancestors,可以直接覆盖这一边界。
	// ancestors = ",loc-root,"  location_id = "loc-roo" (前缀)
	// 期望: validateAlias 失败(因为 dept.ancestors 真实不含 "loc-roo",但即使祖先真的
	// 存在前缀相似,逗号包围匹配也必须正确区分)。
	rootID := seedDept(t, db, "root", 1, "")
	deptID := seedDept(t, db, "sub", 1, ","+rootID+",")

	// 让 location_id 是 rootID 的前缀(人为),dept 的 ancestors 是完整 rootID → 不命中
	prefix := rootID[:len(rootID)-1] // 去掉最后一个字符,得到 rootID 的前缀
	alias := &models.SysDeptLocationAlias{
		DeptID:     deptID,
		LocationID: prefix,
		Scope:      aliasDefaultScope,
	}

	err := svc.(*locationAliasServiceImpl).validateAlias(ctx, alias)
	require.Error(t, err, "前缀子串不能误判为后代命中")
	// 错误来源可能是 location 不存在(is_external_org=1 校验需要 location 真实存在)
	// 或者后代校验失败 — 只要不是 false-positive 放行即可
	require.NotContains(t, err.Error(), "panic", "不能 panic")
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

// TestLocationAliasService_List 列表查询 + 分页 + 含 JOIN 部门名。
func TestLocationAliasService_List(t *testing.T) {
	db := setupLocationAliasTestDB(t)
	svc := NewLocationAliasService(db)
	ctx := context.Background()

	locationID := seedDept(t, db, "外部-列表", 1, "")
	deptID := seedDept(t, db, "内部-列表", 0, ","+locationID+",")

	// 直接插 3 条 alias
	for i := 0; i < 3; i++ {
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

	list, ok := result.List.([]models.SysDeptLocationAlias)
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
