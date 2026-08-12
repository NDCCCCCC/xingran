# Phase 13: 查询层与轨迹 - Context

**Gathered:** 2026-06-13
**Status:** Ready for planning

<domain>
## Phase Boundary

在 Phase 12 已建立的 MAC 历史数据采集和分区表基础上，构建 MAC 地址历史数据的**查询与统计层**，
并提供**轨迹可视化**能力。Phase 13 不再涉及采集层（已稳定），专注于让运维人员能"看到"历史变化。

核心交付:
- 后端: MAC 轨迹查询 (QUERY-02) + 连接时长统计 (QUERY-03) + MAC 厂商识别 (QUERY-04)
- 前端: MAC 轨迹可视化 (UI-03) — ECharts 单 MAC 焦点视图
- 修复 Phase 12 UAT 阻塞项（路由已在 router.go:325 注册，需补 UAT 重测）

Phase 13 不新增"实时 MAC 流"、"异常告警"、"MAC 随机化识别"等能力（已在 Out of Scope 中标注）。

</domain>

<decisions>
## Implementation Decisions

### 1. MAC 轨迹查询策略 (QUERY-02)

- **D-13-1.1**: 使用 PostgreSQL `LAG() OVER (PARTITION BY mac_address ORDER BY first_seen)`
  窗口函数实现轨迹查询，单条 SQL 一次拉取所有记录 + 状态转换标记。
- **D-13-1.2**: 节点粒度 = **聚合后区间**（`first_seen` ~ `last_seen` 合并连续状态），
  复用 Phase 12 D-03（flapping 智能合并）逻辑；保留 `event_type` 作为节点元数据
  用于 hover tooltip。
- **D-13-1.3**: 仅查 `sys_device_mac_history` 表，不与 `sys_device_mac_address`
  当前表 LEFT JOIN（保持逻辑简洁，规避 N+1）。
- **D-13-1.4**: 跨分区由 PostgreSQL 原生分区裁剪自动处理（`LAG()` 透明，
  无需应用层补偿）。

### 2. 停留时长 / 连接时长计算口径 (QUERY-02 / QUERY-03)

- **D-13-2.1**: 停留时长口径 = `last_seen - first_seen`（固化区间）。
  不估算"进行中"状态，避免采集周期魔法数。
- **D-13-2.2**: 单位 = `int64` UTC 秒数，UI 按需格式化为"3 天 2 小时 15 分"。
- **D-13-2.3**: 长期占用阈值 = **可配置**，存 `sys_config.network.mac.history.long_occupancy_threshold_days`，
  默认 `30`。
- **D-13-2.4**: 统计输出 = **明细 + Top-N**（按 MAC 长期占用 Top + 按端口热门连接 Top）。

### 3. OUI 厂商库来源与更新 (QUERY-04)

- **D-13-3.1**: 独立表 `sys_mac_oui_vendor(oui_prefix PRIMARY KEY, vendor_name, updated_at)`，
  启动时检测表空 → 从仓库内嵌 JSON 批量导入。
- **D-13-3.2**: 查询性能 = DB LEFT JOIN + 启动时 L1 (Redis) 缓存，
  Redis 不可用降级 SQL 查询。
- **D-13-3.3**: 数据源 = 仓库内嵌 `configs/oui-vendors.json`（git 版本化），
  启动时检测空表导入；不依赖运行时外部网络。
- **D-13-3.4**: 未知 OUI → `vendor_name = "Unknown Vendor"`，不识别随机化 MAC
  （避免增加 Phase 13 复杂度；MAC 随机化识别属后续里程碑）。

### 4. ECharts 轨迹可视化形态 (UI-03)

- **D-13-4.1**: 展示形态 = **单 MAC 焦点视图**（Gantt 风格，横轴时间 + 纵轴端口）。
- **D-13-4.2**: 事件高亮 = 颜色编码（`appeared`=绿 / `disappeared`=红 / `moved`=黄 /
  `vlan_changed`=蓝）+ tooltip 详情（MAC / 设备 / VLAN / 停留时长）。
