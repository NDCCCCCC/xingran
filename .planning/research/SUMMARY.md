# Project Research Summary

**Project:** RuoYi-Next 运维管理系统（ruoyi-go-backend）
**Milestone:** v1.23「前端契约全量收尾与遗留清偿」
**Domain:** 前端契约组件收尾（TextArea / COMP-05 / eslint 收口）+ Go 后端 debug 清偿 + 浏览器 UAT + lint 清偿
**Researched:** 2026-10-27
**Confidence:** HIGH（全部结论基于实测 package.json / go.mod / node_modules 类型定义 / 实跑 eslint 基线 / 7 份 debug 文档逐条核读，非猜测）
**Replaces:** v1.19 SUMMARY.md（网络设备写命令，原文可从 git 历史找回）
**详细文档：** [STACK.md](STACK.md) · [FEATURES.md](FEATURES.md) · [ARCHITECTURE.md](ARCHITECTURE.md) · [PITFALLS.md](PITFALLS.md)

## Executive Summary

v1.23 是一个**零新依赖的清偿型里程碑**：不新建架构，只做「补最后一个契约组件（TextArea）→ 全量迁移（COMP-05）→ 拆除豁免脚手架（eslint 收口）→ 浏览器 UAT」的收尾链，外加 7 个互相独立的 Go 后端 debug 修复。前端侧所有能力已就位：antd 实际版本 6.6.4（非 CLAUDE.md 写的 6.1），`Input.TextArea` 原生支持 `allowClear`（`@rc-component/input` `BaseInputProps` 继承，node_modules 类型定义实证），契约组件扩展零依赖；后端 7 项修复全部落在既有层内，仅建议一次可选的 scrapligo v1.4.2 → v1.4.3 补丁升级。明确排除 jscodeshift、Playwright/Cypress、SNMP 库——Phase 67 已验证的「import 别名降险 + 逐页迁移 + 豁免脚本收缩 + vitest 总闸」打法直接复用。

**关键风险与对策**：①最大工程风险是 ~200+ 文件迁移中 AppSelect 默认 `showSearch`/`optionFilterProp="label"` 对 children/fieldNames 形态下拉的静默行为改变——对策是形态分类 + W1 低流量验证域先行 + 每页冒烟 checklist；②最大流程风险是把「acknowledge 结转」误读为「根因已定位」——7 份 debug 文档中 ≥4 项调查未完成（#1 半阻塞、#3 假设未验证、#4 完全空白、#5 待 DB 实测），**后端第一波必须是重验 spike 而非修复 wave**；③flat config 的块覆盖语义是本项目已踩过并文档化的坑（新 selector 必须追加进既有数组且同步 P68 hex 豁免块，勿另起 config 块）。

三条已裁定/需确认事项：AppTextArea 别名之争建议采纳 ARCHITECTURE 路线（独立组件 + 成员表达式 selector，**不挂静态别名**，理由见 Key Findings）；数量排期口径统一为「eslint selector 命中数 + 豁免清单」，grep 数仅供粗估；#3 的 DB TRIGGER 方案与用户 v1.19 历史「禁用 DB TRIGGER 路线」偏好冲突，**实施前须用户破例确认**。

## Key Findings

### Recommended Stack

零新增依赖（详见 [STACK.md](STACK.md)）。所有能力基于实测：antd 6.6.4 的 TextArea `allowClear` 经 rc 层继承可用，`ComponentProps<typeof Input.TextArea>` 一行类型推导（现有 AppInput/AppSelect 同模式），eslint flat config 的 selector + 豁免 override + gen-p65/p66-exemptions.mjs 幂等脚本体系已就位，Phase 67 迁移方法论（别名降险、每页独立 PR、vitest 3900+ 总闸）直接复用。

