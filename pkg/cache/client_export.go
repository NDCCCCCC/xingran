package cache

import "github.com/redis/go-redis/v9"

// ExportClient 暴露 RedisCache 底层 Redis 客户端。
//
// Phase 153 / RUNTIME-03/04：RPA 取消指令与人工干预通道需要原生 Pub/Sub，
// 而 getClient 是包内未导出方法，跨包接口断言永远不成立（Go 规范：未导出
// 方法名按声明包限定）。本方法提供唯一跨包出口；无 Redis 后端的实现不持有
// 该方法，调用方按 `interface{ ExportClient() *redis.Client }` 断言即可。
func (r *RedisCache) ExportClient() *redis.Client {
	return r.client
}
