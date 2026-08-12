/**
 * 楼宇详情模态框
 * 左侧显示3D楼层堆叠,右侧显示工位平面图。
 * 重构(Phase 2):打开时按 buildingId 懒加载楼层,不再依赖卡片预填充 floors。
 */
import React, { useState, useEffect } from "react";
import { Modal, Spin } from "antd";
import FloorStack from "./FloorStack";
import WorkstationView from "./WorkstationView";
import { floorApi } from "@/lib/opsApi";
import type { Building, Floor, ModalView } from "../types";
import styles from "./styles.module.css";

interface BuildingModalProps {
  building: Building;
  visible: boolean;
  onClose: () => void;
}

const BuildingModal: React.FC<BuildingModalProps> = ({ building, visible, onClose }) => {
  const [modalView, setModalView] = useState<ModalView>("floors");
  const [selectedFloor, setSelectedFloor] = useState<Floor | null>(null);
  const [floors, setFloors] = useState<Floor[]>([]);
  const [loadingFloors, setLoadingFloors] = useState(false);

  // 懒加载该楼宇的楼层(visible 时按 buildingId 加载)
  useEffect(() => {
    if (!visible || !building?.id) {
      return;
    }
    const loadFloors = async () => {
      setLoadingFloors(true);
      try {
        const result = await floorApi.list({ buildingId: building.id, current: 1, pageSize: 50 });
        const list = (result.data?.list || []) as unknown as Array<Record<string, unknown>>;
        setFloors(
          list.map((f) => ({
            id: String(f.id ?? ""),
            buildingId: String(f.buildingId ?? f.building_id ?? building.id),
            floorNo: String(f.floorNo ?? f.floor_no ?? ""),
            name: String(f.name ?? ""),
            workstationCount: Number(f.workstationCount ?? 0),
          }))
        );
      } catch (error) {
        console.error("加载楼层失败:", error);
        setFloors([]);
      } finally {
        setLoadingFloors(false);
      }
    };
    loadFloors();
  }, [visible, building]);

  const handleFloorClick = (floor: Floor) => {
    setSelectedFloor(floor);
    setModalView("workstation");
  };

  const handleBackToFloors = () => {
    setSelectedFloor(null);
    setModalView("floors");
  };

  const handleModalClose = () => {
    setSelectedFloor(null);
    setModalView("floors");
    setFloors([]);
    onClose();
  };

  return (
    <Modal
      title={building.name}
      open={visible}
      onCancel={handleModalClose}
      width={1400}
      footer={null}
      className={styles.buildingModal}
      destroyOnHidden
    >
      <div className={styles.buildingModalContent}>
        {/* 左侧:楼层堆叠视图 */}
        {modalView === "floors" && (
          <div className={styles.modalLeftPanel}>
            <h3 style={{ marginBottom: 16, textAlign: "center" }}>
              楼层列表
            </h3>
            {loadingFloors ? (
              <div style={{ textAlign: "center", padding: 48 }}>
                <Spin size="large" />
              </div>
            ) : (
              <FloorStack
                floors={floors}
                onFloorClick={handleFloorClick}
              />
            )}
          </div>
        )}

        {/* 右侧:工位平面图(WorkstationView 自身懒加载工位) */}
        {modalView === "workstation" && selectedFloor && (
          <div className={styles.modalRightPanel}>
            <WorkstationView
              floor={selectedFloor}
              onBack={handleBackToFloors}
            />
          </div>
        )}
      </div>
    </Modal>
  );
};

export default BuildingModal;
