---
phase: 57-ad-outbox-lite
verified: 2026-10-16
updated: 2026-10-22
re_verified: 2026-10-17
status: passed
scope: 57-01 / 57-02 / 57-03 + F-01~F-06 + Gap 闭环（GAP-01/02/03）
independently_reproduced:
  - "go build ./... → exit 0"
  - "go test ./internal/services/system/ -run TestPhase57 -count=1 → 42 PASS / 0 FAIL（33 原有 + 4 GAP-01 + 5 GAP-02）"
  - "go test ./internal/api/v1/system/ ./internal/services/addomain/ ./internal/core/db/ -count=1 → 全 ok"
  - "8 组变异测试全数还原：F-01/02/03/04/05/06 + GAP-01（UpdateStatus 接线）+ GAP-02（profile 接线），守护用例均真正转红"
  - "git diff 8246173a..HEAD -- internal/api/v1/system/user_handler.go → 空（D-03 红线守住）"
  - "git diff ee6fe937~1..HEAD -- internal/core/db/database.go → 空（F-04 面未被闭环提交触碰）"
re_verification:
  previous_status: gaps_found
  previous_score: "4/4 requirements 达成，3 个高/中危 GAP"
  gaps_closed:
    - "GAP-01 UpdateStatus 停用不入队（ee6fe937，变异验证转红）"
    - "GAP-02 profile 自助路径不入队（f595df59 + 97dd9a4b，变异验证转红）"
    - "GAP-03 Phase 58 移交项漏 import 直推路径（9e7460e9 + SC6 扩为两条）"
  gaps_remaining:
    - "GAP-04 F-10 Delete/BatchDelete 回滚测试（去向：Phase 58 移交待决项 3）"
    - "GAP-05 F-07~F-16 逐条判定未搬进被引用文件（去向：Phase 58 移交待决项 3 执行时一并）"
    - "GAP-06 57-VALIDATION.md 状态栏未回填（文档卫生，低）"
    - "GAP-07 两处 F-06 后失真注释（文档卫生，低）"
  regressions: []
gaps:  # 全部为低危记账/文档类，均注明去向，不阻塞目标
  - truth: "Delete / BatchDelete 的「队列表不可写 → 软删必须回滚」无用例（F-10）"
    status: partial
    reason: "本轮闭环只补了 Update / UpdateStatus / profile 三条路径的回滚测试；Delete/BatchDelete 回滚仍无守护"
    missing:
      - "补两用例：DROP 表 → svc.Delete / svc.BatchDelete → 断言 deleted_at IS NULL 且 sys_user_role 行数不变"
    destination: "Phase 58 移交待决项 3（F-07~F-16 复核）"
  - truth: "F-07~F-16 的逐条判定只存在于 57-REVIEW.md，ROADMAP 指向的 57-REVIEW-FIX.md 只有一行汇总"
    status: partial
    reason: "引用链可导航（REVIEW-FIX → REVIEW），但「逐条判定」措辞与实际不符；F-07（孤儿 disable 行）等项缺可执行描述"
    destination: "Phase 58 移交待决项 3 执行逐条复核时顺带落纸"
  - truth: "57-VALIDATION.md 的 Status 列 10 行仍为 ⬜ pending、File Exists 列仍为 ❌ W0"
    status: partial
    reason: "验证策略实际已执行（42 例全绿），策略文件未收口"
    destination: "文档卫生，随时可修"
  - truth: "user_service_test.go:52-54 注释仍写「tx.Save(&user) 写满全列」；57-03-SUMMARY.md:40 仍写「user 是指针」"
    status: partial
    reason: "F-06 后写路径已是 selective Updates；user 是值类型。零行为影响，仅误导后来者"
    destination: "文档卫生，随时可修"
