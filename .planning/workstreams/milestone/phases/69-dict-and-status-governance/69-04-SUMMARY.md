---
phase: 69
plan: 04
subsystem: backend-status-governance
tags: [dict-01, status-constants, literal-replacement, guard-ratchet, e-cluster-reversal]
requires: ["69-01 (守护脚本+锁值测试)", "69-03 (批2 白名单收缩)"]
provides: ["批3六目录清零", "ADAccountStatus 家族 (D簇三态)", "RPACredentialStatus 家族 (簇A)", "锁值 80→85 常量"]
affects: ["internal/services/vdi", "internal/services/addomain", "internal/services/{notice,knowledge}_service.go", "internal/services/notice_{query,read}_service.go", "internal/services/rpa", "internal/services/scheduler", "internal/models"]
tech-stack:
  added: []
  patterns: ["Where 参数化 (models.Xxx 常量作 SQL arg)", "CASE WHEN 聚合 Sprintf + int(models.Xxx)", "typed 常量→int 字段显式 int() 转换", "services 层本地常量 alias 到 models 真相源"]
key-files:
  created: []
  modified:
    - internal/services/vdi/vm_service_impl.go
    - internal/services/addomain/account_pool.go
    - internal/services/addomain/dept_sync_service.go
    - internal/services/addomain/user_ou_service.go
    - internal/services/notice_service.go
    - internal/services/notice_query_service.go
    - internal/services/notice_read_service.go
    - internal/services/knowledge_service.go
    - internal/services/rpa/credential_service.go
    - internal/services/scheduler/job_service.go
    - internal/services/scheduler/job_log_service.go
    - internal/models/ad_service_account.go
    - internal/models/rpa.go
    - internal/models/status_constants_test.go
    - scripts/check-status-literals.sh
decisions:
  - VDIResourceGroup 命中复用 VDIServerStatus* 家族（vdi.go:81 注释实证同 0=正常/1=停用，计划只允许 ADAccountStatus*/RPACredentialStatus* 两个补缺家族）
  - account_pool.go 的 AccountStatus* 本地常量改为 models.ADAccountStatus* 的别名（单一真相源，外部消费方 ad_account_handler.go 与测试零改动）
  - ADAccountStatus*/RPACredentialStatus* 采用无类型 int 常量（对齐批 2 先例：Status 字段为 int 的实体直接赋值/比较免转换）
metrics:
  duration: 14m
  completed: 2026-08-19
---

# Phase 69 Plan 04: DICT-01 批 3 — vdi/addomain/notice/knowledge/rpa/scheduler 替换（簇 A + D 账号池 + E 反转）Summary

**一句话:** 六目录 30 处裸 status 字面量全部常量化——E 簇反转（knowledge/notice 已发布=1）与 D 簇账号池三态零误替，批内补缺 ADAccountStatus\*/RPACredentialStatus\* 两家族完成锁值登记闭环（80→85 常量），守护白名单 27→17 文件。

## Commit 对照表

| 批次 | Commit | 文件数 | 替换处数 | 说明 |
|------|--------|--------|----------|------|
| 批 3（本 plan，单 commit） | `8620d9b` | 15 | 白名单命中 30 + 守护范围外 11（models 裸比较 4 + switch case 3 + account_pool 常量 alias 3 + NoticeStatus int 转换并入 30 计数） | refactor(services): replace raw status literals with models constants (batch 3) per DICT-01 |

**守护脚本基线前后快照佐证:**
- 批 3 前（69-03 后）：27 文件；六目录命中 = 10 文件 / 30 处（vdi 6 + addomain 4 + notice\* 6 + knowledge 4 + rpa 3 + scheduler 7）
- 批 3 后：`bash scripts/check-status-literals.sh` 退出码 0，白名单 17 文件，六目录条目 0
- `go build ./...` exit 0；`go test ./internal/models/` PASS；vdi/addomain/rpa/scheduler 包测试 ok；根 `go test ./internal/services/` ok（376.8s，含 notice/knowledge）

## 语义簇映射台账（每文件 → 簇 → 所用常量）

