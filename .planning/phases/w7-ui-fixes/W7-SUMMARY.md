---
phase: W7（v1.56 里程碑内 UI 修复波次）
created: 2026-11-26
status: code-complete · real-device UAT partially deferred
前置: 4ab44b20（LIVEVIEW L1 真机验证）+ 用户真实浏览器反馈
---

# W7 SUMMARY —— 真实浏览器暴露的 UI 级缺陷修复

## 0. 一句话结论

三个方面全部修复并在**用户视角**下验收通过（浏览器操作 / 渲染断言 / 真 Chromium 出帧），
但**生产后端未重新部署**（工作树有另一会话未提交的 branding 改动，见 §7 越界与协作），
因此缺陷 1、2 的真机浏览器复核标为「待部署验证」，其证据为「真实浏览器实测复现 + 页面级集成测复现/修复后转绿」。

| 缺陷 | 结论 | 证据类型 |
|---|---|---|
| 1 wizard「返回任务列表」跳 dashboard | **PASS** | 真实浏览器复现根因 + 落地路由断言（渲染→点击→断言 `/vdi/rpa`） |
| 2 接管后遮罩不消失 | **PASS** | 真实浏览器双向复现（mask=1）+ 页面级集成测 5 例 |
| 3 实时画面一直「等待画面显示中」 | **PASS** | 生产 worker 日志实证 124 条 deadline + 真 Chromium「闲置 90s 再开流」出帧 |
| 4 管理员账号显示「陈超」 | **N/A**（非缺陷） | 换账号所致，见 §5 |

---

## 1. 缺陷 1 —— wizard 三处跳转到不存在的路由

### 1.1 根因实证（不是推理，是查了生产 PG）

`xingran-react-frontend/src/pages/operations/rpa/wizard/index.tsx` 三处写死：

```tsx
navigate("/operations/rpa/tasks");   // 保存后 / 返回按钮 / 取消
```

该路径**不匹配任何路由**。RPA 页面不由前端静态注册，而是 sys_menu seed 播种后由
`RouteGenerator.resolvePath` 逐级拼接。查生产 PG（10.62.10.34，`sys_menu`）：

| id | menu_name | parent_id | path | component |
|---|---|---|---|---|
| `770e8400-e29b-41d4-a716-446655440001` | 虚拟机 | (null) | `vdi` | `/operations/room-devices/index` |
| `af1451a4-650c-4ed1-a2b7-b58769dc3920` | RPA 管理 | 770e8400 | `rpa` | `pages/operations/rpa/index` |
| `a1b2c3d4-0255-4000-8000-000000000255` | RPA 实时画面 | af1451a4 | `live` | `pages/operations/rpa/live/index` |
| `a1b2c3d4-0256-4000-8000-000000000256` | 新建自动化任务 | af1451a4 | `wizard` | `pages/operations/rpa/wizard/index` |
| `a1b2c3d4-0258-4000-8000-000000000258` | 定时调度 | af1451a4 | `schedules` | `pages/operations/rpa/schedules/index` |

⇒ 真实路由 = **`/vdi/rpa`**（父级 `vdi` + `rpa`）。写死 `/operations/rpa/tasks` 时 SPA 落到
catch-all，把用户甩到 dashboard（**没有任何报错、没有任何接口异常**，这是假 PASS 的典型形态）。

讽刺点确认：`executions/ExecutionDetailModal.tsx:328` 早有注释写明「真实路由是 `vdi/rpa/live`，
不是 `operations/rpa/live`」——W6 知道事实，但 wizard 里仍然写错，说明**知识没有沉淀成常量**。

### 1.2 修复

新增唯一事实来源 `src/pages/operations/rpa/routes.ts`：

```ts
const RPA_BASE = "/vdi/rpa";
export const RPA_ROUTES = {
  taskList: RPA_BASE,
  live: `${RPA_BASE}/live`,
  wizard: `${RPA_BASE}/wizard`,
  schedules: `${RPA_BASE}/schedules`,
} as const;
export function withBase(path: string): string;      // 子路径部署兼容
export function rpaLiveUrl(executionId: string): string;  // 带 ?executionId=
```

