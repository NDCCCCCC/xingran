# Phase 40: Tech-Debt Cleanup (技术债清理) - Context

**Gathered:** 2026-06-25
**Status:** Ready for planning

<domain>
## Phase Boundary

把 v1.15 累积的 22 个 deferred debug 会话(16 个 `root_cause_found` + 5 个 `awaiting_human_verify` + 1 个 `root-cause-found`)与 6 个 audit 数据质量问题集中处理,让系统回到"无遗留债务"状态,使 `gsd-sdk query audit-open` 输出 `debug_sessions < 5`(从当前 57 大幅下降),并建立可持续的 frontmatter 校验。

**硬约束:**
- 不新增功能(纯清理 milestone,功能开发推 v1.17+)
- 不修改 `gsd-sdk` 工具
- 不破坏现有 dev 流程(22 修复 + 6 frontmatter 规范化 + 1 验收脚本 = 29 commit 内完成)
- 不引入 pre-commit / CI 新框架(项目无 `.pre-commit-config.yaml`,CI 仅 `frontend-build.yml`)

</domain>

<decisions>
## Implementation Decisions

### 提交粒度与 commit 策略 (Area 1)

- **D-01:** **逐个原子 commit(22 个)**
  - 每个 debug 会话一个独立 `fix(40): <slug> — <主题>` commit
  - 强可追溯,git log 精确回滚单个修复
  - 22 个 commit 是项目 `a860b0a2 docs(debug): bulk-defer 22 个` 传统的延续(分阶段处理 22 个,而不是再一次 bulk)

- **D-02:** **Commit 格式:`fix(40): <slug> — <主题>`,body 引用 plan**
  - header 明写 session slug,如 `fix(40): ad-connection-ldap-49-invalid-credentials — admin 账号池凭证解密`
  - body 完整说明修复 + 关联的 plan 文件
  - header 偏长但 git log --grep 一查即得

- **D-03:** **每个 fix 之后跑 `go build ./...` / `npm run build` / 必要 `go test`(严格)**
  - 每个 fix commit 之后立即验证
  - 符合项目 CLAUDE.md "Build check before commit" 习惯
  - fail 立即修复,不累积问题

- **D-04:** **本地 22 commit,一次性 `git push origin main`**
  - 完成后整体推送
  - 不开 PR review(22 个独立 PR 负担过重)
  - 不开 worktree(本地顺序处理足够)

### awaiting_human_verify 验证深度 (Area 2)

- **D-05:** **build + 手动浏览器验证 (Claude 推荐)**
  - `workstation-device-build-errors` 是 TS 编译错误,`npm run build` 即可
  - 其余 4 个(展开子表格、重置主题、筛选保留、HMR 触发)是运行时行为,build 通过≠行为正确,必须 dev 浏览器手动点击
  - 不扩大范围到 E2E 框架(新能力,不属于技术债清理)
  - 项目 memory 提示:"awaiting_human_verify→resolved 需强约束后端冒烟测试通过"——指向 build+smoke

- **D-06:** **验证 fail → 反复修复到通过**
  - 反复直到 dev 验证通过才标 resolved
  - 不推迟到后续 phase
  - 不降级为 investigating(避免 audit-open 数字不下降)

- **D-07:** **验证证据追加到 `.planning/debug/<slug>.md` Resolution 章节 (Claude 推荐)**
  - md 是会话的自然归宿,evidence 与会话同位
  - git log 可看更新时点,frontmatter `status: resolved` 反映状态
  - 与现有 Resolution/Verification/Files Changed 模式一致

- **D-08:** **dev 环境用现有 dev 顺序验证 (Claude 推荐)**
  - 5 个会话的 fix 都已落 commit,awaiting_human_verify 状态意味着 dev 复现未做
  - 运行时行为必须 dev 跑通才算闭环
  - AD 依赖的会话如果 dev 没有 AD config,反复处理到通过(不延后)
  - 不构建最小化环境或跳验证

### frontmatter 规范与脚本设计 (Area 3)

- **D-09:** **统一到 resolved/ 风格 (Claude 推荐)**
  - 130/143 文件已经在用 `slug/status/trigger/created/updated/session_type` 模式
  - 1 个 metadata.* 风格 (`info-point-type-null-import`) + 4 个无 frontmatter (login-400/ops-asset-constraint/sys-mac-filter-rules/knowledge-base) 统一
  - `skip_audit` 字段加在 frontmatter 顶层(不嵌套 metadata)

