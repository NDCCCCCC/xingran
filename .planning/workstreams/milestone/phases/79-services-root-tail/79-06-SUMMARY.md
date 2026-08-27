---
plan: 79-06
phase: 79-services-root-tail
executed: 2026-08-28
commits:
  - 20f9e46 (test(79-06): device ForTesting 连接池种子 helper(D-79-02,AST 守护内))
  - 208a698 (test(79-06): credential helper + discovery Tier-1(IP 数学/isAlive TCP/CRUD))
  - dd66020 (test(79-06): config_backup 纯函数 + sqlite/文件路径(TempDir 隔离))
  - e175caf (test(79-06): device_info_collection 生命周期 + 配置 + processTask(Tier-2 打通))
  - c557a01 (test(79-06): device_monitor setter/nil-guard/版本映射/委托链)
  - e9e6133 (test(79-06): config_execution/command_dispatch validate 段 + executor 派发)
  - c477d40 (test(79-06): snmp fake 移植 + snmpProbe/pingDeviceViaSNMP(D-79-05))
  - 3d8019e (docs(79-06): phase 79 收口 — device 家族达标 + coverage-baseline Phase 79 后回填 + gate 全绿)
  - <followup> (docs(79-06): baseline commit 列回填 TBD → 3d8019e)
---

# 79-06 Summary — device 家族 + 全包复测 + coverage-baseline 回填 (Phase 79 收口)

## 交付

7 个新测试文件(3 381 行,58 个顶层测试函数,约 190 个含子测试的用例)+ 1 个豁免生产树
touch + coverage-baseline 回填。零业务生产行为变更。

| 文件 | 行数 | 覆盖目标 |
|------|------|----------|
| `internal/device/e2e_helpers_79_06_test.go` | 220 | SeedConnectionForTesting 种子复用/覆盖/守卫/多设备(FileTransport 公共 API) |
| `internal/services/device_credential_helper_79_06_test.go` | 327 | 凭证四分支 + 批量装配 + 幽灵凭证 quirk;含共享装配 `newDB7906`(UUID fill 回调) |
| `internal/services/device_discovery_service_79_06_test.go` | 627 | IP 数学四函数表驱动 + isAlive 真 TCP + CRUD 主链 + ExecuteDiscovery(scan/snmp) + ProbeSingleDevice |
| `internal/services/config_backup_service_79_06_test.go` | 711 | 纯函数面 + CRUD/文件链 + loadConfig + Tier-2 CreateBackup/AutoBackup/Batch(种子池) |
| `internal/services/device_info_collection_service_79_06_test.go` | 437 | 生命周期/恢复/配置/命令矩阵 + processTask Tier-2 全链 + Stop→Start quirk |
| `internal/services/device_monitor_service_79_06_test.go` | 416 | setter/nil-guard/convertSNMPVersion/委托链 + CheckDeviceStatus SNMP happy |
| `internal/services/config_execution_command_dispatch_79_06_test.go` | 491 | 校验阶梯 + 连接缺失矩阵 + dispatch errgroup fan-out(种子池)+ QuickCommand |
| `internal/services/snmp_fake_server_79_06_test.go` | 152 | 78-04 fake 裁剪移植(GetRequest-only,D-79-05) |
| `internal/device/e2e_helpers.go` | +51 | **唯一豁免生产树 touch**:`SeedConnectionForTesting`(append-only) |
| `.planning/coverage-baseline.md` | +54 | 『Phase 79 后』段(ratchet 行 + per-package block + 倒退检查 + device 家族判定表) |

### 关键基建(后续 plan 可复用)

- **`newDB7906`(device_credential_helper_79_06_test.go)**:sqlite 装配 +
  `test7906:fill_uuid` gorm Create 前置回调 —— DeviceDiscovery / ConfigExecution /
  ConfigExecutionDetail 等裸 UUID 主键模型没有 BeforeCreate 钩子,PG 有
  gen_random_uuid() 默认而 sqlite 没有,同表第二次 Create 必撞空串主键。
