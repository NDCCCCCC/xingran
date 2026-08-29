# Phase 81: 全仓收口·ratchet 闭环与 milestone audit — Research

**Researched:** 2026-08-28
**Domain:** 覆盖率治理收口(ratchet / gate / audit)— 内部状态测量,非新代码领域
**Confidence:** HIGH(全部结论来自本会话实测命令与 file:line 引用,无外部依赖)

<user_constraints>

## User Constraints (from ROADMAP.md L281-303,本 phase 无 CONTEXT.md)

### Locked Decisions(Phase 定义即约束)

- **Goal**: 全量重测把 `.coverage-threshold` 从 55.5 ratchet 到 ≥70 实测值、删除 P2_RATCHET 豁免行回落全量 70% floor、milestone audit 定案,gate 防线全程不倒退。
- **Depends on**: Phase 75/76/77/78/79/80 全部完成(已核实:34/34 plans 全部有 SUMMARY,STATE.md:6 "Completed Phase 80")。
- **Requirements**: GATE-01, GATE-02, GATE-03。

### Success Criteria(ROADMAP L289-293 原文)

1. 全量 `go test -coverprofile` 重测:加权平均 ≥70%(43652 stmts 口径,SC-a 数学 30556/43652 达成),`.coverage-threshold` 55.5 → 新实测值(UP-only 语义)
2. check-coverage.sh 删除 core/device/agent-server 三条 P2_RATCHET 豁免行,P2 floor 回落全量 70%,删除后 gate 本地 + CI 实跑绿(UP-only 闭环:豁免只减不增)
3. 4 层 gate(weighted-avg / P1 floor exit 4 / P2 floor exit 5 / PR diff coverage ≥80%)在收口 commit 上 CI 全绿,且 milestone 全程无 gate 倒退记录
4. milestone audit 报告落盘:19/19 需求核验、15 项 QUIRK 关闭清单、SC-a..e 证据链、v1.26 SC-a 缺口(6287 stmts)收口数学

### ROADMAP Notes(硬约束)

- 镜像 v1.26 的 74-11 收口:atomic ratchet commit 模式(D-07 六文件先例:threshold + baseline + check-coverage.sh + STATE + ROADMAP + SUMMARY)
- 若个别包最终 <70%(如平台绑定的 pty 路径),豁免必须在 audit 显式文档化并保留对应 P2_RATCHET 行——**不允许静默豁免**

### Deferred / Out of Scope

- BLOCK-05 本 phase 只做**裁决框架**,不做 ldap_client BER 接线实现(见 §BLOCK-05)
- v1.28 frontend-coverage workstream(Phases 82/83/88)不在本 phase 范围,但其提交与本 phase 共享分支(见风险 R7)

</user_constraints>

<phase_requirements>

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| GATE-01 | 加权平均 ≥70%(43652 stmts 口径;SC-a 收口) | 实测已达 **78.01%**(34112/43729,gate exit 0);threshold bump 机制、取值口径(1-decimal 截断先例)与刀口风险见 §Ratchet Mechanics |
| GATE-02 | core/device/agent-server 达标后删除 P2_RATCHET 行,回落 70% 全量 floor | 三包实测 **82.54% / 82.77% / 90.59%**,距 70% 有 12.5-20.6pp 余量;删除点为 `.github/scripts/check-coverage.sh` L242-244 + 5 处注释,见 §Ratchet Mechanics |
| GATE-03 | 4 层 gate + PR diff coverage 全程绿;无 gate 倒退记录 | 本地 4 层全 PASS 实测;**199 个 commit 未推送**是 SC-2/SC-3 CI 验证的前置阻塞,见 §Current State / §Risk R3 |

</phase_requirements>

## Summary

Phase 81 是纯测量/治理收口 phase:不写生产代码,核心工作是「一次干净的全量重测 → threshold 与 gate 脚本的机械修订 → 一份审计文档」。本研究的核心实测结论:**v1.27 的数字目标已经全部达成且余量巨大** —— 全仓加权平均实测 **78.01%**(34112/43729 stmts,gate exit 0,本会话 2026-08-28 全量跑),三条 P2_RATCHET 豁免包实测 internal/core **82.54%**(624/756)、internal/device **82.77%**(1047/1265)、internal/agent/server **90.59%**(568/627),全部远超 70%,删除豁免行的本地机械风险接近零。

但有三个非显而易见的阻塞/风险必须进 plan:**(R1)** 全量跑在满载下暴露一个预存 flake —— `internal/scheduler` 的 workorder 任务测试报 `no such table: sys_duty_schedule`(`workorder_tasks.go:196`),单包复跑 9.3s 通过、满载 74.7s 失败,属已知「sqlite 缺表 family pattern」的满载形态,不修掉它 SC 的「零失败全量重测」与收口 CI 绿都无法成立;**(R3)** 本地 main 领先 origin/main **199 个 commit**(`git rev-list --left-right --count origin/main...HEAD` = `0 199`),Phase 77-80 的全部后端覆盖工作 CI 从未见过,SC-2 的「删除后 CI 实跑绿」与 SC-3 的「收口 commit CI 全绿」都被这次 push 卡着;**(R5)** stmts 分母在三套数字间漂移(43652 v1.26 口径 / 43893 Phase 79 后 / 43729 本次实测,仪器漂移 + 新增生产 stmts),audit 的 SC-a 数学必须显式声明口径。

