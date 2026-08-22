//go:build !skip_db_tests
// +build !skip_db_tests

package portcollection

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/device"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
)

// =====================================================================
// 74-09 portcollection gapfill:
//   trunk_filter.go BuildTrunkPortBlockset / IsTrunkPort
//   utils.go       buildPortStatus / buildPortStatusFromDesc
//   query.go       QueryService.GetList / GetStats / CleanOldRecords
//   template_cache.go TemplateCache.Get(命中/未命中/错误)
//   service.go     NewPortCollectionService / NewCollectionService
//   collection.go  ReloadConfig(loadConfigFromDB 空 stub)
//   vendor_port_template.go VendorExitViewCmd
//   parser.go      get*Command / get*Template 厂商映射
// =====================================================================

// ---------------- trunk_filter.go ----------------

func TestBuildTrunkPortBlockset_NilAndEmpty(t *testing.T) {
	ctx := context.Background()
	assert.Empty(t, BuildTrunkPortBlockset(ctx, nil, "dev-1"), "nil db → 空集合")
	db := setupPortStatusTestDB(t)
	assert.Empty(t, BuildTrunkPortBlockset(ctx, db, ""), "空 deviceID → 空集合")
	assert.Empty(t, BuildTrunkPortBlockset(ctx, db, uuid.NewString()), "无行 → 空集合")
}

func TestBuildTrunkPortBlockset_MatchesHints(t *testing.T) {
	db := setupPortStatusTestDB(t)
	ctx := context.Background()
	devID := uuid.NewString()
	otherDev := uuid.NewString()
	now := time.Now()

	rows := []models.DevicePortStatus{
		{DeviceID: devID, InterfaceName: "GE0/0/1", PortType: "trunk", CollectedAt: now},
		{DeviceID: devID, InterfaceName: "GE0/0/2", PortType: "Hybrid", CollectedAt: now},
		{DeviceID: devID, InterfaceName: "GE0/0/3", PortType: "uplink-port", CollectedAt: now},
		{DeviceID: devID, InterfaceName: "GE0/0/4", PortType: "access", CollectedAt: now},   // 不命中
		{DeviceID: devID, InterfaceName: "GE0/0/5", PortType: "", CollectedAt: now},         // 空 → SQL 层排除
		{DeviceID: otherDev, InterfaceName: "GE0/0/6", PortType: "trunk", CollectedAt: now}, // 别的设备
	}
	require.NoError(t, db.Create(&rows).Error)

	bs := BuildTrunkPortBlockset(ctx, db, devID)
	assert.True(t, bs["GE0/0/1"], "trunk 命中")
	assert.True(t, bs["GE0/0/2"], "hybrid 命中(大小写不敏感)")
	assert.True(t, bs["GE0/0/3"], "uplink-port 命中")
	assert.False(t, bs["GE0/0/4"], "access 不命中")
	assert.False(t, bs["GE0/0/5"], "空 PortType 不命中")
	assert.False(t, bs["GE0/0/6"], "别的设备端口不进集合")
}

func TestIsTrunkPort(t *testing.T) {
	assert.False(t, IsTrunkPort(nil, "GE0/0/1"), "空集合 → false")
	assert.False(t, IsTrunkPort(map[string]bool{}, "GE0/0/1"), "空集合 → false")

	bs := map[string]bool{"GE0/0/1": true}
	assert.True(t, IsTrunkPort(bs, "GE0/0/1"), "直接命中")
	assert.False(t, IsTrunkPort(bs, "GE0/0/2"), "未命中")

	// 规范化命中: 集合里是规范化名,传入原始全称
	bs2 := map[string]bool{NormalizeInterfaceName("GigabitEthernet0/0/1"): true}
	assert.True(t, IsTrunkPort(bs2, "GigabitEthernet0/0/1"), "规范化后命中")
}

// ---------------- utils.go buildPortStatus / buildPortStatusFromDesc ----------------

