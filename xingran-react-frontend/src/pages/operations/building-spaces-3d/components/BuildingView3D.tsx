/**
 * 3D 楼宇视图组件
 * 点击地图上的楼宇标记后，显示该楼宇的 3D 模型
 */

import { useState, useCallback, useEffect } from "react";
import { Button, Space, Spin, Alert, Tag, Select } from "antd";
import { ArrowLeftOutlined, DesktopOutlined, BuildOutlined } from "@ant-design/icons";
import { useVisualizationStore } from "@/store/visualizationStore";
import { floorApi, workstationApi, buildingApi } from "@/lib/opsApi";
import type { Floor } from "@/types/operations";
import { handleApiError } from "@/utils/errorHandler";
import { convertApiWorkstations, calculateWorkstationStats } from "../utils";
import BuildingModel3D from "./BuildingModel3D";
import FloorPlan3D from "./FloorPlan3D";

// ============ 类型定义 ============

interface FloorData {
  id: string;
  name: string;
  code: string;
  floorNo: string;
  workstationCount: number;
  status: number;
}

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

interface BuildingData {
  id: string;
  name: string;
  cityName?: string;
  address?: string;
  status: number;
}

// ============ 组件 ============

const BuildingView3D: React.FC = () => {
  // State
  const [loading, setLoading] = useState(true);
  const [buildings, setBuildings] = useState<BuildingData[]>([]);
  const [currentBuilding, setCurrentBuilding] = useState<BuildingData | null>(null);
  const [floors, setFloors] = useState<FloorData[]>([]);
  const [selectedFloor, setSelectedFloor] = useState<FloorData | null>(null);
  const [workstations, setWorkstations] = useState<WorkstationData[]>([]);
  const [loadingWorkstations, setLoadingWorkstations] = useState(false);

  const { selectedBuilding, navigateToMap } = useVisualizationStore();

  // ============ 数据加载 ============

  const loadBuildings = useCallback(async () => {
    try {
      const result = await buildingApi.list({
        current: 1,
        pageSize: 100,
      });
      setBuildings(result.data?.list || []);
    } catch (error) {
      handleApiError(error, "加载楼宇列表");
    }
  }, []);

  const loadFloors = useCallback(
    async (buildingId?: string) => {
      const targetBuildingId = buildingId || selectedBuilding?.id;
      if (!targetBuildingId) return;

      try {
        setLoading(true);
        const result = await floorApi.list({
          buildingId: targetBuildingId,
          current: 1,
          pageSize: 100,
        });

        const floorList = result.data?.list || [];

        // 获取每层楼的工位数量
        const floorsWithCount = await Promise.all(
          floorList.map(async (floor: Floor) => {
            try {
              const wsResult = await workstationApi.list({
                floorCode: floor.id,
                current: 1,
                pageSize: 1,
              });
              return {
                id: floor.id,
                name: floor.name || "",
                code: floor.code,
                floorNo: String(floor.floorNo),
                status: floor.status,
                workstationCount: wsResult.data?.total || 0,
              };
            } catch {
              return {
                id: floor.id,
                name: floor.name || "",
                code: floor.code,
                floorNo: String(floor.floorNo),
                status: floor.status,
                workstationCount: 0,
              };
            }
          })
        );

        setFloors(floorsWithCount);
      } catch (error) {
        handleApiError(error, "加载楼层数据");
      } finally {
        setLoading(false);
      }
    },
    [selectedBuilding]
  );

  useEffect(() => {
    loadBuildings();
    if (selectedBuilding) {
      setCurrentBuilding(selectedBuilding);
      loadFloors(selectedBuilding.id);
    }
  }, [selectedBuilding, loadBuildings, loadFloors]);

  // ============ 事件处理 ============

  const handleBuildingChange = useCallback(
    async (buildingId: string) => {
      const building = buildings.find((b) => b.id === buildingId);
      if (building) {
        setCurrentBuilding(building);
        setSelectedFloor(null);
        setWorkstations([]);
        await loadFloors(buildingId);
      }
    },
    [buildings, loadFloors]
  );

  const loadWorkstations = useCallback(async (floor: FloorData) => {
    try {
      setLoadingWorkstations(true);
      const result = await workstationApi.list({
        floorCode: floor.id,
        current: 1,
        pageSize: 500,
      });

      const workstationList = result.data?.list || [];
      const convertedWorkstations = convertApiWorkstations(workstationList);

      setWorkstations(convertedWorkstations);
    } catch (error) {
      handleApiError(error, "加载工位数据");
      setWorkstations([]);
    } finally {
      setLoadingWorkstations(false);
    }
  }, []);

  const handleFloorClick = useCallback(
    (floor: FloorData) => {
      setSelectedFloor(floor);
      loadWorkstations(floor);
    },
    [loadWorkstations]
  );

  // ============ 渲染 ============

  if (!selectedBuilding) {
    return <NoBuildingSelectedAlert onBackToMap={navigateToMap} />;
  }

  const statusColor = selectedBuilding.status === 0 ? "success" : "default";
  const statusText = selectedBuilding.status === 0 ? "正常" : "停用";
  const totalWorkstations = floors.reduce(
    (sum, f) => sum + (f.workstationCount || 0),
    0
  );

  return (
    <div style={styles.container}>
      {/* 顶部导航栏 */}
      <HeaderBar
        building={selectedBuilding}
        floor={selectedFloor}
        floorCount={floors.length}
        workstationCount={totalWorkstations}
        statusColor={statusColor}
        statusText={statusText}
      />

      {/* 主内容区 */}
      <div style={styles.mainContent}>
        {loading ? (
          <LoadingView />
        ) : (
          <>
            {/* 左侧：3D 楼宇模型 */}
            <BuildingPanel
              floors={floors}
              buildings={buildings}
              currentBuilding={currentBuilding}
              selectedFloorId={selectedFloor?.id}
              onFloorClick={handleFloorClick}
              onBuildingChange={handleBuildingChange}
              onBackToMap={navigateToMap}
            />

            {/* 右侧：工位平面图 */}
            {selectedFloor && (
              <FloorPlanPanel
                floor={selectedFloor}
                workstations={workstations}
                loading={loadingWorkstations}
              />
            )}
          </>
        )}
      </div>
    </div>
  );
};

