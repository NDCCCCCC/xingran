# Phase 93: config_backup 三处 TODO 闭环 (🟡 中优 P2) - Context

**Gathered:** 2026-09-05
**Status:** Ready for planning

<domain>
## Phase Boundary

实现 `internal/services/config_backup_service.go` 三个 TODO 空函数并闭环端到端：

1. **:158 压缩** — `CreateBackup` 大配置文件落盘时 gzip 压缩（`BatchBackupDevices` 固定 `CompressLarge: true`，生产可达）
2. **:206 解压** — `GetBackupContent` 读取压缩备份时 gzip 解压还原
3. **:543 恢复** — `RestoreBackup` 从"功能待实现"stub 变为**异步任务化**的设备配置恢复（读备份 → 清洗 → 下发 → 回读校验）

配套：gzip 压缩 helper 统一手动/批量/自动三条备份路径、`DeviceExecutor.RestoreConfig` 新方法、恢复任务表 + migration、`config_backup_service_93_NN_test.go` 回归测试（含 FileTransport e2e）、BACKUP-CLOSED-05 端到端自动化验收、REQUIREMENTS/ROADMAP 措辞校准、CLAUDE.md Convention 段、2 处顺手 bug 修复（各独立 commit）。

**关键现实校准（2026-09-05 侦察确认，以本 CONTEXT 为准）：**

1. **ROADMAP 93-02 措辞失准** — "事务化（读备份 → 校验 schema → 批量 upsert → 失效缓存）"是数据库语汇；代码现实是**网络设备配置下发**（`backup_handler.go:285` 注释明示"把备份配置下发到设备 — 属高危写操作"）。恢复语义按 D-01 校准，REQUIREMENTS + ROADMAP 措辞同步修订（Phase 92 D-06 先例）。
2. **Restore 端点已就绪** — POST `/network/backups/:id/restore`（`backup_handler.go:266`）已存在，operlog 已记录、deviceId 绑定已就绪；本期改造其 service 层为异步语义。
3. **压缩生产可达** — `BatchBackupDevices:241` 固定 `CompressLarge: true`，压缩实现后立即在批量备份路径生效。
4. **auto 备份路径现状不一致** — `createNewAutoBackup:389` 写文件无压缩分支（D-25 统一）。
5. **`configBackupAllowedSortFields` 含不存在的 `status` 列** — 用户传 `orderByColumn=status` 排序会 SQL 报错 500（现存 bug，D-33 顺手修）。

**不在本 phase：**
- 前端独立恢复任务管理页（仅备份页恢复按钮异步交互 + Modal 内进度/结果，D-19）
- 任务级重试 / 任务取消 / 任务清理 cron（D-20/22/34）
- 真机恢复 site-visit UAT（自动化测试落地验收，D-30；真机 UAT 可后续登记）
- operlog OperType 新增常量（D-13 复用 Update，25 常量锁值防线零改动）
- 18 项真实 TODO 中非 config_backup 的 17 项（v1.29 Out of Scope）

</domain>

<decisions>
## Implementation Decisions

### Restore 语义与技术路径 (Area 1)

