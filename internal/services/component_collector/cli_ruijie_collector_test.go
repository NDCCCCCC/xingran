package component_collector

import (
	"strings"
	"testing"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
)

// TestRuijieCliParseShowVersionModules loads the real Ruijie RS8607E
// show_version fixture and asserts the chassis + 7 module slots are
// extracted with correct SNs. The fixture has Slot M1 (engine), Slot 1
// through Slot 5 (cards), and Slot M2 (engine).
func TestRuijieCliParseShowVersionModules(t *testing.T) {
	c := NewRuijieCliCollector()
	raw, err := LoadFixture("ruijie_10_62_63_21_show_version.txt")
	if err != nil {
		t.Fatalf("LoadFixture error: %v", err)
	}
	comps, err := c.ParseShowVersionModules(raw)
	if err != nil {
		t.Fatalf("ParseShowVersionModules error: %v", err)
	}
	if len(comps) < 7 {
		t.Fatalf("expected at least 7 components (1 chassis + M1/M2/1-5 modules), got %d", len(comps))
	}

	// Build a slot→SN lookup for assertion.
	bySlot := map[string]Component{}
	chassisSN := ""
	for _, comp := range comps {
		if comp.ComponentType == ComponentTypeChassis {
			chassisSN = comp.SerialNumber
			continue
		}
		bySlot[comp.Slot] = comp
	}
	if chassisSN != "G1HLC0R000096" {
		t.Errorf("chassis SN: expected G1HLC0R000096, got %s", chassisSN)
	}
	// Real-machine SN assertions (RQ-001 table).
	wantSlots := map[string]string{
		"Slot M1": "G1HLC0R000096",
		"Slot 1":  "G1P7286000129",
		"Slot 2":  "G1N20TZ00011A",
		"Slot 3":  "G1MV41U000047",
		"Slot 4":  "G1NRBA000001B",
		"Slot 5":  "G1MV41U00001C",
		"Slot M2": "G1HLB1R000196",
	}
	for slot, wantSN := range wantSlots {
		comp, ok := bySlot[slot]
		if !ok {
			t.Errorf("expected slot %s in output, not found", slot)
			continue
		}
		if comp.SerialNumber != wantSN {
			t.Errorf("slot %s SN: expected %s, got %s", slot, wantSN, comp.SerialNumber)
		}
		// Slot M1/M2 should be engines; numeric slots should be cards.
		if strings.Contains(slot, "M") {
			if comp.ComponentType != ComponentTypeEngine {
				t.Errorf("slot %s: expected ComponentType=%s, got %s",
					slot, ComponentTypeEngine, comp.ComponentType)
			}
		} else {
			if comp.ComponentType != ComponentTypeCard {
				t.Errorf("slot %s: expected ComponentType=%s, got %s",
					slot, ComponentTypeCard, comp.ComponentType)
			}
		}
		if comp.Source != SourceRuijieCLI {
			t.Errorf("slot %s: expected Source=%s, got %s",
				slot, SourceRuijieCLI, comp.Source)
		}
	}
}

// TestRuijieCliParseInterfacesStatus parses the real Ruijie
// show_interfaces_status fixture. The fixture has many copper ports
// (most up) and many fiber ports (mostly down); we only sanity-check
// that some up ports are returned without asserting a specific count
// (the count is fixture-dependent).
func TestRuijieCliParseInterfacesStatus(t *testing.T) {
	c := NewRuijieCliCollector()
	raw, err := LoadFixture("ruijie_10_62_63_21_show_interfaces_status.txt")
	if err != nil {
		t.Fatalf("LoadFixture error: %v", err)
	}
	up, err := c.ParseInterfacesStatus(raw)
	if err != nil {
		t.Fatalf("ParseInterfacesStatus error: %v", err)
	}
	if len(up) == 0 {
		t.Fatalf("expected at least one up interface, got 0")
	}
	t.Logf("parsed %d up interfaces from ruijie fixture", len(up))
	// Sanity: a known up port from the fixture should appear.
	found := false
	for _, name := range up {
		if strings.HasPrefix(name, "GigabitEthernet 1/2") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected GigabitEthernet 1/2 (status=up in fixture) in up list")
	}
}

// TestRuijieCliParseTransceiverDDM loads the real Ruijie transceiver
// fixture (10GE 1/47 + 1/48 are the only populated SFP+ cages). The
// up-filter is set to admit both interfaces. WARNING 4 mitigation: this
// test exercises the _ddm.textfsm template end-to-end, proving the
// template is referenced by production code (no dead TextFSM templates).
func TestRuijieCliParseTransceiverDDM(t *testing.T) {
	c := NewRuijieCliCollector()
	raw, err := LoadFixture("ruijie_10_62_63_21_show_interfaces_transceiver.txt")
	if err != nil {
		t.Fatalf("LoadFixture error: %v", err)
	}
	comps, err := c.ParseTransceiverDDM(raw, []string{
		"TenGigabitEthernet 1/47",
		"TenGigabitEthernet 1/48",
	})
	if err != nil {
		t.Fatalf("ParseTransceiverDDM error: %v", err)
	}
	if len(comps) != 2 {
		t.Fatalf("expected 2 transceiver components (1/47 + 1/48), got %d", len(comps))
	}
	// Real-machine SN assertions (RQ-001 / show_interfaces_transceiver.txt).
	want := map[string]string{
		"TenGigabitEthernet 1/47": "G1PT549427799",
		"TenGigabitEthernet 1/48": "G1PT54942708A",
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
		// DDM evidence should be captured in Raw (WARNING 4 — prove the
		// _ddm template's Bias/Tx/Rx fields are read).
		if !strings.Contains(comp.Raw, "bias=") {
			t.Errorf("interface %s: DDM evidence missing in Raw (got %q)", comp.Slot, comp.Raw)
		}
	}
}

