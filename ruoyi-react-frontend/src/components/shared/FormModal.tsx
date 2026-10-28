/**
 * Phase 66 契约组件 — FormModal（BaseEditModal + 脏检查 + 统一提交/错误分流，UI-SPEC §3）
 *
 * 用法：
 * ```tsx
 * const form = useFormModalForm();
 * <FormModal
 *   open={open} title="岗位管理" form={form}
 *   onFinish={async (values) => { await saveApi(values); }}
 *   onCancel={handleClose}
 *   initialValues={editing}
 * >
 *   <Form.Item name="postName" label="岗位名称" rules={[...]}>
 *     <AppInput />
 *   </Form.Item>
 * </FormModal>
 * ```
 *
 * 契约：open=true 时 resetFields()+setFieldsValue(initialValues)；onOk 包装 async
 * 提交（confirmLoading + cancelButtonDisabled 防重）；onCancel 时脏检查
 * （isFieldsTouched 且非提交中 → Modal.confirm 引导，63 §3 文案固定口径）；
 * onFinish 抛错按 isFormValidationError 前置分流（表单校验静默，其余
 * handleApiError(error, "保存")）且弹窗保持打开。
 * 注意：本文件是业务代码中 Modal.confirm 的唯一新增合法调用点（65 §3 豁免白名单）。
 * 硬约束：BaseEditModal 对外 API 零变化——Form 由本组件内部提供，页面只写 Form.Item。
 */
import { useEffect, useState } from "react";
import { Form, Modal } from "antd";
import type { FormInstance } from "antd";
import BaseEditModal from "@/components/modal/BaseEditModal";
import { handleApiError, isFormValidationError } from "@/utils/errorHandler";

export interface FormModalProps {
  open: boolean;
  title: string;
  /** 提交回调：收 validateFields 的值；抛错即失败路径（弹窗保持打开） */
  onFinish: (values: Record<string, unknown>) => Promise<void>;
  onCancel: () => void;
  /** useFormModalForm() 或 useTableManager.editForm 的实例 */
  form: FormInstance;
  /** open=true 时回填（编辑传整条记录） */
  initialValues?: Record<string, unknown> | null;
  width?: number;
  okText?: string;
  /** 脏检查开关（默认 true；onCancel 时 isFieldsTouched → 确认弹窗） */
  dirtyCheck?: boolean;
  /**
   * 遮罩点击关闭（默认 false，沿用 BaseEditModal / Phase 64 FIX-02 已收紧口径）。
   * 68.1-T5 minor#7：向后兼容可选透传位，仅供调用页按原弹窗取值显式声明；不得放宽默认值。
   */
  maskClosable?: boolean;
  /**
   * Esc 关闭（默认 false，同上）。68.1-T5 minor#7：向后兼容可选透传位。
   */
  keyboard?: boolean;
  /** Form.Item 列表；或 (form) => Form.Item 列表 */
  children: React.ReactNode | ((form: FormInstance) => React.ReactNode);
}

/** FormModal 专用 FormInstance 已拆至 ./hooks/useFormModalForm（react-refresh 组件文件只导组件） */

/**
 * FormModal — 63 惯例编辑弹窗封装（Form horizontal labelCol 4 / wrapperCol 20）。
 * Modal.confirm 文案（63 §3 固定口径，不得改动）：
 * 标题「更改尚未保存」/ 内容「当前修改尚未保存，关闭后将丢失，是否确认关闭？」/
 * okText「放弃更改」/ cancelText「继续编辑」/ ok 不带 danger。
 */
export function FormModal({
  open,
  title,
  onFinish,
  onCancel,
  form,
  initialValues,
  width = 520,
  okText = "保存",
  dirtyCheck = true,
  maskClosable = false,
  keyboard = false,
  children,
}: FormModalProps) {
  const [submitting, setSubmitting] = useState(false);

  // open=true：重置并回填（依赖 [open, initialValues]；受控数据用引用，勿传内联对象）
  useEffect(() => {
    if (open) {
      form.resetFields();
      if (initialValues) form.setFieldsValue(initialValues);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- form 实例由调用方持有，稳定
  }, [open, initialValues]);

  const handleOk = async () => {
    setSubmitting(true);
    try {
      const values = await form.validateFields();
      await onFinish(values as Record<string, unknown>);
    } catch (error) {
      // isFormValidationError 前置分流：表单校验错误 antd 已内联提示，静默；
      // 其余统一出口 handleApiError(error, "保存")。两种情况弹窗都保持打开。
      if (!isFormValidationError(error)) {
        handleApiError(error, "保存");
      }
      // rethrow 交给 antd onOk 语义（reject 不关闭）
      throw error;
    } finally {
      setSubmitting(false);
    }
  };

  const handleCancel = () => {
    if (dirtyCheck && !submitting && form.isFieldsTouched()) {
      Modal.confirm({
        title: "更改尚未保存",
        content: "当前修改尚未保存，关闭后将丢失，是否确认关闭？",
        okText: "放弃更改",
        cancelText: "继续编辑",
        onOk: () => {
          onCancel();
        },
      });
      return;
    }
    onCancel();
  };

  return (
    <BaseEditModal
      open={open}
      title={title}
      // Modal（非 confirm）不吞 onOk rejection：自行 catch 保持弹窗打开（错误已在
      // handleOk 内分流：表单校验静默 / handleApiError 统一出口）
      onOk={() => {
        handleOk().catch(() => {});
      }}
      onCancel={handleCancel}
      confirmLoading={submitting}
      cancelButtonDisabled={submitting}
      width={width}
      okText={okText}
      // 68.1-T5 minor#7：maskClosable/keyboard 默认 false（BaseEditModal 口径，不改变已收紧行为），
      // 调用页可按原弹窗取值显式声明
      // Phase 88-04（VISFIX-04）：对外 prop 名 maskClosable 冻结（v1.22 契约，调用方零改动）；
      // antd 6 新形态（mask={{ closable }}）的转发统一收敛在 BaseEditModal 内部，
      // 此处仍走本组件自有契约 prop，不新增 mask 透传（不扩契约）。
      maskClosable={maskClosable}
      keyboard={keyboard}
    >
      <Form form={form} layout="horizontal" labelCol={{ span: 4 }} wrapperCol={{ span: 20 }}>
        {typeof children === "function" ? children(form) : children}
      </Form>
    </BaseEditModal>
  );
}

export default FormModal;
