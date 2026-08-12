---
slug: login-idle-captcha-stale
status: resolved
trigger: 当在登录页停留很久之后，不知道什么过期，登录时输入账号密码点击登录提示错误：请求参数错误。无法弹出图形验证码滑块页面，必须刷新浏览器才能正常登录，请排查原因，优化代码，使用更友好的提示或者自动刷新，提供最佳解决方案。
goal: find_and_fix
created: 2026-07-24T00:00:00+08:00
updated: 2026-07-24T09:55:00+08:00
tdd_mode: true
---

# Debug Session: 登录页长时间停留后验证码流程失效

## Symptoms

- **Expected:** 登录页无论停留多久，点击登录都应使用服务端当前安全配置并进入正确验证码流程；临时配置刷新失败时应显示可操作提示。
- **Actual:** 页面长时间停留后点击登录返回“请求参数错误”，滑动验证码不弹出，必须刷新浏览器。
- **Reproduction:** 打开登录页 → 长时间停留 → 输入账号密码 → 点击登录 → 400 且不弹滑块 → 刷新浏览器后恢复。

## Root Cause

登录页面生命周期内有三份可能陈旧的前端状态：

1. `src/lib/api.ts` 的模块变量 `ENABLE_REQUEST_ENCRYPTION` 只在应用启动时初始化；后端配置变化后，前端可能继续按旧值包装或不包装请求体。
2. `src/utils/sm2.ts` 的 `cachedPublicKeyHex` 原本永久缓存；本地未配置固定密钥时后端重启会轮换 SM2 密钥。
3. `src/pages/login/index.tsx` 的 `captchaEnabled` 只在 mount 时读取一次；后端验证码类型变化后仍按旧值决定 slider/normal/disabled。

典型 400 链路：前端按陈旧 `ENABLE_REQUEST_ENCRYPTION=true` 发送加密 envelope，而后端已按新值关闭解密，中间件透传 envelope；`internal/api/v1/auth.go` 的 `ShouldBindJSON(LoginRequest)` 找不到 `username/password`，返回字面量“请求参数错误”。浏览器刷新会重跑全部初始化，所以表面上恢复。

## Fix

### 1. 登录提交前安全预检

新增 `xingran-react-frontend/src/lib/loginPreflight.ts`：

- 每次点击登录时并发刷新请求加密开关、SM2 公钥、验证码配置。
- 三项互不依赖：captcha 路径命中 `/system/auth/captcha` 加密黑名单；public-key 是 GET 且也在黑名单，不受旧加密开关影响。
- 单项与总体等待上限均为 5 秒。
- 任一失败立即阻止登录并显示：`登录安全配置已过期，自动更新失败，请检查网络后重试`。
- 不整页刷新，不丢失用户已填写的用户名和密码。

### 2. 使用最新验证码类型

`src/pages/login/index.tsx` 的 `handleFinish` 不再使用 mount 时的旧 `captchaEnabled`，而是使用本次 preflight 返回值决定：

- `slider` → 弹出滑块；
- `normal` → 显示/校验文本验证码；
- `disabled` → 直接登录。

按钮进入 loading 后会阻止快速重复提交；回归测试锁定只启动一次 preflight。

### 3. 验证码类型变化局部恢复

- `CaptchaModal` 新增 `onError`，将 slider 内的 `CAPTCHA_TYPE_MISMATCH` 上报父页面。
- 父页面局部刷新安全配置、关闭旧 modal、清理旧 captcha/pending 数据并提示重新验证。
- 删除原 `window.location.reload()` 恢复方式。

### 4. 防止迟到公钥覆盖

`src/utils/sm2.ts` 新增 `publicKeyCacheGeneration`：

- `clearPublicKeyCache()` 与 `fetchPublicKey(true)` 都推进代次；
- 请求返回时只有代次仍为最新才能写入缓存；
- 较早请求即使迟到，也不能覆盖较新公钥。

### 5. 配置刷新结果显式化

`refreshEncryptionConfig()` 从 `Promise<void>` 改为 `Promise<boolean>`：

- 成功应用新值返回 `true`；
- 网络、超时、非成功响应返回 `false`，保持旧值；
- 原调用方只 `await` 且忽略结果，保持兼容。

## TDD Evidence

- RED：`loginPreflight` 模块不存在时测试无法解析；实现后 GREEN。
- RED：`refreshEncryptionConfig resolve(false)` 被旧实现误判成功；修正 boolean 契约后 GREEN。
- RED：旧 SM2 请求迟到会把 `NEW_PUBLIC_KEY` 覆盖为 `OLD_PUBLIC_KEY`；generation guard 后 GREEN。
- RED：slider mismatch 未上报父页面，局部刷新提示不存在；增加 `CaptchaModal.onError` 后 GREEN。

最终目标测试：

```text
3 test files passed
14 tests passed
```

覆盖：三项并发、最新 captcha 类型、false/reject/timeout、友好提示、SM2 强制刷新、迟到响应、页面阻止登录、slider modal、TextCaptcha/SliderCaptcha mismatch、快速重复提交。

## Runtime Verification

通过 Chrome DevTools 驱动真实 `http://localhost:4000/login`：

1. 正常点击登录后 Network 同时出现并成功：
   - `GET /system/auth/encryption-config` 200
   - `GET /system/auth/public-key` 200
   - `POST /system/auth/captcha/config` 200
2. 页面随后正常弹出“安全验证”滑动验证码 modal；用户完成验证后 `/system/auth/login` 返回 200，并进入 dashboard。
3. 隔离浏览器上下文中切换为 Offline 后点击登录：三项 preflight 均 `ERR_INTERNET_DISCONNECTED`，页面保留账号密码并显示友好 Alert，没有发起错误 login 请求，也没有 reload。
4. 恢复网络后不刷新页面再次点击：三项 preflight 均 200，滑块 modal 自动恢复弹出。

运行时截图：`C:\Users\CPIC\AppData\Local\Temp\login-preflight-runtime.png`。

## Verification

- `npx vitest run src/utils/sm2.test.ts src/lib/__tests__/loginPreflight.test.ts src/pages/login/index.test.tsx --pool=threads --maxWorkers=1` → 3 files / 14 tests passed。
- `npm run type-check` → 通过。
- `npm run build` → `✓ built in 37.01s`。
- `go build ./...` → 通过（后端未改动）。
- 相关新增代码 ESLint 无新增错误；现有 `CaptchaModal` effect 与登录页旧 `any` 仍有历史 lint 项，未扩大修复范围。
- 独立代码复核 → 无 Critical/Important 问题。

## Files Changed

- `xingran-react-frontend/src/lib/api.ts`
- `xingran-react-frontend/src/lib/loginPreflight.ts`
- `xingran-react-frontend/src/lib/__tests__/loginPreflight.test.ts`
- `xingran-react-frontend/src/pages/login/index.tsx`
- `xingran-react-frontend/src/pages/login/index.test.tsx`
- `xingran-react-frontend/src/components/captcha/CaptchaModal.tsx`
- `xingran-react-frontend/src/utils/sm2.ts`
- `xingran-react-frontend/src/utils/sm2.test.ts`

## Commit

未提交；项目约定必须先获得用户明确确认。
