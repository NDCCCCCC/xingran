# Phase 35: operlog.exclude_paths 配置驱动白名单 - Context

**Gathered:** 2026-06-16
**Status:** Ready for planning
**Source:** Explore Session (`.planning/notes/260616-rpa-heartbeat-log-flooding.md`)

---

<domain>

## Phase Boundary

**In scope:**
- 在 `internal/utils/operlog` 包内引入配置驱动的排除路径机制
- `Record` / `RecordWithBody` 函数体首行判定 `c.Request.URL.Path` 是否在排除列表中，命中即早退（不调用 `RecordAsync`）
- 新增 `func Configure(paths []string)` + `func IsExcludedPath(path string) bool`
- `core.go` 在配置加载后调一次 `operlog.Configure(...)`
- `configs/config.yaml` / `configs/config.dev.yaml` 新增 `operlog.exclude_paths` 配置项
- regression_test.go 新增 `TestExcludedPathsEarlyReturn`，operlog_test.go 新增 `TestIsExcludedPath`

**Out of scope (deferred):**
- rpaApi.ts 路径错配（前端 `/${id}/progress` 与后端 `/progress` 不匹配）→ 独立 todo
- `/rpa/workers/autoscale/config` (POST) 无持久化（handler 内 `// TODO`）→ 独立 todo
- operlog 运行时热改 `Refresh()` API → 待运维需求出现
- 修改 rpa-worker Agent 端的 heartbeat 频率 → 不在 v1.16 范围

</domain>

<decisions>

## Implementation Decisions (LOCKED)

### D-01: 配置项命名与位置
- **配置项**：`operlog.exclude_paths` （数组，位于 `configs/config.yaml` 顶层 `operlog:` 节点）
- **默认值**：`["/api/v1/rpa/workers/*/heartbeat"]`
- **不默认排除**：`/api/v1/rpa/workers/progress`（任务进度审计价值高）、`/api/v1/rpa/workers/register`（Worker 生命周期事件）

### D-02: 排除判定位置
- 在 `internal/utils/operlog/operlog.go` 的 `Record` 与 `RecordWithBody` 函数体**首行**判定（`RecordWithBody` 必须在 `c.GetRawData()` 之前，否则 body 已被消费）
- handler 层不动：所有 handler 仍显式调用 `operlog.Record(...)`，与 Phase 34 哲学一致

### D-03: 匹配实现
- 复用 `filepath.Match` + `/*` 后缀通配，与 `pkg/middleware/request_decryption.go:294-315` 的 `isExcludedPath` 风格完全一致
- `*` 是单段通配（`filepath.Match` 语义），不会跨 `/`，避免误伤
- 暴露 `func IsExcludedPath(path string) bool` 给外部（不私有化，便于测试与复用）

### D-04: 配置加载时机
- `core.go` 在配置加载后、调起任何 handler 前调一次 `operlog.Configure(core.Config.OperLog.ExcludePaths)`
- 启动期单次调用，无并发问题；包级 `var ExcludedPaths []string` 不加 mutex

### D-05: 不破坏 Phase 34 regression_test.go 锁定的 API
- `Record(6 参数 + variadic)` 签名不变
- `RecordWithBody(5 参数非 variadic)` 签名不变
- 25 个 OperType 常量值不变（OPERLOG-03）
- 34 个敏感关键词列表不变
- 新增 API 都是扩展：`Configure`、`IsExcludedPath`、包级 `ExcludedPaths`

### D-06: 测试覆盖
- `regression_test.go` 新增 `TestExcludedPathsEarlyReturn`：mock Recorder，断言路径命中时 `RecordAsync` 不被调用、路径未命中时仍被调用；断言 `RecordWithBody` 在排除路径上不消费 request body（`GetRawData` 不被调用）
- `operlog_test.go` 新增 `TestIsExcludedPath`：字面量 exact match、`/*` 后缀通配、无匹配（return false）、`/list` 误伤防护测试

### Claude's Discretion
- 包级 `ExcludedPaths` 变量是否导出（大写 Exported vs 小写 unexported）— 选择导出，便于调试时直接读包级状态
- `Configure` 函数是否加 mutex（单次启动期调用 → 不需要）
- 是否提供 `Refresh()` 函数用于运行时热改 — 不在 v1.16 范围，deferred

</decisions>

<canonical_refs>

## Canonical References (下游 agent 必读)

