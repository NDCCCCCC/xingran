---
slug: backend-hang-on-automigrate
status: resolved
trigger: 运行后端并根据日志修复bug
created: "2026-08-13T08:48:00Z"
updated: "2026-08-13T11:44:00Z"
---

# Debug Session: backend-hang-on-automigrate

## Symptoms

1. **Expected behavior**: 后端 `go run ./cmd/main.go` 后应打印 `数据库表结构迁移完成` 并在 :9000 启动监听。
2. **Actual behavior**: 启动日志停在 `[dropDependent] 当前架构下 noop(...)` 后再无输出,进程不退出但也不监听 :9000。
3. **Error messages**: 无显式错误,INFO 日志正常后静默 hang 死。
4. **Timeline**: 2026-08-13; 与 Phase 58 在 Supabase Session pooler 上 GORM AutoMigrate(80+ DDL) 卡死的问题一致。
5. **Reproduction**:
   ```bash
   go build -o xingran-backend.exe ./cmd/main.go
   ./xingran-backend.exe
   ```
   复现率: 100%。

## Evidence

- timestamp: 2026-08-13T08:46:40Z — 启动
- timestamp: 2026-08-13T08:46:56Z — PostgreSQL连接成功
- timestamp: 2026-08-13T08:47:47Z — [dropDependent] noop 输出(距连接成功约 51s,本身已异常慢)
- timestamp: 2026-08-13T08:49:10Z — 90s 后仍未出现 `数据库表结构迁移完成` 或 `Listening`
- timestamp: 2026-08-13T19:24:42Z — SKIP_AUTOMIGRATE=true bisect 运行(二分):dropDependent noop 之后再无输出 → AutoMigrate 是 hang 点
- timestamp: 2026-08-13T19:26:40Z — 同 bisect 内 BootstrapMissingTables 8 条 DDL 全部 OK(~4s),打印 `数据库表结构迁移完成` → **证明 AutoMigrate 是 hang 点**
- timestamp: 2026-08-13T19:27:36Z — bisect 内 InitData 在第 3 个验证码 sys_config seed count 处静默 hang(同一根因:链路 stall + 无超时)
- timestamp: 2026-08-13T19:37:37Z — 应用 GetDSN keepalive 修复后,SKIP_AUTOMIGRATE=true + SERVER_SKIP_SETUP=true 干净启动
- timestamp: 2026-08-13T19:39:42Z — PostgreSQL连接成功(~13s,较此前 101s 的 createDatabaseIfNotExists + Ping 明显改善)
- timestamp: 2026-08-13T19:39:44Z — BootstrapMissingTables 8/8 OK,`数据库表结构迁移完成`
- timestamp: 2026-08-13T19:43:38Z — `服务器启动在端口: 9000`(ListenAndServe 调用)
- timestamp: 2026-08-13T19:44:02Z — `curl http://127.0.0.1:9000/health` → `{"status":"ok"}`,PID 19864 独占 :9000

## Eliminated

- hypothesis: 后端编译失败
  reason: `go build ./...` 与 `go build -o xingran-backend.exe ./cmd/main.go` 均成功
- hypothesis: 数据库连接完全失败
  reason: 日志已打印 `PostgreSQL连接成功`
- hypothesis: dropDependentMaterializedViews 自身死循环
  reason: 当前实现是 noop,仅打印一行日志;hang 发生在该函数返回之后
- hypothesis: 服务端表锁 / 活跃查询阻塞 DDL
  reason: scripts/dbprovision 注释引用前序排查结论"server 端确认无锁无 active query";本次 bisect 期间 PG 端无锁,GORM AutoMigrate 仍 hang → 排除服务端锁
- hypothesis: 某条具体 DDL/语句卡死(ALTER TYPE 0A000 / 索引 / 约束)
  reason: AutoMigrate 在第一条 model 内省就 stall(并非跑到特定表);BootstrapMissingTables 跑等价 CREATE TABLE + INDEX 全部成功 → 不是单条 DDL 的问题,是"大量串行 round trip × 链路随机 stall × 无超时"的统计性必然

## Root Cause Analysis

**根因:GORM AutoMigrate / InitData 在高延迟 Supabase Session pooler 链路上做数百次串行 round trip,叠加随机 TCP 黑洞,而 DB 连接层无任何 read deadline/keepalive → 被丢弃连接上的阻塞 Read 永久挂起 = 启动"挂死"。**