- **`newExecutor7906` / `cex7906SeedExecutor` + `writeFixture7906` /
  `newDriver7906FromFixture`**:公共构造器装配(pool → scheduler → executor)+
  `device.SeedConnectionForTesting` 种子 FileTransport 连接;pool.Close 用 3s
  watchdog 防 fixture 耗尽读阻塞(QUIRK-P2/S-2)。
- **`fakeSNMPServer7906`**:78-04 fake 裁剪移植,normal/errorStatus/drop 三档。

## Coverage checkpoint(`go test -count=1 -coverprofile` 全包一次,445s)

| 文件 | 基线(RESEARCH §2) | 实测 | 目标 | 结果 |
|---|---|---|---|---|
| device_discovery_service.go | 1.4% (4/293) | **87.4%** (256/293) | ≥60% | ✅ |
| device_info_collection_service.go | 29.5% (115/390) | **66.9%** (261/390) | ≥65% | ✅ |
| config_backup_service.go | 0% (0/244) | **86.9%** (212/244) | ≥65% | ✅ |
| device_monitor_service.go | 0% (0/189) | **68.3%** (129/189) | ≥65% | ✅ |
| config_execution_service.go | 0% (0/152) | **65.1%** (99/152) | ≥60% | ✅ |
| command_dispatch_service.go | 3.4% (4/116) | **90.5%** (105/116) | ≥60% | ✅ |
| device_credential_helper.go | 0% (0/47) | **95.7%** (45/47) | ≥70% | ✅ |
| **7 文件合计** | 123/1431 (8.6%) | **1107/1431 (77.4%)** | — | +984 covered |

**包级收口(SC-1/SC-3)**:internal/services root **81.60%**(5202 stmts / 4245 covered;
79-05 后 62.9% → 本 plan +974 covered)。全仓 `go test -coverprofile ./...` +
`check-coverage.sh`:**weighted avg 70.90%,gate exit 0**(P1 ×8、P2 ×10 floor 全 PASS)。
root 包 45 个文件 **0 个 <50%**(SC-2 discharge,`coverage.out` 聚合实测)。
数字已回填 `.planning/coverage-baseline.md`『Phase 79 后』段(commit 列 TBD,71-01b 先例)。

## D-79-02 豁免改动清单(唯一生产树 touch)

- `internal/device/e2e_helpers.go` append-only 追加 **`SeedConnectionForTesting(pool,
  deviceID, conn)`**:poolLock 写 connections map;`getDeviceLock` 对齐 pc.mu
  (78-03 seedPool78 锁一致性规则);nil pool / nil conn / 空 deviceID 三重 no-op;
  注释沿用 ForTesting 三层隔离契约(TEST-ONLY、跳过的 bookkeeping 清单、production
  引用即 AST guard fail)。
- **不需要** `NewDeviceConnectionPoolForTesting` / `SetPoolForTesting`:公共构造器
  `NewDeviceConnectionPool(db, cipher, cfg)` → `NewDeviceTaskScheduler(pool, cfg)` →
  `NewDeviceExecutor(scheduler, cfg)` 已可跨包装配完整 executor(执行时探针结论,
  Task 1 计划里的「择一追加」条款未触发)。
- `for_testing_guard_test.go` 全绿(production 零引用);`git diff` 确认 e2e_helpers.go
  之外 internal/device 生产文件 diff = 0。
- Tier-2 战果:CollectDeviceInfo/processTask、CreateBackup/AutoBackup、dispatch
  errgroup fan-out、pool GetConnection 复用路径全部经种子池真实走通。

## D-79-05 处置记录(SNMP fake)

- fake 落 `internal/services/snmp_fake_server_79_06_test.go`(同包 test-only,文件头
  溯源注释含 78_04),只保留 GetRequest 分支(三个调用点均为单 OID Get),
  gosnmp 公共编解码 + RequestID 原样回填,`net.ResolveUDPAddr("udp", "127.0.0.1:0")`,
  conn/goroutine 经 t.Cleanup 收口。