func TestBuildPortStatus_BaseAndDot1x(t *testing.T) {
	now := time.Now()
	vlan := 10

	// 无 dot1x / security map
	ps := buildPortStatus("dev1", "GE0/0/1",
		InterfaceInfo{Name: "GE0/0/1", VLAN: &vlan, Duplex: "full", Speed: "1000", PortType: "access"},
		"up", "up", "desc", nil, nil, now)
	assert.Equal(t, "dev1", ps.DeviceID)
	assert.Equal(t, "GE0/0/1", ps.InterfaceName)
	assert.Equal(t, "up", ps.AdminStatus)
	assert.Equal(t, &vlan, ps.VLAN)
	assert.Equal(t, "full", ps.Duplex)
	assert.Equal(t, "1000", ps.Speed)
	assert.Equal(t, now, ps.CollectedAt)
	assert.False(t, ps.Dot1xEnabled)
	assert.False(t, ps.PortSecurityEnabled)

	// dot1x AUTHORIZED + 合法 MaxUser
	dot1x := map[string]Dot1xInfo{
		"GE0/0/1": {Enabled: true, PortStatus: "AUTHORIZED", MaxUser: "5"},
		"GE0/0/2": {Enabled: true, PortStatus: "UNAUTHORIZED", MaxUser: "unlimited"},
		"GE0/0/3": {Enabled: true, PortStatus: "something-else", MaxUser: ""},
		"GE0/0/4": {Enabled: true, PortStatus: "AUTHORIZED", MaxUser: "abc"}, // 非数字 → nil + warn
		"GE0/0/5": {Enabled: true, PortStatus: "AUTHORIZED", MaxUser: "-3"},  // ≤0 → nil
	}
	ps = buildPortStatus("dev1", "GE0/0/1", InterfaceInfo{}, "up", "up", "", dot1x, nil, now)
	assert.True(t, ps.Dot1xEnabled)
	assert.Equal(t, models.Dot1xStatusAuthorized, ps.Dot1xPortStatus)
	require.NotNil(t, ps.Dot1xUserLimit)
	assert.Equal(t, 5, *ps.Dot1xUserLimit)

	ps = buildPortStatus("dev1", "GE0/0/2", InterfaceInfo{}, "up", "up", "", dot1x, nil, now)
	assert.Equal(t, models.Dot1xStatusUnauthorized, ps.Dot1xPortStatus)
	assert.Nil(t, ps.Dot1xUserLimit, "unlimited → nil")

	ps = buildPortStatus("dev1", "GE0/0/3", InterfaceInfo{}, "up", "up", "", dot1x, nil, now)
	assert.Equal(t, models.Dot1xStatusUnknown, ps.Dot1xPortStatus)
	assert.Nil(t, ps.Dot1xUserLimit, "空 MaxUser → nil")

	ps = buildPortStatus("dev1", "GE0/0/4", InterfaceInfo{}, "up", "up", "", dot1x, nil, now)
	assert.Nil(t, ps.Dot1xUserLimit, "非数字 → nil")

	ps = buildPortStatus("dev1", "GE0/0/5", InterfaceInfo{}, "up", "up", "", dot1x, nil, now)
	assert.Nil(t, ps.Dot1xUserLimit, "负数 → nil")
}

