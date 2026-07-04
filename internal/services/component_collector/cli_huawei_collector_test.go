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
