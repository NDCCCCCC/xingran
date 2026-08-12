# 🔍 XingRan-Next 后端代码审查总报告

**报告日期**: 2026-06-12
**审查范围**: `internal/` + `pkg/` 全部 Go 源码 (485 个文件)
**审查方式**: 6 个并行 Explore agent + `go build` + `go vet`
**审查维度**: 惯用性 / 并发 / 错误处理 / 性能 / 安全
**审查者**: Claude Code (golang-pro skill)

---

## 📊 编译与静态分析结果

| 检查项 | 结果 | 说明 |
|------|------|------|
| `go build ./...` | ✅ 通过 | 无编译错误 |
| `go vet ./...` | ⚠️ 5 处警告 | 见下方 |

**`go vet` 警告**:
1. `internal/core/db/migrations/migration_140_vdi_last_sync_time.go:32` — **struct tag 语法错误**: ``gorm:"column:last_sync_time" type:timestamp"`` 缺引号(高优先级,可能导致字段映射失败)
2. `internal/services/rate_limiter_test.go:17` — `assert.NotNil` 复制了含 `sync.Map` 的锁值
3. `internal/api/v1/system/user_handler_test.go:19` — 变量 `router` 声明未使用
4. `scripts/vdi_test_standalone.go` / `verify_migration.go` — `fmt.Println` 多余换行符

---

## 🚨 P0 严重问题汇总(必须立即修复 — 共 22 项)

### 🔐 安全(8 项)— **最高优先级**

| # | 位置 | 问题 | 风险 |
|---|------|------|------|
| 1 | `internal/services/addomain/ldap_client.go:58` | `InsecureSkipVerify: true` 硬编码 | AD 流量 MITM,凭据窃取 |
| 2 | `internal/services/addomain/utils.go:93-114` | AES 加密密钥硬编码 `"xingran-ad-domain-key-16"` | 二进制反汇编即可解密所有 AD 密码 |
| 3 | `internal/services/addomain/utils.go:86` | 解密失败时回退明文 | 绕过加密保护 |
| 4 | `internal/core/config.go:284` | JWT 默认密钥 `"xingran-next-secret-key"` | Token 可被伪造 |
| 5 | `pkg/middleware/request_decryption.go:138-160` | 解密后明文请求体被写入 INFO 日志 | **密码等敏感数据落盘** |
| 6 | `pkg/middleware/request_decryption.go:158` | 错误响应暴露内部解密错误细节 | 帮助攻击者探测密钥/IV |
| 7 | `pkg/middleware/websocket.go:34-40` | CORS 允许任意来源 + AllowCredentials:true | CSRF 盗用 token |
| 8 | `internal/api/v1/scheduler/job_handler.go:38-46` | `InvokeTarget` 无白名单校验 | 任意注册函数可被持久化为定时任务,命令注入风险 |

### ⚡ 并发(6 项)

| # | 位置 | 问题 | 影响 |
|---|------|------|------|
| 9 | `internal/services/data_cache_service.go:114` | `GetOrSet` 中裸 `go func` 启动写缓存,无 recover/重试/协调 | goroutine 堆积,Redis 连接耗尽 |
| 10 | `internal/api/v1/system/user_handler.go:146-153` | Update 中 AD 同步 goroutine 无 panic 保护、无重试、无 dedupe | 数据漂移源头 |
| 11 | `internal/scheduler/workorder_tasks.go:130-147` | `generateWorkOrderNo` 用 `Count+1` 生成工单号 | 并发场景下工单号重复 |
| 12 | `internal/services/workorder/periodic.go:361-383` | 轮询分配 `TotalGenerated` 字段存在 lost update | 多人分配给同一人 |
| 13 | `internal/websocket/notice_hub.go:167-175` | 注册向无缓冲 channel 同步发送 | 客户端可耗尽 FD |
| 14 | `internal/device/connection_pool.go:212-224` | `GetConnection` 释放锁后重新获取期间引用可能失效 | 死锁/数据竞争 |

### 🐛 数据一致性(4 项)

| # | 位置 | 问题 |
|---|------|------|
| 15 | `internal/core/core.go:140-145` | `Init()` 数据库初始化失败时 `return nil` 吞错 |
| 16 | `internal/core/core.go:240-323` | `Init()` 子步骤错误全部 `Warnf` 继续执行 |
| 17 | `internal/services/system/config_service.go:70-72` | 系统参数禁止修改判断写反:应是 `ConfigKey` 而非 `ConfigName` |
| 18 | `internal/services/system/role_cache_impl.go:200-208` | `InvalidateRoleCache` 中模式键(`*`)被当字面量精确删除 — **失效操作完全无效**!|

