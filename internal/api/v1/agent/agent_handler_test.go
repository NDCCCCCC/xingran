package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// =====================================================================
// Phase 74-04: agent handler tests
//
// Scope:
//   - NewAgentHandler / WithCore (constructor + DI)
//   - RegisterAgent (POST /api/agent/register)
//   - matchVMByFingerprint (4 priority branches: IP/MAC/Hostname/MachineGUID + fallback)
//   - generateRandomString
//   - SetupAgentRouter (source-grep route-table lock)
//
// Per D-12: zero business code changes. operlog.RecordWithBody is exercised
// against a stub recorder so coverage of the call site is real but no audit
// row is actually persisted.
// =====================================================================

// stubRecorder satisfies both services.OperLogService (3 methods) and
// operlog.Recorder (single RecordAsync method) so agent_handler.RegisterAgent's
// RecordWithBody call lands here without panicking on nil OperLogService.
type stubRecorder struct {
	called        int
	lastTitle     string
	lastBusType   int
	lastOperParam *string
}

func (s *stubRecorder) RecordOperLog(_ context.Context, _ *gorm.DB, _ *models.OperLog) error {
	s.called++
	return nil
}

func (s *stubRecorder) RecordFromGinContext(_ *gin.Context, _ *gorm.DB, title string, businessType int, _ string) {
	s.called++
	s.lastTitle = title
	s.lastBusType = businessType
}

func (s *stubRecorder) RecordAsync(_ *gorm.DB, title string, businessType int, _, _, _ string,
	_, _, _ *string, _ *string, operParam, _, _ *string, _ int, _ int64) {
	s.called++
	s.lastTitle = title
	s.lastBusType = businessType
	s.lastOperParam = operParam
}

// invokeAgentHandler builds a gin context and invokes the supplied handler method.
func invokeAgentHandler(t *testing.T, method, path string, body interface{}, params gin.Params,
	handler func(*gin.Context)) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	var buf *bytes.Buffer
	if body != nil {
		switch v := body.(type) {
		case string:
			buf = bytes.NewBufferString(v)
		default:
			b, _ := json.Marshal(body)
			buf = bytes.NewBuffer(b)
		}
	} else {
		buf = bytes.NewBuffer(nil)
	}
	c.Request = httptest.NewRequest(method, path, buf)
	c.Request.Header.Set("Content-Type", "application/json")
	if params != nil {
		c.Params = params
	}
	if handler != nil {
		handler(c)
	}
	return w
}

// setupAgentDB creates a SQLite in-memory DB with the sys_vdi_vm table.
func setupAgentDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, db.Exec(`
		CREATE TABLE sys_vdi_vm (
			id TEXT PRIMARY KEY,
			vm_id TEXT UNIQUE NOT NULL,
			name TEXT NOT NULL,
			resource_id TEXT,
			ip_address TEXT,
			mac_address TEXT,
			os_type TEXT,
			power_state TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)
	`).Error)
	return db
}

// seedVM inserts a single VM row and returns its generated vm_id.
func seedVM(t *testing.T, db *gorm.DB, opts map[string]interface{}) string {
	t.Helper()
	vmID := uuid.NewString()
	id := uuid.NewString()
	name, _ := opts["name"].(string)
	if name == "" {
		name = "vm-" + vmID[:8]
	}
	row := map[string]interface{}{
		"id":          id,
		"vm_id":       vmID,
		"name":        name,
		"resource_id": opts["resource_id"],
		"ip_address":  opts["ip_address"],
		"mac_address": opts["mac_address"],
		"os_type":     opts["os_type"],
	}
	require.NoError(t, db.Table("sys_vdi_vm").Create(row).Error)
	return vmID
}

// newAgentHandlerWithStub builds an AgentHandler whose core.OperLogService is a stub
// so RegisterAgent's operlog.RecordWithBody call lands in our recorder.
func newAgentHandlerWithStub(t *testing.T, db *gorm.DB) (*AgentHandler, *stubRecorder) {
	t.Helper()
	h := NewAgentHandler(db)
	stub := &stubRecorder{}
	h.core = &core.Core{
		CoreInfra:    &core.CoreInfra{},
		CoreServices: &core.CoreServices{OperLogService: stub},
	}
	return h, stub
}

// ----------------------------------------------------------------------------
// Constructor + DI
// ----------------------------------------------------------------------------

func TestNewAgentHandler_DefaultsDBOnly(t *testing.T) {
	db := setupAgentDB(t)
	h := NewAgentHandler(db)
	require.NotNil(t, h)
	assert.Equal(t, db, h.db)
	assert.Nil(t, h.core, "core should be nil until WithCore is called")
}

func TestAgentHandler_WithCore_ReturnsSameHandler(t *testing.T) {
	db := setupAgentDB(t)
	h := NewAgentHandler(db)
	got := h.WithCore(&core.Core{})
	assert.Same(t, h, got, "WithCore should return the same receiver for chaining")
	assert.NotNil(t, h.core)
}

