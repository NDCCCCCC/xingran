---
phase: 69-dict-and-status-governance
plan: 01
subsystem: backend-status-constants
tags: [DICT-01, models, constants, regression-test, ratchet-guard, refactor]
requires: []
provides:
  - "internal/models 状态常量唯一真相源补齐（DictStatus/OperLogStatus/LoginLogStatus/JobLogStatus/VDIServerStatus/NoticeStatus 六新家族）"
  - "internal/models/status_constants_test.go AST 锁值测试（31 watched 前缀 / 74 常量）"
  - "scripts/check-status-literals.sh 四模式 ratchet 守护（基线 43 文件/149 命中）"
  - "批 1 替换完成：internal/services/system/ 5 文件 15 处裸字面量常量化，白名单收缩至 38 文件/134 命中"
affects:
  - "69-02（migration_208 字典 seed 直接引用 DictStatusNormal）"
  - "69-03/69-04/69-05（批 2-4 替换直接引用本 plan 常量清单与守护白名单）"
  - "69-08 DICT-04（CLAUDE.md Status Convention 改指向 models 常量）"
tech-stack:
  added: []   # 零新依赖：Go 标准库 go/ast + go/parser + bash/grep
  patterns:
    - "operlog regression_test 三件套改造（期望 map / AST 双向断言 / 多文件读取器）"
    - "ratchet 白名单守护脚本（只降不升 + F 簇永久豁免）"
    - "CASE WHEN 聚合 fmt.Sprintf + int(models.Xxx)；Where/Raw 一律参数化"
key-files:
  created:
    - internal/models/status_constants_test.go
    - scripts/check-status-literals.sh
  modified:
    - internal/models/dict.go
    - internal/models/log.go
    - internal/models/vdi.go
    - internal/models/notice_enhanced.go
    - internal/services/system/dict_service.go
    - internal/services/system/post_service.go
    - internal/services/system/role_service.go
    - internal/services/system/user_service.go
    - internal/services/system/widget_data_fetcher.go
decisions:
  - "以 internal/models 为状态常量唯一真相源；internal/constants（既有叶子工具包，无任何 status 常量）不放状态语义，不新建平行真相源"
  - "NoticeStatus 用 0=正常/1=关闭（Notice.Status 字段注释语义），命名 NoticeStatusClosed 而非 plan 草案的 Stopped"
  - "锁值测试双向断言 + 跨包同名常量值冲突检测（models 与 models/operations 双定义家族同名同值容忍、异值即 fail）"
  - "post/role statistics 结构体行尾裸值注释改写为常量名引用（守护只排整行注释，行尾注释计命中；不改写则白名单无法移除）"
metrics:
  duration: 约 50 分钟（2026-08-19 13:13–13:59 UTC）
  completed: 2026-08-19
---

# Phase 69 Plan 01: 后端常量真相源收口 Summary

一句话：models 层补齐 6 个缺失状态常量家族并用 AST 锁值测试（74 常量）+ 四模式 ratchet 守护脚本（43 文件/149 命中基线）封住改值与回归两条路，批 1 完成 services/system 5 文件 15 处裸字面量常量化。

## Task × Commit 对照表

| Task | 内容 | Commit |
|------|------|--------|
| T1 | models 缺失常量补齐 + status_constants_test.go AST 锁值测试 | `72608d2` |
| T2 | scripts/check-status-literals.sh ratchet 守护脚本（四模式 + 43 文件基线） | `d72a691` |
| T3 | 批 1：services/system 5 文件 15 处字面量替换 + 白名单收缩 | `da5d0a0` |

## 新增常量完整清单（批 2-4 / 69-02 / 69-08 直接引用）

| 家族 | 常量 | 值 | 所在文件 | 语义簇 |
|------|------|----|---------|--------|
| DictStatus | DictStatusNormal | 0 // 正常 | internal/models/dict.go | A（DictType.Status 与 DictData.Status 共用；69-02 seed 必用） |
| DictStatus | DictStatusDisabled | 1 // 停用 | internal/models/dict.go | A |
| OperLogStatus | OperLogStatusSuccess | 0 // 成功 | internal/models/log.go | C（69-05 批 4 oper_log_handler.go:190 用） |
| OperLogStatus | OperLogStatusFailure | 1 // 失败 | internal/models/log.go | C |
| LoginLogStatus | LoginLogStatusSuccess | 0 // 成功 | internal/models/log.go | C（auth.go recordLoginLog 实证 0=成功/1=失败） |
| LoginLogStatus | LoginLogStatusFailure | 1 // 失败 | internal/models/log.go | C |
| JobLogStatus | JobLogStatusSuccess | 0 // 成功 | internal/models/log.go | C（scheduler/cron.go:43 `Status: 0, // 成功` 实证） |
| JobLogStatus | JobLogStatusFailure | 1 // 失败 | internal/models/log.go | C |
| VDIServerStatus | VDIServerStatusNormal | 0 // 正常 | internal/models/vdi.go | A（vm_service_impl.go:163 注释实证 enable=1→status=0） |
| VDIServerStatus | VDIServerStatusStopped | 1 // 停用 | internal/models/vdi.go | A |
| NoticeStatus | NoticeStatusNormal | 0 // 正常 | internal/models/notice_enhanced.go | A（Notice.Status，区别于 PublishStatus） |
| NoticeStatus | NoticeStatusClosed | 1 // 关闭 | internal/models/notice_enhanced.go | A（Notice.Status 字段注释「0=正常 1=关闭」） |

