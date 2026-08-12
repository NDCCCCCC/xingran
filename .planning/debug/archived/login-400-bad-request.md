---
slug: login-400-bad-request
status: resolved
trigger: POST /api/v1/system/auth/login 返回 400 错误（前后端加密 enabled 不同步）
created: 2026-06-16
updated: 2026-06-25
session_type: bug
---

# Debug: Login 400 Bad Request - 请求参数错误

**Slug:** login-400-bad-request
**Created:** 2026-06-16 10:56
**Status:** investigating
**Reporter:** Console browser log
**Target endpoint:** `POST /api/v1/system/auth/login`
**Backend URL:** `http://10.62.10.33:9000`

## Symptom

前端登录页提交后，浏览器控制台报：

```
POST http://10.62.10.33:9000/api/v1/system/auth/login 400 (Bad Request)
10:54:59.791 index.tsx:59 登录失败: Error: 请求参数错误
    at handleHttpResponseError (errorHandler.ts:221:17)
    at api.ts:454:21
    at async Axios.request (axios.js:2219:14)
    at async login (authStore.ts:76:23)
    at async performLogin (index.tsx:50:7)
    at async handleFinish (index.tsx:94:5)
```

## Stack Trace (call site)

```
Login form (index.tsx:172 <Form>)
  → onFinish (index.tsx:94 handleFinish)
    → performLogin (index.tsx:50)
      → authStore.login (authStore.ts:76)
        → api.post (api.ts:473)
          → handleHttpResponseError (api.ts:454 → errorHandler.ts:221)
            → throw Error("请求参数错误")
```

## Initial Hypotheses (待 debugger 验证)

1. **入参缺失/类型错误** — username/password/captcha 字段未传或格式不符
2. **SM2+SM4 加密失败** — 该项目登录属于 `security.request_encryption.exclude_paths` 排除项？若未排除，加密失败会致 400
3. **Request body 已被消费** — 加密中间件已读取 body 但未恢复，导致 handler 看到空 body
4. **Gin binding tag 不匹配** — 后端 LoginRequest struct 的 json/binding tag 与前端 payload key 不一致
5. **请求体被 `ShouldBindJSON` 解析但前端未做该 JSON 序列化** — 例如前端发 FormData 而后端期望 JSON
6. **前端 api.ts:473 实际发的不是 login 路径** — 路径拼接错

## Investigation Notes (debugger filled)

### 完整请求链路

1. **前端 Login 表单 (src/pages/login/index.tsx:69-94)** —
   `handleFinish({username, password, captcha?, captchaId?})` → `performLogin(loginData)` →
   `authStore.login(credentials)`。

2. **authStore.login (src/store/authStore.ts:60-78)** —
   - `getEncryptedLoginRequest(username, password)` 用 SM2 公钥加密 password，**返回 `{ username, password: <SM2密文base64>, encryptedPassword: true }`**。
   - 构造 `loginRequest = { username, password, encryptedPassword, captcha, captchaId }`。
   - 调用 `post('/system/auth/login', loginRequest)`。

3. **api.ts 请求拦截器 (src/lib/api.ts:200-267)** —
   - URL `/system/auth/login` **不在 ENCRYPTION_BLACKLIST** 中（黑名单只列了 `public-key`/`captcha`/`encryption-config`/`upload`）。
   - `ENABLE_REQUEST_ENCRYPTION` 通过 `initEncryptionConfig` 从后端 `/system/auth/encryption-config` 读取 — 该端点 `GetEncryptionConfigFromCache` 在缓存未初始化时**默认返回 `true`**（fail-secure）。`initEncryptionConfig` 三次重试都失败时也会回退到 `true`。
   - **结论**: 实际发出去的 body 是加密后的 `{ encrypted: true, data, sm4Key, iv, timestamp, nonce }`，**不是**原始 LoginRequest。

4. **后端中间件 (pkg/middleware/request_decryption.go)** —
   - `cmd/main.go:200` 启动时调 `api.SetupRouter(apiRouter, coreModule, ...)` → `setupEncryptionMiddlewares` 读 `core.Config.Security.RequestEncryption.Enabled = true` → 挂载 `r.Use(middleware.RequestDecryption(...))` 和 `ResponseEncryption(...)`。
   - `configs/config.yaml:82-95` 显示 `request_encryption.enabled: true`，**`/api/v1/system/auth/login` 不在 exclude_paths**（注释"登录接口已移除 - 启用请求体加密 Phase 18"）。
   - 所以后端**会**走 RequestDecryption → 读取 body → 检查 `encrypted: true` → 调 `DecryptRequestWithKeyInfo` 解密 → 替换 body → `c.Next()`。

5. **后端 handler (internal/api/v1/auth.go:89-95)** —
   - `c.ShouldBindJSON(&req)` 把解密后的 body 反序列化为 `LoginRequest`。
   - 如果 `Username` 或 `Password` 缺失或为 `""`，binding:`required` 失败 → 400 "请求参数错误"。

### 关键证据

