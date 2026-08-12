/**
 * 楼宇空间3D可视化主页面
 * 展示楼宇卡片、3D楼层堆叠、工位平面图
 *
 * 重构(Phase 2):统计卡片改调专用 COUNT 端点;楼宇卡片分页加载;
 * 不再一次性拉 floor/workstation 全量做前端关联。
 */
import { useState, useEffect, useCallback } from "react";
import { App, Card, Row, Col, Statistic, Spin, Pagination } from "antd";
import { ApartmentOutlined, DesktopOutlined } from "@ant-design/icons";
import { buildingApi, floorApi, workstationApi } from "@/lib/opsApi";
import BuildingCard from "./components/BuildingCard";
import BuildingModal from "./components/BuildingModal";
import type { Building } from "./types";
import styles from "./components/styles.module.css";

const PAGE_SIZE = 12;

const BuildingSpacesPage: React.FC = () => {
  const { message } = App.useApp();
  const [buildings, setBuildings] = useState<Building[]>([]);
  const [loading, setLoading] = useState(false);
  const [selectedBuilding, setSelectedBuilding] = useState<Building | null>(null);
  const [current, setCurrent] = useState(1);
  const [total, setTotal] = useState(0);
  const [statistics, setStatistics] = useState({
    totalBuildings: 0,
    totalFloors: 0,
    totalWorkstations: 0,
  });

  // 加载楼宇卡片(分页;后端 list 已带 totalFloors 维护字段 + workstationCount 子查询)
  const loadBuildings = useCallback(async (page: number) => {
    setLoading(true);
    try {
      const result = await buildingApi.list({ current: page, pageSize: PAGE_SIZE });
      setBuildings((result.data?.list || []) as unknown as Building[]);
      setTotal(result.data?.total || 0);
    } catch (error: unknown) {
      console.error("加载楼宇数据失败:", error);
      message.error((error as { message?: string })?.message || "加载楼宇数据失败");
    } finally {
      setLoading(false);
    }
  }, [message]);

  // 加载全局统计(专用 COUNT 端点,不依赖拉全量)
  const loadStatistics = useCallback(async () => {
    try {
      const [b, f, w] = await Promise.all([
        buildingApi.statistics(),
        floorApi.statistics(),
        workstationApi.statistics(),
      ]);
      setStatistics({
        totalBuildings: b.total || 0,
        totalFloors: f.total || 0,
        totalWorkstations: w.total || 0,
      });
    } catch (error) {
      console.error("加载统计数据失败:", error);
    }
  }, []);

  useEffect(() => {
    loadBuildings(current);
  }, [current, loadBuildings]);

  useEffect(() => {
    loadStatistics();
  }, [loadStatistics]);

  const handleBuildingClick = (building: Building) => {
    setSelectedBuilding(building);
  };

  const handleModalClose = () => {
    setSelectedBuilding(null);
  };

  return (
    <div className={styles.buildingSpacesPage}>
      {/* 统计卡片 */}
      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col xs={24} sm={8}>
          <Card className={styles.statCard}>
            <Statistic
              title="楼宇总数"
              value={statistics.totalBuildings}
              prefix={<ApartmentOutlined />}
              loading={loading}
            />
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card className={styles.statCard}>
            <Statistic
              title="总楼层数"
              value={statistics.totalFloors}
              prefix="🏢"
              loading={loading}
            />
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card className={styles.statCard}>
            <Statistic
              title="总工位数"
              value={statistics.totalWorkstations}
              prefix={<DesktopOutlined />}
              loading={loading}
            />
          </Card>
        </Col>
      </Row>

      {/* 加载状态 / 楼宇卡片网格 */}
      {loading ? (
        <div className={styles.loadingContainer}>
          <Spin size="large">
            <div className="tip-content">加载中...</div>
          </Spin>
        </div>
      ) : (
        <>
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

          {total > PAGE_SIZE && (
            <div style={{ textAlign: "center", marginTop: 24 }}>
              <Pagination
                current={current}
                pageSize={PAGE_SIZE}
                total={total}
                onChange={(page) => setCurrent(page)}
                showSizeChanger={false}
              />
            </div>
          )}
        </>
      )}

      {/* 楼宇详情模态框(懒加载楼层) */}
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
