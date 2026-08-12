# Phase 44: 置信度评分 + IP 段例外 (R3) - Research

**Researched:** 2026-06-28
**Domain:** CIDR 例外规则引擎 + 告警降噪（Go/Gin/GORM + PostgreSQL GiST inet_ops + React/Ant Design）
**Confidence:** HIGH（核心 schema 已在 migration_168 落地；插入点代码已读；GiST inet_ops 是 PG 9.4+ 内置 opclass）

## Summary

Phase 44 (R3) 是 v1.17 资产对账的第三个阶段，目标是通过 **IP 段（CIDR）级别例外规则引擎** 抑制告警风暴，把告警量从 R2 末期下降 ≥60%（ROADMAP success criteria 8）。**命名澄清（关键）**：Phase 44 标题里的"置信度评分"已在 Phase 42（RECON-03, `confidence_score` 字段 + Plan 42-02 Layer 3 引擎）落地，R3 真正交付的是 **IP 段例外规则引擎 + 告警降噪**，planner 不要在 R3 重复实现评分函数。

工程上 R3 的本质是**单点插入 + 单点改造 + 新建 CRUD/工具**：(1) 在 `internal/services/asset/reconciliation_detection.go:DetectLayer3` 循环内、24h 节流 guard（:261）之后、INSERT（:318）之前插入"Layer 3.5 例外匹配"步骤，命中时给 `models.SysDataReconciliation` 填 `ExceptionRuleID` + `AppliedActions` + 调整 `Severity`；(2) 在 `internal/scheduler/reconciliation_tasks.go:cleanupExpiredExceptions` case（:78 placeholder）填软停用真实逻辑；(3) 在 `createWorkorderCritical`/`createWorkorderHigh` 的 SQL 加 `AND 'no_workorder' != ANY(applied_actions)` 条件；(4) 新建例外规则 CRUD handler/service（`/exception-rule/{create,update,delete,test}`）+ admin 页 + Excel 导入导出 + 降噪基线/对比端点；(5) 新建 `migration_174` 补 GiST 索引 + CHECK 约束（migration_168 已建表但缺这两项）。

**核心性能架构（D-R3-A1-03 锁定）**：批量检测循环**不**查 GiST 索引——`DetectLayer3` 入口一次性 `SELECT * FROM sys_reconciliation_exception WHERE is_active=0 AND deleted_at IS NULL` 预加载所有 active 规则到 `[]ExceptionRule` 切片，逐资产用 Go `net.ParseCIDR` + `ipNet.Contains(ip)` 内存匹配（参考 `internal/middleware/apikey.go:110-141` 现成模式）。active 规则量级十几条、资产几千条，循环内零 DB 查询，性能稳定。**GiST inet_ops 索引留给命中测试工具的单点查询**（运维输入 1 个 IP，SQL `WHERE ip_range >> $1::inet` 返回命中规则）。

**R3 不改动 R2 通路**（WS 推送 / SysNotice / resolve API 仍读 `applied_actions` 决定是否执行），只在两个转单 cron 的 SQL 加 `no_workorder != ANY(applied_actions)` 过滤条件。`silence` action 语义统一为"写表但列表隐藏 + 全通路静默"（D-R3-A1-01），异常列表 SQL 默认加 `WHERE NOT ('silence' = ANY(applied_actions))` 兜底。

**Primary recommendation:** 按 2 plans 拆分（与 ROADMAP 一致）—— Plan 44-01 落地"规则引擎 + CRUD + admin 页 + 命中测试工具"（覆盖 EXCEPTION-01/02/04 + SC 1-4/6/7/10），Plan 44-02 落地"Excel 导入导出 + 过期清理 cron 真实实现 + 降噪基线/对比端点 + 转单 SQL 改造"（覆盖 EXCEPTION-03 + SC 5/8/9）。

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions（12 项，verbatim from 44-CONTEXT.md）

**Layer 3.5 拦截语义（Area 1）**

- **D-R3-A1-01（silence 记录形态）**：`silence` action 命中**仍写** `sys_data_reconciliation`（带 `exception_rule_id` + `applied_actions=[silence]`），但**异常列表默认过滤掉 silence 记录**，仅 operlog/审计可查。异常列表是否提供"显示已静默"开关 → 留 planner discretion。
- **D-R3-A1-02（例外过滤执行位置）**：例外过滤**集中在 DetectLayer3 循环内（Layer 3.5）**，一次性匹配后写 `exception_rule_id` + `applied_actions` 到 `sys_data_reconciliation`。下游通路（R2 WS 推送 / SysNotice / 转单 cron）**读 `applied_actions` 决定是否执行**，不各自重查例外表。转单 cron（`createWorkorderCritical`/`createWorkorderHigh`）SQL 加条件：`AND 'no_workorder' != ANY(applied_actions)`。
- **D-R3-A1-03（例外匹配性能架构）**：DetectLayer3 循环**前预加载所有 active 例外规则到内存**，每条资产用 Go `net.ParseCIDR` + `ipNet.Contains` 匹配（参考 `internal/middleware/apikey.go:126` 现成模式）。**GiST 索引留给命中测试工具的单点查询**。循环内零 DB 查询，性能稳定（active 规则十几条，资产几千条）。例外规则变更后，下个 cron 周期重新加载（无缓存陈旧问题）。

**多规则合并语义（Area 2）**

- **D-R3-A2-01（severity_override 多规则冲突）**：多条规则命中同一 (asset, conflict_type) 且各带不同 `severity_override` 时，**取最低（最宽松）严重级**。例：规则A override=low + 规则B override=medium → 最终 `low`。单规则命中时直接用其 override。
- **D-R3-A2-02（skip_severity 语义）**：`skip_severity` = **当前 severity 降一级**（critical→high→medium→low，low 不再降）。仍记录、仍走通路，但按降级后 severity 处理。与 severity_override 协作：**先 skip_severity 降级，再 severity_override 覆盖**（取更宽）。
- **D-R3-A2-03（合并效果可视化）**：命中测试工具 + 规则详情页采用**命中规则列表 + 顶部合并结果卡片**形态。

**作用域与匹配逻辑（Area 3）**

- **D-R3-A3-01（ScopeType 维度 + IP 协作）**：**沿用现有代码 `global/dept/user`** 维度。`global` 规则仅 IP CIDR 匹配即生效；`dept`/`user` 规则需「IP CIDR 命中 **AND** 资产责任人 user_id ∈ 该 dept/user」**双条件**才生效。`dept` scope 是否递归子部门 → 留 planner discretion。
- **D-R3-A3-02（空 conflict_types 语义）**：`conflict_types` 为空数组/null 时，该规则**匹配全部 B-F 冲突类型**。planner 在 service 层 enforce 此语义。
- **D-R3-A3-03（命中测试输入 + dept/user 评估）**：命中测试工具输入 **IP/CIDR（必填）+ 可选 user_id/dept_id**。不填 user/dept 时：`dept`/`user` scope 规则标记"需指定 user/dept 才能评估"，仅 `global` 规则参与合并。

**降噪验证 + 工具（Area 4）**

- **D-R3-A4-01（≥60% 降噪验证方法）**：**基线快照 + 对比端点**。运维在 R3 例外规则生效**前**手动触发"记录基线"，把当前异常总数 / 工单总数 / critical 数按时间窗口快照存 `sys_config`（JSON）。R3 例外生效后 dashboard 加"降噪效果"卡片 + 新增对比端点返回"基线 vs 当前"下降百分比。
- **D-R3-A4-02（Excel 字段映射）**：**逗号分隔 + 名称→UUID 匹配**。列：`name` / `ip_range`(CIDR文本) / `conflict_types`(逗号分隔如 `B,C,D`) / `exception_actions`(逗号分隔如 `no_alert,no_notice`) / `severity_override` / `scope_type` / `scope_name`(部门/用户名称→匹配UUID) / `expires_at`(日期) / `reason`。复用 building 导入的"名称→UUID"解析模式。
- **D-R3-A4-03（过期清理行为）**：**软停用 + 保留外键**。到期 cron（`cleanupExpiredExceptions`）`UPDATE is_active=1`（停用），**不删记录**。历史 `sys_data_reconciliation.exception_rule_id` 仍指向有效（虽停用）记录，**审计链不断**。停用规则在 admin 列表标灰，可重新启用 / 改期。

### Claude's Discretion（R3 plan-phase 自决，verbatim from 44-CONTEXT.md）

- **GiST 索引定义**：`CREATE INDEX ... USING gist (ip_range inet_ops) WHERE is_active=0 AND deleted_at IS NULL` + CHECK 约束（chk_actions / chk_severity_override）实现方式（GORM tag vs SQL migration，参照项目记忆 `xingran-gorm-sql-constraint-naming-conflict`）。
- **dept scope 递归子部门**：参照 `sys_dept` ancestors 递归模式定。
- **"显示已静默"开关**：异常列表是否提供切换显示 silence 记录。
- **CRUD admin 页表单布局**：CIDR 输入 + 冲突类型多选 + actions 多选 + scope 三选 + 有效期 DatePicker 的字段组织。
- **命中测试端点路径**：`POST /asset/reconciliation/exception-rule/test`（queryKeys.matchTest 已注册）。
- **operlog module 常量**：`ModuleReconciliationExceptionRule = "资产对账-例外规则"`（Phase 42 D-16 锁定）。
- **cache 策略**：`CacheKeyReconciliationExceptionRuleList` 等 helper 已在 Phase 42 定义（INFRA-04），R3 service 层接入。
- **IPv4/IPv6 支持**：`net.ParseCIDR` 原生支持双栈，无需额外处理。
- **降噪基线快照存储**：`sys_config` JSON vs 独立表的取舍。
- **降级后 severity 的 SLA 联动**：与 R2 D-A2-03 SLA 分级配合。

### Deferred Ideas (OUT OF SCOPE，verbatim from 44-CONTEXT.md)

