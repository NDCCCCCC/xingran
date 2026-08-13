/**
 * 湖北省地图组件 (WebGL 版本)
 * 集成百度地图 WebGL，显示楼宇标记，支持3D倾斜视角
 */

import { useRef, useState, useEffect, useCallback } from "react";
import { App, Spin, Badge, List, Tag, Button } from "antd";
import {
  EyeOutlined,
  EyeInvisibleOutlined,
  CompressOutlined,
  EnvironmentOutlined,
} from "@ant-design/icons";
import type { BuildingItem } from "../types";
import { useVisualizationStore } from "@/store/visualizationStore";
import { loadBaiduMapGLScript } from "./BaiduMapScript";
import {
  BAIDU_MAP_AK,
  MAP_CONFIG,
  HUBEI_BOUNDARY,
  CLUSTER_PIXEL_THRESHOLD,
  CLUSTER_MARKER_MIN_SIZE,
  CLUSTER_MARKER_MAX_INCREMENT,
  MARKER_COLORS,
  STATUS_TEXT_COLORS,
  ANIMATION_DURATION,
  EASING_FUNCTIONS,
} from "../constants";
import {
  toBase64,
  getOfficialHubeiBoundary,
  parseBoundaryPoints,
  pixelDistance,
  averagePixelPosition,
  isBuildingStopped,
  getBuildingLabel,
  getBuildingStatusText,
  getBuildingStatusColor,
  filterBuildingsByZoom,
  animateViewTransition,
} from "../utils";
import { getBMapGL, type BMapGLNamespace, type BMapMapGL, type BMapMarker, type BMapInfoWindow, type BMapPolygon } from "@/types/baidu-map";

// ============ 类型定义 ============

/** 聚类群组 */
interface ClusterGroup {
  buildings: BuildingItem[];
  centerPixel: { x: number; y: number };
  clusterLng: number;
  clusterLat: number;
}

interface HubeiMapGLProps {
  buildings: BuildingItem[];
}

// ============ 组件 ============

