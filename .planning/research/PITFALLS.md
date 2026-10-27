# Pitfalls Research — v1.23「前端契约全量收尾与遗留清偿」

**Domain:** 前端契约组件扩展 + 大规模组件迁移 + 后端遗留 debug 清偿 + lint 清偿 + 浏览器 UAT（RuoYi-Next v1.23）
**Researched:** 2026-10-27
**Confidence:** HIGH（前端各节基于实际读取的 `AppInput.tsx` / `AppSelect.tsx` / `eslint.config.js` / `usePagination.ts` 源码；后端 debug 节基于 `.planning/debug/` 7 份文档原文；个别断言标注 MEDIUM）
**Note:** 本文件此前为 v1.19 的研究存档，已随 v1.23 研究任务覆盖；v1.19 原文可从 git 历史找回。

**冻结契约基线（v1.22，摘自 `v1.23-seeds.md` §D，所有 pitfall 均以此为约束）:**
- 分页「共 N 条 / M 页」，total=0 不渲染分页 —— `usePagination` 唯一实现（新业务文件已被 eslint 禁止直接 import）
- ConfirmDelete / confirmDelete 唯一危险操作入口；剩余 8 处 Modal.confirm 合法点全部带行内理由注释，`scripts/exemption-reasons.json` + `check-exemption-reasons.mjs` 校验
- AppInput / AppSelect（allowClear/showSearch 默认值下沉）、PageTable / SearchForm / FormModal / RemoteSelect 契约；裸 `<Input>/<Select>` 受 eslint selector 拦截
- handleApiError 唯一错误出口；icon-only 按钮必须 aria-label
- statusHelpers / datetime 常量 / 裸 hex 拦截（hex-exemptions 登记制）

---

## Critical Pitfalls

### 领域 1：AppInput 扩展覆盖 TextArea

### Pitfall 1.1: 把 TextArea 当作「换个 props 的 Input」—— 成员表达式是契约规则的盲区

**What goes wrong:**
现有 `eslint.config.js:114` 注释明确写着「Input.TextArea|Search|Password|OTP 为成员表达式 JSX 名，**不在本 selector 命中面（有意不拦）**」。P65 的 selector 是 `JSXOpeningElement[name.name='Input']`，只能命中 `<Input .../>`，命中不了 `<Input.TextArea .../>`。如果做 AppTextArea 扩展时只改 AppInput.tsx 而不加成员表达式 selector，那么：
1. 74 处 `<Input.TextArea>` 依旧零约束、零迁移压力，"契约全量收尾"在这块静默落空；
2. 反过来，迁移后的代码混用 `AppTextArea` 与 `<Input.TextArea>`，eslint 不会报任何错，验收会被假绿。

**Why it happens:**
做迁移的人拿着 P65 已验证的 selector 模板复制，模板恰好不含成员表达式场景；成员表达式 JSX 的 selector 写法（`JSXMemberExpression` vs `JSXOpeningElement` 的 name 结构不同）不是直觉可写的。

**How to avoid:**
- 新 selector 必须写成成员表达式形态：在 no-restricted-syntax 数组里追加 `JSXMemberExpression[object.name='Input'][property.name='TextArea']` 一类 selector（成员表达式的 name 是 `JSXMemberExpression` 节点，不是带 `name.name` 的简单标识符），而不是复用 `name.name='Input'` 模板。
- 按 Phase 68 DEBT-01 的教训（`eslint.config.js:141-144` 注释）：**新 selector 追加进既有 no-restricted-syntax 数组，勿另起 config 块** —— flat config 同名规则键整体覆盖，另起块会让内网 IP + P65 四个 selector 静默失效。
- 同步检查 `no-restricted-imports`：`antd/Input` 子包 pattern 已拦子包 import，但 `import { Input } from "antd"` 后用 `Input.TextArea` 的侧门仍开着，需要评估是否在迁移期把 TextArea 纳入 import 拦截（分两步：先 selector 提示、后 error）。

**Warning signs:**
- 迁移阶段 lint 输出里 TextArea 相关命中数为 0（74 处存量至少应在新 selector 上线当天产生 74 条 error/warning）。
- 迁移 PR diff 里出现 `AppTextArea` 与 `<Input.TextArea>` 混用但 CI 全绿。

**Phase to address:** 契约扩展设计 phase（写 65-UI-SPEC 续篇时）定 selector；lint phase 上线规则；迁移 phase 收口时用 grep 计数对账（74 处清零或全部进豁免清单）。

---

### Pitfall 1.2: antd TextArea 的 allowClear ≠ Input 的 allowClear（showCount / maxLength / 受控三重耦合）

**What goes wrong:**
AppInput 现在只是 `allowClear=true` 透传（`AppInput.tsx:12`）。直接把同一套"默认 allowClear"复制到 TextArea 会踩 antd 特有的坑：
- **showCount + maxLength 耦合**：antd 的 TextArea 上 `maxLength` 会**截断输入值**（Input 的 maxLength 只限制原生输入、不截断粘贴/程序赋值），叠加 `showCount` 后计数口径是截断后的值。给存量带 showCount 的 TextArea 默认加 allowClear 时，清除行为和计数显示会一起变。
- **受控 vs 非受控**：TextArea 的 allowClear 清除触发的是 `onChange('')`；对受控组件若 onChange 里只处理"非空校验后的值"，清除瞬间会把 `undefined/''` 喂进业务逻辑（例如搜索表单把空串当有效条件发给后端）。
- **autosize + allowClear 的视觉跳动**：autosize 模式下清空后高度收缩，清 icon 与 count 角标在 6.x 的层叠位置有已知观感问题。
- **Form.Item 陷阱**：TextArea 与 Input 的 valuePropName 同为 `value`（这点是安全的），但 allowClear 引入的清除路径会让 `required` 规则的报错时机和 `normalize` 链行为改变 —— 这是 74 个现存表单批量引入默认值后最容易炸的点。

