/**
 * 专线管理页面
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
  CheckCircleOutlined, StopOutlined, WarningOutlined, AppstoreOutlined, TableOutlined,
  ImportOutlined, ExportOutlined, LineChartOutlined,
} from '@ant-design/icons';
import type { DedicatedLine } from '@/types';
import { dedicatedLineApi, serverRoomApi } from '@/lib/opsApi';
import { useTableManager } from '@/hooks/useTableManager';
import { handleApiError, handleSuccess } from '@/utils/errorHandler';
import { createDateTimeColumn } from '@/utils/tableHelpers';
import ActionButtons from '@/components/shared/ActionButtons';
import ExcelImport from '@/components/shared/ExcelImport';
import ExcelExport from '@/components/shared/ExcelExport';

const { Option } = Select;
const { TextArea } = Input;

interface ServerRoomOption {
  id: string;
  name: string;
}

type ViewMode = 'table' | 'card';

const DedicatedLineManagement: FC = () => {
  const [statistics, setStatistics] = useState({ total: 0, normal: 0, fault: 0, disabled: 0 });
  const [viewMode, setViewMode] = useState<ViewMode>('table');
  const [importVisible, setImportVisible] = useState(false);
  const [exportVisible, setExportVisible] = useState(false);
  const [serverRooms, setServerRooms] = useState<ServerRoomOption[]>([]);

  const loadServerRooms = useCallback(async () => {
    try {
      const result = await serverRoomApi.list({ current: 1, pageSize: 1000 });
      const rooms = result.data?.list || [];
      setServerRooms(rooms.map((r: any) => ({ id: r.id, name: r.name })));
    } catch (error) {
      handleApiError(error, '加载机房列表', false);
    }
  }, []);

  const {
    loading, data: lines, total, current, pageSize, selectedRowKeys,
    searchForm, editForm: lineForm, editModalVisible: modalVisible,
    editingItem: editingLine, setSelectedRowKeys, setEditModalVisible: setModalVisible,
    setEditingItem: setEditingLine, setCurrent, setPageSize, handleSearch,
    handleReset, handleAdd, handleEdit, handleModalClose, loadData: loadLines, resetSelection,
  } = useTableManager<DedicatedLine>(
    async (params) => {
      const result = await dedicatedLineApi.list(params);
      return { list: result.data?.list || [], total: result.data?.total || 0 };
    },
    { pageSize: 10 }
  );

  const loadStatistics = useCallback(async () => {
    try {
      const result = await dedicatedLineApi.list({ current: 1, pageSize: 1000 });
      const allLines = result.data?.list || [];
      setStatistics({
        total: allLines.length,
        normal: allLines.filter((l: DedicatedLine) => l.status === 0).length,
        fault: allLines.filter((l: DedicatedLine) => l.status === 1).length,
        disabled: allLines.filter((l: DedicatedLine) => l.status === 2).length,
      });
    } catch (error) { handleApiError(error, '加载统计数据', false); }
  }, []);

  useEffect(() => { loadLines(); loadStatistics(); loadServerRooms(); }, [loadServerRooms]);

  const handleSave = async () => {
    try {
      const values = await lineForm.validateFields();
      if (editingLine) {
        await dedicatedLineApi.update(editingLine.id, values);
        handleSuccess('更新');
      } else {
        await dedicatedLineApi.create(values);
        handleSuccess('创建');
      }
      handleModalClose();
      loadLines();
      loadStatistics();
    } catch (error: any) { if (error.errorFields) return; handleApiError(error, '操作'); }
  };

  const handleDelete = async (id: string) => {
    try { await dedicatedLineApi.delete(id); handleSuccess('删除'); loadLines(); loadStatistics(); }
    catch (error) { handleApiError(error, '删除'); }
  };

  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) return;
    try { await dedicatedLineApi.batch('delete', { ids: selectedRowKeys }); handleSuccess('批量删除'); resetSelection(); loadLines(); loadStatistics(); }
    catch (error) { handleApiError(error, '批量删除'); }
  };

  const openModal = (record?: DedicatedLine) => {
    if (record) { handleEdit(record); lineForm.setFieldsValue(record); }
    else { handleAdd(); lineForm.setFieldsValue({ status: 0, lineType: 'MPLS' }); }
  };

  const handleImportSuccess = () => { loadLines(); loadStatistics(); setImportVisible(false); };

  const getLineTypeText = (type: string) => {
    const types: Record<string, string> = { MPLS: 'MPLS', Internet: '互联网专线', LeasedLine: '租用专线', Fiber: '光纤专线' };
    return types[type] || type;
  };

  const getStatusText = (status: number) => {
    const statusMap = { 0: '正常', 1: '故障', 2: '停用' };
    return statusMap[status as keyof typeof statusMap] || '未知';
  };

  const columns: ColumnsType<DedicatedLine> = [
    { title: '专线编码', dataIndex: 'lineCode', key: 'lineCode', width: 150 },
    {
      title: '专线类型',
      dataIndex: 'lineType',
      key: 'lineType',
      width: 120,
      render: (type) => <Tag>{getLineTypeText(type)}</Tag>,
    },
    { title: '带宽', dataIndex: 'bandwidth', key: 'bandwidth', width: 100, render: (v) => v || '-' },
    { title: '运营商', dataIndex: 'isp', key: 'isp', width: 120 },
    { title: '源机房', dataIndex: 'sourceRoomName', key: 'sourceRoomName', width: 120, render: (v) => v || '-' },
    { title: '源设备', dataIndex: 'sourceDeviceName', key: 'sourceDeviceName', width: 120, render: (v) => v || '-' },
    { title: '源端口', dataIndex: 'sourcePort', key: 'sourcePort', width: 100, render: (v) => v || '-' },
    { title: '目的机房', dataIndex: 'destRoomName', key: 'destRoomName', width: 120, render: (v) => v || '-' },
    { title: '目的设备', dataIndex: 'destDeviceName', key: 'destDeviceName', width: 120, render: (v) => v || '-' },
    { title: 'IP地址', dataIndex: 'ipAddress', key: 'ipAddress', width: 120, render: (v) => v || '-' },
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
    { title: '月费(元)', dataIndex: 'monthlyFee', key: 'monthlyFee', width: 100, render: (v) => v ? `¥${v}` : '-' },
    createDateTimeColumn('createdAt', { width: 180 }),
    {
      title: '操作', key: 'action',
      render: (_, record) => {
        const actions = [
          { key: 'edit', label: '编辑', icon: <EditOutlined />, onClick: () => openModal(record) },
          {
            key: 'delete', label: '删除', icon: <DeleteOutlined />, danger: true,
            render: () => (
              <Popconfirm title="确定要删除这条专线吗？" onConfirm={() => handleDelete(record.id)} okText="确定" cancelText="取消">
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
    if (lines.length === 0) return <div style={{ textAlign: 'center', padding: '40px', color: '#999' }}>暂无数据</div>;
    return (
      <Row gutter={[16, 16]}>
        {lines.map((line) => (
          <Col key={line.id} xs={24} sm={12} md={8} lg={6}>
            <Card
              hoverable
              actions={[
                <EditOutlined key="edit" onClick={() => openModal(line)} />,
                <Popconfirm key="delete" title="确定要删除这条专线吗？" onConfirm={() => handleDelete(line.id)} okText="确定" cancelText="取消">
                  <DeleteOutlined style={{ color: '#ff4d4f' }} />
                </Popconfirm>,
              ]}
            >
              <Card.Meta
                title={
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <span>{line.lineCode}</span>
                    <Tag color={line.status === 0 ? 'success' : line.status === 1 ? 'error' : 'default'}>{getStatusText(line.status)}</Tag>
                  </div>
                }
                description={
                  <div>
                    <div><strong>类型：</strong>{getLineTypeText(line.lineType)}</div>
                    <div><strong>运营商：</strong>{line.isp}</div>
                    {line.bandwidth && <div><strong>带宽：</strong>{line.bandwidth}</div>}
                    {line.ipAddress && <div><strong>IP：</strong>{line.ipAddress}</div>}
                    {line.monthlyFee && <div><strong>月费：</strong>¥{line.monthlyFee}</div>}
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
          <Col span={6}><Card><Statistic title="总专线数" value={statistics.total} prefix={<LineChartOutlined />} /></Card></Col>
          <Col span={6}><Card><Statistic title="正常" value={statistics.normal} styles={{ content: { color: '#3f8600' } }} prefix={<CheckCircleOutlined />} /></Card></Col>
          <Col span={6}><Card><Statistic title="故障" value={statistics.fault} styles={{ content: { color: '#cf1322' } }} prefix={<WarningOutlined />} /></Card></Col>
          <Col span={6}><Card><Statistic title="停用" value={statistics.disabled} styles={{ content: { color: '#8c8c8c' } }} prefix={<StopOutlined />} /></Card></Col>
        </Row>
      )}
      <Card style={{ marginBottom: 16 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', flexWrap: 'wrap', gap: '16px' }}>
          <Form form={searchForm} layout="inline" style={{ flex: 1, minWidth: 0 }}>
            <Form.Item name="lineCode" label="专线编码"><Input placeholder="请输入专线编码" allowClear style={{ width: 150 }} /></Form.Item>
            <Form.Item name="lineType" label="专线类型">
              <Select placeholder="请选择类型" allowClear style={{ width: 120 }}>
                <Option value="MPLS">MPLS</Option><Option value="Internet">互联网专线</Option><Option value="LeasedLine">租用专线</Option><Option value="Fiber">光纤专线</Option>
              </Select>
            </Form.Item>
            <Form.Item name="isp" label="运营商"><Input placeholder="请输入运营商" allowClear style={{ width: 120 }} /></Form.Item>
            <Form.Item name="status" label="状态">
              <Select placeholder="请选择状态" allowClear style={{ width: 100 }}><Option value={0}>正常</Option><Option value={1}>故障</Option><Option value={2}>停用</Option></Select>
            </Form.Item>
            <Form.Item>
              <Space><Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>搜索</Button><Button onClick={handleReset}>重置</Button><Button icon={<ReloadOutlined />} onClick={() => { loadLines(); loadStatistics(); }}>刷新</Button></Space>
            </Form.Item>
          </Form>
          <Space>
            <Radio.Group value={viewMode} onChange={(e) => setViewMode(e.target.value)} buttonStyle="solid">
              <Radio.Button value="table"><TableOutlined /> 表格</Radio.Button><Radio.Button value="card"><AppstoreOutlined /> 卡片</Radio.Button>
            </Radio.Group>
            <Button icon={<ImportOutlined />} onClick={() => setImportVisible(true)}>导入</Button>
            <Button icon={<ExportOutlined />} onClick={() => setExportVisible(true)}>导出</Button>
            {selectedRowKeys.length > 0 && <Button danger icon={<DeleteOutlined />} onClick={handleBatchDelete}>批量删除 ({selectedRowKeys.length})</Button>}
            <Button type="primary" icon={<PlusOutlined />} onClick={() => openModal()}>新增专线</Button>
          </Space>
        </div>
        {selectedRowKeys.length > 0 && <Alert message={<span>已选择 <strong>{selectedRowKeys.length}</strong> 条专线，<Button type="link" size="small" onClick={() => setSelectedRowKeys([])} style={{ padding: 0 }}>取消选择</Button></span>} type="info" showIcon style={{ marginTop: 12 }} />}
      </Card>
      <Card>
        {viewMode === 'table' ? (
          <Table rowSelection={{ selectedRowKeys, onChange: setSelectedRowKeys }} columns={columns} dataSource={lines} loading={loading} rowKey="id" pagination={{ current, pageSize, total, showSizeChanger: true, showQuickJumper: true, showTotal: (total) => `共 ${total} 条`, onChange: (page, size) => { setCurrent(page); setPageSize(size || 10); loadLines({ current: page, pageSize: size || 10 }); } }} />
        ) : renderCardView()}
      </Card>
      <Modal title={editingLine ? '编辑专线' : '新增专线'} open={modalVisible} onOk={handleSave} onCancel={() => { setModalVisible(false); lineForm.resetFields(); setEditingLine(null); }} width={700}>
        <Form form={lineForm} layout="horizontal" labelCol={{ span: 5 }} wrapperCol={{ span: 19 }}>
          <Form.Item name="lineCode" label="专线编码" rules={[{ required: true, message: '请输入专线编码' }]}><Input placeholder="请输入专线编码" disabled={!!editingLine} /></Form.Item>
          <Form.Item name="lineType" label="专线类型" rules={[{ required: true, message: '请选择专线类型' }]}>
            <Select placeholder="请选择专线类型"><Option value="MPLS">MPLS</Option><Option value="Internet">互联网专线</Option><Option value="LeasedLine">租用专线</Option><Option value="Fiber">光纤专线</Option></Select>
          </Form.Item>
          <Row gutter={16}>
            <Col span={12}><Form.Item name="bandwidth" label="带宽"><Input placeholder="如：100M, 1G" /></Form.Item></Col>
            <Col span={12}><Form.Item name="isp" label="运营商" rules={[{ required: true, message: '请输入运营商' }]}><Input placeholder="电信/移动/联通等" /></Form.Item></Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}><Form.Item name="sourceRoomId" label="源机房">
              <Select placeholder="请选择源机房" allowClear showSearch>
                {serverRooms.map(room => <Option key={room.id} value={room.id}>{room.name}</Option>)}
              </Select>
            </Form.Item></Col>
            <Col span={12}><Form.Item name="sourceDeviceName" label="源设备名称"><Input placeholder="请输入源设备名称" /></Form.Item></Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}><Form.Item name="sourcePort" label="源端口"><Input placeholder="请输入源端口" /></Form.Item></Col>
            <Col span={12}></Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}><Form.Item name="destRoomId" label="目的机房">
              <Select placeholder="请选择目的机房" allowClear showSearch>
                {serverRooms.map(room => <Option key={room.id} value={room.id}>{room.name}</Option>)}
              </Select>
            </Form.Item></Col>
            <Col span={12}><Form.Item name="destDeviceName" label="目的设备名称"><Input placeholder="请输入目的设备名称" /></Form.Item></Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}><Form.Item name="destPort" label="目的端口"><Input placeholder="请输入目的端口" /></Form.Item></Col>
            <Col span={12}></Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}><Form.Item name="ipAddress" label="IP地址"><Input placeholder="请输入IP地址" /></Form.Item></Col>
            <Col span={12}><Form.Item name="subnetMask" label="子网掩码"><Input placeholder="请输入子网掩码" /></Form.Item></Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}><Form.Item name="vlan" label="VLAN"><Input placeholder="请输入VLAN" /></Form.Item></Col>
            <Col span={12}><Form.Item name="contractNo" label="合同编号"><Input placeholder="请输入合同编号" /></Form.Item></Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}><Form.Item name="carrierContactName" label="运营商联系人"><Input placeholder="请输入联系人" /></Form.Item></Col>
            <Col span={12}><Form.Item name="carrierContactPhone" label="联系人电话"><Input placeholder="请输入联系电话" /></Form.Item></Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}><Form.Item name="monthlyFee" label="月租费用"><InputNumber min={0} precision={2} placeholder="请输入月租费用" style={{ width: '100%' }} /></Form.Item></Col>
            <Col span={12}><Form.Item name="status" label="状态" rules={[{ required: true, message: '请选择状态' }]}><Select placeholder="请选择状态"><Option value={0}>正常</Option><Option value={1}>故障</Option><Option value={2}>停用</Option></Select></Form.Item></Col>
          </Row>
          <Form.Item name="description" label="描述"><TextArea rows={3} placeholder="请输入描述" /></Form.Item>
        </Form>
      </Modal>
      <ExcelImport entityType="dedicated-line" entityName="专线" visible={importVisible} onClose={() => setImportVisible(false)} onImportSuccess={handleImportSuccess} />
      <ExcelExport entityType="dedicated-line" entityName="专线" visible={exportVisible} onClose={() => setExportVisible(false)} />
    </div>
  );
};

export default DedicatedLineManagement;
