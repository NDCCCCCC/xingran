/**
 * Network Command Modals Hook
 * 网络命令模态框管理 Hook
 */

import { useCallback } from 'react';
import { message } from 'antd';
import { post } from '@/lib/api';

export interface UseCommandModalsReturn {
  handleQuickCommand: (
    selectedRowKeys: string[],
    form: any,
    onSuccess: () => void
  ) => Promise<void>;
  handleCancelExecution: (id: string, onSuccess: () => void) => Promise<void>;
}

export function useCommandModals(): UseCommandModalsReturn {
  const handleQuickCommand = useCallback(
    async (selectedRowKeys: string[], form: any, onSuccess: () => void) => {
      try {
        const values = await form.validateFields();
        await post('/network/command/quick', {
          ...values,
          deviceIds: selectedRowKeys,
        });
        message.success('命令已分发，请查看执行进度');
        form.resetFields();
        onSuccess();
      } catch (error: any) {
        if (error.errorFields) {
          return;
        }
        message.error('分发失败');
      }
    },
    []
  );

  const handleCancelExecution = useCallback(async (id: string, onSuccess: () => void) => {
    try {
      await post(`/network/command/${id}/cancel`, {});
      message.success('取消成功');
      onSuccess();
    } catch (error) {
      message.error('取消失败');
    }
  }, []);

  return {
    handleQuickCommand,
    handleCancelExecution,
  };
}
