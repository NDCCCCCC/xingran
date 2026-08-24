# Phase 77: 阻塞包攻破·零基建先行 (operations + agent/server) - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-24
**Phase:** 77-阻塞包攻破·零基建先行 (operations + agent/server)
**Areas discussed:** 新 quirk 处理策略, 覆盖率差 <2pp 实证机制, Excel fixture 策略, Plan 切分确认

---

## 新 quirk 处理策略

| Option | Description | Selected |
|--------|-------------|----------|
| 分级处理 | 影响测试断言正确性的就地修+原子 commit+翻转断言；不影响测试写的纯行为怪癖只记录留债 | |
| 一律只记录不修 | 严守 v1.26 D-12 先例：77 期间 0 业务代码改动，新 quirk 一律记录，测试绕开写 | |
| 一律顺手修 | Phase 75 模式全面应用：发现即修，每项原子 commit + 同 commit 翻转断言 + 回归用例 | ✓ |

**User's choice:** 一律顺手修
**Notes:** v1.27 milestone 内没有下一个 QUIRK phase 收新债，故放弃 Claude 推荐的分级方案，选择彻底的发现即修——77 允许业务代码改动（显式推翻 v1.26 D-12 在本 phase 的适用）。

### 修复通道

| Option | Description | Selected |
|--------|-------------|----------|
| plan 内就地修 | executor 在 plan 内直接修，quirk 修复作为 deviation 记录在 SUMMARY（附根因+证据） | ✓ |
| 分流 quick-task | quirk 修复分流到独立 /gsd-quick 小任务，77 plan 保持纯测试 | |

**User's choice:** plan 内就地修

### 判定标准

| Option | Description | Selected |
|--------|-------------|----------|
| 有据判定 | 有据可查（models 常量/注释/文档）且代码与据不符 → 判 quirk 修；无据可查 → 按现行为断言 + SUMMARY 记录待人工裁决 | ✓ |
| 逐项 checkpoint | executor 发现疑似 quirk 即停下问用户 | |
| 宽判定 | 凡「反直觉」即判 quirk 修 | |

**User's choice:** 有据判定
**Notes:** Phase 75 修的 15 项有 v1.26 期间既有清单背书；77 是现场发现，需防止把有意设计（如 menu visible 1=可见）误当 bug。

---

## 覆盖率差 <2pp 实证机制

| Option | Description | Selected |
|--------|-------------|----------|
| 收口人工对比 | 77 收口时本地跑 go test -cover 与 push 后 CI per-package 数字对比一次，差值记录进 77-VERIFICATION.md | ✓ |
| 半自动脚本 | 扩展 check-coverage.sh 或新脚本对比本地与 CI artifact coverage.out，≥2pp exit 非 0 | |
| 不单独实证 | 根因已被 re-exec 消除 + P2 floor 过线即视为收口 | |

**User's choice:** 收口人工对比
**Notes:** 根因（echo 平台分支）已被 Phase 76 re-exec 结构性消除，一次性验证足够，不为它加 gate 层级。

### 对比范围

| Option | Description | Selected |
|--------|-------------|----------|
| 两包都对比 | agent/server 按 SC#3 + operations 顺带（边际成本≈0，多一份 excelize 平台一致性信息） | ✓ |
| 仅 agent/server | 严格按 SC#3 字面 | |

**User's choice:** 两包都对比

---

## Excel fixture 策略

| Option | Description | Selected |
|--------|-------------|----------|
| 全内存生成 | 常规输入 excelize.NewFile() + xlsxFileHeader（既有先例）；畸形输入手工字节构造；零二进制进 git | ✓ |
| 内存+testdata 畸形 | 常规内存生成；难以手工构造的畸形 .xlsx 提交最小 testdata 二进制 | |
| testdata 为主 | 提交真实 .xlsx 作主 fixture | |

**User's choice:** 全内存生成
**Notes:** 代码侦察确认 excel_import_export_test.go:28/:45 已有 xlsxFileHeader 同构入口先例，与包内 29 个既有测试文件风格一致。

### 导出断言深度

| Option | Description | Selected |
|--------|-------------|----------|
| 结构断言 | sheet 名/表头行/数据行数 + 抽查关键单元格（如序列号列） | ✓ |
| 全量快照比对 | 读回 xlsx 逐单元格比对（含指令 sheet） | |
| 按风险混合 | 主链路结构断言；Q-12/Q-13 行为点精确断言 | |

**User's choice:** 结构断言

---

## Plan 切分确认

| Option | Description | Selected |
|--------|-------------|----------|
| 照单 5 plan | 77-01 workstation_device / 02 excel导出 / 03 excel导入+杂项 / 04 agent jwt+connmgr / 05 agent handlers+config+account_manager | ✓ |
| 合并 excel → 4 plan | excel 导出/导入合并（同文件内聚） | |
| 合并 agent → 4 plan | agent 04+05 合一（~295 unc，与 Phase 73 单 plan 先例同量级） | |

**User's choice:** 照单 5 plan

### 执行编排

| Option | Description | Selected |
|--------|-------------|----------|
| planner 自主 | 按 execute-phase wave 机制排依赖，不预设顺序偏好 | ✓ |
| operations 先行 | operations 3 plan 串行完成再 agent 2 plan | |
| 双线并行起步 | 77-01 与 77-04 同 wave 起步 | |

**User's choice:** planner 自主

---

## Claude's Discretion

- 测试文件命名沿用 `{topic}_77_NN_test.go`（76 先例）
- jwt_auth/connection_manager 假后端的 httptest 具体形态（researcher 定案）
- SyncFromAD 的 AD 侧不可零基建 fake 部分的处理方式（researcher 按「零基建先行」原则定案）
- 测试内部结构（表驱动 vs 独立函数、helper 粒度）

## Deferred Ideas

None — discussion stayed within phase scope
