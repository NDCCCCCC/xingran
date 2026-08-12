# Feature Research: 网络设备写命令 (Network Device Write Operations)

**Domain:** 网络运维管理系统 — 网络设备 Web 端配置下发 (SSH write)
**Researched:** 2026-07-06
**Milestone:** v1.19 网络设备写命令
**Confidence:** MEDIUM (基于 v1.18 已 ship 的采集基础 + XingRan-Next operlog/permission 既有模式)

## Executive Summary

v1.19 解决网络设备"读+写"闭环的"写"端:Web 端通过 SSH 直接对目标设备下发端口配置命令(启停/描述/dot1x), 成功后立即采集一次端口信息回填审计。功能定位是"受控的危险操作"——任何写操作都不可逆(在生产网络上), 因此 UX 必须以**预防误操作**和**完整可审计**为第一性。

**核心发现 (基于 v1.18 资产 + operlog 既有约束):**

1. **复用密度极高**: operlog (Phase 34 全覆盖)、DeviceInfoCollectionService (v1.18 异步队列)、permission namespace (现有 `network:port:query`/`network:command:execute`)、Scrapli (`huawei_vrp`/`hp_comware`/`ruijie_rjos`) 全部就位。v1.19 主要是**串起来**而非新建。
2. **写操作的本质是事务**: 单端口写 = `command_dispatch.send → write → re-collect → operlog`; 批量写 = `for port { 单端口写 } + fail-fast`。**改后采集**是 v1.18 留给 v1.19 的"挂载点"。
3. **反模式 (anti-feature) 集中在"自动化"**: 不做自动回滚、不做定时写、不做多用户并发仲裁——M&M (Modifying state) 操作的"半自动回滚"在生产网络是反模式, 与用户偏好一致 (见 [user-prefers-code-fixes-no-db-triggers.md]——根因修复走代码层, 禁用 DB TRIGGER 路线; 同样适用于写操作)。
4. **UX 重点是确认弹窗 + 实时反馈**: 端口表格的"修改管理员激活"按钮点击后, 必须**二次确认**(描述、影响端口列表、回退指引), 不能 inline toggle (因为是无后端的 SSH 命令)。
5. **批量执行的"半完成"语义**: MVP 锁定"失败即停 + 报告失败点"——不做 partial commit 也不做 automatic rollback, 这与用户对 MVP 的现实预期一致。

## Table Stakes

用户期望的基础功能。缺失这些 = 产品感觉"没写完"或"危险"。

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| **二次确认弹窗 (Confirm Modal)** | 写操作不可逆, 误操作会导致生产网络中断 | LOW | 复用 antd `Modal.confirm` + 自定义内容 (端口列表 + 目标状态), 必填"操作原因" |
| **操作结果回传 (Result Feedback)** | 用户必须立刻知道命令成功/失败, 看不到 SSH 输出 = 不知是否生效 | LOW | toast 提示 + 弹窗详情面板 (含原始 SSH output 截取) |
| **操作审计日志 (operlog)** | operlog 全覆盖是 XingRan-Next 强制约定, v1.19 写命令是高价值审计点 | LOW | 复用 `operlog.Record(RecordWithBody)`, 模块名"端口配置" + OperTypeStatus/Update |
| **权限控制 (per-port-write)** | 写操作必须有独立权限点, 防止普通用户误碰 | LOW | 新增 `NetworkPortWrite = "network:port:write"`, 加在 port_router 写接口上 |
| **单端口精确操作 (Single Port Op)** | 表格行级操作按钮 (admin status / description / dot1x), 用户最常用 | MEDIUM | 复用 v1.18 端口表格, 新增行级"操作"列 + 弹窗; 命令模板按 vendor 区分 |
| **多端口批量操作 (Batch Op)** | 运维常见场景: 同设备 8/16/24 端口同时改描述; 减少重复点击 | MEDIUM | 表格多选 + 弹窗 + 串行执行 + 进度条 + 失败即停 |
| **改后采集触发 (Re-collect on Success)** | 写操作后必须验证生效, 不能仅靠 SSH output 字符串判断 | LOW | 复用 `DeviceInfoCollectionService.Enqueue(deviceID)` (v1.18 已 ship, `Enqueue` 在 133 行) |
| **失败点定位 (Failure Pointing)** | 批量执行时必须能定位到具体哪个端口失败 + 原因 | LOW | 批量结果含 `failed: {portName, reason}` 数组, 前端显示 |
| **前置状态校验 (Pre-state Check)** | 避免重复命令 (如已经是 shutdown 再下发 shutdown) | LOW | 服务端读 `DevicePortStatus.admin_status`/`dot1x_enabled`, 已是目标态则跳过或提示 |
| **超时/中断可见 (Timeout Visible)** | SSH 命令可能 hang, 用户必须知道何时放弃 | MEDIUM | 默认 30s per-port 超时, 超时返回明确错误码; UI 显示倒计时 |

