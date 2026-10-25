# Phase 67 — 存量迁移（按 ROI 顺序落地契约组件）SUMMARY

> 执行时间：2026-10-26 ｜ 执行者：GSD executor ｜ 任务书：`67-PLAN.md` ｜ 契约：`67-UI-SPEC.md`
> 基线：`63-p0/63-P0-SUMMARY.md`（vitest 3900 / lint 83 errors / P65 豁免 142）｜ 试点模式：`66-page-form/66-SUMMARY.md`

## 1. 行数三栏量化表

| 页面 | 迁移前 | 迁移后（index + 拆分文件） | 净变化 |
|---|---|---|---|
| network/discoveries | 405（index 405） | index 333 + columns 86（含测试文件另计） | **405 → 419**（index −72，逻辑外移） |
| network/backups | 1141（index 1141） | index 637 + modals/RestoreModals 601 | **1141 → 1238**（index −504） |
| network/devices | 1282（index 1282） | index 692 + components/DeviceModals 425 + DeviceStatsCards 71 + DeptTreePanel 24 | **1282 → 1212**（index −590） |
| monitor/cache | 974（index 974） | index 454 + components/MonitorOverview 94 + CacheStatsModal 150 | **974 → 698**（index −520） |
| **合计** | **3802** | **3567（index 合计 2116）** | **−235 总行 / index −1686** |

注：纯行数不降反升的页面（backups）源于弹窗拆分文件的 props 接口与注释成本；核心收益在 index 收敛（−1686）与重复管线（分页/排序/三态/确认删除）下沉契约组件。

## 2. Commit 记录

| 任务 | Commit | 范围 |
|---|---|---|
| T1 discoveries | `6e62b9fd` | index 405→333、columns 收窄无参、createDeleteAction、C 类 1 状态筛选接线（缺陷修复） |
| T2 backups | `46441f54` | index 1141→637、RestoreModals.tsx 601、创建备份→FormModal(okText=确定) |
| T3 devices | `3a7d3ba0` | index 1282→692、三组件拆分、batchDelete INV-8、extraFilters 部门联动 |
| T4 cache | `0ac66bad` | index 974→454、MonitorOverview/CacheStatsModal 拆分、searchFields 3 字段 |

## 3. 基线终值表（硬门槛）

| 指标 | 基线（63-P0） | 终值 | 达标 |
|---|---|---|---|
| type-check | 0 errors | 0 errors | ✅ |
| vitest | 3900 passed / 0 failed | **3906 passed / 0 failed**（净增 +6） | ✅ |
| lint errors | 83（存量） | 83（0 新增） | ✅ |
| lint warnings | 1414 | 1411（净减 3） | ✅ |
| P65 豁免（selector 类） | 142 文件 | **140 文件**（T1 −2、T2 +1 机械搬迁、T3 −1、T4 持平） | ✅ 递减 |
| P66 豁免（hook imports 类） | 45 文件 | **41 文件**（T1 −1、T2 −1、T3 −1、T4 −1） | ✅ 递减 |
| 四组件 + useTableManager API | 冻结 | `git diff 6e124a44..HEAD` = 空 | ✅ |
| 页外文件触碰 | 禁止 | 0（仅 eslint.config.js 豁免锚点写回，PLAN 允许） | ✅ |
| index-v2 并存 | 禁止 | 0 | ✅ |
| 回滚契约（§4） | revert --no-commit 可干净 abort | 演练通过，工作区还原 | ✅ |

## 4. SC1–4 Verdict

- **SC1（契约组件 API 零修改）**：PASS。PageTable/SearchForm/FormModal/RemoteSelect/useTableManager 在 6e124a44..HEAD 无任何 diff；四页全部经 props 声明式接入。
- **SC2（INV-1~12 不变式）**：PASS。分页口径（共 N 条 / M 页）、三态（加载/空/错误重试）、删除确认统一入口、批删 INV-8 逐字文案、搜索回第 1 页均有测试断言锁定（discoveries +6 / devices 6 用例 / cache 32 / backups 36）。B 类豁免文案（backups 两处还原确认、cache 清空确认）逐字保留。
- **SC3（四页迁移完成 + 测试净增）**：PASS。3900→3906（+6，全为契约断言）；4 页无回归失败。
- **SC4（可回滚）**：PASS。4 个 commit 单向依赖契约组件、无共享状态外溢；`git revert --no-commit` 演练干净。

## 5. 每页 PR 描述草稿

### T1 discoveries（`6e62b9fd`）
> refactor(p67-t1): discoveries 迁移 66 契约组件
> - index 405→333：usePagination/useServerSort/setTotal 手工管线 → PageTable request 单实例；request 内联动统计刷新与轮询列表桥接（retry=列表+统计都刷语义保持）
> - 操作列迁页面 rowActions：查看结果 + createDeleteAction（确认删除统一口径，文案对齐 65 模板）
> - columns.tsx 收窄无参（删除操作列与手工 sortOrder 视觉，sorter:true 由管线承接）
> - **缺陷修复（C 类 1）**：搜索卡状态筛选原先未接线，本次接通 SearchForm → list 端点
> - 新增测试：分页 showTotal 口径、错误三态、状态筛选接线、删除确认文案（+6）

### T2 backups（`46441f54`）
> refactor(p67-t2): backups 迁移 PageTable/FormModal + RestoreModals 拆分
> - index 1141→637：request 直连 /network/backups/list，groupBackupsByDevice 在 request 内分组
> - modals/RestoreModals.tsx（601 行）纯搬迁四弹窗（还原选择/版本列表/差异对比/配置内容）+ versionColumns
> - 创建备份 Modal → FormModal（okText="确定"，dirtyCheck 默认启用）
> - B 类豁免逐字保留：还原确认两处 Modal.confirm 原文案
> - 修正：原 renderDiffLine 空 diff 行分支误嵌 BatchExportModal，迁移归位根级

