# Phase 53: W4 — Frontend Drawer + Progress Dialog + API Wrappers - Pattern Map

**Mapped:** 2026-07-07
**Files analyzed:** 7 (4 modify / 3 create; +1 conditional modify)
**Analogs found:** 7 / 7 (all target files have a strong in-repo analog)

> CONTEXT.md 已精确到行号引用各 analog，本 PATTERNS.md 把这些引用转译成可直接复制到 PLAN.md 的代码块。下游 planner 可直接抄 excerpts 而不必重读全文件。

---

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `xingran-react-frontend/src/lib/api/networkApi.ts` (MODIFY) | service / api-wrapper | request-response | `xingran-react-frontend/src/lib/api/networkApi.ts` 自身 (queryMACHistory line 85-93) | exact (同文件扩展) |
| `xingran-react-frontend/src/components/network/port-write/PortWriteModal.tsx` (CREATE) | component (Modal+Form) | request-response | `xingran-react-frontend/src/pages/duty/management/modals/ManualScheduleModal.tsx` | role-match (Modal+Form+Select reason) |
| `xingran-react-frontend/src/components/network/port-write/BulkWriteDrawer.tsx` (CREATE) | component (Drawer+结果面板) | batch → request-response | `xingran-react-frontend/src/components/reconciliation/ReconciliationDrawer.tsx` | role-match (Drawer 结构) |
| `xingran-react-frontend/src/components/network/port-write/constants.ts` (CREATE) | config (常量) | n/a | `xingran-react-frontend/src/pages/duty/management/constants.ts` (MANUAL_REASON_OPTIONS) | exact |
| `xingran-react-frontend/src/pages/network/ports/index.tsx` (MODIFY) | page (Table+操作列+批量按钮) | CRUD/list | 自身 + `xingran-react-frontend/src/pages/network/devices/index.tsx` (操作列 line 193-260) | exact |
| `xingran-react-frontend/src/types/network.ts` (MODIFY; CONTEXT 备选 `types/index.ts`) | model / types | n/a | `xingran-react-frontend/src/types/network.ts` (DevicePortStatus line 117-137) | exact (类型应放此文件，与 DevicePortStatus 同处) |
| `xingran-react-frontend/src/pages/monitor/logs/index.tsx` (CONDITIONAL MODIFY) | page (operlog) | request-response | 自身 + `xingran-react-frontend/src/pages/asset/reconciliation/exceptions/index.tsx` (URL query 预填 line 145-166) | role-match (URL→form 预填模式) |

**说明 / 偏离 CONTEXT 的修正：**
- CONTEXT `<code_context>` Integration Point #6 写 "types/index.ts (或 networkApi.ts inline)"。实际项目 `src/types/index.ts` 只是 barrel (`export * from "./network"`)，真实类型定义在 `src/types/network.ts`。新增 `PortResult / BatchResult / BatchWriteRequest` **必须加到 `src/types/network.ts`**（与 `DevicePortStatus` 同处），planner 不要写进 `index.ts`。

---

## Pattern Assignments

### `xingran-react-frontend/src/lib/api/networkApi.ts` (MODIFY: +6 write wrappers)

**Analog:** 同文件 `queryMACHistory` (line 85-93)

**Imports pattern** (line 1-2, 已存在，复用)：
```typescript
import axios, { type AxiosInstance } from "axios";
import { post } from "../api";
```

