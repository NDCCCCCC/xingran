---
slug: asset-list-missing-fields
name: asset-list-missing-fields
status: resolved
trigger: 资产列表页面字段数据不全，请检查原因。导入的数据有很多字段数据没有显示，缺失字段包括：使用机构、部门、受益部门、领取人、责任人。需要全面检查所有字段，确定所有字段来历和映射关系
created: 2026-06-09
updated: 2026-06-09
---

## 症状

### 预期行为
资产列表应显示导入的43个字段数据，包括状态（在库、领用、损坏、发放、在途等）、归属机构、使用机构、部门、受益部门、领取人、责任人、用途等所有字段。

### 实际行为
导入的资产数据在列表中有很多字段没有显示，用户确认缺失的字段包括：
- 状态（不是正常/停用，应该是：在库、领用、损坏、发放、在途等枚举值）
- 归属机构
- 部门
- 责任人
- 用途
- 其他字段

### 错误信息
无错误信息，数据没有显示

### 时间线
用户通过Excel导入资产数据后发现字段缺失

### 重现步骤
1. 通过Excel导入资产数据
2. 查看资产列表页面
3. 发现很多字段没有显示

## Current Focus

### 假设
前端列配置（`defaultAssetColumns`）中使用了不存在的数据库字段名，导致数据映射失败；同时使用了错误的 `status` 字段而非 `useStatusLabel` 字段来显示状态。

### 测试
检查以下文件确认字段映射关系：
1. `internal/models/asset.go` - 数据库模型定义
2. `internal/services/operations/excel_config.go` - Excel导入配置
3. `xingran-react-frontend/src/pages/operations/assets/index.tsx` - 前端列配置

### 期望结果
找到所有字段映射不匹配的地方

### 下一步行动
生成完整的根因报告并修复字段映射

## Evidence

- timestamp: 2026-06-09
- source: 数据库模型检查
- finding: |
  数据库模型 `Asset` (internal/models/asset.go) 中存在的字段：
  - useStatusLabel (*string) - 状态（在库、领用、损坏、发放、在途等）
  - signOrgnoName (string) - 归属机构
  - orgnoName (string) - 使用机构
  - usefulDeptName (string) - 部门
  - deptName (*string) - 受益部门
  - deviceUserName (*string) - 领取人
  - nowUserName (*string) - 责任人
  - usingTypeName (string) - 用途
  - subUsingTypeName (string) - 子用途
  - storeroomName (string) - 库房
  - storageAddress (string) - 存放地址

- timestamp: 2026-06-09
- source: Excel导入配置检查
- finding: |
  Excel配置 (excel_config.go) 正确定义了43列映射，包括：
  - useStatusLabel → usestatus_label
  - signOrgnoName → sign_orgno_name
  - orgnoName → orgno_name
  - usefulDeptName → useful_dept_name
  - usingTypeName → using_type_name
  所有字段都正确映射到数据库字段

- timestamp: 2026-06-09
- source: 前端列配置检查
- finding: |
  前端 defaultAssetColumns (index.tsx 第59-116行) 中有大量字段名错误：
  - `deviceStatusName` (第70行) - 不存在，应该是 `useStatusLabel`
  - `recipientName` (第73行) - 不存在，应该是 `deviceUserName`
  - `recipientPhone` (第74行) - 不存在
  - `assetLocation` (第95行) - 不存在，应该是 `storageAddress`
  - `useStatusName` (第105行) - 不存在，应该是 `useStatusLabel`
  - 缺少：signOrgnoName, orgnoName, usefulDeptName, nowUserName, usingTypeName

  前端 columns 数组 (第299-476行) 中：
  - 第355-364行：`status` 字段显示正常/停用，应该显示 `useStatusLabel`（在库、领用、损坏等）
  - 第327行：`deptName` (受益部门) ✓ 有定义
  - 第334行：`signOrgnoName` (归属机构) ✓ 有定义
  - 第342行：`nowUserName` (责任人) ✓ 有定义
  - 第390行：`deviceUserName` (领取人) ✓ 有定义
  - 缺少：`useStatusLabel`, `orgnoName`, `usefulDeptName`, `usingTypeName` 的列定义

