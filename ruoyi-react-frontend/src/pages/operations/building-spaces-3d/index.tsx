/**
 * 湖北省楼宇空间 3D 可视化页面
 * 支持切换百度地图普通版和WebGL版本
 */

import { useState, useEffect, useCallback } from 'react';
import { Spin, Alert, Switch } from 'antd';
import { LoadingOutlined } from '@ant-design/icons';
import { useVisualizationStore } from '@/store/visualizationStore';
import { buildingApi } from '@/lib/opsApi';
import { handleApiError } from '@/utils/errorHandler';
import HubeiMap from './components/HubeiMap';
import HubeiMapGL from './components/HubeiMapGL';
import BuildingView3D from './components/BuildingView3D';
import FloorView3D from './components/FloorView3D';
import type { BuildingItem } from './types';

const BuildingSpaces3D: React.FC = () => {
  const [loading, setLoading] = useState(true);
  const [buildings, setBuildings] = useState<BuildingItem[]>([]);
  const [useWebGL, setUseWebGL] = useState(true); // 默认使用WebGL版本

  const { viewLevel } = useVisualizationStore();

  // 加载楼宇数据
  const loadBuildings = useCallback(async () => {
    try {
      setLoading(true);
      const result = await buildingApi.list({ current: 1, pageSize: 1000 });
      const buildingList = result.data?.list || [];
      setBuildings(buildingList as unknown as BuildingItem[]);
    } catch (error) {
      handleApiError(error, '加载楼宇数据');
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
        <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh' }}>
          <Spin size="large" indicator={<LoadingOutlined style={{ fontSize: 48 }} spin />} />
          <div style={{ marginLeft: 16, fontSize: 16 }}>加载地图数据中...</div>
        </div>
      );
    }

    switch (viewLevel) {
      case 'map':
        return useWebGL ? (
          <HubeiMapGL buildings={buildings} />
        ) : (
          <HubeiMap buildings={buildings} />
        );
      case 'building':
        return <BuildingView3D />;
      case 'floor':
        return <FloorView3D />;
      default:
        return useWebGL ? (
          <HubeiMapGL buildings={buildings} />
        ) : (
          <HubeiMap buildings={buildings} />
        );
    }
  };

  return (
    <div style={{ height: '100vh', position: 'relative', overflow: 'hidden' }}>
      {/* 主视图 */}
      {renderView()}

      {/* WebGL版本切换开关 */}
      {viewLevel === 'map' && (
        <div
          style={{
            position: 'absolute',
            top: 16,
            left: 16,
            background: 'rgba(255, 255, 255, 0.95)',
            padding: '12px 16px',
            borderRadius: '8px',
            boxShadow: '0 2px 8px rgba(0, 0, 0, 0.1)',
            zIndex: 100,
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
            <span style={{ fontSize: 14, fontWeight: 'bold', color: '#1890ff' }}>
              🗺️ 地图版本
            </span>
            <Switch
              checked={useWebGL}
              onChange={(checked) => setUseWebGL(checked)}
              checkedChildren="WebGL"
              unCheckedChildren="普通"
            />
          </div>
          <div style={{ fontSize: 11, color: '#999', marginTop: 4 }}>
            {useWebGL ? 'WebGL 版本支持 3D 视角' : '普通版本兼容性更好'}
          </div>
        </div>
      )}

      {/* 提示信息 */}
      {(buildings.length === 0 && !loading) && (
        <Alert
          message="暂无楼宇数据"
          description="请先在楼宇管理中添加楼宇信息"
          type="info"
          showIcon
          style={{ position: 'absolute', top: 16, left: '50%', transform: 'translateX(-50%)', zIndex: 1000 }}
        />
      )}
    </div>
  );
};

export default BuildingSpaces3D;
