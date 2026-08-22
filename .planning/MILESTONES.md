# Milestones

## v1.22 前端品牌化改造 (Frontend Brand Design-System) — ✅ SHIPPED 2026-08-18

**Phases**: 4 (Phases 64-67) | **Plans**: 4 (64-01 / 65-01 / 66-01 / 67-01) | **Tasks**: 33（含 Post-T5 补丁 + 终验）

**Delivered**: 把 `brand-spec.md` 像素实测品牌品牌（深绿 `#156031` × 铜金 `#C09058` × 奶油 `#F0ECE3`）固化进 `xingran-react-frontend/src/design-system/`，后台内部组件与登录页品牌一致；53 屏业务页面自动继承样式（不逐屏施工）。范围严格遵循锁定决策（仅 design-system 层；不逐屏改造 53 屏；D-01 全局替换不保留多主题；D-03 按钮纪律；D-05 业务代码零改动）。

**Key Accomplishments**:

1. ✅ **Phase 64 品牌令牌层落地** — `src/index.css` 253 变量全量改为 brand-spec 实测值（深绿/铜金/奶油/白卡/描边/次级文字） + `tokens/colors.ts` 新增 `xingranBrand` 11 键常量导出（绿6/铜金4/奶油9/渐变2/功能6/onDark3，含 OKLch + WCAG 注释） + `AntdThemeBridge.tsx` 16 项 token + 8 组件覆盖接品牌 + `tokens/shadows.ts/spacing.ts/typography.ts` 双层纸感调性对齐（深绿阴影 + 8px 圆角 + PingFang SC / JetBrains Mono / Songti SC 字体栈） + QA-01 对比度自动验证 vitest module（31 条断言，含 D-03 反向断言锁住 `#FFFFFF on #C09058 < 3.5:1`）

2. ✅ **Phase 65 主题系统收敛** — 删除 `design-system/themes/` 6 套主题目录（minimal/glassmorphism/neumorphism/flat2.0/luxury-quiet/ink-amber × 20 文件）+ `ThemeSwitcher.tsx`（65 行）+ `ColorSwitcher.tsx`（477 行）+ `types/theme.ts` + `utils/color.ts` + `tokens/shadows.ts/spacing.ts` 死文件（themes/ 删除后零消费）；themeStore 重写为单 `mode` 字段；AntdThemeBridge 三层优先级链简化为直接 `xingranBrand.greenPrimary`；ConfigProvider 移除 save-theme-settings 监听；InnovativeLayout 仅删 4 行（2 import + 2 渲染）；light/dark 双模式保留（设置页 + ThemeProvider + data-color-mode）；layoutStore 3 套布局 + 密度切换完整保留（diff-zero 证据）。vendor-react gzip 持平 774.94 kB；源码层 -4357 行

3. ✅ **Phase 66 通用组件层 + 硬编码色扫描** — AntdThemeBridge 扩展补齐 Phase 64 UAT 4 Gap（Table.headerBg / Input.activeBorderColor/hoverBorderColor + 2px 焦点环 / Tag.defaultBg=#FEF3C7 + defaultColor=#B88850 / Button.primaryColor=#FFFFFF 白字）+ 侧边栏深绿化（light 模式 `#14532D` 配方 + `--header-bg` 解耦防顶栏染色；sidebar.tsx 双根因修：内联白底清理 + Menu theme="dark"）+ ECharts 品牌系列色（`tokens/echartsTheme.ts` 新建 brandSeriesColors / brandHeatRamp / brandAxisLine / brandSplitLine / brandTextStyle / brandAreaFade / brandHeatZero 全出自 xingranBrand）+ `scripts/check-hardcoded-colors.mjs` 零依赖 scanner（node 内置 fs/path，相对脚本路径解析；denylist 覆盖 indigo/tailwind/slate/Antd 旧色板；路径 allowlist 5 / 值 allowlist 4）→ `--fix` 模式应用 426 项替换 / 95 文件 + 集成进 `npm run lint` + 包入 CI；scan 报告模式 0 命中 exit 0。Post-T5 补丁 4 commits（scanner 路径 / CmdK Tag 内联覆盖 / 全局 `.ant-tag` 覆盖 / 表头+focus 变量）

