---
phase: 260712-vpj-mac-mac
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - xingran-react-frontend/src/lib/api/networkApi.ts
  - xingran-react-frontend/src/pages/network/ports/index.tsx
autonomous: true
requirements: []
user_setup: []
must_haves:
  truths:
    - "用户在端口管理页面点击行展开后,展开内容中可看到该端口的当前 MAC 列表"
    - "若端口 adminStatus='down' 或当前 MAC 列表为空,展开内容显示该端口最近的历史 MAC(含时间/事件类型)"
    - "若端口有多个当前 MAC(端口安全开启时),所有 MAC 全部以 Tag 形式展示,数量 > 1 时显示计数提示"
    - "展开内容通过按需懒加载(onExpand 触发),不在列表加载阶段产生 N+1 请求"
    - "复用现有后端端点 POST /network/history/port,后端无任何改动"
  artifacts:
    - path: "xingran-react-frontend/src/lib/api/networkApi.ts"
      provides: "新 wrapper getPortMACBundle(deviceId, interfaceName) 并行拉当前 MAC + 历史 MAC"
      exports: ["getPortMACBundle", "PortMACBundle"]
    - path: "xingran-react-frontend/src/pages/network/ports/index.tsx"
      provides: "Table.expandable.expandedRowRender 渲染 PortMACPanel;按需懒加载;"
      contains: "expandable"
  key_links:
    - from: "xingran-react-frontend/src/pages/network/ports/index.tsx"
      to: "xingran-react-frontend/src/lib/api/networkApi.ts"
      via: "import { getPortMACBundle }"
      pattern: "getPortMACBundle"
    - from: "xingran-react-frontend/src/lib/api/networkApi.ts"
      to: "POST /network/history/port"
      via: "post() 调用"
      pattern: "history/port"
    - from: "xingran-react-frontend/src/lib/api/networkApi.ts"
      to: "POST /network/mac/list"
      via: "post() 调用"
      pattern: "network/mac/list"
---

<objective>
在端口管理页面(`src/pages/network/ports/index.tsx`)的行展开(expandable row)内容中,展示该端口的当前 MAC 地址列表;当端口 down 或当前 MAC 为空时,fallback 展示该端口最近的历史 MAC 地址(带事件时间/类型)。若一个端口有多个 MAC,全部以 Tag 形式展示。
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/STATE.md
@.planning/quick/260712-vpj-mac-mac/260712-vpj-PLAN.md

# 必须阅读的源文件(executor 第一步必读,确认数据契约)
@xingran-react-frontend/src/pages/network/ports/index.tsx
@xingran-react-frontend/src/lib/api/networkApi.ts

# 关键后端契约
@internal/services/mac_history_query_service.go (lines 22-76 PortHistoryQuery / MACHistoryRecord 定义;lines 287-409 QueryPortHistory 实现)
@internal/api/v1/network/mac_history_handler.go (lines 26-50 QueryPortHistory handler + POST /network/history/port 路由)
@internal/api/v1/network/mac_history_router.go (lines 11-32 SetupMACHistoryRouter 路由注册)
@internal/api/v1/network/mac_handler.go (lines 16-72 MACAddressListRequest + List endpoint)
@internal/services/portcollection/utils.go (lines 60-145 CurrentMACCount 字段语义:端口安全的当前学习 MAC 数,非 MAC 列表本身)

# 归一化(关键约束,executor 必须理解)
@pkg/normalize/iface.go (lines 70-130 InterfaceName 对称化目标=大写短名 GE0/0/1)
@internal/models/device_port_status.go (lines 69-77 BeforeCreate 强制归一化 InterfaceName)
@internal/models/device_mac_address.go (lines 32-47 BeforeCreate 强制归一化 InterfaceName)
@internal/models/device_mac_history.go (lines 36-53 BeforeCreate 强制归一化 InterfaceName)