| # | 位置 | 观察 |
|---|---|---|
| E1 | `configs/config.yaml:84-89` | 后端 `request_encryption.enabled: true` 且登录接口**不在** `exclude_paths`（注释明确"登录接口已移除"） |
| E2 | `xingran-react-frontend/src/lib/api.ts:57-62` | 前端 `ENCRYPTION_BLACKLIST` **不包含** `/system/auth/login` |
| E3 | `xingran-react-frontend/src/lib/api.ts:179-197` | `shouldEncryptRequest` 在白名单空时默认 `return true` |
| E4 | `xingran-react-frontend/src/lib/api.ts:109-133` | `initEncryptionConfig` 重试全失败时设 `ENABLE_REQUEST_ENCRYPTION = true` (fail-secure) |
| E5 | `pkg/middleware/request_decryption.go:280-289` | `GetEncryptionConfigFromCache` 在缓存未初始化时返回 `true` (默认启用) |
| E6 | `internal/api/v1/auth.go:19-28` | `LoginRequest` 要求 `username`+`password` 都 `binding:"required"` |
| E7 | `xingran-react-frontend/src/store/authStore.ts:60-78` | 实际发出的 payload 字段名 `username`/`password`/`encryptedPassword`/`captcha`/`captchaId` 与后端 json tag 对齐 |
| E8 | `internal/api/v1/auth.go:92-95` | 唯一会返回 400 "请求参数错误" 的地方是 `c.ShouldBindJSON(&req)` 失败 |
| E9 | `pkg/middleware/request_decryption.go:160-172` | 中间件其他 400 路径返回 "解密失败"/"解密数据格式无效"/"加密请求格式错误"，**不是** "请求参数错误" |
| E10 | `cmd/main.go:125-131` | `core.New(cfg)` 失败会 `Fatalf` 退出；SM2 密钥对加载失败时进程不会启动 |
| E11 | `internal/core/db/migrations/migration_086_request_encryption_toggle.go:30` | 迁移默认写入 `sys.request.encryption.enabled = "true"` |

### 假设评估

| 假设 | 评估 | 证据 |
|---|---|---|
| H1: payload 字段名/类型与 binding 不匹配 | **不成立** | E7: 字段名/类型完全对齐 |
| H2: 加密未把 login 加进 exclude_paths，body 解密失败 | **不成立** (后端确实启用) | E1: 后端配置 exclude_paths 注释已移除 login |
| H3: body 被消费但未恢复 | **不成立** | request_decryption.go:181 显式恢复 body |
| H4: 路由 method/路径错位 | **不成立** | `POST /api/v1/system/auth/login` 已注册 |
| H5: 缺少 captcha 字段 | **不成立** | captcha 字段非 binding required，缺失不触发 |
| H6: 密码字段名不一致 | **不成立** | E7: 前端发 `password` 字段，后端 binding 是 `password` |
| H7: **中间件**未启用（SM2 密钥对未加载） | **不可能** | E10: 加载失败进程会退出 |
| **H8**: **前后端 enabled 不同步** | **最可能** | 见下方分析 |

### 关键诊断: H8 (前后端 enabled 状态不同步)

**最可能的根因路径**：

1. **前端 fail-secure 默认 true** (E4+E5): 应用启动时 `initEncryptionConfig()` 三次重试 **都可能失败**（如后端未就绪 / 网络问题 / 早期浏览器缓存），回退到 `ENABLE_REQUEST_ENCRYPTION = true`。即使首次请求成功（`enabled = true`），后续**用户在系统配置界面把后端 `sys.request.encryption.enabled` 改成 `false` 关闭加密**——但**前端全局变量 `ENABLE_REQUEST_ENCRYPTION` 不会自动更新**（因为 `initEncryptionConfig` 只在应用启动时调一次）。

2. **后端 enabled = false** (E1): 用户在配置管理界面关闭加密后，`RefreshEncryptionConfigCache()` 立即使中间件缓存失效，下一次请求 `getConfigFromDB` 读 DB 返回 `false` → `RequestDecryption` 中间件**绕过**（第 59-62 行 `if !enabled { c.Next(); return }`）。

3. **结果**: 前端**仍然加密 body 发**（`ENABLE_REQUEST_ENCRYPTION = true`），后端**不再解密**（enabled = false），handler 收到的是 `{ encrypted: true, data, sm4Key, iv, ... }` — **没有任何字段对应 `LoginRequest.Username`/`Password`**，binding `required` 失败 → **400 "请求参数错误"**。

4. **证据指向**:
   - 后端 `response.go:39` 定义 `ErrBadRequest.Message = "请求参数错误"`，前端 `errorHandler.ts:205` 取 `responseData.message` → 用户看到的就是 "请求参数错误" ✓
   - `request_decryption.go:111-118` 当 body 不是 JSON 或 `encrypted` 字段不存在时**直接透传**到 handler — 与症状一致
   - 该 bug 与 resolved/`request-encryption-config-delay.md` 描述的"前端不会自动重新读取"现象**完全同源**

### 备选诊断 (低概率但需排除)

