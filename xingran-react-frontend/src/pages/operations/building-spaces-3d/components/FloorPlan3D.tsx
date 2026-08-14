/**
 * 3D 工位平面图组件
 * 使用 Three.js + React Three Fiber 渲染交互式 3D 工位平面图
 */

import { useRef, useState, useMemo } from "react";
import { Canvas, useFrame } from "@react-three/fiber";
import { OrbitControls, PerspectiveCamera, Html } from "@react-three/drei";
import * as THREE from "three";
import {
  WORKSTATION_DIMENSIONS,
  MONITOR_DIMENSIONS,
  KEYBOARD_DIMENSIONS,
  MOUSEPAD_DIMENSIONS,
  WORKSTATION_LAYOUT,
  CAMERA_CONFIG,
  FLOOR_CONFIG,
  WORKSTATION_STATUS_COLORS,
  MATERIAL_COLORS,
  STATUS_TEXT,
} from "../constants";
import {
  degreesToRadians,
  calculateWorkstationPositions,
  getWorkstationColor,
  getWorkstationStatusText,
  getWorkstationTypeText,
} from "../utils";

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

interface FloorPlan3DProps {
  workstations: WorkstationData[];
  onWorkstationClick?: (workstation: WorkstationData) => void;
}

// ============ 3D 文本组件 ============

interface Text3DProps {
  position: [number, number, number];
  fontSize: number;
  color: string;
  children: string;
}

const Text3D: React.FC<Text3DProps> = ({ position, fontSize, color, children }) => (
  <Html position={position} center distanceFactor={10} style={{ pointerEvents: "none" }}>
    <div
      style={{
        color,
        fontSize: `${fontSize * 10}px`,
        fontWeight: "bold",
        textShadow: "0 1px 3px rgba(0,0,0,0.3)",
        whiteSpace: "nowrap",
        userSelect: "none",
      }}
    >
      {children}
    </div>
  </Html>
);

// ============ 单个工位组件 ============

interface Workstation3DProps {
  workstation: WorkstationData;
  position: [number, number, number];
  onClick?: () => void;
  isSelected?: boolean;
}

const Workstation3D: React.FC<Workstation3DProps> = ({
  workstation,
  position,
  onClick,
  isSelected,
}) => {
  const groupRef = useRef<THREE.Group>(null);
  const [hovered, setHovered] = useState(false);

  const rotation = degreesToRadians(workstation.rotation || 0);
  const statusColor = getWorkstationColor(workstation);

  // 计算尺寸
  const deskTopY = WORKSTATION_DIMENSIONS.LEG_HEIGHT + WORKSTATION_DIMENSIONS.DESK_HEIGHT / 2;

  // 悬停动画
  useFrame(() => {
    if (groupRef.current) {
      const targetY = hovered || isSelected ? 0.05 : 0;
      groupRef.current.position.y = THREE.MathUtils.lerp(groupRef.current.position.y, targetY, 0.1);
    }
  });

  const statusText = getWorkstationStatusText(workstation.status);
  const typeText = getWorkstationTypeText(workstation.type);

  return (
    <group ref={groupRef} position={position}>
      <group rotation={[0, rotation, 0]}>
        {/* 桌腿 */}
        <DeskLegs />

        {/* 桌面 */}
        <Desktop
          positionY={deskTopY}
          onClick={onClick}
          onPointerOver={() => setHovered(true)}
          onPointerOut={() => setHovered(false)}
          statusColor={statusColor}
        />

        {/* 显示器 */}
        <Monitor
          positionY={deskTopY}
          statusColor={statusColor}
          hovered={hovered}
          isSelected={isSelected}
        />

        {/* 键盘 */}
        <Keyboard positionY={deskTopY} />

        {/* 鼠标垫 */}
        <Mousepad positionY={deskTopY} />

        {/* 椅子 */}
        <Chair statusColor={statusColor} />

        {/* 选中高亮 */}
        {(hovered || isSelected) && (
          <SelectionHighlight positionY={deskTopY} statusColor={statusColor} />
        )}

        {/* 工位标签 */}
        <Text3D position={[0, deskTopY + 1.1, 0]} fontSize={0.15} color="white">
          {workstation.name || workstation.code}
        </Text3D>

        {/* 悬停提示 */}
        {hovered && (
          <Html
            position={[0, deskTopY + 1.6, 0]}
            center
            distanceFactor={10}
            style={{ pointerEvents: "none" }}
          >
            <WorkstationTooltip
              name={workstation.name || workstation.code}
              statusText={statusText}
              typeText={typeText}
              status={workstation.status}
              type={workstation.type}
            />
          </Html>
        )}
      </group>
    </group>
  );
};

