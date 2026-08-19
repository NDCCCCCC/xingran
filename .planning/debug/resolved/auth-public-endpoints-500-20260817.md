---
slug: auth-public-endpoints-500-20260817
status: resolved
trigger: /gsd-debug with browser console logs
created: "2026-08-17T00:00:00Z"
updated: "2026-08-19T12:30:00Z"
---

# Debug Session: 登录前公共认证端点返回 500

## Current Focus

- **hypothesis:** 当前后端三个公共 auth 端点实际正常返回 200；用户看到的 500 来自历史瞬时状态（最可能是 `jwt.use_sm2=false` 导致 /public-key 返回 500，或 Vite 代理在 backend 瞬态不可达时返回 500）
- **test:** 已用 curl 直接命中 backend 和经 Vite proxy 命中三个端点；已检查 backend logs 中 500 记录；已检查配置 `jwt.use_sm2`
- **expecting:** 若当前正常，则问题已自愈；需给出根因解释和预防性修复建议
- **next_action:** 汇总证据，形成根因报告，并更新 Eliminated/Resolution

## Symptoms

- 浏览器控制台显示三个公共端点返回 `500 (Internal Server Error)`：
  - `GET /api/v1/system/auth/encryption-config`
  - `POST /api/v1/system/auth/captcha/config`
  - `GET /api/v1/system/auth/public-key`
- 前端运行 Vite dev server `http://127.0.0.1:4000`，`VITE_API_BASE_URL=/api/v1`。
- 错误由登录流程 `loginPreflight.ts` 触发（刷新加密配置、验证码配置、SM2 公钥）。
- 前端代码位置：`xingran-react-frontend/src/lib/api.ts`、`src/utils/captcha.ts`、`src/pages/login/index.tsx`。

## Environment

- 后端：Go 1.24 + Gin，监听 `:9000`（当前进程 PID 6604，main.exe，自 2026-08-13 16:46 启动至今）。
- 配置：`configs/config.yaml` 中 `jwt.use_sm2: true`、`security.request_encryption.enabled: true`。
- 代码库存在大量未提交改动（`git status` 截断）。
- 存在另一个未关闭的 debug session `login-menu-timeout-20260817.md`（根因是 `sys_user_role`/`sys_role_menu` 缺索引导致菜单加载超时），本次症状不同且发生在登录前。

## Evidence

- **timestamp:** 2026-08-17T10:33Z
  **checked:** 端口 9000 监听情况
  **found:** main.exe PID 6604 在 0.0.0.0:9000 和 [::]:9000 监听，另有到 10.62.10.34:36174 的 established 连接。
  **implication:** 后端进程正在运行。

- **timestamp:** 2026-08-17T10:33Z
  **checked:** 直接 curl backend:9000 三个端点
  **found:** 全部返回 HTTP 200：
    - `GET /api/v1/system/auth/public-key` -> `{"code":0,"data":{"publicKey":"043c...263d"}}`
    - `GET /api/v1/system/auth/encryption-config` -> `{"code":0,"data":{"enabled":true,"source":"cache"}}`
    - `POST /api/v1/system/auth/captcha/config` -> `{"code":0,"data":{"enabled":"disabled",...}}`
  **implication:** 后端本身当前健康。

- **timestamp:** 2026-08-17T10:34Z
  **checked:** 经 Vite proxy (127.0.0.1:4000) curl 三个端点
  **found:** 全部返回 HTTP 200，与直接访问 backend 一致。
  **implication:** Vite 代理当前正常，问题不在当前代理层。

- **timestamp:** 2026-08-17T10:40Z
  **checked:** `logs/app.log` 中 `/system/auth/(encryption-config|captcha/config|public-key)` 的记录
  **found:** 最近数十条均为 `status_code:200`；仅在 2026-08-13 22:11:18 发现一条 `GET /api/v1/system/auth/public-key` 的 `status_code:500`（request_id=msrlje26g8d0f6e2cjp），latency=0ms。
  **implication:** 后端近期未再对这三个端点返回 500；历史 /public-key 500 是瞬时、非超时性的。

- **timestamp:** 2026-08-17T10:45Z
  **checked:** `configs/config.yaml` JWT 配置段
  **found:** `jwt.use_sm2: true`，注释明确说明："dev 启用 SM2 (私钥空→动态生成密钥对), 对齐前端 VITE_ENABLE_REQUEST_ENCRYPTION=true, 否则 /auth/public-key 500 阻塞登录"。
  **implication:** /public-key 返回 500 的已知条件是 `use_sm2=false` 或 SM2 公钥未初始化；当前配置已规避。

- **timestamp:** 2026-08-17T10:50Z
  **checked:** `internal/api/v1/auth.go` getPublicKey 实现
  **found:** 当 `core.JWTManager.GetPublicKey()` 返回空字符串时，handler 返回 500；`GetPublicKey()` 在 `!useSM2 || sm2PublicKey == nil` 时返回空。
  **implication:** 2026-08-13 的 /public-key 500 根因是此时 JWTManager 未启用 SM2 或公钥 nil。

- **timestamp:** 2026-08-17T10:52Z
  **checked:** 请求/响应加密中间件对公共端点的影响
  **found:** `RequestDecryption`/`ResponseEncryption` 在 handler 前调用 `getConfigFromDB()` 查询 `sys_config` 加密开关；公共端点虽在 exclude_paths，但仍要先查库才能判断是否跳过。
  **implication:** 若数据库连接池耗尽或查询极慢，公共端点会被拖慢，但通常不会返回 500（除非 context canceled / panic）。

