---
type: quick
slug: backend-test-coverage-scan
created: 2026-08-20
completed: 2026-08-20
status: complete
description: 后端 Go 代码覆盖率扫描 + 缺失测试模块清单,纯只读不修改源代码
duration: ~14m (go test 跑 10m + 解析 4m)
---

# Summary: 后端测试覆盖率扫描

## TL;DR

| 指标 | 数值 |
|------|------|
| **后端业务 package 总数** | **74** |
| **加权平均覆盖率** | **12.8%**(43652 stmts / 5589 covered) |
| **0% 覆盖(无任何测试)** | **33 个 package** |
| **<10% 低覆盖** | 15 个 |
| **10-50% 中等覆盖** | 17 个 |
| **≥50% 高覆盖** | 9 个 |
| **CI 是否有 coverage gate** | ❌ 无(只跑 `go test`,不带 `-cover` 不带阈值) |
| **已知测试失败** | 1 个(`pkg/cache.TestAsyncRetryWorker_Enqueue`,expected 1 actual 0) |

**结论**:后端测试严重不足——12.8% 加权覆盖率、33/74 package 完全无测试、CI 没有任何 coverage 阈值守护。这正是 v1.26 milestone 应填补的核心空白。

## 扫描方法

```bash
# 全 module 测试 + 生成 coverage.out
go test ./... -coverprofile=coverage.out -covermode=atomic -count=1

# 函数级明细
go tool cover -func=coverage.out > cover-func.txt    # 5152 行

# Per-package 聚合(awk 脚本,见 per-package-coverage.txt 顶部)
awk 'NR > 1 {
    split($1, parts, ":")
    n = split(parts[1], seg, "/")
    pkg = ""; for(i=4; i<=n-1; i++) pkg = (pkg == "") ? seg[i] : pkg "/" seg[i]
    # 排除 scripts/, tests/, node_modules/, *.archive-migrate-* 等工具脚本
    num_stmts = $2; hit_count = $3
    covered = (hit_count > 0) ? num_stmts : 0
    biz_stmts[pkg] += num_stmts; biz_covered[pkg] += covered
} END { ... }' coverage.out
```

**排除范围**:`scripts/`(运维工具脚本)、`tests/scripts/`、`node_modules/`、MAC 清理工具集(`mac/`、`dbactivity`/`dbbootstrap`/`dbprobe`/`dbprovision`、`db/audit_view_refs`)、密钥生成/迁移工具(`crypto/gen_sm2_keys`、`crypto/migrate_sm4_key`)、诊断脚本(`diag/red_4f001`)、env 检查、migration 工具。

**耗时**:go test ~10 分钟(全 module atomic coverage + 部分集成测试用 in-memory SQLite),其他 ~4 分钟。

## 高覆盖率 Top 10(亮点)

| Package | Stmts | Covered | % |
|---------|------:|--------:|---:|
| pkg/normalize | 45 | 44 | **97.8%** |
| internal/config | 147 | 137 | **93.2%** |
| internal/middleware | 196 | 169 | **86.2%** |
| internal/transform | 111 | 95 | **85.6%** |
| internal/services/portwrite | 259 | 221 | **85.3%** |
| internal/services/component_collector | 345 | 285 | **82.6%** |
| internal/utils/operlog | 90 | 74 | **82.2%** |
| internal/services/topology | 73 | 55 | **75.3%** |
| internal/services/lldp | 96 | 57 | **59.4%** |
| internal/core/security | 313 | 153 | **48.9%** |

✅ **亮点模块**(做得好,可作为模板参考):
- **operlog 82.2%** + **regression_test.go** 锁公共 API——是覆盖率治理的范本
- **portwrite 85.3%** 含 e2e + service + 中间件三层覆盖
- **middleware 86.2%** 含 apikey 全链路测试
- **component_collector 82.6%** 含 CLI 厂商解析 + SNMP + owner resolver 全维度

## 缺失测试模块清单(按业务优先级)

### 🔴 P0 核心 — 业务关键、零测试或接近零测试

