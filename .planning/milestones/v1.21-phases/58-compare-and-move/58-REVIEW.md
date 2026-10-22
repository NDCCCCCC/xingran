# Phase 58 咨询式代码审查（ADVISORY REVIEW）

**审查日期:** 2026-10-19
**审查范围:** `b2909660~1..c7e78ee5`（13 个实施提交）
**审查者:** Claude（对抗式审查，非执行 agent）
**基线核对:** `go build ./...` 退出码 0；`go test ./internal/services/addomain/ -run TestPhase58 -count=1` → `ok`。审查开始前基线是绿的，下述问题都不是基线破损造成的。

---

## 0. 总体判定

**没有发现 BLOCKER。**

审计三条否决项（时间戳仲裁 / 新旧值比较 / `sys_job` 注册）与两条次级否决项（JSON 快照 / 删 AD 账号）经逐行核对**均未被命中**，证据见第 6 节。D-02 路由（`MatchADConfigByDN`）的逗号边界与等长歧义处理经推演是正确的，未能构造出错误路由的输入（见第 5 节）。

发现的问题集中在**并发互斥的宣称与实现不符**、**退避曲线在生产不可达**、**终态写入与 AD 实际状态可能脱节**三处，均可归为 HIGH/MEDIUM，没有一条会造成不可逆的数据损坏或越权。

统计：

| 严重度 | 数量 |
|--------|------|
| BLOCKER | 0 |
| HIGH | 5 |
| MEDIUM | 5 |
| LOW | 7 |

---

## 1. HIGH

> **修复状态（2026-10-19）**
>
> | 编号 | 状态 | 提交 |
> |------|------|------|
> | H-01 | ✅ FIXED | `c81890dc` |
> | H-02 | ✅ FIXED（随 H-01 同提交） | `c81890dc` |
> | H-03 | ⏸ DEFERRED（仅加注释记录，逻辑不改） | `550f35e3` |
> | H-04 | ✅ FIXED（哨兵改用新增的 `ErrAllAccountsDialFailed`，理由见提交说明） | `18d6421e` |
> | H-05 | ⚠ PARTIAL FIXED（详见该节末的"修复与残余风险"） | `3f55b7f7` |
>
> 审计三条否决项（禁时间戳仲裁 / 禁新旧值比较 / 禁 `sys_job` 注册）在修复后仍然全部未命中。
> 部署形态为单机单实例，故 H-01 只做构造提升，**未**引入 `FOR UPDATE SKIP LOCKED` 之类的行级认领。

### H-01 排水器的轮次互斥在生产中完全失效 —— `TryLock` 每轮拿到的是一把新锁

> **✅ FIXED — `c81890dc`**
> 排水器改为在 `Start()` 内 `AddJob` 之前构造一次、由 cron 闭包捕获同一实例；`s.getPool()`
> 的惰性初始化保留（拆出 `getPoolLocked()`，因为 `Start()` 全程持 `s.mu`，Go Mutex 不可重入，
> 直接调 `getPool()` 会自死锁 —— 该死锁在实现过程中被 `internal/scheduler` 测试挂死 8 分钟实测到）。
> `mu` / `DrainOnce` / `ad_sync_tasks.go` 三处注释已订正为"单实例互斥、当前无手动触发路径"。

**位置:**
- `internal/scheduler/ad_sync_tasks.go:198-205`（cron 闭包内 `NewADPushDrainer`）
- `internal/services/addomain/ad_push_queue_drain_service.go:80-82`（`mu` 的"事实源"声明）、`:113`（`TryLock`）
- `internal/services/addomain/ad_push_queue_drain_test.go:707-726`（锁住了一个生产中不成立的性质）

**确认方式:** 执行（临时探针，已删除）。

**失败场景（输入 → 错误行为）:**

`ad_sync_tasks.go:201` 把 `addomain.NewADPushDrainer(s.db, s.getPool())` 写在 **cron job 闭包体内**，每 30 秒新建一个 `ADPushDrainer`，因而每轮都拿到一把全新的 `sync.Mutex`。`ad_push_queue_drain_service.go:113` 的 `d.mu.TryLock()` 在自己的实例上永远成功。

代码与注释在三处明确宣称这把锁是可复用的事实源：

- `ad_push_queue_drain_service.go:80-82`：「mu 轮次互斥的**事实源**：cron 的 SkipIfStillRunning 互斥是闭包私有 channel，无法被手动触发路径复用，故此处自持」
- `ad_sync_tasks.go:178-179`：「SkipIfStillRunning 的互斥 channel 是**闭包私有**的……互斥的事实源是 ADPushDrainer.mu（TryLock）」
- `ad_push_queue_drain_service.go:110`：「cron 与手动触发共用本入口，互斥在内部统一处理」

**探针输出:**

```
--- PASS: TestZZReviewProbe_TwoDrainersIndependentMutex (0.06s)
```

探针逻辑：`d1.mu.Lock()`（模拟第一轮仍在跑）→ `d2.DrainOnce()` → 断言 `disableUserCalls == 1` 且行转 `pushed`。即**在 d1 持锁期间 d2 完整跑完了一轮排水并写了一次 AD**。两个实例之间零互斥。

**实际影响（不夸大）:** 当前唯一的生产调用方就是这条 cron，`cron.SkipIfStillRunning` 确实挡住了 cron 轮次重叠，所以**今天不会重复推送**。但：

