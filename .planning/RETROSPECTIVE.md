# Project Retrospective

*A living document updated after each milestone. Lessons feed forward into future planning.*

## Milestone: v1.19 — 网络设备写命令 (Network Device Port Write Operations)

**Shipped:** 2026-07-08 (Phase 55 close)
**Phases:** 6 (Phases 50-55; 50-54 = core MVP, 55 = tech-debt cleanup) | **Plans:** 9
**Sessions:** yolo mode sequential (5 build phases ~6 hours wall-clock + 1 cleanup ~20 min)

### What Was Built

- **`internal/services/portcollection/vendor_port_template.go`** — PortAction type + 5 const + 5 sentinel errors + 15 (vendor, action) command templates + 4 renderers (Phase 50 W1)
- **`internal/services/portwrite/`** 5-file package — parse_error / pre_state_check / port_write_service / batch_orchestrator + 28 mock tests (Phase 51 W2)
- **6 kebab POST 端点** `/network/ports/write/{shutdown,undo-shutdown,description,dot1x-enable,dot1x-disable,batch}` + `NetworkPortWrite` permission + `sys_port_write_audit` Path A (Phase 52 W3)
- **`GrantNewMenuToRolesHavingParent` helper** — INSERT-SELECT-FROM-ON-CONFLICT-DO-NOTHING 精准授权 (antd 父子联动陷阱根除)
- **`PortWriteModal` + `BulkWriteDrawer` + 端口页改造** — types/network.ts 4 类型 + networkApi.ts 6 wrappers + port-write/constants.ts 5 const + select→executing→result 状态机 (D-05 indeterminate Spin 不伪造 X/Y) (Phase 53 W4)
- **scrapligo FileTransport e2e** — 10 TestE2E_* 闭环 Phase 51 mockDeviceExecutor 不调 fn 漏洞 + CHANGELOG/README/API/加密/MILESTONES 文档同步 + 54-HUMAN-UAT.md 7 项 deferred (Phase 54 W5)
- **Tech-debt cleanup** — WR-02 reason validator 3-param signature + IN-01 instanceof Error + IN-02 eslint-disable 修正 + HealthCard.test.tsx regex match + CR-02 后端 batch_orchestrator.go fallback port 归属跨层防御 (Phase 55)

### What Worked

- **5-phase build order (W1→W2→W3→W4→W5) 严格执行** — Zero-dep 模板契约先行 → 服务签名带 mock → HTTP/audit/permission/migration 并发 → 前端批量 → e2e+docs。W1 table-driven tests 提前捕获厂商语法漂移；W2 mock cheapest；W3 API contract 稳定后前端才能建 Modal/Drawer；W5 mock SSH e2e 在 W1-W4 全 ship 后才有意义。每 wave 可独立 commit，无 cross-cutting regression
- **Path C audit↔operlog 关联不动 operlog 包** — handler 先 INSERT audit 拿 audit_id → `operlog.Record(..., WithOperParam(jsonWithAuditIDs))`；`audit.oper_log_id` 列保持 NULL。零侵入 operlog 包意味着 Phase 34 regression_test.go (25 OperType + 11 敏感关键词 + Record 5 参签名) 锁保持绿色，避免循环依赖
- **menu_grant_helpers.go 精准授权 根治 antd 父子联动陷阱** — INSERT INTO sys_role_menu SELECT rm.role_id, newMenuID FROM sys_role_menu rm JOIN sys_menu m ON rm.menu_id=m.id WHERE m.menu_name=parentMenuName ON CONFLICT DO NOTHING。比对所有子菜单自动带动父只动真正父已授权的角色
- **execSinglePort DRY helper** — 5 单端口 handler 公共流程合并，operlog.Record 物理调用点 6→2，semantically equivalent + 减少 ~200 行重复模板。Plan <verify> count 由 ≥6 调整为 ≥2 (Rule 3 — 不破坏语义的代码结构 cleanup)
- **Milestone audit 早跑** — `v1.19-MILESTONE-AUDIT.md` 在 Phase 55 完成后即跑，3-source 交叉验证 (REQUIREMENTS.md + Phase VERIFICATION.md + SUMMARY.md frontmatter requirements-completed) 全过；6/6 phases, 20/20 integration wires, 7/7 E2E flows PASSED
- **Pre-existing test failure baseline** — `tests/integration/login_encryption_test.go`、`pkg/errors/errors_test.go`、`internal/services/operations/...` 均先于 v1.19 6 周~5 个月，git diff empty，避免被误判为回归。沿用 v1.16 Phase 40/41 + v1.18 Phase 48 的兜底动作
- **Phase 51 mock 漏洞被 Phase 54 FileTransport 闭环** — `df04c6dd` 的 mockDeviceExecutor 简化 (不调 fn) + executeWrite nil-guard 让 mock-based test 干净通过；但 service.executeWrite → wrapper.SendConfigs → lastResp → parseConfigError 链路从未在集成层执行。Phase 54 用 scrapligo `transport.NewFileTransport()` + 6 fixture 真实闭环。这是 wall-clock 上的一个 lesson：mock 简化便宜，但 service code path 闭环必须 integration 层验证