// ============ 工位子组件 ============

const DeskLegs: React.FC = () => {
  const { DESK_WIDTH, DESK_DEPTH, LEG_HEIGHT, LEG_THICKNESS } = WORKSTATION_DIMENSIONS;
  const positions = [
    [-DESK_WIDTH / 2 + LEG_THICKNESS, LEG_HEIGHT / 2, -DESK_DEPTH / 2 + LEG_THICKNESS],
    [DESK_WIDTH / 2 - LEG_THICKNESS, LEG_HEIGHT / 2, -DESK_DEPTH / 2 + LEG_THICKNESS],
    [-DESK_WIDTH / 2 + LEG_THICKNESS, LEG_HEIGHT / 2, DESK_DEPTH / 2 - LEG_THICKNESS],
    [DESK_WIDTH / 2 - LEG_THICKNESS, LEG_HEIGHT / 2, DESK_DEPTH / 2 - LEG_THICKNESS],
  ];

  return (
    <>
      {positions.map((pos, i) => (
        <mesh key={i} position={pos as [number, number, number]} castShadow>
          <boxGeometry args={[LEG_THICKNESS, LEG_HEIGHT, LEG_THICKNESS]} />
          <meshStandardMaterial color={MATERIAL_COLORS.DESK_LEG} />
        </mesh>
      ))}
    </>
  );
};

interface DesktopProps {
  positionY: number;
  onClick?: () => void;
  onPointerOver?: () => void;
  onPointerOut?: () => void;
  statusColor: number;
}

const Desktop: React.FC<DesktopProps> = ({
  positionY,
  onClick,
  onPointerOver,
  onPointerOut,
  statusColor,
}) => {
  const { DESK_WIDTH, DESK_DEPTH, DESK_HEIGHT } = WORKSTATION_DIMENSIONS;

  return (
    <>
      <mesh
        position={[0, positionY, 0]}
        onClick={onClick}
        onPointerOver={onPointerOver}
        onPointerOut={onPointerOut}
        castShadow
        receiveShadow
      >
        <boxGeometry args={[DESK_WIDTH, DESK_HEIGHT, DESK_DEPTH]} />
        <meshStandardMaterial color={MATERIAL_COLORS.DESK_TOP} roughness={0.8} metalness={0} />
      </mesh>

      <mesh position={[0, positionY + 0.01, 0]}>
        <boxGeometry args={[DESK_WIDTH + 0.02, 0.01, DESK_DEPTH + 0.02]} />
        <meshStandardMaterial color={statusColor} />
      </mesh>
    </>
  );
};

interface MonitorProps {
  positionY: number;
  statusColor: number;
  hovered: boolean;
  isSelected?: boolean;
}

