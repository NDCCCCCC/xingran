import { useState, useCallback } from "react";
import { App } from "antd";
import {
  getNoticeList,
  createNotice,
  updateNotice,
  deleteNotice,
  batchDeleteNotices,
  publishNotice,
  withdrawNotice,
} from "@/lib/noticeApi";
import type {
  Notice,
  CreateNoticeRequest,
  UpdateNoticeRequest,
  NoticeListParams,
} from "@/types/notice";

interface UseNoticeDataResult {
  notices: Notice[];
  loading: boolean;
  total: number;
  current: number;
  pageSize: number;
  selectedRowKeys: React.Key[];
  loadNotices: (params?: Partial<NoticeListParams>) => Promise<void>;
  handleCreate: (request: CreateNoticeRequest) => Promise<void>;
  handleUpdate: (id: string, request: UpdateNoticeRequest) => Promise<void>;
  handleDelete: (id: string) => Promise<void>;
  handleBatchDelete: (keys: React.Key[]) => Promise<void>;
  handlePublish: (id: string) => Promise<void>;
  handleWithdraw: (id: string) => Promise<void>;
  setSelectedRowKeys: (keys: React.Key[]) => void;
  setCurrent: (page: number) => void;
  setPageSize: (size: number) => void;
}

/**
 * 通知数据管理 Hook
 * 处理通知列表的加载、创建、更新、删除等操作
 */
export function useNoticeData(): UseNoticeDataResult {
  const { message } = App.useApp();
  const [notices, setNotices] = useState<Notice[]>([]);
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);
  const [current, setCurrent] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);

  // 加载通知列表
  const loadNotices = useCallback(
    async (params: Partial<NoticeListParams> = {}) => {
      setLoading(true);
      try {
        const requestParams: NoticeListParams = {
          current: params.current || current,
          pageSize: params.pageSize || pageSize,
          noticeTitle: params.noticeTitle,
          noticeType: params.noticeType,
          // 服务端排序透传：调用方（如 index.tsx fetchList 经 handleTableChange）
          // 携带的 orderByColumn/isAsc 必须透传到 getNoticeList，否则被丢弃导致排序失效。
          orderByColumn: params.orderByColumn,
          isAsc: params.isAsc,
        };
        const result = await getNoticeList(requestParams);
        setNotices(result.data?.list || []);
        setTotal(result.data?.total || 0);
      } catch (error) {
        console.error("加载通知公告失败:", error);
      } finally {
        setLoading(false);
      }
    },
    [current, pageSize]
  );

  // 创建通知
  const handleCreate = useCallback(async (request: CreateNoticeRequest) => {
    await createNotice(request);
    message.success("创建成功");
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
  }, []);

  // 更新通知
  const handleUpdate = useCallback(async (id: string, request: UpdateNoticeRequest) => {
    await updateNotice(id, request);
    message.success("更新成功");
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
  }, []);

  // 删除通知
  const handleDelete = useCallback(async (id: string) => {
    await deleteNotice(id);
    message.success("删除成功");
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
  }, []);

  // 批量删除
  const handleBatchDelete = useCallback(async (keys: React.Key[]) => {
    if (keys.length === 0) {
      message.warning("请选择要删除的数据");
      return;
    }
    await batchDeleteNotices(keys as string[]);
    message.success("批量删除成功");
    setSelectedRowKeys([]);
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
  }, []);

  // 发布通知
  const handlePublish = useCallback(async (id: string) => {
    await publishNotice(id);
    message.success("发布成功");
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
  }, []);

  // 撤回通知
  const handleWithdraw = useCallback(async (id: string) => {
    await withdrawNotice(id);
    message.success("撤回成功");
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
  }, []);

  return {
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
  };
}
