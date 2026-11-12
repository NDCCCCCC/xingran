# Changelog

All notable changes to XingRan-Next are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased] — v1.56 候选

### Added

**数据库全量备份与恢复（PR #83，feat/db-backup-export-restore）**

- 后端 `internal/api/v1/dbbackup/` + `internal/services/dbbackup/`：pg_dump/pg_restore 全库备份导出（异步任务 + taskId 轮询）、上传 zip 导入恢复、定期备份计划（cron + 保留份数）、多存储后端、备份记录下载/删除、pg_dump 能力预检（`GET /system/db-backup/capabilities`）
- 权限模型（85-03 RBAC-01 读写分离）：读组 ANY-of `system:backup:list|export|restore`（含 `GET /:id/download` 下载端点，**无独立 download 权限码**）；写组逐动作严格权限——export→`system:backup:export`、restore→`system:backup:restore`、plan 保存/备份删除→`system:backup:export`
- 前端 `src/pages/system/backup/` + `dbBackupApi`：备份记录列表/统计、立即备份、导入恢复 Modal、计划配置 Tab、按钮门控与后端权限码逐一对齐（下载/删除/保存计划均用 `system:backup:export`，恢复用 `system:backup:restore`）
- 下载/上传端点在 `security.request_encryption.exclude_paths` 中豁免（二进制流不走 SM2+SM4 响应加密）

**系统设置页收敛与低密度页面 Tab 化（PR #84，76fd19ce）**

- 系统设置页目录化平铺：左侧文档目录 + 右侧五分类（邮箱配置 / API 通知配置 / 验证码背景图 / 经理同步 / 备份与恢复）单页锚点闭环；备份与恢复并入设置页 `backup-restore` 分类
- 同步漂移告警并入「AD域控配置」第 3 个 Tab；值班节假日/配置说明并入值班管理、工单分类并入工单管理（Tab 化）；删除 31 个旧壳/残留文件
- 菜单与权限迁移（migration 241/242/243）：隐藏 7 个旧 C 菜单，F 按钮权限原值改挂宿主菜单；权限码零改动
- PageTable 表格区统一包 Card；设置页 Sider 竖线误命中从源头修复（!important 预算不升反降）

**运维须知（PR #84 菜单与权限迁移，发布/回滚必读）**

- **菜单缓存失效**：migration 241/242/243 用裸 `db.Model(...).Updates()` 直接改 `sys_menu`，绕过 menuService，不触发 `internal/services/system/menu_cache_impl.go` 的 `InvalidateMenuCache`（仅菜单写操作 Create/Update/Delete/BatchDelete/UpdateStatus 时调用）；叠加三层缓存——前端内存 `xingran-react-frontend/src/services/cache/TTLMenuCache.ts:12` 5min、sessionStorage `xingran-react-frontend/src/router/DynamicRoutes.tsx:35` 30min（`MENU_CACHE_VERSION=1` 不因数据变更失效）、后端 user-scoped 30min。故发布后最长约 30 分钟用户仍可能看到旧侧边栏/旧菜单；生效方式为等 TTL、菜单管理页写操作触发 `fetchAll(true)` 强刷、或重新登录/401 清缓存，**不需发版**。
- **回滚须知**：`internal/core/db/database.go:1559` 的 `runAndRecord` 不检查版本是否已应用，迁移每次启动重跑，管理员手工把 `visible/status` 改回后下次启动会被再次隐藏；且 `status` 牵动鉴权（`pkg/middleware/permission.go:144-150` 的 `m.status=normal`），回滚必须**同时**处理 visible 与 status，只改 visible 会让旧路由重新出现。
- **隐藏策略边界**：保留菜单行与 `sys_role_menu` 授权行、只改 visible/status，不回收授权，可逆；彻底回收（删除重复行/清理授权）需单独评审。

### Changed

- `scripts/scan-routes.cjs`：路由扫描前先剥 Go 注释（字符串字面量保护、等长替换保行号），注释掉的废弃路由不再入库 `scripts/route-registry.json`（本次清理 workorder 5 条 + rpa 3 条等共 9 条注释假路由，总路由 724 → 715）
- 菜单收敛补强（migration 248/249）：248 补 `ops:workorder:category:list` 的 `status=0` F 型授权载体（修工单分类 Tab 并入后非超管读接口恒 403），249 把所有未软删的 `duty/pools` C 行归位 `visible=0/status=1`（消除 228 历史激活与存量重复行造成的双入口）；同步修订 228 不再激活值班池、243 批量归位全部重复行并修正隐藏判定注释，`registry.go` 补登记 244-249（清偿 `MissingRecords` 对账欠账）。两者均为幂等、双方言、非阻断迁移，不删菜单行、不回收 `sys_role_menu` 授权。

## [v1.21] - 2026-10-22 — AD 双向同步队列化改造 (AD Two-way Sync Queue)

### Added

