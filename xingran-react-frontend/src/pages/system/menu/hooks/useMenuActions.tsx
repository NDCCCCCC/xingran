/**
 * Menu Actions Hook
 * 菜单操作管理 Hook
 */

import { useState, useCallback, useRef, useEffect } from "react";
import { App, Modal, Alert, Checkbox } from "antd";
import { WarningOutlined } from "@ant-design/icons";
import type { Menu } from "@/types";
import type { FormInstance } from "antd/es/form";
import { post } from "@/lib/api";
import type { CheckboxChangeEvent } from "antd/es/checkbox";
import { refreshMenuCache } from "@/store/menuStore";

// 扩展 Modal 实例类型以支持 checkboxRef
interface ModalInstanceWithCheckbox {
  destroy: () => void;
  update: (configUpdate: Record<string, unknown>) => void;
  checkboxRef?: React.MutableRefObject<boolean>;
}

export interface UseMenuActionsParams {
  onLoad: () => void;
  selectedRowKeys: string[];
  setSelectedRowKeys: (keys: string[]) => void;
  onSaveSuccess?: () => void;
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
  handleSave: (editForm: FormInstance<unknown>) => Promise<void>;
  setEditingMenu: (menu: Menu | null) => void;
}

export function useMenuActions(params: UseMenuActionsParams): UseMenuActionsReturn {
  const { message } = App.useApp();
  const { onLoad, selectedRowKeys, setSelectedRowKeys, onSaveSuccess } = params;

  const [editingMenu, setEditingMenu] = useState<Menu | null>(null);
  // 批量删除使用的级联删除状态
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
  const handleDelete = useCallback(
    async (id: string, cascade: boolean = false) => {
      try {
        const url = cascade
          ? `/system/menus/${id}/delete?cascade=true`
          : `/system/menus/${id}/delete`;

        const result = (await post(url)) as { data?: { deletedCount?: number } };

        if (cascade && result.data?.deletedCount) {
          message.success(`删除成功！共删除 ${result.data.deletedCount} 个菜单（包含子菜单）`);
        } else {
          message.success("删除成功");
        }

        onLoad();

        // 静默刷新菜单缓存，不会触发页面跳转
        refreshMenuCache();
      } catch (error: unknown) {
        console.error("删除菜单失败:", error);
        let errorMsg = "删除失败";
        if (error && typeof error === "object") {
          if ("response" in error && typeof error.response === "object" && error.response) {
            if (
              "data" in error.response &&
              typeof error.response.data === "object" &&
              error.response.data
            ) {
              if (
                "message" in error.response.data &&
                typeof error.response.data.message === "string"
              ) {
                errorMsg = error.response.data.message;
              }
            }
          } else if ("message" in error && typeof error.message === "string") {
            errorMsg = error.message;
          }
        }
        message.error(errorMsg);
        throw error; // 重新抛出错误以保持 Modal 打开
      }
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
    [onLoad]
  );

  // 单个删除确认（带级联删除选项）
  const handleDeleteConfirm = useCallback(
    (record: Menu) => {
      const hasChildren = record.children && record.children.length > 0;

      // 创建一个可响应状态更新的确认框
      let _localCascade = false;
      let modalInstance: ModalInstanceWithCheckbox | null = null;

      const DeleteConfirmContent = () => {
        const [checked, setChecked] = useState(false);

        // 使用 ref 来保存最新的状态值，供 onOk 回调使用
        const checkboxRef = useRef(false);

        const handleChange = (e: CheckboxChangeEvent) => {
          const newValue = e.target.checked;
          setChecked(newValue);
          checkboxRef.current = newValue;
          _localCascade = newValue;
        };

        // 将 checkboxRef 挂载到 modalInstance 上，供 onOk 访问
        useEffect(() => {
          if (modalInstance) {
            modalInstance.checkboxRef = checkboxRef;
          }
        }, [checkboxRef]);

        return (
          <div>
            <p>
              确定要删除菜单 <strong>{record.menuName}</strong> 吗？
            </p>
            {hasChildren && (
              <Alert
                message="该菜单下有子菜单，删除时请选择是否同时删除子菜单"
                type="warning"
                showIcon
                style={{ marginBottom: 16 }}
              />
            )}
            <div style={{ marginTop: hasChildren ? 16 : 0 }}>
              <Checkbox checked={checked} onChange={handleChange}>
                级联删除子菜单
              </Checkbox>
              {hasChildren && !checked && (
                <Alert
                  message="不启用级联删除时，如果菜单下有子菜单，删除将失败"
                  type="error"
                  showIcon
                  style={{ marginTop: 12 }}
                />
              )}
            </div>
          </div>
        );
      };

      modalInstance = Modal.confirm({
        title: "删除菜单",
        icon: <WarningOutlined style={{ color: "var(--theme-warning, #faad14)" }} />,
        content: <DeleteConfirmContent />,
        okText: "确定删除",
        okType: "danger",
        cancelText: "取消",
        onOk: () => {
          // 从 modalInstance 获取最新的 checkbox 状态
          const cascadeValue = modalInstance?.checkboxRef?.current || false;
          return handleDelete(record.id, cascadeValue);
        },
      }) as unknown as ModalInstanceWithCheckbox;
    },
    [handleDelete]
  );

  // 批量删除菜单
  const handleBatchDelete = useCallback(() => {
    if (selectedRowKeys.length === 0) {
      message.warning("请至少选择一个菜单");
      return;
    }

    // 创建一个可响应状态更新的确认框
    let modalInstance: ModalInstanceWithCheckbox | null = null;

    const BatchDeleteConfirmContent = () => {
      const [checked, setChecked] = useState(false);
      const checkboxRef = useRef(false);

      const handleChange = (e: CheckboxChangeEvent) => {
        const newValue = e.target.checked;
        setChecked(newValue);
        checkboxRef.current = newValue;
      };

      // 将 checkboxRef 挂载到 modalInstance 上，供 onOk 访问
      useEffect(() => {
        if (modalInstance) {
          modalInstance.checkboxRef = checkboxRef;
        }
      }, [checkboxRef]);

      return (
        <div>
          <p>
            您已选择 <strong>{selectedRowKeys.length}</strong> 个菜单，确定要删除吗？
          </p>
          <Alert
            message="删除后将无法恢复，请谨慎操作！"
            type="warning"
            showIcon
            style={{ marginBottom: 16 }}
          />
          <div style={{ marginTop: 16 }}>
            <Checkbox checked={checked} onChange={handleChange}>
              级联删除（同时删除所有子菜单）
            </Checkbox>
            {checked && (
              <Alert
                message="启用级联删除后，将同时删除所选菜单及其所有子菜单"
                type="info"
                showIcon
                style={{ marginTop: 12 }}
              />
            )}
          </div>
        </div>
      );
    };

    modalInstance = Modal.confirm({
      title: "批量删除菜单",
      icon: <WarningOutlined style={{ color: "var(--theme-warning, #faad14)" }} />,
      content: <BatchDeleteConfirmContent />,
      okText: "确定删除",
      okType: "danger",
      cancelText: "取消",
      onOk: async () => {
        try {
          // 从 modalInstance 获取最新的 checkbox 状态
          const cascadeValue = modalInstance?.checkboxRef?.current || false;
          const result = (await post("/system/menus/batch-delete", {
            ids: selectedRowKeys,
            cascade: cascadeValue,
          })) as { data?: { deletedCount?: number } };

          if (result.data?.deletedCount) {
            message.success(`批量删除成功！共删除 ${result.data.deletedCount} 个菜单`);
          } else {
            message.success("批量删除成功");
          }

          setSelectedRowKeys([]);
          onLoad();

          // 静默刷新菜单缓存，不会触发页面跳转
          refreshMenuCache();
        } catch (error: unknown) {
          console.error("批量删除失败:", error);
          const err = error as { response?: { data?: { message?: string } }; message?: string };
          const errorMsg = err?.response?.data?.message || err?.message || "批量删除失败";
          message.error(errorMsg);
          throw error; // 重新抛出错误以保持 Modal 打开
        }
      },
    }) as unknown as ModalInstanceWithCheckbox;
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
  }, [selectedRowKeys, onLoad, setSelectedRowKeys]);

  // 保存菜单
  const handleSave = useCallback(
    async (editForm: FormInstance<unknown>) => {
      try {
        const values = (await editForm.validateFields()) as Record<string, unknown>;

        // 转换Switch状态值
        // visible: true->1(显示), false->0(隐藏)
        // status: true->0(正常), false->1(停用)
        if (typeof values.visible === "boolean") {
          values.visible = values.visible ? 1 : 0;
        }
        if (typeof values.status === "boolean") {
          values.status = values.status ? 0 : 1;
        }

        if (editingMenu) {
          await post(`/system/menus/${editingMenu.id}/update`, {
            ...values,
            id: editingMenu.id,
          });
        } else {
          await post("/system/menus", values);
        }

        message.success(editingMenu ? "更新成功" : "创建成功");
        setEditingMenu(null);
        onLoad();

        // 静默刷新菜单缓存，不会触发页面跳转
        refreshMenuCache();

        // 调用保存成功回调，用于关闭模态框等操作
        onSaveSuccess?.();
      } catch (error) {
        console.error("保存菜单失败:", error);
      }
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
    [editingMenu, onLoad, onSaveSuccess]
  );

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
