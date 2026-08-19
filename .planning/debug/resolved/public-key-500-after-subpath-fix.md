---
status: resolved
trigger: "nginx 子路径修复后登录预检 /xingran/api/v1/system/auth/public-key 返回 500；前端日志 [LoginPreflight] {encryption:'ok', publicKey:'failed', captcha:'ok'}"
created: 2026-08-19
updated: 2026-08-19
---

# Debug Session: public-key-500-after-subpath-fix

## Symptoms

**Expected behavior:**
登录页面打开后，前端调用 `GET /system/auth/public-key` 获取 SM2 公钥用于后续 SM2+SM4 加密 → 返回 200 + 公钥字符串，前端 LoginPreflight 把 publicKey 标记为 'ok'。

**Actual behavior:**
端点返回 500 (Internal Server Error)。前端日志：
```
[LoginPreflight] 登录安全配置刷新失败 {encryption: 'ok', publicKey: 'failed', captcha: 'ok'}
```
即同次预检里 `encryption` 与 `captcha` 都成功（200），唯独 `publicKey` 500。

**Error messages (browser console):**
```
GET https://212.129.154.78:8000/xingran/api/v1/system/auth/public-key 500 (Internal Server Error)
```
(堆栈：axios xhr → request → onFinish，来自 LoginPreflight 提交前的并发预检)

**Timeline:**
- 修复 nginx 子路径（commit b85abd2）并部署后第一次访问登录页时出现
- 静态资源 404 问题已解决（URL 已正确带 `/xingran/api/v1/` 前缀）；这是修复后暴露的下一个问题

**Reproduction:**
- 腾讯云服务器：`http://212.129.154.78:8000/xingran/`
- 打开登录页 → LoginPreflight 自动请求 public-key → 500

**Environment:**
- 前端：https://212.129.154.78:8000/xingran/（nginx 8000 → Go 后端 9000）
- 后端：Go :9000，JWT 双 token + SM2/SM4 国密加密（参考 .planning/debug/resolved/token-lost-on-refresh.md 同源上下文）

## Architecture context

- 端点： `GET /api/v1/system/auth/public-key`（后端 router.go）
- 实现位置： `internal/api/v1/system/auth_handler.go` 或类似（待调试器定位）
- 同族端点： `/system/auth/encryption-config`、`/system/auth/captcha-config`（同次预检返回 ok）
- LoginPreflight 行为：登录提交前并发取三份安全配置（encryption/publicKey/captcha），任一失败仅记录不阻塞；本次唯 publicKey 失败 → 多半是该端点自身缺陷

**Primary hypothesis（待验证）:**
`/system/auth/public-key` handler 实现依赖启动期初始化的某资源（SM2 密钥对缓存？config 字段？），在某种运行条件下抛 500。与加密配置 / 验证码端点的差异需通过查 handler 源码 + 后端日志定位。

**Secondary hypotheses:**
- 端点依赖 `use_sm2=true` 配置，但缺省异常处理/默认值缺失
- 端点路径带新子路径前缀后路由仍命中但 handler 内的某个共享单例/缓存初始化失败
- 与上一个修复（SM2 JWT issuer）同处 `pkg/crypto/sm2_*.go` 系列，SM2 公钥生成/加载可能在某种情况下抛 panic/err

## Current Focus

- hypothesis: 部署服务器 `/opt/xingran/configs/config.yaml` 中 `jwt.use_sm2: false`（最可能：用户复制了 `configs/config.sqlite.example.yaml`，该模板默认 `use_sm2: false` 用于"首跑 HS256 免 SM2 密钥依赖"），导致 `JWTManager.GetPublicKey()` 返回空 → handler 返回 `response.ErrServerError` (HTTP 500)
- test: 已 grep `getPublicKey` handler 实现 + `JWTManager.GetPublicKey()` 实现 + `response.ErrServerError` HTTPStatus + 比对四份 example yaml + 部署脚本
- expecting: 根因在服务器 config 层，不在代码层；用户需在服务器改 `use_sm2: true` 并配置 SM2 keys（env 或留空接受动态生成）
- next_action: 把诊断结论交付给用户 + 给出 ssh 操作步骤

## Evidence

- **timestamp:** 2026-08-19
  **checked:** `internal/api/v1/auth.go` `getPublicKey` handler (lines 500-512)
  **found:** 当 `core.JWTManager.GetPublicKey()` 返回空串时,handler 调用 `response.Error(c, response.ErrServerError, "SM2 未启用或公钥不可用")`。`response.ErrServerError` 在 `pkg/response/response.go:38` 定义为 `Code:500, HTTPStatus:500`。
  **implication:** 该端点 HTTP 500 根因 = `GetPublicKey()` 返回空串。

