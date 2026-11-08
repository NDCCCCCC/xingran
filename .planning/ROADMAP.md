# Roadmap: RuoYi-Next 运维管理系统

## Current Milestone: v1.54 真实代码缺口清偿 (Code Gap Clearance)

**Status:** ✅ SHIPPED 2026-11-09 · INITIATED/ROADMAP CREATED 2026-11-08 | Phases **142-144**（编号延续 v1.53 结束于 141，不重置）| **7 REQ / 3 Phase / 100% 映射 / 100% 交付** | **W1 四 executor 零依赖并行 + Lead 集成**（按模块隔离）| 全量回归闸门全绿（go build / go test ./... / tsc / lint:ci ALL PASS / 全量 vitest / check-permission-codes 双向零漂移 / check-route-registry）

**Goal:** 清偿 `.planning/STATE.md ## Deferred Items` 中 v1.53 收档时显式披露的 4 项真实代码缺口（发现设备真实导入 / 同步漂移告警通道 / 端口写批量物理参数 / 型号级能力矩阵），使「代码可修」残余真正落地为可运行、可测试的产品能力；不触碰 LOCKED 残余、不伪造人工验证闭环。

**事实基线:** `.planning/STATE.md ## Deferred Items`（v1.53 `code_gaps` / `verification_gaps` 两行）｜`.planning/MILESTONES.md:19-24`（v1.53 Known Gaps）｜`.planning/milestones/v1.53-phases/*-SUMMARY.md`（Phase 自报边界）｜3 个并行只读探索 subagent 的代码面事实 + 本仓实证｜`.planning/PROJECT.md ## Current Milestone: v1.54`。

**范围裁定（用户 2026-11-08）:** 仅 4 项纯代码缺口纳入（拆为 7 REQ）；验证类 3 项 + FIX-04 真机逐型号取证 + 79 条人工验证跟踪 + 6 项 LOCKED 残余不纳入（见下方 Known Gaps / `.planning/REQUIREMENTS.md` Future Requirements & Out of Scope）。跳过领域研究，零新依赖。

### Phases

- [x] Phase 142: 发现设备真实导入 (DISC-01/02) — stub → 真实写 `sys_network_device` — **completed 2026-11-09**（`142-SUMMARY.md`）
- [x] Phase 143: 端口写物理参数收口 (PORT-01/02) — 批量物理参数路径 + 型号级 `vendorCapabilities` 规则 — **completed 2026-11-09**（`143-SUMMARY.md`）
- [x] Phase 144: 同步漂移告警通道收口 (ALRT-01/02/03) — 前端告警页 / WS 广播 / 企微钉钉 webhook — **completed 2026-11-09**（`144-01-backend-SUMMARY.md` + `144-02-frontend-SUMMARY.md`；本 phase 拆为「后端」与「前端」两个并行 executor，文件面互斥）

**依赖拓扑:** `142 ∥ 143 ∥ 144`（W1 单波、三者零依赖并行，按模块隔离）；无 W2、无收口 phase。

#### Phase 142: 发现设备真实导入 (DISC-01/02)

- **需求映射:** DISC-01, DISC-02（唯一 `code_gaps` 权威登记项）
- **文件面:** `internal/services/device_discovery_service*.go`（`device_discovery_service.go:509-523` stub → 真实实现 + 新增 service 单测）、`internal/api/v1/network/discovery_handler.go`（缓存失效注入）、`internal/api/v1/network/discovery_*_test.go`（2 个既有测试文件删 stub 断言）、网络模块测试；**无需新 migration**。
- **实现要点:** 读 `sys_device_discovery_result` 按 `ip_address` 去重 → 软删恢复（`Unscoped` 命中同 ip 已软删行则 `deleted_at` 置空）→ 单事务批量写 `sys_network_device`；字段映射、部分成功/失败语义、错误逐层返回。
- **成功标准（可观察）:**
  1. 用户从一次发现任务执行「导入」后，`sys_device_discovery_result` 中按 `ip_address` 去重的设备真实出现在 `sys_network_device`，字段映射与发现结果一致；重复 ip 只产生 1 条网络设备记录（去重可用 SQL `COUNT` 断言）。
  2. 同 ip 已软删的 `sys_network_device` 行被软删恢复（`deleted_at IS NULL`）而非新建重复行；单事务批量写在「中途失败」用例下整批回滚，`sys_network_device` 零残留（SQLite 用例可证伪）。
  3. `ImportDiscoveredDevices` 不再 `return len(deviceIDs), nil`；新增 service 单测覆盖「空结果 / 全重复 / 软删恢复 / 事务回滚 / 错误逐层返回」5 类分支，既有 2 个测试文件的「简化实现」断言已删除并改为真实写入断言。
  4. success path 在 `response.Success(...)` 之前调用 `operlog.Record(...)`（`OperType=Import`），`discovery_handler.go` 注入缓存失效使网络设备列表即时可见；错误路径由 handler 用 `response.Error(...)` 包装；SQLite + PG 双方言 `go test ./internal/services/ ./internal/api/v1/network/` 均通过（uuid 双侧 CAST）。