1. 任何第二个调用方（手动触发端点、未来的 Phase 59 守卫、第二个调度器）都会拿到自己的锁，重复推送无从阻止。而 `ad_sync_tasks.go:178` 与 `drain_service.go:110` 的注释正在主动邀请后来者"复用同一互斥"。
2. 仓库中**不存在**任何手动触发路径（`grep -rni drain internal/api/` 零命中）。`DrainOnce` 注释里的"手动触发"目前是无对应实现的承诺。
3. 更实质的问题：`pickRows` **不做行级认领**（`drain_service.go:148-151` 的注释说"本结构体的 TryLock 已覆盖轮次重叠"）—— 这个前提在生产中不成立。两个实例并行时会对同一批行发两遍 LDAP 写。

**修复建议:** 把 drainer 提升为 `ADSyncScheduler` 的字段（与 `s.pool` 同级），在 `Start()` 里构造一次、cron 闭包只调 `d.DrainOnce(ctx)`。这同时让 `mu` 真的成为跨调用方的互斥。

---

### H-02 `TestPhase58_Drain_MutualExclusionSkipsSecondRound` 锁定了一个生产中不成立的性质

> **✅ FIXED — `c81890dc`**（随 H-01 同提交）
> 用例重写为三段：① 同一实例持锁期间 `DrainOnce` 必须跳过且 mock 计数不前进
> （`disableUserCalls` / `dnExistsCalls` 均为 0）；② 放锁后同一实例正常执行一轮
> （证明是"跳过"而非"卡死"）；③ 反向记录"另一个实例不被这把锁阻塞"——诚实的单实例语义，
> 不再暗示不存在的跨实例互斥。修 H-01 之后"同一实例"才是生产的真实形态，断言才成立。

**位置:** `internal/services/addomain/ad_push_queue_drain_test.go:707-726`

**确认方式:** 执行（H-01 探针）+ 阅读。

**问题:** 用例自己构造**一个** drainer `d`，手动 `d.mu.Lock()` 后调 `d.DrainOnce`，断言本轮被跳过。它证明的是「同一个实例上 TryLock 有效」——这没错，但它被写成 SC1 互斥性的验收证据，而 SC1 的真实互斥在生产中由 `SkipIfStillRunning` 而非这把锁提供，且生产中每轮是新实例。

**为什么按 HIGH 记:** 58-VERIFICATION.md 的 SC1 结论把该用例列为"互斥"分句的承载证据之一（:62-92 的表格）。这条 VERIFIED 结论实际建立在 H-01 之上 —— 即"互斥的事实源"这一设计主张未被证明，且反例可执行复现。

**修复建议:** 修 H-01 之后，把本用例改为断言"共享 drainer 时第二轮跳过"，并补一条针对共享实例的并发用例（两 goroutine 同时 `DrainOnce`，断言 LDAP 写只发生一次）。

---

### H-03 退避曲线的高两档（16min / 30min）在生产中不可达，实际重试窗口是 15 分钟而非 D-05 声明的 ~61 分钟

> **⏸ DEFERRED — `550f35e3`**（**不改逻辑**，仅在代码里记录事实与出处）
> 决定：等真实环境的账号池熔断数据到位后，再在"把 `maxDrainAttempts` 提到 8"与
> "改写 D-05 的 ~61min 表述"之间二选一。本次只在 `backoffDuration` 的文档注释与
> `maxDrainAttempts` 常量注释里写明"16min/30min 两档生产不可达、实际窗口 15min"，
> 并点明 `TestPhase58_BackoffDuration_Curve` / `_CappedAt30m` 作为纯函数断言测不出该断层。
> 该注释同时是 H-04 后果的上下文（队列会在熔断窗结束前就全部判死）。

**位置:**
- `internal/services/addomain/ad_push_queue_drain_service.go:44-62`（`backoffDuration`）、`:290-293`（判死条件）
- `.planning/phases/58-compare-and-move/58-CONTEXT.md:42`（D-05 原文：「60s→2min→4min→8min→16min 封顶 30min，attempts 达 5 转 dead——总重试窗 ~61min 刻意桥接账号池 30min 熔断周期」）

**确认方式:** 执行（临时探针，已删除）。

**失败场景（输入 → 错误行为）:**

`markFailed` 的判死条件是 `newAttempts >= maxDrainAttempts`（=5），因此 `backoffDuration` 只会被以 1/2/3/4 调用。

**探针输出:**

```
PROBE: 实际可达退避档位 = map[1m0s:true 2m0s:true 4m0s:true 8m0s:true]
```

即从第一次失败到判死，总窗口 = 60s + 2m + 4m + 8m = **15 分钟**。`backoffDuration(5)=16min` 与 `default` 的 30min 封顶在生产中**永不执行**。

**实际影响:** D-05 的立论是"退避窗要跨过账号池 30min 熔断周期，避免熔断窗内继续喂 MarkFailure"。实际只撑 15 分钟 —— 在熔断的 30 分钟里，队列已经全部判死、彻底停止重试。这与设计意图正好相反，且**失败得比预期更早更彻底**。

**顺带一个实质问题（与 H-04 耦合）:** `TestPhase58_BackoffDuration_Curve`（:307-314）直接对纯函数断言 `backoffDuration(5)==16min`。纯函数测试**不可能**发现这个断层（它测的函数本身是对的，错的是调用方永不传 5）。这就是一条"看起来绿、实际支撑的 VERIFIED 结论是空的"断言。

