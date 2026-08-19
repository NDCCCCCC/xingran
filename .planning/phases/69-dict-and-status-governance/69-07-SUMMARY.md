---
phase: 69-dict-and-status-governance
plan: 07
subsystem: frontend-dict-consumption
tags: [DICT-03, useDict, frontend, dict-migration, static-fallback]
requires:
  - "69-02: migration_208 seed 的 11 组字典数据（本 plan 消费其中 4 组：sys_user_sex / ops_workstation_type / duty_holiday_type / network_device_type）"
  - "69-06: src/constants/status.ts 共享常量（本 plan 触碰的 4 个 constants 文件 STATUS 导出不动）"
provides:
  - "四页 type 类下拉 useDict 迁移 + 静态兜底（字典管理页改 label → 消费页下拉/表格随之变化）"
  - "formatGender / getDeviceTypeText 字典优先 label 渲染（dictLabel || 静态映射回退）"
  - "新增表单默认值走字典 isDefault 项（user/workstations/holidays 三处；devices 不引入新默认）"
affects:
  - "69-08（DICT-04 字典链路端到端 checkpoint——本 plan 是其 UI 验收前提，checkpoint 第 3 步「字典改值下拉变化」由本 plan 交付）"
tech-stack:
  added: []   # 零新依赖：useDict hook / React Query / antd 均仓内既有
  patterns:
    - "dedicated-lines 三件套（Option 渲染 + dictLabel 回退 + isDefault 默认值）+ 第四件：字典空态回退静态 OPTIONS 三元分支"
    - "非 hook 上下文（utils.tsx 纯函数 / 独立 Modal）经可选参数/prop 透传 DictItem[]，调用方 hook 拉取"
    - "number 字段消费 string dictValue：Option value 用 Number(d.dictValue)，提交值类型与迁移前一致（T-69-16）"
key-files:
  created: []
  modified:
    - xingran-react-frontend/src/pages/system/user/index.tsx
    - xingran-react-frontend/src/pages/system/user/utils.tsx
    - xingran-react-frontend/src/pages/system/user/constants.ts
    - xingran-react-frontend/src/pages/operations/workstations/index.tsx
    - xingran-react-frontend/src/pages/operations/workstations/modals/EditModal.tsx
    - xingran-react-frontend/src/pages/operations/workstations/constants.tsx
    - xingran-react-frontend/src/pages/duty/holidays/index.tsx
    - xingran-react-frontend/src/pages/duty/holidays/modals/EditModal.tsx
    - xingran-react-frontend/src/pages/duty/holidays/modals/BatchModal.tsx
    - xingran-react-frontend/src/pages/duty/holidays/constants.tsx
    - xingran-react-frontend/src/pages/network/devices/index.tsx
    - xingran-react-frontend/src/pages/network/devices/constants.ts
decisions:
  - "workstations 的 useDict 调用点放 index.tsx 经 prop 透传 EditModal（plan action 字面说「hook 在弹窗组件内调用」与 artifact「index.tsx contains useDict(」冲突，取后者为硬验收门，单一 hook 调用点更省缓存）"
  - "devices 不引入新增默认值：该表单 deviceType 原本无默认（required 字段），强行 isDefault 预选 router 有误提交风险——三件套的默认值件仅应用于原本就有默认值的三处（user/workstations/holidays）"
  - "number 字段（gender / workstation type）Option value 用 Number(dictValue) 保住提交契约；string 字段（holidayType / deviceType）直接用 dictValue"
  - "holidays EditModal 新增默认从静态 \"legal\" 改为 isDefault \"custom\"（plan 明确指示 isDefault 优先，对齐后端 Holiday.HolidayType gorm default:'custom'）——行为变化点已记录"
metrics:
  duration: 13 分钟（2026-08-19 07:28–07:41 UTC）
  completed: 2026-08-19
---

# Phase 69 Plan 07: DICT-03 前端字典线 — 四页 type 下拉迁移 useDict Summary

一句话：四页 8 处 type 下拉 + 2 处表格 label 迁移 useDict（dedicated-lines 三件套 + 字典空态回退静态 OPTIONS 第四件），Option value 经 Number() 对齐保住提交契约，type-check / lint（0 errors）/ vitest（132 tests）全绿。

