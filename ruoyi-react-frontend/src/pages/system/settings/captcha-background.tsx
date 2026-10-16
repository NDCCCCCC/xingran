import { useState, useEffect } from "react";
import type { FC, CSSProperties } from "react";
import {
  App,
  Button,
  Space,
  Modal,
  Form,
  Input,
  Select,
  Upload,
  Image,
  Card,
  Pagination,
  Spin,
} from "antd";
import type { UploadFile, UploadProps } from "antd/es/upload";
import { UploadOutlined, PictureOutlined } from "@ant-design/icons";
import type {
  CaptchaBackground,
  CaptchaBackgroundUpdateRequest,
  PieceShape,
  DifficultyLevel,
  CaptchaBackgroundStatus,
  StatisticsResponse,
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

// ★ status 语义反转（后端契约例外，70-RESEARCH Pitfall 1）：
//   captcha 背景图 1 = 启用 / 0 = 禁用 —— 与全局「0=启用」惯例相反，显示层按此取数，勿「纠正」。
//   统计卡「启用数量」与卡脚「启用」徽标均按 status === 1 取值；启停按钮文案 = 当前 status 取反动作。

// 空状态卡（跨整行白卡：icon + 标题 + 正文 + CTA）
const EMPTY_CARD_STYLE: CSSProperties = {
  gridColumn: "1 / -1",
  display: "flex",
  flexDirection: "column",
  alignItems: "center",
  justifyContent: "center",
  gap: 4,
  padding: "32px 24px",
  background: "var(--theme-bg-surface)",
  border: "1px solid var(--theme-border-primary)",
  borderRadius: "var(--theme-radius-xl)",
};

const CaptchaBackgroundSettingsPage: FC = () => {
  const { message } = App.useApp();
  const [backgrounds, setBackgrounds] = useState<CaptchaBackground[]>([]);
  const [loading, setLoading] = useState(false);

  // 紧凑筛选（工具栏内即时生效：Select 即选即筛，文件名回车/清空生效）
  const [fileNameInput, setFileNameInput] = useState("");
  const [fileNameFilter, setFileNameFilter] = useState("");
  const [shapeFilter, setShapeFilter] = useState<PieceShape | undefined>(undefined);
  const [difficultyFilter, setDifficultyFilter] = useState<DifficultyLevel | undefined>(undefined);
  const [statusFilter, setStatusFilter] = useState<CaptchaBackgroundStatus | undefined>(undefined);

  // 统计（getCaptchaBackgroundStatistics 现成端点：4 卡数据源）
  const [statistics, setStatistics] = useState<StatisticsResponse | null>(null);

  const [uploadModalVisible, setUploadModalVisible] = useState(false);
  const [editModalVisible, setEditModalVisible] = useState(false);
  const [editForm] = Form.useForm();
  const [uploadForm] = Form.useForm();
  const [editingBg, setEditingBg] = useState<CaptchaBackground | null>(null);
  const [fileList, setFileList] = useState<UploadFile[]>([]);
  const [uploading, setUploading] = useState(false);

  // 使用全局分页 hook（list API 契约不变）
  const { paginationProps, setCurrent, setTotal } = usePagination();

  // 加载背景图列表
  const loadBackgrounds = async () => {
    setLoading(true);
    try {
      const result = await captchaService.getCaptchaBackgroundList({
        current: paginationProps.current,
        pageSize: paginationProps.pageSize,
        fileName: fileNameFilter || undefined,
        pieceShape: shapeFilter,
        difficultyLevel: difficultyFilter,
        status: statusFilter,
      });
      setBackgrounds(result.items);
      setTotal(result.total);
    } catch (error) {
      console.error("加载背景图列表失败:", error);
      message.error("加载背景图列表失败，请刷新重试");
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

  // 删除背景图（具名确认弹窗，danger 按钮）
  const handleDelete = async (id: string) => {
    try {
      await captchaService.deleteCaptchaBackground(id);
      message.success("删除成功");
      loadBackgrounds();
      loadStatistics();
    } catch (_error) {
      message.error("删除失败");
    }
  };

  // 切换状态（非破坏性操作，不弹确认；文案按 status 取反）
  const handleToggle = async (id: string) => {
    try {
      await captchaService.toggleCaptchaBackgroundStatus(id);
      message.success("状态更新成功");
      loadBackgrounds();
      loadStatistics();
    } catch (_error) {
      message.error("状态更新失败");
    }
  };

  // 预加载缓存
  const handlePreload = async () => {
    try {
      await captchaService.preloadCaptchaCache();
      message.success("预加载成功");
    } catch (_error) {
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

  // 上传配置（D-09 约束逐字保留：仅图片 + <2MB + maxCount 1 + 手动上传）
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

  useEffect(() => {
    loadBackgrounds();
    loadStatistics();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [
    paginationProps.current,
    paginationProps.pageSize,
    fileNameFilter,
    shapeFilter,
    difficultyFilter,
    statusFilter,
  ]);

  // 启用占比（反转语义：enabledCount 即 status=1 计数）
  const enabledPct =
    statistics && statistics.totalCount > 0
      ? Math.round((statistics.enabledCount / statistics.totalCount) * 100)
      : 0;

  return (
    <>
      {/* 统计卡行：4 卡 — 总数量（默认绿条）/ 启用（sc-green）/ 禁用（sc-gray）/ 总使用次数（sc-gold） */}
      <div className="stat-cards">
        <div className="stat-card">
          <div className="stat-label">总数量</div>
          <div className="stat-value">{statistics?.totalCount ?? 0}</div>
          <div className="stat-trend">全部背景图资产</div>
        </div>
        <div className="stat-card sc-green">
          <div className="stat-label">启用数量</div>
          <div className="stat-value" style={{ color: "var(--theme-success)" }}>
            {statistics?.enabledCount ?? 0}
          </div>
          <div className="stat-trend">占比 {enabledPct}%</div>
        </div>
        <div className="stat-card sc-gray">
          <div className="stat-label">禁用数量</div>
          <div className="stat-value" style={{ color: "var(--theme-error)" }}>
            {statistics?.disabledCount ?? 0}
          </div>
          <div className="stat-trend">不参与抽取</div>
        </div>
        <div className="stat-card sc-gold">
          <div className="stat-label">总使用次数</div>
          <div className="stat-value">{statistics?.totalUsage ?? 0}</div>
          <div className="stat-trend">累计验证码展示</div>
        </div>
      </div>

      {/* 工具栏卡：紧凑筛选（无独立搜索区）+ 右侧按钮组 */}
      <Card style={{ marginBottom: 14 }}>
        <div
          style={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "flex-start",
            flexWrap: "wrap",
            gap: "16px",
          }}
        >
          <Space wrap size={8}>
            <Input
              placeholder="搜索文件名"
              allowClear
              value={fileNameInput}
              onChange={(e) => {
                setFileNameInput(e.target.value);
                if (e.target.value === "") {
                  setFileNameFilter("");
                  setCurrent(1);
                }
              }}
              onPressEnter={() => {
                setFileNameFilter(fileNameInput);
                setCurrent(1);
              }}
              style={{ width: 180 }}
            />
            <Select<PieceShape | undefined>
              placeholder="拼图形状"
              allowClear
              value={shapeFilter}
              onChange={(value) => {
                setShapeFilter(value);
                setCurrent(1);
              }}
              options={SHAPE_OPTIONS}
              onSearch={() => {}}
              style={{ width: 110 }}
            />
            <Select<DifficultyLevel | undefined>
              placeholder="难度"
              allowClear
              value={difficultyFilter}
              onChange={(value) => {
                setDifficultyFilter(value);
                setCurrent(1);
              }}
              options={[
                { value: 1, label: "简单" },
                { value: 2, label: "中等" },
                { value: 3, label: "困难" },
              ]}
              onSearch={() => {}}
              style={{ width: 100 }}
            />
            <Select<CaptchaBackgroundStatus | undefined>
              placeholder="状态"
              allowClear
              value={statusFilter}
              onChange={(value) => {
                setStatusFilter(value);
                setCurrent(1);
              }}
              options={[
                { value: 1, label: "启用" },
                { value: 0, label: "禁用" },
              ]}
              onSearch={() => {}}
              style={{ width: 100 }}
            />
          </Space>
          <Space>
            <Button icon={<PictureOutlined />} onClick={handlePreload}>
              预加载缓存
            </Button>
            <Button
              type="primary"
              icon={<UploadOutlined />}
              onClick={() => setUploadModalVisible(true)}
            >
              上传背景图
            </Button>
          </Space>
        </div>
      </Card>

      {/* 网格墙（D-08）：缩略图卡片网格，图片即视觉主体 */}
      <Spin spinning={loading}>
        <div className="xr-captcha-grid">
          {!loading && backgrounds.length === 0 ? (
            <div style={EMPTY_CARD_STYLE}>
              <PictureOutlined style={{ fontSize: 48, color: "var(--theme-text-secondary)" }} />
              <div style={{ fontSize: 16, fontWeight: 600, color: "var(--theme-text-primary)" }}>
                暂无背景图
              </div>
              <div style={{ fontSize: 12, color: "var(--theme-text-secondary)" }}>
                上传第一张验证码背景图，用于登录页拼图验证码
              </div>
              <Button
                type="primary"
                icon={<UploadOutlined />}
                onClick={() => setUploadModalVisible(true)}
                style={{ marginTop: 8 }}
              >
                上传背景图
              </Button>
            </div>
          ) : (
            backgrounds.map((record) => (
              <div className="xr-captcha-card" key={record.id}>
                {/* 图片区 4:3 + Antd Image 内置预览（不自建灯箱） */}
                <Image
                  className="xr-captcha-card-image"
                  src={record.previewUrl}
                  preview={{ src: record.previewUrl }}
                  style={{ objectFit: "cover" }}
                />
                {/* 卡脚行 1：文件名（mono ellipsis）+ 状态徽标（1=启用 反转语义）。
                    形状/难度/尺寸/大小/使用次数信息不丢 — 以 title 提示承载（planner 裁量，
                    铜金 xr-tag-gold 每屏 ≤2 处预算不允许逐卡渲染难度 Tag） */}
                <div
                  className="xr-captcha-card-foot"
                  title={`${record.fileName} · 形状 ${PIECE_SHAPE_MAP[record.pieceShape]} · 难度 ${DIFFICULTY_MAP[record.difficultyLevel]} · ${record.fileWidth}x${record.fileHeight} · ${(record.fileSize / 1024).toFixed(1)} KB · 使用 ${record.useCount} 次${record.remark ? ` · ${record.remark}` : ""}`}
                >
                  <span className="xr-captcha-card-name">{record.fileName}</span>
                  <span className={`xr-tag ${record.status === 1 ? "xr-tag-green" : ""}`}>
                    {record.status === 1 ? "启用" : "禁用"}
                  </span>
                </div>
                {/* 卡脚行 2：文字链接操作（启停文案 = 当前 status 取反） */}
                <div className="xr-row-ops xr-captcha-card-ops" style={{ padding: "0 8px 8px" }}>
                  <Button type="link" size="small" onClick={() => handleToggle(record.id)}>
                    {record.status === 1 ? "禁用" : "启用"}
                  </Button>
                  <Button type="link" size="small" onClick={() => openEditModal(record)}>
                    编辑
                  </Button>
                  <Button
                    type="link"
                    size="small"
                    className="xr-op-danger"
                    onClick={() => {
                      Modal.confirm({
                        title: "删除背景图？",
                        content: "删除后不可恢复，启用中的拼图验证码将不再使用该背景。",
                        okText: "删除",
                        cancelText: "取消",
                        okButtonProps: { danger: true },
                        onOk: () => handleDelete(record.id),
                      });
                    }}
                  >
                    删除
                  </Button>
                </div>
              </div>
            ))
          )}
        </div>
      </Spin>

      {/* 分页保留（usePagination 沿用，list API 契约不变） */}
      <Pagination {...paginationProps} style={{ marginTop: 16, justifyContent: "flex-end" }} />

      {/* 上传模态框 — D-09 原样保留 */}
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
          <Form.Item
            name="pieceShape"
            label="拼图形状"
            rules={[{ required: true, message: "请选择拼图形状" }]}
          >
            <Select onSearch={() => {}}>
              {SHAPE_OPTIONS.map((opt) => (
                <Option key={opt.value} value={opt.value}>
                  {opt.label}
                </Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item
            name="difficultyLevel"
            label="难度级别"
            rules={[{ required: true, message: "请选择难度" }]}
          >
            <Select onSearch={() => {}}>
              <Option value={1}>简单</Option>
              <Option value={2}>中等</Option>
              <Option value={3}>困难</Option>
            </Select>
          </Form.Item>
          <Form.Item name="allowedShapes" label="允许的形状">
            <Select mode="multiple" placeholder="不限制则默认使用当前形状" onSearch={() => {}}>
              {SHAPE_OPTIONS.map((opt) => (
                <Option key={opt.value} value={opt.value}>
                  {opt.label}
                </Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="remark" label="备注">
            <TextArea rows={3} placeholder="请输入备注" />
          </Form.Item>
        </Form>
      </Modal>

      {/* 编辑模态框 — D-09 原样保留 */}
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
              {SHAPE_OPTIONS.map((opt) => (
                <Option key={opt.value} value={opt.value}>
                  {opt.label}
                </Option>
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
              {SHAPE_OPTIONS.map((opt) => (
                <Option key={opt.value} value={opt.value}>
                  {opt.label}
                </Option>
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
    </>
  );
};

export default CaptchaBackgroundSettingsPage;
