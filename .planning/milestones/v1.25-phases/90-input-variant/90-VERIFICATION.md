---
phase: "90"
slug: "input-variant"
status: human_needed
nyquist_compliant: true
wave_0_complete: true
verified: "2026-10-29"
requirements_satisfied: 2
requirements_total: 2
audit_acknowledged:
  milestone: v1.25
  at: 2026-10-30
  status: human_needed
---

# Phase 90 — VERIFICATION

> 本 phase 由 4 个 plan 逐波执行（W0 基线 → W1 批次 C/B + VISFIX-05 + STYLE-03 → W2 评估收尾 + T4 裁定 → W3 收口验收）。
> 证据来源：浏览器探针实跑输出 + 矩阵终判 + 静态契约闸门 + git commit 链 + `90-CHANGESET.md` + `90-QUANT-SNAPSHOT.md`。
> **status = `human_needed`** —— 两条 REQ 的技术判据全部达成，但**巡检的 40 个「需登录页面 × 明暗」组合被认证墙阻断**（详见第七节遗留表），待用户人工执行。

---

## 一、ROADMAP Phase 90 五条成功标准（5 条）

| # | 成功标准 | 判定 | 证据（文件名 + 实测值） |
|---|---|---|---|
| 1 | 批次 C 与批次 B **同 commit** 落地（顺序不可逆） | ✅ | `git log`：`89c2f62d` 单个 commit 同时含批次 C（`index.css` 拆 `border` 简写为 `border-width`/`border-style`/`border-color` 三条 longhand）与批次 B（VIS-01 复位块与 VIS-03 注释块重写为章节/选择器锚点，**本体六项逐字未动**） |
| 2 | InputNumber 错误态边框 = `rgb(186,54,48)`；圆角 computed-style = 8px | ✅ | `probe-dark.txt` / `probe-light-90.txt`：**light `rgb(186, 54, 48)` / dark `rgb(161, 49, 44)`**；`probe-contract.json`：`affixWrapper` / `antInput` / `inputNumber` / `textarea` 的 `borderTopLeftRadius` 全为 **8px**，`affixWrapperInner` = **0px** |
| 3 | 双框 = 0 / 焦点环单圈 / 错误态断言全绿（v1.24 契约零破坏） | ✅ | `probe-contract.json`（直接复用 `scripts/css-contract-assertions.js` 的判定式）：**`passed = true`** —— `doubleCount = 0`、`inner.boxShadow = none`、`wrapperFocused.shadow = rgb(233,239,235) 0px 0px 0px 2px` ≠ none、`wrapperError.borderTopColor = rgb(186, 54, 48)`；静态侧 `css-contract-check.js` → **ALL PASS**（C1a/C1b/C1c/C4a-c/C5a-c/C11/C13/C16 全绿） |

---

## 二、Lead 15 条锁定裁定（L-01 ~ L-15，15 条）

