/**
 * 系统字典管理页面
 * System Dictionary Management Page
 */

import { useState, useEffect, useCallback, useMemo, useRef, type FC, type Key } from "react";
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
  Card,
  Row,
  Col,
  Statistic,
  Tabs,
  InputNumber,
  Switch,
  Alert,
} from "antd";
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  SearchOutlined,
  ReloadOutlined,
  CheckCircleOutlined,
  StopOutlined,
  LinkOutlined,
} from "@ant-design/icons";
import type { ColumnsType } from "antd/es/table";
import type { TableProps } from "antd";
import { formatDateTime } from "@/utils/datetime";

import type { DictType, DictData } from "@/types";
import ActionButtons from "@/components/shared/ActionButtons";
import { usePagination } from "@/hooks/usePagination";
import { useServerSort, resolveSorter } from "@/hooks/useServerSort";
import type { SortOrder } from "@/hooks/useServerSort";
import type { SorterMeta } from "@/utils/tableHelpers";
import { createSorterMeta } from "@/utils/tableHelpers";

// 导入提取的模块
import { STATUS_OPTIONS, renderStatusTag } from "./constants";
import { useDictData, useDictActions } from "./hooks";

const { Option } = Select;
const { TextArea } = Input;

// ==================== 表格列定义 ====================

interface DictTypeTableColumnsProps {
  openTypeModal: (record: DictType) => void;
  handleDeleteType: (id: string) => void;
  onDictNameClick: (dictType: string) => void;
  getColumnSortOrder: (field: string) => SortOrder | undefined;
}

function getDictTypeTableColumns(props: DictTypeTableColumnsProps): ColumnsType<DictType> {
  const { openTypeModal, handleDeleteType, onDictNameClick, getColumnSortOrder } = props;

  return [
    {
      title: "字典名称",
      dataIndex: "dictName",
      key: "dictName",
      width: 160,
      minWidth: 140,
      sorter: true,
      sortOrder: getColumnSortOrder("dictName"),
      render: (text: string, record: DictType) => (
        <Button
          type="link"
          icon={<LinkOutlined />}
          onClick={() => onDictNameClick(record.dictType)}
          style={{ padding: 0 }}
        >
          {text}
        </Button>
      ),
    },
    {
      title: "字典类型",
      dataIndex: "dictType",
      key: "dictType",
      width: 140,
      minWidth: 120,
      sorter: true,
      sortOrder: getColumnSortOrder("dictType"),
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 80,
      minWidth: 70,
      align: "center" as const,
      sorter: true,
      sortOrder: getColumnSortOrder("status"),
      render: (status: number) => renderStatusTag(status),
    },
    {
      title: "备注",
      dataIndex: "remark",
      key: "remark",
      width: 150,
      minWidth: 120,
      ellipsis: true,
    },
    {
      title: "创建时间",
      dataIndex: "createdAt",
      key: "createdAt",
      width: 180,
      sorter: true,
      sortOrder: getColumnSortOrder("createdAt"),
      render: (date: string) => formatDateTime(date),
    },
    {
      title: "操作",
      key: "action",
      width: 150,
      minWidth: 130,
      fixed: "right" as const,
      render: (_: unknown, record: DictType) => {
        const actions = [
          {
            key: "edit",
            label: "编辑",
            icon: <EditOutlined />,
            onClick: () => openTypeModal(record),
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
                onOk: () => handleDeleteType(record.id),
              });
            },
          },
        ];

        return <ActionButtons actions={actions} />;
      },
    },
  ];
}

interface DictDataTableColumnsProps {
  openDataModal: (record: DictData) => void;
  handleDeleteData: (id: string) => void;
  getColumnSortOrder: (field: string) => SortOrder | undefined;
}

