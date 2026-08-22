---
status: resolved
trigger: "github上ci流程报错：Run npm run format:check → prettier --check . 报 [warn] src/types/global.d.ts 和 [warn] tsconfig.app.json，Code style issues found in 2 files，exit code 1"
created: 2026-08-23
updated: 2026-08-23
---

# Debug Session: prettier-format-check-fail

## Symptoms

**Expected behavior:**
GitHub Actions CI 中 `npm run format:check`（即 `prettier --check .`）通过，退出码 0。

**Actual behavior:**
CI 在 format:check 步骤失败，报 2 个文件存在格式问题，退出码 1：
```
[warn] src/types/global.d.ts
[warn] tsconfig.app.json
[warn] Code style issues found in 2 files. Run Prettier with --write to fix.
```

**Error messages:**
见 trigger（CI 日志原文）。无其他错误。

**Timeline:**
PR #4（当前工作分支相关，最近两次提交 c95c74b / 8a50662 修复了 CI 的 test 与 staticcheck 失败）的 CI 运行推进到 format:check 步骤后暴露。两个文件最后一次内容改动均为 16a26e1（PR #3，dev 依赖批量升级 + TS 7 迁移）。

**Reproduction:**
已本地复现（orchestrator 验证）：`cd xingran-react-frontend && npx prettier --check src/types/global.d.ts tsconfig.app.json` → 同样 2 个 warn，exit=1。非 CI 环境特有。

## Initial Observations (orchestrator pre-scan)

1. **本地 100% 复现**，与 CI 报错完全一致（同样 2 个文件、exit=1）。本地 prettier 版本 3.9.6（node_modules 实测），CI 与本地走同一 package-lock，版本一致。
2. **git blame**：两个文件最后一次改动都是 `16a26e1`（PR #3，dependabot 名义 + 手工 TS 7 迁移提交）。该 PR：
   - `tsconfig.app.json`：TS 7 移除 `baseUrl`，`paths` 改为 `./src/*`，新增 `files` 数组（3 个 .d.ts）。
   - `src/types/global.d.ts`：`any` → `unknown`、新增注释、新增 `import type {} from "@/pages/operations/building-spaces-3d/components/HubeiMapGL"`。
3. **具体格式差异（npx prettier <file> | diff 实测）**：
   - `tsconfig.app.json`：`"files": [...]` 数组当前为多行展开，prettier 3.9.6 要求折叠为单行（总长未超 printWidth）。
   - `src/types/global.d.ts`：存在 `}// 详细对象挂载点...`（右花括号后紧贴行尾注释）等不符合 prettier 输出的写法。
4. **推断根因**：PR #3 的手工改动提交前未运行 `prettier --write`。CI 的 format:check 步骤在 PR #3 合并后首次真正跑过全量检查（或 PR #4 的 CI 修完 test/staticcheck 后才推进到该步骤），从而暴露。

## Current Focus

- hypothesis: CONFIRMED & FIXED & USER-VERIFIED — PR #3 (16a26e1) 手工 TS 7 迁移改动未经 prettier 格式化即提交。
- test: 已完成（format:check 全仓 exit=0、type-check exit=0、lint exit=0；用户在 orchestrator 检查点确认 "confirmed fixed"）。
- expecting: 无（会话关闭）。
- next_action: 无 — 已提交并归档。后续观察 PR #4 CI format:check 步骤转绿。

## Evidence

- timestamp: 2026-08-23 本地复现：`npx prettier --check src/types/global.d.ts tsconfig.app.json` → 2 warn, exit=1（与 CI 一致）。
- timestamp: 2026-08-23 `git log` 显示两文件最后改动均为 16a26e1（PR #3 dev-deps + TS 7 迁移）。
- timestamp: 2026-08-23 `npx prettier tsconfig.app.json | diff` 显示 `files` 数组应折叠为单行。
- timestamp: 2026-08-23 PR #3 diff 显示 global.d.ts 含 `}// 注释` 紧贴写法、tsconfig 新增多行 files 数组，均为手工编辑痕迹。
- timestamp: 2026-08-23 续会话再复现：工作树干净（仅 debug 文件 untracked，HEAD=c95c74b，修复未入库），`npx prettier --check src/types/global.d.ts tsconfig.app.json` → 2 warn, exit=1。假设 CONFIRMED。
- timestamp: 2026-08-23 knowledge-base 无匹配条目（唯一条目 adlogin-ou-test-fail 无关键词重叠）。
- timestamp: 2026-08-23 修复执行：`npx prettier --write src/types/global.d.ts tsconfig.app.json` → 复查 --check 通过 exit=0。git diff 仅 2 处纯格式变更：(1) global.d.ts `}// 注释` → `} // 注释`；(2) tsconfig.app.json `files` 数组折叠为单行。无语义变化，falsification test 通过。
- timestamp: 2026-08-23 回归：全仓 `npm run format:check` → exit=0（无其他 latent 违规）；`npm run type-check` → exit=0；`npm run lint` → exit=0（0 errors，360 warnings 均为存量与本次无关）。CI workflow ci.yml 确认步骤为 format:check → type-check（lint → type-check → vitest → build 顺序）。
- timestamp: 2026-08-23 用户在 orchestrator 检查点确认 "confirmed fixed"；提交前 session-manager 复验：`npx prettier --check .`（全仓）→ "All matched files use Prettier code style!" exit=0。git diff 复核仅上述 2 处纯格式变更。

## Eliminated

## Resolution

root_cause: PR #3 (16a26e1) 的手工 TS 7 迁移对 xingran-react-frontend/tsconfig.app.json 与 src/types/global.d.ts 的改动未经 prettier 格式化即提交（多行展开的 files 数组、`}// 行尾注释` 紧贴等写法不符合 prettier 3.9.6 输出）；CI 的 `prettier --check .` 全量扫描到这两个文件报 Code style issues，exit 1。
fix: `npx prettier --write src/types/global.d.ts tsconfig.app.json`（prettier 3.9.6，与 CI 同 lockfile 版本）。共 2 处变更：global.d.ts 行尾注释前补空格；tsconfig.app.json files 数组折叠为单行。纯格式，无语义变化。
verification: (1) 修复前目标 check 复现 2 warn exit=1；(2) 修复后目标 check exit=0；(3) 全仓 `npm run format:check` exit=0（提交前复验通过）；(4) `npm run type-check` exit=0；(5) `npm run lint` exit=0（仅存量 warnings）；(6) 用户在 orchestrator 检查点确认 "confirmed fixed"。真实 CI 转绿以 PR #4 下次运行为准。
files_changed: xingran-react-frontend/src/types/global.d.ts, xingran-react-frontend/tsconfig.app.json
