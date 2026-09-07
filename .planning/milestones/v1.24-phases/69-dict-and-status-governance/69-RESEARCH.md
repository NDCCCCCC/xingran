# Phase 69: 字典与状态值治理（状态语义单一真相源） - Research

**Researched:** 2026-08-19
**Domain:** Go 后端常量治理 + GORM 字典 seed 迁移 + React 前端字典消费（纯仓内重构，无新外部依赖）
**Confidence:** HIGH（全部结论来自本会话 grep / 读码 / dev 库实测，非训练数据推测）

## Summary

本 phase 的三处真相源拷贝问题全部得到实证量化，且有一个**修正 ROADMAP 审计的关键发现**：dev 数据库（`data/xingran.db`，2026-08-17 由 Supabase PG 切换而来）的 `sys_dict_type` / `sys_dict_data` 表**完全为空（0 行 / 0 行，实测 2026-08-19 12:09）**。ROADMAP 所记「sys_dict seed 仅 network_device_type 一类」描述的是旧 PG 库的残留状态——全部字典 seed（network_device_type、ops_info_point_type、ops_dedicated_line_type、ops_isp、dashboard 系列、asset_reconciliation 系列）只存在于 `archive/` 迁移中，archive 不执行，Go 侧 `init_data.go` 不 seed 字典。**后果：现有 4 个 useDict 消费页在 dev 环境下拉框为空（仅靠 `?? []` 兜底不崩）。** DICT-02 因此从「补充新枚举」升级为「重建整条字典 seed 链路」。

DICT-01 的最大发现是 **models 层已有 86 个枚举类型定义**（`internal/models/base.go:44-135` 已含 UserStatus/RoleStatus/MenuStatus/DeptStatus/PostStatus/VisibleType/Gender/DataScope/MenuType/ConfigType/ConfigIsSystem 全套具名常量，且注释带中文 label），但消费端欠账巨大：api/v1 + services 下仍有 **102 处可执行 raw SQL `status = 0/1` 字面量（31 个文件）+ 22 处结构体字面量（9 文件）+ 6 处比较（1 处为外部 API 契约需排除）**，而既有常量仅被 11 个 system service 文件引用 24 处。**因此 DICT-01 不应新建 `internal/constants/` 平行真相源（那会制造第二份拷贝，与本 phase 目标自相矛盾），正确路径是以 `internal/models` 既有常量为真相源、补齐缺失实体的常量、逐语义簇替换字面量。**

前端侧（DICT-03）：`useDict` hook（`src/hooks/useDict.ts`，React Query 封装，5min stale/30min gc）基建完好但**无内置静态 fallback**；全前端共 **51 组硬编码 OPTIONS 数组（29 个文件）、19 处 status 0/1 选项值**；且已存在成品迁移范本——`dedicated-lines/index.tsx`（此前 Wave 5 已迁移，含 `dictItem?.dictLabel || value` 回退与 `isDefault` 默认值处理），DICT-03 照抄该模式即可。

**Primary recommendation:** DICT-01 以 models 包为常量真相源（补缺失实体常量 + operlog 式 regression_test 锁值），按「语义簇 × 模块目录」分 4-5 批替换；DICT-02 新建 migration_208 字典 seed（照抄 migration_207 幂等模式，**必须同时挂到 database.go 的 PG 与 SQLite 两个分支**，重建 9 组存量 + 新增真枚举）；DICT-03 只迁「type/category 类选项组」到 useDict，status 0/1 下拉改走共享前端常量模块（不进字典）；DICT-04 把 CLAUDE.md 状态表改写为指向 models 常量 + sys_dict 的引用。

---

## 现状审计数据（规划直接引用）

### A1. 后端裸 0/1 字面量分布（DICT-01 替换面）

统计口径：`internal/api/v1/` + `internal/services/`，排除 `*_test.go`，grep 模式实测（2026-08-19）：

| 形态 | 处数 | 说明 |
|------|------|------|
| raw SQL `status = 0/1`（字符串内） | **102**（另 6 处纯注释） | 最大簇；`Where("status = 0")`、`SUM(CASE WHEN status = 0 ...)` 等 |
| 结构体字面量 `Status: 0/1` | **22** | 多为创建默认「正常」态 |
| 比较 `Status == 0/1` / `!=` | **6** | 其中 `geocoding_service.go:332` 是百度 API 响应码（外部契约，**排除**） |
| map/JSON `"status": 0/1` | 2 | — |

**raw SQL 字面量文件分布（Top，全部 31 文件）**：

| 文件 | 处数 | | 文件 | 处数 |
|------|------|-|------|------|
| `internal/services/vdi/vm_service_impl.go` | 7 | | `internal/services/operations/`（building/floor/workstation/server_room/room_device/infopoint/dedicated_line 各一） | 4×7 |
| `internal/services/operations/asset_service.go` | 6 | | `internal/services/system/`（role/post/dict service） | 4×3 |
| `internal/services/workorder/{base,reconciliation_template}.go` | 4+4 | | `internal/services/scheduler/job_log_service.go` | 4 |
| `internal/services/knowledge_service.go` | 4 | | `internal/services/duty_pool_service.go` | 4 |
| `internal/services/device_discovery_service.go` | 4 | | `internal/services/config_execution_service.go` | 4 |
| `internal/services/addomain/account_pool.go` | 3 | | `internal/api/v1/operations/excel_handler.go` | 2 |
| 其余（user_service/notice/rpa/asset/addomain 等 8 文件） | 1-2 | | `internal/services/command_dispatch_service.go` | 2 |