## Differentiators

区分产品、给用户"超额价值"的功能。MVP 可选, 后续 phase 增强。

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| **命令预览 (Command Preview)** | 提前看到 vendor→CLI 命令原文, 高级用户验证正确性 | LOW | `vendor=huawei_vrp + op=shutdown` → `interface GigabitEthernet0/0/1\n shutdown`, 在弹窗内折叠显示 |
| **操作历史查看 (Operation History per Port)** | 追溯"这个端口什么时候被谁 shutdown" | MEDIUM | operlog.module="端口配置" + 弹窗底部"查看历史" 跳转 sys_oper_log 过滤 |
| **多设备批量 (Multi-device Batch)** | 跨设备同操作 (如 5 台接入交换机同时改 dot1x) | HIGH | 复用 `CommandDispatchService.Dispatch` 的多设备能力, 但端口维度匹配更复杂, 推后到 v1.20+ |
| **保存为模板 (Save as Template)** | 多次复用相同命令序列 | HIGH | 复用现有 `NetworkTemplate` 表, 推断"模板化"语义, 推后 |
| **dry-run 模式 (Dry-run)** | 不实际下发, 仅显示"将要执行什么" | MEDIUM | 复用"命令预览" 扩展, 加 `--dry-run` 标志 |
| **操作撤销/回滚 (Rollback)** | 误操作 5 分钟内可反向 | HIGH | 需记录"反向命令"(`shutdown` ↔ `undo shutdown`), 强一致性问题多, **MVP 锁定不做** |
| **dot1x 鉴权方式联动 (dot1x auth-method)** | enable dot1x 时同时配置 eap/radius 等 | HIGH | 涉及 radius 配置, 超出 v1.19 scope |
| **批量冲突解决 (Batch Conflict)** | 部分端口已是目标态, 哪些跳过的可视化 | MEDIUM | 服务端返回 `skipped` 数组, 前端用 Tag 标识"已跳过" |
| **设备不可达预检 (Reachability Pre-check)** | 操作前先 ping/ssh-test 设备, 失败提前报告 | MEDIUM | 复用 `DeviceExecutor.ExecuteOnDevice("ping", ...)` 模式, 1s 内短路 |

## Anti-Features

明确不应构建的功能——通常被用户提出来但实际上制造问题。

| Anti-Feature | Why Requested | Why Problematic | What to Do Instead |
|--------------|---------------|-----------------|-------------------|
| **自动回滚 (Automatic Rollback)** | "我误 shutdown 了, 帮我恢复" | 写操作的反向不是简单的命令反转, 可能涉及 `display this` 解析 + 配置差异 + 中间状态; 5 分钟内人工回退更可靠 | 操作前显示"回退命令" (`undo shutdown`) 给用户手动复制 |
| **定时写操作 (Scheduled Writes)** | "凌晨 3 点批量改 50 台" | 凌晨出事没人响应, 调度器 + operlog + 失败通知都是新工程量 | 用现成的"工作单/工单"系统承载异步任务, 写命令仅限同步 UI 操作 |
| **多用户并发写仲裁 (Concurrent Write Arbitration)** | "我俩同时改一个端口" | 实现涉及分布式锁 + last-writer-wins 策略, 复杂度远超 v1.19 scope | operlog 自然留下时序, 冲突留人工识别; 显示"上次操作: 5 分钟前 alice 改过" 即可 |
| **写命令中转执行 (Local Buffer & Replay)** | "网络断了命令要重试" | 状态变化类操作 (`shutdown`) 重试有副作用风险; idempotent (description) 才有意义 | 仅 description 可重试, 其余一次性 |
| **跨厂商同命令 (Cross-vendor Unified Command)** | "我不要记每家厂商命令" | 厂商命令差异巨大 (Huawei `shutdown` vs Cisco `shutdown` vs H3C 同名), 抽象会丢精度 | vendor→template map 硬编码, 见 PROJECT.md 锁定决策 |
| **实时写命令流 (Live Command Stream)** | "我想看 SSH session 实时" | scrapligo 是 request/response 模型, 不是 stream; 投入产出比低 | 操作后展示完整 output buffer 即可 |
| **AI 智能推荐 (AI Recommend)** | "我描述意图, AI 生成命令" | 写操作不允许"猜", 必须确定性命令 | 锁定"按钮 + 参数表单"模式, 不接 LLM |
| **改前快照/回滚点 (Pre-change Snapshot)** | "操作前自动 backup 配置" | backup 系统已存在, 但与写操作联动是另一 phase 的 scope | 提示用户"如需备份, 前往 配置备份 页" |
| **写入中的取消 (Cancel Mid-execution)** | "我点错了想停" | SSH 命令下发到设备侧已不可中断 (设备正在执行), 中断只对"未下发"有效 | 命令下发前 1s 内可取消, 已下发等待结果 |

