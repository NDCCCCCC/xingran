# Phase 51: W2 — PortWriteService + Batch Orchestrator + Mock Tests - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-06
**Phase:** 51-w2-portwriteservice-batch-orchestrator-mock-tests
**Areas discussed:** SSH 连接生命周期, Pre-state 数据源, Skipped 数组语义, Mock 测试深度

---

## SSH 连接生命周期 (per-port 字面 vs per-batch 复用)

| Option | Description | Selected |
|--------|-------------|----------|
| DeviceExecutor.ExecuteCustom (推荐) | scheduler 管理的 ExecuteCustom 是 project 现成模式 (CollectionService/DeviceInfoCollectionService 同用法): 1 个 batch = 1 次 connection lifecycle,里面循环 50 端口 × SendConfig;获取超时/重试/panic 恢复全部走 scheduler 现有能力。你只要写 fn 内的循环逻辑;fn 跳出 = 释放。与现有 CollectionService.collectDevicePort 同形。 | ✓ |
| 直接 pool.GetConnection + defer ReleaseRef | 绕过 scheduler,裸用 DeviceConnectionPool.GetConnection(ctx, deviceID) + defer pc.ReleaseRef()。手写 ctx 截止 / 重试 / panic recover。多一些样板代码,但不依赖 Executor 的 scheduler 抽象;SC#2"per-port"字面意思最贴近。 | |
| 两条腿 (Executor 单端口 + batch 也用 Executor) | 单端口接口走 PooledConnection.Execute(ctx, fn) (fn 里调 SendConfig 然后释放),batch 接口也在外面包一层 GetConnection + defer ReleaseRef,内部循环 50 端口 SendConfig。保持 ExecuteCustom 的 try-finally 语义但不依赖 scheduler。 | |

**User's choice:** DeviceExecutor.ExecuteCustom (推荐)
**Notes:** 同 device pool 缓存复用连接,50 端口 × GetConnection 实际只返回同一个 pooled conn,故选 scheduler 包装的 ExecuteCustom 享受 timeout/retry/panic recover 一体化收益。SC#2"per-port"由 fn 内循环 SendConfig 达意,SC#2 字面 Acquire/ReleaseRef 由 scheduler 内部完成。

---

## Pre-state 数据源 (DB vs SSH)

| Option | Description | Selected |
|--------|-------------|----------|
| DB 读 device_port_status (推荐) | 读 sys_device_port_status.admin_status + dot1x_enabled。零 SSH 开销,batch 50 端口 = 50 次 DB SELECT (同 1 个 DB round trip 批查询),最差 stale 1h (下次 device_info_update cron 前)。与 PROJECT.md "复用 v1.18 DeviceInfoCollectionService" 战略一致。仅在 port 已 "消失" (DB 里找不到) 时才 fallback 到 SSH 读。 | ✓ |
| SSH 读 display this interface | 随时准确,又能发现上次 cron 后手工改过的状态。但每端口额外 1 条命令 (display this interface),batch 50 端口 ≈ +50–150 秒 latency;且如果 SSH 慢/拒绝,pre-state SSH 失败是 device_rejected 还是 transport_error 可起令. | |
| 混合 (DB 优先 + stale 阈值) | 查 `device_port_status.updated_at`,超过阈值 (10 分钟) 才启 SSH 重读新实时状态。中间情况需采集质量评估,这个 phase 加 complexity。 | |
| 不检查,直接下发 | PORT-06 不实现。shutdown 已 shutdown 的端口会被设备接受还是拒绝看具体固件。FUTURE-06 后续 phase 加。 | |

**User's choice:** DB 读 device_port_status (推荐)
**Notes:** Audit 时间窗最多 1h stale,与 v1.18 已采 cron 一致;batch 50 端口 SSH 延迟得不偿失。port "消失" (DB 行无) 时不 pre-state 检查直接下发,避免误报。

---

## `skipped` 数组语义 (本 phase 填 vs FUTURE-06 占位)

### Q1: 本 phase 是否填充 skipped

| Option | Description | Selected |
|--------|-------------|----------|
| 本 phase 填 (且 {single,batch} 都返回, 推荐) | PORT-06 pre-state 命中后: 单端口接口 (PORT-01..05) 返回 special NoOp 谈 + operlog 记"无需操作"; batch (BATCH-01) 推送进 skipped 数组。FUTURE-06 增强:加上"自动跳过"逻辑代替"判断后跳过"; 本 phase 只是预问预判不主动 skip。 | ✓ |
| 仅 batch 填 (单端口报 400) | 仅 batch 接口检测 PORT-06 并填 skipped,单端口接口报 400 "port already in target state"。UI 体验不均匀。 | |
| 完全留空 (FUTURE-06) | strict BATCH-03 只返 succeeded/failed 2 个数组,skipped 字段保留位但本 phase nil。FUTURE-06 再增。 | |
| 你决定 | 交给 Claude discretion | |