- **R4（Phase 45）** — 工位详情页 HealthCard/HealthBadge/ReconciliationDrawer 组件、资产详情摘要块、HealthScore 函数（0-100）、跨模块调用 N+1 优化、抽屉"申请例外"按钮预填 IP/类型。
- **R5（Phase 46, 可选）** — 高置信度修复建议（confidence ≥0.9）、人工确认 UI、一键回滚、误修复监控。
- **R3 显式不做**：钉钉/邮件告警通道；例外规则版本历史/审计回溯；例外规则批量启用/停用；例外规则导入预览/dry-run。
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| EXCEPTION-01 | 系统能按 CIDR 格式（如 `192.168.0.0/16`）配置 IP 段级别例外规则 | §Architecture Patterns / Pattern 1 (Layer 3.5 内存匹配) + §Code Examples (CIDR 匹配函数) + migration_174 GiST 索引（命中测试工具用） |
| EXCEPTION-02 | 系统能配置 5 种 actions 组合（no_alert / no_notice / no_workorder / skip_severity / silence），多条规则取并集 | §Architecture Patterns / Pattern 2 (多规则合并算法) + §Code Examples (mergeActions 函数) |
| EXCEPTION-03 | 系统能在例外规则中设定有效期（expires_at），到期自动停用 | §Common Pitfalls / Pitfall 4 (软停用幂等) + §Code Examples (cleanupExpiredExceptions cron) |
| EXCEPTION-04 | 系统能提供"命中测试"工具，输入 IP/CIDR 实时返回命中规则与合并效果 | §Architecture Patterns / Pattern 3 (命中测试端点) + §Code Examples (GiST `>>` 单点查询 SQL) |

附加 success criteria（ROADMAP Phase 44）覆盖范围：
- SC 1（CIDR 校验 + GiST）→ migration_174 + service 校验 `net.ParseCIDR`
- SC 2（CRUD + operlog）→ handler/service 新建 + `ModuleReconciliationExceptionRule` 常量
- SC 3（5 actions 多选组合）→ service 层 enforce `chk_actions` CHECK 约束白名单
- SC 4（多规则并集可视化）→ 命中测试合并卡片（D-R3-A2-03）
- SC 5（expires_at 自动停用 cron 每日清理）→ `cleanupExpiredExceptions` 真实实现 + `sys_job` cron `0 0 3 * * *`
- SC 6（命中测试工具）→ `/exception-rule/test` 端点（D-R3-A3-03）
- SC 7（生效统计）→ `ExceptionRuleStats` 端点已实现（Phase 42），R3 数据接入自动生效
- SC 8（告警量下降 ≥60%）→ 降噪基线/对比端点（D-R3-A4-01）
- SC 9（Excel 导入导出）→ `ExcelConfigs["reconciliationExceptionRule"]` 新增条目
- SC 10（命中例外仍记录 + 写 exception_rule_id + applied_actions）→ D-R3-A1-02 Layer 3.5 单一真相源
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| CIDR 例外规则批量匹配（每条资产） | API / Backend（`reconciliation_detection.go` Layer 3.5 内存匹配） | — | 在 cron 触发的检测循环内执行，零 DB 查询保证性能（D-R3-A1-03） |
| CIDR 例外规则单点命中测试（运维输入 1 IP） | API / Backend（命中测试端点 + PG GiST `>>` 查询） | Database（GiST inet_ops 索引加速） | 单点查询走 SQL `ip_range >> $1::inet`，GiST 索引加速（D-R3-A1-03） |
| 例外规则 CRUD | API / Backend（handler-service 双层） | Database（GORM 写 sys_reconciliation_exception + operlog） | 标准项目模式（CLAUDE.md Handler-Service Pattern 强约束） |
| 5 actions 合并算法 | API / Backend（纯 Go 函数 `mergeActions`） | — | 纯函数无副作用，单测易写（D-R3-A2-01/02） |
| 过期例外软停用 | API / Backend（`reconciliation_tasks.go` cron case） | Database（UPDATE is_active=1） | 单 taskType `"reconciliation"` 分发（D-R3-A4-03） |
| 转单 cron SQL 改造（加 no_workorder 过滤） | API / Backend（`reconciliation_tasks.go:createWorkorderCritical/High`） | Database（`applied_actions TEXT[]` + `!= ANY()`） | R2 已有通路，R3 仅加 WHERE 条件（D-R3-A1-02） |
| 降噪基线快照存取 | API / Backend（`config_service.GetByKey` + Create/Update） | Database（sys_config JSON） | 复用 Phase 42 config seed 基建（D-R3-A4-01） |
| silence 默认过滤 | API / Backend（`reconciliation_service.go:ListExceptions` SQL） | — | 在异常列表 SQL 加 `WHERE NOT ('silence' = ANY(applied_actions))`（D-R3-A1-01） |
| 例外规则 admin 页（CRUD UI） | Frontend Server（React + Ant Design） | API / Backend（CRUD 端点） | 标准表单 + 表格组件，复用现有 asset/reconciliation 前端结构 |
| Excel 导入导出 | API / Backend（`excel_config.go` + `excel_service.ImportData`） | Database（sys_reconciliation_exception upsert + sys_dept/sys_user 名称→UUID 解析） | 复用 building/workstation 导入模式（D-R3-A4-02） |
| GiST 索引 + CHECK 约束 | Database（migration_174 SQL） | — | 项目无 GiST 先例，必须纯 SQL `migration_NNN_*.go`（项目记忆 `xingran-migrations-no-sql-autoloader`） |

## Standard Stack

### Core（R3 无新增依赖，全部复用现有）

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib `net` | go1.24.5 内置 | `net.ParseCIDR` + `ipNet.Contains` CIDR 内存匹配 | Go 原生支持 IPv4/IPv6 双栈，无外部依赖；项目已在 `internal/middleware/apikey.go:110-141` 验证此模式 `[CITED: internal/middleware/apikey.go]` |
| `github.com/lib/pq` | v1.10.x（已存在） | `pq.StringArray` for `TEXT[]` columns（`exception_actions` / `applied_actions` / `conflict_types`） | reconciliation.go 模型已用 `[CITED: internal/models/reconciliation.go:50,85,88]` |
| GORM | v1.30.5（已存在） | CRUD + Raw SQL（GiST 索引查询走 `db.Raw(...)`） | 项目 ORM 标准 `[CITED: CLAUDE.md Technology Stack]` |
| PostgreSQL GiST `inet_ops` opclass | PG 18 内置 | CIDR 包含查询 `>>` / `<<` 加速 | PostgreSQL 9.4+ 内置 opclass，**无需 CREATE EXTENSION**（不同于 `btree_gist`）`[VERIFIED: PostgreSQL docs + paquier.xyz 9.4 highlight]` |
| Gin + `gin.Context` | v1.10.0（已存在） | HTTP handler + 中间件 | 项目 Web 框架标准 `[CITED: CLAUDE.md]` |
| `github.com/xuri/excelize/v2` | v2.10.0（已存在） | Excel 导入导出 | 项目 Excel 处理标准 `[CITED: CLAUDE.md]` |
| React 19.2 + Ant Design 6.1 + Zustand 5.0 | 已存在 | admin 页 + dashboard 降噪卡片 | 项目前端栈 `[CITED: CLAUDE.md]` |

### Supporting（R3 不引入）

无新增。所有支撑能力（`operlog`、`config_service`、`ReferenceResolver`、`excel_service`、`notice_hub`）均在 Phase 42/43 落地，R3 直接调用。

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Go `net.ParseCIDR` 内存匹配 | PG GiST `>>` SQL 在循环内逐条查询 | 内存匹配胜出：循环零 DB 查询，性能稳定（D-R3-A1-03 锁定）。GiST 留给命中测试单点查询 |
| PG `cidr` 列 + GiST `inet_ops` | `bigint` 存 IP + 区间 B-Tree（first/last IP） | cidr + inet_ops 原生支持 IPv6 + 语义清晰（`>>` 直接表达"包含"），无需自己写转换函数。项目表已用 `cidr` 类型（migration_168） |
| `sys_config` JSON 存降噪基线 | 独立 `sys_reconciliation_baseline` 表 | sys_config JSON 复用 Phase 42 基建 + 读写走现成 `config_service.GetByKey`；基线数据量极小（单 JSON 字符串），独立表过度设计（D-R3-A4-01 + Claude's Discretion） |

**Installation:** R3 无 `go get` / `npm install` 新增依赖。

**Version verification:** 无新增包需验证。PostgreSQL GiST `inet_ops` 是 PG 18 内置 opclass（PG 9.4+ 引入），无需 `CREATE EXTENSION`。`[VERIFIED: web search paquier.xyz "Postgres 9.4 Feature Highlight — GiST operator class for inet and cidr"]`

## Package Legitimacy Audit

> R3 不安装任何新的外部包（全部复用 go.mod / package.json 现有依赖）。Package Legitimacy Gate 协议要求"every phase that installs external packages must run the verification"——R3 跳过此 gate。

| Package | Registry | Age | Downloads | Source Repo | slopcheck | Disposition |
|---------|----------|-----|-----------|-------------|-----------|-------------|
| (无新增包) | — | — | — | — | — | N/A — R3 零新增依赖 |

