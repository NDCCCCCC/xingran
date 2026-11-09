---
phase: 144
plan: 01
wave: 1
depends_on: []
files_modified:
  - internal/services/addomain/ad_sync_drift_alert_service.go
  - internal/services/addomain/ad_sync_drift_alert_query.go (new)
  - internal/services/addomain/ad_sync_drift_alert_query_test.go (new)
  - internal/services/addomain/ad_sync_drift_alert_service_test.go
  - internal/scheduler/ad_sync_tasks.go
  - internal/scheduler/cron.go
  - internal/websocket/notice_hub.go
  - internal/websocket/notice_hub_broadcast_80_05_test.go
  - internal/api/v1/system/ad_drift_alert_handler.go (new)
  - internal/api/v1/system/ad_drift_alert_router.go (new)
  - internal/api/v1/system/ad_drift_alert_handler_test.go (new)
  - internal/api/router.go
  - internal/api/v1/system/notification_config_handler.go
  - internal/core/db/migrations/migration_238_ad_drift_alert.go (new)
  - internal/core/db/migrations/migration_238_ad_drift_alert_test.go (new)
  - internal/core/db/database.go
  - pkg/permission/config.go
  - pkg/permission/config_test.go
  - scripts/check-permission-codes.mjs 只读核对（不改）
autonomous: true
commit: false   # Lead 统一提交，禁止本 executor git add/commit
---

# Phase 144 · 同步漂移告警通道收口 — 后端（ALRT-01/02/03）

> 本 PLAN 只做**后端 + 权限/迁移**。前端页面由同目录 `PLAN-02-frontend.md` 的独立 executor 并行负责，两者文件面互斥（本 PLAN 不碰 `xingran-react-frontend/**`）。

## 必读（按顺序）
1. `.planning/notes/261108-v154-architecture-decisions.md` **§二**（Lead 已拍板的全部设计裁决，权威）
2. `.planning/ROADMAP.md` Phase 144 段（成功标准 5 条）
3. `.planning/REQUIREMENTS.md` ALRT-01/02/03
4. `AGENTS.md` / `CLAUDE.md` 项目硬约定（尤其 operlog / 双方言 / migration 注册）

## 目标
补齐 v1.53 FIX-11 自报边界：后端查询/配置端点 + WS 广播 + webhook 通道 + 菜单/权限/seed，并**一并修复** `emit()` 绕过发布链路导致铃铛收不到告警的实质缺口。

