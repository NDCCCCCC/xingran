/**
 * 运维管理相关类型
 */

import type { Status, PageResponse, PageParams } from "./base";

// 统一使用 base.ts 中的 PageResponse 类型
export type PageData<T> = PageResponse<T>;

export type BuildingStatus = Status;

export interface Building {
  id: string;
  orgId: string;
  orgName?: string;
  name: string;
  code: string;
  address?: string;
  totalFloors?: number;
  totalArea?: number;
  description?: string;
  longitude?: number;
  latitude?: number;
  level?: number;
  remark?: string;
  status: BuildingStatus;
  createdAt: string;
  updatedAt: string;
}

export type FloorStatus = Status;

export interface Floor {
  id: string;
  buildingId: string;
  buildingCode: string;
  code: string;
  buildingName?: string;
  floorNo: string;
  name?: string;
  area?: number;
  description?: string;
  status: FloorStatus;
  planImageId?: string;
  planImageUrl?: string;
  createdAt: string;
  updatedAt: string;
}

export type WorkstationOpsType = 0 | 1 | 2;

export type WorkstationOpsStatus = 0 | 1 | 2;

export interface WorkstationOps {
  id: string;
  floorId: string;
  floorName?: string;
  floorCode?: string; // 楼层编号
  buildingId?: string; // 楼宇ID
  buildingName?: string; // 楼宇名称
  name: string;
  type: WorkstationOpsType;
  status: WorkstationOpsStatus;
  deptId?: string;
  deptName?: string;
  orgId?: string; // 所属机构ID（来自 building.get 的 orgId，编辑表单回显用）
  userId?: string;
  userName?: string;
  primaryDeviceSerial?: string; // 主设备序列号（设置了主设备时由后端子查询返回）
  positionX?: number;
  positionY?: number;
  rotation?: number;
  width?: number;
  depth?: number;
  deskType?: number;
  description?: string;
  createdAt: string;
  updatedAt: string;
}

export type RoomStatus = Status;

export interface ServerRoom {
  id: string;
  buildingId: string;
  buildingName?: string;
  floorId: string;
  floorName?: string;
  floorNo?: string;
  name: string;
  area?: number;
  rackCount?: number;
  powerCapacity?: number;
  remark?: string;
  status: RoomStatus;
  orgId?: string;
  createdAt: string;
  updatedAt: string;
}

export interface RoomPhoto {
  id: string;
  roomId: string;
  fileId: string;
  fileName?: string;
  fileUrl: string;
  sortOrder: number;
  isPrimary: boolean;
  description?: string;
  createdAt: string;
}

export type RoomDeviceType = string;

export type RoomDeviceStatus = 0 | 1 | 2;

export interface RoomDevice {
  id: string;
  roomId: string;
  roomName?: string;
  deviceCode: string;
  name: string;
  deviceType: RoomDeviceType;
  vendor?: string;
  model?: string;
  serialNumber?: string;
  positionDesc?: string;
  positionU?: number;
  heightU?: number;
  powerConsumption?: number;
  purchaseDate?: string;
  warrantyDate?: string;
  status: RoomDeviceStatus;
  responsibleId?: string;
  responsibleName?: string;
  remark?: string;
  createdAt: string;
  updatedAt: string;
}

export type LineStatus = 0 | 1 | 2;

export interface DedicatedLine {
  id: string;
  name: string;
  lineType: string;
  bandwidth?: string;
  isp: string;
  sourceDeviceId?: string;
  sourceDeviceName?: string;
  sourcePort?: string;
  sourceRoomId?: string;
  sourceRoomName?: string;
  destDeviceId?: string;
  destDeviceName?: string;
  destPort?: string;
  destRoomId?: string;
  destRoomName?: string;
  sourceIpAddress?: string;
  sourceSubnetMask?: string;
  destIpAddress?: string;
  destSubnetMask?: string;
  vlan?: string;
  carrierContactName?: string;
  carrierContactPhone?: string;
  monthlyFee?: number;
  status: LineStatus;
  remark?: string;
  createdAt: string;
  updatedAt: string;
}

export type InfoPointType = string;

export type InfoPointStatus = 0 | 1 | 2;

export interface InfoPoint {
  id: string;
  workstationId: string;
  workstationName?: string;
  floorName?: string; // 楼层名称
  buildingId?: string; // 楼宇ID
  buildingName?: string; // 楼宇名称
  name: string;
  infoPointType: InfoPointType;
  deviceId?: string;
  deviceName?: string;
  portId?: string;
  portName?: string;
  status: InfoPointStatus;
  description?: string;
  remark?: string;
  createdAt: string;
  updatedAt: string;
}

// ==================== 资产管理 ====================

export type AssetStatus = 0 | 1;

export interface Asset {
  id: string;
  createdAt: string;
  updatedAt: string;
  deletedAt?: string;

  // 核心标识
  devicesn: string; // 设备序列号
  sequenceNo?: string; // 序列号
  fixAssetNo?: string; // 固定资产编号

