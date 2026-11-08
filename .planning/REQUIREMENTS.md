# Requirements: RuoYi-Next 运维管理系统

**Defined:** 2026-11-08
**Milestone:** v1.54 真实代码缺口清偿 (Code Gap Clearance)
**Core Value:** 把 v1.53 收档时显式披露、登记在 `.planning/STATE.md ## Deferred Items` 的 4 项真实代码缺口真正落地为可运行、可测试的产品能力；不触碰 LOCKED 残余、不伪造人工验证闭环。

## 取数权威与范围裁定

- **取数**：`.planning/STATE.md ## Deferred Items` 中 v1.53 的 `code_gaps` / `verification_gaps` 两行 + `.planning/MILESTONES.md:19-24`（v1.53 Known Gaps）+ `.planning/milestones/v1.53-phases/*-SUMMARY.md` 的自报边界。
- **范围裁定（用户 2026-11-08）**：仅 4 项**纯代码缺口**纳入；3 项需生产环境/真机的验证类、FIX-04 真机逐型号取证、79 条人工验证跟踪表、6 项 LOCKED 残余**均不纳入**（见 Future Requirements / Out of Scope）。
- **Phase 编号**：延续 v1.53（结束于 141），从 **142** 续编。
- **跳过领域研究**：4 项均带本仓 `文件:行号` 实证、无新领域能力需调研；改由 3 个并行 subagent 做本仓代码面探索作为需求输入（沿用 v1.53 先例）。

## v1.54 Requirements

### DISC — 发现设备真实导入

- [ ] **DISC-01**: 用户可把一次设备发现任务中发现的设备真实导入为网络设备（`NetworkDevice`）——完成 `DiscoveredDevice → NetworkDevice` 字段映射、按可判定键去重、事务边界、部分成功/失败语义；替换 `internal/services/device_discovery_service.go:509-523` 的 `ImportDiscoveredDevices` 简化实现（现状仅 `return len(deviceIDs), nil`）。
- [ ] **DISC-02**: 导入端点符合全仓写操作约定——success path 在 `response.Success(...)` 之前调用 `operlog.Record(...)`、缓存失效、错误以 `error` 逐层返回并由 handler 用 `response.Error(...)` 包装；`discovery_handler.go` 与相关测试同步更新（删除「简化实现」断言），SQLite+PG 双方言均不报错。

### ALRT — 同步漂移告警通道收口

> 补齐 FIX-11 自报边界：v1.53 已交付常态化漂移告警**后端服务**，但前端告警页 / WS 广播 / 企微钉钉 webhook 未接。

- [ ] **ALRT-01**: 用户可在后端查询与配置漂移告警——告警列表/详情/已读状态端点 + 通道配置项（企微/钉钉 webhook URL、启用开关），权限码挂载，敏感 webhook 配置走 `operlog.RecordWithBody` 脱敏。
- [ ] **ALRT-02**: 漂移告警产生时经 `internal/websocket/` 广播到在线客户端（复用现有广播 API 与消息体格式），并支持企微/钉钉 webhook 推送（凭证不落明文日志、失败可观测、不阻塞主流程、operlog 覆盖）。
- [ ] **ALRT-03**: 用户可在前端「同步漂移告警」页面查看告警列表、筛选与标记已读，并经 WS 实时收到新告警——页面注册路由与菜单、store、API 用 `@/lib/api.ts` wrapped functions（禁止裸 axios）。

### PORT — 端口写物理参数收口

> 补齐 FIX-04 / FIX-05 自报边界：v1.53 已交付 3 厂商×3 参数（速率/双工/MTU）单条写 + 校验 + 审计，但型号级能力矩阵与批量路径未做。

- [ ] **PORT-01**: 用户可在批量端口写中使用物理参数——沿用 v1.51 Phase 122 批量写实时进度事件，补齐批量路径对 speed/duplex/mtu 动作的分发、逐项结果与错误聚合，单条路径零回归。
- [ ] **PORT-02**: 端口写物理参数支持型号级 `vendorCapabilities` 规则——型号来源取自设备已有字段（不依赖真机取证）、厂商级默认值回退、未知型号 fail-safe（拒绝或降级到安全默认值，结论须显式登记）、可通过非真机 fixture 单测覆盖全部分支。

