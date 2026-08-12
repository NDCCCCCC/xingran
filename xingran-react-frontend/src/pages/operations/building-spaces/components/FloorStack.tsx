/**
 * 楼层堆叠组件
 * 显示3D堆叠的楼层卡片
 */
import React, { useState } from "react";
import FloorCard3D from "./FloorCard3D";
import type { Floor } from "../types";
import styles from "./styles.module.css";

interface FloorStackProps {
  floors: Floor[];
  onFloorClick: (floor: Floor) => void;
}

const FloorStack: React.FC<FloorStackProps> = ({ floors, onFloorClick }) => {
  const [selectedFloorId, setSelectedFloorId] = useState<string | null>(null);

  const handleFloorClick = (floor: Floor) => {
    setSelectedFloorId(floor.id);
    // 延迟执行点击，让动画先播放
    setTimeout(() => {
      onFloorClick(floor);
      setSelectedFloorId(null);
    }, 300);
  };

  if (floors.length === 0) {
    return (
      <div className={styles.emptyState}>
        <div className={styles.emptyStateIcon}>🏢</div>
        <div className={styles.emptyStateText}>暂无楼层数据</div>
      </div>
    );
  }

  return (
    <div className={styles.scene}>
      <div className={styles.floorStack}>
        {floors.map((floor, index) => (
          <FloorCard3D
            key={floor.id}
            floor={floor}
            index={index}
            isSelected={selectedFloorId === floor.id}
            onClick={() => handleFloorClick(floor)}
          />
        ))}
      </div>
    </div>
  );
};

export default FloorStack;
