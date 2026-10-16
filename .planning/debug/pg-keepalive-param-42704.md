---
slug: pg-keepalive-param-42704
name: pg-keepalive-param-42704
status: resolved
trigger: go run ./cmd/main.go 启动失败: PostgreSQL 连接报 FATAL: unrecognized configuration parameter "keepalive_count" (SQLSTATE 42704)
created: 2026-10-15
updated: 2026-10-15 13:40
---

## 症状 (Symptoms)

**期望行为:**
- `go run ./cmd/main.go` 应用正常启动，数据库初始化成功

**实际行为:**
- 启动即失败退出（exit status 1），数据库初始化失败

**错误信息:**
```
INFO[2026-10-15 12:48:54] 数据库 ruoyi 已存在
ERRO[2026-10-15 12:48:54] [GORM错误] failed to initialize database, got error %v: failed to connect to `host=10.62.10.34 user=postgres database=ruoyi`: server error (FATAL: unrecognized configuration parameter "keepalive_count" (SQLSTATE 42704))
ERRO[2026-10-15 12:48:54] 数据库初始化失败: 连接PostgreSQL失败: 连接PostgreSQL失败: failed to connect to `host=10.62.10.34 user=postgres database=ruoyi`: server error (FATAL: unrecognized configuration parameter "keepalive_count" (SQLSTATE 42704))
exit status 1
```

**时间线:**
- GetDSN 的 keepalive 参数是本地未提交改动新加的（为 debug "backend-hang-on-automigrate" 问题），git diff internal/config/config.go 可证实

**重现步骤:**
1. `go run ./cmd/main.go`
2. 100% 复现

---

## 当前关注点 (Current Focus)

```yaml
reasoning_checkpoint:
  hypothesis: "GetDSN 的 keepalive_idle/interval/count 三个 DSN 参数不被 pgx v5 识别 → 全部进入 RuntimeParams 随 startup packet 发给服务器 → 服务器报 FATAL 42704 unrecognized configuration parameter"
  confirming_evidence:
    - "直接实验: 项目内 pgx v5.5.5 跑 pgx.ParseConfig(GetDSN query) → RuntimeParams 恰含三个 keepalive 参数, ConnectTimeout=20s 被原生消费"
    - "依赖审计: grep pgx v5.5.5/v5.6.0/v5.7.0/v5.9.0/v5.10.0 源码, 均无 keepalives_* DSN 参数处理, 默认 Dialer 仅 KeepAlive=5min"
    - "错误相关性: url.Query 按 key 字母序排列, connect_timeout 被消费后服务器遇到的第一个未知参数恰为 keepalive_count, 与报错完全吻合"
  falsification_test: "pgx.ParseConfig 实验若显示 keepalive 参数被映射到 pgconn 配置字段(而非 RuntimeParams)即推翻假设; 实测结果相反, 假设已被直接观察证实"
  fix_rationale: "移除三个不被识别的参数 = 参数不再进入 startup packet = 服务器不再收到未知 GUC = 根因消除(非症状掩盖); connect_timeout=20 保留(pgx 原生消费, 保留有界握手段意); TCP keepalive 基线由 pgx 默认 Dialer(KeepAlive=5min)天然提供"
  blind_spots: "本步骤未对真实 PG(10.62.10.34)做完整启动验证(按 fix_spec 验证口径为 build/vet/相关测试 + DSN 解析级检查); Supabase pooler 黑洞防护意图被有意放弃(内网 PG 不适用该场景)"
```

**用户已批准:** 方案 a（最小修复）— 移除 3 个 keepalive DSN 参数，保留 connect_timeout=20。

**下一步行动 (Next Action):**
编辑 internal/config/config.go GetDSN: 删除 q.Set("keepalive_*") 三行, 替换误导性注释; 然后 go build ./... + go vet ./internal/config/ + 查找 GetDSN 相关测试并运行。

---

## 证据 (Evidence)

- timestamp: 2026-10-15 12:55:00
  source: code_review
  detail: |
    `internal/config/config.go:550-563` GetDSN:
    ```go
    q.Set("sslmode", c.SSLMode)
    q.Set("timezone", "Asia/Shanghai")
    // 连接健壮性(debug backend-hang-on-automigrate):握手超时 + TCP keepalive。
    // 底层走 pgx(pgx.ParseConfig),解析器把这些 libpq 关键字映射到 net.Dialer。  ← 此认知错误
    q.Set("connect_timeout", "20")
    q.Set("keepalive_idle", "10")
    q.Set("keepalive_interval", "5")
    q.Set("keepalive_count", "3")
    ```
    三个 keepalive 参数均为本地未提交改动（git diff 证实）。

- timestamp: 2026-10-15 12:58:00
  source: dependency_audit
  detail: |
    go.mod: `github.com/jackc/pgx/v5 v5.5.5`。
    实测 pgx v5.5.5 pgconn/config.go 无任何 keepalives_* DSN 参数处理，仅默认 `&net.Dialer{KeepAlive: 5 * time.Minute}`（config.go:803）。
    connect_timeout 在 v5.5.5 被原生支持（config.go:278-283），解释了为何错误只报 keepalive_count 而非 connect_timeout。