# 现有复用模式
@xingran-react-frontend/src/components/network/MACEventsTimeline.tsx (事件渲染风格参考)
@xingran-react-frontend/src/components/network/macEventMeta.ts (EVENT_LABEL/COLOR 单一事实源,可直接复用)
</context>

## 关键决策(quick 模式无 --discuss,以下为默认,executor 必须遵守)

### D-01 后端端点选择:零后端改动
**复用**两个现有端点,无需新建:
1. `POST /network/mac/list` — 当前 MAC(`sys_device_mac_address` 表),按 `(deviceId, interfaceName)` 精确过滤。`/network/mac/list` 已存在(D-09 锁定)。
2. `POST /network/history/port` — 历史 MAC(`sys_device_mac_history` 表),按 `(deviceId, interfaceName)` 精确过滤。已 Phase 15 PERF-03 加缓存(D-12 锁定)。

### D-02 接口名归一化:无需前端归一化
`DevicePortStatus.interfaceName` 字段经过 `BeforeCreate` 钩子(已确认)归一化为大写短名(如 `GE0/0/1`)。后端 `mac_history_query_service.go:328-330` 对 `interface_name = ?` 走精确匹配。两边格式一致,**前端直接传 `record.interfaceName` 即可**,无需再调 `normalize.InterfaceName`。

### D-03 "最近历史 MAC" 判定口径
按 `interface_name` 过滤后,按 `first_seen DESC` 取**最新一条**作为 fallback 展示(label = "最近一次出现过的 MAC")。展开内容历史区段显示该单条 + 时间/事件类型 + VLAN;不展开时间线(避免与 MACEventsTimeline 重复)。

### D-04 数据获取策略:并行 Promise.all
新 wrapper `getPortMACBundle(deviceId, interfaceName)`:
- `Promise.all([currentMAC, recentHistory])` 并行两个端点
- `currentMAC`: `post("/network/mac/list", { current: 1, pageSize: 50, deviceId, interfaceName })` — 当前端口一个端口最多 50 条 MAC(端口安全 MaxMAC 限制 4096 上限,留余量)
- `recentHistory`: `post("/network/history/port", { deviceId, interfaceName, current: 1, pageSize: 1 })` — 严格只取 1 条作为 fallback

### D-05 懒加载:onExpand 触发,缓存到 Map<portId, bundle>
避免 N+1: 用 `useState<Record<string, PortMACBundle>>` 缓存已加载的展开数据,`onExpand(expanded, record)` 回调中按需 fetch,首次展开后才发起请求。同一行折叠再展开命中缓存不重发。

### D-06 展开内容展示形态
按 `D-01/D-03` 优先级展示:
1. **当前 MAC 列表区**:从 `sys_device_mac_address` 拉,无端口安全时也可能为空(取决于采集覆盖)。每条用 Tag 显示 MAC,颜色=标准蓝色;附带 VLAN Tag(若非空);右侧小字显示 `deviceNameSnapshot` 或设备名。
2. **历史 MAC fallback 区**(条件渲染):**仅当**当前 MAC 列表为空 AND (`adminStatus==='down'` OR `operStatus==='down'`) 时显示。label = "最近一次出现过的 MAC",展示单条历史 MAC + 时间 + 事件类型 Tag。
3. **当前 MAC > 1** 时在区段标题加计数提示。

### D-07 operlog / 权限 / 路由注册:无
纯只读展示,无需 operlog;`/network/mac/list` 和 `/network/history/port` 已在 `network` group 的 JWT auth 下,无需新增权限中间件。

### D-08 风格一致性
- 当前 MAC Tag 风格:与列表页 `mac/index.tsx` 现有 MAC Tag 一致(蓝色)
- 历史 MAC 事件 Tag:复用 `EVENT_LABEL/EVENT_TAG_COLOR` from `@/components/network/macEventMeta`(D-10 锁定单一事实源)
- 时间格式化:复用 `@/utils/datetime.formatDateTime`(同列表页)
- Loading:Spin + Skeleton;Error:ErrorAlertWithRetry(已存在于 `@/components/shared`,见 MACEventsTimeline:127)

