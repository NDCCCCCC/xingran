---
phase: 53-w4-frontend-drawer-progress-dialog-api-wrappers
plan: 02
subsystem: frontend
tags: [frontend, react, antd, port-write, modal, drawer, permission-gating, monitor-logs, url-query]
requires:
  - "Phase 53-01: PortWriteAction/PortResult/BatchWriteRequest/BatchResult TypeScript 类型 + 6 wrapper 函数 + 5 共享常量 (networkApi.ts / types/network.ts / constants.ts)"
  - "Phase 52 W3: 6 kebab HTTP 写端点 (port_write_router.go:42-47) + network:port:write 权限 + audit 表"
  - "Phase 51 W2: PortWriteService/BatchResult Go struct 契约 (PortResult/BatchWriteRequest/BatchResult 形状)"
provides:
  - "components/network/port-write/PortWriteModal.tsx: 单端口 5 action 统一 Modal (action prop 切标题/字段/wrapper, reason Select+TextArea, description 特例, 审计跳转 Toast helper)"
  - "components/network/port-write/BulkWriteDrawer.tsx: 批量 Drawer select→executing→result 状态机 + indeterminate Spin + 三 Statistic + 失败明细 Table + 重试只取 failed + 跨设备预校验"
  - "pages/network/ports/index.tsx 改造: 操作列 (5 action ActionButtons) + 批量配置按钮 + canWrite gating + batchInProgress 双按钮禁用 + Modal/Drawer 挂载"
  - "pages/monitor/logs/index.tsx 改造: mount-only useEffect 读 URL ?module=xxx 预填 title 字段并触发 handleSearch"
affects:
  - "Phase 54 W5 UAT: 5 单端口操作 + 批量 + 重试 + 权限隔离 + 审计跳转的手动验证项落到本 plan 提供的入口"
tech-stack:
  added: []
  patterns:
    - "antd Modal+Form+Form.useForm+validateFields (替代 Modal.confirm, reason/description 输入需 Form)"
    - "antd Drawer 三态条件渲染 (select→executing→result), executing 阶段禁用 onClose/maskClosable/closable 防孤儿状态"
    - "indeterminate Spin 替代 Progress 伪造 X/Y (D-05: 后端 batch 同步阻塞无实时进度)"
    - "App.useApp() message.open({ content: <ReactNode> }) + react-router-dom Link 实现含跳转链接的 Toast"
    - "useTableManager + selectedRowKeys 联动 + 父组件 setBatchInProgress 状态上抛 (onExecutingChange callback)"
    - "useSearchParams + mount-only useEffect [] + eslint-disable exhaustive-deps 读 URL query 预填 form (LANDMINE #2)"
key-files:
  created:
    - "xingran-react-frontend/src/components/network/port-write/PortWriteModal.tsx"
    - "xingran-react-frontend/src/components/network/port-write/BulkWriteDrawer.tsx"
  modified:
    - "xingran-react-frontend/src/pages/network/ports/index.tsx"
    - "xingran-react-frontend/src/pages/monitor/logs/index.tsx"
decisions:
  - "单 Modal 覆盖 5 action (D-01), 不建 5 个独立 Modal 不用 Modal.confirm (reason/description 输入需 Form)"
  - "reason 字段预置 Select + __custom__ sentinel 切 TextArea (D-02), 双层 Form.Item shouldUpdate 监听"
  - "description action 特例: 新描述必填 + reason 可空 (D-03), 其他 4 action reason 必填 5-200 字符"
  - "批量 indeterminate Spin 不用 Progress 伪造 X/Y (D-05/ROADMAP #2 笔误纠正)"
  - "重试只取 batchResult.failed.map(p => p.portId), 不重试 skipped (D-06)"
  - "batchInProgress 同时禁用刷新+采集所有设备 (D-07/LANDMINE #4 同类竞态)"
  - "权限源 useMenuStore 非 useAuthStore (D-09/ROADMAP #4 笔误纠正), canWrite 控制操作列与批量配置按钮可见性"
  - "审计跳转路径 '/monitor/logs?module=端口管理' 非 '/monitor/operlog' (route 注册偏差, D-10)"
  - "BulkWriteDrawer 复用 PortWriteModal 的 showAuditLinkToast + AUDIT_LOG_PATH (避免抽 utils 超出 plan 范围)"