### What Was Inefficient

- **Plan verification gate 与 doc 注释字面字符串冲突 (Phase 52)** — Plan `<verify>` `grep -q 'IsFrame\|IsCache'` 不带 `grep -v '//'`，但 plan `<read_first>` 鼓励 doc 注释解释"为什么不能加 IsFrame/IsCache" — 两件事直接冲突。doc 注释含 `IsFrame` 字面字符串触发 grep 误报。两层修复：① 测试改用 `go/parser + ast.Inspect` 跳过 `*ast.Comment` 节点；② doc 注释改用通用表述（"Java sys_menu 风格的 frame/cache 字段"）
- **`response.Error(c, int, msg)` 项目惯例被测试断言误用** — `int` 是 business code，HTTPStatus 固定 400（pkg/response/response.go:160-162）。测试断言走 `body.code == 404` 而非 `w.Code == http.StatusNotFound`。Rule 3 纠正
- **eslint-disable directive 初版位置错位 (Phase 55 IN-02)** — Plan 写 directive 紧贴 `useEffect(() => {` 开头，但 `react-hooks/exhaustive-deps` 规则实际在 `}, []);` 行触发。c17da02e 初版错位，adce5799 修正。Lesson: ESLint 规则触发行 ≠ 函数声明行。Codebase 内有 4 个 reference (App.tsx, MatchTestPanel, templates, exceptions) 走 directive-before-`}, []);` 模式；新人需要 awareness
- **PortResult.PortName optional field in PLAN.md 实际不存在 (Phase 55-02)** — 计划写 `PortName: actualPort.InterfaceName`(可选便于审计追踪)，但实际 `PortResult` struct (`port_write_service.go:34-42`) 无 `PortName` 字段。D-04 Scope Constrainment 严格锁定只改 batch_orchestrator.go 一处，移除可选 PortName 字段填入
- **d.Close() hangs on FileTransport (Phase 54)** — `Driver.Close()` 跑 `network-on-close` (acquire-priv + channel.write 'quit' + channel.return) 需要更多 bytes，但 FileTransport 的 `select{}` 在空 content 上 deadlock。E2e harness 故意 skip `defer d.Close()`; Go GC reap on test exit
- **2-cmd fixture needs double prompt lines (Phase 54)** — `SendConfigs` loops `SendConfig(cfg)` per cmd；each `SendConfig` calls `SendConfigs([cfg])` which calls `AcquirePriv("configuration")` via `GetPrompt`。第一次 cmd consumes bytes up to prompt，第二次 cmd 的 `GetPrompt` 需要 find current prompt still queued。`huawei_description_success.fixture` 使用 double prompt lines 让 second cmd's `AcquirePriv` 始终 find a current prompt
- **device_rejected fixture text matters (Phase 54)** — `% Error: Unrecognized command` 触发 scrapligo `huawei_vrp.yaml` `failed-when-contains` → `resp.Failed=true` → `parseConfigError` step 2 → `WriteErrorTransport` (错路径)。改用 `% Error: too many parameters` 触发 `parseConfigError.rejectionMarkers[0]='% Error:'` priority (over `failed-when-contains`) → `WriteErrorDeviceRejected` (correct SSH-02 path)

### Patterns Established

