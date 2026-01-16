/**
 * 城市标记组件（第一级）
 * 在地图上显示湖北省各城市的位置标记
 */

import { CustomOverlay, InfoWindow } from '@uiw/react-baidu-map';
import { useState, useCallback } from 'react';
import type { CityGroup, BuildingItem } from '../types';
import { getMapMarkerColor } from '@/utils/three/colors';
import { EnvironmentOutlined, BuildOutlined } from '@ant-design/icons';

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

  const handleMarkerClick = useCallback(
    (city: CityGroup) => {
      onCityClick(city.code);
      setActiveWindow(city.code);
    },
    [onCityClick]
  );

  return (
    <>
      {cities.map((city) => {
        const isSelected = selectedCity === city.code;
        const isActive = activeWindow === city.code;
        const colors = getMapMarkerColor('city');

        return (
          <div key={city.code}>
            <CustomOverlay
              position={{ lng: city.center[0], lat: city.center[1] }}
            >
              {/* 自定义标记样式 */}
              <div
                onClick={() => handleMarkerClick(city)}
                style={{
                  width: '32px',
                  height: '32px',
                  borderRadius: '50%',
                  background: (isSelected && 'selected' in colors ? (colors as any).selected?.main : (colors as any).main) || (colors as any).main,
                  border: `3px solid ${isSelected && 'selected' in colors ? (colors as any).selected?.border : (colors as any).border}` || (colors as any).border,
                  boxShadow: `0 2px 8px ${(colors as any).bg}`,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  cursor: 'pointer',
                  transition: 'all 0.3s',
                }}
              >
                <EnvironmentOutlined style={{ color: '#fff', fontSize: 16 }} />
              </div>
            </CustomOverlay>

            {/* 信息窗口 */}
            {isActive && (
              <InfoWindow
                position={{ lng: city.center[0], lat: city.center[1] }}
                onClose={handleCloseWindow}
                visible
              >
                <div style={{ padding: '12px', minWidth: '200px' }}>
                  <h3 style={{ margin: '0 0 8px 0', fontSize: 16 }}>{city.name}</h3>
                  <div style={{ display: 'flex', gap: 16, fontSize: 13 }}>
                    <div>
                      <BuildOutlined /> {city.buildingCount} 栋楼宇
                    </div>
                  </div>
                  {city.buildings.length > 0 && (
                    <div style={{ marginTop: 12, paddingTop: 12, borderTop: '1px solid #eee' }}>
                      <div style={{ fontSize: 12, color: '#999', marginBottom: 8 }}>
                        主要楼宇：
                      </div>
                      {city.buildings.slice(0, 3).map((building) => (
                        <div
                          key={building.id}
                          style={{
                            padding: '4px 8px',
                            cursor: 'pointer',
                            borderRadius: '4px',
                            transition: 'background 0.2s',
                          }}
                          onMouseEnter={(e) => {
                            e.currentTarget.style.background = '#f5f5f5';
                          }}
                          onMouseLeave={(e) => {
                            e.currentTarget.style.background = 'transparent';
                          }}
                          onClick={() => onBuildingClick(building)}
                        >
                          📍 {building.name}
                        </div>
                      ))}
                      {city.buildings.length > 3 && (
                        <div style={{ fontSize: 12, color: '#999', marginTop: 4 }}>
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
