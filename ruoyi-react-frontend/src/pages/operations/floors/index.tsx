/**
 * 楼层管理页面
 */

import { useState, useEffect, useCallback } from 'react';
import type { FC } from 'react';
import {
  Table, Button, Space, Modal, Form, Input, InputNumber, Select, Popconfirm,
  Tag, Card, Row, Col, Statistic, Alert, Radio,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  PlusOutlined, EditOutlined, DeleteOutlined, SearchOutlined, ReloadOutlined,
  CheckCircleOutlined, StopOutlined, AppstoreOutlined, TableOutlined,
  ImportOutlined, ExportOutlined,
} from '@ant-design/icons';
import type { Floor } from '@/types';
import { floorApi, buildingApi } from '@/lib/opsApi';
import { useTableManager } from '@/hooks/useTableManager';
import { usePagination } from '@/hooks/usePagination';
import { handleApiError, handleSuccess } from '@/utils/errorHandler';
import { createStatusColumn, createDateTimeColumn } from '@/utils/tableHelpers';
import ActionButtons from '@/components/shared/ActionButtons';
import ExcelImport from '@/components/shared/ExcelImport';
import ExcelExport from '@/components/shared/ExcelExport';

const { Option } = Select;
const { TextArea } = Input;

type ViewMode = 'table' | 'card';

interface BuildingOption {
  id: string;
  code: string;
  name: string;
}

const FloorManagement: FC = () => {
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

  return (
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
};

export default FloorManagement;
