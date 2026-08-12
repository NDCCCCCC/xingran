---
phase: 14-frontend-ux
plan: fix-01
type: execute
wave: 1
depends_on: []
gap_closure: true
gap_refs:
  - B1
files_modified:
  - internal/api/v1/network/mac_history_router.go
  - internal/api/v1/network/mac_history_handler.go
  - internal/services/mac_history_query_service.go
autonomous: true
requirements:
  - UI-01
  - UI-02

must_haves:
  truths:
    - "POST /network/history/list 接收 MACHistoryQueryParams(current/pageSize/mac/deviceId/interfaceName/vlanId/eventType/status/startTime/endTime) 并返回 {code:0, data:{list,total,current,pageSize}}"
    - "GET /network/history/list?format=xlsx&exportScope=current|all 接收与 POST 相同的过滤条件,返回 application/vnd.openxmlformats-officedocument.spreadsheetml.sheet 二进制流,Content-Disposition: attachment; filename=mac_history_<scope>_<ts>.xlsx"
    - "当前端查询条件命中 0 行时,Excel 导出仍返回带表头的空工作表而非 4xx"
  artifacts:
    - path: "internal/api/v1/network/mac_history_router.go"
      provides: "新增 POST /history/list + GET /history/list 路由"
      contains: "history/list"
    - path: "internal/api/v1/network/mac_history_handler.go"
      provides: "QueryHistory 列表 handler + ExportHistory Excel handler"
      contains: "ExportHistory"
    - path: "internal/services/mac_history_query_service.go"
      provides: "QueryHistory 通用方法 + ExportHistory xlsx 流式输出"
      contains: "ExportHistory"
  key_links:
    - from: "internal/api/v1/network/mac_history_router.go"
      to: "internal/api/v1/network/mac_history_handler.go"
      via: "r.POST /history/list 调用 handler.QueryHistory"
      pattern: "QueryHistory"
    - from: "internal/api/v1/network/mac_history_handler.go"
      to: "internal/services/mac_history_query_service.go"
      via: "historyQueryService.QueryHistory + ExportHistory"
      pattern: "QueryHistory|ExportHistory"
    - from: "xingran-react-frontend/src/lib/api/networkApi.ts"
      to: "/network/history/list"
      via: "queryMACHistory / exportMACHistory"
      pattern: "/network/history/list"
---

<objective>
补齐 Phase 14 列表页与 Excel 导出功能依赖的后端端点 `POST /network/history/list` 与 `GET /network/history/list?format=xlsx`,使 UI-01 的列表数据流和 UI-02 的真实 xlsx 下载可端到端运行。

Purpose: Phase 14 的前端页面均锁定端点 `/network/history/list`(D-01 + D-14),但 Phase 12/13 实际只注册了 `/history/port`、`/history/device`、`/history/trajectory`、`/history/stats`、`/history/vendor`。`queryMACHistory`、`getMACEvents`、`exportMACHistory` 三个前端 API 全部 404 或返回错误响应。该 plan 复用 Phase 13 已交付的 `mac_history_query_service.go` 数据层,新增"通用按条件过滤"的服务方法 + 两个 HTTP handler,并在 router 注册。
Output: 1 个新的 service 方法 (`QueryHistory`) + 1 个新 service 方法 (`ExportHistory`) + 2 个新 handler + 2 行 router 注册。
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/phases/14-frontend-ux/14-CONTEXT.md (D-01 锁定 POST /network/history/list;D-14 锁定 format=xlsx 复用端点;D-13 锁定 exportScope=current|all;D-15 前端下载模式)
@.planning/phases/14-frontend-ux/14-VERIFICATION.md (B1 段 — endpoint 不存在的根因)
@.planning/REQUIREMENTS.md (UI-01 / UI-02 验收:查询 + Excel 导出 ≤ 30 天)
@.planning/STATE.md (Phase 13 已 shipped,服务层 mac_history_query_service.go 完整可用)
@internal/services/mac_history_query_service.go (复用 QueryPortHistory / QueryDeviceHistory 的 GORM chain 模式;现有 DeviceHistoryQuery 与 PortHistoryQuery 结构体可直接复用其字段)
@internal/api/v1/network/mac_history_handler.go (现有 handler 模式 — responseHelpers.HandleJSONBinding + response.Page)
@internal/api/v1/network/mac_history_router.go (现有 router — gin RouterGroup POST 注册)
@pkg/response/response.go (response.Page / response.Success / response.Error wrappers)
</context>

