---
phase: 43-r2
plan: 03
subsystem: asset-reconciliation
tags: [feat, websocket, sys-notice, frontend, dual-channel, R2]
dependency_graph:
  requires: [ReconciliationWorkorderService.CreateWorkorderFromException (43-01), RegisterReconciliationTasks signature (43-01), websocket.NoticeHub.BroadcastToAll (Phase 19/20), services.NoticeService.CreateNoticeWithTargets + PublishNotice (Phase 34), useWebSocket hook (Phase 32), queryKeys.reconciliation factory (Phase 42 R1), reconciliationApi exceptionList (42-R1), POST /asset/reconciliation/exception/:id/resolve (43-02), sys_data_reconciliation.resolvedAt field (43-02)]
  provides: [ReconciliationWorkorderService.CreateWorkorderFromException WS + SysNotice dual-channel (D-A4-01/03), 2 WS event constants critical_exception_detected / critical_workorder_created, sysNoticePrefix '[asset_reconciliation_critical]' filter token, RegisterReconciliationTasks signature extension (wsSvc + noticeSvc params), useReconciliationWebSocket hook (filter 2 critical events + auto queryClient.invalidateQueries), dashboard WS status Badge + onCriticalEvent toast, exceptions page '标记已解决' button + Modal + permission gate, reconciliationApi.exceptionResolve method]
  affects: [internal/services/asset (reconciliation_workorder.go WS + SysNotice), internal/scheduler (reconciliation_tasks.go signature + woSvc ctor), internal/core (core.go call site with c.NoticeHub + NewNoticeService), xingran-react-frontend/src/hooks (new useReconciliationWebSocket.ts), xingran-react-frontend/src/lib (assetApi.ts exceptionResolve), xingran-react-frontend/src/pages/asset/reconciliation/dashboard (WS hook + Badge), xingran-react-frontend/src/pages/asset/reconciliation/exceptions (resolve column + Modal)]
tech_stack:
  added: []
  patterns: [ws + sys_notice dual-channel critical alert (D-A4-01/03), severity='critical' guard before WS broadcast, recover panic for hub.BroadcastToAll race condition, sys_notice.NoticeContent prefix '[asset_reconciliation_critical]' as frontend filter token, services.NewNoticeService + c.NoticeHub DI through scheduler.RegisterReconciliationTasks, useReconciliationWebSocket thin wrapper over useWebSocket + QueryClient.invalidateQueries, useMenuStore.permissions for frontend resolve perm gate, exceptionResolve promise-based with onError message.error + invalidateQueries on duplicate-resolve 400]
key_files:
  created:
    - xingran-react-frontend/src/hooks/useReconciliationWebSocket.ts (~150 lines: WS hook + type defs + critical filter logic)
  modified:
    - internal/services/asset/reconciliation_workorder.go (+227 lines: noticeHub + noticeService DI + 2 broadcast helpers + publishCriticalSysNotice + recover panic guards)
    - internal/scheduler/reconciliation_tasks.go (+13 lines: import services + websocket, signature +wsSvc +noticeSvc, woSvc ctor injection)
    - internal/core/core.go (+2 lines: pass c.NoticeHub + services.NewNoticeService(c.GetDB()) to RegisterReconciliationTasks)
    - xingran-react-frontend/src/lib/assetApi.ts (+14 lines: exceptionResolve method with body {resolutionNote?})
    - xingran-react-frontend/src/pages/asset/reconciliation/dashboard/index.tsx (+44 lines: useQueryClient + useReconciliationWebSocket + toast onCriticalEvent + WS status Badge in header)
    - xingran-react-frontend/src/pages/asset/reconciliation/exceptions/index.tsx (+66 lines: resolve_modal state + useMenuStore perms + '解决' column + Modal with resolution_note Form + handleResolveSubmit)
