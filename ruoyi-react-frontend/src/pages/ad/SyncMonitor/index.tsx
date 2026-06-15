import React, { useState, useEffect } from 'react';
import {
  Card,
  Row,
  Col,
  Statistic,
  Button,
  Table,
  Tag,
  message,
  Progress
} from 'antd';
import { ReloadOutlined, PlayCircleOutlined } from '@ant-design/icons';
import { post } from '@/lib/api';

interface SyncStatus {
  enabled: boolean;
  totalMappings: number;
  activeMappings: number;
  lastSyncAt?: string;
  lastSyncStatus?: string;
}

interface SyncLog {
  id: string;
  configId: string;
  configName: string;
  syncType: string;
  status: string;
  startTime: string;
  endTime?: string;
  successCount: number;
  failureCount: number;
  duration?: number;
  errorMessage?: string;
}

const SyncMonitor: React.FC = () => {
  const [status, setStatus] = useState<SyncStatus | null>(null);
  const [logs, setLogs] = useState<SyncLog[]>([]);
  const [loading, setLoading] = useState(false);
  const [syncing, setSyncing] = useState(false);

  // 获取同步状态
  const fetchStatus = async () => {
    try {
      const result = await post('/ad/groups/sync/status', {});
      setStatus(result.data as SyncStatus);
    } catch (error) {
      message.error('获取状态失败');
    }
  };

  // 获取同步日志
  const fetchLogs = async () => {
    setLoading(true);
    try {
      const result = await post('/ad/groups/sync/logs', {
        current: 1,
        pageSize: 10
      });
      setLogs((result.data as { list: SyncLog[] }).list);
    } catch (error) {
      message.error('获取日志失败');
    } finally {
      setLoading(false);
    }
  };

  // 手动触发同步
  const handleSync = async () => {
    setSyncing(true);
    try {
      await post('/ad/groups/sync', { configId: 'default' });
      message.success('同步已触发');
      fetchStatus();
      fetchLogs();
    } catch (error) {
      message.error('同步失败');
    } finally {
      setSyncing(false);
    }
  };

  useEffect(() => {
    fetchStatus();
    fetchLogs();
    const interval = setInterval(fetchStatus, 10000); // 10秒轮询
    return () => clearInterval(interval);
  }, []);

  const statusColumns = [
    { title: '配置名称', dataIndex: 'configName', key: 'configName' },
    { title: '同步类型', dataIndex: 'syncType', key: 'syncType' },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => {
        const color = status === 'success' ? 'success' : status === 'failed' ? 'error' : 'processing';
        return <Tag color={color}>{status}</Tag>;
      }
    },
    { title: '开始时间', dataIndex: 'startTime', key: 'startTime' },
    { title: '耗时(秒)', dataIndex: 'duration', key: 'duration' },
    { title: '成功数', dataIndex: 'successCount', key: 'successCount' },
    { title: '失败数', dataIndex: 'failureCount', key: 'failureCount' },
    { title: '错误信息', dataIndex: 'errorMessage', key: 'errorMessage', ellipsis: true }
  ];

  return (
    <div>
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={6}>
          <Card>
            <Statistic
              title="映射总数"
              value={status?.totalMappings || 0}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="活跃映射"
              value={status?.activeMappings || 0}
              styles={{ content: { color: '#3f8600' } }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="同步状态"
              value={status?.enabled ? '已启用' : '已禁用'}
              styles={{ content: { color: status?.enabled ? '#3f8600' : '#cf1322' } }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Button
              type="primary"
              icon={<PlayCircleOutlined />}
              onClick={handleSync}
              loading={syncing}
              size="large"
            >
              手动同步
            </Button>
          </Card>
        </Col>
      </Row>

      <Card
        title="同步历史"
        extra={
          <Button
            icon={<ReloadOutlined />}
            onClick={fetchLogs}
          >
            刷新
          </Button>
        }
      >
        <Table
          columns={statusColumns}
          dataSource={logs}
          rowKey="id"
          loading={loading}
          pagination={false}
        />
      </Card>
    </div>
  );
};

export default SyncMonitor;