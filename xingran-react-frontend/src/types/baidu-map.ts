/**
 * 百度地图 JavaScript API 类型声明
 * 支持 BMap（标准版）和 BMapGL（GL 版本）
 */

// ============ 基础类型 ============

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

/** 尺寸 */
export interface BMapSize {
  width: number;
  height: number;
}

/** 地图边界 */
export interface BMapBounds {
  sw: BMapPoint;
  ne: BMapPoint;
  containsPoint(point: BMapPoint): boolean;
  getCenter(): BMapPoint;
}

/** 地图配置选项 */
export interface BMapMapOptions {
  enableMapClick?: boolean;
  minZoom?: number;
  maxZoom?: number;
}

// ============ 事件系统 ============

/** 事件对象 */
export interface BMapEvent {
  type: string;
  target: unknown;
  point?: BMapPoint;
  pixel?: BMapPixel;
  [key: string]: unknown;
}

/** 事件监听器 */
export interface BMapEventListener {
  remove(): void;
}

/** 事件发射器接口 */
export interface BMapEventEmitter {
  addEventListener(event: string, handler: (e: BMapEvent) => void): BMapEventListener;
  removeEventListener(event: string, handler: (e: BMapEvent) => void): void;
  dispatchEvent(event: string, e?: BMapEvent): void;
}

// ============ 覆盖物类型 ============

/** 图标 */
export interface BMapIcon {
  url: string;
  size: BMapSize;
  anchor?: BMapSize;
  imageOffset?: BMapSize;
  imageSize?: BMapSize;
}

/** 标记选项 */
export interface BMapMarkerOptions {
  icon?: BMapIcon;
  offset?: BMapSize;
  enableMassClear?: boolean;
  enableDragging?: boolean;
  title?: string;
  rotation?: number;
  shadow?: BMapIcon;
}

/** 标记类 */
export interface BMapMarker extends BMapEventEmitter {
  setPosition(position: BMapPoint): void;
  getPosition(): BMapPoint;
  setIcon(icon: BMapIcon): void;
  setTitle(title: string): void;
  setRotation(rotation: number): void;
  enableDragging(): void;
  disableDragging(): void;
  show(): void;
  hide(): void;
  isVisible(): boolean;
  setAnimation(animation: number): void;
  setTop(isTop: boolean): void;
}

/** 信息窗口选项 */
export interface BMapInfoWindowOptions {
  width?: number;
  height?: number;
  maxWidth?: number;
  offset?: BMapSize;
  title?: string;
  enableAutoPan?: boolean;
  enableCloseOnClick?: boolean;
}

/** 信息窗口类 */
export interface BMapInfoWindow extends BMapEventEmitter {
  setContent(content: string | HTMLElement): void;
  setPosition(position: BMapPoint): void;
  getPosition(): BMapPoint;
  open(map: BMapMap, position: BMapPoint): void;
  close(): void;
  enableAutoPan(): void;
  disableAutoPan(): void;
  setTitle(title: string): void;
  getTitle(): string;
  setWidth(width: number): void;
  setHeight(height: number): void;
}

/** 多边形选项 */
export interface BMapPolygonOptions {
  strokeColor?: string;
  strokeWeight?: number;
  strokeOpacity?: number;
  strokeStyle?: "solid" | "dashed";
  fillColor?: string;
  fillOpacity?: number;
  enableMassClear?: boolean;
  enableEditing?: boolean;
}

/** 多边形类 */
export interface BMapPolygon extends BMapEventEmitter {
  setPath(path: BMapPoint[]): void;
  getPath(): BMapPoint[];
  setStrokeColor(color: string): void;
  setStrokeWeight(weight: number): void;
  setStrokeOpacity(opacity: number): void;
  setStrokeStyle(style: "solid" | "dashed"): void;
  setFillColor(color: string): void;
  setFillOpacity(opacity: number): void;
  show(): void;
  hide(): void;
  isVisible(): boolean;
}

