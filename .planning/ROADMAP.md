---
last_updated: 2026-10-27
update_trigger: v1.23 前端契约全量收尾与遗留清偿 roadmap CREATED (Phases 69-75, 22/22 REQ coverage)
previous_update: 2026-10-27 v1.22 SHIPPED & ARCHIVED
---

# Roadmap: RuoYi-Next 运维管理系统

## Milestones

- ✅ **v1.0 工位导入部门/用户关联** — Phases 1-2 (shipped 2026-04-16)
- ✅ **v1.1 信息点设备端口关联** — Phase 3 (shipped 2026-04-16)
- ✅ **v1.2 可配置仪表盘生产级改造** — Phases 4-7 (shipped 2026-04-21)
- ✅ **v1.3 技术债清理** — Phases 8-10 (shipped 2026-04-27)
- ✅ **v1.4 MAC地址采集优化** — Phase 11 (shipped 2026-05-09)
- ✅ **v1.5 MAC地址历史数据管理** — Phases 12-15 (shipped 2026-06-15; Phase 13 gap closure retroactive 2026-06-26)
- ✅ **v1.6 API密钥管理系统** — Phase 16 (shipped 2026-05-09)
- ✅ **v1.7 前后端加密配置同步** — Phase 17 (shipped 2026-05-20)
- ✅ **v1.8 登录端点加密增强** — Phase 18 (shipped 2026-05-21)
- ✅ **v1.9 AD域控集成扩展** — Phases 19-20 (shipped 2026-05-24)
- ✅ **v1.10 网络设备权限修复** — Phase 21 (shipped 2026-05-24)
- ✅ **v1.11 AD组自动同步系统** — Phase 23 (shipped 2026-05-26)
- ✅ **v1.13 资产管理模块** — Phase 26 (shipped 2026-06-08)
- ✅ **v1.14 全局列自定义** — Phase 27 (shipped 2026-06-09)
- ✅ **v1.15 工位设备关联** — Phase 28 (shipped 2026-06-09)
- ✅ **v1.12 深信服桌面云集成 (22A+22B)** — Phases 22A/22B (shipped 2026-06-02)
- ✅ **v1.15 部门物理位置映射** — Phase 39 (shipped 2026-06-25)
- ✅ **v1.16 技术债清理 (Tech-Debt Cleanup)** — Phases 40-41 (shipped 2026-06-26)
- ✅ **v1.17 资产对账 (Asset Reconciliation)** — Phases 42-46 (shipped 2026-07-03) — see [milestones/v1.17-ROADMAP.md](milestones/v1.17-ROADMAP.md)
- ✅ **v1.18 网络设备硬件清单 (Device Component Serials)** — Phase 48 (shipped 2026-07-04) — see [milestones/v1.18-ROADMAP.md](milestones/v1.18-ROADMAP.md)
- ✅ **v1.19 网络设备写命令 (Network Device Port Write Operations)** — Phases 50-55 (shipped 2026-07-08) — see [milestones/v1.19-ROADMAP.md](milestones/v1.19-ROADMAP.md) | [audit](milestones/v1.19-MILESTONE-AUDIT.md)
- ✅ **v1.20 网络设备 VLAN + 端口绑定 (Network Device VLAN + Port Binding)** — Phase 56 (shipped 2026-07-10) — see [milestones/v1.20-ROADMAP.md](milestones/v1.20-ROADMAP.md)
- ✅ **v1.21 AD 双向同步 bug 修复 (AD Bidirectional Sync Bug Fixes)** — Phases 57-62 (shipped 2026-10-22) — see [milestones/v1.21-ROADMAP.md](milestones/v1.21-ROADMAP.md) | [audit](milestones/v1.21-MILESTONE-AUDIT.md) | [requirements](milestones/v1.21-REQUIREMENTS.md)
- ✅ **v1.22 前端公用组件整治 (Frontend Shared Component Remediation)** — Phases 63-68.1 (shipped 2026-10-27) — see [milestones/v1.22-ROADMAP.md](milestones/v1.22-ROADMAP.md) | [audit](milestones/v1.22-MILESTONE-AUDIT.md) | [requirements](milestones/v1.22-REQUIREMENTS.md)
- 🚧 **v1.23 前端契约全量收尾与遗留清偿 (Frontend Contract Full Rollout & Legacy Clearance)** — Phases 69-75 (planning)

---

## v1.23 前端契约全量收尾与遗留清偿 — 🚧 In Progress

**Milestone goal:** 把 v1.22 建立但未落地的契约真正推到全量——修复用户主诉的「文本输入框无法一键删除」，并清偿 v1.22 收档结转的 debug / UAT / 流程积压，让里程碑以「无债、无盲区」收档。

**问题来源（用户实机反馈 + v1.22 收档结转）:**

- 用户主诉：「文本输入框无法一键删除」——根因是**多行输入框（TextArea）零 allowClear 覆盖**：`AppInput` 契约只包单行 `Input`（16 行），`eslint.config.js:114` 注释又明确声明 `Input.TextArea` 成员表达式「有意不拦」→ 零约束盲区，验收假绿
- 存量迁移未完成：`AppInput`/`AppSelect` 在 `src/pages/` 仅 15/13 处使用，裸 `<Input>`/`<Select>` 仍在 P65/P66 豁免 override 中（COMP-05 只做了试点页）
- 7 项 debug 遗留经逐份核读，**≥4 项调查未完成**（acknowledge ≠ 已定位），第一波必须是**重验 spike** 而非修复 wave

**研究基线:** `.planning/research/{STACK,FEATURES,ARCHITECTURE,PITFALLS,SUMMARY}.md`（4 维并行研究，overall confidence HIGH，commit `b5b3aeaf`）+ `.planning/v1.23-seeds.md` + `.planning/debug/` 7 份 debug 文档。

### 关键决策 (locked)