核对后**已存在、未改动**：InfoPointStatus（operations/infopoint.go，0/1/2 三态）、DashboardStatus（dashboard.go，0/1）。`internal/models/base.go` 只读母版，`git diff` 为空。

## status_constants_test.go 锁值范围

- **watched 家族前缀集合（31 个）**：UserStatus、RoleStatus、MenuStatus、DeptStatus、PostStatus、DictStatus、Visible、Gender、ConfigType、ConfigIsSystem、ExecutionStatus、KnowledgeArticleStatus、PublishStatus、NoticeStatus、JobStatus、JobLogStatus、LoginLogStatus、OperLogStatus、LineStatus、WorkstationType、WorkstationStatus、DeviceType、DeviceStatus、DutyStatus、DutyPoolStatus、BuildingStatus、FloorStatus、RoomStatus、RoomDeviceStatus、VDIServerStatus、InfoPointStatus、DashboardStatus（较 plan 清单按 duty.go/dashboard.go 实际命名补登记 DutyStatus、DashboardStatus；ConfigType/DeviceType 为字符串枚举，AST 只收整数字面量故天然跳过）。
- **expectedStatusValues 条目数：74**（全部 watched 常量逐一锁值，含反转例外 VisibleShow=1/VisibleHidden=0、E 簇 KnowledgeArticleStatusPublished=1 与 PublishStatusPublished=1、B 簇 ExecutionStatus 0-4 全序、D 簇 LineStatus/InfoPointStatus 0/1/2）。
- **测试函数**：`TestStatusConstantsStability`（operlog 式双向断言：expected→actual 查缺失/改值，actual→expected 查偷加）+ `TestStatusConstantsCriticalFamilies`（14 个语义关键家族子测试，-v 输出可见覆盖 UserStatus/RoleStatus/Visible/KnowledgeArticleStatus/ExecutionStatus/LineStatus/DictStatus 等）。
- **跨包双定义冲突检测**：FloorStatus/RoomStatus/RoomDeviceStatus 在 `internal/models/operations.go`（package models）与 `internal/models/operations/`（package operations）双定义——读取器对同名常量同值容忍（已知双定义：含 `RoomDeviceStatusScrap` 与子包 `RoomDeviceStatusScrapped` 同值 2 的命名分叉）、异值直接 error。
- **登记机制（后续批次强制）**：新家族必须在同一改动中 (1) 前缀加入 watchedStatusPrefixes、(2) 全部常量登记 expectedStatusValues；漏 (2) 会被双向断言抓住，漏 (1) 是静默缺口（文件头注释已写明，reviewer 需双查）。69-03/69-04/69-05 新增的 ADAccountStatus*/RPACredentialStatus* 等家族照此登记。
- **自测**：人为把 DictStatusNormal 期望值改成 5 → 测试 FAIL（报错含定义文件 dict.go）；改回后 PASS。

## 守护脚本（scripts/check-status-literals.sh）

**四条命中模式**（对齐 RESEARCH A1）：
1. raw SQL 字符串：`status = [0-9]`
2. 结构体字面量：`Status: *[0-9]`（不锚定行首，`PublishStatus: 1` 同样命中）
3. 比较：`Status [=!]= *[0-9]`
4. map/JSON 形态：`"status":[[:space:]]*[0-9]`（excel_service.go:1975/:2029 的 `"status": 0,` 此前逃逸于前三模式）

**排除**：`*_test.go`、整行注释（剥前导空白后 `//` 开头）；扫描范围仅 `internal/api/v1` + `internal/services`。

**基线快照（ratchet）**：

| 时点 | 文件数 | 命中数 |
|------|--------|--------|
| 替换前（2026-08-19 注册） | 43 | 149 |
| 批 1 替换后（本次） | 38 | 134 |

- 批 1 移除条目：dict_service.go=4、post_service.go=4、role_service.go=4、user_service.go=2、widget_data_fetcher.go=1（合计 15 处）。
- 白名单剩余 38 条目中 `internal/services/operations/geocoding_service.go=1` 为 **F 簇永久豁免**（百度 API 返回码契约，条目带注释），其余 37 条目待批 2-4（69-03/69-04/69-05）逐批删除。
- 自测通过：白名单外新增（含 map 形态）/白名单内超基线均非零退出并打印违规行；`_test.go` 加命中行不改变退出码；`--baseline` 输出 43 行（当前 38 行）供回填。
- 未接 CI（plan 指明不改 CI 配置；脚本头注释已写明 Phase 63 挂载方式）。

