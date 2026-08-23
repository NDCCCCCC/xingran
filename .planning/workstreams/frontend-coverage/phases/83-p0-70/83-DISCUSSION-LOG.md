# Phase 83: P0 基建层全清 ≥70% - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-24
**Phase:** 83-P0 基建层全清 ≥70%
**Areas discussed:** CR-01 gate 修复前置, Harness 形态与位置, Mock 策略, Plan 切分与波次

---

## CR-01 gate 修复前置

| Option | Description | Selected |
|--------|-------------|----------|
| 83 首个 plan（推荐） | 作为 Phase 83 的 plan 0（wave 1 首位）修复，走完整 plan 流程（含验证 PR）。与 harness 落地同 phase，时序上最安全 | ✓ |
| 先 gsd-quick 修 | 立即用 /gsd-quick 单独修，不占 83 的 plan 位。快但验证强度低（无真实 PR 验证） | |
| 推迟规避 | 不动 gate，Phase 83 期间所有 PR 避开 src/test/ 与白名单文件。技术债后置，约束后续所有 PR | |

**修复范围：**

| Option | Description | Selected |
|--------|-------------|----------|
| 仅 CR-01 | 只修 pathspec 镜像 exclude。最小变更 | |
| CR-01+WR-01 | 两个 gate 失效通道同修 | |
| 全部四项 | CR-01 + WR-01（fail-closed exit 2）+ WR-02（漂移检测锚定）+ WR-03（floors 数值校验） | ✓ |

**验证方式：**

| Option | Description | Selected |
|--------|-------------|----------|
| 本地+试验 PR（推荐） | 空树合成基线复现修复前后行为 + 含 src/test//.d.ts/白名单变更的试验 PR 验证真实 CI（不 merge）；顺带首次真实触发 GOV-04 join 主路径（IN-06 缺口补齐） | ✓ |
| 仅本地验证 | 只本地合成基线验证。快但 IN-06 缺口保留 | |

**User's choice:** 83 首个 plan / 全部四项 / 本地+试验 PR
**Notes:** verifier 在 82-VERIFICATION.md 已建议 Phase 83 前置修复（harness 落 src/test/ 的首个 PR 必踩）。

---

## Harness 形态与位置

**沉淀节奏：**

| Option | Description | Selected |
|--------|-------------|----------|
| 薄三件套 | 首 plan 只建 renderWithProviders/mockApi/message mock，后续按需加 | |
| 全家桶 | 一次建齐所有工具（含 jsdom 补丁等），P1/P2 直接用 | |
| P0 尾声定稿（推荐） | P0 各 plan 期间观察实证重复需求，P0 尾声定稿，84 开工前备好 | ✓ |

**Provider 注入策略：**

| Option | Description | Selected |
|--------|-------------|----------|
| 全量注入 | Router + ConfigProvider + 全部 9 个 store。开箱即用但状态泄漏风险 | |
| 按需注入（推荐） | 默认 Router + ConfigProvider，stores 参数化 + 自动 reset（Zustand 官方模式） | ✓ |
| 不建 render 包装 | 只建 mockApi 与 message mock。最薄但 P1/P2 样板重复最多 | |

**mockApi 形态：**

| Option | Description | Selected |
|--------|-------------|----------|
| 端点工厂（推荐） | createApiMock(endpoint, response) 生成 spy；零新依赖，贴合 wrapped post/get 约定 | ✓ |
| MSW 网络层 | 拦截网络层保留 api.ts 真实加密链路。真实性最强但新依赖+复杂度高 | |
| vi.mock 模板 | 只提供复制粘贴片段。最简单但样板重复回到解放前 | |

**User's choice:** P0 尾声定稿 / 按需注入 / 端点工厂
**Notes:** harness 落 src/test/（已在 coverage.exclude 内，不占分母）；CR-01 修复后 diff gate 不误罚。

---

## Mock 策略

**api.ts mock 边界：**

| Option | Description | Selected |
|--------|-------------|----------|
| 双轨（推荐） | 业务层 vi.mock 整模块；api.ts 自身真实链路 + mock 加密层直测 | ✓ |
| axios 层 mock | 所有测试 mock axios adapter，api.ts 永远在链路里。重模板、慢 | |
| 全模块 mock | 所有测试 mock 模块，api.ts 不测。INFRA-01 主体白给 | |

**国密测试深度：**

| Option | Description | Selected |
|--------|-------------|----------|
| 真实算法+向量（推荐） | 加解密往返/篡密文报错/密钥边界 + 固定密文样本。零 mock | ✓ |
| 契约+抽样向量 | 只测调用契约，算法点测。覆盖快但深度中等 | |

**TokenManager 刷新循环：**

| Option | Description | Selected |
|--------|-------------|----------|
| fake timers（推荐） | vi.useFakeTimers + advanceTimersByTime 直达过期点。快且确定 | ✓ |
| 真实短 TTL | 毫秒级 TTL 自然过期。真实但慢且 flaky | |

**User's choice:** 双轨 / 真实算法+向量 / fake timers
**Notes:** api.ts 是 INFRA-01 最大单体（SM2+SM4 混合加密 + TokenManager 自动刷新 + 401 重试）。

---

## Plan 切分与波次

**Plan 切分：**

| Option | Description | Selected |
|--------|-------------|----------|
| 依赖层切（推荐） | plan0=CR-01；utils+lib（并行）→ hooks+store → 收尾+harness 定稿。依赖清晰，wave 内无文件交集 | ✓ |
| 按需求域 7 plans | INFRA-01~05 各一 plan + harness plan，单线串行。边界最清但时长最长 | |
| 均衡切块 | 按工作量切 4-5 个 plan 混目录。并行度最高但依赖顺序乱 | |

**ratchet bump 节奏：**

| Option | Description | Selected |
|--------|-------------|----------|
| 逐 plan bump（推荐） | 每 plan 完成即 bump 对应目录 floor（实测−0.5pp）+ 基线文档同 PR。回滚粒度细 | ✓ |
| phase 末尾一次 | 期间 floor 不动，末尾一次性 bump。CI 压力小但丢中间锁点 | |

**验收密度：**

| Option | Description | Selected |
|--------|-------------|----------|
| plan 级验收（推荐） | 每 plan verify 含 test:coverage + gate 目录断言 + 159 存量不回归。问题定位在 plan 内 | ✓ |
| phase 级验收 | plan 只跑相关测试，phase 末尾全量验一次。最快但发现晚 | |

**User's choice:** 依赖层切 / 逐 plan bump / plan 级验收
**Notes:** use_worktrees=false（项目配置），wave 内并行 plan 需注意工作树串行约束。

---

## Claude's Discretion

- 测试文件组织（同目录放置、describe/it 分组）——按现有 19 个测试文件模式
- CR-01 pathspec 镜像实现方式（保持单一真相源原则）
- plan 内文件测试优先级排序

## Deferred Ideas

- MSW 网络层 mock —— D-06 未采纳，P1/P2 出现真实拦截需求再评估
- CI 缓存/分片优化 —— 82 遗留 deferred（41s 余量充足）
- harness 全家桶扩展 —— 按 P1/P2 实际需求渐进增补
- E2E 测试层、视觉回归 —— REQUIREMENTS v2 候选
