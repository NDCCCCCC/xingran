---
gsd_state_version: "1.0"
milestone: v1.24
status: Awaiting next milestone
stopped_at: 全部 8 phase 执行完毕：76 双框 P0 + 77 视觉验证收口（17 张证据入库）+ 78 权限双端闭环 + 79 PG 实机验收（三层证据）+ 80 工位诊断展示 + 81 F5 修复（红绿验证）+ 82 lint 清零（0 problems）+ 83 devices 拆分（714→306 行）
last_updated: "2026-10-28T05:08:17.697Z"
last_activity: 2026-10-28
last_activity_desc: Milestone v1.24 completed and archived
state_head: 4477b2bc786739227f2c39de6b123601c6e1a6ba
progress:
  total_phases: 8
  completed_phases: 8
  total_plans: 0
  completed_plans: 0
  percent: 100
milestone_name: 视觉回归修复与遗留债务清偿 (Visual Regression Fix & Legacy Debt Clearance)
current_phase: null
current_phase_name: —
---

# Project State

**Project**: RuoYi-Next 运维管理系统
**Created**: 2026-04-16
**Status**: v1.24 milestone complete
**Last activity**: 2026-10-28
**Prior activity**: 2026-10-28 — v1.24 全量执行完成（Phase 77 视觉验证 + Phase 83 拆分收口 + VIS-03 死规则修复）
**Prior activity (2)**: 2026-10-28 — v1.24 执行期：78 权限双端闭环（后端严格中间件 + 前端三态门控）+ 79 PG 实机验收（三层证据）+ 82 lint 清零
**Prior activity (3)**: 2026-10-28 — v1.23 SHIPPED & ARCHIVED（Phases 69-75, 22/22 REQ satisfied）
**Prior activity (4)**: 2026-10-27 — v1.23 roadmap CREATED（Phases 69-75, 22/22 REQ mapped）

## Current Milestone: v1.24 视觉回归修复与遗留债务清偿

**Core value**: 修复用户主诉的「输入框双重外框」（309 个输入框全站受影响）并清偿三类遗留债务——#16 行操作权限越权（前后端双缺口）、3f130 工位物理链路诊断缺失、F5 刷新回退 dashboard；同时把 lint 存量清零（51 → 0）。

**事实基线**: `.planning/research/{STACK,FEATURES,ARCHITECTURE,PITFALLS}.md`（4 维并行研究）+ 4 份实证 note（`2026-10-28-pg-permission-baseline.md` / `2026-10-28-double-border-rootcause.md` / `2026-10-28-workstation-binding-facts.md` / `2026-10-29-v124-perm-fixture.md`）+ `v1.24-css-audit-register.md`（36 条 CSS 登记）+ `v1.24-visual-verification.md`（21 形态 × 明暗矩阵 + 17 张截图）+ `v1.24-phase78-spec.md`（Lead 审定的执行规格）

## Current Position

Phase: Milestone v1.24 complete
Plan: —
Status: Awaiting next milestone
Last activity: 2026-10-28 — Milestone v1.24 completed and archived

### v1.24 Phase 依赖速查

| Phase | 主题 | 硬依赖 | 需求 | 关键产出 |
|-------|------|--------|------|---------|
| 76 | 双框 P0 修复 + CSS 审计登记册 | — | VIS-01~04 | 复位块 + status 还原；双框 3→0；登记册 36 条 |
| 77 | 视觉回归验证收口 | 76 | VIS-05 | 21 形态 × 明暗矩阵；双框 = 0；17 张证据图入库 |
| 78 | #16 权限双端闭环 | — | PERM-01~04, 06 | 严格权限中间件 + 读/写兄弟分离 + 三态门控 |
| 79 | PG 实机验收 | 78 | PERM-05 | HTTP 8/8 403 + 对照实验；DB 层前后对照 |
| 80 | 3f130 工位物理诊断展示 | — | DIAG-01 | Table+Alert 分级渲染 + 静默失败补提示 |
| 81 | F5 刷新落 /dashboard 修复 | — | NAV-01 | catch-all 语义区分 + LAST_PATH 防自毁（红绿验证） |
| 82 | lint 存量清零 | — | HYG-01~03, 05 | eslint **0 problems**；豁免表 159→38 接入 lint:ci |
| 83 | devices 页拆分 | 78 | HYG-04 | 714 → 306 行（-57%）+ 行操作再拆至 192 行 |

**并行区间**：76/80/81/82 零依赖并行（Wave 1）→ 78/79（Wave 2）→ 83（Wave 3）→ 77 收口。

