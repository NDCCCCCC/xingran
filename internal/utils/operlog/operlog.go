// Package operlog provides a shared operation-log recording helper that any
// api/v1/* handler module can call without importing internal/api/v1/system
// (which would create a circular dependency for non-system modules).
//
// Migrated from internal/api/v1/system/helper.go and
// internal/services/oper_log_service.go on 2026-06-15 as part of Phase 34
// (oper-log-full-coverage). The old `recordOperLog` and `FilterSensitiveParams`
// implementations are kept as thin shims in their original packages for backward
// compatibility — they delegate here.
//
// Bodies > 8192 bytes are not audited for sensitive-field masking. Endpoints
// that may receive such large payloads (file uploads, bulk imports) should use
// explicit operlog.WithOperParam(...) rather than relying on auto-masking.
package operlog

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/utils"
	"gorm.io/gorm"
)

// OperType is an alias for the underlying int column type so handlers can
// optionally reference a named type (operlog.OperType(operlog.OperTypeCreate)).
// Constants remain plain ints so they assign directly to OperLog.BusinessType.
type OperType = int

// Existing OperType constants (values 0-9 mirror the legacy system package).
const (
	OperTypeOther   = 0 // 其他
	OperTypeCreate  = 1 // 新增
	OperTypeUpdate  = 2 // 修改
	OperTypeDelete  = 3 // 删除
	OperTypeGrant   = 4 // 授权
	OperTypeExport  = 5 // 导出
	OperTypeImport  = 6 // 导入
	OperTypeForce   = 7 // 强退
	OperTypeGenCode = 8 // 生成代码
	OperTypeClean   = 9 // 清空数据
)

// New OperType constants added in Phase 34 (values 10-23). These cover
// state-change verbs, audit verbs, and bulk verbs so handlers stop misusing
// OperTypeOther (0) for legitimate semantic types.
const (
	OperTypeStatus   = 10 // 状态变更：启用/停用
	OperTypeReset    = 11 // 密码/密钥重置
	OperTypeEnable   = 12 // 启用
	OperTypeDisable  = 13 // 停用
	OperTypeSync     = 14 // LDAP/VDI/资产/外部同步
	OperTypeMove     = 15 // 移动 OU/部门/虚拟机
	OperTypeBatch    = 16 // 批量新增/删除
	OperTypeUpload   = 17 // 文件上传
	OperTypeDownload = 18 // 文件/模板下载
	OperTypeLogin    = 19 // 登录
	OperTypeLogout   = 20 // 登出
	OperTypeRegister = 21 // Agent/虚拟机注册
	OperTypeApprove  = 22 // 审批通过
	OperTypeReject   = 23 // 审批驳回
	OperTypeUnlock   = 24 // 账号解锁（管理员手动解锁被锁定的用户）
)

// maxFilteredBytes bounds the CPU cost of the sensitive-param filter. Inputs
// larger than this are truncated for scanning and logged with a warning.
const maxFilteredBytes = 8192

// Recorder is the minimal subset of services.OperLogService that Record and
// RecordWithBody actually call. Defining it locally (instead of importing
// internal/services) keeps operlog a leaf package with no internal deps beyond
// internal/utils, so internal/services/oper_log_service.go can import operlog
// for FilterSensitiveParams without forming an import cycle. The concrete
// services.OperLogService implementation satisfies this interface structurally.
type Recorder interface {
	RecordAsync(db *gorm.DB, title string, businessType int, method, requestMethod, operUrl string,
		operatorName, operatorNickname, deptName *string, operIP *string, operParam, jsonResult, errorMsg *string, status int, costTime int64)
}

// ExcludedPaths is the active exclude list (Phase 35 OPERLOG-01). Configure()
// replaces it at startup; IsExcludedPath reads it. Exported so debug
// introspection (e.g. pprof, /debug vars) can inspect the live list without
// poking private state. Not safe for concurrent mutation — single startup call.
var ExcludedPaths []string

