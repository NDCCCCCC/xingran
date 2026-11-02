---
phase: quick-261021-iqq
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - internal/scheduler/ad_sync_tasks.go
  - internal/scheduler/ad_sync_tasks_drain_gate_test.go
  - internal/services/addomain/ad_push_queue_drain_service.go
  - internal/services/addomain/ad_push_queue_drain_test.go
  - internal/core/db/migrations/migration_216_adpush_drain_sysconfig.go
autonomous: true
requirements:
  - DRAIN-HOTRELOAD-01
  - DRAIN-GUARD-01
  - DEFECT-X-SANITIZE-01
must_haves:
  truths:
    - "sys_config 修改 ad_push_queue.drain_interval_seconds 后无需重启：下一个 30s tick 读取新值并打『值变化』日志，之后按新间隔排水（多实例各自读 DB 自动收敛，Test 3 实锤 10.62.10.39/10.62.10.33 双实例）"
    - "cron 固定 */30 最小 tick + lastDrain 门控：now-lastDrain >= interval 才 DrainOnce；interval=30（默认）行为与改前逐刻度一致；interval=300 时有效间隔 300~330s（±30s 抖动，用户已接受）"
    - "interval < 30s（如 15）时有效间隔被 tick 下限托底为 30s；非 60 因子值（45/90/100）不再回落而是直接生效（SecondsToCronSpec 映射层整体退役）"
    - "web 定时任务页面永远删不掉排水器：守护测试锁死 RegisterADSyncTasks/RegisterADAccountPoolTasks 后 taskRegistry 中不存在任何 drain/ad_push_queue 键（D-05『禁注册 sys_job』的回归防线）"
    - "缺陷 X 修复：markFailed/markDead 写 last_error 前统一过 sanitizeFailureReason（ToValidUTF8 + 去 0x00 + 截 1000），LDAP Result Code 50 的 data 0\n\0 不再触发 PG SQLSTATE 22021、不再出现 pending 行 attempts=0 无限重试循环"
    - "既有 Phase 58 排水器测试全绿（go test ./internal/services/addomain/ -run TestPhase58），新增门控/sanitize/守护测试全绿"
  artifacts:
    - "internal/scheduler/ad_sync_tasks.go：drainGate 门控结构（lastDrain/lastInterval + shouldDrain 纯时间逻辑），AddJob 固定 */30 tick 闭包改为每 tick 读间隔→门控→DrainOnce，配置变化 Infof 日志"
    - "internal/scheduler/ad_sync_tasks_drain_gate_test.go（新）：门控零值立即排水 / 未到期跳过 / 到期放行 / 间隔变化检测 + taskRegistry 防删守护"
    - "internal/services/addomain/ad_push_queue_drain_service.go：markFailed/markDead 入口 sanitize；LoadDrainIntervalSeconds 去『改后需重启』措辞与每次 Infof；删除 SecondsToCronSpec 与 F-E 注释（映射层退役）"
    - "internal/services/addomain/ad_push_queue_drain_test.go：NUL/非法 UTF-8 回归用例（markFailed 写入 last_error 不含 0x00 且 utf8.Valid）"
    - "internal/core/db/migrations/migration_216_adpush_drain_sysconfig.go：seed remark 去掉『修改后需重启后端生效』（幂等 seed，不影响已装环境）"
  key_links:
    - "internal/scheduler/ad_sync_tasks.go:173-219 Phase 58 D-05 排水器注册段 —— 本 quick 的主改造点（drainSpec 启动映射 → 固定 tick + 门控）"
    - "internal/services/addomain/ad_push_queue_drain_service.go:608 LoadDrainIntervalSeconds —— 从『启动读一次』改为每 tick 调用，签名不变"
    - "internal/services/addomain/account_pool.go:359 sanitizeFailureReason —— 缺陷 X 复用的既有清洗函数（同包直调，不新增依赖）"
    - "internal/scheduler/cron.go:202 RegisterTask / :217 IsTaskRegistered —— web 可删任务白名单的唯一注册口，守护测试断言点"
    - "实证：12:15-12:50 刻度循环（drain_service.go:581 err.Error() 直拼 → last_error 含 0x00 → UPDATE 22021 → 行留 pending）"
---

<objective>
交付 v1.21 收尾两项用户批准的修复（m00403『在完成测试后就进行修复』+ m00534 方案A裁决）：

1. **排水间隔热重载（方案A）**：robfig/cron spec 注册后不可变 + 多实例部署（10.62.10.39 / 10.62.10.33）使推送式回调只覆盖单实例，故采用拉取式——cron 固定 */30 最小 tick，每 tick 读 sys_config 间隔并与 lastDrain 比较，到期才 DrainOnce。默认 30s 行为不变；300s 时 ±30s 抖动已获用户接受；间隔值变化打日志。多实例各自读 DB 天然收敛。

