package services

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/stretchr/testify/require"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// loadSampleFixture mirrors component_collector.LoadFixture but is duplicated
// here because LoadFixture lives in the component_collector (internal) test
// helper and is not exported. It walks up from the test cwd to find go.mod,
// then resolves internal/templates/embedded/templates/samples/<name>.
// (Path must match component_collector.fixturesRoot — the samples live in the
// embed tree, NOT a root-level templates/samples directory.)
func loadSampleFixture(t *testing.T, name string) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getcwd: %v", err)
	}
	dir := cwd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			b, err := os.ReadFile(filepath.Join(dir, "internal", "templates", "embedded", "templates", "samples", name))
			if err != nil {
				t.Fatalf("read fixture %s: %v", name, err)
			}
			return string(b)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatalf("go.mod not found from %s", cwd)
	return ""
}

// TestCollectDeviceInfo_RuijieChassisSN asserts that the ruijie `show manuinfo`
// real-machine fixture is parsed by ParseShowManuinfo and the physical chassis
// SN (G1M913U000351) is written into info.SerialNumber. The chassis SN value
// MUST match Device 1 / Location: Chassis in the fixture — NOT the M1 engine
// SN (G1M9140000175) that `show version` System serial number incorrectly
// reports. This is the Phase 49-D-11 fix regression test.
func TestCollectDeviceInfo_RuijieChassisSN(t *testing.T) {
	raw := loadSampleFixture(t, "ruijie_10_62_63_23_show_manuinfo.txt")

	svc := &DeviceInfoCollectionService{}
	info := &DeviceInfo{}
	svc.enrichChassisSerial("show manuinfo", models.VendorRuijie, raw, info)

	if info.SerialNumber != "G1M913U000351" {
		t.Fatalf("ruijie chassis SN: expected G1M913U000351 (physical chassis SN from manuinfo), got %q", info.SerialNumber)
	}
	// D-11 regression guard: chassis SN must NOT equal the M1 engine SN.
	// If this fires, the manuinfo parser is reading the wrong row as chassis.
	if info.SerialNumber == "G1M9140000175" {
		t.Fatalf("D-11 regression: chassis SN == M1 engine SN (G1M9140000175) — show version System serial number leaked into chassis path")
	}
}

// TestCollectDeviceInfo_HuaweiChassisESN asserts that the huawei
// `display device esn` real-machine fixture is parsed by
// ParseDisplayDeviceEsn and the chassis ESN is written into info.SerialNumber.
// The expected value is read directly from the fixture line 1
// ("ESN of chassis 1:102599861597") so the test stays in sync if the
// fixture is regenerated.
func TestCollectDeviceInfo_HuaweiChassisESN(t *testing.T) {
	raw := loadSampleFixture(t, "huawei_10_62_25_253_display_device_esn.txt")
	const wantESN = "102599861597"

	svc := &DeviceInfoCollectionService{}
	info := &DeviceInfo{}
	svc.enrichChassisSerial("display device esn", models.VendorHuawei, raw, info)

	if info.SerialNumber != wantESN {
		t.Fatalf("huawei chassis ESN: expected %q, got %q", wantESN, info.SerialNumber)
	}
}

// TestCollectDeviceInfo_HuaweiEsn_UnrecognizedCommand asserts the
// Pitfall 3 semantic: when Huawei V600R024C00 retired `display device esn`,
// the device replies "Error: Unrecognized command". The collector MUST
// treat this as "no data" (empty SerialNumber, nil error) rather than a
// failure — see ParseDisplayDeviceEsn doc comment.
func TestCollectDeviceInfo_HuaweiEsn_UnrecognizedCommand(t *testing.T) {
	raw := "Error: Unrecognized command found at '^' position."

	svc := &DeviceInfoCollectionService{}
	info := &DeviceInfo{}
	svc.enrichChassisSerial("display device esn", models.VendorHuawei, raw, info)

	if info.SerialNumber != "" {
		t.Fatalf("huawei Unrecognized command: expected empty SN, got %q", info.SerialNumber)
	}
}

