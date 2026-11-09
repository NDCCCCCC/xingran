# v1.54 架构裁定（Lead 拍板 / 供 planner 与 executor 作为权威输入）

**日期**：2026-11-08
**依据**：3 个并行只读探索 subagent 报告（发现设备导入 / 告警通道 / 端口写）+ 本仓实证
**地位**：本文件是 v1.54 的「设计裁决」，与 `.planning/REQUIREMENTS.md` 一致时以本文件为实现口径；冲突时回 Lead 裁定。

---

## 一、Phase 142 — 发现设备真实导入（DISC-01/02）

### 已确认事实
- FIX-07（v1.53 Phase 137）已把发现明细落进真实表 `sys_device_discovery_result`（`internal/models/device_discovery_result.go:13-27`），`internal/core/db/database.go:703` 已注册 AutoMigrate。
- `ImportDiscoveredDevices`（`internal/services/device_discovery_service.go:509-523`）现状只 `return len(deviceIDs), nil`；handler `discovery_handler.go:310-327` 恒传 `deviceIDs=nil`、前端 POST `{}`；`DiscoveredDevice` DTO（:74-84）无 ID 字段。
- `NetworkDevice`（`internal/models/network_device.go:34-57`）唯一约束只有 `ip_address` 的 `uniqueIndex`（:41），且**不含 deleted_at** —— 软删行仍占位。
- operlog（`discovery_handler.go:325` 的 `OperTypeImport`）、权限码（`network:discovery:add`）、路由（`POST /network/discoveries/:id/import`）、响应封装均已就位，**不新增**。

### 裁定
| 未决点 | 裁定 | 理由 |
|---|---|---|
| `deviceIDs` 语义 | **忽略该参数、恒全量导入该任务全部结果行**；保留签名兼容，不新增 DTO 字段、不加 `Imported` 标记、不加 migration | 前端 POST `{}`、DTO 无 ID、结果表重跑会换 ID；按 IP 幂等天然防重复，最小改动零 DDL |
| 去重策略 | **按 `ip_address` 归一（trim + `net.ParseIP` 校验）**：① 批内去重保留最后一条 ② 库内未软删命中 → **跳过、不覆盖**（保护人工维护字段）③ `Unscoped` 软删命中 → **恢复**（清 `DeletedAt` + 覆盖字段 + `tx.Save`），复用 `network_device_service.go:296-319` | 与既有 `NetworkDeviceService.Create` / `QuickCreateDevice` 语义一致；避免 Upsert 静默覆盖 |
| 部分成功 vs 原子 | **单事务全原子**：任一行失败整体回滚 `return 0, fmt.Errorf("导入发现设备失败: %w", err)`；无结果行 `(0, nil)` 不算错误；任务不存在保持现有错误 | `(int, error)` 语义干净、与 handler 1500 分支一致；不静默吞错 |
| 字段映射 / 兜底 | `SysName 空 → Device-<末段IP>`；`DeviceType 空/越界 → models.DeviceTypeSwitch`；`Vendor 空 → ""`（列允许空串，not null 是代码层）；`IsAlive=false`/非法 IP 跳过；`Status` 按 `DeviceStatus`（0在线/1离线/2未知）赋值，**不套用通用 0正常/1停用**；`DeptID` 取 `task.GroupID`（校验合法 UUID）；`CreatedBy/UpdatedBy` = 入参 `createdBy`（不要照抄 `network_device_service.go:215-228` 漏赋值的写法）；`SerialNumber/SoftwareVersion/Uptime/CredentialID` 留空 | 与 `IdentifyVendor`/`IdentifyDeviceType` 行为及 `QuickCreateDevice` 恢复分支对齐 |
| 缓存失效 | **在 handler 成功路径、`operlog.Record` 之前**调 `networkServices.CacheService.InvalidateAllDeviceCache(ctx)`（注入 `CacheService` 到 `DiscoveryHandler`）；失败静默 `_ =` | `DeviceDiscoveryService` 只注入了 `*gorm.DB`（`core.go:454`），加 service 依赖会动 3~4 处构造点；走 handler 最小改动。缓存失效放事务外 |
| 读路径 | 导入后**不删** `sys_device_discovery_result`；`GetDiscoveryResults` 不改 | 保留复查/审计；幂等靠 IP |
| 顺带隐患 | `user_id.(string)` 硬断言 panic、`persistDiscoveryResults` 未开事务、`ExecuteDiscovery` 未检查 Save Error —— **不在本里程碑修**，登记为后续候选 | scope constrainment（CLAUDE.md） |

