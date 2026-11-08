---
phase: 142
plan: 01
wave: 1
depends_on: []
files_modified:
  - internal/services/device_discovery_service.go
  - internal/services/device_discovery_service_import_test.go (new)
  - internal/services/device_discovery_service_79_06_test.go
  - internal/api/v1/network/discovery_handler.go
  - internal/api/v1/network/discovery_handler_test.go
  - internal/api/v1/network/network_router.go
autonomous: true
commit: false   # Lead 统一提交，禁止本 executor git add/commit（并发 index.lock 冲突）
---

# Phase 142 · 发现设备真实导入（DISC-01/02）

## 必读（按顺序）
1. `.planning/notes/261108-v154-architecture-decisions.md` **§一**（Lead 已拍板的全部设计裁决，权威）
2. `.planning/ROADMAP.md` Phase 142 段（成功标准 4 条）
3. `.planning/REQUIREMENTS.md` DISC-01/02
4. `.planning/PROJECT.md` / `AGENTS.md` / `CLAUDE.md` 中的项目硬约定

## 目标
把 `internal/services/device_discovery_service.go:509-523` 的 `ImportDiscoveredDevices` stub（`return len(deviceIDs), nil`）替换为真实导入：读 `sys_device_discovery_result` → 按 `ip_address` 去重（含软删恢复）→ 单事务批量写 `sys_network_device` → handler 补缓存失效。

## 已确认事实（不要重新调研）
- FIX-07 已把发现明细落进真实表 `sys_device_discovery_result`（`internal/models/device_discovery_result.go:13-27`，`database.go:703` 已 AutoMigrate）。**无需新 migration、无需新建表**。
- `NetworkDevice`（`internal/models/network_device.go:34-57`）唯一约束只有 `ip_address` 的 `uniqueIndex`（:41），**不含 deleted_at** → 软删行仍占位。
- `DiscoveredDevice` DTO（:74-84）7 字段无 ID；`GetDiscoveryResults`（:712-740）把 ID/Seq 丢掉。
- handler `discovery_handler.go:310-327` 恒传 `deviceIDs=nil`、前端 POST `{}`；operlog（:325 `OperTypeImport`）、权限码（`network:discovery:add`）、路由、响应封装**均已就位，不新增**。
- `DeviceDiscoveryService` 只注入 `*gorm.DB`（服务定义 :21-28，`core.go:454` 构造）。
- `CacheService.InvalidateAllDeviceCache(ctx)` 在 `internal/services/network/cache_impl.go:317-320`（`InvalidatePattern` 清 `network_device:*`）；`DiscoveryHandler` 可注入 `networkServices.CacheService`（参照同目录 `DeviceHandler` 用接口的方式）。
- 测试基建：`internal/api/v1/network/handlers_test_helpers_test.go` 的 `newNetworkTestEnv` 用 `glebarez/sqlite :memory:` + **SetMaxOpenConns(1)**（红线：事务回调内禁止再发 gorm 查询、Rows 未读完禁止嵌套查询，否则死锁）；service 单测用 `newDdv7906`（`internal/services/device_credential_helper_79_06_test.go:45-62`，文件型 sqlite + `test7906:fill_uuid` 回调）。
- 现有测试里两处**写死 stub 断言**必须同步改：`discovery_handler_test.go:317-340`（`{"count":0}`）与 `device_discovery_service_79_06_test.go:419-450`（`n==3` / "stub implementation echoes"）。

