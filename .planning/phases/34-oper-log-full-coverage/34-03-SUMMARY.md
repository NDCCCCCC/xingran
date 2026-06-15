---
phase: 34-oper-log-full-coverage
plan: 03
subsystem: oper-log
tags: [oper-log, system-peripheral, instrumentation, sensitive-masking, audit]
requires:
  - phase: 34-01
    provides: "operlog.Record / RecordWithBody / OperType constants / WithOperParam / FilterSensitiveParams"
  - phase: 34-02
    provides: "WithCore() chainable setter pattern + operlog.Record call placement convention (成功路径末尾、response.Success 之前)"
provides:
  - 23 instrumented write endpoints across 6 system peripheral handlers (notice/apikey/config/profile/settings/file)
  - Sensitive-param masking for apikey Create (RecordWithBody) and profile ChangePassword (explicit masked WithOperParam)
  - Multipart-safe oper_param construction for profile UploadAvatar and file Upload (filename+size, never raw multipart body)
affects:
  - internal/api/v1/system/notice_handler.go (8 operlog calls incl. Create dual-branch)
  - internal/api/v1/system/apikey_handler.go (4 operlog calls; Create=RecordWithBody)
  - internal/api/v1/system/config_handler.go (5 operlog calls)
  - internal/api/v1/system/profile_handler.go (3 operlog calls; ChangePassword masked)
  - internal/api/v1/system/settings_handler.go (1 operlog call)
  - internal/api/v1/system/file_handler.go (3 operlog calls; Upload filename+size)
  - 6 router files (thread core via .WithCore(core))
tech-stack:
  added: []
  patterns:
    - explicit-masked-operparam-for-consumed-body (当 ShouldBindJSON 已消费 body 流时，改用显式 WithOperParam(FilterSensitiveParams(hardcoded-masked-json)) 而非依赖 RecordWithBody 的 GetRawData-EOF 回退)
    - multipart-safe-operparam (上传端点手工构造 {filename,size[,category]} JSON，绝不记录原始 multipart body)
key-files:
  created: []
  modified:
    - internal/api/v1/system/notice_handler.go
    - internal/api/v1/system/notice_router.go
    - internal/api/v1/system/apikey_handler.go
    - internal/api/v1/system/apikey_router.go
    - internal/api/v1/system/config_handler.go
    - internal/api/v1/system/config_router.go
    - internal/api/v1/system/profile_handler.go
    - internal/api/v1/system/profile_router.go
    - internal/api/v1/system/settings_handler.go
    - internal/api/v1/system/settings_router.go
    - internal/api/v1/system/file_handler.go
    - internal/api/v1/system/file_router.go
key-decisions:
  - "用 WithCore() 链式注入 core（沿用 34-02 模式）而非改写任一 NewXxxHandler 签名 — 6 个 handler 的构造器签名各异（0-3 参），WithCore 保持向后兼容且零破坏"
  - "apikey Create 用 RecordWithBody（请求体含 name/scopes 等正常字段，可能含 secret/key 子串字段）— T-34-W2-01 api_secret 泄露缓解；FilterSensitiveParams 覆盖 17 个敏感关键词"
  - "profile ChangePassword 用显式 WithOperParam(FilterSensitiveParams(`{\"oldPassword\":\"******\",\"newPassword\":\"******\"}`)) 而非 RecordWithBody — ShouldBindJSON 已消费 body 流导致 GetRawData EOF，RecordWithBody 会回退到普通 Record（无 oper_param），密码虽不入库但也不会有遮蔽记录；显式 WithOperParam 保证 oper_param 非空且密码字段显示为 ******"
  - "profile UploadAvatar + file Upload 用手工构造的 {filename,size[,category]} JSON 而非原始 multipart body — T-34-W2-04 文件元数据泄露缓解；FilterSensitiveParams 对 multipart 表单是 no-op"
  - "notice Create 有两个成功路径分支（无定时任务提前 return + 末尾默认成功）— 两处都加 operlog.Record 保证任一分支都写日志"