**结构体字面量分布**：`workstation_device_service.go` 独占 11 处，其余 11 处散布在 `scheduler/job_service`、`monitor/server_service`、`api/v1/system/notice_handler`、`rpa/credential_service`、`notice_service`、`asset/fix_suggestion_monitor`、`addomain/user_ou_service`、`api/v1/monitor/oper_log_handler`。

### A2. 语义簇划分（DICT-01 安全替换的关键——绝不能一刀切）

`status = 0/1` 在不同实体上是**不同语义**，逐簇映射到正确常量家族：

| 簇 | 语义 | 涉及实体/文件 | 替换目标常量 |
|----|------|--------------|-------------|
| **A 通用启用/停用** | 0=正常/启用, 1=停用 | dept、role、post、dict、user、building、floor、workstation、server_room、room_device、infopoint、notice、job、vdi_server、ad_config | `models.DeptStatusNormal` / `RoleStatusEnabled` / `PostStatusEnabled` 等（**已存在**，base.go + 各 model 文件） |
| **B 任务状态机** | 0=待执行, 1=执行中, 2=成功, 3=失败, 4=取消 | config_execution、device_discovery、command_dispatch | `models.ExecutionStatusPending/Running/...`（**已存在**，config_execution.go:19-27——且这些文件里 struct 赋值已用常量、raw SQL CASE WHEN 仍用字面量，属同文件半迁移状态） |
| **C 成功/失败** | 0=成功/正常, 1=失败/异常 | oper_log（`oper_log_handler.go:190`）、job_log、`asset/fix_suggestion_monitor.go:181`（`Status: 0, // 0=成功`） | 需查/补：log.go 有 `JobStatus`；OperLog 状态常量可能缺，需新增 |
| **D 多态（3+ 值）** | 0/1/2 | dedicated_line（0=正常/1=故障/2=停用，见 `dedicated-lines/index.tsx:264` 前端映射）、AD 账号池（0=可用/1=停用/2=熔断，account_pool.go）、DeviceStatus（0=在线/1=离线/2=未知） | 逐实体常量（部分已存在） |
| **E 语义反转** | **1=正向** | KnowledgeArticleStatus（0=草稿/1=已发布）、Notice PublishStatus（0=草稿/1=已发布/2=定时/3=撤回） | `models.KnowledgeArticleStatusPublished` 等（**已存在**）——**严禁**套用「1=停用」 |
| **F 外部契约（排除）** | 第三方 API 状态码 | `geocoding_service.go:332`（百度返回码） | 不替换，加注释说明即可 |

### A3. 既有常量盘点（DICT-01 的真相源基础）

- `internal/models/base.go:44-135`：**11 组**带中文注释的枚举常量已存在——Gender(0/1/2)、UserStatus、RoleStatus、DataScope(1-5)、MenuType(M/C/F)、**VisibleType（1=显示/0=隐藏，即 CLAUDE.md 记载的 Menu 例外语义的代码化身）**、MenuStatus、DeptStatus、PostStatus、ConfigType(Y/N)、ConfigIsSystem(0/1)
- models 层全量 **86 个枚举 type 定义**（grep `^type.*(Status|State|Type|Level|Scope|Category)`），重点文件：`network_device.go`（DeviceType 5 值带 label、DeviceStatus 3 态）、`workorder.go`（WorkOrderStatus 0-4、WorkOrderType fault/request/change）、`config_execution.go`（ExecutionStatus 0-4）、`duty.go`（DutyStatus、HolidayType、DutyPoolStatus）、`knowledge.go`、`notice_enhanced.go`（PublishStatus、TargetType）、`workstation.go`、`rpa.go`、`dashboard.go`
- **既有采用率**：仅 11 个文件 24 处引用（`internal/api/v1/auth.go` + `internal/services/system/` 的 user/role/menu/dept/post 的 service 与 cache_impl）——核心 system 模块已部分采用，外围模块零采用
- **缺口**：Building/Floor/ServerRoom/RoomDevice/InfoPoint/VDIServer/Notice/Job 等实体的 status 常量是否齐全需在执行时逐 model 核对（operations 目录下 model 有部分定义如 `WorkstationStatus`，未逐一验证每个实体都有）
- **常量组织先例**：`internal/utils/operlog/`（444 行主文件 + 362 行 `regression_test.go` 锁定常量值/数量/签名）——「常量集 + 回归测试锁值」是项目已验证的模式，DICT-01 照抄

### A4. sys_dict 基础设施现状（DICT-02 素材）

