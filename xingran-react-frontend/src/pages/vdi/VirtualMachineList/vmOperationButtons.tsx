import React from "react";
import { PlayCircleOutlined, StopOutlined, ReloadOutlined, SyncOutlined, DeleteOutlined, UserAddOutlined } from "@ant-design/icons";

export interface VMOprationButton {
  action: "start" | "stop" | "restart" | "sync" | "delete" | "bind";
  permission: string;
  label: string;
  icon: React.ReactNode;
}

export const vmOperationButtons: VMOprationButton[] = [
  { action: "start", permission: "vdi:vm:start", label: "开机", icon: <PlayCircleOutlined /> },
  { action: "stop", permission: "vdi:vm:stop", label: "关机", icon: <StopOutlined /> },
  { action: "restart", permission: "vdi:vm:restart", label: "重启", icon: <ReloadOutlined /> },
  { action: "sync", permission: "vdi:vm:sync", label: "同步", icon: <SyncOutlined /> },
  { action: "delete", permission: "vdi:vm:remove", label: "删除", icon: <DeleteOutlined /> },
  { action: "bind", permission: "vdi:vm:bind", label: "绑定用户", icon: <UserAddOutlined /> },
];
