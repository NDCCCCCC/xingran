---
slug: ad-push-queue-drain-interval-not-hot-reloaded
status: fixed
session_type: bug
trigger: "verify-work 57 Test 1 期间发现：修改 sys_config 的 ad_push_queue.drain_interval_seconds 后不生效，必须重启后端才生效"
created: 2026-10-21T12:05:00+08:00
updated: 2026-10-21T14:10:00+08:00
deferred_to: （已完成）quick 261021-iqq，方案A落地：固定 30s tick + drainGate 门控 + 缺陷X sanitize；待提交
component: internal/services/addomain/ad_push_queue_drain_service.go
phase: 57
related:
  - internal/scheduler/ad_sync_tasks.go
  - internal/core/db/migrations/migration_216_adpush_drain_sysconfig.go
---

# Debug Session: ad_push_queue.drain_interval_seconds 不热重载（改 sys_config 必须重启后端）

## 关键结论（用户首读这一段）

`ad_push_queue.drain_interval_seconds` 这个 sys_config 旋钮**只在后端启动时读一次**，
读到的值被固化成一条固定 cron spec 注册进 scheduler；之后 sys_config 里的值再怎么改，
排水器的节奏都不会变 —— 必须重启后端才生效。

这与项目"sys_config 是运行时热重载配置"的既定设计相悖，也与仓库内既有的热重载先例
（`internal/api/v1/system/config_handler.go:297`，更新验证码相关配置后立即 reload）
不一致，属于该旋钮的实现特例而非全局约定。

**混淆成本已经真实发生**：本次 verify-work 57 Test 1 中，用户把该值改成 300 后在 UI
连改两次仍看到 2 行队列，差点误判为"ON CONFLICT 合并路径失效"；真实原因是排水器仍按
30s 刻度把首行推成了终态（终态行退出部分唯一索引属设计行为）。

**修复时机**：按用户指示，等 verify-work 57 的三项人工测试跑完后再动手。

---

## Symptoms

### Actual Behavior

- `UPDATE sys_config SET config_value='300' WHERE config_key='ad_push_queue.drain_interval_seconds'`
  执行后，排水器仍按旧间隔（默认 30s）运行；队列行的 `pushed_at` 依旧落在每 30 秒的整点刻度。
- 只有**重启后端**后，新值才被读取并映射为 cron spec（300 → `0 */5 * * * *`）。

### Reproduction

1. 后端运行中，`UPDATE sys_config ... config_value='300'`（该 key 真实存在，默认 30）
2. 触发一次入队并等待排水，观察 `pushed_at` 刻度 → 仍是 30s 节奏
3. 重启后端 → 刻度变为 5 分钟

### Error Messages

无报错。属于**静默不生效**，这是本缺陷比普通 bug 更危险的地方。

---

## Root Cause

读取点唯一（无缓存层，直接查库）：

- `internal/services/addomain/ad_push_queue_drain_service.go:600`
  `const drainIntervalSysConfigKey = "ad_push_queue.drain_interval_seconds"`
- `internal/services/addomain/ad_push_queue_drain_service.go:608`
  `func LoadDrainIntervalSeconds(db *gorm.DB) int` —— 读不到/解析失败/非正数 → 回落 30

消费点唯一（在**进程启动**路径上读一次）：

- `internal/scheduler/ad_sync_tasks.go:193-199`
  `drainInterval := addomain.LoadDrainIntervalSeconds(s.db)` →
  `spec, ok := addomain.SecondsToCronSpec(drainInterval)`（非法值 → 回落 30s 并告警）→
  用 robfig/cron 注册**固定 spec**

之后没有任何回调/监听在 sys_config 变更时重新注册 cron；robfig/cron 本身也不支持运行时
替换已注册 job 的 spec，于是这个值事实上被"冻结"在启动时刻。

对照（项目内既有热重载先例，说明本旋钮是特例）：

- `internal/api/v1/system/config_handler.go:297` — 更新验证码相关配置后立即重新加载
- 旋钮的来源迁移 `internal/core/db/migrations/migration_216_adpush_drain_sysconfig.go`
  也明确"缺旋钮只会回落默认 30s"，即它被设计成启动期参数

---

## 影响

- 运维改旋钮不生效且无任何提示 → 误判（本次已发生一次）
- UAT 无法在不停机的前提下调整排水节奏：Phase 57 Test 1 的字面级 UI 双击验证被迫要求重启
- 与 Phase 58/59 运维口径不一致（若文档按"改 sys_config 即生效"编写，会形成错误预期）

---

## 修复方案（待 verify-work 57 完成后执行）

### 方案 A（推荐 —— 真正的运行时热重载）

- 排水 cron 固定为最小轮询刻度（如每 5s / 10s），每个 tick 内读取当前
  `drain_interval_seconds`（走短 TTL 缓存或直接查库；`sys_config` 有唯一索引
  `idx_sys_config_config_key`，单行查询成本极低），按"距上次实际排水是否已达到当前间隔"
  决定本 tick 是否处理一批。
