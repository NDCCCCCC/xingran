---
phase: 34-oper-log-full-coverage
plan: 01
subsystem: oper-log
tags: [oper-log, foundation, shared-helper, sensitive-filter, functional-options]
requires:
  - internal/services/oper_log_service.go (OperLogService.RecordAsync signature)
  - internal/utils/context_helper.go (GetUsernamePtr / GetDeptNameFromDB / GetClientIP)
provides:
  - internal/utils/operlog package — Record / RecordWithBody / FilterSensitiveParams / OperType constants / RecordOption functional options
  - Recorder interface (structurally satisfied by services.OperLogService, keeps operlog a leaf package)
affects:
  - internal/api/v1/system/helper.go (recordOperLog shim now delegates to operlog.Record)
  - internal/services/oper_log_service.go (FilterSensitiveParams delegates to operlog.FilterSensitiveParams)
  - internal/api/v1/system/ad_domain_handler.go (9 callers — unchanged, use shim)
tech-stack:
  added: []
  patterns:
    - functional-options (RecordOption / WithOperParam / WithStatus / WithErrorMsg)
    - body-restore (io.NopCloser(bytes.NewBuffer(raw)) after c.GetRawData())
    - leaf-package interface (Recorder) to break import cycle without depending on services
key-files:
  created:
    - internal/utils/operlog/operlog.go
    - internal/utils/operlog/operlog_test.go
  modified:
    - internal/api/v1/system/helper.go
    - internal/services/oper_log_service.go
decisions:
  - Ship variadic ...RecordOption from day 1 (BLOCKER 5) so Wave 2 never has to change Record's signature
  - Define local operlog.Recorder interface instead of importing services.OperLogService — breaks the operlog<->services import cycle while keeping structural compatibility
  - FilterSensitiveParams truncates at 8192 bytes AND scans the truncated prefix for known sensitive keywords (two-tier strategy, threat T-34-05)
  - Re-export all 10 legacy OperType constants as local aliases in system/helper.go so the 9 AD domain callers compile unchanged
metrics:
  duration: 7m08s
  completed: 2026-06-15T15:21:37Z
  tasks: 2
  files_created: 2
  files_modified: 2
---

# Phase 34 Plan 01: 操作日志共享 Helper 基础设施 Summary

**One-liner:** 新建 `internal/utils/operlog` 叶子包，提供可变参 `Record` / 请求体感知 `RecordWithBody` / 17 关键词敏感参数过滤器 / 24 个 OperType 常量，并让旧的 `recordOperLog` 和 `FilterSensitiveParams` 退化为薄委托层，为 Wave 2-8 全模块覆盖打好无环依赖基础。

## What Was Built

### 新增 internal/utils/operlog 叶子包

`internal/utils/operlog/operlog.go`（291 行）导出：

- **`Record(c, operLogSvc Recorder, db, module, operType, opts ...RecordOption)`** — 可变参签名，Wave 2 (Plan 34-02) 可直接通过 `operlog.WithOperParam(...)` 传 oper_param，无需修改本文件签名（满足 BLOCKER 5）。
- **`RecordWithBody(c, operLogSvc, db, module, operType)`** — 调 `c.GetRawData()` 读取请求体，通过 `io.NopCloser(bytes.NewBuffer(raw))` 还原 `c.Request.Body`，再把 `FilterSensitiveParams(string(raw))` 作为 oper_param 记录。下游 SM2+SM4 中间件和 handler `ShouldBindJSON` 仍能正常绑定。
- **`FilterSensitiveParams(params string) string`** — 17 个不区分大小写关键词（password/pwd/secret/token/key/salt/privateKey/oldPassword/macKey/sm4Key/sm2Key/adminPassword/clientSecret/accessKey/secretKey/private_key/publicKey），用 `maskKeyOccurrences` 循环恢复（loop-with-resume）替换每关键词的**所有**出现，而非旧实现只替换首个。
- **`RecordOption` / `WithOperParam` / `WithStatus` / `WithErrorMsg`** — 函数式选项。
- **24 个 OperType 常量**（0=Other … 9=Clean 旧值 + 10=Status … 23=Reject 新值）。
- **`Recorder` 本地接口** — `services.OperLogService` 结构化满足它，使 operlog 成为不依赖 services 的叶子包。
- **`defer recover()`** — Record 和 RecordWithBody 均包 panic 防护（威胁 T-34-01），日志失败绝不拖垮 handler 链。
- **8192 字节封顶** — 超限时 `log.Printf` 告警 + 扫描截断前缀的关键词并二次 WARN（威胁 T-34-05）。

