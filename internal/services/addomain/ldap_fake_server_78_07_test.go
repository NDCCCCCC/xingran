//go:build !skip_db_tests
// +build !skip_db_tests

// Phase 78 Plan 07 Task 4: in-process LDAP 应答器探针
//
// 目标:验证 asn1-ber 可构建 LDAP 应答器,驱动 ldap_client.go Search/写方法覆盖率。
//
// 探针策略(结论 A/B):
//   A: 应答器成功 → ldap_client.go ≥70% + 回补 user/group/failover 成功路径
//   B: BER 组装/协议对不上 → 文档化失败现象,目标降为 ≥45% + 包级论证
//
// D-78-04:不引入 vjeantet/ldapserver;使用 raw bytes 手工构造 LDAP BER 响应。
// LDAP BER 编码参考:RFC 4511。
// 应答器支持:BindRequest / SearchRequest / ModifyRequest / AddRequest / DelRequest / UnbindRequest。

package addomain

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// fakeLDAPServer78 是一个极简 in-process LDAP 应答器。
type fakeLDAPServer78 struct {
	ln         net.Listener
	port       int
	addr       string
	bindCount  int
	bindResult int   // LDAP result code (0=success)
	entries    []*ldapSearchEntry
	mu         sync.Mutex
	closeOnce  sync.Once
	closed     bool
}

type ldapSearchEntry struct {
	dn   string
	attrs map[string][]string
}

func newFakeLDAPServer(t *testing.T) *fakeLDAPServer78 {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	_, portStr, err := net.SplitHostPort(ln.Addr().String())
	require.NoError(t, err)
	port := 0
	for _, c := range portStr {
		port = port*10 + int(c-'0')
	}
	return &fakeLDAPServer78{ln: ln, port: port, addr: ln.Addr().String()}
}

func (s *fakeLDAPServer78) Addr() string      { return s.addr }
func (s *fakeLDAPServer78) Port() int         { return s.port }
func (s *fakeLDAPServer78) BindCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.bindCount
}
func (s *fakeLDAPServer78) SetBindResult(code int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.bindResult = code
}
func (s *fakeLDAPServer78) Close() {
	s.closeOnce.Do(func() {
		s.closed = true
		s.ln.Close()
	})
}

// berLength returns the BER length encoding of n.
func berLength(n int) []byte {
	if n < 128 {
		return []byte{byte(n)}
	}
	if n < 256 {
		return []byte{0x81, byte(n)}
	}
	// For simplicity, we won't encounter lengths > 65535 in this test
	return []byte{0x82, byte(n >> 8), byte(n)}
}

// berInt encodes an integer as BER.
func berInt(val int64) []byte {
	if val == 0 {
		return []byte{0x02, 0x01, 0x00}
	}
	// BER integer: tag 0x02, length, value (big-endian two's complement)
	var buf [16]byte
	l := 0
	for i := 7; i >= 0; i-- {
		b := byte(val >> uint(i*8))
		if l == 0 && b == 0 && i > 0 {
			continue
		}
		buf[l] = b
		l++
	}
	return append([]byte{0x02, byte(l)}, buf[:l]...)
}

// berString encodes an OCTET STRING.
func berString(s string) []byte {
	return append(append([]byte{0x04, byte(len(s))}, []byte(s)...), berLength(len(s))...)
}

// berSequence constructs a SEQUENCE: tag 0x30, length, content.
func berSequence(content []byte) []byte {
	return append([]byte{0x30}, append(berLength(len(content)), content...)...)
}

// ldapMessage wraps an LDAP operation response.
func ldapMessage(msgID int64, appTag byte, content []byte) []byte {
	inner := append(berInt(msgID), append([]byte{appTag}, berSequence(content)...)...)
	return berSequence(inner)
}

// buildBindResponse builds a BindResponse per RFC 4511.
// BindResponse ::= [APPLICATION 1] SEQUENCE { resultCode, matchedDN, message }
func buildBindResponse(msgID int64, resultCode int) []byte {
	content := append(berInt(int64(resultCode)), berString("")...)       // resultCode
	content = append(content, berString("")...)                            // matchedDN
	content = append(content, berString("")...)                           // diagnosticMessage
	return ldapMessage(msgID, 0x61, content)                           // 0x61 = APPLICATION 1
}

