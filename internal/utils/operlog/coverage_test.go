package operlog

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// stubRecorder captures the arguments passed to RecordAsync so tests can
// assert what operlog.Record / RecordWithBody would have persisted.
type stubRecorder struct {
	title        string
	businessType int
	operParam    *string
}

func (s *stubRecorder) RecordAsync(_ *gorm.DB, title string, businessType int, _, _,
	_ string, _, _, _, _ *string, operParam, _, _ *string, _ int, _ int64) {
	s.title = title
	s.businessType = businessType
	s.operParam = operParam
}

// TestFilterSensitiveParamsCoversAllKeywords verifies every entry in
// sensitiveKeys is masked by FilterSensitiveParams. Phase 34 raised the
// sensitive-key set to 18 entries (17 keywords — the original 5 + 13 added
// in Plan 34-01).
func TestFilterSensitiveParamsCoversAllKeywords(t *testing.T) {
	if len(sensitiveKeys) < 17 {
		t.Fatalf("expected >=17 sensitiveKeys, got %d", len(sensitiveKeys))
	}
	for _, kw := range sensitiveKeys {
		// Build a body whose value is unique per keyword so we can detect
		// both (a) that masking happened AND (b) the plaintext is gone.
		plaintext := "PLAIN_" + kw + "_VALUE"
		body := `{"` + kw + `":"` + plaintext + `","other":"keep"}`
		out := FilterSensitiveParams(body)
		if strings.Contains(out, plaintext) {
			t.Errorf("keyword %q NOT masked: input=%q output=%q (plaintext %q leaked)", kw, body, out, plaintext)
		}
		if !strings.Contains(out, "******") {
			t.Errorf("keyword %q: expected ****** in output, got %q", kw, out)
		}
		if !strings.Contains(out, `"other":"keep"`) {
			t.Errorf("keyword %q: non-sensitive field mangled: %q", kw, out)
		}
	}
}

// TestFilterSensitiveParamsMultipleOccurrences ensures duplicate keys are all
// masked (the legacy implementation only masked the first).
func TestFilterSensitiveParamsMultipleOccurrences(t *testing.T) {
	in := `{"password":"a","password":"b","password":"c"}`
	out := FilterSensitiveParams(in)
	if strings.Contains(out, "\"a\"") || strings.Contains(out, "\"b\"") || strings.Contains(out, "\"c\"") {
		t.Errorf("duplicate keys not fully masked: %q", out)
	}
	if got := strings.Count(out, "******"); got != 3 {
		t.Errorf("expected 3 masks, got %d in %q", got, out)
	}
}

// TestFilterSensitiveParamsCaseInsensitive verifies matching is
// case-insensitive (Password, PASSWORD, password all masked).
func TestFilterSensitiveParamsCaseInsensitive(t *testing.T) {
	cases := []string{"password", "Password", "PASSWORD", "PaSsWoRd"}
	for _, c := range cases {
		in := `{"` + c + `":"secret"}`
		out := FilterSensitiveParams(in)
		if strings.Contains(out, "secret") {
			t.Errorf("case variant %q not masked: %q", c, out)
		}
	}
}

// TestFilterSensitiveParamsEmpty ensures the empty-input fast path returns
// "" without panicking.
func TestFilterSensitiveParamsEmpty(t *testing.T) {
	if got := FilterSensitiveParams(""); got != "" {
		t.Errorf("expected empty output for empty input, got %q", got)
	}
}