decisions:
  - "WS broadcast 用 BroadcastToAll 全量推送(D-A4-01 锁定) — 简化订阅模型,不按角色过滤;dashboard 列表无则忽略"
  - "WS 只推 2 类事件(D-A4-02 锁定):critical_exception_detected + critical_workorder_created;high/medium/low 触发不推"
  - "severity='critical' guard 在 CreateWorkorderFromException 内 — 非 critical 转单(如 high)走 43-01 已建立的转单流程,无 WS / SysNotice"
  - "recover() 包裹 BroadcastToAll — hub 关闭或 channel 写满(256 buffer)时 panic 不会传播到 cron 循环,避免 2min/5min 周期崩溃"
  - "SysNotice notice_type='2'(警告) + NoticeContent 头部拼接 sysNoticePrefix='[asset_reconciliation_critical]' — 前端过滤 token,无 schema notice_type_str 字段时不引入 migration(R3 决定是否扩展)"
  - "PublishNotice 立即发布(非 Draft)— SysNotice 必须可见,运维下次登录收件箱就有告警"
  - "broadcast + publish 都 fail-soft(logrus.Warnf)— 工单已创建并回写 workorder_id,D-A1-03 风格不阻塞业务"
  - "ReconciliationWorkorderService 构造器向后兼容(无 wsSvc/noticeSvc 不破坏)— 单测和非 production 环境可传 nil"
  - "useReconciliationWebSocket 显式 queryClient 传入 — 避免循环依赖 hooks/lib,使用方调用更明确"
  - "WS URL 复用 buildWebSocketUrl()(同 useRPAProgress 共享 noticeApi 模式)— 无新 endpoint / 无新 WS handler / 无新鉴权流程"
  - "前端 resolve perm 用 useMenuStore.permissions.includes('asset:reconciliation:resolve') — 沿用 D-08 established pattern(见 MACHistoryPage + LocationAliasDrawer)"
  - "已解决记录 button disabled + 文本切 '已解决' — 避免重复 resolve(后端 43-02 已防御 + frontend UX 防御)"
  - "resolve 成功后 queryClient.invalidateQueries(queryKeys.reconciliation.all) — dashboard + exceptions + summary 全部 refetch"
  - "重复 resolve 400 时也 invalidate — 后端防御可能 stale 缓存,UI 必须刷新拉最新状态"
metrics:
  duration: ~9 min
  completed_date: 2026-06-28
  tasks: 2
  files_created: 1
  files_modified: 6
  commits: 2
  lines_added: ~573
---

# Phase 43 Plan 03: WebSocket + SysNotice + 标记已解决 UI Summary

## One-liner

critical 异常/工单 WS 全量推送(2 类事件 critical_exception_detected / critical_workorder_created)+ sys_notice 写入(notice_type=2 + NoticeContent 头部 prefix `[asset_reconciliation_critical]`)+ 前端 useReconciliationWebSocket hook(自动 query invalidation)+ dashboard WS 状态 Badge + toast + exceptions 列表"标记已解决"按钮 + Modal,完成 Phase 43 R2 闭环第三步(实时推送 + 用户操作界面),符合 MONITOR-02 + MONITOR-03 + WORKORDER-02 ROADMAP SC 4+5+6。

## What Built

### Task 1: 后端 WS + SysNotice 接入 + cron 触发

- **`internal/services/asset/reconciliation_workorder.go`** (+227 lines)
  - `ReconciliationWorkorderService` 结构扩展:`noticeHub *websocket.NoticeHub` + `noticeService *services.NoticeService` 字段(构造器注入)
  - 构造器签名扩展:`NewReconciliationWorkorderService(db, noticeHub, noticeService)`,**向后兼容**(任一参数为 nil 时跳过对应通道)
  - 2 个 WS 事件类型常量:`EventCriticalExceptionDetected = "critical_exception_detected"` + `EventCriticalWorkorderCreated = "critical_workorder_created"`
  - `sysNoticePrefix = "[asset_reconciliation_critical]"` — 前端过滤 token(SysNotice 内容头部)
  - `broadcastCriticalException(ctx, rec)` — WS 推送 critical_exception_detected(severity='critical' guard)
  - `broadcastCriticalWorkorder(ctx, rec, workorderID, title)` — WS 推送 critical_workorder_created(第 11 步 UPDATE workorder_id 后)
  - `publishCriticalSysNotice(ctx, rec, workorderID, title, assetCode)` — 写 sys_notice + 立即发布(notice_type='2' 警告, Priority=Urgent, TargetType=All)
  - **recover() panic guard** — hub 已关闭或 channel 写满(256 buffer 满)时 panic 不传播到 cron 循环
  - **fail-soft** — WS / SysNotice 任何通道失败仅 logrus.Warnf,不阻塞主流程(D-A1-03 风格)

