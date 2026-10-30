---
phase: "91"
slug: "select-inputnumber-form-table-modal"
status: passed
nyquist_compliant: true
wave_0_complete: true
verified: "2026-10-30"
requirements_satisfied: 1
requirements_total: 1
---

# Phase 91 — VERIFICATION：Select / InputNumber / Form / Table / Modal 域交还

> 本 phase 分 3 个波次（Wave 1 ~ Wave 3，共 3 个 Plan）严格递进执行并原子提交。
> 证据来源：静态契约断言脚本 `scripts/css-contract-check.js`（C1~C16c 全部 PASS）+ `npm run lint:ci`（8 项门禁全通过）+ 全仓 TypeScript 类型检查 + `npm run check:gating` + git commit 链。
> **status = `passed`** —— 自动化与架构契约全部达成，无需人工审核介入。

---

## 一、ROADMAP Phase 91 五条成功标准验收对照

| # | 成功标准 | 判定 | 证据（文件名 + 提交 + 实测结果） |
|---|---|---|---|
| 1 | **Select 域手写规则交还**：手写边框交还 antd CSS 变量机制，清理 `.ant-select-selector` 等死选择器 | ✅ Satisfied | commit `e08c3ec8`（91-01）：`AntdThemeBridge.tsx` 为 Select 注入 `selectorBg`、`optionSelectedBg`、`optionActiveBg`（随模式取板）；`index.css` 清理冗余手写选项高亮规则，C7 / C12 契约全绿。 |
| 2 | **多组件域手写覆盖交还 token**：Table 域（承接 R8）thead/tbody 背景与 hover 态交还 token；Modal / InputNumber / Form 等交还对应 token | ✅ Satisfied | commit `451fe3b6`（91-02）+ `bf133844`（91-03）：<br>1. 解决 **R8** 缺陷：`Table` 配置彻底消除写死的浅色 `cream.*`，全部重构为按 `board` 明暗自适应，并消灭硬编码 hex `#f3eee3`；<br>2. `Pagination`、`InputNumber`、`Form`、`Modal`、`Dropdown` 全面注入 components token。 |
| 3 | **迁出 CSS Module 违例规则**：将 `styles.module.css` 内 6 条 antd 全局选择器迁出，满足 C10 契约 | ✅ Satisfied | commit `bf133844`（91-03）：`building-spaces/components/styles.module.css:114-142` 的 6 处违规选择器彻底删除，就近迁入 `src/index.css` 的 Modal 章节（使用 `.building-space-modal.ant-modal` 命名空间），全仓 `.module.css` 零全局类名污染。 |
| 4 | **全站 antd 6 弃用清单剩余项清零**：Select 域六类弃用 prop 清零，C14 全表 PASS | ✅ Satisfied | `node scripts/css-contract-check.js` 实测 C14、C14c、C14d、C14e、C14f、C14g 全部 PASS；全站 15 个 antd 组件弃用 prop 零命中。 |
| 5 | **量化预算收缩**：C11 `!important` 预算上限下调，M7/M8 hex 数量净减少 | ✅ Satisfied | `scripts/css-contract-check.js` 中 `IMPORTANT_BUDGET` 从 1397 降至 1385（实测当前 1381）；M7 hex 降至 303；M8 hex 降至 437。 |

---

## 二、Requirements 需求闭环（1/1 Satisfied）

| REQ | 需求描述 | 落地文件与提交 | 状态 | 关键验证结果 |
|---|---|---|---|---|
| **STYLE-02 (承接 R8)** | 手写 antd 覆盖规则交还官方 token 体系，清理死选择器，下调量化预算 | `AntdThemeBridge.tsx` (`e08c3ec8`, `451fe3b6`, `bf133844`)<br>`index.css` (`e08c3ec8`, `451fe3b6`, `bf133844`)<br>`styles.module.css` (`bf133844`)<br>`css-contract-check.js` (`bf133844`) | ✅ Satisfied | Select、Table (R8)、InputNumber、Form、Modal、Dropdown 手写覆盖交还官方 token；`index.css` 硬编码 `#f3eee3` 清除；`styles.module.css` 违规选择器清零；C11 预算锁入 1385。 |

---

## 三、原子提交链条（3 个 commit）

```text
e08c3ec8 feat(91-01): Phase 91 Wave 1 表单输入域交还——AntdThemeBridge 扩展 Select/InputNumber/Form，精简 index.css 手写覆盖
451fe3b6 feat(91-02): Phase 91 Wave 2 数据展示域交还——Table (R8) 随 board 自适应，扩展 Pagination，消除 #f3eee3
bf133844 feat(91-03): Phase 91 Wave 3 浮层导航域交还与契约收口——Modal/Dropdown token扩展，styles.module.css 违规选择器迁出，下调 C11 !important 预算
```

---

## 四、全系统回归验证数据

1. **静态样式契约断言**：
   - `node scripts/css-contract-check.js`：**ALL PASS**
   - C11 !important：当前实测 **1381** / 预算 **1385**（较 Phase 90 终态净减少 10 处）
   - C13 hex 全量：M7 = 303 ≤ 310 / M8 = 437 ≤ 452 / M9 = 5547 ≤ 5558
   - C14 弃用 prop：全仓 0 命中
2. **前端代码规范与类型检查**：
   - `npm run type-check`：PASS（0 错误）
   - `npm run check:gating`：PASS（已门控 25 + 豁免 51 = 76）
   - `npm run lint:ci`：PASS（ESLint 0 错误 0 警告，全部 8 项门禁全绿）
3. **后端构建检查**：
   - `go build ./...`：PASS（0 错误）
