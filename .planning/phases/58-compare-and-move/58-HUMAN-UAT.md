---
status: partial
phase: 58-compare-and-move
source: [58-VERIFICATION.md, 58-VALIDATION.md]
started: 2026-10-19T00:00:00+08:00
updated: 2026-10-19T00:00:00+08:00
---

## Current Test

[awaiting human testing]

> **为什么这些项不可自动化**：本仓 Phase 58 的全部自动化验证跑在
> `github.com/glebarez/sqlite` 的 in-memory 基座上 + 手写 `mockLDAPClient`。
> mock 只能证明「我们调用了 MoveUser 且传了这个参数」，**无法证明 AD 端真的执行了
> ModifyDN**；也无法提供真实时钟跨度、真实域控的重启周期、第二个域。
> 假装覆盖这些项比不覆盖更坏 —— Phase 57 已为 PG 专属契约断言留 v1.22 的先例。
>
> **凭据提示（T-58-07-05）**：UAT-58-2 需要临时改错服务账号密码。本清单**不含任何真实
> 密码或密文样例** —— 请自行填写，且**务必执行「恢复」小节**。

---

## Tests

### UAT-58-1. 真 AD 上的 compare-and-move（QUEUE-04 / SC2）

**这项验的是什么：** 自动化用例（`TestPhase58_Drain_MoveUpdatesAdDnAndAdOuDn` 等）只证明
「我们调了 `MoveUser(dn, targetOU)`」。真域控上 ModifyDN 的 wire 行为 —— 旧 RDN 是否真被
删除、RDN 属性是否随 move 保留、`deleteOldRDN=true` 的真实后果 —— 只有域控能回答。

**前置：**
- 真实域控可达，`sys_ad_config` 有一条启用配置（`sync_enabled=true AND status=0`）
- `sys_ad_service_accounts` 中该配置有至少一个 `status=0`（可用）的账号
- `sys_dept_ou_mapping` 已由部门同步建好：用户目标部门有一行 `ou_dn`

**步骤：**
1. 选一个 `ad_dn` 非空、`ad_ou_dn` 指向某个已建 OU 的用户，记下当前 `ad_ou_dn`
2. 在「用户管理」页把该用户的部门改到一个**不同**的、已建 OU 的部门
3. 等 2 个排水周期（默认 30s × 2 + 余量，约 90s；Phase 59 守卫生效后可能更长）

**观察点 / 判据 SQL：**

```sql
-- ① 队列行终态
SELECT status, attempts, last_error, pushed_at
FROM ad_push_queue WHERE entity_id = '<userId>' ORDER BY created_at DESC LIMIT 1;
-- 期望：status = 'pushed'，attempts = 0，last_error = ''，pushed_at 非空

-- ② 本地 AD 引用已回写
SELECT ad_dn, ad_ou_dn FROM sys_user WHERE id = '<userId>';
-- 期望：ad_dn = 'CN=<原RDN>,OU=<新OU>...'，ad_ou_dn = 'OU=<新OU>...'
```

4. 到 AD 管理工具（dsquery / ADSI 编辑器）确认该用户对象确实在**新** OU 下，且
   RDN（如 `CN=zhangsan`）保留、其用户属性（displayName / mail / department）齐全

**幂等判据（SC2 后半）：**

5. 把该队列行手工打回活跃态，模拟异常回滚：

```sql
UPDATE ad_push_queue SET status = 'pending', next_retry_at = NULL, pushed_at = NULL
WHERE entity_id = '<userId>';
```

6. 等一个排水周期

**期望：**
- AD 侧用户**位置不变**（仍在新 OU），AD 的事件日志里**没有第二条 ModifyDN 记录**
- `sys_user.ad_dn` 与步骤 3 记录的值**逐字节相同**（未被二次改写）
- 队列行再次回到 `pushed`

**失败时该回报什么：** 若第 4 步 AD 侧报「源 DN 不存在」（LDAP code 32 / `No Such Object`），
说明 compare-and-move 的幂等点未生效（对已搬移的源 DN 重复 ModifyDN）→ **立即回报**，
这是 QUEUE-04 的核心失败模式。若第 6 步 AD 侧出现第二条 ModifyDN，同样立即回报。

**result:** [pending]

