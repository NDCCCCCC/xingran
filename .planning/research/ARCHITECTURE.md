# Architecture Research: v1.23「前端契约全量收尾与遗留清偿」

**Domain:** Frontend contract-component completion (TextArea / COMP-05 / eslint 收口) + Go backend debug-fix integration
**Researched:** 2026-10-27
**Confidence:** HIGH（前端部分全部回源核验；后端部分基于 debug 文档 + 源码定位，其中 4 项修复方案依赖 DB/远程实测，已显式标注）

---

## Executive Summary

v1.23 是 v1.22 冻结契约的**清偿型里程碑**：不新建架构，只做「补最后一个契约组件（TextArea）→ 全量迁移（COMP-05）→ 拆除豁免脚手架（eslint 收口）→ 浏览器 UAT」的收尾链，外加 7 个互相独立的 Go 后端 debug 修复。

四个核心结论（对应问题 a–d）：

| # | 问题 | 结论 |
|---|------|------|
| a | AppInput 加 `multiline` prop vs 独立 AppTextArea.tsx | **独立 AppTextArea.tsx**。现有 eslint selector 只拦普通 JSX 名 `Input`，成员表达式 `Input.TextArea` 被有意豁免（`eslint.config.js:114` 注释），`multiline` 方案下 74 处存量 TextArea 无法被 selector 命中，仍需另加成员表达式 selector——等于两头都要改；且 antd 自身 API 就是 Input / Input.TextArea 两个组件，props 面（rows/autoSize/showCount）与 AppInput 的 allowClear 契约不重合。独立组件与既有契约模式（一组件一文件一默认值契约）完全同构 |
| b | 58 文件迁移批次顺序 | **组件先行（AppTextArea + SearchForm 扩展）→ 按域分批、per-page = per-PR（v1.22 先例）→ 每批用 gen-p65/p66-exemptions.mjs 收缩豁免清单**。域顺序：低风险验证域 → system → operations → network → duty/ad-domain → workorder/vdi/asset。v1.22 冻结契约（分页口径/ConfirmDelete/handleApiError/statusHelpers）**不在迁移触碰面上**——迁移是纯 JSX 换名，风险点是契约默认值生效（allowClear/showSearch）而非契约破坏 |
| c | 7 个 debug 修复 | 全部**相互隔离**（详见 §3 表格），无一需要 API 契约破坏性变更；至多 1 个可选 migration（R2 纵深防御 DB trigger / sys_job 防重唯一索引）。**其中 scrapligo GetPrompt panic 已在 2026-10-16 修复完毕**（scrapli_wrapper.go 四处修改 + v1.4.2），v1.23 仅剩人工验证归档。4 项修复根因未闭环，依赖 DB 端/远程服务器实测，不能直接进 plan 的 task 层 |
| d | eslint 规则翻转 | selector 本身已是 `error` 级；"翻转"的实质是**收缩/拆除三层豁免 override 块**（P66 存量 / P65 存量 / 封装层），配合 gen 脚本逐批收缩；lint warnings 降基数后才把 `no-unused-vars`/`no-explicit-any` 从 warn→error。flat config 的块覆盖语义是最大陷阱（见 §5 反模式） |

---

## 1. 现状架构基线（回源核验）

### 1.1 契约组件层（v1.22 Phases 65-67 产出）

```
src/components/ui/          AppInput.tsx (16行: allowClear=true 默认, 全 props 透传)
                            AppSelect.tsx (allowClear/showSearch 默认)
                            RemoteSelect.tsx (防抖远程搜索)
                            index.ts (统一出口 + P67 降险别名 AppInput as Input)
src/components/shared/      PageTable / SearchForm / FormModal / ConfirmDelete /
                            confirmDelete.ts / BatchDeleteButton / statusHelpers …
```

关键事实（全部回源确认）：

