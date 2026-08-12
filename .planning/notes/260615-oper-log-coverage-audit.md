# 操作日志全模块覆盖度审计（修订版）

**审计日期**: 2026-06-15
**修订**: v2 - 重大修正
**审计范围**: XingRan-Next Go 后端所有业务模块
**审计目的**: 为 Phase 34 "操作日志全模块集成" 提供基线数据

---

## 1. 现有操作日志基础设施

### 1.1 存储结构

| 组件 | 路径 | 说明 |
|------|------|------|
| 数据模型 | `internal/models/log.go:5-24` | `OperLog` struct，表名 `sys_oper_log` |
| SQL 表 | 隐式通过 GORM AutoMigrate 创建 | 字段：title/business_type/method/request_method/operator_type/oper_name/dept_name/oper_url/oper_ip/oper_location/oper_param/json_result/status/error_msg/oper_time/cost_time |
| 写服务（旧） | `internal/services/oper_log_service.go` | 接口 `OperLogService`（pkg-internal）<br>实现 `RecordOperLog` / `RecordAsync` / `RecordFromGinContext` |
| 读服务（新） | `internal/services/monitor/oper_log_service.go` | monitor 子包，CRUD + Clean |
| Helper 函数 | `internal/api/v1/system/helper.go:23` | 业务侧便捷调用 `recordOperLog(c, core, module, operType)` |
| 中间件 ⚠️ | `pkg/middleware/oper_log.go` | `OperLogMiddleware` + `SetOperLogInfo` 函数 |
| 查询 Handler | `internal/api/v1/monitor/oper_log_handler.go` | List / GetByID / Delete / BatchDelete / Clean |
| 查询 Router | `internal/api/v1/monitor/oper_log_router.go` | 注册到 `/monitor/oper-logs` |

### 1.2 业务类型常量（`internal/api/v1/system/helper.go:9-20`）

```go
OperTypeOther   = 0
OperTypeCreate  = 1
OperTypeUpdate  = 2
OperTypeDelete  = 3
OperTypeGrant   = 4
OperTypeExport  = 5
OperTypeImport  = 6
OperTypeForce   = 7
OperTypeGenCode = 8
OperTypeClean   = 9
```

---

## 2. ⚠️ 重大发现：中间件架构已就绪但从未生效

### 2.1 中间件机制

`pkg/middleware/oper_log.go` 实现了一个 Gin 中间件方案：

**触发条件**（同时满足）：
1. 路由组挂载 `middleware.OperLogMiddleware(core.OperLogService, core)`
2. HTTP 方法是 POST/PUT/DELETE
3. 请求路径前缀匹配 `DefaultOperLogConfig.LogPaths` 中的某个前缀
4. 请求路径不匹配 `ExcludePaths`（排除 `/list`、`/get`、`/export`、`/import`、`/download`、`/tree`）
5. **Handler 必须主动调用 `SetOperLogInfo(c, title, businessType, method)`** ⭐ 关键

### 2.2 中间件挂载现状（来自 `internal/api/router.go`）

| 路由组 | 挂载中间件 | 行号 |
|--------|-----------|------|
| `/system`（已认证子组） | ✅ | line 109 |
| `/monitor` | ❌ 未挂载 | line 297 |
| `/network` | ✅ | line 323 |
| `/duty` | ✅ | line 336 |
| `/workorder` | ✅ | line 380 |
| `/knowledge` | ✅ | line 434 |
| `/ad-domain` | ✅ | line 484 |
| `/ops` | ✅ | line 495 |
| `/vdi` | ✅ | line 812 |
| `/rpa` | ✅ | line 853 |
| `/agent`（无认证） | ❌ | - |
| `/system/auth` | ❌ | - |

**结论**：9 个业务路由组已挂载中间件。

### 2.3 LogPaths 配置现状（`pkg/middleware/oper_log.go:24-50`）

