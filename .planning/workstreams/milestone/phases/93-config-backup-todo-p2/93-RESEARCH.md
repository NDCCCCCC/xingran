# Phase 93: config_backup 三处 TODO 闭环 - Research

**Researched:** 2026-09-05
**Domain:** Go 网络设备配置备份（gzip 压缩/解压）+ scrapligo 配置下发（异步任务化恢复）+ 回归测试基建
**Confidence:** HIGH（所有关键事实均在代码库 / scrapligo v1.4.0 模块源码 / Go 1.24.5 本地 toolchain doc 中逐一验证，标注 file:line）

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Restore 语义与技术路径 (Area 1)**
- **D-01**: 恢复语义按代码现实校准 = 把备份的设备配置下发到网络设备；93-02 措辞按 Phase 92 D-06 先例同步修订 REQUIREMENTS + ROADMAP（同 commit）
- **D-02**: `DeviceExecutor.RestoreConfig` 唯一下发入口 — device 包新增厂商感知恢复方法，与 `GetConfig` 对称；内部走 scrapligo SendConfigs 配置模式批量下发，复用连接池/调度/重试基建。不复用 portwrite wrapper、不用 ExecuteMultipleOnDevice
- **D-03**: 基础清洗后下发 — 过滤空行/注释行（`!`/`#`）；清洗规则放 device 包 RestoreConfig 内部
- **D-04**: 仅限同设备恢复 — service 层校验 `backup.DeviceID == req.DeviceID`
- **D-05**: 独立超时常量 — `pkg/constants/timeouts.go` 新增 RestoreConfig 专用超时（不用 CommandExecTimeout）
- **D-06**: 硬编码 vendor map — vendor→配置模式命令硬编码 Go map（v1.19 W1 `vendor_port_template.go` 先例）
- **D-07**: 结构化 RestoreResult — 下发行数/耗时/恢复后 hash/hashMatched 等；handler 响应同步扩展
- **D-08**: 同设备互斥 — 第二个请求报"恢复进行中"；实现形态 planner 定
- **D-09**: 版本链恢复记录 — 恢复成功后 `sys_config_backup` 插入一条记录（ChangeReason 记"恢复自版本 N"）

**恢复安全护栏 (Area 2)**
- **D-10**: 恢复前自动备份（复用现有备份链路，ChangeReason 记"恢复前自动备份"）
- **D-14**: 备份失败即中止（无法备份 = 无回退退路）
- **D-11**: 回读 hash 警告不失败 — 不一致时 `hashMatched=false` + 警告日志，restore 整体算成功
- **D-12**: fail-fast 即停 — 下发某行报错即停止，RestoreResult 记录进度（第 N/M 行 + 失败行内容）
- **D-13**: 复用 OperTypeUpdate — 25 OperType 锁值防线零改动

**异步任务化**
- **D-15**: 新建恢复任务表（status/进度/结果 JSON/错误信息）；表名/migration 编号 planner 定
- **D-16**: 前端轮询（不引入 WS）
- **D-17**: POST /restore 改为创建任务记录（pending）+ 立即返回 taskId；新增任务查询端点
- **D-18**: handler 发起时记 operlog（success path 末尾）；任务结果只在任务表，不双记
- **D-19**: 后端 + 最小前端（备份页恢复按钮异步交互 + Modal 进度/结果），不做独立任务管理页
- **D-20**: 不做任务清理 — 任务记录与备份记录同等保留
- **D-21**: 不设全局并发上限 — 同设备互斥 + DeviceExecutor 调度器队列自然限流
- **D-22**: 重新发起代替重试 — 不做任务级重试
- **D-34**: 四态状态机 pending → running → success/failed，不支持取消

**压缩/解压细节 (Area 3)**
- **D-23**: `.conf.gz` 后缀 + 双检查（Compressed 标志与 `.gz` 后缀同时检查）
- **D-24**: BackupSize 记原始大小（`len(config)` 口径与 DB 存储路径/统计 totalSize 一致）
- **D-25**: auto 路径统一压缩（`createNewAutoBackup` 大文件也压缩，复用同一 gzip helper）
- **D-26**: 默认压缩级别（gzip DefaultCompression，不做可配置化）

**失败场景与测试 (Area 4)**
- **D-27**: `config_backup_service_93_NN_test.go` 新文件（REQUIREMENTS 原文 `_78_NN` 是笔误）
- **D-28**: 锁 4 场景清单，手法不锁：① 写文件失败 ② 损坏 gzip 字节流 ③ restore 下发中断 → 任务 failed + 已发进度留痕 ④ DB 写入失败 → 记录回滚
- **D-29**: FileTransport e2e — restore 全链路用 `SeedConnectionForTesting` 零 SSH 测试
- **D-30**: 端到端验收 = 自动化集成测试（CreateBackup → GetBackupContent → restore 任务 → hash 一致）

**延伸决策 (Areas 5)**
- **D-31**: 任务端点复用 `/backups` 路由组权限（零新权限点）
- **D-32**: CLAUDE.md 加"配置备份恢复约定" Convention 段（锁三条：gzip helper 统一 / RestoreConfig 唯一入口 / 恢复必须异步）
- **D-33**: 两处顺手修各自独立原子 commit：① REQUIREMENTS `_78_NN`→`93_NN`（与 D-01 措辞修订同 commit）② `configBackupAllowedSortFields` 删除 `status` 行
- **任务表设计惯例（已锁定）**: 状态枚举用 string；UUID 主键走 BeforeCreate hook；AutoMigrate 注册 + MigrateNNN 注册双线齐全

### Claude's Discretion（planner/researcher 决定，无需再问用户）
- RestoreConfig 超时常量的具体值（D-05 只锁独立常量 + 放 timeouts.go）
- vendor map 的具体命令内容（对照 scrapligo priv 机制）
- 清洗规则的具体过滤集；任务表名、字段命名、migration 编号、Task service/handler 文件布局
- 任务查询端点路由形态（/backups/tasks vs /restore-tasks 等）
- RestoreResult/任务表 result JSON 的具体字段集
- 恢复记录的 BackupType 表达（复用 auto/manual vs 新增常量——若新增需同步锁值测试）
- 失败场景的具体模拟手法（对照 79_06 先例）；93_NN 测试文件内用例分组
- 同设备互斥的实现形态（DB 锁/in-progress 标记/内存锁）
- handler 响应 shape 细节（taskId 字段名等）

### Deferred Ideas (OUT OF SCOPE)
- 真机恢复 site-visit UAT（FileTransport 自动化已覆盖验收）
- 任务管理独立页面 / WS 实时推送 / 任务取消 / 任务级重试 / 任务清理 cron / 任务表分库归档
- RestoreConfig dry-run 预览模式（v1.20 FUTURE-06 同源）
- 恢复操作的 Redis 缓存失效联动（ConfigBackup 不在业务缓存体系）
- 18 项真实 TODO 中非 config_backup 的 17 项（v1.29 Out of Scope）
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| BACKUP-CLOSED-01 | 实现 `:158` 压缩逻辑（gzip 标准库，压缩备份内容到 .gz 文件） | §Code Examples gzip helper；D-23/24/25/26 全部有代码级落点；CreateBackup:156-166 与 createNewAutoBackup:388-394 双路径改造点已定位 |
| BACKUP-CLOSED-02 | 实现 `:206` 解压逻辑（识别 .gz 后缀 + Compressed 标志双检查） | §Pitfalls P5 双检查语义；gzip 错误形态（ErrHeader/ErrChecksum/ErrUnexpectedEOF）经 go doc 验证；损坏流测试手法见 §失败场景 |
| BACKUP-CLOSED-03 | 实现 `:543` 配置恢复逻辑（**按 D-01 校准为设备配置下发**，非 REQUIREMENTS 原文的 DB 事务化措辞） | §Architecture Patterns 恢复全流程 + scrapligo SendConfigs 验证行为（模块源码）+ RestoreConfig 集成方案 + 异步任务化先例 |
| BACKUP-CLOSED-04 | 新增 `config_backup_service_93_NN_test.go`（D-27 修正命名），覆盖三路径 + 失败场景（磁盘满/校验失败/事务回滚→按 D-28 四场景清单） | §79_06 测试基建（全部 helper 名 file:line 验证）+ §失败场景模拟手法表 + FileTransport e2e fixture 字节格式 |
| BACKUP-CLOSED-05 | `go test ./internal/services/...` 0 回归；端到端 备份→恢复→配置一致性校验通过（D-30 自动化断言链） | §Validation Architecture V3/V5；端到端断言链设计；TestBackupHandler_Restore stub 锁测试需同步改造（§Pitfalls P12） |
</phase_requirements>

