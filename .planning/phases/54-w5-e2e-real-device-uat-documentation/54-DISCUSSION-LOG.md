# Phase 54: W5 — E2E + Real-Device UAT + Documentation - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-07
**Phase:** 54-w5-e2e-real-device-uat-documentation
**Areas discussed:** Mock SSH 技术方案, e2e 覆盖深度与层级, 文档更新范围, UAT 推迟 + Phase 55 协调

---

## 灰色地带选择（present_gray_areas）

| Area | Selected |
|------|----------|
| Mock SSH 技术方案 | ✓ |
| e2e 覆盖深度与层级 | ✓ |
| 文档更新范围 | ✓ |
| UAT 推迟 + Phase 55 协调 | ✓ |

**User's choice:** 全部 4 个 area 均选中讨论。

---

## Mock SSH 技术方案（SC#1）

| Option | Description | Selected |
|--------|-------------|----------|
| FileTransport + fixture | scrapligo 公开 transport.NewFileTransport() + 预录制 fixture 回放，跑真实 SendConfigs 全链路，CI 友好无端口，满足 SC#1 'in-process mock sshd' 语义 | ✓ |
| 自建 in-process sshd | golang.org/x/crypto/ssh 起 Go ssh server 模拟 config-mode 状态机，scrapligo 真 SSH 连，最真实但工作量最大 | |
| 扩展现有 mock + fake Conn | 让 mockDeviceExecutor 调 fn + fake PooledConnection，最经济但不测 scrapligo 字节解析层 | |

**User's choice:** FileTransport + fixture
**Notes:** discuss 前 scout 确认 scrapligo v1.4.0 无公开 Mock API（无 NewMock/WithMock），但 transport 包有公开 NewFileTransport()——这改变了选项 landscape，使 FileTransport 成为唯一能跑真实 scrapligo SendConfigs 的可行公开 API 方案。

---

## e2e 覆盖深度与层级（SC#1）— 层级

| Option | Description | Selected |
|--------|-------------|----------|
| service 层 | 直接调 PortWriteService + 注入 FileTransport DeviceExecutor，工作量小，验证 service 编排→SendConfigs→parseConfigError→BatchResult 全链路（补 Phase 51 fn 闭包漏洞） | ✓ |
| HTTP handler 层 | gin test engine 打通 router→middleware→6 handler→service→scrapligo，最接近用户路径但需 mock Core 全套依赖，Phase 52 handler 零测试基建 cold start | |
| 混合: service + 路由 | service 层 e2e 打通 scrapligo + HTTP 层轻量测路由注册 + 权限中间件 gating，中等工作量 | |

**User's choice:** service 层
**Notes:** SC#1 字面 "endpoint paths" 语义降级为 service 公开方法路径（ExecutePortWrite/ExecuteBatch）。HTTP handler 层 cold start 工作量过大，HTTP 契约正确性已由 Phase 52 落地 + Phase 53 wrapper 对齐保证。

---

## e2e 覆盖深度与层级（SC#1）— 场景覆盖

| Option | Description | Selected |
|--------|-------------|----------|
| happy + 错误路径 | 5 single + 1 batch happy + transport_error/device_rejected/fail-fast/skipped 4 类错误路径，1 厂商 Huawei，fixture ~6-8 个，补 SSH-02/Pitfall#1 漏洞 | ✓ |
| 仅 happy path | 5 single + 1 batch happy，1 厂商，fixture ~3 个，最快但 SSH-02 仍只单测覆盖 | |
| 完整: 3 厂商 + 错误 | happy + 错误 × 3 厂商，验证厂商命令差异，fixture ~18-24 个，脆性 ×3 | |

**User's choice:** happy + 错误路径
**Notes:** 厂商差异（华为/H3C VRP 同源 vs 锐捷 Cisco-style dot1x）留真机 UAT 验证。错误路径补 STATE.md Pitfall #1（SSH-02 transport vs device_rejected 区分）——Phase 51 mock 绕过了，e2e 必须补。

---

## 文档更新范围（SC#3）— 加密语义

| Option | Description | Selected |
|--------|-------------|----------|
| 保持加密 | 写端点保持 SM2+SM4 加密，SC#3 重解释为"确认正确加密+文档化"，不改 config | ✓ |
| 加入豁免列表 | 写端点加入 exclude_paths 不加密，需改 config + migration，敏感操作裸传降低安全性 | |