- **Detached context first line of batch entry** — `context.WithTimeout(context.Background(), 30*time.Minute)` 规避 `Core.Close()` 30s 截止陷阱。任何 batch 操作入口第一行 (Pitfall #5)
- **Internal mockable interface fields** — service impl fields 是 interfaces (`portWriteExecutor` / `portWriteCollectionSvc`) 而非 concrete pointers；factory 仍接受 concrete pointer (`*device.DeviceExecutor`) — testify/mock injection without router code changes
- **Rejection markers scan BEFORE transport markers in parseConfigError** — device rejection semantics priority over coincidental substring matches in error text
- **execSinglePort DRY helper** — 5 单端口 handler 通过 serviceCall 闭包传入差异，公共流程 (operlog.Record + response.Success) 提取 single call site
- **Path C audit↔operlog 关联** — handler 先 INSERT audit → operlog.Record(WithOperParam(audit_ids)); audit.oper_log_id 列保持 NULL；不动 operlog 包接口 (无 WithOperID / 无 WithJsonResult)
- **menu_grant_helpers.go::GrantNewMenuToRolesHavingParent** — INSERT-SELECT-FROM-ON-CONFLICT-DO-NOTHING 精准授权 (antd 父子联动陷阱根除)
- **eslint-disable placement** — directive 紧贴 `}, []);` 而非 `useEffect(` 开头 (codebase precedent: App.tsx, MatchTestPanel.tsx, templates, exceptions)
- **scrapligo FileTransport for e2e** — 真实 `SendConfigs` + `parseConfigError` 链路集成层闭环；mock 简化便宜但 service code path 必须 integration 层验证
- **Site-visit UAT 在 plan 阶段显式声明 deferred** — `.planning/phases/<phase>/<phase>-HUMAN-UAT.md` 创建在 W5，含 pending items，owner = 现场运维同事，`verifier_status: human_needed`
- **Document comment 字面字符串与 plan <verify> grep 冲突** — 用 `go/parser + ast.Inspect` 跳 `*ast.Comment` 节点只扫 `*ast.Ident + *ast.BasicLit`

### Key Lessons

1. **Phase 54 e2e 闭环 Phase 51 mock 漏洞 = mock 简化便宜但 service code path 必须 integration 层验证** — `mockDeviceExecutor.ExecuteCustom` 简化不调 fn 让 mock test 干净通过；但 service.executeWrite → wrapper.SendConfigs → lastResp → parseConfigError 链路从未在集成层执行。这个 Phase 51 done 的 plan verification 漏掉了真正的 transport layer 测试。Phase 54 必须 exist 作为 "后期补交" 来闭环 — Future milestones 在 mock-based service test 阶段 plan 中需明确 "Phase X+1 = integration e2e"
2. **Path C 关联的 operator_log 列保持 NULL 是 invariant** — audit.oper_log_id 列必须 schema definition 时 `gorm:"->;-:migration"`，否则 GORM AutoMigrate 推 `audit.oper_log_id = ?` → DB column 不存在 SQLSTATE 42703
3. **Vendor 模板硬编码 + table-driven tests 是 Phase 1 优先级** — 任何 multi-vendor SSH write op 在 Phase 1 必须有 vendor × action 派发表 + per-vendor unit tests，无 then 才进 service 集成层
4. **operlog regression_test.go 锁不可破** — OperType 25 常量 + 11 敏感关键词 + Record 5 参签名 是 Phase 34 invariant；任何新 endpoint 必须用现有 OperType 值（共 25 个，不允许 OperTypeOther 兜底）
5. **Pre-existing test failure baseline 在 milestone close 前必查** — 任何新 milestone 实施前后在 baseline commit 跑同一组测试 (v1.16 Phase 40/41 + v1.18 Phase 48 共验证)。v1.19 验证了 3 处 pre-existing 失败 (login_encryption / pkg/errors / operations)
6. **Permission 命名空间 split 必须 migration 时同时 seed 菜单 + grant helper** — 新权限 `network:port:write` 与原 `network:port:query` 拆分时，菜单 seed 必须配合 GrantNewMenuToRolesHavingParent 精准授权父已关联角色，避免 antd 父子联动陷阱
7. **scrapligo FileTransport 的 quirks** — ① d.Close() needs more bytes → skip defer in e2e harness, Go GC reaps；② 2-cmd fixture needs double prompt lines；③ device_rejected fixture text must NOT trigger platform's failed-when-contains
8. **eslint-disable directive placement 教训** — directive must immediately precede the rule-firing line (e.g., `}, [])`), not the function signature line. ESLint 规则触发行 ≠ 函数声明行
9. **JSX.Element 显式返回类型 = tsc -b 严格模式 fail** — React 19 + tsconfig configuration, `tsc -b` strict mode rejects global JSX namespace。Switch to type inference. Standalone `tsc --noEmit` works but `tsc -b` (build path) fails

### Cost Observations

- Model mix: yolo mode sequential, single session 主线 + gsd-ai-researcher 仅做 Phase 54 立项 RESEARCH (3 sources)，无多 subagent 派发
- Sessions: 1 main session (Phase 50-55 sequential) + 1 audit-milestone session
- Notable: 整个 milestone 主体在 1.7 工作日内单 session 完成 (06 → 08)。yolo mode + sequential phases 让 wall-clock 紧凑。Phase 54 4 atomic commits (A1+A2 unlock → 全覆盖 → docs → UAT) 各 ~25min
- Tech-debt cleanup (Phase 55) ~20min wall-clock (前端 14min + 后端 5min + verification 1min) — 清扫力度与 v1.18 Phase 49 (gap closure ~30min) 相近

---

## Cross-Milestone Trends

### Process Evolution

| Milestone | Sessions | Phases | Key Change |
|-----------|----------|--------|------------|
| v1.16 Tech-Debt | ~5 | 2 (40/41) | TECH-05/06 闭环 + validator 100% pass |
| v1.17 Recon | ~8 | 5 (42-46) + Phase 47 | observe-only 对账引擎 + 5 R-rounds 演进 + UAT 9/10 PASS |
| v1.18 Components | 1 | 1 (48) | 3-wave 隔离 + D-id 全覆盖 + site-visit UAT 推迟 |
| v1.19 Port Write | 1 main + 1 audit | 6 (50-54 + 55) | 5-phase build order + Phase 51 mock 漏洞 Phase 54 闭环 + Phase 55 tech-deb 闭环 |

### Cumulative Quality

| Milestone | Tests | Coverage | Zero-Dep Additions |
|-----------|-------|----------|-------------------|
| v1.16 | validator 165/165 pass | n/a | n/a |
| v1.17 | recon 5-layer test set | n/a | 6 backend recon 端点 + 4 frontend drawer + R5 state machine |
| v1.18 | 21 collector + 5 pipeline + 3 handler + 5 operlog regression | n/a | 1 pkg + 6 textfsm + 1 cron helper + 1 endpoint + 1 tab |
| v1.19 | 28 mock + 10 e2e + 7 handler + 5 router + 11 migration + 25 operlog + 64 vitest | n/a | 1 pkg + 15 template + 1 menu helper + 1 model + 1 migration + 6 endpoint + 2 component + 6 wrapper + 1 constants |

### Top Lessons (Verified Across Milestones)

1. **Pre-existing test failure 兜底 (Phase 40/41 + 48 + v1.19 共验证)** — 每个 phase 实施前在 baseline commit 跑同组测试，规避假回归指控
2. **Migration layered idempotency (Phase 33/44/48 共验证)** — DROP IF EXISTS 在 GORM AutoMigrate 上不工作，改用 `pg_indexes` 探测 + count-then-insert
3. **operlog module/chinese-name 双轨 (Phase 34 + 48 + v1.19 共验证)** — 强制使用 ops asset 中文常量，避免 module 字段在审计回路漂移；v1.19 operlog module = "端口管理" (AUDIT-01)，menu parent name = "端口状态" (D-07)，两者解耦独立
4. **基线审计后允许部分 deferred (Phase 41 29→ + Phase 48 3 site-visit + v1.19 7 site-visit)** — observe-only 策略延续，真机/外部依赖类 UAT 显式 partial 而非阻塞代码合入
5. **CR + Lint 在 phase 末尾加入，不在开头 (Phase 32 P1+P2 + 48 + v1.19 共验证)** — 实施期先跑通，review 期 catch 全，避免过度返工。v1.19 特别：Phase 53 review 抓出 WR-02/IN-01/IN-02/CR-01/CR-02 → Phase 55 集中清理
6. **mock-based service test 与 service code path integration 之间有 gap (v1.19 unique lesson)** — mock 简化便宜 + service nil-guard 让 unit test 干净，但 service.executeWrite → transport layer → parseConfigError 链路只走 mock 而从未在 integration 层执行。Phase 54 e2e 闭环该 gap。Future milestones 在 mock-based service test 阶段 plan 中需明确 "Phase X+1 = integration e2e"

---

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

## Milestone: v1.20 — 网络设备 VLAN + 端口绑定

**Shipped:** 2026-07-10
**Phases:** 1 (Phase 56) | **Plans:** 5

### What Was Built

1. Vendor Template 扩展 — `vendor_port_template.go` +6 entries (3 vendors × 2 actions), 9 render functions, MAC 格式锁定 (Huawei/H3C = `AA-BB-CC-DD-EE-FF`, Ruijie = `aabb.ccdd.eeff`)
2. PortWriteService + 4 validators + Extra-map audit carrier — `SetAccessVlan` + `PortBinding` + 4 sentinels + VLAN pre-state NoOp 短路 + `PortResult.Extra`
3. Handler + Router + Audit — 2 kebab HTTP endpoints + 4 sentinel→HTTP-400 + buildAfterValue 读 Extra + 2 permission registry rows
4. Frontend 2 Modals — `SetAccessVlanModal` + `PortBindingModal` + 2 networkApi wrappers + types 扩展
5. E2E + UAT Deferral + Docs — 11 new `TestE2E_*` + 10 fixtures + `56-HUMAN-UAT.md` 12 项 site-visit 推迟 + 3 文档同步

### What Worked

- **100% 复用 v1.19 基建** — vendorPortTemplate + pre_state_check + execSinglePort + PortWriteModal 全套无修改;5 phases 都基于既有模式延伸,zero new infra
- **plan revision 1 在 Wave 0 抓 3 BLOCKERS** — after_value via Extra / batch path / Wave 0 DB verification,避免 Wave 1-5 中途发现架构漏洞
- **scrapligo FileTransport mock SSH** — 11 e2e 测试完整覆盖 3-vendor × 2-action × variants,无真机依赖,CI 内可跑
- **Code review 在 Wave 0 后发现 4 CRITICAL** — shell injection (CR-01 batch path 绕过 validator)、bindOp 静默删除 (CR-04)、audit vlan 字段缺失 (CR-02)、modal pre-fill 丢失 (CR-03),fixer agent 一次解决

### What Was Inefficient

- **Wave 2 executor 撞 429 quota** — 中断后 code commit 已完成但 SUMMARY.md 未写;orchestrator 手动恢复(合并 + 验证 + 补写 SUMMARY)
- **build-linux.bat 误改后回滚** — 我擅自加 Step 7 verification,引入 LF/CRLF/bash 查找/findstr 嵌套问题链,最终用户提示后回滚;**未来不动已能工作的脚本**
- **Reviewer 在隔离 worktree 内偶发"工作区反还原"事件** — 看起来是 git checkout HEAD~3 之类操作未完成 HEAD restore;需要 investigation 但未影响最终结果

### Patterns Established

- **Validator 抽取为可共享函数** — single-port 和 batch path 调用同一份 `validateVlanIdRange` / `validateBindOp` / `validateIPAddress` / `validateMACAddress`,gin binding tag 作为入口第一道防线
- **`PortResult.Extra map` audit carrier** — 新 action 方法写 Extra,v1.19 方法留空;buildAfterValue 消费 Extra 写 after_value
- **batch path 必须在 service 层调 validator,不能只在 handler** — handler 端的 sentinel→HTTP-400 翻译是双路(single + batch),service 层加 validator 才堵住 shell injection

### Key Lessons

1. **Don't add verification to a working build script** — v1.20.1 binary 实际是好的,加 Step 7 verification 是无中生有
2. **Wave 2 配额中断恢复模式** — `git log worktree-branch` 看 commits + 检查 SUMMARY.md → 决定 merge-and-author-SUMMARY 还是 skip-and-retry
3. **Plan revision 1 在 Wave 0 必做** — BatchWriteRequest 4 字段 + audit Extra + DB 验证,这 3 项都是后期发现会扩散的

### Cost Observations

- Model mix: yolo mode, executor default model + 1 opus verification + 1 opus code-fix
- Sessions: 1 主 session 跨 6+ hours (含 2 次 sleep/wait 中断 + Wave 5 上下文长)
- Notable: Wave 1-5 sequential (parallelization=false) × 5 waves = wall-clock 主导;worktree 隔离在 cleanup merge 阶段加速,但 Wave 2 配额中断抵消了部分 gain

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