#### Phase 143: 端口写物理参数收口 (PORT-01/02)

- **需求映射:** PORT-01, PORT-02
- **文件面:** `internal/services/portwrite/`（批量物理参数路径 + 型号级 `vendorCapabilities` 规则）及其 `*_test.go`、可能动前端端口写 Modal（`xingran-react-frontend/src/pages/network/ports/` 端口写组件）。与 142/144 不重叠。
- **实现要点:** 沿用 v1.51 Phase 122 批量写实时进度事件，补齐 speed/duplex/mtu 动作的批量分发、逐项结果与错误聚合；`vendorCapabilities` 规则以设备已有型号字段为来源、厂商级默认值回退、未知型号 fail-safe 结论显式登记；全部用非真机 fixture 单测覆盖。
- **成功标准（可观察）:**
  1. 用户在批量端口写弹窗勾选多端口并选择 speed/duplex/mtu 提交后，逐端口下发并复用 v1.51 Phase 122 实时进度事件；`BatchResult` 的 succeeded/failed/skipped 三类逐项聚合，错误可定位到具体端口。
  2. 单条物理参数写（3 厂商 × 3 参数）零回归：`internal/services/portwrite/` 既有单条测试全绿，批量与单条共享同一「校验 / 审计 / 命令分发」函数（无第二套实现，可静态断言）。
  3. 型号级 `vendorCapabilities`：命中型号→按能力矩阵放行/拒绝，未命中→厂商级默认值回退，完全未知→fail-safe（拒绝或降级到安全默认值，结论写在代码注释与测试名）；fixture 单测覆盖「命中 / 厂商回退 / 未知 fail-safe / 型号字段缺失」全部分支，不依赖真机。
  4. 前端端口写 Modal 按型号能力约束禁用不支持的动作或给出提示，API 走 `@/lib/api.ts` wrapped functions；`tsc` / `lint:ci` / port-write 相关 `vitest` 全绿、vendor-react gzip 零回归。
  5. `go build ./...` + `go test ./internal/services/portwrite/` 全绿。

#### Phase 144: 同步漂移告警通道收口 (ALRT-01/02/03)

