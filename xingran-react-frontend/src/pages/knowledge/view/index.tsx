import { useState, useEffect, useMemo } from "react";
import { useLocation } from "react-router-dom";
import { usePersistedStateController } from "@/hooks/usePersistedState";
import { App, Input, Card, Row, Col, Tag, Space, Empty, Spin, Drawer, Button } from "antd";
import {
  EyeOutlined,
  LikeOutlined,
  FolderOutlined,
  TagOutlined,
  SearchOutlined,
} from "@ant-design/icons";
import {
  searchKnowledgeArticles,
  getKnowledgeArticle,
  likeKnowledgeArticle,
  type KnowledgeArticle,
} from "@/lib/knowledgeApi";
import { formatDateTime } from "@/utils/datetime";
import type { UnknownError } from "@/types/common";
import { getKnowledgeCategoryList, type KnowledgeCategory } from "@/lib/knowledgeApi";
import { getAllKnowledgeTags, type KnowledgeTag } from "@/lib/knowledgeApi";

const { Search } = Input;

import type { FC } from "react";

const KnowledgeViewPage: FC = () => {
  const { message } = App.useApp();
  const [loading, setLoading] = useState(false);
  const [articles, setArticles] = useState<KnowledgeArticle[]>([]);
  const [categories, setCategories] = useState<KnowledgeCategory[]>([]);
  const [tags, setTags] = useState<KnowledgeTag[]>([]);

  const location = useLocation();
  const [selectedCategory, setSelectedCategory] = usePersistedStateController<string>({
    keyPrefix: location.pathname,
    keySuffix: "selectedCategory",
    defaultValue: "",
  });
  const [selectedTag, setSelectedTag] = usePersistedStateController<string>({
    keyPrefix: location.pathname,
    keySuffix: "selectedTag",
    defaultValue: "",
  });
  const [keyword, setKeyword] = usePersistedStateController<string>({
    keyPrefix: location.pathname,
    keySuffix: "keyword",
    defaultValue: "",
  });

  const [detailVisible, setDetailVisible] = useState(false);
  const [selectedArticle, setSelectedArticle] = useState<KnowledgeArticle | null>(null);

  // 获取文章列表
  const fetchArticles = async () => {
    setLoading(true);
    try {
      const result = await searchKnowledgeArticles({
        keyword,
        categoryId: selectedCategory,
        tagId: selectedTag,
      });
      setArticles(result.data?.list || []);
    } catch (error) {
      message.error("搜索失败");
    } finally {
      setLoading(false);
    }
  };

  // 获取分类列表
  const fetchCategories = async () => {
    try {
      const result = await getKnowledgeCategoryList({ status: 1 });
      setCategories(result.data || []);
    } catch (error) {
      console.error("获取分类列表失败:", error);
    }
  };

  // 获取标签列表
  const fetchTags = async () => {
    try {
      const result = await getAllKnowledgeTags();
      setTags(result.data || []);
    } catch (error) {
      console.error("获取标签列表失败:", error);
    }
  };

  useEffect(() => {
    fetchArticles();
    fetchCategories();
    fetchTags();
  }, [selectedCategory, selectedTag, keyword]);

  const handleSearch = (value: string) => {
    setKeyword(value);
  };

  const handleCategoryClick = (categoryId: string) => {
    setSelectedCategory(categoryId === selectedCategory ? "" : categoryId);
  };

  const handleTagClick = (tagId: string) => {
    setSelectedTag(tagId === selectedTag ? "" : tagId);
  };

  const handleViewArticle = async (article: KnowledgeArticle) => {
    setSelectedArticle(article);
    setDetailVisible(true);

    // 增加浏览次数
    try {
      await getKnowledgeArticle(article.id);
    } catch (error) {
      console.error("增加浏览次数失败:", error);
    }
  };

  const handleLike = async (id: string) => {
    try {
      await likeKnowledgeArticle(id);
      message.success("点赞成功");
      fetchArticles();
    } catch (error: unknown) {
      const err = error as UnknownError;
      message.error(err.message || "点赞失败");
    }
  };

  // 扁平化分类树
  const flatCategories = useMemo(() => {
    const flatten = (cats: KnowledgeCategory[]): KnowledgeCategory[] => {
      const result: KnowledgeCategory[] = [];
      for (const cat of cats) {
        result.push(cat);
        if (cat.children && cat.children.length > 0) {
          result.push(...flatten(cat.children));
        }
      }
      return result;
    };
    return flatten(categories);
  }, [categories]);

  // 分类树项组件
  const CategoryTreeItem: FC<{ category: KnowledgeCategory; level: number }> = ({
    category,
    level,
  }) => {
    const isSelected = selectedCategory === category.id;
    const hasChildren = category.children && category.children.length > 0;

    return (
      <div>
        <div
          className={`flex items-center justify-between py-2 px-3 cursor-pointer rounded hover:bg-gray-100 ${
            isSelected ? "bg-blue-50 text-blue-600" : ""
          }`}
          onClick={() => handleCategoryClick(category.id)}
          style={{ marginLeft: `${level * 16}px` }}
        >
          <Space>
            <FolderOutlined />
            <span>{category.categoryName}</span>
          </Space>
        </div>
        {hasChildren &&
          category.children!.map((child) => (
            <CategoryTreeItem key={child.id} category={child} level={level + 1} />
          ))}
      </div>
    );
  };

  return (
    <div className="p-6">
      <div className="max-w-7xl mx-auto">
        {/* 搜索栏 */}
        <Card className="mb-6">
          <Search
            placeholder="搜索知识库文章..."
            allowClear
            className="user-form-input"
            enterButton={
              <Button type="primary" icon={<SearchOutlined />}>
                搜索
              </Button>
            }
            size="large"
            onSearch={handleSearch}
            onChange={(e) => {
              if (e.target.value === "") {
                setKeyword("");
              }
            }}
          />
        </Card>

        <Row gutter={16}>
          {/* 左侧分类和标签 */}
          <Col span={6}>
            {/* 分类列表 */}
            <Card title="文章分类" className="mb-4" size="small">
              {categories.length === 0 ? (
                <Empty description="暂无分类" image={Empty.PRESENTED_IMAGE_SIMPLE} />
              ) : (
                <div>
                  {categories.map((cat) => (
                    <CategoryTreeItem key={cat.id} category={cat} level={0} />
                  ))}
                </div>
              )}
            </Card>

            {/* 标签列表 */}
            <Card title="热门标签" size="small">
              <Space size={[8, 8]} wrap>
                {tags.slice(0, 20).map((tag) => (
                  <Tag
                    key={tag.id}
                    icon={<TagOutlined />}
                    className="cursor-pointer"
                    color={selectedTag === tag.id ? "blue" : "default"}
                    onClick={() => handleTagClick(tag.id)}
                  >
                    {tag.tagName} ({tag.useCount})
                  </Tag>
                ))}
              </Space>
            </Card>
          </Col>

          {/* 右侧文章列表 */}
          <Col span={18}>
            <Card
              title={
                <Space>
                  <span>文章列表</span>
                  {selectedCategory && (
                    <Tag closable onClose={() => setSelectedCategory("")}>
                      {flatCategories.find((c) => c.id === selectedCategory)?.categoryName}
                    </Tag>
                  )}
                  {selectedTag && (
                    <Tag closable onClose={() => setSelectedTag("")}>
                      {tags.find((t) => t.id === selectedTag)?.tagName}
                    </Tag>
                  )}
                </Space>
              }
            >
              <Spin spinning={loading}>
                {articles.length === 0 ? (
                  <Empty description="暂无文章" />
                ) : (
                  <Row gutter={[16, 16]}>
                    {articles.map((article) => (
                      <Col span={8} key={article.id}>
                        <Card
                          hoverable
                          className="h-full"
                          onClick={() => handleViewArticle(article)}
                        >
                          <div className="mb-2">
                            <h3 className="text-base font-semibold truncate">{article.title}</h3>
                          </div>
                          {article.summary && (
                            <p className="text-gray-500 text-sm mb-3 line-clamp-2">
                              {article.summary}
                            </p>
                          )}
                          <div className="mb-2">
                            <Tag color="blue">{article.category?.categoryName}</Tag>
                            {article.tags &&
                              article.tags.slice(0, 3).map((tag) => (
                                <Tag key={tag.id} color="default">
                                  {tag.tagName}
                                </Tag>
                              ))}
                          </div>
                          <div className="flex justify-between items-center text-gray-400 text-xs">
                            <Space>
                              <span>
                                <EyeOutlined className="mr-1" />
                                {article.viewCount || 0}
                              </span>
                              <span>
                                <LikeOutlined className="mr-1" />
                                {article.likeCount || 0}
                              </span>
                            </Space>
                            <span>{formatDateTime(article.createdAt, "YYYY-MM-DD")}</span>
                          </div>
                        </Card>
                      </Col>
                    ))}
                  </Row>
                )}
              </Spin>
            </Card>
          </Col>
        </Row>

        {/* 文章详情抽屉 */}
        <Drawer
          title={null}
          placement="right"
          size="large"
          open={detailVisible}
          onClose={() => setDetailVisible(false)}
        >
          {selectedArticle && (
            <div>
              <h1 className="text-xl font-bold mb-4">{selectedArticle.title}</h1>

              <div className="mb-4 pb-4 border-b">
                <Space orientation="vertical" className="w-full">
                  <div className="flex justify-between">
                    <Space>
                      <Tag color="blue">{selectedArticle.category?.categoryName}</Tag>
                      {selectedArticle.tags &&
                        selectedArticle.tags.map((tag) => <Tag key={tag.id}>{tag.tagName}</Tag>)}
                    </Space>
                  </div>
                  <div className="flex justify-between text-gray-400 text-sm">
                    <Space>
                      <span>
                        <EyeOutlined className="mr-1" />
                        浏览: {selectedArticle.viewCount || 0}
                      </span>
                      <span>
                        <LikeOutlined className="mr-1" />
                        点赞: {selectedArticle.likeCount || 0}
                      </span>
                    </Space>
                    <span>{formatDateTime(selectedArticle.createdAt, "YYYY-MM-DD HH:mm")}</span>
                  </div>
                </Space>
              </div>

              {selectedArticle.summary && (
                <div className="mb-4 p-4 bg-gray-50 rounded">
                  <strong>摘要：</strong>
                  <p className="mt-1">{selectedArticle.summary}</p>
                </div>
              )}

              <div className="prose max-w-none" style={{ whiteSpace: "pre-wrap" }}>
                {selectedArticle.content}
              </div>

              <div className="mt-6 pt-4 border-t">
                <Button
                  type="primary"
                  icon={<LikeOutlined />}
                  onClick={() => handleLike(selectedArticle.id)}
                >
                  点赞
                </Button>
              </div>
            </div>
          )}
        </Drawer>
      </div>
    </div>
  );
};

export default KnowledgeViewPage;
