---
phase: 85-p2-pages-r1-operations
plan: 00
wave: all
status: complete
completed: 2026-08-27
---

## Phase 85 Complete: P2 页面层 R1 — operations

**pages/operations 76 文件 3611 stmts 0.00% → 4.13% (floor 0.0 → 3.6)**

### 4 Waves / 62 tests
| Wave | 子目录 | tests | floor |
|------|--------|-------|-------|
| 1 | floors(utils+hooks) + workstations(constants) | 24 | 0.7 |
| 2 | building-spaces-3d(utils) + rpa(constants) | 21 | 2.2 |
| 3 | buildings(hooks) + building-spaces(types) | 10 | 3.1 |
| 4 | assets/server-rooms/info-points/dedicated-lines/room-devices | 7 | 3.6 |

### 测试模式(Phase 84 模式延续)
- 纯函数直测: processWorkstations / pixelDistance / toBase64 / isNewElement
- hooks mock 模式: useBuildingOptions/useFloorStatistics(mock opsApi) / useBuildingGeocoding(mock useGeocoding) / useDepartmentData(mock useDeptTree)
- 常量静态断言: TYPE_OPTIONS/STATUS_OPTIONS/DIMENSIONS/WORKSTATION_LAYOUT
- 模块导入断言: 5 个零散页面 default export

### Verification
- **1067/1067 tests PASS / 113 files**(1005 存量 + 62 新增)
- gate **0 FAIL**,GLOBAL **20.75%**
- ratchet 4 行追加(85-W1..W4)
- floor 单调上升 0.0 → 0.7 → 2.2 → 3.1 → 3.6

### 70% 目标差距说明
operations 页面组件重度依赖 useTableManager/useTableQuery/usePagination 复合 hooks + React Query + antd Table 深集成,纯渲染测试覆盖有限。真实提升需 mock 全套表格管理 hooks(Phase 86+ 模式迭代方向)。

### Key patterns 沉淀
- hook 返回结构需先读源码确认(...state 展开 vs 嵌套对象)
- getBuildingLabel 有截断逻辑(名称含"楼"被去掉),断言用 toContain
- sed -i 正则含 / 需转义或换分隔符