- **需求映射:** ALRT-01, ALRT-02, ALRT-03
- **文件面:** `internal/services/addomain/ad_sync_drift_alert_*.go`（FIX-11 后端服务已存在）、`internal/scheduler/ad_sync_tasks.go`、`internal/websocket/notice_hub.go`、`internal/api/v1/system/ad_drift_alert_*.go`（新）、`internal/api/router.go`（**只此 phase 动**）、`internal/core/db/migrations/migration_238_*.go`（新，菜单/权限/seed，**只此 phase 动**）、`xingran-react-frontend/src/{lib,store,types,pages/system/ad-drift-alert,constants}`。与 142/143 不重叠。
- **实现要点:** 补齐 FIX-11 自报边界——后端查询/配置端点 + WS 广播 + 前端告警页 + 企微/钉钉 webhook（凭证脱敏、失败可观测、不阻塞主流程）。
- **成功标准（可观察）:**
  1. 用户在「同步漂移告警」页面看到告警列表（分页/筛选/详情/已读），数据来自新端点（如 `POST /system/ad-drift-alert/list`）；已读状态切换持久化、刷新后保留。
  2. 漂移告警产生时 `internal/websocket/notice_hub.go` 复用现有广播 API 向在线客户端推送，前端 store 订阅后无需刷新即实时插入列表（WS 消息体可断言）；WS 广播失败不阻塞主同步流程。
  3. 企微/钉钉 webhook 通道：URL + 启用开关在后端可查询/配置，推送在非阻塞 goroutine 执行、失败可观测（日志/计数）；`operlog.RecordWithBody` 把 webhook URL/secret 脱敏为 `******`，单测断言脱敏且明文 URL 不落日志。
  4. `migration_238_*` 播种菜单/权限码/seed 且 SQLite+PG 双方言幂等，`internal/api/router.go` 注册新路由；受限角色无权限码→403、授权后→200（handler/fixture 断言，不依赖生产库）。
  5. 前端页面用 `@/lib/api.ts` wrapped functions（禁止裸 axios）+ Zustand store + `src/types` 类型；`go build ./...`、`go test ./internal/services/addomain/ ./internal/api/v1/system/ ./internal/websocket/`、`tsc` / `lint:ci` / `vitest` 全绿。

### 并行冲突面（W1 并行许可凭证 · 函数级归属表）

沿用 v1.53 惯例：W1 并行前以**函数级归属表**登记每个 phase 独占的写集合与顶层符号，确认两两交集为空；共享只读基座（`pkg/response`、`internal/utils/operlog`、`internal/models`、`internal/core`）只 import 不修改。

| Phase | 独占文件面（写集合） | 函数级归属（独占符号） | 与其它 phase 交集 |
|-------|----------------------|------------------------|-------------------|
| 142 | `internal/services/device_discovery_service*.go`、`internal/api/v1/network/discovery_*`、网络模块测试 | `ImportDiscoveredDevices`、`DeviceDiscoveryService` 导入方法、`DiscoveryHandler` 缓存失效注入 | ∅ |
| 143 | `internal/services/portwrite/*` + 其 `*_test.go`、前端端口写 Modal | `BatchWritePorts` 物理参数分发、`vendorCapabilities` 规则表、型号能力判定函数 | ∅ |
| 144 | `internal/services/addomain/ad_sync_drift_alert_*`、`internal/scheduler/ad_sync_tasks.go`、`internal/websocket/notice_hub.go`、`internal/api/v1/system/ad_drift_alert_*`(新)、`internal/api/router.go`、`internal/core/db/migrations/migration_238_*`(新)、`xingran-react-frontend/src/{lib,store,types,pages/system/ad-drift-alert,constants}` | `AdDriftAlertHandler` / `SetupAdDriftAlertRouter`、`notice_hub` 漂移广播函数、webhook 推送函数、migration 238 `Up()` | ∅ |

**W1 并行许可凭证:** 三者写集合两两交集 = ∅，按模块（发现导入 / 端口写 / 告警通道）隔离，无共享可变文件，故 `142 ∥ 143 ∥ 144` 同波并行成立。两处全局唯一归属硬登记：**`internal/api/router.go` 与 `internal/core/db/migrations/migration_238_*` 仅归 Phase 144**，142/143 不得触碰；若 `plan-phase` 实测出现跨 phase 交集（如前端 `src/lib/api.ts` barrel 同时被 143/144 改导出），须补函数级坐标并把该文件降级串行或拆 commit，否则维持并行判定。

### 关键约束（全局硬约束，沿用 v1.53 惯例）

