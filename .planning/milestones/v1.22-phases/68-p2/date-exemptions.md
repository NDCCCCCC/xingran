# Phase 68 日期格式豁免登记（T2 / SC2）

> 表头约定同 hex-exemptions.md。仅登记「确属展示性场景保留」或「非展示语境（payload/解析）不改」的项。

| 文件路径:行号 | format | 场景 | 豁免理由（登记人/日期） |
|---|---|---|---|
| pages/duty/management/components/WeeklyView.tsx:17 | `YYYY年MM月DD日` ×2 | 周视图标题「2026年10月26日 - 2026年11月01日」 | 确属展示性场景（中文周区间标题，ISO 风格破坏可读性），按 SPEC §3.2 登记保留（executor / 2026-10-26） |
| pages/duty/schedules/utils.tsx:24 | `YYYY年MM月DD日` ×2 | 排班周标题（同上句式） | 同上，展示性中文周标题（executor / 2026-10-26） |
| pages/system/notice/index.tsx:257 | `YYYY-MM-DDTHH:mm:ss` | 定时发布时间 API payload（+时区偏移，ISO 8601） | **非展示**：提交给后端的 ISO 8601 payload，T 分隔符不会渲染给用户；改空格格式反而破坏后端解析。已上报 parent 待裁决，暂保留（executor / 2026-10-26） |
| pages/system/notice/index.tsx:280 | `YYYY-MM-DDTHH:mm:ss` | 周期任务 endDate API payload | 同上（executor / 2026-10-26） |

## 说明

- SPEC §3.2 称 2 处 T 分隔符「原样渲染给用户」与实测不符：全仓仅 notice/index.tsx 两处，均为提交 payload 构建。SC2 的「T bug 修复证据」以此表代替：**经核实非展示 bug，不修改**（避免破坏后端解析）。若 parent 判定必须改，另行返工。
- 展示语境的 dayjs 直调已全部收敛至 formatDateTime/formatDate + 常量（FORMAT_DATETIME_MINUTE / FORMAT_DATE_SHORT / FORMAT_TIME_MINUTE）；剩余 `YYYY-MM-DD` / `YYYY-MM-DD HH:mm:ss` 字面量均为**表单提交 payload 构建**或局部既有 helper（pages/operations/rpa/workers/columns.tsx:11 局部 formatDateTime，含 `+HH:MM` 偏移剥离，与 datetime.ts 口径不同，收敛有行为风险，留待后续），属基准格式不违反 SC2。