## Project Constraints (from CLAUDE.md)

- **Handler-Service 模式**：handler 结构体 + service interface + 构造器注入；router 内 `SetupXxxRouter(r, core)` 装配
- **operlog 约定（强制）**：业务写操作 handler 必须在 success path 末尾、`response.Success` 之前 `operlog.Record(...)`；敏感端点用 `RecordWithBody`（本 phase D-18 依据；OperType 25 常量锁值由 `internal/utils/operlog/regression_test.go` AST 防线守护，D-13 承诺零改动）
- **pkg/constants 集中化**：超时常量进 `pkg/constants/timeouts.go`（time.Duration 强类型；赋 int 字段用 `int(constants.Xxx.Seconds())`）；禁止内联 `time.Minute * N` 字面量
- **迁移双线注册**：新 model 必须 (1) `internal/core/db/database.go` `MigrateModelList` AutoMigrate 注册（sqlite 缺表 family pattern 已重复 5 次）(2) UUID 主键走 BeforeCreate hook（BaseModel 无 DB default，PG 23502 教训）(3) `internal/core/db/migrations/migration_211_xxx.go` MigrateNNN 双方言分支注册（见 §Migration）
- **状态常量唯一真相源**：`internal/models/` 具名常量；本 phase 新任务状态枚举用 string（跟随 ConfigBackup BackupType/StorageType 风格），0/1 status 规则不适用（无 status 列语义）
- **`go build ./...` 0 错误纪律**：改 Go 代码后必须跑；逐文件修复不批量修
- **`go test ./internal/services/...` 0 失败**：BACKUP-CLOSED-05 验收线
- **atomic commit 纪律（v1.29 D-04）**：每个独立修复/功能一个 commit（D-33 两处顺手修各自独立 commit）
- **前端**：useEffect 依赖必须稳定（useMemo/useCallback 防无限轮询循环）；API 调用走 `@/lib/api` 包装函数；本地 vitest 测试禁目录名/绝对路径断言
- **临时文件**：根目录 `temp_*.go`/`test_*.go` 会导致 main redeclared，build 前清理

## Summary

Phase 93 的三处 TODO 中，:158/:206（压缩/解压）是纯标准库工作（`compress/gzip`，Go 1.24.5 本地验证 API 与错误形态），技术风险低但有三条决策绑定：`.conf.gz` 后缀双检查（D-23）、BackupSize 原始大小口径（D-24）、三条备份路径统一 helper（手动 CreateBackup:156 / 批量 BatchBackupDevices:241 固定 CompressLarge=true / 自动 createNewAutoBackup:389 现状无压缩分支）。

:543 恢复是本 phase 主战场：按 D-01 校准为"设备配置下发"，经异步任务化（D-15..D-22/D-34）。下发内核已由 scrapligo v1.4.0 模块源码逐行验证：`network.Driver.SendConfigs = AcquirePriv("configuration") + 泛型裸发`——进入配置模式由 scrapligo 按 platform 定义自动处理（huawei_vrp 发 `system-view`），**vendor map（D-06）的实际职责收窄为"退出配置模式的清理命令 + 厂商错误标记"**，与 portwrite V7"每端口后置 exit/quit"先例一致。RestoreConfig 的 executor 集成点推荐 `ExecuteCustom`（executor.go:161，done-channel 信号模式，修过 30s 假超时 bug 并有注释说明）。

异步任务执行在项目内有完整先例可抄：`DeviceInfoCollectionService`（DB 任务表 + string 状态枚举 + channel 队列 + worker goroutine + 启动恢复 + Enqueue 同设备 pending/running 去重——恰好是 D-08 互斥的现成模式）；detached context 先例在 `batch_orchestrator.go:64`（`context.WithTimeout(context.Background(), ...)`）。测试基建 79_06 五个 helper 全部按名验证可直接复用；SendConfigs e2e 的 fixture 字节格式由 portwrite testdata 真实 fixture（17 个，huawei+ruijie）给出模板。**无新增外部依赖**——gzip 是标准库，其余全部复用 go.mod 既有包。

**Primary recommendation:** 按 93-01（压缩/解压 + gzip helper 统一三路径）/ 93-02（RestoreConfig + 任务表 + 异步 service/handler + 前端最小改动）/ 93-03（93_NN 测试 + e2e + 断言链）/ 93-04（措辞修订 + D-33 两处顺手修 + CLAUDE.md Convention 收口）拆分；RestoreConfig 用 ExecuteCustom + vendor map 只放退出命令；互斥抄 Enqueue 去重模式；**必须处理崩溃残留 running 任务的互斥死锁问题（§Pitfalls P2）**。

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| gzip 压缩/解压 | API/Backend（services 层） | — | 文件存储分支的落盘/读盘细节，ConfigBackupService 私有 helper；handler 无感知 |
| 设备配置下发（RestoreConfig） | API/Backend（device 包） | — | 与 GetConfig 对称的 device 包能力；scrapligo 交互封装在 device 层，services 层不碰 wrapper |
| 恢复清洗（滤空行/注释行） | API/Backend（device 包） | — | D-03 锁定在 RestoreConfig 内部，与厂商 switch 同处 |
| 恢复任务状态机 | API/Backend（services 层 + models） | Database/Storage（新任务表） | 任务生命周期是业务语义；DB 表持久化状态 |
| 同设备互斥 | API/Backend（services 层） | Database/Storage（pending/running 查询） | 抄 DeviceInfoCollectionService.Enqueue 去重模式（DB 查询型互斥） |
| 恢复发起 operlog | API/Backend（handler 层） | — | CLAUDE.md 约定 handler success path 末尾 Record（D-18） |
| 任务进度查询端点 | API/Backend（handler 层） | — | 挂 `/backups` 组继承权限（D-31） |
| 恢复按钮异步交互 + 轮询 | Browser/Client（React） | API/Backend（任务查询端点） | useDiscoveryPolling 同款 3s setInterval 模式（D-16） |
| 版本链恢复记录 | Database/Storage（sys_config_backup） | API/Backend（service 层写入） | D-09 任务成功后插入；复用 ConfigBackup model |

## Standard Stack

### Core（零新增依赖 — 全部为 go.mod 既有包 + Go 标准库）
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `compress/gzip` | Go 1.24.5 stdlib | 压缩/解压（D-26 DefaultCompression） | 标准库，RFC 1952；`gzip.NewWriter`/`NewReader`/`ErrHeader`/`ErrChecksum` 经本地 `go doc` 验证 |
| `github.com/scrapli/scrapligo` | v1.4.0（go.mod:24） | SendConfigs 配置下发 + FileTransport 测试回放 | 项目既有依赖（portwrite 51-56 全链路验证）；模块源码验证 SendConfigs priv 语义 |
| `github.com/glebarez/sqlite` | v1.11.0（go.mod:12） | 测试用 sqlite file DB（newDB7906 模式） | 项目测试基建标配（79_06 先例） |
| `github.com/stretchr/testify` | 既有 | require/assert | 项目全部测试标配 |

