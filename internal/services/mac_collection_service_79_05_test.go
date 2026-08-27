package services

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// -------------------------------------------------------------------------
// 79-05 Task 4: mac_collection_service.go(291 stmts @28.5%)
//
// 覆盖面: 解析纯函数表(parse 三函数/merge/cleanTimestamp)+ 命令选择 +
// CRUD/统计/清理 + loadConfigFromDB/ReloadConfig + executor nil 边界。
// collectDeviceMAC 的 executor 真路径(ExecuteOnDevice/ExecuteMultipleOnDevice)归 79-06,
// 本文件只在 executor=nil 下触达 panic-recovery 边界(见 TestMcl7905_Collect_NilExecutor)。
//
// 既有 mac_collection_service_test.go(零值 &MACCollectionService{} 直调解析函数)勿动,
// 本文件 helper 带 mcl7905 后缀(R5)。
// -------------------------------------------------------------------------

// newMcl7905 sqlite(t.TempDir 文件库)+ 全 nil 依赖的 MACCollectionService。
// 构造函数会调 loadConfigFromDB → 需要 sys_config 表(models.Config)。
func newMcl7905(t *testing.T) (*MACCollectionService, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(strings.ReplaceAll(t.TempDir(), `\`, "/")+"/mcl7905.db"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err, "open sqlite temp db")
	if sqlDB, err := db.DB(); err == nil {
		t.Cleanup(func() { _ = sqlDB.Close() })
	}
	require.NoError(t, db.AutoMigrate(
		&models.DeviceMACAddress{},
		&models.NetworkDevice{},
		&models.Config{},
	), "auto migrate mac collection models")
	return NewMACCollectionService(db, nil, nil, nil), db
}

// mcl7905MAC 构造 canonical MAC(i 序号)。
func mcl7905MAC(i int) string {
	const hexDigits = "0123456789ABCDEF"
	b := []byte("AA:BB:CC:00:00:00")
	b[15] = hexDigits[i%16]
	b[13] = hexDigits[(i/16)%16]
	return string(b)
}

// mcl7905SeedMAC 落一条 DeviceMACAddress(BeforeCreate 会归一化 MAC/接口名,入参用 canonical)。
func mcl7905SeedMAC(t *testing.T, db *gorm.DB, deviceID, mac, iface string, vlan *int, macType models.MACType, collectedAt time.Time) *models.DeviceMACAddress {
	t.Helper()
	row := &models.DeviceMACAddress{
		DeviceID:      deviceID,
		MACAddress:    mac,
		InterfaceName: iface,
		VLANID:        vlan,
		MACType:       macType,
		CollectedAt:   collectedAt,
	}
	require.NoError(t, db.Create(row).Error)
	return row
}

// mcl7905DeviceSeq 自增序号(ip_address 有唯一索引,逐台分配)。
var mcl7905DeviceSeq int

// mcl7905SeedDevice 落一台网络设备。
func mcl7905SeedDevice(t *testing.T, db *gorm.DB, name string, vendor models.DeviceVendor, devType models.DeviceType, status models.DeviceStatus, deptID *string) *models.NetworkDevice {
	t.Helper()
	mcl7905DeviceSeq++
	dev := &models.NetworkDevice{
		DeviceName: name,
		DeviceType: devType,
		Vendor:     vendor,
		IPAddress:  fmt.Sprintf("10.79.%d.%d", mcl7905DeviceSeq/250+5, mcl7905DeviceSeq%250+11),
		DeptID:     deptID,
		Status:     status,
	}
	require.NoError(t, db.Create(dev).Error)
	// QUIRK-79-04-D 同根:status 0(DeviceStatusOnline)被列 default:2 的零值跳过吞掉,
	// 建后强制回写目标状态(测试侧规避,零生产改动)。
	if status != models.DeviceStatusUnknown {
		require.NoError(t, db.Model(dev).Update("status", status).Error)
	}
	return dev
}

// -------------------------------------------------------------------------
// 解析纯函数表
// -------------------------------------------------------------------------

// TestMcl7905_ParseMACTable_ByVendor parseMACAddressTable(:382-416)按 vendor+命令类型。
func TestMcl7905_ParseMACTable_ByVendor(t *testing.T) {
	svc, _ := newMcl7905(t)

	t.Run("huawei_mac_table", func(t *testing.T) {
		output := strings.Join([]string{
			"MAC address table of slot 0:",
			"-------------------------------------------------------------------------------",
			"MAC Address    VLAN/VSI   Learned-From        Type",
			"d89e.f327.2d19 100        GE0/0/1             Dynamic",
			"0011.2233.4455 200        GE0/0/2             Static",
			"-------------------------------------------------------------------------------",
			"Total items displayed: 2",
		}, "\n")
		entries, err := svc.parseMACAddressTable(output, models.VendorHuawei, MACCommandTypeMacTable)
		require.NoError(t, err)
		require.Len(t, entries, 2, "标题/分隔/汇总行应被过滤,仅剩 2 条")
		assert.Equal(t, "D8:9E:F3:27:2D:19", entries[0].MACAddress)
		assert.Equal(t, 100, entries[0].VLANID)
		assert.Equal(t, "Dynamic", entries[0].MACType)
		assert.Equal(t, models.VendorHuawei, models.VendorHuawei)
	})

	t.Run("ruijie_mac_table_and_port_security", func(t *testing.T) {
		tableOutput := strings.Join([]string{
			"VLAN  MAC Address       Type    Interface",
			"----  ----------------  -----   ---------",
			"308   b022.7a2e.4a4f    DYNAMIC GigabitEthernet 2/25",
			"309   b022.7a2e.4a50    DYNAMIC GigabitEthernet 2/26",
		}, "\n")
		entries, err := svc.parseMACAddressTable(tableOutput, models.VendorRuijie, MACCommandTypeMacTable)
		require.NoError(t, err)
		require.Len(t, entries, 2)
		assert.Equal(t, "B0:22:7A:2E:4A:4F", entries[0].MACAddress)
		assert.Equal(t, 308, entries[0].VLANID)

		securityOutput := strings.Join([]string{
			"Total Macros: 2",
			"56   308   b022.7a2e.4a4f  GigabitEthernet 2/25      Dynamic            --          active",
		}, "\n")
		secEntries, err := svc.parseMACAddressTable(securityOutput, models.VendorRuijie, MACCommandTypePortSecurity)
		require.NoError(t, err)
		require.Len(t, secEntries, 1, "port-security 路径应走 parseRuijiePortSecurityLine")
		assert.Equal(t, "B0:22:7A:2E:4A:4F", secEntries[0].MACAddress)
		assert.Equal(t, "GE2/25", secEntries[0].InterfaceName, "接口名应被 NormalizeInterfaceName 压短")
	})

	t.Run("unknown_vendor_returns_empty", func(t *testing.T) {
		entries, err := svc.parseMACAddressTable("aa bb cc", models.DeviceVendor("cisco"), MACCommandTypeMacTable)
		require.NoError(t, err, "未知 vendor 走 default 分支,entry 为零值 → 被 canonical 守卫丢弃")
		assert.Empty(t, entries)
	})

	t.Run("empty_output", func(t *testing.T) {
		entries, err := svc.parseMACAddressTable("", models.VendorHuawei, MACCommandTypeMacTable)
		require.NoError(t, err)
		assert.Empty(t, entries, "空输出应返回空 slice(nil 或空)且无错误")
	})

	t.Run("garbage_lines_skipped", func(t *testing.T) {
		output := strings.Join([]string{
			"# comment line",
			"--- separator ---",
			"==== separator ====",
			"Flags: OK",
			"vlan mac-address type interface learned",
			"", // 空行
			"aabbccddeeff 100 GE0/0/9 Dynamic",
		}, "\n")
		entries, err := svc.parseMACAddressTable(output, models.VendorHuawei, MACCommandTypeMacTable)
		require.NoError(t, err)
		require.Len(t, entries, 1, "全部垃圾行应被跳过")
		assert.Equal(t, "AA:BB:CC:DD:EE:FF", entries[0].MACAddress)
	})
}

// TestMcl7905_ParseMACLine_Table parseMACLine(:420-506)三态表。
func TestMcl7905_ParseMACLine_Table(t *testing.T) {
	svc, _ := newMcl7905(t)
	cases := []struct {
		name     string
		line     string
		vendor   models.DeviceVendor
		cmdType  MACCommandType
		wantOK   bool
		wantMAC  string
		wantVLAN int
		wantType string
	}{
		{
			name: "huawei_full", line: "d89e.f327.2d19 100 GigabitEthernet0/0/1 Dynamic",
			vendor: models.VendorHuawei, cmdType: MACCommandTypeMacTable,
			wantOK: true, wantMAC: "D8:9E:F3:27:2D:19", wantVLAN: 100, wantType: "Dynamic",
		},
		{
			name: "huawei_security_suffix_in_iface", line: "001122334455 200 GE0/0/2 security",
			vendor: models.VendorHuawei, cmdType: MACCommandTypeMacTable,
			wantOK: true, wantMAC: "00:11:22:33:44:55", wantVLAN: 200, wantType: "security",
		},
		{
			name: "huawei_too_few_fields", line: "d89e.f327.2d19 100",
			vendor: models.VendorHuawei, cmdType: MACCommandTypeMacTable, wantOK: false,
		},
		{
			name: "ruijie_table", line: "308 b022.7a2e.4a4f DYNAMIC GigabitEthernet 2/25",
			vendor: models.VendorRuijie, cmdType: MACCommandTypeMacTable,
			wantOK: true, wantMAC: "B0:22:7A:2E:4A:4F", wantVLAN: 308, wantType: "DYNAMIC",
		},
		{
			name: "ruijie_port_security_delegates", line: "56 308 b022.7a2e.4a4f GigabitEthernet 2/25 Dynamic -- active",
			vendor: models.VendorRuijie, cmdType: MACCommandTypePortSecurity,
			wantOK: true, wantMAC: "B0:22:7A:2E:4A:4F", wantVLAN: 308, wantType: "Dynamic",
		},
		{
			name: "maipu_non_numeric_vlan_still_parses_mac", line: "xx 001122334455 STATIC GE0/0/3",
			vendor: models.VendorMaipu, cmdType: MACCommandTypeMacTable,
			wantOK: true, wantMAC: "00:11:22:33:44:55", wantVLAN: 0, wantType: "STATIC",
		},
		{
			name: "garbage_words_rejected_by_canonical_guard", line: "Flags: OK",
			vendor: models.VendorHuawei, cmdType: MACCommandTypeMacTable, wantOK: false,
		},
		{
			name: "empty_line", line: "",
			vendor: models.VendorHuawei, cmdType: MACCommandTypeMacTable, wantOK: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			entry, ok := svc.parseMACLine(tc.line, tc.vendor, tc.cmdType)
			assert.Equal(t, tc.wantOK, ok)
			if tc.wantOK {
				assert.Equal(t, tc.wantMAC, entry.MACAddress)
				assert.Equal(t, tc.wantVLAN, entry.VLANID)
				assert.Equal(t, tc.wantType, entry.MACType)
			}
		})
	}
}

// TestMcl7905_ParseRuijiePortSecurityLine :518-554 两态表。
func TestMcl7905_ParseRuijiePortSecurityLine(t *testing.T) {
	svc, _ := newMcl7905(t)
	cases := []struct {
		name    string
		fields  []string
		wantOK  bool
		wantMAC string
	}{
		{"seven_fields_min", strings.Fields("56 308 b022.7a2e.4a4f GigabitEthernet Dynamic -- active"), true, "B0:22:7A:2E:4A:4F"},
		{"eight_fields_multi_token_iface", strings.Fields("56 308 b022.7a2e.4a4f GigabitEthernet 2/25 Dynamic -- active"), true, "B0:22:7A:2E:4A:4F"},
		{"six_fields_rejected", strings.Fields("56 308 b022.7a2e.4a4f GigabitEthernet Dynamic active"), false, ""},
		{"non_numeric_vlan_rejected", strings.Fields("56 VLAN b022.7a2e.4a4f GE0/0/1 Dynamic -- active"), false, ""},
		{"garbage_mac_rejected", strings.Fields("56 308 Flags: GE0/0/1 Dynamic -- active"), false, ""},
		{"dot_separated_mac_normalized", strings.Fields("1 1 0000.1111.2222 GE0/0/1 Dynamic -- active"), true, "00:00:11:11:22:22"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			entry, ok := svc.parseRuijiePortSecurityLine(tc.fields)
			assert.Equal(t, tc.wantOK, ok)
			if tc.wantOK {
				assert.Equal(t, tc.wantMAC, entry.MACAddress)
			}
		})
	}
}

// TestMcl7905_MergeMACEntries mergeMACEntries(:559-573)按 (MAC,VLAN,Interface) 去重。
func TestMcl7905_MergeMACEntries(t *testing.T) {
	svc, _ := newMcl7905(t)

	bucketA := []MACAddressEntry{
		{MACAddress: "AA:BB:CC:00:00:01", InterfaceName: "GE0/0/1", VLANID: 100},
		{MACAddress: "AA:BB:CC:00:00:02", InterfaceName: "GE0/0/2", VLANID: 100},
	}
	bucketB := []MACAddressEntry{
		// 与 bucketA[0] 三元组完全一致 → 应去重
		{MACAddress: "AA:BB:CC:00:00:01", InterfaceName: "GE0/0/1", VLANID: 100},
		// 同 MAC 同接口但 VLAN 不同 → 保留
		{MACAddress: "AA:BB:CC:00:00:01", InterfaceName: "GE0/0/1", VLANID: 200},
	}

	merged := svc.mergeMACEntries([][]MACAddressEntry{bucketA, bucketB})
	require.Len(t, merged, 3, "4 条输入去重后应剩 3(重复三元组只留首条)")
	// 顺序语义:保持出现序,首见在前
	assert.Equal(t, "AA:BB:CC:00:00:01", merged[0].MACAddress)
	assert.Equal(t, 100, merged[0].VLANID)
	assert.Equal(t, "AA:BB:CC:00:00:02", merged[1].MACAddress)
	assert.Equal(t, 200, merged[2].VLANID)

	assert.Empty(t, svc.mergeMACEntries(nil), "空输入 → 空")
}

// TestMcl7905_CleanTimestampFromInterface cleanTimestampFromInterface(:581-590)表驱动。
func TestMcl7905_CleanTimestampFromInterface(t *testing.T) {
	svc, _ := newMcl7905(t)
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"with_timestamp", "GigabitEthernet 0/12 2026-5-9 0:51:07", "GigabitEthernet 0/12"},
		{"with_padded_timestamp", "GE0/0/1 2026-05-09 08:52:15", "GE0/0/1"},
		{"clean_iface", "GE0/0/1", "GE0/0/1"},
		{"only_timestamp_returns_empty", "2026-5-9 0:51:07", ""},
		{"empty_string", "", ""},
		{"multi_space_after_clean", "XGE0/1   2026-05-09 08:52:15", "XGE0/1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, svc.cleanTimestampFromInterface(tc.in))
		})
	}
}

// -------------------------------------------------------------------------
// 命令选择 + 配置
// -------------------------------------------------------------------------

// TestMcl7905_GetMACCommand_Vendors getMACCommand/getMACCommands(:355-377)。
func TestMcl7905_GetMACCommand_Vendors(t *testing.T) {
	svc, _ := newMcl7905(t)

	t.Run("huawei_h3c_single", func(t *testing.T) {
		for _, vendor := range []models.DeviceVendor{models.VendorHuawei, models.VendorH3C} {
			cmds := svc.getMACCommands(vendor)
			require.Len(t, cmds, 1, "%s 应单命令", vendor)
			assert.Equal(t, "display mac-address", cmds[0].Cmd)
			assert.Equal(t, MACCommandTypeMacTable, cmds[0].Type)
			assert.Equal(t, "display mac-address", svc.getMACCommand(vendor))
		}
	})

	t.Run("ruijie_maipu_dual", func(t *testing.T) {
		for _, vendor := range []models.DeviceVendor{models.VendorRuijie, models.VendorMaipu} {
			cmds := svc.getMACCommands(vendor)
			require.Len(t, cmds, 2, "%s 应双命令(mac-table + port-security)", vendor)
			assert.Equal(t, "show mac-address-table", cmds[0].Cmd)
			assert.Equal(t, MACCommandTypeMacTable, cmds[0].Type)
			assert.Equal(t, "show port-security address", cmds[1].Cmd)
			assert.Equal(t, MACCommandTypePortSecurity, cmds[1].Type)
			assert.Equal(t, "show mac-address-table", svc.getMACCommand(vendor), "首条命令向后兼容")
		}
	})

	t.Run("unknown_vendor_fallback", func(t *testing.T) {
		cmds := svc.getMACCommands(models.DeviceVendor("cisco"))
		require.Len(t, cmds, 1)
		assert.Equal(t, "show mac-address-table", cmds[0].Cmd)
		assert.Equal(t, "show mac-address-table", svc.getMACCommand(models.DeviceVendor("cisco")))
	})
}

// TestMcl7905_GetMACThreshold_NilFilterRule nil filterRuleService 的硬编码阈值表
// (既有 TestGetMACThreshold 覆盖同一分支,这里补设备类型落库形态,不重写既有口径)。
func TestMcl7905_GetMACThreshold_NilFilterRule(t *testing.T) {
	svc, db := newMcl7905(t)
	assert.Nil(t, svc.filterRuleService, "装配前提:filterRuleService 为 nil")

	dev := mcl7905SeedDevice(t, db, "sw-7905", models.VendorHuawei, models.DeviceTypeSwitch, models.DeviceStatusOnline, nil)
	assert.Equal(t, 10, svc.getMACThreshold(dev), "交换机默认阈值 10")

	dev.DeviceType = models.DeviceTypeAP
	assert.Equal(t, 10, svc.getMACThreshold(dev), "未知类型默认阈值 10")
}

// TestMcl7905_LoadConfigFromDB_And_ReloadConfig loadConfigFromDB/ReloadConfig(:779-793)。
func TestMcl7905_LoadConfigFromDB_And_ReloadConfig(t *testing.T) {
	ctx := context.Background()
	svc, db := newMcl7905(t)

	t.Run("seeded_value_applied", func(t *testing.T) {
		require.NoError(t, db.Create(&models.Config{
			ConfigName:  "MAC采集并发",
			ConfigKey:   macConcurrentConfigKey,
			ConfigValue: "5",
			ConfigType:  models.ConfigTypeYes,
		}).Error)
		svc.ReloadConfig()
		assert.Equal(t, 5, svc.maxConcurrent, "ReloadConfig 后应取 DB 值 5")
	})

	t.Run("invalid_values_keep_default", func(t *testing.T) {
		for _, bad := range []string{"abc", "0", "-3"} {
			require.NoError(t, db.Model(&models.Config{}).
				Where("config_key = ?", macConcurrentConfigKey).
				Update("config_value", bad).Error)
			svc.ReloadConfig()
			assert.Equal(t, 5, svc.maxConcurrent, "非法值 %q 不应改变既有并发数", bad)
		}
	})

	t.Run("hot_reload_picks_up_new_value", func(t *testing.T) {
		require.NoError(t, db.Model(&models.Config{}).
			Where("config_key = ?", macConcurrentConfigKey).
			Update("config_value", "42").Error)
		assert.Equal(t, 5, svc.maxConcurrent, "热载前仍是旧值")
		svc.ReloadConfig()
		assert.Equal(t, 42, svc.maxConcurrent, "热载后取新值")
		_ = ctx
	})
}

// -------------------------------------------------------------------------
// CRUD / 统计 / 清理
// -------------------------------------------------------------------------

// TestMcl7905_MACAddressList_Pagination GetMACAddressList(:612-703)分页 + 过滤 + 排序白名单。
func TestMcl7905_MacAddressList_Pagination(t *testing.T) {
	ctx := context.Background()
	svc, db := newMcl7905(t)

	deptID := "11111111-1111-1111-1111-111111111111"
	otherDept := "22222222-2222-2222-2222-222222222222"
	devA := mcl7905SeedDevice(t, db, "dev-1-7905", models.VendorHuawei, models.DeviceTypeSwitch, models.DeviceStatusOnline, &deptID)
	devB := mcl7905SeedDevice(t, db, "dev-2-7905", models.VendorRuijie, models.DeviceTypeSwitch, models.DeviceStatusOnline, &otherDept)

	collected := time.Date(2026, 3, 10, 8, 0, 0, 0, time.UTC)
	for i := 0; i < 3; i++ {
		mcl7905SeedMAC(t, db, devA.ID, mcl7905MAC(i), "GE0/0/1", mhq7905Int(100), models.MACTypeDynamic, collected.Add(time.Duration(i)*time.Minute))
	}
	mcl7905SeedMAC(t, db, devB.ID, mcl7905MAC(9), "GE0/0/2", mhq7905Int(200), models.MACTypeStatic, collected)

	t.Run("all", func(t *testing.T) {
		list, total, err := svc.GetMACAddressList(ctx, 1, 10, "", "", "", "", "", nil)
		require.NoError(t, err)
		assert.Equal(t, int64(4), total)
		require.Len(t, list, 4)
		assert.Equal(t, devA.ID, list[0].DeviceID)
		assert.Equal(t, devA.DeviceName, list[0].DeviceName, "JOIN 应带出设备名")
	})

	t.Run("device_id_filter", func(t *testing.T) {
		list, total, err := svc.GetMACAddressList(ctx, 1, 10, devA.ID, "", "", "", "", nil)
		require.NoError(t, err)
		assert.Equal(t, int64(3), total)
		require.Len(t, list, 3)
	})

	t.Run("dept_id_filter", func(t *testing.T) {
		list, total, err := svc.GetMACAddressList(ctx, 1, 10, "", deptID, "", "", "", nil)
		require.NoError(t, err)
		assert.Equal(t, int64(3), total, "部门子查询应只命中 devA 的 3 条")
		require.Len(t, list, 3)
	})

	t.Run("mac_address_filter_full", func(t *testing.T) {
		// 完整 MAC 归一化后 LIKE 包含匹配
		list, total, err := svc.GetMACAddressList(ctx, 1, 10, "", "", mcl7905MAC(9), "", "", nil)
		require.NoError(t, err)
		assert.Equal(t, int64(1), total)
		require.Len(t, list, 1)
		assert.Equal(t, devB.ID, list[0].DeviceID)
	})

	t.Run("mac_address_filter_partial_matches_all", func(t *testing.T) {
		//QUIRK-79-05-G(锁定): NormalizeMACAddress 对非 12 hex 的部分输入(如 "00:00:09")
		// 返回空串 → 拼出 LIKE '%%' → 命中全部行,且不报错。前端必须传完整 MAC。
		list, total, err := svc.GetMACAddressList(ctx, 1, 10, "", "", "00:00:09", "", "", nil)
		require.NoError(t, err)
		assert.Equal(t, int64(4), total, "部分 MAC 输入退化为全表匹配(QUIRK-79-05-G)")
		assert.Len(t, list, 4)
	})

	t.Run("interface_name_filter", func(t *testing.T) {
		list, total, err := svc.GetMACAddressList(ctx, 1, 10, "", "", "", "GE0/0/2", "", nil)
		require.NoError(t, err)
		assert.Equal(t, int64(1), total)
		require.Len(t, list, 1)
		assert.Equal(t, "GE0/0/2", list[0].InterfaceName)
	})

	t.Run("pagination_slice", func(t *testing.T) {
		list, total, err := svc.GetMACAddressList(ctx, 2, 2, devA.ID, "", "", "", "", nil)
		require.NoError(t, err)
		assert.Equal(t, int64(3), total)
		assert.Len(t, list, 1, "第 2 页(每页 2)应剩 1 条")
	})

	t.Run("order_by_whitelist_asc", func(t *testing.T) {
		asc := true
		list, total, err := svc.GetMACAddressList(ctx, 1, 10, devA.ID, "", "", "", "collectedAt", &asc)
		require.NoError(t, err)
		assert.Equal(t, int64(3), total)
		require.Len(t, list, 3)
		assert.True(t, !list[0].CollectedAt.After(list[2].CollectedAt), "ASC 应首条最早")
	})

	t.Run("order_by_whitelist_desc", func(t *testing.T) {
		asc := false
		list, _, err := svc.GetMACAddressList(ctx, 1, 10, devA.ID, "", "", "", "macAddress", &asc)
		require.NoError(t, err)
		require.Len(t, list, 3)
		assert.GreaterOrEqual(t, list[0].MACAddress, list[2].MACAddress, "DESC 应首条最大")
	})

	t.Run("invalid_order_column_falls_back", func(t *testing.T) {
		//QUIRK-79-05-E(锁定): 非法 orderByColumn 走 ApplySort 白名单回退(warn 不改查询),
		// 又因源码仅在 orderByColumn=="" 时补默认 collected_at DESC → 非法列退化为 sqlite 自然序
		// (与 QUIRK-79-02-A / 79-03-B 同族)。
		list, total, err := svc.GetMACAddressList(ctx, 1, 10, devA.ID, "", "", "", "DROP TABLE", nil)
		require.NoError(t, err)
		assert.Equal(t, int64(3), total)
		assert.Len(t, list, 3, "非法排序列不应报错也不应注入")
	})

	t.Run("empty_result_shape", func(t *testing.T) {
		list, total, err := svc.GetMACAddressList(ctx, 1, 10, "99999999-9999-9999-9999-999999999999", "", "", "", "", nil)
		require.NoError(t, err)
		assert.Zero(t, total)
		assert.NotNil(t, list)
		assert.Empty(t, list)
	})
}

// TestMcl7905_MacStats_Clean_BatchDelete Stats(:715-752)/CleanOldRecords(:755-767)/BatchDelete(:770-776)。
func TestMcl7905_MacStats_Clean_BatchDelete(t *testing.T) {
	ctx := context.Background()
	svc, db := newMcl7905(t)
	deviceID := "aaaaaaaa-1111-1111-1111-111111111111"

	// CleanOldRecords 内部用 time.Now 计算阈值 → 种子只能相对当前时刻构造
	// (200 天前 vs 100 天阈值,宽边距无跨日 flake);截断到秒避免存储精度抖动。
	now := time.Now().Truncate(time.Second)
	old := now.AddDate(0, 0, -200)
	mcl7905SeedMAC(t, db, deviceID, mcl7905MAC(1), "GE0/0/1", mhq7905Int(100), models.MACTypeDynamic, now)
	mcl7905SeedMAC(t, db, deviceID, mcl7905MAC(2), "GE0/0/2", mhq7905Int(100), models.MACTypeStatic, now)
	mcl7905SeedMAC(t, db, deviceID, mcl7905MAC(3), "GE0/0/3", mhq7905Int(100), models.MACTypeSecure, now)
	mcl7905SeedMAC(t, db, deviceID, mcl7905MAC(4), "GE0/0/1", mhq7905Int(100), models.MACTypeDynamic, old)

	t.Run("stats", func(t *testing.T) {
		stats, err := svc.GetMACAddressStats(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(4), stats["totalRecords"])
		assert.Equal(t, int64(1), stats["uniqueDevices"])
		assert.Equal(t, int64(4), stats["uniqueMACs"])
		assert.Equal(t, int64(2), stats["dynamic"])
		assert.Equal(t, int64(1), stats["static"])
		assert.Equal(t, int64(1), stats["secure"])
		latest, ok := stats["latestCollection"].(time.Time)
		require.True(t, ok, "latestCollection 应为 time.Time 形态")
		assert.True(t, latest.Equal(now), "应取 collected_at 最新一条")
	})

	t.Run("clean_old_records", func(t *testing.T) {
		deleted, err := svc.CleanOldRecords(ctx, 100)
		require.NoError(t, err)
		assert.Equal(t, int64(1), deleted, "只应清理 200 天前那条(100 天阈值)")

		stats, err := svc.GetMACAddressStats(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(3), stats["totalRecords"])
	})

	t.Run("clean_nothing", func(t *testing.T) {
		deleted, err := svc.CleanOldRecords(ctx, 10000)
		require.NoError(t, err)
		assert.Zero(t, deleted, "阈值大于全部记录年龄 → 0")
	})

	t.Run("batch_delete", func(t *testing.T) {
		list, _, err := svc.GetMACAddressList(ctx, 1, 10, deviceID, "", "", "", "", nil)
		require.NoError(t, err)
		ids := make([]string, 0, 2)
		for _, r := range list {
			ids = append(ids, r.ID)
			if len(ids) == 2 {
				break
			}
		}
		deleted, err := svc.BatchDelete(ctx, ids)
		require.NoError(t, err)
		assert.Equal(t, int64(2), deleted)

		stats, err := svc.GetMACAddressStats(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(1), stats["totalRecords"])
	})

	t.Run("batch_delete_empty_ids", func(t *testing.T) {
		deleted, err := svc.BatchDelete(ctx, []string{})
		require.NoError(t, err)
		assert.Zero(t, deleted, "空 ID 集不删任何行")
	})
}

// -------------------------------------------------------------------------
// executor nil 边界(collectDeviceMAC 真路径归 79-06)
// -------------------------------------------------------------------------

// TestMcl7905_Collect_NilExecutor executor=nil 下的采集入口行为。
//
// QUIRK-79-05-F(锁定): collectDeviceMAC 顶层有 panic-recovery(:118-122),
// executor 为 nil 时 `s.executor.ExecuteOnDevice(...)` 的 nil 指针解引用被 recover 吞掉,
// 函数在 recover 后返回 nil result(不走正常 return 路径)→
// CollectAllDevices 返回 []*MACCollectionResult{nil},CollectDevice 返回 (nil, nil)。
// executor 真路径(ExecuteOnDevice/ExecuteMultipleOnDevice/解析/过滤/入库/历史)归 79-06。
func TestMcl7905_Collect_NilExecutor(t *testing.T) {
	ctx := context.Background()
	svc, db := newMcl7905(t)

	t.Run("collect_all_no_online_devices", func(t *testing.T) {
		mcl7905SeedDevice(t, db, "dev-off-1", models.VendorHuawei, models.DeviceTypeSwitch, models.DeviceStatusOffline, nil)
		results, err := svc.CollectAllDevices(ctx)
		require.Error(t, err, "没有在线设备时应报错")
		assert.Nil(t, results)
		assert.Contains(t, err.Error(), "没有在线设备")
	})

	t.Run("collect_all_with_online_device_recovers_nil_executor", func(t *testing.T) {
		mcl7905SeedDevice(t, db, "dev-on-1", models.VendorHuawei, models.DeviceTypeSwitch, models.DeviceStatusOnline, nil)
		results, err := svc.CollectAllDevices(ctx)
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Nil(t, results[0], "QUIRK-79-05-F:recover 后 collectDeviceMAC 返回 nil,result 追加 nil")
	})

	t.Run("collect_device_missing", func(t *testing.T) {
		_, err := svc.CollectDevice(ctx, "99999999-9999-9999-9999-999999999999")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "设备不存在")
	})

	t.Run("collect_device_existing_recovers_nil_executor", func(t *testing.T) {
		dev := mcl7905SeedDevice(t, db, "dev-on-2", models.VendorRuijie, models.DeviceTypeSwitch, models.DeviceStatusOnline, nil)
		result, err := svc.CollectDevice(ctx, dev.ID)
		require.NoError(t, err)
		assert.Nil(t, result, "QUIRK-79-05-F:recover 后返回 nil result 且不报错")
	})
}
