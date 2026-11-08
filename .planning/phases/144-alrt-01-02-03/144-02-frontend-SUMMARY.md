---
phase: 144
plan: 02
wave: 1
status: complete
commit: false   # Lead 统一集成提交；本 executor 未执行任何 git add/commit/stash/checkout
---

# Phase 144 · 同步漂移告警通道收口 — 前端（ALRT-03）SUMMARY

> 执行者：gsd-executor（exec-144-front）。后端由 PLAN-01 独立 executor 并行负责，两者文件面互斥；本 executor **未触碰任何 `*.go` 文件**。

## 一、改动清单（全部在独占文件面内，零新依赖）

### 新建
| 文件 | 作用 |
|---|---|
| `src/pages/system/ad-drift-alert/index.tsx` | 主页面：组合状态卡/日志表/配置表单 +「立即检测」+ `onADDriftAlert` 订阅（cleanup 退订）。由 `componentLoader.tsx` 的 `import.meta.glob` 自动发现，**未改 `App.tsx`** |
| `src/pages/system/ad-drift-alert/components/StatusCards.tsx` | 状态卡：累计告警次数 / 基线 / 容差·节流·上次告警 / 4 个通道开关（enabled·noticeEnabled·wsEnabled·webhookEnabled） |
| `src/pages/system/ad-drift-alert/components/TriggerLogTable.tsx` | 触发记录表：`PageTable`（内建三态）+ `EmptyStateWithAction` + `ErrorAlertWithRetry`；分页 current 从 1、按时间/状态筛选、dayjs 时间格式化 |
| `src/pages/system/ad-drift-alert/components/ConfigForm.tsx` | 9 旋钮表单（baseline 只读展示）+ webhook 下拉（`getAPINotificationConfigList({configType:'webhook'})`）+ `:edit` 三态门控 |
| `src/pages/system/ad-drift-alert/__tests__/index.render.test.tsx` | 页面编排测试（6 例）：渲染/API 编排/订阅与退订/WS 刷新/立即检测/权限门控 |
| `src/lib/adDriftAlertApi.ts` | 5 个 API：`getStatus`/`listTriggerLogs`/`getConfig`/`updateConfig`/`triggerOnce`，全部 `import { post } from "@/lib/api"` wrapped，直接返回 `result.data` |
| `src/lib/__tests__/adDriftAlertApi.test.ts` | 7 例：断言走 wrapped `post`、路径正确、无裸 axios、返回 `result.data`、current 从 1 |
| `src/hooks/__tests__/useADDriftAlertWebSocket.test.tsx` | 4 例：`onADDriftAlert` 订阅/退订、其它类型不误触发、静态断言「本 phase 前端文件无第二条 `new WebSocket`」 |

### 修改
| 文件 | 改动 |
|---|---|
| `src/types/notice.ts` | `WebSocketMessageType` 加 `"ad_sync_drift_alert"`；新增 `ADSyncDriftAlertEvent` 接口（memberCount/distinctGroups/baseline/deltas/noticeId/tolerance/throttled/timestamp/title） |
| `src/store/noticeStore.ts` | 模块级 `driftAlertListeners: Set` + 具名导出 `onADDriftAlert(listener)`（返回退订）+ store action 同名复用；`handleWsMessage` 加 `"ad_sync_drift_alert"` 分支：`JSON.parse(message.content)` 容错（失败仅 `console.warn`，不抛）、派发到 listeners（try/catch 隔离单个 listener 异常）；**既有 `new_notice`/`rpa_*`/`port_write_*` 分流保持不变** |
| `src/store/noticeStore.test.ts` | 扩 7 例 drift 分流测试（新类型派发/订阅退订/解析失败/缺 content/非对象/异常隔离/不误触发） |
| `src/constants/routes.ts` | 加 `SYSTEM_AD_DRIFT_ALERT = "/system/ad-drift-alert"` |
| `src/constants/pageTitles.ts` | 加 `PAGE_TITLES.AD_DRIFT_ALERT = "同步漂移告警"` + `SPECIAL_PATH_TITLES` 映射 |
| `src/constants/permissionCodes.ts` | 加 `PERMISSION_CODES.adDriftAlert` 组（4 个 F 型操作码）+ 独立具名常量 `SYSTEM_AD_DRIFT_ALERT_PERM`（C 型页面本体码） |

### 权限码 5 个逐字（三方一致）
`system:adDriftAlert` / `system:adDriftAlert:list` / `system:adDriftAlert:config` / `system:adDriftAlert:edit` / `system:adDriftAlert:trigger`