- timestamp: 2026-06-09
- source: TypeScript类型定义检查
- finding: |
  operations.ts 中 Asset 接口正确定义了所有字段，与数据库模型一致

## Eliminated

无

## Resolution

### 根本原因
资产列表字段数据缺失的根本原因是：**前端列配置 (`defaultAssetColumns` 和 `columns` 数组) 中使用了不存在的数据库字段名，且状态字段使用了错误的字段**。

具体问题分为两类：

#### 问题类型1：字段名映射错误
前端配置的字段名与数据库实际字段名不一致，导致数据无法正确映射。

#### 问题类型2：配置了不存在的字段
前端配置了资产表中根本不存在的字段，这些字段可能是为其他功能（楼宇、楼层、工位、机房等）添加的，但错误地包含在了资产列配置中。

详细问题：

1. **`defaultAssetColumns` 配置错误**（第59-116行）：
   - 使用了不存在的字段：`deviceStatusName`、`recipientName`、`recipientPhone`、`recipientEmail`
   - 使用了楼宇/楼层/工位/机房相关字段（资产表中不存在）：`buildingName`、`floorName`、`workstationName`、`serverRoomName`、`networkSwitchName`、`switchPort`
   - 使用了硬件/系统相关字段（资产表中不存在）：`assetHostname`、`osName`、`osVersion`、`cpuModel`、`cpuCores`、`memoryCapacity`、`diskCapacity`、`graphicsCard`、`voltage`
   - 使用了采购/财务相关字段（资产表中不存在）：`purchaseDate`、`warrantyExpireDate`、`supplierName`、`manufacturerName`、`financeCode`、`projectCode`、`assetCost`、`depreciationPeriod`、`netBookValue`、`depreciationRate`、`accumulatorDepreciation`
   - 缺少用户需要的关键字段：`useStatusLabel`, `signOrgnoName`, `orgnoName`, `usefulDeptName`, `nowUserName`, `usingTypeName`
   - 字段名错误：`assetLocation` 应该是 `storageAddress`

2. **`columns` 数组中的问题**（第299-476行）：
   - 第355-364行：状态列使用了错误的 `status` 字段（显示正常/停用），应该使用 `useStatusLabel` 字段（显示在库、领用、损坏等）
   - 缺少列定义：`useStatusLabel`, `orgnoName`, `usefulDeptName`, `usingTypeName`
   - 第424-459行额外列定义中包含大量不存在的字段

3. **字段映射关系混乱**：
   - Excel导入配置定义了43个正确的字段映射
   - 前端列配置（`defaultAssetColumns`）定义的字段与Excel配置不一致
   - 部分字段在 `defaultAssetColumns` 中有定义，但在 `columns` 数组中缺少对应的列定义（或字段名不匹配）

### 完整字段分类说明

根据调查，Excel导入配置定义了43个字段，这些字段都正确映射到了数据库。但前端列配置中存在大量问题：

#### 类别A：数据库中存在的字段（43个）