- 模型：`internal/models/dict.go`——`DictType`（DictName / **DictType uniqueIndex** / Status / Remark）、`DictData`（DictSort / DictLabel / **DictValue(string)** / DictType / CssClass / **ListClass(tag 颜色)** / **IsDefault** / Status / Remark）；两张表已在 AutoMigrate 注册（`database_test.go:192` 实证），**无需新表、无 sqlite 缺表风险**
- API：`internal/api/v1/system/dict_router.go` 全套 CRUD + `/list` + `/all` + `/statistics` 已挂载；前端字典管理页 `src/pages/system/dict/` 完整可用
- **后端消费为零**：`GetDictDataByTypeKey` 仅 `dict_cache_impl.go` 自用 + `cache_keys.go` 定义（grep 实证 0 个业务 service 调用）
- **dev 库字典表为空**（本研究的最大发现，见 Summary）：`sqlite3 data/xingran.db` 实测 `sys_dict_type=0 行`、`sys_dict_data=0 行`，库文件 2026-08-19 11:14 仍在活跃写入
- **历史 seed 全部在 archive**（不执行）：`archive/legacy-2026-06-15/002`（network_device_type 5 值）、`033`（ops_info_point_type）、`047`（ops_dedicated_line_type）、`048`（ops_isp）、`045/050`（dashboard_widget_type / dashboard_template_scope / dashboard_scope）、`archive/applied/migration_169`（asset_reconciliation_{conflict_type,exception_action,severity,status} 4 组）+ `migration_196`（label 对齐）
- **seed 命名规范**（archive 与前端 key 实测一致）：operations 模块 `ops_` 前缀 snake_case（`ops_dedicated_line_type`/`ops_isp`/`ops_info_point_type`），其他模块直接语义名（`network_device_type`、`asset_reconciliation_*`）；dict_value 为稳定字符串码（router/internet/telecom）
- **seed 幂等模式范本**：
  - `migration_207_menu_catalog_seed.go`——go:embed SQL + 快速路径（根存在即跳过，**尊重用户删除意图**）+ 行级 `ON CONFLICT DO NOTHING` + 单事务 + 失败 WARN 不阻断启动 + 双方言（无 PG 专有语法，SQLite ≥3.24 支持 ON CONFLICT）+ 配套 `_test.go`
  - `migration_203_connection_pool_sysconfig.go`——GORM 风格：count 查重 + `models.Config{}` 结构体 seed + model 常量引用；**对字典 seed 更合适**（字段多、类型安全）

### A5. 前端现状（DICT-03 素材）

- `useDict(dictType)`：`src/hooks/useDict.ts`——React Query，`queryKeys.dict.list(type)` 共享缓存，POST `/system/dicts/data/list` pageSize 1000，staleTime 5min / gcTime 30min / 不 refetch-on-focus；`useInvalidateDict()` 全局失效。**无内置静态 fallback**——消费方自行 `?? []`
- **现消费方（与 ROADMAP「4 页」吻合：3 页 + 1 组件，共 5 个 dictType）**：
  - `pages/operations/dedicated-lines/index.tsx:92-93`（ops_dedicated_line_type + ops_isp）
  - `pages/operations/info-points/index.tsx:125`（ops_info_point_type）
  - `pages/asset/reconciliation/exceptions/index.tsx:124-125`（conflict_type + severity）
  - `components/reconciliation/HealthBadge.tsx:44`（conflict_type）
- **迁移成品范本**（dedicated-lines Wave 5 留下）：`const { data: lineTypeDict = [] } = useDict(...)` → `.map((d) => <Option>)` 直渲染；`getLineTypeText = dictItem?.dictLabel || type` 值回退；`d.isDefault` 优先作表单默认值
- **硬编码 options 总量**：51 组 `OPTIONS = [...]`（29 文件，`src/pages/**/constants.ts(x)` 页级常量文件为主）+ 19 处页内 status 0/1 选项值 + 12 处「正常/停用」字面 label；**各页常量是独立拷贝，无共享模块**（无 import 扇出可依，批次按模块目录划分）
- **label 漂移实证**（同一 0/1 值三种文案）：`user/constants.ts` 启用/禁用、`role/constants.ts` 正常/停用、`system/dict/constants.tsx` 启用/禁用——CLAUDE.md 表格记 User=Enabled/Disabled、Role=Normal/Stopped，三处已经对不齐
- 页面总数：74 个 `pages/**/index.tsx`（ROADMAP「~78 页」口径含子页组件）
- 测试基建：vitest 4.x + @testing-library/react（`vitest.config.ts`、`npm run test`），已有 `port-write/constants.test.ts` 这类纯常量单测先例

### A6. 字典语义的第三、四处隐藏拷贝（规划需知）

- `internal/services/operations/excel_config.go:146`（workstationType 3 值 map，引用 models 常量）与 `:265`（lineType **6 值裸字符串 map** `"internet"/"intranet"/"cloud_desktop"/"mpls"/"fiber"/"leased_line"`）——Excel 导入的 label↔value 转换表与字典/模型三重同步
- `excel_service.go:206-213` 模板示例值硬编码「固定工位」「internet」「telecom」
- 这两处是 DICT-02 枚举值的现成证据来源；是否反向改为读字典属大重构，建议**只抄值不改造**（见 Open Questions）

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| status 语义常量（0/1/多态）真相源 | 后端 `internal/models`（枚举 type + 常量） | 前端共享常量模块（镜像） | 语义由 DB schema 与后端业务逻辑决定；DB 字面量出现在后端 raw SQL |
| type/category 可运营枚举选项真相源 | DB `sys_dict_type/sys_dict_data`（字典管理页可维护） | 后端 models 枚举常量（代码级下限校验） | 管理员需要不改代码调整选项 label/增删项；代码仍需常量做逻辑分支 |
| 字典读取 API | 后端既有 `/system/dicts/data/list` | — | 已建成，零改动 |
| 字典缓存 | 后端 dict_cache_impl（Redis）+ 前端 React Query 双层 | — | 已建成，零改动 |
| 下拉 options 渲染 | 前端页面组件 | `useDict` hook | DICT-03 逐页迁移 |
| 文档（Status Value Convention） | CLAUDE.md → 改为指针 | — | 不再承载值语义本体 |

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| DICT-01 | 后端新增集中状态常量包，消灭 50+ 文件裸 0/1 字面量 | A1/A2/A3：实际为 31+9+6 处分布；models 层 86 个枚举类型已存在，11 组核心常量齐备；建议以 models 为真相源补缺 + 按语义簇 A-E 分批替换（F 排除）；operlog regression_test 模式锁定 |
| DICT-02 | 盘点 type/category 真枚举字段并 seed 进 sys_dict | A4/A6：dev 库字典表为空（0/0）需重建全部 seed；migration_203(GORM 式)/207(embed SQL 式) 双范本；9 组存量 + 新候选清单（见 Architecture Patterns 候选表）；必须挂 database.go PG+SQLite 双分支 |
| DICT-03 | 前端 constants.tsx 硬编码 options 分批迁移 useDict | A5：51 组 OPTIONS/29 文件清单；dedicated-lines Wave 5 成品范本（dictLabel 回退 + isDefault）；批次按模块目录（无共享 import 扇出）；useDict 无内置 fallback，需页面级静态兜底 |
| DICT-04 | CLAUDE.md Status Value Convention 改指向常量真相源 | A3/A5：现行段落是第三份手工拷贝且已与前端 label 漂移（启用/禁用 vs 正常/停用）；改写为指向 models 常量 + sys_dict；VisibleType 例外语义保留但引用代码常量 |
</phase_requirements>

