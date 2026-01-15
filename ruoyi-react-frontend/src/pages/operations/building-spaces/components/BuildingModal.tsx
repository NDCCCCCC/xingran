/**
 * 楼宇详情模态框
 * 左侧显示3D楼层堆叠，右侧显示工位平面图
 */
import React, { useState } from 'react';
import { Modal } from 'antd';
import FloorStack from './FloorStack';
import WorkstationView from './WorkstationView';
import type { Building, Floor, ModalView } from '../types';
import styles from './styles.module.css';

interface BuildingModalProps {
  building: Building;
  visible: boolean;
  onClose: () => void;
}

const BuildingModal: React.FC<BuildingModalProps> = ({ building, visible, onClose }) => {
  const [modalView, setModalView] = useState<ModalView>('floors');
  const [selectedFloor, setSelectedFloor] = useState<Floor | null>(null);

  const handleFloorClick = (floor: Floor) => {
    setSelectedFloor(floor);
    setModalView('workstation');
  };

  const handleBackToFloors = () => {
    setSelectedFloor(null);
    setModalView('floors');
  };

  const handleModalClose = () => {
    setSelectedFloor(null);
    setModalView('floors');
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
        {/* 左侧：楼层堆叠视图 */}
        {modalView === 'floors' && (
          <div className={styles.modalLeftPanel}>
            <h3 style={{ marginBottom: 16, textAlign: 'center' }}>
              楼层列表
            </h3>
            <FloorStack
              floors={building.floors || []}
              onFloorClick={handleFloorClick}
            />
          </div>
        )}

        {/* 右侧：工位平面图 */}
        {modalView === 'workstation' && selectedFloor && (
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
