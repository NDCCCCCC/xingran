# Requirements: XingRan-Next — Milestone v1.30 V130 缺陷治理

**Defined:** 2026-09-06
**Core Value:** 修复 v1.29 期间登记的全部 18 项 V130-CANDIDATES 缺陷候选 + 闭环 2 个 deferred 小项；所有修复附回归测试，使深度复查发现的问题不再带病运行。

**输入来源:**
- `.planning/milestones/v1.29-REQUIREMENTS.md` V130-CANDIDATES 段（CACHEDEF-01..05 + JOBSTAT-01）
- `.planning/milestones/v1.29-DEEP-RECHECK.md`（V130R-01..12）
- `.planning/STATE.md` v1.29 Deferred Items（TESTFILE / 62-HUMAN-UAT）

**锁定决策 (v1.30 init):**
- **D-01 范围**: 18 项全做 + 2 顺带项；不引入新业务功能（2026-09-06 修订：JOBSTAT-01 经 Phase 96 discuss 死代码分析重定性为**删除处置**，修复项 18→17，详见 96-CONTEXT.md）
- **D-02 回归纪律**: 所有修复属行为变更，每项附回归测试（v1.29 D-05 例外条款同款纪律）；七 gate（go build / go test / 后端 coverage ≥77.5 / 前端 45 dirs / lint / type-check / diff coverage）全程不倒退
- **D-03 设计决策项**: V130R-01/02/03/09 的技术方案在 phase 规划时敲定
- **D-04 范围外**: WSNOTICE-01 已提前修复；operlog exclude_paths 继续挂账；前端覆盖率不推新目标
- **D-05 Phase 编号**: 从 Phase 96 起续编

## v1.30 Requirements

### CACHEDEF — 缓存缺陷修复

- [ ] **CACHEDEF-01**: `system/department_cache_impl.go:74-102` — `GetSelectDataWithCache` 写键与 `InvalidateDeptCache` 失效模式统一（现写裸键 `"dept:tree"` 与 `cache:` 前缀模式永不匹配），失效真正命中；附回归测试
- [ ] **CACHEDEF-02**: `system/config_cache_impl.go:71-118` — 单条 Delete 失效补齐 `config:id:<id>`，已删配置不再能经详情接口从缓存读回（30min 窗口消除，`config_router.go:17` 生产可达）；附回归测试
- [ ] **CACHEDEF-03**: `duty/duty_cache_impl.go:333-344` — `parseInt` 的 `len(s) >= 4` 前置修复（2 字符月份切片不再恒返回 0），`GenerateSchedule`/`ManualDuty` 后月度排班缓存失效生效（`duty_handler.go:325` 生产可达）；附回归测试
- [ ] **CACHEDEF-04**: `workorder/workorder_cache_impl.go:207-234` — 待办缓存键补入 Limit 维度，不同 limit 不再共享同一缓存；附回归测试
- [ ] **CACHEDEF-05**: `monitor/cache_service.go:766-771` — `key[:6] == "xingran:"` 切片长度修正（6 字节比 8 字节字面量恒 false），前缀剥离真正生效；附回归测试

### JOBSTAT — 看板统计缺陷

- [ ] **JOBSTAT-01**（2026-09-06 Phase 96 discuss 重定性：**死代码删除处置**）: `internal/api/v1/job_utils.go` — 分析结论：GetJobStatistics 无任何生产调用方（仅定义 + api_v1_tail_80_03_test.go 测试引用），源自初始脚手架（ea528c6）从未接线路由；生产看板实际走 `/monitor/jobs/logs/statistics` → `jobLogService.Statistics`（全时段统计，无「今日」语义），「生产看板缺陷」前提不成立。处置：整个 `job_utils.go` 删除（GetJobStatistics + 同为死代码的 FormatDuration）连同对应测试，原时区日界缺陷随文件删除消解，不修不测

### BACKUPFIX — config_backup 恢复链加固

- [ ] **V130R-01**: `config_restore_task_service.go:82,163-171` — 超时路径互斥原子化：ExecuteCustom 返回后 worker 不再向设备推送、背对背新恢复窗口消除、RestoreResult 数据竞争修复、10min 总预算分段（方案 phase 规划时敲定：分段子 context 预算或超时后确认任务真终止）；附回归测试
- [ ] **V130R-02**: `network_router.go:45` — RecoverStaleRunningTasks 实例归属过滤：多实例/滚动重启下不误杀其他实例在途任务、不致终态翻转（方案 phase 敲定：grace period >RestoreConfigTimeout 或实例标识列）；附回归测试
- [ ] **V130R-03**: `backup_handler.go:283-284` — 业务错误码语义化：「存在进行中恢复任务」映射 409/400、「备份不属于目标设备」映射 400，不再经 HandleServiceError 统一 500（需 pkg/response 业务错误类型体系，设计 phase 敲定）；附回归测试

### CACHEKEY — 缓存键安全与迁移收尾

- [ ] **V130R-04**: `system/user_cache_impl.go:172-216` + `role_cache_impl.go:51-78` — buildListCacheKey 键值转义或参数集哈希，`Username="bob:status:1"` 与 `Username="bob"+Status=1` 不再碰撞污染；附回归测试
- [ ] **V130R-05**: duty/knowledge/network/workorder 四 cache_impl 残留 11 处 interface{} 闭包 GetOrSet + 4 个平行 getExpiration 迁 base 泛型函数族（cache_invariants_92_test warning 档清零）；附回归测试

### OPSFIX — operations 口径统一