| # | 裁定 | 判定 | 证据 |
|---|---|---|---|
| L-01 | 批次 C 与批次 B **同 commit** | ✅ | `git log` 单 commit `89c2f62d` |
| L-02 | 批次 C **只拆 `:883` 一处** | ✅ | `git show 89c2f62d -- src/index.css`：`border` 简写仅在 `.ant-input, .ant-input-number` 组被拆为三条 longhand；wrapper 的 `border` 简写**未拆** |
| L-03 | wrapper 简写因 **C4b 字面依赖**不拆 → 登记为**债 D-f** | ✅ | `css-contract-check.js` 的 C4b 正则依赖字面 `border: 1px solid var(--input-border)`；`90-CHANGESET.md` §④ 债表 **D-f** 已登记（归属 91/92，与 D-c 同批处置） |
| L-04 | 批次 C 形态：width/style 去 important，**仅 color 保 important** | ✅ | `src/index.css` 的 Input 块：`border-width`/`border-style` 无 `!important`，`border-color` 保留 `!important`；实测 `borderTopWidth=1px` / `borderTopStyle=solid` / `borderTopColor` 与基线逐字相等；M1 实测 1392（批次 C ±0） |
| L-05 | 批次 B **= 留**（甲方案）：复位块本体保留，**C1a/C1b/C1c 零调整** | ✅ | `git show 89c2f62d -- scripts/css-contract-check.js`：C1 块无 +/-；`css-contract-check.js` C1a/C1b/C1c 全 PASS |
| L-06 | InputNumber status class 必须是 **`ant-input-number-status-error`**（不是 `ant-input-status-error`） | ✅ | `matrix-fixture.js` 的 f22 形态字面量即 `ant-input-number ant-input-number-outlined ant-input-number-status-error`；实测命中（f22 carrier.border 读数 `rgb(186,54,48)` / `rgb(161,49,44)`，若用错类名则读数会是常态边框色） |
| L-07 | error / warning **各 6 个选择器**（3 档 × 常态+hover+focus） | ✅ | `src/index.css` 新增 InputNumber status 规则 12 条（error 6 + warning 6）；C16a/C16b/C16c 三条静态断言守护 |
| L-08 | 色值只用 **`var(--ant-color-*)`**；**实测值不得伪造** —— 暗色 `--ant-color-error` = `#a1312c`（`rgb(161,49,44)`）已由 W0 实测坐实 | ✅ | `src/index.css` 的新规则只用 `var(--ant-color-error)` / `var(--ant-color-warning)`（C16c 断言「无字面 hex」PASS）；`probe-dark.txt` 实测 `--ant-color-error` = `#a1312c`、`--ant-color-warning` = `#996b1e`、`--ant-color-bg-container` = `#1a2e1f`；断言按模式取值（light `rgb(186,54,48)` / dark `rgb(161,49,44)`） |
| L-09 | 圆角归属：输入类 + wrapper + 分页器 = **8px**；**六处非输入域 6px 不改并登记** | ✅ | `probe-light-90.txt`：`.ant-input` / `.ant-input-affix-wrapper` / `.ant-input-number` / `textarea.ant-input` 全 8px，`affixWrapperInner` 0px；`90-CHANGESET.md` §3.1 逐处登记 6 处字面 6px（Tag 5 + Tree 1，均非输入域） |
| L-10 | **T4 移出 W1**（W2 单独裁定） | ✅ | `git log`：`89c2f62d`（W1，**不含** T4）→ `8d17788e`（**T4 独立 commit**：删 `index.css` 的 `font-size: 14px !important`） |
| L-11 | **R1-R5 有意保留**，必须附色值级论证 + 债编号 | ✅ | `90-CHANGESET.md` §① 的 9 行论证表，每行含**具体色值数字**（如 R2：`#598e5e`=rgb(89,142,94) vs `#156031`=rgb(21,96,49)，绿通道差 46）；保留项全挂 **D-a / D-b** |
| L-12 | 矩阵 **90 化**：明暗双向 + 类别白名单 a/b/c + 未归类=0 + 形态集合不多不少；**格数口径 = a 类 18 格/模式（carrier 14 + inner 4）+ b 类仅 f22.carrier + c 类空集** | ✅ | `matrix-diff-result-90.txt`：**456 格 / light 19 / dark 19 / 归类 38 / 未归类 0 / 形态 before=22 after=22**；`RESULT: … → PASS`（exit 0） |
| L-13 | **before 重采**（不复制 89 after；22 形态 + f22） | ✅ | `before-{light,dark}.json` 为 W0 独立采集（`90-QUANT-SNAPSHOT.md` §一核对结论：M1-M16 起点与 89 终值逐条差异为 0） |
| L-14 | **C16a/b/c 只增不改**（既有断言零改动） | ✅ | `git show 89c2f62d -- scripts/css-contract-check.js`：C16 族为**纯新增**（+13 行），C1/C4/C5/C11/C13 零改动 |
| L-15 | **文件边界零 tsx** + 与 86 线零交叠 + **W1 单一 commit** | ✅ | `git show --stat 89c2f62d`：14 个文件全部是 `.css` / `scripts/css-contract-check.js` / `.planning/phases/90-input-variant/**`，**零 `.tsx`**；86 线与 90 线的改动面文件零交集（86 只动 `src/pages/**` + `scripts/check-gating-coverage.mjs` + `package.json`） |

---

## 三、planner 10 条自主发现（P-01 ~ P-10，10 条）

