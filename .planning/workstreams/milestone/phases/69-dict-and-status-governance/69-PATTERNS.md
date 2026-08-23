# Phase 69: 字典与状态值治理 - Pattern Map

**Mapped:** 2026-08-19
**Files analyzed:** 14 组（6 个新建 + 8 组修改批次，含 Wave 0 测试/守护脚本/共享常量）
**Analogs found:** 12 / 14（含 2 个 exact 级、多个 exact；2 项无近似范本）

> 本 phase 无 CONTEXT.md，文件清单全部来自 `69-RESEARCH.md`（Phase Requirements + Wave 0 Gaps + 各 DICT-0x 实现路径）。

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/models/status_constants_test.go`（新） | test | transform（AST 解析源码锁值） | `internal/utils/operlog/regression_test.go` | exact |
| `internal/core/db/migrations/migration_208_dict_seed.go`（新） | migration | batch（幂等 seed） | `migration_203_connection_pool_sysconfig.go` + `migration_207_menu_catalog_seed.go` | exact（混合双范本） |
| `internal/core/db/migrations/migration_208_dict_seed_test.go`（新） | test | batch | `migration_207_menu_catalog_seed_test.go` | exact |
| `internal/core/db/database.go`（改：注册 208） | config | batch | 同文件 207 的双分支挂法（837-847） | exact |
| `internal/models/base.go` + 各 model 文件（改：补缺常量） | model | — | `base.go:44-135` + `internal/models/operations/server_room.go` | exact |
| `internal/services/system/*` 残余字面量（DICT-01 批1，~5 文件） | service | CRUD | `internal/services/system/department_cache_impl.go:65,93` | exact（目标态已在同目录实现） |
| `internal/services/operations/*` 7 service + `asset_service.go` + `api/v1/operations/excel_handler.go`（批2，~10 文件） | service | CRUD / transform | 同上 + `excel_config.go:146,282`（operations 包常量先例） | exact |
| `internal/services/{vdi,addomain,notice,knowledge,rpa,scheduler}/*`（批3，~8 文件） | service | CRUD / event-driven | `addomain/dept_sync_service.go:167-170`（before 态）+ `department_cache_impl.go`（after 态） | exact |
| `internal/services/{workorder,duty_pool,device_discovery,config_execution,command_dispatch,monitor}/*` + `api/v1/{system/notice_handler,monitor/oper_log_handler}`（批4，~10 文件） | service / controller | batch（CASE WHEN 聚合） | `config_execution_service.go`（同文件半迁移：struct 已用常量、SQL 未替换） | exact |
| `scripts/check-status-literals.*`（新，语言任意） | utility | file-I/O（源码扫描守护） | `scripts/verify/format_unify/main.go`（Go 骨架；无 grep 白名单式源码守护先例） | partial（无完整范本） |
| `xingran-react-frontend/src/constants/status.ts`（新） | config | — | `src/pages/system/user/constants.ts`（内容范本）+ `src/constants/` 既有模块（位置先例） | role-match |
| `xingran-react-frontend/src/constants/status.test.ts`（新） | test | — | `src/components/network/port-write/__tests__/constants.test.ts` | exact |
| 前端 pages 各批（user/workorder/duty/knowledge/network 等 constants + 页面）（改） | component | request-response | `src/pages/operations/dedicated-lines/index.tsx:92-93,241-244,253-261,593-609` | exact |
| `CLAUDE.md`（改：Status Value Convention 段） | config（文档） | — | 无（唯一参照是本文件现行文本） | none |

## Pattern Assignments

### `internal/models/status_constants_test.go`（新，test）

**Analog:** `internal/utils/operlog/regression_test.go`

**头部注释模式**（regression_test.go:3-23）——先写「锁什么、为什么锁」：

```go
// regression_test.go guards the public API of the operlog package against
// silent drift introduced by future refactors. ...
// What is pinned and why:
//   - OperType constant values (TestOperTypeConstantStability): ...
//   - OperType count == 25 (TestOperTypeCountEquals25): ...
```

**期望值 map 模式**（regression_test.go:45-71）：

```go
var expectedOperTypeValues = map[string]int{
    "OperTypeOther":    0,
    "OperTypeCreate":   1,
    // ... 25 项全量枚举，缺失/新增都会 fail
}
```

**核心锁值模式**（regression_test.go:76-105）——AST 解析源文件取实际常量值，与期望 map 双向比对（多出来的常量也报错，逼新增者显式更新）：

```go
func TestOperTypeConstantStability(t *testing.T) {
    t.Parallel()
    actual, err := readOperTypeConsts("operlog.go")
    if err != nil {
        t.Fatalf("failed to parse operlog.go: %v", err)
    }
    for name, want := range expectedOperTypeValues {
        got, ok := actual[name]
        if !ok {
            t.Errorf("OperType constant %q is missing ... removing a constant is a breaking change", name)
            continue
        }
        if got != want {
            t.Errorf("OperType constant %q = %d, want %d (renumbering would mislabel ...)", name, got, want)
        }
    }
    for name, got := range actual {
        if _, ok := expectedOperTypeValues[name]; !ok {
            t.Errorf("unexpected OperType constant %q = %d — if intentional, add it to expectedOperTypeValues and update CLAUDE.md ...", name, got)
        }
    }
}
```

**AST 读取器**（regression_test.go:229-282）`readOperTypeConsts` 可整段改造复用：`parser.ParseFile` + 遍历 `*ast.GenDecl{Tok: token.CONST}` + 前缀过滤。对本文件：前缀不是单一 `OperType`，需要按「常量家族名集合」（UserStatus*/DeptStatus*/ExecutionStatus*/KnowledgeArticleStatus*/...）过滤 `internal/models` 下多个文件——建议扫描包目录（`filepath.Glob("*.go")`）而非单文件。