- **D-13-4.3**: 跨设备呈现 = 纵轴按设备分组（设备 A 端口组 / 设备 B 端口组），
  MAC 跨设备时在另一设备组中出现新区间。
- **D-13-4.4**: 空状态 = Ant Design `Empty` 组件 + 骨架屏加载 + `Alert` 错误提示。
  不做智能引导（"该 MAC 可能是首次出现"等），保持简单。

### Claude's Discretion

- Phase 12 提到的 MAC 厂商"非阻塞加载"（即 OUI 表加载失败不影响主服务启动）
  由 Claude 决定降级策略，建议：表为空时**警告但继续**，查询时返回 "Unknown Vendor"。
- 轨迹查询的 API 路径（建议 `POST /network/history/trajectory` 与现有 `/history/port`
  `/history/device` 风格保持一致）。
- 统计 API 的具体返回 JSON 结构（明细 + Top-N 两段式）。
- ECharts 组件文件名（建议 `MACTrajectoryChart.tsx` 放 `xingran-react-frontend/src/components/network/`）。

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 需求与里程碑
- `.planning/REQUIREMENTS.md` § v1.5 — QUERY-02/03/04 + UI-03 需求定义
- `.planning/ROADMAP.md` — Phase 13 边界（Phases 12-15 v1.5 MAC地址历史数据管理）

### Phase 12 已锁决策（继承）
- `.planning/phases/12-data-model-integration/12-CONTEXT.md` — D-01 ~ D-10 全部继承
- `.planning/phases/12-data-model-integration/12-RESEARCH.md` — 数据模型与采集集成研究
- `.planning/phases/12-data-model-integration/12-UAT.md` — UAT 已知问题清单
  (cleanup task 未注册 / 路由注册需复查)

### 现有代码（实现基础）
- `internal/models/device_mac_history.go` — 历史表模型（继承使用）
- `internal/services/mac_history_service.go` — 变更检测 + flapping 合并（D-03 复用）
- `internal/services/mac_history_query_service.go` — 已有 `QueryPortHistory` / `QueryDeviceHistory`
  (276 行),Phase 13 新增 `QueryMACTrajectory` / `QueryConnectionStats` / `GetVendor`
- `internal/api/v1/network/mac_history_handler.go` — 已有端点 + TODO `GetStats`
- `internal/api/v1/network/mac_history_router.go` — 路由已注册 `SetupMACHistoryRouter`
- `internal/api/router.go:325` — 路由已在主路由注册（UAT 阻塞项已修复）

### 项目规范
- `CLAUDE.md` § 关键配置 + § Cache System — Redis 键前缀 `xingran:`
- `CLAUDE.md` § Handler-Service Pattern — 标准服务接口模式
- `.planning/codebase/ARCHITECTURE.md` — 分层架构、Core DI、CacheProvider
- `.planning/codebase/STACK.md` — ECharts 6.0 / echarts-for-react / React Query v5.90.12

### 外部参考
- PostgreSQL Window Functions — https://www.postgresql.org/docs/current/tutorial-window.html
- PostgreSQL Partitioning — https://www.postgresql.org/docs/current/ddl-partitioning.html
  (跨分区 LAG() 透明)
- IEEE OUI List — https://standards-oui.ieee.org/oui/oui.csv (Phase 13 不直接拉取,
  仅参考生成 oui-vendors.json)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets (from Phase 12)
- **`DeviceMACHistory` 模型** (`internal/models/device_mac_history.go`) —
  Phase 13 查询直接使用，已含 `event_type` / `first_seen` / `last_seen` / `vlan_id`。
- **`macHistoryServiceImpl.MergeFlappingRecords`** —
  D-13-1.2 区间聚合可复用此 flapping 合并逻辑。
