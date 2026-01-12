import { useState, useEffect, useMemo } from 'react';
import {
  Button,
  Form,
  Input,
  Select,
  Table,
  Modal,
  Space,
  Tag,
  message,
  Popconfirm,
  Card,
  Row,
  Col,
  Statistic,
} from 'antd';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  SearchOutlined,
  ReloadOutlined,
  FileTextOutlined,
  EyeOutlined,
  LikeOutlined,
} from '@ant-design/icons';
import type { ColumnsType, TablePaginationConfig } from 'antd/es/table';
import dayjs from 'dayjs';
import {
  getKnowledgeArticleList,
  createKnowledgeArticle,
  updateKnowledgeArticle,
  deleteKnowledgeArticle,
  likeKnowledgeArticle,
  type KnowledgeArticle,
  type KnowledgeArticleCreateRequest,
  type KnowledgeArticleUpdateRequest,
  KnowledgeArticleStatus,
} from '@/lib/knowledgeApi';
import ActionButtons from '@/components/shared/ActionButtons';
import {
  getKnowledgeCategoryList,
  type KnowledgeCategory,
} from '@/lib/knowledgeApi';
import {
  getAllKnowledgeTags,
  type KnowledgeTag,
} from '@/lib/knowledgeApi';

const { Option } = Select;
const { TextArea } = Input;

// 文章状态配置
const STATUS_CONFIG = {
  [KnowledgeArticleStatus.Draft]: { text: '草稿', color: 'default' },
  [KnowledgeArticleStatus.Published]: { text: '已发布', color: 'green' },
};

import type { FC } from 'react';

