---
quick_id: 260911-m76
slug: cad-3d-render
phase: 76-cad-3d-render
plan: 05
type: execute
wave: 1
depends_on: []
files_modified:
  - xingran-react-frontend/src/components/cad-elements/WorkstationElement.tsx
  - xingran-react-frontend/src/components/cad-elements/DoorElement.tsx
  - xingran-react-frontend/src/components/cad-elements/WallElement.tsx
  - xingran-react-frontend/src/components/cad-elements/TextElement.tsx
  - xingran-react-frontend/src/components/cad-editor/CADFloorPlanEditor.tsx
  - xingran-react-frontend/src/components/cad-editor/CADFloorPlanEditor.less
  - xingran-react-frontend/src/index.css
  - xingran-react-frontend/src/pages/operations/building-spaces-3d/components/FloorPlan3D.tsx
  - xingran-react-frontend/src/pages/operations/building-spaces-3d/components/BuildingModel3D.tsx
  - xingran-react-frontend/src/components/network/MACEventsTimeline.tsx
  - xingran-react-frontend/src/pages/operations/workstations/views/FloorPlanView.tsx
autonomous: false
requirements: []
must_haves:
  truths:
    - CAD 元素坐标精度取整到 0.01（snapCoord helper）
    - CAD 编辑器 lastMousePos state 改为 ref（高频态走 ref）
    - cad-elements 组件全部 React.memo 化
    - 3D useFrame 收敛即停（lerp 收敛判定 < 1e-3 时直接赋值 return）
    - SVG transition 改 transform scale 方案
    - MACEventsTimeline TimelineItem React.memo + content-visibility
    - CAD 按 id 查找 Map 化（O(n) → O(1)）
    - FloorPlanView 死代码 _fullWorkstation O(n²) find 已删除
  artifacts:
    - path: xingran-react-frontend/src/components/cad-elements/
      contains: "React.memo"
    - path: xingran-react-frontend/src/components/cad-editor/CADFloorPlanEditor.tsx
      contains: "Map<id"
---

<objective>
优化 CAD/3D 渲染性能（坐标精度、高频态优化、memo 化、3D useFrame 收敛、Map 化查找）。
</objective>

<context>
@xingran-react-frontend/src/components/cad-elements/WorkstationElement.tsx:76-91,308-324（rotatePoint 坐标取整）
@xingran-react-frontend/src/components/cad-editor/CADFloorPlanEditor.tsx:127（lastMousePos state）
</context>

<tasks>

<task type="auto">
  <name>Task 1: SVG 坐标精度 snapCoord helper（5.1）</name>
  <files>
    xingran-react-frontend/src/components/cad-elements/WorkstationElement.tsx
    xingran-react-frontend/src/components/cad-elements/DoorElement.tsx
  </files>
  <action>
在 `src/components/cad-elements/` 内新建小工具文件 `coord.ts`：
```ts
/** 坐标取整到 0.01，防止 SVG 亚像素抖动 */
export function snapCoord(v: number): number {
  return Math.round(v * 100) / 100;
}
```

WorkstationElement.tsx 的 `rotatePoint`（:76-91, :308-324）和 DoorElement.tsx 的三角函数出口坐标（:90-104, :159-185）全部替换为 `snapCoord(v)`。WallElement/TextElement 同理（若存在同样模式）。
</action>
  <verify>
    <automated>cd xingran-react-frontend && npm run type-check 2>&1 | tail -5</automated>
  </verify>
  <done>snapCoord helper 创建；WorkstationElement/DoorElement 三角函数出口全部取整</done>
</task>

<task type="auto">
  <name>Task 2: CAD 编辑器高频态走 ref（5.2）</name>
  <files>xingran-react-frontend/src/components/cad-editor/CADFloorPlanEditor.tsx</files>
  <action>
1. 第 127 行 `lastMousePos` state → ref（周围 :129-133 已全是 ref，模式现成）
2. 第 630 行非 drag 态的 `setState` 同步改 ref
3. 确认 `boxSelectEnd` 保留 state（需视觉反馈）
</action>
  <verify>
    <automated>grep -n "lastMousePos" xingran-react-frontend/src/components/cad-editor/CADFloorPlanEditor.tsx</automated>
  </verify>
  <done>lastMousePos 改为 ref；type-check 通过</done>
</task>

<task type="auto">
  <name>Task 3: cad-elements 组件 memo 化（5.3）</name>
  <files>
    xingran-react-frontend/src/components/cad-elements/WallElement.tsx
    xingran-react-frontend/src/components/cad-elements/DoorElement.tsx
    xingran-react-frontend/src/components/cad-elements/WorkstationElement.tsx
    xingran-react-frontend/src/components/cad-elements/TextElement.tsx
    xingran-react-frontend/src/components/cad-editor/CADFloorPlanEditor.tsx
  </files>
  <action>
**每个组件**用 `React.memo` 包裹：
```ts
export const WallElement = React.memo(function WallElement(props: WallElementProps) { ... });
```