## Feature Dependencies

```
v1.18 DeviceInfoCollectionService.Enqueue(deviceID)
  └──required by──→ 改后采集触发 (write op success → enqueue re-collect)

operlog.Record / RecordWithBody (Phase 34)
  └──required by──→ 所有写端点 (operlog.module="端口配置", OperType=Status|Update)

DeviceExecutor.ExecuteOnDevice (internal/device/executor.go)
  └──required by──→ 写命令服务 (SSH command send)

ScrapliWrapper vendor mapping (huawei_vrp/hp_comware/ruijie_rjos)
  └──required by──→ vendor→command template map

PermissionCode 命名空间 (pkg/permission/config.go:186)
  ├──NetworkPortQuery  ──existing──→ 端口读取
  └──NetworkPortWrite  ──NEW───────→ 端口写命令 (v1.19)

v1.18 sys_device_port_status 表
  ├──admin_status ──required by──→ 前置状态校验
  ├──description ──updated by──→ description 写操作
  └──dot1x_enabled ──updated by──→ dot1x 写操作

PortStatusHandler /useTableManager (前端 v1.18)
  └──enhanced by──→ 批量多选 + 行级操作按钮 + 弹窗
```

**关键依赖:**

1. **v1.18 DeviceInfoCollectionService.Enqueue** (已 ship): 写命令成功后调用, 触发"改后采集"——这是 v1.19 闭环的核心。
2. **Phase 34 operlog 全覆盖**: 所有写端点必须 `operlog.Record/RecordWithBody`, 不能漏; 新建"端口配置"模块中文名, 与 `OperTypeStatus`(状态变更) / `OperTypeUpdate`(修改) 对应。
3. **vendor→command template map**: 硬编码 map (`map[Vendor]map[Op]Command` + 端口名占位符), 三厂商覆盖; Cisco 推后。
4. **网络设备写权限独立命名空间**: `network:port:write` 与 `network:port:query` 分离, 避免运维普通用户误碰; 写入 `pkg/permission/config.go` 后端 + 同步前端按钮 disable。
5. **port_status 表的 device_id 是 varchar 还是 uuid**: 历史上 `ops_info_points.*_id` 是 varchar 但 `sys_device_port_status` 走标准 UUID; 写命令的 device_id/port_id 来源于 `sys_device_port_status` (标准 UUID), 无 varchar 陷阱。
6. **"改后采集" 的 op 主路径在 v1.18 异步队列**: 必须用 `Enqueue` 不是直接 `CollectOneDevice`, 否则阻塞请求线程 (v1.18 worker=5, queue=1000, 见 device_info_collection_service.go:52)。

## MVP Definition (v1.19 锁定)

### Launch With (v1.19 必须实现)

最小可行产品——验证"Web 端写命令 + 改后采集 + 审计"闭环。

- [x] **后端: 写命令服务 (WriteCommandService)**
  - 路由: `POST /network/ports/write` (单端口) + `POST /network/ports/batch-write` (批量)
  - 入参: `deviceId`, `portIds[]`, `op` (shutdown/undo_shutdown/description/dot1x_on/dot1x_off), `params` (op-specific)
  - 内部流程: validate → load port (admin_status/description/dot1x_enabled) → select vendor template → SSH send → re-collect enqueue → operlog.Record
  - Why essential: 没有这个 service, 前端按钮无后端支撑

- [x] **vendor→command template map**
  - Huawei VRP: `interface {iface}\n shutdown` / `undo shutdown` / `description {text}` / `dot1x enable` / `undo dot1x enable`
  - H3C Comware: 同 Huawei 语法 (语法相同, 平台 YAML 不同)
  - 锐捷 RGOS: `interface {iface}\n shutdown` / `no shutdown` / `description {text}` / `dot1x port-control auto` / `no dot1x port-control`
  - Why essential: 写命令的 vendor 差异是最大的实现风险, 必须 MVP 锁定三厂商

- [x] **前端: 行级操作按钮 + 弹窗**
  - 端口表格新增"操作"列: 含 [修改管理员激活] [修改描述] [修改 dot1x] 三个按钮
  - 弹窗组件: `<PortWriteModal op={op} ports={ports} />`
    - 必填"操作原因" (5-200 字符)
    - 显示"影响端口列表" + "目标状态"
    - 显示"回退命令" (折叠, 高级用户)
    - "确认执行" 按钮 (二次确认)
  - Why essential: 这是用户的"写命令入口", 没有这个整个功能不可用

- [x] **前端: 批量操作弹窗 (Batch Port Write)**
  - 表格多选 + 顶部 "批量操作" 下拉: shutdown / undo shutdown / 改描述 / 启用 dot1x / 停用 dot1x
  - 弹窗内: 选中端口数 / 设备分布 / 操作原因
  - 执行时: 显示进度条 (X/Y 端口, Y 已成功, 0 已失败) + 失败点列表
  - fail-fast: 第一个失败即停, 已成功的保留 (operlog 标 success)
  - Why essential: 批量是运维核心场景, 单端口太慢