### v1.24 执行期关键决策（locked）

- **双框根因**：`index.css:871` 的 `.ant-input{border:...!important}` 命中 antd 6 affix-wrapper 的**内层** `input.ant-input`（antd 原生契约要求内层 `border:none`）→ 双框。`:not(组合子)` 在 CSS3 下整条规则静默丢弃 ⇒ 必须用 `.ant-input-affix-wrapper > input.ant-input`
- **复位块不写 box-shadow**：焦点环归 wrapper（antd 内层 `box-shadow:none!important` 特异性 0,4,1 自足）
- **#16 双病灶必须同时修**：组级 ANY-of（`router.go`）+ 模块级降级（`permission.go` Step 3）；只藏前端按钮 = 假修复
- **gin `Group()` 继承父组中间件链** ⇒ 写组必须是 `authorized` 的**兄弟组**（写成读组的子组则读组 ANY-of 仍放行写操作）
- **三态门控禁用 `permissions.length===0` 判未就绪**：空权限集与未加载完在数据上不可区分
- **hydrate 路径必须同步置 `permissionsLoaded`**：直接 `setState` 绕过 `setPermissions` ⇒ 硬刷新后所有操作按钮灰一阵
- **CSS status class 名**：antd 6 把 status 打在 wrapper **自身**（`ant-input-status-error`），**不存在** `ant-input-affix-wrapper-status-error`（该写法 = 永不命中的死规则，契约检查必须双向断言）
- **权限判定有 30s TTL 缓存**且 `InvalidateMiddlewarePermCache` 未接线 ⇒ 改权限后须等 ≤30s 生效

### v1.25 已立项候选（v1.24 结转）

- **候选①**：操作级 RBAC 全站治理（108 页面 + 340 路由；`roles/menus/depts/posts` 4 组 ANY-of 拆分 + `pkg/permission/config.go` 死表接线 + 动作级 F 型菜单播种）
- **候选②**：样式层收敛到 antd 6 variant（方案 C）
- **候选③**：权限模型三缺口（`data_scope` 未被列表消费 / `GetUserPermissions` 不断过滤 `menu.status` / Step 2 语义易写反）+ 生产库无迁移记录表 + GORM schema 漂移
- **候选④**：Phase 77 视觉新发现（暗色 `--input-bg` 第 3 处就地重定义 / 暗色 Select 白底两套边框体系 / `.dept-tree-search` 死规则 / antd 6 弃用警告 2 类 / InputNumber 错误态仍灰）
- **候选⑤**（待议）：`room-devices/index.tsx` 815 行（比 devices 更长，未扩面）

### Phase 依赖速查（v1.23）

