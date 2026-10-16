/**
 * Knowledge Article Table Columns
 * 知识文章表格列定义
 */

import type { ColumnsType } from "antd/es/table";
import { Space, Tag, Modal } from "antd";
import { EyeOutlined, LikeOutlined, EditOutlined, DeleteOutlined } from "@ant-design/icons";
import type { KnowledgeArticle } from "@/lib/knowledgeApi";
import { KnowledgeArticleStatus } from "@/lib/knowledgeApi";
import { STATUS_CONFIG } from "./constants";
import ActionButtons from "@/components/shared/ActionButtons";
import { formatDateTime } from "@/utils/datetime";
import type { SorterMeta } from "@/utils/tableHelpers";

export interface ArticleColumnsParams {
  handlePreview: (record: KnowledgeArticle) => void;
  handleEdit: (record: KnowledgeArticle) => void;
  handlePublish: (record: KnowledgeArticle) => Promise<void>;
  handleLike: (id: string) => Promise<void>;
  handleDelete: (id: string) => Promise<void>;
  current: number;
  pageSize: number;
  /** 列级 sortOrder：返回当前排序列的方向，其余 undefined（受控高亮） */
  getColumnSortOrder?: (field: string) => "ascend" | "descend" | null | undefined;
  /** 可排序列白名单（对应后端白名单 key） */
  sorterMetas?: Array<SorterMeta<KnowledgeArticle> | undefined>;
}

export function getArticleColumns(params: ArticleColumnsParams): ColumnsType<KnowledgeArticle> {
  const {
    handlePreview,
    handleEdit,
    handlePublish,
    handleLike,
    handleDelete,
    current,
    pageSize,
    getColumnSortOrder,
    sorterMetas: _sorterMetas,
  } = params;

  return [
    {
      title: "序号",
      key: "index",
      width: 60,
      render: (_: unknown, __: unknown, index: number) => (current - 1) * pageSize + index + 1,
    },
    {
      title: "标题",
      dataIndex: "title",
      key: "title",
      width: 200,
      ellipsis: true,
      sorter: true,
      sortOrder: getColumnSortOrder?.("title"),
    },
    {
      title: "分类",
      key: "category",
      width: 120,
      render: (_: unknown, record: KnowledgeArticle) => record.category?.categoryName || "-",
    },
    {
      title: "标签",
      key: "tags",
      width: 150,
      render: (_: unknown, record: KnowledgeArticle) => {
        const tags = record.tags || [];
        if (tags.length === 0) return "-";
        return (
          <Space size={[0, 4]} wrap>
            {tags.map((tag) => (
              <Tag key={tag.id}>{tag.tagName}</Tag>
            ))}
          </Space>
        );
      },
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 90,
      sorter: true,
      sortOrder: getColumnSortOrder?.("status"),
      render: (status: KnowledgeArticleStatus) => {
        const config = STATUS_CONFIG[status];
        return <Tag color={config?.color}>{config?.text}</Tag>;
      },
    },
    {
      title: "浏览",
      dataIndex: "viewCount",
      key: "viewCount",
      width: 80,
      sorter: true,
      sortOrder: getColumnSortOrder?.("viewCount"),
      render: (count: number) => (
        <span>
          <EyeOutlined className="mr-1" />
          {count}
        </span>
      ),
    },
    {
      title: "点赞",
      dataIndex: "likeCount",
      key: "likeCount",
      width: 80,
      sorter: true,
      sortOrder: getColumnSortOrder?.("likeCount"),
      render: (count: number) => (
        <span>
          <LikeOutlined className="mr-1" />
          {count}
        </span>
      ),
    },
    {
      title: "创建人",
      key: "createdBy",
      width: 100,
      render: (_: unknown, record: KnowledgeArticle) => record.createdBy,
    },
    {
      title: "创建时间",
      dataIndex: "createdAt",
      key: "createdAt",
      width: 160,
      sorter: true,
      sortOrder: getColumnSortOrder?.("createdAt"),
      render: (date: string) => formatDateTime(date, "YYYY-MM-DD HH:mm"),
    },
    {
      title: "操作",
      key: "action",
      width: 100,
      fixed: "right",
      render: (_: unknown, record: KnowledgeArticle) => {
        const actions = [
          {
            key: "preview",
            label: "预览",
            icon: <EyeOutlined />,
            onClick: () => handlePreview(record),
          },
          {
            key: "edit",
            label: "编辑",
            icon: <EditOutlined />,
            onClick: () => handleEdit(record),
          },
          ...(record.status === KnowledgeArticleStatus.Draft
            ? [
                {
                  key: "publish",
                  label: "发布",
                  onClick: () => handlePublish(record),
                },
              ]
            : []),
          {
            key: "like",
            label: "点赞",
            icon: <LikeOutlined />,
            onClick: () => handleLike(record.id),
          },
          {
            key: "delete",
            label: "删除",
            icon: <DeleteOutlined />,
            danger: true,
            onClick: () => {
              Modal.confirm({
                title: "确定要删除吗？",
                okText: "确定",
                cancelText: "取消",
                okButtonProps: { danger: true },
                onOk: () => handleDelete(record.id),
              });
            },
          },
        ];

        return <ActionButtons actions={actions} />;
      },
    },
  ];
}