// TestCollectDeviceInfo_HuaweiElabelBriefChassisESN 锁死 Phase 49-D-12
// 的 chassis SN 兜底:当 `display device esn` 被 V600R024C00+ 退役时,
// `display device elabel brief` 第一行 "Equipment SN(ESN): <esn>" 仍可
// 拿到 chassis ESN,enrichChassisSerial 必须正确提取。
//
// 真机 S8700 输出(templates/samples/huawei_CX-WH-RUITONG-26F-SWL3-HW-S8700
// _display_device_elabel_brief.txt)首行:Equipment SN(ESN): 102599861598
func TestCollectDeviceInfo_HuaweiElabelBriefChassisESN(t *testing.T) {
	raw := loadSampleFixture(t, "huawei_CX-WH-RUITONG-26F-SWL3-HW-S8700_display_device_elabel_brief.txt")

	svc := &DeviceInfoCollectionService{}
	info := &DeviceInfo{}
	svc.enrichChassisSerial("display device elabel brief", models.VendorHuawei, raw, info)

	if info.SerialNumber != "102599861598" {
		t.Fatalf("huawei elabel brief chassis ESN: expected 102599861598, got %q", info.SerialNumber)
	}
}

// TestHuaweiElabelChassisESN_RegexVariants 直接打 helper regex 边界:
// 真实输出 / 缩进变体 / 多余空行 / 缺 ESN 行 / 命令 echo 行。
func TestHuaweiElabelChassisESN_RegexVariants(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{
			"real S8700 output (first 2 lines)",
			"Equipment SN(ESN): 102599861598\nLicense ESN: 102599861598\n",
			"102599861598",
		},
		{
			"leading hostname prompt line",
			"<CX-WH-RUITONG-26F-SWL3-HW-S8700>dis device elabel brief\nEquipment SN(ESN): 102599861598\n",
			"102599861598",
		},
		{
			"multiple blank lines between",
			"Equipment SN(ESN):    2102311ABCDEF\n\n\nLicense ESN: 2102311ABCDEF\n",
			"2102311ABCDEF",
		},
		{
			"no ESN line at all (old firmware)",
			"Some other text\nLicense ESN: 2102311ABCDEF\n",
			"",
		},
		{
			"empty input",
			"",
			"",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := huaweiElabelChassisESN(c.raw)
			if got != c.want {
				t.Errorf("huaweiElabelChassisESN(...) = %q, want %q", got, c.want)
			}
		})
	}
}

// TestCollectDeviceInfo_ChassisSN_DoesNotOverwriteExisting asserts the
// only-if-empty semantic: when info.SerialNumber is already populated
// (e.g. by an earlier command in the loop), enrichChassisSerial MUST NOT
// overwrite it. This preserves the existing updateDeviceInfo contract.
//
// Phase 49-D-11: switched to the show manuinfo fixture/cmd path since
// `show version` is no longer the chassis-SN source for ruijie.
func TestCollectDeviceInfo_ChassisSN_DoesNotOverwriteExisting(t *testing.T) {
	raw := loadSampleFixture(t, "ruijie_10_62_63_23_show_manuinfo.txt")

	svc := &DeviceInfoCollectionService{}
	info := &DeviceInfo{SerialNumber: "ALREADY-PRESENT"}
	svc.enrichChassisSerial("show manuinfo", models.VendorRuijie, raw, info)

	if info.SerialNumber != "ALREADY-PRESENT" {
		t.Fatalf("enrichChassisSerial overwrote existing SN: got %q", info.SerialNumber)
	}
}

// TestCollectDeviceInfo_LegacyParseStillRuns is a regression guard asserting
// that the legacy parseRuijieDeviceInfo / parseHuaweiDeviceInfo path (which
// still feeds Model / SoftwareVersion / Uptime) keeps working alongside the
// new chassis-SN enrichment. parseDeviceInfo is the legacy entry point and
// must remain untouched (CLAUDE.md Scope Constrainment).
func TestCollectDeviceInfo_LegacyParseStillRuns(t *testing.T) {
	svc := &DeviceInfoCollectionService{}

	// Ruijie legacy parser: "Software Version" keyword must populate
	// info.SoftwareVersion. This is the format parseRuijieDeviceInfo was
	// designed for (idx+7 trim logic).
	legacyRuijie := "Software Version 11.0(4)B8P3"
	info := &DeviceInfo{}
	svc.parseDeviceInfo(legacyRuijie, models.VendorRuijie, info)
	if info.SoftwareVersion == "" {
		t.Errorf("ruijie legacy parse: expected SoftwareVersion populated, got empty")
	}
}