## Source Coverage Audit

| Source | Item | Plan Coverage |
|--------|------|---------------|
| **GOAL** (quick 描述) | "端口管理页面添加 MAC 地址和历史 MAC 地址的显示" | 任务 1+2 |
| **GOAL** | "展示在展开的内容中" | 任务 2 (替换 expandedRowRender) |
| **GOAL** | "优先展示当前 MAC 地址" | D-06 当前 MAC 区优先 |
| **GOAL** | "若端口 down、没有当前 MAC 地址,则展示最近的历史 MAC" | D-03 + D-06 fallback 条件 |
| **GOAL** | "若一个端口有多个 MAC 地址,则全部展示" | D-04 pageSize=50 + D-06 多 MAC Tag 列表 |
| **REQ** | 无 ROADMAP phase_req(quick 模式) | N/A |
| **RESEARCH** | 无 RESEARCH.md(quick 模式,无研究阶段) | N/A |
| **CONTEXT** | 无 CONTEXT.md(quick 模式,无 discuss) | D-01~D-08 决策即默认 |
| **CONSTRAINTS** | "复用真实文件" | executor Step 1 强制 Read 三个契约文件 |
| **CONSTRAINTS** | "原子、自包含" | 2 个任务,聚焦前端单文件 + 1 个 API wrapper |
| **CONSTRAINTS** | "~30% 上下文占用" | 简单纯前端改动,符合目标 |
| **DOMAIN MEMORY** | interface_name 必须归一化 | D-02 已对齐(后端 BeforeCreate 兜底) |
| **DOMAIN MEMORY** | MAC 格式带冒号 | D-08 与现有展示一致(后端 `NormalizeMACAddress` 已统一) |
| **DOMAIN MEMORY** | 华为 PHY 语义(已解析 adminStatus) | D-06 直接用 `record.adminStatus` 字段,不重解析 |
| **DOMAIN MEMORY** | ops_info_points.device_id 84% 漂移 | N/A(本次不走 info_point JOIN,直接用 port 自己的 device_id) |
| **DOMAIN MEMORY** | 前端 API 一律用封装 | D-07 全部走 `post()` from `@/lib/api` |

No unplanned items. No deferred items violated. No scope reduction.

<tasks>

