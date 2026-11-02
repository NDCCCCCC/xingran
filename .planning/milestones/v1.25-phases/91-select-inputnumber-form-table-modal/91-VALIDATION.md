# Phase 91: Select / InputNumber / Form / Table / Modal 域交还 - 验证矩阵

> 记录 Phase 91 各 Plan 的任务级验收命令、预期输出与失败条件。

---

## 验证锚点矩阵

| Plan | Task | 目标 | 验证命令 | 期望输出 | 失败条件 |
|---|---|---|---|---|---|
| **91-01** | 1.1 | AntdThemeBridge 扩展表单域 token | `cd xingran-react-frontend && npm run type-check` | exit 0 | TypeScript 类型检查报错 |
| **91-01** | 1.2 | AppSelect 组件兼容透传 | `cd xingran-react-frontend && npm run type-check` | exit 0 | 类型或语法错误 |
| **91-01** | 1.3 | index.css 精简表单覆盖规则 | `cd xingran-react-frontend && node scripts/css-contract-check.js && npm run lint:ci` | ALL PASS | C11 预算超限或 CSS 契约报错 |
| **91-02** | 2.1 | AntdThemeBridge 重构 Table 消除写死 cream | `cd xingran-react-frontend && npm run type-check` | exit 0 | 类型错误或未按 board 派生 |
| **91-02** | 2.2 | index.css 清理 Table 手写覆盖与硬编码 hex | `cd xingran-react-frontend && node scripts/css-contract-check.js && npm run lint:ci` | ALL PASS | 出现新增 hex 或 C13 报错 |
| **91-03** | 3.1 | AntdThemeBridge 扩展 Modal / Dropdown | `cd xingran-react-frontend && npm run type-check` | exit 0 | 类型或语法错误 |
| **91-03** | 3.2 | styles.module.css 违规选择器迁出与 Modal 整理 | `cd xingran-react-frontend && node scripts/css-contract-check.js` | ALL PASS | C10 契约报错（仍含 .ant-） |
| **91-03** | 3.3 | 量化指标下调与全系统回归基线 | `cd xingran-react-frontend && npm run lint:ci && npm run test && cd .. && go build ./...` | 全部绿灯 | 任何门禁非零退出 |

---

## 全局量化指标监控（M1-M16）

- **M1 (!important 全仓)**：起点 1391 → 预期降至 ≤ 1360
- **M7 (index.css hex)**：起点 304 → 预期降至 ≤ 300（消除 `#f3eee3` 等）
- **C10 (module.css 无 .ant-)**：`styles.module.css` 6 处违规选择器归零
- **C14 (antd 6 弃用 prop)**：全站保持 0 Warning 命中
