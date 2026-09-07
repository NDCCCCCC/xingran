---
milestone: v1.31
status: defined
---

# Requirements: XingRan-Next — Milestone v1.31 V131 技术债清偿 (Tech Debt Retirement)

**Defined:** 2026-09-07
**Core Value:** 清偿 2026-09-07 全量技术债务审计台账的全部 12 组未修复项（F-06~F-17）+ 顺带 nilness 观察项，达成：非测试代码 TODO 清零、status/cache-key/分页/协议字面量清零、缓存闭包收敛 base 单一权威、wire 契约统一、skip 测试尽力恢复。

**输入来源:**

- `.planning/notes/260907-audit-fix-tech-debt-findings.md`（F-06~F-09 not-attempted + F-10~F-17 manual-only + 观察项）
- `.planning/PROJECT.md` v1.31 段（D-01~D-05 锁定决策）

**锁定决策 (v1.31 init):**

- **D-01 范围**: 台账 12 组全做；F-15 逐项决策实现或删除、不留兼容壳
- **D-02 回归纪律**: 行为变更附回归测试；七 gate（go build / go test / 后端 coverage ≥78.33 基线 / 前端 45 dirs / lint / type-check / diff coverage）全程不倒退
- **D-03 设计决策项**: WIRE-01 wire 契约方向、FEMAP-03 颜色 token 选择在 phase 规划时敲定
- **D-04 范围外**: captcha-background 1=启用语义（QUIRK-80-03-D 锁定非 bug，禁改）；观察项中的有意设计（agent 裸 c.JSON / 三层 adapter / 协议默认值）不动；operlog exclude_paths 继续挂账
- **D-05 Phase 编号**: 从 Phase 102 续编（v1.30 用 96-101，v1.29 用 89-95）

## v1.31 Requirements

### CACHE — 缓存键残余常量化

- [ ] **CACHE-01**: `internal/core/captcha.go`（:297,326,361,367,419,426 storageKey / :503,529 failKey）与 `internal/core/captcha_background.go`（:146,243,295,310 list/pool key）共 12 处内联缓存键收敛为包内具名常量（或注册 cache_keys.go），行为等价
- [ ] **CACHE-02**: notice（:208,210）/ settings（:53,64）/ duty（:144,215,273,287,301）/ workorder（:215,264,271）/ knowledge（:129-134,223,230,244）/ network（:298,319）/ api_endpoint（:63,194）/ mac vendor（:255）/ widget（:59-65）/ rpa selector（:361）等模块 ~35 处内联 cache key 注册进 cache_keys.go 或模块级注册表并引用；失效 pattern 字符串同步具名化

### STATUS — 状态字面量清零

- [x] **STATUS-01**: 剩余 12 处 status 字面量全部引用 models 具名常量：workorder/base.go:183（`[]int{0,1}`）、scheduler/job_service.go:331、scheduler/cron.go:43,62,235,407,435,832、scheduler/vdi_sync_tasks.go:48,85、workorder_tasks.go:195,328,451、reconciliation_tasks.go:196、mac_history_tasks.go:127、mac_history_matview_tasks.go:56（按实际常量存在性逐处核对；geocoding 百度 API 白名单豁免）

### PAGI — 分页口径归一

- [ ] **PAGI-01**: `internal/utils/pagination.go` ParsePagination（cap=MaxListPageSize）收敛到 `pkg/query.NormalizePagination` 单一口径，调用方行为不变（差异点注释自证历史分叉，收敛时逐调用方核对）

### CONV — 缓存闭包收敛 base 单一权威

- [x] **CONV-01**: mac_history_query_service.go 4 处 legacy `GetOrSet(func() (interface{}, error))`（:307,432,835）+ :259-281 手写 Get/Set cache-aside + heatmap_service.go:118 迁移 `base.GetOrSetJSON[T]`
- [x] **CONV-02**: asset/reconciliation_service.go:799-820 GetByWorkstation 手写读穿透（GetJSON 短路 + Marshal + Set）迁移 `base.GetOrSetJSON[T]`
- [x] **CONV-03**: rpa/selector_learner.go:169-174,226 GetBestSelector/SaveSelector 手写 JSON cache-aside 迁移 `base.GetOrSetJSON[T]`
- [x] **CONV-04**: `cache_invariants_92_test.go` 扫描口径扩展至 services 根 / asset / rpa 包（interface{} 闭包式 GetOrSet 硬失败），守护新收敛面不回潮