- **D-01: 恢复语义按代码现实校准** — 恢复 = 把备份的设备配置下发到网络设备。93-02 措辞按 Phase 92 D-06 先例同步修订 REQUIREMENTS + ROADMAP（同 commit）。
- **D-02: `DeviceExecutor.RestoreConfig` 唯一下发入口** — device 包新增厂商感知恢复方法，与 `GetConfig` 对称（GetConfig 按 vendor 选读命令的 switch 先例在 `executor.go:267-300`）；内部走 scrapligo SendConfigs 配置模式批量下发，复用连接池/调度/重试基建。不复用 portwrite wrapper（端口写语义绑定）、不用 ExecuteMultipleOnDevice（exec 模式无配置模式语义）。
- **D-03: 基础清洗后下发** — 下发前过滤空行/注释行（`!`/`#`）等非命令行；清洗规则放 device 包 RestoreConfig 内部（与厂商 switch 同处）。
- **D-04: 仅限同设备恢复** — service 层校验 `backup.DeviceID == req.DeviceID`，不一致报错拒绝（防跨设备误推配置）。
- **D-05: 独立超时常量** — `pkg/constants/timeouts.go` 新增 RestoreConfig 专用超时常量（值 planner 结合 scrapligo 行为定），不用 CommandExecTimeout（300s 单命令语义不够）。
- **D-06: 硬编码 vendor map** — vendor→配置模式命令（华为/H3C system-view vs 思科系 conf t/end 等）硬编码 Go map，v1.19 W1 `vendor_port_template.go` 同款先例。
- **D-07: 结构化 RestoreResult** — 携带下发行数/耗时/恢复后 hash/hashMatched 等，handler 响应同步扩展（前端展示"发了多少行、是否全部成功"）。
- **D-08: 同设备互斥** — 同一设备同时只允许一个 restore 进行中，第二个请求报"恢复进行中"；实现形态（锁/标记）planner 定。
- **D-09: 版本链恢复记录** — 恢复成功后在 `sys_config_backup` 插入一条记录（ChangeReason 记"恢复自版本 N"；BackupType 枚举形态 planner 定），版本链可追溯恢复事件。

### 恢复安全护栏 (Area 2)

- **D-10: 恢复前自动备份** — RestoreBackup 下发前先对目标设备自动创建备份（复用现有备份链路，ChangeReason 记"恢复前自动备份"），把高危操作变成可回退。
- **D-14: 备份失败即中止** — 恢复前备份失败（设备连不上/GetConfig 出错）则中止恢复并报错——无法备份 = 无回退退路，安全优先。
- **D-11: 回读 hash 警告不失败** — 恢复后重新 GetConfig 与备份内容比对：不一致时 `RestoreResult.hashMatched=false` + 警告日志，restore 整体算成功（真实设备回读与备份文本完全一致几乎不可能；下发已不可逆，回读差异 ≠ 恢复失败）。
- **D-12: fail-fast 即停** — 下发某行报错即停止后续行，RestoreResult 记录进度（第 N/M 行 + 失败行内容）；v1.19 批量端口写串行 fail-fast 同款先例；设备半恢复态可从"恢复前备份"回退。
- **D-13: 复用 OperTypeUpdate** — 25 OperType 常量 AST 锁值防线零改动；恢复经 module="配置备份" + oper_param 区分。

### 异步任务化（用户明确选择，超出推荐的同步方案）

- **D-15: 新建恢复任务表** — 独立任务表存任务生命周期（status/进度/结果 JSON/错误信息），与备份版本链职责分离；D-09 版本链恢复记录在任务成功后写入。表名/migration 编号 planner 定。
- **D-16: 前端轮询** — 任务进度经前端轮询任务状态到达（现有页面轮询模式），不引入 WS 推送。
- **D-17: 触发返回 taskId** — POST /restore 改为：参数校验 + 同设备/备份校验 + 创建任务记录（pending）+ 立即返回 taskId；新增任务查询端点（列表/详情，路由 planner 定）。
- **D-18: handler 发起时记 operlog** — handler 成功返回 taskId 时记 operlog（符合 CLAUDE.md "success path 末尾记录"约定，零特殊处理）；任务结果只在任务表，不双记。
- **D-19: 后端 + 最小前端** — 备份页恢复按钮改异步交互（发起 → taskId → 轮询 → Modal 内进度/结果展示），不做独立任务管理页。
- **D-20: 不做任务清理** — 任务记录与备份记录同等保留（低频高危操作量级小）。
- **D-21: 不设全局并发上限** — 同设备互斥（D-08）+ DeviceExecutor 调度器队列自然限流。
- **D-22: 重新发起代替重试** — 不做任务级重试；失败后重新发起 restore（每次重新走完整流程含恢复前备份，幂等安全）。
- **D-34: 四态状态机，不支持取消** — pending → running → success/failed；scrapli 下发中的操作无法安全中断（设备半恢复态比继续更糟）。

### 压缩/解压细节 (Area 3)

