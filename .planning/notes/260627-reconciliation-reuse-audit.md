# 资产对账架构 — 系统已有设计复用度审计

**审计日期**: 2026-06-27
**审计范围**: v0.3 资产对账架构 vs XingRan-Next 现有系统
**审计目标**: 确保新模块充分复用字典、参数、缓存、操作日志、Excel、Status Convention 等已有设计
**审计方法**: 代码取证 + 命名约定 + 项目记忆清单，逐项打勾
**审计结论**: ⚠️ v0.3 设计有 7 项未充分复用，需在 R1 plan-phase 之前补齐
**关联决策记录**: [.planning/notes/asset-reconciliation-strategy.md](./asset-reconciliation-strategy.md) §15

---

## 1. 审计总览

| 类别 | 数量 | 状态 |
|------|------|------|
| ✅ 已正确复用 | 13 项 | — |
| ⚠️ 部分复用 / 需补充 | 4 项 | 中 |
| ❌ 未复用 / 需新增 | 7 项 | 高 |
| **复用完整度** | **65%** | 需补齐到 ≥90% |

---

## 2. 已正确复用（13 项）✅

### 2.1 数据字典后端基础设施

**代码证据**：
- `internal/models/dict.go:3-24` — `DictType` / `DictData` 模型
- `internal/services/system/dict_cache_impl.go` — 字典缓存实现
- 命名约定：`<module>_<resource>_<field>`（如 `ops_dedicated_line_type`、`ops_isp`、`ops_info_point_type`）

**v0.3 复用点**：
- ✅ 字典查询走 `system.DictService` 而非新建
- ✅ 字典类型命名遵循模块前缀（asset 模块 → `asset_reconciliation_*`）

### 2.2 参数管理基础设施

**代码证据**：
- `internal/models/config.go:4-12` — `Config` 模型
- `internal/services/system/config_service.go:23-34` — `ConfigService` 接口
- `GetByKey(ctx, configKey)` 方法在 `config_service.go:201-211`
- 表名 `sys_config`，字段 `config_key` + `config_value` + `config_type` (Y/N) + `is_system`

**v0.3 复用点**：
- ✅ 配置项查询走 `system.ConfigService.GetByKey()`
- ✅ R1 的物化视图刷新频率、评分权重等用 `sys_config` 存储

### 2.3 操作日志体系（Phase 34 落地）

**代码证据**：
- CLAUDE.md 强制约定：所有业务写操作必须 `operlog.Record(...)` / `operlog.RecordWithBody(...)`
- `internal/utils/operlog/regression_test.go` 锁定 25 个 OperType 常量值
- 11 个强制敏感关键词（password/pwd/secret/token/key 等）

**v0.3 复用点**：
- ✅ 异常标记已解决：`operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "资产管理", operlog.OperTypeUpdate)`
- ✅ 例外规则 CRUD：`operlog.Record` (reason 字段无敏感词)
- ✅ 自动转工单：`operlog.Record(..., operlog.OperTypeApprove)`
- ✅ Module 中文名：`资产管理`（与 `excelEntityModuleNames` map 中 `"asset": "资产管理"` 对齐）

### 2.4 通用 CRUD 模式

**代码证据**：
- `internal/api/v1/system/user_router.go:11-53` 是标准模板
- `SetupXxxRouter` + 服务构造（带/不带 cache 分支）+ Handler 注入
- Handler-Service 模式 + 接口 + 私有实现 + 构造函数

**v0.3 复用点**：
- ✅ `internal/api/v1/asset/reconciliation_router.go` 用 `SetupReconciliationRouter(r, core)` 命名
- ✅ Service 拆 `ReconciliationService` interface + `reconciliationServiceImpl` 私有实现
- ✅ Cache 分支：`if core.DataCacheService != nil { ... } else { ... }`

### 2.5 双缓存架构

**代码证据**：
- `internal/services/system/` 下的 `*_cache_impl.go`（dict/menu/user/post/dept/role/notice/settings）
- `cacheProvider := system.NewCacheAdapter(core.Cache)` 模式
- CLAUDE.md 缓存 Key 必须经 helper 函数，不能硬编码

