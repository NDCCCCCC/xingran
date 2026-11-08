> ℹ️ 本文件为各里程碑**收档时点快照**（含当时的 Known Gaps / deferred 表述），仅供追溯；当前状态以 `.planning/STATE.md` frontmatter、`.planning/ROADMAP.md` / `.planning/MILESTONES.md` 为准。例如本文 v1.24 段落「Known Gaps 转 v1.25 候选，用户裁定暂不立项」是当时记录，实际 v1.25 已全部立项并 SHIPPED（见 PROJECT.md:201）。

# Milestones

## v1.53 开放项全量清偿 (Shipped: 2026-11-08)

**Phases completed:** 8 phases (134-141) | **Requirements:** 24 REQ（18 交付 + 6 项人工验证跟踪，不伪造闭环）| **21 条开放项全覆盖** | **回归闸门**：go build / go test 全仓 / tsc / lint:ci / vitest 592 文件 4128 用例全绿

**Key accomplishments:**
- AD push 队列 enable 方向 + SAFE-01 身份校验/双向压制、全量同步指数退避节流、常态化同步漂移告警服务
- 补齐 main.go docs blank import、两生成脚本改产物到 internal/docs 并修 bat 层级、删除 docs/ 下 5 份 Swagger 死文件
- 接通用户导出/工单评价端点与前端入口，用 migration 236 seed 闭环 OQ-7 五个读权限码恒 403
- GetDiscoveryResults 补持久化查询链路、rpa-worker 双副本去 browser stub 并修复前端副本基线编译
- scheduler Rotation 抽共享纯函数对齐 service 双路径、3 厂商×3 参数端口物理参数写全闭环、3 处排序收尾落产品决策结论
- 对账 MV 30s 超时配置化（默认不变）、5 个登录加密 benchmark 空桩补国密实测并标注测量边界
- 从 git 历史精确重建 5 份缺失 UAT 清单 + 79 条全部待执行的统一跟踪表与 GSD 产物回写口径
- 全仓清理临时清单权威引用、归档临时清单与历史 UAT 重建件、用 GSD milestone complete 把权威交回自动生成的 ROADMAP/REQUIREMENTS/STATE/MILESTONES

**Known Gaps（登记，非阻断）**：
- **人工验证**：6 项（VAL-01~06）交付物已落盘，实际执行受生产 PG 只读 / 写操作授权 / 现场真机 / DBA 回传约束，跟踪表见 `.planning/notes/261108-human-verification-tracker.md`（79 条全部待执行、0 勾选）。
- **LOCKED 残余**：A8 四项 + B13 两项只落结论不改代码，见 `.planning/notes/261108-v153-locked-residuals.md`。
- **真实缺口（2026-11-08 收档后补登记）**：`ImportDiscoveredDevices`（`internal/services/device_discovery_service.go:509-523`）仍是简化实现、仅返回 `len(deviceIDs)`，不真正写入 `NetworkDevice`；Phase 137 (FIX-07) 按范围约束未动，属本次临时清单 B10 之外的独立缺口，此前未进入任何 deferred 登记，现补入 `STATE.md ## Deferred Items`。
- **Phase 自报边界（同上补登记）**：FIX-01 实机 AD/LDAP UAT｜FIX-04 三厂商物理参数真机逐型号取证 + 型号级能力矩阵规则｜FIX-09 端到端 HTTP P99 压测（本次仅计算路径近似基线）｜FIX-11 前端告警页 / WS / 企微钉钉 webhook｜FIX-13 生产库 migration 236/237 待执行｜FIX-05 批量物理参数路径未接入。逐项见 `.planning/milestones/v1.53-phases/*-SUMMARY.md` 与 `STATE.md ## Deferred Items`。
- 上述补登记为**显式披露**，非 GSD `audit-open` 可扫描的 artifact；收档时 `audit-open` / `audit-uat` 均为全清。

---

## v1.52 运维域数据权限统一收口（终态） (Shipped: 2026-11-07 · Archived: 2026-11-08)

**Phases completed:** 4 phases (130-133) | **Requirements:** 7/7 (DS-01~07) | **Commits:** 24 | **PR:** #75 merged `b56f1286`

**Key accomplishments:**

1. ✅ **ScopeProvider 单一真实源**（`pkg/datascope`）—— 可见部门集统一实现（All/0/unset 特判短路、Custom/Dept/DeptChild 展开、Self 资源语义钩子、异常枚举 fail-closed、多角色 MAX），`DataScopePermission` 注入 ops 全域 26 处挂载，`user_service` 迁移到单一真实源。
2. ✅ **四域读端点 + Excel 导出独立收口** —— 20 个读函数统一「有效机构 ∈ 可见集」，缓存 key 增 scope 维度，交付 `131-CACHE-INVENTORY.md`，403 越权矩阵 + 跨用户缓存对照 + PG contract 扩展。
3. ✅ **四域写端点收口** —— Create/Update/Delete/Batch/positions 单条越界显式 403（`NewWithHTTPStatus` 规避框架 401 坑），批量按可见集裁剪，backfill 三层注入，Excel 导入逐行校验，前端「本次范围」四类文案。
4. ✅ **扩展实体与收尾** —— asset/serverRoom/dedicatedLine/roomDevice 收口（115 个语义断言）、vdi 迁移、删除 `pkg/middleware` 三份死实现 + 死测试、前端 DeptSidebar 联动、5 项显式豁免面登记。
5. ✅ **独立审查 + CI 红灯修复** —— 3 P0（迁移 235 误嵌 / 权限码 401 / backfill 全库扫描）全修；PR #75 全 checks pass 后合并；回归闸门全绿（go build / go test / vitest 4128 / lint:ci 0 / diff coverage 83.37%）。

