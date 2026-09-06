---
last_updated: 2026-09-06
milestone: v1.30
update_trigger: v1.30 workstream ROADMAP 创建 — v1.29 内容已随 milestone 归档（milestones/v1.29-ROADMAP.md）；22 requirements → 6 phases（96-101）全覆盖映射（CACHEDEF/JOBSTAT/BACKUPFIX/CACHEKEY/OPSFIX/FEFIX/CLOSEOUT 七类别）
---

# Roadmap: XingRan-Next 运维管理系统 — v1.30 milestone workstream

> **v1.29 及更早的 milestone 历史已归档**: `.planning/milestones/`(v1.29-ROADMAP.md / v1.29-REQUIREMENTS.md / v1.29-phases/)。
> 本文件自 2026-09-06 起只追踪 **v1.30 V130 缺陷治理 (Defect Remediation)**。

## Current Milestone: v1.30 V130 缺陷治理 (Defect Remediation)

**Goal:** 修复 v1.29 期间登记的全部 18 项 V130-CANDIDATES 缺陷候选 + 闭环 2 个 deferred 小项（4 个未跟踪测试文件入库决策 + 62-HUMAN-UAT 3 场景）。所有修复属行为变更，每项附回归测试（v1.29 D-05 例外条款同款纪律）；七 gate 全程不倒退，使深度复查发现的问题不再带病运行。

**Source planning data:**

- `.planning/REQUIREMENTS.md` (7 类别 / 22 requirements，Traceability 已回填 phase 映射)
- `.planning/PROJECT.md` (Current Milestone v1.30 段, D-01..D-05 locked decisions)
- `.planning/milestones/v1.29-DEEP-RECHECK.md` (V130R-01..12 深度复查详情)
- `.planning/milestones/v1.29-REQUIREMENTS.md` V130-CANDIDATES 段 (CACHEDEF-01..05 + JOBSTAT-01)

**Milestone success criteria:**

- SC-a (覆盖): 22/22 requirements 全部交付，每项修复附回归测试（或守卫 / UAT 人工验证回写）
- SC-b (gate): 七 gate 全程不倒退——go build / go test / 后端 coverage ≥77.5 / 前端 45 dirs / lint / type-check / diff coverage
- SC-c (设计决策): V130R-01/02/03/09 的技术方案在对应 phase 规划时敲定并落盘（CONTEXT.md，D-03 锁定决策项）
- SC-d (闭环): 4 个未跟踪测试文件入库决策终结（TESTFILE-01）+ 62-HUMAN-UAT 3 场景真实 PG 人工验证回写

**锁定决策 (v1.30 init):**

- **D-01 范围**: 18 项 V130-CANDIDATES 全做 + 2 顺带项；不引入新业务功能
- **D-02 回归纪律**: 所有修复属行为变更，每项附回归测试；七 gate 全程不倒退
- **D-03 设计决策项**: V130R-01 / V130R-02 / V130R-03 / V130R-09 的方案在 phase 规划时敲定
- **D-04 范围外**: WSNOTICE-01 已提前修复；operlog exclude_paths 继续挂账；前端覆盖率不推新目标
- **D-05 Phase 编号**: 从 Phase 96 起续编（v1.29 用 89-95，v1.28 用 82-88）

## Phases

- [x] **Phase 96: 确定性缓存/看板缺陷修复** — CACHEDEF-01..05 五项确定性缺陷 + JOBSTAT-01 死代码删除处置，修复项每项附回归测试 ✓（2026-09-06）
- [x] **Phase 97: config_backup 恢复链加固** — 互斥原子性/实例归属/业务错误码（V130R-01..03）✓（2026-09-07 recovery 落库 2dd46a3，见 RECOVERY-NOTE）
  **Plans**: 3 plans (97-01: V130R-01 超时互斥原子化; 97-02: V130R-02 多实例归属过滤; 97-03: V130R-03 业务错误码语义化)
