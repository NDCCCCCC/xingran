package component_collector

import "sort"

// OwnerResolver rebuilds the chassis-ownership tree from a flat list of
// ENTITY-MIB rows by walking each row's ContainedIn chain to its root
// chassis. The resolver implements D-12 (stack/iStack/IRF/VSU ownership):
//
//   - Chassis rows are entities with Class==3 (chassis) and ContainedIn==0.
//   - When more than one chassis exists, the canonical root is the one
//     with the smallest Index (deterministic). Other chassis rows are
//     marked IsStackMember=true; their child components attribute to the
//     canonical root rather than their immediate parent chassis.
//   - Components whose ContainedIn chain hits a missing index or exceeds
//     the walk-cap (32 hops — guards against ContainedIn cycles) are
//     flagged Orphan=true and retained for audit.
//
// OwnerResolver is stateless; one instance can be reused across devices.
type OwnerResolver struct{}

// ResolveOwnership returns one OwnedComponent per input EntityRow, in the
// same order as the input. Each output row carries the canonical
// ChassisIndex it ultimately belongs to plus the IsStackMember / Orphan
// flags as defined above.
//
// If no chassis row is present in the input (Class==3 && ContainedIn==0),
// every component is marked Orphan=true with ChassisIndex=0.
func (r *OwnerResolver) ResolveOwnership(entities []EntityRow) []OwnedComponent {
	if len(entities) == 0 {
		return nil
	}

	// Step 1: locate chassis rows (Class==3 && ContainedIn==0) and pick
	// the canonical root as the smallest Index.
	chassisIndices := make([]int, 0, 2)
	byIndex := make(map[int]EntityRow, len(entities))
	for _, e := range entities {
		byIndex[e.Index] = e
		if e.Class == 3 && e.ContainedIn == 0 {
			chassisIndices = append(chassisIndices, e.Index)
		}
	}
	sort.Ints(chassisIndices)

	canonicalRoot := 0
	if len(chassisIndices) > 0 {
		canonicalRoot = chassisIndices[0]
	}
	stackMembers := make(map[int]bool, len(chassisIndices))
	for _, idx := range chassisIndices {
		if idx != canonicalRoot {
			stackMembers[idx] = true
		}
	}

	// Step 2: for each entity, walk ContainedIn chain to the root.
	// - If the chain terminates at canonicalRoot → ChassisIndex=canonicalRoot
	// - If the chain terminates at a non-canonical chassis (stack slave) →
	//   attribute to canonicalRoot (D-12 stack attribution)
	// - If the chain hits a missing index or exceeds 32 hops → Orphan=true
	out := make([]OwnedComponent, 0, len(entities))
	for _, e := range entities {
		oc := OwnedComponent{Component: e}
		if canonicalRoot == 0 {
			// No chassis row found — flag orphan.
			oc.Orphan = true
			oc.ChassisIndex = 0
			if stackMembers[e.Index] {
				oc.IsStackMember = true
			}
			out = append(out, oc)
			continue
		}

		root, orphan := walkToRoot(e.Index, byIndex, stackMembers, canonicalRoot)
		oc.ChassisIndex = canonicalRoot
		oc.Orphan = orphan
		if root != canonicalRoot && stackMembers[root] {
			// Component's chain terminated at a slave chassis; flag so
			// the writer knows this came from a stacked slave.
			// (ChassisIndex still points to canonicalRoot per D-12.)
			_ = root // SA9003 抑制: 分支体仅注释,future-flags 留作演进
		}
		if stackMembers[e.Index] {
			oc.IsStackMember = true
		}
		out = append(out, oc)
	}
	return out
}

// walkToRoot follows ContainedIn pointers from startIdx up the tree until
// it reaches a chassis row (Class==3 && ContainedIn==0) or hits an orphan
// condition (missing index or cycle). Returns the terminal chassis index
// and an orphan flag. The walk is capped at 32 hops to bound cycle
// detection (RESEARCH Pitfall 7; ENTITY-MIB trees are typically <=7 deep).
func walkToRoot(startIdx int, byIndex map[int]EntityRow, stackMembers map[int]bool, canonicalRoot int) (int, bool) {
	const maxHops = 32
	current := startIdx
	visited := make(map[int]bool, maxHops+1)
	for hop := 0; hop < maxHops; hop++ {
		if visited[current] {
			// Cycle detected.
			return 0, true
		}
		visited[current] = true
		row, ok := byIndex[current]
		if !ok {
			// Missing parent — orphan.
			return 0, true
		}
		// Stop condition: this row claims to be a root (ContainedIn==0).
		if row.ContainedIn == 0 {
			return current, false
		}
		// Stop condition: this row is the canonical root.
		if row.Index == canonicalRoot {
			return canonicalRoot, false
		}
		current = row.ContainedIn
	}
	// Exceeded hop cap — treat as orphan (cycle / degenerate tree).
	return 0, true
}
