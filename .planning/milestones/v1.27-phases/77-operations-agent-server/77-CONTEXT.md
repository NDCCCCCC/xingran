# Phase 77: 阻塞包攻破·零基建先行 (operations + agent/server) - Context

**Gathered:** 2026-08-24
**Status:** Ready for planning

<domain>
## Phase Boundary

用既有零基建手段（sqlite / httptest / 假策略 / excelize 内存生成）把投入产出排名前二的两个包推过 70% 线：

- `internal/services/operations`: 61.1% → ≥70%（补 ~330 stmts；workstation_device_service 445 unc + excel_service 399 unc 为主力）
- `internal/agent/server`: 22.1% → ≥70%（补 ~295 stmts；jwt_auth 107 + handlers 106 + connection_manager 80 走 httptest 假后端 + 假策略 + re-exec stub）
- Windows 本地与 ubuntu CI 的 agent 包覆盖率差 <2pp（env-branch divergence 收口）
- phase 边界 `go test ./...` 全绿，4 层 gate（weighted-avg / P1 floor / P2 floor / diff coverage）不倒退

**不在本 phase**: core/device/addomain（Phase 78 基建解锁）、长尾包（79/80）、ratchet bump 与 P2_RATCHET 豁免行删除（统一留 Phase 81 收口）。

</domain>

<decisions>
## Implementation Decisions

### 新 quirk 处理策略
- **D-01: 发现即修。** 补测试期间发现新业务 quirk 一律顺手修，全面应用 Phase 75 纪律：每项原子 commit + 同 commit 翻转断言 + 回归用例。**显式推翻 v1.26 D-12「0 业务代码改动」在 Phase 77 的适用**——v1.27 milestone 内没有下一个 QUIRK phase 来收新债。
- **D-02: plan 内就地修。** quirk 修复由 executor 在当前 plan 内直接完成，不走 /gsd-quick 分流；修复作为 deviation 记录在 SUMMARY（附根因 + 证据）。
- **D-03: 有据判定。** 判定「quirk 该修 vs 按现行为断言」的唯一标准是有据可查：models 常量 / 字段注释 / 开发规范文档 / API 响应规范 与代码行为不符 → 判 quirk 修；无据可查 → 保守按现行为断言 + SUMMARY 记录待人工裁决，不臆断设计意图。

### 覆盖率差 <2pp 实证机制（SC#3）
- **D-04: 收口人工对比一次。** phase 收口时本地 Windows 跑 `go test -cover`，与 push 后 CI 的 per-package 数字人工对比，差值记录进 77-VERIFICATION.md。不写对比脚本、不加 gate 层级（根因已被 Phase 76 re-exec 结构性消除，属一次性验证）。
- **D-05: 两包都对比。** `internal/agent/server` 按 SC#3 字面必测；`internal/services/operations` 顺带对比（本地/CI 数字本来都要跑，边际成本≈0，顺带获得 excelize 平台一致性信息）。

### Excel fixture 策略
- **D-06: 全内存生成，零二进制进 git。** 常规输入沿用既有先例 `excelize.NewFile()` + `xlsxFileHeader` 包装（excel_import_export_test.go:28/:45，与 multipart 上传入口同构）；畸形输入（非 zip 魔数 / 截断字节流）用手工字节构造（如 `[]byte("not a zip")`）。不新增 testdata/ 二进制文件。
- **D-07: 导出链结构断言。** ExportData/writeDataRows/writeInstructions/appendWorkstationDeviceSheets 断言 sheet 名 / 表头行 / 数据行数 + 抽查关键单元格（如工位设备 sheet 的序列号列）。不做全量逐单元格快照比对（列顺序调整即碎）。

### Plan 切分与编排
- **D-08: 照单 5 plan。** 按 ROADMAP 建议切分：77-01 workstation_device_service（GetADDevices/SyncFromAD/SyncFromAsset/mergeBySerial/SetPrimaryDevice*）/ 77-02 excel_service 导出链 / 77-03 excel_service 导入剩余 + reference_resolver + workstation/floor/code_generator/excel_raw_rows / 77-04 agent jwt_auth + connection_manager（httptest 假后端，backendURL 明文参数）/ 77-05 agent handlers（gin + Recorder）+ config 校验/注册 + account_manager（假策略上层 + re-exec 真策略体）。
- **D-09: planner 自主排 wave。** 不预设执行顺序偏好；operations 3 plan 与 agent 2 plan 无硬依赖，按 execute-phase 的 wave 机制自行编排并行。

