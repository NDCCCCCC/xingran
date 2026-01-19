/**
 * 机房设备管理页面
 */

import { useState, useEffect, useCallback } from 'react';
import type { FC } from 'react';
import {
  Table, Button, Space, Modal, Form, Input, InputNumber, Select, Popconfirm,
  Tag, Card, Row, Col, Statistic, Alert, Radio, DatePicker,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  PlusOutlined, EditOutlined, DeleteOutlined, SearchOutlined, ReloadOutlined,
  CheckCircleOutlined, StopOutlined, WarningOutlined, AppstoreOutlined, TableOutlined,
  ImportOutlined, ExportOutlined, AppstoreAddOutlined,
} from '@ant-design/icons';
import type { RoomDevice } from '@/types';
import { roomDeviceApi, serverRoomApi } from '@/lib/opsApi';
import { useTableManager } from '@/hooks/useTableManager';
import { usePagination } from '@/hooks/usePagination';
import { handleApiError, handleSuccess } from '@/utils/errorHandler';
import { createDateTimeColumn } from '@/utils/tableHelpers';
import ActionButtons from '@/components/shared/ActionButtons';
import ExcelImport from '@/components/shared/ExcelImport';
import ExcelExport from '@/components/shared/ExcelExport';

const { Option } = Select;
const { TextArea } = Input;

type ViewMode = 'table' | 'card';

interface RoomOption {
  id: string;
  name: string;
}