> **校验脚本机制说明**：C 型页面本体码按 `permissionCodes.ts` 文件头既有惯例**不进 `PERMISSION_CODES` 操作码表** —— `check-permission-codes.mjs` 用组内 `codes[0]` 的 `lastIndexOf(":")` 推导前缀并圈定校验 scope，若把 `system:adDriftAlert` 混进组里会把 scope 退化为 `system:` 并把所有 `system:*` 后端常量误报成 b 类漂移。故 4 个 F 型码进表、C 型码以独立具名常量导出并在注释逐字列出 5 码清单。脚本实测 scope 正确（`[adDriftAlert] 作用域 system:adDriftAlert:`）。

## 二、ALRT-03 满足情况

| 要求 | 落地 | 证据 |
|---|---|---|
| 前端「同步漂移告警」页面 | ✅ | `src/pages/system/ad-drift-alert/index.tsx`，菜单驱动 glob 自动发现，`App.tsx` 未改 |
| 告警列表 / 筛选 / 分页 | ✅ | `TriggerLogTable`：`PageTable` + current 从 1 + 时间/状态 `searchFields` + 三态组件 |
| WS 实时收到新告警 | ✅ | 复用**全局唯一** `/system/ws/notices` 连接（`NotificationBell.tsx` 既有），只扩 `noticeStore` 分流 + 类型 + listener；页面 `onADDriftAlert` 订阅（cleanup 退订），新告警 → `getAppMessage().warning` + 刷新状态卡 + 日志 `refreshToken +1`；**无第二条 `new WebSocket`** |
| API 全走 wrapped `post` | ✅ | 5 个 API 全部 `import { post } from "@/lib/api"`，直接 `result.data`；无裸 axios、无手动判 `code===0`（测试结构性断言） |
| store / 类型 | ✅ | `src/types/notice.ts` + `src/store/noticeStore.ts`（Zustand 既有 store，非新建 store） |
| 页面注册路由与菜单 | ✅ | 前端不改 `App.tsx`；`routes.ts`/`pageTitles.ts` 加常量；菜单由后端 migration 238 播种（已并行落地，component=`system/ad-drift-alert/index`） |
| 9 旋钮 + webhook 下拉 + 立即检测 | ✅ | `ConfigForm`（`:edit` 门控）/ 页面「立即检测」（`:trigger` 门控，`triggerOnce` 后刷新状态与日志） |
| 三态 / 唯一错误出口 / aria-label | ✅ | `PageTable`+`EmptyStateWithAction`+`ErrorAlertWithRetry`；`handleApiError` 唯一错误出口；icon-only 按钮带 `aria-label` |
| `useEffect` 依赖稳定 | ✅ | 依赖全为 primitive / `useCallback` / `useMemo` 稳定引用；`request`/`columns`/`searchFields`/`extraFilters`/`rowKey` 均 memo 化；禁止对象字面量进依赖 |
| webhook 脱敏 | ✅ | 下拉只取 `id`/`configName`，不读取/打印 `apiUrl`/`headers`/`authConfig`；不在前端日志打印 secret 或 URL |
| 零新依赖 | ✅ | 仅用既有 antd / dayjs / react / zustand / @testing-library |

## 三、验证结果

### 1. `npm run type-check` + 4 组 vitest（串联，`cd xingran-react-frontend`）
```
> tsc --noEmit -p tsconfig.app.json      → 通过（无错误）
npx vitest run src/store/noticeStore.test.ts \
  src/lib/__tests__/adDriftAlertApi.test.ts \
  src/pages/system/ad-drift-alert \
  src/hooks/__tests__/useADDriftAlertWebSocket.test.tsx
→ Test Files 4 passed (4) / Tests 37 passed (37) / FINAL_CHAIN_EXIT=0
```
分组：`noticeStore.test.ts` 20 例、`adDriftAlertApi.test.ts` 7 例、`index.render.test.tsx` 6 例、`useADDriftAlertWebSocket.test.tsx` 4 例。无 act() 警告（已收敛首屏异步 setState）。

