# Phase 70 — 存量 TextArea 四象限审计 + 契约默认值定稿（T1）

> 审计口径：grep `<(Input.TextArea|TextArea)` JSX 开标签，范围 `xingran-react-frontend/src`，排除 `__tests__`/`*.test.*`/`*.spec.*`、`src/components/ui/**`、`src/components/shared/**`。每处取开标签起 5 行上下文判属性；`rows` 检测用 `(?<![a-zA-Z])rows=`（排除 maxRows/minRows 误匹配）。
>
> **实测总数：51 处**（成员表达式 `<Input.TextArea` 14 处 + 解构/具名 `<TextArea` 37 处），涉及 44 个文件。计划期粗估「约 74」（STACK 研究期 18+43=61 开标签口径 / grep 行数 26+48=74 行口径）与本次实测的偏差来自统计口径（行数 vs 开标签数、上下文窗口宽窄），**权威数以 T5 eslint selector 实跑为准**（PLAN 事实核查注记 7 已预告此差异）。

## 一、四象限逐处清单

四象限：① showCount ② maxLength ③ 受控（value= 显式受控）④ onChange 显式传递。附加列：rows / autoSize / Form.Item 包裹。

| # | 文件 | 行 | 形态 | showCount | maxLength | value= | onChange | rows= | autoSize |
|---|---|---|---|---|---|---|---|---|---|
| 1 | components/dashboard/settings/DashboardSettings.tsx | 147 | TextArea | 否 | 否 | 否 | 否 | 是 | 否 |
| 2 | components/dashboard/settings/WidgetSelector.tsx | 271 | TextArea | 否 | 否 | 否 | 否 | 是 | 否 |
| 3 | components/dashboard/widgets/WidgetEditor.tsx | 156 | TextArea | 否 | 否 | 否 | 否 | 是 | 否 |
| 4 | components/operations/WorkstationDeviceTable/index.tsx | 643 | Input.TextArea | 否 | 否 | 否 | 否 | 是 | 否 |
| 5 | pages/ad-domain/configs/AccountPoolTab.tsx | 445 | Input.TextArea | **是** | **是** | 否 | 否 | 是 | 否 |
| 6 | pages/ad-domain/groups/index.tsx | 579 | Input.TextArea | 否 | 否 | 否 | 否 | 是 | 否 |
| 7 | pages/ad-domain/users/index.tsx | 688 | Input.TextArea | 否 | 否 | 否 | 否 | 是 | 否 |
| 8 | pages/asset/reconciliation/fix-suggestion/index.tsx | 564 | Input.TextArea | 否 | 否 | 否 | 否 | 是 | 否 |
| 9 | pages/duty/holidays/modals/EditModal.tsx | 129 | TextArea | 否 | 否 | 否 | 否 | 是 | 否 |
| 10 | pages/duty/pools/index.tsx | 536 | Input.TextArea | 否 | 否 | 否 | 否 | 是 | 否 |
| 11 | pages/duty/schedules/index.tsx | 701 | Input.TextArea | 否 | 否 | 否 | 否 | 是 | 否 |
| 12 | pages/duty/schedules/index.tsx | 779 | Input.TextArea | 否 | 否 | 否 | 否 | 是 | 否 |
| 13 | pages/knowledge/articles/modals/index.tsx | 112 | TextArea | 否 | 否 | 否 | 否 | 是 | 否 |
| 14 | pages/knowledge/articles/modals/index.tsx | 120 | TextArea | 否 | 否 | 否 | 否 | 是 | 否 |
| 15 | pages/monitor/cache/index.tsx | 438 | TextArea | 否 | 否 | 否 | 否 | 是 | 否 |
| 16 | pages/monitor/job/index.tsx | 309 | TextArea | 否 | 否 | 否 | 否 | 是 | 否 |
| 17 | pages/monitor/job/index.tsx | 338 | TextArea | 否 | 否 | 否 | 否 | 是 | 否 |
| 18 | pages/network/backups/index.tsx | 607 | TextArea | 否 | 否 | 否 | 否 | 是 | 否 |
| 19 | pages/network/credentials/index.tsx | 597 | TextArea | 否 | 否 | 否 | 否 | 是 | 否 |
| 20 | pages/network/devices/components/DeviceModals.tsx | 193 | TextArea | 否 | 否 | 否 | 否 | 是 | 否 |
| 21 | pages/network/devices/components/DeviceModals.tsx | 296 | TextArea | 否 | 否 | 否 | 否 | 是 | 否 |
| 22 | pages/network/templates/modals/EditModal.tsx | 119 | TextArea | 否 | 否 | 否 | 否 | 是 | 否 |
| 23 | pages/operations/buildings/index.tsx | 653 | TextArea | 否 | 否 | 否 | 否 | 是 | 否 |
| 24 | pages/operations/dedicated-lines/index.tsx | 828 | TextArea | 否 | 否 | 否 | 否 | 是 | 否 |
| 25 | pages/operations/floors/components/FloorModal.tsx | 159 | TextArea | 否 | 否 | 否 | 否 | 是 | 否 |
| 26 | pages/operations/info-points/index.tsx | 1138 | TextArea | 否 | 否 | 否 | 否 | 是 | 否 |
| 27 | pages/operations/room-devices/index.tsx | 757 | TextArea | 否 | 否 | 否 | 否 | 是 | 否 |
| 28 | pages/operations/rpa/tasks/modals/EditModal.tsx | 293 | TextArea | 否 | 否 | 否 | 否 | 是 | 否 |
| 29 | pages/operations/server-rooms/index.tsx | 657 | TextArea | 否 | 否 | 否 | 否 | 是 | 否 |
| 30 | pages/operations/workstations/modals/EditModal.tsx | 298 | TextArea | 否 | 否 | 否 | 否 | 是 | 否 |
| 31 | pages/operations/workstations/LocationAliasDrawer.tsx | 269 | Input.TextArea | 否 | 否 | 否 | 否 | 是 | 否 |
| 32 | pages/profile/index.tsx | 364 | Input.TextArea | 否 | 否 | 否 | 否 | 是 | 否 |
| 33 | pages/system/apikeys/index.tsx | 755 | TextArea | 否 | 否 | 否 | 否 | 是 | 否 |
| 34 | pages/system/captcha-background/modals/EditModal.tsx | 86 | TextArea | 否 | 否 | 否 | 否 | 是 | 否 |
| 35 | pages/system/captcha-background/modals/UploadModal.tsx | 93 | TextArea | 否 | 否 | 否 | 否 | 是 | 否 |
| 36 | pages/system/config/index.tsx | 537 | TextArea | 否 | 否 | 否 | 否 | 是 | 否 |
| 37 | pages/system/dept/modals/EditModal.tsx | 162 | Input.TextArea | 否 | 否 | 否 | 否 | 是 | 否 |
| 38 | pages/system/dict/index.tsx | 891 | TextArea | 否 | 否 | 否 | 否 | 是 | 否 |
| 39 | pages/system/dict/index.tsx | 957 | TextArea | 否 | 否 | 否 | 否 | 是 | 否 |
| 40 | pages/system/menu/index.tsx | 478 | Input.TextArea | 否 | 否 | 否 | 否 | 是 | 否 |
| 41 | pages/system/post/index.tsx | 256 | TextArea | 否 | 否 | 否 | 否 | 是 | 否 |
| 42 | pages/system/role/index.tsx | 402 | Input.TextArea | 否 | 否 | 否 | 否 | 是 | 否 |
| 43 | pages/system/settings/api-config.tsx | 550 | TextArea | 否 | 否 | 否 | 否 | 是 | 否 |
| 44 | pages/system/settings/captcha-background.tsx | 522 | TextArea | 否 | 否 | 否 | 否 | 是 | 否 |
| 45 | pages/system/settings/captcha-background.tsx | 583 | TextArea | 否 | 否 | 否 | 否 | 是 | 否 |
| 46 | pages/system/settings/email-config.tsx | 492 | TextArea | 否 | 否 | 否 | 否 | 是 | 否 |
| 47 | pages/workorder/categories/index.tsx | 305 | Input.TextArea | 否 | 否 | 否 | 否 | 是 | 否 |
| 48 | pages/workorder/orders/index.tsx | 834 | TextArea | 否 | 否 | 否 | 否 | 是 | 否 |
| 49 | pages/workorder/orders/index.tsx | 838 | TextArea | 否 | 否 | 否 | 否 | 是 | 否 |
| 50 | pages/workorder/orders/index.tsx | 937 | TextArea | 否 | 否 | 否 | 否 | 是 | 否 |
| 51 | pages/workorder/periodic/templates/index.tsx | 603 | TextArea | 否 | 否 | 否 | 否 | 是 | 否 |