const HubeiMapGL: React.FC<HubeiMapGLProps> = ({ buildings }) => {
  const { message } = App.useApp();
  // Refs
  const mapRef = useRef<HTMLDivElement>(null);
  const mapInstanceRef = useRef<BMapMapGL | null>(null);
  const markersRef = useRef<BMapMarker[]>([]);
  const infoWindowRef = useRef<BMapInfoWindow | null>(null);
  const polygonRef = useRef<BMapPolygon | null>(null);

  // State
  const [mapLoaded, setMapLoaded] = useState(false);
  const [loading, setLoading] = useState(true);
  const [hoveredBuildings, setHoveredBuildings] = useState<BuildingItem[]>([]);
  const [tooltipPosition, setTooltipPosition] = useState<{ x: number; y: number } | null>(null);
  const [selectedCluster, setSelectedCluster] = useState<ClusterGroup | null>(null);
  const [currentZoom, setCurrentZoom] = useState(MAP_CONFIG.DEFAULT_ZOOM);
  const [is3DMode, setIs3DMode] = useState(false);
  const [currentTilt, setCurrentTilt] = useState(0);

  const { clearSelection, navigateToBuilding } = useVisualizationStore();

  // ============ 地图初始化 ============

  useEffect(() => {
    if (!mapRef.current || !BAIDU_MAP_AK) {
      if (!BAIDU_MAP_AK) {
        message.error("未配置百度地图 AK");
      }
      return;
    }

    const initMap = async () => {
      try {
        setLoading(true);
        await loadBaiduMapGLScript(BAIDU_MAP_AK);

        if (!mapRef.current) return;

        const BMapGL = getBMapGL();
        if (!BMapGL) {
          message.error("百度地图GL加载失败");
          return;
        }

        const map = new BMapGL.Map(mapRef.current, {
          enableMapClick: false,
          showControls: false,
        });
        mapInstanceRef.current = map;

        // 设置中心点和缩放级别
        const point = new BMapGL.Point(MAP_CONFIG.CENTER[0], MAP_CONFIG.CENTER[1]);
        map.centerAndZoom(point, MAP_CONFIG.DEFAULT_ZOOM);

        // 限制缩放级别
        map.setMinZoom(MAP_CONFIG.MIN_ZOOM);
        map.setMaxZoom(MAP_CONFIG.MAX_ZOOM);

        // 设置地图样式
        map.setMapStyleV2({ styleId: MAP_CONFIG.STYLE_ID });

        // 启用交互
        map.enableScrollWheelZoom(true);
        map.enableDoubleClickZoom(true);

        // 添加控件
        addMapControls(map);

        // 创建信息窗口
        infoWindowRef.current = new BMapGL.InfoWindow("", {
          width: 280,
          height: 0,
          title: "",
          enableAutoPan: true,
          enableCloseOnClick: true,
        });

        // 添加湖北省边界
        addHubeiMask(map, BMapGL).catch(() => {
          // 静默处理
        });

        // 监听事件
        map.addEventListener("zoomend", () => setCurrentZoom(map.getZoom()));
        map.addEventListener("tiltend", () => setCurrentTilt(map.getTilt()));

        setMapLoaded(true);
        setLoading(false);
      } catch (error) {
        message.error("百度地图GL加载失败，请检查 AK 配置");
        console.error(error);
        setLoading(false);
      }
    };

    initMap();
  }, []);

  // ============ 地图控件 ============

  const addMapControls = (map: BMapMapGL) => {
    const BMapGL = getBMapGL();
    if (!BMapGL) return;

    // 缩放控件
    const zoomControl = new BMapGL.ZoomControl({
      anchor: (window.BMAPGL_ANCHOR_BOTTOM_RIGHT as unknown as number) ?? 0,
      offset: new BMapGL.Size(20, 50),
    });
    map.addControl(zoomControl);

    // 比例尺控件
    const scaleControl = new BMapGL.ScaleControl({
      anchor: (window.BMAPGL_ANCHOR_BOTTOM_LEFT as unknown as number) ?? 0,
      offset: new BMapGL.Size(80, 30),
    });
    map.addControl(scaleControl);
  };

  // ============ 湖北省边界 ============

  const addHubeiMask = async (map: BMapMapGL, BMapGL: BMapGLNamespace) => {
    try {
      const boundaries = await getOfficialHubeiBoundary(BMapGL);
      boundaries.forEach((boundaryStr: string) => {
        const points = parseBoundaryPoints(boundaryStr, BMapGL);
        const polygon = new BMapGL.Polygon(points, {
          strokeColor: "var(--theme-info, #1890ff)",
          strokeWeight: 4,
          strokeOpacity: 1,
          fillColor: "transparent",
          fillOpacity: 0,
        });
        map.addOverlay(polygon);
      });
    } catch (error) {
      console.warn("获取官方边界失败，使用简化边界:", error);
      addFallbackMask(map, BMapGL);
    }
  };

  const addFallbackMask = (map: BMapMapGL, BMapGL: BMapGLNamespace) => {
    const points = HUBEI_BOUNDARY.map(
      (coord) => new BMapGL.Point(coord[0], coord[1])
    );
    const polygon = new BMapGL.Polygon(points, {
      strokeColor: "var(--theme-info, #1890ff)",
      strokeWeight: 4,
      strokeOpacity: 0.8,
      fillColor: "transparent",
      fillOpacity: 0,
    });
    map.addOverlay(polygon);
    polygonRef.current = polygon;
  };

  // ============ 3D 视角控制 ============

  const toggle3DMode = useCallback(() => {
    const map = mapInstanceRef.current;
    if (!map) return;

    const targetTilt = is3DMode ? 0 : MAP_CONFIG.DEFAULT_TILT;
    setIs3DMode(!is3DMode);

    animateViewTransition(
      map,
      { tilt: targetTilt },
      ANIMATION_DURATION.TOGGLE_3D,
      EASING_FUNCTIONS.easeInOutCubic
    );

    setTimeout(() => setCurrentTilt(targetTilt), ANIMATION_DURATION.TOGGLE_3D);
  }, [is3DMode]);

  const resetView = useCallback(() => {
    const map = mapInstanceRef.current;
    if (!map) return;

    setIs3DMode(false);
    setCurrentTilt(0);

    animateViewTransition(
      map,
      {
        center: MAP_CONFIG.CENTER,
        zoom: MAP_CONFIG.DEFAULT_ZOOM,
        tilt: 0,
        heading: 0,
      },
      ANIMATION_DURATION.DEFAULT,
      EASING_FUNCTIONS.easeInOutCubic
    );
  }, []);

  // ============ 标记渲染 ============

  useEffect(() => {
    if (!mapInstanceRef.current || !mapLoaded) return;

    const map = mapInstanceRef.current;
    const BMapGL = getBMapGL();
    if (!BMapGL) return;

    // 清除现有标记
    markersRef.current.forEach((marker) => map.removeOverlay(marker));
    markersRef.current = [];

    // 过滤楼宇
    const filteredBuildings = filterBuildingsByZoom(buildings, currentZoom);
    const buildingsWithCoords = filteredBuildings.filter(
      (b) => b.longitude && b.latitude
    );

    // 聚类处理
    const clusterGroups: ClusterGroup[] = [];
    const processedBuildings = new Set<string>();

    buildingsWithCoords.forEach((building) => {
      if (processedBuildings.has(building.id)) return;

      const buildingPoint = new BMapGL.Point(building.longitude!, building.latitude!);
      const buildingPixel = map.pointToOverlayPixel(buildingPoint);

      const overlappedBuildings: BuildingItem[] = [building];
      const pixels: Array<{ x: number; y: number }> = [buildingPixel];

      buildingsWithCoords.forEach((otherBuilding) => {
        if (
          otherBuilding.id === building.id ||
          processedBuildings.has(otherBuilding.id)
        ) {
          return;
        }

        const otherPoint = new BMapGL.Point(
          otherBuilding.longitude!,
          otherBuilding.latitude!
        );
        const otherPixel = map.pointToOverlayPixel(otherPoint);

        const distance = pixelDistance(buildingPixel, otherPixel);

        if (distance < CLUSTER_PIXEL_THRESHOLD) {
          overlappedBuildings.push(otherBuilding);
          pixels.push(otherPixel);
          processedBuildings.add(otherBuilding.id);
        }
      });

      const avgPixel = averagePixelPosition(pixels);
      const centerPoint =
        map.pixelToPoint?.(new BMapGL.Pixel(avgPixel.x, avgPixel.y)) ||
        { lng: building.longitude!, lat: building.latitude! };

      const cluster: ClusterGroup = {
        buildings: overlappedBuildings,
        centerPixel: avgPixel,
        clusterLng: centerPoint.lng || building.longitude!,
        clusterLat: centerPoint.lat || building.latitude!,
      };

      clusterGroups.push(cluster);
      processedBuildings.add(building.id);
    });

    // 渲染标记
    clusterGroups.forEach((cluster) => {
      renderClusterMarker(cluster, map, BMapGL);
    });
  }, [mapLoaded, buildings, currentZoom]);

  const renderClusterMarker = (
    cluster: ClusterGroup,
    map: BMapMapGL,
    BMapGL: BMapGLNamespace
  ) => {
    const { buildings, clusterLng, clusterLat, centerPixel } = cluster;
    const isCluster = buildings.length > 1;
    const point = new BMapGL.Point(clusterLng, clusterLat);

    if (isCluster) {
      const size =
        CLUSTER_MARKER_MIN_SIZE +
        Math.min(buildings.length * 2, CLUSTER_MARKER_MAX_INCREMENT);
      const clusterIcon = new BMapGL.Icon(
        "data:image/svg+xml;base64," +
          toBase64(`
          <svg xmlns="http://www.w3.org/2000/svg" width="${size}" height="${size}" viewBox="0 0 ${size} ${size}">
            <defs>
              <filter id="clusterShadow" x="-50%" y="-50%" width="200%" height="200%">
                <feDropShadow dx="0" dy="2" stdDeviation="3" flood-color="${MARKER_COLORS.CLUSTER_SHADOW}" flood-opacity="0.7"/>
              </filter>
            </defs>
            <circle cx="${size / 2}" cy="${size / 2}" r="${size / 2 - 1}" fill="${MARKER_COLORS.CLUSTER_BG}"/>
            <circle cx="${size / 2}" cy="${size / 2}" r="${size / 2 - 4}" fill="${MARKER_COLORS.CLUSTER}" stroke="#fff" stroke-width="3" filter="url(#clusterShadow)"/>
            <text x="${size / 2}" y="${size / 2 + 8}" text-anchor="middle" fill="white" font-size="${size / 3}" font-weight="bold">${buildings.length}</text>
          </svg>
        `),
        new BMapGL.Size(size, size),
        new BMapGL.Size(size / 2, size / 2)
      );

      const marker = new BMapGL.Marker(point, { icon: clusterIcon });

      marker.addEventListener("mouseover", () => {
        showTooltip(buildings, centerPixel);
      });

      marker.addEventListener("mouseout", hideTooltip);

      marker.addEventListener("click", () => {
        setSelectedCluster(cluster);
      });

      map.addOverlay(marker);
      markersRef.current.push(marker);
    } else {
      const building = buildings[0];
      const stopped = isBuildingStopped(building);

      const mainColor = stopped ? MARKER_COLORS.STOPPED_MAIN : MARKER_COLORS.NORMAL_MAIN;
      const borderColor = stopped ? MARKER_COLORS.STOPPED_BORDER : MARKER_COLORS.NORMAL_BORDER;
      const shadowColor = stopped ? MARKER_COLORS.STOPPED_SHADOW : MARKER_COLORS.NORMAL_SHADOW;
      const label = getBuildingLabel(building);

      const marker = new BMapGL.Marker(point, {
        icon: new BMapGL.Icon(
          "data:image/svg+xml;base64," +
            toBase64(`
            <svg xmlns="http://www.w3.org/2000/svg" width="42" height="48" viewBox="0 0 42 48">
              <defs>
                <filter id="shadow" x="-50%" y="-50%" width="200%" height="200%">
                  <feDropShadow dx="0" dy="2" stdDeviation="2" flood-color="${shadowColor}" flood-opacity="0.6"/>
                </filter>
              </defs>
              <path d="M21 4 L21 4 Q21 4 21 4 C9 4 4 14 4 22 C4 32 21 44 21 44 C21 44 38 32 38 22 C38 14 33 4 21 4 Z"
                fill="${mainColor}" stroke="${borderColor}" stroke-width="3" filter="url(#shadow)"/>
              <circle cx="21" cy="20" r="8" fill="white" opacity="0.3"/>
              <text x="21" y="25" text-anchor="middle" fill="white" font-size="11" font-weight="bold">${label}</text>
            </svg>
          `),
          new BMapGL.Size(42, 48),
          new BMapGL.Size(21, 48)
        ),
      });

      marker.addEventListener("mouseover", () => {
        showTooltip([building], centerPixel);
      });

      marker.addEventListener("mouseout", hideTooltip);

      marker.addEventListener("click", () => {
        showBuildingInfo(building, map, BMapGL);
      });

      map.addOverlay(marker);
      markersRef.current.push(marker);
    }
  };

  // ============ 工具提示 ============

  const showTooltip = (buildings: BuildingItem[], point: { x: number; y: number }) => {
    setHoveredBuildings(buildings);
    setTooltipPosition({ x: point.x, y: point.y });
  };

  const hideTooltip = () => {
    setHoveredBuildings([]);
    setTooltipPosition(null);
  };

  const showBuildingInfo = (building: BuildingItem, map: BMapMapGL, BMapGL: BMapGLNamespace) => {
    const stopped = isBuildingStopped(building);
    const statusColor = stopped
      ? STATUS_TEXT_COLORS.STOPPED
      : STATUS_TEXT_COLORS.NORMAL;
    const statusBg = stopped
      ? STATUS_TEXT_COLORS.STOPPED_BG
      : STATUS_TEXT_COLORS.NORMAL_BG;

    const content = `
      <div style="padding: 12px; min-width: 260px;">
        <h3 style="margin: 0 0 12px 0; font-size: 16px; color: var(--theme-info, #1890ff); border-bottom: 2px solid #1890ff; padding-bottom: 8px;">
          ${building.name}
        </h3>
        <div style="font-size: 13px; color: var(--theme-text-tertiary, #666); line-height: 1.8;">
          <div>📌 ${building.address || "暂无地址"}</div>
          <div style="margin-top: 8px;">
            <span style="display: inline-block; padding: 2px 10px; background: ${statusBg}; color: ${statusColor}; border-radius: 12px; font-size: 12px; font-weight: 500;">
              ${stopped ? "⏸ 已停用" : "✓ 正常"}
            </span>
            <span style="margin-left: 12px;">🏢 ${building.floorCount || 0} 层</span>
            <span style="margin-left: 12px;">🪑 ${building.workstationCount || 0} 工位</span>
          </div>
        </div>
        <div style="margin-top: 12px; text-align: right;">
          <button
            onclick="window.viewBuildingDetailsGL('${building.id}')"
            style="
              padding: 6px 18px;
              background: linear-gradient(135deg, #1890ff, #096dd9);
              color: white;
              border: none;
              border-radius: 6px;
              cursor: pointer;
              font-size: 13px;
              font-weight: 500;
              box-shadow: 0 2px 4px rgba(24, 144, 255, 0.3);
            "
          >
            查看详情
          </button>
        </div>
      </div>
    `;

    infoWindowRef.current?.setContent(content);
    const point = new BMapGL.Point(building.longitude!, building.latitude!);
    const infoWindow = infoWindowRef.current;
    if (infoWindow) {
      map.openInfoWindow(infoWindow, point);
    }
  };

  // ============ 全局函数 ============

  useEffect(() => {
    (window as any).viewBuildingDetailsGL = (buildingId: string) => {
      const building = buildings.find((b) => b.id === buildingId);
      if (building) {
        navigateToBuilding({
          id: building.id,
          orgId: "",
          name: building.name,
          code: building.code || "",
          address: building.address,
          longitude: building.longitude,
          latitude: building.latitude,
          status: (building.status ?? 0) as 0 | 1,
          createdAt: "",
          updatedAt: "",
        });
      }
    };

    return () => {
      delete window.viewBuildingDetailsGL;
    };
  }, [buildings, navigateToBuilding]);

  // ============ 渲染 ============

  if (!BAIDU_MAP_AK) {
    return (
      <div
        style={{
          display: "flex",
          justifyContent: "center",
          alignItems: "center",
          height: "100vh",
          flexDirection: "column",
          gap: 16,
        }}
      >
        <div style={{ fontSize: 18, color: "var(--theme-error, #ff4d4f)" }}>未配置百度地图 AK</div>
        <div style={{ color: "var(--theme-text-tertiary, #666)" }}>
          请在 .env.development 文件中配置 VITE_BAIDU_MAP_AK
        </div>
      </div>
    );
  }

  return (
    <div style={{ height: "100vh", width: "100%", position: "relative" }}>
      {/* 加载提示 */}
      {loading && (
        <div
          style={{
            position: "absolute",
            top: "50%",
            left: "50%",
            transform: "translate(-50%, -50%)",
            zIndex: 1000,
            textAlign: "center",
          }}
        >
          <Spin size="large" />
          <div style={{ marginTop: 8 }}>加载 WebGL 地图中...</div>
        </div>
      )}

      {/* 地图容器 */}
      <div
        ref={mapRef}
        style={{ height: "100vh", width: "100%" }}
        onClick={() => clearSelection()}
      />

      {/* 3D模式切换按钮 */}
      <div
        style={{
          position: "absolute",
          top: 16,
          right: 16,
          zIndex: 100,
          display: "flex",
          flexDirection: "column",
          gap: 8,
        }}
      >
        <Button
          type={is3DMode ? "primary" : "default"}
          icon={is3DMode ? <EyeOutlined /> : <EyeInvisibleOutlined />}
          onClick={toggle3DMode}
        >
          {is3DMode ? "3D 视角" : "2D 视角"}
        </Button>
        <Button icon={<CompressOutlined />} onClick={resetView}>
          重置
        </Button>
      </div>

      {/* 视角信息 */}
      <MapInfoPanel currentZoom={currentZoom} is3DMode={is3DMode} currentTilt={currentTilt} />

      {/* 悬停提示 */}
      {hoveredBuildings.length > 0 && tooltipPosition && (
        <TooltipPanel
          buildings={hoveredBuildings}
          position={tooltipPosition}
        />
      )}

      {/* 操作提示 */}
      <ControlHints is3DMode={is3DMode} />

      {/* 聚类列表侧边栏 */}
      {selectedCluster && (
        <ClusterSidebar
          cluster={selectedCluster}
          onClose={() => setSelectedCluster(null)}
          onBuildingClick={(building) => {
            setSelectedCluster(null);
            navigateToBuilding({
              id: building.id,
              orgId: "",
              name: building.name,
              code: building.code || "",
              address: building.address,
              longitude: building.longitude,
              latitude: building.latitude,
              status: (building.status ?? 0) as 0 | 1,
              createdAt: "",
              updatedAt: "",
            });
          }}
        />
      )}
    </div>
  );
};