**v0.3 复用点**：
- ✅ `reconciliation_normalized` 物化视图 ETL 状态缓存走 `CacheProvider`
- ✅ 例外规则查询缓存走同一接口
- ⚠️ **需补充**：缓存 Key helper 函数定义（参考 `cache_keys.go` 现有 `GetDictDataByTypeKey` 模式）

### 2.6 响应式分页（Phase A 基建）

**代码证据**：
- CLAUDE.md：`BaseListRequest` + `ApplySort` 白名单
- 项目记忆 `xingran-server-side-sort-infra`：Phase A 已就位
- `internal/services/operations/pagination_helper.go:17`：`MaxPageSize = 10000`
- `internal/services/system/config_service.go:32-33`：Statistics 用专用 COUNT 端点

**v0.3 复用点**：
- ✅ 异常列表 `ExceptionListParams extends BaseListRequest`
- ✅ 排序字段白名单 `reconAllowedSortFields`
- ✅ Dashboard 用 `Statistics` 专用 COUNT 端点

### 2.7 Excel 导入导出（基础设施）

**代码证据**：
- `internal/api/v1/operations/excel_handler.go:46-50` — `SetupExcelRouter(r, entityType, core)` 通用路由
- `internal/services/operations/excel_config.go:50+` — `ExcelConfigs` map
- 已注册 entityType：`department`、`user`、`building`、`floor`、`workstation`、`asset`
- operlog 中文模块名映射：`excelEntityModuleNames` map + `excelModuleName()` 默认 `"Excel导入导出"`

**v0.3 复用点**：
- ✅ 异常列表导出：复用 `SetupExcelRouter(r, "reconciliationException", core)`
- ✅ 例外规则导入导出：复用 `SetupExcelRouter(r, "reconciliationExceptionRule", core)`

### 2.8 UUID + 软删除 + BaseModel

**代码证据**：
- `internal/models/base.go:11-19` — `BaseModel` 嵌入
- `BeforeCreate` hook（`base.go:22-27`）自动生成 UUID

**v0.3 复用点**：
- ✅ `sys_data_reconciliation` / `sys_reconciliation_exception` 表继承 BaseModel
- ✅ UUID 主键 + 软删除（与现有约定一致）

### 2.9 Status Value Convention

**代码证据**：
- CLAUDE.md：`0 = enabled/normal/visible, 1 = disabled/stopped/hidden`
- 例外：菜单可见性 `1 = visible, 0 = hidden`（boolean 语义）

**v0.3 复用点**：
- ✅ `sys_reconciliation_exception.is_active` 用 `int`（0=启用 1=停用）
- ✅ `resolved_at IS NULL` 表示 open / not null 表示 resolved（无单独 status 字段，避免冗余）

### 2.10 前端 useDict hook + queryKeys

**代码证据**：
- `xingran-react-frontend/src/hooks/useDict.ts:44-59` — `useDict(dictType)` 实现
- `xingran-react-frontend/src/hooks/useDict.ts:67-70` — `useInvalidateDict()`

**v0.3 复用点**：
- ✅ 冲突类型标签：`useDict("asset_reconciliation_conflict_type")`
- ✅ 严重级别：`useDict("asset_reconciliation_severity")`
- ✅ 例外 actions：`useDict("asset_reconciliation_exception_action")`

### 2.11 前端 opsApi.ts 工厂模式

**代码证据**：
- CLAUDE.md：`buildingApi.list({ current, pageSize, ... })` 模式
- 已有：building/floor/workstation/serverRoom/dedicatedLine/infoPoint/asset/excel

**v0.3 复用点**：
- ✅ 新建 `src/lib/assetApi.ts`（参考 opsApi.ts 模式）

### 2.12 ECharts 6 复用

**代码证据**：
- 项目记忆 `echarts6-customchart-tree-shaking-noop`：CustomChart-only 按需引入对 echarts 6 无效
- 项目已统一使用 `echarts-for-react` v3.0.5

