# Phase 102: 机械常量化（缓存键 / 状态 / 分页） - Research

**Researched:** 2026-09-07
**Domain:** Go 后端行为等价重构（缓存键具名化 / status 字面量清零 / 分页口径归一）
**Confidence:** HIGH（全部位点经本会话 grep / go list / 逐文件实读核实；零外部依赖、零新包安装）

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-102-1:** 业务模块 ~35 处键**全集中注册进 `internal/services/system/cache_keys.go`**（与 user/role/menu/dept/post 键族同文件），维持「cache_keys.go = 缓存键唯一真相源」既有声明。不采用模块包内注册表、不做核心/边缘切分。
- **D-102-2:** captcha 12 处键（`captcha:rate/attempts/data:%s` + `login:fail:%s` + `captcha_background.go` 的 `captcha:bg:list` / `captcha:cache:pool` 族）**进 `pkg/constants/cache.go`**——跟随本模块既有先例 `CaptchaVerifiedKeyFormat`（captcha.go:385 已引用），同族键聚齐；`internal/core` 不反向 import `internal/services/system`。
- **D-102-3:** 形态分工——cache_keys.go 内用**前缀常量 + `GetXxxKey()` helper**（同 `GetDictDataByTypeKey` 模式，CLAUDE.md 明文约定），调用处 `fmt.Sprintf` 改 helper 调用；pkg/constants 的 captcha 键用 **Sprintf 格式常量**形态（同 `CaptchaVerifiedKeyFormat`）。
- **D-102-4:** 失效 pattern **前缀派生**——`CacheKeyManager.BuildPattern` 或前缀常量 + `":*"`，不新增独立 pattern 常量（键与 pattern 单一来源防漂移）。典型位点：`internal/services/network/cache_impl.go:319` 的 `"network_device:*"`。
- **D-102-5:** 缓存键**双档守护**：① 键值等价测试（常量值 == 原字面量快照，含 TTL 不变断言）；② invariants 风格**硬失败扫描**——覆盖包内内联 `fmt.Sprintf` cache key 字面量清零（白名单豁免机制）。
- **D-102-6:** STATUS 守护 = **AST 使用点扫描**，`internal/models/status_constants_test.go` 同文件扩展：业务代码中 status 数字字面量赋值/比较硬失败，geocoding 百度 API 入白名单表。不用字符串 grep（误报高）、不只验证时人工跑。
- **D-102-7:** 缓存键扫描**窄扫本相收敛面**（captcha 2 文件 + CACHE-02 的 10 模块清单）；Phase 103 由 CONV-04 自行扩口（mac_history/heatmap/rpa selector 位点闭包未迁前不入扫描面，避免临时豁免表）。
- STATUS 扫描口径取 SC-2 自然口径 = **全后端**（audit 时点全后端仅剩 12 处 + geocoding 白名单；执行中新暴露位点按同规则顺带清）。
- 分页侧**无新增守护**——`utils/pagination.go` 整文件删除即终结；CLAUDE.md 已声明 `NormalizePagination` 唯一入口，重引入属 code review 范畴。
- **D-102-8:** 12 处**优先引用既有常量**（绝大多数已存在）。确需新增常量的 model **只加用到的值**（机械相最小 diff）；新增走 CLAUDE.md 既有流程：先 `internal/models/<file>.go` 命名常量 → 同步 `status_constants_test.go` 期望表 → 业务代码按常量引用。
- **D-102-9:** 唯一生产调用方 `internal/api/v1/system/file_handler.go:160` 迁移 `query.NormalizePaginationWithMax(c, s, constants.MaxListPageSize)`——**cap 严格保持 100 行为等价**（不用默认 cap=200）；原 cap=100 vs pkg 默认 200 的历史分叉在调用处注释自证（SC-3 要求逐调用方核对清单落盘）。
- **继承锁定（Phase 99 D-03-3，用户长程标准）**：迁移后 `internal/utils/pagination.go` **整文件删除**（ParsePagination / PaginationParams / Offset / Limit / BuildPaginationResponse——后者无其他调用方），`internal/utils/utils_74_12_test.go` 的 `TestParsePaginationAndOffset` 一并删除，不留兼容壳。

### 继承的锁定决策（不再讨论）

- **v1.31 D-01**: 台账 12 组全做，不留兼容壳；**D-02**: 行为变更附回归测试、七 gate（coverage ≥78.33 基线）不倒退；**D-04**: captcha-background 1=启用语义禁改（QUIRK-80-03-D 锁定非 bug）
- **ROADMAP Phase 102 Notes**: rpa selector（:361）/ mac vendor（:255）键**本相只注册、Phase 103 才迁闭包**（CONV-01/03 同文件族，顺序不可倒）
- **Phase 99 D-03-10**: `pkg/constants` 是全项目唯一常量包
- **CLAUDE.md**: §Status Value Convention / §Pagination Constants Convention / §Cache Service Convention / §Timeout 常量约定

### Claude's Discretion

- captcha_background 池键族（`captcha:cache:pool:%s:%d` + `:counter` / `:%d` 派生键）在 pkg/constants 的常量切分方式（整格式 vs 前缀+后缀拼接）
- 等价测试的文件组织与命名（快照表驱动 vs 逐断言）、两个扫描守护的测试文件命名与放置包
- 12 处 status 位点 → 既有常量的逐处映射（按实际常量存在性逐处核对——本研究的核实结论见 STATUS 清单表）
- CAPTCHA-01 审计清单与 CACHE-02 模块清单的逐处位点核对（以 REQUIREMENTS 行号清单为起点，实际以 grep 结果为准——本研究已完成实跑核对）

### Deferred Ideas (OUT OF SCOPE)

- **captcha 内联 TTL 字面量**（如 failKey `1*time.Hour`、pool 项 `24*time.Hour`）顺带常量化——属 Timeout 常量约定范畴但非审计台账项，不在本相（本相只做键；TTL 值保持逐处等价并在等价测试中锁定）
- **`utils.BuildListResponse` 的 `page/pageSize` 响应键名**与前端 PageData `current` 约定的历史分叉——契约问题非常量化范畴，如需治理另立 phase
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| CACHE-01 | captcha.go（storageKey/failKey）与 captcha_background.go（list/pool key）12 处内联缓存键收敛具名常量，行为等价 | 实跑核实为 **16 个直接 Sprintf 位点 + 4 个派生构造点**、6 种键格式（行号清单见 §CACHE-01 清单）；注册目标 pkg/constants/cache.go（D-102-2）；先例 idiom 已在 captcha.go:385/450/491/512 就位 |
| CACHE-02 | 10 模块 ~35 处内联 cache key 及失效 pattern 注册具名常量并引用 | 实跑核实为 **47 个调用点 / 33 种键格式（含 pattern）**（行号清单见 §CACHE-02 清单）；发现 **services 根包 2 文件因 import 环无法引用 cache_keys.go**，须按 D-102-2 先例落 pkg/constants（详见 §关键约束）；rpa selector / mac vendor 键本相只注册不迁闭包 |
| STATUS-01 | 剩余 12 处 status 字面量全部引用 models 具名常量；AST 锁全程绿；白名单外 grep 无残留 | 实跑核实为 **21 个位点全部映射既有常量、零新增**（清单见 §STATUS 清单）；新暴露位点 1 处（base.go:191）按 D-102-6 口径顺带清；migrations archive 须入扫描排除表 |
| PAGI-01 | internal/utils ParsePagination 收敛 pkg/query 单一口径，调用方行为不变，逐调用方核对清单落盘 | 语义逐行等价已证（见 §PAGI 证据）；生产调用方**仅 file_handler.go:160 一处**（全仓 grep 实证）；file_handler_test.go + file_service_test.go 既有测试网在位 |
</phase_requirements>

