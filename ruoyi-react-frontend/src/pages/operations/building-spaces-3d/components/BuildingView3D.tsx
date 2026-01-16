/**
 * 3D 楼宇视图组件
 * 点击地图上的楼宇标记后，显示该楼宇的 3D 模型
 */

import { useState, useCallback, useEffect } from 'react';
import { Button, Space, Spin, Alert, Tag, Empty, Select } from 'antd';
import { ArrowLeftOutlined, DesktopOutlined, BuildOutlined } from '@ant-design/icons';
import { useVisualizationStore } from '@/store/visualizationStore';
import { floorApi, workstationApi, buildingApi } from '@/lib/opsApi';
import { handleApiError } from '@/utils/errorHandler';
import BuildingModel3D from './BuildingModel3D';
import FloorPlan3D from './FloorPlan3D';

interface FloorData {
  id: string;
  name: string;
  floorNo: string;
  workstationCount: number;
  status: number; // 0=正常, 1=停用
}

interface WorkstationData {
  id: string;
  name: string;
  code: string;
  status: number; // 0=空闲, 1=占用, 2=维护
  type: number; // 0=固定, 1=灵活, 2=管理
  positionX?: number;
  positionY?: number;
  rotation?: number; // 旋转角度
}

interface BuildingData {
  id: string;
  name: string;
  cityName?: string;
  address?: string;
  status: number;
}

