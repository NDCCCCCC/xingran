# Phase 55: 技术债清理 Phase 53 leftover sweep - Context

**Gathered:** 2026-07-08
**Status:** Ready for planning

<domain>
## Phase Boundary

清理 Phase 53 code review (`53-REVIEW.md`) 与 verification (`53-VERIFICATION.md`) 记录的 **5 项可执行遗留代码技术债**。纯技术债，无新功能需求。范围被 ROADMAP 锁定为固定 5 项：

1. **WR-02** — port-write reason validator 签名 bug（前端 TS）
2. **IN-01** — `ports/index.tsx` `handleBatchExport` `error: any` 类型收窄（前端 TS）
3. **IN-02** — `ports/index.tsx` mount useEffect 补 eslint-disable（前端 TS）
4. **CR-02** — `batch_orchestrator.go` fallback 路径 port 归属跨层防御（后端 Go）
5. **HealthCard.test.tsx** — 修复既有失败测试（前端测试）

**不在本阶段范围**（已记录他处）：8 项真机/浏览器 UAT → Phase 54 HUMAN-UAT；Security 审计 → `/gsd:secure-phase 53`；commits push → git 操作。

</domain>

<decisions>
## Implementation Decisions

### WR-02 — port-write reason validator 修复
- **D-01（修/不修决策）:** **无条件修**。原 ROADMAP 设计"由 54-HUMAN-UAT #7 现场使用频率驱动 修/wontfix"，但现场访问未发生、UAT #7 仍 `[pending]` 无数据。经代码查证：validator **签名 bug 客观存在且与使用频率无关** —— `validateReasonOptional(_, reasonSelect, reasonText)` 期望 3 个参数，但 antd validator 实际只传 `(rule, value)`，导致 `reasonText` 恒 `undefined`，custom-reason 的 `REASON_MIN=5` 长度下限校验彻底失效（用户可填 1 字符绕过）。UAT 频率只影响优先级，不影响 bug 是否存在。
- **D-02（修复方式）:** 用 `Form.useWatch("reasonText", form)` 跨字段取值修正 validator 签名（不再依赖不存在的第 3 个参数），并把 `validateReasonOptional` / `validateReasonRequired` helper 抽到 `src/components/network/port-write/constants.ts` 共享，`PortWriteModal.tsx` 与 `BulkWriteDrawer.tsx` 统一引用。ROADMAP 备注即倾向 `Form.useWatch` 方案。
- **D-03（两处都要修对）:** `PortWriteModal.tsx` 是签名错（validator 存在但拿不到 reasonText）；`BulkWriteDrawer.tsx` 是校验缺失（`reasonSelect` 无 rules、`reasonText` 只有 `{ max }` 无 `REASON_MIN` 下限、`handleBatch` 对 description action 完全跳过 reason 校验）。两处修完后行为一致：单端口路径与批量路径的 description-action reason 长度校验对齐。

### CR-02 — 后端 fallback 跨层防御
- **D-04:** **仅在 fallback 分支加校验**（不做全路径）。正常路径已由 `WHERE device_id = ? AND id IN ?` 隔离；风险仅存在于 port 查不到（`!exists`）走 fallback `executeWrite(ctx, portID, req.DeviceID, ...)` 时完全信任前端 `req.DeviceID`。修复：fallback 分支先 `s.db.First(&port, "id = ?", portID)` 查 port 真实 deviceID，与 `req.DeviceID` 不一致则归入 `result.Failed` 并标 `error: "port does not belong to device"`，**不调 SSH**。属跨层防御纵深（前端 CR-01 已修根因，此为后端兜底双保险）。额外 1 次 DB 查询仅命中 fallback 罕见路径，正常路径零开销。

### HealthCard.test.tsx — 失败测试修复
- **D-05:** **只修 2 处断言**。实测确认失败根因是 **断言 bug 而非环境/时序**（ROADMAP 原假设"疑似环境/时序"被推翻）：测试用 `getByText("该工位暂无关联资产。")` 精确匹配，但组件把「对账健康度:」前缀与消息渲染在**同一文本节点**（实际渲染 `对账健康度:该工位暂无关联资产。`），精确匹配失败。修复：2 处 exact `getByText` 改为 substring/regex 匹配（如 `getByText(/该工位暂无关联资产/)`）。
- **D-06:** 80s import 耗时异常（ROADMAP 提到 112s）**记为 deferred，不在本阶段排查** —— 是独立的性能/依赖图问题，与断言 bug 无关，混入会发散范围。

### IN-01 / IN-02 — pre-existing lint 清理
- **D-07:** **只动 `ports/index.tsx` 报告的 2 处**，不扩范围（遵循 CLAUDE.md "Scope Constrainment" 规则：先修报告的具体项、不主动扫其他模块 / 不跨文件、不同文件内顺带）。
  - IN-01: `handleBatchExport` 的 `catch (error: any)` → `instanceof Error` 安全取 message。
  - IN-02: 第一个 mount-only `useEffect`（依赖数组为空但调用 loadDevices/loadStatistics/loadPortStatus）补 `// eslint-disable-next-line react-hooks/exhaustive-deps`。

### Claude's Discretion
- 5 项之间的 plan 拆分粒度、提交策略、验收方式交由 planner 决定。建议关注：前端 3 项（WR-02/IN-01/IN-02 + HealthCard 测试）可归一个前端 plan，后端 CR-02 单独一个 Go plan（涉及 `go build ./...` + `go test`）。

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase 53 遗留来源（决策依据）
- `.planning/phases/53-w4-frontend-drawer-progress-dialog-api-wrappers/53-REVIEW.md` — 5 项遗留的原始记录。WR-02 §153-236；IN-01 §238；IN-02 §264；CR-01/CR-02 §40-113；汇总表 §333-345（含各项 severity 与 deferred 理由）
- `.planning/phases/53-w4-frontend-drawer-progress-dialog-api-wrappers/53-VERIFICATION.md` — Phase 53 验证记录（遗留项的第二来源）