// Configure replaces the package-level exclude list. Call exactly once at
// startup from core.New() before any handler may invoke Record or
// RecordWithBody. Subsequent calls overwrite the list (used by tests for
// cleanup via `defer Configure(nil)`).
func Configure(paths []string) {
	ExcludedPaths = paths
}

// IsExcludedPath returns true if path matches any pattern in ExcludedPaths
// using filepath.Match + /* suffix wildcard semantics (mirrors
// pkg/middleware/request_decryption.go:294-315 isExcludedPath so the two
// exclude lists share one mental model).
//
// filepath.Match's `*` is a single-segment wildcard that does NOT cross `/`,
// so "/api/v1/rpa/workers/*/heartbeat" matches any worker id but will NOT
// accidentally match "/api/v1/rpa/workers/list" or any path under a deeper
// segment.
func IsExcludedPath(path string) bool {
	for _, pattern := range ExcludedPaths {
		if pattern == path {
			return true
		}
		if matched, _ := filepath.Match(pattern, path); matched {
			return true
		}
		// /* suffix means "this prefix plus anything starting with / or empty".
		if prefix, ok := strings.CutSuffix(pattern, "/*"); ok {
			if rest := strings.TrimPrefix(path, prefix); rest != path && (rest == "" || strings.HasPrefix(rest, "/")) {
				return true
			}
		}
	}
	return false
}

// sensitiveKeys is the case-insensitive keyword set masked by
// FilterSensitiveParams. Order does not matter; matching is performed in a
// loop-with-resume so every occurrence per keyword is replaced.
//
// IMPORTANT: the matcher is a contiguous-substring search for `"<key>":"`,
// NOT a prefix-tolerant or word-boundary match. Each camelCase or snake_case
// variant of a sensitive concept MUST be added as its own literal entry
// below — relying on a base word (e.g. "key") to match "apiKey" or "sm4Key"
// would silently fail. See regression_test.go's `mandatorySensitiveKeywords`
// for the locked minimum set.
var sensitiveKeys = []string{
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
	"v1Key",
	"v2Key",
	"appKey",
	"appSecret",
	"hmacKey",
	"signKey",
	"aesKey",
	"desKey",
	"rsaPrivateKey",
	"rsaPublicKey",
	"certPassword",
	"keystorePassword",
	"truststorePassword",
	"adminPassword",
	"clientSecret",
	"accessKey",
	"accessKeyId",
	"accessKeySecret",
	"secretKey",
	"private_key",
	"publicKey",
}

// recordConfig holds optional fields that functional options can override.
// Zero values mean "use the RecordAsync default" (nil / 0).
type recordConfig struct {
	operParam  *string
	jsonResult *string
	errorMsg   string
	status     int
	costTime   int64
}

// RecordOption is a functional option for Record. Use WithOperParam,
// WithStatus, or WithErrorMsg to override the recorded row.
type RecordOption func(*recordConfig)

// WithOperParam overrides the recorded oper_param with the given (already
// filtered) string. Callers are responsible for masking — pass the output of
// FilterSensitiveParams if the param originated from a request body.
func WithOperParam(s string) RecordOption {
	return func(c *recordConfig) { c.operParam = &s }
}

// WithStatus sets the recorded status (1 = failure, 0 = success) — used for
// error-path logging where the handler caught an error but still wants the
// attempt recorded.
func WithStatus(status int) RecordOption {
	return func(c *recordConfig) { c.status = status }
}

// WithErrorMsg records an error message in the row. Implies WithStatus(1) is
// recommended but not enforced — callers that want failure status must also
// pass WithStatus(1).
func WithErrorMsg(msg string) RecordOption {
	return func(c *recordConfig) { c.errorMsg = msg }
}

