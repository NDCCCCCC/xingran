/**
 * RPA 定时调度管理页面
 *
 * 面向一般员工：主区域只出现「每天 / 每周 / 每隔若干分钟」这类业务选项，
 * 定时规则（cron 表达式）只在默认折叠的「高级设置」里出现。
 */

import { useCallback, useEffect, useMemo, useState } from "react";
import type { FC } from "react";
import { App, Button, Card, Form, Layout, Space, Table } from "antd";
import { PlusOutlined, ReloadOutlined, SearchOutlined } from "@ant-design/icons";
import { useTableManager } from "@/hooks/useTableManager";
import { usePagination } from "@/hooks/usePagination";
import { handleSuccess as showSuccessMessage } from "@/utils/errorHandler";
// Phase 71 契约组件：禁裸 antd Input/Select/TextArea。
import { AppInput, AppSelect } from "@/components/ui";
import { useActionPermission } from "@/components/shared/useActionPermission";
import FilterBar from "@/components/shared/FilterBar";
import { PERMISSION_CODES } from "@/constants/permissionCodes";
import { getScheduleColumns } from "./columns";
import { ScheduleModal, type ScheduleFormValues } from "./modals/ScheduleModal";
import {
  SCHEDULE_STATUS_DISABLED,
  SCHEDULE_STATUS_ENABLED,
  createSchedule,
  deleteSchedule,
  listSchedules,
  listTaskBriefs,
  setScheduleStatus,
  triggerSchedule,
  updateSchedule,
  type ScheduleRecord,
  type TaskBrief,
} from "./api";

const { Content } = Layout;

/**
 * 状态筛选项：数值必须保持 number —— 后端 ScheduleListParams.Status 是 *int，
 * 传字符串会被 binding 直接拒绝（与 CONTRACT-02 任务页同款口径）。
 */
const SCHEDULE_STATUS_OPTIONS = [
  { label: "启用", value: SCHEDULE_STATUS_ENABLED },
  { label: "停用", value: SCHEDULE_STATUS_DISABLED },
];

/** 关联任务候选项一次拉取上限（够覆盖常规规模，且不触发「大分页下拉」反模式） */
const TASK_OPTIONS_PAGE_SIZE = 200;

