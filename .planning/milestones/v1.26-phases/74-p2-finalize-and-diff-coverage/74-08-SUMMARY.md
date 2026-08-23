---
phase: 74-p2-finalize-and-diff-coverage
plan: 8
subsystem: p2-gap-closure
status: complete
date: 2026-08-22
---

# 74-08 SUMMARY: P2 gap-closure — 9 packages coverage lift

## Result

74-08 在 Wave 编排中漏执行(74-11 Task 1 测量时发现无 SUMMARY),按 74-11 escalation
条款先行补齐。**D-12 STRICT 满足: 11 个新文件全部为 `*_test.go`,git diff 零业务代码变更**,
`go build ./...` + `go test ./...` 全绿。

## Coverage 表(before → after)

| 包 | before | after | Δ | 达 70%? |
|----|--------|-------|---|---------|
| internal/services/scheduler | 4.8% | **89.8%** | +85.0 | ✅ |
| internal/templates | 44.4% | **88.1%** | +43.7 | ✅ |
| internal/core/security | 48.9% | **74.8%** | +25.9 | ✅ |
| internal/core/db | 37.5% | **72.9%** | +35.4 | ✅ |
| pkg/crypto | 33.7% | **71.8%** | +38.1 | ✅ |
| pkg/middleware | 14.3% | **68.8%** | +54.5 | ⚠️ 差 1.2pp |
| pkg/cache | 24.6% | **52.6%** | +28.0 | ❌ Redis 路径阻塞 |
| internal/device | 2.5% | **39.1%** | +36.6 | ❌ SSH 路径阻塞 |
| internal/core | 2.1% | **38.3%** | +36.2 | ❌ Init 链路阻塞 |

## 各包落地内容

- **scheduler**: JobService Create/Update/Delete/GetByID/List/UpdateStatus/Execute 全分支
  (重名/cron 校验/F-08 InvokeTarget 白名单/mock SchedulerClient 错误注入/暂停不同步) +
  JobLogService Create/List(时间范围/排序白名单)/Statistics 空/CleanOldLogs。
- **security**: NewJWTManager 三重密钥校验(空/弱默认/短)+ SM2 自动生成与 hex 注入;
  ValidateToken HS256 全错误分支(过期/未生效/错 issuer/none-alg)+ SM2 路径;
  RefreshToken roundtrip(含 SM2)+ 非 refresh 角色拒绝;DecryptPassword roundtrip;
  GenerateRandomToken;ADAuthenticator 拨号失败(plain/SSL/StartTLS)/账号池 nil/
  getDefaultRoleID;工厂账号池注入。
- **templates**: ParseTemplate embed/无前缀/绝对路径/错误路径;resolveEmbedPath 逃逸;
  findProjectRoot;四种 Value 形态 + 状态定义;ParseText 记录提取/状态转换
  (Continue/Record/Error/未知)/Clone 并发隔离(8 goroutine);escapeRegexLiteral 边角。
- **core/db**: FilterLogger Info/Warn 级别语义(LogLevel>=msgLevel)/Error 五分支/
  Trace 慢查询/错误/ErrRecordNotFound/保留路径;keepalive 生命周期(幂等/ping/Close×2);
  GetDB;InitData sqlite 全链幂等 + 缺表失败;BootstrapMissingTables 幂等;
  AutoMigrate sqlite 全量(12 表抽查)+ SKIP_AUTOMIGRATE 仅 PG 语义 + PG 守卫
  fail-safe;createDatabaseIfNotExists 非法名。
- **device**: ModelExtractor 5 厂商表驱动;IdentifyVendor/DeviceType;SNMP 辅助
  (toUpper/contains*/isDigit*/ScanIPRange 非法输入/ipToUint32/nextIP/ConvertPortToInt);
  entity MIB stub SNMPGetter(Walk 聚合/单 Get 五属性容忍/索引提取/类型转换);
  PlatformName/platformIdentifier(patched yaml);checkDeviceReachable 快速失败;
  GetCommandForVendor/GetLLDPCommand/containsEOF/containsConnectionError/ElapsedTime;
  Manager sqlite 全查询分支(关联凭证/默认凭证/无凭证/幽灵凭证)+ 兼容空操作;
  连接池 Stats/Lifecycle/GetDevice;任务调度器 Submit 校验/SubmitAndWait 快速回调/
  GetStats;executor 访问器 + 禁用池错误路径。
