---
last_updated: 2026-08-13
update_trigger: v1.21 milestone INITIATED — ROADMAP drafted (Phases 57-61, regression fix for v1.6 Phase 16; Phase 61 conditional on Phase 60 AUTH-03=启用)
last_plan_update: 2026-08-13 — Phase 61 plans finalized after plan-checker revision (2 plans, Wave 1 Plan 01 / Wave 2 Plan 02 sequential)
previous_update: 2026-07-10 after v1.20 milestone SHIPPED + ARCHIVED
---

# Roadmap: XingRan-Next 运维管理系统

## Milestones

- ✅ **v1.0 工位导入部门/用户关联** — Phases 1-2 (shipped 2026-04-16)
- ✅ **v1.1 信息点导入设备端口关联** — Phase 3 (shipped 2026-04-16)
- ✅ **v1.2 可配置仪表盘生产级改造** — Phases 4-7 (shipped 2026-04-21)
- ✅ **v1.3 技术债清理** — Phases 8-10 (shipped 2026-04-27)
- ✅ **v1.4 MAC地址采集优化** — Phase 11 (shipped 2026-05-09)
- ✅ **v1.5 MAC地址历史数据管理** — Phases 12-15 (shipped 2026-06-15)
- ✅ **v1.6 API密钥管理系统** — Phase 16 (shipped 2026-05-19) ← **本里程碑回归对象**
- ✅ **v1.7 前后端加密配置同步** — Phase 17 (shipped 2026-05-20)
- ✅ **v1.8 登录端点加密增强** — Phase 18 (shipped 2026-05-21)
- ✅ **v1.9 AD域控集成扩展** — Phases 19-20 (shipped 2026-05-24)
- ✅ **v1.10 网络设备权限修复** — Phase 21 (shipped 2026-05-24)
- ✅ **v1.11 AD组自动同步系统** — Phase 23 (shipped 2026-05-26)
- ✅ **v1.12 深信服桌面云集成 (22A+22B)** — Phases 22A/22B (shipped 2026-06-02)
- ✅ **v1.13 资产管理模块** — Phase 26 (shipped 2026-06-08)
- ✅ **v1.14 全局列自定义** — Phase 27 (shipped 2026-06-09)
- ✅ **v1.15 工位设备关联 + 部门物理位置映射** — Phases 28 + 39 (shipped 2026-06-10 / 2026-06-25)
- ✅ **v1.16 技术债清理 (Tech-Debt Cleanup)** — Phases 40-41 (shipped 2026-06-26)
- ✅ **v1.17 资产对账 (Asset Reconciliation)** — Phases 42-46 + Phase 47 root-cause (shipped 2026-07-03)
- ✅ **v1.18 网络设备硬件清单 (Device Component Serials)** — Phase 48 + Phase 49 gap closure (shipped 2026-07-04)
- ✅ **v1.19 网络设备写命令 (Network Device Port Write Operations)** — Phases 50-55 (shipped 2026-07-08) — see [milestones/v1.19-ROADMAP.md](milestones/v1.19-ROADMAP.md)
- ✅ **v1.20 网络设备 VLAN + 端口绑定** — Phase 56 (shipped 2026-07-10) — see [milestones/v1.20-ROADMAP.md](milestones/v1.20-ROADMAP.md)
- 🚧 **v1.21 API Key 认证链修复 + 能力补全 (API Key Auth Chain Repair + Feature Completion)** — Phases 57-61 (in planning; regression fix 57-60 + post-enable feature completion 61)

---

## Phases (v1.21) — IN PLANNING

**Milestone Goal:** 修复 API Key 认证系统的 P0/P1/P2 缺陷,回归 v1.6「API 密钥管理系统」(Phase 16)的可用性与可观测性,让 MultiAuth 认证链代码就绪可启用,使用日志真实反映请求结果。

**Regression scope (调查已确认的根因):**

