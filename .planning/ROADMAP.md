---
last_updated: 2026-09-07
milestone: v1.31
update_trigger: v1.31 ROADMAP 创建 — 12 类别 29 requirements → 7 phases（102-108）全覆盖映射；Phase 编号从 102 续编（D-05）；输入 = 260907 审计台账 F-06~F-17
---

# Roadmap: XingRan-Next 运维管理系统 — v1.31 milestone

> **v1.30 及更早的 milestone 历史已归档**: `.planning/milestones/`（v1.30-ROADMAP.md / v1.30-REQUIREMENTS.md / v1.30-MILESTONE-AUDIT.md）。
> 本文件自 2026-09-07 起只追踪 **v1.31 V131 技术债清偿 (Tech Debt Retirement)**。

## Current Milestone: v1.31 V131 技术债清偿 (Tech Debt Retirement)

**Goal:** 清偿 2026-09-07 全量技术债务审计台账（`.planning/notes/260907-audit-fix-tech-debt-findings.md`）的全部 12 组未修复项（F-06~F-17）+ 顺带 nilness 观察项，达成：非测试代码 TODO 清零、status/cache-key/分页字面量清零、缓存闭包收敛 base 单一权威、wire 契约统一、skip 测试尽力恢复。

**Source planning data:**

- `.planning/REQUIREMENTS.md`（12 类别 / 29 requirements，Traceability 已回填 phase 映射）
- `.planning/PROJECT.md`（Current Milestone v1.31 段，D-01..D-05 locked decisions）
- `.planning/notes/260907-audit-fix-tech-debt-findings.md`（F-06~F-09 auto 残余 + F-10~F-17 manual-only + 观察项）

**Milestone success criteria:**

- SC-a (覆盖): 29/29 requirements 全部交付——行为变更附回归测试（或决策表落档 / HUMAN-UAT 台账回写）
- SC-b (gate): 七 gate 全程不倒退——go build / go test / 后端 coverage ≥78.33 基线 / 前端 45 dirs / lint / type-check / diff coverage
- SC-c (终态): 非测试代码 TODO 计数 0（grep 守护进 CI/invariants）；status 字面量（12 处）/ 内联 cache key（~47 处）/ 分页双口径 / interface{} 闭包 GetOrSet（扫描面硬失败档）/ 手写 CRUD 五件套（4 文件 19 处）/ `as any`（11 处）/ 无理由 eslint-disable（72 处）全部清零
- SC-d (设计决策): WIRE-01 wire 契约方向、FEMAP-02 漂移归一与 FEMAP-03 颜色 token、TODO-01..05 逐项「实现或删除」决策 在对应 phase 规划时敲定并落盘（D-03 + D-01）

**锁定决策 (v1.31 init):**

- **D-01 范围**: 台账 12 组全做；F-15 逐项决策实现或删除、不留兼容壳
- **D-02 回归纪律**: 行为变更附回归测试；七 gate（go build / go test / 后端 coverage ≥78.33 基线 / 前端 45 dirs / lint / type-check / diff coverage）全程不倒退
- **D-03 设计决策项**: WIRE-01 wire 契约方向、FEMAP-03 颜色 token 选择在 phase 规划时敲定
- **D-04 范围外**: captcha-background 1=启用语义（QUIRK-80-03-D 锁定非 bug，禁改）；观察项中的有意设计（agent 裸 c.JSON / 三层 adapter / 协议默认值）不动；operlog exclude_paths 继续挂账
- **D-05 Phase 编号**: 从 Phase 102 续编（v1.30 用 96-101，v1.29 用 89-95）

## Phases