### 2. `node scripts/check-permission-codes.mjs`（**脚本真实位置说明**）
- PLAN 字面写「仓库根执行 `node scripts/check-permission-codes.mjs`」→ 实测仓库根 `scripts/` 无此文件，`MODULE_NOT_FOUND`（exit 1）。脚本真实位置是 `xingran-react-frontend/scripts/check-permission-codes.mjs`（与 `package.json` 的 `lint:ci` 用法一致），脚本内部按 `scriptDir` 自解析 frontendRoot/repoRoot，故从前端 cwd 执行为正确方式。
- **首次执行（后端 PLAN-01 尚未落地时）**：`adDriftAlert` 组 8 项漂移（4 码 × a 类「后端 config.go 无常量」+ c 类「migrations sys_menu 未播种」），其余所有既有模块零漂移，exit 1。按 PLAN 要求**如实记录、未硬改任何 `.go` 文件**。
- **后端并行落地后重跑**：`pkg/permission/config.go` 已补 5 个常量、migration 238 已播种 C 型 `perms=system:adDriftAlert` + 4 个 F 型 seed → 
  ```
  [adDriftAlert] 作用域 system:adDriftAlert:
    TS 声明 4 个 / 后端常量 4 个 / 菜单 F 型 4 个
  ✓ 双向零漂移（TS ↔ 后端常量 ↔ sys_menu 菜单种子）   PERM_EXIT=0
  ```
- 三方逐字一致已只读核对：前端 `permissionCodes.ts`（4 F 型 + `SYSTEM_AD_DRIFT_ALERT_PERM` 具名常量 + 注释 5 码清单）／后端 `pkg/permission/config.go:363-367`（5 常量）／`migration_238_ad_drift_alert.go`（C 型 `perms` + 4 F 型 seed）。
- **给 Lead 的稳妥提示**：本 executor 验证时后端已落地且校验已转绿；若集成时后端权限文件面再有变动，建议在最终集成提交前**复跑一次** `cd xingran-react-frontend && node scripts/check-permission-codes.mjs` 兜底（非本 phase 未完成项）。

### 3. 未执行（按 PLAN 明确禁止）
- 未跑全量 `npm run test`、未跑任何 Go 命令（`go build` / `go test`）、未跑 `lint:ci` 全链 —— Lead 集成后统一跑。
- 未启动任何长驻进程。

## 四、Known Gaps（如实登记，非阻断）

1. **「标记已读」未实现**：ROADMAP Phase 144 成功标准 1 提到「标记已读、刷新后保留」，但 PLAN-01 已确认的 5 端点契约（`status/logs/config/config/update/trigger`）**无「标记已读」端点**，且 `logs` 数据来自 `sys_oper_log`（触发记录，无已读字段）。本 phase 严格按 PLAN-02 Task 3 明示的「分页/筛选/三态」交付；已读能力属后端契约缺口，登记为后续候选，不伪造闭环。
2. **响应字段名口径**：`ADDriftAlertStatus` / `ADDriftAlertConfig` / `ADDriftTriggerLog` 的 camelCase 字段名依 PLAN「已确认事实」+ 后端 service 的 `jsonResult` 口径推定（如 `throttleMinutes`、`noticeEnabled`、`webhookConfigId`）。若后端 handler 最终用 snake_case，只需改 `src/lib/adDriftAlertApi.ts` 一处类型/映射即可收口（API 层集中定义）。`baseline` 在 ConfigForm 只读展示、提交时原样带回，后端若视其为只读应忽略该字段。
3. **webhook 下拉降级**：`getAPINotificationConfigList` 失败时展示 `<Alert type="warning">` 并把下拉置空，用户可手动填配置 ID 或稍后重试；不展示明文 secret/URL（符合脱敏裁决）。
4. **WS 提示口径**：按 PLAN Task 1「选一种并写清」——**提示放在页面订阅回调里**，用 `@/utils/antdMessage` 的 `getAppMessage()`（context-aware），`noticeStore` 只做纯分流；未在 store 内使用静态 `message`（避免 antd 静态函数无法消费 ConfigProvider context 的告警）。
5. **测试隔离**：`index.render.test.tsx` 用轻量占位 mock 隔离 `PageTable`/`Form` 的 jsdom 复杂度，专注验证 `index.tsx` 编排；`TriggerLogTable` 的 `PageTable` 内建三态/分页由组件自身契约保证，未在本 phase 单独渲染测试（属既有契约组件，非本 phase 新增逻辑）。
6. **权限脚本路径**：PLAN 字面「仓库根执行」与脚本真实位置（`xingran-react-frontend/scripts/`）不一致，已在上方如实说明；不影响校验结论。

## 五、Git 状态

**未执行任何 `git add` / `git commit` / `git stash` / `git checkout`。** 4 个 executor 并行写同一工作区，所有改动仅落盘为工作区文件，由 **Lead 统一集成提交**。本 SUMMARY 亦不提交。

---
**结论**：`## PHASE 144 FRONTEND COMPLETE` —— ALRT-03 前端页面/类型/store/API/常量/测试全部交付，type-check 绿、37 例 vitest 全绿、权限码双向校验（后端落地后）零漂移；未触碰禁止文件面、零新依赖、未提交。