// Record records an operation log entry. It is panic-safe: any failure inside
// the recording path is recovered so a logging bug never crashes the handler
// chain. Pass variadic options (WithOperParam / WithStatus / WithErrorMsg) to
// override the recorded fields; with no options, oper_param/json_result/error
// are left nil and status defaults to 0 (success).
//
// For sensitive endpoints that need to capture the request body, prefer
// RecordWithBody which reads+restores c.Request.Body for you.
func Record(c *gin.Context, operLogSvc Recorder, db *gorm.DB, module string, operType int, opts ...RecordOption) {
	// Panic-safety (threat T-34-01): a logging failure must never crash the
	// handler chain.
	defer func() {
		_ = recover()
	}()

	// Path-based early return (Phase 35 OPERLOG-01): configured high-frequency
	// endpoints (e.g. rpa-worker heartbeat at 30s/Worker) skip logging to
	// avoid drowning sys_oper_log in low-value audit rows.
	if c != nil && c.Request != nil && IsExcludedPath(c.Request.URL.Path) {
		return
	}

	if c == nil || operLogSvc == nil || db == nil {
		return
	}

	cfg := recordConfig{}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	clientIP := utils.GetClientIP(c)
	var errMsgPtr *string
	if cfg.errorMsg != "" {
		errMsgPtr = &cfg.errorMsg
	}

	operLogSvc.RecordAsync(
		db,
		module,
		operType,
		"",
		c.Request.Method,
		c.Request.URL.String(),
		utils.GetUsernamePtr(c),
		utils.GetNicknamePtrWithDB(c, db),
		utils.GetDeptNameFromDB(c, db),
		&clientIP,
		cfg.operParam,
		cfg.jsonResult,
		errMsgPtr,
		cfg.status,
		cfg.costTime,
	)
}

// RecordBackground records an operation log entry from a non-request context
// (cron workers, async pipelines, scheduled sync tasks). It is the cron-path
// counterpart of Record: instead of pulling operator/IP/URL from a
// *gin.Context, it takes an explicit operatorName (e.g. "system-cron") and
// fills the remaining fields with sentinel values that distinguish the row
// from interactive requests in sys_oper_log.
//
// Phase 48 D-13 uses this for component-asset UPDATEs emitted by the
// DeviceInfoCollectionService cron hook (no *gin.Context available).
//
// method="BACKGROUND" / requestMethod="CRON" / operUrl="" / operIP=nil /
// deptName=nil. operParam is JSON-encoded when non-nil. The call is
// panic-safe (a logging bug never crashes the caller).
//
// Note: D-13 scope is the UPDATE path only. Failure-path events from cron
// pipelines use applogger.Warnf (application log), NOT RecordBackground —
// the asymmetry avoids audit noise and keeps sys_oper_log semantic.
func RecordBackground(operLogSvc Recorder, db *gorm.DB, module string, operType int, operatorName string, params map[string]interface{}) {
	// Panic-safety (threat T-34-01): a logging failure must never crash the
	// caller.
	defer func() {
		_ = recover()
	}()

	if operLogSvc == nil || db == nil {
		return
	}

	var operParam *string
	if params != nil {
		if raw, err := json.Marshal(params); err == nil {
			s := string(raw)
			operParam = &s
		}
	}

	// operatorName doubles as both operator + nickname for cron rows.
	namePtr := operatorName

	operLogSvc.RecordAsync(
		db,
		module,
		operType,
		"BACKGROUND",
		"CRON",
		"",
		&namePtr,
		&namePtr,
		nil, // deptName — cron has no dept
		nil, // operIP — cron has no client IP
		operParam,
		nil, // jsonResult
		nil, // errorMsg
		0,   // status — success
		0,   // costTime
	)
}