human_verification:
  - test: "PG 真实环境验证 ON CONFLICT 部分索引合并路径"
    expected: "连续两次编辑同一用户 → ad_push_queue 仅 1 行 pending；若报 42P10 there is no unique or exclusion constraint matching the ON CONFLICT specification → 谓词写错。sqlite 侧 TestPhase57_EnqueueUpdate_MergeOnSecondEdit 已提供等价回归保护，但 PG 的冲突目标推断路径只能实机确认"
    why_human: "本仓无 PG 测试基座；QUEUE-02 的「DB 层合并语义成立」在 PG 上的实际执行未跑过"
    resolution: "PASS（2026-10-21 11:44 生产 PG 10.62.10.34/ruoyi 实机）：以生产同款子句 ON CONFLICT (entity_type, entity_id, action_type) WHERE status IN (pending, failed) DO UPDATE 推断成功未报 42P10；两次入队后仅 1 行且同一 id，status=pending/attempts=0/last_error/next_retry_at/failed_at 全清；再把该行打成 failed/attempts=3/last_error=boom 后第三次入队仍合并为 1 行（事务内 ROLLBACK 无残留）。UI 首测出现的 2 行 pushed 归因为终态行退出唯一索引的设计行为（TestPhase57_EnqueueUpdate_SyncsWithTerminalRow），非合并失效。证据在 57-HUMAN-UAT.md Test 1"
  - test: "进程重启不丢已入队行（QUEUE-01 / SC1 后半）"
    expected: "PG 部署后编辑用户入队 → kill -9 → 重启 → SELECT * FROM ad_push_queue WHERE status='pending' 行仍在"
    why_human: "需真实部署二进制 + 启停；57-VALIDATION.md 声明为 Manual-Only 且未执行"
    resolution: "PASS（2026-10-21 12:03-12:15 kill -9 实测）：行 4b743635-972c-4ec1-bc06-0fe2c51d5b0f（entity 8bd62962=chenchao-076, update_user）12:03:35.075 入队 pending/attempts=0；12:05:29 复核仍 pending；12:09:5x taskkill /PID 49428 /F 强杀后端（9000 释放）→ 杀后仍 pending/attempts=0/字段零漂移；12:11 重启（app.log 12:11:42 确认 drain_interval_seconds=300 生效、cron 0 */5 * * * * 注册）→ 12:15:47 复核仍 pending/attempts=0/updated_at 未变。持久化断言成立。证据在 57-HUMAN-UAT.md Test 2"
  - test: "AD 端复活链路闭环（QUEUE-06 端到端）"
    expected: "sys 删除用户 → Phase 58 排水器执行 DisableUser → AD 端 UAC 514 → bind 失败（data 533）"
    why_human: "需真实 AD；执行侧（排水器）已存在（internal/services/addomain/ad_push_queue_drain_service.go:400 的 `case models.ADPushActionDisable:` → `c.DisableUser(*user.AdDn)`，UAC 514），本 phase 只能验证队列行产生（已由测试锁定）"
    resolution: "PASS（2026-10-21 12:51 端到端）：UI 删除 d_chenchao（sys_user f5d4152b…，ad_dn CN=d_chenchao,OU=孝感中心支公司,…）→ 同事务入队 disable 行 df40eff4（created 12:51:02.472，比 user.deleted_at 晚 6ms）→ 12:51:30.004 该行被并发实例 10.62.10.39 推成 pushed（attempts=0、无 last_error）→ 用户确认域控账号已被禁用（UAC 514 生效）。AD 登录 data 533 步骤经用户裁决省略（域控账号禁用后不可登录属 AD 标准语义）。证据在 57-HUMAN-UAT.md Test 3"
---

# Phase 57 验证报告（AD Push Queue / outbox-lite 基建）

**Phase Goal**：队列写入路径的原子性与合并语义在 DB 层成立 — AD 侧属性变化可被可靠记录、合并与抑制，为 Phase 58 排水器提供可靠地基。
**结论**：**passed** —— 代码侧全部达成：4 条 requirement 落地、8 组变异测试证明守护真实可红、本轮 3 个高/中危 GAP 已闭环且经独立变异复核、无 Phase 58/59 越界。3 项人工项已于 2026-10-21 在真实 PG/真实 AD 环境执行完毕并全部 PASS（PG 合并路径、kill -9 重启耐久性、AD 禁用链路），证据见 57-HUMAN-UAT.md Test 1~3。剩余 4 项（GAP-04~GAP-07）均为低危记账/文档类且已注明去向，不阻塞目标。

