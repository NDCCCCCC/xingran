import { useState, useRef, useEffect, useCallback } from "react";
import type { FC, MouseEvent } from "react";
import { App, Badge, Dropdown, Empty, Spin, Tag, Avatar } from "antd";
import { BellOutlined, DeleteOutlined } from "@ant-design/icons";
import { useNavigate } from "react-router-dom";
import { USER_NOTICES } from "@/constants/routes";
import { useNoticeStore } from "@/store/noticeStore";
import {
  getNotificationList,
  markNoticeAsRead,
  markAllNoticesAsRead,
  ignoreNotice,
} from "@/lib/noticeApi";
import type { NoticeListItem } from "@/types/notice";
import {
  PRIORITY_COLORS,
  PRIORITY_LABELS,
  NOTICE_TYPE_COLORS,
  NOTICE_TYPE_LABELS,
} from "@/types/notice";
import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";
import "dayjs/locale/zh-cn";

dayjs.extend(relativeTime);
dayjs.locale("zh-cn");

// ==================== 样式常量 ====================
// 遵循 Vercel React Best Practices: rendering-hoist-jsx
// 将静态样式对象提取到组件外部，避免每次渲染重新创建

const NOTIFICATION_CONTENT_STYLE = {
  width: "400px",
  maxHeight: "480px",
  background: "var(--theme-bg-primary)",
} as const;

const HEADER_STYLE = {
  borderBottomColor: "var(--theme-border-primary)",
  background: "linear-gradient(to bottom, var(--theme-bg-primary), var(--theme-bg-secondary))",
} as const;

const LIST_CONTAINER_STYLE = {
  maxHeight: "400px",
  background: "var(--theme-bg-secondary)",
} as const;

const EMPTY_STYLE = {
  color: "var(--theme-text-secondary)",
} as const;

const FOOTER_STYLE = {
  borderTopColor: "var(--theme-border-secondary)",
  background: "var(--theme-bg-tertiary)",
} as const;

const BADGE_STYLE = {
  background: "var(--theme-error)",
  boxShadow: "0 2px 8px rgba(239, 68, 68, 0.3)",
} as const;

const AVATAR_WRAPPER_STYLE = {
  display: "inline-block",
} as const;

const AVATAR_STYLE = {
  background: "linear-gradient(135deg, var(--theme-primary) 0%, var(--theme-primary-hover) 100%)",
  boxShadow: "0 2px 8px rgba(79, 70, 229, 0.2)",
  transition: "transform 300ms, box-shadow 300ms",
} as const;

/**
 * 通知铃铛组件
 * 显示未读数量，点击展开通知列表
 */
