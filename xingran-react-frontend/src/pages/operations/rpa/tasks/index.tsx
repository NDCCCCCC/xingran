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
// Phase 156 / NOCODE-04：新增任务走三步向导
import { TaskWizard } from "../wizard/TaskWizard";
import type { WizardDraft } from "../wizard/payload";
import { toTaskPayload } from "../wizard/payload";
import type { Action } from "@/types/rpa";
import { post } from "@/lib/api";
// Phase W6 / REC-01：录制面板（录制/拾取会话的前端入口）
import { RecordingPanel } from "../recording/RecordingPanel";
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

  // Phase 156 / NOCODE-04：「新增任务」改为打开三步向导（不再直接弹编辑器）。
  // 归属协议补登项：tasks/index.tsx:212-223 的「新增任务」按钮改开向导。
  // 编辑器仍保留（编辑既有任务时使用），只是「新增」走更友好的向导路径。
  const [wizardOpen, setWizardOpen] = useState(false);

  // Phase W6 / REC-01：录制面板开关（向导第 1 步「录一段操作」的真按钮）。
  const [recordingOpen, setRecordingOpen] = useState(false);
  // 录制产物回填给向导（第 2 步步骤列表的直接来源）。
  const [recordedActions, setRecordedActions] = useState<Action[]>([]);


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
                  <Space>
                    {/* NOCODE-04：主入口走三步向导（一般员工路径） */}
                    <Button
                      type="primary"
                      icon={<PlusOutlined />}
                      data-testid="task-add-wizard"
                      onClick={() => {
                        setWizardOpen(true);
                      }}
                    >
                      新增任务
                    </Button>
                    {/* 高级入口：直接开编辑器（熟悉动作配置的人用，保留既有能力） */}
                    <Button
                      data-testid="task-add-advanced"
                      onClick={() => {
                        handleAdd();
                        setModalVisible(true);
                      }}
                    >
                      高级新建
                    </Button>
                  </Space>
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

        {/*
          NOCODE-04：三步向导承载。
          用 Modal（宽 900）而不是跳页：向导是「新建任务」的上下文内动作，
          弹窗能保留用户原有的筛选/分页状态，关闭即回到列表，符合
          Power Automate Desktop 的交互习惯。
        */}
        <Modal
          open={wizardOpen}
          onCancel={() => setWizardOpen(false)}
          footer={null}
          width={900}
          destroyOnHidden
          title="新建自动化任务"
          data-testid="task-wizard-modal"
        >
          <TaskWizard
            key={recordedActions.length > 0 ? "with-recording" : "fresh"}
            initialActions={recordedActions.length > 0 ? recordedActions : undefined}
            onSave={async (draft: WizardDraft) => {
              await post("/rpa/tasks", toTaskPayload(draft));
              showSuccessMessage("创建");
              setWizardOpen(false);
              refreshData();
            }}
            onCancel={() => setWizardOpen(false)}
            onOpenAIGenerator={() => {
              // 向导内部已自带 AI 生成器入口；此处仅作为无动作时的兜底提示
              message.info("请在上方「用中文说」里点下一步，系统会打开智能生成");
            }}
            // Phase W6 / REC-01：「录制」从指引文案改为真按钮 → 打开录制面板。
            // 录制完成（Worker 产出 actions 经后端落库）→ 直接填进第 2 步步骤列表。
            onOpenRecorder={() => setRecordingOpen(true)}
          />
        </Modal>

        {/*
          Phase W6 / REC-01：录制面板承载。
          独立 Modal（宽 1080）：录制画面需要更大的可视区域；关闭面板即中止会话
          （面板卸载时 abort，后端闲置超时兜底回收执行机资源）。
        */}
        <Modal
          open={recordingOpen}
          onCancel={() => setRecordingOpen(false)}
          footer={null}
          width={1080}
          destroyOnHidden
          title="录制一段操作"
          data-testid="recording-panel-modal"
        >
          <RecordingPanel
            taskId={undefined}
            targetUrl={undefined}
            onFinish={(actions) => {
              // REC-01 验收路径：停止 → actions 直接填进第 2 步步骤列表。
              // 实现方式：暂存录制产物 → 关录制面板 → 重开向导（key 变化触发
              // TaskWizard 重挂载，initialActions 通道把动作带回第 1 步预览，
              // 员工点「下一步」即在步骤列表里核对）。
              setRecordedActions(actions);
              setRecordingOpen(false);
              message.success(`录制完成，共整理出 ${actions.length} 个步骤`);
              setWizardOpen(true);
            }}
            onCancel={() => setRecordingOpen(false)}
          />
        </Modal>
      </Content>
    </Layout>
  );
};

export default TaskManagement;