**修复建议:** 要么把 `maxDrainAttempts` 提到 8（覆盖 16m/30m 档，总窗 ~61min，与 D-05 一致），要么把 D-05 的"~61min"改写为"~15min"并重估熔断桥接结论。两选一，但必须显式决定；当前是实现与决策各说各话。

---

### H-04 账号全部拨号失败被当作可重试错误，15 分钟后整批判 dead（永久终态）

> **✅ FIXED — `18d6421e`**
> `ExecuteWithFailover` 用 `dialedAny` 区分"一个账号都没拨通"与"拨通但业务全失败"，前者包上
> 新的池侧哨兵 `ErrAllAccountsDialFailed`；排水器把它与 `ErrAllAccountsUnavailable` 同款处理
> 为池侧故障（不烧 attempts、不判 dead、纯退避）。
>
> **与本节"修复建议"的偏离（需知悉）**：原文建议复用 `ErrAllAccountsUnavailable`。实现时发现
> 复用会打破 Phase 78 的三条**同名**用例（`TestGrp78_AddMember_DialFail` /
> `TestSync78_SyncDataInternal_AllAccountsDialFail` / `TestUsr78_Update_AllAccountsDialFail`），
> 它们明确断言"拨号失败**不得**被当作 `ErrAllAccountsUnavailable`"，并把 group / user / sync
> 三条路径的用户可见文案锁定为"连接AD服务器失败"而非"账号池无可用账号"。合并两个哨兵 =
> 跨模块行为变更 + 削弱既有断言，违反"不得为让测试通过而弱化断言"的约束。
> 故改为新增独立哨兵，只在排水器一处同款处理，Phase 78 契约与全部既有用例零改动。
> 若要收敛为单一哨兵，需单独决策（代价：改 3 条 Phase 78 用例 + 3 个模块的错误文案）。

**位置:**
- `internal/services/addomain/failover_client.go:88`（`return fmt.Errorf("账号池 %d 个账号均失败: %w", maxAttempts, lastErr)`）
- `internal/services/addomain/ad_push_queue_drain_service.go:253-259`（`poolExhausted := errors.Is(err, ErrAllAccountsUnavailable)`）
- `internal/services/addomain/ad_push_queue_drain_service.go:290-293`

**确认方式:** 阅读。

**失败场景（输入 → 错误行为）:**

`ExecuteWithFailover` 有两条返回错误的路径，只有一条是 `ErrAllAccountsUnavailable`：

| 情形 | 返回值 | `errors.Is(…, ErrAllAccountsUnavailable)` | 后果 |
|------|--------|-------------|------|
| `ListAvailable` 返回空（池熔断） | 裸 `ErrAllAccountsUnavailable` | ✅ true | `poolExhausted=true` → 永不判死，可重试 ✅ |
| **所有账号 `Connect()` 失败** | `"账号池 N 个账号均失败: %w"` | ❌ **false** | `poolExhausted=false` → 退避重试，第 5 次 → **dead** ❌ |

于是：域控宕机 / 网络中断 / 全部服务账号被锁 / DNS 故障等**可恢复的基础设施故障**，会让整组（单轮最多 100 行）在约 15 分钟后全部转 `dead` 终态。`dead` 行退出活跃谓词、退出唯一索引、**不会被任何后续编辑复活**（除非用户再次被编辑），要恢复必须人工逐个重编辑用户。

叠加 H-03：账号池熔断是 3 次失败后 30 分钟窗口。拨号失败每轮都会调 `MarkFailure`（`failover_client.go:75`，前缀 `dial:`），所以第 3 轮就熔断；此时队列在第 5 轮（约 15 分钟）判死，**仍在 30 分钟熔断窗内**。设计里"退避窗桥接熔断周期"的机制被这个未识别的 `errors.Is` 缺口直接击穿。

**修复建议:** 把 `poolExhausted` 的判定从"错误是不是 `ErrAllAccountsUnavailable`"改成"闭包有没有跑起来"—— 闭包未执行（连接阶段全失败或池空）即视为池侧故障，可重试；闭包执行过才按行级结果分流。最小改法是在 `executeGroup` 里用一个 `closureRan bool` 标志。

---

### H-05 `MoveUser` 成功后本地 `ad_dn` 回写失败只 Warn 不上抛 → 下一次推送必定 code 32 并抹掉用户的 AD 绑定

> **⚠ PARTIAL FIXED — `3f55b7f7`**
>
> **已修复**：回写失败上抛（`return false, err`）→ `handlePushError` → `markFailed` + 退避，
> 行不再被写成 `pushed`；`last_error` 以 `retry:move_ou:` 暴露真实原因；错误注释已替换为
> 准确描述。新增 `TestPhase58_Drain_MoveWriteBackFailureIsNotPushed`（已验证 RED→GREEN）。
>
> **未修复（残余风险，已写进代码注释）**：**退避重试并不能自动修复绑定**。核查了
> `MoveUser` 的重放安全性，结论是**不安全**：
> - 重试时行会重读到**陈旧的** `ad_dn`，`syncUserAttributes` 先跑，其
>   `DNExists(陈旧 DN)` 返回 false → 清空 `ad_dn`/`ad_ou_dn`/`ad_synced_at` → 返回 nil →
>   `drainRow` 重读判空 → `markDead("unrecoverable:target-not-found")`；
> - 即使属性载荷为空（`syncUserAttributes` 在 `len(attributes)==0` 处早退）而走到
>   `MoveUser`，`ModifyDN` 打在已不存在的陈旧 DN 上会得到 code 32
>   （`ldap_client.go:350-361`，`deleteOldRDN=true` 先删旧 RDN）→ 同样清空三列 + dead。
> - `MoveUser` 对"对象已在目标 OU"本身是安全的（`ModifyDN` 同父改名，AD 视为 no-op），
>   但重试路径根本到不了那一步。
>
> 即：本修复让这次丢绑定**如实可见**（不再被 `pushed` 掩盖，Phase 59 的信号守卫也不会把它
> 读成"系统一致"），但没有让它可自愈。要自愈需要新的设计决策，例如在重放时先探测
> `RDN,targetOUDN` 是否已存在并只补写回 —— 这会给排水的热路径增加一次 LDAP 读，超出本次
> 四条 findings 的授权范围，故未实施。

