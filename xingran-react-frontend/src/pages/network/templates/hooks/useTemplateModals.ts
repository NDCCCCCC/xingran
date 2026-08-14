/**
 * Template Modals Hook
 * 模板模态框管理 Hook
 */

import { useState, useCallback } from "react";
import { App } from "antd";
import type { ConfigTemplate } from "@/types";
import type { FormInstance } from "antd/es/form";
import { post } from "@/lib/api";
import type { TemplateModalState } from "../types";

export interface UseTemplateModalsReturn extends TemplateModalState {
  setEditModalVisible: React.Dispatch<React.SetStateAction<boolean>>;
  setPreviewVisible: React.Dispatch<React.SetStateAction<boolean>>;
  setVariablesModalVisible: React.Dispatch<React.SetStateAction<boolean>>;
  setEditingTemplate: React.Dispatch<React.SetStateAction<ConfigTemplate | null>>;
  setSelectedRowKeys: React.Dispatch<React.SetStateAction<React.Key[]>>;
  setPreviewContent: React.Dispatch<React.SetStateAction<string>>;
  setTemplateVariables: React.Dispatch<React.SetStateAction<Record<string, unknown>>>;

  openModal: (record?: ConfigTemplate, editForm?: FormInstance<unknown>) => void;
  closeModal: (editForm?: FormInstance<unknown>) => void;
  handleCreate: (editingTemplate: ConfigTemplate | null, editForm: FormInstance<unknown>, onSuccess: () => void) => Promise<void>;
  handleDelete: (id: string, onSuccess: () => void) => Promise<void>;
  handleBatchDelete: (selectedRowKeys: React.Key[], onSuccess: () => void) => Promise<void>;
  handlePreview: (id: string) => Promise<void>;
  handleClone: (id: string, onSuccess: () => void) => Promise<void>;
  handleGetVariables: (id: string) => Promise<void>;
  handleApiError: (error: unknown, defaultMessage: string) => void;
}

export function useTemplateModals(): UseTemplateModalsReturn {
  const { message } = App.useApp();
  const [editModalVisible, setEditModalVisible] = useState(false);
  const [previewVisible, setPreviewVisible] = useState(false);
  const [variablesModalVisible, setVariablesModalVisible] = useState(false);
  const [editingTemplate, setEditingTemplate] = useState<ConfigTemplate | null>(null);
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [previewContent, setPreviewContent] = useState("");
  const [templateVariables, setTemplateVariables] = useState<Record<string, unknown>>({});

  const handleApiError = useCallback((error: unknown, defaultMessage: string) => {
    if (error && typeof error === "object" && "message" in error) {
      message.error(error.message as string);
    } else {
      message.error(defaultMessage);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
  }, []);

  const handleSuccess = useCallback((msg: string) => {
    message.success(msg);
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
  }, []);

  const openModal = useCallback((record?: ConfigTemplate, editForm?: FormInstance<unknown>) => {
    if (record && editForm) {
      setEditingTemplate(record);
      editForm.setFieldsValue({
        ...record,
        variables: record.variables ? JSON.stringify(record.variables, null, 2) : undefined,
      });
    } else if (editForm) {
      setEditingTemplate(null);
      editForm.resetFields();
      editForm.setFieldsValue({ templateType: "config" });
    }
    setEditModalVisible(true);
  }, []);

  const closeModal = useCallback((editForm?: FormInstance<unknown>) => {
    setEditModalVisible(false);
    if (editForm) {
      editForm.resetFields();
    }
    setEditingTemplate(null);
  }, []);

  const handleCreate = useCallback(
    async (editingTemplate: ConfigTemplate | null, editForm: FormInstance<unknown>, onSuccess: () => void) => {
      try {
        const formValues = await editForm.validateFields();
        const values = formValues as Record<string, unknown> & { variables?: string };
        const data = {
          ...(values as Record<string, unknown>),
          variables: values.variables ? JSON.parse(values.variables) : undefined,
        };
        if (editingTemplate) {
          await post(`/network/templates/${editingTemplate.id}/update`, data);
          handleSuccess("更新成功");
        } else {
          await post("/network/templates", data);
          handleSuccess("创建成功");
        }
        closeModal(editForm);
        onSuccess();
      } catch (error) {
        if (error && typeof error === "object" && "errorFields" in error) {
          return;
        }
        handleApiError(error, "操作失败");
      }
    },
    [closeModal, handleSuccess, handleApiError]
  );

  const handleDelete = useCallback(
    async (id: string, onSuccess: () => void) => {
      try {
        await post(`/network/templates/${id}/delete`, {});
        handleSuccess("删除成功");
        onSuccess();
      } catch (error) {
        handleApiError(error, "删除失败");
      }
    },
    [handleSuccess, handleApiError]
  );

  const handleBatchDelete = useCallback(
    async (selectedRowKeys: React.Key[], onSuccess: () => void) => {
      if (selectedRowKeys.length === 0) {
        message.warning("请选择要删除的数据");
        return;
      }
      try {
        await post("/network/templates/batch-delete", { ids: selectedRowKeys });
        handleSuccess("批量删除成功");
        setSelectedRowKeys([]);
        onSuccess();
      } catch (error) {
        handleApiError(error, "批量删除失败");
      }
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
    [handleSuccess, handleApiError]
  );

  const handlePreview = useCallback(async (id: string) => {
    try {
      const result = await post(`/network/templates/${id}/preview`, {}) as { data?: { content?: string } };
      setPreviewContent(result.data?.content || "");
      setPreviewVisible(true);
    } catch (error) {
      handleApiError(error, "预览失败");
    }
  }, [handleApiError]);

  const handleClone = useCallback(
    async (id: string, onSuccess: () => void) => {
      try {
        await post(`/network/templates/${id}/clone`, {});
        handleSuccess("克隆成功");
        onSuccess();
      } catch (error) {
        handleApiError(error, "克隆失败");
      }
    },
    [handleSuccess, handleApiError]
  );

  const handleGetVariables = useCallback(async (id: string) => {
    try {
      const result = await post(`/network/templates/${id}/variables`, {}) as { data?: { variables?: Record<string, unknown> } };
      setTemplateVariables(result.data?.variables || {});
      setVariablesModalVisible(true);
    } catch (error) {
      handleApiError(error, "获取变量失败");
    }
  }, [handleApiError]);

  return {
    editModalVisible,
    previewVisible,
    variablesModalVisible,
    editingTemplate,
    selectedRowKeys,
    previewContent,
    templateVariables,
    setEditModalVisible,
    setPreviewVisible,
    setVariablesModalVisible,
    setEditingTemplate,
    setSelectedRowKeys,
    setPreviewContent,
    setTemplateVariables,
    openModal,
    closeModal,
    handleCreate,
    handleDelete,
    handleBatchDelete,
    handlePreview,
    handleClone,
    handleGetVariables,
    handleApiError,
  };
}
