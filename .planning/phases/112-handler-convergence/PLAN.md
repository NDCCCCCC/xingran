---
phase: "112"
plan: "01"
type: execute
wave: 1
depends_on: []
files_modified:
  - internal/api/v1/operations/building_handler.go
  - internal/api/v1/operations/floor_handler.go
  - internal/api/v1/operations/workstation_handler.go
  - internal/api/v1/operations/building_handler_test.go
  - internal/api/v1/operations/floor_handler_test.go
  - internal/api/v1/operations/workstation_handler_test.go
autonomous: true
requirements_addressed: [HANDLER-01, HANDLER-02, HANDLER-03]
must_haves:
  truths:
    - "operations building/floor/workstation Statistics + Search*Options + GetWorkstationDeptOptions handlers use HandleServiceError (no err.Error() leak, no int-as-first-arg quirk)"
    - "operations floor handler.List at line 102 uses HandleServiceError instead of InternalServerErrorWithMsg (preserves err for server-side log)"
    - "GUARD-06 (TestBuildingStatistics_ErrorBody_DoesNotLeakSQL) goes GREEN: SQL keywords absent from response body"
    - "Existing TestBuildingHandler_Statistics_Error / TestBuildingHandler_SearchBuildingOptions_Error assertion http.StatusBadRequest is replaced with http.StatusInternalServerError (HTTPStatus for CodeServerError=1500 is 500) — quirky int→400 path is closed"
  artifacts:
    - path: "internal/api/v1/operations/building_handler.go:40"
      provides: "Statistics handler using HandleServiceError"
    - path: "internal/api/v1/operations/building_handler.go:55"
      provides: "SearchBuildingOptions handler using HandleServiceError"
    - path: "internal/api/v1/operations/floor_handler.go:36"
      provides: "Statistics handler using HandleServiceError"
    - path: "internal/api/v1/operations/floor_handler.go:51"
      provides: "SearchFloorOptions handler using HandleServiceError"
    - path: "internal/api/v1/operations/floor_handler.go:102"
      provides: "List handler using HandleServiceError (replaces InternalServerErrorWithMsg)"
    - path: "internal/api/v1/operations/workstation_handler.go:58"
      provides: "Statistics handler using HandleServiceError"
    - path: "internal/api/v1/operations/workstation_handler.go:83"
      provides: "GetWorkstationDeptOptions handler using HandleServiceError"
    - path: "internal/api/v1/operations/workstation_handler.go:101"
      provides: "SearchWorkstationOptions handler using HandleServiceError"
  key_links:
    - from: "internal/api/v1/operations/{building,floor,workstation}_handler.go"
      to: "pkg/response.HandleServiceError"
      via: "if !response.HandleServiceError(c, err, operation) { return }"
      pattern: "HandleServiceError"
    - from: "pkg/response/handler_helpers.go (D-104-5)"
      to: "apperrors.AppError + sanitized message"
      via: "HandleServiceError suppresses err.Error() from body for non-AppError"
      pattern: "operation.*失败"
---

<objective>
Fix HANDLER-01..03: Replace `response.Error(c, http.StatusInternalServerError, err.Error())` patterns in three operations handlers (`building_handler.go`, `floor_handler.go`, `workstation_handler.go`) with the project-standard `response.HandleServiceError(c, err, operation)` helper. This closes the SQL/infrastructure error leak path and the pre-existing `int`-first-arg `toAppError` quirk (which maps int to HTTP 400 instead of 500). GUARD-06 (`TestBuildingStatistics_ErrorBody_DoesNotLeakSQL`) goes GREEN as a side effect.

Per Phase 104 D-104-5 convention (see `pkg/response/handler_helpers.go:23-40`), `HandleServiceError`:
- passes through `*apperrors.AppError` (preserves business code + HTTPStatus)
- for bare errors, emits `operation + "失败"` as the response message - never `err.Error()`

For `floor_handler.go:List` (line 102), the current code uses `apperrors.InternalServerErrorWithMsg("查询失败")` which discards the underlying err. Migrate to `HandleServiceError(c, err, "查询")` to keep err in the server-side log while sanitizing the response.
</objective>

<context>
@internal/api/v1/operations/building_handler.go:35-59 (Statistics + SearchBuildingOptions - RED state)
@internal/api/v1/operations/floor_handler.go:32-107 (Statistics + SearchFloorOptions + List)
@internal/api/v1/operations/workstation_handler.go:52-105 (Statistics + GetWorkstationDeptOptions + SearchWorkstationOptions)
@pkg/response/handler_helpers.go:23-40 (HandleServiceError - authoritative sanitization helper)
@pkg/errors/errors.go (apperrors.InternalServerError / Wrap / New APIs)
@internal/api/v1/operations/building_handler_test.go:447-618 (GUARD-06 test + quirky Error assertions)
@internal/api/v1/operations/floor_handler_test.go (TestFloorStatistics_Error / TestFloorSearchFloorOptions_Error mirrors)
@internal/api/v1/operations/workstation_handler_test.go (TestWorkstationStatistics_Error / TestWorkstationSearchOptions_Error mirrors)