- [x] **Phase 102: 机械常量化（缓存键 / 状态 / 分页）** — captcha 12 处 + 10 模块 ~35 处内联 cache key、12 处 status 字面量、分页双口径全部收敛具名常量/单一权威，行为等价
- [ ] **Phase 103: 缓存闭包收敛 base 单一权威** — mac_history/heatmap、asset reconciliation、rpa selector 三域 legacy GetOrSet 与手写 cache-aside 迁 base.GetOrSetJSON[T] + invariants 扫描扩口
- [ ] **Phase 104: handler 层架构收敛（wire 契约 + 样板去重）** — 错误响应契约单一权威 + operations 14 handler 收敛（含 server_room 漂移修复）+ monitor 双 handler 去重
- [ ] **Phase 105: 前端 CRUD 收敛 apiFactory** — adDomain/knowledge/duty/workorder 19 处手写五件套迁移 createResourceApi，invariants 基线归零
- [ ] **Phase 106: 前端映射统一与类型卫生** — 选项/Tag 颜色映射收敛 status.ts / 模块共享 constants + `as any` 11 处收窄 + 无理由 eslint-disable 72 处清零
- [ ] **Phase 107: TODO 清零 + nilness 排查** — 22 处非测试 TODO 逐项实现或删除 + 决策表/grep 守护 + NIL-01 根因闭环
- [ ] **Phase 108: skip 测试恢复** — HybridAuthenticator interface 化 + 嵌入式基建恢复 skip + HUMAN-UAT 决策表

### Phase Dependency Graph

```
Phase 102 (CACHE/STATUS/PAGI 机械常量化)
   └─→ Phase 103 (CONV 缓存闭包收敛；selector_learner / mac 域同文件族——102 先注册键、103 再迁闭包)
          └─→ Phase 104 (WIRE 契约先决 + HANDLER 切换；login_log_handler 本相去重)
                 └─→ Phase 107 (TODO 逐项决策；selector_learner/login_log_handler/assets 等
                        文件在 103/104/106 先完成结构收敛，再决策实现或删除)
Phase 105 (FEAPI lib 层 CRUD 收敛；与后端 phases 零文件重叠)
   └─→ Phase 106 (FEMAP+TS pages 层映射/类型；assets 页先收敛，107 再动 TODO-05)
Phase 108 (SKIP 测试恢复；独立，建议最后在稳定代码上恢复测试并量 coverage)
```

**分组理由**: 机械常量化类（CACHE/STATUS/PAGI）依赖最低先行；CONV 独立成相（含 CONV-04 invariants 扩口作相内收口）；WIRE/HANDLER 同相——WIRE-01 先定契约、HANDLER 随后切 handler（契约决策与应用不分家）；前端拆 lib（FEAPI）/pages（FEMAP+TS）两相，避免 11-req 巨相；TODO 量最大（22 处跨前后端）独立成相且刻意靠后（依赖前相文件稳定后再逐项决策）；SKIP-01 refactor 前置、SKIP-02 依赖其模式（相内顺序），放最后在稳定代码上恢复测试。**并行机会**: 105/106（前端）与 102-104（后端）零文件重叠可并行（config `parallelization: false`，默认顺序执行）；108 独立。

---

## Phase Details

**Status**: Completed (2026-09-07) — All 5 plans done (102-01 through 102-05)

**Goal**: 后端残余硬编码清零——captcha 12 处 + notice/settings/duty/workorder/knowledge/network/api_endpoint/mac vendor/widget/rpa selector ~35 处内联 cache key（含失效 pattern）注册具名常量、12 处 status 字面量全部引用 models 具名常量、internal/utils ParsePagination 收敛 pkg/query 单一口径。全部行为等价，无业务语义变化。

**Depends on**: Nothing (v1.31 first phase)

**Requirements**: CACHE-01, CACHE-02, STATUS-01, PAGI-01

**Success Criteria** (what must be TRUE):

  1. 缓存键具名化清零：`internal/core/captcha.go`（storageKey/failKey）+ `captcha_background.go`（list/pool key）12 处与 10 模块 ~35 处内联缓存键及失效 pattern 字符串全部引用具名常量（cache_keys.go 或模块级注册表）；键值与 TTL 逐处等价，缓存读写/失效行为有回归测试锁（CACHE-01 / CACHE-02）
  2. status 字面量清零：workorder/base.go:183、scheduler（job_service/cron/vdi_sync_tasks）、workorder_tasks、reconciliation_tasks、mac_history(_matview)_tasks 共 12 处全部引用 models 具名常量；`status_constants_test.go` AST 锁值全程绿；geocoding 百度 API 白名单豁免外 grep 无残留（STATUS-01）
  3. 分页单口径：`internal/utils/pagination.go` ParsePagination（cap=MaxListPageSize）收敛到 `pkg/query.NormalizePagination`，全部调用方行为不变（逐调用方核对清单落盘，历史分叉差异点注释自证）（PAGI-01）
  4. 回归纪律：常量化为行为等价重构；`go build ./...` + `go test ./...` 0 失败，七 gate 不倒退

