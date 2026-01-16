import { useState, useEffect, useCallback } from 'react';
import type { FC } from 'react';
import { Form, message } from 'antd';
import { useNavigate } from 'react-router-dom';
import type {
  Notice,
  CreateNoticeRequest,
  UpdateNoticeRequest,
  NoticeChannelRequest,
} from '@/types/notice';
import type { NoticeStatistics as NoticeStatisticsType } from '@/types/notice';
import { getNoticeStatistics } from '@/lib/noticeApi';
import type { ExecutionType, RecurrenceType } from '@/types/notice';
import dayjs from 'dayjs';
import {
  useNoticeData,
  useNoticeStatistics,
  useTargetSelector,
  useAPIConfig,
} from './hooks';
import {
  StatisticsDrawer,
  NoticeForm,
  NoticeList,
  NoticeStatisticsCard,
} from './components';

/**
 * 管理端通知公告页面（重构版）
 * 支持：Markdown编辑、定向推送、定时发布、优先级、阅读统计
 */
const NoticeManagement: FC = () => {
  const navigate = useNavigate();
  const [searchForm] = Form.useForm();
  const [editForm] = Form.useForm();

  // 使用自定义 Hooks
  const {
    notices,
    loading,
    total,
    current,
    pageSize,
    selectedRowKeys,
    loadNotices,
    handleCreate,
    handleUpdate,
    handleDelete,
    handleBatchDelete,
    handlePublish,
    handleWithdraw,
    setSelectedRowKeys,
    setCurrent,
    setPageSize,
  } = useNoticeData();

  const { statistics, loadStatistics } = useNoticeStatistics();

  const {
    deptTree,
    roles,
    users,
    loadingDepts,
    loadingRoles,
    loadingUsers,
    loadDeptTree,
    loadRoles,
    loadUsers,
  } = useTargetSelector();

  const {
    apiConfigs,
    loadingAPIConfigs,
    loadAPIConfigs,
  } = useAPIConfig();

  // 模态框状态
  const [modalVisible, setModalVisible] = useState(false);
  const [editingNotice, setEditingNotice] = useState<Notice | null>(null);

  // 执行类型和周期配置状态
  const [executionType, setExecutionType] = useState<ExecutionType>('once');
  const [recurrenceType, setRecurrenceType] = useState<RecurrenceType>('daily');

  // 渠道选择状态
  const [selectedChannels, setSelectedChannels] = useState<string[]>(['web']);
  const [selectedAPIConfigId, setSelectedAPIConfigId] = useState<string | undefined>();

  // 统计抽屉状态
  const [statisticsVisible, setStatisticsVisible] = useState(false);
  const [currentStatistics, setCurrentStatistics] = useState<NoticeStatisticsType | null>(null);
  const [statisticsLoading, setStatisticsLoading] = useState(false);

  // Markdown编辑器状态
  const [markdownContent, setMarkdownContent] = useState<string>('');

  // 自定义渠道地址状态
  const [customEmails, setCustomEmails] = useState<string>('');
  const [customWeComUsers, setCustomWeComUsers] = useState<string>('');

  // 初始化加载数据
  useEffect(() => {
    loadNotices();
    loadStatistics();
  }, [loadNotices, loadStatistics]);

  // 打开编辑模态框
  const openModal = useCallback(async (record?: Notice) => {
    setModalVisible(true);
    setExecutionType('once');
    setRecurrenceType('daily');
    setSelectedChannels(['web']);
    setSelectedAPIConfigId(undefined);
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
        setSelectedChannels(['web']);
      }
      editForm.setFieldsValue({
        ...record,
        publishTime: record.publishTime ? dayjs(record.publishTime) : null,
        targetType: record.targetType ?? 0,
        targetDepts: record.targets?.filter((t: any) => t.targetType === 'dept').map((t: any) => t.targetId) || [],
        targetRoles: record.targets?.filter((t: any) => t.targetType === 'role').map((t: any) => t.targetId) || [],
        targetUsers: record.targets?.filter((t: any) => t.targetType === 'user').map((t: any) => t.targetId) || [],
      });
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
  }, [editForm, loadDeptTree, loadRoles, loadUsers, loadAPIConfigs]);

  // 提交表单
  const handleSubmit = useCallback(async () => {
    try {
      const values = await editForm.validateFields();

      // 处理定时发布时间
      let publishTimeStr: string | undefined = undefined;
      if (values.publishTime) {
        const offsetMinutes = new Date().getTimezoneOffset();
        const offsetHours = Math.abs(Math.floor(offsetMinutes / 60));
        const offsetMins = Math.abs(offsetMinutes % 60);
        const offsetSign = offsetMinutes <= 0 ? '+' : '-';
        const offsetStr = `${offsetSign}${String(offsetHours).padStart(2, '0')}:${String(offsetMins).padStart(2, '0')}`;
        publishTimeStr = values.publishTime.format('YYYY-MM-DDTHH:mm:ss') + offsetStr;
      }

      // 处理执行类型和周期配置
      let executionTypeValue: string | undefined = undefined;
      let recurrenceConfigValue: any = undefined;

      if (executionType === 'recurring') {
        executionTypeValue = 'recurring';
        const executeTime = values.recurrenceConfig?.executeTime;
        const executeTimeStr = executeTime ? executeTime.format('HH:mm') : '09:00';

        recurrenceConfigValue = {
          type: recurrenceType,
          executeTime: executeTimeStr,
        };

        if (recurrenceType === 'weekly' && values.recurrenceConfig?.weekDay !== undefined) {
          recurrenceConfigValue.weekDay = values.recurrenceConfig.weekDay;
        }
        if (recurrenceType === 'monthly' && values.recurrenceConfig?.monthDay !== undefined) {
          recurrenceConfigValue.monthDay = values.recurrenceConfig.monthDay;
        }
        if (recurrenceType === 'custom' && values.recurrenceConfig?.cronExpression) {
          recurrenceConfigValue.cronExpression = values.recurrenceConfig.cronExpression;
        }
        if (values.recurrenceConfig?.endDate) {
          recurrenceConfigValue.endDate = values.recurrenceConfig.endDate.format('YYYY-MM-DDTHH:mm:ss');
        }
      }

      const request: CreateNoticeRequest | UpdateNoticeRequest = {
        ...values,
        executionType: executionTypeValue,
        recurrenceConfig: recurrenceConfigValue,
      };

      if (executionType !== 'recurring') {
        (request as any).publishTime = publishTimeStr;
      } else {
        delete (request as any).publishTime;
      }

      // 处理发送渠道配置
      const channels: NoticeChannelRequest[] = [];
      if (selectedChannels.includes('email')) {
        const emailRecipients = customEmails.trim()
          ? customEmails.split(',').map(e => e.trim()).filter(e => e)
          : undefined;
        channels.push({
          channelType: 'email',
          ...(emailRecipients && emailRecipients.length > 0 && { customRecipients: emailRecipients })
        });
      }
      if (selectedChannels.includes('api')) {
        if (!selectedAPIConfigId) {
          message.error('请选择企微机器人配置');
          return;
        }
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
      if (channels.length > 0) {
        (request as any).channels = channels;
      }

      if (editingNotice) {
        const updateRequest = request as UpdateNoticeRequest;
        if (editingNotice.publishTime && !values.publishTime) {
          updateRequest.clearPublishTime = true;
          updateRequest.publishTime = undefined;
        }
        await handleUpdate(editingNotice.id, updateRequest);
      } else {
        await handleCreate(request as CreateNoticeRequest);
      }

      setModalVisible(false);
      editForm.resetFields();
      setEditingNotice(null);
      loadNotices();
    } catch (error) {
      if (error && typeof error === 'object' && 'errorFields' in error) {
        return;
      }
      message.error('操作失败: ' + ((error as Error).message || '未知错误'));
    }
  }, [editForm, executionType, recurrenceType, selectedChannels, selectedAPIConfigId, customEmails, customWeComUsers, editingNotice, handleCreate, handleUpdate, loadNotices]);

  // 查看统计
  const handleViewStatistics = useCallback(async (notice: Notice) => {
    setStatisticsVisible(true);
    setStatisticsLoading(true);
    try {
      const response = await getNoticeStatistics(notice.id);
      setCurrentStatistics(response.data || null);
    } catch (error) {
      console.error('加载统计数据失败:', error);
      message.error('加载统计数据失败');
    } finally {
      setStatisticsLoading(false);
    }
  }, []);

  // 删除通知（包含统计刷新）
  const handleDeleteWithRefresh = useCallback(async (id: string) => {
    await handleDelete(id);
    loadNotices();
    loadStatistics();
  }, [handleDelete, loadNotices, loadStatistics]);

  // 批量删除（包含统计刷新）
  const handleBatchDeleteWithRefresh = useCallback(async () => {
    await handleBatchDelete(selectedRowKeys);
    loadNotices();
    loadStatistics();
  }, [handleBatchDelete, loadNotices, loadStatistics, selectedRowKeys]);

  // 发布通知（包含列表刷新）
  const handlePublishWithRefresh = useCallback(async (id: string) => {
    await handlePublish(id);
    loadNotices();
  }, [handlePublish, loadNotices]);

  // 撤回通知（包含列表刷新）
  const handleWithdrawWithRefresh = useCallback(async (id: string) => {
    await handleWithdraw(id);
    loadNotices();
  }, [handleWithdraw, loadNotices]);

  // 搜索处理
  const handleSearch = useCallback((params?: any) => {
    const values = searchForm.getFieldsValue();
    loadNotices({
      ...params,
      noticeTitle: values.noticeTitle,
      noticeType: values.noticeType,
    });
  }, [loadNotices, searchForm]);

  // 分页处理
  const handlePageChange = useCallback((page: number, size: number) => {
    setCurrent(page);
    setPageSize(size || 10);
    loadNotices({ current: page, pageSize: size || 10 });
  }, [loadNotices, setCurrent, setPageSize]);

  return (
    <div>
      {/* 统计卡片 */}
      <NoticeStatisticsCard statistics={statistics} />

      {/* 列表和搜索 */}
      <NoticeList
        notices={notices}
        loading={loading}
        total={total}
        current={current}
        pageSize={pageSize}
        selectedRowKeys={selectedRowKeys}
        searchForm={searchForm}
        onSearch={handleSearch}
        onAdd={() => openModal()}
        onEdit={openModal}
        onDelete={handleDeleteWithRefresh}
        onBatchDelete={handleBatchDeleteWithRefresh}
        onPublish={handlePublishWithRefresh}
        onWithdraw={handleWithdrawWithRefresh}
        onView={(id) => navigate(`/system/notice/${id}`)}
        onStatistics={handleViewStatistics}
        onSelectedRowKeysChange={setSelectedRowKeys}
        onPageChange={handlePageChange}
      />

      {/* 编辑模态框 */}
      <NoticeForm
        visible={modalVisible}
        editingNotice={editingNotice}
        executionType={executionType}
        recurrenceType={recurrenceType}
        selectedChannels={selectedChannels}
        selectedAPIConfigId={selectedAPIConfigId}
        apiConfigs={apiConfigs}
        loadingAPIConfigs={loadingAPIConfigs}
        deptTree={deptTree}
        roles={roles}
        users={users}
        loadingDepts={loadingDepts}
        loadingRoles={loadingRoles}
        loadingUsers={loadingUsers}
        markdownContent={markdownContent}
        customEmails={customEmails}
        customWeComUsers={customWeComUsers}
        form={editForm}
        onCancel={() => {
          setModalVisible(false);
          editForm.resetFields();
          setEditingNotice(null);
          setMarkdownContent('');
          setCustomEmails('');
          setCustomWeComUsers('');
        }}
        onSubmit={handleSubmit}
        onExecutionTypeChange={setExecutionType}
        onRecurrenceTypeChange={(type) => {
          setRecurrenceType(type);
          editForm.setFieldValue(['recurrenceConfig', 'weekDay'], undefined);
          editForm.setFieldValue(['recurrenceConfig', 'monthDay'], undefined);
          editForm.setFieldValue(['recurrenceConfig', 'cronExpression'], undefined);
        }}
        onChannelsChange={setSelectedChannels}
        onAPIConfigChange={setSelectedAPIConfigId}
        onMarkdownContentChange={setMarkdownContent}
        onCustomEmailsChange={setCustomEmails}
        onCustomWeComUsersChange={setCustomWeComUsers}
        onTargetTypeChange={() => {
          editForm.setFieldValue('targetDepts', []);
          editForm.setFieldValue('targetRoles', []);
          editForm.setFieldValue('targetUsers', []);
        }}
        onDeptChange={(keys) => editForm.setFieldValue('targetDepts', keys)}
        onRoleChange={(values) => editForm.setFieldValue('targetRoles', values)}
        onUserChange={(values) => editForm.setFieldValue('targetUsers', values)}
      />

      {/* 统计抽屉 */}
      <StatisticsDrawer
        visible={statisticsVisible}
        onClose={() => setStatisticsVisible(false)}
        loading={statisticsLoading}
        statistics={currentStatistics}
      />
    </div>
  );
};

export default NoticeManagement;
