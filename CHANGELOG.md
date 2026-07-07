# Changelog

All notable changes to RuoYi-Next are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v1.19] - 2026-07-07 — 网络设备写命令 (Network Device Port Write Operations)

### Added

**Backend**

- 6 HTTP POST 端点 (`POST /network/ports/write/{shutdown,undo-shutdown,description,dot1x-enable,dot1x-disable,batch}`) — 通过 SSH 直连对目标设备下发端口配置命令
- `internal/services/portwrite` 包：`PortWriteService` 接口 + 5 单端口方法 + 1 批量方法，`BatchResult` 三数组 succeeded/failed/skipped 编排，BATCH-02 serial fail-fast 语义
- `internal/services/portcollection.RenderCommand` vendor × action 派发表（Huawei / H3C / Ruijie 3 厂商 × 5 操作）；description 操作自动产出 `interface X` + `description Y` 2 cmds；Ruijie 走 Cisco 风格 dot1x (`dot1x port-control auto` / `no dot1x port-control`)
- 审计真相源 `sys_port_write_audit` 表（未脱敏 CommandSent）+ `sys_oper_log` 高层摘要（OperType 25 常量映射 CONV-01..04：status → `OperTypeStatus (10)`、description → `OperTypeUpdate (2)`、batch → `OperTypeBatch (16)`）
- 权限粒度 `network:port:write`（与既有 `network:port:query` 隔离）
- 错误分类 `WriteErrorTransport` vs `WriteErrorDeviceRejected` — 区分 SSH transport-level 错误（连接失败/超时/EOF）与设备 CLI 拒绝（`% Error:` / `Unrecognized command` / `Illegal`），批量 fail-fast 依赖此分类
- 写端点成功路径自动触发 `DeviceInfoCollectionService.Enqueue` 后台采集（v1.18 复用）
- `scrapligo v1.4.0` FileTransport 集成：service 层 e2e 测试使用 `transport.NewFileTransport()` + 预录制 Huawei VRP fixture 回放

**Frontend**

- `PortWriteModal` — 单端口写操作对话框（5 操作 × 自定义 reason 审计字段）
- `BulkWriteDrawer` — 批量写操作抽屉（多端口选择 + 实时进度）
- ports/logs 页面改造（菜单/表格/批量操作入口）
- types/network.ts + networkApi.ts 6 wrapper + port-write/constants.ts 5 const

**Testing**

- service 层 10 个 `TestE2E_*` e2e 测试（5 single happy + 1 batch happy + 4 类错误路径：device_rejected / transport_error / batch_failfast / noop），使用 scrapligo FileTransport 真实跑 SendConfigs + parseConfigError 链路（Phase 51 mock 不调 fn 漏洞闭环）
- operlog regression guard：25 OperType 常量 + 11 敏感关键词 + Record 5 参签名不回归
- 8 个 Huawei VRP fixture（testdata/huawei_*.fixture）覆盖 1 厂商链路

**Documentation**

- `docs/API响应规范.md` 新增"网络设备端口写操作"小节（6 端点签名 + PortWriteRequest + PortResult + BatchWriteRequest + BatchResult schema）
- `docs/安全和认证设计（国密）.md` 新增"网络设备写端点加密行为"小节（明确区分 SSH vs HTTP 两层加密 + config.yaml exclude_paths 实证）

### Phases

- Phase 50 (W1: vendor templates + unit tests + action/command map)
- Phase 51 (W2: PortWriteService + batch orchestrator + mock tests)
- Phase 52 (W3: router/handler/operlog/permission/migration)
- Phase 53 (W4: frontend Drawer + progress dialog + API wrappers)
- Phase 54 (W5: E2E FileTransport + real-device UAT deferral + documentation)

### Deferred

- 真机 SSH 写命令验证（6 项 site-visit UAT，owner = 现场运维同事，下次现场访问）— 详见 `.planning/phases/54-w5-e2e-real-device-uat-documentation/54-HUMAN-UAT.md`
- 跨固件版本命令差异（Huawei V200R005 vs V600R024C00）→ follow-up
- Real-device SSH 往返延迟测量 / batch per-port timeout 标定 → follow-up
- HTTP handler 层 e2e → v1.19.x+（Phase 52 handler 零测试基建 cold start 工作量大）
- 3 厂商（Huawei/H3C/Ruijie）e2e fixture 全覆盖 → v1.19.x+（fixture 字节序列脆性 ×3）
- BATCH-05 批量实时进度（SSE/WS）→ v1.19.x（Phase 53 indeterminate spinner MVP 过渡）
- `sys_port_write_audit` 详情查看 UI → v1.19.x+

## [v1.18] - 2026-07-04 — 网络设备硬件清单 (Device Component Serials)

详见 `.planning/MILESTONES.md` v1.18 段。