---

## Gap Closure Re-check（2026-10-17 第二轮）

### GAP-01 → 已闭环（`ee6fe937`，变异验证 ✓）

- **代码事实**：`user_service.go` `UpdateStatus` 现为 `s.db.Transaction(...)` 包裹 —— 事务内 `tx.Model(&user).Update("status", ...)` + `EnqueueDisableOnStatusTransition(tx, oldStatus, models.UserStatus(status), id)`，入队失败 `return apperrors.DatabaseError(err)` 使 Commit 跳过。事务外 `First` 仅取旧状态做 0→1 diff，不参与写。与 Update/Delete/BatchDelete 同一形状。
- **注释改正**：`ad_push_queue_service.go` 的 `EnqueueDisableOnStatusTransition` 注释已改为如实陈述「两个入口现已同时接线」，不再保留与代码相反的旧表述。
- **文档诚实性**：`57-REVIEW-FIX.md` F-02 节**原文保留** + 显式更正块（①②两条事实错误逐条指出）+ 新增「GAP-01 补全」小节 —— 审计轨迹记录了「当时错在哪」，无回溯性美化。
- **变异复核（本人实跑）**：把接线换成 `_ = oldStatus; _ = id` → `TestPhase57_UserService_UpdateStatus_Disable_EnqueuesDisable` 与 `..._RollsBackWhenQueueInsertFails` 转红；`..._NoTransition_NoDisable` 与 `..._Disable_NoADConfig_NoEnqueue` 保持绿（防过度入队方向正确）。与提交信息声称的 RED 面完全一致。
- **方向性**：1→0 仍为 no-op（v1.21 无 enable 动作），与既有契约一致。

### GAP-02 → 已闭环（`f595df59` + `97dd9a4b`，变异验证 ✓）

- **代码事实**：`profile_service.go` `UpdateUserInfo` 包进 `s.db.Transaction(...)`：事务内读旧值快照（同 snapshot，T-57-02 同款约束）→ selective `Updates` → `EnqueueUpdateIfChanged(tx, &oldUser, ...)` 复用与管理员路径完全相同的 diff 纯函数与 D-02 两层降级。DeptID 留 nil（自助路径无部门字段）不参与 diff，语义正确。
- **无包环**：system → addomain 是既有依赖方向；DI 沿用零参内联构造，未改 `NewProfileService` 签名。
- **fixture 跟进**：`97dd9a4b` 给 api 层 `setupProfileTestDB` 补两张表（HYGIENE-06 第五套环境），`go test ./internal/api/v1/system/` 全绿。
- **变异复核（本人实跑）**：把入队调用换成空实现 → `TestPhase57_ProfileService_UpdateUserInfo_Change_Enqueues` 与 `..._RollsBackWhenQueueInsertFails` 转红；`NoADFieldChange` / `SameValue` / `NoADConfig` 三条防过度入队用例保持绿。
- **测试误报自纠**：提交信息如实记录了首版 RollsBack 断言写错（回滚后旧值保留 vs 期望 NULL）并修正 —— 生产代码无误。

**信任边界决策复核（ coordinator 要求明示立场）：我接受该决策，理由经独立核实成立，无冲突。**