- `AppInput.tsx:12` — `export function AppInput({ allowClear = true, ...rest }: AppInputProps)`，`AppInputProps = ComponentProps<typeof Input>`。**同构模板**：AppTextArea 照抄此模式即可（默认值换成 allowClear=true + autoSize 可选）。
- `ui/index.ts` — P67 迁移降险策略已锁定为「**只改 import 行**」：`export { AppInput as Input }` 别名让存量页把 `import { Input } from "antd"` 换成 `import { Input } from "@/components/ui"` 即可零 JSX 改动。**lint 规则认 import 路径不认局部名**（该文件注释原话），别名不豁免拦截。
- `SearchForm.tsx:16-17,65-71` — 契约：渲染端点只允许 AppInput / AppSelect / RemoteSelect / `custom.render`，**field type 枚举目前只有 input/select/remoteSelect/custom**。TextArea 进搜索区需要给 SearchForm 加一个 `textarea` field type（渲染 `<AppTextArea rows={2}>`），这是 TextArea 集成 FormModal/PageTable 搜索区的唯一接线点。
- `FormModal.tsx` — children 是 `Form.Item` 列表，对字段组件无感知（页面自己写 `<AppTextArea />` 进 Form.Item），**FormModal 无需任何修改**即可承载 TextArea。脏检查（isFieldsTouched）对 TextArea 字段天然生效。

### 1.2 eslint 契约块（eslint.config.js，544 行）

| 块 | 位置 | 内容 | 与 v1.23 的关系 |
|---|---|---|---|
| Phase 65 契约块 | `:115-164` | `no-restricted-syntax`: 4 selector（`JSXOpeningElement[name.name='Input']` / `['Select']` / `Modal.confirm` 调用 / `Popconfirm`）+ hex selector（P68 追加）；`no-restricted-imports`: antd/Input、antd/Select 子包 | TextArea 新 selector 的落点（往同一数组追加，勿另起块） |
| P68 hex 豁免块 | `:171-231` | 结构性目录 + 存量 hex 文件，重声明完整数组（豁免 hex，保留 65 契约） | 新增 TextArea selector 时**此块必须同步** |
| 封装层豁免块 | `:239-251` | `src/components/ui/**` + `src/components/shared/**` 重声明为仅内网 IP selector —— AppTextArea 自身写裸 `<Input.TextArea>` 的合法点 | AppTextArea 落在 ui/ 目录，天然覆盖 |
| Phase 66 契约块 | `:259-285` | 拦截 `@/hooks/useTableManager` / `usePagination` 直 import（新列表页唯一入口 = PageTable） | COMP-05 迁移时勿动 |
| P66 存量豁免块 | `:313-368` | `__P66_EXEMPTION_FILES__` 锚点，36 文件 | COMP-05 收敛后由脚本清空、整块删除 |
| P65 存量豁免块 | `:373-509` | `__P65_EXEMPTION_FILES__` 锚点，124 文件；重声明为仅内网 IP selector（65 契约全豁免） | 同上 |
| 测试文件豁免块 | `:515-535` | `no-restricted-syntax: off` + imports 重声明 | 不动 |

两条**成文约定**（写在配置注释里，v1.23 必须遵守）：

1. `:108-113` — 65 契约块是独立 config 块，「勿与上方主块合并——flat config 同一 rules 对象内同名键互相覆盖」；68 追加 hex selector 时也明确「追加进本数组，勿另起 config 块——会使上方 4 个 selector 与内网 IP 拦截静默失效」。
2. `:114` — 「Input.TextArea|Search|Password|OTP 为成员表达式 JSX 名，不在本 selector 命中面（**有意不拦**）」——这是 v1.23 要翻转的事实：TextArea 收口意味着新增成员表达式 selector。

### 1.3 用量实测（2026-10-27，`src/pages` 全量 grep）