- [x] **前端: 结果回显 (Result Toast + Detail Modal)**
  - toast: 成功/部分成功/失败 三档
  - 失败时弹出"失败详情" Modal, 含: 失败端口 + 设备 + 错误信息 + 原始 SSH output 截取
  - 成功时显示"操作 ID, 可在操作日志中查看" 链接
  - Why essential: 写操作没有 result = 不可信

- [x] **权限: `network:port:write` 命名空间**
  - 后端: `pkg/permission/config.go` 新增 `NetworkPortWrite = "network:port:write"`
  - 路由: port write/batch-write 端点用 `RequirePermissions([]string{"network:port:write"}, core)`
  - 前端: useAuthStore / 按钮 disable 显隐控制
  - migration_NNN seed: 把权限加到 sys_menu 的 "端口配置" 父菜单上, 并把该菜单关联 admin 角色
  - Why essential: 写操作不能无门槛

- [x] **operlog 集成 (强制)**
  - 所有 write/batch-write 端点 success path 末尾 `operlog.Record(...)` 前调用, 模块名"端口配置"
  - operParam 用 `RecordWithBody` (含端口 ID 列表 + 操作类型 + 操作原因)
  - 失败时 `operlog.WithStatus(1) + WithErrorMsg(err.Error())`
  - Why essential: operlog 是 XingRan-Next 强制约定, Phase 34 已锁

- [x] **改后采集触发 (Re-collect)**
  - write 成功后, 服务端调用 `core.DeviceInfoCollectionService.Enqueue(deviceID)`
  - 前端不感知, 1-2s 后端口状态自动刷新
  - Why essential: 写操作必须"自我验证", 没有这一步 = 用户不知命令是否生效

### Add After Validation (v1.19.x 后续 phase)

- [ ] **失败点自动重试 (Retry Failed Ports)** — 触发条件: 用户反馈"批量失败但想单条重试"
- [ ] **写操作历史页 (Write Operation History)** — 触发条件: 审计/合规需求
- [ ] **定时写任务 (Scheduled Write)** — 触发条件: 业务方有凌晨批量需求, 但必须先有"工作单系统"承载
- [ ] **多设备批量 (Multi-device Batch)** — 触发条件: 跨设备同操作场景频繁出现
- [ ] **dot1x auth-method 联动 (RADIUS)** — 触发条件: dot1x 全面铺开后, 鉴权方式成为下一步
- [ ] **Cisco 厂商支持** — 触发条件: 用户接入 Cisco 设备
- [ ] **操作前配置 backup (Pre-change Backup)** — 触发条件: 操作失误率统计显示 backup 能降低回退时间
- [ ] **rollback 操作 (Reverse Operation)** — 触发条件: 误操作率统计支持投入产出

### Future Consideration (v2+)

- [ ] **AI 辅助命令生成** — 写操作不允许 LLM 介入, 推后到 NMS 范畴
- [ ] **跨厂商命令翻译** — 复杂度高, 价值存疑
- [ ] **实时 SSH stream** — scrapligo 模型不支持, 需换 transport
- [ ] **多用户协同 (lock + last-writer-wins)** — 工程量巨大, 推后

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| 后端写命令服务 (WriteCommandService) | HIGH | MEDIUM | **P1** |
| vendor→command template map (Huawei/H3C/锐捷) | HIGH | LOW | **P1** |
| 行级操作按钮 + 弹窗 (前端) | HIGH | MEDIUM | **P1** |
| 批量操作弹窗 (前端) | HIGH | MEDIUM | **P1** |
| 结果回显 toast + 详情 | HIGH | LOW | **P1** |
| `network:port:write` 权限 | HIGH | LOW | **P1** |
| operlog 集成 | HIGH (合规) | LOW | **P1** |
| 改后采集触发 | HIGH | LOW | **P1** |
| 二次确认弹窗 | HIGH | LOW | **P1** |
| 前置状态校验 | MEDIUM | LOW | **P1** |
| 失败点定位 | MEDIUM | LOW | **P1** |
| 命令预览 (vendor→CLI) | MEDIUM | LOW | P2 |
| 操作历史链接 | MEDIUM | LOW | P2 |
| 设备不可达预检 | MEDIUM | MEDIUM | P2 |
| 批量冲突解决 (skip 已目标态) | MEDIUM | LOW | P2 |
| dry-run 模式 | MEDIUM | MEDIUM | P2 |
| 多设备批量 | MEDIUM | HIGH | P3 (v1.20+) |
| 保存为模板 | LOW | HIGH | P3 (v1.20+) |
| 自动回滚 | HIGH (用户提) | HIGH | **Anti-feature (MVP)** |
| 定时写 | LOW | HIGH | P3 (v1.21+) |
| 多用户并发仲裁 | LOW | HIGH | **Anti-feature (P3+)** |