1. **入队面核实**：`profile_handler.go` 四处 `c.Get("user_id")` —— userID 来自会话而非请求体，用户只能为自己入队，无跨用户写面、无权限提升。nickname/email/phone 本就是自助可改字段，队列行只是把这次自编辑的既有效果延伸到自己的 AD 账号。
2. **「不入队 = 数据丢失」的论据核实为真**：`user_sync_service.go` `updateExistingUser` 对 AD 同步用户的 email/phone **无条件**用 AD 侧值覆盖（nickname 仅空时回填）。即若不入队，AD 同步用户在个人中心改邮箱/手机会被下次同步**静默回滚**，且无队列行 → Phase 59 GUARD-01 以 pending/failed 行为判据会把这次分叉判成「一致」。入队是 v1.21 架构内让该编辑生效的唯一通路。
3. **不越界**：队列行在 Phase 58 排水器落地前是惰性记录；本路径不新增任何 LDAP 语义（D-05 重读 + 幂等 Modify 与管理员路径同一执行面）；门控仍是全局 D-02，未按「自助/管理员」新增区分。
4. **与 phase 上下文的一致性**：CONTEXT 的 D-01 字面只提管理员 Update，但 F-02（用户明确指示本 phase 修）已把「非 Update 路径的属性变更入队」带进范围；GAP-02 是同一缺口的自然补全而非新决策。选择「现在修」而非「记到 Phase 58」与我对该洞的结构性判断一致（守卫误判「一致」比缺功能更危险）。

### GAP-03 → 已闭环（`9e7460e9`，基于 `c725e88f`）

- ROADMAP Phase 58 移交待决项 1 现点名**两条**遗留直推路径：`user_handler.go:194`（单用户编辑）与 `user_import_handler.go:114` `triggerADSyncAfterImport`（批量导入，含其专用语义：无 ad_dn 跳过、并发上限、失败降级）。
- Success Criterion 6 扩为「**两条**遗留直推路径……或删除、或与排水互斥，不接受两者并存且都可能触发的终态」。具体、可验收。

### GAP-04 ~ GAP-07 → 未闭合（低危，去向已注明）

见 frontmatter `gaps`。本轮闭环未触及：Delete/BatchDelete 回滚测试仍缺（GAP-04 → Phase 58 移交项 3）；REVIEW-FIX 的 F-07~F-16 仍只有一行汇总（GAP-05 → Phase 58 移交项 3 执行时落纸）；VALIDATION 状态栏仍未回填（GAP-06）；两处失真注释仍在（GAP-07）。均为零行为影响的记账/文档项。

---

## 1. Requirement 追溯表

| Requirement | 计划声明 | 代码证据 | 判定 |
|---|---|---|---|
| **QUEUE-01**（编辑后同事务入队、重启不丢） | 57-02 / 57-03 | 四条写路径全部同事务入队并失败回滚：`user_service.go` Update `:256-259`、UpdateStatus（GAP-01 闭环后）、Delete `:308`、BatchDelete `:648`；外加 `profile_service.go` UpdateUserInfo（GAP-02 闭环后）。回滚用例 ×3（Update / UpdateStatus / profile）均 DROP 表断言业务写未落库。队列为 DB 表，重启天然不丢（重启实机验证列入 human_verification） | **✅ 达成** |
| **QUEUE-02**（同实体多次编辑合并，仅留最新） | 57-01 / 57-02 | 部分唯一索引 `uq_ad_push_queue_active ... WHERE status IN ('pending','failed')`（migration_215:79-84，谓词取自 `models.ADPushQueueActivePredicate` 单一事实源）；`adPushOnConflictClause` 谓词放 **`TargetWhere`**、裸 SQL 用 `clause.Expr`、无表名限定；`TestPhase57_EnqueueUpdate_MergeOnSecondEdit`（failed 行打回 pending、退避字段清零）、`..._SyncsWithTerminalRow`、`TestPhase57_ActiveRowPredicate_MatchesIndexDDL`。PG 实机路径列入 human_verification | **✅ 达成**（sqlite 层面；PG 实机待验） |
| **QUEUE-06**（sys 删除用户入队 disable） | 57-02 / 57-03 | Delete / BatchDelete 事务内 `EnqueueDisable(tx, …)`；F-01 压制同实体活跃 `update_user` 行；**GAP-01 闭环后停用两入口（编辑表单 + 列表页开关）均入队 disable** | **✅ 达成**（AD 端 UAC 533 闭环属 Phase 58 执行侧） |
| **HYGIENE-06**（sqlite 测试补建表、套件全绿） | 57-01 / 57-03 | **五套** sqlite 环境就位：user_service_test①、user_handler_test②、user_cache_impl_test③（计划外）、profile_service_test④（GAP-02）、profile_handler_test⑤（GAP-02 跟进）。`TestPhase57` 42 例全绿；全量套件唯一历史 FAIL 为 `internal/device` 既有 flaky（单包 3/3 PASS，不在本 phase diff 内） | **✅ 达成** |

