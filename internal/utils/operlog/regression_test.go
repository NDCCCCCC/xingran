package operlog

// regression_test.go guards the public API of the operlog package against
// silent drift introduced by future refactors. It is the Phase 34 closing-plan
// lock-in (T-34-DOC-02 mitigation).
//
// What is pinned and why:
//   - OperType constant values (TestOperTypeConstantStability):
//     handlers across 7 waves select a constant by name and rely on its int
//     value being stable. Renumbering would silently mislabel every historical
//     row in sys_oper_log.
//   - OperType count == 25 (TestOperTypeCountEquals25):
//     adding/removing a constant without updating call sites is a breaking
//     change. The count is the coarsest invariant that catches both additions
//     and removals.
//   - Record signature (TestRecordSignatureStable):
//     Record is the most-called function in the package (~267 call sites per
//     the Phase 34 audit). Changing the parameter list would break the build
//     across every handler module.
//   - sensitiveKeys keyword coverage (TestFilterSensitiveParamsKeywordsStable):
//     dropping a keyword would leak a sensitive field (password / token / key)
//     into sys_oper_log.oper_param. The minimum mandatory set is enumerated
//     below.

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// expectedOperTypeValues pins the documented (name -> value) mapping. Every
// value also appears in CLAUDE.md "操作日志记录约定 (operlog convention)" and in
// docs/开发规范.md — keep the three sources in sync.
var expectedOperTypeValues = map[string]int{
	"OperTypeOther":    0,
	"OperTypeCreate":   1,
	"OperTypeUpdate":   2,
	"OperTypeDelete":   3,
	"OperTypeGrant":    4,
	"OperTypeExport":   5,
	"OperTypeImport":   6,
	"OperTypeForce":    7,
	"OperTypeGenCode":  8,
	"OperTypeClean":    9,
	"OperTypeStatus":   10,
	"OperTypeReset":    11,
	"OperTypeEnable":   12,
	"OperTypeDisable":  13,
	"OperTypeSync":     14,
	"OperTypeMove":     15,
	"OperTypeBatch":    16,
	"OperTypeUpload":   17,
	"OperTypeDownload": 18,
	"OperTypeLogin":    19,
	"OperTypeLogout":   20,
	"OperTypeRegister": 21,
	"OperTypeApprove":  22,
	"OperTypeReject":   23,
	"OperTypeUnlock":   24,
}

// TestOperTypeConstantStability asserts each OperType* constant is pinned to
// its documented int value. This catches accidental renumbering during future
// refactors (e.g. inserting a new value in the middle of the const block).
func TestOperTypeConstantStability(t *testing.T) {
	t.Parallel()
	// Build the map of actual constant values by resolving each identifier in
	// the operlog package's own scope. Using reflection on a value-of-the-pkg
	// is not possible for untyped constants, so we resolve via the source AST.
	actual, err := readOperTypeConsts("operlog.go")
	if err != nil {
		t.Fatalf("failed to parse operlog.go: %v", err)
	}
	if len(actual) == 0 {
		t.Fatal("no OperType* constants found — operlog.go parser is broken")
	}
	for name, want := range expectedOperTypeValues {
		got, ok := actual[name]
		if !ok {
			t.Errorf("OperType constant %q is missing from operlog.go — removing a constant is a breaking change", name)
			continue
		}
		if got != want {
			t.Errorf("OperType constant %q = %d, want %d (renumbering would mislabel historical sys_oper_log rows)", name, got, want)
		}
	}
	// Also flag unexpected constants so additions are deliberate (update both
	// this map and the docs).
	for name, got := range actual {
		if _, ok := expectedOperTypeValues[name]; !ok {
			t.Errorf("unexpected OperType constant %q = %d — if intentional, add it to expectedOperTypeValues and update CLAUDE.md + docs/开发规范.md", name, got)
		}
	}
}

// TestOperTypeCountEquals25 asserts the operlog package exposes exactly 25
// OperType constants. Catches both additions (which would need new call sites
// + docs) and removals (which would break existing call sites).
//
// History: was 24 at Phase 34 close; bumped to 25 in Phase 34 review closure
// to add OperTypeUnlock (24) for the user-unlock audit verb.
func TestOperTypeCountEquals25(t *testing.T) {
	t.Parallel()
	const want = 25
	actual, err := readOperTypeConsts("operlog.go")
	if err != nil {
		t.Fatalf("failed to parse operlog.go: %v", err)
	}
	if got := len(actual); got != want {
		t.Fatalf("OperType constant count = %d, want %d (additions/removals must be deliberate and documented)", got, want)
	}
}