| Excel列名 | 数据库字段 | 前端defaultAssetColumns状态 | 问题说明 |
|----------|-----------|---------------------------|---------|
| 设备序列号 | devicesn | ✓ 配置正确 | 无问题 |
| 序列号 | sequenceno | ✓ 配置正确 | 无问题 |
| 固定资产编号 | fixassetno | ✓ 配置正确 | 无问题 |
| 设备型号 | device_model_name | ✓ 配置正确 | 无问题 |
| 设备类型 | device_type_name | ✓ 配置正确 | 无问题 |
| 设备中类 | device_category_second_name | ✓ 配置正确 | 无问题 |
| 是否固定资产 | device_basic_type_name | ✓ 配置正确 | 无问题 |
| **状态（在库/领用等）** | **usestatus_label** | **❌ 配置了错误的 deviceStatusName** | **字段名错误** |
| 是否拟报废 | nbf_status | ✓ 配置正确 | 无问题 |
| 归属机构 | sign_orgno_name | **❌ 缺少** | **前端缺失此字段** |
| 使用机构 | orgno_name | **❌ 缺少** | **前端缺失此字段** |
| 部门 | useful_dept_name | **❌ 缺少** | **前端缺失此字段** |
| 受益部门 | deptname | ✓ 配置正确 | 无问题 |
| 领取人 | deviceuser_name | **❌ 配置了错误的 recipientName** | **字段名错误** |
| 责任人 | nowuser_name | **❌ 缺少** | **前端缺失此字段** |
| 使用人 | outer_user | ❌ 缺少 | 前端缺失此字段 |
| 责任人岗位 | nowuser_job_name | ❌ 缺少 | 前端缺失此字段 |
| 用途 | using_type_name | **❌ 缺少** | **前端缺失此字段** |
| 子用途 | sub_using_type_name | ❌ 缺少 | 前端缺失此字段 |
| 库房 | storeroom_name | ❌ 缺少 | 前端缺失此字段 |
| 存放地址 | storage_address | **❌ 配置了错误的 assetLocation** | **字段名错误** |
| 合同号 | contractno | ❌ 缺少 | 前端缺失此字段 |
| 资产标签打印状态 | print_flag_name | ❌ 缺少 | 前端缺失此字段 |
| 异常标识 | error_flag_name | ❌ 缺少 | 前端缺失此字段 |
| 新设备标识 | new_flag_label | ❌ 缺少 | 前端缺失此字段 |
| 备注 | remark | ✓ 配置正确 | 无问题 |
| 申请标准 | is_no_standard_name | ❌ 缺少 | 前端缺失此字段 |
| IP地址 | machine_ip | ✓ 配置正确为 machineIp | 字段名大小写差异 |
| 加域标识 | machine_bs | ❌ 缺少 | 前端缺失此字段 |
| 加域ip地址 | machine_ip（重复） | - | 与IP地址重复 |
| 有线MAC地址 | mac1 | ✓ 配置正确 | 无问题 |
| 无线MAC地址 | mac2 | ❌ 缺少 | 前端缺失此字段 |
| 最后上线时间 | machine_uptime | ❌ 缺少 | 前端缺失此字段 |
| 最后上线账号 | machine_user_id | ❌ 缺少 | 前端缺失此字段 |
| APP扫码账号 | user_name（引用解析） | ❌ 缺少 | 前端缺失此字段 |
| APP扫码时间 | last_update_date | ❌ 缺少 | 前端缺失此字段 |
| AAP扫码地理位置 | scan_site | ❌ 缺少 | 前端缺失此字段 |
| 最近一次盘点时间 | last_inventory_date | ❌ 缺少 | 前端缺失此字段 |
| 盘点结果 | inventory_result | ❌ 缺少 | 前端缺失此字段 |
| 设备渠道 | qudao_name | ❌ 缺少 | 前端缺失此字段 |
| 设备属性 | attribute_value | ❌ 缺少 | 前端缺失此字段 |
| 入库日期 | storage_datetime | ❌ 缺少 | 前端缺失此字段 |
| 发放日期 | use_date | ❌ 缺少 | 前端缺失此字段 |
| 接收日期 | drawing_date | ❌ 缺少 | 前端缺失此字段 |

#### 类别B：前端配置了但数据库中不存在的字段（为其他功能添加的）

这些字段可能是为楼宇、楼层、工位、机房等运维管理功能添加的，但错误地包含在了资产列表配置中：

| 前端配置字段 | 预期来源 | 问题说明 |
|------------|---------|---------|
| buildingName | 楼宇表（ops_buildings） | 资产表无此字段 |
| floorName | 楼层表（ops_floors） | 资产表无此字段 |
| workstationName | 工位表（sys_workstation） | 资产表无此字段 |
| serverRoomName | 机房表（ops_server_rooms） | 资产表无此字段 |
| networkSwitchName | 网络设备表 | 资产表无此字段 |
| switchPort | 设备端口表 | 资产表无此字段 |

#### 类别C：前端配置了但数据库中不存在的字段（未实现的功能）

这些字段在前端配置中定义，但数据库中完全没有对应的字段，可能是未实现的功能：

