import { useState, useEffect, type FC } from 'react';
import {
  Table,
  Button,
  Form,
  Input,
  Select,
  Card,
  Row,
  Col,
  Statistic,
  Space,
} from 'antd';
import {
  PlusOutlined,
  SearchOutlined,
  ReloadOutlined,
  FileTextOutlined,
} from '@ant-design/icons';
import { useArticleData } from './hooks';
import { getArticleColumns } from './columns';
import { EditModal, PreviewModal } from './modals';
import { KnowledgeArticleStatus } from '@/lib/knowledgeApi';
import { message } from 'antd';

const { Option } = Select;

const KnowledgeArticlePage: FC = () => {
  const [form] = Form.useForm();
  const [editForm] = Form.useForm();

  const [current, setCurrent] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [modalVisible, setModalVisible] = useState(false);
  const [previewVisible, setPreviewVisible] = useState(false);
  const [editingRecord, setEditingRecord] = useState<any | null>(null);
  const [previewRecord, setPreviewRecord] = useState<any | null>(null);

  // 数据管理
  const {
    articles,
    flatCategories,
    tags,
    loading,
    total,
    statistics,
    fetchList,
    fetchCategories,
    fetchTags,
    handleDelete,
    handleLike,
    handlePublish,
    handleSave,
  } = useArticleData({
    current,
    pageSize,
  });

  // 初始化加载
  useEffect(() => {
    fetchList(1, 10);
    fetchCategories();
    fetchTags();
  }, []);

  // 表格列
  const columns = getArticleColumns({
    handlePreview: (record) => {
      setPreviewRecord(record);
      setPreviewVisible(true);
    },
    handleEdit: (record) => {
      setEditingRecord(record);
      setTimeout(() => {
        editForm.setFieldsValue({
          title: record.title,
          content: record.content,
          summary: record.summary,
          categoryId: record.categoryId,
          status: record.status,
          tagIds: record.tags?.map((t: any) => t.id) || [],
        });
      }, 0);
      setModalVisible(true);
    },
    handlePublish,
    handleLike,
    handleDelete,
    current,
    pageSize,
  });

  // 搜索
  const handleSearch = () => {
    fetchList(1, 10);
  };

  // 重置
  const handleReset = () => {
    form.resetFields();
    fetchList(1, 10);
  };

  // 新增
  const handleAdd = () => {
    setEditingRecord(null);
    editForm.resetFields();
    editForm.setFieldsValue({
      status: KnowledgeArticleStatus.Draft,
      tagIds: [],
    });
    setModalVisible(true);
  };

  // 保存
  const onModalOk = async () => {
    try {
      await handleSave(editingRecord, await editForm.validateFields());
      setModalVisible(false);
    } catch (error: any) {
      if (error.errorFields) return;
      message.error(editingRecord ? '更新失败' : '创建失败');
    }
  };

  // 表格分页变化
  const handleTableChange = (pagination: any) => {
    setCurrent(pagination.current);
    setPageSize(pagination.pageSize);
    fetchList(pagination.current, pagination.pageSize);
  };

  return (
    <div className="p-6">
      {/* 统计卡片 */}
      {statistics.total > 10 && (
        <Card title={null} className="mb-4">
          <Row gutter={16}>
            <Col span={5}>
              <Statistic title="总文章" value={statistics.total} prefix={<FileTextOutlined />} />
            </Col>
            <Col span={5}>
              <Statistic title="草稿" value={statistics.draft} styles={{ content: { color: '#faad14' } }} />
            </Col>
            <Col span={5}>
              <Statistic title="已发布" value={statistics.published} styles={{ content: { color: '#52c41a' } }} />
            </Col>
            <Col span={4}>
              <Statistic title="总浏览" value={statistics.totalViews} />
            </Col>
            <Col span={5}>
              <Statistic title="总点赞" value={statistics.totalLikes} />
            </Col>
          </Row>
        </Card>
      )}

      {/* 筛选表单和操作按钮 */}
      <Card style={{ marginBottom: 16 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', flexWrap: 'wrap', gap: '16px' }}>
          <Form form={form} layout="inline" style={{ flex: 1, minWidth: 0 }}>
            <Form.Item name="title" label="标题">
              <Input placeholder="请输入标题" allowClear className="user-form-input" style={{ width: 150 }} />
            </Form.Item>
            <Form.Item name="categoryId" label="分类">
              <Select placeholder="请选择分类" allowClear className="user-form-input" style={{ width: 120 }}>
                {flatCategories.map((cat) => (
                  <Option key={cat.id} value={cat.id}>
                    {cat.categoryName}
                  </Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item name="tagId" label="标签">
              <Select placeholder="请选择标签" allowClear className="user-form-input" style={{ width: 120 }}>
                {tags.map((tag) => (
                  <Option key={tag.id} value={tag.id}>
                    {tag.tagName}
                  </Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item name="status" label="状态">
              <Select placeholder="请选择状态" allowClear className="user-form-input" style={{ width: 100 }}>
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
      </Card>

      {/* 知识库文章表格 */}
      <Card>
        <Table
          rowKey="id"
          columns={columns}
          dataSource={articles}
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
      <EditModal
        open={modalVisible}
        editingRecord={editingRecord}
        flatCategories={flatCategories}
        tags={tags}
        onOk={onModalOk}
        onCancel={() => setModalVisible(false)}
      />

      {/* 预览弹窗 */}
      <PreviewModal
        open={previewVisible}
        previewRecord={previewRecord}
        onClose={() => setPreviewVisible(false)}
      />
    </div>
  );
};

export default KnowledgeArticlePage;
