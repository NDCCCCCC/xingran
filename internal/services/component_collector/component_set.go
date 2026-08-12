// Package component_collector implements the Phase 48 Wave 2 collectors that
// produce ComponentSet data structures from SNMP ENTITY-MIB and CLI output
// (Huawei display / Ruijie show). The package ONLY parses — it never writes
// to ops_asset nor emits reconciliation anomalies. Wave 3 (48-03) wires the
// ComponentSet into the device_info_collection_service pipeline.
//
// This file defines the pure data types shared across all collectors.
package component_collector

// Component type enumerations (D-05). Stored as plain strings in
// ops_asset.component_type (VARCHAR(32), no PG CHECK constraint — dictionary
// driven so future types like sled/fabric don't require a schema change).
const (
	// ComponentTypeChassis is the canonical root entity of a device frame.
	ComponentTypeChassis = "chassis"
	// ComponentTypeCard is a line-card / service-board plugged into a slot.
	ComponentTypeCard = "card"
	// ComponentTypeEngine is a master/slave routing engine (e.g. M1/M2).
	ComponentTypeEngine = "engine"
	// ComponentTypePower is a power supply unit (PSU).
	ComponentTypePower = "power"
	// ComponentTypeFan is a fan tray / fan module.
	ComponentTypeFan = "fan"
	// ComponentTypeTransceiver is a pluggable optical module (SFP/QSFP).
	ComponentTypeTransceiver = "transceiver"
)

// Source enumerations identify which collector produced a Component.
const (
	// SourceSNMP marks components collected via ENTITY-MIB single-GET.
	SourceSNMP = "snmp"
	// SourceHuaweiCLI marks components parsed from Huawei display CLI output.
	SourceHuaweiCLI = "cli-huawei"
	// SourceRuijieCLI marks components parsed from Ruijie show CLI output.
	SourceRuijieCLI = "cli-ruijie"
)

// Component is the collector-agnostic representation of one serialised
// field-replaceable unit (chassis, board, PSU, fan, transceiver).
//
// It is the smallest unit the Wave 3 ops_asset_writer consumes: the writer
// looks up ops_asset by SerialNumber and UPDATEs parent_asset_id /
// source_device_id / component_type / component_slot. Component therefore
// carries everything the writer needs except the relational IDs which are
// derived at write time.
type Component struct {
	// ComponentType is one of ComponentType* constants above.
	ComponentType string `json:"componentType"`
	// Slot is the human-readable physical location, e.g. "Slot 1", "PWR1",
	// "10GE5/0/4". Forwarded to ops_asset.component_slot on UPDATE.
	Slot string `json:"slot"`
	// SerialNumber is the factory SN. Empty for placeholder rows from SNMP
	// (Huawei dual-class module(5) row); those rows are dropped by the
	// SNMP collector before reaching Component.
	SerialNumber string `json:"serialNumber"`
	// Model is the vendor part number (e.g. "M8600E-24GT20SFP4XS-ED").
	Model string `json:"model"`
	// Source identifies which collector produced this Component. One of
	// SourceSNMP / SourceHuaweiCLI / SourceRuijieCLI.
	Source string `json:"source"`
	// Raw is a free-form evidence blob (the original CLI block or SNMP
	// OID values) preserved for audit/debugging. May be empty.
	Raw string `json:"raw,omitempty"`
}

// ComponentSet is the aggregate output of one device collection cycle.
// Wave 3 pipeline receives this and feeds Chassis + Components into the
// ops_asset writer.
//
// Chassis may be nil if no canonical root entity could be identified
// (e.g. all entities had ContainedIn cycles — extremely unlikely in
// practice but defensively represented).
type ComponentSet struct {
	// Chassis is the canonical root component (the device frame itself).
	// In stack mode (iStack/IRF/VSU), this is the master chassis only;
	// slave chassis appear as Components with IsStackMember=true.
	Chassis *Component `json:"chassis"`
	// Components lists every non-chassis serialised field-replaceable
	// unit, including stack-member chassis rows.
	Components []Component `json:"components"`
}

// EntityRow is the intermediate representation of one ENTITY-MIB table row
// after the SNMP single-GET loop decodes the 5 attributes (Serial, Model,
// Name, Class, ContainedIn). It is the input to OwnerResolver.
//
// Class uses ENTITY-MIB INTEGER enum semantics:
//   3=chassis, 5=module, 6=port, 7=port (overlap), 8=powerSupply, 9=fan.
//
// (Ruijie reports PSU as class=6/port with Name="power0" — see RESEARCH
// Pitfall 1 / Open Question 5; the collector maps this to power.)
type EntityRow struct {
	Index        int    `json:"index"`
	Class        int    `json:"class"`
	ContainedIn  int    `json:"containedIn"`
	Name         string `json:"name"`
	Serial       string `json:"serial"`
	Model        string `json:"model"`
}

// OwnedComponent is the output of OwnerResolver.ResolveOwnership: it
// wraps an EntityRow with ownership metadata derived from the
// entPhysicalContainedIn tree.
type OwnedComponent struct {
	// Component is the original entity row.
	Component EntityRow `json:"component"`
	// ChassisIndex is the canonical root chassis EntityRow.Index that
	// this component ultimately belongs to. For stack modes this is the
	// master chassis even when ContainedIn chains through a slave.
	ChassisIndex int `json:"chassisIndex"`
	// IsStackMember is true for non-canonical chassis rows (slave chassis
	// in an iStack/IRF/VSU). The Wave 3 writer treats these as children
	// of the canonical root.
	IsStackMember bool `json:"isStackMember"`
	// Orphan is true when ContainedIn points to a missing entity index
	// or contains a cycle. Orphan components are retained for audit but
	// flagged so the writer can decide how to handle them.
	Orphan bool `json:"orphan"`
}
