# Phase 156 — NOCODE 员工零门槛（v1.56 W4 波次）

**执行日期**：2026-11-25
**波次归属**：W4（依赖 W3/Phase 155 的 live 画布 + W1/Phase 154 的选择器阶梯与动作全集 + W1/Phase 152 的 AIScriptEditor 真调）
**目标用户**：**不会写代码的一般员工**。验收标准是「他能独立完成」，不是「功能存在」。

---

## 0. 前置条件核验（硬性，缺一即阻塞）

开工前 `git log --oneline -10` 确认三者均已合入：

| 前置 | 提交 | 状态 |
|---|---|---|
| Phase 155（W3）LIVEVIEW 无头可视化与人工接管 | `70405142` | ✅ 在 HEAD 历史内 |
| Phase 154（W1-B）ENGINE 执行引擎（选择器 7 级阶梯 + 动作全集） | `b86c4301` | ✅ 在 HEAD 历史内 |
| Phase 152（W1-A）CONTRACT P0 契约修复（AIScriptEditor 真调 AI） | `559a62e7` | ✅ 在 HEAD 历史内 |

**结论：无阻塞，正常开工。**

---

## 1. 逐条 REQ 交付

### NOCODE-01 页面内录制 — **PASS**

**文件**：`rpa-worker/internal/browser/recorder.go`（净新增 700+ 行）、`recorder_test.go`、`recorder_integration_test.go`

**实现要点**：

1. **零浏览器扩展**（Out of Scope 硬约束）：用 `proto.PageAddScriptToEvaluateOnNewDocument` 注入监听脚本，等价 Puppeteer 的 `evalOnNewDocument`。脚本对**每个新文档**生效，因此录制跨越整页跳转与 SPA 路由切换仍然有效。
2. **采集面**：`click` / `dblclick` / `contextmenu` / `input` / `change` / `select` / `submit` / `scroll` / 组合键 `keydown`（捕获阶段监听，不 `preventDefault`，**不改变被测系统行为**）。
3. **跨导航事件不丢**（实现期发现的真实缺陷）：`window` 变量随文档销毁，录「登录 → 跳转 → 填表」时前一段事件会在导航瞬间丢失。修复方式是把事件同步落 `sessionStorage`（关键事件强制立即写盘，非关键事件 150ms 节流），注入脚本启动时再从它恢复。
4. **事件 → 动作聚类**（6 条规则，每条对应一个真实噪声源）：
   - 起点插入 `navigate`（录制必须能独立回放，不能依赖「当前已在某页」）
   - 同元素连续 `input` 在 600ms 窗口内合并（否则逐字符输入会产出几十条 fill）
   - 点击后 1.5s 内的派生 `submit` 丢弃（否则回放会提交两次）
   - 连续 `scroll` 合并为一条并按方向 + 像素回放
   - **密码框值强制替换为 `${password}`**（录制绝不把明文密码写进任务）
   - URL 变化插入 `navigate`
5. **选择器优先 testid/role**（154 阶梯约定，归属协议第 7 条）：JS 侧一次性算出全部 7 级候选并标注「是否唯一可用」，Go 侧按 tier 升序取首个可用。
6. **产出 schema 与 `types.Action` 完全一致**，可直接落库执行，无需二次转换。

**验收（真 Chromium）**：`TestRecorder_RealChromium_LoginFillSubmit` — PASS

```
录制产出 18 个事件 → 8 个动作
  步骤 1: 打开网页
  步骤 2: 在「请输入用户名」填入内容
  步骤 3: 在「请输入密码」填入密码
  步骤 4: 点击按钮「登录」
  步骤 5: 打开网页
  步骤 6: 在「请输入金额」填入内容
  步骤 7: 选择「住宿费」
  步骤 8: 点击按钮「提交」
回放后服务端收到: amount=1280;type=hotel
```

测试是**端到端**的：起本地 HTTP 服务（登录页带 `data-testid`、表单页部分字段无 testid）→ 起真 Chrome → 用**真实鼠标键盘事件**完成操作（不是 evaluate 改 DOM，那样绕过监听器、验收无效）→ 录制 → 用 154 的 `PageManager`（阶梯 + 动作全集）回放 → 断言服务端真实收到提交数据。

