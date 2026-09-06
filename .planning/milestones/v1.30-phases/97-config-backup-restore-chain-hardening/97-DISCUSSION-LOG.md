# Phase 97: config_backup 恢复链加固 — Discussion Log

**Date:** 2026-09-06
**Phase:** 97-config-backup-restore-chain-hardening
**Domains Discussed:** 3

---

## Area 1: 超时互斥原子化（V130R-01）

**Question:** V130R-01 超时互斥原子化：ExecuteCustom 超时后继续推送命令、背对背双任务窗口，如何修复？

**Options Presented:**
1. 分段子 context 两段预算（推荐）— 恢复前备份（短TTL）+ RestoreConfig下发（长TTL）；任一段超时goroutine检测ctx.Done()立即停止；ctx cancel后下发真正停止，不依赖ExecuteCustom内部超时
2. 超时后确认任务真终止 — ExecuteCustom返回后检查是否真正停止；若仍在运行则fail task+记录已发进度；不依赖ctx cancel作为唯一停止信号
3. 保持+轮询检测 — ctx cancel后等待grace period再标记failed；确保下发goroutine有响应时间；最简单但 grace period 长度难以确定

**Selection:** 分段子 context 两段预算（推荐）

**Follow-up Notes:**
- D-01: runRestore 内部分两段 context：阶段① 恢复前备份（短 TTL）+ 阶段② RestoreConfig 下发（长 TTL）
- D-02: constants.RestoreConfigTimeout 拆分为 RestoreBackupTimeout（30s）和 RestoreConfigExecTimeout（5min）
- 超时后任务 fail，下发 goroutine 检测 ctx.Done() 立即退出

---

## Area 2: 多实例归属过滤（V130R-02）

**Question:** V130R-02 多实例归属过滤：RecoverStaleRunningTasks 多实例重启时误杀其他实例任务，如何修复？

**Options Presented:**
1. grace period 容忍（推荐）— WHERE status='running' AND updated_at < now() - grace_period；grace_period=2×RestoreConfigTimeout（约10分钟）；崩溃后30分钟内running不收敛，等待自愈；最简单，进程级单条记录粒度
2. 实例标识列 — ADD COLUMN instance_id VARCHAR(64)；进程启动时生成hostname+pid；RecoverStaleRunningTasks 只收敛instance_id=当前实例的任务；准确但需新增列和migration
3. 不收敛running，只收敛pending — pending是孤儿（无goroutine认领）→收敛；running态进程还在DB事务可能已提交→不收敛；保守，running态崩溃时可能还在执行

**Selection:** grace period 容忍（推荐）

**Follow-up Notes:**
- D-03: grace_period = 2 × RestoreConfigTimeout（约10分钟）
- pending 态照常收敛（无 goroutine 认领，是真孤儿）
- A5 discretion 方案①，状态机自洽性收口，不违 Phase 93 D-20

---

## Area 3: 业务错误码语义化（V130R-03）

**Question:** V130R-03 业务错误码语义化：'进行中'→409、'不属于设备'→400，统一500如何修复？

**Options Presented:**
1. 新建 BusinessError 类型（推荐）— pkg/response/business_error.go结构体携带HTTPStatus+Code；HandleServiceError识别*BusinessError返回对应HTTP状态+业务码；其他error走默认500；可跨模块复用
2. 扩展HandleServiceError关键词判断 — HandleServiceError内部判断error message关键词：'进行中'→409，'不属于'→400；简单但脆弱，message变了就失效
3. handler层特判config_backup — Restore handler不走HandleServiceError，自己判断error类型返回正确状态码；最简单但不符合复用精神

**Selection:** 新建 BusinessError 类型（推荐）

**Follow-up Notes:**
- D-04: pkg/response/business_error.go，HTTPStatus + Code + Message
- HandleServiceError 识别 *BusinessError，返回对应 HTTP status
- StartRestore 返回错误改用 BusinessError：409001（进行中）/ 400001（跨设备）
- 其他模块可复用 BusinessError

---

## Summary

| Area | Decision | Key Points |
|------|----------|-------------|
| V130R-01 超时互斥 | 分段子 context 两段预算 | 两段 TTL + ctx.Done() 检测停止 |
| V130R-02 实例归属 | grace period 容忍 | 10min grace period，pending 收敛 running 不收敛 |
| V130R-03 错误码 | 新建 BusinessError | HTTPStatus+Code，HandleServiceError 识别 |