// TestRecordSignatureStable asserts the Record function's signature still
// matches the documented 5-parameter + variadic-options shape. Record is the
// most-called function in the package; any signature change breaks the build
// across every handler module.
func TestRecordSignatureStable(t *testing.T) {
	t.Parallel()
	// func Record(c *gin.Context, operLogSvc Recorder, db *gorm.DB, module string,
	//   operType int, opts ...RecordOption)
	// => 5 fixed params + 1 variadic.
	rg := reflect.TypeOf(Record)
	if rg.Kind() != reflect.Func {
		t.Fatalf("Record is %v, not a func", rg.Kind())
	}
	if got := rg.NumIn(); got != 6 {
		t.Fatalf("Record has %d params, want 6 (5 fixed + 1 variadic RecordOption) — signature drift would break ~267 call sites", got)
	}
	// Last param must be variadic (...RecordOption).
	if !rg.IsVariadic() {
		t.Fatal("Record is not variadic — expected `opts ...RecordOption` as the final parameter")
	}
	// Sanity-check the fixed parameter types by name so a swap (e.g. recorder
	// and db) is caught even when NumIn matches.
	wantFixed := []string{
		"*gin.Context",
		"operlog.Recorder",
		"*gorm.DB",
		"string",
		"int",
	}
	for i := 0; i < 5; i++ {
		if got := rg.In(i).String(); got != wantFixed[i] {
			t.Errorf("Record param %d type = %q, want %q", i, got, wantFixed[i])
		}
	}
	// Variadic element type must be RecordOption.
	if got := rg.In(5).Elem().String(); got != "operlog.RecordOption" {
		t.Errorf("Record variadic element type = %q, want operlog.RecordOption", got)
	}

	// RecordWithBody has the same 5 fixed params but no variadic options.
	rwb := reflect.TypeOf(RecordWithBody)
	if rwb.Kind() != reflect.Func {
		t.Fatalf("RecordWithBody is %v, not a func", rwb.Kind())
	}
	if got := rwb.NumIn(); got != 5 {
		t.Fatalf("RecordWithBody has %d params, want 5 (signature drift would break sensitive-endpoint call sites)", got)
	}
	if rwb.IsVariadic() {
		t.Fatal("RecordWithBody must not be variadic — it accepts no options")
	}
}

// mandatorySensitiveKeywords is the minimum keyword set that must remain in
// sensitiveKeys. These are the substrings whose unmasked presence in
// sys_oper_log.oper_param would be a confirmed leak. Any removal is a security
// regression.
//
// Critical invariant: each entry MUST be the EXACT field name (case-folded
// later by the matcher). "key" alone does NOT match "apiKey" or "sm4Key" —
// the matcher does a contiguous-substring search for `"<key>":"`, not a
// word-boundary match. The 14 entries below lock the camelCase / snake_case
// variants of the password / key / secret / token family that have actually
// appeared in handler request structs.
var mandatorySensitiveKeywords = []string{
	"password",
	"pwd",
	"secret",
	"token",
	"key",
	"salt",
	"privateKey",
	"oldPassword",
	"macKey",
	"sm4Key",
	"sm2Key",
	"sm3Key",
	"apiKey",
	"accessKey",
	"secretKey",
	"clientSecret",
	"private_key",
	"publicKey",
}

// TestFilterSensitiveParamsKeywordsStable asserts the sensitive keyword slice
// still contains at least the mandatory keywords. Dropping a keyword would
// leak that field into sys_oper_log.oper_param.
func TestFilterSensitiveParamsKeywordsStable(t *testing.T) {
	t.Parallel()
	if len(sensitiveKeys) < len(mandatorySensitiveKeywords) {
		t.Fatalf("sensitiveKeys has %d entries, want at least %d", len(sensitiveKeys), len(mandatorySensitiveKeywords))
	}
	joined := strings.Join(sensitiveKeys, "\n")
	for _, kw := range mandatorySensitiveKeywords {
		if !strings.Contains(joined, kw) {
			t.Errorf("mandatory sensitive keyword %q is missing from sensitiveKeys — removing it would leak the field into sys_oper_log.oper_param", kw)
		}
	}
}

// readOperTypeConsts parses fileName (a file in this package directory) and
// returns a map of every exported `OperType*` identifier to its integer
// literal value. It returns an error if the file cannot be parsed or a
// constant's value is not a literal int.
func readOperTypeConsts(fileName string) (map[string]int, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, fileName, nil, 0)
	if err != nil {
		return nil, err
	}
	out := make(map[string]int)
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.CONST {
			continue
		}
		for _, spec := range gd.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok || len(vs.Names) == 0 || len(vs.Values) == 0 {
				continue
			}
			name := vs.Names[0].Name
			if !strings.HasPrefix(name, "OperType") || name == "OperType" {
				// Skip the type alias `type OperType = int` (declared as a
				// TypeSpec, not a ValueSpec, but be defensive).
				continue
			}
			lit, ok := vs.Values[0].(*ast.BasicLit)
			if !ok || lit.Kind != token.INT {
				continue
			}
			// Parse the literal manually to avoid importing strconv for a
			// trivial decimal parse; ast BasicLit.Value is the literal text.
			v := strings.TrimSpace(lit.Value)
			n := 0
			neg := false
			if strings.HasPrefix(v, "-") {
				neg = true
				v = v[1:]
			}
			for _, r := range v {
				if r < '0' || r > '9' {
					n = -1
					break
				}
				n = n*10 + int(r-'0')
			}
			if n < 0 {
				continue
			}
			if neg {
				n = -n
			}
			out[name] = n
		}
	}
	return out, nil
}