| 文件 | 簇 | 实体/字段判定 | 所用常量 |
|------|----|--------------|----------|
| vdi/vm_service_impl.go :66 | A | VDIServer.Status（获取客户端选启用服务器） | `models.VDIServerStatusNormal` |
| vdi/vm_service_impl.go :163-167 | A | **VDIResourceGroup** enable→status 映射变量（int 局部变量，显式 int() 转换） | `int(models.VDIServerStatusStopped)` / `int(models.VDIServerStatusNormal)` |
| vdi/vm_service_impl.go :444 | A | VDIServer.Status（vdiServerID 查询） | `models.VDIServerStatusNormal` |
| vdi/vm_service_impl.go :453 | A | **VDIResourceGroup**（ListResourceGroups 本地库查询） | `models.VDIServerStatusNormal` |
| vdi/vm_service_impl.go :513 | A | VDIServer.Status（CreateVM 前置校验） | `models.VDIServerStatusNormal`（与 id 双参数化） |
| vdi/vm_service_impl.go :591 | A | VDIServer.Status（ListVMs 启用计数） | `models.VDIServerStatusNormal` |
| addomain/dept_sync_service.go :168 | A | Department.Status（根部门查询） | `models.DeptStatusNormal` |
| addomain/user_ou_service.go :202 | A | **ADConfig.Status**（获取启用 AD 配置） | `models.ADConfigStatusEnabled`（既有家族，ad_domain.go:13-18，字段类型即 ADConfigStatus） |
| addomain/user_ou_service.go :239 | A | Department.Status（同名部门匹配） | `models.DeptStatusNormal` |
| addomain/user_ou_service.go :257 | A | Department.Status（自动建部门字面量，字段 DeptStatus 类型直赋） | `models.DeptStatusNormal` |
| notice_service.go :67 | A | **Notice.Status**（启停字段，int 字段 + typed 常量 → int() 转换） | `int(models.NoticeStatusNormal)` |
| notice_service.go :122-124 | **E** | **publish_status**（发布态字段，非 status！）CASE WHEN 1/0/2 | `PublishStatusPublished` / `PublishStatusDraft` / `PublishStatusScheduled` |
| notice_query_service.go :40 | **A+E 双字段** | 见下方逐处判定记录 | `PublishStatusPublished` + `NoticeStatusNormal` |
| notice_read_service.go :40 | **A+E 双字段** | 同上 | `PublishStatusPublished` + `NoticeStatusNormal` |
| knowledge_service.go :46-47 | **E 反转** | KnowledgeArticle.Status：**0=草稿 1=已发布**（1 是正向值） | `KnowledgeArticleStatusDraft` / `KnowledgeArticleStatusPublished` |
| scheduler/job_service.go :145 | A | Job.Status（创建默认，字段 JobStatus 类型直赋；语义=正常/暂停） | `models.JobStatusNormal` |
| scheduler/job_service.go :212 | A | Job.Status（更新调度器条件） | `models.JobStatusNormal` |
| scheduler/job_service.go :365 | **C** | JobLog.Status（手动执行成功日志，int 字段） | `int(models.JobLogStatusSuccess)` |
| scheduler/job_log_service.go :147-148 | **C** | JobLog.Status CASE WHEN 0/1（成败，非启停） | `JobLogStatusSuccess` / `JobLogStatusFailure` |
| rpa/credential_service.go :91 | A | RPACredential.Status（创建默认，int 字段+无类型常量直赋） | `models.RPACredentialStatusNormal`（本批新增） |
| rpa/credential_service.go :227/:234 | A | RPACredential.Status（执行用有效凭证过滤） | `models.RPACredentialStatusNormal` |
| models/ad_service_account.go :46/:49/:59/:64 | **D 三态** | ADServiceAccount.Status 裸比较 | `ADAccountStatusDisabled` / `ADAccountStatusBreaker` |
| models/ad_service_account.go StatusText switch | **D 三态** | case 0/1/2 → 常量 case | `ADAccountStatusAvailable/Disabled/Breaker` |
| addomain/account_pool.go :18-23 | **D 三态** | 本地 AccountStatus\* 三常量 → models 别名 | `= models.ADAccountStatus{Available,Disabled,Breaker}` |

**E 簇零误替源码断言（acceptance 实证）:** knowledge_service.go 的 diff 中 `Disabled|Stopped` 出现 0 次；notice 三文件 diff 中 `Enabled|Stop|Disabled` 出现 0 次——已发布语义全部走 Published 常量。

