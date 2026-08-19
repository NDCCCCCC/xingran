---
gsd_state_version: 1.0
milestone: v1.22 + v1.23 启动块
milestone_name: milestone
status: executing
stopped_at: Completed 69-03-PLAN.md (batch 2, commit ac33b2a)
last_updated: "2026-08-19T07:06:33.904Z"
last_activity: 2026-08-19
progress:
  total_phases: 6
  completed_phases: 4
  total_plans: 19
  completed_plans: 8
  percent: 42
---

# Project State

**Project**: XingRan-Next 运维管理系统
**Created**: 2026-04-16
**Status**: v1.22 前端品牌化改造 SHIPPED 2026-08-18（Phases 64-67 / 4 phases / 4 plans / 15 requirements 100% 勾选 / 33 tasks）。Phase 63 前端工具链自动化独立 IN PROGRESS。**v1.23 启动块**: Phase 68 部署稳健性 & 文档一致性(SM2 密钥配置闭环)EXECUTED 2026-08-19(5 commits: a21dcec..25ded8f)。
**Last activity**: 2026-08-19 — Phase 68-01 EXECUTED（5 commits: a21dcec / a764825 / 65093b9 / 52685fd / 25ded8f）。来源：归档会话 `.planning/debug/resolved/public-key-500-after-subpath-fix.md`（DEPLOY-01~05 全量修复：env 变量名文档一致 / setup-server.sh secrets.env 模板补 SM2 keys 段 / gen_sm2_keys header 注释路径修正 / getPublicKey handler 500 时打印 useSM2 状态 / sqlite 模板 use_sm2 默认值 + 迁移指引）。Phase 63 63-01-PLAN.md 在 `.planning/phases/63-frontend-toolchain-automation/` IN PROGRESS(2026-08-14 起)。此前 2026-08-17 quick-260817-hfl: 后端 Supabase → 本地 SQLite + sqlite 兼容收尾(PG-only 守卫/缺表注册/方言修复/admin 全量菜单种子 migration_207);同日 quick-260817-ucz: 第 6 套主题「墨绿琥珀」(ink-amber)延伸到控制台。2026-08-15 Phase 62 全部 5/5 plan 完成;2026-08-14 Phase 57-61 全部完成 + Phase 62 plans created from cross-AI reviews。

# Project State

**Project**: XingRan-Next 运维管理系统
**Created**: 2026-04-16
**Status**: v1.22 前端品牌化改造 SHIPPED 2026-08-18（Phases 64-67 / 4 phases / 4 plans / 15 requirements 100% 勾选 / 33 tasks）。Phase 63 前端工具链自动化独立 IN PROGRESS。
**Last activity**: 2026-08-18 — v1.22 ROADMAP drafted(Phases 64-67),15 requirements mapped 100% to TOKEN(64) / THEME(65) / COMP+QA-02(66) / QA-03+QA-04(67)。Phase 63 63-01-PLAN.md 在 `.planning/phases/63-frontend-toolchain-automation/` IN PROGRESS(2026-08-14 起)。此前 2026-08-17 quick-260817-hfl: 后端 Supabase → 本地 SQLite + sqlite 兼容收尾(PG-only 守卫/缺表注册/方言修复/admin 全量菜单种子 migration_207);同日 quick-260817-ucz: 第 6 套主题「墨绿琥珀」(ink-amber)延伸到控制台。2026-08-15 Phase 62 全部 5/5 plan 完成;2026-08-14 Phase 57-61 全部完成 + Phase 62 plans created from cross-AI reviews。

## Project Reference

See: [.planning/PROJECT.md](PROJECT.md) (updated 2026-08-18)

**Core value**: 端到端运维可观测与审计能力——每个写操作产生可追溯记录(who/when/what/from-where/before-after-state),敏感字段自动脱敏。v1.22 在不破坏既有核心价值的前提下,把后台内部视觉统一到登录页品牌(深绿 × 铜金 × 奶油纸感),让 53 屏业务页面在 design-system 层自动继承品牌样式。

**Current focus**: v1.22 milestone INITIATED + ROADMAP drafted,4 phases(64-67),等待 plan-phase 64 启动。

## Current Position

Phase: 69 (字典与状态值治理) — EXECUTING (并行 Phase 70 会话执行中)
Plan: 5 of 8
Status: Ready to execute
Last activity: 2026-08-19

## Accumulated Context

### Roadmap Evolution

