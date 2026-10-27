# Phase 68 T3 hex 豁免清单

判定依据 UI-SPEC hex 三问；以下非豁免目录中保留的裸 hex / 双写兜底，全部登记豁免理由。
基线 526 处 → 批①②③后非豁免域残留 32 处，全部在本清单内。

## 0. DEBT-01 lint 豁免链（eslint.config.js，Phase 68 新增规则后实测 83 errors/0 新增）

- 规则：hex selector 追加进 Phase 65 契约块 `no-restricted-syntax` 数组（flat config 后块整体覆盖同名键，勿另起块）。
- 层面① 封装层：`src/components/ui/**`、`src/components/shared/**`（既有 65 豁免块顺带豁免 hex；shared 的 ExcelExport/ImageGallery/FloorPlanEditor 命中因此被覆盖）。
- 层面③ 结构性 + 存量：「Phase 68 hex 豁免 override」块（结构性 glob + 14 个未 grandfather 存量文件）；命中 P65 存量豁免块的文件按先例 65+hex 均豁免（清单固定、新文件不豁免）。
- 测试文件：既有整条 off 的 override 天然覆盖。
- 层面② 逐行 eslint-disable 预留：新代码确需裸 hex 时用 `// eslint-disable-next-line no-restricted-syntax -- <理由>` 并登记本文件。

## A. 语义色差（diff/徽章/装饰），映射主题 token 会失真

| 位置 | 色值 | 理由 |
| --- | --- | --- |
| pages/network/backups/index.tsx:277-302 | #ffced0/#d4edda/#c92a2a/#2b7a41 | 配置 diff 高亮（删除红底/新增绿底/删除红字/新增绿字），语义为 diff 标注非主题状态色 |
| pages/network/backups/modals/RestoreModals.tsx:475,492 | #ffffff | diff 面板白底卡片，与代码块渲染强绑定 |
| components/reconciliation/HealthBadge.tsx:38 | #d4d4d8 | default 态中性灰徽章，无对应主题 token |
| components/shared/ImageGallery.tsx:121 | #fff7e6/#fff | 主图高亮底（warning 淡色）+ 纯白底，画廊特有 |
| components/shared/ExcelExport.tsx:119 | #f0f7ff | 导出提示信息框淡蓝底，无对应 token |
| pages/asset/reconciliation/dashboard/index.tsx:309 | #722ed1 | Trophy 装饰紫，无 token |
| pages/duty/my-duty/index.tsx:327 | #722ed1 | 统计卡装饰紫 |
| pages/duty/management/components/WeeklyView.tsx:75、pages/duty/schedules/index.tsx:401 | #e6f7ff | isToday「今天」高亮底（antd today 色），无 token |

## B. 彩色导航卡分类色（主题化意图弱，随主题渐变会破坏辨识度）

| 位置 | 色值 |
| --- | --- |
| components/layout/InnovativeLayout.tsx:55,71,79,95,111,174 | #10b981/#c09058(var(--theme-purple) 伪双写，变量不存在)/#ec4899/#14b8a6/#14542e |
| components/layout/shared/QuickNav.tsx:20,27,29 | #10b981/#c09058(伪双写)/#ec4899 |

注：--theme-purple 在 src/index.css 未定义，属伪双写，兜底值即实际生效色，故不清偿兜底（清了会失效）。

## C. echarts/svg/canvas 绘图上下文（T3 脚本 ECHARTS_RE 整行跳过）

| 位置 | 内容 |
| --- | --- |
| pages/asset/reconciliation/dashboard/index.tsx:177,184,191 | 图表 itemStyle 三色（#337ab0/#f5222d/#2d8949） |
| pages/network/mac/heatmap.tsx、components/network/MACHeatmapChart.tsx | 热力图绘图色与双写兜底 |
| pages/monitor/cache/components/MonitorOverview.tsx:3 | 注释行（非代码） |

## D. 豁免目录（结构性画布/3D/图标选择器，不改）

building-spaces-3d/**、cad-editor/**、cad-elements/**、IconSelect/**、
FloorPlanEditor.tsx、FloorPlanView.tsx、MACHeatmapChart.tsx、network/mac/heatmap.tsx、macEventMeta.ts
（含 captcha-background 的 status===1 映射反转域——语义正确，勿修正）
