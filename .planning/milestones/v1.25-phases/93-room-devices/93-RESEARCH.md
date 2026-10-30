# Phase 93: room-devices 拆分 + 视觉回归验收 - Technical Research

> 记录 Phase 93 的技术架构分析、大文件解耦方案、视觉终验比对清单与风险防线。
> 日期：2026-10-30

---

## 一、代码拆分现状分析（SPLIT-01）

### 1. 文件规模与结构解构
当前 `ruoyi-react-frontend/src/pages/operations/room-devices/index.tsx` 共有 **896 行**，承担了过多职责：

| 代码行段 | 职责类别 | 拟解耦目标 | 预估行数 |
|---|---|---|:---:|
| `:30-140` | 类型定义 `RoomDevice`、字典映射表、文本转换辅助函数 | `constants.ts` | ~90 行 |
| `:140-418` | 状态初始化、API 交互（增删改查/批量/导出）、弹窗控制、联动逻辑 | `hooks/useRoomDeviceActions.ts` | ~200 行 |
| `:419-505` | Table 列定义（含三态操作按钮门控） | `columns.tsx` | ~110 行 |
| `:506-605` | 卡片视图渲染 `renderCardView` | `components/RoomDeviceCardView.tsx` | ~110 行 |
| `:606-896` | 顶部统计卡片、搜索区定义、主 JSX 渲染与 Modal 挂载 | `index.tsx` (主容器) | **~240 行** (≤350) |

### 2. 外部注入协议与关键细节（零行为变化保证）
- **`reloadRef` 注入链路**：
  - `:171` `const reloadRef = useRef<() => void>(() => {});`
  - `:240` 在批量导入、保存成功后调用 `reloadRef.current()` 刷新数据；
  - `:615` 在 `loadData` 定义后将其实例赋给 `reloadRef.current`；
  - 该链路必须 100% 完整保留。
- **`columns` 的缓存语义**：
  - 列定义中引用的 `permEdit` / `permDelete` 等三态门控权限对象与操作回调函数，在抽离到 `columns.tsx` 工厂时，必须保持参数化注入，确保权限变更与行操作联动无任何滞后。

---

## 二、视觉终验方案与移交项承接（STYLE-05）

### 1. 21 形态 computed-style 机器断言
- 复用在 Phase 90/92 中验证成熟的矩阵工具链；
- 严格断言 STYLE-05 三大硬指标：
  1. 暗色模式下 `--input-bg` 解析值 = `#0f2e1b`；
  2. InputNumber 错误态边框色 = `rgb(186, 54, 48)`（`var(--ant-color-error)` 品牌红）；
  3. 输入类/选择类控件 `borderTopLeftRadius` = `8px`（内层 input 为 0px，焦点环单圈）。

### 2. Phase 92 移交的 14 个业务页面巡检
承接 `92-HANDOFF.md` 登记项：
- 巡检涵盖：用户管理、角色管理、部门管理、菜单管理、岗位管理、楼宇、楼层、工位、资产、机房、专线、机房设备、网络设备、命令执行等 14 个核心页面；
- 产出 `93-VISUAL-INSPECTION-REPORT.md`，确认全站无暗色白块、无双框元素、无未登记断层。

---

## 三、两波执行规划

```text
Phase 93: room-devices 拆分 + 视觉回归验收
 ├── Plan 93-01 (Wave 1: 代码重构拆分): room-devices 纯重构拆分至 index.tsx ≤350 行，子文件 ≤250 行，单测全绿
 └── Plan 93-02 (Wave 2: 视觉终验与里程碑收官): STYLE-05 21形态机器断言 + 14页巡检报告，全系统双端门禁，归档 93-VERIFICATION.md
```
