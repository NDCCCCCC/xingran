// Package device — snmp_entity_mib.go provides high-level helpers around
// RFC 6933 ENTITY-MIB for Phase 48 Wave 2 component serial collection.
//
// Per D-08, the collector uses single GET (not GetBulk — Huawei S8700
// rejects GetBulk). The two-step strategy:
//
//   1. CountPhysicalEntitiesByClass — one Walk on oidEntPhysicalClass
//      filtered by the caller-supplied class set (chassis/module/fan/
//      powerSupply/etc.) collects the set of relevant entity indices.
//      This bounds the subsequent GET loop and avoids pulling ~600
//      sensor rows on Ruijie.
//   2. GetEntityAttrs — for each index, five single GETs retrieve
//      Serial / Model / Name / Class / ContainedIn.
//
// The helpers type-switch on the interface{} returned by SNMPClient.Get
// (gosnmp values may arrive as string, int64 or other). parseSNMPValue
// in snmp_client.go is package-private so this file re-implements the
// narrow type switch locally (RESEARCH §Code Examples #1).
package device

import (
	"fmt"
	"strconv"
	"strings"
)

// ENTITY-MIB OID base values per RFC 6933 (entPhysicalTable entry columns).
// The full per-entity OID is "<base>.<entityIndex>". Exported so that
// callers in component_collector (and other future packages) can build
// per-index OIDs and test fixtures without re-declaring the constants.
const (
	// OidEntPhysicalSerialNum — .1.3.6.1.2.1.47.1.1.1.1.11 (DisplayString)
	// Vendor serial number; primary key for ops_asset lookup.
	OidEntPhysicalSerialNum = "1.3.6.1.2.1.47.1.1.1.1.11"
	// OidEntPhysicalModelName — .1.3.6.1.2.1.47.1.1.1.1.13 (DisplayString)
	// Vendor model / part number.
	OidEntPhysicalModelName = "1.3.6.1.2.1.47.1.1.1.1.13"
	// OidEntPhysicalClass — .1.3.6.1.2.1.47.1.1.1.1.5 (INTEGER enum)
	// 3=chassis 4=backplane 5=container(=module slot) 6=powerSupply
	// 7=fan 8=sensor 9=module 10=port 11=stack 12=processor
	// (Vendors historically disagree on which int maps to "power" —
	// see RESEARCH Pitfall 1 / Open Question 5.)
	OidEntPhysicalClass = "1.3.6.1.2.1.47.1.1.1.1.5"
	// OidEntPhysicalContainedIn — .1.3.6.1.2.1.47.1.1.1.1.4 (Integer32)
	// entPhysicalIndex of the parent entity; 0 means "no parent".
	OidEntPhysicalContainedIn = "1.3.6.1.2.1.47.1.1.1.1.4"
	// OidEntPhysicalName — .1.3.6.1.2.1.47.1.1.1.1.7 (DisplayString)
	// Vendor-assigned name (e.g. "power0", "temprature1", "FAN 1").
	OidEntPhysicalName = "1.3.6.1.2.1.47.1.1.1.1.7"
)

// Aliases retained in lower-case form for internal usage within the device
// package. External callers should use the exported constants above.
const (
	oidEntPhysicalSerialNum   = OidEntPhysicalSerialNum
	oidEntPhysicalModelName   = OidEntPhysicalModelName
	oidEntPhysicalClass       = OidEntPhysicalClass
	oidEntPhysicalContainedIn = OidEntPhysicalContainedIn
	oidEntPhysicalName        = OidEntPhysicalName
)

// EntityAttrs holds the five ENTITY-MIB attributes retrieved per index by
// GetEntityAttrs. Class and ContainedIn are decoded from int64 to int for
// convenience; Serial / Model / Name keep their DisplayString values.
type EntityAttrs struct {
	Serial      string
	Model       string
	Name        string
	Class       int
	ContainedIn int
}

// SNMPGetter is the minimal subset of *SNMPClient consumed by the
// entity-MIB helpers. Defined as an interface so unit tests can inject
// a stub without instantiating a live gosnmp connection.
type SNMPGetter interface {
	Get(oid string) (interface{}, error)
	Walk(oid string, callback func(oid string, value interface{}) bool) error
}