### Supporting（既有项目基建，直接复用）
| Component | Location | Purpose | When to Use |
|-----------|----------|---------|-------------|
| `device.ExecuteCustom` | internal/device/executor.go:161-193 | RestoreConfig 的任务提交/等待骨架 | D-02 RestoreConfig 实现（done-channel 信号，成功/失败立即返回） |
| `device.SeedConnectionForTesting` / `NewPooledConnectionForTesting` | internal/device/e2e_helpers.go:76/:32 | 零 SSH 注入 FileTransport 连接 | D-29 e2e |
| `DeviceInfoCollectionService` 异步模式 | internal/services/device_info_collection_service.go:74-240 | DB 任务表 + 状态机 + worker + 启动恢复 + Enqueue 去重 | D-15/D-08 任务表与互斥的实现模板 |
| `operlog.Record` | internal/utils/operlog/operlog.go:215 | handler 发起时审计 | D-18（`RecordBackground`:282 存在但本期按 D-18 不用于任务完成记录） |
| `base.ApplySort` + 白名单 | internal/services/config_backup_service.go:437 | 任务列表端点排序 | D-31 任务查询端点分页排序 |
| `useDiscoveryPolling` | xingran-react-frontend/src/pages/network/discoveries/hooks/useDiscoveryPolling.ts | 3s setInterval 轮询 hook + running 态自动启停 + unmount 清理 + 配套 test | D-16 前端轮询模板 |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `ExecuteCustom` 包装 RestoreConfig | ExecuteMultipleOnDevice | exec 模式无配置模式语义（D-02 明确排除）；ExecuteCustom 闭包内可控 priv/清洗/fail-fast |
| device 包自写失败检测 | import portwrite 的 parseConfigError | **不可行**：parse_error.go 小写未导出，且 portwrite → device import 方向形成环（port_write_service.go:5 import device）；device 包需自带精简版 marker 检测（§Pitfalls P7） |
| gzip.NewWriterLevel | gzip.NewWriter | NewWriter 即 DefaultCompression（D-26 锁定默认级别，无需 Level 变体） |

**Installation:** 无 — 本 phase 零 `go get`、零 `npm install`。

**Version verification:** Go toolchain `go1.24.5 windows/amd64`（本地 `go version` 实测）；scrapligo v1.4.0 / glebarez sqlite v1.11.0（go.mod grep 实测）；Node v24.19.0（实测）。

## Package Legitimacy Audit

本 phase **不安装任何新外部包**（gzip = Go 标准库；scrapligo/testify/glebarez-sqlite 均为 go.mod 既有锁定依赖，已在本项目 1688+ 测试中长期验证）。无需 slopcheck 流程；无新增 checkpoint:human-verify 门。

## Architecture Patterns

### System Architecture Diagram — 恢复全流程（D-01..D-22/D-34 综合，CONTEXT specifics 段已验证可行性）

```
前端备份页 Modal（D-19）
    │  POST /network/backups/:id/restore {deviceId}
    ▼
BackupHandler.Restore（backup_handler.go:266，现同步 stub → 改异步）
    │  binding 校验（backupId path + deviceId body）
    │  operlog.Record（"配置备份", OperTypeUpdate）  [D-13/D-18]
    ▼
ConfigBackupService（或新 RestoreTaskService，planner 定布局）
    │  ① backup.DeviceID == deviceId 校验          [D-04]（不一致 4xx 拒绝）
    │  ② 互斥检查：同 device_id 存在 pending/running 任务 → "恢复进行中"  [D-08]
    │     （先例：DeviceInfoCollectionService.Enqueue:133-163 同款 IN 查询去重）
    │  ③ 创建任务记录 status=pending
    ▼
    │  立即返回 { taskId }（response.Success）
    ▼
异步 goroutine（detached context！）                    [P1]
    │  status=pending → running
    │  ④ 恢复前自动备份 CreateBackup(BackupType=auto, ChangeReason="恢复前自动备份")
    │     失败 → 任务 failed（错误信息），中止（无回退退路不下发）  [D-10/D-14]
    │  ⑤ 读备份内容 GetBackupContent（内部走解压路径）  [BACKUP-CLOSED-02]
    │  ⑥ DeviceExecutor.RestoreConfig(ctx, deviceID, config, timeout)
    │        │  GetDevice → vendor                       [GetConfig 先例 executor.go:278]
    │        │  基础清洗：滤空行 / `!` / `#` 行            [D-03]
    │        │  ExecuteCustom：
    │        │    wrapper.SendConfigs(清洗后行)            [scrapligo 自动 AcquirePriv(configuration)]
    │        │      fail-fast：第 i 行 err/Failed/marker 命中 → 停  [D-12]
    │        │    末尾追加 vendor 退出命令（quit/exit）      [D-06/P6]
    │        │  返回 RestoreResult{已发 N/M、失败行、耗时}
    │        │  任一行失败 → 任务 failed + 进度留痕         [D-12/D-28③]
    │  ⑦ 回读 GetConfig → calculateHash 比对备份 ConfigHash
    │        不一致 → hashMatched=false + 警告日志，不算失败  [D-11]
    │  ⑧ sys_config_backup 插入恢复记录（ChangeReason="恢复自版本 N"）  [D-09]
    │  ⑨ 任务 success（result JSON 存 RestoreResult）
    ▼
前端轮询任务查询端点（3s，running 态轮询，Modal 展示进度/结果）  [D-16/D-19]

读取侧（压缩闭环，独立于恢复流程）：
CreateBackup:156 大文件分支 ──┐
BatchBackupDevices:241（固定 CompressLarge=true）─┤→ gzipCompress helper ─→ .conf.gz 落盘
createNewAutoBackup:389（现状无压缩 → D-25 补齐）─┘   BackupSize=len(原始 config)  [D-24]
GetBackupContent:205 → Compressed && strings.HasSuffix(path,".gz") 双检查 → gzipDecompress  [D-23]
```

### Recommended Project Structure（新增/改动文件布局，命名均在 planner discretion）

```
internal/
├── device/
│   ├── executor.go                  # +RestoreConfig（与 GetConfig 对称；ExecuteCustom 骨架）
│   └── restore_vendor_map (可并入 executor.go 或独立文件)   # D-06 vendor→退出命令/错误标记 map
├── models/
│   └── config_restore_task.go       # D-15 任务 model（string 状态枚举 + BeforeCreate UUID）
├── services/
│   ├── config_backup_service.go     # :158/:206 gzip helper 实现 + :543 异步化 + D-33 sort 白名单删 status
│   ├── config_restore_task_service.go (命名 planner 定)   # 任务 CRUD/互斥/状态流转
│   └── config_backup_service_93_NN_test.go              # D-27 测试（压缩/解压/清洗/e2e/4 失败场景/断言链）
├── api/v1/network/
│   ├── backup_handler.go            # Restore 改异步语义 + 任务查询 handler（同文件或新文件）
│   ├── backup_handler_test.go       # TestBackupHandler_Restore stub 锁测试同步改造（P12）
│   └── network_router.go            # /backups 组内挂任务查询端点（D-31）
├── core/db/
│   ├── database.go                  # MigrateModelList 注册新 model（:511 ConfigBackup 附近）
│   └── migrations/migration_211_config_restore_task.go   # Migrate211（双方言分支注册）
└── pkg/constants/timeouts.go        # +RestoreConfigTimeout（D-05）

