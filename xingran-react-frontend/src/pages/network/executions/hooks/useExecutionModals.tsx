/**
 * Config Execution Modals Hook
 * 配置执行模态框状态管理 Hook
 */

import { useState, useCallback } from "react";
import { App, Modal } from "antd";
import type { FormInstance } from "antd/es/form";
import { post } from "@/lib/api";
import type { ModalState, ExecutionDataState } from "../types";

export interface UseExecutionModalsParams {
  dataState: ExecutionDataState;
  setDataState: React.Dispatch<React.SetStateAction<ExecutionDataState>>;
  loadExecutions: () => Promise<void>;
}

export interface UseExecutionModalsReturn {
  modalState: ModalState;
  selectedRowKeys: string[];

  setModalState: React.Dispatch<React.SetStateAction<ModalState>>;
  setSelectedRowKeys: React.Dispatch<React.SetStateAction<string[]>>;

  openExecuteModal: () => void;
  handleTemplateChange: (templateId: string) => void;
  handleExecuteByTemplate: (executeForm: FormInstance<unknown>) => Promise<void>;
  handleCancelExecution: (id: string) => Promise<void>;
  handleViewDetail: (record: { id: string }) => void;
  handleViewOutput: (output: string) => void;
  closeDetailDrawer: () => void;
  closeVariableModal: () => void;
  closeExecuteModal: () => void;
}

export function useExecutionModals(params: UseExecutionModalsParams): UseExecutionModalsReturn {
  const { message } = App.useApp();
  const { dataState, setDataState, loadExecutions } = params;

  const [modalState, setModalState] = useState<ModalState>({
    executeModalVisible: false,
    variableModalVisible: false,
    detailDrawerVisible: false,
  });

  const [selectedRowKeys, setSelectedRowKeys] = useState<string[]>([]);

  // 打开执行模态框
  const openExecuteModal = useCallback(async () => {
    // This should be called from the main component with access to loadDevices and loadTemplates
    setModalState(prev => ({ ...prev, executeModalVisible: true }));
    setSelectedRowKeys([]);
    setDataState(prev => ({ ...prev, selectedTemplate: null }));
  }, [setDataState]);

  // 模板选择变化
  const handleTemplateChange = useCallback((templateId: string) => {
    const template = dataState.templates.find(t => t.id === templateId);
    if (template) {
      setDataState(prev => ({ ...prev, selectedTemplate: template }));
      // 解析模板变量
      if (template.variables && Object.keys(template.variables).length > 0) {
        setModalState(prev => ({ ...prev, variableModalVisible: true }));
      }
    }
  }, [dataState.templates, setDataState]);

  // 通过模板执行配置
  const handleExecuteByTemplate = useCallback(async (executeForm: FormInstance<unknown>) => {
    try {
      const formValues = await executeForm.validateFields();
      const values = formValues as {
        executionName: string;
        templateId: string;
        templateVariables?: Record<string, unknown>;
      };
      const data = {
        executionName: values.executionName,
        templateId: values.templateId,
        deviceIds: selectedRowKeys,
        templateVariables: values.templateVariables || {},
      };
      await post("/network/executions/template/execute", data);
      message.success("配置执行任务已创建");
      setModalState(prev => ({ ...prev, executeModalVisible: false }));
      executeForm.resetFields();
      setSelectedRowKeys([]);
      setDataState(prev => ({ ...prev, selectedTemplate: null }));
      loadExecutions();
    } catch (error: unknown) {
      if ((error as { errorFields?: unknown }).errorFields) {
        return;
      }
      message.error("创建执行任务失败");
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedRowKeys, loadExecutions, setDataState]);

  // 取消执行
  const handleCancelExecution = useCallback(async (id: string) => {
    try {
      await post(`/network/executions/${id}/cancel`, {});
      message.success("取消成功");
      loadExecutions();
    } catch (_error) {
      message.error("取消失败");
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [loadExecutions]);

  // 查看执行详情
  const handleViewDetail = useCallback(async (_record: Record<string, unknown>) => {
    // This should be called from the main component with access to loadExecutionDetails
    setModalState(prev => ({ ...prev, detailDrawerVisible: true }));
  }, []);

  // 查看输出
  const handleViewOutput = useCallback((output: string) => {
    Modal.info({
      title: "配置输出",
      width: 800,
      content: (
        <pre style={{ maxHeight: 500, overflow: "auto", background: "#f5f5f5", padding: 12, ["white-space"]: "pre-wrap" } as React.CSSProperties}>
          {output}
        </pre>
      ),
    });
  }, []);

  // 关闭详情抽屉
  const closeDetailDrawer = useCallback(() => {
    setModalState(prev => ({ ...prev, detailDrawerVisible: false }));
  }, []);

  // 关闭变量模态框
  const closeVariableModal = useCallback(() => {
    setModalState(prev => ({ ...prev, variableModalVisible: false }));
  }, []);

  // 关闭执行模态框
  const closeExecuteModal = useCallback(() => {
    setModalState(prev => ({ ...prev, executeModalVisible: false }));
  }, []);

  return {
    modalState,
    selectedRowKeys,
    setModalState,
    setSelectedRowKeys,
    openExecuteModal,
    handleTemplateChange,
    handleExecuteByTemplate,
    handleCancelExecution,
    handleViewDetail,
    handleViewOutput,
    closeDetailDrawer,
    closeVariableModal,
    closeExecuteModal,
  };
}
