import React, { useState, useEffect } from "react";
import { useLocation } from "react-router-dom";
import { usePersistedStateController } from "@/hooks/usePersistedState";
import { App, Card, Table, Tag, Space, Button, Select } from "antd";
import {
  ReloadOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  LoadingOutlined,
} from "@ant-design/icons";
import type { ColumnsType } from "antd/es/table";
import { getADSyncLogs, getADConfigList, type ADSyncLog, type ADConfig } from "@/lib/adDomainApi";

const ADSyncLogsPage: React.FC = () => {
  const { message } = App.useApp();
  const [logs, setLogs] = useState<ADSyncLog[]>([]);
  const [configs, setConfigs] = useState<ADConfig[]>([]);
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);
  const [current, setCurrent] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const location = useLocation();
  const [selectedConfigId, setSelectedConfigId] = usePersistedStateController<string | undefined>({
    keyPrefix: location.pathname,
    keySuffix: "selectedConfigId",
    defaultValue: undefined,
  });

  useEffect(() => {
    fetchConfigs();
    fetchLogs();
    // 仅在分页或筛选条件变化时刷新;函数引用变化不应触发
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [current, pageSize, selectedConfigId]);

  const fetchConfigs = async () => {
    try {
      const res = await getADConfigList({ current: 1, pageSize: 100 });
      if (res.code === 0) {
        setConfigs(res.data?.list ?? []);
      }
    } catch {
      message.error("获取AD配置失败");
    }
  };

  const fetchLogs = async () => {
    setLoading(true);
    try {
      const res = await getADSyncLogs(selectedConfigId ?? "", {
        current,
        pageSize,
      });
      if (res.code === 0) {
        setLogs(res.data?.list ?? []);
        setTotal(res.data?.total ?? 0);
      }
    } catch {
      message.error("获取同步日志失败");
    } finally {
      setLoading(false);
    }
  };

  const getStatusTag = (status: string) => {
    switch (status) {
      case "running":
        return (
          <Tag icon={<LoadingOutlined />} color="processing">
            运行中
          </Tag>
        );
      case "success":
        return (
          <Tag icon={<CheckCircleOutlined />} color="success">
            成功
          </Tag>
        );
      case "failed":
        return (
          <Tag icon={<CloseCircleOutlined />} color="error">
            失败
          </Tag>
        );
      default:
        return <Tag>{status}</Tag>;
    }
  };

  const getSyncTypeTag = (type: string) => {
    switch (type) {
      case "full":
        return <Tag color="blue">全量同步</Tag>;
      case "incremental":
        return <Tag color="green">增量同步</Tag>;
      default:
        return <Tag>{type}</Tag>;
    }
  };

  const columns: ColumnsType<ADSyncLog> = [
    {
      title: "同步类型",
      dataIndex: "syncType",
      key: "syncType",
      width: 100,
      render: (type: string) => getSyncTypeTag(type),
    },
    {
      title: "同步状态",
      dataIndex: "syncStatus",
      key: "syncStatus",
      width: 100,
      render: (status: string) => getStatusTag(status),
    },
    {
      title: "开始时间",
      dataIndex: "startTime",
      key: "startTime",
      width: 180,
      render: (text: string) => new Date(text).toLocaleString("zh-CN"),
    },
    {
      title: "结束时间",
      dataIndex: "endTime",
      key: "endTime",
      width: 180,
      render: (text: string | undefined) => (text ? new Date(text).toLocaleString("zh-CN") : "-"),
    },
    {
      title: "耗时(秒)",
      dataIndex: "duration",
      key: "duration",
      width: 100,
      render: (duration: number | undefined) => (duration ? `${duration}s` : "-"),
    },
    {
      title: "OU数量",
      dataIndex: "ouCount",
      key: "ouCount",
      width: 80,
    },
    {
      title: "用户组数量",
      dataIndex: "groupCount",
      key: "groupCount",
      width: 100,
    },
    {
      title: "用户数量",
      dataIndex: "userCount",
      key: "userCount",
      width: 80,
    },
    {
      title: "错误数量",
      dataIndex: "errorCount",
      key: "errorCount",
      width: 80,
      render: (count: number) => <Tag color={count > 0 ? "error" : "default"}>{count}</Tag>,
    },
    {
      title: "错误信息",
      dataIndex: "errorMessage",
      key: "errorMessage",
      ellipsis: true,
      render: (msg: string | undefined) =>
        msg ? <span style={{ color: "#ff4d4f" }}>{msg}</span> : "-",
    },
  ];

  return (
    <Card>
      <div style={{ marginBottom: 16 }}>
        <Space>
          <Select
            placeholder="选择AD配置"
            style={{ width: 200 }}
            allowClear
            value={selectedConfigId}
            onChange={setSelectedConfigId}
            onSearch={() => {}}
          >
            {configs.map((config) => (
              <Select.Option key={config.id} value={config.id}>
                {config.configName}
              </Select.Option>
            ))}
          </Select>
          <Button icon={<ReloadOutlined />} onClick={fetchLogs}>
            刷新
          </Button>
        </Space>
      </div>

      <Table
        columns={columns}
        dataSource={logs}
        loading={loading}
        rowKey="id"
        pagination={{
          current,
          pageSize,
          total,
          showSizeChanger: true,
          showTotal: (t) => `共 ${t} 条`,
          onChange: (page, size) => {
            setCurrent(page);
            setPageSize(size);
          },
        }}
      />
    </Card>
  );
};

export default ADSyncLogsPage;
