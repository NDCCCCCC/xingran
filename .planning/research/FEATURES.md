# Feature Research — v1.23「前端契约全量收尾与遗留清偿」

**Domain:** 前端表单契约收尾 + 后端 debug 清偿（企业运维管理系统，Go + React 19.2 + Ant Design 6.6.4）
**Researched:** 2026-10-27
**Confidence:** HIGH（全部结论基于本地代码实测 / 7 份 debug 文档逐条核读；antd 行为结论基于本仓库实际安装的 antd 6.6.4 + @rc-component/input 源码与编译产物，非文档转述）

> 数据校准说明（与 v1.23-seeds.md / 里程碑上下文的差异，规划时应以本节数字为准）：
> - seeds 写「74 usages TextArea」——实测 **JSX 层 74 处**（`<Input.TextArea` 26 + 解构后 `<TextArea` 48），分布在约 23 个文件（pages/ 18 处 + components/ 5 处）。
> - seeds 写「157/151 bare Input/Select」——`pages/` 下实测裸 `<Input` 约 188 行、`<Select` 约 271 行（grep 行数，含多行 JSX 拆分误差；真实节点数以 eslint selector 命中数为准）。数量级一致。
> - seeds 写「UAT §8 第 15/16 条（分页 + 批量删除）」——**实际 §8 第 15/16 条是「固定列不失效」「行操作权限不变」**（68.1-UI-SPEC.md:315-316）；分页口径是第 13 条、批量删除是第 5/6/7 条。浏览器 UAT 应补 §8 全表 19 条中依赖真实渲染的条目（重点 13/15/16 + 2/3/7）。
> - lint 基线实测：**errors 83 / warnings 1367**，与 seeds 完全吻合。

## Feature Landscape

### Table Stakes（本里程碑「不做完就不算收尾」的部分）

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| F1 TextArea 一键清空（allowClear 契约下沉） | 用户直接报告的痛点：备注/描述/原因类多行输入 74 处无清空按钮，只能手动逐字删除 | **LOW** | 见 §F1 详析。唯一正确实现路径 = 契约层下沉（新 AppTextArea 默认 `allowClear`），禁止逐页补 prop——与 v1.22 已锁定决策「默认值必须下沉封装层」（PROJECT.md:30）完全同构 |
| F2 COMP-05 裸组件迁移收尾 | v1.22 契约体系（AppInput/AppSelect + eslint selector + 子包 import 拦截）已上线但存量豁免未清零，「有契约没人守」= 契约失效 | **MEDIUM（量大但单点简单）** | 见 §F2 详析。机械替换为主，风险集中在 68.1 §3.3「迁移不得改变行为」红线（查询参数一一对应、排序映射不变） |
| F3 7 项 debug 清偿 | 每项均已有 debug 文档（部分根因已定位到文件:行号），v1.22 收档时显式 acknowledge 到 v1.23 | LOW～HIGH（按项差异大，见 §F3 逐项表） | 7 项中 1 项零代码（#6 已修复等验证）、2 项差一次 DB 实测（#5/#7）、1 项方案就绪待执行（#2）、其余为中间态 |
| F4 浏览器 UAT（68.1 §8 遗留） | 分页/批删此前仅 jsdom 覆盖；jsdom 测不出固定列、真实滚动、权限渲染 | LOW | 按 Phase 63 冒烟流程（登录 → 目标页 → 翻页/勾选/取消）；seeds 中「15/16」编号需校正为 §8 原文条目 |
| F5 lint warnings 清偿 1367 → 目标 TBD | 流程性积压（seeds §C）；errors 已锁定 83 零新增 | LOW～MEDIUM | 分布已实测：**no-explicit-any 958（70%）+ no-unused-vars 363（27%）+ react-refresh 33 + no-console 5 + 规则null 8**。见 §F5 分类处置 |

