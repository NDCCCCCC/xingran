/**
 * Template Variables Modal
 * 模板变量查看模态框
 */

import { Modal, Button, Table } from 'antd';

export interface VariableRow {
  key: string;
  description: string;
  defaultValue: string;
}

interface TemplateVariable {
  default?: string;
  description?: string;
}

export interface TemplateVariablesModalProps {
  open: boolean;
  variables: Record<string, TemplateVariable>;
  onClose: () => void;
}

export function TemplateVariablesModal({ open, variables, onClose }: TemplateVariablesModalProps) {
  const dataSource = Object.entries(variables).map(([key, value]: [string, TemplateVariable]) => ({
    key,
    description: value?.description || '-',
    defaultValue: value?.default || '-',
  }));

  const columns = [
    { title: '变量名', dataIndex: 'key', key: 'key' },
    { title: '描述', dataIndex: 'description', key: 'description' },
    { title: '默认值', dataIndex: 'defaultValue', key: 'defaultValue' },
  ];

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
        <p style={{ textAlign: 'center', color: '#999' }}>此模板没有定义变量</p>
      )}
    </Modal>
  );
}