### Phase 54 UAT（WR-02 决策上下文）
- `.planning/phases/54-w5-e2e-real-device-uat-documentation/54-HUMAN-UAT.md` §7 — WR-02 custom-reason 现场使用频率观察，状态 `[pending]`（现场访问未发生 → 无频率数据 → 本阶段改为"无条件修"，不再等 UAT）

### 待修改代码（前端 TS）
- `xingran-react-frontend/src/components/network/port-write/PortWriteModal.tsx` — `validateReasonOptional`/`validateReasonRequired` §91-113（签名错）、`composeReason` §82-88、reasonRules §192-193、reasonText Form.Item §233-237
- `xingran-react-frontend/src/components/network/port-write/BulkWriteDrawer.tsx` — SelectView reasonSelect/reasonText Form.Item（无/弱 rules）、`handleBatch` reason 校验缺失
- `xingran-react-frontend/src/components/network/port-write/constants.ts` — `REASON_MIN=5`/`REASON_MAX=200`/`PRESET_REASONS`/`REASON_CUSTOM_SENTINEL`；helper 抽取目标文件
- `xingran-react-frontend/src/pages/network/ports/index.tsx` — `handleBatchExport` `catch (error: any)` §232、mount useEffect §175
- `xingran-react-frontend/src/components/reconciliation/__tests__/HealthCard.test.tsx` — 失败断言 §127（及另一处同类）

### 待修改代码（后端 Go）
- `internal/services/portwrite/batch_orchestrator.go` — 批量 pre-state 查询 §50-60、fallback 分支 `executeWrite(ctx, portID, req.DeviceID, ...)` §70-87（CR-02 校验插入点）

### 约定参考
- `D:\code\ClaudeCode\xingran-go-backend\CLAUDE.md` §"Debugging & Bug Fixing → Scope Constrainment"（IN-01/IN-02 不扩范围依据）；§"Compilation & Build Verification"（后端改动后 `go build ./...`）；§"Status Value Convention"

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `constants.ts` 已集中 `REASON_MIN`/`REASON_MAX`/`PRESET_REASONS`/`REASON_CUSTOM_SENTINEL`/`ACTION_TITLE` —— 是 WR-02 helper 抽取的天然落点，两组件已 import 该文件。
- `composeReason(reasonSelect, reasonText)` 逻辑已存在于 PortWriteModal，抽取时应连同 validator 一起下沉到 constants.ts 保持内聚。
- `batch_orchestrator.go` fallback 分支已有 `s.executeWrite(...)` 调用与 `result.Failed` append 模式，CR-02 校验可直接复用现有 Failed 归类结构。

### Established Patterns
- antd Form 跨字段校验规范：validator 只接 `(rule, value)`，跨字段值必须经 `Form.useWatch(name, form)` 获取（本次 WR-02 修复的核心正确姿势）。
- Go fallback fail-fast loop（`batch_orchestrator.go` §68 起）：`break // fail-fast` 语义，CR-02 校验失败应归 Failed 后 `continue`（不 break，因为是数据校验失败而非 SSH 失败）—— planner 需明确此处 continue vs break 语义。
- 前端错误处理：项目 TS 严格风格用 `instanceof Error` 收窄，非 `error: any`（IN-01 对齐点）。

### Integration Points
- WR-02 修复后，`PortWriteModal` 与 `BulkWriteDrawer` 的 description-action reason 校验行为需与后端 `PortWriteRequest.reason` 约定一致（`REASON_MAX=200` 已对齐后端保守上限）。
- CR-02 后端校验的 error message `"port does not belong to device"` 会经 `result.Failed` 返回前端 ResultView 失败明细，需确认前端能正常渲染该 message。

</code_context>

<specifics>
## Specific Ideas

- WR-02 修复方式 ROADMAP 已明确点名 `Form.useWatch` —— 属用户/roadmap 既定倾向，非开放选择。
- CR-02 error message 文案固定为 `"port does not belong to device"`（53-REVIEW §113 原文）。
- HealthCard 修复采用 substring/regex（如 `/该工位暂无关联资产/`），保留对「对账健康度:」前缀节点的容忍。

</specifics>

<deferred>
## Deferred Ideas

- **HealthCard.test.tsx import 耗时 80s（ROADMAP 记 112s）异常** — 独立性能/依赖图问题（疑似某 barrel import / three.js / echarts 重依赖被拉进测试环境），本阶段只修断言不排查。建议后续单开 phase 或性能专项。
- **WR-02 现场使用频率 UAT #7** — `54-HUMAN-UAT.md` §7 仍 `[pending]`，需下次现场访问后 1 周观察。本阶段已因 bug 客观存在改为"无条件修"，故此 UAT **不再阻塞修复决策**，但观察本身仍应在现场访问时完成并回写（informational）。
- **WR-03 / WR-04**（53-REVIEW 记录的其余 warning：BulkWriteDrawer selectedPorts 快照失真、ACTION_OPTIONS 类型断言不安全）— 不在 ROADMAP Phase 55 锁定的 5 项内，未纳入本阶段。
- **CR-02 全路径归属校验**（对所有 portID 而非仅 fallback 验证）— 本阶段选仅 fallback，更彻底的全路径方案留待未来需要时评估。

### Reviewed Todos (not folded)
None — 无 todo 匹配本阶段。

</deferred>

---

*Phase: 55-phase-53-leftover-sweep*
*Context gathered: 2026-07-08*