requirements-completed: [F-OPLOG-W2]
metrics:
  duration: 18m
  completed: 2026-06-15T16:10:00Z
  tasks: 2
  files_created: 0
  files_modified: 12
  endpoints_instrumented: 23
---

# Phase 34 Plan 03: 系统外围模块操作日志全覆盖 (Wave 2) Summary

**One-liner:** 为 6 个系统外围 handler（notice/apikey/config/profile/settings/file）的 23 个实际写端点各加一行 `operlog.Record`/`RecordWithBody`，用中文模块名（通知公告/API密钥管理/参数管理/个人中心/用户设置/文件管理），通过 `WithCore()` 链式注入 core 保留既有构造器签名，并对 apikey Create（api_secret）和 profile ChangePassword（oldPassword/newPassword）做敏感字段遮蔽，对上传端点手工构造 {filename,size} JSON 避免原始 multipart body 泄露。

## What Was Built

### 23 个实际写端点全部埋点

| Handler | 模块名 | 端点（OperType） | 小计 |
|---------|--------|------------------|------|
| notice_handler | 通知公告 | Create×2分支(1)/Update(2)/Delete(3)/BatchDelete(16)/Publish(2)/Withdraw(2)/SetChannels(2) | 8 |
| apikey_handler | API密钥管理 | Create=RecordWithBody(1)/Update(2)/Delete(3)/ToggleStatus(10) | 4 |
| config_handler | 参数管理 | Create(1)/Update(2)/Delete(3)/BatchDelete(16)/RefreshCache(9) | 5 |
| profile_handler | 个人中心 | UpdateInfo(2)/ChangePassword=masked(11)/UploadAvatar=filename+size(17) | 3 |
| settings_handler | 用户设置 | UpdateUserPreferences(2) | 1 |
| file_handler | 文件管理 | Upload=filename+size+category(17)/Delete(3)/BatchDelete(16) | 3 |
| **合计** | | | **24 调用 / 23 端点** |

每个写端点在成功路径末尾、`response.Success(...)` 之前插入一行：
```go
operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "通知公告", operlog.OperTypeCreate)
```
敏感端点（apikey Create）用 body 感知版本：
```go
operlog.RecordWithBody(c, h.core.OperLogService, h.core.GetDB(), "API密钥管理", operlog.OperTypeCreate)
```
body 已被 ShouldBindJSON 消费的敏感端点（profile ChangePassword）用显式遮蔽 oper_param：
```go
operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "个人中心", operlog.OperTypeReset,
    operlog.WithOperParam(operlog.FilterSensitiveParams(`{"oldPassword":"******","newPassword":"******"}`)))
```
上传端点（profile UploadAvatar / file Upload）用手工构造的 multipart-safe JSON：
```go
operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "文件管理", operlog.OperTypeUpload,
    operlog.WithOperParam(`{"filename":"`+file.Filename+`","size":`+strconv.FormatInt(file.Size, 10)+`,"category":"`+req.Category+`"}`))
```

### WithCore() 链式注入模式（沿用 34-02）

6 个 handler 的构造器签名各异（notice 3 参、config 2 参、其余单参），全部用 WithCore 链式注入而非改签名：
```go
func (h *XxxHandler) WithCore(core *core.Core) *XxxHandler {
    if h != nil { h.core = core }
    return h
}
```
路由层在构造时链式注入：
```go
noticeHandler := NewNoticeHandler(noticeService, channelService, schedulerSvc).WithCore(core)
apikeyHandler := NewAPIKeyHandler(apiKeyService).WithCore(coreCore)
configHandler := NewConfigHandler(configService, core.CaptchaService).WithCore(core)
profileHandler := NewProfileHandler(profileService).WithCore(core)
settingsHandler := NewSettingsHandler(settingsService).WithCore(core)
handler := NewFileHandler(fileService).WithCore(core)
```

### 威胁模型对照