**Core technologies（全为现有栈）：**
- **antd 6.6.4**：TextArea 契约载体 — allowClear/showCount/maxLength 原生共存，无需 polyfill
- **TypeScript 5.9.3**：`ComponentProps` 推导 — AppTextArea 照抄 AppInput 16 行透传模式
- **eslint 9.39.1 flat config**：契约拦截与收口 — selector 追加 + 豁免锚点脚本收缩，规则不可降级
- **vitest 4.0.18**：迁移回归总闸 — 全量 `npm run test -- --run` 0 failed 作为每波闸门
- **agent-browser skill**（本环境已装）：浏览器 UAT — 覆盖 68.1 §8 实机回归，替代引入 Playwright
- **scrapligo v1.4.2 → v1.4.3（可选）**：唯一建议的依赖变更 — 补丁级、无 API 破坏；池层 close-vs-use 守卫才是根因修复，库升级为辅助

### 数量口径裁定（跨文档统一结论）

四份文档 + seeds 的数字不一致，**统一裁定：以 eslint selector 实跑命中数 + gen-p65/p66-exemptions.mjs 豁免清单为权威排期口径，grep 数仅供粗估**。eslint 命中的才是被拦截的存量，且豁免清单是收口对账的唯一分母（收口判据 = 清单条目数与 grep 计数同时归零）。

| 项 | seeds | STACK 实测 | FEATURES 实测 | ARCHITECTURE 实测 | 权威口径 |
|---|---|---|---|---|---|
| 裸 `<Input>` | ~157 | 202（pages） | ~188 行 | 220 处/71 文件 | **eslint 命中数 + P65 豁免清单** |
| 裸 `<Select>` | ~151 | 271（pages） | ~271 行 | 271 处/86 文件 | **eslint 命中数 + P66 豁免清单** |
| TextArea | 74 | 61（43 `<TextArea>` + 18 `Input.TextArea`） | 74（26 + 48，含 components/） | 61 处/47 文件 | **迁移对账以 JSX 逐处清单为准**（形态分两种，见下） |
| lint 基线 | — | 83 errors / 1367 warnings | 同左 | 同左 | **83 / 1367**（三方实测一致，无分歧） |

差异来源：grep 行数 vs JSX 节点数、pages-only vs 含 components/、多行 JSX 拆分误差。数量级一致，规划时先跑一次权威计数脚本落盘基线（PITFALLS 2.1 第一 pitfall 即「规划前分母用错」）。

### 裁定 1：AppTextArea 别名收口（STACK vs ARCHITECTURE 分歧）

**共识**：独立 `AppTextArea.tsx` 组件文件（不采用 union type / polymorphic / `multiline` prop 方案），默认 `allowClear = true`，其余透传；`ui/index.ts` 具名导出 + `__tests__` 契约测试；FormModal 零修改、SearchForm 加 `textarea` field type。

**分歧与裁定建议**：STACK 建议挂载 `AppInput.TextArea = AppTextArea` 静态属性别名（让 26 处 `<Input.TextArea>` 形态只改 import 行、零 JSX 改动）；ARCHITECTURE 反对加别名（`const { TextArea } = Input` 解构形态无法用 import 别名降险，48 处必须改 JSX）。

**建议裁定：采纳 ARCHITECTURE——独立具名导出，不挂静态别名。** 决定性理由是 selector 收口的自洽性：现有 selector 只拦普通 JSX 名，成员表达式 `Input.TextArea` 有意不拦（`eslint.config.js:114`）；v1.23 必须新增成员表达式 selector 才能收口 TextArea 存量。但成员表达式 selector **无法区分**「`AppInput.TextArea`（挂载别名后的合法契约用法）」与「antd `Input.TextArea`（非法裸用法）」——两者 JSXMemberExpression 的 object name 均为 `Input`。一旦挂载别名，该 selector 会误伤全部合法用法，等于永久放弃成员表达式拦截面，74 处收口假绿（PITFALLS 1.1 正是警告这一点）。STACK 方案省下的 26 处 import 行改动，代价是拦截面残废；且其自己承认 48 处解构形态仍须逐处改 JSX，别名收益不完整。**执行形态**：pages 全部改用 `<AppTextArea>`；eslint 追加 `JSXOpeningElement[name.name='TextArea']` + 成员表达式两条 selector 进既有数组（65 契约块 + P68 hex 豁免块两处同步）；`ui/` 封装层豁免块内是唯一合法裸 `Input.TextArea` 写法。此项为研究裁定建议，SPEC 阶段定稿。

