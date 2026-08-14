/**
 * RPA 执行记录页面
 */

import { useState, useCallback, useEffect, useMemo } from "react";
import type { FC } from "react";
import {
  Table, Button, Space, Form, Input, Select, Card, Tag, Layout,
} from "antd";
import {
  SearchOutlined, ReloadOutlined, FilterOutlined,
} from "@ant-design/icons";
import type { Execution } from "@/types/rpa";
import type { PageResponse } from "@/types/base";
import { useTableManager } from "@/hooks/useTableManager";
import { useServerSort } from "@/hooks/useServerSort";
import { createSorterMeta } from "@/utils/tableHelpers";
import { usePagination } from "@/hooks/usePagination";
import { getExecutionColumns } from "./columns";
import { ExecutionDetailModal } from "./ExecutionDetailModal";
import { EXECUTION_STATUS_OPTIONS } from "../constants";
import { post } from "@/lib/api";

const { Option } = Select;
const { Content } = Layout;

const ExecutionManagement: FC = () => {
  const { paginationProps, setCurrent, setPageSize, setTotal } = usePagination();

  // 统计数据(专用 COUNT 端点,不受分页/筛选影响,不依赖当前页列表)
  const [statistics, setStatistics] = useState({
    total: 0,
    pending: 0,
    running: 0,
    success: 0,
    failed: 0,
  });
  const loadStatistics = useCallback(async () => {
    try {
      const result = await post<{
        total: number;
        pending: number;
        running: number;
        success: number;
        failed: number;
      }>("/rpa/executions/statistics");
      setStatistics(result.data ?? { total: 0, pending: 0, running: 0, success: 0, failed: 0 });
    } catch (error) {
      console.error("加载统计数据失败:", error);
    }
  }, []);

  // 服务端排序:field 必须与 columns 的 dataIndex 一致
  const sorterMetas = useMemo(
    () => [
      createSorterMeta<Execution>("taskName"),
      createSorterMeta<Execution>("status"),
      createSorterMeta<Execution>("workerName"),
      createSorterMeta<Execution>("startTime"),
      createSorterMeta<Execution>("endTime"),
      createSorterMeta<Execution>("createdAt", "date"),
    ],
    []
  );
  const { orderByColumn, isAsc, handleTableChange: handleExecSortChange, sortOrder: execSortOrder } = useServerSort<Execution>({
    sorterMetas,
  });

  const {
    loading,
    data: executions,
    total: _total,
    searchForm,
    loadData: loadExecutions,
    handleSearch,
    handleReset,
  } = useTableManager<Execution>(
    async (params) => {
      const result = await post<PageResponse<Execution>>("/rpa/executions/list", {
        ...params,
        ...(orderByColumn ? { orderByColumn, isAsc } : {}),
      });
      // 搜索/分页/自动刷新(均经此 fetcher)顺带刷新全局统计为真实 COUNT
      loadStatistics();
      return {
        list: result.data?.list || [],
        total: result.data?.total || 0,
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

  const [detailModalVisible, setDetailModalVisible] = useState(false);
  const [selectedExecution, setSelectedExecution] = useState<Execution | null>(null);

  const refreshData = useCallback(() => {
    loadExecutions();
  }, [loadExecutions]);

  useEffect(() => {
    loadExecutions();
    // 自动刷新运行中的任务
    const interval = setInterval(() => {
      loadExecutions();
    }, 5000);
    return () => clearInterval(interval);
  }, [loadExecutions]);

  const handleViewDetail = useCallback((record: Execution) => {
    setSelectedExecution(record);
    setDetailModalVisible(true);
  }, []);

  const columns = useMemo(
    () =>
      getExecutionColumns({
        handleViewDetail,
        getSortOrder: (field) => (orderByColumn === field ? (execSortOrder ?? null) as "ascend" | "descend" | null : null),
      }),
    [handleViewDetail, orderByColumn, execSortOrder]
  );

  return (
    <Layout style={{ background: "#000", minHeight: "calc(100vh - 64px)" }}>
      <Content style={{ background: "#fff", padding: 16 }}>
      <Card style={{ marginBottom: 16 }}>
        <div
          style={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "flex-start",
            flexWrap: "wrap",
            gap: "16px",
          }}
        >
          <Form form={searchForm} layout="inline" style={{ flex: 1, minWidth: 0 }}>
            <Form.Item name="taskName" label="任务名称">
              <Input placeholder="请输入任务名称" allowClear style={{ width: 200 }} />
            </Form.Item>
            <Form.Item name="status" label="执行状态">
              <Select placeholder="请选择" allowClear style={{ width: 150 }} onSearch={() => {}}>
                {EXECUTION_STATUS_OPTIONS.map((opt) => (
                  <Option key={opt.value} value={opt.value}>
                    {opt.label}
                  </Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item name="workerId" label="Worker">
              <Input placeholder="请输入Worker ID" allowClear style={{ width: 150 }} />
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
        </div>

        {/* 统计信息 */}
        <Space style={{ marginTop: 12, paddingTop: 12, borderTop: "1px solid #f0f0f0" }}>
          <Tag icon={<FilterOutlined />} color="blue">
            总计: {statistics.total}
          </Tag>
          <Tag color="processing">运行中: {statistics.running}</Tag>
          <Tag color="success">成功: {statistics.success}</Tag>
          <Tag color="error">失败: {statistics.failed}</Tag>
        </Space>
      </Card>

      <Card>
        <Table
          columns={columns}
          dataSource={executions}
          loading={loading}
          rowKey="id"
          pagination={paginationProps}
          onChange={(pagination, _filters, sorter) => {
            handleExecSortChange(pagination, _filters, sorter);
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
            loadExecutions(searchParams);
          }}
        />
      </Card>

      <ExecutionDetailModal
        open={detailModalVisible}
        execution={selectedExecution}
        onClose={() => {
          setDetailModalVisible(false);
          setSelectedExecution(null);
        }}
      />
      </Content>
    </Layout>
  );
};

export default ExecutionManagement;
