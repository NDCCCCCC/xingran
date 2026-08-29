---
last_updated: 2026-08-23
update_trigger: v1.28 started in frontend-coverage workstream (frontend all-src coverage 3.67% -> >=70% + CI gate parity with backend v1.26)
previous_update: 2026-08-23 v1.27 started (backend coverage 55.6->>=70%, milestone workstream)
---

## Current Milestone: v1.28 前端测试覆盖率优秀 (Frontend Test Coverage Excellence)

> **并行结构（2026-08-23 起）:** 本项目现在用 GSD workstreams 并行推进两个里程碑——
> - `milestone` workstream: **v1.27 后端测试覆盖率优秀 II**（另一会话，STATE/ROADMAP 在 `.planning/workstreams/milestone/`）
> - `frontend-coverage` workstream: **v1.28 前端测试覆盖率优秀**（本段，STATE/ROADMAP 在 `.planning/workstreams/frontend-coverage/`）
>
> PROJECT.md / MILESTONES.md 跨 workstream 共享；Phase 编号错开（v1.27 用 75-81，v1.28 从 82 起）。

**Goal:** 前端全量口径（vitest `coverage.include` 全 src，白名单排除重画布低确定性 UI）语句覆盖率 **3.67% → ≥70%（优秀）**，建成与后端 v1.26 对称的 4 层 CI 防倒退 gate。