| 决策 | 理由 |
|------|------|
| `AppTextArea` **独立具名导出**，不挂 `AppInput.TextArea` 静态别名 | 成员表达式 selector 无法区分「合法别名用法」与「非法裸用法」，两者 JSXMemberExpression 的 object name 都是 `Input`；挂别名会永久放弃成员表达式拦截面、74 处收口假绿 |
| 数量排期口径 = **eslint selector 实跑命中数 + 豁免清单条目数**（权威），grep 数仅供粗估 | 研究文档间口径 157-220 不一致（grep 行数 vs JSX 节点数 vs pages-only）；**每个 phase 的成功标准以「权威计数落盘/对账」表述，不硬编码某个 grep 数** |
| DBG-03 走 **Go 侧防线**（写入方枚举 + 显式 normalize + 回归测试），**不用 DB TRIGGER** | 维持 v1.19「禁用 DB TRIGGER 路线」决策；规避 SQLite 双实现成本与数据流隐式化 |
| 零新依赖 | 前端不引 jscodeshift / Playwright，UAT 用环境已有的 agent-browser skill；后端 scrapligo v1.4.2 现状即可 |
| 排序硬约束：迁移 → 收口 → lint → UAT | UAT 必须测**迁移后**的页面；lint 排在迁移后（迁移波次自身产生新 warn 峰值） |
| 后端 debug 线与前端线**全程无共享文件，可并行** | Phase 69/72/73（后端）与 Phase 70/71（前端）并行执行；Phase 74/75 必须等前端线完成 |
| **#3 先于 #2** | 同一文件 `mac_collection_service.go`：先落格式归一化防线，再叠加 Cisco 厂商扩展，否则复发验证会被新写入面污染 |
| `cron.go` 改动**独立隔离** | 它是全部 job 的共享引擎 → 回归闸门锁 `go test ./internal/scheduler/` 全套 |

### 并行结构与依赖矩阵

```
后端 debug 线 ─────────────────────────────────────────────┐
  Phase 69 重验 spike ──► Phase 72 修复 wave ──► Phase 73 DB 实测日 │
      (DBG-01)              (DBG-02/03/04)       (DBG-05/06/07)  │
                                                                 │
前端契约线 ─────────────────────────────────────────────┐       │
  Phase 70 契约基建 ──► Phase 71 分域迁移 ──┬──► Phase 74 收口 ──► Phase 75 UAT
  (TEXT-01/03/04,HYG-05) (TEXT-02,MIG-01/02/04)│   (MIG-03,HYG-01~04) (UAT-01/02)
                                               │
                            两线并行区间：Phase 69/70/71 ∥ 72/73 ──────┘
```

| Phase | 前置硬依赖 | 可并行对象 |
|-------|-----------|-----------|
| 69 | 无（v1.23 首 phase） | 70 / 71 |
| 70 | 无（可与 69 并行） | 69 / 72 |
| 71 | 70（契约组件与 selector 必须先就位） | 72 / 73 |
| 72 | 69（spike 确认根因已定位/方案就绪） | 70 / 71 |
| 73 | 72（MAC 格式防线落地后 #7 复测才有判读力）+ 69（SQL 清单） | 70 / 71 |
| 74 | 71（收口必须建立在迁移完成之上） | 73（后端收尾） |
| 75 | 74（UAT 必须测收口后的最终页面）+ 72/73（debug 文档 Resolution 回填） | — |

### 贯穿全程的红线（每 phase 的 plan 必须遵守）

1. **eslint flat config 陷阱**（ARCHITECTURE §5.2）：新 selector **必须追加进既有数组**，勿另起 config 块——后块会覆盖同名 rule key，静默杀掉 65 契约块的 4 个 selector + 内网 IP 规则；**P68 hex 豁免块重声明了完整数组，两处必须同步更新**；豁免块**禁止用 `"off"`**（连带杀掉 antd 子包拦截）；锚点 files 数组**禁止手改**（跑 gen 脚本）。
2. **迁移风险三条红线**（PITFALLS 2.1-2.4）：①豁免清单脱节（手工删清单 / 半迁移态提交 / 规则置 off 都会静默破坏契约）；②AppSelect 无条件 `optionFilterProp="label"` 对 children 形态 Select 是**破坏性**的；③allowClear 全量下沉给存量控件引入 `onChange(undefined)` → UUID 校验 400 风险。
3. **v1.22 冻结契约不得破坏**：分页口径「共 N 条 / M 页」（usePagination 唯一实现）/ ConfirmDelete 唯一危险操作入口 + `exemption-reasons.json` 校验 / AppInput·AppSelect·PageTable·SearchForm·FormModal·RemoteSelect 契约 / handleApiError 唯一错误出口 + icon-only 必须 aria-label / statusHelpers·datetime 常量·hex-exemptions 登记制。
4. **后端 debug 线**：禁止在用户数据回填前写 #1 / #7 的修复代码（PITFALLS 3.4）；`cron.go` 改动只允许打在**注册层**（scheduler 启动 + sys_job 唯一性），handler guard 只能作纵深防御（PITFALLS 3.3）。

## Phases

- [ ] **Phase 69: 后端 debug 重验 spike + 信息收集下发** — 7 项 debug 三分类定级落盘 + 给用户下发 #1/#5/#7 的 SQL/命令清单 + #2 方案就绪确认 + #6 关单（DBG-01）
- [ ] **Phase 70: TextArea 契约基建 (W0)** — AppTextArea.tsx + ui/index.ts 导出 + 契约测试 + SearchForm `textarea` field type + eslint 成员表达式/解构形态双 selector（65 契约块 + P68 hex 豁免块两处同步）+ 权威计数基线落盘（TEXT-01, TEXT-03, TEXT-04, HYG-05）
- [ ] **Phase 71: TextArea 存量迁移 + COMP-05 分域迁移 (W1-W6)** — W1 monitor/profile/knowledge/settings → W2 system → W3 operations → W4 network → W5 duty/ad-domain → W6 workorder/vdi/asset；每文件原子迁移 + 测试文件同步 + gen 脚本 diff 入 commit（TEXT-02, MIG-01, MIG-02, MIG-04）
- [ ] **Phase 72: 后端 debug 修复 wave** — #3 确定性动作先行（写入方枚举 + 显式 normalize + 回归测试）→ #2 Cisco vendor 全链路 → #1-b 日志修复半段（DBG-02, DBG-03, DBG-04）
- [ ] **Phase 73: DB 实测日 + cron.go 双执行调查** — #5 场景 a-d 定位与修复 + #7 H1-H4 判定与数据修复 + #4 注册层调查与修复，验收 = sys_job_log 每 tick 双条消失（DBG-05, DBG-06, DBG-07）
- [ ] **Phase 74: COMP-05 收口 + lint 清偿第一波 + 流程积压归档** — P65/P66 豁免锚点清空 → 豁免块删除 → gen 脚本转 --dry 校验模式 + lint errors ≤ 83 脚本断言 + unused-vars/no-console 归零 + `--max-warnings N` 门禁 + any 降 50% + `.planning/quick/` 94 目录归属归档（MIG-03, HYG-01, HYG-02, HYG-03, HYG-04）
- [ ] **Phase 75: 浏览器实机 UAT + 收档** — 68.1 UI-SPEC §8 全表 19 条真实浏览器冒烟（重点 #13/15/16 + #2/3/7）+ 结果回写并带证据关闭 + debug 文档 Resolution 收尾（UAT-01, UAT-02）

