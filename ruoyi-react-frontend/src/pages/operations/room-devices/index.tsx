/**
 * 机房设备管理页面 — Phase 68.1 T6b：operations 域最后一页（PageTable 全契约形态）
 *
 * 迁移自页面级双 hook 数据层实现：数据层 / 搜索区 /
 * 分页 / 三态 / 删除确认全部收敛到 PageTable（UI-SPEC §3.2），本页只保留业务块
 * （部门侧边栏、统计卡片、已选择 Alert、视图切换、导入导出、新增/编辑弹窗表单）。
 *
 * 部门筛选：selectedDeptId 经 PageTable extraFilters（orgId）驱动——引用变化即回第 1 页
 * 重载；部门切换时先清空搜索字段，再交由 applyFilters 合并 extra，保持迁移前"切部门清筛选"
 * 的语义。
 *
 * 视图切换（表格 / 卡片）：卡片数据取自同一 request 的最近一次快照；表格仍由 PageTable
 * 承载、在卡片模式下隐藏——搜索 / 排序 / 分页 / 批删 / 三态与表格视图共用同一数据层
 * （PageTableRenderContext 不暴露 data，故镜像一份快照）。
 */

import { useState, useCallback, useEffect, useRef, useMemo } from "react";
import type { FC, Key } from "react";
import type { FormInstance } from "antd";
import { useLocation } from "react-router-dom";
import { usePersistedStateController } from "@/hooks/usePersistedState";
import {
  Button,
  Space,
  Modal,
  Form,
  InputNumber,
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
import { useSidebarDeptFilter } from "@/hooks/useSidebarDeptFilter";
import { handleApiError, handleSuccess, isFormValidationError } from "@/utils/errorHandler";
import { createDateTimeColumn, createSorterMeta } from "@/utils/tableHelpers";
import { PageTable } from "@/components/shared";
import type { PageTableRenderContext, SearchField } from "@/components/shared";
import { createDeleteAction } from "@/components/shared/ActionButtons";
import ExcelImportLazy from "@/components/shared/ExcelImportLazy";
import ExcelExport from "@/components/shared/ExcelExport";
import { confirmDelete } from "@/components/shared/confirmDelete";
import { DeptSidebar } from "@/components/operations/DeptSidebar";
import { StatisticsCards } from "@/components/operations/StatisticsCards";
// Phase 71 (W3)：裸 Input/Select/TextArea → 契约组件。
// roomId 为 UUID 字段，allowClear + 自定义 filterOption（按 label 子串）原样保留；
// deviceType/status 为固定枚举 → options 改写并显式 showSearch={false}。
// 注：本文件原有 `@/components/ui/AppSelect` 直接导入（搜索区已在用），统一收敛到
// `@/components/ui` 具名导出，避免重复标识符。
import { AppInput, AppSelect, AppTextArea } from "@/components/ui";

const { Content } = Layout;

// 模块级静态映射（Phase m76-js-misc 6.3）
const ROOM_DEVICE_STATUS_MAP: Record<number, string> = { 0: "正常", 1: "故障", 2: "报废" };

// 设备类型固定枚举（AppSelect 需 options 形态）
const ROOM_DEVICE_TYPE_OPTIONS = [
  { label: "服务器", value: "server" },
  { label: "存储设备", value: "storage" },
  { label: "UPS", value: "ups" },
  { label: "PDU", value: "pdu" },
  { label: "防火墙", value: "firewall" },
  { label: "KVM", value: "kvm" },
  { label: "机柜", value: "cabinet" },
  { label: "其他", value: "other" },
];

// 搜索区选项（值沿用后端 status 口径 0/1/2）
const ROOM_DEVICE_STATUS_OPTIONS = [
  { label: "正常", value: 0 },
  { label: "故障", value: 1 },
  { label: "报废", value: 2 },
];

// 设备类型选项（值同后端 deviceType 口径）
const DEVICE_TYPE_OPTIONS = [
  { label: "服务器", value: "server" },
  { label: "存储设备", value: "storage" },
  { label: "UPS", value: "ups" },
  { label: "PDU", value: "pdu" },
  { label: "防火墙", value: "firewall" },
  { label: "KVM", value: "kvm" },
  { label: "机柜", value: "cabinet" },
  { label: "其他", value: "other" },
];

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
  const [submitting, setSubmitting] = useState(false);
  const [exportVisible, setExportVisible] = useState(false);
  const [exportFilters, setExportFilters] = useState<Record<string, unknown>>({});
  const [roomOptions, setRoomOptions] = useState<RoomOption[]>([]);
  const [statistics, setStatistics] = useState<Statistics>({
    total: 0,
    normal: 0,
    fault: 0,
    scrapped: 0,
  });

  // 卡片视图数据快照（与表格同一 request，不另建数据层）
  const [cardDevices, setCardDevices] = useState<RoomDevice[]>([]);

  // 新增/编辑弹窗态（原表格管理器 hook 的 editForm/editModalVisible/editingItem 位）
  const [modalVisible, setModalVisible] = useState(false);
  const [editingDevice, setEditingDevice] = useState<RoomDevice | null>(null);
  const [deviceForm] = Form.useForm();

  // PageTable render-prop 上下文提升到业务回调（提交/删除/导入后刷新 + 导出筛选 + 清筛选）
  const reloadRef = useRef<() => void>(() => {});
  const searchFormRef = useRef<FormInstance | null>(null);

  // 使用 useSidebarDeptFilter hook 管理部门筛选（清字段在切换入口手动做，见下）
  const { selectedDeptId, handleDeptSelect } = useSidebarDeptFilter();

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

  // 列表请求：查询条件（current/pageSize/filters/排序，含部门 orgId）由 PageTable 组装后透传
  const request = useCallback(async (params: Record<string, unknown>) => {
    const result = await roomDeviceApi.list(params);
    const list = result.data?.list || [];
    // 卡片视图快照：与表格同一份数据源
    setCardDevices(list);
    return { list, total: result.data?.total || 0 };
  }, []);

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

  // 统计卡片刷新（列表刷新由 PageTable 内部 loadData 承担）
  const refreshStatistics = useCallback(() => {
    void loadStatistics().then(setStatistics);
  }, [loadStatistics]);

  // 统一的数据刷新函数：列表 + 统计
  const refreshData = useCallback(() => {
    reloadRef.current();
    refreshStatistics();
  }, [refreshStatistics]);

  // 初始化加载（列表首载由 PageTable 内建；此处只载统计）
  useEffect(() => {
    void loadStatistics().then(setStatistics);
  }, [loadStatistics]);

  // 部门变化：重载机房选项（列表由 extraFilters 联动回第 1 页重载）
  useEffect(() => {
    void loadRoomOptions(selectedDeptId || undefined);
  }, [selectedDeptId, loadRoomOptions]);

  const closeModal = () => {
    setModalVisible(false);
    deviceForm.resetFields();
    setEditingDevice(null);
  };

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
      closeModal();
      refreshData();
    } catch (error: unknown) {
      if (isFormValidationError(error)) return;
      handleApiError(error, "操作");
    }
  };

  // 单删（经 createDeleteAction → ConfirmDelete；失败由 confirmDelete 统一反馈后 rethrow）
  const handleDelete = async (id: string) => {
    await roomDeviceApi.delete(id);
    refreshData();
  };

  // 批删（确认/成功提示由 PageTable → BatchDeleteButton → confirmDelete 承担；
  // 成功后 PageTable 自动清空选中并重载列表，这里只补统计同步）
  const handleBatchDelete = async (keys: Key[]) => {
    await roomDeviceApi.batch("delete", { ids: keys });
    refreshStatistics();
  };

  const openModal = (record?: RoomDevice) => {
    if (record) {
      setEditingDevice(record);
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
      setModalVisible(true);
    } else {
      setEditingDevice(null);
      deviceForm.resetFields();
      deviceForm.setFieldsValue({ status: 0, deviceType: "server" });
      setModalVisible(true);
    }
  };

  const handleImportSuccess = () => {
    refreshData();
    setImportVisible(false);
  };

  // 导出：沿用迁移前口径——取当前搜索表单非空值作为 filters
  const handleOpenExport = () => {
    const values = (searchFormRef.current?.getFieldsValue() ?? {}) as Record<string, unknown>;
    const currentFilters: Record<string, unknown> = {};
    Object.keys(values).forEach((key) => {
      const value = values[key];
      if (value !== undefined && value !== null && value !== "") {
        currentFilters[key] = value;
      }
    });
    setExportFilters(currentFilters);
    setExportVisible(true);
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
    return ROOM_DEVICE_STATUS_MAP[status] || "未知";
  };

  const roomSelectOptions = useMemo(
    () => roomOptions.map((r) => ({ label: r.name, value: r.id })),
    [roomOptions]
  );

  // 搜索区 schema：字段名与后端查询参数一一对应（roomId/name/deviceType/status）；
  // roomId 依赖侧边栏部门态（disabled/placeholder 动态），走 custom 逃生口。
  const searchFields = useMemo<SearchField[]>(
    () => [
      {
        type: "custom",
        name: "roomId",
        render: (value, onChange) => (
          <Form.Item label="所属机房" style={{ marginBottom: 0 }}>
            <AppSelect
              value={(value as string | undefined) ?? undefined}
              onChange={(v) => onChange(v)}
              placeholder={selectedDeptId ? "请选择机房" : "请先选择部门"}
              disabled={!selectedDeptId}
              showSearch
              style={{ width: 150 }}
              options={roomSelectOptions}
            />
          </Form.Item>
        ),
      },
      { type: "input", name: "name", label: "设备名称", placeholder: "请输入设备名称" },
      {
        type: "select",
        name: "deviceType",
        label: "设备类型",
        placeholder: "请选择类型",
        options: DEVICE_TYPE_OPTIONS,
      },
      {
        type: "select",
        name: "status",
        label: "状态",
        placeholder: "请选择状态",
        options: ROOM_DEVICE_STATUS_OPTIONS,
      },
    ],
    [roomSelectOptions, selectedDeptId]
  );

  // 部门筛选联动：引用变化 → PageTable applyFilters 回第 1 页重载
  const extraFilters = useMemo(
    () => (selectedDeptId ? { orgId: selectedDeptId } : {}),
    [selectedDeptId]
  );

  // 部门切换入口：迁移前 effect 在部门变化时清空搜索字段（roomId/name/deviceType/status），
  // 这里在触发联动前清空，避免旧筛选被 applyFilters 合并进 filters
  const handleDeptSelectAndReset = useCallback(
    (selectedKeys: Key[], info: Parameters<typeof handleDeptSelect>[1]) => {
      searchFormRef.current?.resetFields();
      handleDeptSelect(selectedKeys, info);
    },
    [handleDeptSelect]
  );

  const columns: ColumnsType<RoomDevice> = [
    {
      title: "设备名称",
      dataIndex: "name",
      key: "name",
      width: 150,
      sorter: true,
    },
    {
      title: "设备编码",
      dataIndex: "deviceCode",
      key: "deviceCode",
      width: 150,
      sorter: true,
    },
    {
      title: "所属机房",
      dataIndex: "roomName",
      key: "roomName",
      width: 120,
      sorter: true,
      render: (roomName) => roomName || "-",
    },
    {
      title: "设备类型",
      dataIndex: "deviceType",
      key: "deviceType",
      width: 100,
      sorter: true,
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
    }),
  ];

  // 行操作：编辑 + 删除（数量/顺序与迁移前一致；删除走 createDeleteAction 唯一入口）
  const rowActions = (record: RoomDevice) => [
    { key: "edit", label: "编辑", icon: <EditOutlined />, onClick: () => openModal(record) },
    createDeleteAction(record.id, handleDelete, { target: `机房设备 ${record.name}` }),
  ];

  const renderCardView = () => {
    if (cardDevices.length === 0)
      return (
        <div
          style={{
            textAlign: "center",
            padding: "40px",
            color: "var(--theme-text-tertiary)",
          }}
        >
          暂无数据
        </div>
      );
    return (
      <Row gutter={[16, 16]}>
        {cardDevices.map((device) => (
          <Col key={device.id} xs={24} sm={12} md={8} lg={6}>
            <Card
              hoverable
              actions={[
                <EditOutlined key="edit" onClick={() => openModal(device)} />,
                <DeleteOutlined
                  key="delete"
                  style={{ color: "var(--theme-error)" }}
                  onClick={() => {
                    confirmDelete({
                      target: `机房设备 ${device.name}`,
                      onDelete: () => handleDelete(device.id),
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

  // 业务区：统计卡片（total>10 才显示）+ 已选择 Alert（PageTable render-prop 顶部）
  const renderBusinessArea = ({
    selectedRowKeys,
    clearSelection,
    total,
    loadData,
    searchForm,
  }: PageTableRenderContext<RoomDevice>) => {
    // 提升到业务回调：弹窗提交/删除/导入成功后刷新列表；导出取当前搜索条件
    reloadRef.current = loadData;
    searchFormRef.current = searchForm;
    return (
      <>
        <StatisticsCards
          show={total > 10}
          items={[
            { title: "总设备数", value: statistics.total, prefix: <AppstoreAddOutlined /> },
            {
              title: "正常",
              value: statistics.normal,
              styles: { content: { color: "var(--theme-success)" } },
              prefix: <CheckCircleOutlined />,
            },
            {
              title: "故障",
              value: statistics.fault,
              styles: { content: { color: "var(--theme-error)" } },
              prefix: <WarningOutlined />,
            },
            {
              title: "报废",
              value: statistics.scrapped,
              styles: { content: { color: "var(--theme-text-tertiary)" } },
              prefix: <StopOutlined />,
            },
          ]}
        />
        {selectedRowKeys.length > 0 && (
          <Alert
            title={
              <span>
                已选择 <strong>{selectedRowKeys.length}</strong> 个设备，
                <Button type="link" size="small" onClick={clearSelection} style={{ padding: 0 }}>
                  取消选择
                </Button>
              </span>
            }
            type="info"
            showIcon
            style={{ marginBottom: 16 }}
          />
        )}
      </>
    );
  };

  return (
    <Layout style={{ background: "var(--theme-bg-secondary)", minHeight: "calc(100vh - 64px)" }}>
      <DeptSidebar selectedDeptId={selectedDeptId} onSelect={handleDeptSelectAndReset} />
      <Content style={{ background: "var(--theme-bg-surface)", padding: 16 }}>
        <PageTable<RoomDevice>
          request={request}
          columns={columns}
          rowKey={(record) => record.id}
          searchFields={searchFields}
          sorterMetas={sorterMetas}
          extraFilters={extraFilters}
          rowActions={rowActions}
          batchDelete={{ onDelete: handleBatchDelete, target: "机房设备" }}
          toolbar={
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
              <Button icon={<ExportOutlined />} onClick={handleOpenExport}>
                导出
              </Button>
              <Button type="primary" icon={<PlusOutlined />} onClick={() => openModal()}>
                新增设备
              </Button>
            </Space>
          }
          // 卡片模式：表格由 PageTable 承载但隐藏（搜索/排序/分页/批删/三态保持同一数据层）
          style={viewMode === "card" ? { display: "none" } : undefined}
        >
          {renderBusinessArea}
        </PageTable>
        {viewMode === "card" && <Card style={{ marginTop: 16 }}>{renderCardView()}</Card>}
        <Modal
          title={editingDevice ? "编辑设备" : "新增设备"}
          open={modalVisible}
          onOk={async () => {
            setSubmitting(true);
            try {
              await handleSave();
            } finally {
              setSubmitting(false);
            }
          }}
          cancelButtonProps={{ disabled: submitting }}
          onCancel={closeModal}
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
              {/* 形态分类器①：已是 options 形态；机房 UUID 字段，allowClear 原已有 */}
              <AppSelect
                allowClear
                placeholder="请选择所属机房"
                showSearch
                filterOption={(input, option) =>
                  String(option?.label ?? "")
                    .toLowerCase()
                    .includes(input.toLowerCase())
                }
                onSearch={() => {}}
                options={roomSelectOptions}
              />
            </Form.Item>
            <Form.Item
              name="name"
              label="设备名称"
              rules={[{ required: true, message: "请输入设备名称" }]}
            >
              <AppInput placeholder="请输入设备名称" />
            </Form.Item>
            <Form.Item
              name="deviceCode"
              label="设备编码"
              rules={[{ required: true, message: "请输入设备编码" }]}
            >
              <AppInput placeholder="请输入设备编码" />
            </Form.Item>
            <Form.Item
              name="deviceType"
              label="设备类型"
              rules={[{ required: true, message: "请选择设备类型" }]}
            >
              {/* 形态分类器②④：固定枚举 children → options；业务禁搜 */}
              <AppSelect
                allowClear
                placeholder="请选择设备类型"
                showSearch={false}
                onSearch={() => {}}
                options={ROOM_DEVICE_TYPE_OPTIONS}
              />
            </Form.Item>
            <Row gutter={16}>
              <Col span={12}>
                <Form.Item name="vendor" label="厂商">
                  <AppInput placeholder="请输入厂商" />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="model" label="型号">
                  <AppInput placeholder="请输入型号" />
                </Form.Item>
              </Col>
            </Row>
            <Row gutter={16}>
              <Col span={12}>
                <Form.Item name="serialNumber" label="序列号">
                  <AppInput placeholder="请输入序列号" />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="positionDesc" label="位置描述">
                  <AppInput placeholder="请输入位置描述" />
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
                  {/* 形态分类器②④：状态枚举（业务禁搜），children → options */}
                  <AppSelect
                    allowClear
                    placeholder="请选择状态"
                    showSearch={false}
                    onSearch={() => {}}
                    options={ROOM_DEVICE_STATUS_OPTIONS}
                  />
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
              <AppTextArea rows={3} placeholder="请输入描述" />
            </Form.Item>
          </Form>
        </Modal>
        <ExcelImportLazy
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