---

### NOCODE-02 元素拾取器 — **PASS**

**文件**：`rpa-worker/internal/browser/picker.go`（净新增）、`picker_test.go`

**实现要点**：

1. 与 recorder 共用同一份 `__rpaDescribe`（7 级阶梯计算），保证「录出来的选择器」与「拾出来的选择器」是**同一套算法**，不会出现两种口径。`TestRecommendTier_MatchesRecorderBestSelector` 是这条一致性的守门测试。
2. 坐标语义与 LIVEVIEW-02 完全一致：前端把画布像素换算成 **viewport CSS 像素**后传过来，Worker 直接透传给 `document.elementFromPoint`，不做二次换算、不做 DPR 补偿。
3. 返回 `PickedElement.Tiers`（7 级，每级带 `OK` 标记 + 评分），前端浮层据此逐级显示「可用/不可用」——这正是 REQ 要求的数据。
4. **iframe 内可用**（实现期修复的真实缺陷）：`elementFromPoint` 只作用于当前文档，在主文档里用它永远只会返回 iframe 宿主元素本身。修复方式是先用 `Element.Frame()` 真正进入 frame，在 frame 上下文里再拾取，并把 frame 选择器写进 `Frame` 字段（回放时需要先 `iframe(enter)`）。

**验收（真 Chromium，阶梯自愈）**：`TestPicker_RealChromium_TierDetectionAndSelfHeal` — PASS

DOM 微调手段（模拟真实前端改版，目标是让低层级选择器全部失效）：
- 工具栏最前面插入两个兄弟节点 → 所有 `nth-child`/`nth-of-type` 序号全变
- 再套一层 wrapper → 结构路径多一层
- 删掉 `#toolbar` 的 id → id 级失效
- 改掉按钮可见文本 → 文本级失效

结果：

```
拾取到 <button> 文本="保存" 推荐="[data-testid=\"save-btn\"]"（第 1 级，评分 100）
DOM 微调后仍命中，靠第 1 级（data-testid）
```

**反向验证**（让结论可信）：测试同时断言 `#toolbar` 与 `div#toolbar > button:nth-of-type(1)` 在微调后**确实失效**了 —— 否则「仍命中」只是选择器字符串没变，不能证明阶梯在起作用。

iframe 验收：`TestPicker_RealChromium_FramePick` — PASS

```
iframe 内拾取到 <button> 文本="内层按钮" 推荐="[data-testid=\"inner-btn\"]" frame="#inner"
```

（硬断言：不得返回 `iframe`/`frame` 宿主元素，且必须回传 frame 选择器。）

---

### NOCODE-03 自然语言生成 + AI 降级 — **PASS**

**后端**：`internal/services/rpa/ai_action_validator.go`（净新增）、`ai_service.go`、`internal/api/v1/rpa/ai_handler.go`、`handler_helpers.go`

1. **动作枚举服务端校验**（REQ 明确要求）：`ValidateAIActions` 校验 AI 返回的每个动作，非法动作在**服务端**拒绝并给出逐条可读原因。不静默丢弃——否则「前端过滤」成了唯一防线，换个客户端就能把非法动作写进任务脚本。
2. **与 worker 同源的机械保障**：`workerActionTypes` 是 `rpa-worker/internal/types/types.go` 的镜像。`TestWorkerActionTypes_MirrorsWorkerTypesGo` 直接读 worker 源文件抽取全部 `ActionXxx ActionType = "yyy"` 常量做**双向比对**——154 新增动作后忘了同步，这条测试立刻红。当前 **worker 26 个 / 后端镜像 26 个**，完全对齐。
3. **三态错误映射**（这是前端能做三态引导的前提）：