### 必须同步改的既有测试
- `internal/api/v1/network/discovery_handler_test.go:317-340`（写死 `{"count":0}` 的 stub 断言）
- `internal/services/device_discovery_service_79_06_test.go:419-450`（写死 `n==3` 的 stub 断言）

### 新增测试（就近 `*_test.go`）
`internal/services/device_discovery_service_import_test.go`：全量导入 / 批内同 IP 去重 / 库内已存在跳过 / 软删恢复 / 死 IP 与非法 IP 跳过与兜底 / 空结果 / 任务不存在 / 原子回滚 / 幂等（第二次 count=0）/ 导入后结果表仍在。
**测试红线**：`handlers_test_helpers_test.go` 的 `SetMaxOpenConns(1)` 单连接下，事务回调内禁止再发 gorm 查询、Rows 未读完禁止嵌套查询，否则死锁；service 单测用 `newDdv7906` 文件型 sqlite 并追加 `&models.NetworkDevice{}` 迁移。

---

## 二、Phase 144 — 同步漂移告警通道收口（ALRT-01/02/03）

### 已确认事实
- FIX-11 后端服务已存在：`internal/services/addomain/ad_sync_drift_alert_service.go`（418 行，5 个 `sys_config` 热重载旋钮、`detectADDrift`、`CheckAndAlert` 软失败、`emit()` 三通道：applogger / `sys_oper_log` direct INSERT / `sys_notice`）；cron `internal/scheduler/ad_sync_tasks.go:226-241`；11 例单测。
- **实质缺口大于 SUMMARY 自报**：`emit()` 直接 `s.db.Create(notice)`（service.go:405），**绕过 `NoticeService.CreateNoticeWithTargets + PublishNotice` 发布链路**，导致 `cron.go:715` 的 `new_notice` 铃铛推送也不触发 —— 现状连铃铛都收不到。
- `scheduler.NoticeHub` 接口（`cron.go:507-509`）与 `noticeHubAdapter`（`router.go:38-47`）只有 `BroadcastToUsers`，**没有 `BroadcastToAll`**；`internal/websocket/notice_hub.go` 本身有 `BroadcastToAll`(L233) 与 `NoticeMessage`(L49-58)。
- 前端 `NotificationBell.tsx:76-101` 已建立**全局唯一** WS 连接 `/system/ws/notices` 并转 `noticeStore.handleWsMessage`；`src/types/notice.ts:200-206` 的 `WebSocketMessageType` 无 drift 类型。
- 通用 Webhook 能力完整：`sys_api_notification_config`（`configType="webhook"`）、`APISenderService.Send / SendWithDefaultConfig / SendWebhook`（重试+30s 超时+2xx 判定）、`notification_config_handler.go` 已用 `RecordWithBody`；**无企微/钉钉专用实现**（仅 `rpa/notification.go:21` 注释枚举）。
- `configs/config.yaml` 与 `internal/config/config.go` **无** webhook/alert 段；5 旋钮走 `sys_config` 每轮热重载；SYS_CONFIG 默认行无 seed（`134-SUMMARY.md:32`）。
- 活跃 migration 最大编号 **237**；`src/constants/permissionCodes.ts` / `pkg/permission/config.go` / `sys_menu.perms` 三处须逐字一致（`scripts/check-permission-codes.mjs` 会 exit 1）；漏 F 型菜单 seed 则非 admin 恒 403（`migration_237` 同型坑）。
- 用户提及的 `sys_notice_config` **不存在**，实际是 `sys_email_config` / `sys_api_notification_config` / `sys_notification_channel`。

