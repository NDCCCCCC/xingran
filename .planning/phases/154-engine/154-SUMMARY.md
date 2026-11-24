# Phase 154 · ENGINE 执行引擎（SUMMARY）

> 执行：v1.56 里程碑 W1 波次，Phase 154 executor（与 Phase 152 并行；**全程在独立 worktree `wt-154` 内读写**，未触碰主工作区）。
> 需求映射：ENGINE-01 ~ ENGINE-05。依据：`.planning/REQUIREMENTS.md` :66-70、`.planning/ROADMAP.md` Phase 154 段 + 并行归属协议第 3/5/7 条、`.planning/notes/261123-rpa-worker-capacity-model.md`、`.planning/notes/261123-v156-research-liveview-nocode.md` §3。
> 输出简体中文；代码标识符 / 路径 / 报错原文保持英文。

---

## 0. 结论速览

| 项 | 结论 |
|---|---|
| ENGINE-01 浏览器承载改造 | **PASS**（1 Chrome + N page；N=1→N=8 每 page 增量 ≈46MB，实测数据见 §3） |
| ENGINE-02 动作全集（14 缺口） | **PASS**（14/14 在真 Chromium fixture 页面上通过；枚举↔switch 一致性静态断言已加并做红绿变异验证） |
| ENGINE-03 选择器阶梯与自愈 | **PASS**（7 级阶梯 + 命中层级上报；两个真实页面 fixture 场景通过） |
| ENGINE-04 变量与数据流 | **PASS**（三层作用域 + `${a.b[0].c}` + strict/empty 策略 + extract 嵌套回写 + 敏感脱敏） |
| ENGINE-05 真实浏览器验收基线 | **PASS**（executor/worker/communication 三包首次拥有真实用例；`-short` 保留轻量兜底） |
| `go build ./...` | **PASS**（exit 0） |
| `go build -o /dev/null ./...` | **PASS**（Windows 等价 `go build -o NUL ./...` exit 0，无仓库产物：`*.exe/*.test/*.out` 零命中） |
| `go test ./...`（真 Chromium，冷缓存） | **PASS**（browser 28.7s / communication 2.1s / config 1.4s / executor 8.2s / worker 1.3s） |
| `go test ./... -short`（CI 兜底） | **PASS**（browser 4.0s / executor 0.2s，重型用例正确 skip） |
| 零新增 Go 依赖 | **PASS**（`go.mod` / `go.sum` 一字未改，全部用 go-rod v0.116.2 既有 API） |

---

## 1. ENGINE-01 浏览器承载改造

### 1.1 旧模型 → 新模型

| | 旧（D17 缺陷） | 新（本 phase） |
|---|---|---|
| 进程数 | 每个 page 一个 Chrome（`chrome_page.go:MustLaunch`） | **全 worker 1 个 Chrome**（`Manager.ensureBrowser` 懒启动一次） |
| page 创建 | `browser.MustPage()` 各自新建浏览器后再 `MustPage` | `browser.Page(proto.TargetCreateTarget{})` 在同一 `*rod.Browser` 上开标签页 |
| `Close()` 语义 | `page.Close()` + **`browser.Close()`** | **只 `page.Close()`**（硬语义，见 §4） |
| `max_pages` 语义 | Chrome 进程数上限 | 与 `worker.max_concurrency` 合并为**唯一「并发页面数」** |

### 1.2 改动清单

| 文件 | 改动 |
|---|---|
| `rpa-worker/internal/browser/manager.go`（**新增** 315 行） | `Manager` 持有 1 个 `*rod.Browser` + 1 个 `launcher`；`Acquire/Release/Close/Browser/PageCount/PID/Capacity/SetCapacity`；Chrome **懒启动**（首次 `Acquire` 才拉起，worker 初始化不阻塞）；容量用 `inUse/maxPages/notify` 而非 channel 信号量，**`Acquire` 全程 `ctx` 感知**（旧实现的 `select { case page := <-p.pages }` 在收到 `close(pages)` 后会一直返回 nil，是隐藏死循环）。`Pool` 保留为 `type Pool = Manager` 别名。 |
| `rpa-worker/internal/browser/pool.go`（重写为接口） | `Page` 接口新增 `RawPage() *rod.Page` / `Browser() *rod.Browser` / `Closed() bool` / `Reset()` / `ContextPage() *rod.Page` + ENGINE-02 的 11 个动作原语方法。原 `Pool` 结构体删除（语义已由 Manager 取代）。 |
| `rpa-worker/internal/browser/chrome_page.go`（375→833 行） | `NewChromePageFromBrowser(ctx, b, cfg, log)` 为 Manager 唯一 page 创建入口；`Close()` 移除 `p.browser.Close()`（仅 `ownsBrowser=true` 的独立路径才关 Chrome）；新增 `ownsBrowser` / `frame` / `mainPage` / `baseTargets` 状态与全量动作原语。 |
| `rpa-worker/internal/browser/launcher.go`（**新增**） | `newLauncher(cfg)` 抽公共启动参数，消除「Manager 路径」与「独立路径」的 flags 漂移；顺带修 `--chrome-flag=value` 形态（launcher 不允许 flag 名含 `=`，原来会 panic）。 |
| `rpa-worker/internal/executor/engine.go` | 字段 `browserPool *browser.Pool` → `browserManager *browser.Manager`；`Acquire` 直接拿 `browser.Page`（不再 `pooledPage.Page()`）。 |
| `rpa-worker/internal/worker/worker.go` | `browser.NewPool` → `browser.NewManager`；启动期把 `config.Warnings` 逐条 `logger.Warn`；`scaleUp/scaleDown` 同步 `browserManager.SetCapacity()`（否则扩容只改消费计数、池容量不动 = 空放）。 |
| `rpa-worker/internal/config/config.go` | 合并 schema（见 §1.3）；`Config.Warnings []string`（`mapstructure:"-"`）承载旧键告警。 |
| `rpa-worker/configs/config.yaml` | `max_concurrency` 注释改为「并发页面数 —— 唯一真相」；`max_pages` 注释为「已废弃（保留可读）」。 |