## 已确认事实（不要重新调研）
- FIX-11 后端服务已存在：`internal/services/addomain/ad_sync_drift_alert_service.go`（418 行）：5 个 `sys_config` 热重载旋钮（L52-67：`ad_sync.drift_alert.enabled` / `.tolerance` / `.throttle_minutes` / `.notice_enabled` / `.baseline`）、`loadConfig`(L240-281)、`detectADDrift`(L112-134)、`CheckAndAlert`(L170-217，**软失败：任何错误仅 WARN、返回 nil、绝不阻断 cron** L169)、`shouldAlert`(L336-348)、`emit()`(L361-411：applogger / `sys_oper_log` direct INSERT `BusinessType=14` / `sys_notice` **L405 直接 `s.db.Create(notice)` 绕过发布链路**)、L141 明确「无 WS / 无 NoticeService 依赖」。
- cron：`internal/scheduler/ad_sync_tasks.go:226-241`（`"0 12 * * * *"`，`NewADSyncDriftAlertService(s.db)`，注册失败非阻断）。
- **实质缺口**：`emit()` 绕过 `NoticeService.CreateNoticeWithTargets + PublishNotice`，导致 `cron.go:715` 的 `new_notice` 铃铛推送不触发。
- WS：`internal/websocket/notice_hub.go` 有 `NoticeMessage`(L49-58: Type/NoticeID/Title/Content/Priority/Timestamp/UnreadCount)、`BroadcastToUsers`(L190)、`BroadcastToAll`(L233)；常量区 L13-18。`scheduler.NoticeHub` 接口（`cron.go:507-509`）与 `noticeHubAdapter`（`router.go:38-47`）**只有 `BroadcastToUsers`，无 `BroadcastToAll`**；`scheduler.GetNoticeHub()`（`cron.go:527`）。注入链 `router.go:85-95` `setupNoticeHub`，WS 端点 `/api/v1/system/ws/notices`（`router.go:477-482`）。
- 广播先例（含 `defer recover()` 兜底，hub 关闭写 channel 会 panic）：`internal/services/asset/fix_suggestion_monitor.go:138-153`（+ `156-173` 的 `NoticeService.CreateNoticeWithTargets` + `PublishNotice`）、`internal/services/asset/reconciliation_workorder.go:333-381`（业务 payload JSON 内嵌 `Content` + `Type` 透传自定义事件名 + recover L366-371）、`internal/scheduler/cron.go:715-740` `broadcastNoticeToUsers`。
- 通用 Webhook 能力：`internal/models/notification_config.go`（`sys_api_notification_config` = `APINotificationConfig` L91、`configType=webhook` L52、`apiUrl`/`headers`/`templateBody`/`authConfig`/`retryCount`/`timeout`/`status`）；`internal/services/api_sender_service.go`（`Send` L60-107 重试+200ms×i 退避+30s 超时+2xx、`SendWithDefaultConfig` L110-139、`SendWebhook` L383、`APIMessage{Title,Content,Data}`）；`notification_config_handler.go`（已用 `operlog.RecordWithBody` L165/203/398/436，**但 `GetAPINotificationConfig` L357-371 / `ListAPINotificationConfigs` L294-343 未遮 `authConfig`/`headers` 中的 token —— 复用前必须补**）。
- operlog 脱敏：`internal/utils/operlog/operlog.go:125-170` `sensitiveKeys`（按 JSON key 名子串匹配，**URL 值里的 `?key=`/`access_token=` 不会被遮**）；`RecordWithBody` L331-364；`regression_test.go` 的 `mandatorySensitiveKeywords` 与 25 常量值有回归守护。
- `configs/config.yaml` / `internal/config/config.go` **无** webhook/alert 段；5 旋钮走 `sys_config` 每轮热重载；`134-SUMMARY.md:32` 记「sys_config 旋钮无 seed」。
- 权限三方一致：`src/constants/permissionCodes.ts` / `pkg/permission/config.go` / `sys_menu.perms`，`scripts/check-permission-codes.mjs` 双向校验会 exit 1；漏 F 型菜单 seed 则非 admin 恒 403（`migration_237:13-15` 同型坑）。菜单 seed 范例 `migration_221_manager_sync_menus.go`（C 型菜单 L79-102、`ensureManagerSyncMenu` L130-165 幂等 count-then-insert、F 按钮 L168-197、`grantMenuToParentRoles` L200-212、`canonicalSystemRootID` 挂「系统管理」根 L63-77）；权限 seed 范例 `migration_237_workorder_rating_perm_seed.go`（`F_TYPE_SEED_BEGIN/END` + `actionPermissionSeed` + `resolveActionPermissionParents`/`ensureActionPermissionMenu`/`grantActionPermissionToParentHolders`）。
- **migration 注册**：活跃最大编号 **237**；234-237 走 `internal/core/db/database.go` 的 `runAndRecord(...)` **两处调用点**（PG 分支约 `:1215` 之后、sqlite 分支约 `:1352` 之后）。新 238 必须：① 新建 `migration_238_ad_drift_alert.go` 导出 `Migrate238ADDriftAlert(db *gorm.DB) error` ② **在 database.go 两处调用点各加一行** `runAndRecord(238, "ad_drift_alert_menu_perm_seed", migrations.Migrate238ADDriftAlert)` ③ `registry.go` 的 `ActiveMigrations` 是 175-233 的历史清单（234+ 不在其中），**按现状不要往 registry 加 238**。
- 用户提及的 `sys_notice_config` **不存在**，实际是 `sys_email_config` / `sys_api_notification_config` / `sys_notification_channel`。
- `pkg/permission/config.go` 已登记 `/trunk`、`/portsec`，**未登记 `/network/ports/write/physical`**（Phase 143 移交项，见下）。