- AD 推送队列（`sys_ad_push_queue`）：AD 变更不再直推，用户更新/删除在业务事务内入队异步推送，靠部分唯一索引实现同对象合并语义，队列写失败即整体回滚、不留残留
- 队列排水执行器（ADPushDrainer）：重试退避曲线、六状态跃迁状态机、按 AD DN 分组推送（每组一次账号池故障切换），并由 cron 定时任务常态化执行
- 全量同步末尾对 AD 端已消失的用户/OU/计算机按类别自动软删除，带 LDAP 0 条拒绝 + 拟删超 50% 拒绝双闸门保护

### Fixed

- 软删除用户在登录或批量同步时被误恢复：登录路径与批量同步均增加 DN 身份校验，被拒项计入 Skipped 并绕开降级管道
- 软删行唯一键冲突导致同步失败：OU/Group/User/Computer 四类 upsert 支持软删行复活
- AD 组成员同步半途失败导致永久丢成员：成员重建包进事务、重放幂等

## [v1.22] - 2026-10-27 — 前端公用组件整治 (Frontend Shared Component Governance)

### Added

- 列表契约组件四件套：PageTable（受控分页表格）、SearchForm（schema 驱动搜索区）、FormModal（防重复提交+脏检查）、RemoteSelect（防抖+竞态守卫），并以 system/post 双试点起步
- 统一输入契约组件 AppInput / AppSelect（默认交互下沉、逃生口保留），eslint 双闸拦截裸组件
- 危险操作唯一入口 ConfirmDelete（声明式 + helper + 批量按钮三形态），统一「删除后不可恢复」文案

### Fixed

- 51 个列表页分页口径错误（总条数被算成页数）全线修正；备份/发现页翻页断链接通恢复
- 批量删除补二次确认、提交中禁止取消弹窗、假入口（假编辑/假下载/空 Tab）删除

### Changed

- 存量页面分批迁移到契约组件（discoveries/backups/devices/cache 等四页净减约千行）；状态色、日期格式、裸 hex 颜色全站收敛并建立豁免登记制

## [v1.23] - 2026-10-28 — 前端契约全量收尾与遗留清偿 (Contract Rollout Completion)

### Added

- AppTextArea 契约组件落地并全量接入，文本输入框支持一键清空（用户主诉闭环）

### Fixed

- 分页状态恢复 bug（PageTable 挂载硬编码回写根因）实机复验通过
- 7 项历史 debug 遗留全部终结（Cisco MAC 采集、双执行、历史查询归一化、scrapligo、空因诊断端点等）

### Changed

- 契约组件六波迁移 121 个文件：裸组件归零、豁免清零；lint 债大幅清偿（eslint no-explicit-any 958→0、warnings 1366→34）并上线预算断言门禁防回退

## [v1.24] - 2026-10-28 — 视觉回归修复与遗留债务清偿 (Visual Regression Fixes)

### Fixed

- 全站文本输入框双重外框（用户主诉，309 个输入框受影响）根因修复，双框元素 3→0，错误态恢复品牌红
- 行操作权限越权（只读角色调用删除接口返回 200 且真删数据）：新增严格权限匹配中间件 + 前端三态门控 + 权限码双向校验
- F5 刷新后回退到 /dashboard 的问题（含回归测试红绿验证）

### Changed

- devices 页拆分重构（714→306 行）；eslint 存量 51 项清零并接入 CI 门禁

## [v1.25] - 2026-10-30 — 全站权限治理与样式收敛 (Site-wide Permission Governance & Style Convergence)

### Changed

- 全站权限治理与样式收敛（10 个 phase / 41 个 plan），后续在 v1.51 作为基线同步收口

## [v1.26] - 2026-08-22 — 后端测试覆盖率提升 (Backend Test Coverage Lift)

### Added

- PR diff 覆盖率 ≥80% 的 CI 门禁（GOV-03）

### Changed

- 后端整体测试覆盖率 25.9% → 55.5%：vdi、rpa、network/system/agent handler、operations、asset、pkg 工具包等多包批量补测

## [v1.27] - 2026-08-29 — 测试与 CI 稳定性治理 (Test & CI Stability)

### Fixed

- 一批 CI flaky 测试根因治理（时区依赖、CPU 差值断言、sqlite 内存库关闭时序等）
- knowledge 查询歧义列、权限测试断言等零散缺陷

### Changed

- 前端组件/页面测试多批次补齐（shared、network、system、workorder 等模块）；里程碑审计与文档债清偿（7 phases 收口）

## [v1.29] - 2026-09-06 — 缺陷复查集中修复 (Recheck Fixes)

### Fixed

- 同设备恢复互斥原子化（部分唯一索引）；崩溃恢复收敛 pending 任务，防止设备永久死锁
- 备份恢复端点权限收口（专用权限子组）；workstation 列表分页上限恢复（修复平面图/3D 视图静默截断）；多个 List 端点分页归一化
- 删除 WebSocket handler 遗留读循环（消除双读者/双写者崩溃风险）；WS origin 校验收紧
- reconciliation 排序死分支、工单吞错上抛、角色批删缓存失效、config_backup 留痕补全等零散缺陷

