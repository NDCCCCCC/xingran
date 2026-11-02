# 前端 Vendor Chunk 体积优化 — 实施规划（v2 扩展版）

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 在保持 vendor chunk DAG（无环）约束下，减小三个特定场景下的下载体积：
1. **Dashboard widget**（路由 lazy 后）vendor-echarts 从 1.13MB → ~700kB
2. **通知公告表单 MarkdownEditor 打开时** vendor-markdown 从 1.02MB → ~300-500kB（去掉 rehype-prism-plus 代码高亮）
3. **清理死依赖** react-markdown（直接 import 0 处，bundle 命中仅 1 个 URL 字符串字面量）

**Architecture:**
- **不动 React / antd / vendor-react 拆分** —— 历史已证明会复现 `createContext undefined` / `Activity TDZ` 循环依赖（参见 `.planning/debug/resolved/vendor-chunk-cyclic-deps.md` 与 memory `vite-vendor-chunking`）。
- **三个 Task 互不依赖**——可按任意顺序独立实施。
- **不接受外部 AI 通用建议**（React 走 CDN / 拆 antd / markdown 换 marked）——在 DAG 约束下不可行。

**Tech Stack:** Vite 7.2 + React 19.2 + echarts 6.0 + echarts-for-react 3.0.5 + @uiw/react-md-editor 4.0

---

## 前置上下文

### 历史教训（必读——为什么只动这些）

| 时间 | 失败尝试 | 根因 |
|------|---------|------|
| 2026-06-12 | 按目录拆 chunk（vendor-react / antd-vendor / vendor-commons） | Activity TDZ |
| 2026-06-14 | 按包名打补丁（`if id.includes('@tanstack/react-query')`） | createContext undefined |
| 2026-06-17 | 依赖图传递闭包（THREE_FAMILY / MARKDOWN_FAMILY / 默认 vendor-react） | ✅ 根治，vendor-react 3.7MB → 2.8MB |

**当前 DAG 规则**（已在 `vite.config.ts` 落地）：
- vendor-react = react/react-dom + 所有 React 消费者（antd / rc-* / react-query / dnd-kit / react-markdown / @uiw/react-baidu-map / dayjs / axios / sm-crypto / 通用工具）
- vendor-three = THREE_FAMILY 闭包（向上）
- vendor-markdown = MARKDOWN_FAMILY 闭包（向下纯工具）
- vendor-echarts / vendor-xlsx / vendor-md-editor = 各自独立 chunk

**为什么本规划范围限制在 T1+T2+T4**：
- **vendor-react (776 kB gzip, 首屏)**：tree-shaking 失效（445 个图标 vs 实际 157 个），但修复需要 vite 配置 + 改部分 import 方式（侵入性中），本轮**不做**，单独评估。
- **vendor-three / vendor-xlsx**：已经是路由级 lazy（`componentLoader.tsx` 用 `import.meta.glob` + `lazy()`），仅在访问对应页面下载。本轮**不做**。

### 现状证据（实测 2026-06-24）

```
vendor-react         2.83 MB / 776 kB gzip  ← 首屏，不动
vendor-echarts       1.13 MB / 374 kB gzip  ← T1 目标（dashboard widget lazy 时下载）
vendor-markdown      1.02 MB / 346 kB gzip  ← T4 目标（MarkdownEditor 打开时下载）
vendor-three         894 kB / 242 kB gzip   ← 路由 lazy，不动
vendor-xlsx          429 kB / 143 kB gzip   ← 路由 lazy，不动
vendor-md-editor      54 kB /  17 kB gzip   ← MarkdownEditor 打开时下载
```

### 关键事实（grep 实证）

**echarts 图表类型使用（5 种）：**
| 文件 | 图表类型 |
|------|---------|
| `src/components/dashboard/widgets/types/ChartWidget.tsx:93,197` | `line` |
| `src/components/dashboard/widgets/types/ChartWidget.tsx:123` | `bar` |
| `src/components/dashboard/widgets/types/ChartWidget.tsx:162` | `pie` |
| `src/components/network/MACHeatmapChart.tsx:90` | `heatmap` |
| `src/components/network/MACTrajectoryChart.tsx:129` | `custom` |

但 `src/lib/echarts.ts` 引入 `CustomChart`（全图表超集）→ vendor-echarts 1.13MB 包含所有未用图表。