**Why it happens:**
开发者按 AppInput 的经验照抄「默认 allowClear = 无害」；Input 上确实近乎无害（纯 UI 增强），TextArea 上它和 count/length/受控逻辑是语义耦合的。

**How to avoid:**
- AppTextArea 不做"全量默认 allowClear"，改为**白名单式默认**：契约文档里明确哪些组合允许默认开（无 showCount、无自定义 onChange 校验），哪些必须显式传 `allowClear={false}`。
- 写一个 codemod 前置扫描：枚举 74 处用法里带 `showCount` / `maxLength` / `onChange` / 受控 `value=` 的子集，产出到文件级清单，逐处人工裁定，而不是一把梭默认值。
- 为 AppTextArea 增加 vitest 用例矩阵：受控 + 清除、showCount + 清除、maxLength 截断后清除、Form required + 清除。

**Warning signs:**
- 迁移后表单出现"点 X 清空后提交了空串/undefined"的 bug 报告；showCount 显示数与实际不符；TextArea 高度抖动的 UI 反馈。
- snapshot/截图 diff 中 TextArea 区域宽度或角标位置变化。

**Phase to address:** 契约扩展设计 phase（AppTextArea 契约 + 默认值白名单）；执行 phase 用 codemod 清单落地；UAT phase 抽查有 showCount 的表单页。

---

### 领域 2：~157 裸 `<Input>` + ~151 裸 `<Select>` 批量迁移契约组件

> 基线核对提示：里程碑上下文给的数（157/151/58 文件）与本次 grep 快照（52 个文件 import Input、65 个文件 import Select、26 处 TextArea 行）口径不同（次数 vs 文件数 vs 用法数）。**规划前先跑一次权威计数脚本落盘基线**，否则收口对账会用错分母 —— 这本身就是本节的第一个 pitfall。

### Pitfall 2.1: 迁移中途 eslint 豁免清单与真实代码状态脱节（锚点脚本体系被绕过）

**What goes wrong:**
`eslint.config.js` 有三层豁免结构，全部靠锚点注释维护：`__P65_EXEMPTION_FILES__`（:375）、`__P66_EXEMPTION_FILES__`（:315），清单由 `scripts/gen-p65-exemptions.mjs` / `gen-p66-exemptions.mjs` 生成，注释明言「P67 收敛后移除」。v1.23 的迁移正是"收口期"。常见错误：
- 迁移完一个文件后**手工**把它从 files 数组删掉，而不是重跑生成脚本 → 清单漂移、锚点注释失效、下次脚本重跑把已迁移文件又加回来或把未迁移文件漏掉。
- 反向错误：文件迁移了一半（Input 改了、Select 没改）就提交 → 文件还在豁免清单里所以 lint 绿，但文件里已经 import 了 AppInput + antd Input 两套 → 运行时双包行为 + 后续收口时以为该文件早已清偿。
- 最危险的：把某文件的 no-restricted-syntax 违规直接 `eslint-disable-next-line` 或把规则置 `off` 来"压绿"。`eslint.config.js:311-312` 已明文警告：**置 off 会连带关闭 P65 的 antd 子包拦截**，flat config 的 override 是整条规则覆盖。

**Why it happens:**
三层 override + 两个生成脚本 + 锚点注释的维护模型在 pressure 下记不住；开发者只看"我这条 lint 报错怎么消掉"。

**How to avoid:**
- 迁移 phase 立规矩：**每文件原子迁移**（一个 commit 内该文件的 Input/Select 全部转契约组件），迁完即重跑 `gen-p65-exemptions.mjs` / `gen-p66-exemptions.mjs`，diff 清单作为 commit 的一部分。
- 收口验收用计数对账而非 lint 通过：`grep` 裸 Input/Select/TextArea 计数 == 豁免清单剩余条目数，两个数必须同时归零。
- 在 `check-exemption-reasons.mjs` 同层加一个 `check-p65-p66-anchors.mjs`：校验锚点注释存在、清单内文件确实仍含裸组件、清单外文件不含（防双向漂移）。

**Warning signs:**
- lint 通过但 grep 仍能搜到 `<Input ` / `<Select `；豁免清单里出现已经不 import antd Input 的文件（死条目）。
- 迁移 PR 中出现 `eslint-disable` 相关注释；`eslint.config.js` 的 override 块数量在 milestone 内增长。

**Phase to address:** 迁移执行 phase 全程；lint 基线 phase 落对账脚本。

---

### Pitfall 2.2: AppSelect 的 showSearch 自动策略在迁移时"帮倒忙"

