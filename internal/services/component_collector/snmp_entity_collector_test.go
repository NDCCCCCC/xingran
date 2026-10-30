package component_collector

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/xingran-next/xingran-go-backend/internal/device"
)

// stubSNMPClient implements device.SNMPGetter for unit tests. Each Walk /
// Get call hits an in-memory table so tests can construct synthetic
// ENTITY-MIB shapes (chassis + fan + dual-class module/fan + temprature*
// noise) without a live device.
type stubSNMPClient struct {
	// classes maps entity index → entPhysicalClass integer.
	classes map[int]int
	// attrs maps "<oidBase>.<idx>" → pre-decoded value (string or int).
	attrs map[string]interface{}
	// walkErr forces Walk to fail when non-nil.
	walkErr error
}

func (s *stubSNMPClient) Walk(oid string, cb func(oid string, value interface{}) bool) error {
	if s.walkErr != nil {
		return s.walkErr
	}
	for idx, class := range s.classes {
		// gosnmp Walk delivers OID prefixed with a leading dot.
		full := "." + oid + "." + itoa(idx)
		if !cb(full, int64(class)) {
			return nil
		}
	}
	return nil
}

func (s *stubSNMPClient) Get(oid string) (interface{}, error) {
	// Strip the leading dot if a caller mistakenly includes it.
	oid = strings.TrimPrefix(oid, ".")
	if v, ok := s.attrs[oid]; ok {
		return v, nil
	}
	return nil, errors.New("noSuchObject")
}

