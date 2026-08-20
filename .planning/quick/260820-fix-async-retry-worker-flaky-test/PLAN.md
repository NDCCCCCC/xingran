---
type: quick
slug: fix-async-retry-worker-flaky-test
created: 2026-08-20
status: in_progress
description: 修复 pkg/cache TestAsyncRetryWorker_Enqueue 时序竞态 flake(测试断言 bug,非生产代码 bug)
---

# Plan: 修复 TestAsyncRetryWorker_Enqueue flaky test

## 背景

由 quick-260820-bcs(后端测试覆盖率扫描)发现:`pkg/cache.TestAsyncRetryWorker_Enqueue` 在 `go test ./...` 全量跑时失败,单跑通过。报错 `retry_test.go:209 expected 1 actual 0`。

## 根因(systematic-debugging 4 phases)

**测试断言设计错误(时序竞态),生产代码无 bug。**

```go
queueSize := worker.QueueSize()          // ① 入队后立即读:0 或 1(取决于 worker 是否已取走)
time.Sleep(200 * time.Millisecond)      // ② worker 在此期间消费队列
stats := worker.GetStats()
assert.Equal(t, queueSize, stats["queue_size"])  // ③ 断言"不变"——必然竞态
```

- Case A(worker 慢):queueSize=1 → sleep 后被消费 stats=0 → Equal(1,0) **FAIL** ← 观测到的失败
- Case B(worker 快):queueSize=0 → stats=0 → PASS

失败率实测 ~10%(10 次中 1 次,并发压力下 worker 在 Enqueue 与 QueueSize 之间未取走任务的概率)。

## 修复方案

1. **删掉竞态断言**:`assert.Equal(t, queueSize, stats["queue_size"])` 改为 `assert.GreaterOrEqual(t, statsQueueSize, 0)`(队列在并发消费下随时变化,非负是唯一可靠不变量)
2. **新增确定性测试 `TestAsyncRetryWorker_QueueSemantics`**(不启动 worker,无消费者):精确验证入队 → 队列大小 +1 的语义,弥补删掉断言的覆盖损失

## 不做

- 不改生产代码 retry.go(无 bug)
- 不改 l2_writer.go / l2_writer_test.go(相邻测试,working pattern 参考已记录)
- 不重构 AsyncRetryWorker(超出 scope)

## 成功标准

- [x] 单跑 TestAsyncRetryWorker* 通过
- [x] `go test ./pkg/cache/...` 连续 15 次全过(修复前 10 次 1 失败)
- [x] `go build ./...` 全过
- [ ] CI 同款命令 `go test -timeout 15m -count=1 ./internal/... ./pkg/... ./cmd/...` 全过(后台运行中)
