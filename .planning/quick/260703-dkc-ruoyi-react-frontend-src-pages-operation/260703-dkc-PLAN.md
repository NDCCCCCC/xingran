---
quick: 260703-dkc-xingran-react-frontend-src-pages-operation
type: execute
---

# 目标

重写 `xingran-react-frontend/src/pages/operations/assets/index.tsx` 的 `columns` 数组，按 `defaultAssetColumns` 的 52 个 key 全部补齐 col 定义；删除 4 个假列；修复 key/dataIndex 错配；将 `recipientName` 合并到 `deviceUserName`（后端真实字段）。同时修正 `defaultAssetColumns` 中重复/不存在的 key。

# 上下文

- 项目根 CLAUDE.md：已入仓
- `.planning/STATE.md`：v1.16 已闭环,v1.17 待启动,当前 quick 任务流
- `internal/models/asset.go` — 后端 Asset model 所有 json 字段
- `internal/services/operations/asset_service.go` — /ops/asset/list 返回 Find(&list) 完整 model
- `internal/services/operations/excel_config.go:181` — 确认 `deviceUserName` 是真"领取人"字段(替代不存在的 recipientName)

# 诊断摘要

## A. 4 个假列(原 columns 数组 line 363-392)

| 行号 | title | dataIndex | key(错) | 修复后 key | 修复后 title | 修复后 dataIndex | 修复后 render |
|---|---|---|---|---|---|---|---|
| 363-376 | 型号 | deviceTypeName | **deptName** | deviceTypeName | 设备类型 | deviceTypeName | 字符串 ellipsis |
| 377-383 | 拟报废 | machineIp | **mac1** | nbfStatus | 拟报废 | nbfStatus | Tag 0=否 green / 1=是 orange |
| 384-392 | 最后上线 | lastInventoryDate | **remark** | machineUptime | 最后上线 | machineUptime | dayjs YYYY-MM-DD HH:mm:ss |
| 348 保留 | 设备序列号 | devicesn | devicesn | devicesn | 设备序列号 | devicesn | Tooltip+span(已有) |

额外:line 363-376 的 render `(status: number) => Tag 0=正常/1=停用` 错把 deviceTypeName(字符串"台式机")当数字用,必须删。

## B. columns 数组缺失的 15 个 visible 列

| order | key | title | dataIndex | width | render 备注 |
|---|---|---|---|---|---|
| 4 | deviceModelName | 设备型号 | deviceModelName | 120 | ellipsis |
| 9 | deptName | 所属部门 | deptName | 120 | ellipsis |
| 13 | machineIp | 加域IP | machineIp | 120 | ellipsis |
| 14 | mac1 | 有线MAC | mac1 | 140 | ellipsis,转大写 |
| 27 | assetLocation | 资产位置 | assetLocation | 120 | ellipsis |
| 28 | buildingName | 所在大楼 | buildingName | 100 | ellipsis |
| 29 | floorName | 所在楼层 | floorName | 80 | ellipsis |
| 44 | signOrgnoName | 归属机构 | signOrgnoName | 150 | ellipsis |
| 45 | nowUserName | 责任人 | nowUserName | 100 | ellipsis |
| 46 | nowUserDeptCode | 部门编码 | nowUserDeptCode | 120 | ellipsis |
| 48 | status | 状态 | status | 80 | Tag 0=正常 green / 1=停用 red |
| 49 | nbfStatus | 拟报废 | nbfStatus | 80 | Tag 0=否 green / 1=是 orange |
| 50 | drawingDate | 接收日期 | drawingDate | 120 | dayjs YYYY-MM-DD |
| 51 | machineUptime | 最后上线 | machineUptime | 150 | dayjs YYYY-MM-DD HH:mm:ss |
| 52 | lastInventoryDate | 盘点日期 | lastInventoryDate | 120 | dayjs YYYY-MM-DD,sorter:true(已有 sorterMeta) |

## C. defaultAssetColumns 修正

- **删除** order=10 `key:"recipientName", label:"领取人"` 整行(后端无此字段,永远空)
- **保留** order=47 `key:"deviceUserName", label:"领取人"`(真字段,DB: deviceuser_name)
- 其他 51 个 key 不变

