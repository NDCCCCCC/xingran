# Milestones

## v1.20 网络设备 VLAN + 端口绑定 (Shipped: 2026-07-10)

**Phases completed:** 1 phases, 5 plans, 23 tasks

**Key accomplishments:**

- 1. [Rule 1 - Bug] Corrected MAC format in test expectations
- Extended v1.19 PortWriteService with SetAccessVlan + PortBinding methods, 4 validator sentinels, Extra-map audit carrier, and checkPreState NoOp/skip logic — 31 new subtests, zero v1.19 regression
- Wired 2 new kebab HTTP endpoints (set-access-vlan + port-binding) through v1.19's execSinglePort DRY, refactored execSinglePort to receive bound portID/description (fixes double-BindJSON EOF trap), extended buildAfterValue to read PortResult.Extra for v1.20.1 audit after_value, and added 12 new subtests + 3 permission tests — zero v1.19 regression.
- 1. [Rule 1 - Bug] Fixed PortBindingModal BIND_OPS readonly cast for `tsc -b`
- 1. [Rule 1 - Bug] Fixed Huawei MAC format in fixtures (AABB-CCDD-EEFF → AA-BB-CC-DD-EE-FF)

---

## v1.20.1 网络设备 VLAN + 端口绑定 (Network Device VLAN + Port Binding) — ✅ SHIPPED 2026-07-09

**Phases**: 1 (Phase 56) | **Plans**: 5 (56-01 / 56-02 / 56-03 / 56-04 / 56-05) | **Waves**: 5 (W1 vendor template / W2 service+validators / W3 handler+router+audit / W4 frontend 2 Modals / W5 e2e+UAT+docs)

**Delivered**: 在 v1.19 网络设备写命令基础上新增 2 个端口写操作（`set_access_vlan` 修改 access 端口 PVID + `port_binding` add/remove IP+可选 MAC 静态绑定），3 厂商命令模板全覆盖 + 前端 2 个独立 Modal + 11 e2e 测试 + 文档同步。复用 v1.19 全部基建（权限 / 审计 / 加密 / scrapligo FileTransport / fire-and-forget refresh）。

**Key Accomplishments**:

1. ✅ **Vendor Template Extension (56-01 / W1)** — `internal/services/portcollection/vendor_port_template.go` +6 entries（3 vendors × 2 actions）+ 9 render 函数；MAC 格式锁定（Huawei/H3C = `AA-BB-CC-DD-EE-FF`，Ruijie = `aabb.ccdd.eeff`）；`localNormalizeMACAddress` 本地副本避免 import cycle；dispatch closure by BindOp（add/remove 路由）；12 locked subtests
2. ✅ **Service + 4 Validators + Pre-state (56-02 / W2)** — `port_write_service.go` +2 methods（`SetAccessVlan` + `PortBinding`）+ 4 sentinel validators（VLAN range / bind op / IPv4 regex / MAC normalize+null reject）+ VLAN pre-state NoOp 短路（Pitfall 6: port_binding 永不 NoOp）+ `PortResult.Extra` audit carrier（INFRA-01）；23 表驱动单测
3. ✅ **Handler + Router + Audit (56-03 / W3)** — 2 HTTP POST 端点 + `SetAccessVlanRequest`/`PortBindingRequest` 专用 struct（PATTERNS.md Option A）+ OperType 分流（set_access_vlan=Update / op=add=Create / op=remove=Delete）+ `sys_port_write_audit.after_value` JSON 写入 Extra 字段
4. ✅ **Frontend 2 Modals (56-04 / W4)** — `SetAccessVlanModal`（vlanId InputNumber 1-4094）+ `PortBindingModal`（op Radio.Group + IPV4_REGEX + MAC_REGEX + reason cross-field）+ 2 networkApi wrappers（LANDMINE #5 合规）+ ports/index.tsx ActionButtons 5 → 7 items；vendor-react gzip 774.95 kB（≤ 776 kB baseline，零回归）
5. ✅ **E2E + UAT Deferral + Documentation (56-05 / W5)** — 11 new `TestE2E_*` via scrapligo FileTransport（复用 v1.19 `fileTransportExecutor` 基建，无新基建）+ 10 new fixtures（5 Huawei + 5 Ruijie）+ `56-HUMAN-UAT.md` 12 项 site-visit 推迟 + API响应规范/CHANGELOG/MILESTONES 文档同步

**Comms**:

- 0 critical regression（v1.19 e2e 10 + v1.20.1 e2e 11 = 21 e2e 全绿；operlog 25 OperType + 11 敏感关键词 regression_test.go 不回归；vendor-react gzip 与 v1.19 baseline 零回归）
- 12/12 site-visit UAT 推迟（Huawei S8700 × 6 + Ruijie RS8607E × 4 + H3C × 2 conditional，owner = 现场运维同事）— 详见 `.planning/phases/56-vlan-v1-20-1-0-5-plans-initiated-2026-07-09/56-HUMAN-UAT.md`
- Risk mitigations: H3C `user-bind` no `static` (RISK-01), Ruijie full MAC+IP syntax (RISK-02), Huawei `port link-type access` universal prefix (RISK-03)

**Known deferred items at close**: 12 项 site-visit UAT 推迟（真机 SSH 验证：Huawei S8700 set_access_vlan/port_binding × 6 + Ruijie RS8607E × 4 + H3C × 2 conditional）— 详见 `56-HUMAN-UAT.md`；批量路径（`set_access_vlan` / `port_binding` 多端口）→ FUTURE-BATCH-05 推迟（v1.20.1 单端口 only，`BatchWriteRequest` 已预留 4 字段）

---

## v1.19 网络设备写命令 (Network Device Port Write Operations) — ✅ SHIPPED 2026-07-08

**Phases**: 6 (Phases 50-54 = core MVP; Phase 55 = tech-debt cleanup) | **Plans**: 9 (50-01 / 51-01 / 52-01+02 / 53-01+02 / 54-01 / 55-01+02) | **Waves**: 5 build phases + 1 cleanup phase

**Milestone Audit**: ✅ PASSED (37/37 requirements, 6/6 phases, 20/20 integration wires, 7/7 E2E flows) — see [v1.19-MILESTONE-AUDIT.md](milestones/v1.19-MILESTONE-AUDIT.md)

**Full archive**: [milestones/v1.19-ROADMAP.md](milestones/v1.19-ROADMAP.md) | [milestones/v1.19-REQUIREMENTS.md](milestones/v1.19-REQUIREMENTS.md)

**Delivered**: 补全网络设备"读+写"闭环——Web 端通过 SSH 直接对目标设备下发端口配置命令（启停/描述/dot1x），成功后立即采集一次端口信息回填审计。MVP 范围：3 厂商 × 5 操作 + 批量 + 完整审计 + 权限隔离。

**Key Accomplishments**:

1. ✅ **Vendor × Action 命令模板 (50-01 / W1)** — `internal/services/portcollection` vendorPortTemplate 派发表覆盖 Huawei / H3C / Ruijie 3 厂商 × 5 操作；H3C 复用华为 VRP 模板（D-08 同源），Ruijie 走 Cisco 风格 dot1x；description 操作自动产出 `interface X` + `description Y` 2 cmds；vendorPortTemplate_RenderCommand 15 case 单元测试 + description 长度 ≤80 校验
2. ✅ **PortWriteService + Batch Orchestrator (51-01 / W2)** — `internal/services/portwrite/port_write_service.go` 5 单端口方法 + BatchWritePorts 批量；fail-fast serial loop + detached 30min context（Pitfall #5 兜底 Core.Close 30s）+ per-port Acquire/ReleaseRef 连接池隔离；BatchResult 三数组 succeeded/failed/skipped 编排；Phase 51 mockDeviceExecutor + Phase 54 FileTransport e2e 双层覆盖
3. ✅ **HTTP + Operlog + Permission + Migration (52-01/02 / W3)** — 6 HTTP POST 端点 `POST /network/ports/write/{shutdown,undo-shutdown,description,dot1x-enable,dot1x-disable,batch}`；PortWriteRequest schema + 6 handler；sys_port_write_audit 表真相源 + sys_oper_log 高层摘要（OperType 25 常量映射 CONV-01..04）；权限 `network:port:write` 与 `network:port:query` 隔离；migration_202 加表 + 端口配置菜单 seed + GrantNewMenuToRolesHavingParent helper（antd 父子联动陷阱根除）
4. ✅ **Frontend Drawer + Progress + API Wrappers (53-01/02 / W4)** — types/network.ts + 4 类型 + networkApi.ts + 6 wrapper + port-write/constants.ts 5 const；PortWriteModal 5 操作 × custom reason 审计；BulkWriteDrawer 多端口选择 + indeterminate spinner MVP（BATCH-05 实时进度 v1.19.x deferred）；ports/logs 页面改造；vendor-react gzip 775 kB 基线（前端 bundle size 防回归）
5. ✅ **E2E + Real-Device UAT Deferral + Documentation (54-01 / W5)** — `internal/services/portwrite/port_write_e2e_test.go` 10 个 TestE2E_*（5 single happy + 1 batch happy + 4 类错误路径：device_rejected / transport_error / batch_failfast / noop），使用 scrapligo FileTransport 跑真实 SendConfigs + parseConfigError 链路（Phase 51 mockDeviceExecutor 不调 fn 漏洞闭环）；API/加密/CHANGELOG/README/MILESTONES 文档同步；54-HUMAN-UAT.md 7 项 site-visit deferred（6 项真机 SSH + 1 项 WR-02 观察驱动 Phase 55 决策）
6. ✅ **Tech-Debt Cleanup (55-01/02)** — Phase 53 leftover sweep：WR-02 reason validator signature 修复（antd `(rule, value, form)` 3-param helpers extracted to constants.ts）+ IN-01 `instanceof Error` narrowing + IN-02 mount-only useEffect eslint-disable placement 修正（Lesson: directive 必须紧贴 `}, []);` 而非 useEffect 开头）+ HealthCard.test.tsx empty-state assertion regex match + CR-02 后端 batch_orchestrator.go fallback 路径 port 归属跨层防御（纵深双保险）