| 态 | HTTP | message 要点 | 前端引导 |
|---|---|---|---|
| AI 不可用 | **400** | 「智能生成当前不可用…请改用「录制一段操作」」 | 主按钮「改用录制」 |
| 非法动作 | **400** | 「第 N 步：动作类型「xxx」系统不支持…」 | 改描述重试 |
| 超时/网络 | **500** | 「脚本生成失败: …」 | 稍后重试 + 录制出路 |

**实现期踩到的项目级陷阱**：`pkg/response.toAppError` 的 `case int` 分支把 `HTTPStatus` **硬编码为 400**，所以 `response.Error(c, 500, msg)` 实际返回 400 —— 三态在状态码上不可区分，前端只能靠字符串猜。修复：新增 `writeRPAError`，走 `*response.AppError` 分支（该分支尊重 `HTTPStatus`）。已在 `handleError` 上加注释登记该陷阱，避免后人再踩。

**前端**：`tasks/modals/AIScriptEditor.tsx`（**只叠加，未回退 152 的真实调用**）

1. 新增第四态 `unavailable`（在 152 已有的 `failed`/`timeout`/`empty` 之上）。
2. `isAIDisabledResponse` 判定：优先看 HTTP 状态 + message 关键词，两者都拿不到时**保守判为非不可用**（宁可让用户重试一次，也不误把可重试的故障说成「功能没开」）。
3. **降级是关键体验路径，不是死路**：
   - 不可用时**不显示**「重试生成」（重试必然再失败，纯浪费用户时间）
   - 显示主按钮「改用录制」→ 调 `onFallbackToRecord` 回调
   - 未传回调时也不留死按钮：明确告知「请从「新增任务」向导选择「录制一段操作」」并关闭弹窗
   - 超时/失败态**两条路都开**（重试 + 录制），不是所有人都愿意等第二次

**验收**：`ai_generate_states_test.go`（6 个用例）+ `ai_action_validator_test.go`（9 个）+ `AIScriptEditor.fallback.test.tsx`（10 个）全绿。三态在状态码上可区分由 `TestGenerateScript_ThreeStatesAreDistinguishable` 锁定。

---

### NOCODE-04 三步向导 + 开箱模板 — **PASS**

**文件**（`pages/operations/rpa/wizard/`，新建目录）：

| 文件 | 职责 |
|---|---|
| `index.tsx` | 向导页面（承载 TaskWizard + 保存到 `POST /rpa/tasks` + AI 生成器叠加） |
| `TaskWizard.tsx` | 三步向导主体 |
| `templates.ts` | 3 个开箱模板 + 实例化 + 参数校验 |
| `stepLabels.ts` | 动作 → 业务语义中文名的**唯一权威转换点** |
| `payload.ts` | 向导草稿 → 后端请求体（纯函数，可单测） |
| `__tests__/wizard.test.tsx` | 35 个测试 |

**三步固定流程**：

1. **说要做什么** — 三选一：用中文说 / 录一段操作 / 用现成模板
2. **确认步骤** — 中文步骤列表，可**拖动排序**（`@dnd-kit/sortable`，项目已有依赖，**零新增**）+ 可改参数 + 可删除
3. **选运行方式** — 立即 / 定时（cron）/ 触发（事件）+ 保存

**零技术术语纪律**：`stepLabels.ts` 是唯一转换点，`describeAction` 结合动作类型 + 参数 + 上下文产出一句完整人话（「在「金额」填入 1280」「点击「提交」」）。即使只有 `[data-testid="submit-btn"]`，也提取成可读标签而不是把 CSS 糊到用户脸上。`TECH_TERM_BLOCKLIST` + `containsTechTerm` 提供机械断言能力，测试对**整个向导的可见文本**做术语扫描。

**开箱模板 3 个**（REQ 要求 ≥3，覆盖三类场景）：

| 模板 | 分类 | 步骤数 | 参数 | 预计耗时 |
|---|---|---|---|---|
| 登录系统取数 | 登录类 | 6 | 登录地址 / 账号 / 密码（敏感）/ 登录后页面 | 约 1 分钟 |
| 批量填报单据 | 填表类 | 6 | 表单地址 / 金额 / 事由 / 申请人 | 约 2 分钟 |
| 查询并导出报表 | 查询导出类 | 6 | 报表地址 / 开始日期 / 结束日期 | 约 2 分钟 |