**What goes wrong:**
`AppSelect.tsx:18` 的契约是 `showSearch = showSearch ?? count >= 8`，且**无条件**设 `optionFilterProp="label"`（:19）。批量迁移把裸 `<Select>` 换成 AppSelect 后：
1. **options ≥ 8 的 Select 被静默打开搜索**。如果业务上不该搜（如状态枚举 10 个、固定审批人列表），用户在搜索框输入却过滤行为异常，观感是"坏了"。
2. **optionFilterProp="label" 对非 label 形态的 options 是破坏性的**：存量裸 Select 大量使用 children JSX `<Option>` 写法或 `fieldNames` 自定义、`labelInValue`、options 里只有 `value` 没有 `label` 字段的 —— 过滤 prop 指向不存在的字段，搜索表现为"输什么都过滤光"。
3. 依赖 antd 默认（children 模式按 children 文本过滤）的 Select 换成 AppSelect 后行为改变，且**无任何编译期/lint 报错**。

**Why it happens:**
契约默认值是按"P65 时点抽样的典型用法"设计的；151 处存量的形态分布比抽样基线宽（children 写法、fieldNames、labelInValue 都在）。

**How to avoid:**
- 迁移 codemod 里加形态分类器：`options` 数组形态 / children 形态 / fieldNames / labelInValue 四类分别标注；children 与 fieldNames 形态默认**不**套 optionFilterProp，或迁移时顺手改写成 options 形态。
- "不该搜索"的清单化：children 形态 + 数量 < 8 的保持默认；明确业务上禁搜的（状态、开关类）显式传 `showSearch={false}` 并注释。
- vitest 对 AppSelect 补 4 类形态的过滤行为快照测试，契约组件改默认值时有红灯。

**Warning signs:**
- 迁移后用户反馈"下拉框多了搜索框/搜索过滤不出来"。
- 迁移 diff 中 Select 的 options 写法被 codemod 大面积机械改写（说明形态分类器没起作用）。

**Phase to address:** 迁移执行 phase（codemod 设计阶段就要做形态分类）；UAT phase 抽查 ≥8 项的枚举下拉页。

---

### Pitfall 2.3: allowClear 下沉到全量存量 → `onChange(undefined)` 语义炸弹 + 展示型字段被意外清空

**What goes wrong:**
v1.22 审查基线是"400 个 Select 仅 30% allowClear"。契约下沉把覆盖从 30% 拉到 100%，等于**一次性给 ~300 个从未有过清除按钮的控件装上清除**：
- 点 X 后受控字段变 `undefined`。存量提交处理代码普遍假设"这个字段是 string"（本系统 `org_id` 必须是合法 UUID、`building_service.go` 的 `validateOrg()` 有 UUID 正则校验），`undefined` 传到后端变成参数校验 400，前端 handleApiError 弹的却是含混的通用错误 —— 排查成本极高。
- **disabled 之外还有展示型场景**：antd 的 allowClear 在 disabled 下不显示（安全），但"展示型 Select"（详情回显、代码值展示、UUID 选择器）不 disabled 也不该有清除 —— 用户能"看起来清空"一个本不该编辑的字段。
- 手抖清除后表单 required 校验只在 submit 时爆，用户不知道是点过 X。

**Why it happens:**
默认值下沉的正确性在 P65 新页面上验证过（新页面从零写，onChange 链路是新代码）；迁移是把这套默认值施加到**按"没有清除按钮"假设写的存量代码**上，假设被反向打破。

**How to avoid:**
- 迁移 codemod 对每处 Select/Input 标注上下文：是否 disabled / 是否在详情回显 / onChange 是否直接进 API 参数；可清除性当作**每处一个决策**，默认值只是起点。
- 把「清除后值为 undefined」写进 SearchForm/FormModal 契约：统一在封装层做 undefined→''（或 undefined→不提交该参数）的归一化，业务层不感知 —— 这是本系统 handleApiError/opsApi 契约精神的延续：语义在唯一出口处消化。
- 对 UUID 类字段（org_id、user_id、device_id）全局 grep 其 Select/RemoteSelect 用法，逐处确认清除后的提交防护。

**Warning signs:**
- 迁移后 400 参数错误类工单/日志上升（可在后端 operlog 或 request log 里做前后对比）。
- 用户反馈"字段莫名其妙被清了"。diff 中出现手动补 `?? ''` 的散点（说明归一化没下沉到封装层）。

**Phase to address:** 迁移执行 phase；FormModal/SearchForm 契约若在 v1.23 有改动，归一化必须在封装层 phase 一次做完。

---

### Pitfall 2.4: import 路径 churn 打断测试与 mock 体系 + snapshot 海啸

**What goes wrong:**
- 测试里 `vi.mock("antd")` 按 antd 组件名 mock 的用例，迁移后渲染的是 AppInput/AppSelect（真实组件包着 mock 失效的 antd Input）→ mock 断言失效、事件派发路径变化，测试红一片却找不到 diff 点。
- eslint 测试文件豁免块（`eslint.config.js:516-534`）把 no-restricted-syntax 整条 off，**测试文件里的裸 Input/Select 不会被 selector 发现** → 测试仍在测裸组件路径，业务代码已换契约组件，测试覆盖率出现假象。
- snapshot/HTML diff 类测试对 allowClear 图标、clear icon DOM 结构敏感，一次迁移产生数百条 snapshot 更新 → 团队养成"闭眼 update snapshot"习惯，真实回归被淹没。

**Why it happens:**
迁移以"lint 驱动"，测试文件的豁免使 lint 对测试侧零信号；snapshot 海啸让 diff 审查疲劳。