## Task × Commit 对照表

| Task | 内容 | Commit |
|------|------|--------|
| T1 | user 页性别 + workstations 页工位类型迁移（6 文件，+101/-22） | `3a133dc` |
| T2 | holidays 页假日类型 + devices 页设备类型迁移（6 文件，+93/-28） | `235b8f7` |

## 四页迁移台账

| 页面 | dictType | 涉及文件 | 迁移点 | fallback 方式 | 提交值类型结论 |
|------|----------|---------|--------|--------------|----------------|
| 系统用户 | sys_user_sex（seed isDefault="2" 保密） | user/index.tsx、user/utils.tsx、user/constants.ts | 编辑弹窗性别 Select（Option value=`Number(d.dictValue)`）；表格性别列 `formatGender(gender, genderDict)`；新增默认 `defaultGender`=isDefault 项（"2"）空态回退 0 | Select 三元分支（dict 非空渲染 dict / 空渲染 GENDER_OPTIONS）；label 函数 dict 命中返回 dictLabel，未命中/空态回退静态 map→"保密" | **number 不变**——handleSave 原有 `Number(values.gender)` 保留 + Option value 即 number（T-69-16 双保险） |
| 工位管理 | ops_workstation_type（seed isDefault="0" 固定工位） | workstations/index.tsx、modals/EditModal.tsx、constants.tsx | EditModal 工位类型 Select（dict 经 `typeDict` prop 透传）；新增默认 `defaultType`=isDefault 项（"0"）空态回退 0（index.tsx handleOpenModal 与 EditModal useLayoutEffect 双处对齐） | Select 三元分支回退 TYPE_OPTIONS | **number 不变**——Option value=`Number(d.dictValue)`，与静态 TYPE_OPTIONS number 值同型 |
| 假日管理 | duty_holiday_type（seed isDefault="custom"） | duty/holidays/index.tsx、modals/EditModal.tsx、modals/BatchModal.tsx、constants.tsx | EditModal 类型 Select + BatchModal 行内类型 Select（dict 均经 `holidayTypeDict` prop 透传）；EditModal 新增默认 = isDefault 项（"custom"）空态回退 "legal" | Select 三元分支回退 HOLIDAY_TYPE_OPTIONS | **string 不变**——dictValue 与 HolidayType 码（legal/workday/custom）同型直用 |
| 网络设备 | network_device_type（seed isDefault="router"） | network/devices/index.tsx、constants.ts | 搜索表单设备类型 Select + 手动新增/编辑弹窗 Select（字符串 dictValue 直用）；表格类型 Tag 与详情弹窗 label 走 `getDeviceTypeText`（useCallback 包裹，dictLabel ‖ getOptionLabel 静态回退） | Select 三元分支回退 DEVICE_TYPE_OPTIONS；label 函数字典优先 | **string 不变**——dictValue 与 deviceType 码（router/switch/...）同型直用；**不引入新增默认值**（见 decisions） |

## 明确不迁清单及理由

| 不迁项 | 文件/位置 | 理由 |
|--------|----------|------|
| VENDOR_OPTIONS（设备厂商） | network/devices/constants.ts、index.tsx 搜索/编辑 Select | 无对应字典组（69-02 seed 11 组中不含厂商），保持静态现状；plan 明确圈定不迁 |
| workstations columns.tsx 类型 Tag | workstations/columns.tsx:90 `renderWorkstationTypeTag` | plan files_modified 未含 columns.tsx；非 hook 上下文且 getWorkstationColumns 消费于 useMemo——迁入需改 columns 签名，留后续批次 |
| holidays columns.tsx 类型 Tag | duty/holidays/columns.tsx `renderHolidayTypeTag` | plan 明确「columns.tsx/utils.tsx 的 Tag 渲染保留静态映射兜底（非 hook 上下文）」 |
| holidays useHolidayModals 行默认 "legal" | duty/holidays/hooks/useHolidayModals.ts:55,137,156 | hook 文件不在 plan 文件清单；EditModal 的 isDefault 默认在表单打开时覆盖生效，批量行初始值保持静态 |
| workstations 视图层类型文本 | views/CardView.tsx:70、views/FloorPlanView.tsx:155 `getWorkstationTypeText` | views 不在 plan 文件清单（非 hook 上下文），后续批次候选 |
| devices STATUS_OPTIONS / VENDOR Tag | devices/constants.ts STATUS 族 | status 类不进字典（Q2 决策，69-06 已收敛共享常量）；vendor 无字典组 |
| 其余 OPTIONS 组（全仓 ~70 组未迁） | 各模块 constants | 后续批次候选——本 plan 只圈定 4 个已 seed 的 type 类 dictType（planner Q4 范围） |