### WIRE — 错误响应契约统一

- [ ] **WIRE-01**: operations/base_handler.go:10-25 本地 handleJSONBinding/handleServiceError 与 `pkg/response/handler_helpers.go` 合并为单一权威，operations 14 handler 全量切换；wire 契约方向（CodeParamError/CodeServerError vs http.Status*+BusinessError 409）在 phase 规划敲定后全仓一致；响应格式回归测试覆盖

### HANDLER — Handler 样板收敛

- [ ] **HANDLER-01**: operations 14 个同构 CRUD handler（wall/server_room/floor/door/building/room_device/floor_plan_text/dedicated_line/location_alias/infopoint/workstation/workstation_device/asset/asset_component）样板收敛（泛型 helper 或等价方案）；server_room List:101-103 与 Statistics:37 的手写错误路径漂移一并修复
- [ ] **HANDLER-02**: monitor oper_log_handler.go:54-160 与 login_log_handler.go:49-157 五方法复制去重；login 侧手写 `response.Error(apperrors.InternalServerError(err))` 统一走 HandleServiceError

### FEAPI — 前端 CRUD 收敛 apiFactory

- [ ] **FEAPI-01**: adDomainApi.ts:261,295,385 三处手写 update/delete 五件套迁移 createResourceApi（D-14 单参 delete 契约保持，invariants 基线同步归零）
- [ ] **FEAPI-02**: knowledgeApi.ts:174,179,228,245,250 五处迁移（同上约束）
- [ ] **FEAPI-03**: dutyApi.ts:193,198,233,275,279 五处迁移（同上约束）
- [ ] **FEAPI-04**: workorderApi.ts:429,434,532,582,587 六处迁移（同上约束）

### FEMAP — 前端选项/颜色映射统一

- [ ] **FEMAP-01**: server-rooms/index.tsx:653-654,417-418 与 MACHistoryPage.tsx:654-655 内联 正常/停用 选项+Tag → `constants/status.ts` NORMAL_STOP_OPTIONS / NORMAL_STOP_TAG_CONFIG
- [ ] **FEMAP-02**: fix-suggestion fixStatusColor/fixStatusLabel 双份拷贝（index.tsx:68-75 vs FixSuggestionDetailDrawer.tsx:22-29）抽模块共享 constants，漂移（orange vs magenta）按 phase 决策归一
- [ ] **FEMAP-03**: 11 处内联 `status === 0 ? "success" : "error"` 三元 Tag（DashboardList:215、assets:387、FloorCardView:67-68、menu:110、email-config:268-269、api-config:252-253、FloorView3D:90-91、BuildingView3D:183-184、ad-domain/configs:253,474、exception-rules:285）统一 NORMAL_STOP_TAG_CONFIG；success/green token 选择 phase 规划敲定

### TODO — 非测试代码 TODO 清零

- [ ] **TODO-01**: workorder 域：评价功能占位（workorder_router.go:65）——实现或删除+落档理由
- [ ] **TODO-02**: RPA 域：扩缩容配置读取/持久化（worker_handler.go:260,279）、ListSessions（credential_handler.go:154）、回滚逻辑（error_handling.go:311）、selector_learner 使用情况提取/通知机制（:235,367）、task_service 部门 ID（:296）——逐项实现或删除+落档理由
- [ ] **TODO-03**: system/monitor 域：config 缓存刷新（config_service.go:257）、解锁用户逻辑（login_log_handler.go:194）、oper_log follow-up（:213）、init_data 未用函数（:638）——逐项实现或删除+落档理由
- [ ] **TODO-04**: 基础设施域：core.go:911 Redis 接入、device_discovery_service.go:662、agent/server/handlers.go:304、reconciliation_exception.go:578 R3+ 缓存失效——逐项实现或删除+落档理由
- [ ] **TODO-05**: 前端 6 处：LayoutToolbar Widget 选择器/仪表盘设置（:132,139）、useGeocoding 逆解析（:131）、WorkstationView 编辑（:73）、assets 编辑（:582）、AIScriptEditor AI 生成（:97）——逐项实现或删除+落档理由
- [ ] **TODO-06**: 决策表落盘（每项：实现 / 删除 + 理由），非测试代码 TODO 计数归零（grep 守护进 CI 或 invariants）

### SKIP — Skip 测试恢复