**Plans**: 5 plans（3 waves）

Plans:
**Wave 1**

- [x] 102-01-PLAN.md — CACHE-01：captcha 键族常量化（pkg/constants 6 常量 + 16 位点替换 + 等价快照测试）
- [x] 102-04-PLAN.md — STATUS-01：status 位点替换（含新暴露 raw SQL 2 处）+ TestNoStatusLiteralUsage AST 扫描 + WorkOrderStatus 值锁补登记
- [x] 102-05-PLAN.md — PAGI-01：file_handler 迁移 NormalizePaginationWithMax(cap=100) + utils/pagination.go 整文件删除 + 调用方核对清单

**Wave 2** *(blocked on Wave 1 completion)*

- [ ] 102-02-PLAN.md — CACHE-02 注册面：cache_keys.go 8 模块 + pkg/constants 根包 2 格式 + 等价快照（含 D-102-1 落点二分修订披露）

**Wave 3** *(blocked on Wave 2 completion)*

- [ ] 102-03-PLAN.md — CACHE-02 替换面：10 模块 47 调用点替换（含 knowledge :134 条件后缀位）+ TestCacheKeyInlineResidue 内联扫描守护（窄扫 12 文件）

**Notes**: CACHE-02 的 rpa selector（:361）/ mac vendor（:255）键注册与本相后的 Phase 103（CONV-01/03 闭包迁移）同文件族——本相先注册键、103 再迁闭包，顺序不可倒。Wave 结构：Wave 1 = 102-01/102-04/102-05（零文件重叠并行）；Wave 2 = 102-02（依赖 01 的 pkg/constants/cache.go）；Wave 3 = 102-03（依赖 02 注册表）。

---

### Phase 103: 缓存闭包收敛 base 单一权威

**Goal**: v1.29 Phase 92 缓存统一的补遗收口——mac_history/heatmap、asset reconciliation、rpa selector_learner 三域残余 legacy interface{} 闭包 GetOrSet 与手写 cache-aside 全部迁 `base.GetOrSetJSON[T]`；`cache_invariants_92_test.go` 扫描口径扩口至 services 根 / asset / rpa（硬失败档），守护新收敛面不回潮。

**Depends on**: Phase 102（软依赖——CACHE-02 先注册 mac vendor / rpa selector 缓存键，本相再迁同文件闭包，避免同文件冲突）

**Requirements**: CONV-01, CONV-02, CONV-03, CONV-04

**Success Criteria** (what must be TRUE):

  1. mac_history 域收敛：`mac_history_query_service.go` 4 处 legacy `GetOrSet(func() (interface{}, error))`（:307,432,835 及 :259-281 手写 Get/Set cache-aside）+ `heatmap_service.go:118` 全部收敛 `base.GetOrSetJSON[T]` 单 return，services 根不再有 interface{} 闭包式 GetOrSet（CONV-01）
  2. asset reconciliation 收敛：`reconciliation_service.go:799-820` GetByWorkstation 手写读穿透（GetJSON 短路 + Marshal + Set）迁 `base.GetOrSetJSON[T]`，读穿透语义等价（命中短路/回源/回填时序，回归测试锁）（CONV-02）
  3. rpa selector 收敛：`selector_learner.go:169-174,226` GetBestSelector/SaveSelector 手写 JSON cache-aside 迁 `base.GetOrSetJSON[T]`（CONV-03）
  4. invariants 扩口：`cache_invariants_92_test.go` 扫描口径扩展至 services 根 / asset / rpa 包，interface{} 闭包式 GetOrSet 硬失败，扫描全绿（CONV-04）
  5. 回归纪律：`go test ./internal/services/...` 0 失败，七 gate 不倒退

**Plans**: TBD

---

### Phase 104: handler 层架构收敛（wire 契约 + 样板去重）

**Goal**: 错误响应契约单一权威——operations/base_handler.go 本地 helper 与 `pkg/response/handler_helpers.go` 收敛（wire 契约方向 phase 内 discuss 敲定后全仓一致）；operations 14 个同构 CRUD handler 样板收敛 + server_room 漂移修复；monitor oper/login_log 双 handler 五方法去重。