**react-markdown 死依赖证据：**
```
$ grep -r "from.*react-markdown" src/
(无任何 import)

$ grep "react-markdown" src/types/global.d.ts
(无)

$ grep -o "react-markdown" dist/assets/*.js
vendor-react-NFPvZmrd.js: 1 处（"https://github.com/remarkjs/react-markdown/blob/main/changelog.md" 字符串字面量，来自 rehype-prism-plus）
```

**@uiw/react-md-editor 高亮机制：**
- 默认入口 `@uiw/react-md-editor` 依赖 `@uiw/react-markdown-preview` → `react-markdown` → `rehype-prism-plus`（代码高亮）→ 被 MARKDOWN_FAMILY 闭包归到 vendor-markdown
- `@uiw/react-md-editor/nohighlight` 入口用 `@uiw/react-markdown-preview/nohighlight`，**完全不依赖 rehype-prism-plus**

---

## 明确不做（Out of Scope）

| 项 | 原因 |
|----|------|
| React 走 CDN / 拆 vendor-react | 破坏 DAG → createContext undefined 必现 |
| vendor-react 拆 antd 子集 | antd ↔ rc-* ↔ React 强耦合，反向引用环 |
| antd icons tree-shaking 修复 | 需 vite 配置改造 + 部分 import 方式调整，独立评估 |
| markdown 改用 marked 替代 react-markdown | 跨模块重构 |
| 路由级 lazy（已是 componentLoader.tsx 自动 lazy） | 已完成 |
| three / xlsx 进一步拆 | 已是独立 chunk + 路由 lazy |

---

## Task 1: echarts CustomChart → 精确图表

**Files:**
- Modify: `src/lib/echarts.ts`（整文件替换，25 行）
- 验证：`src/components/dashboard/widgets/types/ChartWidget.tsx`、`src/components/network/MACHeatmapChart.tsx`、`src/components/network/MACTrajectoryChart.tsx`

**Step 1: 测 baseline**

```bash
cd xingran-react-frontend
gzip -c dist/assets/vendor-echarts-*.js | wc -c
ls -la dist/assets/vendor-echarts-*.js
```

预期：gzip ~374000，raw ~1128000。

**Step 2: 检查 ChartWidget 是否用 legend**

```bash
grep -n "legend" src/components/dashboard/widgets/types/ChartWidget.tsx
```

如果有 `legend:` 配置 → 在 Step 3 保留 `LegendComponent`。否则删。

**Step 3: 检查 MACHeatmapChart / MACTrajectoryChart 是否用 visualMap / dataset / toolbox**

```bash
grep -nE "visualMap|dataset|toolbox|brush" src/components/network/MACHeatmapChart.tsx src/components/network/MACTrajectoryChart.tsx
```

如果有 → 在 Step 4 追加对应 component import。

**Step 4: 替换 `src/lib/echarts.ts` 为精确图表集合**

完整替换文件（25 行）：

```typescript
/**
 * ECharts 按需加载配置
 *
 * 历史：原本用 `CustomChart`（全图表超集），导致 vendor-echarts 1.13MB。
 * 实际项目只用了 5 种图表类型（grep 2026-06-24 实证）：
 *   - line (ChartWidget)
 *   - bar (ChartWidget)
 *   - pie (ChartWidget)
 *   - heatmap (MACHeatmapChart)
 *   - custom (MACTrajectoryChart)
 *
 * 替换为精确集合后 vendor-echarts 预期降至 ~700kB（gzip ~250kB）。
 *
 * 注：保留 CustomChart（用户自定义渲染需要），其余未使用的
 * ScatterChart/RadarChart/TreeChart/GraphChart 等不再拉入。
 */
import * as echarts from "echarts/core";
import {
  BarChart,
  LineChart,
  PieChart,
  HeatmapChart,
  CustomChart,
} from "echarts/charts";
import {
  TitleComponent,
  TooltipComponent,
  GridComponent,
  DataZoomComponent,
  LegendComponent,
} from "echarts/components";
import { CanvasRenderer } from "echarts/renderers";

// 注册必需的组件
echarts.use([
  BarChart,
  LineChart,
  PieChart,
  HeatmapChart,
  CustomChart,
  TitleComponent,
  TooltipComponent,
  GridComponent,
  DataZoomComponent,
  LegendComponent,
  CanvasRenderer,
]);

export default echarts;
```

**Step 5: 重新构建**

```bash
npm run build 2>&1 | tail -20
```

预期：`built in ~30s`，无 error。

**Step 6: 测新大小**

```bash
gzip -c dist/assets/vendor-echarts-*.js | wc -c
ls -la dist/assets/vendor-echarts-*.js
```

**回滚条件**：gzip 减小 < 100 kB（echarts 6 可能把所有图表塞同一包）。