// CountPhysicalEntitiesByClass performs one Walk on oidEntPhysicalClass
// and returns the set of entity indices whose class value is in classes.
//
// The Walk is bounded by the device's actual entity table size (typically
// <=700 rows even on Ruijie RS8607E with full sensor tree). Indices are
// extracted from the OID suffix ".<idx>" appended by the agent.
//
// Unknown / non-integer class values are silently skipped.
func CountPhysicalEntitiesByClass(client SNMPGetter, classes []int) (map[int]struct{}, error) {
	want := make(map[int]struct{}, len(classes))
	for _, c := range classes {
		want[c] = struct{}{}
	}
	out := make(map[int]struct{})
	err := client.Walk(OidEntPhysicalClass, func(oid string, val interface{}) bool {
		idx := extractIndexFromOID(oid, oidEntPhysicalClass)
		if idx <= 0 {
			return true
		}
		class, ok := toInt(val)
		if !ok {
			return true
		}
		if _, hit := want[class]; hit {
			out[idx] = struct{}{}
		}
		return true
	})
	if err != nil {
		return nil, fmt.Errorf("walk entPhysicalClass: %w", err)
	}
	return out, nil
}

// GetEntityAttrs issues five single GETs to retrieve the per-entity
// attributes for index idx. Errors on individual OIDs are tolerated
// (the agent may legitimately return noSuchObject for a sparse column);
// the corresponding field stays zero-valued. A nil error is returned
// unless the SNMP transport itself fails.
func GetEntityAttrs(client SNMPGetter, idx int) (EntityAttrs, error) {
	var a EntityAttrs
	if idx <= 0 {
		return a, fmt.Errorf("invalid entity index %d", idx)
	}
	if v, err := client.Get(fmt.Sprintf("%s.%d", OidEntPhysicalSerialNum, idx)); err == nil {
		a.Serial = strings.TrimSpace(toString(v))
	}
	if v, err := client.Get(fmt.Sprintf("%s.%d", OidEntPhysicalModelName, idx)); err == nil {
		a.Model = strings.TrimSpace(toString(v))
	}
	if v, err := client.Get(fmt.Sprintf("%s.%d", OidEntPhysicalName, idx)); err == nil {
		a.Name = strings.TrimSpace(toString(v))
	}
	if v, err := client.Get(fmt.Sprintf("%s.%d", OidEntPhysicalClass, idx)); err == nil {
		if c, ok := toInt(v); ok {
			a.Class = c
		}
	}
	if v, err := client.Get(fmt.Sprintf("%s.%d", OidEntPhysicalContainedIn, idx)); err == nil {
		if c, ok := toInt(v); ok {
			a.ContainedIn = c
		}
	}
	return a, nil
}

// extractIndexFromOID parses the integer suffix appended to base by the
// SNMP agent. Returns 0 when the suffix is absent or non-numeric.
//
// Example: extractIndexFromOID(".1.3.6.1.2.1.47.1.1.1.1.5.42", base)
// returns 42.
func extractIndexFromOID(oid, base string) int {
	// Trim leading dot (gosnmp returns OIDs prefixed with ".").
	trimmed := strings.TrimPrefix(oid, ".")
	prefix := strings.TrimPrefix(base, ".")
	if !strings.HasPrefix(trimmed, prefix+".") {
		return 0
	}
	suffix := strings.TrimPrefix(trimmed, prefix+".")
	// Sub-OID may itself contain dots for compound keys; ENTITY-MIB
	// uses a single integer key so we only need the first segment.
	if dot := strings.IndexByte(suffix, '.'); dot >= 0 {
		suffix = suffix[:dot]
	}
	n, err := strconv.Atoi(suffix)
	if err != nil {
		return 0
	}
	return n
}

// toString coerces SNMP return values to DisplayString. Tolerates the
// gosnmp convention of returning []byte for OctetString.
func toString(v interface{}) string {
	switch x := v.(type) {
	case string:
		return x
	case []byte:
		return string(x)
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", x)
	}
}

// toInt coerces SNMP return values to int. gosnmp delivers INTEGER as
// int64; some legacy agents / proxies may emit int or uint64.
func toInt(v interface{}) (int, bool) {
	switch x := v.(type) {
	case int:
		return x, true
	case int32:
		return int(x), true
	case int64:
		return int(x), true
	case uint64:
		return int(x), true
	case string:
		n, err := strconv.Atoi(x)
		return n, err == nil
	default:
		return 0, false
	}
}
