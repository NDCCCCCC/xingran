# Milestone v1.23 Requirements — 前端契约全量收尾与遗留清偿

> **Defined**: 2026-10-27
> **Core Value**: 把 v1.22 建立但未落地的契约真正推到全量——修复用户主诉的「文本输入框无法一键删除」，并清偿 v1.22 收档结转的 debug / UAT / 流程积压，让里程碑以「无债、无盲区」收档。
>
> **事实基线**: `.planning/research/{STACK,FEATURES,ARCHITECTURE,PITFALLS,SUMMARY}.md`（4 维并行研究，overall confidence HIGH，commit `b5b3aeaf`）+ `.planning/v1.23-seeds.md` + `.planning/debug/` 7 份 debug 文档。

## 用户主诉与根因

> 「v1.22 完成后我最开始的问题，文本输入框无法一键删除等细节问题还是没有解决」

代码核实结论（2026-10-27 实测）：

| 项 | 实测 | 判定 |
|----|------|------|
| 单行裸 `<Input>` | 157 处（58 文件）**全部显式带 allowClear** | v1.22 逐页补齐已生效，非主诉根因 |
| `<Input.TextArea>` / `<TextArea>` | **74 处，0 处有 allowClear** | **主诉根因** |
| `components/ui/AppInput.tsx` | 16 行，仅包单行 `Input`，**无 TextArea 处理** | 契约盲区 |
| `eslint.config.js:114` | 注释明确 `Input.TextArea` 成员表达式「有意不拦」 | **零约束盲区，验收假绿** |
| `AppInput`/`AppSelect` 在 `src/pages/` 使用量 | 15 / 13 处 | COMP-05 只做试点页，存量大头未迁 |

## v1.23 Requirements

### TEXT — TextArea 一键清空契约（核心）

- [ ] **TEXT-01**: 用户在任意表单弹窗的多行输入框（备注/描述/原因）能一键清空内容，无需手动逐字删除
- [ ] **TEXT-02**: 存量 74 处多行输入框全部接入 `AppTextArea` 契约，行为一致（清空按钮位置与触发时机统一）
- [ ] **TEXT-03**: 搜索区（`SearchForm`/`PageTable`）支持多行输入字段类型，契约默认值无需逐页补
- [ ] **TEXT-04**: 裸多行输入框被 eslint 拦截（成员表达式 `<Input.TextArea>` + 解构形态 `<TextArea>` 双覆盖），杜绝新增盲区

### MIG — COMP-05 全量收尾（契约组件迁移）

- [ ] **MIG-01**: 存量裸 `<Input>` 全部改用 `AppInput` 契约（零 JSX 差异获得 allowClear 等默认值）
- [ ] **MIG-02**: 存量裸 `<Select>` 全部改用 `AppSelect` 契约
- [ ] **MIG-03**: 迁移完成后 P65/P66 豁免清单清空、豁免块移除，eslint 契约规则对全 `src/` 生效
- [ ] **MIG-04**: 迁移不破坏 v1.22 冻结契约（分页口径「共 N 条/M 页」/ `ConfirmDelete` 唯一危险操作入口 / `handleApiError` 唯一错误出口 / `statusHelpers`+datetime 常量 / hex-exemptions 登记制）

### DBG — Debug 遗留清偿（7 项，含重验）

> ⚠️ 经逐份核读，**≥4 项调查实际未完成**（acknowledge ≠ 已定位）。第一波必须是重验 spike，而非直接修复。