- **v1.22 ROADMAP drafted (2026-08-18)**: 4 phases (64-67), 15 requirements, 100% coverage
  - Phase 64: 品牌令牌层落地 + 对比度验证 (TOKEN-01/02/03/04 + QA-01)
  - Phase 65: 主题系统收敛 (THEME-01/02/03)
  - Phase 66: 通用组件样式 + 硬编码色扫描 (COMP-01/02/03/04 + QA-02)
  - Phase 67: 构建回归 + 视觉确认 (QA-03 + QA-04)
- **v1.23 启动块 ROADMAP drafted (2026-08-19)**: Phase 68 — 部署稳健性 & 文档一致性(SM2 密钥配置闭环)
  - DEPLOY-01: env 变量名文档-代码一致（XINGRAN_JWT_SM2_* → JWT_SM2_*，全仓17 处）
  - DEPLOY-02: setup-server.sh secrets.env 模板补 SM2 keys 段
  - DEPLOY-03: gen_sm2_keys header 注释路径修正
  - DEPLOY-04: getPublicKey handler 500 时打印 useSM2/sm2PublicKey 状态
  - DEPLOY-05: config.sqlite.example.yaml use_sm2 默认值对齐 + 迁移指引
  - 来源: `.planning/debug/resolved/public-key-500-after-subpath-fix.md` §Specialist Review + §Related bugs
- **v1.24 启动块 ROADMAP drafted (2026-08-19)**: Phase 69 — 字典与状态值治理(状态语义单一真相源)
  - DICT-01: 后端集中状态常量定义,消灭 50+ 文件裸 0/1 字面量
  - DICT-02: 真枚举(type/category)盘点 + sys_dict seed
  - DICT-03: 前端 constants.tsx 硬编码 options → useDict 分批迁移
  - DICT-04: CLAUDE.md Status Value Convention 改指向常量真相源
  - 来源: 2026-08-19 字典消费率审计 — 后端 GetDictDataByTypeKey 仅 dict_cache_impl 内部自用(0 业务 service)、前端 4/78 页用 useDict、sys_dict seed 仅 network_device_type 一类
- **v1.25 启动块 ROADMAP drafted (2026-08-19)**: Phase 70 — 系统设置页面布局重构(对齐 v1.22 品牌设计理念)
  - 按 v1.22 品牌理念(深绿 × 铜金 × 奶油纸感、双层纸感卡片、按钮纪律)重新设计 `src/pages/system/settings-page/` + `src/pages/settings/` 页面布局
  - 清理多主题时代布局残留(含 default-theme 入口删除后的遗留文件)
  - 纯前端布局重构,不改业务逻辑与 API;依赖 Phase 67 已 SHIPPED 的品牌基线,与 Phase 69 无硬依赖
- **Phase 63 IN PROGRESS (2026-08-14)**: 前端工具链自动化 — `.planning/phases/63-frontend-toolchain-automation/63-01-PLAN.md` 独立进行,本 milestone 不占用 63
- **v1.21 SHIPPED + ARCHIVED 2026-08-18**: Phases 57-62 / 14 v1 requirements + 14 跨 AI review items
  - Phase 62 added 2026-08-12: 数据库核心安全加固(跨 AI 评审修复) — 来源 `.planning/reviews/260814-internal-core-db-REVIEWS.md`(codex + opencode 交叉评审,共识 C1-C7 + 单方 HIGH/MEDIUM)
  - v1.21 ROADMAP archived to `.planning/milestones/v1.21-ROADMAP.md` + `.planning/milestones/v1.21-REQUIREMENTS.md`

### v1.22 Milestone — Critical Decisions (locked at init)

