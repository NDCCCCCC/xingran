# Project Retrospective

*A living document updated after each milestone. Lessons feed forward into future planning.*

## Milestone: v1.52 — 运维域数据权限统一收口（终态）

**Shipped:** 2026-11-07 | **Archived:** 2026-11-08
**Phases:** 4 (130-133) | **Requirements:** 7/7 (DS-01~07) | **Commits:** 24 | **PR:** #75 merged `b56f1286`

### What Was Built

- `pkg/datascope` ScopeProvider 单一真实源（可见部门集五档 / 0-unset→All 特判 / 异常枚举 fail-closed / 多角色 MAX=最严档），ops 全域 26 处读写组挂载 `DataScopePermission`。
- 四域（workstation/floor/building/infopoint）读端点 20 个 + Excel 导出两条独立路径统一「有效机构 ∈ 可见集」；缓存 key 增 scope 维度并交付 `131-CACHE-INVENTORY.md`。
- 四域写端点越界显式 403（`NewWithHTTPStatus` 绕开 `CodePermissionDenied` 默认 401 坑）、批量按可见集裁剪、backfill 三层注入、Excel 导入逐行校验、前端「本次范围」四类文案。
- asset/serverRoom/dedicatedLine/roomDevice 收口（115 个语义断言）+ vdi 调用方迁移 + 删除 `pkg/middleware` 三份死实现与死测试 + DeptSidebar 联动 + 5 项显式豁免面登记。

### What Worked

