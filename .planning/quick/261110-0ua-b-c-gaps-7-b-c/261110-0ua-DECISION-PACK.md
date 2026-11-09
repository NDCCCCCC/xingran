---
quick_id: 261110-0ua
type: decision-pack
status: awaiting_user_decision
created: 2026-11-10
scope: v1.55 收档挂账 B 类（7 项需产品/架构拍板）+ C 类（人工/QA/DBA 执行项）
method: 2 个只读 subagent 全仓代码盘点 + Lead 亲自复核 C 类载体；零代码改动、零 git 写操作
---

# B/C 类 gaps 决策包（quick 261110-0ua）

> **本文件不是完成报告，是待拍板清单。** B 类每项都卡在「产品/架构决策」而非编码；
> C 类受「AI 不擅自写生产数据 / 不伪造闭环」硬约束，AI 只剩「组织」职责（已完成）。
> **不修改 79 条跟踪表任何勾选、不代跑任何生产/真机验证、不代 QA 裁定窗口与验收人。**

## 0. 总览与实证纠偏（重要）

| 类 | 项数 | 性质 | 本轮 AI 能做的边界 |
|---|---|---|---|
| B | 7 | 需产品/架构拍板才能写代码 | 产出带 `文件:行号` 证据的决策简报（本文件）+ 拍板后按 GSD 立项执行 |
| C | 5 组 | 需人工/QA/DBA 执行或裁定 | 复核交付物完整性 + 补「可直接执行」脚手架（不代执行、不代勾选） |

**盘点暴露的两处收档口径需修正（非新增缺口，是纠正描述）**：
1. **B-1「无已读设施」不成立**：项目已有完整 `sys_notice` / `sys_notice_read` 已读体系
   （`internal/api/v1/system/notice_user_handler.go:37-41,140-270`：按用户隔离 / 未读数 / 全部已读 /
   详情自动已读 / operlog 记录），且漂移告警 emit 已写 `sys_notice`、payload 带 `noticeId`
   （`internal/services/addomain/ad_sync_drift_alert_service.go:450-468,512-524`）。真缺口 = 「把已读能力
   接到 ad-drift-alert 页面 + 补一个标记端点」，**不是**给 append-only 的 `sys_oper_log` 加列。
2. **B-4「无自动回滚」应限定为「批量路径 + 物理参数」**：走 `writeSinglePort` 的单端口 critical 动作
   （shutdown / set_access_vlan / trunk / portsec static_bind）**已有** PORTW-04 快照+逆序回滚+
   rollback_failed 告警（`internal/services/portwrite/safety.go:17-23,196-328`、`port_write_service.go:483-522`）。
   批量路径对所有 action 都无回滚（不只物理，`batch_orchestrator.go:197/222` 直接 executeWrite）。
   **另发现真实误导**：警示文案说「按 `audit.before_value` 恢复」，但 `buildBeforeValue`
   （`internal/api/v1/network/port_write_handler.go:486-509`）只含 admin_status/dot1x/description/
   interface_name/vlan，**不含 speed/duplex/mtu 原值** → 人工按该字段实际恢复不了。

**另有一项实为真实代码缺陷（B-7）**：`room_photo` / `location_alias` 两 handler 未透传
`ErrScopeDenied` 的 403：room_photo 越界恒 500（`room_photo_handler.go:81,141,170,218,238,264` 全用
`response.ErrServerError`）、location_alias 越界恒 400（`location_alias_handler.go:93,110,130,146`，
`case int` quirk 让名义 500 实落 400）。而 `149-UAT-RECHECK.md:43` 已写「单条越界 → 显式 HTTP 403」，
walls/doors/texts 与 asset_handler 均用 `HandleServiceError`/`errors.As` 正确透传 403。
**这是 service 已修、HTTP 层漏接的契约不符**，不需要产品拍板，只需批准「是否算破坏性变更」。

---

## 1. B 类逐项决策简报

