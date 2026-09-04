# Phase 89: PAGINATION 常量集中化 - Context

**Gathered:** 2026-09-04
**Status:** Ready for planning

<domain>
## Phase Boundary

抽取 `pkg/constants/pagination.go` 3 个分页常量(行业最佳实践对齐),统一通过 `pkg/query.NormalizePagination()` 纯函数入口处理默认值与上限截断,替换 8 个文件中 12+ 处 `current=1/pageSize=10` 硬编码;通过 AST 锁值测试 + invariants 扫描保证未来不出现重复字面量;`go test ./...` 0 失败回归。

**3 个常量最终定义**:
```go
package constants

const (
    DefaultCurrent  = 1
    DefaultPageSize = 10
    MaxPageSize     = 200  // 行业惯例对齐 (GitHub=100, Stripe=100, Twitter=100; 200 略宽,平衡 DoS 防护 + 大数据查询场景)
)
```

**审计源 (2026-09-03)**: 12+ 处 `current=1/pageSize=10` 散布于 8 个文件 + 3 处模块特异硬编码 (`knowledge_service.go:100/500`, `account_pool.go:20/200`)。模块特异值经用户深度讨论后**判定为拍脑袋决定,无业务/性能依据**,统一为 `DefaultPageSize=10` + `MaxPageSize=200`。

**v1.29 全局原则 (D-PRINCIPLE)**: 改造以**行业最佳实践**为唯一依据,允许任何形式的重构(D-05 零业务行为变更约束在此 phase 放宽)。理由:项目尚未投入使用,需达一定质量基线方可投产;要求去除硬编码、处理所有 todo、消除重复实现、合理抽象。**模块特异默认值是错误抽象** — 业务决策应通过客户端 API 调用显式传 pageSize 或配置中心实现,不应硬编码到 const 块。

**不在本 phase**: TIMEOUTS/PORT/PROTOCOL/CONCURRENCY(Phase 90);CRUD 复用(Phase 91);缓存层统一(Phase 92);config_backup 闭环(Phase 93);前端 API 工厂化(Phase 94);v1.28 收口(Phase 95)。

</domain>

<decisions>
## Implementation Decisions

### 常量包归属 (Area 1)

- **D-01:** 在 `pkg/constants/pagination.go` 新建 leaf const package,定义 3 个分页常量(纯 const 块,无函数)。
- **D-02:** `pkg/query.PaginationRequest.Normalize()` 内部委托调用 `constants.NormalizePagination`,消除 `pkg/query/pagination.go` 内的 `1`/`10` 字面量(单向依赖 `pkg/query → pkg/constants`,零循环风险)。
- **D-15 (修订 D-03):** `PaginationRequest` 的 `binding:"min=1,max=100"` 改为 `binding:"min=1"`。`100` 字面量违反 D-PRINCIPLE(去除硬编码);200 截断由 `NormalizePagination` 负责。**业务行为变更**:pageSize>100 的请求从 400 变 200(以静默裁减),项目接受此变更。
- **D-19 (修订 D-04):** `NormalizePagination` helper **不**放 `pkg/constants/pagination.go`(违反 leaf 纯 const 惯例),改放 `pkg/query/pagination.go` 与 `PaginationRequest` 同 home,内部引用 `pkg/constants` 3 个常量。

### 范围与 helper 设计 (Area 2)

- **D-05:** `NormalizePagination(current, pageSize int) (int, int)` 是**纯函数**(Go 惯例,匹配 `strings.TrimSpace` / `filepath.Clean`)。签名采用值返回而非指针原地改。
- **D-06:** 现有 4 处 `setPaginationDefaults` / `Normalize` 重复实现全部统一为调用 `constants.NormalizePagination`(注:D-19 后实际是 `pkg/query.NormalizePagination`):(1) `internal/api/v1/rpa/handler_helpers.go:47` setPaginationDefaults; (2) `internal/api/v1/monitor/cache_handler.go:50` setPaginationDefaults; (3) `internal/api/v1/system/ad_domain_handler.go:40-44` 内联守卫; (4) `pkg/query.PaginationRequest.Normalize()`。`ad_domain` 当前 `== 0` 统一为 `<= 0`(附带 bug 修复:负值原本穿透,现在归一为 1)。
- **D-07:** AST 锁值测试**双重防护**:`pkg/constants/pagination_test.go` 锁 3 个常量值(用 `expectedPaginationValues = map[string]int{...}` 模式,参考 `internal/utils/operlog/regression_test.go`)+ invariants 扫描全仓 .go,发现 `*current = 1` / `pageSize = 10` 字面量赋值且同 file/func 未引用 `pkg/constants` 时,报告为 warning(不 fail,记录到测试日志供人工 triage)。
- **D-10 (修订):** `NormalizePagination` 内部启用 `MaxPageSize=200` 截断:`if pageSize > MaxPageSize { pageSize = MaxPageSize }`。**200 是行业惯例的对齐**(GitHub=100, Stripe=100, Twitter=100;200 略宽,平衡 DoS 防护 + 大数据查询场景)。原计划 10000 已被用户判定为反模式 — 实质等于无上限,无 DoS 防护价值。