| 项 | 实测 | 备注 |
|---|---|---|
| 裸 `<Input` JSX | 220 处 / 71 文件 | 里程碑口径 ~157 处 / ~58 文件（约=非豁免文件子集），实测含豁免内文件 |
| 裸 `<Select` JSX | 271 处 / 86 文件 | 同上 |
| TextArea 用法 | 61 处 / 47 文件（43 处 `<TextArea` + 18 处 `Input.TextArea`） | 其中 32 文件用 `const { TextArea } = Input` 解构形态 |
| `AppInput` 已被 pages 引用 | 仅 2 文件 | COMP-05 迁移面巨大，别名降险策略必要 |
| lint 基线 | 83 errors（锁零新增）/ 1367 warnings | v1.23-seeds §C |

域分布（Input 命中文件数/域）：system ~15、operations ~14、network ~9、duty ~8、ad-domain ~6、workorder ~3、vdi ~2、asset ~4、monitor ~4、其余零散。Select 分布类似但 duty/management 组件目录更重。

### 1.4 后端集成架构（Handler-Service + Core DI，无新增层）

7 个 debug 修复全部落在既有层内：`internal/device/`（scrapligo 封装）、`internal/services/`（mac_collection / mac_history_query / operations/workstation_device）、`internal/scheduler/`（cron 引擎）、`pkg/logger` + `pkg/middleware/logger.go`（日志）、`internal/core/db/migrations/`（可选 migration）。**无新端点、无新模块、无 operlog 新增**（全部是缺陷修复/只读路径）。

---

## 2. (a) TextArea 契约设计：独立 AppTextArea.tsx（裁定 + 依据）

### 裁定

**新建 `src/components/ui/AppTextArea.tsx`，从 `ui/index.ts` 具名导出；不采用 AppInput `multiline` prop。**

### 依据（按权重）

1. **eslint selector 面不匹配**。现有 4 selector 全部匹配普通 JSX 名（`name.name='Input'`），成员表达式 `Input.TextArea` 有意不拦（§1.2 事实 2）。若选 `multiline` prop 方案：
   - 74 处存量 `<Input.TextArea>` / `<TextArea>`（解构自 Input）依旧命中不了任何 selector，要收口**仍然必须新增** `JSXOpeningElement[name.object.name='Input'][name.property.name='TextArea']` 这类成员表达式 selector —— 等于两种方案都要动 selector，`multiline` 的「改 import 行即可」优势不复存在；
   - 迁移时 `multiline` 方案要把 `<Input.TextArea>` 拆成 `<AppInput multiline>`，JSX 改动量比 `<AppTextArea>` 换名更大，且破坏了「AppInput = Input 1:1 透传」的既有心智模型。
2. **契约同构性**。v1.22 契约模式是「一 antd 组件 → 一封装文件 → 一组默认值」（AppInput:allowClear / AppSelect:allowClear+showSearch）。antd 自身 API 中 `Input` 与 `Input.TextArea` 就是两个组件、props 面不同（TextArea 有 rows/autoSize/showCount/maxLength 语义差异；allowClear 在 TextArea 上需 antd ≥5.13 行为也与单行不同）。硬塞进 AppInput 需要 prop 判别或条件透传，违背「封装层是 16 行薄透传」的现有形态（`AppInput.tsx` 全文 16 行）。
3. **集成点极小**：
   - FormModal：零修改（§1.1 事实 4）；
   - PageTable 搜索区：SearchForm 加 `textarea` field type（一处 switch 分支，渲染 `<AppTextArea rows={2} onPressEnter={onSearch}>`）；注意 74 处 TextArea 绝大多数在 FormModal 表单内，搜索区 TextArea 属罕见场景，field type 是补全枚举而非高频路径；
   - `ui/index.ts`：加 `export { AppTextArea }` + `export type { AppTextAreaProps }`；**不要**加 `TextArea` 降险别名 —— `const { TextArea } = Input` 解构形态无法用 import 别名降险，迁移必须改 JSX 行，但该形态总量 32 文件可控。