// ============ 子组件 ============

interface MapInfoPanelProps {
  currentZoom: number;
  is3DMode: boolean;
  currentTilt: number;
}

const MapInfoPanel: React.FC<MapInfoPanelProps> = ({
  currentZoom,
  is3DMode,
  currentTilt,
}) => (
  <div
    style={{
      position: "absolute",
      top: 16,
      left: 16,
      background: "rgba(255, 255, 255, 0.95)",
      padding: "12px 16px",
      borderRadius: "8px",
      boxShadow: "0 2px 8px rgba(0, 0, 0, 0.1)",
      fontSize: 12,
      color: "var(--theme-text-tertiary, #666)",
      zIndex: 100,
    }}
  >
    <div style={{ fontWeight: "bold", marginBottom: 8, fontSize: 14 }}>
      <EnvironmentOutlined /> 地图信息
    </div>
    <div>
      缩放级别:{" "}
      <Badge count={currentZoom} showZero style={{ backgroundColor: "var(--theme-info, #1890ff)" }} />
    </div>
    <div>
      3D 视角:
      <Badge
        count={is3DMode ? `${currentTilt}°` : "关闭"}
        showZero
        style={{
          backgroundColor: is3DMode ? "var(--theme-purple, #722ed1)" : "#d9d9d9",
          marginLeft: 8,
        }}
      />
    </div>
    {is3DMode && (
      <div style={{ fontSize: 11, color: "var(--theme-text-tertiary, #999)", marginTop: 4 }}>
        拖拽地图旋转视角
      </div>
    )}
  </div>
);