### 💉 SQL 注入与崩溃(2 项)

| # | 位置 | 问题 |
|---|------|------|
| 19 | `internal/services/operations/workstation_service.go:172-229` | `BatchUpdatePositions` 用字符串拼接 SQL,含 UUID `'` 字符可注入 |
| 20 | `internal/services/operations/excel_service.go:294-296` | 循环边界 `rowNum-1 >= len(rows)` 越界 panic |

### 🚪 越权(2 项)

| # | 位置 | 问题 |
|---|------|------|
| 21 | `internal/services/system/dashboard_service.go:386-402` | `GetTemplates` scope 为 nil 时返回所有模板(含私人) |
| 22 | `internal/services/addomain/user_ou_service.go:38-50` | `Unscoped` 查软删用户,可触发账户接管 |

---

## ⚠️ P1 重要问题(应该修复 — 共 47 项,精选 15 项)

### 安全相关
- `pkg/crypto/sm2_jwt.go:222-266` — 手写 JWT 解析未校验 `alg` 头,易受算法混淆攻击
- `pkg/crypto/request_encryption.go:88-103` — timestamp 窗口 ±300s 过宽,降低抗重放效力
- `pkg/crypto/nonce_storage.go` — `cleanupExpiredNonces` 已定义但**从未被调用**,内存无限增长
- `pkg/middleware/permission.go:106-118` — 子菜单权限自动包含父菜单权限,存在权限提升风险
- `internal/core/security/password.go:25` — PBKDF2 迭代仅 1000 轮,远低于 OWASP 推荐(≥600000)
- `internal/core/security/password.go:145-166` — `GenerateRandomPassword` 用 `%` 取模,模偏置降低熵
- `internal/services/operations/excel_handler.go:67-72` — 文件上传仅校验后缀,可绕过

### 并发与一致性
- `internal/services/addomain/sync.go:35` 等 — 同步方法未实现并发互斥,scheduler+手动触发可造成数据错乱
- `internal/services/addomain/group_sync_service.go:336-376` — AD 不可达时所有 DB 组被误判 stale 软删除
- `internal/services/addomain/dept_ou_mapper.go:60-100` — "先 delete 再 insert" 两步独立事务,崩溃后脏数据
- `internal/collectors/port_collector.go:353-381` — 端口状态逐条插入,N 台设备打爆连接池
- `internal/websocket/notice_hub.go` — 缺少 `readPump`,僵尸连接无法探测
- `internal/services/operations/excel_service.go:905-906` — `validateUniqueness` 逐行查询,5000 行 N 列 = 5000N 次往返

### 业务逻辑
- `internal/services/system/config_service.go:91-99` — `sys.request.encryption.enabled` 修改后缓存未失效,必须重启
- `internal/services/system/user_service.go:404` — `List` 中 `buildDepartmentPaths` 被调用两次

---

## 💡 P2 改进建议(共 80+ 项,精选)

- **Core 拆分**: `core.Core` 是巨型 god struct(25+ 字段),应拆为 `CoreServices`/`CoreInfra`
- **缓存键体系冲突**: `cache_keys.go` 与 `data_cache_service.go` 并存两套键定义,极易冲突
- **重复代码**: `user_service.go` 与 `user_service_optimized.go` 并存,优化版未在 router 中使用
- **迁移文件编号冲突**: `027/028/029/030/031/036` 多处同号文件,顺序不确定
- **错误处理风格不一**: `role_service.go` 用 `fmt.Errorf`,`user_service.go` 用 `apperrors`,Handler 无法统一映射
- **测试覆盖严重不足**: `ldap_client_test.go` 仅测工具函数,关键路径(Connect/Bind/Search)无 mock
- **空测试文件**: `stripBaseDN_test.go`, `dept_ou_mapper_test.go` 实为空壳无断言
- **设备子进程管理**: scrapli/Python subprocess 缺少僵尸进程清理

---

## ✅ 良好实践(值得保留)