**Depends on**: Nothing 硬依赖（WIRE-01 契约方向在本相内先决、HANDLER 随后切换；建议排在 102/103 后，避免与 services 层改动交叉审查）

**Requirements**: WIRE-01, HANDLER-01, HANDLER-02

**Success Criteria** (what must be TRUE):

  1. wire 契约统一：本地 handleJSONBinding/handleServiceError 与 `pkg/response/handler_helpers.go` 合并为单一权威；契约方向（CodeParamError/CodeServerError vs http.Status*+BusinessError 409）经 discuss 敲定后 operations 14 handler 全量切换、全仓错误响应口径一致；响应格式回归测试覆盖（WIRE-01）
  2. operations handler 收敛：14 个同构 CRUD handler（wall/server_room/floor/door/building/room_device/floor_plan_text/dedicated_line/location_alias/infopoint/workstation/workstation_device/asset/asset_component）样板收敛（泛型 helper 或等价方案）；server_room List:101-103 与 Statistics:37 手写错误路径漂移一并修复（HANDLER-01）
  3. monitor 双 handler 去重：oper_log_handler.go:54-160 与 login_log_handler.go:49-157 五方法复制去重；login 侧手写 `response.Error(apperrors.InternalServerError(err))` 统一走 HandleServiceError（HANDLER-02）
  4. 回归纪律：响应契约行为变更附回归测试；operlog 全覆盖约定不回退（收敛后写端点 `operlog.Record` 调用点逐一核对）；七 gate 不倒退

**Plans**: TBD

**Notes**: D-03 设计决策项（wire 契约方向）在本相 plan-phase 前置 discuss 敲定。本相去重后的 login_log_handler.go 由 Phase 107 再决策 TODO-03 解锁用户项（同文件顺序）。

---

### Phase 105: 前端 CRUD 收敛 apiFactory

**Goal**: `src/lib` 四文件 19 处手写 CRUD 五件套迁移 `createResourceApi` 单一权威——adDomainApi 3 / knowledgeApi 5 / dutyApi 5 / workorderApi 6；D-14 单参 delete 契约与全部 export 签名保持（消费文件零改动）；apiFactory invariants 基线同步归零。

**Depends on**: Nothing（v1.29 Phase 94 `src/lib/apiFactory.ts` 单一权威 + D-12 双档 AST 扫描防线为既定基线）

**Requirements**: FEAPI-01, FEAPI-02, FEAPI-03, FEAPI-04

**Success Criteria** (what must be TRUE):

  1. 四文件迁移归零：adDomainApi.ts:261,295,385 / knowledgeApi.ts:174,179,228,245,250 / dutyApi.ts:193,198,233,275,279 / workorderApi.ts:429,434,532,582,587 手写五件套全部走 createResourceApi（对象 spread + override 唯一扩展惯用法），四文件手写 CRUD 模板清零（FEAPI-01..04）
  2. invariants 基线归零：`apiFactory.invariants.test.ts` 对应文件 warning/hard 档计数与 KEEP 白名单同步更新为 0 残留，新增手写 CRUD 立即红
  3. 契约零破坏：单参 delete 契约（D-14）保持；全部 export 函数签名（含返回类型注解）不变，消费文件 diff 为 0
  4. 回归纪律：type-check / lint / vitest 全绿，前端 45 dirs gate 不倒退

**Plans**: TBD

---

### Phase 106: 前端映射统一与类型卫生

**Goal**: 前端展示映射收敛共享常量——server-rooms/MACHistory 内联选项、fixStatusColor 双份拷贝、11 处内联三元 Tag 统一 `constants/status.ts` / 模块共享 constants（颜色 token phase 规划敲定）；类型卫生——11 处 `as any` 逐处收窄、72 处无理由 eslint-disable 补理由或修复根因移除。

**Depends on**: Phase 105（软依赖——同为前端收敛，lib 层先行减少 pages 层 diff 噪声；零硬文件重叠）

**Requirements**: FEMAP-01, FEMAP-02, FEMAP-03, TS-01, TS-02

