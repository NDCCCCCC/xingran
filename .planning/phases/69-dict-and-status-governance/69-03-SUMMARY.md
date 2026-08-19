---
phase: 69-dict-and-status-governance
plan: 03
subsystem: backend-status-constants
tags: [DICT-01, refactor, operations-module, ratchet-guard, constants]
requires:
  - "69-01：常量真相源 + status_constants_test.go 锁值 + check-status-literals.sh 四模式守护"
provides:
  - "批 2 完成：internal/services/operations/ 10 文件 + api/v1/operations/excel_handler 共 58 处裸字面量常量化"
  - "新常量族 WorkstationDeviceStatus / AssetStatus / AssetNBFStatus（无类型 int，已登记 AST 锁值测试）"
  - "守护白名单收缩 38 → 27 文件（批 2 的 11 条目删除，geocoding F 簇豁免保留）"
affects:
  - "69-04（批 3）与 69-05（批 4）：共享守护白名单 scripts/check-status-literals.sh，工作面为剩余 27 条目"
  - "69-08 DICT-04：CLAUDE.md Status Convention 指向常量真相源时新增三族纳入说明"
tech-stack:
  added: []
  patterns:
    - "Statistics CASE WHEN 聚合统一 fmt.Sprintf + int(models.Xxx)（批 1 范本推广到 8 个 operations service）"
    - "raw SQL 内嵌 status 过滤参数化（ip.status = ? + Raw 追加常量参数，占位符顺序 = 出现顺序）"
    - "map[string]any 字面量值直接放 models 常量（typed 常量装箱为 any，GORM 正常序列化）"
    - "无类型 int 常量族适配 int 型 struct 字段（避免改字段类型，守住零行为变化承诺）"
key-files:
  created: []
  modified:
    - internal/models/asset.go
    - internal/models/workstation_device.go
    - internal/models/status_constants_test.go
    - internal/services/operations/building_service.go
    - internal/services/operations/floor_service.go
    - internal/services/operations/workstation_service.go
    - internal/services/operations/workstation_device_service.go
    - internal/services/operations/server_room_service.go
    - internal/services/operations/room_device_service.go
    - internal/services/operations/infopoint_service.go
    - internal/services/operations/dedicated_line_service.go
    - internal/services/operations/asset_service.go
    - internal/services/operations/excel_service.go
    - internal/api/v1/operations/excel_handler.go
    - scripts/check-status-literals.sh
decisions:
  - "WorkstationDevice.Status（int，0=正常/1=停用）无既有常量族——WorkstationStatus 是三态（空闲/占用/维护）、DeviceStatus 是在线/离线/未知，语义均不匹配；按 69-01 登记机制新增无类型 WorkstationDeviceStatus 族而非语义错配复用"
  - "Asset.Status 与 Asset.NBFStatus 同理新增 AssetStatus / AssetNBFStatus 两族（nbf 为独立布尔维度，非簇 A）"
  - "新增族一律无类型 int 常量：字段是 int 型，typed 常量无法直接赋值，不改字段类型以守住零行为变化"
  - "统计结构体行尾裸值注释改写为常量名引用（沿用批 1 post/role 先例，否则白名单无法移除）"
metrics:
  duration: 约 15 分钟（2026-08-19 06:54–07:09 UTC）
  completed: 2026-08-19
---

# Phase 69 Plan 03: DICT-01 批 2 — operations 模块 + excel 链路字面量替换 Summary

一句话：批 2 清偿 58 处裸 status 字面量（operations 10 service + excel_handler + excel_service map 形态），按 model struct 实际所在包引用常量规避双包陷阱，为无既有常量族的 WorkstationDevice/Asset 新增三族并登记锁值，守护白名单 38 → 27 文件单调收缩。

## Task × Commit 对照表

| Task | 内容 | Commit |
|------|------|--------|
| T1 | 批 2 替换（15 文件：10 service + excel_handler + excel_service + 3 models 文件）+ 白名单收缩 | `ac33b2a` |

**守护脚本基线前后快照：**