- [x] **Phase 98: 缓存键安全与 base 迁移收尾** — 列表键防碰撞 + 四包 interface{} 残留迁 base 泛型（V130R-04..05）✓（2026-09-06，2/2 plans；helper 补遗 2b15574）
- [x] **Phase 99: operations 口径统一** — Total 软删/换楼乱序/orgId 子部门筛选/分页 clamp 收敛（V130R-06..09）✓（2026-09-07 执行完成，6/6 plans 含 99-06 gap closure；99-VERIFICATION gaps 已关闭）
- [x] **Phase 100: 前端契约修复** — 幽灵方法处置/rpaApi 契约对齐/networkApi 下载链收敛（V130R-10..12）✓（2026-09-07，3/3 plans；100-VERIFICATION 6/6 SC + gates 独立实跑全绿）
- [x] **Phase 101: 收口——测试文件入库 + 62-UAT + audit** — TESTFILE-01 + UAT62-01..03 + 七 gate 全绿 + audit ✓（2026-09-07，2/2 plans；UAT62-03 passed 带证据链，UAT62-01/02 诚实 pending 待真实 PG——runbook `.planning/phases/101-closeout-uat-audit/UAT-RUNBOOK.md` 就绪）

### Phase Dependency Graph

```
Phase 96 (CACHEDEF+JOBSTAT 确定性缺陷) ─→ Phase 98 (CACHEKEY 键安全 + 四包迁移；同 cache_impl 文件族)
Phase 97 (BACKUPFIX 恢复链加固；与 96 零文件重叠)
Phase 99 (OPSFIX 口径统一；v1.29 Phase 91 base.Repository 基线)      ─┐
Phase 100 (FEFIX 前端契约；v1.29 Phase 94 apiFactory 基线)           ─┼─→ Phase 101 (收口，必须最后)
```

**并行机会**: Phase 97/99/100 与 96/98 零文件重叠可并行（config `parallelization: false`，默认顺序执行）；Phase 98 必须在 96 后（duty/workorder cache_impl 同文件族——先修缺陷再迁移）；Phase 101 必须最后。

---

## Phase Details

### Phase 96: 确定性缓存/看板缺陷修复

**Goal**: 修复 5 项缓存确定性缺陷，使缓存失效真正命中、缓存键不再互相污染；另删除 JOBSTAT-01 死代码（`job_utils.go`，2026-09-06 discuss 分析无生产调用方，原时区日界缺陷随删除消解）；每项修复附回归测试防倒退。全部为小而确定的修复（方案已知，无设计决策）。

**Depends on**: Nothing (v1.30 first phase)

**Requirements**: CACHEDEF-01, CACHEDEF-02, CACHEDEF-03, CACHEDEF-04, CACHEDEF-05, JOBSTAT-01（删除处置）

**Success Criteria** (what must be TRUE):

  1. 缓存失效真正命中：部门树删除/更新后 `InvalidateDeptCache` 命中写键、缓存读回为新数据（CACHEDEF-01）；单条删除配置后 `config:id:<id>` 同步失效，30 分钟窗口内详情接口不再从缓存读回已删配置（CACHEDEF-02）
  2. 缓存键口径修复：duty 月份解析不再恒 0，`GenerateSchedule`/`ManualDuty` 后月度排班缓存失效生效（CACHEDEF-03）；不同 limit 的待办查询各占独立缓存键、互不污染（CACHEDEF-04）
  3. 缓存监控前缀剥离生效：`key[:6]` 切片长度修正（6 字节比 8 字节字面量恒 false）后，含 `xingran:` 前缀的键在缓存监控操作中命中真实键（CACHEDEF-05）
  4. JOBSTAT-01 死代码删除：`internal/api/v1/job_utils.go`（GetJobStatistics + FormatDuration，均无生产调用方）连同对应测试删除，`go build ./...` 通过，REQUIREMENTS/ROADMAP 账目同步
  5. 回归纪律：5 项修复每项附回归测试；`go build ./...` + `go test ./...` 0 失败，七 gate 不倒退

**Plans**: 3 plans ✓（96-01 CACHEDEF-01..04 回归修复; 96-02 CACHEDEF-05 monitor prefix; 96-03 JOBSTAT-01 死代码删除，2026-09-06 完成）