### B-1 告警「标记已读」
- **证据**：后端 5 端点无 read（`ad_drift_alert_router.go:29-49`、`ad_drift_alert_handler.go:50-119`）；
  logs 来自 `sys_oper_log` 投影（`ad_sync_drift_alert_query.go:116-151`）无已读字段；`sys_oper_log`
  （`internal/models/log.go:14-33`）append-only 无 read 列；前端 `TriggerLogTable.tsx` 无已读列。
- **附带既存契约漂移（建议一并拍板）**：前端 `ADDriftTriggerLog` 期望 createdAt/drifted/memberCount，
  后端返回 id/title/method/operTime/jsonResult/status；前端传 startTime/endTime/status 筛选后端未实现；
  前端 Config 有 `webhookTemplate` 后端是 9 个 sys_config 旋钮。
- **需拍板**：① 语义 = (a) 按用户维度的通知已读〔复用 `sys_notice_read`〕／(b) 告警「已处置/已确认」
  〔与用户无关，需 handled_by/handled_at〕／(c) 给 `sys_oper_log` 加 is_read 列〔污染共享审计表，不推荐〕；
  ② 是否要「全部已读 / 未读数徽标 / 按未读筛选」；③ 上述契约漂移是补后端还是删前端。
- **推荐**：**(a) 复用 `sys_notice_read`**（新增按 noticeId 标记的薄端点 + 前端已读列/按钮/未读筛选，
  零新表、零新权限码、零 migration，约 4-6 文件）。若产品要「处置确认」语义则必须走 (b) 新表
  `sys_ad_drift_alert_read`（约 8-12 文件 + migration 239+ + 第 5 个 F 型权限码）。
- **执行方式**：拍板后走 `/gsd-quick`（a 方案）或 `/gsd-new-milestone`（b 方案）。

### B-2 企微/钉钉专用 markdown + 加签（路线 B）
- **证据**：`api_sender_service.go` 的 `buildRequestBody`/`buildFromTemplate`（:226-277）为通用扁平 JSON、
  纯字符串替换无 JSON 转义；`setAuthentication`（:304-348）仅 none/basic/bearer/apikey，无 HMAC；
  `sys_api_notification_config`（`internal/models/notification_config.go:90-116`）无 platform/sign_secret/
  msgtype 列；全仓 Go/前端源码对 企微/钉钉/wecom/dingtalk **0 命中**。
- **需拍板**：① 平台范围（企微 / 钉钉 / 飞书 / 可扩展 provider）；② 配置落点（增列 / 塞
  AuthConfig-Headers〔有「读端点遮蔽 + Update 覆盖真值」坑，不推荐〕/ 新表）；③ 消息体渲染责任
  （整段 JSON 模板需结构化替换+转义 vs 只填 markdown text 字段由代码包 msgtype 外壳）；④ 加签时机
  （每次重发重新计算 timestamp/sign，不能一次算好复用）；⑤ `SendWithDefaultConfig` 同类型多平台如何区分。
- **推荐**：**增列 + provider 化**（`sys_api_notification_config` 增 Platform/SignSecret/SignMode/MsgType，
  Platform 为空时路线 A 行为零回归；`APISenderService` 内按 Platform 分派 signer 与 body builder；
  每次重发重算加签）。约 10-15 文件 + 1 个双方言幂等 migration + HMAC golden vector 单测 + 前端表单。
- **执行方式**：拍板后 `/gsd-new-milestone`（含 migration 与前端表单，超出 quick 体量）。

### B-3 `sys_object_id` 已采集未落库
- **证据**：`internal/device/snmp_client.go:328,347-364` 确实采到 OID，但 `ScanDevice`（:784-814）
  丢进 `ScanResult` 时未透传；`device_discovery_service.go:341` 主链路连 OID 都没 Get；
  `NetworkDevice`（`internal/models/network_device.go:34-57`）无列；`physical_capability.go:6-8,118-134`
  型号 key 仅 `NetworkDevice.Model` 子串匹配。全仓 `sys_object_id` 字面量 0 命中。
