# Phase 92: 暗色 mega-cluster 合并 - 验证矩阵

> 各 Plan 任务级验收命令、预期输出与失败条件。量化目标来源：ROADMAP Phase 92 成功标准。

---

## 验证锚点矩阵

| Plan | Task | 目标 | 验证命令 | 期望输出 | 失败条件 |
|---|---|---|---|---|---|
| **92-01** | 1.1 | AST 扫描脚本 + 结构化清单 | `node .planning/phases/92-mega-cluster/tools/scan-dark-rules.cjs --check` | 提取数 ≥ 区间规则总数，全部分类 | 漏项或未分类 |
| **92-01** | 1.2 | 例外表复核 + 扩表清单定稿 | 92-EQUIVALENCE.md 存在性检查 | exception ≤12 且逐条有理由 | 缺表或无理由 |
| **92-02** | 2.1 | Bridge 按 R6 扩表 | `cd ruoyi-react-frontend && npm run type-check` | exit 0 | TS2339 / token 键不存在 |
| **92-02** | 2.2 | R6 块级删除 | `scan-dark-rules.cjs --check-r6-cleared && css-contract-check && lint:ci` | R6 零残留 + ALL PASS | 残留或契约 FAIL |
| **92-03** | 3.1 | Bridge 按 R7 扩表 | `cd ruoyi-react-frontend && npm run type-check` | exit 0 | type-check 报错 |
| **92-03** | 3.2 | R7 块级删除（含尾段切割） | `scan-dark-rules.cjs --check-r7-cleared && css-contract-check` | 仅剩例外表登记项 | 残留超例外范围 |
| **92-03** | 3.3 | 散存量清零与移交登记 | `css-contract-check && lint:ci && test && check:gating && go build ./...` | 全绿 + 92-HANDOFF 登记 | 任一门禁失败 |
| **92-04** | 4.1 | 清除 R7 僵尸章节注释 | `cd ruoyi-react-frontend && node scripts/css-contract-check.js` | ALL PASS，M9 净减 ~200 行 | 契约报错或误删保护项 |
| **92-04** | 4.2 | 浅色区精简与 !important 削减 | `cd ruoyi-react-frontend && node scripts/css-contract-check.js` | M1 ≤ 480，M9 ≤ 3200 | M1 > 480 或契约断言 FAIL |
| **92-04** | 4.3 | 高频 Hex 变量化与预算锁定 | `cd ruoyi-react-frontend && node scripts/css-contract-check.js && npm run lint:ci` | M7 ≤ 60，C11 ≤ 480 锁定 | M7 > 60 或 lint 报错 |
| **92-05** | 5.1 | 复制改造 92 矩阵工具链 | 92-visual-evidence 存在性检查 | OK 92 matrix tools ready | 工具缺失或语法错误 |
| **92-05** | 5.2 | 运行 21 形态矩阵比对 | `92-DIFF-REPORT.md` 生成核验 | OK 92 diff report ready | 报告缺失或存在未登记异常 |
| **92-05** | 5.3 | 全系统回归与 92-VERIFICATION 归档 | `npm run lint:ci && npm run test && npm run check:gating && go build ./...` | 全部绿灯，92-VERIFICATION 归档 | 任一门禁失败 |

---

## 全局量化终态达标监控

| 指标 | 起点 | 92-03 当前值 | 92 终态目标 | 计划收口 Plan |
|---|---|---|---|---|
| **M1 全仓 `!important`** | 1381 | 671 | **≤480** | Plan 92-04 (Task 4.2) |
| **M2 index.css `!important`** | 1314 | ~630 | **≤420** | Plan 92-04 (Task 4.2) |
| **M4 暗色选择器命中** | 570 | ~130 | **≤150** | ✅ 已达标 |
| **M5 暗色分区** | ~25 | 11 | **≤12** | ✅ 已达标 |
| **M9 index.css 行数** | 5546 | 3440 | **≤3200** | Plan 92-04 (Task 4.1/4.2) |
| **M7 index.css hex** | 309 | 262 | **≤60** | Plan 92-04 (Task 4.3) |
| **成功标准 5 矩阵验收** | - | 待采样 | **差异零未登记** | Plan 92-05 (Task 5.1/5.2) |
