# Phase 93 Research: config_backup 三处 TODO 闭环

**Researched:** 2026-09-05（researcher 摘要 + orchestrator 磁盘核验修正版）
**Confidence:** High（所有关键事实均在代码库中逐一验证，标注 file:line）

## 1. scrapligo SendConfigs 行为（D-02 下发路径的根基）

- **调用形态**：`wrapper.SendConfigs(fullCmds)` 返回 `responses []*SendConfigsResponse`，逐条 `.Result` 判失败 — 先例 `internal/services/portwrite/port_write_service.go:468-481`（含 per-line 失败日志模式 `resp[%d] ... failed=%v result=%q`）。
- **priv 提升**：scrapligo 的 SendConfigs 假设目标 priv=configuration，进入配置模式由 scrapligo 按 platform 自动处理（`port_write_service.go:409-420` 注释记录了 v1.19 的 priv 状态跟踪教训——跨 ExecuteCustom 的 priv 状态需要 wrapper 自己保证）。
- **D-06 vendor map 修正**：由于 scrapligo 按 platform 名自动处理进入配置模式，vendor→命令 map 比预想薄——主要内容是 **vendor→scrapligo platform 名映射**（huawei/h3c → `huawei_vrp` / `h3c_comware`、ruijie/maipu → cisco 风格 platform）+ 退出配置模式命令（华为 `return`、思科系 `end`）。参照 `GetConfig` 的 vendor switch（`internal/device/executor.go:267-300`）。
- **e2e 可测性**：v1.19 `port_write_e2e_test.go` 已验证 FileTransport 预录字节流回放 → 真实 SendConfigs 管道全链路可测（"pre-recorded Huawei VRP byte streams against the real SendConfigs pipeline"，:3-60）；fixture 中 SendConfigs 的命令回显要写入 fixture（:123）。

## 2. RestoreConfig 融入 executor（D-02/D-05）

- **推荐注入点**：`ExecuteCustom(ctx, deviceID, executeFunc func(ctx, *PooledConnection) error, timeout)`（`executor.go:161-190`）——通用闭包执行器，含 scheduler.Submit + done channel + timeout+buffer 兜底。RestoreConfig 内部用 ExecuteCustom 包装 SendConfigs 闭包即可，无需触碰 scheduler 内部。
- **与 GetConfig 对称性**：GetConfig 模式 = `pool.GetDevice(deviceID)` 取 vendor → switch 选命令 → ExecuteCustom 执行。RestoreConfig 同构：GetDevice → vendor 映射 platform → ExecuteCustom(SendConfigs)。
- **超时常量**：`pkg/constants/timeouts.go` 新增 `RestoreConfigTimeout`（建议 10min 量级，planner 定值）；Phase 90 惯例 `int(constants.X.Seconds())` 转换模式见 CLAUDE.md。

## 3. Migration 与 AutoMigrate（任务表 D-15）

- **编号：Migrate211**（researcher 摘要说 209 有误——实测 `migration_209_update_settings_menu_component.go` 与 `migration_210_normalize_menu_parent_id.go` 均已占用）。
- **AutoMigrate 注册点**：`internal/core/db/database.go:511`（`&models.ConfigBackup{}` 所在的模型清单）——sqlite/postgres 双分支同函数注册（XingRan 惯例：model 存在但 AutoMigrate 未注册 = sqlite 缺表，此 family pattern 已重复 5 次，必查）。
- **model 惯例**：`models.ConfigRestoreTask` + `TableName() = "sys_config_restore_task"`；UUID 主键必须 `BeforeCreate` hook 生成（ConfigBackup 同款 `models/config_backup.go:51-64`；BaseModel 无 DB default 的 23502 教训）。
- **状态枚举**：string 常量（`TaskStatusPending/Running/Success/Failed = "pending"/"running"/"success"/"failed"`），跟随 ConfigBackup 的 BackupType/StorageType 风格；不放 sys_dict（任务状态机是代码分支语义，非运营可维护枚举）。