// TestRuijieTransceiverFiltersDownPorts is the D-10 hard test for the
// Ruijie side. Constructs a fixture with one up port (TenGigabitEthernet
// 1/47) and one down port (TenGigabitEthernet 1/45), passes
// upInterfaces=["TenGigabitEthernet 1/47"], asserts the down port is
// filtered out.
func TestRuijieTransceiverFiltersDownPorts(t *testing.T) {
	c := NewRuijieCliCollector()
	// Minimal transceiver fixture mirroring the real format.
	fixture := strings.Join([]string{
		"========Interface TenGigabitEthernet 1/47========",
		"Transceiver Type    :  10GBASE-SR-SFP+",
		"Vendor Serial Number           : G1PT549427799",
		"",
		"Current diagnostic parameters[AP:Average Power]:",
		"Temp(Celsius)   Voltage(V)      Bias(mA)            RX power(dBm)       TX power(dBm)",
		"39(OK)          3.27(OK)        6.68(OK)            -2.99(OK)[AP]       -2.40(OK)",
		"",
		"========Interface TenGigabitEthernet 1/45========",
		"Transceiver Type    :  10GBASE-SR-SFP+",
		"Vendor Serial Number           : SN-B-DOWN-PORT",
		"",
		"Current diagnostic parameters[AP:Average Power]:",
		"Temp(Celsius)   Voltage(V)      Bias(mA)            RX power(dBm)       TX power(dBm)",
		"40(OK)          3.30(OK)        6.70(OK)            -3.00(OK)[AP]       -2.50(OK)",
		"",
	}, "\n")
	comps, err := c.ParseTransceiverDDM(fixture, []string{"TenGigabitEthernet 1/47"})
	if err != nil {
		t.Fatalf("ParseTransceiverDDM error: %v", err)
	}
	if len(comps) != 1 {
		t.Fatalf("D-10 expected 1 transceiver (down port filtered), got %d: %+v", len(comps), comps)
	}
	if comps[0].Slot != "TenGigabitEthernet 1/47" {
		t.Errorf("expected TenGigabitEthernet 1/47 to survive, got %s", comps[0].Slot)
	}
	if comps[0].SerialNumber != "G1PT549427799" {
		t.Errorf("expected G1PT549427799, got %s", comps[0].SerialNumber)
	}
	// Defensive: explicitly assert the down-port SN never appears.
	for _, comp := range comps {
		if comp.SerialNumber == "SN-B-DOWN-PORT" {
			t.Errorf("D-10 violation: down-port TenGigabitEthernet 1/45 transceiver leaked (SN=%s)",
				comp.SerialNumber)
		}
	}
}

// TestGetCollectorCommands asserts the dispatcher returns the correct
// command lists including the D-10 two-step transceiver pipeline.
func TestGetCollectorCommands(t *testing.T) {
	// Huawei chassis.
	got := GetCollectorCommands(models.VendorHuawei, CmdKindChassis)
	want := []string{"display device esn"}
	if len(got) != len(want) || got[0] != want[0] {
		t.Errorf("huawei/chassis: expected %v, got %v", want, got)
	}

	// Huawei transceiver — D-10 two-step pipeline (status + transceiver).
	got = GetCollectorCommands(models.VendorHuawei, CmdKindTransceiver)
	if len(got) != 2 || got[0] != "display interface status" || got[1] != "display interface transceiver" {
		t.Errorf("huawei/transceiver D-10 pipeline: expected [display interface status, display interface transceiver], got %v", got)
	}

	// Ruijie module.
	got = GetCollectorCommands(models.VendorRuijie, CmdKindModule)
	if len(got) != 1 || got[0] != "show version" {
		t.Errorf("ruijie/module: expected [show version], got %v", got)
	}

	// Ruijie transceiver — D-10 two-step pipeline.
	got = GetCollectorCommands(models.VendorRuijie, CmdKindTransceiver)
	if len(got) != 2 || got[0] != "show interfaces status" || got[1] != "show interface transceiver" {
		t.Errorf("ruijie/transceiver D-10 pipeline: expected [show interfaces status, show interface transceiver], got %v", got)
	}

	// Unknown vendor — should return nil.
	got = GetCollectorCommands(models.VendorH3C, CmdKindChassis)
	if got != nil {
		t.Errorf("h3c/chassis (out of scope): expected nil, got %v", got)
	}

	// Unknown kind for known vendor — should return nil.
	got = GetCollectorCommands(models.VendorHuawei, CmdKindModule)
	if got != nil {
		t.Errorf("huawei/module (not applicable): expected nil, got %v", got)
	}
}