## D. 必须保留的特性

- devicesn 列 `fixed:"left"` + sorter
- action 列 `fixed:"right"`
- reconciliation 列(对账健康,占位 "-")
- getColumnSortOrder 3 个排序字段(devicesn/deviceTypeName/lastInventoryDate)继续可用
- dayjs 日期格式化
- ellipsis
- Tag 渲染
- useColumnConfig 列配置/列设置功能不破坏

# 任务

## Task 1: 修正 defaultAssetColumns + 重写 columns 数组

**文件**: `xingran-react-frontend/src/pages/operations/assets/index.tsx`

**Action**:

1. 在 `defaultAssetColumns` 数组中(line 74-142)删除 order=10 整行:
   ```ts
   { key: "recipientName", label: "领取人", visible: true, order: 10, width: 100, group: "部门与用户" },
   ```

2. 重写 `columns` 数组(line 348-456),从当前 4 个 fixed 列 + 35 个松散列重写为 53 个 col 定义(52 default + 1 action),顺序按 defaultAssetColumns 的 order 排列,并在末尾追加 action + reconciliation 特殊列。

**完整 columns 数组定义**(按 order 顺序):

```ts
const columns: ColumnsType<Asset> = [
  // order=1
  {
    title: "设备序列号", dataIndex: "devicesn", key: "devicesn", width: 150,
    fixed: "left", sorter: true, sortOrder: getColumnSortOrder("devicesn"),
    render: (text) => (
      <Tooltip title={text}><span style={{ cursor: "pointer" }}>{text}</span></Tooltip>
    ),
  },
  // order=2 sequenceNo(visible=false 但仍需定义否则列设置改可见时缺定义)
  { key: "sequenceNo", title: "序列号", dataIndex: "sequenceNo", width: 120, ellipsis: true },
  // order=3
  { key: "fixAssetNo", title: "固定资产编号", dataIndex: "fixAssetNo", width: 120, ellipsis: true },
  // order=4 ★新增
  { key: "deviceModelName", title: "设备型号", dataIndex: "deviceModelName", width: 120, ellipsis: true },
  // order=5 ★修复 key/sorter 错位
  {
    key: "deviceTypeName", title: "设备类型", dataIndex: "deviceTypeName", width: 100,
    ellipsis: true, sorter: true, sortOrder: getColumnSortOrder("deviceTypeName"),
  },
  // order=6
  { key: "deviceCategorySecondName", title: "设备中类", dataIndex: "deviceCategorySecondName", width: 120, ellipsis: true },
  // order=7
  { key: "deviceBasicTypeName", title: "是否固定资产", dataIndex: "deviceBasicTypeName", width: 100, ellipsis: true },
  // order=8
  { key: "deviceStatusName", title: "设备状态", dataIndex: "deviceStatusName", width: 100, ellipsis: true },
  // order=9 ★新增
  { key: "deptName", title: "所属部门", dataIndex: "deptName", width: 120, ellipsis: true },
  // order=10 已删除(recipientName 后端不存在)
  // order=11
  { key: "recipientPhone", title: "领取人电话", dataIndex: "recipientPhone", width: 120, ellipsis: true },
  // order=12
  { key: "recipientEmail", title: "领取人邮箱", dataIndex: "recipientEmail", width: 150, ellipsis: true },
  // order=13 ★新增
  { key: "machineIp", title: "加域IP", dataIndex: "machineIp", width: 120, ellipsis: true },
  // order=14 ★新增
  { key: "mac1", title: "有线MAC", dataIndex: "mac1", width: 140, ellipsis: true,
    render: (text) => (text ? String(text).toUpperCase() : "-") },
  // order=15
  { key: "assetHostname", title: "主机名", dataIndex: "assetHostname", width: 120, ellipsis: true },
  // order=16
  { key: "osName", title: "操作系统", dataIndex: "osName", width: 100, ellipsis: true },
  // order=17
  { key: "osVersion", title: "系统版本", dataIndex: "osVersion", width: 100, ellipsis: true },
  // order=18
  { key: "cpuModel", title: "CPU型号", dataIndex: "cpuModel", width: 120, ellipsis: true },
  // order=19
  { key: "cpuCores", title: "CPU核心数", dataIndex: "cpuCores", width: 80, ellipsis: true },
  // order=20
  { key: "memoryCapacity", title: "内存容量", dataIndex: "memoryCapacity", width: 100, ellipsis: true },
  // order=21
  { key: "diskCapacity", title: "硬盘容量", dataIndex: "diskCapacity", width: 100, ellipsis: true },
  // order=22
  { key: "graphicsCard", title: "显卡型号", dataIndex: "graphicsCard", width: 120, ellipsis: true },
  // order=23
  { key: "purchaseDate", title: "购置日期", dataIndex: "purchaseDate", width: 120,
    render: (date) => (date ? dayjs(date).format("YYYY-MM-DD") : "-"), ellipsis: true },
  // order=24
  { key: "warrantyExpireDate", title: "保修到期日", dataIndex: "warrantyExpireDate", width: 120,
    render: (date) => (date ? dayjs(date).format("YYYY-MM-DD") : "-"), ellipsis: true },
  // order=25
  { key: "supplierName", title: "供应商", dataIndex: "supplierName", width: 120, ellipsis: true },
  // order=26
  { key: "manufacturerName", title: "制造商", dataIndex: "manufacturerName", width: 120, ellipsis: true },
  // order=27 ★新增
  { key: "assetLocation", title: "资产位置", dataIndex: "assetLocation", width: 120, ellipsis: true },
  // order=28 ★新增
  { key: "buildingName", title: "所在大楼", dataIndex: "buildingName", width: 100, ellipsis: true },
  // order=29 ★新增
  { key: "floorName", title: "所在楼层", dataIndex: "floorName", width: 80, ellipsis: true },
  // order=30
  { key: "workstationName", title: "所在工位", dataIndex: "workstationName", width: 120, ellipsis: true },
  // order=31
  { key: "serverRoomName", title: "所在机房", dataIndex: "serverRoomName", width: 120, ellipsis: true },
  // order=32
  { key: "networkSwitchName", title: "网络交换机", dataIndex: "networkSwitchName", width: 120, ellipsis: true },
  // order=33
  { key: "switchPort", title: "交换机端口", dataIndex: "switchPort", width: 120, ellipsis: true },
  // order=34
  { key: "voltage", title: "电压", dataIndex: "voltage", width: 80, ellipsis: true },
  // order=35
  { key: "useStatusName", title: "使用状态", dataIndex: "useStatusName", width: 100, ellipsis: true },
  // order=36
  { key: "financeCode", title: "财务编码", dataIndex: "financeCode", width: 100, ellipsis: true },
  // order=37
  { key: "projectCode", title: "项目编码", dataIndex: "projectCode", width: 100, ellipsis: true },
  // order=38
  { key: "assetCost", title: "资产成本", dataIndex: "assetCost", width: 100, ellipsis: true },
  // order=39
  { key: "depreciationPeriod", title: "折旧年限", dataIndex: "depreciationPeriod", width: 100, ellipsis: true },
  // order=40
  { key: "netBookValue", title: "账面净值", dataIndex: "netBookValue", width: 100, ellipsis: true },
  // order=41
  { key: "depreciationRate", title: "折旧率", dataIndex: "depreciationRate", width: 100, ellipsis: true },
  // order=42
  { key: "accumulatorDepreciation", title: "累计折旧", dataIndex: "accumulatorDepreciation", width: 120, ellipsis: true },
  // order=43
  { key: "remark", title: "备注", dataIndex: "remark", width: 200, ellipsis: true },
  // order=44 ★新增
  { key: "signOrgnoName", title: "归属机构", dataIndex: "signOrgnoName", width: 150, ellipsis: true },
  // order=45 ★新增
  { key: "nowUserName", title: "责任人", dataIndex: "nowUserName", width: 100, ellipsis: true },
  // order=46 ★新增
  { key: "nowUserDeptCode", title: "部门编码", dataIndex: "nowUserDeptCode", width: 120, ellipsis: true },
  // order=47 deviceUserName 取代 recipientName(真领取人字段)
  { key: "deviceUserName", title: "领取人", dataIndex: "deviceUserName", width: 100, ellipsis: true },
  // order=48 ★新增
  {
    key: "status", title: "状态", dataIndex: "status", width: 80,
    render: (s: number) => (
      <Tag color={s === 0 ? "green" : "red"}>{s === 0 ? "正常" : "停用"}</Tag>
    ),
  },
  // order=49 ★新增 + 修复旧错位(原 line 377-383 把 machineIp 当 nbfStatus 渲染日期)
  {
    key: "nbfStatus", title: "拟报废", dataIndex: "nbfStatus", width: 80,
    render: (s: number) => (
      <Tag color={s === 1 ? "orange" : "default"}>{s === 1 ? "是" : "否"}</Tag>
    ),
  },
  // order=50 ★新增
  { key: "drawingDate", title: "接收日期", dataIndex: "drawingDate", width: 120,
    render: (date) => (date ? dayjs(date).format("YYYY-MM-DD") : "-"), ellipsis: true },
  // order=51 ★新增 + 修复旧错位(原 line 384-392 remark key + 实际渲染盘点日期)
  { key: "machineUptime", title: "最后上线", dataIndex: "machineUptime", width: 150,
    render: (date) => (date ? dayjs(date).format("YYYY-MM-DD HH:mm:ss") : "-"), ellipsis: true },
  // order=52
  {
    key: "lastInventoryDate", title: "盘点日期", dataIndex: "lastInventoryDate", width: 120,
    ellipsis: true, sorter: true, sortOrder: getColumnSortOrder("lastInventoryDate"),
    render: (date) => (date ? dayjs(date).format("YYYY-MM-DD") : "-"),
  },
  // 末尾特殊列:action + reconciliation(不参与 defaultAssetColumns 排序)
  {
    title: "操作", key: "action", width: 120, fixed: "right",
    render: (_, record) => (
      <AssetRow record={record} onEdit={handleEdit} onDelete={handleDelete} />
    ),
  },
  {
    title: "对账健康", key: "reconciliation", width: 96,
    render: (_: unknown, _record: Asset) => <>-</>,
  },
];
```

