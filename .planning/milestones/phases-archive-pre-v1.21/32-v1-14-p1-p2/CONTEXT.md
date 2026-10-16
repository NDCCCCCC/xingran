# Phase 32 — v1.14 P1 重构与 P2 架构优化 — Context

## 来源

本 phase 直接源自 [20260612 后端代码审查报告](../../reviews/20260612-backend-code-review.md)
中**全部非 P0**的发现。P0 部分由 Phase 31（F-14 connection_pool + F-17 ConfigUpdateRequest）以及前序 quick task 收尾。

## 任务清单（与 ROADMAP.md Phase 32 Requirements 一一对应）

### 🔒 P1 安全加固（7 项）

| ID | 文件:行 | 问题 | 修复要点 |
|---|---|---|---|
| P1-S1 | `pkg/crypto/sm2_jwt.go:222-266` | 手写 JWT 解析未校验 `alg` 头 | 解析前白名单校验 `alg`，拒绝 `none`/不一致算法 |
| P1-S2 | `pkg/crypto/request_encryption.go:88-103` | timestamp 窗口 ±300s 过宽 | 收紧到 ±60s 并提供 `security.replay_window_sec` 配置 |
| P1-S3 | `pkg/crypto/nonce_storage.go` | `cleanupExpiredNonces` 定义但未调用 | 启动时起 ticker goroutine（带 ctx 取消） |
| P1-S4 | `pkg/middleware/permission.go:106-118` | 子菜单自动继承父菜单权限 | 取消自动继承，权限按菜单显式声明 |
| P1-S5 | `internal/core/security/password.go:25` | PBKDF2 迭代 1000 轮 | 提升到 ≥600000；新老哈希共存 + 登录时按需迁移 |
| P1-S6 | `internal/core/security/password.go:145-166` | 随机密码模偏置 | 改 `crypto/rand` 拒绝采样或字符表整除分组 |
| P1-S7 | `internal/services/operations/excel_handler.go:67-72` | 仅校验后缀 | 校验 ZIP magic `PK\x03\x04` + content-type |

### ⚡ P1 并发与一致性（6 项）

| ID | 文件:行 | 问题 | 修复要点 |
|---|---|---|---|
| P1-C1 | `internal/services/addomain/sync.go:35` 等 | 多入口同步并发无互斥 | `singleflight.Group` 按 config_id 去重 |
| P1-C2 | `internal/services/addomain/group_sync_service.go:336-376` | LDAP 不可达全表误删 | 删除比例阈值（默认 30%）超过即拒删 + 告警 |
| P1-C3 | `internal/services/addomain/dept_ou_mapper.go:60-100` | UpsertMapping 两步独立事务 | 单 `Transaction` 包裹 delete+insert |
| P1-C4 | `internal/collectors/port_collector.go:353-381` | 端口状态逐条插入 | `clause.OnConflict` 批量插入，batch=500 |
| P1-C5 | `internal/websocket/notice_hub.go` | 缺 readPump 僵尸连接 | 添加 readPump + ping/pong + ReadDeadline |
| P1-C6 | `internal/services/operations/excel_service.go:905-906` | `validateUniqueness` N+1 | 收集所有值后 `WHERE col IN (?)` 一次查询 |

### 🐛 P1 业务逻辑（2 项）

| ID | 文件:行 | 问题 | 修复要点 |
|---|---|---|---|
| P1-B1 | `internal/services/system/config_service.go:91-99` | 加密开关改后需重启 | `Update` 完成后立即调 `InvalidateConfigCache` |
| P1-B2 | `internal/services/system/user_service.go:404` | `buildDepartmentPaths` 重复调用 | 删除第二次调用 |

### 🛠️ P2 架构债（精选 8 类）

| ID | 范围 | 改进点 |
|---|---|---|
| P2-A1 | `internal/core/core.go` | 拆 god struct 为 `CoreServices` / `CoreInfra` |
| P2-A2 | `cache_keys.go` + `data_cache_service.go` | 合并到唯一 `cache_keys.go` |
| P2-A3 | `user_service.go` vs `user_service_optimized.go` | 删除旧版，router 切到 optimized |
| P2-A4 | `internal/core/db/migrations/` | 027/028/029/030/031/036 重号文件重命名 + 来源注释 |
| P2-A5 | 错误处理 | `role_service` 等 `fmt.Errorf` → `apperrors`，Handler 统一映射 |
| P2-A6 | AD 模块测试 | 补 `ldap_client` Connect/Bind/Search mock、`group_sync`、`user_ou_service`；删除/补全 `stripBaseDN_test.go`、`dept_ou_mapper_test.go` |
| P2-A7 | `internal/device/` 子进程 | process group + Wait + 僵尸进程定时清理 |
| P2-A8 | Excel 导入事务 | `processThreeLevelDepartments` 等子流程纳入事务 |

## 范围之外（Out of Scope）

- 已由 Phase 31 处理的 P0 收尾项（F-14 connection_pool / F-17 ConfigUpdateRequest）
- 已通过 quick task 修复的 P0 项（见 `.planning/debug/resolved/`）
- 报告中"良好实践"章节列举的不需要修改的项

## 依赖

- **强依赖**: Phase 31 完成（P0 收尾确认）
- **建议先做**: P1 安全加固优先于 P2 架构债

## 验收

- 所有 P1/P2 ID 在 commit message 中可追溯（如 `fix(security): P1-S4 取消子菜单权限继承`）
- `go vet ./...` 零警告
- 关键路径单测覆盖 ≥70%（AD/JWT/password 模块）
- 与 20260612 报告同维度复审通过

## 后续

运行 `/gsd-plan-phase 32` 生成各 Wave 的 detailed PLAN 文件。