/** 标签选项 */
export interface BMapLabelOptions {
  position?: BMapPoint;
  offset?: BMapSize;
}

/** 标签类 */
export interface BMapLabel extends BMapEventEmitter {
  setContent(content: string): void;
  setPosition(position: BMapPoint): void;
  getPosition(): BMapPoint;
  setStyle(style: Record<string, string>): void;
  setTitle(title: string): void;
  show(): void;
  hide(): void;
}

// ============ 地图类 ============

/** 地图类 */
export interface BMapMap extends BMapEventEmitter {
  // 视图控制
  centerAndZoom(center: BMapPoint, zoom: number): void;
  setCenter(point: BMapPoint): void;
  getCenter(): BMapPoint;
  setZoom(level: number): void;
  getZoom(): number;
  setMinZoom(minZoom: number): void;
  setMaxZoom(maxZoom: number): void;
  panTo(point: BMapPoint, opts?: { noAnimation?: boolean }): void;
  panBy(x: number, y: number): void;

  // 坐标转换
  pointToPixel(point: BMapPoint): BMapPixel;
  pixelToPoint(pixel: BMapPixel): BMapPoint;
  pointToOverlayPixel(point: BMapPoint): BMapPixel;
  getBounds(): BMapBounds;
  getSize(): BMapSize;

  // 覆盖物管理
  addOverlay(overlay: unknown): void;
  removeOverlay(overlay: unknown): void;
  clearOverlays(): void;

  // 控件管理
  addControl(control: unknown): void;
  removeControl(control: unknown): void;

  // 交互
  enableScrollWheelZoom(enabled?: boolean): void;
  disableScrollWheelZoom(): void;
  enableDragging(): void;
  disableDragging(): void;
  enableDoubleClickZoom(enabled?: boolean): void;
  disableDoubleClickZoom(): void;
  enableKeyboard(): void;
  disableKeyboard(): void;
  enableInertialDragging(): void;
  disableInertialDragging(): void;
  enableContinuousZoom(): void;
  disableContinuousZoom(): void;
  enablePinchToZoom(): void;
  disablePinchToZoom(): void;

  // 信息窗口
  openInfoWindow(infoWindow: BMapInfoWindow, point: BMapPoint): void;
  closeInfoWindow(): void;
}

/** 地图 GL 类（扩展标准地图，增加 3D 视角控制） */
export interface BMapMapGL extends BMapMap {
  // 3D 视角控制
  getTilt(): number;
  setTilt(tilt: number): void;
  getHeading(): number;
  setHeading(heading: number): void;
  setMapStyleV2(config: { styleId: string }): void;
}

// ============ 边界查询 ============

/** 边界结果 */
export interface BMapBoundaryResult {
  boundaries: string[];
}

/** 边界类 */
export interface BMapBoundary {
  get(name: string, callback: (result: BMapBoundaryResult) => void): void;
}

// ============ 构造函数 ============

/** 点构造函数 */
export interface BMapPointConstructor {
  new (lng: number, lat: number): BMapPoint;
}

/** 尺寸构造函数 */
export interface BMapSizeConstructor {
  new (width: number, height: number): BMapSize;
}

/** 图标构造函数 */
export interface BMapIconConstructor {
  new (url: string, size: BMapSize, opts?: BMapSize): BMapIcon;
}

/** 标记构造函数 */
export interface BMapMarkerConstructor {
  new (position: BMapPoint, opts?: BMapMarkerOptions): BMapMarker;
}

/** 信息窗口构造函数 */
export interface BMapInfoWindowConstructor {
  new (content?: string | HTMLElement, opts?: BMapInfoWindowOptions): BMapInfoWindow;
}

/** 多边形构造函数 */
export interface BMapPolygonConstructor {
  new (points: BMapPoint[], opts?: BMapPolygonOptions): BMapPolygon;
}

