# Phase 76 Plan 05: CAD/3D Render Performance — Summary

**Batch 5 of quick 260911-m76** | 2026-09-11

## One-liner
CAD 渲染性能优化：snapCoord 取整、lastMousePos ref 化、cad-elements React.memo、3D useFrame 收敛即停、SVG transition 改 transform、MAC 时间线 memo+content-visibility、CAD 查找 Map 化、死代码删除。

---

## Commits

| Hash | Message |
|------|---------|
| `93d047a` | perf(cad-3d-render): snapCoord helper, lastMousePos ref, cad-elements React.memo |
| `550f7a4` | perf(cad-3d-render): useFrame convergence, SVG transition, memo, Map lookup, dead code |

---

## Task Completion

### 5.1 SVG 坐标精度 snapCoord helper
- 新建 `src/components/cad-elements/coord.ts`：`snapCoord(v) = Math.round(v * 100) / 100`
- 应用于 `WorkstationElement.tsx` 的 `rotatePoint` 和 `rotateChairLocal`
- 应用于 `DoorElement.tsx` 的 `leafEndX/Y` 和 `leafEndX2/Y2`（双开门第二扇）
- grep 自证：`grep -c "snapCoord"` → WorkstationElement 5 / DoorElement 5 / coord.ts 1

### 5.2 CAD 编辑器高频态走 ref
- `lastMousePos` state → `lastMousePosRef = useRef<Point | null>(null)`
- 清除 `handleCanvasMouseMove` useCallback deps 中的 `lastMousePos`
- `boxSelectEnd` 保留 state（需视觉反馈）

### 5.3 cad-elements 组件 memo 化
- `WallElement` / `DoorElement` / `WorkstationElement` / `CADTextElement` 全部 `React.memo` 包裹
- `CADFloorPlanEditor.tsx` map 内联回调 `() => handleSelectElement(id, type)` 改为 `handleSelectElement.bind(null, id, type)` 稳定引用

### 5.4 3D useFrame 收敛即停
- `FloorPlan3D.tsx:98` 工位悬停 y 位置：收敛判定 `Math.abs(targetY - cur) < 1e-3` 时直接赋值并 return
- `BuildingModel3D.tsx:41` 楼层悬停缩放：收敛判定 `Math.abs(targetScale - cur) < 1e-3` 时直接赋值并 return
- 减少 60fps 常驻回调

### 5.5 SVG 几何属性 CSS transition
- `CADFloorPlanEditor.less` `.cad-wall-control-point` / `.cad-door-control-point`：`transition: r` → `transform: scale(1)` + `transform-box: fill-box` + `transform-origin: center` + `transition: transform 0.15s ease`；`:hover` 改 `scale(1.5)`
- `index.css` `.sidebar-toggle` transition 从 `svg` 移至按钮层 `.sidebar-toggle.is-collapsed { transform: translateY(-50%) rotate(180deg) }`

### 5.6 MAC 时间线累积列表
- `TimelineItem` 包 `React.memo`
- 累积容器 Timeline 包裹 `<div style={{ contentVisibility: "auto", containIntrinsicSize: "64px" }}>`

### 5.7 CAD 按 id 查找 Map 化
- `_selectedElements` useMemo：构建 `Map<id, {el, type}>`（O(n)），多选查找 O(1)
- `findNearbyWallNode`：平方距离比较 `dx*dx + dy*dy < threshold*threshold` 替代 sqrt

### 5.8 平面图视图死代码
- `FloorPlanView.tsx:156` 删除 `const _fullWorkstation = allWorkstations.find(...)` 无引用变量

---

## Verification Results

| Check | Result |
|-------|--------|
| `npm run type-check` | PASS (0 errors) |
| `npm run lint` | PASS (0 new errors in changed files) |
| `npx vitest run src/components/cad src/pages/operations/workstations` | PASS (14 files, 108 tests) |
| `grep "Math.round(v \* 100) / 100" cad-elements/*.tsx` | PASS (coord.ts contains helper) |
| `grep "lastMousePos" CADFloorPlanEditor.tsx` (no ref) | PASS (empty — only lastMousePosRef) |
| `grep "Math.abs" FloorPlan3D.tsx BuildingModel3D.tsx` | PASS (2 occurrences — convergence checks) |
| `grep "_fullWorkstation" FloorPlanView.tsx` | PASS (empty — dead code deleted) |

---

## Deviations
- 5.3 内联回调稳定化：未改为 useCallback，而是用 `.bind()` 替代箭头函数闭包（语义相同，更简洁）
- 5.5 sidebar-toggle：直接在 `.sidebar-toggle.is-collapsed` 上用 transform，移除原 `.is-collapsed svg` 旋转规则（transition 在按钮层）

---

## Files Changed

| File | Change |
|------|--------|
| `src/components/cad-elements/coord.ts` | 新建 — snapCoord helper |
| `src/components/cad-elements/WorkstationElement.tsx` | snapCoord 应用 + React.memo |
| `src/components/cad-elements/DoorElement.tsx` | snapCoord 应用 + React.memo |
| `src/components/cad-elements/WallElement.tsx` | React.memo |
| `src/components/cad-elements/TextElement.tsx` | React.memo |
| `src/components/cad-editor/CADFloorPlanEditor.tsx` | lastMousePos ref + Map lookup + sqrt 移除 + bind 回调 |
| `src/components/cad-editor/CADFloorPlanEditor.less` | transform scale 替代 r transition |
| `src/index.css` | sidebar-toggle transition 移至按钮层 |
| `src/pages/operations/building-spaces-3d/components/FloorPlan3D.tsx` | useFrame 收敛即停 |
| `src/pages/operations/building-spaces-3d/components/BuildingModel3D.tsx` | useFrame 收敛即停 |
| `src/components/network/MACEventsTimeline.tsx` | TimelineItem React.memo + content-visibility |
| `src/pages/operations/workstations/views/FloorPlanView.tsx` | _fullWorkstation 死代码删除 |

---

## Self-Check
- [x] All changed files exist
- [x] Both commit hashes verified in git log
- [x] Type-check passes
- [x] Lint passes (no new errors in changed files)
- [x] Tests pass (108/108)
- [x] All grep self-checks pass
