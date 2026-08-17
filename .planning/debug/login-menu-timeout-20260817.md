# Debug Session: 登录后菜单加载超时

**slug:** login-menu-timeout-20260817  
**started:** 2026-08-17  
**status:** awaiting_human_verify (Round 4)  
**severity:** high — blocks login flow  

## Current Focus (Round 4 — 2026-08-17, FIX APPLIED)

**status:** awaiting_human_verify — Round 4 全部修复已应用,build/test/DB 层验证通过。
**applied:**
1. **议题1-500(认证关键路径 Redis 依赖):**
   - A) `pkg/middleware/auth.go` — JWTAuthWithBlacklist 黑名单检查 fail-open(err → Warn 日志放行,JWT 7200s 过期兜底),不再 fail-closed 500。
   - B) `internal/services/token_blacklist_service.go` — 30s 进程内负缓存("未拉黑"判定不回源)+ 30s 错误熔断窗口(故障期每 30s 最多 1 次 Redis 阻塞,不再每请求 10s);AddToBlacklist/RemoveFromBlacklist 立即使对应负缓存失效。回归测试 4 个全 PASS。
   - C) `pkg/cache/redis.go` — MultiLevelCache.Exists L1 前置(L1 hit→true 恒正确: Set 同步先写 L1,Delete 先删 L1;L1 miss 仍回源 L2,无假阴性)。
2. **议题2-启动 23505:**
   - 代码: `internal/core/db/init_data.go` ensureDept 改两段式查询 — ① dept_code(真唯一业务键) ② dept_name+parent_id(兼容历史无编码种子行,C5 部分恢复语义不变)。回归测试 TestCreateDefaultDept_ParentDriftRecovers PASS(精确复现事故: 物理删根→重建→孤儿领养,旧代码必 23505)。
   - 数据(带外,已提交并验证): 探针证实旧根 c64513be **未被物理删除,而是被重命名为"若依科技有限公司"且 dept_code 被清空**(2026-08-13 创建) — "物理删除"假设被证伪。12:21:50 启动撞出的重复根 b4fbab6d(无子/无用户/无工位引用)已物理删除;c64513be 恢复 dept_code='ROOT'。验证: ROOT 行=1(锚定 c64513be),悬挂 parent_id=0。
3. **清理:** scripts/deptfix、scripts/deptprobe 已删除。

**next_action:** 用户验证 — 重启后端:① 启动日志不再有 23505/initData 报错;② 登录 + my-menus 系列在 Redis 抖动时不再 500(可有 warn 日志 `[JWTAuthWithBlacklist] 令牌黑名单检查失败,放行请求`);③ 部门管理页面应只显示"若依科技有限公司"一棵树,无重复"总公司"。  

## Symptoms

- 浏览器控制台：`Failed to load resource: net::ERR_NAME_NOT_RESOLVED` (utils.js:1)
- 登录流程在 `getUserMenus()` (`menuApi.ts:8`) 抛出：`Error: 请求超时，请检查网络连接`
- 堆栈：`api.ts:472` → `handleNetworkError` (`errorHandler.ts:238`)
- 登录接口本身是否成功未知（日志中未见 `/system/auth/login` 近期记录）

## Environment

- Frontend dev server: Vite on `:4000`, `host: true`
- `VITE_API_BASE_URL=/api/v1` (relative path)
- Vite proxy: `/api` → `http://localhost:9000`, `/uploads` → `http://localhost:9000`
- Backend: `0.0.0.0:9000`, mode debug, last startup 2026-08-13 16:46:40
- Backend logs: `logs/app.log` (~99k lines), `logs/backend-run.log`

## Symptoms (immutable)

- 浏览器控制台：`Failed to load resource: net::ERR_NAME_NOT_RESOLVED` (utils.js:1)
- 登录流程在 `getUserMenus()` (`menuApi.ts:8`) 抛出：`Error: 请求超时，请检查网络连接`
- 堆栈：`api.ts:472` → `handleNetworkError` (`errorHandler.ts:238`)

## Evidence

- **timestamp:** 2026-08-17T12:21-12:26 (Round 4, 运行时验证)
  **checked:** Round 3 修复后重启的运行日志(12:21:50 新二进制,pool-keepalive 日志证实新代码生效)
  **found:** (a) 12:21 重启后零 `[L2WriteWorker] 任务已取消` 日志(最后一条 11:13:11 来自旧进程) — H8 修复运行时确认;(b) 12:26:32 的 6 个并发登录后请求中 5 个最终 200(permissions 25.8s/39.9s, my-menus/all 29.5s/54.7s, my-menus 56.1s) — 冷缓存 + Supabase 网络退化下仍慢但不再 500;(c) DB 慢查询仍 2-7s(pooler 网络退化,12:22:16 pool-keepalive dial timeout)。
  **implication:** Round 3 修复方向正确且生效;残余慢查询是网络环境问题,非代码问题。

