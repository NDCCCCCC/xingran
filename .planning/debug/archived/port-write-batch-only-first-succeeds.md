---
slug: port-write-batch-only-first-succeeds
status: resolved
trigger: "用户 2026-07-08 真机测试：批量执行命令，只有第一个端口成功，后面的端口都没有执行成功"
created: 2026-07-08
resolved: 2026-07-08
related:
  - port-write-shutdown-multi-layer-bug (已 resolved 同日)
  - conn-pool-full-on-excess-devices (设备连接池)
---

# Debug Session: port-write-batch-only-first-succeeds

## 症状

- **Expected behavior:** 批量端口写操作（同一设备，多个端口）应该全部成功执行
- **Actual behavior:**
  1. 用户在 BulkWriteDrawer 选 3-5 个端口（如 GE4/18、GE4/19、GE4/20）
  2. 提交批量关闭/启用
  3. **只有第一个端口** successed
  4. 后续端口都 failed（result.Failed 列表里有 Succeeded=0, Failed=N-1）
- **Reproduction:** 端口管理页 → 批量操作 → 选 ≥2 端口 → 提交
- **关键：** 单端口命令完全正常（Bug #1 的 4 层修复已让单端口端到端工作）

## 怀疑方向（已通过 Phase 1 排除）

- **假设 A**（detached ctx 被 cancel）：被 `TestBatchRepro_Port2CtxState` 排除（3 端口全看到 `ctx.Err()=nil` + `deadlineLeft=~30min`）
- **假设 B**（SSH pool 满）：被"同一设备多端口应复用同一连接"逻辑排除
- **假设 C**（task_id 冲突）：`generateTaskID` 用 `time.Now().UnixNano()`，但 task_id 只是日志 label，Submit/worker 都不做去重
- **假设 D**（detached ctx 早 cancel）：`BatchWritePorts` 用 `context.Background()` + 30min timeout，与 caller ctx 无关
- **假设 E**（scrapli wrapper state 污染 prompt 匹配）：scrapli 内部用 joined prompt pattern（OR 所有 priv pattern），`configuration` 含空格模式 `(?im)^[\w.\-@/:]{1,63}\([\+\w.\-@/:+ ]{0,64}\)#$` 已匹配 `(config-if-GigabitEthernet 4/18)#`（参见 patched yaml 修复），单端口→batch 第 1 端口后状态保留 OK

## 根因（最终确定）

**单端口写模板 cmds = `["interface <X>", <action>]` 不含 `quit`，每个端口写完后设备留在 `(config-if-<X>)#` 子视图。**

Batch 路径下，下一个端口的 scrapli `AcquirePriv("configuration")` 判定当前已在 configuration priv → `noAction` → 直接 `SendCommands(["interface <Y>", ...])`。

不同厂商对"嵌套 interface 视图"行为不一：
- **华为/H3C**：允许嵌套（同 Cisco IOS），新 interface view 替换旧的 → port 2+ 正常
- **锐捷 RGOS（实测）**：**拒绝**嵌套 entry（设备返回 `% Error: ...`）

锐捷拒绝后，service 端 `parseConfigError` 把 `resp.Result` 中的 `% Error:` 命中 `rejectionMarkers`（vendor-independent transport error map），归类为 `WriteErrorDeviceRejected` → batch fail-fast `break` → 后续端口不进任何数组 → 用户看到 "第 1 个成功，其余 N-1 失败"。

## 修复

在 `BatchWritePorts` 串行循环里，每个非末尾端口的 `executeWrite` 成功后，插入一条独立的 `quit` cleanup cmd 让设备从 `(config-if-<X>)#` 退到 `(config)#`，下一个端口的 `interface <Y>` 才能正确进入新子视图。

- 新增 `sendInterfaceExitCleanup(ctx, deviceID, portIdx, totalPorts)` 辅助方法
- 调用点：主循环里 `succeeded` 分支末尾（fail-fast break 的分支不需要 — 下一次循环不会再跑）
- 末尾端口不调 cleanup（设备停在 interface view 不影响下次会话；新 SSH session 从头开始）
- cleanup 失败仅日志，不影响当前端口的 `PortResult` 与 fail-fast 语义
  - cleanup 失败 = SSH 通道异常 → 下一个端口的 `executeWrite` 自然也会失败
  - 显式 break 不必要：cleanup 是 best-effort 状态卫生，不是业务逻辑

### 代码要点

```go
func (s *portWriteServiceImpl) sendInterfaceExitCleanup(
    ctx context.Context, deviceID string, portIdx, totalPorts int,
) {
    if portIdx >= totalPorts-1 { return }  // 末尾端口不需要
    cleanupErr := s.deviceExecutor.ExecuteCustom(ctx, deviceID, func(execCtx, pc) error {
        wrapper := pc.GetWrapper()
        _, err := wrapper.SendConfigs([]string{"quit"})
        return err
    }, singlePortTimeout)
    if cleanupErr != nil {
        applogger.Warnf("[portwrite] batch 端口间 quit 清理失败 ...")
    }
}
```

SendConfigs 路径会先调 `AcquirePriv("configuration")`：当前已在 configuration priv → noAction → `SendCommands(["quit"])` 直接发命令 + 等 prompt，从 `(config-if-<X>)#` 退到 `(config)#`。scrapli d.CurrentPriv 保持 "configuration"，下一个端口的 AcquirePriv 仍是 noAction → 进入新 interface view 正常工作。

## 证据