---

## Standard Stack

**本 phase 零新增依赖**——纯仓内重构 + seed 数据 + 测试。后端 Go 1.24 标准库 + GORM + 既有 testing；前端 React Query（已装）+ vitest（已装）。

### Core（复用既有）

| 库/设施 | 版本 | 用途 | 说明 |
|---------|------|------|------|
| GORM | 仓内既有 | migration_208 seed（DictType/DictData 结构体插入） | migration_203 同款 count 查重模式 |
| go:embed + raw SQL | 标准库 | 备选：大量 INSERT 时照 migration_207 | 双方言注意无 PG 专有语法 |
| @tanstack/react-query | 仓内既有 | useDict 底层 | 零改动 |
| vitest | 4.x | 前端常量/fallback 断言 | `port-write/constants.test.ts` 先例 |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| models 包作常量真相源 | 新建 `internal/constants/` 平行包 | SC#1 原文是「如 internal/constants/」（举例非强制）；平行包 = 第二真相源，与「单一真相源」目标矛盾；**除非做纯 re-export 门面否则不推荐** |
| GORM 结构体 seed | embed SQL 批量 INSERT | 字典条目多（9 组 × 若干值）时 SQL 更紧凑，但 GORM 式类型安全 + 直接引用 models 常量 + 单条幂等更清晰；两组混合亦可（type 用 GORM、data 用 GORM 循环） |

## Package Legitimacy Audit

无需安装任何外部包（纯仓内重构）——审计不适用，无 [SLOP]/[SUS] 风险项。

## Architecture Patterns

### System Architecture Diagram

```
                    ┌──────────────── 真相源层 ────────────────┐
                    │                                           │
  [DICT-01]         ▼                    [DICT-02]              ▼
  internal/models 枚举常量          migration_208 字典 seed   sys_dict_type
  (UserStatus/DeptStatus/...)  ──►  (幂等,PG+SQLite 双分支)   sys_dict_data
        ▲   ▲                                                  (dev 库当前 0/0 行)
        │   │                                                        │
  替换  │   │ regression_test                                        │ 既有 API(零改动)
  102+22+6│   │ 锁值                                                 ▼
  处字面量│   └──────────┐                            POST /system/dicts/data/list
        │                │                                        │
  ┌─────┴────┐   ┌──────┴───────┐                        ┌───────┴────────┐
  │ 后端     │   │ CLAUDE.md    │                        │ 前端           │
  │ services │   │ [DICT-04]    │                        │ [DICT-03]      │
  │ handlers │   │ 改为指针引用 │                        │ useDict()      │
  └──────────┘   └──────────────┘                        │ +静态fallback  │
                                                   51 组 OPTIONS 分批迁移
                                                         └────────────────┘
```

### DICT-01 实现路径（含批次划分依据）

**形状决策：以 `internal/models` 为唯一真相源。**
1. 核对/补齐：逐 model 检查簇 A 实体是否都有常量（building/floor/server_room/room_device/infopoint/notice/job/vdi_server 等缺则在各自 model 文件按 base.go 风格补 `type XxxStatus int` + 常量 + 中文注释；OperLog 成功/失败若缺则补）
2. 替换（raw SQL 改参数化：`Where("status = ?", models.DeptStatusNormal)`；CASE WHEN 用 `fmt.Sprintf("SUM(CASE WHEN status = %d THEN 1 ELSE 0 END)", int(models.XxxPending))`）
3. 锁定：`internal/models/status_constants_test.go`（operlog regression_test 模式）断言关键常量值与数量（UserStatusEnabled=0 等），防止后人改值

**批次依据 = 语义簇 × 模块目录**（每批独立 commit、`go build ./...` 常绿）：

| 批 | 范围 | 文件数 | 语义簇 | 风险 |
|----|------|--------|--------|------|
| 1 | `internal/services/system/`（role/post/dict/user service 残余）+ 既有常量已采用文件的收尾 | ~5 | A | 低（同文件已有常量先例） |
| 2 | `internal/services/operations/` 7 个 service + excel_handler + asset_service | ~10 | A + D(专线三态) | 中（workstation_device 11 处结构体字面量） |
| 3 | `internal/services/{vdi,addomain,notice,knowledge,rpa,scheduler}` | ~8 | A + D(账号池三态) + E(knowledge 反转) | 中 |
| 4 | `internal/services/{workorder,duty_pool,device_discovery,config_execution,command_dispatch,monitor}` + `api/v1/{system/notice,monitor/oper_log}` | ~10 | B + C + A | 中高（状态机 CASE WHEN 多） |
| 5 | 常量补齐收尾 + regression_test + F 簇注释 | 2-3 | — | 低 |