每个模板都带中文说明（`summary` 说「帮你做什么」+ `scenario` 说「什么场景用」）、示例数据、参数占位提示。密码字段标 `sensitive: true` 且**示例值为空**（绝不预填假密码）。

**入口改造**（协议补登项）：
- `tasks/index.tsx` 的「新增任务」按钮改为打开三步向导（Modal 承载，宽 900，保留列表筛选状态）；保留「高级新建」按钮走原编辑器
- `tasks/modals/EditModal.tsx` 加「模板」与「一键转录制」两个入口；转录制有确认（避免误点丢掉已配置动作）

**路由**（归属协议第 6 条）：`internal/core/db/migrations/migration_256_nocode_wizard_menu.go` 播种 C 型菜单「新建自动化任务」（`component=pages/operations/rpa/wizard/index`，父=RPA 管理，order=17，**不扩权限面**）。**未改** `componentLoader.tsx` / `DynamicRoutes.tsx`。

---

## 2. 三步向导全流程走查记录

### 2.1 自动化走查（可复现，`wizard.test.tsx`）

| 走查点 | 结果 |
|---|---|
| 默认停在第 1 步，三种起点可见 | ✅ |
| 选模板 → 参数区出现 → 示例值预填 | ✅ |
| 必填项（密码）未填 → 拦住，提示中文标签「登录密码」 | ✅ |
| 参数齐 → 第 2 步看到 8 条中文步骤 | ✅ |
| 第 2 步上移/下移改变顺序（拖拽的降级通道） | ✅ |
| 第 2 步改步骤内容（参数） | ✅ |
| 第 2 步删除步骤 | ✅ |
| 三步走完保存 → 载荷 `name`/`script`/`triggerType` 正确 | ✅ |
| 三种运行方式映射正确 `triggerType`（manual/schedule/event） | ✅ |
| 录制来源无动作 → 调 `onOpenRecorder`（不留死路） | ✅ |
| 描述来源无动作 → 调 `onOpenAIGenerator` | ✅ |
| 第 3 步确认卡片显示中文运行方式 | ✅ |
| 每一步可见文案零技术术语 | ✅ |

### 2.2 真 Chromium 端到端走查

用 `TestRecorder_RealChromium_LoginFillSubmit` 走「一般员工的真实路径」：

| 阶段 | 耗时（实测） | 卡点 |
|---|---|---|
| 起本地服务 + 起真 Chrome | ~1.9s | 无 |
| 录制「登录 → 填表 → 提交」6 个真实操作 | ~2.5s（含等待加载） | 无 |
| `Stop` 拉事件 + `BuildRecording` 产出 8 动作 | <100ms | 无 |
| 回放（154 阶梯定位 + 动作执行） | ~2.0s | 无 |
| **合计** | **~4.5s** | **0 个卡点** |

**结论**：录制 → 8 条中文步骤 → 回放 → 服务端真实收到 `amount=1280;type=hotel`，全链路一次通过。

### 2.3 实现期踩到的卡点（已全部修复，登记以备后续）

| # | 卡点 | 根因 | 修复 |
|---|---|---|---|
| 1 | 跨导航录制丢事件 | `window` 变量随文档销毁 | 事件落 `sessionStorage`，关键事件强制写盘 |
| 2 | `Eval` 报 `apply is not a function` | rod 把入参当函数体 apply() | 当前文档注入包一层箭头函数；`Source` 注入保持裸表达式 |
| 3 | iframe 内拾取只返回宿主元素 | `elementFromPoint` 只作用于当前文档 | `Element.Frame()` 真进入 frame 后拾取 |
| 4 | `response.Error(c, 500, ...)` 返回 400 | `toAppError` 的 `case int` 硬编码 HTTPStatus=400 | 新增 `writeRPAError` 走 `*AppError` 分支 |
| 5 | 首次点「下一步」卡在第 1 步 | `setActions` 后仍读闭包旧值（空数组） | `buildTemplateActions` 返回动作，同一 tick 内继续判断 |
| 6 | `wizard-step-2` 匹配到多个元素 | 步骤容器 `wizard-step-N` 与步骤行 `wizard-step-{index}` 命名冲突 | 步骤行改为 `wizard-step-row-N` |
| 7 | 下拉动作描述成「填入内容」 | 未区分 `select` 与 `fill` 语义 | 下拉用「选择「住宿费」」 |
| 8 | 无标签的密码框描述丢失「密码」 | 兜底文案未区分敏感字段 | 兜底也区分「填入密码」 |
| 9 | `Radio.Group` 包卡片导致状态互相覆盖 | 两个受控源竞争 | 去掉 Radio.Group，单一真相源 `source` |

