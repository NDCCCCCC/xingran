# Phase 65「前端契约组件与 lint 拦截」执行 SUMMARY

> 执行者：gsd-executor；权威任务书 `.planning/phases/65-contracts/65-PLAN.md`（T1-T8）；
> 权威契约 `.planning/phases/65-contracts/65-UI-SPEC.md`（§1 AppInput / §2 AppSelect /
> §2.3 RemoteSelect 仅契约 / §3 lint / §4 ConfirmDelete / §5 迁移清单）。
> 验证基线：64-P1-SUMMARY（type-check 0 / vitest 3843+0 / lint 存量 errors 判闸只数新增）。

## 逐任务完成状态

| 任务 | 内容 | Commit | 关键证据 |
|------|------|--------|----------|
| T1 | 契约组件 AppInput/AppSelect + ui/index.ts（具名+`as Input/Select` 别名）+ 单测；post 页搜索区试点（2 Input→AppInput、1 Select→AppSelect） | `be9a882c` | 定向 vitest 16/16；全量 3855 passed（3843 基线 + 12 新增） |
| T2 | eslint no-restricted-syntax 4 selector + no-restricted-imports 子包拦截 + 三层面豁免 + `scripts/gen-p65-exemptions.mjs` 清单脚本 | `c4b63a97` | 业务目录探针 4 报错 / 移入封装层 0；侧门 import 探针 1 报错；lint errors 规则前后均 83（新增 0） |
| T3 | `confirmDelete()` helper（唯一 Modal.confirm 调用点，文案/okText/danger/onOk rethrow 逐字按 §4） | `0b0a9f98` | 单测 5 项（文案逐字 / title 覆盖 / 成功 / 失败 rethrow / extra 透传） |
| T4 | `ConfirmDelete` 声明式组件（复用 helper，danger link + DeleteOutlined + `action-btn-link-danger`） | `e5d58b39` | 单测 4 项；与 T3 合并为 `confirmDelete.test.tsx` 9/9（见偏离 3） |
| T5 | buttonStyles 常量升级 + BatchDeleteButton/createDeleteAction 内部切换（Popconfirm→confirmDelete/ConfirmDelete，props 零变化） | `077b5614` | 两文件 grep Popconfirm=0；batchOperations 4 测（含文案逐字断言）+ ActionButtons.render 测试更新 |
| T6 | 整合校验 + 豁免脚本自清空悖论修复（临时 config 禁用豁免扫描） | `c188cb73` | shared __tests__ 258/258；脚本复跑命中 147 且幂等（diff 仅锚点行 +1） |
| T7 | 63 已收敛 11 处批删 handler 迁移到 confirmDelete（7 system + 4 network），豁免清单 147→143 | `64997cd5` | 净 -108 行；type-check 0；全量 vitest 3864 passed / 0 failed（首轮 2 flake 复跑消失） |
| T8 | 本 SUMMARY（不 commit） | — | — |

## 偏离记录（计划/契约 vs 实现）

1. **lint 存量基线漂移**：64 SUMMARY 记 125 errors 存量，T2 实测规则上线前后全量 lint 均
   **83 errors / 1403 warnings**。判闸口径不变（只数新增，= 0），但绝对基线已漂移，P67 收敛
   时应以 83 为参照。
2. **T1/T5 视觉降级**：createDeleteAction 由 Popconfirm ghost 按钮降级为 ConfirmDelete link
   danger（§4 契约形态）；BatchDeleteButton 保留 danger ghost 按钮本体（仅确认弹层换 Modal）。
   ActionButton 接口与全部消费者零变化。
3. **Windows 大小写不敏感**：`ConfirmDelete.tsx` 与 `confirmDelete.ts` 同目录共存时，
   `import from "./ConfirmDelete"` 会被解析到 `.ts` 文件（组件 undefined）。所有组件 import
   均写显式扩展名 `./ConfirmDelete.tsx`；两个同名测试文件无法共存，合并为单文件
   `confirmDelete.test.tsx`（两个 describe：命令式 5 测 + 声明式 4 测）。
4. **T7 等价替换中的统一化收口**（均为契约内行为的收敛，非功能变化）：
   - 错误 context 由各处 `handleApiError(error, "批量删除")` 统一为 helper 内建 `"删除"`
     （§4 契约：失败提示 `删除失败: {原因}`，不逐处手写）。
   - network devices 批删原有的 `confirmInstance.update` 取消按钮防抖为 helper 契约外增强，
     随迁移放弃（antd onOk 返回 Promise 时确认按钮自动 loading，风险极低）。
   - credentials/ports/mac 三处原 `withErrorHandling` 包装由 helper 内建成功/失败提示替代，
     成功提示文案不变（`已删除`）。
   - helper onOk rethrow 使失败时 Modal 保持打开可重试（§4 契约行为，原实现失败即关闭）。
