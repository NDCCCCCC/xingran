# Phase 101: 收口——测试文件入库 + 62-UAT 人工验证 + audit - Context

**Gathered:** 2026-09-07
**Status:** Ready for planning
**Mode:** Auto-generated（autonomous smart-discuss——phase 无 D-03 设计决策项）

## Phase Boundary

v1.30 收尾：TESTFILE-01 四个未跟踪测试文件入库决策终结 + 62-HUMAN-UAT 3 场景（真实 PG 人工验证项）自动化前置准备与人工执行回写 + 七 gate 全绿 + v1.30-MILESTONE-AUDIT.md 落盘。

## Implementation Decisions

### TESTFILE-01（测试文件入库）

- **D-101-1:** 4 个未跟踪测试文件**全部入库**并纳入常规 `go test ./...` gate——2026-09-07 预验证已确认 3 个所属包全部 PASS（rpa 0.5s / cache 54s / sysmetrics 15s）：
  - `internal/models/rpa/rpa_model_methods_test.go`
  - `internal/pkg/cache/manager_coverage_test.go`
  - `internal/pkg/system/sysmetrics_common_test.go`
  - `internal/pkg/system/sysmetrics_windows_test.go`
- 入库后从"未跟踪测试文件"口径终结（95-02 Pitfall 4 选项 c 悬置解除）。

### UAT62-01..03（62-HUMAN-UAT 3 场景）

源：`.planning/workstreams/milestone/phases/62-ai-internal-core-db/62-HUMAN-UAT.md`（status: resolved，3 场景 pending 转入 v1.30）

- **D-101-2:** executor 负责每个场景的**自动化前置**——可自动部分全部脚本化（准备 SQL/命令/验证步骤清单），人工执行后回写 result：
  - 场景 1（UAT62-01）Migrate176 R1/R2→R5 就地升级：准备"构造 pre-R5 旧结构 MV 的 PG 脚本"（DROP 新列重建旧形态）+ 启动验证步骤 + 日志 grep 断言清单（回退原因日志 / 新列就位 / REFRESH CONCURRENTLY 快路径）
  - 场景 2（UAT62-02）Advisory lock 双实例：准备双实例启动步骤 + `[advisory-lock] 另一实例正在执行启动迁移` WARN 断言 + pgL advisory_locks 无残留检查 SQL
  - 场景 3（UAT62-03）空库首启 admin 种子告警：全自动可跑（空库 + env 开关两次启动 + 日志断言 + salt 非 default 查询）；若本机有真实 PG 则 executor 直接执行并回写
- **D-101-3:** 人工执行项（真实 PG 环境无自动化基建时）保持 `result: [pending]` 并向用户输出验证步骤清单——这是 milestone 唯一合法的 human-gated 项；不可谎报 passed。

### 收口 gate 与 audit

- **D-101-4:** 七 gate 全量实测落盘（go build / go test / 后端 coverage ≥77.5 / 前端 45 dirs / lint 0 errors / type-check / diff coverage ≥80），实测值记入 phase SUMMARY
- **D-101-5:** `.planning/v1.30-MILESTONE-AUDIT.md` 由 milestone lifecycle（gsd-audit-milestone）产出，Phase 101 不重复手写 audit，只准备 audit 输入（requirements traceability 终表）

## Canonical References

- `.planning/workstreams/milestone/phases/62-ai-internal-core-db/62-HUMAN-UAT.md` — 3 场景 expected 定义与回写目标
- `.planning/REQUIREMENTS.md` — 22 requirements traceability（audit 输入）
- `.planning/STATE.md` — 七 gate 基线（v1.29 收口实测）
- 迁移实现：`internal/core/db/migrations/`（Migrate176 MV 就地升级 / advisory lock 块 / admin 种子）

## Existing Code Insights

- 4 个测试文件 2026-09-07 预验证全 PASS（internal/models/rpa、internal/pkg/cache、internal/pkg/system ×2）
- v1.29 收口七 gate 实测基线：coverage 78.33% / lint 0 errors（1389 warnings）/ 前端 554 文件 3800 tests——v1.30 各 phase 执行后 lint warnings 已降至 1385，前端 553/3787+（Phase 100 执行后待实测）
- Migrate176/advisory lock/admin seed 已在 v1.27-v1.29 交付，UAT 是验证而非开发

## Specific Ideas

- 场景 1 的"旧结构 MV 构造脚本"是最有复用价值的自动化前置资产（后续 MV 演进 UAT 可复用）
- 三场景都可用 docker PG（若本机 docker 可用）替代"真实 PG"——先探测环境再定人工/自动边界

## Deferred Ideas

None — 收口 phase 即为终结悬置项。

---

*Phase: 101-closeout-uat-audit*
*Context gathered: 2026-09-07 via autonomous smart-discuss*
