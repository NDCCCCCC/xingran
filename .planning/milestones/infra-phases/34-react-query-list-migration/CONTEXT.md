---
phase: 34
subsystem: frontend-build
tags: [performance, react-query, list-pages, companion-pattern, cache, d-17, d-18, d-19, d-20, d-21, d-22]
requirements:
  - PERF-RQ-01
  - PERF-RQ-02
  - PERF-RQ-03
  - PERF-RQ-04
  - PERF-RQ-05
dependency_graph:
  requires:
    - 30-03 (queryKeys 工厂 / useTableQuery / useDict / useDeptTree / useRoleList / QueryClient defaults)
    - 30-04 (Widget memo / BaseEditModal memo)
    - 30-05 (companion 模式在 workstations 落地)
    - 33 (vercel-react-best-practices)
  provides:
    - network-devices-react-query-companion
    - vdi-vm-list-react-query-adoption
    - assets-react-query-adoption
    - cross-page-cache-reuse
  affects:
    - network-devices-page-data-flow
    - network-devices-statistics-cache
    - vdi-vm-list-page-data-flow
    - operations-assets-page-data-flow
tech_stack:
  added: []
  patterns:
    - useTableQuery companion to useTableManager (modal/form/selection 分家)
    - queryKeys.list.page/all 工厂 partial-match invalidate
    - useMemo + JSON.stringify 派生稳定 filters key
    - statistics 拆独立 queryKey（避免与 list 数据相互污染）
    - useTableManager 哑函数 + useEffect 同步 total 的回填模式
key_files:
  created: []
  modified: []
  planned:
    - xingran-react-frontend/src/pages/network/devices/index.tsx
    - xingran-react-frontend/src/pages/network/devices/hooks/useDeviceData.ts
    - xingran-react-frontend/src/hooks/useDeviceStatistics.ts (optional)
    - xingran-react-frontend/src/pages/vdi/VirtualMachineList/index.tsx (wave 2)
    - xingran-react-frontend/src/pages/operations/assets/index.tsx (wave 3)
decisions:
  - "D-17: companion 模式是 Phase 34 的统一范式(用户 2026-06-14 确认)"
  - "D-18: filters 序列化用 useMemo + JSON.stringify,严禁直接传 searchForm.getFieldsValue()"
  - "D-19: statistics 与 list 数据分两个 queryKey(不同 pageSize,不同业务意图)"
  - "D-20: useTableManager 哑函数策略,保持 hook 公共契约不变(8+ 现有 consumer)"
  - "D-21: invalidate 只在 mutation 成功路径调用,翻页/搜索/重置不调"
  - "D-22: total 同步用 useEffect 监听 queryResult.data.total 写回 externalPagination.setTotal"
metrics:
  duration: null
  completed: null
  status: planning
---

# Phase 34 Context: 列表页 React Query 迁移

## 1. 背景与目标

30-03 阶段在 XingRan-Next 前端建好了 React Query 基础设施，但截至目前 `useTableQuery` **只有一个真实消费者**（workstations 页）。其他 18 个列表页仍跑在 `useTableManager` 的命令式数据流上：每次进页面、每次切筛选、每次翻页都直接打后端，跨页无缓存，跨页签切换会重拉。

Phase 34 的目标：

1. 建立**可复用的列表页迁移范式**（companion 模式），让 18 个列表页按图施工
2. **Wave 1 验证三个难点**：invalidate 配套、useTableManager 哑函数、externalPagination.setTotal 同步
3. **Wave 2 / 3 按相同 checklist 批量化**，把已验证的范式应用到 VDI VM List 和 Assets
4. 跨页签二次进入从"必发请求"变成"< 50ms 命中缓存"

预期收益：

- 切部门 → 触发请求 → 30s 内切回原部门不再发请求（staleTime 命中）
- 跨页签切换：dict / dept / role / list 全部命中 5min/30min 缓存
- 12 个交互路径（列表加载、切部门、搜索、重置、翻页、quickCreate、edit、delete、batchDelete、刷新、加载统计、跨页签切换）的请求次数从 N×12 降到 N+少量

## 2. 30-03 基础设施回顾

