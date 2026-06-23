---
gsd_state_version: 1.0
milestone: v1.15
milestone_name: 工位设备关联 (Phase 28)
status: milestone_complete
stopped_at: Milestone complete (Phase 38 was final phase)
last_updated: 2026-06-23T04:49:37.509Z
progress:
  total_phases: 16
  completed_phases: 10
  total_plans: 80
  completed_plans: 144
  percent: 63
---

# Project State

**Project**: RuoYi-Next 运维管理系统
**Created**: 2026-04-16
**Status**: 🚧 IDLE — Phase 34 review closure complete, awaiting v1.16 scope
**Last activity**: 2026-06-23 - Quick task 260623-batch-sync 用户导入后 AD 批量同步优化（复用单连接+errgroup信号量，2274 用户 ~10分钟→~10秒）

                + 导入后触发 AD 域同步 (commits 5d872a5, cc2a77d, 7d40b32, 4758d36)

**See**: `.planning/milestones/20260616-state-sync-report.md` for the
        reconciliation report that motivated this state update.

## Current Position

Phase: 38
Plan: Not started
**Latest shipped milestone**: v1.15 工位设备关联 (Phase 28, 2026-06-10)
**Latest shipped phase**: Phase 34 操作日志全模块集成 + review (2026-06-16)
**Most recent commit**: 3084a04 refactor(38-04): migration_164 放宽列约束 + 幂等补迁账号池 (Task 2)

**Shipped since last state file update (v1.15 was the milestone label
in PROJECT.md before this sync)**:

- Phase 30 (frontend perf) — 5/5 plans, 2026-06-13
- Phase 31 (P0 close: connection_pool lifecycle + ConfigKey API) — 5/5 plans
- Phase 32 (v1.14 P1/P2 重构) — 7 waves, 20 commits
- Phase 33 (Vercel React Best Practices) — 4 waves, 25/25 perf defects fixed
- Phase 34 (operlog full coverage) — 10 plans + gap-closure + 4 review commits
  + 1 critical fix (sensitiveKeys 17→34 expansion to cover camelCase
  field names that the substring-search needle did not match)

**Audit**: 103 open items in `gsd-sdk query audit-open` (45 debug
sessions, 51 quick tasks, 4 UAT gaps, 3 verification gaps). Most
debug sessions are `root_cause_found` from May 2026 — bulk-defer
candidate for v1.16.

**Next step**: `git log` shows main is at `5c2fc95`, build is clean,
operlog test suite green. v1.16 scope is undecided. Recommended
candidates (from state-sync report):

1. Triage the 103 audit items
2. Decide on VDI 22C/22D vs new domain
3. Decide on Phase 34 operlog follow-up (CI gate, property test)

Run `/gsd:new-milestone` to formally start v1.16.

## Project Reference

**Core value**: 

- Excel 导入时自动关联实体，提升批量管理效率
- 可配置仪表盘系统提供监控可视化能力
- AD 域控集成实现组织架构双向同步
- API 密钥管理系统提供安全的第三方访问
- 完整的国密加密体系保护数据传输
- 网络设备细粒度权限控制保障安全
- 深信服 VDI 集成实现虚拟桌面统一管理

## Phase Status

| Phase | Name | Status | Progress |
|-------|------|--------|----------|
| 1-11 | v1.0-v1.4 基础功能 | ✅ Completed | 27/27 plans |
| 12 | 数据模型与采集集成 | ✅ Completed | 5/5 plans |
| 13 | 查询层与轨迹 | ✅ Completed | 6/6 plans |
| 14 | 前端与用户体验 | 📋 Planned | 0/5 plans |
| 15 | 性能优化 | ✅ Completed | 5/5 plans |
| 16 | API 密钥管理 | ✅ Completed | 10/10 plans |
| 17 | 前后端加密配置同步 | ✅ Completed | 6/6 plans |
| 18 | 登录端点加密增强 | ✅ Completed | 4/4 plans |
| 19 | AD域控账号登录 | ✅ Completed | 6/6 plans |
| 20 | AD域控OU映射 | ✅ Completed | 5/5 plans |
| 21 | 网络设备权限修复 | ✅ Completed | 1/1 plan |
| 22A | VDI 基础集成 | ✅ Completed | 5/5 plans |
| 22B | VM Agent 服务 | ✅ Completed | 1/1 plan |
| 22C | 账号管理与密码 | 📋 Planned | 0/2 plans |
| 22D | Web Console 监控 | 📋 Planned | 0/2 plans |
| 23 | AD组自动同步 | ⚠️ Partial | 6/7 plans (23-13未实现) |
| 24 | UUID类型一致性优化 | ✅ Completed | 2/2 plans |
| 25 | 虚拟机数据范围权限配置 | 📋 Planned | 0/4 plans |
| 26 | 资产管理模块 | ✅ Completed | 6/6 plans |
| 27 | 全局列自定义显示功能 | ✅ Completed | 1/1 plan |
| 28 | 工位设备关联子表格 | ✅ Completed | 4/4 plans |
| 29 | sys_dict 重构 | 📋 Planned | 0/0 plans |
| 30 | 前端性能优化 | ✅ Completed | 5/5 plans |
| 31 | P0 收尾:连接池 lifecycle + Config API 重设计 | ✅ Completed | 5/5 plans |
| 32 | v1.14 P1 重构与 P2 架构优化 | 📋 Planned | 0/0 plans |
| 33 | 前端 Vercel React Best Practices 修复 | ✅ Completed | 4/4 plans |
| 34 | 操作日志全模块集成 | ✅ Completed | 10/10 plans |