### 裁定
| 未决点 | 裁定 | 理由 |
|---|---|---|
| WS 注入方案 | **方案 A：扩 `scheduler.NoticeHub` 接口加 `BroadcastToAll(message interface{})` + 同步扩 `noticeHubAdapter`**，`emit()` 经 `scheduler.GetNoticeHub()` 调用；**不用**方案 B（从 core 注入 `*websocket.NoticeHub`） | 贴合服务「零新依赖/自包含」哲学，改动面小，不动 core 装配 |
| 广播兜底 | 广播处必须 `defer recover()`（hub 关闭写 channel 会 panic，先例 `fix_suggestion_monitor.go:140-145`）；只在 `shouldAlert==true` 的 `emit()` 内触发（避免每轮刷屏）；失败仅 WARN、保持 `CheckAndAlert` 软失败契约（L169/L360） | 不阻断 cron |
| 铃铛收不到 | **一并修**：`emit()` 通道 3 改走 `services.NewNoticeService(db).CreateNoticeWithTargets(...) + PublishNotice(...)`（照 `fix_suggestion_monitor.go:156-173`），让 `new_notice` 与新的 `ad_sync_drift_alert` 双事件都能推 | 属同一缺口的根因，不修则 WS 块无意义 |
| 消息格式 | 复用 `websocket.NoticeMessage`，`type: "ad_sync_drift_alert"`，业务 payload JSON 内嵌 `content`（照 `reconciliation_workorder.go:349-377`）；在 `notice_hub.go:13-18` 加常量 `MessageTypeADSyncDriftAlert` | 与既有自定义事件先例一致 |
| 前端 WS | **不新建第二条连接**：扩 `src/types/notice.ts` 的 `WebSocketMessageType` + `ADSyncDriftAlertEvent`，扩 `noticeStore.handleWsMessage` 分支 + 模块级 `driftAlertListeners` Set（照 portWrite/RPA 模式），页面订阅 + `App` message.warning | 双连接会导致双消息 |
| webhook 路线 | **路线 A（最小增量、零新表）**：复用 `sys_api_notification_config`（`configType=webhook`）+ `APISenderService.SendWithDefaultConfig(ctx, models.APIConfigTypeWebhook, msg)`；新增 2 个 `sys_config` 旋钮 `ad_sync.drift_alert.webhook_enabled`(默认 false) / `webhook_config_id`(空=默认)；**不做**企微/钉钉专用 markdown + 加签（路线 B，列后续） | 全仓无专用实现可参照，工作量与风险高；路线 A 完全复用既有设施 |
| 配置双源 | **统一走 `sys_config`**，不新增 `configs/config.yaml` 段、不加 `internal/config` struct 字段 | 避免与每轮热重载形成双源冲突 |
| 密钥脱敏（重点风险） | ① 所有告警配置写端点用 `operlog.RecordWithBody`；② **复用前必须补** `notification_config_handler.go` 的 `GetAPINotificationConfig`(L357-371) / `ListAPINotificationConfigs`(L294-343) 对 `authConfig`/`headers` 中 token 的主动遮蔽；③ webhook URL 值里的 `?key=`/`access_token=` 不会被 `FilterSensitiveParams`（按 JSON key 名遮，operlog.go:129-134）遮住 → 若需新增关键词，必须**同步** `internal/utils/operlog/regression_test.go` 的 `mandatorySensitiveKeywords`（常量集有回归守护） | 防止泄露机器人密钥 |
| 菜单/权限 | 新 **migration 238**：C 型页面菜单 + F 型操作码 seed（参照 `migration_221_manager_sync_menus.go` 幂等 count-then-insert + `migration_237_workorder_rating_perm_seed.go` 的 `F_TYPE_SEED_BEGIN/END` + `grantActionPermissionToParentHolders`）；顺带补 5+4 个 `sys_config` 默认 seed（补 `134-SUMMARY.md:32` 缺口） | 单 migration 收口 DDL/seed |
| 权限码命名 | `system:adDriftAlert`（C 型）+ `:list` / `:config` / `:edit` / `:trigger`（F 型）；挂「系统管理」根；三处逐字一致；operlog module 中文名 `"同步漂移告警"` / `"同步漂移告警配置"` | 与通知配置 `system:config` 同域 |
| 端点 | `POST /system/ad-drift-alert/{status,logs,config,config/update,trigger}`；`status/logs/config` 走读权限，`config/update`+`trigger` 走 `RequireStrictPermissions`；全部 `response.Success()/Error()`、`c.Request.Context()` 透传 | 与 `notification_config_router.go` 惯例一致 |
| 前端路由 | **不改 `App.tsx` 静态表**：菜单驱动 `componentLoader.tsx:36-53` 的 glob 自动收 `system/ad-drift-alert/index`；在 `src/constants/routes.ts` + `pageTitles.ts` 加常量 | 既有架构 |
| 新增 `ws_enabled` 旋钮 | 加 `ad_sync.drift_alert.ws_enabled`(默认 true)，允许关 WS 只留铃铛 | 可观测/可降级 |