**断言内容**（按 RESEARCH A2 簇表取关键值）：`UserStatusEnabled=0`、`VisibleShow=1`（反转例外）、`KnowledgeArticleStatusPublished=1`（E 簇反转）、`ExecutionStatusPending=0..Cancelled=4`（B 簇状态机）、`LineStatusNormal/Fault/Disabled=0/1/2`（D 簇多态，注意取 `internal/models/operations/dedicated_line.go` 包）。

---

### `internal/core/db/migrations/migration_208_dict_seed.go`（新，migration/batch seed）

**Analog:** `migration_203_connection_pool_sysconfig.go`（GORM 式，主范本）+ `migration_207_menu_catalog_seed.go`（幂等语义注释范本）

**导入与结构模式**（migration_203:3-9）：

```go
import (
    "log"

    "github.com/xingran-next/xingran-go-backend/internal/models"
    applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
    "gorm.io/gorm"
)
```

**核心模式：count 查重 → 跳过 → 结构体 Create**（migration_203:45-66）：

```go
for _, c := range configs {
    var existingCount int64
    if err := db.Model(&models.Config{}).Where("config_key = ?", c.configKey).Count(&existingCount).Error; err != nil {
        return err
    }
    if existingCount > 0 {
        continue
    }
    cfg := &models.Config{
        ConfigName:  c.configName,
        ConfigKey:   c.configKey,
        ConfigValue: c.configValue,
        ConfigType:  models.ConfigTypeNo, // N = 非系统参数, 前端可编辑
        IsSystem:    models.ConfigIsSystemNo,
        Remark:      c.remark,
    }
    if err := db.Create(cfg).Error; err != nil {
        log.Printf("Migration 203: create config %s failed: %v", c.configKey, err)
        continue
    }
    applogger.Infof("[迁移] sys_config seed: %s = %s", c.configKey, c.configValue)
}
```

migration_208 改造点：外层循环单位是「字典组」（dictType），组级查重用 `db.Model(&models.DictType{}).Where("dict_type = ?", ...)`；组内 `models.DictData{}` 循环 Create；外裹 `db.Transaction`（207:70-98 的事务模式）。**不要照抄 203 的「失败 continue」——字典组应事务回滚整组，失败返回 err 由调用方 WARN**。

**幂等/失败策略文档注释模式**（migration_207:41-53）——「快速路径跳过（尊重用户删除意图）+ 行级幂等 + 单事务 + 失败 WARN 不阻断」四条注释照抄改写。

**被 seed 的模型**（`internal/models/dict.go:3-24`）——字段名直接决定 seed 结构：

