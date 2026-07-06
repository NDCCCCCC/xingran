---
type: requirements
project: RuoYi-Next
milestone: v1.19
status: planning
created: 2026-07-06
---

# Requirements: v1.19 网络设备写命令 (Network Device Write Operations)

**Project:** RuoYi-Next 网络运维管理系统
**Milestone:** v1.19 网络设备写命令 (Network Device Write Operations) — Planning
**Goal:** 补全网络设备"读+写"闭环——Web 端通过 SSH 直接对目标设备下发端口配置命令（启停/描述/dot1x），成功后立即采集一次端口信息回填审计。

---

## v1.19 Requirements — MVP Scope

### SSH — SSH 写命令基础设施

- [ ] **SSH-01**: 系统能通过 `scrapli/scrapligo v1.3.3` 的 `SendConfig` 接口在配置模式下向 Huawei VRP / H3C Comware / Ruijie RGOS 设备下发命令
- [ ] **SSH-02**: 系统能解析 `SendConfig` 返回结果中的 `% Error:` / `Unrecognized command` / `Illegal` 标记，区分 `transport_error` 与 `device_rejected`，并把错误信息回传前端
- [ ] **SSH-03**: 系统能复用 `PooledConnection.GetConnection` + `defer conn.ReleaseRef()` 模式管理 SSH 连接生命周期（per-port Acquire/Release，不 per-batch）
- [ ] **SSH-04**: 系统能在批处理中使用脱离 HTTP 请求上下文的 30 分钟超时 context（避免 `Core.Close()` 30s 截止陷阱）
- [ ] **SSH-05**: 系统能维护硬编码 `vendor → action → command_sequence` 映射（落地为先，后续 phase 抽象数据库）
- [ ] **SSH-06**: 系统能通过 `DeviceCredentialHelper` 解析 device_id → 凭据（默认 fallback 策略沿用 v1.18）

### PORT — 端口操作（单端口精确）

- [ ] **PORT-01**: 运维能在 Web 端对单个端口执行 `shutdown` 操作（Huawei/H3C: `shutdown`；Ruijie: `shutdown`）
- [ ] **PORT-02**: 运维能在 Web 端对单个端口执行 `undo shutdown` 操作（Huawei/H3C: `undo shutdown`；Ruijie: `no shutdown`）
- [ ] **PORT-03**: 运维能在 Web 端对单个端口设置 `description`（端口描述，≤ 80 字符）
- [ ] **PORT-04**: 运维能在 Web 端对单个端口开启 `dot1x`（Huawei/H3C: `dot1x enable`；Ruijie: `dot1x port-control auto`）
- [ ] **PORT-05**: 运维能在 Web 端对单个端口关闭 `dot1x`（Huawei/H3C: `undo dot1x enable`；Ruijie: `no dot1x port-control`）
- [ ] **PORT-06**: 系统能在写命令执行前读取端口当前 `admin_status` / `dot1x_enabled`，对已处于目标状态的端口提示"无需操作"

### BATCH — 批量操作

- [ ] **BATCH-01**: 运维能在 Web 端一次性选中多端口（同设备）并对所有端口执行同一操作
- [ ] **BATCH-02**: 系统能以串行 + 失败即停（fail-fast）方式执行批量操作，第一条失败立即停止并回传失败点
- [ ] **BATCH-03**: 系统能返回部分执行结果 `{succeeded: [...], failed: [...], skipped: [...]}`，明确每端口状态 + 失败原因
- [ ] **BATCH-04**: 系统能限制批量操作最大端口数为 50（硬上限），超过时前端警告 + 后端拒绝
- [ ] **BATCH-05**: 系统能在批量操作执行期间为前端提供进度反馈（已完成 / 总数 / 当前操作端口）

### AUDIT — 审计

- [ ] **AUDIT-01**: 系统能在每个写操作成功路径末尾、`response.Success()` 之前调用 `operlog.Record(...)` 写入操作日志（module=`端口管理`，OperType 见下表）
- [ ] **AUDIT-02**: 系统能为每条写操作记录关键操作参数：`device_id` / `port_id` / `action` / `description` / `operator` / `result_status`
- [ ] **AUDIT-03**: 系统能新建 `sys_port_write_audit` 表，存储每次写操作的 `before_value` / `after_value` / `command_sent` / `device_response` / `status` / `failure_reason`，支持后续合规追溯（operlog 会被 11 个敏感关键词脱敏，此表为未脱敏真相源）
- [ ] **AUDIT-04**: 系统能在写操作成功 1-2 秒后通过 `DeviceInfoCollectionService.Enqueue(deviceID)` 触发该设备端口信息采集（复用 v1.18 D-09 dedup 防队列重复）

### UI — 前端

- [ ] **UI-01**: 系统能在端口列表 / 端口详情页提供"端口操作"入口（按钮 / 右键菜单），按端口状态显示"启停" / "修改描述" / "dot1x 启停"操作项
- [ ] **UI-02**: 系统能在单端口操作时弹出二次确认 Modal（必须输入"操作原因" 5-200 字符），确认后下发
- [ ] **UI-03**: 系统能在批量操作时弹出 `BulkWriteDrawer`，显示已选端口列表 + 操作类型 + 进度条 + 失败明细
- [ ] **UI-04**: 系统能在操作完成后通过 antd Toast 反馈成功 / 失败，并提供"查看审计日志"跳转链接到 `sys_oper_log` 详情
- [ ] **UI-05**: 系统能在批量操作进行时禁用端口列表的"刷新"按钮（避免与 `Enqueue` 后台采集竞态）
- [ ] **UI-06**: 系统能在 `src/lib/api/networkApi.ts` 提供 5 个 wrapped POST 函数（`writeShutdown` / `writeUndoShutdown` / `writeDescription` / `writeDot1xEnable` / `writeDot1xDisable`）+ 1 个 batch 函数（`batchWritePorts`）