function getDictDataTableColumns(props: DictDataTableColumnsProps): ColumnsType<DictData> {
  const { openDataModal, handleDeleteData, getColumnSortOrder } = props;

  return [
    {
      title: "字典标签",
      dataIndex: "dictLabel",
      key: "dictLabel",
      width: 140,
      minWidth: 120,
      sorter: true,
      sortOrder: getColumnSortOrder("dictLabel"),
    },
    {
      title: "字典键值",
      dataIndex: "dictValue",
      key: "dictValue",
      width: 120,
      minWidth: 100,
      sorter: true,
      sortOrder: getColumnSortOrder("dictValue"),
    },
    {
      title: "字典排序",
      dataIndex: "dictSort",
      key: "dictSort",
      width: 100,
      minWidth: 80,
      sorter: true,
      sortOrder: getColumnSortOrder("dictSort"),
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 80,
      minWidth: 70,
      align: "center" as const,
      sorter: true,
      sortOrder: getColumnSortOrder("status"),
      render: (status: number) => renderStatusTag(status),
    },
    {
      title: "备注",
      dataIndex: "remark",
      key: "remark",
      width: 150,
      minWidth: 120,
      ellipsis: true,
    },
    {
      title: "创建时间",
      dataIndex: "createdAt",
      key: "createdAt",
      width: 180,
      sorter: true,
      sortOrder: getColumnSortOrder("createdAt"),
      render: (date: string) => formatDateTime(date),
    },
    {
      title: "操作",
      key: "action",
      width: 150,
      minWidth: 130,
      fixed: "right" as const,
      render: (_: unknown, record: DictData) => {
        const actions = [
          {
            key: "edit",
            label: "编辑",
            icon: <EditOutlined />,
            onClick: () => openDataModal(record),
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
                onOk: () => handleDeleteData(record.id),
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

const DictManagement: FC = () => {
  const [searchForm] = Form.useForm();
  const [dataSearchForm] = Form.useForm();
  const [typeForm] = Form.useForm();
  const [dataForm] = Form.useForm();
  const [selectedRowKeys, setSelectedRowKeys] = useState<Key[]>([]);
  const location = useLocation();
  const [activeTab, setActiveTab] = usePersistedStateController<string>({
    keyPrefix: location.pathname,
    keySuffix: "activeTab",
    defaultValue: "type",
  });

  // 使用全局分页 hook
  const { paginationProps, setCurrent, setPageSize, setTotal } = usePagination();

  // 使用数据管理 Hook
  const {
    dictTypes,
    dictDataList,
    loading,
    total,
    selectedType,
    typeStatistics,
    dataStatistics,
    setSelectedType,
    loadDictTypes,
    loadDictData,
    loadTypeStatistics,
    loadDataStatistics,
  } = useDictData(
    searchForm,
    dataSearchForm,
    paginationProps.current ?? 1,
    paginationProps.pageSize ?? 10
  );

  // 使用操作管理 Hook
  const {
    editingType,
    editingData,
    typeModalVisible,
    dataModalVisible,
    setEditingType,
    setEditingData,
    setTypeModalVisible,
    setDataModalVisible,
    handleCreateType,
    handleDeleteType,
    handleBatchDeleteType,
    handleCreateData,
    handleDeleteData,
    handleBatchDeleteData,
    handleRefreshCache,
    openTypeModal,
    openDataModal,
  } = useDictActions({
    selectedType,
    loadDictTypes,
    loadDictData,
    loadTypeStatistics,
    loadDataStatistics,
  });

  // ==================== 服务端排序（字典类型表 + 字典数据表） ====================
  // field 对应后端 dictTypeAllowedSortFields / dictDataAllowedSortFields 白名单 key
  const typeSorterMetas = useMemo<Array<SorterMeta<DictType>>>(
    () => [
      createSorterMeta<DictType>("dictName"),
      createSorterMeta<DictType>("dictType"),
      createSorterMeta<DictType>("status", "number"),
      createSorterMeta<DictType>("createdAt", "date"),
    ],
    []
  );
  const dataSorterMetas = useMemo<Array<SorterMeta<DictData>>>(
    () => [
      createSorterMeta<DictData>("dictLabel"),
      createSorterMeta<DictData>("dictValue"),
      createSorterMeta<DictData>("dictSort", "number"),
      createSorterMeta<DictData>("status", "number"),
      createSorterMeta<DictData>("createdAt", "date"),
    ],
    []
  );

  const typeSort = useServerSort<DictType>({ sorterMetas: typeSorterMetas });
  const dataSort = useServerSort<DictData>({ sorterMetas: dataSorterMetas });

  // 列级 sortOrder（与 useTableManager.getColumnSortOrder 语义一致）
  const getTypeColumnSortOrder = useCallback(
    (field: string): SortOrder | undefined => {
      if (typeSort.orderByColumn !== String(field)) return undefined;
      return typeSort.sortOrder;
    },
    [typeSort.orderByColumn, typeSort.sortOrder]
  );
  const getDataColumnSortOrder = useCallback(
    (field: string): SortOrder | undefined => {
      if (dataSort.orderByColumn !== String(field)) return undefined;
      return dataSort.sortOrder;
    },
    [dataSort.orderByColumn, dataSort.sortOrder]
  );

  // 排序 ref：CRUD 刷新/初始化/搜索时携带最新排序，规避 setState 时序
  const typeSortRef = useRef<{ orderByColumn?: string; isAsc?: boolean }>({});
  const dataSortRef = useRef<{ orderByColumn?: string; isAsc?: boolean }>({});
  typeSortRef.current = { orderByColumn: typeSort.orderByColumn, isAsc: typeSort.isAsc };
  dataSortRef.current = { orderByColumn: dataSort.orderByColumn, isAsc: dataSort.isAsc };

  // 统一 Table onChange：分页 + 排序一起处理并 load。
  const handleTypeTableChange = useCallback<NonNullable<TableProps<DictType>["onChange"]>>(
    (pagination, filters, sorter) => {
      typeSort.handleTableChange(pagination, filters, sorter);
      const { orderByColumn, isAsc } = resolveSorter(sorter, typeSorterMetas);
      typeSortRef.current = { orderByColumn, isAsc };
      setCurrent(pagination.current ?? 1);
      setPageSize(pagination.pageSize ?? 10);
      loadDictTypes({
        current: pagination.current ?? 1,
        pageSize: pagination.pageSize ?? 10,
        ...(orderByColumn ? { orderByColumn, isAsc } : {}),
      });
    },
    [typeSort, typeSorterMetas, setCurrent, setPageSize, loadDictTypes]
  );

  const handleDataTableChange = useCallback<NonNullable<TableProps<DictData>["onChange"]>>(
    (pagination, filters, sorter) => {
      dataSort.handleTableChange(pagination, filters, sorter);
      const { orderByColumn, isAsc } = resolveSorter(sorter, dataSorterMetas);
      dataSortRef.current = { orderByColumn, isAsc };
      setCurrent(pagination.current ?? 1);
      setPageSize(pagination.pageSize ?? 10);
      loadDictData({
        current: pagination.current ?? 1,
        pageSize: pagination.pageSize ?? 10,
        ...(orderByColumn ? { orderByColumn, isAsc } : {}),
      });
    },
    [dataSort, dataSorterMetas, setCurrent, setPageSize, loadDictData]
  );

  // 搜索/刷新时保留当前排序（避免点搜索后排序丢失）
  const loadTypesWithSort = useCallback(() => {
    loadDictTypes(
      typeSortRef.current.orderByColumn
        ? { orderByColumn: typeSortRef.current.orderByColumn, isAsc: typeSortRef.current.isAsc }
        : {}
    );
  }, [loadDictTypes]);
  const loadDataWithSort = useCallback(() => {
    loadDictData(
      dataSortRef.current.orderByColumn
        ? { orderByColumn: dataSortRef.current.orderByColumn, isAsc: dataSortRef.current.isAsc }
        : {}
    );
  }, [loadDictData]);

  // 初始化加载
  useEffect(() => {
    if (activeTab === "type") {
      loadDictTypes({
        ...(typeSortRef.current.orderByColumn
          ? { orderByColumn: typeSortRef.current.orderByColumn, isAsc: typeSortRef.current.isAsc }
          : {}),
      });
      loadTypeStatistics();
    }
  }, [
    activeTab,
    paginationProps.current,
    paginationProps.pageSize,
    loadTypeStatistics,
    loadDictTypes,
  ]);

  useEffect(() => {
    if (activeTab === "data" && selectedType) {
      loadDictData({
        ...(dataSortRef.current.orderByColumn
          ? { orderByColumn: dataSortRef.current.orderByColumn, isAsc: dataSortRef.current.isAsc }
          : {}),
      });
      loadDataStatistics();
    }
  }, [
    activeTab,
    selectedType,
    paginationProps.current,
    paginationProps.pageSize,
    loadDataStatistics,
    loadDictData,
  ]);

  // 表格列
  const typeColumns = getDictTypeTableColumns({
    openTypeModal: (record) => openTypeModal(record, typeForm),
    handleDeleteType,
    getColumnSortOrder: getTypeColumnSortOrder,
    onDictNameClick: (dictType) => {
      setSelectedType(dictType);
      setActiveTab("data");
      setCurrent(1);
      loadDictData();
    },
  });

  const dataColumns = getDictDataTableColumns({
    openDataModal: (record) => openDataModal(record, dataForm),
    handleDeleteData,
    getColumnSortOrder: getDataColumnSortOrder,
  });

  return (
    <div>
      <Tabs
        activeKey={activeTab}
        onChange={setActiveTab}
        items={[
          {
            key: "type",
            label: "字典类型",
            children: (
              <>
                {/* 统计卡片 */}
                {typeStatistics.total > 10 && (
                  <Row gutter={16} style={{ marginBottom: 16 }}>
                    <Col span={8}>
                      <Card>
                        <Statistic
                          title="总类型数"
                          value={typeStatistics.total}
                          prefix={<CheckCircleOutlined />}
                        />
                      </Card>
                    </Col>
                    <Col span={8}>
                      <Card>
                        <Statistic
                          title="正常类型"
                          value={typeStatistics.active}
                          styles={{ content: { color: "var(--theme-success, #3f8600)" } }}
                          prefix={<CheckCircleOutlined />}
                        />
                      </Card>
                    </Col>
                    <Col span={8}>
                      <Card>
                        <Statistic
                          title="停用类型"
                          value={typeStatistics.inactive}
                          styles={{ content: { color: "var(--theme-error, #cf1322)" } }}
                          prefix={<StopOutlined />}
                        />
                      </Card>
                    </Col>
                  </Row>
                )}

                {/* 搜索表单和操作按钮 */}
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
                      <Form.Item name="dictName" label="字典名称">
                        <Input
                          placeholder="请输入字典名称"
                          allowClear
                          className="user-form-input"
                          style={{ width: 150 }}
                        />
                      </Form.Item>
                      <Form.Item name="dictType" label="字典类型">
                        <Input
                          placeholder="请输入字典类型"
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
                            type="primary"
                            icon={<SearchOutlined />}
                            onClick={loadTypesWithSort}
                          >
                            搜索
                          </Button>
                          <Button
                            icon={<ReloadOutlined />}
                            onClick={() => {
                              searchForm.resetFields();
                              loadTypesWithSort();
                              loadTypeStatistics();
                            }}
                          >
                            刷新
                          </Button>
                        </Space>
                      </Form.Item>
                    </Form>
                    <Space>
                      {selectedRowKeys.length > 0 && (
                        <Button
                          icon={<DeleteOutlined />}
                          style={{ color: "var(--theme-error, #ff4d4f)" }}
                          onClick={() => handleBatchDeleteType(selectedRowKeys, setSelectedRowKeys)}
                        >
                          批量删除 ({selectedRowKeys.length})
                        </Button>
                      )}
                      <Button
                        type="primary"
                        icon={<PlusOutlined />}
                        onClick={() => openTypeModal(undefined, typeForm)}
                      >
                        新增类型
                      </Button>
                      <Button icon={<ReloadOutlined />} onClick={handleRefreshCache}>
                        刷新缓存
                      </Button>
                    </Space>
                  </div>
                  {selectedRowKeys.length > 0 && (
                    <Alert
                      message={
                        <span>
                          已选择 <strong>{selectedRowKeys.length}</strong> 个字典类型，
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

                {/* 数据表格 */}
                <Card>
                  <Table
                    rowSelection={{
                      selectedRowKeys,
                      onChange: setSelectedRowKeys,
                    }}
                    columns={typeColumns}
                    dataSource={dictTypes}
                    loading={loading}
                    rowKey="id"
                    pagination={paginationProps}
                    onChange={handleTypeTableChange}
                  />
                </Card>
              </>
            ),
          },
          {
            key: "data",
            label: "字典数据",
            children: (
              <>
                {/* 统计卡片 */}
                {dataStatistics.total > 10 && (
                  <Row gutter={16} style={{ marginBottom: 16 }}>
                    <Col span={8}>
                      <Card>
                        <Statistic
                          title="总数据数"
                          value={dataStatistics.total}
                          prefix={<CheckCircleOutlined />}
                        />
                      </Card>
                    </Col>
                    <Col span={8}>
                      <Card>
                        <Statistic
                          title="正常数据"
                          value={dataStatistics.active}
                          styles={{ content: { color: "var(--theme-success, #3f8600)" } }}
                          prefix={<CheckCircleOutlined />}
                        />
                      </Card>
                    </Col>
                    <Col span={8}>
                      <Card>
                        <Statistic
                          title="停用数据"
                          value={dataStatistics.inactive}
                          styles={{ content: { color: "var(--theme-error, #cf1322)" } }}
                          prefix={<StopOutlined />}
                        />
                      </Card>
                    </Col>
                  </Row>
                )}

                {/* 搜索表单和操作按钮 */}
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
                    <Form form={dataSearchForm} layout="inline" style={{ flex: 1, minWidth: 0 }}>
                      <Form.Item label="字典类型">
                        <Select
                          placeholder="请选择字典类型"
                          value={selectedType}
                          onChange={setSelectedType}
                          style={{ width: 200 }}
                          onSearch={() => {}}
                        >
                          {dictTypes
                            .filter((t) => t.status === 0)
                            .map((t) => (
                              <Option key={t.id} value={t.dictType}>
                                {t.dictName}
                              </Option>
                            ))}
                        </Select>
                      </Form.Item>
                      <Form.Item name="dictLabel" label="字典标签">
                        <Input
                          placeholder="请输入字典标签"
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
                            type="primary"
                            icon={<SearchOutlined />}
                            onClick={loadDataWithSort}
                          >
                            搜索
                          </Button>
                          <Button
                            icon={<ReloadOutlined />}
                            onClick={() => {
                              dataSearchForm.resetFields();
                              loadDataWithSort();
                              loadDataStatistics();
                            }}
                          >
                            刷新
                          </Button>
                        </Space>
                      </Form.Item>
                    </Form>
                    <Space>
                      {selectedRowKeys.length > 0 && (
                        <Button
                          icon={<DeleteOutlined />}
                          style={{ color: "var(--theme-error, #ff4d4f)" }}
                          onClick={() => handleBatchDeleteData(selectedRowKeys, setSelectedRowKeys)}
                        >
                          批量删除 ({selectedRowKeys.length})
                        </Button>
                      )}
                      <Button
                        type="primary"
                        icon={<PlusOutlined />}
                        onClick={() => openDataModal(undefined, dataForm)}
                        disabled={!selectedType}
                      >
                        新增数据
                      </Button>
                    </Space>
                  </div>
                  {selectedRowKeys.length > 0 && (
                    <Alert
                      message={
                        <span>
                          已选择 <strong>{selectedRowKeys.length}</strong> 个字典数据，
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

                {/* 数据表格 */}
                <Card>
                  <Table
                    rowSelection={{
                      selectedRowKeys,
                      onChange: setSelectedRowKeys,
                    }}
                    columns={dataColumns}
                    dataSource={dictDataList}
                    loading={loading}
                    rowKey="id"
                    pagination={paginationProps}
                    onChange={handleDataTableChange}
                  />
                </Card>
              </>
            ),
          },
        ]}
      />

      {/* 字典类型编辑模态框 */}
      <Modal
        title={editingType ? "编辑字典类型" : "新增字典类型"}
        open={typeModalVisible}
        onOk={() => handleCreateType(typeForm)}
        onCancel={() => {
          setTypeModalVisible(false);
          typeForm.resetFields();
          setEditingType(null);
        }}
        width={600}
      >
        <Form form={typeForm} layout="horizontal" labelCol={{ span: 4 }} wrapperCol={{ span: 20 }}>
          <Form.Item
            name="dictName"
            label="字典名称"
            rules={[{ required: true, message: "请输入字典名称" }]}
          >
            <Input placeholder="请输入字典名称" className="user-form-input" />
          </Form.Item>
          <Form.Item
            name="dictType"
            label="字典类型"
            rules={[{ required: true, message: "请输入字典类型" }]}
          >
            <Input
              placeholder="请输入字典类型"
              disabled={!!editingType}
              className="user-form-input"
            />
          </Form.Item>
          <Form.Item name="status" label="状态" rules={[{ required: true }]}>
            <Select className="user-form-input" onSearch={() => {}}>
              {STATUS_OPTIONS.map((opt) => (
                <Option key={opt.value} value={opt.value}>
                  {opt.label}
                </Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="remark" label="备注">
            <TextArea rows={3} placeholder="请输入备注" />
          </Form.Item>
        </Form>
      </Modal>

      {/* 字典数据编辑模态框 */}
      <Modal
        title={editingData ? "编辑字典数据" : "新增字典数据"}
        open={dataModalVisible}
        onOk={() => handleCreateData(dataForm)}
        onCancel={() => {
          setDataModalVisible(false);
          dataForm.resetFields();
          setEditingData(null);
        }}
        width={600}
      >
        <Form form={dataForm} layout="horizontal" labelCol={{ span: 4 }} wrapperCol={{ span: 20 }}>
          <Form.Item
            name="dictSort"
            label="字典排序"
            rules={[{ required: true, message: "请输入字典排序" }]}
          >
            <InputNumber min={0} style={{ width: "100%" }} placeholder="请输入字典排序" />
          </Form.Item>
          <Form.Item
            name="dictLabel"
            label="字典标签"
            rules={[{ required: true, message: "请输入字典标签" }]}
          >
            <Input placeholder="请输入字典标签" className="user-form-input" />
          </Form.Item>
          <Form.Item
            name="dictValue"
            label="字典键值"
            rules={[{ required: true, message: "请输入字典键值" }]}
          >
            <Input placeholder="请输入字典键值" className="user-form-input" />
          </Form.Item>
          <Form.Item name="cssClass" label="CSS类名">
            <Input placeholder="请输入CSS类名" className="user-form-input" />
          </Form.Item>
          <Form.Item name="listClass" label="列表类名">
            <Input placeholder="请输入列表类名" className="user-form-input" />
          </Form.Item>
          <Form.Item name="isDefault" label="是否默认" valuePropName="checked">
            <Switch checkedChildren="是" unCheckedChildren="否" />
          </Form.Item>
          <Form.Item name="status" label="状态" rules={[{ required: true }]}>
            <Select className="user-form-input" onSearch={() => {}}>
              {STATUS_OPTIONS.map((opt) => (
                <Option key={opt.value} value={opt.value}>
                  {opt.label}
                </Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="remark" label="备注">
            <TextArea rows={3} placeholder="请输入备注" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default DictManagement;