---

### UAT-58-2. 账号池熔断期间的退避节奏（QUEUE-03 / SC1 后半）

**这项验的是什么：** 退避曲线（60s→2min→4min→8min→16min，attempts 达 5 转 dead）与账号池
30min 熔断周期的**桥接**关系。自动化用例只验纯函数 `backoffDuration(n)` 的返回值，
不验「熔断真的发生了」也不验「attempts 在真实时钟下真的按这个节奏涨」。

**前置：** 生产/预发环境，**不要在生产做**（本项会短暂耗尽账号池）。
另备一个可手动恢复的 DB 备份。

**步骤：**
1. 记录起始状态：

```sql
SELECT status, attempts, next_retry_at, failed_at, last_error
FROM ad_push_queue WHERE status IN ('pending','failed') ORDER BY created_at LIMIT 5;
```

2. 手工把某配置的服务账号密码改错（**不要把真实密文写进任何文档或工单**）：

```sql
UPDATE sys_ad_service_accounts SET password_ciphertext = '<任意错误值>'
WHERE config_id = '<cfgId>';
```

3. 触发若干推送（编辑几个用户的部门 / 停用几个用户），然后每 5 分钟观察一次：

```sql
SELECT status, attempts, next_retry_at, failed_at, last_error
FROM ad_push_queue WHERE entity_id IN (...) ORDER BY created_at;
```

**期望：**
- `attempts` 依次递增，间隔约 **60s → 2min → 4min → 8min → 16min**
- `last_error` 前缀为 `pool:` 或 `retry:`（**不得**是 `dead:`，`dead` 说明误判）
- `attempts` 到达 **5** 后行转 `status='dead'`、`last_error='dead:max-attempts'`、
  `next_retry_at` 清空

4. 确认熔断确实被触发：

```sql
SELECT status, failure_count, circuit_breaker_until, last_failure_reason
FROM sys_ad_service_accounts WHERE config_id = '<cfgId>';
-- 期望：连续 3 次失败 → status = 2（熔断），circuit_breaker_until ≈ now + 30min
```

**失败时该回报什么：** 若退避间隔明显短于曲线（例如连续几秒内 attempts 从 1 涨到 5），
说明 D-05 的「熔断窗内不密集重试」意图未兑现 —— 立即回报。
若熔断状态下退避反而**完全停止**（`next_retry_at` 一直不变），回报。

**恢复（必做）：**
- 把 `password_ciphertext` 改回正确值，然后**重启后端**，或等
  `RecoverExpiredBreakers` cron（每 5 分钟）到期
- 确认 `sys_ad_service_accounts.status` 回到 `0`（可用）
- 确认卡在 `dead` 的队列行已被重新编辑触发入队（`dead` 行不自动复活 —— 见 UAT-58-7 的说明）

**result:** [pending]

---

### UAT-58-3. `sys_config` 排水间隔旋钮重启生效（D-05）

**这项验的是什么：** `LoadDrainIntervalSeconds` 在**启动时读取一次**，改后需重启。
自动化用例只验纯函数 `LoadDrainIntervalSeconds` / `SecondsToCronSpec` 的映射（含 6 段 cron
的 `*/N` vs `0 */M` 语义陷阱 F-E），不验「重启后 cron 真的按新间隔跑」。

**步骤：**
1. 在「参数设置」页找到 `ad_push_queue.drain_interval_seconds`（migration_216 seed，
   默认值 `30`，`config_type='N'` 非系统参数，可编辑），改为 `60`
2. **重启后端**
3. 在启动日志里找这一行：

```
[ad_push_queue] 排水间隔配置: ad_push_queue.drain_interval_seconds = 60s（改后需重启后端生效）
AD同步定时任务已启动（全量同步: 每5分钟 + 账号池恢复: 每5分钟 + ad_push_queue 排水: 0 */1 * * * *）
```

**观察点 / 期望：**
- 日志出现 `drain_interval_seconds = 60s`
- cron spec 是 **`0 */1 * * * *`**（6 段，第 6 位是分钟），
  **不是** `*/60 * * * * *`（后者秒位 `*/60` 会被解析成 `{0}`，实际每分钟只跑一次 —— F-E 陷阱）
- 连续两条队列行的 `pushed_at` 间隔约 **60s**

