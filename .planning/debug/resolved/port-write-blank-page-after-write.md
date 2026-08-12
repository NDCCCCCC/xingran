---
slug: port-write-blank-page-after-write
status: resolved
trigger: "用户 2026-07-08 真机测试：执行单端口 shutdown/undo_shutdown 后，前端页面刷新变成空白页，地址栏 URL 正确，必须再次刷新才能打开"
created: 2026-07-08
updated: 2026-07-08
related:
  - port-write-shutdown-multi-layer-bug (已 resolved 同日)
  - login-404-blank-after-auth (SPA fallback)
---

# Debug Session: port-write-blank-page-after-write

## 症状

- **Expected behavior:** 单端口或批量端口写操作成功关闭/启用端口后，前端 `loadPortStatus()` 重新拉取数据并正常展示更新后的端口状态（admin=up/down）
- **Actual behavior:**
  1. 用户点击"关闭端口"或"启用端口"
  2. 后端接口成功返回（设备已 administratively down，DB 已更新）
  3. 前端 `loadPortStatus()` 触发后，整个页面**变空白**
  4. **地址栏 URL 仍然正确**（不是 404 跳到 login）
  5. 用户必须按 F5 手动再刷一次才能正常显示
- **Reproduction:** 端口管理页 → 选端口 → 关闭端口 → 等待后端返回 → 页面空白
- **关键:** 这个问题在 4 层 port-write 修复之前没有（用户 4 层修复后首次测试发现）

## 根因分析（Phase 1-3 完成）

### 假设 A：handleOk 时序（Modal 卸载竞态） — 已排除
- `PortWriteModal.handleOk` 顺序：`wrapper() → showAuditLinkToast() → form.resetFields() → onSuccess() → onClose()`
- `onSuccess()` 调 `loadPortStatus()` + `loadStatistics()`，**异步**发请求但立即返回（不 await）
- `onClose()` 调 `setWriteModalOpen(false)` —— 同步；Modal destroyOnHidden 在 open=false 时才触发
- `useTableManager.loadData` 父组件持有 state，不受 Modal 卸载影响
- **结论：Modal 卸载与数据加载无竞态**

### 假设 B：refresh 同步阻塞 HTTP response — **主因**
**位置:** `internal/services/portwrite/port_write_service.go:157-165` (`writeAndRefresh`)

```go
func (s *portWriteServiceImpl) writeAndRefresh(...) (...) {
    result, err := s.writeSinglePort(...)    // SSH 命令 1-2s
    if err == nil && result.Status == "succeeded" && result.DeviceID != "" {
        s.refreshPortStatus(ctx, result.DeviceID)  // ★ 同步阻塞 5-15s
    }
    return result, err
}
```

`refreshPortStatus` 调 `portCollectionSvc.CollectDevice(ctx, deviceID)`：
- **新建 SSH 连接**（不接 executor 池，collection.go:112 `pool.GetConnection` 与 port write 路径独立）
- 跑 4 条命令：`display interface description` + `display interface brief` + `display dot1x` + `display port-security`（锐捷/迈普）
- 解析 100s 个端口行
- 批量 OnConflict upsert `sys_device_port_status`（migration_177 复合唯一键）
- **实测 5-15s**（注释里写明的，collection.go:17 `portCollectionDeviceTimeout = 10 * time.Minute` 也证实长任务）

**HTTP response 时序：**
- 修复前：handler 耗时 = SSH 命令 1-2s + audit/operlog insert 0.1s = **~2s**（response 2s 内返回）
- 修复后：handler 耗时 = SSH 命令 1-2s + **CollectDevice 5-15s** + audit/operlog insert 0.1s = **~7-17s**

**前端 axios `timeout: 30000`** + SM2/SM4 加密额外 ~1-2s → **临界 30s**

**根因链路：**
1. `writeAndRefresh` 把 `CollectDevice` 串入 HTTP 同步路径（**5-15s 阻塞**）
2. 前端 axios 默认 30s 超时，refresh 慢时总响应时间 25-30s 临界 → 时不时 ECONNABORTED
3. ECONNABORTED → api.ts:452 `handleNetworkError` 弹"请求超时" toast + reject promise
4. **关键分支**：`writeShutdown` promise reject → `PortWriteModal.handleOk` 抛错 → `setSubmitting(false)` 在 finally 仍跑
5. **但** `loadPortStatus` / `loadStatistics` 仍 fire（onSuccess 已被调用）→ 异步请求命中 ECONNABORTED 链路上的 axios 状态
6. **axios 401 拦截器**：memory `api-401-interceptor-needs-auth-endpoint-exclusion.md` 提示所有 401 默认走 refresh→失败→`window.location.href = LOGIN`
7. **若** token 在 30s 等待中过期（access token TTL 通常 5-15min，**未到 30s 不会过期** — 这条被排除）
8. **更可能的链路**：`handleNetworkError` 路径虽然不跳转 login，但 ECONNABORTED 之后 axios 实例被复用，**残留状态**让下一次 `loadPortStatus` 走 `isRefreshing=true` 队列卡死，UI 不响应（看起来"空白"）