- **D-10:** **双模式 validator:`scripts/validate_debug_frontmatter.sh` (默认 warn-only + `--strict` exit 1) (Claude 推荐)**
  - 默认 warn-only 输出 pass rate,适配 dev 验证
  - `--strict` 启用 exit 1,适配 pre-commit / CI
  - 灵活但不复杂

- **D-11:** **validator 全量覆盖 (Claude 推荐)**
  - 状态枚举验证(合法值: `resolved` / `root_cause_found` / `awaiting_human_verify` / `investigating` / `verifying` / `fixed` / `diagnosed` / `debug_complete` / `root_cause_identified` / `fix_applied`)
  - 必填字段存在验证(`slug` / `status` / `trigger` / `created` / `updated`)
  - 日期格式验证(ISO 8601)
  - slug 格式验证(小写 + 连字符)
  - `skip_audit: true` 字段识别(validator 跳过)
  - frontmatter 解析错误检测(无 `---` 边界)

- **D-12:** **仅手动运行 (Claude 推荐)**
  - 项目无 `.pre-commit-config.yaml`,无现成 pre-commit 框架
  - CI 仅 `.github/workflows/frontend-build.yml`
  - 加 pre-commit / 新 CI workflow 属新能力(技术债清理范围外)
  - phase 40 验收 = "全量扫描通过率 100%" + "audit-open debug_sessions < 5",手动跑一次足够
  - 未来加 pre-commit/CI 是 deferred idea

### knowledge-base.md 与状态值归一 (Area 4)

- **D-13:** **`knowledge-base.md` 加 `skip_audit: true` 顶层字段 (Claude 推荐)**
  - 保持文件名不变
  - 加 frontmatter `skip_audit: true` 顶层字段(不嵌套 metadata)
  - validator 识别后跳过
  - 不移走/不重命名(保持 GSD 知识库可发现性)

- **D-14:** **`apikey-route-path-duplication` 改 `root_cause_found` (Claude 推荐)**
  - 当前是 `root-cause-found`(连字符),与 130/143 用 `root_cause_found`(下划线)不一致
  - gsd-sdk 按字面值读并计为 open debug_session
  - 改为下划线与规范一致,validator 只需接受一种值

- **D-15:** **6 个 frontmatter 规范化用 1 个 batch commit `docs(40): standardize debug frontmatter` (Claude 推荐)**
  - 涵盖:4 个无 frontmatter (login-400 / ops-asset-constraint / sys-mac-filter-rules / knowledge-base)+ 1 个 metadata.* (info-point-type-null)+ 1 个连字符 status (apikey-route-path-duplication)
  - 延续 `a860b0a2 docs(debug): bulk-defer 22 个` 传统(同类型变更聚合)
  - commit body 列出 6 个文件路径
  - 与 22 个 code fix 原子提交清晰分离(29 commit 总数)

- **D-16:** **`scripts/verify_phase40.sh` 验收脚本 (Claude 推荐)**
  - 涵盖 Phase 40 验收 2 条独立标准:validator 100% pass + audit-open debug_sessions < 5
  - phase 40 结束跑一次即可
  - 不必污染 validator 本体

### Claude's Discretion

下列决策由 Claude 在 discussion 中推荐,用户接受("我让 Claude 决定"):
- D-05: build + 手动浏览器验证
- D-07: 验证证据追加到 md Resolution 章节
- D-08: dev 环境用现有 dev 顺序验证
- D-09: 统一到 resolved/ 风格
- D-10: 双模式 validator
- D-11: validator 全量覆盖
- D-12: 仅手动运行
- D-13: 加 skip_audit 顶层字段
- D-14: 改 root_cause_found
- D-15: 1 个 batch commit
- D-16: 加 verify_phase40.sh 验收脚本

### Folded Todos

**已折叠的 todo**:`operlog-exclude-paths.md` (operlog.exclude_paths 配置驱动白名单,score 0.6)
- **决定**:不折叠到 Phase 40
- **理由**:operlog.exclude_paths 已在 Phase 35 (35-operlog-exclude-paths) 完成 (见 `.planning/phases/35-operlog-exclude-paths/` 目录),与 Phase 40 清理范畴无关(是 2026 旧 todo 但已被 Phase 35 覆盖)。保持原状,audit-open 计数会自然下降。

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase 0 规划文档
- `.planning/ROADMAP.md` — Phase 40 目标和 8 条 success criteria
- `.planning/REQUIREMENTS.md` — TECH-01..05 五项需求
- `.planning/PROJECT.md` — v1.16 上下文 + 项目核心价值
- `.planning/STATE.md` — v1.16 planning ready 状态

