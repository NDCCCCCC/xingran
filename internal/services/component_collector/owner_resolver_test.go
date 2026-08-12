package component_collector

import (
	"sort"
	"testing"
)

// TestOwnerResolverSingleChassis verifies D-12 ownership resolution in the
// non-stacked case: one chassis (Class=3, ContainedIn=0) plus N children
// whose ContainedIn chain walks back to that chassis.
func TestOwnerResolverSingleChassis(t *testing.T) {
	entities := []EntityRow{
		{Index: 1, Class: 3, ContainedIn: 0, Name: "chassis", Serial: "SN-CHASSIS", Model: "S8700"},
		{Index: 2, Class: 9, ContainedIn: 1, Name: "fan", Serial: "SN-FAN1", Model: "FAN"},
		{Index: 3, Class: 9, ContainedIn: 1, Name: "fan", Serial: "SN-FAN2", Model: "FAN"},
		{Index: 4, Class: 5, ContainedIn: 1, Name: "module", Serial: "SN-MOD1", Model: "MOD"},
		{Index: 5, Class: 8, ContainedIn: 1, Name: "power0", Serial: "SN-PWR1", Model: "PSU"},
		{Index: 6, Class: 8, ContainedIn: 1, Name: "power1", Serial: "SN-PWR2", Model: "PSU"},
	}
	r := &OwnerResolver{}
	owned := r.ResolveOwnership(entities)
	if len(owned) != 6 {
		t.Fatalf("expected 6 OwnedComponent (1 chassis + 5 children), got %d", len(owned))
	}

	// All components (including the chassis itself) should map to chassis index=1
	for _, o := range owned {
		if o.ChassisIndex != 1 {
			t.Errorf("entity %d: expected ChassisIndex=1, got %d", o.Component.Index, o.ChassisIndex)
		}
		if o.Orphan {
			t.Errorf("entity %d: unexpected Orphan=true in single-chassis case", o.Component.Index)
		}
		if o.Component.Index == 1 && o.IsStackMember {
			t.Errorf("canonical root chassis should not be marked stack member")
		}
	}
}

// TestOwnerResolverStackMode verifies D-12 stack handling: two chassis rows
// with ContainedIn=0. First chassis becomes canonical root; second becomes
// a stack member. All components (including second chassis's children)
// attribute to the canonical root.
func TestOwnerResolverStackMode(t *testing.T) {
	entities := []EntityRow{
		{Index: 1, Class: 3, ContainedIn: 0, Name: "chassis-master", Serial: "SN-M"},
		{Index: 2, Class: 5, ContainedIn: 1, Name: "module", Serial: "SN-M-MOD"},
		{Index: 10, Class: 3, ContainedIn: 0, Name: "chassis-slave", Serial: "SN-S"},
		{Index: 11, Class: 5, ContainedIn: 10, Name: "module", Serial: "SN-S-MOD"},
		{Index: 12, Class: 9, ContainedIn: 10, Name: "fan", Serial: "SN-S-FAN"},
	}
	r := &OwnerResolver{}
	owned := r.ResolveOwnership(entities)
	if len(owned) != 5 {
		t.Fatalf("expected 5 OwnedComponent, got %d", len(owned))
	}

	// Identify slave-chassis row and verify it's flagged as stack member
	foundSlave := false
	for _, o := range owned {
		if o.ChassisIndex != 1 {
			t.Errorf("stack: entity %d should attribute to canonical root ChassisIndex=1, got %d",
				o.Component.Index, o.ChassisIndex)
		}
		if o.Component.Index == 10 {
			foundSlave = true
			if !o.IsStackMember {
				t.Errorf("slave chassis (index=10) should be marked IsStackMember=true")
			}
		}
		// Stack members are not orphans
		if o.Orphan {
			t.Errorf("stack: entity %d unexpected Orphan=true", o.Component.Index)
		}
	}
	if !foundSlave {
		t.Errorf("slave chassis row (index=10) missing from output")
	}
}

// TestOwnerResolverOrphanComponent verifies graceful handling when a
// component's ContainedIn points to a non-existent entity index. The
// resolver must mark Orphan=true rather than panic, and walk-cap (32
// iterations) protects against ContainedIn cycles.
func TestOwnerResolverOrphanComponent(t *testing.T) {
	entities := []EntityRow{
		{Index: 1, Class: 3, ContainedIn: 0, Name: "chassis", Serial: "SN-C"},
		// Orphan: ContainedIn=999 references a non-existent entity
		{Index: 2, Class: 5, ContainedIn: 999, Name: "module", Serial: "SN-ORPHAN"},
		// Cycle: 3 → 4 → 3 (ContainedIn loop)
		{Index: 3, Class: 5, ContainedIn: 4, Name: "module-a", Serial: "SN-CY-A"},
		{Index: 4, Class: 5, ContainedIn: 3, Name: "module-b", Serial: "SN-CY-B"},
	}
	r := &OwnerResolver{}
	owned := r.ResolveOwnership(entities)

	orphanIndices := map[int]bool{}
	for _, o := range owned {
		if o.Orphan {
			orphanIndices[o.Component.Index] = true
		}
	}
	if !orphanIndices[2] {
		t.Errorf("entity 2 (ContainedIn=999 missing) should be Orphan=true")
	}
	if !orphanIndices[3] || !orphanIndices[4] {
		t.Errorf("entities 3,4 (ContainedIn cycle) should be Orphan=true")
	}
}

// TestOwnerResolverCanonicalRootPicksFirstByIndex verifies that when
// multiple chassis rows exist, the one with the smallest Index is chosen
// as canonical root (deterministic selection).
func TestOwnerResolverCanonicalRootPicksFirstByIndex(t *testing.T) {
	entities := []EntityRow{
		{Index: 5, Class: 3, ContainedIn: 0, Name: "chassisB", Serial: "B"},
		{Index: 2, Class: 3, ContainedIn: 0, Name: "chassisA", Serial: "A"},
		{Index: 6, Class: 5, ContainedIn: 5, Name: "mod", Serial: "M"},
	}
	r := &OwnerResolver{}
	owned := r.ResolveOwnership(entities)
	for _, o := range owned {
		if o.ChassisIndex != 2 {
			t.Errorf("canonical root should be smallest chassis index (2); got %d for entity %d",
				o.ChassisIndex, o.Component.Index)
		}
	}
	// sanity: input order doesn't matter — re-pass sorted-descending and re-check
	sort.Slice(entities, func(i, j int) bool { return entities[i].Index > entities[j].Index })
	owned2 := r.ResolveOwnership(entities)
	for _, o := range owned2 {
		if o.ChassisIndex != 2 {
			t.Errorf("canonical root should be 2 regardless of input order; got %d", o.ChassisIndex)
		}
	}
}