const KnowledgeArticlePage: FC = () => {
  const [form] = Form.useForm();
  const [editForm] = Form.useForm();

  const [loading, setLoading] = useState(false);
  const [dataSource, setDataSource] = useState<KnowledgeArticle[]>([]);
  const [total, setTotal] = useState(0);
  const [current, setCurrent] = useState(1);
  const [pageSize, setPageSize] = useState(10);

  const [modalVisible, setModalVisible] = useState(false);
  const [previewVisible, setPreviewVisible] = useState(false);
  const [editingRecord, setEditingRecord] = useState<KnowledgeArticle | null>(null);
  const [previewRecord, setPreviewRecord] = useState<KnowledgeArticle | null>(null);

  // 统计数据
  const [stats, setStats] = useState({ total: 0, draft: 0, published: 0, totalViews: 0, totalLikes: 0 });

  // 分类和标签数据
  const [categories, setCategories] = useState<KnowledgeCategory[]>([]);
  const [tags, setTags] = useState<KnowledgeTag[]>([]);

  // 获取列表数据
  const fetchList = async (page?: number, pageSize?: number) => {
    setLoading(true);
    try {
      const values = form.getFieldsValue();
      const result = await getKnowledgeArticleList({
        current: page ?? current,
        pageSize: pageSize ?? pageSize,
        title: values.title,
        categoryId: values.categoryId,
        tagId: values.tagId,
        status: values.status,
      });
      setDataSource(result.data?.list ?? []);
      setTotal(result.data?.total ?? 0);
      setCurrent(result.data?.current ?? 1);
      setPageSize(result.data?.page_size ?? 10);

      // 计算统计数据
      const list = result.data?.list ?? [];
      setStats({
        total: list.length,
        draft: list.filter((a) => a.status === KnowledgeArticleStatus.Draft).length,
        published: list.filter((a) => a.status === KnowledgeArticleStatus.Published).length,
        totalViews: list.reduce((sum, a) => sum + (a.viewCount || 0), 0),
        totalLikes: list.reduce((sum, a) => sum + (a.likeCount || 0), 0),
      });
    } catch (error) {
      message.error('获取文章列表失败');
    } finally {
      setLoading(false);
    }
  };

  // 获取分类列表
  const fetchCategories = async () => {
    try {
      const result = await getKnowledgeCategoryList();
      setCategories(result.data || []);
    } catch (error) {
      console.error('获取分类列表失败:', error);
    }
  };

  // 获取标签列表
  const fetchTags = async () => {
    try {
      const result = await getAllKnowledgeTags();
      setTags(result.data || []);
    } catch (error) {
      console.error('获取标签列表失败:', error);
    }
  };

  useEffect(() => {
    fetchList(1, 10);
    fetchCategories();
    fetchTags();
  }, []);

  const handleTableChange = (pagination: TablePaginationConfig) => {
    fetchList(pagination.current, pagination.pageSize);
  };

  const handleSearch = () => {
    fetchList(1, 10);
  };

  const handleReset = () => {
    form.resetFields();
    fetchList(1, 10);
  };

  const handleAdd = () => {
    setEditingRecord(null);
    editForm.resetFields();
    editForm.setFieldsValue({
      status: KnowledgeArticleStatus.Draft,
      tagIds: [],
    });
    setModalVisible(true);
  };

  const handleEdit = (record: KnowledgeArticle) => {
    setEditingRecord(record);
    setModalVisible(true);
    setTimeout(() => {
      editForm.setFieldsValue({
        title: record.title,
        content: record.content,
        summary: record.summary,
        categoryId: record.categoryId,
        status: record.status,
        tagIds: record.tags?.map((t) => t.id) || [],
      });
    }, 0);
  };

  const handlePreview = (record: KnowledgeArticle) => {
    setPreviewRecord(record);
    setPreviewVisible(true);
  };

  const handleDelete = async (id: string) => {
    try {
      await deleteKnowledgeArticle(id);
      message.success('删除成功');
      fetchList();
    } catch (error: any) {
      message.error(error.message || '删除失败');
    }
  };

  const handleLike = async (id: string) => {
    try {
      await likeKnowledgeArticle(id);
      message.success('点赞成功');
      fetchList();
    } catch (error: any) {
      message.error(error.message || '点赞失败');
    }
  };

  const handleModalOk = async () => {
    try {
      const values = await editForm.validateFields();

      if (editingRecord) {
        const updateData: KnowledgeArticleUpdateRequest = {
          title: values.title,
          content: values.content,
          summary: values.summary,
          categoryId: values.categoryId,
          status: values.status,
          tagIds: values.tagIds,
        };
        await updateKnowledgeArticle(editingRecord.id, updateData);
        message.success('更新成功');
      } else {
        const createData: KnowledgeArticleCreateRequest = {
          title: values.title,
          content: values.content,
          summary: values.summary,
          categoryId: values.categoryId,
          status: values.status,
          tagIds: values.tagIds,
        };
        await createKnowledgeArticle(createData);
        message.success('创建成功');
      }
      setModalVisible(false);
      fetchList();
    } catch (error: any) {
      if (error.errorFields) {
        return;
      }
      message.error(editingRecord ? '更新失败' : '创建失败');
    }
  };

  const handlePublish = async (record: KnowledgeArticle) => {
    try {
      await updateKnowledgeArticle(record.id, { status: KnowledgeArticleStatus.Published });
      message.success('发布成功');
      fetchList();
    } catch (error: any) {
      message.error(error.message || '发布失败');
    }
  };

  // 扁平化分类树
  const flatCategories = useMemo(() => {
    const flatten = (cats: KnowledgeCategory[]): KnowledgeCategory[] => {
      const result: KnowledgeCategory[] = [];
      cats.forEach((cat) => {
        result.push(cat);
        if (cat.children && cat.children.length > 0) {
          result.push(...flatten(cat.children));
        }
      });
      return result;
    };
    return flatten(categories);
  }, [categories]);

  const columns: ColumnsType<KnowledgeArticle> = [
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
      render: (status: KnowledgeArticleStatus) => (
        <Tag color={STATUS_CONFIG[status]?.color}>{STATUS_CONFIG[status]?.text}</Tag>
      ),
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
      render: (date: string) => dayjs(date).format('YYYY-MM-DD HH:mm'),
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
                <Button type="link" danger icon={<DeleteOutlined />} size="small">
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

  return (
    <div className="p-6">
      <Card title={null} className="mb-4">
        <Row gutter={16}>
          <Col span={5}>
            <Statistic title="总文章" value={stats.total} prefix={<FileTextOutlined />} />
          </Col>
          <Col span={5}>
            <Statistic title="草稿" value={stats.draft} styles={{ content: { color: '#faad14' } }} />
          </Col>
          <Col span={5}>
            <Statistic title="已发布" value={stats.published} styles={{ content: { color: '#52c41a' } }} />
          </Col>
          <Col span={4}>
            <Statistic title="总浏览" value={stats.totalViews} />
          </Col>
          <Col span={5}>
            <Statistic title="总点赞" value={stats.totalLikes} />
          </Col>
        </Row>
      </Card>

      <Card title="知识库文章管理">
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', flexWrap: 'wrap', gap: '16px' }}>
          <Form form={form} layout="inline" style={{ flex: 1, minWidth: 0 }}>
            <Form.Item name="title" label="标题">
              <Input placeholder="请输入标题" allowClear style={{ width: 150 }} />
            </Form.Item>
            <Form.Item name="categoryId" label="分类">
              <Select placeholder="请选择分类" allowClear style={{ width: 120 }}>
                {flatCategories.map((cat) => (
                  <Option key={cat.id} value={cat.id}>
                    {cat.categoryName}
                  </Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item name="tagId" label="标签">
              <Select placeholder="请选择标签" allowClear style={{ width: 120 }}>
                {tags.map((tag) => (
                  <Option key={tag.id} value={tag.id}>
                    {tag.tagName}
                  </Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item name="status" label="状态">
              <Select placeholder="请选择状态" allowClear style={{ width: 100 }}>
                <Option value={KnowledgeArticleStatus.Draft}>草稿</Option>
                <Option value={KnowledgeArticleStatus.Published}>已发布</Option>
              </Select>
            </Form.Item>
            <Form.Item>
              <Space>
                <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>
                  查询
                </Button>
                <Button icon={<ReloadOutlined />} onClick={handleReset}>
                  重置
                </Button>
              </Space>
            </Form.Item>
          </Form>
          <Space>
            <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
              新增文章
            </Button>
          </Space>
        </div>

        <Table
          rowKey="id"
          columns={columns}
          dataSource={dataSource}
          loading={loading}
          scroll={{ x: 1300 }}
          pagination={{
            current,
            pageSize,
            total,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (t) => `共 ${t} 条`,
          }}
          onChange={handleTableChange}
        />
      </Card>

      {/* 新增/编辑弹窗 */}
      <Modal
        title={editingRecord ? '编辑文章' : '新增文章'}
        open={modalVisible}
        onOk={handleModalOk}
        onCancel={() => setModalVisible(false)}
        width={800}
        destroyOnHidden
      >
        <Form form={editForm} layout="vertical" preserve={false}>
          <Form.Item
            name="title"
            label="文章标题"
            rules={[{ required: true, message: '请输入文章标题' }]}
          >
            <Input placeholder="请输入文章标题" />
          </Form.Item>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="categoryId"
                label="文章分类"
                rules={[{ required: true, message: '请选择文章分类' }]}
              >
                <Select placeholder="请选择文章分类">
                  {flatCategories.map((cat) => (
                    <Option key={cat.id} value={cat.id}>
                      {cat.categoryName}
                    </Option>
                  ))}
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="status"
                label="状态"
                rules={[{ required: true, message: '请选择状态' }]}
              >
                <Select placeholder="请选择状态">
                  <Option value={KnowledgeArticleStatus.Draft}>草稿</Option>
                  <Option value={KnowledgeArticleStatus.Published}>已发布</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Form.Item
            name="tagIds"
            label="标签"
          >
            <Select mode="tags" placeholder="请选择标签" style={{ width: '100%' }}>
              {tags.map((tag) => (
                <Option key={tag.id} value={tag.id}>
                  {tag.tagName}
                </Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item
            name="summary"
            label="摘要"
          >
            <TextArea rows={2} placeholder="请输入摘要" />
          </Form.Item>

          <Form.Item
            name="content"
            label="文章内容"
            rules={[{ required: true, message: '请输入文章内容' }]}
          >
            <TextArea rows={10} placeholder="请输入文章内容（支持Markdown格式）" />
          </Form.Item>
        </Form>
      </Modal>

      {/* 预览弹窗 */}
      <Modal
        title="文章预览"
        open={previewVisible}
        onCancel={() => setPreviewVisible(false)}
        footer={null}
        width={800}
      >
        {previewRecord && (
          <div>
            <h1 className="text-xl font-bold mb-4">{previewRecord.title}</h1>
            <div className="mb-4 text-gray-500 text-sm">
              <Space>
                <span>分类: {previewRecord.category?.categoryName}</span>
                <span>浏览: {previewRecord.viewCount}</span>
                <span>点赞: {previewRecord.likeCount}</span>
              </Space>
            </div>
            {previewRecord.summary && (
              <div className="mb-4 p-4 bg-gray-50 rounded">
                <strong>摘要：</strong>{previewRecord.summary}
              </div>
            )}
            <div className="prose max-w-none" style={{ whiteSpace: 'pre-wrap' }}>
              {previewRecord.content}
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
};

export default KnowledgeArticlePage;