- **D-23: `.gz` 后缀 + 双检查** — 压缩文件命名 `.conf.gz`（FilePath 存实际路径）；GetBackupContent 解压触发同时检查 Compressed 标志与 `.gz` 后缀（ROADMAP "识别 .gz 后缀"落此）。
- **D-24: BackupSize 记原始大小** — `len(config)` 口径与 DB 存储路径/统计 totalSize 一致；磁盘占用由文件系统自反映，不加字段。
- **D-25: auto 路径统一压缩** — `createNewAutoBackup` 大文件也压缩（复用同一 gzip helper），消除 Compressed true/false 状态混杂。
- **D-26: 默认压缩级别** — gzip DefaultCompression，不做可配置化。

### 失败场景与测试 (Area 4)

- **D-27: `config_backup_service_93_NN_test.go` 新文件** — REQUIREMENTS 原文 `_78_NN` 是笔误（D-33 修订）；不追加 79_06 文件（职责分离）。
- **D-28: 锁 4 场景清单，手法不锁** — 必须覆盖：① 写文件失败（t.Chdir 只读目录/非法路径模拟磁盘满）② 损坏 gzip 字节流（截断/随机字节触发校验失败）③ restore 下发中断 → 任务 failed + 已发进度留痕 ④ DB 写入失败 → 记录回滚。模拟手法由 researcher 对照 79_06 现有手法定。
- **D-29: FileTransport e2e** — restore 全链路（发起→任务执行→清洗→SendConfigs→回读 hash→状态流转）用 `SeedConnectionForTesting` 零 SSH 测试；79_06 同款基建 + v1.19 TestE2E_* 先例，可进 CI。
- **D-30: 端到端验收 = 自动化集成测试** — BACKUP-CLOSED-05 "备份→恢复→配置一致"落地为自动化断言链（CreateBackup → GetBackupContent → restore 任务 → hash 一致）；Phase 92 D-09 先例，不做手动冒烟。

### 延伸决策 (Areas 5)

- **D-31: 任务端点复用组权限** — 任务查询端点放进 `/backups` 路由组（`network_router.go:153`），继承组级 4 权限点（list/add/restore/diff 任一）；零新权限点、零菜单/角色迁移。
- **D-32: CLAUDE.md 加 Convention 段** — 收口时新增"配置备份恢复约定"段，锁三条：压缩/解压统一走 gzip helper、恢复唯一入口 `DeviceExecutor.RestoreConfig`、恢复必须走异步任务模式（禁止同步下发）。Phase 90/92 收口惯例。
- **D-33: 两处顺手修，各自独立原子 commit** — ① REQUIREMENTS 测试文件名笔误 `_78_NN`→`93_NN`（与 D-01 措辞修订同 commit）② `configBackupAllowedSortFields` 删除 `status` 行（列不存在，排序必 SQL 500；附带 bug 修复，Phase 91 附带修复 2 现存 bug 先例）。
- **任务表设计惯例（不另行提问，直接锁定）** — 状态枚举用 string（"pending"/"running"/"success"/"failed"，跟随 ConfigBackup 的 BackupType/StorageType string 枚举风格）；UUID 主键走 BeforeCreate hook（BaseModel DB default 缺失踩 23502 教训）；AutoMigrate 注册 + `internal/core/db/migrations/` MigrateNNN 注册双线齐全（sqlite 缺表 family pattern 已重复 5 次的教训）。

### Claude's Discretion

