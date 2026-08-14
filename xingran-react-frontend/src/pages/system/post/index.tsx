import { useState, useEffect, useCallback, useMemo } from "react";
import type { FC } from "react";
import {
  Table,
  Button,
  Space,
  Form,
  Input,
  InputNumber,
  Select,
  Popconfirm,
  Card,
  Row,
  Col,
  Statistic,
  Alert,
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
} from "@ant-design/icons";
import type { Post } from "@/types";
import { post } from "@/lib/api";
import { useTableManager } from "@/hooks/useTableManager";
import { usePagination } from "@/hooks/usePagination";
import { handleApiError, handleSuccess, isFormValidationError } from "@/utils/errorHandler";
import { createStatusColumn, createDateTimeColumn, createSorterMeta } from "@/utils/tableHelpers";
import ActionButtons from "@/components/shared/ActionButtons";
import { BaseEditModal } from "@/components/modal/BaseEditModal";

const { Option } = Select;
const { TextArea } = Input;

const PostManagement: FC = () => {
  const [statistics, setStatistics] = useState({
    total: 0,
    active: 0,
    inactive: 0,
  });

  // 使用全局分页 hook
  const { paginationProps, setCurrent, setPageSize, setTotal } = usePagination();

  // 服务端排序:field 对应后端 postAllowedSortFields 白名单 key
  const sorterMetas = useMemo(
    () => [
      createSorterMeta<Post>("postCode"),
      createSorterMeta<Post>("postName"),
      createSorterMeta<Post>("postSort"),
      createSorterMeta<Post>("status"),
      createSorterMeta<Post>("remark"),
      createSorterMeta<Post>("createdAt", "date"),
    ],
    []
  );

  const {
    loading,
    data: posts,
    total,
    selectedRowKeys,
    searchForm,
    editForm: postForm,
    editModalVisible: modalVisible,
    editingItem: editingPost,
    setSelectedRowKeys,
    setEditModalVisible: setModalVisible,
    setEditingItem: setEditingPost,
    handleSearch,
    handleReset,
    handleAdd,
    handleEdit,
    handleModalClose,
    loadData: loadPosts,
    resetSelection,
    getColumnSortOrder,
    handleTableChange,
  } = useTableManager<Post>(
    async (params) => {
      const result = (await post("/system/posts/list", params)) as {
        data: { list: Post[]; total: number };
      };
      return { list: result.data.list || [], total: result.data.total || 0 };
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

  // 加载统计数据(专用 COUNT 端点,不受 MaxPageSize=100 钳制)
  const loadStatistics = useCallback(async () => {
    try {
      const result = await post<{ total: number; active: number; inactive: number }>(
        "/system/posts/statistics"
      );
      setStatistics(result.data ?? { total: 0, active: 0, inactive: 0 });
    } catch (error) {
      handleApiError(error, "加载统计数据", false);
    }
  }, []);

  useEffect(() => {
    loadPosts();
    loadStatistics();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // 创建岗位
  const handleCreate = async () => {
    try {
      const values = await postForm.validateFields();
      if (editingPost) {
        await post(`/system/posts/${editingPost.id}/update`, values);
        handleSuccess("更新");
      } else {
        await post("/system/posts", values);
        handleSuccess("创建");
      }
      handleModalClose();
      loadPosts();
      loadStatistics();
    } catch (error: unknown) {
      if (isFormValidationError(error)) {
        return; // 表单验证错误
      }
      handleApiError(error, "操作");
    }
  };

  // 删除岗位
  const handleDelete = async (id: string) => {
    try {
      await post(`/system/posts/${id}/delete`, {});
      handleSuccess("删除");
      loadPosts();
      loadStatistics();
    } catch (error) {
      handleApiError(error, "删除");
    }
  };

  // 批量删除岗位
  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) {
      return;
    }
    try {
      await post("/system/posts/batch-delete", { ids: selectedRowKeys });
      handleSuccess("批量删除");
      resetSelection();
      loadPosts();
      loadStatistics();
    } catch (error) {
      handleApiError(error, "批量删除");
    }
  };

  // 打开编辑模态框
  const openModal = (record?: Post) => {
    if (record) {
      handleEdit(record);
      postForm.setFieldsValue(record);
    } else {
      handleAdd();
      postForm.setFieldsValue({ postSort: 0, status: 0 });
    }
  };

  // 表格列定义
  const columns: ColumnsType<Post> = [
    {
      title: "岗位编码",
      dataIndex: "postCode",
      key: "postCode",
      width: 150,
      sorter: true,
      sortOrder: getColumnSortOrder("postCode"),
    },
    {
      title: "岗位名称",
      dataIndex: "postName",
      key: "postName",
      width: 150,
      sorter: true,
      sortOrder: getColumnSortOrder("postName"),
    },
    {
      title: "显示排序",
      dataIndex: "postSort",
      key: "postSort",
      width: 100,
      sorter: true,
      sortOrder: getColumnSortOrder("postSort"),
    },
    createStatusColumn("status", {
      width: 100,
      sorter: true,
      sortOrder: getColumnSortOrder("status"),
    }),
    {
      title: "备注",
      dataIndex: "remark",
      key: "remark",
      ellipsis: true,
      sorter: true,
      sortOrder: getColumnSortOrder("remark"),
    },
    createDateTimeColumn("createdAt", {
      width: 180,
      sorter: true,
      sortOrder: getColumnSortOrder("createdAt"),
    }),
    {
      title: "操作",
      key: "action",
      render: (_, record) => {
        const actions = [
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
            render: () => (
              <Popconfirm
                title="确定要删除这个岗位吗？"
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

  return (
    <div>
      {/* 统计卡片 - 只在总数大于10时显示 */}
      {total > 10 && (
        <Row gutter={16} style={{ marginBottom: 16 }}>
          <Col span={8}>
            <Card>
              <Statistic
                title="总岗位数"
                value={statistics.total}
                prefix={<CheckCircleOutlined />}
              />
            </Card>
          </Col>
          <Col span={8}>
            <Card>
              <Statistic
                title="正常岗位"
                value={statistics.active}
                styles={{ content: { color: "var(--theme-success, #3f8600)" } }}
                prefix={<CheckCircleOutlined />}
              />
            </Card>
          </Col>
          <Col span={8}>
            <Card>
              <Statistic
                title="停用岗位"
                value={statistics.inactive}
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
            <Form.Item name="postCode" label="岗位编码">
              <Input
                placeholder="请输入岗位编码"
                allowClear
                className="user-form-input"
                style={{ width: 150 }}
              />
            </Form.Item>
            <Form.Item name="postName" label="岗位名称">
              <Input
                placeholder="请输入岗位名称"
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
                <Button onClick={handleReset}>重置</Button>
                <Button
                  icon={<ReloadOutlined />}
                  onClick={() => {
                    loadPosts();
                    loadStatistics();
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
                onClick={handleBatchDelete}
              >
                批量删除 ({selectedRowKeys.length})
              </Button>
            )}
            <Button type="primary" icon={<PlusOutlined />} onClick={() => openModal()}>
              新增岗位
            </Button>
          </Space>
        </div>
        {selectedRowKeys.length > 0 && (
          <Alert
            message={
              <span>
                已选择 <strong>{selectedRowKeys.length}</strong> 个岗位，
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
          columns={columns}
          dataSource={posts}
          loading={loading}
          rowKey="id"
          pagination={paginationProps}
          onChange={handleTableChange}
        />
      </Card>

      {/* 编辑模态框 */}
      <BaseEditModal
        title={editingPost ? "编辑岗位" : "新增岗位"}
        open={modalVisible}
        onOk={handleCreate}
        onCancel={() => {
          setModalVisible(false);
          postForm.resetFields();
          setEditingPost(null);
        }}
        width={600}
      >
        <Form form={postForm} layout="horizontal" labelCol={{ span: 4 }} wrapperCol={{ span: 20 }}>
          <Form.Item
            name="postCode"
            label="岗位编码"
            rules={[{ required: true, message: "请输入岗位编码" }]}
          >
            <Input
              placeholder="请输入岗位编码"
              disabled={!!editingPost}
              className="user-form-input"
            />
          </Form.Item>
          <Form.Item
            name="postName"
            label="岗位名称"
            rules={[{ required: true, message: "请输入岗位名称" }]}
          >
            <Input placeholder="请输入岗位名称" className="user-form-input" />
          </Form.Item>
          <Form.Item
            name="postSort"
            label="显示排序"
            rules={[{ required: true, message: "请输入显示排序" }]}
          >
            <InputNumber min={0} style={{ width: "100%" }} placeholder="请输入显示排序" />
          </Form.Item>
          <Form.Item name="status" label="状态" rules={[{ required: true, message: "请选择状态" }]}>
            <Select placeholder="请选择状态" className="user-form-input" onSearch={() => {}}>
              <Option value={0}>正常</Option>
              <Option value={1}>停用</Option>
            </Select>
          </Form.Item>
          <Form.Item name="remark" label="备注">
            <TextArea rows={3} placeholder="请输入备注" />
          </Form.Item>
        </Form>
      </BaseEditModal>
    </div>
  );
};

export default PostManagement;