```go
type DictType struct {
    BaseModel
    DictName string `gorm:"size:100;not null" json:"dictName"`
    DictType string `gorm:"uniqueIndex;size:100;not null" json:"dictType"`
    Status   int    `gorm:"default:0" json:"status"`
    Remark   string `gorm:"size:500" json:"remark,omitempty"`
}

type DictData struct {
    BaseModel
    DictSort  int     `gorm:"default:0" json:"dictSort"`
    DictLabel string  `gorm:"size:100;not null" json:"dictLabel"`
    DictValue string  `gorm:"size:100;not null" json:"dictValue"`   // 注意: string 类型
    DictType  string  `gorm:"size:100;index" json:"dictType"`
    ListClass *string `gorm:"size:100" json:"listClass,omitempty"`   // tag 颜色
    IsDefault bool    `gorm:"default:false" json:"isDefault"`
    Status    int     `gorm:"default:0" json:"status"`
    ...
}
```

**坑：DictValue 是 string。** `WorkstationType`/`Gender` 等 model 常量是 int——seed 时写 `"0"/"1"/"2"` 字符串（前端 `DictItem.dictValue: string`，dedicated-lines 实证消费即字符串）。值来源（抄值不改链路，RESEARCH A6）：
- `ops_dedicated_line_type` 6 值：`internal/services/operations/excel_config.go:265` 的裸 map（`"internet": "互联网专线", "intranet": "内网专线", "cloud_desktop": "云桌面专线", "mpls": "MPLS VPN", "fiber": "光纤专线", "leased_line": "租用专线"`）
- `ops_workstation_type`：`internal/models/workstation.go:16-18`（Fixed=固定工位/HotDesk=灵活工位/Manager=管理工位）
- `network_device_type` 5 值：`internal/models/network_device.go:7-11`（router/switch/firewall/ap/loadbalancer，label 见行尾中文注释）
- `ops_isp` / `ops_info_point_type` / `asset_reconciliation_*`：`internal/core/db/migrations/archive/`（033/047/048/169，见 RESEARCH A4）

---

### `internal/core/db/migrations/migration_208_dict_seed_test.go`（新，test）

**Analog:** `internal/core/db/migrations/migration_207_menu_catalog_seed_test.go`

**内存 sqlite 夹具模式**（207 test:12-28）：

```go
import (
    "github.com/glebarez/sqlite"   // 纯 Go 驱动, 非 mattn/go-sqlite3
    ...
)

func freshSQLiteDBForMigrate207(t *testing.T) *gorm.DB {
    t.Helper()
    dsn := "file:" + t.Name() + "?mode=memory&cache=shared&_busy_timeout=5000"
    db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
    if err != nil {
        t.Fatalf("open sqlite: %v", err)
    }
    if err := db.AutoMigrate(&models.Menu{}, &models.Role{}, &models.RoleMenu{}); err != nil {
        t.Fatalf("AutoMigrate: %v", err)
    }
    return db
}
```

migration_208 版本：AutoMigrate 最小表集改为 `&models.DictType{}, &models.DictData{}`。

**三个必写用例**（对应 207 test:42-91, 95-125）：
1. 全新库 seed → 断言每组 dictType 行数与总行数 → **再跑一遍断言行数不变**（幂等，207:84-90）
2. 「尊重用户删除意图」：预插一个 dictType → seed 跳过该组不复活（组级 count 查重验证，对应 207:95-125）
3. （可选，207:230-277 集成风格）消费端语义：按 `dict_type = ?` 查 DictData 能取到 isDefault 项

---

### `internal/core/db/database.go`（改：注册 migration_208 双分支）

**Analog:** 同文件 207 的挂法——这是**唯一必须逐字照抄的位置模式**（Pitfall 1 的标准答案）

**PG 分支（advisory-lock 块内）**（database.go:837-842）：

```go
// 规范菜单目录种子 (239 条, 幂等; debug admin-role-incomplete-menus 修复)
// 必须在 InitData 的 createOperationsManagementMenus 之前执行:
// 规范「运维管理」根就位后, Go 种子按 (name, parent) 查重自然全部跳过
if err := migrations.Migrate207SeedCanonicalMenuCatalog(d.DB); err != nil {
    applogger.Errorf("规范菜单目录种子失败 (非阻断,留待下次启动): %v", err)
}
```

**SQLite else 分支**（database.go:843-847）：

```go
} else {
    // sqlite 分支: 规范菜单目录种子 (双方言迁移; PG 分支在上方 advisory-lock 块内执行)
    if err := migrations.Migrate207SeedCanonicalMenuCatalog(d.DB); err != nil {
        applogger.Errorf("规范菜单目录种子失败 (非阻断,留待下次启动): %v", err)
    }
```

migration_208：PG 分支追加在 207 调用之后（块尾）、SQLite 分支追加在 207 调用之后；错误处理统一 `applogger.Errorf(... 非阻断,留待下次启动 ...)`。反例（勿学）：202-206 只在 PG 分支，`configs/config.yaml:12-16` 注释已文档化该缺口。