4. ✅ **Phase 67 终验** — `npm run build` exit 0 / `type-check` exit 0 / `lint` exit 0（eslint 0 errors + scanner 0 命中）/ `vitest` 14 files / 120 tests（5 fail 为 Phase 53 v1.19 网络设备端口写 UI 测试，与 v1.22 正交，**不阻塞 SHIPPED**）；vendor-react gzip 774.94 kB 持平；dist chunk 数 134 / 最大单 app chunk 131 kB 无回归；六屏视觉确认（仪表盘/系统用户/工位管理/监控仪表盘/资产对账看板/登录页）品牌化框架实测全 PASS；与 refs/ 旧截图对比 indigo/slate/冷蓝灰冲突色全部清零

**关键成果**：后台内部 `#4F46E5` indigo 主按钮 + `#F1F5F9` 冷蓝灰表头 + `#1e293b` slate 侧栏三个最大品牌化缺口全修复；登录页 → 后台视觉一脉相承（深绿 + 铜金 + 奶油 + 白卡双层纸感）；后端零改动；业务页面零改动。

**遗留 v1.23+ 候选**（不阻塞 SHIPPED）：

- **PROTO-01..04 逐屏原型对齐**：`PROTOTYPE-VS-ACTUAL.md` 差异清单（24 处路由前缀、53 屏字段 / 表头 / 工具栏 / 菜单 / 空状态 / 统计卡对齐）
- **VIS-01..03 视觉深化**：3D 楼宇可视化（Three.js）配色接品牌令牌 / 登录页-后台过渡动效统一 / 打印导出样式品牌化
- **Tag 默认配方收紧**：`#B88850 on #FEF3C7` 实测约 2.8:1（低于 AA 4.5:1）—— 当前为装饰性品牌锚点（orchestrator 锁定决策），收紧候选 `#905D00` on `#FEF3C7` 5.03:1 已实测达标
- **index.css 叠层规则去重**：4 处 `[data-color-mode="dark"] .ant-table-thead` 规则（L931/L2343/L3844/L4642）与多组 dark `.ant-tag` 规则叠层，后置规则胜出 —— 当前渲染均品牌族内但建议合并去重
- **Phase 53 网络设备端口写 UI 测试修复**：5 个 vitest 失败（BulkWriteDrawer / ports/index 异步 timing 与 mock），与 v1.22 正交

**Refs 关联**：

- 素材：`655aa291-9bfe-4e94-ad5d-b3c8b2d24984/brand-spec.md`（像素实测 + WCAG 验证）、`admin-design-plan.md`（系统现状侦察 + 缺口定位）
- Phase 64 SUMMARY：`D:\code\ClaudeCode\guoguo\.planning\phases\64-brand-token-layer-and-contrast-verification\64-01-SUMMARY.md`
- Phase 65 SUMMARY：`.planning\phases\65-theme-system-consolidation\65-01-SUMMARY.md`
- Phase 66 SUMMARY：`66-component-styles-and-hardcoded-color-scan\66-01-SUMMARY.md`
- Phase 67 SUMMARY：`67-build-regression-and-visual-confirmation\67-01-SUMMARY.md`

---

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

## v1.26 后端测试覆盖率优秀 (Backend Test Coverage Excellence) — ✅ SHIPPED 2026-08-22

**Phases**: 4 (Phases 71-74) | **Plans**: 34 (11 规划 + 1 escalation 74-12 + 历史未入库补齐) | **Tasks**: ~70+ | **Timeline**: 2026-08-20 → 2026-08-22 (2 days)

**Delivered**: 后端测试覆盖率从 12.8% → **55.5%** (ratchet chain 12.8 → 21.5 → 25.9 → 55.5; +42.76pp / 4.34×),0% 包从 33 → **5** (全部入口/装配/生成代码),4 层 enforcement gate (weighted-avg + P1 8 包 ≥70% + P2 10 包 floor + PR diff coverage ≥80%) 全部 CI 生效;业务代码变更 0(D-12 STRICT 全程,~40 个 `*_test.go` 文件)。

