---
phase: "93"
slug: "room-devices"
status: complete
created: "2026-10-30"
mode: discuss
---

# Phase 93 — CONTEXT：room-devices 拆分 + 视觉回归验收

> 本文档锁定 Phase 93 的实施决策。作为 v1.25 里程碑的最终收官战，下游 researcher / planner / executor 以此为准，无需再次询问用户。

---

## 已锁定决策（USER DECISIONS）

- **D-01:** room-devices 代码拆分组织架构采用「复用 Phase 83 范式标准拆分」——将 `ruoyi-react-frontend/src/pages/operations/room-devices/index.tsx`（当前 896 行）严格解耦为：
  - `columns.tsx`：列定义工厂（包含操作列与三态权限门控）；
  - `searchFields.ts`：搜索栏字段配置与初始值；
  - `hooks/useRoomDeviceActions.ts`：CRUD 操作、状态切换、导入导出及弹窗控制逻辑；
  - `index.tsx`：主页面 JSX 容器，行数严格控制在 **≤ 350 行**；拆分出的各新文件行数严格控制在 **≤ 250 行**。
  - Reversibility: reversible。

- **D-02:** 严格保证「拆分行为零变化」与关键接口协议——
  - `reloadRef` 外部注入链路（4 段）必须逐字完整保留；
  - `columns` 的 `useMemo` 依赖（若有故意不完整依赖）保持原状，绝不「顺手修正」引发意外无限渲染；
  - 既有单元测试 `room-devices/__tests__` 必须 100% 保持全绿。
  - Reversibility: costly（保证零回归是重构底线）。

- **D-03:** 视觉回归终验采用「全自动机器比对 + 巡检报告」——
  - 复用已建立的 21 形态 computed-style 断言脚本进行自动化双向比对；
  - 对 Phase 92 移交的 14 个业务页面生成标准化比对报告（`93-VISUAL-INSPECTION-REPORT.md`），验证无异常样式断层；
  - 断言 STYLE-05 硬指标全部达成（暗色 `--input-bg = #0f2e1b`、InputNumber 错误态边框 = `rgb(186, 54, 48)`、输入控件统一 8px 圆角）。
  - Reversibility: reversible。

- **D-04:** 执行节奏划分为「2 波递进推进」——
  - **Wave 1 (Plan 93-01)**：room-devices 单体大文件纯重构拆分（SPLIT-01），满足行数契约（主 ≤350，子 ≤250），单测与门控全绿；
  - **Wave 2 (Plan 93-02)**：STYLE-05 明暗双主题视觉终验（21 形态 + 14 页巡检报告）、全系统双端门禁（Go 测试 + 前端 4025+ 测试 + lint:ci 0 warnings + 76 页面门控），归档 `93-VERIFICATION.md` 完成 v1.25 终局收官。
  - Reversibility: costly。

---

## 边界与红线

- 绝不改动 `src/constants/permissionCodes.ts`、`scripts/gating-exemptions.json`、`check-gating-coverage.mjs`。
- 绝不破坏已交付的 VIS-01/02/03/05 双框复位、焦点环与错误态边框契约。
- 保证前后端全套自动化测试（后端 `go test ./...`、前端 vitest 4025+ 测试）100% 绿灯。
- `room-devices/index.tsx` 行数必须实测 ≤ 350 行。