| Package | Stmts | % | 说明 |
|---------|------:|---:|------|
| internal/api/v1/workorder | 297 | **0.0%** | 工单 HTTP handler,**完全无测试** |
| internal/api/v1/monitor | 518 | **0.0%** | 监控 HTTP handler,**完全无测试** |
| internal/api/v1/scheduler | 152 | **0.0%** | 调度 HTTP handler,**完全无测试** |
| internal/services/workorder | 715 | 0.6% | 工单服务(有 `_statistics_test.go` 但覆盖近乎零) |
| internal/api/v1/system | 3039 | 0.5% | 系统管理 HTTP handler(3039 语句,业务核心) |
| internal/services/system | 3483 | 10.2% | 系统服务(3483 语句,user/role/menu/dept/dict/post/config 大本营) |

### 🟡 P1 重要 — 完全无测试

| Package | Stmts | % | 说明 |
|---------|------:|---:|------|
| internal/api/v1/duty | 265 | **0.0%** | 值班 HTTP handler |
| internal/api/v1/knowledge | 273 | **0.0%** | 知识库 HTTP handler |
| internal/api/v1/rpa | 612 | **0.0%** | RPA HTTP handler |
| internal/api/v1/vdi | 298 | **0.0%** | VDI HTTP handler |
| internal/services/duty | 114 | **0.0%** | 值班服务 |
| internal/services/knowledge | 85 | **0.0%** | 知识库服务 |
| internal/services/monitor | 485 | **0.0%** | 监控服务 |
| internal/services/network | 127 | **0.0%** | 网络服务 |

### 🟢 P2 重要 — 有测试但极低(<10%)

| Package | Stmts | % | 说明 |
|---------|------:|---:|------|
| internal/services/rpa | 1865 | 1.1% | RPA 服务(1865 大块,仅 statistics 测了) |
| internal/services/vdi | 1127 | 2.7% | VDI 服务 |
| internal/core | 754 | 2.1% | Core 初始化 |
| internal/services/scheduler | 167 | 4.8% | 调度服务 |
| internal/agent/server | 616 | 2.1% | Agent server |
| internal/device | 1249 | 2.5% | 设备连接 |
| internal/utils | 531 | 4.5% | 工具集 |
| internal/api/v1/operations | 1285 | 3.0% | operations HTTP |
| internal/api/v1/asset | 420 | 8.3% | asset HTTP |
| internal/api/v1/network | 1971 | 7.6% | network HTTP |
| internal/services/system | 3483 | 10.2% | system 服务 |

### ⚙️ 辅助工具 — 完全无测试(可后置)

| Package | Stmts | 说明 |
|---------|------:|------|
| internal/models/{operations,rpa,system,system/requests} | 237 | 数据模型(纯 struct 通常不必测,但有逻辑的方法需要) |
| internal/pkg/{cache,system} | 512 | 内部 pkg |
| internal/api | 417 | api 根 |
| internal/server, internal/docs | 3 | 入口/文档 |
| pkg/{captcha,gormutil,ldaputils,logger,query,response,time} | 1138 | 通用工具(低风险) |
| cmd, cmd/agent | 165 | 入口 main |
| internal/agent/pkg/retry | 33 | retry 工具 |
| internal/api/v1/agent | 38 | Agent HTTP(已并入 agent/server 体系) |
| internal/api/v1/operations/requests | 15 | operations 请求模型 |
| internal/services/common | 1 | 几乎空文件 |

## 已知问题

### 1 个测试失败(独立、不影响覆盖率数据)

```
--- FAIL: TestAsyncRetryWorker_Enqueue (0.20s)
    retry_test.go:209:
        Error: Not equal:
               expected: 1
               actual  : 0
FAIL    github.com/.../pkg/cache    7.907s
coverage: 24.6% of statements
```

⚠️ **修复建议**:测试期望值与并发实际行为不匹配(`expected 1 actual 0`)。可能是 timing issue 或 AsyncRetryWorker 在测试环境下未触发重试逻辑。这是个**潜在 P1 bug**——生产环境如果 AsyncRetryWorker 行为不符预期,缓存层重试就失效。建议作为 v1.26 启动块前修复。