1. **Handler-Service 依赖注入** 全面落地,符合规范
2. **GORM 参数化查询** 100+ 处使用占位符,无 SQL 注入(除上述 2 处例外)
3. **SM3 密码哈希 + ConstantTimeCompare** 防时序攻击
4. **抗重放设计** 完整实现 timestamp+nonce(尽管参数需调整)
5. **GORM `clause.OnConflict`** 大量使用,确保并发幂等
6. **ScrapliWrapper 双锁** (stateMu + opMu) 优雅处理 scrapligo 内部 goroutine 竞态
7. **DeviceTaskScheduler 设备级串行 worker** 避免 SSH 命令混淆
8. **WebSocket sync.Pool 复用** 减少 GC 压力
9. **AD 同步用 semaphore 控制并发** 优于裸 goroutine
10. **缓存接口 + NoOp 兜底** 实现透明降级
11. **递归 CTE** 在菜单/部门树查询中替代 N+1
12. **响应格式统一** 使用 `response.Success/Error` 而非裸 JSON
13. **路由约定一致** `POST /list`, `POST /:id/update` 等
14. **JWT 双 token + 黑名单** 续签机制完整

---

## 🎯 推荐修复优先级路线图

### 🔥 紧急修复(本周内,P0)
1. **修复 struct tag 语法错误**(`migration_140_vdi_last_sync_time.go:32`)— `go vet` 已报错,可能导致字段映射失败
2. **删除明文密码日志**(`request_decryption.go:138-160`)— 立即危险
3. **修复 `role_cache_impl.go:200-208` 缓存失效逻辑** — 当前角色更新后缓存永不失效,会引发权限错乱生产事故
4. **修复 `config_service.go:70-72` 字段写反** — 系统参数保护实际失效
5. **修复 SQL 注入**(`workstation_service.go:172-229`)
6. **修复工单号生成竞态**(`workorder_tasks.go:130-147`)— 改用 UUID/雪花 ID

### 🔒 安全加固(2 周内,P0 安全)
7. **AD 模块密钥与 TLS 配置**: 从 env/KMS 注入 AES 密钥,移除 `InsecureSkipVerify`
8. **JWT 默认密钥移除** — 强制启动时必须从 env 读取
9. **WebSocket 鉴权与 CORS 收紧**
10. **InvokeTarget 白名单** — 防止任意命令注入

### ⚙️ 并发与一致性修复(1 月内,P0 并发)
11. **裸 goroutine 全部加 recover 与协调**
12. **AD 同步加 singleflight 互斥**
13. **`Init()` 错误传播** — 至少 DB/Cache/Scheduler 失败应终止
14. **`handleDeletedGroups` 加入阈值保护** — 避免 LDAP 故障误删全部组

### 🛠️ 重构与改进(持续,P1+P2)
15. 迁移文件编号去冲突
16. 缓存键体系统一到 `cache_keys.go`
17. 统一错误处理到 `apperrors`
18. 补足 AD 模块关键路径单元测试
19. 拆分 `core.Core` god struct
20. Excel 导入全量使用事务

---

## 📈 总体评估

| 维度 | 评分 | 说明 |
|------|------|------|
| **架构清晰度** | ⭐⭐⭐⭐ | Handler-Service-Repository 分层清晰,模块边界明确 |
| **惯用性** | ⭐⭐⭐ | 大量使用 GORM/Gin 惯用模式,但存在重复实现 |
| **并发安全** | ⭐⭐ | 关键路径存在多处竞态,goroutine 管理欠缺 |
| **错误处理** | ⭐⭐⭐ | 设计了 `apperrors` 但落地不彻底 |
| **性能** | ⭐⭐⭐ | 缓存设计良好,但部分 N+1 查询和无界增长问题 |
| **安全性** | ⭐⭐ | 加密设计完整,但**关键密钥硬编码与 TLS 跳过验证** 是严重缺陷 |
| **测试覆盖** | ⭐⭐ | 关键模块(AD/JWT)单元测试缺失,部分测试是空壳 |

**总体结论**: 项目架构成熟、模式统一,但**安全配置(硬编码密钥/默认 JWT/TLS 跳过/明文日志)与若干隐蔽的并发缺陷(缓存失效逻辑/工单号竞态/AD 全删)需在生产部署前必须解决**。建议按上述路线图分阶段推进,先紧急修复 6 项再逐步加固。

---

## 📂 详细模块报告

以下是 6 个并行 agent 输出的详细模块审查报告。

---

## 模块 1: System 模块审查报告

### 1. 严重问题 (P0 - 必须修复)

