/**
 * 湖北省地图组件 (WebGL 版本)
 * 集成百度地图 WebGL，显示楼宇标记，支持3D倾斜视角
 * 功能：3D视角、平滑动画过渡、悬停显示详细信息、处理重叠楼宇
 */

import { useRef, useState, useEffect, useCallback } from 'react';
import { Spin, message, Badge, List, Tag, Button, Space } from 'antd';
import {
  ApartmentOutlined,
  EyeOutlined,
  EyeInvisibleOutlined,
  CompressOutlined,
  EnvironmentOutlined,
} from '@ant-design/icons';
import type { BuildingItem } from '../types';
import { useVisualizationStore } from '@/store/visualizationStore';
import { loadBaiduMapGLScript, isBaiduMapGLLoaded } from './BaiduMapGLScript';

// 聚类群组类型
interface ClusterGroup {
  buildings: BuildingItem[];
  centerPixel: { x: number; y: number };
  clusterLng: number;
  clusterLat: number;
}

// 获取百度地图 AK
const BAIDU_MAP_AK = import.meta.env.VITE_BAIDU_MAP_AK || '';

// 地图配置
const MAP_CONFIG = {
  // 湖北省中心点
  center: [114.305393, 30.593099],
  // 默认缩放级别（显示整个湖北省）
  defaultZoom: 8,
  // 最小缩放级别
  minZoom: 8,
  // 最大缩放级别
  maxZoom: 18,
  // 3D 倾斜角度
  defaultTilt: 60,
  // 默认旋转角度
  defaultHeading: 0,
};

interface HubeiMapGLProps {
  buildings: BuildingItem[];
}

declare global {
  interface Window {
    BMapGL: any;
  }
}

// 辅助函数：正确处理包含 Unicode 字符的 base64 编码
const toBase64 = (str: string): string => {
  try {
    return btoa(unescape(encodeURIComponent(str)));
  } catch (e) {
    return btoa(str);
  }
};

// 缓动函数集合
const easingFunctions = {
  // easeInOutCubic - 先慢后快再慢
  easeInOutCubic: (t: number): number => {
    return t < 0.5 ? 4 * t * t * t : 1 - Math.pow(-2 * t + 2, 3) / 2;
  },
  // easeInOutQuad - 平滑的加速减速
  easeInOutQuad: (t: number): number => {
    return t < 0.5 ? 2 * t * t : 1 - Math.pow(-2 * t + 2, 2) / 2;
  },
  // easeOutCubic - 快速开始，缓慢结束
  easeOutCubic: (t: number): number => {
    return 1 - Math.pow(1 - t, 3);
  },
};