- **后端 SM4 key 长度异常**: `request_encryption.go:259-261` 在 sm4Key 长度不是 16 字节时报 "密钥长度无效"。但前端 `generateSM4Key` 必返回 32 字符 hex，理论不会触发。
- **时间戳超出 60s 窗口**: 用户报 400 时刻若在 60s 内不会触发。

## Resolution

**root_cause:** ✅ **确证** — H8 (前后端 enabled 不同步)。

**运行时证据 (2026-06-16 11:16)**:
- DB: `SELECT config_value FROM sys_config WHERE config_key='sys.request.encryption.enabled'` → `false`
- 用户通过配置管理界面 (`/system/config`) 修改的
- 重启前端 dev server 后登录成功
- 根因：5 月修复 `if (values.configKey === 'sys.request.encryption.enabled')` 检查失败（表单的 configKey 字段可编辑 + Vite HMR 可能重置模块），导致 `refreshEncryptionConfig` **未真正被调用**，前端 `ENABLE_REQUEST_ENCRYPTION` 保留旧值 `true`，与后端 `false` 失配

**fix:** ✅ 应用方案 B (2026-06-16 11:30)
- 文件: `xingran-react-frontend/src/pages/system/config/index.tsx:142-153`
- 改动: 去掉 `if (values.configKey === 'sys.request.encryption.enabled')` 条件判断
- 任何配置更新（包括非加密配置）后都会调用 `refreshEncryptionConfig()`，确保前端 `ENABLE_REQUEST_ENCRYPTION` 始终与后端 DB 同步
- 代价: 每次保存参数多 1 次 `/system/auth/encryption-config` 网络调用 (~5ms)
- 收益: 防御所有"用户改表单字段 / HMR 重置 / 后续新增加密相关配置项"的场景

**verification:**
- ✅ TypeScript 类型检查通过 (`npm run type-check`)
- ⚠️ `npm run build` 报 3 个错误，全部位于 `src/pages/system/apikeys/LogsModal.tsx:344`（unterminated string literal — 上一笔 commit `80d4a3a` 引入的预先存在 bug），**与本次修改无关**
- 待用户在浏览器重现流程验证

**specialist_hint:** typescript
**status:** resolved - fix applied, awaiting user verification + commit approval

---

## 等待用户提供的运行时证据

请在浏览器登录页复现一次 400 错误，然后提供下列 4 项数据：

### 1. 后端数据库当前配置

```bash
psql -h <DB_HOST> -U postgres -d xingran -c \
  "SELECT config_key, config_value, update_by, updated_at FROM sys_config WHERE config_key='sys.request.encryption.enabled';"
```

（DB_HOST 默认 10.62.10.33 或 localhost，按你本地 .env 为准）

### 2. 后端 GET /system/auth/encryption-config 响应

```bash
curl -s http://10.62.10.33:9000/api/v1/system/auth/encryption-config
```

期望格式:
```json
{ "code": 0, "data": { "enabled": <true|false>, "source": "cache" | "database" }, ... }
```

### 3. 浏览器 DevTools Network 面板 — 失败请求 body

打开 F12 → Network → 触发一次登录失败 → 点击 `/system/auth/login` 请求 → 看 "Request Payload":

- **情况 A**: 含 `encrypted: true`、`data`、`sm4Key`、`iv` 字段 → 前端在加密（H8 命中候选）
- **情况 B**: 含 `username`/`password`/`encryptedPassword`/`captcha`/`captchaId` 字段 → 前端没加密，根因在别处
- **情况 C**: body 为空或 415 错误 → 请求根本没到 handler

把 Request Payload 的实际 JSON 复制贴回来。

### 4. 后端日志（同一时刻）

```bash
# 找最近 1 分钟的日志，按关键字过滤
tail -n 200 <LOG_FILE> | grep -E "auth/login|request_decryption|解密|encryption.config|request.encryption"
```

关键字含义:
- 出现 `请求解密成功` → 后端解密成功，问题在解密后 body
- 出现 `解密失败` / `解密数据格式无效` / `加密请求格式错误` → 解密失败（SM2 密钥对不匹配/时间戳超窗等）
- 出现 `刷新加密配置缓存` → 缓存被刷新过
- 完全无加密中间件相关日志 → H8 强命中（中间件被 `enabled=false` 旁路）

### 期望收敛路径

| 证据组合 | 根因 | 修复 |
|---|---|---|
| DB=`false` + /encryption-config=`enabled:false` + Request body 是加密 envelope | **H8 命中** | 前端需在收到 `enabled:false` 后重读配置（api.ts） |
| DB=`true` + /encryption-config=`enabled:true` + Request body 是明文 | **H4 路由错位/反序列化错** | 检查 api.ts 黑名单 + axios content-type |
| Request body 是明文 + 后端日志 "解密失败" | **SM2 密钥对不一致** | 重启后端重生成密钥对 |
| Request body 是加密 envelope + 后端无加密中间件日志 + DB=`false` | **H8 命中** | 同上 |

提供以上 4 项数据后，我会在 5 分钟内给出 100% 锁定的根因 + 最小修复方案。