## Summary

本相是纯后端**行为等价**重构，全部工作量在既有代码内部，不安装任何新包、不引入任何新外部依赖。研究以审计台账（F-06~F-09）行号为起点做了全量 grep 实跑核对，结论：**台账行号存在小幅漂移（±1~7 行）且系统性低估位点数**——captcha 实为 16 个直接位点（台账 12）、CACHE-02 实为 47 调用点/33 格式（台账 ~35）、STATUS 实为 21 位点（台账 12~16）。所有位点均已定位并逐处给出映射目标，执行时以本研究清单为准，不再依赖台账行号。

三项关键发现影响规划：

1. **import 环约束（最重要）**：`internal/services/system` 包（cache_keys.go 宿主）反向 import 了 `internal/services` 根包（adapter.go 等 5+ 文件）。因此 CACHE-02 清单中驻留根包的 2 个文件（`api_endpoint_service.go`、`mac_history_query_service.go`）**物理上不可能**引用 cache_keys.go 常量。经 `go list -deps` 实证：根包传递依赖不含任何 services 子包 → 唯一可行落点是 `pkg/constants/cache.go`（恰为 D-102-2 已锁定的 captcha 落点，先例一致）。这是对 D-102-1 字面的唯一必要修订，精神（集中注册、单一真相源）不变。
2. **STATUS 全部零新增**：21 个位点（含台账漏列的 base.go:191 同型位点）全部可引用既有常量（JobStatusNormal/Pause、JobLogStatusSuccess/Failure、DutyStatusNormal、VDIServerStatusNormal、WorkOrderStatusPending/Processing）。cron.go:832 是 raw SQL 内嵌字面量，须转占位符参数形态而非字符串拼接。
3. **PAGI 语义逐行等价已证**：`utils.ParsePagination` 与 `query.NormalizePaginationWithMax(page, size, constants.MaxListPageSize)` 归一化逻辑完全同构；生产调用方全仓仅 1 处（file_handler.go:160）。

