/**
 * 3D 工位平面图组件
 * 使用 Three.js + React Three Fiber 渲染交互式 3D 工位平面图
 */

import { useRef, useState, useMemo } from 'react';
import { Canvas, useFrame } from '@react-three/fiber';
import { OrbitControls, PerspectiveCamera, Html } from '@react-three/drei';
import * as THREE from 'three';

// 工位数据接口
interface WorkstationData {
  id: string;
  name: string;
  code: string;
  status: number; // 0=空闲, 1=占用, 2=维护
  type: number; // 0=固定, 1=灵活, 2=管理
  positionX?: number;
  positionY?: number;
  rotation?: number; // 旋转角度
}

interface FloorPlan3DProps {
  workstations: WorkstationData[];
  onWorkstationClick?: (workstation: WorkstationData) => void;
}

// 3D 文本组件 - 使用HTML避免字体加载问题
const Text3D: React.FC<{
  position: [number, number, number];
  fontSize: number;
  color: string;
  children: string;
}> = ({ position, fontSize, color, children }) => {
  return (
    <Html position={position} center distanceFactor={10} style={{ pointerEvents: 'none' }}>
      <div style={{
        color: color,
        fontSize: `${fontSize * 10}px`,
        fontWeight: 'bold',
        textShadow: '0 1px 3px rgba(0,0,0,0.3)',
        whiteSpace: 'nowrap',
        userSelect: 'none',
      }}>
        {children}
      </div>
    </Html>
  );
};