4. **验收基线**：debug 文档 `mac-collection-cisco-no-vendor-support.md` 等的「Files to Change」模式 → v1.23 的 TextArea phase 应产出等价的精确文件清单：`AppTextArea.tsx`(NEW) / `ui/index.ts`(MOD) / `SearchForm.tsx`(MOD: 新 field type) / `eslint.config.js`(MOD: 追加 selector × 2 处数组：65 契约块 + P68 hex 豁免块) / `gen-p65-exemptions.mjs` 命中判定 message 前缀（若 selector message 归入 no-restricted-syntax 同族则无需改脚本，回源确认 `:92-99` 按 message 前缀 + ruleId 判定）。

### AppTextArea 契约草案（供 SPEC 阶段定稿）

```tsx
/** 契约封装：默认 allowClear（antd TextArea 自 5.13 支持），autoSize 默认 { minRows: 2, maxRows: 6 }，其余透传 */
export function AppTextArea({ allowClear = true, autoSize = { minRows: 2, maxRows: 6 }, ...rest }: AppTextAreaProps)
// AppTextAreaProps = Omit<ComponentProps<typeof Input.TextArea>, 'allowClear'|'autoSize'> 或直接 ComponentProps
```

---

## 3. (c) 7 个 debug 修复：集成点 / 隔离性 / 依赖矩阵

> 状态列来自各 debug 文档 frontmatter + `v1.23-seeds.md` §A（2026-10-26 v1.22 收档审计 acknowledge）。**前 4 项中有 4 项根因未闭环**，plan 阶段应把「DB 端/远程实测」作为 task 前置而非直接排修复。

