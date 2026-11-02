/**
 * Holiday Modals Hook
 * 节假日模态框状态管理 Hook
 */

import { useState, useCallback } from "react";
import { App } from "antd";
import dayjs from "dayjs";
import type { FormInstance } from "antd/es/form";
import type { Holiday } from "@/lib/dutyApi";
import type { BatchHolidayRow, ModalState, BatchState } from "../types";
import { createHoliday, updateHoliday, deleteHoliday, batchCreateHolidays } from "@/lib/dutyApi";

export interface UseHolidayModalsParams {
  year: number | undefined;
  availableYears: number[];
  fetchList: () => Promise<void>;
}

export interface UseHolidayModalsReturn {
  modalState: ModalState;
  batchState: BatchState;

  setModalVisible: (visible: boolean) => void;
  setBatchModalVisible: (visible: boolean) => void;
  setBatchHolidays: React.Dispatch<React.SetStateAction<BatchHolidayRow[]>>;

  handleAdd: () => void;
  handleEdit: (record: Holiday) => void;
  handleDelete: (id: string) => Promise<void>;
  handleModalOk: (editForm: FormInstance<unknown>) => Promise<void>;
  handleBatchAdd: () => void;
  addBatchRow: () => void;
  removeBatchRow: (index: number) => void;
  updateBatchRow: (index: number, field: string, value: unknown) => void;
  handleBatchSubmit: () => Promise<void>;
}

export function useHolidayModals(params: UseHolidayModalsParams): UseHolidayModalsReturn {
  const { year, availableYears, fetchList } = params;
  const { message } = App.useApp();

  const [modalState, setModalState] = useState<ModalState>({
    modalVisible: false,
    batchModalVisible: false,
    editingRecord: null,
  });

  const [batchState, setBatchState] = useState<BatchState>({
    batchHolidays: [
      {
        holidayDate: dayjs(),
        holidayName: "",
        isOffday: true,
        holidayType: "legal",
        year: new Date().getFullYear(),
      },
    ],
  });

  const getDefaultYear = () => year ?? availableYears[0] ?? new Date().getFullYear();

  // 新增节假日
  const handleAdd = useCallback(() => {
    setModalState((prev) => ({ ...prev, editingRecord: null, modalVisible: true }));
  }, []);

  // 编辑节假日
  const handleEdit = useCallback((record: Holiday) => {
    setModalState((prev) => ({ ...prev, editingRecord: record, modalVisible: true }));
  }, []);

  // 删除节假日
  const handleDelete = useCallback(
    async (id: string) => {
      try {
        await deleteHoliday(id);
        message.success("删除成功");
        fetchList();
      } catch (_error) {
        message.error("删除失败");
      }
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
    [fetchList]
  );

  // 保存节假日
  const handleModalOk = useCallback(
    async (editForm: FormInstance<unknown>) => {
      try {
        const values = (await editForm.validateFields()) as {
          holidayDate: dayjs.Dayjs;
          holidayName: string;
          isOffday: boolean;
          holidayType: "legal" | "workday" | "custom";
          year: number;
          remark?: string;
        };
        const data = {
          holidayDate: values.holidayDate.format("YYYY-MM-DD"),
          holidayName: values.holidayName,
          isOffday: values.isOffday,
          holidayType: values.holidayType,
          year: values.year,
          remark: values.remark,
        };

        if (modalState.editingRecord) {
          await updateHoliday(modalState.editingRecord.id, data);
          message.success("更新成功");
        } else {
          await createHoliday(data);
          message.success("创建成功");
        }

        setModalState((prev) => ({ ...prev, modalVisible: false }));
        fetchList();
      } catch (error: unknown) {
        if (error && typeof error === "object" && "errorFields" in error) return;
        message.error(modalState.editingRecord ? "更新失败" : "创建失败");
      }
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
    [modalState.editingRecord, fetchList]
  );

  // 批量新增
  const handleBatchAdd = useCallback(() => {
    const defaultYear = getDefaultYear();
    setBatchState({
      batchHolidays: [
        {
          holidayDate: dayjs(),
          holidayName: "",
          isOffday: true,
          holidayType: "legal",
          year: defaultYear,
        },
      ],
    });
    setModalState((prev) => ({ ...prev, batchModalVisible: true }));
    // eslint-disable-next-line react-hooks/exhaustive-deps -- getDefaultYear is render-defined; year/availableYears tracked
  }, [year, availableYears]);

  // 添加批量行
  const addBatchRow = useCallback(() => {
    const defaultYear = getDefaultYear();
    setBatchState((prev) => ({
      batchHolidays: [
        ...prev.batchHolidays,
        {
          holidayDate: dayjs(),
          holidayName: "",
          isOffday: true,
          holidayType: "legal",
          year: defaultYear,
        },
      ],
    }));
    // eslint-disable-next-line react-hooks/exhaustive-deps -- getDefaultYear is render-defined; year/availableYears tracked
  }, [year, availableYears]);

  // 删除批量行
  const removeBatchRow = useCallback((index: number) => {
    setBatchState((prev) => ({
      batchHolidays: prev.batchHolidays.filter((_, i) => i !== index),
    }));
  }, []);

  // 更新批量行
  const updateBatchRow = useCallback((index: number, field: string, value: unknown) => {
    setBatchState((prev) => {
      const newRows = [...prev.batchHolidays];
      newRows[index] = { ...newRows[index], [field]: value };
      return { batchHolidays: newRows };
    });
  }, []);

  // 提交批量创建
  const handleBatchSubmit = useCallback(async () => {
    const { batchHolidays } = batchState;

    // 验证所有行
    for (let i = 0; i < batchHolidays.length; i++) {
      const row = batchHolidays[i];
      if (!row.holidayName) {
        message.error(`第 ${i + 1} 行：请输入节假日名称`);
        return;
      }
    }

    try {
      const data = batchHolidays.map((row) => ({
        holidayDate: row.holidayDate.format("YYYY-MM-DD"),
        holidayName: row.holidayName,
        isOffday: row.isOffday,
        holidayType: row.holidayType as "legal" | "workday" | "custom",
        year: row.year,
        remark: row.remark,
      }));

      await batchCreateHolidays(data);
      message.success(`成功创建 ${data.length} 条节假日记录`);

      setModalState((prev) => ({ ...prev, batchModalVisible: false }));
      fetchList();
    } catch (_error) {
      message.error("批量创建失败");
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
  }, [batchState, fetchList]);

  return {
    modalState,
    batchState,

    setModalVisible: (visible: boolean) =>
      setModalState((prev) => ({ ...prev, modalVisible: visible })),
    setBatchModalVisible: (visible: boolean) =>
      setModalState((prev) => ({ ...prev, batchModalVisible: visible })),
    setBatchHolidays: (holidays: React.SetStateAction<BatchHolidayRow[]>) => {
      setBatchState((prev) => ({
        batchHolidays: typeof holidays === "function" ? holidays(prev.batchHolidays) : holidays,
      }));
    },

    handleAdd,
    handleEdit,
    handleDelete,
    handleModalOk,
    handleBatchAdd,
    addBatchRow,
    removeBatchRow,
    updateBatchRow,
    handleBatchSubmit,
  };
}