/** 标签构造函数 */
export interface BMapLabelConstructor {
  new (content?: string, opts?: BMapLabelOptions): BMapLabel;
}

/** 边界构造函数 */
export interface BMapBoundaryConstructor {
  new (): BMapBoundary;
}

/** 地图构造函数 */
export interface BMapMapConstructor {
  new (container: string | HTMLElement, opts?: BMapMapOptions): BMapMap;
}

/** 地图 GL 构造函数 */
export interface BMapMapGLConstructor {
  new (
    container: string | HTMLElement,
    opts?: BMapMapOptions & { showControls?: boolean }
  ): BMapMapGL;
}

/** 缩放控件配置 */
export interface BMapZoomControlOptions {
  anchor: number;
  offset: BMapSize;
}

/** 比例尺控件配置 */
export interface BMapScaleControlOptions {
  anchor: number;
  offset: BMapSize;
}

/** 导航控件配置 */
export interface BMapNavigationControlOptions {
  anchor?: number;
  offset?: BMapSize;
  type?: number;
}

/** 缩放控件 */
export interface BMapZoomControl {
  initialize(map: BMapMap): HTMLElement;
}

/** 比例尺控件 */
export interface BMapScaleControl {
  initialize(map: BMapMap): HTMLElement;
}

/** 导航控件 */
export interface BMapNavigationControl {
  initialize(map: BMapMap): HTMLElement;
}

/** 缩放控件构造函数 */
export interface BMapZoomControlConstructor {
  new (opts?: BMapZoomControlOptions): BMapZoomControl;
}

/** 比例尺控件构造函数 */
export interface BMapScaleControlConstructor {
  new (opts?: BMapScaleControlOptions): BMapScaleControl;
}

/** 导航控件构造函数 */
export interface BMapNavigationControlConstructor {
  new (opts?: BMapNavigationControlOptions): BMapNavigationControl;
}

// ============ 命名空间 ============

/** 百度地图命名空间（BMap - 标准2D版本） */
export interface BMapNamespace {
  Map: BMapMapConstructor;
  Point: BMapPointConstructor;
  Pixel: new (x: number, y: number) => BMapPixel;
  Size: BMapSizeConstructor;
  Icon: BMapIconConstructor;
  Marker: BMapMarkerConstructor;
  InfoWindow: BMapInfoWindowConstructor;
  Polygon: BMapPolygonConstructor;
  Label: BMapLabelConstructor;
  Boundary: BMapBoundaryConstructor;
  NavigationControl: BMapNavigationControlConstructor;
  ScaleControl: BMapScaleControlConstructor;

  // 常量
  ANIMATION_BOUNCE: number;
  ANIMATION_DROP: number;
}

/** 百度地图 GL 命名空间（BMapGL - 3D版本） */
export interface BMapGLNamespace extends BMapNamespace {
  Map: BMapMapGLConstructor;
  ZoomControl: BMapZoomControlConstructor;
  ScaleControl: BMapScaleControlConstructor;
}

// ============ 全局 Window 扩展 ============
// 这些类型扩展需要在 global.d.ts 中声明

// ============ 类型助手函数 ============

/**
 * 获取百度地图命名空间（标准2D版本）
 */
export function getBMap(): BMapNamespace | undefined {
  return window.BMap as BMapNamespace | undefined;
}

/**
 * 获取百度地图 GL 命名空间（3D版本）
 */
export function getBMapGL(): BMapGLNamespace | undefined {
  return window.BMapGL as unknown as BMapGLNamespace | undefined;
}

/**
 * 检查百度地图是否已加载
 */
export function isBMapLoaded(): boolean {
  return typeof window.BMap !== "undefined" || typeof window.BMapGL !== "undefined";
}

/**
 * 检查百度地图 GL 是否已加载
 */
export function isBMapGLLoaded(): boolean {
  return typeof window.BMapGL !== "undefined";
}