interface TooltipPanelProps {
  buildings: BuildingItem[];
  position: { x: number; y: number };
}

const TooltipPanel: React.FC<TooltipPanelProps> = ({ buildings, position }) => (
  <div
    style={{
      position: "absolute",
      left: position.x,
      top: position.y,
      transform: "translate(-50%, -100%)",
      marginTop: -10,
      background: "white",
      border: "1px solid #d9d9d9",
      borderRadius: "4px",
      padding: "8px 12px",
      boxShadow: "0 2px 8px rgba(0, 0, 0, 0.15)",
      zIndex: 1000,
      minWidth: "200px",
      pointerEvents: "none",
    }}
  >
    {buildings.length === 1 ? (
      <div>
        <div style={{ fontWeight: "bold", color: "var(--theme-text-accent, #1890ff)", marginBottom: 4 }}>
          {buildings[0].name}
        </div>
        <div style={{ fontSize: 11, color: "var(--theme-text-tertiary, #666)" }}>
          {buildings[0].floorCount || 0}层 · {buildings[0].workstationCount || 0}
          工位
        </div>
      </div>
    ) : (
      <div>
        <div style={{ fontWeight: "bold", color: "var(--theme-error, #ff4d4f)", marginBottom: 4 }}>
          {buildings.length} 栋楼宇
        </div>
        {buildings.slice(0, 3).map((b, i) => (
          <div key={i} style={{ fontSize: 11, color: "var(--theme-text-tertiary, #666)" }}>
            · {b.name}
          </div>
        ))}
        {buildings.length > 3 && (
          <div style={{ fontSize: 11, color: "var(--theme-text-tertiary, #999)" }}>
            ...还有 {buildings.length - 3} 栋
          </div>
        )}
      </div>
    )}
  </div>
);