---

## 2. 逐 must_have 判定

### 57-01（schema 层）

| must_have | 判定 | 证据 |
|---|---|---|
| 双方言幂等建表 | ✅ | migration_215 PG `timestamptz` / sqlite `datetime` 双常量；`TestPhase57_Migration215_SQLiteDDL`（含幂等重入） |
| 部分唯一索引只在 pending/failed 生效 | ✅ | `TestPhase57_Migration215_PartialUniqueIndex`（裸 SQL 插入，能红） |
| 模型无 DeletedAt / 无 uniqueIndex tag | ✅ | 无 `DeletedAt`/`BaseModel`/`uniqueIndex`；pragma 断言无 `deleted_at` 列 |
| sqlite 时间列可扫回 `time.Time` | ✅ | 往返用例断言 CreatedAt/UpdatedAt 非零 |
| key_link：database.go 注册 | ⚠️ 形式偏离、功能更优 | 计划要求双分支各直调一次；实际收敛为两分支共调 `ensureADPushQueue()` → 单一 DDL 事实源（F-03 改造后形态）。计划的字面 grep 判据失效，语义超额满足 |

### 57-02（入队 service）

| must_have | 判定 | 证据 |
|---|---|---|
| 四字段实际变更才入队 | ✅ | `HasADFieldChanged` 纯函数 + 13 子用例边界 |
| 连续编辑合并、failed 打回 pending | ✅ | MergeOnSecondEdit（三次入队恒 1 行 + 五字段重置断言） |
| 无启用配置 → 不入队 + WARN + 放行 | ✅ | 两条触发路径均断言；`IsADSyncEnabled` 用同一 tx |
| INSERT 失败返回 error | ✅ | PropagatesInsertFailure + 三条路径的接线级回滚用例 |
| delete 入队 disable、批量去重 | ✅ | 4 用例；去重真断言在纯函数（F-05） |
| key_link：ON CONFLICT 谓词与索引一致 | ✅ | 共用常量 + DDL 断言双防线 |

### 57-03（user_service 接线）

| must_have | 判定 | 证据 |
|---|---|---|
| 编辑同事务入队、失败回滚 | ✅ | 快照 `:163` + 入队 `:256-259`；回滚用例 |
| Delete / BatchDelete disable 行 | ✅ | `:308` / `:648` |
| 所有触碰写路径的 sqlite 环境有表 | ✅ | 实测五套（计划列两套） |
| handler 生产代码零改动 | ✅ | `git diff 8246173a..HEAD -- user_handler.go` 空；fixture 改动仅限测试文件 |
| key_link：传 tx、闭包内、错误上抛 | ✅ | 全部调用点在 Transaction 闭包内 `return nil` 之前 |

---

## 3. F-01 ~ F-16 逐条状态（变异测试独立验证）