**Packages removed due to slopcheck [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

*R3 是纯代码/schema/config 修改 phase，所有依赖（Go stdlib `net` / `lib/pq` / GORM / Gin / excelize / React / antd）均已在 Phase 42-43 验证可用。*

## Architecture Patterns

### System Architecture Diagram

```
[cron: reconciliation:detectLayer3 @every 6m]
        │
        ▼
┌──────────────────────────────────────────────────────────┐
│ DetectLayer3 (reconciliation_detection.go:207)            │
│  1. SELECT * FROM reconciliation_normalized               │
│  2. ClassifySignals + ClassifyType (A-F)                  │
│  3. Type A → skipped                                       │
│  4. R2 guard 1: 7d 静默期 (last_resolved_at)               │
│  5. R2 guard 2: 24h 节流 (sys_data_reconciliation COUNT)   │
│  6. ★R3 Layer 3.5 例外匹配 (插入点 :262 前)★               │
│     │                                                      │
│     ▼                                                      │
│  preloadActiveRules() → []ExceptionRule (内存)            │
│     │                                                      │
│     ▼                                                      │
│  matchException(assetIP, assetUserID, conflictType)       │
│   → returns (ruleID, mergedActions, finalSeverity,        │
│              isSilence)                                    │
│     │                                                      │
│     ├─ 无命中: AppliedActions=nil, ExceptionRuleID=nil    │
│     │                                                      │
│     └─ 命中: AppliedActions=mergedActions (UNION)         │
│              ExceptionRuleID=firstMatchedRuleID            │
│              Severity=MIN(skip降级后, override)            │
│  7. INSERT sys_data_reconciliation                         │
│     (含 exception_rule_id + applied_actions + severity)    │
└──────────────────────────────────────────────────────────┘
        │
        ▼ (R2 cron 读 applied_actions 决定行为)
┌──────────────────────────────────────────────────────────┐
│ [cron: createWorkorderCritical @every 2m]                 │
│  SQL 加: AND 'no_workorder' != ANY(applied_actions)       │
│  → 命中 no_workorder 的异常不转单                          │
└──────────────────────────────────────────────────────────┘
        │
        ▼
┌──────────────────────────────────────────────────────────┐
│ 异常列表 / WS / SysNotice (R2 通路)                       │
│  - ListExceptions SQL 加: WHERE NOT ('silence' = ANY(     │
│      applied_actions))  ← silence 默认隐藏                 │
│  - WS 推送 + SysNotice 读 severity 决定 critical 级别     │
│      → skip_severity 降级后 severity 不再触发 critical     │
└──────────────────────────────────────────────────────────┘

[运维 CRUD 例外规则]                       [运维触发"记录基线"]
  POST /exception-rule/create               POST /baseline/snapshot
  POST /exception-rule/test                 (存 sys_config JSON)
        │                                          │
        ▼                                          ▼
  sys_reconciliation_exception              sys_config
  (GiST inet_ops 索引 + CHECK)              (config_key=asset.reconciliation.baseline)

[cron: cleanupExpiredExceptions 0 0 3 * * *]
  UPDATE sys_reconciliation_exception SET is_active=1
  WHERE expires_at < NOW() AND is_active=0 AND deleted_at IS NULL
  (软停用，保留 exception_rule_id 外键链)
```

### Recommended Project Structure

```
internal/
├── api/v1/asset/
│   ├── reconciliation_exception_handler.go    # ★R3 扩展 Create/Update/Delete/Test/CleanupNow/Baseline/Compare handler
│   ├── reconciliation_exception_router.go     # ★R3 扩展 /exception-rule/{create,update,delete,test} + /baseline/{snapshot,compare}
│   └── reconciliation_handler.go              # 加 ModuleReconciliationExceptionRule 常量
├── services/asset/
│   ├── reconciliation_exception.go            # ★R3 扩展 Create/Update/Delete + 校验 + scope 解析
│   ├── reconciliation_exception_matcher.go    # ★R3 新建：纯函数 matchException + mergeActions + preloadActiveRules
│   ├── reconciliation_detection.go            # ★R3 改造：DetectLayer3 循环内插入 Layer 3.5 例外匹配
│   ├── reconciliation_baseline.go             # ★R3 新建：降噪基线 snapshot + compare 服务
│   └── cache_keys.go                          # 已就位（CacheKeyReconciliationExceptionRuleList 等）
├── scheduler/
│   └── reconciliation_tasks.go                # ★R3 改造：cleanupExpiredExceptions case 真实实现 + 转单 SQL 加 no_workorder
├── core/db/migrations/
│   └── migration_174_reconciliation_exception_gist.go  # ★R3 新建：GiST 索引 + CHECK 约束
└── services/operations/
    └── excel_config.go                        # ★R3 新增 "reconciliationExceptionRule" 条目

xingran-react-frontend/src/
├── pages/asset/reconciliation/
│   ├── exception-rules/index.tsx              # ★R3 新建：CRUD admin 页
│   └── exceptions/index.tsx                   # ★R3 改造：silence 默认过滤 + "显示已静默"开关
├── lib/assetApi.ts                            # ★R3 扩展 exceptionRule.{list,create,...,test} + baseline.{snapshot,compare}
└── components/asset/reconciliation/
    ├── ExceptionRuleForm.tsx                  # ★R3 新建：表单（CIDR + 冲突类型多选 + actions 多选 + scope 三选 + DatePicker）
    └── MatchTestPanel.tsx                     # ★R3 新建：命中测试面板（合并卡片 + 命中规则列表）
```

### Pattern 1: Layer 3.5 内存 CIDR 匹配（D-R3-A1-03）

**What:** DetectLayer3 入口预加载 active 规则到内存切片，循环内逐资产用 `net.ParseCIDR` + `ipNet.Contains` 匹配，零 DB 查询。

**When to use:** 批量检测循环（cron 触发，几千资产 × 十几规则）。

**Why not SQL GiST in loop:** 循环内逐条 SQL 查询会 N 次往返（N=资产数），性能差。内存匹配 O(rules × assets)，常数小。

**关键设计：**
- `preloadActiveRules()` 在循环前调用一次，返回 `[]*compiledRule`，每个 `compiledRule` 是 `net.IPNet`（已 ParseCIDR）+ 原始 `models.SysReconciliationException`。
- 解析失败的规则（CIDR 格式错误）logrus.Warnf 跳过，不阻塞检测。
- 规则变更后，下个 cron 周期（`@every 6m`）自动重载，无缓存陈旧。

**Example:**
```go
// Source: D-R3-A1-03 + internal/middleware/apikey.go:110-141 现成模式
type compiledRule struct {
    rule  models.SysReconciliationException
    ipNet *net.IPNet // net.ParseCIDR(rule.IPRange) 预编译
}

func preloadActiveRules(db *gorm.DB) []compiledRule {
    var rules []models.SysReconciliationException
    db.Where("is_active = ? AND deleted_at IS NULL", 0).Find(&rules)
    out := make([]compiledRule, 0, len(rules))
    for _, r := range rules {
        _, ipNet, err := net.ParseCIDR(r.IPRange)
        if err != nil {
            logrus.Warnf("[reconciliation] 例外规则 %s CIDR 解析失败(%s): %v", r.ID, r.IPRange, err)
            continue
        }
        out = append(out, compiledRule{rule: r, ipNet: ipNet})
    }
    return out
}

// matchException 纯函数：返回首条命中的规则 ID + 合并后的 actions + 最终 severity
// 注：D-R3-A2-01 多规则 severity_override 取最低（最宽松），故需遍历全部命中规则
func matchException(rules []compiledRule, assetIP string, assetUserID string, conflictType string) (
    matchedRuleID string, appliedActions pq.StringArray, finalSeverity string, isSilence bool,
) {
    // ... 详见 Pattern 2 mergeActions
}
```

### Pattern 2: 多规则合并算法（D-R3-A2-01/02/03）

**What:** 同一 (asset, conflict_type) 命中多条规则时：`final_actions = UNION(各规则 actions)`；`final_severity = MIN(skip降级后的 severity, 各 override)`；`is_silence = 'silence' ∈ final_actions`。

**When to use:** DetectLayer3 例外匹配步骤 + 命中测试端点。

**降级链（D-R3-A2-02）：** `原始severity --skip_severity--> 降一级 --severity_override--> 取更宽`

**Example:**
```go
// Source: D-R3-A2-01/02 锁定语义
var severityOrder = map[string]int{"low": 0, "medium": 1, "high": 2, "critical": 3}

// applySkipSeverity 当前 severity 降一级（critical→high→medium→low，low 不再降）
func applySkipSeverity(s string) string {
    level := severityOrder[s]
    if level <= 0 { return "low" }
    for k, v := range severityOrder {
        if v == level-1 { return k }
    }
    return s
}

// mergeActions 合并多规则命中结果
// 输入：原始 severity + 全部命中规则
// 输出：合并后的 actions 切片 + 最终 severity + isSilence
func mergeActions(originalSeverity string, matched []compiledRule) (
    actions pq.StringArray, finalSeverity string, isSilence bool,
) {
    // step 1: 是否触发 skip_severity（任一规则含此 action 即触发）
    skipTriggered := false
    for _, r := range matched {
        for _, a := range r.rule.ExceptionActions {
            if a == "skip_severity" { skipTriggered = true; break }
        }
    }
    sev := originalSeverity
    if skipTriggered { sev = applySkipSeverity(sev) }

    // step 2: severity_override 取最低（最宽松）
    for _, r := range matched {
        if r.rule.SeverityOverride != nil {
            ov := *r.rule.SeverityOverride
            if severityOrder[ov] < severityOrder[sev] { sev = ov }
        }
    }
    finalSeverity = sev

    // step 3: actions 取并集（去重）
    seen := map[string]struct{}{}
    for _, r := range matched {
        for _, a := range r.rule.ExceptionActions {
            if _, ok := seen[a]; !ok {
                seen[a] = struct{}{}
                actions = append(actions, a)
            }
        }
    }

    // step 4: isSilence 判定
    for _, a := range actions {
        if a == "silence" { isSilence = true; break }
    }
    return
}
```

### Pattern 3: 命中测试端点（D-R3-A3-03，EXCEPTION-04）

**What:** 运维输入 IP/CIDR（必填）+ 可选 user_id/dept_id，端点返回命中规则列表 + 顶部合并卡片。

**When to use:** SC 6 命中测试工具。

**实现选择：** 用 GiST `>>` SQL 单点查询（D-R3-A1-03 锁定 GiST 留给此场景）。若 active 规则仅十几条，也可走内存匹配（与 DetectLayer3 一致），但 GiST 是 SC 1 验收要求。

**Example:**
```go
// Source: D-R3-A3-03 + PG inet_ops opclass（GiST 索引加速）
// SQL: ip_range >> $1::inet  表示 "ip_range 包含 $1"
type MatchTestResult struct {
    MatchedRules    []models.SysReconciliationException `json:"matchedRules"`
    MergedActions   pq.StringArray                      `json:"mergedActions"`
    FinalSeverity   string                              `json:"finalSeverity"`
    IsSilence       bool                                `json:"isSilence"`
    NeedsUserDept   bool                                `json:"needsUserDept"` // 未指定 user/dept 时为 true（dept/user scope 规则未参与合并）
}

// handler 路径: POST /asset/reconciliation/exception-rule/test
// body: { "ip": "192.168.1.10", "userId": "uuid?", "deptId": "uuid?" }
func (h *ReconciliationExceptionHandler) Test(c *gin.Context) {
    var req struct {
        IP     string `json:"ip" binding:"required"`
        UserID string `json:"userId"`
        DeptID string `json:"deptId"`
    }
    // ...
    result, err := h.service.MatchTest(c.Request.Context(), req.IP, req.UserID, req.DeptID)
    // ...
}
```

### Pattern 4: silence 默认过滤（D-R3-A1-01）

**What:** 异常列表 SQL 默认排除 silence 记录。

**When to use:** `reconciliation_service.go:ListExceptions` 查询路径。

**Example:**
```go
// Source: D-R3-A1-01 锁定
// 在 ListExceptions 基础 query 上追加：
if !params.ShowSilenced {  // 默认 false
    query = query.Where("NOT ('silence' = ANY(sys_data_reconciliation.applied_actions))")
}
// "显示已静默"开关（Claude's Discretion）：ExceptionListParams 加 ShowSilenced bool 字段
```

### Pattern 5: 过期软停用 cron（D-R3-A4-03）

**What:** `cleanupExpiredExceptions` cron 真实实现：UPDATE is_active=1（停用），不删记录。

**When to use:** SC 5 每日清理。

**Example:**
```go
// Source: D-R3-A4-03 + internal/scheduler/reconciliation_tasks.go:78 placeholder
// case "cleanupExpiredExceptions":
result := db.Model(&models.SysReconciliationException{}).
    Where("expires_at IS NOT NULL AND expires_at < NOW() AND is_active = ? AND deleted_at IS NULL", 0).
    Update("is_active", 1)  // 0→1 即 启用→停用（Status Convention）
applogger.Infof("[reconciliation:cleanupExpiredExceptions] 软停用 %d 条过期例外规则", result.RowsAffected)
```

**幂等性：** WHERE 条件含 `is_active=0`（仅启用），重复 cron 调用第二次 rowsAffected=0，幂等。

### Pattern 6: 转单 cron SQL 加 no_workorder 过滤（D-R3-A1-02）

**What:** `createWorkorderCritical`/`createWorkorderHigh` 的 SELECT SQL 加 `AND 'no_workorder' != ANY(applied_actions)`。

**When to use:** R3 改造 R2 转单 cron。

**Why single source of truth:** Layer 3.5 已写入 `applied_actions`，转单 cron 仅读此字段，不重查例外表（D-R3-A1-02）。

**Example:**
```go
// Source: D-R3-A1-02 + internal/scheduler/reconciliation_tasks.go:188-194 现有 SQL
// 原始 SQL（R2）:
//   Where("severity = ? AND deleted_at IS NULL AND resolved_at IS NULL AND workorder_id IS NULL", severity)
// R3 改造:
db.WithContext(ctx).
    Where("severity = ? AND deleted_at IS NULL AND resolved_at IS NULL AND workorder_id IS NULL "+
          "AND 'no_workorder' != ANY(applied_actions)", severity).
    Order("detected_at ASC").Limit(limit).Find(&exceptions)
```

### Pattern 7: 降噪基线 + 对比端点（D-R3-A4-01）

**What:** 运维触发"记录基线"把当前异常总数 / 工单总数 / critical 数存 sys_config JSON；对比端点返回"基线 vs 当前"下降百分比。

**When to use:** SC 8 量化验证。

**Why sys_config JSON:** 复用 Phase 42 config 基建 + `config_service.GetByKey`；基线数据极小（单 JSON 字符串）。

**Example:**
```go
// Source: D-R3-A4-01 + Claude's Discretion（sys_config JSON vs 独立表）
// config_key: asset.reconciliation.baseline
// config_value JSON 结构:
// {
//   "snapshot_at": "2026-06-28T10:00:00+08:00",
//   "total_exceptions": 500,
//   "total_workorders": 120,
//   "critical_exceptions": 30
// }

type BaselineSnapshot struct {
    SnapshotAt          time.Time `json:"snapshot_at"`
    TotalExceptions     int64     `json:"total_exceptions"`
    TotalWorkorders     int64     `json:"total_workorders"`
    CriticalExceptions  int64     `json:"critical_exceptions"`
}

// 对比端点逻辑（独立 COUNT 查询，禁止用 list.length — stat-cards-from-list-length-capped-at-100）
func computeReduction(baseline, current BaselineSnapshot) map[string]float64 {
    return map[string]float64{
        "exceptions_reduction_pct": pct(baseline.TotalExceptions, current.TotalExceptions),
        "workorders_reduction_pct": pct(baseline.TotalWorkorders, current.TotalWorkorders),
        "critical_reduction_pct":   pct(baseline.CriticalExceptions, current.CriticalExceptions),
    }
}
```

### Anti-Patterns to Avoid

- **在 DetectLayer3 循环内查 SQL GiST：** N 次 DB 往返，性能差。必须预加载内存匹配（D-R3-A1-03）。
- **例外过滤分散到下游通路：** WS/SysNotice/转单 cron 各自重查例外表，会出现一致性漂移。集中在 Layer 3.5 一次性写 `applied_actions`（D-R3-A1-02 单一真相源）。
- **silence 完全不写表：** strategy §4.2 字面"不记录"与 D4 审计"命中仍记录"冲突。统一为"写表但列表隐藏 + 全通路静默"（D-R3-A1-01）。
- **过期清理硬删除：** 历史 `sys_data_reconciliation.exception_rule_id` 外键断链。必须软停用 is_active=1（D-R3-A4-03 + AUDIT-02）。
- **GORM tag 定义 CHECK 约束：** GORM v1.30.5 对 CHECK 约束支持不稳定，命名不可控（项目记忆 `xingran-gorm-sql-constraint-naming-conflict`）。必须用纯 SQL `migration_NNN_*.go` + `DO $$ ... ADD CONSTRAINT chk_xxx`。
- **降噪对比用 `list.length`：** MaxPageSize=100 钳制（项目记忆 `stat-cards-from-list-length-capped-at-100`）。必须独立 COUNT 查询。
- **Excel 导入按表头名匹配：** `validateAndParseRow` 按列位置 `row[i]` 取值（项目记忆 `xingran-excel-import-column-position-matching`）。Columns 顺序必须与 Excel 列顺序一一对应。

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| CIDR 格式校验 + IP 包含判断 | 正则 + 手写位运算 | `net.ParseCIDR` + `ipNet.Contains` | Go stdlib 原生支持 IPv4/IPv6 双栈，已在 `internal/middleware/apikey.go:110-141` 验证 `[CITED: internal/middleware/apikey.go]` |
| CIDR 包含 SQL 查询加速 | 自己写 first/last IP 区间表 | PG GiST `inet_ops` opclass（`CREATE INDEX ... USING gist (ip_range inet_ops)`） | PG 9.4+ 内置，原生支持 cidr 类型，SC 1 验收要求 `[VERIFIED: paquier.xyz + PG docs]` |
| 名称→UUID 解析（Excel 导入） | 手写 SELECT 查询 | `operations.ReferenceResolver.ResolveSingle` + `ExcelColumn.Reference` 配置 | 项目已有批量解析基建，支持 `sys_dept.dept_name` / `sys_user.username` `[CITED: internal/services/operations/reference_resolver.go:298-322]` |
| 操作日志 | 手写插入 sys_oper_log | `operlog.Record(c, h.core.OperLogService, h.core.GetDB(), module, operType)` | CLAUDE.md 强制约定 + Phase 34 已落地，11 敏感关键词自动脱敏 `[CITED: CLAUDE.md operlog convention]` |
| 配置读写（降噪基线） | 手写 SELECT/UPDATE sys_config | `config_service.GetByKey(ctx, key)` + `config_service.Create/Update` | Phase 42 基建 `[CITED: internal/services/system/config_service.go:201-211]` |
| 异常列表 silence 过滤 | 在 service 层 if 过滤 | 在 SQL `WHERE NOT ('silence' = ANY(applied_actions))` | PG 原生 array 操作符，分页前过滤避免 MaxPageSize 钳制 |
| 多 actions 数组存储 | JSON 列 | `pq.StringArray` + `TEXT[]` column | 模型已用（reconciliation.go:50/85/88），PG ANY() 操作符原生支持 |
| Excel 模板生成 + 解析 | 手写 | `excel_service.GenerateTemplate` + `ImportData` | Phase 1 落地，已支持 building/floor/workstation/asset `[CITED: internal/services/operations/excel_service.go:238]` |

**Key insight:** R3 的所有"看似需要新建"的能力（CIDR 匹配、名称→UUID 解析、操作日志、配置读写、Excel 处理、PG array 操作）都已有现成基建。R3 的工程量集中在**编排这些现成能力**，而非重新发明。

## Runtime State Inventory

> R3 是新增功能 phase（例外规则引擎 + admin 页 + cron 真实实现），**不涉及 rename/refactor/migration 字符串替换**。但 migration_174 会修改 PG schema（加索引 + 约束），需评估运行时状态影响。

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | `sys_reconciliation_exception` 表已存在（migration_168 建表，Phase 42 R1 seed 几条 global 规则）；migration_174 仅加索引 + 约束，不改数据 | 无数据迁移。migration_174 是 ADD INDEX/CONSTRAINT，PG 会自动校验现有数据是否符合 CHECK 约束（若 seed 数据 actions 含非白名单值会 FATA）|
| Live service config | `sys_job` 表已有"对账-例外规则清理"job（reconciliation_tasks.go:145-151 注册），InvokeTarget=`reconciliation:cleanupExpiredExceptions`，cron=`0 0 3 * * *` | R3 把 case `cleanupExpiredExceptions`（:78 placeholder）填真实实现，无需新建 sys_job 记录 |
| OS-registered state | 无（无 pm2/systemd/launchd 注册引用 reconciliation_exception 字符串） | 无 |
| Secrets/env vars | 无（R3 不引入新 env var，降噪基线存 sys_config 不 env） | 无 |
| Build artifacts | 无（无 egg-info/binaries 引用 reconciliation_exception） | 无 |

**canonical question 回答：** migration_174 落地后，唯一可能持有"旧 schema"缓存的运行时系统是 PG 自身的查询计划缓存（query plan cache）。GiST 索引新建后，PG 优化器会自动选用（ANALYZE 后），无需手动干预。其他运行时系统（cron scheduler、WS hub、admin UI）都通过 GORM/SQL 读取，schema 改动对它们透明。

## Common Pitfalls

### Pitfall 1: GORM tag 定义 CHECK 约束 → 命名不可控

**What goes wrong:** 用 GORM `check:` tag 定义 CHECK 约束，GORM 自动生成约束名（如 `chk_sys_reconciliation_exception_exception_actions`），与项目 `chk_*` 命名规范不一致，且不同 GORM 版本生成规则可能变化。

**Why it happens:** GORM v1.30.5 对 CHECK 约束支持是后加的，命名规则不稳定；项目记忆 `xingran-gorm-sql-constraint-naming-conflict` 已记录类似坑（UNIQUE 约束 PG 自动 `*_key` vs GORM 期望 `uni_*_*`）。

**How to avoid:** migration_174 用**纯 SQL** `DO $$ BEGIN ... IF NOT EXISTS ... EXECUTE 'ALTER TABLE ... ADD CONSTRAINT chk_actions CHECK (...)' ... END$$;`。参照 migration_168:139-156 的 DO$$ 块 + `pg_indexes` 检查模式（CHECK 约束查 `pg_constraint` 而非 `pg_indexes`）。

**Warning signs:** 启动日志出现 `constraint "chk_xxx" already exists` 或 `SQLSTATE 42710`。

### Pitfall 2: Excel 导入按表头名匹配 → 列错位读脏数据

**What goes wrong:** ExcelConfig.Columns 顺序与 Excel 实际列顺序不一致，`validateAndParseRow` 按 `row[i]` 位置取值导致 severity 列读到 actions 值，service 层 `net.ParseCIDR` 失败报 500。

**Why it happens:** 项目记忆 `xingran-excel-import-column-position-matching`：`validateAndParseRow` 按列位置 `row[i]` 取值不按表头名，Header 字段仅用于错误提示。

**How to avoid:** planner 在 Plan 44-02 Task 定义 `reconciliationExceptionRule` ExcelConfig 时，Columns 顺序必须严格匹配模板生成顺序（先 `GenerateTemplate` 导出标准模板，运维按此填）。Columns 顺序：`name` / `ip_range` / `conflict_types` / `exception_actions` / `severity_override` / `scope_type` / `scope_name` / `expires_at` / `reason`（D-R3-A4-02）。

**Warning signs:** 导入报 "CIDR 解析失败" 或 "actions 不在白名单"。

### Pitfall 3: Excel UpsertKey 列漏 DBField → 冲突键失效

**What goes wrong:** `name` 列设 `UpsertKey: true` 但漏配 `DBField: "name"`，`prepareRecordsForUpsert` 跳过该字段，upsert 无冲突键报 "没有有效的冲突键值" 500。

**Why it happens:** 项目记忆 `xingran-excel-import-upsertkey-needs-dbfield`：`prepareRecordsForUpsert` 对无 DBField 字段 continue 跳过不写库。

**How to avoid:** `reconciliationExceptionRule` ExcelConfig 的 `name` 列（UpsertKey=true）必须配 `DBField: "name"`。其他写库字段（ip_range/conflict_types/exception_actions/severity_override/scope_type/scope_id/expires_at/reason）也都配 DBField。

**Warning signs:** 导入报 "没有有效的冲突键值" 或重复导入创建重复记录。

### Pitfall 4: 过期清理硬删除 → 外键断链

**What goes wrong:** `cleanupExpiredExceptions` 写 `db.Delete(...)` 软删除记录，历史 `sys_data_reconciliation.exception_rule_id` 仍指向该 ID，但查询时 `deleted_at IS NULL` 过滤掉，审计链断。

**Why it happens:** strategy §5.2 原始 schema 没明确"过期=软停用 vs 删除"，容易误删。

**How to avoid:** 严格按 D-R3-A4-03：`UPDATE is_active=1`（停用），**不** `db.Delete`。`sys_reconciliation_exception.deleted_at` 仅在 admin 显式删除时填。

**Warning signs:** 审计查询历史异常时 `JOIN sys_reconciliation_exception` 返回空（外键指向已删除记录）。

### Pitfall 5: 降噪对比用 list.length → MaxPageSize=100 钳制

**What goes wrong:** 前端对比端点调 ListExceptions(pageSize=99999) 取 list.length，被 MaxPageSize=100 钳制，返回 100 而非真实总数，下降百分比计算错误。

**Why it happens:** 项目记忆 `stat-cards-from-list-length-capped-at-100`：system 模块 MaxPageSize=100。

**How to avoid:** 降噪对比端点必须独立 COUNT 查询（复用 `reconciliation_statistics.go:Summary` 或新写 COUNT），禁止用 list.length。

**Warning signs:** 下降百分比显示 -0% 或异常正值（基线 500 vs 当前显示 100 而非真实值）。

### Pitfall 6: GORM AutoMigrate 误删 GiST 索引

**What goes wrong:** migration_174 建完 GiST 索引后，下次启动 GORM AutoMigrate 检测到 `SysReconciliationException.IPRange` 的 `gorm:"type:cidr"` tag，可能尝试重建索引或与 GiST 冲突。

**Why it happens:** GORM AutoMigrate 不认识 GiST opclass，项目记忆 `xingran-gorm-sql-constraint-naming-conflict` 记录过类似 AutoMigrate 与显式 SQL 约束的冲突。

**How to avoid:** migration_174 用 `CREATE INDEX IF NOT EXISTS`（幂等），且 GORM AutoMigrate 对已存在表只 ADD COLUMN 不会 DROP INDEX。model tag 保持 `gorm:"type:cidr;not null;column:ip_range"`，**不**加 `gorm:"index"`（避免 GORM 自动建 btree 索引与 GiST 冲突）。GiST 索引完全由 migration_174 SQL 管理。

**Warning signs:** 启动日志出现 `index "idx_exc_active_range" already exists` 或 GORM 尝试 DROP INDEX。

### Pitfall 7: silence 写入但异常列表不过滤 → UI 噪声淹没

**What goes wrong:** Layer 3.5 正确写入 `applied_actions=[silence]`，但异常列表 SQL 忘加 silence 过滤，运维看到一堆 silence 记录，降噪效果感知不到。

**Why it happens:** D-R3-A1-01 silence 语义被误解为"完全不写表"，导致 planner 漏掉"写表但列表过滤"两层语义。

**How to avoid:** 严格按 D-R3-A1-01：silence 命中**仍写表**，但 `ListExceptions` SQL 默认 `WHERE NOT ('silence' = ANY(applied_actions))`。Pattern 4 给出代码。

**Warning signs:** 运维反馈"规则生效了但列表里还是一堆异常"。

### Pitfall 8: severity_override 字典值与 chk_severity_override 不一致

**What goes wrong:** service 层接受 `severity_override: "critical"`（用户输入），但 CHECK 约束 `chk_severity_override CHECK (severity_override IS NULL OR severity_override IN ('low','medium','high'))` 不含 critical，INSERT 报 SQLSTATE 23514。

**Why it happens:** strategy §5.2 原始 CHECK 只允许 low/medium/high（override 是"降级"语义，不能升到 critical），但 service 层校验与 DB CHECK 不同步。

**How to avoid:** service 层 `ValidateSeverityOverride` 函数白名单 = `["low", "medium", "high"]`（与 CHECK 一致），且文档化"override 不能升到 critical"。合并算法 `mergeActions` 已保证取最低（D-R3-A2-01）。

**Warning signs:** 创建规则时报 SQLSTATE 23514 或 service 校验通过但 INSERT 失败。

## Code Examples

### Layer 3.5 插入点（reconciliation_detection.go:262 前）

```go
// Source: D-R3-A1-02 + internal/services/asset/reconciliation_detection.go:225-329 现有循环
// 插入位置：guard 2 (24h 节流) 之后 (:261)、脏数据防御 (:263-271) 之前

// 在 DetectLayer3 函数开头（rows 加载后），预加载 active 规则
activeRules := s.preloadActiveRules()  // 新增方法

// 在循环内 guard 2 之后插入 Layer 3.5：
// === R3 / D-R3-A1-02 Layer 3.5: 例外规则匹配 ===
var exceptionRuleID *string
var appliedActions pq.StringArray
matchedSeverity := severity  // 默认不变

if row.AssetIP != nil && *row.AssetIP != "" {
    assetUserID := ""
    if row.AssetUserID != nil { assetUserID = *row.AssetUserID }
    ruleID, actions, finalSev, _ := matchException(
        activeRules, *row.AssetIP, assetUserID, conflictType,
    )
    if ruleID != "" {
        exceptionRuleID = &ruleID
        appliedActions = actions
        matchedSeverity = finalSev  // 可能因 skip_severity/override 降级
    }
}

// 然后构造 rec 时填入：
rec := &models.SysDataReconciliation{
    // ... 现有字段
    Severity:         matchedSeverity,  // 用 matchedSeverity 替换原 severity
    ExceptionRuleID:  exceptionRuleID,
    AppliedActions:   appliedActions,
    // ...
}
```

### migration_174: GiST 索引 + CHECK 约束

```go
// Source: migration_168:139-156 DO$$ 模式 + PG inet_ops opclass + 项目 chk_* 命名规范
package migrations

import (
    "log"
    "applogger"
    "gorm.io/gorm"
)

func Migrate174ReconciliationExceptionGist(db *gorm.DB) error {
    log.Println("Running migration 174: Phase 44 R3 GiST inet_ops 索引 + CHECK 约束")

    if !isPostgreSQL(db) {
        applogger.Infof("[迁移] GiST/CHECK 跳过(非 PostgreSQL)")
        return nil
    }

    // 1. GiST inet_ops 索引（partial index，仅 active + 未删除）
    // D-R3-A1-03 + Claude's Discretion: WHERE is_active=0 AND deleted_at IS NULL
    // PG 9.4+ 内置 inet_ops opclass，无需 CREATE EXTENSION
    gistIdxSQL := `
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes
        WHERE indexname = 'idx_recon_exc_active_range'
          AND schemaname = 'public'
    ) THEN
        EXECUTE 'CREATE INDEX idx_recon_exc_active_range
                 ON sys_reconciliation_exception USING gist (ip_range inet_ops)
                 WHERE is_active = 0 AND deleted_at IS NULL';
    END IF;
