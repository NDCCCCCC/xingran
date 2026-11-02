/**
 * Captcha Background Management Page
 * 验证码背景管理页面
 */

import { useEffect } from "react";
import type { FC } from "react";
import { Table, Button, Space, Form, Card, Row, Col, Statistic } from "antd";
// Phase 71 (W2)：裸 Input/Select → 契约组件。三处搜索下拉均为动态 children 固定枚举
// → 改写 options 形态；形状/难度/状态均为业务禁搜枚举 → 显式 showSearch={false}
import { AppInput, AppSelect } from "@/components/ui";
import { PlusOutlined, SearchOutlined, ReloadOutlined, UploadOutlined } from "@ant-design/icons";
import type { CaptchaBackground } from "@/types/captcha";
import { useCaptchaData, useCaptchaModals } from "./hooks";
import { getCaptchaColumns } from "./columns";
import { CaptchaUploadModal, CaptchaEditModal } from "./modals";
import { SHAPE_OPTIONS, DIFFICULTY_OPTIONS, STATUS_OPTIONS } from "./constants";
import { usePagination } from "@/hooks/usePagination";

const CaptchaBackgroundManagement: FC = () => {
  const [searchForm] = Form.useForm();
  const [uploadForm] = Form.useForm();
  const [editForm] = Form.useForm();

  // 使用全局分页 hook（current/pageSize 为原始值，effect 依赖不会因 total 变化误触发）
  const { paginationProps, current: pageNo, pageSize, setTotal } = usePagination();

  const {
    backgrounds,
    loading,
    total: _total,
    statistics,
    loadBackgrounds,
    loadStatistics,
  } = useCaptchaData(searchForm, setTotal);

  const {
    uploadModalVisible,
    editModalVisible,
    editingBg,
    fileList,
    uploading,
    uploadProps,
    setUploadModalVisible,
    setFileList,
    openEditModal,
    closeUploadModal,
    closeEditModal,
    handleUpload,
    handleUpdate,
    handleDelete,
    handleToggle,
    handlePreload,
  } = useCaptchaModals();

  useEffect(() => {
    loadBackgrounds({ current: pageNo, pageSize });
    loadStatistics();
  }, [pageNo, pageSize, loadBackgrounds, loadStatistics]);

  // 操作成功后刷新
  const handleSuccess = () => {
    loadBackgrounds({ current: pageNo, pageSize });
    loadStatistics();
  };

  const columns = getCaptchaColumns({
    handleEdit: (record: CaptchaBackground) => openEditModal(record, editForm),
    handleToggle: (id: string) => handleToggle(id, handleSuccess),
    handleDelete: (id: string) => handleDelete(id, handleSuccess),
  });

  return (
    <div>
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
                styles={{ content: { color: "var(--theme-success)" } }}
              />
            </Card>
          </Col>
          <Col span={6}>
            <Card>
              <Statistic
                title="禁用数量"
                value={statistics.disabledCount}
                styles={{ content: { color: "var(--theme-error)" } }}
              />
            </Card>
          </Col>
          <Col span={6}>
            <Card>
              <Statistic title="总使用次数" value={statistics.totalUsage} />
            </Card>
          </Col>
        </Row>
      )}

      {/* 搜索表单 */}
      <Form form={searchForm} layout="inline" style={{ marginBottom: 16 }}>
        <Form.Item name="fileName" label="文件名">
          <AppInput placeholder="请输入文件名" className="user-form-input" />
        </Form.Item>
        <Form.Item name="pieceShape" label="拼图形状">
          <AppSelect
            placeholder="请选择"
            allowClear
            className="user-form-input"
            style={{ width: 120 }}
            onSearch={() => {}}
            showSearch={false}
            options={[...SHAPE_OPTIONS]}
          />
        </Form.Item>
        <Form.Item name="difficultyLevel" label="难度">
          <AppSelect
            placeholder="请选择"
            allowClear
            className="user-form-input"
            style={{ width: 100 }}
            onSearch={() => {}}
            showSearch={false}
            options={[...DIFFICULTY_OPTIONS]}
          />
        </Form.Item>
        <Form.Item name="status" label="状态">
          <AppSelect
            placeholder="请选择"
            allowClear
            className="user-form-input"
            style={{ width: 100 }}
            onSearch={() => {}}
            showSearch={false}
            options={[...STATUS_OPTIONS]}
          />
        </Form.Item>
        <Form.Item>
          <Space>
            <Button type="primary" icon={<SearchOutlined />} onClick={() => loadBackgrounds()}>
              查询
            </Button>
            <Button
              icon={<ReloadOutlined />}
              onClick={() => {
                searchForm.resetFields();
                loadBackgrounds();
              }}
            >
              重置
            </Button>
          </Space>
        </Form.Item>
      </Form>

      {/* 操作按钮 */}
      <div style={{ marginBottom: 16 }}>
        <Space>
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => {
              uploadForm.resetFields();
              setFileList([]);
              setUploadModalVisible(true);
            }}
          >
            上传背景图
          </Button>
          <Button icon={<UploadOutlined />} onClick={handlePreload}>
            预加载缓存
          </Button>
          <Button
            icon={<ReloadOutlined />}
            onClick={() => {
              loadBackgrounds();
              loadStatistics();
            }}
          >
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
        scroll={{ x: "max-content" }}
      />

      {/* 上传模态框 */}
      <CaptchaUploadModal
        open={uploadModalVisible}
        uploading={uploading}
        uploadProps={uploadProps}
        onOk={async () => {
          await handleUpload(fileList, uploadForm, handleSuccess);
        }}
        onCancel={() => closeUploadModal(uploadForm)}
      />

      {/* 编辑模态框 */}
      <CaptchaEditModal
        open={editModalVisible}
        editingBg={editingBg}
        onOk={async () => {
          await handleUpdate(editingBg!, editForm, handleSuccess);
        }}
        onCancel={() => closeEditModal(editForm)}
      />
    </div>
  );
};

export default CaptchaBackgroundManagement;