| 缺陷 ID | 根因 | 文件:行 | 类别 |
|---------|------|---------|------|
| P0-1 | MultiAuth 及下游中间件未挂载(死代码) | router.go / apikey.go | AUTH |
| P0-2 | `setUserContextForAPIKey` 把 `*models.APIKey` 断言为局部值类型 `apiKeyType`,恒 false | apikey.go:146-179 | AUTH |
| P1-1 | 前端 GET/PUT/DELETE vs 后端 POST 路由契约不匹配 → 404 | apikey_router.go / apikey.ts | CONTRACT |
| P1-2 | 使用日志 goroutine 在 `c.Next()` 前记录,StatusCode/Duration/Success 全零 → successRate≈0% | apikey.go:60-75 | OBSERV |
| P2-a | `RateLimitByScope` 响应头 `string(rune(int))` 编码错误 (Limit=100 → "d") | apikey.go:274-275 | QUAL |
| P2-b | 使用日志 goroutine 复用 `c.Request.Context()`,请求结束 → ctx.Canceled,记录丢弃 | apikey.go:66 | OBSERV |
| P2-c | API Key `Key` 字段明文存储 | apikey_service.go / migration_085 | SEC |
| P3 | migration_085 `idx_api_keys_key` 与 `uniqueIndex` 重复 | migration_085 | SEC |

**Phase Numbering:** 从 v1.20 末尾 Phase 56 续编 (57-61)。整数 phase 为计划里程碑工作;小数 phase (如 57.1) 为紧急插入。

- [x] **Phase 57: 认证链核心修复 + 回归测试** - 修复 setUserContextForAPIKey 类型断言 (P0-2),消除 MultiAuth 下游死代码 (P0-1),用集成测试锁住认证链防止 P0-2 回归
- [x] **Phase 58: 前后端路由契约对齐** - 修复前端 getAPIKey/updateAPIKey/deleteAPIKey 三个操作 404 (P1-1),与后端 POST 路由方法对齐;+ 修复字段命名契约断裂(CONTRACT-02,前端 snake→camelCase,Create/Update 不再静默丢字段、List/详情复合字段正常显示) — code-complete,SC#1-SC#4 E2E 延期(dev DB 性能,见 58-01-SUMMARY)
- [x] **Phase 59: 可观测性 / 使用日志修复** - 修复使用日志记录时机 (P1-2),让 successRate 可信 (P1-2 连锁),消除 ctx 取消竞态 (P2-b)
- [x] **Phase 60: 安全加固与启用决策** - MultiAuth 启用决策 + 安全评估,密钥哈希存储决策,移除重复索引,修复限流头编码 (P2-a)
- [x] **Phase 61: 资源级权限矩阵 + 限流生产调优** - MultiAuth 启用后落地 RequireAPIKeyResourcePermission 的 resource 参数真实生效 (AUTH-04, ex-FUTURE-APIKEY-01) + RateLimitByScope 生产接入与调优 (QUAL-03, ex-FUTURE-APIKEY-02);仅在 Phase 60 AUTH-03=启用 时执行

---

## Phase Details

### Phase 57: 认证链核心修复 + 回归测试

**Goal**: API Key 认证链代码功能正确——`setUserContextForAPIKey` 真实把上下文写入 gin context (修复 P0-2 类型断言恒 false),MultiAuth 及其下游 `RequireScope` / `RequireAPIKeyResourcePermission` / `RateLimitByScope` 类型签名、参数传递、作用域匹配逻辑经审查正确且具备被路由挂载的条件 (消除 P0-1 死代码),并由集成测试锁住"API Key 认证 → 上下文写入 → 作用域校验"完整链路,防止 P0-2 类型断言回归。

**Depends on**: Nothing (本 milestone 第一个 phase,所有后续 phase 依赖认证链代码就绪)

**Requirements**: AUTH-01, AUTH-02, QUAL-02

**Success Criteria** (what must be TRUE):

1. 携带有效 API Key 的请求经过 MultiAuth → `setUserContextForAPIKey` 后,gin context 中 `user_id` / `api_key_id` / `scopes` / `auth_type="api_key"` 四个键被下游 handler 成功读取且值非空(由新增集成测试断言,而非手工打印);P0-2 恒 false 分支消除(直接 import `internal/models` 包,无 `interface{}` workaround 残留)
2. `MultiAuth` / `RequireScope` / `RequireAPIKeyResourcePermission` / `RateLimitByScope` 四个中间件经类型签名与调用路径审查无死代码缺陷,`services.NewUsageLogger` 与 `services.NewRateLimiter` 在代码库内有真实实例化调用点(`grep -rn` 证据,而非仅定义)
3. 新增集成测试覆盖完整链路三条路径:① 有效 key + 正确 scope → 通过;② 有效 key + 缺失 scope → 403;③ 无效 key → 401;且原 `apikey_test.go` 3 个纯函数测试与全量 `go test ./...` 不回归
4. `go build ./...` 退出码 0,无 "interface{} 避免循环导入" 型 workaround 残留注释

