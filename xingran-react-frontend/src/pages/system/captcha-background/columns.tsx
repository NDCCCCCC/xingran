/**
 * Captcha Background Columns
 * 验证码背景表格列定义
 */

import type { ColumnsType } from "antd/es/table";
import { Button, Space, Modal, Image } from "antd";
import { EditOutlined, DeleteOutlined } from "@ant-design/icons";
import type { CaptchaBackground } from "@/types/captcha";
import { renderPieceShapeTag, renderDifficultyTag, renderStatusTag } from "./constants";

export interface CaptchaColumnsParams {
  handleEdit: (record: CaptchaBackground) => void;
  handleToggle: (id: string) => void;
  handleDelete: (id: string) => void;
}

export function getCaptchaColumns(params: CaptchaColumnsParams): ColumnsType<CaptchaBackground> {
  const { handleEdit, handleToggle, handleDelete } = params;

  return [
    {
      title: "预览",
      key: "preview",
      width: 70,
      render: (_: unknown, record: CaptchaBackground) => (
        <Image
          width={45}
          height={27}
          src={record.filePath}
          preview={{
            src: record.filePath,
          }}
          style={{ objectFit: "cover", cursor: "pointer" }}
        />
      ),
    },
    {
      title: "文件名",
      dataIndex: "fileName",
      key: "fileName",
      width: 80,
      ellipsis: true,
    },
    {
      title: "拼图形状",
      dataIndex: "pieceShape",
      key: "pieceShape",
      width: 80,
      render: (shape) => renderPieceShapeTag(shape),
    },
    {
      title: "难度",
      dataIndex: "difficultyLevel",
      key: "difficultyLevel",
      width: 70,
      render: (level) => renderDifficultyTag(level),
    },
    {
      title: "尺寸",
      key: "size",
      width: 90,
      render: (_: unknown, record: CaptchaBackground) => `${record.fileWidth}x${record.fileHeight}`,
    },
    {
      title: "文件大小",
      dataIndex: "fileSize",
      key: "fileSize",
      width: 90,
      render: (size: number) => `${(size / 1024).toFixed(1)} KB`,
    },
    {
      title: "使用次数",
      dataIndex: "useCount",
      key: "useCount",
      width: 90,
      sorter: (a: CaptchaBackground, b: CaptchaBackground) => a.useCount - b.useCount,
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 70,
      render: (status: number) => renderStatusTag(status),
    },
    {
      title: "备注",
      dataIndex: "remark",
      key: "remark",
      width: 80,
      ellipsis: true,
    },
    {
      title: "操作",
      key: "action",
      fixed: "right",
      width: 180,
      render: (_: unknown, record: CaptchaBackground) => (
        <Space size="small">
          <Button
            type="link"
            size="small"
            icon={<EditOutlined />}
            onClick={() => handleEdit(record)}
          >
            编辑
          </Button>
          <Button type="link" size="small" onClick={() => handleToggle(record.id)}>
            {record.status === 1 ? "禁用" : "启用"}
          </Button>
          <Button
            type="link"
            size="small"
            icon={<DeleteOutlined />}
            onClick={() => {
              Modal.confirm({
                title: "确认删除?",
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