const Monitor: React.FC<MonitorProps> = ({ positionY, statusColor, hovered, isSelected }) => {
  const { DESK_DEPTH } = WORKSTATION_DIMENSIONS;
  const {
    WIDTH,
    HEIGHT,
    THICKNESS,
    BORDER_INCREMENT,
    BORDER_HEIGHT_INCREMENT,
    BORDER_THICKNESS_INCREMENT,
    STAND_WIDTH,
    STAND_HEIGHT,
    STAND_DEPTH,
    STAND_BOTTOM_Y,
    SCREEN_Y,
  } = MONITOR_DIMENSIONS;

  const monitorY = positionY + 0.35;
  const zPos = -DESK_DEPTH / 2 + 0.15;

  return (
    <group position={[0, monitorY, zPos]}>
      <mesh position={[0, STAND_BOTTOM_Y, 0]} castShadow>
        <boxGeometry args={[STAND_WIDTH, STAND_HEIGHT, STAND_DEPTH]} />
        <meshStandardMaterial color={MATERIAL_COLORS.MONITOR_STAND} />
      </mesh>

      <mesh position={[0, SCREEN_Y, 0]} castShadow>
        <boxGeometry args={[WIDTH, HEIGHT, THICKNESS]} />
        <meshStandardMaterial
          color={MATERIAL_COLORS.MONITOR_SCREEN}
          roughness={0.3}
          metalness={0.8}
        />
      </mesh>

      <mesh position={[0, SCREEN_Y, 0]}>
        <boxGeometry
          args={[
            WIDTH + BORDER_INCREMENT,
            HEIGHT + BORDER_HEIGHT_INCREMENT,
            THICKNESS + BORDER_THICKNESS_INCREMENT,
          ]}
        />
        <meshStandardMaterial color={MATERIAL_COLORS.MONITOR_BORDER} />
      </mesh>

      {(hovered || isSelected) && (
        <mesh position={[0, SCREEN_Y, 0.01]}>
          <planeGeometry args={[WIDTH - 0.02, HEIGHT - 0.04]} />
          <meshBasicMaterial color={statusColor} transparent opacity={0.4} />
        </mesh>
      )}
    </group>
  );
};

interface KeyboardProps {
  positionY: number;
}

const Keyboard: React.FC<KeyboardProps> = ({ positionY }) => {
  const { WIDTH, DEPTH, HEIGHT, KEY_HEIGHT, KEY_WIDTH_REDUCTION, KEY_DEPTH_REDUCTION } =
    KEYBOARD_DIMENSIONS;

  return (
    <>
      <mesh position={[0.1, positionY + 0.015, 0.15]} castShadow>
        <boxGeometry args={[WIDTH, HEIGHT, DEPTH]} />
        <meshStandardMaterial color={MATERIAL_COLORS.KEYBOARD} />
      </mesh>

      <mesh position={[0.1, positionY + 0.023, 0.15]}>
        <boxGeometry
          args={[WIDTH - KEY_WIDTH_REDUCTION, KEY_HEIGHT, DEPTH - KEY_DEPTH_REDUCTION]}
        />
        <meshStandardMaterial color={MATERIAL_COLORS.KEYBOARD_KEY} />
      </mesh>
    </>
  );
};

interface MousepadProps {
  positionY: number;
}

const Mousepad: React.FC<MousepadProps> = ({ positionY }) => {
  const { WIDTH, DEPTH, HEIGHT } = MOUSEPAD_DIMENSIONS;

  return (
    <mesh position={[0.4, positionY + 0.01, 0.2]} castShadow>
      <boxGeometry args={[WIDTH, HEIGHT, DEPTH]} />
      <meshStandardMaterial color={MATERIAL_COLORS.MOUSEPAD} roughness={0.9} />
    </mesh>
  );
};

interface ChairProps {
  statusColor: number;
}