const NotificationBell: FC = () => {
  const { message } = App.useApp();
  const navigate = useNavigate();
  const {
    unreadCount,
    notifications,
    loading,
    markAsRead,
    markAllAsRead,
    removeNotification,
    setNotifications,
  } = useNoticeStore();

  // WebSocket 连接已在 Header 组件中初始化，此处无需重复调用

  const [dropdownOpen, setDropdownOpen] = useState(false);
  const hasLoadedRef = useRef(false);

  // 下拉框打开时加载通知列表（只加载一次）
  // 遵循 Vercel React Best Practices: 直接在 useEffect 中调用 API，避免不必要的 useCallback
  useEffect(() => {
    if (dropdownOpen && notifications.length === 0 && !hasLoadedRef.current) {
      getNotificationList({ current: 1, pageSize: 10 })
        .then((response) => {
          setNotifications(response.data?.list || []);
        })
        .catch((error) => {
          console.error("加载通知列表失败:", error);
        });
      hasLoadedRef.current = true;
    }
    // 当下拉框关闭时，重置加载标记，以便下次打开时可以重新加载
    if (!dropdownOpen) {
      hasLoadedRef.current = false;
    }
  }, [dropdownOpen, notifications.length, setNotifications]);

  // 点击通知项 - 使用 useCallback 避免每次渲染重新创建函数
  // 遵循 Vercel React Best Practices: rerender-dependencies
  const handleClickNotification = useCallback(
    async (notice: NoticeListItem) => {
      // 标记为已读
      if (!notice.isRead) {
        try {
          await markNoticeAsRead(notice.id);
          markAsRead(notice.id);
        } catch (error) {
          console.error("标记已读失败:", error);
        }
      }

      // 跳转到通知详情页面
      setDropdownOpen(false);
      navigate(`/my-notices/${notice.id}`);
    },
    [markAsRead, navigate]
  );

  // 全部标记为已读
  const handleMarkAllRead = useCallback(async () => {
    try {
      await markAllNoticesAsRead();
      markAllAsRead();
      message.success("已全部标记为已读");
    } catch (error) {
      console.error("标记全部已读失败:", error);
      message.error("操作失败，请稍后重试");
    }
  }, [markAllAsRead]);

  // 删除通知（调用忽略API）
  const handleDelete = useCallback(
    async (e: MouseEvent, noticeId: string) => {
      e.stopPropagation();
      try {
        await ignoreNotice(noticeId);
        // 从前端列表中移除
        removeNotification(noticeId);
        message.success("已忽略该通知");
      } catch (error) {
        console.error("忽略通知失败:", error);
        message.error("操作失败，请稍后重试");
      }
    },
    [removeNotification]
  );

  // 查看全部通知
  const handleViewAll = useCallback(() => {
    setDropdownOpen(false);
    navigate(USER_NOTICES);
  }, [navigate]);

  // 渲染通知列表
  const notificationContent = (
    <div
      className="rounded-lg overflow-hidden border border-slate-200 shadow-lg"
      style={NOTIFICATION_CONTENT_STYLE}
    >
      {/* 头部 */}
      <div className="flex items-center justify-between px-4 py-3 border-b" style={HEADER_STYLE}>
        <div className="flex items-center gap-2">
          <BellOutlined className="text-sm" style={{ color: "var(--theme-primary)" }} />
          <span className="font-semibold text-sm" style={{ color: "var(--theme-text-primary)" }}>
            通知中心
          </span>
          {unreadCount > 0 && (
            <span
              className="px-2 py-0.5 rounded-full text-xs font-medium"
              style={{
                background: "var(--theme-primary)",
                color: "var(--theme-text-inverse)",
              }}
            >
              {unreadCount}
            </span>
          )}
        </div>
        <div className="flex items-center gap-3">
          {unreadCount > 0 && (
            <span
              className="text-xs cursor-pointer transition-colors duration-200 font-medium"
              style={{
                color: "var(--theme-primary)",
              }}
              onClick={handleMarkAllRead}
            >
              全部已读
            </span>
          )}
          <span
            className="text-xs cursor-pointer transition-colors duration-200"
            style={{
              color: "var(--theme-text-secondary)",
            }}
            onClick={handleViewAll}
            onMouseEnter={(e) => (e.currentTarget.style.color = "var(--theme-primary)")}
            onMouseLeave={(e) => (e.currentTarget.style.color = "var(--theme-text-secondary)")}
          >
            查看全部
          </span>
        </div>
      </div>

      {/* 通知列表 */}
      <Spin spinning={loading}>
        {notifications.length === 0 ? (
          <div className="py-12 px-4 text-center">
            <Empty
              description="暂无通知"
              image={Empty.PRESENTED_IMAGE_SIMPLE}
              style={EMPTY_STYLE}
            />
          </div>
        ) : (
          <div className="overflow-y-auto" style={LIST_CONTAINER_STYLE}>
            {notifications.map((notice, index) => (
              <div
                key={notice.id}
                className="transition-all duration-200 cursor-pointer relative"
                style={{
                  padding: "12px 16px",
                  borderBottom:
                    index < notifications.length - 1
                      ? "1px solid var(--theme-border-secondary)"
                      : "none",
                  background: !notice.isRead
                    ? "linear-gradient(to right, var(--theme-primary-light), transparent)"
                    : "transparent",
                  borderLeft: !notice.isRead
                    ? "3px solid var(--theme-primary)"
                    : "3px solid transparent",
                }}
                onClick={() => handleClickNotification(notice)}
                onMouseEnter={(e) => {
                  if (!notice.isRead) return;
                  e.currentTarget.style.background = "var(--theme-primary-light)";
                }}
                onMouseLeave={(e) => {
                  if (!notice.isRead) return;
                  e.currentTarget.style.background = "transparent";
                }}
              >
                <div className="flex gap-3">
                  {/* 通知图标 */}
                  <div
                    className="flex-shrink-0 w-8 h-8 rounded-lg flex items-center justify-center"
                    style={{
                      background: NOTICE_TYPE_COLORS[notice.noticeType]
                        ? `${NOTICE_TYPE_COLORS[notice.noticeType]}15`
                        : "var(--theme-neutral-200)",
                    }}
                  >
                    <BellOutlined
                      className="text-sm"
                      style={{
                        color:
                          NOTICE_TYPE_COLORS[notice.noticeType] || "var(--theme-text-secondary)",
                      }}
                    />
                  </div>

                  {/* 通知内容 */}
                  <div className="flex-1 min-w-0">
                    <div className="flex items-start justify-between gap-2 mb-1">
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 mb-1">
                          {!notice.isRead && (
                            <span
                              className="inline-block w-2 h-2 rounded-full"
                              style={{
                                background: "var(--theme-primary)",
                                boxShadow: "0 0 0 3px var(--theme-primary-light)",
                              }}
                            />
                          )}
                          <span
                            className="font-medium text-sm truncate"
                            style={{
                              color: !notice.isRead
                                ? "var(--theme-text-primary)"
                                : "var(--theme-text-primary)",
                              fontWeight: !notice.isRead ? 600 : 400,
                            }}
                          >
                            {notice.noticeTitle}
                          </span>
                        </div>
                        <div className="flex items-center gap-2 flex-wrap">
                          <Tag
                            color={NOTICE_TYPE_COLORS[notice.noticeType]}
                            variant="filled"
                            className="text-xs"
                            style={{ margin: 0 }}
                          >
                            {NOTICE_TYPE_LABELS[notice.noticeType]}
                          </Tag>
                          {notice.priority > 0 && (
                            <Tag
                              color={PRIORITY_COLORS[notice.priority]}
                              variant="filled"
                              className="text-xs"
                              style={{ margin: 0 }}
                            >
                              {PRIORITY_LABELS[notice.priority]}
                            </Tag>
                          )}
                          <span className="text-xs" style={{ color: "var(--theme-text-tertiary)" }}>
                            {dayjs(notice.createdAt).fromNow()}
                          </span>
                        </div>
                      </div>
                      <DeleteOutlined
                        className="flex-shrink-0 text-sm transition-all duration-200"
                        style={{ color: "var(--theme-text-tertiary)" }}
                        onMouseEnter={(e) => {
                          e.currentTarget.style.color = "var(--theme-error)";
                          e.currentTarget.style.transform = "scale(1.1)";
                        }}
                        onMouseLeave={(e) => {
                          e.currentTarget.style.color = "var(--theme-text-tertiary)";
                          e.currentTarget.style.transform = "scale(1)";
                        }}
                        onClick={(e) => handleDelete(e, notice.id)}
                      />
                    </div>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </Spin>

      {/* 底部 */}
      {notifications.length > 0 && (
        <div className="px-4 py-2 border-t text-center" style={FOOTER_STYLE}>
          <span
            className="text-xs cursor-pointer transition-colors duration-200 font-medium"
            style={{ color: "var(--theme-primary)" }}
            onClick={handleViewAll}
          >
            查看全部通知 →
          </span>
        </div>
      )}
    </div>
  );

  return (
    <Dropdown
      open={dropdownOpen}
      onOpenChange={setDropdownOpen}
      popupRender={() => notificationContent}
      placement="bottomRight"
      trigger={["click"]}
    >
      <Badge count={unreadCount} size="small" offset={[0, 8]} style={BADGE_STYLE}>
        <div
          className="cursor-pointer transition-all duration-300"
          style={AVATAR_WRAPPER_STYLE}
          onMouseEnter={(e) => {
            const avatar = e.currentTarget.querySelector(".ant-avatar") as HTMLElement;
            if (avatar) {
              avatar.style.transform = "scale(1.05)";
              avatar.style.boxShadow = "0 4px 16px rgba(79, 70, 229, 0.3)";
            }
          }}
          onMouseLeave={(e) => {
            const avatar = e.currentTarget.querySelector(".ant-avatar") as HTMLElement;
            if (avatar) {
              avatar.style.transform = "scale(1)";
              avatar.style.boxShadow = "0 2px 8px rgba(79, 70, 229, 0.2)";
            }
          }}
        >
          <Avatar size="large" icon={<BellOutlined />} style={AVATAR_STYLE} />
        </div>
      </Badge>
    </Dropdown>
  );
};

export default NotificationBell;