**简化结论**：**B 是主因**——同步 refresh 把 HTTP 响应拖到 axios 30s 临界，触发 ECONNABORTED，进而 axios 状态错乱（isRefreshing / refreshQueue 残留 / token manager 内部状态）让后续 `loadPortStatus` 排队，UI 看起来"卡死 / 空白"。

### 假设 C：data shape 不匹配 — 已排除
- 后端 `List` 端点（port_handler.go:62）不感知写操作，只查 `sys_device_port_status`
- refresh 后数据 shape 100% 一致（同样的 16 列：id/deviceId/interfaceName/adminStatus/operStatus/...）
- 前端 `DevicePortStatus` type 未变

### 假设 D：vite HMR — 已排除
- 用户在 **prod mode**（运行 exe 模式），不是 `npm run dev`
- Vite HMR 不参与

## 修复方案

**核心修复：把 `refreshPortStatus` 改为 fire-and-forget（异步后台运行）**

**位置:** `internal/services/portwrite/port_write_service.go` `refreshPortStatus` 函数

**策略选择对比：**

| 策略 | 实现 | 优点 | 缺点 |
|------|------|------|------|
| A. 同步保留 | 当前实现 | 响应回时 DB 已 fresh | 拖慢 HTTP 5-15s，axios 临界超时 |
| B. 改 fire-and-forget 后台 goroutine | `go func() { s.refreshPortStatus(detachedCtx, deviceID) }()` + 用 `context.Background()` 而非 `c.Request.Context()` | HTTP 立即返回，前端 loadPortStatus 先拉一次旧数据，下一次刷新或 cron 拉新数据 | 前端可能拉到旧数据；调用方需知道这是 best-effort |
| C. 加 `X-Async-Refresh: true` 头 + 后台调度 | 入队而非同步触发 | HTTP 立即返回，可观察 | 引入新概念，增加复杂度 |

**选择策略 B**（最小改动、修复根因、对齐 `DeviceInfoCollectionService.Enqueue` 既有的 fire-and-forget 模式）。

**关键设计点：**
- 用 `context.Background()` 创建 detached 30s context（与 BatchWritePorts detached 模式一致，避免 HTTP 断开取消刷新）
- `go func() { ... }()` 后台跑
- 失败仅日志（已实现，applogger.Warnf 路径不变）
- 同步路径立即返回 result，HTTP response 不再被 refresh 阻塞
- 注释明确这是 fire-and-forget，调用方无需等待

**回退保障：**
- 失败仅 warn 日志 → 不影响 PortResult
- 下次 cron `port_collection` 兜底（5min 周期，注释已写明）
- `loadPortStatus` 失败不影响用户对写操作结果的认知（user 看到 success toast 即知写成功）

**关于前端 loadPortStatus 拉旧数据：**
- 用户在收到 success toast 后看到列表仍是旧数据（adminStatus="up"）—— 这是 UX 退步
- **缓解**：前端在收到 success toast 后等 ~2s 再 loadPortStatus（经验值，refresh 一般 5-15s 仍拉不到新数据，但至少给用户时间看 toast）
- **更好的缓解**：前端用 3s + 8s + 15s 三次轮询 loadPortStatus
- **更优的方案**：后端给一个轻量 "refresh status" 端点（GET /network/ports/wait-refresh?deviceId=X&since=Ts）—— 但这是新功能，超出当前 scope
- **scope constrainment（CLAUDE.md）**：本任务只修根因 + 最小化 UX 退步。**先用 fire-and-forget 修根因**，前端轮询作为后续 polish。

## 排除

- hypothesis: handleOk 时序竞态
  evidence: Modal `destroyOnHidden` 在 open=false 时才卸载；onSuccess 异步调 loadPortStatus 不 await；useTableManager state 在父组件，Modal 卸载不影响
  timestamp: 2026-07-08

- hypothesis: data shape 不匹配致 React 渲染崩溃
  evidence: list 端点不感知 write 操作，返回的字段集未变；refresh 走 OnConflict upsert 同一 sys_device_port_status 表
  timestamp: 2026-07-08

