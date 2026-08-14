/**
 * Network Command Modals Hook
 * 网络命令模态框管理 Hook
 */

import { useCallback } from "react";
import { App } from "antd";
import type { FormInstance } from "antd/es/form";
import { post } from "@/lib/api";

export interface UseCommandModalsReturn {
  handleQuickCommand: (
    selectedRowKeys: string[],
    form: FormInstance<unknown>,
    onSuccess: () => void
  ) => Promise<void>;
  handleCancelExecution: (id: string, onSuccess: () => void) => Promise<void>;
}

export function useCommandModals(): UseCommandModalsReturn {
  const { message } = App.useApp();
  const handleQuickCommand = useCallback(
    async (selectedRowKeys: string[], form: FormInstance<unknown>, onSuccess: () => void) => {
      try {
        const values = await form.validateFields();
        await post("/network/command/quick", {
          ...(values as Record<string, unknown>),
          deviceIds: selectedRowKeys,
        });
        message.success("命令已分发，请查看执行进度");
        form.resetFields();
        onSuccess();
      } catch (error: unknown) {
        if ((error as { errorFields?: unknown }).errorFields) {
          return;
        }
        message.error("分发失败");
      }
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
    []
  );

  const handleCancelExecution = useCallback(async (id: string, onSuccess: () => void) => {
    try {
      await post(`/network/executions/${id}/cancel`, {});
      message.success("取消成功");
      onSuccess();
    } catch (error) {
      message.error("取消失败");
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
  }, []);

  return {
    handleQuickCommand,
    handleCancelExecution,
  };
}