---

### `internal/models/base.go` + 各 model 文件（改：补缺失实体常量）

**Analog:** `base.go:44-135`（system 族）与 `internal/models/operations/server_room.go:7-13`（operations 族）

**常量定义风格**（base.go:53-59——type + 常量块 + 行尾中文 label 注释）：

```go
// UserStatus 用户状态枚举
type UserStatus int

const (
    UserStatusEnabled  UserStatus = 0 // 启用
    UserStatusDisabled UserStatus = 1 // 禁用
)
```

operations 子包版本（server_room.go:7-13，同风格）：

```go
// RoomStatus 机房状态枚举
type RoomStatus int

const (
    RoomStatusNormal  RoomStatus = 0 // 正常
    RoomStatusStopped RoomStatus = 1 // 停用
)
```

**已核实存在、无需补**：Building（`operations/building.go:11-12`）、Floor/Room/RoomDevice（`models/operations.go`，无中文注释的旧版）、Line（`operations/dedicated_line.go:11-14`）、WorkstationType/Status（`models/workstation.go:4-18`）、DeviceType/DeviceStatus（`models/network_device.go`）、ExecutionStatus（`models/config_execution.go:19-27`）、KnowledgeArticleStatus（`models/knowledge.go:16-17`）、PublishStatus（`models/notice_enhanced.go:23-26`）、JobStatus（`models/log.go:51-56`）。
**执行时需核对/可能补**：Notice、VDIServer、InfoPoint、OperLog 成功/失败（RESEARCH A1 假设；簇 C）。

**CRITICAL 双定义陷阱（本 mapping 新发现，planner 必须写进批 2 任务描述）：** `RoomStatus` 等在**两处**定义——`internal/models/operations.go`（package `models`，旧、无中文注释）与 `internal/models/operations/server_room.go`（package `operations`，新、带注释）。`internal/services/operations/excel_config.go` 通过 `import "github.com/xingran-next/xingran-go-backend/internal/models/operations"` 以 `operations.RoomStatusNormal` 引用的是**子包版**。operations service 批 2 替换时：优先引用 model struct 实际所在的包（OpsServerRoom 在子包 `operations`；旧 ServerRoom/RoomDevice 在 `models`）；不要新造第三份。

---

### DICT-01 批 1-4：`internal/services/**` 与 `internal/api/v1/**` 字面量替换（~34 文件修改）

**Analog（目标态，已存在的参数化常量写法）:** `internal/services/system/department_cache_impl.go:65,93`、`role_service.go:500`

**Before（仓内现状）**——`internal/services/addomain/dept_sync_service.go:167-170`：

```go
err := s.db.WithContext(ctx).
    Preload("Children.Children.Children"). // 预加载3层子部门
    Where("parent_id IS NULL").            // UUID字段不能使用空字符串比较
    Where("status = 0").                   // 0=正常
    Find(&depts).Error
```

**After（department_cache_impl.go:65 的既有先例）**：

```go
query = query.Where("status = ?", models.DeptStatusNormal)
```

**CASE WHEN 聚合替换（批 4 主形态）**——before：`internal/services/duty_pool_service.go:37-41`：

```go
Select(
    "COUNT(*) AS total",
    "SUM(CASE WHEN status = 0 THEN 1 ELSE 0 END) AS enabled",
    "SUM(CASE WHEN status = 1 THEN 1 ELSE 0 END) AS disabled",
).
```

after（无法参数化处用 Sprintf + int() 常量，RESEARCH Pitfall 5）：

```go
fmt.Sprintf("SUM(CASE WHEN status = %d THEN 1 ELSE 0 END) AS enabled", int(models.DutyPoolStatusEnabled))
```

**半迁移先例（批 4 的安全参照）**：`internal/services/config_execution_service.go`——struct 赋值已用常量（:136 `Status: models.ExecutionStatusPending`、:151-159），同文件 raw SQL 仍是字面量（:46-49 `"SUM(CASE WHEN status = 0 THEN 1 ELSE 0 END) AS pending"`）。批 4 就是把这类文件补齐到自洽。

**结构体字面量批 2 主战场**：`workstation_device_service.go` 11 处 `Status: 0/1` → 对应 model 常量（参照 `user_sync_service.go:584` 的 `Status: models.UserStatusEnabled` 写法）。