### 1.3 配置合并与**旧键兼容行为**（供 Phase 157 CI 漂移校验读取）

单一真相：`config.EffectiveMaxPages(cfg)` == `cfg.Worker.MaxConcurrency`；`cfg.Browser.MaxConcurrency` 是**派生只读视图**（每次 `Load` 后由 `syncPageConcurrency` 归一化，环境变量覆盖后也会再同步）。

| # | 配置形态 | 行为 | WARN |
|---|---|---|---|
| 1 | 同时给 `worker.max_concurrency` 与 `browser.max_pages` | 以 `max_concurrency` 为准，`max_pages` **不覆盖** | 1 条：「`browser.max_pages=N` 已废弃并合并进 `worker.max_concurrency=M`；本次以 max_concurrency 为准，请删除旧键」 |
| 2 | 只给 `browser.max_pages`（旧部署形态） | 值**迁移**到 `worker.max_concurrency`，旧键字段保留原值可读 | 1 条：「检测到旧键 `browser.max_pages=N`，已合并为 `worker.max_concurrency=N`；请迁移配置（旧键仍可读但不再有独立语义）」 |
| 3 | 两者都不给 | 默认 **5**（不再保留历史的 `worker=3` / `browser=10` 双默认分叉） | 无 |
| 4 | `MAX_CONCURRENCY` 环境变量覆盖 | 覆盖 `worker.max_concurrency` 后再次 `syncPageConcurrency`，视图跟随 | 无 |

- **`BrowserConfig.MaxPages` 字段保留**（可读 + 旧代码可读），语义已废弃；**Phase 157 只改取值、不得改 schema**。
- **Phase 155 新增键**：只加 screencast 相关键，不动上面四条判定。
- 单测守护：`internal/config/config_test.go` 的 `TestMerge_*`（4 种组合）+ `TestEnvOverrideMaxConcurrency` + `TestEffectiveMaxPages`。

### 1.4 验收：N=1 / N=8 的 RSS 基线（真 Chrome，实测）

测试：`internal/browser/rss_test.go::TestManager_RSSBaseline`（root PID = Chrome 主进程，CIM 采样整棵进程树 `WorkingSetSize`）。

| 采样点 | RSS |
|---|---|
| Chrome 空载（0 page） | **349.1 MB** |
| N=1 page | **416.6 MB** |
| N=8 page | **738.8 MB** |
| 每多一个 page 的增量（(N8−N1)/7） | **≈46.0 MB** |

> 两次独立运行：`空载=415.8MB / N1=427.8MB / N8=753.3MB / 每 page≈46.5MB` 与 `349.1 / 416.6 / 738.8 / 46.0`——量级稳定。
> 对照形态 C（8 个独立 Chrome）：按每个 Chrome 100–180MB 基线估算 ≈ 8×150 ≈ **1.2 GB 且 8 次冷启动**。实测 8 page 仅 738.8MB（含 1 份 Chrome 基线 349MB），**每 page 46MB ≪ 一个 Chrome 基线**，证明是「1 Chrome + N page」而非「N Chrome」。
> 断言硬化：`rss8 > 2GB` 或 `每 page 增量 > 60MB` 直接判红（退化即红）。

### 1.5 「Close 只关 page」的可执行证据

`TestManager_SingleChrome`：两个 page 的 `Browser()` **指针相同**（1 Chrome），`RawPage()` **不同**（2 个标签页）；关闭 page#1 后 page#2 仍能 `Evaluate` 成功（日志实测 `1+1=2`）——若 `Close` 误关 browser，这一步必炸。

---

## 2. ENGINE-02 动作全集（14 个缺口，逐条对照）

### 2.1 差集 A（常量已声明、switch 原本无 case）—— 5 个

