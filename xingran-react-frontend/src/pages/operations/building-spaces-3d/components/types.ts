/**
 * 3D 楼宇视图组件共享类型定义
 */

// 楼宇数据接口
export interface BuildingItem {
  id: string;
  name: string;
  cityName?: string;
  address?: string;
  longitude?: number;
  latitude?: number;
  floorCount?: number;
  workstationCount?: number;
  status: number; // 0=正常, 1=停用
  level: number; // 1=一级(城市), 2=二级(具体楼宇)
}

// 楼层数据接口
export interface FloorData {
  id: string;
  name: string;
  code: string;
  floorNo: string;
  buildingName?: string;
  buildingId?: string;
  workstationCount: number; // 改为必需
  status: number; // 0=正常, 1=停用
}

// 工位数据接口
export interface WorkstationData {
  id: string;
  name: string;
  code: string;
  status: number; // 0=空闲, 1=占用, 2=维护
  type: number; // 0=固定, 1=灵活, 2=管理
  positionX?: number;
  positionY?: number;
  rotation?: number;
}

// 城市分组接口
export interface CityGroup {
  code: string;
  name: string;
  center: [number, number];
  buildingCount: number;
  buildings: BuildingItem[];
}

// 聚类群组接口
export interface ClusterGroup {
  buildings: BuildingItem[];
  centerPixel: { x: number; y: number };
  clusterLng: number;
  clusterLat: number;
}

// 工位统计接口
export interface WorkstationStats {
  total: number;
  available: number;
  occupied: number;
  flexible: number;
  fixed: number;
}

// 地址值接口
export interface AddressValue {
  cityCode: string;
  cityName: string;
  address: string;
  longitude?: number;
  latitude?: number;
}

// 地图类型
export type MapType = "normal" | "webgl";

// 视图级别
export type ViewLevel = "map" | "building" | "floor";
