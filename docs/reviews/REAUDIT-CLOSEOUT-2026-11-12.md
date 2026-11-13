# 复查整改集成验收（A9 / 2026-11-12）

> 对应 2026-11-12 产品复查发现的问题闭环。验收基线 `main @ 0cbd0738`，所有改动留在工作树、未提交。
> 本轮 A1~A6 由 DSH Harness 卡片执行，A7/A8 同通道但 A7 会话未结算，A9 由主会话亲自执行；定时汇报卡已停用并删除。

## 9 项验收结论

| # | 项目 | 结论 | 证据 |
|---|---|---|---|
| 1 | 全链 | ✅ PASS | `go build ./...` exit 0；`go test ./...` exit 0；`tsc --noEmit` exit 0；`npx prettier --check .` exit 0；`npm run lint:ci` exit 0（eslint 0 problems + 8 治理脚本全过）；`check-route-registry` OK 726 路由 / 327 写 / 148 deferred；`check-frontend-api-urls` 523 URL / 命中 523 / **WARN 0 / 豁免 0 / 不匹配 0**；`check-permission-codes` 双向零漂移。A7 新增 5 文件 15 用例定向 vitest 全绿。格式化前 `format:check` 曾报 4 个文件，已用 `prettier --write` 只修这 4 个并复验通过 |
| 2 | 文档漂移 | ✅ PASS | CHANGELOG 与手册 §4.5 均为 `system:backup:remove`、4 个 F 码、写端点 `export/restore/export/remove`；旧表述「删除→export / 3 个 F 型」0 命中；版本段时间倒序且含 v1.28 与 v1.30–v1.50 的「无独立发布版本」说明；PR #85~#89 共 10 处记录；手册含 SQLite=3 / 停服=1 / 覆盖=4 / 会话=3 / 全库敏感=1；手册版本行「v1.56 候选（Unreleased）」与 CHANGELOG 一致 |
| 3 | 产品层 | 🟡 PASS（有声明例外） | `docs/PRODUCT.md` 含一句话定位 / 三类角色 / 三个可量化指标 / 「不做什么」；README 技术栈 8 项与实际版本逐项一致；28 个 `docs/` 链接经精确解析**无死链**；RuoYi-Next 残留 27 文件全在 `.planning/` 历史归档（例外外 0），`docs/README.md:106` 已声明品牌例外；docs 根仍保留 **19 组同名双份**，理由：16 份被 CLAUDE.md / CHANGELOG / Go 注释 / 手册等白名单外文件引用，按「宁可少删不可误删」保留，`docs/README.md` 已逐一点名并声明子目录为权威版 —— 属**已登记、有清偿窗口**的残留，非静默遗留 |
| 4 | 版本 | ✅ PASS | `LICENSE` = MIT，Copyright 2025-2026 XingRan-Next；`vite.config.ts` 注入 `__APP_VERSION__`（2 处）并新增 `AboutCard`；`cmd/main.go` 保留 `-ldflags -X` 可注入；STATE `milestone: v1.56 / 进行中`，CHANGELOG 与手册均 `v1.56 候选`，四者口径一致 |
| 5 | 验收债 | ✅ PASS（严格 0 勾选） | tracker 合计行 `79 / 0 已勾选 / 0 已完成`；真实 `- [x]` = 0、`| ✓ |` = 0；94 行「已排期·待执行」覆盖 owner / W1~W4 / 前置授权 / 证据落盘格式；`UAT-EXECUTION-PLAN.md` 与 `SHIPPED-DEFINITION.md` 存在；`.planning/frontend-coverage-baseline.md` 与 `.coverage-fe-floors` 一致（`pages/ad-domain 33.4→36.2`，实测 36.73%−0.5pp，593 文件 / 4161 测试证据已登记） |
| 6 | 备份安全 | ✅ PASS | `internal/services/dbbackup/maintenance.go:34` 导出 `IsRestoring() bool`；`internal/api/restore_write_guard.go` 实现恢复期写入阻断（503、放行只读与任务轮询）；`config.go` 「按字面写入」明文注释 0 命中（凭据已加密，GetPlan 仍脱敏）；`RestoreModal.tsx` SQLite 前置提示 6 处、`BackupListTab.tsx` 整库敏感提示 4 处；恢复确认文案含覆盖/停写/重新登录 |
| 7 | 豁免 | 🟡 PASS（部分转台账） | URL 豁免 0 条、WARN 0 条（原 16 条工厂派生告警已通过删死函数清零）；`rpaApi.ts` 死代码已删；`scripts/EXEMPTIONS-POLICY.md`、`scripts/FIX-PROTOCOL.md`、`scripts/RBAC-EXCEPTIONS.md` 三份策略文档齐全；**权限豁免仍 104 条**，`check-permission-codes.mjs` 输出「104 码按存量豁免跳过 —— 后续 task-3 整改落地后须逐条清理」。按任务书允许「降到 0 或写入明确清偿台账」，A6 已为每条补 reason 与窗口，但**彻底清零仍需后续 phase**（见 FAIL 项） |
| 8 | e2e | ✅ PASS | `tests/e2e/*.spec.ts` 孤儿 0 个；`tests/e2e/README.md` 写明 CI 现状与可选 Playwright 接线条件；价值迁移为 5 个真实 Vitest 冒烟文件 / 15 用例，定向运行全绿（登录加密契约 6、日志清空 login 分支 1、周期工单 3、恢复确认 2、SQLite 常驻提示 3） |
| 9 | 无越界 | ✅ PASS | HEAD 仍 `0cbd0738`（无 commit / checkout / stash）；工作树 89 个改动可按 A1~A9 文件所有权归因；A9 只新增本报告；A7 会话仍在 running 但产出已完整且经主会话复验 |