- **`internal/scheduler/reconciliation_tasks.go`** (+13 lines)
  - 新增 import:`internal/services` + `internal/websocket`
  - `RegisterReconciliationTasks` 签名扩展:`+ wsSvc *websocket.NoticeHub, + noticeSvc *services.NoticeService`
  - `woSvc := asset.NewReconciliationWorkorderService(db, wsSvc, noticeSvc)` — 构造器注入 DI
  - 注释更新:R2 闭环(D-A4-01 + D-A4-03)章节

- **`internal/core/core.go`** (+2 lines)
  - `RegisterReconciliationTasks` 调用处:`scheduler.RegisterReconciliationTasks(c.Scheduler, c.GetDB(), c.NoticeHub, services.NewNoticeService(c.GetDB()))`
  - 注释更新:Phase 43 R2 注入说明,nil 跳过保持向后兼容

### Task 2: 前端 useReconciliationWebSocket hook + dashboard 集成 + 标记已解决 Modal

- **`xingran-react-frontend/src/hooks/useReconciliationWebSocket.ts`** (新文件,~150 lines)
  - 2 个 type literal:`ReconciliationCriticalEvent = 'critical_exception_detected' | 'critical_workorder_created'`
  - `ReconciliationCriticalPayload` interface:workorder_id / exception_id / asset_code / conflict_type / severity / title / detected_at
  - `useReconciliationWebSocket({ queryClient, onCriticalEvent, enabled })` 主 hook
  - 内部用 `useWebSocket({ url: buildWebSocketUrl() })` 复用项目现有 WS endpoint `/system/ws/notices`
  - **事件过滤**:`msg.type in ['critical_exception_detected', 'critical_workorder_created']` 才处理,其他(new_notice / rpa_* 等)直接忽略
  - **自动 query invalidation**:`queryClient.invalidateQueries({ queryKey: queryKeys.reconciliation.all })` — 触发 dashboard + exceptions + summary 全部 useQuery 重新拉取
  - `onCriticalEvent` 业务回调:toast / Badge 闪烁等 UI 反馈
  - 复用 `useWebSocket` 内置指数退避(2s/4s/8s/.../30s,最多 10 次)

- **`xingran-react-frontend/src/lib/assetApi.ts`** (+14 lines)
  - `reconciliationApi.exceptionResolve(id, body)` 方法:`POST /asset/reconciliation/exception/${id}/resolve`
  - Body:`{ resolutionNote?: string }`(可选)
  - 返回:`{ id, resolvedAt, resolvedBy, resolutionNote }`
  - 注释:权限说明(R3 后端强制)+ operlog 后端自动写入

- **`xingran-react-frontend/src/pages/asset/reconciliation/dashboard/index.tsx`** (+44 lines)
  - 新增 import:`useQueryClient` from `@tanstack/react-query` + `Badge` from antd + `App.useApp()` + `useReconciliationWebSocket`
  - 顶部:`const queryClient = useQueryClient(); const { message } = App.useApp(); const { status: wsStatus } = useReconciliationWebSocket({ queryClient, onCriticalEvent })`
  - onCriticalEvent:toast.info(中文 label 映射:`critical_exception_detected → "新 critical 异常"`, `critical_workorder_created → "已生成 critical 工单"`)
  - 页面 header 增加 WS 状态 Badge:connected → 绿色"实时推送已连接" / connecting → 蓝色"正在连接..." / error → 红色"推送连接异常" / default → 灰色"推送已断开"

