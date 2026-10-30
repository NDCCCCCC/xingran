# Phase 91 计划索引：Select / InputNumber / Form / Table / Modal 域交还

> 波次执行拓扑与依赖关系索引。

---

## 波次拓扑（Wave Architecture）

```text
Wave 1 (表单输入域)
└── Plan 91-01: Select + InputNumber + Form 交还 token，清理手写边框与死选择器 (3 tasks, ready)
      ↓
Wave 2 (数据展示与容器域)
└── Plan 91-02: Table (R8) + Pagination + Tabs + Tag 样式交还 board token，消除硬编码 hex (2 tasks)
      ↓
Wave 3 (浮层导航域 + 契约收口)
└── Plan 91-03: Modal + Dropdown 样式交还，styles.module.css 全局选择器迁出，量化预算下调与全系统终验 (3 tasks)
```

---

## 计划明细表

| 计划 ID | 波次 | 依赖 | 自治执行 | 核心任务概要 | 关键修改文件 |
|---|:---:|---|:---:|---|---|
| **91-01** | Wave 1 | 无 | ✅ | 1.1 AntdThemeBridge 扩展表单输入域 token<br>1.2 AppSelect 组件透传与弃用属性防护<br>1.3 精简 index.css 中表单手写覆盖 | `AntdThemeBridge.tsx`<br>`AppSelect.tsx`<br>`index.css` |
| **91-02** | Wave 2 | 91-01 | ✅ | 2.1 AntdThemeBridge 重构 Table (R8) 并扩展 Pagination/Tabs/Tag<br>2.2 清理 index.css 中 Table 手写覆盖与硬编码 hex (#f3eee3) | `AntdThemeBridge.tsx`<br>`index.css` |
| **91-03** | Wave 3 | 91-02 | ✅ | 3.1 AntdThemeBridge 扩展 Modal / Dropdown 域 token<br>3.2 styles.module.css 违规全局选择器迁出与 index.css Modal 整理<br>3.3 量化指标更新与全系统回归验证 | `AntdThemeBridge.tsx`<br>`styles.module.css`<br>`index.css` |
