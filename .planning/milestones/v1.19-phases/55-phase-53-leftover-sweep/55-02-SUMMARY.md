---
phase: 55
plan: 02
title: 后端 batch_orchestrator.go fallback 路径 port 归属跨层防御 (CR-02)
subsystem: portwrite
tags: [tech-debt, security, defense-in-depth, portwrite]
dependency_graph:
  requires: [phase-53-port-write-foundation, phase-53-w4-frontend-fixes]
  provides: [cr02-backend-fallback-defense]
  affects: [internal/services/portwrite/batch_orchestrator.go]
tech-stack:
  added: []
  patterns: [cross-layer-defense, fail-closed-on-mismatch]
key-files:
  created: []
  modified:
    - internal/services/portwrite/batch_orchestrator.go
decisions:
  - D-04: 仅 fallback 分支加校验,不做全路径(正常路径已有 WHERE device_id = ? AND id IN ? 隔离零开销)
  - D-04: error message 文案固定为 "port does not belong to device"(53-REVIEW §113 原文)
  - Plan+Context deviation: PortResult struct 无 PortName 字段,移除可选的 PortName 字段填入(CLAUDE.md Scope Constrainment + D-04 严格限制只改 batch_orchestrator.go 一处)
  - DB 查询失败(端口已软删/物理删)归 Failed + continue (不 break) — 数据校验错误不影响后续端口
  - 归属校验失败 continue (不 break) — 与 SSH 失败的 fail-fast 语义区分
metrics:
  duration: ~5 min
  completed_date: 2026-07-08
  tasks_completed: 2
  files_modified: 1
  tests_passing: 35 (portwrite package)
---

# Phase 55 Plan 02: 后端 batch_orchestrator.go fallback 路径 port 归属跨层防御 (CR-02) Summary

后端 fallback 路径加 1 次 DB 查询校验 port 真实 deviceID,与 `req.DeviceID` 不一致则归 `result.Failed` + error `"port does not belong to device"` 不调 SSH。纵深防御双保险(前端 CR-01 根因 Phase 53 已修,本计划补后端兜底)。

## One-liner

Fallback 路径前置 port 归属 DB 校验,deviceId 不匹配时拒绝 SSH 下发并归 Failed,纵深防御 CR-01 修复后仍存在的跨设备写入风险。

## Tasks Completed

| Task | Name                                              | Commit   | Files Modified |
| ---- | ------------------------------------------------- | -------- | -------------- |
| 1    | CR-02 fallback 分支前置 port 归属校验             | 4f357dae | 1 (batch_orchestrator.go) |
| 2    | 验证 — go build + go test 全绿                    | (1 内含) | 0              |

Task 2 was bundled into Task 1's verification flow: `go build ./...` exit 0 and `go test ./internal/services/portwrite/...` all 35 tests PASS (TestE2E_*, TestParseConfigError, TestShutdown_*, TestUndoShutdown_*, TestSetDescription_*, TestEnableDot1x_*, TestDisableDot1x_*, TestBatchWritePorts_*).

## What Was Built

### Core Change — batch_orchestrator.go line 72-95 (fallback 分支)

```go
if !exists {
    // CR-02 跨层防御(Phase 55 leftover sweep):
    // fallback 路径下前端 req.DeviceID 可能错位(CR-01 根因已修但兜底双保险),
    // 先查 port 真实 deviceID,与 req.DeviceID 不一致则归 Failed 不调 SSH。
    // 正常路径已有 preStateMap 的 WHERE device_id = ? AND id IN ? 隔离零开销。
    var actualPort models.DevicePortStatus
    if err := s.db.WithContext(ctx).
        Where("id = ?", portID).
        First(&actualPort).Error; err != nil {
        // DB 查不到该 port(可能已软删/物理删)— 归 Failed,继续校验下一个
        result.Failed = append(result.Failed, PortResult{
            PortID: portID,
            Error:  fmt.Sprintf("port not found: %v", err),
        })
        continue
    }
    if actualPort.DeviceID != req.DeviceID {
        // port 真实 deviceID 与请求 deviceID 不一致 — 拒绝 SSH 下发
        result.Failed = append(result.Failed, PortResult{
            PortID: portID,
            Error:  "port does not belong to device",
        })
        continue
    }

    // 校验通过 — 端口"消失"(D-13 fallback)但归属正确,直接下发
    writeResult, werr := s.executeWrite(ctx, portID, req.DeviceID, req.Action, req.Description, operator, "")
    ...
}
```

### Key Properties

1. **零主路径开销**: 仅在 `preStateMap[portID]` 未命中(`!exists`)时新增 1 次 `db.First(&port, "id = ?")` 查询,正常路径(已由 `WHERE device_id = ? AND id IN ?` 隔离)零开销。
2. **拒绝 SSH 调用**: 校验失败时 `continue` 到下一个 portID,完全不调 `s.executeWrite(...)`(纵深防御核心)。
3. **Fail-closed on mismatch**: `actualPort.DeviceID != req.DeviceID` 立即归 Failed,无 try-retry,无 side-effect。
4. **DB 错误非传染**: 端口已软删/物理删(DB First 返 err)归 Failed + `continue`,不阻断 batch 中其他端口校验。
5. **校验失败 continue(不 break)**: 数据校验错误与 SSH 传输错误语义区分 — SSH 失败 fail-fast,数据错误继续。
6. **Error message 固定**: `"port does not belong to device"` 严格按 53-REVIEW §113 原文,前端 ResultView `dataIndex: "error"` 会直接渲染此值。