**Verification**: 全量闸门全绿（详见 `.planning/MILESTONE-v1.52-SUMMARY.md` §回归闸门）。`closeout_type: override_closeout` —— GSD `audit-open` 前置审计在收档时检出 3 项 `uat_gaps`（`129-HUMAN-UAT.md` / `129-UAT-BACKFILL.md` / `133-UAT.md`，均 `open_scenario_count: 0`），已按 [A] Acknowledge 全部登记为 deferred；另有 3 项历史 debug suppression 结转。**Known verification overrides: 3 newly acknowledged, 3 carried forward**。人工 UAT 挂账见 STATE.md Deferred Items 表（A2/A3/A5/A8）。

**归档产物（2026-11-08 /gsd-complete-milestone）**: `.planning/milestones/v1.52-ROADMAP.md`、`v1.52-REQUIREMENTS.md`、`v1.52-phases/`（6 目录）、`v1.52-quick/`（4 目录）；路径映射见 `.planning/ARCHIVE-INDEX.md`。

---

## v1.25 全站权限治理与样式收敛 (Shipped: 2026-10-30)

**Phases completed:** 10 phases, 41 plans, 0 tasks

**Key accomplishments:**
- (none recorded)

---

## v1.24 视觉回归修复与遗留债务清偿 (Shipped: 2026-10-28)

**Phases completed:** 8 phases (76-83) | **Plans:** 0（subagent 直接执行，按刀原子提交） | **Tasks:** 8 phase × 9 项闸门
**Requirements:** 18/18 Satisfied | **Git range:** `dbdd1707` (roadmap CREATED) → `4477b2bc` (夹具清理记录) | 46 commits / 104 files / +5,419 −2,232
**Timeline:** 2026-10-28 09:19（需求定稿）→ 12:59（收口），约 3.5 小时

**Delivered**: 修复用户主诉「文本输入框双重外框」（309 个输入框全站受影响），并清偿 v1.23 收档结转的 4 项遗留债务（#16 行操作权限越权 / 3f130 空因展示 / F5 跳 /dashboard / ColumnConfigModal 豁免注释）与 2 项延后技术债（lint 存量 / react-refresh + devices 拆分）。

**Key Accomplishments**:

1. ✅ **双框根治 + 全局 CSS 一致性审计（用户主诉闭环）** — 根因是 `index.css:871` 的 `.ant-input{border:...!important}` 命中 antd 6 affix-wrapper 的**内层** input（antd 原生契约要求内层 `border:none`）；复位块 `.ant-input-affix-wrapper > input.ant-input`（6 项、**刻意不写 box-shadow**，焦点环归 wrapper）+ CSS 审计登记册 **36 条**（P0 已修 8 / P1 待修 7 / P2 登记 19 / 保留 1 / 样板 1）；**双框元素数 3 → 0**（harness 42 行 + 15 真实页 + 登录页，明暗双测）；错误态由灰 `rgb(219,215,206)` → 品牌红 `rgb(186,54,48)`
2. ✅ **Phase 77 视觉验证反查出 Phase 76 自身的 P0 缺陷** — VIS-03 的选择器写成 antd 6 **不存在**的 `ant-input-affix-wrapper-status-error`（antd 把 status class 打在 wrapper **自身**，名为 `ant-input-status-error`）→ 永不命中的死规则，叠加 `allowClear` 默认开 → **站内绝大多数输入框错误态一直不变红**；更糟的是契约检查 C5 **校验的正是这条死规则** → 检查通过而规则从未生效。已修并把 C5 升级为**双向断言**（正确 class 必须存在 + 死变体不得出现）
3. ✅ **#16 行操作权限前后端双端闭环（三层证据）** — 根因是**两个叠加病灶**：组级 ANY-of（`router.go`）+ 模块级降级（`permission.go` Step 3 把 `system:user:remove` 降级匹配到页面本体）→ 只读角色调 delete **200 且真删数据**；新增 `RequireStrictPermissions`（只做 Step1 精确匹配）+ `/system/users` 读组/写组**兄弟分离**（逐动作单一权限点）+ 前端 `useActionPermission` 三态门控 + 权限码单一真相源双向校验；证据：**DB 层前后对照**（旧 8/8 放行 → 严格 8/8 拒绝）+ **HTTP 实测**（8/8 写接口 403，含决定性对照：临时加 `xtgl` → delete 200 且真删除）+ **前端组件测试 7/7**
4. ✅ **F5 刷新回退 /dashboard 修复（红绿验证）** — catch-all 区分「菜单未就绪」(InitializingFallback) 与「真无路由」(跳 dashboard) + `saveLastPath` 加守卫防自毁反写 + 抽出 `lastPathStorage.ts`（顺带清 3 处 react-refresh）；**新增 3 条回归测试并做红绿验证**（回退实现 2 条失败 / 修复后 3 条全绿）
5. ✅ **lint 存量清零 + 门禁防回退** — 51（17 errors + 34 warnings）→ **eslint 0 problems**；预算双写归零（`package.json --max-warnings 0` + `assert-lint-budget` 0/0）+ react-refresh 33 三类分流 + 豁免理由表 159 → 38 且 `check-exemption-reasons.mjs` exit 0 并接入 `lint:ci`
6. ✅ **devices 页拆分（HYG-04）+ #16 前端侧复用** — 714 → **306 行（-57%）**：D1 列定义工厂(130) + D2 搜索区构造函数(63) + D3 handlers+行操作+权限门控(413→278) + D4 页面 JSX 外壳(194) + D5 行操作再拆(192)；`reloadRef` 外部注入链路 4 段逐字保留；`columns` useMemo **故意不完整**依赖未被"修正"；v1.19 port-write 62 测试零回归且 diff 为空

