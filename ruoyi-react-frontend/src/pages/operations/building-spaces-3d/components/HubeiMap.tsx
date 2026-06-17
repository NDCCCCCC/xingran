/**
 * 湖北省地图组件
 * 集成百度地图，显示楼宇标记
 * 功能：限制缩放范围、悬停显示详细信息、处理重叠楼宇
 */

import { useRef, useState, useEffect } from "react";
import { App, Spin, Badge, List, Tag } from "antd";
import type { BuildingItem } from "../types";
import { useVisualizationStore } from "@/store/visualizationStore";
import { getMapMarkerColor } from "@/utils/three/colors";
import { loadBaiduMapScript } from "./BaiduMapScript";
import { getBMap, type BMapNamespace, type BMapMap, type BMapMarker, type BMapInfoWindow, type BMapPolygon, type BMapPoint } from "@/types/baidu-map";

// 聚类群组类型
interface ClusterGroup {
  buildings: BuildingItem[];
  centerPixel: { x: number; y: number };
  clusterLng: number;
  clusterLat: number;
}

// 获取百度地图 AK
const BAIDU_MAP_AK = import.meta.env.VITE_BAIDU_MAP_AK || "";

// 地图配置
const MAP_CONFIG = {
  // 湖北省中心点
  center: [114.305393, 30.593099],
  // 默认缩放级别（显示整个湖北省）
  defaultZoom: 8,
  // 最小缩放级别（只能缩小到8级）
  minZoom: 8,
  // 最大缩放级别（只能放大到10级）
  maxZoom: 10,
};

interface HubeiMapProps {
  buildings: BuildingItem[];
}

// 辅助函数：正确处理包含 Unicode 字符的 base64 编码
const toBase64 = (str: string): string => {
  try {
    return btoa(unescape(encodeURIComponent(str)));
  } catch (_e) {
    return btoa(str);
  }
};

