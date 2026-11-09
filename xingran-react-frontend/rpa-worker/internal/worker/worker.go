// Package worker 保留前端副本 rpa-worker 的 worker 包占位。
//
// Phase 137 / FIX-08：本文件此前是空文件（无 package 声明），导致
// `go build ./...` 直接报 "expected 'package', found 'EOF'"。
// 前端副本当前以 browser/executor/communication 为活动代码面，worker 包
// 暂无实现且无其它包引用，故仅补合法包声明，不引入任何行为。
package worker