## Phase Details

### Phase 69: 后端 debug 重验 spike + 信息收集下发

**Goal**: 把 7 项「acknowledge 结转」的 debug 逐份重读定级——按「根因已定位待修 / 调查中断需续查 / 需外部输入」三分类落盘，把需要外部信息的三项转成**用户可直接复制执行**的 SQL / 命令清单下发；同时把零代码的 #6 scrapligo panic 走完人工验证关单。本 phase 零生产代码改动。
**Depends on**: Nothing（v1.23 首个 phase；与前端线 Phase 70/71 全程并行）
**Requirements**: DBG-01
**Notes**: PITFALLS 3.1「acknowledge ≠ 已定位」是本 phase 存在的唯一理由——7 份文档中 ≥4 项的 Evidence/Eliminated/Resolution 节是空的或只有假设。**修复任务只对第一类下发**；#1/#7 在用户数据回填前禁止写修复代码（PITFALLS 3.4）。产出物：`debug/` 7 份文档 status 字段与定级一致、给用户的外部信息清单（#1 的 6 条 Linux 诊断命令 / #5 的 4 条 verification SQL / #7 的 §4.5 综合 SQL）、#2 方案就绪确认（模板 + 双命令 + 解析分支 + 导出白名单，验收基准锁定 `b022.7a2e.4a4f`）、#6 关单（scrapligo v1.4.2 已修 + 人工确认生产无复现 → archive + knowledge base）。#1 诊断半段与 #6 的归档索引在 Phase 75 随里程碑收档做最后确认（DBG-01 的 requirement 归本 phase，Phase 75 只做收尾核对，不重复计）。
**Success Criteria** (what must be TRUE):
  1. 运维人员拿到一份 7 项 debug 的定级清单，每项明确标注「根因已定位待修 / 调查中断需续查 / 需外部输入」之一；清单条目数与 `.planning/debug/` 文档数严格一致（7 = 7，无遗漏、无多列、无重复项）。
  2. 用户手上有一份**可直接复制执行**的外部信息清单：#1 的 6 条 Linux 诊断命令、#5 的 4 条 verification SQL、#7 的 §4.5 综合 SQL，三份清单各自独立可执行且注明回填位置。
  3. #2 Cisco 采集方案在动手前被确认可执行，验收基准「MAC `b022.7a2e.4a4f` 入库且格式统一」被写进后续修复的验收条款。
  4. scrapligo GetPrompt panic 经人工确认在生产环境不再复现后，其 debug 文档以「已修复 + 验证无复现」结案归档并进入知识库；本 phase **零生产代码改动**（`git diff` 只含规划文档）。
  5. 7 份 debug 文档中，每份的 status 字段与本轮定级结论一致；任何一份都不再停留在「已 acknowledge 结转但无结论」状态。

**Plans**: TBD
**UI hint**: no

### Phase 70: TextArea 契约基建 (W0)

**Goal**: 补上契约体系的最后一块——`AppTextArea` 独立具名导出（默认 `allowClear`），SearchForm 补 `textarea` field type，eslint 追加**成员表达式 + 解构形态双 selector**（同步进 65 契约块与 P68 hex 豁免块两处数组），并把**权威计数基线**落盘，为后续 6 个 phase 提供对账分母。
**Depends on**: Nothing 硬依赖（可与 Phase 69 并行）
**Requirements**: TEXT-01, TEXT-03, TEXT-04, HYG-05
**Notes**: 契约基建必须先于一切迁移页——SearchForm 是搜索区唯一渲染端点，若迁移先行而 SearchForm 不支持 textarea，搜索区字段只能塞 `custom.render` 逃逸口，产生 v1.22 要消灭的**第二路径**（ARCHITECTURE §4.2 W0 排序理由）。契约草案：`AppTextArea({ allowClear = true, ...rest }: ComponentProps<typeof Input.TextArea>)`，照抄 AppInput 16 行透传模式；**不挂 `AppInput.TextArea` 静态别名**（locked 决策）。eslint 陷阱：成员表达式 selector 写成 `JSXMemberExpression[object.name='Input'][property.name='TextArea']` 形态（不是 `name.name` 简单标识符），**追加进既有 no-restricted-syntax 数组**，并同步 P68 hex 豁免块重声明的完整数组。gen 脚本先跑 `--dry` 预登记存量命中文件，保证 selector 上线当天**零新增 error**。默认值白名单（是否默认 autoSize）须先对存量用法做 showCount/maxLength/受控/onChange 四象限审计再定稿（PITFALLS 1.2）。
**Success Criteria** (what must be TRUE):
  1. 开发者在任意表单弹窗用 AppTextArea 写备注/描述/原因，多行框出现**一键清空**入口，点击后内容立即清空——契约默认值生效，页面作者无需逐页补 prop。
  2. 开发者可在 SearchForm / PageTable 的搜索区声明一个多行文本字段，渲染为 AppTextArea 且同样具备清空能力；搜索区**不需要** `custom.render` 逃生口（不产生第二路径）。
  3. 团队成员写出裸 `<Input.TextArea>`（成员表达式）或解构形态 `<TextArea>` 时 **eslint 直接报错**；而 `src/components/ui/**` 封装层内的合法裸写法不报错（封装层豁免块保留）。
  4. 新 selector 上线当天，存量多行输入框在 lint 输出中**全部可见**（命中数与权威基线一致），不存在「lint 全绿但 grep 仍有命中」的假绿。
  5. **权威计数基线已落盘**（eslint selector 实跑命中数 + P65/P66 豁免清单条目数 + grep 粗估三数对照），作为 Phase 71/74 的对账分母；基线文件位置与后续 phase 引用路径在文档中写明。

**Plans**: TBD
**UI hint**: yes

### Phase 71: TextArea 存量迁移 + COMP-05 分域迁移 (W1-W6)

