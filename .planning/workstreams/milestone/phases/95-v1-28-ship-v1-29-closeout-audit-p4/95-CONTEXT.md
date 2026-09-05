# Phase 95: v1.28 SHIP 收口 + v1.29 closeout + audit (🟢 长期 P4) - Context

**Gathered:** 2026-09-06
**Status:** Ready for planning
**Decision mode:** 自主裁定（用户 2026-09-06 预授权「全部自主运行到里程碑结束」，gray areas 由 Claude 依先例与证据定夺，全部决策透明记录于 95-DISCUSSION-LOG.md）

<domain>
## Phase Boundary

v1.29 里程碑收尾相：核对 v1.28 阶段性收口文档落地（CLOSEOUT-01/02 大部分已于 2026-09-04 完成）、跑 v1.29 完整验证 gate、处置 Phase 94 遗留的 type-check gate 空转缺陷、生成 `v1.29-MILESTONE-AUDIT.md` 验证报告、设置 v1.29 SHIPPED 状态。

**关键现实校准（2026-09-06 侦察确认，与 REQUIREMENTS/ROADMAP 原文有重大偏差，以本 CONTEXT 为准）：**

1. **CLOSEOUT-01 实际已完成** — `.planning/MILESTONES.md:32-53` v1.28 段已存在且完整（✅ SHIPPED + 阶段性收口 2026-09-04 + 45.13% 理由 + 距 70% 缺口 24.87pp + Phase 88 备选 + 归档位置四件套）。CLOSEOUT-01 剩余动作 = **核对确认**而非添加
2. **CLOSEOUT-02 实际已完成** — `.planning/PROJECT.md:35` v1.28 段已标「✅ SHIPPED + ARCHIVED 2026-09-04 (阶段性收口 45.13%)」；`.planning/workstreams/frontend-coverage/` 已在 MILESTONES 段登记「**保留作历史**」——REQUIREMENTS 原文「归档到 .archive/ **或保留作历史**」的后者分支已满足，**目录零移动**
3. **CLOSEOUT-03 是工作主体** — 完整 gate 跑批（go build / go test / npm 四件套 / 双 coverage gate）+ 7 项行动完成确认 + v1.29-MILESTONE-AUDIT.md（模板 = `.planning/milestones/v1.27-MILESTONE-AUDIT.md`）
4. **Phase 94 遗留 2 项 HUMAN-UAT**（94-VERIFICATION 判 human_needed，用户自主授权下由本相裁决）：
   a. **前端 type-check gate 空转**（项目级前置缺陷，非 94 引入）：`npm run type-check` 是裸 `tsc --noEmit`，根 tsconfig 为 solution-style（`files: []`），实际检查 **0 文件恒 exit 0**（CI 同受影响）。94 verifier 以真实配置探针实证：全仓仅 1 处存量错误 `xingran-react-frontend/src/pages/network/backups/hooks/useRestoreTask.ts:47`（Phase 93 a4bfc71 引入，不在 94 改动清单），94 的 16 个交付文件真实配置 0 类型错误
   b. **两处外观级变化产品确认**：asset excel 导出文件名英文化（content-disposition 优先）+ 下载错误文案归一——94-03-SUMMARY deviation 6 已登记、测试已锁定新行为
5. **ROADMAP SC-1/SC-2 措辞按现实校准** — 「写入/标记」已是既成事实，SC 落地为核对确认（Phase 91/92/94 措辞校准先例，同 commit 修订 ROADMAP）
6. **记账项已由 Phase 94 收口修复** — REQUIREMENTS API-FACTORY-01..05 全部勾选、workstream ROADMAP Progress 表 Phase 94 Complete 3/3（commit 6a36a0b）

**不在本 phase：**
- V130-CANDIDATES 清单处置（WR-01..05 等 v1.30 候选，仅 audit 报告转记确认）
- milestone 完整 archive 流程（`.planning/milestones/v1.29-phases/` 迁移等 —— `/gsd-complete-milestone` 独立工作流，用户另行触发）
- Phase 88 前端覆盖率续推（v1.28 deferred，重启属 v1.30+ 决策）
- lint 1390 warnings 清零（pre-existing debt，保持「0 errors 为门」口径）
- operlog.exclude_paths todo（.planning/todos/pending，v1.29 D-05 明确 deferred）

</domain>

<decisions>
## Implementation Decisions

### CLOSEOUT-01/02 核对确认 (Area 1)

