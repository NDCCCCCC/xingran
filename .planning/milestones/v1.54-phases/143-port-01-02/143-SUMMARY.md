---
phase: 143
plan: 01
status: complete
commit: false   # 未 git commit，由 Lead 统一集成提交
executor: gsd-executor / exec-143
---

# Phase 143 · 端口写物理参数收口（PORT-01/02）SUMMARY

## 改动清单（仅本 phase 授权文件面）

### 后端 · portcollection（型号级矩阵框架 / PORT-02）

| 文件 | 改动 |
|---|---|
| `internal/services/portcollection/physical_capability.go`（新） | `physicalProfile{AllowedSpeeds, AllowedDuplexes, MTUMin/Max, SpeedCommand, DuplexCommand, MTUCommand, RequiresView, Evidence, level}` + `modelPhysicalRule` + `PhysicalEvidence` 导出载体 + `PhysicalEvidenceLevel()` + `ResolvePhysicalProfile(vendor, model)` + `CheckPhysicalSupport(vendor, model, action, params)` + `validatePhysicalParams`。`defaultPhysicalProfile()` = FIX-04 三家交集（`{10,100,1000,10000}` / `half\|full\|auto` / `[1280,9216]` / `speed %s\|duplex %s\|mtu %d`），`Evidence=""` + `level="unverified"`。回退链严格对齐 `lookupSupport` 优先级（型号 unsupported → 型号画像 → 厂商 unsupported → 厂商基线 → 全局默认）。 |
| `internal/services/portcollection/device_capability.go` | `capabilityEntry` 扩 `physical *physicalProfile` / `modelPhysical []modelPhysicalRule` 两字段（不改既有字段语义、不改 `CheckActionSupport*` / `lookupSupport` 签名与行为）；Cisco / Maipu 条目把物理三动作显式标 `vendorUnsupported`（理由「渲染层未实现 + 无真机取证依据」，`physicalVendorUnsupportedReason`）。 |
| `internal/services/portcollection/vendor_port_template.go` | `PortTemplateParams` 加 `Model string`（`RenderCommand` 整体透传 params，签名不变）。 |
| `internal/services/portcollection/physical_template.go` | `renderPhysical` 改走 `CheckPhysicalSupport(vendor, p.Model, action, p)`，按画像 `SpeedCommand/DuplexCommand/MTUCommand` 分叉命令形态 + `RequiresView` 决定 interface 前缀；默认画像命令体三家与 FIX-04 完全一致（单条零回归）。保留 `SpeedAllowed/DuplexAllowed/MTUMin/MTUMax` 导出 helper 向后兼容。 |
| `internal/services/portcollection/physical_template_test.go` | 既有 `RendererRejectsInvalid` 断言同步为同源画像文案（`ErrPhysicalParamInvalid` + 值域信息）；追加「未取证默认画像 + 三家命令体零回归」「Cisco 显式 Unsupported」两组。 |
| `internal/services/portcollection/physical_capability_test.go`（新） | 表驱动：默认画像等价 FIX-04 / 回退链（型号命中·大小写·未命中·空 Model·多规则优先级·厂商基线·全局默认）/ 完整拒绝链 / 真实 Cisco+Maipu 矩阵 / 枚举范围越界 / 命令形态分叉（型号画像改命令体）/ 渲染层显式 Unsupported。沿用 `device_capability_matrix_test.go` 的 map 备份 + `t.Cleanup` 恢复手法，不依赖真机。 |

### 后端 · portwrite（单条型号维度 + 批量接入 / PORT-01）