- **`macHistoryQueryService` 接口** (`internal/services/mac_history_query_service.go`) —
  扩展 `QueryMACTrajectory` / `QueryConnectionStats` / `GetVendor` 三个方法。
- **Handler-Service 模式** — 已有 `MACHistoryHandler` 模板，Phase 13 端点直接挂载。
- **`Router` 注册位置** `internal/api/router.go:325` — Phase 13 不需重新注册。

### Established Patterns
- **PostgreSQL 时间分区 + BRIN 索引** — 跨月查询自动分区裁剪。
- **响应包装**: `response.Success()` / `response.Error()` / `response.Page()` —
  列表/分页场景标准用法。
- **Redis 缓存前缀** `xingran:` — OUI L1 缓存键 `mac:vendor:lookup` 自动加前缀。
- **配置存储** `sys_config` 表 — `long_occupancy_threshold_days` 走标准 ConfigService。

### Integration Points
- **前端** `xingran-react-frontend/src/pages/network/mac/index.tsx` —
  现有采集页面，Phase 13 新增"历史轨迹" tab 或独立路由
  `/network/mac/trajectory`。
- **前端 ECharts 集成** — 仓库已有 echarts-for-react 3.0.5 (Phase 30 按需加载已优化)。
- **ECharts 按需加载** — 参考 Phase 30 Plan 02 的 `echarts/charts` 引入模式。

### Known Constraints
- **Phase 12 UAT 阻塞项**: `SetupMACHistoryRouter` 在 `router.go:325` 已注册，
  UAT 报告已陈旧；UAT 需重测或更新文档。
- **Phase 12 UAT MAJOR**: 清理任务未在调度器注册（root_cause 已分析），
  不在 Phase 13 scope，但需在 Phase 13 验证任务时一并验证。

</code_context>

<specifics>
## Specific Ideas

- 轨迹 API 应返回设备名称快照（`device_name_snapshot`）而非关联查询，
  保持与 Phase 12 D-04 一致。
- ECharts Gantt 实现可使用 `custom` series + `dataItem` 渲染区间块，
  参考 echarts-gallery 的 gantt 案例。
- 长期占用 Top-N 排序：`ORDER BY total_duration DESC LIMIT 10`，
  配合 D-13-2.3 配置阈值过滤。
- OUI 表的 `oui_prefix` 应存储为标准化格式（如 `AABBCC` 大写无分隔符），
  与 `DeviceMACAddress.MACAddress` 前 3 字节比较时无需转换。
- 前端 MAC 查询输入框应支持 `AA:BB:CC:DD:EE:FF` / `aabb.ccdd.eeff` /
  `aabb-ccdd-eeff` / `aabbccddeeff` 多格式输入（复用 Phase 12 `normalizeMACAddress`）。

</specifics>

<deferred>
## Deferred Ideas

- **MAC 地址随机化识别** (REQUIREMENTS 列入 v1.7+ ADV-01) —
  Phase 13 不实现，保持 D-13-3.4 简化处理。
- **多 MAC 对比视图** (Area 4 选项 B) — 不在 Phase 13 scope，
  后续如安全审计需求再开新 phase。
- **轨迹回放功能** (REQUIREMENTS v1.7+ ADV-02) — 时间滑块、播放/暂停，
  属于新能力，Phase 13 不做。
- **异常告警通知** (REQUIREMENTS v1.6+ ALERT-01/02) — 频繁移动检测、
  异常时间移动等超出 Phase 13。
- **MAC flapping 检测** (P1-C5 关联) — Phase 13 输出 flapping 计数
  作为统计指标，但不发告警，告警集成后续里程碑。
- **Phase 12 清理任务注册** (UAT MAJOR) — 不属 Phase 13 scope，
  但 Phase 13 验证时需主动验证 `network.mac.history.retention_days`
  配置项和清理任务存在性。

</deferred>

---

*Phase: 13-query-layer-trajectory*
*Context gathered: 2026-06-13*
