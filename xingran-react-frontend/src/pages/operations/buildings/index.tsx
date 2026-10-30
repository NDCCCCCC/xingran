/**
 * 楼宇管理页面 — Phase 68.1 T6b：operations 域 PageTable 契约形态
 *
 * 迁移自页面级双 hook 数据层实现：数据层 / 搜索区 / 分页 / 三态 / 删除确认
 * 全部收敛到 PageTable（UI-SPEC §3.2），本页只保留
 * 业务块（左侧部门树、统计卡片、已选择 Alert、视图切换、导入导出、新增/编辑弹窗表单）。
 *
 * 部门树联动：选中部门经 PageTable extraFilters（{ orgId }）触发回第 1 页重载；
 * request 侧再兜底注入 orgId，保证搜索 / 重置 / 翻页 / 排序等任何重载路径都不丢部门过滤。
 * 统计口径与列表共用同一次 request 的过滤条件（分页/排序参数剥离后传 statistics）。
 *
 * 视图切换（表格 / 卡片）：卡片数据取自同一 request 的最近一次快照；表格仍由
 * PageTable 承载、在卡片模式下隐藏——搜索 / 排序 / 分页 / 批删 / 三态与表格视图
 * 共用同一数据层（PageTableRenderContext 不暴露 data，故镜像一份快照）。
 */

import { useState, useCallback, useRef, useMemo, type FC, type Key, type CSSProperties } from "react";
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
  Statistic,
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
  AppstoreOutlined,
  TableOutlined,
  ImportOutlined,
  ExportOutlined,
} from "@ant-design/icons";

const { Sider, Content } = Layout;
import type { Building } from "@/types/operations";
import { buildingApi } from "@/lib/opsApi";
import { handleApiError, handleSuccess, isFormValidationError } from "@/utils/errorHandler";
import { createStatusColumn, createDateTimeColumn, createSorterMeta } from "@/utils/tableHelpers";
import { PageTable, useActionPermission } from "@/components/shared";
import type { PageTableRenderContext, SearchField } from "@/components/shared";
import { PERMISSION_CODES } from "@/constants/permissionCodes";
import { createDeleteAction } from "@/components/shared/ActionButtons";
import ExcelImportLazy from "@/components/shared/ExcelImportLazy";
import ExcelExport from "@/components/shared/ExcelExport";
import { confirmDelete } from "@/components/shared/confirmDelete";
import { DepartmentTreeSelect } from "@/components/shared";
import DeptTree from "@/components/DeptTree";
import { useDeptTree } from "@/hooks/useDeptTree";
import { findDeptNode } from "@/utils/deptUtils";
// Phase 71 (W3)：裸 Input/Select/TextArea → 契约组件。
// 状态枚举为 children 形态 → 改写 options 形态并显式 showSearch={false}（业务禁搜）。
import { AppInput, AppSelect, AppTextArea } from "@/components/ui";
import { useBuildingGeocoding } from "./useBuildingGeocoding";

type ViewMode = "table" | "card";

// 搜索区状态选项（值沿用后端 status 口径 0=正常 / 1=停用）
const BUILDING_STATUS_OPTIONS = [
  { label: "正常", value: 0 },
  { label: "停用", value: 1 },
];

/**
 * 卡片视图图标动作的禁用态样式（Phase 86 W2）。
 * antd `Card.actions` 只接受 ReactNode，无法给图标挂 `disabled` 属性 →
 * 未就绪（loading/error）时用「不响应指针 + 降透明度」表达禁用态，
 * 并配合 `title={decision.hint}` 提示原因（与表格视图 ActionButtons 的三态同源）。
 */
const cardActionStyle = (disabled: boolean): CSSProperties => ({
  opacity: disabled ? 0.45 : 1,
  pointerEvents: disabled ? "none" : "auto",
  cursor: disabled ? "not-allowed" : "pointer",
});

