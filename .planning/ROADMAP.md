# Roadmap: RuoYi-Next 运维管理系统

## Milestones

- ◆ **v1.53 开放项全量清偿（OPEN-ITEMS Full Clearance）** — Phases 134-141 (initiated 2026-11-08, 24 REQ / 21 条开放项全覆盖) — [REQUIREMENTS.md](REQUIREMENTS.md) · [OPEN-ITEMS.md](OPEN-ITEMS.md)
- ✅ **v1.52 运维域数据权限统一收口（终态）** — Phases 130-133 (shipped 2026-11-07, 7 REQ；archived 2026-11-08) — [v1.52-ROADMAP.md](milestones/v1.52-ROADMAP.md) · [v1.52-REQUIREMENTS.md](milestones/v1.52-REQUIREMENTS.md) · [归档索引](ARCHIVE-INDEX.md)
- ✅ **v1.51 端口写生产级操作体验与平台质量收口** — Phases 121-129 (shipped 2026-11-02, 21 REQ) — [v1.51-REQUIREMENTS.md](milestones/v1.51-REQUIREMENTS.md)
- ✅ **v1.25 全站权限治理与样式收敛** — Phases 84-93 (shipped 2026-10-30) — [v1.25-ROADMAP.md](milestones/v1.25-ROADMAP.md)
- ✅ **v1.24 视觉回归修复与遗留债务清偿** — Phases 76-83 (shipped 2026-10-28) — [v1.24-ROADMAP.md](milestones/v1.24-ROADMAP.md)
- ✅ **v1.23 前端契约全量收尾与遗留清偿** — Phases 69-75 (shipped 2026-10-28) — [v1.23-ROADMAP.md](milestones/v1.23-ROADMAP.md)
- ✅ **v1.22 前端组件契约与设计系统对齐** — Phases 63-68.1 (shipped 2026-10-27) — [v1.22-ROADMAP.md](milestones/v1.22-ROADMAP.md)
- ✅ **v1.21 后端架构统一与接口解耦** — Phases 57-62 (shipped 2026-10-22) — [v1.21-ROADMAP.md](milestones/v1.21-ROADMAP.md)
- ✅ **v1.20 工位管理与空间可视化演进** — Phases 48-56 (shipped 2026-07-10) — [v1.20-ROADMAP.md](milestones/v1.20-ROADMAP.md)

---

## Phases

<details open>
<summary>◆ v1.53 开放项全量清偿 (Phases 134-141) — INITIATED 2026-11-08</summary>

> 需求基线：`.planning/REQUIREMENTS.md`（**24 REQ**：FIX-01~14 / VAL-01~07 / ACK-01~02 / GOV-01）｜唯一输入：`.planning/OPEN-ITEMS.md`（2026-11-08 唯一权威清单，A 节 8 + B 节 13 = **21 条真实开放项**）
> Phase 编号：延续 v1.52 从 **134** 续编
> 范围裁定（用户 2026-11-08）：**21 条全覆盖**，按「代码可修 / 待人工验证 / 已裁定接受」三类分别闭环；C 节 13 类防误读清单不纳入；**跳过领域研究**（21 条均带本仓 `文件:行号` 实证，无新领域能力需调研）；纳入 `seeds/SEED-ad-push-enable-direction.md` 为 FIX-01 输入
> **人工验证类不伪造闭环**：VAL-* 交付物 = 清单重建 + 统一跟踪表 + 回写口径，实际执行受生产 PG 只读 / 写操作授权 / 现场真机 / DBA 回传约束；AI 不擅自写生产数据（沿用 v1.25 Phase 90 先例）
> **登记接受类只落结论**：ACK-* 对应 OPEN-ITEMS.md 明示 LOCKED /「仅登记，不驱动修复」条目，不写修复代码