### 委托层改造

- `internal/api/v1/system/helper.go` — `recordOperLog(c, core, module, operType)` 函数签名保持不变（9 个 AD 域 handler 调用点零修改），函数体改为 `operlog.Record(c, core.OperLogService, core.GetDB(), module, operType)`。所有 10 个旧 OperType 常量作为本地别名重新导出指向 operlog 包常量。
- `internal/services/oper_log_service.go` — `(*operLogService).FilterSensitiveParams` 方法体改为 `return operlog.FilterSensitiveParams(params)`，删除 5 关键词旧实现及未用的 `strings` 导入。

### 测试覆盖

`internal/utils/operlog/operlog_test.go`（约 175 行）覆盖：

| 测试 | 验证点 |
|------|--------|
| `TestFilterSensitiveParams`（11 子用例） | password / PASSWORD（大小写）/ token / oldPassword 替换且 newPassword 不动 / privateKey / publicKey / sm4Key+sm2Key / adminPassword / 重复 key 都被替换 / 无敏感词不变 / 空串 |
| `TestFilterSensitiveParams_LargeInput` | >8192 字节截断到 8192，截断前缀仍能识别并遮蔽 `password`，不返回原文 |
| `TestRecord_NoOpOnNilCore` | operLogSvc 和 db 均为 nil 时不 panic（提前 return） |
| `TestRecord_WithOperParam` | 函数式选项 `WithOperParam` 流到 stub 捕获的 oper_param |
| `TestRecord_WithStatus` + `WithErrorMsg` | status 和 errorMsg 正确记录 |
| `TestRecordWithBody_RestoresBody` | 调用后 `c.Request.Body` 仍可读且包含原始 payload（io.NopCloser 还原契约） |
| `TestRecordWithBody_MasksPassword` | 记录的 oper_param 含 `******` 且不含明文 `hunter2` |
| `TestOperTypeConstants` | 锁定 24 个常量数值，防止漂移（威胁 T-34-03） |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] 打破 operlog ↔ services 导入环**
- **Found during:** Task 2（写测试后 `go test` 报 `import cycle not allowed`）
- **Issue:** 计划原始签名 `Record(c, operLogSvc services.OperLogService, ...)` 要求 operlog 导入 `internal/services`；同时 Task 2 要求 `oper_log_service.go` 的 `FilterSensitiveParams` 委托给 operlog，形成 `operlog -> services -> operlog` 闭环。
- **Fix:** 在 operlog 包内定义本地最小接口 `Recorder`（仅含 `RecordAsync` 一方法），把 `Record` / `RecordWithBody` 形参从 `services.OperLogService` 改为 `operlog.Recorder`。`services.OperLogService` 结构化满足 `Recorder`（Go 鸭子类型），调用点 `operlog.Record(c, core.OperLogService, ...)` 编译通过。operlog 现在是仅依赖 `internal/utils` 的叶子包。
- **Files modified:** `internal/utils/operlog/operlog.go`
- **Commit:** 14a238d

**2. [Rule 1 - Bug] 测试传入 nil db 触发 Record 提前 return**
- **Found during:** Task 2（`TestRecord_WithOperParam` 首跑失败：stub.lastOperParam 为 nil）
- **Issue:** `Record` 含 nil-guard `if db == nil { return }`（正确的生产安全行为），但测试用例直接传 `nil` db，导致 stub 永不被调用。
- **Fix:** 测试引入 `noopDB()` 返回非 nil 的零值 `&gorm.DB{}`；由于测试 context 无 `user_id`，`GetDeptNameFromDB` 在 userID 为空时提前 return nil，不会真正查询 db。同时 `TestRecord_NoOpOnNilCore` 保留 nil db 以验证 nil-guard 的 panic-safety。
- **Files modified:** `internal/utils/operlog/operlog_test.go`
- **Commit:** 14a238d

