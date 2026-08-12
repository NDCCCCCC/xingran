/**
 * 城市标记组件（第一级）
 * 在地图上显示湖北省各城市的位置标记
 */

import { CustomOverlay, InfoWindow } from "@uiw/react-baidu-map";
import { useState, useCallback, useMemo } from "react";
import { EnvironmentOutlined, BuildOutlined } from "@ant-design/icons";
import type { CityGroup, BuildingItem } from "./types";
import { MARKER_COLORS } from "./constants";

interface CityMarkersProps {
  cities: CityGroup[];
  selectedCity: string | null;
  onCityClick: (cityCode: string) => void;
  onBuildingClick: (building: BuildingItem) => void;
}

const CityMarkers: React.FC<CityMarkersProps> = ({
  cities,
  selectedCity,
  onCityClick,
  onBuildingClick,
}) => {
  const [activeWindow, setActiveWindow] = useState<string | null>(null);

  const handleCloseWindow = useCallback(() => {
    setActiveWindow(null);
  }, []);

  const handleMarkerClick = useCallback((city: CityGroup) => {
    onCityClick(city.code);
    setActiveWindow(city.code);
  }, [onCityClick]);

  const handleBuildingMouseEnter = useCallback((e: React.MouseEvent<HTMLDivElement>) => {
    e.currentTarget.style.background = "#f5f5f5";
  }, []);

  const handleBuildingMouseLeave = useCallback((e: React.MouseEvent<HTMLDivElement>) => {
    e.currentTarget.style.background = "transparent";
  }, []);

  return (
    <>
      {cities.map((city) => {
        const isSelected = selectedCity === city.code;
        const isActive = activeWindow === city.code;
        const showMainBuildings = city.buildings.slice(0, 3);
        const hasMoreBuildings = city.buildings.length > 3;

        return (
          <div key={city.code}>
            <CustomOverlay
              position={{ lng: city.center[0], lat: city.center[1] }}
            >
              <div
                onClick={() => handleMarkerClick(city)}
                style={{
                  width: "32px",
                  height: "32px",
                  borderRadius: "50%",
                  background: isSelected ? MARKER_COLORS.NORMAL.MAIN : MARKER_COLORS.BOUNDARY,
                  border: `3px solid ${isSelected ? MARKER_COLORS.NORMAL.BORDER : MARKER_COLORS.BOUNDARY}`,
                  boxShadow: `0 2px 8px ${MARKER_COLORS.NORMAL.SHADOW}`,
                  display: "flex",
                  alignItems: "center",
                  justifyContent: "center",
                  cursor: "pointer",
                  transition: "all 0.3s",
                }}
              >
                <EnvironmentOutlined style={{ color: "var(--theme-text-inverse, #fff)", fontSize: 16 }} />
              </div>
            </CustomOverlay>

            {/* 信息窗口 */}
            {isActive && (
              <InfoWindow
                position={{ lng: city.center[0], lat: city.center[1] }}
                onClose={handleCloseWindow}
                visible
              >
                <div style={{ padding: "12px", minWidth: "200px" }}>
                  <h3 style={{ margin: "0 0 8px 0", fontSize: 16 }}>{city.name}</h3>
                  <div style={{ display: "flex", gap: 16, fontSize: 13 }}>
                    <div>
                      <BuildOutlined /> {city.buildingCount} 栋楼宇
                    </div>
                  </div>
                  {city.buildings.length > 0 && (
                    <div style={{ marginTop: 12, paddingTop: 12, borderTop: "1px solid #eee" }}>
                      <div style={{ fontSize: 12, color: "var(--theme-text-tertiary, #999)", marginBottom: 8 }}>
                        主要楼宇：
                      </div>
                      {showMainBuildings.map((building) => (
                        <div
                          key={building.id}
                          style={{
                            padding: "4px 8px",
                            cursor: "pointer",
                            borderRadius: "4px",
                            transition: "background 0.2s",
                          }}
                          onMouseEnter={handleBuildingMouseEnter}
                          onMouseLeave={handleBuildingMouseLeave}
                          onClick={() => onBuildingClick(building)}
                        >
                          📍 {building.name}
                        </div>
                      ))}
                      {hasMoreBuildings && (
                        <div style={{ fontSize: 12, color: "var(--theme-text-tertiary, #999)", marginTop: 4 }}>
                          ...等 {city.buildings.length} 栋楼宇
                        </div>
                      )}
                    </div>
                  )}
                </div>
              </InfoWindow>
            )}
          </div>
        );
      })}
    </>
  );
};

export default CityMarkers;