### 现有 debug 会话与 frontmatter 规范
- `.planning/debug/resolved/` 子目录 — 130/143 文件的 frontmatter 规范范本(`slug/status/trigger/created/updated/session_type`)
- `.planning/debug/resolved/asset-export-404-error.md` — 典型 resolved 风格范例
- `.planning/debug/ad-connection-ldap-49-invalid-credentials.md` — 16 个 root_cause_found 之一,需 fix(40) 处理
- `.planning/debug/reset-theme-hardcoded-colors.md` — 5 个 awaiting_human_verify 之一,需 dev 复现验证
- `.planning/debug/apikey-route-path-duplication.md` — 唯一 `root-cause-found` 异类,需改正
- `.planning/debug/info-point-type-null-import.md` — 唯一 `metadata.*` 风格异类,需转 resolved/ 风格
- `.planning/debug/login-400-bad-request.md` — 无 frontmatter,需补
- `.planning/debug/ops-asset-constraint-uni-ops-asset-devicesn-not-exist.md` — 无 frontmatter,需补
- `.planning/debug/sys-mac-filter-rules-relation-does-not-exist.md` — 无 frontmatter,需补
- `.planning/debug/knowledge-base.md` — GSD 知识库文件,加 `skip_audit: true`

### 项目规范与历史
- `CLAUDE.md` — "Build check before commit" 习惯 + "Always ask user before committing" 原则
- `CLAUDE.md` § "操作日志记录约定" — operlog 模式参考
- `.planning/PROJECT.md` § "Known technical debt (post-v1.15)" — 103 个 audit-open 项目的来源说明
- `.planning/phases/39-workstation-dept-location-alias/39-CONTEXT.md` — 最新已 ship phase 的 CONTEXT 范例

### 项目历史 commit 模式
- `a860b0a2` — `docs(debug): bulk-defer 22 个旧 debug 会话 → v1.16-tech-debt 清理候选`(batch commit 传统)
- `506865dd` — `fix(39 advisory): CR-02 + CR-03a/b` (Phase 39 修复 commit 风格)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **`.planning/debug/resolved/`** — 130/143 已有 frontmatter 范例,可直接 Read 作为规范化模板
- **`scripts/` 目录** — 项目已存在的脚本位置 (`scripts/generate_swagger.sh`, `scripts/check-bundle.sh` 等)
- **`gsd-sdk query audit-open`** — 现有 audit 工具,提供 `debug_sessions` 计数
- **`a860b0a2` commit 模式** — batch 文档提交可参考(同类型变更聚合)

### Established Patterns
- **commit 格式**: `fix(<phase-num>): <slug> — <主题>` (Phase 39 风格)
- **frontmatter 范式**: `slug/status/trigger/created/updated/session_type` (130/143 一致)
- **Phase 39 验收 8 步**:`autopass AC-01..08` + UAT 流程可作为 Phase 40 验收参考

### Integration Points
- **`gsd-sdk query audit-open`** — Phase 40 验收时输出 `debug_sessions < 5`
- **`scripts/validate_debug_frontmatter.sh`** — 新建,挂 `scripts/` 目录
- **`scripts/verify_phase40.sh`** — 新建,挂 `scripts/` 目录,phase 结束跑一次
- **`.planning/debug/`** — 143 个文件位置,Phase 40 操作范围

</code_context>

<specifics>
## Specific Ideas

- Phase 39 的 8 项 AC 验证模式可作为 Phase 40 验收参考(但 Phase 40 验收更简单,只需 2 步:validator 100% + audit-open < 5)
- 22 个 debug 修复 + 6 个 frontmatter 规范化 + 1 个 batch docs commit + 1 个 validator 脚本 + 1 个 verify 脚本 = 30-31 commit 总数(包含 22 atomic fix + 1 batch + 1 docs + 1-2 scripts)

</specifics>

<deferred>
## Deferred Ideas

| 想法 | 原因 |
|------|------|
| 加 pre-commit hook 自动运行 validator | 项目无 pre-commit 框架,属新能力 |
| 加 GitHub Actions 跑 validator | 项目 CI 仅 frontend-build,加新 workflow 属新能力 |
| 写 E2E 自动化测试覆盖 awaiting_human_verify | 需 vitest / Playwright 框架,属新能力 |
| 改 audit-open 工具支持 `root-cause-found` 别名 | D-14 已用 D-13 规范值 (root_cause_found) 修正,无需改工具 |
| bulk-defer 模式(把所有 22 修复放一个 mega commit) | 已选 D-01 逐个原子 commit |

</deferred>

---

*Phase: 40-Tech-Debt Cleanup*
*Context gathered: 2026-06-25*
