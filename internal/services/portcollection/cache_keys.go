package portcollection

// Cache key 常量 — 端口写操作结果/批量任务 (Phase 52 INFRA-03 占位)
//
// 命名规则: `port:write:<sub>:<suffix>`  (%s = port_id / batch_id)
//
// Redis 前缀处理 (CLAUDE.md Cache Key Prefix Handling):
//   - 实际 Redis key 会自动加 `xingran:` 前缀,本文件定义的 key 是"逻辑键"
//   - 调用 cache.Set(ctx, key, ...) 时,直接传本文件的 key 即可,不要手动加前缀
//
// MVP 演进 (D-10 锁定):
//   - Phase 52: 仅 cache_keys 文件就位,无运行时调用(INFRA-03 占位)
//   - Phase 53+: 接入 CacheProvider 后启用（如"最近一次写结果"缓存）
const (
	// CacheKeyPortWriteResult 单端口最近一次写结果缓存 key（%s = port_id）
	CacheKeyPortWriteResult = "port:write:result:%s"
	// CacheKeyPortWriteBatch 批量任务汇总结果缓存 key（%s = batch_id）
	CacheKeyPortWriteBatch = "port:write:batch:%s"
)

// GetPortWriteResultKey / GetPortWriteBatchKey 占位 helper（Phase 53+ 接入 CacheProvider 时启用）
//
// 取消注释后可直接使用,无需再次修改 cache_keys.go。
//
// func GetPortWriteResultKey(portID string) string { return fmt.Sprintf(CacheKeyPortWriteResult, portID) }
// func GetPortWriteBatchKey(batchID string) string  { return fmt.Sprintf(CacheKeyPortWriteBatch, batchID) }