### 命名规范 (Area 3) — 已重判

- **D-08 (删除):** ~~KnowledgeDefaultPageSize=100 + KnowledgeMaxPageSize=500~~ — **删除**。用户深度讨论判定:
  - knowledge_service.go:490 `pageSize = 100` 是 2024 年早期拍脑袋决定,无用户研究/性能数据支撑
  - knowledge_service.go:492 `pageSize = 500` cap 偏高(行业惯例 100-200)
  - 业务决策不应硬编码到 const,应由客户端 API 调用显式传 pageSize 或配置中心实现
- **D-09 (删除):** ~~AccountPoolDefaultPageSize=20~~ — **删除**。同理,无业务依据。account_pool.go:207 `if pageSize < 1 || pageSize > 200 { pageSize = 20 }` 改为走 `NormalizePagination` 标准逻辑(default=10, max=200)。
- **D-16 (删除):** ~~knowledge 500 字面量走常量~~ — **不再适用**(D-08 已删除该常量)。knowledge_service.go:492 整段 `else if pageSize > 500 { pageSize = 500 }` 删除,改由 `NormalizePagination` 统一 200 cap 处理。
- **D-14 (简化):** ~~领域特异 limit 留调用方~~ — **简化**:`NormalizePagination` 是唯一的 limit 入口,所有 endpoint 走统一 default=10 + max=200。**无领域特异 limit**。

### 实施策略 (Area 5)

- **D-11:** Pilot 先行,89-01 = 新建 constants pkg + `knowledge_service.go` 迁移跑通 + AST 锁值测试 + invariants 扫描;89-02 = 复制模式迁移剩余 7 个文件(handler 5 + service 2) + `pkg/query.PaginationRequest.Normalize()` 内部委托。
- **D-13:** Pilot 文件选 `internal/services/knowledge_service.go`(同时覆盖 `pageSize = 10`(line 177)与 `pageSize = 100 + 500 cap`(line 490/492)两个不同代码路径,验证统一逻辑可同时替代)。

### 项目级同步

- **D-12:** 在 `CLAUDE.md` 已有 `Status Value Convention` 段后新增 `## Pagination Constants Convention` 段,锁定纪律:"所有分页默认值/上限走 `pkg/constants/pagination.go`(3 个常量),不再字面量;`pkg/query.NormalizePagination` 是首选守卫入口"。

### Claude's Discretion (no-op 评估)

- **D-17:** `TotalPages` math(`int(total)/pageSize`)不需常量——`pageSize` 是运行时值,非硬编码。
- **D-18:** `binding:"min=1"` 不抽常量——`1` 只在 binding 一处出现,抽常量属过度抽象。

### 业务行为变更清单 (D-PRINCIPLE 允许)

| 端点 | 旧行为 | 新行为 | 用户接受 |
|------|--------|--------|----------|
| `PaginationRequest` (所有用此结构的端点) | binding max=100 → HTTP 400 | 移除 max → pageSize>200 静默裁减到 200 | ✓ |
| `ad_domain_handler` | `== 0` 守卫,负值穿透到 DB | `<= 0` 守卫,负值归一为 1 | ✓ (bug 修复) |
| `knowledge_service.go:SearchKnowledgeArticles` | default=100, max=500 | default=10, max=200 | ✓ |
| `account_pool.go:ListAll` | `if <1 \|\| >200` 回退到 20 | `if <=0` 默认 10; `if >200` clamp 到 200(标准 NormalizePagination 行为) | ✓ |
| 任意端点 pageSize=99999 | 透传(可能慢查询) | 静默裁减到 200(防 DoS) | ✓ |
| 任意端点 pageSize=0/-1 | 各自 `setPaginationDefaults` 守卫 | 统一走 `NormalizePagination` | ✓ |

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### v1.29 milestone 全局
- `.planning/ROADMAP.md` — v1.29 milestone § Phase 89 (Goal / 11 requirements / 3 plans / 5 Success Criteria)
- `.planning/REQUIREMENTS.md` — `PAGINATION-01..11` (本 phase 11 项需求)
- `.planning/PROJECT.md` — v1.29 Current Milestone 段,D-01..D-06 locked decisions + D-PRINCIPLE 行业最佳实践

