/**
 * Cache 缓存管理页面
 */

import { useState, useEffect, useCallback, useMemo, type FC } from "react";
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
  Row,
  Col,
  Statistic,
  Tabs,
  App,
  Divider,
  Popconfirm,
} from "antd";
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
} from "@ant-design/icons";
import type { ColumnsType } from "antd/es/table";
import { post, get } from "@/lib/api";
import ActionButtons from "@/components/shared/ActionButtons";
import { usePagination } from "@/hooks/usePagination";
import type { PageResponse } from "@/types";
import { formatDateTime } from "@/utils/datetime";

// 导入提取的文件
import type { CacheInfo, CacheMonitor, CacheSearchForm } from "./types";
import { TYPE_OPTIONS, OPERATION_OPTIONS, LEVEL_OPTIONS, LEVEL_TAG_CONFIG } from "./constants";
import {
  formatMemorySize,
  formatTTL,
  formatDateTime as cacheFormatDateTime,
  exportCacheAsJson,
} from "./utils";
import { createSorter } from "@/utils/tableHelpers";

const { TextArea } = Input;

// ==================== 表格列定义 ====================

interface CacheTableColumnsProps {
  handleViewDetail: (key: string) => void;
  handleBatchOperate: (operation: string, keys: string[]) => void;
  formatTTL: (seconds: number) => string;
  formatMemorySize: (bytes: number) => string;
}

