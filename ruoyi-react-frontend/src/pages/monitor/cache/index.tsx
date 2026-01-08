import React, { useState, useEffect } from 'react';
import {
  Card,
  Table,
  Button,
  Space,
  Input,
  Select,
  Modal,
  Form,
  InputNumber,
  Tag,
  Tooltip,
  Row,
  Col,
  Statistic,
  Progress,
  Tabs,
  message,
  Alert,
  List,
  Divider,
  Popconfirm,
  Typography
} from 'antd';
import {
  SearchOutlined,
  ReloadOutlined,
  DeleteOutlined,
  EyeOutlined,
  PlusOutlined,
  SettingOutlined,
  BarChartOutlined,
  ExportOutlined,
  ExclamationCircleOutlined,
  DatabaseOutlined,
  ThunderboltOutlined
} from '@ant-design/icons';
import { post, get } from '@/lib/api';
import type { ColumnsType } from 'antd/es/table';

const { TextArea } = Input;

interface CacheInfo {
  key: string;
  value: string;
  ttl: number;
  size: number;
  type: string;
  createdAt: string;
  updatedAt: string;
}

interface CacheStats {
  cacheType: string;
  hitCount: number;
  missCount: number;
  hitRate: number;
  totalMemory: number;
  usedMemory: number;
  keyCount: number;
  expiredCount: number;
  collectTime: string;
}

interface CacheMonitor {
  status: {
    connected: boolean;
    type: string;
    version: string;
    uptime: string;
  };
  stats: {
    hitCount: number;
    missCount: number;
    hitRate: number;
    totalMemory: number;
    usedMemory: number;
    keyCount: number;
  };
  distribution: { [key: string]: number };
  memoryTrend: Array<{ time: string; memory: number }>;
  hitRateTrend: Array<{ time: string; hitRate: number }>;
}