- **core**: CaptchaService LoadConfig 10 键默认/覆盖/非法开关回退 normal/非法数字保留;
  Verify 数字/滑块全分支(容差/token/verified 标记);登录锁定(重试递增/锁定/清除);
  getIPRateLimit 三态;滑动生成 auto 降级;CaptchaBackgroundService validateFile/
  calculateMD5/getImageDimensions/Upload 校验失败/GetRandomEnabled 精确命中+缓存/
  IncrementUseCount/缓存池取放/PreGeneratePool 空库;loadConnectionPoolConfig
  默认/覆盖/非法回退;parseDuration;initSM4Cipher(空/默认警告/非法/roundtrip);
  MetricsCacheService 直采。
- **pkg/cache**: MemoryCache 全 API — Get/Set(string/[]byte/int 断言)/Delete/Exists/
  MGet/MSet/MDelete/Increment/Decrement(string/int 值)/Expire/TTL(永续 -1)/Keys
  (通配)/FlushDB/LRU 淘汰/手动 cleanup/JSON(单+批量)/Int/Bool 访问器/Hash
  (HSet 混合类型/HGet/HGetAll/HKeys/HDel 空表删 key)/GetStats 命中率;
  matchPattern/CacheItem.IsExpired/errors 谓词。
- **crypto/middleware**: 前段已完成(见 git log 同 commit)。

## 未达标根因(诚实记录)

| 包 | 阻塞点 |
|----|--------|
| pkg/middleware 68.8% | 剩余为 WebSocketAuth 完整 core.Core 依赖(重依赖集成路径) |
| pkg/cache 52.6% | RedisCache 全部方法需真实 Redis;D-12 禁止新增 miniredis 依赖 |
| internal/device 39.1% | ScrapliWrapper 持具体 scrapligo `*network.Driver` 不可注入;connection_pool createConnection 需真实 SSH 握手;e2e_helpers 显式 TEST-ONLY |
| internal/core 38.3% | `Core.Init()` 全链依赖 Redis/调度器/RPA/子进程 reaper,单测无法构造 |

## QUIRKS(D-12 不修复,测试内注释锁定)

1. **MemoryCache.IncrementBy** 对不存在 key 在 `item.Expiration` nil 解引用直接
   panic(memory.go:215)— 生产 Redis 路径无此问题,MemoryCache 限 dev。
2. **MemoryCache.IncrementBy** 非数值字符串 ParseInt 失败被静默吞掉按 0 继续累加。
3. **ModelExtractor** 锚点 `(?:^|[\s\r\n])` 的前导字符进入 FindString 结果,随后
   `^[A-Z0-9-]+` 提取为空 → 仅型号在串首才命中;行首/空格分隔的真实 sysDescr 形状
   返回空串(caller 有 ExtractModelFromSysDescr 旧回退)。
4. **gmsm v1.4.1 sm2.Decrypt** 对合法 base64 但非 SM2 密文 panic(makeslice: len
   out of range),无长度预检。
5. **CaptchaBackgroundService.validateFile** 无扩展名文件名 `ext[1:]` 越界 panic。
6. **GetRandomEnabled** fallback 查询含 PG-only `@>`/jsonb_array_length,sqlite 下
   报 unrecognized token("没有可用背景图"分支仅 PG 可达)。
7. **MetricsCacheService.Stop** 二次调用 close 已关闭 channel panic(非幂等)。
8. **USG6000E** 型号正则 `USG[0-9]{4,5}(?:-[A-Z0-9]+)?` 不含尾字母 → 提取为 USG6000。
9. **nextIP(255.255.255.255)** 因 net.IP 4/16 字节形态差异返回全零 IP 而非 nil
   (调用方 ipToUint32=0 终止,不发散)。

## 原子提交

- `test(74-08): p2 gap-closure coverage lift across 9 packages`(11 files, +~3600)