**How to avoid:**
- 迁移 phase 规定：业务文件迁移的同一 commit 内同步迁移/修正其测试文件的渲染目标（测试文件虽然 eslint 豁免，但要人工对账）。
- snapshot 更新必须逐文件 review，规则：**只允许新增 clear icon 相关 DOM 变化，其他 DOM 变化一律红灯人工看**。
- 为高频页面（user/workstations/workorder orders 等 PageTable 列表页）先建 vitest 交互冒烟（翻页、清空筛选、提交空表单）再迁移，迁移后冒烟必须绿。

**Warning signs:**
- 迁移分支上 snapshot 更新行数远超业务改动行数；测试通过数下降但没人报 bug（测试被静默 skip/mock 失效）。

**Phase to address:** 迁移执行 phase；每个 wave 的验证步骤固定含"测试同步迁移"。

---

### 领域 3：7 项后端 debug 修复（此前 acknowledge 结转）

### Pitfall 3.1: 把「acknowledge 结转」误读为「根因已定位」—— 7 份文档里近半数尚未完成调查

**What goes wrong:**
本次实际读取 7 份文档的 frontmatter status：
- `linux-log-missing-2026-07-09.md` — status: **diagnosing**，next_action 是"请用户在 Linux 服务器执行诊断命令收集信息"（Evidence 只到两路日志分流的观察，根因未定位）
- `mac-history-mv-double-execution.md` — status: **investigating**，Evidence / Eliminated / Resolution 三节全是「等待 gsd-debugger 填充」，即**双重执行根因从未找到**
- `mac-history-query-returns-empty.md` — status: **investigating**，五个待验证 SQL 没有执行记录
- `scrapligo-getprompt-nil-panic.md` — status: **awaiting_human_verify**，仅一次崩溃、无手动复现路径，且文档发现**版本漂移**：项目记录 scrapligo v1.3.3，stack trace 是 **v1.4.0**
- 仅 `mac-collection-cisco-no-vendor-support`（diagnosed，根因明确：DeviceVendor 枚举 4 值无 default 分支静默丢弃）、`mac-format-recurrence-R2`（investigated，已量化脏数据）、`workstation-3f130`（investigated，H1-H4 假设已列好等用户 SQL）算有实质调查。

直接按种子清单"领任务→写修复"会造出**修复不存在根因的代码**：比如给 MV 刷新 job 包一层去重锁（掩盖 scheduler 双重注册的真相）、给日志加 flush 兜底（而真实原因可能是磁盘/journald 配置）。

**Why it happens:**
"prior investigation doc 存在" 被当成 "investigation 完成"；GSD 收档时 acknowledge 的语义是"承认未完、结转"，不是"已诊断待修"。

**How to avoid:**
- v1.23 排 phase 时**第一波是重验 spike，不是修复 wave**：逐文档核对 status 字段，把 7 项分成「根因已定位待修」（≤3 项）/「调查中断需续查」（≥3 项）/「需外部输入」（linux-log 需用户远程命令输出、workstation-3f130 需用户 SQL）三类，修复任务只对第一类下发。
- 每份文档修复前强制重跑「复现」：linux-log 与 scrapligo panic **没有复现路径**，修复必须在能复现或至少能构造最小等价场景（scrapligo：FileTransport e2e 已有基建可模拟 transport nil 窗口）后才允许写修复代码。
- 版本漂移先校准：scrapligo 修复前先 `git log go.mod` 确认 v1.3.3→v1.4.0 升级时间线，panic 可能是升级引入且新版本已修，直接 patch 自己代码可能白做。

**Warning signs:**
- 修复 PR 的描述里没有复现步骤/复现脚本链接；debug 文档的 Resolution 节在修复合并时仍是空的（正确流程是修复回填文档再关闭）。
- 同一症状在修复后复发（本仓库已有先例：mac-format 问题在 R2 复发）。

**Phase to address:** 重验/复现 spike phase（v1.23 首个后端 phase）；修复 wave 只消费 spike 的输出。

---

### Pitfall 3.2: MAC 格式类修复只清数据/只修一处写入 —— 本仓库已实证会 R2 复发

**What goes wrong:**
`mac-format-recurrence-R2.md` 的证据链是教科书级的：所有 525 条坏数据精确产生于 HH:00:01 cron tick，`mac_address` 是 Cisco dot 格式（`9c7b.ef2f.0222`）、`interface_name` 带**空格**（`FastEthernet 1/0`）—— **这个格式不可能由 normalize.MACAddress / normalize.InterfaceName 产生**，即存在**绕过 normalize 层的写入路径**（某条 cron/采集代码直写 DB）。v1.23 若再修：只清洗数据 → 下一个 tick 又脏；只给已知 normalize 调用点加测试 → 绕路写入点继续裸写；不查 `sys_job_log` 时间对齐 → 根因继续隐藏。
同理 Cisco 支持修复的文档明确指出 `parseMACLine` 的 vendor switch **没有 default 分支，静默丢弃** —— "静默"意味着修完 Cisco 后可能还有未知厂商/格式在静默丢，验收只有那一个 MAC 用例的话测不出。

**Why it happens:**
"修好报告的那个症状"惯性；绕路写入点是代码考古问题，比改一行 normalize 贵，压力下被跳过。

