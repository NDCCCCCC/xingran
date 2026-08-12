/**
 * RPA 管理主页面
 * 包含任务管理、执行记录、Worker 监控三个子页面
 */

import { useState, useCallback } from "react";
import type { FC } from "react";
import { Card, Tabs } from "antd";
import {
  RobotOutlined, HistoryOutlined, CloudServerOutlined,
} from "@ant-design/icons";
import TaskManagement from "./tasks";
import ExecutionManagement from "./executions";
import WorkerMonitor from "./workers";

const RpaManagement: FC = () => {
  const [activeTab, setActiveTab] = useState("tasks");

  const handleTabChange = useCallback((key: string) => {
    setActiveTab(key);
  }, []);

  return (
    <div style={{ padding: 0, height: "100%" }}>
      <Card variant="borderless" style={{ height: "100%" }}>
        <Tabs
          activeKey={activeTab}
          onChange={handleTabChange}
          tabBarStyle={{ marginBottom: 16 }}
          items={[
            {
              key: "tasks",
              label: (
                <span>
                  <RobotOutlined />
                  任务管理
                </span>
              ),
              children: <TaskManagement />,
            },
            {
              key: "executions",
              label: (
                <span>
                  <HistoryOutlined />
                  执行记录
                </span>
              ),
              children: <ExecutionManagement />,
            },
            {
              key: "workers",
              label: (
                <span>
                  <CloudServerOutlined />
                  Worker 监控
                </span>
              ),
              children: <WorkerMonitor />,
            },
          ]}
        />
      </Card>
    </div>
  );
};

export default RpaManagement;