**v0.3 复用点**：
- ✅ Dashboard 所有图表用 `echarts-for-react`
- ⚠️ 不追求按需引入，按项目现实经验打包

### 2.13 响应式 UI 库

**代码证据**：
- Ant Design 6.1 + Tailwind CSS 4.1
- 现有页面统一用 `antd` 组件 + `Row/Col` 栅格

**v0.3 复用点**：
- ✅ 所有页面用 antd `Table`、`Form`、`Modal`、`Drawer`、`Tabs`、`Card`、`Select`、`DatePicker`
- ✅ 不引入新的 UI 库

---

## 3. 部分复用 / 需补充（4 项）⚠️

### 3.1 P1：数据字典枚举值定义未在 v0.3 中明确

**现状**：v0.3 提到用字典，但未具体列出 4 个字典的完整 dict_data。

**需补充**：R1 启动时新建以下字典 seed：

| dict_type | dict_label | dict_value | list_class |
|-----------|------------|------------|------------|
| `asset_reconciliation_conflict_type` | 物理有/责任人有且一致 | A | success |
| | 物理有/责任人无 | B | warning |
| | 物理有/责任人不一致 | C | error |
| | 物理无/责任人有 | D | warning |
| | 三路均无 | E | default |
| | AD 单独不一致 | F | processing |
| `asset_reconciliation_severity` | 低 | low | default |
| | 中 | medium | warning |
| | 高 | high | error |
| | 严重 | critical | error |
| `asset_reconciliation_exception_action` | 不告警 | no_alert | default |
| | 不通知 | no_notice | default |
| | 不开工单 | no_workorder | default |
| | 降低严重级 | skip_severity | warning |
| | 全静默 | silence | default |
| `asset_reconciliation_status` | 未解决 | open | warning |
| | 已解决 | resolved | success |

**migration 文件位置**：`internal/core/db/migrations/migration_NNN_reconciliation_dicts.go`

### 3.2 P2：参数管理阈值配置项未在 v0.3 中明确

**现状**：v0.3 提到用 config，但未列出具体 key 与默认值。

**需补充**：R1 启动时在 `sys_config` 表 seed 以下配置项：

| config_key | config_value | config_type |
|------------|--------------|-------------|
| `asset.reconciliation.view.refresh_interval` | `5m` | Y |
| `asset.reconciliation.score.physical` | `0.5` | Y |
| `asset.reconciliation.score.declared` | `0.3` | Y |
| `asset.reconciliation.score.ad` | `0.2` | Y |
| `asset.reconciliation.exception.default_expiry_days` | `30` | Y |
| `asset.reconciliation.alert.critical_threshold` | `5` | Y |
| `asset.reconciliation.alert.silence_after_resolved_hours` | `168` | Y |
| `asset.reconciliation.health.score_weights` | `{normal:1.0, drift:0.5, conflict:0.0, nodata:0.7}` | Y |

### 3.3 P3：Excel 导入配置未在 v0.3 中明确

**现状**：v0.3 提到复用 ExcelConfig 模式，但未列出具体字段。

**需补充**：R3 实施时在 `internal/services/operations/excel_config.go` 新增：

```go
"reconciliationException": {
    SheetName:     "对账异常列表",
    TableName:     "sys_data_reconciliation",
    EntityName:    "对账异常",
    HasHeader:     true,
    StartRow:      2,
    CachePatterns: []string{"reconciliation:*"},
    UniqueKeys:    []string{"id"},
    PartialUpdate: false,
    Columns: []ExcelColumn{
        {Field: "id", Header: "异常ID", DBField: "id"},
        {Field: "assetCode", Header: "资产编号", DBField: "..." /* via JOIN */},
        {Field: "conflictType", Header: "冲突类型", DBField: "conflict_type"},
        {Field: "severity", Header: "严重级别", DBField: "severity"},
        // ... 完整字段定义
    },
},
"reconciliationExceptionRule": {
    SheetName:     "对账例外规则",
    TableName:     "sys_reconciliation_exception",
    EntityName:    "例外规则",
    HasHeader:     true,
    StartRow:      2,
    CachePatterns: []string{"reconciliation:*"},
    UniqueKeys:    []string{"id"},
    PartialUpdate: true,
    Columns: []ExcelColumn{
        {Field: "name", Header: "规则名称", Required: true, MaxLength: 128, DBField: "name"},
        {Field: "ipRange", Header: "IP段(CIDR)", Required: true, DBField: "ip_range"},
        // ... 完整字段定义
    },
},
```