**簇对照（写进每个 plan 任务的替换表，摘自 RESEARCH A2）：**

| 簇 | 例 | 目标常量 | 严禁 |
|----|----|---------|------|
| A 通用启停 | `building_service.go` 等 `status = 0` | `BuildingStatusNormal` 等 | — |
| B 状态机 | `config_execution_service.go:46-49` | `ExecutionStatusPending/Running/...` | 套 Enabled/Stop |
| C 成败 | `oper_log_handler.go:190`、`fix_suggestion_monitor.go:181` | 查/补 log.go 常量 | — |
| D 多态 | `account_pool.go`（0可用/1停用/2熔断） | 逐实体常量 | 一刀切 |
| E 反转 | knowledge/notice `status = 1` = 已发布 | `KnowledgeArticleStatusPublished` / `PublishStatusPublished` | **套 1=停用** |
| F 排除 | `geocoding_service.go:332` 百度返回码 | 不替换，加注释 | — |

---

### `xingran-react-frontend/src/constants/status.ts` + `status.test.ts`（新，DICT-03 批 4）

**位置先例：** `src/constants/` 目录已存在（`buttonStyles.tsx` / `pageTitles.ts` / `routes.ts` / `storage.ts` / `upload.ts`）——新文件直接落此处，无目录创建问题。

**内容范本（迁移源头的单一拷贝）**——`src/pages/system/user/constants.ts:11-27`：

```typescript
export const GENDER_OPTIONS: SelectOption[] = [
  { label: "男", value: 0 },
  { label: "女", value: 1 },
  { label: "保密", value: 2 },
];

export const STATUS_OPTIONS: SelectOption[] = [
  { label: "启用", value: 0 },
  { label: "禁用", value: 1 },
];

export const STATUS_TAG_CONFIG: Record<number, { text: string; color: string }> = {
  0: { text: "启用", color: "success" },
  1: { text: "禁用", color: "error" },
};
```

**label 漂移实证（本模块要消除的三份拷贝）**：`system/role/constants.ts:17-20` 用「正常/停用」、`system/dict/constants.tsx:10-20` 用「启用/禁用」——status.ts 落定时需对齐 models 常量的中文注释（UserStatus=启用/禁用、RoleStatus=正常/停用按 base.go 注释），并在常量旁注释对应后端常量名（`// 对齐 models.UserStatusEnabled=0`）。各页迁移后保留原 OPTIONS 导出作静态 fallback（Pitfall 4：不删除只降级）。

**测试范本**——`src/components/network/port-write/__tests__/constants.test.ts:1-53`（vitest 纯常量自洽测试：describe/it + `expect` 断言 label 非空、value 唯一、数值锁定）。status.test.ts 断言：STATUS_OPTIONS 覆盖 0/1、与 STATUS_TAG_CONFIG 键一致、label 与后端常量注释对齐的快照。

---

### 前端 DICT-03 批 2-3：各页 constants → useDict（~29 文件中的 type 类子集）

**Analog:** `xingran-react-frontend/src/pages/operations/dedicated-lines/index.tsx`（Wave 5 成品，唯一完整范本）

**hook 消费（:91-93）**：

```typescript
// Wave 5: useDict replaces raw post() calls for both line type + ISP dicts
const { data: lineTypeDict = [] } = useDict("ops_dedicated_line_type");
const { data: ispDict = [] } = useDict("ops_isp");
```

**isDefault 默认值（:241-244）**：

```typescript
const defaultType =
  lineTypeDict.find((d) => d.isDefault)?.dictValue || lineTypeDict[0]?.dictValue;
const defaultIsp = ispDict.find((d) => d.isDefault)?.dictValue || ispDict[0]?.dictValue;
lineForm.setFieldsValue({ status: 0, lineType: defaultType, isp: defaultIsp });
```

**dictLabel 回退（:253-261）**：

```typescript
const getLineTypeText = (type: string) => {
  const dictItem = lineTypeDict.find((d) => d.dictValue === type);
  return dictItem?.dictLabel || type;
};
```

**Option 渲染（:593-596）**：

```typescript
{lineTypeDict.map((d) => (
  <Option key={d.dictValue} value={d.dictValue}>
```

**hook 本体（零改动，`src/hooks/useDict.ts`）**：`DictItem` 接口（dictValue: string / isDefault?: boolean）已在 :16-29 定义；无内置静态 fallback（:53 `result.data?.list ?? []`），故迁移页必须自带 `?? 原静态数组` 兜底。