<tasks>

<task type="auto">
  <name>Task 1: 后端实现 QueryHistory + ExportHistory(service + handler + router)</name>
  <files>internal/services/mac_history_query_service.go, internal/api/v1/network/mac_history_handler.go, internal/api/v1/network/mac_history_router.go</files>
  <read_first>
    - internal/services/mac_history_query_service.go(复用 QueryPortHistory / QueryDeviceHistory 的 GORM chain;`MACHistoryRecord` 字段顺序)
    - internal/api/v1/network/mac_history_handler.go(现有 handler 模板 — responseHelpers.HandleJSONBinding + response.Page)
    - go.mod(确认 xuri/excelize/v2 ^2.10.0 已就位)
    - internal/services/operations/excel_service.go(参考现有 xlsx 写文件模式,如 Phase 16 工位导出)
    - pkg/response/response.go(response.Error / response.Page / response.Success)
  </read_first>
  <action>
    1. 在 `internal/services/mac_history_query_service.go` 的 `MACHistoryRecord` 结构体(行 40-51)下方新增 `MACHistoryListQuery` 结构体,字段:
       ```go
       type MACHistoryListQuery struct {
           Current        int    `json:"current" form:"current" binding:"min=1"`
           PageSize       int    `json:"pageSize" form:"pageSize" binding:"min=1,max=100"`
           MAC            string `json:"mac,omitempty" form:"mac"`
           DeviceID       string `json:"deviceId,omitempty" form:"deviceId"`
           InterfaceName  string `json:"interfaceName,omitempty" form:"interfaceName"`
           VLANID         *int   `json:"vlanId,omitempty" form:"vlanId"`
           EventType      string `json:"eventType,omitempty" form:"eventType"`
           Status         *int   `json:"status,omitempty" form:"status"`
           StartTime      string `json:"startTime,omitempty" form:"startTime"`
           EndTime        string `json:"endTime,omitempty" form:"endTime"`
           ExportScope    string `json:"exportScope,omitempty" form:"exportScope"` // "current" or "all",仅 ExportHistory 使用
       }
       ```
    2. 在 `MACHistoryQueryService` interface(行 151-158)增加 2 个方法:
       ```go
       QueryHistory(ctx context.Context, req *MACHistoryListQuery) (*MACHistoryQueryResult, error)
       ExportHistory(ctx context.Context, req *MACHistoryListQuery, w io.Writer) error
       ```
    3. 在 `macHistoryQueryServiceImpl` 实现 `QueryHistory`(参照 QueryDeviceHistory 行 397-496 的模式):
       - 调 `validateMACAddress(req.MAC)` 仅当 `req.MAC != ""`,失败返回 error。
       - 若 `req.DeviceID != ""`,调 `uuid.Parse` 校验,失败返回 "无效的设备ID格式"。
       - 默认值:`req.Current < 1 → 1`,`req.PageSize < 1 → 20`。
       - 时间范围 maxQueryRange 复用 `365 * 24 * time.Hour`(行 319);若 UI-02 要 30 天硬上限,加 `if req.ExportScope == "" && endTime.Sub(startTime) > 30*24*time.Hour` 时只在 ExportHistory 报错 — ExportHistory 单独 enforce 30 天上限,QueryHistory 允许 365 天给前端列表页用。
       - 构建 `query := s.db.WithContext(ctx).Table("sys_device_mac_history")`,链式 Where 依次加:device_id(若非空)、interface_name(若非空)、mac_address(若非空,先 normalizeMAC)、vlan_id(若 *int 非空)、event_type(若非空)、status(若 *int 非空)、first_seen 时间范围(若 start/end 非空)。
       - `Count(&total)`、`Order("first_seen DESC").Limit(req.PageSize).Offset((Current-1)*PageSize).Find(&records []models.DeviceMACHistory)`。
       - 映射 `[]MACHistoryRecord`,返回 `&MACHistoryQueryResult{List, Total, Current, PageSize}`。
    4. 实现 `ExportHistory`:
       - 同样先 validate mac + uuid,解析时间。
       - **强制 30 天上限**:`if endTime.Sub(startTime) > 30*24*time.Hour { return fmt.Errorf("导出范围最大 30 天,请缩小查询条件") }`。
       - 构造 query(与 QueryHistory 相同的 Where chain,但不加分页),`Find(&records)` 限 100000 行(`LIMIT 100000`)。
       - `f := excelize.NewFile()`;`f.SetSheetName("Sheet1", "MAC 历史")`;按列顺序写表头到 row 1: `f.SetCellValue("MAC 历史", "A1", "时间"); "B1", "MAC"; "C1", "设备"; "D1", "端口"; "E1", "VLAN"; "F1", "事件类型"; "G1", "首次出现"; "H1", "最后出现"; "I1", "采集时间"`。
       - 从 row 2 起逐行 `f.SetCellValue(...)`,时间列用 `record.FirstSeen.Format("2006-01-02 15:04:05")`;VLAN 用 `*record.VLANID`(若非空);事件类型用 `record.EventType`(原始值)。
       - `return f.Write(w)` — 直接写到传入的 `io.Writer`(handler 用 `bytes.Buffer` 缓冲)。
    5. 在 `mac_history_handler.go` 新增 `QueryHistory` handler:
       - `var req mac_history_query_service.MACHistoryListQuery`
       - `responseHelpers.HandleJSONBinding(c, &req)`(POST body 走 JSON binding)
       - `result, err := h.historyQueryService.QueryHistory(c.Request.Context(), &req)` + `responseHelpers.HandleServiceError`
       - `response.Page(c, result.List, result.Total, result.Current, result.PageSize)`
    6. 新增 `ExportHistory` handler:
       - `var req mac_history_query_service.MACHistoryListQuery`
       - `if err := c.ShouldBindQuery(&req); err != nil { response.Error(c, http.StatusBadRequest, "参数错误: "+err.Error()); return }`(GET query 参数 binding)
       - 若 `req.ExportScope == ""`,设为 `"current"`。
       - **关键**:先 `buf := new(bytes.Buffer)` + `if err := h.historyQueryService.ExportHistory(c.Request.Context(), &req, buf); err != nil { response.Error(c, http.StatusInternalServerError, "导出失败: "+err.Error()); return }`,成功后再 `c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")` + `filename := fmt.Sprintf("mac_history_%s_%s.xlsx", req.ExportScope, time.Now().Format("20060102_150405"))` + `c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))` + `c.Status(http.StatusOK)` + `c.Writer.Write(buf.Bytes())`。顺序保证:错误时返回 JSON envelope,不写 xlsx 头;成功时 buffer 已完整。
       - import `"bytes"`、`"time"`、`"net/http"`(若尚未 import)。
    7. 在 `mac_history_router.go`(行 11-26)插入:
       ```go
       r.POST("/history/list", historyHandler.QueryHistory)
       r.GET("/history/list", historyHandler.ExportHistory)
       ```
       位置:在 `r.POST("/history/vendor", ...)` 之后(行 23)。同步更新日志字符串为:`"MAC历史查询路由已注册: /history/port, /history/device, /history/trajectory, /history/stats, /history/vendor, /history/list (POST/GET)"`。
  </action>
  <verify>
    <automated>go build ./...  # 退出码 0</automated>
    <automated>grep -c "/history/list" internal/api/v1/network/mac_history_router.go  # >= 3 (POST 注册 + GET 注册 + 日志字符串)</automated>
    <automated>grep -c "ExportHistory" internal/services/mac_history_query_service.go  # >= 3 (interface + impl 函数名 + 内部引用)</automated>
    <automated>grep -c "ExportHistory\|QueryHistory" internal/api/v1/network/mac_history_handler.go  # >= 4</automated>
    <automated>grep -c "excelize.NewFile\|f.SetCellValue\|f.Write" internal/services/mac_history_query_service.go  # >= 3</automated>
    <automated>grep -c "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" internal/api/v1/network/mac_history_handler.go  # >= 1</automated>
  </verify>
  <acceptance_criteria>
    - `go build ./...` 退出码 0
    - `grep -c "history/list" internal/api/v1/network/mac_history_router.go` >= 3
    - `grep -c "ExportHistory" internal/services/mac_history_query_service.go` >= 3
    - `grep -c "ExportHistory\|QueryHistory" internal/api/v1/network/mac_history_handler.go` >= 4
    - `grep -c "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" internal/api/v1/network/mac_history_handler.go` >= 1
    - `grep -c "Content-Disposition" internal/api/v1/network/mac_history_handler.go` >= 1
    - `grep -c "30\*24\*time.Hour" internal/services/mac_history_query_service.go` >= 1(导出 30 天上限)
    - xlsx 表头 9 列(时间/MAC/设备/端口/VLAN/事件类型/首次出现/最后出现/采集时间)
    - Excel 流先写入 bytes.Buffer 再设响应头(grep 顺序: 错误处理 → write 头 → Write)
  </acceptance_criteria>
  <done>POST /network/history/list 与 GET /network/history/list 双端点上线;service 层 QueryHistory + ExportHistory 实现;router 注册;导出 30 天硬上限;xlsx 流经 bytes.Buffer 缓冲后写头;go build 0 退出码。</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| client → API | untrusted network input crosses here for POST/GET /history/list |
