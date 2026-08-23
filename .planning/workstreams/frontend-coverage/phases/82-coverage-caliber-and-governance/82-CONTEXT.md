# Phase 82: 口径修正与治理基建 - Context

**Gathered:** 2026-08-23
**Status:** Ready for planning

<domain>
## Phase Boundary

本 phase 不新增任何业务测试,只完成三件事:

1. **切换全量口径**:Vitest 4 移除 `coverage.all` 后,必须显式用 `coverage.include` 圈定 `src/**/*.{ts,tsx}`,让 584 个文件全部进入报告,未测试文件以 0% 计入。
2. **白名单正式登记**:cad-editor(804 stmts) + cad-elements(224 stmts) 合计 1028 stmts(≈4.5% ≤ 5%)以「排除理由 / 面积 / 复审条件」三项登记,并同步到 `coverage.exclude`。
3. **落地 4 层 CI 防倒退 gate**:全局阈值 gate + per-directory floor gate + PR diff coverage ≥80% gate + baseline ratchet 文档,整体对称后端 v1.26 的治理模式。

**Out of scope**: 测试补齐、公共 harness 沉淀、组件/页面具体覆盖提升——这些属于 Phase 83-87。

</domain>

<decisions>
## Implementation Decisions

### CI 布局与时长预算
- **D-01:** 全局阈值 gate 与 per-dir floor gate 内嵌在 `ci.yml` 的 `frontend` job 中:将原有 `npm run test` 步骤替换为 `npm run test:coverage`,其后接 gate 步骤。对称后端 `backend` job 布局,避免独立 job 带来的 npm ci + 全量测试重跑。
- **D-02:** PR diff coverage ≥80% gate 通过 artifact 复用实现:新增 `frontend-coverage-diff` job 下载 `frontend` job 上传的 vitest json 产物,基于 `git diff` 计算,零重复测试。
- **D-03:** CI 中保留 `text` / `json` / `html` 三种 reporter,html 目录 zip 后作为 artifact 上传,供本地 debug 下载。
- **D-04:** `frontend` job 的 timeout 保持 15 分钟,先观察全量口径 transform 的实际增量,若不足再在执行阶段末尾调整。

### Per-directory floor 粒度与 ratchet
- **D-05:** 目录粒度采用 `src/` 下一级目录一个 floor,同时 `pages/` 按二级子目录拆分(operations / system / network / duty / …)。这样与 ROADMAP Phase 83-87 的 wave 验收(例如“pages/operations ≥70%”)直接对齐。
- **D-06:** 未达标目录的 ratchet 初值采用保守下界:取当期实测值减去 0.5 个百分点。对称后端 P2_RATCHET 教训(测量噪声会把照抄的实测值变成 gate 失败)。
- **D-07:** per-dir floor 阈值表存储在独立数据文件 `.coverage-fe-floors` 中,文件内同时包含全局阈值行与 per-dir floor 表;gate 脚本读取该文件作为输入。ratchet bump 是纯数据变更,不修改脚本。
- **D-08:** 初始 ratchet 表由 gate 脚本提供 `--init` 模式,从第一次全量口径 coverage json 自动生成(已套用 D-06 的 −0.5pp 余量),人工只审白名单部分。

### 白名单登记与同源
- **D-09:** 前端覆盖率基线文档新建为 `.planning/frontend-coverage-baseline.md`,与后端 `.planning/coverage-baseline.md` 对称并列,包含 ratchet 记录表与白名单登记段。
- **D-10:** 白名单排除的单一真相源是 `xingran-react-frontend/vitest.config.ts` 中的 `coverage.exclude` 数组;gate 脚本做漂移检测,若 cad-editor / cad-elements 仍出现在 coverage json 中则视为配置漂移并失败。
- **D-11:** 白名单复审条件为定量触发:当白名单目录自身语句覆盖率达到 ≥70%(或执行阶段商定的中间 bar)即可启动移除;每个 milestone 收口时强制重审一次。
- **D-12:** 白名单锁死:仅允许 cad-editor 与 cad-elements 两项,未来任何新增必须走 milestone 级显式决策,并重新核算总面积 ≤5%。

### Gate 指标维度
- **D-13:** 全局阈值 gate 与 per-dir floor gate 只使用 **statements** 维度,与后端 `check-coverage.sh` 加权平均口径对称,也符合 ROADMAP 全篇使用 statements 的表述。
- **D-14:** 全局阈值设定为 **3.8%**(一位小数),白名单后实测约 3.85%,留 0.05pp 最小余量。
- **D-15:** PR diff coverage ≥80% gate 按 **lines** 维度计算:git diff 中的新增/改动行与 vitest json 的 line coverage 对齐(ROADMAP 字面为“新增/改动行覆盖”)。
- **D-16:** vitest 原生 `coverage.thresholds` 删除或禁用,全部 gate 逻辑由外部 bash 脚本承担,作为唯一真相源,避免 vitest thresholds 与 `.coverage-fe-floors` 双轨维护导致漂移。

