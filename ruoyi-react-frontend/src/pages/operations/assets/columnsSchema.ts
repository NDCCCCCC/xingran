/**
 * 资产列表列定义 — 单一真理源
 *
 * 此文件为资产列表 (asset.list) 列定义的单一真理源。
 * 后端 sync-columns-schema.mjs 会序列化此文件到
 * internal/services/system/asset_columns_schema.json，
 * 编辑后必须跑 `npm run sync-columns-schema` 同步后端 embed JSON。
 *
 * 字段约束（与后端 models/system/requests.ColumnConfigItem 一致）：
 *   - key     列唯一标识（必须与后端 Asset DTO 字段对应）
 *   - label   表头中文文案
 *   - visible 默认是否可见
 *   - order   默认显示顺序（从 1 开始）
 *   - width   列宽（px）
 *   - group   列分组（用于列设置面板的分组展示）
 */

export interface AssetColumnConfig {
  key: string;
  label: string;
  visible: boolean;
  order: number;
  width: number;
  group: string;
}

export const defaultAssetColumns: AssetColumnConfig[] = [
  // 核心标识 (3列)
  { key: "devicesn", label: "设备序列号", visible: true, order: 1, width: 150, group: "核心标识" },
  { key: "sequenceNo", label: "序列号", visible: false, order: 2, width: 120, group: "核心标识" },
  { key: "fixAssetNo", label: "固定资产编号", visible: true, order: 3, width: 120, group: "核心标识" },
  // 设备信息 (4列)
  { key: "deviceModelName", label: "设备型号", visible: true, order: 4, width: 120, group: "设备信息" },
  { key: "deviceTypeName", label: "设备类型", visible: true, order: 5, width: 100, group: "设备信息" },
  { key: "deviceCategorySecondName", label: "设备中类", visible: false, order: 6, width: 120, group: "设备信息" },
  { key: "deviceBasicTypeName", label: "是否固定资产", visible: true, order: 7, width: 100, group: "设备信息" },
  // 设备状态 (1列)
  { key: "deviceStatusName", label: "设备状态", visible: true, order: 8, width: 100, group: "设备状态" },
  // 所属部门和领取人 (4列)
  { key: "usefulDeptName", label: "所属部门", visible: true, order: 9, width: 120, group: "部门与用户" },
  { key: "recipientPhone", label: "领取人电话", visible: false, order: 11, width: 120, group: "部门与用户" },
  { key: "recipientEmail", label: "领取人邮箱", visible: false, order: 12, width: 150, group: "部门与用户" },
  // 网络信息 (2列)
  { key: "machineIp", label: "加域IP", visible: true, order: 13, width: 120, group: "网络信息" },
  { key: "mac1", label: "有线MAC", visible: true, order: 14, width: 140, group: "网络信息" },
  // 主机和操作系统 (2列)
  { key: "assetHostname", label: "主机名", visible: false, order: 15, width: 120, group: "系统信息" },
  { key: "osName", label: "操作系统", visible: false, order: 16, width: 100, group: "系统信息" },
  { key: "osVersion", label: "系统版本", visible: false, order: 17, width: 100, group: "系统信息" },
  // 硬件信息 (5列)
  { key: "cpuModel", label: "CPU型号", visible: false, order: 18, width: 120, group: "硬件信息" },
  { key: "cpuCores", label: "CPU核心数", visible: false, order: 19, width: 80, group: "硬件信息" },
  { key: "memoryCapacity", label: "内存容量", visible: false, order: 20, width: 100, group: "硬件信息" },
  { key: "diskCapacity", label: "硬盘容量", visible: false, order: 21, width: 100, group: "硬件信息" },
  { key: "graphicsCard", label: "显卡型号", visible: false, order: 22, width: 120, group: "硬件信息" },
  // 采购与保修 (3列)
  { key: "purchaseDate", label: "购置日期", visible: false, order: 23, width: 120, group: "采购信息" },
  { key: "warrantyExpireDate", label: "保修到期日", visible: false, order: 24, width: 120, group: "采购信息" },
  { key: "supplierName", label: "供应商", visible: false, order: 25, width: 120, group: "采购信息" },
  { key: "manufacturerName", label: "制造商", visible: false, order: 26, width: 120, group: "采购信息" },
  // 位置信息 (5列)
  { key: "assetLocation", label: "资产位置", visible: true, order: 27, width: 120, group: "位置信息" },
  { key: "buildingName", label: "所在大楼", visible: true, order: 28, width: 100, group: "位置信息" },
  { key: "floorName", label: "所在楼层", visible: true, order: 29, width: 80, group: "位置信息" },
  { key: "workstationName", label: "所在工位", visible: false, order: 30, width: 120, group: "位置信息" },
  { key: "serverRoomName", label: "所在机房", visible: false, order: 31, width: 120, group: "位置信息" },
  // 网络连接 (2列)
  { key: "networkSwitchName", label: "网络交换机", visible: false, order: 32, width: 120, group: "网络连接" },
  { key: "switchPort", label: "交换机端口", visible: false, order: 33, width: 120, group: "网络连接" },
  { key: "voltage", label: "电压", visible: false, order: 34, width: 80, group: "网络连接" },
  // 使用状态 (1列)
  { key: "useStatusLabel", label: "使用状态", visible: true, order: 35, width: 100, group: "使用状态" },
  // 财务信息 (7列)
  { key: "financeCode", label: "财务编码", visible: false, order: 36, width: 100, group: "财务信息" },
  { key: "projectCode", label: "项目编码", visible: false, order: 37, width: 100, group: "财务信息" },
  { key: "assetCost", label: "资产成本", visible: false, order: 38, width: 100, group: "财务信息" },
  { key: "depreciationPeriod", label: "折旧年限", visible: false, order: 39, width: 100, group: "财务信息" },
  { key: "netBookValue", label: "账面净值", visible: false, order: 40, width: 100, group: "财务信息" },
  { key: "depreciationRate", label: "折旧率", visible: false, order: 41, width: 100, group: "财务信息" },
  { key: "accumulatorDepreciation", label: "累计折旧", visible: false, order: 42, width: 120, group: "财务信息" },
  // 其他信息 (1列)
  { key: "remark", label: "备注", visible: true, order: 43, width: 200, group: "其他" },
  // 归属与责任 (5列)
  { key: "signOrgnoName", label: "归属机构", visible: true, order: 44, width: 150, group: "归属与责任" },
  { key: "nowUserName", label: "责任人", visible: true, order: 45, width: 100, group: "归属与责任" },
  { key: "nowUserDeptCode", label: "部门编码", visible: true, order: 46, width: 120, group: "归属与责任" },
  { key: "deviceUserName", label: "领取人", visible: true, order: 47, width: 100, group: "归属与责任" },
  // 状态与日期 (4列)
  { key: "status", label: "状态", visible: true, order: 48, width: 80, group: "状态与日期" },
  { key: "nbfStatus", label: "拟报废", visible: true, order: 49, width: 80, group: "状态与日期" },
  { key: "drawingDate", label: "接收日期", visible: true, order: 50, width: 120, group: "状态与日期" },
  { key: "machineUptime", label: "最后上线", visible: true, order: 51, width: 150, group: "状态与日期" },
  { key: "lastInventoryDate", label: "盘点日期", visible: true, order: 52, width: 120, group: "状态与日期" },
  { key: "deptName", label: "受益部门", visible: false, order: 53, width: 120, group: "部门与用户" },
];