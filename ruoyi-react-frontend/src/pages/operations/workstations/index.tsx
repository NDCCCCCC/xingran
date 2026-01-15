/**
 * 工位管理页面
 */

import { useState, useEffect, useCallback } from 'react';
import type { FC } from 'react';
import {
  Table, Button, Space, Modal, Form, Input, InputNumber, Select, Popconfirm,
  Tag, Card, Row, Col, Statistic, Alert, Radio, TreeSelect,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  PlusOutlined, EditOutlined, DeleteOutlined, SearchOutlined, ReloadOutlined,
  CheckCircleOutlined, StopOutlined, AppstoreOutlined, TableOutlined,
  ImportOutlined, ExportOutlined, DesktopOutlined, BgColorsOutlined, CloseOutlined,
} from '@ant-design/icons';
import type { WorkstationOps } from '@/types';
import { workstationApi, floorApi } from '@/lib/opsApi';
import { post } from '@/lib/api';
import { useTableManager } from '@/hooks/useTableManager';
import { handleApiError, handleSuccess } from '@/utils/errorHandler';
import { createStatusColumn, createDateTimeColumn } from '@/utils/tableHelpers';
import ActionButtons from '@/components/shared/ActionButtons';
import ExcelImport from '@/components/shared/ExcelImport';
import ExcelExport from '@/components/shared/ExcelExport';
import FloorPlanEditor, { type WorkstationNode } from '@/components/shared/FloorPlanEditor';

const { Option } = Select;
const { TextArea } = Input;

type ViewMode = 'table' | 'card' | 'floorplan';

interface FloorOption {
  code: string;
  name: string;
}

interface UserOption {
  id: string;
  name: string;
}

interface DeptTreeNode {
  title: string;
  value: string;
  key: string;
  children?: DeptTreeNode[];
}