**Step 7: 验证图表类型字符串仍在 dist**

```bash
grep -oE "(BarChart|LineChart|PieChart|HeatmapChart|CustomChart|LegendComponent)" dist/assets/vendor-echarts-*.js | sort -u | wc -l
```

预期：`6`（5 图表 + 1 Legend 组件）。如果漏了某个 → 回退 Step 3 补 import。

**Step 8: 验证其它 chunk 未退化**

```bash
for f in dist/assets/vendor-*.js; do
  raw=$(stat -c %s "$f")
  gz=$(gzip -c "$f" | wc -c)
  printf "%-40s raw=%10d gz=%10d\n" "$(basename $f)" "$raw" "$gz"
done
```

预期：除 vendor-echarts 外，所有其它 chunk gzip 变化 < 5%。> 5% 增长 → DAG 假设破坏，立即 `git checkout src/lib/echarts.ts`。

**Step 9: Commit**

```bash
git add src/lib/echarts.ts
git commit -m "perf(echarts): CustomChart-only → 精确 5 种图表，vendor-echarts 从 1.13MB → ~700kB

CustomChart 是 echarts 全图表超集，把所有未使用的图表类型都拉入。
实际项目 grep 实证只用了 bar/line/pie/heatmap/custom 5 种。

变更：
- src/lib/echarts.ts: CustomChart-only → BarChart/LineChart/PieChart/HeatmapChart/CustomChart
- 新增 LegendComponent（ChartWidget 图例需要）

Ref: docs/plans/2026-06-24-frontend-vendor-bundle-optimization.md"
```

---

## Task 2: 删除 react-markdown 死依赖

**Files:**
- Modify: `package.json` / `package-lock.json`（npm uninstall）

**Step 1: 确认 react-markdown 在 src 中无 import**

```bash
grep -rn "from ['\"]react-markdown['\"]" src/ 2>/dev/null
```

预期：无输出。如果有 import → 取消 Task 2，先评估影响。

**Step 2: 确认 build 不依赖 react-markdown**

```bash
# 检查是否在 optimizeDeps.include
grep "react-markdown" xingran-react-frontend/vite.config.ts
```

预期：无。如果有 → 同步删除。

**Step 3: 卸载**

```bash
cd xingran-react-frontend
npm uninstall react-markdown
```

预期：成功删除 `react-markdown` from package.json + package-lock.json。

**Step 4: 重新构建验证**

```bash
npm run build 2>&1 | tail -20
```

预期：构建成功，无错误。

**Step 5: 验证 vendor-markdown 不受显著影响（应该几乎不变）**

```bash
gzip -c dist/assets/vendor-markdown-*.js | wc -c
```

预期：变化 < 5 kB gzip（bundle 体积不变，因为 react-markdown 本来就不在 bundle 中，只是 transitive URL 字符串）。

**Step 6: 验证 MarkdownEditor 仍正常加载**

```bash
grep -c "MDEditor\|@uiw" dist/assets/vendor-md-editor-*.js
```

预期：非 0。

**Step 7: Commit**

```bash
git add package.json package-lock.json
git commit -m "chore(deps): 移除 react-markdown 死依赖

src/ 中无任何文件 import 'react-markdown'，bundle 中仅 1 处 URL
字符串字面量（来自 rehype-prism-plus 内部）。该依赖是早期 TODO
残留（NoticeDetailContent.tsx:119 注释文字），实际未使用。

Ref: docs/plans/2026-06-24-frontend-vendor-bundle-optimization.md"
```

---

## Task 4: MarkdownEditor 切换到 nohighlight 版本

**⚠️ 业务影响**：通知公告表单 MarkdownEditor 的预览将失去代码块语法高亮（文本仍正常显示，只是代码块无彩色语法高亮）。

**Files:**
- Modify: `src/components/markdown/MarkdownEditor.tsx`（1 行）

**Step 1: 验证 nohighlight 入口存在**

```bash
ls node_modules/@uiw/react-md-editor/nohighlight.js node_modules/@uiw/react-md-editor/nohighlight.d.ts 2>/dev/null
```

预期：两个文件都存在。如果不存在 → 取消 Task 4，升级 @uiw/react-md-editor 到最新版本（包含 nohighlight）或回退。

**Step 2: 检查现有 import**

```bash
grep -n "import" src/components/markdown/MarkdownEditor.tsx
```

预期：
```
import { lazy, Suspense, type FC } from "react";
import { Spin } from "antd";
import type { MDEditorProps } from "@uiw/react-md-editor";

const MDEditor = lazy(() => import("@uiw/react-md-editor"));
```