### DICT-02 实现路径

1. **候选枚举清单**（分两档）：

| 档 | dictType | 值来源证据 | 备注 |
|----|----------|-----------|------|
| 重建（dev 库已丢） | network_device_type | archive/002 SQL + `models.DeviceType*` 5 常量 | router/switch/firewall/ap/loadbalancer |
| 重建 | ops_info_point_type / ops_dedicated_line_type / ops_isp | archive/033/047/048 SQL + 前端 key 实测一致 | 047 六值见 excel_config.go:265 |
| 重建 | asset_reconciliation_{conflict_type,exception_action,severity,status} | archive/applied/migration_169 | HealthBadge/exceptions 页正在消费（当前空数据） |
| 重建（可选） | dashboard_widget_type / dashboard_template_scope / dashboard_scope | archive/045/050 | 前端是否消费需执行时确认，未见 useDict 引用 |
| **新增** | ops_workstation_type | excel_config.go:146 + `models.WorkstationType{Fixed,HotDesk,Manager}` | 固定/灵活/管理工位 |
| **新增** | workorder_type / workorder_priority | `models.WorkOrderType`(fault/request/change...) + WorkOrderPriority | 工单筛选下拉 |
| **新增** | sys_user_sex | useDict.ts:4 docstring 本来就以它举例；`models.Gender` 0/1/2 | 前端 GENDER_OPTIONS 迁移目标 |
| 新增（候选） | notice_target_type / duty_holiday_type | `models.TargetType` / `models.HolidayType`(legal/workday/custom) | 值少收益中等，planner 取舍 |

2. **seed 迁移**：`migration_208_dict_seed.go`（GORM 式，migration_203 模式）：逐 dictType count 查重（`sys_dict_type` 存在即跳过该组——尊重管理员后续增删）→ 事务内 Create DictType + DictData 循环 → 失败 WARN 不阻断。**注册到 `database.go` 两处**：PG 分支 advisory-lock 块内 + SQLite else 分支（`config.yaml:16` 已文档化「sqlite 模式下 202-206 seed 不执行」是已知缺口——208 决不能重蹈）
3. 配套 `migration_208_dict_seed_test.go`（migration_207 test 模式）：sqlite 内存库跑 seed → 断言行数 → 再跑一遍断言幂等（行数不变）

### DICT-03 实现路径（批次 = 模块目录，按「字典依赖已就绪」排序）

| 批 | 页面/组件 | 迁移项 | 前置 |
|----|-----------|--------|------|
| 1（修现状） | dedicated-lines / info-points / reconciliation exceptions / HealthBadge | 无需迁移，**DICT-02 seed 落地后下拉自动恢复**——作为字典链路的端到端验证点 | DICT-02 |
| 2 | user（GENDER_OPTIONS → sys_user_sex）、workstation/workorders 等含 type 下拉的页 | type/性别类 options → useDict + `?? 静态数组` 兜底 | DICT-02 |
| 3 | duty holidays/management/schedules、knowledge、network 系列 constants | type/原因类 options → useDict | DICT-02 |
| 4 | **status 0/1 下拉不进字典**：新建 `src/constants/status.ts` 共享模块（STATUS_OPTIONS/STATUS_TAG_CONFIG 单一拷贝，值与后端常量对齐注释）替换各页独立拷贝 | 消除「启用/禁用 vs 正常/停用」漂移 | 无 |

**迁移模板（dedicated-lines 实证模式）**：
```typescript
const { data: typeDict = [] } = useDict("ops_workstation_type");
// 渲染: typeDict.map((d) => <Option key={d.dictValue} value={d.dictValue}>{d.dictLabel}</Option>)
// 回退: const text = typeDict.find((d) => d.dictValue === v)?.dictLabel ?? STATIC[v] ?? v;
// 默认值: typeDict.find((d) => d.isDefault)?.dictValue ?? typeDict[0]?.dictValue
```

### DICT-04 实现路径

CLAUDE.md `### Status Value Convention (IMPORTANT)` 段改写为：通用规则一句话 + 指向 `internal/models/base.go`（具名常量）与 `sys_dict`（可运营枚举）+ 保留 Menu visible 例外说明（引用 `models.VisibleShow/VisibleHidden`）。删除 6 行模块值表格（那份表格是漂移源头）。同步更新 `## Common Gotchas` 无需动（其引用的是规则不是表格）。

### Anti-Patterns to Avoid

- **新建 `internal/constants/` 平行常量包**：制造第二真相源，与 phase 目标自相矛盾（除非纯 re-export 门面且有充分理由）
- **一刀切全局 StatusEnabled/StatusDisabled**：簇 B/C/D/E 语义不同（1 可能是「执行中」「已发布」「故障」），盲替必产 bug
- **把 status 0/1 也 seed 进字典供下拉**：status 是代码分支语义（if/switch 依赖），进字典后管理员改值即破坏逻辑；字典只放 type/category 类「运营可维护选项」
- **seed 只挂 PG 分支**：dev 是 SQLite，会立刻在 dev 不可见（config.yaml:16 已有前车之鉴注释）

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| 字典缓存/失效 | 页面自建 state + useEffect 拉字典 | 既有 `useDict` + `useInvalidateDict` | React Query 缓存/失效已处理好 |
| 字典 CRUD API/管理页 | 新端点 | 既有 `/system/dicts/*` + `pages/system/dict` | 完整可用 |
| 幂等 seed 事务 | 自写 INSERT OR IGNORE 方言分支 | migration_203/207 既有模式 | 双方言已验证 |
| 常量防回归 | 依赖 code review | regression_test 锁值（operlog 先例） | 362 行测试锁 25 常量的成熟做法 |

