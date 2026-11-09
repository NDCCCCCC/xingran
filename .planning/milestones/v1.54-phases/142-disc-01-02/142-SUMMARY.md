---
phase: 142
plan: 01
status: complete
commit: false   # 未 git commit，由 Lead 统一集成提交
executor: gsd-executor / exec-142
---

# Phase 142 · 发现设备真实导入（DISC-01/02）SUMMARY

## 改动清单（仅本 phase 授权文件面，6 个文件）

| 文件 | 改动 |
|---|---|
| `internal/services/device_discovery_service.go` | 替换 `ImportDiscoveredDevices` stub（`return len(deviceIDs), nil`）为真实导入；新增私有 `loadDiscoveryResultRows` / `buildNetworkDevice` / `errDiscoveryTaskNotFound` 哨兵 / `discoveryImportMaxBatch=1000`。import 追加 `errors`、`strings`、`pkg/constants`（UUID 校验）。 |
| `internal/api/v1/network/discovery_handler.go` | `DiscoveryHandler` 加可选字段 `invalidateDeviceCache func(context.Context) error` + `WithDeviceCacheInvalidator`；`ImportDevices` 成功路径在 `operlog.Record` 之前调缓存失效（nil 即 no-op，失败静默 `_ =`）。 |
| `internal/api/v1/network/network_router.go` | 装配处 `NewDiscoveryHandler(...).WithCore(core).WithDeviceCacheInvalidator(deviceService.InvalidateAllDeviceCache)`（复用已装配的 `networkServices.CacheService` 方法值）。 |
| `internal/api/v1/network/discovery_handler_test.go` | 删除写死 `{"count":0}` 的 stub 断言，改 seed 2 条 `DeviceDiscoveryResult` + 断言 `count==2` / `sys_network_device` 实际 2 行 / `lastBusinessType==6`；保留 `not_found_1500`。 |
| `internal/services/device_discovery_service_79_06_test.go` | 删除 `n==3` / "stub implementation echoes" 断言，拆成 `import_ignores_deviceIDs_and_missing_task`（空结果 `(0,nil)` + 任务不存在报错）与 `results_persisted`（原 GetDiscoveryResults 断言完整保留）。 |
| `internal/services/device_discovery_service_import_test.go`（新） | 同包 `services`，复用 `newDdv7906`（经 `newDB7906`）并追加 `&models.NetworkDevice{}` 迁移；9 个用例。 |

**未触碰**：`pkg/permission/config.go`、`internal/api/router.go`、`internal/core/db/database.go`、`internal/services/portwrite/**`、`internal/services/addomain/**`、`xingran-react-frontend/**`、任何 migration、`GetDiscoveryResults`、`persistDiscoveryResults`。零新 migration、零新依赖、零 PG 专属 SQL、零裸 `c.JSON`、零 git 写操作（`git add`/`commit`/`stash`/`checkout` 全部未执行）。

## DISC-01 / DISC-02 如何满足

### DISC-01（真实导入）
- **全量导入**：`deviceIDs` 参数忽略（`_ = deviceIDs`），恒读 `sys_device_discovery_result` 该任务全部结果行；保留 `(int, error)` 签名兼容。前端 POST `{}` 不受影响。
- **去重（按 `ip_address` trim + `net.ParseIP` 校验）**：
  - 批内同 IP 去重，保留 `seq` 最大（最后）一条；
  - 库内未软删命中 → **跳过、不覆盖**（保护人工维护的 `DeviceName`/`Port`/`Status` 等）；
  - `Unscoped` 软删命中 → **恢复**（清 `DeletedAt` + 覆盖发现字段 + `tx.Unscoped().Save`），不新建重复行。
  - 预查用一次 `ip_address IN ?` 加载已存在（含软删）行，避免 N+1。