**Verification**: 收口前由 Lead **独立复核**（不只采信 agent 报告）——闸门六道全绿：type-check 0 / eslint **0 problems** / lint 预算 0-0 / 豁免表 38↔38 / 权限码双向零漂移 / CSS 契约 ALL PASS；`go build ./...` + `go test` 相关包 PASS；**vitest 569 文件 3960 用例全绿**（独立复跑，确认 1 例超时为环境 flaky 非回归）；`npm run build` ✓

**Known Gaps（转 v1.25 候选，用户裁定暂不立项）**:
- ① 操作级 RBAC 全站治理（108 页面 + 340 路由；`roles/menus/depts/posts` 4 组 ANY-of 拆分 + `pkg/permission/config.go` 死表接线 + 动作级 F 型菜单播种）
- ② 样式层收敛到 antd 6 variant（方案 C，需先重构 AntdThemeBridge 明暗色板）
- ③ 权限模型三缺口（`data_scope` 未被列表查询消费 / `GetUserPermissions` 不过滤 `menu.status` / Step 2 语义易写反）+ 生产库无迁移记录表 + GORM schema 漂移
- ④ Phase 77 视觉新发现（暗色 `--input-bg` 第 3 处就地重定义 / 暗色 Select 白底两套边框体系 / `.dept-tree-search` 死规则 / antd 6 弃用警告 2 类 / InputNumber 错误态仍灰）
- ⑤ `room-devices/index.tsx` 815 行（比 devices 更长，未扩面）

**生产库状态**: #16 验收夹具已按用户指示在单事务内清理，6 项残留自检全 0，`sys_role` 7→6、`sys_user` 4232→4231（恰为夹具本身），**既有行零改动**；证据（角色 id / perms 集合 / HTTP 返回码 / 前后对照表）已固化在 `.planning/notes/2026-10-29-v124-perm-fixture.md`

---

## v1.23 前端契约全量收尾与遗留清偿 (Shipped: 2026-10-28)

**Phases completed:** 7 phases (69-75), 121+ 文件迁移 | **Requirements:** 22/22 Satisfied

**Delivered**: 修复用户主诉「文本输入框无法一键删除」（TextArea 74 处 0 覆盖根因），把 v1.22 契约推到全量落地（裸组件归零、P65 豁免 127→0），清偿 7 项 debug 遗留与流程积压，lint 债务大幅清偿并上线防回退门禁。

**Key Accomplishments**:

1. ✅ **TextArea 一键清空契约（用户主诉闭环）** — AppTextArea 独立契约组件 + SearchForm textarea 字段类型 + eslint 成员表达式/解构形态双 selector（探针验证非假绿）+ 存量全量接入；实机 UAT 抽查 3/3 通过
2. ✅ **COMP-05 全量收尾** — W1-W6 六波迁移 121 文件（Input/Select/TextArea），P65 豁免 127→0、无豁免口径裸组件命中全归零；Select 形态七类分类器全量执行；v1.22 冻结契约零破坏（port-write 62 测试全绿）
3. ✅ **7 项 debug 遗留全终结** — #1 Linux 日志（不复现）/ #2 Cisco MAC 采集（b022.7a2e.4a4f 验收）/ #3 MAC 格式 Go 侧防线 / #4 双执行（双实例部署）/ #5 历史查询双侧归一化 / #6 scrapligo（关单）/ #7 空因可诊断端点——多项实证反转原认知（acknowledge ≠ 已定位）
4. ✅ **lint 债务清偿 + 防回退门禁** — errors 81→17（全部为范围外存量）、warnings 1366→**34**、no-explicit-any 958→**0**、no-unused-vars 364→0、no-console 5→0；`--max-warnings 34` + assert-lint-budget.mjs 断言上线
5. ✅ **浏览器实机 UAT** — 68.1 §8 全表 18/19 通过 + FAIL-1 分页恢复 bug 修复实机复验（PageTable mount 硬编码回写根因）；§8 回写关闭 68.1 结转债务；quick 目录 94 个归档清零

**Verification overrides**: 7 个 phase 未走 GSD verifier 流程（subagent 直接执行 + 每波真实闸门：type-check/vitest/eslint 预算/build + UAT 实机），override 收档；3 项审计开放项 acknowledge（debug 已 resolved / UAT 0 pending / CONTEXT 6 问已执行回填）

**Known Gaps（转 v1.24 债）**:
- #16 行操作权限需 PG 环境 UI 复测定责（sqlite 角色创建 500 环境限制）
- 3f130 绑定 user_id（业务核对后数据修复，代码已能诊断该空因）
- F5 刷新落 /dashboard（UAT 顺带发现）
- ColumnConfigModal:136 缺豁免注释（低优先级）

**全程质量基线**: type-check 0 错误、vitest 全量绿（3946 tests）、eslint errors 17（≤基线 83）、build 通过、零新依赖

---

## v1.22 前端公用组件整治 (Shipped: 2026-10-27)

**Phases completed:** 7 phases, 7 plans, 0 tasks

