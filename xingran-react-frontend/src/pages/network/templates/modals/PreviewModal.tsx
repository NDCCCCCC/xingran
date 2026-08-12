/**
 * Template Preview Modal
 * 模板预览模态框
 */

import { Modal, Button } from "antd";

export interface TemplatePreviewModalProps {
  open: boolean;
  content: string;
  onClose: () => void;
}

export function TemplatePreviewModal({ open, content, onClose }: TemplatePreviewModalProps) {
  return (
    <Modal
      title="模板预览"
      open={open}
      onCancel={onClose}
      footer={[
        <Button key="close" onClick={onClose}>
          关闭
        </Button>,
      ]}
      width={800}
    >
      <pre
        style={{
          background: "#f5f5f5",
          padding: 16,
          borderRadius: 4,
          maxHeight: 500,
          overflow: "auto",
        }}
      >
        {content}
      </pre>
    </Modal>
  );
}
