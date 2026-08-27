// Phase 78-04 (BLOCK-04) — fake UDP SNMP server helper for snmp_client tests.
//
// D-78-06: Uses net.ListenUDP (127.0.0.1:0) + gosnmp's own public codecs
// (SnmpDecodePacket / SetRequestID / SnmpEncodePacket). No hand-written BER,
// no asn1-ber import, no net.PacketConn abstraction.
//
// The server runs in a goroutine attached to t.Context — t.Cleanup closes the
// UDP conn which causes ReadFrom to return an error and the goroutine exits
// cleanly. RequestCount is atomic for retry-verification assertions.
package device

import (
	"net"
	"sync/atomic"
	"testing"

	"github.com/gosnmp/gosnmp"
)

// behavior models the three injectable server-side behaviors.
type behavior int

const (
	behaviorNormal     behavior = iota // respond with the matching table entry
	behaviorErrorStatus                // respond with Error != NoError
	behaviorDrop                       // silently swallow the request (triggers client timeout/retry)
)

// fakeSNMPServer78 is a process-local UDP SNMP responder.
//
// Table lookup is OID-prefix-based: for GETNEXT we walk forward through the
// sorted table and return the first entry whose OID > requested OID. All
// network targets are strictly 127.0.0.1.
type fakeSNMPServer78 struct {
	pc       *net.UDPConn
	table    []gosnmp.SnmpPDU        // ordered OID->value entries for Walk/GetNext
	reqCount int64                   // atomic; number of requests received
	behavior behavior                // injectable behavior
}

// newFakeSNMPServer78 creates a listening UDP server backed by table.
// The table is sorted by OID (ascending) to support GetNext semantics.
// Caller must call Close() to stop the server (t.Cleanup handles this).
func newFakeSNMPServer78(t *testing.T, table []gosnmp.SnmpPDU) *fakeSNMPServer78 {
	pc, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatalf("ListenUDP failed: %v", err)
	}

	f := &fakeSNMPServer78{pc: pc, table: table, behavior: behaviorNormal}

	go f.serve()

	t.Cleanup(func() {
		pc.Close() // causes ReadFrom to return; serve goroutine exits
	})

	return f
}

// serve runs in a goroutine. It exits when pc is closed.
func (f *fakeSNMPServer78) serve() {
	buf := make([]byte, 65535)
	for {
		n, addr, err := f.pc.ReadFrom(buf)
		if err != nil {
			return // pc closed — exit cleanly
		}
		atomic.AddInt64(&f.reqCount, 1)
		f.handlePacket(buf[:n], addr)
	}
}

// handlePacket decodes a request, looks up the response in the table, and sends it.
func (f *fakeSNMPServer78) handlePacket(raw []byte, addr net.Addr) {
	if f.behavior == behaviorDrop {
		return // swallow
	}

	decoder := &gosnmp.GoSNMP{Version: gosnmp.Version2c, Community: "public"}
	req, err := decoder.SnmpDecodePacket(raw)
	if err != nil {
		return
	}

	var respPDUs []gosnmp.SnmpPDU

	switch req.PDUType {
	case gosnmp.GetRequest:
		// GET: exact OID match
		for _, pdu := range f.table {
			if pdu.Name == req.Variables[0].Name {
				respPDUs = []gosnmp.SnmpPDU{{Name: pdu.Name, Type: pdu.Type, Value: pdu.Value}}
				break
			}
		}

	case gosnmp.GetNextRequest:
		// GETNEXT: first entry with OID > requested
		for _, pdu := range f.table {
			if pdu.Name > req.Variables[0].Name {
				respPDUs = []gosnmp.SnmpPDU{{Name: pdu.Name, Type: pdu.Type, Value: pdu.Value}}
				break
			}
		}

	case gosnmp.GetBulkRequest:
		// Return up to maxRepetitions entries starting from the OID
		maxRep := uint32(5)
		if req.MaxRepetitions > 0 {
			maxRep = req.MaxRepetitions
		}
		startIdx := -1
		for i, pdu := range f.table {
			if pdu.Name >= req.Variables[0].Name {
				startIdx = i
				break
			}
		}
		if startIdx >= 0 {
			end := startIdx + int(maxRep)
			if end > len(f.table) {
				end = len(f.table)
			}
			respPDUs = f.table[startIdx:end]
		}

	default:
		return
	}

	if len(respPDUs) == 0 {
		return // no matching entry — client will timeout
	}

	var out []byte
	if f.behavior == behaviorErrorStatus {
		// Inject an error status
		respPkt := &gosnmp.SnmpPacket{
			Version:    req.Version,
			Community:  req.Community,
			PDUType:    gosnmp.GetResponse,
			RequestID:  req.RequestID,
			Error:      gosnmp.NoSuchName, // != NoError
			ErrorIndex: 1,
			Variables:  respPDUs,
		}
		out, _ = respPkt.MarshalMsg()
	} else {
		// Build response packet directly to preserve req.RequestID
		// (SnmpEncodePacket generates a new RequestID via atomic counter)
		respPkt := &gosnmp.SnmpPacket{
			Version:    req.Version,
			Community:  req.Community,
			PDUType:    gosnmp.GetResponse,
			RequestID:  req.RequestID,
			Error:      gosnmp.NoError,
			ErrorIndex: 0,
			Variables:  respPDUs,
		}
		out, _ = respPkt.MarshalMsg()
	}

	f.pc.WriteTo(out, addr)
}

// Port returns the allocated UDP port.
func (f *fakeSNMPServer78) Port() uint16 {
	return uint16(f.pc.LocalAddr().(*net.UDPAddr).Port)
}

// Addr returns the server's UDP address string.
func (f *fakeSNMPServer78) Addr() string {
	return f.pc.LocalAddr().String()
}

// RequestCount returns the number of requests received (atomic).
func (f *fakeSNMPServer78) RequestCount() int64 {
	return atomic.LoadInt64(&f.reqCount)
}

// SetBehavior changes the injectable behavior.
func (f *fakeSNMPServer78) SetBehavior(b behavior) {
	f.behavior = b
}

// Close closes the UDP listener. Idempotent via t.Cleanup.
func (f *fakeSNMPServer78) Close() {
	f.pc.Close()
}