**Current RED state (building_handler.go:40 + 55):**
```go
// Statistics
result, err := h.service.Statistics(c.Request.Context(), params)
if err != nil {
    response.Error(c, http.StatusInternalServerError, err.Error()) // LEAKS err
    return
}

// SearchBuildingOptions
result, err := h.service.SearchBuildingOptions(c.Request.Context(), params)
if err != nil {
    response.Error(c, http.StatusInternalServerError, err.Error()) // LEAKS err
    return
}
```

**RED quirk** (response.go:157-162): `response.Error(c, int, ...)` calls `toAppError` with `case int`, which hardcodes `HTTPStatus: http.StatusBadRequest`. So even though `http.StatusInternalServerError` is passed, the actual HTTP status returned is **400**. The handler "appears" to leak err.Error() AND misreports status.

**After fix:**
```go
result, err := h.service.Statistics(c.Request.Context(), params)
if !response.HandleServiceError(c, err, "查询统计数据") {
    return
}
```

HandleServiceError:
- If err is nil -> return true (success path continues)
- If err is `*apperrors.AppError` -> use its Code/Message/HTTPStatus verbatim
- Otherwise -> emit `apperrors.InternalServerError(err)` wrapper; the message becomes "查询统计数据失败" (no err detail leaked); HTTPStatus = 500

**Floor handler.List (line 102) is a special case** - current code discards err entirely:
```go
result, err := h.service.List(c.Request.Context(), params)
if err != nil {
    response.Error(c, apperrors.InternalServerErrorWithMsg("查询失败")) // err lost
    return
}
```
Migrate to `HandleServiceError(c, err, "查询")` so the err survives server-side logging (the apperrors.Wrap path retains it).

**Existing tests that document the quirk (must update):**
- `TestBuildingHandler_Statistics_Error` (line 452): currently asserts `http.StatusBadRequest`. After fix, must assert `http.StatusInternalServerError`.
- `TestBuildingHandler_SearchBuildingOptions_Error` (line 502): same.
- `TestBuildingHandler_GetByID_NotFound` (line 292): uses `apperrors.BuildingNotFound()` -> DefaultHTTPStatus=400 (code 3010 is in 2000-8999 range). This is **out of scope** for HANDLER-01 (the code is already an AppError, not the int-quirk path). Leave this test alone.

Check floor_handler_test.go and workstation_handler_test.go for any Statistics/Search*Options Error tests that document the quirk - update them to assert `http.StatusInternalServerError`.
</context>

<tasks>

<task type="auto">
  <name>Task 1: Migrate building_handler Statistics + SearchBuildingOptions (HANDLER-01)</name>
<files>
internal/api/v1/operations/building_handler.go
</files>
<read_first>
- internal/api/v1/operations/building_handler.go:35-59 (current Statistics + SearchBuildingOptions)
- pkg/response/handler_helpers.go:23-40 (HandleServiceError)
</read_first>
<action>
In `internal/api/v1/operations/building_handler.go`, modify two handlers.

**Statistics handler (lines 35-44):**

Replace:
```go
func (h *BuildingHandler) Statistics(c *gin.Context) {
    var params map[string]interface{}
    _ = c.ShouldBindJSON(&params)
    result, err := h.service.Statistics(c.Request.Context(), params)
    if err != nil {
        response.Error(c, http.StatusInternalServerError, err.Error())
        return
    }
    response.Success(c, result)
}
```

With:
```go
func (h *BuildingHandler) Statistics(c *gin.Context) {
    var params map[string]interface{}
    _ = c.ShouldBindJSON(&params)
    result, err := h.service.Statistics(c.Request.Context(), params)
    if !response.HandleServiceError(c, err, "查询统计数据") {
        return
    }
    response.Success(c, result)
}
```

**SearchBuildingOptions handler (lines 46-59):**

Replace:
```go
func (h *BuildingHandler) SearchBuildingOptions(c *gin.Context) {
    var params map[string]interface{}
    if err := c.ShouldBindJSON(&params); err != nil {
        params = map[string]interface{}{}
    }
    result, err := h.service.SearchBuildingOptions(c.Request.Context(), params)
    if err != nil {
        response.Error(c, http.StatusInternalServerError, err.Error())
        return
    }
    response.Success(c, result)
}
```

With:
```go
func (h *BuildingHandler) SearchBuildingOptions(c *gin.Context) {
    var params map[string]interface{}
    if err := c.ShouldBindJSON(&params); err != nil {
        params = map[string]interface{}{}
    }
    result, err := h.service.SearchBuildingOptions(c.Request.Context(), params)
    if !response.HandleServiceError(c, err, "查询楼宇下拉选项") {
        return
    }
    response.Success(c, result)
}
```

After both edits, the `net/http` import becomes unused (only Statistics/SearchBuildingOptions used it for the int first arg). Keep the import ONLY if other handlers in the file use `http.StatusInternalServerError` directly - check the rest of building_handler.go. GetByID, Geocode, etc. use apperrors.X helpers and response.HandleServiceError, so the `net/http` import is no longer needed. Remove the import line.
</action>
<verify>
go build ./internal/api/v1/operations/...
go test ./internal/api/v1/operations/ -run "TestBuildingHandler_Statistics|TestBuildingHandler_SearchBuildingOptions" -v
</verify>
<done>Building handler Statistics + SearchBuildingOptions use HandleServiceError; net/http import removed if unused; no regressions</done>
</task>

