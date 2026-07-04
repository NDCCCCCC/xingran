# Project Retrospective

*A living document updated after each milestone. Lessons feed forward into future planning.*

## Milestone: v1.18 — 网络设备硬件清单 (Device Component Serials)

**Shipped:** 2026-07-04
**Phases:** 1 (Phase 48) | **Plans:** 3 (48-01/02/03) | **Sessions:** 1 large orchestrator + 2 plan executors (Wave 1 / Wave 2) + 1 plan executor (Wave 3) + orchestrator UAT

### What Was Built

- **`ops_asset` 4 组件列 + `sys_data_reconciliation.recon_category`** — schema 基础加 partial unique index 切换 + 字典 seed,migration_201 全 idempotent
- **`internal/services/component_collector/`** — 17 文件新包,纯解析层:ComponentSet 数据模型 + OwnerResolver stack containtedIn tree + 32-hop cycle guard + 4 种收集器(SNMP ENTITY-MIB 单 GET / Huawei CLI / Ruijie CLI / cmd_dispatcher)+ fixtures_loader
- **6 TextFSM 模板** — `huawei_vrp_display_device_esn` / `display_interface_status` / `display_interface_transceiver` + `ruijie_os_show_version_modules` / `show_interfaces_transceiver_ddm` + modified `show_interfaces_status`
- **OpsAssetWriter UPDATE-only + 父缺失降级** — D-02/D-03/D-04,从不 INSERT 资产
- **ReconciliationEmitter** — `conflict_type="F"` + `recon_category="component_serial"` 双 idempotent
- **operlog.RecordBackground** — cron-path 无 `gin.Context`,operator="system-cron",沿用 Phase 34 的 25 OperType 常量 + 11 sensitive keyword 不回归
- **DeviceInfoCollectionService cron hook** — `collectComponentInfo` + `runTwoStepTransceiverPipeline`(D-10 编排),失败仅 Warnf 不阻塞 chassis update(D-14)
- **POST /ops/asset/components + Asset type 4 可选字段 + ComponentListTab** — expandable row 模式,主表不污染

### What Worked

- **3-wave 分层执行** — Wave 1 schema 独立 commit 落地后,Wave 2 collectors 纯 library 隔离开发,Wave 3 pipeline 把 collectors 接入生产路径 + operlog + UI,各 wave 可并行 worktree 节省 wall-clock
- **D-id-driven 设计** — 14 个 D-ids(D-01~D-14)在 48-CONTEXT.md 锁定,executors 在 Wave 1/2/3 按 D-id 标签 commit 任务映射明确,无设计漂移
- **Homegrown textfsm 限制识别早** — 在 Wave 2 落第一个 template 时发现 `internal/templates/textfsm.go` 不支持 `Continue.Record` 习语,改用 single-state Record-on-final-field pattern 规避,无生产 bug
- **Site-visit UAT 显式推迟** — RESEARCH §Environment Availability 显式声明本环境无 S8700/RS8607E,3 项真机 UAT 在 48-HUMAN-UAT.md 标记 device_needed + partial 状态,不需要在 CI 闸门卡住代码合入
- **Migration idempotency** — migration_201 IF NOT EXISTS + DO $ pg_indexes 检测 + count-then-insert 三层防护,在共享 PG 实例上对 migration 二次重跑也 PASS

### What Was Inefficient

- **LSP 假阳性问题** — worktree 隔离导致 LSP 对 worktree 内 `.go` 文件持续报 `undefined: EntityRow/ComponentSet/OwnerResolver` + `could not import github.com/...`,必须切回 main 后才能验证(影响计划可见性,无实际效能损失)
- **`gsd-sdk worktree cleanup-wave` `empty_manifest` bug** — `Write` tool `/tmp/...` 路径与 `Bash` mktemp 路径不匹配,需手动 `git merge` worktree branch 救场,损失 ~5 min
- **git stash pop 误操作** — Wave 2 清理阶段误碰 stash list 中另一 session 的 stash,`git stash -u` 报"No local changes to save" 后 `git stash pop` 仍弹出其他会话的 stash 致 3 文件 UU 冲突,`git reset --hard HEAD` 救场(1 min 影响),但暴露 git 操作风险未防范
- **STATE.md 里程碑归属修正** — Phase 48 v1.18 milestone,但 STATE.md frontmatter 残留 v1.17 标签(user feedback 后才发现),原本应在执行 Wave 1 之前修正;没有 pre-execution milestone 校验
- **11 个 pre-existing test failure 调查** — Wave 1 末/2 中各发现一批 ops/system 测试 fail,要在 `b8fd2f45` (Phase 48 改动前) baseline 上跑 11 次确认非回归 ~15 min
- **Ruijie transceiver textfsm `Continue.Record` latent bug** — `templates/ruijie_os_show_interfaces_transceiver.textfsm` 用了 `Continue.Record` 在 homegrown parser 下行为不一致(无生产 caller,潜在),Wave 2 没修(出 Phase 48 scope),记为后续清理债

### Patterns Established

