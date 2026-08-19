# Deferred Items — quick-260817-hfl

执行期发现但不属于本任务范围(或不阻塞 sqlite dev 链路)的事项:

| Item | 发现于 | 说明 | 建议处置 |
|------|--------|------|----------|
| `default_theme_service.go:157` 表名 typo `sys_user_preferences`(复数) | 协调者排查指引 | 实际表是单数 `sys_user_preference`;该 typo 在 PG 上同样必失败(pre-existing bug,非 sqlite 引入)。SyncUserThemeToDefault 为动作型端点,非登录首屏链路 | 独立 quick 任务修复(`Table("sys_user_preference")`);注意 SyncUserThemeToDefault 业务语义也需复核 |
| 个别 handler 原生 SQL 含 PG 方言(::uuid / ILIKE / pg_catalog) | PLAN 勘察已声明 | sqlite 下运行期报错 | 按需后续修(参考 quick-260814-211 模式) |
| 预存在 `internal/api/v1/auth` TestADLoginWithOUProcessing 失败 | Phase 62 已记录 | 与本任务无关 | 见 phase 62 deferred-items.md |

## 2026-08-17 追加(debug sqlite-startup-pg-only-errors 第 3 轮方言排查)

以下 PG-only SQL 方言点在 sqlite 路径必然报错(临时探针实测:`::` cast → unrecognized token ":";`NOW()`/`gen_random_uuid()` → no such function;`INTERVAL` → syntax error),均已实测确认但不属本轮"对账/monitor"修复范围:

| Item | 位置 | 触发条件 | 建议处置 |
|------|------|----------|----------|
| RPA worker Register 原生 SQL 用 `gen_random_uuid()` + `NOW()` | `internal/services/rpa/worker_service.go:122-138` | 外部 RPA worker 启动/重注册时(心跳路径不受影响,已验证 200) | sqlite 分支:Go 侧生成 UUID + 时间戳参数(RETURNING 在 SQLite 3.35+ 可用,ON CONFLICT 双方言兼容) |
| 楼层软删恢复原生 SQL `updated_at = NOW()` | `internal/services/operations/floor_service.go:100-107` | 创建楼层命中同 building_id+floor_no 的软删记录时 | `NOW()` → `CURRENT_TIMESTAMP`(双方言兼容,UTC 与 CDX-M-UTC NowFunc 对齐)或参数化时间 |
| AD 角色分配原生 SQL `VALUES (?, ?, NOW())` ×2 | `internal/services/system/user_sync_service.go:363`、`767` | sqlite 下执行 AD 用户同步分配角色时 | 同上 |
| `login_log_service.go:172` Clean 删 `sys_login_log` | `internal/services/monitor/login_log_service.go:172` | 清空登录日志端点;模型表实为 `sys_logininfor`(LoginLog.TableName),该 SQL 在 PG 上同样必失败 —— 双方言预存 bug | 独立 quick 任务修复为 `sys_logininfor` |
| 大范围 `::uuid`/`::text` cast(operations 模块列表/导出查询) | `workstation_service.go:17,69,255,273,436,446`、`floor_service.go:170-171,194,229-230,264`、`infopoint_service.go:133-135,182-185,212-214`、`server_room_service.go:137-138,209`、`room_device_service.go:106,125-126`、`excel_query_builder.go:222-236`、`building_service.go:153`、`workstation_device_service.go:402-488,1785-1843`、`reconciliation_service.go:353,371,380-383,392,485,779-780,1011`、`mac_history_query_service.go:890,954,1002`(EXTRACT…::bigint) | sqlite 下打开对应列表/导出页面即报 syntax error(用户尚未报,推测未点击这些页面) | 系统性方言治理任务:参考 `workstation_service.go:97-100` 与 `location_alias_service.go:26` 已落地的 `CAST(x AS TEXT)` 双方言写法(标准 SQL,PG/SQLite 行为一致),或按本 debug 会话的 `Dialector.Name()` 分支范式逐文件处理;mac_history_query_service 依赖 PG 分区表,属既有 deferred #4 范畴 |

本轮已修(对账/monitor 族):`fix_suggestion_service.go` Stats 6 Count(INTERVAL)、3 处 `su.id::text` JOIN、Apply `NOW() + INTERVAL '7 day'`、Rollback `> NOW()`;`fix_suggestion_generator.go` 非 PG 早退;`reconciliation_tasks.go` checkPortStatusDrift `::text` 方言分支。详见 `.planning/debug/sqlite-startup-pg-only-errors.md`。