- **[internal/services/data_cache_service.go:114]** `GetOrSet` 中使用 `go func() { _ = s.Set(context.Background(), key, data, expiration) }()` 启动 goroutine 后未做 panic 保护、未做错误传播、未用 WaitGroup/sync 协调 | 数据写入失败完全静默,可能因请求取消导致写入未完成但缓存又未命中,产生数据一致性问题,且 goroutine 在高并发下可能堆积导致 Redis 连接耗尽 | 建议:要么同步写缓存,要么引入工作队列 + 重试机制,而不是裸 `go func`

- **[internal/api/v1/system/department_handler.go:18-25]** `DepartmentHandler` 持有了 `*gorm.DB`,并在 `GetUsers` 中绕过 service 层直接用 `db.Table("sys_user")...Find(&users)` 查询(行 310-315) | 业务逻辑穿透到 handler 层,绕过 service 的缓存、权限、软删除过滤,`GetDB` 接口断言破坏 service 抽象 | 建议:在 `DepartmentService` 接口中新增 `GetDepartmentUsers(ctx, deptID)` 方法,由 service 实现查询

- **[internal/api/v1/system/user_handler.go:146-153]** `Update` 中 AD 同步 goroutine `go func() { ctx := context.Background(); ... }()` 完全无 panic 保护、无重试、无 dedupe | AD 调用失败仅 `applogger.Errorf` 记录,系统内用户已更新但 AD 端不一致,会成为数据漂移的源头 | 建议:将 AD 同步接入已有的 `command_dispatch_service.go`/消息队列或工作池,加入失败重试和监控

- **[internal/services/system/user_service.go:255-294]** `fillUserRoles` 在 GetByID 中执行两次顺序查询,与已经实现的 `user_service_optimized.go` 的 `loadUserRolesOptimized` 方案不一致,出现两个并行的用户服务实现 | 代码重复且行为分叉 | 建议:统一为 `userServiceOptimized.loadUserRolesOptimized` 方案,删除 `fillUserRoles`

### 2. 重要问题 (P1)

- **[internal/services/system/role_service.go]** 全部错误使用 `fmt.Errorf` 而非 `apperrors.RoleNameExists()` 等业务错误类型 | Handler 层无法映射业务错误码 | 建议:统一使用 `apperrors`
- **[internal/services/system/role_cache_impl.go:200-208]** `InvalidateRoleCache` 中 `keys` 包含模式(`*`),但 `InvalidateCacheByKey` 是按精确键 `Delete` | 失效操作实际不生效,角色数据更新后缓存仍是旧的 | 建议:拆分为 `InvalidateCacheByPattern` 调用
- **[internal/services/system/config_service.go:70-72]** 系统内置参数判断写反:应该是 `ConfigKey` 而不是 `ConfigName` | 实际未阻止修改系统参数 ConfigKey
- **[internal/services/system/config_service.go:91-99]** 修改 `sys.request.encryption.enabled` 后,运行中的进程不会感知到新值,必须重启 | 建议:在 `Update` 完成后立即失效缓存
- **[internal/services/system/dashboard_service.go:386-402]** `GetTemplates` 中当 `scope` 为 nil 时,会查询所有模板(包括私人模板) | 任何登录用户可越权读他人模板
- **[internal/services/system/user_service.go:339-340]** 硬编码 `dept_id::uuid` cast,字段类型不匹配时直接 SQL 报错 500
- **[internal/services/system/user_service.go:404]** `List` 中调用 `s.buildDepartmentPaths(ctx, list)` 两次
- **[internal/services/system/department_service.go:170-252]** 命名+状态过滤组合下表现不一致
- **[internal/api/v1/system/profile_handler.go:128-135]** 通过 `err.Error() == "旧密码错误"` 字符串比较判断错误类型

### 3. 改进建议 (P2)

- `user_service_optimized.go` 与 `user_service.go` 并存,接口相同但实现不同
- 缓存键管理在 `cache_keys.go` 和 `data_cache_service.go` 两套体系
- `CacheAdapter` 实现散落在三处文件中
- `menu_service.go:286-300` `collectAncestors` 单条递归查询,应改 CTE
- 大量重复的 `var rawReq map[string]interface{}` 类型断言代码

### 4. 良好实践