**Goal**: 把存量多行输入框与全部裸 `<Input>` / `<Select>` 迁进契约组件，**行为不变**，按 6 个域波次推进：W1 monitor/profile/knowledge/settings（低流量验证域）→ W2 system → W3 operations → W4 network → W5 duty/ad-domain → W6 workorder/vdi/asset。
**Depends on**: Phase 70（硬依赖：契约组件与 selector 必须先就位，否则存量无迁移压力信号）
**Requirements**: TEXT-02, MIG-01, MIG-02, MIG-04
**Notes**: 复用 Phase 67 已验证方法论：**P67 别名降险法**（`AppInput as Input`，只改 import 行）+ 每文件原子迁移 + 测试文件同步迁移 + gen 脚本 diff 入 commit（**禁手工改锚点数组**）。三个契约默认值的行为外溢逐页检查：①`allowClear` 出现在搜索框 → 有 `onPressEnter` 的搜索框确认清空不误触搜索；②`showSearch = count>=8` + 无条件 `optionFilterProp="label"` 对 children / fieldNames / labelInValue 形态是**破坏性**的 → 迁移 codemod 必须带**形态分类器**，children 与 fieldNames 形态单独决策、明确禁搜的业务下拉显式传 `showSearch={false}`；③`onChange(undefined)` 语义炸弹 → UUID 类字段（org_id / user_id / device_id）逐处 grep 防护，归一化在封装层一次做完。测试文件虽被 eslint 豁免，但必须**人工对账同步迁移**（PITFALLS 2.4），snapshot 变更只允许 clear icon 相关 DOM。
**Success Criteria** (what must be TRUE):
  1. 用户在系统任一表单弹窗的多行输入框（备注/描述/原因）都能**一键清空**，存量多行输入框全部接入 AppTextArea 契约，且清空按钮位置与触发时机一致。
  2. 全站单行输入框都具备清空能力且页面作者无需额外配置；业务代码中裸 `<Input>` 归零（以权威计数对账为准，不硬编码某个 grep 数）。
  3. 全站下拉框按形态分类正确：options 形态可搜索且过滤命中；children / fieldNames / labelInValue 形态的下拉**搜索不会过滤失效**；明确禁搜的业务下拉（状态/开关类）保持禁搜。
  4. **v1.22 冻结契约在迁移后仍然成立**：列表分页显示「共 N 条 / M 页」、危险删除只走 ConfirmDelete 唯一入口、失败提示只经 handleApiError 出口、状态与日期展示仍走 statusHelpers / datetime 常量、icon-only 按钮仍带 aria-label。
  5. 每个域波次完成后**权威计数对账一致**（该域裸组件 grep 命中数 = 豁免清单中该域剩余条目数），且没有任何文件处于半迁移态（同一文件不同时 import 契约组件与裸 antd 组件）。

**Plans**: TBD
**UI hint**: yes

### Phase 72: 后端 debug 修复 wave

**Goal**: 对 Phase 69 确认「根因已定位 / 方案就绪」的项落地修复，顺序严格为 **#3 → #2**：先在 `mac_collection_service.go` 上建立 MAC 格式写入防线（Go 侧），再叠加 Cisco 厂商采集扩展（同文件）。（2026-10-27 更新：原 #1-b Linux 日志修复已随 spike 回填取消——实测 app.log 正常落盘、不复现，DBG-04 已 Satisfied，剩余仅部署文档项。）
**Depends on**: Phase 69（spike 输出：#3 的确定性动作与 #2 的方案就绪确认；#1-b 的诊断结论已下发）
**Requirements**: DBG-02, DBG-03（DBG-04 已 Satisfied——2026-10-27 实测日志正常落盘、不复现，剩余部署文档项）
**Notes**: **#3 先于 #2**——同一文件 `mac_collection_service.go`，Cisco 双命令会改变写入行数与格式覆盖面，先落归一化防线再做厂商扩展，否则复发验证被新写入面污染（PITFALLS 3.2：R2 已实证单点修复会 R3 复发）。#3 方案 = **Go 侧防线**（全写入方枚举 → 显式 normalize → 写路径回归测试），**不用 DB TRIGGER**（locked 决策，维持 v1.19 路线）；`parseMACLine` 的 vendor switch 必须有 **default 分支显式报错 + 计数可观测**，把「静默丢弃」改成可观测失败。#2 = vendor 枚举 +1 + Cisco 双命令 + 解析分支（点分 `xxxx.xxxx.xxxx` 与 7 字段 port-security 行）+ 导出白名单同步（`network_export_handler.go` / `batch_export_handler.go`）+ 回归测试，验收用例 **`b022.7a2e.4a4f`**。#1-b = `pkg/logger/logger.go` + `cmd/main.go` 的文件侧双写 + 「启动即写文件」自检日志；HTTP 访问日志进 journald 不进 app.log 属**架构事实**，须在部署说明中写明。#3 的写路径回归测试落在 `internal/services/` 侧。
**Success Criteria** (what must be TRUE):
  1. 运维人员在 Cisco 设备上执行 MAC 采集，接口 MAC 正常入库且格式统一（验收用例 `b022.7a2e.4a4f` 通过）；未知厂商 / 未知格式**不再被静默丢弃**，而是产生可观测的失败记录与计数。
  2. 无论采集侧喂进什么原始格式（点分 / 连字符 / 无分隔），写入 `sys_device_mac_address` / `sys_device_mac_history` 的 MAC 一律为统一大写连字符格式、接口名不含空格；**全部写入方**都经过归一化（枚举条数 = grep 出的全部 INSERT/Save/BatchUpsert 调用点数），且存在写路径回归测试锁定这一不变量。
  3. `go build ./...` 与相关包 `go test` 全绿；MAC 采集相关既有 e2e / 单测零回归。（原 Linux 日志 SC 已移除——DBG-04 于 2026-10-27 spike 回填后 Satisfied，部署文档项随 Phase 75 收档补记。）

**Plans**: TBD
**UI hint**: no

### Phase 73: DB 实测日 + cron.go 双执行调查

