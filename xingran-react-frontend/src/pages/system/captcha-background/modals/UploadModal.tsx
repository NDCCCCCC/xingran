/**
 * Captcha Background Upload Modal
 * 验证码背景上传模态框
 */

import { Modal, Form, Select, Upload, Input } from "antd";
import type { UploadProps } from "antd/es/upload";
import { UploadOutlined } from "@ant-design/icons";
import { SHAPE_OPTIONS, DIFFICULTY_OPTIONS } from "../constants";

const { Option } = Select;
const { TextArea } = Input;

export interface CaptchaUploadModalProps {
  open: boolean;
  uploading: boolean;
  uploadProps: UploadProps;
  onOk: () => Promise<void>;
  onCancel: () => void;
}

export function CaptchaUploadModal({
  open,
  uploading,
  uploadProps,
  onOk,
  onCancel,
}: CaptchaUploadModalProps) {
  const [form] = Form.useForm();

  return (
    <Modal
      title="上传背景图"
      open={open}
      onOk={async () => {
        await onOk();
      }}
      onCancel={() => {
        onCancel();
        form.resetFields();
      }}
      confirmLoading={uploading}
      width={600}
      destroyOnHidden
    >
      <Form form={form} labelCol={{ span: 6 }} wrapperCol={{ span: 16 }}>
        <Form.Item label="图片文件" required>
          <Upload {...uploadProps}>
            {(uploadProps.fileList?.length ?? 0) === 0 && (
              <div>
                <UploadOutlined />
                <div style={{ marginTop: 8 }}>选择图片</div>
              </div>
            )}
          </Upload>
        </Form.Item>
        <Form.Item
          name="pieceShape"
          label="拼图形状"
          rules={[{ required: true, message: "请选择拼图形状" }]}
        >
          <Select onSearch={() => {}}>
            {SHAPE_OPTIONS.map((opt) => (
              <Option key={opt.value} value={opt.value}>
                {opt.label}
              </Option>
            ))}
          </Select>
        </Form.Item>
        <Form.Item
          name="difficultyLevel"
          label="难度级别"
          rules={[{ required: true, message: "请选择难度" }]}
        >
          <Select onSearch={() => {}}>
            {DIFFICULTY_OPTIONS.map((opt) => (
              <Option key={opt.value} value={opt.value}>
                {opt.label}
              </Option>
            ))}
          </Select>
        </Form.Item>
        <Form.Item name="allowedShapes" label="允许的形状">
          <Select mode="multiple" placeholder="不限制则默认使用当前形状" onSearch={() => {}}>
            {SHAPE_OPTIONS.map((opt) => (
              <Option key={opt.value} value={opt.value}>
                {opt.label}
              </Option>
            ))}
          </Select>
        </Form.Item>
        <Form.Item name="remark" label="备注">
          <TextArea rows={3} placeholder="请输入备注" />
        </Form.Item>
      </Form>
    </Modal>
  );
}
