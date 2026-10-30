/**
 * 楼层表格列配置
 */

import type { ColumnsType } from "antd/es/table";
import { Tag, Button } from "antd";
import { EditOutlined, DeleteOutlined, BgColorsOutlined } from "@ant-design/icons";
import type { Floor } from "@/types";
import { createStatusColumn, createDateTimeColumn } from "@/utils/tableHelpers";
import ActionButtons from "@/components/shared/ActionButtons";
import type { ActionPermissionDecision } from "@/components/shared";
import { confirmDelete } from "@/components/shared/confirmDelete";
import type { SortOrder } from "@/hooks/useServerSort";

interface FloorTableColumnCallbacks {
  onEdit: (floor: Floor) => void;
  onEditFloorPlan: (floor: Floor) => void;
  onDelete: (id: string) => void;
  getColumnSortOrder: (field: string) => SortOrder | undefined;
  /**
   * Phase 86 W2：行操作权限决策**由页面层传入**（列工厂内不得调 hook —— 它不是组件）。
   * 不传 = 不做权限判定（保持既有调用方行为零变更）。
   */
  editPermission?: string;
  deletePermission?: string;
  /**
   * 「查看平面图」行操作的决策对象 —— **与平面图保存按钮共用同一个四码合并决策**
   * （`opsFloorPlan.edit ∧ opsWall.* ∧ opsDoor.* ∧ opsWorkstation.edit`）。
   * 该入口打开的就是**可保存**的编辑器，必须受同一决策约束，否则绕过门控直通编辑器。
   * 不传 = 不做判定（既有调用方行为零变更）。
   */
  floorPlanDecision?: ActionPermissionDecision;
}

export function createFloorTableColumns(callbacks: FloorTableColumnCallbacks): ColumnsType<Floor> {
  const {
    onEdit,
    onEditFloorPlan,
    onDelete,
    getColumnSortOrder,
    editPermission,
    deletePermission,
    floorPlanDecision,
  } = callbacks;

  return [
    {
      title: "楼层名称",
      dataIndex: "name",
      key: "name",
      width: 150,
      sorter: true,
      sortOrder: getColumnSortOrder("name"),
    },
    {
      title: "楼层号",
      dataIndex: "floorNo",
      key: "floorNo",
      width: 100,
      sorter: true,
      sortOrder: getColumnSortOrder("floorNo"),
    },
    {
      title: "所属楼宇",
      dataIndex: "buildingName",
      key: "buildingName",
      width: 150,
      sorter: true,
      sortOrder: getColumnSortOrder("buildingName"),
      render: (_, record) => record.buildingName || record.buildingCode,
    },
    {
      title: "面积(m²)",
      dataIndex: "area",
      key: "area",
      width: 120,
      sorter: true,
      sortOrder: getColumnSortOrder("area"),
      render: (value) => value || "-",
    },
    {
      // 工位统计：后端 List 按页内 floor_id 批量聚合注入（无工位楼层为 undefined）。
      // 术语对齐 sys_workstation 独立枚举：0=空闲(未使用)/1=占用(已使用)/2=维护，
      // 禁止按全局 0=正常/1=停用 惯例错标。
      title: "工位统计",
      dataIndex: "workstationStats",
      key: "workstationStats",
      width: 260,
      render: (_, record) => {
        const stats = record.workstationStats;
        if (!stats || stats.total === 0) return "-";
        const items = [
          { key: "total", label: `总 ${stats.total}`, color: "default" as const },
          { key: "used", label: `已用 ${stats.used}`, color: "processing" as const },
          { key: "unused", label: `空闲 ${stats.unused}`, color: "success" as const },
          ...(stats.maintenance > 0
            ? [{ key: "maintenance", label: `维护 ${stats.maintenance}`, color: "warning" as const }]
            : []),
        ];
        return (
          <div style={{ display: "flex", flexWrap: "wrap", gap: 4 }}>
            {items.map((item) => (
              <Tag key={item.key} color={item.color} style={{ marginInlineEnd: 0 }}>
                {item.label}
              </Tag>
            ))}
          </div>
        );
      },
    },
    createStatusColumn("status", {
      width: 100,
      sorter: true,
      sortOrder: getColumnSortOrder("status"),
    }),
    { title: "描述", dataIndex: "description", key: "description", ellipsis: true },
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
            onClick: () => onEdit(record),
            permission: editPermission,
          },
          // 「查看平面图」→ 与平面图保存按钮共用同一四码合并决策对象。
          // 决策为 visible=false（ready 缺任一码）→ 不渲染；未就绪 → 渲染但 disabled + hint。
          // 这里用自渲染形态（render）而非 permission 字段：四码合并结果不是单码，
          // 由页面层把合并后的 ActionPermissionDecision 传进来（列工厂内不得调 hook）。
          ...(floorPlanDecision?.visible === false
            ? []
            : [
                {
                  key: "floorPlan",
                  label: "查看平面图",
                  icon: <BgColorsOutlined />,
                  render: () => (
                    <Button
                      type="link"
                      size="small"
                      icon={<BgColorsOutlined />}
                      disabled={floorPlanDecision?.disabled}
                      title={floorPlanDecision?.hint ?? "查看平面图"}
                      aria-label={floorPlanDecision?.hint ?? "查看平面图"}
                      onClick={() => onEditFloorPlan(record)}
                    >
                      查看平面图
                    </Button>
                  ),
                },
              ]),
          {
            key: "delete",
            label: "删除",
            icon: <DeleteOutlined />,
            danger: true,
            permission: deletePermission,
            onClick: () =>
              confirmDelete({
                target: `楼层 ${record.name || record.floorNo}`,
                onDelete: async () => {
                  await onDelete(record.id);
                },
              }),
          },
        ];
        return <ActionButtons actions={actions} />;
      },
    },
  ];
}
