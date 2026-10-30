package system

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	pkgcache "github.com/xingran-next/xingran-go-backend/pkg/cache"
)

// =====================================================================
// Phase 74-07: dashboard_service.go 全量测试(0% → 覆盖 CRUD/模板/版本/
// 导入导出/权限矩阵/端点过滤)。sqlite + MemoryCache。
// =====================================================================

func newDashboardTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:dash_"+t.Name()+"?mode=memory&cache=shared&_enable_boolean=true"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.Dashboard{}, &models.DashboardVersion{}, &models.WidgetConfig{}))
	return db
}

func newDashSvc(t *testing.T, mem *pkgcache.MemoryCache) DashboardService {
	t.Helper()
	return NewDashboardService(newDashboardTestDB(t), mem, nil)
}

func sampleLayout() models.LayoutConfig {
	return models.LayoutConfig{
		Columns:   models.Columns{Desktop: 12, Tablet: 8, Mobile: 4},
		RowHeight: 60,
		Margin:    []int{10, 10},
		Widgets: []models.WidgetConfig{{
			ID: "w1", Type: "stat", Title: "总资产",
			Position:   models.WidgetPosition{},
			DataSource: models.DataSourceConfig{},
			Display:    models.DisplayConfig{},
		}},
	}
}

func TestDashboardService_ListAndGet(t *testing.T) {
	mem := pkgcache.NewMemoryCache(50, time.Minute)
	svc := newDashSvc(t, mem)
	ctx := context.Background()

	created, err := svc.CreateDashboard(ctx, "u1", CreateDashboardRequest{
		Name: "运维大盘", Description: "desc", Layout: sampleLayout(), RefreshInterval: 60,
	})
	require.NoError(t, err)
	require.NotEmpty(t, created.ID)

	_, err = svc.CreateDashboard(ctx, "u2", CreateDashboardRequest{
		Name: "监控模板", Layout: sampleLayout(), IsTemplate: true,
	})
	require.NoError(t, err)
	_, err = svc.CreateDashboard(ctx, "u2", CreateDashboardRequest{
		Name: "停用盘", Layout: sampleLayout(),
	})
	require.NoError(t, err)
	// 将第三条置为停用
	stopped := models.DashboardStatusStopped
	var third models.Dashboard
	require.NoError(t, newDashDBOf(t, svc).Where("name = ?", "停用盘").First(&third).Error)
	require.NoError(t, svc.UpdateDashboard(ctx, "u2", third.ID, UpdateDashboardRequest{Status: &stopped}))

	// 列表:keyword / isTemplate / status 过滤 + 分页
	page, err := svc.GetDashboards(ctx, DashboardListParams{Current: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(3), page.Total)
	page, err = svc.GetDashboards(ctx, DashboardListParams{Current: 1, PageSize: 10, Keyword: "大盘"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), page.Total)
	page, err = svc.GetDashboards(ctx, DashboardListParams{Current: 1, PageSize: 10, IsTemplate: true})
	require.NoError(t, err)
	assert.Equal(t, int64(1), page.Total)
	page, err = svc.GetDashboards(ctx, DashboardListParams{Current: 1, PageSize: 10, Status: &stopped})
	require.NoError(t, err)
	assert.Equal(t, int64(1), page.Total)
	page, err = svc.GetDashboards(ctx, DashboardListParams{Current: 2, PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, int64(1), int64(len(page.List)), "第 2 页余 1 条")

	// 详情:未命中 → DashboardNotFound;命中 → 写缓存
	_, err = svc.GetDashboard(ctx, "ghost")
	require.Error(t, err)
	got, err := svc.GetDashboard(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, "运维大盘", got.Name)
	keys, _ := mem.Keys(ctx, "dashboard:*")
	assert.NotEmpty(t, keys, "详情查询应写缓存")
}

// newDashDBOf 从服务实现取回底层 db(断言库内状态用)。
func newDashDBOf(t *testing.T, svc DashboardService) *gorm.DB {
	t.Helper()
	return svc.(*DashboardServiceImpl).db
}

func TestDashboardService_UpdateDeleteDuplicateDefault(t *testing.T) {
	svc := newDashSvc(t, pkgcache.NewMemoryCache(50, time.Minute))
	ctx := context.Background()

	d1, err := svc.CreateDashboard(ctx, "u1", CreateDashboardRequest{Name: "盘1", Layout: sampleLayout()})
	require.NoError(t, err)
	d2, err := svc.CreateDashboard(ctx, "u1", CreateDashboardRequest{Name: "盘2", Layout: sampleLayout()})
	require.NoError(t, err)

	// 非 owner 修改 → PermissionDenied
	err = svc.UpdateDashboard(ctx, "u2", d1.ID, UpdateDashboardRequest{})
	require.Error(t, err)

	// Update 全字段
	newName := "盘1改"
	newDesc := "新描述"
	interval := 120
	status := models.DashboardStatusStopped
	require.NoError(t, svc.UpdateDashboard(ctx, "u1", d1.ID, UpdateDashboardRequest{
		Name: &newName, Description: &newDesc, RefreshInterval: &interval, Status: &status,
	}))
	got, _ := svc.GetDashboard(ctx, d1.ID)
	assert.Equal(t, "盘1改", got.Name)
	assert.Equal(t, 120, got.RefreshInterval)

	// 非存在 ID → DashboardNotFound
	require.Error(t, svc.UpdateDashboard(ctx, "u1", "ghost", UpdateDashboardRequest{}))
	require.Error(t, svc.DeleteDashboard(ctx, "u1", "ghost"))

	// Duplicate:副本归当前用户
	dup, err := svc.DuplicateDashboard(ctx, "u2", d1.ID)
	require.NoError(t, err)
	assert.Equal(t, "盘1改 (副本)", dup.Name)
	assert.Equal(t, "u2", dup.OwnerID)
	assert.False(t, dup.IsTemplate)
	_, err = svc.DuplicateDashboard(ctx, "u1", "ghost")
	require.Error(t, err)

	// SetDefault:先默认 d1 再切 d2 → d1 被清
	require.NoError(t, svc.SetDefaultDashboard(ctx, "u1", d1.ID))
	require.NoError(t, svc.SetDefaultDashboard(ctx, "u1", d2.ID))
	var d1Row models.Dashboard
	require.NoError(t, newDashDBOf(t, svc).Where("id = ?", d1.ID).First(&d1Row).Error)
	assert.False(t, d1Row.IsDefault)
	require.Error(t, svc.SetDefaultDashboard(ctx, "u1", "ghost"))
	// 模板对任何 owner 可操作(IsTemplate 豁免所有权检查)
	tpl, err := svc.CreateDashboard(ctx, "u9", CreateDashboardRequest{Name: "T", Layout: sampleLayout(), IsTemplate: true})
	require.NoError(t, err)
	require.NoError(t, svc.SetDefaultDashboard(ctx, "u1", tpl.ID))

	// Delete
	require.NoError(t, svc.DeleteDashboard(ctx, "u1", d2.ID))
	_, err = svc.GetDashboard(ctx, d2.ID)
	require.Error(t, err)
}

func TestDashboardService_TemplatesVersionsImportExport(t *testing.T) {
	svc := newDashSvc(t, pkgcache.NewMemoryCache(50, time.Minute))
	ctx := context.Background()

	// 模板:global/dept/personal 各一 + 非模板一
	deptScope := models.TemplateScopeDept
	personalScope := models.TemplateScopePersonal
	mkTpl := func(name string, scope models.TemplateScope) *models.Dashboard {
		d, err := svc.CreateDashboard(ctx, "admin", CreateDashboardRequest{
			Name: name, Layout: sampleLayout(), IsTemplate: true, TemplateScope: scope,
		})
		require.NoError(t, err)
		return d
	}
	globalTpl := mkTpl("全局模板", models.TemplateScopeGlobal)
	deptTpl := mkTpl("部门模板", deptScope)
	mkTpl("个人模板", personalScope)
	plain, err := svc.CreateDashboard(ctx, "admin", CreateDashboardRequest{Name: "非模板", Layout: sampleLayout()})
	require.NoError(t, err)

	// nil scope → 仅 global(F-21);dept scope → dept+global;无效 → 仅 global
	tpls, err := svc.GetTemplates(ctx, nil)
	require.NoError(t, err)
	require.Len(t, tpls, 1)
	assert.Equal(t, "全局模板", tpls[0].Name)
	tpls, err = svc.GetTemplates(ctx, strPtrStd("dept"))
	require.NoError(t, err)
	assert.Len(t, tpls, 2)
	tpls, err = svc.GetTemplates(ctx, strPtrStd("bogus"))
	require.NoError(t, err)
	assert.Len(t, tpls, 1)

	// CreateFromTemplate:非模板 → 错;命中 → 复制 layout
	_, err = svc.CreateFromTemplate(ctx, "u1", plain.ID, "我的盘")
	require.ErrorContains(t, err, "not a template")
	from, err := svc.CreateFromTemplate(ctx, "u1", deptTpl.ID, "我的盘")
	require.NoError(t, err)
	assert.Equal(t, "我的盘", from.Name)
	assert.False(t, from.IsTemplate)
	assert.Equal(t, len(globalTpl.Layout.Widgets), len(from.Layout.Widgets))
	_, err = svc.CreateFromTemplate(ctx, "u1", "ghost", "x")
	require.Error(t, err)

	// 版本:创建 / 列表 / 恢复 / 未命中
	v1, err := svc.CreateVersion(ctx, "u1", from.ID, "初版")
	require.NoError(t, err)
	require.NotEmpty(t, v1.ID)
	versions, err := svc.GetVersions(ctx, from.ID)
	require.NoError(t, err)
	assert.Len(t, versions, 1)
	require.NoError(t, svc.RestoreVersion(ctx, "u1", from.ID, v1.ID))
	require.ErrorContains(t, svc.RestoreVersion(ctx, "u1", from.ID, "ghost"), "version not found")
	_, err = svc.CreateVersion(ctx, "u1", "ghost", "c")
	require.Error(t, err)

	// 导出/导入 roundtrip
	exported, err := svc.ExportDashboard(ctx, from.ID)
	require.NoError(t, err)
	assert.Contains(t, exported, "我的盘")
	imported, err := svc.ImportDashboard(ctx, "u2", exported)
	require.NoError(t, err)
	assert.Equal(t, "u2", imported.OwnerID)
	assert.NotEqual(t, from.ID, imported.ID)
	_, err = svc.ImportDashboard(ctx, "u2", "not-json")
	require.Error(t, err)
	_, err = svc.ExportDashboard(ctx, "ghost")
	require.Error(t, err)
}

func strPtrStd(s string) *string { return &s }

func TestDashboardService_CheckAccessMatrix(t *testing.T) {
	svc := newDashSvc(t, pkgcache.NewMemoryCache(10, time.Minute)).(*DashboardServiceImpl)
	deptID := "dept-1"
	otherDept := "dept-2"

	owner := &models.Dashboard{OwnerID: "u1", Scope: models.DashboardScopePrivate}
	assert.True(t, svc.CheckAccess(owner, &AccessContext{UserID: "u1"}), "创建者恒可访问")
	assert.False(t, svc.CheckAccess(owner, &AccessContext{UserID: "u2"}), "私有盘他人不可访问")

	global := &models.Dashboard{OwnerID: "u1", Scope: models.DashboardScopeGlobal}
	assert.True(t, svc.CheckAccess(global, &AccessContext{UserID: "anyone"}))

	dept := &models.Dashboard{OwnerID: "u1", Scope: models.DashboardScopeDept, DeptID: &deptID}
	assert.True(t, svc.CheckAccess(dept, &AccessContext{UserID: "u2", DataScope: "all"}), "all 数据范围看所有部门盘")
	assert.True(t, svc.CheckAccess(dept, &AccessContext{UserID: "u2", DataScope: "custom"}))
	assert.True(t, svc.CheckAccess(dept, &AccessContext{UserID: "u2", UserDeptID: deptID}), "本部门可见")
	assert.False(t, svc.CheckAccess(dept, &AccessContext{UserID: "u2", UserDeptID: otherDept}), "他部门不可见")

	noDept := &models.Dashboard{OwnerID: "u1", Scope: models.DashboardScopeDept}
	assert.False(t, svc.CheckAccess(noDept, &AccessContext{UserID: "u2", UserDeptID: deptID}), "无部门关联的部门盘不可见")
}

func TestDashboardService_AccessibleListAndDefault(t *testing.T) {
	svc := newDashSvc(t, pkgcache.NewMemoryCache(10, time.Minute))
	ctx := context.Background()
	deptID := "dept-1"
	otherDept := "dept-2"

	// u1 私有盘 / 全局盘 / 本部门盘 / 他部门盘 + u2 私有盘
	_, err := svc.CreateDashboardWithPermissions(ctx, "u1", deptID, "dept", false, CreateDashboardRequest{
		Name: "u1私有", Layout: sampleLayout(),
	})
	require.NoError(t, err)
	_, err = svc.CreateDashboardWithPermissions(ctx, "u1", deptID, "all", true, CreateDashboardRequest{
		Name: "全局盘", Layout: sampleLayout(), Scope: models.DashboardScopeGlobal,
	})
	require.NoError(t, err)
	_, err = svc.CreateDashboardWithPermissions(ctx, "u1", deptID, "dept", false, CreateDashboardRequest{
		Name: "本部门盘", Layout: sampleLayout(), Scope: models.DashboardScopeDept, DeptID: &deptID,
	})
	require.NoError(t, err)
	_, err = svc.CreateDashboardWithPermissions(ctx, "u1", deptID, "all", true, CreateDashboardRequest{
		Name: "他部门盘", Layout: sampleLayout(), Scope: models.DashboardScopeDept, DeptID: &otherDept,
	})
	require.NoError(t, err)
	_, err = svc.CreateDashboardWithPermissions(ctx, "u2", otherDept, "dept", false, CreateDashboardRequest{
		Name: "u2私有", Layout: sampleLayout(),
	})
	require.NoError(t, err)

	// dataScope=dept 的 u2:自己 1 + 全局 1 + 本部门盘(dept-2)1 → 3
	page, err := svc.GetAccessibleDashboards(ctx, DashboardListParams{Current: 1, PageSize: 10}, "u2", otherDept, "dept")
	require.NoError(t, err)
	assert.Equal(t, int64(3), page.Total)
	// dataScope=all 的 u2:自己 + 全局 + 部门盘 2 → 4
	page, err = svc.GetAccessibleDashboards(ctx, DashboardListParams{Current: 1, PageSize: 10}, "u2", otherDept, "all")
	require.NoError(t, err)
	assert.Equal(t, int64(4), page.Total)
	// keyword
	page, err = svc.GetAccessibleDashboards(ctx, DashboardListParams{Current: 1, PageSize: 10, Keyword: "全局"}, "u2", otherDept, "all")
	require.NoError(t, err)
	assert.Equal(t, int64(1), page.Total)

	// 默认盘优先级:无默认+无系统盘 → nil
	none, err := svc.GetAccessibleDefaultDashboard(ctx, "u2", otherDept, "dept")
	require.NoError(t, err)
	assert.Nil(t, none)

	// u2 设默认自己的盘 → 命中
	var u2Dash models.Dashboard
	require.NoError(t, newDashDBOf(t, svc).Where("name = ?", "u2私有").First(&u2Dash).Error)
	require.NoError(t, svc.SetDefaultDashboard(ctx, "u2", u2Dash.ID))
	def, err := svc.GetAccessibleDefaultDashboard(ctx, "u2", otherDept, "dept")
	require.NoError(t, err)
	require.NotNil(t, def)
	assert.Equal(t, u2Dash.ID, def.ID)

	// 无用户默认但有系统全局盘 → 系统盘兜底
	sysDash, err := svc.CreateDashboardWithPermissions(ctx, "admin", deptID, "all", true, CreateDashboardRequest{
		Name: "系统盘", Layout: sampleLayout(), Scope: models.DashboardScopeGlobal, IsSystem: true,
	})
	require.NoError(t, err)
	def, err = svc.GetAccessibleDefaultDashboard(ctx, "u3", otherDept, "dept")
	require.NoError(t, err)
	require.NotNil(t, def)
	assert.Equal(t, sysDash.ID, def.ID)
}

func TestDashboardService_CreateWithPermissionsGuards(t *testing.T) {
	svc := newDashSvc(t, pkgcache.NewMemoryCache(10, time.Minute))
	ctx := context.Background()
	deptID := "dept-1"
	otherDept := "dept-2"

	// 部门盘必须指定部门
	_, err := svc.CreateDashboardWithPermissions(ctx, "u1", deptID, "dept", false, CreateDashboardRequest{
		Name: "x", Layout: sampleLayout(), Scope: models.DashboardScopeDept,
	})
	require.ErrorContains(t, err, "必须指定部门")

	// 非管理员且数据范围受限 → 不能选他部门
	_, err = svc.CreateDashboardWithPermissions(ctx, "u1", deptID, "dept", false, CreateDashboardRequest{
		Name: "x", Layout: sampleLayout(), Scope: models.DashboardScopeDept, DeptID: &otherDept,
	})
	require.ErrorContains(t, err, "无权限选择该部门")

	// 受限但选本部门 → 放行
	_, err = svc.CreateDashboardWithPermissions(ctx, "u1", deptID, "dept", false, CreateDashboardRequest{
		Name: "x", Layout: sampleLayout(), Scope: models.DashboardScopeDept, DeptID: &deptID,
	})
	require.NoError(t, err)

	// 全局盘/系统盘仅管理员
	_, err = svc.CreateDashboardWithPermissions(ctx, "u1", deptID, "all", false, CreateDashboardRequest{
		Name: "x", Layout: sampleLayout(), Scope: models.DashboardScopeGlobal,
	})
	require.ErrorContains(t, err, "仅管理员可创建全局仪表盘")
	_, err = svc.CreateDashboardWithPermissions(ctx, "u1", deptID, "all", false, CreateDashboardRequest{
		Name: "x", Layout: sampleLayout(), IsSystem: true,
	})
	require.ErrorContains(t, err, "仅管理员可创建系统仪表盘")

	// 管理员建全局系统盘成功(scope 空 → 默认 private)
	made, err := svc.CreateDashboardWithPermissions(ctx, "u1", deptID, "all", true, CreateDashboardRequest{
		Name: "adm", Layout: sampleLayout(), Scope: models.DashboardScopeGlobal, IsSystem: true,
	})
	require.NoError(t, err)
	assert.Equal(t, models.DashboardScopeGlobal, made.Scope)
	private, err := svc.CreateDashboardWithPermissions(ctx, "u1", deptID, "dept", false, CreateDashboardRequest{
		Name: "pv", Layout: sampleLayout(),
	})
	require.NoError(t, err)
	assert.Equal(t, models.DashboardScopePrivate, private.Scope, "空 scope 默认 private")
}

func TestDashboardService_WidgetAndEndpoints(t *testing.T) {
	mem := pkgcache.NewMemoryCache(10, time.Minute)
	svc := newDashSvc(t, mem)
	ctx := context.Background()

	// GetWidgetData:widget 不存在。
	// QUIRK(D-12 记录不修): WidgetConfig.Position(WidgetPosition)既无
	// driver.Valuer 也无 sql.Scanner → GORM 对该列的写入/读取都会报
	// "unsupported type/Scan",即 GetWidgetData 的 happy path 在任何方言下
	// 都无法走到 user_id 校验之后 —— 生产上该表从未经模型成功读写。
	_, err := svc.GetWidgetData(ctx, "ghost-widget", "/x", nil)
	require.ErrorContains(t, err, "widget not found")

	// GetBatchWidgetData:ctx 无 user_id
	_, err = svc.GetBatchWidgetData(ctx, []string{"w-real"}, false)
	require.ErrorContains(t, err, "user not authenticated")

	// endpointService 为 nil 的分支
	eps, err := svc.GetUserAccessibleEndpoints(ctx, "u1")
	require.NoError(t, err)
	assert.Empty(t, eps)
	_, err = svc.ValidateEndpoint("/x", "GET")
	require.Error(t, err)
	svc.InvalidateUserCache(ctx, "u1") // no-panic

	// FilterEndpointsByWidgetType:命中保留 / 未命中剔除 / 空分类剔除
	categories := []services.CategoryEndpoints{{
		Module: "ops", Category: "资产", Icon: "i",
		Endpoints: []services.EndpointDetail{
			{Route: "/a", Method: "GET", SupportedWidgets: []string{"stat", "chart"}},
			{Route: "/b", Method: "GET", SupportedWidgets: []string{"table"}},
		},
	}, {
		Module: "sys", Category: "用户", Icon: "i",
		Endpoints: []services.EndpointDetail{
			{Route: "/c", Method: "GET", SupportedWidgets: []string{"table"}},
		},
	}}
	filtered := svc.FilterEndpointsByWidgetType(categories, "stat")
	require.Len(t, filtered, 1)
	assert.Equal(t, "ops", filtered[0].Module)
	require.Len(t, filtered[0].Endpoints, 1)
	assert.Equal(t, "/a", filtered[0].Endpoints[0].Route)
	assert.Empty(t, svc.FilterEndpointsByWidgetType(categories, "none"))
}

// Dashboard JSON roundtrip(Layout 序列化)冒烟。
func TestDashboardLayoutJSONRoundtrip(t *testing.T) {
	layout := sampleLayout()
	data, err := json.Marshal(layout)
	require.NoError(t, err)
	var back models.LayoutConfig
	require.NoError(t, json.Unmarshal(data, &back))
	assert.Equal(t, len(layout.Widgets), len(back.Widgets))
	assert.Equal(t, layout.Columns.Desktop, back.Columns.Desktop)
}