### 软失败契约（不可破）
`CheckAndAlert`(L169) + `emit`(L360)：任何错误仅 `applogger.Warnf`、返回 nil、绝不阻断 cron。WS / webhook 新通道必须保持（recover + nil error）。

---

## 三、Phase 143 — 端口写物理参数收口（PORT-01/02）

### 已确认事实
- v1.53 FIX-04 已交付：`internal/services/portcollection/physical_template.go`（`set_speed`/`set_duplex`/`set_mtu` 三动作 × huawei/h3c/ruijie 三家模板）、`internal/services/portwrite/physical.go`（`validatePhysicalRequest` + `PhysicalWrite`）、`internal/api/v1/network/port_write_physical_handler.go` + `write.POST("/physical", ...)`、dry-run 预览、审计 `buildAfterValue`、4 组测试。**后端单条链路完整，但前端零调用**（`networkApi.ts` 无 `writePhysical`、`PortWriteAction` 联合类型 15 动作不含物理三动作）。
- 批量链路（Phase 122）`batch_orchestrator.go`：`BatchWriteRequest` 无 4 个物理字段；入口 validator switch 只认 `set_access_vlan`/`port_binding`；循环内 `executeWrite(..., nil, nil)` 丢弃 `renderExtras`；不调 `CheckActionSupportParams`；`PortResult.Extra` 物理回显只写在单条 `PhysicalWrite` ⇒ **批量 `audit.after_value` 恒 `{}`**；handler sentinel 表缺 `physicalSentinelMessage`/`ErrActionNotSupported` 的 400 分支。
- 能力矩阵 `internal/services/portcollection/device_capability.go`（362 行）：`capabilityEntry{baseline, modelRules, vendorUnsupported}` + `modelCapabilityRule{matchModelSubstr, unsupported, supported}`，`lookupSupport` 优先级 = 型号 unsupported > 型号 supported > 厂商 unsupported > 基线 > 默认 Supported。**三个物理动作在矩阵中零条目 → 恒默认 `ActionSupported`**；`physical_template.go:17-20` 注释自述「命令未针对三家真机逐一验证…若现场型号关键字不同，需真机 UAT 后追加 modelRules」。
- **型号来源已接通**：`writeSinglePort`（`port_write_service.go:438-449`）与 `dry_run.go:239-249` 已把 `dev.Vendor, dev.Model` 传入 `CheckActionSupportParams`；`PreviewResult.Model` 已回显前端（BulkWriteDrawer 第 1093 行显示 `{vendor} {model}`）。`NetworkDevice.Model string`（`network_device.go:40`），来源 = 手工 / SNMP 发现 `NewModelExtractor` / SSH `display version` 仅空时回填。`sys_object_id` 已被 `GetSystemInfo().ObjectID` 采集但**未落库**。
- 进度事件 `batch_progress.go` 的 `newProgressPublisher` / `emit` / `emitCompleted` 与 `PortWriteProgressMessage.Action string` 均**与 action 无关，零改动可复用**；前端 `BulkWriteDrawer` 的 `LiveProgress` / WS 订阅 / `audit/list` 断线对账已完备。
- `constants.ts` `ACTION_TITLE` 恰好 15 键，且 `__tests__/constants.test.ts:57` 断言 "exactly 15 keys" 会卡扩展。
- `pkg/permission/config.go` 的 route-permission registry 已登记 `/trunk`、`/portsec`，**未登记 `/network/ports/write/physical`**（组级 RBAC 中间件仍生效，但注册表口径不一致）。