**Wrapper 写法模板** (复制 `queryMACHistory` line 85-93 的 `post<T>(url, body)` + `result.data!` 解包风格)：
```typescript
// 已存在的写法（queryMACHistory, line 85-93）— 新 wrapper 镜像此风格
export const queryMACHistory = async (
  params: MACHistoryQueryParams
): Promise<MACHistoryPageResult> => {
  const result = await post<MACHistoryPageResult>(
    "/network/history/list",
    params
  );
  return result.data!;
};

// ===== Phase 53 新增 6 个 wrapper（D-08 锁定签名）=====
import type { PortResult, BatchResult, BatchWriteRequest } from "@/types/network";

export const writeShutdown = async (portId: string, reason: string): Promise<PortResult> => {
  const result = await post<PortResult>("/network/ports/write/shutdown", { portId, reason });
  return result.data!;
};

export const writeUndoShutdown = async (portId: string, reason: string): Promise<PortResult> => {
  const result = await post<PortResult>("/network/ports/write/undo-shutdown", { portId, reason });
  return result.data!;
};

export const writeDescription = async (
  portId: string,
  description: string,
  reason?: string
): Promise<PortResult> => {
  const result = await post<PortResult>("/network/ports/write/description", {
    portId,
    description,
    reason,
  });
  return result.data!;
};

export const writeDot1xEnable = async (portId: string, reason: string): Promise<PortResult> => {
  const result = await post<PortResult>("/network/ports/write/dot1x-enable", { portId, reason });
  return result.data!;
};

export const writeDot1xDisable = async (portId: string, reason: string): Promise<PortResult> => {
  const result = await post<PortResult>("/network/ports/write/dot1x-disable", { portId, reason });
  return result.data!;
};

export const batchWritePorts = async (req: BatchWriteRequest): Promise<BatchResult> => {
  const result = await post<BatchResult>("/network/ports/write/batch", req);
  return result.data!;
};
```

**Default export 更新** (line 241-246)：
```typescript
export default {
  queryMACHistory,
  getMACEvents,
  exportMACHistory,
  batchExport,
  // Phase 53 新增
  writeShutdown,
  writeUndoShutdown,
  writeDescription,
  writeDot1xEnable,
  writeDot1xDisable,
  batchWritePorts,
};
```

**post() 底座** (`xingran-react-frontend/src/lib/api.ts` line 468-470) — wrapper 不直接碰，但 planner 须知它已内置 SM2+SM4 加密 + token refresh + BaseResponse envelope 解包 + `getAppMessage().error(data.message)` 错误 Toast：
```typescript
export function post<T = unknown>(url: string, data?: unknown): Promise<BaseResponse<T>> {
  return api.post(url, data);
}
```
错误路径 (`api.ts` line 362-364)：非 0 code 自动 `getAppMessage().error(data.message)` + `Promise.reject(new Error(data.message))`。**这是 D-08 "wrapper 不翻译 sentinel" 的依据** — 后端 handler 已把 sentinel 翻成中文 `message`，`post()` 拦截器统一弹 Toast，wrapper 只透传 reject。

---

### `xingran-react-frontend/src/components/network/port-write/PortWriteModal.tsx` (CREATE)

**Analog:** `xingran-react-frontend/src/pages/duty/management/modals/ManualScheduleModal.tsx`（Modal+Form+Select reason 模式）

**Modal+Form 骨架** (复制 ManualScheduleModal.tsx line 19-71)：
```typescript
import { Modal, Form, Select, Input } from "antd";

interface PortWriteModalProps {
  open: boolean;
  action: "shutdown" | "undo_shutdown" | "description" | "dot1x_enable" | "dot1x_disable";
  portRecord: DevicePortStatus | null;
  onClose: () => void;
  onSuccess: () => void;
}

export function PortWriteModal({ open, action, portRecord, onClose, onSuccess }: PortWriteModalProps) {
  const [form] = Form.useForm();
  const { message } = App.useApp();  // ← D-10 Toast 链接用

  const handleOk = async () => {
    try {
      const values = await form.validateFields();
      // 按 action 分支调对应 wrapper（D-01）
      // ... 成功后 message.open(...) 带"查看审计日志"链接（D-10）
      form.resetFields();
      onSuccess();
    } catch (err) {
      if (err && typeof err === "object" && "errorFields" in err) return; // validateFields 失败
      // post() 拦截器已弹 Toast，这里不重复
    }
  };

  return (
    <Modal
      title={`${ACTION_TITLE[action]} - ${portRecord?.interfaceName ?? ""}`}  // D-01 动态标题
      open={open}
      onOk={handleOk}
      onCancel={onClose}
      destroyOnHidden
      width={520}
    >
      <Form form={form} layout="vertical">
        {/* D-03: description action 特例 — 显示"新描述"必填输入框 */}
        {action === "description" && (
          <Form.Item
            name="description"
            label="新描述"
            rules={[
              { required: true, message: "请输入新端口描述" },
              { max: 80, message: "描述不超过 80 字符" },  // D-03 保守上限
            ]}
          >
            <Input placeholder="请输入新端口描述" maxLength={80} showCount />
          </Form.Item>
        )}

        {/* D-02: reason 字段 — 预置 Select + 其他 TextArea */}
        <Form.Item
          name="reason"
          label="操作原因"
          // D-03: description action 时 reason 可空；其他 4 action 必填
          rules={action === "description"
            ? [{ validator: validateReasonOptional }]
            : [{ required: true, validator: validateReasonRequired }]}
        >
          <ReasonInput />  {/* 内部封装 Select 切换 TextArea 逻辑 */}
        </Form.Item>
      </Form>
    </Modal>
  );
}
```