## Completed Milestones

- ✅ **v1.0 工位导入部门/用户关联** (2026-04-16) — 2 phases, 7 plans
- ✅ **v1.1 信息点导入设备端口关联** (2026-04-16) — 1 phase, 1 plan
- ✅ **v1.2 可配置仪表盘生产级改造** (2026-04-21) — 4 phases, 11 plans
- ✅ **v1.3 技术债清理** (2026-04-27) — 3 phases, 9 plans
- ✅ **v1.4 MAC地址采集优化** (2026-05-09) — 1 phase, 4 plans
- ✅ **v1.6 API密钥管理系统** (2026-05-19) — 1 phase, 10 plans
- ✅ **v1.7 前后端加密配置同步** (2026-05-20) — 1 phase, 6 plans
- ✅ **v1.8 登录端点加密增强** (2026-05-21) — 1 phase, 4 plans
- ✅ **v1.9 AD域控集成扩展** (2026-05-24) — 2 phases, 11 plans
- ✅ **v1.10 网络设备权限修复** (2026-05-24) — 1 phase, 1 plan
- ✅ **v1.11 AD组自动同步系统** (2026-05-26) — 1 phase (23), 18 plans
- ✅ **v1.5 MAC地址历史数据管理** (2026-06-15) — 4 phases (12-15), 26 plans

## Session History

