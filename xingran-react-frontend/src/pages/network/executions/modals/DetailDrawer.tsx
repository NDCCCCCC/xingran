/**
 * Execution Detail Drawer
 * 执行明细抽屉
 */

import { Drawer, Card, Steps, Progress, Table, Divider } from "antd";
import type { ConfigExecution, ConfigExecutionDetail } from "@/types";
import { getDetailColumns } from "../columns";

export interface DetailDrawerProps {
  open: boolean;
  currentExecution: ConfigExecution | null;
  executionDetails: ConfigExecutionDetail[];
  onClose: () => void;
  handleViewOutput: (output: string) => void;
}

export function DetailDrawer({
  open,
  currentExecution,
  executionDetails,
  onClose,
  handleViewOutput,
}: DetailDrawerProps) {
  const detailColumns = getDetailColumns({ handleViewOutput });

  return (
    <Drawer
      title={`执行明细 - ${currentExecution?.executionName || ""}`}
      placement="right"
      size="large"
      open={open}
      onClose={onClose}
    >
      {currentExecution && (
        <div>
          <Card size="small" style={{ marginBottom: 16 }}>
            <Steps
              current={currentExecution.status === "pending" ? 0 : currentExecution.status === "running" ? 1 : 2}
              status={currentExecution.status === "failed" ? "error" : undefined}
              size="small"
              items={[
                { title: "待执行" },
                { title: "执行中" },
                { title: "已完成" },
              ]}
            />
            <Divider style={{ margin: "12px 0" }} />
            <Progress
              percent={currentExecution.totalDevices > 0
                ? Math.round(((currentExecution.successCount + currentExecution.failureCount) / currentExecution.totalDevices) * 100)
                : 0}
              status={currentExecution.status === "failed" ? "exception" : undefined}
            />
            <div style={{ marginTop: 8, display: "flex", gap: 24 }}>
              <span>总设备: {currentExecution.totalDevices}</span>
              <span style={{ color: "var(--theme-success, #52c41a)" }}>成功: {currentExecution.successCount}</span>
              <span style={{ color: "var(--theme-error, #ff4d4f)" }}>失败: {currentExecution.failureCount}</span>
            </div>
          </Card>

          <Table
            columns={detailColumns}
            dataSource={executionDetails}
            rowKey="id"
            scroll={{ x: 1200 }}
            pagination={false}
            size="small"
          />
        </div>
      )}
    </Drawer>
  );
}