**Success Criteria** (what must be TRUE):

  1. 选项/Tag 配置收敛：server-rooms/index.tsx:653-654,417-418 与 MACHistoryPage.tsx:654-655 内联 正常/停用 选项+Tag 全部引用 NORMAL_STOP_OPTIONS / NORMAL_STOP_TAG_CONFIG（FEMAP-01）；fix-suggestion fixStatusColor/fixStatusLabel 双份抽模块共享 constants，orange vs magenta 漂移按 phase 决策归一（FEMAP-02）
  2. 11 处内联三元 Tag 清零：DashboardList:215、assets:387、FloorCardView、menu:110、email-config、api-config、FloorView3D、BuildingView3D、ad-domain/configs、exception-rules 全部统一 NORMAL_STOP_TAG_CONFIG；success/green token 选择 phase 规划敲定，受影响组件渲染有测试锁定（FEMAP-03）
  3. `as any` 清零：11 处逐处收窄——window 注入（HubeiMap:490 / HubeiMapGL:481）declare global 声明合并、customRequest（ExcelImport:261 / FileUpload:259）正确 UploadRequest 签名、login/WidgetRenderer/assets/EditModal 等逐处收窄，`as any` 计数归零（TS-01）
  4. eslint-disable 卫生：72 处无理由 disable（VirtualMachineList/useRoleActions/buildings/info-points/mac/externals.d.ts/helpers/ParamsEditor/CronSelector 等 ~31 文件）逐处补理由注释或修复根因移除，无理由 disable 计数归零（TS-02）
  5. 回归纪律：type-check / lint / vitest 全绿，前端 45 dirs + 覆盖率 gate 不倒退

**Plans**: TBD
**UI hint**: yes

---

### Phase 107: TODO 清零 + nilness 排查

**Goal**: 22 处非测试代码 TODO 逐项决策——实现或删除 + 落档理由（D-01 不留兼容壳）：workorder 评价 / RPA 扩缩容/ListSessions/回滚/selector_learner 通知 / system 解锁用户 / 基础设施 Redis 接入 / 前端 6 处功能占位；决策表落盘 + TODO 计数 grep 守护进 CI/invariants；顺带 rpa/data_mapper.go:332 nilness 根因闭环。

**Depends on**: Phase 103 + Phase 104 + Phase 106（同文件族顺序——selector_learner 先迁缓存（103）、login_log_handler 先去重（104）、assets 页先收敛映射/类型（106），再在结构收敛后的稳定文件上逐项决策 TODO）

**Requirements**: TODO-01, TODO-02, TODO-03, TODO-04, TODO-05, TODO-06, NIL-01

**Success Criteria** (what must be TRUE):

  1. 后端 TODO 决策完成：workorder 评价占位（workorder_router.go:65，TODO-01）、RPA 域 6 处（worker_handler/credential_handler/error_handling/selector_learner/task_service，TODO-02）、system/monitor 域 4 处（config_service/login_log_handler/oper_log/init_data，TODO-03）、基础设施域 4 处（core.go Redis 接入/device_discovery/agent handlers/reconciliation_exception，TODO-04）逐项实现或删除 + 落档理由，无悬置
  2. 前端 TODO 决策完成：LayoutToolbar Widget 选择器/仪表盘设置、useGeocoding 逆解析、WorkstationView 编辑、assets 编辑、AIScriptEditor AI 生成 6 处逐项实现或删除 + 落档（TODO-05）
  3. 决策表 + 守护：每项「实现 / 删除 + 理由」决策表落盘；非测试代码 TODO grep 计数归零，守护进 CI 或 invariants 防回增（TODO-06）
  4. NIL-01 闭环：rpa/data_mapper.go:332 non-nil == nil 根因查明——死代码则删除，真 bug 则修复 + 回归测试
  5. 回归纪律：每项「实现」类决策附回归测试；七 gate 不倒退

**Plans**: TBD
**UI hint**: yes

**Notes**: 本相是 D-01「逐项决策、不留兼容壳」的主要落点——plan-phase 时按域拆 plan，每项决策可追溯（决策表 + commit）。体量最大（22 处 + 决策表 + 守护），预留 escalation 空间。

---

### Phase 108: skip 测试恢复

