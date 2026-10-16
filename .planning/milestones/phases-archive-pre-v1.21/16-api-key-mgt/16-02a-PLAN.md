---
phase: 16-api-key-mgt
plan: 02a
type: execute
wave: 2
depends_on: [16-01]
files_modified:
  - internal/services/system/apikey_service.go
  - internal/services/usage_logger.go
autonomous: true
requirements: ["INDEPENDENT"]
must_haves:
  truths:
    - APIKeyService 接口扩展包含使用日志方法
    - 使用日志记录功能实现（独立服务或服务方法）
    - 日志查询支持分页和筛选
    - 统计汇总功能实现（请求数、成功率、平均耗时）
    - 日志记录异步执行，不影响请求性能
  artifacts:
    - path: internal/services/system/apikey_service.go
      provides: APIKey 服务实现（扩展方法）
      min_lines: 100 (additional)
      contains:
        - func.*ListUsageLogs
        - func.*GetUsageLogSummary
    - path: internal/services/usage_logger.go
      provides: 使用日志记录服务
      min_lines: 150
      contains:
        - type UsageLogger interface
        - type usageLoggerImpl struct
        - func NewUsageLogger
        - func.*LogUsage
  key_links:
    - from: internal/services/usage_logger.go
      to: internal/models/api_key_usage_log.go
      via: GORM 操作
      pattern: db.*WithContext.*Model.*APIKeyUsageLog
    - from: internal/services/system/apikey_service.go
      to: internal/services/usage_logger.go
      via: 依赖注入
      pattern: usageLogger.*UsageLogger
---

<objective>
创建 API 密钥管理的服务层 - 使用日志和统计功能

目的：实现 API 密钥使用日志记录、查询和统计功能
输出：UsageLogger 服务，APIKeyService 扩展方法，统计汇总功能

**说明：** 这是独立功能模块，不依赖 REQUIREMENTS.md 中的具体需求 ID。本计划专注于使用日志和统计功能（LogUsage, ListUsageLogs, GetUsageLogSummary），与基础 CRUD（16-02）并行开发。
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/phases/16-api-key-mgt/16-CONTEXT.md
@.planning/phases/16-api-key-mgt/16-PATTERNS.md
@internal/models/api_key_usage_log.go
@internal/services/system/apikey_service.go
</context>

<tasks>

<task type="auto" tdd="false">
  <name>Task 1: 创建 UsageLogger 服务接口和实现</name>
  <files>internal/services/usage_logger.go</files>
  <read_first>
    - internal/services/system/apikey_service.go
    - internal/models/api_key_usage_log.go
  </read_first>
  <action>
创建 internal/services/usage_logger.go 文件，实现使用日志记录服务：

1. 定义 UsageLogger 接口：
   - LogUsage(ctx context.Context, req *LogUsageRequest) error
   - 接口设计为公开方法，供中间件直接调用

2. 定义 LogUsageRequest 结构体：
   - APIKeyID string
   - UserID string
   - Method string
   - Path string
   - StatusCode int
   - ClientIP string
   - UserAgent *string
   - Duration int
   - Success bool

3. 定义 usageLoggerImpl 结构体：
   - db *gorm.DB

4. 实现 NewUsageLogger 构造函数：
   - 参数：db *gorm.DB
   - 返回：UsageLogger

5. 实现 LogUsage 方法：
   - 异步执行（使用 goroutine）
   - 创建 APIKeyUsageLog 记录
   - 插入数据库
   - 错误处理：记录日志但不阻塞主流程
   - 返回 error（调用方可选择是否等待）
  </action>
  <verify>
    <automated>grep -E "type UsageLogger interface|type usageLoggerImpl struct|func NewUsageLogger|func.*LogUsage" internal/services/usage_logger.go</automated>
  </verify>
  <done>
    - UsageLogger 接口定义完整
    - 实现结构体定义正确
    - LogUsage 方法异步执行
    - 错误处理完善
  </done>
</task>

<task type="auto" tdd="false">
  <name>Task 2: 在 APIKeyService 中添加日志查询方法</name>
  <files>internal/services/system/apikey_service.go</files>
  <read_first>
    - internal/services/system/apikey_service.go
  </read_first>
  <action>
在 apikey_service.go 中添加使用日志查询功能：

1. 添加依赖到 apiKeyServiceImpl：
   - usageLogger UsageLogger（可选，如果需要直接调用）

2. 定义 ListUsageLogsParams 结构体：
   - APIKeyID string: 密钥ID
   - Current int: 当前页
   - PageSize int: 每页数量
   - StartTime *string: 开始时间
   - EndTime *string: 结束时间
   - Success *bool: 成功筛选