### 3.4 P4：operlog 中文模块名映射未在 v0.3 中明确

**现状**：v0.3 提到 operlog，但未确认 module 中文名与现有约定的一致性。

**需补充**：R1 启动时把以下 4 个 module 常量定义在 `internal/api/v1/asset/reconciliation_handler.go` 顶部：

| 常量名 | 中文值 | 适用场景 |
|--------|--------|---------|
| `ModuleReconciliationException` | 资产对账 | 异常标记已解决、批量操作 |
| `ModuleReconciliationExceptionRule` | 资产对账-例外规则 | 例外规则 CRUD |
| `ModuleReconciliationAutoWorkorder` | 资产对账-自动转工单 | 自动转工单 |
| `ModuleReconciliationReportExport` | 资产对账-报告导出 | 看板导出 |

**关键约束**：复用现有 `excelEntityModuleNames` map 风格，避免散落字符串。

---

## 4. 未复用 / 需新增（7 项）❌

### 4.1 F1：缓存 Key helper 函数未定义

**问题**：v0.3 提到物化视图 ETL 缓存 + 例外规则查询缓存，但未定义 cache key 常量。

**代码位置**：`internal/services/asset/cache_keys.go`（新建）

```go
package asset

import "fmt"

const (
    CacheKeyReconciliationDashboard           = "reconciliation:dashboard:%s"
    CacheKeyReconciliationExceptionList       = "reconciliation:exception:list:%s"
    CacheKeyReconciliationExceptionByID       = "reconciliation:exception:byID:%s"
    CacheKeyReconciliationExceptionRuleList   = "reconciliation:exceptionRule:list"
    CacheKeyReconciliationExceptionRuleByID   = "reconciliation:exceptionRule:byID:%s"
    CacheKeyReconciliationViewLastRefresh     = "reconciliation:view:lastRefresh"
    CacheKeyReconciliationHealthByWorkstation = "reconciliation:health:workstation:%s"
    CacheKeyReconciliationHealthByAsset       = "reconciliation:health:asset:%s"
)

func GetReconciliationDashboardKey(scope string) string {
    return fmt.Sprintf(CacheKeyReconciliationDashboard, scope)
}
// ... 完整 helper 函数
```

**复用参考**：`internal/services/cache_keys.go` 现有 `GetDictDataByTypeKey(dictType)` 模式。

### 4.2 F2：Statistics 专用 COUNT 端点未在 v0.3 中明确

**问题**：v0.3 提到 Dashboard，但未明确使用 Statistics 端点（防 MaxPageSize=100 钳制）。

**代码位置**：`internal/services/asset/reconciliation_statistics.go`（新建）

| 端点 | 用途 |
|------|------|
| `POST /asset/reconciliation/statistics/summary` | 总览 KPI |
| `POST /asset/reconciliation/statistics/by-conflict-type` | 冲突类型分布 |
| `POST /asset/reconciliation/statistics/by-severity` | 严重级别分布 |
| `POST /asset/reconciliation/statistics/health-trend` | 健康度趋势 |
| `POST /asset/reconciliation/statistics/top-unresolved` | Top 10 长期未解决 |
| `POST /asset/reconciliation/statistics/exception-rule-stats` | 例外规则生效统计 |

**接口定义**：

```go
type ReconciliationStatistics interface {
    Summary(ctx context.Context, filters StatsFilter) (*SummaryResult, error)
    ByConflictType(ctx context.Context, filters StatsFilter) (map[string]int64, error)
    BySeverity(ctx context.Context, filters StatsFilter) (map[string]int64, error)
    HealthTrend(ctx context.Context, filters StatsFilter) ([]TrendPoint, error)
    TopUnresolved(ctx context.Context, limit int) ([]ExceptionSummary, error)
    ExceptionRuleStats(ctx context.Context) ([]RuleStats, error)
}
```