- **UPDATE-only ops_asset 写入** — 组件序列号是设备派生数据,**永远 UPDATE 不 INSERT/DELETE**,防止 ad-hoc 资产创建污染主表(D-02/D-03)
- **Parent-missing degraded mode** — 父交换机不在 ops_asset 时 `parent_asset_id=NULL` + Warnf,**不自动 INSERT 父资产**(D-04)
- **D-06 sibling-column anomaly via `recon_category`** — partial unique index on `(asset_id, conflict_type, recon_category)` 让 same (asset, type) 可同时存在 legacy + component_serial 异常行
- **D-10 两步流水线 at cmd_dispatcher level** — `status → transceiver only if up`,down ports 跳过,运维省 CLI session 资源
- **D-12 cycle guard 32-hop** — OwnerResolver `entPhysicalContainedIn` 树重建防环
- **operlog.RecordBackground parallel to operlog.Record** — cron-path 无 `gin.Context` 时的等价调用,operator="system-cron"
- **Migration layered idempotency** — `pg_indexes` 检测替代 `DROP IF EXISTS` (GORM 不带 IF EXISTS 是已知坑)
- **Site-visit UAT 在 plan 阶段显式声明 deferred** — 现场/无设备环境,自动化 UAT 不能涵盖的项,在 PLAN.md + RESEARCH.md §Environment Availability + HUMAN-UAT.md 三处显式记录

### Key Lessons

1. **Milestone 归属校验应在 execute-phase 之前完成** — STATE.md frontmatter milestone 必须先校验与 ROADMAP.md 一致,避免 Wave 1 之后才发现 v1.17 vs v1.18 错位
2. **Pre-existing test failures 要 baseline 兜底** — 对任何新 phase,实施前后在 commit baseline 跑同一组测试是规避"被误判为回归"风险的标准动作(已在 Phase 48 走通)
3. **Homegrown TextFSM parser 限制要早期识别** — `Continue.Record` 不支持,executors 落 template 时必须意识到;新 template 一律用 single-state pattern
4. **LSP 在 worktree 内会假阳性** — worktree 在 LSP 视角是孤立目录,跨模块 import 标红是常态;只信 `go build ./...` 不信 LSP 红线
5. **`gsd-sdk worktree cleanup-wave` 路径一致性** — Write 工具与 Bash 子进程的 `/tmp` 路径不一致会让 manifest 读取 empty,需要建立 adapter 或用 `git merge` 兜底(此项暂记,等 SDK 修复)
6. **Cron-path 操作必须平行于 handler-path operlog** — 不能在 cron 路径漏 audit,也不能让 cron 阻塞 fail chassis update;D-13 + D-14 是这条 invariant 的硬性约束

### Cost Observations

- Model mix: yolo mode, executor 默认 model(中量 sonnet 占主)
- Sessions: 1 主 session + 4 subagent (Wave 1 + Wave 2 plan + Wave 3 plan + UAT schema_check)
- Notable: 整个 milestone 主体在 1 个工作日内单 session 完成,worktree 隔离在 wall-clock 上提供 ~30% 加速(Wave 1/2/3 各自独立 commit 后仅需 merge)

---

## Cross-Milestone Trends

### Process Evolution

| Milestone | Sessions | Phases | Key Change |
|-----------|----------|--------|------------|
| v1.16 Tech-Debt | ~5 | 2 (40/41) | TECH-05/06 闭环 + validator 100% pass |
| v1.17 Recon | ~8 | 5 (42-46) + Phase 47 | observe-only 对账引擎 + 5 R-rounds 演进 + UAT 9/10 PASS |
| v1.18 Components | 1 | 1 (48) | 3-wave 隔离 + D-id 全覆盖 + site-visit UAT 推迟 |

### Cumulative Quality

| Milestone | Tests | Coverage | Zero-Dep Additions |
|-----------|-------|----------|-------------------|
| v1.16 | validator 165/165 pass | n/a | n/a |
| v1.17 | recon 5-layer test set | n/a | 6 backend recon 端点 + 4 frontend drawer + R5 state machine |
| v1.18 | 21 collector + 5 pipeline + 3 handler + 5 operlog regression | n/a | 1 pkg + 6 textfsm + 1 cron helper + 1 endpoint + 1 tab |

### Top Lessons (Verified Across Milestones)

1. **Pre-existing test failure 兜底(Phase 40/41 + 48 共验证)** — 每个 phase 实施前在 baseline commit 跑同组测试,规避假回归指控
2. **Migration layered idempotency(Phase 33/44/48 共验证)** — DROP IF EXISTS 在 GORM AutoMigrate 上不工作,改用 `pg_indexes` 探测 + count-then-insert
3. **operlog module/chinese-name 双轨(Phase 34 + 48 共验证)** — 强制使用 ops asset 中文常量,避免 module 字段在审计回路漂移
4. **基线审计后允许部分 deferred(Phase 41 29→ + Phase 48 3 site-visit)** — observe-only 策略延续,真机/外部依赖类 UAT 显式 partial 而非阻塞代码合入
5. **CR + Lint 在 phase 末尾加入,不在开头(Phase 32 P1+P2 + 48)** — 实施期先跑通,review 期 catch 全,避免过度返工