## Lead 裁决（必须遵守）
| 项 | 裁决 |
|---|---|
| WS 注入 | **方案 A**：扩 `scheduler.NoticeHub` 接口加 `BroadcastToAll(message interface{})` + 同步扩 `noticeHubAdapter`（`router.go:38-47`）；`emit()` 经 `scheduler.GetNoticeHub()` 调用。**不用**方案 B（不从 core 注入 `*websocket.NoticeHub`） |
| 广播兜底 | 广播处必须 `defer recover()`；只在 `shouldAlert==true` 的 `emit()` 内触发（避免每轮刷屏）；失败仅 `applogger.Warnf`，保持软失败契约（任何错误不阻断 cron） |
| 铃铛收不到 | **一并修**：`emit()` 通道 3 改走 `services.NewNoticeService(db).CreateNoticeWithTargets(...) + PublishNotice(...)`（照 `fix_suggestion_monitor.go:156-173`），让 `new_notice` 与新的 `ad_sync_drift_alert` 双事件都能推 |
| 消息格式 | 复用 `websocket.NoticeMessage`：`Type: "ad_sync_drift_alert"`，`Content` 内嵌业务 payload JSON（memberCount/distinctGroups/baseline/deltas/noticeId/tolerance/throttled），`Priority` 用紧急；在 `notice_hub.go:13-18` 加常量 `MessageTypeADSyncDriftAlert = "ad_sync_drift_alert"` |
| 端点 | `POST /system/ad-drift-alert/{status,logs,config,config/update,trigger}`；`status/logs/config` 读权限，`config/update`+`trigger` 走 `RequireStrictPermissions`；全部 `response.Success()/Error()`、`c.Request.Context()` 透传；分页用 `list/total/current/pageSize`（current 从 1） |
| 配置 | 统一走 `sys_config`，新增 `ad_sync.drift_alert.ws_enabled`(默认 true) / `webhook_enabled`(默认 false) / `webhook_config_id`(空=默认)；**不新增** `configs/config.yaml` 段、不加 `internal/config` struct 字段（避免双源冲突） |
| webhook | **路线 A**：复用 `sys_api_notification_config`（`configType=webhook`）+ `APISenderService.SendWithDefaultConfig(ctx, models.APIConfigTypeWebhook, msg)` 或按 configID `Send(...)`；`emit()` 构造 `APIMessage{Title,Content,Data}`，失败仅 WARN、不返回 error、不阻断 cron；触发结果写 `sys_oper_log`（`BusinessType=14`、Method `ADSyncDriftAlertService.Webhook`、JsonResult 带 httpCode/success/retryCount、URL 脱敏）。**不做**企微/钉钉专用 markdown + 加签（路线 B 列后续） |
| 脱敏 | ① 所有告警配置写端点用 `operlog.RecordWithBody`；② **必须补** `notification_config_handler.go` 的 `GetAPINotificationConfig`/`ListAPINotificationConfigs` 对 `authConfig`/`headers` 中 token 的主动遮蔽（照该文件 `:98-104` 对 Password 的处理）；③ 若需给 `sensitiveKeys` 加 `webhookKey`/`webhookUrl`/`access_token` 字面量，**必须同步** `internal/utils/operlog/regression_test.go` 的 `mandatorySensitiveKeywords`；④ 响应体不得回明文 webhook secret |
| 菜单/权限 | 新 `migration_238_ad_drift_alert.go`：C 型页面菜单 + F 型操作码 seed（幂等 count-then-insert + `grantActionPermissionToParentHolders` 精准授权，参照 221/237）+ `sys_config` 默认 seed（5 既有 + 4 新增 = 9 行，幂等 INSERT）；挂「系统管理」根 |
| 权限码 | `system:adDriftAlert`（C 型）+ `:list` / `:config` / `:edit` / `:trigger`（F 型）；`pkg/permission/config.go` 增常量；`src/constants/permissionCodes.ts` 由**前端 PLAN** 增对应组（两边逐字一致，命名以此为准）；`scripts/check-permission-codes.mjs` 校验通过 |
| 移交项（Phase 143 → 本 PLAN） | 在 `pkg/permission/config.go` 的 route-permission registry 补登 `POST /network/ports/write/physical`（复用 `NetworkPortWrite` 常量），与 `/trunk`、`/portsec` 同型；`pkg/permission/config_test.go` 加断言 |
| operlog module | `"同步漂移告警"` / `"同步漂移告警配置"`（新 module 名，登记到 SUMMARY） |
| 软失败契约 | `CheckAndAlert`(L169) + `emit`(L360)：任何错误仅 WARN、返回 nil、绝不阻断 cron。WS / webhook 新通道必须保持 |