- **timestamp:** 2026-08-17T12:26:42 (Round 4, 新失败模式)
  **checked:** `POST /api/v1/system/my-menus` 500, latency=10010ms, request_id=mswqes1i3i920cn3jqv
  **found:** 12:26:32→42 的 10s 窗口内**零 GORM 慢查询/错误日志** — 请求未达 DB;延迟精确等于 pkg/cache/redis.go:17-20 的 poolTimeout=10s。调用链: auth.go:49 JWTAuthWithBlacklist → blacklistSvc.IsBlacklisted → token_blacklist_service.go:54 cache.Exists → redis.go:710-712 MultiLevelCache.Exists **绕过 L1 直达 L2**(远程 Upstash TLS) → 网络退化窗口内阻塞至 poolTimeout → error → auth.go fail-closed 500。
  **implication:** 认证关键路径对每个请求都强依赖远程 Redis 可用性与延迟,且 fail-closed;同窗口其他 5 个请求先抢到池连接而幸存。

- **timestamp:** 2026-08-17T12:21:50 (Round 4, 议题 2)
  **checked:** 启动日志 initData → createDefaultDept INSERT '深圳总公司' SQLSTATE 23505 idx_sys_dept_dept_code
  **found:** (a) 远程库 live 查询确认: 旧根 c64513be 被**物理删除**,孤儿 深圳总公司(1b52ace0)/长沙分公司(a61f1a9b) parent_id 悬挂;12:21:50 启动 ensureDept("总公司",nil) 未命中 → 新建根 b4fbab6d → 深圳总公司按 (name, parent=b4fbab6d) 未命中 → INSERT 撞 dept_code 唯一索引;(b) 代码确认: models/dept.go:7 `DeptCode uniqueIndex;not null` 是真唯一业务键,而 init_data.go:80-85 ensureDept 按 (dept_name, parent_id) 查 — 键不匹配;(c) 应用侧删除路径 department_service.go:296 为软删除(无 Unscoped) — 物理删除只能来自带外(Supabase 控制台/手工 SQL),app 代码路径不可复现。
  **implication:** 代码修复=ensureDept 按 dept_code 优先查询(幂等对 parent 漂移鲁棒);数据修复=带外 SQL 把孤儿重挂到新根并修正 ancestors 前缀。

- **timestamp:** 2026-08-17T13:00 (Round 4, FIX — 议题1)
  **checked:** 三项修复的 build/vet/test
  **found:** `go build ./...` / `go vet`(middleware/cache/services/db) 通过;新增 4 个黑名单回归测试全 PASS(负缓存 5 次查询仅 1 次回源;已拉黑不缓存每次回源;熔断窗口内直接放行且不再打缓存;AddToBlacklist 立即使负缓存失效);`go test ./pkg/cache/ ./pkg/middleware/ ./internal/middleware/ ./internal/core/db/` 全 PASS。`internal/services` 包内 TestCollectBoardsInto_*/TestCollectDeviceInfo_*/TestNormalizeMACAddress(网络设备 MAC/板卡解析)失败为**既有失败**,与本次改动零交集(本包仅改 token_blacklist_service.go)。
  **implication:** 议题1 修复在单元层闭环:Redis 故障 → 熔断 fail-open(不再 500、不再每请求 10s 阻塞);正常期同 token 30s 内零远程往返。

- **timestamp:** 2026-08-17T13:05 (Round 4, FIX — 议题2,关键证伪)
  **checked:** 只读探针 scripts/deptprobe 列出 sys_dept 全部行(含软删除)
  **found:** **"物理删除"假设被证伪** — c64513be 行仍 live,dept_name 被改为"若依科技有限公司"、dept_code 被清空(行创建时间 2026-08-13 14:02,早于证据中的 08-15 估计);子树(深圳/长沙/研发/市场/测试)parent/ancestors 完整挂在 c64513be 上,无任何悬挂。12:21:50 启动因 ensureDept 按 (dept_name='总公司', parent IS NULL) 查不到改名行 → 新建重复根 b4fbab6d(code=ROOT, 实际创建于 10:18 — 说明 23505 自 10:18 起每次启动必现) → SHENZHEN INSERT 撞唯一索引。
  **implication:** open question 闭环:没有物理删除者,是**重命名 + 清编码**破坏了 (name,parent) 幂等键;dept_code 优先查询正是正确治本(改名不影响 code 锚定 — 本例 code 也被清空属极端情况,由数据修复恢复锚点)。

