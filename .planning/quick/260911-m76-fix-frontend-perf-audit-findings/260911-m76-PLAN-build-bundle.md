---
quick_id: 260911-m76
slug: build-bundle
phase: 76-build-bundle
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - xingran-react-frontend/vite.config.ts
  - xingran-react-frontend/src/lib/echarts.ts
  - xingran-react-frontend/src/components/charts/EChartsWrapper.tsx
  - xingran-react-frontend/src/pages/system/notice/components/NoticeForm.tsx
autonomous: true
requirements: []
must_haves:
  truths:
    - 首屏 JS gzip 体积从 ~1.43MB 降至 ~850KB 以内
    - vendor-echarts chunk 不再包含全量 echarts-for-react 导入
    - echarts 图表（line/bar/pie/area）在 dashboard 和 MAC 热力图正常渲染
    - vendor-react modulepreload 不再包含 vendor-markdown
    - @uiw/react-md-editor CSS 随懒加载 chunk 加载，不在首屏
  artifacts:
    - path: xingran-react-frontend/vite.config.ts
      contains: "vendor-md-editor"
    - path: xingran-react-frontend/src/lib/echarts.ts
      contains: "LineChart, BarChart, PieChart"
    - path: xingran-react-frontend/src/components/charts/EChartsWrapper.tsx
      contains: "echarts-for-react/esm/core"
    - path: xingran-react-frontend/src/pages/system/notice/components/NoticeForm.tsx
      does_not_contain: "@uiw/react-md-editor/markdown-editor.css"
  key_links:
    - from: vite.config.ts
      to: vendor-md-editor chunk
      via: manualChunks rule for @uiw/react-md-editor
    - from: echarts.ts
      to: EChartsWrapper.tsx
      via: default export used in lazy load
    - from: NoticeForm.tsx
      to: MarkdownEditor.tsx
      via: CSS import removed from static, moved to lazy side
---

<objective>
削减首屏 JS bundle 体积（1.43MB gzip → ~850KB gzip）。纯配置 + 图表入口改动，零业务逻辑变化。
</objective>

<context>
@xingran-react-frontend/vite.config.ts
@xingran-react-frontend/src/lib/echarts.ts
@xingran-react-frontend/src/components/charts/EChartsWrapper.tsx
@xingran-react-frontend/src/pages/system/notice/components/NoticeForm.tsx
@xingran-react-frontend/src/components/markdown/MarkdownEditor.tsx

**Existing echarts.ts registration (partial):** CustomChart + TitleComponent/TooltipComponent/GridComponent/DataZoomComponent + CanvasRenderer。缺少 LineChart, BarChart, PieChart（dashboard line/bar/pie/area 需要）。

**Existing EChartsWrapper.tsx lazy pattern:** `lazy(() => import("echarts-for-react/esm/index"))`，传入全量导出对象，从未使用 `@/lib/echarts` 的按需注册实例。
</context>

<tasks>

<task type="auto">
  <name>Task 1: vite.config.ts 删除 echarts-for-react 特殊分支 + 加 preload-helper 规则</name>
  <files>xingran-react-frontend/vite.config.ts</files>
  <action>
1. 删除 `vite.config.ts` 第 189-191 行的 echarts-for-react 特殊分支（`if (id.includes("echarts-for-react")) return "vendor-react";`），让 echarts-for-react 落入后面的 `pkgName === "echarts" || "zrender"` 规则所在 chunk（vendor-echarts）。
2. 更新第 187-191 行附近的注释（原注释解释该特殊分支，已失效）。
3. 在 manualChunks 函数开头（`if (!id.includes("node_modules")) return undefined;` 之前）加：
   ```ts
   // Vite preload helper（\0vite/preload-helper.js）被所有动态 import 共享，
   // 单独成 chunk 阻止它被分配进 vendor-three 造成 entry→vendor-three 静态边
   if (id.includes("preload-helper")) return "runtime-helper";
   ```
   放在所有其他分支之前（最早匹配优先）。
</action>
  <verify>
    <automated>cd xingran-react-frontend && npm run build 2>&1 | grep -E "vendor-echarts|modulepreload" | head -20</automated>
  </verify>
  <done>echarts-for-react 特殊分支已删除；runtime-helper chunk 规则已加入；build 无错误</done>
</task>

<task type="auto">
  <name>Task 2: vite.config.ts 修 @uiw/ 兜底钉死 markdown 生态</name>
  <files>xingran-react-frontend/vite.config.ts</files>
  <action>
