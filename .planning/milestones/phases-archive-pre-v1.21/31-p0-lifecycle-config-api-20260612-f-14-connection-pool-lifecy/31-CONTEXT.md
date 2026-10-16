# Phase 31 设计上下文 (CONTEXT)

**目的**: 完成 20260612 后端代码审查报告剩余 2 项 P0 finding。

## 来源 finding

| ID | 位置 | 审查诊断 |
|---|------|------|
| F-14 | `internal/device/connection_pool.go:212-224` | GetConnection 释放 deviceLock 后,pc 引用在 caller 调用 Acquire 前可能被 cleanup goroutine 关闭 → 死锁/数据竞争 |
| F-17 | `internal/services/system/config_service.go:70-72` | 系统参数禁修改键名判断写反:应是 `ConfigKey` 而非 `ConfigName`,且 `ConfigUpdateRequest` 根本未暴露 `ConfigKey` 字段 |

## 设计决策(用户对话已锁定)

### D-01 F-14 修复策略: refCount 实质性 fix
- 现状: `PooledConnection.Acquire()/Release()` 已实现 refCount 原子操作,但 `GetConnection()` 返回 pc 后才让 caller 调 Acquire,二者之间存在 race window
- 决策: **GetConnection 内部直接持有 refCount**(获取 deviceLock + IsReady → atomic.AddInt32(refCount,1) → 释放 deviceLock → 返回 pc),caller 不再需要 Acquire,但用完必须 Release
- 影响范围: GetConnection 内部逻辑 + 所有 5 个 caller 的 try/defer 模式
- 排除:
  - 闭包 + defer pattern (B) — API 变化大,所有 caller 重写
  - 取消连接池复用 (C) — 违背性能设计意图
  - 仅 defensive check (D) — 治标不治本

### D-02 F-17 修复策略: 加 ConfigKey 字段 + 修订校验
- 现状: 校验 `req.ConfigName != config.ConfigName` (写反),且 `ConfigUpdateRequest` 缺 `ConfigKey` 字段使保护不可达
- 决策: **ConfigUpdateRequest 添加可选 ConfigKey 字段 + 修订校验**(`req.ConfigKey != "" && req.ConfigKey != config.ConfigKey` → 拒绝)
- 行为:
  - 调用方未传 ConfigKey: 行为不变(仅更新 name/value/type/remark)
  - 调用方传不同的 ConfigKey: 拒绝(系统内置参数键不可改名)
- 排除:
  - 删除死代码 (B) — 失去未来 audit 痕迹
  - 仅加注释 (C) — 实际安全防护未建立

### D-03 PLAN 粒度: 中(按改动点拆,5 plan)
- 31-01: F-14 GetConnection 内部 Acquire 重构
- 31-02: F-14 caller 适配(5 处)
- 31-03: F-14 并发回归测试
- 31-04: F-17 ConfigUpdateRequest + Update 校验
- 31-05: F-17 验证 + Swagger/前端类型同步

## 测试与验证策略

- 每个 plan 必须通过 `go build ./...` + `go vet ./...`
- F-14 plans 必须通过 `go test ./internal/device/... ./internal/services/portcollection/... ./internal/collectors/...`
- F-17 plans 必须通过 `go test ./internal/services/system/...`
- baseline RED 数量(28)不允许新增

## 依赖与约束

- 不破坏前端: F-17 前端 settings-page 可能依赖 ConfigUpdateRequest payload,需同步 TypeScript 类型
- 不破坏并发: F-14 改动后的 race 窗口必须比之前更小(单元测试加并发场景)
- 向后兼容: ConfigUpdateRequest 加字段是 additive change(可选),不破坏现有 caller

## 相关 commit (已完成的 audit-fix 上下文)

- `1cc662d` F-22 软删用户接管 (同一审查报告)
- `8920ad1` F-03 decryptPassword 不回退明文 (同一审查报告)
- 共 50+ commit 在 `e32f46e..HEAD` 区间内
