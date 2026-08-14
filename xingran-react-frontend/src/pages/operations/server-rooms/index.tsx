/**
 * 机房管理页面
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
  Select,
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
  SearchOutlined,
  ReloadOutlined,
  CheckCircleOutlined,
  StopOutlined,
  AppstoreOutlined,
  TableOutlined,
  ImportOutlined,
  ExportOutlined,
} from "@ant-design/icons";
import type { ServerRoom, Building, Floor } from "@/types";
import { serverRoomApi, buildingApi, floorApi } from "@/lib/opsApi";
import { useTableManager } from "@/hooks/useTableManager";
import { usePagination } from "@/hooks/usePagination";
import { useSidebarDeptFilter } from "@/hooks/useSidebarDeptFilter";
import { handleApiError, handleSuccess, isFormValidationError } from "@/utils/errorHandler";
import { createStatusColumn, createDateTimeColumn, createSorterMeta } from "@/utils/tableHelpers";
import ActionButtons from "@/components/shared/ActionButtons";
import ExcelImport from "@/components/shared/ExcelImport";
import ExcelExport from "@/components/shared/ExcelExport";
import { DeptSidebar } from "@/components/operations/DeptSidebar";
import { StatisticsCards } from "@/components/operations/StatisticsCards";

const { Option } = Select;
const { TextArea } = Input;
const { Content } = Layout;

type ViewMode = "table" | "card";

interface BuildingOption {
  id: string;
  name: string;
  orgId: string;
}

interface FloorOption {
  id: string;
  name: string;
  floorNo: string;
}

interface Statistics {
  total: number;
  active: number;
  inactive: number;
}

const ServerRoomManagement: FC = () => {
  const location = useLocation();
  const [viewMode, setViewMode] = usePersistedStateController<ViewMode>({
    keyPrefix: location.pathname,
    keySuffix: "viewMode",
    defaultValue: "table",
  });
  const [importVisible, setImportVisible] = useState(false);
  const [exportVisible, setExportVisible] = useState(false);
  const [exportFilters, setExportFilters] = useState<Record<string, unknown>>({});
  const [buildingOptions, setBuildingOptions] = useState<BuildingOption[]>([]);
  const [floorOptions, setFloorOptions] = useState<FloorOption[]>([]);
  const [statistics, setStatistics] = useState<Statistics>({ total: 0, active: 0, inactive: 0 });

  const { paginationProps, setCurrent, setPageSize, setTotal } = usePagination();

  // 使用 useSidebarDeptFilter hook 管理部门筛选
  const { selectedDeptId, handleDeptSelect, setSelectedDeptId } = useSidebarDeptFilter({
    searchForm: undefined, // 机房管理不需要清空搜索表单字段
    clearFieldNames: [],
  });

  // 使用 ref 存储最新的 selectedDeptId，避免闭包问题
  const selectedDeptIdRef = useRef<string>("");
  useEffect(() => {
    selectedDeptIdRef.current = selectedDeptId;
  }, [selectedDeptId]);

  // 服务端排序:field 必须与 columns 的 dataIndex 一致
  const sorterMetas = useMemo(
    () => [
      createSorterMeta<ServerRoom>("name"),
      createSorterMeta<ServerRoom>("buildingName"),
      createSorterMeta<ServerRoom>("floorName"),
      createSorterMeta<ServerRoom>("status"),
      createSorterMeta<ServerRoom>("createdAt", "date"),
    ],
    []
  );

  const {
    loading,
    data: serverRooms,
    total,
    selectedRowKeys,
    searchForm,
    editForm: serverRoomForm,
    editModalVisible: modalVisible,
    editingItem: editingServerRoom,
    setSelectedRowKeys,
    setEditModalVisible: setModalVisible,
    setEditingItem: setEditingServerRoom,
    handleSearch,
    handleReset,
    handleAdd,
    handleEdit,
    handleModalClose,
    loadData: loadServerRooms,
    resetSelection,
    setData,
    setLoading,
    handleTableChange: handleServerRoomTableChange,
    getColumnSortOrder: getServerRoomColumnSortOrder,
  } = useTableManager<ServerRoom>(
    async (params) => {
      // 使用 ref 获取最新的部门 ID，避免闭包捕获旧值
      const currentDeptId = selectedDeptIdRef.current;
      const searchParams = currentDeptId ? { ...params, orgId: currentDeptId } : params;
      const result = await serverRoomApi.list(searchParams);
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
  const loadServerRoomsByDept = useCallback(async () => {
    setCurrent(1);
    loadServerRooms();
  }, [setCurrent, loadServerRooms]);

  useEffect(() => {
    loadServerRoomsByDept();
  }, [selectedDeptId, loadServerRoomsByDept]);

  const loadStatistics = useCallback(async (): Promise<Statistics> => {
    try {
      const stats = await serverRoomApi.statistics();
      return { total: stats.total ?? 0, active: stats.active ?? 0, inactive: stats.inactive ?? 0 };
    } catch (error) {
      handleApiError(error, "加载统计数据", false);
      return { total: 0, active: 0, inactive: 0 };
    }
  }, []);

  const loadBuildingOptions = useCallback(async (orgId?: string) => {
    try {
      const params: Record<string, unknown> = { current: 1, pageSize: 50 };
      if (orgId) params.orgId = orgId;
      const result = await buildingApi.list(params);
      const buildings = result.data?.list || [];
      setBuildingOptions(
        buildings.map((b: Building) => ({
          id: b.id,
          name: b.name,
          orgId: b.orgId,
        }))
      );
    } catch (error) {
      handleApiError(error, "加载楼宇选项", false);
    }
  }, []);

  const loadFloorOptions = useCallback(async (buildingId: string) => {
    if (!buildingId) {
      setFloorOptions([]);
      return;
    }
    try {
      const result = await floorApi.list({ current: 1, pageSize: 50, buildingId });
      const floors = result.data?.list || [];
      setFloorOptions(
        floors.map((f: Floor) => ({
          id: f.id,
          name: f.name || "",
          floorNo: String(f.floorNo),
        }))
      );
    } catch (error) {
      handleApiError(error, "加载楼层选项", false);
    }
  }, []);

  // 统一的数据刷新函数
  const refreshData = useCallback(() => {
    loadServerRooms();
    loadStatistics().then(setStatistics);
  }, [loadServerRooms, loadStatistics]);

  // 初始化加载
  useEffect(() => {
    loadStatistics().then(setStatistics);
  }, [loadStatistics]);

  // 部门变化时清空搜索表单并加载楼宇选项
  useEffect(() => {
    searchForm.setFieldValue("buildingId", undefined);
    searchForm.setFieldValue("floorNo", undefined);
    searchForm.setFieldValue("name", undefined);
    searchForm.setFieldValue("status", undefined);
    loadBuildingOptions(selectedDeptId || undefined);
  }, [selectedDeptId, searchForm, loadBuildingOptions]);

  const handleSave = async () => {
    try {
      const values = (await serverRoomForm.validateFields()) as Partial<ServerRoom>;
      if (editingServerRoom) {
        await serverRoomApi.update(editingServerRoom.id, values);
        handleSuccess("更新");
      } else {
        await serverRoomApi.create(values);
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
      await serverRoomApi.delete(id);
      handleSuccess("删除");
      refreshData();
    } catch (error) {
      handleApiError(error, "删除");
    }
  };

  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) return;
    try {
      await serverRoomApi.batch("delete", { ids: selectedRowKeys });
      handleSuccess("批量删除");
      resetSelection();
      refreshData();
    } catch (error) {
      handleApiError(error, "批量删除");
    }
  };

  const openModal = async (record?: ServerRoom) => {
    if (record) {
      handleEdit(record);
      serverRoomForm.setFieldsValue(record);
      await loadFloorOptions(record.buildingId);
    } else {
      handleAdd();
      serverRoomForm.setFieldsValue({ status: 0 });
      setFloorOptions([]);
      if (selectedDeptId && buildingOptions.length > 0) {
        const firstBuilding = buildingOptions.find((b) => b.orgId === selectedDeptId);
        if (firstBuilding) {
          serverRoomForm.setFieldValue("buildingId", firstBuilding.id);
          await loadFloorOptions(firstBuilding.id);
        }
      }
    }
  };

  const handleImportSuccess = () => {
    refreshData();
    setImportVisible(false);
  };

  const columns: ColumnsType<ServerRoom> = [
    {
      title: "机房名称",
      dataIndex: "name",
      key: "name",
      width: 150,
      sorter: true,
      sortOrder: getServerRoomColumnSortOrder("name"),
    },
    {
      title: "所在楼宇",
      dataIndex: "buildingName",
      key: "buildingName",
      width: 120,
      sorter: true,
      sortOrder: getServerRoomColumnSortOrder("buildingName"),
      render: (_, record) => record.buildingName || record.buildingId,
    },
    {
      title: "楼层",
      dataIndex: "floorName",
      key: "floorName",
      width: 100,
      sorter: true,
      sortOrder: getServerRoomColumnSortOrder("floorName"),
      render: (_, record) => record.floorName || record.floorNo || "-",
    },
    createStatusColumn("status", {
      width: 100,
      sorter: true,
      sortOrder: getServerRoomColumnSortOrder("status"),
    }),
    { title: "描述", dataIndex: "remark", key: "remark", ellipsis: true },
    createDateTimeColumn("createdAt", {
      width: 180,
      sorter: true,
      sortOrder: getServerRoomColumnSortOrder("createdAt"),
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
                title: "确定要删除这个机房吗？",
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
    if (serverRooms.length === 0)
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
        {serverRooms.map((room) => (
          <Col key={room.id} xs={24} sm={12} md={8} lg={6}>
            <Card
              hoverable
              actions={[
                <EditOutlined key="edit" onClick={() => openModal(room)} />,
                <DeleteOutlined
                  key="delete"
                  style={{ color: "var(--theme-error, #ff4d4f)" }}
                  onClick={() => {
                    Modal.confirm({
                      title: "确定要删除这个机房吗？",
                      okText: "确定",
                      cancelText: "取消",
                      okButtonProps: { danger: true },
                      onOk: () => handleDelete(room.id),
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
                    <span>{room.name}</span>
                    <Tag color={room.status === 0 ? "success" : "error"}>
                      {room.status === 0 ? "正常" : "停用"}
                    </Tag>
                  </div>
                }
                description={
                  <div>
                    <div>
                      <strong>位置：</strong>
                      {room.buildingName || room.buildingId} {room.floorNo}层
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
      <DeptSidebar selectedDeptId={selectedDeptId} onSelect={handleDeptSelect} />
      <Content style={{ background: "#fff" }}>
        <StatisticsCards
          show={total > 10}
          items={[
            { title: "总机房数", value: statistics.total, prefix: <CheckCircleOutlined /> },
            {
              title: "正常机房",
              value: statistics.active,
              styles: { content: { color: "var(--theme-success, #3f8600)" } },
              prefix: <CheckCircleOutlined />,
            },
            {
              title: "停用机房",
              value: statistics.inactive,
              styles: { content: { color: "var(--theme-error, #cf1322)" } },
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
              <Form.Item name="buildingId" label="所在楼宇">
                <Select
                  placeholder="请先选择部门"
                  options={buildingOptions.map((b) => ({ label: b.name, value: b.id }))}
                  disabled={!selectedDeptId}
                  onSearch={() => {}}
                />
              </Form.Item>
              <Form.Item name="floorNo" label="楼层">
                <Input placeholder="请输入楼层" />
              </Form.Item>
              <Form.Item name="name" label="机房名称">
                <Input placeholder="请输入机房名称" />
              </Form.Item>
              <Form.Item name="status" label="状态">
                <Select placeholder="请选择状态" onSearch={() => {}}>
                  <Option value={0}>正常</Option>
                  <Option value={1}>停用</Option>
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
                  // 获取当前筛选条件
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
                  style={{ color: "var(--theme-error, #ff4d4f)" }}
                  onClick={handleBatchDelete}
                >
                  批量删除 ({selectedRowKeys.length})
                </Button>
              )}
              <Button type="primary" icon={<PlusOutlined />} onClick={() => openModal()}>
                新增机房
              </Button>
            </Space>
          </div>
          {selectedRowKeys.length > 0 && (
            <Alert
              title={
                <span>
                  已选择 <strong>{selectedRowKeys.length}</strong> 个机房，
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
              dataSource={serverRooms}
              loading={loading}
              rowKey="id"
              pagination={paginationProps}
              onChange={handleServerRoomTableChange}
            />
          ) : (
            renderCardView()
          )}
        </Card>
        <Modal
          title={editingServerRoom ? "编辑机房" : "新增机房"}
          open={modalVisible}
          onOk={handleSave}
          onCancel={() => {
            setModalVisible(false);
            serverRoomForm.resetFields();
            setEditingServerRoom(null);
            setFloorOptions([]);
          }}
          width={600}
        >
          <Form
            form={serverRoomForm}
            layout="horizontal"
            labelCol={{ span: 5 }}
            wrapperCol={{ span: 19 }}
          >
            <Form.Item
              name="buildingId"
              label="所在楼宇"
              rules={[{ required: true, message: "请选择所在楼宇" }]}
            >
              <Select
                placeholder="请选择所在楼宇"
                className="user-form-input"
                options={buildingOptions.map((b) => ({ label: b.name, value: b.id }))}
                onChange={(value) => {
                  loadFloorOptions(value);
                  serverRoomForm.setFieldValue("floorId", undefined);
                }}
                onSearch={() => {}}
              />
            </Form.Item>
            <Form.Item
              name="floorId"
              label="楼层"
              rules={[{ required: true, message: "请选择楼层" }]}
            >
              <Select
                placeholder="请选择楼层"
                className="user-form-input"
                disabled={floorOptions.length === 0}
                options={floorOptions.map((f) => ({
                  label: `${f.floorNo} - ${f.name}`,
                  value: f.id,
                }))}
                onSearch={() => {}}
              />
            </Form.Item>
            <Form.Item
              name="name"
              label="机房名称"
              rules={[{ required: true, message: "请输入机房名称" }]}
            >
              <Input placeholder="请输入机房名称" className="user-form-input" />
            </Form.Item>
            <Form.Item
              name="status"
              label="状态"
              rules={[{ required: true, message: "请选择状态" }]}
            >
              <Select
                placeholder="请选择状态"
                className="user-form-input"
                options={[
                  { label: "正常", value: 0 },
                  { label: "停用", value: 1 },
                ]}
                onSearch={() => {}}
              />
            </Form.Item>
            <Form.Item name="remark" label="描述">
              <TextArea rows={3} placeholder="请输入描述" className="user-form-input" />
            </Form.Item>
          </Form>
        </Modal>
        <ExcelImport
          entityType="serverRoom"
          entityName="机房"
          visible={importVisible}
          onClose={() => setImportVisible(false)}
          onImportSuccess={handleImportSuccess}
        />
        <ExcelExport
          entityType="serverRoom"
          entityName="机房"
          visible={exportVisible}
          onClose={() => setExportVisible(false)}
          filters={exportFilters}
        />
      </Content>
    </Layout>
  );
};

export default ServerRoomManagement;
