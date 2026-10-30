# Phase 92: 暗色 mega-cluster 合并 - Technical Research

> 日期：2026-10-30。行号基于 Phase 91 收口后 HEAD（`86f770b9`），`index.css` 共 5546 行。

---

## 一、量化基线（实测）

| 指标 | 实测值 | 终态目标 | 削减量 |
|---|---|---|---|
| M1 全仓 `!important` | 1381 | ≤480 | ~900 |
| M2 index.css `!important` | 1314 | ≤420 | ~894 |
| M4 index.css 暗色选择器命中行 | 570 | ≤150 | ~420 |
| M5 暗色分区（>30 行间隔切分） | ~25 块 | ≤12 | ~13 |
| M9 index.css 行数 | 5546 | ≤3200 | ~2346 |
| M7 index.css hex | 309（脚本口径 303） | ≤60 | ~249 |

---

## 二、R6/R7 残块精确边界（实测章节结构）

经 Phase 89/90/91 三轮删改后，原 ROADMAP 行号已漂移。当前实测边界：

| Cluster | 实测区间 | 内容 | 规模 |
|---|---|---|---|
| **R6 主体** | `:2225`（暗色模式专用全局样式章）→ `:2799` 前 | 组件暗色副本大块 | ~570 行 |
| **R7 细分章节** | `:2799`（Premium Glow）→ `:3564` | 30 个「XX 组件暗色模式」小节：Alert / Transfer / Skeleton / Result / Descriptions / List / Collapse / Slider / Rate / Spin / Empty / Steps / Upload / Calendar / Badge / Progress / Carousel / QRCode / Comment / Notification-Card / Typography / Segmented / FloatButton / AutoComplete / Switch / TimePicker / Anchor / Space / Divider / Tooltip / Popover | ~765 行 |
| **R7 大块尾段** | `:3564`（暗色模式组件增强）→ `:4434` 前 | 混合暗色增强大块 | ~870 行 |
| 不在删除范围 | `:4440` 起（Modern Tag / Fixed Action Button / Theme Component Overrides） | 明暗共用样式 | 保留 |

**非暗色穿插项**（夹在区间内、删除时须避开）：`:2106` 特殊状态 / `:2145` 侧边栏宽度 / `:2176` Tailwind 主题配置 / `:2201` 全局样式 / `:2216` Tailwind 组件层 / `:2859` Focus Visible 可访问性（无障碍规则，保留）。

---

## 三、删除前提核验（Phase 91 已铺平）

`AntdThemeBridge.tsx` components 表现已覆盖：Button / Table（board 化）/ Input / InputNumber / Select（board 化）/ Form / Menu / Tabs / Tag / Card / Modal（board 化）/ Pagination。
**删除顺序纪律（D-02）**：Wave 2/3 每个删除块必须「先扩 token → 同 commit 删块」，禁止先删后补。

---

## 四、token 无法表达、预计进例外表的候选（≤12 块）

1. Premium Glow Effects（`:2799`，纯装饰光晕，无 antd token）；
2. Scrollbar 滚动条（`:2824`，浏览器原生伪元素）；
3. Selection 选中文本（`:2846`，浏览器原生伪元素）；
4. ECharts 图表暗色适配（图表 canvas 非 DOM token 可达）；
5. Three.js / 3D 场景暗色适配（WebGL）；
6. 登录页插画区（`login.css` 域）；
7. Focus Visible 可访问性环（`:2859`，WCAG 要求，保留并登记）；
8. 其余以 Wave 1 AST 扫描分类结果为准，例外表逐条补理由。

---

## 五、工具链与验证资产

- **矩阵脚本**：复用 `.planning/phases/90-input-variant/90-visual-evidence/`（matrix-fixture / matrix-sample / matrix-diff-90.cjs / p90-run.cjs）改造为 92 口径（`92-visual-evidence/`），暗色 21 形态 computed-style 断言。
- **等价性扫描脚本**：新建 `.planning/phases/92-mega-cluster/tools/scan-dark-rules.cjs`：剥离注释 → 提取 `[data-color-mode="dark"]` 规则 → 输出（行号/选择器/属性/值）JSON → 与 Bridge components 表键名匹配分类 → 生成 `92-EQUIVALENCE.md` 三类清单。
- **静态门禁**：`node scripts/css-contract-check.js`（C11/C13 ratchet 只降不升）+ `npm run lint:ci`（8 项）。

---

## 六、风险与防线

1. **删错相邻明色规则**：R7 尾段（`:3564-4434`）为混合块，必须以 AST 提取的暗色规则行号区间为准做块边界切割，禁止按视觉估算删。
2. **C13 per-file 新增阻断**：任何 token 扩表若把值写成 hex 到 `.css` 会触发阻断 —— 全部新值写进 `colors.ts`（TS 文件不计 M7/M8）。
3. **M9 收敛依赖连带删除**：删暗色块后残留的孤儿注释/空章节一并清除，否则行数不达标。
