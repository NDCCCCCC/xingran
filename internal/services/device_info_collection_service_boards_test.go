package services

import (
	"testing"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/component_collector"
)

// TestCollectBoardsInto_RuijieBoards verifies that the ruijie `show version`
// output is parsed by ParseShowVersionModules and the engine/card board rows
// (M1, Slot 1-5, M2) are appended to set.Components — at least 6 boards.
// This is the core Gap-1 fix: collectComponentInfo must actually collect
// boards, not just transceivers.
func TestCollectBoardsInto_RuijieBoards(t *testing.T) {
	raw := loadSampleFixture(t, "ruijie_10_62_63_21_show_version.txt")
	runner := func(cmd string) (string, error) {
		if cmd == "show version" {
			return raw, nil
		}
		return "", nil
	}
	set := &component_collector.ComponentSet{Components: []component_collector.Component{}}

	device := &models.NetworkDevice{Vendor: models.VendorRuijie}
	if err := collectBoardsInto(device, runner, set); err != nil {
		t.Fatalf("collectBoardsInto: unexpected error: %v", err)
	}

	// Count engine/card boards — must be >= 6 (M1/1-5/M2 from fixture).
	boardCount := 0
	for _, c := range set.Components {
		if c.ComponentType == component_collector.ComponentTypeEngine ||
			c.ComponentType == component_collector.ComponentTypeCard {
			boardCount++
		}
	}
	if boardCount < 6 {
		t.Fatalf("expected >= 6 board components (engine/card), got %d", boardCount)
	}
}

// TestCollectBoardsInto_ChassisRowDropped asserts the BLOCKER-2 invariant:
// the chassis row from ParseShowVersionModules is NOT appended to
// set.Components. The chassis asset already exists (parent device), and
// Pipeline.Run only consumes set.Components, never *set.Chassis. Therefore
// chassis rows must be dropped here so they are never written to ops_asset.
//
// This test also verifies set.Chassis stays nil.
func TestCollectBoardsInto_ChassisRowDropped(t *testing.T) {
	raw := loadSampleFixture(t, "ruijie_10_62_63_21_show_version.txt")
	runner := func(cmd string) (string, error) {
		if cmd == "show version" {
			return raw, nil
		}
		return "", nil
	}
	set := &component_collector.ComponentSet{Components: []component_collector.Component{}}

	device := &models.NetworkDevice{Vendor: models.VendorRuijie}
	if err := collectBoardsInto(device, runner, set); err != nil {
		t.Fatalf("collectBoardsInto: unexpected error: %v", err)
	}

	if set.Chassis != nil {
		t.Errorf("set.Chassis MUST stay nil (Pipeline.Run never consumes it); got %+v", set.Chassis)
	}
	for _, c := range set.Components {
		if c.ComponentType == component_collector.ComponentTypeChassis {
			t.Errorf("chassis row must NOT be appended to set.Components: found %+v", c)
		}
	}
}

// TestCollectBoardsInto_HuaweiNoBoards verifies the D-08 deferred semantic:
// huawei board collection goes through ENTITY-MIB SNMP, which is deferred in
// this environment. So huawei must NOT issue any module command —
// collectBoardsInto must be a no-op for huawei. (49-01 handles huawei chassis
// SN writeback to sys_network_device; board collection stays deferred.)
func TestCollectBoardsInto_HuaweiNoBoards(t *testing.T) {
	called := false
	runner := func(cmd string) (string, error) {
		called = true
		return "", nil
	}
	set := &component_collector.ComponentSet{Components: []component_collector.Component{}}

	device := &models.NetworkDevice{Vendor: models.VendorHuawei}
	if err := collectBoardsInto(device, runner, set); err != nil {
		t.Fatalf("collectBoardsInto: unexpected error: %v", err)
	}
	if called {
		t.Errorf("huawei must NOT issue any module command (ENTITY-MIB deferred); runner was called")
	}
	if len(set.Components) != 0 {
		t.Errorf("huawei must NOT append any component; got %d", len(set.Components))
	}
	if set.Chassis != nil {
		t.Errorf("huawei must keep set.Chassis nil; got %+v", set.Chassis)
	}
}

// TestCollectBoardsInto_OutOfScopeVendor asserts the existing short-circuit
// semantic is preserved: H3C / Maipu / unknown vendors are out of scope (per
// v1.18 D-08 ENTITY-MIB deferred and the existing collectComponentInfo line
// 691-694 short-circuit). collectBoardsInto must NOT call the runner and
// must NOT append anything.
func TestCollectBoardsInto_OutOfScopeVendor(t *testing.T) {
	called := false
	runner := func(cmd string) (string, error) {
		called = true
		return "", nil
	}
	set := &component_collector.ComponentSet{Components: []component_collector.Component{}}

	device := &models.NetworkDevice{Vendor: models.VendorH3C}
	if err := collectBoardsInto(device, runner, set); err != nil {
		t.Fatalf("collectBoardsInto: unexpected error: %v", err)
	}
	if called {
		t.Errorf("out-of-scope vendor (H3C) must NOT trigger runner; runner was called")
	}
	if len(set.Components) != 0 {
		t.Errorf("out-of-scope vendor must NOT append components; got %d", len(set.Components))
	}
}

// TestCollectBoardsInto_CommandErrorTolerated verifies D-14 fault tolerance:
// if the SSH command execution fails, collectBoardsInto MUST log + return
// nil (not block the transceiver pipeline that runs after it).
func TestCollectBoardsInto_CommandErrorTolerated(t *testing.T) {
	runner := func(cmd string) (string, error) {
		return "", errFakeCommandFailure
	}
	set := &component_collector.ComponentSet{Components: []component_collector.Component{}}

	device := &models.NetworkDevice{Vendor: models.VendorRuijie}
	if err := collectBoardsInto(device, runner, set); err != nil {
		t.Fatalf("D-14: command error MUST be tolerated (return nil); got %v", err)
	}
	if len(set.Components) != 0 {
		t.Errorf("on command error, set.Components must remain empty; got %d", len(set.Components))
	}
}

// errFakeCommandFailure is a sentinel for TestCollectBoardsInto_CommandErrorTolerated.
var errFakeCommandFailure = errFake("ssh command failed")

type errFake string

func (e errFake) Error() string { return string(e) }