- **`xingran-react-frontend/src/pages/asset/reconciliation/exceptions/index.tsx`** (+66 lines)
  - 新增 import:`Modal` from antd + `CheckOutlined` from @ant-design/icons + `useQueryClient` + `useMenuStore` + `reconciliationApi` + `queryKeys`
  - 顶部:`const permissions = useMenuStore((s) => s.permissions); const canResolve = permissions.includes('asset:reconciliation:resolve')` (D-08 established pattern)
  - resolve Modal 状态:`{ open, exceptionId, note, submitting }` + `resolveForm`
  - 新增列"解决"(resolve_btn,fixed right):无 perm 时不渲染;有 perm 时显示按钮;已 resolved 时 disabled + 文本"已解决";未 resolved 时 onClick 打开 Modal
  - handleResolveSubmit:validate Form → reconciliationApi.exceptionResolve → message.success → invalidateQueries → 关闭 Modal
  - 失败处理:errMsg 含 "已解决" 时也 invalidate(后端 400 stale 状态同步刷新)
  - Modal 底部:Input.TextArea(resolution_note 可选,maxLength 500 + showCount)
  - Table scroll x 从 1280 增加到 1380(新列宽度)

## Verification

| Criterion | Status | Evidence |
|-----------|--------|----------|
| go build ./... exit 0 | PASS | `go build ./...` no output |
| go test ./internal/services/asset/... ./internal/api/v1/asset/... ./internal/scheduler/... | PASS | All PASS, no regression |
| npm run build exit 0 | PASS | `built in 1m 25s` |
| critical_workorder_created constant | PASS | `reconciliation_workorder.go:72` |
| critical_exception_detected constant | PASS | `reconciliation_workorder.go:71` |
| asset_reconciliation_critical prefix | PASS | `reconciliation_workorder.go:76,406` |
| BroadcastToAll 调用 | PASS | `reconciliation_workorder.go:316,378`(2 处) |
| CreateNoticeWithTargets + PublishNotice 调用 | PASS | `reconciliation_workorder.go:411,418`(publishCriticalSysNotice 内) |
| RegisterReconciliationTasks 签名扩展 | PASS | `reconciliation_tasks.go:39` + `core.go:326` |
| useReconciliationWebSocket 导出 | PASS | `useReconciliationWebSocket.ts:147 export default` |
| exceptionResolve 方法 | PASS | `assetApi.ts:224 exceptionResolve` |
| dashboard WS status Badge | PASS | `dashboard/index.tsx:243-261 Badge status/text` |
| exceptions 标记已解决按钮 + Modal | PASS | `exceptions/index.tsx:283-313 resolve_btn col + 499-523 Modal` |
| 权限控制 asset:reconciliation:resolve | PASS | `exceptions/index.tsx:90 canResolve` |

## Deviations from Plan

### Plan Adjustments

**1. [Refactor] 构造器签名变化注入 — 向后兼容而非 breaking change**
- **Found during:** Task 1 review
- **Issue:** plan 写"扩展构造器注入",但 43-01 已部署在生产环境,破坏性签名变更会要求所有调用方同步修改
- **Fix:** `NewReconciliationWorkorderService(db, noticeHub, noticeService)` 三参版本,nil 时跳过对应通道;core.go 调用方传 `(c.Scheduler, c.GetDB(), c.NoticeHub, services.NewNoticeService(c.GetDB()))`,非 production 环境的 test 可继续传 nil
- **Files modified:** internal/services/asset/reconciliation_workorder.go, internal/scheduler/reconciliation_tasks.go, internal/core/core.go
- **Commit:** 9a32c80a

**2. [Plan refinement] useReconciliationWebSocket 显式 queryClient 参数而非 hook 内部获取**
- **Found during:** Task 2 design review
- **Issue:** plan 提到"内部用 useQueryClient()"可能导致 hooks/lib 间循环依赖,且组件解耦性差
- **Fix:** `useReconciliationWebSocket({ queryClient, onCriticalEvent, enabled })` 显式 queryClient 传入,使用方代码更明确,易于测试
- **Files modified:** xingran-react-frontend/src/hooks/useReconciliationWebSocket.ts
- **Commit:** 9a9d1fdf

## Auth Gates

None — 计划无外部认证依赖;WS endpoint 复用现有 /system/ws/notices(已 JWT 鉴权),前端 WS URL 通过 SecureTokenStorageImpl.getAccessToken() 自动拼接 token query param。

