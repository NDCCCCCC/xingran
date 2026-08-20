---
type: quick
slug: backend-test-coverage-scan
created: 2026-08-20
status: in_progress
description: 后端 Go 代码覆盖率扫描 + 缺失测试模块清单,纯只读不修改源代码
---

# Plan: 后端测试覆盖率扫描

## 目标

摸底 XingRan-Next 后端 Go 项目的测试覆盖率现状,产出可决策的 per-package 报告与缺失测试模块清单,作为后续 v1.26「后端测试覆盖率治理」milestone 规划的输入。

## 范围

- **扫描范围**:`./...`(整个 module,含 `cmd/`、`internal/`、`pkg/`)
- **只读**:不修改任何源代码、配置、CI 文件
- **产出物**:
  1. `_test.go` 文件分布统计
  2. `go test ./... -cover` per-package 覆盖率数据
  3. per-package 覆盖率报告(总览 + 函数级)
  4. 缺失测试模块清单(按优先级分层)

## 方法

1. 收集所有 `*_test.go` 文件,按 package 分组
2. 跑 `go test ./... -coverprofile=coverage.out -covermode=atomic`(短超时,失败不致命)
3. 解析 `coverage.out`,用 `go tool cover -func` 生成函数级覆盖率
4. 按 package 维度聚合覆盖率,标记"无测试"包
5. 缺失测试模块按业务优先级分级:
   - **P0 核心**:system/operations/workorder/scheduler 的 service/handler
   - **P1 重要**:network/monitor/duty/knowledge/addomain 的 service/handler
   - **P2 边缘**:utils/parser/template/collector 等基础工具

## 不做(留给 v1.26 milestone)

- 不写新测试
- 不改 CI 配置
- 不重构源代码
- 不引入新的 mock/test framework

## 成功标准

- [ ] per-package 覆盖率报告产出(总览 + 函数级 top-N)
- [ ] 缺失测试模块清单产出(P0/P1/P2 分级)
- [ ] SUMMARY.md 落入 `.planning/quick/260820-backend-test-coverage-scan/`
- [ ] STATE.md "Quick Tasks Completed" 表更新