## 任务（XML）
<task id="1">
<title>扩 NoticeHub 接口 + 漂移告警广播 + 修发布链路</title>
<files>internal/scheduler/cron.go, internal/api/router.go, internal/websocket/notice_hub.go, internal/services/addomain/ad_sync_drift_alert_service.go</files>
<action>
① `scheduler.NoticeHub`（`cron.go:507-509`）加 `BroadcastToAll(message interface{})`；`noticeHubAdapter`（`router.go:38-47`）实现它（转调 `*websocket.NoticeHub.BroadcastToAll`）；检查其它实现该接口的地方（如 mock）同步补方法。② `notice_hub.go` 常量区加 `MessageTypeADSyncDriftAlert`。③ `emit()`：把通道 3 的 `s.db.Create(notice)` 改为 `services.NewNoticeService(s.db).CreateNoticeWithTargets(...) + PublishNotice(...)`（照 `fix_suggestion_monitor.go:156-173`，保持 sys_notice 行内容与 Title="AD 同步漂移告警" / PriorityUrgent / TargetAll 语义）；新增 WS 通道：构造 `websocket.NoticeMessage{Type: MessageTypeADSyncDriftAlert, ...}`，经 `scheduler.GetNoticeHub().BroadcastToAll(msg)` 发送，用 `defer func(){ if r:=recover(); r!=nil { applogger.Warnf(...) } }()` 包裹；只在 `shouldAlert==true` 分支内触发；受 `ws_enabled` 旋钮控制；失败仅 WARN。④ 新增 webhook 通道（任务 2）。**保持 `CheckAndAlert` / `emit` 的软失败契约：任何新通道出错都不返回 error、不 panic、不阻断 cron。**
</action>
</task>

<task id="2">
<title>webhook 通道 + 配置旋钮 + 脱敏补齐</title>
<files>internal/services/addomain/ad_sync_drift_alert_service.go, internal/api/v1/system/notification_config_handler.go, internal/utils/operlog/operlog.go (仅当确需扩关键词), internal/utils/operlog/regression_test.go (仅当扩关键词)</files>
<action>
① `loadConfig` 读 3 个新旋钮（缺省即默认值，不依赖 seed 才能跑）；`emit()` 在 `webhook_enabled=true` 时用 `APISenderService.SendWithDefaultConfig(ctx, models.APIConfigTypeWebhook, msg)`（或按 `webhook_config_id` 取配置后 `Send`），构造 `APIMessage{Title:"AD 同步漂移告警", Content: 业务摘要, Data: ...}`，失败仅 WARN；触发结果写 `sys_oper_log`（Method `ADSyncDriftAlertService.Webhook`，JsonResult 含 httpCode/success/retryCount，**URL 不得明文落日志**，用脱敏函数处理）。② 补 `notification_config_handler.go` 的 `GetAPINotificationConfig` / `ListAPINotificationConfigs` 对 `authConfig`/`headers` 内 token/key/secret 的主动遮蔽（置 `"******"`，参照 `:98-104`）。③ 若给 `sensitiveKeys` 加新字面量，必须同步 `regression_test.go` 的 `mandatorySensitiveKeywords` 与相关用例，并跑 `go test ./internal/utils/operlog/` 全绿（25 常量值不可重排）。
</action>
</task>

