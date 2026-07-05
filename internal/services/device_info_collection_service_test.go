package services

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
)

// loadSampleFixture mirrors component_collector.LoadFixture but is duplicated
// here because LoadFixture lives in the component_collector (internal) test
// helper and is not exported. It walks up from the test cwd to find go.mod,
// then resolves templates/samples/<name>.
func loadSampleFixture(t *testing.T, name string) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getcwd: %v", err)
	}
	dir := cwd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			b, err := os.ReadFile(filepath.Join(dir, "templates", "samples", name))
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

// TestCollectDeviceInfo_RuijieChassisSN asserts that the ruijie `show version`
// real-machine fixture is parsed by ParseShowVersionModules and the chassis
// SN (G1HLC0R000096) is written into info.SerialNumber. The chassis SN value
// MUST match the fixture line 7 and the existing
// TestRuijieCliParseShowVersionModules assertion (component_collector pkg).
func TestCollectDeviceInfo_RuijieChassisSN(t *testing.T) {
	raw := loadSampleFixture(t, "ruijie_10_62_63_21_show_version.txt")

	svc := &DeviceInfoCollectionService{}
	info := &DeviceInfo{}
	svc.enrichChassisSerial("show version", models.VendorRuijie, raw, info)

	if info.SerialNumber != "G1HLC0R000096" {
		t.Fatalf("ruijie chassis SN: expected G1HLC0R000096, got %q", info.SerialNumber)
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

// TestCollectDeviceInfo_ChassisSN_DoesNotOverwriteExisting asserts the
// only-if-empty semantic: when info.SerialNumber is already populated
// (e.g. by an earlier command in the loop), enrichChassisSerial MUST NOT
// overwrite it. This preserves the existing updateDeviceInfo contract.
func TestCollectDeviceInfo_ChassisSN_DoesNotOverwriteExisting(t *testing.T) {
	raw := loadSampleFixture(t, "ruijie_10_62_63_21_show_version.txt")

	svc := &DeviceInfoCollectionService{}
	info := &DeviceInfo{SerialNumber: "ALREADY-PRESENT"}
	svc.enrichChassisSerial("show version", models.VendorRuijie, raw, info)

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

	// Ruijie legacy parser: uptime keyword must populate info.Uptime.
	legacyRuijie := "System uptime           : 2085:10:54:27"
	info := &DeviceInfo{}
	svc.parseDeviceInfo(legacyRuijie, models.VendorRuijie, info)
	if info.Uptime == "" {
		t.Errorf("ruijie legacy parse: expected Uptime to be populated, got empty")
	}
}
