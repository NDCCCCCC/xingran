---
phase: 69-dict-and-status-governance
plan: 02
subsystem: dict-seed-migration
tags: [DICT-02, migration_208, sys_dict, seed, sqlite, idempotent]
requires:
  - "69-01: models.DictStatusNormal/Disabled 常量家族（本 plan seed 的 Status 唯一引用来源）"
provides:
  - "migration_208 11 组字典 seed（8 组 archive 存量重建 + 3 组新增），组级幂等 + 单事务 + WARN 非阻断"
  - "database.go PG advisory-lock 分支 + SQLite else 分支双分支挂载（dev sqlite 可见）"
  - "dev 库 sys_dict_type/sys_dict_data 从 0/0 恢复到 11/44，4 个既有 useDict 页数据前提就绪"
  - "sqlite 内存库 4 用例测试锁定幂等/不复活/isDefault 恰一/软删防复活语义"
affects:
  - "69-07（前端四页 type 下拉 useDict 迁移——直接消费本 plan 的 11 组数据）"
  - "69-08（DICT-04 字典链路端到端 checkpoint——本 plan 是其数据前提）"
tech-stack:
  added: []   # 零新依赖：GORM + glebarez/sqlite（仓内既有）
  patterns:
    - "migration_203 GORM count 查重 + migration_207 幂等语义四注释 + db.Transaction 混合范本"
    - "组级查重走 Unscoped（软删防复活 + 防 uniqueIndex 占位撞约束）"
    - "database.go 双分支各一次 MigrateNNN 调用（207 挂法照抄，规避 202-206 PG-only 前车之鉴）"
key-files:
  created:
    - internal/core/db/migrations/migration_208_dict_seed.go
    - internal/core/db/migrations/migration_208_dict_seed_test.go
  modified:
    - internal/core/db/database.go
decisions:
  - "组级查重用 Unscoped count：软删的 dict_type 同样视为组已存在（防复活 + 防软删行占位 uniqueIndex 致每次启动 INSERT 撞约束 WARN 刷屏）"
  - "isDefault 取值规则：有 archive/模型来源照抄（含 gorm default 注释），无来源组取组内第一条——sys_user_sex 默认 \"2\" 保密（User.Gender gorm default:2）而非 0=男"
  - "duty_holiday_type 默认 custom（Holiday.HolidayType gorm default:'custom'）"
  - "dashboard_* 三组与 workorder_* 两组不重建（前端零 useDict 消费，planner Q4 圈定）"
  - "DictSort 一律 1 起递增（archive/169 的 0 弃用，对齐 plan 规范）"
metrics:
  duration: 42 分钟（2026-08-19 06:07–06:50 UTC）
  completed: 2026-08-19
---

# Phase 69 Plan 02: DICT-02 — sys_dict seed 链路重建 Summary

一句话：migration_208 用 GORM 结构体 seed 重建 11 组字典（8 组 archive 存量 + 3 组新增），组级 Unscoped count 幂等 + 单事务 + 双分支挂载，dev sqlite 库从 0/0 行恢复到 11 类型/44 数据值且二次启动零写入。

## Task × Commit 对照表

| Task | 内容 | Commit |
|------|------|--------|
| T1 | migration_208_dict_seed.go（11 组 seed）+ migration_208_dict_seed_test.go（4 用例） | `c3a1e4a` |
| T2 | database.go PG/SQLite 双分支注册 + dev 库 smoke（0/0 → 11/44 → 二次启动不变） | `a8687f7` |

## 11 组 dictType 完整清单（69-07 / 69-08 直接引用）