替换点（4 处硬编码 → 常量）：
- `wizard/index.tsx` 三处 `navigate(RPA_ROUTES.taskList)`
- `executions/ExecutionDetailModal.tsx` 整页跳转改 `window.location.assign(rpaLiveUrl(execution.id))`

核查结论（归属协议要求的「同步检查」）：`tasks/index.tsx` 的向导入口是**弹窗**（`setWizardOpen(true)`），
不做跳转，因此无需改；`executions/index.tsx`、`workers/index.tsx` 也不含 RPA 跳转硬编码。

### 1.3 用户视角验收证据

**A. 真实浏览器复现（修复前，生产 v1.56.4）**

- 入口：`http://10.62.10.39:9000/vdi/rpa/wizard`（菜单「RPA 管理 → 新建自动化任务」）
- 操作：点右上角「返回任务列表」
- 结果：地址栏变为 `/dashboard`，页面显示仪表盘 —— 与用户反馈一致
- 附证（非判定依据）：Network 无失败请求，Console 无报错

**B. 落地路由断言（修复后，渲染→点击→断言）**

新增 `wizard/__tests__/wizardRoute.w7.test.tsx`（2 例，PASS）：

```
✓ 点「返回任务列表」→ 落到 /vdi/rpa（不是 dashboard，也不是 obsolete 路径）
✓ 向导内的「取消」也落到 /vdi/rpa（同一个常量，不出现第二种口径）
```

断言方式是渲染 `TaskWizardPage` + 真实 `MemoryRouter`，`fireEvent.click` 按钮，
再断言 `[data-testid="task-list-page"]` 出现、`dashboard` 与 `wrong-path` 均不存在、
location probe 内容 = `/vdi/rpa`。

**C. 回归拦截（防再次散落硬编码）**

新增 `__tests__/routes.test.ts`（8 例，PASS）：常量值对齐 PG 实证；扫描源码禁止出现
`"/operations/rpa"` / `"/vdi/rpa"` 裸字符串（先剥注释，保留「历史错误路径」这类说明文字）。

---

## 2. 缺陷 2 —— 接管成功后遮罩不消失

### 2.1 根因实测（任务书要求「必须先实测确认根因再改」，本条的实证过程）

**先说结论**：根因是 **(c) 两边取值口径不一致**，且有一个更底层的放大因素 ——
**SM2 分支的 JWT 里根本没有 nickname 字段**，导致后端 operatorName 恒回落 username。

#### 实证 1：生产 JWT payload 里没有 nickname

用真实浏览器登录 `ninedrunk` 后，在页面内解出 access token 中间段：

```json
{"user_id":"7fd49999-af45-42b5-a936-fc4fe64cb2ad",
 "username":"ninedrunk",
 "roles":["aec7fa7b-...","7d099ccc-...","e797633b-..."],
 "iss":"XingRan-Next","sub":"...","exp":...,"nbf":...,"iat":...,"jti":"..."}
```

**没有 `nickname` 键**。而代码链是：

```
pkg/crypto/sm2_jwt.go  Claims struct {UserID, Username, Roles, Iss, Sub, Aud}   ← 无 Nickname
internal/core/security/jwt.go  SM2 分支构造 crypto.Claims{...}                ← 不传 Nickname
pkg/middleware/auth.go:103     c.Set("nickname", claims.Nickname)              ← 恒空
internal/api/v1/rpa/liveview_handler.go operatorName()                         ← 恒回落 username
```

HS256 分支的 `CustomClaims` **一直有** `Nickname` —— 两个分支字段不一致才是根因。
排查清单里的 (a) State 未带 snapshot / (b) api 解包层数 / (d) 未推送状态，经实测均**不成立**：
- (a) 不成立：`liveview_hub.go:811 State()` 读的是 `s.context`，而 `Takeover` 同时写
  `s.takeoverBy` 与 `s.context.TakeoverBy`（`hub.go:467/470`），是同一条链。
- (b) 不成立：`lib/api.ts:391` 拦截器在 `data.code === 0` 时 **return data**（即 BaseResponse 外壳），
  故 `res.data.context` 正好是内层 data —— 层数正确。
- (d) 不成立：接管后 `refreshState()` 会重新 GET state 并 `setContext`，且 Hub 也在推 context。

#### 实证 2：真实浏览器双向复现（决定性）