- **timestamp:** 2026-08-17T13:06 (Round 4, FIX — 议题2 数据修复)
  **checked:** scripts/deptfix 带外执行(前置校验: dup 根无子部门/无 sys_user.dept_id/无 sys_workstation.org_id 引用)
  **found:** b4fbab6d code 改 ROOT_DUP_20260817 → c64513be 恢复 code='ROOT' → 物理删除 b4fbab6d,事务提交成功。验证: live ROOT 行=1 且锚定 c64513be("若依科技有限公司");悬挂 parent_id 活行=0。用户重排后的树完整保留(ancestors 无需改写)。下次启动: ROOT code→c64513be 领养,SHENZHEN code→1b52ace0 领养,子部门同理,幂等闭环。
  **implication:** 议题2 代码+数据双层修复完成;scripts/deptfix 与 scripts/deptprobe 已删除。

- **timestamp:** 2026-08-17T01:30:07Z
  **checked:** backend connectivity via curl (127.0.0.1:9000 / localhost:9000 public-key)
  **found:** Both endpoints return HTTP 200 with public key in ~1ms (backend process PID 26580 listening on 0.0.0.0:9000 and [::]:9000).
  **implication:** Backend is alive and responsive for simple endpoints; not a total network/process failure.

- **timestamp:** 2026-08-17T09:06-09:07
  **checked:** `logs/app.log` for `/system/my-menus`, `/system/my-menus/all`, `/system/my-menus/permissions`
  **found:** Requests ARE reaching backend from client_ip `::1` (browser via Vite proxy). Multiple `POST /api/v1/system/my-menus` and `/api/v1/system/my-menus/all` returned 500 after latencies of ~30020ms; some returned 200 after 15-27s. `/permissions` returned 200 in 3-10s.
  **implication:** Login failure is caused by backend handler slowness, not DNS/connectivity failure.

- **timestamp:** 2026-08-17T09:06-09:07
  **checked:** backend slow query warnings around login window
  **found:** `SELECT * FROM "sys_user_role" WHERE user_id = '...'` took 3.3s and 6.8s for 1 row; `SELECT id, parent_id FROM "sys_menu" WHERE ...` took 3.2s for 239 rows; `SELECT count(*) FROM "sys_rpa_executions" WHERE status = 'pending'` took 5.0s.
  **implication:** Individual DB queries to remote Supabase are extremely slow, even for small result sets.

- **timestamp:** 2026-08-17
  **checked:** `configs/config.yaml` database configuration
  **found:** Backend connects to remote Supabase PostgreSQL (`db.bkixsntumwntnwpxavfu.supabase.co:5432`, sslmode=require). `max_open_conns` was bumped to 25 with comment "Phase 62-DBG-01: 10→25 缓解 my-menus 池饥饿".
  **implication:** Known my-menus performance issue; previous mitigation increased pool size but did not fix query latency.

- **timestamp:** 2026-08-17
  **checked:** `internal/models/user.go` UserRole and `internal/models/role.go` RoleMenu GORM tags
  **found:** Neither `UserRole` nor `RoleMenu` define any `index` tags. Only `type:uuid;not null`.
  **implication:** `sys_user_role` and `sys_role_menu` lack indexes on the foreign-key columns used by `GetUserMenus`, forcing full table scans amplified by remote DB network latency.

- **timestamp:** 2026-08-17
  **checked:** `internal/services/system/menu_service.go` GetUserMenus implementation
  **found:** Sequential queries: (1) `sys_user_role` by user_id, (2) `sys_role_menu` DISTINCT menu_id by role_id IN, (3) `sys_menu` ancestor lookup, (4) `sys_menu` details by id IN. A 2026-08-14 comment notes an N+1 fix replaced per-menu ancestor queries with a single full-table load.
  **implication:** Even after the N+1 fix, the remaining sequential scans on unindexed join tables are slow enough to exceed 30s.

- **timestamp:** 2026-08-17
  **checked:** `xingran-react-frontend/src/lib/api.ts` axios config
  **found:** `timeout: 30000` (30 seconds).
  **implication:** Backend latencies of ~30020ms exactly match axios timeout, triggering `ECONNABORTED` → "请求超时，请检查网络连接".

- **timestamp:** 2026-08-17
  **checked:** `logs/app.log` for `context canceled` around my-menus
  **found:** GORM errors on `sys_menu` single-row lookups with "context canceled" after hundreds of ms to 19s, matching client-side abort.
  **implication:** Client aborts the request, backend context is canceled, and gin logs the request as 500 internal server error.

- **timestamp:** 2026-08-17
  **checked:** frontend source for `utils.js` and Baidu Maps script loading
  **found:** No `utils.js` source references. Baidu Maps SDK is loaded dynamically only on the 3D building spaces page (`BaiduMapScript.tsx`), not on login.
  **implication:** `ERR_NAME_NOT_RESOLVED` at `utils.js:1` is likely a browser extension, source-map artifact, or unrelated external resource; it is not the cause of the login timeout.