**Notes**: 涉及文件：system/department_cache_impl.go / system/config_cache_impl.go / duty/duty_cache_impl.go / workorder/workorder_cache_impl.go / monitor/cache_service.go + 删除 api/v1/job_utils.go——Phase 98 将再触 duty/workorder cache_impl（迁移），本相先修缺陷。

---

### Phase 97: config_backup 恢复链加固

**Goal**: config_backup 恢复链三项缺陷修复——超时后 worker 不再向设备推送、多实例不误杀在途任务、业务错误语义化返回。含 3 个开放设计决策（V130R-01/02/03），phase 内先 discuss 敲定方案再执行。

**Depends on**: Nothing（与 Phase 96 零文件重叠；涉及 config_restore_task_service.go / network_router.go / backup_handler.go）

**Requirements**: V130R-01, V130R-02, V130R-03

**Success Criteria** (what must be TRUE):

  1. 超时互斥原子化：恢复任务超时后 worker 不再继续向设备推送命令，背对背新恢复请求不再出现双任务窗口，RestoreResult 无数据竞争（V130R-01，分段 context 预算方案经 discuss 敲定）
  2. 实例归属过滤：RecoverStaleRunningTasks 在多实例/滚动重启下不误杀其他实例在途任务、不致终态翻转（V130R-02，grace period >RestoreConfigTimeout 或实例标识列方案经 discuss 敲定）
  3. 业务错误码语义化：「存在进行中恢复任务」映射 409/400、「备份不属于目标设备」映射 400，不再统一经 HandleServiceError 返回 500（V130R-03，pkg/response 业务错误类型体系方案经 discuss 敲定）
  4. 回归纪律：3 项修复每项附回归测试（超时路径 / 归属过滤 / 错误码映射）；七 gate 不倒退

**Plans**: 3 plans (97-01: V130R-01 超时互斥原子化; 97-02: V130R-02 多实例归属过滤; 97-03: V130R-03 业务错误码语义化)

**Notes**: 本 phase 是 v1.30 设计决策密度最高的 phase（D-03 三项全在此）——plan-phase 前置 discuss 产出 CONTEXT.md 后再拆 plan。V130R-03 的业务错误类型体系是跨模块基建，落地后其他 phase 错误路径可复用。

---

### Phase 98: 缓存键安全与 base 迁移收尾

**Goal**: 缓存列表键防碰撞 + duty/knowledge/network/workorder 四包残留 interface{} GetOrSet 全量迁 `base.GetOrSetJSON[T]` 泛型函数族，cache_invariants_92_test warning 档清零——v1.29 Phase 92 缓存统一的完全收口。

**Depends on**: Phase 96（duty/workorder cache_impl 同文件族——先修缺陷再迁移，避免同文件冲突）

**Requirements**: V130R-04, V130R-05

**Success Criteria** (what must be TRUE):

  1. 列表缓存键不再碰撞：`Username="bob:status:1"` 与 `Username="bob"+Status=1` 不再生成同一缓存键（键值转义或参数集哈希，方案 plan 时敲定），用户/角色列表缓存无交叉污染（V130R-04）
  2. 四包迁移收尾：duty/knowledge/network/workorder cache_impl 的 11 处 interface{} 闭包 GetOrSet 全部收敛 `base.GetOrSetJSON[T]` 单 return，4 个平行 getExpiration 收敛 base TTL 解析（V130R-05）
  3. invariants 锁升级：`cache_invariants_92_test.go` warning 档四包清零（移入硬档或残留计数 = 0）
  4. 回归纪律：键防碰撞附回归测试；`go test ./internal/services/...` 0 失败，七 gate 不倒退

**Plans**: 2 plans ✓（98-01 V130R-04 键防碰撞; 98-02 V130R-05 四包迁移 base 泛型，2026-09-06 完成）

---

### Phase 99: operations 口径统一

**Goal**: operations 域四处口径/语义缺陷统一——List Total 软删过滤、换楼同步乱序、orgId 子部门筛选共享 helper、分页 clamp 三口径收敛；V130R-09 合并方案 phase 内 discuss 敲定。

**Depends on**: Nothing（v1.29 Phase 91 `base.GORMRepository[T]` / Phase 89 `pkg/constants` 为既定基线）