### 裁定
| 未决点 | 裁定 | 理由 |
|---|---|---|
| 型号级能力矩阵载体 | **代码常量 + 测试锁定**（新增 `portcollection/physical_capability.go`：`physicalProfile{AllowedSpeeds, AllowedDuplexes, MTUMin/Max, SpeedCommand, DuplexCommand, MTUCommand, RequiresView, Evidence}` + `capabilityEntry` 扩 `physical` / `modelPhysical` 字段 + `ResolvePhysicalProfile(vendor, model)` + `CheckPhysicalSupport(vendor, model, action, params)`） | 与现有 `vendorCapabilities`（注释即「真机事实唯一真相源，测试锁定」）同构；真机结论必须走 code review + `*_test.go` 守护，不用 DB 表/yaml 覆盖层（避免引入写操作/权限/迁移） |
| 型号 key | **只用 `NetworkDevice.Model`**，子串匹配复用现有 `matchModelRules` 语义；**不做** `SysObjectID` 落库 + migration | 零 schema 变更、零 DDL；避免与 Phase 144 的 migration 238 / 跨 phase 冲突；OID 仅采集未持久化登记为 Known Gap |
| 回退链 | 型号 `modelRules.unsupported` 命中 → 拒绝（400，Evidence 必含依据）→ 型号 `modelPhysicalRule` 命中 → 用该型号 profile → 厂商 `vendorUnsupported` 命中 → 拒绝 → 厂商 `entry.physical` 基线 → 用厂商 profile（Evidence 标「厂商基线，未逐型号取证」）→ 全局默认（FIX-04 现状交集 `{10,100,1000,10000}` / `half|full|auto` / `[1280,9216]` / `"mtu %d"`，`Evidence=""`） | 严格对齐现有 `lookupSupport` 优先级 |
| 「默认放行 ≠ 有证据」 | 新增 `Evidence` 空值语义 + `PreviewResult` 回显 `evidenceLevel: "unverified"` + 审计 `Extra` 留 `physicalEvidence:""`；**不阻断写操作**、不冒充真机结论 | 借鉴 `ciscoUnsupportedReason` 诚实标注教训；REQ 未要求真机 |
| 校验/渲染同源 | `PortTemplateParams` 加 `Model`；`RenderCommand` 传型号到 `renderPhysical` 按 `ResolvePhysicalProfile` 取命令形态与范围；`validatePhysicalRequest` 改为接 `(vendor, model, action, params)`，与 `CheckPhysicalSupport` 同源，保证「校验放行 ⇔ 渲染接受」 | 型号化后不能靠注释维持单一真相源 |
| Cisco / 迈普物理动作 | **显式标 `vendorUnsupported`**（理由「渲染层未实现 + 无真机取证依据」），不补模板 | 不能继续靠 `ErrUnsupportedVendor` 含糊兜底 |
| 批量接入 | **允许**：`BatchWriteRequest` 加 4 字段；入口 switch 加物理三动作 `validatePhysicalRequest` + `CheckPhysicalSupport`；循环两处 `executeWrite` 传 `&renderExtras{physical:...}`；抽 `applyPhysicalAuditExtra` 公共函数修 `audit.after_value`；handler sentinel 补 400 分支；`SkipPreflight` 字段显式登记「批量不支持跳过预检」 | 复用进度事件零改动、三数组错误聚合零改动 |
| 批量回滚/瞬断风险 | **不纳入 critical 自动回滚**（沿用物理动作现有非 critical 语义）；在 dry-run 预览与审计显式警示「批量物理参数无自动回滚，失败需人工按 `audit.before_value` 恢复」；登记为 Known Gap | 避免引入新快照/逆序命令复杂度与 `DevicePortStatus` 无 MTU 字段的不可回溯；scope constrainment |
| 前端 | 补 `networkApi.writePhysical()` wrapper（wrapped `post`，不手动 `message.error`）；`PortWriteAction` 15→18、`ACTION_TITLE` 15→18 + 物理选项常量、测试断言同步改 18；`BulkWriteDrawer` 的 `ACTION_OPTIONS` 加 3 项 + `buildRequest` 组装 4 字段 + 条件表单 + 预览 `evidenceLevel` 徽标；单条 `PortWriteModal` 加物理分支（或独立 Modal，挂 `ActionButtons`，`canWrite` 门控不变） | 让 FIX-04 从「后端可调用」变成「用户可用」 |
| 权限注册表 | 顺手补登 `/network/ports/write/physical` 到 `pkg/permission/config.go`（复用 `NetworkPortWrite` 常量）+ 测试 | 收尾清理，口径一致 |
| e2e 归位 / OID 落库 / 真机逐型号取证 | **本里程碑不做**，登记为后续候选 / 待人工验证，不伪造闭环 | scope constrainment；真机结论不可由 AI 编造 |

