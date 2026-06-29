import { Table, Card, Form, Input, Select, Button, Space, Alert, Modal, Tag } from "antd";
import type { FormInstance } from "antd";
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  SearchOutlined,
  ReloadOutlined,
  EyeOutlined,
  BarChartOutlined,
} from "@ant-design/icons";
import type { ColumnsType } from "antd/es/table";
import type { TableProps } from "antd";
import ActionButtons from "@/components/shared/ActionButtons";
import type { Notice } from "@/types/notice";
import {
  PRIORITY_COLORS,
  PRIORITY_LABELS,
  NOTICE_TYPE_COLORS,
  NOTICE_TYPE_LABELS,
  PUBLISH_STATUS_COLORS,
  PUBLISH_STATUS_LABELS,
  TARGET_TYPE_LABELS,
} from "@/types/notice";
import type { NoticePriority, TargetType } from "@/types/notice";
import { formatDateTime } from "@/utils/datetime";
import type { SortOrder } from "@/hooks/useServerSort";

const { Option } = Select;

export interface NoticeListProps {
  notices: Notice[];
  loading: boolean;
  total: number;
  current: number;
  pageSize: number;
  selectedRowKeys: React.Key[];
  searchForm: FormInstance;
  onSearch: (params?: Record<string, unknown>) => void;
  onAdd: () => void;
  onEdit: (record: Notice) => void;
  onDelete: (id: string) => void;
  onBatchDelete: () => void;
  onPublish: (id: string) => void;
  onWithdraw: (id: string) => void;
  onView: (id: string) => void;
  onStatistics: (record: Notice) => void;
  onSelectedRowKeysChange: (keys: React.Key[]) => void;
  onPageChange: (page: number, size: number) => void;
  /** 列级 sortOrder：返回当前排序列的方向，其余 undefined（受控高亮） */
  getColumnSortOrder?: (field: string) => SortOrder | undefined;
  /** antd Table onChange 统一处理分页+排序（启用服务端排序时替代 pagination.onChange） */
  onTableChange?: TableProps<Notice>["onChange"];
}

/**
 * 通知列表组件
 * 包含搜索表单、数据表格、批量操作等功能
 */
