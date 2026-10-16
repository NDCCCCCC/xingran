# Phase 35 Plan 01: operlog.exclude_paths — Summary

**Executed:** 2026-06-16
**Plan:** `.planning/phases/35-operlog-exclude-paths/35-01-PLAN.md`
**Wave:** 1
**Status:** ✅ COMPLETE — all 5 tasks shipped, all gates green

---

## What Shipped

配置驱动的操作日志排除路径机制 — 让 `sys_oper_log` 摆脱 RPA Worker 心跳（30s/Worker，~120N rows/h）的低价值审计写入,同时为后续运维热改（健康检查、监控推送等）提供单一可配置入口。

### 1. 配置层（2 个文件）
- **`configs/config.yaml`** — 新增顶层 `operlog:` 块,默认 `exclude_paths: ["/api/v1/rpa/workers/*/heartbeat"]`
- **`configs/config.dev.yaml`** — 镜像 schema,显式 `exclude_paths: []`(dev 保留心跳审计便于本地排查 Agent 状态)

### 2. Config 结构（1 个文件）
- **`internal/config/config.go`** — 新增 `OperLogConfig` struct(含 `ExcludePaths []string` 字段 + `mapstructure:"operlog"` tag),挂载到顶层 `Config` 结构

### 3. operlog helper 层（1 个文件）
- **`internal/utils/operlog/operlog.go`**:
  - 新增 `path/filepath` import
  - 新增包级 `var ExcludedPaths []string`(导出便于调试期 introspect)
  - 新增 `func Configure(paths []string)`(启动期单次调用,无 mutex)
  - 新增 `func IsExcludedPath(path string) bool`(语义对齐 `pkg/middleware/request_decryption.go:294-315` 的 `isExcludedPath`,支持 `filepath.Match` + `/*` 后缀通配)
  - 在 `Record` 函数体首行(`defer recover()` 之后,`c == nil` 检查之前)插入路径早退
  - 在 `RecordWithBody` 函数体首行、**`c.GetRawData()` 之前**插入路径早退(关键 — 避免 body 被消费但未记录)

### 4. 启动期 wiring（1 个文件）
- **`internal/core/core.go`**:
  - 新增 `internal/utils/operlog` import
  - `New()` 函数内、在 `return &Core{...}` 之前调一次 `operlog.Configure(cfg.OperLog.ExcludePaths)`

### 5. 测试（2 个文件）
- **`internal/utils/operlog/operlog_test.go`** — 新增 `TestIsExcludedPath` 表驱动测试,12 个子测试覆盖 exact match / `/*` single-segment wildcard / no-match / `/list` 误伤防护 / 空列表 / 多模式 OR 语义 / 大小写敏感 / `/*` 不跨 `/` 等关键语义
- **`internal/utils/operlog/regression_test.go`** — 新增 `TestExcludedPathsEarlyReturn`:
  - Phase A: `Record` 在 heartbeat 路径早退(0 调用),register/progress 路径仍调用 RecordAsync(2 调用)
  - Phase B: `RecordWithBody` 在 heartbeat 路径**完全跳过 `GetRawData`**(调用后 `io.ReadAll` 仍可读完整 body),register 路径通过 masking+restore 正常记录

---

## Verification Results

| Gate | Result |
|------|--------|
| `go build ./...` | ✅ exit 0 |
| `go vet ./...` | ✅ exit 0 |
| `go test -count=1 ./internal/utils/operlog/...` | ✅ exit 0 — 全部测试通过 |
| `TestIsExcludedPath` (12 子测试) | ✅ all PASS |
| `TestExcludedPathsEarlyReturn` (2 子测试) | ✅ all PASS |
| `TestOperTypeConstantStability` | ✅ PASS — 25 OperType 常量值不变 |
| `TestOperTypeCountEquals25` | ✅ PASS — 常量数 = 25 |
| `TestRecordSignatureStable` | ✅ PASS — Record 5 fixed + 1 variadic;RecordWithBody 5 fixed |
| `TestFilterSensitiveParamsKeywordsStable` | ✅ PASS — 34 敏感关键词未丢失 |
| `grep "^operlog:" configs/config.yaml` | ✅ match (line 118) |
| `grep "^operlog:" configs/config.dev.yaml` | ✅ match (line 50) |
| `grep "operlog\.Configure(cfg\.OperLog\.ExcludePaths)" internal/core/core.go` | ✅ match (line 103) |
| `grep "var ExcludedPaths \[\]string" internal/utils/operlog/operlog.go` | ✅ match (line 87) |
| `grep "type OperLogConfig struct" internal/config/config.go` | ✅ match (line 124) |
| `grep "func IsExcludedPath(" internal/utils/operlog/operlog.go` | ✅ match (line 106) |
| `grep "func TestExcludedPathsEarlyReturn" internal/utils/operlog/regression_test.go` | ✅ match (line 306) |
| `grep "func TestIsExcludedPath" internal/utils/operlog/operlog_test.go` | ✅ match (line 254) |

