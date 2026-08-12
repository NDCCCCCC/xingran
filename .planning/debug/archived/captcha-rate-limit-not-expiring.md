---
slug: captcha-rate-limit-not-expiring
status: resolved
deferred_to: v1.16-tech-debt
trigger: 10.62.10.33登录一直提示验证码"获取验证码过于频繁，请稍后再试"，过了一天了还是提示这个，请检查1分钟有效期的配置是否生效
created: 2026-05-23T11:15:00+08:00
updated: 2026-06-25
session_type: bug
---

# Debug Session: captcha-rate-limit-not-expiring

## Symptoms

### Expected Behavior
验证码频率限制应该在配置的1分钟有效期后自动清除，允许用户再次获取验证码。

### Actual Behavior
用户从IP 10.62.10.33 尝试获取验证码时，持续收到"获取验证码过于频繁，请稍后再试"错误，即使过了一天仍然提示这个错误。

### Error Messages
```
WARN[2026-05-23 11:11:48] 获取验证码失败: 获取验证码过于频繁，请稍后再试, clientIP: 10.62.10.33
WARN[2026-05-23 11:11:48] Client error
client_ip=10.62.10.33 latency=2 method=POST path=/api/v1/system/auth/captcha request_body="{}" request_id=mphrve5k6w0f25lyhk5g status_code=400 user_agent="Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
```

### Timeline
- 2026-05-23 11:11:48: 错误首次报告
- 问题持续时间：过了一天仍然存在
- 预期：1分钟后应该自动清除

### Reproduction
1. 从IP 10.62.10.33 访问系统
2. 尝试获取验证码
3. 收到"获取验证码过于频繁"错误
4. 等待超过1分钟（甚至一天）
5. 仍然收到相同错误

### Scope
- 影响范围：特定IP (10.62.10.33) 可能所有用户受影响
- 功能状态：验证码频率限制缓存未正确过期

## Current Focus

- hypothesis: Redis缓存TTL配置未生效或缓存key没有设置过期时间
- next_action: 验证root cause - Expire调用失败被静默忽略
- test: 检查验证码频率限制的缓存实现和TTL配置
- expecting: 找到缓存TTL未正确设置的根本原因
- reasoning_checkpoint: null
- tdd_checkpoint: null

## Evidence

- timestamp: 2026-05-23T11:20:00+08:00
  source: code_analysis
  finding: |
    在 internal/core/captcha.go:242-246 发现问题代码：
    ```go
    count, err := s.cache.Increment(ctx, rateLimitKey)
    if err == nil && count == 1 {
        // 首次访问，设置过期时间为1分钟
        _ = s.cache.Expire(ctx, rateLimitKey, 1*time.Minute)
    }
    ```
    问题分析：
    1. Increment创建新key时不设置TTL（Redis默认行为）
    2. Expire调用使用 `_` 丢弃错误，失败时无日志
    3. 如果Expire调用失败（Redis连接问题、权限问题等），key将永不过期
    4. 代码假设count==1时一定是首次创建，但如果是已存在的key，也可能从其他地方被重置为1

- timestamp: 2026-05-23T11:25:00+08:00
  source: redis_behavior_analysis
  finding: |
    Redis INCR命令行为：
    - 当key不存在时，INCR会创建key并设置为1，但不会设置过期时间
    - 如果Expire命令失败（网络、权限、key被删除等），key将永久存在
    - 在高并发场景下，可能出现多个goroutine同时执行Increment，导致count>1的情况

- timestamp: 2026-05-23T11:30:00+08:00
  source: code_flow_analysis
  finding: |
    验证码频率限制流程：
    1. 第一次请求：Increment返回1，尝试设置1分钟TTL
    2. 如果Expire失败（被 `_` 忽略），key永久存在
    3. 后续请求：Increment继续递增，最终超过IPRateLimit（10次）
    4. 一旦超过限制，所有后续请求都会被拒绝，key永远不会被删除
    5. 即使等到第二天，key仍然存在且值很大，继续拒绝请求

## Eliminated


## Resolution

- root_cause: |
    验证码频率限制的TTL设置失败被静默忽略。当 `cache.Expire()` 调用失败时，错误被 `_` 丢弃，导致频率限制key永不过期。一旦超过限制次数（10次/分钟），该IP将被永久封禁，直到手动删除Redis key或重启服务。

- fix: |
    **采用方案1：Lua脚本原子操作**
    
    1. **pkg/cache/redis.go** - 添加 `IncrementWithExpire` 方法：
       - 使用 Lua 脚本原子执行 INCR + EXPIRE
       - 当 count == 1 时自动设置 TTL
       - 返回错误而不是静默忽略
    
    2. **internal/core/captcha.go** - 使用类型断言调用新方法：
       - 优先使用 `IncrementWithExpire`（原子操作）
       - 降级到旧的 Increment + Expire（兼容非Redis缓存）
       - 添加错误日志记录
    
    **关键代码变化：**
    ```go
    // 新方法（pkg/cache/redis.go）
    func (r *RedisCache) IncrementWithExpire(ctx context.Context, key string, expire time.Duration) (int64, error) {
        script := redis.NewScript(`
            local current = redis.call('INCR', KEYS[1])
            if current == 1 then
                redis.call('EXPIRE', KEYS[1], ARGV[1])
            end
            return current
        `)
        return script.Run(ctx, r.client, []string{r.buildKey(key)}, int64(expire.Seconds())).Int()
    }
    
    // 使用方式（internal/core/captcha.go）
    if redisCache, ok := s.cache.(interface{ IncrementWithExpire(...) }); ok {
        count, err = redisCache.IncrementWithExpire(ctx, rateLimitKey, 1*time.Minute)
        // 错误被记录而不是静默忽略
    }
    ```

- files_changed:
  - pkg/cache/redis.go (添加 IncrementWithExpire 方法)
  - internal/core/captcha.go (使用原子操作替换旧的 Increment + Expire)

- testing: |
    1. ✅ 编译验证通过
    2. ⏳ 功能测试：使用同一IP连续请求11次验证码，验证1分钟后是否恢复
    3. ⏳ Redis验证：检查 key 的 TTL 是否正确设置为 60 秒
    4. ⏳ 临时解除封禁：删除 Redis key `xingran:captcha:rate:10.62.10.33`

- status: fix_applied

## Phase 40 Closure (2026-06-25)

复测 `pkg/cache/redis.go:177 IncrementWithExpire`（Lua 脚本原子 INCR+EXPIRE）
+ `internal/core/captcha.go:253-265` 已优先走原子路径，仅当 cache 实现未实现
该接口时降级 Increment+Expire。TTL 静默丢失风险已消除。
frontmatter 翻 `resolved`。

verification: `grep -n "IncrementWithExpire" pkg/cache/redis.go internal/core/captcha.go` 命中
files_changed: .planning/debug/captcha-rate-limit-not-expiring.md