**位置:** `internal/services/addomain/ad_push_queue_drain_service.go:456-468`（尤其 :463-465 的 `applogger.Warnf` 后 `return false, nil`）

**确认方式:** 阅读（路径已在 `user_ad_sync_service.go:197-215` 的注释与实现中确认）。

**失败场景（输入 → 错误行为）:**

1. 用户 `zhangsan` 当前在 `OU=IT`，部门映射到 `OU=IT2`，`ad_ou_dn='OU=IT,...'`。
2. 排水器 compare-and-move 判定不一致 → `MoveUser` 成功（AD 侧对象已搬到 `OU=IT2`）。
3. 紧接着的 `Updates(ad_dn, ad_ou_dn)` 因一次瞬时 DB 错误（连接池耗尽 / 语句超时）失败 → **只打 Warn**，`compareAndMoveOU` 返回 `(false, nil)`。
4. `drainRow` 落到 `markPushed`（`:388`）→ 队列行终态 `pushed`。
5. 本地 `sys_user.ad_dn` 仍是**旧的** `CN=zhangsan,OU=IT,DC=corp,DC=com`。

代码注释说「本地回写失败下轮会重算」（:465）——**这个缓解是错的**：队列行已经是 `pushed` 终态、退出活跃谓词，不会再被拾取，"下轮"根本不会发生。

真正的后果出现在下一次推送：用户再次被编辑 → 入队 → 排水 → `syncUserAttributes` 调 `DNExists(旧 DN)` → AD 侧已无此对象 → **清空 `ad_dn`/`ad_ou_dn`/`ad_synced_at`**（`user_ad_sync_service.go:206-213`）→ `drainRow` 重读判空 → `markDead('unrecoverable:target-not-found')`。**AD 账号明明还在（在新 OU 下），系统却永久丢失了绑定**，此后所有推送一律 code 32 → dead。一次瞬时写库失败被放大成用户 AD 资产的静默失联。

**修复建议:** 回写失败必须上抛（`return false, err`），让该行走 `markFailed` 退避重试。重放是安全的：`compareAndMoveOU` 的幂等点在下一轮会因为…（注意：重放时 `ad_ou_dn` 仍是旧值、`MoveUser` 会被再调一次 → 需要把幂等判据改成"以 `ad_dn` 的 OU 段为准"或在本轮 `MoveUser` 成功后先把内存中的 `user.AdOuDn` 置为 target 再判回写失败）。最低限度：不要在回写失败时写 `pushed`。

---

## 2. MEDIUM

### M-01 `FindOUDNByDeptIDForConfig` 的任何错误都被当作"不可恢复"→ 一次瞬时 DB 故障把行永久打死

**位置:** `internal/services/addomain/ad_push_queue_drain_service.go:428-436`；错误来源 `internal/services/addomain/dept_ou_mapper.go:80-85`

**确认方式:** 阅读。

**失败场景:** `FindOUDNByDeptIDForConfig` 用 `fmt.Errorf` 包了两种错误：`ErrRecordNotFound`（无映射行）与 `%w` 包的真实 DB 错误（连接失败 / 超时 / 表锁）。`compareAndMoveOU` 只看 `mapErr != nil`，**不区分**，一律 `markDead('unrecoverable:no-ou-mapping')`。

对照：`groupByResolvedConfig:186-193` 对同一个"读用户失败"是显式区分的（ErrRecordNotFound → dead，其它 → 跳过本轮不烧 attempts）。两处策略不一致，说明这是疏漏而非设计。

**修复建议:** 让 mapper 返回哨兵错误（如 `ErrOUNotMapped`）或在调用侧用 `strings.Contains`/自定义类型区分"无映射"与"读库失败"，后者走 `markFailed`。

### M-02 清空/置空字段无法传播到 AD，但队列行被标记 `pushed`

**位置:** `internal/services/addomain/ad_push_queue_drain_service.go:355-372` + `:476-491`（`buildDrainUpdateReq`）；被丢弃处 `internal/services/addomain/user_ad_sync_service.go:157-160`

**确认方式:** 阅读。

**失败场景:** 用户昵称从 `张三` 改成空串。
- 入队侧 `adFieldDiffers`（`ad_push_queue_service.go:84-93`）：`nil` 视为无变更，但 `""` 与旧值不等 → **入队**。
- 排水器 `buildDrainUpdateReq`：`Nickname != nil` → 写入 `updateReq["nickname"] = ""`。
- `syncUserAttributes:157-160`：`nickStr != ""` 不成立 → 该属性**不进 payload**；若四个字段全空则在 `:184-187` 提前 `return nil`，**一次 LDAP 写都不发**。
- `drainRow` 继续走 compare-and-move（可能 MoveUser），然后 `markPushed`。

