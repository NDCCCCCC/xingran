import { useState, useEffect, useCallback, useMemo, useRef } from "react";
import type { FC } from "react";
import { App, Form } from "antd";
import type { TableProps } from "antd";
import { useNavigate } from "react-router-dom";
import type {
  Notice,
  CreateNoticeRequest,
  UpdateNoticeRequest,
  NoticeChannelRequest,
  NoticeType,
} from "@/types/notice";
import type { NoticeStatistics as NoticeStatisticsType } from "@/types/notice";
import { getNoticeStatistics } from "@/lib/noticeApi";
import type { ExecutionType } from "@/types/notice";
import dayjs from "dayjs";
import {
  useNoticeData,
  useNoticeStatistics,
  useTargetSelector,
  useAPIConfig,
} from "./hooks";
import {
  StatisticsDrawer,
  NoticeForm,
  NoticeList,
  NoticeStatisticsCard,
} from "./components";
import { usePagination } from "@/hooks/usePagination";
import { useServerSort, resolveSorter } from "@/hooks/useServerSort";
import type { SorterMeta } from "@/utils/tableHelpers";
import { createSorterMeta } from "@/utils/tableHelpers";

/**
 * 管理端通知公告页面（重构版）
 * 支持：Markdown编辑、定向推送、定时发布、优先级、阅读统计
 */
