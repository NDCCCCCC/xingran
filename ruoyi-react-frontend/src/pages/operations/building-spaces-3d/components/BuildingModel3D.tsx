/**
 * 3D 楼宇模型组件
 * 使用 Three.js + React Three Fiber 渲染交互式 3D 楼宇模型
 */

import { useRef, useState } from "react";
import { Canvas, useFrame } from "@react-three/fiber";
import { OrbitControls, PerspectiveCamera, Html, useCursor } from "@react-three/drei";
import * as THREE from "three";
import type { FloorData } from "./types";
import { SCENE_DIMENSIONS } from "./constants";
import { getFloorColor, getFloorStatusText } from "./utils";

interface BuildingModel3DProps {
  floors: FloorData[];
  onFloorClick?: (floor: FloorData) => void;
  selectedFloorId?: string;
}

interface Floor3DProps {
  floor: FloorData;
  position: [number, number, number];
  onClick?: () => void;
  isSelected?: boolean;
}

const EMPTY_STATE = {
  icon: "🏢",
  title: "暂无楼层数据",
  description: "请先为该楼宇添加楼层信息",
};

// 单个楼层组件
const Floor3D: React.FC<Floor3DProps> = ({ floor, position, onClick, isSelected }) => {
  const meshRef = useRef<THREE.Mesh>(null);
  const [hovered, setHovered] = useState(false);
  useCursor(hovered);

  const targetScale = hovered || isSelected ? 1.1 : 1;

  useFrame(() => {
    if (meshRef.current) {
      meshRef.current.scale.x = THREE.MathUtils.lerp(meshRef.current.scale.x, targetScale, 0.1);
      meshRef.current.scale.y = THREE.MathUtils.lerp(meshRef.current.scale.y, targetScale, 0.1);
      meshRef.current.scale.z = THREE.MathUtils.lerp(meshRef.current.scale.z, targetScale, 0.1);
    }
  });

  const baseColor = getFloorColor(floor);
  const { HEIGHT: floorHeight, SIZE: floorSize } = SCENE_DIMENSIONS.FLOOR;

  return (
    <group position={position}>
      {/* 楼层主体 */}
      <mesh
        ref={meshRef}
        position={[0, floorHeight / 2, 0]}
        onClick={onClick}
        onPointerOver={() => setHovered(true)}
        onPointerOut={() => setHovered(false)}
        castShadow
        receiveShadow
      >
        <boxGeometry args={[floorSize, floorHeight, floorSize]} />
        <meshStandardMaterial
          color={baseColor}
          transparent
          opacity={0.9}
          roughness={0.5}
          metalness={0.1}
        />
      </mesh>

      {/* 悬停边框 */}
      {hovered && (
        <mesh position={[0, floorHeight / 2, 0]}>
          <boxGeometry args={[floorSize + 0.15, floorHeight + 0.05, floorSize + 0.15]} />
          <meshBasicMaterial color={0xffa000} wireframe />
        </mesh>
      )}

      {/* 楼层标签 */}
      <Html
        position={[0, floorHeight + 0.5, 0]}
        center
        distanceFactor={8}
        style={{ pointerEvents: "none", userSelect: "none" }}
      >
        <div style={{
          background: hovered || isSelected ? "rgba(24, 144, 255, 0.95)" : "rgba(0, 0, 0, 0.75)",
          color: "var(--theme-text-inverse, #fff)",
          padding: "8px 16px",
          borderRadius: 8,
          fontSize: "18px",
          fontWeight: "bold",
          whiteSpace: "nowrap",
          boxShadow: hovered ? "0 4px 12px rgba(24, 144, 255, 0.4)" : "0 2px 8px rgba(0,0,0,0.15)",
          transition: "all 0.2s",
        }}>
          {floor.name || floor.floorNo}
        </div>
      </Html>

      {/* 悬停详情 */}
      {hovered && (
        <Html
          position={[0, floorHeight + 2.5, 0]}
          center
          distanceFactor={8}
          style={{ pointerEvents: "none" }}
        >
          <div style={{
            background: "#fff",
            borderRadius: 12,
            padding: 16,
            boxShadow: "0 6px 20px rgba(0,0,0,0.2)",
            minWidth: 220,
            border: "1px solid #e8e8e8",
          }}>
            <div style={{
              fontSize: 16,
              fontWeight: "bold",
              color: "var(--theme-text-primary, #262626)",
              marginBottom: 12,
              paddingBottom: 10,
              borderBottom: "1px solid #f0f0f0",
            }}>
              {floor.name || floor.floorNo}
            </div>
            <div style={{ fontSize: 14, color: "var(--theme-text-secondary, #595959)", display: "flex", flexDirection: "column", gap: 6 }}>
              <div style={{ display: "flex", justifyContent: "space-between" }}>
                <span>工位数量：</span>
                <span style={{ fontWeight: "bold", color: "var(--theme-text-accent, #1890ff)", fontSize: 15 }}>
                  {floor.workstationCount} 个
                </span>
              </div>
              <div style={{ display: "flex", justifyContent: "space-between" }}>
                <span>楼层状态：</span>
                <span style={{
                  fontWeight: "bold",
                  color: floor.status === 0 ? "var(--theme-success, #52c41a)" : "#ff4d4f",
                  fontSize: 14
                }}>
                  {getFloorStatusText(floor.status)}
                </span>
              </div>
              <div style={{ fontSize: 13, color: "var(--theme-text-tertiary, #8c8c8c)", marginTop: 6, textAlign: "center", paddingTop: 4 }}>
                👇 点击查看平面图
              </div>
            </div>
          </div>
        </Html>
      )}
    </group>
  );
};