## Known Stubs

无 — 所有代码路径完整,无 TODO/FIXME/placeholder。

注:
- `asset:reconciliation:resolve` 权限粒度:前端按钮渲染按 useMenuStore.permissions 控制可见性;后端 R2 简化不强制 RequirePermissions(R3 增强)
- SysNotice notice_type='2' + NoticeContent 头部 prefix '[asset_reconciliation_critical]' 作为前端过滤 token;无 schema notice_type_str 字段时不引入 migration,R3 决策

## Threat Flags

无新安全面。所有威胁已在 plan frontmatter 评估并 mitigation:

- **T-43-10** (WS 事件伪造) — mitigate,WS endpoint 需 JWT auth(现有 /system/ws/notices);event 字段来自后端 Broadcast 不可被客户端伪造
- **T-43-11** (任意用户 resolve) — mitigate,前端按钮按 useMenuStore.permissions 含 asset:reconciliation:resolve 渲染;后端 43-02 handler 无显式 perm 校验(由 router RequirePermissions 继承 R1 列表 perms)— R3 增强
- **T-43-12** (WS 推送无审计) — mitigate,workorder.BaseService.Create 内部 operlog + sys_notice 写库即审计 + ResolveException handler 显式 operlog.Record(43-02)
- **T-43-13** (WS 频繁推送 DoS) — mitigate,D-A4-02 仅 critical 2 类事件(2min/5min 周期,频率低);D-A3-02 24h 节流抑制 source
- **T-43-14** (WS payload 泄露) — mitigate,payload 只含 workorder_id/exception_id/title/severity/asset_code,无 PII(物理/责任用户名等敏感信息不暴露)
- **T-43-SC** (新依赖) — mitigate,无新依赖,前端用现有 useWebSocket + TanStack Query

## Followup Notes

### UAT 验证项(用户执行)

1. **启动 backend** → 检查 sys_notice 表有 R2 写入(notice_type='2' + NoticeContent 头部含 `[asset_reconciliation_critical]` prefix)
2. **检查 WS endpoint** → `ws://host/system/ws/notices?token=<JWT>` 用浏览器开发者工具或 wscat 连接,验证能收到 critical 事件
3. **打开 dashboard** → 页面 header 看到"实时推送已连接" Badge(connected)
4. **触发 critical 转单** → 在 sys_data_reconciliation 插入一条 severity='critical' 的记录,等待 2min → 验证:
   - WS 推送 2 条事件(`critical_exception_detected` + `critical_workorder_created`)
   - sys_notice 写入 1 条(notice_type='2' + NoticeContent 头部 `[asset_reconciliation_critical]` prefix)
   - toast.info 弹出"新 critical 异常" + "已生成 critical 工单"
   - Dashboard KPI 卡数字刷新(自动 query invalidation)
5. **打开异常列表** → 在有 `asset:reconciliation:resolve` 权限时看到"标记已解决"按钮;无权限时不显示
6. **点"标记已解决"** → 弹 Modal → 填 resolution_note → 确认 → 验证:
   - sys_data_reconciliation.resolved_at / resolved_by / resolution_note 已更新
   - sys_oper_log 新增 OperTypeUpdate 记录(WORKORDER-02)
   - 列表该行 button 变 disabled + 文本"已解决"
   - Dashboard 数字减 1
7. **重复 resolve 同一记录** → 验证返回 400 "该异常已标记为已解决" + 列表自动 invalidate 刷新
8. **非 critical 异常** → 验证不触发 WS 推送(severity='high' / 'medium' / 'low' 不在 D-A4-02 范围)

### 数据流总览