- **三项开工输入裁定前置到 notes/**（Self 档 ops 语义 / Dept-Self 批量与导入豁免 / DS-05 三项归属链）在对应 phase 开工前落盘，避免执行期承载裁定动作——这个模式显著减少了返工。
- **独立只读代码审查**（agent-8）：3 P0 + 11 P1 在合并前拦截，尤其是「迁移 235 被误嵌在 234 失败分支内 → 正常启动永不 seed」这类静态测试照不到的缺陷。
- **全量回归闸门**作为单一收口判据（go build / go test / vitest 4128 / lint:ci 0 / diff coverage 83.37% / 生产构建），不采信单个 agent 报告。

### What Was Inefficient

- PR #75 推送后连遇 4 类 CI 红灯（prettier 4 文件 / antd 6 弃用 prop 3 项 / coverage-diff 76.63%<80% / DeptTree 类型 + staticcheck SA1012），其中 coverage 缺口需临时补 644 行厚测试。**教训**：本地应先跑 `npm run format:check` 与 `check-coverage.sh`，不要等 CI。
- `130-device-evidence/`（v1.51 遗留归档）与 `130-scope-provider/`（v1.52 新 phase）共用 "130-" 前缀，GSD CLI 归档时给出 duplicate phase key 警告（#3355），靠 `STATE.md:67-69` 的消歧说明才没误判。**教训**：历史归档目录改名比写消歧注释更彻底，但会破坏可追溯性——本项目选择保留 + 显式注记。

### Patterns Established

- 数据权限统一收口的「有效机构」表达式走 `effectiveOrgExpr` 单一真实源，禁止内联复制（P2-6 教训）。
- 缓存 key 必含数据范围维度段；`floor:tree` 这类全局固定 key 因按可见集过滤必须改结构或停用（PMODEL-01 跨范围泄漏教训）。
- PG contract 新增测试统一门控变量名 `TEST_PG_DSN`，CI 中 skip 必须可观测。
- 越界语义：单条写显式 403、批量按可见集裁剪（不整批 403）、聚合计数侧信道作为**已裁定接受的已知残余**登记而非假装已解决。

### Key Lessons

1. **accept + 登记 > 假装修好**：P1-5 聚合计数侧信道、5 项显式豁免面均明确写「已知残余风险、接受；未来应加告警通道而非改回整批 403」，不以「间接可见所以安全」结案。
2. **CLI 机械重写必须人工复核**：`/gsd-complete-milestone` 会把 `130-device-evidence` 算作第 6 个 phase、把 `stopped_at` 含反引号的值截断、把 `progress.total_phases` 改成 5、把 MILESTONES 顶部权威声明挤到条目之后。归档后必须逐字段比对备份再修正。
3. **归档前先跑 `audit-open` 并 acknowledge**：3 项 `uat_gaps`（0 pending scenarios）不阻塞收档，但必须按 [A] 分支登记 deferred 并在 STATE.md / MILESTONES.md 留痕，否则下次扫描会重复报。
4. **归档会产生路径死链**：当时的临时清单、代码注释、`MILESTONE-v1.52-SUMMARY.md` 里大量 `phases/...` / `quick/...` 引用，靠一张 `ARCHIVE-INDEX.md` 路径映射换算而不是逐个改历史正文，可保住可追溯性。

### Cost Observations

- 模型与会话：多 subagent 并行（审查 / 执行 / 验证），具体配比为环境动态调度，未单独记账。
- Notable：本里程碑交付后，一次「检查是否存在已记录的缺陷项/待办」的只读审计发现 **37 个文件的文档债**（已落地却仍标待办、状态文件互相矛盾），随后用 2 路甄别 + 4 路并行写入收敛为 1 份临时开放项清单（21 条真实开放项）。**项目规模到 v1.52 时，「记录腐烂」本身已成为比代码缺陷更大的维护成本**——建议每个里程碑收档固定做一次文档债扫描。

---

## Milestone: v1.25 — 全站权限治理与样式收敛 (Site-wide Permission Governance & antd 6 Style Convergence)

**Shipped:** 2026-10-30
**Phases:** 10 (84-93) | **Requirements:** 22/22 Satisfied | **Timeline:** 2.5 天 (2026-10-28 13:11 → 2026-10-30 16:17)
**Git:** 99 commits / 399 files / +56,653 −9,836

### What Was Built

- **线 A·操作级 RBAC 全站治理**：
  - 133 处写路由 ANY-of 拆分为逐动作单一权限点（`RequireStrictPermissions`）；
  - 全站 76 个业务页面操作入口按动作级权限码三态门控全覆盖（恒等式 25 已门控 + 51 豁免 = 76 锁死）；
  - 72 个 F 型按钮权限码全量入库播种，映射死表真实接线，未登记写路由归零；
  - 权限模型三缺口修复（`data_scope` 真实消费、`menu.status` 过滤、子菜单继承语义命名固化）。
- **线 B·样式收敛与巨型单体拆分**：
  - 453 条暗色 mega-cluster 手写覆盖整块交还 antd token 与 `darkAlgorithm`；
  - 量化终态达标锁死：全仓 `!important` 1435 → **452**（≤455 预算，净降 983 处，-68.5%），Hex 311 → **26**（≤60 预算），`index.css` 行数 5632 → **3194**（≤3200 预算）；
  - `room-devices/index.tsx` 896 行大单体解耦为 342 行（≤350 行，子文件均 ≤250 行，零行为变化）；
  - C1~C16c 样式静态契约全部 PASS（双框元素数 0，单圈焦点环，8px 统一圆角）；
  - 14 个核心业务页面视觉巡检零异常断层。

### What Worked

- **双线分治，线内严格串行**：线 A（Go 后端权限）与线 B（前端 CSS 样式）在阶段前期并行推进；样式线内部（88 基础断言 → 89 变量重构 → 90 Input 域 → 91 Select/Table 域 → 92 暗色 mega-cluster → 93 拆分与终验）严格串行，三处不可逆顺序约束写死，极大避免了跨阶段样式冲突与假绿。
- **AST 精确扫描与自动化批处理**：自制 `scan-dark-rules.cjs` 对 570 行暗色规则做 AST 语法级边界识别，区分 453 条可删除覆盖与 46 条合法例外，实现了整块安全清零，无一误伤活规则。
- **量化预算硬锁（Ratchet Guard）**：把 `IMPORTANT_BUDGET`、`C13_M7_BUDGET`、`C13_M9_BUDGET` 写入 CI 契约检查脚本，每攻克一个波次立即下调并锁死预算，形成「只降不升」的机械防线。
- **矩阵采样机器断言先行**：搭建 computed-style 21 形态断言工具链，机器对比毫秒级反馈，确保组件视觉基线零破坏。

### What Was Inefficient

- **大单体暗色清除后的空章节残留**：在 Phase 92 清理完 453 条规则后，遗留了 30 余个空章节注释，导致 M9（代码行数）一度逼近 3440 行，不得不追加 Gap 计划做二次空白与死注释收敛。
- **次级 Hex 的隐蔽扩散**：历史 CSS 中大量存在阴影与边框使用的裸 Hex，在样式收敛晚期才触发 per-file 与全量阈值预警，后续应在前期变量层重构时尽早统一转为标准 `rgb(...)`。

### Key Lessons

1. **样式的根基在 Token，不在选择器覆盖**：手写 `!important` 选择器覆盖只能解决一时，但会破坏 antd 内部算法，最终演变为 1400 多处的债务大山；正解是按 Antd 规范分层配置 Component Token 和 Map Token。
2. **重构大文件必须先锁定外部注入契约**：在拆分 `room-devices` 时，先精确定位 `reloadRef` 注入链路和 `columns` 的缓存依赖，保证重构过程中外部组件调用协议零变动，确保 100% 行为保真。
3. **机器断言比肉眼巡检更可靠**：6px 到 8px 的圆角差异肉眼极难辨识，依托 computed-style 物理单值断言才能铸造不可逾越的质量底座。

---

## Milestone: v1.24 — 视觉回归修复与遗留债务清偿 (Visual Regression Fix & Legacy Debt Clearance)

**Shipped:** 2026-10-28
**Phases:** 8 (76-83) | **Requirements:** 18/18 Satisfied | **Timeline:** 单日完成（09:19 需求定稿 → 12:59 收口，约 3.5 小时）
**Git:** 46 commits / 104 files / +5,419 −2,232

### What Was Built

- **双框根治** — 复位块 `.ant-input-affix-wrapper > input.ant-input`（6 项、**刻意不写 box-shadow**，焦点环归 wrapper）；**双框元素数 3 → 0**（harness 42 行 + 15 真实页 + 登录页，明暗双测）；错误态恢复品牌红
- **CSS 一致性审计登记册** — 36 条（P0 已修 8 / P1 待修 7 / P2 登记 19 / 保留 1 / 样板 1）；契约检查脚本 ALL PASS
- **#16 行操作权限前后端双端闭环** — `RequireStrictPermissions`（只做 Step1 精确匹配）+ `/system/users` 读组/写组**兄弟分离** + `useActionPermission` 三态门控 + 权限码单一真相源；三层证据（DB 前后对照 / HTTP 8/8 403 含决定性对照实验 / 组件测试 7/7）
- **F5 刷新回退修复** — catch-all 语义区分 + LAST_PATH 防自毁反写，**红绿验证通过**
- **3f130 空因诊断展示** — Table+Alert 分级渲染（`no_user_bound` → info 常态 / 其余 → warning）+ 静默失败补提示
- **lint 51 → 0 problems** + devices 页 714 → 306 行（-57%）

### What Worked

- **波次化并行** — W1（76/80/81/82 零依赖）→ W2（78/79 需 PG）→ W3（83 依赖 78 的 gate 模式）；8 phase 在 3.5 小时内跑完，文件域零重叠让并行成为常态而非例外
- **Lead 先定规格再派发** — Phase 78 的执行规格（含完整代码草案 + 8 项验收 + 边界清单）先于派发写死，避免执行 agent 自由发挥导致口径漂移
- **实机验证反查实现** — Phase 77 用真实浏览器把 Phase 76 自己没发现的 P0 死规则查了出来（见 Key Lessons #1）
- **决定性对照实验** — #16 验收不满足于「403」，追加「临时授予 xtgl → 200 且真删除」的反向实验，排除「403 只是接口坏了」的解释
- **收口前 Lead 独立复核** — 不只采信 agent 报告：全量 vitest 独立复跑（确认 1 例超时是环境 flaky 非回归）+ 闸门六道全绿

### What Was Inefficient

- **4 个 subagent 崩溃** — 82/80/81/76 相继崩溃；80 与 81 由主会话亲自接管（81 的遗留改动需逐行审阅后补完测试）
- **凭据传递绕了三轮弯路** — 环境变量继承失败 → 仓库外 TEMP 文件 → 直接贴明文；正确做法应是首次即贴明文（agent 无法被 steer，补发无门）
- **两个 agent 重复做同一份造数** — 幂等（`WHERE NOT EXISTS`）兜住了重复行，但浪费了一个 agent 的运行时间
- **我在自己规格里写错了一个真陷阱** — gin `Group()` 继承父组中间件链 → 写组若挂在读组之下，病灶① 不会被修掉；由实施方识破并修正（见 Key Lessons #4）

### Patterns Established

- **实现 → 实机验证的反查机制** — 静态断言与 computed-style 单点断言都覆盖不到「class 名在当前库版本不存在」这类错误；真实浏览器 + 真实页面是不可替代的验证层
- **契约断言必须双向** — 正向「规则存在」不够，还要反向「不得出现已知死形态」；只看正向会得到「检查通过而功能从未生效」
- **证据图入库** — `.gitignore` 的全局 `*.png` 会让验收凭证只存本地工作区，必须为证据目录加精确例外
- **前后对照表** — 修复类工作必须留「修复前 / 修复后」两套实测数据（DB 层 + HTTP 层），否则修复后无法证明缺陷曾经存在

### Key Lessons

1. **契约检查校验「规则存在」≠ 校验「规则能命中」** — Phase 76 的 VIS-03 写成 `.ant-input-affix-wrapper.ant-input-affix-wrapper-status-error`，而 antd 6.6.4 把 status class 打在 wrapper **自身**（`ant-input-status-error`）→ 该 class 根本不存在 → 永不命中；叠加 `allowClear` 默认开 → **站内绝大多数输入框错误态一直不变红**。而 Phase 76 的 C5 检查**校验的正是这条死规则** → 检查通过、功能从未生效。**教训：契约检查必须双向断言，且断言目标要锚定「当前依赖版本里真实存在的选择器」，不能凭印象写。**
2. **红绿验证是识别「假绿」的唯一手段** — 我第一版 NAV-01 回归测试在**回退实现**下仍然通过（fixture 用 `name` 而非 `menuName` → 菜单被 `validateMenu` 静默跳过；且未预置菜单缓存时组件在 `:246` 提前 return，根本走不到 catch-all）。**教训：写完回归测试必须先回退实现验证它会红，否则测的是空气。**
3. **代码结构体 ≠ 库表结构** — 从 `models/role.go` 的 `RoleMenu{RoleID, MenuID}` 推断「表无 id 列」并据此判另一 agent 的 INSERT 会失败，实测生产库**实有** `id`/`created_at` 列（模型未映射 ≠ 表无列）。**教训：跨模型/迁移场景不得凭结构体推断库表结构；且 PG 能跑 ≠ sqlite 新库能跑。**
4. **Lead 的规格也会写错** — Phase 78 规格里的 `usersWrite := users.Group("")` 有安全 bug（gin 继承父组中间件链 → 读组 ANY-of 仍放行写操作）。**教训：规格定稿不等于免于审查；执行方有义务在实现时指出规格缺陷（本次实施方正确地做了）。**
5. **真实浏览器验证能查出静态断言查不出的东西** — VIS-03 死规则、`.dept-tree-search` 补丁从未命中、暗色 token 就地重定义，三项都只有真浏览器 + computed-style 才暴露。**教训：CSS/视觉类修复的验收层不能只有静态检查。**
6. **PowerShell 的 GBK 显示会制造「文件损坏」假象** — `Get-Content` 把 UTF-8 中文显示成乱码，多次让我误判文件被写坏（实测均正常）。**教训：判定文件编码必须用 node 读 UTF-8，禁用 PowerShell 的 Get-Content 读中文文件。**
7. **subagent 无法被 steer** — `send_message` 对已启动的 subagent 报 not found（它们不在 teammate 树内）。**教训：subagent 的任务书必须一次给全（凭据、路径、方法、边界），不能指望中途补充。**
8. **生产环境知识必须入库** — `SKIP_AUTOMIGRATE` 在 release 模式会 fatal（须同时降 `SERVER_MODE=test`）、JWT 密钥环境变量真名是 `JWT_SM2_*` 而非 config 注释里的 `RUOYI_JWT_SM2_*`、`10.62.10.34:9000` 是 MinIO 不是后端。**教训：这类「试错换来的」环境事实必须落 MEMORY，否则每个新会话都要重踩。**

### Cost Observations

- 8 phases / 46 commits / 104 files / 约 3.5 小时（单日）
- 4 个 subagent（其中 4 个崩溃需重派）+ 主会话亲自接管 Phase 80 与 Phase 81
- Notable: 崩溃率偏高（约 4/9 次派发崩溃）——**单会话内串行完成两个 phase（80/81）反而比反复重派更快**，且接管遗留改动时发现了 agent 遗漏（如 hydrate 未置 `permissionsLoaded`、NAV-01 缺测试）

---

## Milestone: v1.23 — 前端契约全量收尾与遗留清偿 (Frontend Contract Full Rollout & Legacy Clearance)

**Shipped:** 2026-10-28
**Phases:** 7 (69-75) | **Requirements:** 22/22 Satisfied | **Timeline:** 单日完成（2026-10-27 规划+研究 → 10-28 收档），多 subagent 并行

### What Was Built

- **TextArea 一键清空契约（用户主诉闭环）** — AppTextArea 独立组件 + SearchForm textarea 字段 + eslint 成员表达式/解构双 selector（探针验证非假绿）+ 存量全量接入 + 实机 UAT 抽查通过
- **COMP-05 全量收尾** — W1-W6 六波 121 文件具名迁移，P65 豁免 127→0，无豁免口径裸组件命中全归零；Select 形态七类分类器
- **7 项 debug 全终结** — #5 MAC 查询双侧归一化双分支 SQL / #7 空因可诊断端点（纯增量）/ #2 Cisco 采集 / #3 Go 写防线；#1/#4 实证不复现（非 systemd 托管 / 双实例部署）；#6 关单
- **lint 清偿 + 门禁** — errors 81→17、warnings 1366→34、any 958→0、unused 364→0、console 5→0；--max-warnings 34 + assert-lint-budget.mjs 断言
- **流程清偿** — quick 目录 94 个归档、68.1 §8 实机回写关债、FAIL-1 分页恢复 bug 修复实机复验

### What Worked

- **4 维并行研究前置** — 规划前拿到 selector 盲区、豁免锚点脆弱性、acknowledge 不等于已定位等关键情报，直接改变 phase 排序（spike 先行）
- **subagent 深度并行** — 前端线与后端线无共享文件全程并行（70/71 与 72/73），W3 并行 W4、W5 并行 W6 按域切分，零冲突
- **每波真实闸门** — type-check/vitest/eslint 预算/build 四件套每 commit 前跑，82 到 81 到 17 errors 收敛全程零回归
- **用户三次 DB 实测一举清零调查** — spike 阶段下发 SQL/命令清单，实测数据直接钉死 #5/#7 根因，Phase 73 变纯执行
- **探针验证文化** — selector 生效性用临时探针实测而非推理；红绿验证证明测试有效

### What Was Inefficient

- **pwsh 编码事故连环（最痛）** — Get-Content/Set-Content 在 GBK 控制台把中文规划文件读坏，且字符串替换因乱码静默失败，收档时才发现需从 git 历史重建（REQUIREMENTS 全量重建 + 2 份 debug 文档恢复）
- **规划假设被实测推翻** — P67 别名降险法在 JSX 名匹配 selector 下无效，W1 探针才发现；规划期可用 5 分钟探针避免
- **subagent 中途崩溃 3 次** — W3/W5/W6 崩溃但遗留合格半成品，主会话接管补验或重派收尾；要有「验收遗留、接管或重派」固定动作
- **grep 检测面漏检换行形态** — Select 标签行尾换行逃过多波对账，「残留脱节文件」实为漏检；W5/W6 宽松模式补齐
- **UAT jsdom 盲区再现** — 分页恢复 jsdom 通过实机失败（mount 时序覆写），实机 UAT 必须保留为独立 phase

### Patterns Established

- **契约组件收口三件套** — 组件（默认值下沉）+ selector（eslint error 拦截）+ 豁免清单（gen 脚本管理、收口归零、转校验模式）
- **迁移波次闸门四件套** — type-check / 定向 vitest / eslint 预算断言 / build，每 commit 前必跑
- **Select 形态七类分类器** — options 直接迁 / children 必改 options / 禁搜显式 false / 复合子节点保留加显式 label / multiple 显式 showSearch
- **debug 文档生命周期** — investigating 到 root_cause_identified 到 resolved；acknowledge 结转必须先重验 spike
- **数量口径纪律** — eslint 实跑命中加豁免清单为权威；每波「域命中数 = 清单剩余数」对账

### Key Lessons

1. **acknowledge 不等于已定位**：结转 debug 项 7 项里 4 项调查未完成，直接排修复 wave 会修不存在的根因——重验 spike 必须前置
2. **用户实测是最强杠杆**：三次 DB 实测把整个「DB 实测日」调查 phase 清零
3. **契约默认值下沉必须配 selector 拦截**：否则存量永远补不齐、新增继续漏（74 处 TextArea 前车之鉴）
4. **中文规划文件读写工具纪律**：PowerShell 管道在 GBK 控制台静默毁文件且替换静默失败——一律用 node/write 工具

## Milestone: v1.19 — 网络设备写命令 (Network Device Port Write Operations)

**Shipped:** 2026-07-08 (Phase 55 close)
**Phases:** 6 (Phases 50-55; 50-54 = core MVP, 55 = tech-debt cleanup) | **Plans:** 9
**Sessions:** yolo mode sequential (5 build phases ~6 hours wall-clock + 1 cleanup ~20 min)

### What Was Built

- **`internal/services/portcollection/vendor_port_template.go`** — PortAction type + 5 const + 5 sentinel errors + 15 (vendor, action) command templates + 4 renderers (Phase 50 W1)
- **`internal/services/portwrite/`** 5-file package — parse_error / pre_state_check / port_write_service / batch_orchestrator + 28 mock tests (Phase 51 W2)
- **6 kebab POST 端点** `/network/ports/write/{shutdown,undo-shutdown,description,dot1x-enable,dot1x-disable,batch}` + `NetworkPortWrite` permission + `sys_port_write_audit` Path A (Phase 52 W3)
- **`GrantNewMenuToRolesHavingParent` helper** — INSERT-SELECT-FROM-ON-CONFLICT-DO-NOTHING 精准授权 (antd 父子联动陷阱根除)
- **`PortWriteModal` + `BulkWriteDrawer` + 端口页改造** — types/network.ts 4 类型 + networkApi.ts 6 wrappers + port-write/constants.ts 5 const + select→executing→result 状态机 (D-05 indeterminate Spin 不伪造 X/Y) (Phase 53 W4)
- **scrapligo FileTransport e2e** — 10 TestE2E_* 闭环 Phase 51 mockDeviceExecutor 不调 fn 漏洞 + CHANGELOG/README/API/加密/MILESTONES 文档同步 + 54-HUMAN-UAT.md 7 项 deferred (Phase 54 W5)
- **Tech-debt cleanup** — WR-02 reason validator 3-param signature + IN-01 instanceof Error + IN-02 eslint-disable 修正 + HealthCard.test.tsx regex match + CR-02 后端 batch_orchestrator.go fallback port 归属跨层防御 (Phase 55)

### What Worked

- **5-phase build order (W1→W2→W3→W4→W5) 严格执行** — Zero-dep 模板契约先行 → 服务签名带 mock → HTTP/audit/permission/migration 并发 → 前端批量 → e2e+docs。W1 table-driven tests 提前捕获厂商语法漂移；W2 mock cheapest；W3 API contract 稳定后前端才能建 Modal/Drawer；W5 mock SSH e2e 在 W1-W4 全 ship 后才有意义。每 wave 可独立 commit，无 cross-cutting regression
- **Path C audit↔operlog 关联不动 operlog 包** — handler 先 INSERT audit 拿 audit_id → `operlog.Record(..., WithOperParam(jsonWithAuditIDs))`；`audit.oper_log_id` 列保持 NULL。零侵入 operlog 包意味着 Phase 34 regression_test.go (25 OperType + 11 敏感关键词 + Record 5 参签名) 锁保持绿色，避免循环依赖
- **menu_grant_helpers.go 精准授权 根治 antd 父子联动陷阱** — INSERT INTO sys_role_menu SELECT rm.role_id, newMenuID FROM sys_role_menu rm JOIN sys_menu m ON rm.menu_id=m.id WHERE m.menu_name=parentMenuName ON CONFLICT DO NOTHING。比对所有子菜单自动带动父只动真正父已授权的角色
- **execSinglePort DRY helper** — 5 单端口 handler 公共流程合并，operlog.Record 物理调用点 6→2，semantically equivalent + 减少 ~200 行重复模板。Plan <verify> count 由 ≥6 调整为 ≥2 (Rule 3 — 不破坏语义的代码结构 cleanup)
- **Milestone audit 早跑** — `v1.19-MILESTONE-AUDIT.md` 在 Phase 55 完成后即跑，3-source 交叉验证 (REQUIREMENTS.md + Phase VERIFICATION.md + SUMMARY.md frontmatter requirements-completed) 全过；6/6 phases, 20/20 integration wires, 7/7 E2E flows PASSED
- **Pre-existing test failure baseline** — `tests/integration/login_encryption_test.go`、`pkg/errors/errors_test.go`、`internal/services/operations/...` 均先于 v1.19 6 周~5 个月，git diff empty，避免被误判为回归。沿用 v1.16 Phase 40/41 + v1.18 Phase 48 的兜底动作
- **Phase 51 mock 漏洞被 Phase 54 FileTransport 闭环** — `df04c6dd` 的 mockDeviceExecutor 简化 (不调 fn) + executeWrite nil-guard 让 mock-based test 干净通过；但 service.executeWrite → wrapper.SendConfigs → lastResp → parseConfigError 链路从未在集成层执行。Phase 54 用 scrapligo `transport.NewFileTransport()` + 6 fixture 真实闭环。这是 wall-clock 上的一个 lesson：mock 简化便宜，但 service code path 闭环必须 integration 层验证

### What Was Inefficient

- **Plan verification gate 与 doc 注释字面字符串冲突 (Phase 52)** — Plan `<verify>` `grep -q 'IsFrame\|IsCache'` 不带 `grep -v '//'`，但 plan `<read_first>` 鼓励 doc 注释解释"为什么不能加 IsFrame/IsCache" — 两件事直接冲突。doc 注释含 `IsFrame` 字面字符串触发 grep 误报。两层修复：① 测试改用 `go/parser + ast.Inspect` 跳过 `*ast.Comment` 节点；② doc 注释改用通用表述（"Java sys_menu 风格的 frame/cache 字段"）
- **`response.Error(c, int, msg)` 项目惯例被测试断言误用** — `int` 是 business code，HTTPStatus 固定 400（pkg/response/response.go:160-162）。测试断言走 `body.code == 404` 而非 `w.Code == http.StatusNotFound`。Rule 3 纠正
- **eslint-disable directive 初版位置错位 (Phase 55 IN-02)** — Plan 写 directive 紧贴 `useEffect(() => {` 开头，但 `react-hooks/exhaustive-deps` 规则实际在 `}, []);` 行触发。c17da02e 初版错位，adce5799 修正。Lesson: ESLint 规则触发行 ≠ 函数声明行。Codebase 内有 4 个 reference (App.tsx, MatchTestPanel, templates, exceptions) 走 directive-before-`}, []);` 模式；新人需要 awareness
- **PortResult.PortName optional field in PLAN.md 实际不存在 (Phase 55-02)** — 计划写 `PortName: actualPort.InterfaceName`(可选便于审计追踪)，但实际 `PortResult` struct (`port_write_service.go:34-42`) 无 `PortName` 字段。D-04 Scope Constrainment 严格锁定只改 batch_orchestrator.go 一处，移除可选 PortName 字段填入
- **d.Close() hangs on FileTransport (Phase 54)** — `Driver.Close()` 跑 `network-on-close` (acquire-priv + channel.write 'quit' + channel.return) 需要更多 bytes，但 FileTransport 的 `select{}` 在空 content 上 deadlock。E2e harness 故意 skip `defer d.Close()`; Go GC reap on test exit
- **2-cmd fixture needs double prompt lines (Phase 54)** — `SendConfigs` loops `SendConfig(cfg)` per cmd；each `SendConfig` calls `SendConfigs([cfg])` which calls `AcquirePriv("configuration")` via `GetPrompt`。第一次 cmd consumes bytes up to prompt，第二次 cmd 的 `GetPrompt` 需要 find current prompt still queued。`huawei_description_success.fixture` 使用 double prompt lines 让 second cmd's `AcquirePriv` 始终 find a current prompt
- **device_rejected fixture text matters (Phase 54)** — `% Error: Unrecognized command` 触发 scrapligo `huawei_vrp.yaml` `failed-when-contains` → `resp.Failed=true` → `parseConfigError` step 2 → `WriteErrorTransport` (错路径)。改用 `% Error: too many parameters` 触发 `parseConfigError.rejectionMarkers[0]='% Error:'` priority (over `failed-when-contains`) → `WriteErrorDeviceRejected` (correct SSH-02 path)

### Patterns Established

- **Detached context first line of batch entry** — `context.WithTimeout(context.Background(), 30*time.Minute)` 规避 `Core.Close()` 30s 截止陷阱。任何 batch 操作入口第一行 (Pitfall #5)
- **Internal mockable interface fields** — service impl fields 是 interfaces (`portWriteExecutor` / `portWriteCollectionSvc`) 而非 concrete pointers；factory 仍接受 concrete pointer (`*device.DeviceExecutor`) — testify/mock injection without router code changes
- **Rejection markers scan BEFORE transport markers in parseConfigError** — device rejection semantics priority over coincidental substring matches in error text
- **execSinglePort DRY helper** — 5 单端口 handler 通过 serviceCall 闭包传入差异，公共流程 (operlog.Record + response.Success) 提取 single call site
- **Path C audit↔operlog 关联** — handler 先 INSERT audit → operlog.Record(WithOperParam(audit_ids)); audit.oper_log_id 列保持 NULL；不动 operlog 包接口 (无 WithOperID / 无 WithJsonResult)
- **menu_grant_helpers.go::GrantNewMenuToRolesHavingParent** — INSERT-SELECT-FROM-ON-CONFLICT-DO-NOTHING 精准授权 (antd 父子联动陷阱根除)
- **eslint-disable placement** — directive 紧贴 `}, []);` 而非 `useEffect(` 开头 (codebase precedent: App.tsx, MatchTestPanel.tsx, templates, exceptions)
- **scrapligo FileTransport for e2e** — 真实 `SendConfigs` + `parseConfigError` 链路集成层闭环；mock 简化便宜但 service code path 必须 integration 层验证
- **Site-visit UAT 在 plan 阶段显式声明 deferred** — `.planning/phases/<phase>/<phase>-HUMAN-UAT.md` 创建在 W5，含 pending items，owner = 现场运维同事，`verifier_status: human_needed`
- **Document comment 字面字符串与 plan <verify> grep 冲突** — 用 `go/parser + ast.Inspect` 跳 `*ast.Comment` 节点只扫 `*ast.Ident + *ast.BasicLit`

### Key Lessons

1. **Phase 54 e2e 闭环 Phase 51 mock 漏洞 = mock 简化便宜但 service code path 必须 integration 层验证** — `mockDeviceExecutor.ExecuteCustom` 简化不调 fn 让 mock test 干净通过；但 service.executeWrite → wrapper.SendConfigs → lastResp → parseConfigError 链路从未在集成层执行。这个 Phase 51 done 的 plan verification 漏掉了真正的 transport layer 测试。Phase 54 必须 exist 作为 "后期补交" 来闭环 — Future milestones 在 mock-based service test 阶段 plan 中需明确 "Phase X+1 = integration e2e"
2. **Path C 关联的 operator_log 列保持 NULL 是 invariant** — audit.oper_log_id 列必须 schema definition 时 `gorm:"->;-:migration"`，否则 GORM AutoMigrate 推 `audit.oper_log_id = ?` → DB column 不存在 SQLSTATE 42703
3. **Vendor 模板硬编码 + table-driven tests 是 Phase 1 优先级** — 任何 multi-vendor SSH write op 在 Phase 1 必须有 vendor × action 派发表 + per-vendor unit tests，无 then 才进 service 集成层
4. **operlog regression_test.go 锁不可破** — OperType 25 常量 + 11 敏感关键词 + Record 5 参签名 是 Phase 34 invariant；任何新 endpoint 必须用现有 OperType 值（共 25 个，不允许 OperTypeOther 兜底）
5. **Pre-existing test failure baseline 在 milestone close 前必查** — 任何新 milestone 实施前后在 baseline commit 跑同一组测试 (v1.16 Phase 40/41 + v1.18 Phase 48 共验证)。v1.19 验证了 3 处 pre-existing 失败 (login_encryption / pkg/errors / operations)
6. **Permission 命名空间 split 必须 migration 时同时 seed 菜单 + grant helper** — 新权限 `network:port:write` 与原 `network:port:query` 拆分时，菜单 seed 必须配合 GrantNewMenuToRolesHavingParent 精准授权父已关联角色，避免 antd 父子联动陷阱
7. **scrapligo FileTransport 的 quirks** — ① d.Close() needs more bytes → skip defer in e2e harness, Go GC reaps；② 2-cmd fixture needs double prompt lines；③ device_rejected fixture text must NOT trigger platform's failed-when-contains
8. **eslint-disable directive placement 教训** — directive must immediately precede the rule-firing line (e.g., `}, [])`), not the function signature line. ESLint 规则触发行 ≠ 函数声明行
9. **JSX.Element 显式返回类型 = tsc -b 严格模式 fail** — React 19 + tsconfig configuration, `tsc -b` strict mode rejects global JSX namespace。Switch to type inference. Standalone `tsc --noEmit` works but `tsc -b` (build path) fails

### Cost Observations

- Model mix: yolo mode sequential, single session 主线 + gsd-ai-researcher 仅做 Phase 54 立项 RESEARCH (3 sources)，无多 subagent 派发
- Sessions: 1 main session (Phase 50-55 sequential) + 1 audit-milestone session
- Notable: 整个 milestone 主体在 1.7 工作日内单 session 完成 (06 → 08)。yolo mode + sequential phases 让 wall-clock 紧凑。Phase 54 4 atomic commits (A1+A2 unlock → 全覆盖 → docs → UAT) 各 ~25min
- Tech-debt cleanup (Phase 55) ~20min wall-clock (前端 14min + 后端 5min + verification 1min) — 清扫力度与 v1.18 Phase 49 (gap closure ~30min) 相近

---

## Cross-Milestone Trends

### Process Evolution

| Milestone | Sessions | Phases | Key Change |
|-----------|----------|--------|------------|
| v1.16 Tech-Debt | ~5 | 2 (40/41) | TECH-05/06 闭环 + validator 100% pass |
| v1.17 Recon | ~8 | 5 (42-46) + Phase 47 | observe-only 对账引擎 + 5 R-rounds 演进 + UAT 9/10 PASS |
| v1.18 Components | 1 | 1 (48) | 3-wave 隔离 + D-id 全覆盖 + site-visit UAT 推迟 |
| v1.23 Contract | ~6 | 7 (69-75) | 契约全量落地 + lint 门禁上线 + 实机 UAT |
| v1.24 Visual+Debt | ~9 | 8 (76-83) | 波次化并行（零重叠文件域）+ 契约双向断言 + 红绿验证 + 收口前 Lead 独立复核 |
| v1.19 Port Write | 1 main + 1 audit | 6 (50-54 + 55) | 5-phase build order + Phase 51 mock 漏洞 Phase 54 闭环 + Phase 55 tech-deb 闭环 |

### Cumulative Quality

| Milestone | Tests | Coverage | Zero-Dep Additions |
|-----------|-------|----------|-------------------|
| v1.16 | validator 165/165 pass | n/a | n/a |
| v1.17 | recon 5-layer test set | n/a | 6 backend recon 端点 + 4 frontend drawer + R5 state machine |
| v1.18 | 21 collector + 5 pipeline + 3 handler + 5 operlog regression | n/a | 1 pkg + 6 textfsm + 1 cron helper + 1 endpoint + 1 tab |
| v1.23 | 3960 用例（vitest 全量） | n/a | eslint 0 problems + 门禁归零 |
| v1.24 | 3960 用例（vitest 全量，独立复跑）+ 5 严格权限单测 + 7 门控组件测试 + 3 NAV-01 回归（红绿验证） | n/a | 1 中间件 + 1 门控 helper + 1 权限码真相源 + 2 契约脚本（CSS / 权限码） |
| v1.19 | 28 mock + 10 e2e + 7 handler + 5 router + 11 migration + 25 operlog + 64 vitest | n/a | 1 pkg + 15 template + 1 menu helper + 1 model + 1 migration + 6 endpoint + 2 component + 6 wrapper + 1 constants |

### Top Lessons (Verified Across Milestones)

1. **Pre-existing test failure 兜底 (Phase 40/41 + 48 + v1.19 共验证)** — 每个 phase 实施前在 baseline commit 跑同组测试，规避假回归指控
2. **Migration layered idempotency (Phase 33/44/48 共验证)** — DROP IF EXISTS 在 GORM AutoMigrate 上不工作，改用 `pg_indexes` 探测 + count-then-insert
3. **operlog module/chinese-name 双轨 (Phase 34 + 48 + v1.19 共验证)** — 强制使用 ops asset 中文常量，避免 module 字段在审计回路漂移；v1.19 operlog module = "端口管理" (AUDIT-01)，menu parent name = "端口状态" (D-07)，两者解耦独立
4. **基线审计后允许部分 deferred (Phase 41 29→ + Phase 48 3 site-visit + v1.19 7 site-visit)** — observe-only 策略延续，真机/外部依赖类 UAT 显式 partial 而非阻塞代码合入
5. **CR + Lint 在 phase 末尾加入，不在开头 (Phase 32 P1+P2 + 48 + v1.19 共验证)** — 实施期先跑通，review 期 catch 全，避免过度返工。v1.19 特别：Phase 53 review 抓出 WR-02/IN-01/IN-02/CR-01/CR-02 → Phase 55 集中清理
6. **mock-based service test 与 service code path integration 之间有 gap (v1.19 unique lesson)** — mock 简化便宜 + service nil-guard 让 unit test 干净，但 service.executeWrite → transport layer → parseConfigError 链路只走 mock 而从未在 integration 层执行。Phase 54 e2e 闭环该 gap。Future milestones 在 mock-based service test 阶段 plan 中需明确 "Phase X+1 = integration e2e"

---

## Milestone: v1.18 — 网络设备硬件清单 (Device Component Serials)

**Shipped:** 2026-07-04
**Phases:** 1 (Phase 48) | **Plans:** 3 (48-01/02/03) | **Sessions:** 1 large orchestrator + 2 plan executors (Wave 1 / Wave 2) + 1 plan executor (Wave 3) + orchestrator UAT

### What Was Built

- **`ops_asset` 4 组件列 + `sys_data_reconciliation.recon_category`** — schema 基础加 partial unique index 切换 + 字典 seed,migration_201 全 idempotent
- **`internal/services/component_collector/`** — 17 文件新包,纯解析层:ComponentSet 数据模型 + OwnerResolver stack containtedIn tree + 32-hop cycle guard + 4 种收集器(SNMP ENTITY-MIB 单 GET / Huawei CLI / Ruijie CLI / cmd_dispatcher)+ fixtures_loader
- **6 TextFSM 模板** — `huawei_vrp_display_device_esn` / `display_interface_status` / `display_interface_transceiver` + `ruijie_os_show_version_modules` / `show_interfaces_transceiver_ddm` + modified `show_interfaces_status`
- **OpsAssetWriter UPDATE-only + 父缺失降级** — D-02/D-03/D-04,从不 INSERT 资产
- **ReconciliationEmitter** — `conflict_type="F"` + `recon_category="component_serial"` 双 idempotent
- **operlog.RecordBackground** — cron-path 无 `gin.Context`,operator="system-cron",沿用 Phase 34 的 25 OperType 常量 + 11 sensitive keyword 不回归
- **DeviceInfoCollectionService cron hook** — `collectComponentInfo` + `runTwoStepTransceiverPipeline`(D-10 编排),失败仅 Warnf 不阻塞 chassis update(D-14)
- **POST /ops/asset/components + Asset type 4 可选字段 + ComponentListTab** — expandable row 模式,主表不污染

### What Worked

- **3-wave 分层执行** — Wave 1 schema 独立 commit 落地后,Wave 2 collectors 纯 library 隔离开发,Wave 3 pipeline 把 collectors 接入生产路径 + operlog + UI,各 wave 可并行 worktree 节省 wall-clock
- **D-id-driven 设计** — 14 个 D-ids(D-01~D-14)在 48-CONTEXT.md 锁定,executors 在 Wave 1/2/3 按 D-id 标签 commit 任务映射明确,无设计漂移
- **Homegrown textfsm 限制识别早** — 在 Wave 2 落第一个 template 时发现 `internal/templates/textfsm.go` 不支持 `Continue.Record` 习语,改用 single-state Record-on-final-field pattern 规避,无生产 bug
- **Site-visit UAT 显式推迟** — RESEARCH §Environment Availability 显式声明本环境无 S8700/RS8607E,3 项真机 UAT 在 48-HUMAN-UAT.md 标记 device_needed + partial 状态,不需要在 CI 闸门卡住代码合入
- **Migration idempotency** — migration_201 IF NOT EXISTS + DO $ pg_indexes 检测 + count-then-insert 三层防护,在共享 PG 实例上对 migration 二次重跑也 PASS

### What Was Inefficient

- **LSP 假阳性问题** — worktree 隔离导致 LSP 对 worktree 内 `.go` 文件持续报 `undefined: EntityRow/ComponentSet/OwnerResolver` + `could not import github.com/...`,必须切回 main 后才能验证(影响计划可见性,无实际效能损失)
- **`gsd-sdk worktree cleanup-wave` `empty_manifest` bug** — `Write` tool `/tmp/...` 路径与 `Bash` mktemp 路径不匹配,需手动 `git merge` worktree branch 救场,损失 ~5 min
- **git stash pop 误操作** — Wave 2 清理阶段误碰 stash list 中另一 session 的 stash,`git stash -u` 报"No local changes to save" 后 `git stash pop` 仍弹出其他会话的 stash 致 3 文件 UU 冲突,`git reset --hard HEAD` 救场(1 min 影响),但暴露 git 操作风险未防范
- **STATE.md 里程碑归属修正** — Phase 48 v1.18 milestone,但 STATE.md frontmatter 残留 v1.17 标签(user feedback 后才发现),原本应在执行 Wave 1 之前修正;没有 pre-execution milestone 校验
- **11 个 pre-existing test failure 调查** — Wave 1 末/2 中各发现一批 ops/system 测试 fail,要在 `b8fd2f45` (Phase 48 改动前) baseline 上跑 11 次确认非回归 ~15 min
- **Ruijie transceiver textfsm `Continue.Record` latent bug** — `templates/ruijie_os_show_interfaces_transceiver.textfsm` 用了 `Continue.Record` 在 homegrown parser 下行为不一致(无生产 caller,潜在),Wave 2 没修(出 Phase 48 scope),记为后续清理债

### Patterns Established

- **UPDATE-only ops_asset 写入** — 组件序列号是设备派生数据,**永远 UPDATE 不 INSERT/DELETE**,防止 ad-hoc 资产创建污染主表(D-02/D-03)
- **Parent-missing degraded mode** — 父交换机不在 ops_asset 时 `parent_asset_id=NULL` + Warnf,**不自动 INSERT 父资产**(D-04)
- **D-06 sibling-column anomaly via `recon_category`** — partial unique index on `(asset_id, conflict_type, recon_category)` 让 same (asset, type) 可同时存在 legacy + component_serial 异常行
- **D-10 两步流水线 at cmd_dispatcher level** — `status → transceiver only if up`,down ports 跳过,运维省 CLI session 资源
- **D-12 cycle guard 32-hop** — OwnerResolver `entPhysicalContainedIn` 树重建防环
- **operlog.RecordBackground parallel to operlog.Record** — cron-path 无 `gin.Context` 时的等价调用,operator="system-cron"
- **Migration layered idempotency** — `pg_indexes` 检测替代 `DROP IF EXISTS` (GORM 不带 IF EXISTS 是已知坑)
- **Site-visit UAT 在 plan 阶段显式声明 deferred** — 现场/无设备环境,自动化 UAT 不能涵盖的项,在 PLAN.md + RESEARCH.md §Environment Availability + HUMAN-UAT.md 三处显式记录

### Key Lessons

1. **Milestone 归属校验应在 execute-phase 之前完成** — STATE.md frontmatter milestone 必须先校验与 ROADMAP.md 一致,避免 Wave 1 之后才发现 v1.17 vs v1.18 错位
2. **Pre-existing test failures 要 baseline 兜底** — 对任何新 phase,实施前后在 commit baseline 跑同一组测试是规避"被误判为回归"风险的标准动作(已在 Phase 48 走通)
3. **Homegrown TextFSM parser 限制要早期识别** — `Continue.Record` 不支持,executors 落 template 时必须意识到;新 template 一律用 single-state pattern
4. **LSP 在 worktree 内会假阳性** — worktree 在 LSP 视角是孤立目录,跨模块 import 标红是常态;只信 `go build ./...` 不信 LSP 红线
5. **`gsd-sdk worktree cleanup-wave` 路径一致性** — Write 工具与 Bash 子进程的 `/tmp` 路径不一致会让 manifest 读取 empty,需要建立 adapter 或用 `git merge` 兜底(此项暂记,等 SDK 修复)
6. **Cron-path 操作必须平行于 handler-path operlog** — 不能在 cron 路径漏 audit,也不能让 cron 阻塞 fail chassis update;D-13 + D-14 是这条 invariant 的硬性约束

### Cost Observations

- Model mix: yolo mode, executor 默认 model(中量 sonnet 占主)
- Sessions: 1 主 session + 4 subagent (Wave 1 + Wave 2 plan + Wave 3 plan + UAT schema_check)
- Notable: 整个 milestone 主体在 1 个工作日内单 session 完成,worktree 隔离在 wall-clock 上提供 ~30% 加速(Wave 1/2/3 各自独立 commit 后仅需 merge)

---

## Milestone: v1.20 — 网络设备 VLAN + 端口绑定

**Shipped:** 2026-07-10
**Phases:** 1 (Phase 56) | **Plans:** 5

### What Was Built

1. Vendor Template 扩展 — `vendor_port_template.go` +6 entries (3 vendors × 2 actions), 9 render functions, MAC 格式锁定 (Huawei/H3C = `AA-BB-CC-DD-EE-FF`, Ruijie = `aabb.ccdd.eeff`)
2. PortWriteService + 4 validators + Extra-map audit carrier — `SetAccessVlan` + `PortBinding` + 4 sentinels + VLAN pre-state NoOp 短路 + `PortResult.Extra`
3. Handler + Router + Audit — 2 kebab HTTP endpoints + 4 sentinel→HTTP-400 + buildAfterValue 读 Extra + 2 permission registry rows
4. Frontend 2 Modals — `SetAccessVlanModal` + `PortBindingModal` + 2 networkApi wrappers + types 扩展
5. E2E + UAT Deferral + Docs — 11 new `TestE2E_*` + 10 fixtures + `56-HUMAN-UAT.md` 12 项 site-visit 推迟 + 3 文档同步

### What Worked

- **100% 复用 v1.19 基建** — vendorPortTemplate + pre_state_check + execSinglePort + PortWriteModal 全套无修改;5 phases 都基于既有模式延伸,zero new infra
- **plan revision 1 在 Wave 0 抓 3 BLOCKERS** — after_value via Extra / batch path / Wave 0 DB verification,避免 Wave 1-5 中途发现架构漏洞
- **scrapligo FileTransport mock SSH** — 11 e2e 测试完整覆盖 3-vendor × 2-action × variants,无真机依赖,CI 内可跑
- **Code review 在 Wave 0 后发现 4 CRITICAL** — shell injection (CR-01 batch path 绕过 validator)、bindOp 静默删除 (CR-04)、audit vlan 字段缺失 (CR-02)、modal pre-fill 丢失 (CR-03),fixer agent 一次解决

### What Was Inefficient

- **Wave 2 executor 撞 429 quota** — 中断后 code commit 已完成但 SUMMARY.md 未写;orchestrator 手动恢复(合并 + 验证 + 补写 SUMMARY)
- **build-linux.bat 误改后回滚** — 我擅自加 Step 7 verification,引入 LF/CRLF/bash 查找/findstr 嵌套问题链,最终用户提示后回滚;**未来不动已能工作的脚本**
- **Reviewer 在隔离 worktree 内偶发"工作区反还原"事件** — 看起来是 git checkout HEAD~3 之类操作未完成 HEAD restore;需要 investigation 但未影响最终结果

### Patterns Established

- **Validator 抽取为可共享函数** — single-port 和 batch path 调用同一份 `validateVlanIdRange` / `validateBindOp` / `validateIPAddress` / `validateMACAddress`,gin binding tag 作为入口第一道防线
- **`PortResult.Extra map` audit carrier** — 新 action 方法写 Extra,v1.19 方法留空;buildAfterValue 消费 Extra 写 after_value
- **batch path 必须在 service 层调 validator,不能只在 handler** — handler 端的 sentinel→HTTP-400 翻译是双路(single + batch),service 层加 validator 才堵住 shell injection

### Key Lessons

1. **Don't add verification to a working build script** — v1.20.1 binary 实际是好的,加 Step 7 verification 是无中生有
2. **Wave 2 配额中断恢复模式** — `git log worktree-branch` 看 commits + 检查 SUMMARY.md → 决定 merge-and-author-SUMMARY 还是 skip-and-retry
3. **Plan revision 1 在 Wave 0 必做** — BatchWriteRequest 4 字段 + audit Extra + DB 验证,这 3 项都是后期发现会扩散的

### Cost Observations

- Model mix: yolo mode, executor default model + 1 opus verification + 1 opus code-fix
- Sessions: 1 主 session 跨 6+ hours (含 2 次 sleep/wait 中断 + Wave 5 上下文长)
- Notable: Wave 1-5 sequential (parallelization=false) × 5 waves = wall-clock 主导;worktree 隔离在 cleanup merge 阶段加速,但 Wave 2 配额中断抵消了部分 gain

---

## Milestone: v1.21 — AD 双向同步 bug 修复 (AD Bidirectional Sync Bug Fixes)

**Shipped:** 2026-10-22
**Phases:** 6 (57-62) | **Plans:** 22 | **Tasks:** 45

### What Was Built

1. **Phase 57 — ad_push_queue (outbox-lite)** — migration_215 建表 + 部分唯一索引（活跃谓词单一事实源 `models.ADPushQueueActivePredicate` = `status IN ('pending','failed')`）+ 五条写路径同事务入队（Update / UpdateStatus / Delete / BatchDelete / profile.UpdateUserInfo）
2. **Phase 58 — 排水器 + compare-and-move** — in-process cron 排水（非 sys_job）+ 退避重试 + `NormalizeDN` 幂等短路 + 配置路由无歧义；7/7 plans
3. **Phase 59 — 登录回写守卫 + 部门对账视图** — 登录写 dept 前查活跃队列行，命中则跳过覆盖 + WARN（替代被证死的时间戳仲裁）；`v_dept_ou_mismatch` 只读 VIEW
4. **Phase 60 — 软删恢复安全** — ad_dn 校验后恢复（不一致拒绝 + 审计），登录与批量同步共用分类逻辑
5. **Phase 61 — 影子表完整性** — 删除清理两道闸门（LDAP 0 条拒绝 / 拟删 >50% 拒绝）+ 组成员 Delete+Create 事务化 + singleflight 挂载
6. **Phase 62 — 卫生收尾与死代码清理** — HYGIENE-03 死代码删除（`user_ou_service.go` 整文件 + `BatchMoveUsersToNewOU`）+ HYGIENE-04 嵌套组成员过滤（零 DDL）+ HYGIENE-05 OU DN 冲突 typed error 显式化

### What Worked

- **三方 subagent 审计砍掉了两个错误方案** — 时间戳仲裁（GORM 自动刷新 `updated_at` 使其恒失效）与完整 Outbox/FieldRule 引擎（过度设计），落到 ~400 行 outbox-lite 核心层
- **真环境实锤替代「看起来通过」** — 08:05 轮全量同步 3025 次采样在 0.887s 重放窗口内逐条 2815|59 零偏差；嵌套组成员污染 8→0 靠重放自愈；登录守卫双半段（拦截 / 正常覆盖）与软删 DN 漂移拦截全部端到端
- **采样器纪律** — v5 先查 `sys_ad_sync_log` 确认轮次时刻再布防（v4 窗口 02:58-03:12 完全打空，因为全量 cron 是 3 小时周期）
- **quick task 261021-iqq 把「重启才生效」变成热重载** — 固定 30s tick + `drainGate.shouldDrain` 门控，改配置 ≤30s 生效（实测 12s / 29s），无需重启后端
- **里程碑级集成审计补齐单 phase 盲区** — 集成检查器核到 8/8 集成点 VERIFIED + E2E 流 A-D 全通过，并抓出 QUEUE-03 退避窗注释矛盾与 HYGIENE-04 nil 路径可达性两条 finding

### What Was Inefficient

- **Phase 62 引入生产回归** — 嵌套组过滤后 kept 为空 → `tx.Create(&[])` 触发 GORM ErrEmptySlice → 事务回滚 → 整轮同步中止（14:05 轮 failed、error_count=1）。修复 = 过滤块后补早返回 + 回归用例 + 重建二进制发版
- **一次 write 调用漏参把 `58-VERIFICATION.md` 覆成 4 字节** — 用 `git show HEAD` 逐字节恢复后重做（git diff 证明正文零改动）
- **UAT 文档的 frontmatter / 状态字段是解析器契约** — 缺 byte-0 frontmatter、首个 `status: failed` 被优先命中、多个 `status:` 字段并存，都会让 audit-uat 把 18 项真待办报成 32 项或反之
- **ORM 边界要用例锁** — 过滤后空切片、`ON CONFLICT` 合并路径、终态行退出部分唯一索引，这些都不是「显然」行为，必须有 RED→GREEN 用例

### Patterns Established

- **活跃谓词做部分唯一索引** — `status IN ('pending','failed')` 让终态行（pushed/dead）自动退出唯一索引，后续编辑自然新增行而非合并
- **冲突显式化三件套** — typed error（`ErrOUDNConflict` + `Unwrap` 哨兵）+ 计数（`FailedDepts` / `Errors`）+ 人工通道（`UpsertMappingForce` + 预检告警），杜绝静默互相覆盖
- **「重放删除污染行」优于「标记污染行」** — 零 DDL、无数据修复脚本，下一轮成功同步即自愈
- **跨实例并发是既有事实** — 排水器实际跑在另一台机器（10.62.10.39），本地实例只连库；单实例假设会让验证结论失真

### Key Lessons

1. **过滤后空集合必须早返回** — 把空切片交给 GORM `Create` 会得到 ErrEmptySlice 并回滚事务，把局部空集放大成整轮失败
2. **采样窗口必须先对齐真实事件时刻** — 事件周期（3 小时 cron）与窗口不重叠时，再密的采样也只会得到「全恒定」的空结论
3. **发版后必须实测，不能只看 `/health`** — 20:05 轮的 V 形是旧二进制造成；只有 08:05 轮零偏差才是新代码生效的证据
4. **文档状态字段是工具契约，不是格式** — frontmatter 必须在 byte 0、同一语义只保留一个权威字段名

### Cost Observations

- 6 phases / 22 plans / 45 tasks，主 session 跨多轮 goal 续跑；真实域控 + 真实登录的人工验证占了本里程碑最大比重（58/59/60 三个 phase 的终态都靠真环境实锤）
- Notable: 62-01/62-02/62-03 三 plan 的 checker 只提 MAJOR-2 + MINOR-3 + 标签闭合类问题，无架构级返工；代码事实先落 PLAN 再派 executor 的做法明显降低返工

---
## Cross-Milestone Trends

### Process Evolution

| Milestone | Sessions | Phases | Key Change |
|-----------|----------|--------|------------|
| v1.16 Tech-Debt | ~5 | 2 (40/41) | TECH-05/06 闭环 + validator 100% pass |
| v1.17 Recon | ~8 | 5 (42-46) + Phase 47 | observe-only 对账引擎 + 5 R-rounds 演进 + UAT 9/10 PASS |
| v1.18 Components | 1 | 1 (48) | 3-wave 隔离 + D-id 全覆盖 + site-visit UAT 推迟 |

### Cumulative Quality

| Milestone | Tests | Coverage | Zero-Dep Additions |
|-----------|-------|----------|-------------------|
| v1.16 | validator 165/165 pass | n/a | n/a |
| v1.17 | recon 5-layer test set | n/a | 6 backend recon 端点 + 4 frontend drawer + R5 state machine |
| v1.18 | 21 collector + 5 pipeline + 3 handler + 5 operlog regression | n/a | 1 pkg + 6 textfsm + 1 cron helper + 1 endpoint + 1 tab |

### Top Lessons (Verified Across Milestones)

1. **Pre-existing test failure 兜底(Phase 40/41 + 48 共验证)** — 每个 phase 实施前在 baseline commit 跑同组测试,规避假回归指控
2. **Migration layered idempotency(Phase 33/44/48 共验证)** — DROP IF EXISTS 在 GORM AutoMigrate 上不工作,改用 `pg_indexes` 探测 + count-then-insert
3. **operlog module/chinese-name 双轨(Phase 34 + 48 共验证)** — 强制使用 ops asset 中文常量,避免 module 字段在审计回路漂移
4. **基线审计后允许部分 deferred(Phase 41 29→ + Phase 48 3 site-visit)** — observe-only 策略延续,真机/外部依赖类 UAT 显式 partial 而非阻塞代码合入
5. **契约断言必须双向(Phase 76→77 反查)** — 正向「规则存在」会让「永不命中的死规则」通过检查；须同时断言「不得出现已知死形态」，且锚定当前依赖版本里真实存在的选择器
6. **回归测试必须红绿验证(Phase 81)** — 写完先回退实现确认它会红；否则测的是空气（本轮曾因 fixture 字段名错误 + 缺少缓存预置而假绿）
7. **修复类工作必须留前后对照(Phase 79)** — DB 层 + HTTP 层各留「修复前/修复后」实测，否则修复后无法证明缺陷曾经存在
8. **收口前 Lead 独立复核全量闸门(Phase 83 D5 后)** — 不采信 agent 的「全绿」声明；独立复跑 vitest 确认 flaky 与真回归之别
5. **CR + Lint 在 phase 末尾加入,不在开头(Phase 32 P1+P2 + 48)** — 实施期先跑通,review 期 catch 全,避免过度返工
