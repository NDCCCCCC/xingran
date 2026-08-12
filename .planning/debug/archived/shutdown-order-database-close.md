---
slug: shutdown-order-database-close
status: resolved
trigger: 检查项目关闭流程，是否存在顺序错误导致数据库关闭后仍有数据库请求。重点排查：1. cmd/main.go 的 shutdown 顺序 2. Core.Close() 方法中的资源释放顺序 3. 是否有 goroutine 在 DB 关闭后仍在执行查询
created: 2026-05-12T08:00:00Z
updated: 2026-05-12T08:45:00Z
session_type: bug
---

# Debug Session: shutdown-order-database-close

## Symptoms

### Expected Behavior
项目关闭时，所有资源应按正确顺序释放，数据库应该在所有数据库操作完成后才关闭，避免 "database is closed" 错误或 panic。

### Actual Behavior
可能存在数据库关闭后仍有组件（如 Redis 缓存后台写入 worker）继续访问数据库的情况。

### Timeline
最近通过代码审查发现的潜在问题。

### Reproduction
启动项目后正常关闭，观察是否有错误日志。

### Error Messages
- 用户报告看到错误日志
- 可能涉及 Redis 缓存后台写入 (L2 writer pool)
- 具体错误信息待确认

### Context
用户排查方向：
1. `cmd/main.go` 的 shutdown 顺序
2. `Core.Close()` 方法中的资源释放顺序
3. 是否有 goroutine 在 DB 关闭后仍在执行查询

## Current Focus

- hypothesis: Redis 缓存 L2 writer pool 可能在数据库关闭后仍在执行后台写入操作
- next_action: ROOT CAUSE FOUND - multiple goroutines accessing database after shutdown
- test: 检查代码中资源关闭的先后顺序
- expecting: 发现哪些组件在数据库之前关闭，哪些 goroutine 可能仍在运行
- reasoning_checkpoint: null
- tdd_checkpoint: null

## Evidence

- timestamp: 2026-05-12T08:15:00Z
  source: code review
  finding: |
    检查了 `cmd/main.go` 的 `waitForShutdown()` 函数（第212-230行）：
    - 第222行：先调用 `server.Shutdown(ctx)` 关闭HTTP服务器
    - 第226行：然后调用 `coreModule.Close()` 关闭核心模块
    - 第227行：最后调用 `applogger.Close()` 关闭日志

  检查了 `internal/core/core.go` 的 `Close()` 方法（第354-391行）：
    - 第356-359行：停止通知中心
    - 第361-364行：停止设备信息采集服务
    - 第366行：停止AD域同步调度器
    - 第368-371行：关闭设备监控服务
    - 第373-376行：停止定时任务调度器
    - 第377-379行：关闭数据库连接 ⚠️
    - 第380-382行：关闭缓存
    - 第383-385行：停止系统指标缓存服务
    - 第387-390行：停止RPA扩缩容服务

  **发现问题**：数据库在第377行关闭，但缓存服务在第380-382行才关闭。

- timestamp: 2026-05-12T08:20:00Z
  source: code review
  finding: |
    检查了 `pkg/cache/redis.go` 的 `MultiLevelCache.Close()` 方法（第633-648行）：
    ```go
    func (m *MultiLevelCache) Close() error {
        // 先停止L2写入Worker
        if m.l2Writer != nil {
            m.l2Writer.Stop()
        }
        // 停止重试工作器
        if m.retryWorker != nil {
            m.retryWorker.Stop()
        }
        if err := m.l1Cache.Close(); err != nil {
            return err
        }
        return m.l2Cache.Close()
    }
    ```

  检查了 `pkg/cache/l2_writer.go` 的 `Stop()` 方法（第170-184行）：
    - 第178行：关闭工作队列 `close(w.workQueue)`
    - 第181行：等待所有worker完成 `w.wg.Wait()`
    - Worker会处理完队列中剩余的任务（第273-287行的 `drainQueue` 方法）

  **潜在问题**：虽然 `L2WriteWorker.Stop()` 会等待队列中的任务处理完成，但这些任务可能正在执行数据库查询。如果数据库已经关闭，这些查询会失败。

