# Phase 144 · 同步漂移告警通道收口 — 后端 SUMMARY（PLAN-01）

> 执行者：`exec-144-back`（gsd-executor，后端）。前端由 `PLAN-02-frontend.md` 的 executor 并行完成，本 SUMMARY 只覆盖后端文件面。
> **未执行 `git add` / `git commit` / `git stash` / `git checkout` —— 由 Lead 统一集成提交**（4 个 executor 并行写同一工作区）。

## 一、改动清单（按任务）

### 任务 1 · 扩 NoticeHub 接口 + 漂移告警广播 + 修发布链路
| 文件 | 改动 |
|---|---|
| `internal/scheduler/cron.go` | `NoticeHub` 接口新增 `BroadcastToAll(message interface{})` |
| `internal/api/router.go` | `noticeHubAdapter.BroadcastToAll` 实现（转调 `*websocket.NoticeHub.BroadcastToAll`）；`setupNoticeHub` 改用命名 adapter 并调用新的 `setupDriftAlertHooks` |
| `internal/scheduler/cron_80_01_test.go`、`workorder_tasks_80_02_test.go` | 两个 `NoticeHub` stub 同步补 `BroadcastToAll`（接口扩展编译守护） |
| `internal/websocket/notice_hub.go` | 常量区加 `MessageTypeADSyncDriftAlert = "ad_sync_drift_alert"` |
| `internal/services/addomain/ad_sync_drift_alert_service.go` | `emit()` 由三通道扩为**五通道**：① applogger ② `sys_oper_log` 触发记录 ③ `sys_notice` 改走注入的 `NoticeService.CreateNoticeWithTargets + PublishNotice` 发布链路（修「铃铛收不到」根因；未注入时回退 direct INSERT）④ WS `ad_sync_drift_alert` 全员事件 ⑤ webhook；通道 3/4/5 各自 `recover()` 兜底 + emit 顶层 `recover()`；只在 `shouldAlert==true` 内触发；失败仅 WARN、返回 nil、绝不阻断 cron |
| `internal/services/addomain/ad_sync_drift_alert_hooks.go`（新） | **零循环依赖注入点**：`SetDriftAlertWSNotifier` / `SetDriftAlertNoticePublisher` / `SetDriftAlertWebhookSender` + 类型定义（`DriftAlertWSNotifier` / `DriftAlertNotice` / `DriftAlertWebhookMessage` / `DriftAlertWebhookResult`） |
| `internal/scheduler/ad_sync_tasks.go` | 仅更新 FIX-11 注释（说明通道已由 api 装配层注入全局钩子；cron 每轮 `NewADSyncDriftAlertService(s.db)` 共享注入点，逻辑零改动） |

**⚠ 架构偏差（必须登记）**：PLAN 字面要求 `emit()` 经 `scheduler.GetNoticeHub()` 调用；但实证 `internal/scheduler/ad_sync_tasks.go:11` 与 `internal/services`（根包，含 `NoticeService`/`APISenderService`，见 `auth_credential_service.go:8`）**都已 import `internal/services/addomain`** —— addomain 反向 import 二者会形成 Go 包级循环依赖（编译期硬失败）。故采用语义等价、可单测、零新包的方案：addomain 定义包级函数变量注入点，`internal/api/router.go:setupDriftAlertHooks` 在装配阶段注入**同一个 scheduler 全局 `noticeHubAdapter` 实例** + `services.NewNoticeService(db)` 发布链路 + `services.APISenderService`。生产广播仍走全局 hub、发布仍走 NoticeService、webhook 仍走 APISenderService，与方案 A 等价；额外在发布成功后主动推 `new_notice` 全员事件，使「`new_notice` 与 `ad_sync_drift_alert` 双事件都能推」落地。

