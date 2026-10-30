# Phase 93 计划索引：room-devices 拆分 + 视觉回归验收

## 波次拓扑（Wave Architecture）

```text
Wave 1 (SPLIT-01: 代码重构拆分)
└── Plan 93-01: room-devices 单体大文件解耦拆分至 index.tsx ≤350 行，子文件 ≤250 行 (3 tasks, ready)
      ↓
Wave 2 (STYLE-05: 视觉终验与里程碑收官)
└── Plan 93-02: STYLE-05 机器断言 + 14 页巡检报告 + 全系统双端门禁 + 93-VERIFICATION 归档 (2 tasks)
```

## 计划明细表

| 计划 ID | 波次 | 依赖 | 自治执行 | 核心任务概要 | 关键修改文件 |
|---|:---:|---|:---:|---|---|
| **93-01** | Wave 1 | 无 | ✅ | 1.1 抽离 constants.ts（类型/字典/文本解析）<br>1.2 抽离 columns.tsx 与 CardView 组件<br>1.3 抽离 useRoomDeviceActions 与主文件瘦身（≤350行） | `constants.ts`<br>`columns.tsx`<br>`RoomDeviceCardView.tsx`<br>`useRoomDeviceActions.ts`<br>`index.tsx` |
| **93-02** | Wave 2 | 93-01 | ✅ | 2.1 STYLE-05 computed-style 断言与 14 页巡检报告<br>2.2 全系统全量双端门禁终验与 93-VERIFICATION 归档 | `93-VISUAL-INSPECTION-REPORT.md`<br>`93-VERIFICATION.md` |
