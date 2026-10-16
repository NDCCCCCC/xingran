# Phase 40: Tech-Debt Cleanup (技术债清理) - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in `40-CONTEXT.md` — this log preserves the alternatives considered.

**Date:** 2026-06-25
**Phase:** 40-Tech-Debt Cleanup
**Areas discussed:** 4 (提交粒度 / awaiting_human_verify 验证 / frontmatter 规范 / knowledge-base 与状态值归一)

---

## Area 1: 提交粒度与 commit 策略

| Option | Description | Selected |
|--------|-------------|----------|
| 逐个原子 commit (22 个) | 每会话一个独立 fix(40) commit | ✓ |
| 按域批量 (~5-7 commit) | AD/VDI/infopoint/captcha/api-key/frontend 域内 batch | |
| 按 TECH 需求批量 (3 commit) | TECH-01/02/03 各一 commit | |
| 我让 Claude 决定 | (Claude 推荐按域批量,与 a860b0a2 一致) | |

### Q1: 22 个 debug 修复的基础粒度
**User's choice:** 逐个原子 commit (22 个)
**Notes:** 与项目 `a860b0a2` 传统不完全一致(那是 1 个 bulk commit),但用户选更细粒度,理由是 git log 精确回滚

### Q2: commit message 关联 debug session slug
**User's choice:** header 中明写 slug, body 引用 plan

| Option | Description | Selected |
|--------|-------------|----------|
| header 中明写 slug, body 引用 plan | `fix(40): ad-connection-ldap-49-invalid-credentials — admin 账号池凭证解密` | ✓ |
| 只写主题, slug 放 footer | header 简洁 | |
| 用 commit body footer Refs: | `Refs: .planning/debug/<slug>.md` | |
| 我让 Claude 决定 | | |

### Q3: 全量验证时机
**User's choice:** 每个 fix 之后(严格)

| Option | Description | Selected |
|--------|-------------|----------|
| 每个 fix 之后 (严格) | 22 次手动点 + 22 次 build | ✓ |
| 全部完成后再跑 (快速) | 1 次全量 build,违反 CLAUDE.md 习惯 | |
| 混合: 按域中间点 | 中间点跑验证 | |
| 我让 Claude 决定 | | |

### Q4: 推送策略
**User's choice:** 本地完成, 一次性推 main

| Option | Description | Selected |
|--------|-------------|----------|
| 本地完成, 一次性推 main | 22 commit 一次性 git push | ✓ |
| PR 分批推送 | 批次 PR 需 review,负担重 | |
| worktree 隔离 | 需手动管理分支 | |
| 我让 Claude 决定 | | |

---

## Area 2: awaiting_human_verify 验证深度

| Option | Description | Selected |
|--------|-------------|----------|
| build + 手动浏览器验证 | 1 个 build + 4 个 dev 手动点击 | ✓ (Claude 推荐) |
| 仅 build 验证 | 不推荐,运行时行为 build 不可见 | |
| 写自动化测试覆盖 | E2E 框架,属新能力 | |
| 我让 Claude 决定 | | |

### Q1: 验证手段
**User's choice:** 我让 Claude 决定 → Claude 推荐 build + 手动浏览器验证
**Claude 推荐理由:** 5 个会话中 1 个纯 build,4 个运行时行为必须 dev 验证;memory 提示"awaiting_human_verify→resolved 需强约束后端冒烟测试通过";不扩大范围到 E2E 框架

### Q2: 验证 fail fallback
**User's choice:** 反复修复到通过

| Option | Description | Selected |
|--------|-------------|----------|
| 反复修复到通过 | 反复到通过,phase 范围可能变大 | ✓ |
| 留 awaiting_human_verify 延后 | 不带走遗留问题 | |
| 降级为 investigating | 审计仍能看到跟踪 | |
| 我让 Claude 决定 | | |