| Finding | 状态 | 验证手段 |
|---|---|---|
| **F-01** disable 压制同实体活跃 `update_user` 行 | ✅ 已修（变异 ✓） | 删压制块 → 3 例转红；`dead` 在谓词外退出索引；压制不受 D-02 门控（安全性不变量） |
| **F-02** 编辑表单停用入队 disable | ✅ 已修并补全（GAP-01，变异 ✓） | 0→1 在 Update 事务内入队（删接线 → 2 例红）；**GAP-01 闭环后 UpdateStatus 入口同样接线（删接线 → 2 例红）**；1→0 刻意不入队有断言；REVIEW-FIX 事实错误已原文保留 + 更正块 |
| **F-03** SKIP_AUTOMIGRATE 旁路补建 | ✅ 已修（变异 ✓） | 旁路 → `ensureADPushQueue()` → migration_215 同源 DDL（含部分唯一索引，非裸 CreateTable）；删接线 → 2 例红 |
| **F-04** 索引失败阻断启动 | ✅ 已修（变异 ✓）| PG `:1058` / sqlite `:1091` 均 `return err`；改回非阻断 → 1 例红。**复核结论维持**：trade 正确——与本仓既有硬前置先例一致（`backfillSysAPIKeysKeyHash` database.go:807 直接 return err、`sanitizeSQLiteModelDefaults` :796 同），DDL 全 `IF NOT EXISTS` 只有真失败才触发，且「表建不出来」本就同样致命。闭环提交未触碰该面（`git diff ee6fe937~1..HEAD -- database.go` 空） |
| **F-05** 去重测试有效性 | ✅ 已修（变异 ✓） | `DedupeEntityIDs` 纯函数 7 子用例；改坏 → 5 例红。旧 sqlite 用例确认为假绿且注释已诚实标注 |
| **F-06** 省略字段静默清库 | ✅ 已修（变异 ✓） | selective Updates 只写 `!= nil`；注入旧 nil→NULL 语义 → 2 例红；显式空串清空部门有断言 |
| **F-07** BatchDelete 不校验 id 存在 → 孤儿 disable 行 | ⏳ 延后（属实：`:648` 原样传 ids） | Phase 58 移交项 3 |
| **F-08** 遗留即时推送双推 | ⏳ 延后 | ROADMAP 移交项 1（**现含两条路径**）+ SC6，具体可执行 |
| **F-09** 队列表无 ad_config_id | ⏳ 延后 | ROADMAP 移交项 2（决策点明确：加列 vs 排水时解析） |
| **F-10** Delete/BatchDelete 回滚无测试 | ⏳ 仍缺（= GAP-04） | Phase 58 移交项 3 |
| **F-11** 多次编辑把 attempts 归零 → 毒丸行 | ⏳ 属实 | Phase 58 移交项 3 |
| **F-12** UTC 约定不一致 | ⏳ 属实，低危 | 可接受 |
| **F-13** oldUserSnapshot 注释 | ✅ 已随 F-06 改写（`:157-162` 不变量注释）；SUMMARY 旧文案未同步（= GAP-07） | — |
| **F-14** 模型与 DDL status 默认值不一致 | ⏳ 属实，低危 | — |
| **F-15** fixture 走降级分支 | ⏳ 属实，注释已说明意图 | — |
| **F-16** BareInsertWithoutID 无契约断言 | ⏳ 属实，低危（`assert.NotPanics` + t.Logf） | — |

---

## 4. 目标反推：排水器能否安全建在这块地基上

**能。闭环后这四个问题全部有了答案：**

- **「AD 属性变更已提交却没有队列行」**——GAP-01/02 闭环后，**全部五条**已知触碰 AD 四字段/停用语义的写路径（Update / UpdateStatus / Delete / BatchDelete / profile UpdateUserInfo）都同事务入队并失败回滚。曾排查过的其他写路径不产生队列行的原因均属正确：`ResetPassword`/`ChangePassword`（密码非 AD 推送字段）、Excel 导入 `CreateUser`（新建本地用户，无既有 AD 账号可分叉；其 AD 推送由遗留 `triggerADSyncAfterImport` 承担——已记入 ROADMAP 移交项 1）。
- **「payload 与用户当前状态不符」**——行载荷只有 entity/action/状态（D-05 禁快照），排水器重读 DB 取全量最新值，结构上不存在误导面。
- **「用户已删但编辑行残留」**——F-01 压制把活跃 update_user 行打死进 `dead` 终态（退出唯一索引），且不受配置门控。
- **「合并/夺锁在 DB 层是否真成立」**——部分唯一索引 + TargetWhere 谓词共用单一常量，DDL 与冲突目标结构上无法漂移，sqlite 行为有能红测试锁定；PG 实机执行列入 human_verification（唯一剩余验证面）。

