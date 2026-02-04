/**
 * 百度地图 API 类型声明
 * 用于替代 any 类型
 */

/** 基础点坐标 */
export interface BMapPoint {
  lng: number;
  lat: number;
}

/** 像素点 */
export interface BMapPixel {
  x: number;
  y: number;
}

/** 边界结果 */
export interface BMapBoundaryResult {
  boundaries: string[];
}

/** 边界类 */
export interface BMapBoundaryConstructor {
  new (): BMapBoundary;
}

export interface BMapBoundary {
  get(name: string, callback: (result: BMapBoundaryResult) => void): void;
}

/** 地图类 */
export interface BMapMap {
  addOverlay(overlay: unknown): void;
  removeOverlay(overlay: unknown): void;
  clearOverlays(): void;
  setCenter(point: BMapPoint): void;
  setZoom(level: number): void;
  getCenter(): BMapPoint;
  pointToOverlayPixel(point: BMapPoint): BMapPixel;
  addControl(control: unknown): void;
  panTo(point: BMapPoint): void;
  enableScrollWheelZoom(): void;
  closeInfoWindow(): void;
  openInfoWindow(infoWindow: unknown): void;
}

/** 点构造函数 */
export interface BMapPointConstructor {
  new (lng: number, lat: number): BMapPoint;
}

/** 标记类 */
export interface BMapMarker {
  setPosition(position: BMapPoint): void;
  getPosition(): BMapPoint;
  setIcon(icon: unknown): void;
  setTitle(title: string): void;
  addEventListener(event: string, handler: () => void): void;
}

/** 信息窗口类 */
export interface BMapInfoWindow {
  setContent(content: string): void;
  open(map: BMapMap, position: BMapPoint): void;
  close(): void;
}

/** 多边形类 */
export interface BMapPolygon {
  setPath(path: BMapPoint[]): void;
  setFillColor(color: string): void;
  setFillOpacity(opacity: number): void;
  setStrokeColor(color: string): void;
  setStrokeOpacity(opacity: number): void;
  setStrokeWeight(weight: number): void;
}

/** 百度地图命名空间（BMap - 非 GL 版本） */
export interface BMapNamespace {
  Map: new (container: string | HTMLElement) => BMapMap;
  Point: BMapPointConstructor;
  Marker: new (position: BMapPoint, options?: Record<string, unknown>) => BMapMarker;
  InfoWindow: new (content?: string, options?: Record<string, unknown>) => BMapInfoWindow;
  Polygon: new (points: BMapPoint[], options?: Record<string, unknown>) => BMapPolygon;
  Boundary: BMapBoundaryConstructor;
}

/** 百度地图 GL 命名空间 */
export interface BMapGLNamespace extends BMapNamespace {
  Boundary: BMapBoundaryConstructor;
}

/** 全局 Window 扩展 */
declare global {
  interface Window {
    BMap?: BMapNamespace;
    BMapGL?: BMapGLNamespace;
    initBMapGL?: () => void;
    init?: () => void;
    viewBuildingDetails?: (buildingId: string) => void;
    viewBuildingDetailsGL?: (buildingId: string) => void;
  }
}

export {};