**关键约束**：**严禁** 用 `list.length`，必须用 `SELECT COUNT(*)` 或聚合查询（参考 `stat-cards-from-list-length-capped-at-100`）。

### 4.3 F3：前端 queryKeys.ts 未注册 reconciliation keys

**问题**：前端架构图未明确 queryKeys 注册。

**代码位置**：`src/lib/queryKeys.ts`（修改）

```typescript
export const queryKeys = {
  // ... 已有
  reconciliation: {
    all: ['reconciliation'] as const,
    dashboard: (filters: DashboardFilters) => 
      [...queryKeys.reconciliation.all, 'dashboard', filters] as const,
    exceptionList: (params: ExceptionListParams) => 
      [...queryKeys.reconciliation.all, 'exceptions', params] as const,
    exceptionDetail: (id: string) => 
      [...queryKeys.reconciliation.all, 'exception', id] as const,
    ruleList: () => 
      [...queryKeys.reconciliation.all, 'rules'] as const,
    ruleDetail: (id: string) => 
      [...queryKeys.reconciliation.all, 'rule', id] as const,
    workstationHealth: (workstationId: string) => 
      [...queryKeys.reconciliation.all, 'workstationHealth', workstationId] as const,
    assetHealth: (assetId: string) => 
      [...queryKeys.reconciliation.all, 'assetHealth', assetId] as const,
    matchTest: (ip: string) => 
      [...queryKeys.reconciliation.all, 'matchTest', ip] as const,
  },
};
```

### 4.4 F4：operlog module 映射常量未在 v0.3 中明确

**问题**：v0.3 提到 operlog，但 module 中文名是字符串硬编码，未走常量。

**代码位置**：`internal/api/v1/asset/reconciliation_handler.go`（新建，顶部常量）

```go
const (
    ModuleReconciliationException     = "资产对账"
    ModuleReconciliationExceptionRule = "资产对账-例外规则"
    ModuleReconciliationAutoWorkorder = "资产对账-自动转工单"
    ModuleReconciliationReportExport  = "资产对账-报告导出"
)
```

**关键约束**：复用现有 `excelEntityModuleNames` map 风格，避免散落字符串。

### 4.5 F5：健康度评分函数位置未明确

**问题**：v0.3 提到健康度评分（`health_score = (1 - 异常数/总数) × 100`），但未明确函数位置与复用现有 service 模式。

**代码位置**：`internal/services/asset/reconciliation_health.go`（新建）

```go
package asset

import "context"

type HealthScoreCalculator interface {
    Score(ctx context.Context, workstationID string) (*HealthScore, error)
    BatchScore(ctx context.Context, workstationIDs []string) (map[string]*HealthScore, error)
}

type HealthScore struct {
    Total        int          `json:"total"`
    Normal       int          `json:"normal"`
    Drift        int          `json:"drift"`
    Conflict     int          `json:"conflict"`
    NoData       int          `json:"noData"`
    ExceptionHit int          `json:"exceptionHit"`
    Score        float64      `json:"score"`
    Trend        []TrendPoint `json:"trend"`
}
```

**复用现有模式**：参考 `config_service.go` 的 `Statistics` 接口 + 专用 COUNT 实现。

### 4.6 F6：路由注册位置未在 v0.3 中明确

**问题**：v0.3 提到路由，但未明确注册到主 router 的位置。

**代码位置**：`internal/api/router.go`（修改）

```go
// 在 asset 模块注册分组下添加：
assetGroup := r.Group("/asset")
asset.SetupAssetRouter(assetGroup, core)
asset.SetupReconciliationRouter(assetGroup, core)            // 🆕
asset.SetupReconciliationExceptionRouter(assetGroup, core)  // 🆕
```