<task type="auto">
  <name>Task 2: Migrate floor_handler Statistics + SearchFloorOptions + List (HANDLER-02)</name>
<files>
internal/api/v1/operations/floor_handler.go
</files>
<read_first>
- internal/api/v1/operations/floor_handler.go:32-107 (current RED state for all three)
- pkg/response/handler_helpers.go:23-40
- internal/api/v1/operations/building_handler.go:108-114 (List reference pattern using HandleServiceError)
</read_first>
<action>
In `internal/api/v1/operations/floor_handler.go`, modify three handlers.

**Statistics (lines 32-40):**

Replace:
```go
func (h *FloorHandler) Statistics(c *gin.Context) {
    result, err := h.service.Statistics(c.Request.Context())
    if err != nil {
        response.Error(c, http.StatusInternalServerError, err.Error())
        return
    }
    response.Success(c, result)
}
```

With:
```go
func (h *FloorHandler) Statistics(c *gin.Context) {
    result, err := h.service.Statistics(c.Request.Context())
    if !response.HandleServiceError(c, err, "查询统计数据") {
        return
    }
    response.Success(c, result)
}
```

**SearchFloorOptions (lines 42-55):**

Replace:
```go
func (h *FloorHandler) SearchFloorOptions(c *gin.Context) {
    var params map[string]interface{}
    if err := c.ShouldBindJSON(&params); err != nil {
        params = map[string]interface{}{}
    }
    result, err := h.service.SearchFloorOptions(c.Request.Context(), params)
    if err != nil {
        response.Error(c, http.StatusInternalServerError, err.Error())
        return
    }
    response.Success(c, result)
}
```

With:
```go
func (h *FloorHandler) SearchFloorOptions(c *gin.Context) {
    var params map[string]interface{}
    if err := c.ShouldBindJSON(&params); err != nil {
        params = map[string]interface{}{}
    }
    result, err := h.service.SearchFloorOptions(c.Request.Context(), params)
    if !response.HandleServiceError(c, err, "查询楼层下拉选项") {
        return
    }
    response.Success(c, result)
}
```

**List (lines 92-107) - SPECIAL CASE (err currently discarded via InternalServerErrorWithMsg):**

Replace:
```go
func (h *FloorHandler) List(c *gin.Context) {
    var params map[string]interface{}
    if err := c.ShouldBindJSON(&params); err != nil {
        // 如果JSON解析失败，使用空参数
        params = make(map[string]interface{})
    }

    result, err := h.service.List(c.Request.Context(), params)
    if err != nil {
        response.Error(c, apperrors.InternalServerErrorWithMsg("查询失败"))
        return
    }

    response.Success(c, result)
}
```

With:
```go
func (h *FloorHandler) List(c *gin.Context) {
    var params map[string]interface{}
    if err := c.ShouldBindJSON(&params); err != nil {
        // 如果JSON解析失败，使用空参数
        params = make(map[string]interface{})
    }

    result, err := h.service.List(c.Request.Context(), params)
    if !response.HandleServiceError(c, err, "查询楼层列表") {
        return
    }

    response.Success(c, result)
}
```

Key changes for List: HandleServiceError wraps the bare err as `apperrors.InternalServerError(err)` - this preserves err in the chain (via AppError.Unwrap) for server-side logging while emitting sanitized "查询楼层列表失败" + status 500 to the client. This matches the convention in building_handler.go:List line 109.

After all three edits: check if `net/http` import is still used elsewhere in floor_handler.go. GetByID uses `apperrors.FloorNotFound()`, GetTree uses `apperrors.InternalServerErrorWithMsg`, BatchOperation uses `apperrors.InvalidOperation`. None use raw `http.StatusXxx` constants, so the import is no longer needed. Remove it.

GetTree (line 118-126) is OUT OF SCOPE - it has the same err-discard anti-pattern but HANDLER-02 only covers List.
</action>
<verify>
go build ./internal/api/v1/operations/...
go test ./internal/api/v1/operations/ -run "TestFloorHandler_Statistics|TestFloorHandler_SearchFloorOptions|TestFloorHandler_List" -v
</verify>
<done>Floor handler Statistics + SearchFloorOptions + List use HandleServiceError; net/http import removed if unused; List preserves err server-side</done>
</task>

<task type="auto">
  <name>Task 3: Migrate workstation_handler Statistics + GetWorkstationDeptOptions + SearchWorkstationOptions (HANDLER-03)</name>
<files>
internal/api/v1/operations/workstation_handler.go
</files>
<read_first>
- internal/api/v1/operations/workstation_handler.go:52-105 (current RED state for all three)
- pkg/response/handler_helpers.go:23-40
</read_first>
<action>
In `internal/api/v1/operations/workstation_handler.go`, modify three handlers.

**Statistics (lines 52-62):**

Replace:
```go
func (h *WorkstationHandler) Statistics(c *gin.Context) {
    var params map[string]interface{}
    _ = c.ShouldBindJSON(&params)
    result, err := h.service.Statistics(c.Request.Context(), params)
    if err != nil {
        response.Error(c, http.StatusInternalServerError, err.Error())
        return
    }
    response.Success(c, result)
}
```