// TestParseRuijieDeviceInfo_DoesNotCaptureShowManuinfoNumberColon 锁死
// Phase 49-D-11 bug: legacy parseRuijieDeviceInfo 之前用
// `strings.Contains(line, "Serial Number")` 太宽,show manuinfo 的
// "Device Serial Number:        G1M913U000351" 行被命中后,strings.Fields
// 切出 ["Device","Serial","Number:","G1M913U000351"],legacy 抓到第一个
// "Serial" 后取 fields[i+1]="Number:" 当 SN(因为 != ":" 检查无法识别
// "Number:" 里嵌的冒号),随后 enrichChassisSerial 的 only-if-empty 守卫
// 阻止修复 → DB 落 "Number:"。
//
// 修复:legacy 只匹配 "system serial" 前缀(show version chassis-level 兜底),
// "Device Serial Number"(show manuinfo)和 "Slot N ... Serial number"
// (show version per-slot)都从 legacy 跳过 — 这两类行的 SN 由 textfsm 解析器
// (enrichChassisSerial → ParseShowManuinfo / ParseShowVersionModules)负责。
func TestParseRuijieDeviceInfo_DoesNotCaptureShowManuinfoNumberColon(t *testing.T) {
	svc := &DeviceInfoCollectionService{}

	// 喂 show manuinfo 的 Device Serial Number 行(真生产 RS8607E-03 Device 1)
	info := &DeviceInfo{}
	svc.parseDeviceInfo("    Device Serial Number:        G1M913U000351", models.VendorRuijie, info)
	if info.SerialNumber != "" {
		t.Fatalf("legacy parser corrupt show manuinfo SN: got %q (must be empty, textfsm path owns it)", info.SerialNumber)
	}
	if info.SerialNumber == "Number:" {
		t.Fatalf("D-11 regression: legacy parser captured literal %q from show manuinfo Device Serial Number row", info.SerialNumber)
	}
}

// TestParseRuijieDeviceInfo_LegacyDoesNotWriteSerialNumber 锁死
// Phase 49-D-11 第二次修复:legacy 完全不再写 info.SerialNumber ——
// 之前第一轮修复让 legacy 用 HasPrefix("system serial") 抓 show version
// 的 "System serial number" 行,看似兜底,实际抓到的是 M1 engine SN
// (show version "System serial number" = 活动 M1 主控板 SN,不是物理
// 机箱 SN),而 chassis SN 走 show manuinfo Device 1 / Location: Chassis,
// 顺序 [show manuinfo, show version] 下 legacy 把 manuinfo 真值覆盖成
// M1 SN,enrichChassisSerial 的 only-if-empty 守卫又无法修复。
//
// 第二次修复:legacy 删掉所有 SN 抓取,info.SerialNumber 100% 由
// enrichChassisSerial 走 textfsm 填充。本测试断言 legacy 在 show
// version 的 chassis / slot SN 行上 MUST NOT 写 info.SerialNumber。
func TestParseRuijieDeviceInfo_LegacyDoesNotWriteSerialNumber(t *testing.T) {
	svc := &DeviceInfoCollectionService{}

	cases := []struct{ name, line string }{
		{"show version System serial number (chassis-level, but actually M1 SN)",
			"System serial number    : G1M9140000175"},
		{"show version System serial number lowercase",
			"system serial number    : G1M9140000175"},
		{"show version Slot Serial number (per-slot SN)",
			"    Serial number       : G1MA11N00053A"},
		{"show manuinfo Device Serial Number",
			"    Device Serial Number:        G1M913U000351"},
	}
	for _, c := range cases {
		info := &DeviceInfo{}
		svc.parseDeviceInfo(c.line, models.VendorRuijie, info)
		if info.SerialNumber != "" {
			t.Errorf("legacy parser MUST NOT write SerialNumber (chassis SN owned by enrichChassisSerial/textfsm); "+
				"%s: line=%q got %q, want empty",
				c.name, c.line, info.SerialNumber)
		}
	}
}