生产环境（v1.56.4，`ninedrunk` 账号，`nickname == username == ninedrunk`）：

| 步骤 | 观察结果 |
|---|---|
| 进入 `/vdi/rpa/live?executionId=1483491e-...` | 遮罩 `[data-testid=live-view-mask]` 存在，角标「自动执行」 |
| 点「接管」→ 确认弹窗「接管」 | 遮罩**消失**（mask 0），角标「已接管 · ninedrunk」 |
| **关键对照**：把 localStorage 里 `auth-storage` 的 `user.nickname` 改成「超级管理员」→ 刷新页面 | 遮罩**重新出现**（mask 1），角标「自动执行」 |
| 再点「接管」→ 确认 | 角标变「已接管 · ninedrunk」，但**遮罩仍然存在**（mask **1**） |

第三条是这个缺陷的**精确复现**：前端 `myName = "超级管理员"`，后端写入 `takeoverBy = "ninedrunk"`
（因为 token 无 nickname → operatorName 回落 username），`takeoverBy === myName` 恒 false → 遮罩不消失。
`ninedrunk` 这个账号之所以「碰巧能用」，纯粹因为 nickname 恰好等于 username。

换成 `admin`（nickname=超级管理员 / username=admin）必然不匹配 —— 这解释了用户为什么会说
「管理员账号显示陈超」这类现象（见 §5）。

### 2.2 修复（三层，缺一层就会复发）

**第 1 层：让 token 携带 nickname（消灭取值口径差）**
- `pkg/crypto/sm2_jwt.go`：`Claims` 补 `Nickname string \`json:"nickname,omitempty"\``
- `internal/core/security/jwt.go`：SM2 分支构造 claims 时补 `Nickname: accessClaims.Nickname`

**第 2 层：控制权判据改为 userID 权威（消灭「判据是展示名」这个设计错误）**
- `liveview_hub.go`：`liveViewSession` 新增 `takeoverByID`；`LiveViewContextSnapshot` 新增
  `takeoverById`；`LiveViewWatchState` 新增 `takeoverById`
- `Takeover` 冲突判定优先比 userID（旧会话无 userID 时才回落展示名）
- `Release` / `Input` 签名改为 `(ctx, execID, userID, userName, ...)`，判据一律 userID；
  展示名只用于错误文案与 HUD
- `IngestFrame` / `IngestContext`：帧里的 context 用**后端权威值**回写（不再直接转发 Worker 的
  `up.Context`），保证帧 context 与 state 端点永远同步

**第 3 层：前端同口径 + 纯函数化**
- 新增 `live/controller.ts`：`resolveIsController()`（主判据 takeoverById，兜底展示名）、
  `controllerDisplayName()`、`isTakenOver()`
- `live/index.tsx`：`isController` 改为调用纯函数，并新增 `myUserId` 取 `user.id`
- `live/LiveHud.tsx`：角标「已接管」以 `isTakenOver` 为准，展示名走兜底（避免「已接管 · 」半截文案）
- `live/types.ts` + `liveFrameParser.ts`：新增 `takeoverById` 字段透传

### 2.3 用户视角验收证据

**后端契约（真实 DB，非 mock）** —— 新增 `internal/api/v1/rpa/liveview_w7_test.go`（5 例 PASS）：

```
TestUI02_TakeoverByUserID_NotDisplayName   同 userID 换展示名仍是同一接管者；不同 userID 即使展示名相同也冲突
TestUI02_StateExposesTakeoverByID          state / watch / 帧 context 三处都下发 takeoverById，且帧回传被权威值覆盖
TestUI02_ReleaseUsesUserID                 同 userID 换展示名释放成功，幂等
TestUI02_InputUsesUserID                   同 userID 换展示名输入不被 403；不同 userID 必须 403
TestUI02_ForceOverridesOtherUser           force 抢占语义不变
```

**JWT 字段链路** —— 新增 `pkg/crypto/sm2_jwt_nickname_test.go`（3 例 PASS）：
nickname 往返、payload 里真实存在该字段、空昵称走 omitempty 且反序列化等价空串。

**页面级集成（用户视角，不 mock 内部函数）** —— 新增
`live/__tests__/liveTakeoverMask.w7.test.tsx`（5 例 PASS）：