**Priority key:**
- **P1**: Must have for v1.19 launch
- **P2**: Should have, add in v1.19.x patch release
- **P3**: Nice to have, future phase

## UX Wireframe Notes

### 单端口操作弹窗 (`<PortWriteModal />`)

```
┌─────────────────────────────────────────────────────┐
│ 修改端口描述 (8)                                     │
├─────────────────────────────────────────────────────┤
│ 目标端口:                                            │
│   Huawei-S5700-01 › GigabitEthernet0/0/1 (8)         │
│                                                     │
│ 操作原因 * (5-200字符):                              │
│   ┌─────────────────────────────────────────────┐   │
│   │ 接入A座3F工位, 2026-Q3扩容                    │   │
│   └─────────────────────────────────────────────┘   │
│                                                     │
│ 目标描述:                                            │
│   ┌─────────────────────────────────────────────┐   │
│   │ ToA-3F-WS-042                                │   │
│   └─────────────────────────────────────────────┘   │
│                                                     │
│ ▶ 命令预览 (展开)                                    │
│   interface GigabitEthernet0/0/1                     │
│    description ToA-3F-WS-042                        │
│                                                     │
│ 取消                                  [确认执行]    │
└─────────────────────────────────────────────────────┘
```

### 批量操作弹窗 (`<BatchPortWriteModal />`)

```
┌─────────────────────────────────────────────────────┐
│ 批量 shutdown 端口 (12)                              │
├─────────────────────────────────────────────────────┤
│ 影响范围:                                            │
│   1 台设备 / 12 个端口 / 0 个已目标态 (跳过)        │
│   端口列表: [GE0/0/1, GE0/0/2, ..., GE0/0/12]       │
│                                                     │
│ 操作原因 *:                                          │
│   ┌─────────────────────────────────────────────┐   │
│   │ 2026-Q3 网络整改, 临时下线                    │   │
│   └─────────────────────────────────────────────┘   │
│                                                     │
│ 执行策略: ○ 并行(快) ● 串行(安全, 失败即停)         │
│                                                     │
│ ⚠ 警告: 此操作将立即影响 12 个端口的连通性            │
│                                                     │
│ 取消                                  [确认执行]    │
└─────────────────────────────────────────────────────┘
```

### 批量执行结果 (progress + result)

```
执行中 (3/12):     [████████░░░░░░░░░░░░] 25%  3 成功 0 失败
完成:            [████████████████████] 100% 12 成功 0 失败
                  ┌── 操作 ID: 770e8400-e29b... ──┐
                  │ 查看操作日志 →                  │
                  └─────────────────────────────────┘

完成:            [████████████████████] 100% 8 成功 4 失败
                  失败点:
                    - GE0/0/5: 设备返回 "Permission denied"  
                      原始输出: % ...
                    - GE0/0/8: 命令超时 (30s)
                    ...
                  [重试失败] [关闭]
```

## Per-Operation UX Specifics

### shutdown / undo shutdown (互斥)

- **弹窗设计**: 显示"当前状态: UP" → "目标状态: DOWN" (用箭头可视化)
- **状态校验**: 如果当前已是目标态, 弹窗顶部黄色提示"该端口已是目标状态, 是否仍要执行?"
- **回退命令**: `undo shutdown` (Huawei/H3C) / `no shutdown` (Ruijie) —— 自动显示
- **dot1x 影响提示**: "⚠ 此操作会终止该端口上的 dot1x 会话" (如有 dot1x 启用)

### description

- **字符校验**: 1-80 字符 (Huawei limit), 禁止 `?"<>|` (Windows 文件名非法字符)
- **占位符**: `{iface}` 替换为端口名 (vendor 不同端口名风格: `GigabitEthernet0/0/1` vs `GE0/0/1`)
- **历史回填**: 操作成功后 1-2s, 端口表格的"描述"列自动更新 (走改后采集)
- **特殊字符**: 双引号 `"` 和单引号 `'` 需转义, 但 v1.19 MVP 锁定"仅 ASCII 字母数字 + -_/."

### dot1x enable / disable

- **弹窗设计**: 互斥 radio 按钮 "启用 dot1x 认证" / "停用 dot1x 认证"
- **关联提示**: "⚠ 启用 dot1x 需要设备上已配置 RADIUS 服务器 (本系统不验证)"
- **dot1x auth-method**: MVP 锁定不做 (用户操作 dot1x 启停, 不动 auth-method 字段)
- **回退命令**: `undo dot1x enable` / `no dot1x port-control`

## Edge Cases & Error Handling

