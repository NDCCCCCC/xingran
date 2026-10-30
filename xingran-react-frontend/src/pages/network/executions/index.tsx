import { useState, useEffect, useMemo, useCallback } from "react";
import type { FC } from "react";
import { batchExport } from "@/lib/api/networkApi";
import { Table, Button, Space, Form, Card, Row, Col, Statistic, App } from "antd";
// Phase 71 (W4)：裸 Select → 契约组件（children 形态改写 options）
import { AppSelect } from "@/components/ui";
import {
  PlayCircleOutlined,
  SearchOutlined,
  ReloadOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  ClockCircleOutlined,
  ApiOutlined,
} from "@ant-design/icons";
import { useExecutionData, useExecutionModals } from "./hooks";
import { getExecutionColumns } from "./columns";
import { ConfigExecuteModal, VariableModal, DetailDrawer } from "./modals";
import { STATUS_OPTIONS } from "./constants";
import { usePagination } from "@/hooks/usePagination";
import { useServerSort } from "@/hooks/useServerSort";
import { createSorterMeta } from "@/utils/tableHelpers";
import NetworkExport from "@/components/shared/NetworkExport";
import { BatchExportModal, ErrorAlertWithRetry, useActionPermission } from "@/components/shared";
import { PERMISSION_CODES } from "@/constants/permissionCodes";
import type { ConfigExecution } from "@/types";

