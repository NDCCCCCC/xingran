# v1.51 集成验证窗口准备包（DEBT-03/04 + TIDY-01 + Phase 121-127）

> 维护人：qualguard ｜ 创建：2026-11-01 ｜ 窗口开启条件：Phase 125/126 合并后
>
> 用法：窗口开启时按顺序执行 §1→§5；每项标注【执行人】【前置】【预期输出】。
> 本清单只读不写业务代码；配套产物：
> - `scripts/integration/seed_port_write_audit.sql`（§4 审计页测试数据）
> - `scripts/integration/smoke-v1.51.sh`（§2/§3 冒烟脚本，bash + curl）

## 0. 环境准备

【执行人】lead 或指派成员　【前置】仓库 feat/xingran-sync 最新；无并行线未提交变更（`git status` 干净，portwrite 半成品已合入或 stash）

```bash
# sqlite 模式启动（窗口冒烟用，不依赖 PG 实例）
set DB_TYPE=sqlite
set DB_PATH=data/integration-v151.db   # 干净库，窗口内一次成型
go build -o xingran-backend.exe ./cmd/main.go
.\xingran-backend.exe                  # 启动即自动跑全部 migrations（含 228/229/230）
```

预期：启动日志出现 `Migration 228/229/230` 三行成功记录，无 `非阻断…失败` 告警。

## 1. Migration 三步验证（sqlite 真库）

### 1.1 migration 228（TIDY-01 服务端排序 4 页面菜单）

【执行人】任意成员　【前置】§0 启动完成　【工具】sqlite3 data/integration-v151.db

```sql
-- 步骤 A：4 行菜单可见且启用（visible=1, status=0）
SELECT component, visible, status FROM sys_menu
 WHERE component IN ('duty/pools/index','duty/schedules/index',
                     'knowledge/articles/index','network/templates/index');
-- 预期：恰好 4 行，全部 visible=1 status=0（228 的三态修复语义）

-- 步骤 B：webfe 三步验证（前端侧，人工或 playwright）
--   tree-select 含 4 节点 → 4 路由可达 → 列表排序请求带 orderByColumn/isAsc
```

预期输出：4×`1|0`；前端排序请求 query 带 `orderByColumn=...&isAsc=...`。

### 1.2 migration 229（端口写审计明细页菜单）

```sql
SELECT component, visible, status, perms FROM sys_menu
 WHERE component = 'network/port-audits/index';
-- 预期：1 行，visible=1, status=0, perms='network:port:write'（与写同组非只读码）
```

### 1.3 migration 230（RPA Worker Token 字段）

```sql
PRAGMA table_info(sys_rpa_workers);   -- sqlite；PG 用 information_schema.columns
-- 预期：含 token / token_expires_at / last_token_rotation 三列；
--       存量行三列为 NULL（宽限期兼容，老 Worker 不受影响）
```

### 1.4 migration 231（DEBT-02 审计 operator 复合索引，portwriter）

```sql
-- 索引存在性（双方言）
SELECT name FROM sqlite_master WHERE type='index' AND name='idx_port_write_audit_operator_created';
-- PG: SELECT indexname FROM pg_indexes WHERE indexname = 'idx_port_write_audit_operator_created';
-- 预期：1 行；索引定义为 (operator, created_at)——支撑审计页按操作人+时间过滤
-- 生效验证：§4 seed 后执行 EXPLAIN QUERY PLAN SELECT * FROM sys_port_write_audit
--          WHERE operator='tester' ORDER BY created_at DESC; 应走该索引而非全表扫描
```

## 2. DEBT-03 Worker 认证宽限期冒烟

【执行人】qualguard　【前置】§0；`rpa.worker.auth.auth_required=false`（默认）

```bash
bash scripts/integration/smoke-v1.51.sh worker-auth
```

覆盖矩阵（脚本自动断言）：
| 场景 | 预期 |
|---|---|
| 心跳无 X-Worker-Token（宽限期） | HTTP 200 + WARN 日志 |
| 注册带合法 X-Provisioning-Token | 200 且响应 data.token 以 `wk_` 开头 |
| 注册带错误 provisioning token | HTTP 401 |
| 强制模式（临时改 auth_required=true 重启） | 无 token 心跳 401；:id 与凭证不符 403 |

切强制模式为**手工步骤**（需改配置重启），脚本只跑宽限期矩阵并打印切强制模式的操作提示。

> ✔ **KNOWN ISSUE 已修复（2026-11-01，task-15）**：sqlite 模式下「注册换发」
> 曾失败（`no such function: gen_random_uuid`）。根因是 worker_service.go
> Register 原生 SQL 里的 PG 专属函数（gen_random_uuid/NOW/RETURNING），并非
> migration_205（其本就有 isPostgreSQL 守卫，sqlite 分支跳过）。修复：Register
> 增加 sqlite 方言分支（GORM upsert，应用层 UUID/时间戳；capabilities 写
> []byte），PG 路径零变化。矩阵 4/4 PASS 已连续两轮 sqlite 实测；修复前记录：
> 2 PASS（宽限心跳 200、错 token 401）/ 2 FAIL（注册换发）。

## 3. Phase 121 dry-run 端到端冒烟

【执行人】qualguard　【前置】登录态 token；无需真设备

```bash
bash scripts/integration/smoke-v1.51.sh dry-run <TOKEN>
```

覆盖（错误路径即可验证契约，无需真设备）：
- `POST /api/v1/network/ports/write/preview`，`deviceId` 不存在 → 404「设备不存在」
- `action` 非法 / `vlanId` 越界（>4094）/ 缺 portIds → 400（各错误分支契约字段齐全：`code/message/timestamp/request_id`）
- 权限矩阵：无 `network:port:write` 权限账号 → 403
- 【SKIP 条件】真实设备实发验证不在本窗口（Phase 129 真机 UAT），脚本打印 SKIP 提示

## 4. 审计页测试数据 seed

【执行人】webfe 联调人　【前置】§0 库；【产物】`scripts/integration/seed_port_write_audit.sql`

```bash
sqlite3 data/integration-v151.db < scripts/integration/seed_port_write_audit.sql
```

内容：10 行代表性 `sys_port_write_audit`（固定 UUID 双方言通用）——
succeeded/failed 各 5，覆盖 shutdown/undo_shutdown/description/dot1x_enable/access_vlan，
含 beforeValue/afterValue JSON、commandSent、failureReason（仅 failed 行）。
**幂等**：`DELETE ... WHERE id IN (固定清单)` 后重插，可重复执行。
联调入口：菜单 `网络 → 端口写审计`（migration 229 注册的 component）。

## 5. 收尾核对

- [ ] §1.1-1.3 三步全绿
- [ ] §2 宽限期矩阵 4/4 通过；强制模式切换步骤已演练
- [ ] §3 错误路径契约字段齐全（code/message/timestamp/request_id）
- [ ] §4 审计页前端可分页/过滤（按 status、action）
- [ ] 全量 `bash .github/scripts/check-coverage.sh <profile> .coverage-threshold` PASS（≥76.85）
- [ ] 完成后由 lead 在窗口记录中勾销本清单并归档到 `.planning/`