metrics:
  duration: "11 min"
  completed: "2026-07-07T05:47:59Z"
  tasks: 3
  files: 4
  commits: 3
requirements: [UI-01, UI-02, UI-03, UI-04, UI-05]
---

# Phase 53 Plan 02: PortWriteModal + BulkWriteDrawer + 端口页/monitor/logs 改造 Summary

把 Phase 53-01 落地的 6 wrapper + 4 类型 + 5 常量变成运维可点击的操作入口: 1 个统一单端口 Modal (5 action 共用) + 1 个批量 Drawer (select→executing→result 状态机 + indeterminate + 失败重试) + 端口列表页操作列/批量按钮/权限 gating/batchInProgress 禁用 + monitor/logs URL query 预填补丁。

## What Was Built

### 文件改动清单 (4 文件 / 2 新建 / 2 修改)

| 文件 | 类型 | 改动 | Commit |
|------|------|------|--------|
| `xingran-react-frontend/src/components/network/port-write/PortWriteModal.tsx` | CREATE | 单端口 5 action 统一 Modal + showAuditLinkToast helper + AUDIT_LOG_PATH 常量导出 | `ce795d02` |
| `xingran-react-frontend/src/components/network/port-write/BulkWriteDrawer.tsx` | CREATE | 批量 Drawer select→executing→result 状态机 + 跨设备预校验 + 三 Statistic + 失败明细 + 重试 | `5e80e309` |
| `xingran-react-frontend/src/pages/network/ports/index.tsx` | MODIFY | useMenuStore canWrite gating + 操作列 + 批量配置按钮 + batchInProgress 双禁 + Modal/Drawer 挂载 | `beca9a5e` |
| `xingran-react-frontend/src/pages/monitor/logs/index.tsx` | MODIFY | useSearchParams + mount-only useEffect 读 ?module=xxx 预填 title + handleSearch | `beca9a5e` |

附带 Rule 1 修复 (`beca9a5e` 一同提交): PortWriteModal/BulkWriteDrawer 移除 `JSX.Element` 显式返回类型 (`tsc -b` 严格模式全局 JSX 命名空间不可用, 改类型推断)。

### Task 1: PortWriteModal.tsx 单端口 5 action 统一 Modal

| 模块 | 实现 |
|------|------|
| **D-01 统一 Modal** | 1 个 Modal + `action: PortWriteAction` prop 切标题 (`ACTION_TITLE[action]` + interfaceName) / 切"新描述"字段 (仅 description action) / 切 wrapper 调用 |
| **D-02 reason Select+TextArea** | 外层 `reasonSelect` Select 绑 PRESET_REASONS (末项 `__custom__` sentinel); `Form.Item shouldUpdate` 监听, 选 sentinel 时展开 `reasonText` Input.TextArea (maxLength=REASON_MAX=200, showCount) |
| **D-03 description 特例** | action=description 时"新描述"必填 (maxLength=DESCRIPTION_MAX=80); reason 校验改 `validateReasonOptional` (可空); 其他 4 action 用 `validateReasonRequired` (必填 5-200) |
| **D-10 审计 Toast** | `showAuditLinkToast(message, navigate)` helper: `message.open({ type:'success', content: <span>操作成功，<Link to={AUDIT_LOG_PATH} onClick={()=>navigate(AUDIT_LOG_PATH)}>查看审计日志</Link></span> })`; AUDIT_LOG_PATH = `/monitor/logs?module=` + encodeURIComponent('端口管理') |
| **LANDMINE #5** | wrapper 调用不传 errorMessage 选项, validateFields 失败直接 return (antd 自动标红), wrapper reject 由 post() 拦截器已弹 Toast 不重复 |
| **useEffect 纪律** | `useEffect(() => { if (open) form.resetFields(); }, [open, action, form])` — deps 全稳定 (open/action 原始值, form 来自 useForm 稳定 ref) |

