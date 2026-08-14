/**
 * Network Template Management Page
 * 网络模板管理页面
 */

import { useEffect, useState, useMemo, useCallback, useRef } from "react";
import type { FC } from "react";
import { Table, Button, Space, Form, Input, Select, Card, Row, Col, Statistic, App } from "antd";
import {
  PlusOutlined,
  SearchOutlined,
  ReloadOutlined,
  DeleteOutlined,
  FileTextOutlined,
  AppstoreOutlined,
  SettingOutlined,
  FundOutlined,
} from "@ant-design/icons";
import type { ConfigTemplate } from "@/types";
import { batchExport } from "@/lib/api/networkApi";
import { useTemplateData, useTemplateModals } from "./hooks";
import { useServerSort, resolveSorter } from "@/hooks/useServerSort";
import type { SorterMeta } from "@/utils/tableHelpers";
import { createSorterMeta } from "@/utils/tableHelpers";
import { getTemplateColumns } from "./columns";
import { TemplateEditModal, TemplatePreviewModal, TemplateVariablesModal } from "./modals";
import { VENDOR_OPTIONS, TEMPLATE_TYPE_OPTIONS } from "./constants";
import { usePagination } from "@/hooks/usePagination";
import NetworkExport from "@/components/shared/NetworkExport";
import { BatchExportModal } from "@/components/shared";
import { DownloadOutlined } from "@ant-design/icons";

const { Option } = Select;