- **timestamp:** 2026-08-19
  **checked:** `internal/core/security/jwt.go` `GetPublicKey()` (lines 276-283)
  **found:** `if !j.useSM2 || j.sm2PublicKey == nil { return "" }` → 仅当 `useSM2=false` 或 `sm2PublicKey` 为 nil 时返回空。
  **implication:** 服务器 `jwt.use_sm2` 配置决定此端点是否可工作。

- **timestamp:** 2026-08-19
  **checked:** `configs/*.example.yaml` 的 `jwt.use_sm2` 默认值
  **found:**
  - `config.prod.example.yaml:94` → `use_sm2: true`（生产强制）
  - `config.example.yaml:77` → `use_sm2: true`（开发默认）
  - `config.sqlite.example.yaml:58` → **`use_sm2: false`**（"首跑 HS256 免 SM2 密钥生成依赖"）
  **implication:** 用户的腾讯云服务器很可能用了 sqlite 模板（comment 提到"无外网依赖场景"/"腾讯云轻量云"），默认 `use_sm2: false`。该模板定位即"首跑免密钥"但需要后续手动切到 SM2。

- **timestamp:** 2026-08-19
  **checked:** `scripts/deploy/setup-server.sh` 提供的 `/etc/xingran/secrets.env` 模板 (lines 22-38)
  **found:** 模板仅含 `JWT_ACCESS_SECRET/JWT_REFRESH_SECRET/SM4_KEY`,**没有** SM2 密钥 env var 项。
  **implication:** 即使按官方 setup 脚本走,也不会生成/注入 SM2 密钥 — 与 sqlite 模板的 `use_sm2: false` 默认值配套。

- **timestamp:** 2026-08-19
  **checked:** `internal/config/config.go` BindEnv 表 (lines 359-360)
  **found:** SM2 密钥 env var 为 `JWT_SM2_PRIVATE_KEY` / `JWT_SM2_PUBLIC_KEY`(无 `XINGRAN_` 前缀),但三份 example yaml 和 `docs/deployment/secret-management.md` 第 1.2 节都写成 `XINGRAN_JWT_SM2_PRIVATE_KEY` / `XINGRAN_JWT_SM2_PUBLIC_KEY`(带 `XINGRAN_` 前缀)。
  **implication:** 文档与代码不一致 — 这是潜在 bug,但**不直接导致 500**;即使 env var 名称错也只是退到"动态生成密钥对"分支,handler 仍能拿到公钥。属于另一条独立修复线（用户在 confirm 前不动它）。

- **timestamp:** 2026-08-19
  **checked:** `deploy/xingran.service` (lines 18-33)
  **found:** `WorkingDirectory=/opt/xingran`, `EnvironmentFile=/etc/xingran/secrets.env`, `Environment=SERVER_MODE=release`。无 `XINGRAN_*` / `JWT_SM2_*` env var。CI 部署脚本 `deploy-remote.sh` **不**上传/同步 config.yaml — 由运维手动维护。
  **implication:** 服务器 config 是 user-managed,与代码解耦;本次 500 的根因判定就是该文件当前内容,而非代码 bug。

- **timestamp:** 2026-08-19
  **checked:** `git log` 最近改动
  **found:** `b85abd2` 只动前端/embed/nginx;`00bbde6` 改 SM2 refresh token issuer(不影响 /public-key);无后端 auth handler / JWTManager 改动。
  **implication:** 本次 500 不是回归 bug,而是部署 server 的运行时配置问题。

- **timestamp:** 2026-08-19
  **checked:** 兄弟端点工作正常原因
  **found:** `getEncryptionConfig` 仅读 `middleware.GetEncryptionConfigFromCache()`,`getCaptchaConfig` 走 DB。两者都不依赖 `JWTManager.sm2PublicKey`。
  **implication:** 兄弟端点正常反过来印证 /public-key 的 500 来自 SM2 初始化状态,而非全局后端崩溃。

## Specialist Review

**Verdict:** SUGGEST_CHANGE (root cause correct, three corrections to fold in)

**Specialist:** engineering:debug (general — deployment/config issue, not code change)

**Corrections from specialist:**

1. **Blocker — script path is wrong.** Original `scripts/gen-sm2-keys/main.go` does not exist. Verified locally: actual file is `scripts/crypto/gen_sm2_keys/main.go`. The file's header comment (line 2) still says `go run scripts/gen-sm2-keys.go` (stale comment). Working invocation: `go run scripts/crypto/gen_sm2_keys/main.go` (file carries `//go:build ignore`, which only affects `go build`/`go test`, not direct `go run` of an explicit path).
2. **Blocker — option "leave keys empty" silently force-logs-out every user.** `internal/core/security/jwt.go:84-93` + `internal/config/config.go:357-358` warning confirm: `useSM2=true` with empty keys = `crypto.GenerateKeyPair()` at startup → every `systemctl restart` invalidates all SM2-signed tokens, force-logging-out every user. Acceptable only on a fresh install with zero existing sessions. **For the current Tencent Cloud server with active users, must use fixed keys.**
3. **Missing — backup before sed.** Add `sudo cp /opt/xingran/configs/config.yaml /opt/xingran/configs/config.yaml.bak.$(date +%Y%m%d-%H%M%S)` before the `sed`.
4. **Missing — verification too shallow.** 200 on `/public-key` only proves `sm2PublicKey != nil`, not that the hex parses or SM2 sign+verify round-trips. After restart add `sudo journalctl -u xingran -n 100 --no-pager | grep -iE "sm2|jwt|error"`.

