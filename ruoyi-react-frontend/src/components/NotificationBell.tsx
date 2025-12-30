import React, { useEffect, useState } from 'react';
import { Badge, Dropdown, List, Empty, Spin, Tag, Avatar, message } from 'antd';
import { BellOutlined, DeleteOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { useNoticeStore } from '@/store/noticeStore';
import { useWebSocket } from '@/hooks/useWebSocket';
import { getNotificationList, markNoticeAsRead, markAllNoticesAsRead, ignoreNotice } from '@/lib/noticeApi';
import type { NoticeListItem } from '@/types/notice';
import { PRIORITY_COLORS, PRIORITY_LABELS, NOTICE_TYPE_COLORS, NOTICE_TYPE_LABELS } from '@/types/notice';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import 'dayjs/locale/zh-cn';

dayjs.extend(relativeTime);
dayjs.locale('zh-cn');

/**
 * 通知铃铛组件
 * 显示未读数量，点击展开通知列表
 */
const NotificationBell: React.FC = () => {
  const navigate = useNavigate();
  const { unreadCount, notifications, loading, markAsRead, markAllAsRead, removeNotification, setNotifications } =
    useNoticeStore();
  const { wsConnected } = useWebSocket();
  const [dropdownOpen, setDropdownOpen] = useState(false);

  // 加载通知列表
  const loadNotifications = async () => {
    try {
      const response = await getNotificationList({ current: 1, pageSize: 10 });
      setNotifications(response.data?.list || []);
    } catch (error) {
      console.error('加载通知列表失败:', error);
    }
  };

  // 下拉框打开时加载通知列表
  useEffect(() => {
    if (dropdownOpen && notifications.length === 0) {
      loadNotifications();
    }
  }, [dropdownOpen, notifications.length]);

  // 点击通知项
  const handleClickNotification = async (notice: NoticeListItem) => {
    // 标记为已读
    if (!notice.isRead) {
      try {
        await markNoticeAsRead(notice.id);
        markAsRead(notice.id);
      } catch (error) {
        console.error('标记已读失败:', error);
      }
    }

    // 跳转到通知详情页面
    setDropdownOpen(false);
    navigate(`/my-notices/${notice.id}`);
  };

  // 全部标记为已读
  const handleMarkAllRead = async () => {
    try {
      await markAllNoticesAsRead();
      markAllAsRead();
      message.success('已全部标记为已读');
    } catch (error) {
      console.error('标记全部已读失败:', error);
      message.error('操作失败，请稍后重试');
    }
  };

  // 删除通知（调用忽略API）
  const handleDelete = async (e: React.MouseEvent, noticeId: string) => {
    e.stopPropagation();
    try {
      await ignoreNotice(noticeId);
      // 从前端列表中移除
      removeNotification(noticeId);
      message.success('已忽略该通知');
    } catch (error) {
      console.error('忽略通知失败:', error);
      message.error('操作失败，请稍后重试');
    }
  };

  // 渲染通知列表
  const notificationContent = (
    <div className="w-96 max-h-96 overflow-y-auto">
      <div className="flex items-center justify-between px-4 py-2 border-b">
        <span className="font-medium">通知</span>
        <div className="flex items-center gap-2">
          {unreadCount > 0 && (
            <span
              className="text-xs text-blue-600 cursor-pointer hover:text-blue-700"
              onClick={handleMarkAllRead}
            >
              全部已读
            </span>
          )}
          <span
            className="text-xs text-gray-500 cursor-pointer hover:text-gray-700"
            onClick={() => {
              setDropdownOpen(false);
              navigate('/my-notices');
            }}
          >
            查看全部
          </span>
        </div>
      </div>

      <Spin spinning={loading}>
        {notifications.length === 0 ? (
          <Empty description="暂无通知" className="py-8" image={Empty.PRESENTED_IMAGE_SIMPLE} />
        ) : (
          <List
            dataSource={notifications}
            renderItem={(notice) => (
              <List.Item
                key={notice.id}
                className={`cursor-pointer hover:bg-gray-50 px-4 py-3 ${!notice.isRead ? 'bg-blue-50' : ''}`}
                onClick={() => handleClickNotification(notice)}
              >
                <div className="flex-1">
                  <div className="flex items-start justify-between gap-2">
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 mb-1">
                        {!notice.isRead && <div className="w-2 h-2 rounded-full bg-blue-500" />}
                        <span className="font-medium text-sm truncate">{notice.noticeTitle}</span>
                      </div>
                      <div className="flex items-center gap-2 text-xs text-gray-500">
                        <Tag color={NOTICE_TYPE_COLORS[notice.noticeType]} bordered={false}>
                          {NOTICE_TYPE_LABELS[notice.noticeType]}
                        </Tag>
                        {notice.priority > 0 && (
                          <Tag color={PRIORITY_COLORS[notice.priority]} bordered={false}>
                            {PRIORITY_LABELS[notice.priority]}
                          </Tag>
                        )}
                        <span>{dayjs(notice.createdAt).fromNow()}</span>
                      </div>
                    </div>
                    <DeleteOutlined
                      className="text-gray-400 hover:text-red-500"
                      onClick={(e) => handleDelete(e, notice.id)}
                    />
                  </div>
                </div>
              </List.Item>
            )}
          />
        )}
      </Spin>
    </div>
  );

  return (
    <Dropdown
      open={dropdownOpen}
      onOpenChange={setDropdownOpen}
      dropdownRender={() => notificationContent}
      placement="bottomRight"
      trigger={['click']}
    >
      <Badge count={unreadCount} size="small" offset={[0, 8]}>
        <Avatar
          size="large"
          icon={<BellOutlined />}
          className="cursor-pointer bg-gradient-to-br from-blue-400 to-blue-500 hover:from-blue-500 hover:to-blue-600 transition-all duration-300 shadow-md"
        />
      </Badge>
    </Dropdown>
  );
};

export default NotificationBell;