| 动作 | 实现方式 | 落点 | 测试用例（真 Chromium） | 结果 |
|---|---|---|---|---|
| `upload` | `Element.SetFiles(paths)`；`files` 支持 `[]interface{}` / `[]string` / 逗号分隔字符串；先 `os.Stat` 校验存在性（缺文件立即报错，不做静默重试） | `engine_actions.go::executeUpload` → `PageManager.Upload` → `chrome_page.go::Upload` | `TestEngine02_AllActions/upload`：写入临时文件 → upload → 断言 `input.files.length === 1` | PASS |
| `download` | `Browser.WaitDownload(dir)` 返回一次性提交函数，**先跑触发动作再提交等待**（顺序不能反，否则错过 `Page.downloadWillBegin`）；落盘名 GUID，归档为 `SuggestedFilename`；产物路径回写 `targetVar`（默认 `downloadPath`） | `engine_actions.go::executeDownload` → `PageManager.Download` → `chrome_page.go::Download` | `TestEngine02_AllActions/download`：点击 `/download` 链接 → 断言落盘文件存在且内容含 `ENGINE-02 download payload`（实测 `engine02.txt` 27 bytes） | PASS |
| `evaluate` | `PageManager.Evaluate` 返回值入变量（默认 `result`，可配 `targetVar`）；`ChromePage.Evaluate` 做**函数形态兼容**：先按 rod 语义（JS 函数定义）执行，失败再包成 `() => (expr)` 重试（rod 的 `Eval` 会对裸表达式 `.apply` 报错） | `engine_actions.go::executeEvaluate` → `PageManager.Evaluate` → `chrome_page.go::Evaluate` | `TestEngine02_AllActions/evaluate`：取 `#sel` 的 value 写 `selValue` → `fill` 用 `${selValue}` 引用 → assert value=`前缀-a` | PASS |
| `close` | `CloseCurrentTab()`：还有其它页签 → 关闭并激活剩余；**最后一页保护** → 新建空白页继续承载（避免 close 把执行打断）。关闭后以「实时列表」为准（不用关闭前快照，Chrome target 销毁异步，快照 Activate 会报 `No target with given id found`） | `engine_actions.go::executeClose` → `PageManager.CloseCurrentTab` → `chrome_page.go::CloseCurrentTab` | `TestEngine02_AllActions/close`：close 后另起一次执行仍能正常跑动作（page 可用） | PASS |
| `condition` | **真条件分支**：`mode=js`（`expression`/`condition`，包成 `Boolean(...)` 求值）或 `mode=variable`（`left/op/right`，op: `eq\|ne\|gt\|ge\|lt\|le\|contains\|empty`）；`then`/`else` 是嵌套动作数组，经 `nestedActionsFrom` 归一化后复用 `executeAction`（与 `executeLoop` 同族） | `engine_actions.go::executeCondition` | `TestEngine02_AllActions/condition`：JS 条件为真 → 走 then 分支并 assert 值=`then-branch` | PASS |

### 2.2 差集 B（常量与 case 双缺）—— 8 个 + 1 个既有语义补全

| 动作 | 实现方式 | 落点 | 测试用例（真 Chromium） | 结果 |
|---|---|---|---|---|
| `assert` | 6 种断言：`text`（默认）/`value`/`html`/`attribute`/`expression`(JS 布尔)/`exists`(阶梯等待)；`contains=true` 走包含而非相等；**失败返回 error 由 `Execute` 短路终止任务**（失败即终止策略） | `engine_actions.go::executeAssert` | `click`/`fill`/`select`/`hover`/`doubleClick`/`rightClick`/`drag`/`keyboard`/`switchTab`/`iframe`/`condition` 用例**全部以 assert 收口**（11 处断言同时验证了 assert 本身） | PASS |
| `hover` | `Element.Hover()`（`WaitInteractable` + 鼠标移动，触发 `mouseenter` / CSS `:hover`） | `engine_actions.go::executeHover` → `PageManager.Hover` | `TestEngine02_AllActions/hover`：hover `#hoverbox` → assert `#hover-count`=`1`（页面自己的 mouseenter 计数） | PASS |
| `doubleClick` | `Element.Click(ButtonLeft, 2)`（`ClickWith` 统一承载 button+count） | `engine_actions.go::executeDoubleClick` → `PageManager.DoubleClick` | `TestEngine02_AllActions/doubleClick`：双击 → assert `#dbl-count`=`1` | PASS |
| `rightClick` | `Element.Click(ButtonRight, 1)` | `engine_actions.go::executeRightClick` → `PageManager.RightClick` | `TestEngine02_AllActions/rightClick`：右键 `#ctx-area` → assert `#ctx-count`=`1`（页面 `contextmenu` 计数） | PASS |
| `drag` | **HTML5 兼容双策略**：① `Input.dispatchMouseEvent` 序列（pressed → 8 步 moved → released）优先；② 用 **DOM 变更计数器**做效果见证（Chrome 的鼠标序列常「不报错但也不触发 dragstart/drop」），无变更即降级；③ 降级方案 `DataTransfer` + `dragstart/dragenter/dragover/drop/dragend` JS 序列。关键坑：rod 只把 `*proto.RuntimeRemoteObject` 当远程对象传参，必须传 `el.Object`（传 `*rod.Element` 会被序列化成 JSON，报 `src.dispatchEvent is not a function`） | `engine_actions.go::executeDrag` → `PageManager.Drag` → `dragElements`/`dragByInput`/`dragByJS` | `TestEngine02_AllActions/drag`：拖 `#draggable`→`#dropzone` → assert `#drop-count`=`1`（页面 drop 处理器的副作用） | PASS |
| `keyboard` | 组合键解析（`ctrl/shift/alt/meta` 别名 + `enter/esc/tab/方向键/F1-F12` + 单字符）→ `page.KeyActions()` 逐个 `Press` 后统一 `Do()`（rod 自动补 release）；`PageManager.PressKeys` → `Page.Keyboard.Press` 语义 | `engine_actions.go::executeKeyboard` → `PageManager.PressKeys` → `chrome_page.go::PressKeys`/`parseKeyCombo` | `TestEngine02_AllActions/keyboard`：点击 `#keyin` → `ctrl+shift+a` → assert `#key-log` 含 `ctrl+shift`（页面 keydown 记录的修饰键） | PASS |
| `switchTab` | `browser.Pages()` 列举 + **稳定序号规则**：`0`=当前页签，`1..k`=本次执行新开的页签（不在借出快照 `baseTargets` 里，按 TargetID 升序），余下=其它既有页签。激活后 `WaitLoad`（新页签常「先 about:blank 再导航」）。`Reset` 会关闭本次执行新开的页签，避免孤儿 target 吃满并发（实测不加会卡死 `Acquire`） | `engine_actions.go::executeSwitchTab` → `PageManager.SwitchTab` → `switchTarget`/`SwitchTab` | `TestEngine02_AllActions/switchTab`：点 `#open-tab` 开新页签 → `switchTab index=1` → assert 新页签 `#tab2-marker`=`tab2` | PASS |
| `iframe` | `params.action=enter`（`Element.Frame()` 取 frame 句柄，之后所有元素级动作走 `ContextPage()` 即 frame 上下文）/ `exit`（回主文档）。`RawPage()` 恒为「当前活动页签」，不受 frame 影响（155 的 screencast 要的是页签句柄） | `engine_actions.go::executeIframe` → `PageManager.EnterFrame/ExitFrame` → `chrome_page.go` | `TestEngine02_AllActions/iframe`：进入 `#frame1` → 点帧内按钮 → assert 帧内 input value=`from-frame` → 退出 frame | PASS |
| `scroll`（元素内滚动，REQUIREMENTS 里属 ENGINE-02 清单） | `scroll` 动作既有 `down/up/toElement`，本 phase **补 `direction=in\|element`**：`el.Eval(function(dx,dy){this.scrollBy(dx,dy)})`，只滚容器不滚整页 | `engine.go::executeScroll`（新增 case 分支）→ `PageManager.ScrollInElement` | `TestEngine02_AllActions/scrollInElement`：注入 60px 高滚动容器 → `direction=in, deltaY=100` → assert `scrollTop > 0` | PASS |

