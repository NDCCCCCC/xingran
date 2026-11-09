---
phase: 143
plan: 01
wave: 1
depends_on: []
files_modified:
  - internal/services/portcollection/physical_capability.go (new)
  - internal/services/portcollection/physical_capability_test.go (new)
  - internal/services/portcollection/device_capability.go
  - internal/services/portcollection/physical_template.go
  - internal/services/portcollection/physical_template_test.go
  - internal/services/portcollection/vendor_port_template.go
  - internal/services/portwrite/physical.go
  - internal/services/portwrite/physical_test.go
  - internal/services/portwrite/port_write_service.go
  - internal/services/portwrite/batch_orchestrator.go
  - internal/services/portwrite/physical_batch_test.go (new)
  - internal/services/portwrite/dry_run.go
  - internal/api/v1/network/port_write_handler.go
  - internal/api/v1/network/port_write_physical_handler_test.go
  - internal/api/v1/network/port_write_physical_batch_handler_test.go (new)
  - xingran-react-frontend/src/types/network.ts
  - xingran-react-frontend/src/components/network/port-write/constants.ts
  - xingran-react-frontend/src/components/network/port-write/__tests__/constants.test.ts
  - xingran-react-frontend/src/components/network/port-write/PortWriteModal.tsx
  - xingran-react-frontend/src/components/network/port-write/BulkWriteDrawer.tsx
  - xingran-react-frontend/src/lib/api/networkApi.ts
  - xingran-react-frontend/src/lib/api/__tests__/networkApi.test.ts
  - xingran-react-frontend/src/components/network/port-write/__tests__/BulkWriteDrawer.test.tsx
  - xingran-react-frontend/src/components/network/port-write/__tests__/PortWriteModal.test.tsx
autonomous: true
commit: false   # Lead 统一提交，禁止本 executor git add/commit
---

# Phase 143 · 端口写物理参数收口（PORT-01/02）

## 必读（按顺序）
1. `.planning/notes/261108-v154-architecture-decisions.md` **§三**（Lead 已拍板的全部设计裁决，权威）
2. `.planning/ROADMAP.md` Phase 143 段（成功标准 5 条）
3. `.planning/REQUIREMENTS.md` PORT-01/02
4. `AGENTS.md` / `CLAUDE.md` 项目硬约定

## 目标
补齐 v1.53 FIX-04 / FIX-05 自报边界：① 型号级 `vendorCapabilities` 规则（可代码化框架）② 批量物理参数路径接入 ③ 前端单条/批量入口。

