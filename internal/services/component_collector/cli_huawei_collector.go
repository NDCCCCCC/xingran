package component_collector

import (
	"fmt"
	"strings"

	"github.com/xingran-next/xingran-go-backend/internal/templates"
)

// HuaweiCliCollector parses the three Huawei VRP CLI outputs that the
// ENTITY-MIB cannot supply on its own (D-09):
//
//   - display device esn                — chassis ESN cross-verification
//   - display device elabel brief       — chassis ESN fallback (V600R024C00+)
//                                         + per-slot/per-board/per-fan/per-power SN
//                                         (D-12: replaces deferred ENTITY-MIB board path)
//   - display interface status          — D-10 up-interface pre-filter
//   - display interface transceiver     — optical-module SN + DDM
//
// Per D-10, transceiver parsing MUST be gated by the up-interface list
// returned by ParseInterfaceStatus. The collector never the less exposes
// ParseInterfaceTransceiver with an explicit upInterfaces parameter so
// the Wave 3 pipeline can supply the list directly when status parsing
// is bypassed.
//
// Templates are loaded via the project's homegrown textfsm parser
// (internal/templates/textfsm.go) — no external regex/TextFSM dependency.
type HuaweiCliCollector struct {
	// tmplPath maps a logical template name to its path under templates/.
	// Exposed as a struct field so tests can substitute alternative
	// templates without modifying production paths.
	tmplPath huaweiTemplatePaths
}

// huaweiTemplatePaths groups the three Huawei template paths so they can
// be overridden in tests if necessary.
type huaweiTemplatePaths struct {
	esn         string
	elabelBrief string
	status      string
	transceiver string
}

// DefaultHuaweiTemplatePaths returns the production template paths used
// by NewHuaweiCliCollector.
func DefaultHuaweiTemplatePaths() huaweiTemplatePaths {
	return huaweiTemplatePaths{
		esn:         "templates/huawei_vrp_display_device_esn.textfsm",
		elabelBrief: "templates/huawei_vrp_display_device_elabel_brief.textfsm",
		status:      "templates/huawei_vrp_display_interface_status.textfsm",
		transceiver: "templates/huawei_vrp_display_interface_transceiver.textfsm",
	}
}

// NewHuaweiCliCollector constructs a collector bound to the default
// production template paths.
func NewHuaweiCliCollector() *HuaweiCliCollector {
	return &HuaweiCliCollector{tmplPath: DefaultHuaweiTemplatePaths()}
}

// ParseDisplayDeviceEsn extracts chassis ESN components from the output
// of `display device esn`. Returns an empty slice (and nil error) when
// the device replied with "Error: Unrecognized command" — Huawei
// V600R024C00 retired this command on some chassis (Pitfall 3), so the
// collector treats the absence as "no data" rather than a failure.
func (h *HuaweiCliCollector) ParseDisplayDeviceEsn(raw string) ([]Component, error) {
	records, err := parseWithTemplate(h.tmplPath.esn, raw)
	if err != nil {
		return nil, fmt.Errorf("huawei esn: %w", err)
	}
	out := make([]Component, 0, len(records))
	for _, r := range records {
		esn := r["ESN"]
		if esn == "" {
			continue
		}
		out = append(out, Component{
			ComponentType: ComponentTypeChassis,
			Slot:          fmt.Sprintf("chassis %s", r["ChassisID"]),
			SerialNumber:  esn,
			Source:        SourceHuaweiCLI,
		})
	}
	return out, nil
}

// ParseDisplayDeviceElabelBrief parses the output of
// `display device elabel brief` and returns one Component per row
// (BackPlane chassis, business card, engine, fan tray, power supply).
//
// Phase 49-D-12: this is the consolidated board/fan/power SN source
// for Huawei VRP. Replaces the deferred D-08 ENTITY-MIB board path
// for devices that have `display device elabel brief` (most S5700+
// running V200R003 or later). For legacy devices without this
// command, ParseDisplayDeviceEsn still covers chassis SN and the
// pipeline silently skips board collection (same D-14 tolerance as
// before).
//
// Returns:
//   - chassis row (BackPlane, SLOTID="--") → ComponentTypeChassis, dropped
//     by the caller to avoid duplicating ParseDisplayDeviceEsn's chassis row
//   - numeric slot + TYPE matches SRU/MPU/SFU model → ComponentTypeEngine
//   - numeric slot + other TYPE                 → ComponentTypeCard
//   - SLOTID prefix FAN                         → ComponentTypeFan
//   - SLOTID prefix PWR                         → ComponentTypePower
//   - table header row (SLOTID literal "SlotID") → silently skipped
//
// Pitfall: the textual table header line
//   `SlotID     Sub    Type                     SN                       P/N`
// matches the data row template (SLOTID="SlotID" TYPE="Type" SN="SN" PN="P/N")
// because each "value" is a non-space token. The header detection is
// done here (SLOTID == "SlotID") rather than in the textfsm template
// to keep the template focused on real data rows.
func (h *HuaweiCliCollector) ParseDisplayDeviceElabelBrief(raw string) ([]Component, error) {
	records, err := parseWithTemplate(h.tmplPath.elabelBrief, raw)
	if err != nil {
		return nil, fmt.Errorf("huawei elabel brief: %w", err)
	}
	out := make([]Component, 0, len(records))
	for _, r := range records {
		slotID := strings.TrimSpace(r["SLOTID"])
		typ := strings.TrimSpace(r["TYPE"])
		sn := strings.TrimSpace(r["SN"])
		if slotID == "" || sn == "" {
			continue
		}
		// Skip textual table header (the column-name row).
		if slotID == "SlotID" || slotID == "Slot" {
			continue
		}
		// Skip BackPlane (chassis) — already handled by ParseDisplayDeviceEsn
		// (and emitting it here would double-count when both commands run).
		if slotID == "--" || typ == "BackPlane" {
			continue
		}
		out = append(out, Component{
			ComponentType: huaweiElabelComponentType(slotID, typ),
			Slot:          fmt.Sprintf("Slot %s", slotID),
			SerialNumber:  sn,
			Model:         typ,
			Source:        SourceHuaweiCLI,
		})
	}
	return out, nil
}