END$$;
`
    if err := db.Exec(gistIdxSQL).Error; err != nil {
        return fmt.Errorf("创建 GiST inet_ops 索引失败: %w", err)
    }

    // 2. CHECK chk_actions: exception_actions 必须是 5 个白名单值的子集
    // PG 数组操作符 <@ 表示"是...的子集"
    chkActionsSQL := `
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'chk_recon_exc_actions'
    ) THEN
        EXECUTE 'ALTER TABLE sys_reconciliation_exception
                 ADD CONSTRAINT chk_recon_exc_actions CHECK (
                     exception_actions <@ ARRAY[''no_alert'',''no_notice'',''no_workorder'',''skip_severity'',''silence'']
                 )';
    END IF;
END$$;
`
    if err := db.Exec(chkActionsSQL).Error; err != nil {
        return fmt.Errorf("创建 chk_recon_exc_actions 失败: %w", err)
    }

    // 3. CHECK chk_severity_override: severity_override 仅允许 low/medium/high（不含 critical）
    chkSevSQL := `
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'chk_recon_exc_severity_override'
    ) THEN
        EXECUTE 'ALTER TABLE sys_reconciliation_exception
                 ADD CONSTRAINT chk_recon_exc_severity_override CHECK (
                     severity_override IS NULL OR severity_override IN (''low'',''medium'',''high'')
                 )';
    END IF;
