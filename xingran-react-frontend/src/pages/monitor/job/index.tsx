/**
 * 定时任务管理页面
 * Job Manager Page
 */

import { useState, useEffect, type FC } from "react";
import {
  Card,
  Table,
  Button,
  Space,
  Input,
  Select,
  Modal,
  Form,
  Switch,
  Row,
  Col,
  Statistic,
  Drawer,
} from "antd";
import {
  PlusOutlined,
  ReloadOutlined,
  SearchOutlined,
  ClockCircleOutlined,
} from "@ant-design/icons";

import type { JobInfo } from "./types";
import { STATUS_OPTIONS, MISFIRE_POLICY_OPTIONS, DEFAULT_FORM_VALUES, DEFAULT_SEARCH_FORM } from "./constants";
import { getJobColumns, getJobLogColumns } from "./columns";
import { useJobData, useJobActions } from "./hooks";
import CronSelector from "@/components/CronSelector";
import { usePagination } from "@/hooks/usePagination";

const { TextArea } = Input;

// ==================== 主组件 ====================

const JobManager: FC = () => {
  const [searchForm, setSearchForm] = useState(DEFAULT_SEARCH_FORM);
  const [form] = Form.useForm();

  // 使用全局分页 hook
  const { paginationProps, setCurrent, setPageSize } = usePagination();

  // 使用数据管理 Hook
  const {
    jobs,
    jobLogs,
    jobLogStats,
    loading,
    total,
    fetchJobs,
    fetchJobLogs,
  } = useJobData({
    searchForm,
    current: paginationProps.current ?? 1,
    pageSize: paginationProps.pageSize ?? 10,
  });

  // 使用操作管理 Hook
  const {
    modalVisible,
    modalTitle,
    isEdit,
    logDrawerVisible,
    selectedJob,
    setModalVisible,
    setIsEdit,
    setLogDrawerVisible,
    openModal,
    handleSubmit,
    handleDelete,
    handleToggleStatus,
    handleExecute,
    handleViewLogs,
    handleReset,
  } = useJobActions({
    fetchJobs,
    fetchJobLogs,
  });

  // 初始化加载
  useEffect(() => {
    fetchJobs();
  }, [paginationProps.current, paginationProps.pageSize, fetchJobs]);

  // 表格列
  const columns = getJobColumns({
    handleToggleStatus,
    handleExecute,
    handleViewLogs,
    openModal: (record) => openModal(record, form),
    handleDelete,
  });

  const jobLogColumns = getJobLogColumns();

  // 搜索
  const handleSearch = () => {
    setCurrent(1);
    fetchJobs();
  };

  // 刷新
  const handleRefresh = () => {
    fetchJobs();
  };

  // 分页变化
  const handleTableChange = (pagination: { current?: number; pageSize?: number }) => {
    setCurrent(pagination.current ?? 1);
    setPageSize(pagination.pageSize ?? 10);
    fetchJobs();
  };

  return (
    <div className="p-6">
      <div className="mb-6">
        <h1 className="text-2xl font-bold mb-4">定时任务管理</h1>

        {/* 搜索表单 */}
        <Card>
          <Row gutter={16}>
            <Col xs={24} sm={8} md={6}>
              <Input
                placeholder="任务名称"
                value={searchForm.jobName}
                onChange={(e) => setSearchForm({ ...searchForm, jobName: e.target.value })}
                allowClear
                className="user-form-input"
              />
            </Col>
            <Col xs={24} sm={8} md={6}>
              <Input
                placeholder="任务组"
                value={searchForm.jobGroup}
                onChange={(e) => setSearchForm({ ...searchForm, jobGroup: e.target.value })}
                allowClear
                className="user-form-input"
              />
            </Col>
            <Col xs={24} sm={8} md={6}>
              <Select
                placeholder="状态"
                value={searchForm.status}
                onChange={(value) =>    setSearchForm({ ...searchForm, status: value })}
                allowClear
                className="user-form-input"
                style={{ width: "100%" }}
                options={STATUS_OPTIONS}
               onSearch={() => {}}/>
            </Col>
            <Col xs={24} sm={24} md={6}>
              <Space>
                <Button type="primary" icon={<PlusOutlined />} onClick={() => openModal(undefined, form)}>
                  新增
                </Button>
                <Button icon={<SearchOutlined />} onClick={handleSearch}>
                  搜索
                </Button>
                <Button onClick={() => handleReset(setSearchForm, setCurrent, fetchJobs)}>重置</Button>
                <Button icon={<ReloadOutlined />} onClick={handleRefresh}>
                  刷新
                </Button>
              </Space>
            </Col>
          </Row>
        </Card>
      </div>

      {/* 任务列表 */}
      <Card>
        <Table
          columns={columns}
          dataSource={jobs}
          rowKey="id"
          loading={loading}
          pagination={paginationProps}
          onChange={(pagination) => {
            setCurrent(pagination.current ?? 1);
            setPageSize(pagination.pageSize ?? 10);
            fetchJobs();
          }}
          scroll={{ x: 1500 }}
        />
      </Card>

      {/* 新增/编辑模态框 */}
      <Modal
        title={modalTitle}
        open={modalVisible}
        onCancel={() => {
          setModalVisible(false);
          setIsEdit(false);
        }}
        onOk={() => handleSubmit(form)}
        width={700}
      >
        <Form
          form={form}
          layout="vertical"
          initialValues={DEFAULT_FORM_VALUES}
        >
          <Form.Item name="id" hidden>
            <Input />
          </Form.Item>
          <Form.Item
            label="任务名称"
            name="jobName"
            rules={[{ required: true, message: "请输入任务名称" }]}
          >
            <Input placeholder="请输入任务名称" />
          </Form.Item>
          <Form.Item
            label="任务组"
            name="jobGroup"
            rules={[{ required: true, message: "请输入任务组" }]}
          >
            <Input placeholder="请输入任务组，如：DEFAULT" />
          </Form.Item>
          <Form.Item
            label="调用目标"
            name="invokeTarget"
            rules={[{ required: true, message: "请输入调用目标字符串" }]}
            extra="任务执行时调用的目标方法或脚本"
          >
            <TextArea
              placeholder="请输入调用目标字符串"
              rows={3}
            />
          </Form.Item>
          <Form.Item
            label="Cron表达式"
            name="cronExpression"
            rules={[{ required: true, message: "请输入Cron表达式" }]}
          >
            <CronSelector />
          </Form.Item>
          <Form.Item
            label="执行策略"
            name="misfirePolicy"
            rules={[{ required: true }]}
            extra="任务错过执行时间的处理策略"
          >
            <Select options={MISFIRE_POLICY_OPTIONS}  onSearch={() => {}}/>
          </Form.Item>
          <Form.Item
            label="并发执行"
            name="concurrent"
            valuePropName="checked"
            extra="是否允许同一任务同时执行多个实例"
          >
            <Switch checkedChildren="允许" unCheckedChildren="禁止" />
          </Form.Item>
          <Form.Item
            label="状态"
            name="status"
            rules={[{ required: true }]}
          >
            <Select options={STATUS_OPTIONS}  onSearch={() => {}}/>
          </Form.Item>
          <Form.Item
            label="备注"
            name="remark"
          >
            <TextArea placeholder="请输入备注" rows={3} />
          </Form.Item>
        </Form>
      </Modal>

      {/* 任务日志抽屉 */}
      <Drawer
        title={`任务执行日志 - ${selectedJob?.jobName}`}
        placement="right"
        onClose={() => setLogDrawerVisible(false)}
        open={logDrawerVisible}
        size={800}
        extra={
          <Space>
            <Button icon={<ReloadOutlined />} onClick={() => fetchJobLogs(selectedJob?.jobName, selectedJob?.jobGroup)}>
              刷新
            </Button>
          </Space>
        }
      >
        <div className="mb-4">
          <Row gutter={16}>
            <Col span={8}>
              <Card size="small">
                <Statistic
                  title="总执行次数"
                  value={jobLogStats.total}
                  prefix={<ClockCircleOutlined />}
                />
              </Card>
            </Col>
            <Col span={8}>
              <Card size="small">
                <Statistic
                  title="成功次数"
                  value={jobLogStats.success}
                  styles={{ content: { color: "var(--theme-success, #3f8600)" } }}
                />
              </Card>
            </Col>
            <Col span={8}>
              <Card size="small">
                <Statistic
                  title="失败次数"
                  value={jobLogStats.fail}
                  styles={{ content: { color: "var(--theme-error, #cf1322)" } }}
                />
              </Card>
            </Col>
          </Row>
        </div>

        <Table
          columns={jobLogColumns}
          dataSource={jobLogs}
          rowKey="id"
          size="small"
          pagination={{
            pageSize: 50,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => `共 ${total} 条记录`,
          }}
        />
      </Drawer>
    </div>
  );
};

export default JobManager;