// TestRecordWithBodyMasksAndRestores verifies the smoke-test for
// RecordWithBody: it must (1) read the body, (2) restore it so downstream
// handlers can still bind, and (3) record a masked oper_param.
func TestRecordWithBodyMasksAndRestores(t *testing.T) {
	gin.SetMode(gin.TestMode)

	body := `{"username":"alice","password":"hunter2","oldPassword":"pw1"}`
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(body))

	stub := &stubRecorder{}
	// Non-nil zero-value DB satisfies RecordWithBody's internal nil-check.
	db := &gorm.DB{}
	RecordWithBody(c, stub, db, "用户管理", OperTypeReset)

	// (3) oper_param must be masked and plaintext must be gone.
	if stub.operParam == nil {
		t.Fatal("operParam not recorded")
	}
	if strings.Contains(*stub.operParam, "hunter2") || strings.Contains(*stub.operParam, "pw1") {
		t.Errorf("plaintext password leaked into oper_param: %q", *stub.operParam)
	}
	if !strings.Contains(*stub.operParam, "******") {
		t.Errorf("expected ****** in oper_param, got %q", *stub.operParam)
	}
	if got := strings.Count(*stub.operParam, "******"); got < 2 {
		t.Errorf("expected >=2 masks (password + oldPassword), got %d in %q", got, *stub.operParam)
	}
	// oper_param must still carry the non-sensitive field.
	if !strings.Contains(*stub.operParam, "alice") {
		t.Errorf("non-sensitive username dropped from oper_param: %q", *stub.operParam)
	}

	// (1)+(2) The request body must be restored and re-readable.
	if c.Request.Body == nil {
		t.Fatal("request body not restored")
	}
	restored, err := io.ReadAll(c.Request.Body)
	if err != nil {
		t.Fatalf("restored body unreadable: %v", err)
	}
	if !bytes.Equal(restored, []byte(body)) {
		t.Errorf("restored body differs from original:\n want=%q\n got=%q", body, string(restored))
	}

	// Module + operType propagation.
	if stub.title != "用户管理" {
		t.Errorf("title got=%q want=用户管理", stub.title)
	}
	if stub.businessType != OperTypeReset {
		t.Errorf("businessType got=%d want=%d", stub.businessType, OperTypeReset)
	}
}

// TestRecordNilContextDoesNotPanic verifies the panic-safety guard: nil
// context / recorder / db must be a no-op, never a panic.
func TestRecordNilContextDoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Record panicked on nil inputs: %v", r)
		}
	}()
	Record(nil, nil, nil, "x", OperTypeOther)
	RecordWithBody(nil, nil, nil, "x", OperTypeOther)
}

// TestRecordWithOperParamOption verifies the WithOperParam functional option
// overrides the recorded oper_param.
func TestRecordWithOperParamOption(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/x", nil)

	stub := &stubRecorder{}
	Record(c, stub, &gorm.DB{}, "角色管理", OperTypeGrant, WithOperParam("custom-param"))
	if stub.operParam == nil || *stub.operParam != "custom-param" {
		t.Errorf("WithOperParam not applied: got=%v", stub.operParam)
	}
	if stub.businessType != OperTypeGrant {
		t.Errorf("businessType got=%d want=%d", stub.businessType, OperTypeGrant)
	}
}

// TestRecordSampleModuleOperTypeCombos is a sampling test that verifies
// operlog.Record is callable for representative (module, operType) combos
// drawn from the 30 e2e sample endpoints. This is a compile-time + call-path
// smoke test — it passes a zero-value non-nil *gorm.DB so Record's nil-guard
// does not short-circuit, but no actual DB write happens because the stub
// recorder ignores its db argument.
func TestRecordSampleModuleOperTypeCombos(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/x", nil)
	// Non-nil zero-value DB satisfies Record's nil-check without opening a
	// connection. The stub recorder never uses it.
	db := &gorm.DB{}

	type combo struct {
		module string
		op     int
	}
	samples := []combo{
		{"用户管理", OperTypeCreate},
		{"用户管理", OperTypeReset},
		{"用户管理", OperTypeBatch},
		{"用户管理", OperTypeStatus},
		{"角色管理", OperTypeGrant},
		{"部门管理", OperTypeCreate},
		{"菜单管理", OperTypeCreate},
		{"字典管理", OperTypeCreate},
		{"通知管理", OperTypeCreate},
		{"API密钥", OperTypeCreate},
		{"参数配置", OperTypeCreate},
		{"楼宇管理", OperTypeCreate},
		{"工位管理", OperTypeCreate},
		{"网络设备", OperTypeCreate},
		{"设备凭据", OperTypeCreate},
		{"工单管理", OperTypeCreate},
		{"工单管理", OperTypeApprove},
		{"工单管理", OperTypeReject},
		{"虚拟机管理", OperTypeStatus},
		{"定时任务", OperTypeOther},
		{"缓存监控", OperTypeClean},
		{"RPA任务", OperTypeCreate},
		{"RPA凭据", OperTypeCreate},
		{"Agent注册", OperTypeRegister},
		{"仪表盘管理", OperTypeCreate},
		{"列设置", OperTypeUpdate},
	}

	for _, s := range samples {
		stub := &stubRecorder{}
		Record(c, stub, db, s.module, s.op)
		if stub.title != s.module {
			t.Errorf("combo %+v: title got=%q", s, stub.title)
		}
		if stub.businessType != s.op {
			t.Errorf("combo %+v: businessType got=%d", s, stub.businessType)
		}
	}
}