### Differentiators（做了明显提升，但不做不阻塞收尾）

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| D1 AppTextArea 独立契约组件（而非仅 AppInput 透传） | TextArea 有 autoSize/showCount 特有语义，独立封装可未来下沉 maxLength 约定、统一 status/校验文案 | LOW | 推荐：新建 `AppTextArea`（默认 allowClear）比给 AppInput 挂 `.TextArea` 静态属性更符合 ui/ 目录组织（AppInput.tsx 仅 16 行纯透传，见 `src/components/ui/AppInput.tsx:12`） |
| D2 lint 预算门禁（--max-warnings 进 CI） | 「warnings 只降不升」比一次性清零更可维护 | MEDIUM | F5 收官动作：`--max-warnings N` 进 npm script，超预算 CI 失败 |
| D3 DB 层 MAC 归一化 TRIGGER | mac-format-recurrence-R2 文档明确记载 M196 TRIGGER「RECOMMENDED but NEVER IMPLEMENTED」；DB 兜底可终结「格式脏数据复发→排查→复发」循环 | MEDIUM | ⚠️ 注意 v1.19 研究曾记录用户偏好「根因修复走代码层，禁用 DB TRIGGER 路线」（旧 FEATURES.md:16 引 user-prefers-code-fixes-no-db-triggers）——**实施前须与用户确认是否为此场景破例**（R2 是复发型数据污染，defense-in-depth 论据更强） |
| D4 MAC 轨迹查询错误传播修复 | mac-history-query-returns-empty 文档 (c) 项：Raw SQL 吞 error 返回空数组——即使本次根因是数据问题，「吞错误」本身是独立缺陷，修掉可让未来此类问题 10 分钟定位 | LOW | 改 GORM chainable + handler 返回 500；与 F3#5 联动 |

### Anti-Features（看似合理、应明确排除）

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| A1 逐页给 74 处 TextArea 手动加 `allowClear` | 改动局部、每处只加一个词 | 直接违反 v1.22 锁定决策（「禁止逐页补 allowClear——横跨 73 文件不可维护」）；新增文件会再次遗漏 | F1 契约层下沉 + eslint selector 拦截裸 TextArea |
| A2 TextArea 清空做成「focus 时常显 ✕」自定义浮层 | 想要更强可见性 | 与 antd 内建交互模型冲突（见 §F1 行为详析），双清空入口打架；自定义浮层要处理 autoSize/showCount/IME composition 全部边界 | 直接用 antd 内建 allowClear，与单行 AppInput 行为同源，零学习成本 |
| A3 devices 页 692 行拆分 | 代码整洁冲动 | seeds §B 已记为设计决策：「拆分风险大于收益」 | 维持不拆；若拆追加 ~2h 单独排期 |
| A4 no-explicit-any 一次性清零（958 处） | 数字好看 | 大量 any 在 chart/ECharts widget 类型层，强行断言引入运行时错误；958 处一次改无法审查 | F5 分层：机械可修的先修，类型边界处进白名单 + D2 预算门禁 |
| A5 lint 清零后立即把所有 warn 规则翻 error | 锁定收益 | react-refresh 等 33 处与页面结构耦合，翻 error 代价不成比例；未清零前翻 error 阻塞开发 | 预算制逐步收紧 |

## §F1 详析：TextArea 一键清空

### antd TextArea allowClear 行为实测（基于本仓库 antd 6.6.4 安装产物）

- **可用性**：`Input.TextArea` 原生支持 `allowClear`（antd ≥5.15 起正式支持 TextArea 的 allowClear 与 showCount 共存；本仓库 antd 6.6.4 完整支持，无需 polyfill）。
- **交互模型**（读 `@rc-component/input/es/TextArea.js` + `BaseInput.js` + antd dist CSS 得出）：
  - TextArea 走 **affix-wrapper 布局**：`allowClear` 时 textarea 获得 `padding-inline-end: var(--ant-padding-lg)` 预留空间，清空按钮渲染在 suffix 层 `position: absolute` 右上角。
  - 清空按钮 `<button class="ant-input-clear-icon">` 常驻 DOM，通过 `ant-input-clear-icon-hidden`（`visibility: hidden`）切换可见性——可见性由 `needClear`（hasValue && focused/hover）决定，即标准 antd 行为：**有值 + hover（或 focus）时显示 ✕**，点击触发 `onChange('')` 并回调 allowClear.onClear。
  - `isPureTextArea = !autoSize && !showCount && !allowClear`——三者任一启用都切到 affix-wrapper 渲染路径，与项目已有 autoSize/showCount 用法**正交兼容**。