```
┌─────────────────┐
│ Layer 3 检测    │ (43-02 已有,DetectLayer3 写入 sys_data_reconciliation)
└────────┬────────┘
         ↓
┌─────────────────────────────────────┐
│ cron reconciliation:createWorkorder  │ (43-01 已建,@every 2m/5m)
│ Critical 触发路径                    │
└────────┬────────────────────────────┘
         ↓
┌─────────────────────────────────────────────────┐
│ ReconciliationWorkorderService                  │
│  CreateWorkorderFromException (43-01 + 43-03)  │
│   1. SELECT 异常                                  │
│   2. workorder.BaseService.Create → 工单            │
│   3. UPDATE workorder_id                          │
│   4. WS BroadcastToAll critical_workorder_created │ ← 本 plan 新增
│   5. NoticeService.CreateNoticeWithTargets +     │ ← 本 plan 新增
│      PublishNotice (notice_type='2')              │
└────────┬───────────────────────────────────────┬─┘
         ↓                                       ↓
┌─────────────────┐                    ┌──────────────────┐
│ WebSocket Hub   │                    │ sys_notice 表    │
│ (NoticeHub)     │                    │ (D-A4-03 双通道) │
└────────┬────────┘                    └──────────────────┘
         ↓
┌─────────────────────────────────────┐
│ useReconciliationWebSocket Hook     │ (本 plan 新建)
│  1. 过滤 critical_* 2 类事件         │
│  2. queryClient.invalidateQueries   │
│  3. onCriticalEvent 回调(toast)     │
└────────┬────────────────────────────┘
         ↓
┌─────────────────────────────────────┐
│ Dashboard + Exceptions 列表自动刷新  │
│  - queryKeys.reconciliation.all     │
│  - summary / byConflictType / by    │
│    Severity / healthTrend / exceptionList 全部 refetch │
└─────────────────────────────────────┘
```

### 端到端验证清单(运维侧)

- [ ] Dashboard 页面有 WS Badge 显示 connected
- [ ] 触发 critical 工单 → toast 弹出 2 次 + Dashboard 数字刷新
- [ ] 离线运维登录 → sys_notice 收件箱看到 R2 写入的通知(双通道设计意图)
- [ ] 异常列表点"标记已解决" → 工单独立关闭流程(不在本 plan scope,D-A4-04 锁定不联动)
- [ ] 7d 静默期 + 24h 节流继续生效(43-02 验证)

## Self-Check: PASSED

- `xingran-react-frontend/src/hooks/useReconciliationWebSocket.ts` exists ✓
- `internal/services/asset/reconciliation_workorder.go` updated with WS + SysNotice dual-channel ✓
- `internal/scheduler/reconciliation_tasks.go` updated signature + woSvc injection ✓
- `internal/core/core.go` updated call site with c.NoticeHub + NewNoticeService ✓
- `xingran-react-frontend/src/lib/assetApi.ts` updated with exceptionResolve ✓
- `xingran-react-frontend/src/pages/asset/reconciliation/dashboard/index.tsx` updated with useReconciliationWebSocket + Badge ✓
- `xingran-react-frontend/src/pages/asset/reconciliation/exceptions/index.tsx` updated with resolve_btn col + Modal ✓
- Commits `9a32c80a` (Task 1) + `9a9d1fdf` (Task 2) exist in git log ✓
- `go build ./...` exit 0 ✓
- `go test ./internal/services/asset/... ./internal/api/v1/asset/... ./internal/scheduler/...` PASS ✓
- `npm run build` exit 0 ✓
## Self-Check: PASSED

- `xingran-react-frontend/src/hooks/useReconciliationWebSocket.ts` exists
- `internal/services/asset/reconciliation_workorder.go` updated with WS + SysNotice dual-channel
- `internal/scheduler/reconciliation_tasks.go` updated signature + woSvc injection
- `internal/core/core.go` updated call site with c.NoticeHub + NewNoticeService
- `xingran-react-frontend/src/lib/assetApi.ts` updated with exceptionResolve
- `xingran-react-frontend/src/pages/asset/reconciliation/dashboard/index.tsx` updated with useReconciliationWebSocket + Badge
- `xingran-react-frontend/src/pages/asset/reconciliation/exceptions/index.tsx` updated with resolve_btn col + Modal
- Commits `9a32c80a` (Task 1) + `9a9d1fdf` (Task 2) + `82cc63c1` (SUMMARY) + `af9f2225` (state metadata) exist in git log
- `go build ./...` exit 0
- `go test ./internal/services/asset/... ./internal/api/v1/asset/... ./internal/scheduler/...` PASS
- `npm run build` exit 0
