/**
 * 备份弹窗管理 Hook
 */

import { useState, useCallback } from "react";
import type { ConfigBackup, BaseResponse } from "@/types";
import type { FormInstance } from "antd/es/form";
import type { DeviceBackupGroup } from "../types";
import { post } from "@/lib/api";

interface UseBackupModalsOptions {
  onLoad: () => void;
}

interface UseBackupModalsReturn {
  backupModalVisible: boolean;
  restoreModalVisible: boolean;
  contentDrawerVisible: boolean;
  versionListDrawerVisible: boolean;
  selectedBackup: ConfigBackup | null;
  selectedRestoreBackup: ConfigBackup | null;
  selectedDeviceGroup: DeviceBackupGroup | null;
  backupContent: string;
  openBackupModal: () => void;
  closeBackupModal: (form?: FormInstance<unknown>) => void;
  openRestoreModal: (backup: ConfigBackup) => void;
  closeRestoreModal: () => void;
  openContentDrawer: (backup: ConfigBackup) => Promise<void>;
  closeContentDrawer: () => void;
  openVersionListDrawer: (group: DeviceBackupGroup) => void;
  closeVersionListDrawer: () => void;
  handleBackup: (form: FormInstance<unknown>) => Promise<void>;
  handleRestore: () => Promise<void>;
}

export function useBackupModals(
  options: UseBackupModalsOptions
): UseBackupModalsReturn {
  const { onLoad } = options;

  const [backupModalVisible, setBackupModalVisible] = useState(false);
  const [restoreModalVisible, setRestoreModalVisible] = useState(false);
  const [contentDrawerVisible, setContentDrawerVisible] = useState(false);
  const [versionListDrawerVisible, setVersionListDrawerVisible] = useState(false);
  const [selectedBackup, setSelectedBackup] = useState<ConfigBackup | null>(null);
  const [selectedRestoreBackup, setSelectedRestoreBackup] = useState<ConfigBackup | null>(null);
  const [selectedDeviceGroup, setSelectedDeviceGroup] = useState<DeviceBackupGroup | null>(null);
  const [backupContent, setBackupContent] = useState("");

  // 打开备份弹窗
  const openBackupModal = useCallback(() => {
    setBackupModalVisible(true);
  }, []);

  // 关闭备份弹窗
  const closeBackupModal = useCallback((form?: FormInstance<unknown>) => {
    setBackupModalVisible(false);
    if (form) {
      form.resetFields();
    }
  }, []);

  // 打开恢复弹窗
  const openRestoreModal = useCallback((backup: ConfigBackup) => {
    setSelectedRestoreBackup(backup);
    setRestoreModalVisible(true);
  }, []);

  // 关闭恢复弹窗
  const closeRestoreModal = useCallback(() => {
    setRestoreModalVisible(false);
    setSelectedRestoreBackup(null);
  }, []);

  // 打开内容抽屉
  const openContentDrawer = useCallback(async (backup: ConfigBackup) => {
    try {
      const result = await post<{ content: string }>("/network/backups/content", { id: backup.id });
      setBackupContent(result.data?.content || "");
      setSelectedBackup(backup);
      setContentDrawerVisible(true);
    } catch (error) {
      console.error("加载备份内容失败:", error);
    }
  }, []);

  // 关闭内容抽屉
  const closeContentDrawer = useCallback(() => {
    setContentDrawerVisible(false);
    setSelectedBackup(null);
    setBackupContent("");
  }, []);

  // 打开版本列表抽屉
  const openVersionListDrawer = useCallback((group: DeviceBackupGroup) => {
    setSelectedDeviceGroup(group);
    setVersionListDrawerVisible(true);
  }, []);

  // 关闭版本列表抽屉
  const closeVersionListDrawer = useCallback(() => {
    setVersionListDrawerVisible(false);
    setSelectedDeviceGroup(null);
  }, []);

  // 创建备份
  const handleBackup = useCallback(async (form: FormInstance<unknown>) => {
    try {
      const values = await form.validateFields() as Record<string, unknown>;
      // 调用批量备份端点，添加 backupType 字段
      await post("/network/backups/batch", {
        deviceIds: values.deviceIds,
        backupType: "manual",
        changeReason: values.changeReason,
      });
      closeBackupModal(form);
      onLoad();
    } catch (error: unknown) {
      if ((error as { errorFields?: unknown }).errorFields) {
        return;
      }
      console.error("创建备份失败:", (error as Error).message);
    }
  }, [closeBackupModal, onLoad]);

  // 恢复备份
  const handleRestore = useCallback(async () => {
    if (!selectedRestoreBackup) {
      return;
    }
    try {
      await post(`/network/backups/${selectedRestoreBackup.id}/restore`, {});
      closeRestoreModal();
      onLoad();
    } catch (error) {
      console.error("恢复备份失败:", error);
    }
  }, [selectedRestoreBackup, closeRestoreModal, onLoad]);

  return {
    backupModalVisible,
    restoreModalVisible,
    contentDrawerVisible,
    versionListDrawerVisible,
    selectedBackup,
    selectedRestoreBackup,
    selectedDeviceGroup,
    backupContent,
    openBackupModal,
    closeBackupModal,
    openRestoreModal,
    closeRestoreModal,
    openContentDrawer,
    closeContentDrawer,
    openVersionListDrawer,
    closeVersionListDrawer,
    handleBackup,
    handleRestore,
  };
}
