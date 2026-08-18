/**
 * 机房设备管理页面
 */

import { useState, useCallback, useEffect, useRef, useMemo } from "react";
import type { FC } from "react";
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
  Alert,
  Radio,
  DatePicker,
  Layout,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  SearchOutlined,
  ReloadOutlined,
  CheckCircleOutlined,
  StopOutlined,
  WarningOutlined,
  AppstoreOutlined,
  TableOutlined,
  ImportOutlined,
  ExportOutlined,
  AppstoreAddOutlined,
} from "@ant-design/icons";
import type { RoomDevice } from "@/types";
import { roomDeviceApi, serverRoomApi } from "@/lib/opsApi";
import { useTableManager } from "@/hooks/useTableManager";
import { usePagination } from "@/hooks/usePagination";
import { useSidebarDeptFilter } from "@/hooks/useSidebarDeptFilter";
import { handleApiError, handleSuccess, isFormValidationError } from "@/utils/errorHandler";
import { createDateTimeColumn, createSorterMeta } from "@/utils/tableHelpers";
import ActionButtons from "@/components/shared/ActionButtons";
import ExcelImport from "@/components/shared/ExcelImport";
import ExcelExport from "@/components/shared/ExcelExport";
import { DeptSidebar } from "@/components/operations/DeptSidebar";
import { StatisticsCards } from "@/components/operations/StatisticsCards";

const { Option } = Select;
const { TextArea } = Input;
const { Content } = Layout;

type ViewMode = "table" | "card";

interface RoomOption {
  id: string;
  name: string;
  buildingId: string;
  orgId: string;
}

interface Statistics {
  total: number;
  normal: number;
  fault: number;
  scrapped: number;
}