## 已确认事实（不要重新调研）
- FIX-04 已交付：`internal/services/portcollection/physical_template.go`（`set_speed`/`set_duplex`/`set_mtu` × huawei/h3c/ruijie 模板，`SpeedAllowed`/`DuplexAllowed`/`MTUMin=1280`/`MTUMax=9216`，`renderPhysical`）、`internal/services/portwrite/physical.go`（`validatePhysicalRequest` + `PhysicalWrite`，成功后写 `result.Extra["speed"|"duplex"|"mtu"]`）、`internal/api/v1/network/port_write_physical_handler.go` + `port_write_router.go:76` `write.POST("/physical", ...)`、dry-run 预览、`buildAfterValue`（`port_write_handler.go:564-592` 从 `Extra` 读）、4 组测试。**后端单条链路完整**。
- 批量 `batch_orchestrator.go`：`BatchWriteRequest` 无 4 个物理字段（:84-97）；入口 validator switch（:88-103）只认 `set_access_vlan`/`port_binding`；循环两处 `s.executeWrite(..., nil, nil)`（:164/:187）丢弃 `renderExtras`；不调 `CheckActionSupportParams`；`PortResult.Extra` 物理回显只写在单条 `PhysicalWrite`（`physical.go:92-109`）⇒ 批量 `audit.after_value` 恒 `{}`；handler `BatchWrite` sentinel switch（:365-397）缺 `physicalSentinelMessage` / `ErrActionNotSupported` 400 分支；`SkipPreflight` 字段存在但从未消费。
- 能力矩阵 `internal/services/portcollection/device_capability.go`（362 行）：`capabilityEntry{baseline map[PortAction]ActionSupport, modelRules []modelCapabilityRule, vendorUnsupported map[PortAction]string}`；`modelCapabilityRule{matchModelSubstr, unsupported, supported}`；`lookupSupport` 优先级 = 型号 unsupported > 型号 supported > 厂商 unsupported > 基线 > 默认 Supported。**三个物理动作零条目 → 恒默认 `ActionSupported`**；`physical_template.go:17-20` 注释自述「命令未针对三家真机逐一验证」。
- `PortTemplateParams`（`vendor_port_template.go:65-73`）已有 Speed/SpeedMode/Duplex/MTU 四字段；`RenderCommand(vendor, action, params)`（:225）先查 `SupportEvidence` 厂商级 Unsupported 再查派发表；`vendorPortTemplate` 派发表仅注册 huawei/h3c/ruijie 三家的物理动作，Cisco/Maipu 未注册也未标 `vendorUnsupported`。
- 型号来源已接通：`writeSinglePort`（`port_write_service.go:438-449`）查 `models.NetworkDevice` 取 `dev.Vendor, dev.Model` 传入 `CheckActionSupportParams`；`dry_run.go:239-249` 同；`PreviewResult.Model`（:297）已回显前端（BulkWriteDrawer:1093 显示 `{vendor} {model}`）。`NetworkDevice.Model string`（`network_device.go:40`）。`sys_object_id` 已采集但**未落库**。
- 进度事件 `batch_progress.go` 的 `newProgressPublisher(ctx, operator, deviceID, action, total)` / `emit` / `emitCompleted` 与 `PortWriteProgressMessage.Action string` **与 action 无关，零改动可复用**；前端 `BulkWriteDrawer` 的 `LiveProgress` / WS 订阅 / `audit/list` 断线对账已完备。
- 前端：`src/types/network.ts:280-295` `PortWriteAction` 15 动作不含物理三动作、`:332-364` `BatchWriteRequest` 无 4 字段；`constants.ts:42-75` `ACTION_TITLE` 恰好 15 键且 `__tests__/constants.test.ts:57` 断言 "exactly 15 keys"；`networkApi.ts` 只有 `writeTrunk`（:350）无 `writePhysical`；`BulkWriteDrawer.tsx` `ACTION_OPTIONS`（:84-100）13 项、`buildRequest`（:299-337）不组装物理参数；`PortWriteModal.tsx:165-204` action 分支不含物理。
- `pkg/permission/config.go` 的 route-permission registry 已登记 `/trunk`、`/portsec`，**未登记 `/network/ports/write/physical`**。
- 能力矩阵测试手法：`device_capability_matrix_test.go:138-175` 用「备份 `vendorCapabilities[VendorX]` → 替换成自定义 `capabilityEntry` → `defer` 恢复」表驱动，不依赖真机。
- fixture：`internal/services/portwrite/testdata/*.fixture` 是纯文本 scrapli 会话回显（FIX-04 只加了 `huawei_set_speed_success.fixture`）；`physical_e2e_test.go` 在业务包内用 `FileTransport` 回放。