**Reason 字段 Select+TextArea 切换** — 参考 `MANUAL_REASON_OPTIONS` (`duty/management/constants.ts` line 15-20) 的 `{label, value}` 数组结构，用 antd `Form.Item noStyle` + `shouldUpdate` 监听 reasonSelect 字段切换。

**Modal confirm + Popconfirm 替代** — CONTEXT D-01 明确 "不用 antd Modal.confirm（reason 字段 + description 输入需 Form，confirm 不够）"，必须用 `<Modal>` + `<Form>` 走 `form.validateFields()`。

---

### `xingran-react-frontend/src/components/network/port-write/BulkWriteDrawer.tsx` (CREATE)

**Analog:** `xingran-react-frontend/src/components/reconciliation/ReconciliationDrawer.tsx`（Drawer + 内部分区结构）

**Drawer 骨架** (复制 ReconciliationDrawer.tsx line 21-22 import + line 55-66 props)：
```typescript
import { Drawer, Spin, Statistic, Table, Tag, Collapse, Button, Space, Typography } from "antd";

interface BulkWriteDrawerProps {
  open: boolean;
  selectedPorts: DevicePortStatus[];   // D-04: Drawer 不重选，直接读父组件 selectedRowKeys
  onClose: () => void;
  onSuccess: () => void;
  onExecutingChange?: (inProgress: boolean) => void;  // D-07: 上抛 batchInProgress 给父组件禁刷新
}

// 状态机（D-06）：select → executing → result → (retry) → executing → result'
type DrawerPhase = "select" | "executing" | "result";
```

**Drawer 主体三态切换** (基于 ReconciliationDrawer.tsx 的 `<Drawer>` + 内部条件渲染)：
```typescript
<Drawer
  title="批量配置端口"
  open={open}
  onClose={onClose}
  width={720}
  destroyOnHidden
>
  {phase === "select" && <SelectView ... />}
  {phase === "executing" && (
    <Spin tip="正在批量配置...（预计 ~1s/端口）" />  // D-05 indeterminate，不伪造 X/Y
  )}
  {phase === "result" && (
    <ResultView
      result={batchResult}
      onRetry={handleRetryFailed}   // D-06 一键重试 failed
    />
  )}
</Drawer>
```

**结果面板三 Statistic 卡片** — 参考 ports/index.tsx line 322-362（已用 Row+Col+Statistic 的模式）：
```typescript
<Row gutter={16}>
  <Col span={8}>
    <Card><Statistic title="✓ 成功" value={result.succeeded.length}
      valueStyle={{ color: "var(--theme-success, #3f8600)" }} /></Card>
  </Col>
  <Col span={8}>
    <Card><Statistic title="⚠ 跳过" value={result.skipped.length}
      valueStyle={{ color: "var(--theme-text-secondary, #8c8c8c)" }} /></Card>
  </Col>
  <Col span={8}>
    <Card><Statistic title="✗ 失败" value={result.failed.length}
      valueStyle={{ color: "var(--theme-error, #cf1322)" }} /></Card>
  </Col>
</Row>

{/* 失败明细 Table — 参考 ports/index.tsx expandable (line 466-484) 展开行展示 commandSent */}
<Table
  dataSource={result.failed}
  rowKey="portId"
  expandable={{
    expandedRowRender: (port) => (
      <Typography.Text type="secondary" code>
        {port.commandSent}
      </Typography.Text>
    ),
  }}
  columns={[
    { title: "接口名", dataIndex: "portId" /* 或 JOIN interfaceName */, width: 150 },
    { title: "错误原因", dataIndex: "error", ellipsis: true },
  ]}
/>

{/* 跳过明细折叠 — D-05 默认收起 */}
<Collapse items={[{ key: "skipped", label: `跳过明细 (${result.skipped.length})`, children: <...> }]} />

{/* D-06 重试按钮 */}
<Button
  type="primary"
  disabled={result.failed.length === 0}
  onClick={() => handleRetryFailed(result.failed.map(p => p.portId))}
>
  重试失败端口 ({result.failed.length})
</Button>
```

