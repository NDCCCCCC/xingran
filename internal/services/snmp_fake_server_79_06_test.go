// copied/trimmed from internal/device/snmp_fake_server_78_04_test.go (in-package
// test-only, not importable) — D-79-05.
//
// Phase 79-06 (TAIL-01) — process-local UDP SNMP responder for the
// internal/services root device family (snmpProbe / discoverBySNMP /
// pingDeviceViaSNMP). Same design as the 78-04 original: net.ListenUDP on
// 127.0.0.1:0 + gosnmp's own public codecs (SnmpDecodePacket / MarshalMsg), no
// hand-written BER, no asn1-ber import.
//
// Trimmed for 79-06: only GetRequest is answered — that is the only PDU the
// three call sites above issue (single-OID Gets on sysName/sysDescr). The
// GetNext/GetBulk walk branches of the 78-04 original are not ported.
//
// Lifecycle: the serve goroutine exits when t.Cleanup closes the UDP conn
// (ReadFrom returns). All targets are strictly 127.0.0.1.
package services

import (
	"net"
	"sync/atomic"
	"testing"

	"github.com/gosnmp/gosnmp"
)

// snmp7906Behavior models the three injectable server-side behaviors.
type snmp7906Behavior int

const (
	snmp7906Normal      snmp7906Behavior = iota // respond with the matching table entry
	snmp7906ErrorStatus                         // respond with Error != NoError (client sees an error, fast)
	snmp7906Drop                                // silently swallow the request (client timeout path)
)

// fakeSNMPServer7906 is a process-local UDP SNMP responder with OID-exact GET
// matching over an ordered table.
type fakeSNMPServer7906 struct {
	pc       *net.UDPConn
	table    []gosnmp.SnmpPDU
	reqCount int64 // atomic
	behavior snmp7906Behavior
}

// newFakeSNMPServer7906 starts the responder and binds its shutdown to
// t.Cleanup. The table is lookup-only for GET (order irrelevant), matching the
// trimmed scope.
func newFakeSNMPServer7906(t *testing.T, table []gosnmp.SnmpPDU) *fakeSNMPServer7906 {
	t.Helper()

	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("ResolveUDPAddr failed: %v", err)
	}
	pc, err := net.ListenUDP("udp", addr)
	if err != nil {
		t.Fatalf("ListenUDP failed: %v", err)
	}

	f := &fakeSNMPServer7906{pc: pc, table: table, behavior: snmp7906Normal}
	go f.serve()

	t.Cleanup(f.Close) // closes the conn → serve goroutine exits
	return f
}

// serve exits when pc is closed.
func (f *fakeSNMPServer7906) serve() {
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

// handlePacket answers GetRequest only (trimmed scope, see file header).
func (f *fakeSNMPServer7906) handlePacket(raw []byte, addr net.Addr) {
	if f.behavior == snmp7906Drop {
		return // swallow — drives the client timeout path
	}

	decoder := &gosnmp.GoSNMP{Version: gosnmp.Version2c, Community: "public"}
	req, err := decoder.SnmpDecodePacket(raw)
	if err != nil {
		return
	}
	if req.PDUType != gosnmp.GetRequest {
		return // trimmed: walk PDUs are not served by the 79-06 fake
	}

	var respPDUs []gosnmp.SnmpPDU
	for _, pdu := range f.table {
		if pdu.Name == req.Variables[0].Name {
			respPDUs = []gosnmp.SnmpPDU{{Name: pdu.Name, Type: pdu.Type, Value: pdu.Value}}
			break
		}
	}
	if len(respPDUs) == 0 {
		return // no matching entry — client will time out
	}

	// Build the response packet directly to preserve req.RequestID
	// (SnmpEncodePacket generates a fresh RequestID via an atomic counter).
	out, err := (&gosnmp.SnmpPacket{
		Version:    req.Version,
		Community:  req.Community,
		PDUType:    gosnmp.GetResponse,
		RequestID:  req.RequestID,
		Error:      gosnmp.NoError,
		ErrorIndex: 0,
		Variables:  respPDUs,
	}).MarshalMsg()
	if f.behavior == snmp7906ErrorStatus && err == nil {
		out, _ = (&gosnmp.SnmpPacket{
			Version:    req.Version,
			Community:  req.Community,
			PDUType:    gosnmp.GetResponse,
			RequestID:  req.RequestID,
			Error:      gosnmp.NoSuchName,
			ErrorIndex: 1,
			Variables:  respPDUs,
		}).MarshalMsg()
	}
	if err != nil {
		return
	}

	_, _ = f.pc.WriteTo(out, addr)
}

// Port returns the allocated UDP port.
func (f *fakeSNMPServer7906) Port() uint16 {
	return uint16(f.pc.LocalAddr().(*net.UDPAddr).Port)
}

// RequestCount returns the number of requests received (atomic).
func (f *fakeSNMPServer7906) RequestCount() int64 {
	return atomic.LoadInt64(&f.reqCount)
}

// Close closes the UDP listener (idempotent via t.Cleanup).
func (f *fakeSNMPServer7906) Close() {
	_ = f.pc.Close()
}

// SetBehavior changes the injectable behavior mid-test.
func (f *fakeSNMPServer7906) SetBehavior(b snmp7906Behavior) {
	f.behavior = b
}