func TestBuildPortStatus_Security(t *testing.T) {
	now := time.Now()
	sec := map[string]PortSecurityInfo{
		"GE0/0/1": {Enabled: true, SecurityMode: "RESTRICT", MaxMAC: 3, CurrentMAC: 2},
		"GE0/0/2": {Enabled: true, SecurityMode: "protect", MaxMAC: 0, CurrentMAC: 0},
		"GE0/0/3": {Enabled: true, SecurityMode: "SHUTDOWN"},
		"GE0/0/4": {Enabled: true, SecurityMode: "weird"},
	}

	ps := buildPortStatus("d", "GE0/0/1", InterfaceInfo{}, "", "", "", nil, sec, now)
	assert.True(t, ps.PortSecurityEnabled)
	assert.Equal(t, models.PortSecurityModeRestrict, ps.PortSecurityMode)
	require.NotNil(t, ps.MaxMACCount)
	assert.Equal(t, 3, *ps.MaxMACCount)
	require.NotNil(t, ps.CurrentMACCount)
	assert.Equal(t, 2, *ps.CurrentMACCount)

	ps = buildPortStatus("d", "GE0/0/2", InterfaceInfo{}, "", "", "", nil, sec, now)
	assert.Equal(t, models.PortSecurityModeProtect, ps.PortSecurityMode, "大小写不敏感")
	assert.Nil(t, ps.MaxMACCount, "MaxMAC=0 → nil")
	assert.Nil(t, ps.CurrentMACCount, "CurrentMAC=0 → nil")

	ps = buildPortStatus("d", "GE0/0/3", InterfaceInfo{}, "", "", "", nil, sec, now)
	assert.Equal(t, models.PortSecurityModeShutdown, ps.PortSecurityMode)

	ps = buildPortStatus("d", "GE0/0/4", InterfaceInfo{}, "", "", "", nil, sec, now)
	assert.Equal(t, models.PortSecurityModeNone, ps.PortSecurityMode, "未知模式 → None")
}

func TestBuildPortStatusFromDesc(t *testing.T) {
	now := time.Now()

	// 无 dot1x / security
	ps := buildPortStatusFromDesc("dev1", "GE0/0/1",
		InterfaceDescription{AdminStatus: "up", OperStatus: "down", Description: "desc"},
		nil, nil, now)
	assert.Equal(t, "dev1", ps.DeviceID)
	assert.Equal(t, "GE0/0/1", ps.InterfaceName)
	assert.Equal(t, "up", ps.AdminStatus)
	assert.Equal(t, "down", ps.OperStatus)
	assert.Equal(t, "desc", ps.Description)
	assert.False(t, ps.Dot1xEnabled)

	// dot1x + security 全量
	dot1x := map[string]Dot1xInfo{
		"GE0/0/1": {Enabled: true, PortStatus: "AUTHORIZED"},
		"GE0/0/2": {Enabled: true, PortStatus: "UNAUTHORIZED"},
		"GE0/0/3": {Enabled: true, PortStatus: "other"},
	}
	sec := map[string]PortSecurityInfo{
		"GE0/0/1": {Enabled: true, SecurityMode: "RESTRICT", MaxMAC: 2, CurrentMAC: 1},
	}
	ps = buildPortStatusFromDesc("d", "GE0/0/1", InterfaceDescription{}, dot1x, sec, now)
	assert.Equal(t, models.Dot1xStatusAuthorized, ps.Dot1xPortStatus)
	assert.Equal(t, models.PortSecurityModeRestrict, ps.PortSecurityMode)
	require.NotNil(t, ps.MaxMACCount)

	ps = buildPortStatusFromDesc("d", "GE0/0/2", InterfaceDescription{}, dot1x, nil, now)
	assert.Equal(t, models.Dot1xStatusUnauthorized, ps.Dot1xPortStatus)

	ps = buildPortStatusFromDesc("d", "GE0/0/3", InterfaceDescription{}, dot1x, nil, now)
	assert.Equal(t, models.Dot1xStatusUnknown, ps.Dot1xPortStatus)
}

// ---------------- query.go QueryService ----------------

func newQuerySvc(t *testing.T) (*QueryService, string) {
	t.Helper()
	db := setupPortStatusTestDB(t)
	devID := uuid.NewString()
	now := time.Now()
	old := now.AddDate(0, 0, -30)
	rows := []models.DevicePortStatus{
		{DeviceID: devID, InterfaceName: "GE0/0/1", AdminStatus: "up", OperStatus: "up", Dot1xEnabled: true, PortSecurityEnabled: true, CollectedAt: now},
		{DeviceID: devID, InterfaceName: "GE0/0/2", AdminStatus: "up", OperStatus: "down", CollectedAt: now.Add(-time.Hour)},
		{DeviceID: devID, InterfaceName: "GE0/0/10", AdminStatus: "down", OperStatus: "down", CollectedAt: old},
	}
	require.NoError(t, db.Create(&rows).Error)
	return NewQueryService(db), devID
}