const RoomDeviceManagement: FC = () => {
  const [statistics, setStatistics] = useState({ total: 0, normal: 0, fault: 0, scrapped: 0 });
  const [viewMode, setViewMode] = useState<ViewMode>('table');
  const [importVisible, setImportVisible] = useState(false);
  const [exportVisible, setExportVisible] = useState(false);
  const [roomOptions, setRoomOptions] = useState<RoomOption[]>([]);

  // 使用全局分页 hook
  const { paginationProps, setCurrent, setPageSize, setTotal } = usePagination();

  const {
    loading, data: devices, total, selectedRowKeys,
    searchForm, editForm: deviceForm, editModalVisible: modalVisible,
    editingItem: editingDevice, setSelectedRowKeys, setEditModalVisible: setModalVisible,
    setEditingItem: setEditingDevice, handleSearch,
    handleReset, handleAdd, handleEdit, handleModalClose, loadData: loadDevices, resetSelection,
  } = useTableManager<RoomDevice>(
    async (params) => {
      const result = await roomDeviceApi.list(params);
      return { list: result.data?.list || [], total: result.data?.total || 0 };
    },
    {
      externalPagination: {
        current: paginationProps.current,
        pageSize: paginationProps.pageSize,
        setCurrent,
        setPageSize,
      },
    }
  );

  const loadStatistics = useCallback(async () => {
    try {
      const result = await roomDeviceApi.list({ current: 1, pageSize: 1000 });
      const allDevices = result.data?.list || [];
      setStatistics({
        total: allDevices.length,
        normal: allDevices.filter((d: RoomDevice) => d.status === 0).length,
        fault: allDevices.filter((d: RoomDevice) => d.status === 1).length,
        scrapped: allDevices.filter((d: RoomDevice) => d.status === 2).length,
      });
    } catch (error) { handleApiError(error, '加载统计数据', false); }
  }, []);

  const loadRoomOptions = useCallback(async () => {
    try {
      const result = await serverRoomApi.list({ current: 1, pageSize: 1000 });
      const rooms = result.data?.list || [];
      setRoomOptions(rooms.map((r: any) => ({ id: r.id, name: r.name })));
    } catch (error) { handleApiError(error, '加载机房选项', false); }
  }, []);

  useEffect(() => {
    // 使用 setTimeout 避免同步 setState
    setTimeout(() => {
      loadDevices();
      loadStatistics();
      loadRoomOptions();
    }, 0);
  }, [paginationProps.current, paginationProps.pageSize]);

  const handleSave = async () => {
    try {
      const values = await deviceForm.validateFields();
      if (editingDevice) {
        await roomDeviceApi.update(editingDevice.id, values);
        handleSuccess('更新');
      } else {
        await roomDeviceApi.create(values);
        handleSuccess('创建');
      }
      handleModalClose();
      loadDevices();
      loadStatistics();
    } catch (error: any) { if (error.errorFields) return; handleApiError(error, '操作'); }
  };

  const handleDelete = async (id: string) => {
    try { await roomDeviceApi.delete(id); handleSuccess('删除'); loadDevices(); loadStatistics(); }
    catch (error) { handleApiError(error, '删除'); }
  };

  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) return;
    try { await roomDeviceApi.batch('delete', { ids: selectedRowKeys }); handleSuccess('批量删除'); resetSelection(); loadDevices(); loadStatistics(); }
    catch (error) { handleApiError(error, '批量删除'); }
  };

  const openModal = (record?: RoomDevice) => {
    if (record) { handleEdit(record); deviceForm.setFieldsValue(record); }
    else { handleAdd(); deviceForm.setFieldsValue({ status: 0, deviceType: 'server' }); }
  };

  const handleImportSuccess = () => { loadDevices(); loadStatistics(); setImportVisible(false); };

  const getDeviceTypeText = (type: string) => {
    const types: Record<string, string> = { server: '服务器', storage: '存储设备', ups: 'UPS', pdu: 'PDU', firewall: '防火墙', kvm: 'KVM', cabinet: '机柜', other: '其他' };
    return types[type] || type;
  };

  const getStatusText = (status: number) => {
    const statusMap = { 0: '正常', 1: '故障', 2: '报废' };
    return statusMap[status as keyof typeof statusMap] || '未知';
  };

  const columns: ColumnsType<RoomDevice> = [
    { title: '设备名称', dataIndex: 'deviceName', key: 'deviceName', width: 150 },
    { title: '设备编码', dataIndex: 'deviceCode', key: 'deviceCode', width: 150 },
    {
      title: '所属机房',
      dataIndex: 'roomName',
      key: 'roomName',
      width: 120,
      render: (_, record) => record.roomName || record.roomCode,
    },
    {
      title: '设备类型',
      dataIndex: 'deviceType',
      key: 'deviceType',
      width: 100,
      render: (type) => <Tag>{getDeviceTypeText(type)}</Tag>,
    },
    { title: '厂商', dataIndex: 'vendor', key: 'vendor', width: 100, render: (v) => v || '-' },
    { title: '型号', dataIndex: 'model', key: 'model', width: 120, render: (v) => v || '-' },
    { title: '位置', dataIndex: 'positionDesc', key: 'positionDesc', width: 100, render: (v) => v || '-' },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 80,
      render: (status) => {
        const colors: Record<number, string> = { 0: 'success', 1: 'error', 2: 'default' };
        return <Tag color={colors[status as keyof typeof colors]}>{getStatusText(status)}</Tag>;
      },
    },
    createDateTimeColumn('createdAt', { width: 180 }),
    {
      title: '操作', key: 'action',
      render: (_, record) => {
        const actions = [
          { key: 'edit', label: '编辑', icon: <EditOutlined />, onClick: () => openModal(record) },
          {
            key: 'delete', label: '删除', icon: <DeleteOutlined />, danger: true,
            render: () => (
              <Popconfirm title="确定要删除这个设备吗？" onConfirm={() => handleDelete(record.id)} okText="确定" cancelText="取消">
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
    if (devices.length === 0) return <div style={{ textAlign: 'center', padding: '40px', color: '#999' }}>暂无数据</div>;
    return (
      <Row gutter={[16, 16]}>
        {devices.map((device) => (
          <Col key={device.id} xs={24} sm={12} md={8} lg={6}>
            <Card
              hoverable
              actions={[
                <EditOutlined key="edit" onClick={() => openModal(device)} />,
                <Popconfirm key="delete" title="确定要删除这个设备吗？" onConfirm={() => handleDelete(device.id)} okText="确定" cancelText="取消">
                  <DeleteOutlined style={{ color: '#ff4d4f' }} />
                </Popconfirm>,
              ]}
            >
              <Card.Meta
                title={
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <span>{device.deviceName}</span>
                    <Tag color={device.status === 0 ? 'success' : device.status === 1 ? 'error' : 'default'}>{getStatusText(device.status)}</Tag>
                  </div>
                }
                description={
                  <div>
                    <div><strong>编码：</strong>{device.deviceCode}</div>
                    <div><strong>机房：</strong>{device.roomName || device.roomCode}</div>
                    <div><strong>类型：</strong>{getDeviceTypeText(device.deviceType)}</div>
                    {device.vendor && <div><strong>厂商：</strong>{device.vendor}</div>}
                    {device.model && <div><strong>型号：</strong>{device.model}</div>}
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
          <Col span={6}><Card><Statistic title="总设备数" value={statistics.total} prefix={<AppstoreAddOutlined />} /></Card></Col>
          <Col span={6}><Card><Statistic title="正常" value={statistics.normal} styles={{ content: { color: '#3f8600' } }} prefix={<CheckCircleOutlined />} /></Card></Col>
          <Col span={6}><Card><Statistic title="故障" value={statistics.fault} styles={{ content: { color: '#cf1322' } }} prefix={<WarningOutlined />} /></Card></Col>
          <Col span={6}><Card><Statistic title="报废" value={statistics.scrapped} styles={{ content: { color: '#8c8c8c' } }} prefix={<StopOutlined />} /></Card></Col>
        </Row>
      )}
      <Card style={{ marginBottom: 16 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', flexWrap: 'wrap', gap: '16px' }}>
          <Form form={searchForm} layout="inline" style={{ flex: 1, minWidth: 0 }}>
            <Form.Item name="roomId" label="所属机房">
              <Select placeholder="请选择机房">
                {roomOptions.map(r => <Option key={r.id} value={r.id}>{r.name}</Option>)}
              </Select>
            </Form.Item>
            <Form.Item name="name" label="设备名称">
              <Input placeholder="请输入设备名称" />
            </Form.Item>
            <Form.Item name="deviceType" label="设备类型">
              <Select placeholder="请选择类型">
                <Option value="server">服务器</Option>
                <Option value="storage">存储设备</Option>
                <Option value="ups">UPS</Option>
                <Option value="pdu">PDU</Option>
                <Option value="firewall">防火墙</Option>
                <Option value="kvm">KVM</Option>
                <Option value="cabinet">机柜</Option>
                <Option value="other">其他</Option>
              </Select>
            </Form.Item>
            <Form.Item name="status" label="状态">
              <Select placeholder="请选择状态">
                <Option value={0}>正常</Option>
                <Option value={1}>故障</Option>
                <Option value={2}>报废</Option>
              </Select>
            </Form.Item>
            <Form.Item>
              <Space>
                <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>搜索</Button>
                <Button onClick={handleReset}>重置</Button>
                <Button icon={<ReloadOutlined />} onClick={() => { loadDevices(); loadStatistics(); }}>刷新</Button>
              </Space>
            </Form.Item>
          </Form>
          <Space>
            <Radio.Group value={viewMode} onChange={(e) => setViewMode(e.target.value)} buttonStyle="solid">
              <Radio.Button value="table"><TableOutlined /> 表格</Radio.Button><Radio.Button value="card"><AppstoreOutlined /> 卡片</Radio.Button>
            </Radio.Group>
            <Button icon={<ImportOutlined />} onClick={() => setImportVisible(true)}>导入</Button>
            <Button icon={<ExportOutlined />} onClick={() => setExportVisible(true)}>导出</Button>
            {selectedRowKeys.length > 0 && <Button danger icon={<DeleteOutlined />} onClick={handleBatchDelete}>批量删除 ({selectedRowKeys.length})</Button>}
            <Button type="primary" icon={<PlusOutlined />} onClick={() => openModal()}>新增设备</Button>
          </Space>
        </div>
        {selectedRowKeys.length > 0 && <Alert title={<span>已选择 <strong>{selectedRowKeys.length}</strong> 个设备，<Button type="link" size="small" onClick={() => setSelectedRowKeys([])} style={{ padding: 0 }}>取消选择</Button></span>} type="info" showIcon style={{ marginTop: 12 }} />}
      </Card>
      <Card>
        {viewMode === 'table' ? (
          <Table rowSelection={{ selectedRowKeys, onChange: setSelectedRowKeys }} columns={columns} dataSource={devices} loading={loading} rowKey="id" pagination={paginationProps} />
        ) : renderCardView()}
      </Card>
      <Modal title={editingDevice ? '编辑设备' : '新增设备'} open={modalVisible} onOk={handleSave} onCancel={() => { setModalVisible(false); deviceForm.resetFields(); setEditingDevice(null); }} width={700}>
        <Form form={deviceForm} layout="horizontal" labelCol={{ span: 5 }} wrapperCol={{ span: 19 }}>
          <Form.Item name="roomId" label="所属机房" rules={[{ required: true, message: '请选择所属机房' }]}>
            <Select placeholder="请选择所属机房" className="user-form-input">
              {roomOptions.map(r => <Option key={r.id} value={r.id}>{r.name}</Option>)}
            </Select>
          </Form.Item>
          <Form.Item name="name" label="设备名称" rules={[{ required: true, message: '请输入设备名称' }]}>
            <Input placeholder="请输入设备名称" className="user-form-input" />
          </Form.Item>
          <Form.Item name="deviceType" label="设备类型" rules={[{ required: true, message: '请选择设备类型' }]}>
            <Select placeholder="请选择设备类型" className="user-form-input">
              <Option value="server">服务器</Option>
              <Option value="storage">存储设备</Option>
              <Option value="ups">UPS</Option>
              <Option value="pdu">PDU</Option>
              <Option value="firewall">防火墙</Option>
              <Option value="kvm">KVM</Option>
              <Option value="cabinet">机柜</Option>
              <Option value="other">其他</Option>
            </Select>
          </Form.Item>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="vendor" label="厂商">
                <Input placeholder="请输入厂商" className="user-form-input" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="model" label="型号">
                <Input placeholder="请输入型号" className="user-form-input" />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="serialNumber" label="序列号">
                <Input placeholder="请输入序列号" className="user-form-input" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="positionDesc" label="位置描述">
                <Input placeholder="请输入位置描述" className="user-form-input" />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="positionU" label="U位置">
                <InputNumber min={0} placeholder="U位置" style={{ width: '100%' }} className="user-form-input" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="heightU" label="高度(U)">
                <InputNumber min={0} placeholder="高度" style={{ width: '100%' }} className="user-form-input" />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="powerConsumption" label="功耗(W)">
                <InputNumber min={0} placeholder="功耗" style={{ width: '100%' }} className="user-form-input" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="status" label="状态" rules={[{ required: true, message: '请选择状态' }]}>
                <Select placeholder="请选择状态" className="user-form-input">
                  <Option value={0}>正常</Option>
                  <Option value={1}>故障</Option>
                  <Option value={2}>报废</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="purchaseDate" label="购买日期">
                <DatePicker style={{ width: '100%' }} className="user-form-input" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="warrantyDate" label="保修到期">
                <DatePicker style={{ width: '100%' }} className="user-form-input" />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item name="description" label="描述">
            <TextArea rows={3} placeholder="请输入描述" className="user-form-input" />
          </Form.Item>
        </Form>
      </Modal>
      <ExcelImport entityType="room-device" entityName="机房设备" visible={importVisible} onClose={() => setImportVisible(false)} onImportSuccess={handleImportSuccess} />
      <ExcelExport entityType="room-device" entityName="机房设备" visible={exportVisible} onClose={() => setExportVisible(false)} />
    </div>
  );
};

export default RoomDeviceManagement;