<task id="3">
<title>查询/配置/触发端点</title>
<files>internal/services/addomain/ad_sync_drift_alert_query.go (new), internal/api/v1/system/ad_drift_alert_handler.go (new), internal/api/v1/system/ad_drift_alert_router.go (new), internal/api/router.go</files>
<action>
query 文件：`GetStatus(ctx)`（暴露 `enabled/tolerance/throttleMinutes/noticeEnabled/wsEnabled/webhookEnabled/baseline/lastAlertAt/lastAlertDrifted/alertCount`，注意 `mu` 保护）、`ListTriggerLogs(ctx, current, pageSize)`（查 `sys_oper_log WHERE method='ADSyncDriftAlertService.CheckAndAlert'` 分页 `list/total/current/pageSize`）、`GetConfig/UpdateConfig(ctx)`（读写 9 个 sys_config 旋钮）、`TriggerOnce(ctx)`（手动跑一轮 `CheckAndAlert`，写操作）。
handler/router：5 个 POST 端点，一律 `response.Success()/Error()`（禁裸 `c.JSON`）、`c.Request.Context()` 透传；读组挂 `RequirePermissions`，写组（`config/update`、`trigger`）挂 `RequireStrictPermissions`；写端点 success path 在 `response.Success` **之前** `operlog.Record`（trigger 用 `OperTypeSync`）/`RecordWithBody`（config/update 用 `OperTypeUpdate`，module `"同步漂移告警配置"`）；权限码 `system:adDriftAlert:*`。`internal/api/router.go` 在 `system/authorized` 下挂新 group（照 `notification_config_router.go` + `router.go:159-164`）。
</action>
</task>

<task id="4">
<title>migration 238 + 两处注册 + 权限常量补齐</title>
<files>internal/core/db/migrations/migration_238_ad_drift_alert.go (new), internal/core/db/database.go, pkg/permission/config.go, pkg/permission/config_test.go</files>
<action>
① 新建 `Migrate238ADDriftAlert(db *gorm.DB) error`：C 型页面菜单（component 用 `system/ad-drift-alert/index`，挂 `canonicalSystemRootID`「系统管理」根，perms=`system:adDriftAlert`）+ 4 个 F 型按钮权限（`:list`/`:config`/`:edit`/`:trigger`，用 `F_TYPE_SEED_BEGIN/END` + `actionPermissionSeed` + `grantActionPermissionToParentHolders`，参照 `migration_237`） + 9 行 `sys_config` 默认 seed（5 既有 + 4 新增，幂等 `NOT EXISTS`/count-then-insert，**双方言纯 GORM 或 `db.Exec` 两库可跑**，避免 `::text` 等 PG 专属语法）。**必须幂等**（重复执行不报错、不重复插）。
② `database.go` 的 PG 分支（约 `:1215` 之后）与 sqlite 分支（约 `:1352` 之后）**各加一行** `if err := d.runAndRecord(238, "ad_drift_alert_menu_perm_seed", migrations.Migrate238ADDriftAlert); err != nil { ... }`（照 237 的写法，注意两处都要加）。**不要**改 `registry.go` 的 `ActiveMigrations`（234+ 按现状不在其中）。
③ `pkg/permission/config.go`：新增 5 个权限常量（逐字 `system:adDriftAlert` / `system:adDriftAlert:list` / `:config` / `:edit` / `:trigger`）+ 5 条路由注册；**并补登 Phase 143 移交的 `POST /network/ports/write/physical`（复用 `NetworkPortWrite` 常量）**；`config_test.go` 加断言（含 `/physical` 一条）。
④ 运行 `node scripts/check-permission-codes.mjs` 双向校验（前端 `permissionCodes.ts` 由 PLAN-02 同步加，若此时前端还没写导致校验失败，如实记录并在 SUMMARY 说明「待前端 PLAN 完成后由 Lead 复跑」）。
</action>
</task>