**排水器仍须自担的输入（均已记入 ROADMAP 移交项，非本 phase 缺陷）**：F-07 孤儿 disable 行（排水时 entity 落空 → 走 QUEUE-05 dead 语义）、F-09 配置归属解析、移交项 1 的双推窗口、F-11 attempts 续命。

---

## 5. 故意不做的边界（过度膨胀检查）

| 应属 Phase 58/59 | 本 phase 是否越界 | 证据 |
|---|---|---|
| 排水器 / in-process cron | ❌ 未越界 | 无新增 cron；队列只有写入方，无消费者 |
| compare-and-move / OU 移动 | ❌ 未越界 | 无 MoveUser 相关代码 |
| 登录回写守卫 / GUARD | ❌ 未越界 | 无守卫代码 |
| 对账 VIEW | ❌ 未越界 | 无新增 VIEW 迁移 |
| Delete AD 账号（明令禁止） | ❌ 未越界 | 无 DeleteUser；动作仅 disable |
| JSON payload 快照（已判死刑） | ❌ 未越界 | 无 payload 列 |
| handler 旧推送路径删除 | ❌ 未越界 | `user_handler.go` 零 diff |
| GAP-01/02 闭环 | ✅ 在边界内 | 同一入队模式补全既有写路径覆盖，未引入任何 58/59 语义；profile 入队的执行面与管理员路径同一 |

变更面（8246173a..HEAD）：模型 + 迁移 + database.go + addomain 入队 service + user_service/profile_service 接线 + 6 个测试文件 + 规划文档。**无越界。**

---

## 6. 无法验证的部分（诚实清单）

| 项 | 为什么无法验证 | 去向 |
|---|---|---|
| PG 上的 42P10 推断路径 / 部分索引冲突目标 | 无 PG 测试基座；sqlite 等价面已锁，TargetWhere 写法经静态核对 | human_verification #1 |
| kill -9 重启后队列行存续 | 需真实部署启停 | human_verification #2 |
| AD 端 UAC 533 闭环 | 需真实 AD + Phase 58 执行侧 | human_verification #3（本质是 Phase 58 验收项） |
| PG 分支 fail-fast（database.go:1058） | 代码走查确认，无 PG 测试路径 | Phase 58/部署验证清单 |
| 去重防的 PG 专属故障（cannot affect row a second time） | sqlite 不复现 | 代码注释已指向 Phase 58 清单 |
| SKIP_AUTOMIGRATE 真实启动 | 单元层验证 BootstrapMissingTables 行为 | 部署验证清单 |
| `timestamptz` PG 往返 | sqlite 路径验证 | 部署验证清单 |

---

## 7. 最终判定

**human_needed。**

代码侧：四条 requirement 全部落地；8 组变异测试（F-01~F-06 + GAP-01/02）证明每条守护都真正能红；五套 sqlite 环境就位；42 例 TestPhase57 全绿；无 Phase 58/59 越界；REVIEW-FIX 文档在补全的同时保留了带更正块的历史原文。**phase goal「队列写入路径的原子性与合并语义在 DB 层成立」在可程序验证的范围内全部成立。**

剩余事项分两类：
1. **4 项低危记账/文档债**（frontmatter gaps，均注明去向）——不阻塞，建议 Phase 58 动工前顺手清掉 GAP-06/07（纯文档）。
2. **3 项人工/环境验证**（PG 实机合并、重启耐久、AD 端闭环）——phase 自己的 VALIDATION 策略声明为 Manual-Only 且未执行；其中第三项本质是 Phase 58 验收项。等待人工确认后本 phase 可完全关账。

---

*验证者：Claude（verify-work）· 两轮验证，8 组变异测试全部还原，`git status --porcelain -- internal/` 为空，未提交任何文件*
