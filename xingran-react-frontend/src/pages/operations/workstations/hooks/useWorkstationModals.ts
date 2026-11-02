/**
 * Workstation Modals Hook
 * 工位模态框管理 Hook
 */

import { useCallback } from "react";
import type { WorkstationOps } from "@/types";
import type { FormInstance } from "antd/es/form";
import { workstationApi } from "@/lib/opsApi";
import { handleSuccess, handleApiError } from "@/utils/errorHandler";

export interface UseWorkstationModalsReturn {
  openModal: (record?: WorkstationOps, form?: FormInstance<unknown>) => void;
  closeModal: (form?: FormInstance<unknown>) => void;
  handleSave: (
    editingWorkstation: WorkstationOps | null,
    form: FormInstance<unknown>,
    onSuccess: () => void
  ) => Promise<void>;
  handleDelete: (id: string, onSuccess: () => void) => Promise<void>;
  handleBatchDelete: (
    selectedRowKeys: React.Key[],
    onSuccess: () => void,
    resetSelection: () => void
  ) => Promise<void>;
}

export function useWorkstationModals(
  loadUserOptions?: (deptId?: string) => Promise<void>
): UseWorkstationModalsReturn {
  const openModal = useCallback(
    async (record?: WorkstationOps, _form?: FormInstance<unknown>) => {
      // 编辑模式下，如果有部门ID，加载该部门的用户列表
      // 返回 record 以便调用者使用
      if (record?.deptId && loadUserOptions) {
        await loadUserOptions(record.deptId);
      }
      return record;
    },
    [loadUserOptions]
  );

  const closeModal = useCallback((form?: FormInstance<unknown>) => {
    if (form) {
      form.resetFields();
    }
  }, []);

  const handleSave = useCallback(
    async (
      editingWorkstation: WorkstationOps | null,
      form: FormInstance<unknown>,
      onSuccess: () => void
    ) => {
      try {
        const values = await form.validateFields();
        if (editingWorkstation) {
          await workstationApi.update(editingWorkstation.id, values as Partial<WorkstationOps>);
          handleSuccess("更新");
        } else {
          await workstationApi.create(values as Partial<WorkstationOps>);
          handleSuccess("创建");
        }
        closeModal(form);
        onSuccess();
      } catch (error: unknown) {
        if ((error as { errorFields?: unknown }).errorFields) return;
        handleApiError(error, "操作");
      }
    },
    [closeModal]
  );

  const handleDelete = useCallback(async (id: string, onSuccess: () => void) => {
    try {
      await workstationApi.delete(id);
      handleSuccess("删除");
      onSuccess();
    } catch (error) {
      handleApiError(error, "删除");
    }
  }, []);

  const handleBatchDelete = useCallback(
    async (selectedRowKeys: React.Key[], onSuccess: () => void, resetSelection: () => void) => {
      if (selectedRowKeys.length === 0) return;
      try {
        await workstationApi.batch("delete", { ids: selectedRowKeys });
        handleSuccess("批量删除");
        resetSelection();
        onSuccess();
      } catch (error) {
        handleApiError(error, "批量删除");
      }
    },
    []
  );

  return {
    openModal,
    closeModal,
    handleSave,
    handleDelete,
    handleBatchDelete,
  };
}
