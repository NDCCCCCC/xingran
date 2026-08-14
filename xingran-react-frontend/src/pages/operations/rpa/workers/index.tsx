/**
 * RPA Worker 监控页面
 */

import { useState, useCallback, useEffect, useMemo } from "react";
import type { FC } from "react";
import {
  Table, Button, Space, Form, Input, Select, Card, Row, Col, Statistic,
  Tag, Alert, Progress, Layout,
} from "antd";
import {
  SearchOutlined, ReloadOutlined, CloudServerOutlined,
  CheckCircleOutlined, CloseCircleOutlined, ClockCircleOutlined,
  LoadingOutlined,
} from "@ant-design/icons";
import type { Worker } from "@/types/rpa";
import { useTableManager } from "@/hooks/useTableManager";
import { useServerSort } from "@/hooks/useServerSort";
import { createSorterMeta } from "@/utils/tableHelpers";
import { usePagination } from "@/hooks/usePagination";
import { getWorkerColumns } from "./columns";
import { WORKER_STATUS_OPTIONS } from "../constants";
import { post } from "@/lib/api";

const { Option } = Select;
const { Content } = Layout;

const WorkerMonitor: FC = () => {
  const { paginationProps, setCurrent, setPageSize, setTotal } = usePagination();

  // 统计数据(专用聚合端点,后端按实时心跳判定 online/offline,不依赖当前页列表)
  const [statistics, setStatistics] = useState({
    total: 0,
    online: 0,
    offline: 0,
    busy: 0,
    error: 0,
    totalCapacity: 0,
    usedCapacity: 0,
  });
  const loadStatistics = useCallback(async () => {
    try {
      const result = await post<{
        total: number;
        online: number;
        offline: number;
        busy: number;
        error: number;
        totalCapacity: number;
        usedCapacity: number;
      }>("/rpa/workers/statistics");
      setStatistics(
        result.data ?? {
          total: 0,
          online: 0,
          offline: 0,
          busy: 0,
          error: 0,
          totalCapacity: 0,
          usedCapacity: 0,
        }
      );
    } catch (error) {
      console.error("加载统计数据失败:", error);
    }
  }, []);

  // 服务端排序:field 必须与 columns 的 dataIndex 一致
  const sorterMetas = useMemo(
    () => [
      createSorterMeta<Worker>("workerName"),
      createSorterMeta<Worker>("workerId"),
      createSorterMeta<Worker>("ipAddress"),
      createSorterMeta<Worker>("status"),
      createSorterMeta<Worker>("createdAt", "date"),
    ],
    []
  );
  const { orderByColumn, isAsc, handleTableChange: handleWorkerSortChange, sortOrder: workerSortOrder } = useServerSort<Worker>({
    sorterMetas,
  });

  const {
    loading,
    data: workers,
    total: _total,
    searchForm,
    loadData: loadWorkers,
    handleSearch,
    handleReset,
  } = useTableManager<Worker>(
    async (params) => {
      const result = await post("/rpa/workers/list", {
        ...params,
        ...(orderByColumn ? { orderByColumn, isAsc } : {}),
      });
      // 类型断言：后端返回的数据结构
      const data = result.data as unknown as { list?: Worker[]; total?: number } | undefined;
      // 搜索/分页/自动刷新(均经此 fetcher)顺带刷新全局统计
      loadStatistics();
      return {
        list: data?.list || [],
        total: data?.total || 0,
      };
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

  const refreshData = useCallback(() => {
    loadWorkers();
  }, [loadWorkers]);

  useEffect(() => {
    loadWorkers();
    // 自动刷新 Worker 状态
    const interval = setInterval(() => {
      loadWorkers();
    }, 10000); // 10秒刷新一次
    return () => clearInterval(interval);
  }, [loadWorkers]);

  const columns = useMemo(
  () => getWorkerColumns({
    getSortOrder: (field) => (orderByColumn === field ? (workerSortOrder ?? null) as "ascend" | "descend" | null : null),
  }),
  [orderByColumn, workerSortOrder]
);

  // 容量使用率(基于专用统计端点的全局容量,不再用当前页 reduce)
  const capacityUsagePercent = statistics.totalCapacity > 0
    ? (statistics.usedCapacity / statistics.totalCapacity) * 100
    : 0;

  return (
    <Layout style={{ background: "#000", minHeight: "calc(100vh - 64px)" }}>
      <Content style={{ background: "#fff", padding: 16 }}>
      {/* 统计卡片 */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={4}>
          <Card>
            <Statistic
              title="Worker 总数"
              value={statistics.total}
              prefix={<CloudServerOutlined />}
              styles={{ content: { color: "var(--theme-info, #1890ff)" } }}
            />
          </Card>
        </Col>
        <Col span={5}>
          <Card>
            <Statistic
              title="在线"
              value={statistics.online}
              prefix={<CheckCircleOutlined />}
              styles={{ content: { color: "var(--theme-success, #52c41a)" } }}
            />
          </Card>
        </Col>
        <Col span={5}>
          <Card>
            <Statistic
              title="忙碌"
              value={statistics.busy}
              prefix={<LoadingOutlined />}
              styles={{ content: { color: "var(--theme-info, #1890ff)" } }}
            />
          </Card>
        </Col>
        <Col span={5}>
          <Card>
            <Statistic
              title="离线"
              value={statistics.offline}
              prefix={<ClockCircleOutlined />}
              styles={{ content: { color: "var(--theme-text-tertiary, #8c8c8c)" } }}
            />
          </Card>
        </Col>
        <Col span={5}>
          <Card>
            <Statistic
              title="错误"
              value={statistics.error}
              prefix={<CloseCircleOutlined />}
              styles={{ content: { color: "var(--theme-error, #ff4d4f)" } }}
            />
          </Card>
        </Col>
      </Row>

      {/* 容量使用率 */}
      <Card style={{ marginBottom: 16 }}>
        <Row gutter={16}>
          <Col span={18}>
            <div style={{ marginBottom: 8 }}>
              <Space>
                <span>总容量使用率</span>
                <Tag color="blue">{statistics.usedCapacity}/{statistics.totalCapacity}</Tag>
              </Space>
            </div>
            <Progress
              percent={Math.round(capacityUsagePercent)}
              status={capacityUsagePercent > 80 ? "exception" : "active"}
            />
          </Col>
          <Col span={6}>
            <Statistic
              title="可用容量"
              value={statistics.totalCapacity - statistics.usedCapacity}
              suffix="个任务"
            />
          </Col>
        </Row>
      </Card>

      {/* 筛选条件 */}
      <Card style={{ marginBottom: 16 }}>
        <Form form={searchForm} layout="inline">
          <Form.Item name="name" label="Worker 名称">
            <Input placeholder="请输入Worker名称" allowClear style={{ width: 200 }} />
          </Form.Item>
          <Form.Item name="status" label="状态">
            <Select placeholder="请选择" allowClear style={{ width: 150 }} onSearch={() => {}}>
              {WORKER_STATUS_OPTIONS.map((opt) => (
                <Option key={opt.value} value={opt.value}>
                  {opt.label}
                </Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>
                搜索
              </Button>
              <Button onClick={handleReset}>重置</Button>
              <Button icon={<ReloadOutlined />} onClick={refreshData}>
                刷新
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Card>

      {/* Worker 列表 */}
      <Card>
        {statistics.offline > 0 && statistics.offline === statistics.total && (
          <Alert
            message="所有 Worker 离线"
            description="请检查 Worker 服务是否正常运行，以及网络连接是否正常"
            type="warning"
            showIcon
            style={{ marginBottom: 16 }}
          />
        )}
        {statistics.error > 0 && (
          <Alert
            message={`${statistics.error} 个 Worker 处于错误状态`}
            description="请查看 Worker 日志获取详细错误信息"
            type="error"
            showIcon
            style={{ marginBottom: 16 }}
          />
        )}
        <Table
          columns={columns}
          dataSource={workers}
          loading={loading}
          rowKey="id"
          pagination={paginationProps}
          onChange={(pagination, _filters, sorter) => {
            handleWorkerSortChange(pagination, _filters, sorter);
            const newPage = pagination.current ?? 1;
            const newPageSize = pagination.pageSize ?? 10;
            setCurrent(newPage);
            setPageSize(newPageSize);
            const formValues = searchForm.getFieldsValue() as Record<string, unknown>;
            const searchParams: Record<string, unknown> = { current: newPage, pageSize: newPageSize, ...(orderByColumn ? { orderByColumn, isAsc } : {}) };
            Object.keys(formValues).forEach(key => {
              const value = formValues[key];
              if (value !== undefined && value !== null && value !== "") {
                searchParams[key] = value;
              }
            });
            loadWorkers(searchParams);
          }}
          rowClassName={(record) => {
            if (record.status === "offline") return "row-offline";
            if (record.status === "error") return "row-error";
            return "";
          }}
        />
      </Card>

      <style>{`
        .row-offline {
          background-color: #fafafa;
        }
        .row-error {
          background-color: #fff2f0;
        }
      `}</style>
      </Content>
    </Layout>
  );
};

export default WorkerMonitor;