// huaweiElabelComponentType maps a (SLOTID, TYPE) pair from
// `display device elabel brief` to a ComponentType. Heuristics:
//
//   - "FAN*" → fan   (fan tray slots)
//   - "PWR*" → power (PSU slots)
//   - numeric + TYPE contains SRU/MPU/SFU → engine
//     (HUAWEI main control / switch fabric units; e.g. S8700 LSG7SRUEX1C0)
//   - numeric + other TYPE → card (business line card)
//
// The TYPE→engine detection is intentionally substring-based: SRU
// (Switch Route Unit) covers S5700/S6700/S7700/S8700/S9700 chassis;
// MPU (Main Processing Unit) covers modular chassis like S9300/S9306;
// SFU (Switch Fabric Unit) covers some chassis' fabric modules.
func huaweiElabelComponentType(slotID, typ string) string {
	switch {
	case strings.HasPrefix(slotID, "FAN"):
		return ComponentTypeFan
	case strings.HasPrefix(slotID, "PWR"):
		return ComponentTypePower
	case strings.Contains(typ, "SRU"), strings.Contains(typ, "MPU"), strings.Contains(typ, "SFU"):
		return ComponentTypeEngine
	default:
		return ComponentTypeCard
	}
}

// ParseInterfaceStatus extracts the list of "up" interface names from
// `display interface status`. The returned slice feeds
// ParseInterfaceTransceiver as the D-10 up-interface filter.
func (h *HuaweiCliCollector) ParseInterfaceStatus(raw string) ([]string, error) {
	records, err := parseWithTemplate(h.tmplPath.status, raw)
	if err != nil {
		return nil, fmt.Errorf("huawei status: %w", err)
	}
	out := make([]string, 0, len(records))
	for _, r := range records {
		if r["STATUS"] == "up" {
			if name := r["INTERFACE"]; name != "" {
				out = append(out, name)
			}
		}
	}
	return out, nil
}

// ParseInterfaceTransceiver parses `display interface transceiver`
// output and applies the D-10 up-interface filter. When upInterfaces is
// non-nil, only transceiver blocks whose Interface column matches an
// entry in upInterfaces are retained (RESEARCH Pitfall 5).
//
// Passing upInterfaces=nil disables the filter (used by tests that
// exercise the parser directly without a status fixture). Production
// callers MUST always supply a non-nil slice — typically the output of
// ParseInterfaceStatus.
func (h *HuaweiCliCollector) ParseInterfaceTransceiver(raw string, upInterfaces []string) ([]Component, error) {
	records, err := parseWithTemplate(h.tmplPath.transceiver, raw)
	if err != nil {
		return nil, fmt.Errorf("huawei transceiver: %w", err)
	}
	upSet := make(map[string]bool, len(upInterfaces))
	for _, n := range upInterfaces {
		upSet[n] = true
	}
	filterActive := len(upInterfaces) > 0

	out := make([]Component, 0, len(records))
	for _, r := range records {
		iface := r["INTERFACE"]
		sn := r["SN"]
		if iface == "" || sn == "" {
			continue
		}
		if filterActive && !upSet[iface] {
			// D-10: drop transceivers on non-up interfaces.
			continue
		}
		out = append(out, Component{
			ComponentType: ComponentTypeTransceiver,
			Slot:          iface,
			SerialNumber:  sn,
			Source:        SourceHuaweiCLI,
		})
	}
	return out, nil
}

// parseWithTemplate is the shared helper that loads a TextFSM template
// via the project parser and runs it against raw. All collector
// implementations delegate here so template-loading conventions stay in
// one place.
func parseWithTemplate(path, raw string) ([]map[string]string, error) {
	fsm, err := templates.ParseTemplate(path)
	if err != nil {
		return nil, err
	}
	return fsm.ParseText(raw)
}