1. **Handler-Service 分层**：service 返回 `error`；handler 用 `response.Success()` / `response.Error()` 包装，禁止裸 `c.JSON()`；`c.Request.Context()` 逐层透传。
2. **operlog 写操作约定**：所有业务写操作 success path 在 `response.Success(...)` **之前**调用 `operlog.Record(...)`（OperType 按语义，25 常量不可重排）；含 token/secret/webhook 的端点用 `operlog.RecordWithBody` 脱敏（`******`）。
3. **Status `0=正常 1=停用`**；UUID 主键；**SQLite + PG 双方言**（避免 PG 专属 SQL、uuid 双侧 CAST、migration 幂等）。
4. **前端 wrapped API**：`@/lib/api.ts` 的 `post` 等 + Zustand store，禁止裸 axios；`useEffect` 依赖用 primitive/useMemo 稳定。
5. **构建产物只落 `out/`**；Excel 模板动态生成、禁止落盘 `.xlsx`；`data/` 不入库；Go 单测就近 `*_test.go`、前端 `*.test.ts(x)`；零新依赖。
6. **不伪造人工验证闭环**：验证类残余（见 Known Gaps）不建 phase、不勾选、不回写为已完成；本里程碑只交付可代码化能力。
7. **LOCKED 残余不写修复代码**：`.planning/notes/261108-v153-locked-residuals.md` 6 项只维持「接受/仅登记」结论。
8. **79 条人工验证跟踪表保持 0 勾选**：`.planning/notes/261108-human-verification-tracker.md` 按 `notes/261108-uat-writeback-policy.md` 由人工执行后回写。
9. **回归基线**：`go build ./...` + `go test ./...` + `tsc` + `lint:ci` + `vitest` 全绿；提交由 Lead 统一执行（roadmapper 不 git commit）。

### Known Gaps（登记，非阻断）

以下 4 项为**验证类 / 真机类残余**，在 v1.54 显式登记，**不建 phase、不纳入 7 REQ、不伪造闭环**；保持 `.planning/REQUIREMENTS.md` Future Requirements / Out of Scope 与 STATE.md Deferred Items 的原登记状态：

| 项 | 边界 | 处置 |
|----|------|------|
| FIX-01 实机 AD/LDAP UAT（UAC 544 + 登录恢复） | 需生产 AD 实机 | 待人工执行，79 条跟踪表承载 |
| FIX-09 端到端 HTTP P99 压测 | v1.53 仅计算路径近似基线，需真实压测环境 | 待人工执行 |
| FIX-13 生产库 migration 236/237 | 需生产库写授权后受限角色才真正 403→200 | 待 DBA/人工执行，AI 不擅自写生产数据 |
| FIX-04 真机逐型号取证 | 三厂商物理参数逐型号真机结论 | 待现场取证；PORT-02 只做可代码化的规则框架，不替代真机结论 |

> 另：79 条人工验证跟踪项（0 勾选）、6 项 LOCKED 残余、FUTURE-01（P1-5 侧信道告警通道合并评估）继续挂账，见 `.planning/REQUIREMENTS.md` 与 `.planning/notes/261108-*`。
>
> **v1.54 交付后新增登记（2026-11-09，非阻断）**：① 告警「标记已读」未实现（后端 5 端点无该契约、`logs` 来自 `sys_oper_log` 无已读字段）② 企微/钉钉专用 markdown + 加签（路线 B）未做，当前只走路线 A 通用 Webhook ③ `sys_object_id` 已采集未落库、型号 key 仅 `NetworkDevice.Model` ④ 批量物理参数无自动回滚（预览/审计已显式警示）⑤ 发现导入链路顺带隐患（`user_id.(string)` 硬断言 / `persistDiscoveryResults` 未开事务 / `ExecuteDiscovery` 未检查 Save Error）⑥ e2e 归位 ⑦ `RequiresInterfaceView` 与型号画像 `RequiresView` 联动点。逐项见各 phase `*-SUMMARY.md` 与 `STATE.md ## Deferred Items` 的 v1.54 行。
>
> **交付证据（4 份 SUMMARY）**：`.planning/phases/142-disc-01-02/142-SUMMARY.md`、`.planning/phases/143-port-01-02/143-SUMMARY.md`、`.planning/phases/144-alrt-01-02-03/144-01-backend-SUMMARY.md`、`.planning/phases/144-alrt-01-02-03/144-02-frontend-SUMMARY.md`。架构裁定 `.planning/notes/261108-v154-architecture-decisions.md`；集成清单 `.planning/notes/261108-v154-integration-checklist.md`。