// TestParseRuijieDeviceInfo_SkipsShowVersionSlotSN 锁死 legacy 不再污染
// show version 的 per-slot "Serial number" 行(否则会把 slot SN 错装到
// info.SerialNumber,enrichChassisSerial 又会跳过 textfsm 真值)。
func TestParseRuijieDeviceInfo_SkipsShowVersionSlotSN(t *testing.T) {
	svc := &DeviceInfoCollectionService{}

	// show version Slot M1 的 Serial number 行 — 是板卡 SN,不是 chassis
	info := &DeviceInfo{}
	svc.parseDeviceInfo("    Serial number       : G1MA11N00053A", models.VendorRuijie, info)
	if info.SerialNumber != "" {
		t.Fatalf("legacy parser should skip show version per-slot SN row: got %q (must be empty, ParseShowVersionModules owns it)", info.SerialNumber)
	}
}

// TestStop_TimesOutWhenWorkerHangs 锁死 shutdown-hang-after-port-close 修复。
//
// 模拟 worker 内部死循环(Simulate 持有 wg 不 Done),验证 Stop() 在
// deviceInfoStopTimeout 后强制返回,不让 Core.Close 流程被拖累。
//
// 失败 = 修复被回退,Stop 回到无超时 wg.Wait 死等 → core.Close 永远不退。
func TestStop_TimesOutWhenWorkerHangs(t *testing.T) {
	svc := &DeviceInfoCollectionService{
		stopChan:    make(chan struct{}),
		workerCount: 1,
		isRunning:   true,
	}

	// 注册一个永不退出的 worker(模拟 SSH hang 的真实场景)
	svc.wg.Add(1)
	hangCh := make(chan struct{})
	go func() {
		<-hangCh // 永远阻塞,模拟 SSH 长连接
		// 注意:此处故意不调 wg.Done(),模仿真实 worker hang
	}()
	defer close(hangCh) // 测试结束时释放 goroutine(防止 goroutine 泄漏,不影响测试结果)

	start := time.Now()
	svc.Stop()
	elapsed := time.Since(start)

	// 必须在 deviceInfoStopTimeout 附近返回(允许 1s 调度延迟)
	maxAllowed := deviceInfoStopTimeout + 1*time.Second
	if elapsed > maxAllowed {
		t.Errorf("Stop() blocked too long: %v (expected <= %v)", elapsed, maxAllowed)
	}
	if elapsed < deviceInfoStopTimeout/2 {
		t.Errorf("Stop() returned too fast: %v (expected >= %v, 应等到兜底超时)", elapsed, deviceInfoStopTimeout/2)
	}
	if svc.isRunning {
		t.Errorf("Stop() 完成后 isRunning 应为 false,实际为 true")
	}

	// 重置 wg 以避免影响其他测试(虽然这个 svc 是局部的)
	svc.wg = sync.WaitGroup{}
}

// TestStop_NoopWhenNotRunning 锁死幂等性:重复 Stop() 不应 panic / 不应阻塞。
func TestStop_NoopWhenNotRunning(t *testing.T) {
	svc := &DeviceInfoCollectionService{
		stopChan:  make(chan struct{}),
		isRunning: false,
	}

	start := time.Now()
	svc.Stop()
	elapsed := time.Since(start)

	if elapsed > 100*time.Millisecond {
		t.Errorf("Stop() on not-running service should return immediately, took %v", elapsed)
	}
}

// ============================================================================
// syncOpsAssetChassisSN 测试 — Ruijie Gap 2 自动同步(Phase 49-D-11 修复)
//
// 验证 ops_asset.chassis.devicesn 跟随 sys_network_device.serial_number 同步:
//   1. Ruijie: M1 SN → 真 chassis SN 过渡 → ops_asset.chassis.devicesn 跟着更新
//   2. Ruijie: SN 未变更 → 不动 ops_asset
//   3. 非 Ruijie vendor → 完全跳过(不影响 huawei 等其他厂商)
//   4. Ruijie: 新 SN 为空 → 跳过
//   5. 板卡行(component_type IS NOT NULL)绝不被影响
// ============================================================================

