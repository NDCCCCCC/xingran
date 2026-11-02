import { useState, useCallback, useRef } from "react";
import type { Key } from "react";
import { Form } from "antd";
import type { RoomDevice } from "@/types";
import { roomDeviceApi, serverRoomApi } from "@/lib/opsApi";
import { handleApiError, handleSuccess, isFormValidationError } from "@/utils/errorHandler";
import type { RoomOption, Statistics } from "../constants";

export function useRoomDeviceActions() {
  const [roomOptions, setRoomOptions] = useState<RoomOption[]>([]);
  const [statistics, setStatistics] = useState<Statistics>({
    total: 0,
    normal: 0,
    fault: 0,
    scrapped: 0,
  });
  const [cardDevices, setCardDevices] = useState<RoomDevice[]>([]);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingDevice, setEditingDevice] = useState<RoomDevice | null>(null);
  const [deviceForm] = Form.useForm();
  const [importVisible, setImportVisible] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [exportVisible, setExportVisible] = useState(false);
  const [exportFilters, setExportFilters] = useState<Record<string, unknown>>({});

  // PageTable render-prop 上下文提升到业务回调（提交/删除/导入后刷新 + 导出筛选 + 清筛选）
  const reloadRef = useRef<() => void>(() => {});

  const loadRoomOptions = useCallback(async (orgId?: string) => {
    try {
      const params: Record<string, unknown> = { current: 1, pageSize: 50 };
      if (orgId) params.orgId = orgId;
      const result = await serverRoomApi.list(params);
      const rooms = result.data?.list || [];
      setRoomOptions(
        rooms.map((r) => ({
          id: r.id,
          name: r.name,
          buildingId: r.buildingId,
          orgId: r.orgId || "",
        }))
      );
    } catch (error) {
      handleApiError(error, "加载机房选项", false);
    }
  }, []);

  const loadStatistics = useCallback(async (): Promise<Statistics> => {
    try {
      const stats = await roomDeviceApi.statistics();
      return {
        total: stats.total ?? 0,
        normal: stats.normal ?? 0,
        fault: stats.fault ?? 0,
        scrapped: stats.scrapped ?? 0,
      };
    } catch (error) {
      handleApiError(error, "加载统计数据", false);
      return { total: 0, normal: 0, fault: 0, scrapped: 0 };
    }
  }, []);

  const refreshStatistics = useCallback(() => {
    void loadStatistics().then(setStatistics);
  }, [loadStatistics]);

  const refreshData = useCallback(() => {
    reloadRef.current();
    refreshStatistics();
  }, [refreshStatistics]);

  const closeModal = () => {
    setModalVisible(false);
    deviceForm.resetFields();
    setEditingDevice(null);
  };

  const handleSave = async () => {
    try {
      setSubmitting(true);
      const values = await deviceForm.validateFields();
      if (editingDevice) {
        await roomDeviceApi.update(editingDevice.id, values as Partial<RoomDevice>);
        handleSuccess("更新");
      } else {
        await roomDeviceApi.create(values as Partial<RoomDevice>);
        handleSuccess("创建");
      }
      closeModal();
      refreshData();
    } catch (error: unknown) {
      if (isFormValidationError(error)) return;
      handleApiError(error, "操作");
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async (id: string) => {
    await roomDeviceApi.delete(id);
    refreshData();
  };

  const handleBatchDelete = async (keys: Key[]) => {
    await roomDeviceApi.batch("delete", { ids: keys });
    refreshStatistics();
  };

  const openModal = (record?: RoomDevice) => {
    if (record) {
      setEditingDevice(record);
      if (record.roomId) {
        setRoomOptions((prev) =>
          prev.find((r) => r.id === record.roomId)
            ? prev
            : [
                ...prev,
                {
                  id: record.roomId,
                  name: record.roomName || "未知机房",
                  buildingId: "",
                  orgId: (record as unknown as { orgId?: string }).orgId || "",
                },
              ]
        );
      }
      deviceForm.setFieldsValue({
        ...record,
        status: record.status ?? 0,
      });
    } else {
      setEditingDevice(null);
      deviceForm.resetFields();
      deviceForm.setFieldsValue({ status: 0 });
    }
    setModalVisible(true);
  };

  const request = useCallback(async (params: Record<string, unknown>) => {
    const result = await roomDeviceApi.list(params);
    const list = result.data?.list || [];
    setCardDevices(list);
    return { list, total: result.data?.total || 0 };
  }, []);

  return {
    roomOptions,
    setRoomOptions,
    loadRoomOptions,
    statistics,
    setStatistics,
    loadStatistics,
    refreshStatistics,
    refreshData,
    cardDevices,
    modalVisible,
    setModalVisible,
    editingDevice,
    deviceForm,
    closeModal,
    openModal,
    handleSave,
    handleDelete,
    handleBatchDelete,
    importVisible,
    setImportVisible,
    submitting,
    exportVisible,
    setExportVisible,
    exportFilters,
    setExportFilters,
    reloadRef,
    request,
  };
}
