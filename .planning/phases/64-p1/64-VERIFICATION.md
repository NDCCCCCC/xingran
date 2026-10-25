# Phase 64 验证报告 — P1 修复：反馈与可访问性（FIX-01..05）

> 验证：gsd-verifier ｜ 日期：2026-10-25 ｜ 方式：只读取证（git show / grep / type-check / 全量 vitest 亲跑）
> 对象：ab59a977(T1) 2d566a92(T2) efc72abe(T3) 7b6999ff(T4) 20b425a5(T5) 2e7aa388(T6) eb41070a(T7) ac02c0f5(T8) 2a296934(T9)，9 commit 均存在
> **裁决：ISSUES** —— 2 blocking（SC3 排序链路断裂、SC5 aria 大面积无效），4 non-blocking

---

## 一、逐 SC 验证证据

### SC1（FIX-01）内嵌表 loading — ✅ PASS
- 复扫亲跑：`grep -rL "loading=" $(grep -rl "<Table" src/pages/network src/pages/monitor src/components/network --include=*.tsx)`
- 命中 3 文件，与 SUMMARY 偏离记录 3 完全一致：`VariablesModal.tsx`（同步数据）、`BulkWriteDrawer.tsx`（batchResult 同步）、`jobLogColumns.test.tsx`（测试文件）。均为合理豁免，未发现伪造 `loading={false}`。
- 回源抽查：`src/pages/monitor/job/index.tsx:262` `<Table loading={loading}>` 接 `useJobData` 真实 loading state ✓。

### SC2（FIX-02）遮罩防误关 — ✅ PASS
- `BaseEditModal.tsx:46-47`：`maskClosable = false` / `keyboard = false`，:61-62 透传 Modal ✓（T1 hunk 确认为默认值收紧，未删既有 prop）。
- T3：14 文件 +45/-1，19 处业务 Modal 显式补齐（backups×3、devices×3、cache×2 等）✓。
- fails_when 反向检查：9 commit 全量 diff grep `isFieldsTouched` = **0 命中** ✓；`Modal.confirm`（如 `cache/index.tsx:305` handleClearAll）未被加 maskClosable，行为保留 ✓。

### SC3（FIX-03）服务端排序 — ❌ **FAIL（BLOCKING B1）**
- 字段名对齐（此部分合格）：前端 `createSorterMeta` 字段 = 后端白名单逐字核对：
  - cache 7 列 key/type/location/ttl/size/createdAt/updatedAt = `internal/services/monitor/cache_service.go:143-151` `cacheAllowedSortFields` ✓
  - job 4 列 jobName/jobGroup/status/createdAt = `internal/services/scheduler/job_service.go:77-82` ✓
  - jobLog status + startTime→createdAt 映射（`job/index.tsx:78-81` mapJobLogSortField）= `job_log_service.go:60-65` ✓
  - 内存排序 `createSorter` 在 cache/job 目录 0 命中 ✓；白名单外 7 列（invokeTarget/cronExpression/concurrent/nextRunTime/duration/jobMessage/exceptionInfo）无 sorter ✓
- **断裂点（BLOCKING）**：全部 13 个受控排序列**只写 `sortOrder={getSortOrder(...)}`，漏写 `sorter: true`**：
  - `src/pages/monitor/cache/index.tsx:70-138`（getCacheTableColumns 7 列）
  - `src/pages/monitor/job/columns/jobColumns.tsx:38/45/73/100`（4 列）
  - `src/pages/monitor/job/columns/jobLogColumns.tsx:24/46`（2 列）
  - antd Table 列无 `sorter` 属性时：表头不渲染排序箭头、不可点击、onChange 的 sorter 参数不带 column → `useServerSort` 的 `handleSortChange` 永远收不到有效排序事件。**用户无法按任何列触发排序，SC3 交互链路断裂。**
  - 先例对照：`src/pages/network/discoveries/columns.tsx:29/37/74/84/92` 与 `src/pages/monitor/logs/columns/loginColumns.tsx:33/54/82/98` 均为 `sorter: true` + 受控 sortOrder。执行器未对齐先例写法。
  - 请求透传层（useJobData orderByColumn/isAsc、fetchCaches sort 覆盖参、回 1 页逻辑）实现正确，仅缺列级 `sorter: true`。