结果：队列显示 `pushed`（Phase 59 的 GUARD-01 信号守卫会把"无 pending/failed 行"读成"系统一致"），但 AD 上的 `displayName` 永远是 `张三`。四字段（email/phone 同理）都有这个洞。

**修复建议:** `buildDrainUpdateReq` 对四个字段**始终**写入（nil 视作空串），让 `syncUserAttributes` 用 `Replace(attr, []string{""})` 清空 AD 侧属性。这不涉及任何新旧值比较，不触碰审计否决项。注意这会改变"四字段全空"的早退分支，需要同时确认 `Replace` 空值在 AD 侧的行为。

### M-03 `markPushed` / `markFailed` / `markDead` 的返回错误被全部丢弃，零日志

**位置:** `internal/services/addomain/ad_push_queue_drain_service.go:137`、`:353`、`:388`、`:333`、`:341`、`:368`、`:391`、`:434`

**确认方式:** 阅读。

**失败场景:** 终态转换是整个 phase 的可观测性承诺（QUEUE-03「终态留表可查」、SC1「可通过查询确认」）。若 `Updates` 因 DB 故障失败，行会静默留在 `pending`/`failed`：`pushed` 失败 → 下一轮重复推送（幂等，可接受）；`markDead` 失败 → 行继续重试直到 DB 恢复（可接受）；但**全程没有任何一条日志**，排障时只能靠"AD 明明成功了、队列行却还是 pending"这种间接现象反推。

**修复建议:** 三处调用点改为 `if err := d.markX(...); err != nil { applogger.Errorf(...) }`。

### M-04 `TestPhase58_LoadDrainIntervalSeconds_FallsBackOnGarbage` 的 flake 是本 phase 自己引入的，且有确定修法

**位置:** `internal/services/addomain/ad_push_queue_drain_test.go:1041-1043`

**确认方式:** 执行。

**失败场景:** DSN 为 `file:<t.Name()>?mode=memory&cache=shared`。`cache=shared` 的内存库在同一进程内跨重复轮次存活，第二次 `AutoMigrate` 后 `sys_config` 里已存在上次插入的 `config_key`，`INSERT` 撞唯一约束。

**实测输出（`-count=2`，第二轮失败）:**

```
--- PASS: TestPhase58_LoadDrainIntervalSeconds_FallsBackOnGarbage (0.11s)   ← 第 1 轮
constraint failed: UNIQUE constraint failed: sys_config.config_key (2067)   ← 第 2 轮 4 个子用例全红
```

**为什么仍要记:** 58-VERIFICATION.md:424-432 把它归因为"既有的共享缓存 DSN 问题……该测试由 Plan 05 交付、**本 plan 未触碰**……属既有问题，**本 plan 不修**（超出 58 范围）"。但该测试文件本身就是 Phase 58 的 Plan 05 产物、`freshDrainSysConfigDB` 也是 Phase 58 新写的 helper —— 这是**本 phase 引入的缺陷被误归因为遗留**。按审查要求不作为回归报告，但确定存在可修根因。

**修复建议:** DSN 加每轮唯一后缀（如 `t.Name()+"-"+uuid.NewString()`），或去掉 `cache=shared`。

### M-05 终态行无任何清理策略，且入队侧缺 `ad_dn` 非空预过滤 → `dead` 行无界增长

**位置:** `internal/models/ad_push_queue.go:59-60`（队列表不软删、无保留策略）；`internal/services/addomain/ad_push_queue_service.go:201-215`、`:357-394`（两个入队入口均不预过滤 `ad_dn`）

**确认方式:** 阅读。

**失败场景:** 本地账号（从未 AD 同步、`ad_dn` 为 NULL）每次被编辑都会入队一行 `update_user`，排水器在 `groupByResolvedConfig:196-200` 判 `no-ad-dn` → `dead`。每次导入也会为 `ad_dn` 为空的行入队（D-01 决策里写的"`ad_dn` 非空预过滤"实际落在调用方 `user_import_handler.go:132-136`，不在 `EnqueueUpdateBatch` 里）。队列表不软删、无 TTL、无 purge → 每次编辑/导入都在表里留一行终态记录。

D-01 明确要求批量入队变体带 `ad_dn` 非空预过滤，实现把它放在了唯一调用方。功能等价，但契约与实现位置不一致，且下一个调用方很容易踩空。

**修复建议:** 把 `ad_dn IS NOT NULL AND ad_dn <> ''` 下沉进 `EnqueueUpdateBatch`（调用方的过滤保留亦可，双保险）；另加一个定期归档终态行的 cron 或运维 SQL（可留 Phase 59）。

---

## 3. LOW