3. 实现 ListUsageLogs 方法：
   a. 构建查询（db.WithContext(ctx).Model(&models.APIKeyUsageLog{})）
   b. 添加筛选条件：
      - APIKeyID = ?
      - StartTime: created_at >= ?
      - EndTime: created_at <= ?
      - Success: success = ?
   c. 统计总数
   d. 分页查询（按 created_at DESC 排序）
   e. 返回分页结果

4. 定义 UsageSummary 结构：
   - TotalRequests int64
   - SuccessRate float64
   - AvgDuration float64
   - RequestsByMethod map[string]int64
   - RequestsByPath map[string]int64
   - ErrorsByStatus map[int]int
  </action>
  <verify>
    <automated>grep -E "type ListUsageLogsParams|func.*ListUsageLogs|type UsageSummary" internal/services/system/apikey_service.go</automated>
  </verify>
  <done>
    - 日志查询方法实现正确
    - 筛选条件完整
    - 分页功能正常
    - 统计结构定义清晰
  </done>
</task>

<task type="auto" tdd="false">
  <name>Task 3: 实现统计汇总功能</name>
  <files>internal/services/system/apikey_service.go</files>
  <read_first>
    - internal/services/system/apikey_service.go
  </read_first>
  <action>
在 apikey_service.go 中实现统计汇总功能：

1. 实现 GetUsageLogSummary 方法：
   a. 查询指定密钥的所有日志
   b. 计算总请求数：
      - COUNT(*)
   c. 计算成功率：
      - SUM(CASE WHEN success = true THEN 1 ELSE 0 END) / COUNT(*) * 100
   d. 计算平均耗时：
      - AVG(duration)
   e. 按方法分组统计：
      - GROUP BY method
      - COUNT(*)
   f. 按路径分组统计（TOP 10）：
      - GROUP BY path
      - ORDER BY COUNT(*) DESC
      - LIMIT 10
   g. 按状态码分组错误统计：
      - WHERE success = false
      - GROUP BY status_code
      - COUNT(*)
   h. 返回 UsageSummary 结构

2. 使用 SQL 聚合优化性能：
   - 使用单次查询获取所有统计
   - 或使用多个小查询并缓存结果

3. 处理空结果：
   - 无日志时返回零值
   - 避免除零错误
  </action>
  <verify>
    <automated>grep -E "func.*GetUsageLogSummary|TotalRequests|SuccessRate|AvgDuration" internal/services/system/apikey_service.go</automated>
  </verify>
  <done>
    - 统计汇总计算正确
    - SQL 聚合优化性能
    - 空结果处理完善
    - 所有指标计算准确
  </done>
</task>

<task type="auto" tdd="false">
  <name>Task 4: 验证编译和集成</name>
  <files>-</files>
  <read_first>
    - internal/services/usage_logger.go
    - internal/services/system/apikey_service.go
  </read_first>
  <action>
验证所有代码编译通过并正确集成：

1. 运行编译验证：
   - go build ./internal/services/...
   - 确保无编译错误

2. 验证接口一致性：
   - UsageLogger 接口方法签名正确
   - APIKeyService 新增方法与接口定义一致

3. 检查依赖注入：
   - 如果 APIKeyService 需要 UsageLogger，更新 NewAPIKeyService 构造函数
   - 或保持独立，中间件直接创建 UsageLogger 实例

4. 验证常量定义：
   - 确保无重复常量
   - 类型定义一致

5. 准备中间件集成：
   - 确认 LogUsage 方法可以被中间件调用
   - 验证异步实现不阻塞主流程
  </action>
  <verify>
    <automated>go build ./internal/services/... && go build ./internal/models/...</automated>
  </verify>
  <done>
    - 所有代码编译通过
    - 接口定义一致
    - 依赖注入正确
    - 准备好与中间件集成
  </done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| 日志记录 | 使用日志需要异步记录，避免影响请求性能 |
| 数据库操作 | 日志查询需要防止慢查询 |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-16-11 | Repudiation | 使用日志 | mitigate | 异步记录所有 API 调用，包含完整上下文信息 |
| T-16-12 | Denial of Service | 日志查询 | mitigate | 使用索引优化查询，限制分页大小，防止慢查询 |
| T-16-13 | Information Disclosure | 日志数据 | mitigate | 日志包含敏感路径和参数，需要权限控制访问 |
</threat_model>

<verification>
1. 检查所有服务文件是否存在且语法正确
2. 运行 go build ./... 验证编译通过
3. 验证日志记录异步执行
4. 验证统计汇总计算准确
5. 测试日志查询性能
</verification>

<success_criteria>
1. UsageLogger 服务实现正确
2. 日志记录异步执行，不影响性能
3. 日志查询支持分页和筛选
4. 统计汇总计算准确
5. 代码编译通过
</success_criteria>

<output>
执行完成后，创建 .planning/phases/16-api-key-mgt/16-02a-SUMMARY.md
</output>
