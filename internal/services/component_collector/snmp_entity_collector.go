package component_collector

import (
	"context"
	"strings"

	"github.com/xingran-next/xingran-go-backend/internal/device"
)

// snmpEntityClasses is the set of ENTITY-MIB class values the SNMP
// collector admits as candidate components. Restricting the Walk to this
// set bounds the GET loop and avoids pulling ~600 sensor rows on Ruijie
// (RESEARCH Pitfall 1).
//
// Class enum per RFC 6933 + vendor quirks (RQ-001):
//   3=chassis, 5=module(Huawei slot placeholder), 6=port(Ruijie power
//   workaround per Open Question 5), 7=fan, 8=powerSupply(Ruijie
//   temprature* noise also sits here — D-11 filter), 9=module(Huawei fan
//   table — Pitfall 2 dual-class dedup).
var snmpEntityClasses = []int{3, 5, 6, 7, 8, 9}

// EntityCollector implements the SNMP ENTITY-MIB single-GET path per
// D-08. It walks entPhysicalClass once to learn the relevant entity index
// set, then issues five single GETs per index to retrieve the per-entity
// attributes (Serial / Model / Name / Class / ContainedIn). The collected
// rows are filtered (D-11 temprature*, dual-class dedup), assembled into
// EntityRow values, fed through OwnerResolver for D-12 ownership, and
// finally projected into a ComponentSet.
//
// The collector depends on the device.SNMPGetter interface so unit tests
// can substitute a stub client without touching the live gosnmp stack.
type EntityCollector struct {
	client    device.SNMPGetter
	resolver  *OwnerResolver
	community string
}

// NewEntityCollector constructs an EntityCollector bound to the supplied
// SNMP client. community is retained for future use (D-08 — community
// is selected from sys_auth_credential.snmp_communities by the Wave 3
// pipeline; the collector itself just stores it for diagnostics).
func NewEntityCollector(client device.SNMPGetter, community string) *EntityCollector {
	return &EntityCollector{
		client:    client,
		resolver:  &OwnerResolver{},
		community: community,
	}
}

// Collect executes the full ENTITY-MIB collection pipeline and returns
// the resulting ComponentSet. The set contains one Chassis (the canonical
// root, possibly nil if the device returned no Class=3/ContainedIn=0
// row) plus zero or more Components (boards / PSUs / fans / modules).
func (c *EntityCollector) Collect(ctx context.Context) (*ComponentSet, error) {
	if c.client == nil {
		return &ComponentSet{}, nil
	}
	indices, err := device.CountPhysicalEntitiesByClass(c.client, snmpEntityClasses)
	if err != nil {
		return nil, err
	}

	// Build EntityRow list, applying D-11 temprature* filter at the source.
	rows := make([]EntityRow, 0, len(indices))
	for idx := range indices {
		attrs, _ := device.GetEntityAttrs(c.client, idx)
		// D-11: drop Class==8 (powerSupply) AND Name starts with "temprature"
		// (Ruijie typo-noise — RESEARCH Pitfall 1). Real PSUs (Class=8,
		// Name="power0"/"power1") are preserved.
		if attrs.Class == 8 && strings.HasPrefix(attrs.Name, "temprature") {
			continue
		}
		rows = append(rows, EntityRow{
			Index:       idx,
			Class:       attrs.Class,
			ContainedIn: attrs.ContainedIn,
			Name:        attrs.Name,
			Serial:      attrs.Serial,
			Model:       attrs.Model,
		})
	}

	// Huawei dual-class dedup (Pitfall 2): same engine reported as
	// module(5, SN empty) + fan(9, SN populated). Walk the row list and,
	// when the same Name prefix appears in both Class=5 (SN empty) and
	// Class=9 (SN populated), drop the empty-SN Class=5 placeholder.
	rows = applyHuaweiDualClassDedup(rows)

	// Resolve D-12 ownership.
	owned := c.resolver.ResolveOwnership(rows)

	// Project to ComponentSet. Chassis = canonical root, Components = rest.
	set := &ComponentSet{Components: []Component{}}
	for _, o := range owned {
		// Skip empty-SN rows except chassis — they are noise placeholders.
		if o.Component.Serial == "" && o.Component.Class != 3 {
			continue
		}
		comp := entityRowToComponent(o.Component)
		if o.Component.Class == 3 && o.ChassisIndex == o.Component.Index && !o.IsStackMember {
			if set.Chassis == nil {
				chassis := comp
				set.Chassis = &chassis
				continue
			}
			// Additional root-level chassis in a non-stack setup: treat
			// as a regular component (rare in practice).
		}
		if o.Orphan {
			// Orphans are retained for audit but flagged in Raw.
			comp.Raw = "orphan:contained-in-chain-unresolved"
		}
		set.Components = append(set.Components, comp)
	}
	return set, nil
}