**Key accomplishments:**
- **P0 止血（Phase 63）**：usePagination.ts:122 分页口径修正为「共 N 条 / M 页」并全线生效（此前 51 个列表页把总条数算成页数）；backups/discoveries setTotal 断链接通恢复翻页；OU 树改服务端分页；7 处批量删除补 Modal.confirm 二次确认；71 处弹窗补提交中取消禁用（cancelButtonProps）；删除 3 处假入口（假编辑/假下载/空 Tab）；8 处 fixed 列与 scroll.x 配对修复；运行时浏览器冒烟 7/7 通过
- **反馈与可访问性（Phase 64）**：BaseEditModal 默认 maskClosable=false+keyboard 关闭防丢表单；7 处表格 loading 反馈；cache/job/jobLog 白名单内服务端排序（此前初版误判后端不支持）；52 处 catch 收敛到 handleApiError 唯一错误出口；73 处 icon-only 按钮补 aria-label（修复 38 处误写 children 的形态缺陷）
- **危险操作唯一入口（Phase 65）**：AppInput/AppSelect 契约组件落地（allowClear/showSearch 默认值下沉，rest 全透传+逃生口）；confirmDelete helper + ConfirmDelete 声明式 + BatchDeleteButton 三形态统一文案「将删除{target}，删除后不可恢复」；eslint no-restricted-syntax 4 selector + no-restricted-imports 双闸拦截裸组件；11 处批删迁移；存量豁免 147 文件登记
- **列表契约组件（Phase 66）**：PageTable（受控 props+插槽+三态内建）、SearchForm（schema 驱动+children 逃生口）、FormModal（onOk Promise 防重+脏检查三文案）、RemoteSelect（debounce+竞态守卫）四大件落地；system/post 双试点迁移；useTableManager 降格为内部引擎废双轨
- **存量迁移（Phase 67）**：discoveries 405→333、backups 1141→637、devices 1282→692、cache 974→454，四页全部切换 PageTable/SearchForm/confirmDelete 契约
- **一致性收敛（Phase 68）**：shared/statusHelpers.tsx 六域状态唯一落点；裸 hex 526→非豁免残留 32（lint 拦截+hex-exemptions 登记制）；any 52→21（any-exemptions 登记制）；日期格式 20 文件收敛 datetime.ts 常量；gzip 体积守卫进 CI 口径
- **收尾清偿（Phase 68.1）**：44 处删除类裸 Modal.confirm + 10 处破坏性 confirm + 14 处 Popconfirm 全部收口到 ConfirmDelete 唯一入口（剩 8 处合法点全部带行内理由）；契约 minor 7 项补课（aria-label=删除{target} 解耦、actionContext 批量语义、focusable 焦点落点、aria-busy、提交中禁取消）；豁免清单 181→159 且 100% 带结构化理由（exemption-reasons.json + 校验脚本）；operations 4 页迁移净 -375 行；独立 verifier 发现并修复 15 处 onDelete 丢 Promise 回归
- 全程质量基线：type-check 0 错误、vitest 3937/0、eslint 83 errors 零新增、build 通过；5 天 124 commits，534 文件 +29173/−7695

---

## v1.21 AD 双向同步 bug 修复 (Shipped: 2026-10-22)

**Phases completed:** 6 phases, 22 plans, 45 tasks

**Key accomplishments:**
- ad_push_queue 队列表以最小动作行落地：`uq_ad_push_queue_active` 部分唯一索引在 pending/failed 上夺锁实现 QUEUE-02 合并语义，PG/sqlite 双方言 DDL 幂等建表并由 Wave 0 sqlite 测试锁定行为
- 四字段 diff 纯函数 + 事务内 `sys_ad_config` 启用行前置检查 + 靠 `uq_ad_push_queue_active` 部分唯一索引夺锁的 upsert 入队（update_user 与 disable 两条入口），T-57-06 三重防线在 sqlite 上由合并测试直接锁定
- Update 的四字段 diff 入队与 Delete/BatchDelete 的 disable 入队全部落在业务事务闭包内、失败即回滚，队列表写不可用时 sys_user 无任何残留；HYGIENE-06 实际覆盖到三套 sqlite 环境（含计划外的 userCacheService 那一套）
- addomain 包五表排水测试基座（ad_push_queue 走迁移、其余三表复用既有 const 零副本）+ mockLDAPClient 调用计数 + F-F/F-G/F-15/F-16 四项 Phase 57 遗留缺口处置，为 Wave 1~6 的 RED 测试提供可判真的环境
- NormalizeDN/BuildOUDN 纯函数落地并接线唯一 OU DN 拼接点，MatchADConfigByDN 以逗号边界最长后缀唯一匹配锁定 D-02 契约（8 个用例含父子域反例），三处 First() 歧义收敛为两个语义分明的 addomain 入口，双份 getDefaultADConfigID 归零
- ADPushDrainer 落地 QUEUE-03 排水执行面：D-05 拾取谓词（next_retry_at IS NULL = 可拾取）+ 退避曲线纯函数 + 六状态跃迁状态机 + 按 ad_dn 归属分组（每组一次账号池故障切换）+ disable(UAC 514)/update_user(属性) 两条真实推送路径；F-11 禁令以 markFailed 单点判定锁死，F-12 四处时间戳 UTC 与 F-14 model default tag 一并落地，13 个 Drain/拾取用例全绿、Phase 57 sentinel 与全仓 `go test ./...` 零回归
- QUEUE-04 的 compare-and-move 执行面落地：闭包内重读最新值 → NormalizeDN 双侧归一化比较目标 OU → 一致跳过 `MoveUser`（幂等点）→ 不一致才移动并
- D-05 单旋钮三层齐备（migration_216 幂等 seed + 启动读取 + SecondsToCronSpec 合法值映射 / 非法值回落 + 行为断言可被 cron parser 解析），cron 接线第三个 AddJob 落生产（AddJob + NewChain + FuncJob，cron 日志接入 applogger，账号池经 s.getPool() 字面取单例，
- SC6 终态达成 —— 两条遗留直推路径（单用户编辑 goroutine 直推 + 导入后 `triggerADSyncAfterImport`）及其字段/方法零残留，5 条 grep 静态证伪全部 == 0；导入路径改为同步批量入队 `EnqueueUpdateBatch`（D-01，无 diff）；F-07 入队侧 Pluck 掐根 + F-10 两个删除回滚用例落地，Phase 57 sentinel 零回归。
- 2026-10-19
- 登录路径软删恢复在哈希与写库前增加 NormalizeDN 身份校验，拒绝经 sentinel → 专用原因码 → 固定中文文案传播，且拒绝零写入、DN 仅进 WARN。
- 批量同步在阶段 2 用与登录路径相同的 NormalizeDN 校验过滤软删恢复，被拒项计入 Skipped 并绕开降级管道，双路径判定与 DB 终态由成对用例证伪。
- 全量同步末尾对 AD 端已消失的 sys_ad_user/ou/computer 活行按类别软删除：三类独立双闸门（LDAP 0 条拒绝 / 拟删 >50% 拒绝、恰 50% 放行）+ syncDataInternal 单点挂载（singleflight 闭包内、增量 no-op）+ SyncResult 三计数，12 条 sqlite 可证伪用例全绿、全包零回归。
- 四类 upsert（OU/Group/User/Computer）软删行复活：existing 查询 Unscoped 化 + OnConflict 补 deleted_at + 命中行 DeletedAt 清零，5 条 TestSync_Resurrect 先 RED（UNIQUE 2067）后 GREEN，D-78-05c 两用例翻转，全包零回归。
- syncGroupMembers 的 Delete+Create 包进 GORM `Transaction` 闭包（每组一个事务），以 fixture 表级 CHECK 注入证伪「Create 半途失败永久丢成员」，重放幂等锁定，三调用方零改动

