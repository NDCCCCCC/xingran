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
import {
  convertApiWorkstations,
  calculateWorkstationStats,
  getWorkstationStatusText,
  getWorkstationTypeText,
  getWorkstationStatusColorCSS,
  getWorkstationTypeColorCSS,
} from '../utils';
import FloorPlan3D from './FloorPlan3D';

// ============ 类型定义 ============

interface WorkstationData {
  id: string;
  name: string;
  code: string;
  status: number;
  type: number;
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

// ============ 组件 ============

const FloorView3D: React.FC = () => {
  // State
  const [loading, setLoading] = useState(true);
  const [workstations, setWorkstations] = useState<WorkstationData[]>([]);
  const [stats, setStats] = useState<WorkstationStats>({
    total: 0,
    available: 0,
    occupied: 0,
    flexible: 0,
    fixed: 0,
  });

  const { selectedFloor, selectedBuilding, navigateToBuilding, navigateToMap } =
    useVisualizationStore();

  // ============ 数据加载 ============

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
      const convertedWorkstations = convertApiWorkstations(workstationList);

      setWorkstations(convertedWorkstations);
      setStats(calculateWorkstationStats(workstationList));
    } catch (error) {
      handleApiError(error, '加载工位数据');
    } finally {
      setLoading(false);
    }
  }, [selectedFloor]);

  useEffect(() => {
    loadWorkstations();
  }, [loadWorkstations]);

  // ============ 渲染 ============

  if (!selectedFloor) {
    return <NoFloorSelectedAlert onBackToMap={navigateToMap} />;
  }

  const statusColor = selectedFloor.status === 0 ? 'success' : 'default';
  const statusText = selectedFloor.status === 0 ? '正常' : '停用';

  return (
    <div style={styles.container}>
      <Card
        style={styles.card}
        title={
          <Space>
            <ApartmentOutlined />
            <span>{selectedFloor.name || `楼层 ${selectedFloor.floorNo}`}</span>
            <Tag color={statusColor}>{statusText}</Tag>
          </Space>
        }
        extra={
          <Space>
            <Button
              icon={<ArrowLeftOutlined />}
              onClick={() => selectedBuilding && navigateToBuilding(selectedBuilding)}
              disabled={!selectedBuilding}
            >
              返回楼宇
            </Button>
            <Button onClick={navigateToMap}>返回地图</Button>
          </Space>
        }
      >
        {loading ? (
          <LoadingView />
        ) : (
          <>
            <FloorDescriptions floor={selectedFloor} />
            <WorkstationStatsCard stats={stats} />
            <FloorPlan3D
              workstations={workstations}
              onWorkstationClick={() => {
                // 工位点击事件处理
              }}
            />
            {workstations.length > 0 && (
              <WorkstationList workstations={workstations} />
            )}
          </>
        )}
      </Card>
    </div>
  );
};

// ============ 子组件 ============

interface NoFloorSelectedAlertProps {
  onBackToMap: () => void;
}

const NoFloorSelectedAlert: React.FC<NoFloorSelectedAlertProps> = ({
  onBackToMap,
}) => (
  <div style={styles.centerContainer}>
    <Alert
      message="未选择楼层"
      description="请先选择一个楼层"
      type="info"
      showIcon
      action={
        <Button type="primary" onClick={onBackToMap}>
          返回地图
        </Button>
      }
    />
  </div>
);

const LoadingView: React.FC = () => (
  <div style={styles.loadingContainer}>
    <Spin />
  </div>
);

interface FloorDescriptionsProps {
  floor: {
    code: string;
    floorNo: string;
    buildingName: string;
  };
}

const FloorDescriptions: React.FC<FloorDescriptionsProps> = ({ floor }) => (
  <Descriptions column={3} bordered size="small">
    <Descriptions.Item label="楼层编码">{floor.code}</Descriptions.Item>
    <Descriptions.Item label="楼层号">{floor.floorNo}</Descriptions.Item>
    <Descriptions.Item label="所属楼宇">{floor.buildingName}</Descriptions.Item>
  </Descriptions>
);