5. **T7 清单收缩 147→143 而非 147→138**：11 处批删分布于 10 个文件，其中 4 个纯 hook
   `.ts` 文件（dict×2/role/notice/templates 所在 hook）完全清零移出清单；6 个页面文件因
   同时存在裸 Input/Select/Popconfirm 存量命中（归 P67）保留在清单。全部按 selector 实测。

## Verification 结果（收口总闸）

| 检查 | 命令 | 结果 |
|------|------|------|
| 类型 | `cd ruoyi-react-frontend && npm run type-check` | **0 错误**（每任务后均验） |
| 单测 | `npm run test -- --run` | **3864 passed / 0 failed**（基线 3843 + 21 新增；首轮 2 failed 复跑消失判 flake） |
| lint | 定向探针（禁全量口径调整） | 新增 errors = **0**；业务目录 4 selector 全报 / 封装层 0 / 侧门 import 拦截生效 |
| 豁免清单 | `node scripts/gen-p65-exemptions.mjs` | 命中 147（T2）→ **143**（T7 收缩后），幂等可复跑 |

## SC1-4 verdict（对照 UI-SPEC）

| SC | 内容 | verdict |
|----|------|---------|
| SC1 | AppInput/AppSelect 契约组件可用、默认 allowClear、rest 透传、别名导出 | ✅ PASS（T1，单测 16 项含 props 覆盖/显式传参覆盖默认值） |
| SC2 | 裸组件/裸 Modal.confirm/裸 Popconfirm 被 lint 拦截或有明确豁免清单 | ✅ PASS（T2/T6：4 selector + 侧门 import 拦截均探针验证；存量 143 文件清单 + `// P67 收敛后移除` 注释） |
| SC3 | 危险删除统一入口 ConfirmDelete/confirmDelete，无绕过 | ✅ PASS（T3-T5：唯一 Modal.confirm 调用点在 helper 内；lint 拦截裸调用；T7 迁移 11 处批删；BatchDeleteButton/createDeleteAction 内部已切换，新代码经 lint 强制） |
| SC4 | 删除确认文案模板（`确认删除`/`将删除{target}，删除后不可恢复，是否继续？`/`删除`/`取消`/`已删除`）逐字落地 | ✅ PASS（T3 单测逐字断言；batchOperations 测试含 `将删除选中的 2 项…` 逐字断言；buttonStyles 常量同步升级） |

## 上报裁决项（范围外发现，未擅改）

1. **业务页 9 处字面量旧文案**（不 import 常量、不受常量升级影响）：workorder×3 / knowledge×1 /
   duty×4 / system/menu×1 仍写死 `确定要删除吗？`——归 P67 存量收敛（含 system/menu 域，与
   PLAN「operations/assets 与 system/menu 严禁触碰」边界一致，本 phase 未动）。
2. **lint 存量基线漂移**（偏离 1）：83 errors 与 64 SUMMARY 的 125 差 42，建议 P67 立项前
   重测全量基线并归档成因。
3. **工作区非本任务残留**：`internal/api/tmp_userrole_schema_test.go`（未跟踪临时文件，
   会干扰 go build/test）与 `.planning/{STATE.md,milestone.lock,state.json}` 本地改动——
   非 Phase 65 产物，未触碰，请 lead 裁决处置。
4. **RemoteSelect 仅契约**（§2.3），实现归 P66；useDebouncedSearch 未单独抽（现网 0 命中）。

## 手工冒烟建议（留给 lead / QA）

- post 页搜索区：AppInput 一键清空、AppSelect 可搜索 + 一键清空。
- 任一批删页（如 system/post）：批量删除→取消→Modal 关闭不删；确认→成功 toast `已删除`；
  断网场景→toast `删除失败: …` 且 Modal 保持打开可重试。
- 行删除（ActionButtons 场景）：ConfirmDelete link danger 按钮弹 Modal（原 Popconfirm 气泡
  不再出现，视觉降级已记录）。

## Verifier 交接说明

- 判闸口径：type-check 0 / vitest 0 新增失败 / lint 新增 errors 0（全量 lint 禁用口径调整，
  一律定向命令）。dashboard 存在已知 flake，失败用例单独复跑判定。
- 豁免清单维护：T7 后收缩动作已做；P67 每迁移一域复跑
  `node scripts/gen-p65-exemptions.mjs`，脚本已内置自清空悖论防护（临时 config 扫描），
  锚点行 `__P65_EXEMPTION_FILES__` 不可删。
- 封装层 override 的内网 IP selector 字面量与主块须双向同步（搜索「禁止硬编码内网 IP」定位两处）。
