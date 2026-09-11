# 前端性能审计修复 — PLAN 总览

**quick_id:** 260911-m76
**来源:** Vercel React Best Practices 全量审计（6 个并行审计 agent）+ 主会话逐条人工验证
**目标:** 修复所有审计发现问题，提升首屏性能、修复功能 bug、消除过度重渲染、并行化串行 waterfall、优化 CAD/3D 渲染
**执行模式:** 串行批次（每批次 type-check + lint 守护）；发现清单外问题记录到 deferred 节

---

## 执行顺序与依赖关系

| 批次 | Plan 文件 | 依赖 | 核心修复内容 | 验证方式 |
|------|-----------|------|-------------|---------|
| **W1** | `260911-m76-PLAN-build-bundle.md` | 无（独立） | 首屏 1.43MB→850KB；echarts 按需；vendor 分组修正 | `npm run build` + 体积表 |
| **W2** | `260911-m76-PLAN-bugfix-ws-polling.md` | 无（独立） | WS 链路修复；孤儿页删除；persist version；encryptionKeyStore 清理；jsonata 卸载 | `npm run type-check` + 用户 WS 验证 |
| **W3** | `260911-m76-PLAN-zustand-selectors.md` | 依赖 W1（3.1 涉及 MarkdownEditor.tsx） | dashboard 族 selector 拆分；布局壳 selector 拆分；useUserOptions 共享缓存 | `npm run type-check` + grep 无参调用 |
| **W4** | `260911-m76-PLAN-waterfall-parallel.md` | 无（独立） | 平面图保存并行化；duty 双刷新并行；VDI 分层并行；工单 currentRef | `npm run type-check` |
| **W5** | `260911-m76-PLAN-cad-3d-render.md` | 无（独立） | CAD 坐标精度；高频态 ref；memo 化；3D useFrame 收敛；Map 化查找 | `npm run type-check` + 用户 CAD 验证 |
| **W6** | `260911-m76-PLAN-js-misc.md` | 无（独立） | DeptTree 单遍；ports Set 优化；静态映射提升；湖北地图 tooltip | `npm run type-check` |

---

## 批次说明

### Wave 1（可并行执行，6 个 plan 全独立）

**build-bundle**（P0 必修）
- 首屏体积削减 ~50%（1.43MB→850KB gzip）
- vite.config.ts: 删除 echarts-for-react 特殊分支 + 加 runtime-helper + 修 @uiw/ 分组
- echarts.ts: 注册 LineChart/BarChart/PieChart
- EChartsWrapper.tsx: 改 esm/core 入口 + echarts prop
- NoticeForm.tsx: 删除静态 CSS 导入

**bugfix-ws-polling**（P0 必修）
- 通知 WS 链路修复（NotificationBell 实际建立连接）
- SyncMonitor 孤儿页 + useRealtimeUpdates 死代码删除
- dataFetcher wsConnections 泄漏修复
- 4 个 persist store 补 version: 1
- encryptionKeyStore finally 统一清理
- jsonata 卸载 + TabBar resize 删除 + requestIdleCallback 分片

**zustand-selectors**（P1）
- 依赖：build-bundle（MarkdownEditor.tsx 在批次 1 验证后稳定）
- dashboard 族 selector 拆分（cacheWidgetData 每次生成新 Map 导致集体重渲染）
- 布局壳/树根 selector 拆分
- useUserOptions 共享缓存 hook（替换 5 处手拉）

**waterfall-parallel**（P1）
- 平面图保存串行→并行（Promise.allSettled 分批 10）
- duty/VDI/info-points 串行→并行
- 工单 fetchList currentRef 模式

**cad-3d-render**（P1）
- CAD 坐标 snapCoord 精度
- lastMousePos state→ref
- cad-elements React.memo
- 3D useFrame 收敛即停
- CAD 查找 Map 化

**js-misc**（P2）
- DeptTree 双遍历改单遍
- ports 选中集 useMemo+Set
- 52 处静态映射提升模块级
- 湖北地图 tooltip 优化

---

## 验证策略

### 每批次完成验证（强制）
```bash
cd xingran-react-frontend
npm run type-check
npm run lint
```

### build-bundle 完成验证（强制）
```bash
npm run build
# 首屏 JS gzip ≤ 850KB
# vendor-echarts 体积显著下降
# modulepreload 无 vendor-markdown
```

### bugfix-ws-polling 用户验证（checkpoint:human-verify）
- 启动 dev server，登录
- DevTools > Network > Filter: WS 确认通知 WS 连接成功

### cad-3d-render 用户验证（checkpoint:human-verify）
- 打开楼层平面图编辑器
- 拖拽元素确认无亚像素抖动
- 快速移动鼠标确认流畅

### 全批次完成验证（强制）
```bash
npm run build && npm test
```

---

## 提交规范

每批次原子提交，commitlint conventional：
```
fix(frontend): build-bundle — reduce首屏JS从1.43MB降至850KB
fix(frontend): bugfix-ws-polling — 修复通知WS链路/删除孤儿页/补persist version
fix(frontend): zustand-selectors — 拆分dashboard族selector消除集体重渲染
fix(frontend): waterfall-parallel — 平面图保存/duty/VDI串行改并行
fix(frontend): cad-3d-render — CAD坐标精度/高频态ref/memo化/3D收敛
fix(frontend): js-misc — DeptTree单遍/ports优化/静态映射提升
```

**注意：** `internal/services/system/asset_columns_schema.json` 不要提交（主仓根有无关未提交改动）。

---

## deferred（发现清单外疑似问题）

- 发现清单外的疑似问题记录到此节，不在本 quick 顺手修复
- 格式：`file:line — 描述 — 建议处理方式`