// 单个工位组件
const Workstation3D: React.FC<{
  workstation: WorkstationData;
  position: [number, number, number];
  onClick?: () => void;
  isSelected?: boolean;
}> = ({ workstation, position, onClick, isSelected }) => {
  const groupRef = useRef<THREE.Group>(null);
  const contentRef = useRef<THREE.Group>(null);
  const [hovered, setHovered] = useState(false);

  // 旋转角度（转换为弧度）
  const rotation = (workstation.rotation || 0) * (Math.PI / 180);

  // 根据状态选择颜色（适配白色背景）
  const getColor = () => {
    if (workstation.status === 1) return 0xd32f2f; // 占用 - 深红色
    if (workstation.status === 2) return 0xf57c00; // 维护 - 深橙色
    if (workstation.type === 1) return 0x7b1fa2; // 灵活 - 深紫色
    if (workstation.type === 2) return 0x13c2c2; // 管理 - 深青色
    return 0x388e3c; // 空闲固定 - 深绿色
  };

  const statusColor = getColor();
  const deskWidth = 1.2;
  const deskDepth = 0.8;
  const deskHeight = 0.04;
  const legHeight = 0.75;  // 桌腿高度
  const deskTopY = legHeight + deskHeight / 2;  // 桌面顶部Y坐标

  // 悬停动画 - 整体轻微抬起
  useFrame(() => {
    if (groupRef.current) {
      const targetY = hovered || isSelected ? 0.05 : 0;
      groupRef.current.position.y = THREE.MathUtils.lerp(groupRef.current.position.y, targetY, 0.1);
    }
  });

  return (
    <group ref={groupRef} position={position}>
      <group ref={contentRef} rotation={[0, rotation, 0]}>
      {/* 桌腿 - 四条腿 */}
      <mesh position={[-deskWidth / 2 + 0.08, legHeight / 2, -deskDepth / 2 + 0.08]} castShadow>
        <boxGeometry args={[0.06, legHeight, 0.06]} />
        <meshStandardMaterial color={0x2d2d2d} />
      </mesh>
      <mesh position={[deskWidth / 2 - 0.08, legHeight / 2, -deskDepth / 2 + 0.08]} castShadow>
        <boxGeometry args={[0.06, legHeight, 0.06]} />
        <meshStandardMaterial color={0x2d2d2d} />
      </mesh>
      <mesh position={[-deskWidth / 2 + 0.08, legHeight / 2, deskDepth / 2 - 0.08]} castShadow>
        <boxGeometry args={[0.06, legHeight, 0.06]} />
        <meshStandardMaterial color={0x2d2d2d} />
      </mesh>
      <mesh position={[deskWidth / 2 - 0.08, legHeight / 2, deskDepth / 2 - 0.08]} castShadow>
        <boxGeometry args={[0.06, legHeight, 0.06]} />
        <meshStandardMaterial color={0x2d2d2d} />
      </mesh>

      {/* 桌面 - 抬高到桌腿顶部 */}
      <mesh
        position={[0, deskTopY, 0]}
        onClick={onClick}
        onPointerOver={() => setHovered(true)}
        onPointerOut={() => setHovered(false)}
        castShadow
        receiveShadow
      >
        <boxGeometry args={[deskWidth, deskHeight, deskDepth]} />
        <meshStandardMaterial
          color={0xf5f5f5}
          roughness={0.8}
          metalness={0.0}
        />
      </mesh>

      {/* 桌面边框 */}
      <mesh position={[0, deskTopY + 0.01, 0]}>
        <boxGeometry args={[deskWidth + 0.02, 0.01, deskDepth + 0.02]} />
        <meshStandardMaterial color={statusColor} />
      </mesh>

      {/* 显示器 - 在桌面上 */}
      <group position={[0, deskTopY + 0.35, -deskDepth / 2 + 0.15]}>
        {/* 显示器支架 */}
        <mesh position={[0, 0.12, 0]} castShadow>
          <boxGeometry args={[0.12, 0.24, 0.08]} />
          <meshStandardMaterial color={0x333333} />
        </mesh>
        {/* 显示器屏幕 */}
        <mesh position={[0, 0.28, 0]} castShadow>
          <boxGeometry args={[0.45, 0.3, 0.02]} />
          <meshStandardMaterial color={0x1a1a1a} roughness={0.3} metalness={0.8} />
        </mesh>
        {/* 显示器边框 */}
        <mesh position={[0, 0.28, 0]}>
          <boxGeometry args={[0.47, 0.32, 0.025]} />
          <meshStandardMaterial color={0x2d2d2d} />
        </mesh>
        {/* 屏幕发光效果 */}
        {(hovered || isSelected) && (
          <mesh position={[0, 0.28, 0.01]}>
            <planeGeometry args={[0.43, 0.26]} />
            <meshBasicMaterial color={statusColor} transparent opacity={0.4} />
          </mesh>
        )}
      </group>

      {/* 键盘 - 在桌面上 */}
      <mesh position={[0.1, deskTopY + 0.015, 0.15]} castShadow>
        <boxGeometry args={[0.32, 0.015, 0.14]} />
        <meshStandardMaterial color={0x2d2d2d} />
      </mesh>
      {/* 键盘按键细节 */}
      <mesh position={[0.1, deskTopY + 0.023, 0.15]}>
        <boxGeometry args={[0.3, 0.008, 0.12]} />
        <meshStandardMaterial color={0x1a1a1a} />
      </mesh>

      {/* 鼠标垫 - 在桌面上 */}
      <mesh position={[0.4, deskTopY + 0.01, 0.2]} castShadow>
        <boxGeometry args={[0.18, 0.01, 0.22]} />
        <meshStandardMaterial color={0x4a4a4a} roughness={0.9} />
      </mesh>

      {/* 椅子 - 在桌面前方，面向桌子 */}
      <group position={[0, 0, deskDepth / 2 + 0.35]}>
        {/* 椅子座位 */}
        <mesh position={[0, 0.45, 0]} castShadow>
          <boxGeometry args={[0.42, 0.04, 0.42]} />
          <meshStandardMaterial color={0x3d3d3d} />
        </mesh>
        {/* 椅子靠背 - 面向桌子（前侧） */}
        <mesh position={[0, 0.7, 0.18]} castShadow>
          <boxGeometry args={[0.42, 0.35, 0.04]} />
          <meshStandardMaterial color={statusColor} />
        </mesh>
        {/* 左扶手 */}
        <mesh position={[-0.23, 0.52, 0]} castShadow>
          <boxGeometry args={[0.04, 0.12, 0.04]} />
          <meshStandardMaterial color={0x2d2d2d} />
        </mesh>
        {/* 右扶手 */}
        <mesh position={[0.23, 0.52, 0]} castShadow>
          <boxGeometry args={[0.04, 0.12, 0.04]} />
          <meshStandardMaterial color={0x2d2d2d} />
        </mesh>
        {/* 椅子腿 */}
        <mesh position={[-0.18, 0.22, 0.18]} castShadow><boxGeometry args={[0.025, 0.44, 0.025]} /><meshStandardMaterial color={0x1a1a1a} /></mesh>
        <mesh position={[0.18, 0.22, 0.18]} castShadow><boxGeometry args={[0.025, 0.44, 0.025]} /><meshStandardMaterial color={0x1a1a1a} /></mesh>
        <mesh position={[-0.18, 0.22, -0.18]} castShadow><boxGeometry args={[0.025, 0.44, 0.025]} /><meshStandardMaterial color={0x1a1a1a} /></mesh>
        <mesh position={[0.18, 0.22, -0.18]} castShadow><boxGeometry args={[0.025, 0.44, 0.025]} /><meshStandardMaterial color={0x1a1a1a} /></mesh>
      </group>

      {/* 选中高亮 */}
      {(hovered || isSelected) && (
        <mesh position={[0, deskTopY, 0]}>
          <boxGeometry args={[deskWidth + 0.05, 0.02, deskDepth + 0.05]} />
          <meshBasicMaterial color={statusColor} transparent opacity={0.3} />
        </mesh>
      )}

      {/* 工位标签 */}
      <Text3D
        position={[0, deskTopY + 1.1, 0]}
        fontSize={0.15}
        color="white"
      >
        {workstation.name || workstation.code}
      </Text3D>

      {/* 鼠标悬停显示详细信息 */}
      {hovered && (
        <Html
          position={[0, deskTopY + 1.6, 0]}
          center
          distanceFactor={10}
          style={{
            pointerEvents: 'none',
          }}
        >
          <div style={{
            background: 'rgba(255, 255, 255, 0.95)',
            borderRadius: 8,
            padding: '12px 16px',
            boxShadow: '0 4px 12px rgba(0,0,0,0.15)',
            minWidth: 180,
            border: '1px solid #e8e8e8',
          }}>
            <div style={{
              fontSize: 14,
              fontWeight: 'bold',
              color: '#262626',
              marginBottom: 8,
            }}>
              {workstation.name || workstation.code}
            </div>
            <div style={{ fontSize: 12, color: '#595959', display: 'flex', flexDirection: 'column', gap: 4 }}>
              <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                <span>状态：</span>
                <span style={{
                  fontWeight: 'bold',
                  color: workstation.status === 0 ? '#52c41a' : workstation.status === 1 ? '#ff4d4f' : '#faad14',
                }}>
                  {workstation.status === 0 ? '空闲' : workstation.status === 1 ? '占用' : '维护'}
                </span>
              </div>
              <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                <span>类型：</span>
                <span style={{ fontWeight: 'bold', color: workstation.type === 0 ? '#1890ff' : workstation.type === 1 ? '#722ed1' : '#13c2c2' }}>
                  {workstation.type === 0 ? '固定' : workstation.type === 1 ? '灵活' : '管理'}
                </span>
              </div>
            </div>
          </div>
        </Html>
      )}
      </group>
    </group>
  );
};

