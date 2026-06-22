package operlog

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func newTestContext(method, path string, body []byte) *gin.Context {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}
	c.Request = httptest.NewRequest(method, path, bodyReader)
	if body != nil {
		c.Request.Header.Set("Content-Type", "application/json")
	}
	return c
}

func TestFilterSensitiveParams(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "password masked",
			in:   `{"username":"alice","password":"hunter2"}`,
			want: `{"username":"alice","password":"******"}`,
		},
		{
			name: "PASSWORD case-insensitive",
			in:   `{"PASSWORD":"secret"}`,
			want: `{"PASSWORD":"******"}`,
		},
		{
			name: "token masked",
			in:   `{"token":"abc.def.ghi"}`,
			want: `{"token":"******"}`,
		},
		{
			name: "oldPassword replaced, newPassword untouched",
			in:   `{"oldPassword":"pw1","newPassword":"pw2"}`,
			want: `{"oldPassword":"******","newPassword":"pw2"}`,
		},
		{
			name: "privateKey masked",
			in:   `{"privateKey":"-----BEGIN-----"}`,
			want: `{"privateKey":"******"}`,
		},
		{
			name: "publicKey masked",
			in:   `{"publicKey":"MFkwEwYHKoZI"}`,
			want: `{"publicKey":"******"}`,
		},
		{
			name: "sm4Key and sm2Key masked",
			in:   `{"sm4Key":"k1","sm2Key":"k2"}`,
			want: `{"sm4Key":"******","sm2Key":"******"}`,
		},
		{
			// Phase 34 review CRITICAL: apiKey MUST be masked. Prior to the
			// sensitiveKeys expansion in this same commit, "key" alone did
			// NOT match "apiKey" — substring `"key":"` does not appear inside
			// `"apiKey":"supersecret"` (positions 4-10 are `ey":"su`, not
			// `"key":"`). This test pins the new "apiKey" entry.
			name: "apiKey masked (Phase 34 review critical)",
			in:   `{"apiKey":"supersecret"}`,
			want: `{"apiKey":"******"}`,
		},
		{
			name: "adminPassword masked",
			in:   `{"adminPassword":"root"}`,
			want: `{"adminPassword":"******"}`,
		},
		{
			name: "duplicate keys both masked",
			in:   `{"password":"a","password":"b"}`,
			want: `{"password":"******","password":"******"}`,
		},
		{
			name: "no sensitive key returns unchanged",
			in:   `{"username":"alice","role":"admin"}`,
			want: `{"username":"alice","role":"admin"}`,
		},
		{
			name: "macKey (camelCase prefix) masked",
			in:   `{"macKey":"k1"}`,
			want: `{"macKey":"******"}`,
		},
		{
			name: "secretKey (camelCase prefix) masked",
			in:   `{"secretKey":"sk-1234"}`,
			want: `{"secretKey":"******"}`,
		},
		{
			name: "empty string returns empty",
			in:   ``,
			want: ``,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := FilterSensitiveParams(tc.in)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestFilterSensitiveParams_LargeInput(t *testing.T) {
	// Build a body > 8192 bytes that still contains a sensitive keyword.
	// FilterSensitiveParams must truncate to 8192 bytes and return the
	// truncated prefix (after masking), not return the input unchanged.
	filler := strings.Repeat("x", 9000)
	in := `{"password":"hunter2","data":"` + filler + `"}`
	got := FilterSensitiveParams(in)

	// Result must be <= 8192 bytes (truncation applied).
	assert.LessOrEqual(t, len(got), maxFilteredBytes, "large input must be truncated to <= %d bytes", maxFilteredBytes)
	// The truncated prefix still contains the password key BEFORE the filler,
	// so it must have been masked.
	assert.Contains(t, got, "******")
	assert.NotContains(t, got, "hunter2", "sensitive value must not survive truncation+masking")
}

func TestRecord_NoOpOnNilCore(t *testing.T) {
	c := newTestContext(http.MethodPost, "/test", nil)
	// Must not panic when operLogSvc and db are nil (Record returns early).
	assert.NotPanics(t, func() {
		Record(c, nil, nil, "test", OperTypeCreate)
	})
}

// noopDB is a non-nil *gorm.DB used so Record's nil-guard does not short-
// circuit. It is never actually queried because GetDeptNameFromDB returns nil
// early when the context has no user_id (which all these tests ensure).
func noopDB() *gorm.DB { return &gorm.DB{} }

func TestRecord_WithOperParam(t *testing.T) {
	// Use a stub operLogSvc to capture the oper_param passed to RecordAsync.
	stub := &stubOperLogSvc{}
	c := newTestContext(http.MethodPost, "/test", nil)
	// No user_id in context -> GetDeptNameFromDB returns nil without touching db.

	param := `{"foo":"bar"}`
	Record(c, stub, noopDB(), "test", OperTypeCreate, WithOperParam(param))
	// The stub ignores db, so this validates the option flows through.
	assert.NotNil(t, stub.lastOperParam)
	assert.Equal(t, param, *stub.lastOperParam)
}

func TestRecord_WithStatus(t *testing.T) {
	stub := &stubOperLogSvc{}
	c := newTestContext(http.MethodPost, "/test", nil)

	Record(c, stub, noopDB(), "test", OperTypeCreate, WithStatus(1), WithErrorMsg("boom"))
	assert.Equal(t, 1, stub.lastStatus)
	assert.NotNil(t, stub.lastErrorMsg)
	assert.Equal(t, "boom", *stub.lastErrorMsg)
}

func TestRecordWithBody_RestoresBody(t *testing.T) {
	original := []byte(`{"password":"hunter2","username":"alice"}`)
	c := newTestContext(http.MethodPost, "/test", original)

	stub := &stubOperLogSvc{}
	RecordWithBody(c, stub, noopDB(), "test", OperTypeCreate)

	// After the call, c.Request.Body must still be readable and contain the
	// ORIGINAL payload (proves io.NopCloser(bytes.NewBuffer(raw)) restored it).
	restored, err := io.ReadAll(c.Request.Body)
	assert.NoError(t, err)
	assert.Equal(t, string(original), string(restored))
}

func TestRecordWithBody_MasksPassword(t *testing.T) {
	original := []byte(`{"password":"hunter2","username":"alice"}`)
	c := newTestContext(http.MethodPost, "/test", original)

	stub := &stubOperLogSvc{}
	RecordWithBody(c, stub, noopDB(), "test", OperTypeCreate)

	// The recorded oper_param must contain the masked password, not hunter2.
	assert.NotNil(t, stub.lastOperParam, "RecordWithBody must record an oper_param")
	assert.Contains(t, *stub.lastOperParam, "******")
	assert.NotContains(t, *stub.lastOperParam, "hunter2")
}

func TestOperTypeConstants(t *testing.T) {
	// Guard against accidental constant-value drift (threat T-34-03).
	assert.Equal(t, 0, OperTypeOther)
	assert.Equal(t, 1, OperTypeCreate)
	assert.Equal(t, 2, OperTypeUpdate)
	assert.Equal(t, 3, OperTypeDelete)
	assert.Equal(t, 4, OperTypeGrant)
	assert.Equal(t, 5, OperTypeExport)
	assert.Equal(t, 6, OperTypeImport)
	assert.Equal(t, 7, OperTypeForce)
	assert.Equal(t, 8, OperTypeGenCode)
	assert.Equal(t, 9, OperTypeClean)
	// New constants.
	assert.Equal(t, 10, OperTypeStatus)
	assert.Equal(t, 11, OperTypeReset)
	assert.Equal(t, 12, OperTypeEnable)
	assert.Equal(t, 13, OperTypeDisable)
	assert.Equal(t, 14, OperTypeSync)
	assert.Equal(t, 15, OperTypeMove)
	assert.Equal(t, 16, OperTypeBatch)
	assert.Equal(t, 17, OperTypeUpload)
	assert.Equal(t, 18, OperTypeDownload)
	assert.Equal(t, 19, OperTypeLogin)
	assert.Equal(t, 20, OperTypeLogout)
	assert.Equal(t, 21, OperTypeRegister)
	assert.Equal(t, 22, OperTypeApprove)
	assert.Equal(t, 23, OperTypeReject)
}

// stubOperLogSvc is a no-op Recorder used to capture the arguments passed to
// RecordAsync without touching the database. Only the fields exercised by the
// tests are recorded.
type stubOperLogSvc struct {
	lastOperParam *string
	lastStatus    int
	lastErrorMsg  *string
}

// Compile-time assertion that stubOperLogSvc satisfies the local Recorder
// interface (which services.OperLogService also satisfies structurally).
var _ Recorder = (*stubOperLogSvc)(nil)

func (s *stubOperLogSvc) RecordAsync(_ *gorm.DB, _ string, _ int, _ string, _ string, _ string,
	_ *string, _ *string, _ *string, _ *string, operParam, _ *string, errorMsg *string, status int, _ int64) {
	s.lastOperParam = operParam
	s.lastStatus = status
	s.lastErrorMsg = errorMsg
}

// TestIsExcludedPath（Phase 35 OPERLOG-02 + OPERLOG-05）
// 表驱动测试覆盖 filepath.Match + /* 后缀通配的所有关键语义:
// exact match / /* single-segment wildcard / no-match / /list 误伤防护 /
// 空列表 / 多模式 OR 语义 / 大小写敏感 / /* 不跨 /。
func TestIsExcludedPath(t *testing.T) {
	cases := []struct {
		name     string
		patterns []string
		path     string
		want     bool
	}{
		{
			name:     "exact match",
			patterns: []string{"/api/v1/rpa/workers/register"},
			path:     "/api/v1/rpa/workers/register",
			want:     true,
		},
		{
			name:     "/* single-segment wildcard",
			patterns: []string{"/api/v1/rpa/workers/*/heartbeat"},
			path:     "/api/v1/rpa/workers/w-001/heartbeat",
			want:     true,
		},
		{
			name:     "/* does not cross /",
			patterns: []string{"/api/v1/rpa/workers/*/heartbeat"},
			path:     "/api/v1/rpa/workers/list",
			want:     false,
		},
		{
			name:     "/* matches any single worker id",
			patterns: []string{"/api/v1/rpa/workers/*/heartbeat"},
			path:     "/api/v1/rpa/workers/abc-def-123/heartbeat",
			want:     true,
		},
		{
			name:     "no match returns false",
			patterns: []string{"/api/v1/rpa/workers/*/heartbeat"},
			path:     "/api/v1/system/user/list",
			want:     false,
		},
		{
			name:     "empty list returns false",
			patterns: []string{},
			path:     "/anything/at/all",
			want:     false,
		},
		{
			name:     "/list mis-fire protection",
			patterns: []string{"/api/v1/rpa/workers/*/heartbeat"},
			path:     "/system/user/list",
			want:     false,
		},
		{
			name:     "multiple patterns OR-semantics",
			patterns: []string{"/a", "/b/*", "/c/*/d"},
			path:     "/c/x/d",
			want:     true,
		},
		{
			name:     "non-match with prefix only",
			patterns: []string{"/api/v1/rpa/workers/*/heartbeat"},
			path:     "/api/v1/rpa/workers",
			want:     false,
		},
		{
			name:     "trailing-slash only pattern",
			patterns: []string{"/api/v1/upload/*"},
			path:     "/api/v1/upload/",
			want:     true,
		},
		{
			name:     "trailing-slash with subpath",
			patterns: []string{"/api/v1/upload/*"},
			path:     "/api/v1/upload/foo/bar.jpeg",
			want:     true,
		},
		{
			name:     "case-sensitive match",
			patterns: []string{"/API/v1/rpa/workers/*/heartbeat"},
			path:     "/api/v1/rpa/workers/w-001/heartbeat",
			want:     false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			Configure(tc.patterns) // arrange
			defer Configure(nil)   // cleanup so other tests start clean
			got := IsExcludedPath(tc.path)
			assert.Equal(t, tc.want, got, "patterns=%v path=%q", tc.patterns, tc.path)
		})
	}
}