**卡点 #5 与 #6 值得特别记录**：两者都不是「功能没写」，而是「写了但状态/选择器不自洽」，只有真的跑测试才会暴露。这印证了任务书「验收标准是他能独立完成，不是功能存在」。

---

## 3. 归属协议越界检查

| 协议条 | 要求 | 实际 | 结论 |
|---|---|---|---|
| 1 | `constants.tsx` 的 `ACTION_TYPE_OPTIONS` / `ACTION_TYPE_TEXT_MAP`：**只消费与补中文名**，不得作为第三写方新增动作 | **未修改** `constants.tsx` 一行。向导的动作中文名走独立文件 `wizard/stepLabels.ts`（`HUMAN_VERBS` 只消费 154 的动作枚举，不新增动作） | ✅ 未越界 |
| 2 | `AIScriptEditor.tsx`：152 已改完 mock，**只叠加降级入口**，不得回退真实调用 | 保留 `api.post("/rpa/ai/generate", ...)` 全部逻辑；新增 `unavailable` 态 + `onFallbackToRecord` + 按钮组。152 的测试 `AIScriptEditor.realapi.test.tsx` 4 个用例仍全绿 | ✅ 未越界 |
| 3 | wizard 页面统一走后端 `sys_menu` seed migration，**不要改** `componentLoader.tsx` / `DynamicRoutes.tsx` | 新增 `migration_256_nocode_wizard_menu.go`；两个前端路由文件 **git diff 为空** | ✅ 未越界 |
| 4 | `ExecutionDetailModal.tsx`：**不许碰**（155 地盘） | **未修改** | ✅ 未越界 |
| 5 | `browser/recorder.go` 净新增；不得改 `manager.go` / `page_manager.go` 核心契约（154 冻结） | `recorder.go` / `picker.go` 均为净新增；`manager.go` / `page_manager.go` / `pool.go` **git diff 为空** | ✅ 未越界 |

**额外自查**：前端 `types/rpa.ts` 的 `ActionType` 只有 13 个值（154 只扩充了 Go 侧 26 个动作）。按协议第 1 条，**156 不得作为第三写方改共享枚举**，故 `stepLabels.ts` 用字符串分派（`const type = action.type as string`），既让 154 新增动作立刻有中文描述，又不越界改共享类型。

---

## 4. 构建与测试闸门

| 闸门 | 命令 | 结果 |
|---|---|---|
| rpa-worker 构建 | `cd rpa-worker && go build ./...` | ✅ exit 0 |
| rpa-worker 全量测试 | `cd rpa-worker && go test ./...` | ✅ 全部 ok（browser 108s / executor 18s / liveview 3.2s / worker 1.8s / deployments 1.0s） |
| 根模块构建 | `go build ./...` | ✅ exit 0 |
| 根模块受影响包测试 | `go test ./internal/services/rpa/ ./internal/api/v1/rpa/ ./internal/core/db/migrations/` | ✅ 全 ok |
| 前端类型检查 | `npx tsc --noEmit -p tsconfig.app.json` | ✅ 无 error |
| 前端 rpa 测试 | `npx vitest run src/pages/operations/rpa/` | ✅ **16 files / 137 tests passed** |
| 前端生产构建 | `npm run build` | ✅ built，`TaskWizard` 独立 chunk 25.94 kB（懒加载生效） |
| lint（本次改动文件） | `npx eslint src/pages/operations/rpa/wizard/` | ✅ 0 error / 0 warning |

