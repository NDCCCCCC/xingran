import React, { useState, useEffect } from 'react';
import {
  Table,
  Button,
  Space,
  Form,
  Input,
  InputNumber,
  Select,
  message,
  Card,
  Row,
  Col,
  Statistic,
  Tag,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  SearchOutlined,
  ReloadOutlined,
  DeleteOutlined,
  CloudServerOutlined,
  AppleOutlined,
} from '@ant-design/icons';
import type { DeviceMACAddress, NetworkDevice } from '@/types';
import { post } from '@/lib/api';

const { Option } = Select;

const MACAddressPage: React.FC = () => {
  const [macAddresses, setMacAddresses] = useState<DeviceMACAddress[]>([]);
  const [devices, setDevices] = useState<NetworkDevice[]>([]);
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);
  const [current, setCurrent] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [searchForm] = Form.useForm();
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);

  // 统计数据
  const [statistics, setStatistics] = useState({
    total: 0,
    dynamic: 0,
    static: 0,
    secure: 0,
  });

  // 加载MAC地址列表
  const loadMACAddresses = async (params: any = {}) => {
    setLoading(true);
    try {
      const values = searchForm.getFieldsValue();
      const result = await post('/network/mac/list', {
        current: params.current || current,
        pageSize: params.pageSize || pageSize,
        ...values,
      });
      setMacAddresses(result.data.list);
      setTotal(result.data.total);
      // 更新统计数据
      updateStatistics(result.data.list || []);
    } catch (error) {
      console.error('加载MAC地址列表失败:', error);
    } finally {
      setLoading(false);
    }
  };

  // 加载统计数据
  const loadStatistics = async () => {
    try {
      const result = await post('/network/mac/list', { current: 1, pageSize: 10000 });
      const allMACs = result.data.list || [];
      updateStatistics(allMACs);
    } catch (error) {
      console.error('加载统计数据失败:', error);
    }
  };

  // 更新统计数据
  const updateStatistics = (macList: DeviceMACAddress[]) => {
    const dynamicCount = macList.filter(m => m.macType === 'dynamic').length;
    const staticCount = macList.filter(m => m.macType === 'static').length;
    const secureCount = macList.filter(m => m.macType === 'secure').length;
    setStatistics({
      total: macList.length,
      dynamic: dynamicCount,
      static: staticCount,
      secure: secureCount,
    });
  };

  // 加载设备列表
  const loadDevices = async () => {
    try {
      const result = await post('/network/devices/list', {
        current: 1,
        pageSize: 1000,
      });
      setDevices(result.data.list || []);
    } catch (error) {
      console.error('加载设备列表失败:', error);
    }
  };

  // 批量删除
  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) {
      message.warning('请选择要删除的数据');
      return;
    }
    try {
      await post('/network/mac/batch-delete', { ids: selectedRowKeys });
      message.success('批量删除成功');
      setSelectedRowKeys([]);
      loadMACAddresses();
      loadStatistics();
    } catch (error) {
      message.error('批量删除失败');
    }
  };

  // 搜索
  const handleSearch = () => {
    loadMACAddresses();
  };

  // 重置
  const handleReset = () => {
    searchForm.resetFields();
    loadMACAddresses();
  };

  // 刷新
  const handleRefresh = () => {
    loadMACAddresses();
    loadStatistics();
  };

  // MAC类型选项
  const macTypeOptions = [
    { label: '动态', value: 'dynamic' },
    { label: '静态', value: 'static' },
    { label: '安全', value: 'secure' },
  ];

  // 表格列
  const columns: ColumnsType<DeviceMACAddress> = [
    { title: '设备名称', dataIndex: 'deviceName', key: 'deviceName', width: 150 },
    { title: 'MAC地址', dataIndex: 'macAddress', key: 'macAddress', width: 160 },
    { title: '接口', dataIndex: 'interfaceName', key: 'interfaceName', width: 150 },
    {
      title: 'VLAN ID',
      dataIndex: 'vlanId',
      key: 'vlanId',
      width: 100,
      render: (vlanId: number) => vlanId || '-',
    },
    {
      title: 'MAC类型',
      dataIndex: 'macType',
      key: 'macType',
      width: 100,
      render: (macType: string) => {
        const option = macTypeOptions.find(o => o.value === macType);
        const color = macType === 'dynamic' ? 'blue' : macType === 'static' ? 'green' : 'orange';
        return <Tag color={color}>{option?.label || macType}</Tag>;
      },
    },
    { title: '采集时间', dataIndex: 'collectedAt', key: 'collectedAt', width: 180 },
  ];

  useEffect(() => {
    loadMACAddresses();
    loadStatistics();
    loadDevices();
  }, [current, pageSize]);

  return (
    <div>
      {/* 统计卡片 */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={6}>
          <Card>
            <Statistic
              title="MAC地址总数"
              value={statistics.total}
              prefix={<AppleOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="动态MAC"
              value={statistics.dynamic}
              styles={{ content: { color: '#1890ff' } }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="静态MAC"
              value={statistics.static}
              styles={{ content: { color: '#52c41a' } }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="安全MAC"
              value={statistics.secure}
              styles={{ content: { color: '#faad14' } }}
            />
          </Card>
        </Col>
      </Row>

      {/* 搜索表单和操作按钮 */}
      <Card style={{ marginBottom: 16 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', flexWrap: 'wrap', gap: '16px' }}>
          <Form form={searchForm} layout="inline" style={{ flex: 1, minWidth: 0 }}>
            <Form.Item name="deviceId" label="设备">
              <Select placeholder="请选择设备" allowClear style={{ width: 200 }} showSearch optionFilterProp="deviceName">
                {devices.map(device => (
                  <Option key={device.id} value={device.id}>{device.deviceName}</Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item name="macAddress" label="MAC地址">
              <Input placeholder="请输入MAC地址" allowClear style={{ width: 180 }} />
            </Form.Item>
            <Form.Item name="interfaceName" label="接口">
              <Input placeholder="请输入接口名称" allowClear style={{ width: 150 }} />
            </Form.Item>
            <Form.Item name="vlanId" label="VLAN ID">
              <InputNumber placeholder="VLAN ID" style={{ width: 120 }} />
            </Form.Item>
            <Form.Item name="macType" label="MAC类型">
              <Select placeholder="请选择类型" allowClear style={{ width: 120 }}>
                {macTypeOptions.map(opt => (
                  <Option key={opt.value} value={opt.value}>{opt.label}</Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item>
              <Space>
                <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>查询</Button>
                <Button icon={<ReloadOutlined />} onClick={handleReset}>重置</Button>
                <Button icon={<ReloadOutlined />} onClick={handleRefresh}>刷新</Button>
              </Space>
            </Form.Item>
          </Form>
          <Space>
            <Button danger icon={<DeleteOutlined />} onClick={handleBatchDelete} disabled={selectedRowKeys.length === 0}>
              批量删除 ({selectedRowKeys.length})
            </Button>
          </Space>
        </div>
      </Card>

      {/* MAC地址表格 */}
      <Card>
        <Table
          rowSelection={{
            selectedRowKeys,
            onChange: setSelectedRowKeys,
          }}
          columns={columns}
          dataSource={macAddresses}
          loading={loading}
          rowKey="id"
          scroll={{ x: 1200 }}
          pagination={{
            current,
            pageSize,
            total,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => `共 ${total} 条`,
            onChange: (page, size) => { setCurrent(page); setPageSize(size || 10); },
          }}
        />
      </Card>
    </div>
  );
};

export default MACAddressPage;
