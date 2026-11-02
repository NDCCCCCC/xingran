/**
 * 专线管理页面 — Phase 68.1 T6a：operations 域样板页（PageTable 全契约形态）
 *
 * 迁移自页面级双 hook 数据层实现：数据层 / 搜索区 / 分页 /
 * 三态 / 删除确认全部收敛到 PageTable（UI-SPEC §3.2），本页只保留业务块
 * （统计卡片、已选择 Alert、视图切换、导入导出、新增/编辑弹窗表单）。
 *
 * 视图切换（表格 / 卡片）：卡片数据取自同一 request 的最近一次快照；表格仍由
 * PageTable 承载、在卡片模式下隐藏——搜索 / 排序 / 分页 / 批删 / 三态与表格视图
 * 共用同一数据层（PageTableRenderContext 不暴露 data，故镜像一份快照）。
 */

import { useState, useCallback, useEffect, useMemo, useRef } from "react";
import type { FC, Key, CSSProperties } from "react";
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
  LineChartOutlined,
} from "@ant-design/icons";
import type { DedicatedLine } from "@/types";
import { dedicatedLineApi, serverRoomApi } from "@/lib/opsApi";
import { useDict } from "@/hooks/useDict";
import { handleApiError, handleSuccess, isFormValidationError } from "@/utils/errorHandler";
import { createDateTimeColumn, createSorterMeta } from "@/utils/tableHelpers";
import { PageTable, useActionPermission } from "@/components/shared";
import type { PageTableRenderContext, SearchField } from "@/components/shared";
import { PERMISSION_CODES } from "@/constants/permissionCodes";
import { createDeleteAction } from "@/components/shared/ActionButtons";
import ExcelImportLazy from "@/components/shared/ExcelImportLazy";
import ExcelExport from "@/components/shared/ExcelExport";
import { confirmDelete } from "@/components/shared/confirmDelete";
import { StatisticsCards } from "@/components/operations/StatisticsCards";
// Phase 71 (W3)：裸 Input/Select/TextArea → 契约组件。
// - sourceRoomId/destRoomId：UUID 字段，原 allowClear + showSearch + 自定义
//   filterOption（按 label 子串），已改写为 options 形态，AppSelect 的
//   optionFilterProp="label" 与之一致；自定义 filterOption 保留（行为不变）
// - lineType/isp/status：字典/枚举 children → options；枚举类显式 showSearch={false}
import { AppInput, AppSelect, AppTextArea } from "@/components/ui";

const { Content } = Layout;

/**
 * 卡片视图图标动作的禁用态样式（Phase 86 W2，与 buildings/room-devices 同口径）。
 * antd `Card.actions` 只接受 ReactNode → 未就绪时用「不响应指针 + 降透明度」
 * 表达禁用态，并配合 `title={decision.hint}` 提示原因。
 */
const cardActionStyle = (disabled: boolean): CSSProperties => ({
  opacity: disabled ? 0.45 : 1,
  pointerEvents: disabled ? "none" : "auto",
  cursor: disabled ? "not-allowed" : "pointer",
});

// 模块级静态映射（Phase m76-js-misc 6.3）
const DEDICATED_LINE_STATUS_MAP: Record<number, string> = { 0: "正常", 1: "故障", 2: "停用" };

// 搜索区状态选项（值沿用后端 status 口径 0/1/2）
const DEDICATED_LINE_STATUS_OPTIONS = [
  { label: "正常", value: 0 },
  { label: "故障", value: 1 },
  { label: "停用", value: 2 },
];

interface ServerRoomOption {
  id: string;
  name: string;
  orgId: string;
}

type ViewMode = "table" | "card";

interface Statistics {
  total: number;
  normal: number;
  fault: number;
  disabled: number;
}