| 文件 | 改动 |
|---|---|
| `internal/services/portwrite/physical.go` | `validatePhysicalRequest` 签名改 `(vendor, model, action, params)` 并走 `CheckPhysicalSupport`（校验/渲染同源），返回 `portcollection.PhysicalEvidence`；`translatePhysicalSupportError` 把能力矩阵错误映射到既有 400 sentinel（`ErrActionNotSupported` 原样透传，参数越界细分）。抽公共 **`applyPhysicalAuditExtra(action, params, evidence, result)`**（把 FIX-04 单条 `PhysicalWrite:92-109` 的 Extra 回显逻辑搬出，额外写 `physicalEvidence` / `physicalEvidenceLevel`）供单条与批量共用。`PhysicalWrite` 入口两步查设备拿 vendor/model，成功后调公共函数。 |
| `internal/services/portwrite/port_write_service.go` | `BatchWriteRequest` 加 4 字段（`speed`/`speedMode`/`duplex`/`mtu`，json tag 与单条 DTO 同名）；`SkipPreflight` 注释显式登记「**批量不支持跳过预检**（字段不被消费）」。`executeWrite` 把 `dev.Model` 透传进 `PortTemplateParams.Model`。进度事件 / 三数组 / `createAuditRowsInTx` / operlog / response 零改动。 |
| `internal/services/portwrite/batch_orchestrator.go` | 入口 validator：物理三动作查一次 dev → `validatePhysicalRequest(vendor, model, ...)` + `CheckActionSupportParams`（Cisco/Maipu/型号 unsupported 显式拒绝）。循环外构造共享 `renderExtras{physical}`；**两处** `executeWrite` 由 `nil` 改为传该 extras（4 参数透传）；每个成功 `writeResult` 调 `applyPhysicalAuditExtra` 修「批量 `audit.after_value` 恒 `{}`」。其余零改动。 |
| `internal/services/portwrite/dry_run.go` | `PreviewResult` 加 `EvidenceLevel string`（`unverified`/`model`/`vendor`）+ `PhysicalEvidence string`（空串=未取证）。设备查询提前到入口 validator 之前（物理 validator 需要 vendor/model）；probe 与 preview 的 `RenderCommand` 均传 `Model: dev.Model`；物理动作 `Summary` 追加「⚠️ 批量物理参数无自动回滚，失败需人工按 `audit.before_value` 恢复」警示。 |
| `internal/services/portwrite/physical_test.go` | 既有 validator 表驱动改新签名（huawei+空型号=全局默认画像，全部既有断言保留）；追加 `TestPhysical143_ValidateVendorModelMatrix`（三家默认放行且 Evidence 空串 / Cisco+Maipu 显式拒绝）。 |
| `internal/services/portwrite/physical_batch_test.go`（新） | 批量入口 validator（3 类非法参数零 SSH 流量）、4 参数透传 + `CommandSent` 含物理命令 + `Extra` 回显（修 audit.after）、fail-fast 三数组聚合、进度逐端口事件（`fakeProgressNotifier` + seed sys_user）、Cisco/Maipu 批量拒绝。 |

### 后端 · handler

| 文件 | 改动 |
|---|---|
| `internal/api/v1/network/port_write_handler.go` | `BatchWrite` sentinel switch 在 `default:500` 之前补 `portcollection.ErrActionNotSupported` 400 分支 + `physicalSentinelMessage(err)` 400 分支（与 `execSinglePort` 单条共用同一套物理 sentinel）。 |
| `internal/api/v1/network/port_write_physical_batch_handler_test.go`（新） | 批量 4 字段透传、5 类 sentinel → 400（不写 audit / 不调 operlog）、成功路径 `operlog.Record` 在 `response.Success` 之前 + `audit.after_value` 非空、router `/physical` 注册断言保留。 |

### 前端

| 文件 | 改动 |
|---|---|
| `xingran-react-frontend/src/types/network.ts` | `PortWriteAction` 15→18（`set_speed`/`set_duplex`/`set_mtu`）；`BatchWriteRequest` 加 4 物理字段；`PortWritePreviewResult` 加 `evidenceLevel` / `physicalEvidence`。 |
| `xingran-react-frontend/src/components/network/port-write/constants.ts` | `ACTION_TITLE` 15→18 + 物理三标题；新增 `PHYSICAL_SPEED_OPTIONS` / `PHYSICAL_SPEED_MODE_AUTO` / `PHYSICAL_DUPLEX_OPTIONS` / `PHYSICAL_MTU_MIN/MAX` / `PHYSICAL_NO_ROLLBACK_HINT` / `PHYSICAL_EVIDENCE_LABEL`；`CISCO_UNSUPPORTED_ACTIONS` 补物理三动作。 |
| `xingran-react-frontend/src/lib/api/networkApi.ts` | 新增 `writePhysical()` wrapped `post<PortResult>`（遵守 LANDMINE #5：无 try/catch、无 `message.error`、无裸 axios）。 |
| `xingran-react-frontend/src/components/network/port-write/PortWriteModal.tsx` | handleOk 加物理三动作分支调 `writePhysical`；条件表单（set_speed: 速率模式 Select + 条件速率 Select；set_duplex: Select；set_mtu: InputNumber 1280~9216），`canWrite` 门控不变；`useEffect` deps 仍为原始值 / 稳定 ref。 |
| `xingran-react-frontend/src/components/network/port-write/BulkWriteDrawer.tsx` | `ACTION_OPTIONS` 加 3 项（共 16 项）；`TrunkParams` 载体扩 4 物理字段（复用既有参数快照，确认写/重试与预览同参）；`buildRequest` 按 action 白名单附带 4 字段；`handlePreview` 组装物理参数；`SelectView` 加物理条件表单；`PreviewView` 加 `evidenceLevel` 徽标 + `unverified` 常驻「未取证（默认放行）」告警（含无自动回滚警示）。**未新建第二条 WebSocket**（沿用 `noticeStore` 的 `port_write_progress` 分流）。 |
| `xingran-react-frontend/src/components/network/port-write/__tests__/constants.test.ts` / `constants.test.tsx` | 15→18 键断言同步 + 物理常量 / CISCO 集合断言。 |
| `xingran-react-frontend/src/lib/api/__tests__/networkApi.test.ts` | `writePhysical` 路径与三动作参数透传。 |
| `xingran-react-frontend/src/components/network/port-write/__tests__/BulkWriteDrawer.test.tsx` | 追加 3 例：set_mtu preview→confirm 参数透传 + evidenceLevel 徽标 + 未取证告警；set_speed auto 透传；set_duplex 透传。 |
| `xingran-react-frontend/src/components/network/port-write/__tests__/PortWriteModal.test.tsx` | 追加 3 例：set_mtu / set_speed(auto) / set_duplex 物理条件表单调 `writePhysical`。 |

