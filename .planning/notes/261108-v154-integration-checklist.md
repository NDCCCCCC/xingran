# v1.54 W1 集成验收清单（Lead 用）

> 4 个 executor 并行写同一工作区、均**不执行 git 写操作**；Lead 在全部 COMPLETE 后统一集成。
> 生成日期 2026-11-09。基线（实现前实测全绿）：`go build ./...` ✅ / `go test ./...` ✅ / `tsc --noEmit` ✅。

## A. 收齐产物（4 个 executor 全部返回后才能开始）
- [ ] exec-142 → `## PHASE 142 COMPLETE` + `.planning/phases/142-disc-01-02/142-SUMMARY.md`
- [ ] exec-143 → `## PHASE 143 COMPLETE` + `.planning/phases/143-port-01-02/143-SUMMARY.md`
- [ ] exec-144-back → `## PHASE 144 BACKEND COMPLETE` + `.planning/phases/144-alrt-01-02-03/144-01-backend-SUMMARY.md`
- [ ] exec-144-front → `## PHASE 144 FRONTEND COMPLETE` + `.planning/phases/144-alrt-01-02-03/144-02-frontend-SUMMARY.md`
- [ ] 任一 BLOCKED → 先按报告的阻塞点处理，不强行集成

## B. 文件面冲突复核（最重要）
- [ ] `git status --short` 里**没有**非本里程碑预期的文件被改
- [ ] `pkg/permission/config.go`：只有 exec-144-back 改；确认含 5 个 `system:adDriftAlert*` 常量 + `/network/ports/write/physical` 补登（Phase 143 移交项）
- [ ] `src/constants/permissionCodes.ts`：只有 exec-144-front 改；5 个码与 `pkg/permission/config.go` / migration 238 **逐字一致**
- [ ] `internal/api/router.go`：只有 exec-144-back 改（新增 ad-drift-alert group）
- [ ] `internal/core/db/database.go`：**两处**调用点各有一行 `runAndRecord(238, ...)`（PG 分支 + sqlite 分支）
- [ ] `internal/core/db/migrations/registry.go`：**未**被误加 238（234+ 按现状不进 ActiveMigrations）
- [ ] `internal/services/device_discovery_service*.go` 与 `internal/services/portwrite/**` / `internal/services/addomain/**` 两两无交叉改动
- [ ] `src/lib/api.ts`（barrel）、`src/App.tsx`、`src/components/NotificationBell.tsx`、`src/types/network.ts`、`src/lib/api/networkApi.ts` **未被误改**（分别被 143/144 前端 PLAN 列为禁改或独占）

## C. 契约逐项复核
- [ ] `ImportDiscoveredDevices` 不再 `return len(deviceIDs), nil`；单事务、按 ip 去重（含软删恢复）、字段兜底、handler 缓存失效在 `operlog.Record` 之前
- [ ] 两处旧 stub 断言已改；新增 service 单测覆盖去重/回滚/幂等/空结果/missing task
- [ ] 型号级 `physicalProfile` 有回退链 + `Evidence=""` 未取证标注；Cisco/Maipu 显式 `vendorUnsupported`
- [ ] 批量物理参数：`BatchWriteRequest` 4 字段、入口 validator、`renderExtras`、`applyPhysicalAuditExtra` 修 `after_value`、sentinel 400；进度事件/三数组/operlog 未改
- [ ] 单条物理写 3 厂商×3 参数零回归
- [ ] `emit()` 走 `NoticeService` 发布链路 + `ad_sync_drift_alert` WS 广播；`scheduler.NoticeHub`/adapter 有 `BroadcastToAll`；有 `recover()`；软失败契约未破
- [ ] 5 个端点 + 权限分组；写端点 `Record`/`RecordWithBody` 在 `response.Success` 之前；webhook 配置/日志密钥脱敏有单测
- [ ] migration 238 幂等、双方言；9 行 sys_config seed；菜单/权限 seed
- [ ] 前端：页面在 `src/pages/system/ad-drift-alert/index.tsx`（glob 可发现）；**无第二条 `new WebSocket`**；5 个 API 全走 wrapped `post`；`useEffect` 依赖稳定
- [ ] `check-permission-codes.mjs` 双向校验通过（两端都落地后由 Lead 复跑）

## D. 全量回归闸门（Lead 统一跑，禁止 executor 跑全量）
- [ ] `go build ./...`
- [ ] `go test ./...`
- [ ] `cd xingran-react-frontend && npm run type-check`
- [ ] `cd xingran-react-frontend && npm run lint:ci`（或 `npm run lint` 按 package.json 实际脚本）
- [ ] `cd xingran-react-frontend && npx vitest run`（或 `npm run test`；记录用例总数，基线 592 文件 / 4128 用例）
- [ ] `node scripts/check-permission-codes.mjs`
- [ ] `node scripts/check-route-registry.mjs`（若存在，确认 `/physical` 补登后路由注册表一致）

## E. 收档
- [ ] 4 份 SUMMARY 齐全；按各 SUMMARY 的 Known Gaps 汇总成 v1.54 Known Gaps
- [ ] 更新 `.planning/STATE.md`：`status: v1.54 SHIPPED` / `progress.percent: 100` / `completed_phases: 3`；`## Deferred Items` 把 4 项代码缺口标为已清偿（写明 commit + 证据），验证类残余保持原登记
- [ ] 更新 `.planning/ROADMAP.md`：3 个 phase 勾选 `[x]` + completed 日期
- [ ] 更新 `.planning/REQUIREMENTS.md`：7 个 REQ 勾选 `[x]`，Traceability Status → Satisfied
- [ ] 更新 `.planning/MILESTONES.md`：新增 v1.54 条目（Phases 142-144 / 7 REQ / Known Gaps）
- [ ] 更新 `.planning/PROJECT.md`：v1.54 → SHIPPED & ARCHIVED
- [ ] **79 条人工验证跟踪表保持 0 勾选**；LOCKED 残余 6 项不改代码；FIX-01/04/09/13 验证类不伪造闭环
- [ ] 统一 `git add` + 原子提交（实现代码 + 规划产物分开或合并按 diff 复核结果决定）

## F. 提交前复核（CLAUDE.md）
- [ ] 展示 `git diff --stat` 与关键 diff 摘要
- [ ] 用户已授权本里程碑自动提交（2026-11-08 确认），无需再次逐次确认