// ============ 子组件 ============

interface NoBuildingSelectedAlertProps {
  onBackToMap: () => void;
}

const NoBuildingSelectedAlert: React.FC<NoBuildingSelectedAlertProps> = ({
  onBackToMap,
}) => (
  <div style={styles.centerContainer}>
    <Alert
      message="未选择楼宇"
      description="请先在地图上选择一个楼宇"
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

interface HeaderBarProps {
  building: BuildingData;
  floor: FloorData | null;
  floorCount: number;
  workstationCount: number;
  statusColor: string;
  statusText: string;
}

const HeaderBar: React.FC<HeaderBarProps> = ({
  building,
  floor,
  floorCount,
  workstationCount,
  statusColor,
  statusText,
}) => (
  <div style={styles.headerBar}>
    <div>
      <Space>
        <BuildOutlined style={{ color: "var(--theme-text-accent, #1890ff)" }} />
        <span style={styles.headerTitle}>{building.name}</span>
        <Tag color={statusColor}>{statusText}</Tag>
        {floor && (
          <>
            <span style={styles.headerArrow}>→</span>
            <Tag color="blue">{floor.name || floor.floorNo}</Tag>
          </>
        )}
      </Space>
    </div>

    <div style={styles.headerInfo}>
      <Space split={<span style={styles.splitter}>|</span>}>
        <span>{building.cityName || "-"}</span>
        <span>{floorCount} 层</span>
        <span>{workstationCount} 工位</span>
      </Space>
    </div>
  </div>
);

interface BuildingPanelProps {
  floors: FloorData[];
  buildings: BuildingData[];
  currentBuilding: BuildingData | null;
  selectedFloorId: string | undefined;
  onFloorClick: (floor: FloorData) => void;
  onBuildingChange: (buildingId: string) => void;
  onBackToMap: () => void;
}

const BuildingPanel: React.FC<BuildingPanelProps> = ({
  floors,
  buildings,
  currentBuilding,
  selectedFloorId,
  onFloorClick,
  onBuildingChange,
  onBackToMap,
}) => (
  <div
    style={{
      ...styles.buildingPanel,
      flex: selectedFloorId ? "3" : "1",
    }}
  >
    {/* 楼宇切换下拉框 */}
    <div style={styles.controlPanel}>
      <Space direction="vertical" size="small" style={{ width: "100%" }}>
        <Button
          icon={<ArrowLeftOutlined />}
          onClick={onBackToMap}
          size="small"
          style={{ width: "100%" }}
        >
          返回地图
        </Button>
        {buildings.length > 1 && (
          <>
            <div style={styles.controlLabel}>切换楼宇</div>
            <Select
              style={{ width: "100%" }}
              placeholder="选择楼宇"
              value={currentBuilding?.id}
              onChange={onBuildingChange}
              options={buildings.map((b) =>    ({
                label: b.name,
                value: b.id,
              }))}
             onSearch={() => {}}/>
          </>
        )}
      </Space>
    </div>

    {/* 3D 楼宇模型 */}
    <div style={styles.modelContainer}>
      <BuildingModel3D
        floors={floors}
        onFloorClick={onFloorClick}
        selectedFloorId={selectedFloorId}
      />
    </div>
  </div>
);

interface FloorPlanPanelProps {
  floor: FloorData;
  workstations: WorkstationData[];
  loading: boolean;
}

const FloorPlanPanel: React.FC<FloorPlanPanelProps> = ({
  floor,
  workstations,
  loading,
}) => (
  <div style={styles.floorPlanPanel}>
    {/* 平面图标题 */}
    <div style={styles.panelHeader}>
      <Space>
        <DesktopOutlined style={{ color: "var(--theme-text-accent, #1890ff)" }} />
        <span style={styles.panelTitle}>
          {floor.name || floor.floorNo} - 工位平面图
        </span>
        <Tag color="blue">{workstations.length} 个工位</Tag>
      </Space>
    </div>

    {/* 平面图内容 */}
    <div style={styles.panelContent}>
      {loading ? (
        <LoadingView message="加载工位数据..." />
      ) : workstations.length === 0 ? (
        <EmptyView message="该楼层暂无工位" />
      ) : (
        <div style={{ height: "100%", width: "100%" }}>
          <FloorPlan3D workstations={workstations} />
        </div>
      )}
    </div>
  </div>
);

const LoadingView: React.FC<{ message?: string }> = ({ message }) => (
  <div style={styles.centerContainer}>
    <Spin size="large">
      <div style={{ minHeight: 120 }} />
    </Spin>
    {message && (
      <div style={{ marginTop: 8, color: "rgba(0, 0, 0, 0.45)" }}>{message}</div>
    )}
  </div>
);

const EmptyView: React.FC<{ message: string }> = ({ message }) => (
  <div style={styles.centerContainer}>
    <span style={{ color: "var(--theme-text-tertiary, #999)" }}>{message}</span>
  </div>
);

// ============ 样式 ============

const styles = {
  container: {
    height: "100vh",
    display: "flex",
    flexDirection: "column" as const,
    overflow: "hidden",
    background: "#fff",
  },
  centerContainer: {
    height: "100vh",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
  },
  headerBar: {
    background: "#fff",
    borderBottom: "1px solid #e8e8e8",
    padding: "12px 24px",
    display: "flex",
    justifyContent: "space-between",
    alignItems: "center",
    zIndex: 100,
    boxShadow: "0 2px 8px rgba(0,0,0,0.06)",
  },
  headerTitle: {
    fontSize: 16,
    fontWeight: "bold",
    color: "var(--theme-text-primary, #262626)",
  },
  headerArrow: {
    color: "var(--theme-text-tertiary, #8c8c8c)",
  },
  headerInfo: {
    fontSize: 12,
    color: "var(--theme-text-tertiary, #8c8c8c)",
  },
  splitter: {
    color: "var(--theme-border-divider, #d9d9d9)",
  },
  mainContent: {
    flex: 1,
    display: "flex",
    overflow: "hidden",
  },
  buildingPanel: {
    position: "relative" as const,
    background: "#f5f5f5",
    borderRight: "1px solid #e8e8e8",
    transition: "flex 0.3s ease",
    minWidth: 0,
    overflow: "hidden",
    display: "flex",
    flexDirection: "column" as const,
  },
  controlPanel: {
    position: "absolute" as const,
    top: 16,
    left: 16,
    zIndex: 10,
    background: "rgba(255, 255, 255, 0.95)",
    backdropFilter: "blur(10px)",
    borderRadius: 8,
    padding: "12px",
    boxShadow: "0 2px 8px rgba(0,0,0,0.1)",
    border: "1px solid #e8e8e8",
    minWidth: 220,
  },
  controlLabel: {
    fontSize: 12,
    color: "var(--theme-text-tertiary, #8c8c8c)",
    marginTop: 4,
  },
  modelContainer: {
    height: "100%",
    width: "100%",
  },
  floorPlanPanel: {
    flex: "7",
    display: "flex",
    flexDirection: "column" as const,
    background: "#fff",
    minWidth: 0,
    overflow: "hidden",
  },
  panelHeader: {
    padding: "12px 16px",
    borderBottom: "1px solid #e8e8e8",
    background: "#fafafa",
  },
  panelTitle: {
    fontSize: 14,
    fontWeight: "bold",
    color: "var(--theme-text-primary, #262626)",
  },
  panelContent: {
    flex: 1,
    position: "relative" as const,
    overflow: "hidden",
  },
};

export default BuildingView3D;
