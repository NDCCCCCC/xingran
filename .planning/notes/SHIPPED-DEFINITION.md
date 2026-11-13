---
doc_type: 里程碑收档判定定义（SHIPPED Definition / Definition of Done）
created: 2026-11-12
created_by: 状态口径统一（v1.56 候选）
status: 口径生效中 / 纯文档 / 零勾选 / 零代码
applies_to: 当前及今后所有里程碑（含 v1.56 候选）
related:
  - .planning/STATE.md（当前状态段引用本文件）
  - .planning/notes/261108-human-verification-tracker.md（79 条人工验证跟踪表）
  - .planning/notes/UAT-EXECUTION-PLAN.md（三桶排期与解锁路径）
  - .planning/notes/261108-uat-writeback-policy.md（回写口径与证据模板）
---

# SHIPPED 判定定义

> 本文件回答一个问题：**一个里程碑在什么条件下才允许被标成 `SHIPPED`**。
> 用于纠正历史惯例里「代码闸门全绿 = SHIPPED」的提前收档：代码绿只证明「能编译、能跑测试」，
> 不证明「用户拿得到价值、真实环境验过、对外文档同步」。

## 1. 三要素（AND，缺一不可）

**SHIPPED = 代码绿（build/test/lint/门禁全过）+ 关键 UAT 通过 + 对外文档同步**

| # | 要素 | 可核对的判据 | 证据形态 |
|---|---|---|---|
| ① | **代码绿** | `go build ./...` / `go test ./...` / `tsc` / `lint:ci`（含 forbidigo、check-route-registry、check-permission-codes 等门禁）/ 全量 vitest **全过**；无「报错即预期」的假绿、无静默降级 | CI / 本地闸门日志、命令输出、commit hash |
| ② | **关键 UAT 通过** | 本里程碑承诺的关键用户场景在**真实环境**（生产 / 预发 / 现场真机 / 真实 PG）由人工执行人按判据验证通过；证据落盘并含 `清偿日期 + 执行人 + 证据位 + 判定` | `evidence/<条目ID>-<YYYYMMDD>.md`、截图索引、真机回显、PG 查询输出、DBA 回传记录 |
| ③ | **对外文档同步** | `CHANGELOG.md`、README / 手册、`docs/`、对外发布说明与本里程碑实际交付一致；Known Gaps / deferred 如实登记，不隐瞒 | 文档 diff、发布说明链接 |

三要素全部满足 → 才可在 `STATE.md` / `MILESTONES.md` / `ROADMAP.md` / `CHANGELOG.md` 标 `SHIPPED` 并跑收档流程。

## 2. 无 UAT 证据时的唯一允许状态

> **无 UAT 证据只允许标 `code-complete·verification-pending`。**

- `code-complete`：代码候选已落地、要素 ① 已满足（或已按门禁自证）。
- `verification-pending`：要素 ② 缺位 —— 关键 UAT 未执行 / 证据未落盘 / 前置授权（生产 PG 写授权、现场真机、DBA 回传、发布窗口）未具备。
- **禁止**：把「已排期」「待执行」「史实 RESOLVED」「独立证据可核销」折算成「已验收」；禁止为凑闭环率预置 `[x]`；禁止用本地 fixture / 模拟输出冒充真实环境证据。
- 处于 `verification-pending` 的里程碑：不得标 `SHIPPED`、不得写「Awaiting next milestone 已完成」、不得跑 `/gsd-complete-milestone` 收档归档。

## 3. 状态词表（唯一口径）

| 状态词 | 含义 | 允许出现在 |
|---|---|---|
| `SHIPPED` | 三要素齐备，已收档 | 历史里程碑条目、CHANGELOG 已发布版本 |
| `code-complete·verification-pending` | 代码完成、UAT 证据缺失 | 当前候选里程碑 / 未收档里程碑 |
| `已排期·待执行` | 已登记 owner / 计划窗口 / 前置授权 / 证据格式，**尚未执行** | 79 条人工验证逐条状态（tracker） |
| `待执行（不标已完成）` | 历史旧词，等价于「已排期·待执行」的前身，**本轮统一改为新词** | 仅历史原文引用 |

> 纪律：**79 条必须保持 0 勾选、0 已完成**。本轮把 tracker 状态从「待执行」统一为「已排期·待执行」，
> 只是补齐排期字段与状态措辞，**不是执行、不是验收、不是勾选**。

## 4. 今后每个里程碑的准入线（强制）

> **今后每个里程碑至少含 1 条用户可感知新价值 + 1 条真实环境验收证据。**

- **1 条用户可感知新价值**：交付物必须让最终用户（运维 / DBA / 管理员 / 发布负责人）能感知到变化 —— 新功能、新页面、新的可操作入口、可观测性提升、性能或安全体感改善；纯重构 / 纯文档 / 纯内部清理不单独算「用户可感知新价值」，须与价值项同批交付。
- **1 条真实环境验收证据**：至少 1 条关键 UAT 在真实环境执行并落盘（生产 / 预发 / 现场真机 / 真实 PG / DBA 回传），满足要素 ②；该证据须能追溯到具体条目 ID、执行人、日期、判定。
- 立项时（`/gsd-new-milestone` / 阶段计划）就要把这两条写进 REQUIREMENTS / 验收标准；**缺任一条不得进入收档判定**。
- 若受环境约束（如当前无生产 PG 写授权、无真机）确实拿不到证据：按 `code-complete·verification-pending` 挂账，明确列出解锁路径（见 `.planning/notes/UAT-EXECUTION-PLAN.md`），**不伪造、不提前 SHIPPED**。

## 5. 当前适用结论（v1.56 候选）

- 候选期内代码已落地（`CHANGELOG.md` `## [Unreleased] — v1.56 候选`：数据库全量备份与恢复、系统设置页收敛与低密度页面 Tab 化、菜单与权限迁移 241/242/243 + 补强 248/249、B 类 quick），要素 ① 按各 quick / PR 的闸门记录自证。
- 要素 ② **未满足**：79 条人工验证全部「已排期·待执行」，**0 勾选 / 0 已完成**；无真实环境 UAT 证据。
- 要素 ③ 部分满足（CHANGELOG 已写），但口径以本文件与状态文件统一后才算同步完成。
- **故 v1.56 候选当前状态 = `code-complete·verification-pending`**，不得标 SHIPPED。
- 本环境无生产 PG 写授权、无真机，只能排期不能执行；解锁后按 `.planning/notes/261108-uat-writeback-policy.md` 回写 GSD 产物，再重新判定。

## 6. 与既有口径的关系

- 不修改 `.planning/notes/261108-uat-writeback-policy.md` 的最高纪律（未执行不得勾选 / AI 不代执行 / 不写生产数据 / 不切生产认证模式），本文件是其**收档层**补充。
- 不修改 5 份重建 UAT 清单 / 133-UAT 跟踪副本 / `.planning/milestones/` 归档原件。
- 历史里程碑（v1.55 及更早）的 `SHIPPED` 快照为**当时口径下的收档记录**，按 `MILESTONES.md` 顶部说明仅供追溯；本定义**不追溯改写**历史条目，只约束当前与今后。