| # | 位置 | 问题 |
|---|------|------|
| L-01 | `ad_push_queue_drain_service.go:152-161` | `pickRows` 不过滤 `entity_type`。当前只有 `user` 一种实体，但 `models/ad_push_queue.go:18` 已预留扩展位；将来加 dept 行时会被当成 user 去 `loadUserUnscoped` 查 `sys_user` → 全部误判 `user-not-found` → dead。建议 `Where("entity_type = ?", ADPushEntityUser)`。 |
| L-02 | `ad_push_queue_drain_service.go:231-238` | `executeGroup` 重读配置用 `First` 无 `status` 条件。分组与执行之间配置被停用的窗口内仍会推送（秒级窗口，影响可忽略），但配置被**软删**时 `First` 返回 not found → 整组 `unrecoverable:no-ad-config` 永久打死。 |
| L-03 | `ad_push_queue_drain_service.go:566-574` + `ad_sync_tasks.go:186-196` | `SecondsToCronSpec(1)` 返回 `"*/1 * * * * *"`，即每秒一轮。`sys_config` 是管理员可编辑的（`ConfigTypeNo`），误填 `1` 会让排水器每秒查一次库并在队列非空时高频打 AD。无下限保护，建议 `< 5` 视为非法回落默认。 |
| L-04 | `ad_push_queue_drain_service.go:128-143` vs `:230-261` | 分组阶段（`groupByResolvedConfig`）用分组时读到的 `ad_dn` 解析配置，`drainRow` 再重读一次用户。两次读之间若用户被跨域移动，按旧配置拨号会打错域控。窗口极窄，仅记录。 |
| L-05 | `internal/api/v1/system/user_import_handler.go:128-131` | `enqueueImportUsersToAD` 用 `context.Background()`，未绑定 `c.Request.Context()`。客户端断连后入队仍会跑完（几千行的 `IN` 查询 + 批量 upsert）。仓库内该函数已删除的旧版本也是 `context.Background()`，非本 phase 新增的偏差。 |
| L-06 | `ad_push_queue_service.go:161-169` | D-06 合并子句重置了 `status/attempts/last_error/next_retry_at/failed_at/created_at/updated_at`，唯独没重置 `pushed_at`。当前不可达（`pushed` 行不在唯一索引内，永远不会被合并命中），属防御性缺口。 |
| L-07 | `user_ad_sync_service.go:74-80` | `SyncUserUpdateToAD` 里的 `newDeptIDStr != currentDeptID` 是**典型的新旧值比较**。该方法已被 SC6 声明"无生产调用方"，因此**不构成审计否决项的命中**（否决项约束的是推送路径）。但注释说"保留作显式管理面 / 未来复用"——若将来被 Phase 60 SAFE-01 之类重新接线，会把已判死刑的仲裁形态带回来。建议在那行加显式禁接线标记。 |

---

## 4. 针对重点问题的正面结论（不是发现，但审查范围内）

1. **`terminal` 短路是正确的。** `compareAndMoveOU:432-435` 写完 `dead` 后返回 `(true, nil)`，`drainRow:385-387` 见到 `terminal` 立即 `return`，不会落到 `markPushed`。另一条写终态的路径（`:366-370` 的 `target-not-found`）自带 `return`。两条都无覆盖漏洞。
2. **部分唯一索引的交互是正确的。** `pushed`/`dead` 行退出 `uq_ad_push_queue_active`，再次编辑会插入并行行而非撞 42P10 —— 这是 Phase 57 D-06 的正确设计，58-07 那次 flaky 的根因判定准确、生产语义无需改动。拾取谓词 `status IN ('pending','failed') AND (next_retry_at IS NULL OR next_retry_at <= now)` 既不会多捞也不会漏捞。
3. **F-07 `BatchDelete` 的 Pluck 顺序正确。** `user_service.go:650-660` 的 Pluck 在 `Delete` 之前且用默认软删作用域，恰好等于"本轮将被软删的那批行"；注释里禁用 `Unscoped()` 的理由（会把历史已软删行算进来，破坏可断言口径）也成立。
4. **`ldap.IsErrorWithCode` 确实穿透 `%w` 包装** —— `TestPhase58_Drain_MoveCode32GoesDead`（:942-963）真跑通了 OU 路径的 code 32 → dead + 清 `ad_dn`，不是假绿。
5. **`TestPhase58_Drain_NoBareAttemptsDeadAfterMerge`（:601-617）不是套套逻辑。** 它铺 `attempts=5, status=failed` 并让本轮成功，断言转 `pushed`；若有人在拾取阶段加裸 `attempts>=5 → dead` 判定，用例必红。F-11 禁令的验收是实的。

---

## 5. D-02 路由与 DN 归一化的对抗性推演（未发现缺陷）

**(a) 逗号边界能否被"子串但非分量边界"的后缀骗过？** 不能。`ad_config_resolver.go:60` 判定为 `strings.HasSuffix(norm, ","+base)`。任何长度 L 的后缀由字符串末尾 L 个字符唯一确定，因此若 `",X"` 与 `",Y"` 都是 `norm` 的后缀且等长，则 X ≡ Y——这同时意味着下面的 (b)。

**(b) 等长歧义是否被正确识别而非任意取一条？** 是。`tiedBaseDNs` 只在归一后 `base_dn` **完全相同**时才累积到 >1（即两条启用配置指向同一域的误配置），此时返回 error → 队列行 `unrecoverable:no-ad-config` 转 dead，符合 D-02"多匹配 → dead 不推"的契约。测试 `AmbiguousEqualsLength`（:82-91）用的正是这个形态，且 `tiedBaseDNs` 的三段式 switch（`>` 重置 / `==` 累积 / `<` 忽略）在长度序列 10,10,12 与 12,10,10 两种排列下都正确。

**(c) 归一是否两侧一致？** 是。`norm := NormalizeDN(dn)`（:40）与 `base := NormalizeDN(cfg.BaseDN)`（:55）走同一函数。