- **单事务全原子**：`db.WithContext(ctx).Transaction(...)`，事务内全部用 `tx`（sqlite 单连接死锁红线）；任一行失败整体回滚 `return 0, fmt.Errorf("导入发现设备失败: %w", err)`；无结果行 `(0, nil)`；任务不存在保持 `fmt.Errorf("发现任务不存在: %w", err)`（用 `errors.Is` 哨兵从事务里还原该独立文案）。
- **字段映射（按 Lead 裁决表）**：`IPAddress` trim 后直接（空/非法跳过）；`SysName` 空 → `Device-<末段IP>`；`DeviceType` 空/越界 → `models.DeviceTypeSwitch`（枚举白名单）；`Vendor` 空 → `""`；`Model` 直接；`SysDescr` → `Description`；`IsAlive=false` 跳过；`Status` 按 `models.DeviceStatusOnline(0)` 赋值（不套用通用 0正常/1停用）；`DeptID` 取 `task.GroupID` 且用 `constants.UUIDPattern` 校验合法 UUID（非法不写脏数据）；`CreatedBy`/`UpdatedBy` = 入参 `createdBy`；`Port=22`；`SNMPPort` 取 `task.SNMPPort`（0 → 默认 161）；`SerialNumber`/`SoftwareVersion`/`Uptime`/`CredentialID` 留空；`ID` 由 `BaseModel.BeforeCreate` 自动补，不手写。
- **批量写入**：`tx.Create(&rows)` 按 `1000` 分批（避免 PG 65535 参数上限），不用 `clause.OnConflict`。
- **读路径**：导入后不删结果表；`GetDiscoveryResults` 未改；幂等靠 IP 去重。

### DISC-02（端点约定）
- success path 顺序：`ImportDiscoveredDevices` → **缓存失效**（`InvalidateAllDeviceCache(ctx)`，事务外、commit 之后、失败静默）→ `operlog.Record(..., "网络设备发现", operlog.OperTypeImport)` → `response.Success(c, gin.H{"count": count})`。错误路径 `response.Error(c, apperrors.InternalServerError(err))`。
- 缓存注入采用 PLAN 允许的**最小方案**：handler 加可选函数字段 + router 装配时注入，测试 `newDiscoveryHandler`/`newNetworkTestEnv` 不注入即 no-op，**现有构造点零改动**。不给 service 加 CacheProvider 依赖。
- operlog 行、权限码 `network:discovery:add`、路由、响应结构均未改。
- `user_id.(string)` 既有写法保持原样（PLAN 明确不在本 phase 修）。

## 实现说明 / 与裁决表的一处必要偏离（已在代码注释登记）

`sys_network_device.status` 列带 `gorm:"default:2"`（未知）。GORM 对带 `DefaultValueInterface` 的字段会把**零值无条件替换成默认值**：即使结构体赋 `DeviceStatusOnline(0)`，`tx.Create(&rows)` / `tx.Save` 落库仍是 `2`，且 `Select("*")` / 指定列 `Select` 均无法绕过（实测 4 种写法只有 map create 能写 0）。

为同时满足「裁决表 `tx.Create(&rows)` 批量语义 + `BaseModel.BeforeCreate` 自动 UUID」与「Status=0 在线」，实现保留 `tx.Create(&rows)` / `tx.Save(&restored)`，并在写入后对**仅新建+恢复行**的 IP 做一次 `tx.Model(&NetworkDevice{}).Where("ip_address IN ?", writtenIPs).Update("status", 0)` 显式回写。单列 `Update` 不受 create 的 default 逻辑影响，SQLite/PG 双方言都可靠；"跳过不覆盖"的人工维护行不在 `writtenIPs` 中，语义不受影响。

测试夹具同理：`DeviceDiscoveryResult.IsAlive` 带 `default:true`，结构体零值 `false` 会被 GORM 改写为 `true`，故新单测的 seed 用 `map[string]interface{}` 显式写列以造出「死设备」用例。

## 验证（Task5，逐项结果）

| 命令 | 结果 |
|---|---|
| `go build ./...` | **exit 0 / 全绿**（首次运行时曾因并行 executor 的 `migration_238`/`portwrite` 中间态短暂报错，对方补齐后复跑通过） |
| `go test -count=1 ./internal/services/ ./internal/api/v1/network/` | **exit 0**：`ok github.com/xingran-next/xingran-go-backend/internal/services 134.095s`、`ok github.com/xingran-next/xingran-go-backend/internal/api/v1/network 1.189s` |
| `go test -count=1 -run TestImportDiscoveredDevices ./internal/services/` | `ok`（9 个新用例单独复跑通过） |