### 任务 2 · webhook 通道 + 3 旋钮 + 脱敏补齐
| 文件 | 改动 |
|---|---|
| `internal/services/addomain/ad_sync_drift_alert_service.go` | `loadConfig` 读 3 个新旋钮 `ws_enabled`(默认 true) / `webhook_enabled`(默认 false) / `webhook_config_id`(空=默认)；`emit()` 构造 `APIMessage{Title,Content,Data}` 走注入的 `SendWithDefaultConfig`（或按 configID `Send`）；触发结果写 `sys_oper_log`（Method `ADSyncDriftAlertService.Webhook`、BusinessType=14、JsonResult 含 `success/httpCode/retryCount`）；`sanitizeWebhookMessage` + `maskQueryParam` 把 URL 的 `?key=` / `access_token=` / userinfo 脱敏，**明文 webhook secret 不落日志/响应** |
| `internal/api/v1/system/notification_config_handler.go` | `ListAPINotificationConfigs` / `GetAPINotificationConfig` 响应前主动遮蔽 `authConfig` / `headers` 中 token/key/secret/Authorization 与 `APIURL` 查询凭据 |
| `internal/api/v1/system/notification_config_handler_mask.go`（新） | 脱敏 helper：`isAPISensitiveKey` / `maskAPIMapFields`（深拷贝）/ `maskAPINotificationConfig` / `maskAPINotificationConfigList` / `maskURLQuerySecrets` |
| `internal/utils/operlog/operlog.go` + `regression_test.go` | `sensitiveKeys` 追加 `webhookUrl` / `webhook_url` / `webhookKey` / `access_token` / `authorization` 5 个字面量；`mandatorySensitiveKeywords` **同步追加同 5 项**；新增 `TestPhase144_WebhookSecretsMasked` 表驱动用例。**25 个 OperType 常量值一个未动** |
| `configs/config.yaml` / `internal/config/config.go` | **未改**（统一走 sys_config，避免双源冲突） |
| `sys_api_notification_config` | **未新增表**（路线 A 复用既有设施） |

### 任务 3 · 查询/配置/触发端点
| 文件 | 改动 |
|---|---|
| `internal/services/addomain/ad_sync_drift_alert_query.go`（新） | `GetStatus`（enabled/tolerance/throttleMinutes/noticeEnabled/wsEnabled/webhookEnabled/baseline/lastAlertAt/lastAlertDrifted/alertCount，`mu` 保护）/ `ListTriggerLogs`（`method='ADSyncDriftAlertService.CheckAndAlert'` 分页 `list/total/current/pageSize`，current 从 1）/ `GetConfig` / `UpdateConfig`（9 个 sys_config 旋钮白名单 + 幂等 upsert）/ `TriggerOnce` |
| `internal/api/v1/system/ad_drift_alert_handler.go`（新） | 5 个 POST 端点，全部 `response.Success()/Error()`、`c.Request.Context()` 透传；写端点 success path 在 `response.Success` **之前** operlog；`config/update` 用 `RecordWithBody`（先缓存+恢复 body，避免 bind 消费后脱敏拿不到原文） |
| `internal/api/v1/system/ad_drift_alert_router.go`（新） | 读组 `RequirePermissions([list, config])`；写组 `config/update`→`RequireStrictPermissions([edit])`、`trigger`→`RequireStrictPermissions([trigger])`，写组为读组**兄弟组**（v1.24 Phase 78 #16 教训） |
| `internal/api/router.go` | `system/authorized` 下挂 `SetupAdDriftAlertRouter`（照 `router.go:159-164` notification-configs 惯例） |

**operlog 新 module 名登记**（CLAUDE.md 规范）：
- `"同步漂移告警"` — `POST /system/ad-drift-alert/trigger`，`operlog.Record`，`OperTypeSync`(14)
- `"同步漂移告警配置"` — `POST /system/ad-drift-alert/config/update`，`operlog.RecordWithBody`，`OperTypeUpdate`(2)