**注意**:
- 旧 columns 中 line 363-376 那个错把 `deviceTypeName` 字符串当 status 数字用 Tag 渲染"正常/停用"的 render 必须删(改回纯 ellipsis 或 sorter 标记)
- 旧 columns 中 line 377-383 那个把 `machineIp` 当 `nbfStatus` 渲染日期的必须删
- 旧 columns 中 line 384-392 那个把 `lastInventoryDate` 渲染日期但 key 写 remark 的必须删
- 旧 columns 末尾"额外的列定义"注释下的 35 行(line 394-429)与新定义重复,整段替换

**Verify**:
```bash
cd xingran-react-frontend
npm run type-check 2>&1 | tail -20
npm run lint 2>&1 | tail -20
```

**Done**:
- 4 个假列(错 key / 错 render)全部删除
- 15 个缺失的 visible 列全部新增
- `defaultAssetColumns` 删 1 行(order=10 recipientName),剩 51 行
- `columns` 数组共 53 个 col(52 default + action + reconciliation)
- `npm run type-check` 0 错误
- `npm run lint` 0 错误(允许既有 warning)
- 列设置弹窗打开后能正确显示所有 51 个 default 列(已用列宽覆盖列设置)

---

# 验证

- `npm run type-check` 0 错误(关键:ColumnsType<Asset> 所有 dataIndex 都在 Asset 类型上,除 recipientName 已删)
- `npm run lint` 0 错误
- 浏览器手动检查(可选,quick 任务不强制):打开 /ops/assets,确认"设备类型"列显示字符串而非"正常/停用"Tag;"拟报废"列显示 0/1 Tag;"最后上线"显示完整时间;15 个新增列能正常显示

# 成功标准

- [ ] 4 个假列全删
- [ ] 15 个缺失 visible 列全补
- [ ] `defaultAssetColumns` order=10 删,剩 51 行
- [ ] `columns` 数组 53 个定义齐
- [ ] `npm run type-check` 0 错误
- [ ] `npm run lint` 0 错误

# 输出

完成后无需写 SUMMARY(quick 任务流),直接在 git status 报告修改的文件清单等待用户提交确认。