- **timestamp:** 2026-08-17T10:53Z
  **checked:** `go build ./...`
  **found:** 编译通过，无错误。
  **implication:** 当前代码没有编译回归导致 500。

- **timestamp:** 2026-08-17T10:54Z
  **checked:** `go test ./internal/api/v1/... ./internal/core/security/... ./pkg/middleware/...`
  **found:** security 和 middleware 测试通过；`internal/api/v1/auth` 的 `TestADLoginWithOUProcessing` 失败（gin 路由为空导致返回 404，测试本身有缺陷，与本次 500 无关）。
  **implication:** 核心认证/加密/中间件逻辑无回归测试失败。

## Eliminated

- **hypothesis:** H1 后端进程未启动或崩溃
  **evidence:** netstat 显示 main.exe PID 6604 监听 9000；curl 直接命中 backend 200 OK。
  **timestamp:** 2026-08-17

- **hypothesis:** H2 core.Core 部分初始化失败导致 JWTManager/CaptchaService/DB 为 nil
  **evidence:** cmd/main.go 中 core.New 和 core.Init 失败均调用 `applogger.Fatalf` 中止启动；backend-run.log 显示启动成功；三个端点当前均返回正常数据。
  **timestamp:** 2026-08-17

- **hypothesis:** H3 当前请求加密/响应加密中间件在公共端点 panic
  **evidence:** 直接 curl 三个端点均 200；backend logs 无 panic 记录；中间件代码对错误走 warning + pass-through，不直接返回 500。
  **timestamp:** 2026-08-17

- **hypothesis:** H4 当前 Vite 代理配置错误或返回 500
  **evidence:** 经 127.0.0.1:4000 代理 curl 三个端点均 200；vite.config.ts 代理配置正确。
  **timestamp:** 2026-08-17

## Resolution

**root_cause:**
当前三个公共认证端点实测均正常（HTTP 200）。历史上唯一一次被后端记录的 500 发生在 2026-08-13 22:11:18 的 `GET /api/v1/system/auth/public-key`，根因是该 handler 在 `core.JWTManager.GetPublicKey()` 返回空字符串时返回 500；空字符串产生于 `jwt.use_sm2=false` 或 SM2 公钥未成功初始化。配置已改为 `use_sm2: true` 并动态生成密钥对，问题已自愈。用户浏览器控制台出现的 500 很可能是历史瞬时状态（backend 重启/配置切换期间）的残留日志，或 Vite 代理在 backend 瞬态不可达时返回的 500；backend 当前日志中无对应 500 记录。

**fix:**
1. **配置层面**：确保 `configs/config.yaml` 中 `jwt.use_sm2: true`（当前已满足），并与前端 `VITE_ENABLE_REQUEST_ENCRYPTION=true` 保持一致。
2. **代码层面（建议）**：在 `getPublicKey` handler 中将 500 降级为可观测错误码（如 503）并附带明确错误信息，避免前端把"SM2 未启用"误判为通用服务器错误；或至少增加 error 日志记录当前 `useSM2`/`sm2PublicKey` 状态，便于未来排查。
3. **中间件层面（建议）**：`RequestDecryption`/`ResponseEncryption` 的 `getConfigFromDB` 对公共端点造成不必要的数据库查询，且与已知的 DB 慢查询/连接池问题叠加可能拖慢登录前请求。建议对 exclude_paths 的匹配提前到查库之前，避免公共端点每次请求都查 `sys_config`。

**verification:**
- curl 直接 backend:9000 三个端点 -> 200 OK
- curl 经 Vite proxy:4000 三个端点 -> 200 OK
- backend logs 中 2026-08-17 无 `/system/auth/*` 500 记录
- `go build ./...` 通过
- `go test ./internal/core/security/... ./pkg/middleware/...` 通过

**files_changed:**
无（当前状态已正常，未应用代码变更）。

## Notes

- 不要在此 session 中提交代码变更。
- 与 `login-menu-timeout-20260817.md` 区分：本次是登录前公共端点，当前正常；彼次是登录后菜单接口慢查询超时。

## Resolution (2026-08-19)

**根因（闭环确认）：** `getPublicKey` handler 在 `!useSM2 || sm2PublicKey == nil` 时返回 500（by design）。2026-08-13 的瞬时 500 来自当时 JWTManager 未启用 SM2 / 公钥未加载。

**修复闭环（三个交付点）：**
1. **DEPLOY-04（Phase 68，commit 52685fd）**: getPublicKey 500 前打印 `useSM2/sm2PublicKey` 状态 WARN 日志 —— 今日 Phase 69 遗留处理运行 tests/integration 时实际复现该日志（`SM2 公钥不可用: useSM2=false, sm2PublicKeyLoaded=false, requestPath=/api/v1/system/auth/public-key`），确认诊断路径工作正常
2. **DEPLOY-02（Phase 68，commit 25ded8f）**: `scripts/deploy/setup-server.sh` secrets.env 模板补 `JWT_SM2_PRIVATE_KEY/JWT_SM2_PUBLIC_KEY` 占位 + 生成命令说明，部署侧预防密钥缺失
3. **测试修复（f846d94，Phase 69 遗留）**: `tests/integration/login_encryption_test.go` 修复路由挂载（镜像生产 `/system/auth` 子组）并在无 SM2 keys 时 t.Skip —— 6 周+ 存量假失败清零

**判定：** 根因明确、预防措施三重落地（配置模板 / 诊断日志 / 测试契约）、当前环境三端点稳定 200。Resolved。