**Goal**: 集中一次数据库实测环境，把 **#5 查询返回空**、**#7 物理链路为空** 两条线一次查清并修完——一次环境准备换两个清单。（2026-10-27 更新：原 #4 物化视图双执行调查已取消——用户确认根因 = 10.62.10.39 上存在第二套实例并行运行（双实例部署），非代码缺陷，DBG-05 已 Satisfied；部署约束「调度器单实例」已记录在 debug 文档。）
**Depends on**: Phase 72（#2/#3 落地后 MAC 数据可信，#7 复测才有判读力）+ Phase 69（SQL / 命令清单已下发并回填）
**Requirements**: DBG-06, DBG-07（DBG-05 已 Satisfied，见 Goal 更新说明）
**Notes**: **#5** = `mac_history_query_service.go:842-915` 的 Raw SQL → GORM chainable + 错误传播（空 → 500，不再吞 error）；执行场景 a-d 四组 verification SQL 判定根因；`sys_device_mac_history` 是 **PG 分区表**且 model 缺 `deleted_at`，改动前先跑基线 SQL 确认数据基线，半吊子加过滤会把存量数据「变没」。**2026-10-27 spike 首批回填实测**：分区树 2026_06~2026_12 共 7 个月度分区、EXPLAIN 剪枝正常（只扫 2026_06 索引，Index Actual Rows=6）、但样本 MAC `B0227A2E4A4F` 全表/分区 count 均为 0——「Rows=6 与 count=0 并存」不能排除查询字面量格式与存储格式不一致（正是原始疑点）；**执行本 phase 前需用户更换现存样本 MAC 复查 + 确认 deleted_at 列是否存在**。**#7** = 按 H1-H4 判读 §4.5 综合 SQL 结论后修数据，并做**可观测性增强**（区分「未绑定 user_id / 绑定错位 / 设备 orphan / 数据本身为空」四种空因），倾向单点数据修正而非 migration。（原 #4 cron.go 调查已取消——双实例部署根因，本 phase 不再触碰共享调度引擎。）
**Success Criteria** (what must be TRUE):
  1. 运维人员查询 MAC 历史轨迹能返回**真实结果**；DB 错误不再被吞成空结果（查询失败时接口返回可见错误而不是空数组），#5 场景 a-d 四组实测结论逐条回填（含换用现存样本 MAC 的复查结论与 deleted_at 列确认）。
  2. 工位 3F130 的「物理链路为空」能被区分出**具体空因**（未绑定用户 / 绑定错位 / 设备已删除 / 数据本身为空），运维人员看到的是可诊断结论而不是空白。
  3. 两条线的实测结论（#5 场景判读 / #7 H1-H4 命中项）全部回填进对应 debug 文档的 Eliminated 与 Resolution 节，无一份仍是空白。

**Plans**: TBD
**UI hint**: no

### Phase 74: COMP-05 收口 + lint 清偿第一波 + 流程积压归档

**Goal**: 拆除 eslint 豁免脚手架、偿还 lint 债——让契约规则对全 `src/` 真正生效，并给 lint 装上防回退门禁。（2026-10-27 更新：原 `.planning/quick/` 94 个历史目录清算已提前完成——HYG-01 Satisfied，92 个按日期窗口+主题归入对应里程碑 `<version>-phases/quick/`，2 个 2025 年目录归 `milestones/obsolete-quick/` 附理由，零删除，清单见 `.planning/notes/2026-10-27-quick-archive.md`。）
**Depends on**: Phase 71（收口必须建立在迁移完成之上；lint 排在迁移后是硬排序约束——迁移波次自身会产生新的 warn 峰值）
**Requirements**: MIG-03, HYG-02, HYG-03, HYG-04（HYG-01 已 Satisfied——2026-10-27 提前完成归档）
**Notes**: **收口 = 拆脚手架，不是改级别**——裸组件 selector 本来就是 `error` 级，存量页不报错是因为 P65/P66 豁免 override 块把它们整条重声明掉了；收口动作 = P66 锚点 files 数组清空 → 整块删除 → P65 锚点 files 数组清空 → 整块删除 → gen 脚本转 **--dry 校验模式**（连跑两次 diff 为空，68.1 §8 #18 幂等口径）。**保留**：封装层豁免块（`src/components/ui|shared`）+ 测试文件豁免块 + P68 hex 豁免块（收缩后）+ 65 契约块。lint 清偿**按规则分组、禁全量 `--fix`**（`--fix` 会删掉 API 形态文档性死代码、`any→unknown` 引发类型蔓延再撒 `as any`、fixer 可能击穿 83 errors 线）：机械安全规则先行 → `no-unused-vars` + `no-console` 归零（≈368 条）→ `no-explicit-any` 软目标降 50%（958 → <500，分两 wave = 类型文件 wave + 组件边界 wave，chart/ECharts 边界不强行清零）→ `--max-warnings N` 门禁进 npm script。**errors ≤ 83 必须写成脚本断言**（对比 lint 输出计数），不靠人眼。`.planning/quick/` 94 个历史目录（全部早于 v1.22）逐个判定归属：归档到对应里程碑或标记 OBSOLETE，**不手工删**。
**Success Criteria** (what must be TRUE):
  1. 全 `src/` 范围内不再存在 P65/P66 存量豁免块，脚手架移除后契约规则对新代码持续生效——开发者新写裸 `<Input>` / `<Select>` / `<TextArea>` 会**立刻**被 lint 拦下。
  2. `npm run lint` 的 error 数不高于 83 基线（**脚本断言**，非人眼），`no-unused-vars` 与 `no-console` 命中归零，`no-explicit-any` 相对权威基线下降 ≥ 50%，`npm run type-check` 全绿。
  3. npm script 携带 `--max-warnings N` 门禁：warnings 超预算时本地/CI 直接失败——lint 债务不再静默回退。
  4. **收口对账报告落盘**：权威计数与 Phase 70 基线的差异被逐项解释（每一条差异都有对应迁移动作或豁免理由），差异条数与豁免清单剩余条目数能够相互印证。（原 SC「quick 目录归属」已随 HYG-01 提前完成而移除）

**Plans**: TBD
**UI hint**: yes

### Phase 75: 浏览器实机 UAT + 收档