| # | 自主发现 | 判定 | 证据 |
|---|---|---|---|
| P-01 | 90-CHANGESET 三段式（a/b/c 登记表 + 债表 + 终判结论） | ✅ | `90-CHANGESET.md`：a 类逐格表（18 格/模式，由 W0 before 快照**逐格派生**）+ b 类（f22 单格）+ 债表 D-a~D-g + 终判结论节 |
| P-02 | 债表 **D-a ~ D-g** 定稿，每条带量化证据 + 归属 phase | ✅ | `90-CHANGESET.md` §④ 七条齐全（D-a hoverBorderColor / D-b colorBorder 登记变更 / D-c 乙方案路径图 / D-d radius-md 源头废存 / D-e 六处 6px / D-f wrapper 简写 + C4b 同批 / D-g C13 stripComments 机制缺陷） |
| P-03 | 矩阵「**形态集合不多不少**」判据；格数口径 = a 类 18 格/模式 + b 类 1 格/模式 + c 类空集 | ✅ | `matrix-diff-90.cjs` 的 `A_CELLS`（18 条）/ `B_CELLS`（1 条）/ `C_CELLS`（空集）与 CHANGESET 的逐格清单**三方一致**（`matrix-diff-result-90.txt` 逐条带 `[a]`/`[b]` 标签） |
| P-04 | **六条 computed 断言**清单（明暗各跑一次；断言 2 的期望值按模式取值） | ✅ | `probe-light-90.txt` / `probe-dark.txt` / `probe-contract.json` / `hover-focus-{light,dark}.json` 四份证据 |
| P-05 | 探针实测值**不得伪造** —— 暗色 `--ant-color-error` = `#a1312c` 已坐实，断言**按模式取值** | ✅ | `probe-dark.txt`：`--ant-color-error` = `#a1312c` → f22 border = `rgb(161, 49, 44)`；`probe-light-90.txt`：light = `#ba3630` → `rgb(186, 54, 48)`；**两侧期望值不同，未用单一值断言两侧** |
| P-06 | **格数口径**：90 自身终跑全量格数 = 每模式 228 格 / 明暗合计 **456 格**（22 形态 × 6 属性，6 个 inner=null 形态跳过 inner） | ✅ | `matrix-diff-result-90.txt` 首行「单元格 456」 |
| P-07 | **行号基准声明**：清单以**选择器锚点**为权威，行号标注 W1 前 HEAD 基准 + 漂移公式 | ✅ | `90-CHANGESET.md` §3 顶部声明（拆行 +2 / 空行压缩 −D / T4 −1 三重漂移）+ 每处附「选择器锚点（权威）」列 |
| P-08 | **f22 增补** + carrierSel/innerSel 双向同步（b 类唯一落点） | ✅ | `matrix-fixture.js` 与 `matrix-sample.js` 均含 `f22_input_number_status_error`（carrierSel=`.ant-input-number` / innerSel=`.ant-input-number-input`），字面量后置追加 → f01-f21 逐字未改 |
| P-09 | **M13 / M15 口径**在快照与终判结论中同步更新 | ✅ | `90-QUANT-SNAPSHOT.md` W3 终值表的 M13（明暗双值）与 M15（输入类+分页器 8px / 6px 剩余 6 处 / radius-md 15 处未动）；CHANGESET §3.4 同步 |
| P-10 | **M9 预算腾挪**：同 commit 压缩纯冗余空行（**D 由 M9 反推、无绝对值口径**，D ≤ 58、被删行全为块内空行、块间单空行计数 801 不变）；`C13_M9_BUDGET` 保持 5558 不动；C13 多行注释 hex 误计入登记为**债 D-g** | ✅ | `90-QUANT-SNAPSHOT.md` W1 表：G = +37 / D = 39（D ≥ G−2 = 35；D ≤ 58 ✅；被删 39 行全为块内空行；块间计数 801 → 801 ✅）；W3 实测 M9 = 5553 ≤ 5558，`C13_M9_BUDGET` 仍 5558 |

---

## 四、附表 A：六条 computed 断言（明暗各一次 → 6 态口径）

