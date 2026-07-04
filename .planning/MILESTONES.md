# Milestones

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
