/**
 * RPA 执行记录页面
 */

import { useState, useCallback, useEffect, useMemo } from 'react';
import type { FC } from 'react';
import {
  Table, Button, Space, Form, Input, Select, Card, Tag,
} from 'antd';
import {
  SearchOutlined, ReloadOutlined, FilterOutlined,
} from '@ant-design/icons';
import type { Execution } from '@/types/rpa';
import type { PageResponse } from '@/types/base';
import { useTableManager } from '@/hooks/useTableManager';
import { usePagination } from '@/hooks/usePagination';
import { getExecutionColumns } from './columns';
import { ExecutionDetailModal } from './ExecutionDetailModal';
import { EXECUTION_STATUS_OPTIONS } from '../constants';
import { post } from '@/lib/api';

const { Option } = Select;

const ExecutionManagement: FC = () => {
  const { paginationProps, setCurrent, setPageSize, setTotal } = usePagination();

  const {
    loading,
    data: executions,
    total,
    searchForm,
    loadData: loadExecutions,
    handleSearch,
    handleReset,
  } = useTableManager<Execution>(
    async (params) => {
      const result = await post<PageResponse<Execution>>('/rpa/executions/list', params);
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

  const columns = getExecutionColumns({
    handleViewDetail,
  });

  // 统计数据 - 使用 useMemo 缓存计算结果
  // 遵循 Vercel React Best Practices: rerender-memo
  // 避免每次渲染时重新计算统计数据
  const stats = useMemo(() => ({
    total: executions.length,
    running: executions.filter(e => e.status === 'running').length,
    completed: executions.filter(e => e.status === 'completed').length,
    failed: executions.filter(e => e.status === 'failed').length,
  }), [executions]);

  return (
    <div style={{ padding: 16, background: '#f0f2f5', minHeight: 'calc(100vh - 64px)' }}>
      <Card style={{ marginBottom: 16 }}>
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'flex-start',
            flexWrap: 'wrap',
            gap: '16px',
          }}
        >
          <Form form={searchForm} layout="inline" style={{ flex: 1, minWidth: 0 }}>
            <Form.Item name="taskName" label="任务名称">
              <Input placeholder="请输入任务名称" allowClear style={{ width: 200 }} />
            </Form.Item>
            <Form.Item name="status" label="执行状态">
              <Select placeholder="请选择" allowClear style={{ width: 150 }}>
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
        <Space style={{ marginTop: 12, paddingTop: 12, borderTop: '1px solid #f0f0f0' }}>
          <Tag icon={<FilterOutlined />} color="blue">
            总计: {stats.total}
          </Tag>
          <Tag color="processing">运行中: {stats.running}</Tag>
          <Tag color="success">成功: {stats.completed}</Tag>
          <Tag color="error">失败: {stats.failed}</Tag>
        </Space>
      </Card>

      <Card>
        <Table
          columns={columns}
          dataSource={executions}
          loading={loading}
          rowKey="id"
          pagination={paginationProps}
          onChange={(pagination) => {
            const newPage = pagination.current ?? 1;
            const newPageSize = pagination.pageSize ?? 10;
            setCurrent(newPage);
            setPageSize(newPageSize);
            loadExecutions({ current: newPage, pageSize: newPageSize });
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
    </div>
  );
};

export default ExecutionManagement;