- **Scope**: 仅 design-system 层(D-05 锁定)——落地 brand-spec.md 品牌令牌到 `xingran-react-frontend/src/design-system/` 与 `src/index.css` 253 变量层,53 屏业务页面自动继承样式,不逐屏改造
- **Theme strategy**: 全局替换,不保留多主题(D-01 锁定,不可逆)——移除 6 套主题(minimal/glassmorphism/neumorphism/flat2.0/luxury-quiet/ink-amber)与 ThemeSwitcher / ColorSwitcher;保留 light / dark 双模式(单一品牌色相);保留 layoutStore 的布局与密度切换(D-03)
- **Color source**: `brand-spec.md` 为唯一色值真相源(D-02)——标注「实测」直接采用,「推导」微调须重跑对比度验证
- **Button discipline**: 主按钮一律 `--primary` `#156031` 绿底白字(7.64:1);hover `#2E7444`(5.68:1);铜金 `#C09058` 不做实心主按钮(2.85:1 不达标);必须铜金实心时用 `#B88850` + ≥16px 半粗体白字(3.15:1 大字达标),hover 只许加深至 `#AA7B42`(D-03)
- **Phase numbering**: 从 Phase 63 续编(64+);Phase 63 独立 IN PROGRESS,不占用(D-04)
- **Research skipped**: 视觉层重构,brand-spec.md 与 admin-design-plan.md 现状侦察已落盘,代码与改造范围均已调查清楚(`.planning/research/` 不需要新建)
- **Granularity**: standard(项目配置),4 phases (64-67) 为本 milestone 自然交付边界 —— 每个 phase 保持应用在边界处可构建(buildable at phase boundary),避免大爆炸式重构

### v1.22 — Phase Dependency Graph

```
Phase 64 (品牌令牌 + 对比度验证)
   │
   └─→ Phase 65 (主题系统收敛) [depends on 64]
          │
          └─→ Phase 66 (通用组件样式 + 硬编码扫描) [depends on 65]
                 │
                 └─→ Phase 67 (构建回归 + 视觉确认) [depends on 66]
```

Phase 63 (frontend-toolchain-automation) 独立 IN PROGRESS,提供 CI / lint / 测试基建,v1.22 Phase 67 的回归门将直接受益于其 CI gate 与 lint-staged hook。

### v1.21 Milestone — Critical Decisions (locked at init, for archive)

- **Scope**: 全修复 + 就绪 + 能力补全 — 修复全部 P0/P1/P2 确定性缺陷,MultiAuth 代码修好并已挂载;Phase 61 落地 AUTH-04 资源级权限矩阵与 QUAL-03 限流生产调优
- **Regression nature**: 对 v1.6「API 密钥管理系统」(Phase 16 / 2026-05-19 / 10 plans) 的回归修复,非新功能
- **Research skipped**: 回归修复场景,代码与问题均已调查清楚(`.planning/research/` 不存在)
- **Phase numbering**: 从 v1.20 末尾 Phase 56 续编(57-61,后增 62 为跨 AI 评审修复)
- **Granularity**: standard(项目配置),5 phases (57-61) 为本 milestone 自然交付边界;Phase 61 为 2026-08-12 重规划新增(能力补全,conditional)
- **Scope evolution (2026-08-12 re-plan)**: 原"全修复 + 就绪"扩展为"全修复 + 就绪 + 能力补全"——FUTURE-APIKEY-01/02 升级为 v1 AUTH-04/QUAL-03 归 Phase 61(资源级权限矩阵 + 限流生产调优),仅在 Phase 60 AUTH-03=启用 后执行

### v1.21 — 根因调查结论(ground-truth 已验证,for archive)

| 缺陷 | 文件:行 | 根因 |
|------|---------|------|
| P0-2 | `internal/middleware/apikey.go:146-179` | `setUserContextForAPIKey(apiKey interface{})` 接收 `*models.APIKey` 指针,但断言为局部值类型 `apiKeyType`,恒 false → context 从未写入 |
| P0-1 | `internal/api/router.go` | `MultiAuth` 及下游 `RequireScope` / `RequireAPIKeyResourcePermission` / `RateLimitByScope` 未挂载任何生产路由(死代码) |
| P1-1 | `internal/api/v1/system/apikey_router.go` vs `src/api/apikey.ts` | 前端 GET/PUT/DELETE 与后端 POST 路由方法不匹配 → 单条 get/update/delete 404 |
| P1-2 | `internal/middleware/apikey.go:60-75` | 使用日志 goroutine 在 `c.Next()` 前触发,LogUsageRequest 仅填 Method/Path/ClientIP,StatusCode/Duration/Success 全零 → successRate ≈ 0% |
| P2-a | `internal/middleware/apikey.go:274-275` | `string(rune(result.Limit))` 编码错误:Limit=100 → "d" 而非 "100" |
| P2-b | `internal/middleware/apikey.go:66` | 异步 goroutine 复用 `c.Request.Context()`,请求结束 → ctx.Canceled,error 在 `usage_logger.go:73` 被 `_ = err` 吞掉 |
| P2-c | `internal/services/system/apikey_service.go` + migration_085 | API Key `Key` 字段明文存储(`WHERE key = ?` 直查) |
| P3 | migration_085 | `idx_api_keys_key` 与 `uniqueIndex` 重复 |