**How to avoid:**
- 修复模式定为「**先枚举全部写入方，再修，最后在写入方层加回归测试**」：grep 所有对 `sys_device_mac_address` / `sys_device_mac_history` 的 INSERT/Save/BatchUpsert 调用点，逐点确认过 normalize。
- 写入方回归测试落在写路径的包里（`go test ./internal/services/...`），断言「任何 vendor/任何原始格式进 normalize 后必为 `^[A-F0-9]{2}(:[A-F0-9]{2}){5}$`、interface 无空格」—— 防 R3。
- vendor switch 强制 default 分支返回**显式错误并计数**（进 job log / operlog），把"静默丢弃"改成"可观测失败"；这也是本系统 operlog 可追溯价值观的延伸。
- 脏数据清洗脚本必须幂等可重跑（本仓库 migration 有双执行前科，见 Pitfall 3.3），并先在 R2 量化口径上验证清洗前后行数。

**Warning signs:**
- 修复上线后下一个 HH:00 tick 再次出现坏格式（时间对齐 grep 是最强信号）。
- `parseMACLine` 新 vendor 分支没有对应 TextFSM 模板和 e2e fixture（v1.19/v1.20 的 3-vendor fixture 模式该复用）。

**Phase to address:** 修复 wave（写入方枚举作为 plan 的第一个 task）；回归测试作为该 plan 的验收项而非可选项。

---

### Pitfall 3.3: 迁移/清洗脚本双执行修复 —— 修复手段本身可能重蹈「双执行」

**What goes wrong:**
`mac-history-mv-double-execution.md` 的症状是同一 job 每 5 分钟两条 sys_job_log 且**双双成功** —— 高度指向双重注册（sys_job 双行 / scheduler 启动期双 goroutine），但根因节为空。这个仓库还有相邻前科：`recoverPendingTasks` 清理 running 僵尸任务曾暴露、v1.21 有 cron 双写类排查史。修复该问题时容易犯：
- 在 job handler 里加互斥锁/幂等 guard 当作"修复" → 双注册的第二个 goroutine 还在空转，5 分钟一次的无效刷新继续消耗 DB，只是不再可见。
- 反向：给 MAC history 清洗/迁移脚本（v1.23 要写的数据清洗）不加执行 guard → 复刻同一类双执行事故。
- MAC history 是 **PG 分区表**（`sys_device_mac_history_2026_06` 子分区，见 query-empty 文档），清洗脚本若按根表直写/直删，分区路由与 `deleted_at` 缺失（model 文件缺 `deleted_at` 且 Raw SQL 无过滤，query-empty 文档原话）都会让"清洗成功"是假象。

**Why it happens:**
双执行是并发/注册时序问题，单测全绿、生产才炸；分区表 + 缺 deleted_at 是 schema 级事实，应用层修复看不到。

**How to avoid:**
- 双执行的修复入口只允许在**注册处**（scheduler 启动代码 + sys_job 表唯一性），先跑文档已列出的三步验证（搜注册点 / 查 sys_job 重复行 / 查独立 refresh goroutine）；handler 层 guard 只能作为纵深防御并注释说明。
- 数据清洗脚本带 job log 唯一性断言：执行前 SELECT 当 tick 是否已有同 job 记录；清洗 SQL 走分区感知写法并在文档中记录目标分区名。
- 对 `deleted_at` 缺失问题：任何查询/清洗改动都要同步决定「是否补 deleted_at 语义」——半吊子加过滤会把存量数据"变没"（query-empty 的候选根因之一就是过滤语义），改动前先跑文档里那 5 条 SQL 确认数据基线。

**Warning signs:**
- sys_job_log 同 tick 双条在修复后仍出现（只是内容变了）→ 修复没打到注册层。
- 清洗后前端轨迹查询返回行数不升反降 → deleted_at/分区语义被误伤。

**Phase to address:** 调查续接（Pitfall 3.1 的 spike）先定性；修复 wave 中把「sys_job_log 双条消失」列为该修复的验收标准。

---

### Pitfall 3.4: workstation 物理链路 / linux 日志类修复在缺少现场数据时"静态推演成稿"

**What goes wrong:**
`workstation-3f130-physical-link-empty.md` 已把 H1-H4 四个互斥假设和验证 SQL 列好（信息点绑定错位 / workstation 软删 / user_id 空早退 / 设备软删），且明确「5F003 修复（commit 6575a1e7）已在 HEAD，可排除同源」。风险是执行 agent 拿到任务后直接挑一个假设开修（通常挑改动最小的 user_id 早退分支），而真实原因可能是 H2 软删 —— 修错分支后 3f130 好了、别的工位坏了。`linux-log-missing` 同理：文档明确排除过 3 个假设（MaxSize/目录/level），剩下必须靠**用户在 Linux 服务器执行诊断命令**的输出，本地无法复现。

**Why it happens:**
远程/现场依赖阻塞感强，静态分析总能"产出点什么"；GSD 执行 phase 有完成压力。

**How to avoid:**
- 这类任务在 phase 计划里显式标注 `blocked_on: user_input`，第一步就是产出"给用户的 SQL/命令清单"（3f130 的 4 条 SQL 文档里已备好，直接引用），**禁止**在用户回填数据前写修复代码。
- 用户数据回填后走假设排除流程并回填文档 Eliminated 节（保持该 debug 文档既有格式），再决定修哪个分支。
- linux-log 的修复若最终落在 logger 初始化（journald vs 文件双路分流是文档已确认的事实），改动范围锁定 `pkg/logger/logger.go` + `cmd/main.go`，并补一条「启动即写文件」的自检日志。

**Warning signs:**
- 修复 commit 出现在用户诊断数据回填之前；debug 文档的 Eliminated 节没有新增条目却有 Resolution。

