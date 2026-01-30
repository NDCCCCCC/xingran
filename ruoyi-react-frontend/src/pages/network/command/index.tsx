/**
 * Network Command Dispatch Page
 * 网络命令分发页面
 */

import { useEffect, useState } from 'react';
import type { FC } from 'react';
import {
  Table,
  Button,
  Space,
  Form,
  Input,
  Select,
  Card,
  Row,
  Col,
  Statistic,
} from 'antd';
import {
  ThunderboltOutlined,
  SearchOutlined,
  ReloadOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  ClockCircleOutlined,
  ApiOutlined,
} from '@ant-design/icons';
import type { ConfigExecution, NetworkDevice } from '@/types';
import { useCommandData, useCommandModals } from './hooks';
import { getExecutionColumns, deviceColumns } from './columns';
import { CommandDispatchModal, CommandDetailDrawer } from './modals';
import { STATUS_OPTIONS } from './constants';
import { usePagination } from '@/hooks/usePagination';

const { Option } = Select;

const CommandDispatch: FC = () => {
  const [execLoading, setExecLoading] = useState(false);
  const [dispatchModalVisible, setDispatchModalVisible] = useState(false);
  const [detailDrawerVisible, setDetailDrawerVisible] = useState(false);
  const [dispatchForm] = Form.useForm();
  const [selectedRowKeys, setSelectedRowKeys] = useState<string[]>([]);
  const [devices, setDevices] = useState<NetworkDevice[]>([]);
  const [currentExecution, setCurrentExecution] = useState<ConfigExecution | null>(null);
  const [executionDetails, setExecutionDetails] = useState<any[]>([]);

  // 使用全局分页 hook
  const { paginationProps, setCurrent, setPageSize, setTotal } = usePagination();

  const {
    executions,
    execTotal,
    statistics,
    setExecTotal,
    loadExecutions,
    loadStatistics,
    loadDevices,
    loadExecutionDetails,
  } = useCommandData(setExecLoading, {
    current: paginationProps.current,
    pageSize: paginationProps.pageSize,
  });

  // 同步 total 到全局分页状态
  useEffect(() => {
    setTotal(execTotal);
  }, [execTotal, setTotal]);

  const { handleQuickCommand, handleCancelExecution } = useCommandModals();

  useEffect(() => {
    loadExecutions();
    loadStatistics();
  }, [paginationProps.current, paginationProps.pageSize, loadExecutions, loadStatistics]);

  // 操作成功后刷新
  const handleSuccess = () => {
    loadExecutions();
    loadStatistics();
  };

  // 查看执行详情
  const handleViewDetail = async (record: ConfigExecution) => {
    const { execution, details } = await loadExecutionDetails(record.id);
    setCurrentExecution(execution);
    setExecutionDetails(details);
    setDetailDrawerVisible(true);
  };

  // 打开命令分发模态框
  const openDispatchModal = async () => {
    const loadedDevices = await loadDevices();
    setDevices(loadedDevices);
    setSelectedRowKeys([]);
    dispatchForm.resetFields();
    setDispatchModalVisible(true);
  };

  const columns = getExecutionColumns({
    handleViewDetail,
    handleCancel: (id: string) => handleCancelExecution(id, handleSuccess),
  });

  return (
    <div>
      {/* 统计卡片 */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={5}>
          <Card>
            <Statistic title="执行总数" value={statistics.total} prefix={<ApiOutlined />} />
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
        <Col span={5}>
          <Card>
            <Statistic
              title="执行中"
              value={statistics.running}
              styles={{ content: { color: '#1890ff' } }}
              prefix={<ApiOutlined />}
            />
          </Card>
        </Col>
        <Col span={4}>
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

      {/* 主要内容 */}
      <Card>
        <div style={{ marginBottom: 16 }}>
          <Form layout="inline">
            <Form.Item label="状态">
              <Select
                placeholder="请选择状态"
                allowClear
                className="user-form-input"
                style={{ width: 120 }}
                onChange={() => {
                  handleSuccess();
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
                    handleSuccess();
                  }}
                >
                  查询
                </Button>
                <Button
                  icon={<ReloadOutlined />}
                  onClick={() => {
                    handleSuccess();
                  }}
                >
                  刷新
                </Button>
                <Button
                  type="primary"
                  icon={<ThunderboltOutlined />}
                  onClick={openDispatchModal}
                >
                  快速命令
                </Button>
              </Space>
            </Form.Item>
          </Form>
        </div>

        <Table
          columns={columns}
          dataSource={executions}
          loading={execLoading}
          rowKey="id"
          scroll={{ x: 1400 }}
          pagination={paginationProps}
          onChange={(pagination) => {
            setCurrent(pagination.current);
            setPageSize(pagination.pageSize);
            loadExecutions({
              current: pagination.current,
              pageSize: pagination.pageSize,
            });
          }}
        />
      </Card>

      {/* 命令分发模态框 */}
      <CommandDispatchModal
        open={dispatchModalVisible}
        devices={devices}
        selectedRowKeys={selectedRowKeys}
        onOk={async () => {
          await handleQuickCommand(selectedRowKeys, dispatchForm, () => {
            setDispatchModalVisible(false);
            handleSuccess();
          });
        }}
        onCancel={() => setDispatchModalVisible(false)}
        onSelectionChange={(keys) => setSelectedRowKeys(keys as string[])}
      />

      {/* 执行明细抽屉 */}
      <CommandDetailDrawer
        open={detailDrawerVisible}
        execution={currentExecution}
        details={executionDetails}
        onClose={() => setDetailDrawerVisible(false)}
      />
    </div>
  );
};

export default CommandDispatch;