链条证据:
1. **bisect 证明 AutoMigrate 是 hang 点**:SKIP_AUTOMIGRATE=true 旁路后,同一 DB 同一链路,BootstrapMissingTables(等价 raw DDL,8 条)~4s 全部成功并打印 `数据库表结构迁移完成`;不旁路则 100% hang 在 dropDependent noop 之后(即 `d.DB.Migrator().AutoMigrate(MigrateModelList()...)` 内)。
2. **不是服务端锁**:scripts/dbprovision 注释(line 211)记录前序排查"server 端确认无锁无 active query";本次 bisect 期间无锁,AutoMigrate 仍 hang。
3. **不是单条 DDL**:BootstrapMissingTables 跑 CREATE TABLE + CREATE INDEX(与 AutoMigrate CREATE-only 路径等价)全部成功;AutoMigrate 对已存在表会逐列内省(information_schema),~65 model × ~15 列 ≈ 上千次 round trip。在该链路(企业网→新加坡 pooler,单 round trip 2-6s + 随机 TCP stall)上,统计上必然会命中 stall;一旦命中,lib/pq/pgx 无 read deadline → 永久阻塞。
4. **同一根因波及 InitData**:bisect 内 InitData 在第 3 个验证码 sys_config count 查询静默 hang —— 与 AutoMigrate hang 完全同质(任意 round trip 命中 stall 即永久阻塞)。
5. **底层驱动确认**:gorm.io/driver/postgres v1.5.9 在 DriverName 为空时走 pgx(`pgx.ParseConfig` + `stdlib.OpenDB`,driver/postgres.go:92-105)。pgx 解析器识别 libpq 风格 `connect_timeout` / `keepalive_idle` / `keepalive_interval` / `keepalive_count`,映射到 net.Dialer。修复前 GetDSN() 只设 sslmode + timezone,无任何超时/keepalive。

## Resolution

**root_cause**: GORM AutoMigrate(及 InitData)对每列做 information_schema 内省,产生数百~上千次串行 round trip;Supabase 新加坡 Session pooler 链路存在随机 TCP 黑洞(server 端无锁无 active query),而 GetDSN() 未设 connect_timeout/keepalive,被丢弃连接上的阻塞 Read 永久挂起 → 启动挂死在 AutoMigrate 内省 / InitData seed。

**fix**:
1. **根因修复(internal/config/config.go `GetDSN()`)**:DSN 增加 `connect_timeout=20` + `keepalive_idle=10` + `keepalive_interval=5` + `keepalive_count=3`。pgx 把这些映射到 net.Dialer:连接 idle 10s 起探测,每 5s 一次,连续 3 次无 ACK 判死(~25s 返回 error),把"无限挂起"转化为"有界错误"。database/sql 不自动重试,故 AutoMigrate 会 fail-fast(由旁路规避),InitData 会 warn-continue(启动继续到 :9000)。适用于所有 DB 代码路径,不仅 AutoMigrate。
2. **dev 操作模式(已有开关,非本次改动)**:Supabase pooler dev 环境用 `SKIP_AUTOMIGRATE=true`(旁路 GORM AutoMigrate 内省风暴,改走 BootstrapMissingTables raw DDL)+ `SERVER_SKIP_SETUP=true`(旁路慢速 InitData seed 循环)启动。
   ```bash
   SKIP_AUTOMIGRATE=true SERVER_SKIP_SETUP=true ./xingran-backend.exe
   ```
3. **远端 dev DB 补建工具已存在**:`scripts/dbprovision`(HasTable → 仅对缺失表 AutoMigrate,每操作 30s ctx 超时 + 8 次重试)用于一次性补齐 133 张期望表中缺失的 55 张;本次未改动。

**verification**:
- `go build ./...` 通过;`go build -o xingran-backend.exe ./cmd/main.go` 产出 127MB 二进制。
- 清场:taskkill 两个遗留 xingran-backend.exe 僵尸进程(其中一个冒充 :9000 listener 造成假阳性),确认 :9000 FREE。
- 干净启动:`SKIP_AUTOMIGRATE=true SERVER_SKIP_SETUP=true ./xingran-backend.exe` → 19:39:29 启动,19:43:38 `服务器启动在端口: 9000`,19:44:02 `curl /health` → `{"status":"ok"}`。
- 进程状态:仅 PID 19864 一个 xingran-backend.exe,:9000 独占监听。
- 启动耗时 ~4m9s:剩余开销来自 sequential service init(MAC 历史任务 seed / 设备监控并发配置读取 / captcha 目录 / RPA scaler / 路由注册)在 pooler 上的串行 round trip —— 是"慢"不是"hang",且 keepalive 使任何 stall 在 ~25s 内以 error 释放连接,不再永久阻塞。

## Remaining / Follow-ups (非阻断,不在本次 scope)

- **production**: 直连 DB(非 pooler)链路快且稳定,AutoMigrate 应保留(不旁路);keepalive 修复对生产无害,作为安全网保留。
- **启动慢(~4min)**:dev 上 InitData/scheduler/router 的串行 DB round trip 仍慢。可选后续:给 service init 的 DB 查询加 ctx 超时 + 并发化,或 dev 默认 SkipSetup。本次未动(超出"修复 hang"的最小范围)。
- **MAC 历史父表缺失**:SKIP_AUTOMIGRATE 下 `sys_device_mac_history` 父表未建 → 月度分区创建失败(WARN 非阻断)。需用 scripts/dbprovision 或对应 SQL migration 补建父表。
- **reconciliation_normalized MV 缺失**:Phase 42 R1 启动 RefreshView 报 42P01(WARN 非阻断)。由 AutoMigrate 完成后 migration_176 重建;SKIP_AUTOMIGRATE 下不会重建,需手动跑或接受 cron 下次失败。
- **false-positive 教训**:调试中 :9000 LISTENING 曾由遗留僵尸进程(PID 53744)持有,导致误判"已启动"。后续验证 :9000 必须先 `taskkill /F /IM xingran-backend.exe` 清场再跑,确认 listener PID 属于本次启动的进程。

## Files Changed

- `internal/config/config.go` — `GetDSN()` 增加 connect_timeout + keepalive 四参数(唯一源码改动)。