// RecordWithBody is a body-aware variant of Record for sensitive endpoints
// (password reset, password change, API key creation, agent registration). It
// reads c.Request.Body once, restores it via io.NopCloser(bytes.NewBuffer(...))
// so downstream SM2+SM4 middleware and handler binding still work, then records
// the masked body as oper_param.
//
// Prefer RecordWithBody ONLY for sensitive write endpoints. For non-sensitive
// write endpoints prefer plain Record to avoid reading the body unnecessarily.
func RecordWithBody(c *gin.Context, operLogSvc Recorder, db *gorm.DB, module string, operType int) {
	// Panic-safety for the body-read path.
	defer func() {
		_ = recover()
	}()

	// Path-based early return (Phase 35 OPERLOG-01): MUST happen BEFORE
	// c.GetRawData() below, otherwise GetRawData would silently consume the
	// request body even though we end up not logging anything — which would
	// break downstream SM2+SM4 decryption middleware and handler body binding
	// for excluded endpoints.
	if c != nil && c.Request != nil && IsExcludedPath(c.Request.URL.Path) {
		return
	}

	if c == nil || c.Request == nil {
		Record(c, operLogSvc, db, module, operType)
		return
	}

	raw, err := c.GetRawData()
	if err != nil {
		// Fall back to plain Record; do not block the handler chain.
		Record(c, operLogSvc, db, module, operType)
		return
	}

	// Restore the body so downstream handlers (and the SM2+SM4 decryption
	// middleware, if mounted later in the chain) can still bind it.
	c.Request.Body = io.NopCloser(bytes.NewBuffer(raw))

	masked := FilterSensitiveParams(string(raw))
	Record(c, operLogSvc, db, module, operType, WithOperParam(masked))
}

// FilterSensitiveParams masks the value of any JSON key whose name
// case-insensitively contains one of the sensitiveKeys substrings. It performs
// a loop-with-resume so EVERY occurrence per keyword is replaced (the legacy
// implementation only masked the first). Inputs larger than maxFilteredBytes
// (8192) are logged with a warning and only the truncated prefix is scanned;
// callers needing full masking on large bodies should pass an explicit
// WithOperParam.
//
// Example:
//
//	in:  {"username":"alice","password":"hunter2","oldPassword":"pw1"}
//	out: {"username":"alice","password":"******","oldPassword":"******"}
func FilterSensitiveParams(params string) string {
	if params == "" {
		return params
	}

	// Cap input length to bound CPU (threat T-34-02 + T-34-05).
	if len(params) > maxFilteredBytes {
		log.Printf("operlog: input > %d bytes, sensitive-param filter operating on truncated prefix", maxFilteredBytes)
		// Scan the truncated prefix so we still detect (and warn about) known
		// sensitive patterns even when we cannot safely mask the full body.
		truncated := params[:maxFilteredBytes]
		lower := strings.ToLower(truncated)
		for _, kw := range sensitiveKeys {
			if strings.Contains(lower, strings.ToLower(kw)) {
				log.Printf("operlog WARN: truncated input still contains sensitive keyword %q — endpoint should pass explicit WithOperParam", kw)
				break
			}
		}
		// Operate on the truncated slice only.
		params = truncated
	}

	filtered := params
	for _, key := range sensitiveKeys {
		filtered = maskKeyOccurrences(filtered, key)
	}
	return filtered
}

// maskKeyOccurrences replaces the value of every `"<key>":"<value>"` occurrence
// in s with ******. Matching is case-insensitive on the key. The search resumes
// after each replacement so duplicate keys are all masked.
func maskKeyOccurrences(s, key string) string {
	// Build a case-insensitive search by lowercasing both sides and tracking
	// offsets. To preserve the original casing of the surrounding JSON we
	// locate via the lowercased haystack but slice the original.
	lowerHay := strings.ToLower(s)
	searchNeedle := `"` + strings.ToLower(key) + `":"`
	needleLen := len(searchNeedle)

	from := 0
	for {
		idx := strings.Index(lowerHay[from:], searchNeedle)
		if idx == -1 {
			return s
		}
		absIdx := from + idx
		valueStart := absIdx + needleLen
		// Find the closing quote of the value in the ORIGINAL string (values
		// are not lowercased, so slice from `s`).
		relEnd := strings.Index(s[valueStart:], `"`)
		if relEnd == -1 {
			return s
		}
		valueEnd := valueStart + relEnd
		// Replace value with ******.
		s = s[:valueStart] + "******" + s[valueEnd:]
		// The haystack length changed; rebuild the lowercased view from the
		// replacement point onward by re-lowercasing the tail. Cheaper than
		// re-lowercasing the whole string on every hit for typical inputs.
		lowerHay = lowerHay[:valueStart] + "******" + strings.ToLower(s[valueStart+6:])
		from = valueStart + 6
		if from >= len(s) {
			return s
		}
	}
}