## Lead 裁决（必须遵守，不得自行改设计）
| 项 | 裁决 |
|---|---|
| 矩阵载体 | **代码常量 + 测试锁定**：新增 `portcollection/physical_capability.go`（`physicalProfile{AllowedSpeeds []int, AllowedDuplexes []string, MTUMin, MTUMax int, SpeedCommand, DuplexCommand, MTUCommand string, RequiresView bool, Evidence string}` + `capabilityEntry` 扩 `physical *physicalProfile` / `modelPhysical []modelPhysicalRule` 字段 + `ResolvePhysicalProfile(vendor, model) physicalProfile` + `CheckPhysicalSupport(vendor, model, action, params) (physicalProfile, error)`） |
| 型号 key | **只用 `NetworkDevice.Model`**，子串匹配复用现有 `matchModelRules` 语义；**不做** `SysObjectID` 落库 + migration（零 DDL，避免与 Phase 144 的 migration 238 冲突）；OID 仅采集未持久化登记为 Known Gap |
| 回退链 | 型号 `modelRules.unsupported` 命中 → 拒绝（400，Evidence 必含依据）→ 型号 `modelPhysicalRule` 命中 → 用该型号 profile → 厂商 `vendorUnsupported` 命中 → 拒绝 → 厂商 `entry.physical` 基线 → 用厂商 profile（Evidence 标「厂商基线，未逐型号取证」）→ 全局默认（FIX-04 现状 `{10,100,1000,10000}` / `half|full|auto` / `[1280,9216]` / `"mtu %d"`，`Evidence=""`） |
| 默认放行 ≠ 有证据 | 新增 `Evidence` 空值语义 + `PreviewResult` 回显 `evidenceLevel: "unverified"` + 审计 `Extra` 留 `physicalEvidence:""`；**不阻断写操作**、不冒充真机结论；不新增 `ActionUnverified` 枚举值（避免改动 `ActionSupport` 枚举与既有测试） |
| 校验/渲染同源 | `PortTemplateParams` 加 `Model string`；`RenderCommand` 把型号传给 `renderPhysical`，按 `ResolvePhysicalProfile` 取命令形态与范围；`validatePhysicalRequest` 改接 `(vendor, model, action, params)`，与 `CheckPhysicalSupport` 同源，保证「校验放行 ⇔ 渲染接受」 |
| Cisco / 迈普 | **显式标 `vendorUnsupported`**（理由「渲染层未实现 + 无真机取证依据」），不补模板；同时把物理动作加进前端 `CISCO_UNSUPPORTED_ACTIONS` |
| 批量接入 | **允许**：`BatchWriteRequest` 加 4 字段（json tag 与单条 DTO 同名）；入口 switch 加物理三动作 `validatePhysicalRequest` + `CheckPhysicalSupport`（循环内查一次 `dev` 后判定）；循环两处 `executeWrite` 传 `&renderExtras{physical: &PhysicalParams{...}}`；抽公共函数 `applyPhysicalAuditExtra(action, params, result)`（把 `physical.go:92-109` 搬出）修 `audit.after_value`；handler sentinel 补 `physicalSentinelMessage` 与 `ErrActionNotSupported` 两个 400 分支（放在 `default: 500` 之前）；`SkipPreflight` 显式登记「批量不支持跳过预检」 |
| 批量回滚 | **不纳入 critical 自动回滚**（沿用物理动作现有非 critical 语义）；在 dry-run 预览与审计显式警示「批量物理参数无自动回滚，失败需人工按 `audit.before_value` 恢复」；登记 Known Gap |
| 进度/错误聚合 | **零改动复用** `BatchResult{Succeeded,Failed,Skipped}` + `createAuditRowsInTx` + `operlog.Record(OperTypeBatch)` + `progress.emit/emitCompleted` |
| 前端 | 补 `networkApi.writePhysical()` wrapped `post<PortResult>`（不手动 `message.error`）；`PortWriteAction` 15→18、`ACTION_TITLE` 15→18 + 物理选项常量、测试断言同步 18；`BulkWriteDrawer` `ACTION_OPTIONS` 加 3 项 + `buildRequest` 组装 4 字段 + 条件表单 + 预览 `evidenceLevel` 徽标；`PortWriteModal` 加物理分支（条件表单，`canWrite` 门控不变） |
| 权限注册表 | **本 phase 不改 `pkg/permission/config.go`**（该文件已划归 Phase 144 独占）；把「补登 `/network/ports/write/physical`」写入 SUMMARY 的移交项，由 Lead 在集成时安排 Phase 144 后端一起补 |
| 真机取证 / OID / e2e 归位 | **本里程碑不做**，写进 SUMMARY Known Gaps，不伪造闭环 |