const NoticeManagement: FC = () => {
  const { message } = App.useApp();
  const navigate = useNavigate();
  const [searchForm] = Form.useForm();
  const [editForm] = Form.useForm();

  // 使用全局分页 hook
  const { paginationProps, setCurrent, setPageSize, setTotal } = usePagination();

  // 使用自定义 Hooks
  const {
    notices,
    loading,
    total,
    selectedRowKeys,
    loadNotices,
    handleCreate,
    handleUpdate,
    handleDelete,
    handleBatchDelete,
    handlePublish,
    handleWithdraw,
    setSelectedRowKeys,
  } = useNoticeData();

  // 服务端排序:field 对应后端 noticeAllowedSortFields 白名单 key
  const sorterMetas = useMemo<Array<SorterMeta<Notice>>>(
    () => [
      createSorterMeta<Notice>("noticeTitle"),
      createSorterMeta<Notice>("noticeType"),
      createSorterMeta<Notice>("priority", "number"),
      createSorterMeta<Notice>("publishTime", "date"),
      createSorterMeta<Notice>("createdAt", "date"),
    ],
    []
  );
  const sort = useServerSort<Notice>({ sorterMetas });

  // 列级 sortOrder：只对当前排序列返回方向，其余 undefined。
  // useServerSort 未直接暴露 getColumnSortOrder（那是 useTableManager 的封装），这里自行实现一致语义。
  const getColumnSortOrder = useCallback(
    (field: string): "ascend" | "descend" | null | undefined => {
      if (sort.orderByColumn !== String(field)) return undefined;
      return sort.sortOrder;
    },
    [sort.orderByColumn, sort.sortOrder]
  );

  // 排序 ref：保留最新排序值，供所有 loadNotices 调用（初始化/搜索/CRUD 刷新/分页）携带，
  // 避免 setState 时序导致 onChange 同周期读到旧值。
  const sortRef = useRef<{ orderByColumn?: string; isAsc?: boolean }>({});
  sortRef.current = {
    orderByColumn: sort.orderByColumn,
    isAsc: sort.isAsc,
  };

  // 统一的列表请求：合并当前分页 + 排序 ref + 可选 override（搜索条件等）。
  // 覆盖所有既有 loadNotices({current,pageSize}) 调用点，保证排序在 CRUD/搜索/分页/初始化后保持。
  const fetchList = useCallback((override?: {
    current?: number;
    pageSize?: number;
    noticeTitle?: string;
    noticeType?: string;
  }) => {
    loadNotices({
      current: override?.current ?? paginationProps.current,
      pageSize: override?.pageSize ?? paginationProps.pageSize,
      noticeTitle: override?.noticeTitle,
      noticeType: override?.noticeType as NoticeType | undefined,
      ...(sortRef.current.orderByColumn
        ? { orderByColumn: sortRef.current.orderByColumn, isAsc: sortRef.current.isAsc }
        : {}),
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps -- paginationProps is an object; .current/.pageSize tracked as primitives
  }, [loadNotices, paginationProps.current, paginationProps.pageSize]);

  const { statistics, loadStatistics } = useNoticeStatistics();

  // deptTree 由 useDeptTree 自动管理 (Phase 37 收敛),无 loadDeptTree
  const {
    deptTree,
    roles,
    users,
    loadingDepts,
    loadingRoles,
    loadingUsers,
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

  // 执行类型
  const [executionType, setExecutionType] = useState<ExecutionType>("once");

  // 渠道选择状态
  const [selectedChannels, setSelectedChannels] = useState<string[]>(["web"]);

  // 统计抽屉状态
  const [statisticsVisible, setStatisticsVisible] = useState(false);
  const [currentStatistics, setCurrentStatistics] = useState<NoticeStatisticsType | null>(null);
  const [statisticsLoading, setStatisticsLoading] = useState(false);

  // Markdown编辑器状态
  const [markdownContent, setMarkdownContent] = useState<string>("");

  // 初始化加载数据
  useEffect(() => {
    fetchList();
    loadStatistics();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [paginationProps.current, paginationProps.pageSize]);

  // 打开编辑模态框
  const openModal = useCallback(async (record?: Notice) => {
    setModalVisible(true);
    setExecutionType("once");
    setSelectedChannels(["web"]);

    // 加载目标选择数据和API配置
    // (deptTree 由 useDeptTree 自动拉取,无需手动触发;只加载 roles/users/apiConfigs)
    loadRoles();
    loadUsers();
    loadAPIConfigs();

    // 延迟设置表单值，等待 Modal 完全打开
    setTimeout(() => {
      if (record) {
        setEditingNotice(record);
        // 加载已有渠道配置
        let customEmailsValue = "";
        let customWeComUsersValue = "";
        let apiConfigIdValue = undefined;

        if (record.channels && record.channels.length > 0) {
          const channelTypes = record.channels.map((c: { channelType: string }) => c.channelType);
          setSelectedChannels(channelTypes);

          // 恢复 API 配置 ID
          const apiChannel = record.channels.find((c: { channelType: string; apiConfigId?: string }) => c.channelType === "api");
          if (apiChannel?.apiConfigId) {
            apiConfigIdValue = apiChannel.apiConfigId;
          }

          // 恢复自定义邮件地址
          const emailChannel = record.channels.find((c: { channelType: string; customRecipients?: string[] }) => c.channelType === "email");
          if (emailChannel?.customRecipients && emailChannel.customRecipients.length > 0) {
            customEmailsValue = emailChannel.customRecipients.join(", ");
          }

          // 恢复自定义企微用户
          const apiChannelRecipients = record.channels.find((c: { channelType: string; customRecipients?: string[] }) => c.channelType === "api");
          if (apiChannelRecipients?.customRecipients && apiChannelRecipients.customRecipients.length > 0) {
            customWeComUsersValue = apiChannelRecipients.customRecipients.join(", ");
          }
        }

        editForm.setFieldsValue({
          ...record,
          publishTime: record.publishTime ? dayjs(record.publishTime) : null,
          targetType: record.targetType ?? 0,
          targetDepts: record.targets?.filter((t: { targetType: string }) => t.targetType === "dept").map((t: { targetId: string }) => t.targetId) || [],
          targetRoles: record.targets?.filter((t: { targetType: string }) => t.targetType === "role").map((t: { targetId: string }) => t.targetId) || [],
          targetUsers: record.targets?.filter((t: { targetType: string }) => t.targetType === "user").map((t: { targetId: string }) => t.targetId) || [],
          apiConfigId: apiConfigIdValue,
          customEmails: customEmailsValue,
          customWeComUsers: customWeComUsersValue,
        });
        setMarkdownContent(record.noticeContent || "");
      } else {
        setEditingNotice(null);
        setMarkdownContent("");
        editForm.resetFields();
        editForm.setFieldsValue({
          noticeType: "1",
          status: 0,
          priority: 0,
          targetType: 0,
          isMarkdown: false,
          targetDepts: [],
          targetRoles: [],
          targetUsers: [],
          apiConfigId: undefined,
          customEmails: "",
          customWeComUsers: "",
        });
      }
    }, 0);
  }, [editForm, loadRoles, loadUsers, loadAPIConfigs]);

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
        const offsetSign = offsetMinutes <= 0 ? "+" : "-";
        const offsetStr = `${offsetSign}${String(offsetHours).padStart(2, "0")}:${String(offsetMins).padStart(2, "0")}`;
        publishTimeStr = values.publishTime.format("YYYY-MM-DDTHH:mm:ss") + offsetStr;
      }

      // 处理执行类型和周期配置
      let executionTypeValue: string | undefined = undefined;
      let recurrenceConfigValue: { cronExpression?: string; endDate?: string } | undefined = undefined;

      if (executionType === "recurring") {
        executionTypeValue = "recurring";
        const cronExpression = values.recurrenceConfig?.cronExpression;

        if (!cronExpression) {
          message.error("请输入 Cron 表达式");
          return;
        }

        recurrenceConfigValue = {
          cronExpression: cronExpression,
        };

        if (values.recurrenceConfig?.endDate) {
          recurrenceConfigValue.endDate = values.recurrenceConfig.endDate.format("YYYY-MM-DDTHH:mm:ss");
        }
      }

      const request: CreateNoticeRequest | UpdateNoticeRequest = {
        ...values,
        executionType: executionTypeValue,
        recurrenceConfig: recurrenceConfigValue,
      };

      if (executionType !== "recurring") {
        (request as CreateNoticeRequest & { publishTime?: string }).publishTime = publishTimeStr;
      } else {
        delete (request as CreateNoticeRequest & { publishTime?: string }).publishTime;
      }

      // 处理发送渠道配置
      const channels: NoticeChannelRequest[] = [];

      // 站内信渠道
      if (selectedChannels.includes("web")) {
        channels.push({ channelType: "web" });
      }

      // 邮件通知渠道
      if (selectedChannels.includes("email")) {
        const emailRecipients = values.customEmails?.trim()
          ? values.customEmails.split(",").map((e: string) => e.trim()).filter((e: string) => e)
          : undefined;
        channels.push({
          channelType: "email",
          ...(emailRecipients && emailRecipients.length > 0 && { customRecipients: emailRecipients })
        });
      }

      // 企微机器人渠道
      if (selectedChannels.includes("api")) {
        const apiConfigId = values.apiConfigId;
        if (!apiConfigId) {
          message.error("请选择企微机器人配置");
          return;
        }
        const weComRecipients = values.customWeComUsers?.trim()
          ? values.customWeComUsers.split(",").map((u: string) => u.trim()).filter((u: string) => u)
          : undefined;
        channels.push({
          channelType: "api",
          apiConfigId: apiConfigId,
          ...(weComRecipients && weComRecipients.length > 0 && { customRecipients: weComRecipients })
        });
      }

      // 短信渠道（预留）
      if (selectedChannels.includes("sms")) {
        channels.push({ channelType: "sms" });
      }

      if (channels.length > 0) {
        (request as CreateNoticeRequest & { channels?: NoticeChannelRequest[] }).channels = channels;
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
      fetchList();
    } catch (error) {
      if (error && typeof error === "object" && "errorFields" in error) {
        return;
      }
      message.error("操作失败: " + ((error as Error).message || "未知错误"));
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
  }, [editForm, executionType, selectedChannels, editingNotice, handleCreate, handleUpdate, fetchList]);

  // 查看统计
  const handleViewStatistics = useCallback(async (notice: Notice) => {
    setStatisticsVisible(true);
    setStatisticsLoading(true);
    try {
      const response = await getNoticeStatistics(notice.id);
      setCurrentStatistics(response.data || null);
    } catch (error) {
      console.error("加载统计数据失败:", error);
      message.error("加载统计数据失败");
    } finally {
      setStatisticsLoading(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
  }, []);

  // 删除通知（包含统计刷新）
  const handleDeleteWithRefresh = useCallback(async (id: string) => {
    await handleDelete(id);
    fetchList();
    loadStatistics();
  }, [handleDelete, fetchList, loadStatistics]);

  // 批量删除（包含统计刷新）
  const handleBatchDeleteWithRefresh = useCallback(async () => {
    await handleBatchDelete(selectedRowKeys);
    fetchList();
    loadStatistics();
  }, [handleBatchDelete, fetchList, loadStatistics, selectedRowKeys]);

  // 发布通知（包含列表刷新）
  const handlePublishWithRefresh = useCallback(async (id: string) => {
    await handlePublish(id);
    fetchList();
  }, [handlePublish, fetchList]);

  // 撤回通知（包含列表刷新）
  const handleWithdrawWithRefresh = useCallback(async (id: string) => {
    await handleWithdraw(id);
    fetchList();
  }, [handleWithdraw, fetchList]);

  // 搜索处理
  const handleSearch = useCallback((params?: Record<string, unknown>) => {
    const values = searchForm.getFieldsValue();
    fetchList({
      ...(params as { current?: number; pageSize?: number }),
      noticeTitle: values.noticeTitle,
      noticeType: values.noticeType,
    });
  }, [fetchList, searchForm]);

  // 分页处理（NoticeList 内联分页 onChange 直接调用）
  const handlePageChange = useCallback((page: number, size: number) => {
    fetchList({ current: page, pageSize: size || 10 });
  }, [fetchList]);

  // antd Table onChange：分页 + 服务端排序一起处理。
  // resolveSorter 同步取排序新值（规避 setState 时序），写 sortRef 供 fetchList 立即使用。
  const handleTableChange = useCallback<NonNullable<TableProps<Notice>["onChange"]>>(
    (pagination, filters, sorter) => {
      // 排序受控 UI（更新 sortOrder/orderByColumn/isAsc state）
      sort.handleTableChange(pagination, filters, sorter);
      // 同步取排序值写 ref（fetchList 不依赖尚未提交的 setState）
      const { orderByColumn, isAsc } = resolveSorter(sorter, sorterMetas);
      sortRef.current = { orderByColumn, isAsc };
      // 立即加载：带新分页 + 新排序 + 旧搜索条件（searchForm）
      const values = searchForm.getFieldsValue() as { noticeTitle?: string; noticeType?: string };
      fetchList({
        current: pagination.current,
        pageSize: pagination.pageSize,
        noticeTitle: values.noticeTitle,
        noticeType: values.noticeType,
      });
    },
    [sort, sorterMetas, searchForm, fetchList]
  );

  return (
    <div>
      {/* 统计卡片 */}
      <NoticeStatisticsCard statistics={statistics} />

      {/* 列表和搜索 */}
      <NoticeList
        notices={notices}
        loading={loading}
        total={total}
        current={paginationProps.current ?? 1}
        pageSize={paginationProps.pageSize ?? 10}
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
        getColumnSortOrder={getColumnSortOrder}
        onTableChange={handleTableChange}
      />

      {/* 编辑模态框 */}
      <NoticeForm
        visible={modalVisible}
        editingNotice={editingNotice}
        executionType={executionType}
        selectedChannels={selectedChannels}
        apiConfigs={apiConfigs}
        loadingAPIConfigs={loadingAPIConfigs}
        deptTree={deptTree}
        roles={roles}
        users={users}
        loadingDepts={loadingDepts}
        loadingRoles={loadingRoles}
        loadingUsers={loadingUsers}
        markdownContent={markdownContent}
        form={editForm}
        onCancel={() => {
          setModalVisible(false);
          editForm.resetFields();
          setEditingNotice(null);
          setMarkdownContent("");
        }}
        onSubmit={handleSubmit}
        onExecutionTypeChange={setExecutionType}
        onChannelsChange={setSelectedChannels}
        onMarkdownContentChange={setMarkdownContent}
        onTargetTypeChange={() => {
          editForm.setFieldValue("targetDepts", []);
          editForm.setFieldValue("targetRoles", []);
          editForm.setFieldValue("targetUsers", []);
        }}
        onDeptChange={(keys) => editForm.setFieldValue("targetDepts", keys)}
        onRoleChange={(values) => editForm.setFieldValue("targetRoles", values)}
        onUserChange={(values) => editForm.setFieldValue("targetUsers", values)}
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