| # | 修复项 | 状态 | 集成点（具体文件） | Migration | API 契约变更 | 隔离性 |
|---|---|---|---|---|---|---|
| 1 | Linux 日志缺失（6-24 起 logs/ 无新日志） | `diagnosing`（需用户在 Linux 服务器跑诊断命令） | `pkg/logger/logger.go`（createFileWriter/lumberjack 配置、ConsoleOutput 硬编码 true 的双路问题）、`cmd/main.go`（initializeLogger）、`pkg/middleware/logger.go`（HTTP 请求日志走 journald 不走 app.log 的架构事实） | 无 | 无 | 完全隔离（日志核）；修复方向大概率是「file+stdout 双写策略」或「部署文档/诊断结论」，**先诊断后修** |
| 2 | Cisco 设备 MAC 采集缺厂商支持 | `diagnosed`（根因高置信，未实施） | `internal/models/network_device.go`（+`VendorCisco`）、`internal/services/mac_collection_service.go`（`getMACCommands` → Cisco 双命令 `show mac address-table` + `show port-security address`；`parseMACLine` + Cisco case，点分格式 `xxxx.xxxx.xxxx` 与 7 字段 port-security 行）、`internal/services/portcollection/trunk_filter.go`（Cisco hint tokens）、`internal/api/v1/network/network_export_handler.go:89,205` + `batch_export_handler.go:167,226`（vendor 白名单）、`templates/`（可选 TextFSM）、前端 `pages/network/device/` vendor 下拉 | 无（vendor 是 varchar，不加列） | **加法扩展**：vendor 枚举 +1（前后端白名单同步，向后兼容） | 隔离；但与 #3 同文件（mac_collection_service.go），**必须同域排序**（见 §4） |
| 3 | MAC 格式脏数据复发（R2） | `investigated`（假设 A/B/C 三个未决：GORM 切片插入绕 hook / 并发 / 构建不一致） | `internal/services/mac_collection_service.go:291`（`Create(&macRecords)` 切片写入路径，需运行时验证 hook 是否绕过）、`internal/models/device_mac_address.go:42`（BeforeCreate normalize hook）、`pkg/normalize/mac.go` / `iface.go`；**可选纵深防御**：`internal/core/db/migrations/` 新增 DB trigger migration（REGEXP_REPLACE 归一化，R2 文档 §6.2 建议，原 M196 提案未实施） | 可选（trigger 路线则 1 个 migration；纯 Go 路线无） | 无 | 隔离；与 #2 同文件同域，建议同一 phase 波次 |
| 4 | mac_history 物化视图任务每轮双执行 | `investigating`（根因未定位） | `internal/scheduler/cron.go`（`Start():224-238` 从 DB 全量加载 + `AddJob():279-297` 先 Create DB 再 addJob 的注册链、`RegisterTask():202` 注册表）、`internal/core/core.go:545`（唯一注册点 `RegisterMACHistoryMatViewTasks`）、`internal/scheduler/mac_history_matview_tasks.go`（`:29-39` 按 `job_name` First 查重的入库守卫——首启竞态或 sys_job 双行嫌疑）；`sys_job_log` 双行同时戳为判定证据 | 可选（若定位为 sys_job 同名双行 → 唯一索引 migration） | 无 | 隔离；**但 cron.go 是全调度器共享引擎**，改动影响所有 job，回归面 = `internal/scheduler/` 全部测试（现有 cron_test.go / mac_tasks / ad_sync / vdi / workorder 各 *_80_02_test.go） |
| 5 | MAC 轨迹查询返回空 | `investigating`（代码层 4 假设全排除，待 DB 实测 a-d 四场景） | `internal/services/mac_history_query_service.go:842-915`（`QueryMACTrajectory` Raw SQL；`:849-852` normalize；无 deleted_at 过滤——与 `migration_175_reconciliation_physical_link.go:69` 注释自相矛盾处）、`internal/models/device_mac_history.go`（确认无 DeletedAt）、`internal/api/v1/network/mac_history_router.go:11-32`（POST /history/trajectory 入口） | 无 | 行为级：若按 (c) 场景修——Raw 改 GORM chainable + 错误传播（空 → 500），属**错误可见性修复**非契约变更 | 隔离；前置 = DB 四项实测（debug 文档 §verification SQL 全给出） |
| 6 | scrapligo GetPrompt nil panic | `awaiting_human_verify`（**已修复** 2026-10-16） | 已改：`internal/device/scrapli_wrapper.go`（OpenContext init goroutine 全程持 opMu.RLock / ctx.Done 走 w.Close() / closeStarted CAS 幂等 / closeOnce 放行 WaitForReady）、`internal/device/scrapli_wrapper_initrace_test.go`（3 回归测试）、`go.mod` v1.4.0→v1.4.2 | 无（go.mod 升级已完成） | 无 | **v1.23 无代码工作**：人工验证「不再出现进程级 panic」后 archive + knowledge base。唯一不占开发工时的项 |
| 7 | 工位 3F130 物理链路为空 | `investigated`（静态分析穷尽，5F003 同源已排除，剩 H1-H4 数据侧假设） | `internal/services/operations/workstation_device_service.go`（`:320` GetPhysicalDevices、`:339` user_id 空早退、`:391-404` CTE 三道过滤）、`internal/api/v1/operations/workstation_device_handler.go:152`；验证 SQL 模板全套在 debug 文档 §4 | 无（除非落「数据修复 migration」，但倾向单点数据修正非 migration） | 无（若加「空原因诊断字段」则为加法扩展，SPEC 阶段裁决） | 隔离；前置 = 跑 §4.5 综合诊断 SQL；**代码修复形态大概率是可观测性增强**（区分「未绑定 user_id / 绑定错位 / 设备 orphan」三种空态），不是逻辑重写 |

### 依赖与排序约束

- **#2 → #3**：同一文件（mac_collection_service.go），Cisco 双命令解析会改变写入行数与格式覆盖面；先落 #2 会让 #3 的复发验证更复杂。建议同一 phase 内先做 #3 的归一化防线（含可选 trigger），再做 #2 厂商扩展；或至少串行不同 PR。
- **#2/#3 → #7**：#7 的验证依赖 `sys_device_mac_address` 数据可信（H5 已被用户排除，但 R2 显示 525 行脏格式），格式防线落地后 3F130 复测才有判读力。**软依赖，可并行开发只影响验证时点。**
- **#1 / #4 / #5 / #6 / #7** 之间零依赖，可全并行分派。
- 7 项全部不需要动 `internal/api/router.go`、权限（`pkg/permission/`）、operlog（只读修复无写端点新增）。

