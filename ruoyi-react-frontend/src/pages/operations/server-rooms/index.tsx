/**
 * 机房管理页面
 */

import { useState, useEffect, useCallback } from 'react';
import type { FC } from 'react';
import {
  Table, Button, Space, Modal, Form, Input, Select, Popconfirm,
  Tag, Card, Row, Col, Statistic, Alert, Radio,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  PlusOutlined, EditOutlined, DeleteOutlined, SearchOutlined, ReloadOutlined,
  CheckCircleOutlined, StopOutlined, AppstoreOutlined, TableOutlined,
  ImportOutlined, ExportOutlined,
} from '@ant-design/icons';
import type { ServerRoom } from '@/types';
import { serverRoomApi, buildingApi } from '@/lib/opsApi';
import { useTableManager } from '@/hooks/useTableManager';
import { handleApiError, handleSuccess } from '@/utils/errorHandler';
import { createStatusColumn, createDateTimeColumn } from '@/utils/tableHelpers';
import ActionButtons from '@/components/shared/ActionButtons';
import ExcelImport from '@/components/shared/ExcelImport';
import ExcelExport from '@/components/shared/ExcelExport';

const { Option } = Select;
const { TextArea } = Input;

type ViewMode = 'table' | 'card';

interface BuildingOption {
  code: string;
  name: string;
}