---

## v1.20 网络设备 VLAN + 端口绑定 (Shipped: 2026-07-10)

**Phases completed:** 1 phases, 5 plans, 23 tasks

**Key accomplishments:**

- 1. [Rule 1 - Bug] Corrected MAC format in test expectations
- Extended v1.19 PortWriteService with SetAccessVlan + PortBinding methods, 4 validator sentinels, Extra-map audit carrier, and checkPreState NoOp/skip logic — 31 new subtests, zero v1.19 regression
- Wired 2 new kebab HTTP endpoints (set-access-vlan + port-binding) through v1.19's execSinglePort DRY, refactored execSinglePort to receive bound portID/description (fixes double-BindJSON EOF trap), extended buildAfterValue to read PortResult.Extra for v1.20.1 audit after_value, and added 12 new subtests + 3 permission tests — zero v1.19 regression.
- 1. [Rule 1 - Bug] Fixed PortBindingModal BIND_OPS readonly cast for `tsc -b`
- 1. [Rule 1 - Bug] Fixed Huawei MAC format in fixtures (AABB-CCDD-EEFF → AA-BB-CC-DD-EE-FF)

---

## v1.20.1 网络设备 VLAN + 端口绑定 (Network Device VLAN + Port Binding) — ✅ SHIPPED 2026-07-09

**Phases**: 1 (Phase 56) | **Plans**: 5 (56-01 / 56-02 / 56-03 / 56-04 / 56-05) | **Waves**: 5 (W1 vendor template / W2 service+validators / W3 handler+router+audit / W4 frontend 2 Modals / W5 e2e+UAT+docs)

**Delivered**: 在 v1.19 网络设备写命令基础上新增 2 个端口写操作（`set_access_vlan` 修改 access 端口 PVID + `port_binding` add/remove IP+可选 MAC 静态绑定），3 厂商命令模板全覆盖 + 前端 2 个独立 Modal + 11 e2e 测试 + 文档同步。复用 v1.19 全部基建（权限 / 审计 / 加密 / scrapligo FileTransport / fire-and-forget refresh）。

**Key Accomplishments**:

1. ✅ **Vendor Template Extension (56-01 / W1)** — `internal/services/portcollection/vendor_port_template.go` +6 entries（3 vendors × 2 actions）+ 9 render 函数；MAC 格式锁定（Huawei/H3C = `AA-BB-CC-DD-EE-FF`，Ruijie = `aabb.ccdd.eeff`）；`localNormalizeMACAddress` 本地副本避免 import cycle；dispatch closure by BindOp（add/remove 路由）；12 locked subtests
2. ✅ **Service + 4 Validators + Pre-state (56-02 / W2)** — `port_write_service.go` +2 methods（`SetAccessVlan` + `PortBinding`）+ 4 sentinel validators（VLAN range / bind op / IPv4 regex / MAC normalize+null reject）+ VLAN pre-state NoOp 短路（Pitfall 6: port_binding 永不 NoOp）+ `PortResult.Extra` audit carrier（INFRA-01）；23 表驱动单测
3. ✅ **Handler + Router + Audit (56-03 / W3)** — 2 HTTP POST 端点 + `SetAccessVlanRequest`/`PortBindingRequest` 专用 struct（PATTERNS.md Option A）+ OperType 分流（set_access_vlan=Update / op=add=Create / op=remove=Delete）+ `sys_port_write_audit.after_value` JSON 写入 Extra 字段
4. ✅ **Frontend 2 Modals (56-04 / W4)** — `SetAccessVlanModal`（vlanId InputNumber 1-4094）+ `PortBindingModal`（op Radio.Group + IPV4_REGEX + MAC_REGEX + reason cross-field）+ 2 networkApi wrappers（LANDMINE #5 合规）+ ports/index.tsx ActionButtons 5 → 7 items；vendor-react gzip 774.95 kB（≤ 776 kB baseline，零回归）
5. ✅ **E2E + UAT Deferral + Documentation (56-05 / W5)** — 11 new `TestE2E_*` via scrapligo FileTransport（复用 v1.19 `fileTransportExecutor` 基建，无新基建）+ 10 new fixtures（5 Huawei + 5 Ruijie）+ `56-HUMAN-UAT.md` 12 项 site-visit 推迟 + API响应规范/CHANGELOG/MILESTONES 文档同步

**Comms**:

- 0 critical regression（v1.19 e2e 10 + v1.20.1 e2e 11 = 21 e2e 全绿；operlog 25 OperType + 11 敏感关键词 regression_test.go 不回归；vendor-react gzip 与 v1.19 baseline 零回归）
- 12/12 site-visit UAT 推迟（Huawei S8700 × 6 + Ruijie RS8607E × 4 + H3C × 2 conditional，owner = 现场运维同事）— 详见 `.planning/phases/56-vlan-v1-20-1-0-5-plans-initiated-2026-07-09/56-HUMAN-UAT.md`
- Risk mitigations: H3C `user-bind` no `static` (RISK-01), Ruijie full MAC+IP syntax (RISK-02), Huawei `port link-type access` universal prefix (RISK-03)

**Known deferred items at close**: 12 项 site-visit UAT 推迟（真机 SSH 验证：Huawei S8700 set_access_vlan/port_binding × 6 + Ruijie RS8607E × 4 + H3C × 2 conditional）— 详见 `56-HUMAN-UAT.md`；批量路径（`set_access_vlan` / `port_binding` 多端口）→ FUTURE-BATCH-05 推迟（v1.20.1 单端口 only，`BatchWriteRequest` 已预留 4 字段）

---

## v1.19 网络设备写命令 (Network Device Port Write Operations) — ✅ SHIPPED 2026-07-08

**Phases**: 6 (Phases 50-54 = core MVP; Phase 55 = tech-debt cleanup) | **Plans**: 9 (50-01 / 51-01 / 52-01+02 / 53-01+02 / 54-01 / 55-01+02) | **Waves**: 5 build phases + 1 cleanup phase

**Milestone Audit**: ✅ PASSED (37/37 requirements, 6/6 phases, 20/20 integration wires, 7/7 E2E flows) — see [v1.19-MILESTONE-AUDIT.md](milestones/v1.19-MILESTONE-AUDIT.md)

**Full archive**: [milestones/v1.19-ROADMAP.md](milestones/v1.19-ROADMAP.md) | [milestones/v1.19-REQUIREMENTS.md](milestones/v1.19-REQUIREMENTS.md)

**Delivered**: 补全网络设备"读+写"闭环——Web 端通过 SSH 直接对目标设备下发端口配置命令（启停/描述/dot1x），成功后立即采集一次端口信息回填审计。MVP 范围：3 厂商 × 5 操作 + 批量 + 完整审计 + 权限隔离。

**Key Accomplishments**:

1. ✅ **Vendor × Action 命令模板 (50-01 / W1)** — `internal/services/portcollection` vendorPortTemplate 派发表覆盖 Huawei / H3C / Ruijie 3 厂商 × 5 操作；H3C 复用华为 VRP 模板（D-08 同源），Ruijie 走 Cisco 风格 dot1x；description 操作自动产出 `interface X` + `description Y` 2 cmds；vendorPortTemplate_RenderCommand 15 case 单元测试 + description 长度 ≤80 校验
2. ✅ **PortWriteService + Batch Orchestrator (51-01 / W2)** — `internal/services/portwrite/port_write_service.go` 5 单端口方法 + BatchWritePorts 批量；fail-fast serial loop + detached 30min context（Pitfall #5 兜底 Core.Close 30s）+ per-port Acquire/ReleaseRef 连接池隔离；BatchResult 三数组 succeeded/failed/skipped 编排；Phase 51 mockDeviceExecutor + Phase 54 FileTransport e2e 双层覆盖
3. ✅ **HTTP + Operlog + Permission + Migration (52-01/02 / W3)** — 6 HTTP POST 端点 `POST /network/ports/write/{shutdown,undo-shutdown,description,dot1x-enable,dot1x-disable,batch}`；PortWriteRequest schema + 6 handler；sys_port_write_audit 表真相源 + sys_oper_log 高层摘要（OperType 25 常量映射 CONV-01..04）；权限 `network:port:write` 与 `network:port:query` 隔离；migration_202 加表 + 端口配置菜单 seed + GrantNewMenuToRolesHavingParent helper（antd 父子联动陷阱根除）
4. ✅ **Frontend Drawer + Progress + API Wrappers (53-01/02 / W4)** — types/network.ts + 4 类型 + networkApi.ts + 6 wrapper + port-write/constants.ts 5 const；PortWriteModal 5 操作 × custom reason 审计；BulkWriteDrawer 多端口选择 + indeterminate spinner MVP（BATCH-05 实时进度 v1.19.x deferred）；ports/logs 页面改造；vendor-react gzip 775 kB 基线（前端 bundle size 防回归）
5. ✅ **E2E + Real-Device UAT Deferral + Documentation (54-01 / W5)** — `internal/services/portwrite/port_write_e2e_test.go` 10 个 TestE2E_*（5 single happy + 1 batch happy + 4 类错误路径：device_rejected / transport_error / batch_failfast / noop），使用 scrapligo FileTransport 跑真实 SendConfigs + parseConfigError 链路（Phase 51 mockDeviceExecutor 不调 fn 漏洞闭环）；API/加密/CHANGELOG/README/MILESTONES 文档同步；54-HUMAN-UAT.md 7 项 site-visit deferred（6 项真机 SSH + 1 项 WR-02 观察驱动 Phase 55 决策）
6. ✅ **Tech-Debt Cleanup (55-01/02)** — Phase 53 leftover sweep：WR-02 reason validator signature 修复（antd `(rule, value, form)` 3-param helpers extracted to constants.ts）+ IN-01 `instanceof Error` narrowing + IN-02 mount-only useEffect eslint-disable placement 修正（Lesson: directive 必须紧贴 `}, []);` 而非 useEffect 开头）+ HealthCard.test.tsx empty-state assertion regex match + CR-02 后端 batch_orchestrator.go fallback 路径 port 归属跨层防御（纵深双保险）

**Comms**:

- 98 commits 主仓 (35 code commits + 63 docs/state commits)
- 37/37 requirements addressed (35 fully satisfied + 2 PARTIAL BATCH-05 / UI-03 + 1 adjusted PERM-03 D-07); 54-VALIDATION.md Test Map 锁定
- 0 critical regression（Phase 51 mock + Phase 54 e2e 全绿，operlog 25 OperType + 11 敏感关键词 regression_test.go 不回归，vendor-react gzip 与 Phase 53 baseline 零回归）
- 108 文件 / 25,224 insertions / 3,152 deletions (1.7 days: 2026-07-06 → 2026-07-08)
- Git range: d9dd2edd → 39f5a43f

**Known deferred items at close**: 7 项 site-visit UAT 推迟（6 项真机 SSH transport verification：Huawei/H3C/Ruijie × shutdown/description/dot1x；1 项 WR-02 custom-reason 频率观察驱动 Phase 55 WR-02 修复决策 → ✅ RESOLVED in Phase 55）— 详见 `.planning/phases/54-w5-e2e-real-device-uat-documentation/54-HUMAN-UAT.md`；3 厂商 e2e fixture 全覆盖 / HTTP handler 层 e2e / 跨固件版本命令差异 / Real-device SSH 往返延迟测量 / BATCH-05 实时进度 / sys_port_write_audit 详情查看 UI / 自动回滚 / 多用户并发写互斥 / Cisco IOS 支持 推迟到 v1.19.x+

---

## v1.18 网络设备硬件清单 (Device Component Serials) — ✅ SHIPPED 2026-07-04

**Phases**: 1 (Phase 48) | **Plans**: 3 (48-01 / 48-02 / 48-03) | **Waves**: 3 (schema → collectors → pipeline+operlog+frontend)

**Delivered**: 实现"一机多序列号"——网络设备的板卡/引擎卡、电源、风扇、光模块各自的序列号各起一条 `ops_asset`(`DeviceSN`=组件 SN,`parent_asset_id` 指向主机/`component_type`/`component_slot` 列保留),实时发现时通过 SNMP ENTITY-MIB + Huawei/Ruijie CLI 双路径采集;采集器 UPDATE-only 写 ops_asset,匹配不到的 SN 走 `sys_data_reconciliation`(`conflict_type="F"`,`recon_category="component_serial"`);前端组件清单 Tab 在资产列表 expandable row 内展示。

**Key Accomplishments**:

1. ✅ **Schema 落地 (48-01 / Wave 1)** — migration_201 给 `ops_asset` 加 4 列(parent_asset_id / source_device_id / component_type / component_slot)+ `sys_data_reconciliation.recon_category` varchar(32) 列,partial unique index 切换 `uniq_recon_asset_type_open → uniq_recon_asset_type_cat_open(asset_id, conflict_type, recon_category) WHERE open`,字典 `asset_reconciliation_recon_category` rows=2 seed,migration 全 idempotent + 远程 PG schema introspection 验证全 OK
2. ✅ **组件收集器包 (48-02 / Wave 2)** — 新建 `internal/services/component_collector/` 17 文件:SNMP ENTITY-MIB 单 GET loop(D-08 替代 BulkWalk,因华为/锐捷拒绝)+ Ruijie `temprature*` 噪声过滤(Class=8 && HasPrefix("temprature",13 比例)→ D-11)+ Huawei dual-class SN 非空优先去重 + OwnerResolver stack containtedIn tree 重建 + 32-hop cycle guard + 最小 chassis canonical root(D-12)+ Huawei/Ruijie CLI 收集器 + cmd_dispatcher(D-10 status→transceiver 两步编排)+ 6 TextFSM 模板
3. ✅ **Pipeline + 审计 (48-03 / Wave 3)** — OpsAssetWriter UPDATE-only(D-02/D-03)+ parent-missing degraded mode(D-04 仅 Warnf,从不自动 INSERT 父资产)+ ReconciliationEmitter 双 idempotent(D-06 partial unique + pre-INSERT dedup)+ Pipeline 编排 + operlog.RecordBackground(D-13 cron-path 无 gin.Context,operator="system-cron",沿用 Phase 34 OperType 常量 + 11 sensitive keyword 不回归)+ DeviceInfoCollectionService processTask 在 chassis 更新后调用 collectComponentInfo + runTwoStepTransceiverPipeline(D-10),失败仅 Warnf 不阻塞 chassis(D-14)
4. ✅ **Index + 查询不污染主表** — `asset_service.List/Statistics` 默认 `WHERE component_type IS NULL`(D-07),组件不污染主设备视图;POST `/ops/asset/components` 独立端点带 ops:asset:list group-level middleware 复用,3 handler 测试通过
5. ✅ **前端 ComponentListTab** — Antd Table expandable row + Tag + useEffect primitive deps `[parentAssetId]` + componentApi factory + `Asset` 类型扩展 4 可选 component 字段;无组件时不破坏主表布局
6. ✅ **真机 UAT 推迟声明(为 P0 完成让路)** — 3 项 site-visit(S8700 SNMP 单 GET / RS8607E 627 行噪声过滤 / D-10 S8700 `10GE5/0/4` 两步流水线)按 RESEARCH §Environment Availability 显式推迟到下次现场访问;自动化 UAT 全绿(go build / 21 collector tests / 5 D-10 流水线测试 / 3 handler tests / 5 operlog regression 不回归 / tsc / vite build 1m40s vendor-react 775kB gzip 基线与 Phase 48 前一致 / 远程 PG schema introspection)

**Comms**:

- 6 commits 主仓 + 13 merge 嵌套(`9157be4a` `b8fd2f45`→`6a25596c`),19 commits GitHub-style 紧凑统计上 commit-log 范围
- 14/14 D-id 全覆盖(代码 + 测试证据),48-VERIFICATION.md `score: 13/14` + 3 项 site-visit(human_needed informational, 非失败)
- 0 critical regression(11 pre-existing operations/system 测试失败详见 `deferred-items.md`,在 `b8fd2f45` baseline 跑过确认非 Phase 48 引入)
- 44 文件 / 4,712 insertions(+57 deletions)