---

## 4. (b) COMP-05 迁移批次架构

### 4.1 批次 = 页面 = PR（v1.22 先例不变）

v1.22 先例「one page = one PR」是 reviewable unit 的正确粒度，COMP-05 沿用。但 58+ 文件按页零散迁移会让 PR 数失控，采用**两层结构**：域内按页拆 commit，域 = 一个 PR 串（或按里程碑 phase 划分，每 phase 收敛 1-2 个域）。

### 4.2 依赖有序的构建顺序

```
W0  基建（1 phase 内完成，先于一切迁移页）
    ├─ AppTextArea.tsx + ui/index.ts 导出 + ui/__tests__ 契约测试
    ├─ SearchForm 新增 textarea field type
    ├─ eslint 追加 TextArea selector（65 契约块 + P68 hex 豁免块两处数组同步）
    └─ 跑 gen-p65-exemptions.mjs（--dry 先验）把新命中文件写进豁免锚点
        —— selector 上线瞬间不产生新 error，存量进豁免清单

W1  验证域（低流量单文件页，验证迁移模式）：monitor/*、profile、knowledge、settings、dashboard-system
W2  system 域（~15 文件，命中最多；全是核心 CRUD 页，逐页 commit）
W3  operations 域（~14 文件；68.1 §3 已迁 4 个列表页的形态基线在此，回归红线 §3.3 直接复用）
W4  network 域（~9 文件；注意 port-write 组件组有 v1.19/v1.20 冻结交互，纯换名不碰逻辑）
W5  duty + ad-domain（~14 文件；duty/management 组件目录 Select 密度最高）
W6  workorder + vdi + asset + 零散（~10 文件）
W7  收口：豁免锚点清空 → 删除 P65/P66 存量豁免块与 gen 脚本（或留 --dry 校验模式）
    → lint warnings 降基（§5）→ 浏览器 UAT（§6）
```

排序理由：

1. **W0 必须最先**：SearchForm 是 PageTable 搜索区的唯一渲染端点（§1.1），若迁移页先行而 SearchForm 不支持 textarea，搜索区字段只能塞 custom.render 逃逸口，产生第二路径——这正是 v1.22 要消灭的东西。
2. **域顺序按「验证→高频→组件密度」**：先用低流量域把「换名 + 契约默认值生效」的回归模式跑通（allowClear 出现在搜索框、showSearch 改变 Select 交互），再进 system/operations 高频页。同域内先迁**已同时出现在 P65 与 P66 豁免清单的文件**——一次迁移收缩两份清单，收敛效率最高（对照 §1.2 两个锚点清单求交集）。
3. **v1.22 冻结契约不在触碰面上**：COMP-05 是纯 JSX 换名 + import 换路径，不改任何 Service/handler/状态逻辑。真正要盯的回归点是三个契约默认值的**行为外溢**（见下表），每页 PR 的 checklist 固定为：`npm run lint`（error ≤ 83）+ `npm run type-check` + 页面手工冒烟（搜索框清空按钮 / 下拉可搜索 / TextArea 回车行为）。

| 契约默认值 | 行为外溢 | 迁移时检查 |
|---|---|---|
| AppInput allowClear=true | 所有单行输入框出现清空按钮 | 有 `onPressEnter` 的搜索框确认清空不误触搜索 |
| AppSelect showSearch=true | 下拉变可输入过滤 | options 数据源支持本地 filterOption（默认行为）；受控 filterOption 页面是否显式传了 false |
| AppTextArea autoSize 默认 | 弹窗表单高度变化 | FormModal 长表单滚动是否正常 |

### 4.3 lint warnings 清偿（任务 5 的规则落点）

规则全部在 `eslint.config.js` 单文件内：

