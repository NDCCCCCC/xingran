/**
 * 操作日志表格列定义
 */

import type { ColumnsType } from "antd/es/table";
import { EyeOutlined } from "@ant-design/icons";
import ActionButtons from "@/components/shared/ActionButtons";
import type { OperLog } from "../types";
import { getBusinessTypeLabel, renderRequestMethodTag, renderLogStatusTag, formatLocalTime } from "../utils";

interface GetOperLogColumnsParams {
  handleViewDetail: (record: OperLog) => void;
}

export function getOperLogColumns(params: GetOperLogColumnsParams): ColumnsType<OperLog> {
  const { handleViewDetail } = params;

  return [
    {
      title: "日志编号",
      dataIndex: "id",
      key: "id",
      width: 100,
    },
    {
      title: "操作模块",
      dataIndex: "title",
      key: "title",
      width: 120,
    },
    {
      title: "业务类型",
      dataIndex: "businessType",
      key: "businessType",
      width: 100,
      render: (type: number) => getBusinessTypeLabel(type),
    },
    {
      title: "请求方式",
      dataIndex: "requestMethod",
      key: "requestMethod",
      width: 100,
      render: renderRequestMethodTag,
    },
    {
      title: "操作人员",
      dataIndex: "operName",
      key: "operName",
      width: 100,
    },
    {
      title: "部门名称",
      dataIndex: "deptName",
      key: "deptName",
      width: 120,
      render: (text: string) => text || "-",
    },
    {
      title: "操作地址",
      dataIndex: "operUrl",
      key: "operUrl",
      width: 200,
      ellipsis: true,
    },
    {
      title: "操作地点",
      dataIndex: "operLocation",
      key: "operLocation",
      width: 150,
    },
    {
      title: "操作状态",
      dataIndex: "status",
      key: "status",
      width: 100,
      render: (status: number) => renderLogStatusTag(status, "oper"),
    },
    {
      title: "操作时间",
      dataIndex: "operTime",
      key: "operTime",
      width: 180,
      render: formatLocalTime,
    },
    {
      title: "操作",
      key: "action",
      width: 100,
      fixed: "right",
      render: (_: unknown, record: OperLog) => {
        const actions = [
          {
            key: "detail",
            label: "详情",
            icon: <EyeOutlined />,
            onClick: () => handleViewDetail(record),
          },
        ];
        return <ActionButtons actions={actions} />;
      },
    },
  ];
}