```
✓ 人格 A（昵称==账号名）：接管 → 遮罩消失 + 已接管角标 + 出现释放按钮
✓ 人格 B（昵称≠账号名，缺陷原形）：接管 → 遮罩同样消失
✓ 他人接管：遮罩保持，且没有释放按钮（说明判定不是恒真）
✓ 人格 B 接管后：收到帧 → 画布上移动鼠标 → /liveview/input 被调用（payload 校验 executionId/kind/x/y）
✓ 反向：非接管者即使收到帧也不下发任何 input（遮罩拦在源头）
```

其中「人格 B」直接对应生产复现路径（store nickname=超级管理员 / username=ninedrunk，
后端 takeoverBy=ninedrunk / takeoverById=uid-b2）。第 4 条把「鼠标输入能到达远端」这件事
断在**请求发出**这一层：帧到达后 `deviceRef` 有尺寸 → 坐标换算成立 →
`POST /rpa/executions/{id}/liveview/input` 被调用且坐标 = `(640, 320)`
（`clientX=200` 在 400×225 的显示矩形内、1280×720 device 下换算所得）。

**纯函数层** —— `live/__tests__/controller.test.ts`（13 例 PASS），含「修前/修后」对照断言：

```
expect(oldLogic).toBe(false)   // 修前：takeoverBy("ninedrunk") !== myName("超级管理员")
expect(resolveIsController({...})).toBe(true)  // 修后：userID 相等
```

**真机（生产浏览器）证据**：修复前 mask=1 复现；修复后的真机复核待部署（见 §7）。

---

## 3. 缺陷 3 —— 实时画面一直「等待画面显示中」，帧从未到达

### 3.1 根因

`rpa-worker/internal/browser/chrome_page.go:101`（修复前）：

```go
defaultTimeout := cfg.Timeout                       // 30s
if cfg.NavigationTimeout > defaultTimeout { ... }   // 60s
page = page.Timeout(defaultTimeout)                 // ← 一次性 deadline
```

rod@v0.116.2 `context.go:71` 的实现：

```go
func (p *Page) Timeout(d time.Duration) *Page {
    ctx, cancel := context.WithTimeout(p.ctx, d)   // 从此刻起倒计时
    return p.Context(context.WithValue(ctx, timeoutContextKey{}, &timeoutContextVal{p.ctx, cancel}))
}
```

这个 ctx 挂在 page 句柄上**永不重置**。Manager 把 page 放池里跨执行复用 → 闲置超过 60s 后
ctx 已过期 → 之后任何 `proto.XXX.Call(page)` 立即 `context deadline exceeded`。

**生产实证（10.62.10.34 `rpa-worker`，2026-11-25）**：

| 证据 | 数值 |
|---|---|
| 容器日志里 `context deadline exceeded` 总数 | **124** 条 |
| `Page.startScreencast 失败` 分类 | `context deadline exceeded` **6** 条 + `context canceled` **3** 条 |
| 同源症状（执行侧） | `task execution failed: 等待 body 元素超时: context deadline exceeded`、`导航失败: context deadline exceeded` |
| 时间线 | worker 08:40:31 启动 → 08:51 才有第一个 watch（闲置 11 分钟） |

**为什么单元测试测不出来（同构性缺口）**：155 的所有 screencast 用例都是
`newLivePage()` 里 `Acquire` 后**立刻** `StartScreencast`，ctx 还在有效期内 → fps 正常。
这是「测试用例的生命周期与生产不一致」的典型，也是本轮要补的关键用例。

### 3.2 修复（方案 A + B 组合，取舍见 §3.4）

**A（根治）：创建时不再设全局 Timeout**
- `newChromePageOn()` 删除 `page = page.Timeout(defaultTimeout)`（只保留 SetViewport）
- 同时清理三处**同类隐患**（它们都是「把一次性 deadline 挂到长期复用的句柄上」）：
  - `Reset` 兜底新建空白页：`fresh.Timeout(...)` → `fresh`
  - `EnterFrame`：`frame.Timeout(...)` → `frame`（frame 会长期挂在 `p.frame` 上供后续动作使用）