2. **定时任务防删守护**：排水器按 D-05 本就禁注册 sys_job（web 定时任务页面管不到），补一条 taskRegistry 守护测试锁死该不变量。

3. **缺陷 X（NUL in last_error）**：UAT 实测 12:15-12:50 每 5 分钟刻度循环 —— LDAP code 50 错误消息含 0x00，drain_service.go 三处 err.Error() 直拼进 markFailed/markDead 未清洗 → UPDATE 整体 22021 失败 → 行留 pending/attempts=0/next_retry_at=NULL 无限重试。修复 = 写入口统一过同包 sanitizeFailureReason + NUL 回归用例。

Output: 固定 tick + 门控的排水调度、守护测试、sanitize 修复与回归用例；SecondsToCronSpec 映射层退役。
</objective>

<execution_context>
按 GSD quick 单任务流程执行：写代码 → go build ./... → 定向测试（addomain TestPhase58 + scheduler 新测试）→ 全量 go build 验证 → SUMMARY.md → 更新 STATE.md Quick Tasks Completed。提交前必须征得用户确认（repo 规则）。
</execution_context>

<tasks>

<task id="1" name="方案A热重载：固定 tick + drainGate 门控">
- internal/scheduler/ad_sync_tasks.go 排水注册段（:173-219）重写：
  - 删除启动时 LoadDrainIntervalSeconds→SecondsToCronSpec 映射（:193-203）。
  - 新增包级 `type drainGate struct { lastDrain time.Time; lastInterval int }` 与 `func (g *drainGate) shouldDrain(now time.Time, interval int) bool`：零值 lastDrain → 放行（启动后首个 tick 立即排水，保持旧行为）；now.Sub(lastDrain) < interval → 跳过；否则放行并更新 lastDrain；interval != lastInterval 时 Infof『排水间隔配置变化 X→Y（热重载生效，无需重启）』并更新 lastInterval。
  - AddJob 固定 "*/30 * * * * *"，闭包内：s.db == nil → interval=30；否则 addomain.LoadDrainIntervalSeconds(s.db)；经 gate.shouldDrain 决定是否 DrainOnce。drainer 仍闭包外构造一次（mu 互斥不变）。
  - 更新 D-05 注释块与启动日志（不再含 drainSpec）。
- internal/services/addomain/ad_push_queue_drain_service.go：
  - LoadDrainIntervalSeconds 注释改『每 tick 读取（方案A热重载）』，删除成功路径 Infof（每 30s 会刷屏），保留读取失败/非法值 Warnf。
  - 删除 SecondsToCronSpec 函数与 F-E 陷阱注释（唯一调用方已移除）。
- 新测试 internal/scheduler/ad_sync_tasks_drain_gate_test.go：drainGate 四态（零值/未到期/到期/间隔变化）纯时间断言。
</task>

<task id="2" name="防删守护测试（web 页面删不掉排水器）">
- 同测试文件内：构造空 Scheduler{}，RegisterADSyncTasks + RegisterADAccountPoolTasks 后，遍历 taskRegistry 断言不存在含 "drain" 或 "ad_push_queue" 的键（IsTaskRegistered("ad_push_queue_drain")==false）。锁死 D-05『禁注册 sys_job』——web 删除链路（job_service.go Delete 只碰 sys_job 表）因此结构性够不着排水 cron。
</task>

<task id="3" name="缺陷 X：markFailed/markDead 过 sanitizeFailureReason + NUL 回归用例">
- internal/services/addomain/ad_push_queue_drain_service.go：markFailed 与 markDead 在组装 Updates map 前对 msg/reason 调 sanitizeFailureReason（同包 account_pool.go:359，覆盖 :581/:298/:390 全部 err.Error() 直拼调用点；markPushed 写常量空串无需处理）。
- internal/services/addomain/ad_push_queue_drain_test.go 新增 TestPhase58_MarkFailed_SanitizesNulInLastError：sqlite 建 ad_push_queue 单行 → markFailed 传含 "\x00" 与无效 UTF-8 的 msg → 断言 UPDATE 成功且 last_error 不含 0x00、utf8.Valid；markDead 同款一例。
</task>

<task id="4" name="收尾：migration remark、构建与测试、SUMMARY">
- migration_216 seed remark 去『修改后需重启后端生效』（幂等 seed 只影响新装环境；用户现网 remark 属数据，交付时提示一条 UPDATE 可选执行）。
- go build ./... 全绿；go test ./internal/services/addomain/ -run TestPhase58 -count=1 全绿；go test ./internal/scheduler/ -run 'DrainGate|DrainGuard' 全绿。
- 写 261021-iqq-SUMMARY.md；更新 .planning/STATE.md Quick Tasks Completed 表。
</task>

</tasks>