**批 2-3 迁移目标清单锚点**（RESEARCH A5/Batch 表）：user 页 GENDER_OPTIONS → `sys_user_sex`；workorder/orders、duty 三页、knowledge/articles、network 六个 constants 文件中的 type/原因类 OPTIONS → 对应 dictType（DICT-02 seed 落地后才可用，顺序依赖：DICT-02 → 批 2/3）。

---

### `CLAUDE.md`（改：DICT-04）

**无代码范本。** 改写目标段即项目 CLAUDE.md 现行 `### Status Value Convention (IMPORTANT)` 段（通用规则一句 + Menu visible 例外 + 6 行模块值表格）。改写为：指向 `internal/models/base.go`（具名常量）+ `sys_dict`（可运营枚举）+ 保留 Menu 例外并引用 `models.VisibleShow/VisibleHidden`；删除 6 行值表格。verify：grep CLAUDE.md 确认无该表格残留。

---

## Shared Patterns

### 幂等 seed（组级查重 + 事务 + 失败不阻断）
**Source:** `internal/core/db/migrations/migration_203_connection_pool_sysconfig.go:45-66`（count 查重）+ `migration_207_menu_catalog_seed.go:54-98`（快速路径 + Transaction + WARN 语义注释）
**Apply to:** migration_208_dict_seed.go、其 test 的「尊重用户删除意图」用例

### 迁移双分支注册（PG advisory-lock 块 + SQLite else 块）
**Source:** `internal/core/db/database.go:837-847`
**Apply to:** migration_208 注册（仅此一处修改点，两分支各一次）

### raw SQL 参数化常量
**Source:** `internal/services/system/department_cache_impl.go:65` — `query = query.Where("status = ?", models.DeptStatusNormal)`
**Apply to:** DICT-01 全部批次的 Where 替换；CASE WHEN 聚合用 `fmt.Sprintf("... %d ...", int(models.Xxx))`

### 常量定义风格（type + const 块 + 行尾中文 label）
**Source:** `internal/models/base.go:44-135`（system 族）、`internal/models/operations/server_room.go:7-13`（operations 子包族）
**Apply to:** 补缺实体的新常量；两处风格一致，按 model 文件所在包择一

### 常量锁值 regression test
**Source:** `internal/utils/operlog/regression_test.go:45-105, 229-282`（期望 map + AST 解析 + 双向断言 + 数量锁）
**Apply to:** `internal/models/status_constants_test.go`

### useDict 消费三件套（渲染 + 回退 + isDefault 默认）
**Source:** `xingran-react-frontend/src/pages/operations/dedicated-lines/index.tsx:92-93, 241-244, 253-261, 593-609`
**Apply to:** DICT-03 批 2-3 所有迁移页（配 `?? 原静态数组` 兜底）

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `scripts/check-status-literals.*` | utility | file-I/O | 仓内无「源码 grep 白名单守护脚本」先例；最近的 `scripts/verify/format_unify/main.go` 是 DB 数据校验器（仅可借其 Go 入口骨架：包注释写验证范围 + `go run ./scripts/...` 用法 + CheckResult 结构）。建议按 RESEARCH A1 的 grep 模式自写（bash 最简），白名单排除 `*_test.go`、注释、F 簇 `geocoding_service.go` |
| `CLAUDE.md` DICT-04 改写 | config（文档） | — | 纯文档指针化改写，无代码范本；参照物是其自身现行文本 + RESEARCH DICT-04 实现路径 |

## Metadata

**Analog search scope:** `internal/models/`（含 `operations/` 子包）、`internal/services/`（system/operations/addomain）、`internal/api/v1/`、`internal/core/db/`（database.go + migrations/ + archive/）、`internal/utils/operlog/`、`scripts/`、`xingran-react-frontend/src/{constants,hooks,pages,components}`
**Files scanned:** 约 25 个（全部实读或精确行号验证；Grep 工具异常，改用 bash grep 等效验证）
**Pattern extraction date:** 2026-08-19
**给 planner 的两个本 mapping 新增事实（RESEARCH 未记）：**
1. `RoomStatus` 等常量在 `internal/models/operations.go`（package models）与 `internal/models/operations/`（package operations）**双定义**——批 2 替换须按 model struct 实际所在包引用，禁止第三份拷贝。
2. `src/constants/` 目录**已存在**（5 个文件）——status.ts 不是新建目录，直接落入即可。