interface WorkstationStatsCardProps {
  stats: WorkstationStats;
}

const WorkstationStatsCard: React.FC<WorkstationStatsCardProps> = ({ stats }) => (
  <Card size="small" title={<DesktopOutlined /> 工位统计} style={styles.statsCard}>
    <Row gutter={16}>
      <Col span={6}>
        <StatCard value={stats.total} label="总工位数" color="#1890ff" />
      </Col>
      <Col span={6}>
        <StatCard value={stats.available} label="空闲工位" color="#52c41a" />
      </Col>
      <Col span={6}>
        <StatCard value={stats.occupied} label="占用工位" color="#ff4d4f" />
      </Col>
      <Col span={6}>
        <StatCard value={stats.flexible} label="灵活工位" color="#722ed1" />
      </Col>
    </Row>
  </Card>
);

interface StatCardProps {
  value: number;
  label: string;
  color: string;
}

const StatCard: React.FC<StatCardProps> = ({ value, label, color }) => (
  <Card size="small">
    <div style={styles.statCardContainer}>
      <div style={{ ...styles.statValue, color }}>{value}</div>
      <div style={styles.statLabel}>{label}</div>
    </div>
  </Card>
);

interface WorkstationListProps {
  workstations: WorkstationData[];
}

const WorkstationList: React.FC<WorkstationListProps> = ({ workstations }) => (
  <div style={styles.workstationListContainer}>
    <h4 style={styles.workstationListTitle}>
      <DesktopOutlined /> 工位列表 ({workstations.length} 个)
    </h4>
    <div style={styles.workstationGrid}>
      {workstations.map((workstation) => (
        <WorkstationCard key={workstation.id} workstation={workstation} />
      ))}
    </div>
  </div>
);

interface WorkstationCardProps {
  workstation: WorkstationData;
}

const WorkstationCard: React.FC<WorkstationCardProps> = ({ workstation }) => {
  const statusColor = getWorkstationStatusColorCSS(workstation.status);
  const typeColor = getWorkstationTypeColorCSS(workstation.type);
  const statusText = getWorkstationStatusText(workstation.status);
  const typeText = getWorkstationTypeText(workstation.type);

  return (
    <Card
      size="small"
      hoverable
      style={{
        ...styles.workstationCard,
        borderLeft: `4px solid ${statusColor}`,
      }}
    >
      <div style={styles.workstationName}>
        {workstation.name || `工位 ${workstation.code}`}
      </div>
      <div style={styles.workstationStatus}>
        <Tag color={statusColor}>{statusText}</Tag>
      </div>
      <div style={styles.workstationType}>{typeText}</div>
    </Card>
  );
};

// ============ 样式 ============

const styles = {
  container: {
    height: '100vh',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
    padding: 24,
  },
  centerContainer: {
    height: '100vh',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
  },
  card: {
    width: '100%',
    maxWidth: 1000,
    maxHeight: '85vh',
    overflowY: 'auto' as const,
  },
  loadingContainer: {
    textAlign: 'center',
    padding: 40,
  },
  statsCard: {
    marginTop: 16,
  },
  statCardContainer: {
    textAlign: 'center' as const,
  },
  statValue: {
    fontSize: 24,
    fontWeight: 'bold',
  },
  statLabel: {
    fontSize: 12,
    color: '#666',
  },
  workstationListContainer: {
    marginTop: 24,
  },
  workstationListTitle: {
    marginBottom: 16,
  },
  workstationGrid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fill, minmax(150px, 1fr))',
    gap: 12,
  },
  workstationCard: {
    textAlign: 'center' as const,
  },
  workstationName: {
    fontSize: 14,
    fontWeight: 'bold',
    marginBottom: 8,
  },
  workstationStatus: {
    fontSize: 12,
    color: '#666',
    marginBottom: 4,
  },
  workstationType: {
    fontSize: 11,
    color: '#999',
  },
};

export default FloorView3D;