const CacheManager: React.FC = () => {
  const [caches, setCaches] = useState<CacheInfo[]>([]);
  const [stats, setStats] = useState<CacheStats[]>([]);
  const [monitor, setMonitor] = useState<CacheMonitor | null>(null);
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);
  const [current, setCurrent] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [searchForm, setSearchForm] = useState({
    key: '',
    type: ''
  });
  const [operateModalVisible, setOperateModalVisible] = useState(false);
    const [statsModalVisible, setStatsModalVisible] = useState(false);
  const [form] = Form.useForm();

  // 获取缓存列表
  const fetchCaches = async () => {
    setLoading(true);
    try {
      const result = await post('/monitor/cache/list', {
        ...searchForm,
        current,
        pageSize
      });

      setCaches(result.data.list || []);
      setTotal(result.data.total || 0);
    } catch (error) {
      console.error('获取缓存列表失败:', error);
      message.error('网络错误，请稍后重试');
    } finally {
      setLoading(false);
    }
  };

  // 获取缓存统计
  const fetchStats = async () => {
    try {
      const result = await post('/monitor/cache/stats/list', {
        current: 1,
        pageSize: 10
      });

      setStats(result.data.list || []);
    } catch (error) {
      console.error('获取缓存统计失败:', error);
      message.error('获取缓存统计失败');
    }
  };

  // 获取监控数据
  const fetchMonitor = async () => {
    try {
      const result = await post('/monitor/cache/monitor', {});
      setMonitor(result.data);
    } catch (error) {
      console.error('获取监控数据失败:', error);
      message.error('获取监控数据失败');
    }
  };

  // 操作缓存
  const handleOperate = async () => {
    try {
      const values = await form.validateFields();
      const result = await post('/monitor/cache/operate', {
        ...values,
        ttl: values.ttl || undefined
      });

      message.success('操作成功');
      setOperateModalVisible(false);
      form.resetFields();
      fetchCaches();
    } catch (error) {
      console.error('操作失败:', error);
      message.error('操作失败');
    }
  };

  // 批量操作
  const handleBatchOperate = async (operation: string, keys?: string[]) => {
    try {
      await post('/monitor/cache/batch', {
        keys: keys || [],
        operation
      });

      message.success('批量操作成功');
      setBatchModalVisible(false);
      fetchCaches();
    } catch (error) {
      console.error('批量操作失败:', error);
      message.error('批量操作失败');
    }
  };

  // 清空缓存
  const handleClearAll = () => {
    Modal.confirm({
      title: '确认清空',
      icon: <ExclamationCircleOutlined />,
      content: '确定要清空所有缓存吗？此操作不可恢复！',
      onOk: async () => {
        try {
          await post('/monitor/cache/clear', {});

          message.success('清空成功');
          fetchCaches();
          fetchMonitor();
        } catch (error) {
          console.error('清空失败:', error);
          message.error('清空失败');
        }
      }
    });
  };

  // 导出缓存
  const handleExport = async () => {
    try {
      const result = await post('/monitor/cache/export', {
        ...searchForm
      });

      // 创建下载链接
      const dataStr = JSON.stringify(result.data, null, 2);
      const dataUri = 'data:application/json;charset=utf-8,' + encodeURIComponent(dataStr);
      const exportFileDefaultName = `cache_export_${new Date().toISOString().slice(0, 19).replace(/:/g, '-')}.json`;

      const linkElement = document.createElement('a');
      linkElement.setAttribute('href', dataUri);
      linkElement.setAttribute('download', exportFileDefaultName);
      linkElement.click();

      message.success('导出成功');
    } catch (error) {
      console.error('导出失败:', error);
      message.error('导出失败');
    }
  };

  // 查看缓存详情
  const handleViewDetail = async (key: string) => {
    try {
      const result = await get(`/monitor/cache/${key}`);

      Modal.info({
        title: '缓存详情',
        width: 600,
        content: (
          <div>
            <p><strong>键名：</strong>{result.data.key}</p>
            <p><strong>值：</strong></p>
            <pre className="bg-gray-100 p-3 rounded">
              {JSON.stringify(JSON.parse(result.data.value || '{}'), null, 2)}
            </pre>
            <p><strong>类型：</strong>{result.data.type}</p>
            <p><strong>大小：</strong>{(result.data.size || 0).toLocaleString()} 字节</p>
            <p><strong>TTL：</strong>{formatTTL(result.data.ttl || -1)}</p>
            <p><strong>创建时间：</strong>{result.data.createdAt ? new Date(result.data.createdAt).toLocaleString() : '-'}</p>
          </div>
        )
      });
    } catch (error) {
      console.error('获取缓存详情失败:', error);
      message.error('获取缓存详情失败');
    }
  };

  // 搜索
  const handleSearch = () => {
    setCurrent(1);
    fetchCaches();
  };

  // 重置
  const handleReset = () => {
    setSearchForm({
      key: '',
      type: ''
    });
    setCurrent(1);
  };

  // 刷新
  const handleRefresh = () => {
    fetchCaches();
    fetchMonitor();
  };

  // 分页变化
  const handleTableChange = (pagination: any) => {
    setCurrent(pagination.current);
    setPageSize(pagination.pageSize);
  };

  useEffect(() => {
    fetchCaches();
    fetchStats();
    fetchMonitor();

    // 设置定时刷新（每30秒）
    const interval = setInterval(() => {
      fetchMonitor();
    }, 30000);

    return () => clearInterval(interval);
  }, []);

  // 格式化内存大小
  const formatMemorySize = (bytes: number): string => {
    const units = ['B', 'KB', 'MB', 'GB', 'TB'];
    let size = bytes;
    let unitIndex = 0;

    while (size >= 1024 && unitIndex < units.length - 1) {
      size /= 1024;
      unitIndex++;
    }

    return `${size.toFixed(2)} ${units[unitIndex]}`;
  };

  // 格式化TTL时间
  const formatTTL = (seconds: number): string => {
    if (seconds <= 0) {
      return '永久';
    }

    const units = [
      { name: '秒', value: 60 },
      { name: '分钟', value: 60 },
      { name: '小时', value: 24 },
      { name: '天', value: Infinity }
    ];

    let value = seconds;
    let unitIndex = 0;

    while (unitIndex < units.length - 1 && value >= units[unitIndex].value) {
      value /= units[unitIndex].value;
      unitIndex++;
    }

    // 如果小于1，显示为更小的单位
    if (value < 1 && unitIndex > 0) {
      value *= units[unitIndex - 1].value;
      unitIndex--;
    }

    // 格式化：如果整数就不显示小数，否则最多显示2位小数
    const formatted = Number.isInteger(value) ? value.toString() : value.toFixed(2);

    return `${formatted}${units[unitIndex].name}`;
  };

  // 缓存类型选项
  const typeOptions = [
    { label: '字符串', value: 'string' },
    { label: '列表', value: 'list' },
    {label: '哈希', value: 'hash' },
    { label: '集合', value: 'set' },
    { label: '有序集合', value: 'zset' },
    { label: '其他', value: 'other' }
  ];

  // 操作选项
  const operationOptions = [
    { label: '获取值', value: 'get' },
    { label: '设置值', value: 'set' },
    { label: '删除键', value: 'del' },
    { label: '检查存在', value: 'exists' },
    { label: '设置过期时间', value: 'expire' },
    { label: '获取TTL', value: 'ttl' }
  ];

  const columns: ColumnsType<CacheInfo> = [
    {
      title: '缓存键',
      dataIndex: 'key',
      key: 'key',
      width: 300,
      ellipsis: true,
      render: (text: string) => (
        <code className="bg-gray-100 px-2 py-1 rounded text-xs">{text}</code>
      ),
    },
    {
      title: '缓存类型',
      dataIndex: 'type',
      key: 'type',
      width: 100,
      render: (type: string) => (
        <Tag color="blue">{type}</Tag>
      ),
    },
    {
      title: 'TTL',
      dataIndex: 'ttl',
      key: 'ttl',
      width: 120,
      render: (ttl: number) => (
        <Tag color={ttl > 0 ? 'green' : 'default'}>
          {formatTTL(ttl)}
        </Tag>
      ),
    },
    {
      title: '大小',
      dataIndex: 'size',
      key: 'size',
      width: 100,
      render: (size: number) => formatMemorySize(size),
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 180,
      render: (time: string) => time ? new Date(time).toLocaleString() : '-',
    },
    {
      title: '更新时间',
      dataIndex: 'updatedAt',
      key: 'updatedAt',
      width: 180,
      render: (time: string) => time ? new Date(time).toLocaleString() : '-',
    },
    {
      title: '操作',
      key: 'action',
      width: 200,
      render: (_, record: CacheInfo) => (
        <Space size="middle">
          <Tooltip title="查看详情">
            <Button
              type="link"
              icon={<EyeOutlined />}
              onClick={() => handleViewDetail(record.key)}
            />
          </Tooltip>
          <Tooltip title="删除">
            <Popconfirm
              title="确定要删除这个缓存吗？"
              onConfirm={() => handleBatchOperate('del', [record.key])}
            >
              <Button
                type="link"
                danger
                icon={<DeleteOutlined />}
              />
            </Popconfirm>
          </Tooltip>
        </Space>
      ),
    },
  ];

  return (
    <div className="p-6">
      <div className="mb-6">
        <h1 className="text-2xl font-bold mb-4">缓存管理</h1>

        {/* 监控概览 */}
        <Row gutter={16} className="mb-6">
          <Col xs={24} sm={12} md={6}>
            <Card>
              <Statistic
                title="总键值对"
                value={monitor?.stats?.keyCount || 0}
                prefix={<DatabaseOutlined />}
              />
            </Card>
          </Col>
          <Col xs={24} sm={12} md={6}>
            <Card>
              <Statistic
                title="命中率"
                value={monitor?.stats?.hitRate || 0}
                precision={1}
                suffix="%"
                styles={{ content: { color: '#3f8600' } }}
              />
            </Card>
          </Col>
          <Col xs={24} sm={12} md={6}>
            <Card>
              <Statistic
                title="内存使用"
                value={monitor?.stats?.usedMemory || 0}
                formatter={(value) => formatMemorySize(Number(value))}
                suffix={`/ ${formatMemorySize(monitor?.stats?.totalMemory || 0)}`}
              />
            </Card>
          </Col>
          <Col xs={24} sm={12} md={6}>
            <Card>
              <Statistic
                title="命中次数"
                value={monitor?.stats?.hitCount || 0}
                prefix={<ThunderboltOutlined />}
              />
            </Card>
          </Col>
        </Row>

        {/* 连接状态 */}
        <Alert
          title={
            <span>
              缓存服务状态：{' '}
              <Tag color={monitor?.status?.connected ? 'success' : 'error'}>
                {monitor?.status?.connected ? '已连接' : '未连接'}
              </Tag>
              {monitor?.status?.type && (
                <span className="ml-2">
                  类型：{monitor?.status?.type} | 版本：{monitor?.status?.version}
                </span>
              )}
            </span>
          }
          type={monitor?.status?.connected ? 'success' : 'error'}
          showIcon
          className="mb-4"
        />

        {/* 搜索表单 */}
        <Card className="mb-4">
          <Row gutter={16}>
            <Col xs={24} sm={8} md={6}>
              <Input
                placeholder="缓存键"
                value={searchForm.key}
                onChange={(e) => setSearchForm({ ...searchForm, key: e.target.value })}
                allowClear
                prefix={<SearchOutlined />}
              />
            </Col>
            <Col xs={24} sm={8} md={6}>
              <Select
                placeholder="缓存类型"
                value={searchForm.type}
                onChange={(value) => setSearchForm({ ...searchForm, type: value })}
                allowClear
                style={{ width: '100%' }}
                options={typeOptions}
              />
            </Col>
            <Col xs={24} sm={8} md={12}>
              <Space>
                <Button
                  type="primary"
                  icon={<PlusOutlined />}
                  onClick={() => setOperateModalVisible(true)}
                >
                  操作缓存
                </Button>
                <Button icon={<SearchOutlined />} onClick={handleSearch}>
                  搜索
                </Button>
                <Button onClick={handleReset}>重置</Button>
                <Button icon={<ReloadOutlined />} onClick={handleRefresh}>
                  刷新
                </Button>
                <Button icon={<SettingOutlined />} onClick={() => setStatsModalVisible(true)}>
                  统计信息
                </Button>
                <Button icon={<ExportOutlined />} onClick={handleExport}>
                  导出
                </Button>
                <Button danger icon={<DeleteOutlined />} onClick={handleClearAll}>
                  清空
                </Button>
              </Space>
            </Col>
          </Row>
        </Card>
      </div>

      <Tabs
        defaultActiveKey="list"
        items={[
          {
            key: 'list',
            label: (
              <span>
                <DatabaseOutlined />
                缓存列表
              </span>
            ),
            children: (
              <Table
                columns={columns}
                dataSource={caches}
                rowKey="key"
                loading={loading}
                pagination={{
                  current,
                  pageSize,
                  total,
                  showSizeChanger: true,
                  showQuickJumper: true,
                  showTotal: (total) => `共 ${total} 条记录`,
                  onChange: handleTableChange,
                }}
              />
            ),
          },
          {
            key: 'stats',
            label: (
              <span>
                <BarChartOutlined />
                缓存统计
              </span>
            ),
            children: (
              <div>
                <Row gutter={16} className="mb-4">
                  <Col span={24}>
                    <Card title="内存使用情况">
                      <Progress
                        percent={monitor ? Math.round((monitor.stats.usedMemory / monitor.stats.totalMemory) * 100) : 0}
                        status={monitor?.stats?.usedMemory && monitor.stats.usedMemory / monitor.stats.totalMemory > 0.8 ? 'exception' : 'normal'}
                        format={() => `${formatMemorySize(monitor?.stats?.usedMemory || 0)}`}
                      />
                    </Card>
                  </Col>
                </Row>

                <Row gutter={16}>
                  <Col xs={24} md={12}>
                    <Card title="命中率趋势">
                      <List
                        dataSource={monitor?.hitRateTrend || []}
                        renderItem={(item) => (
                          <List.Item>
                            <span>{item.time}</span>
                            <Tag color="blue">{item.hitRate}%</Tag>
                          </List.Item>
                        )}
                      />
                    </Card>
                  </Col>
                  <Col xs={24} md={12}>
                    <Card title="内存使用趋势">
                      <List
                        dataSource={monitor?.memoryTrend || []}
                        renderItem={(item) => (
                          <List.Item>
                            <span>{item.time}</span>
                            <Tag color="green">{formatMemorySize(item.memory)}</Tag>
                          </List.Item>
                        )}
                      />
                    </Card>
                  </Col>
                </Row>
              </div>
            ),
          },
        ]}
      />

      {/* 操作缓存模态框 */}
      <Modal
        title="缓存操作"
        open={operateModalVisible}
        onCancel={() => setOperateModalVisible(false)}
        onOk={handleOperate}
        width={500}
      >
        <Form form={form} layout="vertical">
          <Form.Item
            label="缓存键"
            name="key"
            rules={[{ required: true, message: '请输入缓存键' }]}
          >
            <Input placeholder="请输入缓存键" />
          </Form.Item>
          <Form.Item
            label="操作类型"
            name="operation"
            rules={[{ required: true, message: '请选择操作类型' }]}
          >
            <Select options={operationOptions} />
          </Form.Item>
          <Form.Item
            label="缓存值"
            name="value"
            dependencies={['operation']}
            rules={[
              {
                required: true,
                message: '请输入缓存值',
              },
              ({ getFieldValue }) => ({
                validator: (_, value) => {
                  if (getFieldValue('operation') === 'set' && !value) {
                  return Promise.reject(new Error('请输入缓存值'));
                }
                  return Promise.resolve();
                },
              }),
            ]}
          >
            <TextArea
              placeholder="请输入缓存值"
              rows={4}
            />
          </Form.Item>
          <Form.Item
            label="过期时间(秒)"
            name="ttl"
            dependencies={['operation']}
          >
            <InputNumber
              placeholder="留空表示永不过期"
              min={1}
              style={{ width: '100%' }}
            />
          </Form.Item>
        </Form>
      </Modal>

      {/* 统计信息模态框 */}
      <Modal
        title="缓存统计信息"
        open={statsModalVisible}
        onCancel={() => setStatsModalVisible(false)}
        footer={null}
        width={800}
      >
        <Row gutter={16}>
          {stats.map((stat, index) => (
            <Col key={index} span={8}>
              <Card size="small" title={stat.cacheType}>
                <Statistic
                  title="命中次数"
                  value={stat.hitCount}
                  styles={{ content: { color: '#3f8600' } }}
                />
                <Divider />
                <Statistic
                  title="未命中次数"
                  value={stat.missCount}
                  styles={{ content: { color: '#cf1322' } }}
                />
                <Divider />
                <Statistic
                  title="命中率"
                  value={stat.hitRate}
                  precision={1}
                  suffix="%"
                  styles={{ content: { color: '#1890ff' } }}
                />
                <Divider />
                <Statistic
                  title="总内存"
                  value={formatMemorySize(stat.totalMemory)}
                />
                <Divider />
                <Statistic
                  title="已用内存"
                  value={formatMemorySize(stat.usedMemory)}
                />
                <Divider />
                <Statistic
                  title="键值对数量"
                  value={stat.keyCount}
                />
                <Divider />
                <Statistic
                  title="过期键数量"
                  value={stat.expiredCount}
                />
                <Divider />
                <div className="text-sm text-gray-500">
                  采集时间：{new Date(stat.collectTime).toLocaleString()}
                </div>
              </Card>
            </Col>
          ))}
        </Row>
      </Modal>
    </div>
  );
};

export default CacheManager;