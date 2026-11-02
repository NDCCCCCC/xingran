# Phase 92 — 等价性清单（92-EQUIVALENCE）

> 由 `tools/scan-dark-rules.cjs` 生成（D-01），数据明细见 `92-EQUIVALENCE-DATA.json`。
> 扫描时间：2026-10-30，基于 HEAD `491e5273` 的 `index.css`（5546 行）。

---

## 扫描总览

| 分类 | 条数 | 处置 |
|---|---|---|
| **covered**（Bridge 已有 token 等价物） | 183 | Wave 2/3 直接删 |
| **extendable**（antd 有 token、Bridge 未配） | 267 | Wave 2/3 先扩表后删 |
| **exception**（antd 无法表达） | 49 | 保留，归入例外分区（聚合后 ≤12 块） |
| **合计** | **499** | 暗色区 `!important` 共 767 处 |

> 注：条数（499）大于 ROADMAP 估算（431），系 89-91 三轮新增残留与规则计数口径差异；以本扫描为准。

---

## 例外登记表（人工复核，D-02）

| # | 例外类 | 代表选择器 | 保留理由 |
|---|---|---|---|
| 1 | 浏览器滚动条伪元素 | `::-webkit-scrollbar*`、`.dept-tree-container::-webkit-scrollbar-thumb` | 浏览器原生伪元素，antd token 不可达 |
| 2 | 选中文本伪元素 | `::selection` | 同上 |
| 3 | 可访问性焦点环 | `:focus-visible` | WCAG 2.1 要求，Phase 92 保留并登记 |
| 4 | 全局元素底座 | `body`、`a`、`[disabled]`、裸 `[data-color-mode="dark"]` 变量定义块 | 非 antd 组件，属主题变量层（其中变量定义块可并入既有 dark 分区） |
| 5 | 自定义业务组件 | `.cron-selector*`、`.status-badge*`、`.modern-tag-*`、`.data-card*` | 项目自有类，非 antd 渲染树，只能手写 |
| 6 | Premium Glow 装饰 | glow 相关 | 纯装饰光晕，无对应 token |

**分区聚合判定**：49 条例外规则按 `index.css` 位置聚合后落在 ≤12 个连续分区内（以 Wave 2/3 删除完成后的复跑扫描为准，M5 ≤12 达标路径成立）。

---

## Bridge 扩表草案（Wave 2/3 输入）

extendable 267 条集中在以下组件族，扩表值一律取 `ruoyiBrand.dark` / `ruoyiBrand.cream` 键（零裸 hex）：

| 组件族 | 代表 token | 波次 |
|---|---|---|
| Alert / Transfer / Skeleton / Result / Descriptions / List | `colorBgContainer` / `colorBorder` / `colorText*`（组件级可回落全局，优先不扩） | W3 |
| Collapse / Slider / Rate / Spin / Empty / Steps | `colorFillSecondary` / `railBg` / `remainingColor` | W3 |
| Upload / Calendar / Badge / Progress / Carousel | `itemSelectedBg` / `textBorderColor` / `remainingColor` | W3 |
| QRCode / Comment / Notification / Typography | `colorBgTextContainer` / `colorText` | W3 |
| Segmented / FloatButton / AutoComplete / Switch / TimePicker | `itemSelectedBg` / `trackBg` | W3 |
| Anchor / Space / Divider / Tooltip / Popover | `colorSplit` / `colorBgElevated` | W3 |

**扩表纪律**：凡全局 `token.*` 已能覆盖的（colorBgContainer / colorBorder / colorText 三大件），**优先不扩 components**，让 antd 回落全局；只有组件级专属 token（如 `Switch.trackBg`）才进 `components`。这样扩表量收敛、M 指标压力最小。

## R6 / R7 删除行区间

明细见 `92-EQUIVALENCE-DATA.json`（每条含 startLine/endLine）。块级删除时以 JSON 的行区间并集为施工边界，穿插非暗色章节（`:2201` 全局样式 / `:2216` Tailwind 组件层 / `:2859` Focus Visible）严格避让。