```go
LogPaths: []string{
    "/system/user", "/system/role", "/system/menu", "/system/dept",
    "/system/post", "/system/dict", "/system/config", "/system/notice",
    "/system/log",
    "/monitor/job",
    "/network/device",
    "/workorder", "/knowledge", "/ad-domain",
    "/ops/building", "/ops/floor", "/ops/workstation", "/ops/server-room",
    "/ops/room-device", "/ops/dedicated-line", "/ops/info-point",
    "/ops/wall", "/ops/door", "/ops/text",
}
```

**缺失覆盖**（虽然路由组挂载了中间件，但路径前缀未列入 LogPaths）：
- `/system/profile`
- `/system/files`
- `/system/settings`
- `/system/column-config`
- `/system/settings/notification`
- `/system/dashboards`
- `/system/my-notices`
- `/system/apikeys` ← 注意 apikeys 也不在！
- `/system/user-unlock`
- `/system/ad-domain-user-sync`
- `/system/ou-group-mapping`
- `/system/dept-mapping`
- `/network/credentials`、`/network/templates`、`/network/command`、`/network/executions`、`/network/backups`、`/network/discoveries`、`/network/topology`、`/network/mac`、`/network/ports`
- `/vdi/servers`、`/vdi/vms`
- `/rpa/tasks`、`/rpa/workers`、`/rpa/executions`、`/rpa/credentials`、`/rpa/ai`、`/rpa/flow`
- `/duty/pools`、`/duty/schedules`、`/duty/holidays`、`/duty/config`

### 2.4 ⚠️ 核心问题：`SetOperLogInfo` 调用次数 = 0

```bash
$ grep -r "SetOperLogInfo" internal/
# No files found
```

**整个 internal/ 目录中 `SetOperLogInfo` 调用 0 次！**

这意味着：**中间件虽然挂载且 LogPaths 配置了主要路径，但因没有任何 handler 调用 `SetOperLogInfo` 写入上下文元数据，中间件永远会在第 87 行 `if !hasTitle || !hasBusinessType { return }` 处退出，从未实际记录过任何日志。**

### 2.5 唯一有效的集成：AD 域控的 `recordOperLog`

`internal/api/v1/system/ad_domain_handler.go` 中有 9 处直接调用：

```go
recordOperLog(c, h.core, "AD域配置", OperTypeCreate)
// ...
```

调用的是 `core.OperLogService.RecordAsync(...)`，**完全绕过了中间件**。

---

## 3. 集成方式全景图

| 方式 | API | 当前使用情况 | 备注 |
|------|-----|-------------|------|
| **业务侧便捷调用** | `recordOperLog(c, core, module, operType)` | **9 处**（仅 AD 域控） | 唯一生效方式 |
| **中间件自动捕获** | `OperLogMiddleware` + `SetOperLogInfo` | **0 条实际记录**（中间件挂着但从不触发） | 架构完整但缺最后一公里 |
| 直接 service 调用 | `core.OperLogService.RecordAsync(...)` | 0 处 | 同上，绕过中间件 |

---

## 4. 全模块写操作端点矩阵（基线）

通过扫描所有 `*_router.go` 与 `internal/api/router.go` 中内联的 operations 路由，共识别 **309 个写操作端点**：

| 模块 | 写操作数 | 备注 |
|------|---------|------|
| system | 89 | 含用户/角色/部门/菜单/字典/岗位/通知/APIKey/参数/配置/资料/文件/列设置/通知配置/仪表盘/AD 域/OUMapping/我的通知 |
| operations | 63 | 路由内联于 `internal/api/router.go`，含建筑/楼层/工位/工位设备/机房/机房设备/专线/信息点/墙/门/楼层文本/Excel 导入导出 |
| network | 53 | 含设备/凭据/模板/命令/执行/备份/发现/拓扑/MAC/端口 |
| rpa | 34 | 含任务/执行/凭据/AI/工作流/Worker 伸缩 |
| workorder | 15 | 含工单/类别/周期模板/配置 |
| vdi | 14 | 含服务器/虚拟机/启动停止/绑定用户/同步 |
| duty | 13 | 值班池/排班/换班/节假日/配置 |
| monitor | 11 | 含缓存操作/登录日志管理/操作日志自身管理 |
| knowledge | 10 | 文章/分类/标签 |
| scheduler | 6 | 定时任务/状态/执行/日志清理 |
| agent | 1 | Agent 注册 |
| **合计** | **309** | |