| Date | Action | Phase |
|------|--------|-------|
| 2026-04-16 | Project initialized | — |
| 2026-04-16 | Milestone v1.1 完成 | — |
| 2026-04-21 | Milestone v1.2 完成 | — |
| 2026-04-27 | Milestone v1.3 完成 | — |
| 2026-05-09 | Milestone v1.4 完成 | 11 |
| 2026-05-11 | Phase 12 完成 | 12 |
| 2026-05-19 | Phase 16 完成 (API密钥管理) | 16 |
| 2026-05-20 | Phase 17 完成 (加密配置同步) | 17 |
| 2026-05-21 | Phase 18 完成 (登录端点加密) | 18 |
| 2026-05-21 | Phase 19 完成 (AD账号登录) | 19 |
| 2026-05-24 | Phase 20 完成 (AD OU映射) | 20 |
| 2026-05-24 | Phase 21 完成 (权限修复) | 21 |
| 2026-05-25 | 项目状态文档更新 | — |
| 2026-05-25 | Phase 22-01 完成 (VDI数据模型与配置基础) | 22A |
| 2026-05-25 | Phase 22-02 完成 (VDI API客户端与认证) | 22A |
| 2026-05-25 | Phase 22-03 完成 (VDI服务层实现) | 22A |
| 2026-05-25 | Phase 22-04 完成 (VDI后端API层) | 22A |
| 2026-05-25 | Phase 22-05 完成 (VDI前端UI实现) | 22A |
| 2026-06-02 | Phase 22A 完成 (VDI基础集成) - 5个计划全部完成 | 22A |
| 2026-06-02 | Phase 22B 完成 (VM Agent 服务) - 1个计划全部完成 | 22B |
| 2026-06-02 | Quick task完成: VMP服务器API验证脚本 | — |
| 2026-05-26 | Phase 23 部分完成 (AD组自动同步) - 6/7计划，23-13未实现 | 23 |
| 2026-06-05 | Phase 23 验证更新：15分钟组同步Cron任务未在代码库中找到 | 23 |
| 2026-06-05 | Quick task 260605-iq7: AD域组成员同步数据源分析完成 | — |
| 2026-06-09 | Quick task完成: 域控用户页面多选框显示修复 | — |
| 2026-06-09 | Phase 27 完成 (全局列自定义显示功能) | 27 |
| 2026-06-10 | Phase 28 完成 (工位设备关联子表格) - 7个计划全部完成 | 28 |
| 2026-06-13 | Phase 30 Plan 01 完成 (Bundle 基础设施) - Wave 1 of 4 | 30 |
| 2026-06-13 | Phase 30 Plan 02 完成 (重库按需加载) - Wave 2 of 4 | 30 |
| 2026-06-13 | Phase 30 Plan 03 完成 (查询层 React Query 推广) - Wave 3 of 4 | 30 |
| 2026-06-13 | Phase 30 Plan 04 完成 (渲染层 memo/虚拟滚动/ESLint) - Wave 4 of 4 ✅ Phase 30 COMPLETE | 30 |
| 2026-06-13 | Phase 30 Plan 05 完成 (缺口闭合) - Wave 5 of 4 ✅ Phase 30 COMPLETE (25/25) | 30 |
| 2026-06-13 | Phase 31 完成 (P0 收尾) - F-14 connection_pool refCount + F-17 ConfigKey API,5 plan + UAT 2 pass | 31 |
| 2026-06-13 | 20260612 后端审查报告 P0 100% 覆盖 (20/22 完整 + 2/22 部分) - 58 commit | — |
| 2026-06-13 | Phase 32 Wave 1 完成 (P1 安全加固快赢) - P1-S2/S3/S4/S7 ✅ | 32 |
| 2026-06-13 | Phase 32 Wave 2 完成 (P1 密码强度迁移) - P1-S1/S5/S6 ✅ | 32 |
| 2026-06-13 | Phase 32 Wave 3 完成 (P1 并发与一致性) - P1-C1~C6 ✅ | 32 |
| 2026-06-13 | Phase 32 Wave 4 完成 (P1 业务逻辑收尾) - P1-B1/B2 ✅ | 32 |
| 2026-06-13 | Phase 32 Wave 5 完成 (P2 核心重构) - P2-A1/A2/A3/A5 ✅ | 32 |
| 2026-06-13 | Phase 32 Wave 6 完成 (P2 迁移与子进程) - P2-A4/A7/A8 ✅ | 32 |
| 2026-06-13 | Phase 32 Wave 7 完成 (P2 测试覆盖) - P2-A6 ✅ — PHASE 32 FULLY COMPLETE (7/7 waves, 20 commits) | 32 |
| 2026-06-13 | Phase 33 完成 (前端 Vercel React Best Practices) - 25/25 性能缺陷修复 (Waves 1-4) ✅ | 33 |
| 2026-06-13 | Phase 13 完成 (查询层与轨迹) - 6/6 计划全部完成 ✅ | 13 |
| 2026-06-15 | Phase 34 Plan 01 完成 (操作日志共享 Helper 基础设施) - Wave 1 of 7, 共享 operlog 包 + 委托层 + 24 OperType + 17 关键词过滤器 ✅ | 34 |
| 2026-06-15 | Phase 34 Plan 02 完成 (系统核心模块操作日志全覆盖) - Wave 2 of 7, 6 handler 31 写端点埋点 + WithCore() 注入 + SM4 兼容烟雾测试 ✅ | 34 |
| 2026-06-15 | Phase 34 Plan 03 完成 (系统外围模块操作日志) - Wave 2 ext, 4 handler 23 写端点 (api_config/dict/config/notice) ✅ | 34 |
| 2026-06-15 | Phase 34 Plan 04 完成 (operations 模块操作日志) - Wave 3, 10 handler 56 写端点 (building/floor/workstation/excel/...) ✅ | 34 |
| 2026-06-15 | Phase 34 Plan 05 完成 (network 模块操作日志) - Wave 4, 10 handler 44 写端点 (device/credential/template/discovery/...) + RecordWithBody 脱敏 ✅ | 34 |
| 2026-06-15 | Phase 34 Plan 06 完成 (跨域模块操作日志 Wave 5) - 5 个模块 59 写端点 (vdi/workorder/duty/knowledge/scheduler) ✅ | 34 |
| 2026-06-16 | Phase 34 Plan 07 完成 (跨模块操作日志 Wave 6) - monitor+rpa+agent 11 handler 45 写端点 (cache/login_log/oper_log/server/agent/task/credential/execution/ai/flow/worker) ✅ | 34 |
| 2026-06-16 | Phase 34 Plan 08 完成 (system 子模块操作日志 Wave 7 — FINAL Wave 2 plan) - 8 handler 31 写端点 (dashboard/column_config/notification_config/notice_user/ou_group_mapping/ou_mapping/ad_domain_user_sync/ad_dept_sync) ✅ | 34 |
| 2026-06-16 | Phase 34 Plan 09 完成 (e2e 验证 harness) - scripts/operlog_e2e_verify.sh 32 抽样端点 + Go 包装 + coverage_test.go 单元测试 ✅ | 34 |
| 2026-06-16 | Phase 34 Plan 10 完成 (文档 + 回归测试) - CLAUDE.md + docs/开发规范.md 新增强制操作日志约定 + regression_test.go 4 测试锁定公共 API ✅ **PHASE 34 CLOSED (10/10)** | 34 |
| 2026-06-16 | Phase 34 Gap-closure 完成 (34-gap) - 修复 34-VERIFICATION 发现的 25 个遗漏写端点 (6 handler 文件) + e2e 差异扫描加固 (阈值 250→290 + handler-vs-operlog 全集校验) + user_unlock 合规审计 ✅ | 34 |
| 2026-06-22 | Phase 37 Plan 01 完成 (批 0 公共层收敛) - deptUtils 新增 toFullPathTree/toShortNameDataNode 语义区分转换函数 + DepartmentTreeSelect 删本地 Department 接口与 convertDeptTreeData (改引用 SimpleDept + toFullPathTree, 保持受控) + DeptTree 删内部 post fetch/transformToTreeData (改消费 useDeptTree, 搜索/展开逻辑保留); 3 task commit a859975/a52bc36/79cc349; type-check + build 通过; D-LOCKED 决策全部尊重 | 37 |
| 2026-06-22 | Phase 37 Plan 02 完成 (批 1 system 模块迁移 user+notice) - useUserData 去 fetch (post /system/departments/tree) 改消费 useDeptTree + 删本地 Department 接口 (改 SimpleDept) + utils.tsx 删 convertDeptTreeData (实测短名, 归 toShortNameDataNode, 保留 renderDeptTreeOptions Select 专用渲染) + index.tsx 清理 loadDepartments 调用链与死 import + useTargetSelector 删 GET fetch + convertTree 改 useDeptTree+useMemo+toShortNameDataNode + Target 接口保留 (roles/users 子树依赖); 2 task commit 76973d6/86b3231; type-check + build 通过; role/dept/AD 排除边界零触碰 | 37 |
| 2026-06-22 | Phase 37 Plan 04 完成 (批 3 network 模块迁移 devices) - useDeviceData 删 departments useState + loadDepartments useCallback (含 post /system/departments/tree + handleApiError 兜底) 改消费 useDeptTree; 返回对象保留 departments 字段删除 loadDepartments; 本地 Department 类型改 ReturnType<typeof useDeptTree>['data'] 类型透传; index.tsx 删 loadDepartments 解构与 openModal/openQuickCreateModal 两处调用点; 1 task commit 669b25b; type-check 通过; devices/ 全目录 loadDepartments 与 /system/departments/tree 清零; 最简单批次无转换函数无双向语义 | 37 |
| 2026-06-22 | Phase 37 Plan 05 完成 (批 4 duty+workorder 模块迁移 + workorderApi 类型去重) - duty/pools 删 fetchDepts+depts useState+getDeptTree import+useEffect 触发改 useDeptTree (DepartmentTreeSelect 受控模式保持, form 筛选 + edit modal 两处 departments={depts} 由 hook 喂入); useWorkOrderData 同步删 fetchDepts+depts useState+return 字段+getDeptTree import 改 useDeptTree; workorderApi.SimpleDept 本地 interface 改 export type { SimpleDept } from ./dutyApi (T-37-09 mitigate: 外部 import 路径不变); workorderApi.getDeptTree 副本删除 (唯一消费方 useWorkOrderData 已迁移); 2 task commit ccbd504/0cde4f3; type-check 通过; SimpleDept 全项目定义数从 2 (dutyApi+workorderApi) 收敛为 1 (仅 dutyApi canonical 锚点); duty/workorder 目录 /system/departments/tree 清零 | 37 |
| 2026-06-22 | Phase 37 Plan 06 完成 (批 5 收尾 TargetSelector + Phase 37 整体验收) - TargetSelector 删 GET 自 fetch + deptTree useState + useEffect dept 分支, 改消费 useDeptTree + useMemo(toShortNameDataNode(rawDept)) 派生 antd Tree DataNode 形状 (与 37-02 useTargetSelector 完全一致); types/notice.ts DeptTreeNode 接口 + 本地 Department alias 迁移后变孤儿 Rule 1 删除 (grep 确认 0 消费方); Task 2 全量 build 验收暴露并修复 37-05 latent bug (workorderApi.SimpleDept re-export 不引入本地 binding, tsc -b 报 TS2304, 顶部补 import type 修复, Rule 1 auto-fix bug); 2 task commit 214c615/8309947; **Phase 37 整体验收达标** (6/6 plans): 全量 grep 非排除项命中=0 (dutyApi canonical + dept 管理页 + role tree-select 三处合法排除), SimpleDept 唯一定义 (dutyApi:281), build + type-check 双通过; 10 个迁移点 UI 行为核对全部与迁移前一致 | 37 |
| 2026-06-23 | Phase 38 Plan 01 完成 (Wave 1 前置 DI 脚手架) - 为 8 个 addomain 服务 struct 注入 pool AccountPool 字段 + NewADDomainService(db,pool,cipher...) 签名 + router/scheduler caller 透传同一 AccountPool 实例 (Pitfall 4 缓解); 业务逻辑零改动 (22 处 NewLDAPClient(config) 数量不变, 留待 38-02); SHA-5 严格遵守 updateUserAttributeFn 测试钩子保留 (7 个 TestSyncManagersToAD_* 全绿); 2 task commit abdd3e0/21ebb33; 2 Rule 3 auto-fix (测试 caller + user_router/dept_sync_tasks 额外 caller 适配新签名); go build ./... + addomain 全量测试通过 | 38 |

