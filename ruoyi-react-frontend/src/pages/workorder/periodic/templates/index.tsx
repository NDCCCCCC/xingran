import { useState, useEffect } from 'react';
import {
  Button,
  Form,
  Input,
  Select,
  Table,
  Modal,
  Space,
  Tag,
  message,
  Popconfirm,
  Card,
  Row,
  Col,
  Statistic,
  Drawer,
  Timeline,
  Switch,
  TreeSelect,
} from 'antd';
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
} from '@ant-design/icons';
import type { ColumnsType, TablePaginationConfig } from 'antd/es/table';
import dayjs from 'dayjs';
import utc from 'dayjs/plugin/utc';

dayjs.extend(utc);

const formatTime = (date: string | Date, format: string = 'YYYY-MM-DD HH:mm') => {
  if (!date) return '-';
  const dateStr = typeof date === 'string' ? date.replace('Z', '').replace('z', '') : date;
  return dayjs(dateStr).format(format);
};
import {
  getPeriodicTemplateList,
  createPeriodicTemplate,
  updatePeriodicTemplate,
  deletePeriodicTemplate,
  enablePeriodicTemplate,
  disablePeriodicTemplate,
  generateWorkOrderNow,
  getPeriodicLogs,
  type PeriodicWorkOrderTemplate,
  type PeriodicWorkOrderLog,
  type CreatePeriodicTemplateRequest,
  type UpdatePeriodicTemplateRequest,
  PeriodicAssignType,
  WorkOrderPriority,
  WorkOrderType,
} from '@/lib/workorderApi';
import {
  getEnabledWorkOrderCategories,
  type WorkOrderCategory,
} from '@/lib/workorderApi';
import {
  getUserList,
  type SimpleUser,
} from '@/lib/workorderApi';
import ActionButtons from '@/components/shared/ActionButtons';
import {
  getDutyPoolList,
  type DutyPool,
} from '@/lib/dutyApi';
import CronSelector from '@/components/CronSelector';

const { Option } = Select;
const { TextArea } = Input;

// 将工单分类数据转换为 TreeSelect 需要的树形结构
const buildCategoryTree = (categories: WorkOrderCategory[]): any[] => {
  return categories.map((cat) => ({
    title: cat.categoryName,
    value: cat.id,
    key: cat.id,
    children: cat.children && cat.children.length > 0 ? buildCategoryTree(cat.children) : undefined,
  }));
};

// 优先级配置
const PRIORITY_CONFIG = {
  [WorkOrderPriority.Low]: { text: '低', color: 'default' },
  [WorkOrderPriority.Medium]: { text: '中', color: 'blue' },
  [WorkOrderPriority.High]: { text: '高', color: 'orange' },
  [WorkOrderPriority.Urgent]: { text: '紧急', color: 'red' },
};

// 工单类型配置
const TYPE_CONFIG = {
  [WorkOrderType.Fault]: { text: '故障' },
  [WorkOrderType.Request]: { text: '请求' },
  [WorkOrderType.Change]: { text: '变更' },
  [WorkOrderType.Incident]: { text: '事件' },
  [WorkOrderType.Question]: { text: '咨询' },
};

// 分配类型配置
const ASSIGN_TYPE_CONFIG = {
  [PeriodicAssignType.Manual]: { text: '手动指定' },
  [PeriodicAssignType.DutyPool]: { text: '当天值班人员' },
  [PeriodicAssignType.Rotation]: { text: '轮询' },
};

import type { FC } from 'react';