| # | dictType | dictName | 值数 | 值来源 | isDefault 项 | ListClass 来源 |
|---|----------|----------|------|--------|--------------|----------------|
| 1 | network_device_type | 网络设备类型 | 5 | archive/legacy-2026-06-15/002 + models.DeviceType 注释（ap label 取模型注释「无线接入点」） | **router**（archive 全 false → 取组内第一条） | archive 002（全 default） |
| 2 | ops_dedicated_line_type | 专线类型 | 6 | archive/legacy-2026-06-15/047；六值 internet/intranet/cloud_desktop/mpls/fiber/leased_line 与 excel_config.go:265 逐字一致（Pitfall 6 硬约束已核） | **internet**（archive 047） | archive 047 |
| 3 | ops_isp | 运营商 | 5 | archive/legacy-2026-06-15/048 | **telecom**（archive 048） | archive 048 |
| 4 | ops_info_point_type | 信息点类型 | 2 | archive/legacy-2026-06-15/033 | **network**（archive 033） | archive 033 |
| 5 | asset_reconciliation_conflict_type | 资产对账冲突类型 | 6 | archive/applied/migration_169 + **migration_196 对齐后 label/listClass**（B=error、F=warning 等） | **A**（169 全 false → 取组内第一条） | 169 + 196 修正 |
| 6 | asset_reconciliation_exception_action | 资产对账例外动作 | 5 | archive/applied/migration_169 | **no_alert**（169） | 169 |
| 7 | asset_reconciliation_severity | 资产对账严重度 | 4 | archive/applied/migration_169 | **low**（169） | 169 |
| 8 | asset_reconciliation_status | 资产对账状态 | 2 | archive/applied/migration_169 | **open**（169） | 169 |
| 9 | ops_workstation_type | 工位类型（新增） | 3 | models.WorkstationType（int 0/1/2 → dictValue "0"/"1"/"2"）+ excel_config.go:146 | **"0" 固定工位**（Workstation.WorkstationType gorm default:0） | 无来源 → nil |
| 10 | sys_user_sex | 用户性别（新增） | 3 | models.Gender（"0"=男/"1"=女/"2"=保密） | **"2" 保密**（User.Gender gorm default:2——非 0=男） | 无来源 → nil |
| 11 | duty_holiday_type | 节假日类型（新增） | 3 | models.HolidayType（legal/workday/custom） | **custom**（Holiday.HolidayType gorm default:'custom'） | 无来源 → nil |

合计：sys_dict_type = 11 行，sys_dict_data = 44 行。

**既有前端消费方（seed 落地后自动恢复，零代码改动）：** dedicated-lines/index.tsx（ops_dedicated_line_type + ops_isp）、info-points/index.tsx（ops_info_point_type）、asset/reconciliation/exceptions/index.tsx + HealthBadge.tsx（conflict_type/severity）。

## dev 库行数快照与幂等证据（T2 smoke 实测）

| 时点 | sys_dict_type | sys_dict_data | 证据 |
|------|---------------|---------------|------|
| seed 前（基线） | 0 | 0 | sqlite3 直查（与 69-RESEARCH A4 实测一致） |
| 首次启动后 | **11** | **44** | 启动日志 11 条「migration 208: 字典组 X seed 完成」；GROUP BY dict_type 分布与上表值数完全一致 |
| 二次启动后 | **11** | **44** | 第二次启动日志 0 条 seed 写入（grep 计数 = 0），行数不变——真实库幂等成立 |

- 后端以 `go build` 二进制后台启动（:9000 正常监听），验证后进程终止、临时二进制与 pid 文件已清理。
- 未走 T2 备选路径（临时文件库）——后端环境完好，全流程自动化验证完成；字典管理页 UI 目检按 plan 归入 69-08 checkpoint。

## 剔除决策记录（planner Q4 圈定，本 plan 遵守）

| 剔除项 | 理由 |
|--------|------|
| dashboard_widget_type / dashboard_template_scope / dashboard_scope | 前端零 useDict 消费（RESEARCH A2 假设 + plan Source Audit 复核），多 seed 无消费方 |
| workorder_type / workorder_priority | workorder/orders/constants.ts 实测无 OPTIONS 下拉，无消费方 |

剔除的执行层保障：migration_208 文件内 `dashboard_` grep 零命中（验收门），注释以「仪表盘三组」表意。

## 幂等语义实现（对齐 migration_207 四条注释 + 一处 Rule 2 加固）

