/**
 * Duty Schedule Modals Hook
 * 值班排班弹窗状态管理 Hook
 */

import { useState, useCallback } from "react";
import { Form, App } from "antd";
import type { FormInstance } from "antd/es/form";
import {
  generateSchedule,
  swapDuty,
  manualDuty,
  deleteDutySchedule,
  batchDeleteDutySchedules,
} from "@/lib/dutyApi";

export interface UseScheduleModalsParams {
  onLoad?: () => void;
  allSchedules: unknown[];
  dataSource: unknown[];
  current: number;
}

export interface UseScheduleModalsReturn {
  // 弹窗显示状态
  generateModalVisible: boolean;
  swapModalVisible: boolean;
  manualModalVisible: boolean;

  // 表单实例
  generateForm: FormInstance<unknown>;
  swapForm: FormInstance<unknown>;
  manualForm: FormInstance<unknown>;

  // 弹窗控制方法
  openGenerateModal: () => void;
  closeGenerateModal: () => void;
  openSwapModal: () => void;
  closeSwapModal: () => void;
  openManualModal: () => void;
  closeManualModal: () => void;

  // 操作方法
  handleGenerate: () => Promise<void>;
  handleSwap: () => Promise<void>;
  handleManual: () => Promise<void>;
  handleDelete: (id: string) => Promise<void>;
  handleBatchDelete: (selectedRowKeys: string[], setSelectedRowKeys: (keys: string[]) => void) => Promise<void>;
}

export function useScheduleModals(params: UseScheduleModalsParams): UseScheduleModalsReturn {
  const { onLoad, allSchedules: _allSchedules, dataSource: _dataSource, current: _current } = params;
  const { message } = App.useApp();

  const [generateModalVisible, setGenerateModalVisible] = useState(false);
  const [swapModalVisible, setSwapModalVisible] = useState(false);
  const [manualModalVisible, setManualModalVisible] = useState(false);

  const [generateForm] = Form.useForm();
  const [swapForm] = Form.useForm();
  const [manualForm] = Form.useForm();

  // 生成排班
  const handleGenerate = useCallback(async () => {
    try {
      const values = await generateForm.validateFields();
      const result = await generateSchedule({
        poolId: values.poolId,
        startDate: values.dateRange[0].format("YYYY-MM-DD"),
        endDate: values.dateRange[1].format("YYYY-MM-DD"),
        dutyType: values.dutyType,
        clearExists: values.clearExists || false,
      });
      message.success(`成功生成 ${result.data?.count || 0} 条排班记录`);
      setGenerateModalVisible(false);
      generateForm.resetFields();
      onLoad?.();
    } catch (error: unknown) {
      if (error && typeof error === "object" && "errorFields" in error) return;
      message.error("生成排班失败");
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
  }, [generateForm, onLoad]);

  // 调班
  const handleSwap = useCallback(async () => {
    try {
      const values = await swapForm.validateFields();
      await swapDuty({
        fromScheduleId: values.fromScheduleId,
        toScheduleId: values.toScheduleId,
        reason: values.reason,
      });
      message.success("调班成功");
      setSwapModalVisible(false);
      swapForm.resetFields();
      onLoad?.();
    } catch (error: unknown) {
      if (error && typeof error === "object" && "errorFields" in error) return;
      message.error("调班失败");
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
  }, [swapForm, onLoad]);

  // 手动排班
  const handleManual = useCallback(async () => {
    try {
      const values = await manualForm.validateFields();
      await manualDuty({
        poolId: values.poolId,
        dutyDate: values.dutyDate.format("YYYY-MM-DD"),
        dutyType: values.dutyType,
        userIds: values.userIds,
        reason: values.reason,
      });
      message.success("手动排班成功");
      setManualModalVisible(false);
      manualForm.resetFields();
      onLoad?.();
    } catch (error: unknown) {
      if (error && typeof error === "object" && "errorFields" in error) return;
      message.error("手动排班失败");
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
  }, [manualForm, onLoad]);

  // 删除单条排班
  const handleDelete = useCallback(async (id: string) => {
    try {
      await deleteDutySchedule(id);
      message.success("删除成功");
      onLoad?.();
    } catch (_error) {
      message.error("删除失败");
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
  }, [onLoad]);

  // 批量删除排班
  const handleBatchDelete = useCallback(async (selectedRowKeys: string[], setSelectedRowKeys: (keys: string[]) => void) => {
    if (selectedRowKeys.length === 0) {
      message.warning("请先选择要删除的排班记录");
      return;
    }
    try {
      await batchDeleteDutySchedules(selectedRowKeys);
      message.success(`成功删除 ${selectedRowKeys.length} 条排班记录`);
      setSelectedRowKeys([]);
      onLoad?.();
    } catch (_error) {
      message.error("批量删除失败");
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
  }, [onLoad]);

  return {
    generateModalVisible,
    swapModalVisible,
    manualModalVisible,
    generateForm,
    swapForm,
    manualForm,
    openGenerateModal: () => setGenerateModalVisible(true),
    closeGenerateModal: () => {
      setGenerateModalVisible(false);
      generateForm.resetFields();
    },
    openSwapModal: () => setSwapModalVisible(true),
    closeSwapModal: () => {
      setSwapModalVisible(false);
      swapForm.resetFields();
    },
    openManualModal: () => setManualModalVisible(true),
    closeManualModal: () => {
      setManualModalVisible(false);
      manualForm.resetFields();
    },
    handleGenerate,
    handleSwap,
    handleManual,
    handleDelete,
    handleBatchDelete,
  };
}