### 2.3 `types.go` ↔ `engine.go` 一致性静态断言（提交即拦截）

`internal/executor/engine_action_coverage_test.go`：
- **AST 扫描**（非反射）：解析 `types/types.go` 收集所有 `Type: ActionType` 常量 → 解析 `engine.go` 定位 `executeAction` 内 `switch action.Type` 收集全部 `case types.ActionXxx` → **差集非空即红**；
- 反向断言：switch 不得引用 types 包不存在的动作常量（防拼写错）；
- `switch` 必须有 `default`（未支持动作走可读错误而非静默）；
- 常量字符串值唯一性断言（防两个常量映射同一字符串导致 case 不可达）。

**红绿变异验证（已实跑）**：手工删掉 `case types.ActionIframe:` → `TestActionTypeSwitchCoverage` **FAIL**（`types.go 声明但 engine.go switch 无 case 的动作 1 个: ActionIframe`）→ 恢复 → **PASS**。当前 `26 个 ActionType 常量全部有 switch case`。

### 2.4 `ACTION_TYPE_OPTIONS` 归属（协议第 7 条）

本 phase 是动作枚举的上游，**已在 `types.go` 补齐 8 个常量 + `script`**；前端 `xingran-react-frontend/src/pages/operations/rpa/constants.tsx` 的 `ACTION_TYPE_OPTIONS` **未在本 phase 改动**（归属 154 但属前端写面，与 ENGINE-01~05 后端验收无关，且 156 只消费）。已在 §6「遗留项」登记：`ACTION_TYPE_OPTIONS` 需补 `evaluate/hover/doubleClick/rightClick/drag/keyboard/switchTab/iframe/condition/close/download/scroll/loop/waitFor/screenshot/autologin` 与中文标签，建议在 156 消费前由一个前端动作统一提交（避免与 152/156 争写同一文件）。

---

## 3. ENGINE-03 选择器阶梯与自愈

### 3.1 7 级阶梯（顺序 = 稳定性，`SelectorTier` 常量序即实现序）

| 级 | 层级 | 评分 | 探测原语 | 候选生成（`ladderCandidates`） |
|---|---|---|---|---|
| 1 | `data-testid` | 100 | `Page.Has(css)` | `[data-testid="x"]` / `[data-testid*="x"]` |
| 2 | `role + accessible name` | 90 | `Page.HasR(css, 文本正则)` | 先 `[role]`，再语义化标签 `button/a/label/input[type=submit\|button]`（**刻意不用裸 `*`**：那会把整页任意文本当命中，破坏分层价值） |
| 3 | `aria-label` | 80 | `Page.Has(css)` | `[aria-label="x"]` / `[aria-label*="x"]` |
| 4 | 稳定属性（id / name） | 65 | `Page.Has(css)` | 原始 CSS 直试 + `[name=]` + `#id` + `[id^=\|*=\|$=]` × **id 容错变体**（原值 / 去尾号 `user-row-42→user-row` / 首段 `order-2024-0001→order`） |
| 5 | 可见文本 | 45 | `Page.Search(text)` + `Visible()` 校验 | 仅对「纯文本/中文标签」形态生效；**CSS 形态不生成候选**（避免把 `div.whatever` 当字面量全页搜索，实测单级要耗 10s+） |
| 6 | 结构路径 | 20 | `Page.Has(css)` | 仅当输入含 `>` 的 CSS 时直试 |
| 7 | 绝对 XPath | 5 | `Page.HasX(xpath)` | XPath 原串；非 XPath 输入派生 `//*[contains(text(),...)]` / `@id` / `@name` |