- **Windows loopback 降级条款未触发**:绑定 socket 的收发在 Windows 本机稳定,
  snmpProbe/discoverBySNMP/pingDeviceViaSNMP/CheckDeviceStatus 的 happy 路径全部
  直接测通(78-04 记录的跨 socket 丢弃问题未复现)。
- 唯一慢用例:snmpProbe drop 超时分支(5s,生产超时常量)。pingDeviceViaSNMP 的
  drop 变体不驱动(5s×2 重试×双 OID ≈ 30s),改用「关闭端口 → ICMP port-unreachable
  → connected socket 读快速失败」锁 false 分支;注释落档。
- **QUIRK-79-06-I(新发现)**:gosnmp v1.35 不把 SNMP error-status(NoSuchName)转为
  Go error,而生产代码只看 err → error-status 应答仍算 alive/ping 成功且解析回显
  变量;调用方无法区分「无此 OID」与「真值」。只锁不修。

## Tier-2 未达段清单(按 plan「尽力而为」条款)

| 段 | 位置 | 原因 | 处置 |
|---|---|---|---|
| SendConfig happy 路径 | config_execution_service.go `executeConfigOnDevice`(43.6%)、`executeConfig`(48.4%,parallel 分支) | 手写 fixture 的 huawei_vrp config-mode 字节形态与 scrapligo 期望不匹配,读阻塞至 ctx 截止(portwrite 的可回放形态依赖其服务的多行 config 序列) | 连接缺失失败矩阵 + 校验阶梯 + 行/明细写链已覆盖;SUMMARY 落档 |
| 组件采集 hook | device_info_collection_service.go `collectComponentInfo`(14.3%)、`runDeviceCommand`(16.7%)、`GetByDeviceSN`(0%) | 需要真实组件解析链 + Asset 表拓扑;boards/transceiver 管线已有 79 前测试文件覆盖 | runDeviceCommand nil 守卫已断言;其余留档 |
| 全量状态巡检 | device_monitor_service.go `CheckAllDevicesStatus`(0%) | 单文件已 68.3% ≥65 目标;并发巡检为 CheckDeviceStatus 的薄循环 | 留档(后续补差可在 +1 测试内完成) |
| pingDeviceViaSNMP drop 变体 | device_monitor_service.go | 生产配置下 ≈30s/布尔值 | 关闭端口分支等价锁 false;注释落档 |

## Quirks 处置(全部「只锁不修」,零生产改动)

- **QUIRK-79-06-A** `GetCredentialsForDevices`:设备 CredentialID 指向已删除凭证行时
  该设备被静默丢出结果 map(不报错),调用方须按「缺 key = 无凭证」处理。
- **QUIRK-79-06-B** `calculateIPCount`:字节差按 uint8 运算,末字节 end<start 时回绕
  (2-10 → 248),end<start 的区间计 249 而非 0 —— TotalIPs 静默多计。
- **QUIRK-79-06-C** `ipLessEqual` 无 To4() nil 守卫,IPv6 入参会 panic(生产链路只产
  To4 归一化的 IP,不可达,但公共语义脆弱)。
- **QUIRK-79-06-D** `CollectDeviceInfo` 过了凭证守卫后直接解引用 nil executor 会
  panic(`runDeviceCommand` 反而有显式守卫 —— 同文件守卫不一致)。
- **QUIRK-79-06-E**(现网可见)`DeviceInfoCollectionService` Stop 后再 Start:isRunning
  守卫放行但 stopChan 已关闭 → 新 worker 立即退出、任务永 pending;且下一次 Stop 在
  `close(s.stopChan)` 上 panic(close of closed channel)。
- **QUIRK-79-06-F** `DeviceMonitorService.Close` 零值服务(未初始化)静默返回 nil,
  无「未初始化」信号。
- **QUIRK-79-06-G**(现网可见)`ExecuteByTemplate`/`Dispatch` 在 executeConfig 返回后
  无条件把执行行盖 ExecutionStatusSuccess 戳 —— 全部设备失败时执行行仍是 success,
  统计口径(GemStatistics)随之失真。
