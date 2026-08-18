import { useState, useEffect } from "react";
import { Card, Row, Col, Statistic, Table, Tag } from "antd";
import {
  FileTextOutlined,
  ClockCircleOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
} from "@ant-design/icons";
import type { ColumnsType } from "antd/es/table";
import { getWorkOrderStatistics, type WorkOrderStatistics } from "@/lib/workorderApi";
import { WorkOrderPriority } from "@/lib/workorderApi";

import type { FC } from "react";
import { createSorter } from "@/utils/tableHelpers";

const WorkOrderStatisticsPage: FC = () => {
  const [loading, setLoading] = useState(false);
  const [stats, setStats] = useState<WorkOrderStatistics | null>(null);

  const fetchStatistics = async () => {
    setLoading(true);
    try {
      const result = await getWorkOrderStatistics();
      setStats(result.data || null);
    } catch (error) {
      console.error("获取统计数据失败:", error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchStatistics();
  }, []);

  const assigneeColumns: ColumnsType<{
    assigneeName: string;
    assigneeId: string;
    totalCount: number;
    pendingCount: number;
    doneCount: number;
    avgProcessTime: number;
  }> = [
    {
      title: "处理人",
      dataIndex: "assigneeName",
      key: "assigneeName",
      sorter: createSorter<{
        assigneeName: string;
        assigneeId: string;
        totalCount: number;
        pendingCount: number;
        doneCount: number;
        avgProcessTime: number;
      }>("assigneeName", "string"),
    },
    {
      title: "总工单",
      dataIndex: "totalCount",
      key: "totalCount",
      sorter: createSorter<{
        assigneeName: string;
        assigneeId: string;
        totalCount: number;
        pendingCount: number;
        doneCount: number;
        avgProcessTime: number;
      }>("totalCount", "number"),
    },
    {
      title: "待处理",
      dataIndex: "pendingCount",
      key: "pendingCount",
      sorter: createSorter<{
        assigneeName: string;
        assigneeId: string;
        totalCount: number;
        pendingCount: number;
        doneCount: number;
        avgProcessTime: number;
      }>("pendingCount", "number"),
    },
    {
      title: "已完成",
      dataIndex: "doneCount",
      key: "doneCount",
      sorter: createSorter<{
        assigneeName: string;
        assigneeId: string;
        totalCount: number;
        pendingCount: number;
        doneCount: number;
        avgProcessTime: number;
      }>("doneCount", "number"),
    },
    {
      title: "平均处理时间",
      dataIndex: "avgProcessTime",
      key: "avgProcessTime",
      sorter: createSorter<{
        assigneeName: string;
        assigneeId: string;
        totalCount: number;
        pendingCount: number;
        doneCount: number;
        avgProcessTime: number;
      }>("avgProcessTime", "number"),
      render: (time: number) => `${time.toFixed(1)} 小时`,
    },
  ];

  const departmentColumns: ColumnsType<{
    deptName: string;
    deptId: string;
    totalCount: number;
    doneCount: number;
  }> = [
    {
      title: "部门",
      dataIndex: "deptName",
      key: "deptName",
      sorter: createSorter<{
        deptName: string;
        deptId: string;
        totalCount: number;
        doneCount: number;
      }>("deptName", "string"),
    },
    {
      title: "总工单",
      dataIndex: "totalCount",
      key: "totalCount",
      sorter: createSorter<{
        deptName: string;
        deptId: string;
        totalCount: number;
        doneCount: number;
      }>("totalCount", "number"),
    },
    {
      title: "已完成",
      dataIndex: "doneCount",
      key: "doneCount",
      sorter: createSorter<{
        deptName: string;
        deptId: string;
        totalCount: number;
        doneCount: number;
      }>("doneCount", "number"),
    },
  ];

  const priorityConfig = {
    [WorkOrderPriority.Low]: { text: "低", color: "default" },
    [WorkOrderPriority.Medium]: { text: "中", color: "blue" },
    [WorkOrderPriority.High]: { text: "高", color: "orange" },
    [WorkOrderPriority.Urgent]: { text: "紧急", color: "red" },
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
              styles={{ content: { color: "var(--theme-warning, #b07a20)" } }}
              loading={loading}
            />
          </Card>
        </Col>
        <Col span={4}>
          <Card>
            <Statistic
              title="处理中"
              value={stats?.processing || 0}
              styles={{ content: { color: "var(--theme-info, #337ab0)" } }}
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
              styles={{ content: { color: "var(--theme-success, #2d8949)" } }}
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
              styles={{ content: { color: "var(--theme-text-tertiary, #707068)" } }}
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
            {stats &&
              Object.entries(stats.byPriority).map(([key, value]) => {
                const priority = key as unknown as WorkOrderPriority;
                return (
                  <div key={key} className="mb-4">
                    <div className="flex justify-between items-center">
                      <span>
                        <Tag color={priorityConfig[priority]?.color}>
                          {priorityConfig[priority]?.text}
                        </Tag>
                      </span>
                      <Statistic value={Number(value)} />
                    </div>
                  </div>
                );
              })}
          </Card>
        </Col>

        {/* 按分类统计 */}
        <Col span={12}>
          <Card title="按分类统计" loading={loading}>
            {stats &&
              Object.entries(stats.byCategory).map(([key, value]) => (
                <div key={key} className="mb-4">
                  <div className="flex justify-between items-center">
                    <span>{key}</span>
                    <Statistic value={Number(value)} />
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
                    { title: "日期", dataIndex: "date", key: "date" },
                    { title: "工单数量", dataIndex: "count", key: "count" },
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
