/**
 * API 密钥使用日志和统计 Modal 组件
 */

import { useState, useEffect, useCallback, useMemo, type FC } from "react";
import { Modal, Table, Statistic, Card, Row, Col, Tag, Descriptions, Progress } from "antd";
import type { ColumnsType } from "antd/es/table";
import {
  FileTextOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  ClockCircleOutlined,
} from "@ant-design/icons";
import { listUsageLogs, getUsageSummary } from "@/api/apikey";
import type { APIKeyUsageLog, UsageSummary } from "@/types/apikey";
import type { PageData } from "@/types/apikey";
import { formatDateTime } from "@/utils/datetime";

// ==================== 组件属性 ====================

interface LogsModalProps {
  visible: boolean;
  apiKeyId: string;
  onClose: () => void;
}

// ==================== 工具函数 ====================

/**
 * 根据状态码获取 Tag 颜色
 */
function getStatusTagColor(statusCode: number): string {
  if (statusCode >= 200 && statusCode < 300) return "success";
  if (statusCode >= 300 && statusCode < 400) return "warning";
  if (statusCode >= 400 && statusCode < 500) return "error";
  if (statusCode >= 500) return "error";
  return "default";
}

/**
 * 根据状态码获取标签文本
 */
function getStatusTagLabel(statusCode: number): string {
  if (statusCode >= 200 && statusCode < 300) return "成功";
  if (statusCode >= 300 && statusCode < 400) return "重定向";
  if (statusCode >= 400 && statusCode < 500) return "客户端错误";
  if (statusCode >= 500) return "服务器错误";
  return "未知";
}

/**
 * 格式化 HTTP 方法标签
 */
function renderMethodTag(method: string) {
  const methodColors: Record<string, string> = {
    GET: "green",
    POST: "blue",
    PUT: "orange",
    DELETE: "red",
    PATCH: "purple",
  };

  return (
    <Tag color={methodColors[method] || "default"} style={{ fontWeight: "bold" }}>
      {method}
    </Tag>
  );
}

// ==================== 主组件 ====================

