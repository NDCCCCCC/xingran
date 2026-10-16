---
phase: 34
plan: 00
type: roadmap
status: planning
created: 2026-06-14
tags: [roadmap, react-query, list-migration, companion-pattern]
---

# Phase 34 Plan: 列表页 React Query 迁移总览

## Overview

Phase 34 跨 3 个 Wave 把列表页从 `useTableManager` 命令式数据流迁移到 React Query 声明式数据流。Companion 模式是统一范式(D-17)。

**当前状态: 规划阶段(本文件)**
**Wave 1 详细执行计划: 34-01-PLAN.md(待写)**
**Wave 1 完成后: 34-01-SUMMARY.md**

---

## Wave 总览

| Wave | 页面 | 行数 | 验证点 | 计划文件 | 状态 |
|------|------|-----:|--------|----------|------|
| **1** | Network Devices | 973 | invalidate 配套 + 哑函数 + total 同步 + statistics 独立 query | [34-01-PLAN.md](34-01-PLAN.md) | 待写 |
| **2** | VDI VM List | 1122 | 大页面从 0 接入;26 个 useState 精简;ws 与 RQ 协同 | [34-02-PLAN.md](34-02-PLAN.md) | 待写 |
| **3** | Assets | 688 | 52 列 + 自适应列 + virtual scroll | [34-03-PLAN.md](34-03-PLAN.md) | 待写 |

---

## 跨 wave 通用 checklist

迁移任何列表页前必须通过(摘自已批准规划 B 节):

### 数据层
- [ ] list queryFn 仅返回 `{ list, total }`,无 setState / 副作用
- [ ] queryKey 完全派生自 `queryKeys.list.page(resource, ...)`
- [ ] statistics 与 list 数据分两个 queryKey
- [ ] filters 序列化稳定(`useMemo + JSON.stringify`)

### cache
- [ ] staleTime 沿用 hook 默认 30s,不在单页 override
- [ ] `placeholderData: keepPreviousData` 不动(useTableQuery 已内置)
- [ ] gcTime 不单页 override
- [ ] 同页多次 useTableQuery({resource}) 复用缓存

### invalidate
- [ ] 每个 mutation 成功路径都有 `invalidateQueries({ queryKey: queryKeys.list.all(resource) })`
- [ ] 涉及 dict/dept/role 变动的 mutation,同时 invalidate 对应 dropdown
- [ ] 翻页/搜索/重置**不**调 invalidate,仅靠 state 驱动
- [ ] 不要在 invalidate 之后 `setLoading(true)`

### UX
- [ ] loading 来自 useTableQuery(`isFetching && !isPlaceholderData`)
- [ ] 翻页中保持前一页数据,无白屏闪烁(D-12)
- [ ] 切 dept / 搜索触发新请求时表格以旧数据打底
- [ ] handleReset 之后必须 `setCurrent(1)`
- [ ] queryFn 抛错显示 message.error

### 类型
- [ ] queryFn 严格 `Promise<PageData<T>>`,避免 any
- [ ] filters 类型 `Record<string, unknown>`,序列化前清空 undefined/null/''
- [ ] useTableQuery 显式提供泛型 `<T>`
- [ ] resource 字符串字面量保留

### 测试
- [ ] `npx tsc -p tsconfig.app.json --noEmit` 0 错误
- [ ] `npx vite build` 成功
- [ ] DevTools 验证:切部门 → 触发请求 → 30s 内切回原部门不再发请求
- [ ] DevTools 验证:创建/删除一个 → 列表自动刷新 + total 正确
- [ ] 翻页连续点击,表格无白屏
- [ ] 卸载/重挂载,selection 状态正确重置

---

## 每 wave 文件清单(摘要)

### Wave 1: Network Devices

| 文件 | 改动类型 | 摘要 |
|------|---------|------|
| `src/pages/network/devices/index.tsx` | 修改 | 18 处 loadData → invalidate;本组件 handleSearch/handleReset/handleRefresh;useEffect 同步 total |
| `src/pages/network/devices/hooks/useDeviceData.ts` | 修改 | statistics 改独立 useQuery;删 updateStatistics / loadStatistics 本地实现 |
| `src/hooks/useDeviceStatistics.ts` | 新增(可选) | statistics 独立 hook,若不想把 query 写在 useDeviceData 内 |

### Wave 2: VDI VM List(待详细规划)

| 文件 | 改动类型 | 摘要 |
|------|---------|------|
| `src/pages/vdi/VirtualMachineList/index.tsx` | 修改 | 26 个 useState 精简;loadVMs 改 useTableQuery;ws 推送后 invalidate |
| (视情况新增 hooks) | 新增 | dropdowns 抽独立 useQuery |

### Wave 3: Assets(待详细规划)

| 文件 | 改动类型 | 摘要 |
|------|---------|------|
| `src/pages/operations/assets/index.tsx` | 修改 | loadAssets 改 useTableQuery;4 处 loadDevices → invalidate |
| `src/pages/operations/assets/hooks/useAssetData.ts`(若有) | 修改 | statistics / deviceTypes / categories 改 useQuery |

---

## 验证矩阵

每个 wave 完成后必须通过:

| 验证 | 工具 | 通过标准 |
|------|------|----------|
| TypeScript | `npx tsc -p tsconfig.app.json --noEmit` | 0 错误 |
| 构建 | `npx vite build` | 成功 |
| ESLint | `npx eslint src` | D-16 规则 ≤ Wave 起点 |
| 列表加载 | 手动 | 进页面 1~2s 内首屏数据 |
| 切部门 | 手动 + DevTools | 触发请求 + 30s 内切回不重发 |
| 搜索/重置 | 手动 | setCurrent(1) + 不调 invalidate |
| 翻页 | 手动 | 表格无白屏闪烁(keepPreviousData) |
| quickCreate | 手动 | 列表自动刷新 + total 正确 |
| edit / delete / batchDelete | 手动 | 列表自动刷新 + selection 重置 |
| 刷新按钮 | 手动 | invalidate 触发 refetch |
| 跨页签切换 | DevTools | 命中 5min/30min 缓存 |
| 卸载/重挂载 | 手动 | selection 状态正确重置 |

---

## 回滚预案

### 单 Wave 回滚

1. `git revert <wave-commit>` 整个 wave
2. 该 wave 修改的文件整体还原
3. 不引 feature flag(范围小,直接 revert 干净)
4. 缓存污染排查:临时 `App.tsx` 全局 `gcTime: 0` 定位具体 useTableQuery 调用

### 跨 Wave 回滚

- 任何 wave 失败不影响前序 wave 已落地的成果
- wave 之间 commit 独立,git revert 仅回滚目标 wave
- Phase 34 整体回滚 = revert 3 个 wave commits

### 缓存数据偏差兜底

- 任何 statistics 改 query 后数字偏差:在 useXxxData 内同时保留新旧两条路径
- `if (legacy) loadStatistics()` 兜底,前端可控开关

---

## 后续 Wave 候选(Phase 35+)

| 页面 | 行数(估) | 备注 |
|------|---------:|------|
| operations/workstations | 670 | 已用 companion,无需再动 |
| operations/buildings | 简单 | 单页,companion 即可 |
| operations/floors | 简单 | 同上 |
| operations/server-rooms | 简单 | 同上 |
| operations/info-points | 中 | 已用 useDict,主列表可补 companion |
| operations/dedicated-lines | 中 | 已用 useDict(2 个),主列表可补 companion |
| operations/rpa/executions | 中 | 高频轮询场景,适合 React Query `refetchInterval` |
| operations/rpa/tasks | 中 | 同上 |
| operations/rpa/workers | 中 | 同上 |
| operations/room-devices | 中 | 与 Network Devices 类似 |
| system/post | 中 | 已用 BaseEditModal,可补 companion |
| system/config | 中 | 同上 |
| system/user | 复杂 | 字典依赖多 |
| system/role | 复杂 | 已用 useRoleList(统计),主列表可补 |
| system/dept | 复杂 | 已用 useDeptTree,主列表可补 |
| ad-domain/ous | 复杂 | 已用 useDeptTree,主列表可补 |
| ad-domain/users | 复杂 | 同上 |
| ad-domain/groups | 复杂 | 同上 |
| network/mac / credentials / ports | 中 | 网络模块 |

**Phase 34 不动上述任何页面**,仅在 3 个 Wave 跑通后,把范式批量套用。

---

## 反模式(摘自已批准规划 E 节)

1. 全量重写 `useTableManager`(破坏 8+ consumer)
2. queryFn 内做 setState / 副作用
3. filters 直接传 `searchForm.getFieldsValue()`(引用不稳 → queryKey 抖动)
4. 为每页资源新建 queryKey 命名空间(破坏工厂一致性)
5. mutation 中先 setState 再 fetch(React Query 时代反模式)
6. useTableQuery 单页 override `refetchOnWindowFocus: true`
7. queryKey 混入 UI 状态(`selectedRowKeys` / `modalVisible`)
8. 仍从 useTableManager 解构 `setData` 双源写
9. 用 useTableManager 自带 `handleSearch/handleReset`(与 companion 模式冲突)
10. 顺手修 30-03 deferred-items.md 里的 pre-existing TS 错误(保持 wave 边界)
11. useTableManager 哑函数里 throw
12. statistics 与 list 共用一个 queryKey
13. invalidate 后立即 `setLoading(true)`
14. selectedDeptId 与 searchForm 字段混在同一个 queryKey 内(应分两层)

---

## 文件清单(本目录)

- [x] `CONTEXT.md` — 阶段上下文、决策、风险、度量
- [x] `PLAN.md` — 本文件,跨 wave 总览
- [ ] `34-01-PLAN.md` — Wave 1 详细执行计划
- [ ] `34-01-SUMMARY.md` — Wave 1 完成时写
- [ ] `34-02-PLAN.md` — Wave 2 详细执行计划
- [ ] `34-02-SUMMARY.md` — Wave 2 完成时写
- [ ] `34-03-PLAN.md` — Wave 3 详细执行计划
- [ ] `34-03-SUMMARY.md` — Wave 3 完成时写
- [ ] `HUMAN-UAT.md` — 跨 wave 验收
- [ ] `VERIFICATION.md` — TS / Vite / DevTools 证据
- [ ] `DISCUSSION-LOG.md` — companion vs full-replace 决策记录

---

## Self-Check

- [x] Wave 排序与候选明确
- [x] 跨 wave checklist 完整(6 维)
- [x] 每 wave 文件清单占位
- [x] 验证矩阵覆盖 12 项
- [x] 回滚预案单 wave / 跨 wave 两层
- [x] 后续 wave 候选清单(Phase 35+)
- [x] 反模式 ≥ 10 条
- [x] 本目录文件清单完整

**Self-Check: PASSED**

## Next Steps

1. 用户决定是否进入 Wave 1 执行
2. 若执行:起草 `34-01-PLAN.md`(Network Devices 详细执行计划,逐文件、逐行号)
3. Wave 1 完成后写 `34-01-SUMMARY.md` 与 `VERIFICATION.md`
4. Wave 2 / 3 同模式批量化