- 补齐原本依赖全局 Timeout 的操作（删掉全局后必须自己限定）：
  `Screenshot()`、`Evaluate()`（含 IIFE 重试分支）、`scrollBy()`、`dragByJS()`、`PressKeys()`（KeyActions）
  全部改为 `p.cur().Timeout(p.config.Timeout)...`

**B（防御）：screencast 用干净 ctx 直调 CDP**
- `screencast.go`：新增 `cleanPage := page.Context(context.Background())`，
  `Page.setWebLifecycleState` / `Page.startScreencast` 以及 `Screencast.page`（供
  `screencastFrameAck` / `stopScreencast` 使用）全部改走 `cleanPage`；
  事件订阅仍用 `bgPage`（带可取消 ctx，供 Stop 优雅退出）

### 3.3 用户视角验收证据（真 Chromium，非模拟）

新增 `rpa-worker/internal/browser/idle_screencast_w7_test.go`（4 例全 PASS）：

```
--- PASS: TestUI03_PageCtxHasNoDeadline (2.60s)
    page ctx 类型=*context.cancelCtx，无 deadline，err=nil（符合 W7 / UI-FIX-03 口径）
--- PASS: TestUI03_PoisonedPageDefenseLayer (3.09s)
    阴性对照成立：挂 Timeout(1s) 的 page 在 1.2s 后 ctx 已过期（context deadline exceeded）
    裸 CDP 直调如预期失败：context deadline exceeded
    防御层成立：中毒 page 上仍收到帧 seq=1 480x360
--- PASS: TestUI03_IdlePageStillStreams (92.14s)
    让 pool 内 page 闲置 1m30s 后再开流（模拟生产 08:40 启动 / 08:51 首次 watch）
    闲置 1m30s 后首帧到达：seq=1 480x360 2263 bytes
--- PASS: TestUI03_IdleReusedPageRepeatedRounds (7.18s)
    第 1 轮出帧正常（seq=1,2）
    第 2 轮出帧正常（seq=1,2）
    第 3 轮出帧正常（seq=1,2）
```

四个用例的分工（这是「能拦住这类问题」的机制，不是一次性验证）：

| 用例 | 拦什么 | 耗时 |
|---|---|---|
| `PageCtxHasNoDeadline` | 有人在创建期重新挂 Timeout（**根因级**，CI 秒级可跑） | 2.6s |
| `PoisonedPageDefenseLayer` | 阴性对照（证明真是 deadline 致死）+ 防御层（中毒 page 仍出帧） | 3.1s |
| `IdlePageStillStreams` | 生产时间量级的闲置 90s 后开流（**用户视角**：打开实时画面时 page 可能已闲置十几分钟） | 92s |
| `IdleReusedPageRepeatedRounds` | 闲置后连续多轮 watch/stop 每轮都出帧（防「只在第一轮好」） | 7s |

### 3.4 取舍说明（任务书要求）

选择 **A+B 组合**，理由：

| 方案 | 优点 | 缺点 | 采纳 |
|---|---|---|---|
| A：删全局 Timeout，每操作链式 | 根治；page 生命周期与「一次操作」解耦；对所有复用场景（不止 screencast）同时生效 | 需要逐个检查所有操作点是否都带超时（本轮已全量核查） | ✅ |
| B：screencast 用干净 ctx | 成本极低；即使将来有人在别处又挂上 deadline 也不会让「唯一闲置后才触发」的 screencast 死亡 | 单独用只是绕过症状，不解决执行侧同样会中的问题 | ✅ 作为防御层 |
| 仅 B | —— | 执行侧（导航/等待 body）仍会因闲置 page 死亡；症状会转移不消失 | ❌ |
| 仅 A | 根治 | 无额外保险；未来回归时会直接打穿 screencast | ⚠️ 单用风险偏高 |

---

## 4. 契约影响评估：对 154 ENGINE 动作执行

任务书指定 `page.Timeout()` 删除是**核心契约变更**，必须写清影响。

### 4.1 结论

**无功能回归，且修复了一个既有的隐性缺陷。**

### 4.2 依据一：所有动作都已在调用点链式设超时（代码核查）

`chrome_page.go` 里每一个执行动作的入口都是 `p.cur().Timeout(p.config.Timeout)...`：