// 3D 平面图场景
const FloorPlanScene: React.FC<FloorPlan3DProps> = ({ workstations, onWorkstationClick }) => {
  const groupRef = useRef<THREE.Group>(null);
  const [selectedId, setSelectedId] = useState<string | null>(null);

  // 自动计算工位位置（如果没有预设位置）
  const workstationPositions = useMemo(() => {
    const positions: Array<{ id: string; x: number; z: number }> = [];
    const gridSize = 8;
    const cellSize = 1.5;  // 缩小间距，从1.2改为1.5

    workstations.forEach((ws, index) => {
      if (ws.positionX !== undefined && ws.positionY !== undefined) {
        // 使用预设位置（需要转换坐标系）
        positions.push({
          id: ws.id,
          x: (ws.positionX - 50) / 10,
          z: (ws.positionY - 50) / 10,
        });
      } else {
        // 自动排列：从左到右，从上到下
        const row = Math.floor(index / gridSize);
        const col = index % gridSize;
        positions.push({
          id: ws.id,
          x: (col - gridSize / 2) * cellSize,
          z: (row - gridSize / 2) * cellSize,
        });
      }
    });

    return positions;
  }, [workstations]);

  const positionMap = useMemo(() => {
    const map = new Map<string, { x: number; z: number }>();
    workstationPositions.forEach(pos => {
      map.set(pos.id, pos);
    });
    return map;
  }, [workstationPositions]);

  return (
    <group ref={groupRef}>
      {/* 环境光 */}
      <ambientLight intensity={0.7} />
      <directionalLight position={[10, 15, 10]} intensity={0.8} castShadow />
      <directionalLight position={[-10, 10, -10]} intensity={0.5} />
      <directionalLight position={[0, -10, 0]} intensity={0.3} />

      {/* 无限延伸的白色地面 - 移除网格 */}
      <mesh rotation={[-Math.PI / 2, 0, 0]} position={[0, -0.01, 0]} receiveShadow>
        <planeGeometry args={[200, 200]} />
        <meshStandardMaterial color="#ffffff" opacity={1} transparent={false} />
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
            onClick={() => {
              setSelectedId(ws.id);
              if (onWorkstationClick) {
                onWorkstationClick(ws);
              }
            }}
            isSelected={selectedId === ws.id}
          />
        );
      })}

      {/* 图例 - 浮在地面上方 */}
      <Html position={[-12, 0.3, -12]} center distanceFactor={10} style={{ pointerEvents: 'none' }}>
        <div style={{ display: 'flex', flexDirection: 'column', gap: '8px', color: 'white', fontSize: '14px', fontWeight: 'bold', textShadow: '0 1px 3px rgba(0,0,0,0.3)' }}>
          <div style={{ color: '#388e3c' }}>● 空闲</div>
          <div style={{ color: '#d32f2f' }}>● 占用</div>
          <div style={{ color: '#f57c00' }}>● 维护</div>
        </div>
      </Html>
    </group>
  );
};

