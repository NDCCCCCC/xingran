---
gsd_state_version: 1.0
milestone: v1.18
milestone_name: 网络设备硬件清单 (Device Component Serials)
status: executing
stopped_at: Phase 48 Wave 1 complete (48-01 merged)
last_updated: "2026-07-04T00:52:54.426Z"
last_activity: 2026-07-04 -- Phase 48 Wave 1 (48-01) merged; milestone frontmatter 修正为 v1.18 (对齐 ROADMAP L24; v1.17 已于 2026-07-03 shipped)
progress:
  total_phases: 1
  completed_phases: 0
  total_plans: 3
  completed_plans: 1
  percent: 33
---

# Project State

**Project**: RuoYi-Next 运维管理系统
**Created**: 2026-04-16
**Status**: 🚢 Phase 46/47/48 dropdown governance shipped — branch `worktree-feat-dropdown-governance` pushed, MR pending
**Last activity**: 2026-06-29 - Phase 46-48 Dropdown 反模式全量治理:

  - 4 atomic commits on branch `worktree-feat-dropdown-governance`
  - 7 个后端 dropdown-options 端点 (building/floor/workstation/infoPoint/serverRoom/dedicatedLine/roomDevice)
  - 27 个前端文件重构 (pageSize 1000→50, searchOptions + onSearch debounce)
  - 100+ 文件 ESLint 规则升级 (245 violations → 0)
  - 计划文件: C:\Users\CPIC\.claude\plans\happy-waddling-wand.md (107 行)

**See**: `.planning/milestones/20260616-state-sync-report.md` for the
        reconciliation report that motivated this state update.

## Current Position

Phase: 48 (device-component-serials-planned) — EXECUTING (Wave 1 complete)
Plan: 1/3 complete (48-01 merged to main @ 9157be4a)
Status: Executing Phase 48 — Wave 2 (collectors) next
Last activity: 2026-07-04 -- Phase 48 Wave 1 (48-01) merged; milestone frontmatter 修正 v1.18

## Project Reference

**Core value**:

- 端到端运维可观测与审计能力:每个写操作产生可追溯记录(who/when/what/from-where/before-after-state),敏感字段自动脱敏
- Excel 驱动的批量管理(导入/导出含 dept-name → UUID 自动解析)
- 单一可信 IT 资产系统

**v1.16 目标**: 把 v1.15 累积的 22 个 deferred debug 会话 + 6 个 audit 数据质量问题集中处理,让系统回到"无遗留债务"状态,audit-open 中 debug_sessions < 5

## Phase Status (v1.16)

| Phase | Name | Status | Progress |
|-------|------|--------|----------|
| 40 | Tech-Debt Cleanup | ✅ Complete | 3/3 plans |
| 41 | Debug Session Gap Closure | ✅ Complete | 4/4 plans |
| 13 | Query Layer Trajectory (Gap Closure) | ✅ Complete | 4/4 gap plans (10/10 incl. baseline) |

## Completed Milestones

- ✅ **v1.0 工位导入部门/用户关联** (2026-04-16) — 2 phases, 7 plans
- ✅ **v1.1 信息点导入设备端口关联** (2026-04-16) — 1 phase, 1 plan
- ✅ **v1.2 可配置仪表盘生产级改造** (2026-04-21) — 4 phases, 11 plans
- ✅ **v1.3 技术债清理** (2026-04-27) — 3 phases, 9 plans
- ✅ **v1.4 MAC地址采集优化** (2026-05-09) — 1 phase, 4 plans
- ✅ **v1.5 MAC地址历史数据管理** (2026-06-15) — 4 phases, 26 plans
- ✅ **v1.6 API密钥管理系统** (2026-05-19) — 1 phase, 10 plans
- ✅ **v1.7 前后端加密配置同步** (2026-05-20) — 1 phase, 6 plans
- ✅ **v1.8 登录端点加密增强** (2026-05-21) — 1 phase, 4 plans
- ✅ **v1.9 AD域控集成扩展** (2026-05-24) — 2 phases, 11 plans
- ✅ **v1.10 网络设备权限修复** (2026-05-24) — 1 phase, 1 plan
- ✅ **v1.11 AD组自动同步系统** (2026-05-26) — 1 phase (23), 18 plans
- ✅ **v1.13 资产管理模块** (2026-06-08) — 1 phase (26), 6 plans
- ✅ **v1.14 全局列自定义** (2026-06-09) — 1 phase (27), 1 plan
- ✅ **v1.15 工位设备关联** (2026-06-10) — 1 phase (28), 4 plans
- ✅ **Phase 39 工位部门物理位置映射** (2026-06-25) — 1 phase (39), 8 plans
- ✅ **v1.16 技术债清理 (Tech-Debt Cleanup)** (2026-06-26) — 2 phases (40 + 41), 8 plans (39 commits, audit-open 29→0, verify_phase40.sh [ALL PASS])

