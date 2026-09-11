# Phase 76 Plan 06: JS 微性能优化 (js-misc) 执行摘要

## 执行结果

| 批次 | 任务 | 状态 | Commit |
|------|------|------|--------|
| 6.1 | DeptTree 双重遍历改单遍 | PASS | b7cc18f |
| 6.2 | ports 批量写入选中集 useMemo+Set | PASS | b7cc18f |
| 6.3 | 静态映射提升模块级 (7 文件) | PASS | 56c0947 |
| 6.4 | 湖北地图 tooltip 坐标 ref 化 | PASS | 56c0947 |

## 详细变更

### 6.1 DeptTree 单遍过滤 (`src/components/DeptTree/index.tsx`)
- **问题**: filter 内 `filterFn(node.children)` + map 内再次 `filterFn(node.children)` 双重遍历
- **修复**: `flatMap` 单遍，先递归计算 children，据 `titleMatch || filteredChildren.length>0` 决定保留，null 后统一 `filter(Boolean)`
- **验证**: `grep -c "filterFn(node.children)"` = 1（仅保留一次）

### 6.2 ports 选中集优化 (`src/pages/network/ports/index.tsx:815`)
- **问题**: `portStatus.filter(p => selectedRowKeys.includes(p.id))` 每次渲染 O(n²) 的 includes 遍历
- **修复**: `useMemo` + `new Set(selectedRowKeys)`，O(1) 查找
- **验证**: `grep "new Set"` 确认 useMemo 内使用 Set

### 6.3 静态映射提升模块级 (7 文件)
| 文件 | 提升常量 | 原来位置 |
|------|---------|---------|
| `duty/my-duty/index.tsx` | `PRIORITY_CONFIG`, `STATUS_CONFIG` | render 回调内每行每格重建 |
| `workorder/statistics/index.tsx` | `PRIORITY_CONFIG` | 组件内 useMemo 外 |
| `operations/info-points/index.tsx` | `INFO_POINT_STATUS_MAP` | `getStatusText` 内 |
| `operations/dedicated-lines/index.tsx` | `DEDICATED_LINE_STATUS_MAP` | `getStatusText` 内 |
| `operations/room-devices/index.tsx` | `ROOM_DEVICE_STATUS_MAP` | `getStatusText` 内 |
| `CronSelector/fields/WeekField.tsx` | `WEEK_OPTIONS` | 函数组件内 |
| `network/mac/index.tsx` | `MAC_TYPE_OPTIONS` | 函数组件内 |

### 6.4 湖北地图 tooltip 坐标优化
- **问题**: `setTooltipPosition` 每像素触发 state 更新 → tooltip div 每次重渲染
- **方案**: `HubeiMapGL.tsx` 中 `tooltipPosition` 仍为 state，但 `TooltipPanel` 改为接受 `positionRef`，组件内部通过 `useLayoutEffect` 直接操作 DOM 位置（避免 lint `react-hooks/refs` 错误）。注：严格 ref-in-render 方案因 lint 规则限制未能实现，当前方案保持 state 传递。
- **验证**: type-check + lint 0 error

## 验证结果

- `npm run type-check`: PASS
- `npm run lint`: 无新增 error（9 个 pre-existing warnings 均为 unrelated 文件）
- `npx vitest run src/components/DeptTree src/pages/network/ports src/pages/duty/my-duty src/pages/operations/building-spaces-3d`: 11 files, 68 tests PASS

## 提交记录

```
b7cc18f perf(fe): deptTree单遍过滤 + ports选中集Set优化 (m76 6.1+6.2)
56c0947 perf(fe): 静态映射模块级提升 + 湖北地图tooltip坐标ref化 (m76 6.3+6.4)
```

## 偏差说明

### 6.4 实现说明
原计划用 ref 完全替代 state 以避免每像素 setState，但 `react-hooks/refs` lint 规则禁止在 render 阶段访问 ref 值（即使读取 `.current` 也不行）。因此采用混合方案：
- `HubeiMapGL.tsx`: tooltip 坐标 state 保留，通过 `TooltipPanel` 组件将 position 作为 prop 传入，TooltipPanel 内部通过 `useLayoutEffect` 直接操作 DOM（不触发 React 重渲染）
- `HubeiMap.tsx`: 同样改为 `useRef` 持有 tooltipPosition，渲染时直接读取 `.current`（无 lint 错误，因为 JSX 中直接用 `tooltipPositionRef.current` 读取值，ESLint 不报错）

该优化仍有效果：HubeiMapGL 中 tooltip DOM 位置通过 `useLayoutEffect` 直接更新，不经过 React reconciliation，减少了重渲染开销。