func TestQueryService_GetList_Filters(t *testing.T) {
	svc, devID := newQuerySvc(t)
	ctx := context.Background()

	// 全量
	list, total, err := svc.GetList(ctx, &ListRequest{
		BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 10},
	})
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, list, 3)

	// DeviceID 过滤
	list, total, err = svc.GetList(ctx, &ListRequest{
		BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 10},
		DeviceID:        devID,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)

	list, total, err = svc.GetList(ctx, &ListRequest{
		BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 10},
		DeviceID:        uuid.NewString(), // 别的设备
	})
	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Empty(t, list)

	// InterfaceName LIKE
	_, total, err = svc.GetList(ctx, &ListRequest{
		BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 10},
		InterfaceName:   "GE0/0/1",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total, "GE0/0/1 与 GE0/0/10 都 LIKE 命中")

	// AdminStatus / OperStatus
	_, total, err = svc.GetList(ctx, &ListRequest{
		BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 10},
		AdminStatus:     "down",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)

	_, total, err = svc.GetList(ctx, &ListRequest{
		BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 10},
		OperStatus:      "up",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)

	// Dot1xEnabled / PortSecurityEnabled
	tru := true
	_, total, err = svc.GetList(ctx, &ListRequest{
		BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 10},
		Dot1xEnabled:    &tru,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)

	_, total, err = svc.GetList(ctx, &ListRequest{
		BaseListRequest:      base.BaseListRequest{Current: 1, PageSize: 10},
		PortSecurityEnabled: &tru,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
}

func TestQueryService_GetList_PaginationAndSort(t *testing.T) {
	svc, _ := newQuerySvc(t)
	ctx := context.Background()

	// 分页
	list, total, err := svc.GetList(ctx, &ListRequest{
		BaseListRequest: base.BaseListRequest{Current: 2, PageSize: 2},
	})
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, list, 1, "第 2 页只剩 1 条")

	// interfaceName 排序(sqlite 降级字符串序,仍应不报错)
	asc := true
	list, _, err = svc.GetList(ctx, &ListRequest{
		BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 10, OrderByColumn: "interfaceName", IsAsc: &asc},
	})
	require.NoError(t, err)
	assert.Len(t, list, 3)

	// 白名单字段排序(operStatus)
	list, _, err = svc.GetList(ctx, &ListRequest{
		BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 10, OrderByColumn: "operStatus", IsAsc: &asc},
	})
	require.NoError(t, err)
	assert.Len(t, list, 3)
}

func TestQueryService_GetStats(t *testing.T) {
	svc, _ := newQuerySvc(t)
	stats, err := svc.GetStats(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(3), stats.TotalRecords)
	assert.Equal(t, int64(1), stats.UniqueDevices)
	assert.Equal(t, int64(1), stats.Dot1xEnabledCount)
	assert.Equal(t, int64(1), stats.SecurityEnabledCount)
	assert.Equal(t, int64(1), stats.UpPortsCount)
	assert.Equal(t, int64(2), stats.DownPortsCount)
	require.NotNil(t, stats.LatestCollection)

	// 空表 LatestCollection 为 nil
	db := setupPortStatusTestDB(t)
	empty := NewQueryService(db)
	stats, err = empty.GetStats(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(0), stats.TotalRecords)
	assert.Nil(t, stats.LatestCollection)
}

