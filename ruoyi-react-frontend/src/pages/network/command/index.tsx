/**
 * Network Command Dispatch Page
 * 网络命令分发页面
 */

import { useEffect, useMemo, useState } from "react";
import type { FC } from "react";
import { Table, Button, Space, Form, Card, Row, Col, Statistic, App } from "antd";
// Phase 71 (W4)：裸 Select → 契约组件（children 形态改写 options）
import { AppSelect } from "@/components/ui";
import {
  ThunderboltOutlined,
  SearchOutlined,
  ReloadOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  ClockCircleOutlined,
  ApiOutlined,
} from "@ant-design/icons";
import type { ConfigExecution, ConfigExecutionDetail, NetworkDevice } from "@/types";
import { batchExport } from "@/lib/api/networkApi";
import { useCommandData, useCommandModals } from "./hooks";
import { getExecutionColumns } from "./columns";
import { CommandDispatchModal, CommandDetailDrawer } from "./modals";
import { STATUS_OPTIONS } from "./constants";
import { usePagination } from "@/hooks/usePagination";
import { useServerSort } from "@/hooks/useServerSort";
import { createSorterMeta } from "@/utils/tableHelpers";
import NetworkExport from "@/components/shared/NetworkExport";
import { BatchExportModal, ErrorAlertWithRetry, useActionPermission } from "@/components/shared";
import { PERMISSION_CODES } from "@/constants/permissionCodes";

