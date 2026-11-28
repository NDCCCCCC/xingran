# RPA 系统完成状态报告

> **文档状态**：`code-complete · verification-pending`
> **对齐基线**：v1.56（Phase 152~158 合入后）｜**上次重写**：2026-11-25（Phase 158 / CLEANUP-03 真相化）
> **口径声明**：本文档只陈述「代码里有什么」与「哪些已在本机真跑过」。
> 按 `.planning/notes/SHIPPED-DEFINITION.md`，**缺真实环境 UAT 证据的里程碑不得标 SHIPPED** ——
> 本 RPA 域当前为「代码完成、验证待办」，不是「已验收」。上一版报告（2025-02-25）声称
> 「100% 完成 / 可正常启动运行」且列出的 flow 组能力**已全部下线并删除**，故整体重写。

---

## 1. 现状总览

| 维度 | 现状 | 事实来源 |
|---|---|---|
| 后端服务文件 | 24 个（非测试） | `internal/services/rpa/*.go` |
| 后端 API 文件 | 11 个（非测试） | `internal/api/v1/rpa/*.go` |
| 已注册路由 | **64** 条（ai 11 / credentials 7 / executions 19 / schedules 7 / tasks 8 / workers 12） | `scripts/route-registry.json`（由 `node scripts/scan-routes.cjs` 生成） |
| 公开（免认证）路由 | 5 条（`/workers/register`、`/workers/:id/heartbeat`、`/workers/progress`、`/workers/liveview/frame`、`/workers/liveview/event`） | `rpa_router.go` `SetupPublicWorkerRouter` |
| 前端页面 | RPA 管理入口 + 任务 / 执行记录 / Worker 四个页面 + 实时画面组件 | `xingran-react-frontend/src/pages/operations/rpa/` |
| 表 | `sys_rpa_tasks` / `sys_rpa_workers` / `sys_rpa_executions` / `sys_rpa_schedules` / `sys_rpa_batch_item_reports` / `sys_rpa_credentials` / `sys_rpa_sessions` / `sys_rpa_audit_logs` / `sys_rpa_human_interventions` / `sys_rpa_selector_success` / `sys_rpa_selector_failure` 等 | `internal/models/rpa/` + `internal/core/db/migrations/` |

> **表数量口径**：本表只列「有运行期查询引用」的表；`sys_rpa_variables` / `sys_rpa_templates` /
> `sys_rpa_notifications` 当前仅有模型、无运行期引用（见 `internal/core/db/database.go` 注释）。

---

## 2. 已下线并删除的能力（**不再存在，勿再依赖**）

| 能力 | 原位置 | 下线原因与证据 |
|---|---|---|
| **flow 组 7 端点**（条件评估 / 数据映射 / 错误处理 / 循环） | `internal/api/v1/rpa/flow_handler.go`、`internal/services/rpa/{flow_control,error_handling,data_mapper}.go` | 2026-11-11（PR #87）下线 HTTP 路由；**Phase 158 / CLEANUP-03** 进一步 grep 取证确认 4 个构造器零非测试调用方后**整体删除实现**（7 文件 / 2885 行，含 4 个测试文件） |
| `SetupFlowRouter` 及注册 | `rpa_router.go` | 随 flow 组一并移除（PR #87） |

> **执行引擎的「条件 / 循环 / 数据映射」能力由 Phase 154 ENGINE 重写并落在
> `rpa-worker` 侧**（浏览器内动作全集 + 选择器阶梯 + 变量作用域），
> 与被删除的 server 侧 flow 服务**不是同一实现**，文档不再重复描述旧 flow 语义。

---

## 3. v1.56 里程碑（Phase 152~158）新增能力

| Phase | 能力 | 关键落点 |
|---|---|---|
| **152 CONTRACT** | 凭证 CRUD 不再 500（注入 `core.SM4Cipher`）；任务/执行/Worker 状态枚举与前端同构；任务名列名纠正（`task_name` → `name`）；选择器两表补 `TableName`；handler 契约用例走真实 sqlite 真表 | `internal/api/v1/rpa/rpa_router.go`、`internal/services/rpa/{credential_service,types,task_service}.go` |
| **153 RUNTIME** | 执行闭环：取消指令（Redis Pub/Sub `execution:cancel:{id}`）、人工干预（`worker:human-input:{w}:{e}`）、`paused` 状态机、干预事件落库 | `internal/services/rpa/{redis_publish,execution_state_machine}.go`、`internal/api/v1/rpa/execution_handler.go` |
| **154 ENGINE** | 执行引擎：1 Chrome + N page、动作全集、选择器阶梯、变量作用域 | `rpa-worker/internal/executor/` |
| **155 LIVEVIEW** | 无头可视化与人工接管：Worker 帧上行、Live 画布、人工接管入口、审计留痕 | `internal/services/rpa/liveview_hub.go`、`internal/api/v1/rpa/liveview_*.go`、`pages/operations/rpa/executions/` |
| **156 NOCODE** | 零门槛：浏览器插件录制/拾取器、AI 降级、三步向导页面 | `pages/operations/rpa/tasks/`（向导/编辑器）、migration 256 |
| **157 FLEET** | 承载与 VDI 编排：扩缩容 provider 化（container / vdivm）、一 VM 一容器、容量矩阵实测 | `internal/services/rpa/{scaling_service,docker_client}.go`、`internal/config/config.go`（`RPAScalingConfig`） |
| **158 CLEANUP** | 假数据/假配置清零、定时调度闭环、死代码删除与文档真相化 | 见下节 |

### Phase 158 / CLEANUP 落地明细（本报告新写部分）