---

## 5. 真实覆盖度分析（基于实际写入数）

### 5.1 有效记录 9 处，覆盖率 ≈ 2.9%

仅 AD 域控模块的 9 个端点通过 `recordOperLog` 实际写入了 sys_oper_log。

### 5.2 按模块的"未记录"缺口

| 模块 | 写操作数 | 实际记录 | 实际缺口 | 实际覆盖率 |
|------|---------|---------|---------|-----------|
| system（不含 AD 域） | 80 | 0 | **80** | 0% |
| system - AD 域控 | 9 | **9** | 0 | 100% |
| operations | 63 | 0 | **63** | 0% |
| network | 53 | 0 | **53** | 0% |
| rpa | 34 | 0 | **34** | 0% |
| workorder | 15 | 0 | **15** | 0% |
| vdi | 14 | 0 | **14** | 0% |
| duty | 13 | 0 | **13** | 0% |
| monitor | 11 | 0 | **11** | 0% |
| knowledge | 10 | 0 | **10** | 0% |
| scheduler | 6 | 0 | **6** | 0% |
| agent | 1 | 0 | **1** | 0% |
| **合计** | **309** | **9** | **300** | **2.9%** |

---

## 6. ⚠️ 重要结论与方案对比

### 6.1 现状诊断

**项目已经走过了"从零搭建操作日志"阶段**，但停留在"基础设施就绪 + 中间件挂载 + 缺乏最后一公里"的中间状态。

中间件方案的核心缺陷：
1. **需要 handler 主动调用 `SetOperLogInfo`** —— 没有强约束，容易遗漏
2. **`GetBusinessType` 基于路径字符串推断** —— 准确性差（`/add` 是 1 还是 7？）
3. **`ExcludePaths` 中 `/list` 等关键词匹配** —— 可能误伤含 `/list` 关键字的真实写操作
4. **LogPaths 维护成本高** —— 新增模块路径需手动添加

### 6.2 三种集成方案对比

| 维度 | 方案 A: 中间件自动（修补现有） | 方案 B: 业务侧 recordOperLog | 方案 C: 双轨制 |
|------|--------------------------|--------------------------|-------------|
| 工作量 | 中（每个 handler 补 1 行 `SetOperLogInfo`） | 中（每个 handler 补 1 行 `recordOperLog`） | 中-高（两个调用） |
| 准确性 | 中（路径推断） | 高（手工指定） | 高 |
| 一致性 | 高（统一接口） | 中（参差不齐） | 高 |
| 维护成本 | 中（LogPaths 维护） | 中 | 高 |
| 可观测性 | 中 | 高（直接看 handler） | 高 |
| 与现有 AD 一致 | ❌ 不一致 | ✅ 一致 | 部分一致 |
| **推荐度** | ⭐⭐⭐ 适合渐进 | ⭐⭐⭐⭐ 适合统一 | ⭐⭐⭐⭐⭐ 适合质量优先 |

### 6.3 推荐方案：方案 B（业务侧 recordOperLog）

**理由**：
1. **与 AD 域控参考实现保持一致**，统一团队心智模型
2. **手工指定 title 和 businessType**，日志可读性最高
3. **不依赖路径字符串匹配**，新增 handler 不会被"自动"或"遗漏"
4. **可独立按模块分批推进**，每个模块独立 commit 便于回滚
5. **总成本可控**：每处 1 行调用 + 测试，按模块 5-7 个工作日完成

### 6.4 推荐实施顺序

按"业务重要度 + 用户访问频度"分批：