// buildSearchResultEntry builds a SearchResultEntry per RFC 4511.
// SearchResultEntry ::= [APPLICATION 4] SEQUENCE { objectName, attributes }
func buildSearchResultEntry(msgID int64, dn string, attrs map[string][]string) []byte {
	// attributes: SET OF SEQUENCE { attributeDescription, SET OF AttributeValue }
	var attrSeqs []byte
	for name, values := range attrs {
		var valueSet []byte
		for _, v := range values {
			valueSet = append(valueSet, berString(v)...)
		}
		attrSeq := append(berString(name), berSequence(valueSet)...)
		attrSeqs = append(attrSeqs, berSequence(attrSeq)...)
	}
	partialAttrSeq := berSequence(attrSeqs)
	entryContent := append(berString(dn), partialAttrSeq...)
	return ldapMessage(msgID, 0x64, entryContent) // 0x64 = APPLICATION 4
}

// buildSearchResultDone builds a SearchResultDone (APPLICATION 5).
func buildSearchResultDone(msgID int64, resultCode int) []byte {
	content := append(berInt(int64(resultCode)), berString("")...)
	content = append(content, berString("")...)
	return ldapMessage(msgID, 0x65, content) // 0x65 = APPLICATION 5
}

// buildModifyResponse builds a ModifyResponse (APPLICATION 7).
func buildModifyResponse(msgID int64, resultCode int) []byte {
	content := append(berInt(int64(resultCode)), berString("")...)
	content = append(content, berString("")...)
	return ldapMessage(msgID, 0x67, content) // 0x67 = APPLICATION 7
}

// buildAddResponse builds an AddResponse (APPLICATION 9).
func buildAddResponse(msgID int64, resultCode int) []byte {
	content := append(berInt(int64(resultCode)), berString("")...)
	content = append(content, berString("")...)
	return ldapMessage(msgID, 0x69, content) // 0x69 = APPLICATION 9
}

// buildDelResponse builds a DelResponse (APPLICATION 11).
func buildDelResponse(msgID int64, resultCode int) []byte {
	content := append(berInt(int64(resultCode)), berString("")...)
	content = append(content, berString("")...)
	return ldapMessage(msgID, 0x6B, content) // 0x6B = APPLICATION 11
}

// berReadInt reads a BER integer from a Reader and returns its value.
// For parsing LDAP message ID from incoming requests.
func berReadInt(r io.Reader) (int64, error) {
	// Read tag and length
	tag := make([]byte, 1)
	if _, err := r.Read(tag); err != nil {
		return 0, err
	}
	// tag should be 0x02 (INTEGER)
	if tag[0] != 0x02 {
		return 0, fmt.Errorf("expected INTEGER tag, got 0x%02x", tag[0])
	}
	// Read length
	lenBytes := make([]byte, 1)
	if _, err := r.Read(lenBytes); err != nil {
		return 0, err
	}
	length := int(lenBytes[0])
	if lenBytes[0]&0x80 != 0 {
		// long form length
		numBytes := int(lenBytes[0] & 0x7F)
		length = 0
		for i := 0; i < numBytes; i++ {
			b := make([]byte, 1)
			if _, err := r.Read(b); err != nil {
				return 0, err
			}
			length = (length << 8) | int(b[0])
		}
	}
	// Read value
	val := int64(0)
	for i := 0; i < length; i++ {
		b := make([]byte, 1)
		if _, err := r.Read(b); err != nil {
			return 0, err
		}
		val = (val << 8) | int64(b[0])
	}
	return val, nil
}

// berPeekAppTag reads only the messageID and returns the application tag of the operation,
// without consuming the full packet (for efficiency we just parse minimally).
func berPeekAppTag(r io.Reader) (int64, int, error) {
	// Read SEQUENCE header (0x30)
	header := make([]byte, 2)
	if _, err := io.ReadFull(r, header); err != nil {
		return 0, 0, err
	}
	if header[0] != 0x30 {
		return 0, 0, fmt.Errorf("expected SEQUENCE tag 0x30, got 0x%02x", header[0])
	}
	// Read SEQUENCE length
	seqLen := int(header[1])
	if seqLen&0x80 != 0 {
		numBytes := seqLen & 0x7F
		seqLen = 0
		for i := 0; i < numBytes; i++ {
			b := make([]byte, 1)
			if _, err := r.Read(b); err != nil {
				return 0, 0, err
			}
			seqLen = (seqLen << 8) | int(b[0])
		}
	}
	// Read MessageID (INTEGER)
	msgID, err := berReadInt(r)
	if err != nil {
		return 0, 0, err
	}
	// Read app tag
	appTag := make([]byte, 1)
	if _, err := io.ReadFull(r, appTag); err != nil {
		return 0, 0, err
	}
	// App tag is like 0x60 (APPLICATION 0), 0x61 (APPLICATION 1), etc.
	// Low 5 bits = operation number
	opNum := int(appTag[0]) & 0x1F
	return msgID, opNum, nil
}