// 3D 场景组件
const BuildingScene: React.FC<BuildingModel3DProps> = ({ floors, onFloorClick, selectedFloorId }) => {
  const groupRef = useRef<THREE.Group>(null);

  const getFloorPosition = (index: number): [number, number, number] => {
    const y = 0.3 + index * SCENE_DIMENSIONS.FLOOR.SPACING;
    return [0, y, 0];
  };

  return (
    <group ref={groupRef}>
      {/* 环境光 */}
      <ambientLight intensity={0.7} />
      <directionalLight position={[10, 10, 5]} intensity={1} castShadow />
      <directionalLight position={[-10, 10, -5]} intensity={0.5} />
      <directionalLight position={[0, -10, 0]} intensity={0.3} />

      {/* 楼层堆叠 */}
      {floors.map((floor, index) => (
        <Floor3D
          key={floor.id}
          floor={floor}
          position={getFloorPosition(index)}
          onClick={() => onFloorClick?.(floor)}
          isSelected={selectedFloorId === floor.id}
        />
      ))}
    </group>
  );
};

// 主组件
const BuildingModel3D: React.FC<BuildingModel3DProps> = ({ floors, onFloorClick, selectedFloorId }) => {
  if (floors.length === 0) {
    return (
      <div style={{
        width: "100%",
        height: "100%",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        flexDirection: "column",
        gap: 16,
        background: "#f5f5f5",
        color: "var(--theme-text-tertiary, #8c8c8c)",
      }}>
        <div style={{ fontSize: 48 }}>{EMPTY_STATE.icon}</div>
        <div style={{ fontSize: 16 }}>{EMPTY_STATE.title}</div>
        <div style={{ fontSize: 12, opacity: 0.7 }}>{EMPTY_STATE.description}</div>
      </div>
    );
  }

  return (
    <div style={{ width: "100%", height: "100%", position: "relative" }}>
      <Canvas
        shadows
        dpr={[1, 2]}
        gl={{ antialias: true, alpha: true }}
        style={{ background: "#f5f5f5" }}
      >
        <PerspectiveCamera makeDefault position={[12, 10, 12]} fov={50} />
        <OrbitControls
          enableZoom={true}
          enablePan={true}
          enableRotate={true}
          minDistance={8}
          maxDistance={40}
        />

        <BuildingScene floors={floors} onFloorClick={onFloorClick} selectedFloorId={selectedFloorId} />
      </Canvas>

      {/* 操作提示 */}
      <div style={{
        position: "absolute",
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
      }}>
        <span>🖱️ 拖拽旋转</span>
        <span>🔍 滚轮缩放</span>
        <span>👆 点击楼层查看详情</span>
        <span>💫 悬停查看详情</span>
      </div>
    </div>
  );
};

export default BuildingModel3D;