- **timestamp:** 2026-08-17T11:11:58-11:13:19
  **checked:** 用户提供的新后端日志（应用 Migration 206 修复之后）
  **found:** `sys_user_role` 单条查询仍耗时 17.07s；`sys_role_menu` DISTINCT 查询 26.99s；`sys_menu` IN(230 UUIDs) 查询 54.8s 后 context canceled；`/system/my-menus` 与 `/my-menus/all` 均 latency≈60031-60035ms 返回 500（命中新的 60s axios 超时）。
  **implication:** 索引修复未生效或未真正应用到远程 DB；超时调大只是让失败从 30s 推迟到 60s。

- **timestamp:** 2026-08-17T11:11:58-11:13:19
  **checked:** 新日志中与登录无关的查询
  **found:** `INSERT INTO sys_logininfor` 1.9s；`UPDATE sys_rpa_workers` 心跳 5.1s；`SELECT count(*) FROM sys_rpa_executions WHERE status='pending'` 7.4s；`SELECT id,parent_id FROM sys_menu`(239行) 14.7s/25.7s。
  **implication:** 系统性 DB 延迟，不是单一查询形状问题。指向连接层（直连 vs pooler、sslmode、TLS 握手开销、conn_max_lifetime）、连接池排队（25 连接被慢查询占用）或 Supabase 实例限速。

- **timestamp:** 2026-08-17T11:11:58-11:13:19
  **checked:** 单次登录流程内重复查询
  **found:** 同一次登录中 `sys_user_role` 被查询 4+ 次、`sys_role_menu` 4+ 次、`sys_menu` 全量加载 3+ 次。
  **implication:** handler / service / 权限中间件可能各自独立解析菜单，未复用缓存或上下文结果，慢查询被成倍放大。

- **timestamp:** 2026-08-17T11:11:58-11:13:19
  **checked:** L2WriteWorker 日志
  **found:** `[L2WriteWorker] Worker-N: 任务已取消，跳过 key=menu:user:perms:...` / `menu:user:menus:...` / `menu:user:all:...`。
  **implication:** 异步 L2 缓存写入使用了请求 context，客户端中止后写入被取消 → 缓存永远写不进去 → 下次请求重新计算 → 恶性循环。写入应使用 detached context（context.Background() + 独立超时）。

## Eliminated

- **timestamp:** 2026-08-17T11:20 (Round 2)
  **checked:** H5 — 运行中后端进程与启动日志（netstat/curl/进程枚举/app.log）
  **found:** (a) 后端当前未在运行（:9000 无监听）；(b) `xingran-backend.exe` 编译于 2026-08-13 23:51，早于修复；(c) app.log 显示 2026-08-17 有多次重启（08:31 bind 失败、09:03、10:17、10:23、10:30），但全部启动块都出现 `[SKIP_AUTOMIGRATE=true] 跳过 AutoMigrate,改用 model 派生 DDL 补建(dev 旁路)` 警告，且**整个 app.log 中没有任何 `[迁移] 206` 行**（对比 08-13/08-14 启动有 `[迁移] 204` 等行）；(d) `.env` 确认 `SKIP_AUTOMIGRATE=true`。
  **implication:** `internal/core/db/database.go:404` 的 SKIP_AUTOMIGRATE 分支直接 `return nil`，跳过整个 postgres 迁移块（442-476 行，含 Migrate206）和 AutoMigrate 本身 → Migration 206 与模型层 gorm index 标签都永远不会在该环境生效。即使重新编译重启也没用。

- **timestamp:** 2026-08-17T11:26 (Round 2)
  **checked:** H5 — 只读探针 `scripts/idxprobe` 直连远程 DB 查 pg_indexes
  **found:** 三个 Migration 206 索引全部 **MISSING**；更严重的是 `sys_user_role` 和 `sys_role_menu` **没有任何索引**（连 pkey 都没有），`sys_menu` 仅有 pkey + idx_sys_menu_deleted_at。实际连接目标由 .env 覆盖为 `aws-0-ap-southeast-1.pooler.supabase.com`（pooler，项目 ovgfhrphadkvdkareigj），而非 config.yaml 中的直连 host。
  **implication:** H5 被 DB 端直接证据确认：索引修复从未应用到远程库。

- **timestamp:** 2026-08-17T11:26 (Round 2)
  **checked:** H6 — 探针延迟测量
  **found:** 热连接 `SELECT 1` RTT=146ms；冷连接首查 ~1s；**新建连接握手（TLS+auth，pooler）耗时 4.5-4.8s**；小表 count（sys_user_role=1 行 / sys_role_menu=239 行 / sys_menu=275 行）0.27-1.7s。
  **implication:** 表极小，即使没有索引，空闲池上单查询也只需 ~0.15-1s；不可约基线延迟低。日志中 17-55s 的耗时必然由**池耗尽排队 + 新连接 4.7s 握手**主导（GORM 计时含池获取等待）。H6 是放大器而非主因。

