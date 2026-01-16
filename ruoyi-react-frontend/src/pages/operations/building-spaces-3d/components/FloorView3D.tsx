/**
 * 3D 楼层视图组件
 * 显示楼层堆叠效果，点击楼层进入工位平面图
 */

import { useEffect, useState, useCallback } from 'react';
import { Card, Descriptions, Button, Space, Spin, Alert, Tag, Row, Col } from 'antd';
import {
  ArrowLeftOutlined,
  ApartmentOutlined,
  DesktopOutlined,
} from '@ant-design/icons';
import { useVisualizationStore } from '@/store/visualizationStore';
import { workstationApi } from '@/lib/opsApi';
import { handleApiError } from '@/utils/errorHandler';
import FloorPlan3D from './FloorPlan3D';

// 扩展 Workstation 类型以匹配 FloorPlan3D 的需求
interface WorkstationData {
  id: string;
  name: string;
  code: string;
  status: number; // 0=空闲, 1=占用, 2=维护
  type: number; // 0=固定, 1=灵活, 2=管理
  positionX?: number;
  positionY?: number;
  rotation?: number;
}

interface WorkstationStats {
  total: number;
  available: number;
  occupied: number;
  flexible: number;
  fixed: number;
}

const FloorView3D: React.FC = () => {
  const [loading, setLoading] = useState(true);
  const [workstations, setWorkstations] = useState<WorkstationData[]>([]);
  const [stats, setStats] = useState<WorkstationStats>({
    total: 0,
    available: 0,
    occupied: 0,
    flexible: 0,
    fixed: 0,
  });

  const { selectedFloor, navigateToBuilding, navigateToMap } = useVisualizationStore();

  // 加载工位数据
  const loadWorkstations = useCallback(async () => {
    if (!selectedFloor) return;

    try {
      setLoading(true);
      const result = await workstationApi.list({
        floorCode: selectedFloor.floorNo || selectedFloor.id,
        current: 1,
        pageSize: 1000,
      });

      const workstationList = result.data?.list || [];

      // 转换 Workstation 为 WorkstationData，添加缺失的字段
      const convertedWorkstations: WorkstationData[] = workstationList.map((ws) => ({
        id: ws.id,
        name: ws.name,
        code: ws.name, // 使用 name 作为 code（因为 API 不返回 code 字段）
        status: ws.status as number, // WorkstationStatus (0|1) 转换为 number
        type: ws.type as number, // WorkstationType (0|1) 转换为 number
        positionX: ws.positionX,
        positionY: ws.positionY,
        rotation: 0, // 默认旋转角度
      }));

      setWorkstations(convertedWorkstations);

      // 计算统计数据
      const stats: WorkstationStats = {
        total: workstationList.length,
        available: workstationList.filter((w) => w.status === 0).length,
        occupied: workstationList.filter((w) => w.status === 1).length,
        flexible: workstationList.filter((w) => w.type === 1).length,
        fixed: workstationList.filter((w) => w.type === 0).length,
      };
      setStats(stats);
    } catch (error) {
      handleApiError(error, '加载工位数据');
    } finally {
      setLoading(false);
    }
  }, [selectedFloor]);

  useEffect(() => {
    loadWorkstations();
  }, [loadWorkstations]);

  // 如果没有选中的楼层，返回楼宇
  if (!selectedFloor) {
    return (
      <div style={{ height: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        <Alert
          message="未选择楼层"
          description="请先选择一个楼层"
          type="info"
          showIcon
          action={
            <Button type="primary" onClick={navigateToBuilding}>
              返回楼宇
            </Button>
          }
        />
      </div>
    );
  }

  const statusText = selectedFloor.status === 0 ? '正常' : '停用';
  const statusColor = selectedFloor.status === 0 ? 'success' : 'default';

  return (
    <div
      style={{
        height: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
        padding: 24,
      }}
    >
      <Card
        style={{
          width: '100%',
          maxWidth: 1000,
          maxHeight: '85vh',
          overflowY: 'auto',
        }}
        title={
          <Space>
            <ApartmentOutlined />
            <span>{selectedFloor.name || `楼层 ${selectedFloor.floorNo}`}</span>
            <Tag color={statusColor}>{statusText}</Tag>
          </Space>
        }
        extra={
          <Space>
            <Button icon={<ArrowLeftOutlined />} onClick={navigateToBuilding}>
              返回楼宇
            </Button>
            <Button onClick={navigateToMap}>返回地图</Button>
          </Space>
        }
      >
        {loading ? (
          <div style={{ textAlign: 'center', padding: 40 }}>
            <Spin />
          </div>
        ) : (
          <>
            <Descriptions column={3} bordered size="small">
              <Descriptions.Item label="楼层编码">{selectedFloor.code}</Descriptions.Item>
              <Descriptions.Item label="楼层号">{selectedFloor.floorNo}</Descriptions.Item>
              <Descriptions.Item label="所属楼宇">{selectedFloor.buildingName}</Descriptions.Item>
            </Descriptions>

            {/* 工位统计 */}
            <Card
              size="small"
              title={<><DesktopOutlined /> 工位统计</>}
              style={{ marginTop: 16 }}
            >
              <Row gutter={16}>
                <Col span={6}>
                  <Card size="small">
                    <div style={{ textAlign: 'center' }}>
                      <div style={{ fontSize: 24, fontWeight: 'bold', color: '#1890ff' }}>
                        {stats.total}
                      </div>
                      <div style={{ fontSize: 12, color: '#666' }}>总工位数</div>
                    </div>
                  </Card>
                </Col>
                <Col span={6}>
                  <Card size="small">
                    <div style={{ textAlign: 'center' }}>
                      <div style={{ fontSize: 24, fontWeight: 'bold', color: '#52c41a' }}>
                        {stats.available}
                      </div>
                      <div style={{ fontSize: 12, color: '#666' }}>空闲工位</div>
                    </div>
                  </Card>
                </Col>
                <Col span={6}>
                  <Card size="small">
                    <div style={{ textAlign: 'center' }}>
                      <div style={{ fontSize: 24, fontWeight: 'bold', color: '#ff4d4f' }}>
                        {stats.occupied}
                      </div>
                      <div style={{ fontSize: 12, color: '#666' }}>占用工位</div>
                    </div>
                  </Card>
                </Col>
                <Col span={6}>
                  <Card size="small">
                    <div style={{ textAlign: 'center' }}>
                      <div style={{ fontSize: 24, fontWeight: 'bold', color: '#722ed1' }}>
                        {stats.flexible}
                      </div>
                      <div style={{ fontSize: 12, color: '#666' }}>灵活工位</div>
                    </div>
                  </Card>
                </Col>
              </Row>
            </Card>

            {/* 3D 工位平面图 */}
            <div style={{ marginTop: 24 }}>
              <FloorPlan3D
                workstations={workstations}
                onWorkstationClick={() => {
                  // 工位点击事件处理
                }}
              />
            </div>

            {/* 工位列表 */}
            {workstations.length > 0 && (
              <div style={{ marginTop: 24 }}>
                <h4 style={{ marginBottom: 16 }}>
                  <DesktopOutlined /> 工位列表 ({workstations.length} 个)
                </h4>
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(150px, 1fr))', gap: 12 }}>
                  {workstations.map((workstation) => {
                    const statusColor = workstation.status === 0 ? '#52c41a' : '#ff4d4f';
                    const typeText = workstation.type === 0 ? '固定' : workstation.type === 1 ? '灵活' : '管理';
                    const statusText = workstation.status === 0 ? '空闲' : '占用';

                    return (
                      <Card
                        key={workstation.id}
                        size="small"
                        hoverable
                        style={{
                          textAlign: 'center',
                          borderLeft: `4px solid ${statusColor}`,
                        }}
                      >
                        <div style={{ fontSize: 14, fontWeight: 'bold', marginBottom: 8 }}>
                          {workstation.name || `工位 ${workstation.code}`}
                        </div>
                        <div style={{ fontSize: 12, color: '#666', marginBottom: 4 }}>
                          <Tag color={statusColor}>{statusText}</Tag>
                        </div>
                        <div style={{ fontSize: 11, color: '#999' }}>
                          {typeText}
                        </div>
                      </Card>
                    );
                  })}
                </div>
              </div>
            )}
          </>
        )}
      </Card>
    </div>
  );
};

export default FloorView3D;
