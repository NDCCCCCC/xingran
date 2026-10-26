/**
 * 楼宇管理页面 — Phase 68.1 T6b：operations 域 PageTable 契约形态
 *
 * 迁移自页面级双 hook 数据层实现（useTableManager + usePagination）：数据层 /
 * 搜索区 / 分页 / 三态 / 删除确认全部收敛到 PageTable（UI-SPEC §3.2），本页只保留
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

import { useState, useCallback, useRef, useMemo, type FC, type Key } from "react";
import type { FormInstance } from "antd";
import { useLocation } from "react-router-dom";
import { usePersistedStateController } from "@/hooks/usePersistedState";
import {
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
import { PageTable } from "@/components/shared";
import type { PageTableRenderContext, SearchField } from "@/components/shared";
import { createDeleteAction } from "@/components/shared/ActionButtons";
import ExcelImportLazy from "@/components/shared/ExcelImportLazy";
import ExcelExport from "@/components/shared/ExcelExport";
import { confirmDelete } from "@/components/shared/confirmDelete";
import { DepartmentTreeSelect } from "@/components/shared";
import DeptTree from "@/components/DeptTree";
import { useDeptTree } from "@/hooks/useDeptTree";
import { findDeptNode } from "@/utils/deptUtils";
import { useBuildingGeocoding } from "./useBuildingGeocoding";

const { Option } = Select;
const { TextArea } = Input;

type ViewMode = "table" | "card";

// 搜索区状态选项（值沿用后端 status 口径 0=正常 / 1=停用）
const BUILDING_STATUS_OPTIONS = [
  { label: "正常", value: 0 },
  { label: "停用", value: 1 },
];

const BuildingManagement: FC = () => {
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
      const {
        current: _current,
        pageSize: _pageSize,
        orderByColumn: _orderByColumn,
        isAsc: _isAsc,
        ...filterParams
      } = listParams;
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
  const rowActions = (record: Building) => [
    { key: "edit", label: "编辑", icon: <EditOutlined />, onClick: () => openModal(record) },
    createDeleteAction(record.id, handleDelete, { label: `楼宇 ${record.name}` }),
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
                <EditOutlined key="edit" onClick={() => openModal(building)} />,
                <DeleteOutlined
                  key="delete"
                  style={{ color: "var(--theme-error)" }}
                  onClick={() =>
                    confirmDelete({
                      target: `楼宇 ${building.name}`,
                      onDelete: () => handleDelete(building.id),
                    })
                  }
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
        style={{ background: "#fff", padding: "0 16px 16px 0", borderRight: "1px solid var(--theme-border-primary)" }}
      >
        <DeptTree
          onSelect={(selectedKeys) => handleDeptSelect(selectedKeys)}
          selectedKeys={selectedDeptId ? [selectedDeptId] : []}
          externalOnly={true}
        />
      </Sider>

      {/* 右侧内容区 */}
      <Content style={{ background: "#fff" }}>
        <div>
          <PageTable<Building>
            request={request}
            columns={columns}
            rowKey={(record) => record.id}
            searchFields={searchFields}
            sorterMetas={sorterMetas}
            extraFilters={extraFilters}
            rowActions={rowActions}
            batchDelete={{ onDelete: handleBatchDelete, target: "楼宇" }}
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
                  新增楼宇
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
                <Input allowClear placeholder="请输入楼宇名称" />
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
                <Input allowClear placeholder="请输入地址（保存时将自动解析经纬度）" />
              </Form.Item>

              {/* 经纬度显示（只读，自动解析） */}
              {(geocoding.result || editingBuilding?.longitude) && (
                <Form.Item label="经纬度">
                  <Input allowClear
                    disabled
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
                  message={geocoding.warning}
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
                <Select allowClear placeholder="请选择状态" onSearch={() => {}}>
                  <Option value={0}>正常</Option>
                  <Option value={1}>停用</Option>
                </Select>
              </Form.Item>

              <Form.Item name="remark" label="描述">
                <TextArea rows={3} placeholder="请输入描述" />
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