- [ ] **SKIP-01**: HybridAuthenticator interface 化 refactor（具体类型 LocalAuthenticator/ADAuthenticator 依赖解耦），恢复 hybrid_authenticator_test.go 5 处 t.Skip
- [ ] **SKIP-02**: ad_authenticator_test ×5 / authenticator_test / user_sync_service_test 等 10 处 skip：能以嵌入式基建（v1.27 addomain 嵌入式 LDAP 先例 + sqlite :memory:）恢复的恢复；确需真实 LDAP/DB 环境的落 HUMAN-UAT 决策表

### TS — 前端类型卫生

- [ ] **TS-01**: 11 处 `as any` 类型收窄：window 注入（HubeiMap:490 / HubeiMapGL:481）用 declare global 声明合并；customRequest（ExcelImport:261 / FileUpload:259）用正确 UploadRequest 签名；login:28 / WidgetRenderer:41 / assets:255 / EditModal:182 等逐处收窄
- [ ] **TS-02**: 72 处无理由 eslint-disable（VirtualMachineList ×5、useRoleActions ×3、buildings/info-points/mac ×6、externals.d.ts/helpers/ParamsEditor/CronSelector ×8 及 ~31 文件散布）补理由注释或修复根因移除 disable

### NIL — 观察项排查

- [ ] **NIL-01**: rpa/data_mapper.go:332 nilness（impossible condition: non-nil == nil）根因排查：死代码则删除，真 bug 则修复 + 回归测试

## Future Requirements (deferred)

- **operlog exclude_paths** 细化治理 — 继续挂账（v1.29 D-05 起）
- **LDAP InsecureSkipVerify** 生产证书配置 — 安全事项独立立项（CLAUDE.md 已知）
- **agent/server 裸 c.JSON** 响应包装统一 — 疑为 agent 协议有意设计，需 agent 协议演化时一并处理
- **三层 adapter 彻底合并**（cache_adapter / adapter / data_cache_service）— 架构定性已完成（base 单一权威），物理合并等退役窗口
- **协议默认值回退常量化**（agent/config.go:25、cmd/agent/main.go:29、rpa-worker config.go:211、baidu API URL、cors TrimPrefix）— low 值，随域 phase 顺带

## Out of Scope

- **captcha-background 1=启用语义** — QUIRK-80-03-D 就地锁定（gorm default:1），后端前端一致，非 bug；禁止套用 0=启用共享常量
- **超时字面量模块私有命名常量**（~120 处）— 约定允许的模块局部配置，不强制入 pkg/constants
- **新业务功能** — 本期为纯技术债清偿
- **前端覆盖率新目标** — v1.28 已收口 45.13%，gate 维持不倒退即可

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| CACHE-01 | Phase 102 | Pending |
| CACHE-02 | Phase 102 | Pending |
| STATUS-01 | Phase 102 | Complete |
| PAGI-01 | Phase 102 | Pending |
| CONV-01 | Phase 103 | Complete |
| CONV-02 | Phase 103 | Complete |
| CONV-03 | Phase 103 | Complete |
| CONV-04 | Phase 103 | Complete |
| WIRE-01 | Phase 104 | Pending |
| HANDLER-01 | Phase 104 | Pending |
| HANDLER-02 | Phase 104 | Pending |
| FEAPI-01 | Phase 105 | Pending |
| FEAPI-02 | Phase 105 | Pending |
| FEAPI-03 | Phase 105 | Pending |
| FEAPI-04 | Phase 105 | Pending |
| FEMAP-01 | Phase 106 | Pending |
| FEMAP-02 | Phase 106 | Pending |
| FEMAP-03 | Phase 106 | Pending |
| TS-01 | Phase 106 | Pending |
| TS-02 | Phase 106 | Pending |
| TODO-01 | Phase 107 | Pending |
| TODO-02 | Phase 107 | Pending |
| TODO-03 | Phase 107 | Pending |
| TODO-04 | Phase 107 | Pending |
| TODO-05 | Phase 107 | Pending |
| TODO-06 | Phase 107 | Pending |
| NIL-01 | Phase 107 | Pending |
| SKIP-01 | Phase 108 | Pending |
| SKIP-02 | Phase 108 | Pending |

---
*Requirements defined: 2026-09-07 — Traceability 回填 2026-09-07（29/29 requirements → Phases 102-108，见 ROADMAP.md）*