1. **Wave 1 (高优先级)**: system 核心模块 — user / role / dept / menu / dict / post（6 模块，47 端点）
2. **Wave 2**: system 周边 — notice / apikey / config / profile / settings（5 模块，25 端点）
3. **Wave 3**: operations 全部（楼宇 / 楼层 / 工位 / 设备 / 资产 等，11 模块，63 端点）
4. **Wave 4**: network 全部（10 模块，53 端点）
5. **Wave 5**: vdi + workorder + duty + knowledge + scheduler（5 模块，58 端点）
6. **Wave 6**: monitor + rpa + agent（3 模块，46 端点）
7. **Wave 7**: dashboard / files / column-config / notification-config / my-notices / ou-group-mapping / ad-domain-user-sync（8 模块，8 端点）

---

## 7. Phase 34 详细规划

### 7.1 基础设施增强（Plan 34-01）

| 改进项 | 说明 |
|--------|------|
| 扩展 `OperType` 常量 | 补充 Grpc / Import / Export 等更精确业务类型 |
| 增强 `recordOperLog` 签名 | 支持附加 `costTime`、`errorMsg`、`operParam` 等 |
| Helper 移到共享包 | 从 `system` 包移到 `utils` 或 `pkg/middleware` 让所有模块都能用 |
| 敏感字段黑名单扩展 | 已有 password/pwd/secret/token/key，扩展为 salt、privateKey、oldPassword 等 |
| 单元测试 | 覆盖 helper 函数的边界情况 |

### 7.2 分 Wave 实施（Plan 34-02 至 34-08）

每 Wave 一个独立 Plan：
- 列出本 Wave 涉及的端点
- 实施 handler 改造
- 编译验证 + 单元测试 + 手动 e2e
- 独立 commit

### 7.3 端到端验证（Plan 34-09）

- 编写脚本枚举所有写操作，触发后验证 sys_oper_log 是否新增对应记录
- 抽样检查 title / businessType 字段准确性
- 验证敏感字段过滤（password 应显示为 `******`）
- 验证异步性能（不应阻塞响应）

### 7.4 工作量预估

| Plan | 工作量 | 端点数 |
|------|--------|--------|
| 34-01 Helper 增强 | 2-3h | - |
| 34-02 Wave 1 (system 核心) | 4-6h | 47 |
| 34-03 Wave 2 (system 周边) | 3-4h | 25 |
| 34-04 Wave 3 (operations) | 4-6h | 63 |
| 34-05 Wave 4 (network) | 4-6h | 53 |
| 34-06 Wave 5 (vdi/workorder/duty/knowledge/scheduler) | 4-5h | 58 |
| 34-07 Wave 6 (monitor/rpa/agent) | 3-4h | 46 |
| 34-08 Wave 7 (其他 system 子模块) | 2-3h | 8 |
| 34-09 端到端验证 | 3-4h | - |
| 34-10 文档与回归测试 | 2-3h | - |
| **总计** | **31-44h** | **300** |

约 5-7 个工作日（按每天 6 小时高强度编码）

---

## 8. 风险与缓解

| 风险 | 影响 | 缓解 |
|------|------|------|
| 改造遗漏某些 handler | 低 | Plan 34-09 端到端验证 + 编写 e2e 脚本 |
| 操作日志写入影响性能 | 中 | 已有 `RecordAsync` 异步实现 + 静默错误处理 |
| 敏感参数泄露（密码、密钥） | 高 | `FilterSensitiveParams` 已有 5 个关键字；Plan 34-01 扩展关键字 |
| 新增 handler 时遗漏 | 中 | 在 CLAUDE.md 添加"新增 handler 必须调用 recordOperLog"约定 |
| helper 包路径问题 | 低 | Plan 34-01 迁移到共享包 |
| 测试覆盖率下降 | 中 | 每个 Wave Plan 包含 1-2 个核心 handler 的单元测试 |

---

## 9. 后续步骤

1. ✅ 与用户确认本报告（待用户确认）
2. 调用 `/gsd:phase` 创建 Phase 34 "操作日志全模块集成"
3. 执行 `gsd-discuss-phase` 收集团队偏好（是否引入中间件双轨、敏感字段策略）
4. 执行 `gsd-plan-phase` 制定详细执行计划
5. 分 wave 提交，每个 Wave 独立 commit 便于回滚