- **D-01: 核对确认而非重写** — MILESTONES.md v1.28 段与 PROJECT.md v1.28 SHIPPED+ARCHIVED 标记均已存在（2026-09-04 收口产物）。plan 95-01 动作 = 逐项核对 CLOSEOUT-01/02 内容清单（45.13% 理由 / 24.87pp 缺口 / Phase 88 备选 / 归档位置登记）+ 补漏（若有缺项才补）。REQUIREMENTS CLOSEOUT-01/02 与 ROADMAP SC-1/SC-2 措辞按现实校准（同 commit，Phase 94 D-01 先例）
- **D-02: frontend-coverage workstream 保留作历史，零移动** — MILESTONES v1.28 段已登记「保留作历史」= REQUIREMENTS「或保留作历史」分支满足；`.archive/` 迁移不做（MILESTONES 声明即决策记录）。95-01 仅核对目录完整性（STATE.md/ROADMAP.md/phases 存在性确认）

### Phase 94 HUMAN-UAT 两项裁决 (Area 2)

- **D-03: type-check gate 空转修复优先** — 修 `npm run type-check` 使 tsc 实际检查源码（solution-style 根 tsconfig `files: []` 的问题；方案 researcher 评估：根 config 增加 references/files 或改用 `tsc --noEmit -p tsconfig.app.json` 等最小侵入路径）+ 修复唯一存量错误 `useRestoreTask.ts:47`。**授权边界：若修复暴露 >5 文件新错误或需重设 tsconfig 策略（拆分/重构 config 体系），降级为 audit 报告登记 + V130 候选，零代码改动收口**。理由：v1.29 技术债治理里程碑，验证防线失真是核心技术债；verifier 实证全仓仅 1 处错误成本可控；不修则 CLOSEOUT-03 的「type-check 0 错误」是恒真断言，audit 报告价值打折
- **D-04: 外观级变化接受现状** — asset excel 文件名（后端英文名优先）与错误文案归一：94-03-SUMMARY deviation 6 已登记、测试已锁定、可逆（ignoreContentDisposition 选项为 deferred）。本相关闭该 UAT 项（裁决 = 接受），audit 报告记录；若未来产品要求中文文件名再启用 deferred 选项
- **D-05: 94-HUMAN-UAT.md 状态流转** — 两项裁决落档后 UAT frontmatter status: partial → resolved（type-check 修复线）或 status: partial 保留注记（降级线），audit 报告交叉引用

### CLOSEOUT-03 完整 gate + audit (Area 3)

- **D-06: gate 跑批清单与口径** — ① `go build ./...` 0 错误 ② `go test ./...` 0 失败（全量一次性）③ `npm run type-check`（D-03 修复后为真检查；降级线则如实标注恒真缺陷）④ `npm run lint` 0 errors（1390 warnings pre-existing 口径不变）⑤ `npm run test` 全绿 ⑥ 后端 coverage gate（`bash .github/scripts/check-coverage.sh` 同 CI :77 口径）⑦ 前端 coverage gate（`check-frontend-coverage.sh` 同 CI :183 口径，仓库根执行）。跑批结果逐项写入 audit 报告（命令 + exit code + 关键数字）
- **D-07: v1.29-MILESTONE-AUDIT.md 仿 v1.27 模板** — 落点 `.planning/milestones/v1.29-MILESTONE-AUDIT.md`；章节结构对照 `v1.27-MILESTONE-AUDIT.md`：Milestone SC 验证（v1.29 D-01 目标线 7 项行动 100% + PROJECT.md 锁定决策 D-01..D-06 践行）/ Requirements 追溯（41/41）/ Phase 链 SUMMARY 索引（89-95）/ 最终 gate 配置与实测 / 预存缺陷裁决（V130-CANDIDATES 转记确认）/ 结论。**新增章节**：type-check gate 空转缺陷的发现→处置（或降级）全程记录（v1.29 独有发现，审计价值项）
- **D-08: 7 项行动确认口径** — 按 PROJECT.md v1.29 Progress 段逐项核对：PAGINATION ✅ / TIMEOUTS ✅ / CRUD 复用 ✅ / 缓存统一 ✅ / config_backup 闭环 ✅ / API 工厂化 ✅ / v1.28 收口 ✅（95-01 核对即其收口动作）。每项引用其 phase 的 VERIFICATION.md/关键 commit 为证据
- **D-09: SC-5 SHIPPED 状态 = MILESTONES.md 文档标记** — v1.29 段标题「🚧 STARTED」→「✅ SHIPPED <date>」+ PROJECT.md v1.29 段 SHIPPED 标记；完整 milestone archive（phases 目录迁移等）不在本相（`/gsd-complete-milestone` 另行）

