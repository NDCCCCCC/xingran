/**
 * 楼层管理页面
 */

import { useState, useEffect, useCallback } from 'react';
import type { FC } from 'react';
import {
  Table, Button, Space, Modal, Form, Input, InputNumber, Select, Popconfirm,
  Tag, Card, Row, Col, Statistic, Alert, Radio, Spin, message,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  PlusOutlined, EditOutlined, DeleteOutlined, SearchOutlined, ReloadOutlined,
  CheckCircleOutlined, StopOutlined, AppstoreOutlined, TableOutlined,
  ImportOutlined, ExportOutlined, BgColorsOutlined, ArrowLeftOutlined,
  EditOutlined as EditModeIcon, EyeOutlined,
} from '@ant-design/icons';
import type { Floor, WorkstationOps } from '@/types';
import { floorApi, buildingApi, wallApi, doorApi, workstationApi, floorPlanTextApi } from '@/lib/opsApi';
import { useTableManager } from '@/hooks/useTableManager';
import { usePagination } from '@/hooks/usePagination';
import { handleApiError, handleSuccess } from '@/utils/errorHandler';
import { createStatusColumn, createDateTimeColumn } from '@/utils/tableHelpers';
import ActionButtons from '@/components/shared/ActionButtons';
import ExcelImport from '@/components/shared/ExcelImport';
import ExcelExport from '@/components/shared/ExcelExport';
import { CADFloorPlanEditor } from '@/components/cad-editor';
import type { FloorPlanData, Wall, Door } from '@/components/cad-editor/types';
import type { WorkstationNode } from '@/components/shared/FloorPlanEditor.types';

const { Option } = Select;
const { TextArea } = Input;

type ViewMode = 'table' | 'card';
type PageMode = 'list' | 'editor';

interface BuildingOption {
  id: string;
  code: string;
  name: string;
}

