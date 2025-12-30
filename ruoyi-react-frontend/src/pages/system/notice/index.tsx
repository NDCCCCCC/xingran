import React, { useState, useEffect } from 'react';
import {
  Table,
  Button,
  Space,
  Modal,
  Form,
  Input,
  Select,
  message,
  Popconfirm,
  Tag,
  Card,
  Row,
  Col,
  Statistic,
  DatePicker,
  Switch,
  Drawer,
  Spin,
  Tree,
  Checkbox,
  Radio,
  TimePicker,
} from 'antd';
import MDEditor from '@uiw/react-md-editor';
import '@uiw/react-md-editor/markdown-editor.css';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  SearchOutlined,
  ReloadOutlined,
  EyeOutlined,
  BarChartOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { useNavigate } from 'react-router-dom';
import {
  getNoticeList,
  createNotice,
  updateNotice,
  deleteNotice,
  batchDeleteNotices,
  getNoticeStatistics,
  publishNotice,
  withdrawNotice,
} from '@/lib/noticeApi';
import type {
  Notice,
  CreateNoticeRequest,
  UpdateNoticeRequest,
  NoticeListParams,
  NoticeStatistics,
  TargetType,
  NoticePriority,
  ExecutionType,
  RecurrenceType,
  NoticeChannelRequest,
} from '@/types/notice';
import {
  PRIORITY_COLORS,
  PRIORITY_LABELS,
  NOTICE_TYPE_COLORS,
  NOTICE_TYPE_LABELS,
  PUBLISH_STATUS_COLORS,
  PUBLISH_STATUS_LABELS,
  TARGET_TYPE_LABELS,
} from '@/types/notice';
import dayjs from 'dayjs';
import { get, post } from '@/lib/api';
import { getAPINotificationConfigList } from '@/lib/notificationConfigApi';
import type { APINotificationConfig } from '@/lib/notificationConfigApi';

const { Option } = Select;
const { TextArea } = Input;

/**
 * 管理端通知公告页面（增强版）
 * 支持：Markdown编辑、定向推送、定时发布、优先级、阅读统计
 */
