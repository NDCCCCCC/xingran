# Phase 92 计划索引：暗色 mega-cluster 合并

## 波次拓扑（Wave Architecture）

```text
Wave 1 (等价性扫描)
└── Plan 92-01: AST 扫描脚本 + 三类分类清单 + 例外登记表 + Bridge 扩表草案 (2 tasks) [已完成]
      ↓
Wave 2 (R6 主体)
└── Plan 92-02: Bridge 扩表 + R6 区间块级删除 + 孤儿清理 (2 tasks) [已完成]
      ↓
Wave 3 (R7 残块 + 散存量)
└── Plan 92-03: Bridge 扩表 + R7 删除 + 散存量清零 + 保护清单登记 (3 tasks) [已完成]
      ↓
Wave 4 (Gap Closure 1: 浅色区精简与 Hex 变量化)
└── Plan 92-04: 僵尸章节清理 (M9 ≤3200) + 浅色手写精简 (M1 ≤480) + 高频 Hex 变量化 (M7 ≤60) (3 tasks) [待执行]
      ↓
Wave 5 (Gap Closure 2: 矩阵机器验收与终验)
└── Plan 92-05: 92 矩阵采样与比对工具链 + 差异清单登记 + 92-VERIFICATION 终验归档 (3 tasks) [待执行]
```

## 计划明细表

| 计划 ID | 波次 | 依赖 | 类型 | 自治执行 | 核心任务概要 | 状态 |
|---|:---:|---|:---:|:---:|---|:---:|
| **92-01** | Wave 1 | 无 | standard | ✅ | 1.1 scan-dark-rules.cjs 扫描脚本与结构化清单<br>1.2 例外表复核与扩表清单定稿 | ✅ 已完成 |
| **92-02** | Wave 2 | 92-01 | standard | ✅ | 2.1 Bridge 按 R6 扩表（board 化）<br>2.2 R6 区间块级删除与孤儿清理 | ✅ 已完成 |
| **92-03** | Wave 3 | 92-02 | standard | ✅ | 3.1 Bridge 按 R7 扩表<br>3.2 R7 删除（细分章节+尾段切割）<br>3.3 散存量清零与 92-HANDOFF 登记 | ✅ 已完成 |
| **92-04** | Wave 4 | 92-03 | gap_closure | ✅ | 4.1 清除 R7 僵尸空章节注释 (M9 ≤3200)<br>4.2 浅色区冗余覆盖精简 (M1 ≤480)<br>4.3 高频 Hex 变量化改写 (M7 ≤60) | 🟡 待执行 |
| **92-05** | Wave 5 | 92-04 | gap_closure | ✅ | 5.1 复制改造 92-visual-evidence 矩阵工具链<br>5.2 运行 21 形态矩阵比对并登记差异清单<br>5.3 全系统回归与 92-VERIFICATION 归档 | 🟡 待执行 |