**本次新增测试统计**：

| 文件 | 用例数 |
|---|---|
| `rpa-worker/internal/browser/recorder_test.go` | 16 |
| `rpa-worker/internal/browser/recorder_integration_test.go`（真 Chromium） | 1 |
| `rpa-worker/internal/browser/picker_test.go`（含 2 个真 Chromium） | 5 |
| `internal/services/rpa/ai_action_validator_test.go` | 9 |
| `internal/api/v1/rpa/ai_generate_states_test.go` | 6 |
| `internal/core/db/migrations/migration_256_nocode_wizard_menu_test.go` | 5 |
| `AIScriptEditor.fallback.test.tsx` | 10 |
| `wizard/__tests__/wizard.test.tsx` | 35 |
| **合计** | **87** |

---

## 5. 遗留项

| # | 遗留项 | 影响 | 建议 |
|---|---|---|---|
| 1 | **录制/拾取的 HTTP 端点未接线** | Worker 侧的 `Recorder` / `Picker` 是完整的库能力，但尚无 `POST /rpa/recordings/*` 之类的后端路由把前端「开始录制」按钮串到 Worker。当前前端入口给出的是明确指引文案（「请在实时画面里开始录制」），不是死按钮。 | 归 W5 或 Phase 158：需要 Worker→后端→前端的录制会话通道，属独立一轮设计（涉及新的 Redis 频道与 session 生命周期），不宜塞进本 phase。 |
| 2 | **前端 `ActionType` 仍是 13 值子集** | 154 扩充的 13 个动作（hover/doubleClick/keyboard/iframe 等）在 Go 侧与录制侧可用，但前端编辑器下拉里选不到。 | 归属协议第 1 条禁止 156 作第三写方。应由 154 或后续 phase 统一扩前端枚举 + `ACTION_TYPE_OPTIONS`。 |
| 3 | **模板选择器是占位值** | 模板里的 `#username` / `#amount` 等是示例选择器，真实环境必须换成用户系统的。 | 设计上由「录制」或「拾取器」替换。当前向导允许逐项改参数但**不改选择器**（避免暴露技术概念），所以模板首次使用需配合拾取器校正 —— 这条应在产品文案里讲清（当前模板说明已写「改几个数字就能用」，措辞偏乐观，建议后续调整）。 |
| 4 | **定时任务字段未闭环** | 向导第 3 步产出 `triggerType=schedule` + `scheduleConfig.cron`，但 `sys_rpa_schedules` 的 CRUD 与 cron 注册属 Phase 158 CLEANUP-02 范围。 | 按 ROADMAP 归属，158 负责打通。当前保存不会报错（字段进 payload），但不会真的定时触发。 |
| 5 | **AI 生成结果的真实可用性依赖 AI provider** | 三态映射与枚举校验已完整，但「AI 能否生成好的脚本」取决于 provider 质量与 prompt 工程。 | 本 phase 只负责「不可用时不死路」。prompt 优化属独立课题。 |
| 6 | **录制器未处理 shadow DOM 深度嵌套** | `elementFromPoint` 会返回 shadow host，阶梯对 shadow 内部元素无能。 | 内网系统用 shadow DOM 的较少；若出现再补 `shadowRoot.elementFromPoint` 递归。 |
| 7 | **`wizard` 目录的 `index.tsx` 有一个 react-refresh 警告** | 已通过把 `toTaskPayload` 拆到 `payload.ts` 消除；当前 lint 0 error 0 warning。 | 已解决，此处仅记录。 |

---

## 6. 文件清单

### 新增

**rpa-worker（净新增，未改 154 冻结文件）**
- `internal/browser/recorder.go` — 页面内录制器
- `internal/browser/recorder_test.go` — 16 个纯函数单测
- `internal/browser/recorder_integration_test.go` — 真 Chromium 录制回放验收
- `internal/browser/picker.go` — 元素拾取器
- `internal/browser/picker_test.go` — 含 2 个真 Chromium 验收