| 动作 | 超时来源 | 命中位置 |
|---|---|---|
| Goto / Navigate | `NavigationTimeout`（并另设 `Timeout` 给 body 元素） | `navPage := p.page.Timeout(NavigationTimeout)` + `navPage.Timeout(Timeout).Element("body")` |
| Click / ClickWith / Hover / Fill / Upload / Select / WaitFor / Drag / ScrollInElement | `p.config.Timeout` | 15 处 `p.cur().Timeout(...)` |
| Screenshot / Evaluate / scrollBy / dragByJS / PressKeys | `p.config.Timeout` | **本轮新增**（原依赖全局 Timeout） |
| SwitchTab / EnterFrame | `NavigationTimeout` / `Timeout` | `target.Timeout(...).WaitLoad()` / `p.page.Timeout(...).Element()` |

⇒ 删掉创建期全局 Timeout 后，每个操作的**有效超时窗口不变**（仍是 30s/60s），
只是不再有一个「在创建时刻就开始跑、且永不重置」的额外 deadline 压在 page 上。

### 4.3 依据二：全量测试证明

| 命令 | 结果 |
|---|---|
| `cd rpa-worker && go build ./...` | ✅ 无输出（成功） |
| `cd rpa-worker && go test ./...`（`-count=1 -timeout 2400s`） | **除 `internal/browser` 一次 flaky 外全绿**；`internal/executor` 17.1s、`internal/liveview` 4.0s、`internal/recording` 7.9s、`internal/worker` 2.3s、`internal/communication` 2.5s、`internal/config` 2.0s、`deployments` 1.0s |
| `cd rpa-worker && go test ./internal/browser/`（重跑） | ✅ **213.6s PASS** |

**关于唯一一次 flaky 的说明（必须如实记录）**：首轮全量跑时
`TestRecorder_RealChromium_LoginFillSubmit` 在 `recorder_integration_test.go:267` 失败：
`回放录制结果失败: 第 2 步填写失败（选择器 "[data-testid=\"login-username\"]"）: context deadline exceeded`。
- 单跑该用例：**PASS（5.47s）**
- 随后整包重跑：**PASS（213.6s）**
判定为**时序 flaky**（该用例在 ~30 个真 Chromium 用例并发跑时抢不到资源），
不是本次改动引入的回归 —— 理由：失败点错误文案与修复前的生产症状同源，
但该用例的 page 是 `Acquire` 后立刻使用（ctx 未过期），且修改后单跑与整包重跑均通过。
本轮未擅自调整该用例的时序参数（属 156 记录侧资产），只登记为已知 flaky。

### 4.4 依据三：修复了既有的隐性缺陷

生产日志里同类 `deadline exceeded` 不止 screencast：
`task execution failed: 导航失败: context deadline exceeded`、
`等待 body 元素超时: context deadline exceeded`、
`task execution cancelled: step 1/11, error: context deadline exceeded`。
这些都指向同一个根因（池内闲置 page 的 ctx 过期）。删除创建期全局 Timeout 后，
这些执行侧失败同时被消除 —— 属于**顺带修复**，不是新增风险。

---

## 5. 缺陷 4 —— 管理员账号显示「陈超」：**N/A，非缺陷**

用户换账号登录所致。PG 实证（`sys_user`）：

```
c_chenchao-044 | 陈超     | 0
chenchao-104   | 陈超     | 0
c_chenchao-051 | 陈超     | 0
d_chenchao     | 陈超2    | 0
chenchao-076   | 陈超1    | 0
ninedrunk      | ninedrunk | 0
admin          | 超级管理员 | 0
```

「陈超」是一组真实存在的账号昵称，页面显示的是**当前登录账号的昵称**，符合设计。

**但必须记一笔（任务书要求）**：上一轮的「角色验证结论」
（`.planning/notes/261125-deployment-checklist.md` §5「ninedrunk 重登实测菜单可见/列表真读/新建落库/编辑落库/删除软删，四项 CRUD 全过」）
**只对 `ninedrunk` 这一账号成立**；
用管理员/超级管理员账号看到全权限，**不构成授权验证**（超管恒有全部权限，无法证伪权限模型）。
后续 UAT 必须使用**非超管、且 nickname ≠ username 的普通账号**（本清单 §0.1 已写成硬前置）。