const FloorManagement: FC = () => {
  const [pageMode, setPageMode] = useState<PageMode>('list');
  const [currentFloor, setCurrentFloor] = useState<Floor | null>(null);
  const [floorPlanLoading, setFloorPlanLoading] = useState(false);
  const [floorPlanData, setFloorPlanData] = useState<FloorPlanData | null>(null);
  const [isEditMode, setIsEditMode] = useState(false);

  const [statistics, setStatistics] = useState({ total: 0, active: 0, inactive: 0 });
  const [viewMode, setViewMode] = useState<ViewMode>('table');
  const [importVisible, setImportVisible] = useState(false);
  const [exportVisible, setExportVisible] = useState(false);
  const [buildingOptions, setBuildingOptions] = useState<BuildingOption[]>([]);

  // 使用全局分页 hook
  const { paginationProps, setCurrent, setPageSize, setTotal } = usePagination();

  const {
    loading, data: floors, total, selectedRowKeys,
    searchForm, editForm: floorForm, editModalVisible: modalVisible,
    editingItem: editingFloor, setSelectedRowKeys, setEditModalVisible: setModalVisible,
    setEditingItem: setEditingFloor, handleSearch,
    handleReset, handleAdd, handleEdit, handleModalClose, loadData: loadFloors, resetSelection,
  } = useTableManager<Floor>(
    async (params) => {
      const result = await floorApi.list(params);
      return { list: result.data?.list || [], total: result.data?.total || 0 };
    },
    {
      externalPagination: {
        current: paginationProps.current,
        pageSize: paginationProps.pageSize,
        setCurrent,
        setPageSize,
        setTotal,
      },
    }
  );

  const loadStatistics = useCallback(async () => {
    try {
      const result = await floorApi.list({ current: 1, pageSize: 1000 });
      const allFloors = result.data?.list || [];
      setStatistics({
        total: allFloors.length,
        active: allFloors.filter((f: Floor) => f.status === 0).length,
        inactive: allFloors.filter((f: Floor) => f.status === 1).length,
      });
    } catch (error) { handleApiError(error, '加载统计数据', false); }
  }, []);

  const loadBuildingOptions = useCallback(async () => {
    try {
      const result = await buildingApi.list({ current: 1, pageSize: 1000 });
      const buildings = result.data?.list || [];
      setBuildingOptions(buildings.map((b: any) => ({ id: b.id, code: b.code, name: b.name })));
    } catch (error) { handleApiError(error, '加载楼宇选项', false); }
  }, []);

  useEffect(() => {
    // 使用 setTimeout 避免同步 setState
    setTimeout(() => {
      loadFloors();
      loadStatistics();
      loadBuildingOptions();
    }, 0);
  }, [paginationProps.current, paginationProps.pageSize]);

  const handleSave = async () => {
    try {
      const values = await floorForm.validateFields();
      if (editingFloor) {
        await floorApi.update(editingFloor.id, values);
        handleSuccess('更新');
      } else {
        await floorApi.create(values);
        handleSuccess('创建');
      }
      handleModalClose();
      loadFloors();
      loadStatistics();
    } catch (error: any) { if (error.errorFields) return; handleApiError(error, '操作'); }
  };

  const handleDelete = async (id: string) => {
    try { await floorApi.delete(id); handleSuccess('删除'); loadFloors(); loadStatistics(); }
    catch (error) { handleApiError(error, '删除'); }
  };

  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) return;
    try { await floorApi.batch('delete', { ids: selectedRowKeys }); handleSuccess('批量删除'); resetSelection(); loadFloors(); loadStatistics(); }
    catch (error) { handleApiError(error, '批量删除'); }
  };

  const openModal = (record?: Floor) => {
    if (record) { handleEdit(record); floorForm.setFieldsValue(record); }
    else { handleAdd(); floorForm.setFieldsValue({ status: 0 }); }
  };

  const handleImportSuccess = () => { loadFloors(); loadStatistics(); setImportVisible(false); };

  const handleRefresh = () => { loadFloors(); loadStatistics(); };

  // 打开平面图编辑器
  const handleEditFloorPlan = async (floor: Floor) => {
    setCurrentFloor(floor);
    setPageMode('editor');
    setIsEditMode(false);
    await loadFloorPlanData(floor.id);
  };

  // 加载平面图数据
  const loadFloorPlanData = async (floorId: string) => {
    setFloorPlanLoading(true);
    try {
      const [wallsResult, doorsResult, workstationsResult] = await Promise.all([
        wallApi.list({ floorId, current: 1, pageSize: 1000 }),
        doorApi.list({ floorId, current: 1, pageSize: 1000 }),
        workstationApi.list({ floorId, current: 1, pageSize: 1000 }),
      ]);

      const walls = wallsResult.data?.list || [];
      const doors = doorsResult.data?.list || [];
      const workstations = workstationsResult.data?.list || [];

      // 尝试加载文本元素（如果后端还没重启，则使用空数组）
      let texts: any[] = [];
      try {
        const textsResult = await floorPlanTextApi.list({ floorId, current: 1, pageSize: 1000 });
        texts = textsResult.data?.list || [];
      } catch (textError) {
        console.warn('文本元素API不可用，可能需要重启后端服务:', textError);
        texts = [];
      }

      const parsedWalls: Wall[] = walls.map((wall: any) => ({
        ...wall,
        points: typeof wall.points === 'string' ? JSON.parse(wall.points) : wall.points,
      }));

      const parsedDoors: Door[] = doors.map((door: any) => ({
        ...door,
        position: typeof door.position === 'string' ? JSON.parse(door.position) : door.position,
      }));

      const parsedTexts = texts.map((text: any) => ({
        ...text,
        position: typeof text.position === 'string' ? JSON.parse(text.position) : text.position,
      }));

      // 处理工位数据：自动排列没有位置的工位
      const processedWorkstations = processWorkstations(workstations);

      setFloorPlanData({
        floorId,
        floorName: currentFloor?.name || `${currentFloor?.floorNo}层`,
        width: 2000,
        height: 2000,
        walls: parsedWalls,
        doors: parsedDoors,
        workstations: processedWorkstations,
        texts: parsedTexts,
        gridSize: 20,
        showGrid: true,
        snapToGrid: true,
      });
    } catch (error) {
      console.error('加载平面图失败:', error);
      message.error('加载平面图失败');
      setFloorPlanData({
        floorId,
        floorName: currentFloor?.name || `${currentFloor?.floorNo}层`,
        width: 2000,
        height: 2000,
        walls: [],
        doors: [],
        workstations: [],
        texts: [],
        gridSize: 20,
        showGrid: true,
        snapToGrid: true,
      });
    } finally {
      setFloorPlanLoading(false);
    }
  };

  // 处理工位数据：为没有位置的工位自动分配位置（10个一行）
  const processWorkstations = (workstations: WorkstationOps[]): WorkstationNode[] => {
    const DEFAULT_WORKSTATION_WIDTH = 160;  // 默认工位宽度（毫米）
    const DEFAULT_WORKSTATION_DEPTH = 70;  // 默认工位深度（毫米）
    const GAP = 20;                            // 间距
    const ITEMS_PER_ROW = 10;                  // 每行10个
    const START_X = 100;                        // 起始X坐标
    const START_Y = 100;                        // 起始Y坐标

    // 分离有位置和无位置的工位
    const withPosition: WorkstationNode[] = [];
    const withoutPosition: WorkstationOps[] = [];

    workstations.forEach((ws) => {
      if (ws.positionX !== undefined && ws.positionY !== undefined) {
        // 有位置的工位：使用数据库中的实际值，如果没有尺寸字段则使用默认值
        withPosition.push({
          id: ws.id,
          code: ws.name,
          name: ws.name,
          x: ws.positionX,
          y: ws.positionY,
          width: (ws.width || DEFAULT_WORKSTATION_WIDTH) / 10, // 转换为像素（毫米/10）
          height: (ws.depth || DEFAULT_WORKSTATION_DEPTH) / 10, // 转换为像素（毫米/10）
          rotation: ws.rotation || 0,
          type: ws.deskType || 0, // 0: 一字型, 1: L型
          status: ws.status,
        });
      } else {
        withoutPosition.push(ws);
      }
    });

    // 为没有位置的工位自动分配位置
    withoutPosition.forEach((ws, index) => {
      const row = Math.floor(index / ITEMS_PER_ROW);
      const col = index % ITEMS_PER_ROW;

      const wsWidth = (ws.width || DEFAULT_WORKSTATION_WIDTH) / 10;
      const wsDepth = (ws.depth || DEFAULT_WORKSTATION_DEPTH) / 10;

      withPosition.push({
        id: ws.id,
        code: ws.name,
        name: ws.name,
        x: START_X + col * (wsWidth + GAP),
        y: START_Y + row * (wsDepth + GAP),
        width: wsWidth,
        height: wsDepth,
        rotation: ws.rotation || 0,
        type: ws.deskType || 0,
        status: ws.status,
      });
    });

    return withPosition;
  };

  // 保存平面图
  const handleSaveFloorPlan = async (data: FloorPlanData) => {
    if (!currentFloor) return;
    try {
      const floorId = currentFloor.id;

      // 保存墙体
      for (const wall of data.walls) {
        const isNew = wall.id.startsWith('wall_');
        // 新建时不包含 id，让后端生成 UUID
        const { id: _wallId, ...wallWithoutId } = wall;
        const wallData = {
          ...(isNew ? wallWithoutId : wall),
          floorId,
          points: JSON.stringify(wall.points),
          type: wall.type as 'straight' | 'curved' | 'l_shaped' | 'polyline',
        };

        if (isNew) {
          await wallApi.create(wallData);
        } else {
          await wallApi.update(wall.id, wallData);
        }
      }

      // 保存门
      for (const door of data.doors) {
        const isNew = door.id.startsWith('door_');
        // 新建时不包含 id，让后端生成 UUID
        const { id: _doorId, ...doorWithoutId } = door;
        const doorData = {
          ...(isNew ? doorWithoutId : door),
          floorId,
          position: JSON.stringify(door.position),
        };

        if (isNew) {
          await doorApi.create(doorData);
        } else {
          await doorApi.update(door.id, doorData);
        }
      }

      // 保存文本元素（如果后端还没重启，则跳过）
      if (data.texts && data.texts.length > 0) {
        try {
          for (const text of data.texts) {
            const isNew = text.id.startsWith('text_');
            // 新建时不包含 id，让后端生成 UUID
            const { id: _textId, ...textWithoutId } = text;
            const textData = {
              ...(isNew ? textWithoutId : text),
              floorId,
              position: JSON.stringify(text.position),
            };

            if (isNew) {
              await floorPlanTextApi.create(textData);
            } else {
              await floorPlanTextApi.update(text.id, textData);
            }
          }
        } catch (textError) {
          console.warn('保存文本元素失败，可能需要重启后端服务:', textError);
          // 不阻止整个保存流程
        }
      }

      // 保存工位数据（位置、尺寸、桌型）
      const workstationUpdates = data.workstations.map((ws) => ({
        id: ws.id,
        positionX: Math.round(ws.x),
        positionY: Math.round(ws.y),
        rotation: ws.rotation || 0,
        width: Math.round((ws.width || 160) * 10),  // 转换为毫米（像素*10）
        depth: Math.round((ws.height || 70) * 10),   // 转换为毫米（像素*10）
        deskType: ws.type || 0,  // 0: 一字型, 1: L型
      }));

      if (workstationUpdates.length > 0) {
        await workstationApi.updatePositions(workstationUpdates as any);
      }

      message.success('保存成功');
      await loadFloorPlanData(floorId);
    } catch (error) {
      console.error('保存失败:', error);
      message.error('保存失败');
    }
  };

  // 返回列表视图
  const handleBackToList = () => {
    setPageMode('list');
    setCurrentFloor(null);
    setFloorPlanData(null);
    setIsEditMode(false);
  };

  const columns: ColumnsType<Floor> = [
    { title: '楼层名称', dataIndex: 'name', key: 'name', width: 150 },
    { title: '楼层号', dataIndex: 'floorNo', key: 'floorNo', width: 100 },
    {
      title: '所属楼宇',
      dataIndex: 'buildingName',
      key: 'buildingName',
      width: 150,
      render: (_, record) => record.buildingName || record.buildingCode,
    },
    { title: '面积(m²)', dataIndex: 'area', key: 'area', width: 120, render: (v) => v || '-' },
    createStatusColumn('status', { width: 100 }),
    { title: '描述', dataIndex: 'description', key: 'description', ellipsis: true },
    createDateTimeColumn('createdAt', { width: 180 }),
    {
      title: '操作', key: 'action',
      render: (_, record) => {
        const actions = [
          { key: 'edit', label: '编辑', icon: <EditOutlined />, onClick: () => openModal(record) },
          { key: 'floorPlan', label: '编辑平面图', icon: <BgColorsOutlined />, onClick: () => handleEditFloorPlan(record) },
          {
            key: 'delete', label: '删除', icon: <DeleteOutlined />, danger: true,
            render: () => (
              <Popconfirm title="确定要删除这个楼层吗？" onConfirm={() => handleDelete(record.id)} okText="确定" cancelText="取消">
                <Button type="link" danger icon={<DeleteOutlined />} size="small">删除</Button>
              </Popconfirm>
            ),
          },
        ];
        return <ActionButtons actions={actions} />;
      },
    },
  ];

  const renderCardView = () => {
    if (floors.length === 0) return <div style={{ textAlign: 'center', padding: '40px', color: '#999' }}>暂无数据</div>;
    return (
      <Row gutter={[16, 16]}>
        {floors.map((floor) => (
          <Col key={floor.id} xs={24} sm={12} md={8} lg={6}>
            <Card
              hoverable
              actions={[
                <EditOutlined key="edit" onClick={() => openModal(floor)} />,
                <BgColorsOutlined key="floorPlan" onClick={() => handleEditFloorPlan(floor)} title="编辑平面图" />,
                <Popconfirm key="delete" title="确定要删除这个楼层吗？" onConfirm={() => handleDelete(floor.id)} okText="确定" cancelText="取消">
                  <DeleteOutlined style={{ color: '#ff4d4f' }} />
                </Popconfirm>,
              ]}
            >
              <Card.Meta
                title={
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <span>{floor.name || `${floor.floorNo}层`}</span>
                    <Tag color={floor.status === 0 ? 'success' : 'error'}>{floor.status === 0 ? '正常' : '停用'}</Tag>
                  </div>
                }
                description={
                  <div>
                    <div><strong>楼层号：</strong>{floor.floorNo}</div>
                    <div><strong>楼宇：</strong>{floor.buildingName || floor.buildingCode}</div>
                    {floor.area && <div><strong>面积：</strong>{floor.area}m²</div>}
                  </div>
                }
              />
            </Card>
          </Col>
        ))}
      </Row>
    );
  };

  // 渲染平面图编辑器视图
  const renderFloorPlanEditor = () => {
    if (!currentFloor || !floorPlanData) {
      return (
        <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '60vh', flexDirection: 'column', gap: 16 }}>
          <Spin size="large" />
          <span style={{ color: '#999' }}>加载中...</span>
        </div>
      );
    }

    return (
      <div style={{ height: 'calc(100vh - 180px)', display: 'flex', flexDirection: 'column' }}>
        <Card
          title={`${currentFloor.name || `${currentFloor.floorNo}层`} - 平面图编辑`}
          extra={
            <Space>
              <Button
                icon={isEditMode ? <EyeOutlined /> : <EditModeIcon />}
                onClick={() => setIsEditMode(!isEditMode)}
              >
                {isEditMode ? '预览模式' : '编辑模式'}
              </Button>
              <Button icon={<ReloadOutlined />} onClick={() => loadFloorPlanData(currentFloor.id)}>
                刷新
              </Button>
              <Button icon={<ArrowLeftOutlined />} onClick={handleBackToList}>
                返回列表
              </Button>
            </Space>
          }
          style={{ flex: 1, display: 'flex', flexDirection: 'column' }}
          styles={{ body: { flex: 1, padding: 0, overflow: 'hidden' } }}
        >
          <CADFloorPlanEditor
            floorId={currentFloor.id}
            floorName={currentFloor.name || `${currentFloor.floorNo}层`}
            walls={floorPlanData.walls}
            doors={floorPlanData.doors}
            workstations={floorPlanData.workstations}
            texts={floorPlanData.texts || []}
            onSave={handleSaveFloorPlan}
            readOnly={!isEditMode}
          />
        </Card>
      </div>
    );
  };

  // 渲染列表视图
  const renderListView = () => (
    <div>
      {total > 10 && (
        <Row gutter={16} style={{ marginBottom: 16 }}>
          <Col span={8}><Card><Statistic title="总楼层数" value={statistics.total} prefix={<CheckCircleOutlined />} /></Card></Col>
          <Col span={8}><Card><Statistic title="正常楼层" value={statistics.active} styles={{ content: { color: '#3f8600' } }} prefix={<CheckCircleOutlined />} /></Card></Col>
          <Col span={8}><Card><Statistic title="停用楼层" value={statistics.inactive} styles={{ content: { color: '#cf1322' } }} prefix={<StopOutlined />} /></Card></Col>
        </Row>
      )}
      <Card style={{ marginBottom: 16 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', flexWrap: 'wrap', gap: '16px' }}>
          <Form form={searchForm} layout="inline" style={{ flex: 1, minWidth: 0 }}>
            <Form.Item name="buildingCode" label="所属楼宇">
              <Select placeholder="请选择楼宇" allowClear style={{ width: 150 }}>
                {buildingOptions.map(b => <Option key={b.code} value={b.code}>{b.name}</Option>)}
              </Select>
            </Form.Item>
            <Form.Item name="floorNo" label="楼层号"><Input placeholder="请输入楼层号" allowClear style={{ width: 120 }} /></Form.Item>
            <Form.Item name="name" label="楼层名称"><Input placeholder="请输入楼层名称" allowClear style={{ width: 150 }} /></Form.Item>
            <Form.Item name="status" label="状态">
              <Select placeholder="请选择状态" allowClear style={{ width: 120 }}><Option value={0}>正常</Option><Option value={1}>停用</Option></Select>
            </Form.Item>
            <Form.Item>
              <Space><Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>搜索</Button><Button onClick={handleReset}>重置</Button><Button icon={<ReloadOutlined />} onClick={handleRefresh}>刷新</Button></Space>
            </Form.Item>
          </Form>
          <Space>
            <Radio.Group value={viewMode} onChange={(e) => setViewMode(e.target.value)} buttonStyle="solid">
              <Radio.Button value="table"><TableOutlined /> 表格</Radio.Button><Radio.Button value="card"><AppstoreOutlined /> 卡片</Radio.Button>
            </Radio.Group>
            <Button icon={<ImportOutlined />} onClick={() => setImportVisible(true)}>导入</Button>
            <Button icon={<ExportOutlined />} onClick={() => setExportVisible(true)}>导出</Button>
            {selectedRowKeys.length > 0 && <Button danger icon={<DeleteOutlined />} onClick={handleBatchDelete}>批量删除 ({selectedRowKeys.length})</Button>}
            <Button type="primary" icon={<PlusOutlined />} onClick={() => openModal()}>新增楼层</Button>
          </Space>
        </div>
        {selectedRowKeys.length > 0 && <Alert title={<span>已选择 <strong>{selectedRowKeys.length}</strong> 个楼层，<Button type="link" size="small" onClick={() => setSelectedRowKeys([])} style={{ padding: 0 }}>取消选择</Button></span>} type="info" showIcon style={{ marginTop: 12 }} />}
      </Card>
      <Card>
        {viewMode === 'table' ? (
          <Table rowSelection={{ selectedRowKeys, onChange: setSelectedRowKeys }} columns={columns} dataSource={floors} loading={loading} rowKey="id" pagination={paginationProps} />
        ) : renderCardView()}
      </Card>
      <Modal title={editingFloor ? '编辑楼层' : '新增楼层'} open={modalVisible} onOk={handleSave} onCancel={() => { setModalVisible(false); floorForm.resetFields(); setEditingFloor(null); }} width={600}>
        <Form form={floorForm} layout="horizontal" labelCol={{ span: 5 }} wrapperCol={{ span: 19 }}>
          <Form.Item name="buildingId" label="所属楼宇" rules={[{ required: true, message: '请选择所属楼宇' }]}>
            <Select placeholder="请选择所属楼宇" disabled={!!editingFloor}>
              {buildingOptions.map(b => <Option key={b.id} value={b.id}>{b.name}</Option>)}
            </Select>
          </Form.Item>
          <Form.Item name="floorNo" label="楼层号" rules={[{ required: true, message: '请输入楼层号' }]}><Input placeholder="请输入楼层号，如：1F" disabled={!!editingFloor} /></Form.Item>
          <Form.Item name="name" label="楼层名称"><Input placeholder="请输入楼层名称" /></Form.Item>
          <Form.Item name="area" label="面积"><InputNumber min={0} placeholder="请输入面积(m²)" style={{ width: '100%' }} /></Form.Item>
          <Form.Item name="status" label="状态" rules={[{ required: true, message: '请选择状态' }]}><Select placeholder="请选择状态"><Option value={0}>正常</Option><Option value={1}>停用</Option></Select></Form.Item>
          <Form.Item name="description" label="描述"><TextArea rows={3} placeholder="请输入描述" /></Form.Item>
        </Form>
      </Modal>
      <ExcelImport entityType="floor" entityName="楼层" visible={importVisible} onClose={() => setImportVisible(false)} onImportSuccess={handleImportSuccess} />
      <ExcelExport entityType="floor" entityName="楼层" visible={exportVisible} onClose={() => setExportVisible(false)} />
    </div>
  );

  return (
    <div>
      {pageMode === 'editor' ? renderFloorPlanEditor() : renderListView()}
    </div>
  );
};

export default FloorManagement;
