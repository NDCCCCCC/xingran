---
slug: login-captcha-no-slider-modal
status: resolved
trigger: 登录时提示"请输入验证码"，但没有弹出图形滑动验证码的模态框。请调查原因，不要修改代码。
goal: find_and_fix
created: 2026-06-17T11:00:00.000Z
updated: 2026-06-17T12:00:00.000Z
---

# Debug Session: 登录滑动验证码模态框未弹出

## Context

- 用户登录时看到提示"请输入验证码"，但期望弹出的"图形滑动验证码"模态框没有出现。
- 用户明确要求：**只调查原因，不修改代码**（diagnose-only）。
- 这是配置/状态层面的问题，不是代码崩溃类 bug。

## Symptoms

- **Expected**: 启用滑动验证码时，点登录应弹出 `CaptchaModal` 滑动拼图模态框，验证通过后再提交登录。
- **Actual**: 点登录后只出现 `message.warning("请输入验证码")` 提示，无模态框弹出。
- **Reproduction**: 进入登录页 → 填用户名密码 → 点"登录"按钮。

## 根因 (Root Cause)

**这是配置不匹配，不是代码 bug。** 验证码类型由数据库 `sys_config` 表中
`config_key = 'sys.account.captchaEnabled'` 的 `config_value` 决定，取值三态：
`disabled` / `normal` / `slider`（`pkg/captcha/base.go:9-11`，与前端
`CaptchaEnabled` 类型 `types/captcha.ts:7` 完全一致）。

**当前该配置值不是 `"slider"`，所以前端走了非 slider 分支：**

`xingran-react-frontend/src/pages/login/index.tsx:76-91` 的分支逻辑：

```js
if (captchaEnabled !== "disabled") {
  if (captchaEnabled === "slider") {
    // slider：弹出模态框（line 80）
    setPendingLoginData(loginData);
    setCaptchaModalVisible(true);   // ← 只有这条分支会弹模态框
    return;
  } else {
    // normal 或其他：表单内输入（line 82-90）
    if (!captchaValue) {
      message.warning("请输入验证码");  // ← 用户看到的提示来自这里
      return;
    }
    ...
  }
}
```

- "请输入验证码"提示**只在 `else`（非 slider）分支**出现（login/index.tsx:85）。
- 滑动验证码模态框**只在 `captchaEnabled === "slider"`** 时弹出（login/index.tsx:77-81）。
- 用户既然看到了"请输入验证码"，证明点击登录时 `captchaEnabled` 走到了 else 分支，
  即**当前配置值不是 `"slider"`**（最可能是 `"normal"`，或拼写错误的字符串）。

### 两个候选根因（运行时需用下方 SQL 确认）

- **R1（最可能）**: `sys.account.captchaEnabled = 'normal'`。系统当前是"数字字母验证码"模式。
  此时表单内会渲染 `<TextCaptcha>` 输入框（login/index.tsx:200-208），用户需输入验证码。
  未输入就点登录 → 提示"请输入验证码"。模态框本就不该弹（行为正确）。

- **R2（高度符合现象，需重点排除）**: `sys.account.captchaEnabled` 的值**不是精确的
  `"slider"`/`"normal"`/`"disabled"` 之一**（例如 `Slider`、`SLIDER`、`slide`、`true`、`1`、
  `滑动`、含首尾空格等）。后端 `core/captcha.go:91-97` 用 `captcha.CaptchaType(val)` 直接
  转换、**不校验取值合法性**，于是 `config.Enabled` 成为一个"四不像"字符串。前端则：
  - `captchaEnabled === "slider"` → false（不弹模态框）
  - `captchaEnabled === "normal"` → false（login/index.tsx:200 不渲染 TextCaptcha 输入框）
  - `captchaEnabled !== "disabled"` → true（进入 else 分支）
  - 结果：**既不弹模态框，也不显示验证码输入框，只提示"请输入验证码"，用户无处输入**
  ——这与用户"没有弹出模态框"的描述高度吻合。

## Evidence