### Q3: 验证证据位置
**User's choice:** 我让 Claude 决定 → Claude 推荐追加到 .planning/debug/*.md Resolution 章节
**Claude 推荐理由:** md 自然归宿,evidence 与会话同位,git log 可看更新时点,与现有 Resolution/Verification/Files Changed 模式一致

### Q4: dev 运行环境
**User's choice:** 我让 Claude 决定 → Claude 推荐用现有 dev 环境顺序验证
**Claude 推荐理由:** 5 个会话 fix 已落 commit,但 awaiting_human_verify 意味着 dev 复现未做;运行时行为必须 dev 跑通才算闭环;AD 依赖的会话反复处理

---

## Area 3: frontmatter 规范与脚本设计

| Option | Description | Selected |
|--------|-------------|----------|
| resolved/ 风格 (130/143) | 多数已用,迁移成本低 | ✓ (Claude 推荐) |
| metadata.* 风格 | 仅 1/143 验证迁移 142 个 | |
| 混合: 两种都接受 | validator 复杂 | |
| 我让 Claude 决定 | | |

### Q1: frontmatter 范式
**User's choice:** 我让 Claude 决定 → Claude 推荐 resolved/ 风格
**Claude 推荐理由:** 130/143 已用;gsd-sdk audit-open 按字面值读;skip_audit 加顶层字段即可不需 metadata 嵌套

### Q2: validator 严格度
**User's choice:** 我让 Claude 决定 → Claude 推荐双模式
**Claude 推荐理由:** 默认 warn-only + --strict 双模式,Phase 40 验收 100% pass + 未来 pre-commit/CI 都能用

| Option | Description | Selected |
|--------|-------------|----------|
| 硬门 (exit 1) | 任何违规 exit 1,CI 硬门 | |
| warn-only (echo + exit 0) | 仅 dev 验证 | |
| 双模式 (--strict) | 默认 warn + --strict 启用 exit 1 | ✓ (Claude 推荐) |
| 我让 Claude 决定 | | |

### Q3: validator 覆盖范围
**User's choice:** 我让 Claude 决定 → Claude 推荐全量
**Claude 推荐理由:** Phase 40 验收 100% pass,需捕获 4 个无 frontmatter + 1 个 metadata.* + 1 个连字符 status + ISO 8601 日期 + skip_audit

| Option | Description | Selected |
|--------|-------------|----------|
| 仅状态值枚举 | 实现简单,4 个无 frontmatter 不报 | |
| 状态 + 必填字段 | 中等覆盖 | |
| 全量 (含 skip_audit + 日期) | 全部覆盖 | ✓ (Claude 推荐) |
| 我让 Claude 决定 | | |

### Q4: validator 运行时机
**User's choice:** 我让 Claude 决定 → Claude 推荐仅手动运行
**Claude 推荐理由:** 项目无 pre-commit 框架,CI 仅 frontend-build;加新 framework 属新能力(技术债清理范围外);未来加 pre-commit/CI 是 deferred

| Option | Description | Selected |
|--------|-------------|----------|
| 仅手动运行 | phase 40 结束前跑一次 | ✓ (Claude 推荐) |
| 手动 + pre-commit hook | 需修 .pre-commit-config.yaml | |
| 手动 + pre-commit + CI | 三重保障 | |
| 我让 Claude 决定 | | |

---

## Area 4: knowledge-base.md 与状态值归一

| Option | Description | Selected |
|--------|-------------|----------|
| 加 skip_audit 顶层字段 | 文件位置不变,加 frontmatter 字段 | ✓ (Claude 推荐) |
| 移出 .planning/debug/ | rename/move 到 .planning/knowledge/ | |
| 双保险 (skip_audit + 移走) | 双重动作 | |
| 我让 Claude 决定 | | |

### Q1: knowledge-base.md 处置
**User's choice:** 我让 Claude 决定 → Claude 推荐加 skip_audit 顶层字段
**Claude 推荐理由:** 保持 GSD 路径约定,加字段最显式最小变更,validator 识别后跳过

