/**
 * Network Command Detail Drawer
 * 网络命令执行明细抽屉
 */

import { Drawer, Card, Steps, Progress, Table } from "antd";
import type { ConfigExecution, ConfigExecutionDetail } from "@/types";
import { detailColumns } from "../columns";

export interface CommandDetailDrawerProps {
  open: boolean;
  execution: ConfigExecution | null;
  details: ConfigExecutionDetail[];
  onClose: () => void;
}

export function CommandDetailDrawer({
  open,
  execution,
  details,
  onClose,
}: CommandDetailDrawerProps) {
  if (!execution) {
    return null;
  }

  const currentStep = execution.status === "pending" ? 0 : execution.status === "running" ? 1 : 2;
  const percent =
    execution.totalDevices > 0
      ? Math.round(
          ((execution.successCount + execution.failureCount) / execution.totalDevices) * 100
        )
      : 0;

  return (
    <Drawer
      title={`执行明细 - ${execution.executionName || ""}`}
      placement="right"
      size="large"
      open={open}
      onClose={onClose}
    >
      <Card size="small" style={{ marginBottom: 16 }}>
        <Steps
          current={currentStep}
          status={execution.status === "failed" ? "error" : undefined}
          size="small"
          items={[{ title: "待执行" }, { title: "执行中" }, { title: "已完成" }]}
        />
        <div style={{ marginTop: 16 }}>
          <Progress
            percent={percent}
            status={execution.status === "failed" ? "exception" : undefined}
          />
          <div style={{ marginTop: 8, display: "flex", gap: 24 }}>
            <span>总设备: {execution.totalDevices}</span>
            <span style={{ color: "var(--theme-success, #2d8949)" }}>
              成功: {execution.successCount}
            </span>
            <span style={{ color: "var(--theme-error, #ba3630)" }}>
              失败: {execution.failureCount}
            </span>
          </div>
        </div>
      </Card>

      <Table
        columns={detailColumns}
        dataSource={details}
        rowKey="id"
        scroll={{ x: 1200 }}
        pagination={false}
        size="small"
      />
    </Drawer>
  );
}
