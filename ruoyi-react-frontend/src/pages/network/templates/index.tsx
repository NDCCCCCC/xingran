/**
 * Network Template Management Page
 * 网络模板管理页面
 */

import { useEffect, useState } from 'react';
import type { FC } from 'react';
import { Table, Button, Space, Form, Input, Select, Card, Row, Col, Statistic, message } from 'antd';
import {
  PlusOutlined,
  SearchOutlined,
  ReloadOutlined,
  DeleteOutlined,
  FileTextOutlined,
  AppstoreOutlined,
  SettingOutlined,
  FundOutlined,
  ExportOutlined,
} from '@ant-design/icons';
import type { ConfigTemplate } from '@/types';
import { batchExport } from '@/lib/api/networkApi';
import { useTemplateData, useTemplateModals } from './hooks';
import { getTemplateColumns } from './columns';
import {
  TemplateEditModal,
  TemplatePreviewModal,
  TemplateVariablesModal,
} from './modals';
import { VENDOR_OPTIONS, TEMPLATE_TYPE_OPTIONS } from './constants';
import { usePagination } from '@/hooks/usePagination';
import NetworkExport from '@/components/shared/NetworkExport';
import { BatchExportModal } from '@/components/shared';
import { DownloadOutlined } from '@ant-design/icons';

const { Option } = Select;

