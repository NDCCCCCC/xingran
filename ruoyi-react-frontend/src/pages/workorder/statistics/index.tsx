import React, { useState, useEffect } from 'react';
import { Card, Row, Col, Statistic, Table, Tag } from 'antd';
import {
  FileTextOutlined,
  ClockCircleOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  UserOutlined,
  TeamOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { getWorkOrderStatistics, type WorkOrderStatistics } from '@/lib/workorderApi';
import { WorkOrderStatus, WorkOrderPriority } from '@/lib/workorderApi';

const WorkOrderStatisticsPage: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [stats, setStats] = useState<WorkOrderStatistics | null>(null);

  const fetchStatistics = async () => {
    setLoading(true);
    try {
      const result = await getWorkOrderStatistics();
      setStats(result.data || null);
    } catch (error) {
      console.error('获取统计数据失败:', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchStatistics();
  }, []);

  const assigneeColumns: ColumnsType<any> = [
    { title: '处理人', dataIndex: 'assigneeName', key: 'assigneeName' },
    { title: '总工单', dataIndex: 'totalCount', key: 'totalCount' },
    { title: '待处理', dataIndex: 'pendingCount', key: 'pendingCount' },
    { title: '已完成', dataIndex: 'doneCount', key: 'doneCount' },
    {
      title: '平均处理时间',
      dataIndex: 'avgProcessTime',
      key: 'avgProcessTime',
      render: (time: number) => `${time.toFixed(1)} 小时`,
    },
  ];

  const departmentColumns: ColumnsType<any> = [
    { title: '部门', dataIndex: 'deptName', key: 'deptName' },
    { title: '总工单', dataIndex: 'totalCount', key: 'totalCount' },
    { title: '已完成', dataIndex: 'doneCount', key: 'doneCount' },
  ];

  const priorityConfig = {
    [WorkOrderPriority.Low]: { text: '低', color: 'default' },
    [WorkOrderPriority.Medium]: { text: '中', color: 'blue' },
    [WorkOrderPriority.High]: { text: '高', color: 'orange' },
    [WorkOrderPriority.Urgent]: { text: '紧急', color: 'red' },
  };

  return (
    <div className="p-6">
      {/* 基本统计卡片 */}
      <Row gutter={16} className="mb-6">
        <Col span={4}>
          <Card>
            <Statistic
              title="总工单"
              value={stats?.total || 0}
              prefix={<FileTextOutlined />}
              loading={loading}
            />
          </Card>
        </Col>
        <Col span={4}>
          <Card>
            <Statistic
              title="待处理"
              value={stats?.pending || 0}
              prefix={<ClockCircleOutlined />}
              styles={{ content: { color: '#faad14' } }}
              loading={loading}
            />
          </Card>
        </Col>
        <Col span={4}>
          <Card>
            <Statistic
              title="处理中"
              value={stats?.processing || 0}
              styles={{ content: { color: '#1890ff' } }}
              loading={loading}
            />
          </Card>
        </Col>
        <Col span={4}>
          <Card>
            <Statistic
              title="已完成"
              value={stats?.completed || 0}
              prefix={<CheckCircleOutlined />}
              styles={{ content: { color: '#52c41a' } }}
              loading={loading}
            />
          </Card>
        </Col>
        <Col span={4}>
          <Card>
            <Statistic
              title="已关闭"
              value={stats?.closed || 0}
              prefix={<CloseCircleOutlined />}
              styles={{ content: { color: '#8c8c8c' } }}
              loading={loading}
            />
          </Card>
        </Col>
        <Col span={4}>
          <Card>
            <Statistic
              title="平均处理时长"
              value={stats?.avgProcessTime?.toFixed(1) || 0}
              suffix="小时"
              loading={loading}
            />
          </Card>
        </Col>
      </Row>

      <Row gutter={16}>
        {/* 按优先级统计 */}
        <Col span={12}>
          <Card title="按优先级统计" loading={loading}>
            {stats && Object.entries(stats.byPriority).map(([key, value]) => (
              <div key={key} className="mb-4">
                <div className="flex justify-between items-center">
                  <span>
                    <Tag color={priorityConfig[key as WorkOrderPriority]?.color}>
                      {priorityConfig[key as WorkOrderPriority]?.text}
                    </Tag>
                  </span>
                  <Statistic value={value as number} />
                </div>
              </div>
            ))}
          </Card>
        </Col>

        {/* 按分类统计 */}
        <Col span={12}>
          <Card title="按分类统计" loading={loading}>
            {stats && Object.entries(stats.byCategory).map(([key, value]) => (
              <div key={key} className="mb-4">
                <div className="flex justify-between items-center">
                  <span>{key}</span>
                  <Statistic value={value as number} />
                </div>
              </div>
            ))}
            {(!stats || Object.keys(stats.byCategory).length === 0) && (
              <p className="text-gray-400 text-center">暂无数据</p>
            )}
          </Card>
        </Col>

        {/* 按处理人统计 */}
        <Col span={24}>
          <Card title="按处理人统计" loading={loading} className="mt-4">
            <Table
              columns={assigneeColumns}
              dataSource={stats?.byAssignee || []}
              rowKey="assigneeId"
              pagination={false}
              size="small"
            />
          </Card>
        </Col>

        {/* 按部门统计 */}
        <Col span={24}>
          <Card title="按部门统计" loading={loading} className="mt-4">
            <Table
              columns={departmentColumns}
              dataSource={stats?.byDepartment || []}
              rowKey="deptId"
              pagination={false}
              size="small"
            />
          </Card>
        </Col>

        {/* 趋势统计 */}
        <Col span={24}>
          <Card title="工单趋势（最近30天）" loading={loading} className="mt-4">
            {stats?.trend && stats.trend.length > 0 ? (
              <div className="h-64">
                {/* 这里可以使用图表库如 ECharts 或 Recharts 来绘制趋势图 */}
                <Table
                  columns={[
                    { title: '日期', dataIndex: 'date', key: 'date' },
                    { title: '工单数量', dataIndex: 'count', key: 'count' },
                  ]}
                  dataSource={stats.trend}
                  rowKey="date"
                  pagination={false}
                  size="small"
                />
              </div>
            ) : (
              <p className="text-gray-400 text-center">暂无数据</p>
            )}
          </Card>
        </Col>
      </Row>
    </div>
  );
};

export default WorkOrderStatisticsPage;