---

### `xingran-react-frontend/src/components/network/port-write/constants.ts` (CREATE)

**Analog:** `xingran-react-frontend/src/pages/duty/management/constants.ts` (MANUAL_REASON_OPTIONS line 15-20)

**完整内容**（复制 MANUAL_REASON_OPTIONS 的 `{label, value} as const` 结构）：
```typescript
/**
 * PortWriteModal / BulkWriteDrawer 共享常量
 */

/** D-02 预置操作原因（5 字符下限已满足） */
export const PRESET_REASONS = [
  { label: "故障排查", value: "故障排查" },
  { label: "安全合规", value: "安全合规" },
  { label: "业务变更", value: "业务变更" },
  { label: "临时测试", value: "临时测试" },
  { label: "其他...", value: "__custom__" },   // 选此项时展开 TextArea
] as const;

/** D-01 5 个 action 的中文标题 */
export const ACTION_TITLE: Record<
  "shutdown" | "undo_shutdown" | "description" | "dot1x_enable" | "dot1x_disable",
  string
> = {
  shutdown: "关闭端口",
  undo_shutdown: "取消关闭",
  description: "修改描述",
  dot1x_enable: "启用 802.1X",
  dot1x_disable: "停用 802.1X",
};

/** D-02 reason 校验上下限 */
export const REASON_MIN = 5;
export const REASON_MAX = 200;
export const DESCRIPTION_MAX = 80;  // D-03 保守上限
```

---

### `xingran-react-frontend/src/pages/network/ports/index.tsx` (MODIFY)

**Analogs:**
- 自身（顶部按钮区 line 410-433 / Table rowSelection line 455-458 / handleRefresh line 91 / handleBatchDelete line 181-199）
- `xingran-react-frontend/src/pages/network/devices/index.tsx` 操作列 (line 193-249)

#### 改动 1: 权限 gating 引入 (D-09)

**Analog:** `devices/index.tsx` line 260-261 + `MACHistoryPage.tsx` line 103-105

```typescript
import { useMenuStore } from "@/store/menuStore";

// 组件内顶部（与 devices/index.tsx line 260-261 完全一致）
const menuPermissions = useMenuStore((s) => s.permissions);
const hasPermission = (perm: string) => menuPermissions.includes(perm);
const canWrite = hasPermission("network:port:write");  // D-09
```

#### 改动 2: 操作列 (新增到 columns 数组)

**Analog:** `devices/index.tsx` `getDeviceTableColumns` line 192-249 (ActionButtons pattern)

复用现有 `ActionButtons` (`@/components/shared/ActionButtons`) 组件，参考 devices/index.tsx line 198-247 的 `actions` 数组写法：
```typescript
import ActionButtons, { type ActionButton } from "@/components/shared/ActionButtons";

// columns 末尾追加（镜像 devices/index.tsx line 192-249）
...(canWrite ? [{
  title: "操作",
  key: "portWriteAction",
  fixed: "right" as const,
  width: 100,
  render: (_: unknown, record: DevicePortStatus) => {
    const actions: ActionButton[] = [
      { key: "shutdown", label: "关闭端口", onClick: () => openWriteModal("shutdown", record) },
      { key: "undo_shutdown", label: "取消关闭", onClick: () => openWriteModal("undo_shutdown", record) },
      { key: "description", label: "修改描述", onClick: () => openWriteModal("description", record) },
      { key: "dot1x_enable", label: "启用 802.1X", onClick: () => openWriteModal("dot1x_enable", record) },
      { key: "dot1x_disable", label: "停用 802.1X", onClick: () => openWriteModal("dot1x_disable", record) },
    ];
    return <ActionButtons actions={actions} />;  // >=3 个自动收纳到 Dropdown
  },
}] : []),
```

