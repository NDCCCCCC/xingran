/**
 * 周期性工单模板管理页面
 * Periodic Work Order Template Management Page
 */

import { useState, useEffect, type FC } from "react";
import {
  Button,
  Form,
  Input,
  Select,
  Table,
  Modal,
  Space,
  Tag,
  Card,
  Row,
  Col,
  Statistic,
  Drawer,
  Timeline,
  Switch,
  TreeSelect,
} from "antd";
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  SearchOutlined,
  ReloadOutlined,
  PlayCircleOutlined,
  PauseCircleOutlined,
  HistoryOutlined,
  ThunderboltOutlined,
} from "@ant-design/icons";
import type { ColumnsType, TablePaginationConfig } from "antd/es/table";

import type { PeriodicWorkOrderTemplate } from "@/lib/workorderApi";
import ActionButtons from "@/components/shared/ActionButtons";
import CronSelector from "@/components/CronSelector";
import { PeriodicAssignType } from "@/lib/workorderApi";

// 导入提取的模块
import { PRIORITY_CONFIG, TYPE_CONFIG, ASSIGN_TYPE_CONFIG } from "./constants";
import { formatDateTime, buildCategoryTree } from "@/lib/workorderApi";
import { useTemplateData, useTemplateActions } from "./hooks";
import { VARIABLE_HELP_CONTENT } from "./utils";
import { usePagination } from "@/hooks/usePagination";

const { Option } = Select;
const { TextArea } = Input;

// ==================== 表格列定义 ====================

interface TemplateTableColumnsProps {
  current: number;
  pageSize: number;
  handleEdit: (record: PeriodicWorkOrderTemplate) => void;
  handleToggleEnabled: (record: PeriodicWorkOrderTemplate) => void;
  handleGenerateNow: (id: string) => void;
  handleViewLogs: (record: PeriodicWorkOrderTemplate) => void;
  handleDelete: (id: string) => void;
}

function getTemplateTableColumns(props: TemplateTableColumnsProps): ColumnsType<PeriodicWorkOrderTemplate> {
  const { current, pageSize, handleEdit, handleToggleEnabled, handleGenerateNow, handleViewLogs, handleDelete } = props;

  return [
    {
      title: "序号",
      key: "index",
      width: 60,
      render: (_: unknown, __: unknown, index: number) => (current - 1) * pageSize + index + 1,
    },
    {
      title: "模板名称",
      dataIndex: "templateName",
      key: "templateName",
      width: 150,
    },
    {
      title: "工单标题",
      dataIndex: "workOrderTitle",
      key: "workOrderTitle",
      width: 200,
      ellipsis: true,
    },
    {
      title: "类型",
      dataIndex: "type",
      key: "type",
      width: 80,
      render: (type: string) => TYPE_CONFIG[type as keyof typeof TYPE_CONFIG]?.text || type,
    },
    {
      title: "优先级",
      dataIndex: "priority",
      key: "priority",
      width: 80,
      render: (priority: number) => (
        <Tag color={PRIORITY_CONFIG[priority as keyof typeof PRIORITY_CONFIG]?.color}>{PRIORITY_CONFIG[priority as keyof typeof PRIORITY_CONFIG]?.text}</Tag>
      ),
    },
    {
      title: "Cron表达式",
      dataIndex: "cronExpression",
      key: "cronExpression",
      width: 120,
    },
    {
      title: "分配类型",
      dataIndex: "assignType",
      key: "assignType",
      width: 120,
      render: (type: string) => ASSIGN_TYPE_CONFIG[type as keyof typeof ASSIGN_TYPE_CONFIG]?.text || type,
    },
    {
      title: "下次执行",
      dataIndex: "nextRunAt",
      key: "nextRunAt",
      width: 180,
      render: (date: string) => formatDateTime(date),
    },
    {
      title: "已生成",
      dataIndex: "totalGenerated",
      key: "totalGenerated",
      width: 80,
    },
    {
      title: "状态",
      dataIndex: "isEnabled",
      key: "isEnabled",
      width: 80,
      render: (isEnabled: boolean) => (
        <Tag color={isEnabled ? "green" : "red"}>{isEnabled ? "已启用" : "已禁用"}</Tag>
      ),
    },
    {
      title: "操作",
      key: "action",
      width: 100,
      fixed: "right",
      render: (_: unknown, record: PeriodicWorkOrderTemplate) => {
        const actions = [
          {
            key: "edit",
            label: "编辑",
            icon: <EditOutlined />,
            onClick: () => handleEdit(record),
          },
          {
            key: "toggle",
            label: record.isEnabled ? "禁用" : "启用",
            icon: record.isEnabled ? <PauseCircleOutlined /> : <PlayCircleOutlined />,
            onClick: () => handleToggleEnabled(record),
          },
          {
            key: "generate",
            label: "立即生成",
            icon: <ThunderboltOutlined />,
            onClick: () => handleGenerateNow(record.id),
          },
          {
            key: "logs",
            label: "执行记录",
            icon: <HistoryOutlined />,
            onClick: () => handleViewLogs(record),
          },
          {
            key: "delete",
            label: "删除",
            icon: <DeleteOutlined />,
            danger: true,
            onClick: () => {
              Modal.confirm({
                title: "确定要删除吗？",
                okText: "确定",
                cancelText: "取消",
                okButtonProps: { danger: true },
                onOk: () => handleDelete(record.id),
              });
            },
          },
        ];

        return <ActionButtons actions={actions} />;
      },
    },
  ];
}

