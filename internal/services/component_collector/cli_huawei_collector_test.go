package component_collector

import (
	"strings"
	"testing"
)

// TestHuaweiCliParseDisplayDeviceEsn loads the real Huawei S8700
// display_device_esn fixture. The current fixture is the V600R024C00
// "Error: Unrecognized command" path (Huawei retired display device esn
// on some chassis — RESEARCH Pitfall 3). The collector must return an
// empty slice without panicking. We additionally feed a synthetic
// success-path fixture to assert ESN extraction works when the command
// is supported.
func TestHuaweiCliParseDisplayDeviceEsn(t *testing.T) {
	c := NewHuaweiCliCollector()

	// Path A: real sample contains "Error: Unrecognized command".
	raw, err := LoadFixture("huawei_10_62_25_253_display_esn.txt")
	if err != nil {
		t.Fatalf("LoadFixture error: %v", err)
	}
	comps, err := c.ParseDisplayDeviceEsn(raw)
	if err != nil {
		t.Fatalf("ParseDisplayDeviceEsn error on error-path sample: %v", err)
	}
	if len(comps) != 0 {
		t.Errorf("error-path sample: expected 0 components, got %d", len(comps))
	}

	// Path B: synthetic success fixture.
	success := "ESN of chassis 1: 102599861597\n"
	comps, err = c.ParseDisplayDeviceEsn(success)
	if err != nil {
		t.Fatalf("ParseDisplayDeviceEsn error on success-path: %v", err)
	}
	if len(comps) != 1 {
		t.Fatalf("success-path: expected 1 component, got %d", len(comps))
	}
	if comps[0].SerialNumber != "102599861597" {
		t.Errorf("chassis SN: expected 102599861597, got %s", comps[0].SerialNumber)
	}
	if comps[0].ComponentType != ComponentTypeChassis {
		t.Errorf("expected ComponentType=%s, got %s", ComponentTypeChassis, comps[0].ComponentType)
	}
}

// TestHuaweiCliParseInterfaceStatus parses a synthetic display interface
// status fixture (the on-site sample for this command was not captured —
// RESEARCH §Wave 0 / 48-02 T2b). The fixture contains 1 up + 1 down port.
func TestHuaweiCliParseInterfaceStatus(t *testing.T) {
	c := NewHuaweiCliCollector()
	fixture := "Interface                     PHY     Protocol Description\n" +
		"10GE5/0/4                     up      up       fiber\n" +
		"10GE5/0/6                     down    down     fiber\n"
	up, err := c.ParseInterfaceStatus(fixture)
	if err != nil {
		t.Fatalf("ParseInterfaceStatus error: %v", err)
	}
	if len(up) != 1 {
		t.Fatalf("expected 1 up interface, got %d: %v", len(up), up)
	}
	if up[0] != "10GE5/0/4" {
		t.Errorf("expected up interface 10GE5/0/4, got %s", up[0])
	}
}

// TestHuaweiCliParseInterfaceTransceiver loads the real Huawei S8700
// display_interface_transceiver fixture and asserts the two populated
// transceiver blocks (10GE5/0/4 + 10GE5/0/5) are extracted with their
// vendor SNs.
func TestHuaweiCliParseInterfaceTransceiver(t *testing.T) {
	c := NewHuaweiCliCollector()
	raw, err := LoadFixture("huawei_10_62_25_253_display_interface_transceiver.txt")
	if err != nil {
		t.Fatalf("LoadFixture error: %v", err)
	}
	// Both interfaces must be in the up-list for the D-10 filter to
	// admit them (per the test name and Pitfall 5 contract).
	comps, err := c.ParseInterfaceTransceiver(raw, []string{"10GE5/0/4", "10GE5/0/5"})
	if err != nil {
		t.Fatalf("ParseInterfaceTransceiver error: %v", err)
	}
	if len(comps) != 2 {
		t.Fatalf("expected 2 transceiver components, got %d", len(comps))
	}
	want := map[string]string{
		"10GE5/0/4": "8000012000082",
		"10GE5/0/5": "1000159211688",
	}
	for _, comp := range comps {
		expectedSN, ok := want[comp.Slot]
		if !ok {
			t.Errorf("unexpected interface %s in transceiver output", comp.Slot)
			continue
		}
		if comp.SerialNumber != expectedSN {
			t.Errorf("interface %s SN: expected %s, got %s", comp.Slot, expectedSN, comp.SerialNumber)
		}
		if comp.ComponentType != ComponentTypeTransceiver {
			t.Errorf("interface %s: expected ComponentType=%s, got %s",
				comp.Slot, ComponentTypeTransceiver, comp.ComponentType)
		}
		if comp.Source != SourceHuaweiCLI {
			t.Errorf("interface %s: expected Source=%s, got %s",
				comp.Slot, SourceHuaweiCLI, comp.Source)
		}
	}
}