**Goal**: 在**真实浏览器**里把 68.1 UI-SPEC §8 全表 19 条跑完并带证据关闭，把 debug 文档的 Resolution 收尾，让里程碑以「无债、无盲区」收档。
**Depends on**: Phase 74（硬排序：UAT 必须测**迁移 + 收口后**的最终页面）+ Phase 72/73（debug 文档 Resolution 回填）
**Requirements**: UAT-01, UAT-02
**Notes**: **前置清单（三项缺一不可）**：①`npm run build && npm run preview` 的 **prod build 产物**（dev server 测不到 vendor chunk 的懒加载/TDZ 类问题）；②`VITE_ENABLE_REQUEST_ENCRYPTION` 按**生产配置开启**（否则真实用户路径 401→refresh→重放测不到）；③**清空 sessionStorage** 后验证 usePagination 的持久化恢复。工具用环境已有的 **agent-browser skill**，**不引入 Playwright**（零新依赖原则）。重点条目：#13 分页口径 / #15 固定列不失效 / #16 行操作权限不变 / #2·#3·#7 提交中行为（确认按钮 loading、提交中弹窗不可关闭、提交中取消按钮禁用）——**以 UI-SPEC 原文为准**（seeds 把 15/16 误记为「分页 + 批删」，实为「固定列不失效」「行操作权限不变」；分页是 #13、批量删除是 #5/#6）。产物沿用 HUMAN-UAT 文档惯例，每条带截图 + build 版本/commit 标识 + 复现步骤。Linux 侧至少一次冒烟（本里程碑同修 Linux 日志项）。收尾动作：结果回写 68.1 UI-SPEC §8 并关闭该项债务；7 份 debug 文档的 Resolution 节与实际修复状态一致（DBG-01 的 #6 已于 Phase 69 关单归档，此处只做归档索引确认；#3/#2/#1/#5/#7/#4 各有结论）。
**Success Criteria** (what must be TRUE):
  1. 68.1 UI-SPEC §8 全表 **19 条**在真实浏览器（prod build 产物 + 请求加密按生产配置开启 + 清空 sessionStorage 后）逐条通过，重点 #13 / #15 / #16 / #2·#3·#7 全部实测；结果表无空白行（19 = 通过 + 显式失败，无「未测」）。
  2. 每条 UAT 记录带**可核查证据**（截图 + build 产物 commit/版本标识 + 复现步骤），不留「一句话带过」的记录；失败项要么当场修复并重测，要么显式转为带 owner 的新债并登记在案。
  3. 结果**回写 68.1 UI-SPEC §8** 回归清单并标记该项债务关闭——§8 中不再有「仅 jsdom 覆盖、未做浏览器实机验证」的无主结转条目。
  4. 7 份 debug 文档的 Resolution 节与实际修复状态一致，没有任何一份仍停在「acknowledge 结转」或空白状态；里程碑以「无债、无盲区」收档。

**Plans**: TBD
**UI hint**: yes

---