BLOCK-05(addomain)实测维持 **58.00%**(1403/2419):到 70% 需再覆盖 **291 stmts**,而 BER 不兼容锁死的 ldap_client 不可达 stmts 只有约 230(Search ~80 + write ~100 + paging ~50,78-07-SUMMARY L76-78),即使 responder 立刻可用仍有 ~61 stmts 缺口要从别处补。关键机制事实:**addomain 不在任何 P1/P2 floor 名单里**(check-coverage.sh L164/L228),它对 4 层 gate 零影响 —— 把 addomain 算 0%,加权平均仍有 74.8%。因此裁决的真实形状是「文档语义」而非「gate 风险」:Phase 80 的 lldp 68.8% 豁免(80-05-SUMMARY L47/L112,executor 依赖真实 device 基建,~2 stmts 差距)已提供完整先例格式,且 v1.27 Future Milestones 里已预埋「CI-only smoke 层(真 OpenLDAP 报文,`//go:build smoke` + 本地自动 skip)」作为 spin-off 落点。

**Primary recommendation:** 3-plan 顺序切分(81-01 flake 修复 + 绿色重测 + ratchet;81-02 豁免行删除 + 本地 4 层验证 + push + CI 盯守;81-03 audit 定案 + 文档债清偿),threshold 新值建议带 0.5pp 缓冲(实测 78.01 → 写 77.5)而非严格截断 78.0(先例是截断,但本次 CI 从未见过这批代码,0.01pp 刀口不值得赌),BLOCK-05 建议「lldp 式豁免文档化 + Future Milestone 指针」结案。

## Architectural Responsibility Map

本 phase 不改运行时行为,职责按「治理能力」映射:

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| 覆盖率测量(全量 profile) | CI 工具层(`go test -coverprofile`,本地 = `scripts/check-ci-local.sh`) | — | 测量口径必须与 ci.yml L64-66 命令 byte-identical,否则阈值不可比 |
| 阈值/floor 治理(ratchet) | 仓库配置层(`.coverage-threshold` + `.github/scripts/check-coverage.sh`) | — | UP-only 语义由脚本 + D-04 流程共同承载,无运行时组件 |
| CI 强制执行(4 层 gate) | CI 层(`.github/workflows/ci.yml` backend + coverage-diff job) | — | ci.yml 已经接好,Phase 81 **不需要改 workflow**(74-11 先例:per-ratchet 零 workflow 编辑,L27-31) |
| Milestone 定案(证据链) | 规划文档层(`.planning/milestones/` + workstream ROADMAP/STATE/REQUIREMENTS) | — | 纯文档,镜像 74-MILESTONE-AUDIT 结构 |

## Standard Stack

无新增库。本 phase 全部工具已就位(实测验证):

| 工具 | 版本/状态 | 用途 | 验证 |
|------|-----------|------|------|
| go | 1.24.5 windows/amd64(go.mod: go 1.24.0 + toolchain go1.24.5) | 全量重测,与 CI `go-version-file: go.mod` 同 toolchain 线 | [VERIFIED: `go version`] |
| bash + awk | Git Bash 自带 | gate 脚本零依赖(D-01) | [VERIFIED: gate 实跑 exit 0] |
| `scripts/check-ci-local.sh` | 已存在 | 本地 CI 模拟(全量 test + gate + `--diff` 差量 gate),81-01/81-02 的验证入口 | [VERIFIED: L73-85] |
| gh CLI | 可用 | CI 盯守(`gh run list/view`;memory 注:`gh run watch` 提前退出不可靠,用轮询 `gh run view`) | [VERIFIED: `gh run list` 返回 5 条] |

**Package Legitimacy Audit: N/A** —— 本 phase 零新依赖(miniredis/httpmock 等已在 Phase 76 引入并入库)。

## Current Measured State(2026-08-28 本会话实测)

### 测量命令与完整性

```
go clean -testcache
go test -timeout 20m -count=1 -coverprofile=cov_full_research.out -covermode=atomic \
  ./internal/... ./pkg/... ./cmd/...
# 结果: 70 ok / 1 FAIL(internal/scheduler)/ EXIT=1,全量墙钟 ~11 分钟
# profile: cov_full_research.out 2.84MB;internal/services 单包 553s 是长尾
bash .github/scripts/check-coverage.sh cov_full_research.out .coverage-threshold
# GATE_EXIT=0(因 scheduler 包仍写入 81.4% coverage,数字有效;但正式 ratchet 必须出自绿色跑)
```

