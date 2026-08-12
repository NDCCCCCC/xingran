---
slug: vendor-chunk-cyclic-deps
status: resolved
trigger: 打包部署到服务器运行时报错：vendor-commons-DTGzsS6G.js:1 Uncaught TypeError: Cannot read properties of undefined (reading 'createContext')；后续每次"修复"后错误换位置：useLayoutEffect undefined，且每次 chunk 文件名/位置都变
created: 2026-06-17T12:30:00.000Z
updated: 2026-06-17T13:50:00.000Z
---

# Debug Session: vendor-chunk-cyclic-deps

## Symptoms

- **Expected**: 前端生产构建部署到服务器后正常加载运行。
- **Actual**: 浏览器打开即白屏崩溃，控制台报 `Cannot read properties of undefined (reading 'createContext')` / `(reading 'useLayoutEffect')`。
- **Error**:
  ```
  vendor-commons-DTGzsS6G.js:1 Uncaught TypeError: Cannot read properties of undefined (reading 'createContext')
      at vendor-commons-DTGzsS6G.js:1:31027
  # 之后每次"修复"错误换位置：
  vendor-commons-DDQCSdGd.js:1 ... (reading 'createContext')   at :1:31586
  vendor-commons-CfY01mYx.js:1 ... (reading 'useLayoutEffect') at :1:34427
  ```
- **关键特征**: 错误**每次换一个 chunk 文件名和位置**——典型的跨 chunk 引用环症状。
- **Timeline**: 历史上已多次"修复"（合并 vendor-antd、加 @tanstack/react-query、加 @uiw/react-baidu-map、加 zustand/react-grid-layout 等），每次都暂缓但换位置复发。本次为根因级修复。

## Root Cause（已确认）

**manualChunks 按包名强行切割，与真实模块依赖图冲突 → Rollup 把共享模块 hoist 后形成 chunk 间双向引用环 → ESM 求值时 React 绑定未就绪。**

报错必须满足：React 命名空间在某模块顶层被访问时仍为 `undefined`。这只在「使用 React 的包」与「React 本身」被拆进成环的不同 chunk 时发生。

dump 真实 chunk 依赖图（临时 generateBundle 插件 + 扫各 chunk 顶部 `import{...}from"./vendor-X"`）发现三类成环桥接：

| 桥接类型 | 具体包 | 成环 |
|---|---|---|
| **重复 React 实例** | `react-baidu-map@1.3.5` 把 `react:0.14.3` 声明为硬依赖（非 peer），node_modules 嵌套第二份 React → 两实例 context 互不可见 | react 0.14.3 vs 19.2.3 |
| **React 组件误入纯逻辑 chunk** | `@uiw/react-markdown-preview`（用 React）被 Rollup 拆进 vendor-markdown | vendor-markdown ↔ vendor-react |
| **纯库依赖了默认进 vendor-react 的共享工具** | three 扩展(three-stdlib/troika-*/camera-controls/stats-gl/@monogrid/gainmap-js/meshline)依赖 `three` 却落进 vendor-react；markdown 解析库(unified/micromark)依赖 extend/is-plain-obj/parse5/hastscript/nth-check/github-slugger 等长尾工具 | vendor-three ↔ vendor-react；vendor-markdown ↔ vendor-react |

## Fix（最佳实践：依赖图传递闭包拆 chunk）

**`xingran-react-frontend/vite.config.ts`** — 配置加载时扫描 node_modules 的 package.json（deps + peerDeps），计算三组传递闭包，按依赖关系（而非包名）分组：

- **THREE_FAMILY**（向上闭包）：所有直接/间接依赖 `three` 的包 → vendor-three。自动覆盖 13 个 three 生态包（含所有扩展），无需手工维护清单。
- **MARKDOWN_FAMILY**（向下闭包）：markdown 解析种子(unified/remark/rehype/micromark/mdast/hast)及其**向下纯工具依赖**(共 135 个包)，排除 REACT/THREE family → vendor-markdown（自包含叶子）。
- **REACT_FAMILY**（默认兜底）：react/react-dom + 所有 React 消费者 → vendor-react。

manualChunks 规则顺序：md-editor → @uiw/* → THREE_FAMILY → MARKDOWN_FAMILY → echarts-for-react(vendor-react) → echarts/zrender → xlsx → 默认 vendor-react。

**`xingran-react-frontend/package.json`** — 删除未使用的遗留依赖 `react-baidu-map`（`npm uninstall`），消除嵌套的 React 0.14.3。

## Verification

最终干净构建 + 验证（`DUMP_CHUNKS` 调试插件已移除）：

```
✓ npm run build 退出码 0（built in 34s）
✓ chunk 依赖图无环（DAG）—— vendor-three/md-editor/echarts 只单向依赖 vendor-react，无双向边
✓ 单一 React 实例（19.2.3），无 React 0.14 古老特征
✓ createContext(104次)/useLayoutEffect(14次) 均在 vendor-react 内，React 完整可用
```

chunk 体积：vendor-react 2.8MB、vendor-echarts 1.1MB、vendor-markdown 1.0MB、vendor-three 894KB、vendor-xlsx 429KB、vendor-md-editor 54KB。vendor-react 从此前的 3.7MB 降到 2.8MB。

**结论**：在「单 React 实例 + 无环 DAG」两个条件下，该报错在数学上不可能发生。

## Lessons Learned

1. **不要按包名逐个枚举打补丁**（`if id.includes('xxx') return vendor-react'`）——每个新依赖都会暴露下一个桥，错误永远换位置。
2. **改 manualChunks 前先 dump 真实依赖图**：临时 generateBundle 插件导出 chunk→packages 映射，扫各 chunk 顶部 import 做环检测（找双向边）。
3. **用传递闭包自动归类**：依赖 three 的进 three chunk，markdown 纯工具进 markdown chunk，其余 React 相关进 vendor-react。
4. **拆出去的纯逻辑 chunk 必须是叶子**：把它依赖的共享工具也一起纳入（向下闭包），否则反向引用 vendor-react 成环。
5. **排查重复 React**：`find node_modules -path "*/react/package.json"`，任何非顶层/非预期的 react 副本都是隐患（react-baidu-map@1.3.5 的 react:0.14.3 硬依赖是典型案例）。

详见记忆 `[[vite-vendor-chunking]]`。

## Related Files

- `xingran-react-frontend/vite.config.ts`（computePackageFamilies + 三族闭包 + manualChunks）
- `xingran-react-frontend/package.json` / `package-lock.json`（react-baidu-map 已删）
- 旧失败笔记：`.planning/debug/vendor-commons-createcontext.md`、`vendor-commons-uselayouteffect.md`