const Chair: React.FC<ChairProps> = ({ statusColor }) => {
  const { DESK_DEPTH } = WORKSTATION_DIMENSIONS;
  const {
    CHAIR_WIDTH,
    CHAIR_SEAT_HEIGHT,
    CHAIR_BACK_HEIGHT,
    CHAIR_BACK_THICKNESS,
    CHAIR_SEAT_Y,
    CHAIR_TOTAL_HEIGHT,
    ARMREST_HEIGHT,
    ARMREST_THICKNESS,
    ARMREST_OFFSET,
    CHAIR_LEG_HEIGHT,
    CHAIR_LEG_THICKNESS,
  } = WORKSTATION_DIMENSIONS;

  const chairZ = DESK_DEPTH / 2 + 0.35;

  return (
    <group position={[0, 0, chairZ]}>
      <mesh position={[0, CHAIR_SEAT_Y, 0]} castShadow>
        <boxGeometry args={[CHAIR_WIDTH, CHAIR_SEAT_HEIGHT, CHAIR_WIDTH]} />
        <meshStandardMaterial color={MATERIAL_COLORS.CHAIR_SEAT} />
      </mesh>

      <mesh position={[0, CHAIR_TOTAL_HEIGHT, 0.18]} castShadow>
        <boxGeometry args={[CHAIR_WIDTH, CHAIR_BACK_HEIGHT, CHAIR_BACK_THICKNESS]} />
        <meshStandardMaterial color={statusColor} />
      </mesh>

      <mesh position={[-ARMREST_OFFSET, CHAIR_SEAT_Y + ARMREST_HEIGHT / 2, 0]} castShadow>
        <boxGeometry args={[ARMREST_THICKNESS, ARMREST_HEIGHT, ARMREST_THICKNESS]} />
        <meshStandardMaterial color={MATERIAL_COLORS.ARMREST} />
      </mesh>

      <mesh position={[ARMREST_OFFSET, CHAIR_SEAT_Y + ARMREST_HEIGHT / 2, 0]} castShadow>
        <boxGeometry args={[ARMREST_THICKNESS, ARMREST_HEIGHT, ARMREST_THICKNESS]} />
        <meshStandardMaterial color={MATERIAL_COLORS.ARMREST} />
      </mesh>

      <mesh position={[-0.18, 0.22, 0.18]} castShadow>
        <boxGeometry args={[CHAIR_LEG_THICKNESS, CHAIR_LEG_HEIGHT, CHAIR_LEG_THICKNESS]} />
        <meshStandardMaterial color={MATERIAL_COLORS.CHAIR_LEG} />
      </mesh>

      <mesh position={[0.18, 0.22, 0.18]} castShadow>
        <boxGeometry args={[CHAIR_LEG_THICKNESS, CHAIR_LEG_HEIGHT, CHAIR_LEG_THICKNESS]} />
        <meshStandardMaterial color={MATERIAL_COLORS.CHAIR_LEG} />
      </mesh>

      <mesh position={[-0.18, 0.22, -0.18]} castShadow>
        <boxGeometry args={[CHAIR_LEG_THICKNESS, CHAIR_LEG_HEIGHT, CHAIR_LEG_THICKNESS]} />
        <meshStandardMaterial color={MATERIAL_COLORS.CHAIR_LEG} />
      </mesh>

      <mesh position={[0.18, 0.22, -0.18]} castShadow>
        <boxGeometry args={[CHAIR_LEG_THICKNESS, CHAIR_LEG_HEIGHT, CHAIR_LEG_THICKNESS]} />
        <meshStandardMaterial color={MATERIAL_COLORS.CHAIR_LEG} />
      </mesh>
    </group>
  );
};

interface SelectionHighlightProps {
  positionY: number;
  statusColor: number;
}

const SelectionHighlight: React.FC<SelectionHighlightProps> = ({ positionY, statusColor }) => {
  const { DESK_WIDTH, DESK_DEPTH } = WORKSTATION_DIMENSIONS;

  return (
    <mesh position={[0, positionY, 0]}>
      <boxGeometry args={[DESK_WIDTH + 0.05, 0.02, DESK_DEPTH + 0.05]} />
      <meshBasicMaterial color={statusColor} transparent opacity={0.3} />
    </mesh>
  );
};

interface WorkstationTooltipProps {
  name: string;
  statusText: string;
  typeText: string;
  status: number;
  type: number;
}