- **单级探测预算** `tierProbeTimeout = 500ms`（7 级上限 3.5s，与「等待」解耦：等待语义交给 `WaitFor` 的轮询）。
- 全部 7 级失败 → `*LadderError`，`Error()` 逐级回显「层级 + 候选 + 原因」，同时以 `success=false` + `tier` 写入选择器事件信封（后端 `SelectorLearner` 累计 ≥3 次触发 `selector_update_required` 告警）。
- 命中层级经 `recordSelectorHit` → `SelectorEventCollector.RecordWithTier`（复用 `selector_events.go:Record` 的同一信封，新增可选键 `tier`，**契约向后兼容**：旧消费方忽略未知键）。
- **「录制生成的选择器优先落 testid/role」约定已写进 `page_manager.go` 文件头注释**（NOCODE-02 录制器必须遵守）。
- **全路径接入**：`Click/Fill/Select/GetText/GetValue/GetHTML/GetAttribute/Hover/Upload/Drag/ScrollInElement/ScrollToElement/EnterFrame/WaitFor` 全部走 `resolveElement` → `findElement`，阶梯不是「只存在于单测」的装饰。

### 3.2 验收：阶梯评分单测 + 真实页面用例

| 用例 | 断言 | 结果 |
|---|---|---|
| `TestLadderScoreMonotonic` | 7 级评分严格递减、testid=100 / xpath=5 | PASS |
| `TestLadderTiersComplete` | 恰好 7 级、常量序未被调换、名称可读 | PASS |
| `TestParseSelectorHints` | testid / aria / `#id` / XPath / `text=` / 纯文本 / `div.nav > span` 七种形态的线索提取 | PASS |
| `TestLadderCandidatesPerTier` | 各级候选形态；**CSS 形态不得有文本级候选** | PASS |
| `TestIDHintVariants` | 动态 id 容错变体 `user-row-42 → [user-row-42, user-row, user]` | PASS |
| `TestLadderErrorFormat` | 全失败错误含「共 7 级」且逐级点名 | PASS |
| **`TestLadder_RealPage_RoleBeforeText`**（fixture A） | 同一文本「保存草稿」既有 `<div role="button" id=...>` 又有 `<span id=...>`；输入「保存草稿」→ **命中 `role+name`（第 2 级，score=90）**，即文本相同的 role 元素必须在可见文本级之前被选中 | **PASS**（实测 `命中层级: role+name (score=90)`） |
| **`TestLadder_RealPage_DynamicID`**（fixture B） | 录制时 `#row-1000`，运行时页面变成 `#row-9001` → 借 `[id*="row"]` 容错命中 `stable-attr`（第 4 级），文本含「订单编号」 | **PASS**（实测 `命中层级: stable-attr (score=65)`） |
| `TestLadder_RealPage_AllTiersFail` | 不存在元素 → `*LadderError` 且 `Attempts` 长度 == 7；事件信封含 `"success":false` | PASS |

> 说明：场景 A 的「testid 缺失但 role 存在」按真实调用形态验证（录制器在内网系统上产出的是可见名/role 锚点，而不是凭空造一个 `[data-testid=x]`——若真输入 testid 选择器，正确行为是降级到文本级而非硬套 role）。

---

## 4. ENGINE-04 变量与数据流

### 4.1 三层作用域（`executor/variables.go`，401 行）

| 层 | 来源 | 写回 | 查找优先级 |
|---|---|---|---|
| `Loop`（循环项级） | 每轮 `executeLoop` 新建（`BreakLoopScope`），挂 `item` 对象整体 + `item.index` + 扁平字段（向后兼容 `${fieldName}`） | `extract` 的 `scope=loop` | 最高 |
| `Task`（任务级） | `task.Variables` 非敏感键；`extract` / `evaluate` / `download` 默认写这里 | `scope=task`（默认） | 中 |
| `Session`（会话级） | `__credentials` / `__sessionData` 等敏感键（跨任务复用） | `scope=session` | 最低 |

### 4.2 表达式与策略