**尝试过的错误路由构造（均失败）:**
- `CN=a,DC=xcorp,DC=com` vs base `dc=corp,dc=com` → 无命中 ✅（测试已锁）
- 子域 `CN=u,DC=na,DC=corp,DC=com` vs 父/子双配置 → 命中更长的子域 ✅
- base_dn 带首尾空格与全大写 → 归一后命中 ✅
- base_dn 为空串 / 全空格 → 显式跳过 ✅（否则空串后缀恒命中）

**DN 归一化的潜在误判（评估后不构成发现）:**
- **值侧 `ToLower` 会不会把两个真 DN 判成同一个（漏搬）?** 对 AD 的 `CN/OU/DC` 三种属性而言其 matchingRule 本身就是 caseIgnoreMatch，值侧小写化与 LDAP 语义一致，不会引入假阳性。
- **RFC4514 转义序列会不会被破坏?** `\,` `\+` `\=` 等在 `NormalizeDN` 的按逗号切分中被原样保留（切分点只在**未转义**的顶层逗号上，因为转义逗号前有反斜杠但函数不看——见下条），十六进制转义 `\AB` → `\ab` 语义等价。
- **但这里确有一个真实缺口:** `NormalizeDN:42` 的 `strings.Split(trimmed, ",")` 不识别转义逗号。对 `CN=Zhang\,San,OU=IT,...` 它切成 3 段再原样拼回，结果字符串与输入一致 → **归一化仍然幂等、比较仍然自洽**，因此不会误判。真正出问题的是 `ldap_client.go:430-436` 的 `extractRDNFromDN`（`dn[:strings.Index(dn, ",")]`），见下条。

---

### M-06（升级自 LOW 的执行确认）`extractRDNFromDN` 遇 RFC4514 转义逗号会截断 RDN

**位置:** `internal/services/addomain/ldap_client.go:430-436`；调用点 `ad_push_queue_drain_service.go:452`（`MoveUser`）、`:461`（回写 `ad_dn`）、`user_ad_sync_service.go:139`、`:332`

**确认方式:** 执行（临时探针，已删除）。

**探针输出:**

```
extractRDNFromDN("CN=Zhang\,San,OU=IT,DC=corp,DC=com") = "CN=Zhang\"
```