### Expected Features

详见 [FEATURES.md](FEATURES.md)。

**Must have（table stakes）：**
- **F1 TextArea 一键清空契约**：用户直接报告的痛点，74 处（口径见上表）无清空按钮。唯一正确路径 = 契约层下沉（AppTextArea 默认 allowClear），禁止逐页补 prop（v1.22 锁定决策）。复杂度 LOW
- **F2 COMP-05 全量收尾**：裸 `<Input>`/`<Select>` 迁到 AppInput/AppSelect，豁免清单清空收口。「迁移完成」须同时声明三条：存量归零（gen 脚本 --dry 连跑两次 diff 为空，即 68.1 §8 #18 判据）+ 豁免 override 拆除（selector 本就是 error 级，翻转的实质是拆脚手架）+ lint errors ≤ 83。遵守 68.1 §3.3 行为不变红线（查询参数一一对应、排序映射不变、批删形态不回退）。复杂度 MEDIUM（量大单点简单）
- **F3 7 项 debug 清偿**：完成度分级见下表
- **F4 浏览器实机 UAT**：按 68.1 UI-SPEC §8 全表 19 条（重点 #13 分页 / #15 固定列不失效 / #16 行操作权限不变 + #2/#3/#7）。**注意：seeds 的「15/16 = 分页 + 批删」编号是错的**——§8 原文 :315-316 第 15/16 条是「固定列不失效」「行操作权限不变」，分页口径是 #13、批量删除是 #5/#6。UAT 前置：prod build（`npm run build && npm run preview`）+ `VITE_ENABLE_REQUEST_ENCRYPTION` 按生产配置 + 清 sessionStorage 验证 usePagination 持久化；结果回写 UI-SPEC §8 并关闭，带截图证据
- **F5 lint warnings 降量**：硬承诺 = `no-unused-vars`（363）+ `no-console`（5）归零 + `--max-warnings N` 预算门禁进 CI；软目标 = `no-explicit-any`（958，占 70%）降 50%（建议目标 <400 而非 0，chart/ECharts 边界的 any 强行清零会引入运行时错误）。**排迁移之后**（迁移波次自身产生新 warn 干扰计数）；不可降规则、禁止全量 `--fix`（实测仅 8 条 warnings 有 fixer）、errors 锁 83 零新增

**Should have（differentiators）：**
- SearchForm `textarea` field type（F1 集成点）
- D4 MAC 轨迹查询错误传播修复（Raw SQL 吞 error → GORM chainable + handler 500）——独立缺陷，无论 #5 根因是什么都建议做
- 逐页顺手清本页 unused-vars/any（迁移逐行过文件时搭车）

**Defer（v2+）：**
- no-explicit-any 深度清偿（958→<400 之外的余量）——需类型体系设计，单独立项
- AppTextArea 下沉 maxLength/字数约定——等实际产品需求
- Cisco vlan_link_type 列——依赖现场 `show interfaces switchport` 采集

### F3：7 项 debug 完成度分级（裁定 3）

**完成度分级表**（依据各 debug 文档 frontmatter + Evidence 节实读，见 [FEATURES.md §F3](FEATURES.md) / [ARCHITECTURE.md §3](ARCHITECTURE.md)）：