### SC4（FIX-04）console 收敛 — ✅ PASS
- 复扫亲跑（标准正则，排除测试）：network+monitor+components/network+CronSelector = **0 命中** ✓。
- 抽验 catch 三分支：`src/pages/network/devices/index.tsx:464-468/493-497`（errorFields 分流 return + handleApiError）、:512-514（直接 handleApiError）结构合规；`cache/index.tsx:243-244/279-280/317-318` 列表 error state + 操作 handleApiError 分工正确。
- 范围纪律：9 commit diff 49 文件全部在 `ruoyi-react-frontend/src` 域内；`src/pages/system` 仍有 49 处 console 未被触碰（其余域 235 处归 Phase 67 口径）✓。

### SC5（FIX-05）可访问名 — ❌ **FAIL（BLOCKING B2）**
- **38 处 `aria-label`/`title` 被误写进 JSX children 位置**（16 文件），如 `src/pages/monitor/cache/index.tsx:607-611`：
  ```jsx
  <Button icon={<SearchOutlined />} onClick={handleSearch}>
    aria-label="搜索"
    title="搜索"
    搜索
  </Button>
  ```
  后果：① `aria-label` 不是属性，读屏器仍读不到可访问名（SC5 核心目标未达成）；② 按钮渲染出字面文本 `aria-label="搜索" title="搜索" 搜索`，可见 UI 回归。同一文件 :598-600 的写法（属性位置）才是正确范式。
  - 涉及文件（children 误写分布）：cache 4、dashboard 1、job 2、logs 6、server 2、backups 2、command 1、credentials 4、devices 4、discoveries 1、executions 1、mac 4、MACHistoryPage 2、ports 2、templates 3（共 38，判定脚本按「前一行以 `>` 结尾」精确区分属性/children 位置）。
- 合格部分：属性位置的 39 处（如 cache:162 删除、cache:599 缓存操作）写法正确；`ActionButtons.tsx:57/94/132` 有 `title={action.label}`/`title="操作"`，但组件内**无 aria-label 输出**（PLAN T8 第 1 条要求 icon-only 输出加 aria-label，未落实）。
- SUMMARY 自报口径失效：复扫 `grep -rEn "aria-label" … | wc -l` = 72 与自报一致，但其中 38 处为无效 children 文本；SUMMARY「SC5 ✅ 73 处 aria-label + title」**失实**。

---

## 二、commit 审计与范围纪律

| 检查项 | 结果 |
|--------|------|
| 9 commit 逐一定位 | 全部存在，主题与任务对应 ✓ |
| diff 文件总数 | 49 文件，全部 `ruoyi-react-frontend/src` 域内（含 3 个测试文件：monitorColumns.test、jobLogColumns.test 随接口签名同步）✓ |
| Out of Scope | 无后端改动、无巨型文件拆分、无 i18n、无业务组件重设计 ✓ |
| 其余域未触碰 | system 等 235 处 console 存量保留 ✓ |
| fails_when 反向检查 | isFieldsTouched 0、无关闭确认弹窗代码、无骨架屏/整页 Spin（grep Skeleton 未见新增）、无白名单外排序字段 ✓ |
| 工作区遗留 | `internal/services/system/asset_columns_schema.json` 有未提交修改（不在 9 commit 内、与 phase 64 无关）；`.probe-elements.json`/`.probe-resolutions.json` untracked（SUMMARY 已上报）；需 Lead 裁决归属或回退（non-blocking N3） |

## 三、验证命令输出摘要

| 命令 | 结果 | 对照 |
|------|------|------|
| `npm run type-check` | 0 错误（exit 0） | ✅ |
| `npm run test -- --run`（全量，795s） | **557 files：3842 passed / 1 failed** | 唯一失败 = `src/pages/asset/reconciliation/dashboard/__tests__/index.test.tsx` 30s 超时，即基线记录的 dashboard flake，非新增回归 |
| FIX-04 console 复扫 | 0 命中 | ✅ |
| FIX-03 `createSorter` 复扫 | cache/job 0 命中 | ✅（但见 B1） |
| FIX-02 BaseEditModal 复扫 | 默认 false + 透传 | ✅ |
| FIX-01 `grep -rL` 复扫 | 3 命中均为已记录豁免 | ✅ |
| FIX-05 aria 复扫 | 72 命中，其中 38 处 children 无效 | ❌（见 B2） |