**Key insight:** 本 phase 全部是「接通既有基础设施」而非新建——常量、字典 API、管理页、useDict、seed 模式五件事都已存在，欠的只是消费与数据。

## Runtime State Inventory

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | `data/xingran.db` sys_dict_type=0 行 / sys_dict_data=0 行（2026-08-19 实测）；dev 库 11:14 活跃写入中 | migration_208 seed（数据迁移，非代码编辑）；旧 PG(Supabase) 库已弃，无其他字典数据源 |
| Live service config | 无——字典不存在于任何外部服务 | — |
| OS-registered state | 无 | — |
| Secrets/env vars | 无——字面量替换不触及任何 env key | — |
| Build artifacts | 无新表/无列变更（DictType/DictData 已注册 AutoMigrate，database_test.go:192）；seed 纯 INSERT 不触发 GORM ALTER → **sqlite 视图依赖 FATA 坑（memory 记录）不适用** | — |

## Common Pitfalls

### Pitfall 1: 迁移只挂 PG 分支，dev（SQLite）看不到
**What goes wrong:** migration_208 写进 `database.go` PG advisory-lock 块，dev sqlite 分支没挂 → 字典管理页依旧空，验收直接失败。
**Why:** database.go:806-857 的 if/else 双分支结构；config.yaml:16 注释已记录 202-206 在 sqlite 不执行的先例。
**How to avoid:** 207 的挂法（PG 块内 + else 分支各一次）是标准答案，照抄。
**Warning signs:** dev 启动日志无 "migration 208" 字样。

### Pitfall 2: 语义簇混替（E 簇反转 / B 簇状态机）
**What goes wrong:** 把 `KnowledgeArticleStatus: 1`（已发布）换成 Disabled 类常量，或把 `CASE WHEN status = 1`（执行中）换成 Stop 常量。
**How to avoid:** 每文件替换前先读该 model 的常量注释；A2 簇表作为替换对照表写进 plan 任务描述；geocoding 的百度 status 明确排除。
**Warning signs:** knowledge/notice/publish 相关 diff 出现 Normal/Stop 字样；config_execution diff 出现 Enabled 字样。

### Pitfall 3: seed 不幂等或复活管理员删除项
**What goes wrong:** 每次重启重复插 dict_data；或管理员删掉的选项重启后回来。
**How to avoid:** 组级 count 查重（dictType 存在整组跳过，207 的「尊重用户删除意图」语义）+ 组内行级查重双保险；事务包裹。
**Warning signs:** 重启后 dict_data 行数增长。

### Pitfall 4: 前端迁移后字典接口异常 → 下拉空白
**What goes wrong:** 直接 `data.map(...)` 无兜底，接口 500 时 Select 无选项。
**How to avoid:** 统一 `?? 静态数组` 模式（DICT-03 SC 明确要求保留静态 fallback）；静态数组即迁移前原 OPTIONS 常量，不删除只作 fallback。
**Warning signs:** 迁移 commit 删掉了原 OPTIONS 导出。

### Pitfall 5: raw SQL 字符串拼接注入风格
**What goes wrong:** 用 `fmt.Sprintf("status = %d", ...)` 拼简单 Where（可用但非首选）扩散。
**How to avoid:** Where 条件一律参数化 `Where("status = ?", int(models.Xxx))`；仅 CASE WHEN 聚合等无法参数化处用 Sprintf+常量。

### Pitfall 6: excel_config/模板示例值与字典漂移
**What goes wrong:** DICT-02 改了字典值但 excel_config.go:265 的 6 值 map 与 excel_service.go 示例值没动，导入校验静默失配。
**How to avoid:** 本 phase 只保证值一致（seed 值从 excel_config 抄），不改造 excel 链路读字典（见 Open Questions Q3）。

## Code Examples

### DICT-01 替换示例（raw SQL → 参数化常量）
```go
// Source: 仓内实证 internal/services/addomain/dept_sync_service.go:168
// Before:
Where("status = 0"). // 0=正常
// After:
Where("status = ?", int(models.DeptStatusNormal))

// Source: 仓内实证 internal/services/duty_pool_service.go:39 (CASE WHEN)
// After:
fmt.Sprintf("SUM(CASE WHEN status = %d THEN 1 ELSE 0 END) AS enabled", int(models.DutyPoolStatusEnabled))
```

### DICT-02 seed 骨架（migration_203 模式 + 207 幂等语义）
```go
// Source: internal/core/db/migrations/migration_203_connection_pool_sysconfig.go (模式实证)
func Migrate208DictSeed(db *gorm.DB) error {
	for _, group := range dictSeedGroups { // []struct{Type models.DictType; Data []models.DictData}
		var n int64
		if err := db.Model(&models.DictType{}).Where("dict_type = ?", group.Type.DictType).Count(&n).Error; err != nil {
			return err
		}
		if n > 0 { continue } // 组级幂等 + 尊重管理员增删
		if err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Create(&group.Type).Error; err != nil { return err }
			return tx.Create(&group.Data).Error
		}); err != nil { return err }
	}
	return nil
}
// 注册: database.go PG 分支(advisory-lock 块内) + sqlite else 分支 各一次(207 挂法)
```

