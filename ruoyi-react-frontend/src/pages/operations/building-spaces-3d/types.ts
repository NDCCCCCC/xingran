// 楼宇空间可视化类型定义

// 城市分组
export interface CityGroup {
  code: string; // 城市代码
  name: string; // 城市名称
  center: [number, number]; // 城市中心坐标 [lng, lat]
  buildings: BuildingItem[]; // 该城市的楼宇列表
  buildingCount: number; // 楼宇总数
}

// 楼宇项（用于地图标记）
export interface BuildingItem {
  id: string;
  name: string;
  code: string;
  cityCode: string;
  cityName: string;
  address: string;
  longitude?: number;
  latitude?: number;
  level: 1 | 2; // 层级：1=城市级汇总，2=具体楼宇
  status: number; // 0=正常, 1=停用
  floorCount?: number; // 楼层数
  workstationCount?: number; // 工位数
}

// 地图标记位置
export interface MarkerPosition {
  lng: number;
  lat: number;
}

// 地图标记数据
export interface MarkerData {
  id: string;
  type: "city" | "building";
  position: MarkerPosition;
  title: string;
  data: CityGroup | BuildingItem;
}