- [ ] **V130R-06**: asset/building List Total 口径收紧（`.Table()` 起链不含软删过滤 → repo `Model(new(T))` 对齐）+ OVR 台账补记 + 软删环境分页器验证；附回归测试
- [ ] **V130R-07**: `floor_service.go:140-178` — 换楼同步乱序修复（乐观条件 WHERE building_id=旧值 或队列串行化）+ First 失败不再静默吞掉 + OVR 补记；附回归测试
- [ ] **V130R-08**: orgId「部门+全部子部门」筛选抽共享 helper 统一四条件口径（workstation/infopoint 三条件形式漏匹配 ancestors 中段修复，6+ 处复制粘贴收敛）；附回归测试
- [ ] **V130R-09**: 分页 clamp 三口径收敛（pagination_helper 10..10000 / requests.GetPagination 10..100 / pkg/query ..200）+ internal/constants 与 pkg/constants 双包合并（consolidation 方案 phase 敲定，base/service.go 注释自认刻意保留处一并处置）；附回归测试

### FEFIX — 前端契约修复

- [ ] **V130R-10**: `rpaApi.ts`（8 处）/`vdiApi.ts`（2 处）工厂 spread 幽灵方法处置——omit 或 apiFactory invariants 增加「新增方法须有后端路由」对照断言（当前零调用方，潜伏 404 面）；附回归测试
- [ ] **V130R-11**: `rpaApi.ts` 大面积后端不存在端点契约对齐专项（scriptApi/scheduleApi/variableApi/templateApi/notificationApi/statisticsApi 全族等）——补路由或裁剪死方法，前后端对账清单落盘；附守卫
- [ ] **V130R-12**: `src/lib/api/networkApi.ts` 平行下载链收敛到权威 download.ts + downloadFilePost 补 JSON 错误体检测（200+JSON 错误不再存成 .xlsx）+ apiFactory invariants readdirSync 改递归扫描；附回归测试

### CLOSEOUT — 收口

- [ ] **TESTFILE-01**: 4 个未跟踪测试文件（`internal/models/rpa/rpa_model_methods_test.go` / `internal/pkg/cache/manager_coverage_test.go` / `internal/pkg/system/sysmetrics_common_test.go` / `internal/pkg/system/sysmetrics_windows_test.go`）入库决策落地——入库补 gate 或明确排除归档（95-02 Pitfall 4 选项 c 口径终结）
- [ ] **UAT62-01**: Migrate176 R1/R2→R5 就地升级 schema 校验回退——带旧结构 MV 的真实 PG 上启动验证（62-HUMAN-UAT 场景 1，归档于 `.planning/milestones/v1.29-phases/` 前身 `.planning/workstreams/milestone/phases/62-ai-internal-core-db/62-HUMAN-UAT.md`）
- [ ] **UAT62-02**: Advisory lock 双实例并发迁移保护——第二实例跳过迁移块 WARN 且正常启动（62-HUMAN-UAT 场景 2）
- [ ] **UAT62-03**: 空库首启 admin 种子凭据告警——默认凭据 WARN / env 覆盖 / salt 非默认（62-HUMAN-UAT 场景 3）

## v1.31+ Requirements (future)

（v1.30 未定义 future 需求；历史候选见各归档 milestone 的 Future 段）

## Out of Scope

| 排除项 | 理由 |
|--------|------|
| WSNOTICE-01（WS 双读者竞态 + origin 前缀绕过） | 已于 2026-09-06 v1.29 深度复查提前修复（503c162 + 6a44659） |
| operlog exclude_paths 白名单 | 独立 deferred pending todo，与缺陷治理不重叠 |
| 新业务功能 | v1.30 锁定为缺陷治理 |
| 前端覆盖率推新目标 | v1.28 已阶段性收口 45.13%（D-04） |
| CACHEDEF/V130R 之外的新扫描发现的缺陷 | 登记新 candidates，不顺手扩scope |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| CACHEDEF-01 | Phase 96 | Pending |
| CACHEDEF-02 | Phase 96 | Pending |
| CACHEDEF-03 | Phase 96 | Pending |
| CACHEDEF-04 | Phase 96 | Pending |
| CACHEDEF-05 | Phase 96 | Pending |
| JOBSTAT-01 | Phase 96 | Delete（重定性删除处置，2026-09-06 discuss） |
| V130R-01 | Phase 97 | Pending |
| V130R-02 | Phase 97 | Pending |
| V130R-03 | Phase 97 | Pending |
| V130R-04 | Phase 98 | Pending |
| V130R-05 | Phase 98 | Pending |
| V130R-06 | Phase 99 | Pending |
| V130R-07 | Phase 99 | Pending |
| V130R-08 | Phase 99 | Pending |
| V130R-09 | Phase 99 | Pending |
| V130R-10 | Phase 100 | Pending |
| V130R-11 | Phase 100 | Pending |
| V130R-12 | Phase 100 | Pending |
| TESTFILE-01 | Phase 101 | Pending |
| UAT62-01 | Phase 101 | Pending |
| UAT62-02 | Phase 101 | Pending |
| UAT62-03 | Phase 101 | Pending |

**Coverage:**
- v1.30 requirements: 22 total
- Mapped to phases: 22（Phase 96: 6 / Phase 97: 3 / Phase 98: 2 / Phase 99: 4 / Phase 100: 3 / Phase 101: 4）
- Unmapped: 0 ✓

---
*Requirements defined: 2026-09-06*
*Last updated: 2026-09-06 — ROADMAP 创建后 Traceability 回填（22/22 → Phase 96-101）*