**未触碰**（核对 `git status` 确认）：`pkg/permission/config.go`、`internal/api/router.go`、`internal/core/db/database.go`、`internal/services/addomain/**`、`internal/services/device_discovery_service*.go`、`xingran-react-frontend/src/lib/api.ts`(barrel)、`src/store/**`、`src/constants/permissionCodes.ts`、`src/pages/system/**`、任何 migration、`NetworkDevice` 模型。**零新依赖、零新 migration、零真机依赖、零裸 axios、零第二条 WS**。全程未执行 `git add` / `git commit` / `git stash` / `git checkout`（Lead 统一集成提交）。

## PORT-01 / PORT-02 满足情况

### PORT-01（批量物理参数路径，单条零回归）
- `BatchWriteRequest` 4 字段 → 入口 validator（与单条 `validatePhysicalRequest` 同源）→ 两处 `executeWrite` 传 `renderExtras{physical}`（4 参数透传到 `RenderCommand`）→ 每个成功结果调公共 `applyPhysicalAuditExtra`。**校验 / 审计 / 命令分发与单条共用同一份实现，无第二套**（`renderPhysical` ↔ `CheckPhysicalSupport`，`applyPhysicalAuditExtra` 单条+批量共用）。
- 批量 `audit.after_value` 由恒 `{}` 修为含 `speed`/`duplex`/`mtu`（`physicalEvidenceLevel` / `physicalEvidence` 附证）。进度事件（逐端口 `progress.emit` + 终态 `emitCompleted`）、三数组 `Succeeded/Failed/Skipped` 聚合、`createAuditRowsInTx`、`operlog.Record(OperTypeBatch)`、`BatchWrite` response **零改动复用**；handler sentinel 补 400 分支。
- 单条 3 厂商 × 3 参数既有测试（`physical_template_test.go` 9 例 + `physical_test.go` + handler 端点 4 例）全绿、零回归；默认画像命令体与 FIX-04 逐字符一致。

### PORT-02（型号级 `vendorCapabilities` 规则）
- 载体 = 代码常量 + 测试锁定：`physicalProfile` / `modelPhysicalRule` 扩进 `capabilityEntry`，`ResolvePhysicalProfile` / `CheckPhysicalSupport` 为唯一真相源。只用 `NetworkDevice.Model` 子串匹配（大小写不敏感，复用 `matchModelRules` 语义），**不落库 `SysObjectID`、不加 migration**。
- 回退链全分支有单测：命中型号 → 型号画像（命令形态/值域分叉）；未命中 → 厂商基线（`level="vendor"`，Evidence 标「厂商基线，未逐型号取证」）；完全未知 → 全局默认 fail-safe（`level="unverified"`，`Evidence=""`，**不阻断写、不冒充真机结论**）；型号/厂商 `unsupported` → `ErrActionNotSupported` 400（Evidence 必含依据）；型号字段缺失（空 Model）→ 厂商基线/默认。
- `PreviewResult.evidenceLevel` + 前端徽标 + 审计 `Extra.physicalEvidence` 把「默认放行 ≠ 有证据」显式暴露；`Evidence=""` 全程只表示「未取证」，注释/日志/文案均未写成「已验证」。
- Cisco / Maipu 物理三动作显式 `vendorUnsupported`（渲染层未实现 + 无真机取证依据），不再靠 `ErrUnsupportedVendor` 含糊兜底；前端 `CISCO_UNSUPPORTED_ACTIONS` 同步。
- 全部用非真机 fixture / 内存 sqlite / mock executor 单测覆盖，零真机依赖。

## 验证（Task 6，逐项结果）