interface ControlHintsProps {
  is3DMode: boolean;
}

const ControlHints: React.FC<ControlHintsProps> = ({ is3DMode }) => (
  <div
    style={{
      position: "absolute",
      bottom: 16,
      left: 16,
      background: "rgba(255, 255, 255, 0.95)",
      padding: "12px 16px",
      borderRadius: "8px",
      boxShadow: "0 2px 8px rgba(0, 0, 0, 0.1)",
      fontSize: 12,
      color: "var(--theme-text-tertiary, #666)",
      zIndex: 100,
    }}
  >
    <div style={{ fontWeight: "bold", marginBottom: 8, fontSize: 14 }}>操作提示</div>
    {is3DMode ? (
      <>
        <div>🎮 当前为 3D 视角模式</div>
        <div style={{ fontSize: 11, color: "var(--theme-text-tertiary, #999)", marginTop: 4 }}>
          • 右键拖拽旋转视角
        </div>
        <div style={{ fontSize: 11, color: "var(--theme-text-tertiary, #999)" }}>• 滚轮缩放地图</div>
        <div style={{ fontSize: 11, color: "var(--theme-text-tertiary, #999)" }}>• 双击重置视角</div>
      </>
    ) : (
      <>
        <div>📊 当前为 2D 视角模式</div>
        <div>👆 滚轮缩放切换层级</div>
        <div>👆 悬停查看简要信息</div>
        <div>🖱️ 点击查看详细信息</div>
      </>
    )}

    <div
      style={{
        marginTop: 8,
        padding: "8px",
        background: "var(--theme-primary-lighter, #e6f7ff)",
        borderRadius: "4px",
        border: "1px solid var(--theme-primary-subtle, #91d5ff)",
        color: "var(--theme-primary, #096dd9)",
      }}
    >
      <div style={{ fontWeight: "bold", marginBottom: 4 }}>📋 WebGL 功能</div>
      <div style={{ fontSize: 11 }}>• 点击右上角按钮切换 2D/3D 视角</div>
      <div style={{ fontSize: 11 }}>• 3D 模式支持60度倾斜视角</div>
      <div style={{ fontSize: 11 }}>• 平滑动画过渡效果</div>
    </div>
  </div>
);

