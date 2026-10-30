package services

// =====================================================================
// Phase 79-04 Task 3: network_device CRUD + nil 依赖注入 + 统计 + getLastIPOctet
//
// 覆盖目标: network_device_service.go 0% → ≥70%(基线 202 stmts 全 unc,
// 79-RESEARCH §2)。
//
// 纪律(79-01 SUMMARY 手注沿用):helper 名带 7904 后缀、sqlite t.TempDir 文件库、
// 禁 t.Parallel、状态断言一律引用 models.DeviceStatus* 具名常量(Phase 69-03 判定)。
//
// quirk 锁定(Phase 73-03 记录,本 plan 复述,SUMMARY 复记):
//   Q3 network GetByID 关联名丢失 —— GetByID(:375)把 loadAssociations 结果写进
//      一次性切片 `&[]models.NetworkDevice{device}`,DeptName/CredentialName 永远
//      不会回填到返回值;List(:95)在真实切片上加载, enrichment 正常。按现行为断言。
//   Q4 GORM 零值跳过 —— DeviceStatusOnline=0 是零值,直插会被列 default:2 覆盖,
//      种子行建后必须显式回写 status(73-03 "seedDevice forces status column" 同款)。
//
// nil 依赖边界(research §3):QuickCreateDevice 的探测(SNMP wire)与成功路径的
// Enqueue(deviceInfoCollectionSvc 的 :346 nil-guard)在本 plan 不可达 —— SNMP fake
// 归 79-06(DQ5)。可达面 = 探测前的全部校验分支 + 探测失败阻止创建分支。
// =====================================================================

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
)

const ndv7904Operator = "operator-7904"

// newNdv7904 装配 NetworkDeviceService + sqlite(t.TempDir 文件库),
// discovery/deviceInfoCollection 两依赖以 nil 注入(CRUD 面不触达,触达处即错误分支)。
func newNdv7904(t *testing.T) (*NetworkDeviceService, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "ndv7904.db")), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err, "open sqlite temp db")
	if sqlDB, err := db.DB(); err == nil {
		t.Cleanup(func() { _ = sqlDB.Close() })
	}
	require.NoError(t, db.AutoMigrate(
		&models.NetworkDevice{},
		&models.AuthCredential{},
		&models.Department{},
	), "auto migrate network device chain models")
	return NewNetworkDeviceService(db, nil, nil), db
}

// ndv7904SeedDevice 种子设备行。status 显式回写(Q4:零值 0 会被 default:2 覆盖)。
func ndv7904SeedDevice(t *testing.T, db *gorm.DB, name, ip string, status models.DeviceStatus,
	deptID, credID *string) *models.NetworkDevice {
	t.Helper()
	dev := &models.NetworkDevice{
		DeviceName:   name,
		DeviceType:   models.DeviceTypeSwitch,
		Vendor:       models.VendorHuawei,
		Model:        "S5735",
		IPAddress:    ip,
		Port:         22,
		SNMPPort:     161,
		CredentialID: credID,
		DeptID:       deptID,
		Location:     "3F 机房",
		Status:       status,
	}
	require.NoError(t, db.Create(dev).Error, "seed device %s", name)
	require.NoError(t, db.Model(&models.NetworkDevice{}).Where("id = ?", dev.ID).
		Update("status", status).Error, "force status column (Q4)")
	require.NoError(t, db.First(dev, "id = ?", dev.ID).Error)
	return dev
}

// ndv7904SeedCredential 种子凭证行(communities 可为空,用于探测失败分支)。
func ndv7904SeedCredential(t *testing.T, db *gorm.DB, name string, communities ...string) *models.AuthCredential {
	t.Helper()
	cred := &models.AuthCredential{
		CredentialName:  name,
		ProtocolType:    models.ProtocolTypeSSH,
		Username:        "admin",
		SNMPCommunities: pq.StringArray(communities),
		SNMPVersion:     models.SNMPVersionV2c,
	}
	require.NoError(t, db.Create(cred).Error, "seed credential %s", name)
	return cred
}