**Plans**: 1 plan (single-wave, fix-then-test)

Plans:

- [x] 57-01-PLAN.md — 修复 setUserContextForAPIKey 类型断言 (AUTH-01/P0-2) + 重写 RequireAPIKeyResourcePermission 反模式 (AUTH-02/P0-1) + 创建集成测试锁住三路径链路 (QUAL-02) + D-02 构造函数证据

---

### Phase 58: 前后端路由契约对齐

**Goal**: 前端 API Key 管理页对单条记录的查询、更新、删除三个操作不再返回 404,前后端路由方法与路径完全对齐,操作真实生效。

**Depends on**: Phase 57 (认证链就绪后再校验契约,避免与 P0 修复交叉污染)

**Requirements**: CONTRACT-01, CONTRACT-02 (CONTRACT-02 为 2026-08-13 discuss 审计发现的字段命名断裂,同属契约层故同 phase 吸收)

**Success Criteria** (what must be TRUE):

1. 前端 API Key 管理页点击单条记录的"编辑"操作,前端能成功拉取该 key 的当前详情字段(返回码 `code:0`,无 404;表单字段完整回填)
2. 前端修改 API Key 属性(如名称、作用域、IP 白名单)并保存后,后端持久化成功,列表刷新展示更新后的值(返回码 `code:0`,无 404;数据库行 `updated_at` 刷新)
3. 前端对单条 API Key 执行"删除"操作后,记录从列表中消失(软删除生效),重复删除或再访问返回明确错误而非 404 路由缺失
4. 前后端方法/路径对齐方向由 phase 内 discuss 决策并落记录(选项 A: 改前端用 POST 对齐 v1.6 后端既有模式;选项 B: 后端补 RESTful GET/PUT/DELETE),决策与 `apikey_router.go` 当前注册路径一致或显式迁移

**Plans**: 1 plan (single-wave; CONTRACT-01 路由方法对齐 + CONTRACT-02 字段命名 camelCase 对齐, 3 个前端文件, 后端零改动)

Plans:

- [x] 58-01-PLAN.md — 前端 apikey.ts 三函数改 POST 对齐 (CONTRACT-01/D-01) + types/apikey.ts & index.tsx 字段命名 snake→camelCase 对齐 (CONTRACT-02/D-02/D-03/D-04/D-05) + 端到端 SC#1-SC#4 验证 checkpoint (D-06) — code-complete;Task 1+2 已提交 1978935/6a4c772,自动化门全绿;SC#1-SC#4 E2E 延期(dev DB 性能,见 58-01-SUMMARY)

**UI hint**: yes (前端 API Key 管理页面编辑/删除交互流程,涉及 `src/api/apikey.ts` + 列表/表单组件)

---

### Phase 59: 可观测性 / 使用日志修复

**Goal**: API Key 使用日志真实反映请求结果——记录时机移到请求处理完成之后,`StatusCode` / `Duration` / `Success` 取真实值,`successRate` 聚合可信,异步 goroutine 不被请求生命周期取消竞态污染。

**Depends on**: Phase 57 (修复必须基于已就绪的认证链,日志记录点在 MultiAuth goroutine 内)

**Requirements**: OBSERV-01, OBSERV-02, OBSERV-03

**Success Criteria** (what must be TRUE):

1. 发起一次成功的 API Key 请求(2xx 响应)后,`sys_api_key_usage_log` 表对应记录的 `StatusCode` 落在 2xx,`Duration > 0`,`Success = true`(数据库行实证,而非代码推断)
2. 发起一次失败的 API Key 请求(权限不足 403 / 错误 key 401 / 限流 429)后,对应记录的 `Success = false`,`StatusCode` 为真实错误码
3. `GetUsageLogSummary` 返回的 `successRate` 在混合成功/失败请求后落入 (0%, 100%) 开区间内可信值,不再恒 ≈ 0%(P1-2 连锁消除)
4. 客户端主动断开连接或请求 context 取消后,使用日志记录仍能完整写入数据库——异步 goroutine 使用独立的、不被请求生命周期取消的 `context.Background()` 派生 context(P2-b 消除;新增测试模拟请求取消,断言记录已入库)
5. `go test ./internal/middleware/... ./internal/services/...` 全绿,新增"记录时机"与"独立 context"用例覆盖 P1-2 与 P2-b 防回归

**Plans**: 2 plans (Wave 1 源码修复 + Wave 2 测试验证)