END$$;
`
    if err := db.Exec(chkSevSQL).Error; err != nil {
        return fmt.Errorf("创建 chk_recon_exc_severity_override 失败: %w", err)
    }

    log.Println("Migration 174 completed: GiST inet_ops + CHECK 约束就位")
    return nil
}
```

**注册到 AutoMigrate：** 在 `internal/core/db/database.go:483`（migration_173 之后）追加：
```go
// Phase 44 R3: sys_reconciliation_exception GiST inet_ops 索引 + CHECK 约束
if err := migrations.Migrate174ReconciliationExceptionGist(d.DB); err != nil {
    applogger.Errorf("Phase 44 R3 reconciliation GiST/CHECK 失败: %v", err)
}
```

### GiST 单点命中查询（命中测试端点 SQL）

```go
// Source: D-R3-A3-03 + PG inet_ops opclass（GiST 索引加速）
// SQL: ip_range >> $1::inet  表示 "ip_range 包含 $1"（>> 是 cidr 包含 inet 操作符）
func (s *reconciliationExceptionServiceImpl) MatchTest(
    ctx context.Context, ip string, userID string, deptID string,
) (*MatchTestResult, error) {
    var matched []models.SysReconciliationException
    // global scope 规则：仅 IP 匹配
    err := s.db.WithContext(ctx).
        Where(`ip_range >> ?::inet AND is_active = ? AND deleted_at IS NULL AND scope_type = ?`,
            ip, 0, "global").
        Find(&matched).Error
    if err != nil { return nil, err }

    // dept/user scope 规则：若指定了 user/dept 才参与（D-R3-A3-03）
    if userID != "" {
        var userScoped []models.SysReconciliationException
        s.db.WithContext(ctx).
            Where(`ip_range >> ?::inet AND is_active = ? AND deleted_at IS NULL AND scope_type = ? AND scope_id = ?`,
                ip, 0, "user", userID).
            Find(&userScoped)
        matched = append(matched, userScoped...)
    }
    // dept scope 递归子部门（参照 sys_dept ancestors 模式）
    if deptID != "" {
        // ... 递归查 deptID 及其子孙部门 ID 列表，再 IN 查询
    }

    // 合并多规则（mergeActions 纯函数，详见 Pattern 2）
    needsUserDept := userID == "" && deptID == ""
    // ... 构造 MatchTestResult 返回
}
```

