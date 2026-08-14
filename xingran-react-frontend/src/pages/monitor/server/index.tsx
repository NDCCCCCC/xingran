import { useState, useEffect } from "react";
import type { FC } from "react";
import {
  Card,
  Table,
  Button,
  Space,
  Tag,
  Input,
  Select,
  Row,
  Col,
  Statistic,
  Progress,
  Drawer,
  Descriptions,
  App,
} from "antd";
import {
  SearchOutlined,
  ReloadOutlined,
  EyeOutlined,
  ClusterOutlined,
  CloudServerOutlined,
} from "@ant-design/icons";
import { post } from "@/lib/api";
import { usePagination } from "@/hooks/usePagination";
import type { ColumnsType } from "antd/es/table";
import type { BaseResponse, PageResponse } from "@/types";
import ActionButtons from "@/components/shared/ActionButtons";
import { formatDateTime } from "@/utils/datetime";
import { createSorter } from "@/utils/tableHelpers";

interface ServerInfo {
  id: string;
  hostName: string;
  os: string;
  arch: string;
  cpuCount: number;
  totalMemory: number;
  availableMemory: number;
  diskTotal: number;
  diskAvailable: number;
  status: number;
  lastActiveAt: string;
  createdAt: string;
}

interface ServerMetrics {
  serverId: string;
  cpuUsage: number;
  memoryUsage: number;
  diskUsage: number;
  networkRx: number;
  networkTx: number;
  processCount: number;
  loadAverage: number;
  timestamp: string;
}

