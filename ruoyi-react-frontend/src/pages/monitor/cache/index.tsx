import { useState, useEffect } from 'react';
import type { FC } from 'react';
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
  Tabs,
  message,
  Divider,
  Popconfirm
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
  DatabaseOutlined
} from '@ant-design/icons';
import { post, get } from '@/lib/api';
import type { ColumnsType } from 'antd/es/table';
import ActionButtons from '@/components/shared/ActionButtons';

const { TextArea } = Input;

interface CacheInfo {
  key: string;
  value: string;
  ttl: number;
  size: number;
  type: string;
  location: string;  // 新增: "l1", "l2", "both"
  createdAt: string;
  updatedAt: string;
}

interface CacheMonitor {
  l1: {
    status: {
      connected: boolean;
      type: string;
    };
    stats: {
      keyCount: number;
      usedMemory: number;
      hitRate: number;
      hitCount: number;
      missCount: number;
    };
  };
  l2: {
    status: {
      connected: boolean;
      type: string;
      version?: string;
      uptime?: string;
    };
    stats: {
      keyCount: number;
      usedMemory: number;
      hitRate: number;
      hitCount: number;
      missCount: number;
    };
  };
}

const CacheManager: FC = () => {
  const [caches, setCaches] = useState<CacheInfo[]>([]);
  const [monitor, setMonitor] = useState<CacheMonitor | null>(null);
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);
  const [current, setCurrent] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [searchForm, setSearchForm] = useState({
    key: '',
    type: '',
    level: 'all'  // 新增: 缓存层级筛选 "all", "l1", "l2"
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
      await post('/monitor/cache/operate', {
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
      type: '',
      level: 'all'
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
      title: '层级',
      dataIndex: 'location',
      key: 'location',
      width: 120,
      render: (location: string) => {
        if (location === 'l1') {
          return <Tag color="blue">L1(内存)</Tag>;
        } else if (location === 'l2') {
          return <Tag color="green">L2(Redis)</Tag>;
        } else if (location === 'both') {
          return <Tag color="purple">L1+L2</Tag>;
        }
        return <Tag>{location}</Tag>;
      },
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
      render: (_, record: CacheInfo) => {
        const actions = [
          {
            key: 'view',
            label: '查看详情',
            icon: <EyeOutlined />,
            onClick: () => handleViewDetail(record.key),
          },
          {
            key: 'delete',
            label: '删除',
            icon: <DeleteOutlined />,
            danger: true,
            render: () => (
              <Popconfirm
                title="确定要删除这个缓存吗？"
                onConfirm={() => handleBatchOperate('del', [record.key])}
              >
                <Button type="link" danger icon={<DeleteOutlined />} size="small">
                  删除
                </Button>
              </Popconfirm>
            ),
          },
        ];
        return <ActionButtons actions={actions} />;
      },
    },
  ];

  return (
    <div className="p-6">
      <div className="mb-6">
        <h1 className="text-2xl font-bold mb-4">缓存管理</h1>

        {/* 监控概览 */}
        <Row gutter={16} className="mb-6">
          {/* L1 内存缓存 */}
          <Col xs={24} sm={12} md={6}>
            <Card size="small" title="L1 内存缓存" extra={<Tag color="blue">L1</Tag>}>
              <Statistic
                title="键数量"
                value={monitor?.l1?.stats?.keyCount || 0}
                prefix={<DatabaseOutlined />}
                valueStyle={{ fontSize: '20px' }}
              />
              <div className="mt-2">
                <small className="text-gray-500">
                  内存: {formatMemorySize(monitor?.l1?.stats?.usedMemory || 0)}
                </small>
              </div>
            </Card>
          </Col>

          {/* L2 Redis缓存 */}
          <Col xs={24} sm={12} md={6}>
            <Card size="small" title="L2 Redis缓存" extra={<Tag color="green">L2</Tag>}>
              <Statistic
                title="键数量"
                value={monitor?.l2?.stats?.keyCount || 0}
                prefix={<DatabaseOutlined />}
                valueStyle={{ fontSize: '20px' }}
              />
              <div className="mt-2">
                <small className="text-gray-500">
                  内存: {formatMemorySize(monitor?.l2?.stats?.usedMemory || 0)}
                </small>
              </div>
            </Card>
          </Col>

          {/* 综合命中率 */}
          <Col xs={24} sm={12} md={6}>
            <Card size="small" title="综合命中率">
              <Statistic
                title="L1命中率"
                value={monitor?.l1?.stats?.hitRate || 0}
                precision={1}
                suffix="%"
                valueStyle={{ fontSize: '20px', color: '#3f8600' }}
              />
              <Divider className="my-2" />
              <Statistic
                title="L2命中率"
                value={monitor?.l2?.stats?.hitRate || 0}
                precision={1}
                suffix="%"
                valueStyle={{ fontSize: '16px', color: '#1890ff' }}
              />
            </Card>
          </Col>

          {/* 状态指示 */}
          <Col xs={24} sm={12} md={6}>
            <Card size="small" title="缓存状态">
              <Space direction="vertical" className="w-full">
                <div className="flex justify-between items-center">
                  <span>L1状态:</span>
                  <Tag color={monitor?.l1?.status?.connected ? 'success' : 'error'}>
                    {monitor?.l1?.status?.connected ? '正常' : '异常'}
                  </Tag>
                </div>
                <div className="flex justify-between items-center">
                  <span>L2状态:</span>
                  <Tag color={monitor?.l2?.status?.connected ? 'success' : 'error'}>
                    {monitor?.l2?.status?.connected ? '正常' : '异常'}
                  </Tag>
                </div>
              </Space>
            </Card>
          </Col>
        </Row>

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
                className="user-form-input"
              />
            </Col>
            <Col xs={24} sm={8} md={6}>
              <Select
                placeholder="缓存类型"
                value={searchForm.type}
                onChange={(value) => setSearchForm({ ...searchForm, type: value })}
                allowClear
                className="user-form-input"
                style={{ width: '100%' }}
                options={typeOptions}
              />
            </Col>
            <Col xs={24} sm={8} md={4}>
              <Select
                placeholder="缓存层级"
                value={searchForm.level}
                onChange={(value) => setSearchForm({ ...searchForm, level: value })}
                allowClear
                className="user-form-input"
                style={{ width: '100%' }}
                options={[
                  { label: '全部', value: 'all' },
                  { label: 'L1(内存)', value: 'l1' },
                  { label: 'L2(Redis)', value: 'l2' },
                ]}
              />
            </Col>
            <Col xs={24} sm={8} md={8}>
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
                  <Col xs={24} md={12}>
                    <Card title={<><Tag color="blue">L1</Tag> 内存使用情况</>}>
                      <Statistic
                        title="内存使用"
                        value={formatMemorySize(monitor?.l1?.stats?.usedMemory || 0)}
                        valueStyle={{ fontSize: '18px' }}
                      />
                    </Card>
                  </Col>
                  <Col xs={24} md={12}>
                    <Card title={<><Tag color="green">L2</Tag> Redis内存使用</>}>
                      <Statistic
                        title="内存使用"
                        value={formatMemorySize(monitor?.l2?.stats?.usedMemory || 0)}
                        valueStyle={{ fontSize: '18px' }}
                      />
                    </Card>
                  </Col>
                </Row>

                <Row gutter={16}>
                  <Col xs={24} md={12}>
                    <Card title={<><Tag color="blue">L1</Tag> 命中率</>}>
                      <Statistic
                        value={monitor?.l1?.stats?.hitRate || 0}
                        precision={2}
                        suffix="%"
                        valueStyle={{ color: '#3f8600' }}
                      />
                      <Divider className="my-2" />
                      <Row gutter={8}>
                        <Col span={12}>
                          <Statistic
                            title="命中"
                            value={monitor?.l1?.stats?.hitCount || 0}
                            valueStyle={{ fontSize: '16px', color: '#3f8600' }}
                          />
                        </Col>
                        <Col span={12}>
                          <Statistic
                            title="未命中"
                            value={monitor?.l1?.stats?.missCount || 0}
                            valueStyle={{ fontSize: '16px', color: '#cf1322' }}
                          />
                        </Col>
                      </Row>
                    </Card>
                  </Col>
                  <Col xs={24} md={12}>
                    <Card title={<><Tag color="green">L2</Tag> 命中率</>}>
                      <Statistic
                        value={monitor?.l2?.stats?.hitRate || 0}
                        precision={2}
                        suffix="%"
                        valueStyle={{ color: '#1890ff' }}
                      />
                      <Divider className="my-2" />
                      <Row gutter={8}>
                        <Col span={12}>
                          <Statistic
                            title="命中"
                            value={monitor?.l2?.stats?.hitCount || 0}
                            valueStyle={{ fontSize: '16px', color: '#3f8600' }}
                          />
                        </Col>
                        <Col span={12}>
                          <Statistic
                            title="未命中"
                            value={monitor?.l2?.stats?.missCount || 0}
                            valueStyle={{ fontSize: '16px', color: '#cf1322' }}
                          />
                        </Col>
                      </Row>
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
        width={900}
      >
        <Row gutter={16}>
          {/* L1统计 */}
          <Col span={12}>
            <Card size="small" title={<><Tag color="blue">L1</Tag> 内存缓存统计</>}>
              <Statistic
                title="键数量"
                value={monitor?.l1?.stats?.keyCount || 0}
                valueStyle={{ fontSize: '24px' }}
              />
              <Divider className="my-2" />
              <Row gutter={8}>
                <Col span={12}>
                  <Statistic
                    title="命中次数"
                    value={monitor?.l1?.stats?.hitCount || 0}
                    valueStyle={{ fontSize: '18px', color: '#3f8600' }}
                  />
                </Col>
                <Col span={12}>
                  <Statistic
                    title="未命中次数"
                    value={monitor?.l1?.stats?.missCount || 0}
                    valueStyle={{ fontSize: '18px', color: '#cf1322' }}
                  />
                </Col>
              </Row>
              <Divider className="my-2" />
              <Statistic
                title="命中率"
                value={monitor?.l1?.stats?.hitRate || 0}
                precision={2}
                suffix="%"
                valueStyle={{ fontSize: '20px', color: '#1890ff' }}
              />
              <Divider className="my-2" />
              <Statistic
                title="内存使用"
                value={formatMemorySize(monitor?.l1?.stats?.usedMemory || 0)}
              />
            </Card>
          </Col>

          {/* L2统计 */}
          <Col span={12}>
            <Card size="small" title={<><Tag color="green">L2</Tag> Redis缓存统计</>}>
              <Statistic
                title="键数量"
                value={monitor?.l2?.stats?.keyCount || 0}
                valueStyle={{ fontSize: '24px' }}
              />
              <Divider className="my-2" />
              <Row gutter={8}>
                <Col span={12}>
                  <Statistic
                    title="命中次数"
                    value={monitor?.l2?.stats?.hitCount || 0}
                    valueStyle={{ fontSize: '18px', color: '#3f8600' }}
                  />
                </Col>
                <Col span={12}>
                  <Statistic
                    title="未命中次数"
                    value={monitor?.l2?.stats?.missCount || 0}
                    valueStyle={{ fontSize: '18px', color: '#cf1322' }}
                  />
                </Col>
              </Row>
              <Divider className="my-2" />
              <Statistic
                title="命中率"
                value={monitor?.l2?.stats?.hitRate || 0}
                precision={2}
                suffix="%"
                valueStyle={{ fontSize: '20px', color: '#1890ff' }}
              />
              <Divider className="my-2" />
              <Statistic
                title="内存使用"
                value={formatMemorySize(monitor?.l2?.stats?.usedMemory || 0)}
              />
            </Card>
          </Col>
        </Row>

        {/* 综合对比 */}
        <Divider />
        <Card size="small" title="综合对比">
          <Row gutter={16}>
            <Col span={8}>
              <Statistic
                title="总键数(L1+L2)"
                value={(monitor?.l1?.stats?.keyCount || 0) + (monitor?.l2?.stats?.keyCount || 0)}
              />
            </Col>
            <Col span={8}>
              <Statistic
                title="总内存使用"
                value={formatMemorySize((monitor?.l1?.stats?.usedMemory || 0) + (monitor?.l2?.stats?.usedMemory || 0))}
              />
            </Col>
            <Col span={8}>
              <Statistic
                title="L1缓存命中率"
                value={monitor?.l1?.stats?.hitRate || 0}
                precision={2}
                suffix="%"
              />
            </Col>
          </Row>
        </Card>
      </Modal>
    </div>
  );
};

export default CacheManager;