#### 改动 3: 顶部"批量配置(N)"按钮 (D-04)

**Analog:** 自身 line 431-433 "批量删除(N)" 按钮

紧挨现有"批量删除"按钮加一个完全对齐 UX 的按钮：
```typescript
import { SettingOutlined } from "@ant-design/icons";
const [bulkWriteDrawerOpen, setBulkWriteDrawerOpen] = useState(false);
const [batchInProgress, setBatchInProgress] = useState(false);  // D-07

<Button
  icon={<SettingOutlined />}
  onClick={() => setBulkWriteDrawerOpen(true)}
  disabled={selectedRowKeys.length === 0}   // 与"批量删除(N)"完全一致
  type="primary"
>
  批量配置 ({selectedRowKeys.length})
</Button>
```

#### 改动 4: batchInProgress 禁用刷新 + 采集按钮 (D-07)

**Analog:** 自身 line 406 `<Button onClick={handleRefresh}>刷新</Button>` + line 428-430 `采集所有设备` 按钮

```typescript
<Button
  icon={<ReloadOutlined />}
  onClick={handleRefresh}
  disabled={batchInProgress}   // ← 新增
>
  刷新
</Button>

<Button
  icon={<CloudSyncOutlined />}
  onClick={handleCollectAll}
  loading={collecting}
  disabled={batchInProgress}   // ← 新增（D-07 同时禁用，同类竞态）
>
  采集所有设备
</Button>
```

#### 改动 5: 挂载 Modal + Drawer

```typescript
<PortWriteModal
  open={writeModalOpen}
  action={writeModalAction}
  portRecord={writeModalRecord}
  onClose={() => setWriteModalOpen(false)}
  onSuccess={() => { loadPortStatus(); loadStatistics(); }}
/>
<BulkWriteDrawer
  open={bulkWriteDrawerOpen}
  selectedPorts={portStatus.filter(p => selectedRowKeys.includes(p.id))}
  onClose={() => setBulkWriteDrawerOpen(false)}
  onSuccess={() => { loadPortStatus(); loadStatistics(); }}
  onExecutingChange={setBatchInProgress}   // D-07
/>
```

#### Toast 链接（D-10）写入 helper

参见下方 [Shared Patterns → 审计跳转 Toast 链接] 章节。

---

### `xingran-react-frontend/src/types/network.ts` (MODIFY: +3 类型)

**Analog:** 同文件 `DevicePortStatus` (line 117-137)

**新增类型**（追加到文件末尾；镜像后端 Go struct `internal/services/portwrite/port_write_service.go` line 34-50 + `batch_orchestrator.go` line 15-19）：
```typescript
/**
 * Phase 53 端口写操作结果（镜像后端 portwrite.PortResult）
 * Source: internal/services/portwrite/port_write_service.go:34-42
 */
export interface PortResult {
  portId: string;
  action: string;            // "shutdown" | "undo_shutdown" | "description" | "dot1x_enable" | "dot1x_disable"
  status: "succeeded" | "failed" | "skipped";
  noOp: boolean;
  currentState?: string;
  error?: string;
  commandSent?: string;
}

/**
 * Phase 53 批量写请求（镜像后端 portwrite.BatchWriteRequest）
 * Source: internal/services/portwrite/port_write_service.go:45-50
 */
export interface BatchWriteRequest {
  deviceId: string;
  action: "shutdown" | "undo_shutdown" | "description" | "dot1x_enable" | "dot1x_disable";
  portIds: string[];
  description?: string;   // 仅 action=description 使用
}

/**
 * Phase 53 批量写结果（镜像后端 portwrite.BatchResult）
 * Source: internal/services/portwrite/batch_orchestrator.go:15-19
 */
export interface BatchResult {
  succeeded: PortResult[];
  failed: PortResult[];
  skipped: PortResult[];
}
```