### operlog helper 包
- `internal/utils/operlog/operlog.go` — `Record` / `RecordWithBody` 当前实现（行 172-212 / 214-260 区间）；无 path 过滤
- `internal/utils/operlog/regression_test.go` — 公共 API 表面锁定（25 OperType、6 参数 Record、5 参数 RecordWithBody、34 关键词）

### 排除路径实现蓝本
- `pkg/middleware/request_decryption.go:294-315` — `isExcludedPath` 函数（`filepath.Match` + `/*` 后缀通配）；可直接复制函数体
- `configs/config.yaml:86-95` — `security.request_encryption.exclude_paths` 现有配置项结构参考

### 现有 exclude 配置加载参考
- `internal/core/core.go` — 启动期初始化流程；需找出现有 `security.request_encryption.exclude_paths` 在 core 启动期的处理位置，加 `operlog.Configure(...)` 一行
- `internal/config/config.go`（或类似）— 配置 struct 反序列化位置；需确认是否能添加 `OperLog.ExcludePaths []string` 字段

### Phase 34 设计哲学（不可违反）
- `.planning/phases/34-oper-log-full-coverage/34-01-PLAN.md:53-55` — "为 Wave 2 (系统核心) 无需修改 Record 签名就可以传 oper_param"，确认 Record 签名是稳定契约
- `.planning/phases/34-oper-log-full-coverage/34-REVIEW.md:280-287` — "Convention Adherence"，handler 显式 Record 不依赖中间件
- `.planning/notes/260615-oper-log-coverage-audit.md:194-200` — 旧中间件 4 大缺陷，禁止在 helper 层做路径过滤的隐式语义

### 调研笔记
- `.planning/notes/260616-rpa-heartbeat-log-flooding.md` — 完整调研结果，含 handler 定位、operlog 能力边界、四种切入点对比

### 实施待办
- `.planning/todos/pending/operlog-exclude-paths.md` — 6 步实施清单 + 风险 + 验收标准

</canonical_refs>

<specifics>

## Specific Ideas

- **匹配函数签名建议**：
  ```go
  // Exported for testability and debug introspection
  var ExcludedPaths []string

  // Configure replaces the package-level exclude list. Call once at startup.
  func Configure(paths []string)

  // IsExcludedPath returns true if path matches any pattern in ExcludedPaths
  // using filepath.Match + /* suffix wildcard (matches security.request_encryption.exclude_paths semantics).
  func IsExcludedPath(path string) bool
  ```

- **Record 函数体首行插入位置**（伪代码）：
  ```go
  func Record(c *gin.Context, operLogSvc Recorder, db *gorm.DB, module string, operType int, opts ...RecordOption) {
      defer func() { recover() }()  // 保留 panic 防护
      if IsExcludedPath(c.Request.URL.Path) {
          return  // NEW: 排除路径早退
      }
      // ... 现有逻辑
  }
  ```

- **RecordWithBody 必须在 GetRawData 之前**：因为 RecordWithBody 内部消费 request body 来脱敏，如果路径排除，必须在 `c.GetRawData()` 之前早退，否则 body 已消费但未记录会污染下游

- **测试断言模板**：
  ```go
  func TestExcludedPathsEarlyReturn(t *testing.T) {
      Configure([]string{"/api/v1/rpa/workers/*/heartbeat"})
      defer Configure(nil)  // cleanup

      // mock Recorder 捕获 RecordAsync 调用
      // 1. POST /api/v1/rpa/workers/w-001/heartbeat → RecordAsync not called
      // 2. POST /api/v1/rpa/workers/register → RecordAsync called
      // 3. POST /api/v1/rpa/workers/progress → RecordAsync called
  }
  ```

</specifics>

<deferred>

## Deferred Ideas

- **DEF-01**: rpaApi.ts 路径错配修复（前端 `/${id}/progress` 与后端 `/progress` 不匹配；`/rpa/workers/online`、`/rpa/workers/:id/statistics`、`/rpa/workers/:id/offline`、`/rpa/workers/:id/restart` 在 rpa_router.go 中**没有对应路由**）
- **DEF-02**: `/rpa/workers/autoscale/config` (POST) 无持久化（handler 内 `// TODO: 保存配置到数据库`）
- **DEF-03**: operlog 运行时热改 `Refresh()` API（待运维需求出现）
- **DEF-04**: progress 端点是否加入排除（待业务确认审计价值；当前默认保留）

</deferred>

---

*Phase: 35-operlog-exclude-paths*
*Context gathered: 2026-06-16 via Explore Session (gsd:explore)*