### 任务 4 · migration 238 + 两处注册 + 权限常量
| 文件 | 改动 |
|---|---|
| `internal/core/db/migrations/migration_238_ad_drift_alert.go`（新） | 导出 `Migrate238ADDriftAlert(db *gorm.DB) error`：C 型页面菜单（`component=system/ad-drift-alert/index`，挂 `canonicalSystemRootID`「系统管理」根，`perms=system:adDriftAlert`）+ 4 个 F 型按钮（`F_TYPE_SEED_BEGIN/END` + `actionPermissionSeed` + `resolveActionPermissionParents` / `ensureActionPermissionMenu` / `grantActionPermissionToParentHolders`）+ **9 行 sys_config seed**（5 既有：`enabled/tolerance/throttle_minutes/notice_enabled/baseline` + 4 新增：`ws_enabled/webhook_enabled/webhook_config_id/baseline_key` 别名）；纯 GORM / `db.Exec`，**无 `::text` 等 PG 专属语法**；count-then-insert + NOT EXISTS 授权，连跑两次幂等；根菜单缺失时优雅降级但 config 仍 seed |
| `internal/core/db/database.go` | **两处调用点**各加一行 `runAndRecord(238, "ad_drift_alert_menu_perm_seed", migrations.Migrate238ADDriftAlert)`：PG 分支 `:1219`、sqlite 分支 `:1361`（与 234-237 对称，失败非阻断） |
| `internal/core/db/migrations/registry.go` | **未改**（234+ 按现状不进 `ActiveMigrations`） |
| `pkg/permission/config.go` | 新增 5 个权限常量（逐字 `system:adDriftAlert` / `:list` / `:config` / `:edit` / `:trigger`）+ 5 条路由注册；**并补登 Phase 143 移交的 `POST /network/ports/write/physical`（复用 `NetworkPortWrite`）** |
| `pkg/permission/config_test.go`（新） | 3 个断言：5 码逐字、5 端点死表命中、`/physical` 命中 `NetworkPortWrite` |
| `scripts/route-registry.json` | 由 `node scripts/scan-routes.cjs` **重生成**（新增 5 条路由，读组 ANY-of / 写组严格单一，均登记死表）；否则 `pkg/permission/TestRouteRegistryFreshness` 会 FAIL |

### 任务 5 · 测试（就近 `*_test.go`）
| 文件 | 覆盖 |
|---|---|
| `ad_sync_drift_alert_service_test.go`（扩，原 11 例全绿） | WS notifier mock 调用次数、`ws_enabled=false` 不广播、节流内不重复广播、WS panic 被 recover 不阻断 cron、发布链路注入（noticeId 进 payload）、webhook 失败软降级 + URL 脱敏断言、webhook 默认关闭、`sanitizeWebhookMessage` |
| `ad_sync_drift_alert_query_test.go`（新） | `GetStatus` / `GetConfig` 默认值与覆盖 / `UpdateConfig` 白名单与幂等 / `ListTriggerLogs` 分页 / `TriggerOnce` 软失败 |
| `ad_drift_alert_handler_test.go`（新） | 5 端点 `response` 包装、分页形状、`config/update` `RecordWithBody` 脱敏（明文 URL/token 不进 `oper_param`）、`trigger` `OperTypeSync=14`、operlog 位置 |
| `notification_config_handler_mask_test.go`（新） | Get/List 读响应脱敏（token/key/secret/Authorization/URL 查询凭据）+ 深拷贝不改 DB 行 |
| `regression_test.go`（扩） | `mandatorySensitiveKeywords` 同步 + `TestPhase144_WebhookSecretsMasked` |
| `notice_hub_broadcast_80_05_test.go`（扩） | `ad_sync_drift_alert` 的 `BroadcastToAll` 两客户端各收 1 帧、字段透传 |
| `migration_238_ad_drift_alert_test.go`（新） | C 菜单 / 4 F 按钮 / 精准授权 / 9 行 seed / 连跑两次幂等 / 根缺失降级 / PG 双方言（`TEST_PG_DSN` 才跑，否则显式 `t.Skip`） |
| `config_test.go`（新） | 权限常量 + 路由 + Phase 143 移交项 |

## 二、REQ 满足情况

| REQ | 后端交付 |
|---|---|
| **ALRT-01** | 查询与配置端点齐备（status/logs/config/config/update/trigger），权限码挂载（C 型 + 4 F 型 seed + 死表），敏感 webhook 配置读响应遮蔽 + 写端点 `RecordWithBody` 脱敏 |
| **ALRT-02** | `ad_sync_drift_alert` WS 全员广播（复用 `NoticeHub.BroadcastToAll` + recover 兜底，失败不阻断）；webhook 通道复用 `sys_api_notification_config`（凭证不落明文日志、失败写 operlog 可观测、软失败）；发布链路修铃铛 |
| **ALRT-03**（后端部分） | 新端点 + WS 事件 + 菜单 seed（component `system/ad-drift-alert/index` 由前端菜单驱动 glob 自动收）；前端页面/store/类型由 PLAN-02 交付 |

