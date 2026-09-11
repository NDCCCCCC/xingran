---
quick_id: 260911-m76
slug: js-misc
phase: 76-js-misc
plan: 06
type: execute
wave: 1
depends_on: []
files_modified:
  - xingran-react-frontend/src/components/DeptTree/index.tsx
  - xingran-react-frontend/src/pages/network/ports/index.tsx
  - xingran-react-frontend/src/pages/duty/my-duty/index.tsx
  - xingran-react-frontend/src/pages/workorder/statistics/index.tsx
  - xingran-react-frontend/src/pages/operations/info-points/index.tsx
  - xingran-react-frontend/src/pages/operations/dedicated-lines/index.tsx
  - xingran-react-frontend/src/pages/operations/room-devices/index.tsx
  - xingran-react-frontend/src/components/CronSelector/fields/WeekField.tsx
  - xingran-react-frontend/src/pages/network/credentials/index.tsx
  - xingran-react-frontend/src/pages/network/mac/index.tsx
  - xingran-react-frontend/src/pages/operations/building-spaces-3d/components/HubeiMapGL.tsx
  - xingran-react-frontend/src/pages/operations/building-spaces-3d/components/HubeiMap.tsx
autonomous: false
requirements: []
must_haves:
  truths:
    - DeptTree 双重遍历改为单遍（性能优化）
    - ports 批量写入选中集改为 useMemo + Set
    - 52 处静态映射提升到模块级（statusMap/colorMap/options 等）
    - 湖北地图 tooltip 坐标不再每像素 setState
  artifacts:
    - path: xingran-react-frontend/src/components/DeptTree/index.tsx
      does_not_contain: "filterFn(node.children).*filterFn"
    - path: xingran-react-frontend/src/pages/network/ports/index.tsx
      contains: "useMemo"
---

<objective>
JS 微性能优化（DeptTree 双遍历改单遍、ports 选中集优化、静态映射提升模块级、湖北地图 tooltip 坐标优化）。
</objective>

<context>
@xingran-react-frontend/src/components/DeptTree/index.tsx:159-178（双重遍历模式）
@xingran-react-frontend/src/pages/duty/my-duty/index.tsx:198-205,215-222（最差样本：render 回调内每行每格重建映射）
</context>

<tasks>

<task type="auto">
  <name>Task 1: DeptTree 双重遍历改单遍（6.1）</name>
  <files>xingran-react-frontend/src/components/DeptTree/index.tsx</files>
  <action>
第 159-178 行：filter 内 `filterFn(node.children)`（:169）与 map 内再次 `filterFn(node.children)`（:174）→ 改单遍：
1. 先递归算 children
2. 据 `titleMatch || children.length > 0` 决定保留
3. null 后统一 `filter(Boolean)`
</action>
  <verify>
    <automated>cd xingran-react-frontend && npm run type-check 2>&1 | tail -5</automated>
  </verify>
  <done>DeptTree 双重遍历改为单遍；type-check 通过</done>
</task>

<task type="auto">
  <name>Task 2: ports 批量写入选中集优化（6.2）</name>
  <files>xingran-react-frontend/src/pages/network/ports/index.tsx</files>
  <action>
第 815 行：`selectedPorts={portStatus.filter(p => selectedRowKeys.includes(p.id))}` → 改为 `useMemo` + `new Set(selectedRowKeys)`：
```ts
const selectedPorts = useMemo(() => {
  const keySet = new Set(selectedRowKeys);
  return portStatus.filter(p => keySet.has(p.id));
}, [portStatus, selectedRowKeys]);
```
</action>
  <verify>
    <automated>cd xingran-react-frontend && npm run type-check 2>&1 | tail -5</automated>
  </verify>
  <done>ports 选中集改为 useMemo + Set；type-check 通过</done>
</task>

<task type="auto">
  <name>Task 3: 静态映射提升模块级（6.3）</name>
  <files>
    xingran-react-frontend/src/pages/duty/my-duty/index.tsx
    xingran-react-frontend/src/pages/workorder/statistics/index.tsx
    xingran-react-frontend/src/pages/operations/info-points/index.tsx
    xingran-react-frontend/src/pages/operations/dedicated-lines/index.tsx
    xingran-react-frontend/src/pages/operations/room-devices/index.tsx
    xingran-react-frontend/src/components/CronSelector/fields/WeekField.tsx
    xingran-react-frontend/src/pages/network/credentials/index.tsx
    xingran-react-frontend/src/pages/network/mac/index.tsx
  </files>
  <action>
**最差样本 my-duty/index.tsx:198-205,215-222：** 表格 render 回调内每行每格重建 statusMap/colorMap/options → 提到组件顶部 `const`（模块级）。

**其余分布（grep 命中）：** statusMap/colorMap/options 等纯数据映射提到模块级 `const`；依赖组件内变量的不动。优先做 render 回调内与高频页。

参照既有模块级先例：`pages/network/devices/utils.tsx:24`。
</action>
  <verify>
    <automated>cd xingran-react-frontend && npm run type-check 2>&1 | tail -10</automated>
  </verify>
  <done>my-duty render 回调内映射提到模块级；高频页静态映射全部提升；type-check 通过</done>
</task>

<task type="auto">
  <name>Task 4: 湖北地图 tooltip 坐标优化（6.4）</name>
  <files>
    xingran-react-frontend/src/pages/operations/building-spaces-3d/components/HubeiMapGL.tsx
    xingran-react-frontend/src/pages/operations/building-spaces-3d/components/HubeiMap.tsx
  </files>
  <action>
**HubeiMapGL.tsx:80,421 与 HubeiMap.tsx:68,430：**
tooltip 跟随光标的 `setTooltipPosition` 每像素 setState → 选改动小的方案：
- 方案 A：tooltip 抽独立小组件自监听 mousemove
- 方案 B：坐标走 ref + 直接改 tooltip DOM transform

选择改动小的方案实施。
</action>
  <verify>
    <automated>cd xingran-react-frontend && npm run type-check 2>&1 | tail -5</automated>
  </verify>
  <done>湖北地图 tooltip 坐标不再每像素 setState；type-check 通过</done>
</task>

</tasks>

<verification>
cd xingran-react-frontend && npm run type-check && npm run lint && npm test
</verification>

<success_criteria>
- DeptTree 双重遍历改为单遍
- ports 选中集 useMemo + Set
- my-duty 最差样本的 render 回调内映射提到模块级
- 湖北地图 tooltip 不再每像素 setState
</success_criteria>

<output>
创建 `.planning/quick/260911-m76-fix-frontend-perf-audit-findings/260911-m76-06-PLAN-SUMMARY.md`
</output>