---

## Milestones（历史 SHIPPED 列表）

<details>
<summary>历史里程碑 SHIPPED（v1.20 ~ v1.53，折叠；非当前状态）</summary>

- ✅ **v1.53 开放项全量清偿（Open Items Clearance）** — Phases 134-141 (shipped 2026-11-08, 24 REQ / 21 条开放项全覆盖) — [v1.53-ROADMAP.md](milestones/v1.53-ROADMAP.md) · [v1.53-REQUIREMENTS.md](milestones/v1.53-REQUIREMENTS.md)
- ✅ **v1.52 运维域数据权限统一收口（终态）** — Phases 130-133 (shipped 2026-11-07, 7 REQ；archived 2026-11-08) — [v1.52-ROADMAP.md](milestones/v1.52-ROADMAP.md) · [v1.52-REQUIREMENTS.md](milestones/v1.52-REQUIREMENTS.md)
- ✅ **v1.51 端口写生产级操作体验与平台质量收口** — Phases 121-129 (shipped 2026-11-02, 21 REQ) — [v1.51-REQUIREMENTS.md](milestones/v1.51-REQUIREMENTS.md)
- ✅ **v1.25 全站权限治理与样式收敛** — Phases 84-93 (shipped 2026-10-30)
- ✅ **v1.24 视觉回归修复与遗留债务清偿** — Phases 76-83 (shipped 2026-10-28)
- ✅ **v1.23 前端契约全量收尾与遗留清偿** — Phases 69-75 (shipped 2026-10-28)
- ✅ **v1.22 前端组件契约与设计系统对齐** — Phases 63-68.1 (shipped 2026-10-27)
- ✅ **v1.21 后端架构统一与接口解耦** — Phases 57-62 (shipped 2026-10-22)
- ✅ **v1.20 工位管理与空间可视化演进** — Phases 48-56 (shipped 2026-07-10)

</details>

---

## Phases（历史归档，SHIPPED）

<details>
<summary>✅ v1.53 开放项全量清偿 (Phases 134-141) — SHIPPED 2026-11-08</summary>

- [x] Phase 134: AD 同步域 —— enable 方向 + 全量同步节流 + 漂移告警产品化 (FIX-01/10/11) — completed 2026-11-08
- [x] Phase 135: Swagger / 文档产物链路修复与死文件清理 (FIX-02) — completed 2026-11-08
- [x] Phase 136: 接线型端点缺口 —— 用户导出 + 工单评价 + OQ-7 五个读权限码 (FIX-03/06/13) — completed 2026-11-08
- [x] Phase 137: Stub 与双副本漂移治理 —— GetDiscoveryResults + rpa-worker (FIX-07/08) — completed 2026-11-08
- [x] Phase 138: 功能分支补齐 —— scheduler Rotation + 端口写物理参数 + 3 处排序 (FIX-04/05/14) — completed 2026-11-08
- [x] Phase 139: 低优先级技术债 —— 对账 MV 超时配置化 + 登录加密 benchmark 实测 (FIX-09/12) — completed 2026-11-08
- [x] Phase 140: 人工验证债组织 —— 5 份缺失 UAT 清单重建 + 79 条跟踪表 + 回写口径 (VAL-01~07) — completed 2026-11-08
- [x] Phase 141: 登记接受残余 + 临时清单归档 + 权威交回 GSD (ACK-01/02 + GOV-01) — completed 2026-11-08

**交付口径**：18 REQ 交付 + 6 项人工验证跟踪（VAL-01~06 交付物已落盘、实际验证待执行，不伪造闭环）；21/21 条开放项全覆盖。全量回归闸门：go build / go test 全仓 / tsc / lint:ci / vitest 592 文件 4128 用例全绿。完整 phase 详情见 [v1.53-ROADMAP.md](milestones/v1.53-ROADMAP.md) 与 `.planning/milestones/v1.53-phases/`。

**Known Gaps（登记，非阻断）**：人工验证跟踪表 `.planning/notes/261108-human-verification-tracker.md`（79 条全部待执行）；LOCKED 残余结论 `.planning/notes/261108-v153-locked-residuals.md`。