- **与 Input allowClear 的差异点**（规划验收标准要写对）：
  1. Input 的 ✕ 框内右侧垂直居中；TextArea 的 ✕ 在**右上角**——视觉位置不同是正常现象，不是 bug。
  2. 带 `showCount` 时计数与 ✕ 共存于 suffix 层（✕ absolute 定位，计数正常）。
  3. 受控组件点 ✕ = 触发一次 `onChange('')`，不超出 onChange 语义——与 Input 完全一致，Form 场景（FormModal 内）零额外接线。
  4. 中文输入法（composition）不受影响：清空走原生 click，不干扰 IME。
- **结论**：**不需要 focus-vs-hover 的定制决策**——直接采用 antd 默认（hover 显示、focus 时也显示），与现有 AppInput 单行输入行为同源同款。

### 实现路径（依赖 Phase 65 封装模式，复杂度 LOW）

1. 新建 `src/components/ui/AppTextArea.tsx`（约 15 行，照抄 AppInput.tsx:12-14 模式：默认 `allowClear = true`，其余全透传 `Input.TextArea`）+ ui/index.ts 导出 + `__tests__/AppTextArea.test.tsx`。
2. 74 处 JSX 替换：`<Input.TextArea` 26 处直接替换 + 解构用法 48 处（`const { TextArea } = Input` 文件改 import AppTextArea）。
3. eslint 收口：在 65 契约块的 `no-restricted-syntax` **追加进现有数组** selector `JSXOpeningElement[name.name='TextArea']` 与成员表达式 `Input.TextArea`。⚠️ 两点：①eslint.config.js:114 注释明确说成员表达式**有意不拦**——v1.23 需把 TextArea 从「有意不拦」清单移入拦截并更新注释；②eslint.config.js:141-144 注释警告「追加进本数组，勿另起 config 块——flat config 后块整体覆盖同名规则键，会使上方 4 个 selector 与内网 IP 拦截静默失效」（本项目已踩过并文档化的坑）。`no-restricted-imports` 不用动（TextArea 走 antd 主包 import）。
4. 验收：74 → 0；AppTextArea 测试绿；任选 3 处（1 处 showCount + 1 处 autoSize + 1 处 Form 内受控）浏览器验证 ✕ 出现、点击清空、onChange 正常。

## §F2 详析：COMP-05 迁移收尾「done」的定义

现状（实测 eslint.config.js）：
- 拦截已是 **error 级**（非 warn）：`JSXOpeningElement[name.name='Input']` / `'Select'`（eslint.config.js:121/126）+ `no-restricted-imports` antd 子包侧门（:156）。
- 豁免三层：①封装层目录 override（ui/ + shared/，:240）②测试文件 override ③**存量命中文件豁免 override（P67 清偿遗留）**。
- 「157/151 usages across ~58 files」全部处于**存量豁免 override 文件**里，规则对它们不生效。

**「迁移里程碑完成」应同时声明三条**（数量归零 + 规则收紧 + 守护可重复）：

1. **存量归零**：pages/ 裸 `<Input>`/`<Select>` 豁免文件清空 → `gen-p65-exemptions.mjs` / `gen-p66-exemptions.mjs` 连跑两次 diff 为空（正是 68.1 §8 第 18 条已定义的判据，直接沿用）。允许保留的合法裸使用点仅剩：ui/、shared/ 封装层 + 测试文件。
2. **规则翻面**：删除豁免 override 后规则对全 src/ 生效并保持 **error 级**（无需 warn→error 翻转——本来就是 error；要翻转的是「豁免文件清单」本身）。个别确需豁免的文件（如 3D 编辑器特殊输入）走 eslint-disable-next-line + 理由注释（与 hex-exemptions 登记制同款）。
3. **回归守护**：`npm run lint` errors ≤ 83 基线（68.1 §8 第 19 条判据）；迁移不得改变行为——遵守 68.1 §3.3 红线（查询参数一一对应、排序映射不变、批删形态不回退）。

