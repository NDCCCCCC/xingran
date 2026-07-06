package component_collector

import (
	"fmt"
	"strings"
)

// RuijieCliCollector parses the three Ruijie RGOS CLI outputs that
// ENTITY-MIB cannot supply on its own (D-09):
//
//   - show version              — per-slot module SNs (Slot M1/M2/1..N)
//   - show manuinfo             — physical chassis SN + per-slot SNs
//                                 (D-11: show version "System serial number"
//                                 is the active M1 engine's SN, NOT the
//                                 chassis SN — see ParseShowManuinfo doc)
//   - show interfaces status    — D-10 up-interface pre-filter
//   - show interface transceiver — optical-module Vendor SN + DDM
//
// Per D-10, transceiver parsing MUST be gated by the up-interface list
// returned by ParseInterfacesStatus. The collector exposes
// ParseTransceiverDDM with an explicit upInterfaces parameter so the
// Wave 3 pipeline can supply the list directly when status parsing is
// bypassed.
//
// Templates are loaded via the project's homegrown textfsm parser
// (internal/templates/textfsm.go) — no external regex/TextFSM dependency.
type RuijieCliCollector struct {
	tmplPath ruijieTemplatePaths
}

// ruijieTemplatePaths groups the Ruijie template paths.
type ruijieTemplatePaths struct {
	version     string
	manuinfo    string
	status      string
	transceiver string
}

// DefaultRuijieTemplatePaths returns the production template paths.
func DefaultRuijieTemplatePaths() ruijieTemplatePaths {
	return ruijieTemplatePaths{
		version:     "templates/ruijie_os_show_version_modules.textfsm",
		manuinfo:    "templates/ruijie_os_show_manuinfo.textfsm",
		status:      "templates/ruijie_os_show_interfaces_status.textfsm",
		transceiver: "templates/ruijie_os_show_interfaces_transceiver_ddm.textfsm",
	}
}

// NewRuijieCliCollector constructs a collector bound to default paths.
func NewRuijieCliCollector() *RuijieCliCollector {
	return &RuijieCliCollector{tmplPath: DefaultRuijieTemplatePaths()}
}

// ParseShowVersionModules parses `show version` output, returning one
// Component per populated module slot (engine or card). It DOES NOT
// emit the chassis row — see ParseShowManuinfo for that.
//
// Phase 49-D-11 history: this method previously also emitted a
// ComponentTypeChassis row sourced from the "System serial number"
// line. That field is actually the active M1 routing-engine SN, not
// the physical chassis SN (verified on RS8607E-03 10.62.63.23: System
// serial = G1M9140000175 == Slot M1 SN, while the real chassis SN
// G1M913U000351 lives only in `show manuinfo` Device 1 / Chassis).
// The chassis row was removed from this output to prevent downstream
// code from continuing to write the M1 SN into sys_network_device.serial_number.
func (r *RuijieCliCollector) ParseShowVersionModules(raw string) ([]Component, error) {
	records, err := parseWithTemplate(r.tmplPath.version, raw)
	if err != nil {
		return nil, fmt.Errorf("ruijie version: %w", err)
	}
	out := make([]Component, 0, len(records))
	for _, rec := range records {
		// Phase 49-D-11: ignore the chassis row emitted by the
		// version template — "System serial number" is the M1
		// engine SN, not the chassis SN. Use ParseShowManuinfo
		// for the chassis row.
		if chassisSN := rec["CHASSIS_SN"]; chassisSN != "" && rec["SLOT"] == "" {
			continue
		}
		sn := rec["SN"]
		slot := rec["SLOT"]
		moduleType := rec["MODULETYPE"]
		if slot == "" || sn == "" {
			continue
		}
		out = append(out, Component{
			ComponentType: slotToComponentType(slot),
			Slot:          fmt.Sprintf("Slot %s", slot),
			SerialNumber:  sn,
			Model:         moduleType,
			Source:        SourceRuijieCLI,
		})
	}
	return out, nil
}

