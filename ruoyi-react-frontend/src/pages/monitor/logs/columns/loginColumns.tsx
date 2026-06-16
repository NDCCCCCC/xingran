/**
 * 登录日志表格列定义
 */

import type { ColumnsType } from "antd/es/table";
import { UserOutlined, EyeOutlined } from "@ant-design/icons";
import { Space } from "antd";
import ActionButtons from "@/components/shared/ActionButtons";
import type { LoginLog } from "../types";
import { renderLogStatusTag, formatLocalTime } from "../utils";

interface GetLoginLogColumnsParams {
  handleViewDetail: (record: LoginLog) => void;
}

export function getLoginLogColumns(params: GetLoginLogColumnsParams): ColumnsType<LoginLog> {
  const { handleViewDetail } = params;

  return [
    {
      title: "访问编号",
      dataIndex: "id",
      key: "id",
      width: 100,
    },
    {
      title: "用户名称",
      dataIndex: "userName",
      key: "userName",
      width: 120,
      render: (text: string) => (
        <Space>
          <UserOutlined />
          {text}
        </Space>
      ),
    },
    {
      title: "登录IP",
      dataIndex: "ipAddr",
      key: "ipAddr",
      width: 140,
    },
    {
      title: "登录地点",
      dataIndex: "loginLocation",
      key: "loginLocation",
      width: 150,
    },
    {
      title: "浏览器",
      dataIndex: "browser",
      key: "browser",
      width: 150,
      render: (text: string) => text || "-",
    },
    {
      title: "操作系统",
      dataIndex: "os",
      key: "os",
      width: 150,
      render: (text: string) => text || "-",
    },
    {
      title: "登录状态",
      dataIndex: "status",
      key: "status",
      width: 100,
      render: (status: number) => renderLogStatusTag(status, "login"),
    },
    {
      title: "操作信息",
      dataIndex: "message",
      key: "message",
      width: 150,
      ellipsis: true,
    },
    {
      title: "登录日期",
      dataIndex: "loginTime",
      key: "loginTime",
      width: 180,
      render: formatLocalTime,
    },
    {
      title: "操作",
      key: "action",
      width: 100,
      fixed: "right",
      render: (_: unknown, record: LoginLog) => {
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
