---
phase: 69-dict-and-status-governance
plan: 06
subsystem: frontend-status-constants
tags: [DICT-03, frontend, shared-constants, vitest, refactor]
requires:
  - "69-01 后端常量家族（label 语义对齐基准：base.go / workstation.go 行尾中文注释）"
provides:
  - "src/constants/status.ts 三组共享常量（ENABLE_DISABLE / NORMAL_STOP / WORKSTATION_STATUS 各含 OPTIONS + TAG_CONFIG）"
  - "src/constants/status.test.ts 12 断言锁值（label 中文字面 + value 数值 + tag 键集合/text 一致性）"
  - "7 个 A 簇 status 定义文件收敛为共享模块别名引用（本地导出名不变、页面文件零改动）"
affects:
  - "69-07（user/workstations 等 constants 文件的 STATUS 导出已收敛，只剩 GENDER/TYPE 等 type 类导出待 useDict 迁移）"
  - "69-08（CLAUDE.md Status Convention 指针化时可引用前端 src/constants/status.ts）"
tech-stack:
  added: []   # 零新依赖：vitest 仓内既有
  patterns:
    - "共享常量模块 + 页面 constants 别名 re-export（导出名与类型面不变 → 页面零改动）"
    - "字符串 value 契约派生：MENU_STATUS_OPTIONS = NORMAL_STOP_OPTIONS.map(o => String(o.value))"
key-files:
  created:
    - xingran-react-frontend/src/constants/status.ts
    - xingran-react-frontend/src/constants/status.test.ts
  modified:
    - xingran-react-frontend/src/pages/system/user/constants.ts
    - xingran-react-frontend/src/pages/system/role/constants.ts
    - xingran-react-frontend/src/pages/system/dict/constants.tsx
    - xingran-react-frontend/src/pages/system/dept/constants.tsx
    - xingran-react-frontend/src/pages/system/menu/constants.tsx
    - xingran-react-frontend/src/pages/operations/workstations/constants.tsx
    - xingran-react-frontend/src/pages/operations/floors/constants.ts
decisions:
  - "workstations 状态是三态业务簇（models.WorkstationStatus: 0=空闲/1=占用/2=维护），不套 NORMAL_STOP 两态组——按 plan 自身「以 model 注释为准」规则新增 WORKSTATION_STATUS 独立组，零 UX 变化"
  - "menu MENU_STATUS_OPTIONS 保留字符串 value 形态（\"0\"/\"1\"，搜索表单既有契约），从 NORMAL_STOP_OPTIONS 派生（String(value)）而非直接别名——数值直接别名会改变 Select value 类型破坏页面"
  - "menu 无任何 VISIBLE 导出存在（全前端 grep 大写 VISIBLE = 0 命中）；在 DEFAULT_FORM_VALUES.visible 处加注释锚定反转语义到 models.VisibleShow=1(显示)/VisibleHidden=0(隐藏)，满足 VISIBLE 保护验收且不引入行为改动"
  - "status 不进 sys_dict（Q2/T-69-13）：status.ts 零字典消费，唯一 sys_dict 字样是决策注释本身"
metrics:
  duration: 约 15 分钟（2026-08-19 05:50–06:05 UTC）
  completed: 2026-08-19
---

# Phase 69 Plan 06: DICT-03 前端 status 共享常量线 Summary

一句话：`src/constants/status.ts` 落地三组启停/业务状态共享常量（ENABLE_DISABLE、NORMAL_STOP、WORKSTATION_STATUS 三态组）并用 12 条 vitest 断言锁死与后端 models 常量注释的对齐关系，7 个页面 constants 文件收敛为别名引用（导出名不变、页面零改动、三份 label 漂移拷贝消除）。

## Task × Commit 对照表

| Task | 内容 | Commit |
|------|------|--------|
| T1 | status.ts + status.test.ts 新建 + 7 文件共享引用改造 | `1aa6f3e` |

单 commit（plan 指定 1 个 commit：T1 是唯一 task，门禁全绿后一次提交）。

## status.ts 常量组 ↔ 后端 models 常量对齐表