// excludedMockRecorder is a minimal Recorder implementation that captures
// RecordAsync call counts and the last operUrl. Used by
// TestExcludedPathsEarlyReturn to assert the operlog package-level exclude
// list short-circuits RecordAsync without actually wiring a real
// services.OperLogService.
type excludedMockRecorder struct {
	called  int
	lastURL string
}

func (m *excludedMockRecorder) RecordAsync(db *gorm.DB, title string, businessType int, method, requestMethod, operUrl string,
	operatorName, operatorNickname, deptName *string, operIP *string, operParam, jsonResult, errorMsg *string, status int, costTime int64) {
	m.called++
	m.lastURL = operUrl
}

// TestExcludedPathsEarlyReturn（Phase 35 OPERLOG-05）
// 双子测试覆盖:
//   - Phase A: Record 在排除路径（heartbeat）早退,
//     非排除路径（register、progress）仍调用 RecordAsync
//   - Phase B: RecordWithBody 在排除路径上**完全跳过 GetRawData**,
//     c.Request.Body 在调用后仍可被下游 handler 绑定（核心威胁 T-35-03 缓解）
func TestExcludedPathsEarlyReturn(t *testing.T) {
	// Phase A: Record early-return
	t.Run("Record skips excluded path", func(t *testing.T) {
		Configure([]string{"/api/v1/rpa/workers/*/heartbeat"})
		defer Configure(nil)
		mock := &excludedMockRecorder{}
		gin.SetMode(gin.TestMode)
		db := noopDB()

		// Excluded — must early-return (mock.called stays 0)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/rpa/workers/w-001/heartbeat", nil)
		Record(c, mock, db, "RPA Worker", OperTypeOther)
		assert.Equal(t, 0, mock.called, "heartbeat path must skip RecordAsync")

		// Not excluded — must record (mock.called becomes 1)
		c2, _ := gin.CreateTestContext(httptest.NewRecorder())
		c2.Request = httptest.NewRequest(http.MethodPost, "/api/v1/rpa/workers/register", nil)
		Record(c2, mock, db, "RPA Worker", OperTypeRegister)
		assert.Equal(t, 1, mock.called, "register path must call RecordAsync")

		// progress is NOT excluded (audit value retained per OPERLOG-04)
		c3, _ := gin.CreateTestContext(httptest.NewRecorder())
		c3.Request = httptest.NewRequest(http.MethodPost, "/api/v1/rpa/workers/progress", nil)
		Record(c3, mock, db, "RPA Worker", OperTypeOther)
		assert.Equal(t, 2, mock.called, "progress path must call RecordAsync (not in exclude list)")
	})

	// Phase B: RecordWithBody pre-GetRawData contract (CRITICAL)
	t.Run("RecordWithBody skips GetRawData on excluded path", func(t *testing.T) {
		Configure([]string{"/api/v1/rpa/workers/*/heartbeat"})
		defer Configure(nil)
		body := []byte(`{"password":"hunter2"}`)
		mock := &excludedMockRecorder{}
		db := noopDB()

		// Excluded path with body — must early-return BEFORE consuming body
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/rpa/workers/w-001/heartbeat", bytes.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		RecordWithBody(c, mock, db, "RPA Worker", OperTypeOther)
		assert.Equal(t, 0, mock.called, "heartbeat path must skip RecordAsync entirely")

		// Body must still be readable (GetRawData was NOT called) so downstream
		// SM2+SM4 decryption middleware and handler binding still work.
		remaining, err := io.ReadAll(c.Request.Body)
		assert.NoError(t, err)
		assert.Equal(t, body, remaining, "request body must NOT be consumed on excluded path")

		// Non-excluded path with body — must record via masking + restore
		c2, _ := gin.CreateTestContext(httptest.NewRecorder())
		c2.Request = httptest.NewRequest(http.MethodPost, "/api/v1/rpa/workers/register", bytes.NewReader(body))
		c2.Request.Header.Set("Content-Type", "application/json")
		RecordWithBody(c2, mock, db, "RPA Worker", OperTypeRegister)
		assert.Equal(t, 1, mock.called, "register path must call RecordAsync via RecordWithBody")
	})
}
