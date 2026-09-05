# Phase 95: v1.28 SHIP 收口 + v1.29 closeout + audit - Discussion Log

> **Audit trail only.** Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-09-06
**Phase:** 95-v1.28 SHIP 收口 + v1.29 closeout + audit
**Decision mode:** 自主裁定（用户 2026-09-06 预授权「全部自主运行，不要因为任何情况停止，一直跑到里程碑结束」——discuss 交互环节由 Claude 依先例与证据代行，全部决策透明记录）

---

## 侦察发现（决策的证据基础）

1. **CLOSEOUT-01 实际已完成** — MILESTONES.md:32-53 v1.28 段完整（SHIPPED + 45.13% 理由 + 24.87pp + Phase 88 备选 + 归档位置），2026-09-04 收口产物
2. **CLOSEOUT-02 实际已完成** — PROJECT.md:35 已标 SHIPPED+ARCHIVED；frontend-coverage 目录已登记「保留作历史」
3. **CLOSEOUT-03 是主体** — 完整 gate + 7 项确认 + v1.29-MILESTONE-AUDIT.md
4. **Phase 94 遗留 2 项 HUMAN-UAT** — type-check gate 空转（真实配置全仓仅 1 处错误，94 verifier 探针实证）+ 两处外观级变化

## 灰色地带与裁定

### Q1: CLOSEOUT-01/02 如何处置？

| Option | Description | Selected |
|--------|-------------|----------|
| 核对确认 (自主裁定) | 文档已存在，逐项核对 + 补漏 + 措辞校准（「写入」→「核对确认(已存在)」） | ✓ |
| 按 REQUIREMENTS 原文重写 | 重复添加已存在的 SHIPPED 段 = 文档重复 | |

**裁定：D-01 核对确认**（Phase 91/92/94 现实校准纪律先例；重复添加违反对账常识）

### Q2: frontend-coverage workstream 目录归档方式？

| Option | Description | Selected |
|--------|-------------|----------|
| 保留作历史 (自主裁定) | MILESTONES 已登记该决策，目录零移动，仅核对完整性 | ✓ |
| 迁移 .archive/ | 目录移动波及引用路径，MILESTONES 声明已否决 | |

**裁定：D-02 保留作历史**（REQUIREMENTS「或保留作历史」分支已满足）

### Q3: type-check gate 空转修不修？（94 HUMAN-UAT 项 1）

| Option | Description | Selected |
|--------|-------------|----------|
| 修复优先 + 降级授权边界 (自主裁定) | 修 tsconfig 检查面 + 修 useRestoreTask.ts:47；若暴露 >5 文件错误或需重设策略则降级为 audit 登记 + V130 候选 | ✓ |
| 仅登记 deferred | CLOSEOUT-03 的 type-check gate 保持恒真断言，audit 价值打折 | |

**裁定：D-03 修复优先**。理由：v1.29 是技术债治理里程碑，验证防线失真是核心技术债；94 verifier 实证全仓真实配置仅 1 处存量错误（成本可控）；不修则审计报告含一个无牙齿的 gate 断言。降级线保护修复不失控。

### Q4: 两处外观级变化（94 HUMAN-UAT 项 2）？

| Option | Description | Selected |
|--------|-------------|----------|
| 接受现状 (自主裁定) | 已登记 + 测试锁定 + 可逆（deferred 选项存在），UAT 关闭 | ✓ |
| 追加 ignoreContentDisposition 选项 | 新增代码面，超出 CLOSEOUT 文档相边界 | |

**裁定：D-04 接受现状**（外观级、无断言依赖旧值、94-03-SUMMARY deviation 6 已登记）

### Q5: v1.29-MILESTONE-AUDIT.md 口径？

**裁定：D-07 仿 v1.27 模板**（`.planning/milestones/` 落点 + 六段结构）+ 新增「type-check gate 空转发现→处置」章节（v1.29 独有审计价值项）。备选「全新结构」被否（无理由偏离四期先例）。

### Q6: SC-5「milestone SHIPPED 状态设置」深度？

**裁定：D-09 = MILESTONES.md + PROJECT.md 文档标记**；完整 archive 流程（phases 目录迁移）留给 `/gsd-complete-milestone` 独立工作流。备选「本相内跑完整 archive」被否（工作流边界：archive 属 milestone 生命周期管理非 phase 交付物）。

### Q7: gate 跑批口径？

**裁定：D-06 七项清单**（go build / go test / npm type-check[修后真检查或降级注记] / lint 0 errors / npm test / 双 coverage gate 同 CI 口径），逐项记录命令+exit code+关键数字进 audit。

## Claude's Discretion

- tsconfig 修复具体方案（references vs -p vs files）— researcher 以 tsc 探针实证选型
- 降级线「>5 文件」判定时机、audit 章节详略、gate 顺序与 commit 粒度、95-01/02 任务切分微调、七项行动证据抽查深度、HUMAN-UAT 状态注记格式

## Deferred Ideas

- milestone 完整 archive（/gsd-complete-milestone 另行）
- V130-CANDIDATES 处置（v1.30 输入，本相仅转记确认）
- operlog.exclude_paths todo（v1.29 D-05 deferred 维持）
- frontend-coverage .archive/ 迁移（MILESTONES 声明已否决）