## D 簇账号池三态判定（本批最高风险点之一）

- 值语义依据：ad_service_account.go struct 注释状态机（0=可用 / 1=管理员手动停用 / 2=熔断中）+ CLAUDE.md Phase 36 AccountPool 记载 + 既有裸比较行内注释（:46 `// 停用`、:49 `// 熔断中且未到期`）三方一致。
- 新常量：`ADAccountStatusAvailable=0 / ADAccountStatusDisabled=1 / ADAccountStatusBreaker=2`（无类型 int——Status 字段为 int，且 CountByStatus 的 int64 switch case 免转换）。
- **account_pool.go 实况与 plan 描述的差异:** plan 称"account_pool.go 3 处"裸字面量，实际该文件早已通过本地常量 `AccountStatus*` 全覆盖（故不在守护基线中）；真实缺口在 models/ad_service_account.go（守护脚本 scope 之外）。处置：account_pool.go 三常量改为 models 别名（值唯一真相源上移，外部消费方 internal/api/v1/system/ad_account_handler.go:115 与 account_pool_test.go 零改动零行为变化），models 文件内 4 处裸比较 + 3 处 switch case 替换为常量。

## notice_service.go 逐处字段语义判定记录（status=启停 vs publish_status=发布态）

| 位置 | 字段 | 判定依据 | 结论 |
|------|------|----------|------|
| notice_service.go:67 | `Status: 0` | Notice 实体两字段并存：Status（`// 0=正常 1=关闭`，启停）与 PublishStatus（草稿/已发布/定时/撤回）；创建默认草稿逻辑在同函数下方操作 PublishStatus，字段名区分明确 | 启停字段 → `NoticeStatusNormal`（int() 转换） |
| notice_service.go:122-124 | `publish_status = 1/0/2` | 字段名即发布态；AS 别名 published/draft/scheduled 与 PublishStatus 四值族前三个一一对应 | 发布态 → `PublishStatus{Published,Draft,Scheduled}` |
| notice_query_service.go:40 | `publish_status = ? AND status = 0` | 同一 Where 双字段：publish_status 已参数化用 PublishStatusPublished（已发布可见性），status 为 Notice 启停（正常状态才可见），注释"已发布 + 正常状态"双重过滤实证 | status → `NoticeStatusNormal`（第二参数化） |
| notice_read_service.go:40 | 同上 | MarkAllNoticesRead 可见通知圈定，语义与 query 完全同构 | 同上 |

## 批内补缺常量清单（锁值登记闭环证据）

| 家族 | 常量 | 值 | 落点文件 | watched 登记 | expectedStatusValues 登记 |
|------|------|----|----------|--------------|--------------------------|
| ADAccountStatus（D 簇三态） | ADAccountStatusAvailable | 0 | internal/models/ad_service_account.go | `"ADAccountStatus"` 加入 watchedStatusPrefixes（batch 3 注释组） | ✅ 含中文语义注释 |
| | ADAccountStatusDisabled | 1 | 同上 | 同上 | ✅ |
| | ADAccountStatusBreaker | 2 | 同上 | 同上 | ✅ |
| RPACredentialStatus（簇 A 凭证启停） | RPACredentialStatusNormal | 0 | internal/models/rpa.go（**非** models/rpa/ 子目录——AST 扫描范围外；对齐 credentials.go:34 `check:status IN (0,1)`） | `"RPACredentialStatus"` 加入 watchedStatusPrefixes | ✅ |
| | RPACredentialStatusStopped | 1 | 同上 | 同上 | ✅ |

**闭环证据:** `go test ./internal/models/ -run TestStatusConstants` 全绿——TestStatusConstantsStability 为双向断言（watched 前缀下未登记值的常量会以 "unexpected watched status constant" fail；登记了值但常量缺失会以 "missing" fail），通过即证明前缀与值两侧登记齐全。登记总数 80 → **85**（awk 实测 85）。新常量全部位于 internal/models/ad_service_account.go 与 internal/models/rpa.go（未放 models/rpa/ 子目录、未放 services 包）。

## 「保持字面量待人工复核」清单

**无。** 全部 30 处白名单命中均完成语义确认后替换，无保守跳过项。