### 1 个相关 quick 备忘

工作区当前已 staged 状态有遗留 `M internal/services/system/asset_columns_schema.json`(在 STATE.md "Operator Next Steps" 中标注为 Phase 68 范围外),可能影响 Phase 63-70 之后的工作,但不影响本次扫描结论。

## CI 现状(没有 coverage gate)

`.github/workflows/ci.yml` 的 backend job 仅跑:
```yaml
- name: Test
  run: go test -timeout 15m -count=1 ./internal/... ./pkg/... ./cmd/...
```

**没有任何** coverage 收集、阈值检查、PR 注释。Phase 63 给前端做了 `vitest --coverage` + thresholds (25/15/22/25),后端对应物完全缺失。

## 对 v1.26 milestone 的建议

### 候选目标(用户决定)

| 候选 | 起点 | 优秀目标(参考前端 Phase 63) |
|------|------|------|
| 加权平均覆盖率 | **12.8%** | **70%+**(对齐前端 Phase 63 阈值 25/15/22/25) |
| 0% 覆盖业务包 | 33 | <10(主要清零 P0 + P1 业务模块) |
| CI coverage gate | 无 | 阈值 + diff coverage(PR 增量) |
| 测试失败 | 1 | 0 |

### 推荐 phase 拆分(参考 v1.22 经验)

```
Phase 71: coverage baseline 治理 + CI gate
  - 修复 pkg/cache TestAsyncRetryWorker_Enqueue
  - 加 `go test -coverprofile` + `go-test-coverage` 阈值 gate 到 CI
  - 落地 baseline 数字与基线

Phase 72: P0 核心补齐(api/v1/{workorder,monitor,scheduler} + services/workorder)
  - 优先 workorder + monitor(高频审计路径)
  - services/system 增量到 ≥50%

Phase 73: P1 重要补齐(api/v1 + services {duty,knowledge,monitor,network,rpa,vdi})
  - 8 个完全无测试业务模块
  - services/system 增量到 ≥70%

Phase 74: P2 增量 + 优秀等级达成
  - api/v1/{operations,asset,network} 增量
  - 加权平均 ≥70%
  - 引入 diff coverage(PR 增量门槛 ≥80%)
```

### 关键成功指标(SC)

参考 v1.22/Phase 63 模式:
- **SC#1** 加权平均 ≥ 70%
- **SC#2** 0% 覆盖业务包 ≤ 5
- **SC#3** CI 引入 coverage threshold gate(失败即阻断)
- **SC#4** 已知测试失败清零
- **SC#5** diff coverage(PR 增量)阈值启用

## 产出文件

- `coverage.out`(2.95 MB,29302 行,Go atomic mode profile)
- `cover-func.txt`(5152 行,函数级明细,可读)
- `per-package-coverage.txt`(76 行,per-package 聚合,本 SUMMARY 的源数据)
- `PLAN.md`(扫描方案)
- `SUMMARY.md`(本文件)

## 不做(留给 v1.26)

- 不写新测试用例
- 不修改源代码 / 测试代码
- 不引入 mock framework(已存在 glebarez sqlite in-memory 即可)
- 不调整 CI 配置
- 不修复 `pkg/cache.TestAsyncRetryWorker_Enqueue`(留给 Phase 71)
- 不清理 `M internal/services/system/asset_columns_schema.json`(由用户决定)

## 给用户的下一步建议

```
yolo 模式 + v1.26 启动块建议路径:
  /clear
  /gsd-new-milestone      ← 收集需求(目标覆盖率、CI gate、phase 拆分)
    ↓
  /gsd-plan-phase 71      ← 治理 baseline + 修测试失败 + CI gate
  /gsd-execute-phase 71   ← 自动执行
    ↓
  /gsd-plan-phase 72      ← P0 核心补齐
  ...
```

或者:
- 单 quick 任务直接修 `pkg/cache.TestAsyncRetryWorker_Enqueue`(小、明确、有价值)
- 用 `go tool cover -html=coverage.out -o coverage.html` 生成可读 HTML 报告