- **timestamp:** 2026-08-17T11:35 (Round 2)
  **checked:** H8 — `pkg/cache/redis.go` MultiLevelCache.Set 与 `pkg/cache/l2_writer.go` processTask
  **found:** **Smoking gun** — `redis.go:633-635`：`enqueueCtx, cancelEnqueue := context.WithTimeout(context.Background(), m.l2Writer.GetFallbackTimeout()); defer cancelEnqueue()`。`Set()` 返回瞬间 defer 取消 enqueueCtx；任务结构体（buildTask:239）保存该 ctx；worker 出队后 `processTask`（l2_writer.go:292-298）首先检查 `task.ctx.Done()` —— 必然已关闭 → 丢弃并打日志 `任务已取消，跳过 key=...`。同时 L1 写入（redis.go:623 `m.l1Cache.Set(ctx, ...)`）用的是**请求 ctx**，客户端中止后 L1 也写不进。
  **implication:** H8 确认，且机制比原假设更糟：不是"用了请求 ctx"，而是 P1 修复引入的 defer-cancel bug 使**几乎 100% 的 L2 异步写入被丢弃**（只有 Set 返回前被 worker 抢到的极端竞态才幸存）。menu:user:* 缓存永远进不了 Redis；请求中止时 L1 也写不进 → 缓存双层永久 miss → 恶性循环的引擎。

- **timestamp:** 2026-08-17T11:40 (Round 2)
  **checked:** H7 — 菜单解析调用链
  **found:** (a) 前端登录后串行调 3 个端点（/my-menus、/my-menus/all、/my-menus/permissions），各有独立缓存 key，缓存 miss 时每个都完整跑一遍 sys_user_role→sys_role_menu→sys_menu 链；(b) `pkg/middleware/permission.go:210` `isSuperAdmin` 每个受保护请求都跑**无缓存**原生 JOIN（sys_user_role×sys_role）；非 admin 还要 `checkUserPermission`→`pkg/permission.GetUserPermissions`（同样无缓存、全链路）按权限逐个重试；(c) 260814 已加 menuCacheService GetOrSet 缓存覆盖，但因 H8 写入恒失败而形同虚设。
  **implication:** H7 确认：单次登录 + 首批页面加载产生 10+ 条顺序慢查询，全部打在无索引小表 + 高延迟远程库上，池被占满后互相排队，GORM 耗时 17-55s 主要是排队等待。

- **timestamp:** 2026-08-17T11:55 (Round 3, FIX)
  **checked:** H8 修复 — l2_writer.go buildTask 改为 `context.WithTimeout(context.WithoutCancel(ctx), taskQueueTTL())` detach 任务 ctx,cancel 存于 task 并由 processTask(defer)/Enqueue/TryEnqueue 失败路径释放;redis.go Set 移除 defer-cancel enqueueCtx、L1 写入与 Get 回填改 `context.WithoutCancel(ctx)`。
  **found:** `go build ./...` / `go vet` 通过;`go test ./pkg/cache/` 全 PASS(6.67s),新增回归测试 TestL2WriteWorker_TaskSurvivesCallerCancel PASS(stats: enqueued=2, completed=2, dropped=0) — 旧代码下该测试必然失败(ctx 取消 → 前置检查丢弃 → setCount=1)。
  **implication:** H8 机制闭环被打破:任务 ctx 不再随请求取消,L2 写入只在真正积压超 TTL(≥30s)时才丢弃。

- **timestamp:** 2026-08-17T11:56 (Round 3, FIX)
  **checked:** H5 修复 — 临时脚本 scripts/idxapply(沿用 idxprobe 连接模式)对远程 Supabase 重放 Migration 206 + 补 pkey,随后 pg_indexes 只读验证。
  **found:** 重复行检查 sys_user_role=0 / sys_role_menu=0;3 个索引 CREATE INDEX IF NOT EXISTS 全部 OK(5.3s/266ms/142ms);sys_user_role_pkey(user_id,role_id) 与 sys_role_menu_pkey(role_id,menu_id) ADD CONSTRAINT 全部 OK。pg_indexes 最终状态:sys_user_role 有 pkey+idx_sys_user_role_user_id_role_id;sys_role_menu 有 pkey+idx_sys_role_menu_role_id_menu_id;sys_menu 有 pkey+idx_sys_menu_deleted_at+idx_sys_menu_parent_status_visible。
  **implication:** H5 消除:join 表外键列全部有索引,全表扫描路径关闭。pooler transaction 模式下单语句 DDL 可行(验证了 blind spot)。