const BuildingManagement: FC = () => {
  // Phase 86 W2（RBAC-02）：操作级权限门控（三态 ready/loading/error）。
  // 权限码单一真相源 = src/constants/permissionCodes.ts；⚠️ ops 八组只有 delete 键，
  // 没有 remove 键 —— 删除类入口一律用 .delete（写 .remove 会 TS2339）。
  const permGate = useActionPermission();
  const permEdit = permGate.decide(PERMISSION_CODES.opsBuilding.edit);
  const permDelete = permGate.decide(PERMISSION_CODES.opsBuilding.delete);
  const permAdd = permGate.decide(PERMISSION_CODES.opsBuilding.add);
  const permImport = permGate.decide(PERMISSION_CODES.opsBuilding.import);
  const permExport = permGate.decide(PERMISSION_CODES.opsBuilding.export);

  // 左侧部门树选中态（列表过滤经 PageTable extraFilters 联动）
  const [selectedDeptId, setSelectedDeptId] = useState<string>("");

  // 使用自定义 Hooks
  const geocoding = useBuildingGeocoding();
  // Phase 37 批 2:直接消费 canonical useDeptTree
  const { data: departments = [], isLoading: departmentLoading } = useDeptTree();
  const getOrgName = useCallback(
    (orgId?: string): string => {
      if (!orgId) return "-";
      return findDeptNode(departments, orgId)?.deptName ?? "-";
    },
    [departments]
  );

  const [statistics, setStatistics] = useState({ total: 0, active: 0, inactive: 0 });
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

  // 卡片视图数据快照（与表格同一 request，不另建数据层）
  const [buildings, setBuildings] = useState<Building[]>([]);

  // 新增/编辑弹窗态（原表格管理器 hook 的 editForm/editModalVisible/editingItem 位）
  const [modalVisible, setModalVisible] = useState(false);
  const [editingBuilding, setEditingBuilding] = useState<Building | null>(null);
  const [buildingForm] = Form.useForm();

  // PageTable render-prop 上下文提升到业务回调（提交/删除/导入后刷新 + 导出筛选）
  const reloadRef = useRef<() => void>(() => {});
  const searchFormRef = useRef<FormInstance | null>(null);

  // 服务端排序:field 对应后端 buildingAllowedSortFields 白名单 key
  const sorterMetas = useMemo(
    () => [
      createSorterMeta<Building>("name"),
      createSorterMeta<Building>("orgId"),
      createSorterMeta<Building>("level"),
      createSorterMeta<Building>("address"),
      createSorterMeta<Building>("totalFloors"),
      createSorterMeta<Building>("status"),
      createSorterMeta<Building>("orderNum"),
      createSorterMeta<Building>("createdAt", "date"),
    ],
    []
  );

  const loadStatistics = useCallback(async (searchParams: Record<string, unknown> = {}) => {
    try {
      const stats = await buildingApi.statistics(searchParams);
      setStatistics({
        total: stats.total ?? 0,
        active: stats.active ?? 0,
        inactive: stats.inactive ?? 0,
      });
    } catch (error) {
      handleApiError(error, "加载统计数据", false);
    }
  }, []);

  // 部门树选中：列表重载由 extraFilters 引用变化内建完成，统计随 request 联动
  const handleDeptSelect = useCallback((selectedKeys: Key[]) => {
    setSelectedDeptId(selectedKeys.length > 0 ? (selectedKeys[0] as string) : "");
  }, []);

  // 部门过滤（PageTable extraFilters：引用变化 → applyFilters 回第 1 页重载）
  const extraFilters = useMemo(
    () => (selectedDeptId ? { orgId: selectedDeptId } : {}),
    [selectedDeptId]
  );

  // 列表请求：查询条件（current/pageSize/filters/排序）由 PageTable 内部组装后透传；
  // 部门过滤在此兜底注入，覆盖搜索 / 重置 / 翻页 / 排序等所有重载路径。
  const request = useCallback(
    async (params: Record<string, unknown>) => {
      const listParams = selectedDeptId ? { ...params, orgId: selectedDeptId } : params;
      const result = (await buildingApi.list(listParams)) as {
        data?: { list: Building[]; total: number };
      };
      const list = result.data?.list || [];
      // 卡片视图快照：与表格同一份数据源
      setBuildings(list);
      // 统计与列表同口径（剥离分页 / 排序参数，只保留过滤条件）
      const filterParams: Record<string, unknown> = { ...listParams };
      (["current", "pageSize", "orderByColumn", "isAsc"] as const).forEach((key) => {
        delete filterParams[key];
      });
      void loadStatistics(filterParams);
      return { list, total: result.data?.total || 0 };
    },
    [selectedDeptId, loadStatistics]
  );

  // ==================== 楼宇操作函数 ====================

  const closeModal = () => {
    setModalVisible(false);
    buildingForm.resetFields();
    setEditingBuilding(null);
    geocoding.reset();
  };

  const handleSave = async () => {
    try {
      const values = (await buildingForm.validateFields()) as Record<string, unknown>;

      // 移除不需要发送给后端的字段
      const { orgName: _orgName, ...submitData } = values;

      // 地址解析：如果有地址，自动解析获取经纬度
      if (submitData.address && String(submitData.address).trim()) {
        const coords = await geocoding.resolveAddress(String(submitData.address));
        if (coords) {
          submitData.longitude = coords.longitude;
          submitData.latitude = coords.latitude;
        }
      } else {
        // 没有地址，清空经纬度
        submitData.longitude = undefined;
        submitData.latitude = undefined;
        geocoding.reset();
      }

      if (editingBuilding) {
        await buildingApi.update(editingBuilding.id, submitData);
        handleSuccess("更新");
      } else {
        await buildingApi.create(submitData);
        handleSuccess("创建");
      }
      closeModal();
      // 刷新列表（请求侧自动带上部门过滤，统计随 request 联动）
      reloadRef.current();
    } catch (error: unknown) {
      if (isFormValidationError(error)) return;
      handleApiError(error, "操作");
    }
  };

  // 单删（经 createDeleteAction → ConfirmDelete；成功/失败提示由 confirmDelete 统一承担）
  const handleDelete = async (id: string) => {
    await buildingApi.delete(id);
    reloadRef.current();
  };

  // 批删（确认/成功提示由 PageTable → BatchDeleteButton → confirmDelete 承担；
  // 成功后 PageTable 自动清空选中并重载列表，统计随 request 联动）
  const handleBatchDelete = async (keys: Key[]) => {
    await buildingApi.batch("delete", { ids: keys });
  };

  const openModal = (record?: Building) => {
    geocoding.reset();

    if (record) {
      setEditingBuilding(record);
      // 只设置表单中实际需要的字段，排除时间字段等不需要的
      const formValues = {
        name: record.name,
        orgId: record.orgId,
        level: record.level ?? 2,
        address: record.address,
        longitude: record.longitude,
        latitude: record.latitude,
        totalFloors: record.totalFloors,
        status: record.status,
        remark: record.remark,
      };
      buildingForm.setFieldsValue(formValues);

      // 如果楼宇已有经纬度，显示出来
      if (record.longitude && record.latitude) {
        geocoding.setResult({
          longitude: record.longitude,
          latitude: record.latitude,
        });
      }
      setModalVisible(true);
    } else {
      setEditingBuilding(null);
      buildingForm.resetFields();
      buildingForm.setFieldsValue({ status: 0, level: 2 });
      setModalVisible(true);
    }
  };

  const handleImportSuccess = () => {
    reloadRef.current();
    setImportVisible(false);
  };

  // 导出：沿用迁移前口径——取当前搜索表单非空值 + 部门过滤作为 filters
  const handleOpenExport = () => {
    const values = (searchFormRef.current?.getFieldsValue() ?? {}) as Record<string, unknown>;
    const currentFilters: Record<string, unknown> = {};
    Object.keys(values).forEach((key) => {
      const value = values[key];
      if (value !== undefined && value !== null && value !== "") {
        currentFilters[key] = value;
      }
    });
    if (selectedDeptId) {
      currentFilters.orgId = selectedDeptId;
    }
    setExportFilters(currentFilters);
    setExportVisible(true);
  };

  // 搜索区 schema：字段名与后端查询参数一一对应（name / status）
  const searchFields = useMemo<SearchField[]>(
    () => [
      { name: "name", label: "楼宇名称", type: "input", placeholder: "请输入楼宇名称" },
      {
        name: "status",
        label: "状态",
        type: "select",
        placeholder: "请选择状态",
        options: BUILDING_STATUS_OPTIONS,
      },
    ],
    []
  );

  const columns: ColumnsType<Building> = [
    {
      title: "楼宇名称",
      dataIndex: "name",
      key: "name",
      width: 150,
      sorter: true,
    },
    {
      title: "所属机构",
      dataIndex: "orgId",
      key: "orgName",
      width: 150,
      sorter: true,
      render: getOrgName,
    },
    {
      title: "层级",
      dataIndex: "level",
      key: "level",
      width: 80,
      sorter: true,
      render: (level: number) => (
        <Tag color={level === 1 ? "purple" : "blue"}>{level === 1 ? "一级" : "二级"}</Tag>
      ),
    },
    {
      title: "地址",
      dataIndex: "address",
      key: "address",
      ellipsis: true,
      width: 200,
      sorter: true,
    },
    {
      title: "经纬度",
      key: "coordinates",
      width: 150,
      render: (_, record) => {
        if (record.longitude && record.latitude) {
          return `${record.longitude.toFixed(6)}, ${record.latitude.toFixed(6)}`;
        }
        return "-";
      },
    },
    {
      title: "楼层数",
      // 实时楼层数（后端 List 按 ops_floors 非软删行数聚合注入，nil → "-"）。
      // 排序参数保持发 totalFloors：后端白名单只认 total_floors 列，
      // 发 floorCount 会落出白名单静默失效（排序参数与 dataIndex 解耦）。
      dataIndex: "floorCount",
      key: "floorCount",
      width: 100,
      sorter: true,
      render: (v) => (v == null ? "-" : v),
    },
    createStatusColumn("status", { width: 100, sorter: true }),
    { title: "描述", dataIndex: "remark", key: "remark", ellipsis: true },
    createDateTimeColumn("createdAt", {
      width: 180,
      title: "创建时间",
      sorter: true,
    }),
    createDateTimeColumn("updatedAt", { width: 180, title: "更新时间" }),
  ];

  // 行操作：编辑 + 删除（数量/顺序与迁移前一致；删除走 createDeleteAction 唯一入口）
  // permission 由 ActionButtons 内建三态门控：ready 无码 → 不渲染；未就绪 → disabled + 提示
  const rowActions = (record: Building) => [
    {
      key: "edit",
      label: "编辑",
      icon: <EditOutlined />,
      onClick: () => openModal(record),
      permission: PERMISSION_CODES.opsBuilding.edit,
    },
    createDeleteAction(record.id, handleDelete, {
      target: `楼宇 ${record.name}`,
      permission: PERMISSION_CODES.opsBuilding.delete,
    }),
  ];

  const renderCardView = () => {
    if (buildings.length === 0)
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
        {buildings.map((building) => (
          <Col key={building.id} xs={24} sm={12} md={8} lg={6}>
            <Card
              hoverable
              actions={[
                // 卡片视图与表格视图是同一页的两种渲染形态 → 同一组门控决策：
                // ready 无码 → 不渲染；未就绪 → 渲染但禁用（图标无 disabled 位，用
                // pointerEvents + opacity 表达禁用态，并挂 title 提示）
                ...(permEdit.visible
                  ? [
                      <EditOutlined
                        key="edit"
                        title={permEdit.hint}
                        style={cardActionStyle(permEdit.disabled)}
                        onClick={() => {
                          if (permEdit.disabled) return;
                          openModal(building);
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
                            target: `楼宇 ${building.name}`,
                            onDelete: () => handleDelete(building.id),
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
                    <span>{building.name}</span>
                    <Tag color={building.status === 0 ? "success" : "error"}>
                      {building.status === 0 ? "正常" : "停用"}
                    </Tag>
                  </div>
                }
                description={
                  <div>
                    {building.address && (
                      <div>
                        <strong>地址：</strong>
                        {building.address}
                      </div>
                    )}
                    <div>
                      <strong>所属机构：</strong>
                      {getOrgName(building.orgId)}
                    </div>
                    <div>
                      <strong>楼层：</strong>
                      {building.totalFloors || 0}层
                    </div>
                  </div>
                }
              />
            </Card>
          </Col>
        ))}
      </Row>
    );
  };

  // 业务区：统计卡片 + 已选择 Alert（PageTable render-prop 顶部，表格 / 卡片视图共用）
  const renderBusinessArea = ({
    selectedRowKeys,
    clearSelection,
    loadData,
    searchForm,
  }: PageTableRenderContext<Building>) => {
    // 提升到业务回调：弹窗提交/删除/导入成功后刷新列表；导出取当前搜索条件
    reloadRef.current = loadData;
    searchFormRef.current = searchForm;
    return (
      <>
        <Row gutter={16} style={{ marginBottom: 16 }}>
          <Col span={8}>
            <Card>
              <Statistic
                title="总楼宇数"
                value={statistics.total}
                prefix={<CheckCircleOutlined />}
              />
            </Card>
          </Col>
          <Col span={8}>
            <Card>
              <Statistic
                title="正常楼宇"
                value={statistics.active}
                styles={{ content: { color: "var(--theme-success)" } }}
                prefix={<CheckCircleOutlined />}
              />
            </Card>
          </Col>
          <Col span={8}>
            <Card>
              <Statistic
                title="停用楼宇"
                value={statistics.inactive}
                styles={{ content: { color: "var(--theme-error)" } }}
                prefix={<StopOutlined />}
              />
            </Card>
          </Col>
        </Row>
        {selectedRowKeys.length > 0 && (
          <Alert
            title={
              <span>
                已选择 <strong>{selectedRowKeys.length}</strong> 个楼宇，
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
      {/* 左侧部门树 */}
      <Sider
        width={360}
        className="dept-list-sider"
        style={{ background: "var(--theme-bg-surface)", padding: "0 16px 16px 0", borderRight: "1px solid var(--theme-border-primary)" }}
      >
        <DeptTree
          onSelect={(selectedKeys) => handleDeptSelect(selectedKeys)}
          selectedKeys={selectedDeptId ? [selectedDeptId] : []}
          externalOnly={true}
        />
      </Sider>

      {/* 右侧内容区 */}
      <Content style={{ background: "var(--theme-bg-surface)" }}>
        <div>
          <PageTable<Building>
            request={request}
            columns={columns}
            rowKey={(record) => record.id}
            searchFields={searchFields}
            sorterMetas={sorterMetas}
            extraFilters={extraFilters}
            rowActions={rowActions}
            // 批量删除：PageTable 的 batchDelete 契约无 permission 位 → 页面层条件传参
            // （ready 无权限 → 不传 → 内建 BatchDeleteButton 与 rowSelection 都不渲染）
            batchDelete={
              permDelete.visible
                ? { onDelete: handleBatchDelete, target: "楼宇" }
                : undefined
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
                {/* Excel 导入/导出的门控点落在触发按钮（modal 容器 :690/:697 保留无条件渲染，
                    按钮不渲染 ⇒ 弹窗永远打不开） */}
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
                    新增楼宇
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
            title={editingBuilding ? "编辑楼宇" : "新增楼宇"}
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
            width={600}
          >
            <Form
              form={buildingForm}
              layout="horizontal"
              labelCol={{ span: 5 }}
              wrapperCol={{ span: 19 }}
            >
              <Form.Item
                name="name"
                label="楼宇名称"
                rules={[{ required: true, message: "请输入楼宇名称" }]}
              >
                <AppInput placeholder="请输入楼宇名称" />
              </Form.Item>

              <Form.Item name="orgId" label="所属机构">
                <DepartmentTreeSelect
                  departments={departments}
                  loading={departmentLoading}
                  placeholder="请选择所属机构"
                />
              </Form.Item>

              <Form.Item
                name="level"
                label="层级"
                rules={[{ required: true, message: "请选择层级" }]}
              >
                <Radio.Group>
                  <Radio value={1}>一级（城市级汇总）</Radio>
                  <Radio value={2}>二级（具体楼宇）</Radio>
                </Radio.Group>
              </Form.Item>

              <Form.Item name="address" label="地址">
                <AppInput placeholder="请输入地址（保存时将自动解析经纬度）" />
              </Form.Item>

              {/* 经纬度显示（只读，自动解析） */}
              {(geocoding.result || editingBuilding?.longitude) && (
                <Form.Item label="经纬度">
                  <AppInput
                    disabled
                    allowClear={false}
                    placeholder="经纬度（保存时根据地址自动解析）"
                    value={
                      geocoding.result
                        ? `${geocoding.result.longitude?.toFixed(6)}, ${geocoding.result.latitude?.toFixed(6)}`
                        : editingBuilding?.longitude && editingBuilding?.latitude
                          ? `${editingBuilding.longitude.toFixed(6)}, ${editingBuilding.latitude.toFixed(6)}`
                          : ""
                    }
                  />
                </Form.Item>
              )}

              {/* 地址解析警告提示 */}
              {geocoding.warning && (
                <Alert
                  title={geocoding.warning}
                  type="warning"
                  showIcon
                  style={{ marginBottom: 16 }}
                />
              )}

              <Form.Item name="totalFloors" label="楼层数">
                <InputNumber
                  min={0}
                  disabled
                  style={{ width: "100%" }}
                  placeholder="根据创建的楼层自动计算"
                />
              </Form.Item>

              <Form.Item
                name="status"
                label="状态"
                rules={[{ required: true, message: "请选择状态" }]}
              >
                {/* 形态分类器②④：楼宇状态枚举（业务禁搜），children → options */}
                <AppSelect
                  allowClear
                  placeholder="请选择状态"
                  showSearch={false}
                  options={BUILDING_STATUS_OPTIONS}
                />
              </Form.Item>

              <Form.Item name="remark" label="描述">
                <AppTextArea rows={3} placeholder="请输入描述" />
              </Form.Item>

              {/* 隐藏字段：确保经纬度被包含在表单值中 */}
              <Form.Item name="longitude" hidden>
                <InputNumber />
              </Form.Item>
              <Form.Item name="latitude" hidden>
                <InputNumber />
              </Form.Item>
            </Form>
          </Modal>
          <ExcelImportLazy
            entityType="building"
            entityName="楼宇"
            visible={importVisible}
            onClose={() => setImportVisible(false)}
            onImportSuccess={handleImportSuccess}
          />
          <ExcelExport
            entityType="building"
            entityName="楼宇"
            visible={exportVisible}
            onClose={() => setExportVisible(false)}
            filters={exportFilters}
          />
        </div>
      </Content>
    </Layout>
  );
};

export default BuildingManagement;
