# Phase 66 SUMMARY — 高级契约组件：PageTable 与 FormModal

日期：2026-10-25 ｜ 执行：GSD executor（T1→T7 顺序 + T8 收口）

## 交付物

| 组件 | 路径 | commit |
|------|------|--------|
| PageTable（最小骨架→完整契约） | `xingran-react-frontend/src/components/shared/PageTable.tsx` | 3a352b05 → 5efc1293 |
| SearchForm（schema 驱动搜索区） | `xingran-react-frontend/src/components/shared/SearchForm.tsx` | c2482c48 |
| FormModal（编辑弹窗契约） | `xingran-react-frontend/src/components/shared/FormModal.tsx` | fe331626 |
| RemoteSelect（远程搜索下拉） | `xingran-react-frontend/src/components/ui/RemoteSelect.tsx` | 5d22cb33 |
| post 页试点迁移 | `xingran-react-frontend/src/pages/system/post/index.tsx`（480→268 行） | 614197a8 |
| lint 增量门 + 豁免脚本 | `xingran-react-frontend/eslint.config.js`、`scripts/gen-p66-exemptions.mjs` | d2776563 |

全部为纯新增组件文件（shared/ 3 + ui/ 1）；useTableManager/usePagination 零改动；BaseEditModal 对外 API 零变化。

## Success Criteria 逐条 verdict（ROADMAP Phase 66 SC1-4）

### SC1 — PageTable 内建分页/loading/空态/错误态 ✅ PASS
- 分页：showTotal 逐字承接 `usePagination.ts:122` 口径（`total===0` 空串；否则 `` `共 ${t} 条 / ${Math.ceil(t/pageSize)} 页` ``）——PageTable.test 断言 `共 128 条 / 13 页` 与 total=0 无"共 0 条"。
- loading：内建 Table loading（`.ant-spin` 存在断言；禁 Skeleton，PLAN 硬边界）。
- 空态：默认 `.ant-empty`，`emptyAction` 可替换为 EmptyStateWithAction。
- 错误态：PageTable 包装 request——成功清 error、失败记 error 后 rethrow（不改 useTableManager 内部），渲染 ErrorAlertWithRetry + 重新加载重试（测试覆盖失败→重试成功链路）。

### SC2 — 模板页代码量显著下降 ⚠️ 部分达成（如实记录）
- post 页 480 → 268 行（-44%）。列表样板（请求/分页/行选择/排序/工具栏/搜索区/操作列/批删）全部契约化，由 PageTable 62 行接入替代原 ~300 行手写样板。
- 268 行未到 "~120 量级"：剩余行数为真实业务逻辑（统计卡片 + 已选择 Alert 保留（PLAN T6 要求）、6 列定义、表单字段、业务处理函数）。契约组件覆盖的"样板"部分已达成 ~120 量级的精神；纯列表型页面（无统计卡片/无复杂表单）预计可落 ~120。
- SearchForm 并入 PageTable 体系 ✅（searchFields schema 4 型 + searchChildren 逃生口二选一）。

### SC3 — FormModal 三项默认生效 ✅ PASS
- confirmLoading 防重复提交：onOk 包装 async 提交 + cancelButtonDisabled 防重（测试断言提交中按钮禁用）。
- 脏检查拦误关：`dirtyCheck && isFieldsTouched && 非提交中` → Modal.confirm 文案逐字（更改尚未保存 / 当前修改尚未保存，关闭后将丢失，是否确认关闭？/ 放弃更改 / 继续编辑 / ok 非 danger）；未 touched 直关。
- destroyOnHidden 重置：经 BaseEditModal 硬编码继承 + `open=true` 时 resetFields+setFieldsValue(initialValues)（回填测试）。
- 提交失败分流：isFormValidationError 静默 → 其余 handleApiError(error,"保存")，弹窗保持打开（两类测试各 1）。
- FormModal 文件内 Modal.confirm 为唯一新增合法调用点（lint 契约块约束）。