| 时点 | 文件数 | 命中数 |
|------|--------|--------|
| 批 1 后（本 plan 起点） | 38 | 134 |
| 批 2 后（本次） | 27 | 76 |

- 批 2 删除条目（11 个，合计 58 处）：excel_handler=1、asset_service=6、building_service=4、dedicated_line_service=6、excel_service=2、floor_service=4、infopoint_service=6、room_device_service=6、server_room_service=4、workstation_device_service=13、workstation_service=6。
- 剩余 76 命中 = 134 − 58，逐文件核对与 `--baseline` 输出一致（见文末快照）。

## 语义簇映射台账（每文件判定）

双包陷阱判定规则：读 service 文件操作的 model struct 定义所在包——子包 `internal/models/operations`（package operations）的 struct 用 `operations.` 前缀；`internal/models`（package models）的 struct 用 `models.` 前缀。全部 11 文件逐一确认 import 与 `Model(&…{})` / `Create(&…{})` 目标类型后选定，未造第三份定义。

| 文件 | 簇 | 实体（model 所在包） | 所用常量 | 处数 |
|------|----|---------------------|---------|------|
| building_service.go | A | OpsBuilding（子包） | operations.BuildingStatusNormal/Stopped | 4（2 行尾注释 + 2 CASE WHEN） |
| floor_service.go | A | OpsFloor（子包） | operations.FloorStatusNormal/Stopped | 4（同上） |
| workstation_service.go | D | Workstation（models） | models.WorkstationStatusAvailable/Occupied/Maintain | 6（3+3，三态不套两态组） |
| server_room_service.go | A | OpsServerRoom（子包） | operations.RoomStatusNormal/Stopped | 4 |
| room_device_service.go | D | OpsRoomDevice（子包） | operations.RoomDeviceStatusNormal/Fault/Scrapped | 6（3+3；子包命名 Scrapped） |
| infopoint_service.go | D | OpsInfoPoint（子包） | operations.InfoPointStatusNormal/Fault/Disabled | 6（3+3） |
| dedicated_line_service.go | D | OpsDedicatedLine（子包） | operations.LineStatusNormal/Fault/Disabled | 6（3+3；1=故障 非 停用） |
| asset_service.go | A + 布尔维度 | Asset（models） | models.AssetStatusNormal/Stopped + models.AssetNBFStatusYes（新族） | 6（3 行尾注释 + 3 CASE WHEN，含 nbf_status） |
| excel_service.go | A | sys_dept 表（models.Dept） | models.DeptStatusNormal | 2（map 形态，见下） |
| excel_handler.go | A | sys_dept 表（models.Dept） | models.DeptStatusNormal | 1（Where 参数化） |
| workstation_device_service.go | A + D | WorkstationDevice（models）/ ops_info_points（子包） | models.WorkstationDeviceStatusNormal（新族）+ operations.InfoPointStatusNormal | 13（11 结构体 + 2 raw SQL） |

**替换形态：**
- CASE WHEN 聚合（8 个 service 共 24 处）：`fmt.Sprintf("COALESCE(SUM(CASE WHEN status = %d THEN 1 ELSE 0 END), 0) AS xxx", int(operations.Xxx))`——沿用批 1 范本；asset_service 的无类型常量直接传 `models.AssetStatusNormal`（无类型适配 %d 的 int 实参）。
- Where 参数化（excel_handler 1 处）：`Where("deleted_at IS NULL AND status = ?", models.DeptStatusNormal)`。
- raw SQL 参数化（workstation_device 2 处）：`AND ip.status = ?` + `Raw(rawSQL, workstationID(s), int(operations.InfoPointStatusNormal))`——两个 SQL 块各恰好 2 个占位符（workstation_id 在前、ip.status 在后），参数顺序核对一致。
- map/JSON 形态（excel_service 2 处）：`"status": models.DeptStatusNormal,`——map 值类型 any，typed 常量装箱正常；原 `// 正常` 行尾注释删除。