1. **组级快速路径**：`db.Unscoped().Model(&DictType{}).Where("dict_type = ?").Count` — dict_type 已存在（**含软删**）整组跳过。
2. **组内不查重**：进入事务的组必为全新组。
3. **单事务**：组内 Type + Data 批量 Create 任一失败整组回滚返回 error，调用方 `applogger.Errorf("字典 seed 失败 (非阻断,留待下次启动)")` 不阻断启动。
4. **双方言**：纯 GORM 结构体 Create（文件内 `INSERT INTO` grep 零命中），PG 与 SQLite 分支各挂载一次（`grep -c Migrate208DictSeed database.go` = 2）。

## 验证结果

| 检查 | 结果 |
|------|------|
| `go build ./...`（主工作树） | 通过 |
| `go test ./internal/core/db/migrations/ -run TestMigrate208 -v` | **4 用例全 PASS**（SeedsAndIdempotent / RespectsExistingGroups / IsDefaultSemantics / RespectsSoftDeletedGroups） |
| `go test ./internal/core/db/...` | ok（db 4.452s + migrations 0.293s） |
| 验收 grep 门 | `INSERT INTO`=0、`dashboard_`=0、裸 `Status: 0`=0、`int(models.DictStatusNormal)`=55（11 类型+44 数据全覆盖）、六值与 excel_config 一致 |
| dev 库 smoke | 0/0 → 11/44 → 二次启动 11/44 不变（见上表） |
| `go test ./...`（干净 detached worktree @ a8687f7） | 仅 `tests/integration`（login_encryption 存量失败，STATE/69-01-SUMMARY 已文档化的 6 周+ 遗留，与本 plan 无关）；其余全部 ok，与 69-01 基线零差异 |

> 全仓回归在 `git worktree add --detach` 干净树执行（69-01 先例）：主树 `internal/services/system` 被遗留 default-theme 改动阻断编译（非本 plan 引入，禁止触碰）；worktree 验证后已即用即删。

## Deviations from Plan

1. **[Rule 2 - 加固] 组级查重用 Unscoped 而非默认 scope**：plan 行文写 `db.Model(&models.DictType{}).Count`，但 DictType 带软删（BaseModel.DeletedAt）——默认 scope 下管理员软删的组 count=0 会被 seed 复活，且软删行占位 dict_type uniqueIndex 会让每次启动 INSERT 撞唯一约束（WARN 刷屏、永不自愈）。Unscoped 把软删视为「组已存在」，同时满足 truth #2「不复活」与 T-69-05 mitigate 处置。新增第 4 个测试用例 TestMigrate208RespectsSoftDeletedGroups 锁定该语义。
2. **[Rule 1 - 验收门字面冲突] 头注释「dashboard_*」改写为「仪表盘三组」**：plan 验收要求本文件 `dashboard_` grep 零命中，原注释字面命中该门；改写后语义不变、门通过。
3. **[计划细化] isDefault 无来源组的取值规则落地**：plan 只要求「每组恰一条 IsDefault=true」未指定无来源组取哪条——落地规则为「模型 gorm default 注释优先（工位 "0"/性别 "2"/假日 custom），archive 全 false 的两组（network_device_type/conflict_type）取组内第一条」，已记入 STATE Decisions。
4. **[流程] 11 组 DictSort 一律 1 起**：archive/169 的 DictSort:0 弃用（plan 规范「1 起递增」）。

## Auth Gates

无。

## Known Stubs

无。

## Threat Flags

无新增威胁面。threat_model 四项处置全部落地：T-69-05（组级 Unscoped 跳过 + 事务 + WARN）、T-69-06（只 seed type/性别/假日枚举，status 不进字典）、T-69-07（双分支挂载 + grep=2 门 + dev 行数断言）、T-69-08（六值与 excel_config 逐字一致已核）。

## Self-Check: PASSED

3 个 key-files 全部存在；2 个 task commit（c3a1e4a / a8687f7）均在 git log 命中。
