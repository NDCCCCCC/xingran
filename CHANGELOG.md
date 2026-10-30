# Changelog

All notable changes to XingRan-Next are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v1.20.1] - 2026-07-09 — 网络设备 VLAN + 端口绑定 (Network Device VLAN + Port Binding)

### Added

**Backend**

- 2 new port write actions: `set_access_vlan`（修改 access 端口 PVID，VLAN ID 1-4094）+ `port_binding`（add/remove IP + 可选 MAC 静态绑定）
- 2 new HTTP POST 端点（`POST /network/ports/write/{set-access-vlan,port-binding}`），复用 v1.19 `network:port:write` 权限 + SM2+SM4 请求体加密 + `sys_port_write_audit` JSONB 审计真相源
- 3 vendors × 2 new actions = 6 vendor-specific 命令模板（Huawei `port default vlan` / H3C `port access vlan` / Ruijie `switchport access vlan`；Huawei/H3C `user-bind static`/`user-bind`，Ruijie `switchport port-security binding`）
- 4 new sentinel validators（VLAN range / bind op / IPv4 严格 regex / MAC 归一 + null MAC 拒绝）— service 层为真相源，handler binding tag 为双重防线
- `PortResult.Extra` 字段（v1.20.1 新增）携带 audit 上下文（vlanId / bindOp / ipAddress / macAddress）供 `sys_port_write_audit.after_value` JSON 写入（INFRA-01）
- OperType 映射：set_access_vlan → `OperTypeUpdate (2)`；port_binding op=add → `OperTypeCreate (1)`；op=remove → `OperTypeDelete (3)`

**Frontend**

- 2 new single-port Modals（`SetAccessVlanModal` + `PortBindingModal`）— vlanId InputNumber 1-4094 / op Radio.Group / ipAddress IPV4_REGEX / macAddress MAC_REGEX（可选）/ reason Select + TextArea（`validateReasonRequired` 跨字段）
- 2 new networkApi wrappers（`writeSetAccessVlan` + `writePortBinding`）— LANDMINE #5 合规（无 try/catch，无 message.error，post() 拦截器统一处理）
- `PortWriteAction` union +2 literals + `BatchWriteRequest` +4 optional fields（为 FUTURE-BATCH-05 批量路径预留）
- ports/index.tsx ActionButtons 菜单 5 → 7 items（`修改 access VLAN` + `端口绑定`，`canWrite` gating 继承）

**Testing**

- 11 new `TestE2E_*` e2e 测试 via scrapligo FileTransport（Huawei + Ruijie × set_access_vlan + port_binding × variants），复用 v1.19 `fileTransportExecutor` 基建（无新基建）
- 10 new test fixtures（5 Huawei + 5 Ruijie）覆盖 2 actions × variants
- W2 service 层 7 个 VLAN validator + 16 个 binding validator 表驱动单测全绿

**Documentation**

- `docs/API响应规范.md` 新增 v1.20.1 扩展小节（2 端点签名 + SetAccessVlanRequest/PortBindingRequest schema + 4 sentinel 错误码 + 3 vendors × 2 actions 命令模板矩阵）
- `56-HUMAN-UAT.md` 12 项 site-visit SSH 验证推迟（Huawei S8700 × 6 + Ruijie RS8607E × 4 + H3C × 2 conditional）

### Phases

- Phase 56 (5 waves: W1 vendor template / W2 service+validators / W3 handler+router+audit / W4 frontend 2 Modals / W5 e2e+UAT+docs)

### Deferred

- 真机 SSH 验证（12 项 site-visit UAT，owner = 现场运维同事，下次现场访问）— 详见 `.planning/phases/56-vlan-v1-20-1-0-5-plans-initiated-2026-07-09/56-HUMAN-UAT.md`
- 跨固件版本命令差异（Huawei V200R005 vs V200R022+ `port link-type access` 前置必要性）→ site-visit 覆盖
- Ruijie 真机 port-security binding 表 MAC 列显示（用户实采输出无 MAC 列，RISK-02）→ site-visit 覆盖
- H3C Comware V7 `user-bind` 无 `static` 关键字验证（RISK-01）→ site-visit 覆盖（视现场 H3C 设备情况）
- 批量路径（`set_access_vlan` / `port_binding` 多端口）→ FUTURE-BATCH-05 推迟（v1.20.1 单端口 only）
- vendor-react bundle gzip: 774.95 kB（与 v1.19 baseline 一致，零回归）

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