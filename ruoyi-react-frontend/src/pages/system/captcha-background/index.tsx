/**
 * Captcha Background Management Page
 * 验证码背景管理页面
 */

import { useEffect } from 'react';
import type { FC } from 'react';
import { Table, Button, Space, Form, Input, Select, Card, Row, Col, Statistic } from 'antd';
import {
  PlusOutlined,
  SearchOutlined,
  ReloadOutlined,
  UploadOutlined,
} from '@ant-design/icons';
import type { CaptchaBackground } from '@/types/captcha';
import { useCaptchaData } from './hooks';
import { getCaptchaColumns } from './columns';
import { CaptchaUploadModal, CaptchaEditModal } from './modals';
import { SHAPE_OPTIONS, DIFFICULTY_OPTIONS, STATUS_OPTIONS } from './constants';

const { Option } = Select;

const CaptchaBackgroundManagement: FC = () => {
  const [searchForm] = Form.useForm();
  const [uploadForm] = Form.useForm();
  const [editForm] = Form.useForm();

  const {
    backgrounds,
    loading,
    total,
    current,
    pageSize,
    statistics,
    setCurrent,
    setPageSize,
    loadBackgrounds,
    loadStatistics,
  } = useCaptchaData(searchForm);

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
    loadBackgrounds();
    loadStatistics();
  }, [current, pageSize, loadBackgrounds, loadStatistics]);

  // 操作成功后刷新
  const handleSuccess = () => {
    loadBackgrounds();
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
                styles={{ content: { color: '#3f8600' } }}
              />
            </Card>
          </Col>
          <Col span={6}>
            <Card>
              <Statistic
                title="禁用数量"
                value={statistics.disabledCount}
                styles={{ content: { color: '#cf1322' } }}
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
          <Input placeholder="请输入文件名" allowClear className="user-form-input" />
        </Form.Item>
        <Form.Item name="pieceShape" label="拼图形状">
          <Select placeholder="请选择" allowClear className="user-form-input" style={{ width: 120 }}>
            {SHAPE_OPTIONS.map((opt) => (
              <Option key={opt.value} value={opt.value}>
                {opt.label}
              </Option>
            ))}
          </Select>
        </Form.Item>
        <Form.Item name="difficultyLevel" label="难度">
          <Select placeholder="请选择" allowClear className="user-form-input" style={{ width: 100 }}>
            {DIFFICULTY_OPTIONS.map((opt) => (
              <Option key={opt.value} value={opt.value}>
                {opt.label}
              </Option>
            ))}
          </Select>
        </Form.Item>
        <Form.Item name="status" label="状态">
          <Select placeholder="请选择" allowClear className="user-form-input" style={{ width: 100 }}>
            {STATUS_OPTIONS.map((opt) => (
              <Option key={opt.value} value={opt.value}>
                {opt.label}
              </Option>
            ))}
          </Select>
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
        pagination={{
          current,
          pageSize,
          total,
          showSizeChanger: true,
          showQuickJumper: true,
          showTotal: (total) => `共 ${total} 条`,
          onChange: (page, size) => {
            setCurrent(page);
            setPageSize(size || 10);
          },
        }}
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