</details>

<details>
<summary>✅ v1.52 运维域数据权限统一收口（终态）(Phases 130-133) — SHIPPED 2026-11-07</summary>

- [x] Phase 130: ScopeProvider 基建 (DS-01/02) — completed 2026-11-07
- [x] Phase 131: 四域读端点 + Excel 导出独立收口 (DS-03/06) — completed 2026-11-07
- [x] Phase 132: 四域写端点收口 (DS-04) — completed 2026-11-07
- [x] Phase 133: 扩展实体 + vdi 合并迁移 + UAT (DS-05/06/07) — completed 2026-11-07

</details>

<details>
<summary>✅ v1.51 端口写生产级操作体验与平台质量收口 (Phases 121-129) — SHIPPED 2026-11-02</summary>

- [x] Phase 121: 端口写 dry-run 预览 + 命令预览 UI — completed 2026-10-31
- [x] Phase 122: 批量写实时进度 SSE/WS — completed 2026-11-01
- [x] Phase 123: 安全三件套 — 快照回滚 + 并发写互斥 + 不可达预检 — completed 2026-11-01
- [x] Phase 124: VLAN 扩展 — trunk native/allowed + hybrid — completed 2026-11-01
- [x] Phase 125: 端口安全 — port-security/sticky/静态 MAC 绑定 — completed 2026-11-01
- [x] Phase 126: Cisco IOS-XE 第 4 厂商 + 跨固件差异矩阵 — completed 2026-11-01
- [x] Phase 127: 审计 UI + 审计治理 + 3 厂商 e2e — completed 2026-11-01
- [x] Phase 128: 平台质量收口 — completed 2026-11-01
- [x] Phase 129: 目录改名 + 双线 planning 合流 + UAT 回补组织 — completed 2026-11-01

</details>

<details>
<summary>✅ v1.25 全站权限治理与样式收敛 (Phases 84-93) — SHIPPED 2026-10-30</summary>

- [x] Phase 84: 权限模型三缺口 — completed 2026-10-28
- [x] Phase 85: 后端动作级严格化 — completed 2026-10-29
- [x] Phase 86: 前端操作入口三态门控铺开 — completed 2026-10-29
- [x] Phase 87: 动作级权限授予闭环 — completed 2026-10-30
- [x] Phase 88: 验证底座与零风险清理 — completed 2026-10-29
- [x] Phase 89: 变量层与明暗色板重构 — completed 2026-10-29
- [x] Phase 90: Input 域交还 variant/token + 圆角统一 — completed 2026-10-30
- [x] Phase 91: Select / InputNumber / Form / Table / Modal 域交还 — completed 2026-10-30
- [x] Phase 92: 暗色 mega-cluster 合并 — completed 2026-10-30
- [x] Phase 93: room-devices 拆分 + 视觉回归验收 — completed 2026-10-30

</details>

<details>
<summary>✅ v1.24 视觉回归修复与遗留债务清偿 (Phases 76-83) — SHIPPED 2026-10-28</summary>

- [x] Phase 76: 双框 P0 修复 + CSS 审计登记册 — completed 2026-10-28
- [x] Phase 77: 双框视觉回归验证 — completed 2026-10-28
- [x] Phase 78: #16 前后端双端闭环 — completed 2026-10-28
- [x] Phase 79: #16 PG 实机验收 — completed 2026-10-28
- [x] Phase 80: 3f130 空因诊断前端接入 — completed 2026-10-28
- [x] Phase 81: F5 刷新落地页修复 — completed 2026-10-28
- [x] Phase 82: lint 清零 + 豁免理由表修复 — completed 2026-10-28
- [x] Phase 83: devices 页拆分 — completed 2026-10-28

</details>

---
*Roadmap v1.54 created: 2026-11-08 by gsd-roadmapper（Phases 142-144，7 REQ / 100% mapped / W1 零依赖并行）；历史里程碑 SHIPPED 列表（v1.53~v1.20）与 v1.53~v1.24 Phases 详情保留为折叠 `<details>`；当前段沿用本仓手工编辑惯例*