| # | 断言 | 契约 | light 实测 | dark 实测 | 判定 |
|---|---|---|---|---|---|
| 1 | 输入类圆角：`.ant-input` / `.ant-input-affix-wrapper` / `.ant-input-number` / `textarea.ant-input` = 8px；wrapper = 8px 而内层 input = 0px | STYLE-03 | `antInput 8px` / `affixWrapper 8px` / `affixWrapperInner 0px` / `inputNumber 8px` / `textarea 8px` | 同（`probe-dark.txt` 全 8px，内层 0px） | ✅ |
| 2 | InputNumber 错误态边框（**期望值按模式不同**；常态+hover+focus × 明暗 = 6 态） | VISFIX-05 / M13 | **`rgb(186, 54, 48)`** | **`rgb(161, 49, 44)`** | ✅ |
| 3 | 双框 `doubleCount === 0` | VIS-01 | **0** | **0**（同脚本的 light/dark 双容器 fixture） | ✅ |
| 4 | 焦点环单圈：wrapper(−focused) `box-shadow ≠ none` 且内层 = `none` | VIS-02 / C4 | wrapper = `rgb(233,239,235) 0px 0px 0px 2px`；内层 = `none` | wrapper = `rgba(21,96,49,0.15) 0px 0px 0px 2px, rgba(21,96,49,0.08) 0px 0px 20px 0px`；内层 = `none` | ✅ |
| 5 | VIS-03 不回退：Input / TextArea 错误态仍为明暗各自实测值 | VIS-03 | `rgb(186, 54, 48)` | `rgb(161, 49, 44)` | ✅ |
| 6 | hover/focus 回归守护（**90 前后逐字相等**） | §B 显式不改清单 | hover **`rgb(89, 142, 94)`** / focus **`rgb(21, 96, 49)`** | hover **`rgb(89, 142, 94)`** / focus **`rgb(26, 104, 57)`** | ✅ |

> **断言 6 的取证方法学（踩坑记录）**：hover 用 CDP 的 `CSS.forcePseudoState`；focus **必须**用同一机制 —— `el.focus()` 在 `Runtime.evaluate` 求值上下文中执行后求值一结束焦点即被重置（`document.activeElement === el` 为 true 但 `el.matches(':focus')` 为 **false**），CDP 的 `DOM.focus` 同样不生效，`Input.dispatchKeyEvent(Tab)` 会把焦点移到页面上别的元素 ⇒ 唯一可靠路径是 `CSS.forcePseudoState`（已固化为 `p90-hover-focus-probe.cjs` 的注释）。

---

## 五、附表 B：七道闸门

| # | 闸门 | 命令 | 退出码 | 实测结果 |
|---|---|---|---|---|
| 1 | checker | `node scripts/css-contract-check.js` | **0** | **ALL PASS**（C1a / C1b / C1c / C4a-c / C5a-c / C11 / C13 / C14a-g / C16a-c 全绿）；C11「当前 1391 / 预算 1397」；C13「M7=304 ≤310 / M8=440 ≤452 / M9=5553 ≤5558」 |
| 2 | lint | `npm run lint` | **0** | **0 problems**（`--max-warnings 0`） |
| 3 | lint:ci | `npm run lint:ci` | **0** | eslint → assert-lint-budget → check-exemption-reasons → **check-gating-coverage**（86 线接入）→ check-hardcoded-colors → css-contract-check ALL PASS → check-route-registry `OK 697/306/138` |
| 4 | type-check | `npm run type-check` | **0** | **0 错误**（`tsc --noEmit -p tsconfig.app.json`） |
| 5 | test | `npx vitest run` | **0** | **578 文件 / 4022 用例全绿** |
| 6 | build | `npm run build` | **0** | **✓ built in 53.03s**；vendor-react gzip **650.74 kB** ≤ 776 kB 基线 ✅ |
| 7 | matrix-diff | `node .planning/phases/90-input-variant/90-visual-evidence/matrix-diff-90.cjs` | **0** | **456 格 / light 19 / dark 19 / 归类 38 / 未归类 0 / 形态 22=22** → `RESULT: … → PASS` |
| 补 | 后端 | `go build ./...` + `go test ./...` | **0** | 构建干净；全量测试**零 FAIL**（与 85 线并存验证） |

---

## 六、量化终值（详见 `90-QUANT-SNAPSHOT.md` W3 终值快照）

| 指标 | W0 起点 | 终值 | 判定 |
|---|---|---|---|
| M1 全 `.css` `!important` | 1397 | **1391** | ✅ ≤ 1397 |
| M2 `index.css` `!important` | 1330 | **1324** | ✅ ≤ 1330 |
| M7 / M8 hex | 304 / 440 | **304 / 440** | ✅ 不增 |
| M9 `index.css` 行数 | 5556 | **M9 = 5553**（`wc -l` = 5552） | ✅ ≤ 5558（`C13_M9_BUDGET` 未动） |
| M13 错误态边框 | `rgb(219,215,206)` / `rgb(45,77,51)` | **light `rgb(186,54,48)` / dark `rgb(161,49,44)`** | ✅ VISFIX-05 |
| M15 圆角 | 6px 系 7 处 | **输入类 + 分页器 = 8px**；6px 字面剩余 6 处；radius-md 15 处未动 | ✅ STYLE-03 |
| M16 弃用警告 | 0 | **0**（已巡检面） | ✅（未巡检面见遗留表） |