| Edge Case | 触发场景 | 处理策略 |
|-----------|---------|---------|
| **设备不可达** | SSH 连接失败 / 网络中断 | 返回 503 + 错误码 `DEVICE_UNREACHABLE`; 前端显示"设备离线, 请检查网络或重试" |
| **凭据失效** | sys_auth_credential 过期/被改 | 返回 401 + 错误码 `CREDENTIAL_INVALID`; 前端提示"凭据失效, 请联系管理员更新" |
| **命令被设备拒绝** | "Permission denied" / "Error: interface not exist" | 返回 422 + 错误码 `COMMAND_REJECTED`; 弹窗显示原始 output (前 200 字符) |
| **命令超时** | SSH 命令 hang > 30s | 强制 cancel, 返回 504 + 错误码 `COMMAND_TIMEOUT`; 弹窗显示"命令超时, 设备可能仍在执行" |
| **批量中某端口失败** | 12 端口中第 5 个失败 | 串行模式: 立即停止, 报告 4 成功 + 5/6/7/8/9 失败 (5 失败点); 已成功的 operlog 标 success |
| **批量中设备切换** | 12 端口分布在 2 台设备 | 每台设备单独 SSH, 设备级 fail-fast (per-device); 不跨设备 fail-fast (避免"半台设备改完"语义不清) |
| **重复操作** | 同一端口连续点击 2 次 shutdown | 第二次: 前置状态校验拦截, 提示"该端口已是目标状态" |
| **并发用户** | alice 和 bob 同时改同一端口 | 无分布式锁, 后执行者覆盖前执行者; operlog 自然留下时序; 前端按钮不 disable (无实现成本) |
| **改后采集失败** | SSH 成功, 但 enqueue re-collect 时设备又离线 | 写操作仍标 success (命令已下发), 改后采集留待下一次 cron 周期 |
| **操作原因过长** | > 200 字符 | 前端 form validation 阻止提交; 后端 binding 兜底 |
| **空设备/空端口选择** | 多选 0 行 | 前端按钮 disabled; 后端 binding required 兜底 |
| **vendor 不在支持列表** | Cisco 设备 | 后端硬编码 map miss → 500 + 错误码 `VENDOR_UNSUPPORTED`; 前端提示"Cisco 设备暂不支持" |

## Audit Log Integration

### operlog 记录规范

| 字段 | 值 | 备注 |
|------|----|------|
| `module` (中文) | "端口配置" | 新模块名, v1.19 首次出现 |
| `operType` | `OperTypeStatus`(shutdown/undo_shutdown/dot1x) 或 `OperTypeUpdate`(description) | 25 常量集中已有 (Phase 34) |
| `operParam` | JSON: `{deviceId, portIds, op, reason, targetValue?}` | 用 `RecordWithBody` 自动读取并 masked (含 reason) |
| `errorMsg` | 失败时填 SSH error / "COMMAND_TIMEOUT" | 用 `WithErrorMsg` |
| `status` | 0=success / 1=failure | 用 `WithStatus` |
| `jsonResult` | `{executionId, affectedPorts, failedPorts, sshOutputTruncated}` | success path 填 |

### 前端 operlog 跳转

- 成功 toast 中嵌入"查看操作日志"链接 → `system/operlog?module=端口配置&startTime=...&endTime=...`
- 复用现有 sys_oper_log 页面 (前端已有, Phase 34)

## Integration Points

### 复用现有组件 (REUSE-AUDIT)

| 组件 | 用途 | 位置 |
|------|------|------|
| **operlog.Record / RecordWithBody** | 写操作审计 | `internal/utils/operlog/operlog.go` |
| **DeviceInfoCollectionService.Enqueue** | 改后采集触发 | `internal/services/device_info_collection_service.go:133` |
| **DeviceExecutor.ExecuteOnDevice** | SSH 命令下发 | `internal/device/executor.go:52` |
| **ScrapliWrapper vendor mapping** | 厂商 SSH 平台 | `internal/device/scrapli_wrapper.go:68-75` |
| **PermissionCode 命名空间** | 权限点定义 | `pkg/permission/config.go:186-197` |
| **RequirePermissions middleware** | 路由权限校验 | `pkg/middleware/permission.go` |
| **networkApi.ts** | 前端 API 封装 | `xingran-react-frontend/src/lib/api/networkApi.ts` |
| **useTableManager hook** | 端口表格 + 多选 + 排序 | `src/hooks/useTableManager.ts` |
| **withErrorHandling** | 错误 toast 统一处理 | `src/utils/errorHandler.ts` |
| **antd Modal.confirm** | 二次确认弹窗 | antd 内置 |
| **operlog.module 常量** | sys_oper_log 模块名查询 | operlog.go 注释 |

### 新增服务 (NEW)