**Goal**: HybridAuthenticator interface 化 refactor（LocalAuthenticator/ADAuthenticator 具体类型依赖解耦）解锁 hybrid_authenticator_test 5 处 skip；SKIP-02 以 v1.27 嵌入式 LDAP（addomain 先例）+ sqlite :memory: 基建尽力恢复 ad_authenticator 等 10 处 skip；确需真实 LDAP/DB 环境的落 HUMAN-UAT 决策表。

**Depends on**: Nothing 硬依赖（建议放最后——在 102-107 结构收敛后的稳定代码上恢复测试并量 coverage；SKIP-02 依赖 SKIP-01 建立的 interface 注入模式，相内先 01 后 02）

**Requirements**: SKIP-01, SKIP-02

**Success Criteria** (what must be TRUE):

  1. HybridAuthenticator interface 化：具体类型依赖解耦（interface 注入），hybrid_authenticator_test.go 5 处 t.Skip 恢复为真实断言且稳定绿（SKIP-01）
  2. 嵌入式基建恢复：ad_authenticator_test ×5 / authenticator_test / user_sync_service_test 等 10 处 skip 中，能以嵌入式 LDAP + sqlite :memory: 恢复的恢复为真实测试（SKIP-02）
  3. HUMAN-UAT 决策表：确需真实 LDAP/DB 环境的 skip 逐项落 HUMAN-UAT 台账（项 / 不可自动化原因 / owner / 前置条件），无静默遗留
  4. 回归纪律：恢复的测试 `-count=10` 无 flake；后端 coverage ≥78.33 基线不倒退（预期净增），七 gate 全绿

**Plans**: TBD

---

## Progress

| Phase | Status | Plans | Requirements | Started | Completed |
|-------|--------|-------|--------------|---------|-----------|
| Phase 102 机械常量化（缓存键/状态/分页） | Completed (5 plans) | 5/5 | CACHE-01..02 + STATUS-01 + PAGI-01 | 2026-09-07 | 2026-09-07 |
| Phase 103 缓存闭包收敛 base 单一权威 | Not started | 0/TBD | CONV-01..04 | - | - |
| Phase 104 handler 层架构收敛（wire+handler） | Not started | 0/TBD | WIRE-01 + HANDLER-01..02 | - | - |
| Phase 105 前端 CRUD 收敛 apiFactory | Not started | 0/TBD | FEAPI-01..04 | - | - |
| Phase 106 前端映射统一与类型卫生 | Not started | 0/TBD | FEMAP-01..03 + TS-01..02 | - | - |
| Phase 107 TODO 清零 + nilness 排查 | Not started | 0/TBD | TODO-01..06 + NIL-01 | - | - |
| Phase 108 skip 测试恢复 | Not started | 0/TBD | SKIP-01..02 | - | - |

**Total:** 7 phases / 29 requirements（12 类别全覆盖；traceability 见 `.planning/REQUIREMENTS.md`）

---

## Out of Scope (locked from v1.31 init)

- **captcha-background 1=启用语义** — QUIRK-80-03-D 就地锁定（gorm default:1），后端前端一致，非 bug；禁止套用 0=启用共享常量
- **超时字面量模块私有命名常量**（~120 处）— 约定允许的模块局部配置，不强制入 pkg/constants
- **新业务功能** — 本期为纯技术债清偿
- **前端覆盖率新目标** — v1.28 已收口 45.13%，gate 维持不倒退即可
- **operlog exclude_paths / LDAP InsecureSkipVerify / agent 裸 c.JSON / 三层 adapter 物理合并 / 协议默认值常量化** — Future Requirements（见 REQUIREMENTS.md），继续挂账

---

*Last updated: 2026-09-07 — v1.31 ROADMAP 创建（29 requirements → 7 phases 102-108 全覆盖映射；设计决策项 WIRE-01 → Phase 104、FEMAP token/漂移归一 → Phase 106、TODO 逐项决策 → Phase 107；分组理由：机械常量化先行 → 缓存闭包收敛 → handler 架构收敛 → 前端 lib/pages 两相 → TODO 跨域清账（依赖前相文件稳定）→ SKIP 收尾）。上一 milestone: v1.30 V130 缺陷治理 SHIPPED + ARCHIVED 2026-09-07（6 phases / 21 plans / 22 requirements，见 milestones/v1.30-ROADMAP.md）。*
