package operations

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"

	"github.com/xingran-next/xingran-go-backend/internal/api/v1/operations/requests"
	sysmodels "github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	operationsmodels "github.com/xingran-next/xingran-go-backend/internal/models/operations"
)

// =====================================================================
// Phase 74-07: asset_service / building_service(+typesafe) /
// floor_plan_text_service / room_device_service 测试。
// =====================================================================

const (
	testUUIDDept = "11111111-1111-1111-1111-111111111111"
	testUUIDUser = "22222222-2222-2222-2222-222222222222"
	testUUIDSub  = "33333333-3333-3333-3333-333333333333"
)

// newAssetTestDB 资产库：operations 全家族 + ops_asset + 最小 sys_user。
// Asset 模型带 default:gen_random_uuid()，sqlite DDL 不支持函数默认值 →
// 不能 AutoMigrate，改为 schema.Parse 动态取全列名建 TEXT 表。
func newAssetTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := newPhotoTestDB(t)

	assetSchema, err := schema.Parse(&sysmodels.Asset{}, &sync.Map{}, db.NamingStrategy)
	require.NoError(t, err)
	var ddl strings.Builder
	ddl.WriteString("CREATE TABLE ops_asset (id TEXT PRIMARY KEY")
	timeCols := map[string]bool{"created_at": true, "updated_at": true, "deleted_at": true}
	for _, name := range assetSchema.DBNames {
		if name == "id" {
			continue
		}
		if timeCols[name] {
			ddl.WriteString(", " + name + " DATETIME")
		} else {
			ddl.WriteString(", " + name + " TEXT")
		}
	}
	ddl.WriteString(")")
	require.NoError(t, db.Exec(ddl.String()).Error)

	require.NoError(t, db.Exec(`CREATE TABLE sys_user (id TEXT PRIMARY KEY, deleted_at DATETIME)`).Error)
	return db
}

func seedAssetRefs(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.Exec(
		`INSERT INTO sys_dept (id, dept_name, dept_code, ancestors, status) VALUES (?, '总部', 'D1', '', 0)`,
		testUUIDDept).Error)
	require.NoError(t, db.Exec(
		`INSERT INTO sys_dept (id, dept_name, dept_code, ancestors, status) VALUES (?, '子部', 'D2', ?, 0)`,
		testUUIDSub, testUUIDDept).Error)
	require.NoError(t, db.Exec(
		`INSERT INTO sys_user (id) VALUES (?)`, testUUIDUser).Error)
}

