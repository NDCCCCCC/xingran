import { useEffect, useCallback } from 'react';
import type { FC } from 'react';
import { post } from '@/lib/api';
import {
  Table,
  Button,
  Space,
  Modal,
  Form,
  Input,
  Select,
  Popconfirm,
  Tag,
  Card,
  Row,
  Col,
  Statistic,
  DatePicker,
  InputNumber,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  SearchOutlined,
  ReloadOutlined,
  DesktopOutlined,
} from '@ant-design/icons';
import { roomDeviceApi } from '@/services/operations/room-devices';
import { useTableManager } from '@/hooks/useTableManager';
import type { RoomDevice } from '@/types/operations';

const { Option } = Select;


const RoomDeviceManagement: FC = () => {
  const fetchRoomDevices = useCallback(async (params: any) => {
    const result = await roomDeviceApi.list(params);
    return { list: result.data.list, total: result.data.total };
  }, []);

  const {
    loading,
    data: roomDevices,
    total,
    current,
    pageSize,
    selectedRowKeys,
    searchForm,
    editForm,
    editModalVisible,
    editingItem,
    setCurrent,
    setPageSize,
    setSelectedRowKeys,
    setEditModalVisible,
    loadData: loadRoomDevices,
    handleAdd,
    handleEdit,
    resetSelection,
  } = useTableManager<RoomDevice>(fetchRoomDevices, { pageSize: 10 });

  useEffect(() => {
    loadRoomDevices();
  }, [loadRoomDevices]);

  const handleSearch = () => {
    const values = searchForm.getFieldsValue();
    const searchParams: any = {};
    Object.keys(values).forEach((key) => {
      const value = values[key];
      if (value !== undefined && value !== null && value !== '') {
        if (key === 'status') {
          searchParams[key] = Number(value);
        } else if (key === 'dateRange' && value) {
          searchParams['activationDateStart'] = value[0];
          searchParams['activationDateEnd'] = value[1];
        } else {
          searchParams[key] = value;
        }
      }
    });
    loadRoomDevices(searchParams);
  };

  const handleReset = () => {
    searchForm.resetFields();
    loadRoomDevices();
  };

  const handleRefresh = () => {
    loadRoomDevices();
  };

  const handleAddRoomDevice = () => {
    handleAdd();
    editForm.setFieldsValue({
      status: 0,
      deviceType: 'internet',
      monthlyFee: 0,
    });
  };

  const handleModalOpenChange = (open: boolean) => {
    if (open && editingItem) {
      const formValues = {
        ...editingItem,
        activationDate: editingItem.activationDate ? new Date(editingItem.activationDate) : null,
        expiryDate: editingItem.expiryDate ? new Date(editingItem.expiryDate) : null,
      };
      editForm.setFieldsValue(formValues);
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await roomDeviceApi.delete(id);
      Modal.success({
        title: '删除成功',
        content: '设备已成功删除',
      });
      loadRoomDevices();
    } catch (error) {
      console.error('删除失败:', error);
    }
  };

  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) return;

    try {
      await roomDeviceApi.batchDelete(selectedRowKeys as string[]);
      Modal.success({
        title: '批量删除成功',
        content: `已成功删除 ${selectedRowKeys.length} 条设备`,
      });
      resetSelection();
      loadRoomDevices();
    } catch (error) {
      console.error('批量删除失败:', error);
    }
  };

  const handleSave = async () => {
    try {
      const values = await editForm.validateFields();
      const url = editingItem
        ? `/ops/room-devices/${editingItem.id}/update`
        : '/ops/room-devices';

      await post(url, values);
      Modal.success({
        title: editingItem ? '更新成功' : '创建成功',
        content: editingItem ? '设备已成功更新' : '设备已成功创建',
      });
      setEditModalVisible(false);
      loadRoomDevices();
    } catch (error) {
      console.error('操作失败:', error);
    }
  };

  const columns: ColumnsType<RoomDevice> = [
    { title: '设备名称', dataIndex: 'name', key: 'name' },
    { title: '设备编码', dataIndex: 'deviceCode', key: 'deviceCode' },
    {
      title: '设备类型',
      dataIndex: 'deviceType',
      key: 'deviceType',
      render: (type) => {
        const typeMap: Record<string, { text: string; color: string }> = {
          internet: { text: '互联网设备', color: 'blue' },
          mpls: { text: '网络设备', color: 'green' },
          leased_line: { text: '租用设备', color: 'orange' },
          fiber: { text: '光纤设备', color: 'purple' },
        };
        const info = typeMap[type] || { text: type, color: 'default' };
        return <Tag color={info.color}>{info.text}</Tag>;
      },
    },
    { title: '带宽', dataIndex: 'roomName', key: 'roomName' },
    { title: '运营商', dataIndex: 'ipAddress', key: 'ipAddress' },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status) => {
        const statusMap: Record<number, { text: string; color: string }> = {
          0: { text: '正常', color: 'success' },
          1: { text: '故障', color: 'error' },
        };
        const info = statusMap[status] || { text: String(status), color: 'default' };
        return <Tag color={info.color}>{info.text}</Tag>;
      },
    },
    {
      title: '月租',
      dataIndex: 'monthlyFee',
      key: 'monthlyFee',
      render: (fee) => `¥${fee}`,
    },
    {
      title: '操作',
      key: 'action',
      render: (_, record) => (
        <Space>
          <Button
            type="link"
            size="small"
            icon={<EditOutlined />}
            onClick={() => handleEdit(record)}
          >
            编辑
          </Button>
          <Popconfirm
            title="确定要删除这条设备吗？"
            onConfirm={() => handleDelete(record.id)}
            okText="确定"
            cancelText="取消"
          >
            <Button type="link" danger size="small" icon={<DeleteOutlined />}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div>
      {/* 统计卡片 */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={6}>
          <Card>
            <Statistic
              title="总设备数"
              value={roomDevices.length}
              prefix={<DesktopOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="正常"
              value={roomDevices.filter((l) => d.status === 0).length}
              valueStyle={{ color: '#3f8600' }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="故障"
              value={roomDevices.filter((l) => d.status === 1).length}
              valueStyle={{ color: '#cf1322' }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="停用"
              value={roomDevices.filter((l) => d.status === 2).length}
              valueStyle={{ color: '#8c8c8c' }}
            />
          </Card>
        </Col>
      </Row>

      {/* 搜索表单和操作按钮 */}
      <Card style={{ marginBottom: 16 }}>
        <div
          style={{
            dipAddresslay: 'flex',
            justifyContent: 'space-between',
            flexWrap: 'wrap',
            gap: '16px',
          }}
        >
          <Form form={searchForm} layout="inline" style={{ flex: 1, minWidth: 0 }}>
            <Form.Item name="name" label="设备名称">
              <Input placeholder="请输入设备名称" />
            </Form.Item>
            <Form.Item name="deviceCode" label="设备编码">
              <Input placeholder="请输入设备编码" />
            </Form.Item>
            <Form.Item name="deviceType" label="设备类型">
              <Select placeholder="请选择类型" style={{ width: 150 }} allowClear>
                <Option value="internet">互联网设备</Option>
                <Option value="network">网络设备</Option>
                <Option value="leased_line">租用设备</Option>
                <Option value="fiber">光纤设备</Option>
              </Select>
            </Form.Item>
            <Form.Item name="ipAddress" label="运营商">
              <Input placeholder="请输入运营商" />
            </Form.Item>
            <Form.Item name="status" label="状态">
              <Select placeholder="请选择状态" style={{ width: 120 }} allowClear>
                <Option value={0}>正常</Option>
                <Option value={1}>故障</Option>
              </Select>
            </Form.Item>
            <Form.Item>
              <Space>
                <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>
                  搜索
                </Button>
                <Button onClick={handleReset}>重置</Button>
                <Button icon={<ReloadOutlined />} onClick={handleRefresh}>
                  刷新
                </Button>
              </Space>
            </Form.Item>
          </Form>
          <Space>
            {selectedRowKeys.length > 0 && (
              <Button
                danger
                icon={<DeleteOutlined />}
                onClick={handleBatchDelete}
              >
                批量删除 ({selectedRowKeys.length})
              </Button>
            )}
            <Button type="primary" icon={<PlusOutlined />} onClick={handleAddRoomDevice}>
              新增设备
            </Button>
          </Space>
        </div>
      </Card>

      {/* 表格 */}
      <Card>
        <Table
          columns={columns}
          dataSource={roomDevices}
          rowKey="id"
          loading={loading}
          pagination={{
            current,
            pageSize,
            total,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => `共 ${total} 条`,
            onChange: (page, size) => {
              setCurrent(page);
              setPageSize(size);
              loadRoomDevices({ current: page, pageSize: size });
            },
          }}
          rowSelection={{
            selectedRowKeys,
            onChange: setSelectedRowKeys,
          }}
        />
      </Card>

      {/* 编辑弹窗 */}
      <Modal
        title={editingItem ? '编辑设备' : '新增设备'}
        open={editModalVisible}
        onOk={handleSave}
        afterOpenChange={handleModalOpenChange}
        onCancel={() => setEditModalVisible(false)}
        width={800}
      >
        <Form form={editForm} layout="horizontal" labelCol={{ span: 4 }} wrapperCol={{ span: 20 }}>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="name"
                label="设备名称"
                rules={[{ required: true, message: '请输入设备名称' }]}
              >
                <Input />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="deviceCode"
                label="设备编码"
                rules={[{ required: true, message: '请输入设备编码' }]}
              >
                <Input />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="deviceType"
                label="设备类型"
                rules={[{ required: true, message: '请选择设备类型' }]}
                initialValue="internet"
              >
                <Select>
                  <Option value="internet">互联网设备</Option>
                  <Option value="network">网络设备</Option>
                  <Option value="leased_line">租用设备</Option>
                  <Option value="fiber">光纤设备</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="roomName"
                label="带宽"
                rules={[{ required: true, message: '请输入带宽' }]}
              >
                <Input placeholder="如：100M, 1G" />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="ipAddress"
                label="运营商"
                rules={[{ required: true, message: '请输入运营商' }]}
              >
                <Input placeholder="电信/移动/联通等" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="manufacturer" label="厂商">
                <Input />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="model" label="型号">
                <Input />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="serialNumber" label="序列号">
                <Input />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="assetNumber" label="资产编号">
                <Input />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="macAddress" label="MAC地址">
                <Input />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="ipAddress" label="IP地址">
                <Input />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="cpu" label="CPU">
                <Input />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="memory" label="内存">
                <Input />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="storage" label="存储">
                <Input />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="os" label="操作系统">
                <Input style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="position" label="机架位置">
                <Input style={{ width: '100%' }} />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item name="status" label="状态" initialValue={0}>
            <Select>
              <Option value={0}>正常</Option>
              <Option value={1}>故障</Option>
            </Select>
          </Form.Item>
          <Form.Item name="remark" label="备注">
            <Input.TextArea rows={3} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default RoomDeviceManagement;