**Phase to address:** spike/信息收集 phase 产出用户清单；修复 wave 严格 gate 在回填之后。

---

### 领域 4：lint warnings 从 1367 基线清偿

### Pitfall 4.1: --fix 批量清偿掩盖真实 bug + 动摇 83 errors 零新增防线

**What goes wrong:**
`eslint.config.js:78-80` 明确记录：`no-unused-vars`（~255 处）与 `no-explicit-any`（~75 处）是"存量噪音，降为 warn 不阻塞 CI，后续逐步清理"。清偿时的典型事故：
- **--fix 删 unused-vars 删掉的是"文档性死代码"**：未使用的接口字段、占位参数记录着 API 形态（本仓库有 `transfer-props-type-mismatch` 的 debug 前科），删了之后下一个人重构时失去参照。
- **no-explicit-any 的 --fix（`any → unknown`）是行为变更**：unknown 收紧后下游类型报错蔓延，type-check 从绿变红，修复方式变成再撒 `as any` —— warnings 变少、any 总数不变或更多。
- **批量跑 --fix 会顺带触发 prettier/quotes 重排**，混进功能 diff；更糟的是个别 fixer 把 warn 修成新 error 类违规，破坏 v1.22 锁定的 **83 errors 零新增**基线 —— 这条是本仓库的明确守卫线。
- 规则降级边界：为了压数字把某条 warn 静默 `off`，等于在 v1.22 建立的"零新增 error"同一防线上再挖洞。

**Why it happens:**
1367 是大数字，有"一次跑完 --fix 提交"的诱惑；warning 清偿天然低优先级，评审深度不足。

**How to avoid:**
- **按规则分组清偿，禁止全量 --fix**：优先机械安全规则（prefer-const、no-var 类），no-unused-vars 逐文件人工确认（删除前 grep 该符号是否出现在字符串模板/动态引用），no-explicit-any **不定为清偿目标**（保持 warn，若必须动则走类型化改造而非 unknown 逃逸）。
- 每个清偿 PR 的 CI 断言三件事：errors ≤ 83、warnings 严格下降、`npm run type-check` 绿。把「83」写成脚本断言（对比 `npm run lint` 输出计数），不靠人眼。
- 基线快照落在 phase 开始时（.planning 下存 lint-baseline.json），收口时输出 delta 报告；规则增删必须走 PR 评审并更新 `eslint.config.js` 注释（该文件注释文化很重，沿用）。

**Warning signs:**
- 单个清偿 PR 的 diff > 50 文件；type-check 在 lint-only 改动后变红；`as any`/`@ts-ignore` 数量在清偿后上升（grep 可量化）。
- errors 计数在 warnings 下降的同时上升 —— 说明 fixer 在改写中制造新违规。

**Phase to address:** lint 清偿 phase（建议排在契约迁移 phase 之后：迁移会产生新的 warn/error 峰值，先迁移再清偿一次到位）；CI 断言脚本在 phase 第一个 task 落地。

---

### 领域 5：浏览器实机 UAT（分页 / 批量删除）

### Pitfall 5.1: 重演「jsdom 全绿、浏览器翻车」—— 恰是本次要清偿的债的成因

**What goes wrong:**
`v1.23-seeds.md §B` 明言：Phase 68.1 的分页交互、批量删除流程「仅由 jsdom 测试覆盖，未做浏览器实机验证」，这正是结转到 v1.23 的 UAT 债。v1.23 再做 UAT 时的典型重复犯错方式：
- **jsdom 覆盖不到的恰好是这三类**：antd Table 分页的真实滚动/焦点/事件冒泡；ConfirmDelete 的真实 Modal 层级与 double-click 竞态（v1.22 P0 基线：27 个业务弹窗 confirmLoading 0 命中，双击重复提交史说明这类竞态真实存在）；usePagination 的 **sessionStorage 按 location.pathname 隔离**（`usePagination.ts:63-76`）—— jsdom MemoryRouter 的 path 切换不代表真实多 tab 场景。
- UAT 跑在 dev build（vite dev）而非 `npm run build` 产物 —— vendor chunk 的懒加载/TDZ 类问题（本仓库有 `react-vendor-activity-tdz`、`css-tree-vendor-chunk-tdz` 前科）只在产物里炸。
- 环境错配：本系统登录链路有 SM2/SM4 请求加密 + 双 token 刷新（`page-refresh-token-refresh-loop-failure`、`file-upload-401-refresh` 前科），UAT 若在加密关闭的环境跑，真实用户路径（401→refresh→重放）测不到；批量删除这类写操作还要走 handleApiError 出口，错误场景必须真实触发一次。
- **Windows dev 环境测完就收** —— 本 milestone 同时在修「Linux 日志缺失」后端项，前后端都要有 Linux 侧至少一次冒烟，否则又制造一笔"只在 Windows 验证过"的隐性债。

**Why it happens:**
jsdom 测试便宜且全绿给人安全感；UAT 缺脚本和记录格式时被压缩成"点了几下没问题"。