### T3 devices（`3a7d3ba0`）
> refactor(p67-t3): devices 迁移 66 契约组件
> - index 1282→692：双 hook 数据源 → PageTable 单实例（request 直连 + loadStatistics 联动）
> - 拆分 components/{DeviceStatsCards,DeptTreePanel,DeviceModals}.tsx
> - 批删 batchDelete={{target:"设备"}} INV-8 逐字文案；单删 createDeleteAction
> - 部门树筛选 extraFilters={deptId}(useMemo) 联动回第 1 页；virtual+scroll 透传
> - BatchExportModal 归位根级（原误嵌搜索卡 div 内）

### T4 cache（`0ac66bad`）
> refactor(p67-t4): monitor/cache 迁移 66 契约组件
> - index 974→454：caches/setTotal/usePagination/useServerSort 手工管线 → PageTable
> - searchFields 3 字段（key/type/level，level 在 request 内兜底 "all"）；操作缓存/统计/导出/清空 4 业务按钮进 toolbar
> - MonitorOverview / CacheStatsModal 拆分（裸 hex 色原样保留）
> - 操作缓存 Modal → FormModal（okText="确定"）；单删 Popconfirm 原样保留（B 类，避免口径回归）
> - Tabs 移除：统计面板为统计 Modal 真子集，信息零损失

## 6. 事实注记汇总（discover 与 PLAN 冲突 / 裁决变化）

1. **BatchDeleteButton target 未接线（T3 发现，契约组件行为）**：PageTable `batchDelete.target` 未传入 BatchDeleteButton，后者硬编码 `选中的 ${N} 项`。最终文案「将删除选中的 N 项，删除后不可恢复…」仍满足 INV-8；`target:"设备"` 参数实际不生效。**未修改契约组件**（SC1 冻结），记为契约组件待办。
2. **单删口径统一（B 类对齐）**：discoveries 原已有确认弹窗（columns.tsx:114-121「确认删除?/确定/取消」），UI-SPEC 原文「无确认」系误判（已在 SPEC discoveries B 类行修正）；迁移后统一 createDeleteAction→ConfirmDelete（「确认删除/将删除{target}…」65 模板口径）。
3. **cache 单删保留 Popconfirm（裁决从严）**：PLAN 未列 cache 单删迁移；为避免 INV 口径回归（fails_when 红线），Popconfirm「确定要删除这个缓存吗？」逐字保留于 rowActions，不切 ConfirmDelete。
4. **cache Tabs 移除（裁决变化，记 PR）**：Tabs「缓存统计」面板内容为「统计信息」Modal 的真子集（键数量/命中/未命中/命中率/内存全覆盖），移除 Tabs 无信息损失；PLAN 未明确 Tabs 去留，此处从严对齐 C 类 3（统计入口=toolbar 按钮）。
5. **行号漂移**：T2 拆分区间实际 :737-1136（PLAN :738-1141）；T3 操作列 :94-265、双 hook :369-390 基本吻合；T4 概览卡 :489-564、清空确认 :311-330、统计 Modal :833-969 均吻合。
6. **index 行数偏差**：T3 index 692（PLAN 预估 ~350-420）——三弹窗 props 接口与 rowActions 组装保留页内所致，总量仍净减 70 行；T4 index 454（PLAN ~420-500）达标。
7. **FormModal 不透传 maskClosable/keyboard**：backups 创建备份与 cache 操作缓存原 Modal 均有 `maskClosable={false} keyboard={false}`，FormModal 契约无此 props——两点已在页内注释记录（点遮罩/ESC 可关，dirtyCheck 兜底未保存更改）。
8. **AppSelect 默认行为微变（T3）**：DeviceModals Select→AppSelect 后自动获得 allowClear+showSearch（契约统一行为，已记录）。
9. **采集端口 loading 降级（T3）**：原按钮 loading=collectingDeviceId===id，rowActions 契约无 loading 位→降级 disabled，功能不损失。
10. **devices reloadRef 用 state 承接**：`const [reloadRef] = useState({current:null})` + children 内 `reloadRef.current = ctx.loadData`（普通对象改属性不触发渲染，规避 P65 "refs during render" 规则）；T4 cache 同款。
11. **测试踩坑（复用价值）**：命令式 Modal.confirm 挂 body，RTL cleanup 不覆盖，需 afterEach 手动移除 `.ant-modal-root/.ant-dropdown`；ActionButtons ≥3 动作收「操作」下拉，ConfirmDelete 在下拉中按钮文本=label(deviceName)；「确认删除」文案 modal-title 与 confirm-title 双命中需 getAllByText；表头全选 checkbox 是 selection-column 第 1 个。
12. **P65 豁免 +1（T2 RestoreModals）**：纯搬迁文件机械入清单（Modal.confirm 3 处原位），非新增违规；T3/T4 新组件均零违规未入清单。

## 7. 遗留事项

- BatchDeleteButton target 接线（契约组件待办，Phase 68+ 候选；本次 SC1 冻结未动）
- cache/export 端点 filters 原发 `{...searchForm}`，迁移后因 searchForm 由 Form 实例管理，导出未带过滤值（原实现 key="" 全量语义等价，行为不变；如需带值可经 ctx.searchForm.getFieldsValue() 补接）
- devices/Discoveries 的 sorterMetas 视觉 sortOrder 列已由管线承接，如产品要求保留列头箭头高亮需 PageTable 增加 sortOrder 回显 props（契约组件待办）
