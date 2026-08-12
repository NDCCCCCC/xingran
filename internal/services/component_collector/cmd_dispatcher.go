package component_collector

import "github.com/xingran-next/xingran-go-backend/internal/models"

// CmdKind enumerates the logical CLI command groups a collector can
// request. The dispatcher maps (vendor, kind) → concrete CLI commands.
type CmdKind string

const (
	// CmdKindChassis requests the chassis SN verification command.
	CmdKindChassis CmdKind = "chassis"
	// CmdKindModule requests the per-slot module SN command.
	CmdKindModule CmdKind = "module"
	// CmdKindTransceiver requests the two-step transceiver pipeline
	// (status + transceiver — D-10 enforced ordering).
	CmdKindTransceiver CmdKind = "transceiver"
)

// GetCollectorCommands returns the CLI command list for a given vendor
// and logical kind. Mirrors the dispatch pattern in
// internal/services/portcollection/parser.go but specialised for the
// Phase 48 component collectors.
//
// Per D-10, the transceiver kind returns TWO commands: first the
// interface-status command (used to derive the up-interface list), then
// the transceiver-detail command. The Wave 3 pipeline MUST run them in
// the returned order; flipping the order defeats the D-10 pre-filter.
//
// Returns nil for unknown vendor/kind combinations so callers can
// short-circuit (e.g. H3C / Maipu are out of scope per CONTEXT.md).
func GetCollectorCommands(vendor models.DeviceVendor, kind CmdKind) []string {
	switch vendor {
	case models.VendorHuawei:
		switch kind {
		case CmdKindChassis:
			return []string{"display device esn"}
		case CmdKindTransceiver:
			// D-10 two-step pipeline: status before transceiver.
			return []string{"display interface status", "display interface transceiver"}
		}
	case models.VendorRuijie:
		switch kind {
		case CmdKindModule:
			return []string{"show version"}
		case CmdKindTransceiver:
			// D-10 two-step pipeline: status before transceiver.
			return []string{"show interfaces status", "show interface transceiver"}
		}
	}
	return nil
}