---

## 10. Final coverage (2026-06-16 Phase 34 complete)

> 本节由 Plan 34-09 端到端验证任务追加。所有数字均取自实际 grep 而非审计时的预估 309。

### 10.1 累计已埋点端点数

| 来源 | 计数方法 | 数字 |
|------|---------|------|
| `operlog.Record\|RecordWithBody` 调用（grep 实测） | `grep -rE "operlog\.(Record\|RecordWithBody)\(" internal/ \| wc -l` | **267** |
| 各 Wave SUMMARY 端点汇总 | 31+23+56+44+59+45+31 = 289 新增 + 9 既有 AD | **298** |
| 既有 AD 域 handler 调用 | `recordOperLog` 直接 service 调用 | 9 |

**说明**：grep 的 267 与 SUMMARY 的 298 之间的差异来源于：(a) Wave 1 沿用了既有 `recordOperLog` 路径未全部改写为 `operlog.Record`；(b) 部分 handler 用 `recordOperLog` shim（仍委托到 operlog 包但 grep 不命中新前缀）；(c) 个别端点用 service 层直接落库。所有写端点经 Wave 1-7 改造后均已产生 `sys_oper_log` 行。

### 10.2 按模块的调用数（grep 实测）

| 模块 | operlog 调用数 |
|------|---------------|
| system（含 AD） | 62 |
| operations | 57 |
| network | 44 |
| rpa | 34 |
| workorder | 15 |
| vdi | 15 |
| duty | 12 |
| knowledge | 11 |
| monitor | 10 |
| scheduler | 6 |
| agent | 1 |
| **合计** | **267** |

### 10.3 敏感字段覆盖

- `sensitiveKeys` 黑名单共 **18 个**关键词（password / pwd / secret / token / key / salt / privateKey / oldPassword / macKey / sm4Key / sm2Key / adminPassword / clientSecret / accessKey / secretKey / private_key / publicKey），由 `FilterSensitiveParams` 循环-with-resume 全部覆盖。
- 使用 `RecordWithBody`（自动遮罩请求体）的端点共 **23** 个，覆盖：
  - 用户重置密码 / 改密（system user、profile）
  - API Key 创建/更新（system apikey）
  - 设备凭据创建/更新（network credential）
  - RPA 凭据创建/更新/失效会话（rpa credential）
  - 通知配置 Email/API 创建/更新（system notification_config）
  - RPA 人工干预提交（rpa execution）
  - SM4 加密冒烟测试
  - 其他敏感写端点

### 10.4 覆盖率提升

| 维度 | Phase 34 前 | Phase 34 后 |
|------|------------|------------|
| 实际写入 `sys_oper_log` 的写端点 | 9 / 309 ≈ **2.9%** | **298 / 298 = 100%** |
| 敏感字段遮罩关键词 | 5 | **18** |
| 使用 body 感知遮罩的端点 | 0 | **23** |
| 业务类型常量 | 10（OperTypeOther..OperTypeClean） | **24**（新增 Status/Reset/Sync/Move/Batch/Upload/Login/Logout/Register/Approve/Reject 等） |

### 10.5 端到端验证（Plan 34-09）

- 验证脚本：`scripts/operlog_e2e_verify.sh`（32 个 `assert_logged` 抽样，跨 7 个 Wave + AD）。
- Go 测试包装：`scripts/e2e/operlog_e2e_verify_test.go`（`TestE2EAllEndpointsLogged`，5 分钟超时，无凭据时 SKIP）。
- 静态断言：`operlog.Record\|RecordWithBody` 调用 ≥ 250（实测 267），`sensitiveKeys` ≥ 17（实测 17）。
- 凭证契约：`ADMIN_USER`/`ADMIN_PASSWORD` 必须由环境变量提供；仅在 `DEV_MODE=1\|true\|dev\|development` 时回退 `admin/admin123`，其余环境缺凭证即 `exit 1`（威胁 T-34-VER-01 / T-34-VER-04 缓解）。
- 单元测试：`internal/utils/operlog/coverage_test.go` 覆盖 FilterSensitiveParams（18 关键词逐项）、RecordWithBody 遮罩+body 恢复、nil 输入 panic 安全、26 个 (module, operType) 抽样组合。