**后端 PortResult struct 原文** (`internal/services/portwrite/port_write_service.go` line 34-42，前端 TS 字段名按 JSON tag 镜像)：
```go
type PortResult struct {
    PortID       string `json:"portId"`
    Action       Action `json:"action"`
    Status       string `json:"status"` // "succeeded" | "failed" | "skipped"
    NoOp         bool   `json:"noOp"`
    CurrentState string `json:"currentState,omitempty"`
    Error        string `json:"error,omitempty"`
    CommandSent  string `json:"commandSent,omitempty"`
}
```

**后端 BatchResult struct 原文** (`internal/services/portwrite/batch_orchestrator.go` line 15-19)：
```go
type BatchResult struct {
    Succeeded []PortResult `json:"succeeded"`
    Failed    []PortResult `json:"failed"`
    Skipped   []PortResult `json:"skipped"`
}
```

---

### `xingran-react-frontend/src/pages/monitor/logs/index.tsx` (CONDITIONAL MODIFY)

**Planner 必须 先确认**：当前 `monitor/logs/index.tsx` **不读取** URL query params。完整文件 (470 行) 未出现 `useSearchParams`，搜索表单字段 (line 214-256) 只绑定到 `operLogManager.searchForm`，未从 `location.search` 注入初值。

**这意味着** D-10 的 `navigate('/monitor/logs?module=端口管理')` 跳过去后 `title` 字段是空的，预填不会发生。

**改造需求**（参考 `asset/reconciliation/exceptions/index.tsx` line 145-172 的 URL→filterValues 预填模式）：

**Analog:** `reconciliation/exceptions/index.tsx` line 88 (`useSearchParams`) + line 145-166 (URL query → state 初始化) + line 168-172 (useEffect 同步到 form)

新增 useEffect 读 `?module=xxx` 预填 `title` form 字段：
```typescript
import { useSearchParams } from "react-router-dom";

const [searchParams] = useSearchParams();

// 仅在 oper tab 时执行一次（mount）
useEffect(() => {
  const moduleFromUrl = searchParams.get("module");
  if (moduleFromUrl && activeTab === "oper") {
    operLogManager.searchForm.setFieldsValue({ title: moduleFromUrl });
    operLogManager.handleSearch();   // 触发带筛选的查询
  }
  // eslint-disable-next-line react-hooks/exhaustive-deps
}, []);   // mount-only
```

**注意 — `title` vs `module`**：后端 operlog 字段是 `title`（操作模块），CONTEXT D-10 写 `?module=端口管理` 是 URL query 参数名（前端自定义），form 字段名是 `title`（对应后端 `sys_oper_log.title` 列）。两者不冲突。

**Backend handler 透传**：后端 `POST /monitor/oper-logs/list` 已支持 `title` 字段过滤（既有的 search form 即用此字段），无需后端改动。

---

## Shared Patterns

### 权限 gating (D-09) — 跨端口页 + PortWriteModal + BulkWriteDrawer

**Source:** `xingran-react-frontend/src/pages/network/devices/index.tsx` line 59-60 import + line 260-261 使用；`MACHistoryPage.tsx` line 52 import + line 103-105 使用

**Apply to:** `ports/index.tsx` (canWrite)、`PortWriteModal.tsx` (无需，入口已 gate)、`BulkWriteDrawer.tsx` (无需，入口已 gate)

```typescript
// import (devices/index.tsx:59 / MACHistoryPage.tsx:52)
import { useMenuStore } from "@/store/menuStore";

// 使用 (devices/index.tsx:260-261 / MACHistoryPage.tsx:103-105)
const menuPermissions = useMenuStore((s) => s.permissions);
const hasPermission = (perm: string) => menuPermissions.includes(perm);
const canWrite = hasPermission("network:port:write");
```

**⚠ CONTEXT D-09 明确纠正 ROADMAP Success Criteria #4 的笔误**：权限源是 `useMenuStore`，不是 `useAuthStore`。`authStore` 只管 token/refresh，**不**含 permissions 数组。

**Permission 字符串来源**：`pkg/permission/config.go:189` `NetworkPortWrite = "network:port:write"`（52 migration_202 GrantNewMenuToRolesHavingParent 已 seed）。