[VERIFIED: 本会话后台运行 + gate 实跑]

### 4 层 gate 现状(P1/P2 全 PASS)

| Gate 层 | 现值 | 阈值 | 结果 |
|---------|------|------|------|
| 加权平均 | **78.01%**(34112/43729) | 55.5(现行) | PASS(exit 0) |
| P1 floor × 8(duty/knowledge/rpa/vdi api + duty/knowledge/network/monitor services) | 83.02 / 84.25 / 79.25 / 76.17 / 95.61 / 95.29 / 92.1x / 95.x | 70.0 | 8/8 PASS |
| P2 floor × 10(7 标准 + 3 ratcheted) | 最小 operations 72.30%;ratcheted:core 82.54 / device 82.77 / agent-server 90.59 | 70.0 / 39.00 / 38.50 / 19.00 | 10/10 PASS |
| PR diff coverage ≥80% | PR-only,本 phase 文档/脚本 commit 无可测 .go 行 → skip(exit 0) | 80 | N/A→PASS |

[VERIFIED: `/tmp/gate_out.txt` 全量输出]

### 三条 P2_RATCHET 豁免行(check-coverage.sh L242-244,逐字)

```bash
P2_RATCHET_internal_core="39.00"          # CI 39.50 / local 40.2
P2_RATCHET_internal_device="38.50"        # CI 38.91 / local 39.07
P2_RATCHET_internal_agent_server="19.00"  # CI 19.48 / local 22.08 (env branches)
```