// ndv7904SeedDept 种子部门(sys_dept,统计 JOIN 与 Create 校验共用)。
func ndv7904SeedDept(t *testing.T, db *gorm.DB, name string) *models.Department {
	t.Helper()
	dept := &models.Department{DeptName: name, DeptCode: "code-" + name}
	require.NoError(t, db.Create(dept).Error, "seed dept %s", name)
	return dept
}

func ndv7904StrPtr(s string) *string { return &s }

// -------------------------------------------------------------------------
// CRUD 主链
// -------------------------------------------------------------------------

// TestNdv7904_CreateAndGet 合法创建 → GetByID 读回;GetByID 关联字段按现行为断言
// (Q3 关联名丢失 quirk,不修);Create 的重复 IP / 凭证不存在 / 部门不存在分支。
func TestNdv7904_CreateAndGet(t *testing.T) {
	ctx := context.Background()
	svc, db := newNdv7904(t)
	dept := ndv7904SeedDept(t, db, "运维部")
	cred := ndv7904SeedCredential(t, db, "主凭证", "public")

	dev, err := svc.Create(ctx, &CreateDeviceRequest{
		DeviceName:   "sw-core-01",
		DeviceType:   models.DeviceTypeSwitch,
		Vendor:       models.VendorHuawei,
		Model:        "S5735-L48",
		IPAddress:    "10.10.0.11",
		Port:         22,
		SNMPPort:     161,
		CredentialID: &cred.ID,
		DeptID:       &dept.ID,
		Location:     "3F",
		Status:       models.DeviceStatusOnline,
		Description:  "核心交换机",
		CreatedBy:    ndv7904Operator,
	})
	require.NoError(t, err)
	require.NotNil(t, dev)
	assert.NotEmpty(t, dev.ID)
	// Q4 锁定(服务层同款):CreateDeviceRequest.Status 是非指针枚举,DeviceStatusOnline=0
	// 是零值 → GORM 跳过该列 → 落列默认值 2(unknown)。调用方"在线"建机实际得到"未知"。
	assert.Equal(t, models.DeviceStatusUnknown, dev.Status,
		"锁定现行为:零值状态被列 default:2 覆盖(Q4)")
	assert.Equal(t, "sw-core-01", dev.DeviceName)

	// GetByID 读回基础字段
	got, err := svc.GetByID(ctx, dev.ID)
	require.NoError(t, err)
	assert.Equal(t, dev.IPAddress, got.IPAddress)
	assert.Equal(t, models.DeviceTypeSwitch, got.DeviceType)
	assert.Equal(t, models.VendorHuawei, got.Vendor)

	// Q3 锁定:即使部门/凭证行真实存在,GetByID 也不回填关联名
	assert.Nil(t, got.DeptName, "锁定现行为:GetByID 关联名丢失(Q3)")
	assert.Nil(t, got.CredentialName, "锁定现行为:GetByID 关联名丢失(Q3)")

	// GetByID 不存在 → 错误
	_, err = svc.GetByID(ctx, uuid.New().String())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "查询设备失败")

	// Create 重复 IP → 拒绝
	_, err = svc.Create(ctx, &CreateDeviceRequest{DeviceName: "dup", IPAddress: "10.10.0.11"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "IP地址已存在")

	// Create 凭证不存在 → 拒绝
	_, err = svc.Create(ctx, &CreateDeviceRequest{
		DeviceName: "x", IPAddress: "10.10.0.12", CredentialID: ndv7904StrPtr(uuid.New().String()),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "授权凭证不存在")

	// Create 部门不存在 → 拒绝
	_, err = svc.Create(ctx, &CreateDeviceRequest{
		DeviceName: "x", IPAddress: "10.10.0.13", DeptID: ndv7904StrPtr(uuid.New().String()),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "部门不存在")
}

// TestNdv7904_List_PaginationFilter 预置多行 → 分页 + 名称/IP/状态/类型/厂商/部门过滤 +
// 排序白名单分支(合法/非法/默认);loadAssociations 在 List 路径正常回填关联名(Q3 对照)。
func TestNdv7904_List_PaginationFilter(t *testing.T) {
	ctx := context.Background()
	svc, db := newNdv7904(t)
	deptA := ndv7904SeedDept(t, db, "运维部")
	cred := ndv7904SeedCredential(t, db, "列表凭证", "public")

	a := ndv7904SeedDevice(t, db, "sw-alpha", "10.1.0.1", models.DeviceStatusOnline, &deptA.ID, &cred.ID)
	_ = ndv7904SeedDevice(t, db, "sw-beta", "10.1.0.2", models.DeviceStatusOnline, &deptA.ID, &cred.ID)
	_ = ndv7904SeedDevice(t, db, "rt-gamma", "10.1.0.3", models.DeviceStatusOffline, nil, nil)
	last := ndv7904SeedDevice(t, db, "fw-delta", "10.1.0.4", models.DeviceStatusUnknown, nil, nil)
	_ = last
	// 差异化类型/厂商,便于类型/厂商过滤断言(seed helper 统一是 switch/huawei)
	require.NoError(t, db.Model(&models.NetworkDevice{}).Where("device_name = ?", "rt-gamma").
		Update("device_type", models.DeviceTypeRouter).Error)
	require.NoError(t, db.Model(&models.NetworkDevice{}).Where("device_name = ?", "fw-delta").
		Update("vendor", models.VendorRuijie).Error)

	// 分页:pageSize=2 current=2 → 2 行 + total=4
	list, total, err := svc.List(ctx, &ListDeviceRequest{BaseListRequest: baseListReq7904(2, 2)})
	require.NoError(t, err)
	assert.Equal(t, int64(4), total)
	assert.Len(t, list, 2)

	// 名称模糊过滤
	_, total, err = svc.List(ctx, &ListDeviceRequest{
		BaseListRequest: baseListReq7904(1, 10), DeviceName: ndv7904StrPtr("sw-"),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)

	// IP 模糊过滤
	list, total, err = svc.List(ctx, &ListDeviceRequest{
		BaseListRequest: baseListReq7904(1, 10), IP: ndv7904StrPtr("10.1.0.3"),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, "rt-gamma", list[0].DeviceName)

	// 状态过滤(models.DeviceStatus* 具名常量)
	offline := models.DeviceStatusOffline
	_, total, err = svc.List(ctx, &ListDeviceRequest{
		BaseListRequest: baseListReq7904(1, 10), Status: &offline,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total, "状态过滤必须命中 models.DeviceStatusOffline")

	// 类型 + 厂商过滤(rt-gamma 已改为 router、fw-delta 已改为 ruijie)
	routerType := models.DeviceTypeRouter
	list, total, err = svc.List(ctx, &ListDeviceRequest{
		BaseListRequest: baseListReq7904(1, 10), DeviceType: &routerType,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total, "类型过滤必须只命中 rt-gamma")
	assert.Equal(t, "rt-gamma", list[0].DeviceName)

	ruijie := models.VendorRuijie
	list, total, err = svc.List(ctx, &ListDeviceRequest{
		BaseListRequest: baseListReq7904(1, 10), Vendor: &ruijie,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total, "厂商过滤必须只命中 fw-delta")
	assert.Equal(t, "fw-delta", list[0].DeviceName)

	// 部门过滤
	list, total, err = svc.List(ctx, &ListDeviceRequest{
		BaseListRequest: baseListReq7904(1, 10), DeptID: &deptA.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	// Q3 对照:List 路径的 loadAssociations 在真实切片上执行 → 关联名正常回填
	require.NotNil(t, list[0].DeptName)
	assert.Equal(t, "运维部", *list[0].DeptName)
	require.NotNil(t, list[0].CredentialName)
	assert.Equal(t, "列表凭证", *list[0].CredentialName)

	// 排序白名单分支:deviceName ASC → alpha 打头
	asc := true
	list, _, err = svc.List(ctx, &ListDeviceRequest{
		BaseListRequest: base.BaseListRequest{
			Current: 1, PageSize: 10, OrderByColumn: "deviceName", IsAsc: &asc,
		},
	})
	require.NoError(t, err)
	require.Len(t, list, 4)
	assert.Equal(t, "fw-delta", list[0].DeviceName, "白名单排序必须生效(device_name ASC)")

	// 排序白名单分支:deviceName DESC → rt-gamma 打头(状态排序同理走白名单)
	desc := false
	list, _, err = svc.List(ctx, &ListDeviceRequest{
		BaseListRequest: base.BaseListRequest{
			Current: 1, PageSize: 10, OrderByColumn: "status", IsAsc: &desc,
		},
	})
	require.NoError(t, err)
	require.Len(t, list, 4)
	assert.Equal(t, models.DeviceStatusUnknown, list[0].Status, "status DESC → unknown(2) 打头")

	// 非法排序字段 → 静默忽略(ApplySort ok=false 分支),回退默认 created_at DESC
	list, total, err = svc.List(ctx, &ListDeviceRequest{
		BaseListRequest: base.BaseListRequest{
			Current: 1, PageSize: 10, OrderByColumn: "1; DROP TABLE sys_network_device",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, int64(4), total)
	assert.Len(t, list, 4)
	assert.Equal(t, a.DeviceName, list[0].DeviceName, "非法排序回退 created_at DESC(最后创建的打头)")
}

// TestNdv7904_Update_Delete_BatchDelete 更新读回;单删/批删计数;不存在 ID 与
// 重复 IP/无效凭证/无效部门的拒绝分支。
func TestNdv7904_Update_Delete_BatchDelete(t *testing.T) {
	ctx := context.Background()
	svc, db := newNdv7904(t)
	dept := ndv7904SeedDept(t, db, "网络部")
	cred := ndv7904SeedCredential(t, db, "更新凭证", "public")
	dev := ndv7904SeedDevice(t, db, "sw-old", "10.2.0.1", models.DeviceStatusUnknown, nil, nil)
	other := ndv7904SeedDevice(t, db, "sw-other", "10.2.0.2", models.DeviceStatusOnline, nil, nil)

	// 更新读回
	require.NoError(t, svc.Update(ctx, &UpdateDeviceRequest{
		ID:           dev.ID,
		DeviceName:   "sw-new",
		DeviceType:   models.DeviceTypeRouter,
		Vendor:       models.VendorH3C,
		Model:        "MSR3620",
		IPAddress:    "10.2.0.9",
		Port:         2222,
		SNMPPort:     1161,
		CredentialID: &cred.ID,
		DeptID:       &dept.ID,
		Location:     "5F",
		Status:       models.DeviceStatusOffline,
		Description:  "已改造",
		UpdatedBy:    ndv7904Operator,
	}))
	got, err := svc.GetByID(ctx, dev.ID)
	require.NoError(t, err)
	assert.Equal(t, "sw-new", got.DeviceName)
	assert.Equal(t, "10.2.0.9", got.IPAddress)
	assert.Equal(t, models.DeviceTypeRouter, got.DeviceType)
	assert.Equal(t, models.VendorH3C, got.Vendor)
	assert.Equal(t, 2222, got.Port)
	assert.Equal(t, models.DeviceStatusOffline, got.Status)

	// 更新:IP 被其他设备占用 → 拒绝
	err = svc.Update(ctx, &UpdateDeviceRequest{ID: dev.ID, IPAddress: "10.2.0.2"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "IP地址已被其他设备使用")

	// 更新:设备不存在 → 拒绝
	err = svc.Update(ctx, &UpdateDeviceRequest{ID: uuid.New().String(), IPAddress: "10.2.0.9"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "设备不存在")

	// 更新:凭证不存在 → 拒绝
	err = svc.Update(ctx, &UpdateDeviceRequest{
		ID: dev.ID, IPAddress: "10.2.0.9", CredentialID: ndv7904StrPtr(uuid.New().String()),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "授权凭证不存在")

	// 更新:部门不存在 → 拒绝
	err = svc.Update(ctx, &UpdateDeviceRequest{
		ID: dev.ID, IPAddress: "10.2.0.9", DeptID: ndv7904StrPtr(uuid.New().String()),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "部门不存在")

	// 批删:含不存在 ID → 报错透传。锁定现行为:BatchDelete 非原子 —— 逐条 Delete,
	// 前面的行已被软删后才在缺失 ID 上失败。
	err = svc.BatchDelete(ctx, []string{other.ID, uuid.New().String()})
	require.Error(t, err)
	var otherGone int64
	require.NoError(t, db.Model(&models.NetworkDevice{}).Where("id = ?", other.ID).Count(&otherGone).Error)
	assert.Equal(t, int64(0), otherGone, "锁定现行为:失败前的 other 已被软删(非原子)")

	// 批删余下一台 → 成功且行数清零
	require.NoError(t, svc.BatchDelete(ctx, []string{dev.ID}))
	var count int64
	require.NoError(t, db.Model(&models.NetworkDevice{}).Count(&count).Error)
	assert.Equal(t, int64(0), count)

	// 单删不存在 → 拒绝
	err = svc.Delete(ctx, uuid.New().String())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "设备不存在")
}

// TestNdv7904_UpdateStatus_SingleAndBatch 合法状态迁移(具名常量)读回;
// 不存在 ID 不报错;非法值分支:实现无白名单校验,任意 DeviceStatus 值都会落库(锁定)。
func TestNdv7904_UpdateStatus_SingleAndBatch(t *testing.T) {
	ctx := context.Background()
	svc, db := newNdv7904(t)
	d1 := ndv7904SeedDevice(t, db, "st-1", "10.3.0.1", models.DeviceStatusOnline, nil, nil)
	d2 := ndv7904SeedDevice(t, db, "st-2", "10.3.0.2", models.DeviceStatusOnline, nil, nil)

	// 单台迁移到离线
	require.NoError(t, svc.UpdateStatus(ctx, d1.ID, models.DeviceStatusOffline))
	got, err := svc.GetByID(ctx, d1.ID)
	require.NoError(t, err)
	assert.Equal(t, models.DeviceStatusOffline, got.Status)

	// 批量迁移到未知
	require.NoError(t, svc.UpdateStatusBatch(ctx, []string{d1.ID, d2.ID}, models.DeviceStatusUnknown))
	got1, err := svc.GetByID(ctx, d1.ID)
	require.NoError(t, err)
	assert.Equal(t, models.DeviceStatusUnknown, got1.Status)
	got2, err := svc.GetByID(ctx, d2.ID)
	require.NoError(t, err)
	assert.Equal(t, models.DeviceStatusUnknown, got2.Status)

	// 不存在 ID:UPDATE 影响 0 行,GORM 不视为错误(锁定 Update 语义)
	require.NoError(t, svc.UpdateStatus(ctx, uuid.New().String(), models.DeviceStatusOffline))
	require.NoError(t, svc.UpdateStatusBatch(ctx, []string{uuid.New().String()}, models.DeviceStatusOffline))

	// 非法值分支(锁定):实现不做白名单校验,越界枚举值同样落库
	require.NoError(t, svc.UpdateStatus(ctx, d2.ID, models.DeviceStatus(99)))
	got2, err = svc.GetByID(ctx, d2.ID)
	require.NoError(t, err)
	assert.Equal(t, models.DeviceStatus(99), got2.Status, "锁定现行为:状态值无校验")
}

// TestNdv7904_QuickCreateDevice QuickCreate 的可达分支:IP 已存在 / 凭证不存在 /
// 部门不存在 / 探测失败阻止创建。探测成功后的建行与 Enqueue 需 SNMP fake(79-06 DQ5)。
func TestNdv7904_QuickCreateDevice(t *testing.T) {
	ctx := context.Background()
	svc, db := newNdv7904(t)
	dept := ndv7904SeedDept(t, db, "快速部门")
	cred := ndv7904SeedCredential(t, db, "快速凭证", "public")
	emptyCred := ndv7904SeedCredential(t, db, "无社区凭证") // 无 SNMP communities

	req := func() *QuickCreateRequest {
		return &QuickCreateRequest{
			IPAddress:    "10.4.0.50",
			CredentialID: cred.ID,
			SNMPPort:     161,
			DeptID:       &dept.ID,
			Location:     "1F",
			Description:  "快速创建",
			CreatedBy:    ndv7904Operator,
		}
	}

	// 分支 1:IP 已存在且未删除 → 拒绝(探测前短路,nil 依赖不触达)
	ndv7904SeedDevice(t, db, "sw-exist", "10.4.0.50", models.DeviceStatusOnline, nil, nil)
	_, err := svc.QuickCreateDevice(ctx, req())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "已存在，请使用编辑功能")

	// 分支 2:凭证不存在 → 拒绝(nil discovery 在此之前不被解引用)
	missing := req()
	missing.CredentialID = uuid.New().String()
	missing.IPAddress = "10.4.0.51"
	_, err = svc.QuickCreateDevice(ctx, missing)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "授权凭证不存在")

	// 分支 3:部门不存在 → 拒绝
	badDept := req()
	badDept.IPAddress = "10.4.0.52"
	badDept.DeptID = ndv7904StrPtr(uuid.New().String())
	_, err = svc.QuickCreateDevice(ctx, badDept)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "部门不存在")

	// 分支 4:探测失败阻止创建。discoveryService 用同包真实装配(仅 db 字段),
	// 凭证无 SNMP communities → ProbeSingleDevice 在任何网络 I/O 之前返回 Success=false。
	probeSvc := NewNetworkDeviceService(db, &DeviceDiscoveryService{db: db}, nil)
	noCommunity := req()
	noCommunity.IPAddress = "10.4.0.53"
	noCommunity.CredentialID = emptyCred.ID
	noCommunity.DeptID = nil
	_, err = probeSvc.QuickCreateDevice(ctx, noCommunity)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "设备探测失败")
	assert.Contains(t, err.Error(), "SNMP community")

	// 分支 5:空 IP → ProbeSingleDevice 参数校验短路返回 Success=false(不触达 s.db)
	emptyIP := req()
	emptyIP.IPAddress = ""
	_, err = probeSvc.QuickCreateDevice(ctx, emptyIP)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "IP地址不能为空")

	// 上述分支均不得建行
	var count int64
	require.NoError(t, db.Model(&models.NetworkDevice{}).Where("ip_address LIKE ?", "10.4.0.5%").
		Count(&count).Error)
	assert.Equal(t, int64(1), count, "只有预置的那台设备,QuickCreate 失败分支不得建行")
}

// TestNdv7904_GetLastIPOctet 表驱动:标准 IP / 无点 / 空串 / 结尾点(包级纯函数)。
func TestNdv7904_GetLastIPOctet(t *testing.T) {
	cases := []struct {
		name string
		ip   string
		want string
	}{
		{name: "标准IPv4", ip: "192.168.1.42", want: "42"},
		{name: "末段多位", ip: "10.0.0.255", want: "255"},
		{name: "无点原样返回", ip: "no-dots", want: "no-dots"},
		{name: "空串", ip: "", want: ""},
		{name: "结尾点返回空段", ip: "1.2.", want: ""},
		{name: "单字符", ip: "5", want: "5"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, getLastIPOctet(tc.ip))
		})
	}
}

// TestNdv7904_GetDeviceStatistics 预置多状态/多类型/多厂商/多部门设备 → 统计 map
// 各键与手算一致(含 ByDept 的 LEFT JOIN 空部门归 "" 键)。
func TestNdv7904_GetDeviceStatistics(t *testing.T) {
	ctx := context.Background()
	svc, db := newNdv7904(t)
	deptA := ndv7904SeedDept(t, db, "运维部")
	deptB := ndv7904SeedDept(t, db, "网络部")

	ndv7904SeedDevice(t, db, "st-a", "10.5.0.1", models.DeviceStatusOnline, &deptA.ID, nil)
	ndv7904SeedDevice(t, db, "st-b", "10.5.0.2", models.DeviceStatusOnline, &deptA.ID, nil)
	ndv7904SeedDevice(t, db, "st-c", "10.5.0.3", models.DeviceStatusOffline, &deptB.ID, nil)
	// 差异化第三行,便于分组断言(seed helper 统一是 switch/huawei)
	require.NoError(t, db.Model(&models.NetworkDevice{}).Where("device_name = ?", "st-c").
		Updates(map[string]any{
			"device_type": string(models.DeviceTypeRouter),
			"vendor":      string(models.VendorH3C),
		}).Error)
	// 无部门设备:类型/厂商也与其他行不同,便于断言分组键
	lonely := &models.NetworkDevice{
		DeviceName: "st-d", DeviceType: models.DeviceTypeAP, Vendor: models.VendorMaipu,
		IPAddress: "10.5.0.4", Status: models.DeviceStatusUnknown, Port: 22, SNMPPort: 161,
	}
	require.NoError(t, db.Create(lonely).Error)
	require.NoError(t, db.Model(&models.NetworkDevice{}).Where("id = ?", lonely.ID).
		Update("status", models.DeviceStatusUnknown).Error)

	stats, err := svc.GetDeviceStatistics(ctx)
	require.NoError(t, err)
	require.NotNil(t, stats)

	assert.Equal(t, int64(4), stats["totalDevices"])
	assert.Equal(t, int64(2), stats["onlineDevices"], "models.DeviceStatusOnline 计数")
	assert.Equal(t, int64(1), stats["offlineDevices"], "models.DeviceStatusOffline 计数")
	assert.Equal(t, int64(1), stats["unknownDevices"], "models.DeviceStatusUnknown 计数")

	byType, ok := stats["byType"].(map[string]int64)
	require.True(t, ok)
	assert.Equal(t, int64(2), byType[string(models.DeviceTypeSwitch)])
	assert.Equal(t, int64(1), byType[string(models.DeviceTypeAP)])

	byVendor, ok := stats["byVendor"].(map[string]int64)
	require.True(t, ok)
	assert.Equal(t, int64(2), byVendor[string(models.VendorHuawei)])
	assert.Equal(t, int64(1), byVendor[string(models.VendorMaipu)])

	byDept, ok := stats["byDept"].(map[string]int64)
	require.True(t, ok)
	assert.Equal(t, int64(2), byDept["运维部"])
	assert.Equal(t, int64(1), byDept["网络部"])
	assert.Equal(t, int64(1), byDept[""], "无部门设备 LEFT JOIN 后归入空键")
}

// TestNdv7904_GetDevicesByDeptAndCredential 按部门/凭据查询命中集;无关联 → 空集。
func TestNdv7904_GetDevicesByDeptAndCredential(t *testing.T) {
	ctx := context.Background()
	svc, db := newNdv7904(t)
	dept := ndv7904SeedDept(t, db, "查询部门")
	cred := ndv7904SeedCredential(t, db, "查询凭证", "public")

	ndv7904SeedDevice(t, db, "q-1", "10.6.0.1", models.DeviceStatusOnline, &dept.ID, &cred.ID)
	ndv7904SeedDevice(t, db, "q-2", "10.6.0.2", models.DeviceStatusOnline, &dept.ID, &cred.ID)
	ndv7904SeedDevice(t, db, "q-3", "10.6.0.3", models.DeviceStatusOnline, nil, nil)

	byDept, err := svc.GetDevicesByDept(ctx, dept.ID)
	require.NoError(t, err)
	assert.Len(t, byDept, 2)

	byCred, err := svc.GetDevicesByCredential(ctx, cred.ID)
	require.NoError(t, err)
	assert.Len(t, byCred, 2)

	// 无关联 → 空集不报错
	emptyDept, err := svc.GetDevicesByDept(ctx, uuid.New().String())
	require.NoError(t, err)
	assert.Empty(t, emptyDept)
	emptyCred, err := svc.GetDevicesByCredential(ctx, uuid.New().String())
	require.NoError(t, err)
	assert.Empty(t, emptyCred)
}

// TestNdv7904_NilDeps_ErrorBranch nil 注入下两依赖的触达边界:
//   - discoveryService 在 QuickCreateDevice 中被解引用于 ProbeSingleDevice(s.db),
//     但探测前有 3 个校验短路 → nil 注入下这些短路必须先行返回(不 panic);
//   - deviceInfoCollectionSvc 只在探测成功后的 :346 nil-guard 中触达,而成功路径
//     需要 SNMP fake(79-06 DQ5)→ 本 plan 用 List 空集(不进 loadAssociations 循环)
//     证明 nil 依赖路径安全。
func TestNdv7904_NilDeps_ErrorBranch(t *testing.T) {
	ctx := context.Background()
	svc, _ := newNdv7904(t)

	// 空表 List → loadAssociations 对空切片早退(len==0 分支),nil 依赖不被触达
	list, total, err := svc.List(ctx, &ListDeviceRequest{BaseListRequest: baseListReq7904(1, 10)})
	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Empty(t, list)

	// nil discovery 下 QuickCreateDevice 必须在探测前被校验短路(不 panic)
	_, err = svc.QuickCreateDevice(ctx, &QuickCreateRequest{IPAddress: "10.7.0.1", CredentialID: "nope"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "授权凭证不存在")
}
