# Phase 93: room-devices 拆分 + 视觉回归验收 - 验证矩阵

> 各 Plan 任务级验收命令、预期输出与失败条件。

---

## 验证锚点矩阵

| Plan | Task | 目标 | 验证命令 | 期望输出 | 失败条件 |
|---|---|---|---|---|---|
| **93-01** | 1.1 | 抽离 constants.ts | `cd ruoyi-react-frontend && npm run type-check` | exit 0 | 类型或语法错误 |
| **93-01** | 1.2 | 抽离 columns.tsx 与 CardView | `cd ruoyi-react-frontend && npm run type-check` | exit 0 | 导出或组件引用报错 |
| **93-01** | 1.3 | 抽离 useRoomDeviceActions 与 index.tsx 瘦身 | `cd ruoyi-react-frontend && node -e "const fs=require('fs');const n=fs.readFileSync('src/pages/operations/room-devices/index.tsx','utf8').split('\n').length;if(n>350)throw new Error('line count '+n);console.log('OK line count='+n);"` | OK line count ≤ 350 | index.tsx 行数 > 350 或单测失败 |
| **93-02** | 2.1 | STYLE-05 机器断言与 14 页巡检报告 | `93-VISUAL-INSPECTION-REPORT.md` 存在性检查 | OK inspection report ready | 巡检报告缺失或断言未通过 |
| **93-02** | 2.2 | 全系统双端门禁与 93-VERIFICATION 归档 | `npm run lint:ci && npm run check:gating && node scripts/css-contract-check.js && cd .. && go build ./...` | 全部绿灯，93-VERIFICATION 归档 | 门禁非零退出或编译错误 |

---

## 核心质量与行数契约指标

- **`room-devices/index.tsx` 行数**：起点 896 行 → 终态 **≤ 350 行**
- **各新拆分文件行数**：各子文件实测 **≤ 250 行**
- **`reloadRef` 注入链路**：4 段协议完整保留，零行为变化
- **全站自动化门禁**：
  - 前端：`lint:ci` 0 problems，`check:gating` 25+51=76，`css-contract-check` ALL PASS
  - 样式预算：C11 ≤ 455（实测 452），C13 M7 ≤ 60（实测 26），M9 ≤ 3200（实测 3194）
  - 后端：`go build ./...` 0 error