- [x] **DBG-01**: scrapligo GetPrompt nil panic 经人工验证无复现后关单（✅ 2026-10-27 用户确认生产无复现，debug 文档 closed）
- [ ] **DBG-02**: Cisco 设备 MAC 地址采集支持厂商模板（双命令 + `parseMACLine` Cisco 分支 + 导出白名单；验收用例 `b022.7a2e.4a4f`）
- [ ] **DBG-03**: MAC 地址格式问题不再复发——写入方枚举 + 显式 normalize + 回归测试（**Go 侧防线，不用 DB TRIGGER**，维持 v1.19 决策）
- [x] **DBG-04**: Linux 环境应用日志正常落盘（✅ 2026-10-27 实测不复现：app.log 65MB 持续在写；误判根源 = 进程非 systemd 托管，journalctl 为空属预期；降级为部署文档项）
- [x] **DBG-05**: MAC 历史迁移任务不重复执行（✅ 2026-10-27 根因确认 = 10.62.10.39 上有第二套实例并行运行（双实例部署共用同一 DB 调度器），非代码缺陷；部署约束「调度器单实例」已记录，多实例 HA 需分布式锁列为 future enhancement）
- [x] **DBG-06**: MAC 历史查询返回正确结果（✅ 2026-10-27 修复：根因 = 查询侧 normalize 无分隔 vs 存储大写冒号，等值永空；改为**双侧归一化**双分支 SQL `mac_address = ? OR REPLACE(...)=?`（PG/SQLite 通用，分支一走 canonical 索引）+ 5 处等值匹配改造 + Raw SQL 错误传播（不再吞 error 成空数组）+ migration_175 的 deleted_at 注释更正。注意：`QueryMACTrajectory` 已在 migration_189 中随前端页面移除，根因同源落到现存 5 处查询路径。commit `da984927`）
- [x] **DBG-07**: 工位 3F130 物理链路为空可区分空因状态（✅ 2026-10-27 修复：根因 = H3 未绑 user_id；**纯增量**新增 `POST /ops/workstation-device/:id/physical-diagnosis` 端点返回 `emptyReason`/`emptyMessage`/`warnings`/`diagnosis` 四类空因 + 十项链路诊断计数，旧端点行为完全不变（向后兼容）。设计决策：`no_user_bound` 优先级压过「有数据」（因 3f130 实测 mac_match_count=1 但前端按用户维度呈现仍空白）。commit `8b85fe6b`；前端接入留 Phase 75）

### UAT — 浏览器实机验证

- [ ] **UAT-01**: 68.1 UI-SPEC §8 回归清单 19 条在真实浏览器全部通过（重点 #13 分页 / #15 固定列不失效 / #16 行操作权限不变 / #2·3·7 危险操作）
- [ ] **UAT-02**: UAT 结果回写 68.1 UI-SPEC §8 并带证据，债务不再无主结转

> ⚠️ seeds 编号勘误：§8 第 15/16 条实际是「固定列不失效」「行操作权限不变」，**分页是第 13 条、批量删除是第 5/6 条**——以 UI-SPEC 原文为准。

### HYG — 流程积压清偿

- [x] **HYG-01**: `.planning/quick/` 94 个历史任务目录完成归属判定与归档（✅ 2026-10-27 完成：92 个按日期窗口 + 主题归属到对应里程碑 `<version>-phases/quick/`，2 个 2025 年目录（早于项目起点 2026-04-16）归 `milestones/obsolete-quick/` 并附 README 理由，零删除；清单见 `.planning/notes/2026-10-27-quick-archive.md`，commit `135bcab4`）
- [ ] **HYG-02**: lint `no-unused-vars` + `no-console` 归零（≈368 条机械可修）
- [ ] **HYG-03**: lint `no-explicit-any` 降低 50%（958 → <500；三档：类型文件机械修 / chart 边界需判断 / 动态结构豁免）
- [ ] **HYG-04**: lint `--max-warnings` 门禁纳入 npm script 防回退
- [ ] **HYG-05**: 权威计数基线落盘（eslint selector 命中数 + 豁免清单条目数），作为收口对账依据

## 关键决策 (v1.23 init, locked)

| 决策 | 理由 |
|------|------|
| `AppTextArea` **独立具名导出**，不挂 `AppInput.TextArea` 静态别名 | 成员表达式 selector 无法区分合法契约用法与非法裸用法；挂别名会永久放弃成员表达式拦截面，74 处收口假绿 |
| 排期口径 = **eslint selector 实跑命中数 + 豁免清单条目数**（权威），grep 数仅供粗估 | 研究文档间口径 157-220 不一致（grep 行数与 JSX 节点数口径不同） |
| DBG-03 走 **Go 侧防线**，不用 DB TRIGGER | 维持 v1.19「禁用 DB TRIGGER 路线」决策；规避 SQLite 双实现成本与数据流隐式化 |
| 零新依赖 | 前端不引 jscodeshift/Playwright，UAT 用已有 agent-browser skill；后端 scrapligo v1.4.2 现状即可 |
| 排序约束：迁移 → 收口 → lint → UAT | UAT 必须测**迁移后**的页面；lint 排在迁移后（迁移波次自身产生新 warn 峰值） |
| 后端 debug 线与前端线可全程并行 | 两线无共享文件 |

## Future Requirements（延后，不在 v1.23 路线图）