interface ClusterSidebarProps {
  cluster: {
    buildings: BuildingItem[];
  };
  onClose: () => void;
  onBuildingClick: (building: BuildingItem) => void;
}

const ClusterSidebar: React.FC<ClusterSidebarProps> = ({
  cluster,
  onClose,
  onBuildingClick,
}) => (
  <div
    style={{
      position: "absolute",
      top: 0,
      right: 0,
      width: 320,
      height: "100vh",
      background: "white",
      boxShadow: "-2px 0 8px rgba(0, 0, 0, 0.1)",
      zIndex: 1000,
      display: "flex",
      flexDirection: "column",
    }}
  >
    <div
      style={{
        padding: "16px",
        borderBottom: "1px solid #f0f0f0",
        display: "flex",
        justifyContent: "space-between",
        alignItems: "center",
      }}
    >
      <div>
        <div style={{ fontWeight: "bold", fontSize: 16, color: "var(--theme-error, #ff4d4f)" }}>
          📍 {cluster.buildings.length} 栋楼宇
        </div>
        <div style={{ fontSize: 12, color: "var(--theme-text-tertiary, #999)", marginTop: 4 }}>
          位置相近，已聚类显示
        </div>
      </div>
      <button
        onClick={onClose}
        style={{
          border: "none",
          background: "none",
          fontSize: 24,
          cursor: "pointer",
          color: "var(--theme-text-tertiary, #999)",
          padding: 0,
          width: 32,
          height: 32,
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
        }}
      >
        ×
      </button>
    </div>

    <div style={{ flex: 1, overflow: "auto", padding: "16px" }}>
      <List
        dataSource={cluster.buildings}
        renderItem={(building) => (
          <List.Item
            key={building.id}
            style={{
              border: "1px solid #f0f0f0",
              borderRadius: "8px",
              marginBottom: "12px",
              padding: "12px",
              cursor: "pointer",
            }}
            onClick={() => onBuildingClick(building)}
          >
            <div style={{ width: "100%" }}>
              <div
                style={{
                  display: "flex",
                  justifyContent: "space-between",
                  alignItems: "center",
                  marginBottom: 8,
                }}
              >
                <div
                  style={{ fontWeight: "bold", color: "var(--theme-text-accent, #1890ff)", fontSize: 14 }}
                >
                  {building.name}
                </div>
                <Tag color={getBuildingStatusColor(building.status)}>
                  {getBuildingStatusText(building.status)}
                </Tag>
              </div>

              <div style={{ fontSize: 12, color: "var(--theme-text-tertiary, #666)", lineHeight: 1.8 }}>
                <div>📌 {building.address || "暂无地址"}</div>
                <div style={{ marginTop: 4 }}>
                  <span>🏢 {building.floorCount || 0} 层</span>
                  <span style={{ marginLeft: 12 }}>
                    🪑 {building.workstationCount || 0} 工位
                  </span>
                </div>
              </div>

              <div style={{ marginTop: 12, textAlign: "right" }}>
                <Tag color="blue">查看详情 →</Tag>
              </div>
            </div>
          </List.Item>
        )}
      />
    </div>

    <div
      style={{
        padding: "12px 16px",
        borderTop: "1px solid #f0f0f0",
        background: "#fafafa",
        fontSize: 11,
        color: "var(--theme-text-tertiary, #999)",
        textAlign: "center",
      }}
    >
      点击楼宇卡片查看 3D 详情
    </div>
  </div>
);

export default HubeiMapGL;
