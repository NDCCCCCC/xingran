/**
 * Log Actions Hook
 * 日志操作管理 Hook
 */

import { useCallback, useState } from "react";
import { Modal, App } from "antd";
import { ExclamationCircleOutlined } from "@ant-design/icons";
import { post } from "@/lib/api";
import type { OperLog, LoginLog } from "../types";
import { handleApiError } from "@/utils/errorHandler";

export interface UseLogActionsParams {
  activeTab: string;
  fetchOperLogs: () => Promise<void>;
  fetchLoginLogs: () => Promise<void>;
}

export interface UseLogActionsReturn {
  detailModalVisible: boolean;
  selectedLog: OperLog | LoginLog | null;
  setDetailModalVisible: React.Dispatch<React.SetStateAction<boolean>>;
  setSelectedLog: React.Dispatch<React.SetStateAction<OperLog | LoginLog | null>>;
  handleViewDetail: (record: OperLog | LoginLog) => void;
  handleClearLogs: () => void;
  handleRefresh: () => void;
}

export function useLogActions(params: UseLogActionsParams): UseLogActionsReturn {
  const { activeTab, fetchOperLogs, fetchLoginLogs } = params;
  const { message } = App.useApp();

  const [detailModalVisible, setDetailModalVisible] = useState(false);
  const [selectedLog, setSelectedLog] = useState<OperLog | LoginLog | null>(null);

  // 查看详情
  const handleViewDetail = useCallback((record: OperLog | LoginLog) => {
    setSelectedLog(record);
    setDetailModalVisible(true);
  }, []);

  // 清空日志
  const handleClearLogs = useCallback(() => {
    Modal.confirm({
      title: "确认清空",
      icon: <ExclamationCircleOutlined />,
      content:
        activeTab === "oper"
          ? "确定要清空所有操作日志吗？此操作不可恢复！"
          : "确定要清空所有登录日志吗？此操作不可恢复！",
      onOk: async () => {
        try {
          const url =
            activeTab === "oper" ? "/monitor/oper-logs/clean" : "/monitor/login-logs/clean";

          await post(url, {});

          message.success("清空成功");
          if (activeTab === "oper") {
            fetchOperLogs();
          } else {
            fetchLoginLogs();
          }
        } catch (error) {
          handleApiError(error, "清空日志");
        }
      },
    });
    // activeTab 仅为闭包读取,变化由调用方触发(handleClearLogs 依赖 activeTab)
  }, [activeTab, fetchOperLogs, fetchLoginLogs, message]);

  // 刷新
  const handleRefresh = useCallback(() => {
    if (activeTab === "oper") {
      fetchOperLogs();
    } else {
      fetchLoginLogs();
    }
  }, [activeTab, fetchOperLogs, fetchLoginLogs]);

  return {
    detailModalVisible,
    selectedLog,
    setDetailModalVisible,
    setSelectedLog,
    handleViewDetail,
    handleClearLogs,
    handleRefresh,
  };
}