### Task 2: BulkWriteDrawer.tsx select→executing→result 状态机

| 视图 | 实现 |
|------|------|
| **select** | 只读汇总 2 Statistic (已选端口 / 唯一设备数); 跨设备预校验 Alert (uniqueDeviceIds.length > 1 时禁用提交); action Select (ACTION_OPTIONS 5 项); description action 时多"新描述"必填字段; reasonSelect + reasonText (同 Modal); 提交按钮 disabled 当 selectedPorts.length===0 或 isMixedDevices |
| **executing (D-05)** | `<Spin size="large" tip="正在批量配置...（预计 ~1s/端口）" />` indeterminate; 不用 Progress 伪造 X/Y (ROADMAP #2 笔误纠正: 后端 batch 同步阻塞 52 D-11 无实时进度) |
| **result** | 三 Statistic 卡片 Row+Col span=8 (✓ 成功绿 / ⚠ 跳过灰带"无需操作"Tag / ✗ 失败红); 失败明细 Table (dataSource=batchResult.failed, rowKey=portId, expandable.expandedRowRender 显示 commandSent); 跳过明细 Collapse (默认收起); 重试按钮 disabled={batchResult.failed.length===0} |
| **LANDMINE #3** | `batchResult.failed/succeeded/skipped` 从 HTTP 200 body 读 (HTTP 200 + status:failed 是正常 resolve), 不靠 .catch 分类; Promise reject 仅网络错误等才走 catch (post() 拦截器已弹 Toast) |
| **D-06 重试只取 failed** | `handleRetryFailed`: `failedIds = batchResult.failed.map(p => p.portId)` → buildRequest(lastAction, failedIds, lastDescription) → batchWritePorts → setBatchResult(newResult) 替换当前结果 (不累加历史); skipped 不重试 |
| **D-07 onExecutingChange** | `useEffect(() => { onExecutingChange?.(phase === 'executing'); }, [phase, onExecutingChange])` 上抛父组件, 父组件 setBatchInProgress 禁用刷新+采集按钮 (LANDMINE #4 同类竞态) |
| **T-53-07** | buildRequest 只发白名单字段 `{deviceId, action, portIds, description?}`, 不 spread 整个 port record; portIds 从 selectedRowKeys 派生 |
| **executing 防孤儿** | Drawer `onClose={isExecuting ? () => {} : onClose}` + `maskClosable={!isExecuting}` + `closable={!isExecuting}` 防用户中途关 Drawer 留下孤儿 batch 状态 |
| **CLAUDE.md useEffect 纪律** | open→reset effect deps=[open, form] (form 稳定); phase→onExecutingChange effect deps=[phase, onExecutingChange] (onExecutingChange 来自父 props 稳定); 无内联对象/数组 deps |

### Task 3: ports/index.tsx + monitor/logs/index.tsx 改造

**ports/index.tsx (5 处改动):**

| 改动 | 实现 |
|------|------|
| **改动 1 权限 gating (D-09)** | `import { useMenuStore }` + `const menuPermissions = useMenuStore((s) => s.permissions)` + `const hasPermission = (perm: string) => menuPermissions.includes(perm)` + `const canWrite = hasPermission("network:port:write")` — 非 useAuthStore (ROADMAP #4 笔误纠正) |
| **改动 2 Modal/Drawer state** | writeModalOpen/writeModalAction/writeModalRecord/bulkWriteDrawerOpen/batchInProgress 5 个 useState + openWriteModal(action, record) helper |
| **改动 3 操作列 (D-01)** | `...(canWrite ? [{ title:'操作', key:'portWriteAction', fixed:'right', width:100, render:(_, record) => <ActionButtons actions={[5 个 openWriteModal 调用]} /> }] : [])` — 无权限整列消失; ActionButtons >=3 项自动收纳到 Dropdown |
| **改动 4 顶部按钮 (D-04/D-07/LANDMINE #4)** | "刷新"和"采集所有设备"都加 `disabled={batchInProgress}` (≥2 处); 紧挨"批量删除(N)"加"批量配置(N)"按钮 `disabled={selectedRowKeys.length === 0 || !canWrite}` type="primary" icon=SettingOutlined |
| **改动 5 挂载两组件** | `<PortWriteModal open action portRecord onClose onSuccess={() => { loadPortStatus(); loadStatistics(); }} />` + `<BulkWriteDrawer open selectedPorts={portStatus.filter(p => selectedRowKeys.includes(p.id))} onClose onSuccess onExecutingChange={setBatchInProgress} />` |

**monitor/logs/index.tsx (改动 6, D-10/LANDMINE #2):**

```typescript
import { useLocation, useSearchParams } from "react-router-dom";
// ...
const [searchParams] = useSearchParams();

// mount-only 读 URL ?module=xxx 预填 title 字段并触发查询
useEffect(() => {
  const moduleFromUrl = searchParams.get("module");
  if (moduleFromUrl && activeTab === "oper") {
    operLogManager.searchForm.setFieldsValue({ title: moduleFromUrl });
    operLogManager.handleSearch();
  }
  // eslint-disable-next-line react-hooks/exhaustive-deps
}, []);
```

参考 `reconciliation/exceptions/index.tsx:88,145-172` URL→form 预填模式 + `[]` mount-only + eslint-disable (CLAUDE.md useEffect 纪律)。

## 5 LANDMINE 全部落地确认

| Landmine | 落地点 | AC grep | 结果 |
|----------|--------|---------|------|
| #1: types 在 network.ts | 53-01 已落地, 本 plan import `@/types/network` | — | ✅ |
| #2: monitor/logs URL 预填 | Task 3 改动 6 useSearchParams + setFieldsValue({title:}) + handleSearch | `useSearchParams` ≥ 1, `searchParams.get("module")` ≥ 1, `setFieldsValue({ title:` ≥ 1 | ✅ |
| #3: HTTP 200 + status:failed 不靠 catch | Task 2 batchResult.failed/succeeded/skipped 从 body 读; Promise reject 仅走 setPhase("select") | `batchResult.failed\|result.failed` ≥ 2 (实际 12) | ✅ |
| #4: batchInProgress 双按钮禁用 | Task 3 改动 4 刷新+采集所有设备均 `disabled={batchInProgress}` | `disabled={batchInProgress}` ≥ 2 (实际 2) | ✅ |
| #5: post 拦截器已弹 Toast 不重复 | Task 1 wrapper 调用不传 errorMessage 选项; Task 2 catch 仅 setPhase 不 message.error | PortWriteModal `errorMessage:` = 0 | ✅ |

## 2 ROADMAP 笔误纠正全部落地

| 笔误 | 纠正 | AC grep | 结果 |
|------|------|---------|------|
| D-05: ROADMAP 字面 "X/Y ports" | BulkWriteDrawer 用 indeterminate `<Spin>`, 不用 `<Progress>` 伪造 | `Spin` ≥ 1 (实际 6), `Progress` 实际 JSX 用法 = 0 (3 处匹配全在注释/变量名) | ✅ |
| D-09: ROADMAP 写 useAuthStore | ports/index.tsx 权限源 useMenuStore (非 useAuthStore) | `useMenuStore` ≥ 1 (实际 2), `useAuthStore` = 0 | ✅ |

## Acceptance Criteria Results

### Task 1 — PortWriteModal.tsx

| AC | Result |
|----|--------|
| 文件存在 | ✅ EXISTS (242 行) |
| `export function PortWriteModal` | ✅ 1 |
| `ACTION_TITLE[action]` ≥ 1 | ✅ 2 (标题 + 类型注释) |
| 5 wrapper 调用 ≥ 5 | ✅ 10 (5 import + 5 调用) |
| `REASON_MIN\|REASON_MAX\|DESCRIPTION_MAX` ≥ 2 | ✅ 16 |
| `__custom__` ≥ 1 | ✅ 5 |
| `monitor/logs?module` ≥ 1 | ✅ 2 |
| `useMenuStore\|authStore` = 0 | ✅ 0 |
| 不含 `Modal.confirm` | ✅ 0 实际调用 (1 处仅注释说明排除原因) |
| 不含 errorMessage option | ✅ 0 |
| `npm run type-check` 退出 0 | ✅ EXIT=0 |

### Task 2 — BulkWriteDrawer.tsx

| AC | Result |
|----|--------|
| 文件存在 | ✅ EXISTS (482 行) |
| `export function BulkWriteDrawer` | ✅ 1 |
| 状态机 `"select"\|"executing"\|"result"` ≥ 3 | ✅ 14 |
| `Spin` ≥ 1 | ✅ 6 |
| `Progress` = 0 (实际用法) | ✅ 0 (3 处匹配全在注释/变量名 batchInProgress) |
| `batchResult.failed\|result.failed` ≥ 2 | ✅ 12 |
| `onExecutingChange` ≥ 2 | ✅ 7 |
| `batchResult.failed.map\|result.failed.map` ≥ 1 | ✅ 2 |
| `Statistic` ≥ 3 | ✅ 8 |
| `monitor/logs?module` ≥ 1 | ✅ 1 (注释引用, 实际跳转复用 PortWriteModal showAuditLinkToast) |
| 跨设备预校验 (mixed\|唯一设备\|设备数\|same device) ≥ 1 | ✅ 11 |
| `batchWritePorts` ≥ 2 | ✅ 5 (import + 2 调用 + 注释) |
| `npm run type-check` 退出 0 | ✅ EXIT=0 |

### Task 3 — ports/index.tsx + monitor/logs/index.tsx

| AC | Result |
|----|--------|
| ports `useMenuStore` ≥ 1 | ✅ 2 |
| ports `useAuthStore` = 0 | ✅ 0 |
| ports `hasPermission("network:port:write")` ≥ 1 | ✅ 1 |
| ports `<PortWriteModal` ≥ 1 | ✅ 1 |
| ports `<BulkWriteDrawer` ≥ 1 | ✅ 1 |
| ports `批量配置` ≥ 1 | ✅ 2 |
| ports `disabled={batchInProgress}` ≥ 2 | ✅ 2 (刷新 + 采集所有设备) |
| ports `onExecutingChange={setBatchInProgress}` = 1 | ✅ 1 |
| ports `openWriteModal("action"` × 5 ≥ 5 | ✅ 5 |
| logs `useSearchParams` ≥ 1 | ✅ 2 |
| logs `searchParams.get("module")` ≥ 1 | ✅ 1 |
| logs `setFieldsValue({ title:` ≥ 1 | ✅ 1 |
| `npm run type-check` 退出 0 | ✅ EXIT=0 |
| `npm run build` 退出 0, vendor-react gzip ≤ 826 kB | ✅ EXIT=0, gzip=774.96 kB (Phase 48 baseline 776 + 50 容差, 与 53-01 baseline 完全一致 = 零回归) |

## Wave 2 Build Verification

| 检查项 | 结果 |
|--------|------|
| `npm run type-check` | ✅ EXIT=0 |
| `npm run build` | ✅ EXIT=0 (built in 36.56s) |
| vendor-react gzip ≤ 826 kB | ✅ 774.96 kB |
| Bundle delta vs 53-01 baseline | 0.00 kB (零回归, 仅加 2 组件 + 2 页面改造, 无新重型依赖) |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] 移除 JSX.Element 显式返回类型**
- **Found during:** Task 3 build 验证 (`tsc -b` 严格模式)
- **Issue:** PortWriteModal/BulkWriteDrawer 用 `: JSX.Element` 显式返回类型注解, `tsc -b` 严格模式下全局 JSX 命名空间不可用 (React 19 + 本项目 tsconfig 配置), 报 `TS2503: Cannot find namespace 'JSX'`
- **Fix:** 移除 4 处 `JSX.Element` 显式返回类型, 改为类型推断 (函数组件返回类型本就由 TSX 推断, 显式注解多余)
- **Files modified:** PortWriteModal.tsx (1 处), BulkWriteDrawer.tsx (3 处: BulkWriteDrawer 主 + SelectView + ResultView 子组件)
- **Commit:** `beca9a5e` (与 Task 3 一同提交, 因 build 失败阻塞 Task 3 验证)

**说明:** `npx tsc --noEmit` (IDE/standalone 模式) 与 `tsc -b` (build 模式, npm run build 走此路径) 对全局 JSX 命名空间处理不同, 前者能通过后者报错。本项目惯例 (`src/components/shared/`) 用 `React.FC` 或不写返回类型, 本 plan 选不写返回类型 (最少代码 + 类型推断)。

## Known Stubs

无 — 本 plan 4 文件全部接入真实数据源 (useTableManager portStatus / selectedRowKeys / networkApi 6 wrapper / useMenuStore permissions), 无 placeholder/mock/hardcoded-empty 流向 UI 的字段。

## Threat Flags

无新增安全相关表面超出 plan `<threat_model>` 范围。

- T-53-05 (前端 canWrite 绕过): accept — 后端 RequirePermissions 是真相源, 已落地 Phase 52 组级中间件
- T-53-06 (XSS): mitigate — 全部 antd 组件文本渲染, 0 处 dangerouslySetInnerHTML (acceptance_criteria grep 已验证)
- T-53-07 (BatchWriteRequest 字段构造): mitigate — buildRequest 白名单字段, portIds 从 selectedRowKeys 派生
- T-53-08 (commandSent 信息披露): accept — 审计字段仅在失败明细 Table 渲染, 不入 console.log
- T-53-09 (URL query module): mitigate — 仅预填 form.title 触发查询, 不直接拼 SQL (post() 走 request body), 空值安全降级
- T-53-10 (重试无限循环): mitigate — 重试按钮 disabled={failed.length===0}, 每次范围只含上次 failed (自然收敛)

## Self-Check: PASSED

- FOUND: xingran-react-frontend/src/components/network/port-write/PortWriteModal.tsx
- FOUND: xingran-react-frontend/src/components/network/port-write/BulkWriteDrawer.tsx
- FOUND: xingran-react-frontend/src/pages/network/ports/index.tsx
- FOUND: xingran-react-frontend/src/pages/monitor/logs/index.tsx
- FOUND: ce795d02 (Task 1 commit)
- FOUND: 5e80e309 (Task 2 commit)
- FOUND: beca9a5e (Task 3 commit)

## 留给 Phase 54 W5 UAT 的手动验证项清单

- 无 network:port:write 权限账号登录 → 端口页无操作列无批量配置按钮 (canWrite gating)
- 有权限账号行内 5 操作 (shutdown/undo_shutdown/description/dot1x_enable/dot1x_disable) → PortWriteModal 弹出 → 标题含 ACTION_TITLE[action] + interfaceName
- 非 description action 时 reason 必填校验 (选预设项直接提交 OK; 选"其他..."不填 TextArea 校验失败)
- description action 时"新描述"必填 + reason 可空
- 提交成功 → Toast 含"查看审计日志"链接 → 点击跳 `/monitor/logs?module=端口管理` → title 字段自动预填"端口管理" + 自动 handleSearch
- 勾选多端口 → 批量配置(N) → BulkWriteDrawer → 选 action + reason → 提交 → indeterminate Spin → 结果面板
- 跨设备勾选 → Drawer 显示 Alert "批量必须同设备" + 禁用提交
- 结果面板三 Statistic + 失败明细 Table (可展开看 commandSent) + 跳过折叠
- 失败端口重试 → 范围只含 failed, 不含 succeeded/skipped
- 批量进行中 → 端口页"刷新"和"采集所有设备"按钮均 disabled (LANDMINE #4)
- 批量进行中 Drawer 关闭按钮/mask 点击无响应 (防孤儿状态)
