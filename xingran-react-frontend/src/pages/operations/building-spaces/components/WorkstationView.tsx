/**
 * 工位平面图视图
 * 复用FloorPlanEditor组件
 */
import React, { useState, useEffect } from "react";
import { App, Button, Spin } from "antd";
import { ArrowLeftOutlined } from "@ant-design/icons";
import FloorPlanEditor from "@/components/shared/FloorPlanEditor";
import type { WorkstationNode } from "@/components/shared/FloorPlanEditor.types";
import { workstationApi } from "@/lib/opsApi";
import type { Floor } from "../types";
import styles from "./styles.module.css";

interface WorkstationViewProps {
  floor: Floor;
  onBack: () => void;
}

const WorkstationView: React.FC<WorkstationViewProps> = ({ floor, onBack }) => {
  const { message } = App.useApp();
  const [workstations, setWorkstations] = useState<WorkstationNode[]>([]);
  const [loading, setLoading] = useState(true);

  // 加载工位数据
  useEffect(() => {
    const loadWorkstations = async () => {
      setLoading(true);
      try {
        const result = await workstationApi.list({
          floorCode: floor.floorNo || floor.id,
          current: 1,
          pageSize: 1000
        });

        const nodes: WorkstationNode[] = (result.data?.list || []).map(ws => ({
          id: ws.id,
          code: ws.name || ws.id,
          name: ws.name || "",
          x: ws.positionX || 0,
          y: ws.positionY || 0,
          width: 80,
          height: 60,
          status: ws.status,
          type: ws.type,
          rotation: ws.rotation || 0,
        }));

        setWorkstations(nodes);
      } catch (error) {
        message.error("加载工位数据失败");
        console.error(error);
      } finally {
        setLoading(false);
      }
    };

    loadWorkstations();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [floor.id]);

  // 更新工位位置
  const handleUpdatePosition = async (items: { id: string; positionX: number; positionY: number; rotation?: number }[]) => {
    try {
      await workstationApi.updatePositions(items);
    } catch (error) {
      message.error("更新位置失败");
      throw error;
    }
  };

  // 编辑工位
  const handleEdit = (workstation: WorkstationNode) => {
    message.info(`编辑工位: ${workstation.name}`);
    // TODO: 打开编辑对话框
  };

  return (
    <div className={styles.workstationView}>
      <Button
        icon={<ArrowLeftOutlined />}
        onClick={onBack}
        className={styles.backButton}
      >
        返回楼层列表
      </Button>

      <div style={{ marginBottom: 16 }}>
        <h3>{floor.name} ({floor.floorNo}F)</h3>
        <p style={{ color: "var(--theme-text-tertiary, #666)", margin: 0 }}>
          共 {workstations.length} 个工位
        </p>
      </div>

      {loading ? (
        <div className={styles.loadingContainer}>
          <Spin size="large">
            <div className="tip-content">加载工位数据中...</div>
          </Spin>
        </div>
      ) : workstations.length === 0 ? (
        <div className={styles.emptyState}>
          <div className={styles.emptyStateIcon}>🪑</div>
          <div className={styles.emptyStateText}>该楼层暂无工位数据</div>
        </div>
      ) : (
        <FloorPlanEditor
          floorId={floor.id}
          workstations={workstations}
          onUpdatePosition={handleUpdatePosition}
          onEdit={handleEdit}
        />
      )}
    </div>
  );
};

export default WorkstationView;