Plans:
**Wave 1**

- [x] 59-01-PLAN.md — 源码修复: apikey.go 记录点后移到 c.Next() 之后 + 填 StatusCode/Duration/Success + 去冗余 goroutine (D-01/D-02a/OBSERV-01) + usage_logger.go logUsageAsync 改 detached context + applogger 替换 _ = err (D-02/D-04/OBSERV-03)

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 59-02-PLAN.md — 测试验证: SC#1/#2 真实 DB 时序/失败集成测试 + SC#4 cancel-race 单元测试 + SC#3 混合 successRate 聚合测试 + waitForUsageLog helper (require.Eventually) + D-03 真实 sqlite 文件 DB + D-03a 既有 fake 测试原样保留

---

### Phase 60: 安全加固与启用决策

**Goal**: 完成 MultiAuth 路由挂载启用与 API Key 哈希存储两项安全决策(产出决策记录),落地可直接执行的硬化项(限流响应头编码修复、重复索引移除),使认证链具备生产启用条件或在显式理由下推迟启用。

**Depends on**: Phase 57 (MultiAuth 代码就绪)、Phase 59 (使用日志可信,为启用决策提供可观测基础)

**Requirements**: AUTH-03, SEC-01, SEC-02, QUAL-01

**Success Criteria** (what must be TRUE):

1. **AUTH-03**: phase 内 discuss 完成 MultiAuth 路由挂载启用决策并产出决策记录,内容含:作用域继承 (`InheritPerms`) 行为、IP 白名单语义、与 JWT 中间件的优先级与回退关系、对现有认证链的安全影响评估;决策为"启用"则附挂载点清单与权限校验矩阵,为"推迟"则附触发条件与再次评估时机
2. **SEC-01**: phase 内 discuss 完成 API Key 存储方式决策并产出决策记录(明文 vs SM3 / argon2id 哈希);若决定迁移,含平滑过渡方案(兼容期双读、回填脚本)与回滚方案;若保留明文,含接受理由与补偿控制(如 DB at-rest 加密、访问审计、轮换流程)
3. **SEC-02**: migration 中 `idx_api_keys_key` 冗余索引被移除,`key` 字段仅保留一个 `uniqueIndex`;数据库 schema introspection (或迁移脚本 idempotent 验证) 证实索引已收敛
4. **QUAL-01**: 触发限流(或单元测试构造 `result.Limit=100` / `result.Remaining=99`)后,响应头 `X-RateLimit-Limit` / `X-RateLimit-Remaining` 为数字字面量字符串 `"100"` / `"99"`(用 `strconv.Itoa`),不再是 `string(rune(100))` 产生的 `"d"` 单字符;curl 或 httpie 验证可被标准工具解析为整数

**Plans**: 2 plans (Wave 1, parallel;互不耦合)

Plans:

- [x] 60-01-PLAN.md — AUTH-03 router MultiAuth + RateLimitByScope 挂载 + 4 维度决策记录 + QUAL-01 apikey.go strconv.Itoa 修复 + TestRateLimitHeaderEncoding(单测) + TestRateLimitHeadersInResponse(集成测)
- [x] 60-02-PLAN.md — SEC-01 models/api_key.go schema 三列替换 + apikey_service.go 三函数改造 + hashAPIKey/generateSalt helper + apikey_service_test.go 按新 schema 重写 + SEC-02 手动 SQL (DROP INDEX IF EXISTS) + 双 dialect 验证查询

---

### Phase 61: 资源级权限矩阵 + 限流生产调优

**Goal**: MultiAuth 生产启用后,落地 API Key 资源级细粒度权限(`RequireAPIKeyResourcePermission` 的 `resource` 参数真实生效,含 resource→permission 映射与继承权限下的资源校验)与 `RateLimitByScope` 生产接入与调优,使 API Key 的权限控制与限流按设计真实生效。

**Depends on**: Phase 60 (AUTH-03 启用决策=启用;本 phase 仅在 MultiAuth 生产挂载后执行,若 AUTH-03=推迟则本 phase 随之 defer)

**Requirements**: AUTH-04 (ex-FUTURE-APIKEY-01), QUAL-03 (ex-FUTURE-APIKEY-02)

**Success Criteria** (what must be TRUE):