const WorkstationTooltip: React.FC<WorkstationTooltipProps> = ({
  name,
  statusText,
  typeText,
  status,
  type,
}) => {
  const statusColor =
    status === 0
      ? "var(--theme-success, #52c41a)"
      : status === 1
        ? "#ff4d4f"
        : "var(--theme-warning, #faad14)";
  const typeColor =
    type === 0
      ? "var(--theme-info, #1890ff)"
      : type === 1
        ? "var(--theme-purple, #722ed1)"
        : "#13c2c2";

  return (
    <div style={styles.tooltip}>
      <div style={styles.tooltipName}>{name}</div>
      <div style={styles.tooltipContent}>
        <TooltipRow label="状态：" value={statusText} valueColor={statusColor} />
        <TooltipRow label="类型：" value={typeText} valueColor={typeColor} />
      </div>
    </div>
  );
};

interface TooltipRowProps {
  label: string;
  value: string;
  valueColor: string;
}

const TooltipRow: React.FC<TooltipRowProps> = ({ label, value, valueColor }) => (
  <div style={styles.tooltipRow}>
    <span>{label}</span>
    <span style={{ fontWeight: "bold", color: valueColor }}>{value}</span>
  </div>
);

// ============ 3D 平面图场景 ============

interface FloorPlanSceneProps {
  workstations: WorkstationData[];
  onWorkstationClick?: (workstation: WorkstationData) => void;
}

const FloorPlanScene: React.FC<FloorPlanSceneProps> = ({ workstations, onWorkstationClick }) => {
  const groupRef = useRef<THREE.Group>(null);
  const [selectedId, setSelectedId] = useState<string | null>(null);

  // 计算工位位置
  const positionMap = useMemo(
    () =>
      calculateWorkstationPositions(
        workstations,
        WORKSTATION_LAYOUT.POSITION_SCALE,
        WORKSTATION_LAYOUT.POSITION_OFFSET
      ),
    [workstations]
  );

  const handleWorkstationClick = (workstation: WorkstationData) => {
    setSelectedId(workstation.id);
    if (onWorkstationClick) {
      onWorkstationClick(workstation);
    }
  };

  return (
    <group ref={groupRef}>
      {/* 环境光 */}
      <ambientLight intensity={0.7} />
      <directionalLight position={[10, 15, 10]} intensity={0.8} castShadow />
      <directionalLight position={[-10, 10, -10]} intensity={0.5} />
      <directionalLight position={[0, -10, 0]} intensity={0.3} />

      {/* 地面 */}
      <mesh rotation={[-Math.PI / 2, 0, 0]} position={[0, FLOOR_CONFIG.Y, 0]} receiveShadow>
        <planeGeometry args={[FLOOR_CONFIG.SIZE, FLOOR_CONFIG.SIZE]} />
        <meshStandardMaterial color={FLOOR_CONFIG.COLOR} opacity={1} transparent={false} />
      </mesh>

      {/* 渲染工位 */}
      {workstations.map((ws) => {
        const pos = positionMap.get(ws.id);
        if (!pos) return null;

        return (
          <Workstation3D
            key={ws.id}
            workstation={ws}
            position={[pos.x, 0, pos.z]}
            onClick={() => handleWorkstationClick(ws)}
            isSelected={selectedId === ws.id}
          />
        );
      })}

      {/* 图例 */}
      <Html position={[-12, 0.3, -12]} center distanceFactor={10} style={{ pointerEvents: "none" }}>
        <div style={styles.legend}>
          <div style={{ color: "var(--theme-success, #388e3c)" }}>● {STATUS_TEXT.AVAILABLE}</div>
          <div style={{ color: "var(--theme-error, #d32f2f)" }}>● {STATUS_TEXT.OCCUPIED}</div>
          <div style={{ color: "var(--theme-warning, #f57c00)" }}>● {STATUS_TEXT.MAINTENANCE}</div>
        </div>
      </Html>
    </group>
  );
};

// ============ 主组件 ============

