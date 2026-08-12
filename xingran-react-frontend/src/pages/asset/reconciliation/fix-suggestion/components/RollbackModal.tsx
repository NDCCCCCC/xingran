/**
 * RollbackModal — 回滚修复建议 Modal(Phase 46 R5 / 46-02)
 *
 * 用途:
 *   - 从修复建议列表/详情触发,要求操作员填写 ≥10 字符的回滚原因(D-C3 审计)
 *   - 由 onSubmit 回调把原因传给父组件的 handleRollbackSubmit
 *
 * Props:
 *   - open: Modal 是否打开
 *   - onCancel: 取消(关闭 Modal + 重置表单)
 *   - onSubmit: 提交回调(reason: string) => Promise<void> | void
 *   - submitting: 提交中态(禁用按钮 + 显示 loading)
 *
 * 状态机约束:
 *   - reason < 10 字符:antd Form rule 拦截 + 提示
 *   - reason >= 10 字符:onSubmit 调起父组件 mutation
 *
 * 关联:
 *   - D-C3 强写 operlog OperTypeReset=11(后端 handler 调)
 *   - D-C4 缓存失效(InvalidateWorkstationHealth by wsID)
 */

import { Form, Input, Modal } from "antd";

interface RollbackModalProps {
  open: boolean;
  onCancel: () => void;
  onSubmit: (reason: string) => Promise<void> | void;
  submitting?: boolean;
}

interface RollbackFormValues {
  rollbackReason: string;
}

export const RollbackModal = ({ open, onCancel, onSubmit, submitting }: RollbackModalProps) => {
  const [form] = Form.useForm<RollbackFormValues>();

  const handleOk = async () => {
    try {
      const values = await form.validateFields();
      await onSubmit(values.rollbackReason.trim());
      form.resetFields();
    } catch (err) {
      // antd validateFields 失败会抛错(已自动显示 message),无需额外处理
      if (err && typeof err === "object" && "errorFields" in err) {
        return;
      }
      // 业务错误(父组件 reject)继续抛给父组件处理
      throw err;
    }
  };

  const handleCancel = () => {
    form.resetFields();
    onCancel();
  };

  return (
    <Modal
      title="回滚修复"
      open={open}
      confirmLoading={submitting}
      onCancel={handleCancel}
      onOk={handleOk}
      okText="确认回滚"
      okButtonProps={{ danger: true }}
      destroyOnClose
    >
      <Form form={form} layout="vertical" preserve={false}>
        <Form.Item
          name="rollbackReason"
          label="回滚原因"
          rules={[
            { required: true, message: "回滚原因不能为空" },
            { min: 10, message: "回滚原因至少 10 字符" },
          ]}
        >
          <Input.TextArea rows={3} placeholder="请说明回滚原因(≥10 字符,7d 窗口内才允许)" maxLength={500} showCount />
        </Form.Item>
      </Form>
    </Modal>
  );
};