| 能力 | 实现 | 测试 |
|---|---|---|
| 扁平 `${var}` | `varExprRe` 非贪婪匹配 `${...}` | `TestResolveTemplate` |
| 嵌套 `${a.b[0].c}`（含负下标 `${items[-1].name}`） | `parsePath` → `pathSeg{key\|index}` → `lookupIn` 逐段下钻（兼容 `map[string]string` / `[]string`） | `TestParsePath` / `TestResolveNestedPath` |
| 默认值 `${a.b[9]:fallback}` | 冒号后缀字面量兜底 | `TestResolveTemplate` |
| 整棵参数树递归求值 | `ResolveValue`（string / `[]interface{}` / `map[string]interface{}`，map 键排序保证确定性） | `TestResolveValueRecursive` |
| **未定义策略可配** | `executor.undefined_variable_policy`：`strict`（报错终止并点名变量）/ `empty`（空串，**默认**，向后兼容）；非法值在 **配置加载期** 报错 | `TestUndefinedVariablePolicy` / `TestEngineExecute_UndefinedPolicyRouting` / `TestResolveTemplate` |
| extract 嵌套写回 | `variables.Set(scope, "rows[1].name", v)`；中间节点缺失自动创建 map/数组；下标越界报错（不静默丢） | `TestExtractNestedWriteBack` |
| extract 结果被后续动作引用 | 端到端：extract `#log` → `fill #name` 用 `${captured}` | `TestEngine02_AllActions/extract` |
| **敏感变量脱敏** | `SensitiveVariableKeys`（`__credentials` / `__sessionData` / `password` / `token` / `accessToken` / `refreshToken`）+ `MaskVariables` / `MaskSensitive`；日志与错误信息一律 `***`；**模板替换里引用敏感变量也输出 `***`** | `TestSensitiveMasking` |

### 4.3 顺带修掉的两处既有缺陷（实测发现）

1. **`fill` 是追加式**：rod 的 `Element.Input` 在已有内容后追加，循环第二轮变成 `L1L2`（`TestEngine02_AllActions/loop` 实测捕获）。已改为「先 `SelectAllText()` 清空再输入」的**覆盖语义**（`chrome_page.go` 与 `page_manager.go` 两处路径都已修）。
2. **`Page` 不能绑执行 ctx**：原 `newChromePageOn` 用 `page.Context(ctx)`，执行结束 ctx cancel 后 page 连带失效（日志 `关闭页面失败 context canceled`），池化复用实际报废。已移除绑定（Manager 跨执行复用 page，生命周期归 Manager）。

---

## 5. ENGINE-05 真实浏览器验收基线

### 5.1 测试矩阵（真 Chromium，无 mock page）

| 包 | 用例文件 | 数量 | 性质 |
|---|---|---|---|
| `browser` | `manager_test.go` / `rss_test.go` / `selector_ladder_test.go` / `ladder_integration_test.go` / `page_manager_test.go` / `selector_events_test.go` | 6 文件 | 单元（无浏览器）+ 真 Chromium（Manager 共享/Close 语义/RSS/阶梯真实页面） |
| `executor` | `engine_action_coverage_test.go`（AST 静态断言）/ `engine_variables_test.go`（变量与数据流）/ `fixture_test.go`（fixture 页 + 引擎工厂）/ `engine_integration_test.go`（14 动作 + loop） | 4 文件 | **首次拥有真实用例**；`-short` 保留静态断言与变量单测 |
| `communication` | `progress_reporter_test.go` | 1 文件 | **首次拥有用例**：httptest 真收包，断言 workerId/timestamp 由 reporter 填充、信封行独占一行且可解析、500 返回可读错误 |
| `worker` | `worker_test.go` | 1 文件 | **首次拥有用例**：合并后的并发页面数、scaleUp/scaleDown 边界与「有任务时不缩容」、health 统计、旧键告警文案含新旧键名 |
| `config` | `config_test.go` | 1 文件 | **首次拥有用例**：max_pages 合并四态、策略默认值与非法值、env 覆盖后视图同步 |

改动前基线：`go test ./...` 只有 `browser` 有测试，其余全 `[no test files]`。
改动后：`browser / communication / config / executor / worker` **五包全绿**（`logger` / `types` 纯常量与工具包仍无测试，非本 phase 范围）。

### 5.2 CI 兜底（`-short`）

| 跳过（重型，真 Chromium） | 保留（轻量兜底） |
|---|---|
| `TestManager_SingleChrome` / `TestManager_CloseShutsBrowser` / `TestManager_RSSBaseline` / `TestLadder_RealPage_*` / `TestEngine02_AllActions` | `TestManager_Unit_*`（容量/死页丢弃）、`TestLadder*`（评分/线索/候选/错误格式）、`TestActionTypeSwitchCoverage` / `TestActionTypeConstantsAreUnique`（**动作路由静态断言**）、`TestResolve*` / `TestParsePath` / `TestVariableScope*` / `TestSensitive*`（参数校验与变量策略）、`TestMerge_*`（配置兼容）、`communication` 全部、`worker` 全部 |

`-short` 实跑：browser 3.99s / communication 2.17s / config 1.57s / executor 0.19s / worker 1.78s，**全绿**。

### 5.3 fixture 页面覆盖面

`fixture_test.go` 内嵌 HTML 提供：`form` + `#name` 输入 + `#sel` 下拉 + `#file` 文件域 + 提交按钮（click 写 `#log`）、双击计数、右键计数、hover 计数、`draggable`→`dropzone`（drop 计数）、`#keyin` 键击日志、`/download` 附件、`window.open` 新页签、`#frame1` iframe（帧内按钮 + input）、运行期注入的滚动容器。下载产物落 `t.TempDir()`，不污染仓库。

---

## 6. 遗留项（显式登记，不在本 phase 谎报完成）