const TemplateManagement: FC = () => {
  const { message } = App.useApp();
  const [searchForm] = Form.useForm();
  const [editForm] = Form.useForm();
  const [batchModalVisible, setBatchModalVisible] = useState(false);
  const [batchExporting, setBatchExporting] = useState(false);

  // 使用全局分页 hook
  const { paginationProps, setTotal } = usePagination();

  const { templates, loading, total, statistics, loadTemplates, loadStatistics } = useTemplateData(
    searchForm,
    setTotal
  );

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

  const sorterMetas = useMemo<Array<SorterMeta<ConfigTemplate> | undefined>>(
    () => [
      createSorterMeta<ConfigTemplate>("templateName"),
      createSorterMeta<ConfigTemplate>("templateCode"),
      createSorterMeta<ConfigTemplate>("templateType"),
      createSorterMeta<ConfigTemplate>("vendor"),
      createSorterMeta<ConfigTemplate>("deviceType"),
      createSorterMeta<ConfigTemplate>("isSystem"),
      createSorterMeta<ConfigTemplate>("version"),
      createSorterMeta<ConfigTemplate>("description"),
      createSorterMeta<ConfigTemplate>("updatedAt", "date"),
    ],
    []
  );
  const sort = useServerSort<ConfigTemplate>({ sorterMetas });

  const getColumnSortOrder = useCallback(
    (field: string): "ascend" | "descend" | null | undefined => {
      if (sort.orderByColumn !== String(field)) return undefined;
      return sort.sortOrder;
    },
    [sort.orderByColumn, sort.sortOrder]
  );

  const sortRef = useRef<{ orderByColumn?: string; isAsc?: boolean }>({});

  const handleTableChange = useCallback(
    (
      pagination: { current?: number; pageSize?: number },
      _filters: Record<string, any>,
      sorter:
        | import("antd/es/table/interface").SorterResult<ConfigTemplate>
        | import("antd/es/table/interface").SorterResult<ConfigTemplate>[]
    ) => {
      const current = pagination.current ?? 1;
      const pageSize = pagination.pageSize ?? 10;
      sort.handleTableChange(pagination, _filters, sorter);
      const { orderByColumn, isAsc } = resolveSorter(sorter, sorterMetas);
      sortRef.current = { orderByColumn, isAsc };
      const values = searchForm.getFieldsValue() as {
        templateName?: string;
        templateCode?: string;
        templateType?: string;
        vendor?: string;
      };
      loadTemplates({ current, pageSize, ...values, orderByColumn, isAsc });
    },
    [sort, sorterMetas, searchForm, loadTemplates]
  );
  useEffect(() => {
    loadTemplates({ current: paginationProps.current, pageSize: paginationProps.pageSize });
    loadStatistics();
  }, [paginationProps.current, paginationProps.pageSize]);

  // 搜索
  const handleSearch = () => {
    const values = searchForm.getFieldsValue() as {
      templateName?: string;
      templateCode?: string;
      templateType?: string;
      vendor?: string;
    };
    loadTemplates({
      current: paginationProps.current,
      pageSize: paginationProps.pageSize,
      ...values,
      ...sortRef.current,
    });
  };

  // 重置
  const handleReset = () => {
    searchForm.resetFields();
    sort.resetSort();
    sortRef.current = {};
    loadTemplates({ current: paginationProps.current, pageSize: paginationProps.pageSize });
  };

  // 刷新
  const handleRefresh = () => {
    loadTemplates({
      current: paginationProps.current,
      pageSize: paginationProps.pageSize,
      ...sortRef.current,
    });
    loadStatistics();
  };

  // 操作成功后刷新
  const handleSuccess = () => {
    loadTemplates({
      current: paginationProps.current,
      pageSize: paginationProps.pageSize,
      ...sortRef.current,
    });
    loadStatistics();
  };

  const columns = getTemplateColumns({
    handlePreview,
    handleClone: (id: string) => handleClone(id, handleSuccess),
    handleDelete: (id: string) => handleDelete(id, handleSuccess),
    openModal: (record: ConfigTemplate) => openModal(record, editForm),
    getColumnSortOrder,
    sorterMetas,
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
              styles={{ content: { color: "var(--theme-warning, #faad14)" } }}
              prefix={<AppstoreOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="自定义模板"
              value={statistics.custom}
              styles={{ content: { color: "var(--theme-info, #1890ff)" } }}
              prefix={<SettingOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="初始化模板"
              value={statistics.init}
              styles={{ content: { color: "var(--theme-success, #52c41a)" } }}
              prefix={<FundOutlined />}
            />
          </Card>
        </Col>
      </Row>

      {/* 搜索表单和操作按钮 */}
      <Card style={{ marginBottom: 16 }}>
        <div
          style={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "flex-start",
            flexWrap: "wrap",
            gap: "16px",
          }}
        >
          <Form form={searchForm} layout="inline" style={{ flex: 1, minWidth: 0 }}>
            <Form.Item name="templateName" label="模板名称">
              <Input
                placeholder="请输入模板名称"
                allowClear
                className="user-form-input"
                style={{ width: 150 }}
              />
            </Form.Item>
            <Form.Item name="templateCode" label="模板编码">
              <Input
                placeholder="请输入模板编码"
                allowClear
                className="user-form-input"
                style={{ width: 150 }}
              />
            </Form.Item>
            <Form.Item name="templateType" label="模板类型">
              <Select
                placeholder="请选择模板类型"
                allowClear
                className="user-form-input"
                style={{ width: 130 }}
                onSearch={() => {}}
              >
                {TEMPLATE_TYPE_OPTIONS.map((opt) => (
                  <Option key={opt.value} value={opt.value}>
                    {opt.label}
                  </Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item name="vendor" label="厂商">
              <Select
                placeholder="请选择厂商"
                allowClear
                className="user-form-input"
                style={{ width: 120 }}
                onSearch={() => {}}
              >
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
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => openModal(undefined, editForm)}
            >
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
                Object.entries(searchForm.getFieldsValue() as Record<string, any>).filter(
                  ([, v]) => v !== undefined && v !== null && v !== ""
                )
              )}
              current={paginationProps?.current ?? 1}
              pageSize={paginationProps?.pageSize ?? 10}
            />
          </Space>
          {/* 批量导出 Modal */}

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
          onChange={handleTableChange}
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
