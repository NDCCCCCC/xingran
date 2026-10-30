/**
 * Captcha Background Edit Modal
 * 验证码背景编辑模态框
 */

import { Modal, Form } from "antd";
// Phase 71 (W2)：裸 Select/Input/TextArea → 契约组件。四处 Select 均为动态 children 固定
// 枚举 → 改写 options 形态；形状/难度/状态业务禁搜 → 显式 showSearch={false}
import { AppInput, AppSelect, AppTextArea } from "@/components/ui";
import { useState } from "react";
import type { CaptchaBackground } from "@/types/captcha";
import { SHAPE_OPTIONS, DIFFICULTY_OPTIONS, STATUS_OPTIONS } from "../constants";

export interface CaptchaEditModalProps {
  open: boolean;
  editingBg: CaptchaBackground | null;
  onOk: () => Promise<void>;
  onCancel: () => void;
}

export function CaptchaEditModal({ open, editingBg: _editingBg, onOk, onCancel }: CaptchaEditModalProps) {
  const [form] = Form.useForm();
  const [submitting, setSubmitting] = useState(false);

  return (
    <Modal
      title="编辑背景图"
      open={open}
      onOk={async () => {
        setSubmitting(true);
        try {
          await onOk();
        } finally {
          setSubmitting(false);
        }
      }}
      cancelButtonProps={{ disabled: submitting }}
      onCancel={() => {
        onCancel();
        form.resetFields();
      }}
      width={600}
      destroyOnHidden
    >
      <Form form={form} labelCol={{ span: 6 }} wrapperCol={{ span: 16 }}>
        <Form.Item name="pieceShape" label="拼图形状" rules={[{ required: true }]}>
          <AppSelect
            allowClear
            onSearch={() => {}}
            showSearch={false}
            options={[...SHAPE_OPTIONS]}
          />
        </Form.Item>
        <Form.Item name="difficultyLevel" label="难度级别" rules={[{ required: true }]}>
          <AppSelect
            allowClear
            onSearch={() => {}}
            showSearch={false}
            options={[...DIFFICULTY_OPTIONS]}
          />
        </Form.Item>
        <Form.Item name="allowedShapes" label="允许的形状">
          {/* mode="multiple"：多选需搜索辅助定位，显式 showSearch */}
          <AppSelect
            allowClear
            mode="multiple"
            placeholder="不限制则使用默认"
            onSearch={() => {}}
            showSearch
            options={[...SHAPE_OPTIONS]}
          />
        </Form.Item>
        <Form.Item name="status" label="状态" rules={[{ required: true }]}>
          <AppSelect
            allowClear
            onSearch={() => {}}
            showSearch={false}
            options={[...STATUS_OPTIONS]}
          />
        </Form.Item>
        <Form.Item name="sortOrder" label="排序" rules={[{ required: true }]}>
          <AppInput type="number" placeholder="数字越小优先级越高" />
        </Form.Item>
        <Form.Item name="remark" label="备注">
          <AppTextArea rows={3} placeholder="请输入备注" />
        </Form.Item>
      </Form>
    </Modal>
  );
}