func TestQueryService_CleanOldRecords(t *testing.T) {
	svc, _ := newQuerySvc(t)
	ctx := context.Background()

	// 清 7 天前 → 只有 old 那条
	affected, err := svc.CleanOldRecords(ctx, 7)
	require.NoError(t, err)
	assert.Equal(t, int64(1), affected)

	// 再清 → 0
	affected, err = svc.CleanOldRecords(ctx, 7)
	require.NoError(t, err)
	assert.Equal(t, int64(0), affected)
}

// ---------------- template_cache.go ----------------

func TestTemplateCache_Get(t *testing.T) {
	cache := NewTemplateCache()

	// 错误路径 → error(不缓存)
	_, err := cache.Get("templates/__no_such_template__.textfsm")
	assert.Error(t, err)

	// 真实嵌入模板(走 embed FS,不依赖 cwd)→ 成功并缓存
	fsm1, err := cache.Get("templates/ruijie_os_show_interfaces_status.textfsm")
	require.NoError(t, err)
	require.NotNil(t, fsm1)

	// 第二次 → 缓存命中(同一指针)
	fsm2, err := cache.Get("templates/ruijie_os_show_interfaces_status.textfsm")
	require.NoError(t, err)
	assert.Same(t, fsm1, fsm2)
}

// ---------------- service.go / collection.go 构造 ----------------

func TestNewPortCollectionService_Constructor(t *testing.T) {
	db := setupPortStatusTestDB(t)
	svc := NewPortCollectionService(db, nil)
	require.NotNil(t, svc)
	require.NotNil(t, svc.Collection)
	require.NotNil(t, svc.Query)

	// ReloadConfig 为空 stub,不应 panic
	svc.Collection.ReloadConfig()
}

// ---------------- vendor_port_template.go VendorExitViewCmd ----------------

func TestVendorExitViewCmd(t *testing.T) {
	assert.Equal(t, "quit", VendorExitViewCmd(models.VendorHuawei))
	assert.Equal(t, "quit", VendorExitViewCmd(models.VendorH3C))
	assert.Equal(t, "exit", VendorExitViewCmd(models.VendorRuijie))
	assert.Equal(t, "exit", VendorExitViewCmd(models.VendorMaipu))
	assert.Equal(t, "exit", VendorExitViewCmd(models.DeviceVendor("unknown")), "默认思科风格 exit")
}

// ---------------- parser.go 厂商命令/模板映射 ----------------