**注意:任务简报里引用的 "core 38.33 / device 39.07 / agent-server 22.08" 是 Phase 74 的测量值(注释与 STATE.md:21),不是现行 floor 值;现行 floor 是 39.00 / 38.50 / 19.00(PR #4 round-5 教训:floor 是带 ≥0.4pp 余量的保守下界,check-coverage.sh L230-241)。** 删除时三处测量注释一并删。

删除的完整清单(不止 3 行):
1. L242-244 三条赋值(上)
2. L39-45 头注释 "3 packages are structurally blocked..." 段
3. L218-225 section 4 注释里 ratcheted 说明 + L230-241 方法论注释
4. L300-304 失败信息里 "3 structurally-blocked packages at UP-ONLY ratcheted floors" 措辞
5. L308 尾行 `"70.0% x 7 + ratcheted x 3 = 10 packages"` → `"70.0% x 10 packages"`
6. `floor_of()`(L246-254)与 `P2_FLOOR`(L227)**保留**——回落机制本身是 uniform 70

删除条件先例(L225/L241 逐字):"Removal condition unchanged: package crosses 70.0% -> delete its entry." 三包全部满足。

### 结构性 0% 包(4 个,172 stmts,不可避免)

`cmd` 106 / `cmd/agent` 63 / `internal/server` 2 / `internal/docs` 1 —— main/装配入口,不计 floor 名单,只进加权平均分母。audit 应记录:78.01% 已含这 172 stmts 的 0% 拖累。

### 分母漂移口径(必须写进 audit)

| 时点 | total stmts | covered | weighted | 来源 |
|------|------------|---------|----------|------|
| v1.26 起点(2026-08-20) | 43652 | 5589 | 12.8% | coverage-baseline.md L13 |
| v1.26 收口 / v1.27 SC-a 口径 | 43652 | 24254(需 30556 = 70%) | 55.5% | L443 |
| Phase 79 后(2026-08-28) | 43893 | 31119 | 70.9% | L510 |
| **本会话实测(81 前夜)** | **43729** | **34112** | **78.01%** | gate PACKAGE 行 |

漂移来源:Go patch 版本插桩漂移(先例:同码 754 vs 767 stmts,±1.7%)+ Phase 76/77 少量生产重构 stmts(INFRA-02/04 注入缝)。**SC-a 数学应统一为公式 `covered_needed = ceil(0.7 × 当前分母)` 并同时呈现 43652 口径的 v1.26 承诺数**;本次实测 surplus = 34112 − 30611(= ceil(0.7×43729))= **+3501 covered stmts 超线**。v1.26 缺口收口数学:24254 → 34112,补了 **9858 covered**,是 6287/6302 缺口的 **~1.56 倍**。

## Ratchet Mechanics(81-01/81-02 的操作面)

### threshold 文件事实

- 现值 `55.5`,**4 字节,无换行**(`xxd`: `3535 2e35`)。脚本用 `THRESHOLD="$(cat ...)"` 读取后 awk `threshold + 0` 比较,有无 LF 均安全(L75,L123)。
- UP-only 是纪律不是代码:脚本没有「新值必须 ≥ 旧值」检查,靠 D-04 手动 ratchet 流程 + baseline 行约束。plan 的验证步骤应含「git diff 确认单向上调」。

### 取值先例(回答 DQ1)

- Phase 72: 21.5(实测 21.55);Phase 73: 25.9(实测 25.97 截断,STATE.md:242 逐字 "阈值取实际测量 25.97% 的 1-decimal 截断值");Phase 74: 55.5(实测 55.56)。
- **先例 = 1-decimal 向下截断。** 按先例本次写 78.0,但本地余量只剩 0.01pp,而 CI 分母/分支漂移先例明确存在且这 199 个 commit CI 从未见过。
- **建议:写 77.5**(0.5pp 缓冲,仍是 55.5 → 77.5 的 +22pp 单向 ratchet;未来再 UP 到 78+ 不违反 UP-only)。这是对先例的有意偏离,须在 baseline 行 Notes 里写明理由("CI 未见的 199-commit 批次 + 首次跨 OS 大跃迁,留 0.5pp 抗漂移缓冲")。

### 原子 ratchet commit(D-07 六文件,STATE.md:226 先例)

`.coverage-threshold` + `.planning/coverage-baseline.md` + `.github/scripts/check-coverage.sh` + `.planning/STATE.md` + workstream ROADMAP(Progress 表同步,见下)+ plan SUMMARY。**ci.yml 零编辑**(先例 74-11:per-ratchet 无 workflow 改动)。

### 文档债(audit 必须顺带清偿,否则 milestone 记录自相矛盾)

1. **ROADMAP Progress 表(L310-316)整体过期**:76 显示 "IN PROGRESS 0/5"、77 "Not started"、78 "In progress 2/7"、79/80 "Not started" —— 实际 6 个 phase 34/34 plans 全 SHIPPED(各 phase 目录 PLAN/SUMMARY 1:1 实数:75=6/6, 76=5/5, 77=5/5, 78=7/7, 79=6/6, 80=5/5)。
2. **Phase 76(L117)/ Phase 77(L150)标题缺 ✅ SHIPPED 标记**(75 L82 / 78 L183 / 79 L220 / 80 L251 都有)。
3. **coverage-baseline.md 缺 Phase 75-78 行**(现只有 起点/71/72/73/74/79 后;79-06 notes 明言 "55.5 不动,Phase 81 收口 bump" —— 79/80 的 ratchet 债务留给 81 是设计内,但 75-78 连测量行都没有,audit 需回填或显式声明不回填的理由)。
4. **baseline L623 不实陈述**:"`-race` … 由 ci.yml Linux race job 兜底" —— ci.yml 全文 240 行只有 backend / coverage-diff / frontend / frontend-coverage-diff 四个 job,**没有 race job**(D-01 明令禁 race 进 coverage 跑)。audit 应更正。
5. **REQUIREMENTS.md BLOCK-01 复选框未翻转**(L30 `- [ ]`),但 Traceability 表 L60 已写 "Complete (operations 83.7%)"、commit a4cdb61 标题即 "BLOCK-01 收口 73.2 → 83.7" —— 勾选框属遗漏,audit 翻转并引用 a4cdb61。
6. 尾部 footer "Last updated: 2026-08-24"(L375)需更新。

## BLOCK-05 裁决框架(只框架,不裁决)

### 实测事实

- `internal/services/addomain` = **58.00%(1403/2419)**,与 78-07 收口值完全一致,Phase 79/80 未触碰 [VERIFIED: profile awk + 单包 `ok ... coverage: 58.0%`]
- 到 70% 需 `ceil(0.7×2419)` = 1694 covered,即 **+291 stmts**
- BER 锁死部分(78-07-SUMMARY L76-78):Search* 族 ~80 + write 族 ~100 + paging helpers ~50 = **~230 stmts**(ldap_client.go,Conclusion B:raw BER 与 go-ldap/v3 解析器不 byte-compatible;probe 文件 `ldap_fake_server_78_07_test.go` 按 D-78-07a 保留作 spin-off 起点)
- 即使 responder 解锁,仍差 **~61 stmts** 需从其余 786 uncovered 中补(error 包装、config/account_pool 分支等)
- **gate 影响 = 0**:addomain 不在 P1(L164)也不在 P2(L228)名单;把它压到 0% 加权平均仍 74.8% > 70%

### 三个选项(证据 + 代价)

| 选项 | 内容 | 先例/基建 | 代价 | 风险 |
|------|------|-----------|------|------|
| (a) lldp 式豁免 | audit 显式文档化:ldap_client BER 不可达为结构性限制,58% 锁定,不设 gate 行 | **80-05 lldp 68.8% 豁免完整先例**(66/96,差 1.2pp≈2 stmts,豁免理由 "executor 依赖 device 真实基建",80-05-SUMMARY L47/L112/L162) | 0(纯文档) | 无 gate 风险;语义上是「关闭」 |
| (b) Phase 82 spin-off | CI-only smoke 层接真 OpenLDAP(`//go:build smoke`,本地自动 skip)跑通 Search/write 族;或修 BER probe | v1.27 ROADMAP "Future Milestones" 已预埋此条目;probe 文件保留(D-78-07a);**注意 INFRA-03 已锁「嵌入式 vjeantet/ldapserver 停更风险不担」,spin-off 若引嵌入式 server 属 revisiting 该锁定决策,须走 discuss** | 新 phase(~1-2 plan);CI 上需 OpenLDAP service container | 与 v1.28 workstream 抢 Phase 编号/带宽;BER 修复工期不可估(78-07 已失败一轮) |
| (c) known gap 记录 | audit 里作为已知缺口记录,不豁免语义,留给未来 | v1.26 SC-a shortfall 的诚实记录模式(74-MILESTONE-AUDIT "⚠️ 部分达成") | 0 | milestone 以「带已知缺口 SHIPPED」收场 |

(a) 与 (c) 机械等价,差别只在语义定性。**研究建议: (a)+(c) 合体** —— 豁免文档化(沿用 lldp 格式:差距 stmts 数 + 结构性根因 + 不可达证明)+ 在 audit 的 known-gaps 段保留 open 指针(Future Milestones 的 smoke 层条目),两者不互斥。附带:addomain 邻接 quirk **QUIRK-79-05-K**(LDAPClient.Connect Bind 失败不置 nil conn → `IsConnected()` 误判,79 deferred-items.md #10)应同段呈报。

## Evidence Inventory(audit 81-03 的证据链锚点)

### Phase 执行证据

| Phase | SHIPPED 标记 | plans | 收口数字 |
|-------|-------------|-------|---------|
| 75 QUIRK | ROADMAP L82 (2026-08-23) | 6/6 | 15 项 QUIRK 全修 |
| 76 基建 | **缺标记(L117)** | 5/5 | miniredis/httpmock/Driver 工厂/TestHelperProcess/AST 守护 |
| 77 阻塞包 | **缺标记(L150)** | 5/5 | operations 83.7%(a4cdb61)/ agent/server 90.4%(47e7be0) |
| 78 阻塞包 | ROADMAP L183 (2026-08-27) | 7/7 | core 82.5% / device 82.6% / addomain 58.0%(Partial) |
| 79 长尾 root | ROADMAP L220 (2026-08-28) | 6/6 | services root 81.6%,baseline L510 行 |
| 80 长尾 tail | ROADMAP L251 (2026-08-28) | 5/5 | +2989 stmts;14 包 13/14 + lldp 豁免 |

### 结构解锁叙事(74-08 → 76-78,audit 的核心因果链)

74-08-SUMMARY L77-78 记录的三个「结构性不可破」根因,全部被后续 phase 拆除:
- device:ScrapliWrapper 持具体 scrapligo `*network.Driver` 不可注入 → **Phase 76 INFRA-02 Driver 工厂** → 实测 82.77%
- core:`Core.Init()` 全链依赖 Redis/调度器/RPA/reaper → **Phase 78-01/78-02**(Init 链 302 stmts + Close 60 stmts)→ 实测 82.54%
- agent/server:子进程 server → **Phase 76 INFRA-04 TestHelperProcess + 77-04/05 platformStrategy** → 实测 90.59%

这就是 GATE-02 删除豁免行的正当性叙事:74 的豁免条件("structurally blocked")已不再成立。

### QUIRK 清单定位(15 项遗留关闭 + 新增登记)

| 清单 | 位置 | 数量 |
|------|------|------|
| v1.26 锁定 15 项(D-12) | 74-MILESTONE-AUDIT.md L104-111 + 74-08-SUMMARY §QUIRKS | 15 |
| 15 项关闭证据 | 75-01..06-SUMMARY(逐项 commit:如 Q-1 `030fdf3` / Q-2 `cf9c88b`) | 15 |
| 新增登记 | 79 deferred-items.md(QUIRK-79-01-A,-02-D/J/K,-05-A/D/E/F/H/K)、80-03-SUMMARY(QUIRK-80-03-A..K,**3 项升级 Threat Flags**:getAuthConfig dest-pollution / Scan-to-map 双列 bug 等)、80-05-SUMMARY(QUIRK-80-05-A..D)、77-02-SUMMARY(QUIRK-77-2 等) | 10+11+4+n |
| 已修 quick fixes | **quirk-p1 `4282983`**(MemoryCache.Close 二次调用幂等,sync.Once,pkg/cache/memory.go)、**quirk-p2 `05afbc8`**(DeviceConnectionPool.Close 确保 startCleanup goroutine 退出)→ 对应 QUIRK-78-02-P1/P2 | 2 |

audit 的「15 项 QUIRK 关闭清单」应呈现为三段:v1.26 15 项 → 75 全修映射表;新增 N 项 → 登记位置 + 处置(锁定/escape hatch 立项);2 项 quick fix commit 证据。

### audit 模板(结构镜像源)

- 主模板:`.planning/milestones/v1.26-phases/74-p2-finalize-and-diff-coverage/74-MILESTONE-AUDIT.md`(119 行,6 段:SC 验证 ×5 → Requirements 追溯 → Phase 链 SUMMARY 索引 → 最终 gate 配置 → QUIRKS → 结论)
- 父级归档位:`.planning/milestones/v1.26-MILESTONE-AUDIT.md`(v1.26 也有根 milestones 目录下的一份)
- 流程模板:`74-11-PLAN.md` + `74-11-SUMMARY.md`(原子收口 commit 全记录)

**落位建议(DQ4)**:两处先例并存(74 的放 phase 目录内,v1.26 的放 milestones 根)。建议 `.planning/milestones/v1.27-MILESTONE-AUDIT.md` 为主(v1.26 根目录先例,可发现性最好),phase 目录内不放副本、只放普通 81-03-SUMMARY 引用它。

## Don't Hand-Roll

| 问题 | 不要自建 | 用现有 | 原因 |
|------|---------|--------|------|
| 全量重测 + gate 本地验证 | 手写测试脚本/自算加权平均 | `scripts/check-ci-local.sh`(与 ci.yml 同命令同口径)+ `check-coverage.sh` | awk 公式与 baseline 快照 byte-identical 的约束(D-01);自算必然口径漂移 |
| CI 状态盯守 | `gh run watch` | 轮询 `gh run view <id>`(memory:push-watch-ci —— watch 提前退出与 --commit 短 sha 均不可靠) | 已知坑 |
| per-package 复盘表 | 手写 awk 变体 | gate 输出 + `go tool cover -func` | 与 baseline 历史行同格式(`%-50s %8d %8d %6.2f%%`) |

## Common Pitfalls(本 phase 专属)

### Pitfall 1: 用带 FAIL 的 profile 正式 ratchet
**What goes wrong:** 本次研究跑 EXIT=1(scheduler flake)但 profile 数字有效,gate 仍 exit 0 —— 若直接取 78.01 入 threshold,SC 的「全量零失败」前提是假的。
**How to avoid:** 81-01 Task 0 先修 flake,再出正式测量;SUMMARY 记录两次跑的差异。
**Warning signs:** `go test` 输出里任何 `^FAIL` 行。

### Pitfall 2: threshold 刀口(0.01pp)
见 §Ratchet Mechanics。**Warning signs:** 实测值 1-decimal 截断后与自身差 <0.1pp。

### Pitfall 3: 删豁免行只看本地绿
agent/server 有明确的 env-branch 分歧史(local 22.08 vs CI 19.48,Phase 74 记录),Windows 本地绿 ≠ Linux CI 绿。**How to avoid:** SC-2 的「CI 实跑绿」是验收项不是仪式;push 后盯守 backend job 的 P2 段输出(PASS 行 ×10)。余量 12.5-20.6pp 使此风险很低,但验证步骤不能省。

### Pitfall 4: 六文件原子 commit 漏 ROADMAP/STATE
D-07 先例是六文件;本 phase 额外欠债:ROADMAP Progress 表 + 76/77 SHIPPED 标记 + REQUIREMENTS BLOCK-01 勾选。漏掉任何一项,audit 就要与仓库现状互相矛盾。

### Pitfall 5: 测量在脏树上做
工作树现有 `internal/services/system/asset_columns_schema.json` 未提交改动,**该文件是 go:embed 的**(column_config_service.go L14)—— 本地测量与 CI 干净 checkout 的测试行为可能分歧(数字不受影响,但绿色/失败可能翻转)。**How to avoid:** 正式测量前 `git stash` 或先落该 commit;同时 `.planning/` 与 v1.28 会话的并发改动需协调(见 R7)。

## Risk Register

| # | 风险 | 证据 | 缓解 |
|-----|------|------|------|
| R1 | scheduler 满载 flake 阻断零失败重测与收口 CI | `workorder_tasks.go:196` no such table `sys_duty_schedule`;单包 9.3s PASS / 满载 74.7s FAIL | 81-01 Task 0:fixture AutoMigrate 注册补齐(sqlite 缺表 family 第 6 例,memory xingran-sqlite-missing-table-pattern 的 4 项 checklist);修复后 `-count=3` 单包压测 |
| R2 | threshold 刀口被 CI 漂移击穿 | 78.01 → 78.0 余量 0.01pp;CI 插桩漂移先例 754→767 stmts | 取 77.5(0.5pp 缓冲);或严格先例 78.0 + 接受重 ratchet 可能 |
| R3 | 199 commit 未推送,CI 从未见过 77-80 后端工作 | `git rev-list --left-right --count origin/main...HEAD` = `0 199`;最后一次 main CI success 是 2026-08-27(33051506568) | push 是 81-02 的显式任务;首跑失败预案:按失败 job 分诊(backend gate / frontend gate),不阻塞 81-03 文档工作 |
| R4 | 删豁免行后某包 CI 实测 <70 → exit 5 | agent-server env-branch 前科 | 本地余量 12.5pp+;CI 盯守为验收;若发生,回滚该行 = 违反「豁免只减不增」→ 正确处置是补测试而非回滚 |
| R5 | 分母漂移让 SC-a 数学看起来不一致 | 43652/43893/43729 三套并存 | audit 用公式化口径 + 三行对照表(§分母漂移) |
| R6 | 并发 v1.28 会话共享工作树 | git log 里 `test(88)` 与 `test(80-05)` 交错;`asset_columns_schema.json` 脏(go:embed) | 正式测量前协调会话暂停 / 干净树测量;closeout commit 前清点 `git status` |
| R7 | 收口 push 触发 frontend gate(v1.28 批次在内) | ci.yml 有 frontend + frontend-coverage-diff job;199 commit 含大量 `test(88)` | run-level 红不等于 backend 四层红;SUMMARY 按 job 分色记录,backend 四层绿即 SC-3 达成,frontend 归 v1.28 |

## Decision Queue(需 planner/discuss 裁决)

| # | 决策 | 选项 | 研究建议 |
|---|------|------|---------|
| DQ1 | threshold 新值 | 78.0(严格 1-decimal 截断先例)/ 77.5(0.5pp 缓冲)/ 78(整数) | **77.5**,理由见 §Ratchet Mechanics;与先例的偏离写进 baseline Notes |
| DQ2 | 三条豁免行一次删还是分批 | 一次删(SC-2 原文)/ 分三个 commit | **一次删**。三包全部 12.5pp+ 达标,分批无验证价值,反而多两次 CI 等待 |
| DQ3 | BLOCK-05 裁决 | (a) 豁免 / (b) Phase 82 spin-off / (c) known gap | **(a)+(c) 合体**:lldp 格式豁免 + known-gaps 段留 open 指针;(b) 仅作为 Future Milestones 已有条目的确认,不在 v1.27 立项(若立,需重开 INFRA-03 锁定决策) |
| DQ4 | audit 文档落位 | `.planning/milestones/v1.27-MILESTONE-AUDIT.md` / phase 目录内 | **milestones 根**(v1.26-MILESTONE-AUDIT.md 先例,可发现性最好);phase 内 81-03-SUMMARY 只引用 |
| DQ5 | 全量跑 flake 处置政策 | 修 fixture 后重测;重测仍 flake 则? | 修一次 + `-count=3` 压测;若仍 flake 升级为确定性修复(AutoMigrate 缺表是确定性根因,不是时序赌运气);**不允许** t.Skip 掩盖(SC 需要零失败证据) |
| DQ6 | push 策略与 diff 层验证 | 单次 push main(里程碑惯例)/ PR 触发 diff gate | **单次 push main**(v1.27 全程惯例);diff 层用 `check-ci-local.sh --diff` 本地补验(push 到 main 不触发 PR-only coverage-diff job,ci.yml L99) |

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | go test(go1.24.5,`-covermode=atomic`,D-01 禁 `-race`) |
| Config file | 无独立配置;ci.yml L64-66 是口径唯一真相源 |
| Quick run | `go test -count=1 -timeout 5m ./internal/scheduler/`(~10s,flake 修复验证) |
| Full suite | `bash scripts/check-ci-local.sh`(全量 test + gate,~11-15 min);`--diff` 追加差量层 |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| GATE-01 | 加权平均 ≥ 新 threshold | gate(集成) | `bash .github/scripts/check-coverage.sh coverage.out .coverage-threshold` → exit 0 且 PACKAGE 行 ≥ 阈值 | ✅ |
| GATE-02 | P2 uniform 70 floor 生效 | gate(集成) | 同上,P2 段 10 行全 PASS 且输出尾行变 "70.0% x 10 packages" | ✅(删行后需确认尾行文案同步) |
| GATE-03 | 4 层 CI 全绿 + 无倒退 | CI + 文档 | push 后 `gh run view <id>` backend job 全 step 绿;audit 引 run id | ✅(依赖 push) |

### Sampling Rate

- Per task commit: 修改脚本的 commit → 本地 `check-coverage.sh` 对既有 profile 快速验证(秒级,不需重跑测试)
- Per wave merge: 全量 `check-ci-local.sh`
- Phase gate: 收口 push 的 CI run 全绿 + audit 落盘

### Wave 0 Gaps

- [x] 测试基建 —— 无缺口(34 plans 已建成;零 Docker 双环境同构)
- [ ] `internal/scheduler` flake 修复(81-01 Task 0,唯一前置缺口)

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| go | 全量重测 | ✓ | 1.24.5(= go.mod toolchain) | — |
| bash + awk | gate 脚本 | ✓ | Git Bash | — |
| gh CLI | CI 盯守 | ✓ | 实测可用 | 网页盯守 |
| `-race` | 无(本 phase 不需要) | ✗ 本地不可执行(Windows cgo,79 notes L624 记录) | — | 不需要;baseline "race job 兜底" 说法不实,audit 更正 |
| Docker | 无(零 Docker 里程碑) | — | — | — |

**Missing dependencies with no fallback:** 无。

## Security Domain

`security_enforcement` 未显式关闭,按启用处理。本 phase 不改生产代码,ASVS 各类目均无新增攻击面:

| ASVS Category | Applies | 说明 |
|---------------|---------|------|
| V2/V3/V4/V6 | no | 零认证/会话/授权/密码学改动;gate 脚本无密钥处理 |
| V5 Input Validation | no(仅测试面) | 唯一生产树触碰是 quirk-p1/p2 已完成(commit 4282983/05afbc8),不在 81 范围 |

审计应呈报的既有安全相关 known-gaps(来自 QUIRK 登记,不修,只定案):
- QUIRK-79-05-K:LDAP `IsConnected()` 在 Bind 失败后误报 true(现网可见,79 deferred-items #10)
- QUIRK-80-03 的 3 项 Threat Flags 升级项(80-03-SUMMARY L174/L186)
- CI 面:ci.yml `permissions: contents: read`(L25)维持最小权限;gate 脚本零外部输入,无注入面(awk 只吃本地 profile)

## Sources

### Primary(HIGH,全部 file:line / 命令输出实测)

- `.github/scripts/check-coverage.sh` L16-20(exit codes)/ L39-45 + L218-245(P2 ratchet)/ L242-244(三行)/ L163-164 + L227-228(P1/P2 名单)
- `.coverage-threshold`(4 字节 `55.5`,xxd 实测)
- `.github/workflows/ci.yml` L28-90(backend job)/ L93-99(coverage-diff PR-only)/ L122+ L202+(frontend 两 job)/ 全文无 race job
- 全量重测 + gate 实跑(本会话:70 ok / 1 FAIL / PACKAGE 43729 34112 78.01% / 10 P2 PASS)
- `internal/scheduler` 单包复跑 PASS(9.349s)
- `cov_full_research.out` awk 提取:addomain 1403/2419=58.00%,lldp 66/96=68.75%,0% 包 ×4
- `.planning/coverage-baseline.md` L11-13 / L443 / L510 / L619-623;`.planning/milestones/v1.26-phases/74-p2-finalize-and-diff-coverage/74-MILESTONE-AUDIT.md`(结构)/ `74-08-SUMMARY.md` L77-78
- workstream ROADMAP L82/L117/L150/L183/L220/L251/L310-316/L375;REQUIREMENTS L30/L50-52/L56-77;STATE.md:6/L21/L226
- 78-07-SUMMARY L7-8/L29-65/L76-78(Conclusion B + 不可达分解);79 deferred-items.md;80-03/80-05-SUMMARY(quirk + lldp 豁免)
- git: `4282983` / `05afbc8`(show 实测)/ `git rev-list --left-right --count origin/main...HEAD` = `0 199` / `gh run list` 最近 5 条

### Tertiary(LOW,未在本会话验证)

- 无外部来源 —— 本 phase 全部结论来自仓库内部状态,零 web/Context7 依赖。

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | CI(Linux)实测加权平均会落在 77.5 之上(基于本地 78.01 + 历史漂移 ≤1.7%) | DQ1/R2 | 若 CI < 77.5,gate fail,需按差额补测试或回调 ratchet(UP-only 允许本次不 bump,风险=收口延期) |
| A2 | scheduler flake 根因是 fixture AutoMigrate 缺注册(确定性可修),非深层时序问题 | R1/DQ5 | 若是深层并发问题,修复工时放大;处置不变(修,不 skip) |
| A3 | push 后 frontend job 状态不影响 SC-3 的「4 层 gate」判定(4 层均属 backend/coverage-diff job) | R7/DQ6 | 若 planner 把 run-level 绿作为 SC-3 口径,v1.28 的 frontend 债务会错误地 block v1.27 收口 —— 需在 plan 里显式写死口径 |
| A4 | BLOCK-01 勾选翻转只需引用 a4cdb61(83.7%)即视为证据充分 | 文档债 #5 | 低;若要求更严可附 77-03-SUMMARY 链接 |

## Metadata

**Confidence breakdown:**
- 测量数字: HIGH —— 全部来自本会话实跑(单次采样;正式 ratchet 前的绿色重测是 81-01 的验收,可能小幅修正 78.01 这个数)
- gate 机制/删除清单: HIGH —— 逐行引用
- 文档债清单: HIGH —— 逐条 file:line
- BLOCK-05 框架: HIGH(数字)/ MEDIUM(裁决建议 —— 是建议不是决定)

**Research date:** 2026-08-28
**Valid until:** 2026-09-27(仓库内部状态,仅随新 commit 失效;81 开工前建议重跑一次 gate 确认无新 commit 改变数字)