## 批 1 替换明细（语义簇标注）

| 文件 | 处数 | 形态 | 语义簇 → 常量 |
|------|------|------|--------------|
| dict_service.go | 4（:52-53、:231-232） | CASE WHEN 聚合 ×2 组（DictType/DictData 两个 Statistics） | A → `fmt.Sprintf(...int(models.DictStatusNormal/Disabled))` |
| post_service.go | 2 CASE WHEN（:66-67）+2 行尾注释改写（:33-34） | CASE WHEN + 注释 | A → `int(models.PostStatusEnabled/Disabled)`；行尾注释 `// status = 0` → `// PostStatusEnabled` |
| role_service.go | 2 CASE WHEN（:55-56）+2 行尾注释改写（:68-69） | 同上 | A → `int(models.RoleStatusEnabled/Disabled)` |
| user_service.go | 2 CASE WHEN（:480-481） | CASE WHEN | A → `int(models.UserStatusEnabled/Disabled)`；新增 fmt import |
| widget_data_fetcher.go | 1（:199） | Raw SQL `AND m.status = 0` | **语义确认：簇 A，sys_menu（m）正常态过滤 → 参数化为 `AND m.status = ?` 传 `models.MenuStatusNormal`**（getUserPermissions 权限查询，非 dashboard 专属语义） |

替换通例：Where/Raw 一律参数化；仅 CASE WHEN 聚合用 Sprintf+int()（RESEARCH Pitfall 5）；无任何逻辑分支/返回值/SQL 结构改动（diff 仅 20 insertions/22 deletions）。

## 验证结果

| 检查 | 结果 |
|------|------|
| `go build ./...`（主工作树，含遗留改动） | 通过 |
| `go test ./internal/models/ -run TestStatusConstants -v` | PASS（Stability + 14 家族子测试全绿） |
| `bash scripts/check-status-literals.sh` | 退出码 0，白名单无 services/system 条目 |
| `grep 'status = [01]'` 非注释命中（services/system） | 0 |
| `go test ./internal/services/system/`（干净隔离 worktree @ da5d0a0） | **ok**（dict_statistics_test / post_statistics_test 等不回归） |
| `go test ./internal/models/`（同上） | ok |
| `go test ./...`（同上） | 43 包 ok；仅 tests/integration login_encryption 3 个用例失败（STATE 已文档化的 6 周+ 存量失败，与本 plan 无关，git 层面未触碰） |

> 主工作树中 `go test ./internal/services/system/` 因**前一会话未提交的 default-theme 遗留改动**编译失败（settings_service.go 移除了 configService 字段，已提交的 settings_service_test.go 仍引用）——与本 plan 无关、禁止触碰，详见 `deferred-items.md` #1；故包级测试在 `git worktree add --detach` 的干净树上验证通过。

## Deviations from Plan

1. **[计划假设修正] internal/constants 已存在**：plan 验收写「`ls internal/constants` 失败为正确状态」，实际该目录是既有叶子工具包（Redis key/分页/时间/UUID，quick-260812-wu5 引入，无任何 status 常量）。「不新建平行真相源」的实质约束完全满足——本 plan 未在该包添加任何内容，状态常量只存在于 internal/models。
2. **[Rule 1 - 计划内部矛盾修正] post/role 行尾注释改写而非保留**：plan T3 第 2 条说「:33-34 行尾注释保留不动（守护脚本已排除注释行）」，但守护脚本按 T2 规范只排除**整行**注释、行尾注释计命中——保留则白名单无法移除（post/role 各剩 2 命中）。按 T3 第 5 条「替换后删除行尾裸值注释」的通例执行：改写为常量名引用（`// PostStatusEnabled`）。
3. **[命名以实际语义为准] NoticeStatusClosed 而非草案的 Stopped**：Notice.Status 字段注释为「0=正常 1=关闭」，采用 NoticeStatusNormal/NoticeStatusClosed。
4. **[登记补全] watched 集合较 plan 清单增加 DutyStatus、DashboardStatus**：plan 授权「以实际文件为准登记」，duty.go 的 DutyStatus 三态与 dashboard.go 的 DashboardStatus 一并纳入锁值。
5. **[流程] commit message body 需 ≤100 字符/行**：commitlint hook 首次提交被拒，重排后通过（三 commit 均已过 hook）。

## Known Stubs

无。

## Threat Flags

无新增威胁面（守护脚本为本地工具，无网络/文件写入面；T-69-01/02/03 缓解措施均已按 threat_model 落地）。

## Self-Check: PASSED

11 个 key-files 全部存在；3 个 task commit（72608d2 / d72a691 / da5d0a0）全部在 git log 中命中。