1. `RequireAPIKeyResourcePermission(resource, action)` 的 `resource` 参数不再被忽略——resource→permission 映射接入,继承权限 (InheritPerms) 下的细粒度资源校验经测试覆盖验证(有效 key 有资源权限→通过 / 无资源权限→403)
2. `RateLimitByScope` 在 MultiAuth 已挂载的生产路由上接入,限流按作用域生效;`X-RateLimit-Limit` / `X-RateLimit-Remaining` 可被标准工具解析为整数(衔接 Phase 60 QUAL-01 的 strconv.Itoa 修复);多 scope key 的限流作用域选择逻辑正确(不再任意只取首个 scope)
3. 资源权限矩阵 + 限流配置均有测试覆盖;`go test ./...` 全绿

**Plans**: 2 plans (Wave 1: Plan 01; Wave 2: Plan 02 — sequential; Plan 02 依赖 Plan 01,因两者都修改 `internal/middleware/apikey.go`,sequential 消除 parallel write race)

Plans:

**Wave 1**

- [x] 61-01-PLAN.md (Wave 1) — AUTH-04: pkg/permission/resource_action_map.go 静态 map(D-01/D-02/D-03/D-04)+ MultiAuth+setUserContextForAPIKey InheritPerms 加载(D-06/D-07/D-09)+ username/nickname 修正(D-10)+ RequireAPIKeyResourcePermission 接入 map(D-03)+ router.go 调用形态变更 + map 单测 + 5 个 ResourcePermission 中间件单测 + 5 个 InheritPerms sqlite 集成测

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 61-02-PLAN.md (Wave 2, depends_on: ["61-01"]) — QUAL-03: RateLimitByScope 接收 action 参数(D-11)+ getRequiredScope 扩展 list→read(D-11)+ 提取纯函数 SelectScope(D-12)+ getScopeFromContext 薄壳包装 SelectScope(D-12)+ CacheConfigService 新增 12 个 rate_limit.* 配置键(D-15/D-16/D-17,拆分为两次单占位符查询沿用既有 pattern)+ RateLimiter 改造为配置驱动接收 RateLimitProvider(D-18)+ reload race 语义(D-19)+ router.go 调用形态变更(`core.CacheConfigService` 字段访问,**非** getter)+ modify existing rate_limiter_test.go(449 行,7 个既有测试函数迁移到 `NewRateLimiter(provider)`,移除 `limiter.limits` 断言)+ 既有 TestRateLimitHeadersInResponse 签名更新 + 9 个 SelectScope 纯函数单测 + 7 个 RateLimitByScope 中间件单测 + 7 个 RateLimiter 配置驱动单测 + 5 个 CacheConfigService rate_limit 单测

**Conditional**: 本 phase 仅在 Phase 60 AUTH-03 决策=启用 时执行;若决策=推迟启用,本 phase 随之 defer(记录触发条件与再次评估时机)。

---

## Progress

**Execution Order:**
Phases execute in numeric order: 57 → 58 → 59 → 60 → 61 (Phase 61 conditional on Phase 60 AUTH-03=启用)

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 57. 认证链核心修复 + 回归测试 | v1.21 | 1/1 | Complete    | 2026-08-13 |
| 58. 前后端路由契约对齐 | v1.21 | 0/TBD | Not started | - |
| 59. 可观测性 / 使用日志修复 | v1.21 | 2/2 | Complete    | 2026-08-13 |
| 60. 安全加固与启用决策 | v1.21 | 2/2 | Complete    | 2026-08-13 |
| 61. 资源级权限矩阵 + 限流生产调优 | v1.21 | 2/2 | Complete    | 2026-08-13 |

---

## Coverage Map

14/14 v1 requirements mapped to exactly one phase (0 orphans, 0 duplicates):

| Requirement | Phase | Category |
|-------------|-------|----------|
| AUTH-01 | Phase 57 | AUTH (P0-2) |
| AUTH-02 | Phase 57 | AUTH (P0-1) |
| QUAL-02 | Phase 57 | QUAL |
| CONTRACT-01 | Phase 58 | CONTRACT (P1-1) |
| CONTRACT-02 | Phase 58 | CONTRACT (字段命名,discuss 新增) |
| OBSERV-01 | Phase 59 | OBSERV (P1-2) |
| OBSERV-02 | Phase 59 | OBSERV (P1-2 连锁) |
| OBSERV-03 | Phase 59 | OBSERV (P2-b) |
| AUTH-03 | Phase 60 | AUTH (启用决策) |
| SEC-01 | Phase 60 | SEC (P2-c 决策) |
| SEC-02 | Phase 60 | SEC (P3) |
| QUAL-01 | Phase 60 | QUAL (P2-a) |
| AUTH-04 | Phase 61 | AUTH (资源级权限, ex-FUTURE-APIKEY-01) |
| QUAL-03 | Phase 61 | QUAL (限流调优, ex-FUTURE-APIKEY-02) |