| # | 遗留 | 影响 | 建议归属 |
|---|---|---|---|
| L1 | `ACTION_TYPE_OPTIONS`（前端 `constants.tsx`）仍是 9 项旧列表，未含本 phase 新增动作 | 前端下拉选不到 `hover/drag/iframe` 等；但**后端已能执行**（录制器/AI 生成即可用） | 156 消费前由前端一次性补齐（协议第 7 条：154 是上游、156 只消费） |
| L2 | RSS 基线只在 **Windows + 本机 Chrome** 实测；`measureLinuxTreeRSS` 已实现但未在容器内跑到 | FLEET-03 的容量矩阵需要 Linux 容器实测数字 | FLEET-03（本 phase 数据仅作量级证明：每 page 46MB，非 Linux 结论） |
| L3 | `download` 依赖 `Browser.WaitDownload` 的 `EachEvent` 长期订阅，多并发同时下载时事件按 GUID 匹配但实现为全局订阅 | 同一 Chrome 内 2+ 任务同时下载可能互相干扰（未实测） | 若 FLEET 压测暴露，改为下载前串行化或独立 BrowserContext |
| L4 | `close` 语义为「关当前页签 + 最后一页保护（新建空白页）」，**不关闭 Chrome**（Chrome 生命周期归 Manager） | 与历史脚本「close 等于结束浏览器」的直觉不同 | 已在 `types.go` 注释与本文说明；若产品需要「结束整个浏览器」，应新增 `closeBrowser` 动作（需 Manager 重启能力，归 155/157） |
| L5 | `iframe` 只支持一级 frame（进入后必须先 exit 才能再 enter） | 多层嵌套 iframe 暂不支持 | 后续按需扩展（`ContextPage` 抽象已就位） |
| L6 | `communication` 无 `-short` 与重型之分（httptest 已很轻，无需分层） | 无 | — |

---

## 7. 155 可依赖的 `browser.Manager` 接口清单（**契约冻结**）

> 155 只依赖下列签名与语义编码；**任何签名/语义变更必须由 155 与 154 owner 双方确认**，否则 155 的 screencast 会在错误时机断流。

### 7.1 构造与生命周期

```go
// 创建管理器。Chrome 懒启动（首次 Acquire 才拉起），构造本身不启动浏览器、不阻塞。
func NewManager(cfg *config.BrowserConfig, log logger.Logger) (*Manager, error)

// 关闭：先关所有空闲 page，再 Close 共享 browser，最后 Kill launcher 进程。
// 幂等（重复调用返回 nil）。Close 之后 Acquire 立即返回错误「浏览器管理器已关闭」。
func (m *Manager) Close() error

// 共享的唯一 rod.Browser（未启动或已 Close 时返回 nil）。
func (m *Manager) Browser() *rod.Browser
```

### 7.2 page 分配与回收

```go
// 取一个 page：优先复用空闲页；否则在容量内新建（复用同一 Chrome）；
// 达上限则阻塞等待 Release/SetCapacity，或 ctx 取消（返回 ctx.Err()）。
// 永不返回已关闭的 page。
func (m *Manager) Acquire(ctx context.Context) (Page, error)

// 归还 page：复位（退出 iframe、关闭本次执行新开的页签、页签上下文回根页签）后入空闲队列；
// 输入 nil 安全；已关闭 page 直接丢弃。
func (m *Manager) Release(p Page)

// 当前已创建 page 数（含空闲）。
func (m *Manager) PageCount() int

// Chrome 主进程 PID（0 = 尚未启动）。RSS 基线与运维排查用。
func (m *Manager) PID() int

// 动态调整并发页面数上限（worker 扩缩容同步调用）。调小不打断已借出 page。
func (m *Manager) SetCapacity(n int)
func (m *Manager) Capacity() int
```

### 7.3 `Page` 接口（`browser/pool.go`，155 的 screencast 入口）

```go
type Page interface {
    // ...（既有 Goto/Click/Fill/... 与 ENGINE-02 动作原语略）

    // RawPage 返回「当前活动页签」的底层 rod 句柄 —— screencast / 高级 CDP 的唯一出口。
    // switchTab 后会跟随到新页签；不受 iframe 影响（进入 frame 不会改变它）。
    RawPage() *rod.Page

    // ContextPage 返回「当前操作上下文」：iframe 内为 frame 句柄，否则等同 RawPage。
    // 元素定位/交互走它；screencast 必须用 RawPage（frame 不能 startScreencast）。
    ContextPage() *rod.Page

    // Browser 返回该 page 所属的共享 rod 浏览器（同一 Manager 下所有 page 指针相同）。
    Browser() *rod.Browser

    // Close 只关闭当前 page（标签页），**绝不关闭 browser**。
    // 幂等；已关闭后 Closed() 返回 true。Manager.Release 会据此丢弃死页。
    Close() error
    Closed() bool

    // Reset 清理一次执行的瞬态状态：退出 iframe、关闭本次执行新开的页签、
    // 页签上下文回根页签、重拍页签快照。Manager.Release 与 Acquire 都会调用。
    Reset()
}
```

### 7.4 硬语义（155 必须遵守的 4 条）

1. **`Page.Close()` 只关 page**：screencast 绑在 page 上；若回收时连带关 Chrome，同容器内其它并发流会一起断。
2. **screencast 必须用 `RawPage()`**（当前页签），不要用 `ContextPage()`（frame 不能开 screencast）。执行中若发生 `switchTab`，`RawPage()` 会跟随，155 需在页签切换时重新 `startScreencast`。
3. **`Manager.Acquire` 的 page 归还前不要自己 `Close()`**（想释放就 `Release`）；主动 `Close()` 后被 `Release` 会按死页丢弃，等效但浪费一次创建。
4. **并发上限 = `Manager.Capacity()`**（= 合并后的 `worker.max_concurrency`）；155 的「单接管者」与「超并发返回可读错误」应以此值为准，不要另建计数器。

