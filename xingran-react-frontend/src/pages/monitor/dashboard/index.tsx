import { useState, useEffect, useRef } from "react";
import type { FC } from "react";
import { Card, Row, Col, Statistic, Table, Tag, Button } from "antd";
import {
  ClusterOutlined,
  ReloadOutlined,
  DatabaseOutlined
} from "@ant-design/icons";
import { post } from "@/lib/api";
import { formatDateTime } from "@/utils/datetime";
import { createSorter } from "@/utils/tableHelpers";
import type { PageResponse } from "@/types";

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

  // 刷新数据 - 直接定义，避免 useCallback 的依赖问题
  // 遵循 Vercel React Best Practices: 将事件处理逻辑直接放在事件处理器中
  const refreshData = async () => {
    // 不是初始加载时才显示 loading
    if (!isInitialMount.current) {
      setLoading(true);
    }
    try {
      const [metricsResult, serversResult, logsResult] = await Promise.all([
        post<ServerMetrics>("/monitor/server-metrics/current", {}),
        post<PageResponse<SystemInfo>>("/monitor/server-info/list", { current: 1, pageSize: 10 }),
        post<PageResponse<LoginLog>>("/monitor/login-logs/list", { current: 1, pageSize: 5 })
      ]);
      setMetrics(metricsResult.data || null);
      setServers(serversResult.data?.list || []);
      setRecentLogs(logsResult.data?.list || []);
    } catch (error) {
      console.error("刷新数据失败:", error);
    } finally {
      setLoading(false);
      isInitialMount.current = false;
    }
  };

  useEffect(() => {
    // 初始加载
    // 遵循 Vercel React Best Practices: 移除不必要的 setTimeout，直接并行加载数据
    let isMounted = true;

    Promise.all([
      post<ServerMetrics>("/monitor/server-metrics/current", {}),
      post<PageResponse<SystemInfo>>("/monitor/server-info/list", { current: 1, pageSize: 10 }),
      post<PageResponse<LoginLog>>("/monitor/login-logs/list", { current: 1, pageSize: 5 })
    ])
      .then(([metricsResult, serversResult, logsResult]) => {
        if (isMounted) {
          setMetrics(metricsResult.data || null);
          setServers(serversResult.data?.list || []);
          setRecentLogs(logsResult.data?.list || []);
          isInitialMount.current = false;
        }
      })
      .catch(error => {
        console.error("初始加载失败:", error);
        if (isMounted) {
          isInitialMount.current = false;
        }
      });

    // 设置定时刷新（每30秒）
    const interval = setInterval(() => {
      refreshData();
    }, 30000);

    return () => {
      isMounted = false;
      clearInterval(interval);
    };
  }, []); // 空依赖数组 - 只在组件挂载时执行一次

  // 格式化内存大小
  const formatMemorySize = (bytes: number): string => {
    const units = ["B", "KB", "MB", "GB", "TB"];
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
    const units = ["B/s", "KB/s", "MB/s", "GB/s"];
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
      title: "用户名",
      dataIndex: "userName",
      key: "userName",
      width: 120,
      minWidth: 100,
      sorter: createSorter<LoginLog>("userName", "string"),
    },
    {
      title: "登录IP",
      dataIndex: "ipAddr",
      key: "ipAddr",
      width: 140,
      minWidth: 120,
      sorter: createSorter<LoginLog>("ipAddr", "string"),
    },
    {
      title: "登录地点",
      dataIndex: "loginLocation",
      key: "loginLocation",
      width: 150,
      minWidth: 120,
      sorter: createSorter<LoginLog>("loginLocation", "string"),
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 80,
      minWidth: 70,
      align: "center" as const,
      sorter: createSorter<LoginLog>("status", "number"),
      render: (status: number) => (
        <Tag color={status === 0 ? "success" : "error"}>
          {status === 0 ? "成功" : "失败"}
        </Tag>
      ),
    },
    {
      title: "登录时间",
      dataIndex: "loginTime",
      key: "loginTime",
      width: 180,
      minWidth: 170,
      sorter: createSorter<LoginLog>("loginTime", "date"),
      render: (time: string) => formatDateTime(time),
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
        <Col span={6}>
          <Card>
            <Statistic
              title="CPU使用率"
              value={metrics?.cpuUsage || 0}
              precision={1}
              suffix="%"
              styles={{ content: { color: metrics?.cpuUsage && metrics.cpuUsage > 80 ? "#cf1322" : "#3f8600" } }}
              prefix={<ClusterOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="内存使用率"
              value={metrics?.memoryUsage || 0}
              precision={1}
              suffix="%"
              styles={{ content: { color: metrics?.memoryUsage && metrics.memoryUsage > 80 ? "#cf1322" : "#3f8600" } }}
              prefix={<DatabaseOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="磁盘使用率"
              value={metrics?.diskUsage || 0}
              precision={1}
              suffix="%"
              styles={{ content: { color: metrics?.diskUsage && metrics.diskUsage > 80 ? "#cf1322" : "#3f8600" } }}
              prefix={<DatabaseOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="进程数量"
              value={metrics?.processCount || 0}
              styles={{ content: { color: "#1890ff" } }}
              prefix={<ClusterOutlined />}
            />
          </Card>
        </Col>
      </Row>

      {/* 详细信息卡片 */}
      <Row gutter={[16, 16]} className="mb-6">
        <Col xs={24} md={8}>
          <Card title="服务器列表">
            <div className="space-y-3">
              {servers.map((server) => (
                <div key={server.id} className="flex justify-between items-center p-3 bg-gray-50 rounded">
                  <div>
                    <div className="font-medium">{server.hostName}</div>
                    <div className="text-sm text-gray-500">{server.os}</div>
                  </div>
                  <Tag color={server.status === 0 ? "success" : "error"}>
                    {server.status === 0 ? "正常" : "异常"}
                  </Tag>
                </div>
              ))}
            </div>
          </Card>
        </Col>

        <Col xs={24} md={8}>
          <Card title="内存详情">
            <div className="space-y-3">
              <div className="flex justify-between">
                <span>总内存</span>
                <span className="font-medium">
                  {metrics ? formatMemorySize(metrics.totalMemory) : "-"}
                </span>
              </div>
              <div className="flex justify-between">
                <span>已用内存</span>
                <span className="font-medium">
                  {metrics ? formatMemorySize(metrics.usedMemory) : "-"}
                </span>
              </div>
              <div className="flex justify-between">
                <span>可用内存</span>
                <span className="font-medium text-green-600">
                  {metrics ? formatMemorySize(metrics.totalMemory - metrics.usedMemory) : "-"}
                </span>
              </div>
            </div>
          </Card>
        </Col>

        <Col xs={24} md={8}>
          <Card title="网络流量">
            <div className="space-y-3">
              <div className="flex justify-between">
                <span>接收流量</span>
                <span className="font-medium text-blue-600">
                  {metrics ? formatNetworkSpeed(metrics.networkRx) : "-"}
                </span>
              </div>
              <div className="flex justify-between">
                <span>发送流量</span>
                <span className="font-medium text-green-600">
                  {metrics ? formatNetworkSpeed(metrics.networkTx) : "-"}
                </span>
              </div>
            </div>
          </Card>
        </Col>
      </Row>

      {/* 最近登录日志 */}
      <Card title="最近登录日志">
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