// ----------------------------------------------------------------------------
// RegisterAgent — happy paths via matchVMByFingerprint
// ----------------------------------------------------------------------------

func TestRegisterAgent_MatchByIPAddress(t *testing.T) {
	db := setupAgentDB(t)
	vmID := seedVM(t, db, map[string]interface{}{
		"name":        "win10-eng-01",
		"ip_address":  "10.20.30.40",
		"mac_address": "aa:bb:cc:dd:ee:01",
	})
	h, _ := newAgentHandlerWithStub(t, db)

	body := AgentRegisterRequest{
		Hostname:   "win10-eng-01",
		IPAddress:  "10.20.30.40",
		MACAddress: "aa:bb:cc:dd:ee:01",
		OSType:     "windows",
	}
	w := invokeAgentHandler(t, "POST", "/api/agent/register", body, nil, h.RegisterAgent)
	require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

	var resp struct {
		Code int                   `json:"code"`
		Data AgentRegisterResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, vmID, resp.Data.VMID)
	assert.True(t, resp.Data.Matched)
	assert.NotEmpty(t, resp.Data.AgentID)
	assert.Contains(t, resp.Data.AgentID, "win10-eng-01")
}

func TestRegisterAgent_MatchByMACAddress(t *testing.T) {
	db := setupAgentDB(t)
	vmID := seedVM(t, db, map[string]interface{}{
		"name":        "ubuntu-prod",
		"mac_address": "00:11:22:33:44:55",
	})
	h, _ := newAgentHandlerWithStub(t, db)

	body := AgentRegisterRequest{
		Hostname:   "ubuntu-prod",
		MACAddress: "00:11:22:33:44:55",
		OSType:     "linux",
	}
	w := invokeAgentHandler(t, "POST", "/api/agent/register", body, nil, h.RegisterAgent)
	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Data AgentRegisterResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, vmID, resp.Data.VMID)
	assert.True(t, resp.Data.Matched)
}

func TestRegisterAgent_MatchByHostname(t *testing.T) {
	db := setupAgentDB(t)
	vmID := seedVM(t, db, map[string]interface{}{
		"name": "host-with-unique-prefix-7890",
	})
	h, _ := newAgentHandlerWithStub(t, db)

	body := AgentRegisterRequest{
		Hostname: "unique-prefix-7890",
		OSType:   "linux",
	}
	w := invokeAgentHandler(t, "POST", "/api/agent/register", body, nil, h.RegisterAgent)
	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Data AgentRegisterResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, vmID, resp.Data.VMID)
	assert.True(t, resp.Data.Matched)
}

func TestRegisterAgent_MatchByMachineGUID(t *testing.T) {
	db := setupAgentDB(t)
	vmID := seedVM(t, db, map[string]interface{}{
		"name":        "win-template",
		"resource_id": "GUID-{ABCDEF12-3456-7890-ABCD-EF1234567890}",
	})
	h, _ := newAgentHandlerWithStub(t, db)

	body := AgentRegisterRequest{
		Hostname:    "win-template",
		OSType:      "windows",
		MachineGUID: "GUID-{ABCDEF12-3456-7890-ABCD-EF1234567890}",
	}
	w := invokeAgentHandler(t, "POST", "/api/agent/register", body, nil, h.RegisterAgent)
	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Data AgentRegisterResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, vmID, resp.Data.VMID)
	assert.True(t, resp.Data.Matched)
}

func TestRegisterAgent_NoMatch_ReturnsTempVMID(t *testing.T) {
	db := setupAgentDB(t)
	h, _ := newAgentHandlerWithStub(t, db)

	body := AgentRegisterRequest{
		Hostname: "unknown-host-xyz",
		OSType:   "linux",
	}
	w := invokeAgentHandler(t, "POST", "/api/agent/register", body, nil, h.RegisterAgent)
	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Data AgentRegisterResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.False(t, resp.Data.Matched, "no VM row should match")
	assert.Contains(t, resp.Data.VMID, "vm-auto-")
	assert.Contains(t, resp.Data.VMID, "unknown-host-xyz")
	assert.NotEmpty(t, resp.Data.AgentID)
}

// ----------------------------------------------------------------------------
// RegisterAgent — binding error
// ----------------------------------------------------------------------------

func TestRegisterAgent_MissingHostname_Returns400(t *testing.T) {
	db := setupAgentDB(t)
	h, _ := newAgentHandlerWithStub(t, db)

	// Hostname is required (binding:"required") → bind error → 400.
	body := map[string]interface{}{
		"ip_address": "10.0.0.1",
	}
	w := invokeAgentHandler(t, "POST", "/api/agent/register", body, nil, h.RegisterAgent)
	require.NotEqual(t, http.StatusOK, w.Code)
}