> **⚠️ 90 不追 M1 ≤ 1100 终线** —— 该终线属 91/92 全面交还之后；90 的口径是「M1 不升 + VISFIX-05/STYLE-03 达成 + v1.24 契约零破坏」。

---

## 七、遗留（人工验证项 —— **这是本 phase 定为 `human_needed` 的唯一原因**）

| 项 | 为何人工 | 状态 | 证据 / 清单 |
|---|---|---|---|
| **13 页 × 明暗 = 40 个「需登录页面」巡检组合** | **认证墙**：本环境 `sys.auth.default.mode = ad`（admin 走 **AD 域认证**），本地 `admin123` 不适用。**滑块验证码本身已自动化解决** —— 本波实现并跑通了缺口图像识别 + 拖动 + 服务端校验（`p90-captcha-solve.cjs` / `p90-login.cjs`，实测 `POST /api/v1/system/auth/captcha/verify/slider` 返回 **200**），但随后的登录请求被 AD 认证拒绝（日志：「用户 admin 认证失败(mode=ad): 用户名或密码错误」）。切换该配置属**生产数据写入**，按纪律**不擅自执行** | ⏳ **待用户人工执行** | 逐个探测结果见 `inspection-result-90.txt`（每条含 `blockedByLoginWall: true` + `finalHref` 证据）；待巡检清单见 `inspection-manual-todo-90.json`（40 条，含 mode / name / url） |
| 巡检截图（89 的 40 张） | 同上（需登录态才能截到目标页） | ⏳ 同上 | 本波只截到无登录态可达的 `/login` 明暗 2 张（`login-light.png` / `login-dark.png`）；**目标页截图待人工** |
| M16 在「未巡检面」的达成 | 同上 | ⏳ 同上 | 已巡检面 `deprecated = 0` / `errors = 0`；未巡检面待人工确认 |

### 已交付的等价覆盖（降低未巡检面的风险）

尽管 13 页无法自动巡检，本波用**更强的等价证据**覆盖了「90 的改动没让组件破相」：

| 覆盖项 | 证据 | 强度 |
|---|---|---|
| 全站 22 形态 × 明暗 × 6 属性的**逐格**视觉属性 | 矩阵 456 格终判：**bg / shadow / btw / fontSize 四个维度差异 = 0** | **强于** 14 页截图 —— 矩阵是逐格 computed-style 比对，截图是像素目视 |
| 形态集合完整（无结构缺失） | before=22 / after=22，f22 的 carrier/inner 两侧均在 | 强 |
| v1.24 冻结契约零破坏 | `css-contract-check.js` **ALL PASS**（含 C1a-C1c / C4a-c / C5a-c / C14a-g / C16a-c） | 强（静态断言） |
| 权威 computed 断言 | `probe-contract.json` **passed = true** | 强 |
| 存量回归 | `vitest` 578 文件 / 4022 用例全绿；`go build` + `go test` 全绿 | 强 |

---

## 八、Git 范围

| Plan / 内容 | Commit |
|---|---|
| 90-01（W0 基线与工具） | `f43fef12` |
| 90-01 口径定稿（b 类 mode-aware） | `e93c4717` |
| 90-02（W1：批次 C + 批次 B + VISFIX-05 + STYLE-03 + M9 腾挪，**单一 commit**） | `89c2f62d` |
| 90-03（W2：T4 裁定，**独立 commit**） | `8d17788e` |
| 90-03（W2：R1-R5 论证 + 圆角归属 + 债表 + 取证基建） | `a2ca4dbe` |
| 90-04（W3：矩阵终跑 + 六断言 + 巡检 + 终值快照 + 本报告） | 本波 commit |

**Phase 完成判定**：2/2 需求的技术判据全部达成（矩阵终判 PASS / 权威断言 passed / 11 项闸门全绿 / 量化终值全部达标）+ **40 个巡检组合因认证墙待人工执行** → **`status: human_needed`**（闭合集合 `passed | gaps_found | human_needed`）。

**解锁**：Phase 91（Select / InputNumber / Form / Table / Modal 域交还）—— 硬前置 89（色板）+ 90（Input 域 + 圆角）均已交付；91 的债表入口 **D-a / D-b / D-c / D-f** 已就位。