---

## Archive: Pre-v1.21 Milestone History

<details>
<summary>✅ Earlier milestone phase history (v1.0–v1.20) preserved for reference</summary>

- ✓ Phases 1-2 — v1.0 工位导入部门/用户关联 (7 plans)
- ✓ Phase 3 — v1.1 信息点导入设备端口关联 (1 plan)
- ✓ Phases 4-7 — v1.2 可配置仪表盘 (11 plans)
- ✓ Phases 8-10 — v1.3 技术债清理 (9 plans)
- ✓ Phase 11 — v1.4 MAC地址采集优化 (4 plans)
- ✓ Phases 12-15 — v1.5 MAC地址历史数据 (26 plans)
- ✓ Phase 16 — v1.6 API密钥管理 (10 plans) ← **v1.21 回归对象**
- ✓ Phase 17 — v1.7 加密配置同步 (6 plans)
- ✓ Phase 18 — v1.8 登录端加密 (4 plans)
- ✓ Phases 19-20 — v1.9 AD域控集成 (11 plans)
- ✓ Phase 21 — v1.10 网络设备权限修复 (1 plan)
- ✓ Phases 22A/22B — v1.12 深信服 VDI (6 plans)
- ✓ Phase 23 — v1.11 AD组自动同步 (18 plans)
- ✓ Phase 26 — v1.13 资产管理 (6 plans)
- ✓ Phase 27 — v1.14 全局列自定义 (1 plan)
- ✓ Phase 28 — v1.15 工位设备关联 (4 plans)
- ✓ Phases 30-34 — 前端性能/P0收尾/P1P2/React best practices/操作日志全模块集成 (~40 plans)
- ✓ Phases 37-39 — 前端部门选择统一/AD账号池统一/部门物理位置映射
- ✓ Phases 40-41 — v1.16 技术债清理 (8 plans)
- ✓ Phases 42-47 — v1.17 资产对账 + 根因修复 (16 plans)
- ✓ Phase 48-49 — v1.18 网络设备硬件清单 + gap closure (5 plans)
- ✓ Phases 50-55 — v1.19 网络设备写命令 (9 plans, 5 build + 1 cleanup) — see [milestones/v1.19-ROADMAP.md](milestones/v1.19-ROADMAP.md)
- ✓ Phase 56 — v1.20 网络设备 VLAN + 端口绑定 (5 plans) — see [milestones/v1.20-ROADMAP.md](milestones/v1.20-ROADMAP.md)

**Total shipped through v1.20**: 156 plans across 49 phases (Phases 1-49 + 50-56).

</details>

### Phase 62: 数据库核心安全加固(跨AI评审修复): internal/core/db 迁移安全+种子凭据+并发保护

**Goal:** [To be planned]
**Requirements**: TBD
**Depends on:** Phase 61
**Plans:** 0 plans

Plans:
- [ ] TBD (run /gsd-plan-phase 62 to break down)

---

*Last updated: 2026-08-13 — Phase 61 planned (2 plans / Wave 1 Plan 01 + Wave 2 Plan 02 sequential; D-01~D-21 全部覆盖、4 层测试策略 [unit+integration+pure-function+migration]、SC#1-3 全部锚定). Phase 60 已完成(2 plans / Wave 1 parallel). Now 5 phases (57-61) / 14 requirements. v1.21 re-planned 2026-08-12: Phase 61 added (资源级权限矩阵 + 限流生产调优, conditional on Phase 60 AUTH-03=启用); FUTURE-APIKEY-01/02 pulled into v1 as AUTH-04/QUAL-03. Core regression fix = Phases 57-60; Phase 61 = post-enable feature completion. Plan-checker revision 2026-08-13: 3 BLOCKERs + 3 WARNINGs 全部修复;Plan 02 改为 Wave 2 依赖 Plan 01,SelectScope 提取为纯函数,既有 rate_limiter_test.go 449 行迁移而非新建,CacheConfigService 12 配置键用单占位符两次查询. v1.20 网络设备 VLAN + 端口绑定 SHIPPED + ARCHIVED 2026-07-10 (Phase 56 / 5 plans).*