func TestAssetService_CRUDAndValidators(t *testing.T) {
	db := newAssetTestDB(t)
	seedAssetRefs(t, db)
	svc := NewAssetService(db)
	ctx := context.Background()

	badDept := "not-uuid"
	subDept := testUUIDSub

	// 校验器：非 UUID 部门 / 不存在部门 / 不存在用户
	a := &sysmodels.Asset{DeviceSN: "SN-1", DeptID: &badDept}
	err := svc.Create(ctx, a)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "必须是有效的UUID格式")

	a = &sysmodels.Asset{DeviceSN: "SN-1", DeptID: &subDept}
	err = svc.Create(ctx, a) // testUUIDSub 是合法 UUID 且已插入 sys_dept → 通过
	require.NoError(t, err)

	ghost := "99999999-9999-9999-9999-999999999999"
	err = svc.Create(ctx, &sysmodels.Asset{DeviceSN: "SN-2", UserID: &ghost})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "所属用户不存在")

	// 序列号唯一
	err = svc.Create(ctx, &sysmodels.Asset{DeviceSN: "SN-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "设备序列号已存在")

	// Update：改自身 SN 不冲突；SN 撞他人仍拦截
	got, err := svc.GetByID(ctx, a.ID)
	require.NoError(t, err)
	require.NoError(t, svc.Update(ctx, got))
	err = svc.Update(ctx, &sysmodels.Asset{ID: "new-id", DeviceSN: "SN-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "设备序列号已存在")

	// GetByDeviceSN：命中 / 未命中返回 nil / 空参数
	hit, err := svc.GetByDeviceSN(ctx, "SN-1")
	require.NoError(t, err)
	require.NotNil(t, hit)
	miss, err := svc.GetByDeviceSN(ctx, "NOPE")
	require.NoError(t, err)
	assert.Nil(t, miss)
	_, err = svc.GetByDeviceSN(ctx, "")
	require.Error(t, err)

	// List：deptId 含子部门（ancestors）+ applyDeptFilter 无命中 → 空
	require.NoError(t, db.Create(&sysmodels.Asset{DeviceSN: "SN-2", Status: 1}).Error)
	page, err := svc.List(ctx, map[string]interface{}{"deptId": testUUIDDept})
	require.NoError(t, err)
	assert.Equal(t, int64(1), page.Total)

	page, err = svc.List(ctx, map[string]interface{}{"deptId": "88888888-8888-8888-8888-888888888888"})
	require.NoError(t, err)
	assert.Zero(t, page.Total)

	// 下拉数据源
	require.NoError(t, db.Exec(
		`UPDATE ops_asset SET device_type_name='交换机', device_category_second_name='网络', usestatus_label='在用' WHERE devicesn='SN-1'`).Error)
	types, err := svc.GetDeviceTypes(ctx)
	require.NoError(t, err)
	require.Len(t, types, 1)
	assert.Equal(t, "交换机", types[0].Value)
	cats, err := svc.GetDeviceCategories(ctx)
	require.NoError(t, err)
	assert.Equal(t, "网络", cats[0].Value)
	statuses, err := svc.GetStatusValues(ctx)
	require.NoError(t, err)
	assert.Equal(t, "在用", statuses[0].Value)

	// Delete / BatchDelete
	require.NoError(t, svc.Delete(ctx, a.ID))
	require.NoError(t, svc.BatchDelete(ctx, []string{"none"}))
	_, err = svc.GetByID(ctx, a.ID)
	require.Error(t, err)
}

func TestBuildingService_CRUDAndValidate(t *testing.T) {
	db := newAssetTestDB(t)
	seedAssetRefs(t, db)
	svc := NewBuildingService(db)
	ctx := context.Background()

	// 非法 org UUID / 不存在 org
	require.Error(t, svc.Create(ctx, &operationsmodels.OpsBuilding{Name: "B1", OrgID: "d1"}))
	ghost := "99999999-9999-9999-9999-999999999999"
	require.Error(t, svc.Create(ctx, &operationsmodels.OpsBuilding{Name: "B1", OrgID: ghost}))

	b1 := &operationsmodels.OpsBuilding{Name: "B1", OrgID: testUUIDDept, Level: 2, Address: "addr"}
	require.NoError(t, svc.Create(ctx, b1))
	got, err := svc.GetByID(ctx, b1.ID)
	require.NoError(t, err)
	assert.Equal(t, "B1", got.Name)

	// 同机构重名 → 拦截；换机构放行
	require.Error(t, svc.Create(ctx, &operationsmodels.OpsBuilding{Name: "B1", OrgID: testUUIDDept}))
	require.NoError(t, svc.Create(ctx, &operationsmodels.OpsBuilding{Name: "B1", OrgID: testUUIDSub, Level: 1}))

	// Update：自身改名不冲突
	got.Address = "addr2"
	require.NoError(t, svc.Update(ctx, got))
	got, _ = svc.GetByID(ctx, b1.ID)
	assert.Equal(t, "addr2", got.Address)

	// List：name 模糊 + orgId 部门子树 + status
	page, err := svc.List(ctx, map[string]interface{}{"orgId": testUUIDDept})
	require.NoError(t, err)
	assert.Equal(t, int64(2), page.Total, "orgId 命中含子部门楼宇")
	page, err = svc.List(ctx, map[string]interface{}{"name": "B1", "status": 0})
	require.NoError(t, err)
	assert.Equal(t, int64(2), page.Total)

	// SearchBuildingOptions：name + orgId 无命中空集
	opts, err := svc.SearchBuildingOptions(ctx, map[string]interface{}{"name": "B1"})
	require.NoError(t, err)
	assert.Len(t, opts, 2)
	opts, err = svc.SearchBuildingOptions(ctx, map[string]interface{}{"orgId": "77777777-7777-7777-7777-777777777777"})
	require.NoError(t, err)
	assert.Empty(t, opts)

	// BatchDelete / Delete
	require.NoError(t, svc.BatchDelete(ctx, []string{b1.ID}))
	require.NoError(t, svc.Delete(ctx, b1.ID))
	_, err = svc.GetByID(ctx, b1.ID)
	require.Error(t, err)
}

func TestBuildingService_TypeSafe(t *testing.T) {
	db := newAssetTestDB(t)
	seedAssetRefs(t, db)
	svc := NewBuildingServiceTypeSafe(db)
	ctx := context.Background()

	b1 := &operationsmodels.OpsBuilding{Name: "TS-1", OrgID: testUUIDDept}
	require.NoError(t, svc.Create(ctx, b1))
	require.NoError(t, svc.Create(ctx, &operationsmodels.OpsBuilding{Name: "TS-2", OrgID: testUUIDSub, Status: 1}))

	got, err := svc.GetByID(ctx, b1.ID)
	require.NoError(t, err)
	assert.Equal(t, "TS-1", got.Name)

	// 类型安全 List：name + orgId 子树 + status
	page, err := svc.List(ctx, requests.BuildingListRequest{OrgID: testUUIDDept})
	require.NoError(t, err)
	assert.Equal(t, int64(2), page.Total)
	page, err = svc.List(ctx, requests.BuildingListRequest{Name: "TS-1"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), page.Total)
	stopped := 1
	page, err = svc.List(ctx, requests.BuildingListRequest{StatusRequest: requests.StatusRequest{Status: &stopped}})
	require.NoError(t, err)
	assert.Equal(t, int64(1), page.Total)

	// Update / Delete / BatchDelete
	got.Name = "TS-1改"
	require.NoError(t, svc.Update(ctx, got))
	require.NoError(t, svc.Delete(ctx, b1.ID))
	require.NoError(t, svc.BatchDelete(ctx, []string{b1.ID}))
}

func TestFloorPlanTextService_CRUD(t *testing.T) {
	db := newPhotoTestDB(t)
	require.NoError(t, db.AutoMigrate(&operationsmodels.FloorPlanText{}))
	_, floorID := seedBuildingFloor(t, db, "fpt-b")
	svc := NewFloorPlanTextService(db)
	ctx := context.Background()

	// 楼层不存在
	require.Error(t, svc.Create(ctx, &operationsmodels.FloorPlanText{FloorID: "missing", Content: "c", Position: "{}"}))

	txt := &operationsmodels.FloorPlanText{FloorID: floorID, Content: "标签A", Position: `{"x":1}`}
	require.NoError(t, svc.Create(ctx, txt))
	got, err := svc.GetByID(ctx, txt.ID)
	require.NoError(t, err)
	assert.Equal(t, "标签A", got.Content)

	require.NoError(t, svc.Create(ctx, &operationsmodels.FloorPlanText{FloorID: floorID, Content: "标签B", Position: "{}"}))

	// List：floorID + content 模糊 + 排序白名单
	page, err := svc.List(ctx, requests.FloorPlanTextListRequest{FloorID: floorID})
	require.NoError(t, err)
	assert.Equal(t, int64(2), page.Total)
	page, err = svc.List(ctx, requests.FloorPlanTextListRequest{Content: "标签A"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), page.Total)
	asc := true
	page, err = svc.List(ctx, requests.FloorPlanTextListRequest{
		PaginationParams: requests.PaginationParams{BaseListRequest: base.BaseListRequest{
			OrderByColumn: "createdAt", IsAsc: &asc,
		}},
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), page.Total)

	// Update / BatchDelete（空 no-op）/ Delete
	got.Content = "标签A改"
	require.NoError(t, svc.Update(ctx, got))
	require.NoError(t, svc.BatchDelete(ctx, nil))
	require.NoError(t, svc.Delete(ctx, txt.ID))
	_, err = svc.GetByID(ctx, txt.ID)
	require.Error(t, err)
}

func TestRoomDeviceService_CRUD(t *testing.T) {
	db := newPhotoTestDB(t)
	require.NoError(t, db.AutoMigrate(&operationsmodels.OpsRoomDevice{}))
	roomID := seedRoom(t, db)
	svc := NewRoomDeviceService(db)
	ctx := context.Background()

	// 机房不存在
	require.Error(t, svc.Create(ctx, &operationsmodels.OpsRoomDevice{Name: "d", DeviceCode: "C1", DeviceType: "switch", RoomID: "missing"}))

	dev := &operationsmodels.OpsRoomDevice{Name: "核心交换", DeviceCode: "C1", DeviceType: "switch", RoomID: roomID}
	require.NoError(t, svc.Create(ctx, dev))
	got, err := svc.GetByID(ctx, dev.ID)
	require.NoError(t, err)
	require.NotNil(t, got.RoomName)
	assert.Equal(t, "photo-room", *got.RoomName, "GetByID JOIN 机房名")

	// 设备编码唯一
	require.Error(t, svc.Create(ctx, &operationsmodels.OpsRoomDevice{Name: "d2", DeviceCode: "C1", DeviceType: "switch", RoomID: roomID}))

	require.NoError(t, svc.Create(ctx, &operationsmodels.OpsRoomDevice{
		Name: "防火墙", DeviceCode: "C2", DeviceType: "firewall", RoomID: roomID, Status: 2,
	}))

	// List 过滤矩阵
	page, err := svc.List(ctx, requests.RoomDeviceListRequest{Name: "核心"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), page.Total)
	page, err = svc.List(ctx, requests.RoomDeviceListRequest{DeviceType: "firewall"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), page.Total)
	page, err = svc.List(ctx, requests.RoomDeviceListRequest{RoomID: roomID})
	require.NoError(t, err)
	assert.Equal(t, int64(2), page.Total)
	scrapped := 2
	page, err = svc.List(ctx, requests.RoomDeviceListRequest{StatusRequest: requests.StatusRequest{Status: &scrapped}})
	require.NoError(t, err)
	assert.Equal(t, int64(1), page.Total)
	page, err = svc.List(ctx, requests.RoomDeviceListRequest{OrgID: testUUIDDept})
	require.NoError(t, err)
	assert.Zero(t, page.Total, "orgId 未命中楼宇 → 空")

	// Statistics
	stats, err := svc.Statistics(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), stats.Total)
	assert.Equal(t, int64(1), stats.Scrapped)

	// Update / Delete / BatchDelete
	got.Status = 1
	require.NoError(t, svc.Update(ctx, got))
	require.NoError(t, svc.Delete(ctx, dev.ID))
	require.NoError(t, svc.BatchDelete(ctx, []string{dev.ID}))
	_, err = svc.GetByID(ctx, dev.ID)
	require.Error(t, err)
}