**User's choice:** 本 phase 填 (且 {single,batch} 都返回, 推荐)
**Notes:** BATCH-03 锁定的 partial result 三字段形状本 phase 直接落地;单端口 NoOp 也返 200 + NoOp 标识,与 batch 语义一致,UI 不需要分支处理。

### Q2: skipped 是否进 sys_port_write_audit + operlog OperType

| Option | Description | Selected |
|--------|-------------|----------|
| 都记,operlog 用 OperTypeStatus (推荐) | PORT-01..05 single port NoOp 仍记 operlog (OperTypeStatus=10),sys_port_write_audit 记 status='skipped' (本 phase schema 预位, Phase 52 AUDIT-03 migration 写 "status" 枚举多添一项 'skipped'); 对 audit 后人查 "这个端口有人看过但未改" 可以查到。 | ✓ |
| 只记 operlog, audit 表不写 | skipped 不进 audit 表只进 operlog (附带结构化样友提示)。 | |
| 什么都不记 | skipped 静默跳过什么都不记 (仅接口返 skipped 数组),audit 表仅记 succeeded/failed 状态。 | |
| 你决定 | 交给 Claude discretion, 但说明考虑 (如 audit 实例个数成本是否可控) | |

**User's choice:** 都记,operlog 用 OperTypeStatus (推荐)
**Notes:** 合规追溯完整性 — 有人看过但未改也留底;audit 表 status 枚举 3 值(succeeded/failed/skipped) 在 Phase 52-02 migration 预位。operlog 不引入新 OperType,OperTypeStatus(=10) 语义最贴近"状态变更意图但未生效"。

---

## Mock 测试深度 (纯 mock vs +fixture parseConfigError)

| Option | Description | Selected |
|--------|-------------|----------|
| Mock + inline 表驱动 marker 测试 (推荐) | 本 phase 无真机记错误响应样本 (Phase 54 W5 才取到),不引 testdata/。在 port_write_service_test.go 内联表驱动 tests, 手工构造 expected Huawei/H3C/Ruijie 错误响应文本 + 运行 parseConfigError。与 Phase 50 风格一致 (_test.go inline 非 testdata/ 调用)。Phase 54 再加 real-fixture 补充。 | ✓ |
| Only mock (SC#7 字面) | SC#7 字面意思: parseConfigError 仅在 mock 路径里被 service 测试覆盖。若 marker 识别逻辑变化 (例如 Sponsor 加 × H3C 尾点型),补看 interface test 中领 logic 不会动。 | |
| Mock + vendor fixture (testdata/) | 同样本 phase 无样本,跳到 Phase 54 一起补 real-fixture (testdata/ 路径)。本 phase 仅 mock。 | |
| 你决定 | 你决定 | |

**User's choice:** Mock + inline 表驱动 marker 测试 (推荐)
**Notes:** Phase 48 fixture 文件位置是 templates/samples/,但那是真机样本;本 phase 仅用 inline 构造保留 AC 覆盖率,Phase 54 site-visit 后再补 testdata/。

---

## Claude's Discretion

| Item | Discretion Choice |
|------|-------------------|
| Sentinel errors 定义位置 | 同文件 `port_write_service.go` 内部 `var Err...` (与 Phase 50 D-09 一致,1 文件 1 关注点) |
| 批量端口同 device 校验 | 强制 batch 内全同 device,错混返 `ErrMixedDevices`(避免跨设备拆 batch 的复杂度) |
| RateLimiter 集成单端口接口 | 不引,依赖前端按钮 disabled(UI-05) |
| PortStatus upsert 紧耦合 audit placeholder | 不写,audit 表由 Phase 52-02 AUDIT-03 建表,service 仅返回 result |
| mock struct 位置 | 同 `port_write_service_test.go` 内 inline (cache_invalidator_test.go 风格),不另建 mocks 子包 |
| Parsing 测试单测结构 | `TestParseConfigError` 表驱动 + `assert.ErrorIs/assert.Equal` (Phase 50 单测风格延续) |

## Deferred Ideas

### Suggestions captured (creator discretion, 非本 phase 落)
- 自动回滚(snapshot before + reverse on failure)— FUTURE-09
- 多用户并发写互斥(同一端口同时 1 operator 可写)— FUTURE-10
- 操作历史 UI(点击端口看历史 operlog)— FUTURE-05
- 写命令 dry-run 模式(预览命令但不发送)— FUTURE-04
- 设备组策略(batch 按 label/role 选端口)— 未来 phase

### Phase 51 内 scope creep 已 redirect (不归本 phase)
- per-port operator 字段从 gin context 注入(Phase 52 handler 责任)
- RateLimiter 单端口接口防 DoS(前端已 disabled 防)
- 混合设备 batch 自动拆分(抛错让客户端拆)
- operlog.Record 调用时机与脱敏兼容(Phase 52 handler 决策)
- real-fixture testdata 文件(Phase 54 site-visit 后有真实样本再补)
- 跨厂商命令抽象(FUTURE,vendor→template map 锁定)