### DICT-03 迁移模板（dedicated-lines 实证）
```typescript
// Source: xingran-react-frontend/src/pages/operations/dedicated-lines/index.tsx:92-96,242,254
const { data: typeDict = [] } = useDict("ops_workstation_type");
const text = typeDict.find((d) => d.dictValue === v)?.dictLabel ?? STATIC_TYPE_MAP[v] ?? v; // 静态 fallback
const def = typeDict.find((d) => d.isDefault)?.dictValue ?? typeDict[0]?.dictValue;
```

## State of the Art

不适用（仓内治理重构，无外部技术演进依赖）。

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Building/Floor/ServerRoom/RoomDevice/InfoPoint/Notice/Job/VDIServer 等实体可能存在 status 常量缺口（仅抽查了 workstation/dict/network_device 等，未逐一核对全部 86 个枚举） | A3 / DICT-01 批 1 | 低——执行时逐 model 核对即可发现，缺则当批补 |
| A2 | dashboard_* 三组字典前端未消费（未见 useDict 引用），是否重建由 planner 取舍 | DICT-02 候选表 | 低——多 seed 无害（幂等），少 seed 无消费方 |
| A3 | 生产 PG 库的 sys_dict 数据状态未知（dev sqlite 为空是实测；生产库字典可能有历史数据）——组级幂等 seed 对两种状态都安全 | DICT-02 | 低——幂等设计已覆盖；若生产库有值且 label 与新 seed 不一致，组级跳过会保留旧值（符合「尊重用户数据」） |
| A4 | 74 个 index.tsx ≈ ROADMAP「~78 页」口径（含子组件页） | A5 | 无影响 |

## Open Questions (RESOLVED)

> 2026-08-19 规划决议回填（checker W4；决议已落地到 plan 69-01..69-08）。

1. **DICT-01 真相源位置：models 包 vs internal/constants 门面包？** — **(RESOLVED → 69-01)**：models 为唯一真相源，不新建 internal/constants 平行包；缺失实体常量（DictStatus / OperLog 成败 / VDIServer / Notice / InfoPoint）由 69-01 T1 按 base.go 风格补齐并锁值。
   - What we know: SC#1 原文「如 internal/constants/」是举例；models 已有 86 个枚举 + 11 组常量
   - What's unclear: 用户/规划是否坚持「物理集中」的形式要求
   - Recommendation: models 为真相源；若坚持 internal/constants，做成纯 re-export 别名门面（零拷贝）。planner 可在 plan 内定，不必打断用户
2. **status 0/1 下拉是否进字典？** — **(RESOLVED → 69-06)**：不进字典。status 走前端共享常量模块 `src/constants/status.ts`（69-06 落地）；type/性别/假日类才走 useDict（69-07）——status 是代码分支语义不宜运营可配（Q2 安全决策，threat T-69-13 背书）。
   - What we know: DICT-03 说「硬编码 options 迁移 useDict」未区分 status 与 type
   - Recommendation: status 走共享常量模块（批 4），type 走 useDict（批 2-3）——status 是代码分支语义不宜运营可配。若用户想要字典化 status 也可行（seed sys_user_status 等），但需接受「管理员加值可能不被代码识别」的说明成本
3. **excel_config Options map 是否改造为读字典？** — **(RESOLVED)**：本 phase 不改（静态配置 + 导入热路径，改造收益低风险高）；只保证 seed 值与 excel_config 一致（69-02 T1 硬约束）。excel_service.go:1975/:2029 的 map 形态字面量仅做常量化（69-03 批 2），不改造为读字典。留 v1.25+ 候选。
   - Recommendation: 本 phase 不改（静态配置 + 导入热路径，改造收益低风险高）；只保证 seed 值与 excel_config 一致。留 v1.25+ 候选
4. **DICT-02 新增候选的最终清单边界**（workorder/notice/duty/sys_user_sex 取哪些）— **(RESOLVED → 69-02)**：按「前端有对应下拉的优先」圈定 11 组 = 8 组 archive 存量重建 + 3 组新增（ops_workstation_type / sys_user_sex / duty_holiday_type）；dashboard_* 三组与 workorder_type/workorder_priority 剔除（前端零消费方，69-02 Source Audit 记录）。

## Environment Availability

