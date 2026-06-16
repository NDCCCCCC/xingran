/**
 * 3D 楼层视图组件
 * 显示楼层堆叠效果，点击楼层进入工位平面图
 */

import { useEffect, useState, useCallback } from "react";
import { Card, Descriptions, Button, Space, Spin, Alert, Tag, Row, Col } from "antd";
import {
  ArrowLeftOutlined,
  ApartmentOutlined,
  DesktopOutlined,
} from "@ant-design/icons";
import { useVisualizationStore } from "@/store/visualizationStore";
import { workstationApi } from "@/lib/opsApi";
import { handleApiError } from "@/utils/errorHandler";
import {
  convertApiWorkstations,
  calculateWorkstationStats,
  getWorkstationStatusText,
  getWorkstationTypeText,
  getWorkstationStatusColorCSS,
  getWorkstationTypeColorCSS,
} from "../utils";
import FloorPlan3D from "./FloorPlan3D";

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
      handleApiError(error, "加载工位数据");
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

  const statusColor = selectedFloor.status === 0 ? "success" : "default";
  const statusText = selectedFloor.status === 0 ? "正常" : "停用";

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
    buildingName?: string;
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

/**
 * 工位统计色板
 * 注意：与 AntD 主题色（colorPrimary / colorSuccess / colorError / 紫色）保持一致
 * 原因：本地 StatCard 组件的 `color` prop 类型为 string，
 *      不支持 AntD status token；如需主题联动需改 StatCard 组件签名（超出本次修复范围）。
 * 当前策略：将字面量提升为命名常量，便于后续统一替换为 token / CSS 变量。
 */
const WORKSTATION_STAT_COLORS = {
  primary: "var(--theme-info, #1890ff)",   // 总工位 - AntD colorPrimary
  success: "var(--theme-success, #52c41a)",   // 空闲工位 - AntD colorSuccess
  error: "#ff4d4f",     // 占用工位 - AntD colorError
  purple: "var(--theme-purple, #722ed1)",    // 灵活工位 - AntD 紫色
} as const;

const WorkstationStatsCard: React.FC<WorkstationStatsCardProps> = ({ stats }) => (
  <Card size="small" title={<><DesktopOutlined /> 工位统计</>} style={styles.statsCard}>
    <Row gutter={16}>
      <Col span={6}>
        <StatCard value={stats.total} label="总工位数" color={WORKSTATION_STAT_COLORS.primary} />
      </Col>
      <Col span={6}>
        <StatCard value={stats.available} label="空闲工位" color={WORKSTATION_STAT_COLORS.success} />
      </Col>
      <Col span={6}>
        <StatCard value={stats.occupied} label="占用工位" color={WORKSTATION_STAT_COLORS.error} />
      </Col>
      <Col span={6}>
        <StatCard value={stats.flexible} label="灵活工位" color={WORKSTATION_STAT_COLORS.purple} />
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
    height: "100vh",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    background: "linear-gradient(135deg, #667eea 0%, #764ba2 100%)",
    padding: 24,
  },
  centerContainer: {
    height: "100vh",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
  },
  card: {
    width: "100%",
    maxWidth: 1000,
    maxHeight: "85vh",
    overflowY: "auto" as const,
  },
  loadingContainer: {
    textAlign: "center" as const,
    padding: 40,
  },
  statsCard: {
    marginTop: 16,
  },
  statCardContainer: {
    textAlign: "center" as const,
  },
  statValue: {
    fontSize: 24,
    fontWeight: "bold",
  },
  statLabel: {
    fontSize: 12,
    color: "var(--theme-text-tertiary, #666)",
  },
  workstationListContainer: {
    marginTop: 24,
  },
  workstationListTitle: {
    marginBottom: 16,
  },
  workstationGrid: {
    display: "grid",
    gridTemplateColumns: "repeat(auto-fill, minmax(150px, 1fr))",
    gap: 12,
  },
  workstationCard: {
    textAlign: "center" as const,
  },
  workstationName: {
    fontSize: 14,
    fontWeight: "bold",
    marginBottom: 8,
  },
  workstationStatus: {
    fontSize: 12,
    color: "var(--theme-text-tertiary, #666)",
    marginBottom: 4,
  },
  workstationType: {
    fontSize: 11,
    color: "var(--theme-text-tertiary, #999)",
  },
};

export default FloorView3D;