### Claude's Discretion
None — all discussed areas have explicit user decisions.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Roadmap and requirements
- `.planning/workstreams/frontend-coverage/ROADMAP.md` — Phase 82 目标、成功标准、锁定决策 D-01~D-04(v1.28 起编 / 75-81 严禁使用 / statements ≥70% / 白名单 ≤5%)
- `.planning/workstreams/frontend-coverage/REQUIREMENTS.md` — GOV-01~GOV-05、QUAL-02 需求定义
- `.planning/PROJECT.md` — v1.28 milestone「前端测试覆盖率优秀」上下文

### Backend reference implementation (symmetric gates)
- `.github/workflows/ci.yml` — CI job 结构:`backend` job 内嵌 coverage gate + `coverage-diff` job 下载 artifact 复用 `coverage.out`
- `.github/scripts/check-coverage.sh` — 加权平均 gate + P1/P2 per-package floor + ratchet 机制 + exit code 语义(1/4/5)
- `.github/scripts/check-diff-coverage.sh` — PR diff coverage ≥80% 的 bash+awk 自实现
- `.planning/coverage-baseline.md` — 后端 ratchet 记录表 + per-package 快照模式(前端基线文档对称参照)
- `.coverage-threshold` — 后端全局阈值单值文件(当前 55.5)

### Frontend current state
- `xingran-react-frontend/vitest.config.ts` — 当前 coverage 配置:无 `coverage.include`, thresholds 为旧口径 24/15/18/24, exclude 已含 test/config/d.ts/mockData/dist
- `xingran-react-frontend/package.json` — test scripts: `test` / `test:ui` / `test:coverage`

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- 后端 `.github/scripts/check-coverage.sh` 的 inline awk 聚合公式与 exit-code 设计(1/4/5)可作为前端版 `check-frontend-coverage.sh` 的结构模板。
- 后端 `.github/scripts/check-diff-coverage.sh` 的三点 diff + profile 解析模式,可改写为 vitest json + git diff 版本。
- 现有 `xingran-react-frontend/vitest.config.ts` 已配置 v8 provider、json/text/html reporter、15s timeout、jsdom 环境、`./src/test/setup.ts` setupFiles — 只需添加 `coverage.include` 并清理 `thresholds`。
- 现有 19 个前端测试文件作为全量口径切换后的“已知通过样本”,用于验证 coverage json 仍正常产出。

### Established Patterns
- **bash+awk 零依赖自实现**是后端治理的锁定范式(D-01),前端版 gate 脚本应保持一致,不引入第三方 coverage action。
- **ratchet 纪律**:阈值提升与基线文档追加在同一 PR 完成(后端 D-04)。
- **白名单面积 ≤5% 总语句**:QUAL-02 已锁定 cad-editor(804 stmts) + cad-elements(224 stmts) = 1028 stmts ≈ 4.5%。
- **CI artifact 复用**:diff gate 不重复跑测试,从 backend `coverage-diff` job 模式迁移。

### Integration Points
- `ci.yml` 的 `frontend` job:Test 步骤由 `npm run test` 改为 `npm run test:coverage`,后接 gate 步骤;`Build` 步骤仍保留在 gate 之后。
- `frontend` job 需上传 coverage json(以及 zip 后的 html 目录)作为 artifact,供 `frontend-coverage-diff` job 下载。
- `.coverage-fe-floors` 作为 gate 脚本的输入;`.planning/frontend-coverage-baseline.md` 作为人类可读的 ratchet 记录。

</code_context>

<specifics>
## Specific Ideas

- Phase 82 **不新增任何测试**;目标是口径切换与 4 层 gate 就绪。
- 全局阈值 **3.8%**(白名单后实测约 3.85%,留 0.05pp 余量)。
- per-dir floor 目标 **70%**;基线期所有目录均未达标,ratchet 初值取实测 −0.5pp。
- diff coverage gate 仅 PR 触发,目标 **80%**,按新增/改动 lines 计算。
- 白名单目录:
  - `xingran-react-frontend/src/components/cad-editor` — 804 stmts
  - `xingran-react-frontend/src/components/cad-elements` — 224 stmts
- 文件/产物命名约定:`.coverage-fe-floors`、`.planning/frontend-coverage-baseline.md`、`check-frontend-coverage.sh`、`check-frontend-diff-coverage.sh`(或等价命名)。

</specifics>

<deferred>
## Deferred Ideas

- 前端具体测试补齐 —— Phase 83(P0 基建层)、Phase 84(P1 组件层)、Phase 85-87(P2 页面层三波)。
- 公共测试 harness 沉淀 —— Phase 83 成功标准之一。
- 若全量 coverage transform 超出 15 分钟,在执行阶段实测后由 Phase 82 末尾决定是否上调 CI timeout。
- CI 缓存/分片优化:当前不引入,先通过 D-04 观察,必要时未来 phase 处理。

</deferred>

---

*Phase: 82-口径修正与治理基建*
*Context gathered: 2026-08-23 via /gsd:discuss-phase*