| 命令 | 结果 |
|---|---|
| `go build ./...` | **exit 0 / 全绿**（最终复跑，含其它并行 executor 的工作区状态） |
| `go test -count=1 ./internal/services/portcollection/ ./internal/services/portwrite/ ./internal/api/v1/network/` | **全绿**：`ok .../portcollection 0.454s`、`ok .../portwrite 121.519s`、`ok .../internal/api/v1/network 1.076s`（含全部新增测试） |
| `cd xingran-react-frontend && npm run type-check` | **exit 0 / 全绿** |
| `npx vitest run src/components/network/port-write src/lib/api/__tests__/networkApi.test.ts` | **8 files / 109 tests passed**（新增 6 例物理测试全部通过；其余 103 例既有测试零回归） |

> 按 PLAN **未跑** `go test ./...` / 全量 vitest（留待 Lead 集成后统一跑）；未跑其它 phase 的包测试；未启动任何长驻进程（一次性命令均前台 / DSH 托管后台 job 执行）。

### must_haves 自检
1. 型号级矩阵命中 / 厂商回退 / 未知 fail-safe 全分支单测，`Evidence=""` 明确标注未取证、不阻断写 ✅
2. 批量 speed/duplex/mtu 走同一套校验/审计/命令分发（`applyPhysicalAuditExtra` + `CheckPhysicalSupport` 静态唯一实现），进度事件零改动复用，`audit.after_value` 非空 ✅
3. 单条物理写（3 厂商 × 3 参数）既有测试全绿、零回归 ✅
4. 前端 `PortWriteAction` 18 键、单条/批量表单可用、wrapped API、`evidenceLevel` 徽标；`constants.test.ts(x)` 断言同步 ✅
5. `go build ./...` + 3 个 Go 包测试 + `type-check` + port-write 相关 vitest 全绿 ✅
6. 未触碰其它 phase 文件面；零新依赖、零新 migration ✅

## Known Gaps / 移交项（不伪造闭环）

1. **真机逐型号取证 7 项**（本里程碑不做，沿用 `physical_template.go:17-20` 的诚实标注）：① 速率关键字 / 10000 合法性；② 双工在光口 / 10G 的合法性；③ MTU 命令形态（`jumboframe enable` / `system mtu` / 接口 `mtu`）与真实上下界；④ 生效语义与链路瞬断；⑤ 回读验证（`sys_device_port_status` 有 Speed/Duplex 无 MTU）；⑥ 型号字符串形态覆盖；⑦ Cisco / 迈普显式 Unsupported 裁定。当前 `modelPhysical` 不预置任何型号结论，全部为「未取证」框架。
2. **`sys_object_id` 未落库**：`GetSystemInfo().ObjectID` 已采集但未持久化，型号 key 只用 `NetworkDevice.Model`；零 DDL，避免与 Phase 144 的 migration 238 冲突。后续若需 OID 级判定须单独设计落库 + migration。
3. **批量无自动回滚**：物理动作沿用非 critical 语义，不纳入 `PORTW-04` 快照自动回滚；dry-run `Summary` 与前端预览已显式警示「失败需人工按 `audit.before_value` 恢复」。
4. **`pkg/permission/config.go` 补登 `/network/ports/write/physical` → 移交 Phase 144**（PLAN 裁决表 + 用户指令：该文件已划归 Phase 144 独占，本 phase 禁止触碰；组级 RBAC 中间件仍生效，但 route-permission registry 口径不一致）。**Lead 集成时请安排 Phase 144 后端一并补登**（复用 `NetworkPortWrite` 常量）+ 注册表测试。注：架构裁决笔记 §三 第 95 行曾写「顺手补登」，以 PLAN 裁决表/禁止项与用户指令为准。
5. **e2e 归位**：真机 / 端到端 HTTP 联调归位未做（本里程碑只交付可代码化规则框架 + 前端入口）。
6. **`RequiresInterfaceView` 与型号画像 `RequiresView` 的关系**：`renderPhysical` 已按画像 `RequiresView` 决定命令体是否带 interface 前缀；`executeWrite` 的视图退出命令仍走既有 `RequiresInterfaceView(vendor, action)`（`lookupSupport(vendor, "", action)`），型号画像 `RequiresView=false` 的系统视图序列目前无真机型号结论触发（本里程碑无型号级 profile），登记为后续接入真机型号画像时的联动点。
7. **`SkipPreflight` 批量不消费**：字段保留为 DTO 向后兼容，批量路径显式登记「不支持跳过预检」（代码注释 + 本 SUMMARY），非遗漏。
8. **工作区为 4 executor 并行共享**：`git status` 中可见 Phase 142/144 的 `router.go` / `database.go` / `migration_238` / `addomain` / `device_discovery*` / `src/store` / `src/constants` 等改动，均为其它 executor 并行写入，与本 phase 文件面无交集，非本 executor 改动。

---

**未 git commit，由 Lead 统一集成提交。**
