---
rebuild: true
rebuild_date: 2026-11-08
rebuild_by: Phase 140 / VAL-03
original_file_missing_on_disk: true
original_refs:
  - .planning/OPEN-ITEMS.md #A4（本次里程碑的临时输入清单，取数用）
  - .planning/REQUIREMENTS.md VAL-03
  - .planning/MILESTONES.md:110-117 / .planning/PROJECT.md:182（v1.21 Phase 57 收口段）
  - .planning/STATE.md:248
  - .planning/debug/ad-push-queue-drain-interval-not-hot-reloaded.md:172-174（Test 2 独立证据）
  - git 历史原件：.planning/milestones/v1.21-phases/57-ad-outbox-lite/57-HUMAN-UAT.md（提交 5f7cf0b6f 归档，d3063a09f 清理前）
status: 重建件 / 原始文件已不在磁盘
milestone: v1.21
phase: 57-ad-outbox-lite
---

# Phase 57 人工 UAT 清单（重建件 / 原始文件已不在磁盘）

> ⚠️ **本文件为 2026-11-08 Phase 140（VAL-03）重建件。** 原始 `57-HUMAN-UAT.md` 已不在磁盘；
> 依据 `.planning/OPEN-ITEMS.md` #A4（本次里程碑的临时输入清单，取数用）、
> `.planning/MILESTONES.md` / `.planning/PROJECT.md` 的 Phase 57 收口记载，以及 git 历史中归档的
> 原文件（提交 `5f7cf0b6f` 归档、`d3063a09f` 清理前）逐项重建，条目内容与状态不改。
>
> **口径**：所有条目统一标「待执行（不标已完成）」；本文件不写生产数据、不切换生产配置。
> 重建件只做组织，实际执行由现场运维同事 / PG 授权方按 `.planning/notes/261108-uat-writeback-policy.md`
> 的口径完成；执行完成后在 GSD 自动生成的产物或未来里程碑的 REQUIREMENTS / UAT 清单中登记证据并勾选，
> **未执行不得勾选**。回写与跟踪见 `.planning/notes/261108-human-verification-tracker.md`。

## 三项来源与现状（VAL-03）

| 条目 | 内容 | 状态（本跟踪口径） | 备注 |
|---|---|---|---|
| 57-UAT-01 | PG 真实环境验证 `ON CONFLICT` 部分索引合并路径（QUEUE-02） | 待执行（不标已完成） | 组织为可执行清单，见下 §1 |
| 57-UAT-02 | 进程重启不丢已入队行（QUEUE-01 / SC1 后半，kill -9 重启耐久） | 待执行（独立证据可核销，按口径登记清偿日期 + 证据） | 独立证据见 `debug/ad-push-queue-drain-interval-not-hot-reloaded.md:172-174`；见下 §2 |
| 57-UAT-03 | AD 端复活链路闭环（QUEUE-06 端到端，AD UAC 514 / 533） | 待执行（不标已完成） | 组织为可执行清单，见下 §3 |

> 说明：原始清单 §1/§2/§3 曾记录 2026-10-21 实机探针/时间线（`[pass]` 等）。按 VAL 共同约定
> 「不伪造闭环」与 REQUIREMENTS.md 第 13 行权威回收裁定，本重建件**不沿用原件的 pass 结论**，
> 统一以「待执行」登记；其中 57-UAT-02 依 A4 裁定可由独立证据核销，核销动作按回写口径
> 「执行完成后在 GSD 产物中登记证据并勾选」办理，并注明清偿日期 + 证据路径，本文件不预置勾选。

## §1 57-UAT-01 — PG 真实环境验证 ON CONFLICT 部分索引合并路径（QUEUE-02）