// ParseShowManuinfo parses `show manuinfo` output, returning one
// chassis Component (Location="Chassis") plus one Component per
// populated module slot (Location="Slot-N" / "Slot-M1" / "Slot-M2").
//
// Phase 49-D-11: this is the authoritative source for the PHYSICAL
// chassis serial number on Ruijie RGOS. `show version` is misleading
// — its "System serial number" line reports the active M1 routing
// engine's SN, which equals Slot M1's SN. The physical chassis SN
// only appears here as Device 1 / Location: Chassis / Device Serial
// Number.
//
// Slot mapping (D-11):
//   - "Slot-M1" / "Slot-M2" → ComponentTypeEngine
//   - "Slot-N" (numeric N)  → ComponentTypeCard
//   - "Chassis"             → ComponentTypeChassis (one per device)
func (r *RuijieCliCollector) ParseShowManuinfo(raw string) ([]Component, error) {
	records, err := parseWithTemplate(r.tmplPath.manuinfo, raw)
	if err != nil {
		return nil, fmt.Errorf("ruijie manuinfo: %w", err)
	}
	out := make([]Component, 0, len(records))
	for _, rec := range records {
		location := strings.TrimSpace(rec["LOCATION"])
		sn := strings.TrimSpace(rec["SN"])
		if location == "" || sn == "" {
			continue
		}
		switch {
		case location == "Chassis":
			out = append(out, Component{
				ComponentType: ComponentTypeChassis,
				Slot:          "chassis",
				SerialNumber:  sn,
				Source:        SourceRuijieCLI,
			})
		case strings.HasPrefix(location, "Slot-"):
			// Strip the "Slot-" prefix to recover the bare slot
			// identifier ("M1", "M2", "1", "2", ...) used by
			// slotToComponentType.
			bareSlot := strings.TrimPrefix(location, "Slot-")
			out = append(out, Component{
				ComponentType: slotToComponentType(bareSlot),
				Slot:          fmt.Sprintf("Slot %s", bareSlot),
				SerialNumber:  sn,
				Model:         strings.TrimSpace(rec["DEVICE_NAME"]),
				Source:        SourceRuijieCLI,
			})
		default:
			// Unknown location tag — ignore rather than fabricate
			// a ComponentType. Audit log via returned error so the
			// caller can surface it; per D-14 we keep collecting
			// the rest of the records.
			continue
		}
	}
	return out, nil
}

// ParseInterfacesStatus extracts the list of "up" interface names from
// `show interfaces status`. The returned slice feeds ParseTransceiverDDM
// as the D-10 up-interface filter.
func (r *RuijieCliCollector) ParseInterfacesStatus(raw string) ([]string, error) {
	records, err := parseWithTemplate(r.tmplPath.status, raw)
	if err != nil {
		return nil, fmt.Errorf("ruijie status: %w", err)
	}
	out := make([]string, 0, len(records))
	for _, rec := range records {
		if rec["STATUS"] == "up" {
			name := strings.TrimSpace(rec["INTERFACE"])
			if name != "" {
				out = append(out, name)
			}
		}
	}
	return out, nil
}

// ParseTransceiverDDM parses `show interface transceiver` output and
// applies the D-10 up-interface filter. When upInterfaces is non-nil,
// only transceiver blocks whose Interface matches an entry in
// upInterfaces are retained (RESEARCH Pitfall 5).
//
// This method is referenced by TestRuijieCliParseTransceiverDDM to
// eliminate the WARNING 4 dead-template risk for
// ruijie_os_show_interfaces_transceiver_ddm.textfsm.
//
// Passing upInterfaces=nil disables the filter (used by tests that
// exercise the parser directly without a status fixture). Production
// callers MUST always supply a non-nil slice — typically the output of
// ParseInterfacesStatus.
func (r *RuijieCliCollector) ParseTransceiverDDM(raw string, upInterfaces []string) ([]Component, error) {
	records, err := parseWithTemplate(r.tmplPath.transceiver, raw)
	if err != nil {
		return nil, fmt.Errorf("ruijie transceiver ddm: %w", err)
	}
	upSet := make(map[string]bool, len(upInterfaces))
	for _, n := range upInterfaces {
		upSet[strings.TrimSpace(n)] = true
	}
	filterActive := len(upInterfaces) > 0

	out := make([]Component, 0, len(records))
	for _, rec := range records {
		iface := strings.TrimSpace(rec["INTERFACE"])
		sn := rec["SN"]
		if iface == "" || sn == "" {
			continue
		}
		if filterActive && !upSet[iface] {
			// D-10: drop transceivers on non-up interfaces.
			continue
		}
		// Compose Raw with DDM evidence so downstream audit can verify
		// the DDM fields were extracted (WARNING 4 mitigation).
		rawEvidence := fmt.Sprintf("bias=%s txpower=%s rxpower=%s",
			rec["BIAS"], rec["TX_POWER"], rec["RX_POWER"])
		out = append(out, Component{
			ComponentType: ComponentTypeTransceiver,
			Slot:          iface,
			SerialNumber:  sn,
			Source:        SourceRuijieCLI,
			Raw:           rawEvidence,
		})
	}
	return out, nil
}

// slotToComponentType maps a Ruijie slot identifier to a D-05 component
// type. M1/M2 are master/slave routing engines; numeric slots are line
// cards.
func slotToComponentType(slot string) string {
	switch slot {
	case "M1", "M2":
		return ComponentTypeEngine
	default:
		return ComponentTypeCard
	}
}