// ==================== 变量帮助组件 ====================

interface VariableHelpProps {
  content: typeof VARIABLE_HELP_CONTENT;
}

function VariableHelp({ content }: VariableHelpProps) {
  return (
    <div className="text-sm text-gray-500">
      <p className="font-semibold mb-2">{content.title}</p>
      <div className="grid grid-cols-2 gap-2">
        {content.variables.map((v, i) => (
          <div key={i}><code>{v.code}</code> - {v.description}</div>
        ))}
      </div>
    </div>
  );
}

// ==================== 主组件 ====================

const PeriodicTemplatePage: FC = () => {
  const [form] = Form.useForm();
  const [editForm] = Form.useForm();
  const [modalVisible, setModalVisible] = useState(false);
  const [logsDrawerVisible, setLogsDrawerVisible] = useState(false);

  // 使用全局分页 hook
  const { paginationProps, setCurrent: _setCurrent, setPageSize: _setPageSize } = usePagination();

  // 使用数据管理 Hook
  const {
    dataSource,
    total: _total,
    loading,
    categories,
    users,
    dutyPools,
    statistics,
    logs,
    selectedTemplate,
    fetchList,
    fetchCategories,
    fetchUsers,
    fetchDutyPools,
    fetchLogs,
    setSelectedTemplate,
  } = useTemplateData(form, paginationProps.current ?? 1, paginationProps.pageSize ?? 10);

  // 使用操作管理 Hook
  const {
    editingRecord,
    setEditingRecord,
    handleAdd,
    handleEdit,
    handleDelete,
    handleToggleEnabled,
    handleGenerateNow,
    handleSave,
  } = useTemplateActions({
    onLoad: fetchList,
  });

  // 初始化加载
  useEffect(() => {
    fetchList(1, paginationProps.pageSize);
    fetchCategories();
    fetchUsers();
    fetchDutyPools();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const handleTableChange = (pagination: TablePaginationConfig) => {
    fetchList(pagination.current || 1, pagination.pageSize || 10);
  };

  const handleSearch = () => {
    fetchList(1, paginationProps.pageSize);
  };

  const handleReset = () => {
    form.resetFields();
    fetchList(1, paginationProps.pageSize);
  };

  // 打开新增弹窗
  const handleOpenAddModal = () => {
    handleAdd(editForm);
    setModalVisible(true);
  };

  // 打开编辑弹窗
  const handleOpenEditModal = (record: PeriodicWorkOrderTemplate) => {
    handleEdit(record, editForm);
    setModalVisible(true);
  };

  // 查看执行记录
  const handleViewLogs = async (record: PeriodicWorkOrderTemplate) => {
    setSelectedTemplate(record);
    setLogsDrawerVisible(true);
    await fetchLogs(record.id);
  };

  // 保存弹窗
  const handleModalOk = async () => {
    await handleSave(editForm);
    setModalVisible(false);
  };

  // 表格列
  const columns = getTemplateTableColumns({
    current: paginationProps.current ?? 1,
    pageSize: paginationProps.pageSize ?? 10,
    handleEdit: handleOpenEditModal,
    handleToggleEnabled,
    handleGenerateNow,
    handleViewLogs,
    handleDelete,
  });

  return (
    <div className="p-6">
      {/* 统计卡片 - 只在数据大于10时显示 */}
      {statistics.total > 10 && (
        <Card title={null} className="mb-4">
          <Row gutter={16}>
            <Col span={6}>
              <Statistic title="总模板" value={statistics.total} />
            </Col>
            <Col span={6}>
              <Statistic title="已启用" value={statistics.enabled} styles={{ content: { color: "var(--theme-success, #3f8600)" } }} />
            </Col>
            <Col span={6}>
              <Statistic title="已禁用" value={statistics.disabled} styles={{ content: { color: "var(--theme-error, #cf1322)" } }} />
            </Col>
            <Col span={6}>
              <Statistic title="已生成工单" value={statistics.totalGenerated} />
            </Col>
          </Row>
        </Card>
      )}

      {/* 筛选表单和操作按钮 */}
      <Card style={{ marginBottom: 16 }}>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start", flexWrap: "wrap", gap: "16px" }}>
          <Form form={form} layout="inline" style={{ flex: 1, minWidth: 0 }}>
            <Form.Item name="title" label="模板名称">
              <Input placeholder="请输入模板名称" allowClear className="user-form-input" style={{ width: 150 }} />
            </Form.Item>
            <Form.Item name="isEnabled" label="状态">
              <Select placeholder="请选择状态" allowClear className="user-form-input" style={{ width: 100 }} onSearch={() => {}}>
                <Option value={true}>已启用</Option>
                <Option value={false}>已禁用</Option>
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
            <Button type="primary" icon={<PlusOutlined />} onClick={handleOpenAddModal}>
              新增模板
            </Button>
          </Space>
        </div>
      </Card>

      {/* 周期性工单模板表格 */}
      <Card>
        <Table
          rowKey="id"
          columns={columns}
          dataSource={dataSource}
          loading={loading}
          scroll={{ x: 1500 }}
          pagination={paginationProps}
          onChange={handleTableChange}
        />
      </Card>

      {/* 新增/编辑弹窗 */}
      <Modal
        title={editingRecord ? "编辑模板" : "新增模板"}
        open={modalVisible}
        onOk={handleModalOk}
        onCancel={() => {
          setModalVisible(false);
          setEditingRecord(null);
        }}
        width={700}
      >
        <Form form={editForm} layout="horizontal" labelCol={{ span: 4 }} wrapperCol={{ span: 20 }}>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="templateName"
                label="模板名称"
                rules={[{ required: true, message: "请输入模板名称" }]}
              >
                <Input placeholder="请输入模板名称" className="user-form-input" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="categoryId"
                label="工单分类"
                rules={[{ required: true, message: "请选择工单分类" }]}
              >
                <TreeSelect
                  placeholder="请选择工单分类"
                  treeData={buildCategoryTree(categories)}
                  showSearch
                  treeNodeFilterProp="title"
                  style={{ width: "100%" }}
                  className="user-form-input"
                />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item
            name="workOrderTitle"
            label="工单标题"
            rules={[{ required: true, message: "请输入工单标题" }]}
            extra={<VariableHelp content={VARIABLE_HELP_CONTENT} />}
          >
            <Input placeholder="例如：{date} 日例行检" className="user-form-input" />
          </Form.Item>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="type"
                label="工单类型"
                rules={[{ required: true, message: "请选择工单类型" }]}
              >
                <Select placeholder="请选择工单类型" className="user-form-input" onSearch={() => {}}>
                  {Object.entries(TYPE_CONFIG).map(([key, { text }]) => (
                    <Option key={key} value={key}>
                      {text}
                    </Option>
                  ))}
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="priority"
                label="优先级"
                rules={[{ required: true, message: "请选择优先级" }]}
              >
                <Select placeholder="请选择优先级" className="user-form-input" onSearch={() => {}}>
                  {Object.entries(PRIORITY_CONFIG).map(([key, { text }]) => (
                    <Option key={key} value={Number(key)}>
                      {text}
                    </Option>
                  ))}
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Form.Item
            name="cronExpression"
            label="Cron表达式"
            rules={[{ required: true, message: "请输入Cron表达式" }]}
          >
            <CronSelector />
          </Form.Item>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="assignType"
                label="分配类型"
                rules={[{ required: true, message: "请选择分配类型" }]}
              >
                <Select placeholder="请选择分配类型" className="user-form-input" onSearch={() => {}}>
                  {Object.entries(ASSIGN_TYPE_CONFIG).map(([key, { text }]) => (
                    <Option key={key} value={key}>
                      {text}
                    </Option>
                  ))}
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item noStyle shouldUpdate={(prevValues, currentValues) => prevValues.assignType !== currentValues.assignType}>
                {({ getFieldValue }) => {
                  const assignType = getFieldValue("assignType");
                  if (assignType === PeriodicAssignType.Manual) {
                    return (
                      <Form.Item name="assignTargetId" label="指定处理人">
                        <Select placeholder="请选择处理人" allowClear showSearch optionFilterProp="children" className="user-form-input" onSearch={() => {}}>
                          {users.map((user) => (
                            <Option key={user.id} value={user.id}>
                              {user.nickName || user.username}
                            </Option>
                          ))}
                        </Select>
                      </Form.Item>
                    );
                  } else if (assignType === PeriodicAssignType.DutyPool) {
                    return (
                      <Form.Item name="assignTargetId" label="选择值班池" rules={[{ required: true, message: "请选择值班池" }]}>
                        <Select placeholder="请选择值班池" allowClear showSearch optionFilterProp="children" className="user-form-input" onSearch={() => {}}>
                          {dutyPools.filter((p) => p.status === 0).map((pool) => (
                            <Option key={pool.id} value={pool.id}>
                              {pool.poolName} ({pool.members?.length || 0}人)
                            </Option>
                          ))}
                        </Select>
                      </Form.Item>
                    );
                  }
                  return null;
                }}
              </Form.Item>
            </Col>
          </Row>

          <Form.Item name="notifyAssignee" label="通知处理人" valuePropName="checked">
            <Switch />
          </Form.Item>

          <Form.Item name="description" label="描述">
            <TextArea rows={3} placeholder="请输入描述" />
          </Form.Item>
        </Form>
      </Modal>

      {/* 执行记录抽屉 */}
      <Drawer
        title="执行记录"
        placement="right"
        width={500}
        open={logsDrawerVisible}
        onClose={() => setLogsDrawerVisible(false)}
      >
        {selectedTemplate && (
          <div>
            <p><strong>模板名称：</strong>{selectedTemplate.templateName}</p>
            <p><strong>已生成工单：</strong>{selectedTemplate.totalGenerated}</p>

            {logs.length === 0 ? (
              <p className="text-gray-400 mt-4">暂无执行记录</p>
            ) : (
              <Timeline className="mt-4">
                {logs.map((log) => (
                  <Timeline.Item
                    key={log.id}
                    color={log.status === "success" ? "green" : "red"}
                  >
                    <div>
                      <strong>{formatDateTime(log.executedAt)}</strong>
                      <Tag color={log.status === "success" ? "green" : "red"} className="ml-2">
                        {log.status === "success" ? "成功" : "失败"}
                      </Tag>
                    </div>
                    <p className="mt-1">{log.result || log.errorMsg}</p>
                  </Timeline.Item>
                ))}
              </Timeline>
            )}
          </div>
        )}
      </Drawer>
    </div>
  );
};

export default PeriodicTemplatePage;