const NoticeManagement: React.FC = () => {
  const navigate = useNavigate();

  // 列表相关状态
  const [notices, setNotices] = useState<Notice[]>([]);
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);
  const [current, setCurrent] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [searchForm] = Form.useForm();
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);

  // 编辑模态框相关状态
  const [modalVisible, setModalVisible] = useState(false);
  const [editForm] = Form.useForm();
  const [editingNotice, setEditingNotice] = useState<Notice | null>(null);
  // 执行类型和周期配置状态
  const [executionType, setExecutionType] = useState<ExecutionType>('once');
  const [recurrenceType, setRecurrenceType] = useState<RecurrenceType>('daily');

  // 渠道选择状态
  const [selectedChannels, setSelectedChannels] = useState<string[]>(['web']);
  const [selectedAPIConfigId, setSelectedAPIConfigId] = useState<string | undefined>();

  // API配置列表状态
  const [apiConfigs, setApiConfigs] = useState<APINotificationConfig[]>([]);
  const [loadingAPIConfigs, setLoadingAPIConfigs] = useState(false);

  // 目标选择器相关状态
  const [deptTree, setDeptTree] = useState<any[]>([]);
  const [roles, setRoles] = useState<any[]>([]);
  const [users, setUsers] = useState<any[]>([]);
  const [loadingDepts, setLoadingDepts] = useState(false);
  const [loadingRoles, setLoadingRoles] = useState(false);
  const [loadingUsers, setLoadingUsers] = useState(false);

  // 统计相关状态
  const [statisticsVisible, setStatisticsVisible] = useState(false);
  const [currentStatistics, setCurrentStatistics] = useState<NoticeStatistics | null>(null);
  const [statisticsLoading, setStatisticsLoading] = useState(false);

  // Markdown编辑器状态
  const [markdownContent, setMarkdownContent] = useState<string>('');

  // 自定义渠道地址状态
  const [customEmails, setCustomEmails] = useState<string>('');
  const [customWeComUsers, setCustomWeComUsers] = useState<string>('');

  // 加载部门树
  const loadDeptTree = async () => {
    setLoadingDepts(true);
    try {
      // 后端返回的格式是 { code: 0, data: { list: [...] }, message: "success" }
      const response: { data: { list: any[] } } = await get('/system/departments/tree');
      // 转换为 Tree 组件需要的格式
      const convertTree = (nodes: any[]): any[] => {
        return nodes.map(node => ({
          ...node,
          title: node.deptName,
          key: node.id,
          value: node.id,
          children: node.children ? convertTree(node.children) : undefined,
        }));
      };
      setDeptTree(convertTree(response.data?.list || []));
    } catch (error) {
      console.error('加载部门树失败:', error);
    } finally {
      setLoadingDepts(false);
    }
  };

  // 加载角色列表
  const loadRoles = async () => {
    setLoadingRoles(true);
    try {
      // 后端返回的格式是 { code: 0, data: [...], message: "success" }
      const response: { data: any[] } = await post('/system/roles/all', {});
      setRoles(response.data || []);
    } catch (error) {
      console.error('加载角色列表失败:', error);
    } finally {
      setLoadingRoles(false);
    }
  };

  // 加载用户列表
  const loadUsers = async () => {
    setLoadingUsers(true);
    try {
      // 后端返回的格式是 { code: 0, data: { list: [...], total: ..., current: ..., pageSize: ... }, message: "success" }
      const response: { data: { list: any[] } } = await post('/system/users/list', { current: 1, pageSize: 100 });
      setUsers(response.data?.list || []);
    } catch (error) {
      console.error('加载用户列表失败:', error);
    } finally {
      setLoadingUsers(false);
    }
  };

  // 加载API配置列表
  const loadAPIConfigs = async () => {
    setLoadingAPIConfigs(true);
    try {
      const result = await getAPINotificationConfigList({
        page: 1,
        pageSize: 100,
      });
      // 只显示状态为正常(0)的配置
      setApiConfigs(result.data.list.filter((c: APINotificationConfig) => c.status === 0));
    } catch (error) {
      console.error('加载API配置失败:', error);
    } finally {
      setLoadingAPIConfigs(false);
    }
  };

  // 加载通知列表
  const loadNotices = async (params: any = {}) => {
    setLoading(true);
    try {
      const values = searchForm.getFieldsValue();
      const requestParams: NoticeListParams = {
        current: params.current || current,
        pageSize: params.pageSize || pageSize,
        noticeTitle: values.noticeTitle,
        noticeType: values.noticeType,
      };
      const result = await getNoticeList(requestParams);
      setNotices(result.data.list);
      setTotal(result.data.total);
    } catch (error) {
      console.error('加载通知公告失败:', error);
    } finally {
      setLoading(false);
    }
  };

  // 打开编辑模态框
  const openModal = async (record?: Notice) => {
    setModalVisible(true);
    // 重置执行类型状态
    setExecutionType('once');
    setRecurrenceType('daily');
    // 重置渠道选择
    setSelectedChannels(['web']);
    setSelectedAPIConfigId(undefined);
    // 重置自定义渠道地址
    setCustomEmails('');
    setCustomWeComUsers('');
    // 加载目标选择数据和API配置
    loadDeptTree();
    loadRoles();
    loadUsers();
    loadAPIConfigs();

    if (record) {
      setEditingNotice(record);
      // 加载已有渠道配置
      if (record.channels && record.channels.length > 0) {
        const channelTypes = record.channels.map((c: any) => c.channelType);
        setSelectedChannels(channelTypes);
      } else {
        setSelectedChannels(['web']); // 默认站内信
      }
      editForm.setFieldsValue({
        ...record,
        publishTime: record.publishTime ? dayjs(record.publishTime) : null,
        targetType: record.targetType ?? 0,
        targetDepts: record.targets?.filter((t: any) => t.targetType === 'dept').map((t: any) => t.targetId) || [],
        targetRoles: record.targets?.filter((t: any) => t.targetType === 'role').map((t: any) => t.targetId) || [],
        targetUsers: record.targets?.filter((t: any) => t.targetType === 'user').map((t: any) => t.targetId) || [],
      });
      // 设置 markdown 内容
      setMarkdownContent(record.noticeContent || '');
    } else {
      setEditingNotice(null);
      setMarkdownContent('');
      editForm.resetFields();
      editForm.setFieldsValue({
        noticeType: '1',
        status: 0,
        priority: 0,
        targetType: 0,
        isMarkdown: false,
        targetDepts: [],
        targetRoles: [],
        targetUsers: [],
      });
    }
  };

  // 创建或更新通知
  const handleSubmit = async () => {
    try {
      const values = await editForm.validateFields();

      // 处理定时发布时间：添加本地时区偏移（+08:00）
      let publishTimeStr: string | undefined = undefined;
      if (values.publishTime) {
        // 获取本地时区偏移（分钟）
        const offsetMinutes = new Date().getTimezoneOffset();
        const offsetHours = Math.abs(Math.floor(offsetMinutes / 60));
        const offsetMins = Math.abs(offsetMinutes % 60);
        const offsetSign = offsetMinutes <= 0 ? '+' : '-'; // 注意：getTimezoneOffset() 返回的是相反数
        const offsetStr = `${offsetSign}${String(offsetHours).padStart(2, '0')}:${String(offsetMins).padStart(2, '0')}`;

        // 格式化为带时区偏移的字符串：2025-12-30T11:06:34+08:00
        publishTimeStr = values.publishTime.format('YYYY-MM-DDTHH:mm:ss') + offsetStr;
      }

      // 处理执行类型和周期配置
      let executionTypeValue: string | undefined = undefined;
      let recurrenceConfigValue: any = undefined;

      if (executionType === 'recurring') {
        executionTypeValue = 'recurring';
        // 处理执行时间
        const executeTime = values.recurrenceConfig?.executeTime;
        const executeTimeStr = executeTime ? executeTime.format('HH:mm') : '09:00';

        // 构建周期配置，只包含有值的字段
        recurrenceConfigValue = {
          type: recurrenceType,
          executeTime: executeTimeStr,
        };

        // 根据周期类型添加相应的字段
        if (recurrenceType === 'weekly' && values.recurrenceConfig?.weekDay !== undefined) {
          recurrenceConfigValue.weekDay = values.recurrenceConfig.weekDay;
        }
        if (recurrenceType === 'monthly' && values.recurrenceConfig?.monthDay !== undefined) {
          recurrenceConfigValue.monthDay = values.recurrenceConfig.monthDay;
        }
        if (recurrenceType === 'custom' && values.recurrenceConfig?.cronExpression) {
          recurrenceConfigValue.cronExpression = values.recurrenceConfig.cronExpression;
        }
        // 处理结束时间（所有周期类型都支持）
        if (values.recurrenceConfig?.endDate) {
          recurrenceConfigValue.endDate = values.recurrenceConfig.endDate.format('YYYY-MM-DDTHH:mm:ss');
        }
      }

      const request: CreateNoticeRequest | UpdateNoticeRequest = {
        ...values,
        executionType: executionTypeValue,
        recurrenceConfig: recurrenceConfigValue,
      };

      // 只有一次性执行才需要 publishTime
      if (executionType !== 'recurring') {
        (request as any).publishTime = publishTimeStr;
      } else {
        // 周期性执行不发送 publishTime
        delete (request as any).publishTime;
      }

      // 处理发送渠道配置
      const channels: NoticeChannelRequest[] = [];
      if (selectedChannels.includes('email')) {
        // 如果有自定义邮件地址，添加到渠道配置
        const emailRecipients = customEmails.trim()
          ? customEmails.split(',').map(e => e.trim()).filter(e => e)
          : undefined;
        channels.push({
          channelType: 'email',
          ...(emailRecipients && emailRecipients.length > 0 && { customRecipients: emailRecipients })
        });
      }
      if (selectedChannels.includes('api')) {
        // API渠道必须选择配置
        if (!selectedAPIConfigId) {
          message.error('请选择企微机器人配置');
          return;
        }
        // 如果有自定义企微用户，添加到渠道配置
        const weComRecipients = customWeComUsers.trim()
          ? customWeComUsers.split(',').map(u => u.trim()).filter(u => u)
          : undefined;
        channels.push({
          channelType: 'api',
          apiConfigId: selectedAPIConfigId,
          ...(weComRecipients && weComRecipients.length > 0 && { customRecipients: weComRecipients })
        });
      }
      if (selectedChannels.includes('sms')) {
        channels.push({ channelType: 'sms' });
      }
      // web 是默认渠道，不需要特别配置，但如果只选择了web则不发送channels字段
      if (channels.length > 0) {
        (request as any).channels = channels;
      }

      if (editingNotice) {
        // 编辑模式：需要处理清除定时发布时间的逻辑
        const updateRequest = request as UpdateNoticeRequest;
        // 如果原来是定时发布状态，现在清空了时间，需要标记清除
        if (editingNotice.publishTime && !values.publishTime) {
          updateRequest.clearPublishTime = true;
          updateRequest.publishTime = undefined;
        }
        await updateNotice(editingNotice.id, updateRequest);
        message.success('更新成功');
      } else {
        await createNotice(request as CreateNoticeRequest);
        message.success('创建成功');
      }

      setModalVisible(false);
      editForm.resetFields();
      setEditingNotice(null);
      loadNotices();
    } catch (error: any) {
      if (error.errorFields) {
        return;
      }
      message.error('操作失败: ' + (error.message || '未知错误'));
    }
  };

  // 删除通知
  const handleDelete = async (id: string) => {
    try {
      await deleteNotice(id);
      message.success('删除成功');
      loadNotices();
    } catch (error) {
      message.error('删除失败');
    }
  };

  // 批量删除
  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) {
      message.warning('请选择要删除的数据');
      return;
    }
    try {
      await batchDeleteNotices(selectedRowKeys as string[]);
      message.success('批量删除成功');
      setSelectedRowKeys([]);
      loadNotices();
    } catch (error) {
      message.error('批量删除失败');
    }
  };

  // 查看统计
  const handleViewStatistics = async (notice: Notice) => {
    setStatisticsVisible(true);
    setStatisticsLoading(true);
    try {
      const response = await getNoticeStatistics(notice.id);
      setCurrentStatistics(response.data);
    } catch (error) {
      console.error('加载统计数据失败:', error);
      message.error('加载统计数据失败');
    } finally {
      setStatisticsLoading(false);
    }
  };

  // 发布通知
  const handlePublish = async (id: string) => {
    try {
      await publishNotice(id);
      message.success('发布成功');
      loadNotices();
    } catch (error) {
      message.error('发布失败');
    }
  };

  // 撤回/取消发布通知
  const handleWithdraw = async (id: string) => {
    try {
      await withdrawNotice(id);
      message.success('撤回成功');
      loadNotices();
    } catch (error) {
      message.error('撤回失败');
    }
  };

  // 表格列定义
  const columns: ColumnsType<Notice> = [
    {
      title: '公告标题',
      dataIndex: 'noticeTitle',
      key: 'noticeTitle',
      ellipsis: true,
      render: (text: string) => <span className="font-medium">{text}</span>,
    },
    {
      title: '类型',
      dataIndex: 'noticeType',
      key: 'noticeType',
      width: 80,
      render: (type: string) => (
        <Tag color={NOTICE_TYPE_COLORS[type as '1' | '2']}>{NOTICE_TYPE_LABELS[type as '1' | '2']}</Tag>
      ),
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      width: 90,
      render: (priority: number) => {
        if (priority === 0) return <span className="text-gray-400">普通</span>;
        return (
          <Tag color={PRIORITY_COLORS[priority as NoticePriority]}>{PRIORITY_LABELS[priority as NoticePriority]}</Tag>
        );
      },
    },
    {
      title: '接收范围',
      dataIndex: 'targetType',
      key: 'targetType',
      width: 100,
      render: (type: number) => <span className="text-gray-500">{TARGET_TYPE_LABELS[type as TargetType]}</span>,
    },
    {
      title: '发布状态',
      dataIndex: 'publishStatus',
      key: 'publishStatus',
      width: 100,
      render: (status: number) => (
        <Tag color={PUBLISH_STATUS_COLORS[status as 0 | 1 | 2 | 3]}>{PUBLISH_STATUS_LABELS[status as 0 | 1 | 2 | 3]}</Tag>
      ),
    },
    {
      title: '发布时间',
      dataIndex: 'publishTime',
      key: 'publishTime',
      width: 180,
      render: (time: string) => time ? dayjs(time).format('YYYY-MM-DD HH:mm:ss') : <span className="text-gray-400">-</span>,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 80,
      render: (status: number) => <Tag color={status === 0 ? 'success' : 'default'}>{status === 0 ? '正常' : '关闭'}</Tag>,
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 180,
    },
    {
      title: '操作',
      key: 'action',
      fixed: 'right',
      width: 280,
      render: (_, record: Notice) => (
        <Space size="small" wrap>
          <Button type="link" size="small" icon={<EyeOutlined />} onClick={() => navigate(`/system/notice/${record.id}`)}>
            查看
          </Button>
          <Button type="link" size="small" icon={<BarChartOutlined />} onClick={() => handleViewStatistics(record)}>
            统计
          </Button>
          {/* 草稿或定时发布状态：显示发布按钮 */}
          {(record.publishStatus === 0 || record.publishStatus === 2) && (
            <Button type="link" size="small" onClick={() => handlePublish(record.id)}>
              发布
            </Button>
          )}
          {/* 已发布状态：显示撤回按钮 */}
          {record.publishStatus === 1 && (
            <Popconfirm title="确认撤回该通知? 撤回后通知将退回到草稿状态" onConfirm={() => handleWithdraw(record.id)}>
              <Button type="link" size="small" danger>
                撤回
              </Button>
            </Popconfirm>
          )}
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => openModal(record)}>
            编辑
          </Button>
          <Popconfirm title="确认删除?" onConfirm={() => handleDelete(record.id)}>
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  useEffect(() => {
    loadNotices();
  }, [current, pageSize]);

  // 使用 Form.useWatch 监听表单值
  const targetType = Form.useWatch('targetType', editForm);
  const targetDepts = Form.useWatch('targetDepts', editForm) || [];
  const targetRoles = Form.useWatch('targetRoles', editForm) || [];
  const targetUsers = Form.useWatch('targetUsers', editForm) || [];
  const isMarkdown = Form.useWatch('isMarkdown', editForm) || false;

  // 渲染目标选择内容
  const renderTargetSelector = () => {

    if (targetType === 0) {
      return <div className="text-gray-500 py-4 text-center">将向所有用户发送此通知</div>;
    }

    if (targetType === 1) {
      // 指定部门
      return (
        <Spin spinning={loadingDepts}>
          <div className="max-h-64 overflow-y-auto border rounded p-4">
            <Tree
              checkable
              checkedKeys={targetDepts}
              onCheck={(keys) => editForm.setFieldValue('targetDepts', keys)}
              treeData={deptTree}
              fieldNames={{ title: 'title', key: 'key', children: 'children' }}
            />
          </div>
          <div className="mt-2 text-sm text-gray-500">已选择 {targetDepts.length} 个部门（包含子部门）</div>
        </Spin>
      );
    }

    if (targetType === 2) {
      // 指定角色
      return (
        <Spin spinning={loadingRoles}>
          <Checkbox.Group
            value={targetRoles}
            onChange={(values) => editForm.setFieldValue('targetRoles', values)}
            className="w-full"
          >
            <div className="grid grid-cols-2 gap-2 max-h-64 overflow-y-auto">
              {roles.map((role) => (
                <Checkbox key={role.id} value={role.id}>
                  {role.roleName}
                  <span className="text-xs text-gray-400 ml-1">({role.roleKey})</span>
                </Checkbox>
              ))}
            </div>
          </Checkbox.Group>
          <div className="mt-2 text-sm text-gray-500">已选择 {targetRoles.length} 个角色</div>
        </Spin>
      );
    }

    if (targetType === 3) {
      // 指定用户
      return (
        <Spin spinning={loadingUsers}>
          <Select
            mode="multiple"
            value={targetUsers}
            onChange={(values) => editForm.setFieldValue('targetUsers', values)}
            placeholder="请选择用户"
            showSearch
            filterOption={(input, option) => {
              const user = users.find((u: any) => u.id === option.value);
              return (
                user?.username?.toLowerCase().includes(input.toLowerCase()) ||
                user?.nickname?.toLowerCase().includes(input.toLowerCase())
              );
            }}
            className="w-full"
            options={users.map((user: any) => ({
              label: `${user.nickname || user.username} (${user.username})`,
              value: user.id,
            }))}
          />
          <div className="mt-2 text-sm text-gray-500">已选择 {targetUsers.length} 个用户</div>
        </Spin>
      );
    }

    return null;
  };

  return (
    <div>
      {/* 搜索表单 */}
      <Form form={searchForm} layout="inline" style={{ marginBottom: 16 }}>
        <Form.Item name="noticeTitle" label="公告标题">
          <Input placeholder="请输入公告标题" allowClear />
        </Form.Item>
        <Form.Item name="noticeType" label="公告类型">
          <Select placeholder="请选择" allowClear style={{ width: 120 }}>
            <Option value="1">公告</Option>
            <Option value="2">警告</Option>
          </Select>
        </Form.Item>
        <Form.Item>
          <Space>
            <Button type="primary" icon={<SearchOutlined />} onClick={() => loadNotices()}>
              查询
            </Button>
            <Button icon={<ReloadOutlined />} onClick={() => searchForm.resetFields()}>
              重置
            </Button>
          </Space>
        </Form.Item>
      </Form>

      {/* 操作按钮 */}
      <div style={{ marginBottom: 16 }}>
        <Space>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => openModal()}>
            新增
          </Button>
          <Button danger icon={<DeleteOutlined />} onClick={handleBatchDelete} disabled={selectedRowKeys.length === 0}>
            批量删除
          </Button>
        </Space>
      </div>

      {/* 数据表格 */}
      <Table
        rowSelection={{
          selectedRowKeys,
          onChange: setSelectedRowKeys,
        }}
        columns={columns}
        dataSource={notices}
        loading={loading}
        rowKey="id"
        scroll={{ x: 1600 }}
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

      {/* 编辑模态框 */}
      <Modal
        title={editingNotice ? '编辑通知公告' : '新增通知公告'}
        open={modalVisible}
        onOk={handleSubmit}
        onCancel={() => {
          setModalVisible(false);
          editForm.resetFields();
          setEditingNotice(null);
          setMarkdownContent('');
          setCustomEmails('');
          setCustomWeComUsers('');
        }}
        width={900}
        footer={[
          <Button key="cancel" onClick={() => {
            setModalVisible(false);
            editForm.resetFields();
            setEditingNotice(null);
            setMarkdownContent('');
            setCustomEmails('');
            setCustomWeComUsers('');
          }}>
            取消
          </Button>,
          <Button key="submit" type="primary" onClick={handleSubmit}>
            {editingNotice ? '更新' : '创建'}
          </Button>,
        ]}
      >
        <Form form={editForm} labelCol={{ span: 4 }} wrapperCol={{ span: 18 }}>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="noticeTitle" label="标题" rules={[{ required: true, message: '请输入公告标题' }]}>
                <Input placeholder="请输入公告标题" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="noticeType" label="类型" rules={[{ required: true }]}>
                <Select>
                  <Option value="1">公告</Option>
                  <Option value="2">警告</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="priority" label="优先级" initialValue={0}>
                <Select>
                  <Option value={0}>普通</Option>
                  <Option value={1}>重要</Option>
                  <Option value={2}>紧急</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="status" label="状态" initialValue={0}>
                <Select>
                  <Option value={0}>正常</Option>
                  <Option value={1}>关闭</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>

          {/* 执行类型和发布配置 */}
          <Form.Item label="执行方式">
            <Radio.Group
              value={executionType}
              onChange={(e) => {
                setExecutionType(e.target.value);
                // 清空相关字段
                editForm.setFieldValue('publishTime', null);
              }}
            >
              <Radio value="once">立即发布/定时发布</Radio>
              <Radio value="recurring">周期性发送</Radio>
            </Radio.Group>
          </Form.Item>

          {executionType === 'once' ? (
            <Row gutter={16}>
              <Col span={12}>
                <Form.Item name="publishTime" label="发布时间">
                  <DatePicker showTime format="YYYY-MM-DD HH:mm:ss" className="w-full" />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="isMarkdown" label="Markdown" valuePropName="checked">
                  <Switch />
                </Form.Item>
              </Col>
            </Row>
          ) : (
            <>
              <Row gutter={16}>
                <Col span={12}>
                  <Form.Item label="周期类型" required>
                    <Select
                      value={recurrenceType}
                      onChange={(value) => {
                        setRecurrenceType(value);
                        // 清空周期相关字段
                        editForm.setFieldValue(['recurrenceConfig', 'weekDay'], undefined);
                        editForm.setFieldValue(['recurrenceConfig', 'monthDay'], undefined);
                        editForm.setFieldValue(['recurrenceConfig', 'cronExpression'], undefined);
                      }}
                    >
                      <Option value="daily">每天</Option>
                      <Option value="weekly">每周</Option>
                      <Option value="monthly">每月</Option>
                      <Option value="custom">自定义 Cron</Option>
                    </Select>
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item name={['recurrenceConfig', 'executeTime']} label="执行时间" required>
                    <TimePicker format="HH:mm" className="w-full" />
                  </Form.Item>
                </Col>
              </Row>

              {recurrenceType === 'weekly' && (
                <Row gutter={16}>
                  <Col span={12}>
                    <Form.Item name={['recurrenceConfig', 'weekDay']} label="选择星期" required>
                      <Select placeholder="请选择星期几">
                        <Option value={1}>周日</Option>
                        <Option value={2}>周一</Option>
                        <Option value={3}>周二</Option>
                        <Option value={4}>周三</Option>
                        <Option value={5}>周四</Option>
                        <Option value={6}>周五</Option>
                        <Option value={7}>周六</Option>
                      </Select>
                    </Form.Item>
                  </Col>
                </Row>
              )}

              {recurrenceType === 'monthly' && (
                <Row gutter={16}>
                  <Col span={12}>
                    <Form.Item name={['recurrenceConfig', 'monthDay']} label="每月几号" required>
                      <Select placeholder="请选择日期" min={1} max={31}>
                        {Array.from({ length: 31 }, (_, i) => (
                          <Option key={i + 1} value={i + 1}>{i + 1}号</Option>
                        ))}
                      </Select>
                    </Form.Item>
                  </Col>
                </Row>
              )}

              {recurrenceType === 'custom' && (
                <Row gutter={16}>
                  <Col span={24}>
                    <Form.Item
                      name={['recurrenceConfig', 'cronExpression']}
                      label="Cron 表达式"
                      required
                      rules={[{ required: true, message: '请输入 Cron 表达式' }]}
                      extra="格式：秒 分 时 日 月 周，例如：0 0 9 * * ? （每天上午9点）"
                    >
                      <Input placeholder="0 0 9 * * ?" />
                    </Form.Item>
                  </Col>
                </Row>
              )}

              <Row gutter={16}>
                <Col span={12}>
                  <Form.Item name={['recurrenceConfig', 'endDate']} label="结束时间（可选）">
                    <DatePicker format="YYYY-MM-DD HH:mm:ss" className="w-full" />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item name="isMarkdown" label="Markdown" valuePropName="checked">
                    <Switch />
                  </Form.Item>
                </Col>
              </Row>
            </>
          )}

          {/* 隐藏字段：执行类型 */}
          <Form.Item name="executionType" hidden>
            <Input />
          </Form.Item>
          <Form.Item name="recurrenceConfig" hidden>
            <Input />
          </Form.Item>

          {/* 目标选择器 - 集成到表单中 */}
          <Form.Item label="接收范围" required>
            <Row gutter={8}>
              <Col span={12}>
                <Form.Item name="targetType" noStyle initialValue={0}>
                  <Select
                    onChange={(value) => {
                      // 切换目标类型时清空已选数据
                      editForm.setFieldValue('targetDepts', []);
                      editForm.setFieldValue('targetRoles', []);
                      editForm.setFieldValue('targetUsers', []);
                    }}
                  >
                    <Option value={0}>全部用户</Option>
                    <Option value={1}>指定部门</Option>
                    <Option value={2}>指定角色</Option>
                    <Option value={3}>指定用户</Option>
                  </Select>
                </Form.Item>
              </Col>
            </Row>
          </Form.Item>

          <Form.Item label=" " colon={false}>
            <div className="border rounded p-4 bg-gray-50">{renderTargetSelector()}</div>
          </Form.Item>

          <Form.Item name="targetDepts" hidden>
            <Input />
          </Form.Item>
          <Form.Item name="targetRoles" hidden>
            <Input />
          </Form.Item>
          <Form.Item name="targetUsers" hidden>
            <Input />
          </Form.Item>

          {/* 发送渠道配置 */}
          <Form.Item label="发送渠道">
            <Checkbox.Group
              value={selectedChannels}
              onChange={(checkedValues) => {
                const newChannels = checkedValues as string[];
                setSelectedChannels(newChannels);
                // 如果取消选择API渠道，清除API配置选择
                if (!newChannels.includes('api')) {
                  setSelectedAPIConfigId(undefined);
                }
              }}
            >
              <Checkbox value="web">站内信</Checkbox>
              <Checkbox value="email">邮件通知</Checkbox>
              <Checkbox value="api">企微机器人</Checkbox>
            </Checkbox.Group>
            <div className="text-xs text-gray-500 mt-1">
              邮件通知：使用系统邮件配置发送；企微机器人：需要选择具体配置
            </div>
          </Form.Item>

          {/* API配置选择器（仅当选择API渠道时显示） */}
          {selectedChannels.includes('api') && (
            <Form.Item
              label="企微机器人配置"
              rules={[{ required: true, message: '请选择企微机器人配置' }]}
            >
              <Select
                placeholder="请选择企微机器人配置"
                value={selectedAPIConfigId}
                onChange={setSelectedAPIConfigId}
                loading={loadingAPIConfigs}
                allowClear
              >
                {apiConfigs.map((config) => (
                  <Option key={config.id} value={config.id}>
                    {config.configName}
                    {config.configType === 'webhook' && ` (${config.httpMethod})`}
                  </Option>
                ))}
              </Select>
              <div className="text-xs text-gray-500 mt-1">
                {apiConfigs.length === 0 && '暂无可用的API配置，请先在系统设置中添加'}
              </div>
            </Form.Item>
          )}

          {/* 自定义邮件地址（仅当选择邮件渠道时显示） */}
          {selectedChannels.includes('email') && (
            <Form.Item label="自定义邮件地址" extra="多个邮件地址用逗号分隔，留空则发送给目标用户">
              <Input
                placeholder="例如: user1@example.com, user2@example.com"
                value={customEmails}
                onChange={(e) => setCustomEmails(e.target.value)}
                allowClear
              />
            </Form.Item>
          )}

          {/* 自定义企微用户代码（仅当选择API渠道时显示） */}
          {selectedChannels.includes('api') && (
            <Form.Item label="自定义企微用户" extra="多个用户代码用逗号分隔，留空则发送给目标用户">
              <Input
                placeholder="例如: USER001, USER002"
                value={customWeComUsers}
                onChange={(e) => setCustomWeComUsers(e.target.value)}
                allowClear
              />
            </Form.Item>
          )}

          <Form.Item name="noticeContent" label="内容" rules={[{ required: true, message: '请输入公告内容' }]}>
            {isMarkdown ? (
              <MDEditor
                value={markdownContent}
                onChange={(val) => {
                  const content = val || '';
                  setMarkdownContent(content);
                  editForm.setFieldValue('noticeContent', content);
                }}
                preview="live"
                height={400}
                data-color-mode="light"
              />
            ) : (
              <TextArea
                rows={10}
                placeholder="请输入公告内容"
                value={editForm.getFieldValue('noticeContent')}
                onChange={(e) => {
                  editForm.setFieldValue('noticeContent', e.target.value);
                }}
              />
            )}
          </Form.Item>
        </Form>
      </Modal>

      {/* 统计抽屉 */}
      <Drawer
        title="阅读统计"
        placement="right"
        size="default"
        open={statisticsVisible}
        onClose={() => setStatisticsVisible(false)}
      >
        {statisticsLoading ? (
          <div className="flex justify-center py-8">
            <Spin />
          </div>
        ) : currentStatistics ? (
          <div className="space-y-6">
            <Card>
              <Statistic title="目标用户数" value={currentStatistics.totalTargets} />
            </Card>
            <Card>
              <Statistic title="已读数" value={currentStatistics.readCount} suffix="人" />
            </Card>
            <Card>
              <Statistic title="未读数" value={currentStatistics.unreadCount} suffix="人" />
            </Card>
            <Card>
              <Statistic
                title="阅读率"
                value={currentStatistics.readRate}
                precision={1}
                suffix="%"
                valueStyle={{ color: currentStatistics.readRate >= 50 ? '#3f8600' : '#cf1322' }}
              />
            </Card>
          </div>
        ) : (
          <div className="text-center text-gray-500 py-8">暂无统计数据</div>
        )}
      </Drawer>
    </div>
  );
};

export default NoticeManagement;