| 资产 | 路径 | 现状 |
|------|------|------|
| `queryKeys` 工厂 | `src/lib/queryKeys.ts` | dict / dept / role / list 四族；`list.page(resource, params)` 已泛化 |
| `useDict(dictType)` | `src/hooks/useDict.ts` | 共享缓存,5min staleTime,30min gcTime |
| `useDeptTree()` | `src/hooks/useDeptTree.ts` | 包裹 `getDeptTree()`,8+ consumer 共享 |
| `useRoleList()` | `src/hooks/useRoleList.ts` | 全量 role 列表,role 页统计已用 |
| `useTableQuery<T>(...)` | `src/hooks/useTableQuery.ts` | list 数据获取,30s staleTime,keepPreviousData 占位 |
| QueryClient 默认 | `src/App.tsx` | 5min staleTime / 30min gcTime / `refetchOnWindowFocus: false` |

## 3. 关键决策

| ID | 决策 | 依据 |
|----|------|------|
| **D-17** | Companion 模式: useTableManager 保留 modal/form/selection,useTableQuery 接管数据 | 用户 2026-06-14 确认;与 30-03 决策一致 |
| **D-18** | filters 必须 `useMemo + JSON.stringify` 派生稳定引用,严禁 `searchForm.getFieldsValue()` 直传 | `getFieldsValue` 每次返回新对象,会触发 queryKey 抖动,引起重复请求 |
| **D-19** | statistics 与 list 数据分两个 queryKey(不同 resource 字符串) | 翻页 pageSize=10 与统计 pageSize=10000 业务意图不同;共享 cache 会互相污染 |
| **D-20** | useTableManager 哑函数策略:`async () => ({list:[], total:0})` | 不改 hook 公共契约,8+ 现有 consumer 保持兼容 |
| **D-21** | invalidate 只在 mutation 成功路径调用,翻页/搜索/重置不调 | React Query 通过 queryKey 变化自动 refetch,手动 invalidate 重复且会闪两次 |
| **D-22** | total 同步用 `useEffect(() => setTotal(total), [total, setTotal])` 写回 externalPagination | 哑函数不再触发 `externalPagination.setTotal`,必须手工补救 |

## 4. Wave 排序与候选

| Wave | 页面 | 文件 | 行数 | 验证重点 |
|------|------|------|-----:|----------|
| **Wave 1** | Network Devices | `src/pages/network/devices/index.tsx` | 973 | invalidate 配套 + 哑函数 + total 同步 + statistics 独立 query |
| **Wave 2** | VDI VM List | `src/pages/vdi/VirtualMachineList/index.tsx` | 1122 | 大页面从 0 接入;26 个 useState 精简;`useRealtimeUpdates` ws 与 React Query 协同 |
| **Wave 3** | Assets | `src/pages/operations/assets/index.tsx` | 688 | 52 列 + 自适应列 + virtual scroll;statistics 与 list 解耦(沿用 Wave 1) |

排序依据：

1. **Network Devices** 18 处 loadData 调用 + statistics 内嵌 queryFn,是最能验证完整 invalidate 配套的"压力测试"目标
2. **VDI VM List** 完全不用 useTableManager,是结构最纯的"重写"目标,但改造面积大,放 Wave 2 拿 Wave 1 范式套用
3. **Assets** 52 列 + 自适应列 + 已用 useColumnConfig,与 30-04 渲染层联调,放 Wave 3 让 Wave 1/2 的范式更稳定后再处理大表

