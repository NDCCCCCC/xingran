/**
 * NetworkDevice 模态框状态管理 Hook
 */

import { useState, useCallback } from "react";
import type { NetworkDevice } from "@/types";

export interface ProbeResult {
  success: boolean;
  message?: string;
  device?: NetworkDevice;
  deviceName?: string;
  deviceType?: string;
  vendor?: string;
  model?: string;
  sysName?: string;
  sysDescr?: string;
}

export interface UseDeviceModalsReturn {
  quickCreateModalVisible: boolean;
  detailModalVisible: boolean;
  viewingDevice: NetworkDevice | null;
  probeResult: ProbeResult | null;
  probing: boolean;
  creating: boolean;
  setQuickCreateModalVisible: (visible: boolean) => void;
  setDetailModalVisible: (visible: boolean) => void;
  setViewingDevice: (device: NetworkDevice | null) => void;
  setProbeResult: (result: ProbeResult | null) => void;
  setProbing: (probing: boolean) => void;
  setCreating: (creating: boolean) => void;
  openDetailModal: (record: NetworkDevice) => void;
  closeQuickCreateModal: () => void;
}

export function useDeviceModals(): UseDeviceModalsReturn {
  const [quickCreateModalVisible, setQuickCreateModalVisible] = useState(false);
  const [detailModalVisible, setDetailModalVisible] = useState(false);
  const [viewingDevice, setViewingDevice] = useState<NetworkDevice | null>(null);
  const [probeResult, setProbeResult] = useState<ProbeResult | null>(null);
  const [probing, setProbing] = useState(false);
  const [creating, setCreating] = useState(false);

  // 打开详情模态框
  const openDetailModal = useCallback((record: NetworkDevice) => {
    setViewingDevice(record);
    setDetailModalVisible(true);
  }, []);

  // 关闭快速创建模态框
  const closeQuickCreateModal = useCallback(() => {
    setQuickCreateModalVisible(false);
    setProbeResult(null);
  }, []);

  return {
    quickCreateModalVisible,
    detailModalVisible,
    viewingDevice,
    probeResult,
    probing,
    creating,
    setQuickCreateModalVisible,
    setDetailModalVisible,
    setViewingDevice,
    setProbeResult,
    setProbing,
    setCreating,
    openDetailModal,
    closeQuickCreateModal,
  };
}