### Claude's Discretion
- 测试文件命名沿用 Phase 76 先例 `{topic}_77_NN_test.go`（如 `redis_miniredis_76_01_test.go` 模式）
- jwt_auth / connection_manager 假后端的具体 httptest 形态由 researcher 调研定案
- workstation_device_service 的 SyncFromAD 若 AD 侧不可零基建 fake，具体处理方式（sqlite 资产段优先 / 轻量 stub）由 researcher 按「零基建先行」原则定案
- 测试内部结构（表驱动 vs 独立函数、helper 抽取粒度）由 planner/executor 按 76-PATTERNS.md analog 决定

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase 定义与需求
- `.planning/workstreams/milestone/ROADMAP.md` §Phase 77 — Goal / 4 条 Success Criteria / 5 plan 建议切分 / Notes（geocoding fakeGeocodeTransport 先例、Q-12/Q-13 已修、pty_manager Skipf 兜底）
- `.planning/workstreams/milestone/REQUIREMENTS.md` — BLOCK-01（operations ≥70%）/ BLOCK-02（agent/server ≥70%）

### v1.27 研究输入（地形图）
- `.planning/research/v1.27-features.md` — 5 阻塞包未覆盖语句地形图（operations 3714 stmts/1446 unc/61.1%；agent/server 616 stmts/480 unc/22.1%；agent 各文件 (c) 类清单与注入点三点齐备说明）
- `.planning/research/v1.27-architecture.md` — 基建接入架构（A1 隔离契约 + AST 守护；77 用到的 INFRA-04 re-exec 语义）
- `.planning/research/v1.27-pitfalls.md` — miniredis/fake SSH 三坑 + QUIRK blast radius

### Phase 76 交付物（77 的直接依赖）
- `.planning/workstreams/milestone/phases/76-test-doubles/76-PATTERNS.md` — 文件级 analog 映射（注意其开头更正：ldap_iface 实际 16 方法，76-VERIFICATION 终态为 20）
- `.planning/workstreams/milestone/phases/76-test-doubles/76-VERIFICATION.md` — INFRA-01..05 全部 SATISFIED 证据；INFRA-04 = TestHelperProcess re-exec（77-04/05 agent 测试直接使用）
- `.planning/workstreams/milestone/phases/76-test-doubles/76-RESEARCH.md` — INFRA 需求定义与 Code Examples

### 覆盖率治理基线
- `.planning/coverage-baseline.md` — ratchet 追踪表（77 期间阈值 55.6 不动，Phase 81 统一收口）
- `.github/scripts/check-coverage.sh` — 4 层 gate 现状（77 不得倒退的门槛）

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/services/operations/excel_import_export_test.go:28,45` — `xlsxFileHeader` helper 把内存 xlsx 字节包装成 `*multipart.FileHeader`，与上传入口同构（excel 导入测试直接复用）
- `internal/services/operations/excel_export_devices_test.go:221,239,270` — excelize 内存生成先例
- `internal/agent/server/agent_smoke_test.go` — httptest.NewRecorder/NewRequest 先例（CORS 测试）+ 构造器策略选择断言（handlers 测试直接扩展）
- `internal/agent/server/subprocess_stub_test.go` / `subprocess_pgroup_test.go` — Phase 76 的 TestHelperProcess re-exec stub（77-05 account_manager 真策略体测试使用）
- `internal/services/operations/geocoding_photo_floor_test.go` — fakeGeocodeTransport(RoundTripper) 先例（httpmock 在本包仅边际价值）
- `internal/services/addomain/ldap_client_mock_test.go` — per-interface `*Func` fields mock 范本（v1.26 D-02 先例）

### Established Patterns
- testify 断言 + `t.Helper()` + `t.Cleanup` 生命周期绑定（76 全部测试文件沿用）
- 测试文件命名 `{topic}_{phase}_{plan}_test.go`
- 闭包注入缝纪律：生产 .go 禁引用 `*ForTesting`（AST 守护 for_testing_guard_test.go 三层隔离契约）
- handler 测试直调 mock service（v1.26 D-09：encryption middleware 不在 handler 层测试范围）

### Integration Points
- gate 链：`go test ./...` 全绿 → check-coverage.sh 4 层不倒退 →（收口后）本地 vs CI per-package 人工对比进 77-VERIFICATION.md
- coverage 数字来源：本地 `go test -cover` per-package 输出；CI 侧 backend-coverage artifact（artifact 内可比数据结构由 researcher 确认）

</code_context>

<specifics>
## Specific Ideas

- quirk「有据判定」的据源优先级：models 常量（status_constants_test.go 锁值）> 字段注释 > `docs/standards/开发规范.md` / `docs/standards/API响应规范.md`
- 导出链抽查关键单元格的锚点：工位设备 sheet 的序列号列（mergeBySerial 的合并主键）
- agent config 测试直接断言 Q-12/Q-13 修复后的新行为（非法 level 返回 error / TLS 空参报错），不写 workaround

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 77-阻塞包攻破·零基建先行 (operations + agent/server)*
*Context gathered: 2026-08-24*
