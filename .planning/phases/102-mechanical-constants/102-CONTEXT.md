# Phase 102: 机械常量化（缓存键 / 状态 / 分页） - Context

**Gathered:** 2026-09-07
**Status:** Ready for planning

<domain>
## Phase Boundary

纯后端**行为等价**重构（零业务语义变化）：

1. **CACHE-01**：captcha 12 处内联缓存键（`internal/core/captcha.go` storageKey/failKey + `captcha_background.go` list/pool 键族）收敛具名常量
2. **CACHE-02**：notice/settings/duty/workorder/knowledge/network/api_endpoint/mac vendor/widget/rpa selector 等 10 模块 ~35 处内联 cache key 及失效 pattern 注册具名常量并引用
3. **STATUS-01**：剩余 12 处 status 字面量全部引用 models 具名常量（geocoding 百度 API 白名单豁免）
4. **PAGI-01**：`internal/utils/pagination.go` ParsePagination 收敛 `pkg/query` 单一口径

成功标准（ROADMAP SC-1..4）：键值与 TTL 逐处等价 + 回归测试锁；`status_constants_test.go` AST 锁全程绿 + 白名单外 grep 无残留；分页逐调用方核对清单落盘；`go build ./...` + `go test ./...` 0 失败，七 gate 不倒退。

</domain>

<decisions>
## Implementation Decisions

### 缓存键注册位置

- **D-102-1:** 业务模块 ~35 处键**全集中注册进 `internal/services/system/cache_keys.go`**（与 user/role/menu/dept/post 键族同文件），维持「cache_keys.go = 缓存键唯一真相源」既有声明。不采用模块包内注册表、不做核心/边缘切分。
- **D-102-2:** captcha 12 处键（`captcha:rate/attempts/data:%s` + `login:fail:%s` + `captcha_background.go` 的 `captcha:bg:list` / `captcha:cache:pool` 族）**进 `pkg/constants/cache.go`**——跟随本模块既有先例 `CaptchaVerifiedKeyFormat`（captcha.go:385 已引用），同族键聚齐；`internal/core` 不反向 import `internal/services/system`。
- **D-102-3:** 形态分工——cache_keys.go 内用**前缀常量 + `GetXxxKey()` helper**（同 `GetDictDataByTypeKey` 模式，CLAUDE.md 明文约定），调用处 `fmt.Sprintf` 改 helper 调用；pkg/constants 的 captcha 键用 **Sprintf 格式常量**形态（同 `CaptchaVerifiedKeyFormat`）。
- **D-102-4:** 失效 pattern **前缀派生**——`CacheKeyManager.BuildPattern` 或前缀常量 + `":*"`，不新增独立 pattern 常量（键与 pattern 单一来源防漂移）。典型位点：`internal/services/network/cache_impl.go:319` 的 `"network_device:*"`。

### 回归守护

- **D-102-5:** 缓存键**双档守护**：① 键值等价测试（常量值 == 原字面量快照，含 TTL 不变断言）；② invariants 风格**硬失败扫描**——覆盖包内内联 `fmt.Sprintf` cache key 字面量清零（白名单豁免机制）。
- **D-102-6:** STATUS 守护 = **AST 使用点扫描**，`internal/models/status_constants_test.go` 同文件扩展：业务代码中 status 数字字面量赋值/比较硬失败，geocoding 百度 API 入白名单表。不用字符串 grep（误报高）、不只验证时人工跑。
- **D-102-7:** 缓存键扫描**窄扫本相收敛面**（captcha 2 文件 + CACHE-02 的 10 模块清单）；Phase 103 由 CONV-04 自行扩口（mac_history/heatmap/rpa selector 位点闭包未迁前不入扫描面，避免临时豁免表）。
- STATUS 扫描口径取 SC-2 自然口径 = **全后端**（audit 时点全后端仅剩 12 处 + geocoding 白名单；执行中新暴露位点按同规则顺带清）。
- 分页侧**无新增守护**——`utils/pagination.go` 整文件删除即终结；CLAUDE.md 已声明 `NormalizePagination` 唯一入口，重引入属 code review 范畴。

### status 常量

- **D-102-8:** 12 处**优先引用既有常量**（绝大多数已存在：`models.JobLogStatusSuccess/Failure` log.go:94、`models.JobStatusNormal` log.go:71、`models.WorkOrderStatus*` workorder.go:17-21 全族 5 值、DutyStatusNormal 等）。确需新增常量的 model **只加用到的值**（机械相最小 diff）；新增走 CLAUDE.md 既有流程：先 `internal/models/<file>.go` 命名常量 → 同步 `status_constants_test.go` 期望表 → 业务代码按常量引用。

### 分页收敛

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
- 12 处 status 位点 → 既有常量的逐处映射（REQUIREMENTS 已注明「按实际常量存在性逐处核对」）
- CAPTCHA-01 审计清单与 CACHE-02 模块清单的逐处位点核对（以 REQUIREMENTS 行号清单为起点，实际以 grep 结果为准）

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 需求与目标