- [ ] Phase 134: AD 同步域 —— enable 方向 + 全量同步节流 + 漂移告警产品化 (FIX-01/10/11) — **验收** = `ADPushActionEnable` 入队/排水/SAFE-01 守卫/F-01 压制 sqlite 用例全绿 + 与 2026-10-21 d_chenchao 对称的实机 UAT；节流退避单测覆盖「连续失败不成风暴」；告警通道可配置可观测，方向与 A8① 裁定一致；`go test ./internal/scheduler/` 零回归
- [ ] Phase 135: Swagger / 文档产物链路修复与死文件清理 (FIX-02) — **验收** = `/swagger/index.html` 取到规范；两脚本产物落 `internal/docs/`；bat 层级 bug 修复；`docs/` 下 5 份死文件清零；`scripts/README.md:256-257` 矛盾表述修正；`go build ./...` 全绿
- [ ] Phase 136: 接线型端点缺口 —— 用户导出 + 工单评价 + 5 个权限码读路由 OQ-7 (FIX-03/06/13) — **验收** = 3 组端点可用 + operlog 覆盖 + 权限码挂载；OQ-7 提供 403→200 对照证据且不破坏 v1.25 严格权限中间件（RBAC 零回归）；前端导出按钮/评价页可用
- [ ] Phase 137: Stub 与双副本漂移治理 —— GetDiscoveryResults + rpa-worker (FIX-07/08) — **验收** = stub 返回真实结果且 handler/测试同步更新；rpa-worker 两副本 `createPage` 真实 Chrome + Wait* 超时断言 + 漂移结论落 `.planning/notes/`；两副本各自 `go build` 通过
- [ ] Phase 138: 功能分支补齐 —— scheduler Rotation + 端口写物理参数 + 3 处排序 (FIX-04/05/14) — **验收** = scheduler/service 双路径一致性测试；端口写 3 厂商模板 + 校验 + 审计零回归；3 处排序产品结论落 `.planning/notes/` 并按结论实现或显式登记「不推进」
- [ ] Phase 139: 低优先级技术债 —— 对账 MV 超时配置化 + 登录加密 benchmark 实测 (FIX-09/12) — **验收** = 30s 超时可配置且默认值不变；5 个 benchmark 产出数值与 P99<350ms 达标结论落盘；零新依赖
- [ ] Phase 140: 人工验证债组织 —— 5 份缺失 UAT 清单重建 + 7 组验证跟踪表 + 回写口径 (VAL-01~07) — **验收** = 重建 `57/48/54/56-HUMAN-UAT.md` 与 133-UAT 跟踪；统一跟踪表按「条目 → 前置授权 → 执行人 → 证据 → 回写状态」覆盖 VAL-01~06 全部条目；口径文档明确「未执行不得勾选」；不标已完成、不写生产数据
- [ ] Phase 141: 登记接受残余结论落盘 + OPEN-ITEMS.md 核销与权威分层复核 (ACK-01/02 + GOV-01) — **验收** = A8 四项 + B13①② 结论逐条落 `.planning/notes/`（含裁定/残余风险/不改代码理由）；21/21 条开放项在 OPEN-ITEMS.md 均有终态结论（已清偿移入 C 节并带日期+commit / 保留挂账并注明前置 / LOCKED 结论登记）；状态权威与开放项权威分层复核记录落盘

**执行波次记录：** W1 零依赖并行（134 AD 同步 / 135 Swagger / 136 端点接线 / 137 stub / 138 功能分支 / 139 技术债 —— 六者文件面基本不重叠，按模块隔离）→ W2（140 人工验证债组织，消费 W1 交付的清单与证据结构）→ W3 收尾（141，按 W1 清偿结果 + W2 跟踪表做统一回写与权威分层复核，**必须最后执行**）。
**并行冲突面**：134 动 `internal/services/system/user_service.go` / `internal/scheduler/ad_sync_tasks.go`；135 动 `cmd/main.go` / `scripts/` / `docs/`；136 动 `internal/api/router.go` / `internal/api/v1/workorder/`；137 动 `internal/services/device_discovery_service.go` / `rpa-worker/`；138 动 `internal/scheduler/workorder_tasks.go` / `internal/services/portwrite/`；139 动 `internal/core/core.go` / `tests/benchmark/` —— 134 与 138 均触及 `internal/scheduler/`，plan-phase 须产出函数级归属表，无法保证无交集则降级为串行（134 → 138）。
**关键约束：** 遵循 Handler-Service / response 包装 / operlog 约定 / 0=正常 1=停用 / 双方言（SQLite+PG，uuid 双侧 CAST）；回归基线 eslint 0 problems / vitest 全绿 / `go build ./...` + `go test ./...` 全绿；**不擅自写生产数据**；**LOCKED 项不写修复代码**；清偿回写严格按 OPEN-ITEMS.md 维护约定 §2（标注日期+证据并移入 C 节，不直接删除）。