| # | 项 | status | 完成度 | v1.23 动作 | 阻塞 |
|---|---|---|---|---|---|
| 6 | scrapligo GetPrompt nil panic | awaiting_human_verify | **已修复完毕**（4 处修改 + 3 回归测试 + v1.4.2，build/vet/test 全绿） | **零代码关单**：人工验证生产无 panic → archive + knowledge base；作被动观察项，不占排期 | 需用户环境观察 |
| 2 | Cisco MAC 采集缺厂商支持 | diagnosed (HIGH) | 文件级已诊断（6 步方案到文件:行号，7 文件） | 直接执行（纳入即视为用户确认） | 无 |
| 5 | MAC 轨迹查询返回空 | investigating | 代码层 4 假设已排除，差 DB 端 SQL 实测（§verification SQL 已写好） | DB 实测日执行 + D4 错误传播修复（无阻塞先行） | DB 访问 |
| 7 | 工位 3F130 物理链路为空 | investigated | 静态分析穷尽，H1-H4 互斥假设 + 综合 SQL 已备好 | DB 实测日执行 + 按命中项修数据 | DB 访问 + 人事核对（H3） |
| 1 | Linux 日志缺失 | diagnosing | **半阻塞**：架构事实已确认（双路日志分流），6 条诊断命令等用户在 Linux 服务器执行 | 修复半段先行（文件 logger 双写或 journald 单一事实源文档化）；诊断半段作 human-UAT 下发 | 用户服务器输出 |
| 3 | MAC 格式脏数据复发 R2 | investigated | 复发路径未闭环（3 假设未验证）；但确定性动作（① go build 替代 go run ③ DB TRIGGER ④ M194 幂等清洗）可先行 | 先行 ①③④，②复现实验附带验证 | TRIGGER 需用户破例确认（见下） |
| 4 | mac_history MV 双执行 | investigating | **唯一空白文档**（无 Evidence/Eliminated，症状+初始假设） | 全新调查：查 sys_job 双行 / 注册点 grep / 独立 goroutine / 多进程；cron.go 是共享引擎，**单独 phase + 全 scheduler 测试套件** | DB 读访问 |

**⚠️ DB TRIGGER 偏好冲突**：#3 的四步方案含 DB TRIGGER 纵深防御（BEFORE INSERT/UPDATE REGEXP 归一化），但 v1.19 研究记录用户历史偏好「根因修复走代码层，禁用 DB TRIGGER 路线」。R2 是复发型数据污染（defense-in-depth 论据更强），**实施前须与用户确认是否为此场景破例**——不确认则先走 Go 侧防线（service 层显式 normalize + 全写入方枚举 + 写路径回归测试）。

**⚠️ acknowledge ≠ 已定位**（PITFALLS 3.1）：7 项中 ≥4 项调查未完成，**后端第一波必须是重验 spike**（逐文档核对 status、产出给用户的 SQL/命令清单、三分类落盘），修复任务只对「根因已定位」类下发；禁止在用户数据回填前写 #1/#7 修复代码。

### Architecture Approach

v1.23 无新架构（详见 [ARCHITECTURE.md](ARCHITECTURE.md)）。前端增量 = `ui/AppTextArea.tsx` + SearchForm 一个 field type + eslint selector 追加；迁移 = 纯 JSX 换名 + import 换路径，v1.22 冻结契约（分页口径/ConfirmDelete/handleApiError/statusHelpers）不在触碰面上，真正要盯的是三个契约默认值的**行为外溢**（allowClear 清空按钮出现在搜索框、showSearch 改变下拉交互、AppTextArea autoSize 改变弹窗高度）。后端 7 项修复全部落在既有层内（`internal/device/`、`internal/services/`、`internal/scheduler/`、`pkg/logger`），无新端点、无新模块、无 operlog 新增。

