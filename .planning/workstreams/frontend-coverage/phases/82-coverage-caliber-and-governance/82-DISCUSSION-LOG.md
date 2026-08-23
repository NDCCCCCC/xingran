# Phase 82: 口径修正与治理基建 - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-23
**Phase:** 82-口径修正与治理基建
**Areas discussed:** CI 布局与时长预算, Per-dir floor 粒度与 ratchet, 白名单登记与同源, Gate 指标维度

---

## CI 布局与时长预算

| Option | Description | Selected |
|--------|-------------|----------|
| 内嵌 frontend job | Test 步骤换成 npm run test:coverage,后接 gate 步骤(对称后端 backend job 布局) | ✓ |
| 独立 coverage job | 新增 job 独立跑 coverage + gate,代价 npm ci + 全量 test 重跑一遍 | |

**User's choice:** 内嵌 frontend job (Recommended)

---

| Option | Description | Selected |
|--------|-------------|----------|
| 复用 artifact | 前端版 coverage-diff job 下载 frontend job 上传的 vitest json,零重复测试 | ✓ |
| Diff job 重跑 | diff job 自己跑一遍 test:coverage 再算 | |

**User's choice:** 复用 artifact (Recommended)

---

| Option | Description | Selected |
|--------|-------------|----------|
| 保留 html artifact | 保留 text+json+html 三 reporter,html 目录 zip 后上传 artifact 供 debug | ✓ |
| 砍掉 CI html | CI 只留 text+json,html 仅本地手动跑 | |

**User's choice:** 保留 html artifact (Recommended)

---

| Option | Description | Selected |
|--------|-------------|----------|
| 保持 15min 观察 | 保持 15 分钟,实测全量 coverage transform 增量后再调 | ✓ |
| 预升 20-25min | 预防性提到 20-25 分钟,避免 Phase 82 合入当天 CI 超时红 | |

**User's choice:** 保持 15min 观察 (Recommended)

**Notes:** 用户四项均选了推荐项。timeout 策略为观察优先,若执行阶段实测超时再在 Phase 82 末尾调整。

---

## Per-dir floor 粒度与 ratchet

| Option | Description | Selected |
|--------|-------------|----------|
| 一级+pages 二级 | lib/utils/hooks/store/components 等一级目录一个 floor,pages/ 按二级子目录各自一个,与 83-87 波次验收对齐 | ✓ |
| 仅一级目录 | pages 整体一个 floor,无法独立验收 85/86/87 | |
| P0/P1/P2 波次组 | 最粗粒度,丢失 per-directory 语义 | |

**User's choice:** 一级+pages 二级 (Recommended)

---

| Option | Description | Selected |
|--------|-------------|----------|
| 保守下界 −0.5pp | 每个未达标目录 ratchet floor = 当期实测 − 0.5pp 余量,防测量噪声 | ✓ |
| 实测值照抄 | 最紧防倒退,但噪声敏感 | |

**User's choice:** 保守下界 −0.5pp (Recommended)

---

| Option | Description | Selected |
|--------|-------------|----------|
| 独立数据文件 | 新建 `.coverage-fe-floors` 含全局阈值行 + per-dir floor 表,脚本读取,ratchet bump 纯数据变更 | ✓ |
| 脚本内嵌表 | 对称后端 P2_RATCHET 变量模式,但前端目录多会让脚本膨胀 | |

**User's choice:** 独立数据文件 (Recommended)

---

| Option | Description | Selected |
|--------|-------------|----------|
| 脚本自动生成 | gate 脚本带 `--init` 模式从第一次全量 json 实测自动生成 floors 文件 | ✓ |
| 手工登记 | 从 baseline.md 快照人工誊写 | |

**User's choice:** 脚本自动生成 (Recommended)

**Notes:** 用户四项均选了推荐项。目录粒度与后续 Phase 83-87 的验收边界直接对齐。

---

## 白名单登记与同源

| Option | Description | Selected |
|--------|-------------|----------|
| 独立 frontend baseline | 新建 `.planning/frontend-coverage-baseline.md`,与后端对称并列 | ✓ |
| 合并进 coverage-baseline.md | 把前端段加到后端文件里,会污染后端叙事 | |
| 放 workstream 目录 | v1.28 归档后基线文档会跟着归档,不够长期 | |

**User's choice:** 独立 frontend baseline (Recommended)

---

| Option | Description | Selected |
|--------|-------------|----------|
| vitest exclude 即真相源 | `vitest.config.ts` 的 `coverage.exclude` 作为唯一真相源,脚本做漂移检测 | ✓ |
| 独立白名单配置文件 | 新建清单文件被 vitest 和脚本同时解析,更严格但更复杂 | |

**User's choice:** vitest exclude 即真相源 (Recommended)

---

| Option | Description | Selected |
|--------|-------------|----------|
| 定量触发 + milestone 复审 | 白名单目录覆盖率 ≥70% 可启动移除,每个 milestone 收口强制重审 | ✓ |
| 纯定性 | 仅写“重构后可移除”,无触发线、无固定节奏 | |

**User's choice:** 定量触发 + milestone 复审 (Recommended)

---

| Option | Description | Selected |
|--------|-------------|----------|
| 锁死白名单 | 仅限 cad-editor/cad-elements,新增必须 milestone 级显式决策 | ✓ |
| 开放流程 | PR 中可新增,只要同步更新文档+exclude+面积 ≤5% | |

**User's choice:** 锁死白名单 (Recommended)

**Notes:** 用户四项均选了推荐项。白名单面积固定为 1028/22602 ≈ 4.5%,由 baseline 文档记录,PR 人工审。

---

## Gate 指标维度

| Option | Description | Selected |
|--------|-------------|----------|
| Statements-only | 全局阈值与 per-dir floor 只锁 statements,与后端对称 | ✓ |
| Statements+branches+functions+lines | 保留四项 thresholds,最严格但维护成本高 | |

**User's choice:** Statements-only (Recommended)

---

| Option | Description | Selected |
|--------|-------------|----------|
| 3.8 | 一位小数,与后端格式一致,留 0.05pp 余量 | ✓ |
| 3.5 缓冲 | 留 0.35pp 缓冲,但与实测差距大 | |
| 3.85 照抄 | 原值两位小数,无缓冲,格式不对称 | |

**User's choice:** 3.8 (Recommended)

---

| Option | Description | Selected |
|--------|-------------|----------|
| Lines | git diff 新增/改动行 × vitest json line coverage,ROADMAP 字面 | ✓ |
| Statements | 与全局 statements-only 对称,但 git diff 按行更自然 | |

**User's choice:** Lines (Recommended)

---

| Option | Description | Selected |
|--------|-------------|----------|
| 外部脚本 gate 唯一真相源 | 删除/禁用 vitest thresholds,全部 gate 逻辑放进 bash 脚本 | ✓ |
| 保留 vitest thresholds | 本地 test:coverage 直接红,但需双轨维护 | |

**User's choice:** 外部脚本 gate 唯一真相源 (Recommended)

**Notes:** 用户四项均选了推荐项。全局阈值取一位小数 3.8,与后端 `.coverage-threshold` 格式一致。

---

## Claude's Discretion

None — user made explicit choices in all four gray areas.

## Deferred Ideas

- 前端具体测试补齐 —— Phase 83-87 范围。
- 公共测试 harness 沉淀 —— Phase 83 成功标准之一。
- CI timeout/cache/sharding 优化 —— 若 15min 不足,在 Phase 82 执行末尾或后续 phase 处理。