const ServerMonitor: FC = () => {
  const { message } = App.useApp();
  const [servers, setServers] = useState<ServerInfo[]>([]);
  const [loading, setLoading] = useState(false);
  const [searchForm, setSearchForm] = useState({
    hostName: "",
    status: undefined,
  });
  const [detailDrawerVisible, setDetailDrawerVisible] = useState(false);
  const [selectedServer, setSelectedServer] = useState<ServerInfo | null>(null);
  const [orderByColumn, setOrderByColumn] = useState("timestamp");
  const [isAsc, setIsAsc] = useState(false);

  // 使用全局分页 hook
  const { paginationProps, setCurrent, setPageSize, setTotal } = usePagination();

  // 获取服务器列表
  const fetchServers = async (sortCol?: string, sortAsc?: boolean) => {
    setLoading(true);
    try {
      const result = await post<PageResponse<ServerInfo>>("/monitor/server-info/list", {
        ...searchForm,
        current: paginationProps.current,
        pageSize: paginationProps.pageSize,
        orderByColumn: sortCol ?? orderByColumn,
        isAsc: sortAsc ?? isAsc,
      });

      setServers(result.data?.list || []);
      setTotal(result.data?.total || 0);
    } catch (error) {
      console.error("获取服务器列表失败:", error);
      message.error("网络错误，请稍后重试");
    } finally {
      setLoading(false);
    }
  };

  // 获取服务器详情
  const fetchServerDetail = async (server: ServerInfo) => {
    try {
      // 获取历史指标数据
      const result = await post<PageResponse<ServerMetrics>>("/monitor/server-metrics/history", {
        serverId: server.id,
        startTime: new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(), // 最近24小时
        endTime: new Date().toISOString(),
        current: 1,
        pageSize: 100,
      });

      // 历史指标数据已获取但暂未使用
    } catch (error) {
      console.error("获取服务器指标历史失败:", error);
    }
  };

  // 查看详情
  const handleViewDetail = (record: ServerInfo) => {
    setSelectedServer(record);
    setDetailDrawerVisible(true);
    fetchServerDetail(record);
  };

  // 搜索
  const handleSearch = () => {
    setCurrent(1);
    // fetchServers 内部用 orderByColumn/isAsc state 兜底，无需重传
    fetchServers();
  };

  // 重置
  const handleReset = () => {
    setSearchForm({
      hostName: "",
      status: undefined,
    });
    setCurrent(1);
  };

  // 刷新
  const handleRefresh = () => {
    fetchServers();
  };

  // 分页变化
  const handleTableChange = (pagination: any, _filters: Record<string, any>, sorter: any) => {
    if (sorter && sorter.field) {
      setOrderByColumn(sorter.field);
      setIsAsc(sorter.order === "ascend");
    }
    setCurrent(pagination.current);
    setPageSize(pagination.pageSize);
    const sortField = sorter && sorter.field ? sorter.field : undefined;
    const sortAsc = sorter && sorter.field ? sorter.order === "ascend" : undefined;
    fetchServers(sortField, sortAsc);
  };

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
    const units = ["B", "KB", "MB", "GB"];
    let size = bytes;
    let unitIndex = 0;

    while (size >= 1024 && unitIndex < units.length - 1) {
      size /= 1024;
      unitIndex++;
    }

    return `${size.toFixed(2)} ${units[unitIndex]}`;
  };

  useEffect(() => {
    // fetchServers 内部用 orderByColumn/isAsc state 兜底，无需重传
    fetchServers();
  }, [paginationProps.current, paginationProps.pageSize]);

  const columns: ColumnsType<ServerInfo> = [
    {
      title: "主机名",
      dataIndex: "hostName",
      key: "hostName",
      width: 160,
      minWidth: 140,
      sorter: createSorter<ServerInfo>("hostName", "string"),
      render: (text: string) => (
        <Space>
          <CloudServerOutlined />
          {text}
        </Space>
      ),
    },
    {
      title: "操作系统",
      dataIndex: "os",
      key: "os",
      width: 140,
      minWidth: 120,
      sorter: createSorter<ServerInfo>("os", "string"),
    },
    {
      title: "架构",
      dataIndex: "arch",
      key: "arch",
      width: 80,
      minWidth: 70,
      sorter: createSorter<ServerInfo>("arch", "string"),
    },
    {
      title: "CPU核心",
      dataIndex: "cpuCount",
      key: "cpuCount",
      width: 100,
      minWidth: 90,
      sorter: createSorter<ServerInfo>("cpuCount", "number"),
      render: (count: number) => (
        <Statistic
          value={count}
          styles={{ content: { fontSize: "14px" } }}
          prefix={<ClusterOutlined />}
        />
      ),
    },
    {
      title: "内存",
      dataIndex: "totalMemory",
      key: "totalMemory",
      width: 180,
      minWidth: 160,
      sorter: createSorter<ServerInfo>("totalMemory", "number"),
      render: (total: number, record: ServerInfo) => (
        <div>
          <div>总计: {formatMemorySize(total)}</div>
          <div className="text-sm text-gray-500">
            可用: {formatMemorySize(record.availableMemory)}
          </div>
          <Progress
            percent={Math.round((1 - record.availableMemory / total) * 100)}
            size="small"
            className="mt-1"
          />
        </div>
      ),
    },
    {
      title: "磁盘",
      dataIndex: "diskTotal",
      key: "diskTotal",
      width: 180,
      minWidth: 160,
      sorter: createSorter<ServerInfo>("diskTotal", "number"),
      render: (total: number, record: ServerInfo) => (
        <div>
          <div>总计: {formatMemorySize(total)}</div>
          <div className="text-sm text-gray-500">
            可用: {formatMemorySize(record.diskAvailable)}
          </div>
          <Progress
            percent={Math.round((1 - record.diskAvailable / total) * 100)}
            size="small"
            className="mt-1"
          />
        </div>
      ),
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 80,
      minWidth: 70,
      align: "center" as const,
      sorter: createSorter<ServerInfo>("status", "number"),
      render: (status: number) => (
        <Tag color={status === 0 ? "success" : "error"}>{status === 0 ? "正常" : "异常"}</Tag>
      ),
    },
    {
      title: "最后活跃时间",
      dataIndex: "lastActiveAt",
      key: "lastActiveAt",
      width: 170,
      minWidth: 160,
      sorter: createSorter<ServerInfo>("lastActiveAt", "date"),
      render: (time: string) => formatDateTime(time),
    },
    {
      title: "操作",
      key: "action",
      width: 100,
      minWidth: 80,
      fixed: "right" as const,
      render: (_, record: ServerInfo) => {
        const actions = [
          {
            key: "detail",
            label: "详情",
            icon: <EyeOutlined />,
            onClick: () => handleViewDetail(record),
          },
        ];
        return <ActionButtons actions={actions} />;
      },
    },
  ];

  return (
    <div className="p-6">
      <div className="mb-6">
        <h1 className="text-2xl font-bold mb-4">服务器监控</h1>

        {/* 搜索表单 */}
        <Card>
          <Row gutter={16}>
            <Col xs={24} sm={8} md={6}>
              <Input
                placeholder="主机名"
                value={searchForm.hostName}
                onChange={(e) => setSearchForm({ ...searchForm, hostName: e.target.value })}
                allowClear
                className="user-form-input"
              />
            </Col>
            <Col xs={24} sm={8} md={6}>
              <Select
                placeholder="状态"
                value={searchForm.status}
                onChange={(value) => setSearchForm({ ...searchForm, status: value })}
                allowClear
                className="user-form-input"
                style={{ width: "100%" }}
                options={[
                  { label: "正常", value: 0 },
                  { label: "异常", value: 1 },
                ]}
                onSearch={() => {}}
              />
            </Col>
            <Col xs={24} sm={8} md={12}>
              <Space>
                <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>
                  搜索
                </Button>
                <Button onClick={handleReset}>重置</Button>
                <Button icon={<ReloadOutlined />} onClick={handleRefresh}>
                  刷新
                </Button>
              </Space>
            </Col>
          </Row>
        </Card>
      </div>

      {/* 服务器列表 */}
      <Card>
        <Table
          columns={columns}
          dataSource={servers}
          rowKey="id"
          loading={loading}
          pagination={paginationProps}
          onChange={handleTableChange} // replaced with handleTableChange
        />
      </Card>

      {/* 详情抽屉 */}
      <Drawer
        title={`服务器详情 - ${selectedServer?.hostName}`}
        placement="right"
        onClose={() => setDetailDrawerVisible(false)}
        open={detailDrawerVisible}
        size={600}
      >
        {selectedServer && (
          <div>
            {/* 基本信息 */}
            <Card title="基本信息" className="mb-4">
              <Descriptions column={1}>
                <Descriptions.Item label="主机名">{selectedServer.hostName}</Descriptions.Item>
                <Descriptions.Item label="操作系统">{selectedServer.os}</Descriptions.Item>
                <Descriptions.Item label="系统架构">{selectedServer.arch}</Descriptions.Item>
                <Descriptions.Item label="CPU核心数">{selectedServer.cpuCount}</Descriptions.Item>
                <Descriptions.Item label="总内存">
                  {formatMemorySize(selectedServer.totalMemory)}
                </Descriptions.Item>
                <Descriptions.Item label="可用内存">
                  {formatMemorySize(selectedServer.availableMemory)}
                </Descriptions.Item>
                <Descriptions.Item label="磁盘总容量">
                  {formatMemorySize(selectedServer.diskTotal)}
                </Descriptions.Item>
                <Descriptions.Item label="磁盘可用容量">
                  {formatMemorySize(selectedServer.diskAvailable)}
                </Descriptions.Item>
                <Descriptions.Item label="状态">
                  <Tag color={selectedServer.status === 0 ? "success" : "error"}>
                    {selectedServer.status === 0 ? "正常" : "异常"}
                  </Tag>
                </Descriptions.Item>
                <Descriptions.Item label="最后活跃时间">
                  {formatDateTime(selectedServer.lastActiveAt)}
                </Descriptions.Item>
              </Descriptions>
            </Card>

            {/* 实时指标 */}
            <Card title="实时指标" className="mb-4">
              <Row gutter={16}>
                <Col span={12}>
                  <Statistic
                    title="CPU使用率"
                    value={75.2}
                    precision={1}
                    suffix="%"
                    styles={{ content: { color: "#cf1322" } }}
                  />
                  <Progress percent={75.2} className="mt-2" />
                </Col>
                <Col span={12}>
                  <Statistic
                    title="内存使用率"
                    value={68.5}
                    precision={1}
                    suffix="%"
                    styles={{ content: { color: "#3f8600" } }}
                  />
                  <Progress percent={68.5} className="mt-2" />
                </Col>
                <Col span={12}>
                  <Statistic
                    title="磁盘使用率"
                    value={45.8}
                    precision={1}
                    suffix="%"
                    styles={{ content: { color: "#3f8600" } }}
                  />
                  <Progress percent={45.8} className="mt-2" />
                </Col>
                <Col span={12}>
                  <Statistic
                    title="系统负载"
                    value={2.35}
                    precision={2}
                    prefix={<ClusterOutlined />}
                  />
                </Col>
              </Row>
            </Card>

            {/* 网络流量 */}
            <Card title="网络流量" className="mb-4">
              <Row gutter={16}>
                <Col span={12}>
                  <Statistic
                    title="接收流量"
                    value={1234.56}
                    precision={2}
                    suffix="MB/s"
                    styles={{ content: { color: "#1890ff" } }}
                  />
                </Col>
                <Col span={12}>
                  <Statistic
                    title="发送流量"
                    value={987.32}
                    precision={2}
                    suffix="MB/s"
                    styles={{ content: { color: "#52c41a" } }}
                  />
                </Col>
              </Row>
            </Card>
          </div>
        )}
      </Drawer>
    </div>
  );
};

export default ServerMonitor;