### PERM — 权限 + 菜单

- [ ] **PERM-01**: 系统能在 `pkg/permission/config.go` 新增 `NetworkPortWrite = "network:port:write"` 权限常量（与 `network:port:query` 区分）
- [ ] **PERM-02**: 系统能在所有 6 个写端点（5 个单端口 + 1 个批量）上加 `RequirePermissions(["network:port:write"])` 中间件
- [ ] **PERM-03**: 系统能在 migration 中 seed "端口配置" 子菜单到 "端口管理" 父菜单下，并通过 `GrantNewMenuToRolesHavingParent(db, "端口管理", newMenuID)` 精准授权父已关联角色（避免 antd 父子联动视觉陷阱）

### INFRA — 基础设施

- [ ] **INFRA-01**: 系统能创建 `sys_port_write_audit` 表（migration_2XX），含 `device_id` / `port_id` / `action` / `before_value` / `after_value` / `command_sent` / `device_response` / `status` / `failure_reason` / `operator` / `created_at`，按 `(device_id, port_id, created_at)` 建索引
- [ ] **INFRA-02**: 系统能注册新菜单路由 `/network/ports/write`，前端可通过 `DynamicRoutes` 动态加载（routeGenerator 拼接路径）
- [ ] **INFRA-03**: 系统能定义 `internal/services/portcollection/cache_keys.go` 中新增 `CacheKeyPortWriteResult = "port:write:result:{port_id}"`（如需要缓存单端口最近一次写结果）+ `CacheKeyPortWriteBatch = "port:write:batch:{batch_id}"`（batch 进度缓存）

### CONV — operlog OperType 映射

- [ ] **CONV-01**: 单端口 `shutdown` / `undo shutdown` → `operlog.OperTypeStatus` (值=10，状态变更)
- [ ] **CONV-02**: 单端口 `description` 修改 → `operlog.OperTypeUpdate` (值=2)
- [ ] **CONV-03**: 单端口 `dot1x enable` / `dot1x disable` → `operlog.OperTypeStatus` (值=10)
- [ ] **CONV-04**: 批量端口写 → `operlog.OperTypeBatch` (值=16)，oper_param 记录 `batch_size` / `succeeded_count` / `failed_count` / `action` 汇总

---

## Out of Scope (explicitly excluded from v1.19)

| Feature | Rationale |
|---------|-----------|
| **自动回滚 (Automatic Rollback)** | 反向命令规划失败模式过多；`sys_port_write_audit.before_value` 已支持人工回退 |
| **定时写操作 (Scheduled Writes)** | 夜间运维无响应；如需走工单系统 |
| **多用户并发写仲裁** | operlog 留时间戳，last-write-wins 对 MVP 可接受 |
| **写命令中转重试** | `shutdown` 重试是危险操作 |
| **跨厂商统一命令抽象** | 精度损失；vendor→template map 已锁定 |
| **Cisco 设备支持** | 后续 phase (v1.19.x+) 加入 |
| **端口绑定 / 速率 / 双工 / VLAN 切换** | 后续 phase (v1.19.x+) 加入 |
| **dry-run 模式** | 后续 phase 加入 |
| **命令预览 UI** | 后续 phase 加入 |
| **操作历史查看 UI** | 后续 phase 加入（operlog 后端已记录）|
| **批量冲突自动解决 (skipped array)** | 后续 phase 加入 |
| **设备不可达预检** | 后续 phase 加入 |

---

## Future Requirements (deferred to v1.19.x+ follow-up)

- **FUTURE-01**: 端口绑定（port-security / sticky MAC / 最大 MAC 数）多厂商命令
- **FUTURE-02**: 速率/双工/MTU 调整
- **FUTURE-03**: VLAN 切换（access / trunk / hybrid）
- **FUTURE-04**: dry-run 模式（前端预览命令但不发送）
- **FUTURE-05**: 操作历史查看 UI（点击端口查看历史 operlog 列表）
- **FUTURE-06**: 批量冲突自动解决（已处于目标状态的端口自动 skipped）
- **FUTURE-07**: 设备不可达预检（写命令前 1s ping）
- **FUTURE-08**: Cisco IOS-XE/XR 设备支持
- **FUTURE-09**: 自动回滚策略（critical 操作前快照 + 失败自动 reverse）
- **FUTURE-10**: 多用户并发写互斥（同一端口同时只有 1 个 operator 可写）

---

## Traceability (to be filled by roadmap)

| Requirement | Phase | Plan | Status |
|-------------|-------|------|--------|
| (pending — roadmap will populate) | | | |

---

*Requirements locked: 2026-07-06*
*Status: planning — awaiting roadmap*
*Total: 28 MVP requirements across 7 categories (SSH=6, PORT=6, BATCH=5, AUDIT=4, UI=6, PERM=3, INFRA=3, CONV=4) + 10 future requirements + 12 out-of-scope items*