const DedicatedLineManagement: FC = () => {
  // Phase 86 W2（RBAC-02）：操作级权限门控（三态）。ops 码组只有 delete 键 → 删除用 .delete。
  const permGate = useActionPermission();
  const permEdit = permGate.decide(PERMISSION_CODES.opsDedicatedLine.edit);
  const permDelete = permGate.decide(PERMISSION_CODES.opsDedicatedLine.delete);
  const permAdd = permGate.decide(PERMISSION_CODES.opsDedicatedLine.add);
  const permImport = permGate.decide(PERMISSION_CODES.opsDedicatedLine.import);
  const permExport = permGate.decide(PERMISSION_CODES.opsDedicatedLine.export);

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
  const [serverRooms, setServerRooms] = useState<ServerRoomOption[]>([]);
  const [statistics, setStatistics] = useState<Statistics>({
    total: 0,
    normal: 0,
    fault: 0,
    disabled: 0,
  });

  // 卡片视图数据快照（与表格同一 request，不另建数据层）
  const [cardLines, setCardLines] = useState<DedicatedLine[]>([]);

  // 新增/编辑弹窗态（原表格管理器 hook 的 editForm/editModalVisible/editingItem 位）
  const [modalVisible, setModalVisible] = useState(false);
  const [editingLine, setEditingLine] = useState<DedicatedLine | null>(null);
  const [lineForm] = Form.useForm();

  // PageTable render-prop 上下文提升到业务回调（提交/删除/导入后刷新 + 导出筛选）
  const reloadRef = useRef<() => void>(() => {});
  const searchFormRef = useRef<FormInstance | null>(null);

  // Wave 5: useDict replaces raw post() calls for both line type + ISP dicts
  const { data: lineTypeDict = [] } = useDict("ops_dedicated_line_type");
  const { data: ispDict = [] } = useDict("ops_isp");

  // 加载所有机房列表（不分部门）
  const loadServerRooms = useCallback(async () => {
    try {
      const result = (await serverRoomApi.list({ current: 1, pageSize: 50 })) as {
        data?: { list: Record<string, unknown>[] };
      };
      const rooms = result.data?.list || [];
      setServerRooms(
        rooms.map((r: Record<string, unknown>) => ({
          id: r.id as string,
          name: r.name as string,
          orgId: r.orgId as string,
        }))
      );
    } catch (error) {
      handleApiError(error, "加载机房列表", false);
    }
  }, []);

  // 服务端排序:field 必须与 columns 的 dataIndex 一致(排序元数据按 field 映射后端参数)
  const sorterMetas = useMemo(
    () => [
      createSorterMeta<DedicatedLine>("name"),
      createSorterMeta<DedicatedLine>("bandwidth"),
      createSorterMeta<DedicatedLine>("status"),
      createSorterMeta<DedicatedLine>("createdAt", "date"),
    ],
    []
  );

  // 列表请求：查询条件（current/pageSize/filters/排序）由 PageTable 内部组装后透传
  const request = useCallback(async (params: Record<string, unknown>) => {
    const result = (await dedicatedLineApi.list(params)) as {
      data?: { list: DedicatedLine[]; total: number };
    };
    const list = result.data?.list || [];
    // 卡片视图快照：与表格同一份数据源
    setCardLines(list);
    return { list, total: result.data?.total || 0 };
  }, []);

  const loadStatistics = useCallback(async (): Promise<Statistics> => {
    try {
      const stats = await dedicatedLineApi.statistics();
      return {
        total: stats.total ?? 0,
        normal: stats.normal ?? 0,
        fault: stats.fault ?? 0,
        disabled: stats.disabled ?? 0,
      };
    } catch (error) {
      handleApiError(error, "加载统计数据", false);
      return { total: 0, normal: 0, fault: 0, disabled: 0 };
    }
  }, []);

  // 统计卡片刷新（列表刷新由 PageTable 内部 loadData 承担）
  const refreshStatistics = useCallback(() => {
    void loadStatistics().then(setStatistics);
  }, [loadStatistics]);

  // 统一的数据刷新函数：列表 + 统计
  const refreshData = () => {
    reloadRef.current();
    refreshStatistics();
  };

  // 初始化加载（列表首载由 PageTable 内建；此处只载机房选项 + 统计）
  useEffect(() => {
    Promise.all([loadServerRooms(), loadStatistics()]).then(([, stats]) => setStatistics(stats));
  }, [loadServerRooms, loadStatistics]);

  const closeModal = () => {
    setModalVisible(false);
    lineForm.resetFields();
    setEditingLine(null);
  };

  const handleSave = async () => {
    try {
      const values = (await lineForm.validateFields()) as Partial<DedicatedLine>;
      if (editingLine) {
        await dedicatedLineApi.update(editingLine.id, values);
        handleSuccess("更新");
      } else {
        await dedicatedLineApi.create(values);
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
    await dedicatedLineApi.delete(id);
    refreshData();
  };

  // 批删（确认/成功提示由 PageTable → BatchDeleteButton → confirmDelete 承担；
  // 成功后 PageTable 自动清空选中并重载列表，这里只补统计同步）
  const handleBatchDelete = async (keys: Key[]) => {
    await dedicatedLineApi.batch("delete", { ids: keys });
    refreshStatistics();
  };

  const openModal = (record?: DedicatedLine) => {
    if (record) {
      setEditingLine(record);
      lineForm.setFieldsValue(record);
      setModalVisible(true);
    } else {
      setEditingLine(null);
      lineForm.resetFields();
      const defaultType =
        lineTypeDict.find((d) => d.isDefault)?.dictValue || lineTypeDict[0]?.dictValue;
      const defaultIsp = ispDict.find((d) => d.isDefault)?.dictValue || ispDict[0]?.dictValue;
      lineForm.setFieldsValue({ status: 0, lineType: defaultType, isp: defaultIsp });
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

  const getLineTypeText = (type: string) => {
    const dictItem = lineTypeDict.find((d) => d.dictValue === type);
    return dictItem?.dictLabel || type;
  };

  const getIspText = (isp: string) => {
    const dictItem = ispDict.find((d) => d.dictValue === isp);
    return dictItem?.dictLabel || isp;
  };

  const getStatusText = (status: number) => {
    return DEDICATED_LINE_STATUS_MAP[status] || "未知";
  };

  const roomOptions = useMemo(
    () => serverRooms.map((room) => ({ label: room.name, value: room.id })),
    [serverRooms]
  );

  // 搜索区 schema：字段名与后端查询参数一一对应（name/sourceRoomId/destRoomId/lineType/isp/status）
  const searchFields = useMemo<SearchField[]>(
    () => [
      { name: "name", label: "专线名称", type: "input", placeholder: "请输入专线名称" },
      {
        name: "sourceRoomId",
        label: "源机房",
        type: "select",
        placeholder: "请选择源机房",
        options: roomOptions,
      },
      {
        name: "destRoomId",
        label: "目的机房",
        type: "select",
        placeholder: "请选择目的机房",
        options: roomOptions,
      },
      {
        name: "lineType",
        label: "专线类型",
        type: "select",
        placeholder: "请选择类型",
        options: lineTypeDict.map((d) => ({ label: d.dictLabel, value: d.dictValue })),
      },
      {
        name: "isp",
        label: "运营商",
        type: "select",
        placeholder: "请选择运营商",
        options: ispDict.map((d) => ({ label: d.dictLabel, value: d.dictValue })),
      },
      {
        name: "status",
        label: "状态",
        type: "select",
        placeholder: "请选择状态",
        options: DEDICATED_LINE_STATUS_OPTIONS,
      },
    ],
    [roomOptions, lineTypeDict, ispDict]
  );

  const columns: ColumnsType<DedicatedLine> = [
    {
      title: "专线名称",
      dataIndex: "name",
      key: "name",
      width: 150,
      sorter: true,
    },
    {
      title: "专线类型",
      dataIndex: "lineType",
      key: "lineType",
      width: 120,
      render: (type) => <Tag>{getLineTypeText(type as string)}</Tag>,
    },
    {
      title: "带宽",
      dataIndex: "bandwidth",
      key: "bandwidth",
      width: 100,
      sorter: true,
      render: (v) => v || "-",
    },
    {
      title: "运营商",
      dataIndex: "isp",
      key: "isp",
      width: 120,
      render: (isp) => <Tag>{getIspText(isp as string)}</Tag>,
    },
    {
      title: "源机房",
      dataIndex: "sourceRoomName",
      key: "sourceRoomName",
      width: 120,
      render: (v) => v || "-",
    },
    {
      title: "源设备",
      dataIndex: "sourceDeviceName",
      key: "sourceDeviceName",
      width: 120,
      render: (v) => v || "-",
    },
    {
      title: "源端口",
      dataIndex: "sourcePort",
      key: "sourcePort",
      width: 100,
      render: (v) => v || "-",
    },
    {
      title: "源IP",
      dataIndex: "sourceIpAddress",
      key: "sourceIpAddress",
      width: 130,
      render: (v) => v || "-",
    },
    {
      title: "目的机房",
      dataIndex: "destRoomName",
      key: "destRoomName",
      width: 120,
      render: (v) => v || "-",
    },
    {
      title: "目的设备",
      dataIndex: "destDeviceName",
      key: "destDeviceName",
      width: 120,
      render: (v) => v || "-",
    },
    {
      title: "目的端口",
      dataIndex: "destPort",
      key: "destPort",
      width: 100,
      render: (v) => v || "-",
    },
    {
      title: "目的IP",
      dataIndex: "destIpAddress",
      key: "destIpAddress",
      width: 130,
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
    {
      title: "月费(元)",
      dataIndex: "monthlyFee",
      key: "monthlyFee",
      width: 100,
      render: (v) => (v != null ? `¥${v}` : "-"),
    },
    createDateTimeColumn("createdAt", {
      width: 180,
      sorter: true,
    }),
  ];

  // 行操作：编辑 + 删除（数量/顺序与迁移前一致；删除走 createDeleteAction 唯一入口）
  // permission 由 ActionButtons 内建三态门控
  const rowActions = (record: DedicatedLine) => [
    {
      key: "edit",
      label: "编辑",
      icon: <EditOutlined />,
      onClick: () => openModal(record),
      permission: PERMISSION_CODES.opsDedicatedLine.edit,
    },
    createDeleteAction(record.id, handleDelete, {
      target: `专线 ${record.name}`,
      permission: PERMISSION_CODES.opsDedicatedLine.delete,
    }),
  ];

  const renderCardView = () => {
    if (cardLines.length === 0)
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
        {cardLines.map((line) => (
          <Col key={line.id} xs={24} sm={12} md={8} lg={6}>
            <Card
              hoverable
              actions={[
                // 卡片视图与表格视图是同一页的两种形态 → 共用同一组门控决策
                ...(permEdit.visible
                  ? [
                      <EditOutlined
                        key="edit"
                        title={permEdit.hint}
                        style={cardActionStyle(permEdit.disabled)}
                        onClick={() => {
                          if (permEdit.disabled) return;
                          openModal(line);
                        }}
                      />,
                    ]
                  : []),
                ...(permDelete.visible
                  ? [
                      <DeleteOutlined
                        key="delete"
                        title={permDelete.hint}
                        style={{
                          ...cardActionStyle(permDelete.disabled),
                          color: permDelete.disabled
                            ? "var(--theme-text-tertiary)"
                            : "var(--theme-error)",
                        }}
                        onClick={() => {
                          if (permDelete.disabled) return;
                          confirmDelete({
                            target: `专线 ${line.name}`,
                            onDelete: () => handleDelete(line.id),
                          });
                        }}
                      />,
                    ]
                  : []),
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
                    <span>{line.name}</span>
                    <Tag
                      color={
                        line.status === 0 ? "success" : line.status === 1 ? "error" : "default"
                      }
                    >
                      {getStatusText(line.status)}
                    </Tag>
                  </div>
                }
                description={
                  <div>
                    <div>
                      <strong>类型：</strong>
                      {getLineTypeText(line.lineType)}
                    </div>
                    <div>
                      <strong>运营商：</strong>
                      {getIspText(line.isp)}
                    </div>
                    {line.bandwidth && (
                      <div>
                        <strong>带宽：</strong>
                        {line.bandwidth}
                      </div>
                    )}
                    {line.sourceIpAddress && (
                      <div>
                        <strong>源IP：</strong>
                        {line.sourceIpAddress}
                      </div>
                    )}
                    {line.destIpAddress && (
                      <div>
                        <strong>目的IP：</strong>
                        {line.destIpAddress}
                      </div>
                    )}
                    {line.monthlyFee != null && (
                      <div>
                        <strong>月费：</strong>¥{line.monthlyFee}
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

  // 业务区：统计卡片（total>10 才显示）+ 已选择 Alert（PageTable render-prop 顶部>
  const renderBusinessArea = ({
    selectedRowKeys,
    clearSelection,
    total,
    loadData,
    searchForm,
  }: PageTableRenderContext<DedicatedLine>) => {
    // 提升到业务回调：弹窗提交/删除/导入成功后刷新列表；导出取当前搜索条件
    reloadRef.current = loadData;
    searchFormRef.current = searchForm;
    return (
      <>
        <StatisticsCards
          show={total > 10}
          items={[
            { title: "总专线数", value: statistics.total, prefix: <LineChartOutlined /> },
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
              title: "停用",
              value: statistics.disabled,
              styles: { content: { color: "var(--theme-text-tertiary)" } },
              prefix: <StopOutlined />,
            },
          ]}
        />
        {selectedRowKeys.length > 0 && (
          <Alert
            title={
              <span>
                已选择 <strong>{selectedRowKeys.length}</strong> 条专线，
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
      <Content style={{ background: "var(--theme-bg-surface)", padding: 16 }}>
        <PageTable<DedicatedLine>
          request={request}
          columns={columns}
          rowKey={(record) => record.id}
          searchFields={searchFields}
          sorterMetas={sorterMetas}
          rowActions={rowActions}
          batchDelete={
            permDelete.visible ? { onDelete: handleBatchDelete, target: "专线" } : undefined
          }
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
              {/* Excel 门控点 = 触发按钮（modal 容器保留无条件渲染） */}
              {permImport.visible && (
                <Button
                  icon={<ImportOutlined />}
                  onClick={() => setImportVisible(true)}
                  disabled={permImport.disabled}
                  title={permImport.hint}
                >
                  导入
                </Button>
              )}
              {permExport.visible && (
                <Button
                  icon={<ExportOutlined />}
                  onClick={handleOpenExport}
                  disabled={permExport.disabled}
                  title={permExport.hint}
                >
                  导出
                </Button>
              )}
              {permAdd.visible && (
                <Button
                  type="primary"
                  icon={<PlusOutlined />}
                  onClick={() => openModal()}
                  disabled={permAdd.disabled}
                  title={permAdd.hint}
                >
                  新增专线
                </Button>
              )}
            </Space>
          }
          // 卡片模式：表格由 PageTable 承载但隐藏（搜索/排序/分页/批删/三态保持同一数据层）
          style={viewMode === "card" ? { display: "none" } : undefined}
        >
          {renderBusinessArea}
        </PageTable>
        {viewMode === "card" && <Card style={{ marginTop: 16 }}>{renderCardView()}</Card>}
        <Modal
          title={editingLine ? "编辑专线" : "新增专线"}
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
            form={lineForm}
            layout="horizontal"
            labelCol={{ span: 5 }}
            wrapperCol={{ span: 19 }}
          >
            <Form.Item
              name="name"
              label="专线名称"
              rules={[{ required: true, message: "请输入专线名称" }]}
            >
              <AppInput placeholder="请输入专线名称" />
            </Form.Item>
            <Form.Item
              name="lineType"
              label="专线类型"
              rules={[{ required: true, message: "请选择专线类型" }]}
            >
              {/* 形态分类器②④：字典 children → options；业务禁搜 */}
              <AppSelect
                allowClear
                placeholder="请选择专线类型"
                showSearch={false}
                onSearch={() => {}}
                options={lineTypeDict.map((d) => ({ value: d.dictValue, label: d.dictLabel }))}
              />
            </Form.Item>
            <Row gutter={16}>
              <Col span={12}>
                <Form.Item name="bandwidth" label="带宽">
                  <AppInput placeholder="如：100M, 1G" className="user-form-input" />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item
                  name="isp"
                  label="运营商"
                  rules={[{ required: true, message: "请选择运营商" }]}
                >
                  {/* 形态分类器②④：字典 children → options；业务禁搜 */}
                  <AppSelect
                    allowClear
                    placeholder="请选择运营商"
                    className="user-form-input"
                    showSearch={false}
                    onSearch={() => {}}
                    options={ispDict.map((d) => ({ value: d.dictValue, label: d.dictLabel }))}
                  />
                </Form.Item>
              </Col>
            </Row>
            <Row gutter={16}>
              <Col span={12}>
                <Form.Item name="sourceRoomId" label="源机房">
                  {/* 形态分类器①：已是 options 形态；房号 UUID 字段，allowClear 原已有 */}
                  <AppSelect
                    placeholder="请选择源机房"
                    allowClear
                    showSearch
                    filterOption={(input, option) =>
                      String(option?.label ?? "")
                        .toLowerCase()
                        .includes(input.toLowerCase())
                    }
                    onSearch={() => {}}
                    className="user-form-input"
                    options={roomOptions}
                  />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="sourceDeviceName" label="源设备名称">
                  <AppInput placeholder="请输入源设备名称" className="user-form-input" />
                </Form.Item>
              </Col>
            </Row>
            <Form.Item name="sourcePort" label="源端口">
              <AppInput
                placeholder="请输入源端口"
                className="user-form-input"
                style={{ width: "50%" }}
              />
            </Form.Item>
            <Row gutter={16}>
              <Col span={12}>
                <Form.Item name="destRoomId" label="目的机房">
                  {/* 形态分类器①：已是 options 形态；房号 UUID 字段，allowClear 原已有 */}
                  <AppSelect
                    placeholder="请选择目的机房"
                    allowClear
                    showSearch
                    filterOption={(input, option) =>
                      String(option?.label ?? "")
                        .toLowerCase()
                        .includes(input.toLowerCase())
                    }
                    onSearch={() => {}}
                    className="user-form-input"
                    options={roomOptions}
                  />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="destDeviceName" label="目的设备名称">
                  <AppInput placeholder="请输入目的设备名称" className="user-form-input" />
                </Form.Item>
              </Col>
            </Row>
            <Form.Item name="destPort" label="目的端口">
              <AppInput
                placeholder="请输入目的端口"
                className="user-form-input"
                style={{ width: "50%" }}
              />
            </Form.Item>
            <Row gutter={16}>
              <Col span={12}>
                <Form.Item name="sourceIpAddress" label="源IP地址">
                  <AppInput placeholder="请输入源IP地址" className="user-form-input" />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="sourceSubnetMask" label="源子网掩码">
                  <AppInput placeholder="请输入源子网掩码" className="user-form-input" />
                </Form.Item>
              </Col>
            </Row>
            <Row gutter={16}>
              <Col span={12}>
                <Form.Item name="destIpAddress" label="目的IP地址">
                  <AppInput placeholder="请输入目的IP地址" className="user-form-input" />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="destSubnetMask" label="目的子网掩码">
                  <AppInput placeholder="请输入目的子网掩码" className="user-form-input" />
                </Form.Item>
              </Col>
            </Row>
            <Row gutter={16}>
              <Col span={12}>
                <Form.Item name="vlan" label="VLAN">
                  <AppInput placeholder="请输入VLAN" className="user-form-input" />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="carrierContactName" label="运营商联系人">
                  <AppInput placeholder="请输入联系人" className="user-form-input" />
                </Form.Item>
              </Col>
            </Row>
            <Row gutter={16}>
              <Col span={12}>
                <Form.Item name="carrierContactPhone" label="联系人电话">
                  <AppInput placeholder="请输入联系电话" className="user-form-input" />
                </Form.Item>
              </Col>
            </Row>
            <Row gutter={16}>
              <Col span={12}>
                <Form.Item name="monthlyFee" label="月租费用">
                  <InputNumber
                    min={0}
                    precision={2}
                    placeholder="请输入月租费用"
                    style={{ width: "100%" }}
                    className="user-form-input"
                  />
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
                    className="user-form-input"
                    showSearch={false}
                    onSearch={() => {}}
                    options={DEDICATED_LINE_STATUS_OPTIONS}
                  />
                </Form.Item>
              </Col>
            </Row>
            <Form.Item name="remark" label="备注">
              <AppTextArea rows={3} placeholder="请输入备注" className="user-form-input" />
            </Form.Item>
          </Form>
        </Modal>
        <ExcelImportLazy
          entityType="dedicatedLine"
          entityName="专线"
          visible={importVisible}
          onClose={() => setImportVisible(false)}
          onImportSuccess={handleImportSuccess}
        />
        <ExcelExport
          entityType="dedicatedLine"
          entityName="专线"
          visible={exportVisible}
          onClose={() => setExportVisible(false)}
          filters={exportFilters}
        />
      </Content>
    </Layout>
  );
};

export default DedicatedLineManagement;
