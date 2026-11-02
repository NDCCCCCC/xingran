# Phase 88-03 量化快照（删除第 1 批 D1-D17 后实测）

> 采样时间：2026-10-28 17:20（本地 sqlite UAT 环境 + :4000 dev server）
> 口径来源：ROADMAP.md §量化口径（M1-M16）

| 指标 | 口径 | 88-03 前 | 88-03 后（实跑） | PLAN 预估 | 备注 |
|---|---|---|---|---|---|
| **M1** | 全 `.css`/`.less` `!important` 计数 | 1435 | **1398** | ≈1395 | C11 ratchet 定值 1398（只许低不许高，预估偏低 3，取实测） |
| **M2** | `index.css` `!important` 计数 | 1359 | **1331** | ≈1328 | 同上偏差 3 |
| **M3** | `index.css` 含 `.ant-` 顶层规则 | 748 | 未清算 | — | Phase 90 范围 |
| **M4** | 暗色顶层规则（行首 `[data-color-mode="dark"]` grep 口径） | 506（含 8 拼写错） | **569**（拼写错 0） | ≈498 | 本 grep 口径含多选择器组的续行；正式清算归 Phase 92 |
| **M9** | `index.css` 行数 | 5632 | **5548** | ≈5580 | 实删 84 行（D1-D15 + 分支行） |
| **M14** | 死样式文件数 | 8 + 1（带 import） | **0**（C8 全绿） | 0 | 9 个文件全删，含 FloorPlanEditor.less + 其 import |

## 死规则 grep 复核（删后）

```
grep -c 'dept-tree-search'   src/index.css → 0
grep -c 'data-color_mode'    src/index.css → 0
grep -c 'ant-select-selector' src/index.css → 0（9 → 0）
grep -c 'user-form-input'    src/index.css → 3（K8/K9 活规则 + 命中保留）
```

## 视觉 diff 判定（唯一硬判据）

- 21 形态 × 明暗 × carrier/inner × 5 属性（bg/border/radius/shadow/btw）= **360 格**
- 删除前后逐格 diff：**0 差异**（`matrix-diff-result.txt`）
- fixture 挂载点：`.ant-app.css-var-*`（antd cssVar scope，88-01 启示）
- 注意：初版 fixture 手写了旧版 antd `.ant-select-selector` DOM，会误中已删死规则产生 40 格伪差；
  已按 antd 6 真实 DOM（`.ant-select-content`/`.ant-select-input`，真实页该 class 命中为 0）修正后重取基线复测。

---

# Phase 88-04 量化快照（antd 6 弃用清理 + C14 断言收口后实测）

> 采样时间：2026-10-28 18:20（本地 sqlite UAT 环境 + :4000 dev server）
> 前置：88-03 快照（CSS 删除批）之上叠加本批**行为/属性层**清理；CSS 未动 → M1-M15 全部沿用 88-03 值

| 指标 | 口径 | 88-04 前 | 88-04 后（实跑） | 备注 |
|---|---|---|---|---|
| **M16** | antd 弃用警告类数（浏览器 console `deprecated`） | **1**（`[antd: Modal] maskClosable is deprecated`，`/monitor/cache` 实测；PLAN 沿用 88-01 基线 2 类） | **0** | 三页（`/system/user`、`/monitor/cache`、`/network/credentials`）**清空 console + 强制刷新**实测 deprecated 行数 = **0**；验收口径 = 「无 deprecated 类警告」，**非**「零 Warning」（rc-select `onSearch` 提示在清理范围外） |
| M1 | 全 `.css`/`.less` `!important` 计数 | 1398 | **1398**（未动 CSS） | C11 ratchet 仍为 1398，`lint:ci` PASS |
| M2 | `index.css` `!important` 计数 | 1331 | **1331** | 未动 CSS |
| M9 | `index.css` 行数 | 5548 | **5548** | 未动 CSS |
| M14 | 死样式文件数 | 0 | **0** | C8 全绿 |

### M16 口径更正（88-05 / 88-06 收口后）

原基线「2 类（`maskClosable` / `onSearch`）」**不成立**——穷举 antd 6.6.4 全库弃用表（`node_modules/antd/es/**`，17 组件 36 条声明）后，实际有 **8 类**会触发 console 弃用警告：

