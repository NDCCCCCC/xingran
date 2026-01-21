// 运维管理通用类型定义

// 楼宇
export interface Building {
  id: string;
  name: string;
  cityCode: string;        // 城市代码（如：wuhan, xiangyang）
  cityName: string;        // 城市名称（如：武汉市, 襄阳市）
  address: string;         // 详细地址
  longitude?: number;      // 经度
  latitude?: number;       // 纬度
  parentId?: string;       // 父级楼宇ID
  level: 1 | 2;            // 层级：1=城市级汇总，2=具体楼宇
  orgId: string;
  orgName?: string;
  totalFloors?: number;    // 楼层数（根据创建的楼层自动计算）
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
  buildingId: string;
  buildingName?: string;
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
  name: string;                // 专线名称
  lineType: string;            // 专线类型（数据字典）
  bandwidth?: string;
  isp: string;
  sourceDeviceId?: string;
  sourceDeviceName?: string;
  sourcePort?: string;
  sourceRoomId?: string;       // 源机房ID
  sourceRoomName?: string;     // 源机房名称
  destDeviceId?: string;
  destDeviceName?: string;
  destPort?: string;
  destRoomId?: string;         // 目的机房ID
  destRoomName?: string;       // 目的机房名称
  ipAddress?: string;
  subnetMask?: string;
  vlan?: string;
  status: LineStatus;
  carrierContactId?: string;
  carrierContactName?: string;
  carrierContactPhone?: string;
  contractNo?: string;
  monthlyFee: number;
  remark?: string;
  createdAt: string;
  updatedAt: string;
}

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

// 信息点
export interface InfoPoint {
  id: string;
  name: string;
  infoPointType: InfoPointType;
  workstationId: string;
  workstationName?: string;
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

export type InfoPointType = 'network' | 'power' | 'other';
export type InfoPointStatus = 0 | 1 | 2; // 0: 正常, 1: 故障, 2: 停用

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