## FAIL / 未关闭项（诚实声明）

| 项 | 状态 | 说明与后续 |
|---|---|---|
| **79 条人工验证** | `code-complete·verification-pending` | 需真机（华为 S8700 / 锐捷 RS8607E / H3C）、生产 PG 只读与写授权、DBA 回传。本环境无授权、无设备，**只能排期不能执行，严禁伪造闭环**。按 `UAT-EXECUTION-PLAN.md` 三桶解锁 |
| **104 条权限码存量豁免** | 已登记、未清零 | 每条有 reason 与清偿窗口，但 `check-permission-codes.mjs` 仍提示需后续 phase 逐条清理。彻底清零需改 `pkg/permission/config.go` / 菜单播种（超出本轮所有权边界） |
| **docs 根 19 组同名双份** | 已声明例外 | 16 份被白名单外文件引用，按 `docs/README.md` 的清偿窗口由各引用维护方迁移后再删；本轮不误删 |
| **A7 Harness 会话未结算** | 编排残留 | 卡片 `d1cbf347…` 仍 `running`（最后写入 22:06），但 floors / 台账 / 5 组冒烟 / e2e README 均已落地且经主会话全链复验。建议确认后手动 settle 该卡片，不留悬挂执行 |

## 全链命令与结果速记

```
go build ./...                              exit 0
go test ./...                               exit 0
npx tsc --noEmit -p tsconfig.app.json        exit 0
npx prettier --check .                       exit 0（修 4 文件后）
npm run lint:ci                              exit 0
node scripts/check-route-registry.mjs        exit 0
node scripts/check-frontend-api-urls.mjs    exit 0（523/523，WARN 0）
node scripts/check-permission-codes.mjs     exit 0（双向零漂移）
npx vitest run <5 个冒烟文件>                5 passed / 15 passed
```

## 元说明：编排与会话

本轮最初使用 `subagent_claude_code`，7 次全部因 `[claude-code:unrecognized_model] {"model":"glm-5.3-flash[1m]"}` 失败；切换为 DSH 任务板 Harness session（每张卡片一个可见持久会话），代价是产生了 9 个执行会话 + 汇报卡每次触发 1 个会话。19:05→20:14 曾因「异步完成不唤醒主会话 + `reuseSession` 汇报卡自锁」停摆。用户指出会话噪音后，已停用并删除两张定时汇报卡，A9 改由主会话直接执行，不再新建会话。

> 结论：**本次复查发现的可自动验证问题已全部闭环，全链门禁全绿**；剩余 3 类为需外部授权/后续 phase 的诚实挂账，1 类为编排残留，均已在上方逐条标注，不以「已完成」掩盖。