**Requirements**: V130R-06, V130R-07, V130R-08, V130R-09

**Plans**: 5 plans ✓ (99-01: V130R-06 Total 口径; 99-02: V130R-07 换楼有序化; 99-03: V130R-08 orgId helper; 99-04: V130R-09 Go 分页口径核心; 99-05: V130R-09 CAD 端点+前端迁移——2026-09-07 全部执行完成)

**Success Criteria** (what must be TRUE):

  1. Total 口径收紧：asset/building List Total 不再计入软删记录（`.Table()` 起链 → repo `Model(new(T))` 对齐），软删环境下分页器页数正确；OVR 台账补记（V130R-06）
  2. 换楼同步有序：floor 换楼同步不再乱序（乐观条件 WHERE building_id=旧值 或队列串行化），First 失败不再被静默吞掉；OVR 补记（V130R-07）
  3. orgId 筛选统一：「部门+全部子部门」抽共享 helper 统一四条件口径，6+ 处复制粘贴收敛；workstation/infopoint 三条件形式漏匹配 ancestors 中段的缺陷修复（V130R-08）
  4. 分页 clamp 收敛：三口径（pagination_helper 10..10000 / requests.GetPagination 10..100 / pkg/query ..200）收敛为单一权威路径；internal/constants 与 pkg/constants 双包合并方案经 discuss 敲定落地，base/service.go 注释自认刻意保留处一并处置（V130R-09）
  5. 回归纪律：4 项修复每项附回归测试；七 gate 不倒退


**Notes**: V130R-09 是 D-03 第四个设计决策项（双包合并路径 phase 规划时敲定）。

---

### Phase 100: 前端契约修复

**Goal**: 前端 API 契约与后端真实路由对齐——幽灵方法处置、rpaApi 全族契约对齐专项（本 milestone 最大单项对齐工程）、networkApi 下载链收敛到权威 download.ts。

**Depends on**: Nothing（v1.29 Phase 94 `src/lib/apiFactory.ts` 单一权威 + D-12 双档扫描防线为既定基线）

**Requirements**: V130R-10, V130R-11, V130R-12

**Plans**: 3 plans (100-01: rpaApi 幽灵清除+全族裁剪 17 存活端态+RECONCILIATION; 100-02: /vdi/vms/operate 补注册+accounts Tab 删除+vdiApi 幽灵 pick 化; 100-03: 下载链收敛 download.ts+JSON 检测+invariants 递归化)

**Success Criteria** (what must be TRUE):

  1. 幽灵方法处置：rpaApi 8 处 / vdiApi 2 处 spread 幽灵方法被 omit（或 apiFactory invariants 增加「新增方法须有后端路由」对照断言），调用面 404 隐患消除（V130R-10）
  2. rpaApi 契约对账清单落盘：scriptApi/scheduleApi/variableApi/templateApi/notificationApi/statisticsApi 全族与后端路由逐一对账，死方法裁剪或后端路由补齐，附守卫防回增（V130R-11）
  3. 下载链收敛：`src/lib/api/networkApi.ts` 平行下载链收敛到权威 download.ts；`downloadFilePost` 补 JSON 错误体检测——200+JSON 错误响应不再存成 .xlsx；apiFactory invariants 的 readdirSync 改递归扫描（V130R-12）
  4. 回归纪律：3 项修复每项附回归测试（vitest）；npm run lint / type-check / test 0 errors，前端 45 dirs gate 不倒退


**Notes**: 全部为 lib 层契约修复，不涉及页面/组件改动。V130R-11 体量最大（全族对账），plan-phase 时可能拆多 plan。

---

### Phase 101: 收口——测试文件入库 + 62-UAT 人工验证 + audit

**Goal**: v1.30 收口——4 个未跟踪测试文件入库决策终结 + 62-HUMAN-UAT 3 场景在真实 PG 环境人工验证闭环 + 七 gate 全绿 + v1.30 audit 落盘。

**Depends on**: Phase 96 + 97 + 98 + 99 + 100（必须最后）

**Requirements**: TESTFILE-01, UAT62-01, UAT62-02, UAT62-03