## Session History

| Date | Action | Phase |
|------|--------|-------|
| 2026-04-16 | Project initialized | — |
| ... (Phases 1-39 shipped, see full history in archive) | | |
| 2026-06-22 | Phase 37 完成 (前端部门选择组件统一收敛) - 6/6 plans | 37 |
| 2026-06-23 | Phase 38 Plan 01 完成 (Wave 1 前置 DI 脚手架) - 8 服务 struct 注入 AccountPool | 38 |
| 2026-06-25 | Milestone v1.16 started — ROADMAP drafted (Phase 40: tech-debt-cleanup) | 40 |
| 2026-06-26 | Phase 41 Plan 03 complete — 10 Cohort B backend/auth sessions closed (1 fix(41) + 9 docs(41), 2 skip_audit) | 41 |
| 2026-06-26 | Phase 41 Plan 02 complete — 6 Cohort B frontend sessions closed (2 fix(41) + 4 docs(41), 4 skip_audit); 1 HELD for browser verify | 41 |
| 2026-06-26 | Phase 41 Plan 04 Task 2 complete — STATE/ROADMAP/REQUIREMENTS updated, v1.16 milestone closed (audit-open debug_sessions=0, verify_phase40.sh [ALL PASS], validator 100%); user approved + 暂不 push | 41 |
| 2026-06-26 | Phase 13 gap closure complete — 4 plans (13-07/08/09/10) shipped, 5/5 code gaps fixed (CR-01/CR-02/CR-03/W4-vendor/W5-echarts), VERIFICATION.md status:passed score:18/18, 8 atomic commits | 13 |

## Quick Tasks Completed (recent)

