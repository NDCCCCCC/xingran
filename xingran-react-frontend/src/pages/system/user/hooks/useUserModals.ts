/**
 * User 模态框状态管理 Hook
 */

import { useState, useCallback } from "react";
import { Form } from "antd";
import type { User } from "@/types";
import type { FormInstance } from "antd/es/form";

export interface UseUserModalsReturn {
  resetPasswordModalVisible: boolean;
  resettingUser: User | null;
  resetPasswordForm: FormInstance<unknown>;
  setResetPasswordModalVisible: (visible: boolean) => void;
  setResettingUser: (user: User | null) => void;
  openResetPasswordModal: (user: User) => void;
  closeResetPasswordModal: () => void;
}

export function useUserModals(): UseUserModalsReturn {
  const [resetPasswordModalVisible, setResetPasswordModalVisible] = useState(false);
  const [resettingUser, setResettingUser] = useState<User | null>(null);
  const [resetPasswordForm] = Form.useForm();

  // 打开重置密码模态框
  const openResetPasswordModal = useCallback(
    (user: User) => {
      setResettingUser(user);
      resetPasswordForm.resetFields();
      setResetPasswordModalVisible(true);
    },
    [resetPasswordForm]
  );

  // 关闭重置密码模态框
  const closeResetPasswordModal = useCallback(() => {
    setResetPasswordModalVisible(false);
    setResettingUser(null);
  }, []);

  return {
    resetPasswordModalVisible,
    resettingUser,
    resetPasswordForm,
    setResetPasswordModalVisible,
    setResettingUser,
    openResetPasswordModal,
    closeResetPasswordModal,
  };
}