| 共享常量组 | 值/label | 对齐后端常量（注释即对齐凭证，前端不引 Go 代码） | 消费方 |
|-----------|----------|------------------------------------------------|--------|
| ENABLE_DISABLE_OPTIONS / _TAG_CONFIG | 0=启用(success) / 1=禁用(error) | `models.UserStatusEnabled=0 // 启用` / `UserStatusDisabled=1 // 禁用`, internal/models/base.go | system/user、system/dict |
| NORMAL_STOP_OPTIONS / _TAG_CONFIG | 0=正常(success) / 1=停用(error) | `models.RoleStatus / DeptStatus / PostStatus / MenuStatus: 0=正常, 1=停用`, internal/models/base.go | system/role、system/dept、system/menu(status)、operations/floors |
| WORKSTATION_STATUS_OPTIONS / _TAG_CONFIG | 0=空闲(success) / 1=占用(error) / 2=维护(warning) | `models.WorkstationStatusAvailable/Occupied/Maintain = 0/1/2`, internal/models/workstation.go | operations/workstations |

后端侧由 69-01 的 `internal/models/status_constants_test.go` AST 锁值测试守卫（74 常量）；前端镜像由本 plan `status.test.ts` 字面锁定——两侧测试共同封死「前端镜像与后端常量漂移」通道（T-69-21）。

## 7 文件 status 收敛映射表

| 文件 | 迁移前 label | 收敛为 | 导出名（均不变） | 页面改动 |
|------|-------------|--------|-----------------|---------|
| system/user/constants.ts | 启用/禁用 | ENABLE_DISABLE 别名 | STATUS_OPTIONS / STATUS_TAG_CONFIG | 0 |
| system/dict/constants.tsx | 启用/禁用 | ENABLE_DISABLE 别名 | STATUS_OPTIONS / STATUS_CONFIG | 0 |
| system/role/constants.ts | 正常/停用 | NORMAL_STOP 别名 | STATUS_OPTIONS | 0 |
| system/dept/constants.tsx | 正常/停用 | NORMAL_STOP 别名 | STATUS_OPTIONS | 0 |
| system/menu/constants.tsx | 正常/停用（字符串 value "0"/"1"） | NORMAL_STOP 派生 `String(value)`（保留字符串契约） | MENU_STATUS_OPTIONS | 0 |
| operations/workstations/constants.tsx | 空闲/占用/维护（三态） | **WORKSTATION_STATUS 组别名**（非 NORMAL_STOP，见 Deviation 1） | STATUS_OPTIONS + 两个 getter | 0 |
| operations/floors/constants.ts | 正常/停用 | NORMAL_STOP 别名 | STATUS_OPTIONS | 0 |

**workstations label 对齐决策：** 现状 label（空闲/占用/维护）与 `models.WorkstationStatus` 注释**完全一致**，无 UX 文案变化；plan 映射表预设的「NORMAL_STOP 组」基于 status=0/1 的错误假设，按 plan 自身「以 models.WorkstationStatus 注释为准」规则改判为独立三态组。附带删除该文件私有的 STATUS_TEXT_MAP / STATUS_COLOR_MAP 两个重复映射，getter 改用共享 WORKSTATION_STATUS_TAG_CONFIG（fallback 行为 `|| "-"` / `|| "default"` 不变）；TYPE_OPTIONS/TYPE_TEXT_MAP（type 类，69-07 工作面）未动。

## VISIBLE 反转例外完好的证据

迁移前全前端大写 `VISIBLE` grep 为 0 命中（menu/constants.tsx 从无 VISIBLE 导出，反转语义在表单 boolean + 提交侧转换）。本 plan 处置：

```console
$ grep -c "VISIBLE" xingran-react-frontend/src/pages/system/menu/constants.tsx
1
$ grep -n "VISIBLE" xingran-react-frontend/src/pages/system/menu/constants.tsx
58:  // VISIBLE 反转例外（严禁与 status 0/1 统一）:
59:  // 对齐 models.VisibleShow=1(显示) / VisibleHidden=0(隐藏), internal/models/base.go
```

即：DEFAULT_FORM_VALUES.visible 处新增注释，把反转例外显式锚定到后端常量名（T-69-15 缓解——VISIBLE 语义未被统一、且首次在前端有了对齐凭证）；`visible: true` 的 boolean 承载与提交侧转换零改动。

## 验证结果