### Progress Table (v1.23)

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 69. 后端 debug 重验 spike + 信息收集下发 | 0/0 | ⬜ Not Started | — |
| 70. TextArea 契约基建 (W0) | 0/0 | ⬜ Not Started | — |
| 71. TextArea 存量迁移 + COMP-05 分域迁移 (W1-W6) | 0/0 | ⬜ Not Started | — |
| 72. 后端 debug 修复 wave (#3→#2→#1-b) | 0/0 | ⬜ Not Started | — |
| 73. DB 实测日 + cron.go 双执行调查 (#5/#7/#4) | 0/0 | ⬜ Not Started | — |
| 74. COMP-05 收口 + lint 清偿第一波 + 流程积压归档 | 0/0 | ⬜ Not Started | — |
| 75. 浏览器实机 UAT + 收档 | 0/0 | ⬜ Not Started | — |

### Coverage Map (v1.23)

```
TEXT-01 → Phase 70   TEXT-02 → Phase 71   TEXT-03 → Phase 70   TEXT-04 → Phase 70
MIG-01  → Phase 71   MIG-02  → Phase 71   MIG-03  → Phase 74   MIG-04  → Phase 71
DBG-01  → Phase 69   DBG-02  → Phase 72   DBG-03  → Phase 72   DBG-04  → Phase 72
DBG-05  → Phase 73   DBG-06  → Phase 73   DBG-07  → Phase 73
UAT-01  → Phase 75   UAT-02  → Phase 75
HYG-01  → Phase 74   HYG-02  → Phase 74   HYG-03  → Phase 74   HYG-04  → Phase 74
HYG-05  → Phase 70

Mapped: 22/22 ✓  No orphans, no duplicates
```

---

## Archive: Shipped Milestones

<details>
<summary>✅ v1.22 前端公用组件整治 (Phases 63-68.1) — SHIPPED 2026-10-27</summary>

### Phases (v1.22) — 7 phases / 7 plans（SHIPPED 2026-10-27）

**Milestone goal:** 消化 2026-10-23 全量前端组件审查的 201 条问题（P0 24 / P1 87 / P2 90），把公共层从"只有零件没有契约"补成"有契约"：默认值下沉（allowClear/showSearch）、分页契约下沉（showTotal/setTotal）、危险操作唯一入口（ConfirmDelete）。

- [x] **Phase 63: P0 止血 — 数据可达与安全** — showTotal 分页口径修正 / backups·discoveries setTotal 断链 / OU 服务端分页 / 批量删除二次确认 / 假功能入口清理 / fixed 列与 scroll.x 配对（STOP-01..07）
- [x] **Phase 64: P1 修复 — 反馈与可访问性** — Modal 脏检查 / 表格 loading / 服务端排序 / console 收敛到 handleApiError / icon-only aria-label（FIX-01..05）
- [x] **Phase 65: 基础契约组件 — 默认值与危险操作唯一入口** — AppInput / AppSelect / ConfirmDelete（COMP-01, COMP-02）
- [x] **Phase 66: 高级契约组件 — PageTable 与 FormModal** — PageTable / SearchForm / FormModal / RemoteSelect（COMP-03, COMP-04）
- [x] **Phase 67: 存量迁移 — 按 ROI 顺序落地契约组件** — discoveries → backups → devices/cache（COMP-05）
- [x] **Phase 68: P2 一致性债务 — 颜色/日期/类型收敛** — design token / statusHelpers / datetime / any 收敛（DEBT-01..03）
- [x] **Phase 68.1: 收尾清偿** — 68 处危险操作收口到 ConfirmDelete 唯一入口 / 7 项契约 minor 补课 / 豁免清单 181→159 且 100% 带理由 / operations 4 页迁移（DEBT-04）

**留转 v1.23:** TextArea 零 allowClear 覆盖（用户主诉根因）+ 裸 Input/Select 存量迁移未完成（P65/P66 豁免块仍在）+ 7 项 debug acknowledge（≥4 项调查未完成）+ 浏览器实机 UAT 债 + `.planning/quick/` 94 个历史目录 + lint 1367 warnings / 83 errors。

**Full archive**: [milestones/v1.22-ROADMAP.md](milestones/v1.22-ROADMAP.md) | [milestones/v1.22-MILESTONE-AUDIT.md](milestones/v1.22-MILESTONE-AUDIT.md) | [milestones/v1.22-REQUIREMENTS.md](milestones/v1.22-REQUIREMENTS.md)

</details>

<details>
<summary>✅ v1.21 AD 双向同步 bug 修复 (Phases 57-62) — SHIPPED 2026-10-22</summary>

### Phases (v1.21) — 6 phases / 22 plans / 45 tasks（SHIPPED 2026-10-22）

**Milestone goal:** 修复 AD↔系统双向同步的 2 个 P0 bug（部门回退乒乓、软删用户接管风险）、3 个 P1 数据一致性缺口和 P2 卫生项，以三方审计修订后的 outbox-lite 架构落地，消除"改了部门又弹回"与静默数据丢失。

- [x] **Phase 57: AD 推送队列基建** — outbox-lite 队列表 + 同事务入队 + 按实体合并 + 删除入队 disable (QUEUE-01/02/06, HYGIENE-06) — completed 2026-10-16 (3/3 plans, code review F-01~F-06 修复 + verifier GAP-01/02/03 闭环, [VERIFICATION](milestones/v1.21-phases/57-ad-outbox-lite/57-VERIFICATION.md) human_needed: 3 项真环境人工验证在 [57-HUMAN-UAT.md](milestones/v1.21-phases/57-ad-outbox-lite/57-HUMAN-UAT.md))
- [x] **Phase 58: 排水器与 compare-and-move 推送** — in-process cron 排水 + 退避重试 + 幂等 OU 移动 + 配置路由无歧义 (HYGIENE-01/02, QUEUE-03/04/05) — 7/7 plans，completed 2026-10-19（SC1~6 全部 VERIFIED，见 [58-VERIFICATION.md](milestones/v1.21-phases/58-compare-and-move/58-VERIFICATION.md)）
- [x] **Phase 59: 登录回写守卫与部门对账视图** — AD 登录不再回滚本地编辑 + dept↔OU 不一致可观测 (GUARD-01/02) — 4/4 plans，completed 2026-10-19（两条 Manual-Only 视觉断言 + PG 真实启动冒烟归 [59-VALIDATION.md](milestones/v1.21-phases/59-ad-login-guard-reconciliation/59-VALIDATION.md)）
- [x] **Phase 60: 软删恢复安全** — ad_dn 校验后恢复 + 登录/批量同步共用分类逻辑 (SAFE-01/02) — 2/2 plans，completed 2026-10-20（12/12 must-haves VERIFIED，见 [60-VERIFICATION.md](milestones/v1.21-phases/60-soft-delete-recovery-safety/60-VERIFICATION.md) human_needed: 3 项真环境人工验证在 [60-UAT.md](milestones/v1.21-phases/60-soft-delete-recovery-safety/60-UAT.md)）
- [x] **Phase 61: 影子表完整性** — 删除清理两道闸门 + 组成员事务化 + singleflight 挂载 (SHADOW-01/02/03) — 3/3 plans，**Complete 2026-10-22**（8/8 must-haves VERIFIED，61-UAT 4/4 pass；真环境人工项见 [61-UAT.md](milestones/v1.21-phases/61-shadow-table-integrity/61-UAT.md) 与 [61-VERIFICATION.md](milestones/v1.21-phases/61-shadow-table-integrity/61-VERIFICATION.md) status=passed）
- [x] **Phase 62: 卫生收尾与死代码清理** — F-22 旧路径删除 + 嵌套组标记 + 同名部门 OU DN 冲突告警 (HYGIENE-03/04/05) — 3/3 plans，completed 2026-10-22（18/18 must-haves VERIFIED，HYGIENE-03 死代码 grep 残留=0 / HYGIENE-04 嵌套组过滤零 DDL + 生产 8 行污染自愈 / HYGIENE-05 OU DN 冲突 typed error 显式化）

### Progress Table (v1.21)

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 57. AD 推送队列基建 | 3/3 | Complete | 2026-10-16 |
| 58. 排水器与 compare-and-move 推送 | 7/7 | Complete（2026-10-22 全收口） | 2026-10-22 |
| 59. 登录回写守卫与部门对账视图 | 4/4 | Complete（verification passed） | 2026-10-22 |
| 60. 软删恢复安全 | 2/2 | Complete（verification passed） | 2026-10-22 |
| 61. 影子表完整性 | 3/3 | Complete（VERIFICATION status=passed） | 2026-10-22 |
| 62. 卫生收尾与死代码清理 | 3/3 | Complete（VERIFICATION status=passed） | 2026-10-22 |

</details>

<details>
<summary>✅ v1.20 网络设备 VLAN + 端口绑定 (Phase 56) — SHIPPED 2026-07-10</summary>


### Phase 56: 网络设备 VLAN + 端口绑定 (v1.20.1) — 5/5 plans, shipped 2026-07-10

**Goal:** 在 v1.19 端口写命令 MVP 基础上扩展 2 个新写命令：① 修改 access VLAN ② IP+MAC+Port 静态绑定。复用 v1.19 vendor 模板 / operlog / 权限 / 批量 / e2e 全套基建。

**Requirements**: 22 (VLAN-01~06, BIND-01~07, INFRA-01~04, UI-01~06, TEST-01~05)

**Plans**: 5 plans (5 waves)

- [x] 56-01-PLAN.md — vendor 模板新增 (set_access_vlan + port_binding × 3 vendors) + 12+ 单元测试
- [x] 56-02-PLAN.md — pre_state_check + port_write_service 新增 2 方法 + 4 校验器 + audit JSONB
- [x] 56-03-PLAN.md — 6 kebab 端点 + 6 handler + operlog OperType 分流 + `network:port:write` 权限复用
- [x] 56-04-PLAN.md — 2 types + 2 networkApi wrappers + SetAccessVlanModal + PortBindingModal + ports/index.tsx
- [x] 56-05-PLAN.md — 11 TestE2E_* + 10 fixtures + 56-HUMAN-UAT.md + 文档同步

45 commits / 51 files / +10,961/-104 LOC。详见 [milestones/v1.20-ROADMAP.md](milestones/v1.20-ROADMAP.md)。

</details>

<details>
<summary>✅ v1.19 网络设备写命令 (Phases 50-54) — SHIPPED 2026-07-08</summary>

- [x] Phase 50: W1 Vendor Templates + Unit Tests — 1/1 plan — completed 2026-07-06
- [x] Phase 51: W2 PortWriteService + Batch + Mock Tests — 1/1 plan — completed 2026-07-06
- [x] Phase 52: W3 Router/Handler/Operlog/Permission/Migration — 2/2 plans — completed 2026-07-07
- [x] Phase 53: W4 Frontend Drawer + Progress Dialog + API Wrappers — 2/2 plans — completed 2026-07-07
- [x] Phase 54: W5 E2E + Real-Device UAT + Documentation — 1/1 plan — completed 2026-07-07
- [x] Phase 55: Tech-Debt Cleanup (Phase 53 leftover sweep) — 2/2 plans — completed 2026-07-08

See [milestones/v1.19-ROADMAP.md](milestones/v1.19-ROADMAP.md) for archived details (108 files / 25,224 insertions / 3,152 deletions / 1.7 days / 37/37 requirements).

</details>

<details>
<summary>✅ v1.18 网络设备硬件清单 (Phase 48) — SHIPPED 2026-07-04</summary>

### Phase 48: 网络设备组件序列号采集 (Device Component Serials) — 3/3 plans, shipped 2026-07-04

**Goal**: 支持"一机多序列号"——将网络设备的板卡/引擎卡、电源、风扇、光模块各自的序列号**作为资产设备纳入资产系统 `ops_asset`**，保存组件从属关系并在前端展示。

**Plans**: 3 plans

- [x] 48-01-PLAN.md — Schema 迁移(migration_201) + model 扩展 + component_type IS NULL 过滤
- [x] 48-02-PLAN.md — component_collector 包(SNMP ENTITY-MIB + CLI 双厂商 + 6 TextFSM 模板)
- [x] 48-03-PLAN.md — OpsAssetWriter + ReconciliationEmitter + Pipeline + 前端组件清单 Tab + UAT 推迟声明

Phase 49 gap closure (2/2 plans, 2026-07-05) — enrichChassisSerial 接入 + collectBoardsInto 接入, 14 台设备 serial_number 回填。

</details>

<details>
<summary>✅ v1.17 资产对账 (Phases 42-46) — SHIPPED 2026-07-03</summary>

- [x] Phase 42: 资产对账观测底座 (R1) — 5/6 plans (42-03 skeleton superseded by R3)
- [x] Phase 43: 告警 + 工单闭环 (R2) — 3/3 plans
- [x] Phase 44: 置信度评分 + IP 例外规则 (R3) — 2/2 plans
- [x] Phase 45: 工位详情整合 (R4) — 2/2 plans
- [x] Phase 46: 资产对账闭环 + 审计 (R5, 可选) — 2/2 plans (UAT 9/10 PASS)
- [x] Phase 47: 资产对账根因修复 — 2/2 plans (2026-07-03)

See [milestones/v1.17-ROADMAP.md](milestones/v1.17-ROADMAP.md) for details.

</details>

<details>
<summary>Pre-v1.17 Phase History (Phases 1-41)</summary>

### Phase Summary (cumulative)

- ✓ Phases 1-11 — v1.0–v1.4 基础功能 (27/27 plans)
- ✓ Phases 12-15 — v1.5 MAC地址历史数据管理 (26/26 plans)
- ✓ Phase 16 — v1.6 API密钥管理 (10/10 plans)
- ✓ Phase 17 — v1.7 加密配置同步 (6/6 plans)
- ✓ Phase 18 — v1.8 登录端加密 (4/4 plans)
- ✓ Phases 19-20 — v1.9 AD域控集成 (11/11 plans)
- ✓ Phase 21 — v1.10 网络设备权限 (1/1 plan)
- ✓ Phases 22A/22B — v1.12 深信服 VDI (6/6 plans)
- ✓ Phase 23 — v1.11 AD组自动同步 (6/7 plans, 23-13 superseded)
- ✓ Phase 24 — UUID一致性优化 (2/2 plans)
- ✓ Phase 25 — VM 数据范围权限 (5/5 plans)
- ✓ Phase 26 — v1.13 资产管理 (6/6 plans)
- ✓ Phase 27 — v1.14 全局列自定义 (1/1 plan)
- ✓ Phase 28 — v1.15 工位设备关联 (4/4 plans)
- ✓ Phase 30 — 前端性能优化 (5/5 plans, 25/25 must-haves)
- ✓ Phase 31 — P0 收尾 (5/5 plans, F-14 + F-17)
- ✓ Phase 32 — v1.14 P1 重构 + P2 架构优化 (7/7 waves, 15 P1 + 8 P2)
- ✓ Phase 33 — Vercel React Best Practices (5/5 waves, 26 defects)
- ✓ Phase 34 — 操作日志全模块集成 (11/10 plans, 100% coverage 298/298)
- ✓ Phase 37 — 前端部门选择组件统一收敛 (6/6 plans, 10 migration points)
- ✓ Phase 38 — AD 账号池统一 (4/4 plans)
- ✓ Phase 39 — v1.15 部门物理位置映射 (6/6 plans)
- ✓ Phases 40-41 — v1.16 技术债清理 (8 plans)
- ✓ Phases 42-46 — v1.17 资产对账 (14/16 plans, 42-03 superseded)
- ✓ Phase 47 — 资产对账根因修复 (2/2 plans)
- ✓ Phase 48 — v1.18 网络设备组件 (3/3 plans)
- ✓ Phase 49 — v1.18 gap closure (2/2 plans)
- ✓ Phases 50-55 — v1.19 网络设备写命令 (9/9 plans, 6 phases)
- ✓ Phase 56 — v1.20 VLAN + 端口绑定 (5/5 plans)

**Total shipped**: 162 plans across 56 phases (Phases 1-56) + v1.21 Phases 57-62 (22 plans) + v1.22 Phases 63-68.1 (7 plans)

</details>

---

*Last updated: 2026-10-27 — v1.23 前端契约全量收尾与遗留清偿 roadmap created (Phases 69-75, 22/22 REQ coverage: TEXT×4 + MIG×4 + DBG×7 + UAT×2 + HYG×5)。上一里程碑 v1.22 shipped 2026-10-27 (Phases 63-68.1, 20/20 REQ satisfied)。*