- 依赖注入而非全局变量(Router 层显式构造)
- 缓存双架构隔离清晰(legacy vs 新 CacheProvider)
- 批量插入避免 N+1(UserRole、RoleMenu、RoleDept)
- GORM 参数化查询(100+ 处)无 SQL 注入风险
- 统一响应格式
- 缓存键双前缀规范降低 Redis 键冲突
- 服务接口可测性(全部以接口+实现方式定义)
- 角色菜单 CTE 递归删除
- GORM `clause.OnConflict` 幂等

---

## 模块 2: Operations 模块审查报告

### 1. 严重问题 (P0)

- **[excel_service.go:294-296]** 循环边界错误,`rowNum-1` 在 `rowNum == len(rows)` 时越界 panic
- **[excel_service.go:1143-1156]** `setColumnWidths` 逐列独立设置,N+1 写入调用
- **[excel_service.go:224-392]** `ImportData` 没有使用事务,多批次 upsert 失败时无法回滚
- **[workstation_service.go:172-229]** `BatchUpdatePositions` 通过字符串拼接构造 SQL,UUID 含 `'` 会导致 SQL 注入
- **[workstation_service.go:179-192]** `WHEN '%s' THEN %d` 中 ID 仍为字符串拼接,SQL 注入面
- **[geocoding_service.go:227-237]** `setToMemoryCache` 内存缓存无界增长,`sync.Map` 永不清扫已过期项
- **[excel_service.go:38]** `getExcelService` 每次请求都创建新 GeocodingService 实例

### 2. 重要问题 (P1)

- `s.batchGeocodeBuildings` 错误仅 warn,失败行仍会被保存
- `validateAndParseRow` 内对每行逐个 `Unique` 列执行 `Count` 查询(N+1)
- `if len(rows) < 2` 误判模板/空文件
- 调试日志直接打印业务数据到 INFO 级别
- `processThreeLevelDepartments` 多遍循环,1000 条数据 = 3000+ 次往返
- `batchUpserter.standardUpsert` 失败时已成功批次不会回滚
- `applyDeptFilter` 的 ancestors 模糊匹配缺前缀匹配
- `isValidExcelFile` 仅校验后缀,可绕过
- `excelize.OpenReader(src)` 没有限制行数/列数,可触发 OOM

### 3. 改进建议 (P2)

- 多个手写工具函数应替换为标准库
- `isDuplicateKeyError` 用 `err.Error()` 字符串匹配 PG 错误码,应使用 `*pgconn.PgError.Code`
- `coordinatesToColumnString` 自实现,但 `excelize.ColumnNumberToName` 已存在

### 4. 良好实践

- `uuidValidator` 预编译正则强制 UUID 格式校验
- `maxBatchSize = 1000` 显式分批
- 内存 LRU + Redis 二级缓存,30 分钟 TTL
- 令牌桶限流器保护百度 API 配额
- `BatchUpdatePositions` 用 `CASE WHEN` 一次 UPDATE 多行(思路正确,但 SQL 拼接需修)
- `BatchDelete` 用单条 SQL 子查询批量更新
- `resolveDependentReferencesBatch` 避免笛卡尔积查询
- 地理编码并发限制 5 个,符合需求

---

## 模块 3: AD 域模块审查报告

### 1. 严重问题 (P0 - 安全风险!)

- **[ldap_client.go:58]** `InsecureSkipVerify: true` 硬编码 | AD 服务器 TLS 证书完全跳过验证,MITM 攻击;员工机器被植入 ARP 欺骗工具,所有 AD 同步流量被解密
- **[utils.go:93-114]** AES 加密密钥硬编码 `xingran-ad-domain-key-16` | 编译后任何人反汇编二进制即可解密所有 AD bind 密码
- **[utils.go:86]** `decryptPassword` 失败时回退明文 | 攻击者通过直接 SQL 注入写入伪加密字符串,绕过加密保护
- **[user.go:38, ou.go:64, group.go:36 等]** 关键同步/查询无 `config_id` 二次校验 | 越权访问其他 AD 配置的数据
- **[config.go:155-157]** 密码"留空不更新"逻辑过于隐式

### 2. 重要问题 (P1)

