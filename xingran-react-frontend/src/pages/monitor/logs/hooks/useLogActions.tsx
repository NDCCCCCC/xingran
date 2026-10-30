/**
 * Log Actions Hook
 * 日志操作管理 Hook
 */

import { useCallback, useState } from "react";
import { post } from "@/lib/api";
import type { OperLog, LoginLog } from "../types";
import { confirmDelete } from "@/components/shared";

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

  const [detailModalVisible, setDetailModalVisible] = useState(false);
  const [selectedLog, setSelectedLog] = useState<OperLog | LoginLog | null>(null);

  // 查看详情
  const handleViewDetail = useCallback((record: OperLog | LoginLog) => {
    setSelectedLog(record);
    setDetailModalVisible(true);
  }, []);

  // 清空日志（68.1-T4 §1.9 非删除破坏性变体）
  const handleClearLogs = useCallback(() => {
    confirmDelete({
      title: "确认清空",
      target: activeTab === "oper" ? "全部操作日志" : "全部登录日志",
      actionLabel: "清空",
      actionContext: "清空日志",
      contentText: "将{target}，操作不可恢复，是否继续？",
      successText: "已清空日志",
      onDelete: async () => {
        const url = activeTab === "oper" ? "/monitor/oper-logs/clean" : "/monitor/login-logs/clean";
        await post(url, {});
        if (activeTab === "oper") {
          fetchOperLogs();
        } else {
          fetchLoginLogs();
        }
      },
    });
    // activeTab 仅为闭包读取,变化由调用方触发(handleClearLogs 依赖 activeTab)
  }, [activeTab, fetchOperLogs, fetchLoginLogs]);

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