### 现有分页相关代码
- `pkg/query/pagination.go` — 既有 `PaginationRequest` struct + `Normalize()` 方法(将被内部委托给 `NormalizePagination`)+ `NewPaginatedResult` helper
- `pkg/query/pagination_80_05_test.go` — 既有 `Normalize()` 行为测试(Phase 80 TAIL 收口时落地)
- `internal/api/v1/rpa/handler_helpers.go:47` — `setPaginationDefaults(current, pageSize *int)` (pilot 之外将被替换)
- `internal/api/v1/monitor/cache_handler.go:50` — `(h *CacheHandler) setPaginationDefaults` (值返回风格)
- `internal/api/v1/system/ad_domain_handler.go:40-44` — 内联守卫(用 `== 0`,与 `<= 0` 不一致,需统一)
- `internal/api/v1/system/notice_user_handler.go:111/114` — ROADMAP 标注的待替换位置
- `internal/services/workorder/base.go:120/124` + `periodic.go:92/96` — service 层内联
- `internal/services/asset/reconciliation_service.go:506/510` + `fix_suggestion_service.go:169/173/177` — service 层
- `internal/services/knowledge_service.go:171/175/488/492` — **pilot**,含两个代码路径(LineArticles 走 1/10,SearchArticles 走 100/500)
- `internal/services/addomain/account_pool.go:207` — `<1 \|\| >200` 回退到 20 的反常逻辑(将被标准 NormalizePagination 替代)

### AST 锁值测试参考 (内部 pattern)
- `internal/utils/operlog/regression_test.go` — `expectedOperTypeValues = map[string]int{...}` 模式 + `TestOperTypeConstantStability` 命名先例
- `internal/models/status_constants_test.go` — 多文件多 family AST scan 模式(本 phase invariants 扫描可参考)
- `internal/models/status_constants_test.go:TestStatusConstantsStability` — 双向 assertion 模式(constant 忘记注册 → fail)

### 项目级约束
- `CLAUDE.md` — `Status Value Convention` (本 phase 新增 `Pagination Constants Convention` 段参考其格式)
- `CLAUDE.md` — `操作日志记录约定 (operlog convention)`(本 phase 业务行为变更如触发 operlog 需遵守)
- `CLAUDE.md` — `Compilation & Build Verification` (Phase 89 完成后 `go build ./...` 必须 0 错误)
- `CLAUDE.md` — `Testing` (Phase 89 完成后 `go test ./...` 必须 0 失败,既有 1688 测试不回归)

### 行业最佳实践参考
- GitHub REST API: default 30, max 100
- Stripe API: default 10, max 100
- Twitter API: default 20, max 100
- **本项目决策**: default 10, max 200(略宽于上述,平衡 DoS 防护 + 内部企业系统的"大查询"场景)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `pkg/query/pagination.go:PaginationRequest.Normalize()` — 既有入口,D-02/D-19 决定内部委托给 `NormalizePagination`,不删除方法签名(零破坏现有调用方)
- `internal/utils/operlog/regression_test.go:expectedOperTypeValues` — AST 锁值测试模板(`map[string]int{...}` + `TestXxxConstantStability` 命名)
- `internal/models/status_constants_test.go:TestStatusConstantsStability` — AST scan 模板(`watchedStatusPrefixes` + 双向 assertion)
- `pkg/query/pagination_80_05_test.go:TestPag8005_Normalize_Offset` — `Normalize()` 既有行为测试用例,可作为 pilot 回归基线

### Established Patterns (D-PRINCIPLE 对齐依据)
- **Leaf const package pattern**: Go 标准库(`io/fs`, `net/http`, `time`)普遍采用 leaf pkg 纯 const,无业务函数。本 phase 严格遵守(D-01 + D-19)
- **Pure value transformation pattern**: `strings.TrimSpace(s) string`, `filepath.Clean(p) string`, `t.Add(d) time.Time`——纯函数接收值返回新值,无副作用(D-05)
- **Single Source of Truth (SSOT)**: D-PRINCIPLE + D-02/D-14 要求所有字面量走 `pkg/constants`,helper 内部引用 const,无第二字面量源
- **行业 API 分页惯例**: GitHub/Stripe/Twitter 全部"小默认 + 小 cap";**禁止**在 const 块里硬编码模块特异默认值(应是配置或客户端控制)(D-08/D-09 删除)
- **Test-before-refactor for AST lock**: Phase 75 (QUIRK 全修) 的 "修复 + 同 commit 翻转断言 + 回归用例 + 原子 commit" 五步法,本 phase 借鉴用于 invariants 扫描测试的引入