xingran-react-frontend/src/pages/network/backups/
├── index.tsx                        # 恢复按钮异步交互 + Modal 进度/结果（D-19）
├── hooks/useRestoreTask.ts (+test)  # 轮询 hook（useDiscoveryPolling 模板）
└── types.ts                         # taskId/RestoreResult 类型
```

### Pattern 1: scrapligo SendConfigs 确切行为（D-02 根基，模块源码验证）

**What:** `scrapligo v1.4.0` `network.Driver.SendConfigs(configs)` 的执行序列（模块缓存源码逐行验证）：

1. `AcquirePriv("configuration")` — 按 platform 定义自动进入配置模式（huawei_vrp 的 configuration priv escalation 命令即 `system-view`；cisco 系为 `configure terminal`）。设备无 configuration priv level 时直接报错
2. `d.Driver.SendCommands(configs, ...)` — **调用的是内嵌泛型驱动的方法（裸发，无 priv 舞步）**；逐条 sendCommand，支持 `StopOnFailed` 操作选项（原生 fail-fast 语义，但项目 wrapper 未暴露该选项——见下）
3. 失败标记：每条 response 带 `Failed bool`（platform `failed_when_contains` 命中或通道错误）；`MultiResponse.Failed` 为任一失败

**When to use:** RestoreConfig 下发内核；与 portwrite 的差异——portwrite 每端口一次 SendConfigs（2-3 行），restore 是一次全量配置（可达千行）。

**项目 wrapper 的形态注意**：`ScrapliWrapper.SendConfigs`（scrapli_wrapper.go:614-637）内部是 `for _, cfg := range configs { w.driver.SendConfig(cfg) }` —— **每行各走一次完整的 scrapligo SendConfig（= 每行各一次 AcquirePriv）**，且中途失败返回 `(已收 responses, error)` 部分结果。这意味着：
- 每行失败可通过返回的 `responses[i].Failed` + err 定位（D-12 进度留痕有数据源）
- 千行配置 = 千次 priv 舞步，效率偏低但**语义安全**（scrapli 每行确保 priv 正确，自动纠正 view 漂移）；RestoreConfig 直接复用此 wrapper 方法即可，无需绕开
- 若需行级 fail-fast：wrapper 中途 err 即 return，外层拿 `len(responses)` 计已发行数（D-12）

```go
// Source: scrapligo v1.4.0 driver/network/sendconfigs.go + sendconfig.go（模块缓存）
func (d *Driver) SendConfigs(configs []string, opts ...util.Option) (*response.MultiResponse, error) {
    targetPriv := op.PrivilegeLevel
    if targetPriv == "" { targetPriv = defaultConfigurationPrivLevel } // "configuration"
    err = d.AcquirePriv(targetPriv)   // ← 进入配置模式：platform 定义自动处理
    if err != nil { return nil, err }
    return d.Driver.SendCommands(configs, opts...)  // ← 泛型裸发（无二次 priv 检查）
}
// SendConfig(config) = strings.Split(config, "\n") → SendConfigs（多 Response 折叠单 Response）
```

### Pattern 2: RestoreConfig 集成 ExecuteCustom（推荐注入点）

**What:** 与 GetConfig 对称的厂商感知方法，内部用 ExecuteCustom 提交任务。
**When to use:** D-02 唯一下发入口。

```go
// Source: internal/device/executor.go:161-193（ExecuteCustom）+ :267-300（GetConfig vendor switch）组装
func (e *DeviceExecutor) RestoreConfig(ctx context.Context, deviceID, config string) (*RestoreResult, error) {
    pool := e.scheduler.GetConnectionPool()
    dev, err := pool.GetDevice(deviceID)          // 与 GetConfig:278 同款
    if err != nil { return nil, fmt.Errorf("获取设备信息失败: %w", err) }

    lines := cleanConfigLines(config)             // D-03：滤空行 / `!` / `#`（TrimSpace 后前缀判断）
    if len(lines) == 0 { return nil, fmt.Errorf("备份内容无有效配置行") }

    fullCmds := append(lines, exitConfigCmd(dev.Vendor))  // D-06/P6：末尾退出配置模式
    var sent int
    var failedLine string
    execErr := e.ExecuteCustom(ctx, deviceID, func(_ context.Context, pc *PooledConnection) error {
        responses, sendErr := pc.GetWrapper().SendConfigs(fullCmds)
        sent = len(responses)                     // 部分结果也计数 → D-12 进度
        if sendErr != nil {
            if sent < len(fullCmds) && sent > 0 { failedLine = fullCmds[sent] }
            return sendErr
        }
        for i, r := range responses {             // wrapper 不透传 scrapligo Failed → 逐条查
            if r != nil && r.Failed { sent = i; failedLine = fullCmds[i]; return fmt.Errorf(...) }
        }
        return nil
    }, constants.RestoreConfigTimeout)            // D-05 独立超时
    // 组装 RestoreResult{TotalLines, SentLines, FailedLine, HashAfter, HashMatched...}  [D-07/D-11/D-12]
    ...
}
```

### Pattern 3: 异步任务 + 同设备互斥（DeviceInfoCollectionService 先例）

**What:** DB 任务表 + string 状态机 + 入队去重；本 phase 无需 channel 队列（restore 量级极小，直接 `go func` + detached context）。
**When to use:** D-15/D-08/D-17。

```go
// Source: internal/services/device_info_collection_service.go:133-163（Enqueue 去重 = D-08 互斥模板）
var existing models.ConfigRestoreTask
err := s.db.Where("device_id = ? AND status IN (?)", deviceID,
    []models.RestoreTaskStatus{models.RestoreTaskStatusPending, models.RestoreTaskStatusRunning}).
    First(&existing).Error
if err == nil {
    return nil, fmt.Errorf("恢复进行中")   // D-08：不排队，立即报错
}
task := &models.ConfigRestoreTask{DeviceID: deviceID, Status: models.RestoreTaskStatusPending, ...}
s.db.Create(task)
go s.runRestore(context.WithTimeout(context.Background(), constants.RestoreConfigTimeout), task.ID)
    // ↑ detached context（P1）——绝不能绑 HTTP request ctx