⚠️ 迁移中的真坑（从现有代码证得）：
- `Select` → AppSelect 时 **showSearch 默认开启**——对纯字典下拉是收益，对「选项即全部」的小下拉产生多余搜索框；AppSelect 需允许 `showSearch={false}` 显式关闭，SPEC 里要写明这是预期用法而非破坏契约。
- 裸 `<Input>` 的 variant/prefix/suffix 全透传无风险；重点审查**受控 defaultValue 与 Form 组合**处；AppInput 透传 `ComponentProps<typeof Input>`，ref 类型一致，风险低。
- 工作量：~58 文件 × 平均 5-6 处，机械但分散——建议按域分波（system → operations → network/monitor → duty/workorder/ad），每波一个 commit，走 68.1 已验证的「迁移不得改变行为」自检清单。

## §F3 详析：7 项 debug 清偿的逐项修复范围（基于各 debug 文档，非泛泛建议）

| # | Debug 文档（status） | 根因状态 | 修复范围（concrete scope） | 复杂度 | 阻塞点 |
|---|---|---|---|---|---|
| 1 | linux-log-missing-2026-07-09（diagnosing） | **未定位**。已确认架构事实：HTTP 请求日志走 stdout→journald，`logs/app.log` 只含 applogger 显式调用；已排除 MaxSize/log_dir/log level 三假设 | 双轨：(a) **诊断半段**——文档「诊断待确认项」6 条命令需用户在 Linux 服务器执行（systemctl status / journalctl / ls logs / df -h / pidof / ActiveEnterTimestamp），输出回来才能在 4 个假设（重启后路径漂移[概率最高] / systemd 重启初始化失败 / 磁盘满 / WorkingDirectory 变更）中定位；(b) **修复半段**（不依赖诊断结论，可先行）——按文档「关键发现」修正日志架构：要么把 HTTP 请求日志接入文件 logger（GetFileLogger()），要么明确 journald 为唯一日志面并在部署文档写清 `journalctl -u szh-backend` 是正确查看方式。v1.23 可只做 (b) + (a) 作为 human-UAT 下发 | MEDIUM | (a) 需用户提供服务器访问；(b) 无阻塞 |
| 2 | mac-collection-cisco-no-vendor-support（diagnosed, Confidence HIGH） | **根因已锁定**：DeviceVendor 枚举封闭（huawei/h3c/ruijie/maipu 四值，`internal/models/network_device.go:17-22`）；`parseMACLine` switch 无 default 分支（`mac_collection_service.go:362-441`）→ Cisco 每行静默丢弃；且系统从未查询 `show port-security address`，sticky/secure MAC 全厂商盲区 | 文档已给出完整 6 步方案（Suggested Fix 节）：①VendorCisco 常量 + 前端 vendor 下拉/导出白名单同步（network_export_handler.go:89,205 / batch_export_handler.go:167,226）②getMACCommand Cisco 返回双命令 `["show mac address-table","show port-security address"]` 合并去重 ③parseMACLine 新增 VendorCisco 分支处理两种输出格式（5 字段 vs 7 字段）④可选 TextFSM 模板 ×2 ⑤trunk_filter 加 Cisco hint tokens（routed/layer3）⑥回归测试用 `b022.7a2e.4a4f`+Gi2/25+VLAN308 作验收用例 | MEDIUM～HIGH（模型枚举/解析/前端/测试 7 文件，但方案已到文件级） | 无——文档遗留「等用户确认进入修复」，v1.23 纳入即视为确认 |
| 3 | mac-format-recurrence-R2（investigated） | **复发路径未闭环**。R2 已证伪「其他 cron 写入」（mac_collection 是唯一 writer）并证实 binary 含 normalize 修复，但脏数据仍以 Cisco 点分格式写入——文档列 3 个未验证假设（GORM 切片插入绕过 BeforeCreate hook / 并发旧连接 / go run 构建不一致），M196 TRIGGER 仅 RECOMMENDED 未实施 | 四步（文档 §6 原文）：①停 `go run`，改 `go build -o ruoyi-backend.exe && ./ruoyi-backend.exe` 生产启动方式（消除 GOCACHE 变量）②复现实验：干净 build + 手动触发 mac_collection，dirty=0 与否 → 判定 3 假设（dirty=0 指向构建/并发；仍 dirty 指向 GORM hook 绕过 → 检查 `Create(&macRecords)` 切片路径，必要时 service 层显式 normalize）③**实施 DB TRIGGER**（BEFORE INSERT/UPDATE REGEXP 归一化两表，defense-in-depth，即 D3——注意用户历史偏好，见 D3 行警告）④跑 M194 幂等清洗存量 | MEDIUM～HIGH（假设验证有不确定性） | ①②需用户配合重启服务；建议先做①③④（确定性收益），②作附带验证 |
| 4 | mac-history-mv-double-execution（investigating） | **完全空白文档**——只有症状（每 5 分钟稳定双行 sys_job_log，时长略异均成功）与初始假设（sys_job 双行注册 / scheduler 启动双重注册），无 Evidence、无 Eliminated | 调查范围（继承文档假设）：①`SELECT ... FROM sys_job GROUP BY` 查双行 ②grep 代码中 job 注册点（internal/scheduler/ + seed migration）确认是否 double-seed ③检查 scheduler 启动阶段是否有独立 refresh goroutine（matview refresh 可能既有 cron 又有启动钩子）④均排除 → 怀疑多实例双进程（ps 确认） | LOW～MEDIUM（纯调查 + 修注册点，无数据风险） | 需 DB 读访问；7 项中唯一「零进展」文档，按全新调查估时 |
| 5 | mac-history-query-returns-empty（investigating） | 代码层已排除 4 假设（分区路由/表名拼接/MAC 格式/时间窗）；根因 **TBD 需 DB 端实测**，最可疑是 migration_175 注释声称有 deleted_at 但实际 DDL/struct 均无（schema 不自洽）+ 4 场景（a 软删标记 / b 数据含不可见字符 / c Raw SQL 吞错返回空 / d 时区 partition prune 错配） | ①文档 §verification 4 条 SQL/EXPLAIN 由用户在 PG 执行（约 10 分钟）②按命中场景修：a → QueryMACTrajectory 加过滤；b → 修 collector 写入格式（联动 #3）；c → **Raw 改 GORM chainable + handler 500 传播**（即 D4，建议无论如何都做——吞错是独立缺陷）；d → SET TIME ZONE/cast | LOW（DB 实测后）+ LOW（D4 修复） | ①需 DB 访问；D4 部分无阻塞可先行 |
| 6 | scrapligo-getprompt-nil-panic（awaiting_human_verify） | **已修复完毕**。root_cause 完整（OpenContext 初始化窗口无互斥 + ctx.Done fire-and-forget Close 竞态 + 双 Close 二次 close(Errs)），fix 已落地 4 处修改 + 3 个回归测试（scrapli_wrapper_initrace_test.go）+ scrapligo bump v1.4.2；build/vet/test 全绿 | **零代码工作**。剩余 = 人工验证：用户确认生产/本地工作流不再出现进程级 panic（原始症状 exit status 2）→ archive + 入 knowledge base。v1.23 动作仅「跟踪验证状态 → 关单」 | LOW（纯流程） | 需用户真实环境观察；建议作为 v1.23 期间被动观察项，不占排期 |
| 7 | workstation-3f130-physical-link-empty（investigated） | 静态分析完备（5F003 同源已排除），剩 4 个互斥假设 H1-H4（info_point.workstation_id 错位 / 软删或 status≠0 / workstation.user_id 空 / device_id orphan），**验证 SQL 已全部写好**（文档 §4.1-4.5 含一条综合诊断 SQL），等待执行 | ①跑文档 §4.5 综合 SQL（一条覆盖 4 假设，判读规则已写明）②按命中项修数据：H1/H4 → UPDATE 指向正确 ID；H2 → 恢复 status/deleted_at；H3 → UPDATE user_id（需与人事核对）③可选加固：infopoint_service Create/Update 补 device_id↔port_id↔workstation_id 一致性校验（文档指出 commit msg 声称有但代码未确认——需先核实） | LOW（执行 SQL + 数据修复）；可选加固 LOW～MEDIUM | 需 DB 访问 + 人事数据核对（H3） |