### 收口惯例 (Area 4)

- **D-10: 措辞校准纪律** — CLOSEOUT-01/02 与 ROADMAP SC-1/SC-2 的「写入/标记」→「核对确认（已存在）」修订，同 commit 落地（Phase 90 3a2efe5 / 94 D-01 先例）
- **D-11: 文档 commit 纪律** — audit 报告、MILESTONES、PROJECT、REQUIREMENTS、ROADMAP 各文档变更 atomic commit；gate 跑批不产生代码变更（除 D-03 修复线）

### Claude's Discretion

以下细节由 planner/researcher 决定，无需再问用户：
- type-check 修复的具体 tsconfig 方案（references vs -p 指向 vs files 补全；researcher 以 tsc 探针实证选型）
- D-03 降级线的「>5 文件」判定时机（修复分支试跑后即知）
- audit 报告各章节的详略粒度与证据引用格式（对照 v1.27 模板自然裁量）
- gate 跑批的执行顺序与分批 commit 粒度
- 95-01/95-02 的任务切分微调（保持「95-01 文档核对 → 95-02 gate+audit」骨架）
- 7 项行动证据引用的抽查深度（VERIFICATION.md 存在性 + 关键数字引用即可，不重跑各相验证）
- HUMAN-UAT 状态流转的具体注记格式

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### v1.29 milestone 全局
- `.planning/REQUIREMENTS.md` § CLOSEOUT（:98-102）— CLOSEOUT-01..03 需求原文（**01/02 按 D-01 校准为核对确认**）
- `.planning/workstreams/milestone/ROADMAP.md` § Phase 95 — Goal / 2 plans 拆分 / 5 Success Criteria（**SC-1/SC-2 按 D-01 校准；SC-3 按 D-03/D-06 口径**）
- `.planning/PROJECT.md` — v1.29 Current Milestone 段（Progress 七项 ✅ 记录 = D-08 确认的证据基础）+ v1.28 SHIPPED+ARCHIVED 段（:35）
- `.planning/MILESTONES.md` — v1.28 段（:32-53，CLOSEOUT-01 核对对象）+ v1.29 段（:3，D-09 SHIPPED 标记落点）

### Phase 94 遗留（本相裁决输入）
- `.planning/workstreams/milestone/phases/94-api-p2/94-VERIFICATION.md` — human_needed 判定 + type-check 空转发现 + 真实配置探针数据（全仓 1 处错误 useRestoreTask.ts:47）
- `.planning/workstreams/milestone/phases/94-api-p2/94-HUMAN-UAT.md` — 2 项人决事项（D-03/D-04/D-05 裁决对象）
- `xingran-react-frontend/src/pages/network/backups/hooks/useRestoreTask.ts` — :47 存量类型错误（D-03 修复对象，Phase 93 a4bfc71 引入）

### audit 模板与 gate 基建
- `.planning/milestones/v1.27-MILESTONE-AUDIT.md` — D-07 报告模板（SC 验证 / REQ 追溯 / SUMMARY 索引 / gate 配置 / 裁决 / 结论 六段结构）
- `.github/workflows/ci.yml` — 后端 gate（:77 check-coverage.sh）+ 前端 gate（:183 check-frontend-coverage.sh，仓库根执行）口径
- `.github/scripts/check-coverage.sh` / `.github/scripts/check-frontend-coverage.sh` + `.coverage-threshold` / `.coverage-fe-floors` — 双 gate 脚本与阈值文件
- `xingran-react-frontend/tsconfig.json` + `tsconfig.app.json` — D-03 type-check 修复的配置现场（solution-style `files: []` 缺陷所在）

### 项目级约束
- `CLAUDE.md` § Frontend API Factory Convention — 94 收口新增，audit 报告引用
- `CLAUDE.md` § Compilation & Build Verification / Testing — go build/test 纪律
- `.planning/REQUIREMENTS.md` § V130-CANDIDATES（:122+）— D-07 转记确认对象

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `v1.27-MILESTONE-AUDIT.md` — 六段式报告结构直接套用（v1.29 数据替换）
- PROJECT.md v1.29 Progress 段 — 七项行动的完成记录已就绪（D-08 证据索引）
- 双 coverage gate 脚本 + 阈值文件 — CI 同款命令本地可跑（94-03 T3 已验证前端 gate 命令链含 cd 切换基准）
- 94-VERIFICATION.md 的 tsc 探针结论 — D-03 修复面的先验数据（全仓 1 处错误），researcher 无需重新大范围探查

