# 260712-vpj — 端口管理页面 MAC/历史 MAC 展示(quick)

## 目标
在端口管理页面(`src/pages/network/ports/index.tsx`)的行展开内容中,展示该端口的当前 MAC 地址列表;当端口 down 或当前 MAC 为空时,fallback 展示该端口最近的历史 MAC 地址(带事件时间/类型)。多 MAC 全部以 Tag 形式展示。

## 改动文件清单(2 个)

| 文件 | 变更 |
|------|------|
| `xingran-react-frontend/src/lib/api/networkApi.ts` | +93 行 — 新增 `PortCurrentMAC` / `PortMACBundle` 类型与 `getPortMACBundle(deviceId, interfaceName)` wrapper |
| `xingran-react-frontend/src/pages/network/ports/index.tsx` | +142 / -2 — 新增内联 `PortMACPanel` 子组件、`macBundleCache` state、`loadPortMACBundle` 函数;Table.expandable 改受控,新增 `expandedRowKeys` + `onExpand` 懒加载触发 |

## 关键决策(执行版)

| # | 决策 | 落地 |
|---|------|------|
| D-01 | 零后端改动,复用 `/network/mac/list` + `/network/history/port` | wrapper 仅 fetch + 不写后端 |
| D-02 | 接口名归一化由后端 BeforeCreate 钩子保证,前端透传 `record.interfaceName` | 不调 `normalize.InterfaceName` |
| D-03 | fallback 取 `firstSeen DESC` 第 1 条历史 MAC | `pageSize=1` 严格只取 1 条 |
| D-04 | `Promise.allSettled` 并行两个端点 | 任一 reject 不互阻塞,error 收集到 `bundle.error` |
| D-05 | 懒加载 + `Record<portId, PortMACBundle>` 缓存 | 首次展开才 fetch,折叠再展开命中 cache |
| D-06 | 当前 MAC > 0 → 当前 MAC 区(>1 显示计数);空 + down → 历史 fallback;空 + up → Empty | 三分支条件渲染 |
| D-07 | 无 operlog / 无权限中间件 / 无路由注册 | 纯只读展示 |
| D-08 | 复用 `EVENT_LABEL/EVENT_TAG_COLOR` from `@/components/network/macEventMeta`;`formatDateTime` from `@/utils/datetime`;`ErrorAlertWithRetry` from `@/components/shared` | 不新建元数据 |

## build 验证

| 命令 | 结果 |
|------|------|
| `tsc --noEmit -p xingran-react-frontend/tsconfig.json` (Task 1) | 0 errors |
| `tsc --noEmit -p xingran-react-frontend/tsconfig.json` (Task 2) | 0 errors |

(使用主仓库 `node_modules/.bin/tsc` 二进制,工作树无 `node_modules`)

## 提交

| Task | Commit | Message |
|------|--------|---------|
| Task 1 | `be99597b` | feat(260712-vpj): ports 展开行 MAC bundle wrapper — Task 1 |
| Task 2 | `bc26f5c4` | feat(260712-vpj): ports 展开行 MAC 展示(懒加载) — Task 2 |

## 未改后端 Go 代码(per D-01)

无 `internal/` 改动,所有数据来自现有 `/network/mac/list` 与 `/network/history/port` 端点。

## 保留行为(per 约束)

- 现有 expandedRowRender 中 dot1x/portSecurity 块未动,PortMACPanel 平行 render 于其下方 12px 间距
- columns、loadPortStatus、handleCollectAll、handleBatchDelete、ActionButtons、PortWriteModal、SetAccessVlanModal、PortBindingModal、BulkWriteDrawer 全部未触
- useEffect/useTableManager/usePagination/useMenuStore 等 hook 未新增/修改
- 无 operlog 调用(纯只读)

## 人工浏览器验证(可选)

- 访问 http://localhost:4000/network/ports
- 点击任意行展开 → 展开区看到"当前 MAC 地址(N)"或"最近一次出现过的 MAC"段
- DevTools Network:点击展开行后才看到 `/network/mac/list` + `/network/history/port` 请求;折叠再展开同一行不再发请求
- portSecurityEnabled + currentMACCount > 1 的端口,展开后能看到多个 MAC Tag