**Success Criteria** (what must be TRUE):

  1. 测试文件入库决策终结：4 个未跟踪 `*_test.go`（rpa_model_methods / cache manager_coverage / sysmetrics_common / sysmetrics_windows）入库并纳入 gate，或明确排除归档——95-02 Pitfall 4 选项 c 口径不再悬置（TESTFILE-01）
  2. UAT 场景 1 回写：带旧结构 MV 的真实 PG 上启动验证 Migrate176 R1/R2→R5 就地升级 schema 校验回退通过，62-HUMAN-UAT.md 场景 1 回写 passed（UAT62-01）
  3. UAT 场景 2 回写：Advisory lock 下第二实例跳过迁移块、打 WARN 且正常启动，62-HUMAN-UAT.md 场景 2 回写 passed（UAT62-02）
  4. UAT 场景 3 回写：空库首启 admin 默认凭据 WARN / env 覆盖 / salt 非默认验证通过，62-HUMAN-UAT.md 场景 3 回写 passed（UAT62-03）
  5. 七 gate 本地全绿（go build / go test / 后端 coverage ≥77.5 / 前端 45 dirs / lint / type-check / diff coverage）+ v1.30-MILESTONE-AUDIT.md 落盘

**Plans**: 2 plans（101-01 TESTFILE-01 入库+七gate 实测+追溯终表; 101-02 UAT62 三场景 runbook+回写）

**Notes**: UAT62-01..03 为人工验证项（真实 PG 环境，owner = 用户/运维）——executor 负责准备验证步骤清单与自动化前置（可自动部分），人工执行后回写。62-HUMAN-UAT.md 位于 `.planning/workstreams/milestone/phases/62-ai-internal-core-db/`（v1.27 workstream 遗留）。

---

## Progress

| Phase | Status | Plans | Requirements | Started | Completed |
|-------|--------|-------|--------------|---------|-----------|
| Phase 96 确定性缓存/看板缺陷修复 | Completed | 3/3 | CACHEDEF-01..05 + JOBSTAT-01 | 2026-09-06 | 2026-09-06 |
| Phase 97 config_backup 恢复链加固 | Completed | 3/3 | V130R-01..03 | 2026-09-06 | 2026-09-07（recovery 落库） |
| Phase 98 缓存键安全与 base 迁移收尾 | Completed | 2/2 | V130R-04..05 | 2026-09-06 | 2026-09-06 |
| Phase 99 operations 口径统一 | Completed | 6/6 | V130R-06..09 | 2026-09-06 | 2026-09-07 |
| Phase 100 前端契约修复 | Completed | 3/3 | V130R-10..12 | 2026-09-07 | 2026-09-07 |
| Phase 101 收口（测试文件入库 + 62-UAT + audit） | Completed | 2/2 | TESTFILE-01 + UAT62-01..03 | 2026-09-07 | 2026-09-07 |

**Total:** 6 phases / 22 requirements（22/22 traceability 见 `.planning/phases/101-closeout-uat-audit/TRACEABILITY-FINAL.md`；UAT62-01/02 诚实 pending 待真实 PG 人工执行，runbook 就绪）

---

## Out of Scope (locked from v1.30 init)

- WSNOTICE-01（WS 双读者竞态 + origin 前缀绕过）——已于 2026-09-06 v1.29 深度复查提前修复（503c162 + 6a44659）
- operlog exclude_paths 白名单——独立 deferred pending todo
- 新业务功能——v1.30 锁定为缺陷治理
- 前端覆盖率推新目标——v1.28 已阶段性收口 45.13%（D-04）
- CACHEDEF/V130R 之外新扫描发现的缺陷——登记新 candidates，不顺手扩 scope

---

*Last updated: 2026-09-06 — v1.30 ROADMAP 创建（22 requirements → 6 phases 96-101 全覆盖映射；设计决策项 V130R-01/02/03 → Phase 97、V130R-09 → Phase 99；UAT62/TESTFILE → Phase 101）。上一 milestone: v1.29 技术债治理 SHIPPED + ARCHIVED 2026-09-06（7 phases / 26 plans / 45 requirements，见 milestones/v1.29-ROADMAP.md）。*