// TestHuaweiTransceiverFiltersDownPorts is the D-10 hard test for the
// Huawei side. It constructs a fixture with one up port (10GE5/0/4 SN=A)
// and one down port (10GE5/0/6 SN=B), then asserts the down port is
// filtered out when upInterfaces=["10GE5/0/4"].
func TestHuaweiTransceiverFiltersDownPorts(t *testing.T) {
	c := NewHuaweiCliCollector()
	// Minimal transceiver fixture mirroring the real format.
	fixture := strings.Join([]string{
		" 10GE5/0/4 transceiver information:",
		" Manufacture information:",
		"   Manu. Serial Number                   :SN-A",
		"",
		" 10GE5/0/6 transceiver information:",
		" Manufacture information:",
		"   Manu. Serial Number                   :SN-B",
		"",
	}, "\n")
	comps, err := c.ParseInterfaceTransceiver(fixture, []string{"10GE5/0/4"})
	if err != nil {
		t.Fatalf("ParseInterfaceTransceiver error: %v", err)
	}
	if len(comps) != 1 {
		t.Fatalf("D-10 expected 1 transceiver (down port filtered), got %d: %+v", len(comps), comps)
	}
	if comps[0].Slot != "10GE5/0/4" {
		t.Errorf("expected 10GE5/0/4 to survive, got %s", comps[0].Slot)
	}
	if comps[0].SerialNumber != "SN-A" {
		t.Errorf("expected SN-A, got %s", comps[0].SerialNumber)
	}
	// Defensive: explicitly assert the down-port SN never appears.
	for _, comp := range comps {
		if comp.SerialNumber == "SN-B" {
			t.Errorf("D-10 violation: down-port 10GE5/0/6 transceiver leaked (SN=%s)", comp.SerialNumber)
		}
	}
}