**Target features:**
- 口径修正 + 治理基建：vitest 加 `coverage.include` 全量口径（Vitest 4 已移除 `coverage.all`，Phase 63 基线 24.58% 是"只算被 import 文件"旧口径，真实全量 **3.67%** = 830/22602 stmts / 584 文件）；阈值 gate 切全量口径 + baseline 落盘 + ratchet 只升不降
- 分层补齐至 ≥70%（白名单后约 21,500 stmts）：P0 基建层 lib/utils/hooks/store/services/router (~3,900) → P1 组件层 components/* (~5,000) → P2 页面层 pages/* (~13,100，最大山头：operations 3611 / system 2203 / network 1962 / duty 1190 / ad-domain 1082)
- CI 全套 gate：全局阈值 gate + per-directory floor + baseline ratchet + PR diff coverage ≥80%

**锁定决策 (v1.28 init):**
- **D-01 目标线**: 语句 ≥70%（全量口径，对齐后端 v1.26/27 优秀定义）
- **D-02 范围**: 全 src + 白名单排除（候选 `components/cad-editor` 804 + `cad-elements` 224 stmts 等重画布 UI；白名单终版在 requirements 定）
- **D-03 CI gate**: 对齐后端全套 4 层（全局阈值 + per-dir floor + ratchet + PR diff coverage ≥80%）
- **D-04 Phase 编号**: 从 Phase 82 起（75-81 留给 v1.27）

**规划输入:** 2026-08-23 本会话实测扫描（npm test:coverage + coverage.include 全量口径，per-dir 数据见上）；对标后端 v1.26 4 层 gate 模式。

**范围边界:** 仅补测试 + 覆盖率治理，不修改业务逻辑（测试暴露的 bug 修复除外）；CAD/3D 重画布 UI 白名单排除不强制覆盖。

---

## Current Milestone: v1.27 后端测试覆盖率优秀 II (Backend Coverage Excellence II) — ✅ SHIPPED + ARCHIVED 2026-08-29

<details>
<summary>v1.27 ROADMAP (archived) — 点击展开</summary>

**Goal:** 加权平均覆盖率 55.6% → **≥70%**(收掉 v1.26 SC-a 缺口),5 个结构阻塞包逐一攻破至 ≥70%,并修复全部 15 个 v1.26 锁定的 QUIRK。

**Target features:**
- 集成测试基建全量引入:miniredis(core) / 嵌入式 LDAP server(addomain) / HTTP mock(operations) / SSH fake(device) / 子进程 stub(agent-server)
- 5 阻塞包各 ≥70%:addomain 2415 / operations 3714 / device 1249 / core 754 / agent-server 616 stmts
- 15 项 QUIRK 全部修复(业务变更,每个带回归测试):MemoryCache IncrementBy nil-panic+静默-0 / ModelExtractor 锚定 / gmsm sm2.Decrypt panic 防御 / validateFile 无扩展名 panic / retry.containsIgnoreCase 恒 true / normalizeParentID 塌缩 等
- ratcheted floor 随包达标逐个解除(70% 全量 floor 收口)

**锁定决策 (v1.27 init):**
- **D-01 目标线**: 加权 ≥70%(v1.26 SC-a 原目标,本 milestone 收口)
- **D-02 基建解禁**: v1.26 D-04"不引入新 mock framework"解除,全量引入测试专用 devDependencies(miniredis / LDAP test server / SSH fake 等);仍禁生产依赖变更
- **D-03 QUIRK 全修**: v1.26 D-12"零业务变更"解除——15 项 QUIRK 全部修复,每项带回归测试 + 独立原子 commit;生产语义变更需在 SUMMARY 记录影响面
- **D-04 防线不倒退**: v1.26 建成的 4 层 gate + diff coverage 全程保持绿;ratcheted floor 只升不降,包达标即解除对应豁免

**范围边界:** 覆盖率补齐 + QUIRK 修复;不做新业务功能;SCALE-02 工具包尾巴(gormutil/query/logger)不在本期范围。

**Progress (final):** Phase 75 (15 QUIRKS) ✅ · Phase 76 (5 INFRA) ✅ · Phase 77 (BLOCK-01/02 operations+agent) ✅ · Phase 78 (BLOCK-03/04/05 core+device+addomain) ✅ · Phase 79 (TAIL-01 services root) ✅ · Phase 80 (TAIL-02/03 scheduler+碎包) ✅ · Phase 81 (GATE-01/02/03 ratchet closeout + audit) ✅

**Ship metrics (CI run 33243477394 PASS):**
- Weighted avg: 55.60% → 78.12% (+22.52pp)
- Threshold ratchet: 55.5 → 77.5 (UP-only)
- P2 floor 10/10 × 70% PASS
- 3 P2_RATCHET lines removed (core/device/agent-server)
- BLOCK packages cleared: 4/5 (BLOCK-05 addomain 58% documented exemption — 12pp gap due to go-ldap/v3 BER incompatibility, gate impact = 0)
- TAIL-01/02/03 all cleared
- 15 canonical QUIRKs closed + ~100 new locked (deferred to future milestones)

**Archive location:** `.planning/milestones/v1.27-{ROADMAP,REQUIREMENTS}.md`

</details>

---

## Current State: v1.26 后端测试覆盖率优秀 — ✅ SHIPPED + ARCHIVED 2026-08-22

**Audit**: [v1.26-MILESTONE-AUDIT.md](milestones/v1.26-MILESTONE-AUDIT.md) — `partial` (4/5 SC fully met + SC-a honestly shortfall-documented)

| Milestone | Title | Phases | Plans | Items | Shipped |
|-----------|-------|--------|-------|-------|---------|
| **v1.26** | 后端测试覆盖率优秀 (Backend Test Coverage Excellence) | 71-74 | 34 | 19 reqs (15 ✅ + 4 partial) | **2026-08-22** |
| **Total** | **4 phases / 34 plans / 19 items / 2 days** | | | | |

**SC 状态**:
- **SC-a** weighted ≥70%: ⚠️ **55.56%** (ratchet 12.8→21.5→25.9→55.5;+42.76pp/4.34×;缺口结构化归因 + 7 pkg P2 ratchet 守住成果)
- **SC-b** 0%-pkg ≤5: ✅ **5** (全部入口/装配/生成代码)
- **SC-c** threshold gate: ✅ **55.5% UP-only** (4 phase ratchet chain 落 `coverage-baseline.md`)
- **SC-d** 0 FAIL: ✅ **65 ok / 0 FAIL / 0 panic**
- **SC-e** PR diff ≥80%: ✅ **74-10 D-14 自实现**(market 上 gocover-diff / ory/xcoverage-action 均不可达,自实现 bash+awk,ci.yml `coverage-diff` PR-only job)

**Delivered**:
- **Phase 71**: `check-coverage.sh` (bash + awk, D-01 零依赖) + ci.yml Coverage gate step + `.coverage-threshold` + `coverage-baseline.md` 起点 row
- **Phase 72**: P0 核心 13 plans — `api/v1/workorder 75.4%` + `api/v1/monitor 71.2%` + `api/v1/scheduler 85.5%` + `services/workorder 73.7%`;ratchet 21.5
- **Phase 73**: P1 重要 5 plans — 8 P1 包全部 ≥70% (D-04/D-10 strict);`p1_package_check` floor exit 4;ratchet 25.9
- **Phase 74**: P2 增量 + diff coverage 收口 — 11 plans + 74-12 escalation;P2 7/10 ≥70% + 3 UP-ONLY ratcheted;`p2_package_check` exit 5 + `coverage-diff` PR-only job;ratchet 55.5;原子 7 文件 commit (1f18e20) + SHA amend (33c8b5c)
- **业务代码变更**: **0** (D-12 STRICT 全程;~40 个 `*_test.go` 文件)

**Archive**:
- Combined: [milestones/v1.26-ROADMAP.md](milestones/v1.26-ROADMAP.md) + [milestones/v1.26-REQUIREMENTS.md](milestones/v1.26-REQUIREMENTS.md)
- Audit: [milestones/v1.26-MILESTONE-AUDIT.md](milestones/v1.26-MILESTONE-AUDIT.md)
- Per-plan: `.planning/phases/74-p2-finalize-and-diff-coverage/74-{01..12}-*` (preserved for full traceability)

<details>
<summary>📜 v1.22 + v1.23 + v1.24 + v1.25 SHIPPED details (archived 2026-08-19)</summary>

**Combined Launch Blocks Delivered** (audit: [v1.22-v1.25-MILESTONE-AUDIT.md](milestones/v1.22-v1.25-MILESTONE-AUDIT.md) — `passed`):

| Block | Title | Phases | Plans | Items | Shipped |
|-------|-------|--------|-------|-------|---------|
| v1.22 | 前端品牌化改造 (Frontend Brand Design-System) | 64-67 | 4 | 15 REQ | 2026-08-18 |
| v1.23 | 部署稳健性 & 文档一致性 (Deploy Robustness & Docs) | 68 | 1 | 5 DEPLOY-XX | 2026-08-19 |
| v1.24 | 字典与状态值治理 (Dict & Status Governance) | 69 | 8 | 4 DICT-XX | 2026-08-19 |
| v1.25 | 系统设置页面布局重构 (Settings Page Redesign) | 70 | 7 | 12 D-XX | 2026-08-19 |
| **Total** | **Combined Launch Blocks** | **7** | **20** | **36 items** | **2026-08-19** |

**Audit scores**: 20/20 reqs + 7/7 phases + 5/5 integration + 4/4 E2E flows

## Current Milestone: v1.26 后端测试覆盖率优秀 (Backend Test Coverage Excellence)

**Goal:** 后端 Go 加权平均测试覆盖率从 **12.8% → ≥70%(优秀等级)**,P0/P1 业务模块零测试清零,CI 落地 coverage 阈值 gate + PR diff coverage,使覆盖率从此不可无声倒退。

**Target features:**
- 治理基线 + CI gate:后端 CI 加 `-coverprofile` + 全局阈值 gate(失败即阻断)+ 基线数字落盘;pkg/cache flaky test 已修复(`5ead742`,Phase 71 直接验收)
- P0 核心补齐:api/v1/{workorder, monitor, scheduler, system}(全 0%)+ services/{workorder 0.6%, system 10.2%} 核心业务链路
- P1 重要补齐:api/v1/{duty, knowledge, rpa, vdi} + services/{duty, knowledge, monitor, network} 8 个零测试模块全清
- P2 增量收尾:api/v1/{operations, asset, network} + services/{rpa, vdi} + core/device 等大块增量,把整体推到 ≥70%,启用 diff coverage ≥80%

**锁定决策 (v1.26 init):**
- **D-01 目标等级**: 整体加权平均 ≥70%(优秀)——对齐 milestone 命名与 quick-260820-bcs 扫描建议
- **D-02 范围边界**: P0+P1 全清 + P2 增量至整体 ≥70%;纯 struct 模型 / cmd 入口 / docs 类辅助包(~2.5k stmts)不强制覆盖
- **D-03 CI gate**: 全局阈值 gate(失败即阻断)+ PR 增量 diff coverage ≥80%,与前端 Phase 63 模式对称
- **D-04 测试基建**: 沿用已有 glebarez sqlite in-memory,不引入新 mock framework
- **D-05 Phase 编号**: 从 v1.25 末尾 Phase 70 续编(71+),4 phases 按扫描建议拆分
- **D-06 Research skipped**: 覆盖率数据/模块清单/CI 现状均已落盘(quick-260820-bcs),无需域研究

**规划输入:** `.planning/quick/260820-backend-test-coverage-scan/SUMMARY.md`(74 业务包 / 12.8% 加权 / 33 个 0% 包 / P0-P2 分级清单 / 4-phase 拆分建议 / 5 SC 建议)

**范围边界:** 仅补测试 + CI 治理,不修改业务逻辑(测试暴露的 bug 修复除外);辅助工具包(models 纯 struct / cmd main / internal/docs)不强制覆盖。

---

## Current Milestone: v1.22 前端品牌化改造 (Frontend Brand Design-System) — ✅ SHIPPED 2026-08-18

<details>
<summary>📜 v1.22 SHIPPED details (archived)</summary>

**Goal:** 把 `brand-spec.md` 的像素实测品牌令牌（深绿 `#156031` × 铜金 `#C09058` × 奶油 `#F0ECE3`）固化进 `xingran-react-frontend/src/design-system/`，让后台内部组件与登录页品牌一致，53 屏业务页面自动继承样式。

**Target features:**
- 品牌令牌层：`tokens/colors.ts` 落地 XingRan 品牌色板（绿梯度 6 阶 / 铜金梯度 4 阶 / 奶油中性 6 阶），含 OKLch + WCAG 对比度注释
- 全局替换：废弃 6 套主题（minimal / glassmorphism / neumorphism / flat2.0 / luxury-quiet / ink-amber）与 ThemeSwitcher / ColorSwitcher，单一品牌视觉
- Antd 6 主题桥：`AntdThemeBridge.tsx` 的 token/component 覆盖全量接品牌令牌
- 清除品牌冲突色：`#4F46E5` indigo 主按钮 → `--primary` 绿底白字；`#F1F5F9` 冷蓝灰表头 → `#E9EFEB` 绿灰淡彩；`#FEF3C7` 由按钮前景改回淡黄标签底
- 侧边栏深绿化：现浅色 → `#14532D` 底 + `#E0E0B0` active 强调
- 通用组件样式统一：表格（表头 / 斑马纹 / 边框）、表单、卡片、按钮体系（按钮纪律：铜金不做实心主按钮，`#C09058` 上白字仅 2.85:1 不达标）
- 对比度门禁：关键前景/背景对满足 WCAG AA

**素材来源:** `655aa291-9bfe-4e94-ad5d-b3c8b2d24984/` — `brand-spec.md`（像素实测令牌 + 对比度验证）、`admin-design-plan.md`（现状侦察 + 品牌化缺口定位）、53 张高保真 HTML 原型屏、`PROTOTYPE-VS-ACTUAL.md` 差异清单、`refs/` 真实系统截图。

**范围边界（用户决策 v1.22 init）:** **仅 design-system 层**。业务页面自动继承样式，不逐屏改造 53 屏；`PROTOTYPE-VS-ACTUAL.md` 里的字段 / 表头 / 路由差异修正转 Future。

**锁定决策 (v1.22 init):**
- **D-01 主题策略**: 全局替换，不保留多主题 —— 品牌令牌直接写入 `tokens/` + `index.css` + `AntdThemeBridge`，移除主题切换能力（用户明确选择，不可逆）
- **D-02 品牌基准**: `brand-spec.md` 为唯一色值来源，标注「实测」的直接采用，「推导」的可微调但需重跑对比度验证
- **D-03 按钮纪律**: 主按钮一律 `--primary` 绿底白字（7.64:1）；铜金仅作点缀 / 描边 / 图标 / 图表系列，实心场景必须 `#B88850` + ≥16px 半粗体白字
- **D-04 Phase 编号**: 从 v1.21 末尾 Phase 62 续编（63+）

**Phase 编号:** 从 v1.21 末尾 Phase 62 续编（63+）。

**已 SHIPPED 2026-08-18** — 详见 [milestones/v1.22-ROADMAP.md](milestones/v1.22-ROADMAP.md) + [milestones/v1.22-REQUIREMENTS.md](milestones/v1.22-REQUIREMENTS.md)

</details>

---

## v1.22 + v1.23 + v1.24 + v1.25 启动块 (Combined Launch Blocks) — ✅ SHIPPED + ARCHIVED 2026-08-19

**Audit**: [v1.22-v1.25-MILESTONE-AUDIT.md](v1.22-v1.25-MILESTONE-AUDIT.md) — `passed` (20/20 reqs + 7/7 phases + 5/5 integration + 4/4 E2E flows)

**Goal**: 四个连续启动块一次性落地 — 品牌化(design-system 层)+ 部署稳健性(SM2 闭环)+ 状态/字典治理(单一真相源)+ 系统设置重构(对齐 v1.22 品牌)。7 phases / 20 plans / 36 items 全部 delivered。

**Target features / 工作池:**

- **v1.22 (Phases 64-67) — Frontend Brand Design-System** — 把 `brand-spec.md` 像素实测品牌令牌固化进 design-system;移除 6 套主题;硬编码色扫描;6 屏视觉确认
  - 品牌令牌层:`tokens/colors.ts` `xingranBrand`(绿 6 / 铜金 4 / 奶油 9 + OKLch + WCAG)+ `index.css` 253 变量 + `AntdThemeBridge` 全量映射
  - 主题收敛:删除 6 套主题(minimal/glassmorphism/neumorphism/flat2.0/luxury-quiet/ink-amber)与 ThemeSwitcher/ColorSwitcher;保留 light/dark + layoutStore 布局/密度切换
  - 组件样式:侧边栏深绿化(`#14532D`)+ 表格/卡片双层纸感 + 按钮纪律 D-03(主按钮 `#156031` 绿底白字)
  - QA:对比度自动验证 + 硬编码色扫描 lint/CI 门 + 构建回归 + 6 屏前后视觉确认
- **v1.23 (Phase 68) — Deploy Robustness & Docs Consistency** — 闭环 SM2 密钥配置部署稳健性
  - DEPLOY-01:17 处 XINGRAN_JWT_SM2 → JWT_SM2(与 BindEnv 一致)
  - DEPLOY-02:setup-server.sh secrets.env 补 SM2 密钥对段(非对称,不能动态生成)
  - DEPLOY-03:gen_sm2_keys header 注释路径修正
  - DEPLOY-04:getPublicKey 500 时 WARN 日志(useSM2/sm2PublicKeyLoaded 状态)+ 2 个最小 getter
  - DEPLOY-05:sqlite use_sm2 默认值与 prod 一致(`true`) + 6 行迁移指引
- **v1.24 (Phase 69) — Dict & Status Governance** — 状态语义单一真相源
  - DICT-01:internal/models 状态常量作为语义真相源;94 常量 AST 锁值测试;白名单 ratchet 守护(终态 1 条 F 簇豁免 geocoding)
  - DICT-02:migration_208 字典 seed 11 组(8 组 archive 重建 + ops_workstation_type/sys_user_sex/duty_holiday_type 新增)
  - DICT-03:前端 4 页 type 下拉 useDict 迁移(user/workstations/holidays/devices)+ status.ts 共享常量(7 文件引用)
  - DICT-04:CLAUDE.md Status Value Convention 改指向常量真相源(删 6 行表格)
- **v1.25 (Phase 70) — Settings Page Layout Redesign** — 系统设置页对齐 v1.22 品牌
  - D-01/D-03/D-04:SettingsShell 布局(左分类 + 右内容)+ URL `?cat=` 驱动 + `<lg` Segmented 降级
  - D-02 混合宽度:表格/网格类撑满 + 用户设置 760px
  - D-05 两页同构:系统设置 + 用户设置共用 SettingsShell.tsx
  - D-06 行式设置项:label+描述+右对齐控件 + 即改即存
  - D-07 v16 范式:统计卡 + 品牌工具栏 + `.xr-table-zebra` 双层纸感
  - D-08 网格墙:验证码 `.xr-captcha-grid` + status 反转语义(1=启用)
  - D-09:5 个 Modal 容器结构与宽度不变
  - D-10:工作区 default-theme 清理 13 文件(+42/-717)首提交入库
  - D-11:三个 settings 目录合并 + Migrate208 sys_menu 迁移 + 菜单缓存失效
  - D-12:settings 范围内死目录 / dead actions / preset Tag / persisted activeTab 键 / fallback 硬编码色清零

**锁定决策 (combined):**

- v1.22 D-01..D-05:全局替换不保留多主题(不可逆)+ brand-spec.md 唯一色值源 + 按钮纪律绿底白字 + Phase 64 续编 + 仅 design-system 层
- v1.23:DEPLOY-01~05 闭环 SM2 配置链路;live user-managed configs/config.yaml + .env 由运维手工迁移(phase 范围外)
- v1.24:internal/models 为状态常量唯一真相源;type/category 真枚举 → sys_dict;前端 constants.tsx → useDict + status.ts 共享常量;CLAUDE.md 指针化
- v1.25:D-01..D-12 全部 PASS — SettingsShell 共用骨架 / 混合宽度 / URL 驱动 / Segmented 降级 / 行式即改即存 / v16 范式 / 网格墙 / Modal 契约不变 / default-theme 清理 / 目录合并 + Migrate208 / 残留清零

**回归性质:** 启动块序列组合,各自独立 + 局部依赖(v1.25 依赖 v1.22 品牌基线;其他独立)。v1.22 是纯 design-system 层重构;v1.23 是部署/文档稳健性修复;v1.24 是状态语义治理;v1.25 是设置页布局重构(对齐 v1.22 品牌理念)。

**Phase 编号:** 从 v1.21 末尾 Phase 62 续编(63+)。Phase 63「前端工具链自动化」独立 IN PROGRESS,提供 CI / lint / 测试基建。

**范围边界(用户决策各 init):**

- v1.22 仅 design-system 层 — 业务页面自动继承样式,不逐屏改造 53 屏;PROTO 逐屏对齐与 VIS 视觉深化转 v1.23+ Future
- v1.23 仅 6 个 `*.example.yaml` + docs + scripts;live 配置由运维手工迁移
- v1.24 守 F 簇(geocoding 百度地图 API 外部契约)永久豁免
- v1.25 仅前端布局重构,不改业务逻辑与 API 契约

**Status:** ✅ SHIPPED + ARCHIVED 2026-08-19

**归档:**

- Combined: [milestones/v1.22-v1.25-ROADMAP.md](milestones/v1.22-v1.25-ROADMAP.md) + [milestones/v1.22-v1.25-REQUIREMENTS.md](milestones/v1.22-v1.25-REQUIREMENTS.md)
- Per-block:milestones/v1.22-*, v1.23-*, v1.24-*, v1.25-*(完整 phase 细节保留)

---

## v1.21 API Key 认证链修复 (API Key Auth Chain Repair) — ✅ COMPLETE 2026-08-17

**Goal:** 修复 API Key 认证系统的 P0/P1/P2 缺陷,回归 v1.6「API 密钥管理系统」(Phase 16)的可用性与可观测性,并让 MultiAuth 认证链代码就绪可启用。

**Target features / 修复项:**
- 修复 `setUserContextForAPIKey` 类型断言恒 false(P0-2)— API Key 上下文无法写入 gin context
- 消除 `MultiAuth`/`RequireScope`/`RequireAPIKeyResourcePermission`/`RateLimitByScope` 死代码,使认证链代码就绪(P0-1)
- 修复前端 `getAPIKey`/`updateAPIKey`/`deleteAPIKey` 与后端 POST 路由契约不匹配导致的 404(P1-1)
- 修复使用日志记录时机导致 `successRate` 永久失真(P1-2)— 在 `c.Next()` 前记录致 Success/StatusCode/Duration 全零值
- 修 P2:限流响应头 `string(rune(int))` 编码错误、使用日志 ctx 取消竞态、密钥明文存储评估、`idx_api_keys_key` 与 uniqueIndex 重复索引

**范围边界(用户决策 v1.21 init):** **全修复 + 就绪** — 修复全部 P0/P1/P2 确定性缺陷,MultiAuth 代码修好可接入;「是否在生产路由挂载 MultiAuth(即 X-API-Key 认证是否真正生效)」作为 phase 内 discuss 决策点,含安全影响评估。研究阶段跳过(回归修复,代码与问题均已调查清楚)。

**回归性质:** 对 v1.6「API 密钥管理系统」(Phase 16 / 2026-05-19 / 10 plans)的回归修复,非新能力。涉及文件:`internal/middleware/apikey.go`、`internal/api/router.go`、`internal/api/v1/system/apikey_router.go`、`internal/services/usage_logger.go`、`internal/services/rate_limiter.go`、`internal/services/system/apikey_service.go`、前端 `src/api/apikey.ts`、migration `migration_085_api_keys.go`。

**Phase 编号:** 从 v1.20 末尾 Phase 56 续编(57+)。

**Phase 54 (2026-07-07)**: W5 收尾验证 phase — scrapligo `transport.NewFileTransport()` e2e 闭环 Phase 51 fn-never-called 漏洞 (10 TestE2E_*) + 5 处文档更新 (API响应规范 + 加密设计 + CHANGELOG 新建 + README 能力扩展 + MILESTONES v1.19 条目 + STATE deferred 同步) + 54-HUMAN-UAT.md 创建 (7 pending: 6 SSH + 1 WR-02) + 全量回归三绿 (vendor-react 零回归 774.96 kB + operlog regression 不回归)。代码审查 5 Info / 0 Critical / 0 Warning;**v1.19 网络设备写命令 milestone SHIPPED**。

**Phase 53 (2026-07-07)**: W4 frontend layer complete — 6 port-write API wrappers (`networkApi.ts`) + 3 TS types mirroring backend `portwrite` Go structs (`types/network.ts`) + shared constants (`port-write/constants.ts`) + `PortWriteModal` (5 single-port actions, reason Select+TextArea, description special-case, audit-link Toast) + `BulkWriteDrawer` (select→executing→result state machine, indeterminate Spin per D-05, three Statistic cards, retry-only-failed per D-06) + `ports/index.tsx` wiring (`useMenuStore` canWrite gating per D-09, operation column, batch-config button, `batchInProgress` disables refresh+collect per D-07/LANDMINE #4) + `monitor/logs` URL query `?module=` prefetch. 8/8 must-haves verified, zero gaps. Code review found 2 BLOCKERs (CR-01/CR-02: BulkWriteDrawer retry `deviceId` snapshot drift → potential cross-device write) — fixed in commit `9b01cc68` (cache `lastDeviceId` + `lastInterfaceMap` at submit). WR-02 (custom-reason validator signature) deferred to Phase 54 UAT. Real-device/browser UAT deferred to Phase 54 (W5) per v1.18 Phase 48 site-visit precedent. `npm run build` exit 0, vendor-react gzip 774.96 kB (zero regression vs Phase 48 baseline 776 kB).

**Phase 52 (2026-07-07)**: W3 wiring layer complete — 6 port-write HTTP endpoints exposed (`/network/ports/write/{shutdown,undo-shutdown,description,dot1x-enable,dot1x-disable,batch}`) with 2-arg `RequirePermissions(["network:port:write"], core)`, `PortWriteAudit` 12-col GORM model + composite indexes via GORM AutoMigrate (Path A), 端口配置 F-type menu seeded under parent 端口状态 (D-07) with precise idempotent role-grant helper. Path C strictly enforced (audit_id embedded via operlog.WithOperParam; audit.oper_log_id stays NULL; operlog package unmodified). 16/17 must-haves verified, 1 ACCEPTED_AS_KNOWN_LIMITATION (WR-05: NetworkPortWrite not yet in GetRoutePermissions, discoverability-only). Human verification deferred to PG dev DB + Phase 53 frontend.


## Current Milestone: v1.19 网络设备写命令 (Network Device Write Operations)

**Goal:** 补全网络设备"读+写"闭环——Web 端通过 SSH 直接对目标设备下发端口配置命令（启停/描述/dot1x），成功后立即采集一次端口信息回填审计。

**Target features / 工作池:**
- SSH 写命令基础设施（基于 Scrapli + 厂商命令模板映射）
- 多厂商命令模板：Huawei + H3C + 锐捷（Cisco 后续 phase）
- Web UI：端口列表 + 设备详情 + 批量操作对话框（单 + 批量）
- 改后采集触发链路（复用 v1.18 DeviceInfoCollectionService）
- 操作审计（operlog.Record 强制约定 + 操作结果回传前端）
- 权限控制（perms 命名空间 network:port:write）
- MVP 范围：端口启停（shutdown/undo shutdown）、端口描述（description）、dot1x 启停
- 批量执行：串行失败即停（避免大批量配置错误）

**锁定决策 (v1.19 init):**
- 策略：device_id 直连（复用现有凭据库），不引入 device_ip 临时建连
- 命令模板：硬编码 vendor→template map（落地为先，后续 phase 抽象为数据库）
- 厂商支持：Huawei / H3C / 锐捷（MVP），Cisco 后续
- 改后采集：复用 DeviceInfoCollectionService.enqueue（v1.18 资产）
- 操作回滚：MVP 仅"失败即停 + 回传失败点"，暂不实现自动回滚配置

**与 v1.18 关系：** v1.18（已 ship）补"读"——板卡/chassis/elabel/接口状态采集；v1.19 补"写"——端口配置下发。两者一起构成网络设备"读+写"完整闭环。

**Replaces v1.19 原候选范围：** 真机 site-visit UAT 闭环 / 残余 deferred debug / VDI 22C/22D 推后到 v1.19+ follow-up phase。

## Active: v1.17 资产对账 (Asset Reconciliation) — SHIPPED 2026-07-03

**Goal:** 建立"实物层 vs 声明层"对账引擎，使资产系统数据保持与实际情况相符。

**Target features / 工作池:**
- 多源数据冲突检测（物理链路 vs 系统责任人 vs AD managed_by）
- Observe-only 策略 + 告警驱动人工修复（不动业务表）
- IP 段级别例外规则（5 种 actions：no_alert/no_notice/no_workorder/skip_severity/silence）
- 工位详情页整合对账健康度（顶部卡片 + 行内徽标 + 抽屉）
- 自动转工单闭环（critical/high 异常）
- 半自动修复建议（高置信度，R5 可选）

**5 phases (42-46):**
- Phase 42 / R1: 观测底座（物化视图 + reconciliation 表 + dashboard + Statistics 端点）
- Phase 43 / R2: 告警 + 工单闭环
- Phase 44 / R3: 置信度评分 + IP 段例外规则 + 例外管理页 + 命中测试
- Phase 45 / R4: 工位详情整合 + 资产详情摘要块
- Phase 46 / R5: 半自动修复（可选）

**关键决策 (locked):**
- 策略：Observe-only（不修改业务表，仅记录 + 告警 + 人工修复）
- 菜单归属：资产管理 / 数据质量（资产对账 + 3 个二级菜单）
- 权限命名：asset:reconciliation:*
- API 路由：/asset/reconciliation/*
- 跨模块调用：ops/workstation → asset/reconciliation（service 层直接 + 权限降级）
- Owner 合并：运维 + 资产 + 权限 同一人（无需双签，单人签字）

**复用审计结论 (v0.4):**
- ✅ 已复用 13 项：字典/参数/operlog/Handler-Service/CacheProvider/Excel/UUID/Status/前端 hooks/opsApi/ECharts/UI库
- ⚠️ 部分复用 4 项：字典枚举值/参数 seed/Excel config/operlog module 常量
- ❌ 必须新增 7 项：cache_key helper/Statistics 端点/queryKeys/operlog module/HealthScore 函数/路由注册/Cron 注册

**R1 启动门槛 (✅ 已满足):**
- 端口采集覆盖率达标（运维 owner 确认）
- T1-T7 基础设施调查完成
- v0.3 架构 + v0.4 复用审计 + 跨模块权限声明已落盘

**决策记录 (.planning/notes/):**
- asset-reconciliation-strategy.md (v0.3 + v0.4)
- 260627-reconciliation-reuse-audit.md
- 260627-cross-module-permission.md
- 260627-port-coverage-audit.md

**Seed:** .planning/seeds/asset-reconciliation-v1.17.md

**Todo:** .planning/todos/pending/v1.17-reconciliation-decisions.md (T1-T30)

# XingRan-Next 运维管理系统

## What This Is

XingRan-Nect is a Go backend + React frontend enterprise IT operations
management system. It has evolved from a workstation-import
association feature (v1.0) into a multi-domain platform covering:

- **Operations domain**: buildings, floors, workstations, server rooms,
  info-points, dedicated lines, assets (CRUD + Excel import/export
  with geocoding).
- **Network domain**: device discovery, SNMP, Scrapli-based command
  execution, credential vault, topology, MAC history, port collection.
- **Identity domain**: AD/LDAP integration (login, OU-dept mapping,
  group auto-sync), user/role/dept/menu management, RBAC.
- **VDI domain** (v1.11): Sangfor VDI virtual desktop management with
  VM CRUD, power operations, user binding, server config.
- **Audit domain** (v1.15 / Phase 34): full-coverage operation logging
  for all 309 write endpoints with sensitive-keyword masking.

National cryptography (SM2/SM3/SM4) is mandatory for in-transit and
at-rest data on sensitive endpoints.

## Core Value

End-to-end operational observability and auditability: every write
action in the system produces a traceable record (who, when, what,
from-where, before/after-state) with sensitive fields auto-masked.
Combined with Excel-driven bulk management (import/export with
auto-resolution of references like dept-name → UUID), this gives
operators a single trustworthy system-of-record for the entire IT
estate.

## Requirements

### Validated

- ✓ 工位 CRUD（创建、读取、更新、删除）— 现有
- ✓ Excel 批量导入/导出工位 — 现有
- ✓ 工位关联 `org_id`（部门 UUID）— 现有字段，导入时通过部门名称匹配
- ✓ 前端工位列表/详情页面 — 现有
- ✓ 部门表 `sys_dept` 含 `dept_name` 字段 — 现有
- ✓ 用户表 `sys_user` 含 `user_name` 字段 — 现有
- ✓ Excel 导入模板添加"所属部门"和"所属用户"两个可选列 — v1.0
- ✓ 导入时通过部门名称/用户名匹配，写入对应 ID — v1.0
- ✓ 匹配失败留空不阻断导入 — v1.0
- ✓ 信息点导入配置添加"所属设备"列 (`deviceName` → `device_id`) — v1.1
- ✓ 信息点导入配置添加"所属端口"列 (`portName` → `port_id`) — v1.1
- ✓ 两个新列均为可选，空值不阻断导入 — v1.1
- ✓ 模板下载包含设备和端口示例值 — v1.1
- ✓ Widget 数据获取机制 (GetWidgetData/GetBatchWidgetData) — v1.2
- ✓ 前端交互完善 (Widget 选择器、仪表盘设置、模板预览) — v1.2
- ✓ 实时数据刷新 (WebSocket 连接、轮询、缓存) — v1.2
- ✓ 用户体验优化 (拖拽布局、加载状态、响应式) — v1.2
- ✓ SNMP panic 修复 (panic 恢复 + 并发安全 + 连接验证) — v1.3
- ✓ 后端代码优化 (Core 清理 + 安全修复) — v1.3
- ✓ 批量导出功能 (多实体 ZIP 打包下载) — v1.3
- ✓ **v1.16 技术债清理** (Tech-Debt Cleanup) — Phases 40/41 — 2026-06-26 (22+29 个 deferred debug session + 6 audit 文件规范化)
- ✓ **v1.17 资产对账** (Asset Reconciliation) — Phases 42-46 — 2026-07-03 (observe-only 对账引擎 + 工单闭环 + IP 段例外 + 工位详情整合 + 半自动修复建议 + UAT 9/10 PASS)
  - R1 观测底座：物化视图 + Statistics 端点 + operlog 全覆盖
  - R2 告警+工单：critical/high 自动转工单 + WebSocket + SysNotice + 7d 静默期
  - R3 IP 段例外：CIDR + 5 actions + 命中测试 + Excel 导入导出
  - R4 工位详情：健康度卡片 + ReconciliationDrawer 3 Tab
  - R5 半自动修复：6 状态机 + 7d 回滚 + 误修复率告警三通道
- ✓ **Phase 47 根因修复** (infoPoint drift / Layer3 UPSERT / port_status 漂移 / MAC parser 校验) — 2026-07-03
- ✓ **v1.18 网络设备硬件清单** (Device Component Serials) — Phase 48 — 2026-07-04 (1 phase / 3 plans / 3 waves, 14/14 D-id 覆盖 + 3 site-visit UAT 推迟)
  - Wave 1 Schema：migration_201 给 `ops_asset` 加 4 列 (parent_asset_id / source_device_id / component_type / component_slot) + `sys_data_reconciliation.recon_category`
  - Wave 2 Collectors：`internal/services/component_collector/` 17 文件包,SNMP ENTITY-MIB 单 GET + Ruijie `temprature*` 噪声过滤 + Huawei dual-class 去重 + OwnerResolver stack containtedIn tree + Huawei/Ruijie CLI 双厂商 + 6 TextFSM 模板
  - Wave 3 Pipeline+Audit+UI：OpsAssetWriter UPDATE-only + ReconciliationEmitter (component_serial recon_category) + DeviceInfoCollectionService cron hook + operlog.RecordBackground (cron-path) + GET /ops/asset/components + ComponentListTab expandable row
- ✓ **v1.19 网络设备写命令 (Network Device Port Write Operations)** — Phases 50-55 — 2026-07-08 (6 phases / 9 plans / 5 build waves + 1 cleanup phase, 37/37 MVP requirements, 108 files / 25,224 insertions / 3,152 deletions in 1.7 days)
  - W1 Vendor Templates: `internal/services/portcollection/vendor_port_template.go` 15 (vendor, action) templates (Huawei/H3C VRP heritage + Ruijie Cisco-style) + 20 table-driven tests
  - W2 PortWriteService + Batch: `internal/services/portwrite/` 5-file package — parseConfigError 5-step priority scan + checkPreState NoOp detector + BatchWritePorts detached 30min context + serial fail-fast; 28 mock unit tests
  - W3 Router/Handler/Operlog/Permission/Migration: 6 kebab POST 端点 `/network/ports/write/{shutdown,undo-shutdown,description,dot1x-enable,dot1x-disable,batch}` + `NetworkPortWrite = "network:port:write"` 组级 2-arg RequirePermissions + `sys_port_write_audit` Path A (AutoMigrate) + `GrantNewMenuToRolesHavingParent("端口状态")` 精准授权 + Path C audit↔operlog 关联 (audit_ids 嵌 operlog.oper_param; audit.oper_log_id 保持 NULL)
  - W4 Frontend: PortWriteModal 5-action unified Modal + BulkWriteDrawer select→executing→result 状态机 (D-05 indeterminate Spin 不伪造 X/Y) + ports/logs 页 useMenuStore canWrite gating + audit-link Toast + monitor/logs URL query 预填 + vendor-react gzip 774.96 kB 零回归
  - W5 E2E + UAT Deferral + Docs: scrapligo FileTransport 10 TestE2E_* (闭环 Phase 51 mockDeviceExecutor 不调 fn 漏洞) + API响应规范 + 安全设计 + CHANGELOG/README/MILESTONES/STATE 文档同步 + 54-HUMAN-UAT.md 7 项 deferred (owner = 现场运维同事)
  - Phase 55 Tech-Debt Cleanup: WR-02 antd validator 3-param signature + IN-01 instanceof Error narrowing + IN-02 eslint-disable placement 修正 (Lesson: directive 紧贴 `}, []);` 而非 useEffect 开头) + HealthCard.test.tsx regex match + CR-02 后端 batch_orchestrator.go fallback port 归属跨层防御 (纵深双保险)
- ✓ **v1.22 + v1.23 + v1.24 + v1.25 启动块 (Combined Launch Blocks)** — Phases 64-70 — 2026-08-18 → 2026-08-19 (7 phases / 20 plans / 36 items, audit `passed` 20/20 reqs + 7/7 phases + 5/5 integration + 4/4 E2E flows)
  - v1.22 (Phases 64-67): 253 CSS 变量 + xingranBrand 11 键 + AntdThemeBridge + 6 主题移除 + 硬编码扫描 + 6 屏视觉确认(15/15 reqs)
  - v1.23 (Phase 68): 17 处 XINGRAN_JWT_SM2 → JWT_SM2 + setup-server.sh SM2 段 + getPublicKey 可观测 + sqlite use_sm2 默认(5/5 DEPLOY-XX)
  - v1.24 (Phase 69): 94 状态常量 AST锁值 + 11 字典 seed + 4 页 useDict 迁移 + CLAUDE.md 指针化(4/4 DICT-XX)
  - v1.25 (Phase 70): SettingsShell 共用骨架 + 14 .xr-* 选择器 + 行式即改即存 + Migrate208 sys_menu 迁移(12/12 D-XX)

### Active

## Current Milestone: v1.19 网络设备写命令 (Network Device Port Write Operations) — SHIPPED 2026-07-08

**v1.19 已闭环** — 6 phases (50-54 build + 55 cleanup) / 9 plans / 5 build waves + 1 cleanup / 37 MVP requirements all addressed (35 fully satisfied + 2 PARTIAL BATCH-05/UI-03 + 1 adjusted PERM-03 D-07 父菜单名). 闭环分析:
- **Build (W1-W5)**: Phase 50 (vendor templates) → Phase 51 (service + batch + mock) → Phase 52 (router/handler/operlog/permission/migration) → Phase 53 (frontend drawer + API wrappers) → Phase 54 (e2e + UAT deferral + docs)
- **Cleanup**: Phase 55 (tech-debt cleanup of Phase 53 leftovers — WR-02 custom-reason validator + IN-01 instanceof Error + IN-02 eslint-disable placement + HealthCard.test.tsx + CR-02 后端 batch_orchestrator 跨层防御)
- **Tech Debt Zero-regression**: Phase 51 mock + Phase 54 e2e 全绿(28+35 tests), operlog 25 OperType + 11 敏感关键词 regression_test.go 不回归, vendor-react gzip 774.96 kB 与 Phase 53 baseline 零回归
- **Phase 51 mock 漏洞 闭环**: scrapligo `transport.NewFileTransport()` 真实 SendConfigs + parseConfigError 链路, 10 TestE2E_* 测试全绿(5 single happy + 1 batch happy + device_rejected + transport_error + batch_failfast + noop)
- **文档同步**: API响应规范.md (网络设备端口写操作小节) + 加密设计.md (SSH vs HTTP 两层加密正交) + 新建 CHANGELOG.md + README.md 核心特性扩展 + MILESTONES.md v1.19 条目
- **UAT 推迟**: 6 项 SSH verification (Huawei/H3C/Ruijie × shutdown/description/dot1x) → 54-HUMAN-UAT.md (推迟到下次现场访问, owner = 现场运维同事); 7 项全 `pending`, 全 `verifier_status: human_needed`
- **Pre-existing 测试失败** (login_encryption_test.go:235 / pkg/errors/errors_test.go / operations 子树) 均先于 Phase 51-54 6 周-5 个月, git diff empty for these files, 不算 v1.19 缺陷

**Phase 54 must-not-break 全部守住**: SC#6 三绿 (go test portwrite + npm build vendor-react gzip 774.96 kB 与 Phase 53 baseline 零回归 + npm type-check); SC#7 operlog regression_test.go (25 OperType + 11 keyword + Record 5 参签名); SC#3 加密配置实证 (config.yaml exclude_paths 不含 /network/ports/write/*, D-04 lock)。

**🔜 Next Milestone (v1.19.x+ follow-up candidates)**:

参见 `milestones/v1.19-ROADMAP.md` §Tech Debt Incurred + `milestones/v1.19-REQUIREMENTS.md` §Future Requirements 列出的 14 项 future work + 5 项 tech debt（WR-01 / WR-03 / WR-04 / WR-05 + BATCH-05 real-time progress）。优先 candidates:
- **BATCH-05 real-time progress via SSE/WS** (MVP 用 indeterminate Spin) — `internal/services/portwrite/` 进度通道 + `BulkWriteDrawer` 改造为真实 X/Y 进度
- **3-vendor e2e fixture 全覆盖** (Huawei done; H3C + Ruijie 待补)
- **6 site-visit SSH verification** (Huawei S5700/S5735/H3C/Ruijie × shutdown/description/dot1x) — owner = 现场运维同事
- **WR-01..05 + IN-XX** remaining tech-debt items

## Current Milestone: v1.18 SHIPPED 2026-07-04 — awaiting v1.19 planning

**v1.18 已闭环** — Phase 48, 3/3 plans, 0 critical regressions, 14/14 D-id 覆盖 + 3 项 site-visit UAT 推迟到下次现场访问(S8700/RS8607E 真机)。

**Phase 49 v1.18 gap-closure (2026-07-05)** — 用户报告 RS8607E-03 展开组件清单为空,诊断出 3 个连锁缺口(Gap 3 chassis SN 全空 → Gap 2 关联断裂 → Gap 1 collectComponentInfo dead code)。2 plans 修复:enrichChassisSerial 接入 + collectBoardsInto 接入(chassis 行丢弃)。生产 E2E 验证:14 台设备 serial_number 回填(部署前 0),RS8607E-03 展开 9 条组件(7 板卡 + 2 光模块),超过 ≥6 门槛。E2E 期间附带修复 ReconciliationEmitter 的 PG `IS ?` SQLSTATE 42601 跨方言 bug。verifier 7/7 passed。

## Active: v1.19 (planning) — 候选范围

候选方向(择 1-3 个,phase scope 决策)：

- **真机 site-visit UAT 闭环** — 把 48-HUMAN-UAT.md 的 3 项 deferred(S8700 / RS8607E SNMP + CLI + D-10 两步流水线)由现场运维同事跑通后回写 `48-HUMAN-UAT.md` 为 `passed`;同时确认 49-HUMAN-UAT Step 3 前端 Tab 渲染
- **Phase 49 deferred follow-ups** — Huawei S5735 `ESN of slot` parser 增强固定形态设备 chassis SN 提取;连接池 TOCTOU 竞态根治(`connection_pool.go:266-273`);`recoverPendingTasks` 同时清理过期 `running` 僵尸任务(06-13 crash 暴露)
- **残余 deferred debug session 清理** — 当前 audit-open 18 debug_sessions + 60 quick_tasks,虽然 Phase 41 已闭环 29 个,仍可继续推回到 debug_sessions < 5
- **VDI 22C / 22D** — 依赖 v1.11 生产稳定性观察数据(账号管理 + Web Console 子阶段)
- **新功能(mobile app / AI features)** — 待业务决策

**Dependency**: v1.19 候选项均与 v1.18 独立;真机 site-visit 是唯一依赖 Phase 48 部署环境的项。

See `.planning/ROADMAP.md` for current state + `.planning/MILESTONES.md` for shipped-milestone history.

### Completed

#### v1.3 技术债清理 — ✅ Completed 2026-04-27

**SNMP-01: SNMP Panic 修复**
- [x] SNMP-01a: 实现 panic 恢复包装器
- [x] SNMP-01b: 实现 RWMutex 并发安全机制
- [x] SNMP-01c: 实现连接就绪性验证

**CODE-02: 后端代码优化**
- [x] CODE-02a: 系统性扫描死代码（无死代码可删除）
- [x] CODE-02b: 清理 Core 结构（删除 2 字段，文档化 12 字段）
- [x] CODE-02c: 修复安全问题（WebSocket、并发安全、错误日志）

**EXPORT-03: 网络设备导出**
- [x] EXPORT-03a: 注册导出路由 ✅
- [x] EXPORT-03b: 前端集成 ✅
- [x] EXPORT-03c: 厂商特定格式 ✅
- [x] D-04: 实现批量导出功能 ✅

#### v1.2 可配置仪表盘生产级改造（仿 Zabbix）— ✅ Completed 2026-04-21

**WDG-01: Widget 数据获取机制**
- [x] WDG-01a: 实现 `GetWidgetData` 方法
- [x] WDG-01b: 实现 `GetBatchWidgetData` 批量获取
- [x] WDG-01c: 支持数据转换配置（transform）
- [x] WDG-01d: 支持三种数据源类型（API、WebSocket、Static）

**FE-02: 前端未完成功能**
- [x] FE-02a: 实现 Widget 选择器组件
- [x] FE-02b: 实现仪表盘设置面板
- [x] FE-02c: 实现模板预览功能
- [x] FE-02d: 完善 Widget 编辑/删除交互

**RT-03: 实时数据刷新**
- [x] RT-03a: 实现 WebSocket 连接管理
- [x] RT-03b: 实现基于刷新间隔的轮询机制
- [x] RT-03c: 实现缓存策略
- [x] RT-03d: 前端实时数据订阅和更新

**UX-04: 用户体验优化**
- [x] UX-04a: Widget 拖拽布局优化
- [x] UX-04b: 仪表盘加载状态和错误处理
- [x] UX-04c: 数据为空时的友好提示
- [x] UX-04d: 响应式布局适配

### Out of Scope

- 导入时自动创建不存在的部门或用户 — 超出导入功能职责范围
- 批量修改已有工位的部门/用户关联 — 本期仅关注导入场景
- 工位与用户的多对多关系 — 当前只需一对一关联
- 设备/端口多对多关系 — 当前只需一对一
- 导入时自动创建不存在的设备或端口 — 超出导入功能职责
- 端口与设备的级联验证（确认端口属于指定设备）— 复杂度高，用户未要求
- **仪表盘系统统一** — 保留两套系统：固定监控仪表盘（/monitor/dashboard）和可配置仪表盘（/dashboard-system）
- 复杂图表类型（3D 图表、地图）— 超出本期范围
- 告警规则配置 — 属于监控系统，非仪表盘核心
- Zabbix 告警集成 — 超出本期范围

## Context

### 技术环境
- **后端**: Go 1.24 + Gin + GORM + PostgreSQL
- **前端**: React 19 + TypeScript + Ant Design 6 + Zustand
- **Excel**: excelize v2 用于导入导出
- **现有导入流程**: `excel_service.go` → `excel_config.go`（列映射）→ `reference_resolver.go`（名称→UUID）→ `batch_upserter.go`（批量写入）
- **仪表盘**: 自定义 Widget 系统，支持 StatCard、Chart、Table、List、Progress 类型
- **两套仪表盘**:
  - `/monitor/dashboard` — 固定监控仪表盘（保留不变）
  - `/dashboard-system` — 可配置仪表盘（v1.2 改进目标）
- **批量导出**: ZIP 打包多个 Excel 文件，支持 9 个实体类型

### Current state (2026-06-16)

- **Shipped**: v1.0 through v1.15 (and Phase 22 VDI 22A/22B partial for v1.11) —
  18 phases, 133 plans completed, ≈85% of total plan count.
- **Phases 30 / 31 / 32 / 33 / 34**: unlabeled (no v1.x tag in ROADMAP
  milestones list) but fully shipped. Phase 34 included a full code
  review with 1 critical security finding (sensitiveKeys did not
  match camelCase field names) and 7 atomic fix commits.
- **Codebase size**: 485 Go files (backend), 506 TS/TSX files
  (frontend) per `CLAUDE.md`.
- **Test coverage**: regression_test.go for operlog locks 25 OperType
  constants, 18 mandatory sensitive keywords, Record signature, keyword
  stability. 14 subtests in TestFilterSensitiveParams + 5 edge cases.
- **Build status** (post-Phase 34 review closure): `go build ./...`
  clean, full operlog test suite PASS, system test suite PASS.

### Known technical debt (post-v1.15)

- VDI 22C (账号管理) and 22D (Web Console) — listed in ROADMAP as
  v1.12 sub-phases but not started. Blocked on 22A/22B production
  observability data.
- Phase 25 (VM 数据范围权限) — depends on Phase 22, also not started.
- `operlog_e2e_verify.sh` live-DB section gated on `SKIP_E2E=1` by
  default — should be promoted to a CI gate.
- `regression_test.go:146` — `for i := 0; i < 5; i++` can be
  modernized to `for i := range 5` (Go 1.22+). Cosmetic, single line.
- 103 open items in `gsd-sdk query audit-open` (most are stale
  `root_cause_found` debug sessions from May 2026).

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| 用户关联字段: 在 workstation 表添加 `user_id` 列 | 与现有 `org_id` 模式一致，一对一关联 | ✓ Good |
| 用户匹配方式: 通过 `user_name` 精确匹配 | 用户名唯一且稳定 | ✓ Good |
| 部门匹配方式: 通过 `dept_name` 精确匹配 | 与现有 `reference_resolver` 模式一致 | ✓ Good |
| 匹配失败策略: 留空不阻断 | 用户明确要求可选关联 | ✓ Good |
| Device Reference: `sys_network_device.device_name` | 设备名是用户可见的标识字段 | ✓ Good |
| Port Reference: `sys_device_port_status.interface_name` | 接口名是网络设备端口的唯一标识 | ✓ Good |
| Port 无 DependsOn: 全局查找不限设备 | 用户未要求级联验证 | ✓ Good |
| 配置驱动模式: 只改 excel_config.go + excel_service.go | 最小改动，复用现有管道 | ✓ Good |
| 保留两套仪表盘: 固定 + 可配置 | 用户明确要求保留固定监控仪表盘 | ✓ Good |
| SNMP panic 修复: 三层防护（恢复+并发+验证） | 防御式编程，确保稳定性 | ✓ Good |
| Core 清理: 文档化而非删除 | 大多数字段有外部依赖 | ✓ Good |
| TDD 方法: RED→GREEN→REFACTOR | 确保测试覆盖和安全修复质量 | ✓ Good |
| 批量导出: ZIP 格式打包多个 Excel | 用户需求，简化下载流程 | ✓ Good |
| VDI 集成: 拆分 22A→22D 4 子阶段 (P1) | 渐进式集成降低风险 | ✓ Good (22A/22B done; 22C/22D pending) |
| operlog 共享包: 提取到 `internal/utils/operlog` (P34) | 避免 handler module 间的循环依赖 | ✓ Good |
| OperType 25 常量集 (P34) | 锁定审计语义，不再用 OperTypeOther 兜底 | ✓ Good |
| OperTypeUnlock = 24 新增 (P34 review) | 解锁是独立审计动作 | ✓ Good |
| sensitiveKeys 34 条 camelCase/snake_case 全覆盖 (P34 review) | 修复 substring-search 不匹配 camelCase 字段的 critical bug | ✓ Good (CR-001) |
| 工位设备关联子表格: Ant Design Table expandable (P28) | 复用现有 Table 组件 | ✓ Good |
| Vendor 命令模板硬编码 Go map (v1.19 W1) | 落地为先，后续 phase 抽象 DB | ✓ Good |
| Path C audit↔operlog 关联 (v1.19 W3) | handler 先 INSERT audit → operlog.Record(WithOperParam(audit_ids)); audit.oper_log_id 列保持 NULL；不入 operlog 包 | ✓ Good |
| detached 30min context (v1.19 W2) | batch 入口 `context.WithTimeout(context.Background(), 30*time.Minute)` 规避 `Core.Close()` 30s 截止 | ✓ Good |
| menu_grant_helpers.go 精准授权 (v1.19 W3) | `INSERT INTO sys_role_menu SELECT...WHERE m.menu_name=parent...ON CONFLICT DO NOTHING` 解决 antd 父子联动陷阱 | ✓ Good (memory `migration-grant-new-menu-precision-helper`) |
| BATCH-05 indeterminate Spin MVP (v1.19 W4 D-05) | 诚实进度，不伪造 X/Y；批量同步阻塞无后端推送 | ⚠ Revisit (FUTURE-11 SSE/WS) |
| execSinglePort DRY helper (v1.19 W3) | 5 单端口 handler 公共流程合并；operlog.Record 物理调用点 6→2，语义等价 | ✓ Good |
| eslint-disable placement 教训 (v1.19 W4 Phase 55) | directive 紧贴 `}, []);` 而非 `useEffect(` 开头 | — Pending (Pattern doc) |

## Constraints

- **Tech Stack**: 必须沿用现有架构模式（Handler-Service、opsApi、excel_config）
- **Backward Compatible**: 新字段可选，不影响现有数据和导入流程
- **UUID Foreign Keys**: 关联字段必须是有效 UUID
- **Status Convention**: 遵循 0=正常, 1=停用 的惯例
- **Response Format**: 使用 `response.Success()` / `response.Error()` 包装
- **Widget 数据获取**: 必须支持权限过滤，用户只能访问有权限的 API 端点
- **保留固定仪表盘**: `/monitor/dashboard` 保持不变，仅改进 `/dashboard-system`
- **SNMP 稳定性**: 必须保证高并发场景下不崩溃
- **测试覆盖**: 安全修复必须有单元测试验证

## Evolution

This document evolves at phase transitions and milestone boundaries.

## Validated — v1.0 through v1.15 (summary)

The full Validated section listing all shipped requirements is preserved
in `.planning/REQUIREMENTS.md` history. High-level milestones:

- ✓ v1.0 工位导入部门/用户关联 (2026-04-16) — 2 phases, 7 plans
- ✓ v1.1 信息点导入设备端口关联 (2026-04-16) — 1 phase, 1 plan
- ✓ v1.2 可配置仪表盘生产级改造 (2026-04-21) — 4 phases, 11 plans
- ✓ v1.3 技术债清理 (2026-04-27) — 3 phases, 9 plans
- ✓ v1.4 MAC地址采集优化 (2026-05-09) — 1 phase, 4 plans
- ✓ v1.5 MAC地址历史数据管理 (2026-06-15) — 4 phases, 26 plans (REQ archive: `.planning/milestones/v1.5-REQUIREMENTS.md`)
- ✓ v1.6 API密钥管理系统 (2026-05-19) — 1 phase, 10 plans
- ✓ v1.7 前后端加密配置同步 (2026-05-20) — 1 phase, 6 plans
- ✓ v1.8 登录端点加密增强 (2026-05-21) — 1 phase, 4 plans
- ✓ v1.9 AD域控集成扩展 (2026-05-24) — 2 phases, 11 plans
- ✓ v1.10 网络设备权限修复 (2026-05-24) — 1 phase, 1 plan
- ✓ v1.11 深信服桌面云集成 (VDI 22A+22B partial) (2026-06-02) — 1 phase (22), 6 plans
- ✓ v1.13 资产管理模块 (Phase 26) (2026-06-08) — 1 phase, 6 plans
- ✓ v1.14 全局列自定义 (Phase 27) (2026-06-09) — 1 phase, 1 plan
- ✓ v1.15 工位设备关联 (Phase 28) (2026-06-10) — 1 phase, 4 plans
- ✓ Phase 30 前端性能优化 (unlabeled, 2026-06-13) — 5 plans
- ✓ Phase 31 P0 收尾 (2026-06-13) — 5 plans
- ✓ Phase 32 v1.14 P1/P2 (unlabeled, 2026-06-13) — 7 waves, 20 commits
- ✓ Phase 33 Vercel React Best Practices (unlabeled, 2026-06-13) — 4 waves
- ✓ Phase 34 操作日志全模块集成 + review (2026-06-16) — 11 plans + 7 review-fix commits
- ✓ Phase 39 工位部门物理位置映射 (workstation dept-location alias) (2026-06-25) — 8 plans + UAT 修复 commits
- ✓ Phase 44 R3 IP 例外规则引擎 (v1.17 资产对账) (2026-06-28) — 2 plans, CIDR GiST 引擎 + Layer 3.5 拦截 + 例外 CRUD/命中测试 + cron 过期清理 + 转单 no_workorder 过滤 + 降噪基线 + Excel 导入导出; VERIFICATION 10/10 passed, 1 CR-03 gap-closure fix (baseline 路由权限), 4 硬化债 BLOCKER 跟踪

---
## Current Milestone: v1.20 网络设备 VLAN + 端口绑定 (Network Device VLAN + Port Binding) — ✅ SHIPPED 2026-07-10

**Goal:** 在 v1.19 端口写命令 MVP 基础上扩展 2 个新写命令：① 修改 access VLAN ② IP+MAC+Port 静态绑定。复用 v1.19 vendor 模板 / operlog / 权限 / 批量 / e2e 全套基建，最小化新基建引入。

**Status:** SHIPPED (Phase 56 / 5 plans / 5 waves). 45 commits / 51 files / +10,961/-104 LOC in 2 days.

**Delivered:**
- 3 vendors × 2 actions 全覆盖（Huawei VRP / H3C Comware V7 / Ruijie RGOS 11.4）
- 2 new kebab HTTP 端点: `POST /network/ports/write/set-access-vlan` + `/port-binding`
- 4 sentinel validators (VLAN 1-4094 range / bind op add|remove / IPv4 regex / null-MAC reject)
- PortResult.Extra map 携带 audit after_value
- 2 前端 Modal + 2 networkApi wrapper + types/network.ts 扩展
- 11 new TestE2E_* via scrapligo FileTransport + 10 new fixtures
- vendor-react bundle gzip 774.96 kB (≤776 kB baseline, zero regression)
- Code review clean after fixing 4 CRITICAL (CR-01/04 batch path validator 共享、CR-02 vlan field、CR-03 modal pre-fill)

**Locked decisions (validated):**
- 复用 v1.19 vendor template 模式 + 新增 2 actions (set_access_vlan + port_binding) ✓
- 权限：单一 `network:port:write` 覆盖新 2 actions ✓
- Audit：`sys_port_write_audit.after_value` JSONB 写 PortResult.Extra ✓
- operlog OperType: VLAN → Update (=2); binding add → Create (=1); binding remove → Delete (=3) ✓
- Pre-state check: set_access_vlan 读 PVID NoOp 短路; port_binding 不 NoOp（Pitfall 6） ✓
- MAC 格式锁定：Huawei/H3C = `AA-BB-CC-DD-EE-FF` (per-byte hyphenated); Ruijie = `aabb.ccdd.eeff` (Cisco 3-pair) ✓
- IPv4 regex 仅做 shell-injection 防御，不验证 segment range（RFC 0.0.0.0/255.255.255.255 合法） ✓

**Deferred (acknowledged at close):**
- **12 site-visit UAT items** (Huawei S8700 × 6 + Ruijie RS8607E × 4 + H3C × 2 conditional) — 详见 `.planning/phases/56-vlan-v1-20-1-0-5-plans-initiated-2026-07-09/56-HUMAN-UAT.md`，沿用 v1.19/v1.18 deferral 模式
- **VLAN-04 / BIND-06 / UI-06** (批量端口写) — 显式 defer 到 FUTURE-BATCH-05；`BatchWriteRequest` 已预留 4 字段（VLANID/BindOp/IPAddress/MACAddress）

**Archived:** `.planning/milestones/v1.20-ROADMAP.md` + `.planning/milestones/v1.20-REQUIREMENTS.md`
- **FUTURE-03 剩余部分**: trunk VLAN / hybrid VLAN / 多 VLAN list
- **FUTURE-04~14**: dry-run / 操作历史查看 UI / 批量冲突自动解决 / 设备不可达预检 / Cisco IOS-XE/XR / 自动回滚 / 多用户并发互斥 / HTTP handler e2e / 3-vendor fixture 全覆盖 / 跨固件命令差异 / BATCH-05 实时进度

**设计文档:** `docs/plans/2026-07-09-v1.20.1-design.md` (commit `f799cecc`，412 行)

**用户已确认数据来源:**
- Huawei S8700 `dis port vlan ge 1/0/0` / `dis user-bind static all` (实采样例)
- Ruijie RS8607E `sh int status` / `sh port-security binding` (实采样例)
- H3C: 复用 Huawei 模板结构 (per user direction)

---

*Last updated: 2026-08-20 — v1.26 后端测试覆盖率优秀 (Backend Test Coverage Excellence) milestone started: 12.8% → ≥70% 加权平均,P0/P1 零测试模块全清,CI coverage 阈值 gate + diff coverage ≥80%;4 phases (71-74) 按 quick-260820-bcs 扫描建议拆分。规划输入: `.planning/quick/260820-backend-test-coverage-scan/SUMMARY.md`。Previous: Combined v1.22-v1.25 SHIPPED + ARCHIVED 2026-08-19 (7 phases / 20 plans / 36 items, audit `passed`); Phase 63 前端工具链自动化 SHIPPED 2026-08-20; v1.21 SHIPPED 2026-08-18 (Phases 57-62).*

---

## Previous Milestone Footer (v1.21 close)

*Last updated: 2026-08-13 — Phase 59「可观测性 / 使用日志修复」COMPLETE (2/2 plans, OBSERV-01/02/03 验证 8/8 must-haves passed; 7 commits e0f4611/2dcb041/e4c80fa/c35e675/7643578/10e5bb1/644bc94 + tracking 66a3d34)。v1.21 API Key 认证链修复 milestone 进行中 (Phase 60-61 待执行)。Phase 58 (CONTRACT-01 前后端路由方法对齐) 已 commit 1978935。v1.20 网络设备 VLAN + 端口绑定 SHIPPED 2026-07-10 (Phase 56 / 5 plans / 5 waves). v1.19 网络设备写命令 SHIPPED 2026-07-08 (Phases 50-55, 6 phases / 9 plans, 37/37 MVP requirements).*
