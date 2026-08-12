/**
 * 楼宇标记组件（第二级）
 * 在地图上显示具体的楼宇位置标记
 */

import { CustomOverlay, InfoWindow } from "@uiw/react-baidu-map";
import { useCallback, useState } from "react";
import { BuildOutlined, ApartmentOutlined } from "@ant-design/icons";
import type { BuildingItem } from "./types";
import { getBuildingMarkerColors } from "./utils";

interface BuildingMarkersProps {
  buildings: BuildingItem[];
  selectedBuilding: BuildingItem | null;
  onBuildingClick: (building: BuildingItem) => void;
}

const BuildingMarkers: React.FC<BuildingMarkersProps> = ({
  buildings,
  selectedBuilding,
  onBuildingClick,
}) => {
  const [activeWindow, setActiveWindow] = useState<string | null>(null);

  const handleCloseWindow = useCallback(() => {
    setActiveWindow(null);
  }, []);

  const handleMarkerClick = useCallback((building: BuildingItem) => {
    onBuildingClick(building);
    setActiveWindow(building.id);
  }, [onBuildingClick]);

  // 过滤出有坐标的楼宇
  const buildingsWithCoords = buildings.filter((b) => b.longitude && b.latitude);

  return (
    <>
      {buildingsWithCoords.map((building) => {
        const isSelected = selectedBuilding?.id === building.id;
        const isActive = activeWindow === building.id;
        const colors = getBuildingMarkerColors(building.status);

        return (
          <div key={building.id}>
            <CustomOverlay
              position={{ lng: building.longitude!, lat: building.latitude! }}
            >
              <div
                onClick={() => handleMarkerClick(building)}
                style={{
                  width: "24px",
                  height: "24px",
                  borderRadius: "4px",
                  background: isSelected ? "#ff4d4f" : colors.main,
                  border: `2px solid ${isSelected ? "#ffccc7" : colors.border}`,
                  boxShadow: `0 2px 8px ${colors.shadow}`,
                  display: "flex",
                  alignItems: "center",
                  justifyContent: "center",
                  cursor: "pointer",
                  transition: "all 0.3s",
                }}
              >
                <BuildOutlined style={{ color: "var(--theme-text-inverse, #fff)", fontSize: 12 }} />
              </div>
            </CustomOverlay>

            {/* 信息窗口 */}
            {isActive && (
              <InfoWindow
                position={{ lng: building.longitude!, lat: building.latitude! }}
                onClose={handleCloseWindow}
                visible
              >
                <div style={{ padding: "12px", minWidth: "250px" }}>
                  <h3 style={{ margin: "0 0 8px 0", fontSize: 16 }}>
                    {building.name}
                  </h3>
                  <div style={{ fontSize: 12, color: "var(--theme-text-tertiary, #666)", marginBottom: 4 }}>
                    <div>📍 {building.cityName}</div>
                    <div>📌 {building.address}</div>
                  </div>
                  <div style={{ display: "flex", gap: 16, marginTop: 8, fontSize: 13 }}>
                    <div>
                      <ApartmentOutlined /> {building.floorCount || 0} 层
                    </div>
                    <div>
                      <BuildOutlined /> {building.workstationCount || 0} 工位
                    </div>
                  </div>
                </div>
              </InfoWindow>
            )}
          </div>
        );
      })}
    </>
  );
};

export default BuildingMarkers;