### Q2: apikey-route-path-duplication status 值
**User's choice:** 我让 Claude 决定 → Claude 推荐改 root_cause_found
**Claude 推荐理由:** 130/143 用下划线,apikey-route-path-duplication 是唯一异类;gsd-sdk 按字面值读;改为下划线与规范一致;1 行改动在 TECH-04 范围

| Option | Description | Selected |
|--------|-------------|----------|
| 改 root_cause_found | 与 130/143 一致,validator 只接受一种值 | ✓ (Claude 推荐) |
| 两值都接受 | 不动文件,validator 容忍 | |
| 查 gsd-sdk 容忍度 | 实际 audit-open 显示按字面值读,容忍度需源码确认 | |
| 我让 Claude 决定 | | |

### Q3: 6 个 frontmatter 规范化时机
**User's choice:** 我让 Claude 决定 → Claude 推荐 1 个 batch commit
**Claude 推荐理由:** 6 个同类型变更,延续 `a860b0a2 docs(debug): bulk-defer 22 个` 传统(同类型聚合);commit body 列 6 个文件;与 22 个 atomic fix 清晰分离

| Option | Description | Selected |
|--------|-------------|----------|
| 单独 commit (与代码 fix 分开) | 6 个独立 frontmatter-only commit | |
| 一个 batch commit (docs(40)) | 1 个 `docs(40): standardize debug frontmatter` | ✓ (Claude 推荐) |
| 与 22 个会话 fix 一起 | 混在 fix commit 里不原子 | |
| 我让 Claude 决定 | | |

### Q4: 最终验收方式
**User's choice:** 我让 Claude 决定 → Claude 推荐加 verify_phase40.sh
**Claude 推荐理由:** Phase 40 验收 2 条独立标准 (validator 100% + audit-open < 5),1 个 verify 脚本涵盖;不必污染 validator 本体

| Option | Description | Selected |
|--------|-------------|----------|
| script + audit-open 手验 | phase 结束跑两次 | |
| 加验收脚本 (verify_phase40.sh) | 1 脚本涵盖 2 验收 | ✓ (Claude 推荐) |
| 只 validator 够 | validator 嵌 gsd-sdk 调用 | |
| 我让 Claude 决定 | | |

---

## Claude's Discretion

用户对以下决策明确选择"我让 Claude 决定",Claude 推荐方案被接受:

| Decision ID | Topic | Claude Recommendation |
|-------------|-------|----------------------|
| D-05 | awaiting_human_verify 验证手段 | build + 手动浏览器验证 |
| D-07 | 验证证据位置 | 追加到 .planning/debug/*.md Resolution 章节 |
| D-08 | dev 运行环境 | 用现有 dev 环境顺序验证 |
| D-09 | frontmatter 范式 | resolved/ 风格 |
| D-10 | validator 严格度 | 双模式 (--strict) |
| D-11 | validator 覆盖 | 全量 (含 skip_audit + 日期) |
| D-12 | validator 运行时机 | 仅手动运行 |
| D-13 | knowledge-base.md 处置 | 加 skip_audit 顶层字段 |
| D-14 | apikey-route-path-duplication status | 改 root_cause_found |
| D-15 | 6 个 frontmatter 规范化时机 | 1 个 batch commit |
| D-16 | 最终验收方式 | 加 verify_phase40.sh |

## Deferred Ideas

| 想法 | 原因 | 类别 |
|------|------|------|
| 加 pre-commit hook 自动运行 validator | 项目无 pre-commit 框架,属新能力 | out of scope (v1.17+) |
| 加 GitHub Actions 跑 validator | 项目 CI 仅 frontend-build,加新 workflow 属新能力 | out of scope (v1.17+) |
| 写 E2E 自动化测试覆盖 awaiting_human_verify | 需 vitest / Playwright 框架,属新能力 | out of scope (v1.17+) |
| 改 audit-open 工具支持 `root-cause-found` 别名 | D-14 已用规范值修正,无需改工具 | n/a |
| bulk-defer 模式(把所有 22 修复放一个 mega commit) | 已选 D-01 逐个原子 commit | n/a |