### Pending Decisions (defer to phase-internal discuss)

None for v1.22. 所有 v1.22 决策(D-01..D-05)已在 init 时锁定,Phase 内不再做策略性讨论;Plan 内部仅做工程实现细节(对比度校验脚本写法 / lint 规则选择 / 截图对比画布工具等)讨论。

### Blockers/Concerns

None currently. v1.22 是纯前端 design-system 层重构,无后端 API / 数据模型 / 权限风险;Phase 65 多主题移除涉及 13 个消费方但每个消费方仅引用清除,可机械重构;Phase 66 硬编码扫描脚本需在 .css / .tsx 中准确识别"非品牌裸 hex"(误报防护需明确白名单如品牌 token 定义本身 / svg fill="none" / rgba 透明色等)。

## Quick Tasks Completed

| Quick ID | Description | Date | Commit | Plan |
|----------|-------------|------|--------|------|
| 260812-wu5 | clean constants dead code and unify pagination constants | 2026-08-12 | 759a65a | [260812-wu5-clean-constants-dead-code-and-unify-pagi](./quick/260812-wu5-clean-constants-dead-code-and-unify-pagi/) |
| 260814-164 | 修复 RPA Worker 注册主键 NULL(23502) + 菜单接口 N+1(context canceled 500) | 2026-08-14 | f0d0a1f / 4c2a900 | [260814-164-fix-rpa-pk-menu-n1](./quick/260814-164-fix-rpa-pk-menu-n1/) |
| 260814-211 | 修复 workstation/list uuid=text 类型错误(42883) + dept.leader 非 uuid 查询防御(22P02) | 2026-08-14 | c9ab875 / 08d97ed | [260814-211-fix-workstation-list-uuid-text-cast-dept](./quick/260814-211-fix-workstation-list-uuid-text-cast-dept/) |
| 260814-ehg | 旧库菜单去重导入 dev 库(保持层级) + admin 全量授权(239 菜单/10 顶级目录) | 2026-08-14 | ef1ba87 / cb81443 / e02d837 | [260814-ehg-dedupe-and-import-legacy-menu-data-into-](./quick/260814-ehg-dedupe-and-import-legacy-menu-data-into-/) |
| 260814-gor | assignAllMenusToAdmin 先删后插→增量幂等差集补全(消除丢权限炸弹+降载) | 2026-08-14 | a0ea57b | [260814-gor-fix-assignallmenustoadmin-delete-then-re](./quick/260814-gor-fix-assignallmenustoadmin-delete-then-re/) |
| 260814-h0e | my-menus 系列接缓存(menuCacheService 补 GetOrSet 覆盖,消除慢库每次打DB) | 2026-08-14 | ba36b1b / f481c93 | [260814-h0e-cache-my-menus-series-via-menucacheservi](./quick/260814-h0e-cache-my-menus-series-via-menucacheservi/) |
| 260814-wxb | 修复两个存量前端测试失败(ACTION_TITLE 7 key + HealthCard 紧凑版断言,零产品代码改动) | 2026-08-14 | 19ac4f6 / bbc5248 | [260814-wxb-port-write-constants-action-title-health](./quick/260814-wxb-port-write-constants-action-title-health/) |
| 260817-hfl | 后端数据库 Supabase(远程PG) → 本地 SQLite(纯 Go glebarez/modernc 驱动,无 CGO);含 sqlite 兼容收尾:PG-only 守卫(分区/statement_timeout/MV刷新)、缺表注册(user_preference/rpa/oui/oper_log/logininfor/reconciliation)、PG 方言修复(INTERVAL/NOW/::text)、migration_207 规范菜单目录种子(admin 全量菜单修复) | 2026-08-17 | 7c422a1~b793304 + 调试修复待提交 | [260817-hfl-supabase-postgresql-sqlite-go-modernc-or](./quick/260817-hfl-supabase-postgresql-sqlite-go-modernc-or/) |
| 260817-ucz | 第 6 套主题「墨绿琥珀」(ink-amber): 登录页 v4 配色(墨绿+琥珀金+米白)延伸到控制台,themes/ink-amber 三件套 + 前端 7 处/后端 2 处白名单注册;纯增量,现有 5 套主题与登录页零改动;自动化门全绿,人工视觉验证 checkpoint pending | 2026-08-17 | de46ceb / 1e0253b | [260817-ucz-login-theme-palette-themestore-design-to](./quick/260817-ucz-login-theme-palette-themestore-design-to/) |

