import { useState, useEffect, useMemo } from "react";
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
  SearchOutlined,
  ReloadOutlined,
  PlusOutlined,
  CloudServerOutlined,
  ClockCircleOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
} from "@ant-design/icons";
import { useDiscoveryData, useDiscoveryPolling } from "./hooks";
import { getDiscoveryColumns } from "./columns";
import { CreateModal, ResultModal } from "./modals";
import { STATUS_OPTIONS } from "./constants";
import { parseIPRanges } from "./utils";
import { post } from "@/lib/api";
import { batchExport } from "@/lib/api/networkApi";
import type { FormInstance } from "antd/es/form";
import NetworkExport from "@/components/shared/NetworkExport";
import { BatchExportModal } from "@/components/shared";
import { DownloadOutlined } from "@ant-design/icons";
import { usePagination } from "@/hooks/usePagination";
import { useServerSort } from "@/hooks/useServerSort";
import { createSorterMeta } from "@/utils/tableHelpers";
import { isFormValidationError } from "@/utils/errorHandler";
import type { DeviceDiscovery, NetworkDevice } from "@/types";

const { Option } = Select;

const DeviceDiscoveryPage: FC = () => {
  const { message } = App.useApp();
  // 使用全局分页 hook
  const { paginationProps, setTotal } = usePagination();

  // 服务端排序:field 与 columns.dataIndex 对齐
  const sorterMetas = useMemo(
    () => [
      createSorterMeta<DeviceDiscovery>("taskName"),
      createSorterMeta<DeviceDiscovery>("discoveryType"),
      createSorterMeta<DeviceDiscovery>("status"),
      createSorterMeta<DeviceDiscovery>("startedAt"),
      createSorterMeta<DeviceDiscovery>("completedAt"),
    ],
    []
  );
  const { orderByColumn, isAsc, handleTableChange: handleDiscoverySortChange, sortOrder: discoverySortOrder } = useServerSort<DeviceDiscovery>({
    sorterMetas,
  });

  const [form] = Form.useForm();
  const [batchModalVisible, setBatchModalVisible] = useState(false);
  const [batchExporting, setBatchExporting] = useState(false);

  // 数据管理
  const {
    discoveries,
    discoveredDevices,
    departments,
    loading,
    total,
    statistics,
    modalState,
    currentDiscovery,
    setModalState,
    setCurrentDiscovery,
    loadDiscoveries,
    loadStatistics,
    loadDepartments,
    loadDiscoveryResults,
  } = useDiscoveryData({
    current: paginationProps.current ?? 1,
    pageSize: paginationProps.pageSize ?? 10,
  });

  // 轮询管理
  useDiscoveryPolling({
    discoveries,
    onPoll: loadDiscoveries,
  });

  // 打开创建模态框
  const openModal = async () => {
    await loadDepartments();
    form.resetFields();
    form.setFieldsValue({ snmpPort: 161, discoveryType: "snmp" });
    setModalState(prev => ({ ...prev, modalVisible: true }));
  };

  // 创建发现任务
  const handleCreate = async (form: FormInstance<unknown>) => {
    try {
      const formValues = await form.validateFields();
      const values = formValues as {
        ipRanges?: string;
        taskName: string;
        discoveryType: string;
        snmpCommunity?: string;
        snmpPort?: number;
        groupId?: string;
        autoImport?: boolean;
      };
      const ipRangesText = values.ipRanges || "";
      const ipRanges = parseIPRanges(ipRangesText);

      if (ipRanges.length === 0) {
        message.error("请输入有效的IP范围");
        return;
      }

      const requestData = {
        taskName: values.taskName,
        discoveryType: values.discoveryType,
        ipRanges: ipRanges,
        snmpCommunity: values.snmpCommunity || "",
        snmpPort: values.snmpPort || 161,
        groupId: values.groupId || null,
        autoImport: values.autoImport || false,
      };

      await post("/network/discoveries/create", requestData);
      message.success("发现任务已创建");
      setModalState(prev => ({ ...prev, modalVisible: false }));
      form.resetFields();
      loadDiscoveries();
      loadStatistics();
    } catch (error: unknown) {
      if (isFormValidationError(error)) {
        return;
      }
      console.error("创建任务失败:", error);
      message.error("创建任务失败");
    }
  };

  // 删除任务
  const handleDelete = async (id: string) => {
    try {
      await post(`/network/discoveries/${id}/delete`, {});
      message.success("删除成功");
      loadDiscoveries();
    } catch (error) {
      message.error("删除失败");
    }
  };

  // 查看发现结果
  const handleViewResult = async (record: DeviceDiscovery) => {
    setCurrentDiscovery(record);
    await loadDiscoveryResults((record as DeviceDiscovery).id);
    setModalState(prev => ({ ...prev, resultModalVisible: true }));
  };

  // 导入发现的设备
  const handleImport = async () => {
    if (!currentDiscovery) return;
    try {
      await post(`/network/discoveries/${currentDiscovery.id}/import`, {});
      message.success("导入成功");
      setModalState(prev => ({ ...prev, resultModalVisible: false }));
      loadDiscoveries();
    } catch (error) {
      message.error("导入失败");
    }
  };

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

  // 表格列
  const discoveryColumns = useMemo(
    () =>
      getDiscoveryColumns({
        handleViewResult,
        handleDelete,
        getSortOrder: (field) => (orderByColumn === field ? (discoverySortOrder ?? null) as "ascend" | "descend" | null : null),
      }),
    [handleViewResult, handleDelete, orderByColumn, discoverySortOrder]
  );

  useEffect(() => {
    loadDiscoveries();
    loadStatistics();
  }, [paginationProps.current, paginationProps.pageSize, loadDiscoveries, loadStatistics]);

  return (
    <div>
      {/* 统计卡片 */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={4}>
          <Card>
            <Statistic
              title="任务总数"
              value={statistics.total}
              prefix={<CloudServerOutlined />}
            />
          </Card>
        </Col>
        <Col span={4}>
          <Card>
            <Statistic
              title="待执行"
              value={statistics.pending}
              styles={{ content: { color: "var(--theme-text-tertiary, #999)" } }}
              prefix={<ClockCircleOutlined />}
            />
          </Card>
        </Col>
        <Col span={4}>
          <Card>
            <Statistic
              title="扫描中"
              value={statistics.running}
              styles={{ content: { color: "var(--theme-info, #1890ff)" } }}
              prefix={<CloudServerOutlined />}
            />
          </Card>
        </Col>
        <Col span={4}>
          <Card>
            <Statistic
              title="已完成"
              value={statistics.completed}
              styles={{ content: { color: "var(--theme-success, #52c41a)" } }}
              prefix={<CheckCircleOutlined />}
            />
          </Card>
        </Col>
        <Col span={4}>
          <Card>
            <Statistic
              title="失败"
              value={statistics.failed}
              styles={{ content: { color: "var(--theme-error, #ff4d4f)" } }}
              prefix={<CloseCircleOutlined />}
            />
          </Card>
        </Col>
        <Col span={4}>
          <Card>
            <Statistic
              title="发现设备"
              value={statistics.totalDevices}
            />
          </Card>
        </Col>
      </Row>

      {/* 搜索表单和操作按钮 */}
      <Card style={{ marginBottom: 16 }}>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start", flexWrap: "wrap", gap: "16px" }}>
          <Form layout="inline" style={{ flex: 1, minWidth: 0 }}>
            <Form.Item label="状态">
              <Select
                placeholder="请选择状态"
                allowClear
                className="user-form-input"
                style={{ width: 120 }}
                onChange={() =>    {
                  loadDiscoveries();
                  loadStatistics();
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
                    loadDiscoveries();
                    loadStatistics();
                  }}
                >
                  查询
                </Button>
                <Button
                  icon={<ReloadOutlined />}
                  onClick={() => {
                    loadDiscoveries();
                    loadStatistics();
                  }}
                >
                  刷新
                </Button>
              </Space>
            </Form.Item>
          </Form>
          <Space>
            <NetworkExport
              entityType="discoveries"
              entityName="设备发现"
              filters={{}}
              current={paginationProps.current}
              pageSize={paginationProps.pageSize}
            />
            <Button type="primary" icon={<PlusOutlined />} onClick={openModal}>
              创建发现任务
            </Button>
          </Space>{/* 批量导出 Modal */}

        <BatchExportModal

          visible={batchModalVisible}

          onConfirm={handleBatchExport}

          onCancel={() => setBatchModalVisible(false)}

          loading={batchExporting}

        />


        </div>
      </Card>

      {/* 发现任务表格 */}
      <Card>
        <Table
          columns={discoveryColumns}
          dataSource={discoveries}
          loading={loading}
          rowKey="id"
          scroll={{ x: 1500 }}
          pagination={paginationProps}
          onChange={(pagination, _filters, sorter) => {
            handleDiscoverySortChange(pagination, _filters, sorter);
            loadDiscoveries({
              current: pagination.current,
              pageSize: pagination.pageSize,
              ...(orderByColumn ? { orderByColumn, isAsc } : {}),
            });
          }}
        />
      </Card>

      {/* 创建发现任务模态框 */}
      <CreateModal
        open={modalState.modalVisible}
        departments={departments}
        onOk={handleCreate}
        onCancel={() => setModalState(prev => ({ ...prev, modalVisible: false }))}
      />

      {/* 发现结果模态框 */}
      <ResultModal
        open={modalState.resultModalVisible}
        currentDiscovery={currentDiscovery}
        discoveredDevices={discoveredDevices}
        onImport={handleImport}
        onClose={() => setModalState(prev => ({ ...prev, resultModalVisible: false }))}
      />
    </div>
  );
};

export default DeviceDiscoveryPage;
