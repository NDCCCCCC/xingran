/**
 * Template Variable Modal
 * 模板变量模态框
 */

import { Modal, Form, Input } from "antd";
import type { FormInstance } from "antd/es/form";
import type { ConfigTemplate } from "@/types";

export interface VariableModalProps {
  open: boolean;
  selectedTemplate: ConfigTemplate | null;
  form: FormInstance<unknown>;
  onOk: () => void;
  onCancel: () => void;
}

export function VariableModal({
  open,
  selectedTemplate,
  form,
  onOk,
  onCancel,
}: VariableModalProps) {
  return (
    <Modal title="模板变量" open={open} onOk={onOk} onCancel={onCancel} width={600}>
      <Form form={form} layout="vertical">
        {selectedTemplate?.variables &&
          Object.entries(selectedTemplate.variables).map(([key, defaultValue]) => (
            <Form.Item key={key} name={["templateVariables", key]} label={key}>
              <Input placeholder={`默认值: ${String(defaultValue)}`} />
            </Form.Item>
          ))}
      </Form>
    </Modal>
  );
}