**受控形态补充**：上表 ③④ 两列统计的是「组件自身 props 上的显式 `value=` / `onChange`」；实测 51 处中有 **50 处包裹在 `<Form.Item name=…>` 内**，由 antd Form 接管 value/onChange（Form 受控，PITFALLS 1.2 的「受控耦合」实际由 Form 层承担）。仅 `#3 WidgetEditor.tsx:156` 一处不在 Form.Item 直接包裹内（独立数据流）。全量 grep **`allowClear={false}` 零命中**——存量无显式关闭清空能力者。

## 二、定稿结论（统计 / 结论 / 理由）

### 1. `allowClear = true` — **下沉为契约默认**（locked 预设，确认成立）

- **统计**：51 处存量无一显式传 `allowClear`（`allowClear={false}` 零命中）；antd TextArea 6.x 原生支持 `allowClear`。
- **结论**：`allowClear = true` 作为唯一预设默认。
- **理由**：TEXT-01 需求本体；存量无反例，无「显式关闭」耦合例外。行为外溢仅发生在**迁移时**（Phase 71），本 phase 组件新增不影响存量（存量不迁移）。

### 2. `showCount` — **不下沉默认**

- **统计**：仅 1/51（`#5 AccountPoolTab.tsx`）显式传 `showCount`。
- **结论**：不写默认，留在 rest 透传，需要处显式传。
- **理由**：下沉 `showCount=true` 会让其余 50 处迁移后突然出现字数计数 UI，属视觉外溢；与 AppInput 先例（仅 allowClear 单默认）一致。