const ScheduleManagement: FC = () => {
  const { message } = App.useApp();
  const { paginationProps, setCurrent, setPageSize, setTotal } = usePagination();
  const [form] = Form.useForm<ScheduleFormValues>();

  // 弹窗状态（自己管，不复用 useTableManager 的编辑态：初始化逻辑是定制的）
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<ScheduleRecord | null>(null);

  // 关联任务（列表页用于展示任务名称，弹窗用于选择）
  const [taskOptions, setTaskOptions] = useState<TaskBrief[]>([]);
  /** 正在切换启用状态的调度标识（该行开关置灰，防重复提交） */
  const [statusLoadingId, setStatusLoadingId] = useState<string | null>(null);

  const {
    loading,
    data: schedules,
    searchForm,
    loadData: loadSchedules,
    handleSearch,
    handleReset,
  } = useTableManager<ScheduleRecord>(
    async (params) => {
      const result = await listSchedules({
        current: Number(params.current ?? 1),
        pageSize: Number(params.pageSize ?? 10),
        name: params.name ? String(params.name) : undefined,
        status:
          params.status === undefined || params.status === null || params.status === ""
            ? undefined
            : Number(params.status),
      });
      return { list: result.list, total: result.total };
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

  /** 任务标识 → 任务名称（「关联任务」列展示用，不暴露内部标识） */
  const taskNameMap = useMemo(() => {
    const map: Record<string, string> = {};
    taskOptions.forEach((task) => {
      map[task.id] = task.taskName;
    });
    return map;
  }, [taskOptions]);

  const loadTaskOptions = useCallback(async () => {
    try {
      const { list } = await listTaskBriefs({ current: 1, pageSize: TASK_OPTIONS_PAGE_SIZE });
      setTaskOptions(list);
    } catch (_error) {
      // 错误提示已由 api 层统一出口处理
    }
  }, []);

  const refreshData = useCallback(() => {
    loadSchedules();
  }, [loadSchedules]);

  useEffect(() => {
    loadSchedules();
  }, [loadSchedules]);

  useEffect(() => {
    loadTaskOptions();
  }, [loadTaskOptions]);

  // rpa RBAC 门控（migration_257 播种的 4 个权限码）：
  // 新建 → add；编辑 / 启停 / 立即运行 → edit；删除 → remove。
  const permGate = useActionPermission();
  const canAdd = permGate.decide(PERMISSION_CODES.rpaSchedule.add).visible;
  const canEdit = permGate.decide(PERMISSION_CODES.rpaSchedule.edit).visible;
  const canDelete = permGate.decide(PERMISSION_CODES.rpaSchedule.remove).visible;

  const handleCreate = useCallback(() => {
    setEditing(null);
    setModalOpen(true);
  }, []);

  const handleEditSchedule = useCallback((record: ScheduleRecord) => {
    setEditing(record);
    setModalOpen(true);
  }, []);

  const handleModalCancel = useCallback(() => {
    setModalOpen(false);
    setEditing(null);
    form.resetFields();
  }, [form]);

  const handleModalOk = useCallback(
    async (values: {
      name: string;
      taskId: string;
      cronExpr: string;
      description?: string;
      status: number;
    }) => {
      if (editing) {
        await updateSchedule(editing.id, values);
        showSuccessMessage("更新");
      } else {
        await createSchedule(values);
        showSuccessMessage("创建");
      }
      setModalOpen(false);
      setEditing(null);
      form.resetFields();
      refreshData();
    },
    [editing, form, refreshData]
  );

  const handleTrigger = useCallback(
    async (record: ScheduleRecord) => {
      try {
        await triggerSchedule(record.id);
        message.success("已提交运行一次");
        refreshData();
      } catch (_error) {
        // 错误提示已由 api 层统一出口处理
      }
    },
    [message, refreshData]
  );

  const handleDelete = useCallback(
    async (record: ScheduleRecord) => {
      // 二次确认由 ConfirmDelete 组件承担，这里只执行真正的删除动作
      await deleteSchedule(record.id);
      refreshData();
    },
    [refreshData]
  );

  const handleStatusChange = useCallback(
    async (record: ScheduleRecord, nextStatus: number) => {
      setStatusLoadingId(record.id);
      try {
        await setScheduleStatus(record.id, nextStatus);
        showSuccessMessage(nextStatus === SCHEDULE_STATUS_ENABLED ? "启用" : "停用");
        refreshData();
      } catch (_error) {
        // 错误提示已由 api 层统一出口处理
      } finally {
        setStatusLoadingId(null);
      }
    },
    [refreshData]
  );

  const columns = useMemo(
    () =>
      getScheduleColumns({
        handleEdit: handleEditSchedule,
        handleTrigger,
        handleDelete,
        handleStatusChange,
        canEdit,
        canDelete,
        taskNameMap,
        statusLoadingId,
      }),
    [
      handleEditSchedule,
      handleTrigger,
      handleDelete,
      handleStatusChange,
      canEdit,
      canDelete,
      taskNameMap,
      statusLoadingId,
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
                    data-testid="schedule-add"
                    onClick={handleCreate}
                  >
                    新建调度
                  </Button>
                )}
              </Space>
            }
          >
            <Form form={searchForm} layout="inline" style={{ flex: 1, minWidth: 0 }}>
              <Form.Item name="name" label="名称">
                <AppInput placeholder="请输入调度名称" style={{ width: 200 }} />
              </Form.Item>
              <Form.Item name="status" label="状态">
                {/* 状态枚举（业务禁搜）→ options */}
                <AppSelect
                  placeholder="请选择"
                  allowClear
                  style={{ width: 150 }}
                  showSearch={false}
                  options={SCHEDULE_STATUS_OPTIONS}
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
        </Card>

        <Card>
          <Table
            columns={columns}
            dataSource={schedules}
            loading={loading}
            rowKey="id"
            pagination={paginationProps}
            onChange={(pagination) => {
              const newPage = pagination.current ?? 1;
              const newPageSize = pagination.pageSize ?? 10;
              setCurrent(newPage);
              setPageSize(newPageSize);
              const formValues = searchForm.getFieldsValue() as Record<string, unknown>;
              const searchParams: Record<string, unknown> = {
                current: newPage,
                pageSize: newPageSize,
              };
              Object.keys(formValues).forEach((key) => {
                const value = formValues[key];
                if (value !== undefined && value !== null && value !== "") {
                  searchParams[key] = value;
                }
              });
              loadSchedules(searchParams);
            }}
          />
        </Card>

        <ScheduleModal
          open={modalOpen}
          form={form}
          editing={editing}
          taskOptions={taskOptions}
          onOk={handleModalOk}
          onCancel={handleModalCancel}
        />
      </Content>
    </Layout>
  );
};

export default ScheduleManagement;
