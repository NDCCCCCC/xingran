---
one-liner: scheduler Rotation 抽共享纯函数对齐 service 双路径、3 厂商×3 参数端口物理参数写全闭环、3 处排序收尾落产品决策结论
---

# Phase 138 — 功能分支补齐（v1.53 / FIX-04 + FIX-05 + FIX-14）

**执行**：2026-11-08，与 Phase 134 串行（同触 `internal/scheduler/`），独立智能体 phase-138。

## 交付清单

| REQ | 文件 | 内容 |
|---|---|---|
| FIX-05 | `internal/services/workorder/rotation.go`（新建） | 共享纯函数 `LoadRotationMembers` + `RotationSelectedIndex`，避免 scheduler/service 复制取模逻辑 |
| FIX-05 | `internal/scheduler/workorder_tasks.go:231-266`、`periodic.go:401-437` | Rotation 分支真实轮询；service 原子/降级两路径统一改用共享函数；「分配前游标 = TotalGenerated」不变量 |
| FIX-05 | `workorder_tasks_138_test.go`（新建）、`workorder_tasks_80_02_test.go` | 5 个游标 × 双路径一致性测试；旧 stub 断言同步到新契约 + 补 DDL |
| FIX-04 | `internal/services/portcollection/physical_template.go` + 测试（新建） | 3 厂商 × 3 参数（speed / duplex / MTU）共 9 个渲染闭包 |
| FIX-04 | `internal/services/portwrite/physical.go` + `physical_test.go` + `physical_e2e_test.go` + fixture（新建） | sentinel validator、`PhysicalWrite`、FileTransport e2e、非法参数零 SSH 流量守护 |
| FIX-04 | `port_write_service.go`、`vlan_trunk.go`、`dry_run.go`、`vendor_port_template.go` | `PortTemplateParams` 加 4 字段、`renderExtras.physical`、dry-run 同源校验与透传 |
| FIX-04 | `port_write_physical_handler.go` + 测试（新建）、`port_write_handler.go:288,559`、`port_write_router.go:75` | `POST /network/ports/write/physical`、operlog、审计 `after_value` 3 分支、组级权限复用 |
| FIX-14 | `.planning/notes/261108-sort-backlog-decision.md`（新建） | 3 处现状 / ≥2 方案 / 推荐+理由 / 契约影响 / 产品确认清单；三处保持既有默认（零回归） |
| FIX-14 | `docs/管理员使用手册.md:814,821,833` | 物理参数改 ✅ 已落地 + 真机取证注记；§22.3 回指结论文档 |

## 验证（智能体自跑）
- `go build ./...` exit 0
- `go test -count=1 ./internal/scheduler/` ok 23.4s（含 134 的 ad_sync 测试，零回归）
- `./internal/services/workorder/`、`./internal/services/portcollection/`、`./internal/services/portwrite/`（全套含 e2e）、`./internal/api/v1/network/` 五包全 ok
- 集成期 Lead 复核：`go build ./...` 与全仓 `go test ./...` 全绿

## 边界（如实声明）
- 3 厂商物理参数命令依据官方 command reference，**未经真机逐型号取证**（Phase 129 UAT 未覆盖），模板头注与手册已标注；型号关键字差异需真机 UAT 后补 modelRules。
- `vendorCapabilities` 未对新 3 动作设 Unsupported（沿用 baseline 默认支持）。
- FIX-14 三处产品决策问题待产品拍板后另立 REQ，本 Phase 未擅自定规则。
- 批量路径未接入物理参数动作（REQ 未要求，沿用 v1.20.1 边界）。