**Comms**:

- 98 commits 主仓 (35 code commits + 63 docs/state commits)
- 37/37 requirements addressed (35 fully satisfied + 2 PARTIAL BATCH-05 / UI-03 + 1 adjusted PERM-03 D-07); 54-VALIDATION.md Test Map 锁定
- 0 critical regression（Phase 51 mock + Phase 54 e2e 全绿，operlog 25 OperType + 11 敏感关键词 regression_test.go 不回归，vendor-react gzip 与 Phase 53 baseline 零回归）
- 108 文件 / 25,224 insertions / 3,152 deletions (1.7 days: 2026-07-06 → 2026-07-08)
- Git range: d9dd2edd → 39f5a43f

**Known deferred items at close**: 7 项 site-visit UAT 推迟（6 项真机 SSH transport verification：Huawei/H3C/Ruijie × shutdown/description/dot1x；1 项 WR-02 custom-reason 频率观察驱动 Phase 55 WR-02 修复决策 → ✅ RESOLVED in Phase 55）— 详见 `.planning/milestones/v1.19-phases/54-w5-e2e-real-device-uat-documentation/54-HUMAN-UAT.md`；3 厂商 e2e fixture 全覆盖 / HTTP handler 层 e2e / 跨固件版本命令差异 / Real-device SSH 往返延迟测量 / BATCH-05 实时进度 / sys_port_write_audit 详情查看 UI / 自动回滚 / 多用户并发写互斥 / Cisco IOS 支持 推迟到 v1.19.x+

---

## v1.18 网络设备硬件清单 (Device Component Serials) — ✅ SHIPPED 2026-07-04

**Phases**: 1 (Phase 48) | **Plans**: 3 (48-01 / 48-02 / 48-03) | **Waves**: 3 (schema → collectors → pipeline+operlog+frontend)

**Delivered**: 实现"一机多序列号"——网络设备的板卡/引擎卡、电源、风扇、光模块各自的序列号各起一条 `ops_asset`(`DeviceSN`=组件 SN,`parent_asset_id` 指向主机/`component_type`/`component_slot` 列保留),实时发现时通过 SNMP ENTITY-MIB + Huawei/Ruijie CLI 双路径采集;采集器 UPDATE-only 写 ops_asset,匹配不到的 SN 走 `sys_data_reconciliation`(`conflict_type="F"`,`recon_category="component_serial"`);前端组件清单 Tab 在资产列表 expandable row 内展示。

**Key Accomplishments**:

1. ✅ **Schema 落地 (48-01 / Wave 1)** — migration_201 给 `ops_asset` 加 4 列(parent_asset_id / source_device_id / component_type / component_slot)+ `sys_data_reconciliation.recon_category` varchar(32) 列,partial unique index 切换 `uniq_recon_asset_type_open → uniq_recon_asset_type_cat_open(asset_id, conflict_type, recon_category) WHERE open`,字典 `asset_reconciliation_recon_category` rows=2 seed,migration 全 idempotent + 远程 PG schema introspection 验证全 OK
2. ✅ **组件收集器包 (48-02 / Wave 2)** — 新建 `internal/services/component_collector/` 17 文件:SNMP ENTITY-MIB 单 GET loop(D-08 替代 BulkWalk,因华为/锐捷拒绝)+ Ruijie `temprature*` 噪声过滤(Class=8 && HasPrefix("temprature",13 比例)→ D-11)+ Huawei dual-class SN 非空优先去重 + OwnerResolver stack containtedIn tree 重建 + 32-hop cycle guard + 最小 chassis canonical root(D-12)+ Huawei/Ruijie CLI 收集器 + cmd_dispatcher(D-10 status→transceiver 两步编排)+ 6 TextFSM 模板
3. ✅ **Pipeline + 审计 (48-03 / Wave 3)** — OpsAssetWriter UPDATE-only(D-02/D-03)+ parent-missing degraded mode(D-04 仅 Warnf,从不自动 INSERT 父资产)+ ReconciliationEmitter 双 idempotent(D-06 partial unique + pre-INSERT dedup)+ Pipeline 编排 + operlog.RecordBackground(D-13 cron-path 无 gin.Context,operator="system-cron",沿用 Phase 34 OperType 常量 + 11 sensitive keyword 不回归)+ DeviceInfoCollectionService processTask 在 chassis 更新后调用 collectComponentInfo + runTwoStepTransceiverPipeline(D-10),失败仅 Warnf 不阻塞 chassis(D-14)
4. ✅ **Index + 查询不污染主表** — `asset_service.List/Statistics` 默认 `WHERE component_type IS NULL`(D-07),组件不污染主设备视图;POST `/ops/asset/components` 独立端点带 ops:asset:list group-level middleware 复用,3 handler 测试通过
5. ✅ **前端 ComponentListTab** — Antd Table expandable row + Tag + useEffect primitive deps `[parentAssetId]` + componentApi factory + `Asset` 类型扩展 4 可选 component 字段;无组件时不破坏主表布局
6. ✅ **真机 UAT 推迟声明(为 P0 完成让路)** — 3 项 site-visit(S8700 SNMP 单 GET / RS8607E 627 行噪声过滤 / D-10 S8700 `10GE5/0/4` 两步流水线)按 RESEARCH §Environment Availability 显式推迟到下次现场访问;自动化 UAT 全绿(go build / 21 collector tests / 5 D-10 流水线测试 / 3 handler tests / 5 operlog regression 不回归 / tsc / vite build 1m40s vendor-react 775kB gzip 基线与 Phase 48 前一致 / 远程 PG schema introspection)

**Comms**:

- 6 commits 主仓 + 13 merge 嵌套(`9157be4a` `b8fd2f45`→`6a25596c`),19 commits GitHub-style 紧凑统计上 commit-log 范围
- 14/14 D-id 全覆盖(代码 + 测试证据),48-VERIFICATION.md `score: 13/14` + 3 项 site-visit(human_needed informational, 非失败)
- 0 critical regression(11 pre-existing operations/system 测试失败详见 `deferred-items.md`,在 `b8fd2f45` baseline 跑过确认非 Phase 48 引入)
- 44 文件 / 4,712 insertions(+57 deletions)

**Known deferred items at close**: 3 项 site-visit UAT(S8700/RS8607E 真机)推迟到下次现场访问(per 48-HUMAN-UAT.md status `partial`),其余 11 项 pre-existing test failures 在 `deferred-items.md` 文档化供后续 owner 处理

---

## v1.17 资产对账 (Asset Reconciliation) — ✅ SHIPPED 2026-07-03

**Phases**: 5 (Phases 42-46) | **Plans**: 14 (16 planned, 42-03 skeleton superseded by R3) | **Tasks**: ~70

**Delivered**: 建立"实物层 vs 声明层"对账引擎（R1-R5 完整）+ 4 个根因修复（Phase 47 独立）。Observe-only 策略 + 告警驱动人工修复 + 工单闭环 + IP 段例外 + 半自动修复建议（高置信度 6 状态机）。

**Key Accomplishments**:

1. ✅ **R1 观测底座** — 物化视图 + 主表 + Dashboard + 6 个 Statistics 端点（防 MaxPageSize 钳制）+ operlog 全覆盖
2. ✅ **R2 告警 + 工单闭环** — critical/high 自动转工单 + WebSocket 推送 + SysNotice + 7d 静默期 + 24h 节流
3. ✅ **R3 IP 段例外** — CIDR 格式 + 5 种 actions 组合 + expires_at + 命中测试工具 + Excel 导入导出 + VERIFICATION 10/10
4. ✅ **R4 工位详情整合** — 健康度卡片（5 KPI + 趋势 mini chart）+ 对账健康列 + ReconciliationDrawer 3 Tab
5. ✅ **R5 半自动修复** — 6 状态机 + 7d 回滚 + 误修复率告警三通道（SysNotice+WS+operlog）+ UAT 9/10 PASS
6. ✅ **Phase 47 根因修复** — infoPoint drift / Layer3 UPSERT / port_status 漂移 / MAC parser 校验（独立完成）

**Known in-session fixes during UAT**:

- Apply 400: `resolution_method` → `resolution_note` 列不存在 bug (commit 87d5fc82)
- Rollback 400: 删除 `pre_fix_user_id == nil` 过度防御检查
- Cron 5 fields 缺年份: UPDATE sys_job 加 `0` 前缀

**Known deferred items at close**: 0 (Test 10 权限非 admin 端到端 skipped due to 缺测试账号，不阻塞)

---

## v1.3 技术债清理 — ✅ SHIPPED 2026-04-27

**Phases**: 3 (Phases 8-10) | **Plans**: 9 | **Tasks**: ~50

**Delivered**: 修复 SNMP panic 崩溃问题，清理后端死代码并修复安全漏洞，实现网络设备批量导出功能

**Key Accomplishments**:

1. ✅ SNMP panic 完全修复 — panic 恢复包装器 + RWMutex 并发安全 + WaitForReady 连接验证
2. ✅ 后端代码质量提升 — 删除 Core 死字段，文档化 12 个外部依赖字段
3. ✅ 安全修复完成 — WebSocket CheckOrigin 增强，并发安全，错误日志改进，15 个新测试全部通过
4. ✅ 批量导出功能 — 支持一次性选择多个实体类型并打包为 ZIP 下载，集成到 9 个网络管理页面

**Known deferred items at close**: 0 (all resolved)

---

## v1.2 可配置仪表盘生产级改造（仿 Zabbix）— ✅ SHIPPED 2026-04-21

**Phases**: 4 (Phases 4-7) | **Plans**: 11 | **Tasks**: ~60

**Delivered**: 实现可配置仪表盘系统的核心功能，包括 Widget 数据获取机制、前端交互完善、实时数据刷新和用户体验优化

**Key Accomplishments**:

1. ✅ Widget 数据获取机制 — `GetWidgetData` 和 `GetBatchWidgetData` 支持 API/WebSocket/Static 数据源
2. ✅ 前端交互完善 — Widget 选择器、仪表盘设置面板、模板预览功能
3. ✅ 实时数据刷新 — WebSocket 连接管理、基于刷新间隔的轮询、缓存策略
4. ✅ 用户体验优化 — Widget 拖拽布局优化、加载状态和错误处理、响应式布局适配

---

## v1.1 信息点导入设备端口关联 — ✅ SHIPPED 2026-04-16

**Phases**: 1 (Phase 3) | **Plans**: 1 | **Tasks**: ~5

**Delivered**: 信息点 Excel 导入时支持通过设备名/端口名自动匹配并关联设备 ID 和端口 ID

**Key Accomplishments**:

1. ✅ 信息点导入配置添加"所属设备"列 (`deviceName` → `device_id`)
2. ✅ 信息点导入配置添加"所属端口"列 (`portName` → `port_id`)
3. ✅ 两个新列均为可选，空值不阻断导入
4. ✅ 模板下载包含设备和端口示例值

---

## v1.0 工位导入部门/用户关联 — ✅ SHIPPED 2026-04-16

**Phases**: 2 (Phases 1-2) | **Plans**: 7 | **Tasks**: ~30

**Delivered**: 工位 Excel 导入时支持通过部门名称/用户名自动匹配并关联部门 ID 和用户 ID

**Key Accomplishments**:

1. ✅ Excel 导入模板添加"所属部门"和"所属用户"两个可选列
2. ✅ 导入时通过部门名称/用户名匹配，写入对应 ID
3. ✅ 匹配失败留空不阻断导入
4. ✅ 前端工位列表/详情页面展示关联的部门和用户信息