</details>

<details>
<summary>✅ v1.52 运维域数据权限统一收口（终态）(Phases 130-133) — SHIPPED 2026-11-07</summary>

> 需求基线：`.planning/REQUIREMENTS.md`（7 REQ：DS-01~07）｜范围裁定（用户 2026-11-07）：**终态方案 = operations 全模块声明式 data_scope 统一收口**，不做逐端点手工接线
> Phase 编号：延续 v1.51 从 **130** 续编
> **立项动因**：v1.51 期间实测确认 operations 全模块未接 data_scope（List orgId 后端不校验 / strict 组不注入 / 三份重复实现），quick-261106-wsb 审计裁定逐端点补丁为非终态

- [x] Phase 130: ScopeProvider 基建 —— 可见部门集统一实现（**只统一 Custom/Dept/DeptChild 三档部门集展开，Self/All 留资源语义钩子**——vdi Self 档按 bound_user_id 与部门集不等价，vm_data_scope_filter.go:87-100）+ data_scope/user_id 注入覆盖 operations 全域（**在 ops 各读组 RequirePermissionsWithQuery 与 strict 写组 RequireStrictPermissions 统一挂载/改造 DataScopePermission，permission.go:494 起；不改两个权限中间件自身逻辑**）(DS-01/02) — fail-closed 对齐 PMODEL-01（**data_scope=0/unset 映射 All，仅非 0 异常枚举 fail-closed；超管零回归用例**）；All 档特判短路性能要求写入本 phase 约束；**范围裁定输入：Self 档 ops 域语义须在本 phase 开工前落 `.planning/notes/`（未产出前仅可起草不含 Self 定案的 ScopeProvider 骨架）**；**验收** = `go test ./...` 全绿 + ScopeProvider 单测（五档 / 0 值 / 异常枚举 fail-closed / 多角色 MAX=最严档 / 无角色 COALESCE→Self(5) / 超管零回归）+ `DataScopePermission` 挂载组数断言（5 组 → ops 全域） — ✅ **completed 2026-11-07**
- [x] Phase 131: 四域读端点 + Excel 导出独立收口 —— workstation/floor/building/infopoint 四域 List/Statistics/SearchOptions/dropdown-options/GetByID/tree/**dept-options/workstations-all(:813)** 统一「有效机构 ∈ 可见集」（复用 effectiveOrgExpr 单一真实源）；缓存 key 增数据范围维度段；**Excel 导出不豁免且不走 List**（`excel_query_builder.go:215-254 WorkstationQueryBuilder.BuildQuery` + `excel_service.go:2407-2453 queryWorkstationIDsForExport` 两条独立路径须注入可见集过滤 + PG contract 覆盖） (DS-03/06 部分) — **验收** = 403 越权矩阵（读端点 × 越界 orgId，含 workstations-all/GetByID/tree）+ **ops 缓存面清点表（逐 key 现状→目标；`floor_cache_impl.go:39-45 floor:tree` 全局固定 key 因按可见集过滤必须改结构或停用缓存、`floor:building:*`、`dropdown_helper.go:32-57` 无 scope 维度）+ 缓存 key 结构断言 + 跨用户对照实验（覆盖 List/tree/dropdown/GetByID，防 PMODEL-01 跨范围泄漏重演）** + PG contract 覆盖收口查询与导出 QueryBuilder；All 档特判短路性能要求同为本 phase 约束 — ✅ **completed 2026-11-07**
- [x] Phase 132: 四域写端点收口（workstation/floor/building/infopoint） —— Create/Update/Delete/BatchOperation 校验 deptId/orgId/资源归属（floor 写 :767-776、infopoint 写 :996-1005 一并收口）；backfill-dept 回填范围限定可见集 + 前端确认框范围可视化 (DS-04) — **范围裁定输入：Dept/Self 批量 + Excel 导入豁免须在本 phase 开工前落 `.planning/notes/`（backfill 批量在本 phase 涉及）**；**验收** = 403 越权矩阵（四域 Create/Update/Delete/Batch × 越界 deptId/orgId）+ backfill 范围限定断言 + 前端确认框范围文案断言 + PG contract 覆盖导入路径 — ✅ **completed 2026-11-07**
- [x] Phase 133: 扩展实体 + vdi 合并迁移 + 按已裁定结论落地 —— asset/serverRoom/dedicatedLine/roomDevice 收口、**vdi 调用方迁移 + 删旧实现（含 middleware 死实现与死测试删除）**、按 130/132 已裁定结论落地 Dept/Self 批量与 Excel 导入豁免、前端 DeptSidebar 联动、UAT (DS-05/06/07) — **开工输入：DS-05 三项裁定结论（dedicatedLine/roomDevice 归属链 scope、workstation-device 归属、location-alias 查询面）须在 133 开工前产出（见 STATE 裁定记录），133 只消费结论、不承载裁定动作**；**内部分批验收** = ① 四实体读写 403 矩阵 ② vdi/超管零回归 + middleware 死实现/死测试删除 ③ 前端 DeptSidebar 可见集断言 ④ UAT — ✅ **completed 2026-11-07**

**执行波次记录：** W1 基建（130）→ W2 读写并行（131 ∥ 132，均只依赖 130）→ W3 收尾（133，按 130/132 已裁定结论落地 + 消费 133 开工前已产出的三项裁定结论）。**W2 并行前置约束（131∥132 文件冲突面）**：131 收口四域读 + 缓存 key 必动 `workstation_service.go`/`floor_service.go`/`building_service.go`/`infopoint_service.go`/`floor_cache_impl.go`/`dropdown_helper.go`/`dept_filter.go`（+ 各自 handler/router），132 四域写/缓存失效同样动这些（`workstation_service.go:644 InvalidatePattern`）——plan-phase 阶段必须产出**覆盖上述全部文件的归属表**（逐函数列明归属 phase，131 只动读路径函数 + 缓存 key，132 只动写路径函数）作为并行许可凭证；若 plan 阶段无法保证函数级无交集，**默认降级为串行（131 → 132）**。

**关键约束：** uuid/varchar 双侧 CAST + PG contract 门控扩展至新查询｜effectiveOrgExpr 单一真实源禁止内联｜fail-closed 对齐 PMODEL-01（0/unset→All，异常枚举才 fail-closed；MAX=最严档语义不变 + 多角色组合/无角色 COALESCE 边界用例）｜缓存 key 必含 scope 维度且 All 档特判短路路径同样可断言；**Phase 131 必交「ops 缓存面清点表」**（`floor:tree` 固定 key 因按可见集过滤必须改结构或停用，PMODEL-01 跨范围泄漏教训）｜PG contract 新增测试统一门控变量名 `TEST_PG_DSN`（与 `pg_contract_test.go:36` 一致）或显式兼容 `XINGRAN_TEST_PG_DSN`，CI 中 skip 必须可观测；**导出/导入路径须纳入 PG contract 覆盖**（现状 `pg_contract_test.go:87-208` 仅覆盖 List/Statistics/SearchOptions/Backfill）｜回归基线 eslint 0 / vitest 全绿 / go build + go test 全绿（含路由双层守卫）

</details>

<details>
<summary>✅ v1.51 端口写生产级操作体验与平台质量收口 (Phases 121-129) — SHIPPED 2026-11-02（21/21 REQ · 真机 UAT 已完成）</summary>

> 需求基线：`.planning/REQUIREMENTS.md`（21 REQ）｜范围裁定：A 端口写深化 + B 平台技术债 + D 收尾杂项（C 组 RPA 编辑器 defer）
> Phase 编号：延续 GitHub v1.33 线从 **121** 续编
> **交付状态：** 21/21 REQ 交付 · PR #38 merged（e602e811）· tag v1.51 · 集成验证窗口 #1 全项 PASS

- [x] Phase 121: 端口写 dry-run 预览 + 命令预览 UI (PORTW-01/02) — completed 2026-10-31
- [x] Phase 122: 批量写实时进度 SSE/WS — BATCH-05 清偿 (PORTW-03) — completed 2026-11-01
- [x] Phase 123: 安全三件套 — 快照回滚 + 并发写互斥 + 不可达预检 (PORTW-04/05/06) — completed 2026-11-01
- [x] Phase 124: VLAN 扩展 — trunk native/allowed + hybrid 华为/H3C (PWX-01) — completed 2026-11-01
- [x] Phase 125: 端口安全 — port-security/sticky/静态 MAC 绑定 (PWX-02) — completed 2026-11-01
- [x] Phase 126: Cisco IOS-XE 第 4 厂商 + 跨固件差异矩阵 (PWX-03) — completed 2026-11-01
- [x] Phase 127: 审计 UI + 审计治理 + 3 厂商 e2e + handler e2e (PWX-04/05/06 + DEBT-02) — completed 2026-11-01
- [x] Phase 128: 平台质量收口 — addomain 测试 / RPA Token / operlog 白名单 / permission 时序 / 排序菜单注册 (DEBT-01/03/04/05 + TIDY-01/03) — completed 2026-11-01
- [x] Phase 129: 目录改名 + 双线 planning 合流 + UAT 回补组织（DEBT-06 + TIDY-02）— ✅ **已完成**：真机 UAT（华为 11 + 锐捷 12 动作，设备零残留）｜DEBT-06 前端目录改名（1355 文件 rename）｜TIDY-02 UAT 回补组织 + Phase 90 登录态巡检闭合（40 组合实跑 + 42 张截图）｜附带清除远端 265 个含密文件 + 口令脱敏

**执行波次记录：** W1 零依赖（121 → 122/128 并行）→ W2 需设备（123 → 124/125）→ W3（126 → 127）→ W4 收尾（129，需真机）

</details>

<details>
<summary>✅ v1.25 全站权限治理与样式收敛 (Phases 84-93) — SHIPPED 2026-10-30</summary>

- [x] Phase 84: 权限模型三缺口 (4/4 plans) — completed 2026-10-28
- [x] Phase 85: 后端动作级严格化 (4/4 plans) — completed 2026-10-29
- [x] Phase 86: 前端操作入口三态门控铺开 (6/6 plans) — completed 2026-10-29
- [x] Phase 87: 动作级权限授予闭环 (5/5 plans) — completed 2026-10-30
- [x] Phase 88: 验证底座与零风险清理 (4/4 plans) — completed 2026-10-29
- [x] Phase 89: 变量层与明暗色板重构 (4/4 plans) — completed 2026-10-29
- [x] Phase 90: Input 域交还 variant/token + 圆角统一 (4/4 plans) — completed 2026-10-30
- [x] Phase 91: Select / InputNumber / Form / Table / Modal 域交还 (3/3 plans) — completed 2026-10-30
- [x] Phase 92: 暗色 mega-cluster 合并 (5/5 plans) — completed 2026-10-30
- [x] Phase 93: room-devices 拆分 + 视觉回归验收 (2/2 plans) — completed 2026-10-30

</details>

<details>
<summary>✅ v1.24 视觉回归修复与遗留债务清偿 (Phases 76-83) — SHIPPED 2026-10-28</summary>

- [x] Phase 76: 双框 P0 修复 + CSS 审计登记册 — completed 2026-10-28
- [x] Phase 77: 双框视觉回归验证 — completed 2026-10-28
- [x] Phase 78: #16 前后端双端闭环 — completed 2026-10-28
- [x] Phase 79: #16 PG 实机验收 — completed 2026-10-28
- [x] Phase 80: 3f130 空因诊断前端接入 — completed 2026-10-28
- [x] Phase 81: F5 刷新落地页修复 — completed 2026-10-28
- [x] Phase 82: lint 清零 + 豁免理由表修复 — completed 2026-10-28
- [x] Phase 83: devices 页拆分 — completed 2026-10-28

</details>

---
*Roadmap updated: 2026-11-08 — v1.53 INITIATED（开放项全量清偿，Phases 134-141 / 24 REQ / 21 条开放项全覆盖；唯一输入 `.planning/OPEN-ITEMS.md`；跳过领域研究；Phase 编号延续 133 从 134 续编）*