```sql
SELECT entity_id, pushed_at FROM ad_push_queue
WHERE pushed_at IS NOT NULL ORDER BY pushed_at DESC LIMIT 5;
```

**反向判据：** **不重启**就改值 → 间隔保持旧值不变（启动读取语义，不是 bug）

**失败时该回报什么：** 日志里的 spec 与期望字面不符（尤其出现 `*/60 * * * * *`），
或改了值不重启就生效了（说明读的是实时配置，与 D-05 的设计相反）→ 回报。

**恢复：** 把该参数改回 `30` 并重启。

**result:** [pending]

---

### UAT-58-4. 多 AD 配置（父子域）最长后缀命中（D-02 / SC3）

> ⚠ **单域环境请把本项标 N/A**，并在 `58-07-SUMMARY.md` 注明。
> 研究 §11 未解问题 1：`sys_ad_config` 在模型上是列表，但登录路径仍只取单条配置，
> D-02 的复杂度主要是**防御性**的 —— 单域环境下 `MatchADConfigByDN` 的歧义分支不可能触发。

**前置：** 两个域（父子域，共享后缀），例如：

| config | base_dn |
|--------|---------|
| 父域 | `DC=corp,DC=com` |
| 子域 | `DC=na,DC=corp,DC=com` |

**自动化侧已覆盖、本项只补真实部分：** `TestPhase58_MatchADConfigByDN_ParentChildSharedSuffix`
证明字符串级 `ends-with("DC=corp,DC=com")` 会误命中子域用户，**最长后缀唯一匹配**才是正解。
真实域控验的是「两条 base_dn 在两台域控上都真实存在，且各自 Modify 打到本域」。

**步骤：**
1. 配好两条启用配置（父子域共享后缀，`sys_dept_ou_mapping` 两条配置的 `ou_dn` 都建好）
2. 两个域各建一个用户，`ad_dn` 分别落在各自 base_dn 下
3. 分别编辑两个用户的部门，触发入队
4. 等一个排水周期

**观察点 / 判据 SQL：**

```sql
-- ① 两行都应 pushed，不得出现 no-ad-config
SELECT entity_id, status, last_error FROM ad_push_queue
WHERE entity_id IN ('<父域userId>','<子域userId>');
-- 期望：两行 status='pushed'，last_error=''（不是 'unrecoverable:no-ad-config'）

-- ② 两人的 OU 映射应落在各自 base_dn 下
SELECT dept_id, ad_config_id, ou_dn FROM sys_dept_ou_mapping
WHERE ou_dn IN (SELECT ad_ou_dn FROM sys_user WHERE id IN ('<父域userId>','<子域userId>'));
```

**期望：**
- 父域用户留在父域的 OU；子域用户留在子域的 OU
- **无跨域推送**（两行都不出现 `unrecoverable:no-ad-config`）
- 反例提示：若子域用户的 `ad_dn` 被误改写到父域 base_dn 下，或其行转
  `unrecoverable:no-ad-config`，说明最长后缀匹配失效

**失败时该回报什么：** 任何一行转 `dead` 且 `last_error='unrecoverable:no-ad-config'`，
或 AD 侧出现跨域 ModifyDN → 立即回报，附上两个 `ad_dn` 字面值与两条 `base_dn`。

**result:** [pending / N/A（单域环境）]

---

### UAT-58-5. code 32 真实触发（QUEUE-05 / SC5）

**这项验的是什么：** 自动化用例用 `ldap.NewError(ldap.LDAPResultNoSuchObject, ...)`
注入 code 32，证明**分流逻辑**正确。真域控验的是「AD 端确实返回了 code 32 而不是别的
错误码」，以及 `clearUserADRefs` 的写回在真实 PG 上生效（sqlite 与 PG 的
`Updates(map{"ad_dn": nil})` 行为理论上同构，但没实机跑过）。

**步骤：**
1. 在 AD 管理工具里删掉某个用户的 **OU**（或整个用户对象）
2. 回到系统，编辑该用户的任一四字段之一（nickname / email / phone / 部门）触发入队
3. 等一个排水周期

**观察点 / 判据 SQL：**