const FloorPlan3D: React.FC<FloorPlan3DProps> = ({ workstations, onWorkstationClick }) => {
  if (workstations.length === 0) {
    return <EmptyView />;
  }

  return (
    <div style={styles.container}>
      <Canvas
        shadows
        dpr={[1, 2]}
        gl={{ antialias: true, alpha: true }}
        style={{ background: "#f5f5f5" }}
      >
        <PerspectiveCamera
          makeDefault
          position={CAMERA_CONFIG.DEFAULT_POSITION}
          fov={CAMERA_CONFIG.FOV}
        />
        <OrbitControls
          enableZoom={true}
          enablePan={true}
          minPolarAngle={0}
          maxPolarAngle={CAMERA_CONFIG.MAX_POLAR_ANGLE}
          minDistance={CAMERA_CONFIG.MIN_DISTANCE}
          maxDistance={CAMERA_CONFIG.MAX_DISTANCE}
        />

        <FloorPlanScene workstations={workstations} onWorkstationClick={onWorkstationClick} />
      </Canvas>

      {/* 操作提示 */}
      <ControlHints workstationCount={workstations.length} />
    </div>
  );
};

const EmptyView: React.FC = () => (
  <div style={styles.emptyContainer}>
    <div style={styles.emptyIcon}>🪑</div>
    <div style={styles.emptyTitle}>暂无工位数据</div>
    <div style={styles.emptySubtitle}>请先为该楼层添加工位信息</div>
  </div>
);

interface ControlHintsProps {
  workstationCount: number;
}

const ControlHints: React.FC<ControlHintsProps> = ({ workstationCount }) => (
  <div style={styles.controlHints}>
    <span>🖱️ 拖拽平移</span>
    <span>🔍 滚轮缩放</span>
    <span>👆 点击工位</span>
    <span>📊 {workstationCount} 工位</span>
  </div>
);

// ============ 样式 ============

const styles = {
  container: {
    width: "100%",
    height: "100%",
    position: "relative" as const,
  },
  emptyContainer: {
    width: "100%",
    height: "100%",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    flexDirection: "column" as const,
    gap: 16,
    background: "#f5f5f5",
    color: "var(--theme-text-tertiary, #8c8c8c)",
  },
  emptyIcon: {
    fontSize: 48,
  },
  emptyTitle: {
    fontSize: 16,
  },
  emptySubtitle: {
    fontSize: 12,
    opacity: 0.7,
  },
  controlHints: {
    position: "absolute" as const,
    bottom: 16,
    left: 16,
    background: "rgba(255,255,255,0.95)",
    backdropFilter: "blur(10px)",
    padding: "10px 14px",
    borderRadius: 6,
    color: "var(--theme-text-secondary, #595959)",
    fontSize: 12,
    display: "flex",
    gap: 16,
    alignItems: "center",
    border: "1px solid #e8e8e8",
    boxShadow: "0 2px 8px rgba(0,0,0,0.06)",
  },
  tooltip: {
    background: "rgba(255, 255, 255, 0.95)",
    borderRadius: 8,
    padding: "12px 16px",
    boxShadow: "0 4px 12px rgba(0,0,0,0.15)",
    minWidth: 180,
    border: "1px solid #e8e8e8",
  },
  tooltipName: {
    fontSize: 14,
    fontWeight: "bold",
    color: "var(--theme-text-primary, #262626)",
    marginBottom: 8,
  },
  tooltipContent: {
    fontSize: 12,
    color: "var(--theme-text-secondary, #595959)",
    display: "flex",
    flexDirection: "column" as const,
    gap: 4,
  },
  tooltipRow: {
    display: "flex",
    justifyContent: "space-between",
  },
  legend: {
    display: "flex",
    flexDirection: "column" as const,
    gap: "8px",
    color: "white",
    fontSize: "14px",
    fontWeight: "bold",
    textShadow: "0 1px 3px rgba(0,0,0,0.3)",
  },
};

export default FloorPlan3D;