---

### 顶部"批量X(N)"按钮 + rowSelection (D-04)

**Source:** `ports/index.tsx` line 431-433（"批量删除(N)" 模板）

**Apply to:** `ports/index.tsx` 新增"批量配置(N)"按钮

```typescript
// rowSelection 已就绪 (ports/index.tsx:455-458)
<Table
  rowSelection={{
    selectedRowKeys,
    onChange: setSelectedRowKeys,
  }}
  ...
/>

// 顶部按钮区 (ports/index.tsx:431-433 模板)
<Button
  icon={<DeleteOutlined />}
  onClick={handleBatchDelete}
  disabled={selectedRowKeys.length === 0}   // ← 关键：与 rowSelection 联动
  style={{ color: "var(--theme-error, #ff4d4f)" }}
>
  批量删除 ({selectedRowKeys.length})
</Button>
```

`useTableManager` 已暴露 `selectedRowKeys / setSelectedRowKeys` (useTableManager.ts line 41, 53) — 批量配置按钮直接读 `selectedRowKeys`，无需新建 state。

---

### 错误处理 / Toast (withErrorHandling + App.useApp)

**Source:** `ports/index.tsx` line 33 import + line 162-178 (`handleCollectAll` 用法) + line 44 `const { message } = App.useApp();`

**Apply to:** 单端口操作 success Toast + D-10 链接 Toast

```typescript
// 单端口操作 — 复用 ports/index.tsx:162-178 withErrorHandling 模式
await withErrorHandling(
  async () => {
    const result = await writeShutdown(portRecord.id, reason);
    return result;
  },
  {
    onSuccess: (result) => {
      // D-10 链接 Toast（见下一节）
      showAuditLinkToast(message, navigate);
      loadPortStatus();
      loadStatistics();
    },
  }
);
```

`withErrorHandling` 签名 (`errorHandler.ts` line 151-174)：成功路径调 `options.onSuccess?.(result)`；失败路径调 `ErrorHandler.handleApiError` (弹 Toast) + `options.onError?.(error)` + 返回 `null`。**注意**：当 `post()` 拦截器已弹 Toast 时，业务层 withErrorHandling 的 errorMessage 选项不要再写一遍（避免双重 Toast）。

---

### 审计跳转 Toast 链接 (D-10)

**Source:** `asset/reconciliation/exceptions/index.tsx` line 350-355 (`<Link to="/monitor/logs">查看日志</Link>`)

**Apply to:** PortWriteModal / BulkWriteDrawer 的 success Toast

**关键约束**：
- **路由路径必须是 `/monitor/logs`**（不是 `/monitor/operlog`，也不是后端 API `/monitor/oper-logs/list`）。exceptions/index.tsx line 340-349 注释明确：React Router 注册的实际 path 是 `monitor/logs`，跳 `/monitor/oper-logs` 或 `/logs` 都会 fallback 到 `/dashboard`（memory `xingran-dynamic-route-menu-not-seeded-fallback-dashboard` 关联坑）。
- antd `message.success(text)` 不支持链接，必须用 `App.useApp()` 的 `message.open({ content: <ReactNode> })`。

```typescript
import { Link } from "react-router-dom";

// helper（建议放 PortWriteModal/BulkWriteDrawer 共享工具或就近一份）
function showAuditLinkToast(
  message: App.useApp returns,
  navigate: (path: string) => void,
) {
  message.open({
    type: "success",
    duration: 5,
    content: (
      <span>
        操作成功，<Link to="/monitor/logs?module=端口管理" onClick={() => navigate("/monitor/logs?module=" + encodeURIComponent("端口管理"))}>查看审计日志</Link>
      </span>
    ),
  });
}
```

**注**：上面 `<Link>` 的 `to` 用 URL-encoded value 更稳（CONTEXT D-10 已写 `encodeURIComponent`）；中文 `端口管理` 是 Phase 52 handler `ModulePortWrite` 常量 (`internal/api/v1/network/port_write_handler.go:25`)，与 `sys_oper_log.title` 列匹配。

---

### 后端契约 (wrapper URL + body shape 对齐)