- **timestamp:** 2026-08-17T11:56 (Round 3, FIX)
  **checked:** H7 修复 — pkg/middleware/permission.go isSuperAdmin/checkUserPermission 加进程内 30s TTL 缓存(sync.Map + 过期时间戳;不用 core.Cache 因 MultiLevelCache L1 硬编码 5min 会放大 TTL 语义);H6 缓解 — database.go 池保活 goroutine(30s 并发 ping ≤min(MaxIdleConns,4) 连接)。
  **found:** `go test ./pkg/middleware/` PASS(0.248s,既有 inherit/query 测试全绿 — 测试用随机 UUID,缓存无串扰);`go test ./internal/core/db/...` PASS(2.885s);`go test ./internal/middleware/` PASS;`go test ./internal/services/system/ ./internal/core/` PASS。
  **implication:** 权限中间件每请求 2-6 条 SQL 降为 30s 内 0 条(命中时);池保活消除低流量期 4.7s 冷握手。

- **timestamp:** 2026-08-17T11:57 (Round 3, CLEANUP)
  **checked:** 清理诊断/修复临时文件。
  **found:** scripts/idxprobe 与 scripts/idxapply 目录已删除;`go build ./...` 最终通过。
  **implication:** 仓库无遗留探针代码。

## Cross-cutting 量化结论

- 不可约部分：远程 pooler RTT ~150ms，空闲池小查询 ~0.15-1s → 单登录 ~10 条顺序查询的**理论下限 ~2-10s**（无缓存、无索引也不致命，因为表只有 1-275 行）。
- 自残部分（主因）：H8 使缓存永远 miss → H7 使每请求重复 3-4 倍查询 → 25 个连接被 3-25s 查询占满 → 排队 + 4.7s 握手 → 60s 超时 → 客户端中止 → L1/L2 写入再被取消 → 下次从零开始。
- 修复优先级：H8（defer-cancel bug）> H7（中间件缓存）> H5（应用索引，需绕过/重放迁移）> H6（调优，可选）。

## Eliminated

- **hypothesis:** H1 localhost resolution failure / proxy target unreachable
  **evidence:** curl to both `127.0.0.1:9000` and `localhost:9000` works. Backend logs show requests arriving from `::1` with correct paths. `netstat` shows backend listening on IPv4 and IPv6.
  **timestamp:** 2026-08-17

- **hypothesis:** H2 backend handler hangs/times out (partial)
  **evidence:** Confirmed the handler IS slow, but not due to infinite loop or deadlock. Slowness is caused by specific slow DB queries on remote Supabase. Simple endpoints (`/auth/public-key`, unauthenticated `/my-menus`) respond instantly.
  **timestamp:** 2026-08-17

- **hypothesis:** H3 external script / map SDK DNS error causes login failure
  **evidence:** Baidu Maps only loads on 3D building page. No `utils.js` references in source. The failing `/my-menus` request reaches backend and returns 500/200, proving the browser can resolve and connect to the API origin.
  **timestamp:** 2026-08-17

- **hypothesis:** H4 browser accessed via unresolvable hostname
  **evidence:** The actual API calls are reaching backend from `::1` and returning data; browser hostname resolution is not blocking API calls.
  **timestamp:** 2026-08-17

## Resolution

**root_cause (Round 2, CONFIRMED):** 登录超时的引擎是 **L2 缓存写入的 defer-cancel bug**：`pkg/cache/redis.go:633-635` `MultiLevelCache.Set` 在 `defer cancelEnqueue()` 后才把任务交给 worker，worker 出队时 `l2_writer.go:292-298` 的前置 `task.ctx.Done()` 检查必然命中 → 几乎 100% 的 L2 异步写被丢弃（日志 `任务已取消，跳过 key=menu:user:*`），L1 又因使用请求 ctx 在客户端中止后写不进 → 菜单/权限缓存双层永久 miss。于是每个请求都重跑 `pkg/middleware/permission.go` 的无缓存 `isSuperAdmin`/`GetUserPermissions` 和 3 个 my-menus 端点的完整 sys_user_role→sys_role_menu→sys_menu 查询链（10+ 条顺序查询），打在无索引（`sys_user_role`/`sys_role_menu` 连 pkey 都没有——Migration 206 因 `.env` 的 `SKIP_AUTOMIGRATE=true` 在 `database.go:404` 被整体旁路，从未在远程库执行，探针已确认 3 个索引 MISSING）的远程 Supabase pooler（RTT 150ms、新连接握手 4.7s）上 → 25 个池连接被 3-25s 查询占满 → 排队使 GORM 耗时飙到 17-55s → axios 60s 中止 → ctx canceled → 缓存写入再次失败，恶性循环自维持。第一轮修复（索引迁移+调大超时）未生效的原因：迁移被 SKIP_AUTOMIGRATE 旁路（H5），且超时调大只把失败从 30s 推迟到 60s，未触碰缓存失效引擎（H8）。

