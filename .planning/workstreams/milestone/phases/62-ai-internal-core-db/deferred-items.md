# Phase 62 Deferred Items

执行期发现的 out-of-scope 问题(SCOPE BOUNDARY:只记录不修)。

## 2026-08-14 (Plan 62-05 执行期发现)

- **[预存在测试失败]** `internal/api/v1/auth` TestADLoginWithOUProcessing 两个子用例
  (`AD登录成功-有OU映射` / `本地登录-不处理OU`)在 auth_handler_test.go:67 断言
  "应触发/不应触发OU处理" 失败。该目录自初始提交 ea528c6 后未改动,与 Phase 62
  全部 5 个 plan 的 diff 无交集(database.go bootstrap 旁路 + core.go SKIP_AUTOMIGRATE
  守卫均不在 AD 登录代码路径上)。隔离重跑 deterministic 失败。建议 /gsd-debug 立项。