后续候选（不在 Phase 34 内）：workstation、ad-domain ous、role、user、system post / config、operations buildings / floors / server-rooms、network mac / credentials / ports、operations rpa/* 等。

## 5. 范围与非范围

### In Scope

- 3 个 Wave 页面（Network Devices、VDI VM List、Assets）的 React Query 迁移
- 跨页通用的 6 维 checklist（数据层 / cache / invalidate / UX / 类型 / 测试）
- statistics 改独立 query（顺手修旧版"跨页累加"bug）
- Phase 34 文档骨架（CONTEXT / PLAN / 各 wave SUMMARY）

### Out of Scope

- `useTableManager.ts` 公共契约改动（D-20 反模式 #1）
- 30-03 deferred-items.md 中的 pre-existing TS 错误（保持 wave 边界）
- 网络层（axios 拦截器、SM2+SM4 加密）
- 后端 API 改造
- 其他 15 个列表页（Phase 35+ 范围）

## 6. 风险登记册

| 风险 | 触发场景 | 缓解 |
|------|----------|------|
| 哑函数不再触发 `externalPagination.setTotal`,分页 total 不变 | query 拉回数据但分页控件不更新 | useEffect 监听 queryResult.data.total 写回(D-22) |
| filters 直接传 `searchForm.getFieldsValue()` 引起 queryKey 抖动 | 每次 render 触发新请求 | useMemo + JSON.stringify 序列化(D-18) |
| statistics 与 list 共用一个 queryKey 互相污染 | 翻页触发 statistics 重算 | 不同 resource 字符串(D-19) |
| handleSearch 改本组件实现后,搜索字段校验不再走 useTableManager | 必填字段校验缺失 | 用 `searchForm.validateFields()` 替代 |
| invalidate 后 `setLoading(true)` 双 loading 闪烁 | 旧习惯代码残留 | 接受 React Query 自动管 isFetching |
| mutation 后两个 queryKey 各自 invalidate,UI 闪两次 | list + stats 同步刷新 | 接受 trade-off(30-03 D-12) |
| React Query 缓存陈旧导致数据不一致 | staleTime 30s 内看不到最新数据 | 临时可在 `App.tsx` 全局 `gcTime: 0` 排查 |
| statistics 改独立 query 后 pageSize=10000 大数据量 | 网络/解析开销 | 仅在 Wave 1 跑通后评估;必要时退化 pageSize=500 |
| WS 推送 (`useRealtimeUpdates`) 与 React Query 协同 | VDI VM List 已用 WS | Wave 2 处理;ws 推送时 invalidate 对应 list |
| 切换 dept 后旧分页 current 不重置 | handleDeptSelect 只 setSelectedDeptId | 显式 `setCurrent(1)` |

## 7. 度量指标

| 指标 | 当前(估算) | 目标(Wave 3 后) |
|------|-----------|-----------------|
| Network Devices 进页面请求数 | 4~6 | 1~2(dept + devices) |
| Network Devices 切部门请求数 | 2(departments 重拉 + devices 重拉) | 0~1(5min 内命中 dept 缓存) |
| 30s 内反复切回原部门 | 2 次 departments + 2 次 devices = 4 次 | 0 次(staleTime 命中) |
| VDI VM List 进页面请求数 | 26 个独立 useState 各发一次 | 1 个 useTableQuery + N 个 dropdown query |
| mutation 后陈旧数据刷新延迟 | 0(立即 loadData) | ≤ 30s(用户感知不可见) |
| 二次进入同页签请求数 | 全量 | ≤ 1(gcTime 30min 内命中) |

度量方法：

- DevTools Network 面板: 每个 wave 执行前后各录一次,对比
- React Query DevTools(若开启): 看 queryKey 数量、cache hit rate
- Chrome Performance: 对比首屏 TTI

## 8. 与其他阶段的边界

| 阶段 | 边界 |
|------|------|
| **30-03** | 提供 queryKeys / useTableQuery / useDict 等基础设施,Phase 34 只消费不改 |
| **30-04** | 渲染层 (memo / virtual scroll / BaseEditModal),Phase 34 不动其产物 |
| **30-05** | Companion 模式在 workstations 落地,Phase 34 沿用并扩展 |
| **33 vercel-react-best-practices** | 评审视角;Phase 34 改造后 ESLint 应通过 |
| **Phase 31** (历史规划中) | 可能处理 exhaustive-deps,Phase 34 自身不修 |

## Self-Check

- [x] 背景与目标明确(第 1 节)
- [x] 30-03 基础设施盘点完整(第 2 节)
- [x] 6 条关键决策可追溯(D-17 ~ D-22,第 3 节)
- [x] Wave 排序与候选明确(第 4 节)
- [x] 范围与非范围清晰(第 5 节)
- [x] 风险登记册 ≥ 5 条(第 6 节)
- [x] 度量指标 ≥ 3 条(第 7 节)
- [x] 与其他阶段边界明确(第 8 节)

**Self-Check: PASSED**

## Next Steps

- [ ] Phase 34 PLAN.md 占位(本目录下)
- [ ] Wave 1: 34-01-PLAN.md 详细执行计划
- [ ] Wave 1 执行 + 34-01-SUMMARY.md
- [ ] Wave 2 / 3 同模式批量化