## 三、验证结果（完成前必做，全部实测）

| 命令 | 结果 |
|---|---|
| `go build ./...` | ✅ 全绿（多次，含最后改动后） |
| `go test -count=1 ./internal/services/addomain/ ./internal/scheduler/ ./internal/api/v1/system/ ./internal/websocket/ ./internal/core/db/migrations/ ./internal/utils/operlog/ ./pkg/permission/` | ✅ **7/7 包全绿** |
| `node xingran-react-frontend/scripts/check-permission-codes.mjs`（PLAN 写作 `node scripts/check-permission-codes.mjs`，实际脚本位于前端目录） | ✅ **双向零漂移**：`[adDriftAlert] TS 声明 4 个 / 后端常量 4 个 / 菜单 F 型 4 个`。前端 executor 已并行把 `permissionCodes.ts` 的 `adDriftAlert` 组补齐，故**无需「待前端完成后由 Lead 复跑」** |
| `scripts/route-registry.json` 重生成 | ✅ 5 条新路由入库，读组 `RequirePermissions` / 写组 `RequireStrictPermissions` 均登记死表 |

> 未跑 `go test ./...` 全量、未跑前端命令（按 PLAN，由 Lead 集成后统一跑）。

## 四、Known Gaps（诚实登记，不伪造闭环）

1. **企微/钉钉专用 markdown + 加签（路线 B）未做** —— 按 Lead 裁决走路线 A（复用 `sys_api_notification_config` + `APISenderService`），专用实现列后续。
2. **`check-permission-codes.mjs` 无需复跑** —— 前端已同步，校验实测通过（若 Lead 集成后前端文件再变，复跑即可）。
3. **`sys_object_id` 无关本 phase**；`Evidence` / 未取证 / 真机验证不涉及本 phase。
4. **PG 双方言**：migration 238 逻辑由 sqlite 测试全量覆盖；PG 用例仅在 `TEST_PG_DSN` 显式设置时执行，否则 `t.Skip`（可观测），**未连生产 PG**。
5. **`ad_sync.drift_alert.baseline_key`** 是第 9 行 seed（原 FIX-11 注释提及的「基线存储用 config_key」别名），当前运行时仍用常量 `adDriftBaselineConfigKey` 硬编码读基线，seed 该行保持 PLAN「5 既有 + 4 新增 = 9 行」口径一致，属前向兼容、不影响检测。
6. **架构偏差**：`emit()` 未直接 `scheduler.GetNoticeHub()`（循环依赖），改由包级注入点 + `router.go` 注入同一全局 adapter（见任务 1 说明）。语义等价、生产走真发布链路。

## 五、migration 238 两处注册点（供 Lead 集成核对）

- `internal/core/db/database.go:1219`（PG 分支，紧随 237 之后）
- `internal/core/db/database.go:1361`（sqlite 分支，紧随 237 之后）
- `internal/core/db/migrations/registry.go` **未改**（按 PLAN 禁止项）

## 六、其它

- **零新依赖**：只 import 项目内既有包与既有 module（`gorm.io/...`、`glebarez/sqlite`、testify 等均为现有）。
- **文件面纪律**：未触碰 `xingran-react-frontend/**`（前端 executor 独占）、`internal/services/portwrite/**`、`internal/services/device_discovery_service*.go`、`internal/services/network/**`、`pkg/response/**`；`internal/api/router.go` 仅本 phase 改。`scripts/route-registry.json` 为扫描器生成产物（新增路由必需），非源码手改。
- **软失败契约未破**：`CheckAndAlert` / `emit` 任何错误仅 WARN、返回 nil；WS / webhook / 发布链路各有 recover。
- **未 git commit，由 Lead 统一集成提交。**