### Integration Points
- `internal/api/v1/rpa/handler_helpers.go` 调用方:`rpa/ai_handler.go`, `rpa/credential_handler.go` (无需改动,setPaginationDefaults 内部行为不变)
- `internal/api/v1/monitor/cache_handler.go` 调用方:`cache_handler.go:138`, `cache_handler.go:332` (无需改动)
- `internal/api/v1/system/ad_domain_handler.go` 调用方:本文件内部 (D-06 替换为 `NormalizePagination`)
- `pkg/query.PaginationRequest.Normalize()` 调用方:目前 v1.27 期间尚少,Phase 80 TAIL 测试 `pkg/query/pagination_80_05_test.go` 引用

</code_context>

<specifics>
## Specific Ideas

### D-PRINCIPLE 来源 (用户原话)
> "做改造时,不要依据当前项目架构,只以行业最佳实践作为依据,因为当前项目还没有投入使用,需要达到一定质量才行,因此允许进行任何形式的重构,要求是:去除硬编码,处理所有todo,不要重复实现,合理抽象等等。"

### 关于"模块特异默认值"的深度思考 (用户输入)
> "为什么特定模块会有特定的常量,不能所有模块都是用defaultcurrent,和defaultpagesize吗?maxpagesize的作用是什么,之前设置maxpagesize为10000是因为要统计总量,现在端点改造,总量已经不依赖这个常量统计了,请深度思考,最终是否可以只保留两个常量,即defaultcurrent和defaultpagesize"

**深度分析结论**:
1. **知识库 `pageSize=100/500` 是 2024 年拍脑袋决定**,无用户研究/性能依据
2. **AD 账号池 `pageSize=20` 同理**
3. **MaxPageSize=10000 实质等于无上限**,无 DoS 防护价值(网关层防 DoS 更合理)
4. **"模块特异默认值" 是错误抽象** — 业务决策应通过:
   - 客户端 API 调用显式传 pageSize(API contract 灵活)
   - 或配置中心(运维可调)
   - 不应硬编码到 const 块

**用户最终选择**: 保留 3 常量(加 MaxPageSize=200) — 略宽于 GitHub/Stripe 100,平衡 DoS 防护 + 大数据查询场景

### 业务行为变更 (用户已显式接受)
- binding 移除 max=100
- 知识搜索 default 100→10, max 500→200
- AD 账号池 default 20→10,`>200` 反常回退逻辑改为标准 clamp
- ad_domain `== 0` 守卫改为 `<= 0`(bug 修复)

### Pilot 验证要点
- `knowledge_service.go` 同时覆盖 `pageSize = 10`(line 177)与 `pageSize = 100 + 500 cap`(line 490/492)两个不同代码路径
- 迁移后两个代码路径都走 `NormalizePagination`,验证统一逻辑可同时替代
- `pkg/query/pagination_80_05_test.go` 既有 `TestPag8005_Normalize_Offset` 行为不变
- `go test ./internal/services/...` 必须全过
- `go build ./...` 0 错误

</specifics>

<deferred>
## Deferred Ideas

### Workstream 同步问题 (非本 phase 范围,需后续 `/gsd:progress` 修复)
- `.planning/workstreams/milestone/STATE.md` 与 `.planning/workstreams/milestone/ROADMAP.md` 仍停留在 v1.27 SHIPPED,未同步到 v1.29
- `gsd-tools.cjs init phase-op 89` 默认查 workstream 路径返回 `phase_found: false`,本 discuss-phase 以 `.planning/ROADMAP.md` 为准继续
- 修复方式待 Phase 95 closeout 评估:workstream 是更新到 v1.29 还是重命名为 `v1.29-milestone`

### 未来 v1.29 其他 phase 引用
- Phase 90 (TIMEOUTS/PORT/PROTOCOL/CONCURRENCY):可借鉴本 phase 的 leaf const + invariants 扫描模式
- Phase 91 (CRUD 复用 `base.Repository[T]`):独立的 CRUD 抽象,不影响
- Phase 92 (缓存层统一):缓存层三处架构合并,可考虑把 `CacheServiceBase` 共同方法放 `pkg/query` 或新 `pkg/cache` 抽象

### 概念延展 (超出 Phase 89 范围)
- 当前 1688 测试 + 45/45 dirs Gate 在 Phase 89 完成后必须保持(CLAUDE.md §Compilation & Build Verification 强制)
- 后续 Phase 95 closeout 时,本 phase 的 invariants 扫描工具可考虑升级为 fail-on-hit(届时若全仓已无字面量,warning 应为 0,加 fail 也无影响)

### 未来可考虑的扩展(不在本 phase)
- 配置中心集成:把 pageSize default 做成可配置项(目前 const 化是正确的简化)
- 客户端 SDK:如果前端有常用 pageSize pattern,可在前端 SDK 提供 helper(如 `usePagination({ defaultSize: 10 })`)

</deferred>

---

*Phase: 89-PAGINATION 常量集中化*
*Context gathered: 2026-09-04*
</content>
</invoke>