## Deferred Items

Items acknowledged and carried forward from previous milestone closes (non-v1.22 work):

| Category | Item | Status | Source |
|----------|------|--------|--------|
| uat_gap | v1.20 VLAN + port_binding 12 项真机 UAT (Huawei S8700 × 6 + Ruijie RS8607E × 4 + H3C × 2 conditional) | deferred (site-visit) | v1.20 close 2026-07-10 |
| requirement | VLAN-04 / BIND-06 / UI-06 批量端口写 | deferred to FUTURE-BATCH-05 | v1.20 close |
| uat_gap | v1.19 7 项真机 SSH transport verification | deferred (site-visit) | v1.19 close 2026-07-08 |
| uat_gap | v1.18 3 项 site-visit UAT (S8700/RS8607E) | deferred (site-visit) | v1.18 close 2026-07-04 |
| tech-debt | WR-01 / WR-03 / WR-04 / WR-05 + 14 项 v1.19.x+ future work | backlog | v1.19 REQUIREMENTS §Future |
| tech-debt | ~88 audit-open historical items (19 debug_sessions / 60 quick_tasks / etc.) | backlog | pre-v1.20 |
| v1.22 future | PROTO-01/02/03/04 逐屏对齐(路由前缀 / 字段表头工具栏 / 菜单组结构 / 空状态文案) | v1.23+ candidate | v1.22 REQUIREMENTS §Future (来源 PROTOTYPE-VS-ACTUAL.md) |
| v1.22 future | VIS-01/02/03 视觉深化(3D 楼宇配色 / 登录后台过渡动效 / 打印导出样式) | v1.23+ candidate | v1.22 REQUIREMENTS §Future |
| v1.22 future | ink-amber 主题 removal hard-deletion 检查(Phase 65 落地) | scope of Phase 65 | v1.22 (quick-260817-ucz 引入) |

Full deferred detail in [milestones/v1.21-ROADMAP.md](milestones/v1.21-ROADMAP.md) + [milestones/v1.20-ROADMAP.md](milestones/v1.20-ROADMAP.md) + [milestones/v1.19-ROADMAP.md](milestones/v1.19-ROADMAP.md).

## Completed Milestones

- ✅ v1.0 工位导入部门/用户关联 (2026-04-16) — 2 phases, 7 plans
- ✅ v1.1 信息点导入设备端口关联 (2026-04-16) — 1 phase, 1 plan
- ✅ v1.2 可配置仪表盘生产级改造 (2026-04-21) — 4 phases, 11 plans
- ✅ v1.3 技术债清理 (2026-04-27) — 3 phases, 9 plans
- ✅ v1.4 MAC地址采集优化 (2026-05-09) — 1 phase, 4 plans
- ✅ v1.5 MAC地址历史数据管理 (2026-06-15) — 4 phases, 26 plans
- ✅ **v1.6 API密钥管理系统 (2026-05-19) — 1 phase, 10 plans**
- ✅ v1.7 前后端加密配置同步 (2026-05-20) — 1 phase, 6 plans
- ✅ v1.8 登录端点加密增强 (2026-05-21) — 1 phase, 4 plans
- ✅ v1.9 AD域控集成扩展 (2026-05-24) — 2 phases, 11 plans
- ✅ v1.10 网络设备权限修复 (2026-05-24) — 1 phase, 1 plan
- ✅ v1.11 AD组自动同步系统 (2026-05-26) — 1 phase, 18 plans
- ✅ v1.12 深信服桌面云集成 22A/22B (2026-06-02) — 6 plans
- ✅ v1.13 资产管理模块 (2026-06-08) — 1 phase, 6 plans
- ✅ v1.14 全局列自定义 (2026-06-09) — 1 phase, 1 plan
- ✅ v1.15 工位设备关联 + 部门物理位置映射 (2026-06-10 / 06-25) — Phases 28 + 39
- ✅ v1.16 技术债清理 (2026-06-26) — Phases 40-41, 8 plans
- ✅ v1.17 资产对账 (2026-07-03) — Phases 42-47, 16 plans
- ✅ v1.18 网络设备硬件清单 (2026-07-04) — Phases 48-49, 5 plans
- ✅ v1.19 网络设备写命令 (2026-07-08) — Phases 50-55, 9 plans
- ✅ v1.20 网络设备 VLAN + 端口绑定 (2026-07-10) — Phase 56, 5 plans
- ✅ v1.21 API Key 认证链修复 + 能力补全 (2026-08-18) — Phases 57-62, 8 plans (1+1+2+2+2+5) [含 Phase 62 跨 AI 评审修复]