## Lead 裁决（必须遵守，不得自行改设计）
| 项 | 裁决 |
|---|---|
| `deviceIDs` 语义 | **忽略该参数、恒全量导入**该任务全部结果行；保留签名兼容；不新增 DTO 字段、不加 `Imported` 标记、不加 migration |
| 去重 | 按 `ip_address` trim + `net.ParseIP` 校验：① 批内去重保留最后一条 ② 库内未软删命中 → **跳过不覆盖**（保护人工维护字段）③ `Unscoped` 软删命中 → **恢复**（清 `DeletedAt` + 覆盖字段 + `tx.Save`，复用 `network_device_service.go:296-319`） |
| 事务 | **单 `db.WithContext(ctx).Transaction(...)` 全原子**；事务内全部用 `tx`，禁混用 `s.db`；任一行失败整体回滚 `return 0, fmt.Errorf("导入发现设备失败: %w", err)`；无结果行 `(0, nil)`；任务不存在保持现有 `fmt.Errorf("发现任务不存在: %w", err)` |
| 字段映射 | `IPAddress` 直接（空/非法跳过）；`SysName` 空 → `Device-<末段IP>`；`DeviceType` 空/越界 → `models.DeviceTypeSwitch`；`Vendor` 空 → `""`（列允许空串）；`Model`/`SysDescr`→`Description` 直接；`IsAlive=false` 跳过；`Status` 按 `models.DeviceStatus`（0在线/1离线/2未知）赋值，**不套用通用 0正常/1停用**；`DeptID` 取 `task.GroupID`（校验合法 UUID，参考 `validateOrg` 正则）；`CreatedBy`/`UpdatedBy` = 入参 `createdBy`；`Port=22` / `SNMPPort` 取 `task.SNMPPort`（默认 161）；`SerialNumber`/`SoftwareVersion`/`Uptime`/`CredentialID` 留空；`ID` 由 `BaseModel.BeforeCreate` 自动补，不手写 |
| 批量写入 | `tx.Create(&rows)`，按 `maxBatchSize=1000` 分批（参照 `internal/services/operations/batch_upserter.go:163-183`），避免 PG 65535 参数上限；**不要**用 `clause.OnConflict` 一把梭 |
| 缓存失效 | **在 handler 成功路径、`operlog.Record` 之前**调 `InvalidateAllDeviceCache(ctx)`；失败静默 `_ =`；放在事务外（commit 之后）。不给 service 加 CacheProvider 依赖 |
| 读路径 | 导入后**不删**结果表；`GetDiscoveryResults` 不改；幂等靠 IP 去重 |
| 顺带隐患 | `user_id.(string)` 硬断言 panic、`persistDiscoveryResults` 未开事务、`ExecuteDiscovery` 未检查 Save Error —— **不在本 phase 修** |

## 任务（XML）
<task id="1">
<title>实现 ImportDiscoveredDevices 真实导入</title>
<files>internal/services/device_discovery_service.go</files>
<action>
替换 :509-523 的 stub。建议新增私有方法 `loadDiscoveryResultRows(ctx, tx, discoveryID)`（读 `models.DeviceDiscoveryResult`，`Where("discovery_id = ?", id).Order("seq ASC, id ASC")`，拿到原始行含 ID/Seq）与 `buildNetworkDevice(row, task, createdBy) (*models.NetworkDevice, bool)`（返回 skip 布尔）。主流程：
1. `db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {...})`
2. 事务内用 `tx` 重新读取任务（校验存在）；读结果行
3. 按上述裁决过滤/去重/预查已存在与软删行（一次 `IN` 查询预加载，避免 N+1）
4. 未软删命中 → skip；软删命中 → 恢复分支；其余 → 新建
5. `tx.Create(&rows)` 分批 / `tx.Save(&restored)`；统计 count = 新建+恢复数
6. 返回 `(count, nil)`；错误 `fmt.Errorf("导入发现设备失败: %w", err)`
全部用 GORM 链式 API，**禁止 PG 专属 SQL**（`::uuid` / `ILIKE` / 裸 `ON CONFLICT`）；import 用全模块路径。
</action>
</task>

<task id="2">
<title>handler 注入缓存失效</title>
<files>internal/api/v1/network/discovery_handler.go, internal/api/v1/network/network_router.go</files>
<action>
参照同目录 `DeviceHandler` 持有 `networkServices.CacheService` 接口的方式，给 `DiscoveryHandler` 增加一个可选缓存失效依赖（构造点 `network_router.go:57/66`）。在 `ImportDevices`（:310-327）成功路径、`operlog.Record`（:325）**之前**调用 `InvalidateAllDeviceCache(c.Request.Context())`，失败静默。不改 operlog 行、不改响应结构、不改权限码、不引入裸 `c.JSON`。`user_id.(string)` 的既有写法保持原样（不在本 phase 修）。
若注入方式会牵动过多测试构造点，可采用最小方案：给 handler 加一个函数字段 `invalidateDeviceCache func(context.Context)`，在 router 装配时注入，测试里不注入即为 no-op —— 保证 `newNetworkTestEnv` 现有构造不破。
</action>
</task>

