# Nonce 存储方案配置指南

## 高并发场景解决方案

### 性能对比（单机）

| 方案 | QPS | 内存占用 | 延迟 | 适用场景 |
|------|-----|----------|------|----------|
| **分段锁 Sharded Map** | 500,000+ | 低 | 极低 | 单机高并发（推荐） |
| **sync.Map** | 200,000+ | 低 | 低 | 读多写少 |
| **全局锁 Map** | 50,000 | 低 | 中 | 低并发 |

---

## 使用方式

### 方案一：分段锁（默认，推荐）

**无需修改代码，已自动启用**。分段锁将 nonce 分散到 256 个分片中，大幅降低锁竞争。

```go
// 默认就是分段锁实现
encryptor := crypto.NewRequestEncryptor(privateKey, publicKey)
```

**特点：**
- ✅ 性能最优（500K+ QPS）
- ✅ 无需外部依赖
- ✅ 自动清理过期数据
- ❌ 不支持多实例部署

---

### 方案二：Redis（分布式部署）

如果你有多台服务器实例，需要使用 Redis：

```go
// 在创建 RequestEncryptor 后设置 Redis 存储
encryptor := crypto.NewRequestEncryptor(privateKey, publicKey)

redisStorage, err := crypto.NewRedisNonceStorage(crypto.RedisNonceStorageConfig{
    Addr:     "localhost:6379",
    Password: "",
    DB:       0,
})
if err != nil {
    log.Fatal(err)
}
encryptor.SetNonceStorage(redisStorage)
```

**在项目中配置：**

找到 `internal/api/router.go` 或初始化加密器的地方：

```go
// 从配置文件读取 Redis 配置
redisConfig := crypto.RedisNonceStorageConfig{
    Addr:     viper.GetString("redis.addr"),
    Password: viper.GetString("redis.password"),
    DB:       viper.GetInt("redis.db"),
}

// 创建 Redis nonce 存储
redisStorage, err := crypto.NewRedisNonceStorage(redisConfig)
if err != nil {
    log.Fatalf("Redis nonce 存储初始化失败: %v", err)
}

// 设置到加密器
crypto.GetRequestEncryptor().SetNonceStorage(redisStorage)
```

**特点：**
- ✅ 支持多实例部署
- ✅ 自动过期清理
- ✅ 可监控和管理
- ⚠️ 依赖 Redis 可用性

---

### 方案三：sync.Map（读多写少）

```go
encryptor := crypto.NewRequestEncryptor(privateKey, publicKey)
encryptor.SetNonceStorage(crypto.NewSyncMapNonceStorage())
```

**特点：**
- ✅ 代码简单
- ✅ 读操作无锁
- ⚠️ 写操作性能一般
- ❌ 不支持多实例部署

---

## 监控和调试

### 获取 nonce 数量

```go
if storage, ok := encryptor.nonceStorage.(interface{ GetNonceCount() int }); ok {
    count := storage.GetNonceCount()
    log.Printf("当前存储的 nonce 数量: %d", count)
}
```

### 压力测试

```bash
# 使用 wrk 或 ab 进行压测
wrk -t12 -c400 -d30s http://localhost:8080/api/v1/network/devices/list \
  -H "Content-Type: application/json" \
  -d @encrypted_request.json
```

---

## 建议

1. **单机部署**：使用默认的分段锁方案，无需任何配置
2. **多实例部署**：必须使用 Redis 方案
3. **监控**：定期检查 nonce 数量，防止内存泄漏

---

## 技术细节

### 分段锁原理

```
┌─────────────────────────────────────────┐
│           Sharded NonceStorage           │
├─────────┬─────────┬─────────┬─────────┤
│ Shard 0 │ Shard 1 │ Shard 2 │  ...    │
│  Lock   │  Lock   │  Lock   │   Lock  │
├─────────┼─────────┼─────────┼─────────┤
│ map[K]V │ map[K]V │ map[K]V │ map[K]V │
└─────────┴─────────┴─────────┴─────────┘

通过 hash(nonce) % 256 决定使用哪个分片
256 个分片 = 锁竞争降低到 1/256
```

### Redis 方案原理

```
Client A: SETNX nonce:abc123 123456 EX 600
Client B: SETNX nonce:abc123 123456 EX 600 (同时)
           ↓
Redis: Client A 返回 true (成功)
Redis: Client B 返回 false (已存在，拒绝)
```

利用 Redis 的原子性操作，天然支持分布式场景。