**失败场景（输入 → 错误行为）:** 用户的 CN 含逗号时，AD 侧 DN 为 `CN=Zhang\,San,OU=IT,DC=corp,DC=com`（AD 会按 RFC4514 转义返回）。排水器判定 OU 不一致 → `MoveUser` 把截断后的 `CN=Zhang\` 作为 `newRelativeDN` 提交给 `ModifyDN(deleteOldRDN=true)` → **AD 把用户重命名为 `Zhang\` 并删除原条目**。同时本地 `newAdDn` 被写成 `CN=Zhang\,OU=IT2,...`（同样损坏）。

**严重度定在 MEDIUM 而非 HIGH 的理由:** `extractRDNFromDN` 与 `MoveUser` 是 Phase 58 之前就存在的代码，本 phase 未改动它们。触发需要 CN 含逗号——而 CN 在本系统里通常来自 username，RuoYi 用户名不允许逗号，故实际可触发性低。但 Phase 58 把它变成了**自动每 30 秒跑一次的全量路径**，且本 phase 刚刚为 OU DN 引入了 `BuildOUDN` 转义（HYGIENE-02）却没给 RDN 侧同等处理，形成新的不一致。

**修复建议:** `extractRDNFromDN` 改为按"未转义逗号"切分（跳过 `\` 转义状态与 `\XX` 十六进制对），或直接用 `ldap.ParseDN` 取 `RelativeDN`。注意该函数被 `ldap_client.go` 多处复用，改动需跑回归。

---

## 6. 审计否决项独立裁定

我按代码逐行核对，**不采信 58-VERIFICATION.md 的自证**。结论如下：

| 否决项 | 裁定 | 行证据 |
|--------|------|--------|
| **禁时间戳仲裁**（不拿 `updated_at`/`created_at`/`ad_synced_at` 决定推/跳） | ✅ **未命中** | `ad_push_queue_drain_service.go` 全文对这三个列的引用只有 4 处：`:157` `Order("created_at ASC")`（**排序**，决定同批内的先后，不决定某行推不推）、`:274/:301/:320` `"updated_at": now`（写入审计列）、`:495/:502/:523` `ad_synced_at` 仅被**清空**。`compareAndMoveOU` 的决策输入只有 `user.DeptID`、`user.AdOuDn`、`user.AdDn` 与 mapper 返回的 `targetOUDN`，无任何时间戳。`markFailed:290` 的判死依据是 `attempts` 计数，非时间戳。 |
| **禁新旧值比较**（闭包重读最新值 vs 目标，不做 old-vs-new diff） | ✅ **未命中** | `drainRow:329` `loadUserUnscoped` 闭包内重读；`:357` `buildDrainUpdateReq(user)` 只取重读值构造载荷，全函数无旧值变量。`compareAndMoveOU:442` 的判据是 `NormalizeDN(*user.AdOuDn) == NormalizeDN(targetOUDN)` —— **当前值 vs 目标值**。全仓无 `oldUserSnapshot`/`before`/`prev` 类变量流入排水器（`user_service.go:163` 的 `oldUserSnapshot` 只服务 Phase 57 的入队 diff，不参与推送决策）。唯一的 old-vs-new 比较在 `user_ad_sync_service.go:80`，属已下线方法的死代码（见 L-07），不构成推送路径命中。 |
| **禁 `sys_job` 注册** | ✅ **未命中** | `internal/scheduler/ad_sync_tasks.go` 全文只有 2 处 `sys_job`，均为**注释**（`:174`「禁注册 sys_job」、drainer 文件 `:4`）。无 seed、无 `RegisterTask`、无 `sys_job_log` 写入。排水器只经 `s.cron.AddJob`（`:198`）注册进 in-process robfig/cron。 |
| **禁 JSON 快照** | ✅ **未命中** | `internal/models/ad_push_queue.go:66-81` 队列表 9 个字段全是标量（id / entity_type / entity_id / action_type / status / attempts / next_retry_at / last_error / 时间戳），无任何 payload/snapshot/data 列。migration_215 与 migration_216 建的也只有唯一索引与 sys_config 行，不含快照列。文件里的 `json:` tag 是 HTTP 响应序列化，与存储快照无关。 |
| **禁删 AD 账号** | ✅ **未命中** | `internal/services/addomain/` 内无 `DeleteUser`、无 `conn.Delete(` 调用。disable 路径走 `LDAPClient.DisableUser`（`ldap_client.go:337`）→ `setUserAccountControl(uacDisabledAccount)`，即 UAC 514 Replace，幂等可重放。 |

---

## 7. 测试质量专项（重点看最后两个提交）

65 个 `TestPhase58_*` 中，绝大多数是有实质断言的真测试（铺真实 sqlite 行、走生产符号、mock 计数 + DB 状态双侧断言）。发现的问题：

1. **H-02**：`TestPhase58_Drain_MutualExclusionSkipsSecondRound` 在单实例上成立、在生产的每轮新建实例下不成立 —— 这是最接近"套套逻辑"的一条，因为它承载的是 SC1 的互斥分句。
2. **H-03**：`TestPhase58_BackoffDuration_Curve` / `_CappedAt30m` 直接测纯函数，对 `backoffDuration(5)` 与封顶档的断言**在生产中永不触发**。纯函数绿 ≠ 退避契约成立。这两条测试没有把"哪一档在什么 attempts 下被调用"与 `maxDrainAttempts` 关联起来，是本 phase 最实质的测试盲区。
3. **58-07 的 flaky 重写（`30af6251`）本身是正确且诚实的**，但产生了一处**文档与代码脱节**：重写把第二轮从"走生产入队器"改成了裸 SQL 行重置，而 58-VERIFICATION.md:145 仍写着「端到端第二轮走的是**生产入队路径**（`EnqueueUpdateBatch` 的 D-06 合并子句把 `pushed` 终态行打回 `pending` 并清零 `attempts`）……这是真实生产形状「用户又被编辑一次」」。该表述在重写后已不成立，且它描述的合并语义（把 `pushed` 行打回 pending）**本来就是错的**——`pushed` 行不在唯一索引内，合并不会命中（这正是 flaky 的根因）。VERIFICATION 未同步修正。（MEDIUM，建议改文档。）
   - 重写后新增的 `countQueueRowsForEntity == 1` 断言是**有效加固**：它把"同一行被重复排水"从隐含前提变成了可证伪断言，去掉了 `reloadDrainRow` 的 `First` 无序歧义。`-count=10` 确定性通过这一说法与我的 `-count=2` 复现一致（该用例现已稳定）。
4. **M-04**：`freshDrainSysConfigDB` 的 `cache=shared` DSN 是本 phase 引入的确定性缺陷，`-count=2` 即可复现，被误归因为"既有"。
5. **无 `assert.True(t, true)` 类空断言**：`grep` 全仓 Phase 58 测试零命中（`user_handler_test.go:42` 那条失效注释已在 `f8f88d4e` 清掉）。

---

## 8. 对已知的三个 flake 的说明

按要求不作为回归报告，补充两点：

- **`TestPhase58_LoadDrainIntervalSeconds_FallsBackOnGarbage`**：确有可修根因，见 M-04（`-count=2` 复现，唯一约束冲突，确定性而非负载相关）。
- **`TestInit78_Close_PartialInit`**（30s 墙钟守卫）：与 Phase 58 无调用关系，是 `internal/core` 的全量并行负载抖动。**无可修根因**（属于测试自身超时预算与全量并行度的取舍），不建议在本 phase 处理。
- **`TestAuth8003_LoginLocalDirect_WrongPassword`**：同上，无可归因的 Phase 58 因素。

---

## 9. 建议的处理顺序

1. **H-01**（drainer 提为 scheduler 字段）—— 一处改动，同时修好并发与 H-02 的前提。
2. **H-04**（`poolExhausted` 改为按"闭包是否执行"判定）—— 与 H-03 同源，一起决定 attempts 上限。
3. **H-03**（明确 attempts 上限是 5 还是 8，并同步 D-05 表述）—— 纯决策 + 一行常量。
4. **H-05**（回写失败上抛）—— 影响 compare-and-move 的重放判据，需与 M-02 一并设计。
5. M-01 / M-02 / M-03 → M-04 / M-05 → LOW 批量。
6. 文档：修正 58-VERIFICATION.md:145 的失实表述（M-04 同段）。

---

*Phase: 58 · Advisory Review · 生成于 2026-10-19*
*本审查未修改任何生产代码；执行验证用的临时探针文件（`zz_review_probe_test.go`）已在验证后删除，`git status` 确认工作区无本次审查引入的改动。*