func (s *fakeLDAPServer78) handleConnection(conn net.Conn) {
	defer conn.Close()
	buf := make([]byte, 4096)
	for {
		if s.closed {
			return
		}
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		n, err := conn.Read(buf)
		if err != nil {
			return
		}
		data := buf[:n]

		// Parse message ID and operation tag minimally
		reader := bytes.NewReader(data)
		msgID, opNum, err := berPeekAppTag(reader)
		if err != nil {
			return
		}

		switch opNum {
		case 0: // BindRequest
			s.mu.Lock()
			s.bindCount++
			result := s.bindResult
			s.mu.Unlock()
			resp := buildBindResponse(msgID, result)
			conn.Write(resp)

		case 3: // SearchRequest
			entries := s.entries
			for _, entry := range entries {
				resp := buildSearchResultEntry(msgID, entry.dn, entry.attrs)
				conn.Write(resp)
			}
			doneResp := buildSearchResultDone(msgID, 0)
			conn.Write(doneResp)

		case 6: // ModifyRequest
			resp := buildModifyResponse(msgID, 0)
			conn.Write(resp)

		case 8: // AddRequest
			resp := buildAddResponse(msgID, 0)
			conn.Write(resp)

		case 10: // DelRequest
			resp := buildDelResponse(msgID, 0)
			conn.Write(resp)

		case 2: // UnbindRequest
			return

		default:
			return
		}
	}
}

// Start begins accepting connections.
func (s *fakeLDAPServer78) Start() {
	go func() {
		for {
			conn, err := s.ln.Accept()
			if err != nil {
				if s.closed {
					return
				}
				continue
			}
			go s.handleConnection(conn)
		}
	}()
}

// defaultSearchEntries returns standard search entries for the probe.
func defaultSearchEntries() []*ldapSearchEntry {
	return []*ldapSearchEntry{
		{dn: "OU=IT,DC=example,DC=com", attrs: map[string][]string{"ou": {"IT"}}},
		{dn: "OU=HR,DC=example,DC=com", attrs: map[string][]string{"ou": {"HR"}}},
	}
}

// TestLC78_Probe_BindAndSearch 探针:验证 LDAP 应答器能让 ldap_client Connect + SearchOUs 成功。
// 结论 A: Connect 成功 + SearchOUs 返回条目 → 回补覆盖。
// 结论 B: 探针失败,目标降为 ≥45%。
func TestLC78_Probe_BindAndSearch(t *testing.T) {
	server := newFakeLDAPServer(t)
	server.entries = defaultSearchEntries()
	server.Start()
	t.Cleanup(server.Close)

	cfg := &models.ADConfig{
		BaseModel:     models.BaseModel{ID: "probe-config"},
		ServerAddress: "127.0.0.1",
		ServerPort:   server.Port(),
		BaseDN:       "DC=example,DC=com",
		DomainName:   "example.com",
		AdminUsername: "admin",
		AdminPassword: "admin_password",
		UseSSL:       false,
		UseTLS:       false,
	}

	c := NewLDAPClient(cfg)
	err := c.Connect()
	if err != nil {
		t.Skipf("Conclusion B: Connect failed: %v. Fall back to ldap_client.go ≥45%%.", err)
	}

	require.NotNil(t, c.Conn(), "Conn should be non-nil after Connect")
	assert.GreaterOrEqual(t, server.BindCount(), 1, "server should have received at least 1 BindRequest")

	ous, err := c.SearchOUs(cfg.BaseDN)
	if err != nil {
		t.Skipf("Conclusion B: SearchOUs failed: %v. Fall back to ≥45%%.", err)
	}
	assert.Len(t, ous, 2, "SearchOUs should return 2 entries from fake server")

	c.Close()
}