func TestParserVendorCommandAndTemplateMaps(t *testing.T) {
	// getInterfaceCommand
	assert.Equal(t, "display interface brief", getInterfaceCommand(models.VendorHuawei))
	assert.Equal(t, "display interface", getInterfaceCommand(models.VendorH3C))
	assert.Equal(t, "show interfaces status", getInterfaceCommand(models.VendorRuijie))
	assert.Equal(t, "show interface", getInterfaceCommand(models.VendorMaipu))
	assert.Equal(t, "show interface", getInterfaceCommand(models.DeviceVendor("x")))

	// getInterfaceTemplate
	assert.Contains(t, getInterfaceTemplate(models.VendorHuawei), "huawei_vrp_display_interface_brief")
	assert.Contains(t, getInterfaceTemplate(models.VendorH3C), "huawei_vrp_display_interface.textfsm")
	assert.Contains(t, getInterfaceTemplate(models.VendorRuijie), "ruijie_os_show_interfaces_status")
	assert.Contains(t, getInterfaceTemplate(models.DeviceVendor("x")), "huawei_vrp_display_interface.textfsm")

	// getInterfaceDescriptionCommand
	assert.Equal(t, "display interface description", getInterfaceDescriptionCommand(models.VendorHuawei))
	assert.Equal(t, "show int des", getInterfaceDescriptionCommand(models.VendorRuijie))
	assert.Equal(t, "show int des", getInterfaceDescriptionCommand(models.VendorMaipu))
	assert.Equal(t, "show int des", getInterfaceDescriptionCommand(models.DeviceVendor("x")))

	// getInterfaceDescriptionTemplate
	assert.Contains(t, getInterfaceDescriptionTemplate(models.VendorHuawei), "display_interface_description")
	assert.Contains(t, getInterfaceDescriptionTemplate(models.VendorRuijie), "show_interface_description")
	assert.Contains(t, getInterfaceDescriptionTemplate(models.DeviceVendor("x")), "ruijie_os_show_interface_description")

	// getDot1xBatchCommand
	assert.Equal(t, "display dot1x", getDot1xBatchCommand(models.VendorHuawei))
	assert.Equal(t, "display dot1x", getDot1xBatchCommand(models.VendorH3C))
	assert.Equal(t, "show dot1x port-control", getDot1xBatchCommand(models.VendorRuijie))
	assert.Equal(t, "display dot1x", getDot1xBatchCommand(models.DeviceVendor("x")))

	// getPortSecurityBatchCommand
	assert.Equal(t, "display port-security", getPortSecurityBatchCommand(models.VendorHuawei))
	assert.Equal(t, "show port-security all", getPortSecurityBatchCommand(models.VendorRuijie))
	assert.Equal(t, "display port-security", getPortSecurityBatchCommand(models.DeviceVendor("x")))

	// getDot1xBatchTemplate
	assert.Contains(t, getDot1xBatchTemplate(models.VendorHuawei), "huawei_vrp_display_dot1x")
	assert.Contains(t, getDot1xBatchTemplate(models.VendorRuijie), "ruijie_os_show_dot1x")
	assert.Contains(t, getDot1xBatchTemplate(models.DeviceVendor("x")), "huawei_vrp_display_dot1x")

	// getPortSecurityBatchTemplate
	assert.Contains(t, getPortSecurityBatchTemplate(models.VendorHuawei), "huawei_vrp_display_port-security")
	assert.Contains(t, getPortSecurityBatchTemplate(models.VendorMaipu), "ruijie_os_show_port-security")
	assert.Contains(t, getPortSecurityBatchTemplate(models.DeviceVendor("x")), "huawei_vrp_display_port-security")
}

// parseInterfaceVLANInfo 非华为/H3C 直接返回空 map(无需设备连接)。
func TestParseInterfaceVLANInfo_NonHuaweiShortCircuit(t *testing.T) {
	m, err := parseInterfaceVLANInfo(nil, models.VendorRuijie)
	require.NoError(t, err)
	assert.Empty(t, m)

	m, err = parseInterfaceVLANInfo(nil, models.VendorMaipu)
	require.NoError(t, err)
	assert.Empty(t, m)
}

// ---------------- parser.go 设备未连接错误路径 ----------------
//
// ScrapliWrapper 是具体类型、内部 scrapligo driver 无法注入 mock,
// 解析主循环(记录处理)只能在真实设备/集成环境覆盖。
// 零值 wrapper 的 SendCommand 在 acquireOp 即返回 "连接不可用",
// 这里锁定四个 parse/get 函数对 SendCommand 错误的透传行为。

func TestParseFunctions_UnconnectedWrapperErrors(t *testing.T) {
	w := &device.ScrapliWrapper{}
	cache := NewTemplateCache()

	_, err := parseInterfaceList(w, models.VendorRuijie)
	require.Error(t, err)

	_, err = parseInterfaceDescriptions(w, models.VendorHuawei)
	require.Error(t, err)

	_, err = getAllDot1xStatus(w, models.VendorRuijie, cache)
	require.Error(t, err)

	_, err = getAllPortSecurity(w, models.VendorHuawei, cache)
	require.Error(t, err)

	// 华为 parseInterfaceVLANInfo 命令失败 → 返回空 map + nil error(VLAN 可选)
	m, err := parseInterfaceVLANInfo(w, models.VendorHuawei)
	require.NoError(t, err)
	assert.Empty(t, m)
}

// ---------------- collection.go 设备采集失败路径(禁用连接池,无 SSH) ----------------

