/**
 * WorkOrder 工单管理页面
 */

import { useState, useEffect, useCallback, useMemo, type FC } from "react";
import {
  Button,
  Form,
  Input,
  Select,
  Table,
  Modal,
  Space,
  Tag,
  App,
  Popconfirm,
  Card,
  Statistic,
  Row,
  Col,
  Drawer,
  Timeline,
  Radio,
  TreeSelect,
  Alert,
} from "antd";
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  SearchOutlined,
  ReloadOutlined,
  FileTextOutlined,
  CheckCircleOutlined,
  EyeOutlined,
} from "@ant-design/icons";
import type { ColumnsType, TablePaginationConfig } from "antd/es/table";

import {
  getWorkOrderList,
  createWorkOrder,
  updateWorkOrder,
  deleteWorkOrder,
  batchDeleteWorkOrders,
  assignToTodayDuty,
  updateWorkOrderStatus,
  addWorkOrderComment,
  getWorkOrderComments,
  type WorkOrder,
  type WorkOrderCreateRequest,
  type WorkOrderUpdateRequest,
  WorkOrderStatus,
  WorkOrderPriority,
  WorkOrderType,
} from "@/lib/workorderApi";
import { getEnabledWorkOrderCategories, type WorkOrderCategory } from "@/lib/workorderApi";
import { getUserList, getDeptList, type SimpleUser, type SimpleDept } from "@/lib/workorderApi";
import ActionButtons from "@/components/shared/ActionButtons";

// 导入提取的常量、工具和 Hook
import { STATUS_CONFIG, PRIORITY_CONFIG, TYPE_CONFIG } from "./constants";
import { formatDateTime, buildCategoryTree } from "@/lib/workorderApi";
import { useWorkOrderData } from "./hooks/useWorkOrderData";
import { useWorkOrderModals, type Comment, type HistoryItem } from "./hooks/useWorkOrderModals";
import { usePagination } from "@/hooks/usePagination";
import { useServerSort } from "@/hooks/useServerSort";
import { createSorterMeta } from "@/utils/tableHelpers";

const { Option } = Select;
const { TextArea } = Input;

// ==================== 表格列定义 ====================

interface WorkOrderTableColumnsProps {
  handleViewDetail: (record: WorkOrder) => void;
  handleEdit: (record: WorkOrder) => void;
  handleAssignToTodayDuty: (id: string) => void;
  handleDelete: (id: string) => void;
  current: number;
  pageSize: number;
  /** 由 useServerSort 注入,返回字段当前排序方向 */
  getSortOrder?: (field: string) => "ascend" | "descend" | null;
}

function getWorkOrderTableColumns(props: WorkOrderTableColumnsProps): ColumnsType<WorkOrder> {
  const {
    handleViewDetail,
    handleEdit,
    handleAssignToTodayDuty,
    handleDelete,
    current,
    pageSize,
    getSortOrder,
  } = props;

  return [
    {
      title: "序号",
      key: "index",
      width: 60,
      render: (_: unknown, __: unknown, index: number) => (current - 1) * pageSize + index + 1,
    },
    {
      title: "工单编号",
      dataIndex: "workOrderNo",
      key: "workOrderNo",
      width: 150,
    },
    {
      title: "标题",
      dataIndex: "title",
      key: "title",
      width: 200,
      ellipsis: true,
    },
    {
      title: "类型",
      dataIndex: "type",
      key: "type",
      width: 80,
      render: (type: WorkOrderType) => TYPE_CONFIG[type]?.text || type,
    },
    {
      title: "优先级",
      dataIndex: "priority",
      key: "priority",
      width: 80,
      sorter: true,
      sortOrder: getSortOrder?.("priority"),
      render: (priority: WorkOrderPriority) => (
        <Tag color={PRIORITY_CONFIG[priority]?.color}>{PRIORITY_CONFIG[priority]?.text}</Tag>
      ),
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 90,
      render: (status: WorkOrderStatus) => (
        <Tag color={STATUS_CONFIG[status]?.color}>{STATUS_CONFIG[status]?.text}</Tag>
      ),
    },
    {
      title: "报告人",
      key: "submitter",
      width: 100,
      render: (_: unknown, record: WorkOrder) =>
        record.submitter?.nickName || record.submitter?.username || "-",
    },
    {
      title: "处理人",
      key: "assignee",
      width: 100,
      render: (_: unknown, record: WorkOrder) =>
        record.assignee?.nickName || record.assignee?.username || "-",
    },
    {
      title: "部门",
      key: "department",
      width: 120,
      render: (_: unknown, record: WorkOrder) => record.department?.deptName || "-",
    },
    {
      title: "创建时间",
      dataIndex: "createdAt",
      key: "createdAt",
      width: 180,
      render: (date: string) => formatDateTime(date),
    },
    {
      title: "操作",
      key: "action",
      width: 100,
      fixed: "right",
      render: (_: unknown, record: WorkOrder) => {
        const actions = [
          {
            key: "view",
            label: "详情",
            icon: <EyeOutlined />,
            onClick: () => handleViewDetail(record),
          },
          {
            key: "edit",
            label: "编辑",
            icon: <EditOutlined />,
            onClick: () => handleEdit(record),
          },
          ...(record.status === WorkOrderStatus.Pending
            ? [
                {
                  key: "assign",
                  label: "分配值班",
                  icon: <CheckCircleOutlined />,
                  onClick: () => handleAssignToTodayDuty(record.id),
                },
              ]
            : []),
          {
            key: "delete",
            label: "删除",
            icon: <DeleteOutlined />,
            danger: true,
            render: () => (
              <Popconfirm
                title="确定要删除吗？"
                onConfirm={() => handleDelete(record.id)}
                okText="确定"
                cancelText="取消"
              >
                <Button
                  type="link"
                  icon={<DeleteOutlined />}
                  style={{ color: "var(--theme-error, #ff4d4f)" }}
                  size="small"
                >
                  删除
                </Button>
              </Popconfirm>
            ),
          },
        ];

        return <ActionButtons actions={actions} />;
      },
    },
  ];
}

