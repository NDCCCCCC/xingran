/**
 * 定时调度表格列定义
 *
 * 列名全部用中文业务语义，员工看不到「cron」「任务标识」这类技术词：
 *   名称 / 关联任务 / 运行方式 / 启用·停用 / 最近触发时间 / 下次触发时间 / 操作
 */

import type { ColumnsType } from "antd/es/table";
import { Switch, Tooltip } from "antd";
import { EditOutlined, PlayCircleOutlined } from "@ant-design/icons";
import ActionButtons, { type ActionButton } from "@/components/shared/ActionButtons";
// 从 shared 出口导入（出口内部用显式扩展名，规避 Windows 大小写不敏感解析）
import { ConfirmDelete } from "@/components/shared";
import { formatDateTime } from "@/utils/datetime";
import { describeRunMode } from "./cronPresets";
import { SCHEDULE_STATUS_ENABLED, type ScheduleRecord } from "./api";

export interface ScheduleColumnsParams {
  /** 编辑 */
  handleEdit: (record: ScheduleRecord) => void;
  /** 立即运行一次 */
  handleTrigger: (record: ScheduleRecord) => void;
  /** 删除（调用方内部已做二次确认所需的删除动作） */
  handleDelete: (record: ScheduleRecord) => void;
  /** 启用/停用切换 */
  handleStatusChange: (record: ScheduleRecord, nextStatus: number) => void;
  /** rpa RBAC 门控：编辑 / 启停 / 立即运行 */
  canEdit: boolean;
  /** rpa RBAC 门控：删除 */
  canDelete: boolean;
  /** 任务标识 → 任务名称（列表页「关联任务」列展示用） */
  taskNameMap: Record<string, string>;
  /** 正在切换状态的调度标识（该行开关置灰，防重复提交） */
  statusLoadingId?: string | null;
}

export function getScheduleColumns(params: ScheduleColumnsParams): ColumnsType<ScheduleRecord> {
  const {
    handleEdit,
    handleTrigger,
    handleDelete,
    handleStatusChange,
    canEdit,
    canDelete,
    taskNameMap,
    statusLoadingId,
  } = params;

  return [
    {
      title: "名称",
      dataIndex: "name",
      key: "name",
      width: 180,
      ellipsis: true,
    },
    {
      title: "关联任务",
      key: "taskName",
      width: 180,
      ellipsis: true,
      // 展示任务的名称而不是内部标识；任务已被删除时回落为「—」
      render: (_: unknown, record: ScheduleRecord) => taskNameMap[record.taskId] ?? "—",
    },
    {
      title: "运行方式",
      key: "runMode",
      width: 160,
      // cron 表达式 → 中文业务文案（每天 09:00 / 每周一 09:00 / 每隔 15 分钟 / 自定义）
      render: (_: unknown, record: ScheduleRecord) => describeRunMode(record.cronExpr ?? ""),
    },
    {
      title: "启用/停用",
      key: "status",
      width: 110,
      render: (_: unknown, record: ScheduleRecord) => {
        const enabled = record.status === SCHEDULE_STATUS_ENABLED;
        const disabled = statusLoadingId === record.id;
        return (
          <Tooltip title={enabled ? "点击停用" : "点击启用"}>
            <Switch
              checked={enabled}
              disabled={disabled}
              loading={disabled}
              aria-label={enabled ? "停用该调度" : "启用该调度"}
              onChange={(checked) => {
                handleStatusChange(record, checked ? SCHEDULE_STATUS_ENABLED : 1);
              }}
            />
          </Tooltip>
        );
      },
    },
    {
      title: "最近触发时间",
      key: "lastRunTime",
      width: 180,
      render: (_: unknown, record: ScheduleRecord) =>
        record.lastRunTime ? formatDateTime(record.lastRunTime) : "尚未运行",
    },
    {
      title: "下次触发时间",
      key: "nextRunTime",
      width: 180,
      render: (_: unknown, record: ScheduleRecord) =>
        record.nextRunTime ? formatDateTime(record.nextRunTime) : "—",
    },
    {
      title: "操作",
      key: "action",
      width: 240,
      fixed: "right",
      render: (_: unknown, record: ScheduleRecord) => {
        const actions: ActionButton[] = [];

        if (canEdit) {
          actions.push({
            key: "edit",
            label: "编辑",
            icon: <EditOutlined />,
            onClick: () => handleEdit(record),
          });
          actions.push({
            key: "trigger",
            label: "立即运行一次",
            icon: <PlayCircleOutlined />,
            onClick: () => handleTrigger(record),
          });
        }

        if (canDelete) {
          actions.push({
            key: "delete",
            label: "删除",
            danger: true,
            // 删除必须二次确认 —— 走 ConfirmDelete 唯一声明式入口
            render: () => (
              <ConfirmDelete
                target={record.name || "该调度"}
                onDelete={async () => {
                  await handleDelete(record);
                }}
              >
                删除
              </ConfirmDelete>
            ),
          });
        }

        // threshold=4：三个操作直接平铺显示（默认 3 会折叠进「操作」下拉，员工不易发现）
        return <ActionButtons actions={actions} threshold={4} />;
      },
    },
  ];
}