在 `if (pkgName.startsWith("@uiw/"))` 分支（第 169-171 行）**之前**加：
```ts
if (pkgName === "@uiw/react-markdown-preview" || pkgName === "react-markdown") {
  return "vendor-md-editor";
}
```
使 @uiw/react-markdown-preview 与 react-markdown 落入 vendor-md-editor（而非 vendor-react），vendor-react 的 modulepreload 不再包含 vendor-markdown。
</action>
  <verify>
    <automated>cd xingran-react-frontend && npm run build 2>&1 | grep -E "vendor-react|vendor-md-editor|vendor-markdown" | head -10</automated>
  </verify>
  <done>@uiw/react-markdown-preview 和 react-markdown 归入 vendor-md-editor；vendor-react modulepreload 无 vendor-markdown</done>
</task>

<task type="auto">
  <name>Task 3: echarts.ts 注册面补齐 + EChartsWrapper 改 core 入口</name>
  <files>
    xingran-react-frontend/src/lib/echarts.ts
    xingran-react-frontend/src/components/charts/EChartsWrapper.tsx
  </files>
  <action>
**echarts.ts 补注册：**
1. 从 `echarts/charts` 导入 `LineChart, BarChart, PieChart`
2. 在 `echarts.use([...])` 数组中追加这三个 chart 类型
3. 确认 CustomChart 保留（MAC 轨迹图需要）

**EChartsWrapper.tsx 改懒加载入口：**
1. 将 `lazy(() => import("echarts-for-react/esm/index"))` 改为 `lazy(() => import("echarts-for-react/esm/core").then(m => ({ default: m.default })))`
2. 渲染时 prop 改为 `echarts={echartsCore}`（从 `@/lib/echarts` 导入 default export）
3. 两个 import（echartsCore + echartsInstance）并入同一 lazy chunk 的 Promise.all 结构（已有则复用）
4. 纯类型导入 `import type { EChartsOption } from "echarts"` 保留（构建时擦除）
</action>
  <verify>
    <automated>cd xingran-react-frontend && npm run type-check && npm run build 2>&1 | tail -20</automated>
  </verify>
  <done>echarts.ts 注册 LineChart/BarChart/PieChart；EChartsWrapper 使用 esm/core + echarts prop；type-check + build 通过</done>
</task>

<task type="auto">
  <name>Task 4: NoticeForm.tsx 删除静态 CSS 导入</name>
  <files>xingran-react-frontend/src/pages/system/notice/components/NoticeForm.tsx</files>
  <action>
删除 `NoticeForm.tsx` 第 17 行的静态 CSS 导入：
```
import "@uiw/react-md-editor/markdown-editor.css";
```
该 CSS 文件已由 `src/components/markdown/MarkdownEditor.tsx`（懒加载侧）内部导入，随 JS lazy chunk 加载。
</action>
  <verify>
    <automated>cd xingran-react-frontend && npm run type-check && npm run lint 2>&1 | tail -10</automated>
  </verify>
  <done>NoticeForm.tsx 不再静态导入 markdown-editor.css；type-check + lint 通过</done>
</task>

<task type="auto">
  <name>Task 5: 构建验证 + size-limit 阈值收紧 + 记录体积表</name>
  <files>xingran-react-frontend/.size-limit.json</files>
  <action>
1. `npm run build`，记录 dist/assets 各 chunk 体积（ls -la dist/assets/*.js）
2. 更新 `.size-limit.json` 阈值：main 首屏 gzip 目标 ≤ 850KB（原来是 1000KB 或无限制）；保留 total 上限
3. 修复前后对比记录（修复前 vendor-react 2031KB / vendor-echarts 1136KB / vendor-three 911KB / vendor-markdown 372KB 全部首屏 modulepreload）
</action>
  <verify>
    <automated>cd xingran-react-frontend && npm run build 2>&1 && echo "--- dist assets ---" && ls -lh dist/assets/*.js | awk '{print $5, $9}'</automated>
  </verify>
  <done>main 首屏 gzip ≤ 850KB（实测验证）；size-limit.json 已更新；体积表记录在 PLAN-SUMMARY.md</done>
</task>

</tasks>

<verification>
cd xingran-react-frontend && npm run type-check && npm run lint && npm run build
</verification>

<success_criteria>
- npm run build 成功，无错误
- 首屏 JS gzip ≤ 850KB（目标 ~690KB）
- vendor-echarts 体积显著下降（不再含全量 echarts-for-react）
- vendor-react modulepreload 不含 vendor-markdown
- echarts 图表（line/bar/pie/area）在 dashboard 和 MAC 热力图正常渲染
</success_criteria>

<output>
创建 `.planning/quick/260911-m76-fix-frontend-perf-audit-findings/260911-m76-01-PLAN-SUMMARY.md`
</output>