## Future Requirements

以下为 v1.53 已显式披露的残余，**本里程碑不交付、不伪造闭环**，保持原登记状态：

- **VAL-类（3 项）**：FIX-01 实机 AD/LDAP UAT（UAC 544 + 登录恢复）｜FIX-09 端到端 HTTP P99 压测（v1.53 仅计算路径近似基线）｜FIX-13 生产库 migration 236/237 待执行后受限角色才真正 403→200。继续由 `.planning/notes/261108-human-verification-tracker.md`（79 条，0 勾选）与回写口径承载。
- **真机取证类**：FIX-04 三厂商物理参数真机逐型号取证（本里程碑 PORT-02 只做**可代码化的规则框架**，不替代真机结论，未决型号差异登记为待人工验证）。
- **FUTURE-01**：P1-5 聚合计数侧信道告警通道与 FIX-11 告警通道合并评估（见 `.planning/notes/261108-v153-locked-residuals.md:63`）。

## Out of Scope

| Feature | Reason |
|---------|--------|
| 79 条人工验证跟踪项的实际执行与勾选 | 需生产 PG 只读 / 写操作授权 / 现场真机 / DBA 回传；沿用 v1.25 Phase 90 先例，AI 不擅自写生产数据、不伪造闭环 |
| 6 项 LOCKED 残余（P1-5 侧信道 / 5 项豁免面 / PG 专属 SQL 债 / operlog_e2e_verify.sh 语义 / `invalidateCache` no-op / `pbkdf2SM3` 多块循环） | v1.53 已裁定「接受/仅登记，不改代码」，见 `.planning/notes/261108-v153-locked-residuals.md` |
| 生产数据库迁移执行、生产配置切换、真机设备命令取证 | 属人工/现场执行，超出代码缺口范围 |
| 新领域能力引入、新外部依赖 | 4 项均为本仓已有能力的补齐；沿用 v1.53「零新依赖」约束 |
| 工作站导入部门/用户关联、v1.0~v1.53 已交付功能 | 历史归档，非当前待办 |

## Constraints (inherited)

- Handler-Service 分层；service 返回 `error`；handler 用 `response.Success()` / `response.Error()` 包装，禁止裸 `c.JSON()`。
- 所有业务写操作 success path 在 `response.Success(...)` **之前**调用 `operlog.Record(...)`；含 token/secret/webhook 的端点用 `operlog.RecordWithBody` 脱敏（包 `github.com/xingran-next/xingran-go-backend/internal/utils/operlog`）。
- Status `0=正常 1=停用`；UUID 主键；SQLite+PG 双方言（避免 PG 专属 SQL，uuid 双侧 CAST）。
- 前端用 wrapped API（`@/lib/api.ts` 的 `post` 等）、Zustand store；禁止裸 axios。
- Go 单测就近放业务包内 `*_test.go`；集成测试 `tests/integration/`；e2e `tests/e2e/`；前端 `*.test.ts(x)`。
- 构建产物统一落 `out/`；Excel 模板动态生成、禁止落盘 `.xlsx`；`data/` 不入库。
- 回归基线：`go build ./...` + `go test ./...` + `tsc` + `lint:ci` + `vitest` 全绿。

## Traceability

Which phases cover which requirements. Updated during roadmap creation (Step 10 by gsd-roadmapper).

| Requirement | Phase | Status |
|-------------|-------|--------|
| DISC-01 | — | Pending |
| DISC-02 | — | Pending |
| ALRT-01 | — | Pending |
| ALRT-02 | — | Pending |
| ALRT-03 | — | Pending |
| PORT-01 | — | Pending |
| PORT-02 | — | Pending |

**Coverage:**
- v1.54 requirements: 7 total
- Mapped to phases: 0（待 roadmap）
- Unmapped: 7

---
*Requirements defined: 2026-11-08*
*Last updated: 2026-11-08 after 用户确认「纯代码缺口 4 项」范围、跳过领域研究、授权自动提交*