// ==================== 主组件 ====================

const WorkOrderPage: FC = () => {
  const { message } = App.useApp();
  const [form] = Form.useForm();
  const [editForm] = Form.useForm();
  const [commentForm] = Form.useForm();

  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);

  // 使用全局分页 hook
  const { paginationProps, setCurrent, setPageSize } = usePagination();

  // 使用自定义 Hooks
  const { loading, dataSource, total, stats, users, depts, categories, fetchList } =
    useWorkOrderData({ form });

  const {
    modalVisible,
    detailDrawerVisible,
    editingRecord,
    selectedRecord,
    comments,
    history,
    commentInternal,
    setModalVisible,
    setDetailDrawerVisible,
    setComments,
    setCommentInternal,
    openAddModal,
    openEditModal,
    openDetailDrawer,
  } = useWorkOrderModals();

  // ==================== 表格操作 ====================

  // 服务端排序:field 与 columns.dataIndex 对齐
  const sorterMetas = useMemo(
    () => [
      createSorterMeta<WorkOrder>("workOrderNo"),
      createSorterMeta<WorkOrder>("title"),
      createSorterMeta<WorkOrder>("type"),
      createSorterMeta<WorkOrder>("priority"),
      createSorterMeta<WorkOrder>("status"),
      createSorterMeta<WorkOrder>("createdAt", "date"),
      createSorterMeta<WorkOrder>("updatedAt", "date"),
    ],
    []
  );
  const {
    orderByColumn,
    isAsc,
    handleTableChange: handleWoSortChange,
    sortOrder: woSortOrder,
  } = useServerSort<WorkOrder>({
    sorterMetas,
  });

  const handleTableChange = useCallback(
    (pagination: TablePaginationConfig, _filters?: unknown, sorter?: unknown) => {
      handleWoSortChange(pagination, _filters as never, sorter as never);
      const sortParams = orderByColumn ? { orderByColumn, isAsc } : undefined;
      fetchList(pagination.current || 1, pagination.pageSize || 10, sortParams);
    },
    [fetchList, handleWoSortChange, orderByColumn, isAsc]
  );

  const handleSearch = useCallback(() => {
    fetchList(1, paginationProps.pageSize);
  }, [fetchList, paginationProps]);

  const handleReset = useCallback(() => {
    form.resetFields();
    fetchList(1, paginationProps.pageSize);
  }, [form, fetchList, paginationProps]);

  // ==================== CRUD 操作 ====================

  const handleAdd = useCallback(() => {
    openAddModal();
    editForm.resetFields();
    editForm.setFieldsValue({
      type: WorkOrderType.Fault,
      priority: WorkOrderPriority.Medium,
      isAutoAssigned: true,
    });
  }, [openAddModal, editForm]);

  const handleEdit = useCallback(
    (record: WorkOrder) => {
      openEditModal(record, editForm);
    },
    [openEditModal, editForm]
  );

  const handleDelete = useCallback(
    async (id: string) => {
      try {
        await deleteWorkOrder(id);
        message.success("删除成功");
        fetchList(paginationProps.current, paginationProps.pageSize);
      } catch (error) {
        message.error("删除失败");
      }
    },
    [fetchList, message]
  );

  const handleBatchDelete = useCallback(async () => {
    if (selectedRowKeys.length === 0) {
      message.warning("请选择要删除的工单");
      return;
    }
    try {
      await batchDeleteWorkOrders(selectedRowKeys as string[]);
      message.success("批量删除成功");
      setSelectedRowKeys([]);
      fetchList(paginationProps.current, paginationProps.pageSize);
    } catch (error) {
      message.error("批量删除失败");
    }
  }, [selectedRowKeys, fetchList, message]);

  const handleAssignToTodayDuty = useCallback(
    async (id: string) => {
      try {
        const result = await assignToTodayDuty(id);
        fetchList(paginationProps.current, paginationProps.pageSize);
      } catch (error: unknown) {
        console.error("分配失败:", error);
      }
    },
    [fetchList]
  );

  const handleModalOk = useCallback(async () => {
    try {
      const values = await editForm.validateFields();

      if (editingRecord) {
        const updateData: WorkOrderUpdateRequest = {
          title: values.title,
          categoryId: values.categoryId,
          type: values.type,
          priority: values.priority,
          description: values.description,
          solution: values.solution,
          deptId: values.deptId,
          assigneeId: values.assigneeId,
          expectedResolveAt: values.expectedResolveAt,
        };
        await updateWorkOrder(editingRecord.id, updateData);
        message.success("更新成功");
      } else {
        const createData: WorkOrderCreateRequest = {
          title: values.title,
          categoryId: values.categoryId,
          type: values.type,
          priority: values.priority,
          description: values.description,
          deptId: values.deptId,
          expectedResolveAt: values.expectedResolveAt,
          isAutoAssigned: values.isAutoAssigned,
          assigneeId: values.assigneeId,
        };
        await createWorkOrder(createData);
        message.success("创建成功");
      }
      setModalVisible(false);
      fetchList();
    } catch (error: unknown) {
      if (error && typeof error === "object" && "errorFields" in error) {
        return;
      }
      message.error(editingRecord ? "更新失败" : "创建失败");
    }
  }, [editForm, editingRecord, fetchList, setModalVisible, message]);

  const handleAddComment = useCallback(async () => {
    try {
      const values = await commentForm.validateFields();
      if (!selectedRecord) return;

      await addWorkOrderComment(selectedRecord.id, {
        content: values.content,
        isInternal: commentInternal,
      });
      message.success("评论添加成功");
      commentForm.resetFields();

      const commentsResult = await getWorkOrderComments(selectedRecord.id);
      setComments(commentsResult.data || []);
    } catch (error) {
      message.error("评论添加失败");
    }
  }, [commentForm, selectedRecord, commentInternal, setComments, message]);

  const handleStatusChange = useCallback(
    async (status: number) => {
      if (!selectedRecord) return;
      try {
        await updateWorkOrderStatus(selectedRecord.id, { status });
        message.success("状态更新成功");
        fetchList();
        await openDetailDrawer(selectedRecord);
      } catch (error: unknown) {
        if (
          error &&
          typeof error === "object" &&
          "message" in error &&
          typeof error.message === "string"
        ) {
          message.error(error.message);
        } else {
          message.error("状态更新失败");
        }
      }
    },
    [fetchList, openDetailDrawer, selectedRecord, message]
  );

  // 表格列 - 使用 useMemo 避免重复创建
  const columns = useMemo(
    () =>
      getWorkOrderTableColumns({
        handleViewDetail: openDetailDrawer,
        handleEdit,
        handleAssignToTodayDuty,
        handleDelete,
        current: paginationProps.current ?? 1,
        pageSize: paginationProps.pageSize ?? 10,
        getSortOrder: (field) =>
          orderByColumn === field ? ((woSortOrder ?? null) as "ascend" | "descend" | null) : null,
      }),
    [
      openDetailDrawer,
      handleEdit,
      handleAssignToTodayDuty,
      handleDelete,
      paginationProps,
      orderByColumn,
      woSortOrder,
    ]
  );

  // 分类树 - 使用 useMemo 避免重复构建
  const categoryTree = useMemo(() => buildCategoryTree(categories), [categories]);

  return (
    <div className="p-6">
      {/* 统计卡片 */}
      {stats.total > 10 && (
        <Card title={null} className="mb-4">
          <Row gutter={16}>
            <Col span={4}>
              <Statistic title="总工单" value={stats.total} prefix={<FileTextOutlined />} />
            </Col>
            <Col span={4}>
              <Statistic
                title="待处理"
                value={stats.pending}
                styles={{ content: { color: "var(--theme-warning, #faad14)" } }}
              />
            </Col>
            <Col span={4}>
              <Statistic
                title="处理中"
                value={stats.processing}
                styles={{ content: { color: "var(--theme-info, #1890ff)" } }}
              />
            </Col>
            <Col span={4}>
              <Statistic
                title="已完成"
                value={stats.completed}
                styles={{ content: { color: "var(--theme-success, #52c41a)" } }}
              />
            </Col>
            <Col span={4}>
              <Statistic
                title="已关闭"
                value={stats.closed}
                styles={{ content: { color: "var(--theme-text-tertiary, #8c8c8c)" } }}
              />
            </Col>
            <Col span={4}>
              <Button
                type="primary"
                danger
                disabled={selectedRowKeys.length === 0}
                onClick={handleBatchDelete}
              >
                批量删除 ({selectedRowKeys.length})
              </Button>
            </Col>
          </Row>
        </Card>
      )}

      {/* 筛选表单和操作按钮 */}
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
          <Form form={form} layout="inline" style={{ flex: 1, minWidth: 0 }}>
            <Form.Item name="workOrderNo" label="工单编号">
              <Input
                placeholder="请输入工单编号"
                allowClear
                className="user-form-input"
                style={{ width: 150 }}
              />
            </Form.Item>
            <Form.Item name="title" label="标题">
              <Input
                placeholder="请输入标题"
                allowClear
                className="user-form-input"
                style={{ width: 150 }}
              />
            </Form.Item>
            <Form.Item name="categoryId" label="分类">
              <TreeSelect
                placeholder="请选择分类"
                allowClear
                className="user-form-input"
                style={{ width: 150 }}
                treeData={categoryTree}
                showSearch
                treeNodeFilterProp="title"
              />
            </Form.Item>
            <Form.Item name="type" label="类型">
              <Select
                placeholder="请选择类型"
                allowClear
                className="user-form-input"
                style={{ width: 100 }}
                onSearch={() => {}}
              >
                {Object.entries(TYPE_CONFIG).map(([key, { text }]) => (
                  <Option key={key} value={key}>
                    {text}
                  </Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item name="priority" label="优先级">
              <Select
                placeholder="请选择优先级"
                allowClear
                className="user-form-input"
                style={{ width: 100 }}
                onSearch={() => {}}
              >
                {Object.entries(PRIORITY_CONFIG).map(([key, { text }]) => (
                  <Option key={key} value={Number(key)}>
                    {text}
                  </Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item name="status" label="状态">
              <Select
                placeholder="请选择状态"
                allowClear
                className="user-form-input"
                style={{ width: 100 }}
                onSearch={() => {}}
              >
                {Object.entries(STATUS_CONFIG).map(([key, { text }]) => (
                  <Option key={key} value={Number(key)}>
                    {text}
                  </Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item name="assigneeId" label="处理人">
              <Select
                placeholder="请选择处理人"
                allowClear
                className="user-form-input"
                showSearch
                style={{ width: 120 }}
                optionFilterProp="children"
                onSearch={() => {}}
              >
                {users
                  .filter((user) => user.id)
                  .map((user) => (
                    <Option key={user.id} value={user.id}>
                      {user.nickName || user.username}
                    </Option>
                  ))}
              </Select>
            </Form.Item>
            <Form.Item>
              <Space>
                <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>
                  查询
                </Button>
                <Button icon={<ReloadOutlined />} onClick={handleReset}>
                  重置
                </Button>
              </Space>
            </Form.Item>
          </Form>
          <Space>
            {selectedRowKeys.length > 0 && (
              <Button
                type="primary"
                onClick={handleBatchDelete}
                style={{ color: "var(--theme-error, #ff4d4f)", borderColor: "#ff4d4f" }}
              >
                批量删除 ({selectedRowKeys.length})
              </Button>
            )}
            <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
              新增工单
            </Button>
          </Space>
        </div>
        {selectedRowKeys.length > 0 && (
          <Alert
            message={
              <span>
                已选择 <strong>{selectedRowKeys.length}</strong> 个工单，
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

      {/* 工单表格 */}
      <Card>
        <Table
          rowKey="id"
          columns={columns}
          dataSource={dataSource}
          loading={loading}
          scroll={{ x: 1400 }}
          rowSelection={{
            selectedRowKeys,
            onChange: setSelectedRowKeys,
          }}
          pagination={paginationProps}
          onChange={handleTableChange}
        />
      </Card>

      {/* 新增/编辑弹窗 */}
      <Modal
        title={editingRecord ? "编辑工单" : "新增工单"}
        open={modalVisible}
        onOk={handleModalOk}
        onCancel={() => setModalVisible(false)}
        width={700}
        destroyOnHidden
      >
        <Form
          form={editForm}
          layout="horizontal"
          labelCol={{ span: 4 }}
          wrapperCol={{ span: 20 }}
          preserve={false}
        >
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="title"
                label="工单标题"
                rules={[{ required: true, message: "请输入工单标题" }]}
              >
                <Input placeholder="请输入工单标题" className="user-form-input" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="categoryId"
                label="工单分类"
                rules={[{ required: true, message: "请选择工单分类" }]}
              >
                <TreeSelect
                  placeholder="请选择工单分类"
                  treeData={categoryTree}
                  showSearch
                  treeNodeFilterProp="title"
                  className="user-form-input"
                />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="type"
                label="工单类型"
                rules={[{ required: true, message: "请选择工单类型" }]}
              >
                <Select
                  placeholder="请选择工单类型"
                  className="user-form-input"
                  onSearch={() => {}}
                >
                  {Object.entries(TYPE_CONFIG).map(([key, { text }]) => (
                    <Option key={key} value={key}>
                      {text}
                    </Option>
                  ))}
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="priority"
                label="优先级"
                rules={[{ required: true, message: "请选择优先级" }]}
              >
                <Select placeholder="请选择优先级" className="user-form-input" onSearch={() => {}}>
                  {Object.entries(PRIORITY_CONFIG).map(([key, { text }]) => (
                    <Option key={key} value={Number(key)}>
                      {text}
                    </Option>
                  ))}
                </Select>
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="deptId" label="所属部门">
                <Select
                  placeholder="请选择部门"
                  allowClear
                  className="user-form-input"
                  onSearch={() => {}}
                >
                  {depts
                    .filter((dept) => dept.id)
                    .map((dept) => (
                      <Option key={dept.id} value={dept.id}>
                        {dept.deptName}
                      </Option>
                    ))}
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="assigneeId" label="处理人">
                <Select
                  placeholder="请选择处理人"
                  allowClear
                  showSearch
                  optionFilterProp="children"
                  className="user-form-input"
                  onSearch={() => {}}
                >
                  {users
                    .filter((user) => user.id)
                    .map((user) => (
                      <Option key={user.id} value={user.id}>
                        {user.nickName || user.username}
                      </Option>
                    ))}
                </Select>
              </Form.Item>
            </Col>
          </Row>
          <Form.Item name="expectedResolveAt" label="期望解决时间">
            <Input type="datetime-local" className="user-form-input" />
          </Form.Item>
          <Form.Item name="description" label="工单描述">
            <TextArea rows={4} placeholder="请输入工单描述" />
          </Form.Item>
          {editingRecord && (
            <Form.Item name="solution" label="解决方案">
              <TextArea rows={4} placeholder="请输入解决方案" />
            </Form.Item>
          )}
        </Form>
      </Modal>

      {/* 详情抽屉 */}
      <Drawer
        title="工单详情"
        placement="right"
        size="large"
        open={detailDrawerVisible}
        onClose={() => setDetailDrawerVisible(false)}
      >
        {selectedRecord && (
          <div>
            <Card title="基本信息" size="small" className="mb-4">
              <p>
                <strong>工单编号：</strong>
                {selectedRecord.workOrderNo}
              </p>
              <p>
                <strong>标题：</strong>
                {selectedRecord.title}
              </p>
              <p>
                <strong>类型：</strong>
                {TYPE_CONFIG[selectedRecord.type]?.text}
              </p>
              <p>
                <strong>优先级：</strong>
                <Tag color={PRIORITY_CONFIG[selectedRecord.priority]?.color}>
                  {PRIORITY_CONFIG[selectedRecord.priority]?.text}
                </Tag>
              </p>
              <p>
                <strong>状态：</strong>
                <Tag color={STATUS_CONFIG[selectedRecord.status]?.color}>
                  {STATUS_CONFIG[selectedRecord.status]?.text}
                </Tag>
              </p>
              <p>
                <strong>报告人：</strong>
                {selectedRecord.submitter?.nickName || selectedRecord.submitter?.username}
              </p>
              <p>
                <strong>处理人：</strong>
                {selectedRecord.assignee?.nickName || selectedRecord.assignee?.username || "-"}
              </p>
              <p>
                <strong>部门：</strong>
                {selectedRecord.department?.deptName || "-"}
              </p>
              <p>
                <strong>创建时间：</strong>
                {formatDateTime(selectedRecord.createdAt)}
              </p>
            </Card>

            <Card title="工单描述" size="small" className="mb-4">
              <p style={{ whiteSpace: "pre-wrap" }}>{selectedRecord.description || "暂无描述"}</p>
            </Card>

            {selectedRecord.solution && (
              <Card title="解决方案" size="small" className="mb-4">
                <p style={{ whiteSpace: "pre-wrap" }}>{selectedRecord.solution}</p>
              </Card>
            )}

            <Card title="状态操作" size="small" className="mb-4">
              <Space>
                <Button
                  type={
                    selectedRecord.status === WorkOrderStatus.Processing ? "primary" : "default"
                  }
                  size="small"
                  onClick={() => handleStatusChange(WorkOrderStatus.Processing)}
                >
                  开始处理
                </Button>
                <Button
                  type={selectedRecord.status === WorkOrderStatus.Completed ? "primary" : "default"}
                  size="small"
                  onClick={() => handleStatusChange(WorkOrderStatus.Completed)}
                >
                  标记完成
                </Button>
                <Button
                  type={selectedRecord.status === WorkOrderStatus.Closed ? "primary" : "default"}
                  size="small"
                  onClick={() => handleStatusChange(WorkOrderStatus.Closed)}
                >
                  关闭工单
                </Button>
              </Space>
            </Card>

            <Card title="添加评论" size="small" className="mb-4">
              <Form form={commentForm} layout="vertical">
                <Form.Item name="content" rules={[{ required: true, message: "请输入评论内容" }]}>
                  <TextArea rows={3} placeholder="请输入评论内容" />
                </Form.Item>
                <Form.Item>
                  <Space>
                    <Radio.Group
                      value={commentInternal}
                      onChange={(e) => setCommentInternal(e.target.value)}
                    >
                      <Radio value={false}>公开评论</Radio>
                      <Radio value={true}>内部评论</Radio>
                    </Radio.Group>
                    <Button type="primary" onClick={handleAddComment}>
                      添加评论
                    </Button>
                  </Space>
                </Form.Item>
              </Form>
            </Card>

            <Card title="评论记录" size="small" className="mb-4">
              {comments.length === 0 ? (
                <p className="text-gray-400">暂无评论</p>
              ) : (
                <Timeline>
                  {comments.map((comment: Comment) => (
                    <Timeline.Item key={comment.id}>
                      <div>
                        <strong>{comment.user?.nickName || comment.user?.username}</strong>
                        {comment.isInternal && (
                          <Tag color="orange" className="ml-2">
                            内部
                          </Tag>
                        )}
                        <span className="text-gray-400 text-sm ml-2">
                          {formatDateTime(comment.createdAt, "YYYY-MM-DD HH:mm")}
                        </span>
                      </div>
                      <p className="mt-1">{comment.content}</p>
                    </Timeline.Item>
                  ))}
                </Timeline>
              )}
            </Card>

            <Card title="操作历史" size="small">
              {history.length === 0 ? (
                <p className="text-gray-400">暂无历史</p>
              ) : (
                <Timeline>
                  {history.map((item: HistoryItem) => (
                    <Timeline.Item key={item.id}>
                      <div>
                        <strong>{item.operator?.nickName || item.operator?.username}</strong>
                        <span className="text-gray-400 text-sm ml-2">
                          {formatDateTime(item.createdAt, "YYYY-MM-DD HH:mm")}
                        </span>
                      </div>
                      <p className="mt-1">
                        {item.remark || item.action}
                        {item.oldValue && item.newValue && (
                          <span className="text-gray-500">
                            （{item.oldValue} → {item.newValue}）
                          </span>
                        )}
                      </p>
                    </Timeline.Item>
                  ))}
                </Timeline>
              )}
            </Card>
          </div>
        )}
      </Drawer>
    </div>
  );
};

export default WorkOrderPage;