Step 2.6: 纯仓内代码/seed/测试变更，无新外部依赖。既有关键设施实测可用：Go 工具链（仓内活跃构建）、Node 24+/vitest（package.json）、sqlite3 CLI（`/d/Program Files/Sqlite3/sqlite3`，本次审计已用它实测 dev 库）、dev 库 `data/xingran.db` 存在且活跃。

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing（后端）/ vitest 4.x + @testing-library/react（前端） |
| Config file | `vitest.config.ts`（Go 用仓内惯例，无统一 ini） |
| Quick run command | `go build ./... && go test ./internal/models/ ./internal/core/db/migrations/` |
| Full suite command | `go test ./...` / 前端 `npm run type-check && npm run lint && npm run test` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| DICT-01 | 常量值/数量不被静默修改（UserStatusEnabled=0 等） | unit(regression) | `go test ./internal/models/ -run TestStatusConstants` | ❌ Wave 0 新建 `internal/models/status_constants_test.go`（operlog regression_test 模式） |
| DICT-01 | 替换后行为不回归（抽样：dict/role/dept service 过滤逻辑） | unit | `go test ./internal/services/system/` | ✅ 既有（dict_statistics_test.go 等），替换后必须仍绿 |
| DICT-01 | 防新增裸字面量回归 | 脚本守护 | `scripts/check-status-literals.*`（grep 白名单式：注释/测试/排除清单外报错） | ❌ Wave 0 新建（可并 go:generate 或 CI 步骤；Phase 63 CI 可挂） |
| DICT-02 | seed 幂等 + 行数正确 + 双方言 | unit | `go test ./internal/core/db/migrations/ -run TestMigrate208` | ❌ Wave 0 新建 `migration_208_dict_seed_test.go`（sqlite 内存库跑两遍断言行数不变） |
| DICT-02 | dev 库字典可见（集成冒烟） | manual/SQL | `sqlite3 data/xingran.db "SELECT COUNT(*) FROM sys_dict_type"` > 0 + 字典管理页目检 | — 手测路径 |
| DICT-03 | 静态 fallback 断言（字典空时下拉仍有选项） | unit | `npx vitest run src/constants/status.test.ts`（新）+ 各批页面组件测试 | ❌ Wave 0 新建 |
| DICT-03 | 字典改值 → 下拉变化 | manual | 字典管理页改 label → 刷新消费页验证（SC#3 验收路径，配合 useInvalidateDict 免刷新亦可） | — 手测路径 |
| DICT-04 | CLAUDE.md 无独立值表 | 静态检查 | grep CLAUDE.md 无 6 行值表格（plan 内 verify 步骤） | — |

### Sampling Rate
- **Per task commit:** `go build ./...` + 受影响包 `go test ./<pkg>/`（后端批任务）；前端批任务 `npm run type-check && npx vitest run <相关文件>`
- **Per wave merge:** `go test ./...` 全绿 + `npm run test` 全绿
- **Phase gate:** 全套 + 守护脚本零新增命中 + dev 库字典管理页/4 个既有 useDict 页目检

### Wave 0 Gaps
- [ ] `internal/models/status_constants_test.go` — DICT-01 常量锁值
- [ ] `internal/core/db/migrations/migration_208_dict_seed_test.go` — DICT-02 幂等
- [ ] `scripts/check-status-literals`（语言任意：bash/mjs/go 皆可，仓内 scripts/ 有 build/deploy 等先例目录）— 防回归守护
- [ ] `xingran-react-frontend/src/constants/status.ts` + `status.test.ts` — DICT-03 批 4 共享常量与 fallback 断言

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | 本 phase 不触认证 |
| V3 Session Management | no | 不触会话 |
| V4 Access Control | no（不变更权限面） | 字典管理页既有权限维持 |
| V5 Input Validation | 间接 | DICT-01 参数化 Where（`Where("status = ?", c)`）顺带消除字符串拼接习惯；字典 dictValue 仅供 UI 展示，不进 SQL/逻辑分支（status 不进字典的设计决策即为此） |
| V6 Cryptography | no | 不触加密 |

### Known Threat Patterns for 本 phase

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| 字典值注入业务逻辑（若 status 进字典，管理员可配出代码未识别值） | Tampering | 研究建议 status 走常量不进字典；type 字典值仅作 label 展示，代码分支仍用 models 常量 |
| seed 覆盖生产数据 | Tampering | 组级幂等跳过（dictType 存在即不写）+ 事务回滚 + WARN 不阻断 |

## Sources

### Primary (HIGH confidence — 全部本会话仓内实测)
- `internal/models/base.go:44-135` — 11 组既有枚举常量
- grep 实测：raw SQL 102+6 处 / 结构体 22 处 / 比较 6 处（2026-08-19，模式见 A1）
- `sqlite3 data/xingran.db` 实测：sys_dict_type=0 / sys_dict_data=0（2026-08-19 12:09）
- `internal/core/db/database.go:806-857` — PG/SQLite 双分支迁移注册
- `internal/core/db/migrations/migration_207_menu_catalog_seed.go` / `migration_203_connection_pool_sysconfig.go` — seed 范本
- `internal/core/db/migrations/archive/`（002/033/047/048/045/050/169/196）— 历史字典 seed 与命名规范
- `xingran-react-frontend/src/hooks/useDict.ts`、`src/pages/operations/dedicated-lines/index.tsx:89-96,242-266` — hook 与迁移范本
- `internal/services/operations/excel_config.go:146,265` — 枚举值第三拷贝
- `configs/config.yaml:12-16` — sqlite 迁移缺口文档化注释

### Secondary (MEDIUM confidence)
- 无外部网络来源引用（纯仓内研究）

### Tertiary (LOW confidence)
- 无

## Metadata

**Confidence breakdown:**
- 现状审计数据: HIGH — 全部 grep/读码/dev 库实测，含精确文件行号
- DICT-01 路径: HIGH — models 常量与 operlog 先例均为仓内实证；MEDIUM 点仅在未逐 model 全量核对（A1 假设）
- DICT-02 路径: HIGH — 双范本 + 双分支坑 + 空库实测三重证据
- DICT-03 路径: HIGH — 51 组清单 + Wave 5 成品范本
- DICT-04: HIGH — 目标段落即 CLAUDE.md 现文

**Research date:** 2026-08-19
**Valid until:** 2026-09-18（仓内代码会漂移，数字需在 plan 执行批次时用同一 grep 模式复测；模式已固化在 A1 表格）
