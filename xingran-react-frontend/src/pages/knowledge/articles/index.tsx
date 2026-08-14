import { useState, useEffect, useMemo, useCallback, useRef, type FC } from "react";
import { Table, Button, Form, Input, Select, Card, Row, Col, Statistic, Space, App } from "antd";
import { PlusOutlined, SearchOutlined, ReloadOutlined, FileTextOutlined } from "@ant-design/icons";
import { useArticleData } from "./hooks";
import { getArticleColumns } from "./columns";
import { EditModal, PreviewModal } from "./modals";
import { KnowledgeArticleStatus, type KnowledgeArticle } from "@/lib/knowledgeApi";
import { useServerSort, resolveSorter } from "@/hooks/useServerSort";
import type { SorterMeta } from "@/utils/tableHelpers";
import { createSorterMeta } from "@/utils/tableHelpers";
import { usePagination } from "@/hooks/usePagination";
import { isFormValidationError } from "@/utils/errorHandler";

const { Option } = Select;

const KnowledgeArticlePage: FC = () => {
  const { message } = App.useApp();
  const [form] = Form.useForm();
  const [editForm] = Form.useForm();

  // 使用全局分页 hook
  const { paginationProps, setCurrent, setPageSize, setTotal: _setTotal } = usePagination();
  const [modalVisible, setModalVisible] = useState(false);
  const [previewVisible, setPreviewVisible] = useState(false);
  const [editingRecord, setEditingRecord] = useState<KnowledgeArticle | null>(null);
  const [previewRecord, setPreviewRecord] = useState<KnowledgeArticle | null>(null);

  // 数据管理
  const {
    articles,
    categories,
    flatCategories,
    tags,
    loading,
    total: _total,
    statistics,
    fetchList,
    fetchCategories,
    fetchTags,
    handleDelete,
    handleLike,
    handlePublish,
    handleSave,
  } = useArticleData({
    current: paginationProps.current ?? 1,
    pageSize: paginationProps.pageSize ?? 10,
  });

  // 服务端排序:field 对应后端 knowledgeArticleAllowedSortFields 白名单 key
  const sorterMetas = useMemo<Array<SorterMeta<KnowledgeArticle> | undefined>>(
    () => [
      createSorterMeta<KnowledgeArticle>("title"),
      createSorterMeta<KnowledgeArticle>("status", "number"),
      createSorterMeta<KnowledgeArticle>("viewCount", "number"),
      createSorterMeta<KnowledgeArticle>("likeCount", "number"),
      createSorterMeta<KnowledgeArticle>("createdAt", "date"),
    ],
    []
  );
  const sort = useServerSort<KnowledgeArticle>({ sorterMetas });

  // 列级 sortOrder：只对当前排序列返回方向，其余 undefined。
  const getColumnSortOrder = useCallback(
    (field: string): "ascend" | "descend" | null | undefined => {
      if (sort.orderByColumn !== String(field)) return undefined;
      return sort.sortOrder;
    },
    [sort.orderByColumn, sort.sortOrder]
  );

  // 排序 ref：resolveSorter 同步取排序新值，规避 setState 时序
  const sortRef = useRef<{ orderByColumn?: string; isAsc?: boolean }>({});

  // 初始化加载
  useEffect(() => {
    fetchList(paginationProps.current ?? 1, paginationProps.pageSize ?? 10);
    fetchCategories();
    fetchTags();
    // eslint-disable-next-line react-hooks/exhaustive-deps -- intentional mount-only load; fetch handlers are stable
  }, [paginationProps.current, paginationProps.pageSize]);

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
          tagIds: record.tags?.map((t: { id: string }) => t.id) || [],
        });
      }, 0);
      setModalVisible(true);
    },
    handlePublish,
    handleLike,
    handleDelete,
    current: paginationProps.current ?? 1,
    pageSize: paginationProps.pageSize ?? 10,
    getColumnSortOrder,
    sorterMetas,
  });

  // 搜索
  const handleSearch = () => {
    setCurrent(1);
    const _values = form.getFieldsValue() as {
      title?: string;
      categoryId?: string;
      tagId?: string;
      status?: number;
    };
    fetchList(
      1,
      paginationProps.pageSize ?? 10,
      sortRef.current.orderByColumn,
      sortRef.current.isAsc
    );
  };

  // 重置
  const handleReset = () => {
    form.resetFields();
    setCurrent(1);
    sort.resetSort();
    sortRef.current = {};
    fetchList(1, paginationProps.pageSize);
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
    } catch (error: unknown) {
      if (isFormValidationError(error)) return;
      message.error(editingRecord ? "更新失败" : "创建失败");
    }
  };

  // 表格分页+排序变化
  const handleTableChange = useCallback(
    (
      pagination: { current?: number; pageSize?: number },
      _filters: Record<string, any>,
      sorter:
        | import("antd/es/table/interface").SorterResult<KnowledgeArticle>
        | import("antd/es/table/interface").SorterResult<KnowledgeArticle>[]
    ) => {
      const current = pagination.current ?? 1;
      const pageSize = pagination.pageSize ?? 10;
      setCurrent(current);
      setPageSize(pageSize);
      // 排序受控 UI
      sort.handleTableChange(pagination, _filters, sorter);
      // 同步取排序值写 ref
      const { orderByColumn, isAsc } = resolveSorter(sorter, sorterMetas);
      sortRef.current = { orderByColumn, isAsc };
      // 读搜索条件
      const _values = form.getFieldsValue() as {
        title?: string;
        categoryId?: string;
        tagId?: string;
        status?: number;
      };
      fetchList(current, pageSize, orderByColumn, isAsc);
    },
    [sort, sorterMetas, form, fetchList, setCurrent, setPageSize]
  );

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
              <Statistic
                title="草稿"
                value={statistics.draft}
                styles={{ content: { color: "var(--theme-warning, #faad14)" } }}
              />
            </Col>
            <Col span={5}>
              <Statistic
                title="已发布"
                value={statistics.published}
                styles={{ content: { color: "var(--theme-success, #52c41a)" } }}
              />
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
        <div
          style={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "flex-start",
            flexWrap: "wrap",
            gap: "16px",
          }}
        >
          <Form form={form} layout="inline" style={{ flex: 1, minWidth: 0 }}>
            <Form.Item name="title" label="标题">
              <Input
                placeholder="请输入标题"
                allowClear
                className="user-form-input"
                style={{ width: 150 }}
              />
            </Form.Item>
            <Form.Item name="categoryId" label="分类">
              <Select
                placeholder="请选择分类"
                allowClear
                className="user-form-input"
                style={{ width: 120 }}
                onSearch={() => {}}
              >
                {flatCategories.map((cat) => (
                  <Option key={cat.id} value={cat.id}>
                    {cat.categoryName}
                  </Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item name="tagId" label="标签">
              <Select
                placeholder="请选择标签"
                allowClear
                className="user-form-input"
                style={{ width: 120 }}
                onSearch={() => {}}
              >
                {tags.map((tag) => (
                  <Option key={tag.id} value={tag.id}>
                    {tag.tagName}
                  </Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item name="status" label="状态">
              <Select
                placeholder="请选择状态"
                allowClear
                className="user-form-input"
                style={{ width: 100 }}
                onSearch={() => {}}
              >
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
          pagination={paginationProps}
          onChange={handleTableChange}
        />
      </Card>

      {/* 新增/编辑弹窗 */}
      <EditModal
        open={modalVisible}
        editingRecord={editingRecord}
        categories={categories}
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