With:
```go
func (h *WorkstationHandler) Statistics(c *gin.Context) {
    var params map[string]interface{}
    _ = c.ShouldBindJSON(&params)
    result, err := h.service.Statistics(c.Request.Context(), params)
    if !response.HandleServiceError(c, err, "查询统计数据") {
        return
    }
    response.Success(c, result)
}
```

**GetWorkstationDeptOptions (lines 64-87):**

Replace:
```go
func (h *WorkstationHandler) GetWorkstationDeptOptions(c *gin.Context) {
    var req struct {
        OrgID string `json:"orgId"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        req.OrgID = ""
    }
    result, err := h.service.GetWorkstationDeptOptions(c.Request.Context(), req.OrgID)
    if err != nil {
        response.Error(c, http.StatusInternalServerError, err.Error())
        return
    }
    response.Success(c, result)
}
```

With:
```go
func (h *WorkstationHandler) GetWorkstationDeptOptions(c *gin.Context) {
    var req struct {
        OrgID string `json:"orgId"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        req.OrgID = ""
    }
    result, err := h.service.GetWorkstationDeptOptions(c.Request.Context(), req.OrgID)
    if !response.HandleServiceError(c, err, "查询工位部门下拉选项") {
        return
    }
    response.Success(c, result)
}
```

**SearchWorkstationOptions (lines 89-105):**

Replace:
```go
func (h *WorkstationHandler) SearchWorkstationOptions(c *gin.Context) {
    var req requests.WorkstationListRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        req = requests.WorkstationListRequest{}
    }
    result, err := h.service.SearchWorkstationOptions(c.Request.Context(), req)
    if err != nil {
        response.Error(c, http.StatusInternalServerError, err.Error())
        return
    }
    response.Success(c, result)
}
```

With:
```go
func (h *WorkstationHandler) SearchWorkstationOptions(c *gin.Context) {
    var req requests.WorkstationListRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        req = requests.WorkstationListRequest{}
    }
    result, err := h.service.SearchWorkstationOptions(c.Request.Context(), req)
    if !response.HandleServiceError(c, err, "查询工位下拉选项") {
        return
    }
    response.Success(c, result)
}
```

After all three edits: check if `net/http` import is still used in workstation_handler.go. The remaining handler paths use apperrors.X helpers or HandleServiceError. Remove `net/http` import.

**NOTE on workstation_handler.go List (lines 142-159)**: currently uses `response.Error(c, apperrors.InternalServerErrorWithMsg("查询失败"))` (also discards err, similar to floor List). This is OUT OF SCOPE for HANDLER-03 - the requirement only covers Statistics / GetWorkstationDeptOptions / SearchWorkstationOptions. Flag for follow-up phase if needed.
</action>
<verify>
go build ./internal/api/v1/operations/...
go test ./internal/api/v1/operations/ -run "TestWorkstationHandler_Statistics|TestWorkstationHandler_GetWorkstationDeptOptions|TestWorkstationHandler_SearchWorkstationOptions" -v
</verify>
<done>Workstation handler Statistics + GetWorkstationDeptOptions + SearchWorkstationOptions use HandleServiceError; net/http import removed if unused; no regressions</done>
</task>

<task type="auto">
  <name>Task 4: Update test assertions for Statistics_Error / Search*Options_Error (quirk closure)</name>
<files>
internal/api/v1/operations/building_handler_test.go
internal/api/v1/operations/floor_handler_test.go
internal/api/v1/operations/workstation_handler_test.go
</files>
<read_first>
- internal/api/v1/operations/building_handler_test.go:447-513 (TestBuildingHandler_Statistics_Error + TestBuildingHandler_SearchBuildingOptions_Error)
- internal/api/v1/operations/building_handler_test.go:597-618 (GUARD-06 TestBuildingStatistics_ErrorBody_DoesNotLeakSQL)
</read_first>
<action>
Update test assertions that document the now-closed int-first-arg quirk.

**building_handler_test.go TestBuildingHandler_Statistics_Error (line 452-464):**

Change the final assertion from:
```go
assert.Equal(t, http.StatusBadRequest, w.Code)
```
to:
```go
assert.Equal(t, http.StatusInternalServerError, w.Code)
```

Remove or update the comment block above it (lines 447-451) that says "Quirky: int first arg... Per D-12 we do NOT fix this - only document." Replace with: "Statistics error returns HTTP 500 via HandleServiceError (HANDLER-01 fixed the int-quirk)."

**building_handler_test.go TestBuildingHandler_SearchBuildingOptions_Error (line 502-513):**

Same change: `http.StatusBadRequest` -> `http.StatusInternalServerError`. Update the comment block (lines 499-501) similarly.

**building_handler_test.go TestBuildingHandler_GetByID_NotFound (line 292-305):**

OUT OF SCOPE - this uses apperrors.BuildingNotFound() which has DefaultHTTPStatus=400 by design (code 3010 in 2000-8999 range maps to 400). Leave the test unchanged.

**GUARD-06 TestBuildingStatistics_ErrorBody_DoesNotLeakSQL (line 597-618):**

The GUARD-06 test should now PASS (currently RED). The test asserts body does NOT contain SQL keywords - after HANDLER-01 fix, the body will be `{"code":1500,"message":"查询统计数据失败"}` (no SQL keywords). Keep the test as-is - it asserts the post-fix invariant. The comment block at 589-596 still describes the RED state accurately - update it to note that after Phase 112 HANDLER-01 fix, the test is now GREEN.

**floor_handler_test.go and workstation_handler_test.go:**

Search for tests that:
- Test Statistics / Search*Options / GetWorkstationDeptOptions endpoints
- Trigger an error path
- Assert `http.StatusBadRequest` with a comment about the int-quirk

For each found test: change status assertion to `http.StatusInternalServerError` and update comment. If no such tests exist, verify with `grep -l "int first arg" floor_handler_test.go workstation_handler_test.go` returning empty (no stale quirk docs remain).

Note: the existing TestBuildingHandler_Statistics_Error / TestBuildingHandler_SearchBuildingOptions_Error comments also need the body-not-contains assertion to align with GUARD-06 (i.e. add `assert.NotContains(t, w.Body.String(), "stats fail")` and similar). This makes the existing tests serve as regression sentinels for HANDLER-01.
</action>
<verify>
go test ./internal/api/v1/operations/ -v
grep -l "int first arg" internal/api/v1/operations/*_test.go && echo "FAIL: stale quirk docs" || echo "PASS: no stale quirk docs"
go test ./internal/api/v1/operations/ -run TestBuildingStatistics_ErrorBody_DoesNotLeakSQL -v
</verify>
<done>All Error-path test assertions updated; GUARD-06 GREEN; quirk comments either removed or updated to describe post-fix state</done>
</task>

</tasks>

<success_criteria>
- GUARD-06 (`TestBuildingStatistics_ErrorBody_DoesNotLeakSQL`) GREEN
- 8 endpoints migrated: building {Statistics, SearchBuildingOptions}, floor {Statistics, SearchFloorOptions, List}, workstation {Statistics, GetWorkstationDeptOptions, SearchWorkstationOptions}
- No remaining `response.Error(c, http.StatusInternalServerError, err.Error())` patterns in operations handlers (grep clean)
- Floor handler.List preserves err via apperrors.InternalServerError wrapper chain (server-side log retains original error)
- All building/floor/workstation handler tests pass with updated Error-path assertions
- `go build ./...` exits 0
- `go test ./internal/api/v1/operations/...` exits 0
- `go test ./...` no regressions
- `grep -rn "int first arg" internal/api/v1/operations/` returns empty (stale quirk docs purged)
</success_criteria>

<output>
Commit: feat(112-01): migrate operations Statistics/Search*Options/List to HandleServiceError (HANDLER-01..03)
Files: internal/api/v1/operations/{building,floor,workstation}_handler.go + _test.go (6 files)
Summary: .planning/phases/112-handler-convergence/112-01-SUMMARY.md
</output>

---

---
phase: "112"
plan: "02"
type: execute
wave: 1
depends_on: []
files_modified:
  - pkg/response/handler_helpers.go
  - internal/api/v1/monitor/login_log_handler.go
autonomous: true
requirements_addressed: [HANDLER-04, HANDLER-05]
must_haves:
  truths:
    - "GUARD-04 (TestHandleGetByID_Returns404_NotBadRequest) GREEN: HandleGetByID returns HTTP 404 when getter errors"
    - "GUARD-05 (TestLoginLog_Clean_NilCore_DoesNotPanic) GREEN: LoginLogHandler.Clean does not panic when h.core is nil"
    - "LoginLogHandler.Clean mirrors the OperLogHandler.Clean nil-guard pattern exactly: `h.core != nil && h.core.OperLogService != nil && h.core.GetDB() != nil`"
    - "HandleGetByID uses apperrors.NewWithHTTPStatus to force HTTP 404 (CodeRecordNotFound default maps to 400 via 1000-1020 range, so HTTPStatus must be explicit)"
  artifacts:
    - path: "pkg/response/handler_helpers.go:62"
      provides: "HandleGetByID returns HTTP 404 via apperrors.NewWithHTTPStatus"
    - path: "internal/api/v1/monitor/login_log_handler.go:106"
      provides: "Clean handler with three-way nil guard (h.core + OperLogService + GetDB)"
  key_links:
    - from: "pkg/response/handler_helpers.go:62"
      to: "pkg/errors.NewWithHTTPStatus"
      via: "NewWithHTTPStatus(CodeRecordNotFound, http.StatusNotFound, msg)"
      pattern: "NewWithHTTPStatus"
    - from: "internal/api/v1/monitor/login_log_handler.go:106"
      to: "operlog.Record"
      via: "guard h.core before deref"
      pattern: "h.core != nil && h.core.OperLogService != nil"
    - from: "internal/api/v1/monitor/oper_log_handler.go:117-119 (sister)"
      to: "LoginLogHandler.Clean"
      via: "mirror three-way nil guard pattern"
      pattern: "h.core != nil && h.core.OperLogService != nil && h.core.GetDB() != nil"
---

<objective>
Fix HANDLER-04 + HANDLER-05:

**HANDLER-04**: `LoginLogHandler.Clean` (login_log_handler.go:106) dereferences `h.core.OperLogService` and `h.core.GetDB()` without nil-guarding `h.core` itself. When the handler is wired without a Core dependency (test injection, partial bootstrap), `h.core` is nil and accessing its fields panics. Mirror the symmetric nil-guard from the sister file `oper_log_handler.go:117-119`.

**HANDLER-05**: `pkg/response/handler_helpers.go:62` calls `Error(c, http.StatusNotFound, notFoundMessage)`. The `int` first arg hits `toAppError` `case int:` which hardcodes `HTTPStatus: http.StatusBadRequest`. The actual HTTP status returned is 400, not the intended 404. Fix: use `apperrors.NewWithHTTPStatus(http.StatusNotFound, apperrors.CodeRecordNotFound, notFoundMessage)` to force HTTP 404.
</objective>

<context>
@pkg/response/handler_helpers.go:52-68 (current HandleGetByID - RED state)
@pkg/response/handler_helpers.go:23-40 (HandleServiceError - reference pattern)
@pkg/response/handler_helpers_test.go:24-46 (GUARD-04 test - already exists, currently RED)
@pkg/response/response.go:138-187 (toAppError case int - confirms quirk)
@pkg/errors/errors.go:77-84 (NewWithHTTPStatus signature)
@pkg/errors/errors.go:9-15 (AppError.HTTPStatus field)
@pkg/errors/codes.go:16 + 195-216 (CodeRecordNotFound + DefaultHTTPStatus returns 400 for 1000-1020 range)
@internal/api/v1/monitor/login_log_handler.go:99-109 (current Clean - RED state)
@internal/api/v1/monitor/oper_log_handler.go:101-119 (sister file - mirror pattern at 117-119)
@internal/api/v1/monitor/login_log_handler_test.go:347-369 (GUARD-05 test - already exists, currently RED)
@internal/utils/operlog/operlog.go:215-231 (operlog.Record nil-safety - already handles operLogSvc/db nil, but NOT h.core nil)

**HANDLER-04 current RED state (login_log_handler.go:106):**
```go
func (h *LoginLogHandler) Clean(c *gin.Context) {
    if err := h.svc.Clean(c.Request.Context()); err != nil {
        response.Error(c, apperrors.InternalServerError(err))
        return
    }

    operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "登录日志", operlog.OperTypeClean)
    //           ^^^^^^^ panics if h.core is nil

    response.Success(c, gin.H{"message": "清空成功"})
}
```

**Symmetric nil guard from oper_log_handler.go:117-119:**
```go
if h.core != nil && h.core.OperLogService != nil && h.core.GetDB() != nil {
    _ = h.core.OperLogService.RecordOperLog(c.Request.Context(), h.core.GetDB(), cleanAuditRow)
}
```

**Note**: `operlog.Record` itself is already nil-safe for `operLogSvc` and `db` (operlog.go:229-231: `if c == nil || operLogSvc == nil || db == nil { return }`). The danger is `h.core` being nil - accessing `h.core.OperLogService` evaluates `h.core` first, which panics with nil pointer dereference if h.core is nil. The three-way guard avoids both panics.

**HANDLER-05 current RED state (handler_helpers.go:62):**
```go
func HandleGetByID(c *gin.Context, getter func(string) (interface{}, error), notFoundMessage string) bool {
    id, ok := HandleIDParam(c)
    if !ok {
        return false
    }

    entity, err := getter(id)
    if err != nil {
        Error(c, http.StatusNotFound, notFoundMessage) // <- int first arg quirk
        return false
    }

    Success(c, entity)
    return true
}
```

`Error(c, http.StatusNotFound, ...)` -> `toAppError(int)` -> `case int:` -> `{Code: 404, HTTPStatus: http.StatusBadRequest}`. Net effect: HTTP 400 instead of 404.

**Fix path (HANDLER-05):**
```go
entity, err := getter(id)
if err != nil {
    Error(c, apperrors.NewWithHTTPStatus(http.StatusNotFound, apperrors.CodeRecordNotFound, notFoundMessage))
    return false
}
```

`NewWithHTTPStatus` (errors.go:77-84) explicitly sets `HTTPStatus: http.StatusNotFound` bypassing the Code default. `CodeRecordNotFound` (1010) defaults to 400 but `HTTPStatus` overrides when set. `Error(c, *apperrors.AppError)` -> `toAppError` `case *apperrors.AppError:` -> uses `GetHTTPStatus()` which returns the explicit HTTPStatus if non-zero.

The fix works because:
1. `NewWithHTTPStatus` returns `*AppError{HTTPStatus: http.StatusNotFound, Code: CodeRecordNotFound, Message: notFoundMessage}`
2. `Error(c, appErr)` -> `toAppError(*AppError)` -> copies Code, Message, `GetHTTPStatus()` (returns 404)
3. `c.JSON(404, Response{Code: 1010, Message: notFoundMessage, ...})` -> GUARD-04 PASSES

**Why not use apperrors.NotFound(msg)?** Because `NotFound(msg)` (errors.go:225-227) calls `New(CodeRecordNotFound, msg)` which sets `HTTPStatus: code.DefaultHTTPStatus() = 400`. Same int-quirk as HandleGetByID. Must use `NewWithHTTPStatus` to force 404.
</context>

<tasks>

<task type="auto">
  <name>Task 1: Fix HandleGetByID HTTP status (HANDLER-05)</name>
<files>
pkg/response/handler_helpers.go
</files>
<read_first>
- pkg/response/handler_helpers.go:52-68 (current HandleGetByID - RED state)
- pkg/response/handler_helpers_test.go:24-46 (GUARD-04 test - confirms expected post-fix behavior)
- pkg/errors/errors.go:77-84 (NewWithHTTPStatus signature)
- pkg/response/response.go:138-187 (toAppError case int - confirms quirk is real)
</read_first>
<action>
In `pkg/response/handler_helpers.go`, modify `HandleGetByID` (line 54-68) to use `apperrors.NewWithHTTPStatus` instead of `Error(c, int, ...)`.

Replace:
```go
func HandleGetByID(c *gin.Context, getter func(string) (interface{}, error), notFoundMessage string) bool {
    id, ok := HandleIDParam(c)
    if !ok {
        return false
    }

    entity, err := getter(id)
    if err != nil {
        Error(c, http.StatusNotFound, notFoundMessage)
        return false
    }

    Success(c, entity)
    return true
}
```

With:
```go
// HandleGetByID 通用的 GetByID 处理逻辑
// 返回 true 表示成功，false 表示失败（参数缺失或 getter 错误）
//
// HANDLER-05 (Phase 112): use apperrors.NewWithHTTPStatus to force HTTP 404 on not-found.
// The previous Error(c, http.StatusNotFound, ...) call hit toAppError `case int:` which
// hardcodes HTTPStatus=400. NewWithHTTPStatus bypasses the Code default (CodeRecordNotFound
// DefaultHTTPStatus returns 400 in 1000-1020 range). GUARD-04 test asserts HTTP 404.
func HandleGetByID(c *gin.Context, getter func(string) (interface{}, error), notFoundMessage string) bool {
    id, ok := HandleIDParam(c)
    if !ok {
        return false
    }

    entity, err := getter(id)
    if err != nil {
        Error(c, apperrors.NewWithHTTPStatus(http.StatusNotFound, apperrors.CodeRecordNotFound, notFoundMessage))
        return false
    }

    Success(c, entity)
    return true
}
```

After the edit: check if `net/http` import is still needed. HandleIDParam (line 43-50) uses `http.StatusBadRequest`. Yes - keep the import.

Add a short comment block above HandleGetByID referencing HANDLER-05 + GUARD-04 so future maintainers know why the explicit NewWithHTTPStatus is used (avoiding the int-quirk regression).

The comment includes:
- HANDLER-05 reference (Phase 112)
- Why NewWithHTTPStatus (not just New)
- GUARD-04 regression guard

This is the canonical fix for the int-first-arg quirk; the same pattern should be used in any future Handle* helper that needs a specific HTTP status. Document this as a convention.
</action>
<verify>
go build ./pkg/response/...
go test ./pkg/response/ -run TestHandleGetByID_Returns404_NotBadRequest -v
go test ./pkg/response/...
grep -r "HandleGetByID" --include="*.go" internal/ pkg/ cmd/
</verify>
<done>GUARD-04 GREEN; HandleGetByID returns HTTP 404 not 400; net/http import retained (HandleIDParam uses it)</done>
</task>

<task type="auto">
  <name>Task 2: Add nil guard to LoginLogHandler.Clean (HANDLER-04)</name>
<files>
internal/api/v1/monitor/login_log_handler.go
</files>
<read_first>
- internal/api/v1/monitor/login_log_handler.go:99-109 (current Clean - RED state)
- internal/api/v1/monitor/oper_log_handler.go:101-119 (sister file with correct nil guard at 117-119)
- internal/utils/operlog/operlog.go:215-231 (operlog.Record nil-safety - already handles nil svc/db, but not nil h.core)
- internal/api/v1/monitor/login_log_handler_test.go:347-369 (GUARD-05 test)
</read_first>
<action>
In `internal/api/v1/monitor/login_log_handler.go`, modify `Clean` (lines 99-109) to add the three-way nil guard mirroring `oper_log_handler.go:117-119`.

Replace:
```go
// Clean 清空登录日志
func (h *LoginLogHandler) Clean(c *gin.Context) {
    if err := h.svc.Clean(c.Request.Context()); err != nil {
        response.Error(c, apperrors.InternalServerError(err))
        return
    }

    operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "登录日志", operlog.OperTypeClean)

    response.Success(c, gin.H{"message": "清空成功"})
}
```

With:
```go
// Clean 清空登录日志
//
// HANDLER-04 (Phase 112): mirror OperLogHandler.Clean three-way nil guard at
// oper_log_handler.go:117-119. Without the guard, h.core.OperLogService panics
// when h.core is nil (e.g. handler wired via WithCore(nil) in tests). GUARD-05
// test asserts no panic. operlog.Record itself is nil-safe for operLogSvc/db
// (operlog.go:229-231) but cannot protect against h.core dereference.
func (h *LoginLogHandler) Clean(c *gin.Context) {
    if err := h.svc.Clean(c.Request.Context()); err != nil {
        response.Error(c, apperrors.InternalServerError(err))
        return
    }

    if h.core != nil && h.core.OperLogService != nil && h.core.GetDB() != nil {
        operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "登录日志", operlog.OperTypeClean)
    }

    response.Success(c, gin.H{"message": "清空成功"})
}
```

Key changes:
- `if h.core != nil && h.core.OperLogService != nil && h.core.GetDB() != nil { ... }` wraps the operlog.Record call
- Comment references HANDLER-04 + GUARD-05 + operlog.go:229-231 nil-safety detail
- Symmetric to oper_log_handler.go:117-119 (same guard structure, same order of checks)
- When any check fails (h.core nil OR OperLogService nil OR GetDB nil), the operlog.Record call is skipped silently - matches OperLogHandler pattern

Verify: `go test ./internal/api/v1/monitor/ -run TestLoginLog_Clean_NilCore_DoesNotPanic -v` should PASS (currently RED).

Verify other LoginLog tests still pass:
- `TestLoginLog_Clean_Success` - normal path, h.core wired via setupLoginLogHandler, guard passes, operlog.Record runs
- `TestLoginLog_Clean_Error` - svc.Clean returns error, returns early before operlog.Record, guard not reached

Note: setupLoginLogHandler in login_log_handler_test.go:88-93 uses `&core.Core{CoreInfra: &core.CoreInfra{DB: &db.Database{}}, CoreServices: &core.CoreServices{}}` - DB is non-nil but OperLogService is zero-value (nil). So:
- TestLoginLog_Clean_Success currently calls operlog.Record with nil OperLogService - operlog.Record's own nil-safety catches it (returns silently)
- After HANDLER-04 fix: the three-way guard ALSO catches it (skips the call entirely)
- Net behavior unchanged for existing tests; GUARD-05 (which uses WithCore(nil)) goes RED -> GREEN
</action>
<verify>
go build ./internal/api/v1/monitor/...
go test ./internal/api/v1/monitor/ -run TestLoginLog_Clean -v
go test ./internal/api/v1/monitor/ -run TestLoginLog_Clean_NilCore_DoesNotPanic -v
go test ./internal/api/v1/monitor/...
</verify>
<done>GUARD-05 GREEN; LoginLogHandler.Clean does not panic when h.core is nil; mirror pattern matches OperLogHandler.Clean</done>
</task>

</tasks>

<success_criteria>
- GUARD-04 (`TestHandleGetByID_Returns404_NotBadRequest`) GREEN
- GUARD-05 (`TestLoginLog_Clean_NilCore_DoesNotPanic`) GREEN
- HandleGetByID returns HTTP 404 (not 400) when getter returns error
- LoginLogHandler.Clean no longer panics when h.core is nil
- Three-way nil guard pattern matches OperLogHandler.Clean exactly
- HandleGetByID uses apperrors.NewWithHTTPStatus to force HTTP 404 (works around CodeRecordNotFound default of 400)
- No regressions in `go test ./...`
- No regressions in `go build ./...`
- net/http import in handler_helpers.go retained (HandleIDParam still uses http.StatusBadRequest)
- Comments document HANDLER-N reference + GUARD-N regression guard
</success_criteria>

<output>
Commit: feat(112-02): fix HandleGetByID 404 + LoginLog.Clean nil guard (HANDLER-04..05)
Files: pkg/response/handler_helpers.go, internal/api/v1/monitor/login_log_handler.go
Summary: .planning/phases/112-handler-convergence/112-02-SUMMARY.md
</output>

---

## Phase 112 Completeness Summary

| Plan | Requirements | Files | GUARDs Going GREEN |
|------|--------------|-------|-------------------|
| 112-01 | HANDLER-01, HANDLER-02, HANDLER-03 | 6 (3 handler + 3 test) | GUARD-06 |
| 112-02 | HANDLER-04, HANDLER-05 | 2 (handler_helpers + login_log) | GUARD-04, GUARD-05 |
| **Total** | **5 requirements** | **8 files** | **3 GUARDs GREEN** |

## Out-of-Scope (Locked D-05)

- `pkg/response/response.go:157-162` toAppError case int → 400: not fixed; only avoided by HandleServiceError / NewWithHTTPStatus callers. Removing the case-int branch would break existing `Error(c, http.StatusBadRequest, ...)` callers throughout the codebase (e.g. `Error(c, http.StatusBadRequest, "缺少 ID 参数")` in HandleIDParam).
- `operations/floor_handler.go:118-126` GetTree: still uses `apperrors.InternalServerErrorWithMsg("查询失败")` (also discards err, same anti-pattern as the migrated List handler). Not in HANDLER-02 scope; flag for a follow-up phase if audit requires.
- `operations/workstation_handler.go:143-158` List: still uses `apperrors.InternalServerErrorWithMsg("查询失败")`. Not in HANDLER-03 scope (requirement only covers Statistics + GetWorkstationDeptOptions + SearchWorkstationOptions); flag for follow-up.
- `apperrors.BuildingNotFound()` DefaultHTTPStatus=400 (code 3010): not fixed; the 3000-range codes map to 400 by design. Per-handler 404 override via `NewWithHTTPStatus` is the established escape hatch when a specific handler needs 404.