## Quick Tasks Completed

| # | Description | Date | Commit | Directory |
|---|-------------|------|--------|-----------|
| 260623-batch-sync | 用户导入后 AD 批量同步（BatchSyncUsersToAD）：复用单 LDAP 连接 + errgroup 信号量(MaxConcurrentADSync=3)，2274 用户 ~10 分钟→~10 秒 | 2026-06-23 | - | [260623-batch-sync-optimize](./quick/20260623-batch-sync-optimize/) |
| 260622-u8n | 从 sys_dept.leader 批量同步用户 AD manager 属性（祖先递归找 leader + 复用单 LDAP 连接 + 信号量并发，POST /system/users/sync-managers） | 2026-06-22 | - | [260622-u8n-ad-manager-leader-ad](./quick/260622-u8n-ad-manager-leader-ad/) |
| 260614-rrk | 工位编辑模态框：所属楼层 Cascader + 所属用户 Select 显示名称/账号（昵称）而非 UUID | 2026-06-14 | b9f8894 | [260614-rrk-id](./quick/260614-rrk-id/) |
| 260614-dpz | 工位设置主设备时按序列号合并资产+域控数据再同步 | 2026-06-14 | dcfd552 | [260614-dpz-workstation-main-device-merge](./quick/260614-dpz-workstation-main-device-merge/) |
| 260612-d2x | 工位管理页面子表格设备信息添加字段ip地址 | 2026-06-12 | 8ad0f4a | [260612-d2x-ip](./quick/260612-d2x-ip/) |
| 260609-rowselect | 修复域控用户页面多选框显示问题 | 2026-06-09 | 7a5b095 | [20260609-ad-user-rowselection-fix](./quick/20260609-ad-user-rowselection-fix/) |
| 260609-009 | 资产列表页面添加类型筛选组件 | 2026-06-09 | e3983df | [260609-009-asset-type-filter](./quick/260609-009-asset-type-filter/) |
| 260609-columns | 添加资产列表5个显示字段：部门编码、归属机构、接收日期、最后上线、盘点日期 | 2026-06-09 | cd62637 | [260609-asset-list-columns-add](./quick/260609-asset-list-columns-add/) |
| 260608-mdsl | 优化监控仪表盘统计组件布局：移除 Progress，调整固定 span=6 | 2026-06-08 | - | [260608-monitor-dashboard-stats-layout](./quick/260608-monitor-dashboard-stats-layout/) |
| 260608-ljx | 修复 migration_146_add_asset_menu_permissions.go 编译错误 | 2026-06-08 | - | [260608-ljx-migration-146-add-asset-menu-permissions](./quick/260608-ljx-migration-146-add-asset-menu-permissions/) |
| 260605-irs | Plan 22b-02 遗留任务：CallBackend 重试逻辑、401/403 令牌刷新、handler 错误消息清理 | 2026-06-05 | 3ea086f | [260605-irs-plan-22b-02-callbackend-401-403-handler](./quick/260605-irs-plan-22b-02-callbackend-401-403-handler/) |
| 260605-iq7 | 检查AD域组成员同步的数据源是否可以从系统用户替换成AD域控用户 | 2026-06-05 | d90371c | [260605-iq7-ad-ad-1-ad-2-3-ad-4](./quick/260605-iq7-ad-ad-1-ad-2-3-ad-4/) |
| 260520-jrp | 参数管理页面添加请求加密开关 | 2026-05-20 | a0cddc9 | quick/ |
| 260521-m45 | 清理所有空文件夹 | 2026-05-21 | d9c1b5e | quick/ |
| 260521-org | 整理 scripts、test、tests 目录 | 2026-05-21 | 2f8391b | quick/ |
| 260521-bel | 创建嵌入式前端构建脚本 | 2026-05-21 | - | quick/ |
| 260525-vdi | 添加 VDI 菜单到 sys_menu 表 | 2026-05-25 | 8625031 | quick/ |
| 260525-lfw | LDAP 客户端连接代码重构 | 2026-05-25 | - | quick/ |
| 260525-w22 | VDI配置页3个Bug修复(500错误/输入警告/弃用警告) | 2026-05-25 | aa992fc | [260525-w22-vdi-1-500-2-input-suffix-3-direction-ori](./quick/260525-w22-vdi-1-500-2-input-suffix-3-direction-ori/) |
| 260613-vp5 | 工位管理页面：还原"单击该列任意位置自动展开/关闭子表"行为，并修复点击按钮时事件冒泡 | 2026-06-13 | 2c407c4 | [260613-vp5-workstation-expand-fix](./quick/260613-vp5-workstation-expand-fix/) |
| 260526-nij | 用户管理页面部门路径显示修复 | 2026-05-26 | - | [260526-nij](./quick/260526-nij/) |
| 260526-nwn | 用户编辑模态框角色不显示修复 | 2026-05-26 | - | [260526-nwn](./quick/260526-nwn/) |
| 260527-gdc | 提交 AD 密码加解密修复代码到 git 仓库 | 2026-05-27 | cc2963b | [260527-gdc-ad-git](./quick/260527-gdc-ad-git/) |
| 260527-gra | 添加 User 模型 AdOuDn 字段并创建数据库迁移 | 2026-05-27 | 89feb82 | [260527-gra-user-adoudn](./quick/260527-gra-user-adoudn/) |
| 260527-ous | OU组织单位用户批量同步功能 | 2026-05-27 | 92638ce,db506ff,37723f8 | [20260527-ou-user-sync](./quick/20260527-ou-user-sync/) |
| 260527-adr | AD用户同步流程重构 - 整合部门解析逻辑 | 2026-05-27 | - | [20260527-ad-user-sync-dept-refactor](./quick/20260527-ad-user-sync-dept-refactor/) |
| 260527-l0k | 修复三个失败的调度任务(VDI同步/AD组同步/部门同步) | 2026-05-27 | - | [260527-enable-three-sync-tasks](./quick/260527-enable-three-sync-tasks/) |
| 260527-vls | 修复VDI服务器表缺少last_sync_time字段 | 2026-05-27 | - | [20260527-vdi-last-sync-time-fix](./quick/20260527-vdi-last-sync-time-fix/) |
| 260528-vxy | 优化VDI虚拟机创建表单：名称自动生成、下拉框选择、滑动条设置 | 2026-05-28 | 371aa50 | [260528-vxy-vdi-id-vdi-id-cpu](./quick/260528-vxy-vdi-id-vdi-id-cpu/) |
| 260529-j0l | 移除VDI虚拟机列表无意义的状态列（全栈） | 2026-05-29 | bbb3335 | [260529-j0l-vdi](./quick/260529-j0l-vdi/) |
| 260529-d95 | VDI创建表单添加资源级联下拉选择器 | 2026-05-29 | 29b80f8,a539277 | [260529-d95-vdi](./quick/260529-d95-vdi/) |
| 260529-ivp | VDI测试独立运行重构 | 2026-05-29 | 6dd7ab7 | [260529-ivp](./quick/260529-ivp-vdi-vdi-test-standalone-go-host-id-run-p/) |
| 260529-j3y | VDI虚拟机创建功能完整实现：添加运行位置（host.id/run_position.id逻辑）、存储位置、网络接口、VTP平台ID选择器 | 2026-05-29 | 132649f | [260529-j3y-vdi-host-id-run-position-id-vtp-id-vdi-t](./quick/260529-j3y-vdi-host-id-run-position-id-vtp-id-vdi-t/) |
| 20260602-config-ip | 移除VDI虚拟机列表页面配置IP功能（后端API+前端UI） | 2026-06-02 | - | [20260602-remove-vdi-config-ip](./quick/20260602-remove-vdi-config-ip/) |
| 20260602-vmp-api-integration | VMP服务器API验证脚本 | 2026-06-02 | a490262 | [20260602-vmp-api-integration](./quick/20260602-vmp-api-integration/) |
| 260602-jk9 | 虚拟机列表页面操作按钮控制逻辑修复 | 2026-06-02 | 644e222 | [260602-jk9-vm-list-controls](./quick/260602-jk9-vm-list-controls/) |
| 260602-k3r | VDI虚拟机列表页面修复：移除配置IP并恢复快速创建功能 | 2026-06-02 | - | [20260602-vdi-vm-list-fix](./quick/20260602-vdi-vm-list-fix/) |
| 260602-k1b | 去除虚拟机重命名功能和名称自动生成逻辑 | 2026-06-02 | 9b754b6 | [260602-k1b-vdi](./quick/260602-k1b-vdi/) |
| 260604-cqo | 虚拟机列表页面绑定用户功能，修改为下拉框组件，可以进行模糊查询，数据来源来自系统用户，前端显示nickname，数据库保存username | 2026-06-04 | 3d26d23 | [260604-cqo-nickname-username](./quick/260604-cqo-nickname-username/) |
| 260605-h3p | AD组自动化同步功能：实现基于OU-组映射的自动成员同步 | 2026-06-05 | 830aa0d | [260605-h3p-ad-group-auto-sync](./quick/260605-h3p-ad-group-auto-sync/) |
| 20260613-frontend-perf | 前端性能优化三连：图标按需导入(-15~-25%)、API加密并行化(-30~-40%)、Excel懒加载(-50~-100KB) | 2026-06-13 | - | [20260613-frontend-perf-optimizations](./quick/20260613-frontend-perf-optimizations/) |
| 260610-wsl | 修复工位管理页面同步设备后筛选条件丢失问题 | 2026-06-10 | - | [20250610-fix-workstation-filter-loss](./quick/20250610-fix-workstation-filter-loss/) |
| 260617-ch1 | 给所有页面表格添加排序功能，按字段排序，点击切换升降序 — 新增 createSorter helper + 25 个 page column 文件已应用 sorter，195+ createSorter 调用 | 2026-06-17 | 828f7df | [260617-ch1-add-table-sorting](./quick/260617-ch1-add-table-sorting/) |
| 260622-lv9 | 系统用户 Excel 导入功能 + 性别默认值改保密 + 导入后触发 AD 域同步（复用通用 ExcelImport，username 唯一键更新，AffectedKeys 通用化） | 2026-06-22 | 5d872a5,cc2a77d,7d40b32,4758d36 | [260622-lv9-xlsx-email](./quick/260622-lv9-xlsx-email/) |