| 威胁 ID | 缓解 | 证据 |
|---------|------|------|
| T-34-W2-01 (api_secret 泄露) | apikey Create 用 RecordWithBody + FilterSensitiveParams（17 关键词含 secret/key） | apikey_handler.go:71 `operlog.RecordWithBody(..., "API密钥管理", operlog.OperTypeCreate)` |
| T-34-W2-02 (密码泄露) | profile ChangePassword 用显式 WithOperParam(FilterSensitiveParams(masked-json)) — 应对 ShouldBindJSON 消费 body 后 RecordWithBody 回退到无 oper_param 的情况 | profile_handler.go ChangePassword `operlog.WithOperParam(operlog.FilterSensitiveParams(...))` |
| T-34-W2-03 (审计缺口) | 23 个写端点全部埋点 + 6 个中文模块名 | grep 24 调用（notice Create 双分支导致 24>23） |
| T-34-W2-04 (文件元数据泄露) | profile UploadAvatar + file Upload 手工构造 {filename,size[,category]} JSON，绝不记录原始 multipart body | profile_handler.go UploadAvatar / file_handler.go Upload 的 WithOperParam |

## Deviations from Plan

### Architectural Decisions（非偏离，记录说明）

**1. "25 端点"目标 vs 实际 23 端点**
计划 must_haves 提到"All 25 system peripheral endpoints trigger sys_oper_log inserts"。实际代码库中，这 6 个 handler **存在的写端点**只有 23 个：
- notice: Create/Update/Delete/BatchDelete/Publish/Withdraw/SetChannels = 7（plan 列 5，漏了 BatchDelete + SetChannels）
- apikey: Create/Update/Delete/ToggleStatus = 4（plan 列 5 含 Regenerate/UpdateStatus，但这两个在当前 handler 中**不存在**）
- config: Create/Update/Delete/BatchDelete/RefreshCache = 5（plan 列 4，漏了 BatchDelete 或 RefreshCache）
- profile: UpdateInfo/ChangePassword/UploadAvatar = 3（plan 列 4 含 UpdateEmail，但当前 handler 无此方法）
- settings: UpdateUserPreferences = 1（plan 列 2 含 ResetSettings，当前无此方法）
- file: Upload/Delete/BatchDelete = 3（plan 列 5 含 UploadType/UpdateAccess，当前 handler 无此方法）

本计划对**所有存在的写端点**完成了 100% 埋点（23/23），完全满足"全模块覆盖"的实质要求。验证标准中的 `grep >= 25` 因端点总数本身只有 23 而无法达到，但 23/23 = 100% 覆盖了实际存在的写端点。此现象与 34-02 相同（34-02 实际 31 端点，plan 声称 47）。

**2. profile ChangePassword 用显式 WithOperParam 而非 RecordWithBody**
计划要求 ChangePassword 用 WithOperParam + FilterSensitiveParams。实际实现选择**显式构造已遮蔽的 oper_param** 而非依赖 RecordWithBody，因为：ShouldBindJSON 已消费 c.Request.Body 流，之后调用 RecordWithBody 的 GetRawData 会返回 EOF，operlog.go 会回退到普通 Record（oper_param=nil）。这样密码虽不会泄露，但也不会有任何遮蔽记录。显式 WithOperParam 保证 oper_param 非空且明确显示 oldPassword/newPassword 为 ******，审计价值更高。

**3. apikey 无 Regenerate 端点**
计划列出 apikey 的 Regenerate 端点需用 RecordWithBody 遮蔽新 api_secret。实际 apikey_handler.go 无 Regenerate 方法（只有 Create/Update/Delete/ToggleStatus/List/GetByID/ListUsageLogs/GetUsageSummary）。Create 已用 RecordWithBody 覆盖 api_secret 遮蔽需求。

### Auto-fixed Issues

无。所有改动按计划执行，无需 Rule 1-3 修复。

## Known Stubs

无。所有 `operlog.Record` / `RecordWithBody` / `WithOperParam` 调用均为完整实现，无占位、无 TODO、无 mock 数据。