### 3. `maxLength` — **不下沉默认**

- **统计**：仅 1/51（同 `#5 AccountPoolTab.tsx`）显式传 `maxLength`。
- **结论**：不写默认。
- **理由**：PITFALLS 1.2——maxLength 是截断语义，静默截断存量长文本属行为破坏；计数与截断语义应成对显式声明。

### 4. `autoSize` — **不下沉默认，由调用方显式传**（Open Question 1 已按默认裁定）

- **统计**：51 处存量 **0 处**传 `autoSize`，51/51 显式传 `rows`（1–6 行不等）。PLAN 预判「若 ≥70% 显式 autoSize 也不改结论」——实测 0%，结论更稳。
- **结论**：`AppTextArea` 不预设 autoSize；显式传入生效（rest 透传）。
- **理由**：下沉默认会把存量显式 `rows=N` 与 autoSize 混合产生未定义高度行为（antd 同传时 autoSize 优先、rows 失效），存量高度行为零外溢是硬边界。

### 5. 受控 + onChange 清除路径 — **Form 层受控，`onChange('')` 直通**

- **统计**：50/51 由 `Form.Item name` 接管；1 处独立受控。
- **结论**：契约组件不做值归一化；清空即 `onChange('')` 事件直通。SearchForm 层（T3）：textarea 字段空串直通 filters，`useTableManager.handleSearch` 的 filterEmpty 已过滤空串，与 input 分支现状一致（Open Question 3 维持现状裁定）。
- **理由**：归一化上收到封装层会改变 50 处 Form 受控页面的数据流预期；PITFALLS 1.2 的四重耦合以「零默认 + 全透传」最小面化解。

## 三、SearchForm textarea 字段（T3）SearchField 可选 props 定稿

按上述结论：

```ts
{ type: "textarea"; name: string; label: string; placeholder?: string;
  rows?: number;                                    // 显式行高（存量惯例 51/51）
  autoSize?: boolean | { minRows: number; maxRows: number }  // 显式自适应（透传）
}
```

渲染分发：`<AppTextArea placeholder rows autoSize />`，**不传 `onPressEnter`**（多行回车必须换行）。

## 四、存量显式 `allowClear={false}` 例外清单

**无**（全量 grep 零命中）。Phase 71 迁移无「清空能力关闭」例外需保护；若迁移中发现语义上不应可清空的字段（如只读展示），由调用方显式传 `allowClear={false}`。