- 同步方法未实现并发互斥(scheduler + 手动同步并发会导致数据错乱)
- 大列表全量加载到内存(万级用户/组同步时 OOM 风险)
- `handleDeletedGroups` 一次性删除,LDAP 不可达时全表被误删
- `UpsertMapping` 事务不完整(先 delete 再 insert 两步独立事务)
- `searchWithPaging` 重试没有上限(可栈溢出)
- LDAP 失败不回滚本地状态
- `HandleUserLoginAD` 自动创建部门无数量上限,可被恶意员工撑爆 DB
- 唯一约束错误判定靠字符串匹配
- `Unscoped` 查软删用户风险(账户接管)
- `TestConnection` 失败时打印密码长度信息(侧信道)
- `BatchSyncADUsers` 无操作审计

### 3. 改进建议 (P2)

- 单连接无心跳/重连
- `queryAllComputerNames` 全表拉取
- `decryptPassword` 每次调用都重新解密(浪费)
- AD 错误码 68 字符串匹配脆弱
- 测试覆盖严重不足
- 单测零断言(空壳)
- 多个 Handler 返回 NotImplemented

### 4. 良好实践

- 分批 + 限流(batchSize=500,批间 sleep 1s)
- 降级策略(AD 失败不回滚业务连续)
- 类型安全的请求/响应分离
- 超时常量分层(同步 30min / 组同步 10min / 单组 2min)
- 后向兼容旧 API
- 审计日志完整
- 密码不在响应中泄露
- 软删除友好

---

## 模块 4: Core 基础设施审查报告

### 1. 严重问题 (P0)

- **[core.go:140-145]** `Init()` 中数据库初始化失败时 `return nil`(吞错)
- **[core.go:240-323]** `Init()` 全部子步骤错误均 `Warnf` 继续执行
- **[config.go:284]** `jwt.secret_key` 默认值 `"xingran-next-secret-key"`
- **[database.go:150-151]** `AutoMigrate` 失败仅 `Warnf`
- **[main.go:160-168]** pprof 和 Swagger 用 `cfg.Server.Mode != "release"` 弱判定

### 2. 重要问题 (P1)

- `Core` 是巨型 god struct(25+ 字段)
- `PwdManager: security.NewPasswordManager(nil)` 入参 `nil`
- `DefaultPasswordConfig.Iterations = 1000`(远低于 OWASP 推荐)
- `GenerateRandomPassword` 取模引入模偏置
- HS256 验证仅检查方法族,易受算法混淆
- JWT `ID = unix` 无随机性
- `configureGORM` 是空函数
- `SetConnMaxLifetime` 设置但无 `SetConnMaxIdleTime`
- `cleanupOldConstraints` 直接拼 SQL
- `performCacheWarmUp` 用 `go func()` 无 recover
- `Close()` 中 `time.Sleep(100ms)` 不可靠
- 模型软删除不一致
- `workstation_device.go` 字段名混用无 CHECK
- **迁移文件编号严重冲突**(027/028/029/030/031/036 多文件并存)
- `migrations/145_fix_bound_user_id_uuid.go` 编号混淆
- `initMACHistoryRetentionConfig` 绕过 service 层直接 `db.Create`

### 3. 改进建议 (P2)

- core 包高层耦合
- 命名/格式不规范
- Viper env 双重机制不一致
- 关键 env 未设置时静默使用默认值
- 缩进混乱
- 时区加载失败直接 `panic`
- shutdown 信号处理不健壮

### 4. 良好实践

- 缓存预热在独立 goroutine 不阻塞启动
- 凭证迁移使用 GORM `Transaction`
- `subtle.ConstantTimeCompare` 防时序攻击
- `BeforeCreate` 自动注入 UUID
- Close 顺序依赖反向
- 迁移前先清理历史遗留约束
- 自动创建数据库
- 管理员 DSN 与目标 DSN 分离

---

## 模块 5: pkg 公共包审查报告

### 1. 严重问题 (P0 - 安全风险!)

- **[pkg/middleware/websocket.go:34-40]** CORS 允许任意来源 + AllowCredentials:true
- **[pkg/middleware/request_decryption.go:138-160]** 解密后明文请求体写入 INFO 日志
- **[pkg/middleware/request_decryption.go:158]** 错误响应暴露内部解密错误细节
- **[pkg/middleware/response_encryption.go:107]** 响应加密时 Content-Type 判断错误
- **[pkg/middleware/permission.go:106-118]** 子菜单权限自动包含父菜单权限(权限提升风险)
- **[pkg/crypto/request_encryption.go:74-79]** RequestEncryptor 持有 SM2 公钥(无意义且增加密钥暴露面)
- **[pkg/middleware/auth.go:39-72]** 黑名单检查无超时/熔断,Redis 故障阻塞请求
- **[pkg/middleware/cors.go:34-39, :61]** CorsByPattern 子串匹配可被绕过