- **QUIRK-79-06-H** `executeConfigOnDevice`/`executeOnDevice` 均无 nil executor 守卫
  (require.Panics 证据;生产恒非 nil 装配,不可达)。
- **QUIRK-79-06-I** gosnmp v1.35 不上抛 SNMP error-status(见 D-79-05 段)。
- **复录 QUIRK-79-04-D** `NetworkDevice.Status` 零值 0(online)被列默认 2 吞掉,
  种子一律 Update 回写。
- **测试基建备忘**:fixture 落进程 temp(非 t.TempDir,FileTransport 长持文件句柄,
  Windows RemoveAll 必炸);`cbk7906Chdir` 的 cwd 隔离同样落进程 temp(applogger 惰性
  打开 ./logs/app.log);IPAddress 带唯一索引,种子用递增序列分配。

## Deviations from Plan

1. **[环境] `go test -race` 本地不可执行**:Windows cgo 工具链故障(`cgo.exe: exit
   status 2`),与 79-01..05 同源。goroutine 纪律由 t.Cleanup 全量停机(worker /
   scheduler / pool cleanup / SNMP / TCP listener)+ 禁 t.Parallel +
   require.Eventually 维持;ci.yml Linux race job 兜底。
2. **[计划-实装口径]** Task 6 的 `TestCex7906_Execute_SeededPath`(SendConfig happy)
   不可达,按 plan 自带「Tier-2 尽力而为 + 不可达段 SUMMARY 记录」条款改为
   `TestCex7906_Execute_MultiDeviceSerial`(连接缺失矩阵 + 行/明细写链 + 统计对照);
   errgroup fan-out happy 经 dispatch 侧(SendCommand)覆盖。Tier-2 深度探针结论:
   helper + 公共构造器足以打通 SendCommand 面的全部 Tier-2,SendConfig 面仅缺
   可回放的 config-mode fixture。
3. **[计划-实装口径]** Task 5 的 monitor SNMP drop 变体以关闭端口分支替代(30s 成本
   考量,注释落档);Task 3 的「备份目录指 t.TempDir()」实现为 t.Chdir 进进程 temp
   目录(applogger 文件锁使 t.TempDir 的 RemoveAll 必失败)。
4. **[Rule 3] Task 6 需要向 Task 3 的文件补一个共享底层 fixture writer
   (`writeFixtureBytes7906`)**,同 plan 内自洽,未新增依赖。
5. **[Rule 1] 测试侧适配若干**(均为测试自身 bug):IPAddress 唯一索引冲突、
   `models.Vendor` → `models.DeviceVendor` 类型名、模板引擎为 Go text/template
   (变量引用须 `{{.name}}`)、QuickCommand 的命令必须与 fixture echo 一致、
   `ExecutionStatistics` 字段名 Failed(非 Failure)、`net.Listen` 返回接口类型。
6. **[提交规范]** 计划中的 `test-infra(79-06)` commit 类型被仓库 commitmsg 钩子
   (conventional type-enum)拒绝,改用 `test(79-06)` 并在正文标注 test-infra 属性;
   subject 首词大写(SNMP)同样被 subject-case 拒绝,改小写。

## Known Stubs

无新增业务 stub。`RestoreBackup` 的「配置恢复功能待实现」与 `GetDiscoveryResults`
的空列表均为**既有生产 stub**,按计划以现行为锁定(断言注释),非本 plan 引入。

## Threat surface scan

无新增安全面:所有网络行为限于 127.0.0.1 loopback(fake SNMP UDP / isAlive TCP /
FileTransport 本地文件);测试凭据全部字面量("public" 等);T-79-06-01 的 ForTesting
生产引用防线(AST guard)实测全绿;T-79-06-05 的 baseline 数字与 gate 输出逐字一致。

## Phase 79 SC-1..SC-4 逐条判定