**Key Accomplishments**:

1. ✅ **Phase 71 gate 基线** — `.coverage-threshold` (12.8) + `check-coverage.sh` (bash + awk,零依赖,D-01) + `ci.yml` Coverage gate step + `coverage-baseline.md` 起点 row;CI backend Coverage gate PASSED on runs 32382995316 + 32384622877;Phase 71 PROTECT-FIX pkg/cache TestAsyncRetryWorker_Enqueue (5ead742 guard) 修复

2. ✅ **Phase 72 P0 核心 13 plans** — `api/v1/workorder 75.4%` + `api/v1/monitor 71.2%` + `api/v1/scheduler 85.5%` + `services/workorder 73.7%` (CORE-01..05 ≥70%); `services/system 53.5%` + `api/v1/system 35.4%` 不足 70% per-sub-module 留 Phase 74 处理;weighted avg 12.8 → 21.5;0%-pkg 33 → 31

3. ✅ **Phase 73 P1 重要 5 plans** — P1 8 包全部 ≥70%(D-04/D-10 strict):`api/v1/duty 83.0%` + `api/v1/knowledge 84.2%` + `api/v1/rpa 79.2%` (含公开路由 3 endpoint per D-04)+ `api/v1/vdi 76.2%` + `services/duty 95.6%` + `services/knowledge 95.3%` + `services/network 92.1%` + `services/monitor 95.3%` (含 oper_log per D-03);check-coverage.sh 扩展 section 3 `p1_package_check` (exit 4);weighted avg 21.5 → 25.9;0%-pkg 31 → 22

4. ✅ **Phase 74 P2 增量 + diff coverage 收口** — 11 plans + 74-12 escalation:操作类 6 包(operations 72.30/asset 84.52/network 75.34/rpa 86.11/vdi 85.09/utils 95.10/scheduler 89.82)全部 ≥70%;`internal/services/rpa` 跨 SCALE-01 + SCALE-02 双重门;`pkg/{response 96.1, ldaputils 97.0, captcha 87.5, time 85.7, gormutil 63.4, query 67.6, logger 64.6}` 高性价比工具包选择性提升;PR diff coverage ≥80% gate CI-live (74-10, D-14 自实现,market 上 gocover-diff / ory/xcoverage-action 均不可达);check-coverage.sh 扩展 section 4 `p2_package_check` (exit 5,10 包:70%×7 + UP-ONLY ratcheted×3)

5. ✅ **74-08 + 74-12 escalation gap-closure** — `pkg/middleware 14.3→68.8%` / `pkg/crypto 33.7→71.8%` / `internal/services/scheduler 4.8→89.8%` / `internal/templates 44.4→88.1%` / `internal/core/db 37.5→72.9%` / `internal/core/security 48.9→74.8%` / `internal/utils 54.99→95.10%` / `internal/utils/operlog 91.7→94.4%` / `internal/agent/pkg/retry 0→93.9%` / `internal/services/common 0→100%` / `models/{operations,rpa,system,system/requests} 0→47.8~90.9%` / `internal/api/v1/operations/requests 0→100%`;0%-pkg 12 → **5** (SC-b 达成)

6. ✅ **v1.26 MILESTONE SHIPPED + 原子 ratchet chain** — `.coverage-threshold` 25.9 → 55.5 (commit `1f18e20`, amend `33c8b5c`);7 文件原子 commit (`.coverage-threshold` + `coverage-baseline.md` + `check-coverage.sh` + `STATE.md` + `ROADMAP.md` + `74-11-SUMMARY.md` + `74-MILESTONE-AUDIT.md`);push deferred to orchestrator/user per 73-05 deviation;全仓 `go test -count=1` 65 packages ok / 0 FAIL / 0 panic

**SC 状态**(详见 `.planning/milestones/v1.26-MILESTONE-AUDIT.md`):

