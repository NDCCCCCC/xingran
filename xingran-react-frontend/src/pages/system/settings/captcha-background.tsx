import { useState, useEffect } from "react";
import type { FC } from "react";
import {
  App,
  Table,
  Button,
  Space,
  Modal,
  Form,
  Input,
  Select,
  Upload,
  Tag,
  Image,
  Statistic,
  Card,
  Row,
  Col,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import type { UploadFile, UploadProps } from "antd/es/upload";
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  ReloadOutlined,
  UploadOutlined,
  PictureOutlined,
} from "@ant-design/icons";
import type {
  CaptchaBackground,
  CaptchaBackgroundUpdateRequest,
  PieceShape,
  DifficultyLevel,
  CaptchaBackgroundStatus,
} from "@/types/captcha";
import * as captchaService from "@/services/captcha";
import { usePagination } from "@/hooks/usePagination";
import { isFormValidationError } from "@/utils/errorHandler";

const { Option } = Select;
const { TextArea } = Input;

// 拼图形状映射
const PIECE_SHAPE_MAP: Record<PieceShape, string> = {
  circle: "圆形",
  square: "方形",
  star: "星形",
  heart: "心形",
};

// 难度级别映射
const DIFFICULTY_MAP: Record<DifficultyLevel, string> = {
  1: "简单",
  2: "中等",
  3: "困难",
};

// 形状选项
const SHAPE_OPTIONS = [
  { label: "圆形", value: "circle" },
  { label: "方形", value: "square" },
  { label: "星形", value: "star" },
  { label: "心形", value: "heart" },
];