<task id="5">
<title>测试（就近 *_test.go，不依赖生产/真机）</title>
<files>internal/services/addomain/ad_sync_drift_alert_service_test.go, ad_sync_drift_alert_query_test.go (new), internal/api/v1/system/ad_drift_alert_handler_test.go (new), internal/websocket/notice_hub_broadcast_80_05_test.go, internal/core/db/migrations/migration_238_ad_drift_alert_test.go (new), internal/utils/operlog/regression_test.go (仅当扩关键词)</files>
<action>
- 扩现有 `ad_sync_drift_alert_service_test.go`（11 例保持全绿）：WS notifier mock（`BroadcastToAll` 被调用次数）、`ws_enabled=false` 不广播、webhook 开关/失败软降级、`AlertCount`、节流内不重复广播、`CheckAndAlert` 出错仍返回 nil。
- 新 `ad_sync_drift_alert_query_test.go`：`GetConfig/UpdateConfig/ListTriggerLogs` 分页（sqlite 内存库 AutoMigrate，参照现有 test 的 `glebarez/sqlite` 用法）。
- 新 `ad_drift_alert_handler_test.go`：response 包装、`RecordWithBody` 脱敏（明文 webhook URL 不进 oper_param）、权限分组、trigger 写操作 operlog 位置。
- 扩 `notice_hub_broadcast_80_05_test.go`：`ad_sync_drift_alert` 的 `BroadcastToAll` 两客户端各收 1 帧（照 L179-201）。
- 新 `migration_238_ad_drift_alert_test.go`：菜单/权限/sys_config seed 幂等（连跑两次）+ 双方言（sqlite；PG 有 `TEST_PG_DSN` 才跑并显式 skip 可观测）。
- 若扩了 `sensitiveKeys`，同步 `regression_test.go` 的 `mandatorySensitiveKeywords` 与 `TestFilterSensitiveParams...` 用例。
</action>
</task>

<task id="6">
<title>自验证（只跑本 phase 相关包）</title>
<action>
`go build ./...`
`go test -count=1 ./internal/services/addomain/ ./internal/scheduler/ ./internal/api/v1/system/ ./internal/websocket/ ./internal/core/db/migrations/ ./internal/utils/operlog/ ./pkg/permission/`
`node scripts/check-permission-codes.mjs`（若因前端未同步而失败，如实记录，不硬改前端文件）
不要跑 `go test ./...` / 前端命令。失败只修本 phase 文件面问题，迭代至通过。
</action>
</task>

<task id="7">
<title>写 SUMMARY（不提交）</title>
<action>
写 `.planning/phases/144-alrt-01-02-03/144-01-backend-SUMMARY.md`：改动清单、REQ 满足情况、验证结果、**Known Gaps**（企微/钉钉专用 markdown+加签未做=路线 B、`sys_object_id` 无关、`Evidence`/未取证不涉及本 phase、`check-permission-codes.mjs` 待前端同步后复跑）、operlog 新 module 名登记、migration 238 两处注册点、以及「未 git commit」说明。
</action>
</task>

## 验证标准（must_haves）
1. `emit()` 改走 `NoticeService` 发布链路 + 新 `ad_sync_drift_alert` WS 广播；`scheduler.NoticeHub` 与 adapter 补齐 `BroadcastToAll`；有 `recover()` 兜底且失败不阻断 cron。
2. 5 个端点可用、权限分组正确、写端点 `operlog.Record`/`RecordWithBody` 在 `response.Success` 之前；webhook 配置与日志的密钥脱敏有单测断言。
3. migration 238 幂等、双方言、`database.go` **两处**注册；权限三方一致（`check-permission-codes.mjs` 通过或如实记录待前端同步）。
4. `Phase 143` 移交的 `/network/ports/write/physical` 注册表补登完成。
5. 7 个 Go 包 `go test -count=1` 全绿、`go build ./...` 全绿；软失败契约未被破坏。
6. 未触碰其它 phase 文件面；零新依赖。

## 禁止
- 禁止 `git add` / `git commit` / `git stash` / `git checkout`（Lead 统一集成提交）。
- **禁止修改 `xingran-react-frontend/**`**（PLAN-02 前端独占）。
- 禁止修改 `internal/services/portwrite/**`、`internal/services/device_discovery_service*.go`、`internal/services/network/**`、`pkg/response/**`（其它 phase 独占）。
- 禁止新增 `configs/config.yaml` / `internal/config` 字段（双源冲突）；禁止改 `registry.go` 的 `ActiveMigrations`。
- 禁止改 25 个 operlog 常量值；禁止把明文 webhook secret 写进日志/响应。
- 禁止启动长驻进程（AGENTS.md）、禁止写生产数据；`migration 238` 只在测试库执行，不连生产 PG。
