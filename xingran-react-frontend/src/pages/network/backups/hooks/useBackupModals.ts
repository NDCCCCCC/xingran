/**
 * 备份弹窗管理 Hook
 */

import { useState, useCallback } from "react";
import type { ConfigBackup } from "@/types";
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
  /** 进行中的恢复任务 ID（非空时恢复 Modal 切换为进度态，Phase 93 D-17/D-19） */
  restoreTaskId: string | null;
  openBackupModal: () => void;
  closeBackupModal: (form?: FormInstance<unknown>) => void;
  openRestoreModal: (backup: ConfigBackup) => void;
  closeRestoreModal: () => void;
  openContentDrawer: (backup: ConfigBackup) => Promise<void>;
  closeContentDrawer: () => void;
  openVersionListDrawer: (group: DeviceBackupGroup) => void;
  closeVersionListDrawer: () => void;
  handleBackup: (form: FormInstance<unknown>) => Promise<void>;
  handleRestore: (backup?: ConfigBackup) => Promise<void>;
}

export function useBackupModals(options: UseBackupModalsOptions): UseBackupModalsReturn {
  const { onLoad } = options;

  const [backupModalVisible, setBackupModalVisible] = useState(false);
  const [restoreModalVisible, setRestoreModalVisible] = useState(false);
  const [contentDrawerVisible, setContentDrawerVisible] = useState(false);
  const [versionListDrawerVisible, setVersionListDrawerVisible] = useState(false);
  const [selectedBackup, setSelectedBackup] = useState<ConfigBackup | null>(null);
  const [selectedRestoreBackup, setSelectedRestoreBackup] = useState<ConfigBackup | null>(null);
  const [selectedDeviceGroup, setSelectedDeviceGroup] = useState<DeviceBackupGroup | null>(null);
  const [backupContent, setBackupContent] = useState("");
  const [restoreTaskId, setRestoreTaskId] = useState<string | null>(null);

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

  // 关闭恢复弹窗（复位任务态——任务本体在服务端继续执行，不受前端关闭影响）
  const closeRestoreModal = useCallback(() => {
    setRestoreModalVisible(false);
    setSelectedRestoreBackup(null);
    setRestoreTaskId(null);
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
  const handleBackup = useCallback(
    async (form: FormInstance<unknown>) => {
      try {
        const values = (await form.validateFields()) as Record<string, unknown>;
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
    },
    [closeBackupModal, onLoad]
  );

  // 恢复备份（Phase 93 异步语义：携带备份自身 deviceId 发起（D-04 前端侧——
  // 修复现状发空 body 缺 deviceId 必 400 的缺陷），捕获 taskId 后 Modal 切换为
  // 进度态，由 useRestoreTask 轮询任务详情（D-16/D-17/D-19），不立即关闭弹窗。
  // v129-recheck WR-01: backup 显式参数——版本抽屉路径在同一同步块内
  // openRestoreModal(backup) 后立即调用，state 尚未传播（且本闭包捕获的
  // selectedRestoreBackup 恒为旧值），无参调用会因 null 守卫静默短路成死代码；
  // 显式传参使恢复必然发起，恢复 Modal 主路径不传参仍走 state。
  const handleRestore = useCallback(
    async (backup?: ConfigBackup) => {
      const target = backup ?? selectedRestoreBackup;
      if (!target) {
        return;
      }
      try {
        const result = await post<{ taskId: string; status: string; message: string }>(
          `/network/backups/${target.id}/restore`,
          { deviceId: target.deviceId }
        );
        setRestoreTaskId(result.data?.taskId || null);
        onLoad();
      } catch (error) {
        console.error("发起恢复任务失败:", error);
      }
    },
    [selectedRestoreBackup, onLoad]
  );

  return {
    backupModalVisible,
    restoreModalVisible,
    contentDrawerVisible,
    versionListDrawerVisible,
    selectedBackup,
    selectedRestoreBackup,
    selectedDeviceGroup,
    backupContent,
    restoreTaskId,
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