**root_cause (Round 4, CONFIRMED):** 两个新根因 —
- **议题1 (my-menus 500 @ 10.010s):** 认证中间件的黑名单检查 fail-closed 且直达远程 Redis。`pkg/middleware/auth.go:49` 把 `IsBlacklisted` 的任何 error 都转 500;`MultiLevelCache.Exists`(redis.go)绕过 L1 直达 L2(Upstash TLS, poolTimeout=10s);黑名单键几乎永不在 L1(未拉黑不回填),故每个请求都付一次远程往返,网络退化窗口内阻塞至 poolTimeout → error → 500(延迟精确 10010ms、窗口内零 GORM 日志双重证实)。同窗口其他 5 个并发请求先抢到池连接而幸存。
- **议题2 (启动 23505):** `ensureDept` 的幂等键与唯一约束不匹配 — 查询用 (dept_name, parent_id),唯一索引却在 dept_code 上。旧种子根 c64513be 被**重命名为"若依科技有限公司"且 dept_code 清空**(探针证伪"物理删除"假设),(name, parent) 查询失配 → 重复建根 → SHENZHEN INSERT 撞 idx_sys_dept_dept_code。

**fix (Round 4 已应用,2026-08-17):**
1. **议题1-A(代码):** `pkg/middleware/auth.go` — 黑名单检查 fail-open:err → `applogger.Warnf` 放行,JWT 7200s 过期兜底,黑名单强制力仅在缓存故障窗口降级。
2. **议题1-B(代码):** `internal/services/token_blacklist_service.go` — 进程内 30s 负缓存("未拉黑"判定)+ 30s 错误熔断窗口(窗口内直接放行,不再每请求阻塞 10s);AddToBlacklist/RemoveFromBlacklist 立即使对应 token 负缓存失效;负缓存容量上限 1024 + 过期清理。
3. **议题1-C(代码):** `pkg/cache/redis.go` MultiLevelCache.Exists L1 前置(L1 hit→true 恒正确;miss 回源 L2 无假阴性)。
4. **议题2-代码:** `internal/core/db/init_data.go` ensureDept 两段式查询:① dept_code ② (dept_name, parent_id) 兼容历史无编码行(C5 语义不变)。
5. **议题2-数据(带外,已提交):** 恢复 c64513be 的 dept_code='ROOT';物理删除事故重复根 b4fbab6d(前置校验零引用)。验证: ROOT 唯一锚定 c64513be,悬挂 parent=0。
6. **清理:** scripts/deptfix、scripts/deptprobe 已删除。

**verification (Round 4, build/test/DB 层 — 运行时验证待用户确认):**
- `go build ./...` PASS;`go vet`(middleware/cache/services/db) PASS
- `go test ./internal/services/ -run 'TestIsBlacklisted|TestAddToBlacklist'` PASS(4 个新回归测试: 负缓存/正例不缓存/熔断 fail-open/拉黑失效)
- `go test ./internal/core/db/` PASS — 含新增 TestCreateDefaultDept_ParentDriftRecovers(精确复现事故场景,旧代码必 23505)
- `go test ./pkg/cache/ ./pkg/middleware/ ./internal/middleware/` PASS
- 远程 DB: ROOT 行=1 锚定 c64513be;悬挂 parent_id 活行=0
- 既有失败(非本次引入,零交集): internal/services 的 TestCollectBoardsInto_*/TestCollectDeviceInfo_*/TestNormalizeMACAddress
- 待用户运行时验证: 重启无 23505;登录链路 Redis 抖动不再 500;部门树无重复根

**files_changed (Round 4, 修复):**
- `pkg/middleware/auth.go`: JWTAuthWithBlacklist 黑名单检查 fail-open + Warn 日志
- `internal/services/token_blacklist_service.go`: 30s 负缓存 + 30s 错误熔断 + 拉黑/移除时失效
- `internal/services/token_blacklist_service_test.go`: 新增 4 个回归测试
- `pkg/cache/redis.go`: MultiLevelCache.Exists L1 前置
- `internal/core/db/init_data.go`: ensureDept dept_code 优先两段式查询
- `internal/core/db/init_data_test.go`: 新增 TestCreateDefaultDept_ParentDriftRecovers
- 远程 DB(带外,非代码): c64513be 恢复 dept_code='ROOT';删除重复根 b4fbab6d
- 已删除: `scripts/deptfix/main.go`、`scripts/deptprobe/main.go`(一次性修复/探针)