**排期观察**：#6 零工作量、#5/#7 差一次 SQL 执行、#1 半阻塞、#2 方案就绪可开工、#4 全新调查、#3 含不确定假设——建议：#2/#3-①③④ 与前端 F1/F2 并行；#4/#5/#7 集中在一个「DB 实测日」处理。

## §F5 详析：1367 warnings 分类与可修性（实测分布）

| 规则 | 数量 | 占比 | 机械可修性 | 处置建议 |
|---|---|---|---|---|
| `@typescript-eslint/no-explicit-any` | 958 | 70% | **三档**：①类型定义文件（types/、widget 配置）→ 直接写类型，机械可修 ②ECharts/chart 组件边界 → 用 echarts 官方类型或 unknown+narrowing，需判断 ③真正动态结构（jsonata 结果、Excel 行）→ 合理豁免进白名单 | 分层收口 + D2 预算制，**不建议清零**；目标建议「<400」而非 0 |
| `@typescript-eslint/no-unused-vars` | 363 | 27% | **高度机械**：删变量/改 `_` 前缀；`npm run lint:fix` 可消化一部分；风险仅在「看似未用实为 JSX 类型引用」的 import | 全修，接近清零；单 wave 机械批量 + type-check 守护 |
| `react-refresh/only-export-components` | 33 | 2.4% | 半机械：组件文件里导出常量/hook 的拆文件或移位，涉及 import 链调整 | 修大头（常量移 constants/）；剩余与页面结构耦合的接受或 eslint-disable 带理由 |
| `no-console` | 5 | 0.4% | 完全机械（v1.22 已收敛 console 错误出口到 handleApiError） | 全修 |
| 规则 id 为 null（解析类） | 8 | 0.6% | 需逐个看（多为 plugin 报告格式问题） | 逐个定性 |