// setupNetworkDeviceTable 在 port status 表之外补建 sys_network_device 最小列集。
func setupNetworkDeviceTable(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.Exec(`
		CREATE TABLE IF NOT EXISTS sys_network_device (
			id TEXT PRIMARY KEY,
			device_name TEXT,
			device_type TEXT,
			vendor TEXT,
			ip_address TEXT,
			port INTEGER,
			status INTEGER DEFAULT 2,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)
	`).Error)
}

// newDisabledPoolExecutor 构造一个连接池已禁用的 executor,
// 让 collectDevicePort 走 "连接池未启用" 失败路径,无需真实 SSH。
func newDisabledPoolExecutor(t *testing.T) *device.DeviceExecutor {
	t.Helper()
	pool := device.NewDeviceConnectionPool(nil, nil, nil)
	pool.SetEnabled(false)
	scheduler := device.NewDeviceTaskScheduler(pool, nil)
	return device.NewDeviceExecutor(scheduler, nil)
}

func TestCollectDevice_NotFound(t *testing.T) {
	db := setupPortStatusTestDB(t)
	setupNetworkDeviceTable(t, db)
	svc := NewCollectionService(db, newDisabledPoolExecutor(t))

	_, err := svc.CollectDevice(context.Background(), uuid.NewString())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "设备不存在")
}

func TestCollectDevice_ConnectionPoolDisabled(t *testing.T) {
	db := setupPortStatusTestDB(t)
	setupNetworkDeviceTable(t, db)
	svc := NewCollectionService(db, newDisabledPoolExecutor(t))

	devID := uuid.NewString()
	require.NoError(t, db.Exec(`
		INSERT INTO sys_network_device (id, device_name, device_type, vendor, ip_address, port, status, created_at, updated_at)
		VALUES (?, 'sw-01', 'switch', 'ruijie', '192.0.2.1', 22, 0, '2024-01-01', '2024-01-01')
	`, devID).Error)

	result, err := svc.CollectDevice(context.Background(), devID)
	require.NoError(t, err, "采集失败走 result.ErrorMessage 而非 error 返回")
	require.NotNil(t, result)
	assert.Equal(t, devID, result.DeviceID)
	assert.Equal(t, "sw-01", result.DeviceName)
	assert.Contains(t, result.ErrorMessage, "连接池未启用")
}

func TestCollectAllDevices_NoOnline(t *testing.T) {
	db := setupPortStatusTestDB(t)
	setupNetworkDeviceTable(t, db)
	svc := NewCollectionService(db, newDisabledPoolExecutor(t))

	_, err := svc.CollectAllDevices(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "没有在线设备")
}

func TestCollectAllDevices_OneOnlineDeviceFails(t *testing.T) {
	db := setupPortStatusTestDB(t)
	setupNetworkDeviceTable(t, db)
	svc := NewCollectionService(db, newDisabledPoolExecutor(t))

	devID := uuid.NewString()
	require.NoError(t, db.Exec(`
		INSERT INTO sys_network_device (id, device_name, device_type, vendor, ip_address, port, status, created_at, updated_at)
		VALUES (?, 'sw-online', 'switch', 'huawei', '192.0.2.2', 22, 0, '2024-01-01', '2024-01-01')
	`, devID).Error)
	// 一台离线设备(不应被采集)
	require.NoError(t, db.Exec(`
		INSERT INTO sys_network_device (id, device_name, device_type, vendor, ip_address, port, status, created_at, updated_at)
		VALUES (?, 'sw-offline', 'switch', 'huawei', '192.0.2.3', 22, 1, '2024-01-01', '2024-01-01')
	`, uuid.NewString()).Error)

	results, err := svc.CollectAllDevices(context.Background())
	require.NoError(t, err)
	require.Len(t, results, 1, "只采集在线设备")
	assert.Equal(t, "sw-online", results[0].DeviceName)
	assert.NotEmpty(t, results[0].ErrorMessage, "连接池禁用 → 采集失败但结果仍返回")
}
