/**
 * Workstation Modals Hook
 * 工位模态框管理 Hook
 */

import { useCallback } from 'react';
import { message } from 'antd';
import type { WorkstationOps } from '@/types';
import { workstationApi } from '@/lib/opsApi';
import { handleSuccess, handleApiError } from '@/utils/errorHandler';

export interface UseWorkstationModalsReturn {
  openModal: (record?: WorkstationOps, form?: any) => void;
  closeModal: (form?: any) => void;
  handleSave: (
    editingWorkstation: WorkstationOps | null,
    form: any,
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
  const openModal = useCallback(async (record?: WorkstationOps, form?: any) => {
    if (record && form) {
      form.setFieldsValue(record);
      // 编辑模式下，如果有部门ID，加载该部门的用户列表
      if (record.deptId && loadUserOptions) {
        await loadUserOptions(record.deptId);
      }
    } else if (form) {
      form.resetFields();
      form.setFieldsValue({ status: 0, type: 0 });
    }
  }, [loadUserOptions]);

  const closeModal = useCallback((form?: any) => {
    if (form) {
      form.resetFields();
    }
  }, []);

  const handleSave = useCallback(
    async (
      editingWorkstation: WorkstationOps | null,
      form: any,
      onSuccess: () => void
    ) => {
      try {
        const values = await form.validateFields();
        if (editingWorkstation) {
          await workstationApi.update(editingWorkstation.id, values);
          handleSuccess('更新');
        } else {
          await workstationApi.create(values);
          handleSuccess('创建');
        }
        closeModal(form);
        onSuccess();
      } catch (error: any) {
        if (error.errorFields) return;
        handleApiError(error, '操作');
      }
    },
    [closeModal]
  );

  const handleDelete = useCallback(async (id: string, onSuccess: () => void) => {
    try {
      await workstationApi.delete(id);
      handleSuccess('删除');
      onSuccess();
    } catch (error) {
      handleApiError(error, '删除');
    }
  }, []);

  const handleBatchDelete = useCallback(
    async (selectedRowKeys: React.Key[], onSuccess: () => void, resetSelection: () => void) => {
      if (selectedRowKeys.length === 0) return;
      try {
        await workstationApi.batch('delete', { ids: selectedRowKeys });
        handleSuccess('批量删除');
        resetSelection();
        onSuccess();
      } catch (error) {
        handleApiError(error, '批量删除');
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