**Primary recommendation:** 按「先注册常量+等价快照测试 → 逐文件机械替换 → 两个扫描守护 → 删 utils/pagination.go」四波推进；CACHE-02 落点按 import 方向二分（8 模块 → cache_keys.go，根包 2 文件 + captcha → pkg/constants/cache.go）；STATUS 直接按清单表机械替换；每波后 `go build ./...` + 包级测试。

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| captcha 键族具名化 | API/Backend（internal/core） | pkg/constants（常量宿主） | internal/core 不反向 import services 层（D-102-2 锁定方向），常量下沉 leaf 包 |
| 业务模块缓存键注册 | API/Backend（internal/services/*） | internal/services/system（cache_keys.go 宿主） | cache_keys.go 是既有声明的键唯一真相源（D-102-1） |
| 根包 2 文件键注册 | API/Backend（internal/services 根） | pkg/constants（常量宿主） | system→root import 环（本研究实证），root 只能引用 leaf 包 |
| status 字面量替换 | API/Backend（internal/scheduler + services） | internal/models（常量真相源） | status 常量唯一真相源在 models（CLAUDE.md §Status Value Convention） |
| 分页口径归一 | API/Backend（pkg/query 唯一权威） | — | NormalizePaginationWithMax 是唯一实现核心（Phase 99 D-03-2） |
| 两个扫描守护 | API/Backend（go test AST 扫描） | — | 复用 cache_invariants_92_test.go / status_constants_test.go 既有宿主模式 |

## Standard Stack

### Core

**本相零新包。** 全部使用仓库既有设施：

| 设施 | 版本/位置 | Purpose | Why Standard |
|------|-----------|---------|--------------|
| Go | 1.24.5（实测 `go version`） | 编译/测试 | 与 CLAUDE.md Go 1.24 一致 [VERIFIED: 本机 go version] |
| `internal/services/system/cache_keys.go` | 375 行既有 | 8 模块键注册宿主（前缀常量 + GetXxxKey helper 形态） | D-102-1 锁定；duty/knowledge/network/workorder 的 cache_impl **已 import 该包**（实读 import 块确认），引用零新增成本 |
| `pkg/constants/cache.go` | 既有（3 个键格式常量） | captcha 键族 + 根包 2 文件键注册宿主 | D-102-2 锁定；leaf 包可被所有层引用 |
| `pkg/query.NormalizePaginationWithMax` | 既有（Phase 99 交付） | 分页归一唯一实现核心 | D-102-9 直接可用，无需新增 pkg 函数 |
| `internal/models` status 常量族 | 既有（log.go:71/94、duty.go:25、vdi.go:51、workorder.go:17-21） | 21 处 STATUS 位点映射目标 | 全部存在，经 grep 核实（见 STATUS 清单表） |
| `go/ast` + `go/parser` | 标准库 | 两个扫描守护的实现 | cache_invariants_92_test.go / status_constants_test.go 同款先例 |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| 根包 2 文件键入 cache_keys.go | 根包键入 pkg/constants/cache.go | 前者 import 环**不可行**（本研究实证）；后者与 captcha 先例同构，为唯一可行解 |
| `CacheKeyManager.BuildPattern` 派生失效 pattern | 前缀常量 + `":*"` 拼接 | BuildPattern 以 `*` 直接追加（无 `:`），`BuildPattern("my_notices", uid)` 产出 `notice:my_notices:{uid}*` ≠ 现值 `notice:my_notices:{uid}:*`——**用 BuildPattern 需补空 part，易错**；推荐前缀派生（D-102-4 两可，择安全者） |
| AST 扫描用正则预筛 | 纯 AST 精确匹配 | 正则误报高（D-102-6 明文否决字符串 grep）；可正则粗筛 + AST 精判，但纯 AST（invariants_92 同款）已够 |

**Installation:**
```bash
# 无安装步骤——零新依赖
go build ./...   # 基线已实测全绿（BUILD_EXIT=0，2026-09-07）
```

## Package Legitimacy Audit

**不适用——本相不安装任何外部包。** 全部工作在仓库既有代码与 Go 标准库内完成。

## 关键约束：import 环与键注册落点（本研究核心发现）

### 实证链条

1. `internal/services/system/adapter.go:7`（及 config_cache_impl.go、dashboard_service.go、department_cache_impl.go、dict_cache_impl.go 等）**import 了 `internal/services` 根包** [VERIFIED: grep]
2. 因此根包（package services）内文件 **import system = 编译环** [VERIFIED: Go import 规则 + 上条实证]
3. `go list -deps ./internal/services/system` 不含 rpa/duty/knowledge/network/workorder → **这些子包 import system 安全** [VERIFIED: go list]
4. `go list -deps ./internal/services` 不含任何 services 子包 → 根包只能引用 leaf 包（pkg/constants 已被 mac_history_query_service.go import，先例在位）[VERIFIED: go list + import 块实读]

### 落点二分结论

| 落点 | 文件 | 键格式数 |
|------|------|----------|
| `internal/services/system/cache_keys.go`（D-102-1 主落点） | notice/settings/widget（同包零 import）+ duty/workorder/knowledge/network（已 import system）+ rpa/selector_learner.go（rpa→system 安全） | 31 |
| `pkg/constants/cache.go`（D-102-2 落点，根包豁免同机理） | internal/core/captcha.go + captcha_background.go（core→root 已存在，core→system 禁止）+ api_endpoint_service.go + mac_history_query_service.go（根包→system 禁止） | 6 + 2 |

> **规划提示**：这是对 D-102-1 字面（"全集中注册进 cache_keys.go"）的唯一必要修订——2 个根包文件改落 pkg/constants。建议 plan 中显式标注此偏离及实证依据，供 discuss/用户知悉；其余 8 模块严格走 cache_keys.go。

## 位点全量清单（实跑核实，替代台账行号）

### CACHE-01：captcha 键族（internal/core，6 格式 / 16 直接位点 + 4 派生点）

**captcha.go（package core）：**

| 行号 | 局部变量 | 键格式 | TTL（保持不变） |
|------|----------|--------|------------------|
| :249 | rateLimitKey | `captcha:rate:%s`（clientIP） | 1min（IncrementWithExpire / Expire 内联，**TTL 常量化出 scope**） |
| :297 | storageKey | `captcha:data:%s`（captchaID） | config.ExpireTime 分钟（动态） |
| :304 | attemptsKey | `captcha:attempts:%s` | 同上 |
| :326 | storageKey | `captcha:data:%s` | 同上 |
| :333 | attemptsKey | `captcha:attempts:%s` | 同上 |
| :356 | attemptsKey | `captcha:attempts:%s` | —（读） |
| :361 | storageKey | `captcha:data:%s` | —（Delete） |
| :367 | storageKey | `captcha:data:%s` | —（读） |
| :415 | attemptsKey | `captcha:attempts:%s` | —（读） |
| :419 | storageKey | `captcha:data:%s` | —（Delete） |
| :426 | storageKey | `captcha:data:%s` | —（GetJSON） |
| :503 | failKey | `login:fail:%s`（username） | 1h（:507 Expire） |
| :529 | failKey | `login:fail:%s` | —（Delete） |

**captcha_background.go（package core）：**

| 行号 | 局部变量 | 键格式 | TTL |
|------|----------|--------|-----|
| :146 | cacheKey | `captcha:bg:list:%s:%d`（shape, difficulty） | 5min（:187） |
| :243 | poolPrefix | `captcha:cache:pool:%s:%d`（string(shape), difficulty） | — |
| :310 | poolPrefix | `captcha:cache:pool:%s:%d` | — |

**派生构造点（无 `captcha:` 字面量，grep 不直接命中，随 poolPrefix 常量化自然收敛）：**

| 行号 | 构造 | 说明 |
|------|------|------|
| :245 / :311 | `counterKey = poolPrefix + ":counter"` | 计数器键，TTL 24h（:303/:340） |
| :295 / :326 | `itemKey = fmt.Sprintf("%s:%d", poolPrefix, N)` | 池项键（N = 环回索引），TTL 24h（:296） |

> 台账/REQUIREMENTS 的 captcha_background "":295" 即 itemKey 派生点——解释了行号清点差异。池键族切分方式是 D-102 discretion（建议：poolPrefix 格式常量 + `":counter"` / `":%d"` 派生，与现构造一一对应，等价性最易证）。

**同文件既有先例（目标 idiom，勿动）**：`constants.CaptchaVerifiedKeyFormat`（:385/:450）、`constants.LoginLockKeyFormat`（:491/:512）。

**测试文件注意**：captcha_78_01_test.go / captcha_background_78_01_test.go / core_74_08_test.go 内嵌键字面量（如 "captcha:cache:pool:circle:1:1"）——键值不变则测试自然绿，**最小 diff 纪律下不改测试字面量**。

### CACHE-02：10 模块内联键（33 格式 / 47 调用点）

| # | 模块 | 文件 | 位点（实跑行号） | 键格式（含 pattern） |
|---|------|------|------------------|----------------------|
| 1 | notice | internal/services/system/notice_cache_impl.go | :208, :210, :240, :289, :296(×2), :302 | `notice:my_notices:%s:page:%d:size:%d`、同+`:status:%s`、`notice:unread_count:%s`、`notice:detail:%s`、pattern `notice:my_notices:%s:*`、pattern `notice:*` |
| 2 | settings | internal/services/system/settings_cache_impl.go | :41, :53, :64 | `settings:user:%s` |
| 3 | duty | internal/services/duty/duty_cache_impl.go | :137, :144, :215, :273, :280, :287, :294, :301 | `duty:today`、`duty:monthly:%d:%d`、`duty:holidays:%d`、pattern `duty:*`、pattern `duty:holidays:*` |
| 4 | workorder | internal/services/workorder/workorder_cache_impl.go | :215, :241, :250, :257, :264, :271 | `workorder:my_pending:%s:limit:%d`、`workorder:statistics`、`workorder:detail:%s`、`workorder:my_pending:%s`、pattern `workorder:*` |
| 5 | knowledge | internal/services/knowledge/knowledge_cache_impl.go | :91, :129, :131, :134, :180, :223, :230, :237, :244 | `kb:article:%s`、`kb:category:tree`、`kb:category:parent:%s`、`"%s:status:%d"`（:134 条件后缀，复用已计数 base 格式）、`kb:tags:all`、pattern `kb:category:*`、pattern `kb:article:*` |
| 6 | network | internal/services/network/cache_impl.go | :268, :275, :282, :291, :298, :305, :312, :319 | `network_device:statistics`、`network_device:dept:%s`、`network_device:credential:%s`、`network_device:detail:%s`、pattern `network_device:*` |
| 7 | api_endpoint | internal/services/api_endpoint_service.go（**根包→pkg/constants**） | :63, :194 | `user_endpoints:%s` |
| 8 | mac vendor | internal/services/mac_history_query_service.go（**根包→pkg/constants**；Phase 103 只注册不迁闭包） | :256 | `mac:vendor:%s` |
| 9 | widget | internal/services/system/widget_data_fetcher.go | :61, :65（buildWidgetCacheKey 内） | `widget:data:%s`、`widget:data:%s:%x`（**%x 动词须原样保留**） |
| 10 | rpa selector | internal/services/rpa/selector_learner.go（Phase 103 只注册不迁闭包） | :362 | `rpa:selector:best:%s:%s` |

> 与台账行号差异：mac vendor :255→实跑 :256；rpa :361→实跑 :362（各漂移 1 行）；duty/workorder/knowledge/network 的台账清单均为子集（实跑多出 plain-literal 键与 pattern 位点）。**knowledge :134 位点**：`GetKnowledgeCategoryList` 的条件键后缀拼接（`cacheKey = fmt.Sprintf("%s:status:%d", cacheKey, *req.Status)`），产出真实缓存键并在 :137 传入 GetOrSetJSON——计入调用点数（47），其格式复用已计数 base 格式故格式数仍为 33（对照 notice 模块 status 变体 :210 为完整独立格式已单列）。TTL 解析全部经 `GetExpiration(configKey, default)` / `getExpiration(...)`（逐文件实读核实），**键替换不触及 TTL 表达式**；`user_endpoints`/`mac:vendor` 无失效调用（纯读缓存）。

**范围外同病位点（扫描面之外，规划时知情即可，默认不动）**：
- `operations/floor_cache_impl.go:50,:71` `floor:building:%s`——**已有注册常量** `CacheKeyFloorByBuilding`（cache_keys.go:140）却仍内联，属"已注册未引用"残留，非本相清单（D-102-7 窄扫口径）；若 planner 愿纳为顺带清零需同步扩 D-102-7 扫描面。
- `operations/excel_config.go` CachePatterns（`dept:*`/`user:*`/...）——Excel 管线 entityType 派发（CLAUDE.md 明文保留架构），**不动**。
- `internal/pkg/cache/manager.go:75` `sys:%s:%s`、`addomain/sync.go:63` `sync:%s:%s`、`api/v1/rpa/worker_handler.go:299` `worker:scale:%s`（channel 名非缓存键）——均不在 10 模块清单。

### STATUS：21 位点 → 既有常量映射表（零新增常量）

| # | 位点 | 现状 | 目标常量（均已存在，grep 核实） | 转换注意 |
|---|------|------|-------------------------------|----------|
| 1 | scheduler/cron.go:43 | `Status: 0, // 成功`（JobLog 复合字面量） | `models.JobLogStatusSuccess` | 直接替换 |
| 2 | scheduler/cron.go:62 | `jobLog.Status = 1 // 失败` | `models.JobLogStatusFailure` | 直接替换 |
| 3 | scheduler/cron.go:235 | `Where("status = ?", 0)`（Job） | `models.JobStatusNormal` | 占位符参数，直接传常量 |
| 4 | scheduler/cron.go:407 | `Update("status", 0)`（Job） | `models.JobStatusNormal` | 同上 |
| 5 | scheduler/cron.go:435 | `Update("status", 1)`（Job） | `models.JobStatusPause` | 同上（**不是 Stopped——log.go:72 命名即 Pause**） |
| 6 | scheduler/cron.go:832 | `Where("ds.schedule_date = ? AND ds.status = 0", date)`（DutySchedule） | `models.DutyStatusNormal` | **raw SQL 内嵌字面量**：转 `ds.status = ?` 占位符 + 常量作参，禁字符串拼接 |
| 7 | scheduler/vdi_sync_tasks.go:48 | `Where("status = ?", 0)`（VDIServer） | `models.VDIServerStatusNormal` | 占位符参数 |
| 8 | scheduler/vdi_sync_tasks.go:85 | `server.Status != 0` | `models.VDIServerStatusNormal` | BinaryExpr 比较替换 |
| 9 | scheduler/workorder_tasks.go:195 | `Where("schedule_date = ? AND status = ?", today, 0)` | `models.DutyStatusNormal` | 已有注释自证 `// DutyStatusNormal = 0` |
| 10 | scheduler/workorder_tasks.go:328 | `Status: 0`（models.Job 复合字面量） | `models.JobStatusNormal` | 直接替换 |
| 11 | scheduler/workorder_tasks.go:451 | `Status: 0, // 启用` | `models.JobStatusNormal` | 同上 |
| 12 | scheduler/reconciliation_tasks.go:196 | `Status: 0, // 0=启用` | `models.JobStatusNormal` | 同上 |
| 13 | scheduler/mac_history_tasks.go:127 | `Status: 0` | `models.JobStatusNormal` | 同上 |
| 14 | scheduler/mac_history_matview_tasks.go:56 | `Status: 0` | `models.JobStatusNormal` | 同上 |
| 15 | services/scheduler/job_service.go:331（实跑 :332） | `if status == 0 { // 启用` | `models.JobStatusNormal` | status 为函数入参 int → `status == int(models.JobStatusNormal)` 或改签名类型（择最小 diff） |
| 16 | services/workorder/base.go:183 | `Where("status IN ?", []int{0, 1})` | `[]int{int(models.WorkOrderStatusPending), int(models.WorkOrderStatusProcessing)}` | 或 `[]models.WorkOrderStatus{...}`（GORM 均可序列化，择一统一） |
| 17 | services/workorder/base.go:191 | 同上（**台账漏列，D-102-6 全后端口径顺带清**） | 同 #16 | 同 #16 |
| — | services/operations/geocoding_service.go:333 | `if baiduResp.Status != 0` | **白名单**（百度 API 返回码，非 DB status） | D-102-6 白名单表首条：`geocoding_service.go + 原因` |

**既有常量坐标** [VERIFIED: grep internal/models]：`log.go:71-72` JobStatusNormal=0/JobStatusPause=1；`log.go:94-95` JobLogStatusSuccess=0/JobLogStatusFailure=1；`duty.go:25` DutyStatusNormal=0；`vdi.go:51-52` VDIServerStatusNormal=0/Stopped=1；`workorder.go:17-21` WorkOrderStatus 0..4 全族。（WorkOrderStatus 族的值锁登记状态见 PATTERNS §SP-6：未入 watched 表，由 102-04 Task 3 补登记。）

**扫描器必须排除/豁免的面**（广谱扫描实测发现）：
- `internal/core/db/migrations/archive/applied/*.go`——**大量 `Status: 0` / `Where("status = 0")` 位点**，是已归档迁移历史，禁改禁扫（改之破坏迁移完整性）。
- 字符串型 status（`asset/fix_suggestion_service.go` 的 `fix_status = "pending"` 族）——AST 扫描限定**数字字面量**自然排除。
- 非 status 的数值复合字面量（`captcha_background.go:213` `difficulties := []int{1,2,3}`、`snmp_entity_collector.go:20` SNMP 类别码）——按**标识符名匹配**（Status 字段键 / `.Status` 选择器 / `"status"` SQL 列名）自然排除。
- `_test.go` 全部排除。

**D-102-6 扫描器需覆盖的 6 种 AST 形态**（全部实测位点归纳）：
1. 复合字面量字段键：`Status: 0`（KeyValueExpr，Key==Ident("Status")）
2. 赋值：`x.Status = 1`（AssignStmt，LHS SelectorExpr.Sel=="Status"）
3. 二元比较：`x.Status != 0` / `status == 0`（BinaryExpr，一侧为 .Status 选择器或 status 标识符，另一侧为 int 字面量）
4. GORM 调用：`Where(<含"status"的 SQL 串>, <int 字面量>)`（CallExpr，Fun 为 Where/Update/Raw 等）
5. `Update("status", <int 字面量>)`
6. `status IN ?` 形态的 `[]int{...}` 字面量切片实参

### PAGI：证据与迁移形态

**语义等价证明**（逐行对照实读）：
- `utils.ParsePagination`（pagination.go:15-29）：`page<=0→DefaultCurrent(1)`；`pageSize<=0→DefaultPageSize(10)`；`pageSize>MaxListPageSize(100)→100`
- `query.NormalizePaginationWithMax`（pkg/query/pagination.go:51-62）：`current<=0→DefaultCurrent`；`pageSize<=0→DefaultPageSize`；`pageSize>max→max`（钳制不重置、无下限放大）
- 两者**完全同构**；cap 差异即唯一历史分叉点（utils=100，pkg 默认=200）→ D-102-9 用 `NormalizePaginationWithMax(page, size, constants.MaxListPageSize)` 精确保留 100。

**调用方全量核对清单**（全仓 `**/*.go` grep 实证，SC-3 落盘基底）：

| 调用方 | 类型 | 处置 |
|--------|------|------|
| internal/api/v1/system/file_handler.go:160 | **唯一生产调用方** | 迁移 `query.NormalizePaginationWithMax(req.Page, req.PageSize, constants.MaxListPageSize)`；下游 `pagination.Offset()`(:163)/`pagination.Limit()`/`pagination.Page`/`pagination.PageSize` 改局部变量+offset 算式 `(current-1)*pageSize`；cap=100 分叉注释自证 |
| internal/utils/utils_74_12_test.go:203-226（TestParsePaginationAndOffset） | 测试 | 随整文件删除（D-102-9 继承锁定） |
| internal/api/v1/operations/requests.PaginationParams 系列（requests_74_12_test.go 等 6 文件） | **同名异型**（operations requests 包自有类型） | **不相关，勿动**——删除 utils 版不影响 |

**删除清单**：`internal/utils/pagination.go` 整文件（ParsePagination/PaginationParams/Offset/Limit/BuildPaginationResponse——BuildPaginationResponse 零其他调用方，全仓 grep 实证）；`internal/utils/utils_74_12_test.go` 中对应测试。**保留**：`internal/utils/response_builder.go`（BuildListResponse，file_handler 仍在用，CONTEXT 明确保留）。

## Architecture Patterns

### System Architecture Diagram

```
[Phase 102 数据流：常量注册 → 引用替换 → 守护锁定]
------------------------------------------------------------------
 原字面量（16+47+21 位点）                    新真相源（注册）
 ┌──────────────────────────┐   D-102-2    ┌─────────────────────────┐
 │ internal/core/captcha*.go ├─────────────►│ pkg/constants/cache.go   │
 └──────────────────────────┘              │  (+6 captcha 格式)       │
 ┌──────────────────────────┐   import 环豁免│  (+2 根包格式)           │
 │ api_endpoint_service.go  ├─────────────►└─────────────────────────┘
 │ mac_history_query_...go  │（root→system 禁止）
 └──────────────────────────┘              ┌─────────────────────────┐
 ┌──────────────────────────┐   D-102-1    │ system/cache_keys.go     │
 │ 8 模块 cache_impl/fetcher ├─────────────►│  (+31 格式)              │
 └──────────────────────────┘              └─────────────────────────┘
 ┌──────────────────────────┐   D-102-8    ┌─────────────────────────┐
 │ scheduler/* + workorder   ├─────────────►│ internal/models 常量族   │
 │ base.go + job_service.go  │   （零新增）  │ （已存在，AST 值锁在位）  │
 └──────────────────────────┘              └─────────────────────────┘
 ┌──────────────────────────┐   D-102-9    ┌─────────────────────────┐
 │ file_handler.go:160       ├─────────────►│ query.Normalize...WithMax│
 └──────────────────────────┘              └──────────┬──────────────┘
                                                      ▼
                                            删 internal/utils/pagination.go
------------------------------------------------------------------
 守护（Wave 0 交付，进 go test ./... → 七 gate）：
 [等价快照测试] 常量值 == 原字面量（D-102-5①）
 [cache-key 内联扫描] 12 文件硬失败清零（D-102-5②/7）
 [status AST 使用点扫描] 全后端硬失败 + geocoding 白名单（D-102-6）
```

### Pattern 1：前缀常量 + GetXxxKey helper（cache_keys.go 形态，D-102-3）

**What:** 键格式注册为前缀常量，参数化键经 helper 拼接；调用点 Sprintf 改 helper。
**When to use:** CACHE-02 全部 8 个 cache_keys.go 落点模块。
**Example:**
```go
// Source: internal/services/system/cache_keys.go:299-301（GetDictDataByTypeKey 既有形态）
// cache_keys.go 新增：
const (
    CacheKeyNoticeDetail     = "notice:detail"      // 通知详情: notice:detail:{noticeID}
    CacheKeyNoticeMyNotices  = "notice:my_notices"  // 我的通知: notice:my_notices:{userID}:page:{page}:size:{size}
    CacheKeyNoticeUnread     = "notice:unread_count"
)
func GetNoticeMyNoticesKey(userID string, page, pageSize int) string {
    return fmt.Sprintf("%s:%s:page:%d:size:%d", CacheKeyNoticeMyNotices, userID, page, pageSize)
}
// 失效 pattern 前缀派生（D-102-4，不新增独立 pattern 常量）：
func GetNoticeMyNoticesPattern(userID string) string {
    return fmt.Sprintf("%s:%s:*", CacheKeyNoticeMyNotices, userID)
}
// 调用点（notice_cache_impl.go:208）：
cacheKey := systemServices.GetNoticeMyNoticesKey(userID, page, pageSize)
```

### Pattern 2：Sprintf 格式常量（pkg/constants 形态，D-102-3）

**What:** 整格式串作常量，调用点保留 `fmt.Sprintf(常量, args...)`。
**When to use:** captcha 6 格式 + 根包 2 格式。
**Example:**
```go
// Source: pkg/constants/cache.go:17（CaptchaVerifiedKeyFormat 既有先例）
// cache.go 新增：
const (
    CaptchaRateLimitKeyFormat  = "captcha:rate:%s"
    CaptchaDataKeyFormat       = "captcha:data:%s"
    CaptchaAttemptsKeyFormat   = "captcha:attempts:%s"
    LoginFailKeyFormat         = "login:fail:%s"
    CaptchaBgListKeyFormat     = "captcha:bg:list:%s:%d"
    CaptchaCachePoolPrefixFormat = "captcha:cache:pool:%s:%d"
)
// 调用点（captcha.go:297）：
storageKey := fmt.Sprintf(constants.CaptchaDataKeyFormat, captchaID)
// 池键族派生（discretion 推荐切分：前缀格式 + 后缀拼接，与现构造一一对应）：
poolPrefix := fmt.Sprintf(constants.CaptchaCachePoolPrefixFormat, shape, difficulty)
counterKey := poolPrefix + ":counter"                                   // captcha_background.go:245 现状
itemKey    := fmt.Sprintf("%s:%d", poolPrefix, (counter%int64(poolSize))+1) // :295 现状，不变
```

### Pattern 3：AST 扫描守护（invariants_92 同款骨架）

**What:** runtime.Caller 相对定位 + parser.ParseFile + ast.Inspect + 白名单 map 硬失败。
**When to use:** D-102-5② cache-key 扫描（12 文件窄扫）与 D-102-6 status 使用点扫描（全后端）。
**Example:**
```go
// Source: internal/services/system/cache_invariants_92_test.go:33-81（allowedResidues + cacheImplResidue92 骨架）
var allowedKeyResidues = map[string]int{} // 文件名 → 豁免额度，初始全空
// status 白名单（D-102-6）：文件 → 豁免原因
var statusLiteralWhitelist = map[string]string{
    "geocoding_service.go": "百度地图 API 返回码（非 DB status，audit 观察项定性）",
}
// 迁移纪律（invariants_92 :121-130 先例）：文件定位用 runtime.Caller 相对推导，
// 禁本地绝对路径断言（v1.27 测试纪律）
```

### Pattern 4：raw SQL 字面量 → 占位符参数（cron.go:832 专属）

```go
// Source: internal/scheduler/cron.go:832 现状
// BEFORE: .Where("ds.schedule_date = ? AND ds.status = 0", date)
// AFTER:
.Where("ds.schedule_date = ? AND ds.status = ?", date, models.DutyStatusNormal)
// 禁止：fmt.Sprintf("... ds.status = %d", models.DutyStatusNormal)——可运行但引入注入面模式先例，劣于占位符
```

### Anti-Patterns to Avoid

- **改键值"顺手优化"**：键格式是运行时 Redis 契约，任何前缀重排/大小写归一都会孤儿化存量键并使失效 pattern 脱靶——常量值必须逐字符等于原字面量。
- **用 BuildPattern 派生 pattern 不验证输出**：其 `*` 直接追加无 `:`（cache_keys.go:34-41），产出与现值差一个分隔符。
- **动 migrations/archive**：哪怕"只是常量化"。
- **扩缓存键扫描面到清单外**：D-102-7 明文窄扫 12 文件；floor_cache_impl 等同病位点留给 Phase 103 CONV-04 扩口。
- **替换后重跑全部测试前提交**：`status_constants_test.go` 与 operlog regression_test.go 是常量邻接面，必须全程绿。

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| 分页归一 | 新写 clamp 逻辑或第三处实现 | `query.NormalizePaginationWithMax` | 唯一实现核心（Phase 99 D-03-2）；再写第三份即重蹈 F-09 |
| AST 扫描框架 | 从零设计扫描器 | 复制 invariants_92 / status_constants_test 既有骨架 | 两宿主已解决相对路径定位、白名单卫生、双档调和等细节 |
| 键冲突防护 | 手写转义 | 既有 `EscapeCacheKeyValue`（cache_keys.go:373） | Phase 98 已交付；键值不变则无需触碰 |
| 缓存读写/失效 | 绕过 base 抽象直写 | 位点现状已是 `base.GetOrSetJSON`/`Invalidate`/`InvalidatePattern` | 本相只换键实参，**不改缓存调用形态**（闭包迁移是 Phase 103） |

**Key insight:** 本相全部"难题"（AST 扫描、键等价锁定、分页归一）在仓库内都有一等先例可直接续用；自创方案只会引入第二口径——恰是本相要消灭的东西。

## Runtime State Inventory

> 重构相强制盘点。canonical question：文件全部更新后，哪些运行时系统仍持有旧串？

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | Redis 业务缓存键（`xingran:` 前缀 + 本研究 39 种键格式的存量实例）。**键值由等价测试逐字符锁定，预期零变化**；若执行中值漂移：存量条目孤儿化 + 失效 pattern 脱靶（脏读风险） | 无迁移——等价快照测试即防线（D-102-5①） |
| Live service config | sys_config 内 TTL 配置键（`cache.notice.my_notices` / `cache.duty.today` / `cache.kb.article` 等，经 GetExpiration 解析）——**本相不改 TTL 配置键**，只改键格式串 | 无 |
| OS-registered state | 无——不涉及可执行体/服务/计划任务注册（已核实：纯源码级重构） | None — verified（无改名/迁移面） |
| Secrets/env vars | 无——39 种键格式与 21 处 status 位点均不涉密钥键名 | None — verified |
| Build artifacts | 无——Go 就地编译无安装产物；`internal/utils/pagination.go` 删除为纯源码操作，无 egg-info/二进制残留面 | None — verified |

## Common Pitfalls

### Pitfall 1：services 根包 import 环（最易翻车点）
**What goes wrong:** 把 api_endpoint/mac vendor 键注册进 cache_keys.go 后根包引用 → `import cycle not allowed` 编译失败。
**Why it happens:** D-102-1 字面读法是"全集中"；system→root 反向依赖（adapter.go 等）在 CLAUDE.md/CONTEXT 均未记载。
**How to avoid:** 按 §关键约束 落点二分表执行；plan 中显式标注对 D-102-1 字面的修订。
**Warning signs:** `go build` 报 `import cycle`；或执行者"顺手"把 system/adapter.go 改为不依赖根包（**禁**——那是架构级重构，远超本相）。

### Pitfall 2：台账行号漂移导致漏改/误改
**What goes wrong:** 按 REQUIREMENTS 行号清单逐行替换，漂移位点（mac vendor +1、rpa +1、workorder/knowledge/network 子集）被漏或错改邻近行。
**Why it happens:** 台账是 2026-09-07 快照，代码在其后仍有提交。
**How to avoid:** 以本研究实跑清单为执行底稿；每文件替换前 `grep -n` 复核行号。
**Warning signs:** 替换计数与清单表对不上。

### Pitfall 3：JobStatusPause 误写为"Stopped/Disabled"
**What goes wrong:** cron.go:435 `Update("status", 1)` 映射到不存在的 JobStatusStopped → 编译失败或自创新常量（违反 D-102-8 零新增）。
**Why it happens:** "1=停用"的普适惯例直觉；models/log.go:72 实名 **JobStatusPause**。
**How to avoid:** 严格按 STATUS 清单表映射列执行。
**Warning signs:** models 包出现新增常量 diff。

### Pitfall 4：raw SQL 内嵌字面量硬替换
**What goes wrong:** 把 `ds.status = 0` 里的 0 直接当独立 token 替换/字符串拼接常量，产出行不通或风格劣化的代码。
**How to avoid:** Pattern 4 占位符参数形态；base.go:183/191 的 `[]int{0,1}` 同理——整切片表达式替换并统一 int 转换口径。
**Warning signs:** SQL 字符串内出现 `%d` 动词拼接。

### Pitfall 5：等价测试写成"常量 == 常量"同义反复
**What goes wrong:** 等价测试用常量自身作期望值，值漂移时测试仍绿，守护失效。
**How to avoid:** 期望列写**原字面量快照**（字符串 "notice:my_notices" / 格式串 "captcha:data:%s"），常量是被测方（CONTEXT Specifics 快照表驱动即此意）。
**Warning signs:** 测试文件里期望值列出现 `constants.Xxx` 引用。

### Pitfall 6：扫描器误伤 / 漏报失衡
**What goes wrong:** 全后端 status 扫描误伤 migrations archive、difficulties 数组、SNMP 类别码（实测均存在）；或 cache-key 扫描把 Phase 103 待迁的闭包位点（selector_learner:169-174/226 手写 cache-aside 内的键）提前硬失败，制造临时豁免表。
**How to avoid:** §STATUS 清单的排除/豁免表 + D-102-7 窄扫 12 文件（selector_learner 只扫 :362 键构造函数所在键字面量，**扫描器按"键格式形态字面量"匹配，天然不含 :169-174 的 Get/Set 调用**——键格式串仍会命中，属预期：本相注册后该文件键字面量同样清零，不产生豁免项）。扫描器匹配规则须防误伤：剥离格式动词后分段校验（见 102-03 Task 3 精确口径），排除错误消息串（含空格段）与纯动词串（`"%s:%d"` 全空段）。
**Warning signs:** 守护测试在干净主干上红；或白名单表出现"临时性"条目。

### Pitfall 7：TTL/configKey 顺改动
**What goes wrong:** 替换键时顺手把 `1*time.Hour`、`24*time.Hour`、`getExpiration("cache.duty.today", ...)` 等一并"整理"——TTL 常量化是 **Deferred**（CONTEXT 明文），configKey 属 sys_config 运行时契约。
**How to avoid:** diff 审查：本相只允许键实参行变化。
**Warning signs:** diff 中出现 time.Duration / getExpiration 行变更。

## Code Examples

### 复合字面量字段替换（STATUS 最常见形态）
```go
// Source: internal/scheduler/workorder_tasks.go:325-330 现状 → 目标
newJob := &models.Job{
    // ...
    MisfirePolicy: models.MisfirePolicyExecuteOnce,
    Status:        models.JobStatusNormal, // 原: Status: 0
    // ...
}
```

### GORM Update/Where 占位符替换
```go
// Source: internal/scheduler/cron.go:407/:435 现状 → 目标
s.db.Model(&models.Job{}).Where("id = ?", jobID).Update("status", models.JobStatusNormal)
s.db.Model(&models.Job{}).Where("id = ?", jobID).Update("status", models.JobStatusPause)
```

### 键格式常量 + helper 双形态落位
```go
// pkg/constants 侧（captcha / 根包）——见 Pattern 2
// cache_keys.go 侧（8 模块）——见 Pattern 1
// 调用点改造前后（duty_cache_impl.go:137 → 注册后）：
// BEFORE: return base.GetOrSetJSON(ctx, s.cache, "duty:today", s.getExpiration("cache.duty.today", 5*time.Minute), ...)
// AFTER:  return base.GetOrSetJSON(ctx, s.cache, systemServices.CacheKeyDutyToday, s.getExpiration("cache.duty.today", 5*time.Minute), ...)
// （plain-literal 键直接引常量；参数化键走 GetXxxKey helper——D-102-3）
```

### 等价快照测试（D-102-5①，快照表驱动）
```go
// 期望值 = 原字面量快照（来自本研究清单），被测方 = 新常量
cases := []struct{ name, want string; got string }{
    {"CaptchaDataKeyFormat", "captcha:data:%s", constants.CaptchaDataKeyFormat},
    {"LoginFailKeyFormat", "login:fail:%s", constants.LoginFailKeyFormat},
    {"CacheKeyNoticeMyNotices", "notice:my_notices", systemServices.CacheKeyNoticeMyNotices},
    // ... 39 格式全量
}
for _, tc := range cases {
    t.Run(tc.name, func(t *testing.T) {
        require.Equal(t, tc.want, tc.got) // 值漂移 = Redis 契约破坏，硬失败
    })
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| 分页字面量散布 + 双 normalize（F-01/F-09） | constants.DefaultCurrent 族 + query.NormalizePaginationWithMax 唯一核心 | Phase 89 / Phase 99（2026-08~09） | 本相 PAGI-01 是该收敛的最后一根尾巴 |
| AD/user status SQL 字面量 | models 常量引用（F-02 先例 commit de1baf0） | 2026-09-07 audit 轮 | 本相 STATUS-01 同法推广至 scheduler/workorder 域 |
| config 缓存键 5 处内联 | cache_keys.go 注册表（F-03 先例，新注册 CacheKeyConfigAll） | 2026-09-07 audit 轮 | 本相 CACHE-02 是同法的 10 模块全量推广 |
| AST 锁值（constants stability） | AST 使用点扫描（本相 D-102-6 新增能力） | 本相 | 从"值不许漂"升级到"使用点必须引常量" |

**Deprecated/outdated:**
- `utils.ParsePagination / PaginationParams / Offset / Limit / BuildPaginationResponse`：本相终结（整文件删除，不留壳——继承锁定）。
- 内联 `fmt.Sprintf("<key-format>", ...)` 键构造：本相后在 12 个收敛文件内成为守护红线。

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | services 根包 2 文件键改落 pkg/constants 是对 D-102-1 字面的**必要**修订（非可选优化）——import 环为 Go 语言级硬约束，已 go list 实证 | 关键约束 | 低：实证充分；若用户坚持字面，唯一替代是架构级解环（system 不再 import root），远超本相范围，需升级决策 |
| A2 | `[]models.WorkOrderStatus{...}` 直接作 GORM `IN ?` 实参与 `[]int{int(...), int(...)}` 行为等价（driver 对 int-kind slice 序列化一致） | STATUS 清单 #16 | 低：GORM 对整型切片统一展开为占位符；选 `[]int{int(...)}` 形态则零风险 |
| A3 | job_service.go:331 的 `status` 入参为 `int`（非 models.JobStatus 类型），需 `int(...)` 转换 | STATUS 清单 #15 | 极低：编译器即时反馈，无行为风险 |
| A4 | gsd-sdk CLI 在本环境不可用（`command not found`），文档提交走原生 git | 环境可用性 | 无：仅影响研究文档提交流程 |

> 除 A1 外无影响执行路径的假设；A1 建议规划时以 checkpoint 形式向用户显式披露。

## Open Questions

1. **reconciliation_tasks.go:195 的 `MisfirePolicy: 1`（紧邻 #12 位点）（RESOLVED）**
   - What we know: 同一复合字面量内的非 status 数值字面量；既有常量 `models.MisfirePolicyImmediately`（log.go:62）存在；workorder_tasks/mac 任务同构处均已用常量。
   - What's unclear: 是否算 STATUS-01"顺带清"范畴（审计未列、非 status 语义、扫描器不会捕获）。
   - Recommendation: **纳入**（1 行、零风险、与同族代码一致性对齐，属"执行中新暴露位点按同规则顺带清"的合理外延）；若 planner 严守字面 scope 则留 Phase 103+，两可皆不阻塞。
   - **处置（planner，2026-09-07）**：纳入——102-04 Task 1 顺带清 1 行（替换前核对常量值 == 1）。
2. **floor_cache_impl.go:50/:71 已注册未引用的内联键（RESOLVED）**
   - What we know: `CacheKeyFloorByBuilding` 常量已存在（cache_keys.go:140），两处内联 Sprintf 与之同值。
   - What's unclear: 是否扩入 D-102-7 扫描面（CONTEXT 窄扫口径未含 operations）。
   - Recommendation: 默认不动（守 D-102-7）；planner 若纳入需同步扩扫描面文件清单并在 plan 中声明口径变更。
   - **处置（planner，2026-09-07）**：默认不动（守 D-102-7 窄扫 12 文件；Phase 103 CONV-04 扩口）。
3. **两个扫描守护测试的放置包（RESOLVED）**
   - What we know: discretion 区域。cache-key 等价测试横跨 pkg/constants 与 services/system 两宿主包。
   - Recommendation: 等价测试拆两处（pkg/constants/cache_102_test.go + internal/services/system/cache_keys_102_test.go）；cache-key 内联扫描与 status 使用点扫描各置一文件（前者可入 internal/services/system 与 invariants_92 同包复用 runtime.Caller 手法，后者入 internal/models/status_constants_test.go 同文件扩展——D-102-6 明文"同文件扩展"）。
   - **处置（planner，2026-09-07）**：按建议落地——等价测试拆 pkg/constants/cache_102_test.go（102-01/102-02）+ internal/services/system/cache_keys_102_test.go（102-02）；cache-key 内联扫描入 cache_keys_102_test.go（102-03）；status 使用点扫描入 status_constants_test.go 同文件扩展（102-04）。

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | 全部（build/test/AST 扫描） | ✓ | 1.24.5 windows/amd64（实测） | — |
| `go build ./...` 基线 | 回归纪律 | ✓ | BUILD_EXIT=0（2026-09-07 实测） | — |
| gsd-sdk CLI | 研究文档提交流程 | ✗ | — | 原生 git commit（本次已用） |
| PostgreSQL / Redis | 单元测试 | 不需要 | — | 既有测试均为 sqlite/mem-cache/mock 体系，本相无新集成测试 |

**Missing dependencies with no fallback:** 无
**Missing dependencies with fallback:** gsd-sdk（原生 git 等价，已用）

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go testing + testify（assert/require，仓库既有） |
| Config file | 无独立配置——go test 原生；守护测试自含 AST 扫描器 |
| Quick run command | `go test ./internal/services/system/ ./internal/models/ ./internal/core/ ./internal/scheduler/ ./pkg/constants/ -count=1` |
| Full suite command | `go test ./...` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| CACHE-01① | captcha 6 格式常量值 == 原字面量快照 | unit（快照表驱动） | `go test ./pkg/constants/ -run TestCaptchaCacheKeyEquivalence -count=1` | ❌ Wave 0（本相交付） |
| CACHE-01② | captcha 2 文件内联键字面量清零 | unit（AST 扫描硬失败） | `go test ./internal/services/system/ -run TestCacheKeyInlineResidue -count=1` | ❌ Wave 0 |
| CACHE-02① | 31+2 格式常量值 == 原字面量快照 | unit | `go test ./internal/services/system/ -run TestCacheKeyEquivalence -count=1` | ❌ Wave 0 |
| CACHE-02② | 10 模块文件内联键字面量清零 | unit（同 CACHE-01② 扫描器，窄扫 12 文件） | 同上 | ❌ Wave 0 |
| STATUS-01① | 21 位点替换后全量测试绿（行为等价） | regression（既有套件） | `go test ./internal/scheduler/ ./internal/services/... -count=1` | ✅ 既有 |
| STATUS-01② | models 常量值锁不倒退 | unit（AST 值锁，既有） | `go test ./internal/models/ -run TestStatusConstants -count=1` | ✅ status_constants_test.go |
| STATUS-01③ | 业务代码 status 数字字面量使用点硬失败 + geocoding 白名单 | unit（AST 使用点扫描） | `go test ./internal/models/ -run TestNoStatusLiteralUsage -count=1` | ❌ Wave 0 |
| PAGI-01 | file_handler 分页行为不变（cap=100） | regression（既有 handler/service 测试） | `go test ./internal/api/v1/system/ ./internal/services/system/ -run TestFile -count=1` | ✅ file_handler_test.go + file_service_test.go |
| PAGI-01② | utils 分页符号归零 | 编译期守护（整文件删除后 `go build ./...` 即守护） | `go build ./...` | ✅ 删除即生效 |

### Sampling Rate

- **Per task commit:** `go build ./...` + 触及包的 quick run（<30s）
- **Per wave merge:** `go test ./...`（0 失败）
- **Phase gate:** 全量套件绿 + 后端 coverage ≥78.33% + 七 gate 不倒退（go build / go test / 后端 coverage / 前端 45 dirs / lint / type-check / diff coverage——本相纯后端，前端 gate 应零变化）

### Wave 0 Gaps

- [ ] `pkg/constants/cache_102_test.go`（建议名）——CACHE-01① captcha 键等价快照
- [ ] `internal/services/system/cache_keys_102_test.go`（建议名）——CACHE-02① 键等价快照 + ② 12 文件内联扫描（可含 invariants_92 同款骨架）
- [ ] `internal/models/status_constants_test.go` 扩展（D-102-6 明文同文件）——STATUS-01③ 使用点扫描 + 白名单表
- [ ] 无框架安装缺口——go test 原生 + testify 既有

## Security Domain

> security_enforcement 未显式关闭，按规保留（本相为纯行为等价重构，安全面结论：零新增攻击面）。

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | 不触及（login:fail 键仅改名引用，语义/值/TTL 不变） |
| V3 Session Management | no | 不触及 |
| V4 Access Control | no | 不触及 |
| V5 Input Validation | no（间接） | 无新输入面；SQL 位点替换保持占位符形态（Pitfall 4），禁止引入 Sprintf 拼 SQL |
| V6 Cryptography | no | 不触及 SM2/SM3/SM4 |

### Known Threat Patterns for 本相触面

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| 键值漂移致失效 pattern 脱靶 → 脏缓存（stale data served） | Tampering | D-102-5① 等价快照测试逐字符锁值 |
| raw SQL 重构引入拼接 | Tampering（SQLi） | 占位符参数形态（Pattern 4）；`go vet` 随套件运行 |
| 扫描白名单被滥用为"豁免堆放场" | Evasion | invariants_92 先例：白名单登记必须显式 + 附原因（diff 可见）+ 零残留卫生提示 |

## Sources

### Primary (HIGH confidence)
- 本会话实跑核对（全部行号/常量/import 结论的唯一直接来源）：`grep -rn`（captcha/cache-key/status 三轮全仓扫描）、逐文件 Read（cache_keys.go / captcha.go / captcha_background.go / pkg/query/pagination.go / internal/utils/pagination.go / file_handler.go / status_constants_test.go / cache_invariants_92_test.go / 各 cache_impl 位点窗）、`go list -deps`（import 环实证）、`go build ./...`（基线绿）
- `CLAUDE.md`（项目约定：Status Value / Pagination Constants / Cache Service / operlog 回归守护 / 测试纪律）

### Secondary (MEDIUM confidence)
- `.planning/phases/102-mechanical-constants/102-CONTEXT.md`（D-102-1..9 锁定决策）
- `.planning/REQUIREMENTS.md` §CACHE/§STATUS/§PAGI（需求原文与台账行号——已被实跑清单修订）
- `.planning/notes/260907-audit-fix-tech-debt-findings.md`（F-06~F-09 审计台账——行号快照性质，漂移已实证）
- `.planning/STATE.md`（七 gate 基线 / coverage 78.33% / 继承决策）

### Tertiary (LOW confidence)
- 无（本研究未引用任何未经验证的外部/社区来源——全域为仓内事实）

## Metadata

**Confidence breakdown:**
- 位点清单（CACHE-01/02、STATUS、PAGI）：HIGH——全量 grep + 逐窗实读，行号与形态逐处核实
- import 环约束与落点二分：HIGH——go list -deps 传递依赖实证，非推测
- 守护测试设计（AST 形态/排除表）：HIGH——6 种形态全部来自实测位点归纳，宿主骨架同款在位
- 执行细节映射（int 转换口径等）：MEDIUM——编译器即时反馈域，无行为风险

**Research date:** 2026-09-07（2026-09-07 checker 修订轮增补 knowledge :134 条件后缀位点，调用点总数 46→47）
**Valid until:** 2026-10-07（稳定域——纯仓内重构，无外部漂移源；若主干在规划前有新提交，执行时按 D-102-7/广谱 grep 复核新暴露位点即可）