<task id="3">
<title>改两处写死 stub 的既有测试断言</title>
<files>internal/api/v1/network/discovery_handler_test.go, internal/services/device_discovery_service_79_06_test.go</files>
<action>
- `discovery_handler_test.go:317-340`：把 `success_returns_zero_count_stub` 改为 seed 2~3 条 `DeviceDiscoveryResult` 后断言 `data.count == 2`（或实际数）、`sys_network_device` 实际行数一致、`lastBusinessType==6`；保留 `not_found_1500` 用例。
- `device_discovery_service_79_06_test.go:419-450`：删除 `n==3` / "stub implementation echoes" 断言，改为真实语义（deviceIDs 为空时按结果行数导入、不存在 task 报错）；该段可与 `GetDiscoveryResults` 断言拆开。
**不要**删除其它既有测试。
</action>
</task>

<task id="4">
<title>新增 service 单测（就近业务包内）</title>
<files>internal/services/device_discovery_service_import_test.go (new)</files>
<action>
同包 `services`，复用 `newDdv7906`，迁移时**追加 `&models.NetworkDevice{}`**。至少覆盖：
`TestImportDiscoveredDevices_AllRows`（3 条不同 IP → count=3、字段逐项映射含 DeptID/CreatedBy/DeviceName/Status）
`TestImportDiscoveredDevices_DedupByIP`（批内同 IP 只 1 行；库内预置同 IP → 跳过不覆盖；`Unscoped` 软删同 IP → 恢复且字段刷新）
`TestImportDiscoveredDevices_SkipsDeadAndBadIP`（IsAlive=false / 空 IP / 非法 IP 跳过；SysName 空兜底；Vendor/DeviceType 空兜底）
`TestImportDiscoveredDevices_EmptyResults`（0, nil）
`TestImportDiscoveredDevices_MissingTask`
`TestImportDiscoveredDevices_AtomicRollback`（构造约束冲突 → 事务回滚、零新增、err != nil）
`TestImportDiscoveredDevices_Idempotent`（连调两次，第二次 count=0、总行数不变）
`TestImportDiscoveredDevices_DoesNotDeleteResults`（导入后 GetDiscoveryResults 仍返回原明细）
注意单连接死锁红线：service 单测用文件型 sqlite（`newDdv7906`），不要用 `newNetworkTestEnv` 跑事务用例。
</action>
</task>

<task id="5">
<title>自验证（只跑本 phase 相关包，不跑全仓）</title>
<action>
依次执行并如实记录结果：
`go build ./...`
`go test -count=1 ./internal/services/ ./internal/api/v1/network/`
不要跑 `go test ./...`（Lead 集成后统一跑全量）。不要跑前端命令。
若失败，只修本 phase 文件面的问题，迭代至通过。
</action>
</task>

<task id="6">
<title>写 SUMMARY（不提交）</title>
<action>
写 `.planning/phases/142-disc-01-02/142-SUMMARY.md`：改了哪些文件、每个 REQ 如何满足、验证命令与结果、Known Gaps / 未决（如枚举兜底实测结论、缓存注入方式）、以及「未 git commit」说明。**不要执行 git add / git commit / git stash**。
</action>
</task>

## 验证标准（must_haves）
1. `ImportDiscoveredDevices` 不再 `return len(deviceIDs), nil`；`sys_device_discovery_result` 的行真实写入 `sys_network_device`。
2. 去重（批内 / 库内已存在 / 软删恢复）、事务回滚、幂等、字段兜底 5 类分支都有单测且通过。
3. 两处旧 stub 断言已改为真实写入断言；既有网络模块测试不回归。
4. success path 在 `response.Success` 之前有 `operlog.Record(OperTypeImport)` 与缓存失效；无裸 `c.JSON`。
5. `go build ./...` + `go test -count=1 ./internal/services/ ./internal/api/v1/network/` 全绿。
6. 零新 migration、零新依赖；未触碰其它 phase 文件面。

## 禁止
- 禁止 `git add` / `git commit` / `git stash` / `git checkout`（Lead 统一集成提交，避免并发 index.lock）。
- 禁止修改 `pkg/permission/config.go`、`internal/api/router.go`、`internal/core/db/database.go`、`internal/services/portwrite/**`、`internal/services/addomain/**`、`xingran-react-frontend/**`（其它 phase 独占）。
- 禁止新增 migration、禁止新增 DTO 字段、禁止改 `GetDiscoveryResults`。
- 禁止启动长驻进程（AGENTS.md：只走 DSH 托管）。
- 禁止写生产数据 / 生产配置。