**excel_service.go map 形态 2 处实体判定：** :1975 与 :2029 均在 `ensureDeptGroupExists` / `ensureDeptExists` 内，向 `sys_dept` 表 Insert 的 `map[string]any` 默认行——实体为 sys_dept，簇 A，用 `models.DeptStatusNormal`（非 building/floor/workstation 模板行，plan 预估的"导入链路默认状态"确认 为部门自动建组/建部门场景）。

## workstation_device_service.go 11 处逐处判定（实体 → 常量）

全部 11 处结构体字面量的目标类型均为 `models.WorkstationDevice`，其 `Status` 字段为 `int`，model 注释 `// 状态: 0=正常, 1=停用`——簇 A 两态语义。逐处上下文：

| 行号（原） | 所在函数 | 场景 | 常量 |
|-----------|---------|------|------|
| 235 | GetADDevices | AD 设备实时转 WorkstationDevice（不落库） | models.WorkstationDeviceStatusNormal |
| 307 | GetAssetDevices | 资产设备实时转 WorkstationDevice（不落库） | 同上 |
| 578 | GetPhysicalDevices | R5 物理链路设备构造 | 同上 |
| 801 | AddDeviceManual | 手动添加设备落库（原注释 `// 默认正常` 已删） | 同上 |
| 862 | SyncFromAD | AD 设备同步落库 | 同上 |
| 953 | SyncFromAsset | 资产设备同步落库 | 同上 |
| 1165 | SetPrimaryAndSave | 合并保存为 manual 设备（tx 内 Create） | 同上 |
| 1247 | SetPrimaryAndSaveBySerial | 序列号键合并保存 | 同上 |
| 1573 | GetADDevicesByWorkstations | 批量 AD 转换（Phase 35 导出） | 同上 |
| 1692 | GetAssetDevicesByWorkstations | 批量资产转换 | 同上 |
| 1933 | GetPhysicalDevicesByWorkstations | 批量物理链路转换 | 同上 |

**为何不用既有族（关键判定）：** plan 提示"若语义为 workstation 启停用 models.WorkstationStatus*，为设备状态用相应 Device 常量"——逐处读上下文后确认两者均语义错配：`WorkstationStatus` 是工位三态（0=空闲/1=占用/2=维护），`DeviceStatus` 是网络设备在线态（0=在线/1=离线/2=未知）。把 0=正常 写成"空闲"或"在线"会造成语义漂移（正是 threat T-69-11 要防的），故按 69-01 登记机制新增 `WorkstationDeviceStatus` 族。另 2 处 raw SQL `ip.status = 0` 的 `ip` 是 `ops_info_points` 别名（注释明言"status ≠ 0 → 无设备返回"），实体为 OpsInfoPoint（子包三态）→ `operations.InfoPointStatusNormal`。

## 新增常量族（已登记锁值测试）

| 家族 | 常量 | 值 | 文件 | 形态 |
|------|------|----|------|------|
| WorkstationDeviceStatus | Normal / Stopped | 0 / 1 | internal/models/workstation_device.go | 无类型 int |
| AssetStatus | Normal / Stopped | 0 / 1 | internal/models/asset.go | 无类型 int |
| AssetNBFStatus | No / Yes | 0 / 1 | internal/models/asset.go | 无类型 int（拟报废布尔维度，与 status 正交） |

登记动作（69-01 机制两条）：watchedStatusPrefixes += 3 前缀；expectedStatusValues += 6 常量（锁值 74 → 80）。前缀无碰撞（AssetStatus 不匹配 AssetNBFStatus*，WorkstationDeviceStatus 不匹配 WorkstationStatus*）。无类型选择理由：三个字段（WorkstationDevice.Status / Asset.Status / Asset.NBFStatus）均为 int 型，typed 常量无法直接赋值，不改字段类型以守住"纯常量替换、行为零变化"。

## 白名单快照（批 2 后剩余 27 条目——批 3/批 4 工作面）