```

### Anti-Patterns to Avoid
- **异步 goroutine 绑 HTTP ctx**：`c.Request.Context()` 在 response 返回后即取消，任务必死于第一步（portwrite 批量 detached 先例 batch_orchestrator.go:64 是正确姿势）
- **把退出配置命令写成 `end`**：portwrite V5 教训（port_write_service.go:403-422 注释）——`end` 退到 privileged EXEC 破坏 scrapli priv 跟踪；华为/H3C 用 `quit`、锐捷/思科系用 `exit`（只退一级）；且**进入**配置模式不要自己发 `system-view`——scrapligo AcquirePriv 负责（重复发送会错位）
- **把 93 测试写进 79_06 文件**：D-27 锁定新文件；职责分离
- **Close FileTransport driver**：close 时对已耗尽 fixture 的读取永久阻塞（78-03 S-2 / port_write_e2e_test.go:42-48 注释）；测试 harness 不调 Close，pool.Close 用 watchdog 兜底（newExecutor7906:154-162 模式）
- **BackupSize 记压缩后大小**：D-24 锁定 `len(config)` 原始口径——`GetBackupStatistics` 的 `SUM(backup_size)` totalSize 口径依赖它
- **恢复记录复用 BackupTypeManual 时不跑锁值测试**：若新增 BackupType 常量（如 restore），需同步 models 锁值测试（CONTEXT discretion 已预告）

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| gzip 压缩/解压 | 自写 deflate/字节流处理 | `compress/gzip` 标准库 | RFC 1952 细节（header/CRC/ISIZE）标准库全包；损坏流错误形态现成 |
| 配置模式进入 | 手发 `system-view`/`conf t` | scrapligo `SendConfigs` 内建 AcquirePriv | platform 定义精确处理 priv 舞步 + prompt 跟踪；手发会在 fixture/真机上双重错位 |
| 同设备互斥 | 内存 sync.Mutex 单例 | 任务表 `status IN (pending,running)` 查询（Enqueue:133 先例） | 内存锁重启即失效且多副本无效；DB 查询天然持久、与任务表同源 |
| 失败命令检测 | 逐厂商自写响应解析 | wrapper `Response.Failed` + 精简 marker 集（portwrite rejectionMarkers 精神） | scrapligo platform `failed_when_contains` 已覆盖主要错误形态；marker 只兜底 |
| 任务状态机/启动恢复 | 自造状态字段整数编码 | string 枚举 + EnrichmentTask 先例（pending/running/success/failed） | CONTEXT 锁定 string 风格；先例含 Status/StartedAt/CompletedAt/ErrorMessage 全套字段 |
| 前端轮询 | 手写 useEffect+setInterval 散逻辑 | useDiscoveryPolling 抽象（3s / running 态启停 / cleanup / 带 test） | CLAUDE.md useEffect 依赖稳定性要求的现成合规实现 |

**Key insight:** 本 phase 没有一处需要引入新库——难点全部在"正确复用既有骨架的位置与形态"（ExecuteCustom vs Submit、wrapper.SendConfigs 的 per-line priv 语义、fixture 字节布局），而非算法。

## Common Pitfalls

### Pitfall 1: 异步任务绑定 HTTP request context
**What goes wrong:** `go func() { svc.runRestore(c.Request.Context(), ...) }` —— handler 返回 `{taskId}` 后 ctx 被 gin 取消，任务在恢复前备份的 GetConfig 处即报 context canceled，永远 failed。
**Why it happens:** 同步思维惯性；gin 的 request ctx 生命周期 = 请求生命周期。
**How to avoid:** 任务 goroutine 用 `context.WithTimeout(context.Background(), constants.RestoreConfigTimeout)`（batch_orchestrator.go:64 detached 先例）。
**Warning signs:** 本地测试任务秒 failed、错误信息含 "context canceled"。

### Pitfall 2: 崩溃残留 running 任务 → 该设备互斥永久死锁
**What goes wrong:** 进程在任务 running 态崩溃/重启 → 任务表残留 `status=running` 行 → D-08 互斥查询（IN pending,running）永久命中 → 该设备永远报"恢复进行中"，只能手改数据库。
**Why it happens:** D-20 明确不做清理 cron、D-34 不支持取消——但"启动时收敛孤儿任务"不属于清理，是状态机自洽性问题。
**How to avoid:** planner 在 discretion 内三选一：① 启动时把 running 任务一次性标 failed（`recoverPendingTasks` 反向先例，device_info_collection_service.go:217）② 互斥查询忽略 `updated_at` 早于 `2×RestoreConfigTimeout` 的 running 行（陈旧容忍）③ CONTEXT 已锁定的 D-22 语义文档化（"重新发起前手动 DB 修"——不推荐）。推荐 ①（一次启动收口，代码 <10 行）。
**Warning signs:** 重启后同设备 restore 必现"恢复进行中"。

### Pitfall 3: gzip.Writer 不 Close 就取字节
**What goes wrong:** `w.Write(data); buf.Bytes()` 拿到的是空/不完整流——gzip 尾部（CRC + ISIZE）只在 Close 时写出（go doc: "compressed bytes are not necessarily flushed until the Writer is closed"）。
**How to avoid:** `defer w.Close()` 不够（顺序），必须显式 `w.Close()` 后再 `buf.Bytes()`；解压侧 `io.ReadAll(r)` 后如需严格校验再查 `ErrChecksum`（ReadAll 内部已验，CRC 错返回 ErrChecksum）。
**Warning signs:** 解压 roundtrip 测试报 ErrUnexpectedEOF 或 CRC mismatch。

### Pitfall 4: 压缩实现后 `TestCbk7906_CreateBackup_LargeConfigGoesToFile` 断言破裂
**What goes wrong:** 79_06 该测试断言 `os.ReadFile(result.FilePath)` 内容 == 原始 config 明文；CompressLarge=false 时无影响，但 D-25 改造 createNewAutoBackup 后 `TestCbk7906_AutoBackupSmartSkip`（大文件路径若被触发）同样受影响。任何现存断言"文件内容==明文"的测试在压缩路径激活后都会失败。
**How to avoid:** 改造时 grep `config_backup_service_79_06_test.go` 中 `os.ReadFile(result.FilePath)` 断言点；79_06 用例 CompressLarge 均未显式置 true（CreateBackup 请求缺省 false）——手动路径不受影响，**auto 路径（D-25）是唯一会碰到现存断言的改造点**；必要时按 D-28 在 93_NN 内自建压缩路径断言而不动 79_06。
**Warning signs:** 实现 D-25 后 `go test ./internal/services/ -run TestCbk7906` 出红。

### Pitfall 5: `.gz` 双检查的方向性
**What goes wrong:** 只查 `Compressed` 标志：手工把非压缩文件标成 compressed 的脏数据 → 解压报错；只查 `.gz` 后缀：FilePath 后缀与实际状态漂移。D-23 要求两条件**同时成立才解压**；任一不匹配（如 Compressed=true 但无 .gz 后缀）应显式报错而非静默按明文返回。
**How to avoid:** `if backup.Compressed && strings.HasSuffix(backup.FilePath, ".gz") { decompress } else if backup.Compressed || strings.HasSuffix(...) { return error("备份压缩状态不一致") }`（保守分支，planner 可简化为日志降级）。
**Warning signs:** GetBackupContent 对混合状态数据 panic/乱码返回。

### Pitfall 6: 配置模式 view 漂移与退出命令选择
**What goes wrong:** 全量配置含大量 `interface X ... #` 段——发完最后一条后设备停在 config（或 config-if）view；若不退出：后续同连接 GetConfig 的 `show running-config` 在错误 view 下执行失败或输出错乱。若退出用 `end`：破坏 scrapli CurrentPriv 跟踪（V5 教训）。
**How to avoid:** 末尾追加厂商"退一级"命令：华为/H3C `quit`、锐捷/思科系 `exit`（VendorExitViewCmd:186-193 同款 map，D-06 vendor map 的主体）；注意 safety net——scrapligo `SendCommand`/`SendCommands`（network 层）在 CurrentPriv≠默认时会自动 AcquirePriv 回 exec（模块源码 sendcommand.go 已验证），但**不要依赖**该兜底（view 漂移时 CurrentPriv 可能假正确）。
**Warning signs:** restore 后同连接 diff/读回报"命令不识别"。

### Pitfall 7: device 包不能 import portwrite 的 parseConfigError
**What goes wrong:** 想复用失败检测 → import cycle（portwrite → device 已存在）；parseConfigError 又是小写未导出。
**How to avoid:** RestoreConfig 内写精简版检测：优先 `resp.Failed`（scrapligo platform failed_when_contains 已标记），再对小结果集做关键词兜底（`Unrecognized command`/`Unknown command`/`Invalid`/`% Error` 等可从 parse_error.go:73-82 抄精神但独立维护）；device 包不 import portwrite/portcollection。
**Warning signs:** 编译报 import cycle not allowed。

