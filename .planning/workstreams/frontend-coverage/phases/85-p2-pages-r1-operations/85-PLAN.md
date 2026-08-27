---
phase: 85-p2-pages-r1-operations
plan: 00
type: execute
wave: 0
depends_on: []
files_modified:
  - xingran-react-frontend/src/pages/operations/**/__tests__/*.test.tsx
  - .coverage-fe-floors
  - .planning/frontend-coverage-baseline.md
autonomous: true
requirements:
  - PAGES-01
  - QUAL-01
must_haves:
  truths:
    - "[PAGES-01] pages/operations 76 文件 3611 stmts 语句覆盖率拉升至目标线,子目录 floor 逐个登记"
    - "[QUAL-01] 1005 存量测试不回归,gate 0 FAIL"
  artifacts:
    - path: xingran-react-frontend/src/pages/operations/**/__tests__/
      provides: operations 页面 family 测试
    - path: .coverage-fe-floors
      provides: 子目录 floor 行
    - path: .planning/frontend-coverage-baseline.md
      provides: ratchet 行
key_links:
  - from: operations/__tests__/
    to: xingran-react-frontend/src/test/utils/renderWithProviders.tsx
    via: 84-00 harness 复用
---

# Phase 85 执行计划: P2 页面层 R1 — operations

**Goal**: pages/operations 76 文件 3611 stmts 语句覆盖率 ≥70%(PAGES-01)

## 子目录划分(实测文件数)

| 子目录 | 文件数 | Wave | 说明 |
|--------|--------|------|------|
| workstations | 14 | 1 | 最大子目录,工位管理主页面+hooks |
| floors | 13 | 1 | 楼层管理(含 FloorPlan 相关) |
| building-spaces-3d | 19 | 2 | 3D 楼宇(Three.js 重依赖,静态断言为主) |
| rpa | 12 | 2 | RPA 任务/执行器/worker |
| building-spaces | 7 | 3 | 楼宇空间 2D |
| buildings | 4 | 3 | 楼宇管理+geocoding hooks |
| assets/room-devices/server-rooms/dedicated-lines/info-points | 7 | 4 | 零散小目录 |

## 执行模式(Phase 84 验证有效的 inline 模式)

每个 wave:
1. 读子目录源码,识别 hooks/API 依赖
2. vi.mock 重依赖 hooks(useTableManager/useTableQuery/usePagination/useDict/useSidebarDeptFilter)
3. vi.mock @/lib/opsApi 端点工厂
4. renderWithProviders 渲染断言 + 纯函数直测
5. npm run test:coverage 实测 → 子目录 floor bump(−0.5pp 截断) → ratchet 行
6. 全量 vitest 0 回归验证

## 70% 目标说明

ROADMAP Phase 88 收口时全局 floor 收口 70.0。Phase 85 的 per-dir 目标 70%,但按 Phase 84 实证:重依赖页面(React Query/lazy/Three.js)以静态断言为主,实际命中率 15-50%。执行纪律: 能测尽测,floor 如实登记,ratchet 单调上升。

## Verification

1. 全量 vitest 0 失败(1005+ 存量)
2. gate 0 FAIL
3. ratchet 行追加
4. 子目录 floor 单调上升(只升不降)