| # | Description | Date | Commit | Directory |
|---|-------------|------|--------|-----------|
| 260623-batch-sync | 用户导入后 AD 批量同步（BatchSyncUsersToAD）：复用单 LDAP 连接 + errgroup 信号量 | 2026-06-23 | - | [260623-batch-sync-optimize](./quick/20260623-batch-sync-optimize/) |
| 260622-lv9 | 系统用户 Excel 导入功能 + 性别默认值改保密 + 导入后触发 AD 域同步 | 2026-06-22 | 5d872a5,cc2a77d,7d40b32,4758d36 | [260622-lv9-xlsx-email](./quick/260622-lv9-xlsx-email/) |
| 260622-u8n | 从 sys_dept.leader 批量同步用户 AD manager 属性 | 2026-06-22 | - | [260622-u8n-ad-manager-leader-ad](./quick/260622-u8n-ad-manager-leader-ad/) |
| 260626-dai | 工位 Excel 导出补充所属部门/人员名称 + 设备序列号/名称/型号（新增 WorkstationQueryBuilder，镜像 List JOIN） | 2026-06-26 | 212f73fb | [260626-dai-workstation-export-fields](./quick/260626-dai-workstation-export-fields/) |
| 260626-lbd | 工位管理主表添加"主设备序列号"列（后端子查询严格 is_primary=true + 前端列，设主设备才显示） | 2026-06-26 | - | [260626-workstation-primary-device-serial](./quick/260626-workstation-primary-device-serial/) |
| 260630-sort | 信息点查询等 16 个前端列表页补齐服务端排序(useServerSort + useTableManager.sorterMetas),含 1 处 params-drop bug 修复 + 5 处后端 ApplySort 白名单补全 | 2026-06-30 | 0cd7ddb0..bb8f4dc9 (18 commits) | [260630-server-sort-promote-16-pages](./quick/20260630-server-sort-promote-16-pages/) |
| 260630-mac-tree | mac 地址页接口列排序 + 左侧 DeptSidebar + 部门联动(后端 deptId JOIN 一处过滤,弃用前端 deviceIds 中间层;含 migration_181 验证修复) | 2026-06-30 | 2f2894d2 | [20260630-mac-page-dept-tree](./quick/20260630-mac-page-dept-tree/) |
| 260630-startup-flow | 启动流程分类文档化: docs/启动流程清单.md(7 阶段 × 3 分类:🔴每次必跑/🟢一次性/🟡混合幂等),含核心.Init() 19+ 步骤 + AutoMigrate/InitData 全表 + 风险清单 + setup/start 拆分建议 | 2026-06-30 | - | [20260630-startup-flow-classify](./quick/20260630-startup-flow-classify/) |
| 260630-skip-setup | 启动流程优化 - 加 SERVER_SKIP_SETUP 开关: config.ServerConfig.SkipSetup + SERVER_SKIP_SETUP 环境变量 + core.Init() 条件化跳过 InitData/默认角色菜单 + cmd/main.go 条件化跳过 MAC retention/OUI seed;docs/启动流程清单.md §6 同步 | 2026-06-30 | (待提交) | [20260630-skip-setup-impl](./quick/20260630-skip-setup-impl/) |
| 260701-iface-sort | 端口采集接口名称列数值排序(速率前缀 + 板卡/子卡/接口号;PG regexp_matches int 数组,dialect-aware 降级 sqlite);修正 GE0/10<GE0/2 字典序;query.go interfaceNameSortExpr + GetList 特殊分支,4 新测试全绿 | 2026-07-01 | (待提交) | [260701-iface-sort](./quick/260701-iface-sort/) |
| 260702-148 | 对账看板"例外规则管理"链接跳仪表盘(三轮: ①d033e510 href 改错(component当路由) ②607da270 migration_195 补种菜单(菜单确未seed) ③a63e1fdf href 改回真实路由 /assets/exception-rules;真根因双重: 菜单未seed + component≠route path;DB实查确认) | 2026-07-02 | d033e510,607da270,a63e1fdf | [260702-148-fix-dashboard-exception-rules-link](./quick/260702-148-fix-dashboard-exception-rules-link/) |
| 260703-dkc | 重写资产列表 columns 数组 — 4 假列全删(错 key/title/dataIndex/render: deptName key 占 deviceTypeName 字符串当 status 数字 Tag 渲染、mac1 key 占 nbfStatus 渲染日期、remark key 占 machineUptime)+ 15 visible 列补齐(deviceModelName/deptName/machineIp/mac1/assetLocation/buildingName/floorName/signOrgnoName/nowUserName/nowUserDeptCode/status/nbfStatus/drawingDate/machineUptime/lastInventoryDate)+ defaultAssetColumns order=10 删(后端无 recipientName;真领取人是 deviceUserName);type-check 0 / lint 8 pre-existing unchanged | 2026-07-03 | 97b49ea7 | [260703-dkc-ruoyi-react-frontend-src-pages-operation](./quick/260703-dkc-ruoyi-react-frontend-src-pages-operation/) |
| 260703-eaj | 实施 B 方案(前后端 schema 单一真理源): columnsSchema.ts 新建 + index.tsx 改 import + sync-columns-schema.mjs(bracket-match + new Function 解析)+ package.json 加 sync-columns-schema + prebuild 钩子 + 后端 //go:embed asset_columns_schema.json 替换 43 列硬编码 → 3 原子提交(43ecb291/1f2e4d21/69ef4161),type-check 0 / sync 51 列 / go build 0 / go test 1 pre-existing;user/role/dept 同模式待推广 | 2026-07-03 | 43ecb291,1f2e4d21,69ef4161 | [260703-eaj-b-schema-embed-ruoyi-react-frontend-src-](./quick/260703-eaj-b-schema-embed-ruoyi-react-frontend-src-/) |