**Confirmed correct:** env var names `JWT_SM2_PRIVATE_KEY` / `JWT_SM2_PUBLIC_KEY` (no `XINGRAN_` prefix) match `internal/config/config.go:359-360`. The docs/example-yaml `XINGRAN_*` drift is correctly flagged out of scope per scope constrainment.

**Final server-side command sequence (with corrections applied):**

```bash
# 1. SSH to server
ssh ubuntu@212.129.154.78

# 2. Backup config + inspect current SM2 config
sudo cp /opt/xingran/configs/config.yaml /opt/xingran/configs/config.yaml.bak.$(date +%Y%m%d-%H%M%S)
sudo grep -n "use_sm2\|sm2_private_key\|sm2_public_key" /opt/xingran/configs/config.yaml

# 3. Flip use_sm2 to true
sudo sed -i 's/^\(\s*use_sm2:\s*\)false/\1true/' /opt/xingran/configs/config.yaml

# 4. Generate fixed SM2 key pair LOCALLY first (verify the path on your checkout!)
go run scripts/crypto/gen_sm2_keys/main.go
# Note: this script has //go:build ignore, so go build ./... won't see it
# but go run of the explicit path works. Header comment in the file mentions
# `scripts/gen-sm2-keys.go` which is a stale path comment.

# 5. Paste the generated hex into /etc/xingran/secrets.env on the server
sudo tee -a /etc/xingran/secrets.env >/dev/null <<'EOF'
JWT_SM2_PRIVATE_KEY=<paste-priv-hex-here>
JWT_SM2_PUBLIC_KEY=<paste-pub-hex-here>
EOF
sudo chmod 600 /etc/xingran/secrets.env

# 6. Restart + verify (direct backend + through nginx)
sudo systemctl restart xingran && sleep 3
curl -sS http://127.0.0.1:9000/api/v1/system/auth/public-key
sudo journalctl -u xingran -n 100 --no-pager | grep -iE "sm2|jwt|error"
curl -sS https://212.129.154.78:8000/xingran/api/v1/system/auth/public-key

# Expected for both curls: {"code":0,"data":{"publicKey":"04<hex>"}} HTTP 200
```

## Eliminated

- **hypothesis:** nginx 子路径部署导致 /public-key 路由没命中(URL 错)
  **evidence:** 同次预检的 /encryption-config 和 /captcha-config 都返回 200,说明 /xingran/api/v1 前缀和 router 都工作。404 已被 b85abd2 修复。
  **timestamp:** 2026-08-19

- **hypothesis:** 后端 panic / 数据库宕机导致 500
  **evidence:** encryption/captcha 端点 200,直接依赖 DB 与 Redis 的 captcha 服务仍能正常返回;无全局性故障特征。
  **timestamp:** 2026-08-19

- **hypothesis:** b85abd2 改动引入了回归
  **evidence:** b85abd2 只改前端、vite.config、embed_frontend_prod.go 和 nginx 子路径前缀;后端 auth handler / JWTManager / config.go 零改动。
  **timestamp:** 2026-08-19

- **hypothesis:** JWTManager.GetPublicKey() 在 SM2 启用但密钥未配置时返回空
  **evidence:** 该情况(jwt.use_sm2=true 且 sm2 keys 空)会触发 NewJWTManager 走"动态生成"分支(crypto.GenerateKeyPair()),生成的 key 会写入 sm2PublicKey,GetPublicKey() 仍能返回非空。**只有 use_sm2=false 才会返回空**。
  **timestamp:** 2026-08-19

## Resolution

**root_cause:**
部署到腾讯云的 `/opt/xingran/configs/config.yaml` 中 `jwt.use_sm2: false`(最可能由用户复制 `configs/config.sqlite.example.yaml` 落地,该模板默认 `use_sm2: false` 用于"首跑免密钥")。`internal/api/v1/auth.go:502` 的 `getPublicKey` handler 调用 `core.JWTManager.GetPublicKey()`,该方法在 `internal/core/security/jwt.go:277-279` 对 `!useSM2 || sm2PublicKey == nil` 返回空串;空串触发 `response.Error(c, response.ErrServerError, ...)`,而 `response.ErrServerError` 在 `pkg/response/response.go:38` 定义 `HTTPStatus:500`。最终 HTTP 500 抵达前端,被 LoginPreflight 记为 `publicKey:'failed'`。兄弟端点 /encryption-config 和 /captcha-config 不依赖 JWTManager,故返回 200。