**Sources:**
- 路由：`internal/api/v1/network/port_write_router.go:42-47`
- request struct：`internal/api/v1/network/port_write_handler.go:59-63` (`PortWriteRequest`) + `internal/services/portwrite/port_write_service.go:45-50` (`BatchWriteRequest`)
- sentinel → HTTP 翻译：`internal/api/v1/network/port_write_handler.go:147-156` (单端口) + line 200-215 (batch)

**6 端点 URL（kebab 命名，与现有 /list /collect /batch-delete 同风格）**：
| Method | Path | Request Body | Response Data |
|--------|------|--------------|---------------|
| POST | `/network/ports/write/shutdown` | `{portId, reason}` | `PortResult` |
| POST | `/network/ports/write/undo-shutdown` | `{portId, reason}` | `PortResult` |
| POST | `/network/ports/write/description` | `{portId, description, reason?}` | `PortResult` |
| POST | `/network/ports/write/dot1x-enable` | `{portId, reason}` | `PortResult` |
| POST | `/network/ports/write/dot1x-disable` | `{portId, reason}` | `PortResult` |
| POST | `/network/ports/write/batch` | `{deviceId, action, portIds[], description?}` | `BatchResult` |

**Sentinel → 中文 message（前端透传，不翻译）**：
- `ErrPortNotFound` → 404 "端口不存在"
- `ErrDeviceNotFound` → 404 "设备不存在"
- `ErrBatchTooLarge` → 400 "批量端口数超过上限 50"
- `ErrEmptyBatch` → 400 "批量端口列表为空"
- `ErrMixedDevices` → 400 "批量端口必须属于同一设备"

**关键：HTTP 200 + `PortResult.status="failed"`**（SSH 执行失败但 nil sentinel err）路径走 audit + operlog + 200，前端 wrapper 拿到的是 `result.status="failed"` 的成功 Promise。BulkWriteDrawer 结果面板必须按 `result.failed[].error` 展示，而不是 catch Promise reject。

---

## No Analog Found

无。所有 7 个目标文件都有强 in-repo analog。Planner 不必回退到 RESEARCH.md 通用模板。

---

## Metadata

**Analog 搜索范围：**
- `xingran-react-frontend/src/lib/api/` (networkApi.ts, api.ts)
- `xingran-react-frontend/src/pages/network/{ports,devices,mac/history}/`
- `xingran-react-frontend/src/pages/duty/management/{modals,constants}/`
- `xingran-react-frontend/src/pages/asset/reconciliation/exceptions/`
- `xingran-react-frontend/src/pages/monitor/logs/`
- `xingran-react-frontend/src/components/{shared,reconciliation,network}/`
- `xingran-react-frontend/src/types/network.ts`
- `xingran-react-frontend/src/hooks/useTableManager.ts`
- `xingran-react-frontend/src/utils/errorHandler.ts`
- `internal/services/portwrite/` (后端契约镜像)
- `internal/api/v1/network/port_write_{router,handler}.go` (后端契约镜像)

**Files scanned:** 14 (含自身)
**Pattern extraction date:** 2026-07-07

**关键发现 / Planner 须注意：**
1. **types 应放 `src/types/network.ts`**（不是 `index.ts`），与 `DevicePortStatus` 同处。
2. **monitor/logs/index.tsx 当前不读 URL query**，D-10 须顺带补 module query 预填逻辑（参考 `reconciliation/exceptions/index.tsx:145-172`）。
3. **路由路径 `/monitor/logs`**（不是 `/monitor/operlog`）；API 路径才是 `/monitor/oper-logs/list`。
4. **权限源 `useMenuStore`**（不是 ROADMAP 写的 `useAuthStore`）。
5. **HTTP 200 + `status:"failed"`** 路径：BulkWriteDrawer 不能靠 catch Promise reject 分类失败，必须读 `result.failed/succeeded/skipped` 数组。
6. **post() 拦截器已统一弹 Toast**，wrapper / withErrorHandling 不要重复弹（双重 Toast 坑）。
7. **batchInProgress 须同时禁用"刷新"+"采集所有设备"**（D-07 同类竞态，不只是刷新）。