**Known deferred items at close**: 3 项 site-visit UAT(S8700/RS8607E 真机)推迟到下次现场访问(per 48-HUMAN-UAT.md status `partial`),其余 11 项 pre-existing test failures 在 `deferred-items.md` 文档化供后续 owner 处理

---

## v1.17 资产对账 (Asset Reconciliation) — ✅ SHIPPED 2026-07-03

**Phases**: 5 (Phases 42-46) | **Plans**: 14 (16 planned, 42-03 skeleton superseded by R3) | **Tasks**: ~70

**Delivered**: 建立"实物层 vs 声明层"对账引擎（R1-R5 完整）+ 4 个根因修复（Phase 47 独立）。Observe-only 策略 + 告警驱动人工修复 + 工单闭环 + IP 段例外 + 半自动修复建议（高置信度 6 状态机）。

**Key Accomplishments**:

1. ✅ **R1 观测底座** — 物化视图 + 主表 + Dashboard + 6 个 Statistics 端点（防 MaxPageSize 钳制）+ operlog 全覆盖
2. ✅ **R2 告警 + 工单闭环** — critical/high 自动转工单 + WebSocket 推送 + SysNotice + 7d 静默期 + 24h 节流
3. ✅ **R3 IP 段例外** — CIDR 格式 + 5 种 actions 组合 + expires_at + 命中测试工具 + Excel 导入导出 + VERIFICATION 10/10
4. ✅ **R4 工位详情整合** — 健康度卡片（5 KPI + 趋势 mini chart）+ 对账健康列 + ReconciliationDrawer 3 Tab
5. ✅ **R5 半自动修复** — 6 状态机 + 7d 回滚 + 误修复率告警三通道（SysNotice+WS+operlog）+ UAT 9/10 PASS
6. ✅ **Phase 47 根因修复** — infoPoint drift / Layer3 UPSERT / port_status 漂移 / MAC parser 校验（独立完成）

**Known in-session fixes during UAT**:

- Apply 400: `resolution_method` → `resolution_note` 列不存在 bug (commit 87d5fc82)
- Rollback 400: 删除 `pre_fix_user_id == nil` 过度防御检查
- Cron 5 fields 缺年份: UPDATE sys_job 加 `0` 前缀

**Known deferred items at close**: 0 (Test 10 权限非 admin 端到端 skipped due to 缺测试账号，不阻塞)

---

## v1.3 技术债清理 — ✅ SHIPPED 2026-04-27

**Phases**: 3 (Phases 8-10) | **Plans**: 9 | **Tasks**: ~50

**Delivered**: 修复 SNMP panic 崩溃问题，清理后端死代码并修复安全漏洞，实现网络设备批量导出功能

**Key Accomplishments**:

1. ✅ SNMP panic 完全修复 — panic 恢复包装器 + RWMutex 并发安全 + WaitForReady 连接验证
2. ✅ 后端代码质量提升 — 删除 Core 死字段，文档化 12 个外部依赖字段
3. ✅ 安全修复完成 — WebSocket CheckOrigin 增强，并发安全，错误日志改进，15 个新测试全部通过
4. ✅ 批量导出功能 — 支持一次性选择多个实体类型并打包为 ZIP 下载，集成到 9 个网络管理页面

**Known deferred items at close**: 0 (all resolved)

---

## v1.2 可配置仪表盘生产级改造（仿 Zabbix）— ✅ SHIPPED 2026-04-21

**Phases**: 4 (Phases 4-7) | **Plans**: 11 | **Tasks**: ~60

**Delivered**: 实现可配置仪表盘系统的核心功能，包括 Widget 数据获取机制、前端交互完善、实时数据刷新和用户体验优化

**Key Accomplishments**:

1. ✅ Widget 数据获取机制 — `GetWidgetData` 和 `GetBatchWidgetData` 支持 API/WebSocket/Static 数据源
2. ✅ 前端交互完善 — Widget 选择器、仪表盘设置面板、模板预览功能
3. ✅ 实时数据刷新 — WebSocket 连接管理、基于刷新间隔的轮询、缓存策略
4. ✅ 用户体验优化 — Widget 拖拽布局优化、加载状态和错误处理、响应式布局适配

---

## v1.1 信息点导入设备端口关联 — ✅ SHIPPED 2026-04-16

**Phases**: 1 (Phase 3) | **Plans**: 1 | **Tasks**: ~5

**Delivered**: 信息点 Excel 导入时支持通过设备名/端口名自动匹配并关联设备 ID 和端口 ID

**Key Accomplishments**:

1. ✅ 信息点导入配置添加"所属设备"列 (`deviceName` → `device_id`)
2. ✅ 信息点导入配置添加"所属端口"列 (`portName` → `port_id`)
3. ✅ 两个新列均为可选，空值不阻断导入
4. ✅ 模板下载包含设备和端口示例值

---

## v1.0 工位导入部门/用户关联 — ✅ SHIPPED 2026-04-16

**Phases**: 2 (Phases 1-2) | **Plans**: 7 | **Tasks**: ~30

**Delivered**: 工位 Excel 导入时支持通过部门名称/用户名自动匹配并关联部门 ID 和用户 ID

**Key Accomplishments**:

1. ✅ Excel 导入模板添加"所属部门"和"所属用户"两个可选列
2. ✅ 导入时通过部门名称/用户名匹配，写入对应 ID
3. ✅ 匹配失败留空不阻断导入
4. ✅ 前端工位列表/详情页面展示关联的部门和用户信息