| 类 | 弃用 prop → 新形态 | 处数 | antd 证据 |
|---|---|---|---|
| Modal / Drawer | `maskClosable` → `mask.closable` | 22（antd 侧） | `es/modal/Modal.js`、`es/drawer/Drawer.js:155` |
| Modal | `destroyOnClose` → `destroyOnHidden` | 6 | `es/modal/Modal.js` |
| Card | `bodyStyle` → `styles.body` | 4（全是 Card） | `es/card/Card.js` |
| Card | `bordered={false}` → `variant="borderless"` | 6 | `es/form/hooks/useVariants.js:15-18` |
| Drawer | `width` → `size` | 9 | `es/drawer/Drawer.js:83-103,155` |
| Progress | `width` → `size` / `trailColor` → `railColor` | 2 + 1 | `es/progress/progress.js:120` |
| Alert | `message` → `title` | 37 | `es/alert/Alert.js:83`（`title ?? message`） |
| Space | `direction` → `orientation` / `split` → `separator` | 17 + 1 | `es/space/index.js:94` |
| notification | `message` → `title` | 2 | `es/notification/useNotification.js:155` |
| Image | `preview.visible` → `open` / `preview.onVisibleChange` → `onOpenChange` | 1 + 1 | `es/image/hooks/usePreviewConfig.js:68` |
| Timeline | `items[].children` → `content` / `items[].dot` → `icon` / `<Timeline.Item>` → `items` 数组 | 6 + 2 + 4 | `es/timeline/Timeline.js:107,116` |

**合计 74 处弃用用法清零**（88-04 六类 + 88-05 Drawer.width + 88-06 其余）；浏览器 **26 页遍历 deprecated = 0**（迁移前基线 11 条——走 `git worktree` 检出 HEAD + 同构探针页复现）。

**M16 终态口径** = 「**全仓 antd 6 弃用警告 = 0**」，由 `css-contract-check.js` 的 **C14 / C14c / C14d / C14e / C14f / C14g** 七项数据驱动守护锁死（`lint:ci` 链尾执行，任何弃用 prop 回潮即 exit 1）。

**不在 M16 口径内**（非 deprecated 类，非本次引入）：rc-select `onSearch should work with showSearch`、`useForm is not connected to any Form element`、Divider `orientation is used for direction` usage 提示。
| 弃用 prop 静态计数 | `destroyOnClose` / `bodyStyle` / `headStyle` / `maskStyle` / `footerStyle` / `contentWrapperStyle` / `size="middle"` / Select 六类 | 6 / 4 / 0 / 0 / 0 / 0 / 17 / 0 | **全 0** | C14 断言守护（`node scripts/css-contract-check.js` exit 0，输出含 `C14 PASS`） |
| 冻结契约位 `maskClosable` | 我方组件对外 prop（声明/解构/透传/调用方） | 7 | **7（保留）** | v1.22 契约冻结；C14b 断言守护 |
| `size="default"` | 唯一处（Drawer） | 1 | **1（豁免登记）** | `drawer/Drawer.d.ts:6` 枚举 `["default","large"]` → Drawer 无 medium |

## 弃用 prop 迁移前后计数对照（速查）

| 弃用类 | before | after |
|---|---|---|
| `maskClosable`（antd 组件上） | 22 | **0** |
| `maskClosable`（我方冻结契约位 / 注释 / 测试） | 7 / 5 / 12 | 7 / 5 / 12（输入侧保留，**内部形态断言改 1 处**） |
| `Card bordered={false}` | 6 | **0**（Table 7 处保留） |
| `bodyStyle` | 4 | **0** |
| `destroyOnClose` | 6 | **0** |
| `size="middle"` | 17 | **0** |
| `size="default"` | 1 | 1（Drawer 豁免） |

## 第二轮矩阵视觉 diff（唯一硬判据）

- 21 形态 × 明暗 × carrier/inner × 5 属性（bg/border/radius/shadow/btw）= **360 格**
- before（88-03 基线）vs after（本批复采）：**0 差异**（`matrix-diff-result.txt`：`单元格 360，差异 0`）
- 采信自检：after 明暗内部差异 **43** 格（基线 before 同为 43）→ 暗色捕获真实生效，非恒等空跑
- fixture 挂载点：`.ant-app.css-var-_r_0_`（antd cssVar scope）
- 第二轮截图 12 张：`88-visual-evidence/r2-*-{light,dark}.png`

## 登记待后续（不属本 plan 锁定清单）

- **`Drawer width` → `size`**：antd `drawer/Drawer.js:155` 弃用表含 `['width','size']`；实测 `/network/mac`、`/ad-domain/computers` 仍报该弃用告警，全仓 **9 处 / 8 文件**。迁移规则：`width={N}`（数字）→ `size={N}` 等价（`Drawer.js:84-85` `isNumber` 直返），但 `height` 与 placement 相关 → 需逐处核对。建议立 VISFIX-05。
