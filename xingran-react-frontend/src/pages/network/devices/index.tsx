/**
 * NetworkDevice 设备管理页面
 */

import { useState, useEffect, useCallback, useMemo, type FC } from "react";
import { useLocation } from "react-router-dom";
import { usePersistedStateController } from "@/hooks/usePersistedState";
import {
  Table,
  Button,
  Space,
  Modal,
  Form,
  Input,
  InputNumber,
  Select,
  Tag,
  Card,
  Row,
  Col,
  Statistic,
  Layout,
  Divider,
  Alert,
  App,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import type { DividerProps } from "antd/es/divider";
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  SearchOutlined,
  ReloadOutlined,
  CloudServerOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  WarningOutlined,
  ThunderboltOutlined,
  ApiOutlined,
  EyeOutlined,
  CloudUploadOutlined,
  HistoryOutlined,
} from "@ant-design/icons";
import type { NetworkDevice } from "@/types";
import type { PageResponse } from "@/types";
import { post } from "@/lib/api";
import { batchExport } from "@/lib/api/networkApi";
import DeptTree from "@/components/DeptTree";
import { useNavigate } from "react-router-dom";
import { useTableManager } from "@/hooks/useTableManager";
import { handleApiError, handleSuccess } from "@/utils/errorHandler";
import ActionButtons from "@/components/shared/ActionButtons";
import { DepartmentTreeSelect } from "@/components/shared";
import NetworkExport from "@/components/shared/NetworkExport";
import { BatchExportModal } from "@/components/shared";
import { usePagination } from "@/hooks/usePagination";
import { useMenuStore } from "@/store/menuStore";
import { createSorterMeta } from "@/utils/tableHelpers";
import type { SortOrder } from "@/hooks/useServerSort";

// 导入提取的常量、工具和 Hook
import {
  VENDOR_OPTIONS,
  DEVICE_TYPE_OPTIONS,
  STATUS_OPTIONS,
  DEVICE_TYPE_TAG_COLOR,
  VENDOR_TAG_COLOR,
} from "./constants";
import {
  formatDateTime,
  getOptionLabel,
  getStatusColor,
} from "./utils";
import { useDeviceData } from "./hooks/useDeviceData";
import { useDeviceModals, type ProbeResult } from "./hooks/useDeviceModals";

const { Option } = Select;
const { TextArea } = Input;
const { Sider, Content } = Layout;

// ==================== 表格列定义 ====================

interface DeviceTableColumnsProps {
  navigate: (path: string) => void;
  handleCollectPorts: (device: NetworkDevice) => void;
  openDetailModal: (device: NetworkDevice) => void;
  openModal: (record?: NetworkDevice) => void;
  handleDelete: (id: string) => void;
  collectingDeviceId: string | null;
  canViewMACHistory: boolean;
  getColumnSortOrder: (field: string) => SortOrder | undefined;
}