## 4. 异步任务执行先例（D-15..D-18）

- **detached context 必须性**：HTTP 请求 ctx 在响应返回后取消——异步 goroutine 必须用 `context.WithTimeout(context.Background(), constants.RestoreConfigTimeout)`（portwrite `BatchWritePorts` detached 30min context 同款先例，`port_write_service.go`）。
- **同设备互斥**：无现成业务层互斥先例（v1.19 FUTURE-09 明确 deferred）；最简实现 = 任务表内 `device_id + status IN (pending,running)` 唯一性检查（创建任务时事务内查询 + 唯一部分索引兜底）。
- **operlog**：handler 发起时 Record（D-18）零特殊处理；`operlog.RecordBackground`（Phase 48 cron 先例，`internal/utils/operlog`）本期不需要。

## 5. 79_06 测试基建（D-29 复用底座）

researcher 摘要的 "newTestExecutor" helper 名有误，**实际 helper 名**（`config_backup_service_79_06_test.go:43-130`）：

- `newCbk7906(t)` — sqlite DB（AutoMigrate ConfigBackup/NetworkDevice/Config）+ 进程级 temp dir + nil executor
- `cbk7906Chdir(t)` — **进程级 MkdirTemp + t.Chdir**（非 t.TempDir：getBackupDir 相对路径 + applogger 打开 ./logs/app.log 的 Windows 文件占用教训，注释写明）
- `newDriver7906FromFixture(t, fixturePath)` — `platform.NewPlatform("huawei_vrp", ..., FileTransport, WithFileTransportFile(fixture))` → GetNetworkDriver → Open
- `writeFixture7906(t, cycles, cmd, output)` — fixture 字节流构造器（banner → screen-length → n × 命令周期 → 8 spare prompts）

**SendConfigs e2e 扩展**：93_NN 的 restore e2e 需要扩展 fixture 写法支持 send_config 阶段回放——v1.19 `port_write_e2e_test.go` 是现成模板（SendConfigs 命令回显 + 多阶段字节流）。`SeedConnectionForTesting(pool, deviceID, conn)`（`internal/device/e2e_helpers.go:76`）注入连接。

## 6. gzip 标准库（D-23..D-26，Go 1.24）

- 压缩：`gzip.NewWriter(buf)`（默认 DefaultCompression）→ `w.Write([]byte(config))` → `w.Close()`（**Close 前数据不 flush，必须先 Close 再取 bytes**）。
- 解压：`gzip.NewReader(bytes.NewReader(data))` → `io.ReadAll`。
- **损坏形态**（D-28 测试用）：截断流 → `io.ErrUnexpectedEOF`；随机字节 → `gzip.ErrHeader`。直接构造 `[]byte` 即可触发，无需磁盘技巧。

## 7. Handler/前端改动面（D-17/D-19/D-31）

- **Handler**：`backup_handler.go:266-288` Restore——改为：校验 → `RestoreBackup` 返回 `(*RestoreResult或taskID, error)` → operlog.Record → `response.Success(c, gin.H{"taskId": ...})`。**注意 `backup_handler_test.go` 有对 RestoreBackup 旧 error 签名的调用需同步**（researcher 提示 3 处，执行时以 grep 为准）。
- **任务查询端点**：挂 `/backups` 组内（`network_router.go:153-170`），自动继承组级 4 权限点；路由形态（如 `GET /backups/restore-tasks/:id`）planner 定。
- **前端**：`xingran-react-frontend/src/pages/network/backups/`（hooks 模式：useBackupData/useBackupDiff/useBackupModals——新增 useRestoreTask 轮询 hook 顺理成章）；API wrapper 在 networkApi（grep `restoreBackup` 定位）。响应 shape 变化需同步 types。

## 8. 失败场景模拟手法（D-28）