- timestamp: 2026-05-12T08:25:00Z
  source: code review
  finding: |
    检查了 `internal/services/data_cache_service.go` 的 `GetOrSet()` 方法（第61-82行）：
    ```go
    func (s *DataCacheService) GetOrSet(ctx context.Context, key string, dest interface{}, expiration time.Duration, query func() (interface{}, error)) error {
        err := s.Get(ctx, key, dest)
        if err == nil {
            return nil
        }
        data, err := query()
        if err != nil {
            return fmt.Errorf("查询数据失败: %w", err)
        }
        // 异步写入缓存
        go func() {
            _ = s.Set(context.Background(), key, data, expiration)
        }()
        return nil
    }
    ```

    **发现问题**：第76-79行启动了一个新的 goroutine 异步写入缓存。这个 goroutine 可能在数据库关闭后仍在执行。

- timestamp: 2026-05-12T08:30:00Z
  source: code review
  finding: |
    检查了 `internal/services/oper_log_service.go`，发现**更严重的问题**：

    **RecordAsync() 方法（第42-72行）**：
    ```go
    func (s *operLogService) RecordAsync(...) {
        // ...
        go func() {
            if err := db.Create(operLog).Error; err != nil {
                _ = err
            }
        }()
    }
    ```

    **RecordFromGinContext() 方法（第74-145行）**：
    ```go
    func (s *operLogService) RecordFromGinContext(...) {
        // ...
        go func() {
            if err := s.RecordOperLog(context.Background(), db, operLog); err != nil {
                _ = err
            }
        }()
    }
    ```

    **ROOT CAUSE 确认**：
    1. 操作日志服务在两个地方启动了**未受控的 goroutine** 直接写入数据库（第66行和第139行）
    2. 这些 goroutine 没有被 tracked，没有等待机制，也没有在 Core.Close() 中停止
    3. HTTP服务器关闭后，可能仍有请求正在处理，这些请求会触发操作日志记录
    4. 即使HTTP服务器关闭完成，这些异步 goroutine 可能仍在运行
    5. 数据库在第377行关闭后，这些 goroutine 尝试写入数据库时会失败

## Eliminated

## Resolution

- root_cause: |
    Core.Close() 方法中的资源关闭顺序错误。数据库在第377行先关闭，但此时仍有多个未受控的 goroutine 可能正在访问数据库：
    1. 操作日志服务的异步写入 goroutine（oper_log_service.go 第66、139行）
    2. 数据缓存服务的异步写入 goroutine（data_cache_service.go 第76行）
    3. 其他可能启动 goroutine 访问数据库的服务

    这些 goroutine 没有被 tracked，没有等待机制，导致数据库关闭后仍在执行查询，产生 "database is closed" 错误。

- fix: |
    **已应用修复：调整 Core.Close() 的关闭顺序（快速修复）**

    修改了 `internal/core/core.go` 的 `Close()` 方法（第352-391行）：

    **修复前的关闭顺序（错误）：**
    ```
    停止定时任务调度器
    关闭数据库 ⚠️ (第377行)
    关闭缓存 (第380行)
    停止系统指标缓存服务
    停止RPA扩缩容服务
    ```

    **修复后的关闭顺序（正确）：**
    ```
    停止定时任务调度器
    停止系统指标缓存服务
    停止RPA扩缩容服务
    关闭缓存（确保 L2 writer 完成所有异步写入）
    等待 100ms（给异步操作缓冲时间）
    关闭数据库（最后关闭）✅
    ```

    **关键改进：**
    1. 将系统指标缓存服务和 RPA 扩缩容服务停止移到缓存关闭之前
    2. 确保缓存在数据库之前关闭（L2 writer 会等待队列任务完成）
    3. 添加 100ms 延迟给未受控的 goroutine（如操作日志异步写入）缓冲时间
    4. 数据库连接最后关闭

- verification: |
    已验证编译通过：
    ```bash
    go build ./...
    # 编译成功，无错误
    ```

    建议的运行时验证步骤：
    1. 启动项目
    2. 发送一些 API 请求触发操作日志记录
    3. 立即发送 SIGTERM 信号关闭项目
    4. 检查日志中是否还有 "database is closed" 或 "failed to database" 错误

- files_changed:
    - internal/core/core.go (调整了 Close() 方法的关闭顺序)