> 按 PLAN **未跑** `go test ./...` 全量（留待 Lead 集成后统一跑）；未跑前端命令；未启动任何长驻进程（一次性命令均前台执行）。

### 新增单测覆盖（9 例，均通过）
`TestImportDiscoveredDevices_AllRows`（3 条不同 IP → count=3、字段逐项含 DeptID/CreatedBy/DeviceName/Status/SNMPPort、非法 GroupID 不落库、SNMPPort=0 回落 161）、`_DedupByIP`（批内去重保留最后 / 库内已存在跳过不覆盖 / 软删恢复且字段刷新，3 子用例）、`_SkipsDeadAndBadIP`（死/空/非法 IP 跳过 + SysName/Vendor/DeviceType 兜底 + IP trim）、`_EmptyResults`（0,nil）、`_MissingTask`、`_AtomicRollback`（gorm create callback 注入约束冲突 → 事务回滚、零新增、err 含「导入发现设备失败」）、`_Idempotent`（二次 count=0、总行数不变）、`_DoesNotDeleteResults`（导入后 GetDiscoveryResults 仍返回原明细）。

### must_haves 自检
1. stub 已删，结果行真实写入 `sys_network_device` ✅
2. 去重 / 事务回滚 / 幂等 / 字段兜底 5 类分支有单测且通过 ✅
3. 两处旧 stub 断言已改，网络模块既有测试不回归（`internal/api/v1/network` 整包 ok）✅
4. success path `operlog.Record(OperTypeImport)` + 缓存失效在 `response.Success` 之前，无裸 `c.JSON` ✅
5. `go build ./...` + 两包 `go test` 全绿 ✅
6. 零新 migration、零新依赖、未触碰其它 phase 文件面 ✅

## Known Gaps / 未决

1. **缓存注入方式**：采用 PLAN「最小方案」——handler 可选函数字段 `invalidateDeviceCache func(context.Context) error`，router 注入 `networkServices.CacheService.InvalidateAllDeviceCache` 方法值。测试不注入即 no-op，故现有 `newNetworkTestEnv` / `newDiscoveryHandler` 构造点零改动。若 Lead 后续希望改为直接注入 `networkServices.CacheService` 接口（裁决表 §一 的原始写法），需同步调整测试构造点；当前方案功能等价且改动面最小。
2. **枚举兜底实测结论**：`DeviceType` 空/越界（如 `"unknown-type"`）均兜底为 `models.DeviceTypeSwitch`，`Vendor` 空落库为 `""`（列允许空串），`Status` 在线=0 需靠单列 `Update` 回写（见上「必要偏离」）——均已由单测实测覆盖，非推断。
3. **Status=0 回写**是一次额外 `UPDATE`（仅新建+恢复行），与裁决表纯 `tx.Create` 有最小偏离；已在代码注释与本节登记，不改变对外语义。
4. **顺带隐患（PLAN 明确不在本 phase 修）**：`user_id.(string)` 硬断言 panic、`persistDiscoveryResults` 未开事务、`ExecuteDiscovery` 未检查 `Save` Error —— 维持原样，登记为后续候选。
5. **PG 双方言**：本 phase 单测只在文件型 sqlite 上跑（PLAN 规定）；PG 路径靠「纯 GORM 链式 API、无 `::uuid`/`ILIKE`/裸 `ON CONFLICT`、`IN ?` 参数化」保证，未连真实 PG 实测（与全仓既有验证基线一致，Lead 集成后统一跑全量）。
6. **工作区为 4 executor 并行共享**：`git status` 中可见 Phase 143/144 的 router.go / database.go / migration_238 / portwrite / addomain / 前端等改动，均为其它 executor 并行写入，与本 phase 文件面无交集，非本 executor 改动。

## 提交状态

**未执行 `git add` / `git commit` / `git stash` / `git checkout`**（PLAN/用户硬约束：4 个 executor 并行写同一工作区，避免 `index.lock` 冲突）。本 phase 只写文件、跑本 phase 验证命令、写本 SUMMARY。**由 Lead 统一集成提交。**
