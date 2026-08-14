// 楼宇空间可视化辅助函数

import type { BuildingItem } from "./types";
import { WORKSTATION_STATUS_COLORS } from "./constants";
import { getBMapGL, type BMapGLNamespace, type BMapPoint } from "@/types/baidu-map";

// ============ 通用辅助函数 ============

/**
 * 正确处理包含 Unicode 字符的 base64 编码
 * @param str 要编码的字符串
 * @returns Base64 编码后的字符串
 */
export function toBase64(str: string): string {
  try {
    return btoa(unescape(encodeURIComponent(str)));
  } catch {
    return btoa(str);
  }
}

/**
 * 延迟执行
 * @param ms 延迟毫秒数
 * @returns Promise
 */
export function delay(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

// ============ 地图相关辅助函数 ============

/**
 * 获取湖北省官方边界（使用百度地图 Boundary API）
 * @param BMapGL 百度地图 GL 构造函数
 * @returns Promise<string[]> 边界坐标数组
 */
export function getOfficialHubeiBoundary(BMapGL: BMapGLNamespace): Promise<string[]> {
  return new Promise((resolve, reject) => {
    const bdary = new BMapGL.Boundary();
    bdary.get("湖北省", (rs: { boundaries?: string[] }) => {
      if (rs.boundaries && rs.boundaries.length > 0) {
        resolve(rs.boundaries);
      } else {
        reject(new Error("未获取到边界数据"));
      }
    });
  });
}

/**
 * 将边界字符串转换为 BMapGL.Point 数组
 * @param boundaryStr 边界字符串（格式："lng,lat;lng,lat;..."）
 * @param BMapGL 百度地图 GL 构造函数
 * @returns BMapGL.Point 数组
 */
export function parseBoundaryPoints(boundaryStr: string, BMapGL: BMapGLNamespace): BMapPoint[] {
  return boundaryStr.split(";").map((point: string) => {
    const [lng, lat] = point.split(",").map(Number);
    return new BMapGL.Point(lng, lat);
  });
}

/**
 * 计算两点之间的像素距离
 * @param point1 第一个点的像素坐标
 * @param point2 第二个点的像素坐标
 * @returns 像素距离
 */
export function pixelDistance(
  point1: { x: number; y: number },
  point2: { x: number; y: number }
): number {
  return Math.sqrt(Math.pow(point1.x - point2.x, 2) + Math.pow(point1.y - point2.y, 2));
}

/**
 * 计算多个点的平均像素坐标
 * @param pixels 像素坐标数组
 * @returns 平均像素坐标
 */
export function averagePixelPosition(pixels: Array<{ x: number; y: number }>): {
  x: number;
  y: number;
} {
  const sumX = pixels.reduce((sum, p) => sum + p.x, 0);
  const sumY = pixels.reduce((sum, p) => sum + p.y, 0);
  return {
    x: sumX / pixels.length,
    y: sumY / pixels.length,
  };
}

// ============ 楼宇相关辅助函数 ============

/**
 * 判断楼宇是否停用
 * @param building 楼宇数据
 * @returns 是否停用
 */
export function isBuildingStopped(building: BuildingItem): boolean {
  return building.status === 1;
}

/**
 * 获取楼宇状态文本
 * @param status 状态值（0=正常, 1=停用）
 * @returns 状态文本
 */
export function getBuildingStatusText(status: number): string {
  return status === 1 ? "已停用" : "正常";
}

/**
 * 获取楼宇状态颜色
 * @param status 状态值（0=正常, 1=停用）
 * @returns Ant Design Tag 颜色
 */
export function getBuildingStatusColor(status: number): string {
  return status === 1 ? "red" : "green";
}

/**
 * 生成楼宇标记标签文本（取前两个字）
 * @param building 楼宇数据
 * @returns 标签文本
 */
export function getBuildingLabel(building: BuildingItem): string {
  return building.name ? building.name.slice(0, 2) : "楼宇";
}

/**
 * 过滤楼宇数据（根据缩放级别）
 * @param buildings 所有楼宇数据
 * @param zoom 当前缩放级别
 * @returns 过滤后的楼宇数据
 */
export function filterBuildingsByZoom(buildings: BuildingItem[], zoom: number): BuildingItem[] {
  return zoom === 10 ? buildings : buildings.filter((b) => b.level === 1);
}

// ============ 工位相关辅助函数 ============

/**
 * 获取工位状态文本
 * @param status 状态值（0=空闲, 1=占用, 2=维护）
 * @returns 状态文本
 */
export function getWorkstationStatusText(status: number): string {
  switch (status) {
    case 0:
      return "空闲";
    case 1:
      return "占用";
    case 2:
      return "维护";
    default:
      return "未知";
  }
}

/**
 * 获取工位类型文本
 * @param type 类型值（0=固定, 1=灵活, 2=管理）
 * @returns 类型文本
 */
export function getWorkstationTypeText(type: number): string {
  switch (type) {
    case 0:
      return "固定";
    case 1:
      return "灵活";
    case 2:
      return "管理";
    default:
      return "未知";
  }
}

/**
 * 获取工位状态颜色（十六进制）
 * @param workstation 工位数据
 * @returns 颜色值（十六进制数字）
 */
export function getWorkstationColor(workstation: { status: number; type: number }): number {
  if (workstation.status === 1) {
    return WORKSTATION_STATUS_COLORS.OCCUPIED;
  }
  if (workstation.status === 2) {
    return WORKSTATION_STATUS_COLORS.MAINTENANCE;
  }
  if (workstation.type === 1) {
    return WORKSTATION_STATUS_COLORS.FLEXIBLE;
  }
  if (workstation.type === 2) {
    return WORKSTATION_STATUS_COLORS.MANAGERIAL;
  }
  return WORKSTATION_STATUS_COLORS.AVAILABLE;
}

/**
 * 获取工位状态颜色（CSS 格式）
 * @param status 状态值（0=空闲, 1=占用, 2=维护）
 * @returns CSS 颜色值
 */
export function getWorkstationStatusColorCSS(status: number): string {
  switch (status) {
    case 0:
      return "var(--theme-success, #52c41a)";
    case 1:
      return "#ff4d4f";
    case 2:
      return "var(--theme-warning, #faad14)";
    default:
      return "#d9d9d9";
  }
}

/**
 * 获取工位类型颜色（CSS 格式）
 * @param type 类型值（0=固定, 1=灵活, 2=管理）
 * @returns CSS 颜色值
 */
export function getWorkstationTypeColorCSS(type: number): string {
  switch (type) {
    case 0:
      return "var(--theme-info, #1890ff)";
    case 1:
      return "var(--theme-purple, #722ed1)";
    case 2:
      return "#13c2c2";
    default:
      return "#d9d9d9";
  }
}

/**
 * 计算工位位置（自动排列）
 * @param workstations 工位数据数组
 * @param positionScale 预设位置转换系数
 * @param positionOffset 预设位置偏移量
 * @returns 工位位置映射
 */
export function calculateWorkstationPositions(
  workstations: Array<{ id: string; positionX?: number; positionY?: number }>,
  positionScale = 10,
  positionOffset = 50
): Map<string, { x: number; z: number }> {
  const positionMap = new Map<string, { x: number; z: number }>();
  const { GRID_SIZE, CELL_SIZE } = getWorkstationLayoutConfig();

  workstations.forEach((ws, index) => {
    if (ws.positionX !== undefined && ws.positionY !== undefined) {
      // 使用预设位置
      positionMap.set(ws.id, {
        x: (ws.positionX - positionOffset) / positionScale,
        z: (ws.positionY - positionOffset) / positionScale,
      });
    } else {
      // 自动排列：从左到右，从上到下
      const row = Math.floor(index / GRID_SIZE);
      const col = index % GRID_SIZE;
      positionMap.set(ws.id, {
        x: (col - GRID_SIZE / 2) * CELL_SIZE,
        z: (row - GRID_SIZE / 2) * CELL_SIZE,
      });
    }
  });

  return positionMap;
}

/**
 * 获取工位布局配置
 * @returns 布局配置对象
 */
function getWorkstationLayoutConfig() {
  return {
    GRID_SIZE: 8,
    CELL_SIZE: 1.5,
  };
}

/**
 * 转换 API 工位数据为内部格式
 * @param apiWorkstation API 返回的工位数据
 * @returns 转换后的工位数据
 */
export function convertApiWorkstation(apiWorkstation: {
  id: string;
  name: string;
  status: number;
  type: number;
  positionX?: number;
  positionY?: number;
}): {
  id: string;
  name: string;
  code: string;
  status: number;
  type: number;
  positionX?: number;
  positionY?: number;
  rotation: number;
} {
  return {
    id: apiWorkstation.id,
    name: apiWorkstation.name,
    code: apiWorkstation.name, // 使用 name 作为 code
    status: apiWorkstation.status,
    type: apiWorkstation.type,
    positionX: apiWorkstation.positionX,
    positionY: apiWorkstation.positionY,
    rotation: 0, // 默认旋转角度
  };
}

/**
 * 批量转换 API 工位数据
 * @param apiWorkstations API 返回的工位数据数组
 * @returns 转换后的工位数据数组
 */
export function convertApiWorkstations(
  apiWorkstations: Array<{
    id: string;
    name: string;
    status: number;
    type: number;
    positionX?: number;
    positionY?: number;
  }>
): Array<{
  id: string;
  name: string;
  code: string;
  status: number;
  type: number;
  positionX?: number;
  positionY?: number;
  rotation: number;
}> {
  return apiWorkstations.map(convertApiWorkstation);
}

/**
 * 计算工位统计数据
 * @param workstations 工位数据数组
 * @returns 统计数据
 */
export function calculateWorkstationStats(
  workstations: Array<{
    status: number;
    type: number;
  }>
): {
  total: number;
  available: number;
  occupied: number;
  flexible: number;
  fixed: number;
} {
  return {
    total: workstations.length,
    available: workstations.filter((w) => w.status === 0).length,
    occupied: workstations.filter((w) => w.status === 1).length,
    flexible: workstations.filter((w) => w.type === 1).length,
    fixed: workstations.filter((w) => w.type === 0).length,
  };
}

// ============ 3D 场景相关辅助函数 ============

/**
 * 将角度转换为弧度
 * @param degrees 角度
 * @returns 弧度
 */
export function degreesToRadians(degrees: number): number {
  return degrees * (Math.PI / 180);
}

/**
 * 线性插值
 * @param start 起始值
 * @param end 结束值
 * @param t 插值因子（0-1）
 * @returns 插值结果
 */
export function lerp(start: number, end: number, t: number): number {
  return start + (end - start) * t;
}

// ============ 动画相关辅助函数 ============

/**
 * 执行缓动动画
 * @param callback 每帧回调函数，接收当前进度作为参数
 * @param duration 动画时长（毫秒）
 * @param easingFn 缓动函数
 * @returns Promise
 */
export function animate(
  callback: (progress: number, easedProgress: number) => void,
  duration: number,
  easingFn: (t: number) => number
): Promise<void> {
  return new Promise((resolve) => {
    const startTime = Date.now();

    function animate() {
      const now = Date.now();
      const elapsed = now - startTime;
      const progress = Math.min(elapsed / duration, 1);
      const eased = easingFn(progress);

      callback(progress, eased);

      if (progress < 1) {
        requestAnimationFrame(animate);
      } else {
        resolve();
      }
    }

    animate();
  });
}

/**
 * 动画过渡地图视图
 * @param map 地图实例
 * @param target 目标状态
 * @param duration 动画时长（毫秒）
 * @param easingFn 缓动函数
 */
export function animateViewTransition(
  map: {
    getCenter(): { lng: number; lat: number };
    getZoom(): number;
    getTilt(): number;
    getHeading(): number;
    setCenter(center: BMapPoint): void;
    setZoom(zoom: number): void;
    setTilt(tilt: number): void;
    setHeading(heading: number): void;
  },
  target: {
    center?: [number, number];
    zoom?: number;
    tilt?: number;
    heading?: number;
  },
  duration = 1500,
  easingFn: (t: number) => number
): void {
  const startTime = Date.now();
  const startState = {
    center: map.getCenter(),
    zoom: map.getZoom(),
    tilt: map.getTilt(),
    heading: map.getHeading(),
  };

  function animate() {
    const now = Date.now();
    const elapsed = now - startTime;
    const progress = Math.min(elapsed / duration, 1);
    const eased = easingFn(progress);

    // 插值计算当前状态
    if (target.center) {
      const lng = startState.center.lng + (target.center[0] - startState.center.lng) * eased;
      const lat = startState.center.lat + (target.center[1] - startState.center.lat) * eased;
      const BMapGL = getBMapGL();
      if (BMapGL) {
        map.setCenter(new BMapGL.Point(lng, lat));
      }
    }

    if (target.zoom !== undefined) {
      const zoom = startState.zoom + (target.zoom - startState.zoom) * eased;
      map.setZoom(zoom);
    }

    if (target.tilt !== undefined) {
      const tilt = startState.tilt + (target.tilt - startState.tilt) * eased;
      map.setTilt(tilt);
    }

    if (target.heading !== undefined) {
      const heading = startState.heading + (target.heading - startState.heading) * eased;
      map.setHeading(heading);
    }

    if (progress < 1) {
      requestAnimationFrame(animate);
    }
  }

  animate();
}