const RoomDeviceManagement: FC = () => {
  const location = useLocation();
  const [viewMode, setViewMode] = usePersistedStateController<ViewMode>({
    keyPrefix: location.pathname,
    keySuffix: "viewMode",
    defaultValue: "table",
  });
  const [importVisible, setImportVisible] = useState(false);
  const [exportVisible, setExportVisible] = useState(false);
  const [exportFilters, setExportFilters] = useState<Record<string, unknown>>({});
  const [roomOptions, setRoomOptions] = useState<RoomOption[]>([]);
  const [statistics, setStatistics] = useState<Statistics>({
    total: 0,
    normal: 0,
    fault: 0,
    scrapped: 0,
  });

  const { paginationProps, setCurrent, setPageSize, setTotal } = usePagination();

  // 使用 useSidebarDeptFilter hook 管理部门筛选
  const {
    selectedDeptId,
    handleDeptSelect,
    setSelectedDeptId: _setSelectedDeptId,
  } = useSidebarDeptFilter({
    searchForm: undefined,
    clearFieldNames: ["roomId"],
  });

  // 使用 ref 存储最新的 selectedDeptId，避免闭包问题
  const selectedDeptIdRef = useRef<string>("");
  useEffect(() => {
    selectedDeptIdRef.current = selectedDeptId;
  }, [selectedDeptId]);

  const loadRoomOptions = useCallback(async (orgId?: string) => {
    try {
      const params: Record<string, unknown> = { current: 1, pageSize: 50 };
      if (orgId) params.orgId = orgId;
      const result = await serverRoomApi.list(params);
      const rooms = result.data?.list || [];
      setRoomOptions(
        rooms.map((r) => ({
          id: r.id,
          name: r.name,
          buildingId: r.buildingId,
          orgId: r.orgId || "",
        }))
      );
    } catch (error) {
      handleApiError(error, "加载机房选项", false);
    }
  }, []);

  // 服务端排序:field 必须与 columns 的 dataIndex 一致
  const sorterMetas = useMemo(
    () => [
      createSorterMeta<RoomDevice>("name"),
      createSorterMeta<RoomDevice>("deviceCode"),
      createSorterMeta<RoomDevice>("roomName"),
      createSorterMeta<RoomDevice>("deviceType"),
      createSorterMeta<RoomDevice>("status"),
      createSorterMeta<RoomDevice>("createdAt", "date"),
    ],
    []
  );

  const {
    loading,
    data: devices,
    total,
    selectedRowKeys,
    searchForm,
    editForm: deviceForm,
    editModalVisible: modalVisible,
    editingItem: editingDevice,
    setSelectedRowKeys,
    setEditModalVisible: setModalVisible,
    setEditingItem: setEditingDevice,
    handleSearch,
    handleReset,
    handleAdd,
    handleEdit,
    handleModalClose,
    loadData: loadDevices,
    resetSelection,
    handleTableChange: handleDeviceTableChange,
    getColumnSortOrder: getDeviceColumnSortOrder,
  } = useTableManager<RoomDevice>(
    async (params) => {
      // 使用 ref 获取最新的部门 ID，避免闭包捕获旧值
      const currentDeptId = selectedDeptIdRef.current;
      const searchParams = currentDeptId ? { ...params, orgId: currentDeptId } : params;
      const result = await roomDeviceApi.list(searchParams);
      return { list: result.data?.list || [], total: result.data?.total || 0 };
    },
    {
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

  // 当部门变化时，重新加载数据并重置分页
  const loadDevicesByDept = useCallback(async () => {
    setCurrent(1);
    loadDevices();
  }, [setCurrent, loadDevices]);

  const loadStatistics = useCallback(async (): Promise<Statistics> => {
    try {
      const stats = await roomDeviceApi.statistics();
      return {
        total: stats.total ?? 0,
        normal: stats.normal ?? 0,
        fault: stats.fault ?? 0,
        scrapped: stats.scrapped ?? 0,
      };
    } catch (error) {
      handleApiError(error, "加载统计数据", false);
      return { total: 0, normal: 0, fault: 0, scrapped: 0 };
    }
  }, []);

  // 统一的数据刷新函数
  const refreshData = useCallback(() => {
    loadDevices();
    loadStatistics().then(setStatistics);
  }, [loadDevices, loadStatistics]);

  // 初始化加载
  useEffect(() => {
    loadStatistics().then(setStatistics);
  }, [loadStatistics]);

  // 当部门变化时，重新加载数据和机房选项
  useEffect(() => {
    loadDevicesByDept();
    // eslint-disable-next-line react-hooks/set-state-in-effect
    loadRoomOptions(selectedDeptId || undefined);
  }, [selectedDeptId, loadDevicesByDept, loadRoomOptions]);

  // 部门变化时清空搜索表单中的机房筛选字段
  useEffect(() => {
    searchForm.setFieldValue("roomId", undefined);
    searchForm.setFieldValue("name", undefined);
    searchForm.setFieldValue("deviceType", undefined);
    searchForm.setFieldValue("status", undefined);
  }, [selectedDeptId, searchForm]);

  const handleSave = async () => {
    try {
      const values = await deviceForm.validateFields();
      if (editingDevice) {
        await roomDeviceApi.update(editingDevice.id, values as Partial<RoomDevice>);
        handleSuccess("更新");
      } else {
        await roomDeviceApi.create(values as Partial<RoomDevice>);
        handleSuccess("创建");
      }
      handleModalClose();
      refreshData();
    } catch (error: unknown) {
      if (isFormValidationError(error)) return;
      handleApiError(error, "操作");
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await roomDeviceApi.delete(id);
      handleSuccess("删除");
      refreshData();
    } catch (error) {
      handleApiError(error, "删除");
    }
  };

  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) return;
    try {
      await roomDeviceApi.batch("delete", { ids: selectedRowKeys });
      handleSuccess("批量删除");
      resetSelection();
      refreshData();
    } catch (error) {
      handleApiError(error, "批量删除");
    }
  };

  const openModal = (record?: RoomDevice) => {
    if (record) {
      handleEdit(record);
      // 机房兜底注入(2026-06-30,同 info-points):loadRoomOptions 异步且 pageSize:50
      // 可能不包含当前 roomId → Select 显示 raw UUID。用 record.roomName 注入。
      if (record.roomId) {
        setRoomOptions((prev) =>
          prev.find((r) => r.id === record.roomId)
            ? prev
            : [
                ...prev,
                {
                  id: record.roomId,
                  name: record.roomName || "未命名机房",
                  buildingId: "",
                  orgId: "",
                },
              ]
        );
      }
      deviceForm.setFieldsValue(record);
    } else {
      handleAdd();
      deviceForm.setFieldsValue({ status: 0, deviceType: "server" });
    }
  };

  const handleImportSuccess = () => {
    refreshData();
    setImportVisible(false);
  };

  const getDeviceTypeText = (type: string) => {
    const types: Record<string, string> = {
      server: "服务器",
      storage: "存储设备",
      ups: "UPS",
      pdu: "PDU",
      firewall: "防火墙",
      kvm: "KVM",
      cabinet: "机柜",
      other: "其他",
    };
    return types[type] || type;
  };

  const getStatusText = (status: number) => {
    const statusMap: Record<number, string> = { 0: "正常", 1: "故障", 2: "报废" };
    return statusMap[status] || "未知";
  };

  const columns: ColumnsType<RoomDevice> = [
    {
      title: "设备名称",
      dataIndex: "name",
      key: "name",
      width: 150,
      sorter: true,
      sortOrder: getDeviceColumnSortOrder("name"),
    },
    {
      title: "设备编码",
      dataIndex: "deviceCode",
      key: "deviceCode",
      width: 150,
      sorter: true,
      sortOrder: getDeviceColumnSortOrder("deviceCode"),
    },
    {
      title: "所属机房",
      dataIndex: "roomName",
      key: "roomName",
      width: 120,
      sorter: true,
      sortOrder: getDeviceColumnSortOrder("roomName"),
      render: (roomName) => roomName || "-",
    },
    {
      title: "设备类型",
      dataIndex: "deviceType",
      key: "deviceType",
      width: 100,
      sorter: true,
      sortOrder: getDeviceColumnSortOrder("deviceType"),
      render: (type) => <Tag>{getDeviceTypeText(type as string)}</Tag>,
    },
    { title: "厂商", dataIndex: "vendor", key: "vendor", width: 100, render: (v) => v || "-" },
    { title: "型号", dataIndex: "model", key: "model", width: 120, render: (v) => v || "-" },
    {
      title: "位置",
      dataIndex: "positionDesc",
      key: "positionDesc",
      width: 100,
      render: (v) => v || "-",
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 80,
      sorter: true,
      sortOrder: getDeviceColumnSortOrder("status"),
      render: (status) => {
        const colors: Record<number, string> = { 0: "success", 1: "error", 2: "default" };
        return (
          <Tag color={colors[status as keyof typeof colors]}>{getStatusText(status as number)}</Tag>
        );
      },
    },
    createDateTimeColumn("createdAt", {
      width: 180,
      sorter: true,
      sortOrder: getDeviceColumnSortOrder("createdAt"),
    }),
    {
      title: "操作",
      key: "action",
      render: (_, record) => {
        const actions = [
          { key: "edit", label: "编辑", icon: <EditOutlined />, onClick: () => openModal(record) },
          {
            key: "delete",
            label: "删除",
            icon: <DeleteOutlined />,
            danger: true,
            onClick: () => {
              Modal.confirm({
                title: "确定要删除这个设备吗？",
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

  const renderCardView = () => {
    if (devices.length === 0)
      return (
        <div
          style={{
            textAlign: "center",
            padding: "40px",
            color: "var(--theme-text-tertiary, #999)",
          }}
        >
          暂无数据
        </div>
      );
    return (
      <Row gutter={[16, 16]}>
        {devices.map((device) => (
          <Col key={device.id} xs={24} sm={12} md={8} lg={6}>
            <Card
              hoverable
              actions={[
                <EditOutlined key="edit" onClick={() => openModal(device)} />,
                <DeleteOutlined
                  key="delete"
                  style={{ color: "var(--theme-error, #ba3630)" }}
                  onClick={() => {
                    Modal.confirm({
                      title: "确定要删除这个设备吗？",
                      okText: "确定",
                      cancelText: "取消",
                      okButtonProps: { danger: true },
                      onOk: () => handleDelete(device.id),
                    });
                  }}
                />,
              ]}
            >
              <Card.Meta
                title={
                  <div
                    style={{
                      display: "flex",
                      justifyContent: "space-between",
                      alignItems: "center",
                    }}
                  >
                    <span>{device.name}</span>
                    <Tag
                      color={
                        device.status === 0 ? "success" : device.status === 1 ? "error" : "default"
                      }
                    >
                      {getStatusText(device.status)}
                    </Tag>
                  </div>
                }
                description={
                  <div>
                    <div>
                      <strong>编码：</strong>
                      {device.deviceCode}
                    </div>
                    <div>
                      <strong>机房：</strong>
                      {device.roomName || "-"}
                    </div>
                    <div>
                      <strong>类型：</strong>
                      {getDeviceTypeText(device.deviceType)}
                    </div>
                    {device.vendor && (
                      <div>
                        <strong>厂商：</strong>
                        {device.vendor}
                      </div>
                    )}
                    {device.model && (
                      <div>
                        <strong>型号：</strong>
                        {device.model}
                      </div>
                    )}
                  </div>
                }
              />
            </Card>
          </Col>
        ))}
      </Row>
    );
  };

  return (
    <Layout style={{ background: "#000", minHeight: "calc(100vh - 64px)" }}>
      <DeptSidebar selectedDeptId={selectedDeptId} onSelect={handleDeptSelect} />
      <Content style={{ background: "#fff" }}>
        <StatisticsCards
          show={total > 10}
          items={[
            { title: "总设备数", value: statistics.total, prefix: <AppstoreAddOutlined /> },
            {
              title: "正常",
              value: statistics.normal,
              styles: { content: { color: "var(--theme-success, #3f8600)" } },
              prefix: <CheckCircleOutlined />,
            },
            {
              title: "故障",
              value: statistics.fault,
              styles: { content: { color: "var(--theme-error, #cf1322)" } },
              prefix: <WarningOutlined />,
            },
            {
              title: "报废",
              value: statistics.scrapped,
              styles: { content: { color: "var(--theme-text-tertiary, #707068)" } },
              prefix: <StopOutlined />,
            },
          ]}
        />
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
              <Form.Item name="roomId" label="所属机房">
                <Select
                  placeholder={selectedDeptId ? "请选择机房" : "请先选择部门"}
                  allowClear
                  showSearch
                  filterOption={(input, option) =>
                    String(option?.label ?? "")
                      .toLowerCase()
                      .includes(input.toLowerCase())
                  }
                  onSearch={() => {}}
                  style={{ width: 150 }}
                  disabled={!selectedDeptId}
                  options={roomOptions.map((r) => ({ label: r.name, value: r.id }))}
                />
              </Form.Item>
              <Form.Item name="name" label="设备名称">
                <Input placeholder="请输入设备名称" allowClear style={{ width: 150 }} />
              </Form.Item>
              <Form.Item name="deviceType" label="设备类型">
                <Select
                  placeholder="请选择类型"
                  allowClear
                  style={{ width: 130 }}
                  onSearch={() => {}}
                >
                  <Option value="server">服务器</Option>
                  <Option value="storage">存储设备</Option>
                  <Option value="ups">UPS</Option>
                  <Option value="pdu">PDU</Option>
                  <Option value="firewall">防火墙</Option>
                  <Option value="kvm">KVM</Option>
                  <Option value="cabinet">机柜</Option>
                  <Option value="other">其他</Option>
                </Select>
              </Form.Item>
              <Form.Item name="status" label="状态">
                <Select
                  placeholder="请选择状态"
                  allowClear
                  style={{ width: 100 }}
                  onSearch={() => {}}
                >
                  <Option value={0}>正常</Option>
                  <Option value={1}>故障</Option>
                  <Option value={2}>报废</Option>
                </Select>
              </Form.Item>
              <Form.Item>
                <Space>
                  <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>
                    搜索
                  </Button>
                  <Button onClick={handleReset}>重置</Button>
                  <Button icon={<ReloadOutlined />} onClick={refreshData}>
                    刷新
                  </Button>
                </Space>
              </Form.Item>
            </Form>
            <Space>
              <Radio.Group
                value={viewMode}
                onChange={(e) => setViewMode(e.target.value)}
                buttonStyle="solid"
              >
                <Radio.Button value="table">
                  <TableOutlined /> 表格
                </Radio.Button>
                <Radio.Button value="card">
                  <AppstoreOutlined /> 卡片
                </Radio.Button>
              </Radio.Group>
              <Button icon={<ImportOutlined />} onClick={() => setImportVisible(true)}>
                导入
              </Button>
              <Button
                icon={<ExportOutlined />}
                onClick={() => {
                  const values = searchForm.getFieldsValue() as Record<string, unknown>;
                  const currentFilters: Record<string, unknown> = {};
                  Object.keys(values).forEach((key) => {
                    const value = values[key];
                    if (value !== undefined && value !== null && value !== "") {
                      currentFilters[key] = value;
                    }
                  });
                  setExportFilters(currentFilters);
                  setExportVisible(true);
                }}
              >
                导出
              </Button>
              {selectedRowKeys.length > 0 && (
                <Button
                  icon={<DeleteOutlined />}
                  style={{ color: "var(--theme-error, #ba3630)" }}
                  onClick={handleBatchDelete}
                >
                  批量删除 ({selectedRowKeys.length})
                </Button>
              )}
              <Button type="primary" icon={<PlusOutlined />} onClick={() => openModal()}>
                新增设备
              </Button>
            </Space>
          </div>
          {selectedRowKeys.length > 0 && (
            <Alert
              title={
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
        <Card>
          {viewMode === "table" ? (
            <Table
              rowSelection={{ selectedRowKeys, onChange: setSelectedRowKeys }}
              columns={columns}
              dataSource={devices}
              loading={loading}
              rowKey="id"
              pagination={paginationProps}
              onChange={handleDeviceTableChange}
            />
          ) : (
            renderCardView()
          )}
        </Card>
        <Modal
          title={editingDevice ? "编辑设备" : "新增设备"}
          open={modalVisible}
          onOk={handleSave}
          onCancel={() => {
            setModalVisible(false);
            deviceForm.resetFields();
            setEditingDevice(null);
          }}
          width={700}
        >
          <Form
            form={deviceForm}
            layout="horizontal"
            labelCol={{ span: 5 }}
            wrapperCol={{ span: 19 }}
          >
            <Form.Item
              name="roomId"
              label="所属机房"
              rules={[{ required: true, message: "请选择所属机房" }]}
            >
              <Select
                placeholder="请选择所属机房"
                showSearch
                filterOption={(input, option) =>
                  String(option?.label ?? "")
                    .toLowerCase()
                    .includes(input.toLowerCase())
                }
                onSearch={() => {}}
                options={roomOptions.map((r) => ({ label: r.name, value: r.id }))}
              />
            </Form.Item>
            <Form.Item
              name="name"
              label="设备名称"
              rules={[{ required: true, message: "请输入设备名称" }]}
            >
              <Input placeholder="请输入设备名称" />
            </Form.Item>
            <Form.Item
              name="deviceCode"
              label="设备编码"
              rules={[{ required: true, message: "请输入设备编码" }]}
            >
              <Input placeholder="请输入设备编码" />
            </Form.Item>
            <Form.Item
              name="deviceType"
              label="设备类型"
              rules={[{ required: true, message: "请选择设备类型" }]}
            >
              <Select placeholder="请选择设备类型" onSearch={() => {}}>
                <Option value="server">服务器</Option>
                <Option value="storage">存储设备</Option>
                <Option value="ups">UPS</Option>
                <Option value="pdu">PDU</Option>
                <Option value="firewall">防火墙</Option>
                <Option value="kvm">KVM</Option>
                <Option value="cabinet">机柜</Option>
                <Option value="other">其他</Option>
              </Select>
            </Form.Item>
            <Row gutter={16}>
              <Col span={12}>
                <Form.Item name="vendor" label="厂商">
                  <Input placeholder="请输入厂商" />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="model" label="型号">
                  <Input placeholder="请输入型号" />
                </Form.Item>
              </Col>
            </Row>
            <Row gutter={16}>
              <Col span={12}>
                <Form.Item name="serialNumber" label="序列号">
                  <Input placeholder="请输入序列号" />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="positionDesc" label="位置描述">
                  <Input placeholder="请输入位置描述" />
                </Form.Item>
              </Col>
            </Row>
            <Row gutter={16}>
              <Col span={12}>
                <Form.Item name="positionU" label="U位置">
                  <InputNumber min={0} placeholder="U位置" style={{ width: "100%" }} />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="heightU" label="高度(U)">
                  <InputNumber min={0} placeholder="高度" style={{ width: "100%" }} />
                </Form.Item>
              </Col>
            </Row>
            <Row gutter={16}>
              <Col span={12}>
                <Form.Item name="powerConsumption" label="功耗(W)">
                  <InputNumber min={0} placeholder="功耗" style={{ width: "100%" }} />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item
                  name="status"
                  label="状态"
                  rules={[{ required: true, message: "请选择状态" }]}
                >
                  <Select placeholder="请选择状态" onSearch={() => {}}>
                    <Option value={0}>正常</Option>
                    <Option value={1}>故障</Option>
                    <Option value={2}>报废</Option>
                  </Select>
                </Form.Item>
              </Col>
            </Row>
            <Row gutter={16}>
              <Col span={12}>
                <Form.Item name="purchaseDate" label="购买日期">
                  <DatePicker style={{ width: "100%" }} />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="warrantyDate" label="保修到期">
                  <DatePicker style={{ width: "100%" }} />
                </Form.Item>
              </Col>
            </Row>
            <Form.Item name="description" label="描述">
              <TextArea rows={3} placeholder="请输入描述" />
            </Form.Item>
          </Form>
        </Modal>
        <ExcelImport
          entityType="roomDevice"
          entityName="机房设备"
          visible={importVisible}
          onClose={() => setImportVisible(false)}
          onImportSuccess={handleImportSuccess}
        />
        <ExcelExport
          entityType="roomDevice"
          entityName="机房设备"
          visible={exportVisible}
          onClose={() => setExportVisible(false)}
          filters={exportFilters}
        />
      </Content>
    </Layout>
  );
};

export default RoomDeviceManagement;
