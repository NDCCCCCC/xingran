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
  {
    key: "fixAssetNo",
    label: "固定资产编号",
    visible: true,
    order: 3,
    width: 120,
    group: "核心标识",
  },
  // 设备信息 (6列)
  {
    key: "deviceModelName",
    label: "设备型号",
    visible: true,
    order: 4,
    width: 120,
    group: "设备信息",
  },
  {
    key: "deviceTypeName",
    label: "设备类型",
    visible: true,
    order: 5,
    width: 100,
    group: "设备信息",
  },
  {
    key: "deviceCategorySecondName",
    label: "设备中类",
    visible: false,
    order: 6,
    width: 120,
    group: "设备信息",
  },
  {
    key: "deviceBasicTypeName",
    label: "是否固定资产",
    visible: true,
    order: 7,
    width: 100,
    group: "设备信息",
  },
  { key: "qudaoName", label: "设备渠道", visible: false, order: 54, width: 100, group: "设备信息" },
  {
    key: "attributeValue",
    label: "设备属性",
    visible: false,
    order: 55,
    width: 120,
    group: "设备信息",
  },
  // 所属部门和领取人 (2列)
  {
    key: "usefulDeptName",
    label: "所属部门",
    visible: true,
    order: 9,
    width: 120,
    group: "部门与用户",
  },
  {
    key: "deptName",
    label: "受益部门",
    visible: false,
    order: 53,
    width: 120,
    group: "部门与用户",
  },
  // 网络信息 (4列)
  { key: "machineIp", label: "加域IP", visible: true, order: 13, width: 120, group: "网络信息" },
  { key: "machineBs", label: "加域标识", visible: false, order: 63, width: 100, group: "网络信息" },
  { key: "mac1", label: "有线MAC", visible: true, order: 14, width: 140, group: "网络信息" },
  { key: "mac2", label: "无线MAC", visible: false, order: 64, width: 140, group: "网络信息" },
  // 使用状态 (1列)
  {
    key: "useStatusLabel",
    label: "使用状态",
    visible: true,
    order: 35,
    width: 100,
    group: "使用状态",
  },
  // 归属与责任 (8列)
  {
    key: "signOrgnoName",
    label: "归属机构",
    visible: true,
    order: 44,
    width: 150,
    group: "归属与责任",
  },
  {
    key: "orgnoName",
    label: "使用机构",
    visible: true,
    order: 58,
    width: 150,
    group: "归属与责任",
  },
  {
    key: "nowUserName",
    label: "责任人",
    visible: true,
    order: 45,
    width: 100,
    group: "归属与责任",
  },
  {
    key: "nowUserDeptCode",
    label: "部门编码",
    visible: true,
    order: 46,
    width: 120,
    group: "归属与责任",
  },
  {
    key: "deviceUserName",
    label: "领取人",
    visible: true,
    order: 47,
    width: 100,
    group: "归属与责任",
  },
  {
    key: "nowUserJobName",
    label: "责任人岗位",
    visible: false,
    order: 60,
    width: 120,
    group: "归属与责任",
  },
  { key: "outerUser", label: "使用人", visible: false, order: 59, width: 100, group: "归属与责任" },
  {
    key: "usingTypeName",
    label: "用途",
    visible: false,
    order: 61,
    width: 100,
    group: "归属与责任",
  },
  {
    key: "subUsingTypeName",
    label: "子用途",
    visible: false,
    order: 62,
    width: 100,
    group: "归属与责任",
  },
  {
    key: "userName",
    label: "APP扫码账号",
    visible: false,
    order: 66,
    width: 100,
    group: "归属与责任",
  },
  {
    key: "scanSite",
    label: "APP扫码地理位置",
    visible: false,
    order: 68,
    width: 200,
    group: "归属与责任",
  },
  // 状态与日期 (8列)
  { key: "status", label: "状态", visible: true, order: 48, width: 80, group: "状态与日期" },
  { key: "nbfStatus", label: "拟报废", visible: true, order: 49, width: 80, group: "状态与日期" },
  {
    key: "drawingDate",
    label: "接收日期",
    visible: true,
    order: 50,
    width: 120,
    group: "状态与日期",
  },
  { key: "useDate", label: "发放日期", visible: false, order: 56, width: 120, group: "状态与日期" },
  {
    key: "storageDatetime",
    label: "入库日期",
    visible: false,
    order: 57,
    width: 120,
    group: "状态与日期",
  },
  {
    key: "machineUptime",
    label: "最后上线",
    visible: true,
    order: 51,
    width: 150,
    group: "状态与日期",
  },
  {
    key: "machineUserId",
    label: "最后上线账号",
    visible: false,
    order: 65,
    width: 120,
    group: "状态与日期",
  },
  {
    key: "lastUpdateDate",
    label: "APP扫码时间",
    visible: false,
    order: 67,
    width: 150,
    group: "状态与日期",
  },
  {
    key: "y07UpdateTime",
    label: "Y07更新时间",
    visible: false,
    order: 69,
    width: 150,
    group: "状态与日期",
  },
  {
    key: "lastInventoryDate",
    label: "盘点日期",
    visible: true,
    order: 52,
    width: 120,
    group: "状态与日期",
  },
  // 标识与盘点 (3列)
  {
    key: "newFlagLabel",
    label: "新设备标识",
    visible: false,
    order: 70,
    width: 100,
    group: "状态与日期",
  },
  {
    key: "errorFlagName",
    label: "异常标识",
    visible: false,
    order: 71,
    width: 100,
    group: "状态与日期",
  },
  {
    key: "inventoryResult",
    label: "盘点结果",
    visible: false,
    order: 72,
    width: 100,
    group: "状态与日期",
  },
  // 其他 (1列)
  { key: "remark", label: "备注", visible: true, order: 43, width: 200, group: "其他" },
];
