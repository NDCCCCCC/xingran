/**
 * Knowledge Article Table Columns
 * 知识文章表格列定义
 */

import type { ColumnsType } from 'antd/es/table';
import { Button, Space, Tag, Popconfirm } from 'antd';
import { EyeOutlined, LikeOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons';
import type { KnowledgeArticle } from '@/lib/knowledgeApi';
import { KnowledgeArticleStatus } from '@/lib/knowledgeApi';
import { STATUS_CONFIG } from './constants';
import ActionButtons from '@/components/shared/ActionButtons';
import { formatDateTime } from '@/utils/datetime';

export interface ArticleColumnsParams {
  handlePreview: (record: KnowledgeArticle) => void;
  handleEdit: (record: KnowledgeArticle) => void;
  handlePublish: (record: KnowledgeArticle) => Promise<void>;
  handleLike: (id: string) => Promise<void>;
  handleDelete: (id: string) => Promise<void>;
  current: number;
  pageSize: number;
}

export function getArticleColumns(params: ArticleColumnsParams): ColumnsType<KnowledgeArticle> {
  const { handlePreview, handleEdit, handlePublish, handleLike, handleDelete, current, pageSize } = params;

  return [
    {
      title: '序号',
      key: 'index',
      width: 60,
      render: (_: unknown, __: unknown, index: number) => (current - 1) * pageSize + index + 1,
    },
    {
      title: '标题',
      dataIndex: 'title',
      key: 'title',
      width: 200,
      ellipsis: true,
    },
    {
      title: '分类',
      key: 'category',
      width: 120,
      render: (_: unknown, record: KnowledgeArticle) => record.category?.categoryName || '-',
    },
    {
      title: '标签',
      key: 'tags',
      width: 150,
      render: (_: unknown, record: KnowledgeArticle) => {
        const tags = record.tags || [];
        if (tags.length === 0) return '-';
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
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 90,
      render: (status: KnowledgeArticleStatus) => {
        const config = STATUS_CONFIG[status];
        return <Tag color={config?.color}>{config?.text}</Tag>;
      },
    },
    {
      title: '浏览',
      dataIndex: 'viewCount',
      key: 'viewCount',
      width: 80,
      render: (count: number) => <span><EyeOutlined className="mr-1" />{count}</span>,
    },
    {
      title: '点赞',
      dataIndex: 'likeCount',
      key: 'likeCount',
      width: 80,
      render: (count: number) => <span><LikeOutlined className="mr-1" />{count}</span>,
    },
    {
      title: '创建人',
      key: 'createdBy',
      width: 100,
      render: (_: unknown, record: KnowledgeArticle) => record.createdBy,
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 160,
      render: (date: string) => formatDateTime(date, 'YYYY-MM-DD HH:mm'),
    },
    {
      title: '操作',
      key: 'action',
      width: 100,
      fixed: 'right',
      render: (_: unknown, record: KnowledgeArticle) => {
        const actions = [
          {
            key: 'preview',
            label: '预览',
            icon: <EyeOutlined />,
            onClick: () => handlePreview(record),
          },
          {
            key: 'edit',
            label: '编辑',
            icon: <EditOutlined />,
            onClick: () => handleEdit(record),
          },
          ...(record.status === KnowledgeArticleStatus.Draft ? [{
            key: 'publish',
            label: '发布',
            onClick: () => handlePublish(record),
          }] : []),
          {
            key: 'like',
            label: '点赞',
            icon: <LikeOutlined />,
            onClick: () => handleLike(record.id),
          },
          {
            key: 'delete',
            label: '删除',
            icon: <DeleteOutlined />,
            danger: true,
            render: () => (
              <Popconfirm title="确定要删除吗？" onConfirm={() => handleDelete(record.id)} okText="确定" cancelText="取消">
                <Button type="link" danger ghost icon={<DeleteOutlined />} size="small">
                  删除
                </Button>
              </Popconfirm>
            ),
          },
        ];

        return <ActionButtons actions={actions} />;
      },
    },
  ];
}