const PeriodicTemplatePage: FC = () => {
  const [form] = Form.useForm();
  const [editForm] = Form.useForm();

  const [loading, setLoading] = useState(false);
  const [dataSource, setDataSource] = useState<PeriodicWorkOrderTemplate[]>([]);
  const [total, setTotal] = useState(0);
  const [current, setCurrent] = useState(1);
  const [pageSize, setPageSize] = useState(10);

  const [modalVisible, setModalVisible] = useState(false);
  const [logsDrawerVisible, setLogsDrawerVisible] = useState(false);
  const [editingRecord, setEditingRecord] = useState<PeriodicWorkOrderTemplate | null>(null);
  const [selectedTemplate, setSelectedTemplate] = useState<PeriodicWorkOrderTemplate | null>(null);
  const [logs, setLogs] = useState<PeriodicWorkOrderLog[]>([]);

  // 统计数据
  const [stats, setStats] = useState({ total: 0, enabled: 0, disabled: 0, totalGenerated: 0 });

  // 分类和用户数据
  const [categories, setCategories] = useState<WorkOrderCategory[]>([]);
  const [users, setUsers] = useState<SimpleUser[]>([]);
  // 值班池数据
  const [dutyPools, setDutyPools] = useState<DutyPool[]>([]);

  // 获取列表数据
  const fetchList = async (page?: number, pageSize?: number) => {
    setLoading(true);
    try {
      const values = form.getFieldsValue();
      const result = await getPeriodicTemplateList({
        current: page ?? current,
        pageSize: pageSize ?? pageSize,
        title: values.title,
        isEnabled: values.isEnabled,
      });
      setDataSource(result.data?.list ?? []);
      setTotal(result.data?.total ?? 0);
      setCurrent(result.data?.current ?? 1);
      setPageSize(result.data?.page_size ?? 10);

      // 计算统计数据
      const list = result.data?.list ?? [];
      setStats({
        total: list.length,
        enabled: list.filter((t) => t.isEnabled).length,
        disabled: list.filter((t) => !t.isEnabled).length,
        totalGenerated: list.reduce((sum, t) => sum + (t.totalGenerated || 0), 0),
      });
    } catch {
      message.error('获取模板列表失败');
    } finally {
      setLoading(false);
    }
  };

  // 获取分类列表
  const fetchCategories = async () => {
    try {
      const result = await getEnabledWorkOrderCategories();
      setCategories(result.data || []);
    } catch (error) {
      console.error('获取分类列表失败:', error);
    }
  };

  // 获取用户列表
  const fetchUsers = async () => {
    try {
      const result = await getUserList({ status: 0 });
      setUsers(result.data?.list || []);
    } catch (error) {
      console.error('获取用户列表失败:', error);
    }
  };

  // 获取值班池列表
  const fetchDutyPools = async () => {
    try {
      const result = await getDutyPoolList({ status: 0 });
      setDutyPools(result.data?.list || []);
    } catch (error) {
      console.error('获取值班池列表失败:', error);
    }
  };

  useEffect(() => {
    fetchList(1, 10);
    fetchCategories();
    fetchUsers();
    fetchDutyPools();
  }, []);

  const handleTableChange = (pagination: TablePaginationConfig) => {
    fetchList(pagination.current, pagination.pageSize);
  };

  const handleSearch = () => {
    fetchList(1, 10);
  };

  const handleReset = () => {
    form.resetFields();
    fetchList(1, 10);
  };

  const handleAdd = () => {
    setEditingRecord(null);
    editForm.resetFields();
    editForm.setFieldsValue({
      type: WorkOrderType.Fault,
      priority: WorkOrderPriority.Medium,
      assignType: PeriodicAssignType.DutyPool,
      notifyAssignee: true,
    });
    setModalVisible(true);
  };

  const handleEdit = (record: PeriodicWorkOrderTemplate) => {
    setEditingRecord(record);
    setModalVisible(true);
    setTimeout(() => {
      editForm.setFieldsValue({
        templateName: record.templateName,
        workOrderTitle: record.workOrderTitle,
        description: record.description,
        categoryId: record.categoryId,
        type: record.type,
        priority: record.priority,
        cronExpression: record.cronExpression,
        assignType: record.assignType,
        assignTargetId: record.assignTargetId,
        notifyAssignee: record.notifyAssignee,
      });
    }, 0);
  };

  const handleDelete = async (id: string) => {
    try {
      await deletePeriodicTemplate(id);
      message.success('删除成功');
      fetchList();
    } catch (error: any) {
      message.error(error.message || '删除失败');
    }
  };

  const handleToggleEnabled = async (record: PeriodicWorkOrderTemplate) => {
    try {
      if (record.isEnabled) {
        await disablePeriodicTemplate(record.id);
        message.success('已禁用');
      } else {
        await enablePeriodicTemplate(record.id);
        message.success('已启用');
      }
      fetchList();
    } catch (error: any) {
      message.error(error.message || '操作失败');
    }
  };

  const handleGenerateNow = async (id: string) => {
    try {
      await generateWorkOrderNow(id);
      message.success('工单生成成功');
      fetchList();
    } catch (error: any) {
      message.error(error.message || '生成失败');
    }
  };

  const handleViewLogs = async (record: PeriodicWorkOrderTemplate) => {
    setSelectedTemplate(record);
    setLogsDrawerVisible(true);

    try {
      const result = await getPeriodicLogs(record.id);
      setLogs(result.data || []);
    } catch (error) {
      console.error('获取执行记录失败:', error);
    }
  };

  const handleModalOk = async () => {
    try {
      const values = await editForm.validateFields();

      if (editingRecord) {
        const updateData: UpdatePeriodicTemplateRequest = {
          templateName: values.templateName,
          workOrderTitle: values.workOrderTitle,
          description: values.description,
          categoryId: values.categoryId,
          type: values.type,
          priority: values.priority,
          cronExpression: values.cronExpression,
          assignType: values.assignType,
          assignTargetId: values.assignTargetId,
          notifyAssignee: values.notifyAssignee,
        };
        await updatePeriodicTemplate(editingRecord.id, updateData);
        message.success('更新成功');
      } else {
        const createData: CreatePeriodicTemplateRequest = {
          templateName: values.templateName,
          workOrderTitle: values.workOrderTitle,
          description: values.description,
          categoryId: values.categoryId,
          type: values.type,
          priority: values.priority,
          cronExpression: values.cronExpression,
          assignType: values.assignType,
          assignTargetId: values.assignTargetId,
          notifyAssignee: values.notifyAssignee,
        };
        await createPeriodicTemplate(createData);
        message.success('创建成功');
      }
      setModalVisible(false);
      fetchList();
    } catch (error: any) {
      if (error.errorFields) {
        return;
      }
      message.error(editingRecord ? '更新失败' : '创建失败');
    }
  };

  const columns: ColumnsType<PeriodicWorkOrderTemplate> = [
    {
      title: '序号',
      key: 'index',
      width: 60,
      render: (_: unknown, __: unknown, index: number) => (current - 1) * pageSize + index + 1,
    },
    {
      title: '模板名称',
      dataIndex: 'templateName',
      key: 'templateName',
      width: 150,
    },
    {
      title: '工单标题',
      dataIndex: 'workOrderTitle',
      key: 'workOrderTitle',
      width: 200,
      ellipsis: true,
    },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      width: 80,
      render: (type: WorkOrderType) => TYPE_CONFIG[type]?.text || type,
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      width: 80,
      render: (priority: WorkOrderPriority) => (
        <Tag color={PRIORITY_CONFIG[priority]?.color}>{PRIORITY_CONFIG[priority]?.text}</Tag>
      ),
    },
    {
      title: 'Cron表达式',
      dataIndex: 'cronExpression',
      key: 'cronExpression',
      width: 120,
    },
    {
      title: '分配类型',
      dataIndex: 'assignType',
      key: 'assignType',
      width: 120,
      render: (type: PeriodicAssignType) => ASSIGN_TYPE_CONFIG[type]?.text || type,
    },
    {
      title: '下次执行',
      dataIndex: 'nextRunAt',
      key: 'nextRunAt',
      width: 160,
      render: (date: string) => formatTime(date),
    },
    {
      title: '已生成',
      dataIndex: 'totalGenerated',
      key: 'totalGenerated',
      width: 80,
    },
    {
      title: '状态',
      dataIndex: 'isEnabled',
      key: 'isEnabled',
      width: 80,
      render: (isEnabled: boolean) => (
        <Tag color={isEnabled ? 'green' : 'red'}>{isEnabled ? '已启用' : '已禁用'}</Tag>
      ),
    },
    {
      title: '操作',
      key: 'action',
      width: 100,
      fixed: 'right',
      render: (_: unknown, record: PeriodicWorkOrderTemplate) => {
        const actions = [
          {
            key: 'edit',
            label: '编辑',
            icon: <EditOutlined />,
            onClick: () => handleEdit(record),
          },
          {
            key: 'toggle',
            label: record.isEnabled ? '禁用' : '启用',
            icon: record.isEnabled ? <PauseCircleOutlined /> : <PlayCircleOutlined />,
            onClick: () => handleToggleEnabled(record),
          },
          {
            key: 'generate',
            label: '立即生成',
            icon: <ThunderboltOutlined />,
            onClick: () => handleGenerateNow(record.id),
          },
          {
            key: 'logs',
            label: '执行记录',
            icon: <HistoryOutlined />,
            onClick: () => handleViewLogs(record),
          },
          {
            key: 'delete',
            label: '删除',
            icon: <DeleteOutlined />,
            danger: true,
            render: () => (
              <Popconfirm title="确定要删除吗？" onConfirm={() => handleDelete(record.id)} okText="确定" cancelText="取消">
                <Button type="link" danger icon={<DeleteOutlined />} size="small">
                  删除
                </Button>
              </Popconfirm>
            ),
          },
        ];

        return <ActionButtons actions={actions} />;
      },
    },
  ];

  // 变量说明
  const variableHelp = (
    <div className="text-sm text-gray-500">
      <p className="font-semibold mb-2">工单标题支持以下变量：</p>
      <div className="grid grid-cols-2 gap-2">
        <div><code>{'{date}'}</code> - 当前日期 (2025-01-04)</div>
        <div><code>{'{datetime}'}</code> - 日期时间 (2025-01-04 15:30:45)</div>
        <div><code>{'{year}'}</code> - 年份 (2025)</div>
        <div><code>{'{month}'}</code> - 月份 (01)</div>
        <div><code>{'{day}'}</code> - 日 (04)</div>
        <div><code>{'{weekday}'}</code> - 星期 (Monday)</div>
        <div><code>{'{hour}'}</code> - 小时 (15)</div>
        <div><code>{'{minute}'}</code> - 分钟 (30)</div>
      </div>
    </div>
  );

  return (
    <div className="p-6">
      {/* 统计卡片 - 只在数据大于10时显示 */}
      {stats.total > 10 && (
        <Card title={null} className="mb-4">
          <Row gutter={16}>
            <Col span={6}>
              <Statistic title="总模板" value={stats.total} />
            </Col>
            <Col span={6}>
              <Statistic title="已启用" value={stats.enabled} styles={{ content: { color: '#3f8600' } }} />
            </Col>
            <Col span={6}>
              <Statistic title="已禁用" value={stats.disabled} styles={{ content: { color: '#cf1322' } }} />
            </Col>
            <Col span={6}>
              <Statistic title="已生成工单" value={stats.totalGenerated} />
            </Col>
          </Row>
        </Card>
      )}

      {/* 筛选表单和操作按钮 */}
      <Card style={{ marginBottom: 16 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', flexWrap: 'wrap', gap: '16px' }}>
          <Form form={form} layout="inline" style={{ flex: 1, minWidth: 0 }}>
            <Form.Item name="title" label="模板名称">
              <Input placeholder="请输入模板名称" allowClear className="user-form-input" style={{ width: 150 }} />
            </Form.Item>
            <Form.Item name="isEnabled" label="状态">
              <Select placeholder="请选择状态" allowClear className="user-form-input" style={{ width: 100 }}>
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
            <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
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
          pagination={{
            current,
            pageSize,
            total,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (t) => `共 ${t} 条`,
          }}
          onChange={handleTableChange}
        />
      </Card>

      {/* 新增/编辑弹窗 */}
      <Modal
        title={editingRecord ? '编辑模板' : '新增模板'}
        open={modalVisible}
        onOk={handleModalOk}
        onCancel={() => setModalVisible(false)}
        width={700}
      >
        <Form form={editForm} layout="horizontal" labelCol={{ span: 4 }} wrapperCol={{ span: 20 }}>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="templateName"
                label="模板名称"
                rules={[{ required: true, message: '请输入模板名称' }]}
              >
                <Input placeholder="请输入模板名称" className="user-form-input" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="categoryId"
                label="工单分类"
                rules={[{ required: true, message: '请选择工单分类' }]}
              >
                <TreeSelect
                  placeholder="请选择工单分类"
                  treeData={buildCategoryTree(categories)}
                  showSearch
                  treeNodeFilterProp="title"
                  style={{ width: '100%' }}
                  className="user-form-input"
                />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item
            name="workOrderTitle"
            label="工单标题"
            rules={[{ required: true, message: '请输入工单标题' }]}
            extra={variableHelp}
          >
            <Input placeholder="例如：{date} 日例行检" className="user-form-input" />
          </Form.Item>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="type"
                label="工单类型"
                rules={[{ required: true, message: '请选择工单类型' }]}
              >
                <Select placeholder="请选择工单类型" className="user-form-input">
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
                rules={[{ required: true, message: '请选择优先级' }]}
              >
                <Select placeholder="请选择优先级" className="user-form-input">
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
            rules={[{ required: true, message: '请输入Cron表达式' }]}
          >
            <CronSelector />
          </Form.Item>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="assignType"
                label="分配类型"
                rules={[{ required: true, message: '请选择分配类型' }]}
              >
                <Select placeholder="请选择分配类型" className="user-form-input">
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
                  const assignType = getFieldValue('assignType');
                  if (assignType === PeriodicAssignType.Manual) {
                    return (
                      <Form.Item name="assignTargetId" label="指定处理人">
                        <Select placeholder="请选择处理人" allowClear showSearch optionFilterProp="children" className="user-form-input">
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
                      <Form.Item name="assignTargetId" label="选择值班池" rules={[{ required: true, message: '请选择值班池' }]}>
                        <Select placeholder="请选择值班池" allowClear showSearch optionFilterProp="children" className="user-form-input">
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
                    color={log.status === 'success' ? 'green' : 'red'}
                  >
                    <div>
                      <strong>{formatTime(log.executedAt, 'YYYY-MM-DD HH:mm:ss')}</strong>
                      <Tag color={log.status === 'success' ? 'green' : 'red'} className="ml-2">
                        {log.status === 'success' ? '成功' : '失败'}
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