---

## 8. 越界检查

| 归属协议 | 要求 | 实际 |
|---|---|---|
| 第 1 条（`config.go` schema 归 154） | 你拥有 schema（含 `max_pages → max_concurrency` 合并），157 只改取值、155 只加 screencast 键 | **只改 rpa-worker 的 `internal/config/config.go`**（+ `configs/config.yaml` 注释）；未触碰根模块 `internal/config/config.go`（那是 157 的 `RPAScalingConfig`，属另一模块）。旧键兼容行为已写进 §1.3 供 157 CI 读取 |
| 第 2 条（`types/types.go` 只加动作常量） | 禁止在 `types.go` 加 liveview/screencast 任何类型 | **只在 `types.go` 增加 9 个 `ActionType` 常量**（8 缺口 + `script`）；无任何 liveview/screencast 字段/类型；155 的 `types/liveview.go` 可无冲突新建 |
| 第 3 条（`browser/manager.go` 契约冻结） | 完成后写「155 可依赖的接口清单」 | 已冻结并写入 **§7**（精确签名 + 语义 + 4 条硬语义），代码文件头也内嵌同一份清单 |
| 第 5 条（155 接口前置） | 154 先落 `Acquire/Release/RawPage`/「Close 只关 page」/`types/liveview.go` 独立 | 前三项已落并有可执行证据（§1.5）；`types/liveview.go` 未创建（留给 155） |
| 第 7 条（`ACTION_TYPE_OPTIONS` 上游是 154） | 154 扩充、156 只消费 | 后端枚举已扩充（`types.go`）；**前端 `constants.tsx` 本 phase 未改**（见 §6 L1，属前端写面，需在 156 前补齐） |
| 152 并行（`task_service.go` / `rpa_router.go` / 前端） | 不得写 152 的文件面 | **零触碰**：本次改动 100% 在 `rpa-worker/` 内（+ `.planning/phases/154-engine/`）；未改根模块任何 `internal/**`、未改前端任何文件 |
| 153 并行（`worker.go` / `engine.go` 函数不相交） | 153 只碰 RUNTIME 相关函数 | 本次 `engine.go` 改的是 `Execute`/`executeAction`/`executeLoop`/`executeExtract`/`executeScroll`/`executeAutoLogin`/`substituteVariables`/`replaceVariables` + 新增 `engine_actions.go`；`applySessionData` **只改 1 行签名与 0 行函数体**（保持 D8 占位现状，交给 153 实现）。`worker.go` 改的是池构造、扩缩容同步、cleanup 三处 |
| 构建产物 | 只落 `out/`；不碰 `rpa-worker/rpa-worker` 旧二进制 | 无任何 `.exe/.test/.out` 产物；`go build -o NUL ./...` 零落盘；旧二进制未触碰（`git status` 无该文件） |
| 零新增 Go 依赖 | 只用 go-rod v0.116.2 既有 API | **`go.mod` / `go.sum` 零 diff** |
| 验收禁止 mock page | 必须真 Chromium | 14 动作 + 阶梯 + Manager 全部真 Chrome 用例（`newTestManager` 只做「无 Chromium → skip」，不替换实现） |

---

## 9. 复现命令

```bash
cd rpa-worker
go build ./...                      # exit 0
go build -o NUL ./...               # Windows（Linux/macOS: -o /dev/null）确认无仓库产物
go vet ./...                        # 无输出
go test ./...                       # 全绿（真 Chromium；本机 28.7s + 8.2s + 2.1s + 1.4s + 1.3s）
go test ./... -short                # CI 兜底全绿（重型用例 skip）
go test ./internal/browser -run TestManager_RSSBaseline -v   # 看 RSS 基线数字
go test ./internal/executor -run TestEngine02_AllActions -v  # 14 动作逐条
```

> 指定浏览器：`RPA_TEST_CHROME_PATH=/path/to/chrome go test ./...`（默认按 Chrome / Edge / chromium 常见路径探测，见 `browser/testchrome.go`）。

---

## 10. 文件改动清单（worktree 内）

**新增（17）**：`browser/manager.go`、`browser/launcher.go`、`browser/testchrome.go`、`browser/selector_ladder.go`、`browser/manager_test.go`、`browser/rss_test.go`、`browser/selector_ladder_test.go`、`browser/ladder_integration_test.go`、`executor/engine_actions.go`、`executor/variables.go`、`executor/engine_action_coverage_test.go`、`executor/engine_variables_test.go`、`executor/fixture_test.go`、`executor/engine_integration_test.go`、`communication/progress_reporter_test.go`、`worker/worker_test.go`、`config/config_test.go`

**修改（10）**：`browser/pool.go`（Pool→Page 契约）、`browser/chrome_page.go`、`browser/page_manager.go`、`browser/page_manager_test.go`（stub 补新接口）、`browser/selector_events.go`（`RecordWithTier` + `tier` 键）、`executor/engine.go`、`types/types.go`（+9 常量）、`config/config.go`、`worker/worker.go`、`configs/config.yaml`

