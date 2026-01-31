/**
 * CAD 楼层平面图编辑器页面
 * 用于测试和使用CAD风格的平面图编辑功能
 */

import { useState, useEffect, useCallback } from 'react';
import { Card, Spin, message, Button, Space, Alert } from 'antd';
import { ReloadOutlined, EditOutlined, EyeOutlined } from '@ant-design/icons';
import { useParams } from 'react-router-dom';
import { CADFloorPlanEditor } from '@/components/cad-editor';
import type { FloorPlanData, Wall, Door } from '@/components/cad-editor/types';
import { wallApi, doorApi } from '@/lib/opsApi';
import type { WorkstationNode } from '@/components/shared/FloorPlanEditor.types';

// ============================================================================
// Constants
// ============================================================================

const API_PAGE_SIZE = 1000;

const DEFAULT_FLOOR_PLAN_CONFIG = {
  width: 2000,
  height: 2000,
  gridSize: 20,
  showGrid: true,
  snapToGrid: true,
} as const;

const TEMP_ID_PREFIX = {
  WALL: 'wall_',
  DOOR: 'door_',
} as const;

const MESSAGES = {
  PARAM_ERROR: '缺少楼层ID参数，请从楼层列表页面进入',
  LOAD_FAILED: '加载平面图失败',
  LOAD_RETRY_FAILED: '无法加载平面图数据，请稍后重试',
  SAVE_SUCCESS: '保存成功',
  SAVE_FAILED: '保存失败',
  LOADING: '加载中...',
} as const;

const BUTTON_LABELS = {
  EDIT_MODE: '编辑模式',
  PREVIEW_MODE: '预览模式',
  REFRESH: '刷新',
  RETRY: '重试',
} as const;

// ============================================================================
// Types
// ============================================================================

type WallApiData = Omit<Wall, 'points'> & { points: string };
type DoorApiData = Omit<Door, 'position'> & { position: string };

// ============================================================================
// Helper Functions
// ============================================================================

/**
 * 解析墙体的 points 数据
 */
function parseWallPoints(wall: WallApiData | Wall): Wall {
  return {
    ...wall,
    points: typeof wall.points === 'string'
      ? JSON.parse(wall.points)
      : wall.points,
  };
}

/**
 * 解析门的 position 数据
 */
function parseDoorPosition(door: DoorApiData | Door): Door {
  return {
    ...door,
    position: typeof door.position === 'string'
      ? JSON.parse(door.position)
      : door.position,
  };
}

/**
 * 判断是否为临时ID（新建未保存的记录）
 */
function isTempId(id: string, prefix: string): boolean {
  return id.startsWith(prefix);
}

/**
 * 创建空平面图数据
 */
function createEmptyFloorPlanData(floorId: string, floorName: string): FloorPlanData {
  return {
    floorId,
    floorName,
    ...DEFAULT_FLOOR_PLAN_CONFIG,
    walls: [],
    doors: [],
    workstations: [],
  };
}

/**
 * 保存单个墙体
 */
async function saveWall(wall: Wall): Promise<void> {
  const wallData: WallApiData = {
    ...wall,
    points: JSON.stringify(wall.points),
    type: wall.type as 'straight' | 'curved' | 'l_shaped' | 'polyline',
  };

  if (isTempId(wall.id, TEMP_ID_PREFIX.WALL)) {
    await wallApi.create(wallData);
  } else {
    await wallApi.update(wall.id, wallData);
  }
}

/**
 * 保存单个门
 */
async function saveDoor(door: Door): Promise<void> {
  const doorData: DoorApiData = {
    ...door,
    position: JSON.stringify(door.position),
  };

  if (isTempId(door.id, TEMP_ID_PREFIX.DOOR)) {
    await doorApi.create(doorData);
  } else {
    await doorApi.update(door.id, doorData);
  }
}

// ============================================================================
// Components
// ============================================================================

/**
 * 加载中状态组件
 */
function LoadingState() {
  return (
    <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh' }}>
      <Spin size="large">
        <div style={{ padding: 24, textAlign: 'center' }}>{MESSAGES.LOADING}</div>
      </Spin>
    </div>
  );
}

/**
 * 参数错误组件
 */
function ParamErrorAlert() {
  return (
    <Alert
      message="参数错误"
      description={MESSAGES.PARAM_ERROR}
      type="error"
      showIcon
    />
  );
}