## Session Continuity

Last session: 2026-08-19T07:06:33.896Z
Stopped at: Completed 69-03-PLAN.md (batch 2, commit ac33b2a)
Resume file: None

**Milestone status:** v1.22 IN PLANNING — **ROADMAP drafted 2026-08-18**,4 phases (64-67) / 15 requirements / 100% coverage。Token 层 (Phase 64) 落地位于 index.css 253 变量 + tokens/colors.ts xingranBrand + AntdThemeBridge + tokens/shadows+spacing+typography + QA-01 对比度校验;主题收敛 (Phase 65) 移除 6 套主题 + ThemeSwitcher/ColorSwitcher/themeStore 残留 + 13 消费方清理,保留 light/dark + layout/density;组件样式 (Phase 66) 侧边栏深绿 + 表格/卡片 + 按钮 D-03 纪律 + 表单/标签/ECharts + QA-02 硬编码扫描;终端验证 (Phase 67) 构建/类型/lint/test 全绿 + bundle 体积不增 + 6 屏前后截图对比。

## Operator Next Steps

- 启动 `/gsd:plan-phase 64` 进入 Phase 64(品牌令牌层落地 + 对比度验证)实施规划
- Phase 63 frontend-toolchain-automation 独立进行,可观察其 CI gate / lint-staged hook 完成时间,Phase 67 验证门集成前确认 CI 可用
- Phase 64 完成后按 64 → 65 → 66 → 67 顺序依次 plan-phase

## Performance Metrics

| Phase | Plan | Duration | Notes |
|-------|------|----------|-------|
| Phase 69 P01 | 46m | 3 tasks | 11 files |
| Phase 69 P06 | 15m | 1 tasks | 9 files |
| Phase 69 P02 | 42m | 2 tasks | 3 files |
| Phase 69 P03 | 15m | 1 tasks | 15 files |

## Decisions

- [Phase 69-01]: internal/models 为状态常量唯一真相源（既有 internal/constants 工具包不放状态语义）；新增 DictStatus/OperLogStatus/LoginLogStatus/JobLogStatus/VDIServerStatus/NoticeStatus 六家族 + status_constants_test.go AST 锁值（31 前缀/74 常量，跨包同名异值即 fail）+ check-status-literals.sh 四模式 ratchet 守护（基线 43 文件/149 命中，geocoding F 簇永久豁免）；批 1 services/system 5 文件 15 处常量化（widget_data_fetcher 的 m.status=0 确认为 sys_menu MenuStatusNormal，簇 A）
- [Phase 69-06]: 前端 status 共享常量落 src/constants/status.ts 三组（ENABLE_DISABLE/NORMAL_STOP/WORKSTATION_STATUS 三态组）——workstation 按 models.WorkstationStatus 注释判为三态业务簇不套两态组；menu 字符串 value 契约用 String(value) 派生保留；status 不进 sys_dict（Q2）由 status.test.ts 12 断言 + 69-01 后端 AST 锁值双向守卫
- [Phase 69-02]: migration_208 字典 seed 组级查重走 Unscoped（软删 dict_type 视为组已存在——防复活且防软删行占位 uniqueIndex 每次启动撞约束）；isDefault 取值规则 = archive/模型 gorm default 注释照抄，无来源组取组内第一条（sys_user_sex 默认 "2" 保密 = User.Gender default:2 而非 0=男；duty_holiday_type 默认 custom = Holiday gorm default）；dashboard_*/workorder_* 剔除不 seed（前端零 useDict 消费，planner Q4 圈定）
- [Phase 69-03]: 批 2 operations+excel 链路 58 处常量化：WorkstationDevice/Asset 无既有族且 WorkstationStatus(三态)/DeviceStatus(在线) 语义错配，按 69-01 登记机制新增无类型 WorkstationDeviceStatus/AssetStatus/AssetNBFStatus 三族(锁值 74→80)；双包陷阱按 model struct 实际所在包引用；白名单 38→27 文件
