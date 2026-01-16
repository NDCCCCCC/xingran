/**
 * Menu Actions Hook
 * 菜单操作管理 Hook
 */

import { useState, useCallback } from 'react';
import { Modal, message, Alert, Checkbox } from 'antd';
import { WarningOutlined } from '@ant-design/icons';
import type { Menu } from '@/types';
import { post } from '@/lib/api';
import { refreshMenuCache } from '@/store/menuStore';

export interface UseMenuActionsParams {
  onLoad: () => void;
  selectedRowKeys: string[];
  setSelectedRowKeys: (keys: string[]) => void;
}

export interface UseMenuActionsReturn {
  // 编辑状态
  editingMenu: Menu | null;
  cascadeDelete: boolean;
  setCascadeDelete: (value: boolean) => void;

  // 操作方法
  handleAdd: () => void;
  handleEdit: (record: Menu) => void;
  handleDeleteConfirm: (record: Menu) => void;
  handleBatchDelete: () => void;
  handleSave: (editForm: any) => Promise<void>;
  setEditingMenu: (menu: Menu | null) => void;
}

export function useMenuActions(params: UseMenuActionsParams): UseMenuActionsReturn {
  const { onLoad, selectedRowKeys, setSelectedRowKeys } = params;

  const [editingMenu, setEditingMenu] = useState<Menu | null>(null);
  const [cascadeDelete, setCascadeDelete] = useState(false);

  // 新增菜单
  const handleAdd = useCallback(() => {
    setEditingMenu(null);
  }, []);

  // 编辑菜单
  const handleEdit = useCallback((record: Menu) => {
    setEditingMenu(record);
  }, []);

  // 删除菜单
  const handleDelete = useCallback(async (id: string, cascade: boolean = false) => {
    try {
      const url = cascade
        ? `/system/menus/${id}/delete?cascade=true`
        : `/system/menus/${id}/delete`;

      const result = await post(url);

      if (cascade && result.data?.deletedCount) {
        message.success(`删除成功！共删除 ${result.data.deletedCount} 个菜单（包含子菜单）`);
      } else {
        message.success('删除成功');
      }

      onLoad();

      // 刷新菜单缓存，使菜单变更立即生效
      await refreshMenuCache();
      message.success('菜单已刷新，变更已生效');
    } catch (error: any) {
      console.error('删除菜单失败:', error);
      const errorMsg = error?.response?.data?.message || error?.message || '删除失败';
      message.error(errorMsg);
    }
  }, [onLoad]);

  // 单个删除确认（带级联删除选项）
  const handleDeleteConfirm = useCallback((record: Menu) => {
    const hasChildren = record.children && record.children.length > 0;

    Modal.confirm({
      title: '删除菜单',
      icon: <WarningOutlined style={{ color: '#faad14' }} />,
      content: (
        <div>
          <p>确定要删除菜单 <strong>{record.menuName}</strong> 吗？</p>
          {hasChildren && (
            <Alert
              message="该菜单下有子菜单，删除时请选择是否同时删除子菜单"
              type="warning"
              showIcon
              style={{ marginBottom: 16 }}
            />
          )}
          <div style={{ marginTop: hasChildren ? 16 : 0 }}>
            <Checkbox
              checked={cascadeDelete}
              onChange={(e) => setCascadeDelete(e.target.checked)}
            >
              级联删除子菜单
            </Checkbox>
            {hasChildren && !cascadeDelete && (
              <Alert
                message="不启用级联删除时，如果菜单下有子菜单，删除将失败"
                type="error"
                showIcon
                style={{ marginTop: 12 }}
              />
            )}
          </div>
        </div>
      ),
      okText: '确定删除',
      okType: 'danger',
      cancelText: '取消',
      onOk: () => handleDelete(record.id, cascadeDelete),
    });
  }, [cascadeDelete, handleDelete]);

  // 批量删除菜单
  const handleBatchDelete = useCallback(() => {
    if (selectedRowKeys.length === 0) {
      message.warning('请至少选择一个菜单');
      return;
    }

    Modal.confirm({
      title: '批量删除菜单',
      icon: <WarningOutlined style={{ color: '#faad14' }} />,
      content: (
        <div>
          <p>您已选择 <strong>{selectedRowKeys.length}</strong> 个菜单，确定要删除吗？</p>
          <Alert
            message="删除后将无法恢复，请谨慎操作！"
            type="warning"
            showIcon
            style={{ marginBottom: 16 }}
          />
          <div style={{ marginTop: 16 }}>
            <Checkbox
              checked={cascadeDelete}
              onChange={(e) => setCascadeDelete(e.target.checked)}
            >
              级联删除（同时删除所有子菜单）
            </Checkbox>
            {cascadeDelete && (
              <Alert
                message="启用级联删除后，将同时删除所选菜单及其所有子菜单"
                type="info"
                showIcon
                style={{ marginTop: 12 }}
              />
            )}
          </div>
        </div>
      ),
      okText: '确定删除',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        try {
          const result = await post('/system/menus/batch-delete', {
            ids: selectedRowKeys,
            cascade: cascadeDelete,
          });

          if (result.data?.deletedCount) {
            message.success(`批量删除成功！共删除 ${result.data.deletedCount} 个菜单`);
          } else {
            message.success('批量删除成功');
          }

          setSelectedRowKeys([]);
          onLoad();

          // 刷新菜单缓存，使菜单变更立即生效
          await refreshMenuCache();
          message.success('菜单已刷新，变更已生效');
        } catch (error: any) {
          console.error('批量删除失败:', error);
          const errorMsg = error?.response?.data?.message || error?.message || '批量删除失败';
          message.error(errorMsg);
        }
      },
    });
  }, [selectedRowKeys, cascadeDelete, onLoad, setSelectedRowKeys]);

  // 保存菜单
  const handleSave = useCallback(async (editForm: any) => {
    try {
      const values = await editForm.validateFields();

      // 转换Switch状态值
      // visible: true->1(显示), false->0(隐藏)
      // status: true->0(正常), false->1(停用)
      if (typeof values.visible === 'boolean') {
        values.visible = values.visible ? 1 : 0;
      }
      if (typeof values.status === 'boolean') {
        values.status = values.status ? 0 : 1;
      }

      if (editingMenu) {
        await post(`/system/menus/${editingMenu.id}/update`, {
          ...values,
          id: editingMenu.id,
        });
      } else {
        await post('/system/menus', values);
      }

      message.success(editingMenu ? '更新成功' : '创建成功');
      setEditingMenu(null);
      onLoad();

      // 刷新菜单缓存，使菜单变更立即生效
      await refreshMenuCache();
      message.success('菜单已刷新，变更已生效');
    } catch (error) {
      console.error('保存菜单失败:', error);
    }
  }, [editingMenu, onLoad]);

  return {
    editingMenu,
    cascadeDelete,
    setCascadeDelete,
    handleAdd,
    handleEdit,
    handleDeleteConfirm,
    handleBatchDelete,
    handleSave,
    setEditingMenu,
  };
}