- `:79-80` `@typescript-eslint/no-unused-vars` / `no-explicit-any` → warn（约 255 + 75 处存量，v1.23 主要清偿对象）；
- `:83` `no-console`（warn）、`:59` `react-refresh/only-export-components`（warn）；
- error 侧冻结项：`react-hooks/exhaustive-deps: error`（:102）、`local/no-large-dropdown-list: error`（:105）、83 errors 基线（68.1-FACTS.md:563）。

收紧策略：**先降基数、后翻级别**。每个迁移 PR 顺手清掉本页的 no-unused-vars/no-explicit-any（迁移本就要逐行过），域完成后统计该域 warning 增量应为负；全部域完成后若两项 warn 清到个位数，再在同一收口 PR 里 warn→error（一次翻转，避免「半新半旧 error」污染中间态基线）。冻结契约不受影响——四类冻结（分页口径 / ConfirmDelete 唯一入口 / 组件契约 / handleApiError + statusHelpers）的守护规则全部是 error 且不在本次翻转集合里。

---

## 5. (d) eslint 收口架构与反模式

### 5.1 「warning → error 翻转」的实质

裸组件 selector **已经是 error**（65 契约块），之所以存量页不报错，是因为 P65/P66 存量豁免 override 块把它们**整条重声明掉了**。因此收口 = 拆脚手架，不是改级别：

```
迁移完成态（W7）:
  P66 锚点 files 数组清空 → 整块删除（:313-368）
  P65 锚点 files 数组清空 → 整块删除（:373-509）
  gen-p65/p66-exemptions.mjs 退役（UI-SPEC §4.4 的幂等校验「连跑两次 diff 为空」自然恒真）
  剩余结构 = 主块(内网IP) + 65契约块 + hex豁免块(收缩后) + 封装层豁免 + 测试豁免 + prettier
```

封装层豁免块（`src/components/ui|shared`）**永久保留**——AppTextArea 内部写裸 `<Input.TextArea>` 的唯一合法点，与 AppInput 现状同构。

### 5.2 反模式（flat config 实战陷阱，全部有成文前科）

| 反模式 | 后果 | 正确做法（配置内注释原话依据） |
|---|---|---|
| 新 selector 另起 config 块 | flat config 后块覆盖前块同名规则键，65 契约 4 selector + 内网 IP 拦截**静默失效**（:140-144 注释明确记载） | 追加进 65 契约块同一数组；且 **P68 hex 豁免块（:171-231）重声明了完整数组，必须同步加** |
| 豁免块用 `"off"` 置空 | 连带关闭 antd 子包拦截等兄弟 selector（P66 存量豁免注释明令「禁止直接置 off」） | 重声明「仅保留需保留 selector」的完整数组 |
| 绕过 gen 脚本手改锚点 files 数组 | 与 `exemption-reasons.json` / hex-exemptions 登记制失配，§4.2「每条豁免可答三问」审计失效 | 每批迁移后跑 gen 脚本收缩清单，理由写在脚本不覆盖的独立清单 |
| 在 gen 脚本会写回的位置写豁免理由 | 被脚本覆盖丢失（UI-SPEC §4.4 明确） | 理由落在独立 md / 结构化数据文件 |

### 5.3 收口验收（可进 phase 的 SC）

1. `grep -rn "Modal.confirm" src` 仅剩 `confirmDelete.ts:34` + `FormModal.tsx:105` + 6 处带理由 disable（68.1 §8 #12 口径）；
2. `npm run lint` errors ≤ 83 且豁免锚点数组为空；
3. gen 脚本 --dry 连跑两次 diff 为空（§8 #18 幂等口径）；
4. 新 selector（TextArea）下 `grep -rE "<TextArea|Input\.TextArea" src/pages` = 0。

---

## 6. 浏览器 UAT（任务 4）：68.1 §8 第 15/16 条

**重要核对发现**：`v1.23-seeds.md` §B 把 §8 第 15/16 条描述为「分页交互、批量删除流程」，但回源 `68.1-UI-SPEC.md §8`（:315-316）实际条目是：

