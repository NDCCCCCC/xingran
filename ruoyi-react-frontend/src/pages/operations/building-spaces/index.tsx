/**
 * 楼宇空间3D可视化主页面
 * 展示楼宇卡片、3D楼层堆叠、工位平面图
 */
import { useState, useEffect } from "react";
import { App, Card, Row, Col, Statistic, Spin } from "antd";
import { ApartmentOutlined, DesktopOutlined } from "@ant-design/icons";
import { buildingApi, floorApi, workstationApi } from "@/lib/opsApi";
import BuildingCard from "./components/BuildingCard";
import BuildingModal from "./components/BuildingModal";
import type { Building } from "./types";
import styles from "./components/styles.module.css";

const BuildingSpacesPage: React.FC = () => {
  const { message } = App.useApp();
  const [buildings, setBuildings] = useState<Building[]>([]);
  const [loading, setLoading] = useState(false);
  const [selectedBuilding, setSelectedBuilding] = useState<Building | null>(null);

  // 加载楼宇数据
  const loadBuildings = async () => {
    setLoading(true);
    try {
      // 并行加载所有需要的数据
      const [buildingResult, floorResult, workstationResult] = await Promise.all([
        buildingApi.list({ current: 1, pageSize: 100 }),
        floorApi.list({ current: 1, pageSize: 1000 }),
        workstationApi.list({ current: 1, pageSize: 10000 })
      ]);

      // 组装带统计的楼宇数据
      const buildingsWithStats = ((buildingResult.data?.list || []) as unknown as Record<string, unknown>[]).map((b) => {
        // 获取该楼宇的楼层
        const buildingFloors = ((floorResult.data?.list || []) as unknown as Record<string, unknown>[]).filter(
          (f) => f.buildingId === b.id
        );

        // 获取该楼宇的所有工位（通过楼层关联）
        const buildingWorkstations = ((workstationResult.data?.list || []) as unknown as Record<string, unknown>[]).filter(
          (ws) => buildingFloors.some((f) => f.id === ws.floorId)
        );

        return {
          ...b,
          floors: buildingFloors,
          totalFloors: buildingFloors.length,
          totalWorkstations: buildingWorkstations.length
        } as unknown as Building;
      });

      setBuildings(buildingsWithStats);
    } catch (error: unknown) {
      console.error("加载楼宇数据失败:", error);
      message.error((error as { message?: string })?.message || "加载楼宇数据失败");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadBuildings();
  }, []);

  const handleBuildingClick = (building: Building) => {
    setSelectedBuilding(building);
  };

  const handleModalClose = () => {
    setSelectedBuilding(null);
  };

  // 计算统计数据
  const totalBuildings = buildings.length;
  const totalFloors = buildings.reduce((sum, b) => sum + (b.totalFloors || 0), 0);
  const totalWorkstations = buildings.reduce((sum, b) => sum + (b.totalWorkstations || 0), 0);

  return (
    <div className={styles.buildingSpacesPage}>
      {/* 统计卡片 */}
      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col xs={24} sm={8}>
          <Card className={styles.statCard}>
            <Statistic
              title="楼宇总数"
              value={totalBuildings}
              prefix={<ApartmentOutlined />}
              loading={loading}
            />
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card className={styles.statCard}>
            <Statistic
              title="总楼层数"
              value={totalFloors}
              prefix="🏢"
              loading={loading}
            />
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card className={styles.statCard}>
            <Statistic
              title="总工位数"
              value={totalWorkstations}
              prefix={<DesktopOutlined />}
              loading={loading}
            />
          </Card>
        </Col>
      </Row>

      {/* 加载状态 */}
      {loading ? (
        <div className={styles.loadingContainer}>
          <Spin size="large">
            <div className="tip-content">加载中...</div>
          </Spin>
        </div>
      ) : (
        <>
          {/* 楼宇卡片网格 */}
          {buildings.length === 0 ? (
            <div className={styles.emptyState}>
              <div className={styles.emptyStateIcon}>🏢</div>
              <div className={styles.emptyStateText}>暂无楼宇数据</div>
            </div>
          ) : (
            <Row gutter={[16, 16]} className={styles.buildingGrid}>
              {buildings.map((building) => (
                <Col key={building.id} xs={24} sm={12} lg={8} xl={6}>
                  <BuildingCard
                    building={building}
                    onClick={() => handleBuildingClick(building)}
                  />
                </Col>
              ))}
            </Row>
          )}
        </>
      )}

      {/* 楼宇详情模态框 */}
      {selectedBuilding && (
        <BuildingModal
          building={selectedBuilding}
          visible={!!selectedBuilding}
          onClose={handleModalClose}
        />
      )}
    </div>
  );
};

export default BuildingSpacesPage;
