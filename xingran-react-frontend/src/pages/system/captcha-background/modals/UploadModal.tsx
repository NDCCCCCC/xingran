/**
 * Captcha Background Upload Modal
 * 验证码背景上传模态框
 */

import { Modal, Form, Upload } from "antd";
// Phase 71 (W2)：裸 Select/TextArea → 契约组件。三处 Select 均为动态 children 形态的固定
// 枚举 → 改写 options 形态（AppSelect 无条件 optionFilterProp="label"，children 下破坏性）
import { AppSelect, AppTextArea } from "@/components/ui";
import type { UploadProps } from "antd/es/upload";
import { UploadOutlined } from "@ant-design/icons";
import { SHAPE_OPTIONS, DIFFICULTY_OPTIONS } from "../constants";

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
      cancelButtonProps={{ disabled: uploading }}
      onCancel={() => {
        onCancel();
        form.resetFields();
      }}
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
          {/* 形态分类器②③④：固定枚举（业务禁搜）→ options 形态 + showSearch={false} */}
          <AppSelect
            allowClear
            onSearch={() => {}}
            showSearch={false}
            options={[...SHAPE_OPTIONS]}
          />
        </Form.Item>
        <Form.Item
          name="difficultyLevel"
          label="难度级别"
          rules={[{ required: true, message: "请选择难度" }]}
        >
          {/* 形态分类器②③④：固定枚举（业务禁搜）→ options 形态 + showSearch={false} */}
          <AppSelect
            allowClear
            onSearch={() => {}}
            showSearch={false}
            options={[...DIFFICULTY_OPTIONS]}
          />
        </Form.Item>
        <Form.Item name="allowedShapes" label="允许的形状">
          {/* mode="multiple"：多选场景需搜索辅助定位，显式 showSearch（不依赖 count>=8） */}
          <AppSelect
            allowClear
            mode="multiple"
            placeholder="不限制则默认使用当前形状"
            onSearch={() => {}}
            showSearch
            options={[...SHAPE_OPTIONS]}
          />
        </Form.Item>
        <Form.Item name="remark" label="备注">
          <AppTextArea rows={3} placeholder="请输入备注" />
        </Form.Item>
      </Form>
    </Modal>
  );
}