const HubeiMapGL: React.FC<HubeiMapGLProps> = ({ buildings }) => {
  const mapRef = useRef<HTMLDivElement>(null);
  const mapInstanceRef = useRef<any>(null);
  const markersRef = useRef<any[]>([]);
  const infoWindowRef = useRef<any>(null);
  const polygonRef = useRef<any>(null);

  const [mapLoaded, setMapLoaded] = useState(false);
  const [loading, setLoading] = useState(true);
  const [hoveredBuildings, setHoveredBuildings] = useState<BuildingItem[]>([]);
  const [tooltipPosition, setTooltipPosition] = useState<{ x: number; y: number } | null>(null);
  const [selectedCluster, setSelectedCluster] = useState<ClusterGroup | null>(null);
  const [currentZoom, setCurrentZoom] = useState(8);
  const [is3DMode, setIs3DMode] = useState(false);
  const [currentTilt, setCurrentTilt] = useState(0);

  const {
    selectBuilding,
    navigateToBuilding,
  } = useVisualizationStore();

  // 湖北省边界坐标（简化版）
  const HUBEI_BOUNDARY = [
    [109.5, 33.5], [111.5, 33.5], [113.5, 33.2], [115.5, 33.0], [116.5, 32.5],
    [116.2, 31.8], [116.1, 31.0], [115.8, 30.2], [115.5, 29.5], [115.2, 29.0],
    [114.8, 28.5], [114.0, 28.2], [113.0, 28.0], [112.0, 28.2], [111.0, 28.5],
    [110.0, 29.0], [109.5, 29.8], [109.2, 30.5], [109.0, 31.2], [109.0, 32.0],
    [109.2, 32.5], [109.5, 33.5]
  ];

  // 获取湖北省官方边界（使用百度地图 Boundary API）
  const getOfficialHubeiBoundary = (BMapGL: any): Promise<string[]> => {
    return new Promise((resolve, reject) => {
      const bdary = new BMapGL.Boundary();
      bdary.get('湖北省', (rs: any) => {
        if (rs.boundaries && rs.boundaries.length > 0) {
          resolve(rs.boundaries);
        } else {
          reject(new Error('未获取到边界数据'));
        }
      });
    });
  };

  // 将边界字符串转换为 BMapGL.Point 数组
  const parseBoundaryPoints = (boundaryStr: string, BMapGL: any): any[] => {
    return boundaryStr.split(';').map((point: string) => {
      const [lng, lat] = point.split(',').map(Number);
      return new BMapGL.Point(lng, lat);
    });
  };

  // 添加湖北省边界轮廓
  const addHubeiMask = async (map: any, BMapGL: any) => {
    try {
      const boundaries = await getOfficialHubeiBoundary(BMapGL);
      boundaries.forEach((boundaryStr) => {
        const points = parseBoundaryPoints(boundaryStr, BMapGL);
        const polygon = new BMapGL.Polygon(points, {
          strokeColor: '#1890ff',
          strokeWeight: 4,
          strokeOpacity: 1,
          fillColor: 'transparent',
          fillOpacity: 0,
        });
        map.addOverlay(polygon);
      });
    } catch (error) {
      console.warn('获取官方边界失败，使用简化边界:', error);
      addFallbackMask(map, BMapGL);
    }
  };

  // 降级方案：使用简化的手动边界
  const addFallbackMask = (map: any, BMapGL: any) => {
    const points = HUBEI_BOUNDARY.map(coord => new BMapGL.Point(coord[0], coord[1]));
    const polygon = new BMapGL.Polygon(points, {
      strokeColor: '#1890ff',
      strokeWeight: 4,
      strokeOpacity: 0.8,
      fillColor: 'transparent',
      fillOpacity: 0,
    });
    map.addOverlay(polygon);
    polygonRef.current = polygon;
  };

  /**
   * 动画过渡函数
   * @param map 地图实例
   * @param target 目标状态
   * @param duration 动画时长（毫秒）
   */
  const animateViewTransition = useCallback((
    map: any,
    target: {
      center?: [number, number];
      zoom?: number;
      tilt?: number;
      heading?: number;
    },
    duration = 1500
  ) => {
    const startTime = Date.now();
    const startState = {
      center: map.getCenter(),
      zoom: map.getZoom(),
      tilt: map.getTilt(),
      heading: map.getHeading(),
    };

    const animate = () => {
      const now = Date.now();
      const elapsed = now - startTime;
      const progress = Math.min(elapsed / duration, 1);

      // 使用缓动函数
      const eased = easingFunctions.easeInOutCubic(progress);

      // 插值计算当前状态
      if (target.center) {
        const lng = startState.center.lng + (target.center[0] - startState.center.lng) * eased;
        const lat = startState.center.lat + (target.center[1] - startState.center.lat) * eased;
        map.setCenter(new window.BMapGL.Point(lng, lat));
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
    };

    animate();
  }, []);

  // 切换3D视角
  const toggle3DMode = useCallback(() => {
    const map = mapInstanceRef.current;
    if (!map) return;

    const targetTilt = is3DMode ? 0 : MAP_CONFIG.defaultTilt;
    setIs3DMode(!is3DMode);

    animateViewTransition(map, { tilt: targetTilt }, 1200);

    // 更新当前倾斜角度状态
    setTimeout(() => {
      setCurrentTilt(targetTilt);
    }, 1200);
  }, [is3DMode, animateViewTransition]);

  // 重置视图
  const resetView = useCallback(() => {
    const map = mapInstanceRef.current;
    if (!map) return;

    setIs3DMode(false);
    setCurrentTilt(0);

    animateViewTransition(map, {
      center: MAP_CONFIG.center as [number, number],
      zoom: MAP_CONFIG.defaultZoom,
      tilt: 0,
      heading: 0,
    });
  }, [animateViewTransition]);

  // 初始化地图
  useEffect(() => {
    if (!mapRef.current || !BAIDU_MAP_AK) {
      if (!BAIDU_MAP_AK) {
        message.error('未配置百度地图 AK');
      }
      return;
    }

    const initMap = async () => {
      try {
        setLoading(true);
        await loadBaiduMapGLScript(BAIDU_MAP_AK);

        if (!mapRef.current) return;

        // 创建 WebGL 地图实例
        const map = new window.BMapGL.Map(mapRef.current, {
          enableMapClick: false,
          showControls: false, // 隐藏默认控件，自定义添加
        });
        mapInstanceRef.current = map;

        // 设置中心点和缩放级别
        const point = new window.BMapGL.Point(MAP_CONFIG.center[0], MAP_CONFIG.center[1]);
        map.centerAndZoom(point, MAP_CONFIG.defaultZoom);

        // 限制缩放级别
        map.setMinZoom(MAP_CONFIG.minZoom);
        map.setMaxZoom(MAP_CONFIG.maxZoom);

        // 设置地图样式
        map.setMapStyleV2({
          styleId: 'c6cd6c1c7b622236b02f5fcc53e7f18', // 简约清新风格
        });

        // 启用滚轮缩放和双击缩放
        map.enableScrollWheelZoom(true);
        map.enableDoubleClickZoom(true);

        // 添加自定义控件
        addMapControls(map);

        // 创建信息窗口实例
        infoWindowRef.current = new window.BMapGL.InfoWindow('', {
          width: 280,
          height: 0,
          title: '',
          enableAutoPan: true,
          enableCloseOnClick: true,
        });

        // 添加湖北省边界轮廓
        addHubeiMask(map, window.BMapGL).catch(() => {
          // 静默处理
        });

        // 监听缩放变化
        const handleZoomEnd = () => {
          const zoom = map.getZoom();
          setCurrentZoom(zoom);
        };
        map.addEventListener('zoomend', handleZoomEnd);

        // 监听倾斜角度变化
        const handleTiltEnd = () => {
          const tilt = map.getTilt();
          setCurrentTilt(tilt);
        };
        map.addEventListener('tiltend', handleTiltEnd);

        setMapLoaded(true);
        setLoading(false);
      } catch (error) {
        message.error('百度地图GL加载失败，请检查 AK 配置');
        console.error(error);
        setLoading(false);
      }
    };

    initMap();
  }, []);

  // 添加地图控件
  const addMapControls = (map: any) => {
    // 缩放控件
    const zoomControl = new window.BMapGL.ZoomControl({
      anchor: window.BMAPGL_ANCHOR_BOTTOM_RIGHT,
      offset: new window.BMapGL.Size(20, 50),
    });
    map.addControl(zoomControl);

    // 比例尺控件
    const scaleControl = new window.BMapGL.ScaleControl({
      anchor: window.BMAPGL_ANCHOR_BOTTOM_LEFT,
      offset: new window.BMapGL.Size(80, 30),
    });
    map.addControl(scaleControl);
  };

  // 渲染楼宇标记（与原版逻辑相同）
  useEffect(() => {
    if (!mapInstanceRef.current || !mapLoaded) return;

    const map = mapInstanceRef.current;
    const BMapGL = window.BMapGL;

    // 清除现有标记
    markersRef.current.forEach(marker => map.removeOverlay(marker));
    markersRef.current = [];

    // 根据缩放级别过滤楼宇层级
    const filteredBuildings = currentZoom === 10
      ? buildings
      : buildings.filter((b) => b.level === 1);

    const CLUSTER_PIXEL_THRESHOLD = 40;

    const buildingsWithCoords = filteredBuildings.filter((b) => b.longitude && b.latitude);
    const clusterGroups: ClusterGroup[] = [];
    const processedBuildings = new Set<string>();

    buildingsWithCoords.forEach((building) => {
      if (processedBuildings.has(building.id)) return;

      const buildingPoint = new BMapGL.Point(building.longitude!, building.latitude!);
      const buildingPixel = map.pointToOverlayPixel(buildingPoint);

      const overlappedBuildings: BuildingItem[] = [building];
      const pixels: Array<{ x: number; y: number }> = [buildingPixel];

      buildingsWithCoords.forEach((otherBuilding) => {
        if (otherBuilding.id === building.id || processedBuildings.has(otherBuilding.id)) return;

        const otherPoint = new BMapGL.Point(otherBuilding.longitude!, otherBuilding.latitude!);
        const otherPixel = map.pointToOverlayPixel(otherPoint);

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

      const avgPixelX = pixels.reduce((sum, p) => sum + p.x, 0) / pixels.length;
      const avgPixelY = pixels.reduce((sum, p) => sum + p.y, 0) / pixels.length;

      const centerPoint = map.pixelToPointToOverlayPixel?.(new BMapGL.Pixel(avgPixelX, avgPixelY))
        || map.overlayPixelToPoint(new BMapGL.Pixel(avgPixelX, avgPixelY));

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
      renderClusterMarker(cluster, map, BMapGL);
    });
  }, [mapLoaded, buildings, currentZoom]);

  // 渲染聚类标记
  const renderClusterMarker = (cluster: ClusterGroup, map: any, BMapGL: any) => {
    const { buildings, clusterLng, clusterLat, centerPixel } = cluster;
    const isCluster = buildings.length > 1;

    const point = new BMapGL.Point(clusterLng, clusterLat);

    if (isCluster) {
      const size = 48 + Math.min(buildings.length * 2, 24);
      const clusterIcon = new BMapGL.Icon(
        'data:image/svg+xml;base64,' + toBase64(`
          <svg xmlns="http://www.w3.org/2000/svg" width="${size}" height="${size}" viewBox="0 0 ${size} ${size}">
            <defs>
              <filter id="clusterShadow" x="-50%" y="-50%" width="200%" height="200%">
                <feDropShadow dx="0" dy="2" stdDeviation="3" flood-color="rgba(255, 77, 79, 0.5)" flood-opacity="0.7"/>
              </filter>
            </defs>
            <circle cx="${size/2}" cy="${size/2}" r="${size/2-1}" fill="rgba(255,77,79,0.1)"/>
            <circle cx="${size/2}" cy="${size/2}" r="${size/2-4}" fill="#ff4d4f" stroke="#fff" stroke-width="3" filter="url(#clusterShadow)"/>
            <text x="${size/2}" y="${size/2 + 8}" text-anchor="middle" fill="white" font-size="${size/3}" font-weight="bold">${buildings.length}</text>
          </svg>
        `),
        new BMapGL.Size(size, size),
        new BMapGL.Size(size / 2, size / 2)
      );

      const marker = new BMapGL.Marker(point, { icon: clusterIcon });

      marker.addEventListener('mouseover', () => {
        showTooltip(buildings, centerPixel);
      });

      marker.addEventListener('mouseout', () => {
        hideTooltip();
      });

      marker.addEventListener('click', () => {
        setSelectedCluster(cluster);
      });

      map.addOverlay(marker);
      markersRef.current.push(marker);
    } else {
      const building = buildings[0];
      const isStopped = building.status === 1;

      const mainColor = isStopped ? '#9e9e9e' : '#ff6b35';
      const borderColor = isStopped ? '#757575' : '#ff4d4f';
      const shadowColor = isStopped ? 'rgba(0,0,0,0.2)' : 'rgba(255, 107, 53, 0.5)';

      const buildingLabel = building.name ? building.name.slice(0, 2) : '楼宇';

      const marker = new BMapGL.Marker(point, {
        icon: new BMapGL.Icon(
          'data:image/svg+xml;base64,' + toBase64(`
            <svg xmlns="http://www.w3.org/2000/svg" width="42" height="48" viewBox="0 0 42 48">
              <defs>
                <filter id="shadow" x="-50%" y="-50%" width="200%" height="200%">
                  <feDropShadow dx="0" dy="2" stdDeviation="2" flood-color="${shadowColor}" flood-opacity="0.6"/>
                </filter>
              </defs>
              <path d="M21 4 L21 4 Q21 4 21 4 C9 4 4 14 4 22 C4 32 21 44 21 44 C21 44 38 32 38 22 C38 14 33 4 21 4 Z"
                fill="${mainColor}" stroke="${borderColor}" stroke-width="3" filter="url(#shadow)"/>
              <circle cx="21" cy="20" r="8" fill="white" opacity="0.3"/>
              <text x="21" y="25" text-anchor="middle" fill="white" font-size="11" font-weight="bold">${buildingLabel}</text>
            </svg>
          `),
          new BMapGL.Size(42, 48),
          new BMapGL.Size(21, 48)
        ),
      });

      marker.addEventListener('mouseover', () => {
        showTooltip([building], centerPixel);
      });

      marker.addEventListener('mouseout', () => {
        hideTooltip();
      });

      marker.addEventListener('click', () => {
        showBuildingInfo(building, map, BMapGL);
      });

      map.addOverlay(marker);
      markersRef.current.push(marker);
    }
  };

  // 显示悬停提示
  const showTooltip = (buildings: BuildingItem[], point: any) => {
    setHoveredBuildings(buildings);
    setTooltipPosition({ x: point.x, y: point.y });
  };

  // 隐藏悬停提示
  const hideTooltip = () => {
    setHoveredBuildings([]);
    setTooltipPosition(null);
  };

  // 显示单个楼宇信息
  const showBuildingInfo = (building: BuildingItem, map: any, BMapGL: any) => {
    const isStopped = building.status === 1;
    const statusColor = isStopped ? '#9e9e9e' : '#ff6b35';
    const statusBg = isStopped ? '#f5f5f5' : '#fff2e8';

    const content = `
      <div style="padding: 12px; min-width: 260px;">
        <h3 style="margin: 0 0 12px 0; font-size: 16px; color: #1890ff; border-bottom: 2px solid #1890ff; padding-bottom: 8px;">
          ${building.name}
        </h3>
        <div style="font-size: 13px; color: #666; line-height: 1.8;">
          <div>📌 ${building.address || '暂无地址'}</div>
          <div style="margin-top: 8px;">
            <span style="display: inline-block; padding: 2px 10px; background: ${statusBg}; color: ${statusColor}; border-radius: 12px; font-size: 12px; font-weight: 500;">
              ${isStopped ? '⏸ 已停用' : '✓ 正常'}
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

    infoWindowRef.current.setContent(content);
    const point = new BMapGL.Point(building.longitude!, building.latitude!);
    map.openInfoWindow(infoWindowRef.current, point);
  };

  // 全局函数：查看楼宇详情
  useEffect(() => {
    (window as any).viewBuildingDetailsGL = (buildingId: string) => {
      const building = buildings.find(b => b.id === buildingId);
      if (building) {
        navigateToBuilding({
          ...building,
          orgId: '',
          orderNum: 0,
          createdAt: '',
          updatedAt: '',
        } as any);
      }
    };

    return () => {
      delete (window as any).viewBuildingDetailsGL;
    };
  }, [buildings, navigateToBuilding]);

  // 如果没有配置 AK，显示提示
  if (!BAIDU_MAP_AK) {
    return (
      <div style={{
        display: 'flex',
        justifyContent: 'center',
        alignItems: 'center',
        height: '100vh',
        flexDirection: 'column',
        gap: 16,
      }}>
        <div style={{ fontSize: 18, color: '#ff4d4f' }}>
          未配置百度地图 AK
        </div>
        <div style={{ color: '#666' }}>
          请在 .env.development 文件中配置 VITE_BAIDU_MAP_AK
        </div>
      </div>
    );
  }

  return (
    <div style={{ height: '100vh', width: '100%', position: 'relative' }}>
      {loading && (
        <div
          style={{
            position: 'absolute',
            top: '50%',
            left: '50%',
            transform: 'translate(-50%, -50%)',
            zIndex: 1000,
            textAlign: 'center',
          }}
        >
          <Spin size="large" />
          <div style={{ marginTop: 8 }}>加载 WebGL 地图中...</div>
        </div>
      )}

      {/* 地图容器 */}
      <div
        ref={mapRef}
        style={{ height: '100vh', width: '100%' }}
        onClick={() => {
          selectBuilding(null);
        }}
      />

      {/* 3D模式切换按钮 */}
      <div
        style={{
          position: 'absolute',
          top: 16,
          right: 16,
          zIndex: 100,
          display: 'flex',
          flexDirection: 'column',
          gap: 8,
        }}
      >
        <Button
          type={is3DMode ? 'primary' : 'default'}
          icon={is3DMode ? <EyeOutlined /> : <EyeInvisibleOutlined />}
          onClick={toggle3DMode}
        >
          {is3DMode ? '3D 视角' : '2D 视角'}
        </Button>
        <Button
          icon={<CompressOutlined />}
          onClick={resetView}
        >
          重置
        </Button>
      </div>

      {/* 视角信息 */}
      <div
        style={{
          position: 'absolute',
          top: 16,
          left: 16,
          background: 'rgba(255, 255, 255, 0.95)',
          padding: '12px 16px',
          borderRadius: '8px',
          boxShadow: '0 2px 8px rgba(0, 0, 0, 0.1)',
          fontSize: 12,
          color: '#666',
          zIndex: 100,
        }}
      >
        <div style={{ fontWeight: 'bold', marginBottom: 8, fontSize: 14 }}>
          <EnvironmentOutlined /> 地图信息
        </div>
        <div>缩放级别: <Badge count={currentZoom} showZero style={{ backgroundColor: '#1890ff' }} /></div>
        <div>
          3D 视角:
          <Badge
            count={is3DMode ? `${currentTilt}°` : '关闭'}
            showZero
            style={{
              backgroundColor: is3DMode ? '#722ed1' : '#d9d9d9',
              marginLeft: 8,
            }}
          />
        </div>
        {is3DMode && (
          <div style={{ fontSize: 11, color: '#999', marginTop: 4 }}>
            拖拽地图旋转视角
          </div>
        )}
      </div>

      {/* 悬停提示 */}
      {hoveredBuildings.length > 0 && tooltipPosition && (
        <div
          style={{
            position: 'absolute',
            left: tooltipPosition.x,
            top: tooltipPosition.y,
            transform: 'translate(-50%, -100%)',
            marginTop: -10,
            background: 'white',
            border: '1px solid #d9d9d9',
            borderRadius: '4px',
            padding: '8px 12px',
            boxShadow: '0 2px 8px rgba(0, 0, 0, 0.15)',
            zIndex: 1000,
            minWidth: '200px',
            pointerEvents: 'none',
          }}
        >
          {hoveredBuildings.length === 1 ? (
            <div>
              <div style={{ fontWeight: 'bold', color: '#1890ff', marginBottom: 4 }}>
                {hoveredBuildings[0].name}
              </div>
              <div style={{ fontSize: 11, color: '#666' }}>
                {hoveredBuildings[0].floorCount || 0}层 · {hoveredBuildings[0].workstationCount || 0}工位
              </div>
            </div>
          ) : (
            <div>
              <div style={{ fontWeight: 'bold', color: '#ff4d4f', marginBottom: 4 }}>
                {hoveredBuildings.length} 栋楼宇
              </div>
              {hoveredBuildings.slice(0, 3).map((b, i) => (
                <div key={i} style={{ fontSize: 11, color: '#666' }}>
                  · {b.name}
                </div>
              ))}
              {hoveredBuildings.length > 3 && (
                <div style={{ fontSize: 11, color: '#999' }}>
                  ...还有 {hoveredBuildings.length - 3} 栋
                </div>
              )}
            </div>
          )}
        </div>
      )}

      {/* 地图控制提示 */}
      <div
        style={{
          position: 'absolute',
          bottom: 16,
          left: 16,
          background: 'rgba(255, 255, 255, 0.95)',
          padding: '12px 16px',
          borderRadius: '8px',
          boxShadow: '0 2px 8px rgba(0, 0, 0, 0.1)',
          fontSize: 12,
          color: '#666',
          zIndex: 100,
        }}
      >
        <div style={{ fontWeight: 'bold', marginBottom: 8, fontSize: 14 }}>操作提示</div>
        {is3DMode ? (
          <>
            <div>🎮 当前为 3D 视角模式</div>
            <div style={{ fontSize: 11, color: '#999', marginTop: 4 }}>
              • 右键拖拽旋转视角
            </div>
            <div style={{ fontSize: 11, color: '#999' }}>
              • 滚轮缩放地图
            </div>
            <div style={{ fontSize: 11, color: '#999' }}>
              • 双击重置视角
            </div>
          </>
        ) : (
          <>
            <div>📊 当前为 2D 视角模式</div>
            <div>👆 滚轮缩放切换层级</div>
            <div>👆 悬停查看简要信息</div>
            <div>🖱️ 点击查看详细信息</div>
          </>
        )}

        <div style={{ marginTop: 8, padding: '8px', background: '#e6f7ff', borderRadius: '4px', border: '1px solid #91d5ff', color: '#096dd9' }}>
          <div style={{ fontWeight: 'bold', marginBottom: 4 }}>📋 WebGL 功能</div>
          <div style={{ fontSize: 11 }}>• 点击右上角按钮切换 2D/3D 视角</div>
          <div style={{ fontSize: 11 }}>• 3D 模式支持60度倾斜视角</div>
          <div style={{ fontSize: 11 }}>• 平滑动画过渡效果</div>
        </div>
      </div>

      {/* 聚类列表侧边栏（与原版相同） */}
      {selectedCluster && (
        <div
          style={{
            position: 'absolute',
            top: 0,
            right: 0,
            width: 320,
            height: '100vh',
            background: 'white',
            boxShadow: '-2px 0 8px rgba(0, 0, 0, 0.1)',
            zIndex: 1000,
            display: 'flex',
            flexDirection: 'column',
          }}
        >
          <div
            style={{
              padding: '16px',
              borderBottom: '1px solid #f0f0f0',
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
            }}
          >
            <div>
              <div style={{ fontWeight: 'bold', fontSize: 16, color: '#ff4d4f' }}>
                📍 {selectedCluster.buildings.length} 栋楼宇
              </div>
              <div style={{ fontSize: 12, color: '#999', marginTop: 4 }}>
                位置相近，已聚类显示
              </div>
            </div>
            <button
              onClick={() => setSelectedCluster(null)}
              style={{
                border: 'none',
                background: 'none',
                fontSize: 24,
                cursor: 'pointer',
                color: '#999',
                padding: 0,
                width: 32,
                height: 32,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
              }}
            >
              ×
            </button>
          </div>

          <div style={{ flex: 1, overflow: 'auto', padding: '16px' }}>
            <List
              dataSource={selectedCluster.buildings}
              renderItem={(building) => (
                <List.Item
                  key={building.id}
                  style={{
                    border: '1px solid #f0f0f0',
                    borderRadius: '8px',
                    marginBottom: '12px',
                    padding: '12px',
                    cursor: 'pointer',
                  }}
                  onClick={() => {
                    setSelectedCluster(null);
                    navigateToBuilding({
                      ...building,
                      orgId: '',
                      orderNum: 0,
                      createdAt: '',
                      updatedAt: '',
                    } as any);
                  }}
                >
                  <div style={{ width: '100%' }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
                      <div style={{ fontWeight: 'bold', color: '#1890ff', fontSize: 14 }}>
                        {building.name}
                      </div>
                      <Tag color={building.status === 1 ? 'red' : 'green'}>
                        {building.status === 1 ? '已停用' : '正常'}
                      </Tag>
                    </div>

                    <div style={{ fontSize: 12, color: '#666', lineHeight: 1.8 }}>
                      <div>📌 {building.address || '暂无地址'}</div>
                      <div style={{ marginTop: 4 }}>
                        <span>🏢 {building.floorCount || 0} 层</span>
                        <span style={{ marginLeft: 12 }}>🪑 {building.workstationCount || 0} 工位</span>
                      </div>
                    </div>

                    <div style={{ marginTop: 12, textAlign: 'right' }}>
                      <Tag color="blue">查看详情 →</Tag>
                    </div>
                  </div>
                </List.Item>
              )}
            />
          </div>

          <div
            style={{
              padding: '12px 16px',
              borderTop: '1px solid #f0f0f0',
              background: '#fafafa',
              fontSize: 11,
              color: '#999',
              textAlign: 'center',
            }}
          >
            点击楼宇卡片查看 3D 详情
          </div>
        </div>
      )}
    </div>
  );
};

export default HubeiMapGL;