**How to avoid:**
- 按 `v1.23-seeds.md §B` 指定的 Phase 63 冒烟模板建 UAT 脚本：登录（加密开启的 .env）→ 目标页 → 翻页/改 pageSize/刷新恢复页码 → 勾选/跨页勾选语义确认 → 批量删除（ConfirmDelete 二次确认 + confirmLoading 期间按钮禁用）→ 取消路径。每步记录到 `*-HUMAN-UAT.md`（沿用 v1.19/v1.20 的 HUMAN-UAT 文档惯例）。
- UAT 前置清单：prod build 预览（`npm run build && npm run preview`）、`VITE_ENABLE_REQUEST_ENCRYPTION` 按生产配置、清一次 sessionStorage 验证 usePagination 持久化恢复。
- 批量删除 UAT 必须覆盖 **exemption-reasons.json 契约**的现实面：ConfirmDelete 是唯一入口，UAT 记录里写明验证的是 ConfirmDelete 路径而非某页面自带的 Modal（防有人 UAT 时点的是遗留 Modal.confirm 合法点）。
- 明确 UAT 产物归属：结果回写 68.1 UI-SPEC §8 回归清单第 15/16 条，关闭时引用证据截图，不让债再次无主结转。

**Warning signs:**
- UAT 记录里没有 build 产物版本/commit hash；没有截图或录屏证据；"分页正常"一句话带过没列 pageSize 恢复、跨页勾选等具体场景。
- UAT 只在 http://localhost:4000（dev server）执行。

**Phase to address:** UAT phase（v1.23 收口段）；UAT 脚本与前置清单在 phase 计划里先行落盘，避免临场即兴。

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| 迁移文件从豁免清单手工删除而非重跑生成脚本 | 快 2 分钟 | 清单漂移，锚点脚本体系失效，下次重跑引入假 error | Never |
| 对 AppTextArea 全量默认 allowClear（照抄 AppInput） | 契约"统一" | showCount/maxLength/受控耦合炸 74 个现存表单 | Never（必须白名单） |
| job handler 加幂等 guard 当作双执行修复 | 当天 sys_job_log 变干净 | 双注册 goroutine 空转消耗 DB，真实根因永闭 | Never（guard 只作纵深） |
| lint 清偿用全量 --fix 一把梭 | warnings 一次砍半 | unused-vars 删掉 API 形态文档、any→unknown 类型蔓延、83 errors 线被 fixer 击穿 | Never；按规则分组 + 人工确认 |
| 修 debug 项时不回填 debug 文档 Resolution | 节省 10 分钟 | 下次复发（R3）无据可查，acknowledge 结转再滚动一轮 | Never |
| UAT 在 dev server + 加密关闭环境跑 | 快、不用 build | vendor chunk/加密链路类回归全部漏测，债再次结转 | Never |
| children 形态 Select 迁移时不做形态分类直接套 AppSelect | codemod 覆盖率好看 | optionFilterProp 指向不存在字段，搜索过滤整体失灵 | Never；children 形态需单独决策 |

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| eslint flat config override 链 | 为新 selector 另起 config 块 | 追加进既有 no-restricted-syntax 数组；记住"同一 rules 对象内同名键互相覆盖"（文件内三处注释反复强调） |
| 豁免登记制（exemption-reasons.json / hex-exemptions / P65-P66 清单） | 直接 eslint-disable 或 off | 走登记制：脚本生成 + 理由注释 + 校验脚本通过 |
| antd 6.1 TextArea | 按 Input 的心智模型套 allowClear | 先审计 showCount/maxLength/autosize/受控四象限，白名单默认 |
| PG 分区表 sys_device_mac_history | 按普通表写清洗/查询 SQL | 分区感知 + deleted_at 语义决策前置（5 条基线 SQL 先跑） |
| scrapligo 版本 | 按文档记录的 v1.3.3 分析 v1.4.0 的 panic | 先校准 go.mod 实际版本与升级时间线，确认是否已在新版修复 |
| scheduler（internal/scheduler 引擎 vs api/v1/scheduler CRUD） | 修双执行时改错层 | 双执行只修注册层（引擎侧），CRUD 层不动 |

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| 迁移期间页面同时 import 契约组件与裸组件（半迁移态） | bundle 出现双份 antd Input 相关模块、行为不一致 | 每文件原子迁移（一个 commit 收干净一个文件） | 迁移中期的每个 PR |
| MAC 采集写入点绕过 normalize 的静默丢弃 | 脏数据按 HH:00 tick 持续再生 | vendor switch default 分支显式报错 + 计数可观测 | 每个采集 tick |
| sys_job 双注册的第二个 goroutine | 每 tick 双份 DB 刷新负载 | 修注册层而非 handler guard | 立即（已在发生，每 5 分钟） |
| 全量默认 showSearch（≥8 options）在长列表页 | 每个下拉渲染过滤索引，且 label 缺失时全量过滤 | 形态分类器 + 业务禁搜清单 | options 数大的页面 |

## "Looks Done But Isn't" Checklist