### 测试（就近 `*_test.go`，全部不依赖真机）
- 新增 `portcollection/physical_capability_test.go`：表驱动（型号命中/未命中/空 Model/大小写/优先级/回退链/`Evidence`/枚举范围越界/命令形态分叉），沿用 `device_capability_matrix_test.go:138-175` 的「备份 map → 替换 → defer 恢复」手法。
- 扩 `portcollection/physical_template_test.go`（型号 profile 改变命令体、未取证默认画像）、`portwrite/physical_test.go`（vendor/model 校验矩阵、Extra 回归）。
- 新增 `portwrite/physical_batch_test.go`：批量入口 validator、4 参数透传、`audit.after_value` 非空、fail-fast、三数组聚合、进度逐端口事件（复用 `batch_progress_test.go` 的 `fakeProgressNotifier`）、型号 Unsupported 批量拒绝。
- 扩 `internal/api/v1/network/port_write_physical_handler_test.go` + 批量 handler sentinel 400 / operlog 位置 / `createAuditRowsInTx` 原子性；`pkg/permission/config_test.go` 补注册表断言。
- 前端：`constants.test.ts`（15→18）、`BulkWriteDrawer.test.tsx`（物理动作 preview→confirm、参数透传、WS 进度）、`PortWriteModal.test.tsx`（物理表单）、`networkApi.test.ts`（`writePhysical` 路径）。

### 明确登记（不伪造闭环）
真机逐型号取证清单 = 速率关键字/10000 合法性、双工在光口/10G 的合法性、MTU 命令形态（`jumboframe enable` / `system mtu` / 接口 `mtu`）与真实上下界、生效语义与链路瞬断、回读验证（`sys_device_port_status` 有 Speed/Duplex 无 MTU）、型号字符串形态覆盖、Cisco/迈普显式 Unsupported 裁定。全部沿用 `physical_template.go:17-20` 的「未取证」诚实标注，写入 SUMMARY 的 Known Gaps。

---

## 四、全局硬约束（所有 phase）
- Handler-Service；service 返回 `error`；`response.Success()` / `response.Error()`（禁止裸 `c.JSON`）；写操作 success path 在 `response.Success` **之前** `operlog.Record(...)`；敏感端点 `RecordWithBody`。
- `0=正常 1=停用`（但 `NetworkDevice.Status` 用 `DeviceStatus` 0在线/1离线/2未知，按既有设计）；UUID 主键；SQLite+PG 双方言（纯 GORM、避免 `::uuid`/`ILIKE`/裸 `ON CONFLICT`）。
- 前端 wrapped API（`@/lib/api.ts` 的 `post/get/put/del`）、Zustand、禁止裸 axios、不新增 WS 端点。
- Go 单测就近业务包内 `*_test.go`；集成 `tests/integration/`；e2e `tests/e2e/`；前端 `*.test.ts(x)` / `__tests__/`。
- 构建产物只落 `out/`；禁止落盘 `.xlsx`；`data/` 不入库。
- **不伪造人工验证闭环**：79 条人工验证跟踪表保持 0 勾选；FIX-01/04/09/13 验证类残余维持原登记；6 项 LOCKED 残余不写修复代码。
- 回归基线：`go build ./...` + `go test ./...` + `tsc` / `type-check` + `lint:ci` + `vitest` + `node scripts/check-permission-codes.mjs` 全绿（基线 `go build ./...` 已于 2026-11-08 实测全绿）。

---

## 五、并行 wave 冲突面（W1 三 phase 零依赖）
| Phase | 独占文件面 | 不动 |
|---|---|---|
| 142 | `internal/services/device_discovery_service*.go`、`internal/api/v1/network/discovery_handler.go`、`internal/api/v1/network/network_router.go`（仅注入 CacheService）、网络模块测试 | `internal/api/router.go`、migrations、前端 |
| 143 | `internal/services/portwrite/**`、端口写前端组件、`internal/services/portwrite/*_test.go` | 同上 |
| 144 | `internal/services/addomain/ad_sync_drift_alert_*.go`、`internal/scheduler/ad_sync_tasks.go`、`internal/scheduler/cron.go`（扩接口）、`internal/websocket/notice_hub.go`、`internal/api/router.go`、新增 `internal/api/v1/system/ad_drift_alert_*`、`migration_238_*`、`xingran-react-frontend/src/**`（告警页相关） | 网络模块、portwrite |

> `internal/api/router.go` 仅 144 动；migration 238 仅 144 动；`internal/scheduler/cron.go` 仅 144 动（142/143 不碰 scheduler）。三者文件面互斥，W1 可并行。
