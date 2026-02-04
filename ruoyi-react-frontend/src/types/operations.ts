/**
 * 运维管理相关类型
 */

import type { Status } from './base';

// PageParams 已在 base.ts 中定义，此处不再重复导出

// ==================== 通用分页类型 ====================

/**
 * 分页数据
 */
export interface PageData<T> {
  list: T[];
  total: number;
  current: number;
  pageSize: number;
}

// ==================== 楼宇相关 ====================

/**
 * 楼宇状态
 */
export type BuildingStatus = Status;

/**
 * 楼宇类型
 */
export interface Building {
  id: string;
  orgId: string;  // 所属机构ID
  name: string;
  code: string;
  address?: string;
  totalFloors?: number;
  totalArea?: number;
  description?: string;
  longitude?: number;
  latitude?: number;
  status: BuildingStatus;
  createdAt: string;
  updatedAt: string;
}

// ==================== 楼层相关 ====================

/**
 * 楼层状态
 */
export type FloorStatus = Status;

/**
 * 楼层类型
 */
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
  createdAt: string;
  updatedAt: string;
}

// ==================== 工位相关 ====================

/**
 * 工位类型：0=固定工位，1=灵活工位，2=管理工位
 */
export type WorkstationOpsType = 0 | 1 | 2;

/**
 * 工位状态：0=空闲可分配，1=占用已分配，2=维护中不可用
 */
export type WorkstationOpsStatus = 0 | 1 | 2;

/**
 * 工位（运维管理）
 */
export interface WorkstationOps {
  id: string;
  floorId: string;
  floorName?: string;
  name: string;
  type: WorkstationOpsType;
  status: WorkstationOpsStatus;
  deptId?: string;
  deptName?: string;
  userId?: string;
  userName?: string;
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

// ==================== 机房相关 ====================

/**
 * 机房状态
 */
export type RoomStatus = Status;

/**
 * 机房类型
 */
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
  createdAt: string;
  updatedAt: string;
}

/**
 * 机房照片
 */
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

// ==================== 机房设备相关 ====================

/**
 * 设备类型
 */
export type RoomDeviceType =
  | 'server'
  | 'storage'
  | 'ups'
  | 'pdu'
  | 'firewall'
  | 'kvm'
  | 'cabinet'
  | 'other';

/**
 * 设备状态：0=正常，1=故障，2=报废
 */
export type RoomDeviceStatus = 0 | 1 | 2;

/**
 * 机房设备
 */
export interface RoomDevice {
  id: string;
  roomCode: string;
  roomName?: string;
  deviceCode: string;
  deviceName: string;
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
  description?: string;
  createdAt: string;
  updatedAt: string;
}

// ==================== 专线相关 ====================

/**
 * 专线状态：0=正常，1=故障，2=停用
 */
export type LineStatus = 0 | 1 | 2;

/**
 * 专线类型
 */
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

// ==================== 信息点相关 ====================

/**
 * 信息点类型
 */
export type InfoPointType = string;

/**
 * 信息点状态：0=正常，1=故障，2=停用
 */
export type InfoPointStatus = 0 | 1 | 2;

/**
 * 信息点
 */
export interface InfoPoint {
  id: string;
  workstationId: string;
  workstationName?: string;
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