- [ ] **AppTextArea 契约**：74 处 TextArea 的 grep 计数是否真的清零/进清单 —— lint 绿 ≠ 覆盖（成员表达式盲区）
- [ ] **AppTextArea 契约**：showCount/maxLength/受控子集是否逐处裁定过 —— 默认值不是免审通行证
- [ ] **Input/Select 迁移**：豁免清单剩余条目数 == grep 裸组件计数，两个数同时归零
- [ ] **Input/Select 迁移**：测试文件同步迁移了吗 —— eslint 对测试文件豁免 no-restricted-syntax，lint 对测试侧零信号
- [ ] **7 项 debug**：每份文档 status 字段与 Resolution 节是否与修复状态一致 —— acknowledge ≠ 已定位（7 份中 ≥3 份证据节为空）
- [ ] **scrapligo 修复**：go.mod 实际版本核对过吗 —— 文档记录 v1.3.3、stack trace v1.4.0
- [ ] **mac-format 修复**：全部写入方枚举了吗 + 写入路径回归测试存在吗 —— R2 复发证明单点修复无效
- [ ] **lint 清偿**：errors ≤ 83 是脚本断言还是人眼 —— type-check 是否同跑
- [ ] **浏览器 UAT**：prod build + 加密开启 + sessionStorage 清空三项前置是否满足；结果是否回写 68.1 UI-SPEC §8 第 15/16 条

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| 豁免清单漂移/锚点失效 | LOW | 重跑 gen-p65/p66-exemptions.mjs → diff 对账 → 补锚点校验脚本 |
| TextArea 默认 allowClear 引发表单 bug | MEDIUM | AppTextArea 默认值回退为 false → 按 74 处清单重新白名单 → 定位受影响表单回归 |
| onChange(undefined) 造成的 400 工单潮 | MEDIUM | 封装层（FormModal/SearchForm）加 undefined 归一化 → 全局 grep UUID 类字段补防护 → 后端日志确认清零 |
| lint fixer 击穿 83 errors 线 | LOW | git revert 该清偿 commit → 按规则重分组 → 加 errors 计数 CI 断言后重跑 |
| 修错 debug 假设分支（如 3f130） | MEDIUM | debug 文档 Eliminated 回填排除依据 → revert 修复 → 按用户回填 SQL 重新定性 |
| 双执行修复只打了 handler 层 | LOW | 回到注册层定位（文档三步验证）→ 修复 → sys_job_log 双条消失作为验收 |
| UAT 又在 dev 环境跑完 | LOW | 补 prod build UAT 轮（成本低，但必须在收档前）—— 明确记入 HUMAN-UAT 文档 |

## Pitfall-to-Phase Mapping

> v1.23 路线图 phase 尚未定稿，此处用逻辑 phase 名；`/gsd-plan-phase` 时对号入座。

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| 1.1 TextArea selector 盲区 | 契约扩展设计 phase | 新 selector 上线日产生 ≥74 条存量命中（计数可 grep） |
| 1.2 TextArea allowClear 语义耦合 | 契约扩展设计 + 执行 | 4 象限 vitest 矩阵全绿；showCount 表单页人工抽查 |
| 2.1 豁免清单脱节 | 迁移执行全程 | `check-p65-p66-anchors.mjs` 过；清单条目数 == grep 计数且归零 |
| 2.2 showSearch 自动策略误伤 | 迁移执行（codemod 设计） | 4 类 options 形态快照测试；禁搜清单落地 |
| 2.3 allowClear onChange(undefined) 炸弹 | 迁移执行 + 封装层契约 | UUID 字段防护 grep；后端 400 类日志前后对比 |
| 2.4 测试 mock/snapshot 海啸 | 迁移执行每个 wave | snapshot 变更仅限 clear icon DOM；测试同步迁移 checklist |
| 3.1 acknowledge ≠ 已定位 | 重验 spike（首个后端 phase） | 7 项按 status 三分类清单落盘；修复任务只对"根因已定位"类下发 |
| 3.2 MAC 格式 R3 风险 | 修复 wave | 全写入方枚举文档 + 写路径回归测试 + default 分支可观测 |
| 3.3 双执行修错层 | 调查续接 → 修复 wave | sys_job_log 同 tick 双条消失（验收 SQL） |
| 3.4 缺现场数据静态推演 | spike / 信息收集 | 修复 commit 晚于用户数据回填；Eliminated 节有新增 |
| 4.1 lint --fix 掩盖 bug | lint 清偿 phase（排迁移后） | errors ≤83 脚本断言 + type-check 绿 + warnings 严格下降 |
| 5.1 jsdom-vs-真实浏览器 | UAT phase（收口段） | prod build + 加密开启 + sessionStorage 清空前置清单；68.1 §8 15/16 条带证据关闭 |

## Sources

- `.planning/PROJECT.md`（v1.22 契约组件整治全程记录、P0/P1 基线、冻结决策）
- `.planning/v1.23-seeds.md`（§A 7 项 debug 结转、§B UAT 债、§D 冻结契约备忘）
- `ruoyi-react-frontend/src/components/ui/AppInput.tsx` / `AppSelect.tsx`（契约实现原文）
- `ruoyi-react-frontend/eslint.config.js`（P65/P66/P68 三层 override、锚点注释、成员表达式盲区声明、83 errors 背景）
- `ruoyi-react-frontend/src/hooks/usePagination.ts`（sessionStorage 持久化实现）
- `.planning/debug/` 7 份文档原文（linux-log-missing-2026-07-09 / mac-collection-cisco-no-vendor-support / mac-format-recurrence-R2 / mac-history-mv-double-execution / mac-history-query-returns-empty / scrapligo-getprompt-nil-panic / workstation-3f130-physical-link-empty）
- 本次 grep 快照：TextArea 26 行用法（0 处 allowClear）、52 文件 import Input、65 文件 import Select（与里程碑口径 157/151/58 存在差异，已列为基线核对 pitfall）
- 历史前科引用：mac-format R2 复发、Phase 68.1 jsdom UAT 债、react-vendor-activity-tdz、page-refresh-token-refresh-loop、transfer-props-type-mismatch

---
*Pitfalls research for: RuoYi-Next v1.23 前端契约全量收尾与遗留清偿*
*Researched: 2026-10-27*
