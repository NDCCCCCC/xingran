/**
 * Network Template Management Page
 * 网络模板管理页面
 */

import { useEffect, useState, useMemo, useCallback, useRef } from "react";
import type { FC } from "react";
import { Table, Button, Space, Form, Card, Row, Col, Statistic, App } from "antd";
import type { FilterValue } from "antd/es/table/interface";
// Phase 71 (W4)：裸 Input/Select → 契约组件（children 形态改写 options）
import { AppInput, AppSelect } from "@/components/ui";
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
import { BatchExportModal, ErrorAlertWithRetry, useActionPermission } from "@/components/shared";
import { PERMISSION_CODES } from "@/constants/permissionCodes";

const TemplateManagement: FC = () => {
  const { message } = App.useApp();

  // Phase 86 W3（RBAC-02）：操作级权限门控（三态 ready / loading / error）。
  // ⚠️ 判定必须走 useActionPermission —— 禁止用 permissions.length === 0 判「未就绪」。
  const permGate = useActionPermission();
  const permAdd = permGate.decide(PERMISSION_CODES.networkTemplate.add);
  const permDelete = permGate.decide(PERMISSION_CODES.networkTemplate.delete);
  // 导出类写入口（数据外带面）：真实触发按钮在 `NetworkExport.tsx:120` 组件内部，
  // 页面上没有独立导出按钮 → 门控点落在 `<NetworkExport>` 元素本身。
  const permExport = permGate.decide(PERMISSION_CODES.networkTemplate.export);

  const [searchForm] = Form.useForm();
  const [editForm] = Form.useForm();
  const [batchModalVisible, setBatchModalVisible] = useState(false);
  const [batchExporting, setBatchExporting] = useState(false);

  // 使用全局分页 hook
  const { paginationProps, setTotal } = usePagination();

  const {
    templates,
    loading,
    total: _total,
    statistics,
    error: templateError,
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
    handleGetVariables: _handleGetVariables,
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
      _filters: Record<string, FilterValue | null>,
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
    // eslint-disable-next-line react-hooks/exhaustive-deps -- load/fetch fn from hook is stable enough; disable to avoid loop
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
    } catch (error: unknown) {
      message.error(`批量导出失败：${(error as Error).message}`);
    } finally {
      setBatchExporting(false);
    }
  };

  return (
    <div>
      {/* FIX-04a: 列表加载失败可重试错误条 */}
      {!!templateError && (
        <ErrorAlertWithRetry
          error={templateError}
          onRetry={() => Promise.all([loadTemplates(), loadStatistics()])}
        />
      )}
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
              styles={{ content: { color: "var(--theme-warning)" } }}
              prefix={<AppstoreOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="自定义模板"
              value={statistics.custom}
              styles={{ content: { color: "var(--theme-info)" } }}
              prefix={<SettingOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="初始化模板"
              value={statistics.init}
              styles={{ content: { color: "var(--theme-success)" } }}
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
              <AppInput
                placeholder="请输入模板名称"
                className="user-form-input"
                style={{ width: 150 }}
              />
            </Form.Item>
            <Form.Item name="templateCode" label="模板编码">
              <AppInput
                placeholder="请输入模板编码"
                className="user-form-input"
                style={{ width: 150 }}
              />
            </Form.Item>
            <Form.Item name="templateType" label="模板类型">
              {/* 形态分类器①/②：children 写法改写 options；固定 3 项 → 显式禁搜 */}
              <AppSelect
                placeholder="请选择模板类型"
                className="user-form-input"
                style={{ width: 130 }}
                onSearch={() => {}}
                showSearch={false}
                options={TEMPLATE_TYPE_OPTIONS}
              />
            </Form.Item>
            <Form.Item name="vendor" label="厂商">
              {/* 形态分类器①/②：children 写法改写 options；厂商固定 5 项 → 显式禁搜 */}
              <AppSelect
                placeholder="请选择厂商"
                className="user-form-input"
                style={{ width: 120 }}
                onSearch={() => {}}
                showSearch={false}
                options={VENDOR_OPTIONS}
              />
            </Form.Item>
            <Form.Item>
              <Space>
                <Button
                  type="primary"
                  icon={<SearchOutlined />}
                  onClick={handleSearch}
                  aria-label="搜索"
                  title="搜索"
                >
                  查询
                </Button>
                <Button
                  icon={<ReloadOutlined />}
                  onClick={handleReset}
                  aria-label="重置"
                  title="重置"
                >
                  重置
                </Button>
                <Button
                  icon={<ReloadOutlined />}
                  onClick={handleRefresh}
                  aria-label="刷新"
                  title="刷新"
                >
                  刷新
                </Button>
              </Space>
            </Form.Item>
          </Form>
          <Space>
            {/* Phase 86 W3：新增入口 —— ready 无码不渲染；未就绪渲染但 disabled + 提示 */}
            {permAdd.visible && (
              <Button
                aria-label="新增模板"
                title={permAdd.hint ?? "新增模板"}
                type="primary"
                icon={<PlusOutlined />}
                onClick={() => openModal(undefined, editForm)}
                disabled={permAdd.disabled}
              >
                新增
              </Button>
            )}
            {/* Phase 86 W3：批量删除 —— 条件渲染（networkTemplate.delete） */}
            {permDelete.visible && (
              <Button
                aria-label="批量删除"
                title={permDelete.hint ?? "批量删除"}
                danger
                icon={<DeleteOutlined />}
                onClick={() => handleBatchDelete(selectedRowKeys, handleSuccess)}
                disabled={permDelete.disabled || selectedRowKeys.length === 0}
              >
                批量删除
              </Button>
            )}
            {/* Phase 86 W3：批量导出 —— 门控点 = `<NetworkExport>` 组件元素本身
                （「导出」按钮在 `NetworkExport.tsx:120` 组件内部，页面上无独立触发按钮）。
                ready 无 networkTemplate.export → 整个组件（含其内部「导出」按钮）不在 DOM。 */}
            {permExport.visible && (
              <span title={permExport.hint} aria-disabled={permExport.disabled || undefined}>
                <NetworkExport
                  entityType="templates"
                  entityName="配置模板"
                  filters={Object.fromEntries(
                    Object.entries(searchForm.getFieldsValue() as Record<string, unknown>).filter(
                      ([, v]) => v !== undefined && v !== null && v !== ""
                    )
                  )}
                  current={paginationProps?.current ?? 1}
                  pageSize={paginationProps?.pageSize ?? 10}
                />
              </span>
            )}
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