| 场景 | 手法 | 先例 |
|------|------|------|
| 磁盘满/写失败 | cbk7906Chdir 后传入只读/不存在父目录路径，或 file 名含非法字符（Windows） | 79_06 error-path 测试模式 |
| 损坏 gzip | 构造截断/随机 []byte 存文件，GetBackupContent 断言 error | §6 错误形态 |
| 下发中断 | fixture 中 SendConfigs 阶段返回 failed Result（v1.19 e2e device_rejected 用例模式） | port_write_e2e_test.go |
| DB 写入失败 | sqlite 注入非法外键/关闭连接 | 79_06 既有手法 |

## 9. 风险与已知坑

1. **RestoreBackup 签名变化** → handler + handler_test 同步改动面（原子 commit 纪律）
2. **detached context**：异步 goroutine 绑 HTTP ctx = 响应返回即取消，任务必死（§4）
3. **sqlite AutoMigrate 双注册**漏注册 = CI sqlite 分支缺表（§3）
4. **applogger + t.TempDir Windows 冲突**：测试用进程级 temp dir（79_06 注释明确）
5. **SendConfigs priv 状态**：跨 ExecuteCustom 需确认 priv=configuration（port_write_service.go:409 注释）
6. **`configBackupAllowedSortFields` status 列不存在**（D-33 修复对象）：`ORDER BY status` 在 PG 报 column not exist → 500
7. **DeviceExecutor 并发上限**：scheduler 队列自然限流（D-21），RestoreConfig 无需自建并发池

## Validation Architecture

> Nyquist 验证架构：每个需求 ≥2 个独立验证器（自动化优先），确保需求-验证覆盖无死角。

| # | 验证需求 | 验证器 1 | 验证器 2 | 覆盖 REQ |
|---|---------|---------|---------|----------|
| V1 | 压缩实现正确（.gz 产出可解压还原） | 93_NN 单元测试：compress→decompress roundtrip 字节相等 | grep 断言 :158 TODO 注释已删除 + `gzip.NewWriter` 存在 | BACKUP-CLOSED-01 |
| V2 | 解压识别双条件（标志 + 后缀） | 单元测试：Compressed=true 或 .gz 后缀分别触发 | 损坏 gzip → error 断言（V-坏流） | BACKUP-CLOSED-02 |
| V3 | 恢复链路端到端（备份→恢复→一致） | FileTransport e2e：CreateBackup→GetBackupContent→restore 任务→hash 一致断言 | 任务状态流转断言（pending→running→success） | BACKUP-CLOSED-03/05 |
| V4 | 失败场景覆盖（4 场景清单） | 写失败测试（只读目录） | 损坏流测试 + 下发中断测试 + DB 失败测试 | BACKUP-CLOSED-04 |
| V5 | 回归零失败 | `go test ./internal/services/...` exit 0 | `go build ./...` exit 0 | 全部 |
| V6 | 异步任务状态机正确 | 单元测试：四态流转 + 非法转移拒绝 | 同设备互斥测试（第二任务拒绝） | D-08/D-15/D-34 |
| V7 | 恢复前自动备份 + 中止语义 | e2e：备份失败→任务 failed 且无下发 | 产物断言：版本链含"恢复前自动备份"记录 | D-10/D-14 |
| V8 | handler/前端契约同步 | handler 测试：响应含 taskId | npm type-check + lint exit 0 | D-17/D-19 |
| V9 | 文档与锁值防线 | operlog regression_test 不回归（25 常量） | status_constants_test 不回归；CLAUDE.md 含新 Convention 段 grep 断言 | D-13/D-32/D-33 |

**验证策略**：V1-V4/V6-V7 落 93_NN 测试文件；V5 是每 plan 的 verify 步骤；V8 前端部分落既有 hooks 测试模式；V9 是收口 plan 的 acceptance_criteria。

## RESEARCH COMPLETE