- **需拍板**：① 落哪张表（`sys_network_device` / 同时 `sys_device_discovery_result`）、存量回填策略；
  ② 采集链补齐方式（修 ScanDevice 透传 vs snmpProbe 增一次 Get）；③ **核心**：匹配键 = Model 子串
  （现状）／「SysObjectID 精确/前缀优先 + Model 兜底」／只用 OID；④ 规则是否仍坚持「纯 Go 常量、
  不落库」（`physical_capability.go:6`）；⑤ OID 命中能否写进 Evidence（不能，会破坏「Evidence 空串=
  未取证」诚实性口径）。
- **推荐**：**「OID 只作为设备属性落 `sys_network_device` 一列 + 匹配键改『OID 精确/前缀优先、
  Model 子串兜底』+ 规则继续留 Go 常量」**。落库成本极低，能把模糊 Model 升级为厂商私有 OID 精确识别，
  且 Model 兜底零回归覆盖存量设备。**不推荐**「只用 OID」（存量全失配）或「OID 规则落库」（扩成配置平台）。
  主要成本在匹配签名扩散（~15-25 文件 + migration + 测试矩阵），非落库本身。
- **执行方式**：拍板后 `/gsd-new-milestone`。

### B-4 批量物理参数无自动回滚
- **证据**：`batch_orchestrator.go:63-269` 直接 executeWrite、不走 `writeSinglePort`（无快照）；
  `dry_run.go:320-324` + 前端 `constants.ts:109-111` / `BulkWriteDrawer.tsx:1220` 已显式警示；
  但 `buildBeforeValue`（`port_write_handler.go:486-509`）不含物理原值（见 §0 纠偏）。
- **需拍板**：① 范围（只补物理批量 / 把**所有批量写**纳入 PORTW-04〔会改变 shutdown/VLAN/trunk
  批量语义，大型架构变更〕）；② 物理逆命令可得性（`undo speed`/`speed auto` 等三家语法不同，
  MTU 默认值厂商不同，speedMode=auto 无单一逆命令）；③ 原值从哪来（扩 before_state/audit 结构
  vs 下发前读 running-config 解析）；④ 失败判定（fail-fast 后回滚整批已成功 / 只回滚失败端口 /
  回滚失败前成功端口）、回滚自身失败时状态；⑤ 回滚是否算写操作、是否单独 operlog/审批。
- **推荐**：**维持现状不新增自动回滚，但补两处诚实性/可恢复性短板**：① 把 `audit.before_value`
  扩成真正含物理原值（或把警示文案改为「按 `audit.command_sent` + 设备 running-config / 采集快照
  人工恢复」），消除误导；② 若确要自动回滚，先限定「单端口 + 有真机取证逆命令的厂商子集」，
  批量明确留「不纳入」。理由：项目已有完整 PORTW-04 框架，真缺口是「before_state 无原值 + 无逆命令模板
  + 批量不走 writeSinglePort」；但「以为回滚成功实则未恢复」比显式警示人工恢复更危险，且
  `physical_capability.go` 的「本里程碑不纳入 critical 自动回滚」是经裁定的诚实取舍。
  **若产品坚持批量自动回滚 → 必须单独立项 + 真机逐厂商逆命令取证 + 失败注入测试（20-30 文件）。**
- **执行方式**：选「补诚实性短板」可走 `/gsd-quick`；选「批量自动回滚」走 `/gsd-new-milestone`。

### B-5 `RequiresInterfaceView` 与型号画像 `RequiresView` 联动
- **证据**：`RequiresInterfaceView`（`device_capability.go:209-215`，model 写死空串）只在
  `port_write_service.go:632-636` 决定要不要追加 `exit`；`RequiresView`（`physical_capability.go:54,92`）
  只在 `physical_template.go:105-109` 决定要不要加 interface 前缀。**两者完全脱钩**；当前所有画像
  `RequiresView=true`、三家默认/基线均未覆盖 → 差异不触发，属「框架预留、未联动」，**当前无实际 bug**。
