import type { FC } from 'react';
import {
  Table,
  Button,
  Space,
  Form,
  Input,
  Select,
  Tag,
  Card,
  Row,
  Col,
  Statistic,
  message,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  SearchOutlined,
  ReloadOutlined,
  DeleteOutlined,
  CloudSyncOutlined,
  ApiOutlined,
  CheckCircleOutlined,
  WarningOutlined,
  ArrowLeftOutlined,
} from '@ant-design/icons';
import type { DevicePortStatus, NetworkDevice } from '@/types';
import { post, get } from '@/lib/api';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { useEffect } from 'react';
import { useTableManager } from '@/hooks/useTableManager';
import { withErrorHandling } from '@/utils/errorHandler';
import { useState } from 'react';
import { formatDateTime } from '@/utils/datetime';
import { usePagination } from '@/hooks/usePagination';

const { Option } = Select;

const PortStatusPage: FC = () => {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const deviceIdFromUrl = searchParams.get('deviceId') || '';
  const isFromDevice = !!deviceIdFromUrl;

  const [devices, setDevices] = useState<NetworkDevice[]>([]);
  const [collecting, setCollecting] = useState(false);

  // 统计数据
  const [statistics, setStatistics] = useState({
    total: 0,
    up: 0,
    down: 0,
    dot1xEnabled: 0,
  });

  // 使用全局分页 hook
  const { paginationProps, setCurrent, setPageSize, setTotal } = usePagination();

  const {
    loading,
    data: portStatus,
    selectedRowKeys,
    setSelectedRowKeys,
    searchForm,
    loadData: loadPortStatus,
    handleSearch,
    handleReset,
    handleRefresh,
  } = useTableManager<DevicePortStatus>(
    async (params) => {
      const formValues = searchForm.getFieldsValue();
      const values = formValues as Record<string, unknown>;
      const result = await post('/network/ports/list', {
        current: params.current ?? paginationProps.current ?? 1,
        pageSize: params.pageSize ?? paginationProps.pageSize ?? 10,
        ...values,
        // 如果从设备页面跳转过来，强制使用 URL 中的 deviceId
        ...(isFromDevice ? { deviceId: deviceIdFromUrl } : {}),
      }) as { data: { list: DevicePortStatus[]; total: number } };
      // 更新统计数据
      updateStatistics(result.data.list || []);
      return result.data;
    },
    {
      externalPagination: {
        current: paginationProps.current ?? 1,
        pageSize: paginationProps.pageSize ?? 10,
        setCurrent,
        setPageSize,
        setTotal,
      },
    }
  );

  // 加载统计数据
  const loadStatistics = async () => {
    try {
      const result = await get('/network/ports/statistics') as { data?: { totalRecords?: number; upPortsCount?: number; downPortsCount?: number; dot1xEnabledCount?: number } };
      const stats = result.data || {};
      setStatistics({
        total: stats.totalRecords || 0,
        up: stats.upPortsCount || 0,
        down: stats.downPortsCount || 0,
        dot1xEnabled: stats.dot1xEnabledCount || 0,
      });
    } catch (error) {
      console.error('加载统计数据失败:', error);
    }
  };

  // 更新统计数据
  const updateStatistics = (portList: DevicePortStatus[]) => {
    const upCount = portList.filter(p => p.operStatus === 'up').length;
    const downCount = portList.filter(p => p.operStatus === 'down').length;
    const dot1xCount = portList.filter(p => p.dot1xEnabled).length;
    setStatistics({
      total: portList.length,
      up: upCount,
      down: downCount,
      dot1xEnabled: dot1xCount,
    });
  };

  // 加载设备列表
  const loadDevices = async () => {
    try {
      const result = await post('/network/devices/list', {
        current: 1,
        pageSize: 1000,
      }) as { data?: { list: NetworkDevice[] } };
      setDevices(result.data?.list || []);
    } catch (error) {
      console.error('加载设备列表失败:', error);
    }
  };

  useEffect(() => {
    // 如果从设备页面跳转过来，初始化表单的 deviceId
    if (isFromDevice && deviceIdFromUrl) {
      searchForm.setFieldsValue({ deviceId: deviceIdFromUrl });
    }
    loadStatistics();
    loadDevices();
    loadPortStatus();
  }, []);

  // 采集所有设备端口状态
  const handleCollectAll = async () => {
    setCollecting(true);
    await withErrorHandling(
      async () => {
        await post('/network/ports/collect-all', {});
        return '采集任务已创建';
      },
      {
        onSuccess: () => {
          loadPortStatus();
          loadStatistics();
        },
        onError: () => {
          setCollecting(false);
        },
      }
    );
    setCollecting(false);
  };

  // 批量删除
  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) {
      message.warning('请选择要删除的数据');
      return;
    }
    await withErrorHandling(
      async () => {
        await post('/network/ports/batch-delete', { ids: selectedRowKeys });
        return '批量删除成功';
      },
      {
        onSuccess: () => {
          setSelectedRowKeys([]);
          loadPortStatus();
          loadStatistics();
        },
      }
    );
  };

  // 表格列
  const columns: ColumnsType<DevicePortStatus> = [
    { title: '接口名称', dataIndex: 'interfaceName', key: 'interfaceName', width: 150 },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      width: 180,
      ellipsis: true,
      render: (desc: string) => desc || '-',
    },
    {
      title: '端口状态',
      dataIndex: 'operStatus',
      key: 'operStatus',
      width: 100,
      render: (status: string) => {
        const color = status === 'up' ? 'success' : 'default';
        return <Tag color={color}>{status?.toUpperCase() || '-'}</Tag>;
      },
    },
    {
      title: '管理员激活',
      dataIndex: 'adminStatus',
      key: 'adminStatus',
      width: 100,
      render: (status: string) => {
        const color = status === 'up' ? 'processing' : 'default';
        return <Tag color={color}>{status?.toUpperCase() || '-'}</Tag>;
      },
    },
    {
      title: 'VLAN',
      dataIndex: 'vlan',
      key: 'vlan',
      width: 80,
      render: (vlan?: number) => vlan ?? '-',
    },
    {
      title: '双工',
      dataIndex: 'duplex',
      key: 'duplex',
      width: 80,
      render: (duplex?: string) => duplex?.toUpperCase() || '-',
    },
    {
      title: '速率',
      dataIndex: 'speed',
      key: 'speed',
      width: 80,
      render: (speed?: string) => speed || '-',
    },
    {
      title: '类型',
      dataIndex: 'portType',
      key: 'portType',
      width: 80,
      render: (type?: string) => type || '-',
    },
    {
      title: '802.1X',
      key: 'dot1x',
      width: 100,
      render: (_, record) => (
        <Space size="small">
          <Tag color={record.dot1xEnabled ? 'blue' : 'default'}>
            {record.dot1xEnabled ? '启用' : '未启用'}
          </Tag>
          {record.dot1xEnabled && (
            <Tag color={record.dot1xPortStatus === 'authorized' ? 'success' : 'warning'}>
              {record.dot1xPortStatus}
            </Tag>
          )}
        </Space>
      ),
    },
    {
      title: '端口安全',
      key: 'portSecurity',
      width: 120,
      render: (_, record) => (
        <Space size="small">
          <Tag color={record.portSecurityEnabled ? 'purple' : 'default'}>
            {record.portSecurityEnabled ? '启用' : '未启用'}
          </Tag>
          {record.portSecurityEnabled && record.portSecurityMode && (
            <Tag>{record.portSecurityMode}</Tag>
          )}
        </Space>
      ),
    },
    {
      title: 'MAC数',
      key: 'macCount',
      width: 100,
      render: (_, record) => (
        <span>
          {record.currentMACCount || 0} / {record.maxMACCount || '-'}
        </span>
      ),
    },
    {
      title: '采集时间',
      dataIndex: 'collectedAt',
      key: 'collectedAt',
      width: 180,
      render: (date: string) => date ? formatDateTime(date) : '-',
    },
  ];

  return (
    <div>
      {/* 返回按钮（仅当从设备页面跳转时显示） */}
      {isFromDevice && (
        <Button
          icon={<ArrowLeftOutlined />}
          onClick={() => navigate('/network/devices')}
          style={{ marginBottom: 16 }}
        >
          返回设备管理
        </Button>
      )}

      {/* 统计卡片 */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={6}>
          <Card>
            <Statistic
              title="端口总数"
              value={statistics.total}
              prefix={<ApiOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="UP端口"
              value={statistics.up}
              styles={{ content: { color: '#3f8600' } }}
              prefix={<CheckCircleOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="DOWN端口"
              value={statistics.down}
              styles={{ content: { color: '#cf1322' } }}
              prefix={<WarningOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="802.1X启用"
              value={statistics.dot1xEnabled}
              styles={{ content: { color: '#1890ff' } }}
              prefix={<ApiOutlined />}
            />
          </Card>
        </Col>
      </Row>

      {/* 搜索表单和操作按钮 */}
      <Card style={{ marginBottom: 16 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', flexWrap: 'wrap', gap: '16px' }}>
          <Form form={searchForm} layout="inline" style={{ flex: 1, minWidth: 0 }}>
            <Form.Item name="deviceId" label="设备">
              <Select placeholder="请选择设备" allowClear className="user-form-input" style={{ width: 200 }} showSearch optionFilterProp="label">
                {devices.map(device => (
                  <Option key={device.id} value={device.id} label={device.deviceName}>{device.deviceName}</Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item name="interfaceName" label="接口">
              <Input placeholder="请输入接口名称" allowClear className="user-form-input" style={{ width: 150 }} />
            </Form.Item>
            <Form.Item name="adminStatus" label="管理状态">
              <Select placeholder="请选择状态" allowClear className="user-form-input" style={{ width: 120 }}>
                <Option value="up">UP</Option>
                <Option value="down">DOWN</Option>
              </Select>
            </Form.Item>
            <Form.Item name="operStatus" label="操作状态">
              <Select placeholder="请选择状态" allowClear className="user-form-input" style={{ width: 120 }}>
                <Option value="up">UP</Option>
                <Option value="down">DOWN</Option>
              </Select>
            </Form.Item>
            <Form.Item name="dot1xEnabled" label="802.1X状态">
              <Select placeholder="请选择" allowClear className="user-form-input" style={{ width: 120 }}>
                <Option value={true}>已启用</Option>
                <Option value={false}>未启用</Option>
              </Select>
            </Form.Item>
            <Form.Item name="portSecurityEnabled" label="端口安全">
              <Select placeholder="请选择" allowClear className="user-form-input" style={{ width: 120 }}>
                <Option value={true}>已启用</Option>
                <Option value={false}>未启用</Option>
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
            <Button icon={<CloudSyncOutlined />} onClick={handleCollectAll} loading={collecting}>
              采集所有设备
            </Button>
            <Button icon={<DeleteOutlined />} onClick={handleBatchDelete} disabled={selectedRowKeys.length === 0} style={{ color: '#ff4d4f' }}>
              批量删除 ({selectedRowKeys.length})
            </Button>
          </Space>
        </div>
      </Card>

      {/* 端口状态表格 */}
      <Card>
        <Table
          rowSelection={{
            selectedRowKeys,
            onChange: setSelectedRowKeys,
          }}
          columns={columns}
          dataSource={portStatus}
          loading={loading}
          rowKey="id"
          scroll={{ x: 1400 }}
          pagination={paginationProps}
          onChange={(pagination) => {
            const newPage = pagination.current ?? 1;
            const newPageSize = pagination.pageSize ?? 10;
            setCurrent(newPage);
            setPageSize(newPageSize);
            loadPortStatus({
              current: newPage,
              pageSize: newPageSize,
            });
          }}
          expandable={{
            expandedRowRender: (record) => (
              <div style={{ padding: 16, background: '#fafafa' }}>
                <p><strong>接口详情:</strong></p>
                <p>管理状态: {record.adminStatus?.toUpperCase() || '-'}</p>
                <p>操作状态: {record.operStatus?.toUpperCase() || '-'}</p>
                {record.dot1xEnabled && (
                  <>
                    <p>802.1X状态: {record.dot1xPortStatus}</p>
                  </>
                )}
                {record.portSecurityEnabled && (
                  <>
                    <p>安全模式: {record.portSecurityMode || '未设置'}</p>
                    <p>最大MAC数: {record.maxMACCount || '未限制'}</p>
                    <p>当前MAC数: {record.currentMACCount || 0}</p>
                  </>
                )}
              </div>
            ),
          }}
        />
      </Card>
    </div>
  );
};

export default PortStatusPage;