两处需要知会而非跳过的判断（已在台账标注）:
1. vm_service_impl.go 的 2 处 VDIResourceGroup 命中复用 `VDIServerStatus*`——vdi.go:81 字段注释实证同 0=正常/1=停用语义；plan 限定批内只允许 ADAccountStatus\*/RPACredentialStatus\* 两个补缺家族，故未为 VDIResourceGroup 单开家族。若后续希望实体各自持族，可在批 4/69-05 提案 VDIResourceGroupStatus 并回改 2 处引用。
2. user_ou_service.go:202 的 ADConfig 命中使用既有 `ADConfigStatusEnabled` 家族（ad_domain.go 既有，非本批新增，未登记锁值 watched 集合——既有家族的登记缺口属 69-05 终态门范围）。

## 白名单快照（批 3 删除后剩余 17 文件 —— 批 4 工作面）

```
internal/api/v1/monitor/oper_log_handler.go=1
internal/api/v1/scheduler/job_handler.go=1
internal/api/v1/system/notice_handler.go=2
internal/services/api_endpoint_service.go=1
internal/services/api_sender_service.go=1
internal/services/asset/fix_suggestion_monitor.go=1
internal/services/command_dispatch_service.go=4
internal/services/config_execution_service.go=8
internal/services/device_discovery_service.go=8
internal/services/duty_pool_service.go=4
internal/services/email_sender_service.go=1
internal/services/monitor/server_service.go=2
internal/services/notification_config_service.go=1
internal/services/oper_log_service.go=1
internal/services/operations/geocoding_service.go=1   # F 簇：百度 API 返回码契约，永久豁免
internal/services/workorder/assignment.go=1
internal/services/workorder/base.go=8
```

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - 现状偏差] account_pool.go 无裸字面量，真实缺口在 models/ad_service_account.go**
- **Found during:** T1 action 3 执行时
- **Issue:** plan 称 account_pool.go 有 3 处裸 status 字面量；实际该文件已有本地常量 AccountStatus\* 全覆盖（不在守护基线），models 层反而有 7 处裸值（4 比较 + 3 case，守护 scope 外）
- **Fix:** 按 plan action 3 的意图（ADAccountStatus\* 落 models + 同族裸比较一并替换）执行：account_pool.go 本地常量改 alias，models 文件裸值全替换；外部消费方与测试零改动
- **Files modified:** internal/models/ad_service_account.go, internal/services/addomain/account_pool.go
- **Commit:** 8620d9b

**2. [Rule 3 - 清单口径] 实际文件数 10 而非 plan files 列表的 7**
- **Found during:** T1 action 1 baseline 比对
- **Issue:** plan files_modified 未列 notice_query_service.go / notice_read_service.go / addomain/dept_sync_service.go（均在六模块命中清单内）
- **Fix:** 按 plan「以脚本实际输出为准，多不漏」授权纳入替换；vm_service_impl plan 称 7 处实为 6 处（基线为准）
- **Commit:** 8620d9b

## 验证记录（全部通过）

- `go build ./...` → exit 0
- `go test ./internal/models/ -run TestStatusConstants -v` → TestStatusConstantsStability PASS + TestStatusConstantsCriticalFamilies 13 family 全 PASS（85 常量）
- `go test ./internal/services/vdi/ ./internal/services/addomain/ ./internal/services/rpa/ ./internal/services/scheduler/` → 4 包全 ok（含 account_pool_test.go 的状态机测试——alias 后值语义零变化的直接证据）
- `go test ./internal/services/`（根包，覆盖 notice/knowledge_service）→ ok 376.834s
- `bash scripts/check-status-literals.sh` → exit 0，白名单无六目录条目
- E 簇源码断言：knowledge diff `Disabled|Stopped` = 0 处；notice diff `Enabled|Stop|Disabled` = 0 处
- post-commit：无文件删除、无新增未跟踪文件；工作区遗留改动（Phase 70 settings/default_theme 13 文件 + .planning 草图）全程未触碰

## Self-Check: PASSED

- FOUND: .planning/phases/69-dict-and-status-governance/69-04-SUMMARY.md
- FOUND: commit 8620d9b（git log）
- FOUND: ADAccountStatus* 家族落位 internal/models/ad_service_account.go
- FOUND: RPACredentialStatus* 家族落位 internal/models/rpa.go
