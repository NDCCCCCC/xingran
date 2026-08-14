import { useState, useEffect, useCallback } from "react";
import type { FC } from "react";
import { App, Card, Table, Tag, Button, Space, Tabs, Tooltip } from "antd";
import { CheckOutlined, CheckSquareOutlined, EyeOutlined } from "@ant-design/icons";
import type { ColumnsType } from "antd/es/table";
import { useNavigate } from "react-router-dom";
import { useNoticeStore } from "@/store/noticeStore";
import { getMyNotices, markNoticeAsRead, markAllNoticesAsRead } from "@/lib/noticeApi";
import { usePagination } from "@/hooks/usePagination";
import type { Notice } from "@/types/notice";
import {
  PRIORITY_COLORS,
  PRIORITY_LABELS,
  NOTICE_TYPE_COLORS,
  NOTICE_TYPE_LABELS,
} from "@/types/notice";
import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";
import "dayjs/locale/zh-cn";
import { formatDateTime } from "@/utils/datetime";
import { createSorter } from "@/utils/tableHelpers";

dayjs.extend(relativeTime);
dayjs.locale("zh-cn");

/**
 * 用户通知中心页面
 * 显示用户可见的所有通知，支持按已读/未读筛选
 */
const MyNoticesPage: FC = () => {
  const { message } = App.useApp();
  const navigate = useNavigate();
  const { unreadCount, setUnreadCount, markAsRead, markAllAsRead } = useNoticeStore();

  const [loading, setLoading] = useState(false);
  const [notices, setNotices] = useState<Notice[]>([]);
  const [total, setTotal] = useState(0);
  const [allTotal, setAllTotal] = useState(0); // 全部通知总数（用于标签显示）
  const [activeTab, setActiveTab] = useState<"all" | "unread" | "read">("all");

  // 使用全局分页 hook
  const { paginationProps, setCurrent, setPageSize } = usePagination();

  // 加载通知列表
  const loadNotices = useCallback(async () => {
    setLoading(true);
    try {
      const status = activeTab === "all" ? undefined : activeTab;
      const response = await getMyNotices({
        current: paginationProps.current,
        pageSize: paginationProps.pageSize,
        status,
      });
      const data = response.data;
      if (data) {
        setNotices(data.list || []);
        setTotal(data.total);

        // 如果是"全部" tab，更新 allTotal
        if (activeTab === "all") {
          setAllTotal(data.total);
        }

        // 更新未读数量
        const unreadCount = (data.list || []).filter((n: Notice) => !n.isRead).length;
        setUnreadCount(unreadCount);
      }
    } catch (error) {
      console.error("加载通知列表失败:", error);
      message.error("加载失败，请稍后重试");
    } finally {
      setLoading(false);
    }
  }, [activeTab, paginationProps.current, paginationProps.pageSize, setUnreadCount]);

  // 初始化时获取全部通知总数
  useEffect(() => {
    const initAllTotal = async () => {
      try {
        const response = await getMyNotices({ current: 1, pageSize: 1 });
        if (response.data) {
          setAllTotal(response.data.total);
        }
      } catch (error) {
        console.error("获取全部通知数量失败:", error);
      }
    };
    initAllTotal();
  }, []);

  // 查看通知详情
  const handleViewDetail = async (notice: Notice) => {
    if (!notice.isRead) {
      try {
        await markNoticeAsRead(notice.id);
        markAsRead(notice.id);
        // 更新本地状态
        setNotices((prev) => prev.map((n) => (n.id === notice.id ? { ...n, isRead: true } : n)));
      } catch (error) {
        console.error("标记已读失败:", error);
      }
    }
    navigate(`/my-notices/${notice.id}`);
  };

  // 标记为已读
  const handleMarkAsRead = async (notice: Notice) => {
    if (notice.isRead) return;
    try {
      await markNoticeAsRead(notice.id);
      markAsRead(notice.id);
      message.success("已标记为已读");
      loadNotices();
    } catch (error) {
      console.error("标记已读失败:", error);
      message.error("操作失败，请稍后重试");
    }
  };

  // 全部标记为已读
  const handleMarkAllAsRead = async () => {
    try {
      await markAllNoticesAsRead();
      markAllAsRead();
      message.success("已全部标记为已读");
      loadNotices();
    } catch (error) {
      console.error("标记全部已读失败:", error);
      message.error("操作失败，请稍后重试");
    }
  };

  // 切换标签页
  const handleTabChange = (key: string) => {
    setActiveTab(key as "all" | "unread" | "read");
    setCurrent(1);
    loadNotices();
  };

  // 表格列定义
  const columns: ColumnsType<Notice> = [
    {
      title: "标题",
      dataIndex: "noticeTitle",
      key: "noticeTitle",
      ellipsis: true,
      sorter: createSorter<Notice>("noticeTitle", "string"),
      render: (text: string, record: Notice) => (
        <Space>
          {!record.isRead && <div className="w-2 h-2 rounded-full bg-blue-500" />}
          <span className={!record.isRead ? "font-medium" : ""}>{text}</span>
        </Space>
      ),
    },
    {
      title: "类型",
      dataIndex: "noticeType",
      key: "noticeType",
      width: 80,
      sorter: createSorter<Notice>("noticeType", "string"),
      render: (type: string) => (
        <Tag color={NOTICE_TYPE_COLORS[type as "1" | "2"]}>
          {NOTICE_TYPE_LABELS[type as "1" | "2"]}
        </Tag>
      ),
    },
    {
      title: "优先级",
      dataIndex: "priority",
      key: "priority",
      width: 80,
      sorter: createSorter<Notice>("priority", "number"),
      render: (priority: number) => {
        if (priority === 0) return <span className="text-gray-400">-</span>;
        return (
          <Tag color={PRIORITY_COLORS[priority as 0 | 1 | 2]}>
            {PRIORITY_LABELS[priority as 0 | 1 | 2]}
          </Tag>
        );
      },
    },
    {
      title: "发布时间",
      dataIndex: "publishTime",
      key: "publishTime",
      width: 180,
      sorter: createSorter<Notice>("publishTime", "date"),
      render: (time: string, record: Notice) => (
        <Tooltip title={formatDateTime(time || record.createdAt)}>
          <span className="text-gray-500">{dayjs(time || record.createdAt).fromNow()}</span>
        </Tooltip>
      ),
    },
    {
      title: "状态",
      dataIndex: "isRead",
      key: "isRead",
      width: 80,
      sorter: createSorter<Notice>("isRead", "boolean"),
      render: (isRead: boolean) => (
        <Tag color={isRead ? "default" : "blue"}>{isRead ? "已读" : "未读"}</Tag>
      ),
    },
    {
      title: "操作",
      key: "action",
      width: 180,
      fixed: "right",
      render: (_, record: Notice) => (
        <Space size="small">
          <Button
            type="link"
            size="small"
            icon={<EyeOutlined />}
            onClick={() => handleViewDetail(record)}
          >
            查看
          </Button>
          {!record.isRead && (
            <Button
              type="link"
              size="small"
              icon={<CheckOutlined />}
              onClick={() => handleMarkAsRead(record)}
            >
              标记已读
            </Button>
          )}
        </Space>
      ),
    },
  ];

  useEffect(() => {
    loadNotices();
  }, [activeTab, paginationProps.current, paginationProps.pageSize, loadNotices]);

  const tabItems = [
    { key: "all", label: `全部通知 (${allTotal})` },
    { key: "unread", label: `未读 (${unreadCount})` },
    { key: "read", label: "已读" },
  ];

  return (
    <div className="p-6">
      <Card
        title={
          <div className="flex items-center justify-between">
            <span>通知中心</span>
            {unreadCount > 0 && (
              <Button
                type="link"
                size="small"
                icon={<CheckSquareOutlined />}
                onClick={handleMarkAllAsRead}
              >
                全部已读
              </Button>
            )}
          </div>
        }
        variant="borderless"
      >
        <Tabs activeKey={activeTab} items={tabItems} onChange={handleTabChange} />

        <Table
          columns={columns}
          dataSource={notices}
          loading={loading}
          rowKey="id"
          pagination={paginationProps}
          onChange={(pagination) => {
            setCurrent(pagination.current ?? 1);
            setPageSize(pagination.pageSize ?? 10);
            loadNotices();
          }}
          onRow={(record) => ({
            className: !record.isRead ? "bg-blue-50" : "",
          })}
        />
      </Card>
    </div>
  );
};

export default MyNoticesPage;