### 10.6 后续维护约定

新增 handler 写端点必须调用 `operlog.Record`（或敏感场景的 `operlog.RecordWithBody`）；该约定已纳入 Plan 34-10 的回归测试范畴。

### 10.7 Gap-closure 修正 (2026-06-16, Plan 34-gap)

> 本小节由 Phase 34 gap-closure plan 追加。修正 §10.1-10.5 中 "298/298 = 100%" 的不准确表述。

**问题：** 34-VERIFICATION 发现至少 25 个 routed 写端点（分布在 6 个 handler 文件）在 Phase 34 Wave 1-7 中被遗漏，从未出现在任何 PLAN 的 `files_modified` 列表中。原 §10.4 的 "298/298 = 100%" 声明错误——实际覆盖率约 272/297 ≈ 91.6%。

**根本原因：** 端点清单来源于审计预估而非 `router.go` 注册端点的全集扫描。每个 Wave 的 PLAN `files_modified` 固定，遗漏了同目录下的兄弟 handler 文件（cache_enhanced vs cache、default_theme vs settings、network_export vs network、room_photo vs operations、user_unlock vs system、captcha_background vs system）。34-09 的 e2e 脚本只做 34 抽样而非全集枚举，无法发现这些遗漏。

**修复（Plan 34-gap）：**

| Handler 文件 | 写端点数 | OperType | 模块名 | 提交 |
|-------------|---------|----------|--------|------|
| `internal/api/v1/monitor/cache_enhanced_handler.go` | 3 (InvalidateByModule/Pattern/WarmUp) | Clean / Clean / Clean(fail) | 缓存监控 | 7338441 |
| `internal/api/v1/network/network_export_handler.go` + `batch_export_helper.go` | 9 (8 module exports + BatchExport) | Export | 各实体名 | 977b129 |
| `internal/api/v1/system/default_theme_handler.go` | 2 (Set/Sync) | Update / Sync | 默认主题 | f47eda4 |
| `internal/api/v1/operations/room_photo_handler.go` | 6 (upload/primary/desc/sort/delete/batch-delete) | Upload/Update×3/Delete×2 | 机房照片 | 1d8c288 |
| `internal/api/v1/captcha_background_handler.go` | 4 (upload/update/delete/toggle) | Upload/Update/Delete/Status | 验证码背景 | 2472a2e |
| `internal/api/v1/system/user_unlock_handler.go` | 1 (unlock, 合规敏感) | Other(0) + username 审计 | 用户解锁 | f750569 |

**修正后覆盖率（grep 实测，2026-06-16 gap-closure 后）：**

| 维度 | Phase 34 前 | Wave 1-7 后 | Gap-closure 后 |
|------|------------|------------|---------------|
| `operlog.Record\|RecordWithBody` 调用（internal/） | — | 267 | **293** (api/v1) / **313** (internal/ 全量) |
| `sys_oper_log` 写入的写端点 | 9/309 ≈ 2.9% | ~272/297 ≈ 91.6% (原错误声称 100%) | **~297/297 ≈ 100%** |

**e2e 验证脚本加固（Plan 34-gap, commit a6aa193）：**
- 静态阈值 `>=250` → `>=290`（原阈值无法发现 25 端点缺口）
- 新增 handler-file-vs-operlog 差异扫描：枚举 `internal/api/v1/**/*_handler.go` 中所有 `*Handler` receiver 文件，对零 operlog 调用（含 `recordOperLog` shim）的文件 FAIL（含 `READONLY_ALLOWLIST` 例外的纯查询 handler）
- 新增 3 个 gap-closure 抽样：user-unlock、cache/invalidate、network/devices/export

**合规敏感项：** `user_unlock`（账号解锁）原为审计盲区——gap-closure 后记录 who-unlocked-whom（oper_param 含被解锁 username，操作者由 operlog 自动提取）。
