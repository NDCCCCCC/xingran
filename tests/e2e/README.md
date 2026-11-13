# tests/e2e 目录说明

本目录存放端到端（e2e）验证资产。分层与命名遵循仓库根 `AGENTS.md`：前端 e2e 用
`*.spec.ts`，后端 e2e 用 `*_test.go` / `*.sh`；统一用 `tests/`（复数），禁止 `test/`。

## 目录内权威文件

| 文件 | 用途 | CI 是否执行 |
| --- | --- | --- |
| `operlog_e2e_verify.sh` | 操作日志 LIVE e2e 唯一权威入口（AGENTS.md 登记） | 由维护者按需在具备服务的环境手动运行，非 CI 步骤 |
| `operlog_e2e_verify_test.go` | 操作日志 LIVE e2e 的 Go 驱动 | 同上 |
| `README.md` | 本说明 | — |

## CI 现状（重要）

- `.github/workflows/ci.yml` 的 frontend job 在 `xingran-react-frontend/` 下执行
  `npm run test:coverage -- --shard=N/4`（vitest 4，`vitest.config.ts` root 为前端子目录）。
- 仓库根 `tests/e2e/` **不在** vitest 的扫描范围内；后端 job 跑 `go test`（按包）。
- 因此置于本目录的 `*.spec.ts` 若未显式接入 workflow，CI 永远不会执行它。

## 处置记录：login-encryption.spec.ts（2026-11-12）

原 `tests/e2e/login-encryption.spec.ts` 曾 `import { test, expect } from '@playwright/test'`，
但两个 `package.json` 均无 `@playwright/test`、仓库无 `playwright.config.*`、任何 workflow
都没有执行该文件的步骤 —— 是一个「import 了未安装依赖且 CI 从不执行」的孤儿 spec。
本轮将其移除，使「CI 里存在从未执行的文件」这一事实消失（价值不丢弃，见下）。

登录请求加密契约现由两处**真实被 CI 执行**的资产覆盖：

- `xingran-react-frontend/src/lib/api.ts` 请求拦截器：`X-Request-Encrypted` 头、
  `{ encrypted, data, sm4Key, iv, timestamp, nonce }` 加密体、SM2/SM4 引导与「解密失败」重放。
- `xingran-react-frontend/src/pages/login/__tests__/login-encryption.smoke.test.ts`：
  零新依赖的 Vitest 冒烟，把原 spec 的核心断言转为源码契约锁，随 `npm run test:coverage`
  在 CI 真正执行。

## 若需恢复真正的浏览器 e2e（可选 Playwright 依赖）

当前**不引入** `@playwright/test`（保守路径，不改 `package.json` / `package-lock.json`）。
如后续确需浏览器级登录加密验证，按下列步骤接线，**全部满足后才可把 `*.spec.ts`
重新提交到仓库**：

1. 安装可选依赖（在前端子目录，会同步更新 `package.json` 与 `package-lock.json`）：
   ```bash
   cd xingran-react-frontend
   npm i -D @playwright/test
   npx playwright install chromium
   ```
2. 新建 `xingran-react-frontend/playwright.config.ts`：`baseURL` 指向 dev server
   （`http://localhost:4000`），用 `webServer` 同时拉起 vite 前端与后端 `:9000`。
3. 本地运行：`npx playwright test`（需前后端服务均已就绪）。
4. **CI 接线条件**（缺一不可，否则文件再次成为孤儿）：
   - `package.json` 的 `devDependencies` 含 `@playwright/test` 且 lock 已提交；
   - workflow 新增步骤 `npx playwright install --with-deps chromium`；
   - workflow 显式启动前后端服务（或由 `webServer` 托管）；
   - workflow 新增 `npx playwright test` 步骤，失败即让 job 变红；
   - 该 spec 不再 import 任何未声明依赖。
5. 未满足 1–4 前，不要在仓库保留 `import '@playwright/test'` 的 `.spec.ts` 文件。