**收敛路径**：no-unused-vars + no-console（≈368，27%）一轮清掉 → 降至 ~1000；no-explicit-any 分两 wave（类型文件 wave + 组件边界 wave）→ 目标 ~400-600；最终 `--max-warnings` 锁预算。**「1367 → TBD」建议定为两档承诺：硬承诺（no-unused-vars 归零 + 预算门禁）与软目标（no-explicit-any 降幅 50%）**，避免清零承诺拖垮里程碑。

## Feature Dependencies

```
[F1 TextArea 契约]
    └──requires──> [Phase 65 AppInput 封装模式 + ui/ 目录豁免 override]（已存在，纯复用）
    └──enhances──> [F2 COMP-05 收尾]（同一 eslint 契约块，TextArea selector 追加须与 F2 豁免清偿同 wave 落，避免 flat config 覆盖陷阱）

[F2 COMP-05 迁移]
    └──requires──> [AppInput/AppSelect/PageTable/FormModal/RemoteSelect]（已存在）
    └──requires──> [68.1 §3.3 行为不变红线]（查询参数/排序/批删形态）
    └──conflicts──> [F4 浏览器 UAT]（必须先迁完再 UAT，否则 UAT 对象漂移；UAT 是 F2 的验收闸门）

[F3 debug 清偿]
    #2 Cisco ──independent── 前端全部（纯后端 + 一处 vendor 下拉）
    #3 MAC TRIGGER ──enhances──> #5 轨迹查询（场景 b 数据格式）与 #7（MAC JOIN 依赖格式归一）
    #6 scrapligo ──independent── 全部（只等人工验证，可并行）
    #4 双执行 ──independent── 其余（唯一零共享上下文项）

[F5 lint 清偿]
    └──requires──> [F2 迁移完成后]再锁 --max-warnings 预算（否则迁移波次自身产生新 warn 干扰计数）
```

### Dependency Notes

- **F1 与 F2 同 eslint 契约块**：eslint.config.js:141-144 注释已警告「追加进本数组，勿另起 config 块——flat config 后块整体覆盖同名规则键，会使上方 4 个 selector 与内网 IP 拦截静默失效」。TextArea selector 追加必须进现有数组——这是本项目已踩过并文档化的坑。
- **F4 依赖 F2 时序**：UAT 第 15/16 条（固定列/权限）测的就是迁移页，先 UAT 后迁移 = 白测。
- **#3 与 #5/#7 弱耦合**：DB TRIGGER 上线后 #5 场景 b、#7 的格式类断点自然消失；但 #5 的错误传播修复（D4）与之独立，先行无妨。

## MVP Definition

### Launch With（v1.23 必交）