- **需拍板**：① 是否现在就统一（建议否，等真有 `RequiresView=false` 的型号画像再动）；② 统一方向
  （扩 `ActionSystemView` 成型号级可表达 vs 让 `physicalProfile.RequiresView` 成为唯一真相源并反哺执行侧）；
  ③ 是否允许改 `portwrite` 包（跨 148/149/150 文件面归属）。
- **推荐**：**保持现状 + 登记为「框架预留、待真实型号触发时再联动」**，不动代码。真要统一需改 6 处 +
  4 组测试（见 subagent 报告），当前零收益且触碰跨 phase 文件面。
- **执行方式**：无需立项，仅在 STATE 把 ⑦ 从「后续候选」改注为「框架预留、无实际触发、不修」。

### B-6 forbidigo 能力边界
- **证据**：`.golangci.yml:29-49` 8 条 forbid pattern；`148-SUMMARY.md:76-82` 已实测 golangci-lint
  2.12.2 只匹配 AST 标识符/选择器调用名，不命中字符串字面量/注释里的 SQL token；真实门禁由
  `internal/services/operations/excel_export_dialect_148_test.go` 的 Go 源码静态断言承担，
  但 **`dialect148SourceFiles`（:31-45）只扫 3 个改造文件**，而全仓 136 个 Go 文件命中方言、
  靠 `.golangci.yml:117-180` 白名单豁免 → 往白名单目录新增方言当前无任何门禁拦截。
- **需拍板**：① 是否接受现状「双层门禁」（标识符准入 + 3 文件静态断言，148 已如实登记、零新依赖）；
  ② 若补强，扫描范围钉死 3 文件还是扩到全仓/非白名单目录；③ 技术路线（扩展现有 Go 静态断言
  〔推荐、零新依赖〕/ 自定义 golangci-lint 插件 / 独立 SQL 字符串扫描工具〔新依赖〕）；
  ④ 白名单收敛归哪个 phase；⑤ 是否给 CI 加 forbidigo 专门步骤。
- **推荐**：**接受现状 + 把静态断言扫描范围从 3 文件扩到「所有非白名单 Go 文件」**（纯测试改动、
  零新依赖、随 `go test ./internal/...` 跑），并给 CI 补一条 `golangci-lint` 显式步骤的可观测输出。
  白名单收敛另立后续 phase，不本期做。
- **执行方式**：拍板后 `/gsd-quick`（扩展断言 + CI 注释）。

### B-7 两 handler HTTP 透传口径 ★ 实为真实代码缺陷
- **证据**：service 层 149 后已统一 `ErrScopeDenied`（`scope_write.go:38-48`，`room_photo_service.go:55-63`、
  `location_alias_service.go:164-165,204-221,265-266`）；但 HTTP 层 room_photo 8 个错误分支全用
  `response.ErrServerError`(500)、location_alias 用固定 400/名义 500 实 400，均**未透传 403**。
  walls/doors/texts（`wall_handler.go:48,74,124...`）与 `asset_handler.go:16-31` 已正确用
  `HandleServiceError`/`errors.As` 透传 403。`149-UAT-RECHECK.md:43` 明确写「单条越界 → 显式 HTTP 403」
  → **代码与验收口径不符**。
- **需拍板**：仅一项 —— **是否批准这次「破坏性变更」**（同一越界场景 HTTP 码 500/400 → 403，
  前端 `api.ts` 错误分支、UAT 断言、按码重试逻辑需核对）。方向上与 149 登记口径和其余端点对齐，
  且 `ErrScopeDenied` 本身已是 403。
- **推荐**：**批准修复**（仿 `asset_handler.go:24-31` 加 `errors.As` 类型分派 helper，或统一换
  `response.HandleServiceError`；service 层/scope 基建/149 T4 矩阵零改动；补契约测试参考
  `location_alias_room_device_handler_test.go`）。这是本决策包里**唯一不需要产品语义决策、
  只需批准破坏性变更**的项。
- **执行方式**：批准后 `/gsd-quick`，约 2-3 文件 + 契约测试。

---

## 2. C 类：交付物现状与「可直接执行」脚手架缺口