```sql
-- ① 队列行终态
SELECT status, attempts, last_error FROM ad_push_queue
WHERE entity_id = '<userId>' ORDER BY created_at DESC LIMIT 1;
-- 期望：status='dead'、attempts=0（未烧）、last_error='unrecoverable:target-not-found'

-- ② AD 引用已清空
SELECT ad_dn, ad_ou_dn, ad_synced_at FROM sys_user WHERE id = '<userId>';
-- 期望：三列均为 NULL
```

4. 再等 3 个周期

```sql
SELECT status, attempts, updated_at FROM ad_push_queue
WHERE entity_id = '<userId>' ORDER BY created_at DESC LIMIT 1;
-- 期望：status 仍为 'dead'，attempts 仍为 0，updated_at 不再变化
```

**失败时该回报什么：**
- ⚠ **若第 ② 步 `ad_dn` 未被清空 → 立即回报**（QUEUE-05 未兑现）
- 若第 ① 步 `attempts` 不是 0（烧了退避次数）→ 立即回报（不可恢复错误不应退避重试）
- 若第 ④ 步 `updated_at` 仍在变 → 说明行仍被反复拾取，终态机制失效

**result:** [pending]

---

## ⚠ 用户必须知晓的两项已知后果（非 bug，勿当回归上报）

### UAT-58-6. 部门同步从未跑过时 `update_user` 行批量转 `dead`（决策 Q2 的刻意后果）

**⚠ 这是本次部署最可能被误报为「功能坏了」的现象，请务必先读这一条。**

**症状：** 在**从未执行过部门 AD 同步**的环境里（`sys_dept_ou_mapping` 表为空），
上线 Phase 58 后，用户一编辑部门，队列行会**批量**转成终态 `dead`：

```sql
SELECT status, last_error, COUNT(*) FROM ad_push_queue
WHERE last_error = 'unrecoverable:no-ou-mapping' GROUP BY status, last_error;
-- 期望：全部 status='dead'
```

对应日志/DB：

| 字段 | 值 |
|------|-----|
| `status` | `dead` |
| `attempts` | `0`（不可恢复分支**不烧**退避次数） |
| `last_error` | `unrecoverable:no-ou-mapping` |
| `pushed_at` | `NULL` |
| LDAP 写次数 | **0**（`moveUserCalls` 为 0，不打扰域控） |

**根因：** compare-and-move 需要知道「目标部门在 AD 树里的哪个 OU」，这唯一来源是
`sys_dept_ou_mapping`（由**部门同步**建，不在本 phase 范围内）。表为空 ⇒ 查不到目标 OU ⇒
按 **Q2 决策**转 `dead`（而不是重试、也不是误报 `pushed`）。

**为什么这是刻意设计：** 查不到 OU 映射时若退避重试，只会无限堆积 `pending` 行
（映射不会自己出现）；若误报 `pushed`，Phase 59 的信号守卫会判成「无待推送 → 已一致」，
把「系统说推了、AD 里其实没推」的偏差永久掩盖 —— 那比一批 `dead` 危险得多。

**如何确认是它、而不是真故障（判别三问）：**

1. `last_error` **恰好是** `unrecoverable:no-ou-mapping` 吗？其它前缀
   （`unrecoverable:target-not-found` / `unrecoverable:no-ad-config` /
   `unrecoverable:user-not-found` / `dead:max-attempts` / `retry:*` / `pool:*`）
   都**不是**这个症状
2. `sys_dept_ou_mapping` 是空的吗？

   ```sql
   SELECT COUNT(*) FROM sys_dept_ou_mapping;
   -- 为 0 → 就是它
   ```

3. 域控侧该用户**没有**被移动过吗？（若是，则确认本地只做了判定、没打扰 AD）

**处置（不是 bug，不需要改代码）：**

1. 执行一次**部门 AD 同步**，把 OU 映射建出来
2. 重新编辑受影响用户的部门（`dead` 行**不会自动复活** —— 见 UAT-58-7），
   或批量把它们打回 `pending`：

```sql
-- 确认清单后再执行；不要盲目全表刷
UPDATE ad_push_queue SET status = 'pending', next_retry_at = NULL
WHERE last_error = 'unrecoverable:no-ou-mapping' AND status = 'dead';
```

3. 等一个排水周期，确认为 `pushed`