const TemplateManagement: FC = () => {
  const [searchForm] = Form.useForm();
  const [editForm] = Form.useForm();
  const [batchModalVisible, setBatchModalVisible] = useState(false);
  const [batchExporting, setBatchExporting] = useState(false);

  // 使用全局分页 hook
  const { paginationProps, setTotal } = usePagination();

  const {
    templates,
    loading,
    total,
    statistics,
    loadTemplates,
    loadStatistics,
  } = useTemplateData(searchForm, setTotal);

  const {
    editModalVisible,
    previewVisible,
    variablesModalVisible,
    editingTemplate,
    selectedRowKeys,
    previewContent,
    templateVariables,
    setPreviewVisible,
    setVariablesModalVisible,
    setSelectedRowKeys,
    openModal,
    closeModal,
    handleCreate,
    handleDelete,
    handleBatchDelete,
    handlePreview,
    handleClone,
    handleGetVariables,
  } = useTemplateModals();

  useEffect(() => {
    loadTemplates({ current: paginationProps.current, pageSize: paginationProps.pageSize });
    loadStatistics();
  }, [paginationProps.current, paginationProps.pageSize]);

  // 搜索
  const handleSearch = () => {
    loadTemplates({ current: paginationProps.current, pageSize: paginationProps.pageSize });
  };

  // 重置
  const handleReset = () => {
    searchForm.resetFields();
    loadTemplates({ current: paginationProps.current, pageSize: paginationProps.pageSize });
  };

  // 刷新
  const handleRefresh = () => {
    loadTemplates({ current: paginationProps.current, pageSize: paginationProps.pageSize });
    loadStatistics();
  };

  // 操作成功后刷新
  const handleSuccess = () => {
    loadTemplates({ current: paginationProps.current, pageSize: paginationProps.pageSize });
    loadStatistics();
  };

  const columns = getTemplateColumns({
    handlePreview,
    handleClone: (id: string) => handleClone(id, handleSuccess),
    handleDelete: (id: string) => handleDelete(id, handleSuccess),
    openModal: (record: ConfigTemplate) => openModal(record, editForm),
  });

  // 批量导出
  const handleBatchExport = async (entityTypes: string[]) => {
    setBatchExporting(true);
    try {
      const filename = await batchExport(entityTypes, {});
      message.success(`批量导出成功，文件: ${filename}`);
      setBatchModalVisible(false);
    } catch (error: any) {
      message.error(`批量导出失败：${error.message}`);
    } finally {
      setBatchExporting(false);
    }
  };

  return (
    <div>
      {/* 统计卡片 */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={6}>
          <Card>
            <Statistic title="模板总数" value={statistics.total} prefix={<FileTextOutlined />} />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="系统模板"
              value={statistics.system}
              styles={{ content: { color: '#faad14' } }}
              prefix={<AppstoreOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="自定义模板"
              value={statistics.custom}
              styles={{ content: { color: '#1890ff' } }}
              prefix={<SettingOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="初始化模板"
              value={statistics.init}
              styles={{ content: { color: '#52c41a' } }}
              prefix={<FundOutlined />}
            />
          </Card>
        </Col>
      </Row>

      {/* 搜索表单和操作按钮 */}
      <Card style={{ marginBottom: 16 }}>
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'flex-start',
            flexWrap: 'wrap',
            gap: '16px',
          }}
        >
          <Form form={searchForm} layout="inline" style={{ flex: 1, minWidth: 0 }}>
            <Form.Item name="templateName" label="模板名称">
              <Input placeholder="请输入模板名称" allowClear className="user-form-input" style={{ width: 150 }} />
            </Form.Item>
            <Form.Item name="templateCode" label="模板编码">
              <Input placeholder="请输入模板编码" allowClear className="user-form-input" style={{ width: 150 }} />
            </Form.Item>
            <Form.Item name="templateType" label="模板类型">
              <Select placeholder="请选择模板类型" allowClear className="user-form-input" style={{ width: 130 }}>
                {TEMPLATE_TYPE_OPTIONS.map((opt) => (
                  <Option key={opt.value} value={opt.value}>
                    {opt.label}
                  </Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item name="vendor" label="厂商">
              <Select placeholder="请选择厂商" allowClear className="user-form-input" style={{ width: 120 }}>
                {VENDOR_OPTIONS.map((opt) => (
                  <Option key={opt.value} value={opt.value}>
                    {opt.label}
                  </Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item>
              <Space>
                <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>
                  查询
                </Button>
                <Button icon={<ReloadOutlined />} onClick={handleReset}>
                  重置
                </Button>
                <Button icon={<ReloadOutlined />} onClick={handleRefresh}>
                  刷新
                </Button>
              </Space>
            </Form.Item>
          </Form>
          <Space>
            <Button type="primary" icon={<PlusOutlined />} onClick={() => openModal(undefined, editForm)}>
              新增
            </Button>
            <Button
              danger
              icon={<DeleteOutlined />}
              onClick={() => handleBatchDelete(selectedRowKeys, handleSuccess)}
              disabled={selectedRowKeys.length === 0}
            >
              批量删除
            </Button>
            <NetworkExport
              entityType="templates"
              entityName="配置模板"
              filters={Object.fromEntries(
                Object.entries(searchForm.getFieldsValue() as Record<string, unknown>).filter(([, v]) => v !== undefined && v !== null && v !== '')
              )}
              current={paginationProps?.current ?? 1}
              pageSize={paginationProps?.pageSize ?? 10}
            />
          </Space>{/* 批量导出 Modal */}

        <BatchExportModal

          visible={batchModalVisible}

          onConfirm={handleBatchExport}

          onCancel={() => setBatchModalVisible(false)}

          loading={batchExporting}

        />


        </div>
      </Card>

      {/* 模板表格 */}
      <Card>
        <Table
          rowSelection={{
            selectedRowKeys,
            onChange: setSelectedRowKeys,
          }}
          columns={columns}
          dataSource={templates}
          loading={loading}
          rowKey="id"
          scroll={{ x: 1500 }}
          pagination={paginationProps}
        />
      </Card>

      {/* 编辑模态框 */}
      <TemplateEditModal
        open={editModalVisible}
        editingTemplate={editingTemplate}
        onOk={async () => {
          await handleCreate(editingTemplate, editForm, handleSuccess);
        }}
        onCancel={() => closeModal(editForm)}
      />

      {/* 预览模态框 */}
      <TemplatePreviewModal
        open={previewVisible}
        content={previewContent}
        onClose={() => {
          setPreviewVisible(false);
        }}
      />

      {/* 变量查看模态框 */}
      <TemplateVariablesModal
        open={variablesModalVisible}
        variables={templateVariables}
        onClose={() => {
          setVariablesModalVisible(false);
        }}
      />
    </div>
  );
};

export default TemplateManagement;