**User's choice:** 保持加密
**Notes:** discuss 前 scout 确认 config.yaml exclude_paths（line 91-99）不含 /network/ports/write/*，即写端点当前已加密。SC#3 字面 "no SM2+SM4 wrap on SSH-derived paths" 措辞误导（SSH 后端协议 vs HTTP 加密正交），实际写端点是敏感操作应保持加密。

---

## 文档更新范围（SC#5）— CHANGELOG 处理

| Option | Description | Selected |
|--------|-------------|----------|
| 新建 CHANGELOG.md | 项目根新建，v1.19 起记，独立于 gsd-doc-writer 生成的 README，避免生成器冲突 | ✓ |
| 合并进 README | README 加版本历史段，减少文件数，但 README 由生成器管理易被覆盖 | |
| 跳过只更新 MILESTONES | SC#5 CHANGELOG 降级，只更新 README 能力 + MILESTONES v1.19 条目 | |

**User's choice:** 新建 CHANGELOG.md
**Notes:** CHANGELOG.md 当前不存在；README.md head 有 `<!-- generated-by: gsd-doc-writer -->` 标记，手改版本段易被覆盖，独立 CHANGELOG 更安全。项目已 v1.19/39 phases，CHANGELOG 是合理治理。

---

## UAT 推迟 + Phase 55 协调（SC#4）— UAT 文件位置

| Option | Description | Selected |
|--------|-------------|----------|
| 54-HUMAN-UAT.md 放 phase 54 目录 | 产出者存放，文件名 phase 号 = 目录 phase 号，需同步 STATE.md deferred 表 50→54 | ✓ |
| 50-HUMAN-UAT.md 放 phase 50 目录 | 遵循 SC#4/STATE.md 字面，但 phase 50 是 W1 vendor template 非主交付，语义弱 | |

**User's choice:** 54-HUMAN-UAT.md 放 phase 54 目录
**Notes:** SC#4 字面路径 50-port-write-network-ports-planned/ 是占位名 + 命名漂移（实际 phase 50 目录是 50-w1-vendor-templates-unit-tests-vendor-action-command-map）。v1.18 先例 48-HUMAN-UAT.md 放 48 目录因 48 兼主交付+产出者；v1.19 拆 5 phase 后产出者是 54。

---

## UAT 推迟 + Phase 55 协调（SC#4）— Phase 55 WR-02 协调

| Option | Description | Selected |
|--------|-------------|----------|
| 加观察条目 | UAT 文档加"WR-02 观察 custom-reason 使用频率"条目，驱动 Phase 55 修复决策，兑现 STATE.md 声明的依赖闭环 | ✓ |
| 不加 | UAT 只记 6 项 SSH verification，WR-02 独立判断 | |

**User's choice:** 加观察条目
**Notes:** STATE.md 明确"Phase 55 WR-02 修复决策由 W5 UAT 观察"，本条目兑现该依赖。观察结果：custom-reason 高频→修，低频→wontfix。

---

## 完成确认

| Option | Description | Selected |
|--------|-------------|----------|
| 还有 gray area 要探索 | 还有未覆盖的实现决策（UAT 设备型号 / SC#6 回归范围 / fixture 来源等） | |
| 可以写 CONTEXT 了 | 4 个 area 决策已清晰，进入 write_context | ✓ |

**User's choice:** 可以写 CONTEXT 了

---

## Claude's Discretion

- **fixture 来源与格式**：优先复用 scrapligo transport/test-fixtures/ 改造，% Error 等错误场景手写补充（researcher 查 fixture 格式）
- **API 响应规范小节具体位置**：插在"批量操作响应"之后或"特殊场景响应"末尾，planner 按连贯性定
- **CHANGELOG 是否补 v1.18**：默认 v1.19 起记，planner 可选补 v1.18 一行
- **UAT automated_gates 清单**：复刻 48 模式列本 phase 实际跑过的闸门
- **DeviceExecutor FileTransport 注入点**：researcher 确认构造函数参数 vs functional option

## Deferred Ideas

- 真机 SSH 写命令验证 → 54-HUMAN-UAT.md site visit
- HTTP handler 层 e2e → v1.19.x+（cold start 基建）
- 3 厂商 e2e fixture 全覆盖 → v1.19.x+（脆性 ×3）
- 跨固件版本命令差异 → follow-up（现场验证）
- fixture 自动录制工具 → v1.19.x+
- BATCH-05 实时进度 → v1.19.x（53 已 deferred）
- sys_port_write_audit 详情 UI → v1.19.x+（53 已 deferred）

---

*Phase: 54-w5-e2e-real-device-uat-documentation*
*Discussion logged: 2026-07-07*
</content>
</invoke>