**长期规避：** 新环境部署后**先跑部门同步再放开用户编辑**。

**result:** [pending]

---

### UAT-58-7. 「编辑即续命」：持续编辑且持续失败的行永不终态（F-11 已接受方向）

**⚠ 这是有意的设计取舍，不是缺陷。报 bug 前请先读这一条。**

**现象：** 某个用户的某个属性（比如一个格式非法的部门名导致 OU 永远建不出来）持续推送失败。
如果有人**持续编辑**这个用户的该属性，每一行会这样循环：

```
编辑 → EnqueueUpdateIfChanged / EnqueueUpdateBatch 的 D-06 合并子句
     → 把行打回 status='pending' 且 attempts 清零（Phase 57 F-11 归零点）
排水 → 本轮失败 → attempts 1 → failed + 退避 60s
编辑 → attempts 清零 → pending
排水 → attempts 1 → failed + 退避 60s
...
```

**结果：`attempts` 永远涨不到 5，行永远不会转 `dead`。**

**为什么这是刻意的：** `attempts` 的语义是「**自上次编辑以来**的连续失败次数」。
一个人正在反复编辑某项设置，说明他还在管这件事；此时把行打成 `dead` 并停止重试，
等于系统在他眼皮底下停止干活。反复重试才是期望行为。

**为什么排水器不能简单地「不管编辑、直接判死」：** Phase 57 的合并子句（每次编辑清零
`attempts`）与排水器的 `maxDrainAttempts` 阈值之间存在一个已知的相互作用。若排水器在
**拾取阶段**或**成功路径**上凭裸 `attempts` 判死，会误杀「attempts 恰好 ≥5 但本轮其实成功了」
的行 —— `TestPhase58_Drain_NoBareAttemptsDeadAfterMerge` 锁死了这一条：`attempts=5` 的行本轮
成功必须转 `pushed`。因此 dead 判定**只允许**发生在**本轮实际失败分支内**，且按
`attempts+1 >= 5` 判定。

**如何观察与确认：**

```sql
SELECT entity_id, status, attempts, last_error, updated_at
FROM ad_push_queue WHERE status IN ('pending','failed') ORDER BY updated_at DESC;
-- 若某行的 attempts 长期在 0/1 之间摆动、status 在 pending/failed 间来回，
-- 且 updated_at 跟着人工编辑时间变动 → 就是这个现象
```

**处置：** 修根因（把那个一直失败的原因修掉），而不是等它自己进 `dead`。
如果确实需要让一行进入终态、不再自动重试，可以手工：

```sql
UPDATE ad_push_queue SET status = 'dead', last_error = 'manual:abandoned', next_retry_at = NULL
WHERE entity_id = '<userId>';
```

（手工写 `dead` 后，只要不再编辑该用户，它就不会被重新入队。）

**result:** [accepted by design — D-04 / F-11]

---

### UAT-58-8. 含特殊字符的历史部门需重跑部门同步（HYGIENE-02 / Q5 的已知取舍）

**现象：** Phase 58 新增了 OU DN 转义 helper（`BuildOUDN`），部门名里的
`,` `=` `+` `<` `>` `#` `;` `\` `"` 等 LDAP DN 保留字符会被正确转义。
但**只作用于新写入**。部署前就已存在的 `sys_dept_ou_mapping.ou_dn` 行不会被回溯改写。

**确认方法：** 找一个名字含逗号等特殊字符的**历史**部门：

```sql
SELECT d.dept_name, m.ou_dn
FROM sys_dept d JOIN sys_dept_ou_mapping m ON m.dept_id = d.id
WHERE d.dept_name ~ '[,=+<>#;\\"]';
-- 期望：ou_dn 里这些字符**未被转义**（没有前置反斜杠）→ 属历史行，需重跑
```

**对照：** 跑一次部门同步后重新查，`ou_dn` 里的保留字符应带反斜杠。

**处置：** 重跑部门同步（该操作会重写映射行）。**不需要改代码。**

**result:** [pending]

---

### UAT-58-9. ⚠ OU 移动成功但本地回写失败 → 该用户 AD 绑定需人工重建（H-05 残余风险）

**⚠ 这是代码审查后**新增**的已知后果（收口时发现，修复为部分修复）。**