const ServerRoomManagement: FC = () => {
  const [statistics, setStatistics] = useState({ total: 0, active: 0, inactive: 0 });
  const [viewMode, setViewMode] = useState<ViewMode>('table');
  const [importVisible, setImportVisible] = useState(false);
  const [exportVisible, setExportVisible] = useState(false);
  const [buildingOptions, setBuildingOptions] = useState<BuildingOption[]>([]);

  const {
    loading, data: serverRooms, total, current, pageSize, selectedRowKeys,
    searchForm, editForm: serverRoomForm, editModalVisible: modalVisible,
    editingItem: editingServerRoom, setSelectedRowKeys, setEditModalVisible: setModalVisible,
    setEditingItem: setEditingServerRoom, setCurrent, setPageSize, handleSearch,
    handleReset, handleAdd, handleEdit, handleModalClose, loadData: loadServerRooms, resetSelection,
  } = useTableManager<ServerRoom>(
    async (params) => {
      const result = await serverRoomApi.list(params);
      return { list: result.data?.list || [], total: result.data?.total || 0 };
    },
    { pageSize: 10 }
  );

  const loadStatistics = useCallback(async () => {
    try {
      const result = await serverRoomApi.list({ current: 1, pageSize: 1000 });
      const allRooms = result.data?.list || [];
      setStatistics({
        total: allRooms.length,
        active: allRooms.filter((r: ServerRoom) => r.status === 0).length,
        inactive: allRooms.filter((r: ServerRoom) => r.status === 1).length,
      });
    } catch (error) { handleApiError(error, '加载统计数据', false); }
  }, []);

  const loadBuildingOptions = useCallback(async () => {
    try {
      const result = await buildingApi.list({ current: 1, pageSize: 1000 });
      const buildings = result.data?.list || [];
      setBuildingOptions(buildings.map((b: any) => ({ code: b.code, name: b.name })));
    } catch (error) { handleApiError(error, '加载楼宇选项', false); }
  }, []);

  useEffect(() => { loadServerRooms(); loadStatistics(); loadBuildingOptions(); }, []);

  const handleSave = async () => {
    try {
      const values = await serverRoomForm.validateFields();
      if (editingServerRoom) {
        await serverRoomApi.update(editingServerRoom.id, values);
        handleSuccess('更新');
      } else {
        await serverRoomApi.create(values);
        handleSuccess('创建');
      }
      handleModalClose();
      loadServerRooms();
      loadStatistics();
    } catch (error: any) { if (error.errorFields) return; handleApiError(error, '操作'); }
  };

  const handleDelete = async (id: string) => {
    try { await serverRoomApi.delete(id); handleSuccess('删除'); loadServerRooms(); loadStatistics(); }
    catch (error) { handleApiError(error, '删除'); }
  };

  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) return;
    try { await serverRoomApi.batch('delete', { ids: selectedRowKeys }); handleSuccess('批量删除'); resetSelection(); loadServerRooms(); loadStatistics(); }
    catch (error) { handleApiError(error, '批量删除'); }
  };

  const openModal = (record?: ServerRoom) => {
    if (record) { handleEdit(record); serverRoomForm.setFieldsValue(record); }
    else { handleAdd(); serverRoomForm.setFieldsValue({ status: 0 }); }
  };

  const handleImportSuccess = () => { loadServerRooms(); loadStatistics(); setImportVisible(false); };

  const columns: ColumnsType<ServerRoom> = [
    { title: '机房名称', dataIndex: 'name', key: 'name', width: 150 },
    {
      title: '所在楼宇',
      dataIndex: 'buildingName',
      key: 'buildingName',
      width: 120,
      render: (_, record) => record.buildingName || record.buildingId,
    },
    { title: '楼层', dataIndex: 'floorNo', key: 'floorNo', width: 80 },
    createStatusColumn('status', { width: 100 }),
    { title: '描述', dataIndex: 'remark', key: 'remark', ellipsis: true },
    createDateTimeColumn('createdAt', { width: 180 }),
    {
      title: '操作', key: 'action',
      render: (_, record) => {
        const actions = [
          { key: 'edit', label: '编辑', icon: <EditOutlined />, onClick: () => openModal(record) },
          {
            key: 'delete', label: '删除', icon: <DeleteOutlined />, danger: true,
            render: () => (
              <Popconfirm title="确定要删除这个机房吗？" onConfirm={() => handleDelete(record.id)} okText="确定" cancelText="取消">
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
    if (serverRooms.length === 0) return <div style={{ textAlign: 'center', padding: '40px', color: '#999' }}>暂无数据</div>;
    return (
      <Row gutter={[16, 16]}>
        {serverRooms.map((room) => (
          <Col key={room.id} xs={24} sm={12} md={8} lg={6}>
            <Card
              hoverable
              actions={[
                <EditOutlined key="edit" onClick={() => openModal(room)} />,
                <Popconfirm key="delete" title="确定要删除这个机房吗？" onConfirm={() => handleDelete(room.id)} okText="确定" cancelText="取消">
                  <DeleteOutlined style={{ color: '#ff4d4f' }} />
                </Popconfirm>,
              ]}
            >
              <Card.Meta
                title={
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <span>{room.name}</span>
                    <Tag color={room.status === 0 ? 'success' : 'error'}>{room.status === 0 ? '正常' : '停用'}</Tag>
                  </div>
                }
                description={
                  <div>
                    <div><strong>位置：</strong>{room.buildingName || room.buildingId} {room.floorNo}层</div>
                  </div>
                }
              />
            </Card>
          </Col>
        ))}
      </Row>
    );
  };

  return (
    <div>
      {total > 10 && (
        <Row gutter={16} style={{ marginBottom: 16 }}>
          <Col span={8}><Card><Statistic title="总机房数" value={statistics.total} prefix={<CheckCircleOutlined />} /></Card></Col>
          <Col span={8}><Card><Statistic title="正常机房" value={statistics.active} styles={{ content: { color: '#3f8600' } }} prefix={<CheckCircleOutlined />} /></Card></Col>
          <Col span={8}><Card><Statistic title="停用机房" value={statistics.inactive} styles={{ content: { color: '#cf1322' } }} prefix={<StopOutlined />} /></Card></Col>
        </Row>
      )}
      <Card style={{ marginBottom: 16 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', flexWrap: 'wrap', gap: '16px' }}>
          <Form form={searchForm} layout="inline" style={{ flex: 1, minWidth: 0 }}>
            <Form.Item name="buildingCode" label="所在楼宇">
              <Select placeholder="请选择楼宇" allowClear style={{ width: 150 }}>
                {buildingOptions.map(b => <Option key={b.code} value={b.code}>{b.name}</Option>)}
              </Select>
            </Form.Item>
            <Form.Item name="floorNo" label="楼层"><Input placeholder="请输入楼层" allowClear style={{ width: 100 }} /></Form.Item>
            <Form.Item name="name" label="机房名称"><Input placeholder="请输入机房名称" allowClear style={{ width: 150 }} /></Form.Item>
            <Form.Item name="status" label="状态">
              <Select placeholder="请选择状态" allowClear style={{ width: 120 }}><Option value={0}>正常</Option><Option value={1}>停用</Option></Select>
            </Form.Item>
            <Form.Item>
              <Space><Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>搜索</Button><Button onClick={handleReset}>重置</Button><Button icon={<ReloadOutlined />} onClick={() => { loadServerRooms(); loadStatistics(); }}>刷新</Button></Space>
            </Form.Item>
          </Form>
          <Space>
            <Radio.Group value={viewMode} onChange={(e) => setViewMode(e.target.value)} buttonStyle="solid">
              <Radio.Button value="table"><TableOutlined /> 表格</Radio.Button><Radio.Button value="card"><AppstoreOutlined /> 卡片</Radio.Button>
            </Radio.Group>
            <Button icon={<ImportOutlined />} onClick={() => setImportVisible(true)}>导入</Button>
            <Button icon={<ExportOutlined />} onClick={() => setExportVisible(true)}>导出</Button>
            {selectedRowKeys.length > 0 && <Button danger icon={<DeleteOutlined />} onClick={handleBatchDelete}>批量删除 ({selectedRowKeys.length})</Button>}
            <Button type="primary" icon={<PlusOutlined />} onClick={() => openModal()}>新增机房</Button>
          </Space>
        </div>
        {selectedRowKeys.length > 0 && <Alert message={<span>已选择 <strong>{selectedRowKeys.length}</strong> 个机房，<Button type="link" size="small" onClick={() => setSelectedRowKeys([])} style={{ padding: 0 }}>取消选择</Button></span>} type="info" showIcon style={{ marginTop: 12 }} />}
      </Card>
      <Card>
        {viewMode === 'table' ? (
          <Table rowSelection={{ selectedRowKeys, onChange: setSelectedRowKeys }} columns={columns} dataSource={serverRooms} loading={loading} rowKey="id" pagination={{ current, pageSize, total, showSizeChanger: true, showQuickJumper: true, showTotal: (total) => `共 ${total} 条`, onChange: (page, size) => { setCurrent(page); setPageSize(size || 10); loadServerRooms({ current: page, pageSize: size || 10 }); } }} />
        ) : renderCardView()}
      </Card>
      <Modal title={editingServerRoom ? '编辑机房' : '新增机房'} open={modalVisible} onOk={handleSave} onCancel={() => { setModalVisible(false); serverRoomForm.resetFields(); setEditingServerRoom(null); }} width={600}>
        <Form form={serverRoomForm} layout="horizontal" labelCol={{ span: 5 }} wrapperCol={{ span: 19 }}>
          <Form.Item name="buildingId" label="所在楼宇" rules={[{ required: true, message: '请选择所在楼宇' }]}>
            <Select placeholder="请选择所在楼宇">
              {buildingOptions.map(b => <Option key={b.code} value={b.code}>{b.name}</Option>)}
            </Select>
          </Form.Item>
          <Form.Item name="floorNo" label="楼层" rules={[{ required: true, message: '请输入楼层' }]}><Input placeholder="请输入楼层，如：3F" /></Form.Item>
          <Form.Item name="name" label="机房名称" rules={[{ required: true, message: '请输入机房名称' }]}><Input placeholder="请输入机房名称" /></Form.Item>
          <Form.Item name="status" label="状态" rules={[{ required: true, message: '请选择状态' }]}><Select placeholder="请选择状态"><Option value={0}>正常</Option><Option value={1}>停用</Option></Select></Form.Item>
          <Form.Item name="remark" label="描述"><TextArea rows={3} placeholder="请输入描述" /></Form.Item>
        </Form>
      </Modal>
      <ExcelImport entityType="server-room" entityName="机房" visible={importVisible} onClose={() => setImportVisible(false)} onImportSuccess={handleImportSuccess} />
      <ExcelExport entityType="server-room" entityName="机房" visible={exportVisible} onClose={() => setExportVisible(false)} />
    </div>
  );
};

export default ServerRoomManagement;