以下细节由 planner/researcher 决定，无需再问用户：
- RestoreConfig 超时常量的具体值（D-05 只锁独立常量 + 放 timeouts.go）
- vendor map 的具体命令内容（进入/退出配置模式，对照 scrapligo priv 机制）
- 清洗规则的具体过滤集（空行/`!`/`#` 之外是否扩展）
- 任务表名、字段命名、migration 编号、Task service/handler 文件布局
- 任务查询端点路由形态（/backups/tasks vs /restore-tasks 等）
- RestoreResult/任务表 result JSON 的具体字段集
- 恢复记录的 BackupType 表达（复用 auto/manual vs 新增常量——若新增需同步锁值测试）
- 失败场景的具体模拟手法（对照 79_06 先例）
- 93_NN 测试文件内用例分组
- 同设备互斥的实现形态（DB 锁/in-progress 标记/内存锁）
- handler 响应 shape 细节（taskId 字段名等）

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### v1.29 milestone 全局
- `.planning/REQUIREMENTS.md` § BACKUP-CLOSED（:78-86）— BACKUP-CLOSED-01..05 需求原文（**`_78_NN` 笔误与 93-02 恢复措辞按 D-33/D-01 修订**）
- `.planning/workstreams/milestone/ROADMAP.md` § Phase 93 — Goal / 3 plans 拆分 / 4 Success Criteria（**93-02 措辞按 D-01 校准为设备配置下发；3 plans 拆分因异步化扩展需 planner 重新权衡**）
- `.planning/PROJECT.md` — v1.29 Current Milestone 段 D-01..D-06 locked decisions（D-03 零回归底线 / D-04 atomic commit / D-05 三处 TODO 是唯一允许的行为变更例外）
- `.planning/workstreams/milestone/phases/92-p2/92-CONTEXT.md` — Phase 92 决策先例：措辞修订纪律（D-06）、端到端=自动化测试（D-09）、CLAUDE.md Convention 收口（D-10）

### 本期改造核心代码
- `internal/services/config_backup_service.go` — 三处 TODO 所在（:158 压缩 / :206 解压 / :543 恢复）；CreateBackup/createNewAutoBackup 双路径（D-25 统一对象）；`configBackupAllowedSortFields`（D-33 修 status 行）
- `internal/api/v1/network/backup_handler.go` — Restore handler（:257-288，已就绪需改异步语义）+ operlog 记录点
- `internal/api/v1/network/network_router.go:150-170` — `/backups` 路由组 + 组级 RequirePermissions（D-31 挂靠点）
- `internal/models/config_backup.go` — ConfigBackup 模型（BackupType/StorageType string 枚举风格、BeforeCreate UUID hook、IsStoredInDatabase/IsStoredInFile helper）
- `internal/device/executor.go` — DeviceExecutor（GetConfig :267 厂商 switch 先例 / ExecuteOnDevice / ExecuteMultipleOnDevice / executeWithRetry / scheduler.Submit 任务模式）— D-02 RestoreConfig 新增处

### 测试基建先例
- `internal/services/config_backup_service_79_06_test.go` — 16 测试；FileTransport + `SeedConnectionForTesting` 零 SSH 模式、t.Chdir(t.TempDir()) 工作目录隔离纪律（getBackupDir 硬编码相对路径无注入点）
- v1.19 `internal/services/portwrite/port_write_e2e_test.go` — TestE2E_* FileTransport SendConfigs 先例（D-29 参照）

### 项目级约束与防线
- `CLAUDE.md` § operlog convention — handler success path 末尾 Record 约定（D-18 依据）
- `CLAUDE.md` § Compilation & Build Verification — `go build ./...` 0 错误
- `CLAUDE.md` § Testing — `go test ./internal/services/...` 0 失败（BACKUP-CLOSED-05）
- `internal/utils/operlog/regression_test.go` — 25 OperType 锁值防线（D-13 零改动承诺）
- `pkg/constants/timeouts.go` — Phase 90 超时常量集中化（D-05 新增处）
- `internal/core/db/migrations/` — MigrateNNN 注册惯例（任务表 migration；sqlite 分支 AutoMigrate 双注册必须齐全）
- `pkg/query` / `base.ApplySort` — 分页与排序白名单（GetBackupList 现状，D-33 涉及）

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `DeviceExecutor.GetConfig` 的厂商 switch（huawei/h3c → display current-configuration 等）— D-02 RestoreConfig 厂商感知的直接参照
- `executor.Submit` 任务模式 + `executeWithRetry` — RestoreConfig 下发的既有调度/重试基建
- `calculateHash`（md5）— 回读 hash 比对复用（D-11）
- `createNewAutoBackup` — D-10 恢复前自动备份的复用链路
- 79_06 测试的 FileTransport 装配（NewDeviceConnectionPool → NewDeviceTaskScheduler → NewDeviceExecutor + SeedConnectionForTesting）— D-29 e2e 直接复用
- `base.ApplySort` + 排序白名单模式 — 任务列表端点分页排序现状模式

