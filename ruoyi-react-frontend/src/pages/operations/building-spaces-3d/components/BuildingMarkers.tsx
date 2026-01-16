/**
 * 楼宇标记组件（第二级）
 * 在地图上显示具体的楼宇位置标记
 */

import { CustomOverlay, InfoWindow } from '@uiw/react-baidu-map';
import { useCallback, useState } from 'react';
import type { BuildingItem } from '../types';
import { getMapMarkerColor } from '@/utils/three/colors';
import { BuildOutlined, ApartmentOutlined } from '@ant-design/icons';

interface BuildingMarkersProps {
  buildings: BuildingItem[];
  selectedBuilding: any | null;
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

  const handleMarkerClick = useCallback(
    (building: BuildingItem) => {
      onBuildingClick(building);
      setActiveWindow(building.id);
    },
    [onBuildingClick]
  );

  return (
    <>
      {buildings
        .filter((b) => b.longitude && b.latitude) // 只显示有坐标的楼宇
        .map((building) => {
          const isSelected = selectedBuilding?.id === building.id;
          const isActive = activeWindow === building.id;
          const isStopped = building.status === 1;
          const colors = getMapMarkerColor('building', building.status) as any;
          // 根据状态确定主颜色
          const mainColor = isStopped
            ? colors.stopped || '#9e9e9e'
            : colors.normal || colors.main || '#52c41a';

          return (
            <div key={building.id}>
              <CustomOverlay
                position={{ lng: building.longitude!, lat: building.latitude! }}
              >
                {/* 自定义标记样式 */}
                <div
                  onClick={() => handleMarkerClick(building)}
                  style={{
                    width: '24px',
                    height: '24px',
                    borderRadius: '4px',
                    background: isSelected ? '#ff4d4f' : mainColor,
                    border: `2px solid ${isSelected ? '#ffccc7' : colors.border}`,
                    boxShadow: `0 2px 8px ${colors.bg}`,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    cursor: 'pointer',
                    transition: 'all 0.3s',
                  }}
                >
                  <BuildOutlined style={{ color: '#fff', fontSize: 12 }} />
                </div>
              </CustomOverlay>

              {/* 信息窗口 */}
              {isActive && (
                <InfoWindow
                  position={{ lng: building.longitude!, lat: building.latitude! }}
                  onClose={handleCloseWindow}
                  visible
                >
                  <div style={{ padding: '12px', minWidth: '250px' }}>
                    <h3 style={{ margin: '0 0 8px 0', fontSize: 16 }}>
                      {building.name}
                    </h3>
                    <div style={{ fontSize: 12, color: '#666', marginBottom: 4 }}>
                      <div>📍 {building.cityName}</div>
                      <div>📌 {building.address}</div>
                    </div>
                    <div style={{ display: 'flex', gap: 16, marginTop: 8, fontSize: 13 }}>
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