const HubeiMap: React.FC<HubeiMapProps> = ({ buildings }) => {
  const { message } = App.useApp();
  const mapRef = useRef<HTMLDivElement>(null);
  const mapInstanceRefLocal = useRef<BMapMap | null>(null);
  const markersRef = useRef<BMapMarker[]>([]);
  const infoWindowRef = useRef<BMapInfoWindow | null>(null);
  const polygonRef = useRef<BMapPolygon | null>(null);
  const [mapLoaded, setMapLoaded] = useState(false);
  const [loading, setLoading] = useState(true);
  const [hoveredBuildings, setHoveredBuildings] = useState<BuildingItem[]>([]);
  const [tooltipPosition, setTooltipPosition] = useState<{ x: number; y: number } | null>(null);
  const [selectedCluster, setSelectedCluster] = useState<ClusterGroup | null>(null);
  const [currentZoom, setCurrentZoom] = useState(8); // 当前缩放级别

  const {
    clearSelection,
    navigateToBuilding,
  } = useVisualizationStore();

  // 湖北省边界坐标（简化版，涵盖主要区域）
  const HUBEI_BOUNDARY = [
    [109.5, 33.5], [111.5, 33.5], [113.5, 33.2], [115.5, 33.0], [116.5, 32.5],
    [116.2, 31.8], [116.1, 31.0], [115.8, 30.2], [115.5, 29.5], [115.2, 29.0],
    [114.8, 28.5], [114.0, 28.2], [113.0, 28.0], [112.0, 28.2], [111.0, 28.5],
    [110.0, 29.0], [109.5, 29.8], [109.2, 30.5], [109.0, 31.2], [109.0, 32.0],
    [109.2, 32.5], [109.5, 33.5]
  ];

  // 获取湖北省官方边界（使用百度地图 Boundary API）
  const getOfficialHubeiBoundary = (BMap: BMapNamespace): Promise<string[]> => {
    return new Promise((resolve, reject) => {
      const bdary = new BMap.Boundary();
      bdary.get("湖北省", (rs: { boundaries?: string[] }) => {
        if (rs.boundaries && rs.boundaries.length > 0) {
          resolve(rs.boundaries);
        } else {
          reject(new Error("未获取到边界数据"));
        }
      });
    });
  };

  // 将边界字符串转换为 BMap.Point 数组
  const parseBoundaryPoints = (boundaryStr: string, BMap: BMapNamespace): BMapPoint[] => {
    return boundaryStr.split(";").map((point: string) => {
      const [lng, lat] = point.split(",").map(Number);
      return new BMap.Point(lng, lat);
    });
  };

  // 添加湖北省边界轮廓（加粗蓝色边线）
  const addHubeiMask = async (map: BMapMap, BMap: BMapNamespace) => {
    try {
      // 获取官方边界
      const boundaries = await getOfficialHubeiBoundary(BMap);

      // 绘制湖北省边界轮廓（加粗蓝色边线）
      boundaries.forEach((boundaryStr) => {
        const points = parseBoundaryPoints(boundaryStr, BMap);
        const polygon = new BMap.Polygon(points, {
          strokeColor: "var(--theme-info, #1890ff)",
          strokeWeight: 4,           // 加粗边界线
          strokeOpacity: 1,          // 完全不透明
          fillColor: "transparent",   // 无填充
          fillOpacity: 0,
        });
        map.addOverlay(polygon);
      });

    } catch (error) {
      addFallbackMask(map, BMap);
    }
  };

  // 降级方案：使用简化的手动边界
  const addFallbackMask = (map: BMapMap, BMap: BMapNamespace) => {
    // 绘制简化边界（只显示边线，无遮罩）
    const points = HUBEI_BOUNDARY.map(coord => new BMap.Point(coord[0], coord[1]));
    const polygon = new BMap.Polygon(points, {
      strokeColor: "var(--theme-info, #1890ff)",
      strokeWeight: 4,           // 加粗边界线
      strokeOpacity: 0.8,
      fillColor: "transparent",   // 无填充
      fillOpacity: 0,
    });
    map.addOverlay(polygon);
    polygonRef.current = polygon;
  };

  // 初始化地图
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
        await loadBaiduMapScript(BAIDU_MAP_AK);

        if (!mapRef.current) return;

        // 创建地图实例
        const BMap = getBMap();
        if (!BMap) {
          message.error("百度地图加载失败");
          return;
        }
        const map = new BMap.Map(mapRef.current, {
          enableMapClick: false,
        });
        mapInstanceRefLocal.current = map;

        // 设置中心点和缩放级别
        const point = new BMap.Point(MAP_CONFIG.center[0], MAP_CONFIG.center[1]);
        map.centerAndZoom(point, MAP_CONFIG.defaultZoom);

        // 限制缩放级别
        map.setMinZoom(MAP_CONFIG.minZoom);
        map.setMaxZoom(MAP_CONFIG.maxZoom);

        // 启用滚轮缩放和双击缩放
        map.enableScrollWheelZoom(true);
        map.enableDoubleClickZoom(true);

        // 添加地图控件
        map.addControl(new BMap.NavigationControl());
        map.addControl(new BMap.ScaleControl());

        // 创建信息窗口实例
        infoWindowRef.current = new BMap.InfoWindow("", {
          width: 280,
          height: 0,
          title: "",
          enableAutoPan: true,
          enableCloseOnClick: true,
        });

        // 添加湖北省遮罩层（异步加载边界数据）
        addHubeiMask(map, BMap).catch(() => {
          // 遮罩层添加失败，静默处理
        });

        // 监听缩放变化，根据缩放级别显示不同层级的楼宇
        const handleZoomEnd = () => {
          const zoom = map.getZoom();
          setCurrentZoom(zoom);
        };
        map.addEventListener("zoomend", handleZoomEnd);

        // 初始设置当前缩放级别
        setCurrentZoom(MAP_CONFIG.defaultZoom);

        setMapLoaded(true);
        setLoading(false);
      } catch (error) {
        message.error("百度地图加载失败，请检查 AK 配置");
        setLoading(false);
      }
    };

    initMap();
  }, []);

  // 渲染楼宇标记（使用聚类算法）
  useEffect(() => {
    if (!mapInstanceRefLocal.current || !mapLoaded) return;

    const map = mapInstanceRefLocal.current;
    const BMap = getBMap();
    if (!BMap) return;

    // 清除现有标记
    markersRef.current.forEach(marker => map.removeOverlay(marker));
    markersRef.current = [];

    // 根据缩放级别过滤楼宇层级
    // zoom=8 显示level=1, zoom=9 显示level=1, zoom=10 显示level=1和level=2
    const filteredBuildings = currentZoom === 10
      ? buildings // zoom=10 显示所有楼宇
      : buildings.filter((b) => b.level === 1); // zoom=8/9 只显示一级楼宇

    // 聚类阈值（像素）
    const CLUSTER_PIXEL_THRESHOLD = 40; // 40px内视为一个聚类

    // 楼宇聚类检测
    const buildingsWithCoords = filteredBuildings.filter((b) => b.longitude && b.latitude);
    const clusterGroups: ClusterGroup[] = [];
    const processedBuildings = new Set<string>();

    buildingsWithCoords.forEach((building) => {
      if (processedBuildings.has(building.id)) return;

      const buildingPoint = new BMap.Point(building.longitude!, building.latitude!);
      const buildingPixel = map.pointToOverlayPixel(buildingPoint);

      // 查找与当前楼宇重叠的其他楼宇
      const overlappedBuildings: BuildingItem[] = [building];
      const pixels: Array<{ x: number; y: number }> = [buildingPixel];

      buildingsWithCoords.forEach((otherBuilding) => {
        if (otherBuilding.id === building.id || processedBuildings.has(otherBuilding.id)) return;

        const otherPoint = new BMap.Point(otherBuilding.longitude!, otherBuilding.latitude!);
        const otherPixel = map.pointToOverlayPixel(otherPoint);

        // 计算像素距离
        const distance = Math.sqrt(
          Math.pow(buildingPixel.x - otherPixel.x, 2) +
          Math.pow(buildingPixel.y - otherPixel.y, 2)
        );

        if (distance < CLUSTER_PIXEL_THRESHOLD) {
          overlappedBuildings.push(otherBuilding);
          pixels.push(otherPixel);
          processedBuildings.add(otherBuilding.id);
        }
      });

      // 计算聚类中心点（像素坐标的平均值）
      const avgPixelX = pixels.reduce((sum, p) => sum + p.x, 0) / pixels.length;
      const avgPixelY = pixels.reduce((sum, p) => sum + p.y, 0) / pixels.length;

      // 将像素中心转换回经纬度
      const centerPoint = map.pixelToPoint?.(new BMap.Pixel(avgPixelX, avgPixelY))
        || { lng: building.longitude!, lat: building.latitude! };

      const cluster: ClusterGroup = {
        buildings: overlappedBuildings,
        centerPixel: { x: avgPixelX, y: avgPixelY },
        clusterLng: centerPoint.lng || building.longitude!,
        clusterLat: centerPoint.lat || building.latitude!,
      };

      clusterGroups.push(cluster);
      processedBuildings.add(building.id);
    });

    // 渲染聚类标记
    clusterGroups.forEach((cluster) => {
      renderClusterMarker(cluster, map, BMap);
    });
  }, [mapLoaded, buildings, currentZoom]);

  // 渲染聚类标记（单个楼宇或多个楼宇聚类）
  const renderClusterMarker = (cluster: ClusterGroup, map: BMapMap, BMap: BMapNamespace) => {
    const { buildings, clusterLng, clusterLat, centerPixel } = cluster;
    const isCluster = buildings.length > 1;

    const point = new BMap.Point(clusterLng, clusterLat);

    if (isCluster) {
      // 聚类标记：显示数字圆圈 - 使用更醒目的样式
      const size = 48 + Math.min(buildings.length * 2, 24); // 动态大小，更大更醒目
      const clusterIcon = new BMap.Icon(
        "data:image/svg+xml;base64," + toBase64(`
          <svg xmlns="http://www.w3.org/2000/svg" width="${size}" height="${size}" viewBox="0 0 ${size} ${size}">
            <defs>
              <filter id="clusterShadow" x="-50%" y="-50%" width="200%" height="200%">
                <feDropShadow dx="0" dy="2" stdDeviation="3" flood-color="rgba(255, 77, 79, 0.5)" flood-opacity="0.7"/>
              </filter>
            </defs>
            <!-- 外圈阴影圆 -->
            <circle cx="${size/2}" cy="${size/2}" r="${size/2-1}" fill="rgba(255,77,79,0.1)"/>
            <!-- 主圆 -->
            <circle cx="${size/2}" cy="${size/2}" r="${size/2-4}" fill="#ff4d4f" stroke="#fff" stroke-width="3" filter="url(#clusterShadow)"/>
            <!-- 数字 -->
            <text x="${size/2}" y="${size/2 + 8}" text-anchor="middle" fill="white" font-size="${size/3}" font-weight="bold">${buildings.length}</text>
          </svg>
        `),
        new BMap.Size(size, size),
        new BMap.Size(size / 2, size / 2)
      );

      const marker = new BMap.Marker(point, { icon: clusterIcon });

      // 鼠标悬停：显示所有楼宇
      marker.addEventListener("mouseover", () => {
        showTooltip(buildings, centerPixel);
      });

      marker.addEventListener("mouseout", () => {
        hideTooltip();
      });

      // 点击：打开聚类列表
      marker.addEventListener("click", () => {
        setSelectedCluster(cluster);
      });

      map.addOverlay(marker);
      markersRef.current.push(marker);
    } else {
      // 单个楼宇标记 - 使用更醒目的样式
      const building = buildings[0];
      const isStopped = building.status === 1;

      // 使用醒目的橙红色作为正常楼宇标记，灰色作为停用楼宇标记
      const mainColor = isStopped ? "#9e9e9e" : "#ff6b35"; // 橙红色更醒目
      const borderColor = isStopped ? "#757575" : "#ff4d4f";
      const shadowColor = isStopped ? "rgba(0,0,0,0.2)" : "rgba(255, 107, 53, 0.5)";

      // 获取楼宇名称前两个字，不足则取全名
      const buildingLabel = building.name ? building.name.slice(0, 2) : "楼宇";

      const marker = new BMap.Marker(point, {
        icon: new BMap.Icon(
          "data:image/svg+xml;base64," + toBase64(`
            <svg xmlns="http://www.w3.org/2000/svg" width="42" height="48" viewBox="0 0 42 48">
              <defs>
                <filter id="shadow" x="-50%" y="-50%" width="200%" height="200%">
                  <feDropShadow dx="0" dy="2" stdDeviation="2" flood-color="${shadowColor}" flood-opacity="0.6"/>
                </filter>
              </defs>
              <!-- 楼宇图标底座（水滴形状） -->
              <path d="M21 4 L21 4 Q21 4 21 4 C9 4 4 14 4 22 C4 32 21 44 21 44 C21 44 38 32 38 22 C38 14 33 4 21 4 Z"
                fill="${mainColor}" stroke="${borderColor}" stroke-width="3" filter="url(#shadow)"/>
              <!-- 内部圆圈 -->
              <circle cx="21" cy="20" r="8" fill="white" opacity="0.3"/>
              <!-- 楼宇名称前两个字 -->
              <text x="21" y="25" text-anchor="middle" fill="white" font-size="11" font-weight="bold">${buildingLabel}</text>
            </svg>
          `),
          new BMap.Size(42, 48),
          new BMap.Size(21, 48)
        ),
      });

      // 鼠标悬停事件
      marker.addEventListener("mouseover", () => {
        showTooltip([building], centerPixel);
      });

      marker.addEventListener("mouseout", () => {
        hideTooltip();
      });

      // 点击事件
      marker.addEventListener("click", () => {
        showBuildingInfo(building, map, BMap);
      });

      map.addOverlay(marker);
      markersRef.current.push(marker);
    }
  };

  // 显示悬停提示
  const showTooltip = (buildings: BuildingItem[], point: { x: number; y: number }) => {
    setHoveredBuildings(buildings);
    setTooltipPosition({ x: point.x, y: point.y });
  };

  // 隐藏悬停提示
  const hideTooltip = () => {
    setHoveredBuildings([]);
    setTooltipPosition(null);
  };

  // 显示单个楼宇信息
  const showBuildingInfo = (building: BuildingItem, map: BMapMap, BMap: BMapNamespace) => {
    const isStopped = building.status === 1;
    const statusColor = isStopped ? "#9e9e9e" : "#ff6b35";
    const statusBg = isStopped ? "#f5f5f5" : "#fff2e8";

    const content = `
      <div style="padding: 12px; min-width: 260px;">
        <h3 style="margin: 0 0 12px 0; font-size: 16px; color: var(--theme-info, #1890ff); border-bottom: 2px solid #1890ff; padding-bottom: 8px;">
          ${building.name}
        </h3>
        <div style="font-size: 13px; color: var(--theme-text-tertiary, #666); line-height: 1.8;">
          <div>📌 ${building.address || "暂无地址"}</div>
          <div style="margin-top: 8px;">
            <span style="display: inline-block; padding: 2px 10px; background: ${statusBg}; color: ${statusColor}; border-radius: 12px; font-size: 12px; font-weight: 500;">
              ${isStopped ? "⏸ 已停用" : "✓ 正常"}
            </span>
            <span style="margin-left: 12px;">🏢 ${building.floorCount || 0} 层</span>
            <span style="margin-left: 12px;">🪑 ${building.workstationCount || 0} 工位</span>
          </div>
        </div>
        <div style="margin-top: 12px; text-align: right;">
          <button
            onclick="window.viewBuildingDetails('${building.id}')"
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
    const point = new BMap.Point(building.longitude!, building.latitude!);
    if (infoWindowRef.current) {
      map.openInfoWindow(infoWindowRef.current, point);
    }
  };

  // 全局函数：查看楼宇详情
  useEffect(() => {
    (window as any).viewBuildingDetails = (buildingId: string) => {
      const building = buildings.find(b => b.id === buildingId);
      if (building) {
        // 转换 BuildingItem 为 Building 类型，补充缺失字段
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
      delete window.viewBuildingDetails;
    };
  }, [buildings, navigateToBuilding]);

  // 如果没有配置 AK，显示提示
  if (!BAIDU_MAP_AK) {
    return (
      <div style={{
        display: "flex",
        justifyContent: "center",
        alignItems: "center",
        height: "100vh",
        flexDirection: "column",
        gap: 16,
      }}>
        <div style={{ fontSize: 18, color: "var(--theme-error, #ff4d4f)" }}>
          未配置百度地图 AK
        </div>
        <div style={{ color: "var(--theme-text-tertiary, #666)" }}>
          请在 .env.development 文件中配置 VITE_BAIDU_MAP_AK
        </div>
      </div>
    );
  }

  return (
    <div style={{ height: "100vh", width: "100%", position: "relative" }}>
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
          <div style={{ marginTop: 8 }}>加载地图中...</div>
        </div>
      )}

      {/* 地图容器 */}
      <div
        ref={mapRef}
        style={{ height: "100vh", width: "100%" }}
        onClick={() => {
          clearSelection();
        }}
      />

      {/* 悬停提示 */}
      {hoveredBuildings.length > 0 && tooltipPosition && (
        <div
          style={{
            position: "absolute",
            left: tooltipPosition.x,
            top: tooltipPosition.y,
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
          {hoveredBuildings.length === 1 ? (
            <div>
              <div style={{ fontWeight: "bold", color: "var(--theme-text-accent, #1890ff)", marginBottom: 4 }}>
                {hoveredBuildings[0].name}
              </div>
              <div style={{ fontSize: 11, color: "var(--theme-text-tertiary, #666)" }}>
                {hoveredBuildings[0].floorCount || 0}层 · {hoveredBuildings[0].workstationCount || 0}工位
              </div>
            </div>
          ) : (
            <div>
              <div style={{ fontWeight: "bold", color: "var(--theme-error, #ff4d4f)", marginBottom: 4 }}>
                {hoveredBuildings.length} 栋楼宇
              </div>
              {hoveredBuildings.slice(0, 3).map((b, i) => (
                <div key={i} style={{ fontSize: 11, color: "var(--theme-text-tertiary, #666)" }}>
                  · {b.name}
                </div>
              ))}
              {hoveredBuildings.length > 3 && (
                <div style={{ fontSize: 11, color: "var(--theme-text-tertiary, #999)" }}>
                  ...还有 {hoveredBuildings.length - 3} 栋
                </div>
              )}
              <div style={{ fontSize: 10, color: "var(--theme-text-tertiary, #999)", marginTop: 4 }}>
                点击查看全部
              </div>
            </div>
          )}
        </div>
      )}

      {/* 地图控制提示 */}
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
        <div style={{ fontWeight: "bold", marginBottom: 8, fontSize: 14 }}>数据统计</div>
        <div>当前缩放: <Badge count={currentZoom} showZero style={{ backgroundColor: "var(--theme-info, #1890ff)" }} /></div>
        <div>显示层级: <Badge count={currentZoom === 10 ? "全部" : "一级(城市)"} showZero style={{ backgroundColor: currentZoom === 10 ? "var(--theme-purple, #722ed1)" : "#13c2c2" }} /></div>
        <div>一级楼宇: <Badge count={buildings.filter(b => b.level === 1).length} showZero /></div>
        <div>二级楼宇: <Badge count={buildings.filter(b => b.level === 2).length} showZero /></div>
        <div>有坐标: <Badge count={buildings.filter(b => b.longitude && b.latitude).length} showZero style={{ backgroundColor: buildings.filter(b => b.longitude && b.latitude).length > 0 ? "var(--theme-success, #52c41a)" : "#ff4d4f" }} /></div>
        <div style={{ marginTop: 8, fontSize: 11, color: "var(--theme-text-tertiary, #999)" }}>
          <div>👆 滚轮缩放切换层级</div>
          <div>👆 悬停查看简要信息</div>
          <div>🖱️ 点击查看详细信息</div>
        </div>

        {/* 层级说明 */}
        <div style={{ marginTop: 8, padding: "8px", background: "var(--theme-primary-lighter, #e6f7ff)", borderRadius: "4px", border: "1px solid var(--theme-primary-subtle, #91d5ff)", color: "var(--theme-primary, #096dd9)" }}>
          <div style={{ fontWeight: "bold", marginBottom: 4 }}>📋 层级说明</div>
          <div style={{ fontSize: 11 }}>• 缩放级别 8-9: 显示一级楼宇（城市级汇总）</div>
          <div style={{ fontSize: 11 }}>• 缩放级别 10: 显示全部楼宇（一级+二级）</div>
        </div>

        {/* 有坐标的楼宇为 0 时的提示 */}
        {buildings.filter(b => b.longitude && b.latitude).length === 0 && buildings.length > 0 && (
          <div style={{ marginTop: 8, padding: "8px", background: "var(--theme-warning-bg, #fff2e8)", borderRadius: "4px", border: "1px solid var(--theme-warning, #ffbb96)", color: "var(--theme-warning, #d46b08)" }}>
            <div style={{ fontWeight: "bold", marginBottom: 4 }}>⚠️ 楼宇缺少坐标</div>
            <div style={{ fontSize: 11 }}>请在楼宇管理中为楼宇设置经纬度坐标</div>
          </div>
        )}
      </div>

      {/* 聚类列表侧边栏 */}
      {selectedCluster && (
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
          {/* 头部 */}
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
                📍 {selectedCluster.buildings.length} 栋楼宇
              </div>
              <div style={{ fontSize: 12, color: "var(--theme-text-tertiary, #999)", marginTop: 4 }}>
                位置相近，已聚类显示
              </div>
            </div>
            <button
              onClick={() => setSelectedCluster(null)}
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
              onMouseEnter={(e) => e.currentTarget.style.color = "var(--theme-text-tertiary, #666)"}
              onMouseLeave={(e) => e.currentTarget.style.color = "#999"}
            >
              ×
            </button>
          </div>

          {/* 楼宇列表 */}
          <div style={{ flex: 1, overflow: "auto", padding: "16px" }}>
            <List
              dataSource={selectedCluster.buildings}
              renderItem={(building) => (
                <List.Item
                  key={building.id}
                  style={{
                    border: "1px solid #f0f0f0",
                    borderRadius: "8px",
                    marginBottom: "12px",
                    padding: "12px",
                    cursor: "pointer",
                    transition: "all 0.2s",
                  }}
                  onMouseEnter={(e) => {
                    e.currentTarget.style.borderColor = "var(--theme-info, #1890ff)";
                    e.currentTarget.style.boxShadow = "0 2px 8px rgba(24, 144, 255, 0.2)";
                  }}
                  onMouseLeave={(e) => {
                    e.currentTarget.style.borderColor = "#f0f0f0";
                    e.currentTarget.style.boxShadow = "none";
                  }}
                  onClick={() => {
                    setSelectedCluster(null);
                    // 转换 BuildingItem 为 Building 类型
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
                >
                  <div style={{ width: "100%" }}>
                    {/* 标题行 */}
                    <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 8 }}>
                      <div style={{ fontWeight: "bold", color: "var(--theme-text-accent, #1890ff)", fontSize: 14 }}>
                        {building.name}
                      </div>
                      <Tag color={building.status === 1 ? "red" : "green"} style={{ margin: 0 }}>
                        {building.status === 1 ? "已停用" : "正常"}
                      </Tag>
                    </div>

                    {/* 详细信息 */}
                    <div style={{ fontSize: 12, color: "var(--theme-text-tertiary, #666)", lineHeight: 1.8 }}>
                      <div>📌 {building.address || "暂无地址"}</div>
                      <div style={{ marginTop: 4 }}>
                        <span>🏢 {building.floorCount || 0} 层</span>
                        <span style={{ marginLeft: 12 }}>🪑 {building.workstationCount || 0} 工位</span>
                      </div>
                    </div>

                    {/* 查看详情按钮 */}
                    <div style={{ marginTop: 12, textAlign: "right" }}>
                      <Tag color="blue" style={{ cursor: "pointer" }}>
                        查看详情 →
                      </Tag>
                    </div>
                  </div>
                </List.Item>
              )}
            />
          </div>

          {/* 底部提示 */}
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
      )}
    </div>
  );
};

export default HubeiMap;