function getCacheTableColumns(props: CacheTableColumnsProps): ColumnsType<CacheInfo> {
  const { handleViewDetail, handleBatchOperate, formatTTL, formatMemorySize } = props;

  return [
    {
      title: "缓存键",
      dataIndex: "key",
      key: "key",
      width: 300,
      ellipsis: true,
      sorter: createSorter<CacheInfo>("key", "string"),
      render: (text: string) => (
        <code className="bg-gray-100 px-2 py-1 rounded text-xs">{text}</code>
      ),
    },
    {
      title: "缓存类型",
      dataIndex: "type",
      key: "type",
      width: 100,
      sorter: createSorter<CacheInfo>("type", "string"),
      render: (type: string) => <Tag color="blue">{type}</Tag>,
    },
    {
      title: "层级",
      dataIndex: "location",
      key: "location",
      width: 120,
      sorter: createSorter<CacheInfo>("location", "string"),
      render: (location: string) => {
        const config = LEVEL_TAG_CONFIG[location];
        if (config) {
          return <Tag color={config.color}>{config.label}</Tag>;
        }
        return <Tag>{location}</Tag>;
      },
    },
    {
      title: "TTL",
      dataIndex: "ttl",
      key: "ttl",
      width: 120,
      sorter: createSorter<CacheInfo>("ttl", "number"),
      render: (ttl: number) => <Tag color={ttl > 0 ? "green" : "default"}>{formatTTL(ttl)}</Tag>,
    },
    {
      title: "大小",
      dataIndex: "size",
      key: "size",
      width: 100,
      sorter: createSorter<CacheInfo>("size", "number"),
      render: (size: number) => formatMemorySize(size),
    },
    {
      title: "创建时间",
      dataIndex: "createdAt",
      key: "createdAt",
      width: 180,
      sorter: createSorter<CacheInfo>("createdAt", "date"),
      render: (value: string) => cacheFormatDateTime(value),
    },
    {
      title: "更新时间",
      dataIndex: "updatedAt",
      key: "updatedAt",
      width: 180,
      sorter: createSorter<CacheInfo>("updatedAt", "date"),
      render: (value: string) => cacheFormatDateTime(value),
    },
    {
      title: "操作",
      key: "action",
      width: 200,
      render: (_, record: CacheInfo) => {
        const actions = [
          {
            key: "view",
            label: "查看详情",
            icon: <EyeOutlined />,
            onClick: () => handleViewDetail(record.key),
          },
          {
            key: "delete",
            label: "删除",
            icon: <DeleteOutlined />,
            danger: true,
            render: () => (
              <Popconfirm
                title="确定要删除这个缓存吗？"
                onConfirm={() => handleBatchOperate("del", [record.key])}
              >
                <Button
                  type="link"
                  icon={<DeleteOutlined />}
                  style={{ color: "#ba3630" }}
                  size="small"
                >
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
}

// ==================== 主组件 ====================

const CacheManager: FC = () => {
  const { message } = App.useApp();
  // 状态管理
  const [caches, setCaches] = useState<CacheInfo[]>([]);
  const [monitor, setMonitor] = useState<CacheMonitor | null>(null);
  const [loading, setLoading] = useState(false);
  const [searchForm, setSearchForm] = useState<CacheSearchForm>({
    key: "",
    type: "",
    level: "all",
  });
  const [operateModalVisible, setOperateModalVisible] = useState(false);
  const [statsModalVisible, setStatsModalVisible] = useState(false);
  const [form] = Form.useForm();
  const [orderByColumn, setOrderByColumn] = useState("createdAt");
  const [isAsc, setIsAsc] = useState(false);

  // 使用全局分页 hook
  const { paginationProps, setCurrent, setPageSize, setTotal } = usePagination();

  // ==================== 数据获取 ====================

  const fetchCaches = useCallback(
    async (sortCol?: string, sortAsc?: boolean) => {
      setLoading(true);
      try {
        const result = await post<PageResponse<CacheInfo>>("/monitor/cache/list", {
          ...searchForm,
          current: paginationProps.current,
          pageSize: paginationProps.pageSize,
          orderByColumn: sortCol ?? orderByColumn,
          isAsc: sortAsc ?? isAsc,
        });

        setCaches(result.data?.list || []);
        setTotal(result.data?.total || 0);
      } catch (error) {
        console.error("获取缓存列表失败:", error);
        message.error("网络错误，请稍后重试");
      } finally {
        setLoading(false);
      }
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
    [searchForm, paginationProps.current, paginationProps.pageSize, orderByColumn, isAsc, setTotal]
  );

  const fetchMonitor = useCallback(async () => {
    try {
      const result = await post<CacheMonitor>("/monitor/cache/monitor", {});
      setMonitor(result.data || null);
    } catch (error) {
      console.error("获取监控数据失败:", error);
      message.error("获取监控数据失败");
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
  }, []);

  // ==================== 缓存操作 ====================

  const handleOperate = useCallback(async () => {
    try {
      const values = await form.validateFields();
      await post<unknown>("/monitor/cache/operate", {
        ...values,
        ttl: values.ttl || undefined,
      });

      message.success("操作成功");
      setOperateModalVisible(false);
      form.resetFields();
      // fetchCaches 内部用 orderByColumn/isAsc state 兜底，无需重传
      fetchCaches();
    } catch (error) {
      console.error("操作失败:", error);
      message.error("操作失败");
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
  }, [form, fetchCaches]);

  const handleBatchOperate = useCallback(
    async (operation: string, keys?: string[]) => {
      try {
        await post<unknown>("/monitor/cache/batch", {
          keys: keys || [],
          operation,
        });

        message.success("批量操作成功");
        // fetchCaches 内部用 orderByColumn/isAsc state 兜底，无需重传
        fetchCaches();
      } catch (error) {
        console.error("批量操作失败:", error);
        message.error("批量操作失败");
      }
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
    [fetchCaches]
  );

  const handleClearAll = useCallback(() => {
    Modal.confirm({
      title: "确认清空",
      icon: <ExclamationCircleOutlined />,
      content: "确定要清空所有缓存吗？此操作不可恢复！",
      onOk: async () => {
        try {
          await post("/monitor/cache/clear", {});

          message.success("清空成功");
          // fetchCaches 内部用 orderByColumn/isAsc state 兜底，无需重传
          fetchCaches();
          fetchMonitor();
        } catch (error) {
          console.error("清空失败:", error);
          message.error("清空失败");
        }
      },
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
  }, [fetchCaches, fetchMonitor]);

  const handleExport = useCallback(async () => {
    try {
      const result = await post<unknown[]>("/monitor/cache/export", {
        ...searchForm,
      });

      exportCacheAsJson(result.data || []);
      message.success("导出成功");
    } catch (error) {
      console.error("导出失败:", error);
      message.error("导出失败");
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
  }, [searchForm]);

  const handleViewDetail = useCallback(
    async (key: string) => {
      try {
        const result = await get<CacheInfo>(`/monitor/cache/${key}`);

        Modal.info({
          title: "缓存详情",
          width: 600,
          content: (
            <div>
              <p>
                <strong>键名：</strong>
                {result.data?.key}
              </p>
              <p>
                <strong>值：</strong>
              </p>
              <pre className="bg-gray-100 p-3 rounded">
                {JSON.stringify(JSON.parse(result.data?.value || "{}"), null, 2)}
              </pre>
              <p>
                <strong>类型：</strong>
                {result.data?.type}
              </p>
              <p>
                <strong>大小：</strong>
                {(result.data?.size || 0).toLocaleString()} 字节
              </p>
              <p>
                <strong>TTL：</strong>
                {formatTTL(result.data?.ttl || -1)}
              </p>
              <p>
                <strong>创建时间：</strong>
                {result.data?.createdAt ? formatDateTime(result.data.createdAt) : "-"}
              </p>
            </div>
          ),
        });
      } catch (error) {
        console.error("获取缓存详情失败:", error);
        message.error("获取缓存详情失败");
      }
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
    [formatTTL]
  );

  // ==================== 搜索和刷新 ====================

  const handleSearch = useCallback(() => {
    setCurrent(1);
    fetchCaches();
  }, [fetchCaches, setCurrent]);

  const handleReset = useCallback(() => {
    setSearchForm({
      key: "",
      type: "",
      level: "all",
    });
    setCurrent(1);
  }, [setCurrent]);

  const handleRefresh = useCallback(() => {
    fetchCaches();
    fetchMonitor();
  }, [fetchCaches, fetchMonitor]);

  const handleTableChange = useCallback(
    (pagination: any, _filters: Record<string, any>, sorter: any) => {
      if (sorter && sorter.field) {
        setOrderByColumn(sorter.field);
        setIsAsc(sorter.order === "ascend");
      }
      setCurrent(pagination.current);
      setPageSize(pagination.pageSize);
      const sortField = sorter && sorter.field ? sorter.field : undefined;
      const sortAsc = sorter && sorter.field ? sorter.order === "ascend" : undefined;
      fetchCaches(sortField, sortAsc);
    },
    [fetchCaches, setCurrent, setPageSize, setOrderByColumn, setIsAsc]
  );

  // ==================== 初始化 ====================

  useEffect(() => {
    fetchCaches();
    fetchMonitor();

    // 设置定时刷新（每30秒）
    const interval = setInterval(() => {
      fetchMonitor();
    }, 30000);

    return () => clearInterval(interval);
    // eslint-disable-next-line react-hooks/exhaustive-deps -- intentional dependency for re-run on change
  }, [paginationProps.current, paginationProps.pageSize, fetchCaches, fetchMonitor]);

  // 表格列 - 使用 useMemo 避免重复创建
  const columns = useMemo(
    () =>
      getCacheTableColumns({
        handleViewDetail,
        handleBatchOperate,
        formatTTL,
        formatMemorySize,
      }),
    // eslint-disable-next-line react-hooks/exhaustive-deps -- imported helpers are stable
    [handleViewDetail, handleBatchOperate, formatTTL, formatMemorySize]
  );

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
                styles={{ content: { fontSize: "20px" } }}
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
                styles={{ content: { fontSize: "20px" } }}
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
                styles={{ content: { fontSize: "20px", color: "#3f8600" } }}
              />
              <Divider className="my-2" />
              <Statistic
                title="L2命中率"
                value={monitor?.l2?.stats?.hitRate || 0}
                precision={1}
                suffix="%"
                styles={{ content: { fontSize: "16px", color: "#337ab0" } }}
              />
            </Card>
          </Col>

          {/* 状态指示 */}
          <Col xs={24} sm={12} md={6}>
            <Card size="small" title="缓存状态">
              <Space orientation="vertical" className="w-full">
                <div className="flex justify-between items-center">
                  <span>L1状态:</span>
                  <Tag color={monitor?.l1?.status?.connected ? "success" : "error"}>
                    {monitor?.l1?.status?.connected ? "正常" : "异常"}
                  </Tag>
                </div>
                <div className="flex justify-between items-center">
                  <span>L2状态:</span>
                  <Tag color={monitor?.l2?.status?.connected ? "success" : "error"}>
                    {monitor?.l2?.status?.connected ? "正常" : "异常"}
                  </Tag>
                </div>
              </Space>
            </Card>
          </Col>
        </Row>

        {/* 搜索表单 */}
        <Card className="mb-4">
          <Row gutter={16}>
            <Col xs={24} sm={8} md={5}>
              <Input
                placeholder="缓存键"
                value={searchForm.key}
                onChange={(e) => setSearchForm({ ...searchForm, key: e.target.value })}
                allowClear
                prefix={<SearchOutlined />}
                className="user-form-input"
              />
            </Col>
            <Col xs={24} sm={8} md={4}>
              <Select
                placeholder="缓存类型"
                value={searchForm.type}
                onChange={(value) => setSearchForm({ ...searchForm, type: value })}
                allowClear
                className="user-form-input"
                style={{ width: "100%" }}
                options={TYPE_OPTIONS}
                onSearch={() => {}}
              />
            </Col>
            <Col xs={24} sm={8} md={3}>
              <Select
                placeholder="缓存层级"
                value={searchForm.level}
                onChange={(value) => setSearchForm({ ...searchForm, level: value })}
                allowClear
                className="user-form-input"
                style={{ width: "100%" }}
                options={LEVEL_OPTIONS}
                onSearch={() => {}}
              />
            </Col>
            <Col xs={24} sm={24} md={12}>
              <Space wrap>
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
                <Button
                  icon={<DeleteOutlined />}
                  style={{ color: "#ba3630" }}
                  onClick={handleClearAll}
                >
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
            key: "list",
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
                pagination={paginationProps}
                onChange={handleTableChange}
              />
            ),
          },
          {
            key: "stats",
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
                    <Card
                      title={
                        <>
                          <Tag color="blue">L1</Tag> 内存使用情况
                        </>
                      }
                    >
                      <Statistic
                        title="内存使用"
                        value={formatMemorySize(monitor?.l1?.stats?.usedMemory || 0)}
                        styles={{ content: { fontSize: "18px" } }}
                      />
                    </Card>
                  </Col>
                  <Col xs={24} md={12}>
                    <Card
                      title={
                        <>
                          <Tag color="green">L2</Tag> Redis内存使用
                        </>
                      }
                    >
                      <Statistic
                        title="内存使用"
                        value={formatMemorySize(monitor?.l2?.stats?.usedMemory || 0)}
                        styles={{ content: { fontSize: "18px" } }}
                      />
                    </Card>
                  </Col>
                </Row>

                <Row gutter={16}>
                  <Col xs={24} md={12}>
                    <Card
                      title={
                        <>
                          <Tag color="blue">L1</Tag> 命中率
                        </>
                      }
                    >
                      <Statistic
                        value={monitor?.l1?.stats?.hitRate || 0}
                        precision={2}
                        suffix="%"
                        styles={{ content: { color: "#3f8600" } }}
                      />
                      <Divider className="my-2" />
                      <Row gutter={8}>
                        <Col span={12}>
                          <Statistic
                            title="命中"
                            value={monitor?.l1?.stats?.hitCount || 0}
                            styles={{ content: { fontSize: "16px", color: "#3f8600" } }}
                          />
                        </Col>
                        <Col span={12}>
                          <Statistic
                            title="未命中"
                            value={monitor?.l1?.stats?.missCount || 0}
                            styles={{ content: { fontSize: "16px", color: "#cf1322" } }}
                          />
                        </Col>
                      </Row>
                    </Card>
                  </Col>
                  <Col xs={24} md={12}>
                    <Card
                      title={
                        <>
                          <Tag color="green">L2</Tag> 命中率
                        </>
                      }
                    >
                      <Statistic
                        value={monitor?.l2?.stats?.hitRate || 0}
                        precision={2}
                        suffix="%"
                        styles={{ content: { color: "#337ab0" } }}
                      />
                      <Divider className="my-2" />
                      <Row gutter={8}>
                        <Col span={12}>
                          <Statistic
                            title="命中"
                            value={monitor?.l2?.stats?.hitCount || 0}
                            styles={{ content: { fontSize: "16px", color: "#3f8600" } }}
                          />
                        </Col>
                        <Col span={12}>
                          <Statistic
                            title="未命中"
                            value={monitor?.l2?.stats?.missCount || 0}
                            styles={{ content: { fontSize: "16px", color: "#cf1322" } }}
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
            rules={[{ required: true, message: "请输入缓存键" }]}
          >
            <Input placeholder="请输入缓存键" />
          </Form.Item>
          <Form.Item
            label="操作类型"
            name="operation"
            rules={[{ required: true, message: "请选择操作类型" }]}
          >
            <Select options={OPERATION_OPTIONS} onSearch={() => {}} />
          </Form.Item>
          <Form.Item
            label="缓存值"
            name="value"
            dependencies={["operation"]}
            rules={[
              {
                required: true,
                message: "请输入缓存值",
              },
              ({ getFieldValue }) => ({
                validator: (_, value) => {
                  if (getFieldValue("operation") === "set" && !value) {
                    return Promise.reject(new Error("请输入缓存值"));
                  }
                  return Promise.resolve();
                },
              }),
            ]}
          >
            <TextArea placeholder="请输入缓存值" rows={4} />
          </Form.Item>
          <Form.Item label="过期时间(秒)" name="ttl" dependencies={["operation"]}>
            <InputNumber placeholder="留空表示永不过期" min={1} style={{ width: "100%" }} />
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
            <Card
              size="small"
              title={
                <>
                  <Tag color="blue">L1</Tag> 内存缓存统计
                </>
              }
            >
              <Statistic
                title="键数量"
                value={monitor?.l1?.stats?.keyCount || 0}
                styles={{ content: { fontSize: "24px" } }}
              />
              <Divider className="my-2" />
              <Row gutter={8}>
                <Col span={12}>
                  <Statistic
                    title="命中次数"
                    value={monitor?.l1?.stats?.hitCount || 0}
                    valueStyle={{ fontSize: "18px", color: "#3f8600" }}
                  />
                </Col>
                <Col span={12}>
                  <Statistic
                    title="未命中次数"
                    value={monitor?.l1?.stats?.missCount || 0}
                    valueStyle={{ fontSize: "18px", color: "#cf1322" }}
                  />
                </Col>
              </Row>
              <Divider className="my-2" />
              <Statistic
                title="命中率"
                value={monitor?.l1?.stats?.hitRate || 0}
                precision={2}
                suffix="%"
                styles={{ content: { fontSize: "20px", color: "#337ab0" } }}
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
            <Card
              size="small"
              title={
                <>
                  <Tag color="green">L2</Tag> Redis缓存统计
                </>
              }
            >
              <Statistic
                title="键数量"
                value={monitor?.l2?.stats?.keyCount || 0}
                styles={{ content: { fontSize: "24px" } }}
              />
              <Divider className="my-2" />
              <Row gutter={8}>
                <Col span={12}>
                  <Statistic
                    title="命中次数"
                    value={monitor?.l2?.stats?.hitCount || 0}
                    valueStyle={{ fontSize: "18px", color: "#3f8600" }}
                  />
                </Col>
                <Col span={12}>
                  <Statistic
                    title="未命中次数"
                    value={monitor?.l2?.stats?.missCount || 0}
                    valueStyle={{ fontSize: "18px", color: "#cf1322" }}
                  />
                </Col>
              </Row>
              <Divider className="my-2" />
              <Statistic
                title="命中率"
                value={monitor?.l2?.stats?.hitRate || 0}
                precision={2}
                suffix="%"
                styles={{ content: { fontSize: "20px", color: "#337ab0" } }}
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
                value={formatMemorySize(
                  (monitor?.l1?.stats?.usedMemory || 0) + (monitor?.l2?.stats?.usedMemory || 0)
                )}
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
