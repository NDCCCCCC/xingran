package liveview

import (
	"context"
	"time"
)

// ctxLike 与 context.Context 等价的别名，便于把「可取消」语义显式写进签名，
// 同时避免调用方误传非 context 的值（Go 里 interface 别名没有额外约束，
// 这里的选择是**直接用 context.Context**，别名只为可读性）。
type ctxLike = context.Context

// newTimeout 便捷超时上下文。
func newTimeout(d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), d)
}

// ctxDone 取 Done channel（nil 安全：返回一个永不关闭的 channel）。
func ctxDone(ctx context.Context) <-chan struct{} {
	if ctx == nil {
		return neverClosed
	}
	return ctx.Done()
}

// ctxErr 取 ctx 的错误（nil 安全）。
func ctxErr(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}

// neverClosed 永不关闭的 channel（ctx 为 nil 时的安全占位）。
var neverClosed = make(chan struct{})