## Deviations from Plan

### Auto-fixed Issues

**None** — 仅 1 处 plan vs 实际代码不符的字段调整(见下)。

### Plan vs Reality Adjustments

**1. [D-04 + Scope Constrainment] 移除 PortResult.PortName 字段填入**

- **Found during:** Task 1 实施 (第一次 `go build`)
- **Issue:** PLAN.md line 55 写 `PortName: actualPort.InterfaceName`(可选,便于审计追踪),但实际 `PortResult` struct(`port_write_service.go:34-42`)无 `PortName` 字段,只有 `PortID / Action / Status / NoOp / CurrentState / Error / CommandSent`。
- **Fix:** 移除 `PortName` 字段,只保留 `PortID + Error`。审计追踪可通过 portID 反查 DB 获得 InterfaceName,不影响 CR-02 防御效果。
- **Files modified:** batch_orchestrator.go (一处 struct literal 字段调整)
- **Decision rationale:**
  - CONTEXT.md D-04 明确 "仅 fallback 分支加校验,不做全路径"(全路径 = 改 PortResult struct 算扩范围)
  - CLAUDE.md "Scope Constrainment" 规则: 不主动改报告之外的代码
  - PLAN.md 自己也标注 "可选,便于审计追踪" — 非 acceptance_criteria 必须项
- **Commit:** 4f357dae (同一提交内调整)

## Decisions

| ID    | Decision                                                              | Source                  |
| ----- | --------------------------------------------------------------------- | ----------------------- |
| D-04  | 仅 fallback 分支加校验,error message = "port does not belong to device" | CONTEXT.md §CR-02       |
| (plan deviated) | 移除 PortResult.PortName 字段(原结构无此字段) | Scope Constrainment     |
| (impl) | DB 查询失败归 Failed + continue(不 break)                            | Plan task §1 数据校验语义 |
| (impl) | 归属校验失败 continue(不 break)                                       | Plan task §1 数据 vs SSH 语义区分 |

## Threat Surface

| Flag | File | Description |
|------|------|-------------|
| (无新增) | internal/services/portwrite/batch_orchestrator.go | 本计划加固既有 fallback 路径,无新增 endpoint / 无新增 auth path / 无 schema 改动 |

## Files Modified

```
internal/services/portwrite/batch_orchestrator.go  | +24 -1
  - 新增 fallback 分支实际 port 查询 + deviceID 归属校验
  - DB 查询失败/归属不匹配均归 result.Failed + continue(不 break)
  - 校验通过后才调 s.executeWrite(SSH 下发)
```

## Verification

- `go build ./...` exit 0 — 整个项目编译通过
- `go test ./internal/services/portwrite/... -v` 全部 PASS(35 个测试)
  - 关键回归测试: TestBatchWritePorts_Success_AllSucceeded / TestBatchWritePorts_FailFast_Transport / TestBatchWritePorts_FailFast_DeviceRejected / TestBatchWritePorts_ExceedsExactly50 / TestE2E_Batch_Huawei_HappyPath / TestE2E_Batch_FailFast 全部通过
  - 35 个 SELECT `record not found` 日志均为测试 fixture 故意构造的缺记录场景(正常行为),非测试失败

## Acceptance Criteria Status

- [x] `batch_orchestrator.go` fallback 分支(`if !exists` 块)新增 `actualPort` 查询 + 归属校验逻辑
- [x] `actualPort.DeviceID != req.DeviceID` 时归 `result.Failed` + `Error: "port does not belong to device"`,**不调 SSH**
- [x] DB 查询失败(端口已删)归 `result.Failed` + `continue`(不 break)
- [x] 归属校验失败用 `continue`(不 break),让 batch 中其他 port 继续校验
- [x] error message 字符串严格为 `"port does not belong to device"`
- [x] `go build ./...` exit 0
- [x] `go test ./internal/services/portwrite/...` 全部 PASS

## Self-Check: PASSED

**1. Files created/modified exist:**

```bash
[ -f "internal/services/portwrite/batch_orchestrator.go" ] && echo "FOUND: internal/services/portwrite/batch_orchestrator.go"
```

Result: FOUND (modified in place).

**2. Commits exist:**

```bash
git log --oneline | grep -q "4f357dae" && echo "FOUND: 4f357dae"
```

Result: FOUND (`fix(55-02): 后端 batch_orchestrator fallback 路径 port 归属校验(CR-02 防御)`).

**3. Verification commands passed:**

```bash
go build ./...                                   # exit 0
go test ./internal/services/portwrite/...        # 35 PASS, ok 3.494s
```

All claims verified.

## Notes

- **Scope 严格**: 仅改 `batch_orchestrator.go` 一处,不动主路径(已有 `WHERE device_id = ? AND id IN ?` 隔离)、不动 `PortResult` struct、不动测试文件。
- **跨层防御纵深**: 前端 CR-01(Phase 53 commit 9b01cc68 已修) + 后端 CR-02(本计划)双保险。即使前端未来再次出现 deviceId 错位,后端兜底能拦截。
- **性能影响**: 仅 fallback 罕见路径多 1 次 DB 查询;正常路径(preStateMap 命中)零开销。
- **未来 follow-up**: 全路径归属校验(`for portID in PortIDs` 之外不再区分 fallback)可作为未来更彻底的加固选项,留待 `audit-open` 评估 — 当前 D-04 已锁定范围。