```
internal/api/v1/monitor/oper_log_handler.go=1
internal/api/v1/scheduler/job_handler.go=1
internal/api/v1/system/notice_handler.go=2
internal/services/addomain/dept_sync_service.go=1
internal/services/addomain/user_ou_service.go=3
internal/services/api_endpoint_service.go=1
internal/services/api_sender_service.go=1
internal/services/asset/fix_suggestion_monitor.go=1
internal/services/command_dispatch_service.go=4
internal/services/config_execution_service.go=8
internal/services/device_discovery_service.go=8
internal/services/duty_pool_service.go=4
internal/services/email_sender_service.go=1
internal/services/knowledge_service.go=4
internal/services/monitor/server_service.go=2
internal/services/notice_query_service.go=1
internal/services/notice_read_service.go=1
internal/services/notice_service.go=4
internal/services/notification_config_service.go=1
internal/services/operations/geocoding_service.go=1   # F 簇永久豁免
internal/services/oper_log_service.go=1
internal/services/rpa/credential_service.go=3
internal/services/scheduler/job_log_service.go=4
internal/services/scheduler/job_service.go=3
internal/services/vdi/vm_service_impl.go=6
internal/services/workorder/assignment.go=1
internal/services/workorder/base.go=8
```

## 验证结果

| 检查 | 结果 |
|------|------|
| `go build ./...` | 通过 |
| `go test ./internal/services/operations/` | ok |
| `go test ./internal/api/v1/...` | 全 ok（含 system 包，遗留 default-theme 改动未阻断该树） |
| `go test ./internal/models/ -run TestStatusConstants -v` | PASS（Stability + 14 家族子测试，锁值扩至 80 常量） |
| `bash scripts/check-status-literals.sh` | 退出码 0，无 ratchet-down 提示（白名单与实际命中精确一致） |
| grep `Status: *[01]\b` workstation_device_service.go | 0 命中 |
| grep `"status":[[:space:]]*[0-9]` excel_service.go（守护模式 d 口径） | 0 命中 |
| dedicated_line diff 仅 LineStatusNormal/Fault/Disabled 三常量 | 确认 |
| excel_config.go:146,265 零改动（Q3 决策） | git diff 为空，确认 |

## Deviations from Plan

1. **[Rule 2 - 语义判定偏离计划提示] 新增 WorkstationDeviceStatus / AssetStatus / AssetNBFStatus 三族**：plan 提示 workstation_device 复用 WorkstationStatus* 或 Device 常量，但两者语义错配（三态工位占用 / 在线离线），asset_service 的 Asset.Status 与 NBFStatus 同样无族可用。按 69-01 SUMMARY 明示的"后续批次新增家族照此登记"机制落地（前缀 + 锁值同步登记），符合 DICT-01 集中真相源目标；"不造第三份"约束针对双包既有族，未违反（diff 无任何既有族的重复定义）。
2. **[Rule 3 - 阻塞解除] 8 个 service 补 `"fmt"` import**：Sprintf 化 CASE WHEN 后编译所需（workstation_service 原有 fmt，其余 7 文件新增）。计划未列出该机械动作，属替换的必要伴随。
3. **[格式化伴随] gofmt 重排既有注释对齐**：asset.go / workstation_device.go / workstation_device_service.go 原非 gofmt-clean（struct tag 行尾注释对齐漂移），本次 gofmt -w 后 diff 行数放大（总计 15 文件 +405/−375，其中语义改动约 60 行）。纯空白/注释对齐，无行为变化。
4. **[流程] commit 前剔除并行会话索引污染**：Phase 70 会话的 11 个 staged 遗留文件（default_theme/settings）在 commit 前用 `git restore --staged` 全部剔出暂存区，本次 commit 仅含本任务 15 文件；遗留改动工作区状态未触碰。

## Known Stubs

无。

## Threat Flags

无新增威胁面。threat_model 三项缓解均落地：T-69-11（双包引错包）按 model struct 实际所在包逐文件判定并记录台账；T-69-12（行为变化）纯等值替换 + 包测试全绿；T-69-17（map 形态逃逸）excel_service 2 处清偿后模式 (d) grep 断言 0 命中。

## Self-Check: PASSED

15 个 modified 文件全部存在且在本 commit 内；commit ac33b2a 在 git log 命中；守护脚本退出码 0 复跑确认。