<task type="auto">
  <name>Task 1: 新增 getPortMACBundle wrapper + PortMACBundle 类型</name>
  <files>xingran-react-frontend/src/lib/api/networkApi.ts</files>
  <action>
    在 `networkApi.ts` 末尾(在 `default` export 之前)新增:
    1. 类型 `PortMACBundle`:
       - `current: Array<{ macAddress: string; vlanId?: number | null; interfaceName: string }>`
       - `recentHistory: MACHistoryRecord | null` (取 pageSize=1 第一条)
       - `loading: boolean`、`error: Error | null`(给调用方状态管理)
    2. wrapper `getPortMACBundle(deviceId: string, interfaceName: string): Promise<PortMACBundle>`:
       - 内部 `Promise.all([fetchCurrent(), fetchRecent()])`,任一 reject 不互相阻塞——各自 try/catch 收集 error
       - `fetchCurrent`: `await post("/network/mac/list", { current: 1, pageSize: 50, deviceId, interfaceName })` — 取出 `result.data?.list`(对应 `MACAddressResponse[]`,字段名需先 Read 一次 `internal/services/mac_collection_service.go` 的 `MACAddressResponse` 定义,见 executor 必读清单)
       - `fetchRecent`: `await post("/network/history/port", { deviceId, interfaceName, current: 1, pageSize: 1 })`,取 `result.data?.list?.[0] || null`
       - **不**做 try/catch 上抛 message.error(`@/lib/api` 的 post 拦截器已统一处理,见 networkApi.ts:260 注释 LANDMINE #5);只把 try/catch 内的 error 收集到 `bundle.error`,方便组件按字段粒度区分展示
    3. 在文件末尾 `default` export 加上 `getPortMACBundle`(顶层 `export const` 已导出,default 仅汇总)

    **关键约束(避免踩坑)**:
    - 与 `queryMACHistory` 同风格:`post<T>(url, params) → result.data!`(勿解 envelope 二层)
    - 不引新依赖;不调 message.error
    - 命名与 `queryMACHistory` 对齐(`query` vs `get` 前缀已在文件内并存,本场景用 `get` 更准确表示"拉单端口 bundle")
  </action>
  <verify>
    <automated>cd D:/CODE/ClaudeCode/xingran-go-backend/xingran-react-frontend && npx tsc --noEmit -p . 2>&1 | head -20</automated>
  </verify>
  <done>
    - `networkApi.ts` 新增 `PortMACBundle` interface + `getPortMACBundle` 函数
    - 函数签名为 `(deviceId: string, interfaceName: string) => Promise<PortMACBundle>`
    - 默认 export 包含 `getPortMACBundle`
    - `npx tsc --noEmit` 无新增错误(允许 pre-existing 错误,关注与本任务相关的 import/类型)
  </done>
</task>

<task type="auto">
  <name>Task 2: 在 ports/index.tsx 展开行中渲染 PortMACPanel(懒加载)</name>
  <files>xingran-react-frontend/src/pages/network/ports/index.tsx</files>
  <action>
    修改 `src/pages/network/ports/index.tsx`,在 Table 的 `expandable.expandedRowRender` 内集成新的 MAC 展示组件:

    **Step A: 新增内联子组件 `PortMACPanel`**(放在文件顶部 `PortStatusPage` 组件**外**或文件末尾,作为函数组件;保持单文件改动):
    - Props: `{ portId: string; deviceId: string; interfaceName: string; adminStatus: string; operStatus: string; bundle?: PortMACBundle; load: () => void; }`
    - 状态管理:由父组件维护 `Map<portId, PortMACBundle>` 缓存,本组件纯展示
    - 渲染分支:
      1. `bundle === undefined`(未展开过) → 显示 "加载中..." Skeleton(active paragraph rows 2)
      2. `bundle.error && (!bundle.current.length && !bundle.recentHistory)` → 显示 `<ErrorAlertWithRetry error={bundle.error} onRetry={load} />`(import from `@/components/shared`,同 MACEventsTimeline)
      3. `bundle.current.length > 0` → **当前 MAC 区**:
         - 标题 `当前 MAC 地址(${bundle.current.length})`
         - 每条 MAC 用 `<Tag color="blue">{mac.macAddress}</Tag>`,若 `vlanId != null` 旁加 `<Tag>VLAN {vlanId}</Tag>`
         - Wrap 在 Space 中,允许换行
      4. `bundle.current.length === 0 && (adminStatus === 'down' || operStatus === 'down') && bundle.recentHistory` → **最近历史 MAC 区**:
         - 标题 `最近一次出现过的 MAC`
         - 复用 `EVENT_LABEL/EVENT_TAG_COLOR` from `@/components/network/macEventMeta`
         - 一行展示: MAC Tag + 事件类型 Tag + 时间(formatDateTime)+ VLAN(若有)
      5. `bundle.current.length === 0 && !down` → 显示 `<Empty description="该端口暂无 MAC 数据" />`(用 antd Empty)

    **Step B: 修改主组件状态**:
    - 引入 `useState<Record<string, PortMACBundle>>` 名为 `macBundleCache`
    - 引入 `loadPortMACBundle` 函数: 命中 cache 直接返回;否则 fetch 并 setState。组件卸载/切换行无需 abort(短请求可接受)

    **Step C: 修改 Table expandable**:
    - 增加 `expandedRowKeys`(受控)和 `onExpand(expanded, record)` 回调
    - `expandedRowKeys` 初始为空,由用户点击控制;onExpand 中若 expanded=true 且 cache 未命中,触发 load
    - `expandedRowRender: (record) => <PortMACPanel {...} />`,把 record.id/deviceId/interfaceName/adminStatus/operStatus 传下去

    **Step D: 新增 imports**:
    - `import { getPortMACBundle, type PortMACBundle } from "@/lib/api/networkApi";`
    - `import { Empty, Skeleton, Tag, Space, Typography } from "antd";`(按需加入,Tag/Space 可能已存在)
    - `import { ErrorAlertWithRetry } from "@/components/shared";`(可能已存在,确认)
    - `import { formatDateTime } from "@/utils/datetime";`(已存在,复用)
    - `import { EVENT_LABEL, EVENT_TAG_COLOR, type MACEventType } from "@/components/network/macEventMeta";`(复用现有元数据)

    **关键约束**:
    - **不**重构现有 `expandedRowRender` 的其他展示内容(dot1x/portSecurity 块);新组件独立 render,与现有 `<p>` 块平行(用 Fragment 或额外 div 包裹)
    - **不**改 columns、统计数据、searchForm、loadPortStatus、handleCollectAll、handleBatchDelete、ActionButtons、PortWriteModal、SetAccessVlanModal、PortBindingModal、BulkWriteDrawer 等任何已有逻辑
    - **不**新增 operlog 调用(纯只读)
    - **不**修改 `useEffect`/`useTableManager`/`usePagination`/`useMenuStore` 等 hook
    - 懒加载的 cache 用普通 Record 即可(端口列表分页最多 10-100 条,内存可接受;无需 LRU)

    **Step E(防退化)**: `useEffect` 现有 deps 是空数组,展开逻辑不影响 mount 流程;无需新增 useEffect。
  </action>
  <verify>
    <automated>cd D:/CODE/ClaudeCode/xingran-go-backend/xingran-react-frontend && npx tsc --noEmit -p . 2>&1 | head -30</automated>
  </verify>
  <done>
    - ports/index.tsx 在 Table expandable 中展开行新增 MAC 展示
    - 展开时按需 lazy load(首次展开才发起 fetch)
    - 当前 MAC 列表为空且端口 down 时显示历史 fallback
    - 当前 MAC > 1 时全部 Tag 展示
    - `npx tsc --noEmit` 无新增 TS 错误
    - `go build ./...` 不受影响(本次未改后端 Go 代码)
    - 已有 expandedRowRender 中的 dot1x/portSecurity 块保留未动
  </done>
</task>

</tasks>

<verification>
- [ ] `npx tsc --noEmit -p .` 无与本任务相关的错误
- [ ] 人工浏览器验证(可选,executor 自决):访问 http://localhost:4000/network/ports,点击任意一行展开,展开区域包含"当前 MAC 地址"或"最近一次出现过的 MAC"段;若该端口 adminStatus=down 且当前 MAC 为空,应展示历史 MAC fallback
- [ ] DevTools Network 面板:点击展开行后才看到 `/network/mac/list` 与 `/network/history/port` 请求;折叠再展开同一行不再发请求
- [ ] 多 MAC 端口(portSecurityEnabled 且 currentMACCount > 1)展开后能看到多个 Tag
</verification>

<success_criteria>
1. 端口管理页面行展开后,展开内容新增 MAC 展示段,符合需求优先级:当前 MAC > 历史 MAC fallback
2. 后端零改动,完全复用 `/network/mac/list` 与 `/network/history/port` 现有端点
3. 展开按需加载,无 N+1,折叠再展开走 cache
4. 保持现有 dot1x/portSecurity 块展示不变
5. TypeScript 通过编译
</success_criteria>

<output>
完成后创建 `.planning/quick/260712-vpj-mac-mac/260712-vpj-SUMMARY.md`(按 quick 模板简化版,记录:改动文件清单 + 关键决策 D-01~D-08 + build 验证结果)
</output>