### Pitfall 8: 测试工作目录纪律（Windows 文件占用）
**What goes wrong:** 用 `t.TempDir()` 承载 getBackupDir 的相对路径输出 → applogger 打开 `./logs/app.log` 后 t.TempDir RemoveAll 在 Windows 报 "file in use" 测试失败。
**How to avoid:** 照抄 `cbk7906Chdir`（79_06:63-70）——进程级 `os.MkdirTemp` + `t.Chdir` + best-effort cleanup；FileTransport fixture 同理不进 t.TempDir。
**Warning signs:** Windows CI 测试 cleanup 阶段红、Linux 本地绿。

### Pitfall 9: 新 model 缺任一注册线 → 启动/CI 缺表
**What goes wrong:** 只写 model 不注册 → sqlite（CI 测试与 dev 环境）缺表 23502/no such table（family pattern 已重复 5 次）；只注册 AutoMigrate 不写 BeforeCreate → PG 下 uuid 列无 default 的 23502。
**How to avoid:** 三件套一次做齐：model BeforeCreate UUID hook（config_backup.go:51-64 模板）→ `database.go` MigrateModelList 注册 → `migration_211` 双方言分支（PG advisory-lock 块 + sqlite else 分支都注册，database.go:808-890 布局）。注意 sqlite 分支若涉 GORM ALTER 需先 DROP 依赖视图（MEMORY xingran-sqlite-view-dependency-gotcha）——新表无此问题。
**Warning signs:** `go test ./internal/core/db/...` 的 AutoMigrate 端到端测试（database_test.go:142 "全量模型迁移必须成功"）红——该测试会捕获漏注册。

### Pitfall 10: TestBackupHandler_Restore 锁死了 stub 行为
**What goes wrong:** `backup_handler_test.go:233` 子测试 `restore_is_a_documented_stub` 断言 restore **必须**失败且含"配置恢复功能待实现"——实现功能后此测试必红。
**How to avoid:** 改造 handler 时同步重写该子测试为异步语义（响应含 taskId、任务可查询）；grep 该文件确认没有其他对 `RestoreBackup` 旧签名 `error` 单返回值的编译依赖。
**Warning signs:** `go test ./internal/api/v1/network/ -run TestBackupHandler` 红。

### Pitfall 11: 前端轮询 hook 的 useEffect 依赖不稳
**What goes wrong:** 轮询对象/回调未 memoize → interval 反复重建/请求风暴（CLAUDE.md useEffect CRITICAL 条款）。
**How to avoid:** 照抄 useDiscoveryPolling（依赖 `[discoveries, onPoll]`，调用方用 useCallback 固化 onPoll）；轮询仅在有 running 态任务时启动（D-16 语义天然满足）。
**Warning signs:** Network 面板同请求毫秒级重复。

### Pitfall 12: 清洗规则误伤
**What goes wrong:** 华为配置中 `#` 是段分隔符（非注释语义）——按行首过滤正确；但若把含 `#`/`!` 的**中段**（如 description 内）整行过滤则可能误删。另外 `sysname X` 行会改设备主机名 → 后续所有 prompt 变化（scrapli 默认 prompt pattern 对主机名通配，理论无碍，但 e2e fixture 必须覆盖）。
**How to avoid:** 清洗规则 = TrimSpace 后 `== ""` 或 前缀 `#`/`!` 才过滤（CONTEXT D-03 锁定的最小集）；93 e2e fixture 的配置内容里刻意放一条 `sysname` 行验证 prompt 漂移韧性。
**Warning signs:** e2e fixture 回放卡死/超时（prompt 不匹配时 scrapli 读到 EOF）。

## Code Examples

### gzip 压缩/解压 helper（D-23/24/26 全语义）
```go
// Source: go doc compress/gzip（Go 1.24.5 本地 toolchain 验证）
// gzipCompress 返回 config 的 gzip 压缩字节（DefaultCompression，D-26）。
func gzipCompress(config string) ([]byte, error) {
    var buf bytes.Buffer
    w := gzip.NewWriter(&buf)          // NewWriter 即 DefaultCompression
    if _, err := w.Write([]byte(config)); err != nil {
        return nil, err
    }
    if err := w.Close(); err != nil {   // 必须先 Close（Pitfall 3）
        return nil, err
    }
    return buf.Bytes(), nil
}

// gzipDecompress 解压 .gz 字节流；损坏数据错误形态：
//   随机字节   → gzip.NewReader 即返回 gzip.ErrHeader
//   截断流     → io.ReadAll 返回 io.ErrUnexpectedEOF
//   CRC 损坏   → io.ErrAllol 返回 gzip.ErrChecksum
func gzipDecompress(data []byte) (string, error) {
    r, err := gzip.NewReader(bytes.NewReader(data))
    if err != nil {
        return "", fmt.Errorf("备份文件损坏: %w", err)
    }
    defer r.Close()
    out, err := io.ReadAll(r)
    if err != nil {
        return "", fmt.Errorf("备份文件解压失败: %w", err)
    }
    return string(out), nil
}
```

### CreateBackup :158 落点改造（D-23/D-24 语义）
```go
// Source: internal/services/config_backup_service.go:150-168（现 TODO 处）
fileName := fmt.Sprintf("%s_v%d_%s.conf", req.DeviceName, newVersion, timestamp)
if req.CompressLarge {
    fileName += ".gz"                                          // D-23 命名
}
filePath := filepath.Join(backupDir, fileName)

content := []byte(config)
if req.CompressLarge {
    compressed, err := gzipCompress(config)                    // TODO 删除点
    if err != nil { return nil, fmt.Errorf("压缩备份内容失败: %w", err) }
    content = compressed
}
if err := os.WriteFile(filePath, content, 0644); err != nil {  // 写失败 → D-28① 场景
    return nil, fmt.Errorf("写入备份文件失败: %w", err)
}
backup.FilePath = filePath
backup.Compressed = req.CompressLarge                          // BackupSize 保持 len(config) 原始口径（D-24）
```

### SendConfigs e2e fixture 字节格式（portwrite 真实 fixture 实测）
```
# Source: internal/services/portwrite/testdata/huawei_shutdown_success.fixture（cat -A 逐行实测）
<Huawei>                              ← driver.Open banner
screen-length 0 temporary             ← open 回显
<Huawei>                              ← post-open prompt
system-view                           ← SendConfigs 的 AcquirePriv(configuration) 上升命令
[Huawei]                              ← 配置模式 prompt
sysname Restored-Switch               ← 配置行 1（建议 93 fixture 包含，验证 prompt 漂移）
[Huawei]                              ← prompt（主机名变了）
interface GE0/0/1                     ← 配置行 2
[Huawei-GE0/0/1]                      ← view 变化 → 新 prompt
shutdown                              ← 配置行 3
[Huawei-GE0/0/1]                      ← ...
quit                                  ← 退出配置模式（D-06 vendor map 追加项）
[Huawei]
return                                ← on-close 操作消耗（79_06 fixture 的 8 个 spare prompt 同理）
<Huawei>
```
规则：**每条命令 = 1 行回显 + 1 行新 prompt**（回显行在输入行之前或重合由 prompt strip 决定——以 fixture 原文为准逐字节构造）；ReadSize=1 逐行消费；结尾留 spare prompt 给 close 读取。

