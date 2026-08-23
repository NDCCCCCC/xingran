# Phase 71: 治理基线 + CI gate - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-20
**Phase:** 71-治理基线 + CI gate
**Areas discussed:** CI 阈值 gate 工具选型, Profile artifact 上传策略, 基线数据存放位置, Ratchet 阈值递进机制

---

## CI 阈值 gate 工具选型

| Option | Description | Selected |
|--------|-------------|----------|
| bash + awk 自检 | 依赖零第三方;直接在 ci.yml 现有 go test 后加一步 `go tool cover -func` + awk 抽加权百分比 + 跟阈值对比 + exit 1。逻辑可见可改,debug 最容易;劣势是 awk 解析可读性低 | ✓ |
| go-test-coverage GitHub Action | 成熟的 [vladopajic/go-test-coverage](https://pkg.go.dev/github.com/vladopajic/go-test-coverage/v2@v2.9.0) Action。只需写 yaml,thresholdFile / threshold / workingDirectory 参数化;内置 per-package threshold、base branch diff coverage、PR comment、badge。劣势:仅支持 total % 或简单 per-package %,**不能直接表达"加权"**;Docker image 依赖 | |
| Custom Makefile target | Makefile + go script 控制逻辑;Makefile 在项目里但不在 ci.yml 里,CI 调用 `make coverage-check`。劣势:增加抽象层,与 yolo 模式的"快速可调"诉求可能冲突 | |

**User's choice:** 锁定 bash + awk;Phase 74 启用 diff coverage 时再评估 vladopajic/go-test-coverage

**Notes:** 起初我推荐 bash+awk 后,用户问"联网搜索是否是最佳方案"。联网搜出 vladopajic/go-test-coverage 远比想象中强大(per-file/per-package threshold、base branch diff、PR comment、badge),但**不支持加权平均**——而 Phase 71 SC-a 明确要求"加权 ≥70%"。权衡后决定 Phase 71 走 bash+awk(精确匹配 SUMMARY 12.8% 加权口径),Phase 74 启用 GOV-03 diff coverage 时(Action 内置 base branch diff)再评估。

---

## Profile artifact 上传策略

| Option | Description | Selected |
|--------|-------------|----------|
| 上传 + 保留 30 天 | actions/upload-artifact@v4 + retention-days: 30。PR 失败时 developer 可下载 coverage.out / coverage.html 调什么包倒退;优点:调试体验最好;劣势:74 包 coverage.out 接近 3MB,占少量 GitHub 存储 | ✓ |
| 仅失败时上传 | 仅 gate 失败时上传 coverage.out + HTML;成功 PR 不占存储;劣势:成功 PR 未来无法回溯当时覆盖率 | |
| 每 PR 都上传 + 保留 7 天 | 同上,但 retention-days: 7(GitHub 1G 免费限额下 30 天可能超额) | |
| 仅 PR 评论贴可视化链接 | 不存 coverage.out,只在 PR comment 里贴 per-package 覆盖率表;轻量,但无法拉原始 profile 重跑 | |

**User's choice:** 上传 + 保留 30 天

**Notes:** 同时上传 coverage.out + coverage.html(后者由 `go tool cover -html` 生成,便于人工浏览器查看)。估算 30 天保留约 8MB/run,远低于 GitHub 1G 限额。

---

## 基线数据存放位置

| Option | Description | Selected |
|--------|-------------|----------|
| .planning/coverage-baseline.md | 新建独立文件,作为 phase 后随手记的"状态快照"起点。独立于 scan,以后看到 .planning/coverage-baseline.md 即可读"v1.26 启动前" vs "Phase 71 后"等多行报表。可与 quick-260820-bcs 保持原始 scan + 本快照互补 | ✓ |
| 回填 quick-260820-bcs | 直接回填 quick-260820-bcs/SUMMARY.md,增加 phase 完后"下一行";劣势:原 SUMMARY 为只读扫描报告,回填会破坏 scan 本身的不变性 | |
| CI workflow artifact | CI 自动生成 artifact (coverage-baseline.md);劣势:需 CI 额外写 logic,且在开发者本地看不见 | |
| CLAUDE.md / ROADMAP 末尾 | 在 .planning/ROADMAP.md 末尾或 CLAUDE.md 记 phase 后数字;劣势:表会越加越长,与其他 phase 内容混在一起 | |

**User's choice:** .planning/coverage-baseline.md(新建独立文件)

**Notes:** quick-260820-bcs SUMMARY.md 不回填,保持 scan 不变性;coverage-baseline.md 作为"phase 推进后的状态快照",与原始 scan 互补。

---

## Ratchet 阈值递进机制

| Option | Description | Selected |
|--------|-------------|----------|
| 手动 bump(commit 时人工) | 每次 phase 完成、Phase 71 闸门实际数字上升后,由 phase execute plan 负责原子提交人工更新 .coverage-threshold(commit message 明示 'coverage ratchet 30.0% → 32.5%')。变更可审计、跟 PR 合并、无魔法。劣势:需人记得更新(但这正是 Ratchet 价值所在——bump 是可控动作不是自动运行) | ✓ |
| 自动计算上一 phase 数字 | CI 自动从 .planning/coverage-baseline.md 读上一行数字,写回 .coverage-threshold;完全免人参与。劣势:阈值与 baseline file 双向同步,出现"谁负责?"的责任边界模糊;以及快路径变更不可审计 | |
| CI PR 评论提示 bump | 不动 .coverage-threshold,但每次 CI 失败时 comment "推荐新阈值 25.0%" 提醒 phase plan 作者;threshold 必须手动提交。优势:半自动提示 + 人工可控 | |
| 三个方案组合 | 手动 bump + CI 校验上一 phase 数字与阈值一致 + comment 提示偏差。前期最重,Phase 71-74 需要额外一轮 sync | |

**User's choice:** 手动 bump(commit 时人工)

**Notes:** 每次 phase execute plan 原子提交"更新 .coverage-threshold + 在 coverage-baseline.md 追加新行"两件事;commit message 标注 'coverage ratchet X.X% → Y.Y%'。

---

## Claude's Discretion

下列子决策未深入讨论,按默认规则走(若 plan 阶段发现需调整,可修改):

- **Gate fail 时的 override 机制** — 严格无 override,不加 [skip-coverage] label 例外
- **加权口径** — 与 quick-260820-bcs 一致,74 个业务包加权平均,排除 scripts/migrations/cmd main/docs
- **Per-package 倒退 PR comment** — 不强制,仅 CI 日志输出
- **覆盖率扫描时机** — go test 单次而非 race-detector,count=1 防缓存
- **Coverage scope 列表存放** — check-coverage.sh 内置 awk 过滤规则,不引入独立配置文件

## Deferred Ideas

讨论中归类为范围外的想法(已记入 CONTEXT.md `<deferred>`,此处仅列出索引):

- GOV-03 PR 增量 diff coverage ≥80%(Phase 74)
- FUT-02 分支覆盖率
- FUT-03 mutation testing
- FUT-04 覆盖率 PR 评论机器人
- Gate fail override 机制(严格无 override)
- vladopajic/go-test-coverage 引入(Phase 74 重评估)
</content>
</invoke>