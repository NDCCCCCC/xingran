/**
 * 运维管理相关类型
 */

import type { Status, PageResponse } from './base';

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
  floorCode?: string;         // 楼层编号
  buildingId?: string;         // 楼宇ID
  buildingName?: string;       // 楼宇名称
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

export type RoomDeviceType =
  | 'server'
  | 'storage'
  | 'ups'
  | 'pdu'
  | 'firewall'
  | 'kvm'
  | 'cabinet'
  | 'other';

export type RoomDeviceStatus = 0 | 1 | 2;

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
  floorName?: string;        // 楼层名称
  buildingId?: string;       // 楼宇ID
  buildingName?: string;       // 楼宇名称
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