**Major components：**
1. **契约组件层**（`src/components/ui/`）— AppTextArea 为唯一新增；封装层豁免块永久保留
2. **eslint 豁免脚手架**（P65/P66 存量豁免块 + gen 脚本）— 收口后拆除退役；封装层/测试/hex 豁免块保留
3. **共享引擎**（`internal/scheduler/cron.go`）— #4 改动的波及面，独立 phase 隔离
4. **UAT 产物归属**（68.1 UI-SPEC §8 回归清单）— 结果回写 + 证据关闭，不让债再次无主结转

### Critical Pitfalls

Top 5（详见 [PITFALLS.md](PITFALLS.md)）：

1. **成员表达式是 selector 盲区** — 现有 selector 只拦 `<Input>`，拦不了 `<Input.TextArea>`/`<TextArea>`；不新增成员表达式 selector 则 74 处收口假绿（lint 全绿但 grep 仍有命中）。新 selector 用 `JSXMemberExpression` 形态，追加进既有数组勿另起块（flat config 同名键整体覆盖，本项目已踩坑并注释成文），且 P68 hex 豁免块重声明了完整数组必须同步。
2. **AppSelect 默认值帮倒忙** — `showSearch = count>=8` + 无条件 `optionFilterProp="label"` 对 children/fieldNames/labelInValue 形态是破坏性的（过滤指向不存在字段）；children 形态与业务禁搜下拉需单独决策，不做形态分类直接套契约 = 搜索整体失灵且无任何编译期报错。
3. **allowClear 下沉的 onChange(undefined) 炸弹** — 契约下沉把覆盖从 30% 拉到 100%，受控字段点 ✕ 变 `undefined` 传到后端（org_id UUID 校验）变 400 而前端报含混错误；归一化（undefined→不提交）必须在封装层一次做完，UUID 类字段逐处 grep 防护。
4. **lint 清偿的 --fix 诱惑** — 全量 `--fix` 会删掉 API 形态文档性死代码、`any→unknown` 类型蔓延反撒 `as any`、fixer 制造新违规击穿 83 errors 线；必须按规则分组 + 人工确认 + errors≤83 脚本断言（写成脚本不靠人眼）。
5. **UAT 重演「jsdom 全绿、浏览器翻车」** — 这正是本次要清偿的债的成因；jsdom 测不到固定列、真实滚动、Modal 层级竞态、sessionStorage 多 tab、加密链路 401→refresh；UAT 必须在 prod build + 加密开启环境跑，且 Windows 之外 Linux 侧至少一次冒烟（本里程碑同时在修 Linux 日志项）。

## Implications for Roadmap

基于研究的建议 phase 结构（吸收 [ARCHITECTURE.md §4.2](ARCHITECTURE.md) 的 W0-W7 波次 + [FEATURES.md](FEATURES.md) 依赖图 + [PITFALLS.md](PITFALLS.md) 排序约束）：

### Phase 1: 后端 debug 重验 spike + 信息收集下发
**Rationale:** PITFALLS 3.1：7 项中 ≥4 项调查未完成，直接排修复 = 修复不存在根因的代码。
**Delivers:** 7 项三分类清单落盘（根因已定位待修 / 调查中断需续查 / 需外部输入）；给用户的 SQL/命令清单下发（#1 的 6 条 Linux 诊断命令、#5 的 4 条 verification SQL、#7 的 §4.5 综合 SQL）；#2 方案就绪确认。
**Addresses:** F3 前置；#6 关单流程启动（被动观察项）。
**Avoids:** PITFALLS 3.1（acknowledge ≠ 已定位）、3.4（缺现场数据静态推演成稿）。

### Phase 2: TextArea 契约基建（W0）
**Rationale:** SearchForm 是搜索区唯一渲染端点，必须先于一切迁移页，否则产生 custom.render 第二路径（v1.22 要消灭的东西）。
**Delivers:** `AppTextArea.tsx` + `ui/index.ts` 导出 + 契约测试（showCount/maxLength/受控/Form required 四象限矩阵）+ SearchForm `textarea` field type + eslint 成员表达式 selector 上线（两处数组同步）+ gen 脚本预登记存量命中文件（selector 上线瞬间不产生新 error）。
**Uses:** AppInput 16 行透传模式、eslint flat config 既有契约块。
**Avoids:** PITFALLS 1.1（selector 盲区假绿）、1.2（allowClear 语义耦合）。
**Research flag:** 契约默认值白名单（是否默认 autoSize）建议 SPEC 阶段结合 74 处用法审计定稿。

