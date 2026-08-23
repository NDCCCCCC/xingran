package device

import (
	"context"
	"errors"
	"fmt"
	"net"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// =====================================================================
// 74-08 Batch A: internal/device 纯函数面 — model_extractor 型号/厂商/
// 类型识别 + snmp_client 辅助函数(ScanIPRange 参数/PingCheck/端口转换/
// Detect*)+ snmp_entity_mib stub SNMPGetter + Manager sqlite 查询路径 +
// executor 禁用池快速失败 + scrapli 纯函数。
// (真实 SSH/SNMP 网络路径不在单测范围 — ScrapliWrapper 持具体 scrapligo
//  *network.Driver,无法注入;错误路径在 portcollection 已覆盖)
// =====================================================================

// ---------------- model_extractor.go ----------------

func TestModelExtractor_Extract(t *testing.T) {
	cases := []struct {
		name     string
		sysDescr string
		vendor   models.DeviceVendor
		want     string
	}{
		{"空输入", "", models.VendorHuawei, ""},
		{"华为 S 系列-串首", "S5735-L48T4X-A Huawei Versatile Routing Platform", models.VendorHuawei, "S5735-L48T4X-A"},
		{"华为 S 系列-行首", "Huawei Platform\nS5735-L48T4X-A\nCopyright", models.VendorHuawei, "S5735-L48T4X-A"},
		{"华为 AR-串首", "AR2220 Router Software", models.VendorHuawei, "AR2220"},
		{"华为 S 系列-空格分隔", "Huawei Versatile S5735-L48T4X-A Platform", models.VendorHuawei, "S5735-L48T4X-A"},
		{"华为 USG-串首", "USG6000E Security Gateway", models.VendorHuawei, "USG6000E"},
		{"华为 USG-尾字母+后缀", "USG6680E-AI Firewall", models.VendorHuawei, "USG6680E-AI"},
		{"华为 USG-无尾字母", "USG6000 Gateway", models.VendorHuawei, "USG6000"},
		{"华为 USG-空格分隔", "Huawei USG6000E Security Gateway", models.VendorHuawei, "USG6000E"},
		{"华为无匹配", "Huawei Versatile Routing Platform Software", models.VendorHuawei, ""},
		{"H3C S 系列-串首", "S5120-28P-SI H3C Comware Software", models.VendorH3C, "S5120-28P-SI"},
		{"H3C MSR-串首", "MSR3640 H3C Comware V7", models.VendorH3C, "MSR3640"},
		{"H3C S 系列-行首", "H3C Comware\nS5120-28P-SI Software", models.VendorH3C, "S5120-28P-SI"},
		{"锐捷 RG-S-串首", "RG-S5750-28GT-P-S Ruijie RGOS 11.4", models.VendorRuijie, "RG-S5750-28GT-P-S"},
		{"锐捷 RG-S-行首", "RGOS\nRG-S5750-28GT-P-S", models.VendorRuijie, "RG-S5750-28GT-P-S"},
		{"锐捷 RSR-串首", "RSR20-04 Ruijie RSR Routing OS", models.VendorRuijie, "RSR20-04"},
		{"迈普 MP-串首", "MP2800 Maipu Router", models.VendorMaipu, "MP2800"},
		{"未知厂商通用提取-串首", "S5700-X Cisco-like System", "", "S5700-X"},
		{"未知厂商无匹配", "plain text no model", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := NewModelExtractor(tc.sysDescr, tc.vendor).Extract()
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestModelExtractor_DualPath_DiscoveryFallback(t *testing.T) {
	// 双路径回归:发现链先走 NewModelExtractor().Extract(),再回退 ExtractModelFromSysDescr()。
	// 修复前,行首/空格分隔样本会让新提取器返回 "",实际落库 model 依赖 :611 回退结果;
	// 修复后两条路径对同一样本结果一致,部分设备落库 model 将从"回退结果"变"新提取器结果"。
	sysDescr := "Huawei Platform\nS5735-L48T4X-A\nCopyright"
	want := "S5735-L48T4X-A"

	gotNew := NewModelExtractor(sysDescr, models.VendorHuawei).Extract()
	gotOld := ExtractModelFromSysDescr(sysDescr, models.VendorHuawei)

	assert.Equal(t, want, gotNew, "NewModelExtractor 应提取行首型号")
	assert.Equal(t, want, gotOld, "ExtractModelFromSysDescr 回退路径结果应一致")
}

func TestIdentifyVendor(t *testing.T) {
	assert.Equal(t, models.VendorHuawei, IdentifyVendor("Huawei Versatile Routing Platform"))
	assert.Equal(t, models.VendorHuawei, IdentifyVendor("HUAWEI S5735"))
	assert.Equal(t, models.VendorH3C, IdentifyVendor("H3C Comware"))
	assert.Equal(t, models.VendorRuijie, IdentifyVendor("Ruijie RG-S5750"))
	assert.Equal(t, models.VendorMaipu, IdentifyVendor("Maipu MyPower"))
	assert.Equal(t, models.DeviceVendor(""), IdentifyVendor("Cisco IOS Software"))
	assert.Equal(t, models.DeviceVendor(""), IdentifyVendor(""))
}

func TestIdentifyDeviceType(t *testing.T) {
	assert.Equal(t, models.DeviceTypeRouter, IdentifyDeviceType("AR2220 Router Software"))
	assert.Equal(t, models.DeviceTypeSwitch, IdentifyDeviceType("S5735 Switch Software"))
	assert.Equal(t, models.DeviceTypeFirewall, IdentifyDeviceType("USG6000 Firewall"))
	assert.Equal(t, models.DeviceTypeFirewall, IdentifyDeviceType("USG6300E Gateway"))
	assert.Equal(t, models.DeviceTypeAP, IdentifyDeviceType("AirEngine Access Point"))
	assert.Equal(t, models.DeviceTypeLoadBalancer, IdentifyDeviceType("Load Balancer LB2000"))
	assert.Equal(t, models.DeviceTypeSwitch, IdentifyDeviceType("unrecognized junk"), "默认交换机")
}

// ---------------- snmp_client.go 纯辅助 ----------------

func TestToUpperContainsHelpers(t *testing.T) {
	assert.Equal(t, "ABC123", toUpper("abc123"))
	assert.Equal(t, "ABC", toUpper("ABC"))

	assert.True(t, contains("Hello RG-S5750", "rg-s5750"), "contains 实为忽略大小写")
	assert.False(t, contains("hello", "xyz"))

	assert.True(t, containsIgnoreCase("Hello World", "WORLD"))
	assert.False(t, containsIgnoreCase("hello", "xyz"))

	assert.Equal(t, 1, indexOfIgnoreCase("hello", "EL"))
	assert.Equal(t, -1, indexOfIgnoreCase("hello", "zz"))
}

func TestIsDigitUpper(t *testing.T) {
	assert.True(t, isDigit('5'))
	assert.False(t, isDigit('a'))
	assert.True(t, isUpper('A'))
	assert.False(t, isUpper('a'))
	assert.True(t, isDigitRune('7'))
	assert.False(t, isDigitRune('x'))
	assert.True(t, isUpperRune('Z'))
	assert.False(t, isUpperRune('z'))
}

func TestPingCheck(t *testing.T) {
	// 本机回环上无法保证 ICMP 权限,只验证 API 稳定不 panic
	_ = PingCheck("192.0.2.1", 50*time.Millisecond) // TEST-NET,预期 false
}

func TestScanIPRange_InvalidInputs(t *testing.T) {
	// 非法 IP → 空结果
	assert.Empty(t, ScanIPRange("not-an-ip", "1.2.3.4", 10*time.Millisecond))
	assert.Empty(t, ScanIPRange("1.2.3.4", "also-bad", 10*time.Millisecond))

	// 起点大于终点 → 空结果(不 panic)
	assert.NotPanics(t, func() {
		_ = ScanIPRange("192.0.2.10", "192.0.2.9", 10*time.Millisecond)
	})
}

func TestIPHelpers(t *testing.T) {
	ip := net.ParseIP("192.0.2.1")
	assert.Equal(t, uint32(0xC0000201), ipToUint32(ip))

	next := nextIP(net.ParseIP("192.0.2.1"))
	assert.Equal(t, "192.0.2.2", next.String())

	// 进位
	next2 := nextIP(net.ParseIP("192.0.2.255"))
	assert.Equal(t, "192.0.3.0", next2.String())

	// 全 255 → 返回非 nil 全零形态 IP(QUIRK D-12: net.IP.Equal 4/16 字节形态差异使
	// 函数内 IPv4zero 判定失效;ScanIPRange 的 ipToUint32(0) 比较使其终止,不发散)
	last := nextIP(net.ParseIP("255.255.255.255"))
	require.NotNil(t, last)
}

func TestDetectVendorAndDeviceType(t *testing.T) {
	assert.Equal(t, models.VendorHuawei, DetectVendor("Huawei S5735"))
	assert.Equal(t, models.VendorH3C, DetectVendor("H3C Comware 7"))
	assert.Equal(t, models.VendorH3C, DetectVendor("HP ProCurve Switch"))
	assert.Equal(t, models.VendorH3C, DetectVendor("3Com SuperStack"))
	assert.Equal(t, models.VendorRuijie, DetectVendor("Ruijie RGOS"))
	assert.Equal(t, models.VendorMaipu, DetectVendor("Maipu MyPower"))
	assert.Equal(t, models.DeviceVendor(""), DetectVendor("Cisco IOS"))
	assert.Equal(t, models.DeviceVendor(""), DetectVendor(""))

	assert.Equal(t, models.DeviceTypeRouter, DetectDeviceType("MSR3640 Software"))
	assert.Equal(t, models.DeviceTypeSwitch, DetectDeviceType("S5735 SI Switch"))
}

func TestConvertPortToInt(t *testing.T) {
	v, err := ConvertPortToInt("22")
	require.NoError(t, err)
	assert.Equal(t, 22, v)

	_, err = ConvertPortToInt("abc")
	assert.ErrorContains(t, err, "无效的端口号")

	_, err = ConvertPortToInt("0")
	assert.ErrorContains(t, err, "超出范围")

	_, err = ConvertPortToInt("70000")
	assert.ErrorContains(t, err, "超出范围")
}

// ---------------- snmp_entity_mib.go (stub SNMPGetter) ----------------

type stubGetter struct {
	values map[string]interface{}
	errs   map[string]error
	walk   map[string][]struct {
		oid string
		val interface{}
	}
	walkErr error
}

func (s *stubGetter) Get(oid string) (interface{}, error) {
	if err, ok := s.errs[oid]; ok {
		return nil, err
	}
	v, ok := s.values[oid]
	if !ok {
		return nil, fmt.Errorf("noSuchObject")
	}
	return v, nil
}

func (s *stubGetter) Walk(oid string, cb func(string, interface{}) bool) error {
	if s.walkErr != nil {
		return s.walkErr
	}
	for _, kv := range s.walk[oid] {
		if !cb(kv.oid, kv.val) {
			break
		}
	}
	return nil
}

func TestCountPhysicalEntitiesByClass(t *testing.T) {
	stub := &stubGetter{
		walk: map[string][]struct {
			oid string
			val interface{}
		}{
			OidEntPhysicalClass: {
				{"." + OidEntPhysicalClass + ".10", 3}, // chassis
				{"." + OidEntPhysicalClass + ".11", 6}, // powerSupply(不在类集)
				{"." + OidEntPhysicalClass + ".12", 3},
				{"." + OidEntPhysicalClass + ".bad", 3}, // 非数字后缀 → 跳过
				{".other.prefix.13", 3},                 // 前缀不符 → 跳过
				{"." + OidEntPhysicalClass + ".14", "not-int"},
			},
		},
	}

	got, err := CountPhysicalEntitiesByClass(stub, []int{3})
	require.NoError(t, err)
	assert.Len(t, got, 2)
	assert.Contains(t, got, 10)
	assert.Contains(t, got, 12)

	// Walk 失败 → 错误
	stubErr := &stubGetter{walkErr: errors.New("snmp timeout")}
	_, err = CountPhysicalEntitiesByClass(stubErr, []int{3})
	assert.ErrorContains(t, err, "walk entPhysicalClass")
}

func TestGetEntityAttrs(t *testing.T) {
	stub := &stubGetter{values: map[string]interface{}{
		OidEntPhysicalSerialNum + ".5":     []byte(" SN-001 "),
		OidEntPhysicalModelName + ".5":     "S5735-L48T4X-A",
		OidEntPhysicalName + ".5":          "Board 5",
		OidEntPhysicalClass + ".5":         int64(3),
		OidEntPhysicalContainedIn + ".5":   int32(1),
	}}
	a, err := GetEntityAttrs(stub, 5)
	require.NoError(t, err)
	assert.Equal(t, "SN-001", a.Serial, "TrimSpace 生效")
	assert.Equal(t, "S5735-L48T4X-A", a.Model)
	assert.Equal(t, "Board 5", a.Name)
	assert.Equal(t, 3, a.Class)
	assert.Equal(t, 1, a.ContainedIn)

	// idx<=0 → 错误
	_, err = GetEntityAttrs(stub, 0)
	assert.ErrorContains(t, err, "invalid entity index")

	// 部分 OID 错误容忍(errs 优先于 values)
	stub2 := &stubGetter{
		values: map[string]interface{}{
			OidEntPhysicalName + ".5": "OnlyName",
		},
		errs: map[string]error{
			OidEntPhysicalSerialNum + ".5": errors.New("noSuchObject"),
		},
	}
	a2, err := GetEntityAttrs(stub2, 5)
	require.NoError(t, err)
	assert.Empty(t, a2.Serial, "错误容忍,字段零值")
	assert.Equal(t, "OnlyName", a2.Name)
}

func TestExtractIndexFromOID(t *testing.T) {
	base := OidEntPhysicalClass
	assert.Equal(t, 42, extractIndexFromOID("."+base+".42", base))
	assert.Equal(t, 7, extractIndexFromOID("."+base+".7.1.2", base), "复合键取首段")
	assert.Equal(t, 0, extractIndexFromOID("."+base+".abc", base), "非数字 → 0")
	assert.Equal(t, 0, extractIndexFromOID("1.2.3.4", base), "前缀不符 → 0")
	assert.Equal(t, 9, extractIndexFromOID(base+".9", base), "无前导点但前缀匹配(两侧都 Trim 前导点)")
}

func TestToStringAndToInt(t *testing.T) {
	assert.Equal(t, "str", toString("str"))
	assert.Equal(t, "bytes", toString([]byte("bytes")))
	assert.Equal(t, "", toString(nil))
	assert.Equal(t, "42", toString(42))

	v, ok := toInt(7)
	assert.True(t, ok)
	assert.Equal(t, 7, v)
	v, ok = toInt(int32(-3))
	assert.True(t, ok)
	assert.Equal(t, -3, v)
	v, ok = toInt(int64(99))
	assert.True(t, ok)
	assert.Equal(t, 99, v)
	v, ok = toInt(uint64(5))
	assert.True(t, ok)
	assert.Equal(t, 5, v)
	v, ok = toInt("123")
	assert.True(t, ok)
	assert.Equal(t, 123, v)
	_, ok = toInt("abc")
	assert.False(t, ok)
	_, ok = toInt(3.14)
	assert.False(t, ok)
}

// ---------------- scrapli_wrapper.go 纯函数 ----------------

func TestPlatformName(t *testing.T) {
	assert.Equal(t, "huawei_vrp", PlatformName(models.VendorHuawei))
	assert.Equal(t, "hp_comware", PlatformName(models.VendorH3C))
	assert.Equal(t, "ruijie_rjos", PlatformName(models.VendorRuijie))
	assert.Equal(t, "cisco_iosxe", PlatformName(models.VendorMaipu))
	assert.Equal(t, "cisco_iosxe", PlatformName("unknown"), "默认 cisco_iosxe")
}

func TestPlatformIdentifier(t *testing.T) {
	// 锐捷 → patched YAML 字节
	id := platformIdentifier(models.VendorRuijie)
	bytes, ok := id.([]byte)
	require.True(t, ok, "锐捷返回 []byte patched yaml")
	assert.NotEmpty(t, bytes)

	// 其他 → 平台名字符串
	assert.Equal(t, "huawei_vrp", platformIdentifier(models.VendorHuawei))
}

func TestCheckDeviceReachable(t *testing.T) {
	// 不可达端口(127.0.0.1:1 拒连)→ 快速失败
	err := checkDeviceReachable("127.0.0.1", 1, 500*time.Millisecond)
	assert.ErrorContains(t, err, "设备不可达")
}

func TestConnectionStateString(t *testing.T) {
	assert.Equal(t, "Initializing", StateInitializing.String())
	assert.Equal(t, "Ready", StateReady.String())
	assert.Equal(t, "Closing", StateClosing.String())
}

func TestGetCommandForVendor(t *testing.T) {
	assert.Equal(t, "display current-configuration", GetCommandForVendor(models.VendorHuawei, "get_config"))
	assert.Equal(t, "show mac-address-table", GetCommandForVendor(models.VendorRuijie, "get_mac"))
	assert.Equal(t, "show running-config", GetCommandForVendor(models.VendorHuawei, "unknown_type"), "未知类型回退")
	assert.Equal(t, "show running-config", GetCommandForVendor("unknown-vendor", "get_config"), "未知厂商回退")
}

func TestGetLLDPCommand(t *testing.T) {
	assert.Equal(t, "display lldp neighbor brief", GetLLDPCommand(models.VendorHuawei))
	assert.Equal(t, "show lldp neighbors", GetLLDPCommand(models.VendorRuijie))
	assert.Equal(t, "show lldp neighbors", GetLLDPCommand("unknown"))
}

func TestContainsErrorHelpers(t *testing.T) {
	assert.True(t, containsEOF("read tcp: connection reset by peer"))
	assert.True(t, containsEOF("unexpected EOF"))
	assert.True(t, containsEOF("broken pipe"))
	assert.False(t, containsEOF("some other error"))

	assert.True(t, containsConnectionError("connection refused"))
	assert.True(t, containsConnectionError("i/o timeout"))
	assert.True(t, containsConnectionError("network unreachable"))
	assert.True(t, containsConnectionError("use of closed"))
	assert.False(t, containsConnectionError("syntax error"))
}

func TestResponseElapsedTime(t *testing.T) {
	// 未完成 → 0
	r := &Response{Started: time.Now()}
	assert.Equal(t, int64(0), r.ElapsedTime())

	r.Finished = r.Started.Add(1500 * time.Millisecond)
	assert.Equal(t, int64(1500), r.ElapsedTime())
}

func TestNewScrapliWrapper_NilDevice(t *testing.T) {
	_, err := NewScrapliWrapper(nil, "u", "p", models.ProtocolTypeSSH)
	assert.ErrorContains(t, err, "设备信息不能为空")

	_, err = NewScrapliWrapperWithPort(nil, "u", "p", 22, models.ProtocolTypeSSH)
	assert.ErrorContains(t, err, "设备信息不能为空")
}

// ---------------- manager.go sqlite 查询路径 ----------------

func newDeviceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "device.db")), &gorm.Config{})
	require.NoError(t, err)
	if sqlDB, err := db.DB(); err == nil {
		t.Cleanup(func() { _ = sqlDB.Close() })
	}
	require.NoError(t, db.Migrator().CreateTable(&models.NetworkDevice{}, &models.AuthCredential{}))
	return db
}