**fix:**
本根因在部署服务器侧,代码无需修改。用户需在腾讯云服务器上:
1. SSH 到 `ubuntu@212.129.154.78`
2. 备份 `/opt/xingran/configs/config.yaml`(见 Specialist Review §3 备份命令)
3. 编辑 `/opt/xingran/configs/config.yaml`,把 `jwt.use_sm2: false` 改为 `jwt.use_sm2: true`
4. **必须**配置固定 SM2 密钥(不能用"留空"选项,否则每次重启踢全部用户):
   - 在**本地仓库**运行 `go run scripts/crypto/gen_sm2_keys/main.go`(真实路径;文件头注释里的 `scripts/gen-sm2-keys.go` 是过时 comment)生成 hex 私钥/公钥
   - 通过 `/etc/xingran/secrets.env` 注入 `JWT_SM2_PRIVATE_KEY=<priv-hex>` 和 `JWT_SM2_PUBLIC_KEY=<pub-hex>`(注意:**无** `XINGRAN_` 前缀,与文档 `secret-management.md §1.2` 表中拼写不一致 — 见下方"related bugs")
   - `chmod 600 /etc/xingran/secrets.env`
5. `sudo systemctl restart xingran && sleep 3` 后:
   - `curl http://127.0.0.1:9000/api/v1/system/auth/public-key` 验证直连后端返回 `{"code":0,"data":{"publicKey":"04<hex>"}}` HTTP 200
   - `sudo journalctl -u xingran -n 100 --no-pager | grep -iE "sm2|jwt|error"` 应无新错误
   - `curl https://212.129.154.78:8000/xingran/api/v1/system/auth/public-key` 验证经 nginx 也 200
6. 浏览器重新打开 `https://212.129.154.78:8000/xingran/` 登录页,LoginPreflight 三项应全 `ok`

**verification:**
- 完成后用 `curl https://212.129.154.78:8000/xingran/api/v1/system/auth/public-key` 期望返回 `{"code":0,"data":{"publicKey":"04<hex>"}}`,HTTP 200
- 后端日志 `sudo journalctl -u xingran -n 100` 不应再有 `status_code:500` 出现在该路径
- 前端 LoginPreflight 控制台输出三项全 `ok`

**files_changed:**
无（待用户在服务器侧操作完成后,本端点将自动恢复。如用户希望代码层加固,可在 `getPublicKey` handler 增加一条 INFO/WARN 日志记录 `useSM2`/`sm2PublicKey==nil` 状态,便于下次快速定位 — 但这是 symptom-level 可观测性,不是根因修复,需要用户明确批准后才能动）。

## Related bugs (out of scope unless user approves)

1. **Env var 名称 docs vs code 不一致**：`internal/config/config.go:359-360` 读 `JWT_SM2_PRIVATE_KEY` / `JWT_SM2_PUBLIC_KEY`,但 `configs/config.prod.example.yaml:86`、`configs/config.sqlite.example.yaml:60`、`configs/config.example.yaml:78-79`、`docs/deployment/secret-management.md:46-47` 都写的是 `XINGRAN_JWT_SM2_PRIVATE_KEY` / `XINGRAN_JWT_SM2_PUBLIC_KEY`。按 docs 设 env var 不会被代码读。
2. **`scripts/deploy/setup-server.sh:24-32`** 的 `/etc/xingran/secrets.env` 模板不包含 SM2 密钥项(JWT_ACCESS_SECRET/JWT_REFRESH_SECRET/SM4_KEY 三项),与 sqlite 模板的 `use_sm2: false` 默认值耦合,使得"按官方 setup 走"与"启用 SM2"互斥。
3. **`scripts/crypto/gen_sm2_keys/main.go` header 注释路径过时**：line 2 写 `go run scripts/gen-sm2-keys.go`,实际文件已搬到 `scripts/crypto/gen_sm2_keys/main.go`。同一文件还保留着陈旧的 `//go:build ignore` 标签 — 仍可用 `go run` 显式路径运行,但 IDE / 静态分析可能误判。
4. **`getPublicKey` handler 缺日志**：`response.ErrServerError` 只能让前端看到 "SM2 未启用或公钥不可用",不暴露 `useSM2`/`sm2PublicKey` 实际值,排查只能 ssh 上服务器 grep 日志。

按 CLAUDE.md scope constrainment ("fix only this issue, don't touch other files") 与用户授权原则,以上不在本次修复范围,等用户确认主线问题解决后再询问是否一起处理。