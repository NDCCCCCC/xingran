import { useState, useEffect } from 'react';
import type { FC } from 'react';
import {
  Table,
  Button,
  Space,
  Form,
  Select,
  Card,
  Row,
  Col,
  Statistic,
} from 'antd';
import {
  PlayCircleOutlined,
  SearchOutlined,
  ReloadOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  ClockCircleOutlined,
  ApiOutlined,
} from '@ant-design/icons';
import { useExecutionData, useExecutionModals } from './hooks';
import { getExecutionColumns, getDetailColumns } from './columns';
import { ConfigExecuteModal, VariableModal, DetailDrawer } from './modals';
import { STATUS_OPTIONS } from './constants';
import { usePagination } from '@/hooks/usePagination';
import type { ConfigExecution } from '@/types';

const { Option } = Select;

const ConfigExecutionPage: FC = () => {
  const [executeForm] = Form.useForm();
  const [searchForm] = Form.useForm();

  // 使用全局分页 hook
  const { paginationProps, setCurrent, setPageSize, setTotal } = usePagination();

  // 数据管理
  const {
    dataState,
    execLoading,
    execTotal,
    statistics,
    loadDevices,
    loadTemplates,
    loadExecutions,
    loadExecutionDetails,
    loadStatistics,
  } = useExecutionData({
    current: 1, // 未使用，保留默认值
    pageSize: paginationProps.pageSize ?? 10,
    execCurrent: paginationProps.current ?? 1,
  });

  // 模态框和操作管理
  const {
    modalState,
    selectedRowKeys,
    setSelectedRowKeys,
    openExecuteModal: openExecuteModalBase,
    handleTemplateChange,
    handleExecuteByTemplate,
    handleCancelExecution,
    handleViewDetail: handleViewDetailBase,
    handleViewOutput,
    closeDetailDrawer,
    closeVariableModal,
    closeExecuteModal,
  } = useExecutionModals({
    dataState,
    setDataState: () => {}, // Not needed here
    loadExecutions,
  });

  // 打开执行模态框时加载设备和模板
  const openExecuteModal = async () => {
    await loadDevices();
    await loadTemplates();
    openExecuteModalBase();
  };

  // 查看详情时加载执行详情
  const handleViewDetail = async (record: ConfigExecution) => {
    await loadExecutionDetails(record.id);
    handleViewDetailBase(record);
  };

  // 表格列
  const executionColumns = getExecutionColumns({
    handleViewDetail,
    handleCancelExecution,
  });

  const detailColumns = getDetailColumns({
    handleViewOutput,
  });

  useEffect(() => {
    Promise.all([loadExecutions(), loadStatistics()]);
  }, [paginationProps.current, paginationProps.pageSize, loadExecutions, loadStatistics]);

  return (
    <div>
      {/* 统计卡片 */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={5}>
          <Card>
            <Statistic
              title="执行总数"
              value={statistics.total}
              prefix={<ApiOutlined />}
            />
          </Card>
        </Col>
        <Col span={5}>
          <Card>
            <Statistic
              title="待执行"
              value={statistics.pending}
              styles={{ content: { color: '#999' } }}
              prefix={<ClockCircleOutlined />}
            />
          </Card>
        </Col>
        <Col span={4}>
          <Card>
            <Statistic
              title="执行中"
              value={statistics.running}
              styles={{ content: { color: '#1890ff' } }}
              prefix={<ApiOutlined />}
            />
          </Card>
        </Col>
        <Col span={5}>
          <Card>
            <Statistic
              title="成功"
              value={statistics.success}
              styles={{ content: { color: '#52c41a' } }}
              prefix={<CheckCircleOutlined />}
            />
          </Card>
        </Col>
        <Col span={5}>
          <Card>
            <Statistic
              title="失败"
              value={statistics.failed}
              styles={{ content: { color: '#ff4d4f' } }}
              prefix={<CloseCircleOutlined />}
            />
          </Card>
        </Col>
      </Row>

      {/* 搜索表单和操作按钮 */}
      <Card style={{ marginBottom: 16 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', flexWrap: 'wrap', gap: '16px' }}>
          <Form form={searchForm} layout="inline" style={{ flex: 1, minWidth: 0 }}>
            <Form.Item label="状态">
              <Select
                placeholder="请选择状态"
                allowClear
                className="user-form-input"
                style={{ width: 120 }}
                onChange={() => {
                  loadExecutions();
                  loadStatistics();
                }}
              >
                {STATUS_OPTIONS.map((opt) => (
                  <Option key={opt.value} value={opt.value}>
                    {opt.label}
                  </Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item>
              <Space>
                <Button
                  icon={<SearchOutlined />}
                  onClick={() => {
                    loadExecutions();
                    loadStatistics();
                  }}
                >
                  查询
                </Button>
                <Button
                  icon={<ReloadOutlined />}
                  onClick={() => {
                    loadExecutions();
                    loadStatistics();
                  }}
                >
                  刷新
                </Button>
              </Space>
            </Form.Item>
          </Form>
          <Space>
            <Button type="primary" icon={<PlayCircleOutlined />} onClick={openExecuteModal}>
              执行配置模板
            </Button>
          </Space>
        </div>
      </Card>

      {/* 执行记录表格 */}
      <Card>
        <Table
          columns={executionColumns}
          dataSource={dataState.executions}
          loading={execLoading}
          rowKey="id"
          scroll={{ x: 1400 }}
          pagination={paginationProps}
          onChange={(pagination) => {
            setCurrent(pagination.current ?? 1);
            setPageSize(pagination.pageSize ?? 10);
            const formValues = searchForm.getFieldsValue() as Record<string, unknown>;
            const searchParams: Record<string, unknown> = { current: pagination.current ?? 1, pageSize: pagination.pageSize ?? 10 };
            Object.keys(formValues).forEach(key => {
              const value = formValues[key];
              if (value !== undefined && value !== null && value !== '') {
                searchParams[key] = value;
              }
            });
            loadExecutions(searchParams);
          }}
        />
      </Card>

      {/* 执行配置模态框 */}
      <ConfigExecuteModal
        open={modalState.executeModalVisible}
        devices={dataState.devices}
        templates={dataState.templates}
        selectedTemplate={dataState.selectedTemplate}
        selectedRowKeys={selectedRowKeys}
        onOk={async () => {
          await handleExecuteByTemplate(executeForm);
        }}
        onCancel={() => {
          closeExecuteModal();
          executeForm.resetFields();
        }}
        onTemplateChange={handleTemplateChange}
        onSelectedRowKeysChange={setSelectedRowKeys}
      />

      {/* 变量输入模态框 */}
      <VariableModal
        open={modalState.variableModalVisible}
        selectedTemplate={dataState.selectedTemplate}
        form={executeForm}
        onOk={closeVariableModal}
        onCancel={closeVariableModal}
      />

      {/* 执行明细抽屉 */}
      <DetailDrawer
        open={modalState.detailDrawerVisible}
        currentExecution={dataState.currentExecution}
        executionDetails={dataState.executionDetails}
        onClose={closeDetailDrawer}
        handleViewOutput={handleViewOutput}
      />
    </div>
  );
};

export default ConfigExecutionPage;