### 失败场景模拟手法（D-28 四场景，对照 79_06/portwrite 现有手法）
| 场景 | 手法 | 先例 |
|------|------|------|
| ① 写文件失败（磁盘满语义） | `t.Chdir` 后构造 MkdirAll/WriteFile 必败路径（Windows 文件名含 `:`/`?` 非法字符，或把目标目录预建为只读 `0500`） | 79_06 error-path 模式（TestCbk7906_CreateBackup_MissingDevice 的前置失败风格） |
| ② 损坏 gzip 字节流 | 直接 `os.WriteFile(path, []byte{0x1f, 0x00, random...})`（坏 magic → ErrHeader）或 gzip 压缩后 `data[:len(data)/2]` 截断（→ ErrUnexpectedEOF），FilePath 指向该文件、Compressed=true | §gzip 错误形态（go doc 验证）；无需磁盘技巧 |
| ③ 下发中断 → 任务 failed + 进度留痕 | e2e：fixture 在第 N 条配置行后缺失 prompt/EOF，或业务行回显错误 marker（portwrite `huawei_device_rejected.fixture` 同款）→ 断言任务 failed、result JSON 含 SentLines=N/Total=M、FailedLine | port_write_e2e_test.go TestE2E_Device_Rejected 先例 |
| ④ DB 写入失败 → 记录回滚 | 对 AutoMigrate 过的 sqlite 注入约束冲突（如超长列值触发 sqlite error），断言任务/备份记录无半写状态；或直接关闭 sqlDB 连接触发失败 | 79_06 既有 DB 手法 |

### 任务查询端点（D-31 挂靠形态）
```go
// Source: internal/api/v1/network/network_router.go:153-174（/backups 组内追加，继承组级 4 权限点）
backups.POST("/restore-tasks/list", restoreTaskHandler.List)       // 命名 planner 定
backups.POST("/restore-tasks/:id", restoreTaskHandler.GetByID)
// handler：operlog 不需要（查询不记审计）；分页排序走 base.ApplySort 白名单
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Restore 同步 stub（"配置恢复功能待实现"） | 异步任务化（D-15..D-22） | 本 phase | handler 响应 shape 变化（taskId）；TestBackupHandler_Restore 重写 |
| portwrite V5 `end` 退出视图 | V7 后置 exit/quit + prompt yaml 修复 | 2026-07-08（v1.19） | D-06 vendor map 必须遵循 V7 结论（quit/exit 只退一级） |
| interface{} closure GetOrSet | base.GetOrSetJSON[T]（Phase 92） | v1.29 | 与本 phase 无关——ConfigBackup 不在业务缓存体系（CONTEXT deferred 确认） |
| compress/gzip API | 稳定（NewWriter/NewReader 自 Go 1.0 语义未变；Go 1.24 现行） | — | 无版本适配风险 |

**Deprecated/outdated:**
- `ExecuteMultipleOnDevice` 用于配置下发：exec 模式无配置模式语义（D-02 排除）
- 手写 gzip 头解析/`gzip.NewWriterLevel` 指定级别：D-26 锁定 DefaultCompression

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `RestoreConfigTimeout` 建议值量级（portwrite singlePortTimeout 与 batch detached 为参照，全量配置下发千行级可能需要分钟级）；**D-05 只锁独立常量，值为 planner discretion** | Pattern 2 / Pitfalls P1 | 偏小 → 大配置真机恢复被误判超时（但 D-12 fail-fast + D-22 重发可恢复）；偏大 → 失败任务滞留 running 更久（放大 P2） |
| A2 | `Migration 211` 为下一个可用编号（实测 210 已占用，archive 目录未逐卷核对是否有 >210 的保留位） | Summary / 结构布局 | 极低——若冲突编译/启动测试立即暴露 |
| A3 | scrapligo 默认 huawei_vrp prompt pattern 对 `sysname` 引起的主机名变化通配（ruijie 需要 patch yaml 的教训提示 prompt pattern 是 platform 定义的；huawei 默认 pattern 未逐行核对 yaml） | Pitfalls P12 | e2e fixture 若不覆盖 sysname 行则真机恢复时 prompt 漂移风险未被发现——**93 e2e fixture 应包含 sysname 行实测验证**（已写入 P12 规避） |
| A4 | 批量/自动备份的 `DeviceName` 进入文件名（config_backup_service.go:152/385）无路径穿越防护——**现存行为，不在本 phase 范围**（scope constrainment 条款） | — | 低——若 planner 认为顺手修需用户确认（现 CONTEXT 未授权第三处顺手修） |
| A5 | 互斥推荐方案①（启动时 running→failed 收敛）符合 D-20"不做清理"的精神边界（属状态机自洽而非清理 cron）；若用户判定其为 D-20 违例，退回方案②陈旧容忍 | Pitfalls P2 | 方案错选 → 违反用户决策边界（planner 应在 plan 中显式标注此 discretion 引用） |

## Open Questions

1. **任务执行失败后的任务行是否需要可见的错误详情分页查询？**
   - What we know: D-19 只要求 Modal 内进度/结果展示；任务表含 ErrorMessage/result JSON。
   - What's unclear: 详情展示走 `GET /restore-tasks/:id` 单条查询是否足够（够——量级小），列表端点是否本期必需（D-17 说"列表/详情"）。
   - Recommendation: 两个端点都做（成本 <30 行），列表页暂无前端消费方也保留（D-17 原文锁了"列表/详情"）。

2. **恢复记录的 BackupType 表达（discretion 已预授权）**
   - What we know: 复用 auto/manual vs 新增常量；`GetBackupStatistics` 按 BackupTypeAuto/Manual 计数——新增 `restore` 类型会引入第三计数维度（前端统计卡片需同步）。
   - Recommendation: **复用 BackupTypeManual** + ChangeReason 区分（"恢复自版本 N"）——统计链路零改动、零锁值测试新增；planner 可直接采纳。

3. **93 测试对真机语义的覆盖边界**
   - What we know: D-30 端到端 = FileTransport 自动化；真机 UAT deferred。
   - What's unclear: 无。
   - Recommendation: e2e 断言链覆盖"hash 一致"即满足 BACKUP-CLOSED-05；真实设备回读 hash 不一致属 D-11 警告分支，用 fixture 构造不一致场景补一条警告断言即可。

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | 全部后端工作 | ✓ | go1.24.5 windows/amd64（实测） | — |
| scrapligo | RestoreConfig 下发 + FileTransport e2e | ✓ | v1.4.0（go.mod 锁定） | — |
| glebarez/sqlite | 93_NN 测试 DB | ✓ | v1.11.0（go.mod） | — |
| testify | 测试断言 | ✓ | go.mod 既有 | — |
| Node.js | 前端最小改动 + hook 测试 | ✓ | v24.19.0（实测，满足 CLAUDE.md 24+） | — |
| vitest | useRestoreTask hook 测试 | ✓ | 4.x（package.json） | — |
| PostgreSQL | 生产方言分支 | 运行时依赖 | — | CI/tests 全部走 sqlite 分支；Migrate211 双分支注册即合规 |

**Missing dependencies with no fallback:** 无
**Missing dependencies with fallback:** 无

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing + testify（后端）；vitest 4.x（前端 hook） |
| Config file | 无独立配置（go test 惯例）；前端 vitest（xingran-react-frontend/package.json `test` script） |
| Quick run command | `go test ./internal/services/ -run 'TestCbk93' -v`（93_NN 新测试快速档） |
| Full suite command | `go test ./internal/services/...`（BACKUP-CLOSED-05 验收）；整库 `go test ./...` + `go build ./...` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| BACKUP-CLOSED-01 | 大配置 CompressLarge=true → .conf.gz 落盘 + Compressed 标志 + BackupSize=原始大小；false → 现状明文 | unit（t.Chdir 隔离 + threshold=0 技巧走文件分支） | `go test ./internal/services/ -run 'TestCbk93.*Compress' -v` | ❌ Wave 0（93_NN 新建） |
| BACKUP-CLOSED-02 | GetBackupContent 双检查解压 roundtrip；损坏流（坏 header/截断）→ error | unit | `go test ./internal/services/ -run 'TestCbk93.*Decompress' -v` | ❌ Wave 0 |
| BACKUP-CLOSED-03 | RestoreBackup 异步化：校验（D-04）→ taskId 返回 → 任务流转 pending→running→success/failed；RestoreConfig 厂商感知 + 清洗 + fail-fast | unit + FileTransport e2e（SeedConnectionForTesting 零 SSH） | `go test ./internal/services/ -run 'TestCbk93.*(Restore|E2E)' -v` + `go test ./internal/device/ -run Restore -v` | ❌ Wave 0 |
| BACKUP-CLOSED-04 | D-28 四失败场景：写失败/损坏流/下发中断留痕/DB 失败回滚 | unit + e2e | `go test ./internal/services/ -run 'TestCbk93.*Fail' -v` | ❌ Wave 0 |
| BACKUP-CLOSED-05 | 端到端断言链 CreateBackup → GetBackupContent → restore 任务 → hash 一致；`go test ./internal/services/...` 0 回归 | integration（D-30 自动化验收） | `go test ./internal/services/ -run 'TestCbk93.*EndToEnd' -v`；`go test ./internal/services/...` | ❌ Wave 0 |
| D-31/D-17 | 任务查询端点挂 /backups 组 + taskId 响应 shape | handler test（改造 TestBackupHandler_Restore） | `go test ./internal/api/v1/network/ -run TestBackupHandler -v` | ⚠️ 存在但需重写 stub 锁（P10） |
| D-16/D-19 | useRestoreTask 轮询启停/cleanup | 前端 vitest（useDiscoveryPolling.test.tsx 模板） | `cd xingran-react-frontend && npx vitest run src/pages/network/backups/hooks/useRestoreTask.test.tsx` | ❌ Wave 0 |
| D-13/D-33/D-32 | operlog 25 常量锁值零回归；sort 白名单删 status；CLAUDE.md Convention 段 | 既有回归测试 + grep 断言 | `go test ./internal/utils/operlog/ -v`；`go test ./internal/models/ -run TestStatusConstants -v` | ✅ 既有防线直接复用 |

### Sampling Rate
- **Per task commit:** `go build ./...` + 相关包 quick 档（<30s）
- **Per wave merge:** `go test ./internal/services/...`（BACKUP-CLOSED-05 线）
- **Phase gate:** `go test ./...` 全绿 + `npm run type-check` + `npm run lint` + 前端 `npm run test`（若前端改动落地）→ `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `internal/services/config_backup_service_93_NN_test.go` — 全部 5 个 BACKUP-CLOSED 测试落点（D-27 新文件；helper 复用 79_06 五件套 + newExecutor7906）
- [ ] `internal/api/v1/network/backup_handler_test.go` — `restore_is_a_documented_stub` 子测试重写为异步契约（P10，非新建）
- [ ] `xingran-react-frontend/src/pages/network/backups/hooks/useRestoreTask.test.tsx` — 轮询 hook 测试（D-16）
- [ ] 无框架安装缺口（Go testing/vitest 均已就绪）

