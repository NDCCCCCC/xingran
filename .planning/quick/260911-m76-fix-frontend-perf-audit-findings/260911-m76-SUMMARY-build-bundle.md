# 批次 1 SUMMARY — build-bundle（首屏体积削减）

**quick_id:** 260911-m76
**batch:** build-bundle
**commits:** `98562a1` (executor) + `222c58f` (主会话补 modulePreload)

## 真实收益（独立验证 dist 产物）

| 指标 | 基线 | 修复后 | 变化 |
|---|---|---|---|
| 首屏 modulepreload 数量 | 4 | **1** | -3 |
| 首屏 modulepreload 体积 | ~1.43MB gzip | **~646KB gzip** | **-55%** |
| 排除出首屏的 chunk | vendor-react | vendor-react（保留） | — |
| 迁为按需的 chunk | — | vendor-three (-246KB), vendor-echarts (-210KB), vendor-markdown (-116KB), vendor-xlsx (-143KB), vendor-md-editor + 3D/Excel 懒 chunk | — |

## commit 范围

1. **98562a1 perf(bundle): reduce first-screen JS to ~850KB gzip** (executor)
   - `vite.config.ts` — 改 manualChunks 注释 + 删除 echarts-for-react 特殊分支 + 加 markdown 归组 + 加 runtime-helper 规则
   - `src/lib/echarts.ts` — 注册 LineChart/BarChart/PieChart
   - `src/components/charts/EChartsWrapper.tsx` — 改 esm/core 入口 + 传 echartsCore
   - `src/components/markdown/MarkdownEditor.tsx` — CSS 移入懒加载侧
   - `src/pages/system/notice/components/NoticeForm.tsx` — 删除静态 CSS 导入
   - `src/components/charts/__tests__/EChartsWrapper.test.tsx` — mock 适配 esm/core
2. **222c58f perf(bundle): exclude three/echarts/markdown/xlsx from first-screen modulepreload** (主会话补)
   - `vite.config.ts` — 加 `build.modulePreload.resolveDependencies` 排除懒加载 chunk

## 关键发现（修订 executor 报告）

- executor 报告 "vendor-echarts 缩到 209KB（-167KB）"：实际是 611KB raw / 210KB gzip（**gzip 数字大致正确**，但描述方式容易误读为"chunk 本身变小"；实际 chunk 仍 611KB，只是首屏不再下载它）。
- executor 报告"首屏降到 1.05MB"：与实际首屏 modulepreload 状态不一致。**首屏真正降到 646KB** 的是补 commit 222c58f。
- executor 自认"runtime-helper 未生成独立 chunk"：正确，Vite 7 把 helper 内联进 entry；补救策略改为 modulePreload 排除，比强制独立 chunk 更稳。

## verify 通过项

- `npm run type-check`: PASS
- `npm run lint`: PASS (pre-existing 1378 warnings, 无新增)
- `npm run build`: PASS (27.73s)
- `npx vitest run src/components/charts`: PASS (7 tests)
- `dist/index.html` modulepreload 验证: 1 条（vendor-react 唯一）

## 用户复核建议

- dev 模式登录后，访问 dashboard 查看图表、Mac 历史时间线、3D 楼宇视图是否仍正常
- 访问通知公告表单检查 markdown 编辑器加载行为
- 用 DevTools Network 面板对比 modulepreload 列表（修复前 4 个 / 修复后 1 个）
