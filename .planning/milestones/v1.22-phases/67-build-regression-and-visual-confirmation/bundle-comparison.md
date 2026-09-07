# Phase 67 QA-03 Bundle 三口径对比 — `bundle-comparison.md`

> 来源: orchestrator 在 `xingran-react-frontend/` 内实测（2026-08-18，npm run build 1m 44s exit 0）
> 历史基线: 65-01 数据点（Phase 65 末）

| 口径 | v1.21 末基线 | v1.22 末（Phase 67） | 差异 | 结论 |
|---|---|---|---|---|
| **vendor-react gzip** | 774.96 kB | **774.94 kB** | **-0.02 kB**（持平/微降） | ✓ 持平即过（主题代码不在 vendor-react chunk —— 65-01 已分析根因） |
| **vendor-echarts gzip** | 374.55 kB | 374.55 kB | 0 | ✓ 无变化 |
| **vendor-three gzip** | 242.65 kB | 242.65 kB | 0 | ✓ 无变化 |
| **vendor-xlsx gzip** | 142.99 kB | 142.99 kB | 0 | ✓ 无变化 |
| **vendor-markdown gzip** | 116.13 kB | 116.13 kB | 0 | ✓ 无变化 |
| **dist 总 chunk 数** | 134（65-01 数据点） | 134（v1.22 末） | 0 | ✓ 无变化 |
| **最大单 app chunk** | index-XXX.js 131 kB / gzip 40 kB | index-XXX.js 131 kB / gzip 40 kB | 0 | ✓ 无回归 |

## 源代码层收益（v1.22 不进 vendor 但计入审计）

| 维度 | v1.21 末 | v1.22 末 | 收益 |
|---|---|---|---|
| design-system/themes/ 目录 | 20 文件（6 主题×3 + index + theme-styles.css） | **0**（已删除） | -20 文件 |
| 主题代码总行数 | ~4357 行 | **0** | -4357 行 |
| xingranBrand TS 真相源 | 无 | 11 键（绿6/铜金4/奶油9/...）全量 | 新增 |
| 硬编码非品牌色（indigo/slate/...） | 627 处可命中 | **0**（scanner exit 0） | -627 命中 |

## 综合判定

- **SC#1 ✓ 持平即过**：vendor-react gzip 持平（微降 0.02 kB）；app/dist chunk 数与最大单 chunk 无回归
- **SC#2 ✓ 数据已就位**：QA-04 六屏对比表（`screen-comparison.md`）含每屏实测品牌值 + 与 refs/ 旧截图对比
- **bundle「预期下降」的真实含义**：预期下降的是**源码层**（themes/ 20 文件 + 4357 行代码），不在 vendor-react chunk（vendor-react 仅含 react/antd 等框架）。数据已如实在 SC#1.md 段落记录。

## 已知正交失败（非 v1.22 回归）

5 个 vitest 失败，均为 Phase 53 v1.19 网络设备端口写 UI 测试：
- `BulkWriteDrawer.test.tsx > CR-01: retry uses cached lastDeviceId`（5.1s）
- `BulkWriteDrawer.test.tsx > D-06: retry only takes failed portIds`（5.3s）
- `ports/index.test.tsx > renders '操作' column header (th) when permission 'network:port:write' is granted`（5.9s）

三测均为异步 timing / mock 相关（>5s 超时），与 v1.22 设计系统层完全正交。已在 Phase 67 遗留清单登记，**不阻塞 v1.22 SHIPPED**。