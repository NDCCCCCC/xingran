/**
 * Knowledge Article Modals
 * 知识文章模态框组件
 */

import { Modal, Form, Row, Col, Space } from "antd";
// Phase 71 (W1)：裸 Input/Select/TextArea → 契约组件。本文件 Select 全为 children 形态，
// 一律改写为 options 形态（AppSelect 无条件 optionFilterProp="label"，children 下破坏性）
import { AppInput, AppSelect, AppTextArea } from "@/components/ui";
import { EyeOutlined, LikeOutlined } from "@ant-design/icons";
import type { FormInstance } from "antd/es/form";
import type { KnowledgeArticle, KnowledgeCategory, KnowledgeTag } from "@/lib/knowledgeApi";
import { STATUS_OPTIONS } from "../constants";

export interface EditModalProps {
  open: boolean;
  editingRecord: KnowledgeArticle | null;
  categories: KnowledgeCategory[];
  flatCategories: KnowledgeCategory[];
  tags: KnowledgeTag[];
  onOk: (form: FormInstance<unknown>) => Promise<void>;
  onCancel: () => void;
}

export function EditModal({
  open,
  editingRecord,
  flatCategories,
  tags,
  onOk,
  onCancel,
}: EditModalProps) {
  const [form] = Form.useForm();

  return (
    <Modal
      title={editingRecord ? "编辑文章" : "新增文章"}
      open={open}
      onOk={() => onOk(form)}
      onCancel={() => {
        onCancel();
        form.resetFields();
      }}
      width={800}
      destroyOnHidden
    >
      <Form
        form={form}
        layout="horizontal"
        labelCol={{ span: 4 }}
        wrapperCol={{ span: 20 }}
        preserve={false}
      >
        <Form.Item
          name="title"
          label="文章标题"
          rules={[{ required: true, message: "请输入文章标题" }]}
        >
          <AppInput placeholder="请输入文章标题" className="user-form-input" />
        </Form.Item>

        <Row gutter={16}>
          <Col span={12}>
            <Form.Item
              name="categoryId"
              label="文章分类"
              rules={[{ required: true, message: "请选择文章分类" }]}
            >
              {/* 形态分类器③：动态 children → options 改写（label 真实存在） */}
              <AppSelect
                allowClear
                placeholder="请选择文章分类"
                className="user-form-input"
                onSearch={() => {}}
                options={flatCategories.map((cat) => ({
                  label: cat.categoryName,
                  value: cat.id,
                }))}
              />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item
              name="status"
              label="状态"
              rules={[{ required: true, message: "请选择状态" }]}
            >
              {/* 形态分类器②④：状态枚举（业务禁搜）→ options 形态 + 显式 showSearch={false} */}
              <AppSelect
                allowClear
                placeholder="请选择状态"
                className="user-form-input"
                onSearch={() => {}}
                showSearch={false}
                // STATUS_OPTIONS 为 as const 只读元组，antd options 需可变数组 → 展开复制
                options={[...STATUS_OPTIONS]}
              />
            </Form.Item>
          </Col>
        </Row>

        <Form.Item name="tagIds" label="标签">
          {/* 形态分类器③ + mode="tags"：tags 模式自建输入项，配合 options 形态需保证
              label 存在（否则 optionFilterProp="label" 破坏性）；tags 模式下搜索是核心
              交互，显式 showSearch 保持开启（不依赖 count>=8 自动策略） */}
          <AppSelect
            allowClear
            mode="tags"
            placeholder="请选择标签"
            style={{ width: "100%" }}
            className="user-form-input"
            onSearch={() => {}}
            showSearch
            options={tags.map((tag) => ({ label: tag.tagName, value: tag.id }))}
          />
        </Form.Item>

        <Form.Item name="summary" label="摘要">
          <AppTextArea rows={2} placeholder="请输入摘要" />
        </Form.Item>

        <Form.Item
          name="content"
          label="文章内容"
          rules={[{ required: true, message: "请输入文章内容" }]}
        >
          <AppTextArea rows={10} placeholder="请输入文章内容（支持Markdown格式）" />
        </Form.Item>
      </Form>
    </Modal>
  );
}

export interface PreviewModalProps {
  open: boolean;
  previewRecord: KnowledgeArticle | null;
  onClose: () => void;
}

export function PreviewModal({ open, previewRecord, onClose }: PreviewModalProps) {
  return (
    <Modal title="文章预览" open={open} onCancel={onClose} footer={null} width={800}>
      {previewRecord && (
        <div>
          <h1 className="text-xl font-bold mb-4">{previewRecord.title}</h1>
          <div className="mb-4 text-gray-500 text-sm">
            <Space>
              <span>分类: {previewRecord.category?.categoryName}</span>
              <span>
                <EyeOutlined className="mr-1" />
                {previewRecord.viewCount}
              </span>
              <span>
                <LikeOutlined className="mr-1" />
                {previewRecord.likeCount}
              </span>
            </Space>
          </div>
          {previewRecord.summary && (
            <div className="mb-4 p-4 bg-gray-50 rounded">
              <strong>摘要：</strong>
              {previewRecord.summary}
            </div>
          )}
          <div className="prose max-w-none" style={{ whiteSpace: "pre-wrap" }}>
            {previewRecord.content}
          </div>
        </div>
      )}
    </Modal>
  );
}