### Phase 3: TextArea 存量迁移 + COMP-05 分域迁移（W1-W6）
**Rationale:** 74 处 TextArea（26 处 `<Input.TextArea>` + 48 处解构）与裸 Input/Select 迁移同用「改 import 行 + 逐页 PR + 豁免脚本收缩」的 P67 打法，合并执行减少波次；W1 低流量验证域先行跑通回归模式。
**Delivers:** 按域分波迁移（W1 monitor/profile/knowledge → W2 system → W3 operations → W4 network → W5 duty/ad-domain → W6 workorder/vdi/asset）；每文件原子迁移 + 测试文件同步迁移 + gen 脚本 diff 入 commit；**#3 先于 #2 的后端项可与前端波次并行**（同文件 mac_collection_service.go，不同 phase 波次）。
**Uses:** P67 别名降险（AppInput as Input）、vitest 总闸、形态分类器（Select children/fieldNames/labelInValue 分类）。
**Avoids:** PITFALLS 2.1（豁免清单脱节）、2.2（showSearch 误伤）、2.3（undefined 炸弹）、2.4（mock/snapshot 海啸）；68.1 §3.3 行为不变红线。

### Phase 4: 后端 debug 修复 wave（#2 / #3 / #1-b / D4）
**Rationale:** 仅对 spike 确认「根因已定位/方案就绪」的项下发；#3 的确定性动作（go build 启动固化 + 可选 TRIGGER + M194 清洗）先于 #2（同文件，先归一化防线再做厂商扩展）；TRIGGER 需用户破例确认。
**Delivers:** Cisco vendor 全链路（枚举+双命令+解析分支+导出白名单+回归测试，验收用例 `b022.7a2e.4a4f`）；MAC 格式防线（全写入方枚举 + default 分支显式报错可观测 + 写路径回归测试）；Linux 日志修复半段；D4 错误传播修复。
**Addresses:** F3#2/#3/#1(b)/D4。
**Avoids:** PITFALLS 3.2（R3 复发：单点修复无效）、3.3（清洗脚本双执行复刻）。
**Research flag:** TRIGGER 路线在用户破例确认 + 性能评审后才进 plan。

### Phase 5: DB 实测日 + cron.go 双执行调查（#4 / #5 / #7 集中）
**Rationale:** 排序约束「DB 实测日集中 #4/#5/#7」；cron.go 是全调度器共享引擎，改动影响所有 job，独立 phase 隔离回归面。
**Delivers:** #5 场景 a-d 定性 + 修复（含 D4 若未在 Phase 4 落地）；#7 H1-H4 判读 + 数据修复 + 可观测性加固；#4 注册层调查（sys_job 双行 / 注册点 / goroutine / 多进程）+ 修复（验收 = sys_job_log 同 tick 双条消失）。
**Uses:** debug 文档已备好的全套 SQL；`go test ./internal/scheduler/` 全套（cron_test / mac_tasks / ad_sync / vdi / workorder）。
**Avoids:** PITFALLS 3.3（双执行修错层：只修注册层，handler guard 仅纵深）、分区表 deleted_at 语义误伤。
**Research flag:** #4 是唯一零进展文档，估时按全新调查；#4 若确认 sys_job 双行根因，唯一索引 migration 一并裁决。