function getDeviceTableColumns(props: DeviceTableColumnsProps): ColumnsType<NetworkDevice> {
  const { navigate, handleCollectPorts, openDetailModal, openModal, handleDelete, collectingDeviceId, canViewMACHistory, getColumnSortOrder } = props;

  return [
    {
      title: "设备名称",
      dataIndex: "deviceName",
      key: "deviceName",
      width: 200,
      sorter: true,
      sortOrder: getColumnSortOrder("deviceName"),
      render: (deviceName: string, record: NetworkDevice) => (
        <a
          onClick={() => navigate(`/network/ports?deviceId=${record.id}`)}
          style={{ color: "var(--theme-info, #1890ff)", cursor: "pointer" }}
        >
          {deviceName}
        </a>
      ),
    },
    {
      title: "设备类型",
      dataIndex: "deviceType",
      key: "deviceType",
      width: 120,
      sorter: true,
      sortOrder: getColumnSortOrder("deviceType"),
      render: (deviceType: string) => (
        <Tag color={DEVICE_TYPE_TAG_COLOR}>{getOptionLabel(DEVICE_TYPE_OPTIONS, deviceType)}</Tag>
      ),
    },
    {
      title: "厂商",
      dataIndex: "vendor",
      key: "vendor",
      width: 100,
      sorter: true,
      sortOrder: getColumnSortOrder("vendor"),
      render: (vendor: string) => (
        <Tag color={VENDOR_TAG_COLOR}>{getOptionLabel(VENDOR_OPTIONS, vendor)}</Tag>
      ),
    },
    {
      title: "型号",
      dataIndex: "model",
      key: "model",
      width: 150,
      sorter: true,
      sortOrder: getColumnSortOrder("model"),
    },
    {
      title: "IP地址",
      dataIndex: "ipAddress",
      key: "ipAddress",
      width: 140,
      sorter: true,
      sortOrder: getColumnSortOrder("ipAddress"),
    },
    {
      title: "端口",
      dataIndex: "port",
      key: "port",
      width: 80,
      sorter: true,
      sortOrder: getColumnSortOrder("port"),
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 90,
      sorter: true,
      sortOrder: getColumnSortOrder("status"),
      render: (status: number) => (
        <Tag color={getStatusColor(status)}>{getOptionLabel(STATUS_OPTIONS, status)}</Tag>
      ),
    },
    { title: "部门", dataIndex: "deptName", key: "deptName", width: 180 },
    {
      title: "最后连接",
      dataIndex: "lastSeenAt",
      key: "lastSeenAt",
      width: 180,
      sorter: true,
      sortOrder: getColumnSortOrder("lastSeenAt"),
      render: (lastSeenAt: string) => formatDateTime(lastSeenAt),
    },
    {
      title: "创建时间",
      dataIndex: "createdAt",
      key: "createdAt",
      width: 180,
      sorter: true,
      sortOrder: getColumnSortOrder("createdAt"),
      render: (createdAt: string) => formatDateTime(createdAt),
    },
    {
      title: "操作",
      key: "action",
      fixed: "right",
      width: 100,
      render: (_, record) => {
        const actions = [
          {
            key: "collect-ports",
            label: "采集端口",
            icon: <CloudUploadOutlined />,
            onClick: () => handleCollectPorts(record),
            loading: collectingDeviceId === record.id,
          },
          {
            key: "detail",
            label: "详情",
            icon: <EyeOutlined />,
            onClick: () => openDetailModal(record),
          },
          // 14-05b:网络设备 → MAC 历史联动入口(D-16 锁定)
          ...(canViewMACHistory
            ? [
                {
                  key: "mac-history",
                  label: "查看 MAC 历史",
                  icon: <HistoryOutlined />,
                  onClick: () =>
                    navigate(`/network/mac/history?deviceId=${record.id}`),
                },
              ]
            : []),
          {
            key: "edit",
            label: "编辑",
            icon: <EditOutlined />,
            onClick: () => openModal(record),
          },
          {
            key: "delete",
            label: "删除",
            icon: <DeleteOutlined />,
            danger: true,
            onClick: () => {
              Modal.confirm({
                title: "确认删除?",
                okText: "确定",
                cancelText: "取消",
                okButtonProps: { danger: true },
                onOk: () => handleDelete(record.id),
              });
            },
          },
        ];

        return <ActionButtons actions={actions} />;
      },
    },
  ];
}

// ==================== 主组件 ====================

