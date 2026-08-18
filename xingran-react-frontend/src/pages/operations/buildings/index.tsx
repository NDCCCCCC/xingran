import { useState, useEffect, useCallback, useRef, useMemo, type FC, type Key } from "react";
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
  Alert,
  Radio,
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
  AppstoreOutlined,
  TableOutlined,
  ImportOutlined,
  ExportOutlined,
} from "@ant-design/icons";

const { Sider, Content } = Layout;
import type { Building } from "@/types/operations";
import { buildingApi } from "@/lib/opsApi";
import { useTableManager } from "@/hooks/useTableManager";
import { usePagination } from "@/hooks/usePagination";
import { handleApiError, handleSuccess } from "@/utils/errorHandler";
import { isFormValidationError } from "@/utils/errorHandler";
import { createStatusColumn, createDateTimeColumn, createSorterMeta } from "@/utils/tableHelpers";
import ActionButtons from "@/components/shared/ActionButtons";
import ExcelImport from "@/components/shared/ExcelImport";
import ExcelExport from "@/components/shared/ExcelExport";
import { DepartmentTreeSelect } from "@/components/shared";
import DeptTree from "@/components/DeptTree";
import { useDeptTree } from "@/hooks/useDeptTree";
import { findDeptNode } from "@/utils/deptUtils";
import { useBuildingGeocoding } from "./useBuildingGeocoding";

const { Option } = Select;
const { TextArea } = Input;

type ViewMode = "table" | "card";

