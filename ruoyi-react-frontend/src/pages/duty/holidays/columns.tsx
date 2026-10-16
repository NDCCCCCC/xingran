/**
 * Holiday Table Columns
 * 节假日表格列定义
 */

import type { ColumnsType } from "antd/es/table";
import { Button, Space, Tag, Modal } from "antd";
import { EditOutlined, DeleteOutlined } from "@ant-design/icons";
import type { Holiday } from "@/lib/dutyApi";
import { WEEKDAY_TEXTS } from "./constants";
import dayjs from "dayjs";
import { formatDate } from "@/utils/datetime";
import { createSorter } from "@/utils/tableHelpers";

export interface HolidayColumnsParams {
  handleEdit: (record: Holiday) => void;
  handleDelete: (id: string) => Promise<void>;
}

export function getHolidayColumns(params: HolidayColumnsParams): ColumnsType<Holiday> {
  const { handleEdit, handleDelete } = params;

  return [
    {
      title: "序号",
      key: "index",
      width: 60,
      render: (_: unknown, __: unknown, index: number) => index + 1,
    },
    {
      title: "日期",
      dataIndex: "holidayDate",
      key: "holidayDate",
      width: 120,
      render: (date: string) => formatDate(date),
      sorter: (a: Holiday, b: Holiday) => dayjs(a.holidayDate).unix() - dayjs(b.holidayDate).unix(),
    },
    {
      title: "星期",
      key: "weekday",
      width: 80,
      render: (_: unknown, record: Holiday) => {
        const weekday = dayjs(record.holidayDate).day();
        return `周${WEEKDAY_TEXTS[weekday]}`;
      },
    },
    {
      title: "节假日名称",
      dataIndex: "holidayName",
      key: "holidayName",
      width: 150,
      sorter: createSorter<Holiday>("holidayName", "string"),
    },
    {
      title: "类型",
      dataIndex: "holidayType",
      key: "holidayType",
      width: 100,
      sorter: createSorter<Holiday>("holidayType", "string"),
      render: (type: string) => {
        if (type === "legal") return <Tag color="red">法定节假日</Tag>;
        if (type === "workday") return <Tag color="orange">调休工作日</Tag>;
        if (type === "custom") return <Tag color="blue">自定义</Tag>;
        return <Tag>{type}</Tag>;
      },
    },
    {
      title: "是否休息",
      dataIndex: "isOffday",
      key: "isOffday",
      width: 100,
      sorter: createSorter<Holiday>("isOffday", "boolean"),
      render: (isOffday: boolean) => (
        <Tag color={isOffday ? "green" : "default"}>{isOffday ? "休息日" : "工作日"}</Tag>
      ),
    },
    {
      title: "备注",
      dataIndex: "remark",
      key: "remark",
      width: 200,
      ellipsis: true,
      sorter: createSorter<Holiday>("remark", "string"),
    },
    {
      title: "操作",
      key: "action",
      width: 150,
      fixed: "right",
      render: (_: unknown, record: Holiday) => (
        <Space>
          <Button
            type="link"
            size="small"
            icon={<EditOutlined />}
            onClick={() => handleEdit(record)}
          >
            编辑
          </Button>
          <Button
            type="link"
            size="small"
            icon={<DeleteOutlined />}
            onClick={() => {
              Modal.confirm({
                title: "确定要删除吗？",
                okText: "确定",
                cancelText: "取消",
                okButtonProps: { danger: true },
                onOk: () => handleDelete(record.id),
              });
            }}
          >
            删除
          </Button>
        </Space>
      ),
    },
  ];
}
