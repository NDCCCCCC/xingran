# Phase 92 计划索引：暗色 mega-cluster 合并

## 波次拓扑（Wave Architecture）

```text
Wave 1 (等价性扫描)
└── Plan 92-01: AST 扫描脚本 + 三类分类清单 + 例外登记表 + Bridge 扩表草案 (2 tasks, ready)
      ↓
Wave 2 (R6 主体)
└── Plan 92-02: Bridge 扩表 + R6 区间块级删除 + 孤儿清理 (2 tasks)
      ↓
Wave 3 (R7 残块 + 收口)
└── Plan 92-03: Bridge 扩表 + R7 删除 + 量化终态锁定 + 92 口径矩阵机器验收 + 93 移交登记 (3 tasks)
```

## 计划明细表

| 计划 ID | 波次 | 依赖 | 自治执行 | 核心任务概要 | 关键修改文件 |
|---|---|---|---|---|---|
| **92-01** | Wave 1 | 无 | ✅ | 1.1 scan-dark-rules.cjs 扫描脚本与结构化清单<br>1.2 例外表复核与扩表清单定稿 | `tools/scan-dark-rules.cjs`<br>`92-EQUIVALENCE.md` |
| **92-02** | Wave 2 | 92-01 | ✅ | 2.1 Bridge 按 R6 扩表（board 化）<br>2.2 R6 区间块级删除与孤儿清理 | `AntdThemeBridge.tsx`<br>`index.css` |
| **92-03** | Wave 3 | 92-02 | ✅ | 3.1 Bridge 按 R7 扩表<br>3.2 R7 删除（细分章节+尾段切割）<br>3.3 量化终态锁定 + 矩阵验收 + 移交登记 | `AntdThemeBridge.tsx`<br>`index.css`<br>`css-contract-check.js`<br>`92-visual-evidence/` |
