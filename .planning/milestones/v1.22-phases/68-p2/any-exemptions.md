# Phase 68 any 豁免登记（T5 / SC4）

> 基线（开工实测）：any 宽口径 52 处。B 类修复 26 处（monitor/logs 16、ExecutionDetailModal 3、ad-domain users/groups 4（handleTableChange 参数 6 处中 any 全清）、duty/schedules 1、AccountPoolTab 1、HubeiMap/GL 2、apikeys 2、monitor/logs/types.ts timeRange 含在内）。
> 注：pages/operations/floors/utils.ts:76 `value as any` 已带 `eslint-disable-next-line` 注释（既有豁免），归入下表 D 类。

## A 类 — 67 迁移路径 / FormModal 遗留（本轮不动）

| 文件路径:行号 | 形态 | 理由 |
|---|---|---|
| pages/network/backups/index.tsx:260 | `catch (error: any)` | network 六域 67 迁移路径遗留，catch 参数类型收窄涉及 UnknownError 模式改造，登记后续统一处理 |
| pages/network/command/index.tsx:141 | 同上 | 同上 |
| pages/network/credentials/index.tsx:213 | 同上 | 同上 |
| pages/network/devices/index.tsx:385 | 同上 | 同上 |
| pages/network/discoveries/index.tsx:197 | 同上 | 同上 |
| pages/network/executions/index.tsx:137 | 同上 | 同上 |
| pages/network/mac/index.tsx:251 | 同上 | 同上 |
| pages/network/templates/index.tsx:190 | 同上 | 同上 |
| pages/monitor/server/index.tsx:150 | handleTableChange `pagination/sorter: any` | A 类（67 收口清单内 monitor/server） |

## C 类 — 共享基础设施（类型面广，改动有扩散风险，登记）

| 文件路径:行号 | 形态 | 理由 |
|---|---|---|
| lib/api.ts:99-100 | `resolve: (value: any); reject: (reason?: any)` | Promise executor 回调，unknown 化会让全部调用点收窄，公共 API 面 |
| lib/api.ts:106 | `processRefreshQueue(error?: any)` | 同上（token 刷新队列） |
| components/shared/ExcelExport.tsx:148 | `formatFilterValue(key, value: any)` | 过滤值多态（string/number/array/Dayjs），unknown 化需全分支收窄 |
| components/shared/ExcelImport.tsx:261 | `customRequest={customRequest as any}` | antd UploadRequestObject 类型边界（rc-upload 签名不兼容） |
| utils/tableHelpers.tsx:122 | `sorter?: ((a: any, b: any) => number)` | 公共列配置 API，any 是 antd compare 签名的超集 |
| components/dashboard/widgets/WidgetRenderer.tsx:41 | `{...(componentProps as any)}` | 动态 widget props 分发，类型未知集合 |
| components/operations/WorkstationDeviceTable/index.tsx:447 | `render: (_: any, record)` | antd render 首参占位，改动波及列类型推断 |

## D 类 — 第三方声明 / 既有 eslint-disable（不动）

| 文件路径:行号 | 形态 | 理由 |
|---|---|---|
| types/externals.d.ts:15,25 | ambient declaration `any` | 第三方无类型库 ambient 声明，本就是 any 的合法落点 |
| pages/system/settings/default-theme.tsx:71,77 | `values: any; toHex = (v: any)` | 表单 values 动态结构 + 颜色转换工具，收益低 |
| pages/operations/rpa/tasks/modals/EditModal.tsx:183 | `(scriptData as any[]).map((item: any)` | localStorage JSON 解析回填，无类型来源 |
| pages/operations/floors/utils.ts:76 | `return value as any` | 已带 eslint-disable 注释（JSON.parse 回退，T 由调用点推断） |
| components/shared/FileUpload.tsx:259 | `customRequest as any` | 已带 eslint-disable 注释（同 ExcelImport） |

## SC4 数字

- 基线 52 → 余量 26（B 修 26 处清零；上表 A 9 + C 7 + D 7 + types/externals 合计 23 个 grep 行 = 26 处 any 计数，见提交时复扫输出）
- 说明：monitor/logs B 类含 types.ts `timeRange: any` → `[Dayjs, Dayjs] | null`、processTimeRangeParams、useLogData（LogFetchParams/Record<string,unknown>）、useLogActions（OperLog|LoginLog|null）、index.tsx loadOperLogs/loadLoginLogs。