### Phase 6: COMP-05 收口 + lint 清偿第一波（W7）
**Rationale:** 排序约束：lint phase 在迁移后（迁移波次自身产生新 warn 干扰计数）；收口 = 拆脚手架而非改级别。
**Delivers:** P65/P66 豁免锚点清空 → 存量豁免块删除 → gen 脚本退役（留 --dry 校验模式）；lint errors ≤ 83 脚本断言；no-unused-vars + no-console 归零（≈368 条，机械先行）+ `--max-warnings N` 门禁进 npm script；no-explicit-any 软目标降 50%（分两 wave：类型文件 wave + 组件边界 wave）。
**Uses:** gen-p65/p66-exemptions.mjs 幂等收缩机制、exemption-reasons.json 登记制。
**Avoids:** PITFALLS 4.1（--fix 掩盖 bug、击穿 83 errors 线）。

### Phase 7: 浏览器实机 UAT + 收档
**Rationale:** 排序约束：UAT 在迁移后（UAT 第 13/15/16 条测的就是迁移页，先 UAT 后迁移 = 白测）；UAT 是 F2 的验收闸门。
**Delivers:** 68.1 §8 全表 19 条实机冒烟（重点 #13/#15/#16 + #2/#3/#7），prod build + 加密开启 + sessionStorage 清空前置；结果回写 UI-SPEC §8 关闭并带证据；#1 诊断半段作 human-UAT 下发；#6 scrapligo 关单；F3 各 debug 文档 Resolution 节回填。
**Uses:** agent-browser skill（零新依赖）、Phase 63 冒烟模板、HUMAN-UAT 文档惯例。
**Avoids:** PITFALLS 5.1（jsdom-vs-真实浏览器重演）、Linux 侧冒烟缺失（本里程碑在修 Linux 日志项）。

### Phase Ordering Rationale

- **前端链是严格串行的漏斗**：契约基建 → 迁移 → 收口 → lint → UAT。迁移依赖 selector 上线（否则存量无迁移压力信号）；lint 依赖迁移完成（计数干扰）；UAT 依赖迁移完成（测的对象是迁移后页面）。唯一并行通道是后端 debug 线（Phase 1/4/5 与 Phase 3 全程无共享文件）。
- **后端线内部次序**：spike 先行（不修没定位的根因）→ #3 先于 #2（同文件 mac_collection_service.go，先防线后扩展）→ cron.go 独立 phase（共享引擎回归面）→ DB 实测日集中 #4/#5/#7（一次环境准备换三个关单）。
- **#6 全程被动**：零代码，跟踪人工验证状态，任何时点可关单。
- **豁免脚本纪律贯穿全程**：每批迁移重跑 gen 脚本（禁手工改锚点数组）、每文件原子迁移（防半迁移态双包）、grep 计数与豁免清单条目数对账归零。

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 2（TextArea 契约）**: 默认值白名单需对 74 处用法做 showCount/maxLength/受控/onChange 四象限审计后再定稿（PITFALLS 1.2 反对无白名单全量默认）
- **Phase 4（TRIGGER 路线）**: 依赖用户对「禁用 DB TRIGGER 路线」历史偏好的破例确认 + 性能评审
- **Phase 5（#4 双执行）**: 唯一空白调查文档，估时不确定性最高；PG 分区表 + deleted_at 缺失的 schema 级事实使查询/清洗改动需先跑 5 条基线 SQL

Phases with standard patterns (skip research-phase):
- **Phase 3（分域迁移）**: P67 已验证方法论，纯机械执行 + checklist
- **Phase 6（lint 清偿第一波）**: unused-vars/no-console 高度机械，分布与策略已实测落盘

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | 全部基于 package.json / go.mod / node_modules 类型定义实测；零新依赖结论可复核 |
| Features | HIGH | 7 份 debug 文档逐条核读；lint 基线 83/1367 三方实测一致；antd 6.6.4 行为基于本仓库安装产物非文档转述 |
| Architecture | HIGH | 前端全部回源核验（eslint.config.js 544 行全文、AppInput/AppSelect/SearchForm 源码）；后端基于源码定位 + debug 文档，4 项修复方案依赖 DB/远程实测已显式标注 |
| Pitfalls | HIGH | 关键 pitfall（flat config 覆盖、selector 盲区、R2 复发、双执行前科）均有本仓库成文前科或实证，非泛泛建议 |
| 数量口径 | MEDIUM | grep 数在四份文档间有差异（口径不同所致），权威口径已裁定为 eslint 命中数 + 豁免清单，但权威基线数字须规划时跑脚本落盘后确定 |

