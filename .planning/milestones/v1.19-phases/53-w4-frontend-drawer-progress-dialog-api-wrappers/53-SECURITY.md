---
phase: 53-w4-frontend-drawer-progress-dialog-api-wrappers
audited: 2026-07-07
asvs_level: default
threats_total: 11
threats_closed: 11
threats_open: 0
status: secured
---

# SECURITY.md — Phase 53 W4 (Frontend Drawer + Progress Dialog + API Wrappers)

**Phase:** 53 — w4-frontend-drawer-progress-dialog-api-wrappers
**Audited:** 2026-07-07
**ASVS Level:** default
**Threats Closed:** 11/11
**Threats Open:** 0
**SECURITY.md state:** created (State B — first audit of phase)

## Audit Scope

实现文件（READ-ONLY，审计未修改）：
- `xingran-react-frontend/src/types/network.ts`
- `xingran-react-frontend/src/lib/api/networkApi.ts`
- `xingran-react-frontend/src/components/network/port-write/constants.ts`
- `xingran-react-frontend/src/components/network/port-write/PortWriteModal.tsx`
- `xingran-react-frontend/src/components/network/port-write/BulkWriteDrawer.tsx`
- `xingran-react-frontend/src/pages/network/ports/index.tsx`
- `xingran-react-frontend/src/pages/monitor/logs/index.tsx`

对照威胁登记表 T-53-01..T-53-10 + T-53-SC（来自 53-01-PLAN.md / 53-02-PLAN.md `<threat_model>`）。

## Threat Verification

| Threat ID | Category | Disposition | Evidence | Status |
|-----------|----------|-------------|----------|--------|
| T-53-01 | Tampering (wrapper URL literals) | mitigate | `networkApi.ts:267, 282, 299, 315, 330, 348` — 6 个 `post<>` 调用 URL 全部严格匹配 Phase 52 `port_write_router.go` kebab 路径（shutdown / undo-shutdown / description / dot1x-enable / dot1x-disable / batch）。Grep 计数 6，零 URL 笔误。 | CLOSED |
| T-53-02 | Information Disclosure (wrapper console.log) | accept | `networkApi.ts` 全文 grep `console.log` = **0 匹配**；6 wrapper 函数体无 console.log，PortResult.commandSent 不会被前端日志泄漏。 | CLOSED (accepted) |
| T-53-03 | Spoofing (wrapper bypass post() interceptor) | accept | 6 wrapper 函数体均为 `const result = await post<...>(url, body); return result.data!;`（networkApi.ts:267/282/299/315/330/348），全部走 `post()` 底座 —— token 注入 + SM2/SM4 加密 + 401 拦截器全保留。无 axios 直调绕过。 | CLOSED (accepted) |
| T-53-04 | Repudiation (PortResult.status field) | mitigate | `types/network.ts:300` — `status: "succeeded" \| "failed" \| "skipped";` 字面量联合类型，编译期锁定三态；同时 `PortWriteAction` (line 282-287) 锁定 5 action。BulkWriteDrawer 结果分区 (`result.failed/succeeded/skipped` 读 body) 配套，typo 会触发 TS 编译错误而非静默漏判。 | CLOSED |
| T-53-05 | Elevation of Privilege (frontend canWrite bypass) | accept | `ports/index.tsx:59-61` 前端 `useMenuStore` + `hasPermission("network:port:write")` 仅控制可见性。后端真相源已落地 Phase 52：`internal/api/v1/network/port_write_router.go:40` `write.Use(middleware.RequirePermissions([]string{string(permission.NetworkPortWrite)}, core))` 组级中间件强制 RBAC，前端绕过 → 后端 403。 | CLOSED (accepted) |
| T-53-06 | XSS (reason/description/commandSent/error render) | mitigate | PortWriteModal.tsx + BulkWriteDrawer.tsx + ports/index.tsx 全部用 antd 组件（Form.Item / Input / Input.TextArea / Select / Statistic / Table / Tag / Typography.Text code / Collapse）。Grep `dangerouslySetInnerHTML`：3 个文件中仅 2 处匹配且都在 JSDoc 注释（明确禁止），**0 处实际使用**。React 默认转义文本节点。 | CLOSED |
| T-53-07 | Tampering (BatchWriteRequest field construction) | mitigate | `BulkWriteDrawer.tsx:137-147` `buildRequest(deviceId, action, portIds, description?)` 仅构造白名单 4 字段对象，无 `...port` spread；`portIds = selectedPorts.map((p) => p.id)` (line 197) 从父组件受控的 `selectedRowKeys` 派生（ports/index.tsx:550 `portStatus.filter(p => selectedRowKeys.includes(p.id))`），不接受外部 URL/输入注入。 | CLOSED |
| T-53-08 | Information Disclosure (commandSent render to failure table) | accept | `BulkWriteDrawer.tsx:443-447` 失败明细 expandable row 通过 `<Typography.Text type="secondary" code>{port.commandSent \|\| "(无命令记录)"}</Typography.Text>` 渲染，仅 UI 可见，不入 console.log；审计真相源 `sys_port_write_audit` 表由 Phase 52 后端写入。 | CLOSED (accepted) |
| T-53-09 | Tampering (monitor/logs URL query module param) | mitigate | `monitor/logs/index.tsx:162-169` mount-only useEffect 读 `searchParams.get("module")` 仅调 `operLogManager.searchForm.setFieldsValue({ title: moduleFromUrl })` (line 165) 预填表单 + `handleSearch()` 触发标准 `post()` request body（line 60 `/monitor/oper-logs/list`）。后端 title 是 LIKE 过滤，无 SQL 字符串拼接；空值/异常值时 `if (moduleFromUrl && activeTab === "oper")` 守卫安全降级（不预填）。 | CLOSED |
| T-53-10 | Denial of Service (BulkWriteDrawer retry infinite loop) | mitigate | `BulkWriteDrawer.tsx:496` 重试按钮 `disabled={result.failed.length === 0}`；`handleRetryFailed` (line 150-173) `failedIds = batchResult.failed.map((p) => p.portId)` (line 152) 每次重试范围只含上次 failed（自然收敛 D-06）；按钮由用户主动 onClick 触发 (line 497) 非自动循环；batchResult.failed.length === 0 时函数提前 return (line 151)。 | CLOSED |
| T-53-SC | Tampering (npm dependencies) | accept | 7 文件均为纯前端 TS/TSX，无 `npm install` 任务，无新包引入；`antd` / `react-router-dom` 均已在 deps（53-01-SUMMARY 与 53-02-SUMMARY 均确认 vendor-react gzip 774.96 kB = 零回归 vs Phase 48 baseline）。 | CLOSED (accepted) |