**CADFloorPlanEditor.tsx:1399-1449 的内联回调：**
`onSelect={() => ...}` / `onHover={...}` 改为 `useCallback` 稳定引用，或改为 (id, type) 数据式回调（改动子组件签名时保持调用语义）。
</action>
  <verify>
    <automated>grep -n "React.memo" xingran-react-frontend/src/components/cad-elements/*.tsx | head -20</automated>
  </verify>
  <done>WallElement/DoorElement/WorkstationElement/TextElement 全部 React.memo；内联回调稳定化</done>
</task>

<task type="checkpoint:human-verify" gate="blocking">
  <name>Task 4: CAD 渲染验证</name>
  <what-built>CADFloorPlanEditor（坐标精度 ref 化 + memo 化）</what-built>
  <how-to-verify>
1. 打开楼层平面图编辑器（/operations/floors）
2. 拖拽墙/门/工位元素，确认无亚像素抖动
3. 快速移动鼠标，确认 lastMousePos 更新流畅（无视觉卡顿）
4. 打开 DevTools > Performance > 60fps 确认
  </how-to-verify>
  <resume-signal>Type "approved" or describe issues</resume-signal>
</task>

<task type="auto">
  <name>Task 5: 3D useFrame 收敛即停（5.4）</name>
  <files>
    xingran-react-frontend/src/pages/operations/building-spaces-3d/components/FloorPlan3D.tsx
    xingran-react-frontend/src/pages/operations/building-spaces-3d/components/BuildingModel3D.tsx
  </files>
  <action>
**FloorPlan3D.tsx（约第 516 行附近）：**
每个工位实例化的 useFrame 回调中，lerp 收敛判定 `Math.abs(target - cur) < 1e-3` 时直接赋值并 return，避免 60fps 常驻回调：
```ts
if (Math.abs(target - cur) < 1e-3) {
  setValue(target);
  return;
}
```

**BuildingModel3D.tsx:41：** 同理添加收敛判定。
</action>
  <verify>
    <automated>cd xingran-react-frontend && npm run type-check 2>&1 | tail -5</automated>
  </verify>
  <done>FloorPlan3D/BuildingModel3D useFrame 添加收敛判定；type-check 通过</done>
</task>

<task type="auto">
  <name>Task 6: SVG transition 改 transform scale + MAC 时间线优化（5.5/5.6）</name>
  <files>
    xingran-react-frontend/src/components/cad-editor/CADFloorPlanEditor.less
    xingran-react-frontend/src/index.css
    xingran-react-frontend/src/components/network/MACEventsTimeline.tsx
  </files>
  <action>
**CADFloorPlanEditor.less:27-34：**
`.cad-wall-control-point` 的 `transition: r` + `:hover { r: 6 }` → 改为 transform scale 方案：
```less
.cad-wall-control-point {
  transform: scale(1);
  transform-box: fill-box;
  transform-origin: center;
  transition: transform 0.15s ease;
  &:hover { transform: scale(1.5); }
}
```

**index.css:4887-4894：** `.sidebar-toggle svg` 的 transform transition 移到按钮包裹层。

**MACEventsTimeline.tsx:188-201：**
TimelineItem 包 `React.memo`；累积容器加 `content-visibility: auto; contain-intrinsic-size: 64px`（内联 style）。
</action>
  <verify>
    <automated>cd xingran-react-frontend && npm run type-check 2>&1 | tail -5</automated>
  </verify>
  <done>SVG transition 改 transform scale；TimelineItem memo + content-visibility；type-check 通过</done>
</task>

<task type="auto">
  <name>Task 7: CAD 按 id 查找 Map 化 + 死代码删除（5.7/5.8）</name>
  <files>
    xingran-react-frontend/src/components/cad-editor/CADFloorPlanEditor.tsx
    xingran-react-frontend/src/pages/operations/workstations/views/FloorPlanView.tsx
  </files>
  <action>
**CADFloorPlanEditor.tsx:407-426（_selectedElements useMemo）：**
floorPlanData 变化时建 `Map<id, {el, type}>`（O(n)），多选查找走 O(1)。

**:601-614 findNearbyWallNode：**
用平方距离比较（`dx*dx+dy*dy < threshold*threshold`）替代 sqrt。

**FloorPlanView.tsx:156：**
删除 `_fullWorkstation` 的 O(n²) find（无任何引用）。
</action>
  <verify>
    <automated>grep -n "_fullWorkstation" xingran-react-frontend/src/pages/operations/workstations/views/FloorPlanView.tsx</automated>
  </verify>
  <done>_selectedElements 改为 Map 查找；sqrt 改为平方距离比较；_fullWorkstation 死代码已删除</done>
</task>

</tasks>

<verification>
cd xingran-react-frontend && npm run type-check && npm run lint && npm test
</verification>

<success_criteria>
- CAD 元素坐标无亚像素抖动
- lastMousePos 高频态走 ref
- cad-elements 全部 React.memo 化
- 3D useFrame 收敛即停
- SVG transition 改 transform scale
- MACEventsTimeline TimelineItem memo + content-visibility
- CAD 查找 O(1) Map
- _fullWorkstation 死代码已删除
</success_criteria>

<output>
创建 `.planning/quick/260911-m76-fix-frontend-perf-audit-findings/260911-m76-05-PLAN-SUMMARY.md`
</output>