## 任务（XML）
<task id="1">
<title>型号级物理能力矩阵框架</title>
<files>internal/services/portcollection/physical_capability.go (new), internal/services/portcollection/device_capability.go, internal/services/portcollection/vendor_port_template.go, internal/services/portcollection/physical_template.go</files>
<action>
按裁决新增 `physicalProfile` / `modelPhysicalRule` / `ResolvePhysicalProfile` / `CheckPhysicalSupport`；扩 `capabilityEntry`（新增字段，不改既有字段语义）；`vendorPortTemplate`/`RenderCommand` 加 `Model` 透传；`renderPhysical` 按 profile 的 `SpeedCommand`/`DuplexCommand`/`MTUCommand` 分叉（保持三家默认命令体不变，除非命中型号规则）；把物理三动作在 Cisco/Maipu 显式标 `vendorUnsupported`。
**关键**：① 不破坏既有 `CheckActionSupport*` / `lookupSupport` 的签名与行为（既有 14 个 portwrite 测试 + `device_capability_matrix_test.go` 必须全绿）；② 全局默认 profile 与 FIX-04 现状等价（三家交集），保证单条物理写零回归；③ 新代码只用 Go 常量，不引入配置/DB 依赖；④ `Evidence` 为空表示「未取证」，严禁在注释/日志里写成「已验证」。
</action>
</task>

<task id="2">
<title>单条链路接入型号维度 + dry-run 回显</title>
<files>internal/services/portwrite/physical.go, internal/services/portwrite/port_write_service.go, internal/services/portwrite/dry_run.go, internal/services/portwrite/physical_test.go</files>
<action>
`validatePhysicalRequest` 接 `(vendor, model, action, params)` 并走 `CheckPhysicalSupport`；`writeSinglePort`/`executeWrite` 把 `dev.Model` 透传进 `PortTemplateParams.Model`；`dry_run.go` 的 `PreviewResult` 加 `EvidenceLevel string`（`"unverified"` / `"model"` / `"vendor"`）与 `PhysicalEvidence string`，预览显示型号与取证级别；抽公共 `applyPhysicalAuditExtra(action, params, result)` 供单条与批量共用（`Extra` 增加 `physicalEvidence`）。保持 `PhysicalWrite` 成功后写 `Extra["speed"|"duplex"|"mtu"]` 的行为不变（`buildAfterValue` 依赖）。
</action>
</task>

<task id="3">
<title>批量物理参数接入</title>
<files>internal/services/portwrite/port_write_service.go, internal/services/portwrite/batch_orchestrator.go, internal/api/v1/network/port_write_handler.go</files>
<action>
按裁决改 `BatchWriteRequest`（加 4 字段）、入口 validator switch（物理三动作）、循环两处 `executeWrite` 的 `renderExtras`、批量查一次 `dev` 后调 `CheckPhysicalSupport`、每个成功 `writeResult` 调 `applyPhysicalAuditExtra`、handler sentinel 补 400 分支、`SkipPreflight` 登记。进度事件/三数组/`createAuditRowsInTx`/operlog/response **均不改**。
</action>
</task>

<task id="4">
<title>前端单条 + 批量入口</title>
<files>xingran-react-frontend/src/types/network.ts, src/components/network/port-write/constants.ts, src/components/network/port-write/PortWriteModal.tsx, src/components/network/port-write/BulkWriteDrawer.tsx, src/lib/api/networkApi.ts, 对应 __tests__/</files>
<action>
按裁决扩 `PortWriteAction`（15→18）、`BatchWriteRequest` 加 4 字段、`ACTION_TITLE`（15→18）+ `PHYSICAL_SPEED_OPTIONS`/`PHYSICAL_DUPLEX_OPTIONS`/`MTU_MIN/MAX` 常量、`CISCO_UNSUPPORTED_ACTIONS` 补物理动作；新增 `networkApi.writePhysical()` wrapped `post`（遵守 LANDMINE #5 不手动 `message.error`）；`PortWriteModal` 加物理条件表单（speed: AppSelect `{10,100,1000,10000}` + `speedMode=auto`；duplex: half/full/auto；mtu: InputNumber 1280~9216）；`BulkWriteDrawer` `ACTION_OPTIONS` 加 3 项、`buildRequest` 按 action 附带 4 字段、条件表单、预览 `evidenceLevel` 徽标（复用第 1093 行已有 `{vendor} {model}` 显示）。**不新建第二条 WebSocket**（沿用 `noticeStore` 的 `port_write_progress` 分流）。`useEffect` 依赖用 primitive/useMemo 稳定（CLAUDE.md）。
</action>
</task>