## Accumulated Context

### v1.16 Tech-Debt Cleanup 范围

**22 个 deferred debug 会话** (frontmatter `deferred_to: v1.16-tech-debt`):

- 16 个 `root_cause_found`(AD 域控 6 + VDI 2 + infopoint/captcha/permission/api-key 等 8)
- 5 个 `awaiting_human_verify`(workstation-device-build-errors、workstation-expand-device-load-empty、reset-theme-hardcoded-colors、user-filter-lost-after-edit、vite-hmr-token-blank-page)
- 1 个 `root-cause-found`(apikey-route-path-duplication,连字符变体状态值)

**6 个 audit 数据质量问题**:

- `login-400-bad-request` / `ops-asset-constraint-uni-ops-asset-devicesn-not-exist` — 缺标准 YAML frontmatter
- `knowledge-base` — 非 debug 会话,需移出 audit 范围
- `info-point-type-null-import` / `sys-mac-filter-rules-relation-does-not-exist` — 实际已 resolved,frontmatter 需统一

**验收指标**:

- `gsd-sdk query audit-open` → `debug_sessions < 5`(从当前 57 大幅下降)
- `.planning/debug/*.md` → 100% 标准 YAML frontmatter

### Roadmap Evolution

- ✅ v1.0–v1.15 完成 (Phases 1-39, 137 plans + 8 gap-closure)
- 🚧 v1.16 规划中: Tech-Debt Cleanup (Phase 40, 5 TECH requirements)

### Known Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| 22 个 debug 会话修复可能引入回归 | Medium | 按 root_cause_found / awaiting_human_verify 分批,每批独立 commit,逐步验证 |
| 5 个 awaiting_human_verify 需在 dev 环境复现 | Low | dev 环境启动后逐个 case 复现,验证修复有效 |
| `knowledge-base` 改名可能影响其他工具 | Low | 优先选择 `skip_audit: true` 注释,避免破坏引用 |

## Next Steps

**当前选项**:

1. **v1.16 milestone 已闭环** —— Phase 40 + Phase 41 全部完成,8/8 plans,audit-open 29→0,TECH-05+TECH-06 双达成。
2. **推送决策** —— 用户选择"暂不 push,本地完成"。您可在合适时机手动执行 `git push origin main`(39 个 phase-41 commit + 已有 phase-40 commit 在 main 分支)。
3. **下一 milestone 规划** —— v1.17 候选范围(可后续规划):
   - VDI 22C / 22D(依赖 v1.11 生产稳定性观察数据)
   - DICT-29 (sys_dict 重构,纯样式)
   - 新功能(mobile app / AI features) — 待业务决策
   - 残余 deferred debug session(若有新增)

**推荐**: v1.16 推送前先做一次完整 UAT,确认 14 个 re-verify-flip + 2 real-fix + 13 wontfix 的 29 个 session 处置与生产表现一致后,再 git push。

## Performance Metrics

| Phase Plan | Duration | Tasks | Files |
|------------|----------|-------|-------|
| Phase 43-r2 P01 | ~12 min | 2 tasks | 6 files |
| Phase 43-r2 P01 | 720 | 2 tasks | 6 files |
| Phase 43 P03 | 9 | 2 tasks | 7 files |
| Phase 44 P01 | 50 | 8 tasks | 20 files |
| Phase 44 P02 | 25 | 4 tasks | 14 files |

## Session Continuity

Last session: 2026-07-03T07:57:33.163Z
Stopped at: Phase 46 context gathered
Resume file: .planning/phases/46-r5/46-CONTEXT.md

**Milestone status:** v1.16 技术债清理 — milestone_complete (2026-06-26), 2 phases (40+41), 8 plans, 39 phase-41 commits + 3 phase-40 plans. **Phase 13 retroactive gap closure** — 4/4 plans (13-07/08/09/10), 8 atomic commits on main, 5/5 code gaps fixed. v1.17 规划待启动。