// 主组件
const FloorPlan3D: React.FC<FloorPlan3DProps> = ({ workstations, onWorkstationClick }) => {
  // 如果没有工位，显示提示
  if (workstations.length === 0) {
    return (
      <div style={{
        width: '100%',
        height: '100%',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        flexDirection: 'column',
        gap: 16,
        background: '#f5f5f5',
        color: '#8c8c8c',
      }}>
        <div style={{ fontSize: 48 }}>🪑</div>
        <div style={{ fontSize: 16 }}>暂无工位数据</div>
        <div style={{ fontSize: 12, opacity: 0.7 }}>请先为该楼层添加工位信息</div>
      </div>
    );
  }

  return (
    <div style={{ width: '100%', height: '100%', position: 'relative' }}>
      <Canvas
        shadows
        dpr={[1, 2]}
        gl={{ antialias: true, alpha: true }}
        style={{ background: '#f5f5f5' }}
      >
        <PerspectiveCamera makeDefault position={[0, 15, 15]} fov={50} />
        <OrbitControls
          enableZoom={true}
          enablePan={true}
          minPolarAngle={0}
          maxPolarAngle={Math.PI / 2.5}
          minDistance={5}
          maxDistance={40}
        />

        <FloorPlanScene workstations={workstations} onWorkstationClick={onWorkstationClick} />
      </Canvas>

      {/* 操作提示 */}
      <div style={{
        position: 'absolute',
        bottom: 16,
        left: 16,
        background: 'rgba(255,255,255,0.95)',
        backdropFilter: 'blur(10px)',
        padding: '10px 14px',
        borderRadius: 6,
        color: '#595959',
        fontSize: 12,
        display: 'flex',
        gap: 16,
        alignItems: 'center',
        border: '1px solid #e8e8e8',
        boxShadow: '0 2px 8px rgba(0,0,0,0.06)',
      }}>
        <span>🖱️ 拖拽平移</span>
        <span>🔍 滚轮缩放</span>
        <span>👆 点击工位</span>
        <span>📊 {workstations.length} 工位</span>
      </div>
    </div>
  );
};

export default FloorPlan3D;