// TestHuaweiCliParseDisplayDeviceElabelBrief loads the real
// CX-WH-RUITONG-26F-SWL3-HW-S8700 `display device elabel brief`
// fixture (Phase 49-D-12) and asserts:
//   - 12 component rows are parsed (1 BackPlane dropped, 4 cards, 2 engines,
//     2 fans, 4 power supplies) — chassis row is intentionally NOT returned
//     (ParseDisplayDeviceEsn already covers it).
//   - Each row's ComponentType is mapped correctly per (SLOTID, TYPE):
//     FANn→fan, PWRn→power, numeric+SRU/MPU/SFU→engine, numeric+other→card.
//   - Model field is populated from the TYPE column (so the frontend
//     "型号" column can show e.g. "锐捷 M8600E-48GT-ED" — Phase 49 D-07).
//   - The textual table header row ("SlotID ... Sub ... Type ... SN ... P/N")
//     is silently skipped (it would otherwise leak as a fake record with
//     SlotID="SlotID" TYPE="Type" SN="SN" PN="P/N").
func TestHuaweiCliParseDisplayDeviceElabelBrief(t *testing.T) {
	c := NewHuaweiCliCollector()

	raw, err := LoadFixture("huawei_CX-WH-RUITONG-26F-SWL3-HW-S8700_display_device_elabel_brief.txt")
	if err != nil {
		t.Fatalf("LoadFixture error: %v", err)
	}
	comps, err := c.ParseDisplayDeviceElabelBrief(raw)
	if err != nil {
		t.Fatalf("ParseDisplayDeviceElabelBrief error: %v", err)
	}

	// Expected: 4 cards (slots 1-4) + 2 engines (slots 5-6 LSG7SRUEX1C0)
	//           + 2 fans (FAN1/2) + 4 power (PWR1-4) = 12 rows.
	// BackPlane row (SLOTID="--") and the table header are dropped.
	if len(comps) != 12 {
		names := make([]string, 0, len(comps))
		for _, c := range comps {
			names = append(names, c.Slot)
		}
		t.Fatalf("expected 12 components (4 cards + 2 engines + 2 fans + 4 power), got %d: %v", len(comps), names)
	}

	// Index by slot for per-slot assertions.
	bySlot := make(map[string]Component, len(comps))
	for _, comp := range comps {
		bySlot[comp.Slot] = comp
	}

	// 4 cards (slots 1-4, LSG7G48VX1E0)
	for _, slot := range []string{"Slot 1", "Slot 2", "Slot 3", "Slot 4"} {
		c, ok := bySlot[slot]
		if !ok {
			t.Errorf("missing card %s", slot)
			continue
		}
		if c.ComponentType != ComponentTypeCard {
			t.Errorf("%s: expected ComponentType=%s, got %s", slot, ComponentTypeCard, c.ComponentType)
		}
		if c.Model != "LSG7G48VX1E0" {
			t.Errorf("%s: expected Model=LSG7G48VX1E0, got %s", slot, c.Model)
		}
	}

	// 2 engines (slots 5-6, LSG7SRUEX1C0 — TYPE contains "SRU" → engine)
	for _, slot := range []string{"Slot 5", "Slot 6"} {
		c, ok := bySlot[slot]
		if !ok {
			t.Errorf("missing engine %s", slot)
			continue
		}
		if c.ComponentType != ComponentTypeEngine {
			t.Errorf("%s: expected ComponentType=%s (TYPE contains SRU), got %s", slot, ComponentTypeEngine, c.ComponentType)
		}
		if c.Model != "LSG7SRUEX1C0" {
			t.Errorf("%s: expected Model=LSG7SRUEX1C0, got %s", slot, c.Model)
		}
		if c.SerialNumber != "102599806032" && c.SerialNumber != "102599806033" {
			t.Errorf("%s: unexpected SN %s", slot, c.SerialNumber)
		}
	}

	// 2 fans (FAN1-2, FAN-240SM-B)
	for _, slot := range []string{"Slot FAN1", "Slot FAN2"} {
		c, ok := bySlot[slot]
		if !ok {
			t.Errorf("missing fan %s", slot)
			continue
		}
		if c.ComponentType != ComponentTypeFan {
			t.Errorf("%s: expected ComponentType=%s, got %s", slot, ComponentTypeFan, c.ComponentType)
		}
		if c.Model != "FAN-240SM-B" {
			t.Errorf("%s: expected Model=FAN-240SM-B, got %s", slot, c.Model)
		}
	}

	// 4 power supplies (PWR1-4, PAC3KS54-DF)
	for _, slot := range []string{"Slot PWR1", "Slot PWR2", "Slot PWR3", "Slot PWR4"} {
		c, ok := bySlot[slot]
		if !ok {
			t.Errorf("missing power %s", slot)
			continue
		}
		if c.ComponentType != ComponentTypePower {
			t.Errorf("%s: expected ComponentType=%s, got %s", slot, ComponentTypePower, c.ComponentType)
		}
		if c.Model != "PAC3KS54-DF" {
			t.Errorf("%s: expected Model=PAC3KS54-DF, got %s", slot, c.Model)
		}
	}

	// Defensive: no BackPlane (chassis) row leaked, no header row leaked.
	for _, c := range comps {
		if c.ComponentType == ComponentTypeChassis {
			t.Errorf("BackPlane/chassis row leaked: slot=%s sn=%s", c.Slot, c.SerialNumber)
		}
		if c.Slot == "Slot SlotID" || c.Slot == "Slot --" {
			t.Errorf("non-data row leaked: slot=%s sn=%s", c.Slot, c.SerialNumber)
		}
	}

	// Defensive: Source must be the huawei CLI marker (D-09 invariant).
	for _, c := range comps {
		if c.Source != SourceHuaweiCLI {
			t.Errorf("%s: expected Source=%s, got %s", c.Slot, SourceHuaweiCLI, c.Source)
		}
	}
}

// TestHuaweiElabelComponentType_TypeVariants 锁死 (SLOTID, TYPE) →
// ComponentType 映射的边界场景(不依赖 fixture,直接打单元表)。
func TestHuaweiElabelComponentType_TypeVariants(t *testing.T) {
	cases := []struct {
		name   string
		slotID string
		typ    string
		want   string
	}{
		{"fan FAN1", "FAN1", "FAN-240SM-B", ComponentTypeFan},
		{"fan FAN2 (case-insensitive prefix)", "FAN12", "FAN-X", ComponentTypeFan},
		{"power PWR1", "PWR1", "PAC3KS54-DF", ComponentTypePower},
		{"power PWR4", "PWR4", "PAC3KS54-DF", ComponentTypePower},
		{"engine SRU (S8700)", "5", "LSG7SRUEX1C0", ComponentTypeEngine},
		{"engine MPU (S9300/S9306)", "1", "MPU-100", ComponentTypeEngine},
		{"engine SFU (fabric module)", "3", "SFU-200", ComponentTypeEngine},
		{"card default numeric", "1", "LSG7G48VX1E0", ComponentTypeCard},
		{"card default numeric (S5700)", "2", "ES5D21X04S00", ComponentTypeCard},
		// Pathological: TYPE contains "SRU" but it's not actually a route unit
		// (model with "SRU" substring). Spec keeps the substring heuristic —
		// false-positive is preferable to over-fitting to specific model IDs.
		{"engine via SRU substring (heuristic)", "1", "LSG7SRUECARD", ComponentTypeEngine},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := huaweiElabelComponentType(c.slotID, c.typ)
			if got != c.want {
				t.Errorf("huaweiElabelComponentType(%q, %q) = %s, want %s", c.slotID, c.typ, got, c.want)
			}
		})
	}
}