/**
 * 加载失败组件
 */
interface LoadErrorAlertProps {
  onRetry: () => void;
}

function LoadErrorAlert({ onRetry }: LoadErrorAlertProps) {
  return (
    <Alert
      message="加载失败"
      description={MESSAGES.LOAD_RETRY_FAILED}
      type="error"
      showIcon
      action={
        <Button type="primary" onClick={onRetry}>
          {BUTTON_LABELS.RETRY}
        </Button>
      }
    />
  );
}

// ============================================================================
// Main Component
// ============================================================================

export default function FloorPlanEditorPage() {
  const { floorId } = useParams<{ floorId: string }>();
  const [loading, setLoading] = useState(true);
  const [floorPlanData, setFloorPlanData] = useState<FloorPlanData | null>(null);
  const [isEditMode, setIsEditMode] = useState(false);
  const [floorName] = useState('平面图编辑');

  // 参数错误提前返回
  if (!floorId) {
    return <ParamErrorAlert />;
  }

  /**
   * 加载平面图数据
   */
  const loadFloorPlanData = useCallback(async () => {
    if (!floorId) return;

    setLoading(true);
    try {
      // 并行加载墙体和门数据
      const [wallsResult, doorsResult] = await Promise.all([
        wallApi.list({ floorId, current: 1, pageSize: API_PAGE_SIZE }),
        doorApi.list({ floorId, current: 1, pageSize: API_PAGE_SIZE }),
      ]);

      const walls = wallsResult.data?.list || [];
      const doors = doorsResult.data?.list || [];

      // 解析JSON格式的坐标数据
      const parsedWalls: Wall[] = walls.map(parseWallPoints);
      const parsedDoors: Door[] = doors.map(parseDoorPosition);

      // TODO: 加载工位数据
      const workstations: WorkstationNode[] = [];

      setFloorPlanData({
        floorId,
        floorName,
        ...DEFAULT_FLOOR_PLAN_CONFIG,
        walls: parsedWalls,
        doors: parsedDoors,
        workstations,
      });
    } catch (error) {
      console.error('加载平面图失败:', error);
      message.error(MESSAGES.LOAD_FAILED);
      // 使用空数据初始化
      setFloorPlanData(createEmptyFloorPlanData(floorId, floorName));
    } finally {
      setLoading(false);
    }
  }, [floorId, floorName]);

  /**
   * 保存平面图
   */
  const handleSave = async (data: FloorPlanData) => {
    try {
      // 保存所有墙体
      await Promise.all(data.walls.map(saveWall));

      // 保存所有门
      await Promise.all(data.doors.map(saveDoor));

      message.success(MESSAGES.SAVE_SUCCESS);
      await loadFloorPlanData();
    } catch (error) {
      console.error('保存失败:', error);
      message.error(MESSAGES.SAVE_FAILED);
    }
  };

  useEffect(() => {
    loadFloorPlanData();
  }, [loadFloorPlanData]);

  if (loading) {
    return <LoadingState />;
  }

  if (!floorPlanData) {
    return <LoadErrorAlert onRetry={loadFloorPlanData} />;
  }

  return (
    <div style={{ height: '100vh', display: 'flex', flexDirection: 'column' }}>
      <Card
        title={`${floorName} - 平面图编辑`}
        extra={
          <Space>
            <Button
              icon={isEditMode ? <EyeOutlined /> : <EditOutlined />}
              onClick={() => setIsEditMode(!isEditMode)}
            >
              {isEditMode ? BUTTON_LABELS.PREVIEW_MODE : BUTTON_LABELS.EDIT_MODE}
            </Button>
            <Button icon={<ReloadOutlined />} onClick={loadFloorPlanData}>
              {BUTTON_LABELS.REFRESH}
            </Button>
          </Space>
        }
        style={{ flex: 1, display: 'flex', flexDirection: 'column' }}
        styles={{ body: { flex: 1, padding: 0, overflow: 'hidden' } }}
      >
        <CADFloorPlanEditor
          floorId={floorId}
          floorName={floorName}
          walls={floorPlanData.walls}
          doors={floorPlanData.doors}
          workstations={floorPlanData.workstations}
          onSave={handleSave}
          readOnly={!isEditMode}
        />
      </Card>
    </div>
  );
}
