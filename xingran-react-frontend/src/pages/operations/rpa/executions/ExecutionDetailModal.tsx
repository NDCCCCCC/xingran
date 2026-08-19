/**
 * RPA 执行详情弹窗
 * 显示执行日志、截图和步骤详情
 */

import { useState, useCallback, useEffect, useMemo } from "react";
import {
  App,
  Modal,
  Descriptions,
  Steps,
  Card,
  Timeline,
  Image,
  Space,
  Tag,
  Alert,
  Typography,
  Spin,
  Empty,
  Button,
  Tabs,
} from "antd";
import {
  FileTextOutlined,
  CameraOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  ClockCircleOutlined,
  DownloadOutlined,
} from "@ant-design/icons";
import type { Execution, ExecutionLog } from "@/types/rpa";
import { renderExecutionStatusTag } from "../constants";
import { post } from "@/lib/api";
import { getAuthHeaders } from "@/utils/authHelpers";

const { Text } = Typography;

export interface ExecutionDetailModalProps {
  open: boolean;
  execution: Execution | null;
  onClose: () => void;
}

export function ExecutionDetailModal({ open, execution, onClose }: ExecutionDetailModalProps) {
  const { message } = App.useApp();
  const [logs, setLogs] = useState<ExecutionLog[]>([]);
  const [logsLoading, setLogsLoading] = useState(false);
  const [activeTab, setActiveTab] = useState("steps");

  const loadLogs = useCallback(async (executionId: string) => {
    setLogsLoading(true);
    try {
      const result = await post(`/rpa/executions/${executionId}/logs`, {});
      // 确保 logs 是数组
      const responseLogs = (result.data as any)?.logs;
      setLogs(Array.isArray(responseLogs) ? responseLogs : []);
    } catch (error) {
      console.error("加载日志失败:", error);
      setLogs([]);
    } finally {
      setLogsLoading(false);
    }
  }, []);

  useEffect(() => {
    if (
      open &&
      execution?.id &&
      (execution.status === "running" || execution.status === "pending")
    ) {
      loadLogs(execution.id);
      // 每3秒刷新一次日志
      const interval = setInterval(() => {
        loadLogs(execution.id);
      }, 3000);
      return () => clearInterval(interval);
    } else if (open && execution) {
      // 确保 execution.logs 是数组
      const execLogs = (execution as any).logs;
      setLogs(Array.isArray(execLogs) ? execLogs : []);
    } else {
      setLogs([]);
    }
  }, [open, execution, loadLogs]);

  // 确保 screenshots 是数组并添加正确的 URL 前缀
  const screenshots = useMemo(() => {
    const execScreenshots = execution?.screenshots;
    let screenshotList: string[] = [];

    if (Array.isArray(execScreenshots)) {
      screenshotList = execScreenshots;
    } else if (typeof execScreenshots === "string" && execScreenshots) {
      // 如果是 JSON 字符串，尝试解析
      try {
        const parsed = JSON.parse(execScreenshots);
        screenshotList = Array.isArray(parsed) ? parsed : [];
      } catch {
        // 如果解析失败，可能是单个路径字符串
        screenshotList = [execScreenshots];
      }
    }

    // 为每个截图 URL 拼接正确的前缀。前缀取 Vite BASE_URL,
    // dev 为 '/' 行为完全等同旧逻辑；prod 为 '/xingran/' 时产出
    // /xingran/uploads/rpa/screenshots/xxx.png,与 nginx 子路径部署一致。
    // 兼容已有前缀('/uploads/' 或 '<BASE_URL>uploads/'):剥掉后重新拼接,
    // 避免产生 '/xingran/uploads/uploads/...' 这种重复前缀。
    const uploadsBase = `${import.meta.env.BASE_URL}uploads/`;
    return screenshotList.map((url) => {
      if (url.startsWith("http://") || url.startsWith("https://")) {
        return url; // 已经是完整 URL
      }
      // 取 "/uploads/" 最后一次出现之后的部分作为相对路径
      const idx = url.lastIndexOf("/uploads/");
      const rel = idx >= 0 ? url.slice(idx + "/uploads/".length) : url.replace(/^\//, "");
      return `${uploadsBase}${rel}`;
    });
  }, [execution?.screenshots]);

  const renderStatusIcon = (level: string) => {
    switch (level) {
      case "info":
        return <CheckCircleOutlined style={{ color: "var(--theme-success, #2d8949)" }} />;
      case "error":
        return <CloseCircleOutlined style={{ color: "var(--theme-error, #ba3630)" }} />;
      case "warn":
        return <CloseCircleOutlined style={{ color: "var(--theme-warning, #b07a20)" }} />;
      default:
        return <ClockCircleOutlined style={{ color: "var(--theme-text-tertiary, #707068)" }} />;
    }
  };

  const renderSteps = () => {
    if (!execution) return null;

    const steps = [];
    for (let i = 0; i < (execution.totalSteps || 0); i++) {
      const currentStep = execution.step || 0;
      let status: "wait" | "process" | "finish" | "error" = "wait";
      if (i < currentStep) status = "finish";
      else if (i === currentStep && execution.status === "running") status = "process";
      else if (i === currentStep && execution.status === "failed") status = "error";

      steps.push({
        title: `步骤 ${i + 1}`,
        status,
      });
    }

    return (
      <Steps
        current={execution.step ?? 0}
        status={
          execution.status === "failed"
            ? "error"
            : execution.status === "completed"
              ? "finish"
              : execution.status === "running"
                ? "process"
                : "wait"
        }
        items={steps}
      />
    );
  };

  const renderLogs = () => {
    if (logsLoading) {
      return (
        <div style={{ textAlign: "center", padding: "40px 0" }}>
          <Spin size="large" />
          <div style={{ marginTop: 16 }}>加载日志中...</div>
        </div>
      );
    }

    if (logs.length === 0) {
      return <Empty description="暂无日志" />;
    }

    return (
      <Timeline
        items={logs.map((log, index) => ({
          color: log.level === "error" ? "red" : log.level === "warn" ? "orange" : "blue",
          dot: renderStatusIcon(log.level),
          children: (
            <Card key={index} size="small" variant="outlined" style={{ marginBottom: 8 }}>
              <Space direction="vertical" style={{ width: "100%" }}>
                <Space>
                  <Text type="secondary">{log.timestamp}</Text>
                  <Tag
                    color={log.level === "error" ? "red" : log.level === "warn" ? "orange" : "blue"}
                  >
                    {log.level.toUpperCase()}
                  </Tag>
                  {log.step !== undefined && <Tag>步骤 {log.step + 1}</Tag>}
                </Space>
                <Text>{log.message}</Text>
                {log.detail && (
                  <Text type="secondary" style={{ fontSize: 12 }}>
                    {log.detail}
                  </Text>
                )}
                {log.screenshotUrl && (
                  <div>
                    <Image width={200} src={log.screenshotUrl} style={{ marginTop: 8 }} />
                  </div>
                )}
              </Space>
            </Card>
          ),
        }))}
      />
    );
  };

  const renderScreenshots = () => {
    // 合并来自 execution.screenshots 和 logs 中的截图
    const executionScreenshotItems = screenshots.map((url, index) => ({
      key: `exec-${index}`,
      image: url,
      message: "执行截图",
    }));

    const logScreenshots = logs
      .filter((log) => log.screenshotUrl)
      .map((log, index) => ({
        key: `log-${index}`,
        timestamp: log.timestamp,
        step: log.step,
        image: log.screenshotUrl,
        message: log.message,
      }));

    const allScreenshots = [...executionScreenshotItems, ...logScreenshots];

    if (allScreenshots.length === 0) {
      return <Empty description="暂无截图" />;
    }

    return (
      <Space direction="vertical" style={{ width: "100%" }}>
        {allScreenshots.map((item: any) => (
          <Card key={item.key} size="small" variant="outlined">
            <Space direction="vertical" style={{ width: "100%" }}>
              {item.timestamp && (
                <Space>
                  <Text type="secondary">{item.timestamp}</Text>
                  {item.step !== undefined && <Tag>步骤 {item.step + 1}</Tag>}
                </Space>
              )}
              {item.message && <Text type="secondary">{item.message}</Text>}
              <Image src={item.image} width="100%" />
            </Space>
          </Card>
        ))}
      </Space>
    );
  };

  // 下载执行产物（ZIP）
  const handleDownloadArtifacts = useCallback(async () => {
    if (!execution?.id) return;

    try {
      // 使用 fetch + Authorization 头获取 blob，避免 window.location.href 走 GET 缺失鉴权头
      const baseUrl = import.meta.env.VITE_API_BASE_URL || "";
      const headers = await getAuthHeaders();
      const response = await fetch(`${baseUrl}/rpa/executions/${execution.id}/download`, {
        headers,
      });

      if (!response.ok) {
        if (response.status === 401 || response.status === 403) {
          message.error("登录已过期，请重新登录");
        } else {
          message.error("下载失败");
        }
        return;
      }

      const blob = await response.blob();
      const blobUrl = window.URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = blobUrl;
      a.download = `execution_${execution.id}.zip`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(blobUrl);
      document.body.removeChild(a);

      message.success("开始下载执行产物...");
    } catch (error) {
      message.error("下载失败");
      console.error("下载执行产物失败:", error);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
  }, [execution?.id]);

  if (!execution) return null;

  return (
    <Modal
      title="执行详情"
      open={open}
      onCancel={onClose}
      width={900}
      destroyOnHidden
      footer={[
        <Button
          key="download"
          icon={<DownloadOutlined />}
          onClick={handleDownloadArtifacts}
          disabled={screenshots.length === 0 && !execution?.logs}
        >
          下载执行产物
        </Button>,
        <Button key="close" onClick={onClose}>
          关闭
        </Button>,
      ]}
    >
      <Descriptions column={2} bordered size="small" style={{ marginBottom: 16 }}>
        <Descriptions.Item label="任务名称">{execution.taskName}</Descriptions.Item>
        <Descriptions.Item label="执行状态">
          {renderExecutionStatusTag(execution.status || "pending")}
        </Descriptions.Item>
        <Descriptions.Item label="Worker">{execution.workerName || "-"}</Descriptions.Item>
        <Descriptions.Item label="当前步骤">
          {execution.totalSteps ? `${execution.step ?? 0}/${execution.totalSteps}` : "-"}
        </Descriptions.Item>
        <Descriptions.Item label="开始时间">{execution.startedAt || "-"}</Descriptions.Item>
        <Descriptions.Item label="结束时间">{execution.completedAt || "-"}</Descriptions.Item>
        <Descriptions.Item label="耗时" span={2}>
          {execution.duration ? `${Math.round(execution.duration / 1000)}秒` : "-"}
        </Descriptions.Item>
        {execution.error && (
          <Descriptions.Item label="错误信息" span={2}>
            <Alert type="error" message={execution.error} showIcon />
          </Descriptions.Item>
        )}
        {execution.message && (
          <Descriptions.Item label="当前消息" span={2}>
            <Text>{execution.message}</Text>
          </Descriptions.Item>
        )}
      </Descriptions>

      <Tabs
        activeKey={activeTab}
        onChange={setActiveTab}
        items={[
          {
            key: "steps",
            label: (
              <span>
                <ClockCircleOutlined /> 步骤进度
              </span>
            ),
            children: <div style={{ padding: "16px 0" }}>{renderSteps()}</div>,
          },
          {
            key: "logs",
            label: (
              <span>
                <FileTextOutlined /> 执行日志
              </span>
            ),
            children: (
              <div style={{ maxHeight: 400, overflowY: "auto", padding: "16px 0" }}>
                {renderLogs()}
              </div>
            ),
          },
          {
            key: "screenshots",
            label: (
              <span>
                <CameraOutlined /> 截图记录
              </span>
            ),
            children: (
              <div style={{ maxHeight: 400, overflowY: "auto", padding: "16px 0" }}>
                {renderScreenshots()}
              </div>
            ),
          },
        ]}
      />
    </Modal>
  );
}
