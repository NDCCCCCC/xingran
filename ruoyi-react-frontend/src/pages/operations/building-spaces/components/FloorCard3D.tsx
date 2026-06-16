/**
 * 3D楼层卡片组件
 * 单个楼层卡片的3D展示
 */
import React from "react";
import type { Floor } from "../types";
import styles from "./styles.module.css";

interface FloorCard3DProps {
  floor: Floor;
  index: number;
  isSelected: boolean;
  onClick: () => void;
}

const FloorCard3D: React.FC<FloorCard3DProps> = ({ floor, index, isSelected, onClick }) => {
  return (
    <div
      className={`${styles.floorCard} ${isSelected ? styles.selected : ""}`}
      data-index={index.toString()}
      data-floor={floor.floorNo}
      onClick={onClick}
    >
      <div className={styles.floorCardNumber}>{floor.floorNo}</div>
      <div className={styles.floorCardName}>{floor.name}</div>
      <div className={styles.floorCardStats}>
        {floor.workstationCount ?? 0} 个工位
      </div>
    </div>
  );
};

export default FloorCard3D;
