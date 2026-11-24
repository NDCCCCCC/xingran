/**
 * RPA 任务管理页面
 */

import { useState, useCallback, useEffect, useMemo } from "react";
import type { FC } from "react";
import { App, Table, Button, Space, Form, Card, Modal, Layout } from "antd";
import { PlusOutlined, SearchOutlined, ReloadOutlined } from "@ant-design/icons";
import type { Task } from "@/types/rpa";
import type { PageResponse } from "@/types/base";
import { useTableManager } from "@/hooks/useTableManager";
import { useServerSort } from "@/hooks/useServerSort";
import { createSorterMeta } from "@/utils/tableHelpers";
import { usePagination } from "@/hooks/usePagination";
import { handleSuccess as showSuccessMessage } from "@/utils/errorHandler";
import { getTaskColumns } from "./columns";
import { TaskEditModal } from "./modals";
import { post } from "@/lib/api";
// Phase 71 (W3)：裸 Input/Select → 契约组件。
// 任务状态为固定枚举（业务禁搜）→ children 改写 options + showSearch={false}。
// CONTRACT-02：任务页筛选必须筛「任务态」——与后端 TaskListParams.Status
// (*rpamodels.TaskStatus，0=启用 / 1=停用) 同构。历史这里是执行态字符串
// （pending/running/completed/failed/cancelled），后端 Status 是 *int → 请求体
// 直接 unmarshal 失败 400（D2 实证 "cannot unmarshal string into ... status of type int"）。
import { AppInput, AppSelect } from "@/components/ui";
import { useActionPermission } from "@/components/shared/useActionPermission";
import FilterBar from "@/components/shared/FilterBar";
import { PERMISSION_CODES } from "@/constants/permissionCodes";

const { Content } = Layout;

// 搜索区任务状态：任务态 0/1（启用/停用），与后端 TaskStatus 枚举同构。
// 数值必须保持 number —— 传字符串会被后端 binding 拒绝（CONTRACT-02）。
const TASK_SEARCH_STATUS_OPTIONS = [
  { label: "启用", value: 0 },
  { label: "停用", value: 1 },
];

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
    total: _total,
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
    resetSelection: _resetSelection,
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
      // eslint-disable-next-line no-restricted-syntax -- 非删除语义：执行 RPA 任务为发起动作，UI-SPEC §1.1 判定规则排除
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
          } catch (_error) {
            // 错误已在 errorHandler 中处理
          } finally {
            setExecuteLoading(false);
          }
        },
      });
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
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

  // rpa RBAC 门控（migration_245 同源）：新增/执行/编辑/删除按端点级权限码过滤
  const permGate = useActionPermission();
  const canAdd = permGate.decide(PERMISSION_CODES.rpaTask.add).visible;
  const canExecute = permGate.decide(PERMISSION_CODES.rpaTask.execute).visible;
  const canEdit = permGate.decide(PERMISSION_CODES.rpaTask.edit).visible;
  const canDelete = permGate.decide(PERMISSION_CODES.rpaTask.delete).visible;

  const columns = useMemo(
    () =>
      getTaskColumns({
        handleEdit: (record: Task) => {
          handleEdit(record);
        },
        handleDelete,
        handleExecute,
        canExecute,
        canEdit,
        canDelete,
        getSortOrder: (field) =>
          orderByColumn === field ? ((taskSortOrder ?? null) as "ascend" | "descend" | null) : null,
      }),
    [
      handleEdit,
      handleDelete,
      handleExecute,
      canExecute,
      canEdit,
      canDelete,
      orderByColumn,
      taskSortOrder,
    ]
  );

  return (
    <Layout style={{ background: "transparent" }}>
      <Content style={{ padding: 0, background: "transparent" }}>
        <Card style={{ marginBottom: 16 }}>
          <FilterBar
            actions={
              <Space>
                {canAdd && (
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
                )}
              </Space>
            }
          >
            <Form form={searchForm} layout="inline" style={{ flex: 1, minWidth: 0 }}>
              <Form.Item name="name" label="任务名称">
                <AppInput placeholder="请输入任务名称" style={{ width: 200 }} />
              </Form.Item>
              <Form.Item name="status" label="状态">
                {/* 形态分类器②④：任务状态枚举（业务禁搜），children → options */}
                <AppSelect
                  placeholder="请选择"
                  allowClear
                  style={{ width: 150 }}
                  showSearch={false}
                  options={TASK_SEARCH_STATUS_OPTIONS}
                />
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
          </FilterBar>

          {selectedRowKeys.length > 0 && (
            <div style={{ marginTop: 12, color: "var(--theme-info)" }}>
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