export const NoticeList: React.FC<NoticeListProps> = ({
  notices,
  loading,
  total,
  current,
  pageSize,
  selectedRowKeys,
  searchForm,
  onSearch,
  onAdd,
  onEdit,
  onDelete,
  onBatchDelete,
  onPublish,
  onWithdraw,
  onView,
  onStatistics,
  onSelectedRowKeysChange,
  onPageChange,
  getColumnSortOrder,
  onTableChange,
}) => {
  const columns: ColumnsType<Notice> = [
    {
      title: "公告标题",
      dataIndex: "noticeTitle",
      key: "noticeTitle",
      ellipsis: true,
      sorter: true,
      sortOrder: getColumnSortOrder?.("noticeTitle"),
      render: (text: string) => <span className="font-medium">{text}</span>,
    },
    {
      title: "类型",
      dataIndex: "noticeType",
      key: "noticeType",
      width: 80,
      sorter: true,
      sortOrder: getColumnSortOrder?.("noticeType"),
      render: (type: string) => (
        <Tag color={NOTICE_TYPE_COLORS[type as "1" | "2"]}>{NOTICE_TYPE_LABELS[type as "1" | "2"]}</Tag>
      ),
    },
    {
      title: "优先级",
      dataIndex: "priority",
      key: "priority",
      width: 90,
      sorter: true,
      sortOrder: getColumnSortOrder?.("priority"),
      render: (priority: number) => {
        if (priority === 0) return <span className="text-gray-400">普通</span>;
        return (
          <Tag color={PRIORITY_COLORS[priority as NoticePriority]}>{PRIORITY_LABELS[priority as NoticePriority]}</Tag>
        );
      },
    },
    {
      title: "接收范围",
      dataIndex: "targetType",
      key: "targetType",
      width: 100,
      render: (type: number) => <span className="text-gray-500">{TARGET_TYPE_LABELS[type as TargetType]}</span>,
    },
    {
      title: "接收范围详情",
      dataIndex: "targets",
      key: "targets",
      width: 150,
      ellipsis: true,
      render: (targets?: { targetType: string; targetName?: string }[]) => {
        if (!targets || targets.length === 0) {
          return <span className="text-gray-400">-</span>;
        }
        // 显示前3个目标，超出显示"等N个"
        const displayTargets = targets.slice(0, 3);
        const remainingCount = targets.length - 3;
        return (
          <span className="text-gray-600" title={targets.map(t => t.targetName || t.targetType).join(", ")}>
            {displayTargets.map(t => t.targetName || t.targetType).join(", ")}
            {remainingCount > 0 && <span className="text-gray-400"> 等 {targets.length} 个</span>}
          </span>
        );
      },
    },
    {
      title: "创建者",
      dataIndex: "createdByName",
      key: "createdByName",
      width: 100,
      ellipsis: true,
      render: (name?: string) => <span className="text-gray-600">{name || "-"}</span>,
    },
    {
      title: "发布状态",
      dataIndex: "publishStatus",
      key: "publishStatus",
      width: 100,
      render: (status: number) => (
        <Tag color={PUBLISH_STATUS_COLORS[status as 0 | 1 | 2 | 3]}>{PUBLISH_STATUS_LABELS[status as 0 | 1 | 2 | 3]}</Tag>
      ),
    },
    {
      title: "发布时间",
      dataIndex: "publishTime",
      key: "publishTime",
      width: 180,
      sorter: true,
      sortOrder: getColumnSortOrder?.("publishTime"),
      render: (time: string) => time ? formatDateTime(time) : <span className="text-gray-400">-</span>,
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 80,
      render: (status: number) => <Tag color={status === 0 ? "success" : "default"}>{status === 0 ? "正常" : "关闭"}</Tag>,
    },
    {
      title: "创建时间",
      dataIndex: "createdAt",
      key: "createdAt",
      width: 180,
      sorter: true,
      sortOrder: getColumnSortOrder?.("createdAt"),
      render: (date: string) => formatDateTime(date),
    },
    {
      title: "操作",
      key: "action",
      fixed: "right",
      width: 100,
      render: (_, record: Notice) => {
        const actions = [
          {
            key: "view",
            label: "查看",
            icon: <EyeOutlined />,
            onClick: () => onView(record.id),
          },
          {
            key: "statistics",
            label: "统计",
            icon: <BarChartOutlined />,
            onClick: () => onStatistics(record),
          },
          ...(record.publishStatus === 0 || record.publishStatus === 2 ? [{
            key: "publish",
            label: "发布",
            onClick: () => onPublish(record.id),
          }] : []),
          ...(record.publishStatus === 1 ? [{
            key: "withdraw",
            label: "撤回",
            danger: true,
            onClick: () => {
              Modal.confirm({
                title: "确认撤回该通知? 撤回后通知将退回到草稿状态",
                okText: "确定",
                cancelText: "取消",
                okButtonProps: { danger: true },
                onOk: () => onWithdraw(record.id),
              });
            },
          }] : []),
          {
            key: "edit",
            label: "编辑",
            icon: <EditOutlined />,
            onClick: () => onEdit(record),
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
                onOk: () => onDelete(record.id),
              });
            },
          },
        ];

        return <ActionButtons actions={actions} />;
      },
    },
  ];

  return (
    <>
      {/* 搜索表单和操作按钮 */}
      <Card style={{ marginBottom: 16 }}>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start", flexWrap: "wrap", gap: "16px" }}>
          <Form form={searchForm} layout="inline" style={{ flex: 1, minWidth: 0 }}>
            <Form.Item name="noticeTitle" label="公告标题">
              <Input placeholder="请输入公告标题" allowClear className="user-form-input" style={{ width: 150 }} />
            </Form.Item>
            <Form.Item name="noticeType" label="公告类型">
              <Select placeholder="请选择" allowClear className="user-form-input" style={{ width: 120 }} onSearch={() => {}}>
                <Option value="1">公告</Option>
                <Option value="2">警告</Option>
              </Select>
            </Form.Item>
            <Form.Item>
              <Space>
                <Button type="primary" icon={<SearchOutlined />} onClick={() => onSearch()}>
                  搜索
                </Button>
                <Button icon={<ReloadOutlined />} onClick={() => { searchForm.resetFields(); onSearch(); }}>
                  刷新
                </Button>
              </Space>
            </Form.Item>
          </Form>
          <Space>
            {selectedRowKeys.length > 0 && (
              <Button
                danger
                icon={<DeleteOutlined />}
                onClick={onBatchDelete}
              >
                批量删除 ({selectedRowKeys.length})
              </Button>
            )}
            <Button type="primary" icon={<PlusOutlined />} onClick={onAdd}>
              新增公告
            </Button>
          </Space>
        </div>
        {selectedRowKeys.length > 0 && (
          <Alert
            title={
              <span>
                已选择 <strong>{selectedRowKeys.length}</strong> 个公告，
                <Button
                  type="link"
                  size="small"
                  onClick={() => onSelectedRowKeysChange([])}
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
            onChange: onSelectedRowKeysChange,
          }}
          columns={columns}
          dataSource={notices}
          loading={loading}
          rowKey="id"
          scroll={{ x: 1600 }}
          pagination={{
            current,
            pageSize,
            total,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => `共 ${total} 条`,
            // 启用服务端排序时由 Table onChange 统一处理分页+排序，
            // 此处不再重复挂分页 onChange（避免分页触发两次）。
            ...(onTableChange ? {} : { onChange: onPageChange }),
          }}
          // 服务端排序：父组件传入 onTableChange 时接管分页+排序；
          // 否则不挂 onChange，分页走 pagination.onChange。
          onChange={onTableChange}
        />
      </Card>
    </>
  );
};