const BuildingManagement: FC = () => {
  // 自定义状态
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
  const [exportVisible, setExportVisible] = useState(false);
  const [exportFilters, setExportFilters] = useState<Record<string, unknown>>({});

  // 使用全局分页 hook
  const { paginationProps, setCurrent, setPageSize, setTotal } = usePagination();

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

  const {
    loading,
    data: buildings,
    total: _total,
    selectedRowKeys,
    searchForm,
    editForm: buildingForm,
    editModalVisible: modalVisible,
    editingItem: editingBuilding,
    setSelectedRowKeys,
    setEditModalVisible: setModalVisible,
    setEditingItem: setEditingBuilding,
    handleSearch: _tableHandleSearch,
    handleReset: _handleReset,
    handleAdd,
    handleEdit,
    handleModalClose,
    loadData: loadBuildings,
    resetSelection,
    getColumnSortOrder,
    handleTableChange,
  } = useTableManager<Building>(
    async (params) => {
      const result = (await buildingApi.list(params)) as {
        data?: { list: Building[]; total: number };
      };
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

  // 使用 ref 跟踪初始化状态，避免循环请求
  const isInitialized = useRef(false);
  const prevPageRef = useRef({ current: 1, pageSize: 10 });

  // 部门树选择处理
  const handleDeptSelect = useCallback(
    (selectedKeys: Key[]) => {
      const searchParams: Record<string, unknown> = {};
      if (selectedKeys.length > 0) {
        setSelectedDeptId(selectedKeys[0] as string);
        searchParams.orgId = selectedKeys[0] as string;
      } else {
        setSelectedDeptId("");
      }
      loadBuildings(searchParams);
      loadStatistics(searchParams);
    },
    [loadBuildings, loadStatistics]
  );

  // 搜索处理（支持部门过滤）
  const handleSearch = useCallback(() => {
    const values = searchForm.getFieldsValue() as Record<string, unknown>;
    const searchParams: Record<string, unknown> = {};
    Object.keys(values).forEach((key) => {
      const value = values[key];
      if (value !== undefined && value !== null && value !== "") {
        searchParams[key] = value;
      }
    });
    if (selectedDeptId) {
      searchParams.orgId = selectedDeptId;
    }
    loadBuildings(searchParams);
    loadStatistics(searchParams);
  }, [searchForm, selectedDeptId, loadBuildings, loadStatistics]);

  // 重置处理（支持部门过滤）
  const handleResetWithDept = useCallback(() => {
    searchForm.resetFields();
    const searchParams: Record<string, unknown> = {};
    if (selectedDeptId) {
      searchParams.orgId = selectedDeptId;
    }
    loadBuildings(searchParams);
    loadStatistics(searchParams);
  }, [searchForm, selectedDeptId, loadBuildings, loadStatistics]);

  // 刷新处理（支持部门过滤）
  const handleRefresh = useCallback(() => {
    const searchParams: Record<string, unknown> = {};
    if (selectedDeptId) {
      searchParams.orgId = selectedDeptId;
    }
    loadBuildings(searchParams);
    loadStatistics(searchParams);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedDeptId]);

  // 初始加载和分页变化处理
  useEffect(() => {
    const { current, pageSize } = paginationProps;
    const prev = prevPageRef.current;

    // 检查分页是否真的发生了变化
    const pageChanged = prev.current !== current || prev.pageSize !== pageSize;

    if (!isInitialized.current || pageChanged) {
      isInitialized.current = true;
      prevPageRef.current = { current: current ?? 1, pageSize: pageSize ?? 10 };

      // 使用 setTimeout 避免同步 setState 导致的循环
      // 部门树由 useDeptTree 自动获取,无需手动触发
      setTimeout(() => {
        loadBuildings();
        loadStatistics();
      }, 0);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [paginationProps.current, paginationProps.pageSize]);

  // ==================== 楼宇操作函数 ====================

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
      handleModalClose();
      // 刷新时保持部门筛选状态
      const searchParams: Record<string, unknown> = {};
      if (selectedDeptId) {
        searchParams.orgId = selectedDeptId;
      }
      loadBuildings(searchParams);
      loadStatistics(searchParams);
    } catch (error: unknown) {
      if (isFormValidationError(error)) return;
      handleApiError(error, "操作");
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await buildingApi.delete(id);
      handleSuccess("删除");
      // 刷新时保持部门筛选状态
      const searchParams: Record<string, unknown> = {};
      if (selectedDeptId) {
        searchParams.orgId = selectedDeptId;
      }
      loadBuildings(searchParams);
      loadStatistics(searchParams);
    } catch (error) {
      handleApiError(error, "删除");
    }
  };

  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) return;
    try {
      await buildingApi.batch("delete", { ids: selectedRowKeys });
      handleSuccess("批量删除");
      resetSelection();
      // 刷新时保持部门筛选状态
      const searchParams: Record<string, unknown> = {};
      if (selectedDeptId) {
        searchParams.orgId = selectedDeptId;
      }
      loadBuildings(searchParams);
      loadStatistics(searchParams);
    } catch (error) {
      handleApiError(error, "批量删除");
    }
  };

  const openModal = (record?: Building) => {
    geocoding.reset();

    if (record) {
      handleEdit(record);
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
    } else {
      handleAdd();
      buildingForm.setFieldsValue({ status: 0, level: 2 });
    }
  };

  const handleImportSuccess = () => {
    const searchParams: Record<string, unknown> = {};
    if (selectedDeptId) {
      searchParams.orgId = selectedDeptId;
    }
    loadBuildings(searchParams);
    loadStatistics(searchParams);
    setImportVisible(false);
  };

  const columns: ColumnsType<Building> = [
    {
      title: "楼宇名称",
      dataIndex: "name",
      key: "name",
      width: 150,
      sorter: true,
      sortOrder: getColumnSortOrder("name"),
    },
    {
      title: "所属机构",
      dataIndex: "orgId",
      key: "orgName",
      width: 150,
      sorter: true,
      sortOrder: getColumnSortOrder("orgId"),
      render: getOrgName,
    },
    {
      title: "层级",
      dataIndex: "level",
      key: "level",
      width: 80,
      sorter: true,
      sortOrder: getColumnSortOrder("level"),
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
      sortOrder: getColumnSortOrder("address"),
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
      dataIndex: "totalFloors",
      key: "totalFloors",
      width: 100,
      sorter: true,
      sortOrder: getColumnSortOrder("totalFloors"),
      render: (v) => v || 0,
    },
    createStatusColumn("status", {
      width: 100,
      sorter: true,
      sortOrder: getColumnSortOrder("status"),
    }),
    { title: "描述", dataIndex: "remark", key: "remark", ellipsis: true },
    createDateTimeColumn("createdAt", {
      width: 180,
      title: "创建时间",
      sorter: true,
      sortOrder: getColumnSortOrder("createdAt"),
    }),
    createDateTimeColumn("updatedAt", { width: 180, title: "更新时间" }),
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
                title: "确定要删除这个楼宇吗？",
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
    if (buildings.length === 0)
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
        {buildings.map((building) => (
          <Col key={building.id} xs={24} sm={12} md={8} lg={6}>
            <Card
              hoverable
              actions={[
                <EditOutlined key="edit" onClick={() => openModal(building)} />,
                <DeleteOutlined
                  key="delete"
                  style={{ color: "var(--theme-error, #ba3630)" }}
                  onClick={() => {
                    Modal.confirm({
                      title: "确定要删除这个楼宇吗？",
                      okText: "确定",
                      cancelText: "取消",
                      okButtonProps: { danger: true },
                      onOk: () => handleDelete(building.id),
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
                    <span>{building.name}</span>
                    <Tag color={building.status === 0 ? "success" : "error"}>
                      {building.status === 0 ? "正常" : "1"}
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

  return (
    <Layout style={{ background: "#000", minHeight: "calc(100vh - 64px)" }}>
      {/* 左侧部门树 */}
      <Sider
        width={360}
        className="dept-list-sider"
        style={{ background: "#fff", padding: "0 16px 16px 0", borderRight: "1px solid #e9efeb" }}
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
                  styles={{ content: { color: "var(--theme-success, #3f8600)" } }}
                  prefix={<CheckCircleOutlined />}
                />
              </Card>
            </Col>
            <Col span={8}>
              <Card>
                <Statistic
                  title="停用楼宇"
                  value={statistics.inactive}
                  styles={{ content: { color: "var(--theme-error, #cf1322)" } }}
                  prefix={<StopOutlined />}
                />
              </Card>
            </Col>
          </Row>
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
                <Form.Item name="name" label="楼宇名称">
                  <Input
                    placeholder="请输入楼宇名称"
                    allowClear
                    className="user-form-input"
                    style={{ width: 150 }}
                  />
                </Form.Item>
                <Form.Item name="status" label="状态">
                  <Select
                    placeholder="请选择状态"
                    allowClear
                    className="user-form-input"
                    style={{ width: 120 }}
                    onSearch={() => {}}
                  >
                    <Option value={0}>正常</Option>
                    <Option value={1}>停用</Option>
                  </Select>
                </Form.Item>
                <Form.Item>
                  <Space>
                    <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>
                      搜索
                    </Button>
                    <Button onClick={handleResetWithDept}>重置</Button>
                    <Button icon={<ReloadOutlined />} onClick={handleRefresh}>
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
                    // 获取当前筛选条件
                    const values = searchForm.getFieldsValue() as Record<string, unknown>;
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
                  新增楼宇
                </Button>
              </Space>
            </div>
            {selectedRowKeys.length > 0 && (
              <Alert
                title={
                  <span>
                    已选择 <strong>{selectedRowKeys.length}</strong> 个楼宇，
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
                dataSource={buildings}
                loading={loading}
                rowKey="id"
                pagination={paginationProps}
                onChange={handleTableChange}
              />
            ) : (
              renderCardView()
            )}
          </Card>
          <Modal
            title={editingBuilding ? "编辑楼宇" : "新增楼宇"}
            open={modalVisible}
            onOk={handleSave}
            onCancel={() => {
              setModalVisible(false);
              buildingForm.resetFields();
              setEditingBuilding(null);
              geocoding.reset();
            }}
            width={600}
            confirmLoading={geocoding.loading}
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
                <Input placeholder="请输入楼宇名称" />
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
                <Input placeholder="请输入地址（保存时将自动解析经纬度）" />
              </Form.Item>

              {/* 经纬度显示（只读，自动解析） */}
              {(geocoding.result || editingBuilding?.longitude) && (
                <Form.Item label="经纬度">
                  <Input
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
                <Select placeholder="请选择状态" onSearch={() => {}}>
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
          <ExcelImport
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