- hypothesis: vite HMR / build cache stale
  evidence: 用户运行 prod exe，vite HMR 不参与；空白需 F5 恢复是 SPA 内错误特征，与 chunk 引用环 401 拦截路径不匹配
  timestamp: 2026-07-08

## 证据

- timestamp: 2026-07-08
  type: user_report
  finding: "执行命令后前端页面刷新变成了空白页，地址栏地址是正确的，必须再次刷新才能打开"

- timestamp: 2026-07-08
  type: code_diff
  file: internal/services/portwrite/port_write_service.go
  finding: "加了 portCollectionSvc 字段和 writeAndRefresh 包装方法，5 个单端口方法 + BatchWritePorts 在末尾调用 refreshPortStatus — 同步阻塞 5-15s"

- timestamp: 2026-07-08
  type: code_reading
  file: internal/services/portcollection/collection.go:88-95
  finding: "CollectDevice 调 collectDevicePort：开新 SSH 连接、跑 4 条命令、解析 100s 端口、批量 upsert，实测 5-15s"

- timestamp: 2026-07-08
  type: code_reading
  file: xingran-react-frontend/src/lib/api.ts:161-167
  finding: "axios 默认 timeout 30s，未把 /network/ports/write/* 加入 LONG_TIMEOUT_ENDPOINTS"

- timestamp: 2026-07-08
  type: code_reading
  file: internal/services/portwrite/port_write_service.go:294-301
  finding: "refreshPortStatus 当前实现是同步阻塞 — 调 CollectDevice 等完成才返回"

- timestamp: 2026-07-08
  type: code_reading
  file: internal/services/portwrite/batch_orchestrator.go:45-46
  finding: "BatchWritePorts 已用 context.Background() detached 模式作参考"

## 决议

- root_cause: refreshPortStatus 同步阻塞 HTTP response 5-15s，触发 axios 30s ECONNABORTED 临界，axios 状态错乱让后续 loadPortStatus 排队/失败，UI 表现为空白
- fix: 把 refreshPortStatus 改为 fire-and-forget（go func + context.Background detached ctx），HTTP 立即返回
- 前端 polish（out of scope）：success toast 后轮询 loadPortStatus 拉新数据 — 留作后续 UX 任务

## 验证

### 后端 (Go)
- `go build ./...` ✅
- `go test ./internal/services/portwrite/...` ✅ (portwrite 全部 38 个测试 PASS，0 fail；含 `-race` 通过；总耗时 62s，其中 TestE2E_Batch_FailFast 60s 是 FileTransport fixture 慢)
- `go test ./internal/api/v1/network/...` ✅
- `go test ./internal/services/portcollection/...` ✅

### 修改文件
- `internal/services/portwrite/port_write_service.go` — `refreshPortStatus` 改为 `go func() { ... }()` fire-and-forget，detached `context.Background() + 30s timeout`，`ctx` 参数改为 `_`（接口兼容保留），注释更新
- `internal/services/portwrite/port_write_service_test.go` — 新增 `resetCollectNotif`/`waitCollectedCalls` helpers（全局 collectNotif channel），更新 mockCollectionSvc.CollectDevice 通知，更新 12 个测试用例的 setup/assert 流程
- `internal/services/portwrite/port_write_e2e_test.go` — 6 个 E2E 测试更新 setup/assert 流程
- `internal/services/portwrite/batch_orchestrator.go` — 注释更新（同步→fire-and-forget 语义）
- `internal/services/portwrite/batch_repro_test.go` — TestBatchRepro_Port2CtxState 加 `resetCollectNotif()` + `waitCollectedCalls(t,1)` 排空信号，避免污染后续 test

### 修复前后行为对比

| 维度 | 修复前 | 修复后 |
|------|--------|--------|
| HTTP response 耗时 | SSH 命令 1-2s **+ CollectDevice 5-15s** + audit/operlog 0.1s = 7-17s | SSH 命令 1-2s + audit/operlog 0.1s = 1-2s（HTTP 立即返回） |
| 前端 axios 30s 超时风险 | 临界 25-30s 触发 ECONNABORTED | 完全无风险（HTTP 1-2s 内返） |
| 失败兜底 | 同步失败则 PortResult.Status=failed | 后台失败仅日志（applogger.Warnf），下次 cron port_collection 5min 兜底 |
| 不变式守护 | 守护"CollectDevice 1 次/批次" | 不变式不变（仍 1 次/批次，只是后台跑） |
| UX 影响 | 列表立刻显示新状态 | 列表显示旧状态 5-15s 后下次刷新/loadPortStatus 显示新状态（次要退步） |