- timestamp: 2026-06-17T11:05:00.000Z
  checked: `xingran-react-frontend/src/pages/login/index.tsx`
  found: |
    - 验证码状态来自 `getCaptchaConfig()`（line 33-44），存入 `captchaEnabled`（line 22，初值 "disabled"）
    - slider 分支 line 77-81 调 `setCaptchaModalVisible(true)` 弹模态框
    - else 分支 line 84-86 在 `!captchaValue` 时 `message.warning("请输入验证码")`
    - TextCaptcha 输入框仅在 `captchaEnabled === "normal"` 时渲染（line 200）
  implication: 用户看到该提示 ⇒ 当前 `captchaEnabled` 走了 else 分支 ⇒ 配置值非 "slider"

- timestamp: 2026-06-17T11:08:00.000Z
  checked: `internal/api/v1/captcha_handler.go:105-116` getCaptchaConfig
  found: 直接返回 `string(config.Enabled)`，来源 `core.CaptchaService.GetConfig()`
  implication: 前端 `captchaEnabled` 直接等于后端 `config.Enabled` 的字符串值

- timestamp: 2026-06-17T11:10:00.000Z
  checked: `internal/core/captcha.go:80-225` LoadConfig
  found: |
    - key = "sys.account.captchaEnabled"，从 sys_config 表 Pluck config_value（line 88, 211-214）
    - 默认值 captcha.CaptchaTypeDisabled = "disabled"（line 90）
    - 解析 `*p = captcha.CaptchaType(val)`（line 91-97）不校验合法性，原样赋值
    - 若该 key 不存在或值为空，回退默认 "disabled"
  implication: 运行时取值完全取决于 sys_config 表里这条记录的实际值

- timestamp: 2026-06-17T11:12:00.000Z
  checked: `pkg/captcha/base.go:6-11` + `xingran-react-frontend/src/types/captcha.ts:7`
  found: 后端 "disabled"/"normal"/"slider" 与前端 CaptchaEnabled 三态**字符串完全一致**
  implication: 排除"前后端字符串不匹配"；问题只能是运行时配置值本身

- timestamp: 2026-06-17T11:15:00.000Z
  checked: 全仓库 `sys.account.captchaEnabled` 的写入/seed 点
  found: |
    - 仓库内仅两处出现：读取点 `core/captcha.go:88`、守卫列表 `config_handler.go:43`
    - 无任何 .sql seed / migration / Go init 写入该 key 的默认值
    - captcha_background.sql 只 seed 了 BackgroundMode/PieceShape/Difficulty 等，不含 Enabled
  implication: 该配置项由**用户手动**通过参数管理页面增删改；初始数据库中很可能根本没有这条记录（→ disabled），或用户手动填了 normal/错误值

- timestamp: 2026-06-17T11:18:00.000Z
  checked: `internal/api/v1/system/config_handler.go:40-60, 228-274`
  found: |
    - isCaptchaConfig() 守卫列表包含 sys.account.captchaEnabled（line 43）
    - Update 成功后若 key 属验证码相关，调 `captchaService.LoadConfig()` 热重载（line 262-268）
  implication: 通过"参数管理"页面改验证码类型是即时生效的（无需重启）。若用户是直接改库而非走页面，则需手动调 `/system/auth/captcha/config/reload` 或重启才生效

## 验证方法（不改代码，由用户执行）

1. **查实际配置值（最直接）**:
   ```sql
   SELECT config_key, config_value, status, deleted_at
   FROM sys_config
   WHERE config_key = 'sys.account.captchaEnabled';
   ```
   - 若无记录 / 值为空 / 已软删除 → 后端回退 "disabled"，理论上不该提示"请输入验证码"
     （与现象矛盾，需进一步排查是否有缓存或中间件改写）
   - 若值 = `'normal'` → 命中 R1，系统就是数字验证码模式
   - 若值非三态精确字符串（如 `'Slider'`、`'true'`、`'1'`） → 命中 R2，前端陷入死状态

2. **查后端实际返回（无需登录）**: 调用 `POST /api/v1/system/auth/captcha/config`，
   看 `data.enabled` 字段返回什么。这就是前端 `captchaEnabled` 的真实来源。

3. **前端确认**: 登录页 F12 → Network → 找 `captcha/config` 请求 → 看 response 的 `enabled`。

## 可能的修复方向（本次未应用，遵循"不改代码"要求）