// newSyncTestDB 创建内存 sqlite + GORM, 只含 sync 测试需要的 ops_asset 表。
//
// syncOpsAssetChassisSN 只从内存读 device.Vendor / device.SerialNumber,
// DB 端只查 ops_asset(不需要 sys_network_device 行),所以只需建 ops_asset。
// 手写 DDL: models.Asset 的 ID tag 包含 `default:gen_random_uuid()`,
// GORM AutoMigrate 会把 Postgres 函数写到 CREATE TABLE,而 modernc.org/sqlite
// 不支持(报 `near "(": syntax error`)。TEXT PRIMARY KEY + 测试显式赋 ID 避开。
func newSyncTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE ops_asset (
		id TEXT PRIMARY KEY,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME,
		devicesn TEXT NOT NULL,
		component_type TEXT,
		parent_asset_id TEXT,
		source_device_id TEXT,
		component_slot TEXT
	)`).Error)
	return db
}

// insertChassisAsset 用 raw SQL 插入 chassis 行(component_type = NULL),
// 避开 GORM 默认插入所有 model 字段的行为。
func insertChassisAsset(t *testing.T, db *gorm.DB, id, deviceSN string) {
	t.Helper()
	require.NoError(t, db.Exec(
		`INSERT INTO ops_asset (id, devicesn) VALUES (?, ?)`,
		id, deviceSN,
	).Error)
}

// insertBoardAsset 用 raw SQL 插入板卡行(component_type = 'card')。
func insertBoardAsset(t *testing.T, db *gorm.DB, id, deviceSN string) {
	t.Helper()
	require.NoError(t, db.Exec(
		`INSERT INTO ops_asset (id, devicesn, component_type) VALUES (?, ?, 'card')`,
		id, deviceSN,
	).Error)
}

// TestSyncOpsAssetChassisSN_RuijieTransition 验证 M1 SN → 真 chassis SN 时 ops_asset.chassis.devicesn 被同步。
func TestSyncOpsAssetChassisSN_RuijieTransition(t *testing.T) {
	db := newSyncTestDB(t)
	svc := &DeviceInfoCollectionService{db: db}

	assetID := "22222222-2222-2222-2222-222222222222"
	// 预置: ops_asset chassis 行 DeviceSN = 旧 M1 SN(从 show version 导入导致语义错)
	insertChassisAsset(t, db, assetID, "G1M9140000175")

	// 调用 syncOpsAssetChassisSN: device.SerialNumber 是内存中的旧值(M1 SN),
	// info.SerialNumber 是新采集到的真 chassis SN。
	// 注意: sync 只从内存读 device 字段,不需要 DB 中有 sys_network_device 行。
	device := &models.NetworkDevice{
		BaseModel: models.BaseModel{ID: "11111111-1111-1111-1111-111111111111"},
		DeviceName: "CX-WH-WH-04F-FL-RS8607E-03",
		Vendor: models.VendorRuijie,
		SerialNumber: "G1M9140000175", // 旧值(updateDeviceInfo 之前)
	}
	info := &DeviceInfo{SerialNumber: "G1M913U000351"} // 真 chassis SN
	svc.syncOpsAssetChassisSN(device, info)

	// 验证 ops_asset.chassis.DeviceSN 已同步为真 chassis SN
	var asset models.Asset
	require.NoError(t, db.First(&asset, "id = ?", assetID).Error)
	require.Equal(t, "G1M913U000351", asset.DeviceSN, "chassis.DeviceSN 应该同步为真 chassis SN")
}

// TestSyncOpsAssetChassisSN_RuijieNoChange 验证 SN 未变更时不动 ops_asset(幂等)。
func TestSyncOpsAssetChassisSN_RuijieNoChange(t *testing.T) {
	db := newSyncTestDB(t)
	svc := &DeviceInfoCollectionService{db: db}

	assetID := "44444444-4444-4444-4444-444444444444"
	insertChassisAsset(t, db, assetID, "G1HLC0R000096") // 已经同步好

	device := &models.NetworkDevice{
		BaseModel: models.BaseModel{ID: "33333333-3333-3333-3333-333333333333"},
		Vendor: models.VendorRuijie,
		SerialNumber: "G1HLC0R000096",
	}
	info := &DeviceInfo{SerialNumber: "G1HLC0R000096"} // 与 device 一致
	svc.syncOpsAssetChassisSN(device, info)

	// DeviceSN 应保持原值(未受影响)
	var asset models.Asset
	require.NoError(t, db.First(&asset, "id = ?", assetID).Error)
	require.Equal(t, "G1HLC0R000096", asset.DeviceSN)
}

// TestSyncOpsAssetChassisSN_NonRuijieSkipped 验证非 Ruijie vendor 完全跳过同步。
func TestSyncOpsAssetChassisSN_NonRuijieSkipped(t *testing.T) {
	db := newSyncTestDB(t)
	svc := &DeviceInfoCollectionService{db: db}

	assetID := "66666666-6666-6666-6666-666666666666"
	insertChassisAsset(t, db, assetID, "102599861597")

	// 模拟 huawei 设备: 即使 SN 在新值里变了(实际不会发生,huawei 走 only-if-empty),
	// sync 应跳过非 ruijie vendor
	device := &models.NetworkDevice{
		BaseModel: models.BaseModel{ID: "55555555-5555-5555-5555-555555555555"},
		Vendor: models.VendorHuawei,
		SerialNumber: "102599861597",
	}
	info := &DeviceInfo{SerialNumber: "999999999999"} // 不同,但 vendor 非 ruijie
	svc.syncOpsAssetChassisSN(device, info)

	var asset models.Asset
	require.NoError(t, db.First(&asset, "id = ?", assetID).Error)
	require.Equal(t, "102599861597", asset.DeviceSN, "huawei vendor 应跳过同步,DeviceSN 不变")
}

// TestSyncOpsAssetChassisSN_EmptyNewSNSkipped 验证 info.SerialNumber 为空时跳过。
func TestSyncOpsAssetChassisSN_EmptyNewSNSkipped(t *testing.T) {
	db := newSyncTestDB(t)
	svc := &DeviceInfoCollectionService{db: db}

	assetID := "88888888-8888-8888-8888-888888888888"
	insertChassisAsset(t, db, assetID, "G1M9140000175")

	device := &models.NetworkDevice{
		BaseModel: models.BaseModel{ID: "77777777-7777-7777-7777-777777777777"},
		Vendor: models.VendorRuijie,
		SerialNumber: "G1M9140000175",
	}
	info := &DeviceInfo{SerialNumber: ""} // parse 失败 / 未填
	svc.syncOpsAssetChassisSN(device, info)

	var asset models.Asset
	require.NoError(t, db.First(&asset, "id = ?", assetID).Error)
	require.Equal(t, "G1M9140000175", asset.DeviceSN, "新 SN 为空应跳过,DeviceSN 不变")
}

// TestSyncOpsAssetChassisSN_BoardRowsUntouched 验证板卡行(component_type 非空)绝不被影响。
func TestSyncOpsAssetChassisSN_BoardRowsUntouched(t *testing.T) {
	db := newSyncTestDB(t)
	svc := &DeviceInfoCollectionService{db: db}

	// chassis 行(DeviceSN = M1 SN)
	insertChassisAsset(t, db, "a-chassis", "G1M9140000175")
	// 板卡行(DeviceSN = 板卡自己的 SN,不是 M1 SN → 不会被 SELECT WHERE 命中)
	insertBoardAsset(t, db, "a-board", "G1MA11N00053A")

	device := &models.NetworkDevice{
		BaseModel: models.BaseModel{ID: "99999999-9999-9999-9999-999999999999"},
		Vendor: models.VendorRuijie,
		SerialNumber: "G1M9140000175",
	}
	info := &DeviceInfo{SerialNumber: "G1M913U000351"}
	svc.syncOpsAssetChassisSN(device, info)

	// chassis 行更新了
	var chassis models.Asset
	require.NoError(t, db.First(&chassis, "id = ?", "a-chassis").Error)
	require.Equal(t, "G1M913U000351", chassis.DeviceSN, "chassis 行应被同步")
	// 板卡行不动(它的 DeviceSN = 板卡自己的 SN,不是 M1 SN)
	var board models.Asset
	require.NoError(t, db.First(&board, "id = ?", "a-board").Error)
	require.Equal(t, "G1MA11N00053A", board.DeviceSN, "板卡行的 DeviceSN 不应被改(它是板卡自己的 SN)")
}