- **SC-a** weighted avg ≥70%: ⚠️ **部分达成** 55.56% (差 14.44pp;缺口结构化归因:addomain 2415 stmts LDAP 池 / operations 3714 stmts Excel+geocoding HTTP / scrapligo SSH / Core.Init 全链 / agent 子进程 server / services/system 53.5% / api/v1/system 35.4%;P2 3 包 UP-ONLY ratcheted)
- **SC-b** 0%-pkg ≤5: ✅ 达成 (5/5)
- **SC-c** threshold gate active: ✅ 达成 (UP-only ratchet chain)
- **SC-d** full go test 零失败: ✅ 达成 (65 ok / 0 FAIL / 0 panic)
- **SC-e** PR diff coverage ≥80%: ✅ 达成 (74-10 D-14 自实现)

**Requirements 19/19**(15 完全达成 + 4 documented gaps):

- ✅ GOV-01 / GOV-02 / GOV-04 / IMP-01..06 (10/10)
- ⚠️ GOV-03 部分(74-10 自实现,缺市场认可工具支撑,可接受)
- ⚠️ CORE-04 (`api/v1/system 35.4%`) + CORE-06 (`services/system 53.5%`) Phase 72 sub-target 遗留,迁 v1.27 路线
- ⚠️ SCALE-01 P2 7/10 + 3 ratcheted(结构阻塞,UP-only)
- ⚠️ SCALE-02 pkg/ 高性价比工具包(完成 7 包,其余未达 70%)
- ⚠️ SCALE-03 weighted 55.5%(未达 70% 目标,诚实文档化)

**遗留 v1.27 候选**(不阻塞 SHIPPED):

- **SCALE-03 续**:`internal/services/addomain` 2415 stmts 21.78% (LDAP 真实连接池;testcontainer/ldap-go 引入路径);`internal/services/operations` 3714 stmts 61.07% (Excel/Baidu geocoding HTTP;HTTP mock 路径);`internal/device` 1249 stmts 39.07% (scrapligo *network.Driver 不可注入;fakeserver 路径);`internal/core` 754 stmts 38.20% (Core.Init Redis/调度器全链;mini-redis 引入路径,D-12 在 v1.26 禁);`internal/agent/server` 616 stmts 22.08% (子进程 server);`internal/services/system` 3483 stmts 53.5% + `internal/api/v1/system` 3039 stmts 35.4% (Phase 72 遗留 sub-target)
- **集成测试基建**:miniredis / testcontainers-go / gossh-fake 等(D-12 v1.26 范围内禁,候选 v1.27 起点)
- **CORE-04 /06 sub-target + CORE-05 65.0%** 返查
- **15 项 QUIRK** 排查:MemoryCache IncrementBy nil-deref + 静默 0 / ModelExtractor 仅串首锚定 / gmsm sm2.Decrypt 垃圾密文 panic / retry.containsIgnoreCase 恒 true / normalizeParentID nil 塌缩等
- **pkg/cache SCALE-02**:MemoryCache Redis 路径阻塞 24.6% → 70% 候选
- **CI push verification**:`gh run watch` 验证 4 layer gate 在 push 后一致(P1 exit 4 不触发、P2 exit 5 不触发、PR diff exit 1 不触发)
- **/gsd-pr-branch** Phase 74 commit chain(若需要 PR 走 code review)
- **/gsd-cleanup** 历史 v1.26 phase directories 入库 → `milestones/v1.26-phases/`

**Refs 关联**:

- [v1.26-ROADMAP.md](v1.26-ROADMAP.md) — 完整 phase 结构
- [v1.26-MILESTONE-AUDIT.md](v1.26-MILESTONE-AUDIT.md) — SC 逐项证据 + 阻塞清单
- [.planning/phases/71-governance-baseline-and-ci-gate/71-VERIFICATION.md](../phases/71-governance-baseline-and-ci-gate/71-VERIFICATION.md) — Phase 71 终验 passed
- [.planning/coverage-baseline.md](../coverage-baseline.md) — 4 phase ratchet row(起点 → 71 → 72 → 73 → 74)
- [.planning/phases/74-p2-finalize-and-diff-coverage/74-11-SUMMARY.md](../phases/74-p2-finalize-and-diff-coverage/74-11-SUMMARY.md) — Phase 74 final ratchet + v1.26 closeout