## Accumulated Context

### Roadmap Evolution

- ✅ v1.0 完成: 工位导入部门/用户关联
- ✅ v1.1 完成: 信息点导入设备端口关联
- ✅ v1.2 完成: 可配置仪表盘生产级改造（4 个阶段）
- ✅ v1.3 完成: 技术债清理（3 个阶段）
- ✅ v1.4 完成: MAC地址采集优化
- 🚧 v1.5 进行中: MAC地址历史数据管理（Phase 12 完成，13-15 待规划）
- ✅ v1.6 完成: API密钥管理系统
- ✅ v1.7 完成: 前后端加密配置同步
- ✅ v1.8 完成: 登录端点加密增强
- ✅ v1.9 完成: AD域控集成扩展（Phase 19+20）
- ✅ v1.10 完成: 网络设备权限修复
- ✅ v1.11 完成: 深信服桌面云集成（Phase 22A - VDI基础集成，22B - VM Agent 服务）
- ✅ v1.12 完成: AD组自动同步系统 (Phase 23)
- 📋 Phase 22C-22D 规划中: 账号管理与密码、Web Console 监控
- 📋 v1.13 规划中: 资产管理模块 (Phase 26)
- 📋 Phase 24 added: 虚拟机数据范围权限配置
- 📋 Phase 23 added: 阶段23前端UI-部门组映射管理界面
- 📋 Phase 28 added: 工位设备关联子表格功能
- 📋 Phase 29 added: 重构工位状态枚举，从硬编码改为使用数据字典（sys_dict），保证前后端一致性
- 📋 Phase 30 added: 前端性能优化（消除瀑布流/包大小/重渲染/JS性能）— 基于 Vercel React 最佳实践审计发现的 70+ 问题
- 📋 Phase 31 added: P0 收尾 — connection_pool lifecycle + Config API 重设计（20260612 后端审查 F-14、F-17）
- 📋 Phase 32 added: v1.14 P1 重构与 P2 架构优化（20260612 后端审查剩余 P1/P2 任务）
- 📋 Phase 33 added: 前端 Vercel React Best Practices 修复 — 修复审查报告 20260613 中的 26 个性能/质量缺陷（7 CRITICAL + 5 MEDIUM-HIGH + 8 MEDIUM + 5 LOW + 1 bug：登录重复请求、WebSocket/Polling 抖动、Table 引用稳定化、Map 缓存化、Excel 改用 axios、R7 楼宇状态显示 bug）
- 📋 Phase 34 added: 操作日志全模块集成 — 审计发现 309 个写操作端点中仅 9 处（2.9%）有效记录（AD 域控），OperLogMiddleware 中间件已挂载但从未生效。基于 AD 域控参考实现分 7 Wave 集成全部端点
- 📋 Phase 37 added: 前端部门选择组件统一收敛 — 将碎片化的部门列表实现收敛到 useDeptTree 单一数据源 + 三组件(DeptTree/DepartmentTreeSelect/DeptSidebar)职责正交不合并 + 类型去重(SimpleDept×2+Department→1) + 转换函数归一(deptUtils)；明确排除 AD 域控整模块(独立 OU 数据源)/role 的 tree-select/dept 管理页。设计见 phases/37-dept-select-unify/DESIGN.md
- 📋 Phase 38 added: AD 账号池统一 — 移除遗留单管理员双轨路径(sys_ad_config.admin_username/admin_password)，所有 AD 连接(同步/管理/登录)统一走 Phase 36 账号池 FailoverClient，收敛为单轨消除解密不对称风险与维护负担。背景见 .planning/debug/adpool-password-not-decrypted.md