<task id="5">
<title>测试（就近，全部不依赖真机）</title>
<files>internal/services/portcollection/physical_capability_test.go (new), physical_template_test.go, internal/services/portwrite/physical_batch_test.go (new), physical_test.go, internal/api/v1/network/port_write_physical_handler_test.go, port_write_physical_batch_handler_test.go (new), 前端 __tests__/</files>
<action>
Go：
- `physical_capability_test.go` 表驱动：型号命中/未命中/空 Model/大小写/多规则优先级/回退链/`Evidence`/枚举范围越界/命令形态分叉（用 `device_capability_matrix_test.go:138-175` 的 map 备份+defer 恢复手法）。
- 扩 `physical_template_test.go`（型号 profile 改变命令体、未取证默认画像、Cisco 显式 Unsupported）。
- 扩 `physical_test.go`（vendor/model 校验矩阵、Extra 回归）。
- 新增 `physical_batch_test.go`：批量入口 validator、4 参数透传、`audit.after_value` 非空、fail-fast、三数组聚合、进度逐端口事件（复用 `batch_progress_test.go` 的 `fakeProgressNotifier`）、型号 Unsupported 批量拒绝。
- 扩 handler 测试：批量 sentinel 400、`operlog.Record` 在 `response.Success` 之前、router 注册断言保留。
前端：`constants.test.ts`（15→18）、`BulkWriteDrawer.test.tsx`（preview→confirm、参数透传、WS 进度）、`PortWriteModal.test.tsx`（物理表单）、`networkApi.test.ts`（`writePhysical`）。
</action>
</task>

<task id="6">
<title>自验证（只跑本 phase 相关包）</title>
<action>
`go build ./...`
`go test -count=1 ./internal/services/portcollection/ ./internal/services/portwrite/ ./internal/api/v1/network/`
`cd xingran-react-frontend && npm run type-check && npx vitest run src/components/network/port-write src/lib/api/__tests__/networkApi.test.ts`
不要跑 `go test ./...` / 全量 vitest（Lead 集成后统一跑）。失败只修本 phase 文件面问题，迭代至通过。
</action>
</task>

<task id="7">
<title>写 SUMMARY（不提交）</title>
<action>
写 `.planning/phases/143-port-01-02/143-SUMMARY.md`：改动清单、REQ 满足情况、验证命令与结果、**Known Gaps**（真机逐型号取证 7 项清单、`sys_object_id` 未落库、批量无自动回滚、`pkg/permission/config.go` 补登 `/network/ports/write/physical` 移交 Phase 144、e2e 归位）、「未 git commit」说明。
</action>
</task>

## 验证标准（must_haves）
1. 型号级矩阵有命中/厂商回退/未知 fail-safe 全分支单测，且 `Evidence=""` 明确标注未取证、不阻断写。
2. 批量 speed/duplex/mtu 走同一套校验/审计/命令分发（可静态断言无第二套实现），进度事件零改动复用，`audit.after_value` 非空。
3. 单条物理写（3 厂商 × 3 参数）既有测试全绿、零回归。
4. 前端 `PortWriteAction` 18 键、单条/批量表单可用、wrapped API、`evidenceLevel` 徽标；`constants.test.ts` 断言同步。
5. `go build ./...` + 3 个 Go 包测试 + `type-check` + port-write 相关 vitest 全绿。
6. 未触碰其它 phase 文件面；零新依赖、零新 migration。

## 禁止
- 禁止 `git add` / `git commit` / `git stash` / `git checkout`（Lead 统一集成提交）。
- **禁止修改 `pkg/permission/config.go`**（划归 Phase 144 独占；补登 `/physical` 写进 SUMMARY 移交）。
- 禁止修改 `internal/api/router.go`、`internal/core/db/database.go`、`internal/services/addomain/**`、`internal/services/device_discovery_service*.go`、`xingran-react-frontend/src/lib/api.ts`（barrel）、`src/store/**`、`src/constants/permissionCodes.ts`（其它 phase 独占）。
- 禁止新增 migration、禁止改 `NetworkDevice` 模型、禁止真机依赖。
- 禁止启动长驻进程（AGENTS.md）、禁止写生产数据。
