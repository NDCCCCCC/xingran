/**
 * RPA 任务管理页面
 */

import { useState, useCallback, useEffect, useMemo } from "react";
import type { FC } from "react";
import { App, Table, Button, Space, Form, Input, Select, Card, Modal, Layout } from "antd";
import {
  PlusOutlined,
  SearchOutlined,
  ReloadOutlined,
  PlayCircleOutlined,
} from "@ant-design/icons";
import type { Task } from "@/types/rpa";
import type { PageResponse } from "@/types/base";
import { useTableManager } from "@/hooks/useTableManager";
import { useServerSort } from "@/hooks/useServerSort";
import { createSorterMeta } from "@/utils/tableHelpers";
import { usePagination } from "@/hooks/usePagination";
import { handleSuccess as showSuccessMessage } from "@/utils/errorHandler";
import { getTaskColumns } from "./columns";
import { TaskEditModal } from "./modals";
import { TASK_STATUS_OPTIONS } from "../constants";
import { post } from "@/lib/api";

const { Option } = Select;
const { Content } = Layout;

const TaskManagement: FC = () => {
  const { message } = App.useApp();
  const { paginationProps, setCurrent, setPageSize, setTotal } = usePagination();

  // 服务端排序:field 必须与 columns 的 dataIndex 一致
  const sorterMetas = useMemo(
    () => [
      createSorterMeta<Task>("taskName"),
      createSorterMeta<Task>("priority"),
      createSorterMeta<Task>("status"),
      createSorterMeta<Task>("createdAt", "date"),
    ],
    []
  );
  const {
    orderByColumn,
    isAsc,
    handleTableChange: handleTaskSortChange,
    sortOrder: taskSortOrder,
  } = useServerSort<Task>({
    sorterMetas,
  });

  const {
    loading,
    data: tasks,
    total,
    selectedRowKeys,
    searchForm,
    editForm: taskForm,
    editModalVisible: modalVisible,
    editingItem: editingTask,
    setSelectedRowKeys,
    setEditModalVisible: setModalVisible,
    handleSearch,
    handleReset,
    handleAdd,
    handleEdit,
    loadData: loadTasks,
    resetSelection,
  } = useTableManager<Task>(
    async (params) => {
      const result = await post<PageResponse<Task>>("/rpa/tasks/list", {
        ...params,
        ...(orderByColumn ? { orderByColumn, isAsc } : {}),
      });
      return {
        list: result.data?.list || [],
        total: result.data?.total || 0,
      };
    },
    {
      externalPagination: {
        current: paginationProps.current ?? 1,
        pageSize: paginationProps.pageSize ?? 10,
        setCurrent,
        setPageSize,
        setTotal,
      },
    }
  );

  const [executeLoading, setExecuteLoading] = useState(false);

  const refreshData = useCallback(() => {
    loadTasks();
  }, [loadTasks]);

  useEffect(() => {
    loadTasks();
  }, [loadTasks]);

  const handleDelete = useCallback(
    async (id: string) => {
      try {
        await post(`/rpa/tasks/${id}/delete`, {});
        showSuccessMessage("删除");
        refreshData();
      } catch (_error) {
        // 错误已在 errorHandler 中处理
      }
    },
    [refreshData]
  );

  const handleExecute = useCallback(
    async (id: string, name: string) => {
      Modal.confirm({
        title: "确认执行",
        content: `确定要执行任务 "${name}" 吗？`,
        okText: "确定",
        cancelText: "取消",
        onOk: async () => {
          setExecuteLoading(true);
          try {
            await post(`/rpa/tasks/${id}/execute`, {});
            message.success("任务已提交执行");
            refreshData();
          } catch (error) {
            // 错误已在 errorHandler 中处理
          } finally {
            setExecuteLoading(false);
          }
        },
      });
    },
    [refreshData]
  );

  const handleModalOk = useCallback(
    async (values: Record<string, unknown>) => {
      if (editingTask) {
        await post(`/rpa/tasks/${editingTask.id}/update`, values);
        showSuccessMessage("更新");
      } else {
        await post("/rpa/tasks", values);
        showSuccessMessage("创建");
      }
      setModalVisible(false);
      taskForm.resetFields();
      refreshData();
    },
    [editingTask, setModalVisible, taskForm, refreshData]
  );

  const handleModalCancel = useCallback(() => {
    setModalVisible(false);
    taskForm.resetFields();
  }, [setModalVisible, taskForm]);

  const columns = useMemo(
    () =>
      getTaskColumns({
        handleEdit: (record: Task) => {
          handleEdit(record);
        },
        handleDelete,
        handleExecute,
        getSortOrder: (field) =>
          orderByColumn === field ? ((taskSortOrder ?? null) as "ascend" | "descend" | null) : null,
      }),
    [handleEdit, handleDelete, handleExecute, orderByColumn, taskSortOrder]
  );

  return (
    <Layout style={{ background: "#000", minHeight: "calc(100vh - 64px)" }}>
      <Content style={{ background: "#fff", padding: 16 }}>
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
              <Form.Item name="name" label="任务名称">
                <Input placeholder="请输入任务名称" allowClear style={{ width: 200 }} />
              </Form.Item>
              <Form.Item name="status" label="状态">
                <Select placeholder="请选择" allowClear style={{ width: 150 }} onSearch={() => {}}>
                  <Option value="pending">等待中</Option>
                  <Option value="running">运行中</Option>
                  <Option value="completed">已完成</Option>
                  <Option value="failed">失败</Option>
                  <Option value="cancelled">已取消</Option>
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
              <Button
                type="primary"
                icon={<PlusOutlined />}
                onClick={() => {
                  handleAdd();
                  setModalVisible(true);
                }}
              >
                新增任务
              </Button>
            </Space>
          </div>
          {selectedRowKeys.length > 0 && (
            <div style={{ marginTop: 12, color: "var(--theme-info, #1890ff)" }}>
              已选择 <strong>{selectedRowKeys.length}</strong> 个任务
            </div>
          )}
        </Card>

        <Card>
          <Table
            rowSelection={{ selectedRowKeys, onChange: setSelectedRowKeys }}
            columns={columns}
            dataSource={tasks}
            loading={loading || executeLoading}
            rowKey="id"
            pagination={paginationProps}
            onChange={(pagination, _filters, sorter) => {
              handleTaskSortChange(pagination, _filters, sorter);
              const newPage = pagination.current ?? 1;
              const newPageSize = pagination.pageSize ?? 10;
              setCurrent(newPage);
              setPageSize(newPageSize);
              const formValues = searchForm.getFieldsValue() as Record<string, unknown>;
              const searchParams: Record<string, unknown> = {
                current: newPage,
                pageSize: newPageSize,
                ...(orderByColumn ? { orderByColumn, isAsc } : {}),
              };
              Object.keys(formValues).forEach((key) => {
                const value = formValues[key];
                if (value !== undefined && value !== null && value !== "") {
                  searchParams[key] = value;
                }
              });
              loadTasks(searchParams);
            }}
          />
        </Card>

        <TaskEditModal
          open={modalVisible}
          form={taskForm}
          editingTask={editingTask}
          onOk={handleModalOk}
          onCancel={handleModalCancel}
        />
      </Content>
    </Layout>
  );
};

export default TaskManagement;
