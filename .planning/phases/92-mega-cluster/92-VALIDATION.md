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
| **92-03** | 3.3 | 量化达标 + 矩阵验收 + 移交登记 | `css-contract-check && lint:ci && test && check:gating && go build ./...` | 全绿 + C11 下调锁定 + 差异零未登记 | 任一门禁失败 |

---

## 全局量化目标监控

| 指标 | 起点（实测） | 终态目标 |
|---|---|---|
| M1 全仓 `!important` | 1381 | ≤480 |
| M2 index.css `!important` | 1314 | ≤420 |
| M4 暗色选择器命中 | 570 | ≤150 |
| M5 暗色分区 | ~25 | ≤12 |
| M9 index.css 行数 | 5546 | ≤3200 |
| M7 index.css hex | 309 | ≤60 |

## 移交 Phase 93 登记项

1. 14 页人工巡检截图（AD 登录墙）→ `92-HANDOFF.md`；
2. X-2 暗色描边对比度终验。