  // 设备信息
  deviceModelName?: string; // 型号
  deviceTypeName?: string; // 类型
  deviceCategorySecondName?: string; // 中类
  deviceBasicTypeName?: string; // 是否固定资产

  // 用户关联
  deviceUserName?: string; // 领取人
  nowUserName?: string; // 责任人
  nowUserP13?: string; // 责任人p13
  deviceUserP13?: string; // 领取人p13

  // 部门关联
  deptName?: string; // 受益部门
  nowUserDeptCode?: string; // 部门编码
  xnDeptCode?: string; // 受益部门编码

  // 状态标识
  useStatusLabel?: string; // 状态
  newFlagLabel?: string; // 新设备标识
  printFlagName?: string; // 打印状态
  nbfStatus?: number; // 是否拟报废 (0=否, 1=是)

  // 时间字段
  drawingDate?: string; // 接收日期
  useDate?: string; // 发放日期
  storageDatetime?: string; // 入库日期
  lastUpdateDate?: string; // APP扫码时间
  y07UpdateTime?: string; // Y07更新时间
  machineUptime?: string; // 最后上线时间
  lastInventoryDate?: string; // 最近一次盘点时间

  // 网络信息
  mac1?: string; // 有线MAC
  mac2?: string; // 无线MAC
  machineIp?: string; // 加域IP
  machineBs?: string; // 加域标识

  // 合同与属性
  contractNo?: string; // 合同号
  attributeValue?: string; // 设备属性

  // 位置与归属
  scanSite?: string; // APP扫码地理位置
  remark?: string; // 备注
  qudaoName?: string; // 设备渠道
  usingTypeName?: string; // 用途
  subUsingTypeName?: string; // 子用途
  orgnoName?: string; // 使用机构
  storeroomName?: string; // 库房
  storageAddress?: string; // 存放地址

  // 机构与标准
  signOrgnoName?: string; // 归属机构
  isNoStandardName?: string; // 申请标准名称
  errorFlagName?: string; // 异常标识

  // 外部与部门用户
  outerUser?: string; // 使用人
  usefulDeptName?: string; // 部门名称
  nowUserJobName?: string; // 责任人岗位
  userName?: string; // APP扫码账号
  machineUserId?: string; // 最后上线账号
  inventoryResult?: string; // 盘点结果

  // 系统关联
  deptId?: string; // 关联 sys_dept.id
  userId?: string; // 关联 sys_user.id

  // 状态
  status: AssetStatus; // 0=正常, 1=停用

  // Phase 48 组件序列号采集(D-01 / D-03 / D-05):板卡/引擎/电源/风扇/光模块作为 ops_asset 行,
  // 通过以下 4 列与父交换机/采集来源建立关联。主设备这些列保持 NULL(D-07 默认 component_type IS NULL 过滤)。
  parentAssetId?: string; // 自引用 → ops_asset.id(父交换机/路由器行)
  sourceDeviceId?: string; // → sys_network_device.id(采集来源设备)
  componentType?: string; // chassis / card / engine / power / fan / transceiver
  componentSlot?: string; // 槽位/接口位置(如 Slot 1 / GE1/0/24)
}

export interface AssetListParams extends PageParams {
  devicesn?: string;
  deviceModelName?: string;
  deptId?: string;
  userId?: string;
  status?: number;
  nbfStatus?: number;
}

// ==================== 工位设备关联 ====================

export type DeviceSource = "ad" | "asset" | "manual" | "physical";

export type OpsDeviceStatus = 0 | 1;

export interface WorkstationDevice {
  id: string;
  workstationId: string;
  deviceSource: DeviceSource;
  deviceSerial?: string;
  deviceName?: string;
  deviceModel?: string;
  deviceType?: string;
  macAddress?: string;
  ipAddress?: string;
  assetId?: string;
  adComputerId?: string;
  responsibleUser?: string;
  responsibleUserId?: string;
  status: OpsDeviceStatus;
  isPrimary: boolean;
  priority: number;
  description?: string;
  createdAt: string;
  updatedAt: string;
  // R5 物理链路(2026-07-02): 置信度 + 历史最后上线时间
  //   confidence: 1.0 = 实测 MAC 命中, 0.5 = 仅历史 MAC 命中(设备离线), undefined = 非物理链路/未分级
  //   historyLastSeen: 仅当 confidence < 1.0 且来自历史 MAC 表时有值
  confidence?: number;
  historyLastSeen?: string;
  // 关联对象
  asset?: Asset;
}

export const DeviceSourceLabels: Record<DeviceSource, string> = {
  ad: "域控",
  asset: "资产",
  manual: "手动",
  physical: "物理链路",
};

// 别名导出，保持命名一致性
export const DEVICE_SOURCE_LABELS = DeviceSourceLabels;

export interface DeviceFormData {
  workstationId: string;
  deviceSerial: string;
  deviceName?: string;
  deviceModel?: string;
  deviceType?: string;
  macAddress?: string;
  ipAddress?: string;
  responsibleUser?: string;
  description?: string;
}