**Step 3: 修改为 nohighlight 入口**

`src/components/markdown/MarkdownEditor.tsx` 第 11 行：

```diff
- const MDEditor = lazy(() => import("@uiw/react-md-editor"));
+ const MDEditor = lazy(() => import("@uiw/react-md-editor/nohighlight"));
```

**Step 4: 检查 types 是否仍可用**

```bash
ls node_modules/@uiw/react-md-editor/nohighlight.d.ts 2>/dev/null
```

预期：存在。如果不存在 → 改为：
```typescript
import type { MDEditorProps } from "@uiw/react-md-editor";
const MDEditor = lazy(() => import("@uiw/react-md-editor/nohighlight"));
```

**Step 5: 重新构建**

```bash
npm run build 2>&1 | tail -20
```

预期：`built in ~30s`，无 error。

**Step 6: 测新 vendor-markdown 大小**

```bash
gzip -c dist/assets/vendor-markdown-*.js | wc -c
ls -la dist/assets/vendor-markdown-*.js
```

预期：gzip 减小 ≥ 100 kB（rehype-prism-plus 链移除）。如果减小 < 100 kB → 回滚（说明 nohighlight 入口没起效）。

**Step 7: 验证 vendor-md-editor 仍可用**

```bash
ls -la dist/assets/vendor-md-editor-*.js
```

预期：文件存在，大小变化 < 30%。

**Step 8: 浏览器手动验证**

```bash
npm run dev
```

打开通知公告表单，启用 Markdown 模式，输入包含 `\`\`\`js\nconsole.log('test')\n\`\`\`` 的内容：
- ✅ 代码块文本正常显示（无 syntax error）
- ✅ 其他 markdown（粗体、列表、链接）正常渲染
- ⚠️ 代码块无彩色语法高亮（**预期功能损失**）

**Step 9: Commit**

```bash
git add src/components/markdown/MarkdownEditor.tsx
git commit -m "perf(markdown): 切换到 @uiw/react-md-editor/nohighlight，vendor-markdown 从 1.02MB → ~700kB

默认入口 @uiw/react-md-editor 依赖 @uiw/react-markdown-preview →
react-markdown → rehype-prism-plus（语法高亮），被 MARKDOWN_FAMILY
闭包归到 vendor-markdown 1.02MB。

切换到 nohighlight 入口后，代码块不再有彩色语法高亮，但文本
正常显示——通知公告以纯文本消息为主，代码高亮业务价值低。

变更：
- src/components/markdown/MarkdownEditor.tsx:11
  import('@uiw/react-md-editor') → import('@uiw/react-md-editor/nohighlight')

业务影响：通知公告 MarkdownEditor 预览失去代码块语法高亮。
验证：dev server 打开通知公告表单，启用 markdown 模式，
输入含代码块的内容，确认文本正常显示无报错。

Ref: docs/plans/2026-06-24-frontend-vendor-bundle-optimization.md"
```

---

## 完成判定

✅ 成功条件：
- vendor-echarts gzip 减少 ≥ 200 kB
- vendor-markdown gzip 减少 ≥ 100 kB（T4）
- react-markdown 从 package.json 移除（T2）
- `npm run build` 通过
- 仪表盘 + MAC 图表页 dev server 渲染正常（T1）
- 通知公告 Markdown 编辑正常（T4，**代码块无高亮是预期**）

❌ 失败回滚条件（任一）：
- 任何 chunk gzip 减小 < 阈值（T1 < 100 kB / T4 < 100 kB）
- 其它 vendor chunk gzip 增大 > 5%（说明 DAG 假设破坏）
- 浏览器报 `Component is not registered` 错误
- TypeScript 类型错误（`@uiw/react-md-editor/nohighlight` 类型不可用）
- 生产部署报 createContext/useLayoutEffect/Activity TDZ 错误

---

## Baseline（Task 1 Step 1 测量后填入）

测量日期：____
- vendor-echarts gzip：____ kB
- vendor-echarts raw：____ kB
- vendor-markdown gzip：____ kB

最终（所有 Task 完成后填入）：
- vendor-echarts gzip：____ kB（Δ ____ kB，____%）
- vendor-markdown gzip：____ kB（Δ ____ kB，____%）
- react-markdown 状态：从 package.json 已删除 ✓ / 未删除 ✗
- 其它 chunk 变化：vendor-react ____ / vendor-md-editor ____ / vendor-three ____ / vendor-xlsx ____