- `.planning/notes/260907-audit-fix-tech-debt-findings.md` — 审计台账：F-06~F-09 逐处位点行号 + Agent 扫描清单 C-03~C-12（CACHE-02 位点全集的权威起点）
- `.planning/REQUIREMENTS.md` §CACHE / §STATUS / §PAGI — 4 条 requirement 原文（含 12 处 status 位点行号清单）
- `.planning/ROADMAP.md` §Phase 102 — Goal / SC-1..4 / Notes（Phase 103 顺序约束）

### 先例与约定

- `.planning/milestones/v1.30-phases/99-operations口径统一/99-CONTEXT.md` — 分页 D-03-1..4 决策先例（NormalizePaginationWithMax 形态、零调用方即删除、pkg/constants 唯一常量包）
- `CLAUDE.md` §Status Value Convention / §Pagination Constants Convention / §Cache Service Convention — 三大 convention 原文

### 代码（迁移目标与参照）

- `internal/services/system/cache_keys.go` — 键注册目标文件（375 行，前缀常量 + helper 既有形态 + CacheKeyManager.Build/BuildPattern）
- `internal/core/captcha.go` / `internal/core/captcha_background.go` — CACHE-01 位点（:249,:297,:304,:326,:333,:356,:361,:367,:415,:419,:426,:503 等）
- `pkg/constants/cache.go` — captcha 键注册目标（CaptchaVerifiedKeyFormat 先例 :17）
- `internal/utils/pagination.go` + `internal/utils/utils_74_12_test.go` — PAGI-01 删除对象（含 :203 TestParsePaginationAndOffset）
- `pkg/query/pagination.go` — NormalizePagination / NormalizePaginationWithMax 唯一权威
- `internal/api/v1/system/file_handler.go:160` — ParsePagination 唯一生产调用方
- `internal/models/status_constants_test.go` — AST 锁（D-102-6 扩展宿主）
- `internal/models/log.go` / `internal/models/workorder.go` — 既有 status 常量族
- `internal/services/network/cache_impl.go:319` — 内联失效 pattern 典型位点
- `internal/scheduler/cron.go` / `internal/services/workorder/base.go:183` / `internal/scheduler/workorder_tasks.go` 等 — STATUS-01 位点（行号见 REQUIREMENTS）

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets

- `internal/services/system/cache_keys.go` 既有「前缀常量 + GetXxxKey helper + CacheKeyManager.BuildPattern」形态——~35 处注册直接续用同风格
- `pkg/constants/cache.go` CaptchaVerifiedKeyFormat + `pkg/constants/time.go` CaptchaCacheExpire——captcha 键族注册的同文件落点
- `query.NormalizePaginationWithMax`（Phase 99 已交付）——D-102-9 直接可用，无需新增 pkg 函数
- `internal/models` 既有 status 常量族（log.go / workorder.go / duty 等）——12 处大多零新增直接引用
- `internal/models/status_constants_test.go` AST 扫描框架——D-102-6 使用点扫描的扩展宿主
- `internal/services/system/cache_invariants_92_test.go` 双档扫描先例——D-102-5 扫描守护的风格参照

### Established Patterns

- 行为等价重构 + 回归测试锁（v1.30 各 phase 一贯模式）；七 gate 实测落盘
- 审计台账行号 → 执行时 grep 重核（行号是快照，以实跑为准）
- 常量值改动必须先改/同步 AST 锁测试（CLAUDE.md 强制流程）

### Integration Points

- `file_handler.go:160` 迁移后 `internal/utils` 分页代码全部消失（BuildListResponse 是另一 helper，保留不动）
- 本相注册的 rpa selector / mac vendor 键是 Phase 103 CONV-01/03 闭包迁移的前置（同文件族，本相只注册不迁闭包）
- D-102-5/6 两个新守护进常规 `go test ./...` → CI 七 gate

</code_context>

<specifics>
## Specific Ideas

- 键值等价测试用「原字面量快照表」驱动：表列 常量名 / 期望字面量 / 期望 TTL，逐行断言——audit 台账行号即快照来源
- AST 使用点扫描复用 status_constants_test.go 既有扫描器实现风格，白名单表按「包路径 + 豁免原因」登记（geocoding 百度 API 是首条）

</specifics>

<deferred>
## Deferred Ideas

- **captcha 内联 TTL 字面量**（如 failKey `1*time.Hour`、pool 项 `24*time.Hour`）顺带常量化——属 Timeout 常量约定范畴但非审计台账项，不在本相（本相只做键；TTL 值保持逐处等价并在等价测试中锁定）
- **`utils.BuildListResponse` 的 `page/pageSize` 响应键名**与前端 PageData `current` 约定的历史分叉——契约问题非常量化范畴，如需治理另立 phase

</deferred>

---

*Phase: 102-mechanical-constants*
*Context gathered: 2026-09-07*