---

## 6. 交付物清单

### 6.1 代码改动（14 个修改 + 7 个新增）

**后端（6）**
| 文件 | 内容 |
|---|---|
| `pkg/crypto/sm2_jwt.go` | `Claims` 补 `Nickname`（omitempty） |
| `internal/core/security/jwt.go` | SM2 分支把 nickname 传进 token |
| `internal/services/rpa/liveview_hub.go` | 新增 `takeoverByID`（会话/context/watch state 三处）；Takeover/Release/Input 判据改 userID；帧 context 用权威值回写 |
| `internal/api/v1/rpa/liveview_handler.go` | Release/Input 调用点补 userID 参数 |
| `internal/api/v1/rpa/liveview_p155_test.go` | 适配新签名 + 补「接管者本人输入必须放行」断言 |
| `internal/api/v1/rpa/liveview_w7_test.go`（新） | 5 个 userID 权威契约用例 |

**worker（3）**
| 文件 | 内容 |
|---|---|
| `rpa-worker/internal/browser/chrome_page.go` | 删除创建期全局 Timeout（含 Reset 兜底新建页、EnterFrame 同类隐患）；补齐 5 处链式 Timeout |
| `rpa-worker/internal/browser/screencast.go` | 新增 cleanPage（background ctx）直调 CDP（防御层） |
| `rpa-worker/internal/browser/idle_screencast_w7_test.go`（新） | 4 个闲置出帧/根因守护用例 |

**前端（5 修改 + 4 新增）**
| 文件 | 内容 |
|---|---|
| `pages/operations/rpa/routes.ts`（新） | `RPA_ROUTES` + `withBase` + `rpaLiveUrl` |
| `pages/operations/rpa/wizard/index.tsx` | 3 处硬编码 → `RPA_ROUTES.taskList` |
| `pages/operations/rpa/executions/ExecutionDetailModal.tsx` | 整页跳转 → `rpaLiveUrl()` |
| `pages/operations/rpa/live/controller.ts`（新） | `resolveIsController` / `controllerDisplayName` / `isTakenOver` |
| `pages/operations/rpa/live/index.tsx` | `myUserId` + isController 走纯函数 |
| `pages/operations/rpa/live/LiveHud.tsx` | 角标以 `isTakenOver` 为准 |
| `pages/operations/rpa/live/types.ts` / `liveFrameParser.ts` | 透传 `takeoverById` |
| 新增测试 4 个 | `routes.test.ts`、`controller.test.ts`、`wizardRoute.w7.test.tsx`、`liveTakeoverMask.w7.test.tsx` |

### 6.2 文档（2）
- `.planning/phases/w7-ui-fixes/W7-SUMMARY.md`（本文件）
- `.planning/notes/261125-ui-uat-checklist.md`（可复用 UI 走查清单）

---

## 7. 闸门结果（全部实际执行，含红项说明）

| # | 命令 | 结果 | 备注 |
|---|---|---|---|
| 1 | `go build ./...`（仓库根） | ✅ PASS | 无输出 |
| 2 | `go test ./internal/...` | ⚠️ **1 红，非本轮** | `internal/services/system [build failed]`：`branding_service_test.go:109` 的 `*brandCacheStub` 缺 `GetStats` 方法。该文件 **CreationTime 2026-11-26 10:38**、未提交（`??`），属**另一会话并行改动的半成品**；其余全部 `ok`（含 `internal/services/rpa` 5.2s、`internal/api/v1/rpa`、`internal/utils/operlog`） |
| 3 | `cd rpa-worker && go build ./...` | ✅ PASS | 交叉编译 `out/rpa-worker-linux-w7` 一次通过 |
| 4 | `cd rpa-worker && go test ./...` | ✅ PASS（重跑） | `internal/browser` 213.6s PASS；详见 §4.3 flaky 说明 |
| 5 | `npx tsc --noEmit` | ✅ PASS | 无 `error TS` |
| 6 | rpa 相关 vitest | ✅ PASS **81 例** | routes 8 + controller 13 + wizardRoute 2 + liveTakeoverMask 5 + live 其余 44 + ExecutionDetailModal liveEntry 2 + wizard 既有 … |
| 7 | W7 目标包 Go 测试 | ✅ PASS | `pkg/crypto` 3.1s、`internal/core/security` 51.6s、`internal/services/rpa` 0.9s、`internal/api/v1/rpa` 88.5s |