## 验证结果

| 检查 | 结果 |
|------|------|
| `npm run type-check`（T1 后 / T2 后各一次） | 通过 |
| `npm run lint` 全量 | 0 errors / 1044 warnings（与迁移前基线持平，零新增；改动文件告警全为既有类别） |
| `npx vitest run`（T1 后 / T2 后各一次） | 15 files / 132 tests 全 PASS（run 模式） |
| grep useDict( 四页 index.tsx | user=2 / workstations=2 / holidays=2 / devices=2（各 ≥1 达标） |
| 四组原 OPTIONS 常量导出保留 | GENDER_OPTIONS / TYPE_OPTIONS / HOLIDAY_TYPE_OPTIONS / DEVICE_TYPE_OPTIONS 均在（降级 fallback 注释标注，未删除） |
| 每个迁移 Select 空态回退分支 | 8 处 Select 均为「dict.length > 0 ? dict.map : 静态 OPTIONS.map」三元结构（源码断言通过） |
| VENDOR/columns/utils/hooks 未动 | git status 确认 holidays/devices 目录仅 6 个计划内文件变更；VENDOR 在 diff 中仅出现于新增注释 |
| 工作区遗留改动隔离 | settings/settings-page/defaultThemeApi 13 文件保持原样未触碰；两次 commit 均逐文件 git add，暂存区无外来条目 |

## Deviations from Plan

1. **[计划内字面冲突调和] workstations 的 useDict 调用点**：plan action 写「EditModal 是独立组件，hook 在弹窗组件内调用」，但 artifact/验收门要求 `workstations/index.tsx` contains `useDict(`（页面搜索表单本无类型 Select，若 hook 只放 EditModal 则 index.tsx 无命中点）。落地取验收门为硬约束：index.tsx 调 `useDict("ops_workstation_type")`（同时供新增默认值计算）经 `typeDict` prop 透传 EditModal——单一 hook 调用点，React Query 缓存无重复请求。
2. **[Rule 3 - 提交受阻] commit header 缩短**：plan done 消息 header 102 字符超 commitlint header-max-length 100，去掉 "with static fallback" 后 81 字符通过（body 中保留完整语义）。首次提交被 pre-commit lint-staged 拦截后暂存区完好，重提无损失。
3. **[计划细化] devices 不应用 isDefault 新增默认值**：三件套的默认值件要求「新增表单默认值优先 isDefault」，但 devices 手动新增表单 deviceType 原本无默认值（required 字段）——引入预选 router 属新增行为且有误提交风险，判定为 UX 变更而非本 plan 目标，仅应用于原本就有默认值的三处（user 0→"2"、workstation 0→"0" 不变、holiday "legal"→"custom"）。

## Auth Gates

无。

## Known Stubs

无。所有迁移点均接线真实字典 API（POST /system/dicts/data/list），静态数组仅作空态/异常兜底（非 stub，是 T-69-14 mitigate 的要求形态）。

## Threat Flags

无新增威胁面。threat_model 处置落地：T-69-14（8 处 Select 空态回退 + 四组静态导出保留）、T-69-16（提交值类型三处结论见台账，type-check 门通过 + diff 人工复核）、T-69-SC（零依赖安装）。

## Self-Check: PASSED

12 个 key-files（全部 modified）存在于工作树；2 个 task commit（3a133dc / 235b8f7）均在 git log 命中。