## 四、SUMMARY 失实核对

| SUMMARY 陈述 | 实测 | 判定 |
|--------------|------|------|
| T4/T5「接服务端排序」 | 字段/透传层正确，但列缺 `sorter: true`，排序交互不可触发 | **失实**（功能链路断裂） |
| T8「73 处补 aria-label + title」 | 38 处写在 children 位置，非属性，无效且污染 UI | **失实** |
| 「3843 passed / 0 failed」 | 实测 3842/1（dashboard flake 超时） | 不精确（flake 口径，非新增） |
| FIX-01/02/04 各复扫数字 | 与亲测一致 | ✓ |
| 偏离记录 1-8（提交策略/3 处 FIX-01 豁免/silent 模式等） | 抽验一致 | ✓ |

## 五、裁决

**verdict: ISSUES**

### Blocking（附修复指令）
- **B1（SC3）**：13 个受控排序列补 `sorter: true`——`src/pages/monitor/cache/index.tsx` getCacheTableColumns 的 key/type/location/ttl/size/createdAt/updatedAt 7 列、`src/pages/monitor/job/columns/jobColumns.tsx` jobName/jobGroup/status/createdAt 4 列、`src/pages/monitor/job/columns/jobLogColumns.tsx` startTime/status 2 列，各在 `sortOrder` 行旁加 `sorter: true,`（对齐 `discoveries/columns.tsx:29` 先例）。改后冒烟：点击表头出现排序箭头、Network 面板 orderByColumn/isAsc 正确。
- **B2（SC5）**：38 处 children 位置的 `aria-label="X"` / `title="X"` 两行从 `<Button ...>` 开标签之后移入开标签属性区（即把
  `<Button icon={…} onClick={…}>\n aria-label="搜索"\n title="搜索"\n 搜索\n</Button>`
  改为
  `<Button icon={…} onClick={…} aria-label="搜索" title="搜索">搜索</Button>`）。
  16 文件分布见 §SC5。同时给 `ActionButtons.tsx` icon-only 输出补 `aria-label={label}`（PLAN T8 第 1 条本要求）。

### Non-blocking
- N1：SUMMARY「3843/0」与复测「3842/1（dashboard flake）」不一致，验收按 flake 口径记录。
- N2：FIX-05 的 72/73 计数口径已因 children 误写失效，修复 B2 后需重扫（期望：aria-label 全部位于属性位置）。
- N3：工作区 `internal/services/system/asset_columns_schema.json` 未提交修改与 phase 64 无关，需 Lead 裁决回退或归属；2 个 `.probe-*.json` 探针文件同（SUMMARY 已上报）。
- N4：`src/pages/monitor/job/index.tsx:264-268` onChange 同时调用 `handleJobSortChange` 与 `resolveSorter`，B1 修复后冒烟确认无双重处理冲突。

> 本文件为验证产物，按约束不 commit，等 Lead 审核。

## 裁决更新（复核 f95d0597 后）

> GSD verifier 复核员 ｜ 2026-10-25 ｜ 方式：亲测复验（git show / grep / 复扫脚本 / type-check / 全量 vitest 亲跑）

**裁决更新：ISSUES → CLEAN**（保留上方首轮 ISSUES 记录）

逐项复核证据（commit `f95d0597`，18 文件，+54/-114）：

