package security

import "context"

// adConfigIDKey context key：传递当前 AD 配置 ID 到用户同步层（多域隔离判据）。
type adConfigIDKey struct{}

// WithADConfigID 将当前 AD 配置 ID 注入 context。
// 登录路径：ADAuthenticator.Authenticate 在调用 userSyncer.SyncADUser 前注入；
// 批量路径：handler 直接填 ADUserInfoForSync.ConfigID，不经 context。
func WithADConfigID(ctx context.Context, configID string) context.Context {
	return context.WithValue(ctx, adConfigIDKey{}, configID)
}

// ADConfigIDFromContext 读取 context 中的 AD 配置 ID（未注入返回空串，语义为"未知"，不阻断同步）。
func ADConfigIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(adConfigIDKey{}).(string); ok {
		return v
	}
	return ""
}