const CommandDispatch: FC = () => {
  const { message } = App.useApp();

  // Phase 86 W3（RBAC-02）：操作级权限门控（三态 ready / loading / error）。
  // ⚠️ 判定必须走 useActionPermission —— 禁止用 permissions.length === 0 判「未就绪」。
  // 本页写码只有 networkCommand.execute（命令执行动作域）与 networkCommand.export（数据外带面）；
  // 行「取消」由 columns.tsx 的 `permission:` 传参承担判定。
  const permGate = useActionPermission();
  const permExecute = permGate.decide(PERMISSION_CODES.networkCommand.execute);
  // 导出类写入口：真实触发按钮在 `NetworkExport.tsx:120` 组件内部，
  // 页面上没有独立导出按钮 → 门控点落在 `<NetworkExport>` 元素本身。
  const permExport = permGate.decide(PERMISSION_CODES.networkCommand.export);
  const [execLoading, setExecLoading] = useState(false);
  const [dispatchModalVisible, setDispatchModalVisible] = useState(false);
  const [detailDrawerVisible, setDetailDrawerVisible] = useState(false);
  const [dispatchForm] = Form.useForm();
  const [searchForm] = Form.useForm();
  const [selectedRowKeys, setSelectedRowKeys] = useState<string[]>([]);
  const [batchModalVisible, setBatchModalVisible] = useState(false);
  const [batchExporting, setBatchExporting] = useState(false);
  const [devices, setDevices] = useState<NetworkDevice[]>([]);
  const [currentExecution, setCurrentExecution] = useState<ConfigExecution | null>(null);
  const [executionDetails, setExecutionDetails] = useState<ConfigExecutionDetail[]>([]);

  // 使用全局分页 hook
  const { paginationProps, setCurrent, setPageSize, setTotal } = usePagination();

  // 服务端排序:field 与 columns.dataIndex 对齐
  const sorterMetas = useMemo(
    () => [
      createSorterMeta<ConfigExecution>("executionName"),
      createSorterMeta<ConfigExecution>("status"),
      createSorterMeta<ConfigExecution>("createdAt", "date"),
    ],
    []
  );
  const {
    orderByColumn,
    isAsc,
    handleTableChange: handleCmdSortChange,
    sortOrder: cmdSortOrder,
  } = useServerSort<ConfigExecution>({
    sorterMetas,
  });

  const {
    executions,
    execTotal,
    statistics,
    error: commandError,
    devicesLoading,
    detailsLoading,
    setExecTotal: _setExecTotal,
    loadExecutions,
    loadStatistics,
    loadDevices,
    loadExecutionDetails,
  } = useCommandData(setExecLoading, {
    current: paginationProps.current ?? 1,
    pageSize: paginationProps.pageSize ?? 10,
  });

  // 同步 total 到全局分页状态
  useEffect(() => {
    setTotal(execTotal);
  }, [execTotal, setTotal]);

  const { handleQuickCommand, handleCancelExecution } = useCommandModals();

  useEffect(() => {
    Promise.all([loadExecutions(), loadStatistics()]);
    // eslint-disable-next-line react-hooks/exhaustive-deps -- intentional dependency for re-run on change
  }, [paginationProps.current, paginationProps.pageSize, loadExecutions, loadStatistics]);

  // 操作成功后刷新
  const handleSuccess = () => {
    loadExecutions();
    loadStatistics();
  };

  // 查看执行详情
  const handleViewDetail = async (record: ConfigExecution) => {
    // FIX-01: 先开抽屉再加载，加载期间内嵌表显示 loading 蒙层
    setCurrentExecution(record);
    setDetailDrawerVisible(true);
    const { details } = await loadExecutionDetails(record.id);
    setExecutionDetails(details as unknown as ConfigExecutionDetail[]);
  };

  // 打开命令分发模态框
  const openDispatchModal = async () => {
    // FIX-01: 先开弹窗再加载设备，加载期间内嵌表显示 loading 蒙层
    setDevices([]);
    setSelectedRowKeys([]);
    dispatchForm.resetFields();
    setDispatchModalVisible(true);
    const loadedDevices = await loadDevices();
    setDevices(loadedDevices);
  };

  const columns = useMemo(
    () =>
      getExecutionColumns({
        handleViewDetail,
        handleCancel: (id: string) => handleCancelExecution(id, handleSuccess),
        getSortOrder: (field) =>
          orderByColumn === field ? ((cmdSortOrder ?? null) as "ascend" | "descend" | null) : null,
      }),
    // eslint-disable-next-line react-hooks/exhaustive-deps -- handleCancelExecution from hook is stable enough
    [handleViewDetail, handleSuccess, orderByColumn, cmdSortOrder]
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

  return (
    <div>
      {/* FIX-04a: 列表加载失败可重试错误条 */}
      {!!commandError && (
        <ErrorAlertWithRetry
          error={commandError}
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
        <Col span={5}>
          <Card>
            <Statistic
              title="执行中"
              value={statistics.running}
              styles={{ content: { color: "var(--theme-info)" } }}
              prefix={<ApiOutlined />}
            />
          </Card>
        </Col>
        <Col span={4}>
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

      {/* 主要内容 */}
      <Card>
        <div style={{ marginBottom: 16 }}>
          <Form form={searchForm} layout="inline">
            <Form.Item label="状态">
              {/* 形态分类器①/②：children 写法改写 options；执行状态固定 4 项 → 显式禁搜 */}
              <AppSelect
                placeholder="请选择状态"
                className="user-form-input"
                style={{ width: 120 }}
                onChange={() => {
                  handleSuccess();
                }}
                onSearch={() => {}}
                showSearch={false}
                options={STATUS_OPTIONS}
              />
            </Form.Item>
            <Form.Item>
              <Space>
                <Button
                  aria-label="查询"
                  title="查询"
                  icon={<SearchOutlined />}
                  onClick={() => {
                    handleSuccess();
                  }}
                >
                  查询
                </Button>
                <Button
                  aria-label="刷新"
                  title="刷新"
                  icon={<ReloadOutlined />}
                  onClick={() => {
                    handleSuccess();
                  }}
                >
                  刷新
                </Button>
                {/* Phase 86 W3：快速命令 —— 条件渲染（networkCommand.execute） */}
                {permExecute.visible && (
                  <Button
                    type="primary"
                    icon={<ThunderboltOutlined />}
                    onClick={openDispatchModal}
                    aria-label="快速命令"
                    title={permExecute.hint ?? "快速命令"}
                    disabled={permExecute.disabled}
                  >
                    快速命令
                  </Button>
                )}
                {/* Phase 86 W3：批量导出 —— 门控点 = `<NetworkExport>` 组件元素本身
                    （「导出」按钮在 `NetworkExport.tsx:120` 组件内部，页面上无独立触发按钮）。
                    ready 无 networkCommand.export → 整个组件（含其内部「导出」按钮）不在 DOM。 */}
                {permExport.visible && (
                  <span title={permExport.hint} aria-disabled={permExport.disabled || undefined}>
                    <NetworkExport
                      entityType="command"
                      entityName="命令分发"
                      filters={Object.fromEntries(
                        Object.entries(
                          searchForm.getFieldsValue() as Record<string, unknown>
                        ).filter(([, v]) => v !== undefined && v !== null && v !== "")
                      )}
                      current={paginationProps?.current ?? 1}
                      pageSize={paginationProps?.pageSize ?? 10}
                    />
                  </span>
                )}
              </Space>
            </Form.Item>
          </Form>
          {/* 批量导出 Modal */}

          <BatchExportModal
            visible={batchModalVisible}

            onConfirm={handleBatchExport}

            onCancel={() => setBatchModalVisible(false)}

            loading={batchExporting}
          />
        </div>

        <Table
          columns={columns}
          dataSource={executions}
          loading={execLoading}
          rowKey="id"
          scroll={{ x: 1400 }}
          pagination={paginationProps}
          onChange={(pagination, _filters, sorter) => {
            handleCmdSortChange(pagination, _filters, sorter);
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

      {/* 命令分发模态框 */}
      <CommandDispatchModal
        open={dispatchModalVisible}
        devices={devices}
        loading={devicesLoading}
        selectedRowKeys={selectedRowKeys}
        onOk={async () => {
          await handleQuickCommand(selectedRowKeys, dispatchForm, () => {
            setDispatchModalVisible(false);
            handleSuccess();
          });
        }}
        onCancel={() => setDispatchModalVisible(false)}
        onSelectionChange={(keys) => setSelectedRowKeys(keys as string[])}
      />

      {/* 执行明细抽屉 */}
      <CommandDetailDrawer
        open={detailDrawerVisible}
        execution={currentExecution}
        details={executionDetails}
        loading={detailsLoading}
        onClose={() => setDetailDrawerVisible(false)}
      />
    </div>
  );
};

export default CommandDispatch;