### 行为观察(待部署后人工 SQL 验证 — OPERLOG-06 残留项)
部署后启动后端 + 一个 rpa-worker Agent 连续运行 5 分钟,执行:
```sql
SELECT COUNT(*) FROM sys_oper_log
WHERE oper_url LIKE '%heartbeat%'
  AND oper_time > NOW() - INTERVAL '5 minutes';
```
预期返回 0(prod 默认配置),或非零(dev 默认空 exclude_paths 保留审计)。

---

## Files Modified

| File | Change Type | LOC delta |
|------|-------------|-----------|
| `configs/config.yaml` | add `operlog:` block | +8 |
| `configs/config.dev.yaml` | add `operlog:` block (empty) | +7 |
| `internal/config/config.go` | add `OperLogConfig` struct + `OperLog` field | +10 |
| `internal/core/core.go` | add import + 1 Configure call | +7 |
| `internal/utils/operlog/operlog.go` | add `path/filepath` import + `ExcludedPaths` + `Configure` + `IsExcludedPath` + 2 early-returns | +58 |
| `internal/utils/operlog/operlog_test.go` | add `TestIsExcludedPath` (12 cases) | +104 |
| `internal/utils/operlog/regression_test.go` | add 4 imports + `excludedMockRecorder` struct + `TestExcludedPathsEarlyReturn` (2 subtests) | +104 |

**Total: 7 files, +298 LOC, 0 deletions**

---

## Phase 34 Contract Preservation (OPERLOG-03)

所有 Phase 34 公共 API 锁定项保持不变:
- ✅ `Record` 签名:5 fixed + 1 variadic(`RecordOption`)(`TestRecordSignatureStable` 验证)
- ✅ `RecordWithBody` 签名:5 fixed 非 variadic
- ✅ 25 OperType 常量值不变(`TestOperTypeConstantStability` + `TestOperTypeCountEquals25` 验证)
- ✅ 34 敏感关键词不变(`TestFilterSensitiveParamsKeywordsStable` 验证)
- ✅ `FilterSensitiveParams` 函数体未改动
- ✅ handler 层不需任何修改 — 显式 `operlog.Record(...)` 调用风格保留

---

## Decisions Locked (per D-01..D-06 in 35-CONTEXT.md)

| ID | Decision | Implementation |
|----|----------|----------------|
| D-01 | `operlog.exclude_paths` 顶层 YAML,默认 heartbeat-only | ✅ `configs/config.yaml` 仅含 heartbeat;progress/register 不在排除列表 |
| D-02 | helper 层首行判定,handler 层不动 | ✅ `Record`/`RecordWithBody` 首行早退,无 handler 改动 |
| D-03 | `filepath.Match` + `/*` 后缀通配,与 `pkg/middleware/request_decryption.go:294-315` 一致 | ✅ `IsExcludedPath` 实现完全镜像蓝图函数语义 |
| D-04 | `core.New()` 单次 `Configure` 调用,无 mutex | ✅ `core.go:103` 启动期单次,无并发问题 |
| D-05 | Phase 34 API 表面零破坏 | ✅ 见上节 contract preservation |
| D-06 | `TestExcludedPathsEarlyReturn` + `TestIsExcludedPath` 覆盖 | ✅ 12 + 2 子测试全 PASS |

---

## Threat Model Mitigation Status

| Threat | Disposition | Verified By |
|--------|-------------|-------------|
| T-35-01 (signature drift) | mitigate | Phase 34 regression tests (4/4 PASS) |
| T-35-02 (unintended path filter) | mitigate | `TestIsExcludedPath` `/list mis-fire protection` 子测试 |
| T-35-03 (silent body consumption) | mitigate | `TestExcludedPathsEarlyReturn` Phase B `io.ReadAll` 断言 |
| T-35-04 (mis-config) | mitigate | config.dev.yaml 默认空数组;prod 仅 1 条目 |
| T-35-05 (race on startup) | mitigate | 单次启动期调用;tests 用 `defer Configure(nil)` 重置 |
| T-35-SC (no new external deps) | accept | 仅 stdlib `path/filepath` + `strings` |

---

## Residual Items (out-of-scope, deferred per 35-CONTEXT.md)

- **DEF-01**: rpaApi.ts 路径错配修复(前端 `/${id}/progress` 与后端 `/progress` 不匹配;4 个 rpa_router.go 中无对应路由的端点)
- **DEF-02**: `/rpa/workers/autoscale/config` (POST) handler 内 `// TODO: 保存配置到数据库`
- **DEF-03**: operlog 运行时热改 `Refresh()` API(待运维需求出现)
- **DEF-04**: `/api/v1/rpa/workers/progress` 是否加入排除(待业务确认审计价值)
- **OPERLOG-06 残留**: 部署后 5 分钟 live SQL `COUNT(*)` 验证(需生产环境 + rpa-worker Agent)

---

## Next Step

Phase 35 单 plan 5 任务全部完成。考虑:
1. 提交 — `git add` 7 个修改文件 + 本 SUMMARY,commit `feat(35): operlog.exclude_paths 配置驱动白名单`
2. 手动验证 OPERLOG-06 live SQL 门(部署后 5 分钟心跳观察)
3. 决定 v1.16 范围(STATE.md 候选:Triage 103 audit items / VDI 22C-22D / Phase 34 operlog CI gate)

Run `/gsd:ship` 创建 PR,或 `/gsd:new-milestone` 启动 v1.16。
