---
phase: 93
plan: 05
subsystem: frontend/backups-page
tags: [react, polling-hook, modal, vitest]
key-files:
  created:
    - xingran-react-frontend/src/pages/network/backups/hooks/useRestoreTask.ts
    - xingran-react-frontend/src/pages/network/backups/hooks/useRestoreTask.test.tsx
  modified:
    - xingran-react-frontend/src/pages/network/backups/types.ts
    - xingran-react-frontend/src/pages/network/backups/hooks/index.ts
    - xingran-react-frontend/src/pages/network/backups/hooks/useBackupModals.ts
    - xingran-react-frontend/src/pages/network/backups/hooks/useBackupModals.test.tsx
    - xingran-react-frontend/src/pages/network/backups/index.tsx
metrics:
  tests_added: 7 (useRestoreTask 5 + useBackupModals restore 契约 2；页面全量 36 用例全绿)
  commits: 2
---

# Plan 93-05 Summary: 前端最小异步交互（恢复按钮 + 轮询 Modal）

## What Was Built

1. **types.ts**：`RestoreTaskStatus` 四态 + `ConfigRestoreTask` interface（字段以后端 model json tag 为准——`resultJson` 是 JSON 字符串而非嵌套对象，plan 原文 `result?` 据此校正）
2. **useRestoreTask**（D-16）：`useRestoreTask(taskId)` → `{ task, polling }`；立即首查 + 3s interval 续查，终态 `clearInterval` 自停；**派生态设计**——对外 task 经 `task.id === taskId` 过滤（taskId 置 null/切换自动失效，零清理 setState），effect 同步路径零 setState（react-hooks cascading-render lint 规则驱动此重构，初稿 setTask/setPolling 同步调用报 error）；单次查询失败仅 console.error 不中断轮询
3. **useBackupModals 异步化**（D-04 前端侧/D-17/D-19）：handleRestore 携带 `{ deviceId: selectedRestoreBackup.deviceId }`（**修复现状空 body 缺 deviceId 必 400 的缺陷**）→ 读 `result.data.taskId` → `restoreTaskId` state（Modal 切进度态，不立即关闭）；closeRestoreModal 复位 taskId
4. **index.tsx 恢复 Modal**：footer 条件化（进度态仅"关闭"）；进度 Card 四态——failed（Alert error + errorMessage + failedLine）/ success（Alert success + 已下发 x/y 行 + hashMatched==false 黄字提示）/ pending "排队等待中" / running "配置下发中"（Spin）；resultJson 经 useMemo JSON.parse
5. **版本列表抽屉恢复入口**：原直接 post 空 body → Modal.confirm + openRestoreModal(backup) + confirmRestore() 复用同一异步进度流程（顺带修复同款缺陷）

## Commits

| Commit | Description |
|--------|-------------|
| a4bfc71 | feat(93): add useRestoreTask polling hook with types and tests (D-16) |
| f14be4f | feat(93): async restore interaction in backup modal (D-04/17/19) |

## Deviations

- useRestoreTask 初稿在 effect 同步路径调用 setState（null 分支清理 + polling 开关），被 `react-hooks` cascading-render 规则判 error——重构为派生态（`activeTask` 按 id 过滤 + `polling` 派生自终态判断），语义不变且更符合 React 惯例。
- 抽屉恢复入口的空 body 缺陷同 plan 意图（D-04 前端侧"不提供跨设备选择"）一并修复，超出 must_haves 文件清单但属同一缺陷族。

## Self-Check: PASSED

- `npm run type-check` exit 0
- `npm run lint` → 0 error（1390 warnings 均为存量）
- `npm run test -- --run src/pages/network/backups` → **36 passed (7 files)**
