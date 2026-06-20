/**
 * 楼宇卡片组件
 * 显示楼宇信息，支持3D悬浮效果
 */
import React from "react";
import { ApartmentOutlined } from "@ant-design/icons";
import type { Building } from "../types";
import styles from "./styles.module.css";

interface BuildingCardProps {
  building: Building;
  onClick: () => void;
}

const BuildingCard: React.FC<BuildingCardProps> = ({ building, onClick }) => {
  return (
    <div className={styles.buildingCard} onClick={onClick}>
      <div className={styles.buildingCardHeader}>
        <ApartmentOutlined className={styles.buildingIcon} />
        <div>
          <h3 className={styles.buildingTitle}>{building.name}</h3>
          <div className={styles.buildingCode}>{building.code}</div>
        </div>
      </div>

      {building.address && (
        <div className={styles.buildingAddress}>
          📍 {building.address}
        </div>
      )}

      <div className={styles.buildingStats}>
        <div className={styles.buildingStat}>
          <div className={styles.buildingStatValue}>{building.totalFloors || 0}</div>
          <div className={styles.buildingStatLabel}>楼层数</div>
        </div>
        <div className={styles.buildingStat}>
          <div className={styles.buildingStatValue}>{building.workstationCount || 0}</div>
          <div className={styles.buildingStatLabel}>工位数</div>
        </div>
      </div>
    </div>
  );
};

export default BuildingCard;