1. **范围纪律 ✓**：`git show f95d0597 --stat` 全部 18 文件均在 `ruoyi-react-frontend/src` 域内，无 `.planning/`、无后端、无其他域越界。
2. **B1 亲验 ✓**：`grep -n "sorter: true"` 计数 cache/index.tsx **7**（:80/91/100/115/124/133/142）、jobColumns.tsx **4**（:38/46/75/103）、jobLogColumns.tsx **2**（:24/47）= 13/13；白名单外 7 列（invokeTarget/cronExpression/concurrent/nextRunTime 等）确认无 sorter。抽样 cache 7 列与 job jobName/status 列：`sorter: true,` 与受控 `sortOrder: getSortOrder(field)` 逐列并存，对齐 `discoveries/columns.tsx:29` 先例写法。
3. **B2 亲验 ✓**：以「向上回溯开标签（上一行以 `>` 结尾即 children 形态）」标准复扫全项目 src 312 个 tsx（排除测试）：children 形态 aria-label/title 残留 **0**。抽验 cache/index.tsx:614、logs/index.tsx、templates/index.tsx 的 diff：`aria-label`/`title` 均已移入 `<Button ...>` 开标签属性区，可见文字保留于 children；`ActionButtons.tsx` 三处 Button 输出补 `aria-label={action.label}` / `aria-label="操作"` / `aria-label={options?.label || "删除"}`（:57/:95/:147），T8 第 1 条落实。（小瑕疵 non-blocking：若干开标签结尾留有一个多余空格如 `title="搜索" >`，不影响功能与 lint。）
4. **验证命令抽跑 ✓**：`npm run type-check` 0 错误（exit 0）；`npm run test -- --run` 全量亲跑 **557 files / 3843 passed / 0 failed**（745s，本轮连基线 dashboard flake 都未触发，无新增失败，优于 3842+1 口径）。
5. **N4 静态核对 ✓**：`src/pages/monitor/job/index.tsx:260-272` onChange 同时调 `handleJobSortChange`（hook 的 handleTableChange，仅更新受控 UI state）与 `resolveSorter`（同步解析本次事件排序参供 fetchJobs 使用）——为 `useServerSort.ts` 文档注释规定的同时调用范式，职责分离；sorter:true 就位后 antd 才会对 job 4 列触发带 field 的 sorter 事件，`resolveSorter` 按 sorterMetas 白名单解析，无双重处理冲突，语义正确。

遗留移交 Lead（均 non-blocking，不阻断验收）：N3 工作区 `asset_columns_schema.json` 未提交修改与 2 个 `.probe-*.json` 探针文件归属裁决；开标签尾随空格可在后续清理。

---

## B1/B2 修复记录

> 修复执行器 ｜ 2026-10-25 ｜ commit `f95d0597`（18 文件，+54/-114）

### B1（SC3）排序列补 `sorter: true` — 13/13 完成

| 文件 | 补齐数 | 列 |
|------|--------|----|
| `ruoyi-react-frontend/src/pages/monitor/cache/index.tsx` getCacheTableColumns | 7 | key/type/location/ttl/size/createdAt/updatedAt |
| `ruoyi-react-frontend/src/pages/monitor/job/columns/jobColumns.tsx` | 4 | jobName/jobGroup/status/createdAt |
| `ruoyi-react-frontend/src/pages/monitor/job/columns/jobLogColumns.tsx` | 2 | startTime（→createdAt 映射）/status |

复扫计数（`grep -c "sorter: true"`）：cache 7 / jobColumns 4 / jobLogColumns 2 ✓。白名单外 7 列未加 sorter ✓。

### B2（SC5）aria/title 从 children 移入开标签 — 38/38 完成

- 15 个页面文件共 38 组（76 行）按「向上回溯至开标签 `>`」判定脚本（`scan_children_aria.py`）修复，属性移入 `<Button ...>` 开标签，children 可见文字保留。
- 分布：cache 4、dashboard 1、job 2、logs 6、server 2、backups 1、command 1、credentials 4、devices 4、discoveries 1、executions 1、mac 4、MACHistoryPage 2、ports 2、templates 3（注：验证报告 §SC5 计 backups 2，实测 children 形态仅 1 组，总数 38 一致）。
- `src/components/shared/ActionButtons.tsx` 三处 Button 输出补 `aria-label`（列表按钮 `aria-label={action.label}`、下拉触发 `aria-label="操作"`、删除 Popconfirm 按钮 `aria-label={options?.label || "删除"}`）。

复扫：children 形态 aria/title 残留 **0**（全项目 src 范围同标准脚本复扫）。

### 验证输出

| 命令 | 结果 |
|------|------|
| `npm run type-check` | 0 错误（exit 0）✓ |
| `npm run test -- --run` | 557 files：3841 passed / 2 failed，唯一失败文件 = 基线记录的 dashboard flake（`src/pages/asset/reconciliation/dashboard/__tests__/index.test.tsx`），单独重跑 2/2 passed，无新增失败 ✓ |
| `npm run build` | 成功（built in 1m23s）✓ |
| B1 复扫 | 7 / 4 / 2 ✓ |
| B2 复扫 | 0 残留 ✓ |