若确认要切到滑动验证码：
- **首选（配置层，零代码改动）**: 系统管理 → 参数设置，新增/修改
  `sys.account.captchaEnabled` = `slider`（小写精确）。页面保存后自动热重载，刷新登录页即生效。
- 若是直改库：改完调一次 `POST /system/auth/captcha/config/reload` 或重启后端。
- 若 R2（拼写错误）：把值改为精确的 `slider` / `normal` / `disabled` 之一即可。
- （代码健壮性建议，超出本次 scope）: 后端 `LoadConfig` 解析 captchaEnabled 时可加白名单校验，
  非法值回退 disabled 并告警；前端 `login/index.tsx` 可对未知 captchaEnabled 值做兜底，避免
  "既不弹框也无输入框"的死状态。

## Eliminated

- hypothesis: 前后端验证码类型字符串不一致导致分支错位
  evidence: `pkg/captcha/base.go` 与 `types/captcha.ts` 三态字符串完全一致
  timestamp: 2026-06-17T11:12:00.000Z

- hypothesis: 前端 useEffect 异步加载配置，用户在配置加载完成前点登录导致状态为初始 "disabled"
  evidence: 若为 "disabled" 会直接 performLogin，不会出现"请输入验证码"提示；现象与该分支不符
  timestamp: 2026-06-17T11:20:00.000Z

- hypothesis: CaptchaModal 组件本身渲染/visible 失败
  evidence: 若走到 slider 分支，`setCaptchaModalVisible(true)` 必然触发；现象是根本没走 slider 分支
  timestamp: 2026-06-17T11:21:00.000Z

## Resolution (用户确认 + 修复 — 2026-06-17)

- **最终根因**: 命中 **R2**。`sys_config.sys.account.captchaEnabled` 的值被某次修改改成了
  **`off`** —— 既非 `disabled`、也非 `normal`、也非 `slider`。
- **为何表现如此**: `off !== "disabled"` ⇒ 进入 `login/index.tsx:76` 外层 if；
  `off === "slider"` 为 false（不弹模态框）、`off === "normal"` 为 false
  （`login/index.tsx:200` 不渲染 TextCaptcha 输入框）；最终落在 else 分支，
  因 `captchaValue` 恒空 → `message.warning("请输入验证码")`，用户**无处输入**。
- **处置**: 用户已将值恢复为合法三态，系统恢复正常。
- **纵深防御修复（用户选择方案：纵深防御）**:
  1. **后端写入校验（根治）** — `internal/api/v1/system/config_handler.go`:
     - 加 `validateCaptchaConfigValue()` 方法，聚焦 captchaEnabled 三态白名单
       （disabled/normal/slider），非法值返回 400，阻止写入
     - Create / Update handler 在调 service 前调用该校验
     - 利用现有 `isCaptchaConfig` 守卫上下文，校验集中
  2. **后端读取兜底（防线）** — `internal/core/captcha.go` LoadConfig:
     - 对 captchaEnabled 加白名单 switch，非法值回退 `normal` + `applogger.Warnf` 告警
     - 防御写入校验拦不住的路径（直改库、历史脏数据）
  3. **前端 login 兜底** — **已回退**（后端读取兜底后冗余）
- root_cause: sys_config.sys.account.captchaEnabled 被写入非法值 `off`，后端 LoadConfig
  不校验合法性、原样接受，导致前端登录页三个验证码分支全部落空。
- verification: `go build ./...` 零错误通过；新增 diagnostics 均为预存风格建议，与本次改动无关。
- files_changed:
  - `internal/core/captcha.go`（LoadConfig 加白名单 + 告警）
  - `internal/api/v1/system/config_handler.go`（加 validateCaptchaConfigValue + Create/Update 调用）

## 遗留建议（超出本次 scope，未执行）

- 后端 `core/captcha.go:91-97` 解析 captchaEnabled 时建议加白名单校验，非法值回退
  `disabled` 并告警，避免静默接受任意字符串。
- 前端 `login/index.tsx` 对未知 captchaEnabled 值建议做兜底（如按 disabled 处理），
  避免"既不弹框也无输入框"的死状态。
- 排查"某次修改"把值写成 `off` 的来源（疑似某处把布尔 false 序列化为字符串 "off"，
  或某次手动/脚本误写），防止再次发生。