| 检查 | 结果 |
|------|------|
| `npm run type-check` | 退出码 0 |
| `npx vitest run src/constants/status.test.ts` | 1 file / **12 tests 全绿**（两组 0/1 字面锁定 + 三态 0/1/2 锁定 + tag 键/text 一致 + 组间不漂移） |
| `npx vitest run`（全量回归） | **15 files / 132 tests 全绿**（含 port-write / reconciliation / login 等全部既有套件） |
| `npm run lint` | 0 errors / 1044 warnings（全部为存量基线；3 个 .tsx constants 文件的 react-refresh/only-export-components 警告系混合导出既有模式，导出结构未变仅行号偏移） |
| `grep -rl 'from "@/constants/status"' src/pages/` | 命中 **7 个文件**（floors/workstations/dept/dict/menu/role/user constants） |
| `grep -c "VISIBLE" src/pages/system/menu/constants.tsx` | 1（≥1 达标） |
| `grep "sys_user_status" src/constants/status.ts` | 0 命中（status 不进字典，Q2/T-69-13 源码断言；文件内唯一 `sys_dict` 字样是决策注释「不进 sys_dict 字典」本身） |
| user/role/dict 三文件 STATUS_OPTIONS label | 启用/禁用、正常/停用、启用/禁用（别名自共享模块，字面与迁移前一致，零 UX 变化，由 status.test.ts 锁定） |
| commit hooks | lint-staged（eslint --fix + type-check + prettier）全过，正常 hooks 提交（未用 --no-verify） |

**工作区护栏遵守：** `src/pages/settings/index.tsx`、`src/pages/system/settings-page/index.tsx` 等 default-theme 遗留改动与本 plan 9 个文件互斥，commit 仅逐路径 add 计划内文件（git status 遗留项状态保持不变）。

## Deviations from Plan

1. **[Rule 1 - 计划事实修正] workstations 是三态业务簇，不收敛进 NORMAL_STOP**：plan 映射表预设 workstations → NORMAL_STOP 组别名（「若现状 label 与组不符，以 models.WorkstationStatus 注释为准」）。实查 `internal/models/workstation.go`：`WorkstationStatusAvailable=0 // 空闲 / Occupied=1 // 占用 / Maintain=2 // 维护`——三态语义（占用/维护不是停用），两态化会丢状态并破坏页面。按 plan 自身「model 注释为准」规则，在 status.ts 新增第三组 WORKSTATION_STATUS_OPTIONS/TAG_CONFIG 并别名引用；接受 grep 验收仍命中 7 文件，零 UX 变化。
2. **[Rule 3 - 计划事实修正] menu 的 STATUS 导出是字符串 value 的 MENU_STATUS_OPTIONS，且全前端无 VISIBLE 导出**：plan 假设 menu/constants.tsx 有 STATUS + VISIBLE 两组导出、VISIBLE 需原样保留。实查：导出为 `MENU_STATUS_OPTIONS`（value 为字符串 "0"/"1"，menu/index.tsx:289 搜索表单契约），大写 VISIBLE 全前端 0 命中。处置：MENU_STATUS_OPTIONS 改为 `NORMAL_STOP_OPTIONS.map(o => ({ label: o.label, value: String(o.value) }))` 派生（引用共享模块 + 保留字符串契约 + 页面零改动）；VISIBLE 保护以注释锚定方式落地（见上节证据）。
3. **[范围裁剪] dept/constants.tsx 的 renderStatusTag 内联三元未改用共享 TAG_CONFIG**：该文件无 STATUS_TAG_CONFIG 导出，renderStatusTag 文案（正常/停用）与共享组一致；plan 改造形态只要求 STATUS_OPTIONS/STATUS_TAG_CONFIG 替换，最小 diff 保留内联实现（69-07 或后续可统一）。

## Known Stubs

无。

## Threat Flags

无新增威胁面。threat_model 三项缓解均落地：T-69-13（status.ts 零字典消费）、T-69-15（VISIBLE 未被统一 + 注释锚定）、T-69-21（status.test.ts 字面锁值 + 69-01 后端 AST 锁值双向守卫）。

## Self-Check: PASSED

9 个 plan key-files + SUMMARY.md 全部存在；2 个 commit（1aa6f3e 任务提交 / 8fa23a4 文档提交）全部在 git log 中命中。