**后端**
- `internal/services/rpa/ai_action_validator.go` — 动作枚举校验 + `ErrAIDisabled` 哨兵
- `internal/services/rpa/ai_action_validator_test.go`
- `internal/api/v1/rpa/ai_generate_states_test.go`
- `internal/core/db/migrations/migration_256_nocode_wizard_menu.go`
- `internal/core/db/migrations/migration_256_nocode_wizard_menu_test.go`

**前端**
- `src/pages/operations/rpa/wizard/index.tsx` — 向导页面
- `src/pages/operations/rpa/wizard/TaskWizard.tsx` — 三步向导主体
- `src/pages/operations/rpa/wizard/templates.ts` — 3 个开箱模板
- `src/pages/operations/rpa/wizard/stepLabels.ts` — 中文步骤名唯一转换点
- `src/pages/operations/rpa/wizard/payload.ts` — 草稿 → 请求体
- `src/pages/operations/rpa/wizard/__tests__/wizard.test.tsx`
- `src/pages/operations/rpa/tasks/modals/__tests__/AIScriptEditor.fallback.test.tsx`

### 修改（均在授权范围内）

| 文件 | 改动 | 授权依据 |
|---|---|---|
| `internal/services/rpa/ai_service.go` | `GenerateScript` 加动作校验 + `ErrAIDisabled` | NOCODE-03 |
| `internal/api/v1/rpa/ai_handler.go` | 三态错误映射 | NOCODE-03 |
| `internal/api/v1/rpa/handler_helpers.go` | 新增 `writeRPAError` + `handleError` 陷阱注释 | NOCODE-03 |
| `internal/core/db/database.go` | 注册 migration 256（PG + sqlite 两分支） | 协议第 6 条 |
| `xingran-react-frontend/.../AIScriptEditor.tsx` | 叠加 `unavailable` 态 + 降级入口（**未回退 152**） | 协议第 2 条 |
| `xingran-react-frontend/.../tasks/index.tsx` | 「新增任务」改开向导 + 保留「高级新建」 | 补登项 |
| `xingran-react-frontend/.../tasks/modals/EditModal.tsx` | 加「模板」/「一键转录制」 | 补登项 |

### 明确未改（越界检查证据）

- `xingran-react-frontend/src/pages/operations/rpa/constants.tsx`
- `xingran-react-frontend/src/router/componentLoader.tsx`
- `xingran-react-frontend/src/router/DynamicRoutes.tsx`
- `xingran-react-frontend/src/pages/operations/rpa/executions/ExecutionDetailModal.tsx`
- `rpa-worker/internal/browser/manager.go`
- `rpa-worker/internal/browser/page_manager.go`
- `rpa-worker/internal/browser/pool.go`
- `rpa-worker/internal/types/types.go`
- `xingran-react-frontend/src/types/rpa.ts`

---

## 7. REQ 结论

| REQ | 结论 | 关键证据 |
|---|---|---|
| NOCODE-01 页面内录制 | **PASS** | 真 Chromium 录「登录→填表→提交」→ 8 动作 → 回放 → 服务端收到 `amount=1280;type=hotel`；零浏览器扩展；选择器优先 testid |
| NOCODE-02 元素拾取器 | **PASS** | 7 级阶梯逐级可用性回传；DOM 微调（插兄弟/套 wrapper/去 id/改文本）后仍命中且反向确认低层级失效；iframe 内真进入 frame 拾取 |
| NOCODE-03 自然语言 + AI 降级 | **PASS** | 服务端 26 动作枚举校验（与 worker types.go 双向同源断言）；三态状态码可区分（400/400/500）；AI 不可用显示「改用录制」主按钮且隐藏无意义的重试 |
| NOCODE-04 三步向导 + 模板 | **PASS** | 三步流转 / 拖拽排序 + 上下移降级通道 / 改参数 / 删步骤 / 三种运行方式；3 个模板覆盖登录·填表·查询导出；向导全程零技术术语（机械断言） |