- timestamp: 2026-07-08
  type: code_diff
  file: internal/services/portwrite/batch_orchestrator.go
  finding: |
    主循环 `for portIdx, portID := range req.PortIDs` 加 `portIdx`；
    两条 succeeded 分支末尾调 `s.sendInterfaceExitCleanup(ctx, req.DeviceID, portIdx, len(req.PortIDs))`；
    新增 `sendInterfaceExitCleanup` 方法（quit via SendConfigs 复用 executor）

- timestamp: 2026-07-08
  type: code_diff
  file: internal/services/portwrite/batch_orchestrator.go
  finding: |
    顶部 import 加 `"github.com/xingran-next/xingran-go-backend/internal/device"` + `applogger "github.com/.../pkg/logger"`

- timestamp: 2026-07-08
  type: code_diff
  file: internal/services/portwrite/port_write_service_test.go
  finding: |
    TestBatchWritePorts_Success_AllSucceeded: `Times(3)` → `Times(5)`（3 端口写 + 2 端口间 quit cleanup）
    TestBatchWritePorts_FailFast_Transport: 加 `Return(nil).Once() // quit cleanup`
    TestBatchWritePorts_FailFast_DeviceRejected: 同上
    TestBatchWritePorts_PartialResult_Structure: 同上
    TestBatchWritePorts_RefreshOnceAtEnd_Invariant: `Times(3)` → `Times(5)`

- timestamp: 2026-07-08
  type: code_diff
  file: internal/services/portwrite/port_write_e2e_test.go
  finding: |
    TestE2E_Batch_FailFast fixtureByCall 加 `huawei_quit_success.fixture`（port-1→port-2 quit cleanup 用）

- timestamp: 2026-07-08
  type: new_file
  file: internal/services/portwrite/testdata/huawei_quit_success.fixture
  finding: |
    `<Huawei>\nquit\n[Huawei]\n` — SendConfigs(["quit"]) 在 file transport 下能正确解析 prompt pattern

## 排除

- hypothesis: "Batch 失败是 detached ctx 被 cancel（假设 A）"
  evidence: |
    TestBatchRepro_Port2CtxState 验证 3 端口 ctx 全是 `ctx.Err()=nil` + `deadlineLeft=~30min`。
    `BatchWritePorts` 第一行 `ctx, cancel := context.WithTimeout(context.Background(), batchDetachedTimeout)`（30min detached），
    与 caller 传入的 ctx 无关。
  timestamp: 2026-07-08

- hypothesis: "Batch 失败是 task_id 冲突导致 scheduler 去重（假设 C）"
  evidence: |
    executor.go:167 `task.ID = generateTaskID()`（time.Now().UnixNano）只用作日志 label；
    `task_scheduler.go:Submit` 按对象引用入队，不做 task_id 去重；
    即使 task_id 碰撞，第二次 Submit 仍会入队并由 worker 独立执行。
  timestamp: 2026-07-08

- hypothesis: "Batch 失败是 SSH pool 满导致快速失败（假设 B）"
  evidence: |
    `device-connection-pool-lru-eviction.md` 修复后 max_connections=50 + LRU 退让，
    单设备只占 1 个连接，batch 50 端口上限远低于池容量；
    真机测试 device=1（用户选端口所在设备），池满概率为 0。
  timestamp: 2026-07-08

## 决议

- root_cause: |
  厂商对"嵌套 interface 视图"行为差异：锐捷 RGOS 拒绝从 `(config-if-X)#` 直接 `interface Y` 进入新子视图，
  设备返回 `% Error: ...` → service `parseConfigError` 归类为 WriteErrorDeviceRejected → batch fail-fast break → 后续端口全部不进任何数组。
  华为/H3C 行为同 Cisco IOS 允许嵌套，所以单端口测试无问题。
- fix: |
  `BatchWritePorts` 串行循环里每个非末尾端口的 `executeWrite` 成功后，插入一条独立 `quit` cleanup cmd
  （`deviceExecutor.ExecuteCustom` 复用 + `wrapper.SendConfigs([]string{"quit"})`），
  让设备从 `(config-if-X)#` 退到 `(config)#`，下一个端口的 `interface Y` 能正常进入新子视图。
  cleanup 失败仅日志，不影响当前端口的 PortResult 与 fail-fast 语义。
- files_changed:
  - internal/services/portwrite/batch_orchestrator.go
  - internal/services/portwrite/port_write_service_test.go
  - internal/services/portwrite/port_write_e2e_test.go
  - internal/services/portwrite/testdata/huawei_quit_success.fixture (new)
- verification: |
  - go build ./... → 0
  - go test -count=1 ./internal/services/portwrite/... → ok (122s；包括 TestE2E_Batch_FailFast 用 3 fixtures PASS)
  - go test ./internal/services/portcollection/... → ok
  - go test ./internal/device/... → ok
  - 端到端验证（真实锐捷 RS8607E）需用户重新部署新二进制后跑批量测试
- design_note: |
  - cleanup 是 best-effort 状态卫生，失败仅日志 —— 不显式 break 也不影响 fail-fast 语义
  - 末尾端口不调 cleanup（设备停在 interface view 不影响下次会话；新 SSH session 从头开始）
  - SendConfigs(["quit"]) 复用 executor 路径而非新写 SendSingleCommand：保持 scrapli AcquirePriv/PromptPattern 一致处理
- deploy_note: 需重新编译部署后端
- related_knowledge: |
  - memory `normalize-iface-reverse-expand-trap` (vendor-aware 全名还原，已在 Bug #1 4 层修复)
  - 锐捷 RGOS 与 Cisco IOS 行为差异：interface view 嵌套 entry 接受/拒绝
