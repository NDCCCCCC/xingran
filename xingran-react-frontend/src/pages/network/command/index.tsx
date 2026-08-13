/**
 * Network Command Dispatch Page
 * 网络命令分发页面
 */

import { useEffect, useMemo, useState } from "react";
import type { FC } from "react";
import {
  Table,
  Button,
  Space,
  Form,
  Select,
  Card,
  Row,
  Col,
  Statistic,
  App,
} from "antd";
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
import { BatchExportModal } from "@/components/shared";

const { Option } = Select;

const CommandDispatch: FC = () => {
  const { message } = App.useApp();
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
  const { orderByColumn, isAsc, handleTableChange: handleCmdSortChange, sortOrder: cmdSortOrder } = useServerSort<ConfigExecution>({
    sorterMetas,
  });

  const {
    executions,
    execTotal,
    statistics,
    setExecTotal,
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
  }, [paginationProps.current, paginationProps.pageSize, loadExecutions, loadStatistics]);

  // 操作成功后刷新
  const handleSuccess = () => {
    loadExecutions();
    loadStatistics();
  };

  // 查看执行详情
  const handleViewDetail = async (record: ConfigExecution) => {
    const { execution, details } = await loadExecutionDetails(record.id);
    setCurrentExecution(execution);
    setExecutionDetails(details as unknown as ConfigExecutionDetail[]);
    setDetailDrawerVisible(true);
  };

  // 打开命令分发模态框
  const openDispatchModal = async () => {
    const loadedDevices = await loadDevices();
    setDevices(loadedDevices);
    setSelectedRowKeys([]);
    dispatchForm.resetFields();
    setDispatchModalVisible(true);
  };

  const columns = useMemo(
    () =>
      getExecutionColumns({
        handleViewDetail,
        handleCancel: (id: string) => handleCancelExecution(id, handleSuccess),
        getSortOrder: (field) => (orderByColumn === field ? (cmdSortOrder ?? null) as "ascend" | "descend" | null : null),
      }),
    [handleViewDetail, handleSuccess, orderByColumn, cmdSortOrder]
  );

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
              styles={{ content: { color: "var(--theme-text-tertiary, #999)" } }}
              prefix={<ClockCircleOutlined />}
            />
          </Card>
        </Col>
        <Col span={5}>
          <Card>
            <Statistic
              title="执行中"
              value={statistics.running}
              styles={{ content: { color: "var(--theme-info, #1890ff)" } }}
              prefix={<ApiOutlined />}
            />
          </Card>
        </Col>
        <Col span={4}>
          <Card>
            <Statistic
              title="成功"
              value={statistics.success}
              styles={{ content: { color: "var(--theme-success, #52c41a)" } }}
              prefix={<CheckCircleOutlined />}
            />
          </Card>
        </Col>
        <Col span={5}>
          <Card>
            <Statistic
              title="失败"
              value={statistics.failed}
              styles={{ content: { color: "var(--theme-error, #ff4d4f)" } }}
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
              <Select
                placeholder="请选择状态"
                allowClear
                className="user-form-input"
                style={{ width: 120 }}
                onChange={() =>    {
                  handleSuccess();
                }}
               onSearch={() => {}}>
                {STATUS_OPTIONS.map((opt) => (
                  <Option key={opt.value} value={opt.value}>
                    {opt.label}
                  </Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item>
              <Space>
                <Button
                  icon={<SearchOutlined />}
                  onClick={() => {
                    handleSuccess();
                  }}
                >
                  查询
                </Button>
                <Button
                  icon={<ReloadOutlined />}
                  onClick={() => {
                    handleSuccess();
                  }}
                >
                  刷新
                </Button>
                <Button
                  type="primary"
                  icon={<ThunderboltOutlined />}
                  onClick={openDispatchModal}
                >
                  快速命令
                </Button>
                <NetworkExport
                  entityType="command"
                  entityName="命令分发"
                  filters={Object.fromEntries(
                    Object.entries(searchForm.getFieldsValue() as Record<string, unknown>).filter(([, v]) => v !== undefined && v !== null && v !== "")
                  )}
                  current={paginationProps?.current ?? 1}
                  pageSize={paginationProps?.pageSize ?? 10}
                />
              </Space>
            </Form.Item>
          </Form>{/* 批量导出 Modal */}

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
            Object.keys(formValues).forEach(key => {
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
        onClose={() => setDetailDrawerVisible(false)}
      />
    </div>
  );
};

export default CommandDispatch;