### 7.1 生产部署与回滚记录（如实登记）

为拿到**真机**证据，本轮做过一次部署尝试，最终按用户决定回滚：

1. 交叉编译 `out/rpa-worker-linux-w7`（14,094,520 B）→ 上传 `10.62.10.34:/opt/rpa-worker-new/`
2. 备份 + 替换容器挂载点 → 重建容器 → `/health` healthy、
   日志确认 `liveview hub initialized` / `worker registered successfully` / `max_concurrency: 8`
3. **发现工作树有另一会话并行写入**（`branding_service.go` 等未提交文件，
   `App.tsx` / `login` / `api.ts` / `router.go` 亦被改动），构建含这些半成品的后端二进制有风险
4. **用户选择「不部署后端」** → 已把 worker 回滚至 W6 版本：
   `rpa-worker` / `rpa-worker-prev-w7`（保留可回滚容器）、
   `rpa-worker-linux-final` 恢复为 `…final.bak-pre-w7`，
   现存镜像 `rpa-worker-linux-w6bak`（W6 二进制副本）
5. 生产后端 `10.62.10.39:/app/szh/ruoyi-backend-embedded-linux` **未被替换**，
   仅多出一个 `ruoyi-backend-embedded-linux.bak-pre-w7` 备份文件（未做任何覆盖）

⇒ 因此**缺陷 1 / 2 的真机复核状态 = 待部署**（证据为真实浏览器复现 + 页面级集成测），
缺陷 3 的真机证据由独立真 Chromium 用例提供（不依赖生产部署）。

---

## 8. 教训（写进流程，不只是写在文档里）

1. **「单测绿灯」不能证明「生产可用」**：三条 P0 缺陷全部在单测绿灯的情况下流到用户面前。
   原因统一是**测试的生命周期与生产不一致**：
   - 缺陷 3：测试 page 刚创建就用，生产 page 已闲置 11 分钟
   - 缺陷 2：测试账号 nickname 恰好等于 username，生产账号不一定
   - 缺陷 1：测试直接渲染组件（路由由 MemoryRouter 提供），生产靠 sys_menu 推导真实路径
2. **知识要沉淀成常量，不能只写在注释里**：`ExecutionDetailModal.tsx:328` 注释早就写对了真实路由，
   但 wizard 仍写错第三次。本轮用 `routes.ts` + 源码扫描测试把它变成**编译/测试期约束**。
3. **控制权/身份类判据永远用 ID，不用展示名**：展示名会随昵称、随 token 分支、随 i18n 漂移。
   本轮把 Takeover/Release/Input 全部改成 userID 权威，展示名只做文案。
4. **一次性 deadline 不能挂在长期复用的句柄上**：rod 的 `Page.Timeout()` 语义是「从此刻总共 d」，
   不是「每次操作 d」。池化对象上出现这种 ctx 就是定时炸弹。
5. **验收必须写「断言什么变化」**：本轮每个修复都配了「用户动作 → 可观察变化」的断言
   （落地路由 / mask 数量 / fps>0 / input 请求 payload），而不是「接口 200」。

---

## 9. 遗留与下一步

| # | 项 | 状态 | 说明 |
|---|---|---|---|
| L1 | 缺陷 1/2 真机复核 | **待部署** | 后端需在「无并行会话改动」的干净工作树上重建后部署（或 CI 出包），再按 `261125-ui-uat-checklist.md` §2/§3 走一遍 |
| L2 | `internal/services/system` 编译红 | 他人资产 | `branding_service_test.go` 缺 `GetStats`；需该会话补齐，与本轮无关 |
| L3 | `TestRecorder_RealChromium_LoginFillSubmit` flaky | 已知 | 并发跑时偶发；单跑稳定 PASS。若后续 CI 稳定复现，需给该用例加串行/重试 |
| L4 | 生产默认 screencast 档位（nth=2 → 4.66fps） | 155 遗留 | 本次未改默认值（运维口径），建议 `nth=1 + q40 + 854×480`（8.86fps / 0.259Mbps） |