- [ ] **57-UAT-01**：连续两次编辑同一用户 → `ad_push_queue` 仅 1 行 `pending`（合并而非新增行）；
  确认未报 `42P10 there is no unique or exclusion constraint matching the ON CONFLICT specification`。
  - **预期**：以生产同款子句 `ON CONFLICT (entity_type, entity_id, action_type) WHERE status IN ('pending','failed') DO UPDATE`
    对部分唯一索引 `uq_ad_push_queue_active` 推断成功；两次入队后仅 1 行且同一 id、`status=pending`、
    `attempts=0`、`last_error/next_retry_at/failed_at` 全清；把该行打成 `failed/attempts=3/last_error=boom`
    后第三次入队仍合并为 1 行（事务内 ROLLBACK，无残留）。
  - **前置授权 / 环境**：生产 PG 只读或受控探针窗口；sqlite 侧 `TestPhase57_EnqueueUpdate_MergeOnSecondEdit`
    已提供等价回归保护，但 PG 冲突目标推断只能实机确认。
  - **已知噪声**：UI 首次实测出现 2 行 pushed 与排水器 30s 刻度有关（终态行退出唯一索引是设计行为，
    见 `TestPhase57_EnqueueUpdate_SyncsWithTerminalRow`），非合并失效；字面级 UI 双击复测待做
    （排水间隔设 300s + 重启后端，在同一窗口内连改两次）。
  - **执行人**：PG 授权方 / 现场运维同事（AI 不擅自写生产数据）。
  - **证据位**：探针 SQL 输出 + `ad_push_queue` 行快照（落 GSD 产物 / 未来里程碑 UAT 清单，见回写口径）。

## §2 57-UAT-02 — 进程重启不丢已入队行（QUEUE-01 / SC1 后半，kill -9 重启耐久）

- [ ] **57-UAT-02**：PG 部署后编辑用户入队 → `kill -9` → 重启 →
  `SELECT * FROM ad_push_queue WHERE status='pending'` 行仍在、`attempts=0`、字段零漂移。
  - **预期**：kill 与重启之间行保持 pending；重启后 attempts 归零、字段不漂移；持久化断言成立。
  - **独立证据（可核销）**：`debug/ad-push-queue-drain-interval-not-hot-reloaded.md:172-174`
    —— 「Test 2 的核心断言（kill -9 → 重启 → pending 行不丢）已经独立成立：行在 kill 与重启之间保持 pending、
    重启后 attempts=0、字段零漂移。12:15 tick 的失败是排水阶段的另一个缺陷（X），不回溯到持久化。」
    另据 `.planning/OPEN-ITEMS.md` #A4（本次里程碑的临时输入清单，取数用），该 12:15 tick 失败属排水阶段另一缺陷，不回溯持久化。
  - **核销口径**：按 `.planning/notes/261108-uat-writeback-policy.md`，执行完成后在 GSD 自动生成的产物 /
    未来里程碑 UAT 清单中登记证据并勾选，标注清偿日期 + 证据路径；**未执行不得勾选**，本件不预置勾选。
  - **执行人**：PG 授权方 / 现场运维同事。
  - **证据位**：`kill -9` 前后 `ad_push_queue` 行快照 + 重启日志 + 上述独立证据引用。

## §3 57-UAT-03 — AD 端复活链路闭环（QUEUE-06 端到端）

- [ ] **57-UAT-03**：sys 删除用户 → 排水器执行 `DisableUser`（UAC 514）→ AD 端账号禁用 →
  AD 域登录 bind 失败 `data 533`（账号已禁用），「已删用户经 AD 登录复活」被物理切断。
  - **预期**：删除路径同事务入队 `disable` 行；排水器推成 `pushed`（`attempts=0`、无 `last_error`）；
    AD 端 UAC 514 生效；用该账号在登录页做 AD 域登录预期失败 `data 533`。
  - **前置授权 / 环境**：生产窗口 + AD 账号可观测权限；用户裁决（2026-10-21 12:30）缺陷 Y
    （AD-sync 服务账号对同组 OU 无属性写权限，LDAP code 50）属环境策略，改用非管理员同组账号执行。
    多实例并发排水（本地 + 远端实例）为观测噪声，核对证据以 `disable` 行为准。
  - **执行人**：现场运维同事 / AD 授权方（AI 不擅自写生产数据、不擅自切换生产认证模式）。
  - **证据位**：`ad_push_queue` disable 行快照 + AD 账号 UAC 值 + 登录失败 data 533 截图/日志。

## Summary（重建口径）

| Metric | Count |
|--------|-------|
| total | 3 |
| 待执行（不标已完成） | 3 |
| 独立证据可核销（待按口径登记） | 1（57-UAT-02） |
| 已勾选 / 已完成（伪造） | 0 |

> 本重建件不改归档原件、不改 REQUIREMENTS/ROADMAP/STATE/PROJECT/MILESTONES 及本次里程碑的临时输入清单，
> 不写业务代码、不写生产数据；仅作为 Phase 140 的人工验证债组织交付物。