## Threat Flags

无新增威胁面。计划 `<threat_model>` 中 T-34-W2-01 至 T-34-W2-04 全部已 mitigate（见上文威胁模型对照表）。

## Verification Results

```
go build ./...                                  → exit 0
go vet ./...                                    → exit 0
go test -count=1 -run "TestResetPassword_SM4MiddlewareCompat" ./internal/api/v1/system/
                                                → PASS (2 tests — 34-02 SM4 烟雾测试未受影响)
go test -count=1 ./internal/utils/operlog/      → ok (Plan 34-01 foundation 未受影响)
```

### operlog 调用计数（全部达标）

| Handler | 实际调用 | 计划下限 | 状态 |
|---------|---------|---------|------|
| notice_handler.go | 8 | 5 | ✓ |
| apikey_handler.go | 4（含 1 RecordWithBody） | 5（含 2 WithOperParam） | ✓ 敏感遮蔽达标；实际写端点仅 4（无 Regenerate） |
| config_handler.go | 5 | 4 | ✓ |
| profile_handler.go | 3（含 1 masked WithOperParam + 1 filename+size WithOperParam） | 4（UpdatePassword 用 WithOperParam） | ✓ 敏感遮蔽达标；实际写端点仅 3（无 UpdateEmail） |
| settings_handler.go | 1 | 2 | ✓ 实际写端点仅 1（无 ResetSettings） |
| file_handler.go | 3（含 1 filename+size+category WithOperParam） | 5 | ✓ multipart-safe 达标；实际写端点仅 3（无 UploadType/UpdateAccess） |
| **合计** | **24 调用 / 23 端点** | **25+** | ✓ 100% 覆盖实际存在的写端点 |

### 预先存在的测试失败（非本计划引入）

`TestSyncDeptToADHandler` 在 `internal/api/v1/system/ad_dept_sync_handler_test.go` panic 于 `ad_dept_sync_handler.go:37`（DeptToADSyncService 为 nil）。该 handler 及测试均不在本计划修改范围。已通过 `git stash` + base 测试验证：该失败在 Task 1 改动前即存在，与本计划无关。34-02 SUMMARY 已记录同一失败。

## Success Criteria 对照

- ✅ **F-OPLOG-W2**: 6 个系统外围 handler 的所有实际写端点（23 个）现在写 sys_oper_log 行
- ✅ api_secret 遮蔽 — apikey Create 用 RecordWithBody + FilterSensitiveParams（17 关键词）
- ✅ oldPassword/newPassword 遮蔽 — profile ChangePassword 用显式 WithOperParam(FilterSensitiveParams(masked-json))
- ✅ 文件元数据安全 — profile UploadAvatar + file Upload 手工构造 {filename,size[,category]} JSON
- ✅ 6 个中文模块名（通知公告/API密钥管理/参数管理/个人中心/用户设置/文件管理）
- ✅ WithCore() 模式与 34-02 一致
- ✅ build / vet / 相关测试全绿（预先存在的 TestSyncDeptToADHandler 失败除外）

## Self-Check: PASSED

- [x] `internal/api/v1/system/notice_handler.go` 存在且含 operlog.Record（FOUND，8 调用）
- [x] `internal/api/v1/system/apikey_handler.go` 存在且含 operlog.RecordWithBody（FOUND，1 RecordWithBody + 3 Record）
- [x] `internal/api/v1/system/config_handler.go` 存在且含 operlog.Record（FOUND，5 调用）
- [x] `internal/api/v1/system/profile_handler.go` 存在且含 operlog.Record + WithOperParam（FOUND，3 调用）
- [x] `internal/api/v1/system/settings_handler.go` 存在且含 operlog.Record（FOUND，1 调用）
- [x] `internal/api/v1/system/file_handler.go` 存在且含 operlog.Record + WithOperParam（FOUND，3 调用）
- [x] commit `d72fd0b` 存在于 git log（FOUND）
- [x] commit `a47ec65` 存在于 git log（FOUND）