### silence 列表过滤（ListExceptions 改造）

```go
// Source: D-R3-A1-01 + internal/services/asset/reconciliation_service.go:296-301 基础 query
// 在 ExceptionListParams 加字段：
type ExceptionListParams struct {
    base.BaseListRequest
    // ... 现有字段
    ShowSilenced bool `json:"showSilenced"` // 默认 false：隐藏 silence 记录
}

// 在 ListExceptions 基础 query 追加：
if !params.ShowSilenced {
    query = query.Where("NOT ('silence' = ANY(sys_data_reconciliation.applied_actions))")
}
```

### cleanupExpiredExceptions cron 真实实现

```go
// Source: D-R3-A4-03 + internal/scheduler/reconciliation_tasks.go:78 placeholder
case "cleanupExpiredExceptions":
    result := db.WithContext(ctx).
        Model(&models.SysReconciliationException{}).
        Where("expires_at IS NOT NULL AND expires_at < NOW() AND is_active = ? AND deleted_at IS NULL", 0).
        Update("is_active", 1)  // Status Convention: 0=启用 → 1=停用
    applogger.Infof("[reconciliation:cleanupExpiredExceptions] 软停用 %d 条过期例外规则",
        result.RowsAffected)
    // 幂等：第二次 cron 调用 WHERE is_active=0 已不匹配，rowsAffected=0
```

### 转单 cron SQL 加 no_workorder（R2 改造）

```go
// Source: D-R3-A1-02 + internal/scheduler/reconciliation_tasks.go:188-194
func createWorkorderBySeverity(ctx context.Context, db *gorm.DB, woSvc *asset.ReconciliationWorkorderService, severity string, limit int) error {
    var exceptions []models.SysDataReconciliation
    if err := db.WithContext(ctx).
        Where("severity = ? AND deleted_at IS NULL AND resolved_at IS NULL AND workorder_id IS NULL "+
              "AND 'no_workorder' != ANY(applied_actions)", severity).  // R3 新增条件
        Order("detected_at ASC").Limit(limit).
        Find(&exceptions).Error; err != nil {
        return fmt.Errorf("查询 %s 异常失败: %w", severity, err)
    }
    // ... 其余逻辑不变
}
```

### 例外规则 Excel 配置（D-R3-A4-02）

```go
// Source: D-R3-A4-02 + internal/services/operations/excel_config.go 模式 + 项目记忆 xingran-excel-import-*
"reconciliationExceptionRule": {
    SheetName:     "对账例外规则",
    TableName:     "sys_reconciliation_exception",
    EntityName:    "例外规则",
    HasHeader:     true,
    StartRow:      2,
    CachePatterns: []string{"reconciliation:*"},
    UniqueKeys:    []string{"name"},
    PartialUpdate: true,
    Columns: []ExcelColumn{
        // 顺序严格匹配 Excel 模板列序（项目记忆 xingran-excel-import-column-position-matching）
        {Field: "name", Header: "规则名称", Required: true, MaxLength: 128, UpsertKey: true, DBField: "name"},
        {Field: "ipRange", Header: "IP段(CIDR)", Required: true, DBField: "ip_range"},
        {Field: "conflictTypes", Header: "冲突类型(逗号分隔,空=全部)", DBField: "conflict_types"}, // service 层逗号分隔解析为 pq.StringArray
        {Field: "exceptionActions", Header: "动作(逗号分隔)", Required: true, DBField: "exception_actions"}, // service 层解析
        {Field: "severityOverride", Header: "严重度覆盖(low/medium/high,可空)", DBField: "severity_override"},
        {Field: "scopeType", Header: "范围(global/dept/user)", DBField: "scope_type"},
        // scope_name 按 scope_type 不同走不同 Reference：
        //   - global：留空（不解析）
        //   - dept：Reference "sys_dept.dept_name" → scope_id
        //   - user：Reference "sys_user.username" → scope_id
        // 因 Reference 是静态配置不支持条件分支，scope_name 不配 Reference，
        // 在 service 层 importData hook（excel_config 支持？）按 scope_type 手动解析
        {Field: "scopeName", Header: "范围名称(部门名/用户名,global留空)"}, // 临时字段，无 DBField，service 层解析
        {Field: "expiresAt", Header: "过期时间(可空,YYYY-MM-DD)", DBField: "expires_at"},
        {Field: "reason", Header: "原因(≥10字符)", Required: true, DBField: "reason"},
    },
},
```

**关键约束（项目记忆）：**
- `name` 是 UpsertKey 必须配 `DBField: "name"`（xingran-excel-import-upsertkey-needs-dbfield）
- `scopeName` 是临时字段无 DBField，`prepareRecordsForUpsert` 自动跳过；service 层需在 importData 后处理 hook 把 scope_name 按 scope_type 解析为 scope_id 写库（**planner 注意：现有 excel_service 不支持 per-row hook，可能需要在 R3 加一个轻量的 post-process 步骤，或改用 importData 返回 AffectedKeys 后逐条 UPDATE scope_id**）
- `conflict_types` / `exception_actions` 是 TEXT[] 列，Excel 单元格是逗号分隔字符串，service 层需 StringSlice 转换（excel_config 标准转换可能不支持，planner 需评估）

**建议（Claude's Discretion 边界）：** planner 在 Plan 44-02 Task 评估 `excel_service.ImportData` 是否原生支持 TEXT[] 转换。若不支持，方案 B 是先用 `ImportData` 写入 name/ip_range/severity_override/scope_type/expires_at/reason（标准列），再用 service 层 `AffectedKeys` 后处理逐条 UPDATE conflict_types/exception_actions/scope_id（数组 + UUID 解析）。

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| GORM `check:` tag 定义 CHECK 约束 | 纯 SQL `DO $$ ... ADD CONSTRAINT chk_xxx $$` | 项目记忆 `xingran-gorm-sql-constraint-naming-conflict` 记录后 | 命名可控，避免 GORM 自动命名与项目规范冲突 |
| strategy §4.2 silence = "不记录" | D-R3-A1-01 silence = "写表但列表隐藏 + 全通路静默" | Phase 44 CONTEXT 决策 | 统一 strategy 与 D4 审计要求的字面冲突，审计链不断 |
| strategy §5.2 scope_type = `building/floor` | D-R3-A3-01 scope_type = `global/dept/user`（沿用现有代码） | Phase 44 CONTEXT 决策 | 与"责任人"维度对齐，"某部门所有资产豁免"场景自然 |
| Excel 导入 dry-run 预览 | 复用标准导入流程（不特殊化） | Phase 44 deferred 显式不做 | 减少复杂度，R3 用标准 ImportData 即可 |
| 降噪对比肉眼主观判定 | D-R3-A4-01 基线快照 + 对比端点量化 | Phase 44 CONTEXT 决策 | SC 8 ≥60% 下降可量化验证 |