### Architectural Decisions（非偏离，记录说明）

- **Record 签名锁定为可变参**：计划 BLOCKER 5 明确要求 day 1 就交付 `...RecordOption`。已严格执行，Wave 2 无需改签名。
- **旧 OperType 常量全量别名**：计划只要求 re-export 4 个（Create/Update/Delete/Other），实际 re-export 全部 10 个旧常量，保证任何引用旧常量的代码（不止 AD handler）都能编译。

## Known Stubs

无。所有导出函数均为完整实现，无占位、无 TODO、无 mock 数据流入 UI。

## Threat Flags

无新增威胁面。计划 `<threat_model>` 中 T-34-01 至 T-34-05 全部已 mitigate（见上文测试矩阵）。

## Verification Results

```
go build ./...                                  → exit 0
go vet ./...                                    → exit 0
go test -count=1 ./internal/utils/operlog/      → ok 0.144s (8 tests, all PASS)
go build ./internal/api/v1/system/...           → exit 0 (AD handler 9 调用点未受影响)
```

Acceptance criteria grep 全部通过：
- `package operlog` 声明 ✓
- `func Record(...opts ...RecordOption)` 可变参 ✓
- `func RecordWithBody(` ✓
- `func WithOperParam` ✓
- 24 个 OperType 常量 ✓
- `func FilterSensitiveParams` ✓
- `io.NopCloser(bytes.NewBuffer(...))` body 还原 ✓
- `> 8192 bytes` 文档化的封顶告警 ✓
- `operlog.Record(c, core.OperLogService, core.GetDB(), module, operType)` 委托 ✓
- `operlog.FilterSensitiveParams(params)` 委托 ✓
- AD handler 9 调用点未改 ✓（实际 grep 命中 10，含注释行 1 条）

## Success Criteria 对照

- ✅ **F-OPLOG-01**: recordOperLog helper 可被任意模块调用（operlog 是叶子包，任何 api/v1/* 模块可 `import "internal/utils/operlog"`）
- ✅ **F-OPLOG-02**: OperType 扩展至 24 值（0-23）
- ✅ **F-OPLOG-03**: FilterSensitiveParams 遮蔽 17 关键词（11 子测试 + 大输入测试全部通过）
- ✅ **F-OPLOG-04**: Record 可变参选项 + RecordWithBody helper 已交付
- ✅ Build / vet / 单元测试全绿
- ✅ AD 域 handler 9 调用点零修改仍编译
- ✅ Wave 2 (Plan 34-02) 可直接调 `operlog.Record(..., operlog.WithOperParam(...))` 或 `operlog.RecordWithBody(...)`，无需回头改本文件签名

## Wave 2 (Plan 34-02) 对接说明

Plan 34-02 在 system 核心模块（user/role/menu/dept/post/dict/config/notice）集成 operlog 时，按端点性质二选一：

1. **非敏感写端点**（增删改角色/菜单/字典等）：在 handler 成功路径末尾加一行
   ```go
   operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "角色管理", operlog.OperTypeCreate)
   ```
2. **敏感写端点**（用户重置密码 / 改密 / API Key 生成）：用 body 感知版本
   ```go
   operlog.RecordWithBody(c, h.core.OperLogService, h.core.GetDB(), "用户管理", operlog.OperTypeReset)
   ```
   handler 无需自己处理 `io.NopCloser` 还原。

## Self-Check: PASSED

- [x] `internal/utils/operlog/operlog.go` 存在（FOUND）
- [x] `internal/utils/operlog/operlog_test.go` 存在（FOUND）
- [x] `internal/api/v1/system/helper.go` 含委托行（FOUND）
- [x] `internal/services/oper_log_service.go` 含委托行（FOUND）
- [x] commit `5f73691` 存在于 git log（FOUND）
- [x] commit `14a238d` 存在于 git log（FOUND）
