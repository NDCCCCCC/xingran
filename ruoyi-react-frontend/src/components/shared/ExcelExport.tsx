/**
 * Excel导出组件
 * 支持导出条件筛选、数据导出
 */

import { useState } from 'react';
import type { FC } from 'react';
import {
  Button,
  Form,
  Input,
  Select,
  Space,
  Modal,
  message,
} from 'antd';
import { ExportOutlined } from '@ant-design/icons';

const { Option } = Select;

export interface ExportFilter {
  name?: string;
  code?: string;
  status?: number;
}

export interface ExcelExportProps {
  entityType: string;
  entityName?: string;
  exportUrl?: string;
  statusOptions?: { value: number; label: string }[];
  visible?: boolean;
  onClose?: () => void;
}

const ExcelExport: FC<ExcelExportProps> = ({
  entityType,
  entityName = entityType,
  exportUrl = `/ops/${entityType}/export`,
  statusOptions,
  visible = false,
  onClose,
}) => {
  const [form] = Form.useForm();
  const [exporting, setExporting] = useState(false);

  const handleExport = async () => {
    const values = await form.validateFields();
    setExporting(true);

    try {
      const token = localStorage.getItem('token');
      const response = await fetch(exportUrl, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(values),
      });

      if (!response.ok) {
        throw new Error('导出失败');
      }

      const filename = extractFilename(response.headers.get('content-disposition'), entityName);
      await downloadFile(response, filename);

      message.success('导出成功');
      form.resetFields();
      onClose?.();
    } catch (error) {
      message.error('导出失败');
    } finally {
      setExporting(false);
    }
  };

  const handleReset = () => {
    form.resetFields();
  };

  return (
    <Modal
      title={`导出${entityName}数据`}
      open={visible}
      onCancel={onClose}
      footer={
        <Space>
          <Button onClick={onClose}>取消</Button>
          <Button onClick={handleReset}>重置</Button>
          <Button
            type="primary"
            icon={<ExportOutlined />}
            onClick={handleExport}
            loading={exporting}
          >
            导出
          </Button>
        </Space>
      }
      destroyOnHidden
    >
      <Form
        form={form}
        layout="vertical"
        initialValues={{
          status: undefined,
        }}
      >
        <Form.Item label="名称" name="name">
          <Input placeholder="请输入名称关键词（可选）" allowClear />
        </Form.Item>

        <Form.Item label="编码" name="code">
          <Input placeholder="请输入编码关键词（可选）" allowClear />
        </Form.Item>

        {statusOptions && statusOptions.length > 0 && (
          <Form.Item label="状态" name="status">
            <Select placeholder="请选择状态（可选）" allowClear>
              {statusOptions.map(option => (
                <Option key={option.value} value={option.value}>
                  {option.label}
                </Option>
              ))}
            </Select>
          </Form.Item>
        )}
      </Form>

      <div style={{ marginTop: 16, padding: 12, background: '#f5f5f5', borderRadius: 4 }}>
        <Space direction="vertical" size={4}>
          <div>• 不填写任何条件将导出所有数据</div>
          <div>• 导出文件为 .xlsx 格式</div>
          <div>• 大量数据导出可能需要较长时间，请耐心等待</div>
        </Space>
      </div>
    </Modal>
  );
};

function extractFilename(contentDisposition: string | null, entityName: string): string {
  if (!contentDisposition) {
    return `${entityName}_${Date.now()}.xlsx`;
  }
  const match = contentDisposition.match(/filename[^;=\n]*=((['"]).*?\2|[^;\n]*)/);
  if (match && match[1]) {
    return decodeURIComponent(match[1].replace(/['"]/g, ''));
  }
  return `${entityName}_${Date.now()}.xlsx`;
}

async function downloadFile(response: Response, filename: string): Promise<void> {
  const blob = await response.blob();
  const url = window.URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  window.URL.revokeObjectURL(url);
  document.body.removeChild(a);
}

export default ExcelExport;
