/**
 * Template Variables Modal
 * 模板变量查看模态框
 */

import { useMemo } from "react";
import { Modal, Button, Table } from "antd";
import { createSorter } from "@/utils/tableHelpers";

export interface VariableRow {
  key: string;
  description: string;
  defaultValue: string;
}

export interface TemplateVariablesModalProps {
  open: boolean;
  variables: Record<string, unknown>;
  onClose: () => void;
}

export function TemplateVariablesModal({ open, variables, onClose }: TemplateVariablesModalProps) {
  const dataSource = Object.entries(variables).map(([key, value]) => {
    const varValue = value as { default?: string; description?: string } | undefined;
    return {
      key,
      description: varValue?.description || "-",
      defaultValue: varValue?.default || String(value ?? "-"),
    };
  });

  const columns = useMemo(() => {
    const baseColumns = [
      { title: "变量名", dataIndex: "key", key: "key" },
      { title: "描述", dataIndex: "description", key: "description" },
      { title: "默认值", dataIndex: "defaultValue", key: "defaultValue" },
    ] as const;
    return baseColumns.map((col) => ({
      ...col,
      sorter: createSorter<VariableRow>(col.dataIndex as keyof VariableRow, "string"),
    }));
  }, []);

  return (
    <Modal
      title="模板变量"
      open={open}
      onCancel={onClose}
      footer={[
        <Button key="close" onClick={onClose}>
          关闭
        </Button>,
      ]}
      width={600}
    >
      {dataSource.length > 0 ? (
        <Table
          dataSource={dataSource}
          columns={columns}
          pagination={false}
          size="small"
          rowKey="key"
        />
      ) : (
        <p style={{ textAlign: "center", color: "var(--theme-text-tertiary, #999)" }}>
          此模板没有定义变量
        </p>
      )}
    </Modal>
  );
}