// applyHuaweiDualClassDedup implements RESEARCH Pitfall 2 dedup:
// when the same entity Name appears as Class=5 with empty SN AND as
// Class=9 with populated SN, drop the Class=5 placeholder.
//
// The dedup is name-prefix-based because Huawei reports the same engine
// as e.g. "LSG7SRUEX1C0 5" (Class=5, SN empty) and "LSG7SRUEX1C0 5
// (Master)" (Class=9, SN populated). The first whitespace-delimited
// token of Name is the dedup key.
func applyHuaweiDualClassDedup(rows []EntityRow) []EntityRow {
	// Index populated-SN rows by first Name token to spot duplicates.
	byNameToken := make(map[string]bool, len(rows))
	for _, r := range rows {
		if r.Class == 9 && r.Serial != "" {
			tok := firstNameToken(r.Name)
			if tok != "" {
				byNameToken[tok] = true
			}
		}
	}
	out := rows[:0]
	for _, r := range rows {
		if r.Class == 5 && r.Serial == "" {
			tok := firstNameToken(r.Name)
			if tok != "" && byNameToken[tok] {
				// Drop — duplicate of an already-populated Class=9 row.
				continue
			}
		}
		out = append(out, r)
	}
	return out
}

// firstNameToken returns the substring of s up to the first whitespace
// character. Empty string when s starts with whitespace or is empty.
func firstNameToken(s string) string {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return ""
	}
	if space := strings.IndexByte(trimmed, ' '); space >= 0 {
		return trimmed[:space]
	}
	return trimmed
}

// entityRowToComponent projects an EntityRow into the collector-agnostic
// Component type, mapping ENTITY-MIB class integers to D-05 component
// type strings.
func entityRowToComponent(r EntityRow) Component {
	return Component{
		ComponentType: classToComponentType(r.Class, r.Name),
		Slot:          r.Name,
		SerialNumber:  r.Serial,
		Model:         r.Model,
		Source:        SourceSNMP,
	}
}

// classToComponentType maps ENTITY-MIB class integers to D-05 component
// type strings. Ruijie reports PSU as Class=6/port with Name="power0"
// (Open Question 5); Huawei reports board engines under Class=9/fan
// (Pitfall 2). The Name prefix is therefore consulted alongside Class.
func classToComponentType(class int, name string) string {
	switch {
	case class == 3:
		return ComponentTypeChassis
	case class == 9 && strings.Contains(strings.ToLower(name), "fan"):
		return ComponentTypeFan
	case class == 9:
		// Huawei dual-class: same engine reported under fan(9) but the
		// Name doesn't say "fan" — this is an engine/card in disguise.
		return ComponentTypeEngine
	case class == 5:
		return ComponentTypeCard
	case class == 8 && strings.HasPrefix(name, "power"):
		return ComponentTypePower
	case class == 6 && strings.HasPrefix(name, "power"):
		// Ruijie PSU workaround (Open Question 5).
		return ComponentTypePower
	case class == 8:
		return ComponentTypeFan // generic sensor fallback
	case class == 7:
		return ComponentTypeFan
	default:
		return ComponentTypeCard
	}
}