const LogsModal: FC<LogsModalProps> = ({ visible, apiKeyId, onClose }) => {
  // ==================== 状态管理 ====================
  const [logs, setLogs] = useState<APIKeyUsageLog[]>([]);
  const [summary, setSummary] = useState<UsageSummary | null>(null);
  const [loading, setLoading] = useState<boolean>(false);
  const [pagination, setPagination] = useState({
    current: 1,
    pageSize: 20,
    total: 0,
  });

  // ==================== 稳定的查询参数 ====================
  const queryParams = useMemo(
    () => ({
      current: pagination.current,
      pageSize: pagination.pageSize,
    }),
    [pagination.current, pagination.pageSize]
  );

  // ==================== 数据加载 ====================

  /**
   * 加载使用日志
   */
  const loadLogs = useCallback(async () => {
    if (!visible || !apiKeyId) {
      return;
    }

    setLoading(true);
    try {
      const result = await listUsageLogs(apiKeyId, queryParams);
      setLogs(result.data?.list || []);
      setPagination((prev) => ({
        ...prev,
        total: result.data?.total || 0,
      }));
    } catch (error) {
      console.error("加载使用日志失败:", error);
    } finally {
      setLoading(false);
    }
  }, [visible, apiKeyId, queryParams]);

  /**
   * 加载统计汇总
   */
  const loadSummary = useCallback(async () => {
    if (!visible || !apiKeyId) {
      return;
    }

    try {
      const result = await getUsageSummary(apiKeyId);
      setSummary(result.data || null);
    } catch (error) {
      console.error("加载统计汇总失败:", error);
    }
  }, [visible, apiKeyId]);

  /**
   * 当 Modal 打开时加载数据
   */
  useEffect(() => {
    if (visible && apiKeyId) {
      loadLogs();
      loadSummary();
    }
  }, [visible, apiKeyId, loadLogs, loadSummary]);

  // ==================== 事件处理 ====================

  /**
   * 分页变化
   */
  const handleTableChange = useCallback((newPagination: any) => {
    setPagination({
      current: newPagination.current || 1,
      pageSize: newPagination.pageSize || 20,
      total: newPagination.total || 0,
    });
  }, []);

  // ==================== 表格列定义 ====================

  const columns: ColumnsType<APIKeyUsageLog> = useMemo(
    () => [
      {
        title: "时间",
        dataIndex: "created_at",
        key: "created_at",
        width: 160,
        render: (text) => formatDateTime(text),
      },
      {
        title: "方法",
        dataIndex: "method",
        key: "method",
        width: 80,
        align: "center" as const,
        render: (method: string) => renderMethodTag(method),
      },
      {
        title: "路径",
        dataIndex: "path",
        key: "path",
        width: 200,
        ellipsis: true,
      },
      {
        title: "状态码",
        dataIndex: "status_code",
        key: "status_code",
        width: 100,
        align: "center" as const,
        render: (statusCode: number) => (
          <Tag color={getStatusTagColor(statusCode)}>{statusCode}</Tag>
        ),
      },
      {
        title: "成功",
        dataIndex: "success",
        key: "success",
        width: 80,
        align: "center" as const,
        render: (success: boolean) => (
          <Tag color={success ? "success" : "error"} style={{ fontWeight: "bold" }}>
            {success ? "是" : "否"}
          </Tag>
        ),
      },
      {
        title: "客户端 IP",
        dataIndex: "client_ip",
        key: "client_ip",
        width: 140,
      },
      {
        title: "耗时",
        dataIndex: "duration",
        key: "duration",
        width: 100,
        align: "right" as const,
        render: (duration: number) => `${duration}ms`,
      },
    ],
    []
  );

  // ==================== 渲染 ====================

  return (
    <Modal
      title="使用日志"
      open={visible}
      onCancel={onClose}
      width={1200}
      footer={null}
      destroyOnHidden
    >
      {summary && (
        <div style={{ marginBottom: 16 }}>
          {/* 统计概览 */}
          <Row gutter={16} style={{ marginBottom: 16 }}>
            <Col span={6}>
              <Card>
                <Statistic
                  title="总请求数"
                  value={summary.total_requests}
                  prefix={<FileTextOutlined />}
                />
              </Card>
            </Col>
            <Col span={6}>
              <Card>
                <Statistic
                  title="成功率"
                  value={(summary.success_rate * 100).toFixed(2)}
                  suffix="%"
                  styles={{
                    content: {
                      color:
                        summary.success_rate >= 0.95
                          ? "var(--theme-success, #3f8600)"
                          : summary.success_rate >= 0.8
                            ? "var(--theme-warning, #faad14)"
                            : "var(--theme-error, #cf1322)",
                    },
                  }}
                  prefix={
                    summary.success_rate >= 0.95 ? (
                      <CheckCircleOutlined />
                    ) : summary.success_rate >= 0.8 ? (
                      <ClockCircleOutlined />
                    ) : (
                      <CloseCircleOutlined />
                    )
                  }
                />
              </Card>
            </Col>
            <Col span={6}>
              <Card>
                <Statistic
                  title="平均耗时"
                  value={summary.avg_duration.toFixed(2)}
                  suffix="ms"
                  styles={{
                    content: {
                      color:
                        summary.avg_duration < 100
                          ? "var(--theme-success, #3f8600)"
                          : summary.avg_duration < 500
                            ? "var(--theme-warning, #faad14)"
                            : "var(--theme-error, #cf1322)",
                    },
                  }}
                />
              </Card>
            </Col>
            <Col span={6}>
              <Card>
                <Statistic
                  title="错误数"
                  value={
                    summary.total_requests -
                    Math.floor(summary.total_requests * summary.success_rate)
                  }
                  styles={{
                    content: {
                      color: "var(--theme-error, #cf1322)",
                    },
                  }}
                />
              </Card>
            </Col>
          </Row>

          {/* 详细统计 */}
          <Row gutter={16}>
            {/* 按方法统计 */}
            <Col span={8}>
              <Card title="按方法统计" size="small">
                <Descriptions column={1} size="small">
                  {Object.entries(summary.requests_by_method).map(([method, count]) => (
                    <Descriptions.Item
                      key={method}
                      label={renderMethodTag(method)}
                      contentStyle={{ textAlign: "right" }}
                    >
                      {count} 次
                    </Descriptions.Item>
                  ))}
                </Descriptions>
                {Object.keys(summary.requests_by_method).length === 0 && (
                  <div
                    style={{
                      textAlign: "center",
                      color: "var(--theme-text-tertiary, #999)",
                      padding: "8px 0",
                    }}
                  >
                    暂无数据
                  </div>
                )}
              </Card>
            </Col>

            {/* 按路径统计（Top 5） */}
            <Col span={8}>
              <Card title="热门路径（Top 5）" size="small">
                <div style={{ maxHeight: 200, overflowY: "auto" }}>
                  {Object.entries(summary.requests_by_path)
                    .sort(([, a], [, b]) => (b as number) - (a as number))
                    .slice(0, 5)
                    .map(([path, count], index) => {
                      const maxCount = Math.max(...Object.values(summary.requests_by_path));
                      const percentage = ((count as number) / maxCount) * 100;

                      return (
                        <div key={path} style={{ marginBottom: 12 }}>
                          <div
                            style={{
                              display: "flex",
                              justifyContent: "space-between",
                              marginBottom: 4,
                            }}
                          >
                            <span
                              style={{ fontSize: 12, color: "var(--theme-text-tertiary, #666)" }}
                            >
                              {index + 1}. {path.length > 30 ? `${path.slice(0, 30)}...` : path}
                            </span>
                            <span style={{ fontSize: 12, fontWeight: "bold" }}>{count} 次</span>
                          </div>
                          <Progress
                            percent={percentage}
                            showInfo={false}
                            strokeColor="var(--theme-info, #1890ff)"
                            trailColor="#f0f0f0"
                            size="small"
                          />
                        </div>
                      );
                    })}
                  {Object.keys(summary.requests_by_path).length === 0 && (
                    <div
                      style={{
                        textAlign: "center",
                        color: "var(--theme-text-tertiary, #999)",
                        padding: "8px 0",
                      }}
                    >
                      暂无数据
                    </div>
                  )}
                </div>
              </Card>
            </Col>

            {/* 错误统计 */}
            <Col span={8}>
              <Card title="错误统计" size="small">
                <div style={{ maxHeight: 200, overflowY: "auto" }}>
                  {Object.entries(summary.errors_by_status)
                    .sort(([, a], [, b]) => (b as number) - (a as number))
                    .map(([status, count]) => (
                      <div key={status} style={{ marginBottom: 8 }}>
                        <Tag color={getStatusTagColor(parseInt(status))}>{status}</Tag>
                        <span style={{ marginLeft: 8 }}>{count} 次</span>
                      </div>
                    ))}
                  {Object.keys(summary.errors_by_status).length === 0 && (
                    <div
                      style={{
                        textAlign: "center",
                        color: "var(--theme-text-tertiary, #999)",
                        padding: "8px 0",
                      }}
                    >
                      暂无错误
                    </div>
                  )}
                </div>
              </Card>
            </Col>
          </Row>
        </div>
      )}

      {/* 日志表格 */}
      <Table
        columns={columns}
        dataSource={logs}
        rowKey="id"
        loading={loading}
        pagination={{
          current: pagination.current,
          pageSize: pagination.pageSize,
          total: pagination.total,
          showSizeChanger: true,
          showQuickJumper: true,
          showTotal: (total) => `共 ${total} 条`,
        }}
        onChange={handleTableChange}
        scroll={{ x: 1000 }}
        size="small"
      />
    </Modal>
  );
};

export default LogsModal;