const CaptchaBackgroundSettingsPage: FC = () => {
  const { message } = App.useApp();
  const [backgrounds, setBackgrounds] = useState<CaptchaBackground[]>([]);
  const [loading, setLoading] = useState(false);
  const [searchForm] = Form.useForm();
  const [uploadModalVisible, setUploadModalVisible] = useState(false);
  const [editModalVisible, setEditModalVisible] = useState(false);
  const [previewVisible, setPreviewVisible] = useState(false);
  const [previewImage, _setPreviewImage] = useState<string | null>(null);
  const [editForm] = Form.useForm();
  const [uploadForm] = Form.useForm();
  const [editingBg, setEditingBg] = useState<CaptchaBackground | null>(null);
  const [fileList, setFileList] = useState<UploadFile[]>([]);
  const [uploading, setUploading] = useState(false);
  const [statistics, setStatistics] = useState<{ totalCount: number; enabledCount: number; disabledCount: number } | null>(null);

  // 使用全局分页 hook
  const { paginationProps, setCurrent, setPageSize, setTotal } = usePagination();

  // 加载背景图列表
  const loadBackgrounds = async (params: { current?: number; pageSize?: number } = {}) => {
    setLoading(true);
    try {
      const values = searchForm.getFieldsValue();
      const result = await captchaService.getCaptchaBackgroundList({
        current: params.current || paginationProps.current,
        pageSize: params.pageSize || paginationProps.pageSize,
        ...values,
      });
      setBackgrounds(result.items);
      setTotal(result.total);
    } catch (error) {
      console.error("加载背景图列表失败:", error);
    } finally {
      setLoading(false);
    }
  };

  // 加载统计信息
  const loadStatistics = async () => {
    try {
      const stats = await captchaService.getCaptchaBackgroundStatistics();
      setStatistics(stats);
    } catch (error) {
      console.error("加载统计信息失败:", error);
    }
  };

  // 上传背景图
  const handleUpload = async () => {
    try {
      const values = await uploadForm.validateFields();
      if (fileList.length === 0) {
        message.warning("请选择要上传的图片");
        return;
      }

      setUploading(true);
      const file = fileList[0].originFileObj as File;

      await captchaService.uploadCaptchaBackground(file, {
        pieceShape: values.pieceShape,
        difficultyLevel: values.difficultyLevel,
        allowedShapes: values.allowedShapes,
        remark: values.remark,
      });

      message.success("上传成功");
      setUploadModalVisible(false);
      uploadForm.resetFields();
      setFileList([]);
      loadBackgrounds();
      loadStatistics();
    } catch (error: unknown) {
      if (isFormValidationError(error)) {
        return;
      }
      message.error("上传失败");
    } finally {
      setUploading(false);
    }
  };

  // 更新背景图
  const handleUpdate = async () => {
    try {
      const values = await editForm.validateFields();
      const updateData: CaptchaBackgroundUpdateRequest = {
        pieceShape: values.pieceShape,
        difficultyLevel: values.difficultyLevel,
        allowedShapes: values.allowedShapes,
        status: values.status,
        sortOrder: values.sortOrder,
        remark: values.remark,
      };

      await captchaService.updateCaptchaBackground(editingBg!.id, updateData);
      message.success("更新成功");
      setEditModalVisible(false);
      editForm.resetFields();
      setEditingBg(null);
      loadBackgrounds();
    } catch (error: unknown) {
      if (isFormValidationError(error)) {
        return;
      }
      message.error("更新失败");
    }
  };

  // 删除背景图
  const handleDelete = async (id: string) => {
    try {
      await captchaService.deleteCaptchaBackground(id);
      message.success("删除成功");
      loadBackgrounds();
      loadStatistics();
    } catch (error) {
      message.error("删除失败");
    }
  };

  // 切换状态
  const handleToggle = async (id: string) => {
    try {
      await captchaService.toggleCaptchaBackgroundStatus(id);
      message.success("状态更新成功");
      loadBackgrounds();
      loadStatistics();
    } catch (error) {
      message.error("状态更新失败");
    }
  };

  // 预加载缓存
  const handlePreload = async () => {
    try {
      await captchaService.preloadCaptchaCache();
      message.success("预加载成功");
    } catch (error) {
      message.error("预加载失败");
    }
  };

  // 打开编辑模态框
  const openEditModal = (record: CaptchaBackground) => {
    setEditingBg(record);
    editForm.setFieldsValue({
      pieceShape: record.pieceShape,
      difficultyLevel: record.difficultyLevel,
      allowedShapes: record.allowedShapes,
      status: record.status,
      sortOrder: record.sortOrder,
      remark: record.remark,
    });
    setEditModalVisible(true);
  };

  // 上传配置
  const uploadProps: UploadProps = {
    listType: "picture-card",
    fileList,
    onChange: ({ fileList: newFileList }) => {
      setFileList(newFileList);
    },
    beforeUpload: (file) => {
      const isImage = file.type.startsWith("image/");
      if (!isImage) {
        message.error("只能上传图片文件");
        return Upload.LIST_IGNORE;
      }
      const isLt2M = file.size / 1024 / 1024 < 2;
      if (!isLt2M) {
        message.error("图片大小不能超过 2MB");
        return Upload.LIST_IGNORE;
      }
      return false; // 阻止自动上传
    },
    maxCount: 1,
  };

  // 表格列
  const columns: ColumnsType<CaptchaBackground> = [
    {
      title: "预览",
      key: "preview",
      width: 80,
      render: (_, record) => (
        <Image
          width={50}
          height={30}
          src={record.previewUrl}
          preview={{
            src: record.previewUrl,
          }}
          style={{ objectFit: "cover", cursor: "pointer" }}
        />
      ),
    },
    { title: "文件名", dataIndex: "fileName", key: "fileName", width: 150, ellipsis: true },
    {
      title: "拼图形状",
      dataIndex: "pieceShape",
      key: "pieceShape",
      width: 120,
      render: (shape: PieceShape) => <Tag color="blue">{PIECE_SHAPE_MAP[shape]}</Tag>,
    },
    {
      title: "难度",
      dataIndex: "difficultyLevel",
      key: "difficultyLevel",
      width: 80,
      render: (level: DifficultyLevel) => <Tag color="orange">{DIFFICULTY_MAP[level]}</Tag>,
    },
    {
      title: "尺寸",
      key: "size",
      width: 100,
      render: (_, record) => `${record.fileWidth}x${record.fileHeight}`,
    },
    {
      title: "文件大小",
      dataIndex: "fileSize",
      key: "fileSize",
      width: 100,
      render: (size: number) => `${(size / 1024).toFixed(1)} KB`,
    },
    {
      title: "使用次数",
      dataIndex: "useCount",
      key: "useCount",
      width: 100,
      sorter: (a, b) => a.useCount - b.useCount,
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 80,
      render: (status: CaptchaBackgroundStatus) => (
        <Tag color={status === 1 ? "success" : "default"}>{status === 1 ? "启用" : "禁用"}</Tag>
      ),
    },
    { title: "备注", dataIndex: "remark", key: "remark", width: 150, ellipsis: true },
    {
      title: "操作",
      key: "action",
      fixed: "right",
      width: 200,
      render: (_, record) => (
        <Space size="small">
          <Button
            type="link"
            size="small"
            icon={<EditOutlined />}
            onClick={() => openEditModal(record)}
          >
            编辑
          </Button>
          <Button
            type="link"
            size="small"
            onClick={() => handleToggle(record.id)}
          >
            {record.status === 1 ? "禁用" : "启用"}
          </Button>
          <Button
            type="link"
            size="small"
            danger
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

  useEffect(() => {
    loadBackgrounds();
    loadStatistics();
  }, [paginationProps.current, paginationProps.pageSize]);

  return (
    <div style={{ padding: "0 0 24px" }}>
      {/* 统计卡片 */}
      {statistics && (
        <Row gutter={16} style={{ marginBottom: 16 }}>
          <Col span={6}>
            <Card>
              <Statistic title="总数量" value={statistics.totalCount} />
            </Card>
          </Col>
          <Col span={6}>
            <Card>
              <Statistic
                title="启用数量"
                value={statistics.enabledCount}
                styles={{ content: { color: "var(--theme-success, #3f8600)" } }}
              />
            </Card>
          </Col>
          <Col span={6}>
            <Card>
              <Statistic
                title="禁用数量"
                value={statistics.disabledCount}
                styles={{ content: { color: "var(--theme-error, #cf1322)" } }}
              />
            </Card>
          </Col>
          <Col span={6}>
            <Card>
              <Statistic title="总使用次数" value={(statistics as { totalCount: number; enabledCount: number; disabledCount: number; totalUsage?: number }).totalUsage ?? 0} />
            </Card>
          </Col>
        </Row>
      )}

      {/* 搜索表单 */}
      <Form form={searchForm} layout="inline" style={{ marginBottom: 16 }}>
        <Form.Item name="fileName" label="文件名">
          <Input placeholder="请输入文件名" allowClear className="user-form-input" style={{ width: 150 }} />
        </Form.Item>
        <Form.Item name="pieceShape" label="拼图形状">
          <Select placeholder="请选择" allowClear className="user-form-input" style={{ width: 120 }} onSearch={() => {}}>
            {SHAPE_OPTIONS.map(opt => (
              <Option key={opt.value} value={opt.value}>{opt.label}</Option>
            ))}
          </Select>
        </Form.Item>
        <Form.Item name="difficultyLevel" label="难度">
          <Select placeholder="请选择" allowClear className="user-form-input" style={{ width: 100 }} onSearch={() => {}}>
            <Option value={1}>简单</Option>
            <Option value={2}>中等</Option>
            <Option value={3}>困难</Option>
          </Select>
        </Form.Item>
        <Form.Item name="status" label="状态">
          <Select placeholder="请选择" allowClear className="user-form-input" style={{ width: 100 }} onSearch={() => {}}>
            <Option value={1}>启用</Option>
            <Option value={0}>禁用</Option>
          </Select>
        </Form.Item>
        <Form.Item>
          <Space>
            <Button type="primary" onClick={() => loadBackgrounds()}>
              查询
            </Button>
            <Button onClick={() => { searchForm.resetFields(); loadBackgrounds(); }}>
              重置
            </Button>
          </Space>
        </Form.Item>
      </Form>

      {/* 操作按钮 */}
      <div style={{ marginBottom: 16 }}>
        <Space>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setUploadModalVisible(true)}>
            上传背景图
          </Button>
          <Button icon={<PictureOutlined />} onClick={handlePreload}>
            预加载缓存
          </Button>
          <Button icon={<ReloadOutlined />} onClick={() => { loadBackgrounds(); loadStatistics(); }}>
            刷新
          </Button>
        </Space>
      </div>

      {/* 表格 */}
      <Table
        columns={columns}
        dataSource={backgrounds}
        loading={loading}
        rowKey="id"
        pagination={paginationProps}
        onChange={(pagination) => {
          setCurrent(pagination.current ?? 1);
          setPageSize(pagination.pageSize ?? 10);
          loadBackgrounds();
        }}
        scroll={{ x: 1200 }}
      />

      {/* 上传模态框 */}
      <Modal
        title="上传背景图"
        open={uploadModalVisible}
        onOk={handleUpload}
        onCancel={() => {
          setUploadModalVisible(false);
          uploadForm.resetFields();
          setFileList([]);
        }}
        confirmLoading={uploading}
        width={600}
      >
        <Form
          form={uploadForm}
          labelCol={{ span: 6 }}
          wrapperCol={{ span: 16 }}
          initialValues={{ pieceShape: "circle", difficultyLevel: 1 }}
        >
          <Form.Item label="图片文件" required>
            <Upload {...uploadProps}>
              {fileList.length === 0 && (
                <div>
                  <UploadOutlined />
                  <div style={{ marginTop: 8 }}>选择图片</div>
                </div>
              )}
            </Upload>
          </Form.Item>
          <Form.Item name="pieceShape" label="拼图形状" rules={[{ required: true, message: "请选择拼图形状" }]}>
            <Select onSearch={() => {}}>
              {SHAPE_OPTIONS.map(opt => (
                <Option key={opt.value} value={opt.value}>{opt.label}</Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="difficultyLevel" label="难度级别" rules={[{ required: true, message: "请选择难度" }]}>
            <Select onSearch={() => {}}>
              <Option value={1}>简单</Option>
              <Option value={2}>中等</Option>
              <Option value={3}>困难</Option>
            </Select>
          </Form.Item>
          <Form.Item name="allowedShapes" label="允许的形状">
            <Select mode="multiple" placeholder="不限制则默认使用当前形状" onSearch={() => {}}>
              {SHAPE_OPTIONS.map(opt => (
                <Option key={opt.value} value={opt.value}>{opt.label}</Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="remark" label="备注">
            <TextArea rows={3} placeholder="请输入备注" />
          </Form.Item>
        </Form>
      </Modal>

      {/* 编辑模态框 */}
      <Modal
        title="编辑背景图"
        open={editModalVisible}
        onOk={handleUpdate}
        onCancel={() => {
          setEditModalVisible(false);
          editForm.resetFields();
          setEditingBg(null);
        }}
        width={600}
      >
        <Form form={editForm} labelCol={{ span: 6 }} wrapperCol={{ span: 16 }}>
          <Form.Item name="pieceShape" label="拼图形状" rules={[{ required: true }]}>
            <Select onSearch={() => {}}>
              {SHAPE_OPTIONS.map(opt => (
                <Option key={opt.value} value={opt.value}>{opt.label}</Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="difficultyLevel" label="难度级别" rules={[{ required: true }]}>
            <Select onSearch={() => {}}>
              <Option value={1}>简单</Option>
              <Option value={2}>中等</Option>
              <Option value={3}>困难</Option>
            </Select>
          </Form.Item>
          <Form.Item name="allowedShapes" label="允许的形状">
            <Select mode="multiple" placeholder="不限制则使用默认" onSearch={() => {}}>
              {SHAPE_OPTIONS.map(opt => (
                <Option key={opt.value} value={opt.value}>{opt.label}</Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="status" label="状态" rules={[{ required: true }]}>
            <Select onSearch={() => {}}>
              <Option value={1}>启用</Option>
              <Option value={0}>禁用</Option>
            </Select>
          </Form.Item>
          <Form.Item name="sortOrder" label="排序" rules={[{ required: true }]}>
            <Input type="number" placeholder="数字越小优先级越高" />
          </Form.Item>
          <Form.Item name="remark" label="备注">
            <TextArea rows={3} placeholder="请输入备注" />
          </Form.Item>
        </Form>
      </Modal>

      {/* 图片预览 */}
      {previewImage && (
        <Image
          style={{ display: "none" }}
          src={previewImage}
          preview={{
            open: previewVisible,
            src: previewImage,
            onOpenChange: (value) => {
              setPreviewVisible(value);
            },
          }}
        />
      )}
    </div>
  );
};

export default CaptchaBackgroundSettingsPage;