| 服务 | 职责 | 位置 |
|------|------|------|
| **PortWriteService** | 写命令业务逻辑 (validate + dispatch + re-collect) | `internal/services/portcollection/write.go` (新文件) |
| **VendorCommandTemplate** | vendor→command 模板 map | `internal/services/portcollection/vendor_templates.go` (新文件) |
| **PortWriteHandler** | HTTP handler | `internal/api/v1/network/port_write_handler.go` (新文件) |
| **PortWriteModal** | 单端口操作弹窗 | `src/pages/network/ports/modals/PortWriteModal.tsx` |
| **BatchPortWriteModal** | 批量操作弹窗 | `src/pages/network/ports/modals/BatchPortWriteModal.tsx` |
| **portWriteApi** | 前端 API 封装 | `src/lib/api/networkApi.ts` (扩展) |

### 数据库

**v1.19 不需要新建表**——写操作的数据载体是 `sys_device_port_status` (description/admin_status/dot1x_enabled) 现有字段; operlog 走现有 `sys_oper_log` 表。

**migration 需求:**
- migration_NNN: seed `network:port:write` 权限点到 `sys_menu` (操作日志父菜单的子菜单)
- 角色授权: 把 `network:port:write` 关联到 admin / 运维主管 (不直接给所有"运维"角色, 避免误操作)

## Comparison with Existing System

| 维度 | v1.18 (read) | v1.19 (write) | 关键差异 |
|------|--------------|---------------|----------|
| **操作方向** | SSH get (read-only) | SSH send-config (mutating) | 写操作有副作用 |
| **失败语义** | 采集失败 = 数据缺失, 可重试 | 写失败 = 设备状态未知, 不能盲目重试 | "已部分成功" 难处理 |
| **审计强度** | 采集是后台任务, 不写 operlog | 写操作是高价值审计点, 必须写 operlog | operlog 强约定 |
| **UX 要求** | 列表/详情展示, 无二次确认 | 必须二次确认, 必须有回退指引 | 写操作不能 inline |
| **并发安全** | 多 worker 采集无状态冲突 | 同一端口并发写 = last-writer-wins, 无保护 | operlog 留时序, 不阻塞 UI |
| **错误回显** | "采集失败 N 台" | "端口 X 失败: <原因>" | 粒度要求更高 |
| **权限粒度** | `network:port:query` | `network:port:write` (独立命名空间) | 写权限必须分离 |

## 风险与缓解

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| **生产网络被误 shutdown** | 端口 down, 业务中断 | 二次确认弹窗 + 必填"操作原因" + operlog 全覆盖 + 改后采集验证 |
| **vendor 命令差异遗漏** | 某厂商命令不工作 | v1.19 MVP 锁定三厂商 (Huawei/H3C/锐捷), Cisco 推后; 真机 UAT 是唯一验证手段 |
| **批量执行卡住** | UI 永远 "执行中" | per-port timeout 30s, 全局 5min hard limit; 失败时强制返回结果 |
| **改后采集漏触发** | 用户看不到"已生效" | write 成功后 server-side Enqueue, 不依赖前端; Enqueue 失败不影响 write operlog 标 success (避免审计 noise) |
| **operlog 漏写** | 审计缺口 | 写端点用 `operlog.Record` 强制约定 (Phase 34 全覆盖) + regression test 锁定 |
| **权限未授权 (admin 角色无权)** | 用户看不见按钮 | migration_NNN seed 把 `network:port:write` 关联到 admin 角色 |
| **前端按钮 disable 与后端权限不同步** | 灰显的按钮实际可点 | 复用 Phase 34 middleware; 前端 useAuthStore 查权限, 与后端 `RequirePermissions` 一致 |
| **端口表格数据陈旧** | 改后采集未触发 | `Enqueue` 异步, 1-2s 后接口刷新; UI 显示"数据采集中" loading 状态 |
| **批量操作时部分端口已被另一用户改** | 后写的覆盖前写的 | operlog 留时序, 不阻止; 显示"操作后状态" 列供用户自查 |

## Open Questions for Phase Planning

1. **写操作的菜单归属**: 是放在"网络设备" → "端口状态" 子菜单下, 还是单独"端口配置" 一级菜单? 取决于 v1.19 是否将"写命令"视为 portcollection 的扩展还是独立子模块。
2. **批量上限**: 单次批量 50 端口? 100 端口? 过大可能超时, 过小用户体验差。建议 100 上限, 超限前端拦截。
3. **操作原因是否强制模板**: 是自由文本, 还是下拉"维护/扩容/整改/事故"几个选项? MVP 自由文本, v1.19.x 改下拉。
4. **改后采集的失败回显**: re-collect 失败是否阻断 operlog success? 锁定不阻断, 改后采集是"附加验证" 而非"前置条件"。
5. **写操作的 timeout 默认值**: per-port 30s 是否合理? 某些命令 (如锐捷 dot1x 同步) 可能要 60s, 但 MVP 30s 够覆盖大部分场景。
6. **写命令是否需要审批流**: 普通用户写 / 主管审批? MVP 锁定无审批, 走权限分离 (运维主管/网工持 `network:port:write`); 审批流推后到 v1.20+。
7. **写命令的国际化 (i18n)**: 模块中文名"端口配置" 是否要做英文版? MVP 仅中文。
8. **测试覆盖**: 单元测试 mock SSH session? 集成测试用真机沙箱? MVP 单元测试覆盖 vendor template map + service 层; 真机 UAT 推后到生产部署前。