**Total shipped**: 72 plans, 18 phases completed

### Key Technical Achievements

**Phase 16 - API 密钥管理系统** (2026-05-19):

- 完整的 CRUD 操作
- 自动密钥生成（rec_ + 64位hex）
- 多重认证中间件（JWT + API Key）
- 速率限制器和 IP 白名单
- 测试覆盖率 81.1%（后端）

**Phase 17 - 前后端加密配置同步** (2026-05-20):

- 运行时配置同步
- 数据库驱动的配置管理
- 前端 5 分钟缓存
- 响应加密统一控制

**Phase 18 - 登录端点加密增强** (2026-05-21):

- SM2+SM4 请求体加密
- 三层加密保护
- 双层 SM2 密码加密（深度防御）
- 向后兼容性支持

**Phase 19 - AD 域控账号登录** (2026-05-21):

- 策略模式认证系统
- 初次登录自动同步
- 参数管理页面集成

**Phase 20 - AD OU 与部门映射** (2026-05-24):

- 双向映射功能
- 定时自动同步（每日凌晨 2 点）
- 用户登录自动设置部门
- 降级处理机制

**Phase 21 - 网络设备权限修复** (2026-05-24):

- 移除 OR 逻辑权限中间件
- 细粒度子路由权限控制
- 修复安全漏洞
- 7 个子路由组独立权限