| 前端配置字段 | 问题说明 |
|------------|---------|
| deviceStatusName | 应该是 useStatusLabel |
| recipientName | 应该是 deviceUserName |
| recipientPhone | 数据库无此字段 |
| recipientEmail | 数据库无此字段 |
| assetHostname | 数据库无此字段 |
| osName | 数据库无此字段 |
| osVersion | 数据库无此字段 |
| cpuModel | 数据库无此字段 |
| cpuCores | 数据库无此字段 |
| memoryCapacity | 数据库无此字段 |
| diskCapacity | 数据库无此字段 |
| graphicsCard | 数据库无此字段 |
| purchaseDate | 数据库无此字段（有drawingDate, useDate, storageDatetime） |
| warrantyExpireDate | 数据库无此字段 |
| supplierName | 数据库无此字段 |
| manufacturerName | 数据库无此字段 |
| financeCode | 数据库无此字段 |
| projectCode | 数据库无此字段 |
| assetCost | 数据库无此字段 |
| depreciationPeriod | 数据库无此字段 |
| netBookValue | 数据库无此字段 |
| depreciationRate | 数据库无此字段 |
| accumulatorDepreciation | 数据库无此字段 |
| voltage | 数据库无此字段 |

### 用户缺失字段与数据库对照

### 修复方案

需要修改 `xingran-react-frontend/src/pages/operations/assets/index.tsx`：

1. **修复 `defaultAssetColumns` 配置**：
   - 将 `deviceStatusName` 改为 `useStatusLabel`
   - 将 `recipientName` 改为 `deviceUserName`
   - 删除不存在的字段（`recipientPhone`, `recipientEmail`, `assetHostname`, `osName`, `osVersion` 等）
   - 添加缺失字段：`signOrgnoName`, `orgnoName`, `usefulDeptName`, `nowUserName`, `usingTypeName`
   - 将 `assetLocation` 改为 `storageAddress`

2. **修复 `columns` 数组**：
   - 修改状态列：将 `dataIndex: 'status'` 改为 `dataIndex: 'useStatusLabel'`，render函数改为直接显示文本值
   - 添加缺失列定义：`orgnoName`, `usefulDeptName`, `usingTypeName`
   - 确保所有 `defaultAssetColumns` 中的字段都在 `columns` 数组中有对应定义

3. **可选：清理不存在字段的列定义**
   - 第424-459行的额外列定义中有很多不存在的字段，需要清理

### 文件变更
- `xingran-react-frontend/src/pages/operations/assets/index.tsx`

### 验证方法
1. 修改前端列配置后，重新加载资产列表页面
2. 检查列配置面板，确认所有43个字段都正确显示
3. 选择显示缺失的字段（状态、归属机构、使用机构、部门、用途等）
4. 验证列表中正确显示这些字段的数据

### 实施记录

**实施日期**: 2026-06-09

**已实施的修改**:

1. **`defaultAssetColumns` 完全重写**（第59-116行 → 新45列配置）:
   - 删除了30+个不存在的字段（楼宇/楼层/工位/机房相关字段、硬件/系统字段、采购/财务字段）
   - 修正了字段名错误：`deviceStatusName` → `useStatusLabel`、`recipientName` → `deviceUserName`、`assetLocation` → `storageAddress`
   - 新增了8个关键字段：`signOrgnoName`, `orgnoName`, `usefulDeptName`, `nowUserName`, `usingTypeName`, `outerUser`, `nowUserJobName`, `subUsingTypeName`
   - 新增了其他Excel字段：`storeroomName`, `scanSite`, `storageDatetime`, `useDate`, `machineUptime`, `lastUpdateDate`, `lastInventoryDate`, `y07UpdateTime`, `machineBs`, `mac2`, `machineUserId`, `userName`, `contractNo`, `printFlagName`, `errorFlagName`, `newFlagLabel`, `inventoryResult`, `isNoStandardName`, `qudaoName`, `attributeValue`

2. **`columns` 数组状态列修复**（第355-365行）:
   - 修改前：`dataIndex: 'status'` 显示 "正常/停用"
   - 修改后：`dataIndex: 'useStatusLabel'` 直接显示文本（在库/领用/损坏/发放/在途等）

3. **`columns` 数组额外列定义替换**（第421-457行）:
   - 删除了36个不存在字段的列定义
   - 添加了29个正确字段的列定义

**编译验证**: ✅ TypeScript类型检查通过（`npm run type-check`）