| REQ | 内容 | 落点 |
|---|---|---|
| CLEANUP-01 | 批量执行报告真落库（`sys_rpa_batch_item_reports` 逐项表），读侧真统计；**无数据返回空统计**（此前恒返回 100/95/5） | `internal/models/rpa/batch_report.go`、`internal/services/rpa/excel_service.go` |
| CLEANUP-01 | 自动扩缩容配置真持久化到 `sys_config`（键 `sys.rpa.autoscale.config`），POST 后 GET 读回一致、重启仍生效（此前 GET 恒为常量、POST 丢弃请求体） | `internal/api/v1/rpa/worker_handler.go` |
| CLEANUP-01 | 删除 `WorkerHandler` 每实例 `redis.NewClient`（连接池泄漏），扩缩容指令改走 `core.Cache` 复用底层客户端 | `internal/api/v1/rpa/worker_handler.go`、`internal/services/rpa/redis_publish.go` |
| CLEANUP-02 | `sys_rpa_schedules` 从「零服务零 API」补齐为完整闭环：CRUD + 到期计算 + 到点真触发 + 停用摘 job | `internal/services/rpa/schedule_service.go`、`internal/api/v1/rpa/schedule_handler.go`、`internal/scheduler/rpa_schedule_tasks.go`、`internal/core/core.go` |
| CLEANUP-03 | flow 组 4 文件 + 3 测试文件删除（2885 行，grep 取证零非测试调用方） | 见第 2 节 |
| CLEANUP-03 | 权限码 `ops:rpa:schedule:{list,add,edit,remove}` 前后端 + sys_menu 三方一致 | `pkg/permission/config.go`、`xingran-react-frontend/src/constants/permissionCodes.ts`、migration 257 |

---

## 4. 定时调度（本 phase 新增）的真实语义

- **触发入口**：`sys_rpa_schedules` 每条「启用」调度对应调度器里一条 cron job
  （`sys_job.invoke_target = rpa_schedule:{scheduleID}`）。
- **执行路径**：到点 → `rpa_schedule` handler → `ScheduleService.TriggerNow` →
  `TaskService.Execute`（**与手动触发共用同一执行入口**，同一 Redis Stream 通道），
  执行记录 `trigger_type = schedule`、`triggered_by = scheduler`。
- **cron 表达式**：支持 5 位（分 时 日 月 周）与 6 位（秒 分 时 日 月 周）。
  调度器引擎为 6 位，5 位表达式在接线层归一为「前置 0 秒」。
- **时间回写**：`last_run_time` / `next_run_time` / `run_count` 每次触发后回写调度行。
- **停用即摘除**：停用或删除调度 → 对应 cron job 从调度器摘除（`sys_job` 行软删）。
- **权限**：`ops:rpa:schedule:list`（读）/ `add` / `edit` / `remove`（写，逐动作 strict 单码）。

---

## 5. 已知问题与**待验证项**（不得当作已完成）

> 以下条目要么是代码层面的已知缺口，要么是**本环境无真机/无生产授权而无法执行的验证**。
> 按 SHIPPED-DEFINITION，它们是「verification-pending」，**不是**「已验收」。

| # | 条目 | 类别 | 现状 |
|---|---|---|---|
| 1 | 批量报告 / 扩缩容配置 / 定时调度的**真实环境**验收（生产 PG + 真实 Redis + 真实 Worker） | 验证待办 | 本机已跑通 sqlite + 本机 Redis 5.0 的自动化用例；生产库写入与真实 Worker 联调未执行 |
| 2 | 前端 **TypeScript 严格化**遗留（API 响应类型断言） | 代码缺口 | `npx tsc --noEmit -p tsconfig.app.json` 当前无报错；旧报告所述「类型推断问题」已不成立 |
| 3 | 浏览器插件录制 / 可视化拖拽编辑器 | 未规划 | Phase 156 已交付「零门槛向导 + 拾取器」，拖拽式编辑器仍未立项 |
| 4 | AI 功能需自备 OpenAI 兼容 API Key | 配置项 | `configs/config.yaml` → `rpa.ai.generator` / `rpa.ai.agent` |
| 5 | 既有基线失败（非本 phase 引入，已用基线 worktree 复现） | 环境/既有 | `internal/services/vdi` `TestAuditService_RecordOperation`；`tests/e2e` `TestE2EAllEndpointsLogged`（缺 bash 路径） |

---

## 6. 部署前检查清单

### 环境要求
- [ ] PostgreSQL 18+（或 sqlite 供开发）
- [ ] Redis 7.4+（**必需**：任务下发走 Redis Stream；缺失时任务执行会失败并回滚执行记录）
- [ ] Go 1.24+ / Node.js 20+
- [ ] Docker（`rpa-worker` 镜像与容器承载，provider=container 时）

### 首次启动
1. [ ] 启动 PostgreSQL 与 Redis
2. [ ] 运行后端（迁移自动执行：含 migration 257 建 `sys_rpa_batch_item_reports` + 补建 `sys_rpa_schedules` + 播种调度权限码）
3. [ ] 确认 `sys_rpa_schedules` 与 `sys_rpa_batch_item_reports` 存在
4. [ ] 启动前端，访问 RPA 管理页
5. [ ] 建一条 1 分钟后的调度，观察到点是否生成 `trigger_type=schedule` 的执行记录

---

## 7. 结论

RPA 域**代码侧**已收口：假数据与假配置清零、定时调度形成闭环、flow 组死代码清除、
权限码前后端三方一致、路由注册表与权限码映射零漂移。

但按 `.planning/notes/SHIPPED-DEFINITION.md`，本域当前状态为
**`code-complete · verification-pending`**：真实环境（生产 PG + 现场真机 VDI +
真实 Worker 集群）的验收证据尚未落盘，第 5 节列出的验证待办清零前，
**不得**在任何对外文档中标注为 SHIPPED 或「100% 完成」。