const DeviceManagement: FC = () => {
  const { message } = App.useApp();
  const navigate = useNavigate();

  // 权限(14-05b:查看 MAC 历史联动入口)— 与 history 页主权限点 network:mac:list 保持一致
  const menuPermissions = useMenuStore((s) => s.permissions);
  const canViewMACHistory = menuPermissions.includes("network:mac:list");

  // 表单实例
  const [quickCreateForm] = Form.useForm();
  const [searchForm] = Form.useForm();
  const [editForm] = Form.useForm();

  // 自定义状态
  const location = useLocation();
  const [selectedDeptId, setSelectedDeptId] = usePersistedStateController<string>({
    keyPrefix: location.pathname,
    keySuffix: "selectedDeptId",
    defaultValue: "",
  });
  const [collectingDeviceId, setCollectingDeviceId] = useState<string | null>(null);
  const [batchModalVisible, setBatchModalVisible] = useState(false);
  const [batchExporting, setBatchExporting] = useState(false);

  // 使用全局分页 hook
  const { paginationProps, setCurrent, setPageSize, setTotal } = usePagination();

  // 使用自定义 Hooks
  const {
    departments,
    credentials,
    statistics,
    loadCredentials,
    loadStatistics,
    ensureCredential,
  } = useDeviceData();

  const {
    quickCreateModalVisible,
    detailModalVisible,
    viewingDevice,
    probeResult,
    probing,
    creating,
    setQuickCreateModalVisible,
    setDetailModalVisible,
    setViewingDevice: _setViewingDevice,
    setProbeResult,
    setProbing,
    setCreating,
    openDetailModal,
    closeQuickCreateModal,
  } = useDeviceModals();

  // 服务端排序:field 对应后端 networkDeviceAllowedSortFields 白名单 key
  const sorterMetas = useMemo(
    () => [
      createSorterMeta<NetworkDevice>("deviceName"),
      createSorterMeta<NetworkDevice>("deviceType"),
      createSorterMeta<NetworkDevice>("vendor"),
      createSorterMeta<NetworkDevice>("model"),
      createSorterMeta<NetworkDevice>("ipAddress", "string"),
      createSorterMeta<NetworkDevice>("port", "number"),
      createSorterMeta<NetworkDevice>("status", "number"),
      createSorterMeta<NetworkDevice>("lastSeenAt", "date"),
      createSorterMeta<NetworkDevice>("createdAt", "date"),
    ],
    []
  );

  // 使用 useTableManager 管理表格状态
  const {
    loading,
    data: devices,
    selectedRowKeys,
    editModalVisible: modalVisible,
    editingItem: editingDevice,
    setSelectedRowKeys,
    setEditModalVisible,
    loadData: loadDevices,
    handleAdd,
    handleEdit,
    resetSelection,
    getColumnSortOrder,
    handleTableChange,
  } = useTableManager<NetworkDevice>(
    async (params) => {
      const values = searchForm.getFieldsValue();
      const result = await post<PageResponse<NetworkDevice>>("/network/devices/list", {
        ...values,
        ...params,
      });
      loadStatistics();
      return { list: result.data?.list || [], total: result.data?.total || 0 };
    },
    {
      pageSize: 10,
      externalPagination: {
        current: paginationProps.current ?? 1,
        pageSize: paginationProps.pageSize ?? 10,
        setCurrent,
        setPageSize,
        setTotal,
      },
      sorterMetas,
    }
  );

  // ==================== 数据操作 ====================

  // 部门树选择处理
  const handleDeptSelect = useCallback((selectedKeys: React.Key[]) => {
    if (selectedKeys.length > 0) {
      setSelectedDeptId(selectedKeys[0] as string);
      loadDevices({ deptId: selectedKeys[0] });
    } else {
      setSelectedDeptId("");
      loadDevices();
    }
  }, [loadDevices, setSelectedDeptId]);

  // 探测设备
  const handleProbe = async () => {
    try {
      const values = await quickCreateForm.validateFields(["ipAddress", "credentialId"]);
      setProbing(true);
      setProbeResult(null);

      const result = await post<ProbeResult>("/network/devices/discover", {
        ipAddress: values.ipAddress,
        credentialId: values.credentialId,
      });

      if (result.data?.success) {
        setProbeResult(result.data);
        handleSuccess("设备探测");
      } else {
        handleApiError(result.data?.message || "探测失败", "探测");
      }
    } catch (error) {
      if (error && typeof error === "object" && "errorFields" in error) {
        handleApiError(error, "请先填写 IP 地址和授权凭证", false);
      } else {
        handleApiError(error, "探测");
      }
    } finally {
      setProbing(false);
    }
  };

  // 快速创建设备
  const handleQuickCreate = async () => {
    try {
      const values = await quickCreateForm.validateFields();
      setCreating(true);

      await post("/network/devices/quick-create", {
        ipAddress: values.ipAddress,
        credentialId: values.credentialId,
        snmpPort: values.snmpPort || 161,
        communities: values.communities || [],
        deptId: values.deptId || undefined,
        location: values.location || "",
        description: values.description || "",
      });

      handleSuccess("设备创建");
      closeQuickCreateModal();
      quickCreateForm.resetFields();
      setProbeResult(null);

      if (selectedDeptId) {
        loadDevices({ deptId: selectedDeptId });
      } else {
        loadDevices();
      }
      loadStatistics();
    } catch (error) {
      if (error && typeof error === "object" && "errorFields" in error) {
        return;
      }
      handleApiError(error, "创建");
    } finally {
      setCreating(false);
    }
  };

  // 创建/更新设备
  const handleCreate = async () => {
    try {
      const values = await editForm.validateFields();
      if (editingDevice) {
        await post(`/network/devices/${editingDevice.id}/update`, values);
        handleSuccess("更新");
      } else {
        await post("/network/devices", values);
        handleSuccess("创建");
      }
      setEditModalVisible(false);
      editForm.resetFields();
      if (selectedDeptId) {
        loadDevices({ deptId: selectedDeptId });
      } else {
        loadDevices();
      }
      loadStatistics();
    } catch (error) {
      if (error && typeof error === "object" && "errorFields" in error) {
        return;
      }
      handleApiError(error, "操作");
    }
  };

  // 删除设备
  const handleDelete = async (id: string) => {
    try {
      await post(`/network/devices/${id}/delete`, {});
      handleSuccess("删除");
      if (selectedDeptId) {
        loadDevices({ deptId: selectedDeptId });
      } else {
        loadDevices();
      }
      loadStatistics();
    } catch (error) {
      handleApiError(error, "删除");
    }
  };

  // 批量删除设备
  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) {
      return;
    }
    try {
      await post("/network/devices/batch-delete", { ids: selectedRowKeys });
      handleSuccess("批量删除");
      resetSelection();
      if (selectedDeptId) {
        loadDevices({ deptId: selectedDeptId });
      } else {
        loadDevices();
      }
      loadStatistics();
    } catch (error) {
      handleApiError(error, "批量删除");
    }
  };

  const handleBatchExport = async (entityTypes: string[]) => {
    setBatchExporting(true);
    try {
      const filename = await batchExport(entityTypes, {}); // 可以根据需要添加筛选条件
      message.success(`批量导出成功，文件: ${filename}`);
      setBatchModalVisible(false);
    } catch (error: any) {
      message.error(`批量导出失败：${error.message}`);
    } finally {
      setBatchExporting(false);
    }
  };

  // 采集设备端口
  const handleCollectPorts = async (device: NetworkDevice) => {
    try {
      setCollectingDeviceId(device.id);
      const result = await post<{ errorMessage?: string; successCount?: number; failedCount?: number }>("/network/ports/collect", { deviceId: device.id });

      const data = result.data;
      if (!data) {
        handleApiError("采集失败: 未获取到端口数据", "采集");
      } else if (data.errorMessage) {
        handleApiError(data.errorMessage, "采集", false);
      } else if (data.successCount && data.successCount > 0) {
        handleSuccess(`采集成功: 成功 ${data.successCount} 个端口${data.failedCount && data.failedCount > 0 ? `，失败 ${data.failedCount} 个` : ""}`);
      } else {
        handleApiError("采集失败: 未获取到端口数据", "采集");
      }
    } catch (error) {
      handleApiError(error, "采集");
    } finally {
      setCollectingDeviceId(null);
    }
  };

  // 打开编辑模态框
  const openModal = (record?: NetworkDevice) => {
    if (record) {
      handleEdit(record);
      // 凭证兜底注入(2026-06-30,同 info-points):loadCredentials 异步且 pageSize:50
      // 可能不覆盖当前凭证 → 编辑回填 Select 显示 raw UUID。用 record.credentialName 注入。
      if (record.credentialId) {
        ensureCredential({ id: record.credentialId, credentialName: record.credentialName });
      }
      editForm.setFieldsValue({
        ...record,
        deptId: record.deptId || undefined,
        credentialId: record.credentialId || undefined,
      });
    } else {
      handleAdd();
      editForm.setFieldsValue({ port: 22, snmpPort: 161, status: 0 });
    }
    loadCredentials();
  };

  // 打开快速创建模态框
  const openQuickCreateModal = () => {
    quickCreateForm.resetFields();
    setProbeResult(null);
    setQuickCreateModalVisible(true);
    loadCredentials();
  };

  // ==================== 搜索和刷新 ====================

  const handleSearch = () => {
    const values = searchForm.getFieldsValue();
    const searchParams: Record<string, unknown> = {};
    Object.keys(values).forEach(key => {
      const value = values[key];
      if (value !== undefined && value !== null && value !== "") {
        searchParams[key] = value;
      }
    });
    if (selectedDeptId) {
      searchParams.deptId = selectedDeptId;
    }
    loadDevices(searchParams);
  };

  const handleReset = () => {
    searchForm.resetFields();
    if (selectedDeptId) {
      loadDevices({ deptId: selectedDeptId });
    } else {
      loadDevices();
    }
  };

  const handleRefresh = () => {
    if (selectedDeptId) {
      loadDevices({ deptId: selectedDeptId });
    } else {
      loadDevices();
    }
    loadStatistics();
  };

  // ==================== 初始化 ====================

  useEffect(() => {
    loadDevices();
    loadStatistics();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // 表格列
  const columns = getDeviceTableColumns({
    navigate,
    handleCollectPorts,
    openDetailModal,
    openModal,
    handleDelete,
    collectingDeviceId,
    canViewMACHistory,
    getColumnSortOrder,
  });

  return (
    <Layout style={{ background: "#000", minHeight: "calc(100vh - 64px)" }}>
      {/* 左侧部门树 */}
      <Sider width={360} className="dept-list-sider" style={{ background: "#fff", padding: "0 16px 16px 0", borderRight: "1px solid #f0f0f0" }}>
        <DeptTree
          onSelect={(selectedKeys) => handleDeptSelect(selectedKeys)}
          selectedKeys={selectedDeptId ? [selectedDeptId] : []}
        />
      </Sider>

      {/* 右侧内容区 */}
      <Content style={{ background: "#fff" }}>
        <div>
          {/* 统计卡片 */}
          {statistics.total > 10 && (
            <Row gutter={16} style={{ marginBottom: 16 }}>
              <Col span={6}>
                <Card>
                  <Statistic
                    title="设备总数"
                    value={statistics.total}
                    prefix={<CloudServerOutlined />}
                  />
                </Card>
              </Col>
              <Col span={6}>
                <Card>
                  <Statistic
                    title="在线设备"
                    value={statistics.online}
                    styles={{ content: { color: "var(--theme-success, #3f8600)" } }}
                    prefix={<CheckCircleOutlined />}
                  />
                </Card>
              </Col>
              <Col span={6}>
                <Card>
                  <Statistic
                    title="离线设备"
                    value={statistics.offline}
                    styles={{ content: { color: "var(--theme-error, #cf1322)" } }}
                    prefix={<CloseCircleOutlined />}
                  />
                </Card>
              </Col>
              <Col span={6}>
                <Card>
                  <Statistic
                    title="未知状态"
                    value={statistics.unknown}
                    styles={{ content: { color: "var(--theme-warning, #faad14)" } }}
                    prefix={<WarningOutlined />}
                  />
                </Card>
              </Col>
            </Row>
          )}

          {/* 搜索表单和操作按钮 */}
          <Card style={{ marginBottom: 16 }}>
            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start", flexWrap: "wrap", gap: "16px" }}>
              <Form form={searchForm} layout="inline" style={{ flex: 1, minWidth: 0 }}>
                <Form.Item name="deviceName" label="设备名称">
                  <Input placeholder="请输入设备名称" allowClear className="user-form-input" style={{ width: 150 }} />
                </Form.Item>
                <Form.Item name="ipAddress" label="IP地址">
                  <Input placeholder="请输入IP地址" allowClear className="user-form-input" style={{ width: 150 }} />
                </Form.Item>
                <Form.Item name="vendor" label="厂商">
                  <Select placeholder="请选择厂商" allowClear className="user-form-input" style={{ width: 120 }} onSearch={() => {}}>
                    {VENDOR_OPTIONS.map(opt => (
                      <Option key={opt.value} value={opt.value}>{opt.label}</Option>
                    ))}
                  </Select>
                </Form.Item>
                <Form.Item name="deviceType" label="设备类型">
                  <Select placeholder="请选择设备类型" allowClear className="user-form-input" style={{ width: 130 }} onSearch={() => {}}>
                    {DEVICE_TYPE_OPTIONS.map(opt => (
                      <Option key={opt.value} value={opt.value}>{opt.label}</Option>
                    ))}
                  </Select>
                </Form.Item>
                <Form.Item name="status" label="状态">
                  <Select placeholder="请选择状态" allowClear className="user-form-input" style={{ width: 100 }} onSearch={() => {}}>
                    {STATUS_OPTIONS.map(opt => (
                      <Option key={opt.value} value={opt.value}>{opt.label}</Option>
                    ))}
                  </Select>
                </Form.Item>
                <Form.Item>
                  <Space>
                    <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>查询</Button>
                    <Button icon={<ReloadOutlined />} onClick={handleReset}>重置</Button>
                    <Button icon={<ReloadOutlined />} onClick={handleRefresh}>刷新</Button>
                  </Space>
                </Form.Item>
              </Form>
              <Space>
                <NetworkExport
                  entityType="devices"
                  entityName="网络设备"
                  filters={(() => {
                    const values = searchForm.getFieldsValue() as Record<string, unknown>;
                    const filtered: Record<string, unknown> = {};
                    Object.keys(values).forEach(key => {
                      const value = values[key];
                      if (value !== undefined && value !== null && value !== "") {
                        filtered[key] = value;
                      }
                    });
                    return filtered;
                  })()}
                  current={paginationProps.current}
                  pageSize={paginationProps.pageSize}
                />
                <Button type="primary" icon={<ThunderboltOutlined />} onClick={openQuickCreateModal}>快速创建</Button>
                <Button type="default" icon={<PlusOutlined />} onClick={() => openModal()}>手动新增</Button>
                {selectedRowKeys.length > 0 && (
                  <Button icon={<DeleteOutlined />} style={{ color: "var(--theme-error, #ff4d4f)" }} onClick={handleBatchDelete}>
                    批量删除 ({selectedRowKeys.length})
                  </Button>
                )}
              </Space>{/* 批量导出 Modal */}

            <BatchExportModal

              visible={batchModalVisible}

              onConfirm={handleBatchExport}

              onCancel={() => setBatchModalVisible(false)}

              loading={batchExporting}

            />


            </div>
            {selectedRowKeys.length > 0 && (
              <Alert
                message={
                  <span>
                    已选择 <strong>{selectedRowKeys.length}</strong> 个设备，
                    <Button
                      type="link"
                      size="small"
                      onClick={() => setSelectedRowKeys([])}
                      style={{ padding: 0 }}
                    >
                      取消选择
                    </Button>
                  </span>
                }
                type="info"
                showIcon
                style={{ marginTop: 12 }}
              />
            )}
          </Card>

          {/* 设备表格 */}
          <Card>
            <Table
              rowSelection={{
                selectedRowKeys,
                onChange: setSelectedRowKeys,
              }}
              columns={columns}
              dataSource={devices}
              loading={loading}
              rowKey="id"
              scroll={{ x: 1600 }}
              pagination={paginationProps}
              onChange={handleTableChange}
            />
          </Card>
        </div>
      </Content>

      {/* 快速创建模态框 */}
      <Modal
        title="快速创建设备"
        open={quickCreateModalVisible}
        onOk={handleQuickCreate}
        onCancel={() => { closeQuickCreateModal(); quickCreateForm.resetFields(); }}
        width={700}
        okText="创建设备"
        confirmLoading={creating}
      >
        <Alert
          title="快速创建说明"
          description="只需输入 IP 地址和授权凭证，系统会自动使用凭证中配置的 SNMP Communities 探测获取设备信息并创建设备。"
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
        />

        <Form form={quickCreateForm} labelCol={{ span: 6 }} wrapperCol={{ span: 16 }}>
          <Form.Item name="ipAddress" label="IP地址" rules={[{ required: true, message: "请输入IP地址" }]}>
            <Input placeholder="请输入设备IP地址" />
          </Form.Item>
          <Form.Item name="credentialId" label="授权凭证" rules={[{ required: true, message: "请选择授权凭证" }]}>
            <Select placeholder="请选择授权凭证" onSearch={() => {}}>
              {credentials.map(cred => (
                <Option key={cred.id} value={cred.id}>{cred.credentialName}</Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item label=" ">
            <Col offset={6}>
              <Space>
                <Button type="primary" icon={<ApiOutlined />} onClick={handleProbe} loading={probing}>
                  探测设备
                </Button>
                <Button onClick={() => setProbeResult(null)}>清除结果</Button>
              </Space>
            </Col>
          </Form.Item>

          {probeResult && (
            <>
              <Divider orientationMargin={0} orientation={"left" as DividerProps["orientation"]}>探测结果</Divider>
              <Alert
                title="探测成功"
                type="success"
                style={{ marginBottom: 16 }}
              />
              <Row gutter={16}>
                <Col span={12}>
                  <Card size="small" title="设备信息">
                    <p><strong>设备名称:</strong> {(probeResult.deviceName as string) || "-"}</p>
                    <p><strong>设备类型:</strong> {(probeResult.deviceType as string) || "-"}</p>
                    <p><strong>厂商:</strong> {(probeResult.vendor as string) || "-"}</p>
                    <p><strong>型号:</strong> {(probeResult.model as string) || "-"}</p>
                  </Card>
                </Col>
                <Col span={12}>
                  <Card size="small" title="系统信息">
                    <p><strong>SysName:</strong> {(probeResult.sysName as string) || "-"}</p>
                    <p><strong>SysDescr:</strong> {(probeResult.sysDescr as string)?.substring(0, 50) || "-"}{(probeResult.sysDescr as string)?.length > 50 ? "..." : ""}</p>
                  </Card>
                </Col>
              </Row>
            </>
          )}

          <Divider orientationMargin={0} orientation={"left" as DividerProps["orientation"]}>其他信息（可选）</Divider>
          <Form.Item name="deptId" label="所属部门">
            <DepartmentTreeSelect departments={departments} />
          </Form.Item>
          <Form.Item name="location" label="位置">
            <Input placeholder="请输入设备位置" />
          </Form.Item>
          <Form.Item name="description" label="备注">
            <TextArea rows={2} placeholder="请输入备注" />
          </Form.Item>
        </Form>
      </Modal>

      {/* 手动新增/编辑模态框 */}
      <Modal
        title={editingDevice ? "编辑设备" : "手动新增设备"}
        open={modalVisible}
        onOk={handleCreate}
        onCancel={() => { setEditModalVisible(false); editForm.resetFields(); }}
        width={700}
      >
        <Alert
          title="手动新增说明"
          description="需要手动填写所有设备信息。如果只想输入IP地址自动获取信息，请使用「快速创建」功能。"
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
        />

        <Form form={editForm} labelCol={{ span: 6 }} wrapperCol={{ span: 16 }}>
          <Form.Item name="deviceName" label="设备名称" rules={[{ required: true, message: "请输入设备名称" }]}>
            <Input placeholder="请输入设备名称" />
          </Form.Item>
          <Form.Item name="deviceType" label="设备类型" rules={[{ required: true, message: "请选择设备类型" }]}>
            <Select placeholder="请选择设备类型" onSearch={() => {}}>
              {DEVICE_TYPE_OPTIONS.map(opt => (
                <Option key={opt.value} value={opt.value}>{opt.label}</Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="vendor" label="厂商" rules={[{ required: true, message: "请选择厂商" }]}>
            <Select placeholder="请选择厂商" onSearch={() => {}}>
              {VENDOR_OPTIONS.map(opt => (
                <Option key={opt.value} value={opt.value}>{opt.label}</Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="model" label="型号">
            <Input placeholder="请输入设备型号" />
          </Form.Item>
          <Form.Item name="ipAddress" label="IP地址" rules={[{ required: true, message: "请输入IP地址" }]}>
            <Input placeholder="请输入IP地址" />
          </Form.Item>
          <Form.Item name="port" label="SSH端口" rules={[{ required: true, message: "请输入SSH端口" }]}>
            <InputNumber min={1} max={65535} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item name="snmpPort" label="SNMP端口" rules={[{ required: true, message: "请输入SNMP端口" }]}>
            <InputNumber min={1} max={65535} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item name="credentialId" label="授权凭证">
            <Select placeholder="请选择授权凭证" allowClear onSearch={() => {}}>
              {credentials.map(cred => (
                <Option key={cred.id} value={cred.id}>{cred.credentialName}</Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="deptId" label="所属部门">
            <DepartmentTreeSelect
              departments={departments}
            />
          </Form.Item>
          <Form.Item name="location" label="位置">
            <Input placeholder="请输入位置" />
          </Form.Item>
          <Form.Item name="status" label="状态" rules={[{ required: true }]}>
            <Select onSearch={() => {}}>
              {STATUS_OPTIONS.map(opt => (
                <Option key={opt.value} value={opt.value}>{opt.label}</Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="description" label="备注">
            <TextArea rows={3} placeholder="请输入备注" />
          </Form.Item>
        </Form>
      </Modal>

      {/* 设备详情模态框 */}
      <Modal
        title="设备详细信息"
        open={detailModalVisible}
        onCancel={() => setDetailModalVisible(false)}
        footer={[
          // 14-05b:详情页头部操作区新增 查看 MAC 历史 联动入口(D-16 锁定)
          ...(canViewMACHistory && viewingDevice
            ? [
                <Button
                  key="mac-history"
                  icon={<HistoryOutlined />}
                  onClick={() => {
                    if (viewingDevice) {
                      navigate(`/network/mac/history?deviceId=${viewingDevice.id}`);
                    }
                  }}
                >
                  查看 MAC 历史
                </Button>,
              ]
            : []),
          <Button key="close" onClick={() => setDetailModalVisible(false)}>关闭</Button>,
        ]}
        width={800}
      >
        {viewingDevice && (
          <div>
            <Divider orientationMargin={0} orientation={"left" as DividerProps["orientation"]}>基本信息</Divider>
            <Row gutter={16}>
              <Col span={12}>
                <p><strong>设备名称:</strong> {viewingDevice.deviceName}</p>
                <p><strong>设备类型:</strong> {getOptionLabel(DEVICE_TYPE_OPTIONS, viewingDevice.deviceType) || "-"}</p>
                <p><strong>厂商:</strong> {getOptionLabel(VENDOR_OPTIONS, viewingDevice.vendor) || "-"}</p>
                <p><strong>型号:</strong> {viewingDevice.model || "-"}</p>
                <p><strong>IP地址:</strong> {viewingDevice.ipAddress}</p>
                <p><strong>SSH端口:</strong> {viewingDevice.port}</p>
                <p><strong>SNMP端口:</strong> {viewingDevice.snmpPort}</p>
              </Col>
              <Col span={12}>
                <p><strong>状态:</strong> {getOptionLabel(STATUS_OPTIONS, viewingDevice.status) || "-"}</p>
                <p><strong>所属部门:</strong> {viewingDevice.deptName || "-"}</p>
                <p><strong>位置:</strong> {viewingDevice.location || "-"}</p>
                <p><strong>授权凭证:</strong> {viewingDevice.credentialName || "-"}</p>
                <p><strong>最后连接:</strong> {viewingDevice.lastSeenAt || "-"}</p>
                <p><strong>创建时间:</strong> {viewingDevice.createdAt}</p>
              </Col>
            </Row>

            <Divider orientationMargin={0} orientation={"left" as DividerProps["orientation"]}>设备详细信息（SSH采集）</Divider>
            <Row gutter={16}>
              <Col span={12}>
                <p><strong>型号:</strong> {viewingDevice.model || "-"}</p>
                <p><strong>序列号:</strong> {viewingDevice.serialNumber || "-"}</p>
              </Col>
              <Col span={12}>
                <p><strong>软件版本:</strong> {viewingDevice.softwareVersion || "-"}</p>
                <p><strong>运行时间:</strong> {viewingDevice.uptime || "-"}</p>
              </Col>
            </Row>

            {viewingDevice.description && (
              <>
                <Divider orientationMargin={0} orientation={"left" as DividerProps["orientation"]}>备注</Divider>
                <p>{viewingDevice.description}</p>
              </>
            )}
          </div>
        )}
      </Modal>
    </Layout>
  );
};

export default DeviceManagement;