**硬约束复核（Lead 亲自读 + subagent 复核）**：`261108-human-verification-tracker.md`（243 行 / 79 条
逐行「条目 ID | 来源 REQ | 验证内容 | 前置授权/环境 | 执行人 | 证据位 | 回写状态」）、
`261108-uat-writeback-policy.md`（173 行 / 含证据落盘模板 §3.2 与「未执行不得勾选」最高纪律）、
`149-UAT-RECHECK.md`（77 行 / D1/D2 作废 + 6 条重跑要点）、`150-UAT-BACKFILL.md`（58 行 / 作废清单 +
All 档零回归复核）、CI `PG contract skip observability`（`ci.yml:67-78`，`-v` 把 skip 原因写进日志）
—— **全部完整、0 勾选、0 伪造完成、口径自洽**。AI 的「组织」职责已完成。

**剩下的全是人工/QA/DBA**：生产 PG 只读、87-6 写操作授权、生产部署窗口、现场真机（Huawei S8700 /
Ruijie / H3C / Cisco）、DBA migration_180 回传、`TEST_PG_DSN` 真实双跑、149/150 重跑窗口与验收人裁定。
AI 不得代跑、不得代勾、不得代签。

**可补的「让人工直接执行」脚手架（均不触碰勾选、不代执行）**：

| # | 缺口 | 建议补什么 | 落点 |
|---|---|---|---|
| C-a | 79 条只有自然语言「验证内容」，无逐条命令/SQL/curl/浏览器步骤 | 为高频组（VAL-01 2 项 / VAL-02 7 项 / VAL-06 1 项）补「可执行命令清单 + pass/fail 判据」 | `.planning/notes/` 新建执行手册（引用跟踪表条目 ID，不回填勾选） |
| C-b | 149/150 UAT 载体的「重跑要点」不是判据表、未与 79 条跟踪表做行级回写映射 | 把 §3 重跑要点 / §5 复核项改造成「条目 ID → 验收判据 → 证据位 → 回写映射」表；引用 `VAL-05-133-D1/D2`、`VAL-05-133-A6` | 新建登记件（不修改现有 UAT 载体原件、不改归档件） |
| C-c | 回写口径有通用证据模板（§3.2）但无针对具体条目 ID 的实例 | 为 149 D1/D2、150 A6、VAL-01-84-1 各生成 1 份空白模板实例（字段齐全、值留空） | 同上 |
| C-d | `audit-uat` / `audit-open` 衔接只有语义、无可直跑命令 | 补「执行人回写后由 Lead 跑的具体命令清单」（参数按届时 GSD 版本填，不硬编码不存在参数） | 回写口径附录 |
| C-e | 无具名验收人 / 窗口日期 / 授权单号 | **必须由 QA / 发布负责人 / DBA 提供**，AI 不代填 | 保持留白，列出「所需输入清单」交用户 |

**推荐**：做 C-a ~ C-d 的纯文档脚手架（零代码、零勾选、零生产数据），C-e 明确列为「需你/QA 提供」。
这能让 79 条从「索引表」升级为「可直接执行的验收手册」，但仍**不改变任何一条的待执行状态**。

---

## 3. 请你拍板（按优先级）

| # | 决策 | 选项 |
|---|---|---|
| 1 | **B-7（真实缺陷）** | 批准修复 403 透传（推荐）／保持挂账 |
| 2 | **B 类其余 6 项** | 逐项选「按推荐立项」／「本次不做、保持挂账」／「给替代方向」；其中 B-5 推荐直接改注为「框架预留、不修」 |
| 3 | **C 类脚手架** | 批准补 C-a~C-d 纯文档（推荐）／只保持现状 |
| 4 | **C-e 输入** | 是否现在提供具名验收人 / 重跑窗口 / 授权单号（不提供则保持留白） |

> 拍板后：B 类按所选走 `/gsd-quick`（小）或 `/gsd-new-milestone`（含 migration/前端/多文件）；
> C 类脚手架按 `/gsd-quick` 落纯文档；**任何一步都不代执行、不代勾选、不写生产数据**。