func itoa(n int) string {
	// tiny helper to avoid pulling in strconv for a tight test stub.
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// buildAttrKey produces "<oidBase>.<idx>".
func buildAttrKey(base string, idx int) string { return base + "." + itoa(idx) }

// newRuijieNoiseFixture builds a synthetic Ruijie ENTITY-MIB fixture:
//   - 1 chassis (Class=3, ContainedIn=0)
//   - 2 real PSUs (Class=8, Name="power0"/"power1" — D-11 retention case)
//   - 1 fan (Class=9)
//   - 1 module (Class=5, board)
//   - 3 Ruijie typo-noise "temprature*" entries (Class=8 — D-11 filter case)
func newRuijieNoiseFixture() *stubSNMPClient {
	const idxChassis, idxPSU0, idxPSU1, idxFan, idxModule = 1, 4, 5, 6, 7
	const idxTemp1, idxTemp2, idxTemp3 = 100, 101, 102
	classes := map[int]int{
		idxChassis: 3, idxPSU0: 8, idxPSU1: 8, idxFan: 9, idxModule: 5,
		idxTemp1: 8, idxTemp2: 8, idxTemp3: 8,
	}
	attrs := map[string]interface{}{
		buildAttrKey(device.OidEntPhysicalClass, idxChassis):       int64(3),
		buildAttrKey(device.OidEntPhysicalClass, idxPSU0):          int64(8),
		buildAttrKey(device.OidEntPhysicalClass, idxPSU1):          int64(8),
		buildAttrKey(device.OidEntPhysicalClass, idxFan):           int64(9),
		buildAttrKey(device.OidEntPhysicalClass, idxModule):        int64(5),
		buildAttrKey(device.OidEntPhysicalClass, idxTemp1):         int64(8),
		buildAttrKey(device.OidEntPhysicalClass, idxTemp2):         int64(8),
		buildAttrKey(device.OidEntPhysicalClass, idxTemp3):         int64(8),
		buildAttrKey(device.OidEntPhysicalName, idxChassis):        "chassis",
		buildAttrKey(device.OidEntPhysicalName, idxPSU0):           "power0",
		buildAttrKey(device.OidEntPhysicalName, idxPSU1):           "power1",
		buildAttrKey(device.OidEntPhysicalName, idxFan):            "Fan 1",
		buildAttrKey(device.OidEntPhysicalName, idxModule):         "Slot 1",
		buildAttrKey(device.OidEntPhysicalName, idxTemp1):          "temprature1",
		buildAttrKey(device.OidEntPhysicalName, idxTemp2):          "temprature2",
		buildAttrKey(device.OidEntPhysicalName, idxTemp3):          "tempratureSensor3",
		buildAttrKey(device.OidEntPhysicalSerialNum, idxChassis):   "G1J40D100022A",
		buildAttrKey(device.OidEntPhysicalSerialNum, idxPSU0):      "A82603150300065",
		buildAttrKey(device.OidEntPhysicalSerialNum, idxPSU1):      "A82603150300066",
		buildAttrKey(device.OidEntPhysicalSerialNum, idxFan):       "FAN-SN-001",
		buildAttrKey(device.OidEntPhysicalSerialNum, idxModule):    "BOARD-SN-001",
		buildAttrKey(device.OidEntPhysicalSerialNum, idxTemp1):     "NOISE-SN-001",
		buildAttrKey(device.OidEntPhysicalSerialNum, idxTemp2):     "NOISE-SN-002",
		buildAttrKey(device.OidEntPhysicalSerialNum, idxTemp3):     "NOISE-SN-003",
		buildAttrKey(device.OidEntPhysicalModelName, idxChassis):   "RG-S8607E",
		buildAttrKey(device.OidEntPhysicalModelName, idxPSU0):      "RG-PA600I",
		buildAttrKey(device.OidEntPhysicalModelName, idxPSU1):      "RG-PA600I",
		buildAttrKey(device.OidEntPhysicalModelName, idxModule):    "M8600E-24GT20SFP4XS-ED",
		buildAttrKey(device.OidEntPhysicalContainedIn, idxChassis): int64(0),
		buildAttrKey(device.OidEntPhysicalContainedIn, idxPSU0):    int64(1),
		buildAttrKey(device.OidEntPhysicalContainedIn, idxPSU1):    int64(1),
		buildAttrKey(device.OidEntPhysicalContainedIn, idxFan):     int64(1),
		buildAttrKey(device.OidEntPhysicalContainedIn, idxModule):  int64(1),
		buildAttrKey(device.OidEntPhysicalContainedIn, idxTemp1):   int64(1),
		buildAttrKey(device.OidEntPhysicalContainedIn, idxTemp2):   int64(1),
		buildAttrKey(device.OidEntPhysicalContainedIn, idxTemp3):   int64(1),
	}
	return &stubSNMPClient{classes: classes, attrs: attrs}
}

// TestEntityCollectorRuijieFiltersTemprature — D-11 filter rule: drop
// entities where Class==8 AND Name starts with "temprature". Real
// power-supply rows (Class=8, Name="power0"/"power1") must be retained.
func TestEntityCollectorRuijieFiltersTemprature(t *testing.T) {
	client := newRuijieNoiseFixture()
	coll := NewEntityCollector(client, "")
	set, err := coll.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	if set.Chassis == nil {
		t.Fatalf("Chassis is nil")
	}
	if set.Chassis.SerialNumber != "G1J40D100022A" {
		t.Errorf("chassis SN: expected G1J40D100022A, got %s", set.Chassis.SerialNumber)
	}
	// Inspect every emitted Component for temprature* leaks.
	for _, c := range set.Components {
		if c.ComponentType == ComponentTypePower && strings.HasPrefix(c.Model, "RG-PA") {
			// Real PSU rows are expected.
			continue
		}
		// The Name field is captured in Raw — but the easiest assertion
		// is: no SerialNumber from the synthetic noise set should appear.
		if c.SerialNumber == "NOISE-SN-001" || c.SerialNumber == "NOISE-SN-002" || c.SerialNumber == "NOISE-SN-003" {
			t.Errorf("D-11 violation: temprature* noise row leaked (SN=%s, Slot=%s)",
				c.SerialNumber, c.Slot)
		}
	}
	// 2 PSU + 1 fan + 1 module = 4 non-chassis components expected
	// (the chassis appears at top-level as set.Chassis, not in Components).
	if len(set.Components) != 4 {
		t.Errorf("expected 4 components (2 PSU + 1 fan + 1 module); got %d", len(set.Components))
		for i, c := range set.Components {
			t.Logf("  comp[%d] = type=%s slot=%s sn=%s", i, c.ComponentType, c.Slot, c.SerialNumber)
		}
	}
}

// TestEntityCollectorPreservesRealPowerSupply — D-11 positive case: a
// Class=8 / Name="power0" row (RG-PA600I PSU) must survive the noise filter.
func TestEntityCollectorPreservesRealPowerSupply(t *testing.T) {
	client := newRuijieNoiseFixture()
	coll := NewEntityCollector(client, "")
	set, err := coll.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	var psuSeen []string
	for _, c := range set.Components {
		if c.ComponentType == ComponentTypePower {
			psuSeen = append(psuSeen, c.SerialNumber)
		}
	}
	if len(psuSeen) != 2 {
		t.Fatalf("expected 2 power components (power0/power1 RG-PA600I), got %d: %v",
			len(psuSeen), psuSeen)
	}
	// Specifically assert the PSU SNs that RQ-001 captured.
	for _, sn := range psuSeen {
		if !strings.HasPrefix(sn, "A826") {
			t.Errorf("PSU SN prefix mismatch: expected A826... RG-PA600I SN, got %s", sn)
		}
	}
}

// newHuaweiDualClassFixture builds a Huawei S8700 ENTITY-MIB fixture where
// the SAME entity appears twice: once as Class=5/module (SN empty) and
// once as Class=9/fan (SN populated). The dual-class dedup rule (Pitfall 2)
// must keep the populated row.
func newHuaweiDualClassFixture() *stubSNMPClient {
	const chassisIdx, modIdx, fanIdx = 1, 10, 11
	classes := map[int]int{chassisIdx: 3, modIdx: 5, fanIdx: 9}
	attrs := map[string]interface{}{
		buildAttrKey(device.OidEntPhysicalClass, chassisIdx):            int64(3),
		buildAttrKey(device.OidEntPhysicalClass, modIdx):                int64(5),
		buildAttrKey(device.OidEntPhysicalClass, fanIdx):                int64(9),
		buildAttrKey(device.OidEntPhysicalName, chassisIdx):             "chassis",
		buildAttrKey(device.OidEntPhysicalName, modIdx):                 "LSG7SRUEX1C0 5",
		buildAttrKey(device.OidEntPhysicalName, fanIdx):                 "LSG7SRUEX1C0 5 (Master)",
		buildAttrKey(device.OidEntPhysicalSerialNum, chassisIdx):        "210235936910HC000001",
		buildAttrKey(device.OidEntPhysicalSerialNum, modIdx):            "", // empty placeholder
		buildAttrKey(device.OidEntPhysicalSerialNum, fanIdx):            "102599806030",
		buildAttrKey(device.OidEntPhysicalModelName, chassisIdx):        "S8700-6",
		buildAttrKey(device.OidEntPhysicalModelName, modIdx):            "LSG7SRUEX1C0",
		buildAttrKey(device.OidEntPhysicalModelName, fanIdx):            "LSG7SRUEX1C0",
		buildAttrKey(device.OidEntPhysicalContainedIn, chassisIdx):      int64(0),
		buildAttrKey(device.OidEntPhysicalContainedIn, modIdx):          int64(1),
		buildAttrKey(device.OidEntPhysicalContainedIn, fanIdx):          int64(1),
	}
	return &stubSNMPClient{classes: classes, attrs: attrs}
}

// TestEntityCollectorHuaweiDualClassDedup — Pitfall 2: Huawei S8700
// reports the same engine as module(5, SN empty) + fan(9, SN populated).
// Collector must keep the populated row.
func TestEntityCollectorHuaweiDualClassDedup(t *testing.T) {
	client := newHuaweiDualClassFixture()
	coll := NewEntityCollector(client, "")
	set, err := coll.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	// Per RESEARCH Pitfall 2: Huawei reports the SAME engine twice — once
	// as Class=5/module (SN empty placeholder) and once as Class=9 (SN
	// populated). The collector must keep the populated Class=9 row. The
	// resulting component is the engine (Class=9 with non-"fan" Name →
	// mapped to ComponentTypeEngine by classToComponentType), NOT a fan.
	var engineSeen bool
	var engineSN string
	for _, c := range set.Components {
		if c.SerialNumber == "102599806030" {
			engineSeen = true
			engineSN = c.SerialNumber
			if c.ComponentType != ComponentTypeEngine {
				t.Errorf("Pitfall 2 row: expected ComponentType=%s (engine), got %s",
					ComponentTypeEngine, c.ComponentType)
			}
		}
	}
	if !engineSeen {
		t.Errorf("expected engine component with SN=102599806030 (Pitfall 2 dual-class dedup); not found")
	} else {
		t.Logf("dual-class dedup OK: retained engine SN=%s", engineSN)
	}
	// Verify NO component with empty SN is emitted (other than the chassis).
	for _, c := range set.Components {
		if c.SerialNumber == "" {
			t.Errorf("component with empty SN leaked: type=%s slot=%s", c.ComponentType, c.Slot)
		}
	}
}

// TestFixturesLoaderGlobCount — CountFixtures must return >0 and
// LoadFixture("ruijie_10_62_63_21_show_version.txt") must return the
// real-machine sample (non-empty string).
func TestFixturesLoaderGlobCount(t *testing.T) {
	n, err := CountFixtures()
	if err != nil {
		t.Fatalf("CountFixtures error: %v", err)
	}
	if n <= 0 {
		t.Fatalf("CountFixtures returned %d; expected >0", n)
	}
	t.Logf("CountFixtures=%d (do not hardcode in production code)", n)

	content, err := LoadFixture("ruijie_10_62_63_21_show_version.txt")
	if err != nil {
		t.Fatalf("LoadFixture error: %v", err)
	}
	if !strings.Contains(content, "System serial number") {
		t.Errorf("loaded ruijie show version does not contain expected header")
	}
}