const WorkstationManagement: FC = () => {
  const [statistics, setStatistics] = useState({ total: 0, available: 0, occupied: 0, maintain: 0 });
  const [viewMode, setViewMode] = useState<ViewMode>('table');
  const [importVisible, setImportVisible] = useState(false);
  const [exportVisible, setExportVisible] = useState(false);
  const [floorOptions, setFloorOptions] = useState<FloorOption[]>([]);
  const [deptTreeData, setDeptTreeData] = useState<DeptTreeNode[]>([]);
  const [userOptions, setUserOptions] = useState<UserOption[]>([]);
  // 平面图视图相关状态
  const [selectedFloorForPlan, setSelectedFloorForPlan] = useState<string>('');
  const [floorPlanWorkstations, setFloorPlanWorkstations] = useState<WorkstationNode[]>([]);

  const {
    loading, data: workstations, total, current, pageSize, selectedRowKeys,
    searchForm, editForm: workstationForm, editModalVisible: modalVisible,
    editingItem: editingWorkstation, setSelectedRowKeys, setEditModalVisible: setModalVisible,
    setEditingItem: setEditingWorkstation, setCurrent, setPageSize, handleSearch,
    handleReset, handleAdd, handleEdit, handleModalClose, loadData: loadWorkstations, resetSelection,
  } = useTableManager<WorkstationOps>(
    async (params) => {
      const result = await workstationApi.list(params);
      return { list: result.data?.list || [], total: result.data?.total || 0 };
    },
    { pageSize: 10 }
  );

  const loadStatistics = useCallback(async () => {
    try {
      const result = await workstationApi.list({ current: 1, pageSize: 1000 });
      const allWorkstations = result.data?.list || [];
      setStatistics({
        total: allWorkstations.length,
        available: allWorkstations.filter((w: WorkstationOps) => w.status === 0).length,
        occupied: allWorkstations.filter((w: WorkstationOps) => w.status === 1).length,
        maintain: allWorkstations.filter((w: WorkstationOps) => w.status === 2).length,
      });
    } catch (error) { handleApiError(error, '加载统计数据', false); }
  }, []);

  const loadFloorOptions = useCallback(async () => {
    try {
      const result = await floorApi.list({ current: 1, pageSize: 1000 });
      const floors = result.data?.list || [];
      setFloorOptions(floors.map((f: any) => ({ code: f.id, name: `${f.buildingName || f.buildingCode} - ${f.name || f.floorNo}` })));
    } catch (error) { handleApiError(error, '加载楼层选项', false); }
  }, []);

  const loadDeptOptions = useCallback(async () => {
    try {
      const result = await post('/system/departments/list', {});
      // 构建树形结构数据
      const buildTreeData = (depts: any[]): DeptTreeNode[] => {
        return depts.map((dept) => ({
          title: dept.deptName || dept.name,
          value: dept.id,
          key: dept.id,
          children: dept.children && dept.children.length > 0 ? buildTreeData(dept.children) : undefined,
        }));
      };
      const deptList = Array.isArray(result.data?.list) ? result.data.list : [];
      setDeptTreeData(buildTreeData(deptList));
    } catch (error) { handleApiError(error, '加载部门选项', false); }
  }, []);

  const loadUserOptions = useCallback(async (deptId?: string) => {
    try {
      const params: any = { current: 1, pageSize: 1000 };
      if (deptId) {
        params.deptId = deptId;
      }
      const result = await post('/system/users/list', params);
      const users = result.data?.list || [];
      setUserOptions(users.map((u: any) => ({ id: u.id, name: u.username })));
    } catch (error) { handleApiError(error, '加载用户选项', false); }
  }, []);

  useEffect(() => { loadWorkstations(); loadStatistics(); loadFloorOptions(); loadDeptOptions(); loadUserOptions(); }, []);

  const handleSave = async () => {
    try {
      const values = await workstationForm.validateFields();
      if (editingWorkstation) {
        await workstationApi.update(editingWorkstation.id, values);
        handleSuccess('更新');
      } else {
        await workstationApi.create(values);
        handleSuccess('创建');
      }
      handleModalClose();
      loadWorkstations();
      loadStatistics();
    } catch (error: any) { if (error.errorFields) return; handleApiError(error, '操作'); }
  };

  const handleDelete = async (id: string) => {
    try { await workstationApi.delete(id); handleSuccess('删除'); loadWorkstations(); loadStatistics(); }
    catch (error) { handleApiError(error, '删除'); }
  };

  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) return;
    try { await workstationApi.batch('delete', { ids: selectedRowKeys }); handleSuccess('批量删除'); resetSelection(); loadWorkstations(); loadStatistics(); }
    catch (error) { handleApiError(error, '批量删除'); }
  };

  const openModal = (record?: WorkstationOps) => {
    if (record) { handleEdit(record); workstationForm.setFieldsValue(record); }
    else { handleAdd(); workstationForm.setFieldsValue({ status: 0, type: 0 }); }
  };

  const handleImportSuccess = () => { loadWorkstations(); loadStatistics(); setImportVisible(false); };

  // 平面图视图 - 加载指定楼层的工位
  const loadFloorPlanWorkstations = useCallback(async (floorCode: string) => {
    if (!floorCode) {
      setFloorPlanWorkstations([]);
      return;
    }
    try {
      const result = await workstationApi.list({ floorCode, current: 1, pageSize: 1000 });
      const list = result.data?.list || [];
      // 转换为 WorkstationNode 格式
      setFloorPlanWorkstations(list.map((w: WorkstationOps) => ({
        id: w.id,
        code: w.name,
        name: w.name,
        x: w.positionX || 0,
        y: w.positionY || 0,
        width: 80,
        height: 60,
        status: w.status,
        type: w.type,
        rotation: w.rotation || 0,
      })));
    } catch (error) {
      handleApiError(error, '加载平面图数据', false);
      setFloorPlanWorkstations([]);
    }
  }, []);

  // 平面图视图 - 更新工位位置和旋转角度
  const handlePositionUpdate = useCallback(async (items: {id: string; positionX: number; positionY: number; rotation?: number}[]) => {
    try {
      await workstationApi.updatePositions(items);

      // 更新本地板图数据
      setFloorPlanWorkstations(prev =>
        prev.map(ws => {
          const updatedItem = items.find(item => item.id === ws.id);
          if (updatedItem) {
            return {
              ...ws,
              x: updatedItem.positionX,
              y: updatedItem.positionY,
              ...(updatedItem.rotation !== undefined && { rotation: updatedItem.rotation })
            };
          }
          return ws;
        })
      );
    } catch (error) {
      handleApiError(error, '更新位置');
      throw error;
    }
  }, []);

  // 平面图视图 - 编辑工位
  const handleFloorPlanEdit = useCallback((workstation: WorkstationNode) => {
    // 从workstations中查找完整数据
    const fullWorkstation = workstations.find(w => w.id === workstation.id);
    if (fullWorkstation) {
      openModal(fullWorkstation);
    }
  }, [workstations]);

  // 平面图视图 - 楼层选择变化
  const handleFloorChangeForPlan = (floorId: string) => {
    setSelectedFloorForPlan(floorId);
    loadFloorPlanWorkstations(floorId);
  };

  // 当切换到平面图视图时，默认加载第一个楼层的工位
  useEffect(() => {
    if (viewMode === 'floorplan' && !selectedFloorForPlan && floorOptions.length > 0) {
      const firstFloor = floorOptions[0].code;
      setSelectedFloorForPlan(firstFloor);
      loadFloorPlanWorkstations(firstFloor);
    }
  }, [viewMode, selectedFloorForPlan, floorOptions, loadFloorPlanWorkstations]);

  const getWorkstationTypeText = (type: number) => {
    const types = { 0: '固定工位', 1: '灵活工位', 2: '管理工位' };
    return types[type as keyof typeof types] || '-';
  };

  const columns: ColumnsType<WorkstationOps> = [
    { title: '工位名称', dataIndex: 'name', key: 'name', width: 150 },
    {
      title: '所属楼层',
      dataIndex: 'floorName',
      key: 'floorName',
      width: 150,
      render: (_, record) => record.floorName || record.floorId,
    },
    {
      title: '工位类型',
      dataIndex: 'type',
      key: 'type',
      width: 100,
      render: (type) => <Tag>{getWorkstationTypeText(type)}</Tag>,
    },
    createStatusColumn('status', { width: 100 }),
    { title: '描述', dataIndex: 'description', key: 'description', ellipsis: true },
    createDateTimeColumn('createdAt', { width: 180 }),
    {
      title: '操作', key: 'action',
      render: (_, record) => {
        const actions = [
          { key: 'edit', label: '编辑', icon: <EditOutlined />, onClick: () => openModal(record) },
          {
            key: 'delete', label: '删除', icon: <DeleteOutlined />, danger: true,
            render: () => (
              <Popconfirm title="确定要删除这个工位吗？" onConfirm={() => handleDelete(record.id)} okText="确定" cancelText="取消">
                <Button type="link" danger icon={<DeleteOutlined />} size="small">删除</Button>
              </Popconfirm>
            ),
          },
        ];
        return <ActionButtons actions={actions} />;
      },
    },
  ];

  const getWorkstationStatusText = (status: number) => {
    const statuses = { 0: '空闲', 1: '占用', 2: '维护' };
    return statuses[status as keyof typeof statuses] || '-';
  };

  const getWorkstationStatusColor = (status: number) => {
    const colors = { 0: 'success', 1: 'error', 2: 'warning' };
    return colors[status as keyof typeof colors] || 'default';
  };

  const renderCardView = () => {
    if (workstations.length === 0) return <div style={{ textAlign: 'center', padding: '40px', color: '#999' }}>暂无数据</div>;
    return (
      <Row gutter={[16, 16]}>
        {workstations.map((workstation) => (
          <Col key={workstation.id} xs={24} sm={12} md={8} lg={6}>
            <Card
              hoverable
              actions={[
                <EditOutlined key="edit" onClick={() => openModal(workstation)} />,
                <Popconfirm key="delete" title="确定要删除这个工位吗？" onConfirm={() => handleDelete(workstation.id)} okText="确定" cancelText="取消">
                  <DeleteOutlined style={{ color: '#ff4d4f' }} />
                </Popconfirm>,
              ]}
            >
              <Card.Meta
                title={
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <span>{workstation.name}</span>
                    <Tag color={getWorkstationStatusColor(workstation.status)}>{getWorkstationStatusText(workstation.status)}</Tag>
                  </div>
                }
                description={
                  <div>
                    <div><strong>楼层：</strong>{workstation.floorName || workstation.floorId}</div>
                    <div><strong>类型：</strong>{getWorkstationTypeText(workstation.type)}</div>
                  </div>
                }
              />
            </Card>
          </Col>
        ))}
      </Row>
    );
  };

  // 平面图视图渲染函数
  const renderFloorPlanView = () => {
    return (
      <div style={{ display: 'flex', gap: 16, height: 'calc(100vh - 280px)', minHeight: 600 }}>
        {/* 左侧：平面图编辑器 */}
        <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
          {/* 楼层选择器 */}
          <div style={{ marginBottom: 12, padding: '12px', background: '#f5f5f5', borderRadius: '4px', flexShrink: 0 }}>
            <Space wrap>
              <span>选择楼层：</span>
              <Select
                style={{ width: 200 }}
                placeholder="请选择要查看的楼层"
                value={selectedFloorForPlan}
                onChange={handleFloorChangeForPlan}
                allowClear
              >
                {floorOptions.map(f => (
                  <Option key={f.code} value={f.code}>{f.name}</Option>
                ))}
              </Select>
              {selectedFloorForPlan && (
                <Tag color="blue">当前楼层：{floorOptions.find(f => f.code === selectedFloorForPlan)?.name}</Tag>
              )}
              <Tag color="cyan">工位数：{floorPlanWorkstations.length}</Tag>
              <Button
                type="default"
                icon={<CloseOutlined />}
                onClick={() => setViewMode('table')}
                size="middle"
              >
                关闭平面图
              </Button>
            </Space>
          </div>
          {/* 平面图编辑器容器 */}
          <div style={{ flex: 1, minHeight: 0 }}>
            {!selectedFloorForPlan ? (
              <div style={{ textAlign: 'center', padding: '60px', color: '#999', height: '100%', display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center' }}>
                <BgColorsOutlined style={{ fontSize: '48px', marginBottom: '16px' }} />
                <div>请选择要查看的楼层平面图</div>
              </div>
            ) : floorPlanWorkstations.length === 0 ? (
              <div style={{ textAlign: 'center', padding: '60px', color: '#999', height: '100%', display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center' }}>
                <DesktopOutlined style={{ fontSize: '48px', marginBottom: '16px' }} />
                <div>该楼层暂无工位数据</div>
              </div>
            ) : (
              <FloorPlanEditor
                key={selectedFloorForPlan}
                floorId={selectedFloorForPlan}
                workstations={floorPlanWorkstations}
                onUpdatePosition={handlePositionUpdate}
                onEdit={handleFloorPlanEdit}
              />
            )}
          </div>
        </div>

        {/* 右侧：工位列表 */}
        <Card
          title="工位列表"
          style={{ width: 320, flexShrink: 0, overflow: 'hidden' }}
          bodyStyle={{ padding: 0, height: 'calc(100% - 57px)', overflow: 'auto' }}
        >
          <div style={{ padding: '8px' }}>
            {floorPlanWorkstations.length === 0 ? (
              <div style={{ textAlign: 'center', padding: '20px', color: '#999' }}>
                暂无工位数据
              </div>
            ) : (
              <Space direction="vertical" style={{ width: '100%' }} size={[8, 8]}>
                {floorPlanWorkstations.map(ws => {
                  const typeText = getWorkstationTypeText(ws.type);
                  const isSelected = selectedFloorForPlan && workstations.find(w => w.id === ws.id);
                  return (
                    <Card
                      key={ws.id}
                      size="small"
                      hoverable
                      style={{
                        cursor: 'pointer',
                        border: isSelected ? '2px solid #1890ff' : '1px solid #f0f0f0',
                      }}
                      onClick={() => {
                        const fullWorkstation = workstations.find(w => w.id === ws.id);
                        if (fullWorkstation) openModal(fullWorkstation);
                      }}
                      bodyStyle={{ padding: '12px' }}
                    >
                      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
                        <span style={{ fontWeight: 'bold', fontSize: 14 }}>{ws.name}</span>
                        <Tag color={getWorkstationStatusColor(ws.status)} style={{ margin: 0 }}>
                          {getWorkstationStatusText(ws.status)}
                        </Tag>
                      </div>
                      <div style={{ fontSize: 12, color: '#666', marginBottom: 4 }}>
                        <div>类型：{typeText}</div>
                        <div>位置：({Math.round(ws.x)}, {Math.round(ws.y)})</div>
                      </div>
                    </Card>
                  );
                })}
              </Space>
            )}
          </div>
        </Card>
      </div>
    );
  };

  return (
    <div>
      {total > 10 && (
        <Row gutter={16} style={{ marginBottom: 16 }}>
          <Col span={6}><Card><Statistic title="总工位数" value={statistics.total} prefix={<DesktopOutlined />} /></Card></Col>
          <Col span={6}><Card><Statistic title="空闲" value={statistics.available} styles={{ content: { color: '#52c41a' } }} prefix={<CheckCircleOutlined />} /></Card></Col>
          <Col span={6}><Card><Statistic title="占用" value={statistics.occupied} styles={{ content: { color: '#ff4d4f' } }} prefix={<StopOutlined />} /></Card></Col>
          <Col span={6}><Card><Statistic title="维护" value={statistics.maintain} styles={{ content: { color: '#faad14' } }} valueStyle={{ color: '#faad14' }} /></Card></Col>
        </Row>
      )}
      <Card style={{ marginBottom: 16 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', flexWrap: 'wrap', gap: '16px' }}>
          <Form form={searchForm} layout="inline" style={{ flex: 1, minWidth: 0 }}>
            <Form.Item name="floorId" label="所属楼层">
              <Select placeholder="请选择楼层" allowClear style={{ width: 150 }}>
                {floorOptions.map(f => <Option key={f.code} value={f.code}>{f.name}</Option>)}
              </Select>
            </Form.Item>
            <Form.Item name="name" label="工位名称"><Input placeholder="请输入工位名称" allowClear style={{ width: 150 }} /></Form.Item>
            <Form.Item name="status" label="状态">
              <Select placeholder="请选择状态" allowClear style={{ width: 120 }}><Option value={0}>空闲</Option><Option value={1}>占用</Option><Option value={2}>维护</Option></Select>
            </Form.Item>
            <Form.Item>
              <Space><Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>搜索</Button><Button onClick={handleReset}>重置</Button><Button icon={<ReloadOutlined />} onClick={() => { loadWorkstations(); loadStatistics(); }}>刷新</Button></Space>
            </Form.Item>
          </Form>
          <Space>
            <Radio.Group value={viewMode} onChange={(e) => setViewMode(e.target.value)} buttonStyle="solid">
              <Radio.Button value="table"><TableOutlined /> 表格</Radio.Button><Radio.Button value="card"><AppstoreOutlined /> 卡片</Radio.Button><Radio.Button value="floorplan"><BgColorsOutlined /> 平面图</Radio.Button>
            </Radio.Group>
            <Button icon={<ImportOutlined />} onClick={() => setImportVisible(true)}>导入</Button>
            <Button icon={<ExportOutlined />} onClick={() => setExportVisible(true)}>导出</Button>
            {selectedRowKeys.length > 0 && <Button danger icon={<DeleteOutlined />} onClick={handleBatchDelete}>批量删除 ({selectedRowKeys.length})</Button>}
            <Button type="primary" icon={<PlusOutlined />} onClick={() => openModal()}>新增工位</Button>
          </Space>
        </div>
        {selectedRowKeys.length > 0 && <Alert message={<span>已选择 <strong>{selectedRowKeys.length}</strong> 个工位，<Button type="link" size="small" onClick={() => setSelectedRowKeys([])} style={{ padding: 0 }}>取消选择</Button></span>} type="info" showIcon style={{ marginTop: 12 }} />}
      </Card>
      <Card>
        {viewMode === 'table' ? (
          <Table rowSelection={{ selectedRowKeys, onChange: setSelectedRowKeys }} columns={columns} dataSource={workstations} loading={loading} rowKey="id" pagination={{ current, pageSize, total, showSizeChanger: true, showQuickJumper: true, showTotal: (total) => `共 ${total} 条`, onChange: (page, size) => { setCurrent(page); setPageSize(size || 10); loadWorkstations({ current: page, pageSize: size || 10 }); } }} />
        ) : viewMode === 'card' ? renderCardView() : renderFloorPlanView()}
      </Card>
      <Modal title={editingWorkstation ? '编辑工位' : '新增工位'} open={modalVisible} onOk={handleSave} onCancel={() => { setModalVisible(false); workstationForm.resetFields(); setEditingWorkstation(null); }} width={700}>
        <Form form={workstationForm} layout="horizontal" labelCol={{ span: 5 }} wrapperCol={{ span: 19 }}>
          <Form.Item name="floorId" label="所属楼层" rules={[{ required: true, message: '请选择所属楼层' }]}>
            <Select placeholder="请选择所属楼层">
              {floorOptions.map(f => <Option key={f.code} value={f.code}>{f.name}</Option>)}
            </Select>
          </Form.Item>
          <Form.Item name="name" label="工位名称" rules={[{ required: true, message: '请输入工位名称' }]}><Input placeholder="请输入工位名称" /></Form.Item>
          <Form.Item name="type" label="工位类型" rules={[{ required: true, message: '请选择工位类型' }]}>
            <Select placeholder="请选择工位类型"><Option value={0}>固定工位</Option><Option value={1}>灵活工位</Option><Option value={2}>管理工位</Option></Select>
          </Form.Item>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="deptId" label="所属部门">
                <TreeSelect
                  placeholder="请选择所属部门"
                  allowClear
                  treeData={deptTreeData}
                  showSearch
                  treeDefaultExpandAll
                  treeCheckable={false}
                  onChange={() => {
                    // 当部门变化时，清空用户选择并重新加载用户
                    workstationForm.setFieldsValue({ userId: undefined });
                    const deptId = workstationForm.getFieldValue('deptId');
                    loadUserOptions(deptId);
                  }}
                  style={{ width: '100%' }}
                />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="userId" label="所属用户">
                <Select placeholder="请先选择部门" allowClear>
                  {userOptions.map(u => <Option key={u.id} value={u.id}>{u.name}</Option>)}
                </Select>
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}><Form.Item name="positionX" label="X坐标"><InputNumber placeholder="X坐标" style={{ width: '100%' }} /></Form.Item></Col>
            <Col span={12}><Form.Item name="positionY" label="Y坐标"><InputNumber placeholder="Y坐标" style={{ width: '100%' }} /></Form.Item></Col>
          </Row>
          <Form.Item name="status" label="状态" rules={[{ required: true, message: '请选择状态' }]}><Select placeholder="请选择状态"><Option value={0}>空闲</Option><Option value={1}>占用</Option><Option value={2}>维护</Option></Select></Form.Item>
          <Form.Item name="description" label="描述"><TextArea rows={3} placeholder="请输入描述" /></Form.Item>
        </Form>
      </Modal>
      <ExcelImport entityType="workstation" entityName="工位" visible={importVisible} onClose={() => setImportVisible(false)} onImportSuccess={handleImportSuccess} />
      <ExcelExport entityType="workstation" entityName="工位" visible={exportVisible} onClose={() => setExportVisible(false)} />
    </div>
  );
};

export default WorkstationManagement;
