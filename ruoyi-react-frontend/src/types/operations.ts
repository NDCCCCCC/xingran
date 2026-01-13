// 运维管理通用类型定义

// 楼宇
export interface Building {
  id: string;
  name: string;
  code: string;
  address: string;
  orgId: string;
  orgName?: string;
  status: BuildingStatus;
  remark?: string;
  orderNum: number;
  createdAt: string;
  updatedAt: string;
}

export type BuildingStatus = 0 | 1; // 0: 正常, 1: 停用

// 楼层
export interface Floor {
  id: string;
  name: string;
  floorNo: string;
  buildingId: string;
  buildingName?: string;
  status: FloorStatus;
  remark?: string;
  orderNum: number;
  createdAt: string;
  updatedAt: string;
}

export type FloorStatus = 0 | 1; // 0: 正常, 1: 停用

// 工位
export interface Workstation {
  id: string;
  code: string;
  name: string;
  floorId: string;
  floorName?: string;
  userId?: string;
  userName?: string;
  deptId?: string;
  deptName?: string;
  phonePoint?: string;
  status: WorkstationStatus;
  type: WorkstationType;
  equipment?: string;
  networkInfo?: string;
  remark?: string;
  positionX?: number;
  positionY?: number;
  createdAt: string;
  updatedAt: string;
}

export type WorkstationStatus = 0 | 1; // 0: 空闲, 1: 占用
export type WorkstationType = 0 | 1; // 0: 固定, 1: 共享

// 机房
export interface ServerRoom {
  id: string;
  name: string;
  code: string;
  floorId: string;
  floorName?: string;
  status: RoomStatus;
  remark?: string;
  photos?: RoomPhoto[];
  networkDevices?: NetworkDevice[];
  createdAt: string;
  updatedAt: string;
}

export type RoomStatus = 0 | 1; // 0: 正常, 1: 停用

export interface RoomPhoto {
  id: string;
  fileId: string;
  fileUrl: string;
  sortOrder: number;
  isPrimary: boolean;
  description?: string;
}

// 专线
export interface DedicatedLine {
  id: string;
  name: string;
  lineCode: string;
  lineType: DedicatedLineType;
  bandwidth: string;
  isp: string;
  sourceDeviceId: string;
  sourceDeviceName?: string;
  sourcePort: string;
  destDeviceId: string;
  destDeviceName?: string;
  destPort: string;
  ipAddress?: string;
  subnetMask?: string;
  gateway?: string;
  vlan?: string;
  status: LineStatus;
  activationDate?: string;
  expiryDate?: string;
  responsibleId?: string;
  responsibleName?: string;
  contractNo?: string;
  monthlyFee: number;
  remark?: string;
  createdAt: string;
  updatedAt: string;
}

export type DedicatedLineType = 'internet' | 'mpls' | 'leased_line' | 'fiber';
export type LineStatus = 0 | 1 | 2; // 0: 正常, 1: 故障, 2: 停用

// 机房设备
export interface RoomDevice {
  id: string;
  name: string;
  deviceCode: string;
  deviceType: RoomDeviceType;
  model?: string;
  vendor?: string;
  serialNumber?: string;
  roomId: string;
  roomName?: string;
  positionU: number;
  positionDesc?: string;
  assetNumber?: string;
  purchaseDate?: string;
  warrantyDate?: string;
  status: RoomDeviceStatus;
  responsibleId?: string;
  responsibleName?: string;
  powerConsumption?: number;
  remark?: string;
  createdAt: string;
  updatedAt: string;
}

export type RoomDeviceType = 'server' | 'storage' | 'ups' | 'pdu' | 'firewall' | 'kvm' | 'cabinet' | 'other';
export type RoomDeviceStatus = 0 | 1 | 2; // 0: 正常, 1: 故障, 2: 报废

// 网络设备（引用）
export interface NetworkDevice {
  id: string;
  name: string;
  ipAddress: string;
}

// 分页数据
export interface PageData<T> {
  list: T[];
  total: number;
  current: number;
  pageSize: number;
}

// 分页参数
export interface PageParams {
  current: number;
  pageSize: number;
  [key: string]: any;
}