func seedDeviceAndCred(t *testing.T, db *gorm.DB) (string, string) {
	t.Helper()
	credID := "cred-1"
	require.NoError(t, db.Exec(`INSERT INTO sys_network_device (id, device_name, device_type, vendor, ip_address, credential_id, status, created_at, updated_at)
		VALUES ('dev-1', 'sw-core', 'switch', 'huawei', '192.0.2.10', ?, 0, datetime('now'), datetime('now'))`, credID).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_auth_credential (id, credential_name, protocol_type, username, is_default, created_at, updated_at)
		VALUES (?, 'main', 'ssh', 'admin', 1, datetime('now'), datetime('now'))`, credID).Error)
	return "dev-1", credID
}

func TestManager_GetDeviceWithCredentials(t *testing.T) {
	ctx := context.Background()

	// 关联凭证路径
	db := newDeviceTestDB(t)
	devID, credID := seedDeviceAndCred(t, db)
	m := NewManager(db)
	dev, cred, err := m.GetDeviceWithCredentials(ctx, devID)
	require.NoError(t, err)
	assert.Equal(t, "sw-core", dev.DeviceName)
	assert.Equal(t, "main", cred.CredentialName)
	_ = credID

	// 设备不存在
	_, _, err = m.GetDeviceWithCredentials(ctx, "no-such")
	assert.ErrorContains(t, err, "查询设备失败")

	// 无关联凭证 + 有默认凭证
	db2 := newDeviceTestDB(t)
	require.NoError(t, db2.Exec(`INSERT INTO sys_network_device (id, device_name, device_type, vendor, ip_address, status, created_at, updated_at)
		VALUES ('dev-2', 'sw-2', 'switch', 'h3c', '192.0.2.11', 0, datetime('now'), datetime('now'))`).Error)
	require.NoError(t, db2.Exec(`INSERT INTO sys_auth_credential (id, credential_name, protocol_type, username, is_default, created_at, updated_at)
		VALUES ('c9', 'default-cred', 'ssh', 'op', 1, datetime('now'), datetime('now'))`).Error)
	m2 := NewManager(db2)
	dev2, cred2, err := m2.GetDeviceWithCredentials(ctx, "dev-2")
	require.NoError(t, err)
	assert.Equal(t, "sw-2", dev2.DeviceName)
	assert.Equal(t, "default-cred", cred2.CredentialName)

	// 无关联凭证 + 无默认凭证 → 错误
	db3 := newDeviceTestDB(t)
	require.NoError(t, db3.Exec(`INSERT INTO sys_network_device (id, device_name, device_type, vendor, ip_address, status, created_at, updated_at)
		VALUES ('dev-3', 'sw-3', 'switch', 'h3c', '192.0.2.12', 0, datetime('now'), datetime('now'))`).Error)
	m3 := NewManager(db3)
	_, _, err = m3.GetDeviceWithCredentials(ctx, "dev-3")
	assert.ErrorContains(t, err, "未找到默认凭证")

	// 关联凭证但凭证行不存在
	db4 := newDeviceTestDB(t)
	require.NoError(t, db4.Exec(`INSERT INTO sys_network_device (id, device_name, device_type, vendor, ip_address, credential_id, status, created_at, updated_at)
		VALUES ('dev-4', 'sw-4', 'switch', 'h3c', '192.0.2.13', 'ghost', 0, datetime('now'), datetime('now'))`).Error)
	m4 := NewManager(db4)
	_, _, err = m4.GetDeviceWithCredentials(ctx, "dev-4")
	assert.ErrorContains(t, err, "查询凭证失败")
}

func TestManager_CompatMethods(t *testing.T) {
	db := newDeviceTestDB(t)
	m := NewManager(db)

	// GetDevice / GetCredential / GetDefaultCredential
	devID, _ := seedDeviceAndCred(t, db)
	dev, err := m.GetDevice(devID)
	require.NoError(t, err)
	assert.Equal(t, "sw-core", dev.DeviceName)
	_, err = m.GetDevice("ghost")
	assert.Error(t, err)

	cred, err := m.GetCredential("cred-1")
	require.NoError(t, err)
	assert.Equal(t, "main", cred.CredentialName)
	_, err = m.GetCredential("ghost")
	assert.Error(t, err)

	def, err := m.GetDefaultCredential()
	require.NoError(t, err)
	assert.True(t, def.IsDefault)

	// 兼容空操作方法
	_, err = m.ConnectToDevice(context.Background(), dev, cred)
	assert.ErrorContains(t, err, "已废弃")
	assert.NoError(t, m.DisconnectFromDevice(devID))
	assert.NoError(t, m.DisconnectAll())
	assert.NoError(t, m.Close())
	assert.Equal(t, -1, m.GetActiveConnectionCount(), "已废弃返回 -1")

	// executor 未初始化
	_, err = m.ExecuteOnDevice(context.Background(), devID, "show version", false)
	assert.ErrorContains(t, err, "DeviceExecutor 未初始化")
	_, err = m.GetConfigFromDevice(context.Background(), devID)
	assert.ErrorContains(t, err, "DeviceExecutor 未初始化")
	results := m.ExecuteOnDevices(context.Background(), []string{"d1", "d2"}, "show version")
	assert.Len(t, results, 2)
	assert.Contains(t, results["d1"], "ERROR")
}

func TestManager_ExecuteOnDevices_WithDisabledPool(t *testing.T) {
	// 真实 executor + 禁用连接池 → 快速失败路径
	db := newDeviceTestDB(t)
	m := NewManager(db)
	pool := NewDeviceConnectionPool(nil, nil, nil)
	pool.SetEnabled(false)
	sched := NewDeviceTaskScheduler(pool, nil)
	exec := NewDeviceExecutor(sched, nil)
	m.SetExecutor(exec)

	results := m.ExecuteOnDevices(context.Background(), []string{"any"}, "show version")
	assert.Contains(t, results["any"], "ERROR", "禁用池错误进结果 map")
}

// ---------------- connection_pool.go / task_scheduler.go / executor.go ----------------

func TestDeviceConnectionPool_StatsAndLifecycle(t *testing.T) {
	pool := NewDeviceConnectionPool(nil, nil, nil)
	assert.True(t, pool.IsEnabled())

	pool.SetEnabled(false)
	assert.False(t, pool.IsEnabled())

	// GetStats 初始值
	stats := pool.GetStats()
	assert.Equal(t, 0, stats["total_connections"])
	assert.Equal(t, 0, stats["active_connections"])
	assert.Equal(t, 0, stats["idle_connections"])
	assert.Equal(t, DefaultPoolConfig().MaxConnections, stats["max_connections"])
	assert.Equal(t, false, stats["enabled"])

	// 禁用后 GetConnection 快速失败
	_, err := pool.GetConnection(context.Background(), "dev-x")
	assert.ErrorContains(t, err, "连接池未启用")

	// 重新启用后 Close 正常
	pool.SetEnabled(true)
	assert.NoError(t, pool.Close())

	// cleanupIdleConnections 手动触发(空池不 panic)
	pool2 := NewDeviceConnectionPool(nil, nil, nil)
	pool2.cleanupIdleConnections()
	assert.NoError(t, pool2.Close())
}

func TestDeviceConnectionPool_GetDevice_SQLite(t *testing.T) {
	// nil db → nil dereference panic(QUIRK D-12 不修复,生产路径 db 由 core 注入)
	db := newDeviceTestDB(t)
	pool := NewDeviceConnectionPool(db, nil, nil)
	defer pool.Close()

	_, err := pool.GetDevice("no-such")
	assert.ErrorContains(t, err, "查询设备失败")
}

func TestPooledConnection_RefCount(t *testing.T) {
	mu := &sync.Mutex{}
	pc := &PooledConnection{deviceID: "d", mu: mu, refCount: 1}
	assert.False(t, pc.IsIdle())

	pc.ReleaseRef()
	assert.True(t, pc.IsIdle())
	assert.Nil(t, pc.GetWrapper())

	// Execute: wrapper nil → Acquire 失败快速返回
	err := pc.Execute(context.Background(), func(w *ScrapliWrapper) error { return nil })
	assert.ErrorContains(t, err, "连接不可用")

	// Close: wrapper nil → nil
	assert.NoError(t, pc.Close())
}

func TestTaskScheduler_SubmitValidation(t *testing.T) {
	s := NewDeviceTaskScheduler(nil, nil)

	// 未启用
	s.SetEnabled(false)
	err := s.Submit(&DeviceTask{DeviceID: "d", Execute: func(context.Context, *PooledConnection) error { return nil }})
	assert.ErrorContains(t, err, "未启用")

	// 启用后参数校验
	s.SetEnabled(true)
	err = s.Submit(nil)
	assert.ErrorContains(t, err, "任务不能为空")
	err = s.Submit(&DeviceTask{Execute: func(context.Context, *PooledConnection) error { return nil }})
	assert.ErrorContains(t, err, "设备ID不能为空")
	err = s.Submit(&DeviceTask{DeviceID: "d"})
	assert.ErrorContains(t, err, "执行函数不能为空")
}

func TestTaskScheduler_SubmitAndWait(t *testing.T) {
	// 真实 sqlite 池:设备不存在 → createConnection 查询失败 → Callback 快速回调
	// (QUIRK: nil 池会让 worker panic 于 nil deref 且不回调,SubmitAndWait 干等超时)
	db := newDeviceTestDB(t)
	pool := NewDeviceConnectionPool(db, nil, nil)
	defer pool.Close()
	s := NewDeviceTaskScheduler(pool, nil)
	defer s.Stop()

	// 任务在 worker 中执行失败(禁用池 → GetConnection 失败 → task.Callback(err))
	// SubmitAndWait 通过 resultCh 立即拿到失败结果(不等超时)
	err := s.SubmitAndWait(context.Background(), &DeviceTask{
		DeviceID: "dev-ok",
		Timeout:  5 * time.Second,
		Execute: func(ctx context.Context, conn *PooledConnection) error {
			return nil
		},
	})
	assert.Error(t, err)

	// ctx 取消
	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel()
	err = s.SubmitAndWait(cancelCtx, &DeviceTask{
		DeviceID: "dev-cancel", Timeout: time.Second,
		Execute: func(context.Context, *PooledConnection) error { return nil },
	})
	assert.Error(t, err)
}

func TestTaskScheduler_GetStats(t *testing.T) {
	s := NewDeviceTaskScheduler(nil, nil)
	defer s.Stop()

	assert.True(t, s.IsEnabled())
	stats := s.GetStats()
	assert.Equal(t, int64(0), stats["total_submitted"])
	assert.Equal(t, true, stats["enabled"])

	// GetConnectionPool 返回构造传入的池
	pool := NewDeviceConnectionPool(nil, nil, nil)
	defer pool.Close()
	s2 := NewDeviceTaskScheduler(pool, nil)
	defer s2.Stop()
	assert.NotNil(t, s2.GetConnectionPool())
}

func TestDeviceExecutor_Accessors(t *testing.T) {
	sched := NewDeviceTaskScheduler(nil, nil)
	exec := NewDeviceExecutor(sched, nil)
	assert.NotNil(t, exec.GetScheduler())

	// 默认配置构造
	exec2 := NewDeviceExecutor(sched, DefaultExecutionConfig())
	assert.NotNil(t, exec2)
}
