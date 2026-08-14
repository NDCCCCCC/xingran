/**
 * 湖北省楼宇空间 3D 可视化页面
 * 支持切换百度地图普通版和WebGL版本
 */

import { useState, useEffect, useCallback } from "react";
import { Spin, Alert, Switch } from "antd";
import { LoadingOutlined } from "@ant-design/icons";
import { useVisualizationStore } from "@/store/visualizationStore";
import { buildingApi } from "@/lib/opsApi";
import { handleApiError } from "@/utils/errorHandler";
import HubeiMap from "./components/HubeiMap";
import HubeiMapGL from "./components/HubeiMapGL";
import { BuildingView3DLazy, FloorView3DLazy } from "@/components/three/BuildingScene";
import type { BuildingItem } from "./types";

// 常量定义
const PAGE_SIZE = 1000;
const FULL_VH = "100vh";
const MAP_SWITCH_STYLE = {
  position: "absolute" as const,
  top: 16,
  left: 16,
  background: "rgba(255, 255, 255, 0.95)",
  padding: "12px 16px",
  borderRadius: "8px",
  boxShadow: "0 2px 8px rgba(0, 0, 0, 0.1)",
  zIndex: 100,
};

const BuildingSpaces3D: React.FC = () => {
  const [loading, setLoading] = useState(true);
  const [buildings, setBuildings] = useState<BuildingItem[]>([]);
  const [useWebGL, setUseWebGL] = useState(true);

  const { viewLevel } = useVisualizationStore();

  // 加载楼宇数据
  const loadBuildings = useCallback(async () => {
    try {
      setLoading(true);
      const result = await buildingApi.list({ current: 1, pageSize: PAGE_SIZE });
      // 后端返回的 Building 类型转换为 BuildingItem，补充缺失字段
      const buildingList = result.data?.list || [];
      setBuildings(
        buildingList.map((b) => ({
          id: b.id,
          name: b.name,
          code: b.code,
          cityCode: "",
          cityName: "",
          address: b.address || "",
          longitude: b.longitude,
          latitude: b.latitude,
          level: 2 as const,
          status: b.status,
        }))
      );
    } catch (error) {
      handleApiError(error, "加载楼宇数据");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadBuildings();
  }, [loadBuildings]);

  // 根据视图级别渲染不同内容
  const renderView = () => {
    if (loading) {
      return (
        <div
          style={{
            display: "flex",
            justifyContent: "center",
            alignItems: "center",
            height: FULL_VH,
          }}
        >
          <Spin size="large" indicator={<LoadingOutlined style={{ fontSize: 48 }} spin />} />
          <div style={{ marginLeft: 16, fontSize: 16 }}>加载地图数据中...</div>
        </div>
      );
    }

    const MapComponent = useWebGL ? HubeiMapGL : HubeiMap;

    switch (viewLevel) {
      case "map":
        return <MapComponent buildings={buildings} />;
      case "building":
        return <BuildingView3DLazy />;
      case "floor":
        return <FloorView3DLazy />;
      default:
        return <MapComponent buildings={buildings} />;
    }
  };

  // 渲染地图版本切换开关
  const renderMapSwitch = () => {
    if (viewLevel !== "map") {
      return null;
    }

    return (
      <div style={MAP_SWITCH_STYLE}>
        <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
          <span
            style={{ fontSize: 14, fontWeight: "bold", color: "var(--theme-text-accent, #1890ff)" }}
          >
            🗺️ 地图版本
          </span>
          <Switch
            checked={useWebGL}
            onChange={setUseWebGL}
            checkedChildren="WebGL"
            unCheckedChildren="普通"
          />
        </div>
        <div style={{ fontSize: 11, color: "var(--theme-text-tertiary, #999)", marginTop: 4 }}>
          {useWebGL ? "WebGL 版本支持 3D 视角" : "普通版本兼容性更好"}
        </div>
      </div>
    );
  };

  return (
    <div style={{ height: FULL_VH, position: "relative", overflow: "hidden" }}>
      {renderView()}
      {renderMapSwitch()}

      {!loading && buildings.length === 0 && (
        <Alert
          message="暂无楼宇数据"
          description="请先在楼宇管理中添加楼宇信息"
          type="info"
          showIcon
          style={{
            position: "absolute",
            top: 16,
            left: "50%",
            transform: "translateX(-50%)",
            zIndex: 1000,
          }}
        />
      )}
    </div>
  );
};

export default BuildingSpaces3D;