### 2. 重要问题 (P1)

- `validateTimestamp` 允许 ±300s 偏差(过大)
- `defaultNonceStorage` 256 个分片间无清理,内存无限增长
- 手写实现 JWT 三段解析,未校验 `alg` 头
- `readRequestBody` 总是读取整个 body
- `checkUserPermission` 每次请求最多 3 次原始 SQL,无缓存
- `IncrementWithExpire` 接口与实现脱节
- `evictLRU` 是 O(n) 全表扫描
- `redis.Nil` 与过期混淆
- 两套错误体系(`response.AppError` 和 `pkg/errors.AppError`)
- SM2 4 次尝试解密属于诊断式容错
- 错误状态码不加密,可能泄漏系统内部信息

### 3. 改进建议 (P2)

- `Cache` 接口过大(40+ 方法)
- 自定义 `contains` 函数与 `slices.Contains` 重复
- `HKeys` 剥前缀逻辑语义错误
- `matchDomainPattern` 可绕过
- `query/builder.go` 字段名直接拼入 SQL
- 包级硬编码路由
- 递归 CTE 在 MySQL 5.7 之前不支持

### 4. 良好实践

- 用 `atomic` 统计 hits/misses
- 分 256 分片降低锁竞争
- 统一 JSON 响应格式
- 数据库配置 + 30s 缓存实现配置热加载
- SM4 使用 GCM 模式(认证加密)
- 错误码分层(1000/2000/3000)
- WITH RECURSIVE CTE 避免 N+1
- 数据权限白名单字段
- timestamp+nonce 抗重放设计思路正确
- CAS 原子标志位避免重复写入
- 集中管理权限字符串常量

---

## 模块 6: 业务模块审查报告

### 1. 严重问题 (P0)

- **[internal/device/connection_pool.go:212-224]** 死锁风险
- **[internal/scheduler/cron.go:69-71]** 任务日志写入失败被静默
- **[internal/scheduler/workorder_tasks.go:130-147]** 工单号生成有并发竞态
- **[internal/services/workorder/periodic.go:361-383]** 轮询分配非原子
- **[internal/websocket/notice_hub.go:167-175]** 注册/注销同步阻塞
- **[internal/collectors/port_collector.go:353-381]** 端口状态单条插入
- **[internal/api/v1/scheduler/job_handler.go:38-46]** `InvokeTarget` 无白名单(命令注入)

### 2. 重要问题 (P1)

- WebSocket 缺少 readPump(僵尸连接无法探测)
- `Close` 中硬编码 sleep
- `removeConnection` 忙等式等待空闲
- `Stop` 等待不严谨
- 串行写日志阻塞任务返回
- 信号量无 ctx 取消
- 值班人员查询结果顺序不稳定
- WebSocket 鉴权缺失
- 凭据明文内存残留
- `StopJob` 删除逻辑冗余
- LIKE 拼接未做长度限制
- 配置加载空实现

### 3. 改进建议 (P2)

- 全局调度器指针可移除
- `InvokeTarget` 解析简陋
- 每次生成都新建 cron 实例
- 工单日志不记失败
- BroadcastRPAProgress 消息结构降级
- 错误吞噬
- 搜索未走缓存
- WarmUpCache 未实现却注册路由

### 4. 良好实践

- WebSocket 写协程独立 + sync.Pool 复用
- ScrapliWrapper 双锁设计
- 设备级锁 + 任务队列
- Scheduler 全局指针用 RWMutex 保护
- 周期工单模板创建用 cron 实例+Start 取 Next
- 模块化拆分清晰
- 缓存接口+NoOp 兜底
- 工单状态机
- AD 同步用 semaphore 控制并发
- 凭据加解密统一在 `addomain.PasswordCipher`
- 优雅停机

---

## 📝 报告生成元数据

- **总审查问题数**: P0=22, P1=47, P2=80+
- **审查时长**: ~3 分钟(并行)
- **token 使用**: 6 个 agent 并行 + 主对话整合
- **下次审查建议**: 修复 P0 后 2 周内复审,验证关键缓存失效/SQL 注入/工单号生成已修复

---

**审查报告完成**。如需针对任一具体问题展开深入分析、生成修复 PR 或为某个模块编写测试,请明确指示。