**fix (Round 3 已应用,2026-08-17 — 运行时已验证):**
1. **H8(主因,代码)**: `pkg/cache/l2_writer.go` — `l2WriteTask` 增加 `cancel` 字段;`buildTask` 用 `context.WithTimeout(context.WithoutCancel(ctx), taskQueueTTL())` 统一 detach 任务 ctx(TTL = max(6×WriteTimeout, 30s));`processTask` defer 释放 cancel;`Enqueue`/`TryEnqueue` 入队失败路径释放 cancel(无泄漏)。`pkg/cache/redis.go` — `Set` 移除 defer-cancel enqueueCtx(直接透传 ctx,detach 收敛到 buildTask);L1 写入(Set)与 L1 回填(Get)改 `context.WithoutCancel(ctx)`,客户端中止不再阻止缓存落盘。
2. **H5(DB,带外迁移)**: 临时脚本 `scripts/idxapply`(用后已删)直连远程 Supabase 重放 `Migrate206AddUserRoleRoleMenuIndexes` 的 3 个索引(CREATE INDEX IF NOT EXISTS),并为 `sys_user_role(user_id, role_id)` / `sys_role_menu(role_id, menu_id)` 补 PRIMARY KEY(前置重复行检查=0)。pg_indexes 已验证全部存在。
3. **H7(代码)**: `pkg/middleware/permission.go` — `isSuperAdmin` / `checkUserPermission` 加进程内 30s TTL 缓存(sync.Map,按 userID / userID|permission 隔离,DB 错误不缓存);提供 `InvalidateMiddlewarePermCache(userID)`(当前未接线,30s 陈旧窗口已接受并注释 — 远小于 JWT 7200s 与 menuCacheService 30min)。不用 core.Cache 的原因:MultiLevelCache L1 硬编码 5min TTL 会静默放大语义。
4. **H6(代码,低风险可选)**: `internal/core/db/database.go` — 池保活 goroutine(30s 间隔并发 ping min(MaxIdleConns,4) 个连接,消除 Supabase pooler 空闲回收后的 4.7s 冷握手;Close 时优雅停止,nil-guard 兼容测试手工构造的 Database)。
5. **清理**: `scripts/idxprobe`、`scripts/idxapply` 已删除。

**verification (Round 3, build/test/DB 层 — 运行时验证待用户确认):**
- `go build ./...` PASS;`go vet ./pkg/cache/ ./pkg/middleware/ ./internal/core/db/` PASS
- `go test ./pkg/cache/` PASS — 含新增回归测试 `TestL2WriteWorker_TaskSurvivesCallerCancel`(enqueued=2, completed=2, dropped=0;旧代码下 dropped=1 必失败)
- `go test ./pkg/middleware/` PASS;`go test ./internal/core/db/...` PASS;`go test ./internal/middleware/` PASS;`go test ./internal/services/system/ ./internal/core/` PASS
- 远程 DB pg_indexes: sys_user_role {sys_user_role_pkey, idx_sys_user_role_user_id_role_id};sys_role_menu {sys_role_menu_pkey, idx_sys_role_menu_role_id_menu_id};sys_menu {sys_menu_pkey, idx_sys_menu_deleted_at, idx_sys_menu_parent_status_visible}
- 待用户运行时验证:重启后端 → 登录 → app.log 无 `任务已取消,跳过 key=menu:user:*`、无 join 表 GORM 慢查询;my-menus 秒级返回

**files_changed (Round 3, 修复):**
- `pkg/cache/l2_writer.go`: l2WriteTask + cancel 字段;buildTask detach ctx(WithoutCancel + taskQueueTTL);processTask defer cancel;Enqueue/TryEnqueue 失败路径 cancel
- `pkg/cache/redis.go`: MultiLevelCache.Set 移除 defer-cancel enqueueCtx;L1 Set/Get 回填改 WithoutCancel
- `pkg/cache/l2_writer_test.go`: 新增 TestL2WriteWorker_TaskSurvivesCallerCancel 回归测试
- `pkg/middleware/permission.go`: isSuperAdmin/checkUserPermission 30s 进程内缓存 + InvalidateMiddlewarePermCache;checkUserPermission 原逻辑移入 checkUserPermissionUncached
- `internal/core/db/database.go`: Database 结构体 + keepaliveStop/Done 字段;NewDatabase 启动池保活;Close 优雅停止;新增 poolKeepaliveLoop/pingPoolOnce
- 远程 DB(带外,非代码): 3 个 Migration 206 索引 + 2 个 join 表 pkey
- 已删除: `scripts/idxprobe/main.go`(诊断探针)、`scripts/idxapply/main.go`(一次性迁移脚本)

## Notes

- Do not apply code fixes until root cause is confirmed.
- User explicitly asked to combine with backend logs.
- Previous mitigation (max_open_conns 10→25) only relieved connection-pool starvation; it did not address missing indexes or remote DB latency.