- timestamp: 2026-10-15 13:00:00
  source: dependency_audit
  detail: |
    下载并 grep pgx v5.6.0 / v5.7.0 / v5.9.0 / v5.10.0 源码：均不支持 keepalives_* DSN 参数（config.go 仅有注释 "rely on GOLANG KeepAlive settings"）。
    结论：pgx v5 系列不支持 libpq keepalives_* 连接参数，升级依赖无法解决；自定义 net.Dialer.KeepAlive 需通过 pgconn.Config.DialFunc 注入（gorm postgres driver 需 Conn 注入方式），或直接移除三个参数。
    TCP keepalive 的原意图场景是 Supabase pooler 链路黑洞；当前环境是内网 PG (10.62.10.34)。

- timestamp: 2026-10-15 13:01:00
  source: error_log_correlation
  detail: |
    错误报 keepalive_count 而非 keepalive_idle/interval：url.Query().Encode() 按 key 字母序排列 → connect_timeout < keepalive_count < keepalive_idle < keepalive_interval。pgx 消费 connect_timeout 后，服务器遇到的第一个未知参数就是 keepalive_count。与 SQLSTATE 42704 (unrecognized_configuration_parameter) 完全吻合。

- timestamp: 2026-10-15 13:08:00
  source: direct_experiment
  detail: |
    临时程序 (tmp_verify_42704/main.go, 已清理) 用项目内 pgx v5.5.5 对 GetDSN 产出的 query 串直接跑 pgx.ParseConfig：
    RuntimeParams = {keepalive_idle:10, keepalive_interval:5, keepalive_count:3, timezone:Asia/Shanghai}
    ConnectTimeout = 20s (原生消费，未进 RuntimeParams)
    假设被直接观察证实：三个 keepalive 参数全部进入 startup packet。Orchestrator 独立 grep 模块缓存佐证 pgx v5.5.5~v5.10.0 均无 keepalives_* 参数支持。

---

## 已排除 (Eliminated)

- hypothesis: PostgreSQL 服务器端配置损坏或版本异常
  reason: 42704 报的是 startup packet 里的客户端传入参数，服务器是收到了未知 GUC 才拒绝；错误发生在连接握手阶段，服务器本身工作正常（同库其他服务正常连接）。

---

## 解决方案 (Resolution)

root_cause: |
  internal/config/config.go GetDSN（本地未提交改动，debug backend-hang-on-automigrate 时加入）写入了
  `keepalive_idle/keepalive_interval/keepalive_count` 三个 DSN 查询参数，误以为 pgx 会像 libpq 一样解析它们。
  实际上 pgx v5（实测 v5.5.5，grep 证实至 v5.10.0 均如此）不认识任何 keepalives_* 形式参数（正确 libpq 拼写
  也不支持），ParseConfig 将未识别参数全部放进 RuntimeParams，随 startup packet 发给 PostgreSQL 服务器；
  服务器对未知 GUC 报 FATAL 42704 unrecognized configuration parameter。connect_timeout=20 被 pgx 原生消费
  （ConnectTimeout=20s），因此错误只报按字母序第一个未知参数 keepalive_count。
fix: |
  方案a（用户批准的最小修复）: internal/config/config.go GetDSN 移除 q.Set("keepalive_idle"/"keepalive_interval"/
  "keepalive_count") 三行，保留 q.Set("connect_timeout", "20")；同时替换误导性注释（原注释错误声称 pgx 会把
  libpq keepalive 关键字映射到 net.Dialer），新注释说明: connect_timeout 由 pgx 原生消费（有界握手）；TCP
  keepalive 基线由 pgx 默认 Dialer（net.Dialer KeepAlive=5min）提供；激进 keepalive（如 Supabase pooler
  黑洞场景）需经 gorm postgres driver 的 Conn 配置注入 pgconn.Config.DialFunc —— pgx v5 不支持任何 keepalive
  DSN 参数（任意拼写均落入 RuntimeParams，服务器报 FATAL 42704）。
verification: |
  全部通过（2026-10-15 13:40）:
  1. go build ./... — 干净无输出
  2. go vet ./internal/config/ — 干净
  3. 回归: TestGetDSN_URLEscape 3 个子测试全 PASS（该测试只断言 URL 转义/sslmode/timezone，不含 keepalive 断言）
  4. 根因级验证（临时测试 TestTmpGetDSN_NoKeepaliveParams，验证后已删除）: 修复后 GetDSN 输出
     postgres://...@10.62.10.34:5432/ruoyi?connect_timeout=20&sslmode=disable&timezone=Asia%2FShanghai，
     pgx.ParseConfig 后 RuntimeParams 仅含 timezone（修复前实验为 keepalive_idle/interval/count 三项），
     ConnectTimeout=20s 被原生消费。与确认根因时用的同一观察方法，机制层面确认三个参数已不再进入 startup packet。
  注: 完整端到端启动（go run ./cmd/main.go 连真实 PG）由用户在部署验证时确认；本次按 fix_spec 口径验证。
files_changed:
  - internal/config/config.go
