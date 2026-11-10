---
doc_type: C 类人工验证执行脚手架（C-a ~ C-d）
created: 2026-11-10
created_by: quick 261110-161（用户批准 C-a~C-d）
status: 纯文档 / 零代码 / 零勾选 / 零生产数据
hard_constraint: 不代执行、不代勾选、不代 QA 裁定、不写生产数据（沿用 261108-uat-writeback-policy.md 最高纪律）
input_note: 本文件是「让人工直接执行」的脚手架，不是执行结果；任何条目的勾选与证据仍须由人工执行人按回写口径落 GSD 产物
---

# C 类人工验证执行脚手架（quick 261110-161）

> 79 条跟踪表（`.planning/notes/261108-human-verification-tracker.md`）已是完整索引，
> 回写口径（`.planning/notes/261108-uat-writeback-policy.md`）已有通用证据模板。
> 本文件补齐 C-a ~ C-d 四类「可直接执行」缺口：**不新建勾选框、不预填判定、
> 不修改 79 条跟踪表与 149/150 UAT 载体原件、不改 `.planning/milestones/` 归档件**。
> C-e（具名验收人 / 重跑窗口 / 授权单号）仍留白，须由 QA / 发布负责人 / DBA 提供，AI 不代填。

## C-a 高频组可执行命令清单 + pass/fail 判据

覆盖 VAL-01（2 项）、VAL-02（7 项）、VAL-06（1 项）共 10 条。命令均为**只读或受控探针**，
执行前须具备对应授权；`{占位}` 由执行人按环境填入，禁止写入真实口令。

| 条目 ID | 可执行动作（脱敏后） | pass 判据 | fail / blocked 判据 |
|---|---|---|---|
| VAL-01-84-1 | 用仅挂 1 个角色的 `data_scope=3` 账号登录 → 调用户列表 → 记录 `total`；改角色后等 ≥31s 复测 | `total` 与「按该角色数据范围过滤后的可见行数」相等，且两次复测一致 | `total` 等于全库行数 → 过滤未生效；多角色污染 → blocked（换单角色账号） |
| VAL-01-84-2 | 生产 PG 只读执行 `\d sys_role_menu`、`\d sys_user_role`、`SELECT conname FROM pg_constraint WHERE conrelid IN (...)` | 列集/约束与 GORM 模型漂移登记一致（含 `created_at` 有无） | 无法连接 PG → blocked（登记授权缺口） |
| VAL-02-87-1 | 按 `87-RESEARCH §E19` 判定 SQL 查受影响生产角色清单 | 清单逐条列明；若仅 admin 则注明「业务角色未配置」 | 查询被拒 → blocked |
| VAL-02-87-2 | 迁移前后计数对照（菜单 / 授权 / 角色 / 用户） | 差 = +65 菜单行 / +N 授权行 / **0 角色变更 / 0 用户变更** | 任一非预期变更 → fail |
| VAL-02-87-3 | 生产部署窗口执行 Migration 226 后跑 `SELECT count(*) FROM sys_menu WHERE menu_type='F' AND perms IN (<65 码>)` | 结果 = 65 | ≠65 → fail；窗口未获批 → blocked |
| VAL-02-87-4 | `\d sys_role_menu` 复核唯一键 | 记录约束名，或确认「无唯一键」（与 Migration 226 `NOT EXISTS` 双方言兼容一致） | 查询被拒 → blocked |
| VAL-02-87-5 | 补授前请求 → 403；等 31s → 200；撤销复原再等 31s → 403 | 三段状态码序列精确为 `403 → 200 → 403` | 任一段不符 → fail（注意 30s TTL） |
| VAL-02-87-6 | **写操作授权门槛**：清 sessionStorage → 编辑角色 → 树中 65 按钮展示 → 勾选保存 → 刷新回读 | 65 按钮全展示、保存后回读保持勾选 | 无写授权 / 无 AD 账号 → blocked（**AI 不得切 local 认证绕过**） |
| VAL-02-87-7 | 发布记录核对「先 Migration 226 → 再前端包与门禁脚本」 | 顺序记录完整 | 顺序颠倒 → fail |
| VAL-06-DBA-01 | DBA 回传 migration_180 对账 MV 超时根治的生产验证结论 | 回传文档含环境、执行时间、结论 | 未回传 → 保持待执行（不标完成） |

> VAL-03/04/05 共 69 条的命令清单按同一格式扩展；现场真机组（VAL-04 29 项）须按
> 「基线采集 → dry-run → `--apply` → 验证 → 恢复」顺序，凭据走 `SITE_USER`/`SITE_PASS`
> 或 ssh-agent，**禁止写入脚本/记录文件**（回写口径 §2.2）。

## C-b 149/150 UAT 载体的判据表 + 回写映射

下列为**判据与映射**，不是勾选结果；原件 `149-UAT-RECHECK.md` / `150-UAT-BACKFILL.md`
保持只读不改，勾选仍由人工执行后按回写口径落到 GSD 产物。