const ConfigExecutionPage: FC = () => {
  const { message } = App.useApp();

  // Phase 86 W3（RBAC-02）：操作级权限门控（三态 ready / loading / error）。
  // ⚠️ 判定必须走 useActionPermission —— 禁止用 permissions.length === 0 判「未就绪」。
  // 本页写码：执行配置模板 / 取消执行 → `networkCommand.execute`（复用）；
  // 导出 → `networkCommand.export`。执行记录域无独立码，本 phase 零新增码。
  const permGate = useActionPermission();
  const permExecute = permGate.decide(PERMISSION_CODES.networkCommand.execute);
  // 导出类写入口：真实触发按钮在 `NetworkExport.tsx:120` 组件内部，
  // 页面上没有独立导出按钮 → 门控点落在 `<NetworkExport>` 元素本身。
  const permExport = permGate.decide(PERMISSION_CODES.networkCommand.export);
  const [executeForm] = Form.useForm();
  const [searchForm] = Form.useForm();
  const [batchModalVisible, setBatchModalVisible] = useState(false);
  const [batchExporting, setBatchExporting] = useState(false);

  // 使用全局分页 hook
  const { paginationProps, setCurrent, setPageSize } = usePagination();

  // 服务端排序:field 与 columns.dataIndex 对齐
  const sorterMetas = useMemo(
    () => [
      createSorterMeta<ConfigExecution>("executionName"),
      createSorterMeta<ConfigExecution>("templateName"),
      createSorterMeta<ConfigExecution>("status"),
      createSorterMeta<ConfigExecution>("createdAt", "date"),
    ],
    []
  );
  const {
    orderByColumn,
    isAsc,
    handleTableChange: handleNetExecSortChange,
    sortOrder: netExecSortOrder,
  } = useServerSort<ConfigExecution>({
    sorterMetas,
  });

  // 数据管理
  const {
    dataState,
    setDataState,
    execLoading,
    error: executionError,
    devicesLoading,
    detailsLoading,
    statistics,
    loadDevices,
    loadTemplates,
    loadExecutions,
    loadExecutionDetails,
    loadStatistics,
  } = useExecutionData({
    current: 1, // 未使用，保留默认值
    pageSize: paginationProps.pageSize ?? 10,
    execCurrent: paginationProps.current ?? 1,
  });

  // 模态框和操作管理
  const {
    modalState,
    selectedRowKeys,
    setSelectedRowKeys,
    openExecuteModal: openExecuteModalBase,
    handleTemplateChange,
    handleExecuteByTemplate,
    handleCancelExecution,
    handleViewDetail: handleViewDetailBase,
    handleViewOutput,
    closeDetailDrawer,
    closeVariableModal,
    closeExecuteModal,
  } = useExecutionModals({
    dataState,
    setDataState: () => {}, // Not needed here
    loadExecutions,
  });

  // 打开执行模态框时加载设备和模板
  // FIX-01: 先开弹窗再加载，加载期间内嵌表显示 loading 蒙层
  const openExecuteModal = async () => {
    openExecuteModalBase();
    await Promise.all([loadDevices(), loadTemplates()]);
  };

  // 查看详情时加载执行详情
  // FIX-01: 先开抽屉再加载，加载期间内嵌表显示 loading 蒙层
  const handleViewDetail = useCallback(
    async (record: ConfigExecution) => {
      // 先用列表行数据填充 currentExecution，抽屉立即可见，明细表 loading
      setDataState((prev) => ({ ...prev, currentExecution: record, executionDetails: [] }));
      handleViewDetailBase(record);
      await loadExecutionDetails(record.id);
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps -- handleViewDetailBase is a stable modal handler
    [loadExecutionDetails]
  );

  // 表格列
  const executionColumns = useMemo(
    () =>
      getExecutionColumns({
        handleViewDetail,
        handleCancelExecution,
        getSortOrder: (field) =>
          orderByColumn === field
            ? ((netExecSortOrder ?? null) as "ascend" | "descend" | null)
            : null,
      }),
    [handleViewDetail, handleCancelExecution, orderByColumn, netExecSortOrder]
  );

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

  useEffect(() => {
    Promise.all([loadExecutions(), loadStatistics()]);
    // eslint-disable-next-line react-hooks/exhaustive-deps -- paginationProps.current is intentional
  }, [paginationProps.current, paginationProps.pageSize, loadExecutions, loadStatistics]);

  return (
    <div>
      {/* FIX-04a: 列表加载失败可重试错误条 */}
      {!!executionError && (
        <ErrorAlertWithRetry
          error={executionError}
          onRetry={() => Promise.all([loadExecutions(), loadStatistics()])}
        />
      )}
      {/* 统计卡片 */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={5}>
          <Card>
            <Statistic title="执行总数" value={statistics.total} prefix={<ApiOutlined />} />
          </Card>
        </Col>
        <Col span={5}>
          <Card>
            <Statistic
              title="待执行"
              value={statistics.pending}
              styles={{ content: { color: "var(--theme-text-tertiary)" } }}
              prefix={<ClockCircleOutlined />}
            />
          </Card>
        </Col>
        <Col span={4}>
          <Card>
            <Statistic
              title="执行中"
              value={statistics.running}
              styles={{ content: { color: "var(--theme-info)" } }}
              prefix={<ApiOutlined />}
            />
          </Card>
        </Col>
        <Col span={5}>
          <Card>
            <Statistic
              title="成功"
              value={statistics.success}
              styles={{ content: { color: "var(--theme-success)" } }}
              prefix={<CheckCircleOutlined />}
            />
          </Card>
        </Col>
        <Col span={5}>
          <Card>
            <Statistic
              title="失败"
              value={statistics.failed}
              styles={{ content: { color: "var(--theme-error)" } }}
              prefix={<CloseCircleOutlined />}
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
            <Form.Item label="状态">
              {/* 形态分类器①/②：children 写法改写 options；状态固定枚举 → 显式禁搜 */}
              <AppSelect
                placeholder="请选择状态"
                className="user-form-input"
                style={{ width: 120 }}
                onChange={() => {
                  loadExecutions();
                  loadStatistics();
                }}
                onSearch={() => {}}
                showSearch={false}
                options={[...STATUS_OPTIONS]}
              />
            </Form.Item>
            <Form.Item>
              <Space>
                <Button
                  aria-label="查询"
                  title="查询"
                  icon={<SearchOutlined />}
                  onClick={() => {
                    loadExecutions();
                    loadStatistics();
                  }}
                >
                  查询
                </Button>
                <Button
                  aria-label="刷新"
                  title="刷新"
                  icon={<ReloadOutlined />}
                  onClick={() => {
                    loadExecutions();
                    loadStatistics();
                  }}
                >
                  刷新
                </Button>
              </Space>
            </Form.Item>
          </Form>
          <Space>
            {/* Phase 86 W3：执行配置模板 —— 条件渲染（networkCommand.execute） */}
            {permExecute.visible && (
              <Button
                type="primary"
                icon={<PlayCircleOutlined />}
                onClick={openExecuteModal}
                aria-label="执行模板"
                title={permExecute.hint ?? "执行模板"}
                disabled={permExecute.disabled}
              >
                执行配置模板
              </Button>
            )}
            {/* Phase 86 W3：批量导出 —— 门控点 = `<NetworkExport>` 组件元素本身
                （「导出」按钮在 `NetworkExport.tsx:120` 组件内部，页面上无独立触发按钮）。
                ready 无 networkCommand.export → 整个组件（含其内部「导出」按钮）不在 DOM。 */}
            {permExport.visible && (
              <span title={permExport.hint} aria-disabled={permExport.disabled || undefined}>
                <NetworkExport
                  entityType="executions"
                  entityName="配置执行"
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

      {/* 执行记录表格 */}
      <Card>
        <Table
          columns={executionColumns}
          dataSource={dataState.executions}
          loading={execLoading}
          rowKey="id"
          scroll={{ x: 1400 }}
          pagination={paginationProps}
          onChange={(pagination, _filters, sorter) => {
            handleNetExecSortChange(pagination, _filters, sorter);
            setCurrent(pagination.current ?? 1);
            setPageSize(pagination.pageSize ?? 10);
            const formValues = searchForm.getFieldsValue() as Record<string, unknown>;
            const searchParams: Record<string, unknown> = {
              current: pagination.current ?? 1,
              pageSize: pagination.pageSize ?? 10,
              ...(orderByColumn ? { orderByColumn, isAsc } : {}),
            };
            Object.keys(formValues).forEach((key) => {
              const value = formValues[key];
              if (value !== undefined && value !== null && value !== "") {
                searchParams[key] = value;
              }
            });
            loadExecutions(searchParams);
          }}
        />
      </Card>

      {/* 执行配置模态框 */}
      <ConfigExecuteModal
        open={modalState.executeModalVisible}
        devices={dataState.devices}
        templates={dataState.templates}
        selectedTemplate={dataState.selectedTemplate}
        selectedRowKeys={selectedRowKeys}
        loading={devicesLoading}
        onOk={async () => {
          await handleExecuteByTemplate(executeForm);
        }}
        onCancel={() => {
          closeExecuteModal();
          executeForm.resetFields();
        }}
        onTemplateChange={handleTemplateChange}
        onSelectedRowKeysChange={setSelectedRowKeys}
      />

      {/* 变量输入模态框 */}
      <VariableModal
        open={modalState.variableModalVisible}
        selectedTemplate={dataState.selectedTemplate}
        form={executeForm}
        onOk={closeVariableModal}
        onCancel={closeVariableModal}
      />

      {/* 执行明细抽屉 */}
      <DetailDrawer
        open={modalState.detailDrawerVisible}
        currentExecution={dataState.currentExecution}
        executionDetails={dataState.executionDetails}
        loading={detailsLoading}
        onClose={closeDetailDrawer}
        handleViewOutput={handleViewOutput}
      />
    </div>
  );
};

export default ConfigExecutionPage;
