---
status: resolved
trigger: "请检查为什么每次刷新都会丢失token，回到登录页"
created: 2026-08-19
updated: 2026-08-19
---

# Debug Session: token-lost-on-refresh

## Symptoms

**Expected behavior:**
刷新页面后应保持登录状态（token 从持久化存储恢复，无需重新登录）。

**Actual behavior:**
每次刷新页面都会丢失 token，被重定向回登录页。

**Error messages:**
未观察（用户尚未查看控制台/网络面板）。

**Timeline:**
最近的前端改动后出现（项目刚完成 v1.22 前端品牌化改造，Phase 64-67，含 Phase 66 布局重构 PageTitle/useRouteTabs/sidebar/header）。此前行为未确认。

**Reproduction:**
开发环境 localhost:4000（npm run dev + go run 后端）：登录 → 刷新页面 → 回到登录页。

**Environment:**
- 开发环境 localhost:4000
- 前端: React 19.2 + Vite 7.2 + Zustand 5.0 (authStore 含 TokenManager)
- 认证: JWT 双 token（access + refresh），token 刷新由 authStore 内 TokenManager 管理

## Current Focus

- hypothesis: RESOLVED — 根因确认、修复应用、用户端到端验证通过，会话归档
- next_action: none（已完成；修复已随本会话提交）

reasoning_checkpoint:
  hypothesis: "后端 GenerateRefreshTokenWithSM2 硬编码 Issuer="XingRan-Next"，与配置 jwt.issuer="XingRan-Next-Dev" 不一致；JWTManager.ValidateToken 的签发者校验（jwt.go:204）恒定拒绝所有 SM2 refresh token → 401 → 前端 clearTokens + 跳登录页"
  confirming_evidence:
    - "logs/app.log: 全部 40 个 /system/auth/refresh 请求 100% 返回 401，0 个成功（含 07:42:21 登录成功后 82s 的 07:43:43 刷新失败）"
    - "pkg/crypto/sm2_jwt.go:289 硬编码 Issuer: \"XingRan-Next\"；configs/config.yaml:60 jwt.issuer: \"XingRan-Next-Dev\"；jwt.go:204 claims.Issuer != j.issuer → response.ErrTokenInvalid (HTTP 401, response.go:45)"
    - "后端自 2026-08-18 19:25:01 起未重启（排除 SM2 密钥轮换）；/system/auth/refresh 在公开路由组（router.go:107-110，排除 auth 中间件 401）；日志显示请求解密成功（排除 SM2/SM4 加解密问题）"
  falsification_test: "单元测试：NewJWTManager(use_sm2=true, issuer=\"XingRan-Next-Dev\") → GenerateTokenPair → ValidateToken(refreshToken)。修复前若 ValidateToken 成功则假设被推翻"
  fix_rationale: "把 issuer 与过期时长作为参数从 JWTManager 配置传入 GenerateRefreshTokenWithSM2（替代两处硬编码），refresh token 与 access token 共用同一配置源，签发者校验恒一致；前端无需改动（其失败处理行为正确）"
  blind_spots: "HS256 路径（use_sm2=false）本就正确不受影响；sm2_jwt.go 自初始提交未改过，说明 SM2 模式下 refresh 从未成功过"

## Evidence

- timestamp: 2026-08-19T00:00Z
  checked: authStore.ts / TokenManager.ts / SecureTokenStorageImpl.ts / DynamicRoutes.tsx / main.tsx / api.ts 全文
  found: 前端恢复链路 = persist rehydrate(user) → onRehydrateStorage → initializeFromStorage → getRefreshToken(sessionStorage 'rt', SM4-CBC) → POST /system/auth/refresh → 成功则 isAuthenticated=true。AccessToken 仅内存。/system/auth/refresh 在 AUTH_WHITELIST（无需 access token）但**不在** ENCRYPTION_BLACKLIST（请求体会被 SM2+SM4 加密，本环境 encryption enabled=true）。refresh 401 → clearTokens + window.location.href=/login。
  implication: 刷新后登录态完全依赖 refresh 调用链成功；任何一环失败都回到登录页。

- timestamp: 2026-08-19T00:00Z
  checked: git log/show — Phase 64-67 提交对 auth 关键文件的影响
  found: 860d255 (Phase 66) 只改 layout/header/sidebar/useRouteTabs/PageTitle/user页/GlobalSearch/index.css，**未触碰** authStore/api.ts/token utils/DynamicRoutes/main.tsx。57bdd51 对 main.tsx 仅移除 1 行 CSS import。auth 关键文件的最近实质改动是 883c941（sm2 解密失败单次重试）与 6e30151（前端审查 P0/P1）。
  implication: 用户归因"Phase 66 后出现"可能不精确；也不能排除环境因素（后端重启等）。index.css 有未提交改动需留意。


- timestamp: 2026-08-19T00:00Z
  checked: logs/app.log — /system/auth/refresh 全量状态码分布 + 登录时间线
  found: 40 个 refresh 请求全部 401，0 个 200。时间线呈"refresh 401 → 数秒~数十秒后用户重新登录 200"循环（07:43:43→07:43:57、07:50:04→07:50:43、07:50:48→07:50:54、07:52:24→07:52:38）。请求解密成功（decrypted≈493B），latency 10-13ms。
  implication: 前端正确发送 refresh 请求；后端 handler 恒定拒绝；症状 100% 确定性复现，与前端改动无关。