### Known Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| v1.5 未完成 | Medium | Phase 12 已完成，13-15 可按需规划 |
| Phase 22 规模较大 | Medium | 拆分为 4 个子阶段，独立执行 |

## Next Steps

**当前选项**:

1. ~~**开始 Phase 22B** (VM Agent 服务)~~ ✅ 已完成
   - ~~1 个计划~~
   - ~~VM Agent 安装与配置~~

2. **开始 Phase 22C** (账号管理与密码)
   - 2 个计划
   - 账号管理、密码重置

3. **开始 Phase 22D** (Web Console 监控)
   - 2 个计划
   - VMP 监控数据采集

4. **继续 v1.5** (MAC地址历史数据管理)
   - Phase 13: 查询层与轨迹
   - Phase 14: 前端与用户体验
   - Phase 15: 性能优化

5. **规划新里程碑**
   - `/gsd-new-milestone` 创建新的里程碑

**推荐**: 开始 Phase 22C (账号管理与密码) 或 22D (Web Console 监控)

## Session Continuity

Last session: 2026-06-23T10:43:00.000Z
Stopped at: Phase 38 Plan 01 complete (Wave 1 前置 DI 脚手架), next: 38-02 连接层统一
Resume file: .planning/phases/38-unify-ad-authentication-onto-account-pool-remove-legacy-sing/38-02-PLAN.md