### Established Patterns
- 措辞校准同 commit 纪律（Phase 90 3a2efe5 / 91 / 92 D-06 / 94 D-01 四期先例）
- atomic commit 纪律（v1.29 D-04）
- audit 报告落 `.planning/milestones/`（v1.19/26/27 三期先例）
- MILESTONES.md 段式（SHIPPED 日期 + Delivered + 理由 + 遗留 + 归档位置）

### Integration Points
- `.planning/workstreams/milestone/STATE.md` — phase 推进状态（95 完成后 is_last_phase）
- `94-HUMAN-UAT.md` — D-05 状态流转落点
- V130-CANDIDATES — audit 转记的下游承接（v1.30 输入，本相只确认不处置）

</code_context>

<specifics>
## Specific Ideas

### v1.29 SC 对照表（D-07 audit 报告主轴预览）
| v1.29 行动 | Phase | 收口证据 | audit 确认口径 |
|-----------|-------|---------|---------------|
| PAGINATION 常量集中化 | 89 | 89-CONTEXT D-01..19 + SHIPPED 2026-09-04 | VERIFICATION + 关键 commit |
| TIMEOUTS/PORT/PROTOCOL/CONCURRENCY | 90 | 10 常量 + AST 锁值 | 同上 |
| CRUD 复用 base.GORMRepository[T] | 91 | 11/11 services + LOC OVR-91-01 | 同上 |
| 缓存层三处统一 | 92 | base 权威 + 32 处收敛 + 207 口径 | 同上 |
| config_backup TODO 闭环 | 93 | gzip/RestoreConfig/异步任务 + 93_NN 19 用例 | 同上 |
| 前端 API 工厂化 | 94 | 工厂单一权威 + 覆盖率 59.79% + review 0 Critical | 同上 |
| v1.28 阶段性收口 | 95-01 | 本相核对确认 | CLOSEOUT-01/02 核对清单 |

### type-check 空转的缺陷链（D-03 修复设计输入）
```
根 tsconfig.json: { "files": [] }  ← solution-style（仅 references，无实体检查面）
npm run type-check = tsc --noEmit   ← 读取根 config → 检查 0 文件 → 恒 exit 0
实际检查面在 tsconfig.app.json（src 树）+ tsconfig.node.json（vite config）
最小修复方向：type-check 脚本改 tsc --noEmit -p tsconfig.app.json（或根 config 补 references）
已知连锁：真实检查面下 useRestoreTask.ts:47 一处错误需同修（94 verifier 探针实证仅此一处）
```

### 估算依据（供 planner 参考）
- 95-01：纯文档核对（MILESTONES/PROJECT/REQUIREMENTS/ROADMAP 四文件核对 + 措辞校准 commit），工作量小
- 95-02：gate 跑批（go build ~1min + go test ~5-10min + npm 四件套 ~15min 含 coverage）+ D-03 修复（tsconfig + 1 文件）+ audit 报告撰写
- audit 报告主体是数据汇编（各相 VERIFICATION/commit 索引），非新验证逻辑

</specifics>

<deferred>
## Deferred Ideas

- **milestone 完整 archive**（.planning/milestones/v1.29-phases/ 迁移、workstream 归位）— `/gsd-complete-milestone` 独立工作流，SC-5 仅做文档 SHIPPED 标记（D-09）
- **V130-CANDIDATES 处置**（WR-01..05 预存缺陷、excel 文件名中文选项、Phase 88 覆盖率续推）— v1.30 输入清单，本相 audit 转记确认即可
- **operlog.exclude_paths todo** — v1.29 D-05 明确 deferred，维持 pending
- **frontend-coverage 目录 .archive/ 迁移** — MILESTONES 已登记「保留作历史」，不再迁移（D-02）

None — discussion stayed within phase scope（自主裁定均在 CLOSEOUT 边界内）

</deferred>

---

*Phase: 95-v1.28 SHIP 收口 + v1.29 closeout + audit*
*Context gathered: 2026-09-06*