- timestamp: 2026-08-19T00:00Z
  checked: pkg/crypto/sm2_jwt.go + internal/core/security/jwt.go + configs/config.yaml + pkg/response/response.go
  found: GenerateRefreshTokenWithSM2 硬编码 Issuer "XingRan-Next"（sm2_jwt.go:289）；config jwt.issuer="XingRan-Next-Dev"（config.yaml:60, use_sm2=true）；ValidateToken 签发者校验 claims.Issuer != j.issuer → ErrTokenInvalid（HTTP 401, response.go:45）。access token 走 GenerateTokenWithSM2 且 Issuer=j.issuer，故登录后 access token 正常。HS256 路径 refreshClaims.Issuer=j.issuer 本就正确。
  implication: 仅 use_sm2=true 时 refresh token 签发即注定 401；这是唯一能同时解释 40/40 失败与"登录成功但刷新页面掉线"的机制。

- timestamp: 2026-08-19T00:00Z
  checked: internal/api/router.go 路由注册 + GenerateRefreshTokenWithSM2 调用点 + git log sm2_jwt.go
  found: /system/auth/refresh 在公开组（无 JWT 中间件）；GenerateRefreshTokenWithSM2 全库唯一调用点是 jwt.go:169；sm2_jwt.go 自初始提交 ea528c6 未改过。
  implication: 排除中间件拦截；修复面收敛到一处调用点；bug 与生俱来，非近期回归。

- timestamp: 2026-08-19T00:00Z
  checked: 用户端到端验证（重启后端加载修复 → 浏览器 localhost:4000 登录 → F5 刷新）
  found: 用户确认"确认修复"——刷新后保持登录态，不再跳转登录页。
  implication: 修复在真实环境验证通过，根因闭环。

## Eliminated

- hypothesis: Phase 66 (860d255) 布局重构破坏了前端 token 恢复
  evidence: git show 860d255 只改 layout/header/sidebar/useRouteTabs/PageTitle/user页/GlobalSearch/index.css，未触碰 authStore/api.ts/token utils/DynamicRoutes/main.tsx；且日志显示 refresh 401 早在 Phase 66 之前就 100% 失败
  timestamp: 2026-08-19T00:00Z

- hypothesis: 后端重启导致 SM2 密钥对轮换，旧 refresh token 验签失败
  evidence: 日志最后一次启动标记 2026-08-18 19:25:01，其后无重启；07:42:21 登录（200）与 07:43:43 refresh 401 发生在同一进程生命周期内，同进程签发不可能验签失败
  timestamp: 2026-08-19T00:00Z

- hypothesis: 前端 SM4 解密 sessionStorage 'rt' 失败 / 未发出 refresh 请求
  evidence: 后端日志有完整请求记录且"请求解密成功"（decrypted_data_size≈493），说明前端成功取出并发送了 refreshToken；401 是 handler 层返回
  timestamp: 2026-08-19T00:00Z

- hypothesis: /system/auth/refresh 被 auth 中间件拦截（缺 access token → 401）
  evidence: router.go:107-110 refresh 注册在"无需认证"的公开组，无 JWT 中间件
  timestamp: 2026-08-19T00:00Z

- hypothesis: refresh token 过期或 roles 不是 ["refresh"]
  evidence: GenerateRefreshTokenWithSM2 正确设置 Roles=["refresh"] 与 7 天有效期；失败发生在 ValidateToken 内部（ErrTokenInvalid 由签发者校验返回），先于 handler 的 roles 检查
  timestamp: 2026-08-19T00:00Z

## Resolution

root_cause: pkg/crypto/sm2_jwt.go GenerateRefreshTokenWithSM2 硬编码 Issuer "XingRan-Next"（及 7 天过期时长），而 JWTManager.ValidateToken 校验 claims.Issuer == 配置的 jwt.issuer（config.yaml: "XingRan-Next-Dev"）。use_sm2=true 时所有 refresh token 签发即与配置不一致 → 每次页面刷新的 token 恢复请求 POST /system/auth/refresh 恒定 401（日志 40/40 全失败，0 成功）→ 前端 api.ts 401 拦截器 clearTokens + 跳转登录页。access token 路径使用 j.issuer 所以登录后短时间可用，仅刷新页面时暴露。自初始提交即存在（sm2_jwt.go 无后续改动），与 Phase 64-67 无关。
fix: GenerateRefreshTokenWithSM2 签名改为 (userID, username, issuer, expiration, privateKey)，issuer 与过期时长由 JWTManager 从配置传入（jwt.go:169 传 j.issuer, j.refreshKeyExpire），消除两处硬编码（"XingRan-Next" / 7天）。refresh token 与 access token 从此共用同一配置源，签发者校验恒一致。前端无需改动。
verification: 单元测试 TestSM2RefreshTokenValidatesWithConfiguredIssuer 精确复现 handler 调用路径（NewJWTManager(use_sm2=true, issuer="XingRan-Next-Dev") → GenerateTokenPair → ValidateToken(refreshToken)）：修复前 RED（ErrTokenInvalid "令牌无效"，与生产 401 同源，response.go:45）；修复后 GREEN。TestSM2RefreshTokenHonorsConfiguredExpiry 验证有效期跟随配置。go build ./... 通过（全库唯一调用点已同步更新）；go vet 两包干净；pkg/crypto 与 internal/core/security 全量测试套件通过。用户端到端验证通过（2026-08-19）：重启后端加载修复 → 重新登录 → F5 刷新保持登录态，不再跳转登录页。
files_changed: pkg/crypto/sm2_jwt.go, internal/core/security/jwt.go, internal/core/security/jwt_refresh_sm2_test.go
