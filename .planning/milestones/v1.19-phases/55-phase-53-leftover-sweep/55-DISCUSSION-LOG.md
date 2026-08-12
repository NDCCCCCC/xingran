# Phase 55: 技术债清理 Phase 53 leftover sweep - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-08
**Phase:** 55-phase-53-leftover-sweep
**Areas discussed:** WR-02 修/不修决策, WR-02 修复方式, CR-02 后端防御深度, HealthCard 测试范围, IN-01/IN-02 lint 清理边界

---

## 区域选择（multiSelect）

| Option | Description | Selected |
|--------|-------------|----------|
| WR-02 修/不修决策 | UAT #7 pending，bug 是否修 | ✓ |
| CR-02 后端防御深度 | fallback 路径 port 归属校验 | ✓ |
| HealthCard 测试范围 | 断言 bug vs import 异常 | ✓ |
| IN-01/IN-02 + lint 清理边界 | scope 收敛 vs 机会性清理 | ✓ |

**User's choice:** 全部 4 项。

---

## WR-02 修/不修决策

| Option | Description | Selected |
|--------|-------------|----------|
| 无条件修（推荐） | bug 客观存在且与频率无关，validator 签名错致长度下限失效 | ✓ |
| defer 出本阶段 | 严格遵循 ROADMAP，等 UAT #7 频率数据 | |
| 最小修（仅签名） | 只修签名生效，不做 helper 抽取 | |

**User's choice:** 无条件修。
**Notes:** 关键推翻——ROADMAP 原设计由 54-UAT #7 频率驱动修/wontfix，但现场访问未发生、UAT 仍 pending 无数据。代码查证 validator 签名 bug（`(_, reasonSelect, reasonText)` vs antd `(rule, value)`，reasonText 恒 undefined）客观存在，与使用频率无关。

## WR-02 修复方式

| Option | Description | Selected |
|--------|-------------|----------|
| useWatch + 抽 constants 共享（推荐） | Form.useWatch 修签名 + helper 下沉 constants.ts 两组件共享 | ✓ |
| 各组件内联修，不抽共享 | 快但两处逻辑重复 | |
| 你决定 | 交 planner 定粒度 | |

**User's choice:** useWatch + 抽 constants 共享。
**Notes:** ROADMAP 备注即倾向 Form.useWatch。两处都要修对：PortWriteModal 签名错、BulkWriteDrawer 校验缺失。

## CR-02 后端防御深度

| Option | Description | Selected |
|--------|-------------|----------|
| 仅 fallback 路径校验（推荐） | 仅 !exists 分支查 port 真实 deviceID，≠req.DeviceID 归 Failed 不调 SSH | ✓ |
| 全路径归属校验 | 所有 portID 都验证，改动面大 | |
| 不加后端防御 | 前端 CR-01 已修，后端不动 | |

**User's choice:** 仅 fallback 路径校验。
**Notes:** 正常路径已被 WHERE device_id 隔离；风险仅 fallback 完全信任 req.DeviceID。额外 1 次 DB 查询仅命中罕见 fallback，正常路径零开销。error message 固定 "port does not belong to device"。

## HealthCard 测试范围

| Option | Description | Selected |
|--------|-------------|----------|
| 只修 2 处断言（推荐） | exact getByText → substring/regex；import 异常 defer | ✓ |
| 修断言 + 查 import 异常 | 一并排查 80s import 根因，范围发散 | |
| 只修断言、不记 import | 连 import 异常都不记录 | |

**User's choice:** 只修 2 处断言。
**Notes:** 实测推翻 ROADMAP "疑似环境/时序" 假设——实为断言 bug：组件把「对账健康度:」前缀与消息渲染同节点，exact getByText 匹配失败。80s import 异常记为 deferred，独立性能问题不混入。

## IN-01/IN-02 lint 清理边界

| Option | Description | Selected |
|--------|-------------|----------|
| 只动报告的 2 处（推荐） | 仅 ports/index.tsx 这 2 处，符合 Scope Constrainment | ✓ |
| 同文件内顺带清 | 顺带同文件其他 error:any | |
| 模块级扫 error:any | 扫整个 network/ports 模块 | |

**User's choice:** 只动报告的 2 处。
**Notes:** 遵循 CLAUDE.md "Scope Constrainment"——先修报告的具体项、不主动扩到其他模块。

---

## Claude's Discretion

- 5 项之间的 plan 拆分粒度、提交策略、验收方式交由 planner 决定（建议前端 3 项 + HealthCard 归一个前端 plan，后端 CR-02 单独 Go plan）。

## Deferred Ideas

- HealthCard.test.tsx 80s（ROADMAP 记 112s）import 耗时异常 — 独立性能/依赖图问题，本阶段只修断言。
- WR-02 现场使用频率 UAT #7（54-HUMAN-UAT §7）仍 pending — 已不阻塞修复决策，观察本身待现场访问完成回写（informational）。
- WR-03 / WR-04（53-REVIEW 其余 warning）— 不在 ROADMAP 锁定的 5 项内，未纳入。
- CR-02 全路径归属校验 — 本阶段选仅 fallback，全路径方案留待未来评估。
