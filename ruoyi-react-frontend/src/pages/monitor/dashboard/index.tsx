import { useState, useEffect, useCallback, useRef } from 'react';
import type { FC } from 'react';
import { Card, Row, Col, Statistic, Progress, Table, Tag, Button } from 'antd';
import {
  ClusterOutlined,
  ReloadOutlined,
  BarChartOutlined,
  DatabaseOutlined
} from '@ant-design/icons';
import { post } from '@/lib/api';
import type { BaseResponse, PageResponse } from '@/types';

interface ServerMetrics {
  cpuUsage: number;
  memoryUsage: number;
  diskUsage: number;
  networkRx: number;
  networkTx: number;
  processCount: number;
  totalMemory: number;
  usedMemory: number;
  timestamp: string;
}

interface SystemInfo {
  id: string;
  hostName: string;
  os: string;
  arch: string;
  cpuCount: number;
  totalMemory: number;
  status: number;
  lastActiveAt: string;
}

interface LoginLog {
  id: string;
  userName: string;
  ipAddr: string;
  loginLocation: string;
  status: number;
  loginTime: string;
}

const Dashboard: FC = () => {
  const [metrics, setMetrics] = useState<ServerMetrics | null>(null);
  const [servers, setServers] = useState<SystemInfo[]>([]);
  const [recentLogs, setRecentLogs] = useState<LoginLog[]>([]);
  const [loading, setLoading] = useState(false);
  const isInitialMount = useRef(true);

  // 获取服务器指标
  const fetchServerMetrics = useCallback(async () => {
    try {
      const result = await post<BaseResponse<ServerMetrics>>('/monitor/server-metrics/current', {});
      setMetrics(result.data || null);
    } catch (error) {
      console.error('获取服务器指标失败:', error);
    }
  }, []);

  // 获取服务器列表
  const fetchServerList = useCallback(async () => {
    try {
      const result = await post<BaseResponse<PageResponse<SystemInfo>>>('/monitor/server-info/list', {
        current: 1,
        pageSize: 10
      });
      setServers(result.data?.list || []);
    } catch (error) {
      console.error('获取服务器列表失败:', error);
    }
  }, []);

  // 获取最近登录日志
  const fetchRecentLogs = useCallback(async () => {
    try {
      const result = await post<BaseResponse<PageResponse<LoginLog>>>('/monitor/login-logs/list', {
        current: 1,
        pageSize: 5
      });
      setRecentLogs(result.data?.list || []);
    } catch (error) {
      console.error('获取登录日志失败:', error);
    }
  }, []);

  // 刷新数据
  const refreshData = useCallback(async () => {
    // 不是初始加载时才显示 loading
    if (!isInitialMount.current) {
      setLoading(true);
    }
    await Promise.all([
      fetchServerMetrics(),
      fetchServerList(),
      fetchRecentLogs()
    ]);
    setLoading(false);
  }, [fetchServerMetrics, fetchServerList, fetchRecentLogs]);

  useEffect(() => {
    // 初始加载 - 使用 setTimeout 避免同步 setState
    setTimeout(() => {
      Promise.all([
        fetchServerMetrics(),
        fetchServerList(),
        fetchRecentLogs()
      ]).finally(() => {
        isInitialMount.current = false;
      });
    }, 0);

    // 设置定时刷新（每30秒）
    const interval = setInterval(refreshData, 30000);

    return () => clearInterval(interval);

  }, [fetchServerMetrics, fetchServerList, fetchRecentLogs, refreshData]);

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

  // 格式化网络流量
  const formatNetworkSpeed = (bytes: number): string => {
    const units = ['B/s', 'KB/s', 'MB/s', 'GB/s'];
    let size = bytes;
    let unitIndex = 0;

    while (size >= 1024 && unitIndex < units.length - 1) {
      size /= 1024;
      unitIndex++;
    }

    return `${size.toFixed(2)} ${units[unitIndex]}`;
  };

  const logColumns = [
    {
      title: '用户名',
      dataIndex: 'userName',
      key: 'userName',
      width: 120,
      minWidth: 100,
    },
    {
      title: '登录IP',
      dataIndex: 'ipAddr',
      key: 'ipAddr',
      width: 140,
      minWidth: 120,
    },
    {
      title: '登录地点',
      dataIndex: 'loginLocation',
      key: 'loginLocation',
      width: 150,
      minWidth: 120,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 80,
      minWidth: 70,
      align: 'center' as const,
      render: (status: number) => (
        <Tag color={status === 0 ? 'success' : 'error'}>
          {status === 0 ? '成功' : '失败'}
        </Tag>
      ),
    },
    {
      title: '登录时间',
      dataIndex: 'loginTime',
      key: 'loginTime',
      width: 170,
      minWidth: 160,
    },
  ];

  return (
    <div className="p-6">
      <div className="mb-6 flex justify-between items-center">
        <h1 className="text-2xl font-bold">监控仪表盘</h1>
        <Button
          type="primary"
          icon={<ReloadOutlined />}
          loading={loading}
          onClick={refreshData}
        >
          刷新
        </Button>
      </div>

      {/* 系统概览卡片 */}
      <Row gutter={[16, 16]} className="mb-6">
        <Col xs={24} sm={12} md={6}>
          <Card>
            <Statistic
              title="CPU使用率"
              value={metrics?.cpuUsage || 0}
              precision={1}
              suffix="%"
              styles={{ content: { color: metrics?.cpuUsage && metrics.cpuUsage > 80 ? '#cf1322' : '#3f8600' } }}
              prefix={<ClusterOutlined />}
            />
            {metrics?.cpuUsage && (
              <Progress
                percent={metrics.cpuUsage}
                showInfo={false}
                strokeColor={metrics.cpuUsage > 80 ? '#cf1322' : '#3f8600'}
                className="mt-2"
              />
            )}
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card>
            <Statistic
              title="内存使用率"
              value={metrics?.memoryUsage || 0}
              precision={1}
              suffix="%"
              styles={{ content: { color: metrics?.memoryUsage && metrics.memoryUsage > 80 ? '#cf1322' : '#3f8600' } }}
              prefix={<DatabaseOutlined />}
            />
            {metrics?.memoryUsage && (
              <Progress
                percent={metrics.memoryUsage}
                showInfo={false}
                strokeColor={metrics.memoryUsage > 80 ? '#cf1322' : '#3f8600'}
                className="mt-2"
              />
            )}
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card>
            <Statistic
              title="磁盘使用率"
              value={metrics?.diskUsage || 0}
              precision={1}
              suffix="%"
              styles={{ content: { color: metrics?.diskUsage && metrics.diskUsage > 80 ? '#cf1322' : '#3f8600' } }}
              prefix={<DatabaseOutlined />}
            />
            {metrics?.diskUsage && (
              <Progress
                percent={metrics.diskUsage}
                showInfo={false}
                strokeColor={metrics.diskUsage > 80 ? '#cf1322' : '#3f8600'}
                className="mt-2"
              />
            )}
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card>
            <Statistic
              title="进程数量"
              value={metrics?.processCount || 0}
              styles={{ content: { color: '#1890ff' } }}
              prefix={<ClusterOutlined />}
            />
          </Card>
        </Col>
      </Row>

      {/* 详细信息卡片 */}
      <Row gutter={[16, 16]} className="mb-6">
        <Col xs={24} md={8}>
          <Card title="服务器列表" extra={<Button size="small" icon={<BarChartOutlined />}>详情</Button>}>
            <div className="space-y-3">
              {servers.map((server) => (
                <div key={server.id} className="flex justify-between items-center p-3 bg-gray-50 rounded">
                  <div>
                    <div className="font-medium">{server.hostName}</div>
                    <div className="text-sm text-gray-500">{server.os}</div>
                  </div>
                  <Tag color={server.status === 0 ? 'success' : 'error'}>
                    {server.status === 0 ? '正常' : '异常'}
                  </Tag>
                </div>
              ))}
            </div>
          </Card>
        </Col>

        <Col xs={24} md={8}>
          <Card title="内存详情" extra={<Button size="small" icon={<BarChartOutlined />}>趋势</Button>}>
            <div className="space-y-3">
              <div className="flex justify-between">
                <span>总内存</span>
                <span className="font-medium">
                  {metrics ? formatMemorySize(metrics.totalMemory) : '-'}
                </span>
              </div>
              <div className="flex justify-between">
                <span>已用内存</span>
                <span className="font-medium">
                  {metrics ? formatMemorySize(metrics.usedMemory) : '-'}
                </span>
              </div>
              <div className="flex justify-between">
                <span>可用内存</span>
                <span className="font-medium text-green-600">
                  {metrics ? formatMemorySize(metrics.totalMemory - metrics.usedMemory) : '-'}
                </span>
              </div>
            </div>
          </Card>
        </Col>

        <Col xs={24} md={8}>
          <Card title="网络流量" extra={<Button size="small" icon={<BarChartOutlined />}>详情</Button>}>
            <div className="space-y-3">
              <div className="flex justify-between">
                <span>接收流量</span>
                <span className="font-medium text-blue-600">
                  {metrics ? formatNetworkSpeed(metrics.networkRx) : '-'}
                </span>
              </div>
              <div className="flex justify-between">
                <span>发送流量</span>
                <span className="font-medium text-green-600">
                  {metrics ? formatNetworkSpeed(metrics.networkTx) : '-'}
                </span>
              </div>
            </div>
          </Card>
        </Col>
      </Row>

      {/* 最近登录日志 */}
      <Card title="最近登录日志" extra={
        <Button type="link">查看更多</Button>
      }>
        <Table
          columns={logColumns}
          dataSource={recentLogs}
          rowKey="id"
          pagination={false}
          size="small"
        />
      </Card>
    </div>
  );
};

export default Dashboard;