| Phase | 主题 | 硬依赖 | 需求 |
|-------|------|--------|------|
| 69 | 后端 debug 重验 spike + 信息收集下发 | — | DBG-01 |
| 70 | TextArea 契约基建 (W0) | — | TEXT-01, TEXT-03, TEXT-04, HYG-05 |
| 71 | TextArea 存量迁移 + COMP-05 分域迁移 (W1-W6) | 70 | TEXT-02, MIG-01, MIG-02, MIG-04 |
| 72 | 后端 debug 修复 wave (#3→#2→#1-b) | 69 | DBG-02, DBG-03, DBG-04 |
| 73 | DB 实测日 + cron.go 双执行调查 | 72 + 69 | DBG-05, DBG-06, DBG-07 |
| 74 | COMP-05 收口 + lint 清偿第一波 + 流程积压归档 | 71 | MIG-03, HYG-01, HYG-02, HYG-03, HYG-04 |
| 75 | 浏览器实机 UAT + 收档 | 74 + 72/73 | UAT-01, UAT-02 |

**并行区间**：Phase 69/70/71（后端线 + 前端线无共享文件）∥ Phase 72/73；Phase 74/75 必须等前端线（71）完成。

## Accumulated Context

### v1.23 关键决策（init + roadmap 期, locked）

- `AppTextArea` **独立具名导出**，不挂 `AppInput.TextArea` 静态别名——成员表达式 selector 无法区分合法别名用法与非法裸用法，挂别名会永久放弃成员表达式拦截面、74 处收口假绿
- 排期口径 = **eslint selector 实跑命中数 + 豁免清单条目数**（权威），grep 数仅供粗估；每个 phase 的成功标准以「权威计数落盘/对账」表述，不硬编码某个 grep 数
- DBG-03 走 **Go 侧防线**（写入方枚举 + 显式 normalize + 回归测试），**不用 DB TRIGGER**（维持 v1.19 决策）
- 零新依赖：不引 jscodeshift / Playwright；UAT 用环境已有的 agent-browser skill
- 排序硬约束：迁移 → 收口 → lint → UAT（UAT 必须测迁移后的页面；lint 排在迁移后因迁移波次自身产生新 warn 峰值）
- 后端 debug 线与前端线**全程无共享文件、可并行**；#3 先于 #2（同文件 mac_collection_service.go）；cron.go 改动独立隔离（回归闸门 = `go test ./internal/scheduler/` 全套）

### v1.23 Phase 71 执行期决策（W1-W2 实证, locked for W3-W6）

- **P67 别名降险法在本项目不可用**（实测推翻了规划假设）：P65/P70 的 selector 是 **JSX 名匹配**（`JSXOpeningElement[name.name='Input']`），不是 import 路径匹配。`import { AppInput as Input }` 后 `<Input/>` 仍报 `no-restricted-syntax`，文件无法脱离豁免清单 → 无法达成「grep 命中 = 豁免剩余」对账。**W3-W6 必须用具名迁移**（JSX 名一并改 `<AppInput/>`/`<AppSelect/>`/`<AppTextArea/>`），diff 仍可极小（多数文件仅 import 行 + 标签名）
- **文件移出 P65 豁免块后，P68 hex 与 P65 内网 IP 两条契约会突然对该文件生效** → 迁移前必须先跑定向 eslint 预清存量裸 hex / 内网 IP 字面量，否则「收缩即超红线」（W2 已暴露 manager-rules 的 `#888` 2 处新 error）
- **Select 形态分类器（W1/W2 实测 56 处）**：①options 形态直接迁；②③children/fieldNames/labelInValue **必须改写为 `options={[...]}`**（否则 `optionFilterProp="label"` 指向不存在字段，过滤会「输什么都过滤光」）；④明确禁搜显式 `showSearch={false}`；⑤children+复合子节点保留写法并显式 `optionFilterProp="label"`；⑦multiple/tags 显式 `showSearch`
- **只读展示型 TextArea 显式 `allowClear={false}`**（如 logs 页 operParam/errorMessage）——默认清空按钮在 readOnly 受控区是纯 UI 倒退
- **UUID 炸弹真实分布**：W1/W2 范围唯一 UUID 下拉（dept 负责人）原本 allowClear=true 行为未变；**W3 operations 需重点排查「迁移后凭空获得清空能力」的 org_id/device_id 字段**（那才需 `allowClear={false}`）
- **P66 豁免清单（hook import）与 P65 是两条独立线**：W1/W2 迁完 P65 127→97，P66 保持 37 不动（迁移文件仍在用 usePagination/useTableManager）

- DB 实测日集中 #4/#5/#7（一次环境准备换三个清单）

### v1.22 关键决策（init + roadmap 期, locked）

- 抽契约不抽业务：资产选择器、部门树选择器等业务语义组件已够用，不抽
- `useDebouncedSearch` 并入 RemoteSelect，不单独抽
- 巨型文件拆分不单独立项，随 COMP-05 迁移顺带
- 默认值必须下沉封装层，禁止逐页补 allowClear（横跨 73 文件，不可维护）
- ConfirmDelete 是危险操作唯一入口；危险操作必须二次确认
- COMP-01/02（AppInput/AppSelect/ConfirmDelete）先于 COMP-03/04（PageTable/FormModal）落地，后者依赖前者
- COMP-05 迁移顺序：C1 试点 discoveries 最小页 → backups → devices/cache 巨型页；每页一个 PR，旧实现保留一个迭代
- FIX-02 Modal 脏检查在 Phase 64 存量修补、Phase 66 契约化到 FormModal（两段式）
- 本里程碑纯前端，不动后端接口；预期收益：消除约 7,000~8,000 行样板

### 继承自 v1.21 的环境约束

- `gsd_run` CLI 在本环境不可用（`$HOME/.claude/get-shit-done/bin/gsd-tools.cjs` 不接受 `query <verb>`）——STATE/ROADMAP/REQUIREMENTS 用手工编辑（Phase 57-62 先例）
- 提交落在 main 分支（`branching_strategy: none`）
- DB migration 编号已用到 218（Phase 59）；v1.23 **可能**新增 migration（#4 若确认 `sys_job` 双行根因 → 唯一索引；#3 走 Go 侧防线**不需要** DB TRIGGER migration），落地时从 219 起编

### Decisions

- [v1.21 遗留] 57-HUMAN-UAT.md 三项真环境验证（PG 合并路径 / kill -9 重启 / UAC 533）仍待人工执行；v1.18/1.19/1.20 的 site-visit UAT 债同样挂账

### Blockers

- 无

### Todos

- Phase 69 / 70 无前置依赖，可立即 `/gsd-plan-phase 69` 与 `/gsd-plan-phase 70` 并行起草
- Phase 71 等 Phase 70 计划落地；Phase 72 等 Phase 69 的 spike 产出；Phase 73 等 Phase 72 落地 + 用户 DB 实测回填
- Phase 74 等 Phase 71 迁移完成（收口硬依赖）；Phase 75 等 Phase 74 + 72/73
- vendor-react gzip 776 kB 基线沿用为 Phase 71/74 零回归守卫
- 权威计数基线（eslint 命中数 + 豁免清单条目数）由 Phase 70 落盘，Phase 71/74 以其为对账分母

### Roadmap Evolution

- Phases 76-83 created for v1.24「视觉回归修复与遗留债务清偿」: 76 双框 P0 修复 + CSS 审计登记册（VIS-01~04）→ 77 视觉回归验证收口（VIS-05）→ 78 #16 前后端双端闭环（PERM-01~04/06）→ 79 PG 实机验收（PERM-05）→ 80 3f130 空因诊断前端接入（DIAG-01）→ 81 F5 刷新落 /dashboard 修复（NAV-01）→ 82 lint 清零 + 豁免理由表修复（HYG-01/02/03/05）→ 83 devices 页拆分（HYG-04）。18/18 需求映射，无孤儿无重复；**全部 SHIPPED 2026-10-28**（tag `v1.24`）。 (CRITICAL)
- Phases 69-75 created for v1.23「前端契约全量收尾与遗留清偿」: 69 后端 debug 重验 spike（DBG-01）→ 70 TextArea 契约基建 W0（TEXT-01/03/04 + HYG-05）→ 71 TextArea 存量 + COMP-05 分域迁移 W1-W6（TEXT-02 + MIG-01/02/04）→ 72 后端 debug 修复 wave（DBG-02/03/04，#3 先于 #2）→ 73 DB 实测日 + cron.go 双执行调查（DBG-05/06/07）→ 74 COMP-05 收口 + lint 清偿 + quick 目录归档（MIG-03 + HYG-01~04）→ 75 浏览器实机 UAT + 收档（UAT-01/02）。22/22 需求映射，无孤儿无重复。 (CRITICAL)
- Phase 68.1 inserted after Phase 68: v1.22 收尾清偿 phase：危险操作收口 + 契约 minor 补课 + 豁免收缩 + 存量迁移收尾（DEBT-04） (URGENT)

## Project Reference

See: `.planning/PROJECT.md`（updated 2026-10-28 after v1.24）

**Core value:** 端到端的运维可观测与可审计——每个写操作都留下可追溯记录（谁、何时、改了什么、从哪来、改前改后），敏感字段自动脱敏；配合 Excel 批量管理与引用自动解析（部门名 → UUID），运维人员对整个 IT 资产域拥有唯一可信的系统记录。
**Current focus:** v1.24 SHIPPED & ARCHIVED（tag `v1.24`）——**等待下个里程碑立项**（v1.25 候选 5 笔已完整登记，用户裁定暂不立项；`.planning/REQUIREMENTS.md` 已按流程删除，由 `/gsd-new-milestone` 重建）

## Session

**Last session:** 2026-10-28T13:10:00+08:00
**Stopped at:** v1.24 milestone completed and archived（tag `v1.24`）
**Resume file:** 无（里程碑已收档；下次从 `/gsd-new-milestone` 起）

## Performance Metrics

| Plan | Duration | Tasks | Files |
|------|----------|-------|-------|
| v1.24（8 phase，subagent 直接执行，按刀原子提交） | ~3.5 h（09:19→12:59，单日） | 8 phase × 9 项闸门 | 104（+5,419 −2,232，46 commits） |
| v1.23（7 phase） | 单日 | 121 文件迁移 | — |

## Operator Next Steps

- **v1.24 已 SHIPPED & ARCHIVED**（tag `v1.24`，归档至 `.planning/milestones/v1.24-{ROADMAP,REQUIREMENTS}.md`）
- 下个里程碑用 `/gsd-new-milestone` 起（会重建 `.planning/REQUIREMENTS.md`）
- **待用户决定是否 push tag**（remote `origin` = gitcode.com/ninedrunk/go_react，upstream `origin/main`）
- v1.25 的 5 笔候选债已完整登记（`REQUIREMENTS.md` 归档版 + `MILESTONES.md` v1.24 条目 Known Gaps），随时可启