const BuildingView3D: React.FC = () => {
  const [loading, setLoading] = useState(true);
  const [buildings, setBuildings] = useState<BuildingData[]>([]);
  const [currentBuilding, setCurrentBuilding] = useState<BuildingData | null>(null);
  const [floors, setFloors] = useState<FloorData[]>([]);
  const [selectedFloor, setSelectedFloor] = useState<FloorData | null>(null);
  const [workstations, setWorkstations] = useState<WorkstationData[]>([]);
  const [loadingWorkstations, setLoadingWorkstations] = useState(false);

  const { selectedBuilding, navigateToMap } = useVisualizationStore();

  // 加载楼宇列表
  const loadBuildings = useCallback(async () => {
    try {
      const result = await buildingApi.list({
        current: 1,
        pageSize: 100,
      });
      setBuildings(result.data?.list || []);
    } catch (error) {
      handleApiError(error, '加载楼宇列表');
    }
  }, []);

  // 加载楼层数据
  const loadFloors = useCallback(async (buildingId?: string) => {
    const targetBuildingId = buildingId || selectedBuilding?.id;
    if (!targetBuildingId) return;

    try {
      setLoading(true);
      const result = await floorApi.list({
        buildingCode: targetBuildingId,
        current: 1,
        pageSize: 100,
      });

      const floorList = result.data?.list || [];

      // 获取每层楼的工位数量
      const floorsWithCount = await Promise.all(
        floorList.map(async (floor: any) => {
          try {
            const wsResult = await workstationApi.list({
              floorCode: floor.id,
              current: 1,
              pageSize: 1,
            });
            return {
              ...floor,
              workstationCount: wsResult.data?.total || 0,
            };
          } catch {
            return {
              ...floor,
              workstationCount: 0,
            };
          }
        })
      );

      setFloors(floorsWithCount);
    } catch (error) {
      handleApiError(error, '加载楼层数据');
    } finally {
      setLoading(false);
    }
  }, [selectedBuilding]);

  useEffect(() => {
    loadBuildings();
    if (selectedBuilding) {
      setCurrentBuilding(selectedBuilding);
      loadFloors(selectedBuilding.id);
    }
  }, [selectedBuilding, loadBuildings, loadFloors]);

  // 处理楼宇切换
  const handleBuildingChange = useCallback(
    async (buildingId: string) => {
      const building = buildings.find(b => b.id === buildingId);
      if (building) {
        setCurrentBuilding(building);
        setSelectedFloor(null);
        setWorkstations([]);
        await loadFloors(buildingId);
      }
    },
    [buildings, loadFloors]
  );

  // 加载选中楼层的工位数据
  const loadWorkstations = useCallback(async (floor: FloorData) => {
    try {
      setLoadingWorkstations(true);
      const result = await workstationApi.list({
        floorCode: floor.id,
        current: 1,
        pageSize: 500,
      });

      // 转换工位数据，添加缺失的 code 字段
      const workstationList = result.data?.list || [];
      const convertedWorkstations: WorkstationData[] = workstationList.map((ws) => ({
        id: ws.id,
        name: ws.name,
        code: ws.name, // 使用 name 作为 code（因为 API 不返回 code 字段）
        status: ws.status as number,
        type: ws.type as number,
        positionX: ws.positionX,
        positionY: ws.positionY,
        rotation: 0, // 默认旋转角度
      }));

      setWorkstations(convertedWorkstations);
    } catch (error) {
      handleApiError(error, '加载工位数据');
      setWorkstations([]);
    } finally {
      setLoadingWorkstations(false);
    }
  }, []);

  // 处理楼层点击
  const handleFloorClick = useCallback(
    (floor: FloorData) => {
      setSelectedFloor(floor);
      loadWorkstations(floor);
    },
    [loadWorkstations]
  );

  // 如果没有选中的楼宇，返回地图
  if (!selectedBuilding) {
    return (
      <div style={{ height: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        <Alert
          message="未选择楼宇"
          description="请先在地图上选择一个楼宇"
          type="info"
          showIcon
          action={
            <Button type="primary" onClick={navigateToMap}>
              返回地图
            </Button>
          }
        />
      </div>
    );
  }

  const statusText = selectedBuilding.status === 0 ? '正常' : '停用';
  const statusColor = selectedBuilding.status === 0 ? 'success' : 'default';

  return (
    <div style={{ height: '100vh', display: 'flex', flexDirection: 'column', overflow: 'hidden', background: '#fff' }}>
      {/* 顶部导航栏 */}
      <div
        style={{
          background: '#fff',
          borderBottom: '1px solid #e8e8e8',
          padding: '12px 24px',
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          zIndex: 100,
          boxShadow: '0 2px 8px rgba(0,0,0,0.06)',
        }}
      >
        <Space size="large">
          <Button icon={<ArrowLeftOutlined />} onClick={navigateToMap} type="text">
            返回地图
          </Button>
          <div>
            <Space>
              <BuildOutlined style={{ color: '#1890ff' }} />
              <span style={{ fontSize: 16, fontWeight: 'bold', color: '#262626' }}>{selectedBuilding.name}</span>
              <Tag color={statusColor}>{statusText}</Tag>
              {selectedFloor && (
                <>
                  <span style={{ color: '#8c8c8c' }}>→</span>
                  <Tag color="blue">{selectedFloor.name || selectedFloor.floorNo}</Tag>
                </>
              )}
            </Space>
          </div>
        </Space>

        <div style={{ fontSize: 12, color: '#8c8c8c' }}>
          <Space split={<span style={{ color: '#d9d9d9' }}>|</span>}>
            <span>{selectedBuilding.cityName || '-'}</span>
            <span>{floors.length} 层</span>
            <span>{floors.reduce((sum, f) => sum + (f.workstationCount || 0), 0)} 工位</span>
          </Space>
        </div>
      </div>

      {/* 主内容区：左右分栏 */}
      <div style={{ flex: 1, display: 'flex', overflow: 'hidden' }}>
        {loading ? (
          <div style={{
            flex: 1,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            background: '#f5f5f5'
          }}>
            <Spin size="large" tip="加载楼层数据..." />
          </div>
        ) : (
          <>
            {/* 左侧：3D 楼宇模型 */}
            <div style={{
              flex: selectedFloor ? '3' : '1',
              position: 'relative',
              background: '#f5f5f5',
              borderRight: selectedFloor ? '1px solid #e8e8e8' : 'none',
              transition: 'flex 0.3s ease',
              minWidth: 0,
              overflow: 'hidden',
              display: 'flex',
              flexDirection: 'column',
            }}>
              {/* 楼宇切换下拉框 */}
              {buildings.length > 1 && (
                <div style={{
                  position: 'absolute',
                  top: 16,
                  left: 16,
                  zIndex: 10,
                  background: 'rgba(255, 255, 255, 0.95)',
                  backdropFilter: 'blur(10px)',
                  borderRadius: 8,
                  padding: '8px 12px',
                  boxShadow: '0 2px 8px rgba(0,0,0,0.1)',
                  border: '1px solid #e8e8e8',
                  minWidth: 200,
                }}>
                  <div style={{ fontSize: 12, color: '#8c8c8c', marginBottom: 4 }}>
                    切换楼宇
                  </div>
                  <Select
                    style={{ width: '100%' }}
                    placeholder="选择楼宇"
                    value={currentBuilding?.id}
                    onChange={handleBuildingChange}
                    options={buildings.map(b => ({
                      label: b.name,
                      value: b.id,
                    }))}
                  />
                </div>
              )}

              {/* 3D 楼宇模型 */}
              <div style={{ height: '100%', width: '100%' }}>
                <BuildingModel3D floors={floors} onFloorClick={handleFloorClick} selectedFloorId={selectedFloor?.id} />
              </div>
            </div>

            {/* 右侧：工位平面图 */}
            {selectedFloor && (
              <div style={{
                flex: '7',
                display: 'flex',
                flexDirection: 'column',
                background: '#fff',
                minWidth: 0,
                overflow: 'hidden',
              }}>
                {/* 平面图标题 */}
                <div style={{
                  padding: '12px 16px',
                  borderBottom: '1px solid #e8e8e8',
                  background: '#fafafa',
                }}>
                  <Space>
                    <DesktopOutlined style={{ color: '#1890ff' }} />
                    <span style={{ fontSize: 14, fontWeight: 'bold', color: '#262626' }}>
                      {selectedFloor.name || selectedFloor.floorNo} - 工位平面图
                    </span>
                    <Tag color="blue">{workstations.length} 个工位</Tag>
                  </Space>
                </div>

                {/* 平面图内容 */}
                <div style={{ flex: 1, position: 'relative', overflow: 'hidden' }}>
                  {loadingWorkstations ? (
                    <div style={{
                      height: '100%',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                    }}>
                      <Spin tip="加载工位数据..." />
                    </div>
                  ) : workstations.length === 0 ? (
                    <div style={{
                      height: '100%',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                    }}>
                      <Empty
                        image={Empty.PRESENTED_IMAGE_SIMPLE}
                        description="该楼层暂无工位"
                      />
                    </div>
                  ) : (
                    <div style={{ height: '100%', width: '100%' }}>
                      <FloorPlan3D workstations={workstations} />
                    </div>
                  )}
                </div>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
};

export default BuildingView3D;
