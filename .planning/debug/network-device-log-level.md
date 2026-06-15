---
slug: network-device-log-level
status: resolved
trigger: "网络设备相关的定时任务和textfsm的相关日志太多了，请合理调整日志级别"
created: 2026-06-15
updated: 2026-06-15
goal: find_and_fix
---

# Symptoms

- **预期**: 网络设备采集 / 定时任务 / TextFSM 解析在正常运行时只输出关键日志(info 级别),过程细节降到 debug
- **实际**: 即使 prod(info 级别),每个 cron 周期、每台设备、每次命令执行都刷大量 `applogger.Infof` 过程日志,淹没关键信息
- **环境**: dev(level=debug) 与 prod(level=info) 都需要调
- **用户倾向**: 严格分级 + 模块级控制 + 配置开关 + 减少采样

# Current Focus

- **hypothesis**: 日志噪音根因不是"日志框架问题",而是"日志级别用错"——大量正常流程步骤(连接获取、命令发送、逐条解析、轮询)被错误地用 `Infof` 记录;且 TextFSM 解析循环内有逐行 Infof。修复无需改框架,降级即可。
- **test**: 改完后 `go build ./...` + 检查 portcollection/device 包日志调用分级
- **expecting**: prod(info) 下采集过程静默,仅保留关键节点(任务开始/失败/完成统计)与 Warn/Error
- **next_action**: 按分级方案修改 executor.go / task_scheduler.go / connection_pool.go / collection.go / parser.go / utils.go / cron.go

# Root Cause

**日志系统** `pkg/logger` 基于 logrus,**仅一个全局 logger**,通过 `SetLevel()` 全局设置级别,**不支持按模块独立级别**。dev=debug / prod=info。

**问题:正常路径步骤普遍误用 `Infof`**,导致 info 级别下仍刷屏。逐点定位:

| 文件 | 位置 | 级别 | 频次 | 性质 | 处置 |
|------|------|------|------|------|------|
| internal/device/executor.go | 197-355 | Infof ×22 | 每命令×重试 | 全过程步骤 | 降 Debug(保留"所有重试失败""命令执行失败"为 Info/Warn) |
| internal/device/task_scheduler.go | 141-326 | Infof ×20 | 每任务×设备 | 全过程步骤 | 降 Debug(保留 panic / 队列满 / 停止 为 Warn/Info) |
| internal/device/connection_pool.go | 194-568 | Infof ×7 | 每设备连接 | 凭证解密/连接成功 | 降 Debug(保留关闭/失败) |
| internal/device/test_service.go | 59-191 | Infof ×10 | 手动测试触发 | 低频 | 保留(低频用户主动操作) |
| internal/services/portcollection/collection.go | 98-249 | Infof ×13 | 每设备每阶段 | 采集过程 + `[调试]`/`[警告]` 前缀 | `[调试]`→Debugf;`[警告]`→Warnf;过程步骤降 Debug |
| internal/services/portcollection/parser.go | 164,170 | Infof ×2 | **循环内逐行** | 最高频(TextFSM 解析每条接口) | 降 Debug |
| internal/services/portcollection/utils.go | 104,106 | Infof ×2 | **循环内逐接口** | 最高频(dot1x 匹配) | 降 Debug |
| internal/scheduler/cron.go | 91,66 | Infof | 每 cron 触发 | 任务执行/成功 | "执行任务"降 Debug;"任务执行成功/失败"保留(关键) |
| internal/scheduler/*_tasks.go | 各处 | Infof | 每 cron 周期 | 任务开始/完成统计 | 保留(低频关键节点);过程明细降 Debug |

**textfsm 相关日志**实际落在 `portcollection/parser.go` 的解析循环(164 行循环内逐条 Infof),是最高频噪音源。

# Fix Plan

**策略: 方案 A(级别降级)为主,见风险最低、见效最快。**

降级规则:
1. 循环内逐条日志 → `Debugf`(parser.go:164, utils.go:104/106)
2. `[调试]` 前缀 Infof → `Debugf`(去掉前缀)
3. `[警告]` 前缀 Infof → `Warnf`(去掉前缀)
4. 连接/命令/任务的成功路径过程步骤 → `Debugf`
5. 保留为 Info 的: 任务级开始/完成统计、首次连接、配置成功、关键里程碑
6. 保留为 Warn/Error 的: 所有错误路径、降级、panic

不改 `pkg/logger` 框架(避免风险);模块级控制作为后续可选增强。

# Evidence

- 2026-06-15 扫描 internal/device、collectors、portcollection、templates、scheduler 全部日志调用点(见上表)
- 2026-06-15 确认 pkg/logger 为全局单 logger,SetLevel 全局生效,无模块级

# Eliminated

(none)

# Resolution

- **root_cause**: 正常流程步骤普遍误用 `Infof`,导致 prod(info 级别) 下仍刷屏;TextFSM 解析循环(`parser.go`/`utils.go`)内逐条 `Infof` 是最高频噪音源
- **fix**: 5 个核心文件按分级规则降级 — 循环内逐条 + 过程明细 → `Debugf`;失败/异常/panic → `Warnf`;去掉 `[调试]`/`[警告]` 冗余前缀;保留调度器/任务生命周期里程碑为 `Info`
- **verification**: `go build ./...` 通过(根目录无临时 .go 干扰);`go vet` 通过(portcollection + device 包,exit 0)
- **files_changed**:
  - `internal/services/portcollection/parser.go` (循环内逐条 Infof→Debugf)
  - `internal/services/portcollection/utils.go` (dot1x 匹配循环 Infof→Debugf)
  - `internal/services/portcollection/collection.go` (17 处:[调试]→Debugf/[警告]→Warnf/过程降级)
  - `internal/device/executor.go` (过程→Debugf;6 处失败/Panic→Warnf)
  - `internal/device/task_scheduler.go` (过程→Debugf;3 处生命周期里程碑保留 Info;4 处 panic/失败→Warnf)
- **scope**: 仅核心噪音源 5 文件,未改 `pkg/logger` 框架,未加 config 开关(按用户确认)。connection_pool.go / scheduler/cron.go 本次未动,留待后续如需再处理
- **effect**: prod(info) 下网络设备采集过程静默,仅保留任务开始/失败/完成统计与 Warn/Error;dev(debug) 全过程仍可见