## Accepted Risks Log

| Threat ID | Risk | Rationale |
|-----------|------|-----------|
| T-53-02 | wrapper 函数未输出日志便于调试 | post() 拦截器已统一处理 reject Toast；保留 commandSent 等敏感字段不进 console，审计追溯走 sys_port_write_audit（Phase 52 已落地） |
| T-53-03 | wrapper 走共享 post() 拦截器 | 此为期望行为：token/SM2/SM4/401 自动注入是项目安全基建，wrapper 复用即继承；绕过反而是风险 |
| T-53-05 | 前端 canWrite 仅控可见性 | 后端 `RequirePermissions(["network:port:write"])` 是真相源（port_write_router.go:40 组级中间件）；前端绕过无法绕过后端 403 |
| T-53-08 | commandSent 在失败明细 UI 可见 | 仅审计用途（运维需看到失败命令）；不入 console，不入额外日志聚合；audit 表 sys_port_write_audit 是真相源 |
| T-53-SC | 不引入新 npm 依赖 | 本 phase 纯类型/wrapper/UI 改造，零新依赖；bundle 零回归验证通过 |

## Unregistered Flags

无。SUMMARY.md `## Threat Flags` 两份均明确："无新增安全相关表面"。所有 mitigation 点均在 plan-time threat register 内验证。

## Audit Trail — Code Review Findings Resolution (commit 9b01cc68)

CR-01 / CR-02（cross-device write via stale deviceId）是 code review 阶段发现、不属于 plan-time threat register 的新增安全 finding，由 orchestrator 在 commit `9b01cc68` 修复。本次审计确认修复落地：

| Finding | Status | Evidence |
|---------|--------|----------|
| CR-01 retry 路径 deviceId 取自漂移快照 | Fixed | `BulkWriteDrawer.tsx:107` `lastDeviceId` state；`:207` 提交时缓存 `setLastDeviceId(deviceId)`；`:154` retry 用 `buildRequest(lastDeviceId, lastAction, failedIds, lastDescription)` 不再从 selectedPorts 重读 |
| CR-02 retry 错位 deviceId 触发后端 fallback 跨设备 SSH 写入 | Fixed | 同 CR-01 根因（前端 deviceId 缓存）；后端 batch_orchestrator port 归属防御属跨层加固建议，已在 53-REVIEW.md 记录为遗留跨层评估项，不阻塞 phase 53 |
| WR-03 result 视图 interfaceName 随父级刷新失真 | Fixed | `BulkWriteDrawer.tsx:108` `lastInterfaceMap` state；`:208` 提交时快照；`:265` ResultView 改用 prop 而非 selectedPorts 实时派生 |
| WR-01 validateFields `throw err` 未处理 rejection | Fixed | `PortWriteModal.tsx:152-154` + `BulkWriteDrawer.tsx:181-184` 改 `console.error + return` |
| IN-03 PortWriteModal 可重复提交 | Fixed | `PortWriteModal.tsx:135` submitting state + `:144/186` setSubmitting + `:205` `okButtonProps={{ loading: submitting }}` |

CR-01/CR-02 是迄今最严重的安全 finding（潜在跨设备误操作），修复经审计验证有效。

## Conclusion

11/11 threats 全部 CLOSED。Phase 53 W4 前端层（types + wrappers + Modal/Drawer + ports 改造 + monitor/logs URL 预填）安全可发布。WR-02 / IN-01 / IN-02 / 后端跨层加固为非阻塞性遗留，已在 53-REVIEW.md 跟踪，建议 Phase 54 UAT 后并入清理。