- 收益：改 sys_config 后下一个 tick 即生效，**无需重启**；不依赖动态重注册 cron。
- 需决策：`SecondsToCronSpec` 的合法值域限制（1..59 且 60%s==0，或 60..3540 的 60 倍数；
  3600 会判 ok=false 回落 30s）在此方案下是保留在读取层还是取消。
- 风险：tick 频率提高 → 每 tick 一次配置查询（加 1s TTL 缓存）+ 一次取数；无 pending 行时
  必须快速返回。

### 方案 B（保留 cron，动态重注册）

在配置更新钩子里向 scheduler 暴露 `RescheduleDrain(spec)`，变更时重注册该 job。
改动面更大（scheduler 层新增 API + 配置更新链路打通），但保持原有 cron 语义。

### 方案 C（文档化"需重启"语义）

成本最低，但与"sys_config 全部热重载"的设计意图不符，只能作为过渡方案。

### 决策

待用户确认采用 A / B / C（默认推荐 A）。

---

## 验收标准

1. 运行中修改 `ad_push_queue.drain_interval_seconds`，**不重启后端**，观察 `pushed_at`
   刻度在一个旧间隔 + 一个新间隔内切换到新节奏
2. 非法值（0 / 负数 / 非 60 倍数）按约定回落并有告警日志，不 panic
3. 回归：`internal/core/db/migrations/migration_216_adpush_drain_sysconfig_test.go` 及
   ad_push_queue 相关既有用例全部通过；新增"运行时改配置无需重启"用例
4. Phase 57 Test 1 的字面级 UI 双击验证可在**不重启后端**的前提下完成

---

## 关联

- Phase 57 Test 1（QUEUE-02 合并路径）/ Phase 58 D-05 旋钮
- `internal/services/addomain/ad_push_queue_drain_service.go:598-640`（旋钮读取 + cron spec 映射）
- `internal/scheduler/ad_sync_tasks.go:182-210`（cron 注册，仓库无手动触发端点）

---

## 追加发现（2026-10-21 12:15 verify-work 57 Test 2 实测时撞出）

Test 2 重启后（12:11:42 启动，日志确认 `drain_interval_seconds = 300s` 生效、cron
`0 */5 * * * *` 注册成功），12:15:00 刻度排水器拾取了 4b743635 行并尝试更新 AD 用户
chenchao-076，但暴露两个**新缺陷**（与热重载无关，独立记录）：

### 缺陷 X：markFailed 写 last_error 未清洗 NUL → 失败状态都落不了库

- 症状：日志 [GORM错误] `UPDATE ad_push_queue SET attempts=1 ... last_error='retry:update_user:更新AD用户属性失败: LDAP Result Code 50 "Insufficient Access Rights": 00002098: SecErr: DSID-031514A2, problem 4003 (INSUFF_ACCESS_RIGHTS), data 0\n\u0000' ... 错误: ERROR: invalid byte sequence for encoding "UTF8": 0x00 (SQLSTATE 22021)`
- 后果：markFailed 的 UPDATE 整体失败 → 行**留在 pending / attempts=0 / next_retry_at=NULL**
  → 下个 tick pickRows（`status IN ('pending','failed') AND (next_retry_at IS NULL OR next_retry_at <= now)`）
  又拾取 → 永久空转循环，attempts 永远是 0，退避机制完全失效，maxDrainAttempts 判死永不可达。
- 根因：drain 的失败路径把含 NUL 的 LDAP 错误直接写 last_error：
  - handlePushError `ad_push_queue_drain_service.go:581` `"retry:"+action+":"+err.Error()`
  - 同类点 :390（读取用户失败）、:577（pool:+err.Error()）
  - 而仓库已有现成清洗器 `account_pool.go:359 sanitizeFailureReason`
    （ToValidUTF8 + 去 0x00 + 截 1000），account_pool 的 MarkFailure :382 已在用，
    drain 侧漏接。
- 修复：markFailed/markDead 入口对 msg 统一过 sanitizeFailureReason（或在三处调用点清洗），
  加回归用例：构造含 \u0000 的 err，断言 UPDATE 成功且 last_error 无 NUL。

### 缺陷 Y：AD-sync 服务账号对用户属性无写权限（环境侧）

- `update_user` 对 chenchao-076 改属性 → LDAP Result Code 50 Insufficient Access Rights
  （00002098, problem 4003），说明账号池当前用的 AD-sync 服务账号在目标 OU 上没有
  写 user 属性的权限。属 AD 管理员侧授权问题，代码无法修。
- 影响：所有 update_user 类队列行在该账号拿到写权限前都会失败（修复 X 后会正常进入
  attempts 退避，attempts 到 5 判 dead）。
- 待用户决策：给 AD-sync 账号授权对应 OU 的写权限，或换有权限的服务账号。

### 与 Test 2 的关系

Test 2 的核心断言（kill -9 → 重启 → pending 行不丢）**已经独立成立**：行在 kill 与重启
之间保持 pending、重启后 attempts=0、字段零漂移。12:15 tick 的失败是**排水阶段的另一个
缺陷**（X），不回溯到持久化。Test 2 可以按证据裁决；缺陷 X/Y 转入修复清单。