| SC | 内容 | 判定 |
|----|------|------|
| SC-1 | root 包 ≥70% | ✅ **81.60%**(5202/4245;基线 11.3%,79-05 后 62.9%) |
| SC-2 | root 文件按 profile 倒序补齐,无单文件 <50% | ✅ 45 文件 0 个 <50%(coverage.out 聚合) |
| SC-3 | `go tool cover -func` 实测回填 coverage-baseline.md | ✅ 『Phase 79 后』段(行 + per-package block + 倒退检查 + device 家族判定表) |
| SC-4 | gate 全绿(不动 `.coverage-threshold`) | ✅ exit 0,weighted 70.9% ≥ 55.5;P1×8 + P2×10 floor 全 PASS;`.coverage-threshold` diff=0 |

## 决策执行情况(D-79-01..07)

| ID | 执行 |
|----|------|
| D-79-01 | 本 plan = research §7 79-06 行(device 家族 7 文件)+ phase 收口;mac_perf_config_seed 已在 79-05 覆盖(81.8%),device_credential_helper 归入本 plan(95.7%)✅ |
| D-79-02 | ForTesting helper 仅 `SeedConnectionForTesting` 一个符号;e2e_helpers.go append-only;AST 守护全绿;Tier-1 先行 + Tier-2 种子池打通(SendCommand 面);不可达段落 SUMMARY ✅ |
| D-79-05 | fake 裁剪移植为同包 `_79_06_test.go`;Windows 降级条款未触发(happy 稳定);drop 高成本变体以关闭端口/注释替代 ✅ |
| D-79-06 | 命名 `<source>_79_06_test.go` + 前缀 TestXxx7906_ / helper newXxx7906 全文件一致 ✅ |
| (沿用) | SC-3 回填 = 73-05 收口同款;`.coverage-threshold` 未动(Phase 81)✅ |
| (沿用) | QUIRK-P2 以 t.Cleanup 全收(pool Close 3s watchdog + scheduler.Stop + svc.Stop),未修生产代码 ✅ |

## 验收标准对照(plan success_criteria)

| 标准 | 结果 |
|---|---|
| 7 测试文件 + e2e_helpers.go 存在且全绿;TestDch7906_/TestDdv7906_ ≥8、TestCbk7906_ ≥7、TestDic7906_ ≥6、TestDmn7906_ ≥5、TestCex/Cdp ≥5、SNMP 双分支 | ✅ 3/13、16、9、8、6(11 顶层两文件)、happy+errorStatus+drop/关闭端口 |
| 7 文件各自目标(≥60-70%) | ✅ 全过(见上表,最低 config_execution 65.1%) |
| root ≥70% 落 SUMMARY + baseline | ✅ 81.60% |
| D-79-02 仅 e2e_helpers.go 且 AST 全绿;D-79-05 同包 test-only + Windows 处置落档 | ✅ |
| gate 全绿 exit 0、`.coverage-threshold` 未动、`go build ./...` + `go test ./...` exit 0、每 task 原子 commit | ✅ 7 个 test commit + 本收口 commit |
| `go test -race ./internal/services/` | ⚠️ 本地 cgo 故障不可执行(Deviation #1,CI 兜底) |
| 生产 .go 改动 = e2e_helpers.go(D-79-02) | ✅ `git log --stat` 核对其余全为 *_test.go / .planning |

## Self-Check: PASSED

- 文件存在:8 个测试文件 + e2e_helpers.go(+51)+ coverage-baseline.md Phase 79 后段
  + 本 SUMMARY —— 全 FOUND。
- 提交存在:20f9e46 / 208a698 / dd66020 / e175caf / c557a01 / e9e6133 / c477d40 /
  收口 commit —— 全 FOUND(`git log --all`)。
- `go build ./...` exit 0;`go test -count=1 ./internal/services/`(含 coverage)exit 0
  (445s,81.6%);`go test -count=1 -coverprofile=coverage.out ./...` exit 0;
  `check-coverage.sh` exit 0。
- `-race` 本地不可执行(cgo,Deviation #1)。
