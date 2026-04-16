/**
 * RPA Worker 监控页面
 */

import { useState, useCallback, useEffect } from 'react';
import type { FC } from 'react';
import {
  Table, Button, Space, Form, Input, Select, Card, Row, Col, Statistic,
  Tag, Alert, Progress,
} from 'antd';
import {
  SearchOutlined, ReloadOutlined, CloudServerOutlined,
  CheckCircleOutlined, CloseCircleOutlined, ClockCircleOutlined,
  LoadingOutlined,
} from '@ant-design/icons';
import type { Worker } from '@/types/rpa';
import { useTableManager } from '@/hooks/useTableManager';
import { usePagination } from '@/hooks/usePagination';
import { getWorkerColumns } from './columns';
import { WORKER_STATUS_OPTIONS } from '../constants';
import { post } from '@/lib/api';
import dayjs from 'dayjs';

const { Option } = Select;

// 心跳超时时间（秒）
const HEARTBEAT_TIMEOUT = 120; // 2分钟无心跳视为离线

// 检查 Worker 是否真正在线（基于最后心跳时间）
function isWorkerActuallyOnline(worker: Worker): boolean {
  if (!worker.lastHeartbeat) return false;
  const lastHeartbeatTime = dayjs.unix(worker.lastHeartbeat);
  const now = dayjs();
  const diffSeconds = now.diff(lastHeartbeatTime, 'second');
  return diffSeconds <= HEARTBEAT_TIMEOUT;
}

// 获取 Worker 的真实状态
function getWorkerActualStatus(worker: Worker): 'online' | 'offline' | 'busy' | 'error' {
  if (worker.status === 'busy' && isWorkerActuallyOnline(worker)) return 'busy';
  if (worker.status === 'error') return 'error';
  return isWorkerActuallyOnline(worker) ? 'online' : 'offline';
}

const WorkerMonitor: FC = () => {
  const { paginationProps, setCurrent, setPageSize, setTotal } = usePagination();

  const {
    loading,
    data: workers,
    total,
    searchForm,
    loadData: loadWorkers,
    handleSearch,
    handleReset,
  } = useTableManager<Worker>(
    async (params) => {
      const result = await post('/rpa/workers/list', params);
      // 类型断言：后端返回的数据结构
      const data = result.data as unknown as { list?: Worker[]; total?: number } | undefined;
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

  const columns = getWorkerColumns();

  // 统计数据（基于真实心跳状态）
  const stats = {
    total: workers.length,
    online: workers.filter(w => getWorkerActualStatus(w) === 'online').length,
    offline: workers.filter(w => getWorkerActualStatus(w) === 'offline').length,
    busy: workers.filter(w => getWorkerActualStatus(w) === 'busy').length,
    error: workers.filter(w => getWorkerActualStatus(w) === 'error').length,
    totalCapacity: workers.reduce((sum, w) => sum + (w.maxConcurrency || w.capabilities?.maxConcurrency || 3), 0),
    usedCapacity: workers.reduce((sum, w) => sum + (w.currentTasks || 0), 0),
  };

  const capacityUsagePercent = stats.totalCapacity > 0
    ? (stats.usedCapacity / stats.totalCapacity) * 100
    : 0;

  return (
    <div style={{ padding: 16, background: '#f0f2f5', minHeight: 'calc(100vh - 64px)' }}>
      {/* 统计卡片 */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={4}>
          <Card>
            <Statistic
              title="Worker 总数"
              value={stats.total}
              prefix={<CloudServerOutlined />}
              valueStyle={{ color: '#1890ff' }}
            />
          </Card>
        </Col>
        <Col span={5}>
          <Card>
            <Statistic
              title="在线"
              value={stats.online}
              prefix={<CheckCircleOutlined />}
              valueStyle={{ color: '#52c41a' }}
            />
          </Card>
        </Col>
        <Col span={5}>
          <Card>
            <Statistic
              title="忙碌"
              value={stats.busy}
              prefix={<LoadingOutlined />}
              valueStyle={{ color: '#1890ff' }}
            />
          </Card>
        </Col>
        <Col span={5}>
          <Card>
            <Statistic
              title="离线"
              value={stats.offline}
              prefix={<ClockCircleOutlined />}
              valueStyle={{ color: '#8c8c8c' }}
            />
          </Card>
        </Col>
        <Col span={5}>
          <Card>
            <Statistic
              title="错误"
              value={stats.error}
              prefix={<CloseCircleOutlined />}
              valueStyle={{ color: '#ff4d4f' }}
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
                <Tag color="blue">{stats.usedCapacity}/{stats.totalCapacity}</Tag>
              </Space>
            </div>
            <Progress
              percent={Math.round(capacityUsagePercent)}
              status={capacityUsagePercent > 80 ? 'exception' : 'active'}
            />
          </Col>
          <Col span={6}>
            <Statistic
              title="可用容量"
              value={stats.totalCapacity - stats.usedCapacity}
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
            <Select placeholder="请选择" allowClear style={{ width: 150 }}>
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
        {stats.offline > 0 && stats.offline === stats.total && (
          <Alert
            message="所有 Worker 离线"
            description="请检查 Worker 服务是否正常运行，以及网络连接是否正常"
            type="warning"
            showIcon
            style={{ marginBottom: 16 }}
          />
        )}
        {stats.error > 0 && (
          <Alert
            message={`${stats.error} 个 Worker 处于错误状态`}
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
          onChange={(pagination) => {
            const newPage = pagination.current ?? 1;
            const newPageSize = pagination.pageSize ?? 10;
            setCurrent(newPage);
            setPageSize(newPageSize);
            const formValues = searchForm.getFieldsValue() as Record<string, unknown>;
	            const searchParams: Record<string, unknown> = { current: newPage, pageSize: newPageSize };
	            Object.keys(formValues).forEach(key => {
	              const value = formValues[key];
	              if (value !== undefined && value !== null && value !== '') {
	                searchParams[key] = value;
	              }
	            });
	            loadWorkers(searchParams);
          }}
          rowClassName={(record) => {
            if (record.status === 'offline') return 'row-offline';
            if (record.status === 'error') return 'row-error';
            return '';
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
    </div>
  );
};

export default WorkerMonitor;