**Overall confidence:** HIGH

### Gaps to Address

- **AppTextArea 默认 autoSize 与否**：STACK 契约草案不含 autoSize 默认，ARCHITECTURE 草案建议 `autoSize = {minRows:2, maxRows:6}`（影响 FormModal 长表单滚动）——SPEC 阶段对 74 处用法审计后裁决
- **lint 收口后是否将已清零规则翻 error**：ARCHITECTURE 建议全部域完成后一次翻转；FEATURES 警告 react-refresh 等 33 处与页面结构耦合、翻 error 代价不成比例——收口时按实际余量裁决
- **#3 TRIGGER 破例确认**：不确认则 Go 侧防线方案（写入方枚举 + 显式 normalize + 回归测试）为默认路线
- **#4 双执行根因**：完全空白，plan 阶段按全新调查估时，修复形态待调查输出
- **权威计数基线**：规划前先跑一次权威计数脚本（eslint 命中数 + 豁免清单条目数 + grep 三数落盘），否则收口对账分母用错
- **scrapligo 版本漂移**：文档记录 v1.3.3、stack trace v1.4.0、go.mod 现为 v1.4.2——#6 关单前核对升级时间线归档即可，无代码风险

## Sources

### Primary (HIGH confidence)
- [STACK.md](STACK.md) — 实测版本矩阵、AppTextArea 三方案裁决、lint 清偿策略、7 项 debug 库需求判定、What NOT to Use
- [FEATURES.md](FEATURES.md) — F1-F5 详析、7 项 debug 完成度分级表、§F2 「done」三条判据、§F5 1367 warnings 实测分布、MVP 定义
- [ARCHITECTURE.md](ARCHITECTURE.md) — eslint 契约块全图（544 行）、W0-W7 波次、7 项集成点/隔离性/依赖矩阵、反模式四条、UAT §8 编号核对
- [PITFALLS.md](PITFALLS.md) — 5 领域 pitfall 全文、Technical Debt Patterns、Looks-Done-But-Isn't Checklist、Pitfall-to-Phase Mapping
- 代码实测（2026-10-27）：`eslint.config.js` 全文、`src/components/ui/*` 源码、`npx eslint .` 实跑 83/1367、`src/pages` 裸组件 grep、`go.mod` + `go list -m -versions`
- `.planning/debug/` 全部 7 份文档逐条核读
- `.planning/milestones/v1.22-phases/68.1-closeout-debt-clearance/68.1-UI-SPEC.md` §3.3/§4.4/§8 原文
- `.planning/v1.23-seeds.md` §A/§B/§C/§D（结转种子与冻结契约）

### Secondary (MEDIUM confidence)
- `.planning/milestones/v1.22-phases/67-roi/67-PLAN.md` — Phase 67 迁移方法论（别名降险、每页 PR、vitest 总闸先例）
- `.planning/debug/mac-format-recurrence-R2.md` — M196 TRIGGER 提案（RECOMMENDED but NEVER IMPLEMENTED）与用户历史偏好冲突的来源

### Tertiary (LOW confidence)
- seeds 数量口径（157/151/74）— 与实测存在系统性偏差，仅供数量级参考，排期勿用
- CLAUDE.md 记载的 antd 6.1 — 实测为 6.6.4，以实测为准

---
*Research completed: 2026-10-27*
*Ready for roadmap: yes*