### Established Patterns
- Handler-Service 分层 + 构造器注入 — 新 Task service/handler 遵循
- 组级 RequirePermissions OR 模式 — D-31 挂靠
- operlog.Record handler 末尾约定 — D-18
- string 枚举 + 常量定义于 models — 任务状态机风格
- migration MigrateNNN + AutoMigrate 双注册 — 新表上线惯例
- atomic commit 纪律（v1.29 D-04）— 每个独立修复/功能一个 commit

### Integration Points
- `core.Core` 装配链 — ConfigBackupService 构造处（NewConfigBackupService(db, executor)），Task service 需要新的装配点
- `POST /network/backups/:id/restore` — 前端 `networkApi`（或 backup 相关 Api 文件）现有调用点，异步化后响应 shape 变化需前端同步
- `operlog` — Restore handler 现有记录点保持
- `pkg/constants/timeouts.go` — D-05 新常量落点

</code_context>

<specifics>
## Specific Ideas

### 异步化后的 restore 流程全景（D-01..D-22 综合预览）
```
POST /restore (backupId, deviceId)
  → handler 校验 + operlog.Record（发起）           [D-18]
  → service: backup.DeviceID == deviceId 校验       [D-04]
  → 创建任务记录 pending → 返回 taskId              [D-15/17]
  → 异步执行:
      恢复前自动备份（失败 → 任务 failed 中止）      [D-10/14]
      读备份内容 → 解压（若压缩）                    [D-23]
      基础清洗 → RestoreConfig 下发（vendor map）    [D-02/03/06]
        fail-fast：失败行即停 → 任务 failed + 进度   [D-12]
      回读 GetConfig → hash 比对（不一致仅警告）     [D-11]
      插入版本链恢复记录 → 任务 success              [D-09]
  → 前端轮询任务状态 → Modal 展示进度/结果          [D-16/19]
```

### 同设备互斥的语义边界
互斥窗口 = 任务 pending+running 全程；第二个 restore 请求在同一设备上有进行中任务时立即报错（不排队等待）。

### 估算提示（供 planner 参考）
- 后端新增：device 包 RestoreConfig（含 vendor map/清洗）+ 任务 model/migration + Task service/handler + backup service 三函数实现 + 措辞/笔误修复
- 前端最小改动：备份页恢复按钮交互 + Modal 进度/结果 + taskId 轮询
- 测试：93_NN 文件（压缩/解压/清洗/恢复 e2e/失败 4 场景/端到端断言链）

</specifics>

<deferred>
## Deferred Ideas

### 本期明确不做（讨论中确认边界）
- **真机恢复 site-visit UAT** — FileTransport 自动化已覆盖验收；真机高危写操作 UAT 可仿 v1.18/19 模式后续登记（owner = 现场运维同事）
- **任务管理独立页面 / WS 实时推送 / 任务取消 / 任务级重试 / 任务清理 cron** — D-16/19/20/22/34 明确排除，若任务量级增长再评估
- **任务表分库/归档策略** — D-20 保留式管理，量级小不做

### 讨论中识别的相关但未纳入项
- **RestoreConfig 的 dry-run 预览模式** — v1.20 FUTURE-06 同源（下发前只解析校验不执行）；本期不做，异步任务架构已为其留了自然扩展点（pending 态预览结果）
- **恢复操作的 Redis 缓存失效联动** — ConfigBackup 不在业务缓存体系（sys_config 缓存键族与本表无关），无需联动

None — discussion stayed within phase scope（异步化经用户明确选择纳入，非 scope creep）

</deferred>

---

*Phase: 93-config_backup 三处 TODO 闭环*
*Context gathered: 2026-09-05*