**Deprecated/outdated:**
- strategy §5.2 原 `chk_actions` / `chk_severity_override` 约束名：R3 重命名为 `chk_recon_exc_actions` / `chk_recon_exc_severity_override`（项目 `chk_<table-suffix>_<field>` 命名规范，参照 legacy SQL `chk_building_org_id_is_uuid` / `chk_credential_type` 模式）。
- strategy §4.2 IP 归属解析顺序（`asset.ip → workstation.ip → network_device.ip`）：Phase 42 D-03 已锁定 R1/R2/R3 都用 `ops_asset.machine_ip` 单值（reconciliation.go:44 `AssetIP *string` + MV `asset_ip = ops_asset.machine_ip`），R3 不做多 IP 解析链。

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | PG GiST `inet_ops` opclass 在 PG 18 内置无需 CREATE EXTENSION | Standard Stack / Don't Hand-Roll / Code Examples | LOW — web search 验证 PG 9.4+ 引入，PG 18 必然包含。migration_174 加 `CREATE EXTENSION` 兜底（IF NOT EXISTS）成本极低 |
| A2 | GORM AutoMigrate 不会 DROP migration_174 创建的 GiST 索引 | Common Pitfalls / Pitfall 6 | MEDIUM — 项目记忆记录过 AutoMigrate 与 SQL 约束冲突。若启动 FATA，需在 database.go AutoMigrate() 加保护（参照 `xingran-gorm-sql-constraint-naming-conflict` 的 cleanupOldConstraints 兜底）|
| A3 | `excel_service.ImportData` 原生支持 TEXT[] 列转换（逗号分隔 → pq.StringArray） | Code Examples / Excel 配置 | MEDIUM — 若不支持，方案 B 是后处理逐条 UPDATE。planner 需在 Plan 44-02 Task 验证 `validateAndParseRow` 对 TEXT[] 列的行为 |
| A4 | `excel_service.ImportData` 支持条件性 Reference 解析（scope_name 按 scope_type 走不同表） | Code Examples / Excel 配置 | HIGH — 现有 Reference 是静态配置 `Reference: "sys_dept.dept_name"`，不支持条件分支。R3 必须 service 层后处理 scope_name → scope_id（按 scope_type 分流 dept/user/global）。**planner 必须在 Plan 44-02 Task 显式实现此 post-process 步骤** |
| A5 | migration_174 注册到 database.go:483 后，AutoMigrate 顺序在 `Migrate168ReconciliationTables` 之后 | Code Examples / migration_174 | LOW — 已确认 database.go:456 已注册 Migrate168，174 在其后追加即可（参照 168-173 链式注册模式） |
| A6 | `sys_dept.ancestors` 字段可用于 dept scope 递归子部门 | Architectural Responsibility Map / Pattern 3 | LOW — 已确认 `internal/models/dept.go:9` 有 `Ancestors string gorm:"size:500"`。递归模式是 `ancestors LIKE 'root_id%'` 或 `FIND_IN_SET`（MySQL），PG 用 `ancestors LIKE '%root_id%'`（Claude's Discretion 留 planner）|
| A7 | Phase 42 R1 seed 的 global 例外规则 actions 都在白名单内（migration_174 CHECK 不会 FATA） | Runtime State Inventory | LOW — 若 seed 数据含非法值，migration_174 会 SQLSTATE 23514。planner 应在 Plan 44-01 Task 跑 migration 前先 SELECT 校验现有 seed 数据 |
| A8 | `operlog.Record` 在例外规则 CRUD handler 可直接调用（无 exclude_paths 冲突） | Standard Stack / Don't Hand-Roll | LOW — operlog.exclude_paths 是 RPA heartbeat 专用（项目记忆 `operlog-exclude-paths.md`），与 `/asset/reconciliation/exception-rule/*` 路径无关 |

## Open Questions (RESOLVED)

> 下列 4 个 OPEN QUESTION 已在 44-01 / 44-02 PLAN.md 中明确解决（plan-checker 迭代 2/3 验证）。每个问题末尾标注 RESOLVED 溯源。

1. **Excel TEXT[] 列原生支持（A3）** — RESOLVED: 见 44-02 Task 4 方案 B（专用 ImportRules handler + service.ImportFromExcel 后处理），不依赖 validateAndParseRow 原生 TEXT[] 支持。
   - What we know: `validateAndParseRow` 按 ExcelColumn 配置转换；Reference 字段走 ReferenceResolver；其他字段按类型推断。
   - What's unclear: TEXT[]（pq.StringArray）列是否原生支持逗号分隔字符串自动转数组。
   - Recommendation: planner 在 Plan 44-02 Task 1 先写一个最小 PoC 测试 `excel_service.ImportData` 对 TEXT[] 列的行为；若不支持，方案 B 后处理 UPDATE。

2. **Excel scope_name 条件性解析（A4）** — RESOLVED: 见 44-02 Task 4 方案 B（ImportFromExcel 返回后逐条按 scope_type 解析 scope_id 并 UPDATE，锁定非 executor discretion）。
   - What we know: 现有 Reference 是静态配置（building 导入固定 Reference `sys_dept.dept_code`）。
   - What's unclear: scope_name 按 scope_type 动态走不同表（dept→sys_dept / user→sys_user / global→留空）是否需 service 层 post-process。
   - Recommendation: planner 在 Plan 44-02 Task 显式实现 post-process：ImportData 返回 AffectedKeys 后，逐条 SELECT name + scope_name + scope_type，按 scope_type 解析 scope_id 并 UPDATE。

3. **dept scope 递归子部门策略（A6 + D-R3-A3-01 Claude's Discretion）** — RESOLVED: 见 44-01 Task 3 WARN-10 锁定（R3 dept scope 不递归，仅匹配 scope_id == deptID 直接部门；R4/R5 再评估）。
   - What we know: sys_dept.ancestors 存逗号分隔祖先 ID（如 `0,root_id,parent_id'`）。
   - What's unclear: dept scope 规则匹配时，是否把子部门资产也算入。
   - Recommendation: planner 决策——若递归，匹配函数 `matchException` 需在 dept scope 规则命中时额外查 `sys_dept WHERE ancestors LIKE '%'||scope_dept_id||'%'` 取子部门 ID 集，再判断 assetUserID ∈ 子部门用户集。简化版不递归，仅匹配 scope_id 对应部门直接成员（D-R3-A3-01 留 planner discretion）。

4. **"显示已静默"开关是否在异常列表提供（D-R3-A1-01 Claude's Discretion）** — RESOLVED: 提供（见 44-01 Task 4 ShowSilenced + 44-01 Task 6c 前端 Switch 组件）。
   - What we know: D-R3-A1-01 留此开关为 planner discretion。
   - Recommendation: 提供（低成本，复用 ExceptionListParams.ShowSilenced 字段 + 前端 Switch 组件），运维审计场景需要。

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| PostgreSQL | GiST inet_ops 索引 + CHECK 约束 + `>>` 操作符 + `applied_actions TEXT[]` | ✓（项目 PG 18） | 18 | SQLite 跳过 GiST（migration_174 已加 `if !isPostgreSQL(db) { return nil }`）|
| Go stdlib `net` | `net.ParseCIDR` + `ipNet.Contains` CIDR 内存匹配 | ✓ | go1.24.5 内置 | — |
| `github.com/lib/pq` | `pq.StringArray` TEXT[] 序列化 | ✓ | 已在 go.mod | — |
| Redis | cache_keys.go helper（CacheKeyReconciliationExceptionRuleList）| ✓ | 7.4 | 若 Redis 不可用，CacheProvider 降级为 non-cached（Phase 42 模式） |
| `sys_job` 表 + cron scheduler | cleanupExpiredExceptions cron 触发 | ✓ | Phase 42 已注册（reconciliation_tasks.go:128-131）| — |
| `config_service.GetByKey` | 降噪基线快照读写 | ✓ | Phase 42 基建 | — |
| `operations.ReferenceResolver` | Excel scope_name → UUID 解析 | ✓ | Phase 1 基建 | — |

**Missing dependencies with no fallback:** 无。

**Missing dependencies with fallback:** 无。

## Validation Architecture

> `workflow.nyquist_validation: true`（.planning/config.json 确认）。本节供 planner 生成 VALIDATION.md。

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go testing（后端）+ Vitest（前端，复用 Phase 42 模式）|
| Config file | `go test`（无 ini）；`xingran-react-frontend/vitest.config.ts` |
| Quick run command | `go test ./internal/services/asset/... ./internal/api/v1/asset/... ./internal/scheduler/... -count=1` |
| Full suite command | `go test ./... && cd xingran-react-frontend && npm run test` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| EXCEPTION-01 | CIDR 格式校验 + GiST 索引创建 | unit（service 层 ValidateCIDR）+ migration 集成 | `go test ./internal/services/asset/ -run TestValidateCIDR -count=1` | ❌ Wave 0 新建 |
| EXCEPTION-01 | net.ParseCIDR + ipNet.Contains 匹配正确性（IPv4/IPv6/边界） | unit（纯函数） | `go test ./internal/services/asset/ -run TestMatchException -count=1` | ❌ Wave 0 新建 |
| EXCEPTION-01 | GiST inet_ops 索引存在 + `>>` 查询返回正确规则 | integration（PG only，sqlmock 不可） | `go test ./internal/services/asset/ -run TestMatchTest -count=1`（需 PG dev DB）| ❌ Wave 0 新建 |
| EXCEPTION-02 | 5 actions 白名单校验（chk_recon_exc_actions CHECK 约束） | unit（service ValidateActions）+ integration（INSERT 拒绝非法值） | `go test ./internal/services/asset/ -run TestValidateActions -count=1` | ❌ Wave 0 新建 |
| EXCEPTION-02 | 多规则合并 actions 取并集 | unit（纯函数 mergeActions） | `go test ./internal/services/asset/ -run TestMergeActions -count=1` | ❌ Wave 0 新建 |
| EXCEPTION-02 | skip_severity 降级链（critical→high→medium→low） | unit（纯函数 applySkipSeverity） | `go test ./internal/services/asset/ -run TestApplySkipSeverity -count=1` | ❌ Wave 0 新建 |
| EXCEPTION-02 | severity_override 多规则取最低（最宽松） | unit（mergeActions 含 override 分支） | `go test ./internal/services/asset/ -run TestMergeActionsSeverityOverride -count=1` | ❌ Wave 0 新建 |
| EXCEPTION-02 | 转单 cron SQL 加 no_workorder 过滤（命中 no_workorder 不转单） | integration（mock workorder service） | `go test ./internal/scheduler/ -run TestCreateWorkorderNoWorkorderFilter -count=1` | ❌ Wave 0 新建 |
| EXCEPTION-03 | cleanupExpiredExceptions 软停用 is_active=1（不删记录） | integration（sqlite 测试 DB 即可） | `go test ./internal/scheduler/ -run TestCleanupExpiredExceptions -count=1` | ❌ Wave 0 新建 |
| EXCEPTION-03 | cleanupExpiredExceptions 幂等（二次调用 rowsAffected=0） | integration | `go test ./internal/scheduler/ -run TestCleanupExpiredExceptionsIdempotent -count=1` | ❌ Wave 0 新建 |
| EXCEPTION-03 | 过期规则软停用后历史 exception_rule_id 仍指向有效记录（审计链不断） | integration | `go test ./internal/services/asset/ -run TestSoftDisablePreservesFK -count=1` | ❌ Wave 0 新建 |
| EXCEPTION-04 | 命中测试端点返回合并卡片 + 命中规则列表 | unit（service MatchTest）+ integration | `go test ./internal/services/asset/ -run TestMatchTestResult -count=1` | ❌ Wave 0 新建 |
| EXCEPTION-04 | 命中测试未指定 user/dept 时 needsUserDept=true（dept/user scope 不参与） | unit | `go test ./internal/services/asset/ -run TestMatchTestNeedsUserDept -count=1` | ❌ Wave 0 新建 |
| SC 7 | silence 默认过滤（ListExceptions 不返回 silence 记录） | integration | `go test ./internal/services/asset/ -run TestListExceptionsSilenceFilter -count=1` | ❌ Wave 0 新建 |
| SC 7 | ShowSilenced=true 时 silence 记录可见 | integration | `go test ./internal/services/asset/ -run TestListExceptionsShowSilenced -count=1` | ❌ Wave 0 新建 |
| SC 8 | 降噪对比端点用独立 COUNT（不用 list.length） | unit + 静态检查（grep list.length） | `go test ./internal/services/asset/ -run TestBaselineCompare -count=1` | ❌ Wave 0 新建 |
| SC 10 | Layer 3.5 命中例外仍写表（exception_rule_id + applied_actions 非空） | integration | `go test ./internal/services/asset/ -run TestDetectLayer3ExceptionHit -count=1` | ❌ Wave 0 新建 |
| AUDIT-01 | 例外规则 CRUD handler 调 operlog.Record | integration（mock operlog Recorder） | `go test ./internal/api/v1/asset/ -run TestExceptionRuleCRUDOperlog -count=1` | ❌ Wave 0 新建 |
| operlog 回归 | ModuleReconciliationExceptionRule 常量不破坏 25 OperType + 11 关键词 | regression（现有 regression_test.go 自动覆盖） | `go test ./internal/utils/operlog/ -count=1` | ✅ 已存在 |

