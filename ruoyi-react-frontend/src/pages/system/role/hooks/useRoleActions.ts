/**
 * Role Actions Hook
 * 角色操作管理 Hook
 */

import { useState, useCallback } from 'react';
import { message } from 'antd';
import type { Role } from '@/types';
import type { Key } from 'antd/es/table/interface';
import type { FormInstance } from 'antd/es/form';
import { post } from '@/lib/api';

export interface UseRoleActionsParams {
  loadRoles: () => Promise<void>;
  loadStatistics: () => Promise<void>;
  loadRoleMenus: (roleId: string) => Promise<string[]>;
  loadRoleDepts: (roleId: string) => Promise<string[]>;
  checkedMenuKeys: Key[];
  checkedDeptKeys: Key[];
  currentDataScope: number;
}

export interface UseRoleActionsReturn {
  editingRole: Role | null;
  editModalVisible: boolean;
  pendingFormData: Record<string, unknown> | null;
  selectedRowKeys: Key[];

  setEditingRole: React.Dispatch<React.SetStateAction<Role | null>>;
  setEditModalVisible: React.Dispatch<React.SetStateAction<boolean>>;
  setPendingFormData: React.Dispatch<React.SetStateAction<Record<string, unknown> | null>>;
  setSelectedRowKeys: React.Dispatch<React.SetStateAction<Key[]>>;

  handleAdd: () => void;
  handleEdit: (record: Role) => Promise<void>;
  handleDelete: (id: string) => Promise<void>;
  handleBatchDelete: (selectedRowKeys: Key[]) => Promise<void>;
  handleUpdateStatus: (id: string, status: number) => Promise<void>;
  handleSave: (editForm: FormInstance<unknown>) => Promise<void>;
  handleDataScopeChange: (value: number, editForm: FormInstance<unknown>) => void;
  handleModalOpenChange: (open: boolean, editForm: FormInstance<unknown>) => void;
}

export function useRoleActions(params: UseRoleActionsParams): UseRoleActionsReturn {
  const {
    loadRoles,
    loadStatistics,
    loadRoleMenus,
    loadRoleDepts,
    checkedMenuKeys,
    checkedDeptKeys,
    currentDataScope,
  } = params;

  const [editingRole, setEditingRole] = useState<Role | null>(null);
  const [editModalVisible, setEditModalVisible] = useState(false);
  const [pendingFormData, setPendingFormData] = useState<Record<string, unknown> | null>(null);
  const [selectedRowKeys, setSelectedRowKeys] = useState<Key[]>([]);

  // 新增角色
  const handleAdd = useCallback(() => {
    setEditingRole(null);
    setPendingFormData(null);
    setEditModalVisible(true);
  }, []);

  // 编辑角色
  const handleEdit = useCallback(async (record: Role) => {
    setEditingRole(record);

    // 加载角色的菜单和部门权限
    const [menuIds, deptIds] = await Promise.all([
      loadRoleMenus(record.id),
      loadRoleDepts(record.id),
    ]);

    // 保存待设置的表单数据
    setPendingFormData({
      ...record,
      menuIds,
      deptIds,
    });
    setEditModalVisible(true);
  }, [loadRoleMenus, loadRoleDepts]);

  // 删除角色
  const handleDelete = useCallback(async (id: string) => {
    try {
      await post(`/system/roles/${id}/delete`);
      message.success('删除成功');
      loadRoles();
      loadStatistics();
    } catch (error) {
      console.error('删除角色失败:', error);
    }
  }, [loadRoles, loadStatistics]);

  // 批量删除
  const handleBatchDelete = useCallback(async (keys: Key[]) => {
    if (keys.length === 0) {
      message.warning('请选择要删除的角色');
      return;
    }

    try {
      await post('/system/roles/batch-delete', { ids: keys });
      message.success('批量删除成功');
      setSelectedRowKeys([]);
      loadRoles();
      loadStatistics();
    } catch (error) {
      console.error('批量删除失败:', error);
    }
  }, [loadRoles, loadStatistics]);

  // 更新角色状态
  const handleUpdateStatus = useCallback(async (id: string, status: number) => {
    try {
      await post(`/system/roles/${id}/status`, { status });
      message.success(status === 0 ? '启用成功' : '停用成功');
      loadRoles();
      loadStatistics();
    } catch (error) {
      console.error('更新角色状态失败:', error);
    }
  }, [loadRoles, loadStatistics]);

  // 保存角色
  const handleSave = useCallback(async (editForm: FormInstance<unknown>) => {
    try {
      const values = await editForm.validateFields() as Record<string, unknown>;

      const roleData = {
        roleName: values.roleName as string,
        roleKey: values.roleKey as string,
        roleSort: parseInt(String(values.roleSort)) || 0,
        dataScope: (values.dataScope as number) || 1,
        menuCheckStrictly: true,
        deptCheckStrictly: true,
        status: parseInt(String(values.status)) || 0,
        remark: values.remark as string,
        menuIds: (values.menuIds as string[]) || [],
        deptIds: (values.deptIds as string[]) || [],
      };

      if (editingRole) {
        await post(`/system/roles/${editingRole.id}/update`, {
          ...roleData,
          id: editingRole.id,
        });
      } else {
        await post('/system/roles', roleData);
      }

      message.success(editingRole ? '更新成功' : '创建成功');
      setEditModalVisible(false);
      loadRoles();
      loadStatistics();
    } catch (error) {
      console.error('保存角色失败:', error);
    }
  }, [editingRole, loadRoles, loadStatistics]);

  // 数据范围变更
  const handleDataScopeChange = useCallback((value: number, editForm: FormInstance<unknown>) => {
    // 如果不是自定义数据权限，清空部门权限选择
    if (value !== 2) {
      editForm.setFieldsValue({ deptIds: [] });
    }
  }, []);

  // Modal 打开后的回调
  const handleModalOpenChange = useCallback((open: boolean, editForm: FormInstance<unknown>) => {
    if (open && pendingFormData) {
      editForm.setFieldsValue(pendingFormData);
      setPendingFormData(null);
    }
  }, [pendingFormData]);

  return {
    editingRole,
    editModalVisible,
    pendingFormData,
    selectedRowKeys,
    setEditingRole,
    setEditModalVisible,
    setPendingFormData,
    setSelectedRowKeys,
    handleAdd,
    handleEdit,
    handleDelete,
    handleBatchDelete,
    handleUpdateStatus,
    handleSave,
    handleDataScopeChange,
    handleModalOpenChange,
  };
}
