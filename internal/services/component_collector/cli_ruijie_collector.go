package component_collector

import (
	"fmt"
	"strings"
)

// RuijieCliCollector parses the three Ruijie RGOS CLI outputs that
// ENTITY-MIB cannot supply on its own (D-09):
//
//   - show version              — chassis SN + 7 module slot SNs (one-shot)
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

// ruijieTemplatePaths groups the three Ruijie template paths.
type ruijieTemplatePaths struct {
	version     string
	status      string
	transceiver string
}

// DefaultRuijieTemplatePaths returns the production template paths.
func DefaultRuijieTemplatePaths() ruijieTemplatePaths {
	return ruijieTemplatePaths{
		version:     "templates/ruijie_os_show_version_modules.textfsm",
		status:      "templates/ruijie_os_show_interfaces_status.textfsm",
		transceiver: "templates/ruijie_os_show_interfaces_transceiver_ddm.textfsm",
	}
}

// NewRuijieCliCollector constructs a collector bound to default paths.
func NewRuijieCliCollector() *RuijieCliCollector {
	return &RuijieCliCollector{tmplPath: DefaultRuijieTemplatePaths()}
}

// ParseShowVersionModules parses `show version` output, returning one
// chassis Component plus one Component per populated module slot. Slot
// "M1"/"M2" → ComponentTypeEngine; numeric slots → ComponentTypeCard.
func (r *RuijieCliCollector) ParseShowVersionModules(raw string) ([]Component, error) {
	records, err := parseWithTemplate(r.tmplPath.version, raw)
	if err != nil {
		return nil, fmt.Errorf("ruijie version: %w", err)
	}
	out := make([]Component, 0, len(records))
	for _, rec := range records {
		// Chassis row — emitted by the first Start rule capturing
		// System serial number.
		if chassisSN := rec["CHASSIS_SN"]; chassisSN != "" && rec["SLOT"] == "" {
			out = append(out, Component{
				ComponentType: ComponentTypeChassis,
				Slot:          "chassis",
				SerialNumber:  chassisSN,
				Source:        SourceRuijieCLI,
			})
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