| # | 实际条目 | 冒烟口径 |
|---|---|---|
| 15 | **固定列不失效**：迁移页横向滚动时固定列不动 | 选一个有 `fixed` 列的迁移后列表页（operations 域 4 页），横向滚动观察 |
| 16 | **行操作权限不变**：无权限用户的删除按钮仍不可见 | 降权账号登录，核对行操作按钮可见性 |

（分页口径是 #13、批量删除对象名是 #5/#6——seeds 的概括与原文有偏移，**plan 阶段以 UI-SPEC 原文为准**，建议把 #13-#16 一并纳入回归清单，成本相同。）

流程沿用 Phase 63 冒烟：登录 → 目标页 → 翻页/勾选/取消；若 COMP-05 迁移了 68.1 §3 的 operations 4 页，W3 完成后立即做这两条，不要拖到 W7。

**附带核对发现**：UI-SPEC §5 记录 antd 实装版本为 **6.6.4**（非 CLAUDE.md 的 6.1）；`focusable.autoFocusButton` 已 @deprecated，TextArea 契约测试勿引用旧 API。

---

## 7. Recommended Project Structure（v1.23 增量，其余全复用）

```
ruoyi-react-frontend/src/components/ui/
├── AppInput.tsx            (不动)
├── AppSelect.tsx           (不动)
├── AppTextArea.tsx         (NEW — §2 契约草案)
├── RemoteSelect.tsx        (不动)
├── index.ts                (MOD +2 导出)
└── __tests__/AppTextArea.test.tsx  (NEW 契约测试, 对齐现有 __tests__ 形态)

internal/core/db/migrations/
└── migration_2xx_mac_format_trigger.go   (可选 — #3 纵深防御)
    migration_2xx_sys_job_name_unique.go  (可选 — #4 若确认双行根因)

internal/device/scrapli_wrapper.go        (#6 已修，仅验证归档)
```

## 8. Scaling / 风险优先级

1. **第一风险**：58+ 文件迁移中某个页面的 Select 有自定义 `optionFilterProp`/受控 search 逻辑，showSearch 默认值引发交互回归 → W1 验证域先行 + 每页冒烟 checklist 兜底。
2. **第二风险**：cron.go（#4）是全调度器共享引擎，改动波及所有 job → 该项单独 phase，回归面锁 `go test ./internal/scheduler/` 全套。
3. **第三风险**：4 项 debug 修复根因未闭环，直接排修复任务会产生空转 → plan 阶段把「DB/远程实测」设计为独立前置 task，实测结论回填 debug 文档后再进修复 task。

## 9. Sources（本次研究证据）

- `.planning/PROJECT.md`（v1.22 契约组件抽取决策、operlog/Handler-Service 约定）
- `.planning/v1.23-seeds.md`（§A 7 项 debug、§B 68.1 遗留、§C lint 基线、§D 冻结契约）
- `.planning/milestones/v1.22-phases/68.1-closeout-debt-clearance/68.1-UI-SPEC.md`（§1/§3.3/§4.4/§8 全文）
- `ruoyi-react-frontend/eslint.config.js`（544 行全文，含全部成文陷阱注释）
- `src/components/ui/AppInput.tsx`、`ui/index.ts`、`src/components/shared/FormModal.tsx`、`SearchForm.tsx`
- `.planning/debug/` 7 份文档全文（状态行号以各文档 frontmatter 为准）
- 源码定位：`internal/scheduler/cron.go:201-360`、`internal/core/core.go:545`、`internal/scheduler/mac_history_matview_tasks.go`、`internal/device/scrapli_wrapper.go:62,297-413`、`internal/services/mac_history_query_service.go:842-915`、`internal/services/operations/workstation_device_service.go:320-575`
- 实测统计：`src/pages` 裸 Input/Select/TextArea grep（2026-10-27，方法见 §1.3）

---
*Architecture research for: v1.23 前端契约全量收尾与遗留清偿*
*Researched: 2026-10-27*