| 载体条目 | 验收判据 | 证据位（建议） | 回写到 79 条跟踪表的映射 |
|---|---|---|---|
| 149 D1 几何/附件越界读 | Dept 档越界 `floorId`/`roomId` 列表返回**空集**（非 403） | `evidence/149-D1-<日期>.md` | `VAL-05-133-D1`（tracker §六） |
| 149 D2 location-alias 越界 | `dept_id ∉ 可见集` 的 alias 行读空集、写 403、批量裁剪为空 | `evidence/149-D2-<日期>.md` | `VAL-05-133-D2` |
| 149 断链 fail-closed | 软删 floor/room/alias 越界行 → 读空集、写 403；All 档短路仍可见 | 同上 | `VAL-05-133-D1/D2` 断链子项 |
| 149 保存侧几何不误删 | 非超管保存整层后，越界几何行数**前后不变** | 同上 | `VAL-05-133-D1` 保存子项 |
| 149 双方言 403 矩阵 | PG + SQLite 同一 `ScopeResult` 的 COUNT 与 List 行数一致 | 同上（需 `TEST_PG_DSN`） | 同组； PG 未注入 → blocked 登记 |
| 150 A6 越界零回显 | 响应体不含 `scopeOutOfScope`/越界 `Row`；`failed`/`errors` 不随越界行数变化；越界仅入 operlog | `evidence/150-A6-<日期>.md` | `VAL-05-133-A6` |
| 150 All 档零回归 | `candidates/toFill/toCorrect/skippedAlreadyValid/samples` 与 130 前一致、`samples ≤20`、响应无 6 个规模字段 | 同上 | `VAL-05-133-A6` All 子项 |
| 150 差分攻击 | 两相邻 scope（Dept/DeptChild）、全库 N 不同但可见 M 相同 → 响应体数值逐字节相等 | 同上 | `VAL-05-133-A6` 差分子项 |

> 重跑窗口与验收人：**待 QA 裁定**（C-e，AI 不代填）。B（Phase 151 ScopedDB）已落地，
> 但「不得宣称 ① 已根治」的历史边界声明以 `150-UAT-BACKFILL.md §6` 原文为准，不在此改写。

## C-c 证据模板实例（空白，值留空）

按 `261108-uat-writeback-policy.md §3.2` 的单条模板生成，**值一律留空**供执行人填写。

```markdown
## VAL-01-84-1
- 来源 REQ：VAL-01（Phase 84 PG 实机验证）
- 执行日期：（留空）
- 执行人：（留空，PG 授权方 / 现场运维同事）
- 前置授权凭证：（留空，授权单号 / PG 只读账号，不含口令）
- 环境：（留空，生产 / 预发）
- 步骤与结果：（留空，脱敏输出：账号角色、total、可见行数、两次复测对照）
- 判定：（留空，通过 / 不通过 / blocked）
- 证据位：（留空）
- 回写动作：（留空，在 GSD 产物登记并勾选本条目）
- 副作用核对：（留空，既有行零改动 / 生产配置未切换）
```

```markdown
## 149-D1
- 来源 REQ：VAL-05-133-D1（Phase 149 D1 重跑）
- 执行日期 / 执行人 / 前置授权凭证 / 环境：（留空）
- 步骤与结果：（留空，越界 floorId 的空集断言输出）
- 判定 / 证据位 / 回写动作 / 副作用核对：（留空）
```

```markdown
## 150-A6
- 来源 REQ：VAL-05-133-A6（Phase 150 backfill 重跑）
- 执行日期 / 执行人 / 前置授权凭证 / 环境：（留空）
- 步骤与结果：（留空，预览/执行响应体 + 差分攻击对照）
- 判定 / 证据位 / 回写动作 / 副作用核对：（留空）
```

> 其余 76 条按同一模板实例化即可；**本文件不预建 evidence 目录、不预填证据内容**
> （回写口径 §3.1：证据一律落在当次执行所在的里程碑/phase 工作区）。

## C-d 回写命令清单（执行人回写后由 Lead 跑）

参数按届时 GSD 版本填，不硬编码不存在的参数（回写口径 §4 语义）：

| 时机 | 命令 / 动作 | 说明 |
|---|---|---|
| 单条执行完 | 在执行所在 phase 工作区落 `evidence/<条目ID>-<YYYYMMDD>.md` | 字段名见 C-c 模板 |
| 回写时 | 在 GSD 产物 / 未来里程碑 UAT 清单把 `- [ ]` 改 `- [x]`，**同处记录** `清偿日期 + 执行人 + 证据位 + 判定` | 只有勾选无证据 = 伪造闭环（回写口径 §3.3） |
| 复核 | `gsd-sdk query audit-uat --raw`（参数按届时版本） | 核对 `uat_gaps` / `open_scenario_count` |
| 收档前 | `gsd-sdk query audit-open --raw` + 对仍 open 项 acknowledge | 沿用 `MILESTONES.md` / `RETROSPECTIVE.md` 先例 |
| blocked 项 | 保持 `- [ ]`，证据位登记阻断原因与解锁条件 | 例如「无 H3C 设备，conditional deferred」 |

## C-e 仍需人工提供的输入（AI 不代填）

- 具名验收人（149/150 重跑、79 条各组）
- 重跑窗口日期
- 87-6 写操作授权单号 / 生产部署窗口编号
- PG 只读账号来源、现场真机型号与固件版本
- DBA migration_180 回传结论

> 上述任一未提供 → 对应条目保持「待执行」，**不得勾选**。