| backend → HTTP response writer | xlsx buffered in bytes.Buffer before write — no filesystem persistence |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-14B1-01 | DoS | QueryHistory / ExportHistory | mitigate | 30-day cap for export (UI-02 lock); ExportHistory LIMIT 100000 rows max; PageSize max=100 |
| T-14B1-02 | Injection | QueryHistory GORM chain | mitigate | All filter values passed as GORM `?` placeholders, never string-concatenated; MAC validated via validateMACAddress; UUID via uuid.Parse |
| T-14B1-03 | Information Disclosure | ExportHistory filename | accept | Filename includes ExportScope + timestamp; scope reflects query params, not user-controlled path (no path traversal) |
| T-14B1-04 | Tampering | response headers before write | mitigate | Buffer xlsx to bytes.Buffer BEFORE setting Content-Type/Content-Disposition; on error, response.Error writes JSON envelope, not partial xlsx |
| T-14B1-05 | Repudiation | QueryHistory logging | mitigate | applogger.Errorf on query failures (mirrors existing handlers) |
| T-14B1-SC | Tampering | npm/pip/cargo installs | mitigate | No new dependencies — excelize v2 already in go.mod; pure refactor + new functions in existing files |
</threat_model>

<verification>
- `cd D:/code/ClaudeCode/xingran-go-backend && go build ./...` exits 0
- `grep -c "history/list" internal/api/v1/network/mac_history_router.go` >= 3
- `grep -c "QueryHistory\|ExportHistory" internal/services/mac_history_query_service.go` >= 6
- `grep -c "QueryHistory\|ExportHistory" internal/api/v1/network/mac_history_handler.go` >= 4
- Manual smoke test (out-of-band): start backend, then `curl -X POST http://localhost:9000/api/v1/network/history/list -H "Authorization: Bearer ..." -d '{"current":1,"pageSize":20,"startTime":"2026-01-01T00:00:00Z","endTime":"2026-06-15T00:00:00Z"}'` returns `{code:0, data:{list:[...], total:N, current:1, pageSize:20}}`
- Manual smoke test (out-of-band): `curl -X GET "http://localhost:9000/api/v1/network/history/list?exportScope=current&startTime=2026-01-01T00:00:00Z&endTime=2026-06-15T00:00:00Z" -H "Authorization: Bearer ..." -o /tmp/out.xlsx` produces a non-empty .xlsx file with content-type header
</verification>

<success_criteria>
- 2 个新 service 方法(QueryHistory + ExportHistory)在 mac_history_query_service.go 中实现并加到 interface
- 2 个新 handler(QueryHistory + ExportHistory)在 mac_history_handler.go 中实现
- router 注册 POST + GET /history/list
- Excel 流式输出经过 bytes.Buffer 缓冲后再写头
- 时间跨度上限 30 天(导出)
- go build ./... 退出码 0,go vet 无警告
- 前端 `queryMACHistory` 与 `exportMACHistory` 调用可正常返回(端到端需后端运行)
</success_criteria>

<output>
Create `.planning/phases/14-frontend-ux/14-fix-01-SUMMARY.md` when done
</output>