### Sampling Rate

- **Per task commit:** `go test ./internal/services/asset/... ./internal/api/v1/asset/... ./internal/scheduler/... -count=1 -short`
- **Per wave merge:** `go test ./... && cd xingran-react-frontend && npm run test && npm run build`
- **Phase gate:** Full suite green + `go build ./...` + `npm run build` 全通过（CLAUDE.md 强制）+ manual UAT 走 SC 1-10 全部 10 条

### Wave 0 Gaps

- [ ] `internal/services/asset/reconciliation_exception_matcher_test.go` — covers EXCEPTION-01/02（matchException + mergeActions + applySkipSeverity 纯函数）
- [ ] `internal/services/asset/reconciliation_exception_test.go` 扩展 — covers EXCEPTION-01/02/04（Create/Update/Delete/MatchTest service 层 + ValidateCIDR/ValidateActions）
- [ ] `internal/services/asset/reconciliation_detection_test.go` 扩展 — covers SC 10（Layer 3.5 插入后命中例外仍写表）
- [ ] `internal/services/asset/reconciliation_service_test.go` 扩展 — covers SC 7（silence 过滤）
- [ ] `internal/services/asset/reconciliation_baseline_test.go` 新建 — covers SC 8（降噪对比）
- [ ] `internal/scheduler/reconciliation_tasks_test.go` 新建/扩展 — covers EXCEPTION-03（cleanupExpiredExceptions 软停用 + 幂等）+ EXCEPTION-02（转单 SQL no_workorder 过滤）
- [ ] `internal/api/v1/asset/reconciliation_exception_handler_test.go` 扩展 — covers AUDIT-01（CRUD operlog 接入）
- [ ] migration_174 集成测试（PG dev DB）：手动验证 `SELECT indexname FROM pg_indexes WHERE indexname='idx_recon_exc_active_range'` + `SELECT conname FROM pg_constraint WHERE conname LIKE 'chk_recon_exc_%'`

*(若 dev DB 不可用，部分 integration 测试可降级为 sqlmock 验证 SQL 语句正确性，但 GiST 索引存在性必须 PG 集成验证)*

## Security Domain

> `security_enforcement` 在 .planning/config.json 未显式设置（absent = enabled）。R3 涉及 admin CRUD 端点 + 命中测试，需评估 ASVS。

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | yes（继承项目 JWT 双 token） | 现有 Auth 中间件（不在 R3 范围） |
| V3 Session Management | yes（继承项目） | 现有 JWT + Refresh Token（不在 R3 范围） |
| V4 Access Control | yes | 权限粒度 `asset:reconciliation:exception:{list,create,update,delete,test}`（44-CONTEXT.md 已锁定）；router 层加 RequirePermissions 中间件 |
| V5 Input Validation | yes | service 层 `ValidateCIDR`（net.ParseCIDR）+ `ValidateActions`（白名单）+ `ValidateSeverityOverride`（low/medium/high）+ DB CHECK 约束兜底（chk_recon_exc_actions / chk_recon_exc_severity_override）|
| V6 Cryptography | no | R3 不涉及加密（例外规则无敏感字段；reason 是普通文本） |
| V7 Logging | yes | operlog.Record 接入所有 CRUD（CLAUDE.md 强制 + AUDIT-01） |

### Known Threat Patterns for R3

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| CIDR 注入（用户构造畸形 CIDR 绕过匹配） | Tampering | service 层 `net.ParseCIDR` 严格校验，解析失败拒绝创建；DB `cidr` 列类型兜底（INSERT 非法值 SQLSTATE 22P02）|
| 越权创建例外规则（普通用户给自己 IP 段开 silence） | Elevation | 权限粒度 `asset:reconciliation:exception:create` 仅 admin 角色；router 层 RequirePermissions 中间件 |
| SQL 注入（命中测试 IP 参数拼接 SQL） | Tampering | 用 GORM 占位符 `Where("ip_range >> ?::inet", ip)`，禁止字符串拼接 |
| 告警风暴（运维误配 `0.0.0.0/0 silence` 全静默） | Denial of Service | service 层校验 `reason ≥10 字符`（强制说明原因）+ `expires_at` 默认 30 天（INFRA-02 default_expiry_days）+ admin 列表高亮"覆盖范围过广"规则（planner discretion） |
| 审计链断裂（例外规则硬删除） | Repudiation | D-R3-A4-03 软停用（is_active=1）不删记录；migration_174 不动 deleted_at 列 |

## Sources

### Primary (HIGH confidence)

- `internal/services/asset/reconciliation_detection.go:191-332` — DetectLayer3 现有循环 + R2 guard 1/2 + 插入点 :262（CITED）
- `internal/models/reconciliation.go:67-113` — SysReconciliationException 模型字段（CITED）
- `internal/core/db/migrations/migration_168_reconciliation_tables.go` — 表 schema + DO$$ partial unique index 模式（CITED）
- `internal/middleware/apikey.go:110-141` — `net.ParseCIDR` + `ipNet.Contains` 现成 CIDR 匹配模式（CITED）
- `internal/scheduler/reconciliation_tasks.go:42-170` — 单 taskType "reconciliation" 分发 + cleanupExpiredExceptions placeholder（CITED）
- `internal/services/operations/excel_config.go:50-297` — ExcelConfigs map + Reference/DBField/UpsertKey 模式（CITED）
- `internal/services/operations/reference_resolver.go:29-100,298-322` — ReferenceResolver 名称→UUID 解析（CITED）
- `internal/services/asset/cache_keys.go` — CacheKeyReconciliationExceptionRuleList helper（CITED）
- `internal/services/asset/reconciliation_statistics.go:443-478` — ExceptionRuleStats 端点已实现（CITED）
- `internal/services/asset/reconciliation_workorder.go:188-266` — 转单 SQL 现有 WHERE（CITED）
- `internal/api/v1/asset/reconciliation_exception_router.go` — 现有 list/:id 路由（CITED）
- `internal/utils/operlog/regression_test.go:45-208` — 25 OperType + 18 mandatory sensitive keywords 锁定（CITED）
- `internal/core/db/database.go:455-483` — migration 168-173 链式注册模式（CITED）
- `internal/services/system/config_service.go:82-211` — config CRUD + GetByKey（CITED）
- `.planning/phases/44-ip-r3/44-CONTEXT.md` — 12 锁定决策 + canonical_refs（CITED）
- `.planning/phases/42-r1/42-CONTEXT.md` — R1 18 决策（D-16 operlog module 锁定）（CITED）
- `.planning/notes/asset-reconciliation-strategy.md` §4 + §5.2 — 例外规则体系 + GiST/CHECK schema 定义（CITED）
- `.planning/notes/260627-reconciliation-reuse-audit.md` §3.3 P3 + §4.1/4.4/4.7 — Excel 骨架 + cache key + operlog module + cron 注册（CITED）

### Secondary (MEDIUM confidence)

- PostgreSQL 官方 GiST 文档 + paquier.xyz "Postgres 9.4 Feature Highlight — GiST operator class for inet and cidr" — inet_ops opclass 内置 PG 9.4+ 无需 CREATE EXTENSION `[VERIFIED: web search]`
- Stack Overflow / DBA Stack Exchange — CIDR 包含查询 `>>` + GiST inet_ops 加速最佳实践 `[VERIFIED: web search]`

### Tertiary (LOW confidence)

- 无 LOW confidence 来源。所有关键技术决策（CIDR 匹配、GiST opclass、合并算法、cron 模式、Excel 模式）均有 CITED 代码证据或 VERIFIED 官方文档。

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — 全部复用现有依赖，无新增包；PG GiST inet_ops 经 web search 验证
- Architecture: HIGH — 12 锁定决策 + 插入点代码已读 + 现有 R1/R2 代码模式可直接复用
- Pitfalls: HIGH — 8 个 pitfalls 全部基于项目记忆（xingran-excel-import-* / stat-cards-from-list-length / gorm-sql-constraint-naming-conflict）+ 已读代码验证
- Excel 导入 TEXT[]/条件 Reference 支持: MEDIUM — A3/A4 需 planner 在 Task 验证

**Research date:** 2026-06-28
**Valid until:** 2026-07-28（30 天，stable schema + 锁定决策无 fast-moving 依赖）