### SC4 — 契约自描述（不读源码可接入）✅ PASS
- PageTable 文件头 JSDoc 完整用法示例 + Router 上下文注记 + 「唯一合法入口」声明；TS 类型全导出（PageTableProps/PageTableRequest/PageTableRenderContext/SearchFormProps/SearchField/FormModalProps）。
- `shared/index.ts`、`ui/index.ts` 桶导出齐备。

## 基线保护（63/64 已验收口径零破坏）

- 全量 vitest：**3900 passed / 0 failed**（564 文件；65 收口基线 3864 → 本期净增 36 测试，零回归）。
- type-check：**0 错误**（每任务定向验证）。
- 全量 lint：**83 errors / 1414 warnings = 65 基线，0 新增 error**。
- useTableManager 公开 API 零改动（25 存量页零破坏，全量回归佐证）。

## lint 契约强制（T7）

- `eslint.config.js` 新增「Phase 66 契约强制」块（65 块后、存量豁免前）：
  - no-restricted-imports patterns = 合并 65 既有 antd 子包 patterns + 新增 `@/hooks/useTableManager`、`@/hooks/usePagination` 两条（R1：同一数组，不另起）；message 指向 `@/components/shared` 的 PageTable。
  - 封装层 `src/components/shared/**`、`src/components/ui/**` 重声明仅保留 antd 子包 patterns（R2）。
  - 未使用任何 `off`（R3）；脚本生成清单前提满足（R4）。
- p66 存量豁免清单：**45 文件**（`__P66_EXEMPTION_FILES__` 锚点；重声明完整数组仅省 hook 两条），gen-p66-exemptions.mjs 复跑幂等一致。
- 探针验证：新业务文件 import hook → 1 error 指向 PageTable；shared 层裸 antd/Input → 1 error（65 拦截仍生效）。
- 测试文件 override 重声明 no-restricted-imports（仅 antd patterns）。

## 豁免清单对账（活文档）

- 65 清单（selector 类）：147 → **142** 文件（T6 复跑 gen-p65-exemptions.mjs，post 页移出）。
- 66 清单（hook imports）：新建 **45** 文件，P67 迁移后逐页收敛。
- 76 个存量业务页未动（P67 范围）。

## 偏离记录

1. **PageTable 增加 `children` render-prop**（PLAN T2 未列）：post 页需「已选择 Alert」（需 selectedRowKeys/clearSelection）与提交后刷新（需 loadData）。依据 UI-SPEC §1 决策 2「PageTable 内部实例化 useTableManager 并经 render-props 暴露」授权，新增 `PageTableRenderContext<T>{selectedRowKeys,clearSelection,total,loadData,searchForm}`。
2. **post 页 268 行 vs ~120 量级**：见 SC2，剩余为保留业务区，样板部分已契约化。
3. **测试文件 2 处 eslint-disable-next-line react-hooks/globals**：React 19 下 `Form.useForm()` 只能在组件内调用，react-compiler 规则禁止 render 内写外部变量；用 let 冒泡 + 精确 ruleId 逐行豁免（RemoteSelect.test.tsx / post 页 useEffect 1 处 set-state-in-effect），非 config off。

## 验证数字汇总

| 项 | 结果 |
|----|------|
| type-check（每任务） | 0 错误 |
| 全量 vitest | 3900 passed / 0 failed |
| 全量 lint | 83 errors（=基线，0 新增）/ 1414 warnings |
| 定向 vitest | PageTable 12、SearchForm 6、FormModal 7、RemoteSelect 6、post 3 文件 6 测 |
| 65 豁免清单 | 147 → 142 文件 |
| 66 豁免清单 | 45 文件（幂等一致） |

## 遗留上报（P67+）

1. SC2 纯列表页 ~120 量级待 P67 discoveries 试点实测复核（post 页含保留业务区不具代表性）。
2. 45 文件 66 hook 豁免清单随 P67 迁移逐页收缩（脚本 gen-p66-exemptions.mjs 已就绪）。
3. RemoteSelect 无跨实例缓存（按契约）；若 P67 大规模接入出现重复 fetch 压力，再评估缓存层（当前刻意不做）。
4. Console warning 提示 npm config `manage-package-manager-versions` 未知项（环境噪音，与本期无关）。