**Active Phase:** 38 (unify-ad-authentication-onto-account-pool-remove-legacy-sing) — 4 plans — Wave-based linear dependency (38-01 → 38-02 → 38-03 → 38-04)

  - Plan 38-01 (Wave 1 前置 DI 脚手架) ✅ COMPLETE — 8 服务 struct 注入 pool + NewADDomainService(db,pool,cipher...) + router/scheduler 共享同一实例
  - Plan 38-02 (Wave 1 连接层统一) ⬜ NEXT — 22 处 NewLDAPClient(config) 改 FailoverClient.ExecuteWithFailover
  - Plan 38-03 (Wave 2 Go 清理 + 前端) ⬜ — 删除 17 处 decryptPassword(config.AdminPassword) + 前端 admin 字段
  - Plan 38-04 (Wave 3 model + migration) ⬜ — model struct tag 放宽 + migration_164 补迁校验

**Planned Phase:** 37 (dept-select-unify) — 6 plans — **ALL 6/6 ✅ COMPLETE (awaiting phase-level verification)**

  - Plan 37-01 (批 0 公共层) ✅ — 硬前置已就位
  - Plan 37-02 (批 1 system user+notice) ✅ — system 模块收敛到 useDeptTree, 本地转换函数清零
  - Plan 37-03 (批 2 operations workstations+buildings) ✅ — workstations 高风险双向语义保留, buildings 方案 A+B 混合
  - Plan 37-04 (批 3 network devices) ✅ — network 模块收敛到 useDeptTree, 无转换函数
  - Plan 37-05 (批 4 duty+workorder) ✅ — duty/pools + useWorkOrderData 收敛到 useDeptTree; workorderApi SimpleDept 改 re-export + getDeptTree 副本删除; 全项目 SimpleDept 定义唯一
  - Plan 37-06 (批 5 收尾 TargetSelector + Phase 37 整体验收) ✅ — TargetSelector 迁移 + types/notice.ts dead code cleanup + 37-05 latent bug 修复 (Rule 1); Phase 37 §6 验收全部通过