**关键约束**：
- ⚠️ 参考 `xingran-excel-import-route-conflict` 历史教训：router.go 不能预注册 `/asset/reconciliation/*` 通用路由（避免与具体 handler 冲突）
- ⚠️ `excel_handler.SetupExcelRouter` 不能预注册 `reconciliationException` entityType（避免冲突）
- ⚠️ `reconciliation` entityType 不属于 ops ExcelConfigs 注册范围，避免与 ops 模块 Excel 路由冲突

### 4.7 F7：Cron 任务注册位置未在 v0.3 中明确

**问题**：v0.3 提到 5min 物化视图刷新 + 7d 静默期回收，但未明确 cron 注册。

**代码位置**：`internal/scheduler/reconciliation_tasks.go`（新建）

| Cron 表达式 | 用途 |
|------------|------|
| `@every 5m` | 物化视图刷新 |
| `@every 10m` | 异常检测 |
| `0 2 * * *` | 静默期到期重检测（每日 02:00） |
| `0 3 * * *` | 临时例外规则清理（每日 03:00） |

**注册位置**：`internal/scheduler/ad_sync_tasks.go` 现有 `StartADSyncScheduler` 旁，新增 `StartReconciliationScheduler`。

**复用现有模式**：参考 `internal/scheduler/ad_sync_tasks.go` 的 cron 注册 + 错误日志模式。

---

## 5. 优先级排序（影响 R1 启动）

| 优先级 | 项 | 任务 |
|--------|-----|------|
| 🔴 高 | F2 Statistics 端点 | 必须在 schema 设计时同步定义 |
| 🔴 高 | F6 路由注册 | 必须在 router.go 设计时规划 |
| 🔴 高 | F3 字典 seed | 必须在 migration 中定义 |
| 🟡 中 | F4 参数 seed | R1 启动时定义 |
| 🟡 中 | F1 cache key helper | R1 service 层第一行 |
| 🟡 中 | F5 queryKeys | 前端第一行代码 |
| 🟢 低 | F7 Excel config | R3 实施时定义 |
| 🟢 低 | F4 operlog module 常量 | R1 实施时统一规划 |
| 🟢 低 | F5 HealthScore 函数 | R4 实施时定义 |
| 🟢 低 | F7 Cron 注册 | R2 实施时定义 |

---

## 6. 已知项目记忆应用

| 记忆 | 应用点 |
|------|--------|
| `xingran-gorm-sql-constraint-naming-conflict` | migration uniqueIndex 用 `uni_*_*` 命名 |
| `xingran-migrations-no-sql-autoloader` | 必须 .go migration + AutoMigrate 显式调用 |
| `migration-sql-name-must-match-model` | menu/migration 字段名匹配 model |
| `xingran-perm-namespace-split-readonly-page` | 跨模块权限边界声明 + `RequirePermissionsWithQuery` |
| `xingran-excel-import-route-conflict` | router.go 不能预注册通用 `/reconciliation/*` |
| `stat-cards-from-list-length-capped-at-100` | Dashboard 必须用 Statistics 端点，不用 list.length |
| `echarts6-customchart-tree-shaking-noop` | echarts 6 不追求按需引入 |
| `xingran-server-side-sort-infra` | Phase A 排序基建已就位 |
| `captcha-enabled-invalid-value-trap` | 字典配置变更后必须 invalidate queries |
| `xingran-gorm-sql-constraint-naming-conflict` | 新表字段必须唯一索引命名规范 |

---

## 7. 后续行动

R1 plan-phase 启动前必须完成：
1. 本审计报告 + 主 Note v0.4 章节全部落盘
2. Todo 中 T20-T26 跟踪项到位
3. R1 plan-phase 第一个 plan 包含：
   - 4 个字典 migration
   - 8 个 config seed
   - cache_keys.go
   - 路由注册 + 冲突规避
   - queryKeys.ts

---

## 8. 审计签字

| 角色 | 签字 | 日期 |
|------|------|------|
| 审计人 | gsd:explore session | 2026-06-27 |
| 复核人 | （待 R1 启动时确认） | — |