func TestRegisterAgent_BadJSON_Returns400(t *testing.T) {
	db := setupAgentDB(t)
	h, _ := newAgentHandlerWithStub(t, db)

	w := invokeAgentHandler(t, "POST", "/api/agent/register", "not-json", nil, h.RegisterAgent)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ----------------------------------------------------------------------------
// matchVMByFingerprint priority test — IP should win over MAC/Hostname/MachineGUID
// ----------------------------------------------------------------------------

func TestRegisterAgent_IPPriorityBeatsMACAndHostname(t *testing.T) {
	db := setupAgentDB(t)
	ipMatched := seedVM(t, db, map[string]interface{}{
		"name":        "by-ip",
		"ip_address":  "10.10.10.10",
		"mac_address": "ff:ff:ff:ff:ff:01",
	})
	seedVM(t, db, map[string]interface{}{
		"name":        "shared-hostname",
		"ip_address":  "10.10.10.99",
		"mac_address": "ff:ff:ff:ff:ff:99",
	})
	h, _ := newAgentHandlerWithStub(t, db)

	body := AgentRegisterRequest{
		Hostname:   "shared-hostname",
		IPAddress:  "10.10.10.10",
		MACAddress: "ff:ff:ff:ff:ff:01",
		OSType:     "linux",
	}
	w := invokeAgentHandler(t, "POST", "/api/agent/register", body, nil, h.RegisterAgent)
	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Data AgentRegisterResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, ipMatched, resp.Data.VMID, "IP should win over hostname")
}

func TestRegisterAgent_MACPriorityBeatsHostname(t *testing.T) {
	db := setupAgentDB(t)
	macMatched := seedVM(t, db, map[string]interface{}{
		"name":        "by-mac",
		"mac_address": "aa:00:bb:00:cc:00",
	})
	h, _ := newAgentHandlerWithStub(t, db)

	body := AgentRegisterRequest{
		Hostname:   "by-mac",
		MACAddress: "aa:00:bb:00:cc:00",
		OSType:     "linux",
	}
	w := invokeAgentHandler(t, "POST", "/api/agent/register", body, nil, h.RegisterAgent)
	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Data AgentRegisterResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, macMatched, resp.Data.VMID, "MAC should win over hostname")
}

func TestRegisterAgent_HostnamePriorityBeatsMachineGUID(t *testing.T) {
	db := setupAgentDB(t)
	hostMatched := seedVM(t, db, map[string]interface{}{
		"name":        "by-hostname",
		"resource_id": "GUID-AAAA-BBBB",
	})
	seedVM(t, db, map[string]interface{}{
		"name":        "different-name",
		"resource_id": "GUID-CCCC-DDDD",
	})
	h, _ := newAgentHandlerWithStub(t, db)

	body := AgentRegisterRequest{
		Hostname:    "by-hostname",
		OSType:      "linux",
		MachineGUID: "GUID-CCCC-DDDD",
	}
	w := invokeAgentHandler(t, "POST", "/api/agent/register", body, nil, h.RegisterAgent)
	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Data AgentRegisterResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, hostMatched, resp.Data.VMID, "hostname should win over machine GUID")
}

// ----------------------------------------------------------------------------
// generateRandomString
// ----------------------------------------------------------------------------

func TestGenerateRandomString_Length(t *testing.T) {
	for _, n := range []int{0, 1, 8, 16, 64} {
		got := generateRandomString(n)
		assert.Equal(t, n, len(got), "len=%d got=%d", n, len(got))
	}
}

func TestGenerateRandomString_OnlyCharset(t *testing.T) {
	got := generateRandomString(128)
	for _, ch := range got {
		ok := (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9')
		assert.True(t, ok, "char %q not in charset", ch)
	}
}

func TestGenerateRandomString_StablePattern(t *testing.T) {
	// The implementation uses charset[i%len(charset)] which is deterministic.
	got := generateRandomString(8)
	assert.Equal(t, "abcdefgh", got, "expected deterministic output from charset prefix")
}

// ----------------------------------------------------------------------------
// SetupAgentRouter source-grep lock (per VALIDATION §4.5 grep-style accepted)
// ----------------------------------------------------------------------------

func TestSetupAgentRouter_RouteShape(t *testing.T) {
	// Use an absolute path so the test does not depend on the caller's cwd.
	src, err := os.ReadFile("D:/CODE/ClaudeCode/guoguo/internal/api/v1/agent/agent_router.go")
	if err != nil {
		// Fallback to relative path (works when test runs from project root).
		src, err = os.ReadFile("internal/api/v1/agent/agent_router.go")
	}
	require.NoError(t, err)
	source := string(src)
	assert.Contains(t, source, `r.Group("/agent")`)
	assert.Contains(t, source, `agentGroup.POST("/register", handler.RegisterAgent)`)
	assert.Contains(t, source, `func SetupAgentRouter(r *gin.RouterGroup, core *core.Core)`)
}

// Confirm models.VDIVirtualMachine is referenced so the package import is live
// (the field is used inside the seeded CREATE TABLE schema above).
var _ = models.VDIVirtualMachine{}
