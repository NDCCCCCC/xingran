/**
 * Captcha Background Edit Modal
 * 验证码背景编辑模态框
 */

import { Modal, Form, Select, Input } from "antd";
import type { CaptchaBackground } from "@/types/captcha";
import { SHAPE_OPTIONS, DIFFICULTY_OPTIONS, STATUS_OPTIONS } from "../constants";

const { Option } = Select;
const { TextArea } = Input;

export interface CaptchaEditModalProps {
  open: boolean;
  editingBg: CaptchaBackground | null;
  onOk: () => Promise<void>;
  onCancel: () => void;
}

export function CaptchaEditModal({ open, editingBg, onOk, onCancel }: CaptchaEditModalProps) {
  const [form] = Form.useForm();

  return (
    <Modal
      title="编辑背景图"
      open={open}
      onOk={async () => {
        await onOk();
      }}
      onCancel={() => {
        onCancel();
        form.resetFields();
      }}
      width={600}
      destroyOnHidden
    >
      <Form form={form} labelCol={{ span: 6 }} wrapperCol={{ span: 16 }}>
        <Form.Item name="pieceShape" label="拼图形状" rules={[{ required: true }]}>
          <Select onSearch={() => {}}>
            {SHAPE_OPTIONS.map((opt) => (
              <Option key={opt.value} value={opt.value}>
                {opt.label}
              </Option>
            ))}
          </Select>
        </Form.Item>
        <Form.Item name="difficultyLevel" label="难度级别" rules={[{ required: true }]}>
          <Select onSearch={() => {}}>
            {DIFFICULTY_OPTIONS.map((opt) => (
              <Option key={opt.value} value={opt.value}>
                {opt.label}
              </Option>
            ))}
          </Select>
        </Form.Item>
        <Form.Item name="allowedShapes" label="允许的形状">
          <Select mode="multiple" placeholder="不限制则使用默认" onSearch={() => {}}>
            {SHAPE_OPTIONS.map((opt) => (
              <Option key={opt.value} value={opt.value}>
                {opt.label}
              </Option>
            ))}
          </Select>
        </Form.Item>
        <Form.Item name="status" label="状态" rules={[{ required: true }]}>
          <Select onSearch={() => {}}>
            {STATUS_OPTIONS.map((opt) => (
              <Option key={opt.value} value={opt.value}>
                {opt.label}
              </Option>
            ))}
          </Select>
        </Form.Item>
        <Form.Item name="sortOrder" label="排序" rules={[{ required: true }]}>
          <Input type="number" placeholder="数字越小优先级越高" />
        </Form.Item>
        <Form.Item name="remark" label="备注">
          <TextArea rows={3} placeholder="请输入备注" />
        </Form.Item>
      </Form>
    </Modal>
  );
}
