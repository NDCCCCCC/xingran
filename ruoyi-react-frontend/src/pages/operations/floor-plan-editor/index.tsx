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

export default function FloorPlanEditorPage() {
  const { floorId } = useParams<{ floorId: string }>();
  const [loading, setLoading] = useState(true);
  const [floorPlanData, setFloorPlanData] = useState<FloorPlanData | null>(null);
  const [isEditMode, setIsEditMode] = useState(false);
  const [floorName, setFloorName] = useState('平面图编辑');

  // 如果没有楼层ID，显示错误提示
  if (!floorId) {
    return (
      <Alert
        message="参数错误"
        description="缺少楼层ID参数，请从楼层列表页面进入"
        type="error"
        showIcon
      />
    );
  }

  // 加载平面图数据
  const loadFloorPlan = useCallback(async () => {
    if (!floorId) return;
    setLoading(true);
    try {
      // 并行加载墙体和门数据
      const [wallsResult, doorsResult] = await Promise.all([
        wallApi.list({ floorId, current: 1, pageSize: 1000 }),
        doorApi.list({ floorId, current: 1, pageSize: 1000 }),
      ]);

      const walls = wallsResult.data?.list || [];
      const doors = doorsResult.data?.list || [];

      // 解析JSON格式的坐标数据
      const parsedWalls: Wall[] = walls.map((wall: any) => ({
        ...wall,
        points: typeof wall.points === 'string' ? JSON.parse(wall.points) : wall.points,
      }));

      const parsedDoors: Door[] = doors.map((door: any) => ({
        ...door,
        position: typeof door.position === 'string' ? JSON.parse(door.position) : door.position,
      }));

      // TODO: 加载工位数据
      const workstations: WorkstationNode[] = [];

      setFloorPlanData({
        floorId,
        floorName,
        width: 2000,
        height: 2000,
        walls: parsedWalls,
        doors: parsedDoors,
        workstations,
        gridSize: 20,
        showGrid: true,
        snapToGrid: true,
      });
    } catch (error) {
      console.error('加载平面图失败:', error);
      message.error('加载平面图失败');
      // 使用空数据初始化
      setFloorPlanData({
        floorId,
        floorName,
        width: 2000,
        height: 2000,
        walls: [],
        doors: [],
        workstations: [],
        gridSize: 20,
        showGrid: true,
        snapToGrid: true,
      });
    } finally {
      setLoading(false);
    }
  }, [floorId]);

  // 保存平面图
  const handleSave = async (data: FloorPlanData) => {
    try {
      // 保存墙体
      for (const wall of data.walls) {
        // 将 points 从 Point[] 转换为 JSON 字符串
        const wallData = {
          ...wall,
          points: JSON.stringify(wall.points),
          type: wall.type as 'straight' | 'curved' | 'l_shaped' | 'polyline',
        };

        if (wall.id.startsWith('wall_')) {
          // 新建的墙体
          await wallApi.create(wallData);
        } else {
          // 更新现有墙体
          await wallApi.update(wall.id, wallData);
        }
      }

      // 保存门
      for (const door of data.doors) {
        if (door.id.startsWith('door_')) {
          // 新建的门
          await doorApi.create({
            ...door,
            position: JSON.stringify(door.position),
          });
        } else {
          // 更新现有的门
          await doorApi.update(door.id, {
            ...door,
            position: JSON.stringify(door.position),
          });
        }
      }

      message.success('保存成功');
      await loadFloorPlan(); // 重新加载数据
    } catch (error) {
      console.error('保存失败:', error);
      message.error('保存失败');
    }
  };

  useEffect(() => {
    loadFloorPlan();
  }, [loadFloorPlan]);

  if (loading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh' }}>
        <Spin size="large">
          <div style={{ padding: 24, textAlign: 'center' }}>加载中...</div>
        </Spin>
      </div>
    );
  }

  if (!floorPlanData) {
    return (
      <Alert
        message="加载失败"
        description="无法加载平面图数据，请稍后重试"
        type="error"
        showIcon
        action={
          <Button type="primary" onClick={loadFloorPlan}>
            重试
          </Button>
        }
      />
    );
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
              {isEditMode ? '预览模式' : '编辑模式'}
            </Button>
            <Button icon={<ReloadOutlined />} onClick={loadFloorPlan}>
              刷新
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