- **react-refresh 33 处清零** — 需拆分文件，v1.23 迁移已在动大量前端文件，叠加拆分会使 diff 难审、回归面翻倍；该规则不影响运行时行为
- **devices 页 692 行拆分** — v1.22 Phase 68.1 已记为设计决策（拆分风险 > 收益，追加 ~2h）；迁移本身要改该文件，同 PR 混合改名 + 拆分难以定位回归
- **三态组件（Empty/Skeleton/ErrorAlert）推广** — 当前使用量 0，PageTable 已内建空态/错误态间接覆盖
- **lint 剩余 `no-explicit-any` 归零** — any→unknown 属行为变更，本轮只做 50% 降量
- **`react-refresh` 升 error** — 收口时按实际余量裁
- **AD 推送队列 enable 方向**（`.planning/seeds/SEED-ad-push-enable-direction.md`）、**全量同步节流**、**同步漂移告警产品化**、**v1.18/1.19/1.20 现场 UAT 债**

## Out of Scope

| Feature | Reason |
|---------|--------|
| 业务语义组件重设计（资产选择器、部门树选择器） | v1.22 已锁定「抽契约不抽业务」，本轮不重开 |
| 巨型文件主动拆分重构 | 同上，仅随迁移顺带 |
| i18n | `useTranslation` 0/全片，另行立项 |
| 引入 Playwright / jscodeshift 等新工具链 | 零新依赖原则；迁移用 Phase 67 别名降险先例，UAT 用 agent-browser skill |
| 新增后端依赖 | 7 项 debug 修复经研究确认零新依赖 |
| 改名 v1.22 冻结契约 | 分页口径 / ConfirmDelete 唯一入口 / handleApiError 唯一出口 / statusHelpers 均为 v1.22 锁定契约，迁移不得破坏 |
| ~~修改 `sys_job` schema~~ | ✅ 争议消失：#4 双执行根因确认为双实例部署（非代码），无需 schema 变更（2026-10-27 用户裁定） |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| TEXT-01 | 70 | Pending |
| TEXT-02 | 71 | Pending |
| TEXT-03 | 70 | Pending |
| TEXT-04 | 70 | Pending |
| MIG-01 | 71 | Pending |
| MIG-02 | 71 | Pending |
| MIG-03 | 74 | Pending |
| MIG-04 | 71 | Pending |
| DBG-01 | 69 | Satisfied (2026-10-27) |
| DBG-02 | 72 | Pending |
| DBG-03 | 72 | Pending |
| DBG-04 | 72 | Satisfied (2026-10-27) |
| DBG-05 | 73 | Satisfied (2026-10-27) |
| DBG-06 | 73 | Satisfied (2026-10-27) |
| DBG-07 | 73 | Satisfied (2026-10-27) |
| UAT-01 | 75 | Pending |
| UAT-02 | 75 | Pending |
| HYG-01 | 74 | Satisfied (2026-10-27) |
| HYG-02 | 74 | Pending |
| HYG-03 | 74 | Pending |
| HYG-04 | 74 | Pending |
| HYG-05 | 70 | Pending |

**Coverage:**
- v1.23 requirements: 22 total
- Mapped to phases: 22（TEXT×4 → 70/71/70/70 · MIG×4 → 71/71/74/71 · DBG×7 → 69/72/72/72/73/73/73 · UAT×2 → 75/75 · HYG×5 → 74/74/74/74/70）
- Unmapped: 0 ✓（无孤儿、无重复；每条需求恰好映射到一个 phase）

**Phase 归属说明（口径）:**
- **DBG-01**（scrapligo 关单）归 **Phase 69**：重验 phase 本身就是「重新核验」的动作，#6 零代码、只需人工确认生产无复现即可关单；Phase 75 只做归档索引与收档核对，不重复计入。
- **DBG-04**（Linux 日志）归 **Phase 72**：代码侧修复半段（文件侧双写 + 启动自检）落在修复 wave；诊断半段（6 条 Linux 诊断命令）在 Phase 69 下发、Phase 75 回填文档，不构成独立需求。
- **HYG-05**（权威计数基线）归 **Phase 70**：基线必须在迁移开始**之前**落盘才有对账意义——Phase 70 是 TextArea selector 上线日，也是 lint/豁免计数基线快照点；Phase 74 用它做收口 delta 对账。
- **TEXT-01** 归 **Phase 70**（契约提供默认清空能力），**TEXT-02** 归 **Phase 71**（存量 74 处接入），二者合起来才是用户主诉的完整闭环。

---
*Requirements defined: 2026-10-27*
*Last updated: 2026-10-27 after Phase 69 spike 回填批次 1（DBG-01/04/05 Satisfied：#6 关单、#1 不复现、#4 双实例部署根因；DBG-02/03 → Phase 72、DBG-06/07 → Phase 73 范围收缩）*