## Sources

- [PROJECT.md v1.19 锁定决策](D:/code/ClaudeCode/xingran-go-backend/.planning/PROJECT.md) — HIGH
- [CLAUDE.md XingRan-Next 架构 + operlog 约定](D:/code/ClaudeCode/xingran-go-backend/CLAUDE.md) — HIGH
- [v1.18 网络设备组件序列号 调研](D:/code/ClaudeCode/xingran-go-backend/.planning/notes/260703-network-device-component-serials.md) — HIGH
- [operlog 强制约定 + 25 OperType 常量](D:/code/ClaudeCode/xingran-go-backend/internal/utils/operlog/operlog.go) — HIGH
- [DeviceInfoCollectionService.Enqueue](D:/code/ClaudeCode/xingran-go-backend/internal/services/device_info_collection_service.go:133) — HIGH
- [Scrapli vendor 平台映射](D:/code/ClaudeCode/xingran-go-backend/internal/device/scrapli_wrapper.go:68-75) — HIGH
- [PermissionCode 命名空间](D:/code/ClaudeCode/xingran-go-backend/pkg/permission/config.go:145-197) — HIGH
- [Port Status 现有 read API](D:/code/ClaudeCode/xingran-go-backend/internal/api/v1/network/port_handler.go) — HIGH
- [Port Status 现有前端页面](D:/code/ClaudeCode/xingran-go-backend/xingran-react-frontend/src/pages/network/ports/index.tsx) — HIGH
- [user-prefers-code-fixes-no-db-triggers.md (用户偏好)](D:/code/ClaudeCode/xingran-go-backend/.planning/notes/../notes/) — MEDIUM (MEMORY 提示)

## Research Confidence

| Area | Confidence | Notes |
|------|------------|-------|
| **核心功能 (Table Stakes)** | HIGH | operlog + permission + DeviceExecutor + Scrapli 全部 v1.18 已 ship, 复用清晰 |
| **vendor 命令模板** | MEDIUM | 基于 v1.18 调研结论 + 通用网络运维知识; 真机 UAT 需 site-visit |
| **UX 模式** | HIGH | XingRan-Next 既有二次确认/toast/operlog 跳转 模式, 直接复用 |
| **Anti-Features** | HIGH | 基于用户偏好 ("根因修复走代码层") + 生产网络常识, 锁定不做 |
| **operlog 集成** | HIGH | Phase 34 全覆盖约定 + 25 OperType 常量集, 模块名"端口配置" 与现有命名风格一致 |
| **批量执行语义** | MEDIUM | "失败即停 + 已成功保留" 是 MVP 决策; 是否做"全成功或全失败" 需 phase 规划时确认 |
| **dot1x auth-method 联动** | MEDIUM | MVP 锁定不做, 推后 v1.19.x; 但需 phase plan 留 hook |
| **多设备批量** | MEDIUM | 推后 v1.20+; v1.19 仅单设备内多端口 |
| **测试策略** | LOW | 单元测试覆盖 vendor template + service 校验; 真机 UAT 推后到部署前 |

## Gaps & Phase-Specific Research

**需要 Phase plan 时细化的:**

1. **vendor→command 模板的最终形态**: 硬编码 map vs 数据库模板? MVP 硬编码, v1.19.x 抽象到 NetworkTemplate 表
2. **批量失败时的 operlog 写法**: 是 1 个 batch operlog (OperTypeBatch) + N 个 sub-detail, 还是 N 个独立 operlog? 需 phase plan 选型
3. **改后采集的轮询方式**: 客户端轮询? WebSocket 推送? MVP 复用 v1.18 cron 节奏 (15min) 不够, 建议前端 3s 间隔轮询
4. **操作原因的存储位置**: 在 operlog.operParam 内 (JSON) 还是独立字段? MVP JSON 内嵌, 后续可提取为独立列
5. **写操作的撤销/回退命令存储**: 仅展示在弹窗, 还是落库到 sys_oper_log 便于自动化? MVP 仅展示
6. **鉴权服务器的强校验**: dot1x enable 时是否 ping radius server? MVP 不做, 仅文字提示

**可推后到 v1.19.x 或 v1.20+:**

- 多设备批量写 (跨设备)
- 保存为模板 (复用 NetworkTemplate)
- 自动回滚 (强一致性问题)
- 定时写 (工作单系统承载)
- AI 辅助命令生成 (LLM 介入写操作风险高)

---

*Feature research for: v1.19 网络设备写命令 (Network Device Write Operations)*
*Researched: 2026-07-06*
*Replaces: v1.5 FEATURES.md (MAC 地址历史数据管理)*
