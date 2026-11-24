/**
 * RPA 执行记录页面
 */

import { useState, useCallback, useEffect, useMemo, useRef } from "react";
import type { FC } from "react";
import { Table, Button, Space, Form, Card, Layout, Input, Modal, App } from "antd";
import { SearchOutlined, ReloadOutlined } from "@ant-design/icons";
import type { Execution } from "@/types/rpa";
import type { PageResponse } from "@/types/base";
import { useTableManager } from "@/hooks/useTableManager";
import { useRPAProgress } from "@/hooks/useRPAProgress";
import { useServerSort } from "@/hooks/useServerSort";
import { createSorterMeta } from "@/utils/tableHelpers";
import { usePagination } from "@/hooks/usePagination";
import { getExecutionColumns } from "./columns";
import { ExecutionDetailModal } from "./ExecutionDetailModal";
import { EXECUTION_STATUS_OPTIONS } from "../constants";
import { post } from "@/lib/api";
// Phase 71 (W3)：裸 Input/Select → 契约组件。
// 执行状态为固定枚举（业务禁搜）→ children 改写 options + showSearch={false}。
import { AppInput, AppSelect } from "@/components/ui";
import { FilterBar, StatCards } from "@/components/shared";

const { Content } = Layout;
const { TextArea } = Input;

const ExecutionManagement: FC = () => {
  const { message } = App.useApp();
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
  const {
    orderByColumn,
    isAsc,
    handleTableChange: handleExecSortChange,
    sortOrder: execSortOrder,
  } = useServerSort<Execution>({
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

  // CONTRACT-05（D16）：执行页不再 setInterval(5000) 轮询 running。
  // 改为订阅全局 WebSocket 的 RPA 进度事件（useRPAProgress → noticeStore
  // → NotificationBell 唯一连接），消息到达时刷新当前页与统计。
  //
  // 保留一个**低频**兜底刷新（30s）：WS 断线期间列表不能完全静默过期。
  // 这与「轮询驱动实时进度」是两件事 —— 实时性由 WS 提供，兜底只防断线。
  const executionsRef = useRef<Execution[]>([]);

  // 保持 ref 与 state 同步（WS 回调里需要判断当前页是否有运行中任务）
  useEffect(() => {
    executionsRef.current = executions;
  }, [executions]);

  const refreshData = useCallback(() => {
    loadExecutions();
  }, [loadExecutions]);

  // WebSocket 实时进度：任一执行有进度/终态事件 → 刷新当前页 + 统计
  const handleRealtimeProgress = useCallback(() => {
    loadExecutions();
  }, [loadExecutions]);

  useRPAProgress({
    onProgress: handleRealtimeProgress,
    onCompleted: handleRealtimeProgress,
    onFailed: handleRealtimeProgress,
  });

  useEffect(() => {
    loadExecutions();
    // WS 断线兜底：仅当本页存在 running 任务时才用 30s 低频对账，
    // 与旧 5s 轮询区分（旧实现是唯一数据来源，现在只是兜底）。
    const interval = setInterval(() => {
      if (executionsRef.current.some((e) => e.status === "running")) {
        loadExecutions();
      }
    }, 30000);
    return () => clearInterval(interval);
  }, [loadExecutions]);

  const handleViewDetail = useCallback((record: Execution) => {
    setSelectedExecution(record);
    setDetailModalVisible(true);
  }, []);

  // Phase 153 / RUNTIME-03：行内取消入口（归属协议第 4 条 —— 落本 handler，
  // 不进 ExecutionDetailModal footer）。二次确认 + 原因留痕（审计：谁/何时/原因）。
  const handleCancelExecution = useCallback(
    (record: Execution) => {
      let reason = "";
      Modal.confirm({
        title: "确认取消执行",
        content: (
          <div>
            <p>
              确定要取消任务「{record.taskName}」的执行吗？Worker 将在下一个动作边界停止。
            </p>
            <TextArea
              placeholder="取消原因（将记入审计日志）"
              rows={2}
              onChange={(e) => {
                reason = e.target.value;
              }}
            />
          </div>
        ),
        okText: "确认取消",
        okButtonProps: { danger: true },
        cancelText: "返回",
        onOk: async () => {
          try {
            await post(`/rpa/executions/${record.id}/cancel`, { reason });
            message.success("取消指令已下发，执行将在下一个动作边界停止");
            loadExecutions();
          } catch (_error) {
            // 错误已在 errorHandler 中处理
          }
        },
      });
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message/loadExecutions 由 hooks 稳定提供
    [loadExecutions]
  );

  const columns = useMemo(
    () =>
      getExecutionColumns({
        handleViewDetail,
        handleCancel: handleCancelExecution,
        getSortOrder: (field) =>
          orderByColumn === field ? ((execSortOrder ?? null) as "ascend" | "descend" | null) : null,
      }),
    [handleViewDetail, handleCancelExecution, orderByColumn, execSortOrder]
  );

  return (
    <Layout style={{ background: "transparent" }}>
      <Content style={{ padding: 0, background: "transparent" }}>
        <Card style={{ marginBottom: 16 }}>
          <FilterBar>
            <Form form={searchForm} layout="inline" style={{ flex: 1, minWidth: 0 }}>
              <Form.Item name="taskName" label="任务名称">
                <AppInput placeholder="请输入任务名称" style={{ width: 200 }} />
              </Form.Item>
              <Form.Item name="status" label="执行状态">
                {/* 形态分类器②④：状态枚举（业务禁搜），children → options */}
                <AppSelect
                  placeholder="请选择"
                  allowClear
                  style={{ width: 150 }}
                  showSearch={false}
                  options={EXECUTION_STATUS_OPTIONS}
                />
              </Form.Item>
              <Form.Item name="workerId" label="Worker">
                <AppInput placeholder="请输入Worker ID" style={{ width: 150 }} />
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
          </FilterBar>

          {/* 统计信息 */}
          <StatCards
            style={{
              marginTop: 12,
              paddingTop: 12,
              borderTop: "1px solid var(--theme-border-primary)",
            }}
            items={[
              { label: "总计", value: statistics.total, tone: "default" },
              { label: "运行中", value: statistics.running, tone: "default" },
              { label: "成功", value: statistics.success, tone: "green" },
              { label: "失败", value: statistics.failed, tone: "danger" },
            ]}
          />
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
              const searchParams: Record<string, unknown> = {
                current: newPage,
                pageSize: newPageSize,
                ...(orderByColumn ? { orderByColumn, isAsc } : {}),
              };
              Object.keys(formValues).forEach((key) => {
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
