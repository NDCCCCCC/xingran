---
phase: 93
plan: 06
subsystem: services/restore-e2e + docs
tags: [e2e, filetransport, fixture, docs-gate]
key-files:
  created:
    - internal/services/config_backup_service_93_02_test.go
  modified:
    - internal/services/config_backup_service.go
    - .planning/REQUIREMENTS.md
    - .planning/workstreams/milestone/ROADMAP.md
    - CLAUDE.md
metrics:
  tests_added: 9（TestCbk93Restore{E2E_Happy, E2E_HashMismatchWarns, E2E_SendInterrupted, PreBackupFailureAborts, CrossDeviceRejected, MutualExclusion}, TestCbk93{RecoverStaleRunning, StartRestoreDBFailure, SortWhitelistNoStatus}）
  commits: 4
---

# Plan 93-06 Summary: restore e2e + 失败场景全量 + 措辞校准 + Convention 收口

## What Was Built

1. **restore e2e 套件**（93_02_test.go，BACKUP-CLOSED-04/05）：
   - `TestCbk93RestoreE2E_Happy`（D-29/D-30 主断言链）：CreateBackup（真实 GetConfig 设备读）→ StartRestore → 异步 runRestore（恢复前备份 → GetBackupContent → RestoreConfig 推送 3/3 行 → 回读）→ 任务 success + hashMatched=true + **版本链恢复记录 hash/内容与源备份一致** + GetBackupStatistics 健康——全程 FileTransport 零 SSH
   - `TestCbk93RestoreE2E_HashMismatchWarns`（D-11）：回读漂移 → 任务仍 success + hashMatched=false
   - `TestCbk93RestoreE2E_SendInterrupted`（D-28③）：设备 % Error 拒绝（failed_when_contains）→ failed + FailedLine="shutdown" + SentLines 部分计数（D-12 留痕）+ 无版本链记录
   - `TestCbk93RestorePreBackupFailureAborts`（D-28①+D-14）：设备缺失 → 恢复前备份失败中止，ErrorMessage 含 "恢复前自动备份失败"
   - `TestCbk93Restore{CrossDeviceRejected, MutualExclusion}`（D-04/D-08 纯 DB）、`TestCbk93{RecoverStaleRunning, StartRestoreDBFailure}`（A5 收敛 + D-28④ DropTable 手法）
2. **REQUIREMENTS/ROADMAP 措辞校准**（D-01/D-33①，commit 4b61a22）：BACKUP-CLOSED-03 "事务化/批量 upsert" → 异步任务化设备下发语义；`_78_NN` 笔误 → `_93_NN`；失败场景族措辞（磁盘满/事务回滚 → 写失败/损坏 gzip/下发中断留痕/DB 写入失败）；ROADMAP plan checkbox 93-01..05 置 [x]
3. **D-33② sort 白名单修复**（独立 commit 02b99ec）：`configBackupAllowedSortFields` 删除不存在的 status 列（用户传 orderByColumn=status 不再 SQL 500，改走默认排序）；守护测试 `TestCbk93SortWhitelistNoStatus` 先红（SQL 错误复现）后绿
4. **CLAUDE.md "Config Backup Restore Convention" 段**（D-32，commit ff5063b）：三条锁定（gzip helper 唯一 / RestoreConfig 唯一下发入口 / 异步任务模式强制）+ 回归守护索引

## Key Discoveries（fixture 字节对齐实证）

- **scrapligo 读循环结构**（debug logger 逐字节重建）：`SendCommand` 内部先 GetPrompt probe（消费一个 prompt 行）再 write+read-until-prompt；`AcquirePriv` 首个 GetPrompt 的 write \n 会连续跳过非 prompt 行（如 system-view 回显）直到 joined-pattern 匹配
- **fail-fast 注入的行序要求**：% Error marker 必须位于 prompt 行**之前**（作为响应记录的一部分被扫描）；放在 prompt 之后会被下一命令的 read 消费、判定失效（Happy 走通后 SendInterrupted 挂 18s 的根因）
- 初版生成器把回读周期并入 GetConfig 循环（多出 1 组周期导致全流错位、spares 被 EOF 吃光后阻塞）——修正为 pre-restore 周期 ×N + SendConfigs 段 + 独立回读段
- 回读 GetConfig 的 `AcquirePriv(exec)` 消费 push 段尾的 quit/[Huawei]/return/<Huawei> 四行——fixture 尾部多余字节无害（EOF 才致命）

## Commits

| Commit | Description |
|--------|-------------|
| 4b61a22 | docs(93): calibrate restore semantics wording and fix test-file-name typo (D-01/D-33) |
| 61c68bb | test(93): restore e2e and failure-scenario regression suite (D-27..D-30) |
| 02b99ec | fix(93): remove nonexistent status column from backup sort whitelist (D-33) |
| ff5063b | docs(93): add Config Backup Restore Convention to CLAUDE.md (D-32) |

## Deviations

- D-28③ 手法：plan 的"fixture 第 N 行后 EOF"不可行（FileTransport EOF = 无限阻塞，78-03 S-2/93-02 先例）→ 设备 % Error 拒绝 marker（93-02 deviation 同款）
- D-28④ 手法：sqlite 无 varchar 长度约束/超长列值不报错 → DropTable 注入任务表写入失败（StartRestore 同步路径"无半写记录"语义成立）
- Happy e2e 未混入大文件压缩路径（93_01 的 TestCbk93CreateBackupCompressedLargeFile + AutoBackupCompressesLargeFile 已独立守护压缩闭环；e2e 聚焦 BACKUP-CLOSED-05 的恢复断言链，避免 chdir/文件存储耦合）
- 版本链 hash 断言用源备份行的 ConfigHash（GetConfig result 字节）而非 fixture 常量（尾部换行归属 scrapligo Response 层）

## Self-Check: PASSED

- `go build ./...` exit 0
- `go test ./internal/services/ -run TestCbk93 -count=1` → ok 7.1s（19 用例：93_01 ×10 + 93_02 ×9）
- **`go test ./... -count=1` 全量 exit 0（v1.29 D-03 零失败底线）**
- `npm run type-check` exit 0；`npm run lint` 0 error；**`npm run test` 551 文件 3780 用例全绿**
- grep：REQUIREMENTS 无 "校验 schema"/"_78_NN"；ROADMAP 无 "批量 upsert"；CLAUDE.md 含 Convention 三标识符
- git log：D-33② 修复位于独立 commit 02b99ec