## [v1.51] - 2026-11-01 — 端口写生产级操作体验与平台质量收口 (Production-grade Port Write)

### Added

- 端口写 dry-run 预览与命令预览 UI：下发前生成将执行的命令序列供人工确认
- 批量端口写实时进度（SSE/WebSocket，清偿 v1.20 遗留的批量路径 BATCH-05）
- critical 操作执行前抓取端口状态快照、失败时自动回滚恢复
- 多用户并发写互斥（同端口同一时刻仅允许一个写会话）与设备不可达预检
- 能力扩展：trunk/hybrid VLAN 修改、端口安全（port-security / sticky MAC / 静态绑定）、Cisco IOS-XE/XR 第 4 厂商支持、端口写审计明细可视化

### Changed

- 平台技术债同步清偿（addomain 等包测试补齐、覆盖率阈值恢复）

## [v1.51.1] - 2026-11-02

### Fixed

- 锐捷 TrunkSetHybrid 命令渲染修正与 port_binding 参数顺序修复

## [v1.51.2] - 2026-11-02

### Changed

- 前端目录由 `ruoyi-react-frontend` 改名为 `xingran-react-frontend`

## [v1.52] - 2026-11-08 — 运维域数据权限统一收口 (Data Scope Unification)

### Added

- 数据权限单一真实源 ScopeProvider：可见部门集统一实现并注入运维域 26 处挂载；20 个读端点统一可见集过滤（缓存 key 增加 scope 维度），四域写端点越界显式 403、批量按可见集裁剪，Excel 导入逐行校验（PR #76 / milestone v1.52）
- xingran-cli 支持 Excel 导入导出（8 类实体 × import/export/template）、Stdin 管道流式输入与 `-o` 多格式输出（table/csv/yaml）、命令树扩展至 12 模块 551 端点（PR #71 / #73 / #68）
- 工位导入增强：表头别名识别、引用解析分批、状态联动保留（PR #69）

### Changed

- CI 耗时大幅缩短（后端 8m→4m，消除前端 15m 超时）（PR #74）

### Fixed

- xingran-cli `--web` 交互登录三处缺陷修复并支持自定义 token 路径（PR #70）
- 工位导入跳过引用关联的问题修复（PR #75）

> 注：本段除 PR #76（v1.52 里程碑主体）外，还并入 PR #67~#75（2026-11-03 至 2026-11-07 合并、无独立 tag 的提交）。（含至 2026-11-07 的合并）

## [v1.53] - 2026-11-08 — 开放项全量清偿 (Open Items Clearout)

### Fixed

- AD push 队列补 enable 方向与身份校验/双向压制，全量同步增加指数退避节流，常态化同步漂移告警服务
- 用户导出、工单评价端点接通前端入口；权限码 seed 修复五个读权限码恒 403 的问题
- 设备发现结果补持久化查询链路；rpa-worker 双副本去 browser stub 并修复前端副本基线编译
- 值班轮换排序双路径对齐、3 厂商 × 3 参数端口物理参数写闭环、对账物化视图 30s 超时配置化、登录加密 benchmark 空桩补国密实测
- 从 git 历史重建 5 份缺失 UAT 清单并建立 79 条人工验证统一跟踪表

### Changed

- 清理 Swagger 死文件与生成脚本产物路径收敛；全仓临时清单权威引用清理并归档

## [v1.54] - 2026-11-09 — 真实代码缺口清偿 (Real Code Gap Fixes)

### Added

- 同步漂移告警通道收口：后端 5 个端点（状态/日志/配置/更新/触发）+ migration 238 菜单权限，前端菜单驱动告警页面（状态卡/日志/配置/立即检测/WS 实时）
- 端口写物理参数收口：型号级能力矩阵（speed/duplex/mtu，含 Cisco/Maipu 显式不支持）、批量物理参数接入、前端 18 种写动作与证据等级徽标

### Fixed

- 发现设备导入从空实现改为真实导入：按 IP 去重后单事务批量写入网络设备表
- 同步漂移告警改走通知发布链路（修复铃铛收不到）+ WS 全员广播 + 通用 Webhook（凭证脱敏）

## [v1.55] - 2026-11-09 — LOCKED 残余根治 (Locked Residual Root-cause Fixes)

### Fixed

- 数据权限侧信道根治：响应删除规模字段、越界行静默跳过，新增差分攻击测试（N 变化响应逐字节相等）；统一 ScopedDB 读链强制数据权限过滤
- rooms/photos 数据权限「连挂都没挂」的端点补齐挂载；写端点越界几何保存防误删
- 数据库方言机制重构：删除 `RightCast` 字符串方言改语义枚举，两处 raw CTE 收敛为唯一共享合并实现
- 对账缓存失效改广播模式（写后立即一致，修复 `invalidateCache` 空函数）；`pbkdf2SM3` 增加输入校验与黄金向量测试
- operlog e2e 静态门禁 Go 单测化（原 shell 静态段从未真正生效的缺口收口）

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