**现象：** 用户调岗触发 OU 移动。AD 侧移动**成功**，但紧接着写回本地 `sys_user.ad_dn` / `ad_ou_dn` 失败（写库瞬时故障）。修复前队列行会被标成 `pushed`（成功）而掩盖问题；**修复后**该行如实转为 `failed` 并排退避，`last_error` 以 `retry:move_ou:` 开头。

**为什么仍需人工介入：** 重试**不能**自愈。核查结论是 `MoveUser` 重放**不安全** —— 重试时行重读到的仍是陈旧 `ad_dn`，而 AD 端该 DN 已随首次移动消失 → `ModifyDN` 打在不存在的对象上得 code 32。

**人工判据：**

```sql
-- 找出本地 DN 指向 AD 端已不存在对象的用户（写回失败后的典型特征）
SELECT u.id, u.username, u.ad_dn, u.ad_ou_dn, q.status, q.last_error
FROM sys_user u
JOIN ad_push_queue q ON q.entity_id = u.id
WHERE q.last_error LIKE 'retry:move_ou:%' OR q.last_error LIKE 'unrecoverable:target-not-found%'
ORDER BY q.updated_at DESC;
```

对命中行：在 AD 端确认用户当前实际所在 OU，把 `sys_user.ad_dn` / `ad_ou_dn` 手工改对（或触发一次走登录同步路径的重新绑定），再删掉该队列行。

**与 UAT-58-5（code 32）的区别：** UAT-58-5 是 **AD 端对象被真删了**，属正常收敛；UAT-58-9 是 **AD 端对象还在、只是本地记录陈旧**，属需人工纠正的数据不一致。用 `Get-ADUser -Identity <RDN> -SearchBase <当前OU>` 确认对象是否真实存在即可区分。

**result:** [pending]

---

## Summary

| 编号 | 标题 | 归属 | 状态 |
|------|------|------|------|
| UAT-58-1 | 真 AD 上的 compare-and-move（含二次排水幂等） | QUEUE-04 / SC2 | [pending] |
| UAT-58-2 | 账号池熔断期间的退避节奏 | QUEUE-03 / SC1 | [pending] |
| UAT-58-3 | `sys_config` 排水间隔旋钮重启生效 | D-05 | [pending] |
| UAT-58-4 | 多 AD 配置（父子域）最长后缀命中 | D-02 / SC3 | [pending / 单域环境 N/A] |
| UAT-58-5 | code 32 真实触发 | QUEUE-05 / SC5 | [pending] |
| UAT-58-6 | ⚠ 部门同步从未跑过时 `update_user` 批量转 `dead` | Q2 刻意后果 | [pending — 必须知晓] |
| UAT-58-7 | ⚠ 「编辑即续命」：持续失败的行永不终态 | F-11 已接受方向 | [accepted by design] |
| UAT-58-8 | ⚠ 含特殊字符的历史部门需重跑部门同步 | Q5 / HYGIENE-02 取舍 | [pending] |
| UAT-58-9 | ⚠ OU 移动成功但本地回写失败 → 需人工重建 AD 绑定 | H-05 残余风险（PARTIAL FIX） | [pending — 必须知晓] |

```
total: 9
passed: 0
issues: 0
pending: 7
accepted_by_design: 1
na: 1 (UAT-58-4, 单域环境)
```

## Gaps

- **UAT-58-1~5 的自动化侧已有等价保护**（mock 层证明分流与调用参数正确），缺的是
  **wire 级真实性**（ModifyDN 真生效、真实熔断、真实重启、第二个域、真实 code 32）。
  二者不是替代关系：自动化证明「分支走对了」，人工证明「AD 端真的按那样响应了」。
- **UAT-58-6/7/8 不是「待验证」，是「待告知」**：它们是本 phase 的已知后果，
  不需要验证是否发生，需要的是**在部署时知道会发生**。UAT-58-6 尤其重要 ——
  未跑部门同步的环境上线后会出现大批 `dead` 行，极易被误报为回归。
- **Phase 59 的前置**：UAT-58-2 会消耗账号池额度，做完必须按「恢复」小节复位，
  否则会连带触发「管理员被锁」类事故（`ad-modify-fail-double-counts-breaker` 前科）。