## Security Domain

### Applicable ASVS Categories
| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | 既有 JWT 中间件覆盖全部新端点（无新增豁免路径） |
| V3 Session Management | no | 不涉及 |
| V4 Access Control | yes | 任务端点挂 `/backups` 组级 RequirePermissions（D-31，零新权限点）；restore 高危写操作已有 `network:backup:restore` 权限点 |
| V5 Input Validation | yes | handler binding required 校验（backupId/deviceId）；D-04 跨设备校验；列表排序走 base.ApplySort 白名单（D-33 删除幻列 status 即属此类加固） |
| V6 Cryptography | no | md5 calculateHash 仅作变更检测非安全用途（现状，不改） |
| V7 Errors/Logging | yes | 失败任务错误信息进任务表（不含敏感内容——配置内容本身不入错误信息）；operlog 全程审计（D-18） |

### Known Threat Patterns for 本 stack
| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| 跨设备误推配置（backup 属 A 设备推给 B 设备） | Tampering/Elevation | D-04 service 层 `backup.DeviceID == req.DeviceID` 硬校验 |
| 高危写操作无审计 | Repudiation | handler 发起时 operlog.Record（OperTypeUpdate，D-13/D-18）+ D-09 版本链恢复记录 + D-10 恢复前自动备份（可回退） |
| 同设备并发恢复竞态（配置交错下发） | Tampering | D-08 同设备互斥（任务表 pending/running 查询型） |
| 恶意/损坏备份文件解压（zip bomb 类） | DoS | gzip 解压无解压炸弹放大（gzip 比率有上限但千级放大可构造）——备份文件来源为系统自身落盘 + 权限组保护，风险可接受；如需加固可限制解压上限（io.LimitReader，planner discretion，非必需） |
| 崩溃残留 running 锁死设备恢复入口 | DoS | Pitfall P2 启动收敛/陈旧容忍 |

## Sources

### Primary (HIGH confidence)
- `internal/services/config_backup_service.go` — 三 TODO 精确行位（:156-166/:200-210/:535-546）+ sort 白名单（:464-469）+ 双备份路径
- `internal/device/executor.go` — ExecuteCustom(:161-193)/GetConfig vendor switch(:267-300)/executeWithRetry(:201-264)
- `internal/device/scrapli_wrapper.go` — SendConfigs(:614-637 per-line SendConfig 循环)/GetConfig(:639-662)/PlatformName(:66-97)
- scrapligo v1.4.0 模块源码（go env GOMODCACHE 实测）— `driver/network/sendconfigs.go`（AcquirePriv(configuration) + 泛型裸发）/`sendconfig.go`/`sendcommand.go`（network 层自动回默认 priv）
- `internal/services/portwrite/port_write_service.go:393-510` + `parse_error.go` + `batch_orchestrator.go:64` — SendConfigs 调用形态/V7 exit 教训/detached context 先例
- `internal/services/device_info_collection_service.go:74-240` — 异步任务全模式（Enqueue 去重/worker/recoverPendingTasks）
- `internal/core/db/database.go:487-511, 790-893` — MigrateModelList + 迁移双分支注册布局；migrations 目录实测最新 210
- `internal/services/config_backup_service_79_06_test.go` — 五 helper 逐一验证（newCbk7906/cbk7906Chdir/newDriver7906FromFixture/writeFixture7906/newExecutor7906）
- `internal/services/portwrite/testdata/huawei_shutdown_success.fixture` — SendConfigs fixture 字节格式（cat -A 实测）+ `port_write_e2e_test.go` — e2e harness 模式
- `compress/gzip` — Go 1.24.5 本地 `go doc`（NewWriter/NewReader/ErrHeader/ErrChecksum/Close-flush 语义）
- `internal/api/v1/network/backup_handler.go:257-288` + `backup_handler_test.go:220-246` — Restore handler 现状 + stub 锁测试
- 前端 `src/pages/network/backups/`（index.tsx:224 + useBackupModals.ts:133 直接 `post` 调用点）+ `useDiscoveryPolling.ts` — 轮询模板

### Secondary (MEDIUM confidence)
- 无（未使用 WebSearch；全部结论源自代码库/模块源码/本地 toolchain）

### Tertiary (LOW confidence)
- 无

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — 零新依赖；gzip/scrapligo 均经本地源码/doc 验证
- Architecture: HIGH — RestoreConfig 集成点（ExecuteCustom）、异步先例（EnrichmentTask）、互斥模式（Enqueue 去重）、fixture 格式全部代码级验证
- Pitfalls: HIGH — 12 条中 11 条有 file:line 或模块源码直接证据；P2（崩溃残留锁）为逻辑推演（先例 recoverPendingTasks 佐证）

**Research date:** 2026-09-05
**Valid until:** 2026-10-05（stable——go.mod 依赖锁定，无快速漂移面）