- [ ] F1：AppTextArea 契约组件 + 74 处迁移 + eslint TextArea selector 收口 — 用户直接报告的痛点，工作量最小、感知最强
- [ ] F2：裸 Input/Select 豁免清零 + 豁免生成脚本 diff 为空 + lint errors ≤83 — 「契约全量收尾」的名字来源
- [ ] F3#2：Cisco vendor 支持全链路（方案已在 debug 文档中到文件级）— 7 项中唯一「方案就绪待执行」的后端项
- [ ] F3#5/#7：DB 实测日（各一次 SQL 执行）+ D4 错误传播修复 — 两次 SQL 换两个关单
- [ ] F3#6：scrapligo 跟踪关单（零代码）
- [ ] F4：浏览器 UAT 按 68.1 §8 原文 19 条冒烟（重点 13/15/16 + 2/3/7）— 收尾验收闸门

### Add After Validation（v1.23 中后段 / v1.23.x）

- [ ] F3#1 (b) 日志架构修正（文件 logger 或 journald 单一事实源文档化）— (a) 诊断等用户服务器输出
- [ ] F3#3：TRIGGER + 生产启动方式固化（go build 替代 go run）— TRIGGER 需先与用户确认破例偏好 + 性能评审
- [ ] F3#4：双执行调查 — 零上下文，随时可插入空档
- [ ] F5 第一波：no-unused-vars + no-console 归零（≈368 条）+ --max-warnings 预算门禁

### Future Consideration（v2+）

- [ ] F5 no-explicit-any 深度清偿（958→<400）— 需类型体系设计，单独立项
- [ ] D1 进阶：AppTextArea 下沉 maxLength/字数约定 — 等实际产品需求
- [ ] Cisco vlan_link_type 列（debug 文档 #2 长期方案，Phase 40 已 defer）— 依赖现场 `show interfaces switchport` 采集

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| F1 TextArea 清空 | HIGH（直接用户报告） | LOW | P1 |
| F2 COMP-05 收尾 | MEDIUM（契约成立的前提，用户无感） | MEDIUM（量大机械） | P1 |
| F3#2 Cisco MAC | HIGH（数据完整性：整厂商 MAC 盲区） | MEDIUM | P1 |
| F3#5/#7 DB 实测 | MEDIUM（2 个存量问题关单） | LOW | P1 |
| F4 浏览器 UAT | MEDIUM（收尾置信度） | LOW | P1 |
| F3#6 scrapligo 关单 | LOW（流程性） | LOW | P2 |
| F3#3 TRIGGER+启动固化 | MEDIUM（防复发纵深） | MEDIUM | P2 |
| F3#4 双执行调查 | LOW（MV 双跑无业务损害） | LOW～MEDIUM | P2 |
| F3#1 日志架构 | MEDIUM（运维可观测性） | MEDIUM | P2 |
| F5 lint 第一波 | LOW～MEDIUM（DX/守护） | LOW | P2 |
| F5 lint 深度清偿 | LOW | HIGH | P3 |

## Sources

- `.planning/PROJECT.md`（v1.22 契约决策：默认值下沉禁逐页补、ConfirmDelete 唯一入口）
- `.planning/v1.23-seeds.md`（A 7 项 debug / B UAT 遗留 / C 流程积压 / D 冻结契约）
- `.planning/debug/` 全部 7 份文档（逐条核读，status 与 fix-scope 见 §F3 表）
- `.planning/milestones/v1.22-phases/68.1-closeout-debt-clearance/68.1-UI-SPEC.md` §3.3/§8（回归红线 + 19 条自检清单原文，L295-319）
- 代码实测：`src/components/ui/AppInput.tsx`（16 行透传封装）、`eslint.config.js`（:121/:126 selector、:114 TextArea 有意不拦注释、:141-144 flat config 覆盖陷阱注释、:156 子包拦截、:240 封装层豁免）
- 本地依赖实测：antd 6.6.4 + `@rc-component/input/es/{TextArea,BaseInput}.js` + antd dist CSS（clear-icon 显隐规则）
- lint 实测：`npx eslint src -f json` → 83 errors / 1367 warnings，分布 no-explicit-any 958 / no-unused-vars 363 / react-refresh 33 / no-console 5 / null 8

---
*Feature research for: v1.23 前端契约全量收尾与遗留清偿*
*Researched: 2026-10-27*
